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
	lat, lon  float64
	noCoords  bool
	countries []string
	types     []string
	populated bool
	sitelinks int
	images    []string
	article   string
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
	mu        sync.Mutex
	search    map[string][]stubHit
	entities  map[string]stubEntity
	countries map[string]string
	nearby    []stubHit
	pageImage map[string]string // article title -> file name
	files     map[string]stubFile
	redirects map[string]string // file title -> target file title
	requests  []*http.Request
	override  func(*http.Request) *stubResponse
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

		entities[id] = map[string]any{"id": id, "claims": claims, "sitelinks": sitelinks}
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

	photo, err := w.api(t).Photo(context.Background(), application.DestinationIdentity{City: "Valencia", Country: "VE", Lat: 10.1800, Lon: -68.0000})
	require.NoError(t, err)
	require.NotNil(t, photo)
	assert.Equal(t, "Q2", photo.SourceID)
	assert.Equal(t, 1, w.count("geosearch"))
}

func TestDestinationPhoto_SkipsTheNearbyFallbackWhenTheSearchMatched(t *testing.T) {
	w := parisWorld()
	_, err := w.api(t).Photo(context.Background(), parisFR)
	require.NoError(t, err)
	assert.Zero(t, w.count("geosearch"))
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
