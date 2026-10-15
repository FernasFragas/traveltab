package main

import (
	"context"
	"encoding/json"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"weatherservice/internal/application"
	"weatherservice/internal/planner"
	"weatherservice/internal/writermap"
)

// As a traveler, I see Lisbon photos through the real map and panel routes using local images.
func TestWriterMapPreview_LisbonPhotosReachBrowser(t *testing.T) {
	base, client := startPreviewProcess(t, "writers-map")
	for _, path := range []string{
		"/map/lisbon-pt.geojson?lat=38.7223&lon=-9.1393",
		"/map/lisbon-pt/Q215003?lat=38.7223&lon=-9.1393",
	} {
		res, err := client.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if err != nil || res.StatusCode != http.StatusOK {
			t.Fatalf("%s status=%d err=%v", path, res.StatusCode, err)
		}
		photos := regexp.MustCompile(`/fixture-photos/lisbon-[a-z]+\.png`).FindAllString(string(body), -1)
		if len(photos) == 0 {
			t.Fatalf("%s has no local Lisbon photo: %s", path, body)
		}
		if strings.Contains(path, ".geojson") {
			if !strings.Contains(string(body), `"photo_url":"/fixture-photos/`) {
				t.Fatalf("%s has no local photo_url property", path)
			}
		} else if !strings.Contains(string(body), `src="/fixture-photos/`) {
			t.Fatalf("%s has no local image src", path)
		}
		if strings.Contains(string(body), "upload.wikimedia.org/fixture-photos/") {
			t.Fatalf("%s leaked a remote fixture marker", path)
		}
		for _, photo := range photos {
			res, err := client.Get(base + photo)
			if err != nil {
				t.Fatal(err)
			}
			_, decodeErr := png.Decode(res.Body)
			_ = res.Body.Close()
			if res.StatusCode != http.StatusOK || decodeErr != nil {
				t.Fatalf("%s status=%d PNG=%v", photo, res.StatusCode, decodeErr)
			}
		}
	}
}

