package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"weatherservice/internal/application"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	parisFR = application.DestinationIdentity{City: "Paris", Country: "FR", Lat: 48.8589, Lon: 2.3200}
	parisTX = application.DestinationIdentity{City: "Paris", Country: "US", Lat: 33.6609, Lon: -95.5555}
)

// stubHit is one wbsearchentities result; match is the label or alias the search matched.
type stubHit struct{ id, label, match string }

// stubEntity is a Wikidata entity, reduced to what the adapter reads.
type stubEntity struct {
	lat, lon    float64
	noCoords    bool
	label       string
	aliases     []string
	countries   []string
	types       []string
	populated   bool
	sitelinks   int
	images      []string
	article     string
	description string
}

// stubFile is Commons image information for one file.
type stubFile struct {
	mime          string
	width, height int
	artist        string
	license       string
	licenseURL    string
	nonFree       string
	restrictions  string
	thumb         string // "" builds a Commons thumbnail URL
	description   string // "" builds a Commons file page URL
	missing       bool
}

// photoWorld is a small offline Wikidata, Wikipedia and Commons. Everything the adapter asks is
// answered from these tables, and every request is recorded.
type photoWorld struct {
	mu         sync.Mutex
	search     map[string][]stubHit
	entities   map[string]stubEntity
	countries  map[string]string
	nearby     []stubHit
	nearSearch []stubHit
	pageImage  map[string]string // article title -> file name
	files      map[string]stubFile
	redirects  map[string]string // file title -> target file title
	requests   []*http.Request
	override   func(*http.Request) *stubResponse
}

func newPhotoWorld() *photoWorld {
	return &photoWorld{
		search:    map[string][]stubHit{},
		entities:  map[string]stubEntity{},
		countries: map[string]string{"Q142": "FR", "Q30": "US", "Q45": "PT", "Q17": "JP"},
		pageImage: map[string]string{},
		files:     map[string]stubFile{},
		redirects: map[string]string{},
	}
}

func goodFile() stubFile {
	return stubFile{mime: "image/jpeg", width: 4000, height: 2500, artist: "Ada Photographer", license: "CC BY-SA 4.0", licenseURL: "https://creativecommons.org/licenses/by-sa/4.0"}
}

func (w *photoWorld) api(t *testing.T) *DestinationPhotoAPI {
	t.Helper()

	return NewDestinationPhotoAPI(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if err := req.Context().Err(); err != nil {
			return nil, err
		}

		w.mu.Lock()
		w.requests = append(w.requests, req)
		w.mu.Unlock()

		stub := stubResponse{status: http.StatusOK, body: w.respond(req)}
		if w.override != nil {
			if replaced := w.override(req); replaced != nil {
				stub = *replaced
			}
		}

		return &http.Response{StatusCode: stub.status, Body: io.NopCloser(strings.NewReader(stub.body)), Header: http.Header{}}, nil
	})})
}

func (w *photoWorld) count(action string) int {
	w.mu.Lock()
	defer w.mu.Unlock()

	n := 0

	for _, req := range w.requests {
		q := req.URL.Query()
		if q.Get("action") == action || action == "geosearch" && q.Get("generator") == "geosearch" ||
			action == "pageimages" && q.Get("prop") == "pageimages" || action == "imageinfo" && q.Get("prop") == "imageinfo" {
			n++
		}
	}

	return n
}

func (w *photoWorld) respond(req *http.Request) string {
	q := req.URL.Query()

	switch {
	case q.Get("action") == "wbsearchentities":
		return w.searchJSON(q.Get("search"))
	case q.Get("action") == "wbgetentities":
		return w.entitiesJSON(strings.Split(q.Get("ids"), "|"))
	case q.Get("action") == "wbgetclaims":
		return w.claimsJSON(q.Get("entity"))
	case q.Get("generator") == "geosearch":
		return w.nearbyJSON()
	case q.Get("action") == "query" && q.Get("list") == "search":
		return w.nearSearchJSON()
	case q.Get("prop") == "pageimages":
		return w.pageImageJSON(q.Get("titles"))
	case q.Get("prop") == "imageinfo":
		return w.imageInfoJSON(strings.Split(q.Get("titles"), "|"))
	}

	return `{"error":{"code":"unknown","info":"unexpected request ` + req.URL.String() + `"}}`
}

func mustJSON(v any) string {
	out, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	return string(out)
}

func (w *photoWorld) searchJSON(name string) string {
	var hits []map[string]any

	for _, hit := range w.search[name] {
		hits = append(hits, map[string]any{"id": hit.id, "label": hit.label, "match": map[string]string{"type": "label", "language": "en", "text": hit.match}})
	}

	return mustJSON(map[string]any{"search": hits})
}

func appendClaim(list any, claim map[string]any) []any {
	existing, _ := list.([]any)

	return append(existing, claim)
}

func item(id string) map[string]any {
	return map[string]any{"mainsnak": map[string]any{"datavalue": map[string]any{"value": map[string]string{"id": id}}}, "rank": "normal"}
}

func (w *photoWorld) entitiesJSON(ids []string) string {
	entities := map[string]any{}

	for _, id := range ids {
		e, ok := w.entities[id]
		if !ok {
			continue
		}

		claims := map[string]any{}
		if !e.noCoords {
			claims["P625"] = []any{map[string]any{"rank": "normal", "mainsnak": map[string]any{"datavalue": map[string]any{"value": map[string]any{
				"latitude": e.lat, "longitude": e.lon, "globe": "http://www.wikidata.org/entity/Q2"}}}}}
		}

		for _, c := range e.countries {
			claims["P17"] = appendClaim(claims["P17"], item(c))
		}

		for _, ty := range e.types {
			claims["P31"] = appendClaim(claims["P31"], item(ty))
		}

		if e.populated {
			claims["P1082"] = []any{map[string]any{"rank": "normal", "mainsnak": map[string]any{"datavalue": map[string]any{"value": map[string]string{"amount": "+1000"}}}}}
		}

		for _, image := range e.images {
			claims["P18"] = appendClaim(claims["P18"], map[string]any{"rank": "normal", "mainsnak": map[string]any{"datavalue": map[string]any{"value": image}}})
		}

		sitelinks := map[string]any{}

		for i := 0; i < e.sitelinks; i++ {
			sitelinks[fmt.Sprintf("site%dwiki", i)] = map[string]string{"title": "x"}
		}

		if e.article != "" {
			sitelinks["enwiki"] = map[string]string{"site": "enwiki", "title": e.article}
		}

		entity := map[string]any{"id": id, "claims": claims, "sitelinks": sitelinks}
		if e.label != "" {
			entity["labels"] = map[string]any{"en": map[string]string{"value": e.label}}
		}
		if len(e.aliases) > 0 {
			var aliases []map[string]string
			for _, alias := range e.aliases {
				aliases = append(aliases, map[string]string{"value": alias})
			}
			entity["aliases"] = map[string]any{"en": aliases}
		}
		if e.description != "" {
			entity["descriptions"] = map[string]any{"en": map[string]string{"language": "en", "value": e.description}}
		}

		entities[id] = entity
	}

	return mustJSON(map[string]any{"entities": entities})
}

func (w *photoWorld) claimsJSON(qid string) string {
	code, ok := w.countries[qid]
	if !ok || code == "" {
		return `{"claims":{}}`
	}

	return mustJSON(map[string]any{"claims": map[string]any{"P297": []any{map[string]any{"rank": "normal", "mainsnak": map[string]any{"datavalue": map[string]any{"value": code}}}}}})
}

