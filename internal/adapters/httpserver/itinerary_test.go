package httpserver

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"weatherservice/internal/planner"
)

func TestItinerary_ShowsTheMaxTemperatureAndConditionOfEachDay(t *testing.T) {
	max, code := 23.6, 1
	body := renderPlan(t, func(req planner.Request) *planner.Plan {
		return &planner.Plan{Request: req, Days: []planner.Day{{
			Date: req.Start, Certain: true, Stops: stayPhotos[:2],
			Forecast: &planner.DayForecast{Date: req.Start, MaxC: &max, WeatherCode: &code},
		}}}
	})

	assert.Contains(t, body, `data-day-temp>24°`)
	assert.Contains(t, body, "Mostly clear")
	assert.Contains(t, body, weatherCodeIcon(code))
}

func TestItinerary_DaysWithoutForecastDataShowNoTemperature(t *testing.T) {
	code := 3
	for name, forecast := range map[string]*planner.DayForecast{
		"no forecast":    nil,
		"no max":         {WeatherCode: &code},
		"empty forecast": {},
	} {
		body := renderPlan(t, func(req planner.Request) *planner.Plan {
			return &planner.Plan{Request: req, Days: []planner.Day{{Date: req.Start, Stops: stayPhotos[:2], Forecast: forecast}}}
		})

		assert.NotContains(t, body, "data-day-temp", name)
		assert.NotContains(t, body, "0°", name)
	}
}

func TestItinerary_FirstStopShowsNoDistanceAndLaterStopsShowTheDistanceFromThePreviousOne(t *testing.T) {
	body := renderPlan(t, func(req planner.Request) *planner.Plan {
		return &planner.Plan{Request: req, Days: []planner.Day{
			{Date: req.Start, Stops: stayPhotos[:2]},
			{Date: req.Start.AddDate(0, 0, 1), Stops: stayPhotos[2:4]},
		}}
	})

	want := []string{
		fmt.Sprintf("%.1f km", planner.DistanceKM(stayPhotos[0], stayPhotos[1])),
		fmt.Sprintf("%.1f km", planner.DistanceKM(stayPhotos[2], stayPhotos[3])),
	}
	assert.Equal(t, 2, countOccurrences(body, "data-stop-distance"), "only the second stop of each day")
	for _, km := range want {
		assert.Contains(t, body, "data-stop-distance>"+km+" from previous")
	}
}

func TestItinerary_HasExportToCalendarAndViewOnMapButtons(t *testing.T) {
	body := renderPlan(t, func(req planner.Request) *planner.Plan {
		return &planner.Plan{Request: req, Days: []planner.Day{{Date: req.Start, Stops: stayPhotos[:2]}}}
	})

	assert.Contains(t, body, "Export to calendar")
	assert.Contains(t, body, ".ics")
	assert.Contains(t, body, "data-view-on-map")
	// No inline handler (a CSP without 'unsafe-inline'); map-dayplan.js handles the click.
	assert.NotContains(t, body, "onclick")
	assert.NotContains(t, body, "setLayerEnabled")
}

func TestItinerary_AlsoSuggestedStillRenders(t *testing.T) {
	body := renderPlan(t, func(req planner.Request) *planner.Plan {
		return &planner.Plan{Request: req, Days: []planner.Day{{Date: req.Start, Stops: stayPhotos[:2]}},
			AlsoSuggested: []planner.Place{{ID: "Q9", Name: "Spare Castle"}}}
	})

	assert.Contains(t, body, `data-also-suggested="Q9"`)
}

func TestSavedLine_IsStyledAndLoaded(t *testing.T) {
	css, err := os.ReadFile("public/redesign/map-saved.css")
	require.NoError(t, err)
	assert.Contains(t, string(css), ".writers-map-saved")

	index, err := os.ReadFile("views/index.go.tpl")
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(index), `href="/redesign/map-saved.css"`))
}

func TestItinerary_ShowsAShortDescriptionUnderTheStopNameWhenItHasOne(t *testing.T) {
	described := stayPhotos[0]
	described.Description = "tower in Lisbon <b>Portugal</b>"

	body := renderPlan(t, func(req planner.Request) *planner.Plan {
		return &planner.Plan{Request: req, Days: []planner.Day{{Date: req.Start, Stops: []planner.Place{described, stayPhotos[1]}}}}
	})

	assert.Equal(t, 1, countOccurrences(body, "itinerary-stop-description"), "only the stop that has one")
	assert.Contains(t, body, `<span class="itinerary-stop-description">tower in Lisbon &lt;b&gt;Portugal&lt;/b&gt;</span>`)
}

func TestItinerary_StopsWithoutADescriptionRenderNoDescriptionElement(t *testing.T) {
	body := renderPlan(t, func(req planner.Request) *planner.Plan {
		return &planner.Plan{Request: req, Days: []planner.Day{{Date: req.Start, Stops: stayPhotos[:2]}}}
	})

	assert.NotContains(t, body, "itinerary-stop-description")
}

func TestItinerary_DescriptionStyleWrapsLongTextWithoutClamp(t *testing.T) {
	css, err := os.ReadFile("public/redesign/planner.css")
	require.NoError(t, err)

	start := strings.Index(string(css), ".itinerary-stop-description {")
	require.GreaterOrEqual(t, start, 0, "planner.css styles .itinerary-stop-description")

	rule := string(css)[start:]
	rule = rule[:strings.Index(rule, "}")]

	assert.NotContains(t, rule, "line-clamp")
	assert.NotContains(t, rule, "overflow: hidden")
	assert.NotContains(t, rule, "-webkit-box")
	assert.Contains(t, rule, "overflow-wrap: anywhere")
}

func TestHeroLine_StyleWrapsLongTextWithoutClamp(t *testing.T) {
	css, err := os.ReadFile("public/redesign/destination.css")
	require.NoError(t, err)

	start := strings.Index(string(css), ".destination-tagline {")
	require.GreaterOrEqual(t, start, 0)

	rule := string(css)[start:]
	rule = rule[:strings.Index(rule, "}")]

	assert.NotContains(t, rule, "line-clamp")
	assert.NotContains(t, rule, "overflow: hidden")
	assert.NotContains(t, rule, "-webkit-box")
	assert.Contains(t, rule, "overflow-wrap: anywhere")
}
