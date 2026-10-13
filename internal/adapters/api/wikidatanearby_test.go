package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"weatherservice/internal/writermap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nearbyTransport func(*http.Request) (*http.Response, error)

func (f nearbyTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestWikidataNearby_Places_MadeiraPinsAndGazetteer(t *testing.T) {
	fixture, err := os.ReadFile("testdata/wikidata_madeira.json")
	require.NoError(t, err)
	var seen []string
	client := nearbyRoutes(t, readFixture(t, "wikidata_madeira_page.json"), string(fixture), &seen)

	got, err := NewWikidataNearbyAPI(client).Places(context.Background(), writermap.Area{Name: "Madeira", Lat: 32.75, Lon: -16.95})
	require.NoError(t, err)
	assert.Len(t, seen, 2, "a thin first ring ends the load after one page query and one label query")
	require.Len(t, got.Pins, 2)
	assert.Equal(t, []string{"Q473169", "Q856866"}, []string{got.Pins[0].QID, got.Pins[1].QID})
	assert.ElementsMatch(t, []string{"Pico Ruivo", "Ruivo Peak", "Pico Ruivo de Santana"}, got.Pins[0].Names)
	assert.Equal(t, []string{"Q8502"}, got.Pins[0].Types)
	assert.Equal(t, "highest mountain on Madeira", got.Pins[0].Description)
	assert.Equal(t, "https://en.wikipedia.org/wiki/Pico_Ruivo", got.Pins[0].WikipediaURL)
	assert.Equal(t, "Open daily · 9:00 – 18:00", got.Pins[1].OpeningHours)
	assert.Empty(t, got.Pins[1].Photo)
	assert.Empty(t, got.Pins[1].WikipediaURL)
	require.Len(t, got.Gazetteer, 4)
	assert.Equal(t, "Q639161", got.Gazetteer[2].QID)
	assert.Equal(t, "Q623736", got.Gazetteer[3].QID)
	assert.Equal(t, []string{"Q13217644"}, got.Gazetteer[3].Types)
}

func TestWikidataNearby_Places_EmptyAreaReturnsNothing(t *testing.T) {
	var seen []string
	client := nearbyRoutes(t, `{"results":{"bindings":[]}}`, "", &seen)
	got, err := NewWikidataNearbyAPI(client).Places(context.Background(), writermap.Area{RadiusKM: 80})
	require.NoError(t, err)
	assert.Empty(t, got.Pins)
	assert.Empty(t, got.Gazetteer)
	assert.Len(t, seen, 1, "an empty ring is thin, so nothing farther is asked")
}

func TestWikidataNearby_Places_CreditsCommonsImage(t *testing.T) {
	nearby, err := os.ReadFile("testdata/wikidata_madeira.json")
	require.NoError(t, err)
	commons, err := os.ReadFile("testdata/wikidata_commons.json")
	require.NoError(t, err)
	commonsCalls := 0
	client := &http.Client{Transport: nearbyTransport(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Host, "commons.wikimedia.org") {
			commonsCalls++
			assert.Contains(t, req.URL.Query().Get("titles"), "File:Pico Ruivo.jpg")
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(commons))), Header: make(http.Header)}, nil
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(madeiraBody(t, req, string(nearby)))), Header: make(http.Header)}, nil
	})}

	got, err := NewWikidataNearbyAPI(client).Places(context.Background(), writermap.Area{})
	require.NoError(t, err)
	assert.Equal(t, 1, commonsCalls)
	assert.Equal(t, "Pico Ruivo.jpg", got.Pins[0].Photo, "photo stays the Commons file identifier")
	assert.Equal(t, "Ana · CC BY-SA 4.0", got.Pins[0].PhotoCredit)
	assert.Equal(t, "https://upload.wikimedia.org/wikipedia/commons/thumb/a/aa/Pico_Ruivo.jpg/640px-Pico_Ruivo.jpg", got.Gazetteer[0].PhotoURL)
}

func TestWikidataNearby_Places_RequestsAThumbnail(t *testing.T) {
	nearby, err := os.ReadFile("testdata/wikidata_madeira.json")
	require.NoError(t, err)
	commons, err := os.ReadFile("testdata/wikidata_commons.json")
	require.NoError(t, err)
	var width string
	client := &http.Client{Transport: nearbyTransport(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Host, "commons.wikimedia.org") {
			width = req.URL.Query().Get("iiurlwidth")
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(commons))), Header: make(http.Header)}, nil
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(madeiraBody(t, req, string(nearby)))), Header: make(http.Header)}, nil
	})}

	_, err = NewWikidataNearbyAPI(client).Places(context.Background(), writermap.Area{})
	require.NoError(t, err)
	assert.Equal(t, "640", width)
}

