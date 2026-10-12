package planner

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dayPlace(id string, fame int, lat, lon float64) Place {
	return Place{ID: id, Name: id, Lat: lat, Lon: lon, Sitelinks: fame, Kind: Outdoor}
}

// lisbonTop20 is Lisbon's famous places as Rank would hand them over: four walkable
// areas (Belém, Baixa, Alfama, Parque das Nações) of five places each, most famous first.
func lisbonTop20() []Place {
	return []Place{
		dayPlace("Q01", 120, 38.6916, -9.2160), // Belém: Torre de Belém
		dayPlace("Q02", 119, 38.7139, -9.1335), // Alfama: castle
		dayPlace("Q03", 118, 38.7633, -9.0939), // Parque das Nações: aquarium
		dayPlace("Q04", 117, 38.7075, -9.1364), // Baixa: riverside square
		dayPlace("Q05", 116, 38.6979, -9.2065), // Belém: monastery
		dayPlace("Q06", 115, 38.7098, -9.1330), // Alfama: cathedral
		dayPlace("Q07", 114, 38.7123, -9.1393), // Baixa: street lift
		dayPlace("Q08", 113, 38.7635, -9.0961), // Parque das Nações: science museum
		dayPlace("Q09", 112, 38.6936, -9.2057), // Belém: monument
		dayPlace("Q10", 111, 38.7150, -9.1249), // Alfama: pantheon
		dayPlace("Q11", 110, 38.7139, -9.1394), // Baixa: main square
		dayPlace("Q12", 109, 38.7717, -9.0947), // Parque das Nações: tower
		dayPlace("Q13", 108, 38.6957, -9.1937), // Belém: art museum
		dayPlace("Q14", 107, 38.7118, -9.1268), // Alfama: fado museum
		dayPlace("Q15", 106, 38.7118, -9.1403), // Baixa: convent ruins
		dayPlace("Q16", 105, 38.7681, -9.0950), // Parque das Nações: casino
		dayPlace("Q17", 104, 38.6963, -9.1997), // Belém: coach museum
		dayPlace("Q18", 103, 38.7178, -9.1316), // Alfama: viewpoint
		dayPlace("Q19", 102, 38.7071, -9.1458), // Baixa: market
		dayPlace("Q20", 101, 38.7597, -9.0961), // Parque das Nações: bridge viewpoint
	}
}

// lisbonArea is the walkable area a lisbonTop20 place belongs to: the fixture lays the
// four areas out in turn, Belém first.
func lisbonArea(id string) string {
	areas := []string{"Belém", "Alfama", "Nações", "Baixa"}
	n, err := strconv.Atoi(strings.TrimPrefix(id, "Q"))
	if err != nil || n < 1 {
		return id
	}
	return areas[(n-1)%len(areas)]
}

func allStopIDs(groups []Group) []string {
	var out []string
	for _, g := range groups {
		out = append(out, ids(g.Stops)...)
	}
	return out
}

func TestGroupDays_NeverMixesTwoFarApartAreas(t *testing.T) {
	west := []Place{
		dayPlace("W1", 90, 38.6916, -9.2160),
		dayPlace("W2", 80, 38.6979, -9.2065),
		dayPlace("W3", 70, 38.6936, -9.2057),
		dayPlace("W4", 60, 38.6963, -9.1997),
	}
	east := []Place{
		dayPlace("E1", 85, 38.7139, -9.1335),
		dayPlace("E2", 75, 38.7098, -9.1330),
		dayPlace("E3", 65, 38.7150, -9.1249),
		dayPlace("E4", 55, 38.7118, -9.1268),
	}
	groups := GroupDays([]Place{west[0], east[0], west[1], east[1], west[2], east[2], west[3], east[3]}, 2)

	require.Len(t, groups, 2)
	for _, g := range groups {
		areas := map[byte]bool{}
		for _, s := range g.Stops {
			areas[s.ID[0]] = true
		}
		assert.Len(t, areas, 1, "a day mixed the two areas: %v", ids(g.Stops))
	}
}

