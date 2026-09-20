package httpserver

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlug_BuildsCityDashCountry(t *testing.T) {
	assert.Equal(t, "lisbon-pt", Slug("Lisbon", "pt"))
}

func TestSlug_LowercasesAndHyphenatesSpaces(t *testing.T) {
	assert.Equal(t, "mexico-city-mx", Slug("Mexico City", "MX"))
}

func TestParseSlug_RecoversCityAndCountry(t *testing.T) {
	city, country, ok := ParseSlug("mexico-city-mx")

	require.True(t, ok)
	assert.Equal(t, "mexico city", city)
	assert.Equal(t, "mx", country)
}

func TestParseSlug_RejectsASlugWithNoCountrySuffix(t *testing.T) {
	for _, slug := range []string{"lisbon", "lisbon-portugal", "", "-pt", "lisbon-", "lisbon-p"} {
		_, _, ok := ParseSlug(slug)
		assert.False(t, ok, "slug=%q", slug)
	}
}

// Slug/ParseSlug must round-trip for every ASCII city+country pair the existing fixtures use.
func TestSlug_RoundTripsExistingFixtureCities(t *testing.T) {
	for _, tc := range []struct{ city, country string }{
		{"Lisbon", "pt"},
		{"Tavira", "pt"},
		{"Kyoto", "jp"},
	} {
		slug := Slug(tc.city, tc.country)
		city, country, ok := ParseSlug(slug)

		require.True(t, ok, "slug=%q", slug)
		assert.Equal(t, strings.ToLower(tc.city), city, "slug=%q", slug)
		assert.Equal(t, tc.country, country, "slug=%q", slug)
	}
}
