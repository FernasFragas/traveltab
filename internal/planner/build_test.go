package planner

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The Wikidata type IDs the Build tests plan with. They stand in for the generated list, so
// these tests don't depend on whatever PlaceTypes holds.
const (
	typeMuseum  = "T_museum"
	typePark    = "T_park"
	typeUnknown = "T_unknown"
)

// The zones the Build tests plan in. Lisbon sits on UTC in March, Tokyo nine hours ahead.
const (
	lisbonTZ = "Europe/Lisbon"
	tokyoTZ  = "Asia/Tokyo"
)

// buildNow is the moment every Build test plans at, and buildStart the trip's first day: two
// days later, well inside the trusted forecast window.
var (
	buildNow   = time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	buildStart = time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
)

// useTestPlaceTypes swaps the generated type list for a small one and puts the real one back
// afterwards. It is global state, so no test that calls it may run in parallel.
func useTestPlaceTypes(t *testing.T) {
	t.Helper()
	original := PlaceTypes
	PlaceTypes = map[string]Kind{typeMuseum: Indoor, typePark: Outdoor}
	t.Cleanup(func() { PlaceTypes = original })
}

// buildPlaces is a PlaceSource with canned results. It records where it was asked about.
type buildPlaces struct {
	places   []Place
	err      error
	lat, lon float64
}

func (s *buildPlaces) PlacesNear(_ context.Context, lat, lon float64) ([]Place, error) {
	s.lat, s.lon = lat, lon
	if s.err != nil {
		return nil, s.err
	}
	return s.places, nil
}

// buildForecast is a ForecastSource with a canned forecast or a canned failure.
type buildForecast struct {
	forecast *Forecast
	err      error
}

func (s *buildForecast) DailyForecast(_ context.Context, _, _ float64) (*Forecast, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.forecast, nil
}

// cancelForecast waits for Build to abandon the lookup when the essential place source fails.
type cancelForecast struct {
	canceled chan struct{}
}

func (s *cancelForecast) DailyForecast(ctx context.Context, _, _ float64) (*Forecast, error) {
	<-ctx.Done()
	close(s.canceled)
	return nil, ctx.Err()
}

// buildStays is a StaySource with canned results. It records where it was asked about.
type buildStays struct {
	stays    []Stay
	err      error
	lat, lon float64
}

func (s *buildStays) StaysNear(_ context.Context, lat, lon float64) ([]Stay, error) {
	s.lat, s.lon = lat, lon
	if s.err != nil {
		return nil, s.err
	}
	return s.stays, nil
}

// gateWait is how long a gated source waits for the other one to start before giving up. A Build
// that loads them one after another spends it once and then reports that it waited alone.
const gateWait = 2 * time.Second

// buildGate lets the place and the forecast source wait for each other. Each one says it has
// started and then waits for the other, so "they ran at the same time" is a fact each source
// records rather than a time the test measures.
type buildGate struct {
	placesStarted     chan struct{}
	forecastStarted   chan struct{}
	placesSawForecast atomic.Bool
	forecastSawPlaces atomic.Bool
}

func newBuildGate() *buildGate {
	return &buildGate{placesStarted: make(chan struct{}), forecastStarted: make(chan struct{})}
}

// arrive announces this source and waits for the other one, for at most gateWait.
func (g *buildGate) arrive(started chan struct{}, other <-chan struct{}, sawOther *atomic.Bool) {
	close(started)

	select {
	case <-other:
		sawOther.Store(true)
	case <-time.After(gateWait):
	}
}

// gatedPlaces answers like buildPlaces, but only once it has waited for the forecast source.
type gatedPlaces struct {
	gate   *buildGate
	places []Place
}

func (s *gatedPlaces) PlacesNear(_ context.Context, _, _ float64) ([]Place, error) {
	s.gate.arrive(s.gate.placesStarted, s.gate.forecastStarted, &s.gate.placesSawForecast)
	return s.places, nil
}

// gatedForecast answers like buildForecast, but only once it has waited for the place source.
type gatedForecast struct {
	gate     *buildGate
	forecast *Forecast
}

