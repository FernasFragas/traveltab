package httpserver

import (
	"fmt"
	"net/url"
	"strings"

	"weatherservice/internal/planner"
)

// googleMapsWalkingURL builds a walking-directions link through stops, needing no API key.
// The first stop is the origin, the last the destination, and everything between becomes a
// waypoint - MaxStopsPerDay (4) keeps this at 2 waypoints, well inside Google's mobile limit.
// A single stop gets a plain search link instead, since there is no route to walk.
func googleMapsWalkingURL(stops []planner.Place) string {
	switch len(stops) {
	case 0:
		return ""
	case 1:
		return "https://www.google.com/maps/search/?" + url.Values{
			"api":   {"1"},
			"query": {coordPair(stops[0])},
		}.Encode()
	}

	v := url.Values{
		"api":         {"1"},
		"travelmode":  {"walking"},
		"origin":      {coordPair(stops[0])},
		"destination": {coordPair(stops[len(stops)-1])},
	}

	if middle := stops[1 : len(stops)-1]; len(middle) > 0 {
		waypoints := make([]string, len(middle))
		for i, stop := range middle {
			waypoints[i] = coordPair(stop)
		}
		v.Set("waypoints", strings.Join(waypoints, "|"))
	}

	return "https://www.google.com/maps/dir/?" + v.Encode()
}

// coordPair is "lat,lon", the format every Google Maps URL parameter here expects.
func coordPair(p planner.Place) string {
	return fmt.Sprintf("%f,%f", p.Lat, p.Lon)
}
