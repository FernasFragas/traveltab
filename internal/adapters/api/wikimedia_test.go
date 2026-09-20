package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"weatherservice/internal/planner"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	taviraLat = 37.1264
	taviraLon = -7.6506

	// wikiSearchCap is the number of results a geosearch returns when it is capped.
	wikiSearchCap = 500
)

// newStubWikimediaAPI answers Wikimedia requests with respond instead of calling the real
// APIs, and records every request it receives. Requests can be in flight together, so the
// recording is guarded; the slice is only safe to read once the call under test has returned.
func newStubWikimediaAPI(respond func(*http.Request) stubResponse) (*WikimediaAPI, *[]*http.Request) {
	var (
		mu       sync.Mutex
		requests []*http.Request
	)

	api := NewWikimediaAPI(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if err := req.Context().Err(); err != nil {
			return nil, err
		}

		mu.Lock()
		requests = append(requests, req)
		mu.Unlock()

		stub := respond(req)

		return &http.Response{
			StatusCode: stub.status,
			Body:       io.NopCloser(strings.NewReader(stub.body)),
			Header:     http.Header{},
		}, nil
	})})

	return api, &requests
}

// inFlight counts the requests being answered at the same moment and remembers the most it ever
// saw at once.
type inFlight struct {
	mu      sync.Mutex
	now     int
	highest int
}

func (f *inFlight) enter() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.now++
	if f.now > f.highest {
		f.highest = f.now
	}

	return f.now
}

func (f *inFlight) leave() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.now--
}

func (f *inFlight) max() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.highest
}

// crowdWait is how long a held request waits for company. A client that sends its requests one
// after another spends it on every request and never sees more than one in flight.
const crowdWait = 50 * time.Millisecond

// crowded holds every answer until wikimediaConcurrency+1 requests are waiting together, or until
// crowdWait passes. A client that keeps to the limit always waits, so a whole wave of requests
// overlaps and gets counted; one that goes over the limit is let straight through and shows up as
// a count above the limit.
func crowded(counter *inFlight, respond func(*http.Request) stubResponse) func(*http.Request) stubResponse {
	full := make(chan struct{})

	var once sync.Once

	return func(req *http.Request) stubResponse {
		if counter.enter() > wikimediaConcurrency {
			once.Do(func() { close(full) })
		}

		defer counter.leave()

		select {
		case <-full:
		case <-time.After(crowdWait):
		}

		return respond(req)
	}
}

// taviraFixtures answers each Wikimedia endpoint with the response recorded for Tavira.
func taviraFixtures(t *testing.T) func(*http.Request) stubResponse {
	t.Helper()

	geosearch := wikiFixture(t, "geosearch_tavira.json")
	pageprops := wikiFixture(t, "pageprops_tavira.json")
	sitelinks := wikiFixture(t, "sitelinks_tavira.json")
	entities := wikiFixture(t, "entities_tavira.json")

	return func(req *http.Request) stubResponse {
		q := req.URL.Query()
		switch {
		case q.Get("list") == "geosearch":
			return stubResponse{http.StatusOK, geosearch}
		case q.Get("prop") == "pageprops":
			return stubResponse{http.StatusOK, pageprops}
		case q.Get("props") == "sitelinks":
			return stubResponse{http.StatusOK, sitelinks}
		case strings.Contains(q.Get("props"), "claims"):
			return stubResponse{http.StatusOK, entities}
		default:
			return stubResponse{http.StatusNotFound, `{"error":"unexpected request"}`}
		}
	}
}

func wikiFixture(t *testing.T, name string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", "wikimedia", name))
	require.NoError(t, err)

	return string(data)
}

// wikiPage is one page in the fake Wikimedia data the stub serves.
type wikiPage struct {
	id        int
	title     string
	lat, lon  float64
	qid       string
	sitelinks int
	types     []string
	image     string
	label     string // empty when Wikidata has no English label
}

func (p wikiPage) place() planner.Place {
	return planner.Place{Lat: p.lat, Lon: p.lon}
}

