package planner

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// The messages a plan carries when it can't be the plan that was asked for.
const (
	shortTripNote          = "%s has enough highlights for %d %s, so here's a %d-day plan."
	farTripNote            = "Forecast appears closer to your trip"
	weatherUnavailableNote = "Weather is unavailable right now, so the days are not ordered by the forecast."
	staysUnavailableNote   = "Places to stay are unavailable right now"
)

// Build runs the whole planning chain: it loads the places, ranks them, splits them into
// walking days, moves those days around the forecast, and looks for somewhere to stay in the
// middle of it all. Only the places are essential: a missing forecast gives a plan without
// weather and an explanatory note, and a missing stay list gives a plan with StaysNote set.
// Time comes in through now, so the same inputs always give the same plan.
func Build(ctx context.Context, req Request, places PlaceSource, forecast ForecastSource, stays StaySource, now time.Time) (*Plan, error) {
	if req.Days < 1 || req.Days > MaxDays {
		return nil, fmt.Errorf("error planning %s: a trip is 1 to %d days, not %d", req.City, MaxDays, req.Days)
	}

	// These lookups are independent. Cancel the forecast if places fail or cannot fill a plan;
	// the buffered result lets its goroutine finish even when we return without reading it.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	type forecastResult struct {
		forecast *Forecast
		err      error
	}
	forecastReady := make(chan forecastResult, 1)
	go func() {
		fc, err := forecast.DailyForecast(ctx, req.Lat, req.Lon)
		forecastReady <- forecastResult{forecast: fc, err: err}
	}()

	candidates, err := places.PlacesNear(ctx, req.Lat, req.Lon)
	if err != nil {
		return nil, fmt.Errorf("error loading places near %s: %w", req.City, err)
	}

	ranked := Rank(candidates, PlaceTypes, Boosts)
	groups := GroupDays(ranked, req.Days)
	if len(groups) == 0 {
		return nil, fmt.Errorf("error planning %s: no place nearby is worth a visit", req.City)
	}
	var notes []string
	if len(groups) < req.Days {
		notes = append(notes, shortTrip(req.City, len(groups)))
	}

	result := <-forecastReady
	fc := result.forecast
	if result.err != nil {
		log.Printf("error loading the forecast for %s, planning without weather: %v", req.City, result.err)
		fc = nil
		notes = append(notes, weatherUnavailableNote)
	}
	today := now.In(forecastLocation(fc))
	if daysBetween(today, req.Start) > ForecastMaxDays {
		notes = append(notes, farTripNote)
	}

	plan := &Plan{
		Request:    req,
		Days:       Schedule(groups, ranked, fc, req.Start, today),
		Note:       strings.Join(notes, " "),
		BookingURL: BookingURL(req.City, req.Country, req.Start, req.Days),
	}

	plan.Center = Medoid(planStops(plan.Days))
	nearby, err := stays.StaysNear(ctx, plan.Center.Lat, plan.Center.Lon)
	if err != nil {
		log.Printf("error loading places to stay near %s: %v", req.City, err)
		plan.StaysNote = staysUnavailableNote
	} else {
		plan.Stays = PickStays(plan.Center, nearby)
	}
	return plan, nil
}

// shortTrip is the note a city gets when it only has highlights for part of the trip.
func shortTrip(city string, days int) string {
	unit := "days"
	if days == 1 {
		unit = "day"
	}
	return fmt.Sprintf(shortTripNote, city, days, unit, days)
}

// planStops is every stop of the plan, so the center can be measured against all of them.
func planStops(days []Day) []Place {
	stops := make([]Place, 0, len(days)*MaxStopsPerDay)
	for _, day := range days {
		stops = append(stops, day.Stops...)
	}
	return stops
}