func (w *photoWorld) nearbyJSON() string {
	var pages []map[string]any

	for _, hit := range w.nearby {
		pages = append(pages, map[string]any{"title": hit.label, "pageprops": map[string]string{"wikibase_item": hit.id}})
	}

	return mustJSON(map[string]any{"query": map[string]any{"pages": pages}})
}

func (w *photoWorld) nearSearchJSON() string {
	var hits []map[string]string
	for _, hit := range w.nearSearch {
		hits = append(hits, map[string]string{"title": hit.id})
	}
	return mustJSON(map[string]any{"query": map[string]any{"search": hits}})
}

func (w *photoWorld) pageImageJSON(title string) string {
	if file, ok := w.pageImage[title]; ok {
		return mustJSON(map[string]any{"query": map[string]any{"pages": []any{map[string]any{"title": title, "pageimage": file}}}})
	}

	return mustJSON(map[string]any{"query": map[string]any{"pages": []any{map[string]any{"title": title}}}})
}

func (w *photoWorld) imageInfoJSON(titles []string) string {
	var pages []any

	var redirects []map[string]string

	for _, title := range titles {
		if target, ok := w.redirects[title]; ok {
			redirects = append(redirects, map[string]string{"from": title, "to": target})
			title = target
		}

		f, ok := w.files[title]
		if !ok || f.missing {
			pages = append(pages, map[string]any{"title": title, "missing": true})
			continue
		}

		name := strings.ReplaceAll(strings.TrimPrefix(title, "File:"), " ", "_")
		thumb := f.thumb

		if thumb == "" {
			thumb = "https://upload.wikimedia.org/wikipedia/commons/thumb/a/ab/" + name + "/1280px-" + name
		}

		desc := f.description
		if desc == "" {
			desc = "https://commons.wikimedia.org/wiki/" + strings.ReplaceAll(title, " ", "_")
		}

		meta := map[string]any{
			"Artist":           map[string]string{"value": f.artist},
			"LicenseShortName": map[string]string{"value": f.license},
			"LicenseUrl":       map[string]string{"value": f.licenseURL},
			"NonFree":          map[string]string{"value": f.nonFree},
			"Restrictions":     map[string]string{"value": f.restrictions},
		}

		pages = append(pages, map[string]any{"title": title, "imageinfo": []any{map[string]any{
			"thumburl": thumb, "thumbwidth": 1280, "thumbheight": 1280 * f.height / f.width,
			"url":   "https://upload.wikimedia.org/wikipedia/commons/a/ab/" + name,
			"width": f.width, "height": f.height, "descriptionurl": desc, "mime": f.mime, "extmetadata": meta}}})
	}

	return mustJSON(map[string]any{"query": map[string]any{"redirects": redirects, "pages": pages}})
}

// parisWorld has Paris, France and Paris, Texas, both matched by the name "Paris", each with a
// photograph on Commons.
func parisWorld() *photoWorld {
	w := newPhotoWorld()
	w.search["Paris"] = []stubHit{
		{"Q90", "Paris", "Paris"}, {"Q483020", "Paris Saint-Germain FC", "Paris Saint-Germain FC"},
		{"Q830149", "Paris", "Paris"}, {"Q18331346", "Paris", "Paris"},
	}
	w.entities["Q90"] = stubEntity{lat: 48.8566, lon: 2.3522, countries: []string{"Q142"}, types: []string{"Q515"}, populated: true, sitelinks: 300, images: []string{"Tour Eiffel.jpg"}, article: "Paris"}
	w.entities["Q830149"] = stubEntity{lat: 33.6625, lon: -95.5477, countries: []string{"Q30"}, populated: true, sitelinks: 60, images: []string{"Main Street Paris Texas.jpg"}, article: "Paris, Texas"}
	w.entities["Q483020"] = stubEntity{lat: 48.84, lon: 2.25, countries: []string{"Q142"}, types: []string{"Q476028"}, images: []string{"PSG logo.jpg"}}
	w.entities["Q18331346"] = stubEntity{noCoords: true}
	w.files["File:Tour Eiffel.jpg"] = goodFile()
	w.files["File:Main Street Paris Texas.jpg"] = goodFile()

	return w
}

func TestDestinationPhoto_ResolvesAttributedPhotoForTheMatchingCity(t *testing.T) {
	w := parisWorld()
	file := goodFile()
	file.artist = `<a href="//commons.wikimedia.org/wiki/User:Ada">Ada <b>Photographer</b></a><script>alert(1)</script><br>with Bob`
	w.files["File:Tour Eiffel.jpg"] = file

	photo, err := w.api(t).Photo(context.Background(), parisFR)
	require.NoError(t, err)
	require.NotNil(t, photo)

	assert.Equal(t, "https://upload.wikimedia.org/wikipedia/commons/thumb/a/ab/Tour_Eiffel.jpg/1280px-Tour_Eiffel.jpg", photo.URL)
	assert.Equal(t, "View of Paris", photo.Alt)
	assert.Equal(t, "Ada Photographer with Bob", photo.Credit, "credit is plain text without markup or script")
	assert.Equal(t, "https://commons.wikimedia.org/wiki/File:Tour_Eiffel.jpg", photo.CreditURL)
	assert.Equal(t, "CC BY-SA 4.0", photo.License)
	assert.Equal(t, "https://creativecommons.org/licenses/by-sa/4.0", photo.LicenseURL)
	assert.Equal(t, 1280, photo.Width)
	assert.Equal(t, 800, photo.Height)
	assert.Equal(t, "Q90", photo.SourceID)
	assert.True(t, photo.Complete())
}

func TestDestinationPhoto_CarriesTheWikidataDescriptionOfTheCity(t *testing.T) {
	w := parisWorld()
	e := w.entities["Q90"]
	e.description = "  capital and largest city of France "
	w.entities["Q90"] = e

	photo, err := w.api(t).Photo(context.Background(), parisFR)
	require.NoError(t, err)
	require.NotNil(t, photo)
	assert.Equal(t, "capital and largest city of France", photo.Description)
}

func TestDestinationPhoto_DescriptionComesFromTheEntityRequestWithoutExtraCalls(t *testing.T) {
	w := parisWorld()
	_, err := w.api(t).Photo(context.Background(), parisFR)
	require.NoError(t, err)

	entityCalls := 0

	for _, req := range w.requests {
		if req.URL.Query().Get("action") == "wbgetentities" {
			entityCalls++

			assert.Contains(t, req.URL.Query().Get("props"), "descriptions")
			assert.Equal(t, "en", req.URL.Query().Get("languages"))
		}
	}

	assert.Equal(t, 1, entityCalls)
}

func TestDestinationPhoto_CityWithoutADescriptionHasNone(t *testing.T) {
	photo, err := parisWorld().api(t).Photo(context.Background(), parisFR)
	require.NoError(t, err)
	require.NotNil(t, photo)
	assert.Empty(t, photo.Description)
}

func TestDestinationPhoto_NamesakeCitiesGetTheirOwnPhotograph(t *testing.T) {
	w := parisWorld()
	api := w.api(t)

	fr, err := api.Photo(context.Background(), parisFR)
	require.NoError(t, err)
	require.NotNil(t, fr)
	assert.Equal(t, "Q90", fr.SourceID)
	assert.Contains(t, fr.URL, "Tour_Eiffel")

	tx, err := api.Photo(context.Background(), parisTX)
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.Equal(t, "Q830149", tx.SourceID)
	assert.Contains(t, tx.URL, "Paris_Texas")
}