// fakeWikimedia answers geosearch with the pages inside the requested radius, closest first and
// capped at the requested limit, and answers the detail endpoints for the IDs they ask about.
func fakeWikimedia(pages []wikiPage) func(*http.Request) stubResponse {
	byPageID := make(map[int]wikiPage, len(pages))
	byQID := make(map[string]wikiPage, len(pages))
	for _, p := range pages {
		byPageID[p.id] = p
		byQID[p.qid] = p
	}

	return func(req *http.Request) stubResponse {
		q := req.URL.Query()
		switch {
		case q.Get("list") == "geosearch":
			return stubResponse{http.StatusOK, fakeGeosearch(pages, q)}
		case q.Get("prop") == "pageprops":
			return stubResponse{http.StatusOK, fakePageProps(byPageID, q.Get("pageids"))}
		case q.Get("props") == "sitelinks":
			return stubResponse{http.StatusOK, fakeSitelinks(byQID, q.Get("ids"))}
		case strings.Contains(q.Get("props"), "claims"):
			return stubResponse{http.StatusOK, fakeEntityDetails(byQID, q.Get("ids"))}
		default:
			return stubResponse{http.StatusNotFound, `{"error":"unexpected request"}`}
		}
	}
}

func fakeGeosearch(pages []wikiPage, q url.Values) string {
	coord := strings.Split(q.Get("gscoord"), "|")
	lat, _ := strconv.ParseFloat(coord[0], 64)
	lon, _ := strconv.ParseFloat(coord[len(coord)-1], 64)
	radiusM, _ := strconv.ParseFloat(q.Get("gsradius"), 64)
	limit, _ := strconv.Atoi(q.Get("gslimit"))

	center := planner.Place{Lat: lat, Lon: lon}

	type hit struct {
		page   wikiPage
		distKM float64
	}

	var hits []hit
	for _, p := range pages {
		if distKM := planner.DistanceKM(center, p.place()); distKM*1000 <= radiusM {
			hits = append(hits, hit{page: p, distKM: distKM})
		}
	}

	sort.Slice(hits, func(i, j int) bool { return hits[i].distKM < hits[j].distKM })
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}

	results := make([]map[string]any, 0, len(hits))
	for _, h := range hits {
		results = append(results, map[string]any{
			"pageid": h.page.id,
			"ns":     0,
			"title":  h.page.title,
			"lat":    h.page.lat,
			"lon":    h.page.lon,
			"dist":   math.Round(h.distKM * 1000),
		})
	}

	return wikiJSON(map[string]any{"query": map[string]any{"geosearch": results}})
}

func fakePageProps(byPageID map[int]wikiPage, ids string) string {
	pages := make([]map[string]any, 0)
	for _, raw := range strings.Split(ids, "|") {
		id, err := strconv.Atoi(raw)
		if err != nil {
			continue
		}

		p, ok := byPageID[id]
		if !ok {
			continue
		}

		entry := map[string]any{"pageid": p.id, "ns": 0, "title": p.title}
		if p.qid != "" {
			entry["pageprops"] = map[string]string{"wikibase_item": p.qid}
		}

		pages = append(pages, entry)
	}

	return wikiJSON(map[string]any{"query": map[string]any{"pages": pages}})
}

func fakeSitelinks(byQID map[string]wikiPage, ids string) string {
	entities := map[string]any{}
	for _, qid := range strings.Split(ids, "|") {
		p, ok := byQID[qid]
		if !ok {
			continue
		}

		sitelinks := map[string]any{}
		for i := 0; i < p.sitelinks; i++ {
			site := fmt.Sprintf("wiki%d", i)
			sitelinks[site] = map[string]string{"site": site, "title": p.title}
		}

		entities[qid] = map[string]any{"id": qid, "sitelinks": sitelinks}
	}

	return wikiJSON(map[string]any{"entities": entities, "success": 1})
}

