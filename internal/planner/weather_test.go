package planner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scheduleStart is the trip's first day in every Schedule test.
var scheduleStart = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

func weatherPlace(id string, kind Kind, lat, lon float64) Place {
	return Place{ID: id, Name: id, Lat: lat, Lon: lon, Sitelinks: 10, Kind: kind}
}

// weatherGroup orders stops the way GroupDays does, so the tests start from real groups.
func weatherGroup(stops ...Place) Group { return WalkingLoop(stops) }

func rainMM(v float64) *float64 { return &v }

// weatherForecast holds one day per rain value, starting at start. A nil value means no data.
func weatherForecast(loc *time.Location, start time.Time, rain ...*float64) *Forecast {
	fc := &Forecast{Timezone: loc.String()}
	y, m, d := start.Date()
	for i, r := range rain {
		fc.Days = append(fc.Days, DayForecast{Date: time.Date(y, m, d+i, 0, 0, 0, 0, loc), RainMM: r})
	}
	return fc
}

func stopIDs(stops []Place) []string {
	out := make([]string, len(stops))
	for i := range stops {
		out[i] = stops[i].ID
	}
	return out
}

// shelteredStops counts the stops that work in the rain.
func shelteredStops(d Day) int {
	n := 0
	for _, s := range d.Stops {
		if s.Kind == Indoor || s.Kind == Mixed {
			n++
		}
	}
	return n
}

// outdoorDay and indoorDay are two groups far apart, one for the rain and one for the sun.
func outdoorDay() Group {
	return weatherGroup(weatherPlace("Q1", Outdoor, 0, 0), weatherPlace("Q2", Outdoor, 0, 0.01), weatherPlace("Q3", Outdoor, 0, 0.02))
}
func indoorDay() Group {
	return weatherGroup(weatherPlace("Q4", Indoor, 1, 0), weatherPlace("Q5", Indoor, 1, 0.01), weatherPlace("Q6", Indoor, 1, 0.02))
}

func TestSchedule_DatesFollowTheStartDate(t *testing.T) {
	days := Schedule([]Group{outdoorDay(), indoorDay(), outdoorDay()}, nil, nil, scheduleStart, scheduleStart)
	require.Len(t, days, 3)
	assert.Equal(t, "2026-03-01", days[0].Date.Format(time.DateOnly))
	assert.Equal(t, "2026-03-02", days[1].Date.Format(time.DateOnly))
	assert.Equal(t, "2026-03-03", days[2].Date.Format(time.DateOnly))
}

func TestSchedule_DatesUseTheForecastTimeZone(t *testing.T) {
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)
	days := Schedule([]Group{indoorDay()}, nil, weatherForecast(tokyo, scheduleStart, rainMM(0)), scheduleStart, scheduleStart)
	require.Len(t, days, 1)
	assert.Equal(t, "Asia/Tokyo", days[0].Date.Location().String())
	assert.Equal(t, "2026-03-01T00:00:00+09:00", days[0].Date.Format(time.RFC3339))
}

func TestSchedule_RainyDayGetsTheMostIndoorGroup(t *testing.T) {
	fc := weatherForecast(time.UTC, scheduleStart, rainMM(10), rainMM(0))
	days := Schedule([]Group{outdoorDay(), indoorDay()}, nil, fc, scheduleStart, scheduleStart)
	require.Len(t, days, 2)
	require.True(t, days[0].Rainy)
	assert.ElementsMatch(t, []string{"Q4", "Q5", "Q6"}, stopIDs(days[0].Stops))
	assert.ElementsMatch(t, []string{"Q1", "Q2", "Q3"}, stopIDs(days[1].Stops))
}

func TestSchedule_RainyDayIsToppedUpToThreeIndoorStops(t *testing.T) {
	// Kyoto-like: the day has a single indoor stop, and the rest of the pool has more.
	stops := []Place{weatherPlace("Q1", Indoor, 35, 135), weatherPlace("Q2", Outdoor, 35.001, 135), weatherPlace("Q3", Outdoor, 35.002, 135)}
	pool := append(append([]Place(nil), stops...), weatherPlace("Q8", Indoor, 35.003, 135), weatherPlace("Q9", Mixed, 35.004, 135))
	days := Schedule([]Group{weatherGroup(stops...)}, pool, weatherForecast(time.UTC, scheduleStart, rainMM(9)), scheduleStart, scheduleStart)
	require.Len(t, days, 1)
	require.True(t, days[0].Rainy)
	assert.Len(t, days[0].Stops, 3)
	assert.GreaterOrEqual(t, shelteredStops(days[0]), MinStopsPerDay)
}

func TestSchedule_TopUpUsesTheNearestUnusedIndoorPlaces(t *testing.T) {
	stops := []Place{weatherPlace("Q1", Indoor, 35, 135), weatherPlace("Q2", Outdoor, 35.001, 135), weatherPlace("Q3", Outdoor, 35.002, 135)}
	pool := append(append([]Place(nil), stops...),
		weatherPlace("Q8", Indoor, 35.003, 135), // about 0.1 km away
		weatherPlace("Q9", Indoor, 35.05, 135),  // about 5 km away
		weatherPlace("Q10", Indoor, 36, 135))    // about 110 km away
	days := Schedule([]Group{weatherGroup(stops...)}, pool, weatherForecast(time.UTC, scheduleStart, rainMM(9)), scheduleStart, scheduleStart)
	require.Len(t, days, 1)
	assert.ElementsMatch(t, []string{"Q1", "Q8", "Q9"}, stopIDs(days[0].Stops))
}

