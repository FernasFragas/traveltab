package main

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"weatherservice/internal/planner"
)

func TestFixtureCacheMiss(t *testing.T) {
	_, err := (silentAnalytics{}).GetCityData("Lisbon")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cache error = %v", err)
	}
}

func TestFixtureVideoQuery(t *testing.T) {
	for _, tc := range []struct {
		query string
		count int
	}{
		{"Turistic places in Porto, pt", 0}, {"Porto, Portugal", 0},
		{"Turistic places in Lisbon, pt", 6}, {"Turistic places in Coimbra, pt", 6},
	} {
		videos, err := (fixtureVideos{}).GenerateReport(context.Background(), tc.query)
		if err != nil || len(*videos) != tc.count {
			t.Fatalf("%q: videos=%v, err=%v", tc.query, videos, err)
		}
	}
}

func TestFixtureScenarios(t *testing.T) {
	req := planner.Request{City: "Lisbon", Country: "pt", Days: 5, Start: time.Now().UTC().AddDate(0, 0, 1)}
	for _, scenario := range []string{"default", "photos-missing", "forecast-absent", "forecast-uncertain", "stays-0", "stays-1", "stays-6", "stays-unavailable", "videos-0", "videos-1", "videos-10", "fallback"} {
		t.Run(scenario, func(t *testing.T) {
			if !validScenario(scenario) {
				t.Fatal("scenario rejected")
			}
			plan, err := (fixtureTrips{scenario: scenario}).Plan(context.Background(), req)
			if err != nil || len(plan.Days) != 5 {
				t.Fatalf("plan=%v, err=%v", plan, err)
			}
			wantStays := 4
			switch scenario {
			case "stays-0", "stays-unavailable", "fallback":
				wantStays = 0
			case "stays-1":
				wantStays = 1
			case "stays-6":
				wantStays = 6
			}
			if len(plan.Stays) != wantStays {
				t.Fatalf("stays=%d, want %d", len(plan.Stays), wantStays)
			}
			if plan.BookingURL == "" {
				t.Fatal("missing booking fallback")
			}
			if (scenario == "stays-unavailable" || scenario == "fallback") && plan.StaysNote == "" {
				t.Fatal("missing stays failure explanation")
			}
			if scenario == "photos-missing" || scenario == "fallback" {
				for _, day := range plan.Days {
					for _, stop := range day.Stops {
						if stop.Image != "" {
							t.Fatal("photo in photo-free fixture")
						}
					}
				}
			}
			for _, day := range plan.Days {
				if (scenario == "forecast-absent" || scenario == "fallback") && day.Forecast != nil {
					t.Fatal("unexpected forecast")
				}
				if scenario == "forecast-uncertain" && (day.Forecast == nil || day.Certain) {
					t.Fatal("uncertain forecast must remain nonnil")
				}
			}
			videos, err := (fixtureVideos{scenario: scenario}).GenerateReport(context.Background(), "Turistic places in Lisbon, pt")
			wantVideos := 6
			switch scenario {
			case "videos-0", "fallback":
				wantVideos = 0
			case "videos-1":
				wantVideos = 1
			case "videos-10":
				wantVideos = 10
			}
			if err != nil || len(*videos) != wantVideos {
				t.Fatalf("videos=%v, err=%v", videos, err)
			}
		})
	}
	if validScenario("typo") {
		t.Fatal("unknown scenario accepted")
	}
	req.Days = 4
	if _, err := (fixtureTrips{}).Plan(context.Background(), req); err == nil {
		t.Fatal("four-day failure lost")
	}
	req.Days = 1
	if p, err := (fixtureTrips{}).Plan(context.Background(), req); err != nil || len(p.Days) != 1 {
		t.Fatal("one-day fixture failed")
	}
}

func TestSlowProvidersHonorCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (fixtureWeather{delay: time.Hour}).GenerateReport(ctx, "Lisbon, Portugal"); !errors.Is(err, context.Canceled) {
		t.Fatalf("search cancellation=%v", err)
	}
	if _, err := (fixtureTrips{delay: time.Hour}).Plan(ctx, planner.Request{Days: 3}); !errors.Is(err, context.Canceled) {
		t.Fatalf("plan cancellation=%v", err)
	}
}

func TestMapFixture(t *testing.T) {
	rec := httptest.NewRecorder()
	fixtureMapHandler(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "style-src 'unsafe-inline'") {
		t.Fatal("map styles blocked by CSP")
	}
	if !strings.Contains(rec.Body.String(), "FIXTURE MAP") {
		t.Fatal("missing fixture label")
	}
}

// Run the actual production server in a child process: Server intentionally exposes no test
// transport. Killing only this child avoids leaking listeners or changing production APIs.
func TestPreviewProcess(t *testing.T) {
	addr := os.Getenv("TRAVELTAB_PREVIEW_TEST_ADDR")
	if addr == "" {
		return
	}
	scenario := os.Getenv("TRAVELTAB_PREVIEW_TEST_SCENARIO")
	if scenario == "" {
		scenario = "default"
	}
	server, cleanup := newPreviewServer("http://127.0.0.1:8088/", false, scenario)
	defer cleanup()
	if err := server.Listen(addr); err != nil {
		t.Fatal(err)
	}
}

