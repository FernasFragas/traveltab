package httpserver

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"weatherservice/internal/planner"
)

func TestDayPlan_EmitsStopCoordinatesPerDay(t *testing.T) {
	body := renderPlan(t, func(req planner.Request) *planner.Plan {
		return &planner.Plan{Request: req, Days: []planner.Day{
			{Date: req.Start, Certain: true, Stops: stayPhotos[:2]},
			{Date: req.Start.AddDate(0, 0, 1), Certain: true, Stops: stayPhotos[2:3]},
		}}
	})

	assert.Equal(t, 2, countOccurrences(body, "data-plan-day="))
	assert.Contains(t, body, `data-plan-day="1"`)
	assert.Contains(t, body, `data-plan-day="2"`)
	assert.Equal(t, 3, countOccurrences(body, "data-plan-stop"))
	assert.Contains(t, body, `data-lat="38.6916" data-lon="-9.2159"`)
	assert.Contains(t, body, `data-name="Tile Museum"`)
}

func TestDayPlan_EmitsNothingWithoutAPlan(t *testing.T) {
	server := planServer(t, fakeTripPlanner{build: func(req planner.Request) *planner.Plan {
		return &planner.Plan{Request: req}
	}})
	_, body := doRequest(t, server, planRequest(nil))

	assert.NotContains(t, body, "data-plan-day")
	assert.NotContains(t, body, "data-plan-stop")
}
