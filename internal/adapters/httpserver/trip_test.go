package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
	"weatherservice/internal/application"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"weatherservice/internal/planner"
)

// planNow is the clock every planner test runs on, so "tomorrow" never moves.
var planNow = time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC)

// fakeTripPlanner stands in for the real planner: it builds a plan from the request, so the
// tests can assert on the rendered HTML instead of on which calls were made.
type fakeTripPlanner struct {
	err   error
	build func(req planner.Request) *planner.Plan
}

func (f fakeTripPlanner) Plan(_ context.Context, req planner.Request) (*planner.Plan, error) {
	if f.err != nil {
		return nil, f.err
	}

	if f.build != nil {
		return f.build(req), nil
	}

	return threeDayPlan(req), nil
}

func rainMM(mm float64) *float64 { return &mm }

func place(id, name, image string, lat, lon float64) planner.Place {
	return planner.Place{ID: id, Name: name, Image: image, Lat: lat, Lon: lon, Sitelinks: 100}
}

// threeDayPlan is a fixed three-day plan: a dry day, a rainy day, and a day past the
// forecast window.
func threeDayPlan(req planner.Request) *planner.Plan {
	dry := planner.Day{
		Date:    req.Start,
		Certain: true,
		WalkKM:  2.5,
		Forecast: &planner.DayForecast{
			Date:   req.Start,
			RainMM: rainMM(0.2),
		},
		Stops: []planner.Place{
			place("Q193386", "Belem Tower", "Torre de Belem.jpg", 38.6916, -9.2159),
			place("Q208420", "Jeronimos Monastery", "Mosteiro dos Jeronimos.jpg", 38.6979, -9.2065),
		},
	}

	rainyDate := req.Start.AddDate(0, 0, 1)
	rainy := planner.Day{
		Date:    rainyDate,
		Certain: true,
		Rainy:   true,
		WalkKM:  3.0,
		Forecast: &planner.DayForecast{
			Date:   rainyDate,
			RainMM: rainMM(12.4),
		},
		Stops: []planner.Place{
			place("Q1770963", "Tile Museum", "Museu do Azulejo.jpg", 38.7254, -9.1146),
			place("Q1543442", "Fado Museum", "", 38.7118, -9.1272),
		},
	}

	far := planner.Day{
		Date:   req.Start.AddDate(0, 0, 2),
		WalkKM: 1.2,
		Stops: []planner.Place{
			place("Q270086", "Castle of Sao Jorge", "Castelo de Sao Jorge.jpg", 38.7139, -9.1335),
		},
	}

	return &planner.Plan{
		Request:    req,
		Days:       []planner.Day{dry, rainy, far},
		Center:     dry.Stops[0],
		BookingURL: "https://www.booking.com/searchresults.html?ss=Lisbon%2C+portugal",
	}
}

// planServer builds a server on the fixed clock with the fake planner already set.
func planServer(t *testing.T, tripPlanner application.TripPlanner) *Server {
	t.Helper()

	server := NewAppServer(nil, nil, testStore)
	server.now = func() time.Time { return planNow }
	server.SetTripPlanner(tripPlanner)

	return server
}

// planRequest builds a valid /plan request, with overrides applied. An empty override
// removes the parameter.
func planRequest(overrides map[string]string) *http.Request {
	query := url.Values{
		"city":    {"Lisbon"},
		"country": {"portugal"},
		"lat":     {"38.7223"},
		"lon":     {"-9.1393"},
		"start":   {"2026-03-11"},
		"days":    {"3"},
	}

	for key, value := range overrides {
		if value == "" {
			query.Del(key)
			continue
		}
		query.Set(key, value)
	}

	return httptest.NewRequest("GET", "/plan?"+query.Encode(), nil)
}

// seedPlannerCity caches a city so the main page renders without calling the video reporter.
func seedPlannerCity(t *testing.T, city string) {
	t.Helper()

	require.NoError(t, testStore.SaveCityData(city, map[string]any{
		"GeneralInfo": createTestGeneralWeatherInfoFor(city),
		"Videos":      createTestVideosStream(),
	}))
}

func TestPlanRoute_RejectsDaysOutsideOneToFive(t *testing.T) {
	server := planServer(t, fakeTripPlanner{})

	for _, days := range []string{"0", "6", "many"} {
		status, body := doRequest(t, server, planRequest(map[string]string{"days": days}))

		assert.Equal(t, fiber.StatusBadRequest, status, "days=%s", days)
		assert.Contains(t, body, "between 1 and 5 days", "days=%s", days)
	}
}

func TestPlanRoute_RejectsABadStartDate(t *testing.T) {
	server := planServer(t, fakeTripPlanner{})

	status, body := doRequest(t, server, planRequest(map[string]string{"start": "11-03-2026"}))

	assert.Equal(t, fiber.StatusBadRequest, status)
	assert.Contains(t, body, "Pick a start date")
}

func TestPlanRoute_RejectsAStartDateInThePast(t *testing.T) {
	server := planServer(t, fakeTripPlanner{})

	// planNow is 2026-03-10, so the day before it is in the past.
	status, body := doRequest(t, server, planRequest(map[string]string{"start": "2026-03-09"}))

	assert.Equal(t, fiber.StatusBadRequest, status)
	assert.Contains(t, body, "start date is in the past")
}

func TestPlanRoute_RejectsMissingCoordinates(t *testing.T) {
	server := planServer(t, fakeTripPlanner{})

	status, body := doRequest(t, server, planRequest(map[string]string{"lat": "", "lon": ""}))

	assert.Equal(t, fiber.StatusBadRequest, status)
	assert.Contains(t, body, "Search for a city first")
}

