package planner

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDistanceKM_LisbonToPortoIsAbout274Km(t *testing.T) {
	assert.InDelta(t, 274, DistanceKM(Place{Lat: 38.7223, Lon: -9.1393}, Place{Lat: 41.1496, Lon: -8.611}), 2.74)
}
func TestDistanceKM_SamePlaceIsZero(t *testing.T) {
	assert.Zero(t, DistanceKM(Place{Lat: 38, Lon: -9}, Place{Lat: 38, Lon: -9}))
}
func TestWalkingLoop_StartsAtTheMostFamousStop(t *testing.T) {
	g := WalkingLoop([]Place{{ID: "Q1", Sitelinks: 1}, {ID: "Q2", Sitelinks: 20}})
	require.Len(t, g.Stops, 2)
	assert.Equal(t, "Q2", g.Stops[0].ID)
}
func TestWalkingLoop_BreaksFameTiesByLowerID(t *testing.T) {
	g := WalkingLoop([]Place{{ID: "Q2"}, {ID: "Q1"}})
	require.Len(t, g.Stops, 2)
	assert.Equal(t, "Q1", g.Stops[0].ID)
}
func TestWalkingLoop_VisitsEveryStopOnce(t *testing.T) {
	s := []Place{{ID: "Q3", Lon: 2}, {ID: "Q1"}, {ID: "Q2", Lon: 1}}
	assert.ElementsMatch(t, s, WalkingLoop(s).Stops)
}
func TestWalkingLoop_WalkKMIncludesTheWayBack(t *testing.T) {
	a, b := Place{ID: "Q1"}, Place{ID: "Q3", Lon: 2}
	assert.InDelta(t, 444.78, WalkingLoop([]Place{a, {ID: "Q2", Lon: 1}, b}).WalkKM, 0.1)
}
func TestWalkingLoop_SingleStopWalksZeroKM(t *testing.T) {
	g := WalkingLoop([]Place{{ID: "Q1"}})
	require.Len(t, g.Stops, 1)
	assert.Zero(t, g.WalkKM)
}
func TestMedoid_PicksThePlaceClosestToAllOthers(t *testing.T) {
	assert.Equal(t, "Q2", Medoid([]Place{{ID: "Q1"}, {ID: "Q2", Lon: 1}, {ID: "Q3", Lon: 3}}).ID)
}
func TestMedoid_BreaksTiesByLowerID(t *testing.T) {
	assert.Equal(t, "Q1", Medoid([]Place{{ID: "Q2", Lon: 1}, {ID: "Q1"}}).ID)
}
func TestImageURL_BuildsACommonsThumbnailURL(t *testing.T) {
	assert.Equal(t, "https://commons.wikimedia.org/wiki/Special:FilePath/Torre_de_Bel%C3%A9m.jpg?width=400", (Place{Image: "Torre de Belém.jpg"}).ImageURL(400))
}
func TestImageURL_IsEmptyWithoutAnImage(t *testing.T) { assert.Empty(t, (Place{}).ImageURL(400)) }
func TestImagePageURL_LinksToTheCommonsFilePage(t *testing.T) {
	assert.Equal(t, "https://commons.wikimedia.org/wiki/File:Torre_de_Bel%C3%A9m.jpg", (Place{Image: "Torre de Belém.jpg"}).ImagePageURL())
}