func startPreviewProcess(t *testing.T, scenario string) (string, *http.Client) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	if err = listener.Close(); err != nil {
		t.Fatal(err)
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binary, "-test.run=^TestPreviewProcess$")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "TRAVELTAB_PREVIEW_TEST_ADDR="+addr, "TRAVELTAB_PREVIEW_TEST_SCENARIO="+scenario)
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	client := &http.Client{Timeout: 2 * time.Second}
	base := "http://" + addr
	deadline := time.Now().Add(10 * time.Second)
	for {
		res, e := client.Get(base + "/redesign/base.css")
		if e == nil {
			_ = res.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("preview did not start: %v", e)
		}
		time.Sleep(20 * time.Millisecond)
	}
	return base, client
}

func TestPreviewRoutes(t *testing.T) {
	base, client := startPreviewProcess(t, "default")
	start := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	for _, tc := range []struct {
		path, contentType string
		texts             []string
	}{
		{"/", "text/html", []string{"Lisbon", `name="country" value="pt"`, "dQw4w9WgXcQ", "/images/destinations/lisbon-alfama.jpg"}},
		{"/process-form/?city_name=Porto%2C%20Portugal", "text/html", []string{"Porto", "No travel videos are available for this destination right now."}},
		{"/trip/lisbon-pt?days=1&from=" + start, "text/html", []string{"Lisbon", "Fixture Boutique Hotel", "Belém Tower"}},
		{"/trip/lisbon-pt.ics?days=1&from=" + start, "text/calendar", []string{"BEGIN:VCALENDAR", "BEGIN:VEVENT", "DTSTART:" + strings.ReplaceAll(start, "-", "")}},
		{"/trip/lisbon-pt.kml?days=1&from=" + start, "application/vnd.google-earth.kml+xml", []string{"<kml", "<Placemark>", "-9.216,38.6916"}},
	} {
		t.Run(tc.path, func(t *testing.T) {
			res, e := client.Get(base + tc.path)
			if e != nil {
				t.Fatal(e)
			}
			defer func() { _ = res.Body.Close() }()
			body, e := io.ReadAll(res.Body)
			if e != nil {
				t.Fatal(e)
			}
			if res.StatusCode != 200 {
				t.Fatalf("status=%d: %s", res.StatusCode, body)
			}
			if !strings.Contains(res.Header.Get("Content-Type"), tc.contentType) {
				t.Fatalf("content type=%s", res.Header.Get("Content-Type"))
			}
			for _, text := range tc.texts {
				if !strings.Contains(string(body), text) {
					t.Errorf("missing %q", text)
				}
			}
		})
	}
}

func TestScenarioRoutes(t *testing.T) {
	start := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	for _, tc := range []struct {
		name, pageText, planText, absentText string
	}{
		{"photos-missing", "Lisbon", "Belém Tower", "itinerary-stop-photo"},
		{"forecast-absent", "Lisbon", "No forecast data for this day", "Forecast less certain"},
		{"forecast-uncertain", "Lisbon", "Forecast less certain", "No forecast data for this day"},
		{"stays-0", "Lisbon", "No matching places to stay were returned", "Fixture Boutique Hotel"},
		{"stays-1", "Lisbon", "Fixture Boutique Hotel", "Fixture Hostel"},
		{"stays-6", "Lisbon", "Fixture Riverside Hotel", ""},
		{"stays-unavailable", "Lisbon", "Nearby stays are temporarily unavailable", "Fixture Boutique Hotel"},
		{"videos-0", "No travel videos are available for this destination right now.", "Belém Tower", ""},
		{"videos-1", "Fixture travel film", "Belém Tower", ""},
		{"videos-10", "Fixture extra travel film", "Belém Tower", ""},
		{"fallback", "No travel videos are available for this destination right now.", "Nearby stays are temporarily unavailable", "itinerary-stop-photo"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base, client := startPreviewProcess(t, tc.name)
			for _, check := range []struct{ path, present, absent string }{
				{"/", tc.pageText, ""},
				{"/plan?city=Lisbon&country=pt&lat=38.7223&lon=-9.1393&start=" + start + "&days=3", tc.planText, tc.absentText},
			} {
				res, err := client.Get(base + check.path)
				if err != nil {
					t.Fatal(err)
				}
				body, err := io.ReadAll(res.Body)
				_ = res.Body.Close()
				if err != nil {
					t.Fatal(err)
				}
				if res.StatusCode != http.StatusOK || !strings.Contains(string(body), check.present) || check.absent != "" && strings.Contains(string(body), check.absent) {
					t.Fatalf("%s: status=%d, present %q=%t, absent %q=%t", check.path, res.StatusCode, check.present, strings.Contains(string(body), check.present), check.absent, strings.Contains(string(body), check.absent))
				}
			}
		})
	}
}