func TestDestinationPhoto_RefusesEntitiesOfAnotherCountryOrPlace(t *testing.T) {
	for name, edit := range map[string]func(*photoWorld){
		"wrong country": func(w *photoWorld) {
			e := w.entities["Q90"]
			e.countries = []string{"Q30"}
			w.entities["Q90"] = e
		},
		"unknown country": func(w *photoWorld) {
			e := w.entities["Q90"]
			e.countries = []string{"Q999"}
			w.entities["Q90"] = e
		},
		"no country": func(w *photoWorld) {
			e := w.entities["Q90"]
			e.countries = nil
			w.entities["Q90"] = e
		},
		"beyond 25 km": func(w *photoWorld) {
			e := w.entities["Q90"]
			e.lat, e.lon = 48.60, 2.32 // about 30 km south
			w.entities["Q90"] = e
		},
		"no coordinates": func(w *photoWorld) {
			e := w.entities["Q90"]
			e.noCoords = true
			w.entities["Q90"] = e
		},
	} {
		t.Run(name, func(t *testing.T) {
			w := parisWorld()
			edit(w)

			photo, err := w.api(t).Photo(context.Background(), parisFR)
			require.NoError(t, err)
			assert.Nil(t, photo, "another place's photograph must never be used")
		})
	}
}

func TestDestinationPhoto_KeepsNamesakesInOneCountryApart(t *testing.T) {
	w := newPhotoWorld()
	w.countries["Q30"] = "US"
	w.search["Springfield"] = []stubHit{{"Q1", "Springfield", "Springfield"}, {"Q2", "Springfield", "Springfield"}}
	w.entities["Q1"] = stubEntity{lat: 39.78, lon: -89.65, countries: []string{"Q30"}, populated: true, sitelinks: 80, images: []string{"Illinois.jpg"}} // Illinois
	w.entities["Q2"] = stubEntity{lat: 42.10, lon: -72.59, countries: []string{"Q30"}, populated: true, sitelinks: 70, images: []string{"Massachusetts.jpg"}}
	w.files["File:Illinois.jpg"] = goodFile()
	w.files["File:Massachusetts.jpg"] = goodFile()

	photo, err := w.api(t).Photo(context.Background(), application.DestinationIdentity{City: "Springfield", Country: "US", Lat: 42.1015, Lon: -72.5898})
	require.NoError(t, err)
	require.NotNil(t, photo)
	assert.Equal(t, "Q2", photo.SourceID)
}

func TestDestinationPhoto_MatchesLocalNamesAndDiacritics(t *testing.T) {
	for _, tc := range []struct {
		name  string
		typed string
		label string
		match string
	}{
		{"alias in another language", "Lisboa", "Lisbon", "Lisboa"},
		{"diacritics dropped by the user", "Sao Paulo", "São Paulo", "São Paulo"},
		{"case and punctuation", "SAINT-DENIS", "Saint Denis", "Saint Denis"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newPhotoWorld()
			w.search[tc.typed] = []stubHit{{"Q1", tc.label, tc.match}}
			w.entities["Q1"] = stubEntity{lat: 10, lon: 20, countries: []string{"Q45"}, populated: true, sitelinks: 5, images: []string{"City.jpg"}}
			w.files["File:City.jpg"] = goodFile()

			photo, err := w.api(t).Photo(context.Background(), application.DestinationIdentity{City: tc.typed, Country: "pt", Lat: 10.01, Lon: 20.01})
			require.NoError(t, err)
			require.NotNil(t, photo)
			assert.Equal(t, "View of "+tc.typed, photo.Alt)
		})
	}
}

func TestDestinationPhoto_IgnoresHitsWhoseNameDoesNotMatch(t *testing.T) {
	w := newPhotoWorld()
	w.search["Porto"] = []stubHit{{"Q40269", "Porto Alegre", "Porto Alegre"}}
	w.entities["Q40269"] = stubEntity{lat: 41.15, lon: -8.61, countries: []string{"Q45"}, populated: true, sitelinks: 50, images: []string{"City.jpg"}}
	w.files["File:City.jpg"] = goodFile()

	photo, err := w.api(t).Photo(context.Background(), application.DestinationIdentity{City: "Porto", Country: "PT", Lat: 41.1494, Lon: -8.6108})
	require.NoError(t, err)
	assert.Nil(t, photo)
	assert.Zero(t, w.count("wbgetentities"), "a hit with another name is never loaded")
}

func TestDestinationPhoto_OnlyPlacesPeopleLiveInQualify(t *testing.T) {
	w := newPhotoWorld()
	w.search["Firenze"] = []stubHit{{"Q707731", "Florence Airport", "Firenze"}}
	w.entities["Q707731"] = stubEntity{lat: 43.80, lon: 11.20, countries: []string{"Q45"}, types: []string{"Q1248784"}, sitelinks: 20, images: []string{"Airport.jpg"}}
	w.files["File:Airport.jpg"] = goodFile()

	photo, err := w.api(t).Photo(context.Background(), application.DestinationIdentity{City: "Firenze", Country: "PT", Lat: 43.7699, Lon: 11.2556})
	require.NoError(t, err)
	assert.Nil(t, photo)

	// A small place with no population statement still counts when it is typed as a village.
	e := w.entities["Q707731"]
	e.types = []string{"Q532"}
	w.entities["Q707731"] = e
	photo, err = w.api(t).Photo(context.Background(), application.DestinationIdentity{City: "Firenze", Country: "PT", Lat: 43.7699, Lon: 11.2556})
	require.NoError(t, err)
	assert.NotNil(t, photo)
}

func TestDestinationPhoto_ChoosesTheEstablishedEntityAndRefusesAmbiguity(t *testing.T) {
	build := func(secondSitelinks int, secondLat float64) *photoWorld {
		w := newPhotoWorld()
		w.search["Alpha"] = []stubHit{{"Q1", "Alpha", "Alpha"}, {"Q2", "Alpha", "Alpha"}}
		w.entities["Q1"] = stubEntity{lat: 10, lon: 20, countries: []string{"Q45"}, populated: true, sitelinks: 100, images: []string{"One.jpg"}}
		w.entities["Q2"] = stubEntity{lat: secondLat, lon: 20, countries: []string{"Q45"}, populated: true, sitelinks: secondSitelinks, images: []string{"Two.jpg"}}
		w.files["File:One.jpg"] = goodFile()
		w.files["File:Two.jpg"] = goodFile()

		return w
	}
	id := application.DestinationIdentity{City: "Alpha", Country: "PT", Lat: 10.05, Lon: 20}

	// Two different places, both plausible and about 11 km apart: no choice is made.
	photo, err := build(90, 10.10+0.05).api(t).Photo(context.Background(), id)
	require.NoError(t, err)
	assert.Nil(t, photo, "comparable entities far apart are ambiguous")

	// A clearly better established entity wins even with a namesake nearby.
	photo, err = build(20, 10.15).api(t).Photo(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, photo)
	assert.Equal(t, "Q1", photo.SourceID)

	// Two records of one place a few kilometres apart, such as a municipality and its seat, are not ambiguous.
	photo, err = build(90, 10.06).api(t).Photo(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, photo)
	assert.Equal(t, "Q1", photo.SourceID)
}