func TestPlanRoute_RejectsInvalidCoordinates(t *testing.T) {
	server := planServer(t, fakeTripPlanner{})
	for _, coordinates := range []map[string]string{
		{"lat": "NaN"}, {"lat": "+Inf"}, {"lat": "-Inf"},
		{"lon": "NaN"}, {"lon": "+Inf"}, {"lon": "-Inf"},
		{"lat": "90.01"}, {"lat": "-90.01"},
		{"lon": "180.01"}, {"lon": "-180.01"},
	} {
		status, body := doRequest(t, server, planRequest(coordinates))
		assert.Equal(t, fiber.StatusBadRequest, status, "%v", coordinates)
		assert.Contains(t, body, "Search for a city first", "%v", coordinates)
	}
}

func TestPlanRoute_RejectsMissingCity(t *testing.T) {
	server := planServer(t, fakeTripPlanner{})
	for _, city := range []string{"", "   "} {
		status, body := doRequest(t, server, planRequest(map[string]string{"city": city}))
		assert.Equal(t, fiber.StatusBadRequest, status)
		assert.Contains(t, body, "Search for a city first")
	}
}

func TestPlanRoute_RendersEveryDayOfThePlan(t *testing.T) {
	server := planServer(t, fakeTripPlanner{})

	status, body := doRequest(t, server, planRequest(nil))

	require.Equal(t, fiber.StatusOK, status)
	assert.NotContains(t, body, "<html", "the planner answers with a fragment")

	for _, heading := range []string{"Day 1", "Day 2", "Day 3"} {
		assert.Contains(t, body, heading)
	}

	for _, date := range []string{"Wed, 11 Mar", "Thu, 12 Mar", "Fri, 13 Mar"} {
		assert.Contains(t, body, date)
	}

	for _, stop := range []string{"Belem Tower", "Jeronimos Monastery", "Tile Museum", "Fado Museum", "Castle of Sao Jorge"} {
		assert.Contains(t, body, stop)
	}

	for _, walk := range []string{"2.5 km", "3.0 km", "1.2 km"} {
		assert.Contains(t, body, walk)
	}
}

func TestPlanRoute_ShowsTheRainBadgeOnARainyDay(t *testing.T) {
	server := planServer(t, fakeTripPlanner{})

	status, body := doRequest(t, server, planRequest(nil))

	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, "12 mm")
	assert.Contains(t, body, "Rainy day")
}

func TestPlanRoute_MarksLessCertainDays(t *testing.T) {
	server := planServer(t, fakeTripPlanner{build: func(req planner.Request) *planner.Plan {
		plan := threeDayPlan(req)
		plan.Days[2].Forecast = &planner.DayForecast{Date: plan.Days[2].Date, RainMM: rainMM(3)}
		return plan
	}})

	status, body := doRequest(t, server, planRequest(nil))

	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, "less certain")
	// Only the third day has a forecast outside the trusted window.
	assert.Equal(t, 1, countOccurrences(body, "less certain"))
}

func countOccurrences(body, needle string) int {
	count := 0
	for i := 0; i+len(needle) <= len(body); i++ {
		if body[i:i+len(needle)] == needle {
			count++
		}
	}
	return count
}

func TestPlanRoute_ShowsAFriendlyMessageWhenPlanningFails(t *testing.T) {
	server := planServer(t, fakeTripPlanner{err: errors.New("overpass is down")})

	status, body := doRequest(t, server, planRequest(nil))

	assert.Equal(t, fiber.StatusInternalServerError, status)
	assert.Contains(t, body, "could not plan your trip")
	assert.NotContains(t, body, "overpass is down", "internal errors stay in the logs")
}

func TestMainPage_HidesThePlannerCardWithoutAPlanner(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, _ := createTestServer(ctrl)
	seedPlannerCity(t, "Planless")
	mockWeather.EXPECT().GenerateReport(gomock.Any(), "Planless").Return(createTestGeneralWeatherInfoFor("Planless"), nil)

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/?city_name=Planless", nil))

	require.Equal(t, fiber.StatusOK, status)
	assert.NotContains(t, body, "Plan my trip")
	assert.NotContains(t, body, "trip-card")
	// The page still looks exactly as it does today.
	assert.Contains(t, body, "Test Video 1")
	assert.Contains(t, body, "Planless")
}

func TestMainPage_ShowsThePlannerCardWithCityAndCoordinates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, _ := createTestServer(ctrl)
	server.now = func() time.Time { return planNow }
	server.SetTripPlanner(fakeTripPlanner{})
	seedPlannerCity(t, "Planburg")
	mockWeather.EXPECT().GenerateReport(gomock.Any(), "Planburg").Return(createTestGeneralWeatherInfoFor("Planburg"), nil)

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/?city_name=Planburg", nil))

	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, "Plan my trip")
	assert.Contains(t, body, `name="city" value="Planburg"`)
	assert.Contains(t, body, `name="country" value="portugal"`)
	assert.Contains(t, body, `name="lat" value="38.7223"`)
	assert.Contains(t, body, `name="lon" value="-9.1393"`)
}

func TestMainPage_PlannerStartsTomorrowByDefault(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, _ := createTestServer(ctrl)
	server.now = func() time.Time { return planNow }
	server.SetTripPlanner(fakeTripPlanner{})
	seedPlannerCity(t, "Planton")
	mockWeather.EXPECT().GenerateReport(gomock.Any(), "Planton").Return(createTestGeneralWeatherInfoFor("Planton"), nil)

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/?city_name=Planton", nil))

	require.Equal(t, fiber.StatusOK, status)
	// planNow is 2026-03-10, so the form starts on the 11th and never offers a past day.
	assert.Contains(t, body, `name="start" value="2026-03-11"`)
	assert.Contains(t, body, `min="2026-03-10"`)
}
