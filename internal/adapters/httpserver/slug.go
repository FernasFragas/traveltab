package httpserver

import "strings"

// Slug turns a city and its country into the /trip/:slug path segment: both lower-cased
// ASCII, the city's spaces turned to hyphens, the two-letter country appended after its own
// hyphen. Slug("Mexico City", "MX") is "mexico-city-mx".
func Slug(city, country string) string {
	return hyphenate(city) + "-" + hyphenate(country)
}

// ParseSlug is Slug's inverse: it splits on the last hyphen-delimited segment, which it takes
// to be the two-letter country, and turns the hyphens in what remains back into spaces for the
// city. It reports ok=false for anything that doesn't end in such a suffix.
func ParseSlug(slug string) (city, country string, ok bool) {
	i := strings.LastIndex(slug, "-")
	if i <= 0 || i == len(slug)-1 {
		return "", "", false
	}

	country = slug[i+1:]
	if !isTwoLetterCountry(country) {
		return "", "", false
	}

	city = strings.ReplaceAll(slug[:i], "-", " ")
	if city == "" {
		return "", "", false
	}

	return city, country, true
}

// hyphenate lower-cases s and turns its runs of whitespace into single hyphens.
func hyphenate(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), "-")
}

// isTwoLetterCountry reports whether s is exactly two lower-case ASCII letters, the shape
// every country code Slug builds has.
func isTwoLetterCountry(s string) bool {
	if len(s) != 2 {
		return false
	}
	for _, r := range s {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}