func TestWikidataNearby_Places_FallsBackToTheOriginalWhenNoThumbnail(t *testing.T) {
	nearby, err := os.ReadFile("testdata/wikidata_madeira.json")
	require.NoError(t, err)
	commons := `{"query":{"pages":[{"title":"File:Pico Ruivo.jpg","imageinfo":[{"url":"https://upload.wikimedia.org/wikipedia/commons/a/aa/Pico_Ruivo.jpg","descriptionurl":"https://commons.wikimedia.org/wiki/File:Pico_Ruivo.jpg","extmetadata":{"Artist":{"value":"Ana"},"LicenseShortName":{"value":"CC BY-SA 4.0"}}}]}]}}`
	client := &http.Client{Transport: nearbyTransport(func(req *http.Request) (*http.Response, error) {
		body := madeiraBody(t, req, string(nearby))
		if strings.Contains(req.URL.Host, "commons.wikimedia.org") {
			body = commons
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}

	got, err := NewWikidataNearbyAPI(client).Places(context.Background(), writermap.Area{})
	require.NoError(t, err)
	assert.Equal(t, "https://upload.wikimedia.org/wikipedia/commons/a/aa/Pico_Ruivo.jpg", got.Gazetteer[0].PhotoURL)
}

func TestWikidataNearby_Places_CommonsUnavailableLeavesPhotoEmpty(t *testing.T) {
	fixture, err := os.ReadFile("testdata/wikidata_madeira.json")
	require.NoError(t, err)
	client := &http.Client{Transport: nearbyTransport(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Host, "commons.wikimedia.org") {
			return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: io.NopCloser(strings.NewReader("unavailable")), Header: make(http.Header)}, nil
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(madeiraBody(t, req, string(fixture)))), Header: make(http.Header)}, nil
	})}
	got, err := NewWikidataNearbyAPI(client).Places(context.Background(), writermap.Area{})
	require.NoError(t, err)
	require.Len(t, got.Pins, 2)
	assert.Empty(t, got.Pins[0].Photo)
	assert.Empty(t, got.Gazetteer[0].Photo)
}

func TestWikidataNearby_Places_RetryAfterRespectsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	client := &http.Client{Transport: nearbyTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		cancel()
		return &http.Response{StatusCode: http.StatusTooManyRequests, Body: io.NopCloser(strings.NewReader("busy")), Header: http.Header{"Retry-After": {"30"}}}, nil
	})}
	_, err := NewWikidataNearbyAPI(client).Places(ctx, writermap.Area{})
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, calls)
}

func TestNearbyPlaces_CapsPinsButKeepsGazetteer(t *testing.T) {
	rows := make([]nearbyBinding, 205)
	for i := range rows {
		rows[i].Item.Value = "http://www.wikidata.org/entity/Q" + strconv.Itoa(i+1)
		rows[i].Coord.Value = "Point(-16.95 32.75)"
		rows[i].Type.Value = "http://www.wikidata.org/entity/Q8502"
		rows[i].EnLabel.Value = "Mountain " + strconv.Itoa(i+1)
		rows[i].Sitelinks.Value = strconv.Itoa(i)
	}
	got := nearbyPlaces(rows)
	require.Len(t, got.Pins, 200)
	require.Len(t, got.Gazetteer, 205)
	assert.Equal(t, "Q205", got.Pins[0].QID)
	assert.Equal(t, "Q6", got.Pins[199].QID)
}

func TestNearbyPlaces_DecodesP18FilePath(t *testing.T) {
	rows := []nearbyBinding{{}}
	rows[0].Item.Value = "http://www.wikidata.org/entity/Q1"
	rows[0].Coord.Value = "Point(-16.95 32.75)"
	rows[0].Type.Value = "http://www.wikidata.org/entity/Q8502"
	rows[0].EnLabel.Value = "Mountain"
	rows[0].Image.Value = "http://commons.wikimedia.org/wiki/Special:FilePath/Pico_Ruivo%20view.jpg"
	got := nearbyPlaces(rows)
	require.Len(t, got.Pins, 1)
	assert.Equal(t, "Pico Ruivo view.jpg", got.Pins[0].Photo)
}

