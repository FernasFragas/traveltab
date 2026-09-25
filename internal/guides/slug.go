package guides

import "weatherservice/internal/slug"

// Slug builds the "city-country" key guides.json is keyed by, e.g. Slug("Mexico City", "MX") ->
// "mexico-city-mx". It is the same format as the HTTP adapter's shareable /trip/:slug segment;
// both delegate to internal/slug, so the generator doesn't depend on the adapter.
func Slug(city, country string) string {
	return slug.Build(city, country)
}
