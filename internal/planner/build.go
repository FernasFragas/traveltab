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
	daysReorderedNote      = "Days reordered for the latest forecast."
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

	days := Schedule(groups, ranked, fc, req.Start, today)
	if daysReordered(groups, days) {
		notes = append(notes, daysReorderedNote)
	}

	plan := &Plan{
		Request:    req,
		Days:       days,
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

// daysReordered reports whether the forecast pulled a grouping away from the day index
// GroupDays naturally gave it. GroupDays never looks at the forecast, so groups[i] is always
// the grouping that would land on day i with no weather-driven reordering; assignGroups may
// move a grouping to a rainy day instead. A day is judged by whichever original grouping
// shares the most of its stops, since a rainy day's own content can still change a little
// (see topUpRainyDays) without that being a change of day order.
func daysReordered(groups []Group, days []Day) bool {
	for i, day := range days {
		if origin := originGroup(groups, day.Stops); origin >= 0 && origin != i {
			return true
		}
	}
	return false
}

// originGroup is the index of the group sharing the most stops with stops, or -1 when none of
// them share any - which Build never actually produces, since every day starts as a copy of
// one group and only ever loses or gains a few stops after that.
func originGroup(groups []Group, stops []Place) int {
	best, bestShared := -1, 0
	for i, g := range groups {
		if shared := sharedStopCount(g.Stops, stops); shared > bestShared {
			best, bestShared = i, shared
		}
	}
	return best
}

// sharedStopCount is how many of b's places also appear in a, compared by Wikidata ID.
func sharedStopCount(a, b []Place) int {
	ids := make(map[string]bool, len(a))
	for _, p := range a {
		ids[p.ID] = true
	}
	shared := 0
	for _, p := range b {
		if ids[p.ID] {
			shared++
		}
	}
	return shared
}

// planStops is every stop of the plan, so the center can be measured against all of them.
func planStops(days []Day) []Place {
	stops := make([]Place, 0, len(days)*MaxStopsPerDay)
	for _, day := range days {
		stops = append(stops, day.Stops...)
	}
	return stops
}
