package planner

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

var rankTypes = map[string]Kind{"museum": Indoor, "park": Outdoor, "city": Deny}

func rankPlace(id string, fame int, types ...string) Place {
	return Place{ID: id, Sitelinks: fame, Types: types}
}
func TestRank_KeepsAPlaceWithAnIndoorType(t *testing.T) {
	got := Rank([]Place{rankPlace("Q1", 5, "museum")}, rankTypes, nil)
	require.Len(t, got, 1)
	assert.Equal(t, Indoor, got[0].Kind)
}
func TestRank_DropsAPlaceWithOnlyUnknownTypes(t *testing.T) {
	got := Rank([]Place{rankPlace("Q1", 5, "museum"), rankPlace("Q2", 8, "bridge")}, rankTypes, nil)
	require.Len(t, got, 1)
	assert.Equal(t, "Q1", got[0].ID)
}
func TestRank_DenyTypeBeatsIndoorType(t *testing.T) {
	got := Rank([]Place{rankPlace("Q1", 5, "museum"), rankPlace("Q2", 8, "museum", "city")}, rankTypes, nil)
	require.Len(t, got, 1)
	assert.Equal(t, "Q1", got[0].ID)
}
func TestRank_IndoorPlusOutdoorTypesMakeMixed(t *testing.T) {
	got := Rank([]Place{rankPlace("Q1", 5, "museum", "park")}, rankTypes, nil)
	require.Len(t, got, 1)
	assert.Equal(t, Mixed, got[0].Kind)
}
func TestRank_SortsByFameThenByID(t *testing.T) {
	got := Rank([]Place{rankPlace("Q3", 10, "park"), rankPlace("Q2", 20, "park"), rankPlace("Q1", 10, "park")}, rankTypes, nil)
	require.Len(t, got, 3)
	assert.Equal(t, []string{"Q2", "Q1", "Q3"}, ids(got))
}
func TestRank_ReturnsAtMostSixtyPlaces(t *testing.T) {
	var in []Place
	for i := 0; i < 70; i++ {
		in = append(in, rankPlace(fmt.Sprintf("Q%03d", i), 100-i, "park"))
	}
	assert.Len(t, Rank(in, rankTypes, nil), 60)
}
func TestRank_SetsKindOnEveryPlace(t *testing.T) {
	got := Rank([]Place{rankPlace("Q1", 5, "museum"), rankPlace("Q2", 8, "park")}, rankTypes, nil)
	require.Len(t, got, 2)
	for _, p := range got {
		assert.NotEmpty(t, p.Kind)
	}
}
func TestRank_BoostedPlaceSkipsTheTypeFilter(t *testing.T) {
	got := Rank([]Place{rankPlace("Q1", 5, "city")}, rankTypes, map[string]Kind{"Q1": Indoor})
	require.Len(t, got, 1)
	assert.Equal(t, "Q1", got[0].ID)
}
func TestRank_BoostedPlaceGetsTheBoostKind(t *testing.T) {
	got := Rank([]Place{rankPlace("Q1", 5, "museum")}, rankTypes, map[string]Kind{"Q1": Outdoor})
	require.Len(t, got, 1)
	assert.Equal(t, Outdoor, got[0].Kind)
}
func TestRank_BoostedPlaceRanksLikeTheTenthPlace(t *testing.T) {
	var in []Place
	for i := 0; i < 60; i++ {
		in = append(in, rankPlace(fmt.Sprintf("Q%03d", i), 100-i, "park"))
	}
	in = append(in, rankPlace("Q999", 7, "park"))
	got := Rank(in, rankTypes, map[string]Kind{"Q999": Indoor})
	require.Len(t, got, 60)
	assert.Contains(t, ids(got[:10]), "Q999")
	for _, p := range got {
		if p.ID == "Q999" {
			assert.Equal(t, 7, p.Sitelinks)
		}
	}
}
func TestRank_BoostWithFewerThanTenPlacesRanksLikeTheLastPlace(t *testing.T) {
	got := Rank([]Place{rankPlace("Q1", 20, "park"), rankPlace("Q2", 10, "park"), rankPlace("Q3", 1)}, rankTypes, map[string]Kind{"Q3": Indoor})
	require.Len(t, got, 3)
	assert.Equal(t, "Q1", got[0].ID)
	assert.Contains(t, ids(got[1:]), "Q3")
}
func TestRank_DoesNotChangeTheInput(t *testing.T) {
	in := []Place{rankPlace("Q2", 5, "museum"), rankPlace("Q1", 8, "park")}
	before := append([]Place(nil), in...)
	require.Len(t, Rank(in, rankTypes, nil), 2)
	assert.Equal(t, before, in)
}
func TestBoosts_HasTheFiveStartingPlaces(t *testing.T) {
	assert.Equal(t, map[string]Kind{"Q652806": Indoor, "Q168001": Outdoor, "Q2063403": Indoor, "Q23579173": Outdoor, "Q11650434": Indoor}, Boosts)
}
func ids(p []Place) []string {
	out := make([]string, len(p))
	for i := range p {
		out[i] = p[i].ID
	}
	return out
}
