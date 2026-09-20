package guides

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCities_HasExactlyTheTwentyStarterCities(t *testing.T) {
	assert.Len(t, Cities, 20)

	names := make(map[string]bool, len(Cities))
	for _, c := range Cities {
		names[c.Name] = true
	}

	for _, want := range []string{
		"Lisbon", "Porto", "Tavira", "Funchal", "Kyoto", "Paris", "Rome", "Barcelona", "London",
		"New York", "Tokyo", "Istanbul", "Prague", "Amsterdam", "Berlin", "Seville", "Florence",
		"Vienna", "Marrakesh", "Mexico City",
	} {
		assert.True(t, names[want], "missing city %q", want)
	}
}

func TestCity_WikivoyageTitle_OverridesNewYork(t *testing.T) {
	c := City{Name: "New York", Country: "us"}
	assert.Equal(t, "New York City", c.WikivoyageTitle())
}

func TestCity_WikivoyageTitle_DefaultsToName(t *testing.T) {
	c := City{Name: "Lisbon", Country: "pt"}
	assert.Equal(t, "Lisbon", c.WikivoyageTitle())
}

func TestCity_Slug_MatchesCityDashCountry(t *testing.T) {
	c := City{Name: "Mexico City", Country: "mx"}
	assert.Equal(t, "mexico-city-mx", c.Slug())
}