// nearbyRoutes answers the page query and the label query from fixtures and records each query text.
func nearbyRoutes(t *testing.T, page, labels string, seen *[]string) *http.Client {
	t.Helper()
	return &http.Client{Transport: nearbyTransport(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, UserAgent, req.Header.Get("User-Agent"))
		if strings.Contains(req.URL.Host, "commons.wikimedia.org") {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"query":{"pages":[]}}`)), Header: make(http.Header)}, nil
		}
		query := req.URL.Query().Get("query")
		*seen = append(*seen, query)
		body := page
		if strings.Contains(query, "VALUES") {
			body = labels
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
}

func readFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)
	return string(data)
}

func TestNearbyPageItems_SendsRadiiOrderLimitOffsetAndFloor(t *testing.T) {
	var seen []string
	client := nearbyRoutes(t, readFixture(t, "wikidata_madeira_page.json"), readFixture(t, "wikidata_madeira.json"), &seen)

	_, fetched, err := NewWikidataNearbyAPI(client).pageItems(context.Background(), 32.75, -16.95, 10, 15, 100)
	require.NoError(t, err)

	assert.Equal(t, 4, fetched)
	require.Len(t, seen, 1, "the page query alone; labels come later")
	page := seen[0]
	assert.Contains(t, page, `wikibase:radius "15"`)
	assert.Contains(t, page, "FILTER(?dist > 10)")
	assert.Contains(t, page, "ORDER BY DESC(?sitelinks) ?item")
	assert.Contains(t, page, "LIMIT 50 OFFSET 100")
	assert.Contains(t, page, "?sitelinks >= 2")
	assert.Contains(t, page, "Point(-16.95 32.75)")
	assert.Contains(t, page, "wdt:P31", "types come with the page so pins can be told apart before labels arrive")
	assert.NotContains(t, page, "rdfs:label", "labels come in a second query")
}

func TestNearbyPageItems_FirstRingHasNoInnerBound(t *testing.T) {
	var seen []string
	client := nearbyRoutes(t, readFixture(t, "wikidata_madeira_page.json"), "", &seen)

	_, _, err := NewWikidataNearbyAPI(client).pageItems(context.Background(), 32.75, -16.95, 0, 5, 0)
	require.NoError(t, err)

	assert.NotContains(t, seen[0], "?dist >")
	assert.Contains(t, seen[0], "OFFSET 0")
}

func TestNearbyPageItems_ReturnsUnnamedPlacesWithCoordinatesSitelinksAndTypes(t *testing.T) {
	var seen []string
	client := nearbyRoutes(t, readFixture(t, "wikidata_madeira_page.json"), "", &seen)

	places, _, err := NewWikidataNearbyAPI(client).pageItems(context.Background(), 32.75, -16.95, 0, 5, 0)
	require.NoError(t, err)

	require.Len(t, places, 4)
	assert.Equal(t, "Q639161", places[0].QID)
	assert.Equal(t, 50, places[0].Sitelinks)
	assert.InDelta(t, -16.774, places[0].Lon, 1e-9)
	assert.Equal(t, []string{"Q6017969"}, places[3].Types)
	for _, place := range places {
		assert.Empty(t, place.Names, "no names until the label query returns")
	}
}

func TestNearbyPageLabels_AsksForExactlyThePageItemsAndJoinsThem(t *testing.T) {
	var seen []string
	client := nearbyRoutes(t, readFixture(t, "wikidata_madeira_page.json"), readFixture(t, "wikidata_madeira.json"), &seen)
	api := NewWikidataNearbyAPI(client)
	places, _, err := api.pageItems(context.Background(), 32.75, -16.95, 0, 5, 0)
	require.NoError(t, err)

	rows, err := api.pageLabels(context.Background(), places)
	require.NoError(t, err)
	got := nearbyPlaces(rows)

	labels := seen[1]
	for _, qid := range []string{"wd:Q639161", "wd:Q623736", "wd:Q473169", "wd:Q856866"} {
		assert.Contains(t, labels, qid)
	}
	assert.Contains(t, labels, "P3025")
	assert.Contains(t, labels, "GROUP BY ?item")
	require.Len(t, got.Pins, 2)
	assert.Equal(t, []string{"Q473169", "Q856866"}, []string{got.Pins[0].QID, got.Pins[1].QID})
	assert.Equal(t, 32, got.Pins[0].Sitelinks)
	assert.InDelta(t, -16.954, got.Pins[0].Lon, 1e-9)
	assert.ElementsMatch(t, []string{"Pico Ruivo", "Ruivo Peak", "Pico Ruivo de Santana"}, got.Pins[0].Names)
	assert.Equal(t, "Open daily · 9:00 – 18:00", got.Pins[1].OpeningHours)
	require.Len(t, got.Gazetteer, 4)
}

func TestNearbyPageItems_AnItemWithSeveralTypeRowsIsOnePlace(t *testing.T) {
	var seen []string
	rows := `{"results":{"bindings":[
{"item":{"value":"http://www.wikidata.org/entity/Q1"},"coord":{"value":"Point(1 2)"},"sitelinks":{"value":"9"},"type":{"value":"http://www.wikidata.org/entity/Q8502"}},
{"item":{"value":"http://www.wikidata.org/entity/Q1"},"coord":{"value":"Point(1 2)"},"sitelinks":{"value":"9"},"type":{"value":"http://www.wikidata.org/entity/Q6017969"}}]}}`
	client := nearbyRoutes(t, rows, "", &seen)

	places, fetched, err := NewWikidataNearbyAPI(client).pageItems(context.Background(), 0, 0, 0, 5, 0)

	require.NoError(t, err)
	assert.Equal(t, 1, fetched, "type rows do not count towards the page size")
	require.Len(t, places, 1)
	assert.Equal(t, []string{"Q8502", "Q6017969"}, places[0].Types)
}

func TestNearbyPageItems_EmptyPageIsEmpty(t *testing.T) {
	var seen []string
	client := nearbyRoutes(t, `{"results":{"bindings":[]}}`, "", &seen)

	places, fetched, err := NewWikidataNearbyAPI(client).pageItems(context.Background(), 32.75, -16.95, 0, 5, 0)

	require.NoError(t, err)
	assert.Empty(t, places)
	assert.Zero(t, fetched)
}

func TestNearbyPageItems_RetriesAfter429ThenSucceeds(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: nearbyTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return &http.Response{StatusCode: http.StatusTooManyRequests, Body: io.NopCloser(strings.NewReader("busy")), Header: http.Header{"Retry-After": {"1"}}}, nil
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"results":{"bindings":[]}}`)), Header: make(http.Header)}, nil
	})}

	_, _, err := NewWikidataNearbyAPI(client).pageItems(context.Background(), 0, 0, 0, 5, 0)

	require.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestNearbyPageItems_ServerErrorIsReturned(t *testing.T) {
	client := &http.Client{Transport: nearbyTransport(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 502, Body: io.NopCloser(strings.NewReader("bad gateway")), Header: make(http.Header)}, nil
	})}

	_, _, err := NewWikidataNearbyAPI(client).pageItems(context.Background(), 0, 0, 0, 5, 0)

	require.ErrorContains(t, err, "status 502")
}