func TestDestinationPhoto_FindsAnEntityTheNameSearchMissed(t *testing.T) {
	w := newPhotoWorld()
	w.countries["Q717"] = "VE"
	w.search["Valencia"] = []stubHit{{"Q8818", "Valencia", "Valencia"}}
	w.entities["Q8818"] = stubEntity{lat: 39.47, lon: -0.37, countries: []string{"Q29"}, populated: true, sitelinks: 200, images: []string{"Spain.jpg"}}
	w.countries["Q29"] = "ES"
	w.nearby = []stubHit{{"Q1", "Some Museum", ""}, {"Q2", "Valencia, Venezuela", ""}, {"Q3", "Valencia (disambiguation)", ""}}
	w.entities["Q2"] = stubEntity{lat: 10.18, lon: -68.0, countries: []string{"Q717"}, populated: true, sitelinks: 40, images: []string{"Venezuela.jpg"}}
	w.entities["Q3"] = stubEntity{noCoords: true}
	w.files["File:Spain.jpg"] = goodFile()
	w.files["File:Venezuela.jpg"] = goodFile()
	w.nearSearch = []stubHit{{id: "Q4"}}
	w.entities["Q4"] = stubEntity{lat: 10.18, lon: -68.01, countries: []string{"Q717"}, populated: true, sitelinks: 90, label: "Valencia", images: []string{"Populated.jpg"}}
	w.files["File:Populated.jpg"] = goodFile()

	photo, err := w.api(t).Photo(context.Background(), application.DestinationIdentity{City: "Valencia", Country: "VE", Lat: 10.1800, Lon: -68.0000})
	require.NoError(t, err)
	require.NotNil(t, photo)
	assert.Equal(t, "Q2", photo.SourceID, "a nearby article's match wins over the populated-place fallback")
	assert.Equal(t, 1, w.count("geosearch"))
}

func TestDestinationPhoto_PrefersTheNameSearchOverTheFallbacks(t *testing.T) {
	w := parisWorld()
	w.nearby = []stubHit{{"Q7", "Paris", ""}}
	w.nearSearch = []stubHit{{id: "Q8"}}
	w.entities["Q7"] = stubEntity{lat: 48.86, lon: 2.33, countries: []string{"Q142"}, populated: true, sitelinks: 900, images: []string{"Nearby.jpg"}}
	w.entities["Q8"] = stubEntity{lat: 48.86, lon: 2.34, countries: []string{"Q142"}, populated: true, sitelinks: 900, label: "Paris", images: []string{"Populated.jpg"}}
	w.files["File:Nearby.jpg"] = goodFile()
	w.files["File:Populated.jpg"] = goodFile()

	photo, err := w.api(t).Photo(context.Background(), parisFR)
	require.NoError(t, err)
	require.NotNil(t, photo)
	assert.Equal(t, "Q90", photo.SourceID, "the name search's match wins over better-linked fallback places")
}

func TestDestinationPhoto_PopulatedNearPrefersMatchingAliasOverMoreSitelinks(t *testing.T) {
	w := newPhotoWorld()
	w.countries["Q30"] = "US"
	w.search["New York"] = []stubHit{{"Q1384", "New York", "New York"}}
	w.entities["Q1384"] = stubEntity{lat: 42.9, lon: -75, countries: []string{"Q30"}, populated: true, sitelinks: 500}
	w.nearSearch = []stubHit{{id: "Q18419"}, {id: "Q60"}}
	w.entities["Q18419"] = stubEntity{lat: 40.65, lon: -73.95, countries: []string{"Q30"}, populated: true, sitelinks: 500, label: "Brooklyn", images: []string{"Brooklyn.jpg"}}
	w.entities["Q60"] = stubEntity{lat: 40.7127, lon: -74.0059, countries: []string{"Q30"}, populated: true, sitelinks: 200, label: "New York City", aliases: []string{"New York"}, images: []string{"New York.jpg"}}
	w.files["File:Brooklyn.jpg"] = goodFile()
	w.files["File:New York.jpg"] = goodFile()

	photo, err := w.api(t).Photo(context.Background(), application.DestinationIdentity{City: "New York", Country: "US", Lat: 40.713, Lon: -74.006})
	require.NoError(t, err)
	require.NotNil(t, photo)
	assert.Equal(t, "Q60", photo.SourceID)
	for _, req := range w.requests {
		if req.URL.Query().Get("list") != "search" {
			continue
		}
		assert.Equal(t, "https://www.wikidata.org/w/api.php", req.URL.Scheme+"://"+req.URL.Host+req.URL.Path)
		assert.Equal(t, "haswbstatement:P1082 nearcoord:25km,40.713000,-74.006000", req.URL.Query().Get("srsearch"))
		assert.Equal(t, "5", req.URL.Query().Get("srlimit"))
	}
}

// newYorkWorld is New York as Wikidata answers it: the name search finds only the state, no
// nearby article is titled "New York", and the populated-place search finds the city.
func newYorkWorld() *photoWorld {
	w := newPhotoWorld()
	w.search["New York"] = []stubHit{{"Q1384", "New York", "New York"}}
	w.entities["Q1384"] = stubEntity{lat: 42.9, lon: -75, countries: []string{"Q30"}, populated: true, sitelinks: 500}
	w.nearby = []stubHit{{"Q11299", "Manhattan", ""}}
	w.nearSearch = []stubHit{{id: "Q60"}}
	w.entities["Q60"] = stubEntity{lat: 40.7127, lon: -74.0059, countries: []string{"Q30"}, populated: true, sitelinks: 200, label: "New York City", aliases: []string{"New York"}, images: []string{"New York.jpg"}}
	w.files["File:New York.jpg"] = goodFile()

	return w
}

var newYorkUS = application.DestinationIdentity{City: "New York", Country: "US", Lat: 40.713, Lon: -74.006}

func TestDestinationPhoto_RunsTheThreeSearchesAtOnce(t *testing.T) {
	w := newYorkWorld()
	var mu sync.Mutex
	arrived := 0
	all := make(chan struct{})
	w.override = func(req *http.Request) *stubResponse {
		q := req.URL.Query()
		if q.Get("action") != "wbsearchentities" && q.Get("generator") != "geosearch" && q.Get("list") != "search" {
			return nil
		}
		mu.Lock()
		arrived++
		if arrived == 3 {
			close(all)
		}
		mu.Unlock()
		// Each search waits for the other two, so searches made one after another never finish.
		select {
		case <-all:
			return nil
		case <-time.After(time.Second):
			return &stubResponse{status: http.StatusServiceUnavailable, body: "the other searches never started"}
		}
	}

	photo, err := w.api(t).Photo(context.Background(), newYorkUS)
	require.NoError(t, err)
	require.NotNil(t, photo)
	assert.Equal(t, "Q60", photo.SourceID)
}

func TestDestinationPhoto_LoadsEveryCandidateInOneEntityRequest(t *testing.T) {
	w := newYorkWorld()

	photo, err := w.api(t).Photo(context.Background(), newYorkUS)
	require.NoError(t, err)
	require.NotNil(t, photo)
	assert.Equal(t, "Q60", photo.SourceID)
	assert.Equal(t, 1, w.count("wbgetentities"), "the state from the name search and the city from the fallback load together")
}

