package httpserver

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// placeKindGroups turns raw Wikidata kinds into the plain-words labels travellers see.
var placeKindGroups = map[string]string{
	"tower": "Historic Landmark", "castle": "Historic Landmark", "fort": "Historic Landmark",
	"fortress": "Historic Landmark", "monument": "Historic Landmark", "palace": "Historic Landmark",
	"memorial": "Historic Landmark", "ruins": "Historic Landmark",
	"church": "Religious Site", "cathedral": "Religious Site", "monastery": "Religious Site",
	"chapel": "Religious Site", "basilica": "Religious Site", "convent": "Religious Site",
}

// placeKindLabel returns a place's kind as a display label; the raw kind still drives icons and tips.
func placeKindLabel(kind string) string {
	if label, ok := placeKindGroups[strings.ToLower(kind)]; ok {
		return label
	}
	return cases.Title(language.English).String(kind)
}