func (s *gatedForecast) DailyForecast(_ context.Context, _, _ float64) (*Forecast, error) {
	s.gate.arrive(s.gate.forecastStarted, s.gate.placesStarted, &s.gate.forecastSawPlaces)
	return s.forecast, nil
}

// buildCandidate is a place as a source returns it: typed, but with no Kind yet.
func buildCandidate(id, placeType string, lat, lon float64, fame int) Place {
	return Place{ID: id, Name: id + " Place", Lat: lat, Lon: lon, Sitelinks: fame, Types: []string{placeType}}
}

// buildCity is an eight-place city: four museums to the north and four parks about 11 km
// south, which is exactly two walking days.
func buildCity() []Place {
	places := make([]Place, 0, 8)
	for i := 0; i < 4; i++ {
		step := float64(i) * 0.002
		places = append(places,
			buildCandidate(fmt.Sprintf("Q10%d", i), typeMuseum, 38.81+step, -9.14+step, 200-i),
			buildCandidate(fmt.Sprintf("Q20%d", i), typePark, 38.71+step, -9.14+step, 150-i),
		)
	}
	return places
}

// buildSmallTown has four places in one small cluster, which is only enough for a single day.
func buildSmallTown() []Place {
	return []Place{
		buildCandidate("Q301", typeMuseum, 37.120, -7.650, 90),
		buildCandidate("Q302", typePark, 37.122, -7.652, 80),
		buildCandidate("Q303", typeMuseum, 37.124, -7.654, 70),
		buildCandidate("Q304", typePark, 37.126, -7.656, 60),
	}
}

// buildStarCity puts one place in the middle of four others, so the plan's center is that
// middle place whichever of the others end up in the day.
func buildStarCity() []Place {
	return []Place{
		buildCandidate("Q401", typeMuseum, 38.71, -9.14, 100),
		buildCandidate("Q402", typePark, 38.72, -9.14, 90),
		buildCandidate("Q403", typePark, 38.70, -9.14, 80),
		buildCandidate("Q404", typeMuseum, 38.71, -9.13, 70),
		buildCandidate("Q405", typeMuseum, 38.71, -9.15, 60),
	}
}

// buildRequest is the Lisbon trip the tests plan, with the number of days filled in.
func buildRequest(days int) Request {
	return Request{City: "Lisbon", Country: "Portugal", Lat: 38.76, Lon: -9.14, Start: buildStart, Days: days}
}

// cityForecast holds one day per rain value, starting at start, at midnight in the named zone.
func cityForecast(t *testing.T, zone string, start time.Time, rain ...float64) *Forecast {
	t.Helper()
	loc, err := time.LoadLocation(zone)
	require.NoError(t, err)
	fc := &Forecast{Timezone: zone}
	y, m, d := start.Date()
	for i, mm := range rain {
		value := mm
		fc.Days = append(fc.Days, DayForecast{Date: time.Date(y, m, d+i, 0, 0, 0, 0, loc), RainMM: &value})
	}
	return fc
}

func TestBuild_PlansTheRequestedNumberOfDays(t *testing.T) {
	useTestPlaceTypes(t)
	req := buildRequest(2)
	source := &buildPlaces{places: buildCity()}

	plan, err := Build(context.Background(), req, source,
		&buildForecast{forecast: cityForecast(t, lisbonTZ, req.Start, 0, 0)}, &buildStays{}, buildNow)

	require.NoError(t, err)
	require.NotNil(t, plan)
	require.Len(t, plan.Days, 2)
	assert.Equal(t, req, plan.Request)
	assert.Equal(t, req.Lat, source.lat, "places are looked up around the request")
	assert.Equal(t, req.Lon, source.lon)
	assert.Equal(t, "2026-03-10", plan.Days[0].Date.Format(time.DateOnly))
	assert.Equal(t, "2026-03-11", plan.Days[1].Date.Format(time.DateOnly))
	for i, day := range plan.Days {
		assert.GreaterOrEqual(t, len(day.Stops), MinStopsPerDay, "day %d", i)
		assert.LessOrEqual(t, len(day.Stops), MaxStopsPerDay, "day %d", i)
	}
	assert.Empty(t, plan.Note, "a full two-day city needs no note")
}

