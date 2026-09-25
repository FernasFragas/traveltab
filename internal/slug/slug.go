// Package slug builds and parses the "city-country" key shared by the public /trip/:slug URLs
// and guides/guides.json. It has no dependencies on adapters, so the HTTP server and the guide
// generator can both use it.
//
// The format is the lower-cased city with each run of whitespace turned into one hyphen, then a
// hyphen, then the lower-cased two-letter country code: Build("Mexico City", "MX") is
// "mexico-city-mx". Only case and whitespace are normalised. Other characters, including
// diacritics ("São Paulo" -> "são-paulo-br"), pass through unchanged, and a city that already
// contains hyphens ("Stratford-upon-Avon") is not distinguishable from a spaced one, so Parse
// returns it with spaces. Changing any of this would break stored links and guide keys.
package slug

import "strings"

// Build turns a city and its country code into a slug.
func Build(city, country string) string {
	return hyphenate(city) + "-" + hyphenate(country)
}

// Parse is Build's inverse: it splits on the last hyphen, which it takes to precede the
// two-letter country, and turns the remaining hyphens back into spaces for the city. It
// reports ok=false for anything that doesn't end in such a suffix.
func Parse(s string) (city, country string, ok bool) {
	i := strings.LastIndex(s, "-")
	if i <= 0 || i == len(s)-1 {
		return "", "", false
	}

	country = s[i+1:]
	if !isTwoLetterCountry(country) {
		return "", "", false
	}

	city = strings.ReplaceAll(s[:i], "-", " ")
	if city == "" {
		return "", "", false
	}

	return city, country, true
}

// hyphenate lower-cases s and turns its runs of whitespace into single hyphens, dropping any
// leading or trailing whitespace.
func hyphenate(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), "-")
}

// isTwoLetterCountry reports whether s is exactly two lower-case ASCII letters, the shape
// every country code Build produces has.
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