func TestDestinationPhoto_IgnoresFallbackFailuresWhenTheNameSearchMatched(t *testing.T) {
	w := parisWorld()
	w.override = func(req *http.Request) *stubResponse {
		q := req.URL.Query()
		if q.Get("generator") == "geosearch" || q.Get("list") == "search" {
			return &stubResponse{status: http.StatusServiceUnavailable, body: "unavailable"}
		}
		return nil
	}

	photo, err := w.api(t).Photo(context.Background(), parisFR)
	require.NoError(t, err)
	require.NotNil(t, photo)
	assert.Equal(t, "Q90", photo.SourceID)
}

func TestDestinationPhoto_PopulatedNearChoosesTorontoWithoutANameMatch(t *testing.T) {
	w := newPhotoWorld()
	w.countries["Q16"] = "CA"
	w.search["Old Toronto"] = []stubHit{{"Q919169", "Old Toronto", "Old Toronto"}}
	w.entities["Q919169"] = stubEntity{noCoords: true}
	w.nearSearch = []stubHit{{id: "Q50816"}, {id: "Q172"}}
	w.entities["Q50816"] = stubEntity{lat: 43.59, lon: -79.65, countries: []string{"Q16"}, populated: true, sitelinks: 50, label: "Mississauga", images: []string{"Mississauga.jpg"}}
	w.entities["Q172"] = stubEntity{lat: 43.653, lon: -79.383, countries: []string{"Q16"}, populated: true, sitelinks: 250, label: "Toronto", images: []string{"Toronto.jpg"}}
	w.files["File:Toronto.jpg"] = goodFile()

	photo, err := w.api(t).Photo(context.Background(), application.DestinationIdentity{City: "Old Toronto", Country: "CA", Lat: 43.653, Lon: -79.384})
	require.NoError(t, err)
	require.NotNil(t, photo)
	assert.Equal(t, "Q172", photo.SourceID)
}

func TestDestinationPhoto_PopulatedNearRefusesAmbiguousCities(t *testing.T) {
	w := newPhotoWorld()
	w.nearSearch = []stubHit{{id: "Q1"}, {id: "Q2"}}
	w.entities["Q1"] = stubEntity{lat: 10, lon: 20, countries: []string{"Q45"}, populated: true, sitelinks: 100, label: "Alpha", images: []string{"One.jpg"}}
	w.entities["Q2"] = stubEntity{lat: 10.15, lon: 20, countries: []string{"Q45"}, populated: true, sitelinks: 90, label: "Beta", images: []string{"Two.jpg"}}
	w.files["File:One.jpg"] = goodFile()
	w.files["File:Two.jpg"] = goodFile()

	photo, err := w.api(t).Photo(context.Background(), application.DestinationIdentity{City: "Missing", Country: "PT", Lat: 10.075, Lon: 20})
	require.NoError(t, err)
	assert.Nil(t, photo)
}

func TestDestinationPhoto_PopulatedNearRefusesForeignCity(t *testing.T) {
	w := newPhotoWorld()
	w.nearSearch = []stubHit{{id: "Q2"}}
	w.entities["Q2"] = stubEntity{lat: 10, lon: 20, countries: []string{"Q30"}, populated: true, sitelinks: 90, label: "Beta", images: []string{"Two.jpg"}}
	w.files["File:Two.jpg"] = goodFile()

	photo, err := w.api(t).Photo(context.Background(), application.DestinationIdentity{City: "Missing", Country: "PT", Lat: 10, Lon: 20})
	require.NoError(t, err)
	assert.Nil(t, photo)
}

func TestDestinationPhoto_PopulatedNearProviderErrorIsNotNoPhoto(t *testing.T) {
	w := newPhotoWorld()
	w.override = func(req *http.Request) *stubResponse {
		if req.URL.Query().Get("list") == "search" {
			return &stubResponse{status: 503, body: "unavailable"}
		}
		return nil
	}
	photo, err := w.api(t).Photo(context.Background(), application.DestinationIdentity{City: "Missing", Country: "PT", Lat: 10, Lon: 20})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "503")
	assert.Nil(t, photo)
}

func TestDestinationPhoto_RejectsUnsuitableImagesAndTriesTheNext(t *testing.T) {
	for name, edit := range map[string]func(*stubFile){
		"svg":             func(f *stubFile) { f.mime = "image/svg+xml" },
		"tiff":            func(f *stubFile) { f.mime = "image/tiff" },
		"portrait":        func(f *stubFile) { f.width, f.height = 2000, 3000 },
		"too small":       func(f *stubFile) { f.width, f.height = 600, 400 },
		"missing author":  func(f *stubFile) { f.artist = "  <br> " },
		"missing license": func(f *stubFile) { f.license = "" },
		"non-free":        func(f *stubFile) { f.nonFree = "true" },
		"restricted":      func(f *stubFile) { f.restrictions = "personality" },
		"wrong image host": func(f *stubFile) {
			f.thumb = "https://evil.example/x.jpg"
		},
		"http image": func(f *stubFile) {
			f.thumb = "http://upload.wikimedia.org/x.jpg"
		},
		"script image": func(f *stubFile) {
			f.thumb = "javascript:alert(1)"
		},
		"wrong source page host": func(f *stubFile) {
			f.description = "https://evil.example/File:X.jpg"
		},
		"missing":    func(f *stubFile) { f.missing = true },
		"unfree url": func(f *stubFile) { f.thumb = "https://user:pw@upload.wikimedia.org/x.jpg" },
	} {
		t.Run(name, func(t *testing.T) {
			w := parisWorld()
			e := w.entities["Q90"]
			e.images = []string{"Bad.jpg", "Good.jpg"}
			w.entities["Q90"] = e
			bad := goodFile()
			edit(&bad)
			w.files["File:Bad.jpg"] = bad
			w.files["File:Good.jpg"] = goodFile()

			photo, err := w.api(t).Photo(context.Background(), parisFR)
			require.NoError(t, err)
			require.NotNil(t, photo)
			assert.Contains(t, photo.URL, "Good.jpg", "the unsuitable file must not be shown")
		})
	}
}

func TestDestinationPhoto_RejectsSymbolsAndMapsByFileName(t *testing.T) {
	for _, name := range []string{"Flag of Paris.jpg", "Paris logo.png", "Map of Paris.jpg", "Blason Paris.png", "Coat of arms of Paris.jpg", "Paris location map.jpg", "Paris_seal.png", "Paris montage.jpg"} {
		t.Run(name, func(t *testing.T) {
			w := parisWorld()
			e := w.entities["Q90"]
			e.images = []string{name}
			e.article = ""
			w.entities["Q90"] = e
			w.files["File:"+name] = goodFile()

			photo, err := w.api(t).Photo(context.Background(), parisFR)
			require.NoError(t, err)
			assert.Nil(t, photo)
		})
	}

	// Words merely containing those letters are fine.
	w := parisWorld()
	e := w.entities["Q90"]
	e.images = []string{"Mapleton street.jpg"}
	w.entities["Q90"] = e
	w.files["File:Mapleton street.jpg"] = goodFile()
	photo, err := w.api(t).Photo(context.Background(), parisFR)
	require.NoError(t, err)
	assert.NotNil(t, photo)
}

func TestDestinationPhoto_PrefersLandscapeAndKeepsOrderOtherwise(t *testing.T) {
	w := parisWorld()
	e := w.entities["Q90"]
	e.images = []string{"Square.jpg", "Wide.jpg"}
	w.entities["Q90"] = e
	square := goodFile()
	square.width, square.height = 3000, 2800
	w.files["File:Square.jpg"] = square
	w.files["File:Wide.jpg"] = goodFile()

	photo, err := w.api(t).Photo(context.Background(), parisFR)
	require.NoError(t, err)
	require.NotNil(t, photo)
	assert.Contains(t, photo.URL, "Wide.jpg", "landscape is preferred for the wide hero")

	delete(w.files, "File:Wide.jpg")
	photo, err = w.api(t).Photo(context.Background(), parisFR)
	require.NoError(t, err)
	require.NotNil(t, photo)
	assert.Contains(t, photo.URL, "Square.jpg", "a near-square photograph is still better than none")
}

