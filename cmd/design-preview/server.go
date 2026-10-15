package main

import (
	"context"
	"strings"
	"time"
	"weatherservice/internal/adapters/httpserver"
	"weatherservice/internal/application"
)

// newPreviewServer wires the real Fiber handlers/templates to fixture reporters. It uses no
// database and disables city caching so repeated checks see deterministic state. Destination
// photos come from fixturePhotos, served by the local fixture origin, and city guides from
// fixtureGuides (the reviewed guides embedded in the binary). cleanup is retained for
// parity with a temporary-storage server, but is currently a no-op.
func newPreviewServer(mapOrigin string, slow bool, scenario string, liveTiles bool) (*previewServer, func()) {
	server, _, cleanup := newPreviewServerWithPhotos(mapOrigin, slow, scenario)
	if liveTiles && scenario == "writers-map" {
		server.SetWriterMapTiles("", "")
		server.SetWriterMapBasemap("", "", "")
	}
	return &previewServer{Server: server, localPlacePhotos: scenario == "writers-map" || scenario == "writers-map-leaflet"}, cleanup
}

// newPreviewServerWithPhotos also returns the photo source, so tests can count its calls.
func newPreviewServerWithPhotos(mapOrigin string, slow bool, scenario string) (*httpserver.Server, *fixturePhotos, func()) {
	delay := time.Duration(0)
	if slow {
		delay = 2 * time.Second
	}
	server := httpserver.NewAppServer(fixtureWeather{mapURL: mapOrigin, delay: delay}, fixtureVideos{scenario: scenario}, silentAnalytics{})
	server.SetPlaceSuggester(fixtureSuggestions{})
	server.SetForecastSource(fixtureForecast{})
	server.SetTripPlanner(fixtureTrips{delay: delay, scenario: scenario})
	photos := &fixturePhotos{origin: mapOrigin, scenario: scenario}
	server.SetDestinationPhotoSource(photos)
	server.SetCityGuideSource(newFixtureGuides(scenario))
	if scenario == "writers-map" || scenario == "writers-map-leaflet" {
		server.SetWriterMapSource(newWriterMapPreviewSource(), true)
		server.SetWriterMapTiles(mapOrigin+"fixture-tiles/{z}/{x}/{y}.png", "Local fixture tiles")
		engine := ""
		if scenario == "writers-map-leaflet" {
			engine = "leaflet" // forces the Leaflet fallback so checks can run on both engines
		}
		server.SetWriterMapBasemap(mapOrigin+"fixture-map/style.json", mapOrigin+"fixture-map/terrain/{z}/{x}/{y}.png", engine)
	}
	return server, photos, func() {}
}

// fixtureSuggestions keeps preview autocomplete checks independent of the live geocoder.
type fixtureSuggestions struct{}

func (fixtureSuggestions) Suggest(_ context.Context, query string) ([]application.PlaceSuggestion, error) {
	switch strings.ToLower(query) {
	case "lis", "lisb", "lisbo", "lisbon":
		return []application.PlaceSuggestion{{Name: "Lisbon", Region: "Lisbon", Country: "Portugal", CountryCode: "PT", Lat: 38.7223, Lon: -9.1393}}, nil
	case "por", "port", "porto":
		return []application.PlaceSuggestion{{Name: "Porto", Country: "Portugal", CountryCode: "PT", Lat: 41.1494, Lon: -8.6108}}, nil
	case "par", "pari", "paris":
		return []application.PlaceSuggestion{
			{Name: "Paris", Region: "Île-de-France", Country: "France", CountryCode: "FR", Lat: 48.85341, Lon: 2.3488},
			{Name: "Paris", Region: "Texas", Country: "United States", CountryCode: "US", Lat: 33.66094, Lon: -95.55551},
			{Name: "Paris", Region: "Tennessee", Country: "United States", CountryCode: "US", Lat: 36.302, Lon: -88.32671},
		}, nil
	}
	return nil, nil
}