func fakeEntityDetails(byQID map[string]wikiPage, ids string) string {
	entities := map[string]any{}
	for _, qid := range strings.Split(ids, "|") {
		p, ok := byQID[qid]
		if !ok {
			continue
		}

		claims := map[string]any{}
		if len(p.types) > 0 {
			instanceOf := make([]any, 0, len(p.types))
			for _, typeID := range p.types {
				instanceOf = append(instanceOf, wikiItemClaim("P31", typeID))
			}
			claims["P31"] = instanceOf
		}
		if p.image != "" {
			claims["P18"] = []any{wikiStringClaim("P18", p.image)}
		}

		entity := map[string]any{"id": qid, "type": "item", "claims": claims}
		if p.label != "" {
			entity["labels"] = map[string]any{"en": map[string]string{"language": "en", "value": p.label}}
		}

		entities[qid] = entity
	}

	return wikiJSON(map[string]any{"entities": entities, "success": 1})
}

func wikiItemClaim(property, itemID string) map[string]any {
	return map[string]any{"mainsnak": map[string]any{
		"snaktype": "value",
		"property": property,
		"datavalue": map[string]any{
			"value": map[string]any{"entity-type": "item", "id": itemID},
			"type":  "wikibase-entityid",
		},
		"datatype": "wikibase-item",
	}}
}

func wikiStringClaim(property, value string) map[string]any {
	return map[string]any{"mainsnak": map[string]any{
		"snaktype":  "value",
		"property":  property,
		"datavalue": map[string]any{"value": value, "type": "string"},
		"datatype":  "commonsMedia",
	}}
}

func wikiJSON(v any) string {
	encoded, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	return string(encoded)
}

// wikiTownPages spreads n pages around a center, all of them within spreadKM.
func wikiTownPages(n int, lat, lon, spreadKM float64) []wikiPage {
	pages := make([]wikiPage, 0, n)
	for i := 0; i < n; i++ {
		angle := float64(i) * 2 * math.Pi / float64(n)
		radiusKM := spreadKM * float64(i%10+1) / 10

		pages = append(pages, wikiPage{
			id:        1000 + i,
			title:     fmt.Sprintf("Place %d", i),
			lat:       lat + radiusKM/111.32*math.Cos(angle),
			lon:       lon + radiusKM/(111.32*math.Cos(lat*math.Pi/180))*math.Sin(angle),
			qid:       fmt.Sprintf("Q%d", 1000+i),
			sitelinks: 1,
			label:     fmt.Sprintf("Place %d", i),
		})
	}

	return pages
}

// wikiNorthOf is the latitude km kilometres due north of lat.
func wikiNorthOf(lat, km float64) float64 {
	return lat + km/111.32
}

func findPlace(places []planner.Place, id string) *planner.Place {
	for i := range places {
		if places[i].ID == id {
			return &places[i]
		}
	}

	return nil
}

func countGeosearchRequests(requests []*http.Request) int {
	count := 0
	for _, req := range requests {
		if req.URL.Query().Get("list") == "geosearch" {
			count++
		}
	}

	return count
}

func TestWikimedia_PlacesNear_ReturnsPlacesWithWikidataDetails(t *testing.T) {
	api, _ := newStubWikimediaAPI(taviraFixtures(t))

	places, err := api.PlacesNear(context.Background(), taviraLat, taviraLon)
	require.NoError(t, err)

	castle := findPlace(places, "Q5049832")
	require.NotNil(t, castle, "Castle of Tavira should be among the places near Tavira")
	assert.Equal(t, "Castle of Tavira", castle.Name)
	assert.InDelta(t, 37.1252, castle.Lat, 0.0001)
	assert.InDelta(t, -7.6513, castle.Lon, 0.0001)
	assert.Equal(t, 4, castle.Sitelinks)
	assert.Equal(t, []string{"Q23413", "Q210272"}, castle.Types)
	assert.Equal(t, "TaviraCastle-CCBY.jpg", castle.Image)
}

func TestWikimedia_PlacesNear_LeavesKindEmpty(t *testing.T) {
	api, _ := newStubWikimediaAPI(taviraFixtures(t))

	places, err := api.PlacesNear(context.Background(), taviraLat, taviraLon)
	require.NoError(t, err)
	require.NotEmpty(t, places)

	for _, place := range places {
		assert.Empty(t, string(place.Kind), "%s should have no kind yet", place.Name)
	}
}

