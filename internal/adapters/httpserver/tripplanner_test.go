package httpserver

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
	"weatherservice/internal/adapters/sqlite"
	"weatherservice/internal/application"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"weatherservice/internal/planner"
)

// The fakes below use boosted Wikidata IDs, because a boosted place skips the type filter.
// That keeps these tests independent of the generated planner.PlaceTypes list.
const (
	boostedIndoorA = "Q652806"   // indoor
	boostedIndoorB = "Q2063403"  // indoor
	boostedIndoorC = "Q11650434" // indoor
	boostedOutdoor = "Q168001"   // outdoor
)

type fakePlaceSource struct {
	places []planner.Place
	err    error
}

func (f *fakePlaceSource) PlacesNear(_ context.Context, _, _ float64) ([]planner.Place, error) {
	return f.places, f.err
}

type fakeForecastSource struct {
	forecast *planner.Forecast
	err      error
}

func (f *fakeForecastSource) DailyForecast(_ context.Context, _, _ float64) (*planner.Forecast, error) {
	return f.forecast, f.err
}

type fakeStaySource struct {
	stays []planner.Stay
	err   error
}

func (f *fakeStaySource) StaysNear(_ context.Context, _, _ float64) ([]planner.Stay, error) {
	return f.stays, f.err
}

// tomorrowUTC is the first day the route accepts, on the same clock the real planner uses.
func tomorrowUTC() time.Time {
	now := time.Now().UTC().AddDate(0, 0, 1)

	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// sourcePlaces are four Lisbon places a few hundred metres apart: three indoor and one
// outdoor, so a single rainy day is already sheltered enough and keeps all four stops.
func sourcePlaces() []planner.Place {
	return []planner.Place{
		{ID: boostedIndoorA, Name: "Tile Museum", Lat: 38.7223, Lon: -9.1393, Sitelinks: 120, Image: "Museu do Azulejo.jpg"},
		{ID: boostedIndoorB, Name: "Fado Museum", Lat: 38.7240, Lon: -9.1370, Sitelinks: 110},
		{ID: boostedIndoorC, Name: "Coach Museum", Lat: 38.7205, Lon: -9.1410, Sitelinks: 100},
		{ID: boostedOutdoor, Name: "Castle of Sao Jorge", Lat: 38.7250, Lon: -9.1420, Sitelinks: 90},
	}
}

func sourcePlaceNames() []string {
	names := make([]string, 0, len(sourcePlaces()))
	for _, p := range sourcePlaces() {
		names = append(names, p.Name)
	}

	return names
}

// sourceStays sit next to the places above, well inside the stay radius.
func sourceStays() []planner.Stay {
	return []planner.Stay{
		{Name: "Hotel Baixa", Kind: "hotel", Lat: 38.7225, Lon: -9.1395, Website: "https://hotel-baixa.example", Stars: 4},
		{Name: "Alfama Hostel", Kind: "hostel", Lat: 38.7230, Lon: -9.1380},
	}
}

// wetForecast makes the given date a rainy one.
func wetForecast(date time.Time, mm float64) *planner.Forecast {
	return &planner.Forecast{
		Timezone: "UTC",
		Days:     []planner.DayForecast{{Date: date, RainMM: rainMM(mm)}},
	}
}

// realTripPlanner is the adapter under test, fed by fakes instead of the network.
func realTripPlanner(start time.Time) application.TripPlanner {
	return application.NewTripPlanner(
		&fakePlaceSource{places: sourcePlaces()},
		&fakeForecastSource{forecast: wetForecast(start, 9.2)},
		&fakeStaySource{stays: sourceStays()},
	)
}

func stopNames(day planner.Day) []string {
	names := make([]string, 0, len(day.Stops))
	for _, stop := range day.Stops {
		names = append(names, stop.Name)
	}

	return names
}

func TestTripPlanner_PlansFromTheGivenSources(t *testing.T) {
	start := tomorrowUTC()
	req := planner.Request{City: "Lisbon", Country: "Portugal", Lat: 38.7223, Lon: -9.1393, Start: start, Days: 1}

	plan, err := realTripPlanner(start).Plan(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.Equal(t, req, plan.Request)

	require.Len(t, plan.Days, 1)
	assert.ElementsMatch(t, sourcePlaceNames(), stopNames(plan.Days[0]), "every place from the source is planned")
	assert.True(t, plan.Days[0].Rainy, "the forecast source decides the weather")

	require.Len(t, plan.Stays, 2)
	assert.Equal(t, "Hotel Baixa", plan.Stays[0].Name, "the stay with a website comes first")
	assert.Empty(t, plan.StaysNote)
	assert.Contains(t, plan.BookingURL, "www.booking.com")
}

func TestPlanRoute_WithTheRealTripPlanner(t *testing.T) {
	start := tomorrowUTC()
	server := NewAppServer(nil, nil, testStore)
	server.SetTripPlanner(realTripPlanner(start))

	status, body := doRequest(t, server, planRequest(map[string]string{
		"start": start.Format(tripDateLayout),
		"days":  "1",
	}))

	require.Equal(t, fiber.StatusOK, status)
	assert.NotContains(t, body, "could not plan your trip")

	assert.Contains(t, body, "Day 1 · "+start.Format(tripDateHeading))
	for _, name := range sourcePlaceNames() {
		assert.Contains(t, body, name)
	}

	assert.Contains(t, body, "Rainy day")
	assert.Contains(t, body, "9 mm")

	assert.Contains(t, body, "Hotel Baixa")
	assert.Contains(t, body, "Alfama Hostel")
	assert.Contains(t, body, "https://www.booking.com/searchresults.html")
}

// Guard: the full route must keep rendering the same plan when all three providers fail,
// both while entries are fresh and after every source has expired.
func TestPlanRoute_CachedPlanSurvivesAllSourcesOffline(t *testing.T) {
	start := tomorrowUTC()
	places := &fakePlaceSource{places: sourcePlaces()}
	forecast := &fakeForecastSource{forecast: wetForecast(start, 9.2)}
	stays := &fakeStaySource{stays: sourceStays()}
	path := filepath.Join(t.TempDir(), "offline.db")
	storage, err := sqlite.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = storage.Close() })
	inspect, err := sql.Open("sqlite3", path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = inspect.Close() })
	server := NewAppServer(nil, nil, storage)
	server.SetTripPlanner(application.NewTripPlanner(
		storage.NewCachedPlaceSource(places), storage.NewCachedForecastSource(forecast), storage.NewCachedStaySource(stays)))
	request := func() (int, string) {
		return doRequest(t, server, planRequest(map[string]string{
			"start": start.Format(tripDateLayout), "days": "1",
		}))
	}
	status, want := request()
	require.Equal(t, fiber.StatusOK, status)
	require.Contains(t, want, "Tile Museum")
	require.Contains(t, want, "Rainy day")
	require.Contains(t, want, "Hotel Baixa")

	offline := errors.New("all providers are offline")
	places.places, places.err = nil, offline
	forecast.forecast, forecast.err = nil, offline
	stays.stays, stays.err = nil, offline
	for _, expired := range []bool{false, true} {
		if expired {
			_, err := inspect.Exec(`UPDATE source_cache SET fetched_at = ?`, time.Now().Add(-31*24*time.Hour).Unix())
			require.NoError(t, err)
		}
		began := time.Now()
		status, got := request()
		elapsed := time.Since(began)
		require.Equal(t, fiber.StatusOK, status, "expired=%t", expired)
		assert.Equal(t, want, got, "expired=%t", expired)
		t.Logf("all sources offline, expired=%t: %s", expired, elapsed)
	}
}
