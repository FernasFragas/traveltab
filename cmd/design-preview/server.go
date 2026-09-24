package main

import (
	"time"
	"weatherservice/internal/adapters/httpserver"
)

// newPreviewServer wires the real Fiber handlers/templates to fixture reporters. It uses no
// database and disables city caching so repeated checks see deterministic state. cleanup is
// retained for parity with a temporary-storage server, but is currently a no-op.
func newPreviewServer(mapOrigin string, slow bool, scenario string) (*httpserver.Server, func()) {
	delay := time.Duration(0)
	if slow {
		delay = 2 * time.Second
	}
	server := httpserver.NewAppServer(fixtureWeather{mapURL: mapOrigin, delay: delay}, fixtureVideos{scenario: scenario}, silentAnalytics{})
	server.SetTripPlanner(fixtureTrips{delay: delay, scenario: scenario})
	return server, func() {}
}