func TestWikimedia_PlacesNear_UsesTheTitleWhenThereIsNoEnglishLabel(t *testing.T) {
	pages := []wikiPage{{
		id:        16742104,
		title:     "Pego do Inferno",
		lat:       taviraLat,
		lon:       taviraLon,
		qid:       "Q16742104",
		sitelinks: 3,
		types:     []string{"Q34038"},
	}}

	api, _ := newStubWikimediaAPI(fakeWikimedia(pages))

	places, err := api.PlacesNear(context.Background(), taviraLat, taviraLon)
	require.NoError(t, err)
	require.Len(t, places, 1)
	assert.Equal(t, "Pego do Inferno", places[0].Name)
}

func TestWikimedia_PlacesNear_SplitsACappedSearch(t *testing.T) {
	outer := wikiPage{
		id:        9001,
		title:     "Far Chapel",
		lat:       wikiNorthOf(taviraLat, 9.5),
		lon:       taviraLon,
		qid:       "Q9001",
		sitelinks: 2,
		label:     "Far Chapel",
	}
	pages := append(wikiTownPages(wikiSearchCap, taviraLat, taviraLon, 4), outer)

	api, _ := newStubWikimediaAPI(fakeWikimedia(pages))

	places, err := api.PlacesNear(context.Background(), taviraLat, taviraLon)
	require.NoError(t, err)

	assert.NotNil(t, findPlace(places, "Q9001"), "a place 9.5 km out is only found by an outer search")
}

func TestWikimedia_PlacesNear_DropsSplitResultsBeyond10Km(t *testing.T) {
	tooFar := wikiPage{
		id:        9002,
		title:     "Distant Fort",
		lat:       wikiNorthOf(taviraLat, 12),
		lon:       taviraLon,
		qid:       "Q9002",
		sitelinks: 2,
		label:     "Distant Fort",
	}
	pages := append(wikiTownPages(wikiSearchCap, taviraLat, taviraLon, 4), tooFar)

	api, _ := newStubWikimediaAPI(fakeWikimedia(pages))

	places, err := api.PlacesNear(context.Background(), taviraLat, taviraLon)
	require.NoError(t, err)

	assert.Len(t, places, wikiSearchCap)
	assert.Nil(t, findPlace(places, "Q9002"), "a place 12 km out is outside the search radius")
}

func TestWikimedia_PlacesNear_ReturnsEachPlaceOnce(t *testing.T) {
	shared := wikiPage{
		id:        9003,
		title:     "Shared Church",
		lat:       wikiNorthOf(taviraLat, 4.2),
		lon:       taviraLon,
		qid:       "Q9003",
		sitelinks: 5,
		label:     "Shared Church",
	}
	pages := append(wikiTownPages(wikiSearchCap, taviraLat, taviraLon, 9.5), shared)

	// Both the centre search and the northern one, 8.66 km out, reach this place.
	centre := planner.Place{Lat: taviraLat, Lon: taviraLon}
	north := planner.Place{Lat: wikiNorthOf(taviraLat, 8.66), Lon: taviraLon}
	require.Less(t, planner.DistanceKM(centre, shared.place()), 5.0)
	require.Less(t, planner.DistanceKM(north, shared.place()), 5.0)

	api, _ := newStubWikimediaAPI(fakeWikimedia(pages))

	places, err := api.PlacesNear(context.Background(), taviraLat, taviraLon)
	require.NoError(t, err)

	seen := map[string]int{}
	for _, place := range places {
		seen[place.ID]++
	}

	assert.Equal(t, 1, seen["Q9003"], "the centre and the northern search both return this place")
	for id, times := range seen {
		assert.Equal(t, 1, times, "%s should appear once", id)
	}
}

func TestWikimedia_PlacesNear_DoesNotSplitAnUncappedSearch(t *testing.T) {
	pages := wikiTownPages(218, taviraLat, taviraLon, 9)

	api, requests := newStubWikimediaAPI(fakeWikimedia(pages))

	places, err := api.PlacesNear(context.Background(), taviraLat, taviraLon)
	require.NoError(t, err)

	assert.Len(t, places, 218)
	assert.Equal(t, 1, countGeosearchRequests(*requests), "218 results fit in one search")
}

