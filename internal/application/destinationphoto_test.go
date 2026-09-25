package application

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDestinationIdentity(t *testing.T) {
	for name, tc := range map[string]struct {
		city, country string
		lat, lon      float64
		want          DestinationIdentity
		ok            bool
	}{
		"normalizes":        {"  Sao   Paulo ", " br ", -23.55, -46.63, DestinationIdentity{City: "Sao Paulo", Country: "BR", Lat: -23.55, Lon: -46.63}, true},
		"already normal":    {"Paris", "FR", 48.85, 2.35, DestinationIdentity{City: "Paris", Country: "FR", Lat: 48.85, Lon: 2.35}, true},
		"country name":      {"Lisbon", "portugal", 38.7, -9.1, DestinationIdentity{}, false},
		"three letters":     {"Lisbon", "PRT", 38.7, -9.1, DestinationIdentity{}, false},
		"digits":            {"Lisbon", "P1", 38.7, -9.1, DestinationIdentity{}, false},
		"no country":        {"Lisbon", "", 38.7, -9.1, DestinationIdentity{}, false},
		"no city":           {"  ", "PT", 38.7, -9.1, DestinationIdentity{}, false},
		"null island":       {"Lisbon", "PT", 0, 0, DestinationIdentity{}, false},
		"latitude range":    {"Lisbon", "PT", 91, 0, DestinationIdentity{}, false},
		"longitude range":   {"Lisbon", "PT", 38, -181, DestinationIdentity{}, false},
		"NaN":               {"Lisbon", "PT", math.NaN(), 1, DestinationIdentity{}, false},
		"infinite":          {"Lisbon", "PT", 1, math.Inf(1), DestinationIdentity{}, false},
		"equator, meridian": {"Quito", "EC", 0, -78.5, DestinationIdentity{City: "Quito", Country: "EC", Lat: 0, Lon: -78.5}, true},
	} {
		t.Run(name, func(t *testing.T) {
			got, ok := NewDestinationIdentity(tc.city, tc.country, tc.lat, tc.lon)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDestinationIdentityMatches(t *testing.T) {
	paris := DestinationIdentity{City: "Paris", Country: "FR", Lat: 48.8589, Lon: 2.32}

	for name, tc := range map[string]struct {
		other DestinationIdentity
		want  bool
	}{
		"same":                  {paris, true},
		"case and diacritics":   {DestinationIdentity{City: "PÀRIS", Country: "FR", Lat: 48.8589, Lon: 2.32}, true},
		"a few kilometres away": {DestinationIdentity{City: "Paris", Country: "FR", Lat: 48.8566, Lon: 2.3522}, true},
		"Paris, Texas":          {DestinationIdentity{City: "Paris", Country: "US", Lat: 33.6609, Lon: -95.5555}, false},
		"same country, far":     {DestinationIdentity{City: "Paris", Country: "FR", Lat: 43.3, Lon: 5.37}, false},
		"another name":          {DestinationIdentity{City: "Lyon", Country: "FR", Lat: 48.8589, Lon: 2.32}, false},
		"same name, 30 km off":  {DestinationIdentity{City: "Paris", Country: "FR", Lat: 48.8589 + 0.27, Lon: 2.32}, false},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, paris.Matches(tc.other))
			assert.Equal(t, tc.want, tc.other.Matches(paris), "matching is symmetric")
		})
	}
}

func TestFoldName(t *testing.T) {
	for in, want := range map[string]string{
		"São Paulo": "sao paulo", "SAO-PAULO": "sao paulo", "  sao   paulo ": "sao paulo", "München": "munchen",
		"Kraków": "krakow", "Łódź": "lodz", "Reykjavík": "reykjavik", "Saint-Denis": "saint denis", "St. John's": "st john s",
		"Zürich": "zurich", "Tōkyō": "tokyo", "東京": "東京", "": "", "---": "",
	} {
		assert.Equal(t, want, FoldName(in), in)
	}
}

func TestDestinationPhotoComplete(t *testing.T) {
	full := DestinationPhoto{URL: "u", Alt: "a", Credit: "c", CreditURL: "cu", License: "l", Width: 1, Height: 1, SourceID: "Q1"}
	assert.True(t, full.Complete())

	full.LicenseURL = "" // the license link is optional: public-domain files have none
	assert.True(t, full.Complete())

	for name, edit := range map[string]func(*DestinationPhoto){
		"url": func(p *DestinationPhoto) { p.URL = "" }, "alt": func(p *DestinationPhoto) { p.Alt = "" },
		"credit": func(p *DestinationPhoto) { p.Credit = "" }, "credit url": func(p *DestinationPhoto) { p.CreditURL = "" },
		"license": func(p *DestinationPhoto) { p.License = "" }, "width": func(p *DestinationPhoto) { p.Width = 0 },
		"height": func(p *DestinationPhoto) { p.Height = 0 }, "source": func(p *DestinationPhoto) { p.SourceID = "" },
	} {
		p := full
		edit(&p)
		assert.False(t, p.Complete(), name)
	}
}