func TestNearbyPlaces_GroupedFieldsKeepAllValues(t *testing.T) {
	rows := []nearbyBinding{{}}
	rows[0].Item.Value = "http://www.wikidata.org/entity/Q1"
	rows[0].Coord.Value = "Point(-16.95 32.75)"
	rows[0].Type.Value = "http://www.wikidata.org/entity/Q8502\x1fhttp://www.wikidata.org/entity/Q6017969"
	rows[0].EnLabel.Value = "Cliff | View, Madeira"
	rows[0].PtLabel.Value = "Miradouro, Madeira"
	rows[0].EnAlias.Value = "Lookout | Madeira\x1fCliff, Madeira"
	rows[0].Sitelinks.Value = "30"
	rows[0].Hours.Value = "daily\x1e09:00:00\x1e18:00:00\x1fMonday\x1e10:00:00\x1e17:00:00"

	got := nearbyPlaces(rows)

	require.Len(t, got.Pins, 1)
	assert.Equal(t, []string{"Q8502", "Q6017969"}, got.Pins[0].Types)
	assert.ElementsMatch(t, []string{"Cliff | View, Madeira", "Miradouro, Madeira", "Lookout | Madeira", "Cliff, Madeira"}, got.Pins[0].Names)
	assert.Equal(t, "Open daily · 9:00 – 18:00; Open Monday · 10:00 – 17:00", got.Pins[0].OpeningHours)
}

// madeiraBody answers a Wikidata request with the Madeira page or the given label fixture.
func madeiraBody(t *testing.T, req *http.Request, labels string) string {
	t.Helper()
	if strings.Contains(req.URL.Query().Get("query"), "VALUES") {
		return labels
	}
	return readFixture(t, "wikidata_madeira_page.json")
}

// ringWorld serves generated pages: counts[outerKM] visitable items in each ring, 50 per page.
type ringWorld struct {
	t          *testing.T
	counts     map[string]int
	failAt     int // fail the nth page query (1-based); 0 never fails
	failLabels int // fail this many label queries
	labelCalls int
	density    int // items the density count reports
	countCalls int
	failCount  int // fail this many density counts
	countQuery string
	pages      []ringCall
	next       int
}

type ringCall struct {
	Inner, Outer string
	Offset       int
}

var (
	ringRadiusPattern = regexp.MustCompile(`wikibase:radius "([^"]+)"`)
	ringInnerPattern  = regexp.MustCompile(`FILTER\(\?dist > ([^)]+)\)`)
	ringOffsetPattern = regexp.MustCompile(`OFFSET (\d+)`)
	ringQIDPattern    = regexp.MustCompile(`wd:(Q\d+)`)
)

