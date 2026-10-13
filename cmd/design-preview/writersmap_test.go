package main

import (
	"context"
	"encoding/json"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"weatherservice/internal/application"
	"weatherservice/internal/writermap"
)

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
