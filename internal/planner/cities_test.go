package planner_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"weatherservice/internal/adapters/api"
	"weatherservice/internal/planner"
)

// These are the one-pager's "Done when" list written as tests. Each one plans a real city with
// the real API clients, over a transport that answers from planner/testdata/cities/<city>/, so a
// run never touches the network. RECORD_FIXTURES=1 refreshes those files from the live APIs.

// planNow is the moment every city is planned at, and planStart the first day of every trip.
// Both are fixed, so the plans never change with the calendar.
var (
	planNow   = time.Date(2026, 9, 20, 9, 0, 0, 0, time.UTC)
	planStart = time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
)

// rainyDay is the day the hand-edited forecasts put 5 mm or more of rain on: two days after
// planNow, which is also the third day of a trip that starts on planStart.
const rainyDay = 2

// city is one of the three places the one-pager names, with the coordinates the validation run
// used. stayLat/stayLon is the plan center its places cluster around; the stay fixtures are
// recorded there, because that is where Build looks for somewhere to stay.
type city struct {
	dir              string
	name             string
	country          string
	lat, lon         float64
	stayLat, stayLon float64
}

var (
	lisbon = city{
		dir: "lisbon", name: "Lisbon", country: "Portugal",
		lat: 38.7223, lon: -9.1393,
		stayLat: 38.7139, stayLon: -9.1334,
	}
	tavira = city{
		dir: "tavira", name: "Tavira", country: "Portugal",
		lat: 37.1264, lon: -7.6506,
		stayLat: 37.1264, stayLon: -7.6506,
	}
	kyoto = city{
		dir: "kyoto", name: "Kyoto", country: "Japan",
		lat: 35.0116, lon: 135.7681,
		stayLat: 35.0116, stayLon: 135.7681,
	}
)

func TestCities_LisbonIncludesBelemTowerAndJeronimos(t *testing.T) {
	plan := cityPlan(t, lisbon, 3)

	stops := stopNames(plan)
	assert.Contains(t, stops, "Belém Tower", "the most famous place in Lisbon belongs in a 3-day plan")
	assert.Contains(t, stops, "Jerónimos Monastery", "the second most famous place in Lisbon belongs in a 3-day plan")
}

func TestCities_LisbonSkipsBridgesAndTheNationalLibrary(t *testing.T) {
	// Road bridges and the National Library rank high on Wikipedia coverage but are not stops.
	unwanted := map[string]string{
		"Q721152": "25 de Abril Bridge",
		"Q233737": "Vasco da Gama Bridge",
		"Q245966": "the National Library of Portugal",
	}

	candidates := cityCandidates(t, lisbon)
	for id, what := range unwanted {
		require.Contains(t, placeIDs(candidates), id, "the fixture must offer %s, or this test proves nothing", what)
	}

	plan := cityPlan(t, lisbon, 3)
	ranked := planner.Rank(candidates, planner.PlaceTypes, planner.Boosts)
	require.Len(t, plan.Days, 3, "filtering out unsuitable places must still leave a full Lisbon trip")
	require.Contains(t, stopNames(plan), "Belém Tower", "the type filter must keep Lisbon's landmarks")
	top := ranked
	if len(top) > 15 {
		top = top[:15] // the validation run judged each city by its top 15
	}

	for id, what := range unwanted {
		assert.NotContains(t, placeIDs(top), id, "%s should not be among Lisbon's 15 most famous places to visit", what)
		assert.NotContains(t, stopIDs(plan), id, "%s should not be a stop on a Lisbon plan", what)
	}
}

func TestCities_LisbonRainyDayIsMostlyIndoor(t *testing.T) {
	plan := cityPlan(t, lisbon, 3)

	require.Len(t, plan.Days, 3)
	day := plan.Days[rainyDay]
	require.True(t, day.Rainy, "the recorded forecast puts 5 mm or more of rain on %s", day.Date.Format(time.DateOnly))

	indoor := shelteredStops(day)
	assert.Greater(t, indoor, len(day.Stops)-indoor,
		"a rainy day should be mostly indoors, got %v", kindsOf(day))
}

