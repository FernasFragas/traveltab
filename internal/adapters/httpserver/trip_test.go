package httpserver

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
	"weatherservice/internal/application"
	"weatherservice/internal/writermap"
	"weatherservice/internal/writermap/writermaptest"

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
	assert.Contains(t, body, "Rain · 12 mm")
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
	assert.Contains(t, body, "Plan your trip")
	assert.Contains(t, body, `name="city" value="Planburg"`)
	assert.Contains(t, body, `name="country" value="portugal"`)
	assert.Contains(t, body, `name="lat" value="38.7223"`)
	assert.Contains(t, body, `name="lon" value="-9.1393"`)
}

// --- A2: GET /trip/:slug ---

func TestTripPage_RendersTheWeatherAndPlanFromASlug(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, mockVideos := createTestServer(ctrl)
	server.now = func() time.Time { return planNow }
	server.SetTripPlanner(fakeTripPlanner{})
	forgetCity(t, "Sluglisbon")
	mockWeather.EXPECT().GenerateReport(gomock.Any(), "sluglisbon, pt").Return(createTestGeneralWeatherInfoFor("Sluglisbon"), nil)
	expectFreshData(mockVideos, "Sluglisbon")

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/trip/sluglisbon-pt?days=3&from=2026-03-11", nil))

	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, "<html", "a direct visit renders the full page, not a fragment")
	for _, heading := range []string{"Day 1", "Day 2", "Day 3"} {
		assert.Contains(t, body, heading)
	}
	assert.Contains(t, body, "Belem Tower")
}

func TestTripPage_WithNoDatesShowsThePrefilledFormOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, mockVideos := createTestServer(ctrl)
	server.now = func() time.Time { return planNow }
	server.SetTripPlanner(fakeTripPlanner{})
	forgetCity(t, "Slugporto")
	mockWeather.EXPECT().GenerateReport(gomock.Any(), "slugporto, pt").Return(createTestGeneralWeatherInfoFor("Slugporto"), nil)
	expectFreshData(mockVideos, "Slugporto")

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/trip/slugporto-pt", nil))

	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, "Plan your trip")
	assert.Contains(t, body, `name="city" value="Slugporto"`)
	assert.Contains(t, body, `name="country" value="portugal"`, "the fixture's GeneralWeatherInfo carries the country the weather API returned, not the slug's own")
	assert.NotContains(t, body, "Day 1", "no click has happened yet, so there is no plan")
}

func TestTripPage_InvalidDatesFallBackToThePrefilledForm(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, mockVideos := createTestServer(ctrl)
	server.now = func() time.Time { return planNow }
	server.SetTripPlanner(fakeTripPlanner{})
	forgetCity(t, "Slugfaro")
	mockWeather.EXPECT().GenerateReport(gomock.Any(), "slugfaro, pt").Return(createTestGeneralWeatherInfoFor("Slugfaro"), nil)
	expectFreshData(mockVideos, "Slugfaro")

	// A partial or malformed query never errors the page: it just falls back to the form.
	status, body := doRequest(t, server, httptest.NewRequest("GET", "/trip/slugfaro-pt?days=3", nil))

	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, "Plan your trip")
	assert.NotContains(t, body, "Day 1")
}

func TestTripPage_UnknownSlugFallsBackToTheCityQueryFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, mockVideos := createTestServer(ctrl)
	server.now = func() time.Time { return planNow }
	forgetCity(t, "Slugunseen")
	mockWeather.EXPECT().GenerateReport(gomock.Any(), "slugunseen, pt").Return(createTestGeneralWeatherInfoFor("Slugunseen"), nil)
	expectFreshData(mockVideos, "Slugunseen")

	// A city that has never been searched before still renders, exactly like visiting "/"
	// with a new city_name: a fresh fetch, then a cached row for next time.
	status, body := doRequest(t, server, httptest.NewRequest("GET", "/trip/slugunseen-pt", nil))

	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, "Test Video 1")
	waitForCache(t, "Slugunseen")
}