func TestWikimedia_PlacesNear_HasNoTypesBeyondTheTop300(t *testing.T) {
	pages := wikiTownPages(320, taviraLat, taviraLon, 9)
	for i := range pages {
		pages[i].sitelinks = len(pages) - i // most famous first
		pages[i].types = []string{"Q33506"}
		pages[i].image = "Museum.jpg"
	}

	api, _ := newStubWikimediaAPI(fakeWikimedia(pages))

	places, err := api.PlacesNear(context.Background(), taviraLat, taviraLon)
	require.NoError(t, err)
	require.Len(t, places, 320)

	threeHundredth := findPlace(places, pages[299].qid)
	require.NotNil(t, threeHundredth)
	assert.Equal(t, []string{"Q33506"}, threeHundredth.Types, "the 300th most famous place gets its types")

	beyondTop300 := findPlace(places, pages[300].qid)
	require.NotNil(t, beyondTop300)
	assert.Empty(t, beyondTop300.Types, "the 301st most famous place is not looked up")
	assert.Empty(t, beyondTop300.Image)
}

func TestWikimedia_PlacesNear_LooksUpBatchesTogetherButAtMostFourAtATime(t *testing.T) {
	pages := wikiTownPages(320, taviraLat, taviraLon, 9)
	counter := &inFlight{}

	api, requests := newStubWikimediaAPI(crowded(counter, fakeWikimedia(pages)))

	places, err := api.PlacesNear(context.Background(), taviraLat, taviraLon)
	require.NoError(t, err)
	require.Len(t, places, 320)
	require.Greater(t, len(*requests), wikimediaConcurrency,
		"320 places take more than %d requests to describe", wikimediaConcurrency)

	assert.Greater(t, counter.max(), 1, "the batch lookups should not queue up behind each other")
	assert.LessOrEqual(t, counter.max(), wikimediaConcurrency,
		"Wikimedia should never see more than %d requests at once", wikimediaConcurrency)
}

func TestWikimedia_PlacesNear_KeepsItsOrderWhenBatchesFinishOutOfOrder(t *testing.T) {
	// Guard: batches that run together finish in whatever order they like, and the places they
	// describe still have to come back exactly as they do when nothing is delayed.
	pages := wikiTownPages(320, taviraLat, taviraLon, 9)
	for i := range pages {
		pages[i].sitelinks = len(pages) - i // most famous first
		pages[i].types = []string{fmt.Sprintf("Q%d", 33506+i%3)}
		pages[i].image = fmt.Sprintf("Place%d.jpg", i)
	}

	straight, _ := newStubWikimediaAPI(fakeWikimedia(pages))
	want, err := straight.PlacesNear(context.Background(), taviraLat, taviraLon)
	require.NoError(t, err)
	require.Len(t, want, 320)

	backwards, _ := newStubWikimediaAPI(lateFirst(fakeWikimedia(pages)))
	got, err := backwards.PlacesNear(context.Background(), taviraLat, taviraLon)
	require.NoError(t, err)

	assert.Equal(t, want, got, "the order of the answers must not reach the places")
}

func TestWikimedia_PlacesNear_ReportsTheEarliestBatchThatFailed(t *testing.T) {
	// Guard: with batches running together, the error that comes back is the earliest batch's,
	// not whichever request happened to fail first.
	fixtures := fakeWikimedia(wikiTownPages(320, taviraLat, taviraLon, 9))

	api, _ := newStubWikimediaAPI(func(req *http.Request) stubResponse {
		ids := strings.Split(req.URL.Query().Get("pageids"), "|")

		switch {
		case slices.Contains(ids, "1000"): // the first batch of pages, slow to fail
			time.Sleep(30 * time.Millisecond)

			return stubResponse{http.StatusServiceUnavailable, "the first batch is unwell"}
		case slices.Contains(ids, "1300"): // the last batch of pages, quick to fail
			return stubResponse{http.StatusInternalServerError, "the last batch is unwell"}
		}

		return fixtures(req)
	})

	places, err := api.PlacesNear(context.Background(), taviraLat, taviraLon)

	assert.Nil(t, places)
	assert.ErrorContains(t, err, "503", "the earliest batch's failure is the one that is reported")
}