func TestCities_LisbonHasAtMostSixNamedStays(t *testing.T) {
	plan := cityPlan(t, lisbon, 3)
	require.Len(t, plan.Days, 3, "places to stay must be centered on a full Lisbon trip")
	require.Contains(t, stopNames(plan), "Belém Tower", "hotels alone must not satisfy city acceptance")

	assert.Empty(t, plan.StaysNote, "Lisbon has plenty of places to stay, so the card is shown")
	assert.NotEmpty(t, plan.Stays, "OpenStreetMap knows hundreds of hotels around the middle of Lisbon")
	assert.LessOrEqual(t, len(plan.Stays), planner.MaxStays)

	for _, stay := range plan.Stays {
		assert.NotEmpty(t, stay.Name, "a place to stay with no name has nothing to show")
		assert.LessOrEqual(t, planner.DistanceKM(plan.Center, planner.Place{Lat: stay.Lat, Lon: stay.Lon}),
			planner.StayRadiusM/1000.0, "%s should be within walking distance of the plan center", stay.Name)
	}
}

func TestCities_TaviraFiveDaysGetsFewerDaysAndANote(t *testing.T) {
	plan := cityPlan(t, tavira, 5)

	assert.NotEmpty(t, plan.Days)
	assert.Less(t, len(plan.Days), 5, "a small town does not have five days of famous places")
	assert.Equal(t, shortTripNote("Tavira", len(plan.Days)), plan.Note,
		"the plan should say why it is shorter than the trip that was asked for")
}

func TestCities_KyotoRainyDayHasThreeIndoorStops(t *testing.T) {
	plan := cityPlan(t, kyoto, 3)

	require.Len(t, plan.Days, 3)
	day := plan.Days[rainyDay]
	require.True(t, day.Rainy, "the recorded forecast puts 5 mm or more of rain on %s", day.Date.Format(time.DateOnly))

	assert.GreaterOrEqual(t, shelteredStops(day), planner.MinStopsPerDay,
		"Kyoto's famous places are nearly all outdoors, so a rainy day has to reach further down the list, got %v", kindsOf(day))
}

func TestCities_KyotoUsesNearbyHighlightsInsteadOfASingleStopDay(t *testing.T) {
	plan := cityPlan(t, kyoto, 3)
	require.Len(t, plan.Days, 3)
	seen := map[string]bool{}
	for _, day := range plan.Days {
		assert.GreaterOrEqual(t, len(day.Stops), 2, "a nearby unused highlight can fill the Arashiyama day")
		for _, stop := range day.Stops {
			assert.False(t, seen[stop.ID], "%s is repeated", stop.Name)
			seen[stop.ID] = true
		}
	}
	assert.Contains(t, stopIDs(plan), "Q23579173", "the bamboo grove is next to Arashiyama")
}

func TestCities_KyotoHasNoWards(t *testing.T) {
	// A ward is a chunk of the city, not a place to visit. Fushimi Inari and Fushimi Castle are,
	// so nothing here may filter on the name alone.
	wards := map[string]string{
		"Q1197771": "Fushimi-ku",
		"Q1203862": "Higashiyama-ku",
	}

	candidates := cityCandidates(t, kyoto)
	for id, what := range wards {
		require.Contains(t, placeIDs(candidates), id, "the fixture must offer %s, or this test proves nothing", what)
	}

	plan := cityPlan(t, kyoto, 3)
	ranked := planner.Rank(candidates, planner.PlaceTypes, planner.Boosts)
	require.Len(t, plan.Days, 3, "filtering out wards must still leave a full Kyoto trip")
	require.NotEmpty(t, stopIDs(plan), "a plan with no stops must not pass the ward exclusion check")
	require.Contains(t, placeIDs(ranked), "Q714828", "Fushimi Inari-taisha is a landmark, despite sharing the ward's name")

	for id, what := range wards {
		assert.NotContains(t, placeIDs(ranked), id, "%s is a ward of Kyoto, not a place to visit", what)
		assert.NotContains(t, stopIDs(plan), id, "%s should not be a stop on a Kyoto plan", what)
	}
}

