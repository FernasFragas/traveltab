package main

import (
	"strings"
	"sync/atomic"

	guidedata "weatherservice/guides"
	app "weatherservice/internal/application"
	"weatherservice/internal/guides"
)

// fixtureGuides is the preview's city guide source. It answers from the reviewed guides embedded
// in the binary, through the same guides.Book the site uses, so a screenshot shows the real text
// and attribution at their real length, and adds one fixture-only destination with awkward
// content for layout checks. It reads no file at request time and never contacts a model.
//
// Which outcome a search gets depends on the resolved city and country, like the real source:
//
//	Lisbon, Porto, Paris (FR), Tokyo, Tavira   a reviewed guide with its attribution
//	Paris (US), Coimbra, Nophoto and any other  no reviewed guide: the explicit missing state
//	Longguide (PT)                              fixture-only text with an unbroken 93-character
//	                                            token, a long article title and markup characters
//
// The fallback scenario has no guides at all, so it shows the missing state everywhere.
type fixtureGuides struct {
	book     *guides.Book
	scenario string
	calls    atomic.Int64
}

func newFixtureGuides(scenario string) *fixtureGuides {
	book, err := guides.ParseBook(guidedata.JSON)
	if err != nil {
		panic("design preview: embedded guides.json is invalid: " + err.Error())
	}

	return &fixtureGuides{book: book, scenario: scenario}
}

// Guide implements app.CityGuideSource.
func (f *fixtureGuides) Guide(city, country string) (app.CityGuide, bool) {
	f.calls.Add(1)

	if f.scenario == "fallback" {
		return app.CityGuide{}, false
	}

	if strings.EqualFold(strings.TrimSpace(city), "Longguide") && strings.EqualFold(strings.TrimSpace(country), "pt") {
		return app.CityGuide{
			Intro: "FIXTURE ONLY. " + strings.Repeat("Unbroken", 11) + "Token wider than a phone screen, followed by <b>markup characters</b> & an ampersand.\n\n" +
				"A second paragraph, so the spacing between paragraphs is visible.",
			SourceTitle:    "A very long article title used to check that attribution wraps on narrow screens without overflowing",
			SourceRevision: 1234567890123,
			ReviewedOn:     "2026-09-24",
		}, true
	}

	return f.book.Guide(city, country)
}
