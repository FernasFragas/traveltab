package planner

import (
	"sort"
	"time"
)

// Schedule dates the days and lets the forecast decide which group goes on which day. Day i is
// start plus i days in the forecast's time zone. Only the days the forecast can be trusted for
// are rearranged, and a rainy day trades outdoor stops for indoor ones from the pool.
func Schedule(groups []Group, pool []Place, fc *Forecast, start, today time.Time) []Day {
	if len(groups) == 0 {
		return nil
	}
	days := dateDays(len(groups), fc, start, today)
	assignGroups(days, groups)
	topUpRainyDays(days, pool)
	return days
}

// dateDays dates every day of the trip and says how certain and how wet each one is.
func dateDays(count int, fc *Forecast, start, today time.Time) []Day {
	loc := forecastLocation(fc)
	y, m, d := start.Date()
	days := make([]Day, count)
	for i := range days {
		date := time.Date(y, m, d+i, 0, 0, 0, 0, loc)
		ahead := daysBetween(today, date)
		day := Day{Date: date, Forecast: forecastFor(fc, date), Certain: ahead >= 0 && ahead < ForecastTrustDays}
		day.Rainy = day.Certain && day.Forecast != nil && day.Forecast.RainMM != nil && *day.Forecast.RainMM >= RainyDayMM
		days[i] = day
	}
	return days
}

// assignGroups gives every day a group, moving the most indoor groups to the rainy dates.
// Days the forecast can't be trusted for keep the group they started with.
func assignGroups(days []Day, groups []Group) {
	for i := range days {
		days[i].Stops = append([]Place(nil), groups[i].Stops...)
		days[i].WalkKM = groups[i].WalkKM
	}
	var rainy, dry []int
	var plans []dayPlan
	for i := range days {
		if !days[i].Certain {
			continue
		}
		if days[i].Rainy {
			rainy = append(rainy, i)
		} else {
			dry = append(dry, i)
		}
		plans = append(plans, dayPlan{stops: days[i].Stops, walkKM: days[i].WalkKM, date: i})
	}
	if len(rainy) == 0 {
		return
	}
	sort.SliceStable(plans, func(a, b int) bool { return shelteredCount(plans[a].stops) > shelteredCount(plans[b].stops) })
	for n, i := range rainy {
		days[i].Stops, days[i].WalkKM = plans[n].stops, plans[n].walkKM
	}
	rest := plans[len(rainy):]
	sort.SliceStable(rest, func(a, b int) bool { return rest[a].date < rest[b].date })
	for n, i := range dry {
		days[i].Stops, days[i].WalkKM = rest[n].stops, rest[n].walkKM
	}
}

// dayPlan is one day's stops while they move between dates.
type dayPlan struct {
	stops  []Place
	walkKM float64
	date   int // the day it started on, so dry days keep their order
}

// topUpRainyDays trades a rainy day's least famous outdoor stops for the nearest unused indoor
// places, until the day has MinStopsPerDay of them or the pool runs out. No place is used twice.
func topUpRainyDays(days []Day, pool []Place) {
	used := make(map[string]bool)
	for i := range days {
		for _, s := range days[i].Stops {
			used[placeKey(s)] = true
		}
	}
	for i := range days {
		if !days[i].Rainy {
			continue
		}
		changed := false
		for shelteredCount(days[i].Stops) < MinStopsPerDay {
			next, found := nearestSheltered(days[i].Stops, pool, used)
			if !found {
				break
			}
			if out, ok := leastFamousOutdoor(days[i].Stops); ok {
				days[i].Stops = append(days[i].Stops[:out], days[i].Stops[out+1:]...)
			} else if len(days[i].Stops) >= MinStopsPerDay {
				break
			}
			used[placeKey(next)] = true
			days[i].Stops = append(days[i].Stops, next)
			changed = true
		}
		if changed {
			loop := WalkingLoop(days[i].Stops)
			days[i].Stops, days[i].WalkKM = loop.Stops, loop.WalkKM
		}
	}
}

// nearestSheltered picks the unused indoor or mixed place closest to the day's stops.
func nearestSheltered(stops, pool []Place, used map[string]bool) (Place, bool) {
	best, bestKM, found := Place{}, 0.0, false
	for _, c := range pool {
		if used[placeKey(c)] || !sheltered(c) {
			continue
		}
		km := nearestStopKM(c, stops)
		if !found || km < bestKM || km == bestKM && moreFamous(c, best) {
			best, bestKM, found = c, km, true
		}
	}
	return best, found
}

// leastFamousOutdoor finds the stop a rainy day gives up first.
func leastFamousOutdoor(stops []Place) (int, bool) {
	worst, found := 0, false
	for i, s := range stops {
		if sheltered(s) {
			continue
		}
		if !found || moreFamous(stops[worst], s) {
			worst, found = i, true
		}
	}
	return worst, found
}

func nearestStopKM(p Place, stops []Place) float64 {
	nearest := 0.0
	for i, s := range stops {
		if km := DistanceKM(p, s); i == 0 || km < nearest {
			nearest = km
		}
	}
	return nearest
}

func sheltered(p Place) bool { return p.Kind == Indoor || p.Kind == Mixed }

func shelteredCount(stops []Place) int {
	n := 0
	for _, s := range stops {
		if sheltered(s) {
			n++
		}
	}
	return n
}

// placeKey identifies a place, so no place is planned twice.
func placeKey(p Place) string { return p.ID + "|" + p.Name }

func forecastLocation(fc *Forecast) *time.Location {
	if fc == nil || fc.Timezone == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(fc.Timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

// forecastFor returns a copy of the forecast for that date, or nil when the forecast misses it.
func forecastFor(fc *Forecast, date time.Time) *DayForecast {
	if fc == nil {
		return nil
	}
	for _, f := range fc.Days {
		if sameDate(f.Date.In(date.Location()), date) {
			day := f
			return &day
		}
	}
	return nil
}

func sameDate(a, b time.Time) bool {
	y1, m1, d1 := a.Date()
	y2, m2, d2 := b.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// daysBetween counts whole calendar days from one date to the other, ignoring the clock.
func daysBetween(from, to time.Time) int {
	y1, m1, d1 := from.Date()
	y2, m2, d2 := to.Date()
	a := time.Date(y1, m1, d1, 0, 0, 0, 0, time.UTC)
	b := time.Date(y2, m2, d2, 0, 0, 0, 0, time.UTC)
	return int(b.Sub(a) / (24 * time.Hour))
}