// TestGroupDays_EveryDayHasThreeToFourStopsUnlessNothingIsInReach checks that a
// day stays short only when no unused ranked place fits its walking loop.
func TestGroupDays_EveryDayHasThreeToFourStopsUnlessNothingIsInReach(t *testing.T) {
	ranked := append(lisbonTop20(), dayPlace("Q21", 1, 38.7755, -9.0947))
	groups := GroupDays(ranked, 5)

	require.Len(t, groups, 5)
	target := 0
	for _, g := range groups {
		assert.NotEmpty(t, g.Stops, "an empty day was kept")
		assert.LessOrEqual(t, len(g.Stops), MaxStopsPerDay, "day too long: %v", ids(g.Stops))
		if len(g.Stops) >= MinStopsPerDay {
			target++
			continue
		}
		for _, p := range ranked {
			if slices.Contains(allStopIDs(groups), p.ID) {
				continue
			}
			assert.True(t, nearestKM(p, g.Stops) > PullReachKM ||
				WalkingLoop(append(g.Stops, p)).WalkKM > MaxWalkKM,
				"day %v stayed short though unused %s fits its walking limit", ids(g.Stops), p.ID)
		}
	}
	assert.GreaterOrEqual(t, target, 3, "only %d of 5 days reached %d stops", target, MinStopsPerDay)
}

func TestGroupDays_SmallTownGetsFewerDays(t *testing.T) {
	var town []Place
	for i := 0; i < 7; i++ {
		town = append(town, dayPlace(fmt.Sprintf("T%d", i), 70-i, 37.12+float64(i%2)*0.02, -7.65+float64(i)*0.004))
	}

	assert.Len(t, GroupDays(town, 5), 2)
}

func TestGroupDays_OnePlaceGivesOneDay(t *testing.T) {
	groups := GroupDays([]Place{dayPlace("Q1", 5, 38.7, -9.1)}, 3)

	require.Len(t, groups, 1)
	assert.Equal(t, []string{"Q1"}, ids(groups[0].Stops))
}

func TestGroupDays_NoPlacesGivesNoDays(t *testing.T) {
	assert.Empty(t, GroupDays(nil, 3))
}

func TestGroupDays_DoesNotUseFarPlacesBelowTop20(t *testing.T) {
	pool := lisbonTop20()
	for i := 0; i < 4; i++ {
		pool = append(pool, dayPlace(fmt.Sprintf("X%d", i), 10-i, 38.75+float64(i)*0.01, -9.30))
	}

	for _, id := range allStopIDs(GroupDays(pool, 5)) {
		assert.NotContains(t, id, "X", "a place below the top 20 was used")
	}
}

func TestGroupDays_FillsAShortDayFromBeyondTheTop20(t *testing.T) {
	ranked := lisbonTop20()
	for i := 0; i < 4; i++ {
		ranked = append(ranked, dayPlace(fmt.Sprintf("X%d", i), 10-i, 39+float64(i), -10))
	}
	ranked = append(ranked, dayPlace("Q25", 1, 38.7755, -9.0947)) // 0.4 km from Q12

	groups := GroupDays(ranked, 5)

	require.Contains(t, allStopIDs(groups), "Q25")
	for _, g := range groups {
		if hasPlaceID(g.Stops, "Q12") {
			assert.GreaterOrEqual(t, len(g.Stops), MinStopsPerDay)
		}
	}
}

func TestGroupDays_TopsUpTwoStopDaysNotOnlySingletons(t *testing.T) {
	ranked := append(lisbonTop20(), dayPlace("Q21", 1, 38.7755, -9.0947))

	groups := GroupDays(ranked, 5)

	for _, g := range groups {
		if hasPlaceID(g.Stops, "Q12") {
			require.Len(t, g.Stops, MinStopsPerDay)
			assert.Contains(t, ids(g.Stops), "Q21")
			return
		}
	}
	t.Fatal("Q12 day missing")
}

