package application

import (
	"context"
	"time"

	"weatherservice/internal/planner"
)

// TripPlanner turns a request into a day-by-day plan. cmd/web wires the real one in.
type TripPlanner interface {
	Plan(ctx context.Context, req planner.Request) (*planner.Plan, error)
}

// tripPlanner joins the three sources cmd/web wires in to the planning chain. It is the only
// thing between the route and planner.Build, so the application never has to know where the
// places, the forecast and the stays come from.
type tripPlanner struct {
	places   planner.PlaceSource
	forecast planner.ForecastSource
	stays    planner.StaySource
	now      func() time.Time
}

// NewTripPlanner returns the planner the trip card uses. cmd/web passes the cached API clients.
func NewTripPlanner(places planner.PlaceSource, forecast planner.ForecastSource, stays planner.StaySource) TripPlanner {
	return &tripPlanner{places: places, forecast: forecast, stays: stays, now: time.Now}
}

// Plan builds the trip on today's clock, which is what decides how far the forecast is trusted.
func (t *tripPlanner) Plan(ctx context.Context, req planner.Request) (*planner.Plan, error) {
	return planner.Build(ctx, req, t.places, t.forecast, t.stays, t.now())
}
