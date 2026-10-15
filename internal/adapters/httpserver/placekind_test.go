package httpserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// As a traveller, I want a place's type in plain words, so that "tower" reads as a landmark like the mockup.
func TestPlaceKindLabel_GroupsAndTitleCases(t *testing.T) {
	cases := map[string]string{
		"tower":          "Historic Landmark",
		"Castle":         "Historic Landmark",
		"monument":       "Historic Landmark",
		"church":         "Religious Site",
		"monastery":      "Religious Site",
		"museum":         "Museum",
		"viewpoint":      "Viewpoint",
		"nature reserve": "Nature Reserve",
		"":               "",
	}
	for kind, want := range cases {
		assert.Equal(t, want, placeKindLabel(kind), "kind %q", kind)
	}
}
