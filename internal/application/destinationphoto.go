package application

import (
	"context"
	"math"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
	"weatherservice/internal/planner"
)

// DestinationMatchKM is how far apart two records of one destination may be. Namesake cities in
// one country are normally much further apart, and a geocoder's centre for a city rarely differs
// from Wikidata's by more than a few kilometres.
const DestinationMatchKM = 25

// DestinationIdentity is what a search resolved to. Photos and cached pages are only ever reused
// for the same identity, never for the same name.
type DestinationIdentity struct {
	City, Country string // resolved name and normalized ISO 3166-1 alpha-2 country code
	Lat, Lon      float64
}

// DestinationPhoto is one attributed photograph of a destination. URL is an image the browser
// loads; CreditURL is the page that names its author and license.
type DestinationPhoto struct {
	URL, Alt, Credit, CreditURL string
	License, LicenseURL         string
	Width, Height               int
	SourceID                    string // the city entity or article the photograph was selected for
}

// DestinationPhotoSource finds a photograph of a destination. It returns (nil, nil) only when a
// completed lookup found no suitable image; timeouts, malformed responses, rate limits and
// upstream failures are errors.
type DestinationPhotoSource interface {
	Photo(context.Context, DestinationIdentity) (*DestinationPhoto, error)
}

// NewDestinationIdentity normalizes and validates a resolved destination. It reports false when
// the country is not a two-letter code or the coordinates are unusable, so nothing that could
// belong to another place is ever looked up. (0, 0) counts as unusable: it is what a record
// without coordinates decodes to.
func NewDestinationIdentity(city, country string, lat, lon float64) (DestinationIdentity, bool) {
	id := DestinationIdentity{
		City:    strings.Join(strings.Fields(city), " "),
		Country: strings.ToUpper(strings.TrimSpace(country)),
		Lat:     lat,
		Lon:     lon,
	}

	if id.City == "" || len(id.Country) != 2 || !isLetters(id.Country) {
		return DestinationIdentity{}, false
	}

	if math.IsNaN(lat) || math.IsNaN(lon) || math.IsInf(lat, 0) || math.IsInf(lon, 0) ||
		lat < -90 || lat > 90 || lon < -180 || lon > 180 || lat == 0 && lon == 0 {
		return DestinationIdentity{}, false
	}

	return id, true
}

// Matches reports whether other is the same destination: the same country, the same name up to
// case and diacritics, and coordinates within DestinationMatchKM.
func (id DestinationIdentity) Matches(other DestinationIdentity) bool {
	return id.Country == other.Country &&
		FoldName(id.City) == FoldName(other.City) &&
		planner.DistanceKM(planner.Place{Lat: id.Lat, Lon: id.Lon}, planner.Place{Lat: other.Lat, Lon: other.Lon}) <= DestinationMatchKM
}

// Complete reports whether the photo carries everything the page shows and must credit.
func (p DestinationPhoto) Complete() bool {
	return p.URL != "" && p.Alt != "" && p.Credit != "" && p.CreditURL != "" && p.License != "" &&
		p.Width > 0 && p.Height > 0 && p.SourceID != ""
}

// FoldName lower-cases a place name and drops diacritics and punctuation, so "São Paulo",
// "sao paulo" and "SAO-PAULO" compare equal.
func FoldName(name string) string {
	var out strings.Builder

	for _, r := range norm.NFKD.String(name) {
		switch {
		case unicode.Is(unicode.Mn, r):
			continue
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			out.WriteString(strings.ToLower(latinFolds.Replace(string(r))))
		default:
			out.WriteByte(' ')
		}
	}

	return strings.Join(strings.Fields(out.String()), " ")
}

// latinFolds covers the letters that do not decompose into a base letter and a mark.
var latinFolds = strings.NewReplacer("ł", "l", "Ł", "l", "ø", "o", "Ø", "o", "đ", "d", "Đ", "d", "ß", "ss", "æ", "ae", "Æ", "ae", "œ", "oe", "Œ", "oe")

func isLetters(s string) bool {
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}

	return true
}
