package guides

import "strings"

// Slug builds the "city-country" key guides.json is keyed by: both lower-cased ASCII, spaces in
// the city name turned to hyphens, e.g. Slug("Mexico City", "MX") -> "mexico-city-mx".
//
// This matches the HTTP adapter's shareable trip format without depending on that adapter.
// See docs/maintenance.md for the shared-helper follow-up.
func Slug(city, country string) string {
	c := strings.ToLower(strings.TrimSpace(city))
	c = strings.Join(strings.Fields(c), "-")

	return c + "-" + strings.ToLower(strings.TrimSpace(country))
}
