package httpserver

import (
	"bytes"
	"github.com/gofiber/template/html/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"weatherservice/internal/application"
	"weatherservice/internal/planner"
)

func TestOverviewTeasers_PlanAndNoPlan(t *testing.T) {
	for _, planned := range []bool{false, true} {
		engine := html.New("./views", ".go.tpl")
		require.NoError(t, engine.Load())
		var trip *tripCard
		if planned {
			trip = &tripCard{Plan: &tripPlanView{Days: []tripDayView{{Stops: []planner.Place{{Name: "Belém Tower"}}}}, StayCards: []stayCardView{{Name: "Hotel Example", Kind: "hotel", Stars: 5}}}}
		}
		var out bytes.Buffer
		err := engine.Render(&out, "overview_teasers", map[string]any{"GeneralInfo": application.GeneralWeatherInfo{City: "Lisbon"}, "Trip": trip, "Presentation": presentationView{Videos: []videoView{{Title: "Lisbon guide", ThumbnailURL: "/video.jpg"}}}})
		require.NoError(t, err)
		markup := out.String()
		for _, want := range []string{"#itinerary", "#stays", "#videos", "Plan your days", "Where to stay", "Travel videos", "Lisbon guide"} {
			assert.Contains(t, markup, want)
		}
		if planned {
			assert.Contains(t, markup, "Belém Tower")
			assert.Contains(t, markup, "Hotel Example")
			assert.Contains(t, markup, "5-star property")
		} else {
			assert.Contains(t, markup, "Suggested day")
			assert.Contains(t, markup, "Generate a plan to see places to stay")
		}
	}
}

func TestOverviewTeasers_NoData(t *testing.T) {
	engine := html.New("./views", ".go.tpl")
	require.NoError(t, engine.Load())
	var out bytes.Buffer
	require.NoError(t, engine.Render(&out, "overview_teasers", map[string]any{"GeneralInfo": application.GeneralWeatherInfo{City: "Porto"}, "Presentation": presentationView{}, "Trip": &tripCard{Plan: &tripPlanView{}}}))
	assert.NotContains(t, out.String(), "Travel videos")
	assert.Contains(t, out.String(), "No stays available yet")
}