func TestTripPage_MalformedSlugIs404(t *testing.T) {
	server := planServer(t, fakeTripPlanner{})

	status, _ := doRequest(t, server, httptest.NewRequest("GET", "/trip/not-a-country-code", nil))

	assert.Equal(t, fiber.StatusNotFound, status)
}

func TestPlanRoute_SetsHXPushURLToTheShareableLink(t *testing.T) {
	server := planServer(t, fakeTripPlanner{})

	resp, err := server.app.Test(planRequest(map[string]string{"country": "pt"}), 5000)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, "/trip/lisbon-pt?days=3&from=2026-03-11", resp.Header.Get("HX-Push-Url"))
}

func TestPlanRoute_PushURLOmitsTheDateWhenThePlanFailed(t *testing.T) {
	server := planServer(t, fakeTripPlanner{err: errors.New("overpass is down")})

	resp, err := server.app.Test(planRequest(map[string]string{"country": "pt"}), 5000)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, "/trip/lisbon-pt", resp.Header.Get("HX-Push-Url"), "still shareable, but with no dates to a plan that failed")
}

func TestPlanRoute_PushURLIsEmptyWhenThereIsNoCityToShare(t *testing.T) {
	server := planServer(t, fakeTripPlanner{})

	resp, err := server.app.Test(planRequest(map[string]string{"city": ""}), 5000)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.Empty(t, resp.Header.Get("HX-Push-Url"))
}

// --- A3: title, canonical URL, sitemap, and a crawler limit ---

func TestTripPage_HasATitleAndDescription(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, mockVideos := createTestServer(ctrl)
	server.now = func() time.Time { return planNow }
	server.SetTripPlanner(fakeTripPlanner{})
	forgetCity(t, "Slugtitle")
	mockWeather.EXPECT().GenerateReport(gomock.Any(), "slugtitle, pt").Return(createTestGeneralWeatherInfoFor("Slugtitle"), nil)
	expectFreshData(mockVideos, "Slugtitle")

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/trip/slugtitle-pt?days=3&from=2026-03-11", nil))

	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, "<title>3 days in Slugtitle — TravelTabByPatronFragas</title>")
	assert.Contains(t, body, `<meta name="description" content="A day-by-day plan for a 3-day trip to Slugtitle, with weather-aware stops and places to stay.">`)
}

func TestTripPage_UnplannedPageStillHasATitle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, mockVideos := createTestServer(ctrl)
	server.now = func() time.Time { return planNow }
	server.SetTripPlanner(fakeTripPlanner{})
	forgetCity(t, "Slugnodate")
	mockWeather.EXPECT().GenerateReport(gomock.Any(), "slugnodate, pt").Return(createTestGeneralWeatherInfoFor("Slugnodate"), nil)
	expectFreshData(mockVideos, "Slugnodate")

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/trip/slugnodate-pt", nil))

	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, "<title>Plan a trip to Slugnodate — TravelTabByPatronFragas</title>")
	assert.NotContains(t, body, "<title>✈🌤️TravelTab</title>", "the slug page never falls back to the plain home-page title")
}

func TestMainPage_KeepsItsPlainTitle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, mockVideos := createTestServer(ctrl)
	forgetCity(t, "Titlecheck")
	mockWeather.EXPECT().GenerateReport(gomock.Any(), "Titlecheck").Return(createTestGeneralWeatherInfoFor("Titlecheck"), nil)
	expectFreshData(mockVideos, "Titlecheck")

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/?city_name=Titlecheck", nil))

	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, "<title>TravelTabByPatronFragas</title>")
	assert.NotContains(t, body, `<link rel="canonical"`)
}

