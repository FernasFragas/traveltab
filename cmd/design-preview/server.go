package main

import (
	"time"
	"weatherservice/internal/adapters/httpserver"
)

// newPreviewServer wires the real Fiber handlers/templates to fixture reporters. It uses no
// database and disables city caching so repeated checks see deterministic state. Destination
// photos come from fixturePhotos, served by the local fixture origin, and city guides from
// fixtureGuides (the reviewed guides embedded in the binary). cleanup is retained for
// parity with a temporary-storage server, but is currently a no-op.
func newPreviewServer(mapOrigin string, slow bool, scenario string) (*httpserver.Server, func()) {
	server, _, cleanup := newPreviewServerWithPhotos(mapOrigin, slow, scenario)
	return server, cleanup
}

// newPreviewServerWithPhotos also returns the photo source, so tests can count its calls.
func newPreviewServerWithPhotos(mapOrigin string, slow bool, scenario string) (*httpserver.Server, *fixturePhotos, func()) {
	delay := time.Duration(0)
	if slow {
		delay = 2 * time.Second
	}
	server := httpserver.NewAppServer(fixtureWeather{mapURL: mapOrigin, delay: delay}, fixtureVideos{scenario: scenario}, silentAnalytics{})
	server.SetTripPlanner(fixtureTrips{delay: delay, scenario: scenario})
	photos := &fixturePhotos{origin: mapOrigin, scenario: scenario}
	server.SetDestinationPhotoSource(photos)
	server.SetCityGuideSource(newFixtureGuides(scenario))
	return server, photos, func() {}
}
