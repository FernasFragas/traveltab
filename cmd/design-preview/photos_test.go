package main

import (
	"bytes"
	"context"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	app "weatherservice/internal/application"
)

const testOrigin = "http://127.0.0.1:8088/"

func TestFixturePhotoOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   app.DestinationIdentity
		file string // "" when no photo
		err  bool
	}{
		{"Paris, France", app.DestinationIdentity{City: "Paris", Country: "FR"}, "paris", false},
		{"Paris, Texas", app.DestinationIdentity{City: "Paris", Country: "US"}, "paris-texas", false},
		{"Porto", app.DestinationIdentity{City: "Porto", Country: "PT"}, "porto", false},
		{"no photo", app.DestinationIdentity{City: "Nophoto", Country: "PT"}, "", false},
		{"unknown destination", app.DestinationIdentity{City: "Coimbra", Country: "PT"}, "", false},
		{"provider failure", app.DestinationIdentity{City: "Photoerror", Country: "PT"}, "", true},
		{"broken image", app.DestinationIdentity{City: "Brokenimage", Country: "PT"}, "missing", false},
		{"Paris in another country", app.DestinationIdentity{City: "Paris", Country: "PT"}, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			photos := &fixturePhotos{origin: testOrigin}
			photo, err := photos.Photo(context.Background(), tc.id)
			if (err != nil) != tc.err {
				t.Fatalf("err=%v, want error=%t", err, tc.err)
			}
			if photos.calls.Load() != 1 {
				t.Fatalf("calls=%d, want 1", photos.calls.Load())
			}
			if tc.file == "" {
				if photo != nil {
					t.Fatalf("unexpected photo %+v", photo)
				}
				return
			}
			if photo == nil || !photo.Complete() {
				t.Fatalf("incomplete photo %+v", photo)
			}
			if photo.URL != testOrigin+"fixture-photos/"+tc.file+".png" {
				t.Fatalf("url=%s", photo.URL)
			}
			// Fixtures must never point a browser, or a test, at the real provider.
			if strings.Contains(photo.URL, "wikimedia") || strings.Contains(photo.CreditURL, "wikimedia") {
				t.Fatalf("fixture photo references Wikimedia: %+v", photo)
			}
		})
	}
}

func TestFixturePhotoMissingScenariosHaveNoPhotos(t *testing.T) {
	for _, scenario := range []string{"photos-missing", "fallback"} {
		photos := &fixturePhotos{origin: testOrigin, scenario: scenario}
		for _, city := range []string{"Paris", "Porto"} {
			id := app.DestinationIdentity{City: city, Country: map[string]string{"Paris": "FR", "Porto": "PT"}[city]}
			if photo, err := photos.Photo(context.Background(), id); photo != nil || err != nil {
				t.Fatalf("%s/%s: photo=%v err=%v", scenario, city, photo, err)
			}
		}
		if photos.calls.Load() != 2 {
			t.Fatalf("%s: calls=%d", scenario, photos.calls.Load())
		}
	}
}

func TestFixtureWeatherIdentities(t *testing.T) {
	for _, tc := range []struct {
		query, city, country string
		lat                  float64
	}{
		{"Lisbon, Portugal", "Lisbon", "pt", 38.7223},
		{"Porto, Portugal", "Porto", "pt", 41.1494},
		{"tokyo", "Tokyo", "jp", 35.6895},
		{"Tavira, Portugal", "Tavira", "pt", 37.1264},
		{"Coimbra, Portugal", "Coimbra", "pt", 38.7223},
		{"paris", "Paris", "fr", 48.8589},
		{"Paris, France", "Paris", "fr", 48.8589},
		{"paris, fr", "Paris", "fr", 48.8589},
		{"Paris, Texas", "Paris", "us", 33.6609},
		{"paris, us", "Paris", "us", 33.6609},
		{"nophoto", "Nophoto", "pt", 38.7223},
		{"PHOTOERROR", "Photoerror", "pt", 38.7223},
		{"brokenimage, pt", "Brokenimage", "pt", 38.7223},
	} {
		info, err := (fixtureWeather{}).GenerateReport(context.Background(), tc.query)
		if err != nil {
			t.Fatalf("%q: %v", tc.query, err)
		}
		if info.City != tc.city || info.Country != tc.country || info.Lat != tc.lat {
			t.Fatalf("%q: got %s/%s/%v", tc.query, info.City, info.Country, info.Lat)
		}
		// Every resolved destination is a usable photo identity.
		if _, ok := app.NewDestinationIdentity(info.City, info.Country, info.Lat, info.Lon); !ok {
			t.Fatalf("%q: not a usable identity: %+v", tc.query, info)
		}
	}
}