func TestBuild_RejectsZeroOrSixDays(t *testing.T) {
	useTestPlaceTypes(t)
	for _, days := range []int{0, -1, MaxDays + 1} {
		plan, err := Build(context.Background(), buildRequest(days), &buildPlaces{places: buildCity()},
			&buildForecast{forecast: cityForecast(t, lisbonTZ, buildStart, 0, 0)}, &buildStays{}, buildNow)

		require.Error(t, err, "days=%d", days)
		assert.Nil(t, plan, "days=%d", days)
	}
}

func TestBuild_FailsWhenPlacesCannotBeLoaded(t *testing.T) {
	useTestPlaceTypes(t)
	wikimediaDown := errors.New("wikimedia is down")

	plan, err := Build(context.Background(), buildRequest(2), &buildPlaces{err: wikimediaDown},
		&buildForecast{forecast: cityForecast(t, lisbonTZ, buildStart, 0, 0)}, &buildStays{}, buildNow)

	require.Error(t, err)
	assert.ErrorIs(t, err, wikimediaDown, "the source error is wrapped, not swallowed")
	assert.Nil(t, plan)
}

func TestBuild_FailsWhenNoPlaceSurvivesRanking(t *testing.T) {
	useTestPlaceTypes(t)
	untyped := []Place{
		buildCandidate("Q901", typeUnknown, 38.71, -9.14, 100),
		buildCandidate("Q902", typeUnknown, 38.72, -9.15, 90),
		buildCandidate("Q903", typeUnknown, 38.73, -9.16, 80),
	}

	plan, err := Build(context.Background(), buildRequest(2), &buildPlaces{places: untyped},
		&buildForecast{forecast: cityForecast(t, lisbonTZ, buildStart, 0, 0)}, &buildStays{}, buildNow)

	require.Error(t, err)
	assert.Nil(t, plan)
}

func TestBuild_SmallTownNoteNamesTheCity(t *testing.T) {
	useTestPlaceTypes(t)
	req := Request{City: "Tavira", Country: "Portugal", Lat: 37.12, Lon: -7.65, Start: buildStart, Days: 3}

	plan, err := Build(context.Background(), req, &buildPlaces{places: buildSmallTown()},
		&buildForecast{forecast: cityForecast(t, lisbonTZ, req.Start, 0, 0, 0)}, &buildStays{}, buildNow)

	require.NoError(t, err)
	require.NotNil(t, plan)
	require.Len(t, plan.Days, 1, "four places only fill one day")
	assert.Equal(t, "Tavira has enough highlights for 1 day, so here's a 1-day plan.", plan.Note)
}

func TestBuild_PlansWithoutWeatherWhenTheForecastFails(t *testing.T) {
	useTestPlaceTypes(t)

	plan, err := Build(context.Background(), buildRequest(2), &buildPlaces{places: buildCity()},
		&buildForecast{err: errors.New("open-meteo is down")}, &buildStays{}, buildNow)

	require.NoError(t, err, "a missing forecast still gives a plan")
	require.NotNil(t, plan)
	require.Len(t, plan.Days, 2)
	for i, day := range plan.Days {
		assert.Nil(t, day.Forecast, "day %d", i)
		assert.False(t, day.Rainy, "day %d", i)
		assert.NotEmpty(t, day.Stops, "day %d", i)
	}
	assert.Equal(t, "Weather is unavailable right now, so the days are not ordered by the forecast.", plan.Note)
}

func TestBuild_SaysWeatherIsUnavailableWhenTheForecastFails(t *testing.T) {
	useTestPlaceTypes(t)

	plan, err := Build(context.Background(), buildRequest(2), &buildPlaces{places: buildCity()},
		&buildForecast{err: errors.New("open-meteo is down")}, &buildStays{}, buildNow)

	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Equal(t, "Weather is unavailable right now, so the days are not ordered by the forecast.", plan.Note,
		"a plan with no weather should say so, instead of leaving the page to guess")
}

