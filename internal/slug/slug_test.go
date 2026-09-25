package slug

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	for _, tc := range []struct{ name, city, country, want string }{
		{"single word", "Lisbon", "pt", "lisbon-pt"},
		{"upper-case country", "Lisbon", "PT", "lisbon-pt"},
		{"multiword", "Mexico City", "MX", "mexico-city-mx"},
		{"three words", "Rio de Janeiro", "BR", "rio-de-janeiro-br"},
		{"surrounding and repeated whitespace", "  New   York ", " us ", "new-york-us"},
		{"diacritics are kept, only lower-cased", "São Paulo", "BR", "são-paulo-br"},
		{"existing hyphens are kept", "Stratford-upon-Avon", "GB", "stratford-upon-avon-gb"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, Build(tc.city, tc.country))
		})
	}
}

func TestParse(t *testing.T) {
	city, country, ok := Parse("mexico-city-mx")

	require.True(t, ok)
	assert.Equal(t, "mexico city", city)
	assert.Equal(t, "mx", country)
}

func TestParse_RejectsASlugWithNoCountrySuffix(t *testing.T) {
	for _, s := range []string{"lisbon", "lisbon-portugal", "", "-pt", "lisbon-", "lisbon-p", "lisbon-p1", "lisbon-PT"} {
		_, _, ok := Parse(s)
		assert.False(t, ok, "slug=%q", s)
	}
}

func TestBuildThenParse_RoundTrips(t *testing.T) {
	for _, tc := range []struct{ city, country, wantCity string }{
		{"Lisbon", "PT", "lisbon"},
		{"Mexico City", "MX", "mexico city"},
		{"New York", "US", "new york"},
		{"Rio de Janeiro", "BR", "rio de janeiro"},
		{"São Paulo", "BR", "são paulo"},
		{"Zürich", "CH", "zürich"},
		{"Kraków", "PL", "kraków"},
	} {
		s := Build(tc.city, tc.country)
		city, country, ok := Parse(s)

		require.True(t, ok, "slug=%q", s)
		assert.Equal(t, tc.wantCity, city, "slug=%q", s)
		assert.Equal(t, strings.ToLower(tc.country), country, "slug=%q", s)
		assert.Equal(t, s, Build(city, country), "second trip must be stable, slug=%q", s)
	}
}

// Every key in guides/guides.json must survive Parse then Build unchanged, so guide keys and
// stored /trip/:slug links stay the same string.
func TestGuideKeysRoundTrip(t *testing.T) {
	raw, err := os.ReadFile("../../guides/guides.json")
	require.NoError(t, err)

	var guides map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &guides))
	require.NotEmpty(t, guides)

	for key := range guides {
		city, country, ok := Parse(key)

		require.True(t, ok, "key=%q", key)
		assert.Equal(t, key, Build(city, country), "key=%q", key)
	}
}
