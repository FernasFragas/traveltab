package sqlite

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
	"weatherservice/internal/application"
)

// clearVisits empties the visits table. Tracking writes synchronously, so no earlier test
// can add rows afterwards.
func clearVisits(t *testing.T) {
	t.Helper()

	_, err := db.Exec(`DELETE FROM visits`)
	require.NoError(t, err)
}

// insertVisit stores a visit at a fixed time, the same way SaveVisit stores the current time.
func insertVisit(t *testing.T, at time.Time, visitor, path, city string) {
	t.Helper()

	_, err := db.Exec(`INSERT INTO visits (visited_at, visitor, path, city) VALUES (?, ?, ?, ?)`, at.UTC(), visitor, path, city)
	require.NoError(t, err)
}

// Tests for GetVisitStats
func TestGetVisitStats(t *testing.T) {
	clearVisits(t)

	noon := time.Now().UTC().Truncate(24 * time.Hour).Add(12 * time.Hour)
	twoDaysAgo, yesterday := noon.AddDate(0, 0, -2), noon.AddDate(0, 0, -1)

	insertVisit(t, twoDaysAgo, "a", "/", "")
	insertVisit(t, twoDaysAgo, "a", "/process-form/", "porto")
	insertVisit(t, twoDaysAgo, "b", "/", "")
	insertVisit(t, yesterday, "a", "/", "")
	insertVisit(t, yesterday, "c", "/process-form/", "porto")
	insertVisit(t, yesterday, "c", "/process-form/", "faro")
	// Outside the 7-day window.
	insertVisit(t, noon.AddDate(0, 0, -8), "d", "/process-form/", "lisbon")

	stats, err := testStore.GetVisitStats(noon.AddDate(0, 0, -7))
	require.NoError(t, err)

	assert.Equal(t, 3, stats.UniqueVisitors)
	assert.Equal(t, 6, stats.PageViews)
	assert.Equal(t, 3, stats.Searches)
	assert.Equal(t, []application.DailyStats{
		{Day: twoDaysAgo.Format("2006-01-02"), UniqueVisitors: 2, PageViews: 3},
		{Day: yesterday.Format("2006-01-02"), UniqueVisitors: 2, PageViews: 3},
	}, stats.Daily)
	assert.Equal(t, []application.CityStats{
		{City: "porto", Searches: 2},
		{City: "faro", Searches: 1},
	}, stats.TopCities)
}

func TestGetVisitStats_NoVisits(t *testing.T) {
	clearVisits(t)

	stats, err := testStore.GetVisitStats(time.Now().AddDate(0, 0, -7))
	require.NoError(t, err)

	assert.Zero(t, stats.UniqueVisitors)
	assert.Zero(t, stats.PageViews)
	// Empty lists rather than nil, so the JSON has [] instead of null.
	assert.NotNil(t, stats.Daily)
	assert.NotNil(t, stats.TopCities)
}

func TestGetVisitStats_NonUTCSince(t *testing.T) {
	clearVisits(t)

	visitedAt := time.Now().UTC().Add(-2 * time.Hour)
	insertVisit(t, visitedAt, "a", "/", "")

	// The same instant must count the same visits whatever time zone it is expressed in.
	since := visitedAt.Add(-time.Hour)
	for _, loc := range []*time.Location{time.UTC, time.FixedZone("UTC+5", 5*60*60), time.FixedZone("UTC-8", -8*60*60)} {
		stats, err := testStore.GetVisitStats(since.In(loc))
		require.NoError(t, err)
		assert.Equal(t, 1, stats.PageViews, "since expressed in %s", loc)
	}
}