func TestBuild_KeepsTheOtherNotesWhenTheWeatherIsMissing(t *testing.T) {
	useTestPlaceTypes(t)
	req := Request{City: "Tavira", Country: "Portugal", Lat: 37.12, Lon: -7.65, Start: buildStart, Days: 3}

	plan, err := Build(context.Background(), req, &buildPlaces{places: buildSmallTown()},
		&buildForecast{err: errors.New("open-meteo is down")}, &buildStays{}, buildNow)

	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Equal(t,
		"Tavira has enough highlights for 1 day, so here's a 1-day plan. "+
			"Weather is unavailable right now, so the days are not ordered by the forecast.",
		plan.Note, "a short trip with no weather says both things, in that order")
}

func TestBuild_LoadsPlacesAndTheForecastAtTheSameTime(t *testing.T) {
	useTestPlaceTypes(t)
	req := buildRequest(2)
	gate := newBuildGate()

	plan, err := Build(context.Background(), req, &gatedPlaces{gate: gate, places: buildCity()},
		&gatedForecast{gate: gate, forecast: cityForecast(t, lisbonTZ, req.Start, 0, 0)}, &buildStays{}, buildNow)

	require.NoError(t, err)
	require.NotNil(t, plan)
	require.Len(t, plan.Days, 2, "waiting for both sources still gives the whole plan")
	assert.True(t, gate.placesSawForecast.Load(), "the forecast should already be on its way while the places load")
	assert.True(t, gate.forecastSawPlaces.Load(), "the places should already be on their way while the forecast loads")
}

func TestBuild_CancelsTheForecastWhenPlacesFail(t *testing.T) {
	forecast := &cancelForecast{canceled: make(chan struct{})}
	sourceErr := errors.New("wikimedia is down")

	plan, err := Build(context.Background(), buildRequest(2), &buildPlaces{err: sourceErr},
		forecast, &buildStays{}, buildNow)

	assert.Nil(t, plan)
	assert.ErrorIs(t, err, sourceErr)
	select {
	case <-forecast.canceled:
	case <-time.After(gateWait):
		t.Fatal("the parallel forecast lookup was not canceled after places failed")
	}
}

func TestBuild_SaysTheForecastComesLaterForFarTrips(t *testing.T) {
	useTestPlaceTypes(t)
	far := buildRequest(2)
	far.Start = buildNow.AddDate(0, 0, ForecastMaxDays+1)
	edge := buildRequest(2)
	edge.Start = buildNow.AddDate(0, 0, ForecastMaxDays)
	emptyForecast := func() *buildForecast { return &buildForecast{forecast: &Forecast{Timezone: lisbonTZ}} }

	farPlan, err := Build(context.Background(), far, &buildPlaces{places: buildCity()}, emptyForecast(), &buildStays{}, buildNow)
	require.NoError(t, err)
	require.NotNil(t, farPlan)
	edgePlan, err := Build(context.Background(), edge, &buildPlaces{places: buildCity()}, emptyForecast(), &buildStays{}, buildNow)
	require.NoError(t, err)
	require.NotNil(t, edgePlan)

	assert.Equal(t, "Forecast appears closer to your trip", farPlan.Note)
	assert.Empty(t, edgePlan.Note, "sixteen days out is still inside the forecast window")
}

func TestBuild_StillPlansWhenStaysFail(t *testing.T) {
	useTestPlaceTypes(t)
	req := buildRequest(2)

	plan, err := Build(context.Background(), req, &buildPlaces{places: buildCity()},
		&buildForecast{forecast: cityForecast(t, lisbonTZ, req.Start, 0, 0)},
		&buildStays{err: errors.New("overpass is down")}, buildNow)

	require.NoError(t, err, "a missing stay list still gives a plan")
	require.NotNil(t, plan)
	assert.Len(t, plan.Days, 2)
	assert.Equal(t, "Places to stay are unavailable right now", plan.StaysNote)
	assert.Empty(t, plan.Stays)
	assert.NotEmpty(t, plan.BookingURL, "the booking link still works without stays")
}