func TestSchedule_NoPlaceAppearsTwice(t *testing.T) {
	west := []Place{weatherPlace("Q1", Indoor, 35, 135), weatherPlace("Q2", Outdoor, 35.001, 135), weatherPlace("Q3", Outdoor, 35.002, 135)}
	east := []Place{weatherPlace("Q4", Indoor, 35, 136), weatherPlace("Q5", Outdoor, 35.001, 136), weatherPlace("Q6", Outdoor, 35.002, 136)}
	pool := append(append(append([]Place(nil), west...), east...),
		weatherPlace("Q7", Indoor, 35.003, 135.5), weatherPlace("Q8", Indoor, 35.004, 135.5),
		weatherPlace("Q9", Mixed, 35.005, 135.5), weatherPlace("Q10", Indoor, 35.006, 135.5))
	fc := weatherForecast(time.UTC, scheduleStart, rainMM(9), rainMM(9))
	days := Schedule([]Group{weatherGroup(west...), weatherGroup(east...)}, pool, fc, scheduleStart, scheduleStart)
	require.Len(t, days, 2)
	seen := map[string]bool{}
	for _, d := range days {
		for _, id := range stopIDs(d.Stops) {
			assert.False(t, seen[id], "place %s is planned twice", id)
			seen[id] = true
		}
	}
	assert.Len(t, seen, len(days[0].Stops)+len(days[1].Stops))
}

func TestSchedule_FiveMMIsRainy(t *testing.T) {
	days := Schedule([]Group{indoorDay()}, nil, weatherForecast(time.UTC, scheduleStart, rainMM(RainyDayMM)), scheduleStart, scheduleStart)
	require.Len(t, days, 1)
	assert.True(t, days[0].Rainy)
}

func TestSchedule_JustUnderFiveMMIsDry(t *testing.T) {
	days := Schedule([]Group{indoorDay()}, nil, weatherForecast(time.UTC, scheduleStart, rainMM(4.9)), scheduleStart, scheduleStart)
	require.Len(t, days, 1)
	assert.False(t, days[0].Rainy)
	assert.True(t, days[0].Certain)
	require.NotNil(t, days[0].Forecast)
	require.NotNil(t, days[0].Forecast.RainMM)
	assert.Equal(t, 4.9, *days[0].Forecast.RainMM)
}

func TestSchedule_MissingRainIsNotRainy(t *testing.T) {
	days := Schedule([]Group{indoorDay()}, nil, weatherForecast(time.UTC, scheduleStart, nil), scheduleStart, scheduleStart)
	require.Len(t, days, 1)
	assert.False(t, days[0].Rainy)
	require.NotNil(t, days[0].Forecast)
	assert.Nil(t, days[0].Forecast.RainMM)
}

func TestSchedule_DaysSevenOrMoreOutAreNotCertain(t *testing.T) {
	start := scheduleStart.AddDate(0, 0, 6)
	fc := weatherForecast(time.UTC, start, rainMM(10), rainMM(10))
	days := Schedule([]Group{indoorDay(), outdoorDay()}, nil, fc, start, scheduleStart)
	require.Len(t, days, 2)
	assert.True(t, days[0].Certain)
	assert.True(t, days[0].Rainy)
	assert.False(t, days[1].Certain)
	assert.False(t, days[1].Rainy)
	assert.NotNil(t, days[1].Forecast, "a less certain day still shows its forecast")
}

func TestSchedule_DaysSevenOrMoreOutAreNotRearranged(t *testing.T) {
	start := scheduleStart.AddDate(0, 0, 7)
	fc := weatherForecast(time.UTC, start, rainMM(10), rainMM(0))
	days := Schedule([]Group{outdoorDay(), indoorDay()}, nil, fc, start, scheduleStart)
	require.Len(t, days, 2)
	assert.Equal(t, stopIDs(outdoorDay().Stops), stopIDs(days[0].Stops))
	assert.Equal(t, stopIDs(indoorDay().Stops), stopIDs(days[1].Stops))
}

func TestSchedule_DryDaysKeepTheirStops(t *testing.T) {
	groups := []Group{outdoorDay(), indoorDay()}
	fc := weatherForecast(time.UTC, scheduleStart, rainMM(0), rainMM(1))
	days := Schedule(groups, nil, fc, scheduleStart, scheduleStart)
	require.Len(t, days, 2)
	for i, g := range groups {
		assert.Equal(t, stopIDs(g.Stops), stopIDs(days[i].Stops))
		assert.Equal(t, g.WalkKM, days[i].WalkKM)
		assert.False(t, days[i].Rainy)
	}
}

func TestSchedule_NilForecastKeepsTheOrderWithNoWeather(t *testing.T) {
	groups := []Group{outdoorDay(), indoorDay()}
	days := Schedule(groups, nil, nil, scheduleStart, scheduleStart)
	require.Len(t, days, 2)
	for i, g := range groups {
		assert.Equal(t, stopIDs(g.Stops), stopIDs(days[i].Stops))
		assert.Nil(t, days[i].Forecast)
		assert.False(t, days[i].Rainy)
	}
}
