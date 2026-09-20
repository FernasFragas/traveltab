package guides

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlug_BuildsCityDashCountry(t *testing.T) {
	assert.Equal(t, "lisbon-pt", Slug("Lisbon", "pt"))
}

func TestSlug_LowercasesAndHyphenatesSpaces(t *testing.T) {
	assert.Equal(t, "mexico-city-mx", Slug("Mexico City", "MX"))
}

func TestSlug_MultiWordCityWithUppercaseCountry(t *testing.T) {
	assert.Equal(t, "new-york-us", Slug("New York", "US"))
}