func TestTripPage_CanonicalLinkExcludesDaysAndFrom(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, mockVideos := createTestServer(ctrl)
	server.now = func() time.Time { return planNow }
	server.SetTripPlanner(fakeTripPlanner{})
	forgetCity(t, "Slugcanonical")
	mockWeather.EXPECT().GenerateReport(gomock.Any(), "slugcanonical, pt").Return(createTestGeneralWeatherInfoFor("Slugcanonical"), nil)
	expectFreshData(mockVideos, "Slugcanonical")

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/trip/slugcanonical-pt?days=3&from=2026-03-11", nil))

	require.Equal(t, fiber.StatusOK, status)
	assert.Contains(t, body, `<link rel="canonical" href="/trip/slugcanonical-pt">`)
	assert.NotContains(t, body, "canonical\" href=\"/trip/slugcanonical-pt?")
}

func TestSitemap_IsValidXML(t *testing.T) {
	server := planServer(t, fakeTripPlanner{})

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/sitemap.xml", nil))

	require.Equal(t, fiber.StatusOK, status)
	var doc struct {
		XMLName xml.Name `xml:"urlset"`
	}
	require.NoError(t, xml.Unmarshal([]byte(body), &doc))
}

func TestSitemap_ListsOnlyCachedCities(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, mockVideos := createTestServer(ctrl)
	server.now = func() time.Time { return planNow }
	server.SetTripPlanner(fakeTripPlanner{})
	forgetCity(t, "Sitemapcity")
	mockWeather.EXPECT().GenerateReport(gomock.Any(), "sitemapcity, pt").Return(createTestGeneralWeatherInfoFor("Sitemapcity"), nil)
	expectFreshData(mockVideos, "Sitemapcity")

	status, _ := doRequest(t, server, httptest.NewRequest("GET", "/trip/sitemapcity-pt", nil))
	require.Equal(t, fiber.StatusOK, status)
	waitForCache(t, "Sitemapcity")

	require.Eventually(t, func() bool {
		_, body := doRequest(t, server, httptest.NewRequest("GET", "/sitemap.xml", nil))
		return strings.Contains(body, "/trip/sitemapcity-pt")
	}, 2*time.Second, 10*time.Millisecond, "the sitemap index write is async, like every city_data save")

	_, body := doRequest(t, server, httptest.NewRequest("GET", "/sitemap.xml", nil))
	assert.NotContains(t, body, "/trip/never-searched-xx", "a city nobody has asked for is never listed")
}

