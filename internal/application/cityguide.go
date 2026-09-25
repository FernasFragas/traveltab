package application

// CityGuide is a reviewed city intro with the source details its license requires. It exists only
// for text a person has checked against its source: nothing generated and unreviewed is ever a
// CityGuide.
type CityGuide struct {
	Intro          string // plain text; paragraphs are separated by a blank line
	SourceTitle    string // title of the Wikivoyage article the intro is adapted from
	SourceRevision int64  // the article revision that was reviewed
	ReviewedOn     string // YYYY-MM-DD
}

// CityGuideSource looks up the reviewed guide for a destination. Implementations answer from data
// already in memory: serving a request must never generate text, call a model or reach the
// network.
type CityGuideSource interface {
	// Guide returns the reviewed guide for the city in the country (a two-letter ISO code), or
	// false when none is published. Matching is by city and country together, so a namesake city
	// in another country never receives another city's guide.
	Guide(city, country string) (CityGuide, bool)
}