func TestDestinationPhoto_FallsBackToTheArticlePageImage(t *testing.T) {
	w := parisWorld()
	e := w.entities["Q90"]
	e.images = nil
	w.entities["Q90"] = e
	w.pageImage["Paris"] = "Page image.jpg"
	w.files["File:Page image.jpg"] = goodFile()

	photo, err := w.api(t).Photo(context.Background(), parisFR)
	require.NoError(t, err)
	require.NotNil(t, photo)
	assert.Contains(t, photo.URL, "Page_image.jpg")
	assert.Equal(t, 1, w.count("pageimages"))
}

func TestDestinationPhoto_NoImageIsAnEmptyResultNotAnError(t *testing.T) {
	w := parisWorld()
	e := w.entities["Q90"]
	e.images, e.article = nil, ""
	w.entities["Q90"] = e

	photo, err := w.api(t).Photo(context.Background(), parisFR)
	require.NoError(t, err)
	assert.Nil(t, photo)

	// With an article that has no free page image either, the result is the same.
	e.article = "Paris"
	w.entities["Q90"] = e
	photo, err = w.api(t).Photo(context.Background(), parisFR)
	require.NoError(t, err)
	assert.Nil(t, photo)
}

func TestDestinationPhoto_ExaminesAtMostFiveCitiesAndThreeImages(t *testing.T) {
	w := parisWorld()
	e := w.entities["Q90"]
	e.images = []string{"A.jpg", "B.jpg", "C.jpg", "D.jpg", "E.jpg"}
	w.entities["Q90"] = e
	w.pageImage["Paris"] = "F.jpg"

	_, err := w.api(t).Photo(context.Background(), parisFR)
	require.NoError(t, err)

	var searched, titles int

	for _, req := range w.requests {
		q := req.URL.Query()
		if q.Get("action") == "wbsearchentities" {
			searched++
			assert.Equal(t, "5", q.Get("limit"))
		}

		if q.Get("prop") == "imageinfo" {
			titles += len(strings.Split(q.Get("titles"), "|"))
		}
	}

	assert.Equal(t, 1, searched)
	assert.LessOrEqual(t, titles, 3, "no more than three image candidates are resolved")
	assert.Equal(t, 1, w.count("pageimages"), "the article image is the third candidate")
}

func TestDestinationPhoto_FollowsCommonsRedirects(t *testing.T) {
	w := parisWorld()
	w.redirects["File:Tour Eiffel.jpg"] = "File:Tour Eiffel (renamed).jpg"
	w.files["File:Tour Eiffel (renamed).jpg"] = goodFile()

	photo, err := w.api(t).Photo(context.Background(), parisFR)
	require.NoError(t, err)
	require.NotNil(t, photo)
	assert.Contains(t, photo.CreditURL, "renamed")
}

func TestDestinationPhoto_DropsAnUnsafeLicenseLinkButKeepsTheLicenseName(t *testing.T) {
	w := parisWorld()
	f := goodFile()
	f.licenseURL = "javascript:alert(1)"
	w.files["File:Tour Eiffel.jpg"] = f

	photo, err := w.api(t).Photo(context.Background(), parisFR)
	require.NoError(t, err)
	require.NotNil(t, photo)
	assert.Equal(t, "CC BY-SA 4.0", photo.License)
	assert.Empty(t, photo.LicenseURL)
}

func TestDestinationPhoto_UsesTheOriginalWhenNoThumbnailIsOffered(t *testing.T) {
	w := parisWorld()
	w.override = func(req *http.Request) *stubResponse {
		if req.URL.Query().Get("prop") != "imageinfo" {
			return nil
		}

		return &stubResponse{status: 200, body: `{"query":{"pages":[{"title":"File:Tour Eiffel.jpg","imageinfo":[{"url":"https://upload.wikimedia.org/wikipedia/commons/a/ab/Tour_Eiffel.jpg","width":1000,"height":700,"descriptionurl":"https://commons.wikimedia.org/wiki/File:Tour_Eiffel.jpg","mime":"image/jpeg","extmetadata":{"Artist":{"value":"Ada"},"LicenseShortName":{"value":"CC0"}}}]}]}}`}
	}

	photo, err := w.api(t).Photo(context.Background(), parisFR)
	require.NoError(t, err)
	require.NotNil(t, photo)
	assert.Equal(t, "https://upload.wikimedia.org/wikipedia/commons/a/ab/Tour_Eiffel.jpg", photo.URL)
	assert.Equal(t, 1000, photo.Width)
	assert.Equal(t, 700, photo.Height)
	assert.Equal(t, "CC0", photo.License)
	assert.Empty(t, photo.LicenseURL)
}

func TestDestinationPhoto_ClassifiesProviderFailuresAsErrors(t *testing.T) {
	for name, stub := range map[string]stubResponse{
		"rate limited":      {status: http.StatusTooManyRequests, body: "slow down"},
		"server error":      {status: http.StatusInternalServerError, body: "oops"},
		"malformed JSON":    {status: http.StatusOK, body: `{"search":`},
		"error with a 200":  {status: http.StatusOK, body: `{"error":{"code":"ratelimited","info":"too many"}}`},
		"HTML instead JSON": {status: http.StatusOK, body: `<html>maintenance</html>`},
	} {
		for _, action := range []string{"wbsearchentities", "wbgetentities", "wbgetclaims", "imageinfo"} {
			t.Run(name+"/"+action, func(t *testing.T) {
				w := parisWorld()
				w.override = func(req *http.Request) *stubResponse {
					q := req.URL.Query()
					if q.Get("action") == action || action == "imageinfo" && q.Get("prop") == "imageinfo" {
						return &stub
					}

					return nil
				}

				photo, err := w.api(t).Photo(context.Background(), parisFR)
				require.Error(t, err, "a provider failure is never an empty result")
				assert.Nil(t, photo)
			})
		}
	}
}

func TestDestinationPhoto_RateLimitedErrorNamesTheStatus(t *testing.T) {
	w := parisWorld()
	w.override = func(*http.Request) *stubResponse { return &stubResponse{status: 429, body: "slow down"} }

	_, err := w.api(t).Photo(context.Background(), parisFR)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "429")
}

func TestDestinationPhoto_StopsAtTheLookupDeadline(t *testing.T) {
	api := NewDestinationPhotoAPI(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()

		return nil, req.Context().Err()
	})})
	api.timeout = 20 * time.Millisecond

	start := time.Now()
	photo, err := api.Photo(context.Background(), parisFR)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Nil(t, photo)
	assert.Less(t, time.Since(start), time.Second)
}

func TestDestinationPhoto_HonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	photo, err := parisWorld().api(t).Photo(ctx, parisFR)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, photo)
}