func TestPlanRoute_LimitsNewCitiesPerMinute(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, mockVideos := createTestServer(ctrl)
	server.now = func() time.Time { return planNow }
	server.SetTripPlanner(fakeTripPlanner{})
	server.newCityLimiter = newNewCityLimiter(1)

	forgetCity(t, "Ratecity1")
	forgetCity(t, "Ratecity2")
	mockWeather.EXPECT().GenerateReport(gomock.Any(), "ratecity1, pt").Return(createTestGeneralWeatherInfoFor("Ratecity1"), nil).Times(2)
	mockWeather.EXPECT().GenerateReport(gomock.Any(), "ratecity2, pt").Return(createTestGeneralWeatherInfoFor("Ratecity2"), nil)
	expectFreshData(mockVideos, "Ratecity1")
	expectFreshData(mockVideos, "Ratecity2")

	status1, body1 := doRequest(t, server, httptest.NewRequest("GET", "/trip/ratecity1-pt?days=3&from=2026-03-11", nil))
	require.Equal(t, fiber.StatusOK, status1)
	assert.Contains(t, body1, "Day 1", "the first new city of the minute is planned")
	waitForCache(t, "Ratecity1")

	status2, body2 := doRequest(t, server, httptest.NewRequest("GET", "/trip/ratecity2-pt?days=3&from=2026-03-11", nil))
	require.Equal(t, fiber.StatusOK, status2)
	assert.NotContains(t, body2, "Day 1", "a second new city inside the same minute is not planned")
	assert.Contains(t, body2, "Plan your trip")
	waitForCache(t, "Ratecity2")

	// A repeat visit to the already-cached first city is never limited, even though the
	// limiter's single slot for this minute is already used.
	status3, body3 := doRequest(t, server, httptest.NewRequest("GET", "/trip/ratecity1-pt?days=3&from=2026-03-11", nil))
	require.Equal(t, fiber.StatusOK, status3)
	assert.Contains(t, body3, "Day 1", "revisiting a cached city is never rate-limited")
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

// includeServer plans with a recording planner and a fake writers' map holding Q1 and Q2.
func includeServer(t *testing.T, enabled bool, recorded *planner.Request) *Server {
	t.Helper()

	server := planServer(t, fakeTripPlanner{build: func(req planner.Request) *planner.Plan {
		*recorded = req
		plan := threeDayPlan(req)
		plan.AlsoSuggested = []planner.Place{place("Q9", "Extra Garden", "", 38.7, -9.1)}
		return plan
	}})
	server.SetWriterMapSource(&writermaptest.FakeSource{Result: writermap.DestinationResult{Places: []writermap.Place{
		{QID: "Q1", Names: []string{"Saved One", "Alias"}, Lat: 38.71, Lon: -9.14, Sitelinks: 50, Types: []string{"T1"}, Photo: "One.jpg", Description: "Saved description"},
		{QID: "Q2", Lat: 38.72, Lon: -9.15},
	}}}, enabled)

	return server
}

func includeRequest(qids ...string) *http.Request {
	req := planRequest(nil)
	query := req.URL.Query()
	query["include"] = qids
	req.URL.RawQuery = query.Encode()
	req.RequestURI = req.URL.RequestURI()

	return req
}

func TestPlanRoute_SeedsTheIncludedPlacesIntoTheRequest(t *testing.T) {
	var got planner.Request
	server := includeServer(t, true, &got)

	status, _ := doRequest(t, server, includeRequest("Q1", "Q2"))

	require.Equal(t, fiber.StatusOK, status)
	require.Len(t, got.Include, 2)
	assert.Equal(t, planner.Place{ID: "Q1", Name: "Saved One", Lat: 38.71, Lon: -9.14, Sitelinks: 50, Types: []string{"T1"}, Image: "One.jpg", Description: "Saved description"}, got.Include[0])
	assert.Equal(t, "Q2", got.Include[1].ID)
	assert.Equal(t, "Q2", got.Include[1].Name, "a place with no name shows its QID")
}

func TestPlanRoute_IgnoresUnknownAndRepeatedIncludes(t *testing.T) {
	var got planner.Request
	server := includeServer(t, true, &got)

	status, _ := doRequest(t, server, includeRequest("Q404", "Q1", "Q1", ""))

	require.Equal(t, fiber.StatusOK, status)
	require.Len(t, got.Include, 1)
	assert.Equal(t, "Q1", got.Include[0].ID)
}

func TestPlanRoute_ReadsAtMostTwelveIncludes(t *testing.T) {
	var got planner.Request
	server := includeServer(t, true, &got)
	qids := make([]string, 0, 30)
	for i := 0; i < 30; i++ {
		qids = append(qids, fmt.Sprintf("Q5%02d", i))
	}
	qids = append(qids, "Q1", "Q2")
	// Unknown ones count against the limit: the server never looks up more than twelve.
	status, _ := doRequest(t, server, includeRequest(qids...))

	require.Equal(t, fiber.StatusOK, status)
	assert.Empty(t, got.Include)
}

func TestPlanRoute_IgnoresIncludeWhenTheWritersMapIsOff(t *testing.T) {
	var got planner.Request
	server := includeServer(t, false, &got)

	status, _ := doRequest(t, server, includeRequest("Q1"))

	require.Equal(t, fiber.StatusOK, status)
	assert.Empty(t, got.Include)
}

func TestPlanRoute_IgnoresIncludeWithoutAWritersMapSource(t *testing.T) {
	var got planner.Request
	server := planServer(t, fakeTripPlanner{build: func(req planner.Request) *planner.Plan {
		got = req
		return threeDayPlan(req)
	}})

	status, _ := doRequest(t, server, includeRequest("Q1"))

	require.Equal(t, fiber.StatusOK, status)
	assert.Empty(t, got.Include)
}

func TestPlanRoute_ListsPlacesTheDaysHadNoRoomForAsAlsoSuggested(t *testing.T) {
	var got planner.Request
	server := includeServer(t, true, &got)

	_, body := doRequest(t, server, includeRequest("Q1"))

	assert.Contains(t, body, "Also suggested")
	assert.Contains(t, body, "Extra Garden")
}

func TestPlanRoute_HasNoAlsoSuggestedBlockWhenNothingIsLeftOver(t *testing.T) {
	server := planServer(t, fakeTripPlanner{})

	_, body := doRequest(t, server, planRequest(nil))

	assert.NotContains(t, body, "Also suggested")
}

// travelersServer plans with a recording planner.
func travelersServer(t *testing.T, recorded *planner.Request) *Server {
	t.Helper()

	return planServer(t, fakeTripPlanner{build: func(req planner.Request) *planner.Plan {
		*recorded = req
		return threeDayPlan(req)
	}})
}

func TestPlanRoute_PassesTheTravelersToThePlanner(t *testing.T) {
	var got planner.Request
	server := travelersServer(t, &got)

	status, body := doRequest(t, server, planRequest(map[string]string{"travelers": "5"}))

	require.Equal(t, fiber.StatusOK, status)
	assert.Equal(t, 5, got.Travelers)
	assert.Contains(t, body, `<option value="5" selected>5</option>`, "the form keeps the chosen number")
}

func TestPlanRoute_MissingOrInvalidTravelersFallBackToTwo(t *testing.T) {
	for name, value := range map[string]string{"missing": "", "zero": "0", "too many": "9", "text": "many", "negative": "-3"} {
		t.Run(name, func(t *testing.T) {
			var got planner.Request
			server := travelersServer(t, &got)

			status, body := doRequest(t, server, planRequest(map[string]string{"travelers": value}))

			require.Equal(t, fiber.StatusOK, status)
			assert.Equal(t, 2, got.Travelers)
			assert.Contains(t, body, `<option value="2" selected>2</option>`)
		})
	}
}

func TestPlanRoute_KeepsTheTravelersInTheShareableLink(t *testing.T) {
	var got planner.Request
	server := travelersServer(t, &got)

	resp, err := server.app.Test(planRequest(map[string]string{"country": "pt", "travelers": "4"}), 5000)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "/trip/lisbon-pt?days=3&from=2026-03-11&travelers=4", resp.Header.Get("HX-Push-Url"))
}