func TestCities_KyotoOnlyStrandsAnIsolatedStop(t *testing.T) {
	// A singleton may not ignore a nearby stop already assigned to another day. The separate
	// nearby-highlights regression also checks candidates omitted by compact selection.
	for _, days := range []int{3, 4} {
		t.Run(fmt.Sprintf("%d_days", days), func(t *testing.T) {
			plan := cityPlan(t, kyoto, days)
			require.Len(t, plan.Days, days)
			require.NotEmpty(t, stopIDs(plan), "an empty plan must not satisfy this test")

			for i, day := range plan.Days {
				require.NotEmpty(t, day.Stops, "day %d has no stops at all", i+1)
				if len(day.Stops) >= 2 {
					continue
				}

				nearest := nearestStopOnAnotherDayKM(plan, i)
				assert.Greater(t, nearest, planner.PullReachKM,
					"day %d strands %s, yet another day has a stop %.1f km away",
					i+1, day.Stops[0].Name, nearest)
			}
		})
	}
}

// nearestStopOnAnotherDayKM measures from the lone stop of day i to the closest stop on any
// other day, which is what the balancing rule reaches for.
func nearestStopOnAnotherDayKM(plan *planner.Plan, i int) float64 {
	nearest := math.Inf(1)
	for j, other := range plan.Days {
		if j == i {
			continue
		}
		for _, stop := range other.Stops {
			if d := planner.DistanceKM(plan.Days[i].Stops[0], stop); d < nearest {
				nearest = d
			}
		}
	}

	return nearest
}

// cityPlan plans a trip the way the app does, with the real clients over the recorded responses.
func cityPlan(t *testing.T, c city, days int) *planner.Plan {
	t.Helper()

	places, forecast, stays := cityClients(t, c)
	plan, err := planner.Build(context.Background(), planner.Request{
		City:    c.name,
		Country: c.country,
		Lat:     c.lat,
		Lon:     c.lon,
		Start:   planStart,
		Days:    days,
	}, places, forecast, stays, planNow)
	require.NoError(t, err)
	require.NotNil(t, plan)

	return plan
}

// cityCandidates is everything the place source found near the city, before any filtering. The
// tests that check a place is left out use it to prove the fixture really offers that place.
func cityCandidates(t *testing.T, c city) []planner.Place {
	t.Helper()

	places, _, _ := cityClients(t, c)
	found, err := places.PlacesNear(context.Background(), c.lat, c.lon)
	require.NoError(t, err)
	require.NotEmpty(t, found)

	return found
}

// cityClients builds the real API clients over this city's recorded responses. With
// RECORD_FIXTURES=1 it first refreshes those responses from the live APIs.
func cityClients(t *testing.T, c city) (planner.PlaceSource, planner.ForecastSource, planner.StaySource) {
	t.Helper()

	if os.Getenv("RECORD_FIXTURES") == "1" {
		recordOnce(c.dir).Do(func() { recordCity(t, c) })
	}

	client := &http.Client{Transport: &fixtureTransport{dir: c.fixtureDir(), names: newFixtureNames()}}

	return api.NewWikimediaAPI(client), api.NewOpenMeteoForecastAPI(client), api.NewOverpassAPI(client, nil)
}

func (c city) fixtureDir() string { return filepath.Join("testdata", "cities", c.dir) }

// recorded keeps each city to one recording per run, however many tests plan it.
var (
	recordedMu sync.Mutex
	recorded   = map[string]*sync.Once{}
)

func recordOnce(city string) *sync.Once {
	recordedMu.Lock()
	defer recordedMu.Unlock()

	if recorded[city] == nil {
		recorded[city] = &sync.Once{}
	}

	return recorded[city]
}

// recordCity asks the live APIs everything a plan needs and saves their answers as fixtures. It
// calls the sources straight, not through Build, so recording works even when nothing survives
// ranking. Recorded files still have to be trimmed by hand, and the forecasts hand-edited to put
// rain on the third day.
func recordCity(t *testing.T, c city) {
	t.Helper()

	dir := c.fixtureDir()
	require.NoError(t, os.MkdirAll(dir, 0o755))

	client := &http.Client{
		Timeout:   2 * time.Minute,
		Transport: &recordTransport{dir: dir, names: newFixtureNames()},
	}
	ctx := context.Background()

	places, err := api.NewWikimediaAPI(client).PlacesNear(ctx, c.lat, c.lon)
	require.NoError(t, err, "recording places near %s", c.name)

	forecast, err := api.NewOpenMeteoForecastAPI(client).DailyForecast(ctx, c.lat, c.lon)
	require.NoError(t, err, "recording the forecast for %s", c.name)

	stays, err := api.NewOverpassAPI(client, nil).StaysNear(ctx, c.stayLat, c.stayLon)
	require.NoError(t, err, "recording places to stay near %s", c.name)

	t.Logf("recorded %s in %s: %d places, %d forecast days, %d places to stay",
		c.name, dir, len(places), len(forecast.Days), len(stays))
}