func TestDestinationPhoto_RejectsAnUnusableIdentityWithoutRequests(t *testing.T) {
	w := parisWorld()
	api := w.api(t)

	for _, id := range []application.DestinationIdentity{
		{City: "", Country: "FR", Lat: 48, Lon: 2},
		{City: "Paris", Country: "France", Lat: 48, Lon: 2},
		{City: "Paris", Country: "FR", Lat: 0, Lon: 0},
		{City: "Paris", Country: "FR", Lat: 95, Lon: 2},
	} {
		_, err := api.Photo(context.Background(), id)
		require.Error(t, err)
	}

	assert.Empty(t, w.requests)
}

func TestDestinationPhoto_SendsTheUserAgentAndNoKey(t *testing.T) {
	w := parisWorld()
	_, err := w.api(t).Photo(context.Background(), parisFR)
	require.NoError(t, err)
	require.NotEmpty(t, w.requests)

	for _, req := range w.requests {
		assert.Equal(t, UserAgent, req.Header.Get("User-Agent"))
		assert.NotContains(t, strings.ToLower(req.URL.RawQuery), "key")
		assert.Contains(t, []string{"www.wikidata.org", "en.wikipedia.org", "commons.wikimedia.org"}, req.URL.Host)
	}
}

func TestDestinationPhoto_RemembersCountryCodes(t *testing.T) {
	w := parisWorld()
	api := w.api(t)

	_, err := api.Photo(context.Background(), parisFR)
	require.NoError(t, err)

	first := w.count("wbgetclaims")
	require.Positive(t, first)

	_, err = api.Photo(context.Background(), parisFR)
	require.NoError(t, err)
	assert.Equal(t, first, w.count("wbgetclaims"), "an ISO code is fetched once")
}

func TestDestinationPhoto_DeprecatedCountryCodeOnlyWhenNoCurrentCode(t *testing.T) {
	for _, tc := range []struct {
		name   string
		claims string
		want   string
	}{
		{"deprecated only", `{"claims":{"P297":[{"rank":"deprecated","mainsnak":{"datavalue":{"value":"NL"}}}]}}`, "NL"},
		{"normal takes precedence", `{"claims":{"P297":[{"rank":"deprecated","mainsnak":{"datavalue":{"value":"XX"}}},{"rank":"normal","mainsnak":{"datavalue":{"value":"NL"}}}]}}`, "NL"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newPhotoWorld()
			w.override = func(req *http.Request) *stubResponse {
				if req.URL.Query().Get("action") == "wbgetclaims" {
					return &stubResponse{status: http.StatusOK, body: tc.claims}
				}
				return nil
			}
			code, err := w.api(t).countryCode(context.Background(), "Q55")
			require.NoError(t, err)
			assert.Equal(t, tc.want, code)
		})
	}
}

func TestDestinationPhoto_AmsterdamAcceptsDeprecatedOnlyNLCode(t *testing.T) {
	w := newPhotoWorld()
	w.search["Amsterdam"] = []stubHit{{"Q727", "Amsterdam", "Amsterdam"}}
	w.entities["Q727"] = stubEntity{lat: 52.3676, lon: 4.9041, countries: []string{"Q55"}, types: []string{"Q515"}, images: []string{"Amsterdam.jpg"}}
	w.files["File:Amsterdam.jpg"] = goodFile()
	w.override = func(req *http.Request) *stubResponse {
		if req.URL.Query().Get("action") == "wbgetclaims" && req.URL.Query().Get("entity") == "Q55" {
			return &stubResponse{status: http.StatusOK, body: `{"claims":{"P297":[{"rank":"deprecated","mainsnak":{"datavalue":{"value":"NL"}}}]}}`}
		}
		return nil
	}

	photo, err := w.api(t).Photo(context.Background(), application.DestinationIdentity{City: "Amsterdam", Country: "NL", Lat: 52.3676, Lon: 4.9041})
	require.NoError(t, err)
	require.NotNil(t, photo)
	assert.Equal(t, "Q727", photo.SourceID)
}

func TestDestinationPhoto_ImageRequestAsksForABoundedThumbnail(t *testing.T) {
	w := parisWorld()
	_, err := w.api(t).Photo(context.Background(), parisFR)
	require.NoError(t, err)

	for _, req := range w.requests {
		q := req.URL.Query()
		if q.Get("prop") == "imageinfo" {
			assert.Equal(t, "1280", q.Get("iiurlwidth"))
			assert.Contains(t, q.Get("iiprop"), "extmetadata")
			assert.Equal(t, "commons.wikimedia.org", req.URL.Host)
		}
	}
}

func TestHTMLText(t *testing.T) {
	for name, tc := range map[string]struct {
		in   string
		want string
	}{
		"plain":    {"Ada Lovelace", "Ada Lovelace"},
		"links":    {`<a href="//x">Ada</a> and <a href="//y">Bob</a>`, "Ada and Bob"},
		"list":     {"<ul><li>One</li>\n<li>Two</li></ul>", "One Two"},
		"script":   {"Ada<script>alert('x')</script>", "Ada"},
		"style":    {"<style>p{}</style>Ada", "Ada"},
		"entities": {"Ada &amp; Bob &lt;b&gt;", "Ada & Bob <b>"},
		"empty":    {"  <br>  ", ""},
		"long":     {strings.Repeat("a", 200), strings.Repeat("a", 120) + "…"},
		"unclosed": {"<b>Ada", "Ada"},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, htmlText(tc.in, 120))
		})
	}
}

func TestSafeURL(t *testing.T) {
	for raw, ok := range map[string]bool{
		"https://upload.wikimedia.org/a.jpg": true,
		"http://upload.wikimedia.org/a.jpg":  false,
		"javascript:alert(1)":                false,
		"data:image/png;base64,AAAA":         false,
		"//upload.wikimedia.org/a.jpg":       false,
		"https://user@upload.wikimedia.org/": false,
		"https://evil.example/a.jpg":         false,
		"":                                   false,
	} {
		_, got := safeURL(raw, "upload.wikimedia.org")
		assert.Equal(t, ok, got, raw)
	}

	_, ok := safeURL("https://anything.example/x")
	assert.True(t, ok, "without an allow-list any https host passes")
}

func TestNewDestinationPhotoAPIUsesAClientWithATimeout(t *testing.T) {
	api := NewDestinationPhotoAPI(nil)
	require.NotNil(t, api.wiki.client)
	assert.Equal(t, wikimediaTimeout, api.wiki.client.Timeout)
	assert.Equal(t, 4*time.Second, destinationPhotoTimeout)
}