func TestTripPage_ReadsTheTravelersFromTheLink(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var got planner.Request
	server, mockWeather, _ := createTestServer(ctrl)
	server.now = func() time.Time { return planNow }
	server.SetTripPlanner(fakeTripPlanner{build: func(req planner.Request) *planner.Plan {
		got = req
		return threeDayPlan(req)
	}})
	seedPlannerCity(t, "Planton")
	mockWeather.EXPECT().GenerateReport(gomock.Any(), "planton, pt").Return(createTestGeneralWeatherInfoFor("Planton"), nil)

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/trip/planton-pt?days=3&from=2026-03-11&travelers=6", nil))

	require.Equal(t, fiber.StatusOK, status)
	assert.Equal(t, 6, got.Travelers)
	assert.Contains(t, body, `<option value="6" selected>6</option>`)
}

func TestMainPage_TravelersSelectOffersOneToEightWithTwoSelected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, _ := createTestServer(ctrl)
	server.now = func() time.Time { return planNow }
	server.SetTripPlanner(fakeTripPlanner{})
	seedPlannerCity(t, "Planton")
	mockWeather.EXPECT().GenerateReport(gomock.Any(), "Planton").Return(createTestGeneralWeatherInfoFor("Planton"), nil)

	_, body := doRequest(t, server, httptest.NewRequest("GET", "/?city_name=Planton", nil))

	assert.Contains(t, body, `name="travelers"`)
	for n := 1; n <= 8; n++ {
		assert.Contains(t, body, fmt.Sprintf(`<option value="%d"`, n))
	}
	assert.NotContains(t, body, `<option value="9"`)
	assert.Contains(t, body, `<option value="2" selected>2</option>`)
}