func TestGroupDays_FillPrefersFamousPlacesWithinAShortWalk(t *testing.T) {
	stops := [][]Place{{dayPlace("Q1", 100, 0, 0), dayPlace("Q2", 90, 0, 0.001)}}
	ranked := []Place{
		dayPlace("Q3", 80, 0, 0.023), // more famous, 2.5 km away
		dayPlace("Q4", 70, 0, 0.010), // less famous, 1.0 km away
	}

	fillShortDays(stops, ranked)

	assert.Equal(t, []string{"Q1", "Q2", "Q4"}, ids(stops[0]))
}

func TestGroupDays_FillMeasuresFromTheNearestStop(t *testing.T) {
	stops := [][]Place{{dayPlace("Q1", 100, 0, 0), dayPlace("Q2", 90, 0, 0.026)}}
	ranked := []Place{dayPlace("Q3", 80, 0, 0.031)} // 0.6 km from Q2, 3.4 km from Q1

	fillShortDays(stops, ranked)

	assert.Equal(t, []string{"Q1", "Q2", "Q3"}, ids(stops[0]))
}

func TestGroupDays_FillsBeyondThreeKilometresWithinTheDailyLimit(t *testing.T) {
	stops := [][]Place{{dayPlace("Q1", 100, 0, 0), dayPlace("Q2", 90, 0, 0.001)}}
	ranked := []Place{
		dayPlace("Q3", 80, 0, 0.05), // 5.4 km away
		dayPlace("Q4", 85, 0, 0.15), // 16.6 km away
	}

	fillShortDays(stops, ranked)

	assert.Equal(t, []string{"Q1", "Q2", "Q3"}, ids(stops[0]))
}

func TestGroupDays_FillKeepsTheWalkingLoopWithinFifteenKilometres(t *testing.T) {
	stops := [][]Place{{dayPlace("Q1", 100, 0, 0), dayPlace("Q2", 90, 0, 0.001)}}
	ranked := []Place{dayPlace("Q3", 80, 0, 0.10)} // 11 km away, 22 km return walk

	fillShortDays(stops, ranked)

	assert.Equal(t, []string{"Q1", "Q2"}, ids(stops[0]))
}

func TestGroupDays_OneDayKeepsItsWalkWithinFifteenKilometres(t *testing.T) {
	ranked := []Place{dayPlace("Q1", 100, 0, 0), dayPlace("Q2", 90, 0, 0.10)}

	groups := GroupDays(ranked, 1)

	require.Len(t, groups, 1)
	assert.LessOrEqual(t, groups[0].WalkKM, 15.0)
	assert.Contains(t, ids(groups[0].Stops), "Q1")
}

func TestGroupDays_SwapsALoneStopForAnotherArea(t *testing.T) {
	ranked := []Place{
		dayPlace("A1", 100, 0, 0),
		dayPlace("L1", 99, 0, 1), // isolated, but famous enough to seed a day
		dayPlace("A2", 90, 0, 0.001),
		dayPlace("A3", 80, 0, 0.002),
		dayPlace("A4", 70, 0, 0.003),
		dayPlace("B1", 60, 0, 0.2),
		dayPlace("B2", 50, 0, 0.201),
		dayPlace("B3", 40, 0, 0.202),
	}

	groups := GroupDays(ranked, 2)

	require.Len(t, groups, 2)
	assert.NotContains(t, allStopIDs(groups), "L1")
	assert.Contains(t, allStopIDs(groups), "B1")
	for _, g := range groups {
		assert.GreaterOrEqual(t, len(g.Stops), 2)
	}
}

func TestGroupDays_DropsALoneDayWhenNoAreaCanFillIt(t *testing.T) {
	ranked := []Place{
		dayPlace("A1", 100, 0, 0),
		dayPlace("L1", 99, 0, 1),
		dayPlace("A2", 90, 0, 0.001),
		dayPlace("A3", 80, 0, 0.002),
		dayPlace("A4", 70, 0, 0.003),
		dayPlace("A5", 60, 0, 0.004),
	}

	groups := GroupDays(ranked, 2)

	require.Len(t, groups, 1)
	assert.NotContains(t, allStopIDs(groups), "L1")
	assert.Contains(t, allStopIDs(groups), "A1")
}