// As a traveler, I see varied Lisbon pins and complete place details without remote photos.
func TestWriterMapPreview_RichLisbon(t *testing.T) {
	result, err := newWriterMapPreviewSource().Destination(context.Background(), writermap.Area{Name: "Lisbon"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Places) != 12 {
		t.Fatalf("Lisbon places = %d, want 12", len(result.Places))
	}
	kinds := map[string]bool{}
	foundBelem := false
	for _, place := range result.Places {
		kinds[place.Kind] = true
		if place.Area == "" || place.Description == "" || !strings.HasPrefix(place.PhotoURL, "https://upload.wikimedia.org/fixture-photos/") {
			t.Errorf("incomplete local place details: %+v", place)
		}
		photo := httptest.NewRecorder()
		fixturePhotoHandler(photo, httptest.NewRequest(http.MethodGet, strings.TrimPrefix(place.PhotoURL, "https://upload.wikimedia.org"), nil))
		if photo.Code != http.StatusOK {
			t.Errorf("%s photo status = %d", place.Names[0], photo.Code)
		}
		if place.QID == "Q215003" {
			foundBelem = true
			if place.OpeningHours != "Tu-Su 10:00-18:30" || place.WikipediaURL != "https://en.wikipedia.org/wiki/Bel%C3%A9m_Tower" {
				t.Errorf("Belém facts = %+v", place)
			}
			if len(result.Mentions[place.QID]) != 2 || result.WriterCounts[place.QID] != 2 {
				t.Errorf("Belém mentions = %+v", result.Mentions[place.QID])
			}
		}
	}
	if !foundBelem {
		t.Fatal("missing Belém Tower")
	}
	for _, kind := range []string{"museum", "cafe", "viewpoint", "park"} {
		if !kinds[kind] {
			t.Errorf("missing category %s", kind)
		}
	}
}

// As a traveler, I preview a Lisbon day from Belém in the morning to Alfama in the evening.
func TestWriterMapPreview_LisbonDayPlan(t *testing.T) {
	plan, err := (fixtureTrips{scenario: "writers-map"}).Plan(context.Background(), planner.Request{City: "Lisbon", Days: 1})
	if err != nil {
		t.Fatal(err)
	}
	stops := plan.Days[0].Stops
	if len(stops) != 3 {
		t.Fatalf("Lisbon day stops = %d, want 3", len(stops))
	}
	for i, name := range []string{"Belém Tower", "Jerónimos Monastery", "Alfama"} {
		if stops[i].Name != name || stops[i].Image == "" {
			t.Errorf("day slot %d = %+v, want %s with photo", i, stops[i], name)
		}
	}
}

// As a traveler, I can switch between writer, base-only, and unavailable map destinations.
func TestWriterMapPreview_Scenarios(t *testing.T) {
	if !validScenario("writers-map") {
		t.Fatal("writers-map scenario rejected")
	}
	source := newWriterMapPreviewSource()
	for _, tc := range []struct {
		city       string
		wantPlace  string
		wantWriter bool
		wantError  bool
	}{
		{"Madeira", "Pico Ruivo", true, false},
		{"Porto", "Clérigos Tower", false, false},
		{"Coimbra", "", false, false},
	} {
		t.Run(tc.city, func(t *testing.T) {
			result, err := source.Destination(context.Background(), writermap.Area{Name: tc.city})
			if (err != nil) != tc.wantError {
				t.Fatalf("Destination error = %v", err)
			}
			if tc.wantError {
				return
			}
			if tc.city == "Coimbra" {
				if result.Complete || len(result.Places) != 0 || len(result.SourceStatuses) == 0 {
					t.Fatalf("unavailable result = %+v", result)
				}
				return
			}
			found := false
			for _, place := range result.Places {
				if len(place.Names) > 0 && place.Names[0] == tc.wantPlace {
					found = true
					if (result.WriterCounts[place.QID] > 0) != tc.wantWriter {
						t.Fatalf("writer count = %d", result.WriterCounts[place.QID])
					}
				}
			}
			if !found {
				t.Fatalf("missing %q in %+v", tc.wantPlace, result.Places)
			}
		})
	}
}

// As a traveler, I can complete the writer map journey in two distinct cities.
func TestWriterMapPreview_TwoWriterCities(t *testing.T) {
	source := newWriterMapPreviewSource()
	for _, tc := range []struct {
		city, place string
	}{
		{"Funchal", "Pico Ruivo"},
		{"Lisbon", "Belém Tower"},
	} {
		t.Run(tc.city, func(t *testing.T) {
			result, err := source.Destination(context.Background(), writermap.Area{Name: tc.city})
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Itineraries) == 0 {
				t.Fatal("no writer itinerary")
			}
			for _, place := range result.Places {
				if len(place.Names) > 0 && place.Names[0] == tc.place {
					if result.WriterCounts[place.QID] == 0 || len(result.Mentions[place.QID]) == 0 {
						t.Fatalf("%s has no writer posts: %+v", tc.place, result)
					}
					if _, _, err := source.Place(context.Background(), writermap.Area{Name: tc.city}, place.QID); err != nil {
						t.Fatalf("%s panel: %v", tc.place, err)
					}
					return
				}
			}
			t.Fatalf("missing %s", tc.place)
		})
	}
}

func TestWriterMapPreview_FunchalWeatherUsesCityCoordinates(t *testing.T) {
	info, err := (fixtureWeather{}).GenerateReport(context.Background(), "Funchal, PT")
	if err != nil || info.City != "Funchal" || info.Country != "pt" || info.Lat != 32.65 || info.Lon != -16.91 {
		t.Fatalf("Funchal weather identity = %+v, err=%v", info, err)
	}
}

