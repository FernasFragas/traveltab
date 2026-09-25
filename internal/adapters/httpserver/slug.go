package httpserver

import "weatherservice/internal/slug"

// Slug turns a city and its country into the /trip/:slug path segment: both lower-cased, the
// city's spaces turned to hyphens, the two-letter country appended after its own hyphen.
// Slug("Mexico City", "MX") is "mexico-city-mx". The format lives in internal/slug.
func Slug(city, country string) string {
	return slug.Build(city, country)
}

// ParseSlug is Slug's inverse: it splits on the last hyphen-delimited segment, which it takes
// to be the two-letter country, and turns the hyphens in what remains back into spaces for the
// city. It reports ok=false for anything that doesn't end in such a suffix.
func ParseSlug(s string) (city, country string, ok bool) {
	return slug.Parse(s)
}
