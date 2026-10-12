package planner

import (
	"context"
	"fmt"
	"log"
	"math"
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
	groups, extras := seedGroups(req.Include, ranked, req.Days)
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
		BookingURL: BookingURL(req.City, req.Country, req.Start, req.Days, req.Travelers),
	}
	plan.AlsoSuggested = append(extras, droppedSeeds(req.Include, days, extras)...)

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

// seedGroups builds the walking days with the included places first: they are clustered into
// days (close ones together, at most MaxStopsPerDay each), each of those days is filled up to
// MinStopsPerDay with nearby ranked places, and any days left over come from the ranked
// places as usual. Included places the days have no room for come back as extras. With nothing
// included this is exactly GroupDays.
func seedGroups(include, ranked []Place, days int) (groups []Group, extras []Place) {
	seeds := uniquePlaces(include)
	if len(seeds) == 0 {
		return GroupDays(ranked, days), nil
	}
	if len(seeds) > days*MaxStopsPerDay {
		seeds, extras = seeds[:days*MaxStopsPerDay], seeds[days*MaxStopsPerDay:]
	}

	k := dayCount(len(seeds), days)
	if need := (len(seeds) + MaxStopsPerDay - 1) / MaxStopsPerDay; k < need {
		k = need
	}
	clusters, centers := clusterPlaces(seeds, k)
	clusters = capClusters(clusters, centers)
	for i := 0; i < len(clusters); i++ {
		for len(clusters[i]) > 1 && WalkingLoop(clusters[i]).WalkKM > MaxWalkKM {
			at := leastFamousStop(clusters[i])
			moved := clusters[i][at]
			clusters[i] = append(clusters[i][:at], clusters[i][at+1:]...)
			if len(clusters) < days {
				clusters = append(clusters, []Place{moved})
			} else {
				extras = append(extras, moved)
			}
		}
	}

	used := make(map[string]bool)
	for _, p := range seeds {
		used[p.ID] = true
	}
	var rest []Place
	for _, p := range ranked {
		if !used[p.ID] {
			rest = append(rest, p)
		}
	}
	fillShortDays(clusters, rest)
	for _, stops := range clusters {
		if len(stops) == 0 {
			continue
		}
		for _, stop := range stops {
			used[stop.ID] = true
		}
		groups = append(groups, WalkingLoop(stops))
	}

	if len(groups) < days {
		var left []Place
		for _, p := range rest {
			if !used[p.ID] {
				left = append(left, p)
			}
		}
		groups = append(groups, GroupDays(left, days-len(groups))...)
	}
	return groups, extras
}

// uniquePlaces keeps the first of each ID, up to MaxIncluded places.
func uniquePlaces(places []Place) []Place {
	seen := make(map[string]bool, len(places))
	var unique []Place
	for _, p := range places {
		if seen[p.ID] || len(unique) == MaxIncluded {
			continue
		}
		seen[p.ID] = true
		unique = append(unique, p)
	}
	return unique
}

// capClusters moves the stop farthest from a day's centre out of any day with more than
// MaxStopsPerDay stops, into the nearest day that still has room.
func capClusters(clusters [][]Place, centers []Place) [][]Place {
	for i := range clusters {
		for len(clusters[i]) > MaxStopsPerDay {
			far := 0
			for s, p := range clusters[i] {
				if DistanceKM(p, centers[i]) > DistanceKM(clusters[i][far], centers[i]) {
					far = s
				}
			}
			moved := clusters[i][far]
			clusters[i] = append(clusters[i][:far], clusters[i][far+1:]...)
			target, distance := -1, math.Inf(1)
			for j := range clusters {
				if d := DistanceKM(moved, centers[j]); j != i && len(clusters[j]) < MaxStopsPerDay && d < distance {
					target, distance = j, d
				}
			}
			clusters[target] = append(clusters[target], moved)
		}
	}
	return clusters
}

// droppedSeeds are included places that made it into the groups but not into the final days
// (a rainy day trades outdoor stops for indoor ones), so they are suggested instead of lost.
func droppedSeeds(include []Place, days []Day, extras []Place) []Place {
	planned := make(map[string]bool)
	for _, day := range days {
		for _, stop := range day.Stops {
			planned[stop.ID] = true
		}
	}
	for _, p := range extras {
		planned[p.ID] = true
	}
	var dropped []Place
	for _, p := range uniquePlaces(include) {
		if !planned[p.ID] {
			planned[p.ID] = true
			dropped = append(dropped, p)
		}
	}
	return dropped
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
