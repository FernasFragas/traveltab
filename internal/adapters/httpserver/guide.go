package httpserver

import (
	"net/url"
	"strconv"
	"strings"
	"weatherservice/internal/application"
)

const (
	wikivoyageWikiURL  = "https://en.wikivoyage.org/wiki/"
	wikivoyageIndexURL = "https://en.wikivoyage.org/w/index.php"
	guideLicenseName   = "CC BY-SA 4.0"
	guideLicenseURL    = "https://creativecommons.org/licenses/by-sa/4.0/"
)

// SetCityGuideSource switches reviewed city guides on. Without a source, or for a destination the
// source has no guide for, the page shows the "no guide yet" state instead. The source answers
// from memory: nothing here ever generates a guide.
func (s *Server) SetCityGuideSource(source application.CityGuideSource) {
	s.guideSource = source
}

// guideView is what the guide_card template renders. It has two states: Available, a reviewed
// intro with the attribution its license requires, or a plain statement that there is no
// reviewed guide for this destination yet. The two are never mixed, and the text is always
// plain, escaped by the template.
type guideView struct {
	Available bool
	City      string // the destination the card is for, as resolved by this response

	// Available only.
	Paragraphs                       []string
	SourceTitle                      string
	SourcePageURL, SourceRevisionURL string
	SourceHistoryURL                 string
	SourceRevision                   int64
	ReviewedOn                       string
	LicenseName, LicenseURL          string

	// Unavailable only: a Wikivoyage search for the city, which always resolves.
	SearchURL string
}

// guideFor builds the guide state for one resolved destination. The lookup is by city and country
// together, so it can only ever return the guide written for this destination.
func (s *Server) guideFor(info application.GeneralWeatherInfo) guideView {
	city := strings.Join(strings.Fields(info.City), " ")
	view := guideView{City: city}

	if city != "" {
		view.SearchURL = wikivoyageIndexURL + "?" + url.Values{"search": {city}}.Encode()
	}

	if s.guideSource == nil {
		return view
	}

	guide, ok := s.guideSource.Guide(info.City, info.Country)
	if !ok {
		return view
	}

	paragraphs := guideParagraphs(guide.Intro)
	title := strings.TrimSpace(guide.SourceTitle)

	if len(paragraphs) == 0 || title == "" || guide.SourceRevision <= 0 {
		return view // attribution is a condition of showing the text at all
	}

	revision := url.Values{"title": {title}, "oldid": {strconv.FormatInt(guide.SourceRevision, 10)}}
	history := url.Values{"title": {title}, "action": {"history"}}

	view.Available = true
	view.Paragraphs = paragraphs
	view.SourceTitle = title
	view.SourcePageURL = wikivoyageWikiURL + url.PathEscape(strings.ReplaceAll(title, " ", "_"))
	view.SourceRevisionURL = wikivoyageIndexURL + "?" + revision.Encode()
	view.SourceHistoryURL = wikivoyageIndexURL + "?" + history.Encode()
	view.SourceRevision = guide.SourceRevision
	view.ReviewedOn = guide.ReviewedOn
	view.LicenseName, view.LicenseURL = guideLicenseName, guideLicenseURL
	view.SearchURL = ""

	return view
}

// guideParagraphs splits an intro into paragraphs at blank lines.
func guideParagraphs(intro string) []string {
	intro = strings.ReplaceAll(intro, "\r\n", "\n")

	var paragraphs []string

	for _, block := range strings.Split(intro, "\n\n") {
		if text := strings.Join(strings.Fields(block), " "); text != "" {
			paragraphs = append(paragraphs, text)
		}
	}

	return paragraphs
}