func TestWriterMapPreview_RoutesAndLocalTiles(t *testing.T) {
	base, client := startPreviewProcess(t, "writers-map")
	for _, tc := range []struct {
		path, want string
		status     int
	}{
		{"/process-form/?city_name=Madeira", `data-map-destination="madeira-pt"`, 200},
		{"/process-form/?city_name=Porto", `data-map-destination="porto-pt"`, 200},
		{"/process-form/?city_name=Coimbra", `data-map-destination="coimbra-pt"`, 200},
		{"/map/madeira-pt.geojson?lat=32.65&lon=-16.9", "Pico Ruivo", 200},
		{"/map/porto-pt.geojson?lat=41.1494&lon=-8.6108", "Clérigos Tower", 200},
		{"/map/coimbra-pt.geojson?lat=38.7223&lon=-9.1393", `"complete":false`, 200},
	} {
		res, err := client.Get(base + tc.path)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()
		if res.StatusCode != tc.status || !strings.Contains(string(body), tc.want) {
			t.Errorf("%s: status=%d body=%s", tc.path, res.StatusCode, body)
		}
		if strings.HasPrefix(tc.path, "/process-form/") && !strings.Contains(string(body), "data-map-tile-url=") {
			t.Errorf("%s: missing local tile URL", tc.path)
		}
	}

	res := httptest.NewRecorder()
	fixtureTileHandler(res, httptest.NewRequest(http.MethodGet, "/fixture-tiles/12/2048/1363.png", nil))
	if res.Code != 200 || res.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("tile status=%d content type=%q", res.Code, res.Header().Get("Content-Type"))
	}
	if _, err := png.Decode(strings.NewReader(res.Body.String())); err != nil {
		t.Fatalf("tile is not PNG: %v", err)
	}
	res = httptest.NewRecorder()
	fixtureTileHandler(res, httptest.NewRequest(http.MethodGet, "/fixture-tiles/bad/0/0.png", nil))
	if res.Code != 404 {
		t.Fatalf("bad tile status=%d", res.Code)
	}
}

func TestWriterMapPreview_LiveTilesOptIn(t *testing.T) {
	base, client := startPreviewProcess(t, "writers-map", true)
	res, err := client.Get(base + "/trip/madeira-pt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), "data-writers-map") {
		t.Fatalf("live tile page: status=%d body=%s", res.StatusCode, body)
	}
	if strings.Contains(string(body), "data-map-tile-url=") || strings.Contains(string(body), "data-map-attribution=") {
		t.Fatalf("live tile page still overrides the production basemap: %s", body)
	}
	suggestions, err := (fixtureSuggestions{}).Suggest(context.Background(), "lis")
	if err != nil || len(suggestions) != 1 || suggestions[0].Name != "Lisbon" {
		t.Fatalf("live tile preview suggestions = %+v, err = %v", suggestions, err)
	}
}

func TestWriterMapPreview_BaseOnlyJSON(t *testing.T) {
	source := newWriterMapPreviewSource()
	result, err := source.Destination(context.Background(), writermap.Area{Name: "Porto"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "mentions") || !result.Complete {
		t.Fatalf("base-only result = %s", data)
	}
}

func TestWriterMapPreview_PanelsAndSelectedDestination(t *testing.T) {
	source := newWriterMapPreviewSource()
	for _, tc := range []struct {
		city, qid, name string
		wantSummary     bool
	}{
		{"Madeira", "Q473169", "Pico Ruivo", true},
		{"Porto", "Q208954", "Clérigos Tower", false},
	} {
		place, summary, err := source.Place(context.Background(), writermap.Area{Name: tc.city}, tc.qid)
		if err != nil || place == nil || place.Names[0] != tc.name || (summary != nil) != tc.wantSummary {
			t.Fatalf("%s panel: place=%+v summary=%+v err=%v", tc.city, place, summary, err)
		}
	}
	if _, _, err := source.Place(context.Background(), writermap.Area{Name: "Coimbra"}, "Q473169"); err == nil {
		t.Fatal("unavailable source returned a panel")
	}
	info, err := (fixtureWeather{}).GenerateReportForPlace(context.Background(), application.PlaceSelection{
		Name: "Madeira", CountryCode: "PT", Lat: 32.65, Lon: -16.9,
	})
	if err != nil || info.City != "Madeira" || info.Lat != 32.65 || info.Lon != -16.9 {
		t.Fatalf("selected destination = %+v, err=%v", info, err)
	}
}