func TestFixturePhotoHandler(t *testing.T) {
	handler := fixtureOriginHandler()
	fetch := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec
	}

	images := map[string][]byte{}
	for _, name := range []string{"paris", "paris-texas", "porto"} {
		rec := fetch("/fixture-photos/" + name + ".png")
		if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" {
			t.Fatalf("%s: status=%d type=%s", name, rec.Code, rec.Header().Get("Content-Type"))
		}
		img, err := png.Decode(bytes.NewReader(rec.Body.Bytes()))
		if err != nil || img.Bounds().Dx() != fixturePhotoWidth || img.Bounds().Dy() != fixturePhotoHeight {
			t.Fatalf("%s: not a %dx%d PNG: %v", name, fixturePhotoWidth, fixturePhotoHeight, err)
		}
		images[name] = rec.Body.Bytes()
		if again := fetch("/fixture-photos/" + name + ".png"); !bytes.Equal(again.Body.Bytes(), images[name]) {
			t.Fatalf("%s: not deterministic", name)
		}
	}
	if bytes.Equal(images["paris"], images["paris-texas"]) || bytes.Equal(images["paris"], images["porto"]) {
		t.Fatal("cities must be visibly different")
	}
	for _, path := range []string{"/fixture-photos/missing.png", "/fixture-photos/"} {
		if rec := fetch(path); rec.Code != http.StatusNotFound {
			t.Fatalf("%s: status=%d, want 404", path, rec.Code)
		}
	}
	// Only known names are served, whatever the path tries.
	if rec := fetch("/fixture-photos/../secret.png"); rec.Code == http.StatusOK {
		t.Fatal("path traversal served a file")
	}
	// The same origin still serves the map document.
	if rec := fetch("/"); !strings.Contains(rec.Body.String(), "FIXTURE MAP") {
		t.Fatal("map fixture lost")
	}
}

func TestPreviewServerWiresThePhotoFixture(t *testing.T) {
	server, photos, cleanup := newPreviewServerWithPhotos(testOrigin, false, "default")
	defer cleanup()
	if server == nil || photos == nil || photos.origin != testOrigin {
		t.Fatal("photo fixture not wired")
	}
}

// The Fiber server exposes no test transport, so the photo states are checked over HTTP against
// a child process, like the other route tests.
func TestPreviewPhotoRoutes(t *testing.T) {
	base, client := startPreviewProcess(t, "default")
	get := func(path string) string {
		t.Helper()
		res, err := client.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = res.Body.Close() }()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusOK {
			t.Fatalf("%s: status=%d", path, res.StatusCode)
		}
		return string(body)
	}
	for _, tc := range []struct {
		name, path string
		present    []string
		absent     []string
	}{
		{"Lisbon keeps its curated photo", "/process-form/?city_name=Lisbon", []string{`src="/images/destinations/lisbon-alfama.jpg"`, `data-photo-state="ready"`}, []string{"fixture-photos"}},
		{"Paris, France", "/process-form/?city_name=Paris", []string{`src="` + testOrigin + `fixture-photos/paris.png"`, "Fixture Photographer (Paris)", `alt="View of Paris"`, `data-photo-state="ready"`, `name="country" value="fr"`}, []string{"paris-texas"}},
		{"Paris, Texas", "/process-form/?city_name=Paris%2C%20Texas", []string{`fixture-photos/paris-texas.png"`, `name="country" value="us"`}, []string{"fixture-photos/paris.png"}},
		{"shared Paris, France", "/trip/paris-fr", []string{`fixture-photos/paris.png"`}, []string{"paris-texas"}},
		{"shared Paris, Texas", "/trip/paris-us", []string{`fixture-photos/paris-texas.png"`}, []string{"fixture-photos/paris.png"}},
		{"another city", "/process-form/?city_name=Porto%2C%20Portugal", []string{`fixture-photos/porto.png"`, `data-photo-state="ready"`}, []string{"paris"}},
		{"no photograph", "/process-form/?city_name=Nophoto", []string{`data-photo-state="unavailable"`, "Photo unavailable", `<h1 class="destination-title">Nophoto</h1>`, "weather-metrics"}, []string{`class="destination-image"`, "destination-photo-credit"}},
		{"unknown city has none", "/process-form/?city_name=Coimbra%2C%20Portugal", []string{`data-photo-state="unavailable"`}, []string{`class="destination-image"`}},
		{"provider failure", "/process-form/?city_name=Photoerror", []string{`data-photo-state="unavailable"`, "Photo unavailable", "weather-metrics", `id="trip-card"`}, []string{`class="destination-image"`}},
		{"broken image URL", "/process-form/?city_name=Brokenimage", []string{`fixture-photos/missing.png"`, `data-photo-state="ready"`, "destination-photo-fallback", "onerror="}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := get(tc.path)
			for _, text := range tc.present {
				if !strings.Contains(body, text) {
					t.Errorf("missing %q", text)
				}
			}
			for _, text := range tc.absent {
				if strings.Contains(body, text) {
					t.Errorf("unexpected %q", text)
				}
			}
		})
	}
}

func TestPreviewPhotosMissingScenarioRoute(t *testing.T) {
	base, client := startPreviewProcess(t, "photos-missing")
	res, err := client.Get(base + "/process-form/?city_name=Porto%2C%20Portugal")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), `data-photo-state="unavailable"`) || strings.Contains(string(body), "fixture-photos") {
		t.Fatalf("photos-missing must leave Porto photo-free: %s", body[:min(len(body), 200)])
	}
}