func TestGroupDays_KeepsAnIsolatedMostFamousPlace(t *testing.T) {
	ranked := []Place{
		dayPlace("L1", 100, 0, 1),
		dayPlace("A1", 90, 0, 0),
		dayPlace("A2", 80, 0, 0.001),
		dayPlace("A3", 70, 0, 0.002),
		dayPlace("A4", 60, 0, 0.003),
		dayPlace("A5", 50, 0, 0.004),
	}

	groups := GroupDays(ranked, 2)

	assert.Contains(t, allStopIDs(groups), "L1")
}

func TestGroupDays_NoPlaceAppearsTwice(t *testing.T) {
	seen := map[string]bool{}
	for _, id := range allStopIDs(GroupDays(lisbonTop20(), 5)) {
		assert.False(t, seen[id], "%s appears on two days", id)
		seen[id] = true
	}
	assert.NotEmpty(t, seen)
}

func TestGroupDays_OrdersEachDayAsAWalkingLoop(t *testing.T) {
	groups := GroupDays(lisbonTop20(), 4)

	require.Len(t, groups, 4)
	for _, g := range groups {
		assert.Equal(t, WalkingLoop(g.Stops), g, "a day is not in walking order")
	}
}

func TestGroupDays_SameInputGivesTheSameDays(t *testing.T) {
	first := GroupDays(lisbonTop20(), 4)

	require.NotEmpty(t, first)
	for i := 0; i < 5; i++ {
		assert.Equal(t, first, GroupDays(lisbonTop20(), 4))
	}
}

func TestGroupDays_LisbonLikeDaysWalkUnder15Km(t *testing.T) {
	groups := GroupDays(lisbonTop20(), 4)

	require.Len(t, groups, 4)
	for _, g := range groups {
		assert.Less(t, g.WalkKM, 15.0, "a day walks too far: %v", ids(g.Stops))
	}
}

func TestGroupDays_ShortTripsStayCompact(t *testing.T) {
	groups := GroupDays(lisbonTop20(), 2)

	require.Len(t, groups, 2)
	for _, g := range groups {
		assert.Less(t, g.WalkKM, 8.0, "a short-trip day walks too far: %v", ids(g.Stops))
		areas := map[string]bool{}
		for _, s := range g.Stops {
			areas[lisbonArea(s.ID)] = true
		}
		assert.False(t, areas["Belém"] && areas["Alfama"], "a day mixes two far-apart areas: %v", ids(g.Stops))
	}
}

func TestGroupDays_LongTripsStayWalkable(t *testing.T) {
	for _, days := range []int{4, 5} {
		groups := GroupDays(lisbonTop20(), days)

		require.Len(t, groups, days)
		for _, g := range groups {
			assert.LessOrEqual(t, g.WalkKM, 15.0, "a day of the %d-day plan walks too far: %v", days, ids(g.Stops))
		}
	}
}

func TestGroupDays_NoDayIsLeftWithASingleStop(t *testing.T) {
	// Refusing an overlong walk must not strand a day on its own: a day card with one stop
	// reads as broken. A day at MinStopsPerDay may give one up when the new walk fits.
	for days := 2; days <= MaxDays; days++ {
		groups := GroupDays(lisbonTop20(), days)

		for _, g := range groups {
			assert.GreaterOrEqual(t, len(g.Stops), 2, "a day of the %d-day plan is on its own: %v", days, ids(g.Stops))
		}
	}
}

func TestGroupDays_AlwaysKeepsTheMostFamousPlace(t *testing.T) {
	for days := 1; days <= MaxDays; days++ {
		assert.Contains(t, allStopIDs(GroupDays(lisbonTop20(), days)), "Q01", "the most famous place is missing from the %d-day plan", days)
	}
}
