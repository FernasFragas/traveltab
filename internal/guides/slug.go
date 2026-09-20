package guides

import "strings"

// Slug builds the "city-country" key guides.json is keyed by: both lower-cased ASCII, spaces in
// the city name turned to hyphens, e.g. Slug("Mexico City", "MX") -> "mexico-city-mx".
//
// This independently matches the slug format tasks/plan-v2.md settles on for the app's
// shareable trip links (Lane A, internal/adapters/httpserver). It is duplicated here rather than
// imported: a cmd/-adjacent package pulling in internal/adapters/httpserver would cross a layer
// boundary the app doesn't otherwise allow (see docs/architecture.md). If Lane A's slug.go
// lands, a later cleanup could hoist this into a shared, dependency-free package both sides
// import.
func Slug(city, country string) string {
	c := strings.ToLower(strings.TrimSpace(city))
	c = strings.Join(strings.Fields(c), "-")

	return c + "-" + strings.ToLower(strings.TrimSpace(country))
}