func TestBuild_PicksStaysAroundThePlanCenter(t *testing.T) {
	useTestPlaceTypes(t)
	req := buildRequest(1) // the request sits about 5 km north of where the stops turn out to be
	near := Stay{Name: "Near Hotel", Kind: "hotel", Lat: 38.7145, Lon: -9.14}
	far := Stay{Name: "Far Hotel", Kind: "hotel", Lat: 38.73, Lon: -9.14}
	source := &buildStays{stays: []Stay{far, near}}

	plan, err := Build(context.Background(), req, &buildPlaces{places: buildStarCity()},
		&buildForecast{forecast: cityForecast(t, lisbonTZ, req.Start, 0)}, source, buildNow)

	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Equal(t, "Q401", plan.Center.ID, "the center is the stop closest to all the others")
	assert.Equal(t, plan.Center.Lat, source.lat, "stays are looked up around the center, not the request")
	assert.Equal(t, plan.Center.Lon, source.lon)
	assert.Equal(t, []string{"Near Hotel"}, stayNames(plan.Stays))
}

func TestBuild_SetsTheBookingLink(t *testing.T) {
	useTestPlaceTypes(t)
	req := buildRequest(2)

	plan, err := Build(context.Background(), req, &buildPlaces{places: buildCity()},
		&buildForecast{forecast: cityForecast(t, lisbonTZ, req.Start, 0, 0)}, &buildStays{}, buildNow)

	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Equal(t,
		"https://www.booking.com/searchresults.html?ss=Lisbon%2C+Portugal&checkin=2026-03-10&checkout=2026-03-12",
		plan.BookingURL)
}

func TestBuild_TodayIsTakenInTheCityTimeZone(t *testing.T) {
	useTestPlaceTypes(t)
	// 23:30 UTC on 1 March is already 2 March in Tokyo, which pulls an 8 March trip inside the
	// seven days the forecast is trusted for. In Lisbon it is still 1 March, so the same trip is
	// one day too far out.
	now := time.Date(2026, 3, 1, 23, 30, 0, 0, time.UTC)
	req := Request{City: "Tokyo", Country: "Japan", Lat: 35.68, Lon: 139.76, Start: time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC), Days: 1}

	tokyo, err := Build(context.Background(), req, &buildPlaces{places: buildCity()},
		&buildForecast{forecast: cityForecast(t, tokyoTZ, req.Start, 0)}, &buildStays{}, now)
	require.NoError(t, err)
	require.NotNil(t, tokyo)
	lisbon, err := Build(context.Background(), req, &buildPlaces{places: buildCity()},
		&buildForecast{forecast: cityForecast(t, lisbonTZ, req.Start, 0)}, &buildStays{}, now)
	require.NoError(t, err)
	require.NotNil(t, lisbon)

	require.Len(t, tokyo.Days, 1)
	require.Len(t, lisbon.Days, 1)
	assert.Equal(t, tokyoTZ, tokyo.Days[0].Date.Location().String())
	assert.True(t, tokyo.Days[0].Certain, "it is already 2 March in Tokyo, so 8 March is six days out")
	assert.False(t, lisbon.Days[0].Certain, "it is still 1 March in Lisbon, so 8 March is seven days out")
}

func TestBuild_SameInputGivesTheSamePlan(t *testing.T) {
	useTestPlaceTypes(t)
	req := buildRequest(2)
	plans := make([]*Plan, 2)
	for i := range plans {
		plan, err := Build(context.Background(), req, &buildPlaces{places: buildCity()},
			&buildForecast{forecast: cityForecast(t, lisbonTZ, req.Start, 0, 9)},
			&buildStays{stays: []Stay{{Name: "Near Hotel", Kind: "hotel", Lat: 38.81, Lon: -9.14}}}, buildNow)
		require.NoError(t, err)
		require.NotNil(t, plan)
		plans[i] = plan
	}

	assert.Equal(t, plans[0], plans[1])
}
