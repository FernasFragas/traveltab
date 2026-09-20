package httpserver

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"weatherservice/internal/planner"
)

func TestGoogleMapsLink_BuildsWalkingDirectionsThroughTheDay(t *testing.T) {
	stops := []planner.Place{
		place("Q1", "A", "", 38.70, -9.20),
		place("Q2", "B", "", 38.71, -9.19),
		place("Q3", "C", "", 38.72, -9.18),
	}

	link := googleMapsWalkingURL(stops)

	parsed, err := url.Parse(link)
	require.NoError(t, err)
	assert.Equal(t, "www.google.com", parsed.Host)
	assert.Equal(t, "/maps/dir/", parsed.Path)

	q := parsed.Query()
	assert.Equal(t, "1", q.Get("api"))
	assert.Equal(t, "walking", q.Get("travelmode"))
	assert.Equal(t, "38.700000,-9.200000", q.Get("origin"))
	assert.Equal(t, "38.720000,-9.180000", q.Get("destination"))
	assert.Equal(t, "38.710000,-9.190000", q.Get("waypoints"))
}

func TestGoogleMapsLink_RespectsTheWaypointLimit(t *testing.T) {
	// MaxStopsPerDay is 4: origin + destination + at most 2 waypoints, which stays within
	// Google's documented mobile limit.
	stops := make([]planner.Place, planner.MaxStopsPerDay)
	for i := range stops {
		stops[i] = place("Q", "Stop", "", 38.7+float64(i)*0.01, -9.2)
	}

	link := googleMapsWalkingURL(stops)

	parsed, err := url.Parse(link)
	require.NoError(t, err)
	waypoints := strings.Split(parsed.Query().Get("waypoints"), "|")
	assert.Len(t, waypoints, planner.MaxStopsPerDay-2, "origin and destination are not waypoints")
}

func TestGoogleMapsLink_TwoStopsHasNoWaypointsParam(t *testing.T) {
	stops := []planner.Place{
		place("Q1", "A", "", 38.70, -9.20),
		place("Q2", "B", "", 38.71, -9.19),
	}

	link := googleMapsWalkingURL(stops)

	parsed, err := url.Parse(link)
	require.NoError(t, err)
	assert.Empty(t, parsed.Query().Get("waypoints"))
}

func TestGoogleMapsLink_OneStopIsASearchLinkNotDirections(t *testing.T) {
	link := googleMapsWalkingURL([]planner.Place{place("Q1", "A", "", 38.70, -9.20)})

	parsed, err := url.Parse(link)
	require.NoError(t, err)
	assert.Equal(t, "/maps/search/", parsed.Path)
	assert.Equal(t, "38.700000,-9.200000", parsed.Query().Get("query"))
}

func TestGoogleMapsLink_NoStopsIsEmpty(t *testing.T) {
	assert.Empty(t, googleMapsWalkingURL(nil))
}