// lateFirst answers the requests that were sent last the soonest, so batches finish in roughly
// the opposite order to the one they were sent in.
func lateFirst(respond func(*http.Request) stubResponse) func(*http.Request) stubResponse {
	var sent atomic.Int64

	return func(req *http.Request) stubResponse {
		nth := int(sent.Add(1)) - 1
		time.Sleep(time.Duration(wikimediaConcurrency-nth%wikimediaConcurrency) * 5 * time.Millisecond)

		return respond(req)
	}
}

func TestWikimedia_SendsTheUserAgent(t *testing.T) {
	api, requests := newStubWikimediaAPI(taviraFixtures(t))

	_, err := api.PlacesNear(context.Background(), taviraLat, taviraLon)
	require.NoError(t, err)
	require.NotEmpty(t, *requests)

	for _, req := range *requests {
		assert.Equal(t, UserAgent, req.Header.Get("User-Agent"), req.URL.String())
	}
}

func TestWikimedia_ErrorIncludesTheStatusOnNon200(t *testing.T) {
	api, _ := newStubWikimediaAPI(func(*http.Request) stubResponse {
		return stubResponse{http.StatusServiceUnavailable, `upstream connect error`}
	})

	places, err := api.PlacesNear(context.Background(), taviraLat, taviraLon)

	assert.Nil(t, places)
	assert.ErrorContains(t, err, "503")
}

func TestWikimedia_StopsWhenTheContextIsCancelled(t *testing.T) {
	api, requests := newStubWikimediaAPI(taviraFixtures(t))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	places, err := api.PlacesNear(ctx, taviraLat, taviraLon)

	assert.Nil(t, places)
	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, *requests, "a cancelled context stops the requests")
}

func TestWikimedia_GetEntities_ReturnsTheRequestedEntities(t *testing.T) {
	api, _ := newStubWikimediaAPI(taviraFixtures(t))

	entities, err := api.GetEntities(context.Background(), []string{"Q5049832", "Q3505418"}, "claims|labels")
	require.NoError(t, err)

	castle, ok := entities["Q5049832"]
	require.True(t, ok, "the castle entity should be returned")
	assert.Equal(t, "Castle of Tavira", castle.Labels["en"].Value)
	assert.Equal(t, []string{"Q23413", "Q210272"}, castle.ItemIDs("P31"))
	assert.Equal(t, "TaviraCastle-CCBY.jpg", castle.StringValue("P18"))

	church, ok := entities["Q3505418"]
	require.True(t, ok, "the church entity should be returned")
	assert.Equal(t, []string{"Q16970", "Q210272"}, church.ItemIDs("P31"))
}

func TestWikimedia_NewWithoutAClientUsesA30SecondTimeout(t *testing.T) {
	api := NewWikimediaAPI(nil)

	require.NotNil(t, api.client)
	assert.Equal(t, 30*time.Second, api.client.Timeout)
}

func TestWikimediaLive_FindsBelemTowerInLisbon(t *testing.T) {
	if os.Getenv("LIVE_API_TESTS") != "1" {
		t.Skip("set LIVE_API_TESTS=1 to call the real Wikimedia APIs")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	places, err := NewWikimediaAPI(nil).PlacesNear(ctx, 38.7223, -9.1393)
	require.NoError(t, err)
	require.NotEmpty(t, places)

	tower := findPlace(places, "Q215003")
	require.NotNil(t, tower, "Belém Tower should be among the places near Lisbon")
	assert.Equal(t, "Belém Tower", tower.Name)
	assert.NotEmpty(t, tower.Types)
	assert.NotEmpty(t, tower.Image)
	assert.Greater(t, tower.Sitelinks, 10)
}