// fixtureTransport answers from disk, in the order the clients ask: the first geosearch comes
// from geosearch_1.json, the second from geosearch_2.json, and so on.
type fixtureTransport struct {
	dir   string
	names *fixtureNames
}

func (f *fixtureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	name := f.names.next(requestKind(req))

	body, err := os.ReadFile(filepath.Join(f.dir, name))
	if err != nil {
		return nil, fmt.Errorf("no recorded response %s for %s %s: %w", name, req.Method, req.URL.Host, err)
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
		Request:    req,
	}, nil
}

// recordTransport sends the request for real and keeps a copy of every successful answer.
type recordTransport struct {
	dir   string
	names *fixtureNames
}

func (r *recordTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", req.URL.Host, err)
	}

	// A failed attempt is retried by the client, so only answers that count become fixtures.
	if resp.StatusCode == http.StatusOK {
		name := r.names.next(requestKind(req))
		if err := os.WriteFile(filepath.Join(r.dir, name), body, 0o644); err != nil {
			return nil, fmt.Errorf("saving %s: %w", name, err)
		}
	}

	resp.Body = io.NopCloser(bytes.NewReader(body))

	return resp, nil
}

// requestKind names the fixture a request is answered from, one name per kind of call the three
// clients make.
func requestKind(req *http.Request) string {
	query := req.URL.Query()

	switch {
	case strings.Contains(req.URL.Host, "open-meteo"):
		return "forecast"
	case req.Method == http.MethodPost:
		return "stays" // Overpass is the only client that posts
	case query.Get("list") == "geosearch":
		return "geosearch"
	case query.Get("prop") == "pageprops":
		return "pageprops"
	case query.Get("props") == "sitelinks":
		return "sitelinks"
	case query.Get("action") == "wbgetentities":
		return "entities"
	}

	return "unknown"
}

// fixtureNames counts the requests of each kind, so the nth call reads or writes the nth file.
type fixtureNames struct {
	mu   sync.Mutex
	seen map[string]int
}

func newFixtureNames() *fixtureNames { return &fixtureNames{seen: map[string]int{}} }

func (n *fixtureNames) next(kind string) string {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.seen[kind]++

	return fmt.Sprintf("%s_%d.json", kind, n.seen[kind])
}

// shortTripNote is the note Build writes when a city runs out of highlights before the trip ends.
func shortTripNote(city string, days int) string {
	unit := "days"
	if days == 1 {
		unit = "day"
	}

	return fmt.Sprintf("%s has enough highlights for %d %s, so here's a %d-day plan.", city, days, unit, days)
}

func stopNames(plan *planner.Plan) []string {
	var names []string
	for _, day := range plan.Days {
		for _, stop := range day.Stops {
			names = append(names, stop.Name)
		}
	}

	return names
}

func stopIDs(plan *planner.Plan) []string {
	var ids []string
	for _, day := range plan.Days {
		ids = append(ids, placeIDs(day.Stops)...)
	}

	return ids
}

func placeIDs(places []planner.Place) []string {
	ids := make([]string, len(places))
	for i, place := range places {
		ids[i] = place.ID
	}

	return ids
}

// shelteredStops counts the stops of a day that keep a traveller out of the rain.
func shelteredStops(day planner.Day) int {
	count := 0

	for _, stop := range day.Stops {
		if stop.Kind == planner.Indoor || stop.Kind == planner.Mixed {
			count++
		}
	}

	return count
}

// kindsOf describes a day for a failure message: "Belém Tower (outdoor), ...".
func kindsOf(day planner.Day) string {
	parts := make([]string, 0, len(day.Stops))
	for _, stop := range day.Stops {
		parts = append(parts, fmt.Sprintf("%s (%s)", stop.Name, stop.Kind))
	}

	return strings.Join(parts, ", ")
}