func (w *ringWorld) client() *http.Client {
	return &http.Client{Transport: nearbyTransport(func(req *http.Request) (*http.Response, error) {
		query := req.URL.Query().Get("query")
		body := ""
		if strings.Contains(query, "COUNT(") {
			w.countCalls++
			w.countQuery = query
			if w.failCount > 0 {
				w.failCount--
				return &http.Response{StatusCode: 502, Body: io.NopCloser(strings.NewReader("bad gateway")), Header: make(http.Header)}, nil
			}
			body = fmt.Sprintf(`{"results":{"bindings":[{"n":{"value":"%d"}}]}}`, w.density)
		} else if strings.Contains(query, "VALUES") {
			w.labelCalls++
			if w.failLabels > 0 {
				w.failLabels--
				return &http.Response{StatusCode: 502, Body: io.NopCloser(strings.NewReader("bad gateway")), Header: make(http.Header)}, nil
			}
			var rows []string
			for _, match := range ringQIDPattern.FindAllStringSubmatch(query, -1) {
				rows = append(rows, fmt.Sprintf(`{"item":{"value":"http://www.wikidata.org/entity/%s"},"type":{"value":"http://www.wikidata.org/entity/Q8502"},"enLabel":{"value":"Peak %s"}}`, match[1], match[1]))
			}
			body = `{"results":{"bindings":[` + strings.Join(rows, ",") + `]}}`
		} else {
			call := ringCall{Outer: ringRadiusPattern.FindStringSubmatch(query)[1], Inner: "0"}
			if m := ringInnerPattern.FindStringSubmatch(query); m != nil {
				call.Inner = m[1]
			}
			call.Offset, _ = strconv.Atoi(ringOffsetPattern.FindStringSubmatch(query)[1])
			w.pages = append(w.pages, call)
			if w.failAt > 0 && len(w.pages) == w.failAt {
				return &http.Response{StatusCode: 502, Body: io.NopCloser(strings.NewReader("bad gateway")), Header: make(http.Header)}, nil
			}
			total, exact := w.counts[call.Inner+"-"+call.Outer]
			if !exact {
				total = w.counts[call.Outer]
			}
			n := min(nearbyPageSize, total-call.Offset)
			var rows []string
			for i := 0; i < max(n, 0); i++ {
				w.next++
				rows = append(rows, fmt.Sprintf(`{"item":{"value":"http://www.wikidata.org/entity/Q%d"},"coord":{"value":"Point(-16.95 32.75)"},"sitelinks":{"value":"%d"},"type":{"value":"http://www.wikidata.org/entity/Q8502"}}`, w.next, 1000-w.next))
			}
			body = `{"results":{"bindings":[` + strings.Join(rows, ",") + `]}}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
}

func (w *ringWorld) calls() []string {
	var out []string
	for _, c := range w.pages {
		out = append(out, fmt.Sprintf("%s-%s@%d", c.Inner, c.Outer, c.Offset))
	}
	return out
}

func TestWikidataNearby_PlacesFrom_LoadsRingsNearestFirstAndPagesWithinARing(t *testing.T) {
	world := &ringWorld{t: t, counts: map[string]int{"5": 60, "10": 30, "15": 10, "20": 5, "25": 5, "30": 5, "35": 5, "40": 5}}

	got, err := NewWikidataNearbyAPI(world.client()).PlacesFrom(context.Background(), writermap.Area{}, writermap.BaseProgress{}, nil)

	require.NoError(t, err)
	assert.Equal(t, []string{"0-5@0", "0-5@50", "5-10@0", "10-15@0", "15-20@0", "20-25@0", "25-30@0", "30-35@0", "35-40@0"}, world.calls())
	assert.True(t, got.Done)
	assert.Len(t, got.Gazetteer, 60+30+10+5*5)
}

func TestWikidataNearby_PlacesFrom_ReportsProgressAfterEveryPage(t *testing.T) {
	world := &ringWorld{t: t, counts: map[string]int{"5": 60, "10": 20, "15": 10, "20": 5, "25": 5, "30": 5, "35": 5, "40": 5}}
	var seen []int
	var cursors [][2]int

	_, err := NewWikidataNearbyAPI(world.client()).PlacesFrom(context.Background(), writermap.Area{}, writermap.BaseProgress{}, func(p writermap.BaseProgress) {
		if len(p.Unnamed) > 0 {
			return // the nameless publish before a page's labels
		}
		seen = append(seen, len(p.Gazetteer))
		cursors = append(cursors, [2]int{p.Ring, p.Offset})
	})

	require.NoError(t, err)
	assert.Equal(t, []int{50, 60, 80}, seen[:3], "each page adds its items")
	assert.Equal(t, [][2]int{{0, 50}, {1, 0}, {2, 0}}, cursors[:3], "a full page keeps the ring and moves the offset; a short page moves to the next ring")
}

func TestWikidataNearby_PlacesFrom_StopsWhenTwoHundredPinsAreFilled(t *testing.T) {
	world := &ringWorld{t: t, counts: map[string]int{"5": 500, "10": 500}}

	got, err := NewWikidataNearbyAPI(world.client()).PlacesFrom(context.Background(), writermap.Area{}, writermap.BaseProgress{}, nil)

	require.NoError(t, err)
	assert.Equal(t, []string{"0-5@0", "0-5@50", "0-5@100", "0-5@150"}, world.calls(), "no ring past the page that fills the pins")
	assert.True(t, got.Done)
	assert.Len(t, got.Pins, 200)
	assert.Equal(t, "Q1", got.Pins[0].QID, "most-linked first")
}

func TestWikidataNearby_PlacesFrom_AThinRingEndsTheLoad(t *testing.T) {
	world := &ringWorld{t: t, counts: map[string]int{"5": 80, "10": 4, "15": 40}}

	got, err := NewWikidataNearbyAPI(world.client()).PlacesFrom(context.Background(), writermap.Area{}, writermap.BaseProgress{}, nil)

	require.NoError(t, err)
	assert.Equal(t, []string{"0-5@0", "0-5@50", "5-10@0"}, world.calls())
	assert.True(t, got.Done)
	assert.Len(t, got.Gazetteer, 84, "the thin ring's own items are kept")
}

func TestWikidataNearby_PlacesFrom_RadiusEndsTheLastRingAtTheAreaRadius(t *testing.T) {
	world := &ringWorld{t: t, counts: map[string]int{"5": 20, "10": 20, "12": 20, "15": 20}}

	got, err := NewWikidataNearbyAPI(world.client()).PlacesFrom(context.Background(), writermap.Area{RadiusKM: 12}, writermap.BaseProgress{}, nil)

	require.NoError(t, err)
	assert.Equal(t, []string{"0-5@0", "5-10@0", "10-12@0"}, world.calls())
	assert.True(t, got.Done)
}

func TestWikidataNearby_PlacesFrom_AFailingPageReturnsWhatWasLoadedAndResumes(t *testing.T) {
	world := &ringWorld{t: t, counts: map[string]int{"5": 60, "10": 20, "15": 10, "20": 5, "25": 5, "30": 5, "35": 5, "40": 5}, failAt: 3}
	api := NewWikidataNearbyAPI(world.client())

	partial, err := api.PlacesFrom(context.Background(), writermap.Area{}, writermap.BaseProgress{}, nil)

	require.ErrorContains(t, err, "status 502")
	assert.False(t, partial.Done)
	assert.Len(t, partial.Gazetteer, 60, "pages 1 and 2 were kept")
	assert.Equal(t, 1, partial.Ring)

	world.failAt = 0
	world.pages = nil
	got, err := api.PlacesFrom(context.Background(), writermap.Area{}, partial, nil)

	require.NoError(t, err)
	assert.True(t, got.Done)
	assert.Equal(t, "5-10@0", world.calls()[0], "resumes at the stored ring and offset, without refetching earlier pages")
	assert.Len(t, got.Gazetteer, 60+20+10+5*5)
}

func TestWikidataNearby_PlacesFrom_AlreadyDoneProgressMakesNoQueries(t *testing.T) {
	world := &ringWorld{t: t}
	done := writermap.BaseProgress{Done: true}

	got, err := NewWikidataNearbyAPI(world.client()).PlacesFrom(context.Background(), writermap.Area{}, done, nil)

	require.NoError(t, err)
	assert.True(t, got.Done)
	assert.Empty(t, world.pages)
}

func TestWikidataNearby_PlacesFrom_PhotoCreditsAreResolvedPerPage(t *testing.T) {
	commons := readFixture(t, "wikidata_commons.json")
	var titles []string
	client := &http.Client{Transport: nearbyTransport(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Host, "commons.wikimedia.org") {
			titles = append(titles, req.URL.Query().Get("titles"))
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(commons)), Header: make(http.Header)}, nil
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(madeiraBody(t, req, readFixture(t, "wikidata_madeira.json")))), Header: make(http.Header)}, nil
	})}
	var pinPhotoAtFirstPage string

	_, err := NewWikidataNearbyAPI(client).PlacesFrom(context.Background(), writermap.Area{}, writermap.BaseProgress{}, func(p writermap.BaseProgress) {
		if pinPhotoAtFirstPage == "" && len(p.Pins) > 0 {
			pinPhotoAtFirstPage = p.Pins[0].PhotoURL
		}
	})

	require.NoError(t, err)
	assert.NotEmpty(t, pinPhotoAtFirstPage, "a partial pin already has its photo")
	assert.Len(t, titles, 1)
}

func TestWikidataNearby_PlacesFrom_PublishesNamelessPinsBeforeTheLabelQuery(t *testing.T) {
	world := &ringWorld{t: t, counts: map[string]int{"5": 10}}
	var first writermap.BaseProgress
	var labelCallsAtFirst = -1

	_, err := NewWikidataNearbyAPI(world.client()).PlacesFrom(context.Background(), writermap.Area{}, writermap.BaseProgress{}, func(p writermap.BaseProgress) {
		if labelCallsAtFirst < 0 {
			first, labelCallsAtFirst = p, world.labelCalls
		}
	})

	require.NoError(t, err)
	assert.Zero(t, labelCallsAtFirst, "the first publish happens before any label query")
	assert.Len(t, first.Pins, 10, "visitable items are pins as soon as the page returns")
	for _, pin := range first.Pins {
		assert.Empty(t, pin.Names)
	}
	assert.Empty(t, first.Gazetteer, "nameless places are not used for matching")
	assert.Len(t, first.Unnamed, 10)
	assert.Equal(t, 10, first.PageRows)
	assert.Zero(t, first.Offset, "the ring cursor moves only once the page is named")
}

func TestWikidataNearby_PlacesFrom_FinalProgressHasNamesAndNoPendingPage(t *testing.T) {
	world := &ringWorld{t: t, counts: map[string]int{"5": 10}}

	got, err := NewWikidataNearbyAPI(world.client()).PlacesFrom(context.Background(), writermap.Area{}, writermap.BaseProgress{}, nil)

	require.NoError(t, err)
	assert.Empty(t, got.Unnamed)
	assert.Zero(t, got.PageRows)
	require.Len(t, got.Pins, 10)
	assert.NotEmpty(t, got.Pins[0].Names)
}

func TestWikidataNearby_PlacesFrom_FailedLabelQueryKeepsThePinsAndRetriesOnlyTheLabels(t *testing.T) {
	world := &ringWorld{t: t, counts: map[string]int{"5": 10}, failLabels: 1}
	api := NewWikidataNearbyAPI(world.client())

	partial, err := api.PlacesFrom(context.Background(), writermap.Area{}, writermap.BaseProgress{}, nil)

	require.ErrorContains(t, err, "status 502")
	assert.Len(t, partial.Pins, 10, "the nameless pins stay")
	assert.Len(t, partial.Unnamed, 10)
	assert.Equal(t, 1, len(world.pages))

	got, err := api.PlacesFrom(context.Background(), writermap.Area{}, partial, nil)

	require.NoError(t, err)
	assert.Equal(t, []string{"0-5@0", "5-10@0"}, world.calls(), "ring 1 is not queried again: only its labels are retried")
	assert.Equal(t, 2, world.labelCalls)
	assert.True(t, got.Done)
	assert.NotEmpty(t, got.Pins[0].Names)
}

func TestWikidataNearby_PlacesFrom_StopRuleCountsOnlyNamedPins(t *testing.T) {
	world := &ringWorld{t: t, counts: map[string]int{"5": 500}, failLabels: 1}
	api := NewWikidataNearbyAPI(world.client())

	partial, err := api.PlacesFrom(context.Background(), writermap.Area{}, writermap.BaseProgress{}, nil)

	require.Error(t, err)
	assert.False(t, partial.Done)
	assert.Empty(t, partial.Gazetteer)
}

// denseWorld has a full first 5 km page, then 10 items in each 1 km ring and 20 in each later 5 km ring.
func denseWorld(t *testing.T, density int) *ringWorld {
	counts := map[string]int{"0-5": 60, "10": 20, "15": 20, "20": 20, "25": 20, "30": 20, "35": 20, "40": 20, "5-10": 20}
	for i := 0; i < 10; i++ {
		counts[fmt.Sprintf("%d-%d", i, i+1)] = 10
	}
	return &ringWorld{t: t, counts: counts, density: density}
}

func TestNearbyDensity_CountIsCappedAndAsksForItemsAtTheFloorWithinFiveKm(t *testing.T) {
	world := denseWorld(t, 100)

	_, err := NewWikidataNearbyAPI(world.client()).PlacesFrom(context.Background(), writermap.Area{Lat: 51.5, Lon: -0.12}, writermap.BaseProgress{}, nil)

	require.NoError(t, err)
	assert.Equal(t, 1, world.countCalls)
	assert.Contains(t, world.countQuery, `wikibase:radius "5"`)
	assert.Contains(t, world.countQuery, "?sl >= 2")
	assert.Contains(t, world.countQuery, "LIMIT 1501")
	assert.Contains(t, world.countQuery, "Point(-0.12 51.5)")
}

func TestWikidataNearby_PlacesFrom_DenseCityLoadsEightOneKmRoundsAndStops(t *testing.T) {
	world := denseWorld(t, 1501)

	got, err := NewWikidataNearbyAPI(world.client()).PlacesFrom(context.Background(), writermap.Area{}, writermap.BaseProgress{}, nil)

	require.NoError(t, err)
	assert.Equal(t, "dense", got.Mode)
	assert.Equal(t, []string{"0-5@0", "0-1@0", "1-2@0", "2-3@0", "3-4@0", "4-5@0", "5-6@0", "6-7@0", "7-8@0"}, world.calls(),
		"the first page is a normal 5 km page; then exactly 8 rounds of 1 km, and nothing past 8 km")
	assert.True(t, got.Done)
	assert.Equal(t, 1, world.countCalls)
}

func TestWikidataNearby_PlacesFrom_CityAtOrBelowTheThresholdStaysNormal(t *testing.T) {
	for _, density := range []int{144, 797, 1500} {
		world := denseWorld(t, density)

		got, err := NewWikidataNearbyAPI(world.client()).PlacesFrom(context.Background(), writermap.Area{}, writermap.BaseProgress{}, nil)

		require.NoError(t, err)
		assert.Equal(t, "normal", got.Mode, "density %d", density)
		assert.Equal(t, []string{"0-5@0", "0-5@50", "5-10@0", "10-15@0", "15-20@0", "20-25@0", "25-30@0", "30-35@0", "35-40@0"}, world.calls(), "density %d", density)
	}
}

func TestWikidataNearby_PlacesFrom_DenseModeStopsAtASmallerAreaRadius(t *testing.T) {
	world := denseWorld(t, 5000)

	_, err := NewWikidataNearbyAPI(world.client()).PlacesFrom(context.Background(), writermap.Area{RadiusKM: 6}, writermap.BaseProgress{}, nil)

	require.NoError(t, err)
	assert.Equal(t, "5-6@0", world.calls()[len(world.calls())-1])
	assert.Len(t, world.calls(), 1+5+1)
}

func TestWikidataNearby_PlacesFrom_ModeSurvivesAResumeAndIsNotDetectedAgain(t *testing.T) {
	world := denseWorld(t, 5000)
	world.failAt = 4 // the page query of the 2-3 km round
	api := NewWikidataNearbyAPI(world.client())

	partial, err := api.PlacesFrom(context.Background(), writermap.Area{}, writermap.BaseProgress{}, nil)
	require.Error(t, err)
	assert.Equal(t, "dense", partial.Mode)
	stored, jsonErr := json.Marshal(partial)
	require.NoError(t, jsonErr)
	var reloaded writermap.BaseProgress
	require.NoError(t, json.Unmarshal(stored, &reloaded))

	world.failAt, world.pages, world.density = 0, nil, 0
	got, err := api.PlacesFrom(context.Background(), writermap.Area{}, reloaded, nil)

	require.NoError(t, err)
	assert.Equal(t, "dense", got.Mode, "a density of 0 now would say normal; the stored mode wins")
	assert.Equal(t, "2-3@0", world.calls()[0], "resumes at the failed 1 km round")
	assert.Equal(t, 1, world.countCalls, "detected once per destination")
	assert.True(t, got.Done)
}

func TestWikidataNearby_PlacesFrom_FailedDensityCountIsRetriedOnResumeWithoutRefetchingPages(t *testing.T) {
	world := denseWorld(t, 5000)
	world.failCount = 1
	api := NewWikidataNearbyAPI(world.client())

	partial, err := api.PlacesFrom(context.Background(), writermap.Area{}, writermap.BaseProgress{}, nil)

	require.ErrorContains(t, err, "status 502")
	assert.Empty(t, partial.Mode)
	assert.Len(t, partial.Pins, 50, "the first page's pins are kept")
	world.pages = nil
	got, err := api.PlacesFrom(context.Background(), writermap.Area{}, partial, nil)

	require.NoError(t, err)
	assert.Equal(t, "dense", got.Mode)
	assert.Equal(t, "0-1@0", world.calls()[0], "the count is retried, the first page is not")
}

func TestWikidataNearby_PlacesFrom_NoDensityCountWhenTheFirstPageAlreadyEndsTheLoad(t *testing.T) {
	world := &ringWorld{t: t, counts: map[string]int{"5": 3}, density: 9999}

	got, err := NewWikidataNearbyAPI(world.client()).PlacesFrom(context.Background(), writermap.Area{}, writermap.BaseProgress{}, nil)

	require.NoError(t, err)
	assert.True(t, got.Done)
	assert.Zero(t, world.countCalls)
}

func TestWikidataNearby_PlacesFrom_FirstPinsDoNotWaitForTheDensityCount(t *testing.T) {
	world := denseWorld(t, 5000)
	var countsAtFirstPublish = -1

	_, err := NewWikidataNearbyAPI(world.client()).PlacesFrom(context.Background(), writermap.Area{}, writermap.BaseProgress{}, func(p writermap.BaseProgress) {
		if countsAtFirstPublish < 0 {
			countsAtFirstPublish = world.countCalls
		}
	})

	require.NoError(t, err)
	assert.Zero(t, countsAtFirstPublish)
}