// liveDestinations are the cities the live check resolves. LIVE_RUNS repeats each with a fresh
// client, so every run is a cold lookup (only Wikimedia's own caches are warm).
var liveDestinations = []application.DestinationIdentity{
	parisFR,
	parisTX,
	{City: "Porto", Country: "PT", Lat: 41.1494, Lon: -8.6108},
	{City: "Tokyo", Country: "JP", Lat: 35.6895, Lon: 139.6917},
	{City: "Tavira", Country: "PT", Lat: 37.1264, Lon: -7.6506},
	{City: "Obidos", Country: "PT", Lat: 39.3606, Lon: -9.1571},
	{City: "Lisboa", Country: "PT", Lat: 38.7077, Lon: -9.1365},
	{City: "Valencia", Country: "VE", Lat: 10.1620, Lon: -68.0077},
	{City: "Springfield", Country: "US", Lat: 39.7817, Lon: -89.6501},
	{City: "Kandersteg", Country: "CH", Lat: 46.4953, Lon: 7.6742},
	// OpenWeather identities from destination-photo-fallback-plan.md.
	{City: "New York", Country: "US", Lat: 40.713, Lon: -74.006},
	{City: "Los Angeles", Country: "US", Lat: 34.054, Lon: -118.243},
	{City: "San Francisco", Country: "US", Lat: 37.779, Lon: -122.420},
	{City: "Washington", Country: "US", Lat: 38.895, Lon: -77.037},
	{City: "Chicago", Country: "US", Lat: 41.876, Lon: -87.624},
	{City: "Miami", Country: "US", Lat: 25.774, Lon: -80.194},
	{City: "Las Vegas", Country: "US", Lat: 36.167, Lon: -115.148},
	{City: "Mexico City", Country: "MX", Lat: 19.433, Lon: -99.133},
	{City: "Rio de Janeiro", Country: "BR", Lat: -22.911, Lon: -43.209},
	{City: "Buenos Aires", Country: "AR", Lat: -34.608, Lon: -58.437},
	{City: "Old Toronto", Country: "CA", Lat: 43.653, Lon: -79.384},
	{City: "Montreal", Country: "CA", Lat: 45.503, Lon: -73.570},
	{City: "London", Country: "GB", Lat: 51.507, Lon: -0.128},
	{City: "Paris", Country: "FR", Lat: 48.859, Lon: 2.320},
	{City: "Rome", Country: "IT", Lat: 41.893, Lon: 12.483},
	{City: "Venice", Country: "IT", Lat: 45.437, Lon: 12.335},
	{City: "Florence", Country: "IT", Lat: 43.770, Lon: 11.256},
	{City: "Barcelona", Country: "ES", Lat: 41.383, Lon: 2.177},
	{City: "Madrid", Country: "ES", Lat: 40.417, Lon: -3.704},
	{City: "Lisbon", Country: "PT", Lat: 38.708, Lon: -9.137},
	{City: "Porto", Country: "PT", Lat: 41.149, Lon: -8.611},
	{City: "Funchal", Country: "PT", Lat: 32.650, Lon: -16.909},
	{City: "Amsterdam", Country: "NL", Lat: 52.373, Lon: 4.894},
	{City: "Berlin", Country: "DE", Lat: 52.517, Lon: 13.389},
	{City: "Munich", Country: "DE", Lat: 48.137, Lon: 11.575},
	{City: "Frankfurt", Country: "DE", Lat: 50.111, Lon: 8.682},
	{City: "Vienna", Country: "AT", Lat: 48.208, Lon: 16.373},
	{City: "Prague", Country: "CZ", Lat: 50.087, Lon: 14.421},
	{City: "Budapest", Country: "HU", Lat: 47.498, Lon: 19.040},
	{City: "Athens", Country: "GR", Lat: 37.984, Lon: 23.728},
	{City: "Istanbul", Country: "TR", Lat: 41.009, Lon: 28.966},
	{City: "Dubrovnik", Country: "HR", Lat: 42.649, Lon: 18.095},
	{City: "Brussels", Country: "BE", Lat: 50.847, Lon: 4.352},
	{City: "Copenhagen", Country: "DK", Lat: 55.687, Lon: 12.570},
	{City: "Saint Petersburg", Country: "RU", Lat: 59.939, Lon: 30.316},
	{City: "City of Edinburgh", Country: "GB", Lat: 55.953, Lon: -3.188},
	{City: "Tokyo", Country: "JP", Lat: 35.677, Lon: 139.764},
	{City: "Kyoto", Country: "JP", Lat: 35.021, Lon: 135.756},
	{City: "Seoul", Country: "KR", Lat: 37.567, Lon: 126.978},
	{City: "Beijing", Country: "CN", Lat: 39.906, Lon: 116.391},
	{City: "Singapore", Country: "SG", Lat: 1.290, Lon: 103.852},
	{City: "Bangkok", Country: "TH", Lat: 13.752, Lon: 100.494},
	{City: "Ho Chi Minh City", Country: "VN", Lat: 10.776, Lon: 106.702},
	{City: "Hanoi", Country: "VN", Lat: 21.029, Lon: 105.854},
	{City: "Pulau Bali", Country: "ID", Lat: -8.456, Lon: 115.270},
	{City: "Delhi", Country: "IN", Lat: 28.652, Lon: 77.222},
	{City: "Mumbai", Country: "IN", Lat: 19.079, Lon: 72.878},
	{City: "Dubai", Country: "AE", Lat: 25.265, Lon: 55.292},
	{City: "Marrakesh", Country: "MA", Lat: 31.626, Lon: -7.989},
	{City: "Cairo", Country: "EG", Lat: 30.044, Lon: 31.236},
	{City: "Cape Town", Country: "ZA", Lat: -33.929, Lon: 18.417},
	{City: "Sydney", Country: "AU", Lat: -33.870, Lon: 151.208},
	{City: "Melbourne", Country: "AU", Lat: -37.814, Lon: 144.963},
	{City: "Auckland", Country: "NZ", Lat: -36.852, Lon: 174.763},
	{City: "Reykjavik", Country: "IS", Lat: 64.146, Lon: -21.942},
	{City: "Kuala Lumpur", Country: "MY", Lat: 3.152, Lon: 101.694},
	{City: "Vatican City", Country: "VA", Lat: 41.903, Lon: 12.453},
	{City: "Monaco", Country: "MC", Lat: 43.731, Lon: 7.420},
	{City: "Quebec", Country: "CA", Lat: 46.814, Lon: -71.208},
}

// The live check is opt-in: it calls the real Wikimedia APIs, reports what each city resolved to
// and how long the cold lookup took, and downloads the selected image.
func TestDestinationPhotoLive(t *testing.T) {
	if os.Getenv("LIVE_API_TESTS") != "1" {
		t.Skip("set LIVE_API_TESTS=1 to call the real Wikimedia APIs")
	}

	runs, _ := strconv.Atoi(os.Getenv("LIVE_RUNS"))
	if runs < 1 {
		runs = 1
	}

	for _, id := range liveDestinations {
		var latencies []time.Duration

		for run := 0; run < runs; run++ {
			start := time.Now()
			photo, err := NewDestinationPhotoAPI(nil).Photo(context.Background(), id)
			elapsed := time.Since(start)
			latencies = append(latencies, elapsed)
			assert.Less(t, elapsed, destinationPhotoTimeout, "%s, %s: lookup exceeded the four-second deadline", id.City, id.Country)

			if err != nil {
				t.Errorf("%s, %s: error after %s: %v", id.City, id.Country, elapsed.Round(time.Millisecond), err)
				continue
			}

			if photo == nil {
				t.Errorf("%s, %s: no photograph after %s", id.City, id.Country, elapsed.Round(time.Millisecond))
				continue
			}

			if run > 0 {
				continue
			}

			t.Logf("%s, %s: entity %s, %dx%d, credit %q, license %q (%s)\n  image %s\n  source %s",
				id.City, id.Country, photo.SourceID, photo.Width, photo.Height, photo.Credit, photo.License, photo.LicenseURL, photo.URL, photo.CreditURL)

			req, err := http.NewRequest(http.MethodGet, photo.URL, nil)
			require.NoError(t, err)
			req.Header.Set("User-Agent", UserAgent)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)

			body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
			_ = resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode, photo.URL)
			assert.Contains(t, resp.Header.Get("Content-Type"), "image/")
			t.Logf("  downloaded %d bytes (%s)", len(body), resp.Header.Get("Content-Type"))
		}

		t.Logf("%s, %s: cold lookup latencies %v", id.City, id.Country, latencies)
	}
}
