package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"
	"weatherservice/internal/planner"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cachePlaceFake struct {
	calls int
	err   error
}

func (f *cachePlaceFake) PlacesNear(context.Context, float64, float64) ([]planner.Place, error) {
	f.calls++
	return []planner.Place{{ID: "Q1", Name: fmt.Sprintf("call %d", f.calls)}}, f.err
}

type cacheForecastFake struct{ calls int }

func (f *cacheForecastFake) DailyForecast(context.Context, float64, float64) (*planner.Forecast, error) {
	f.calls++
	return &planner.Forecast{Timezone: fmt.Sprintf("call %d", f.calls)}, nil
}

type cacheStayFake struct{}

func (cacheStayFake) StaysNear(context.Context, float64, float64) ([]planner.Stay, error) {
	return []planner.Stay{{Name: "a stay"}}, nil
}

func placeCacheForTest(t *testing.T) (*cachedPlaces, *cachePlaceFake, *time.Time) {
	t.Helper()
	f := &cachePlaceFake{}
	c := testStore.NewCachedPlaceSource(f).(*cachedPlaces)
	c.source = t.Name()
	now := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }
	return c, f, &now
}
func cachePlaceName(t *testing.T, c *cachedPlaces, lat float64) string {
	t.Helper()
	p, err := c.PlacesNear(context.Background(), lat, -9.1393)
	require.NoError(t, err)
	require.Len(t, p, 1)
	return p[0].Name
}
func TestSourceCache_ServesAFreshEntryFromTheCache(t *testing.T) {
	c, _, _ := placeCacheForTest(t)
	assert.Equal(t, "call 1", cachePlaceName(t, c, 38.7223))
	assert.Equal(t, "call 1", cachePlaceName(t, c, 38.7223))
}
func TestSourceCache_RefreshesAnExpiredEntry(t *testing.T) {
	c, _, now := placeCacheForTest(t)
	assert.Equal(t, "call 1", cachePlaceName(t, c, 38.7223))
	*now = now.Add(31 * 24 * time.Hour)
	assert.Equal(t, "call 2", cachePlaceName(t, c, 38.7223))
}
func TestSourceCache_ServesOldDataWhenTheRefreshFails(t *testing.T) {
	c, f, now := placeCacheForTest(t)
	assert.Equal(t, "call 1", cachePlaceName(t, c, 38.7223))
	*now = now.Add(31 * 24 * time.Hour)
	f.err = errors.New("offline")
	assert.Equal(t, "call 1", cachePlaceName(t, c, 38.7223))
}
func TestSourceCache_ReturnsTheErrorWhenNothingIsCached(t *testing.T) {
	c, f, _ := placeCacheForTest(t)
	f.err = errors.New("offline")
	_, err := c.PlacesNear(context.Background(), 38.7223, -9.1393)
	assert.ErrorIs(t, err, f.err)
}
func TestSourceCache_SharesEntriesForNearbyCoordinates(t *testing.T) {
	c, _, _ := placeCacheForTest(t)
	assert.Equal(t, "call 1", cachePlaceName(t, c, 38.72231))
	assert.Equal(t, "call 1", cachePlaceName(t, c, 38.72234))
}
func TestSourceCache_ExpiresForecastsAfterThreeHours(t *testing.T) {
	c := testStore.NewCachedForecastSource(&cacheForecastFake{}).(*cachedForecast)
	c.source = t.Name()
	now := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }
	read := func(want string) {
		t.Helper()
		f, err := c.DailyForecast(context.Background(), 35, 139)
		require.NoError(t, err)
		require.NotNil(t, f)
		assert.Equal(t, want, f.Timezone)
	}
	read("call 1")
	now = now.Add(2 * time.Hour)
	read("call 1")
	now = now.Add(time.Hour)
	read("call 2")
}
func TestSourceCache_KeepsSourcesApart(t *testing.T) {
	p := testStore.NewCachedPlaceSource(&cachePlaceFake{}).(*cachedPlaces)
	s := testStore.NewCachedStaySource(cacheStayFake{})
	assert.Equal(t, "call 1", cachePlaceName(t, p, 12.345))
	stays, err := s.StaysNear(context.Background(), 12.345, -9.1393)
	require.NoError(t, err)
	require.Len(t, stays, 1)
	assert.Equal(t, "a stay", stays[0].Name)
}
func sourceCacheTableName(t *testing.T, conn *sql.DB) string {
	t.Helper()
	var name string
	require.NoError(t, conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='source_cache'").Scan(&name))
	return name
}
func TestInitDB_CreatesTheSourceCacheTable(t *testing.T) {
	// The package database comes from Open in TestMain.
	assert.Equal(t, "source_cache", sourceCacheTableName(t, db))

	// A database file written before the planner existed opens and gets the table too.
	old, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "old.db"))
	require.NoError(t, err)
	defer func() { _ = old.Close() }()
	_, err = old.Exec(`CREATE TABLE city_data (city TEXT PRIMARY KEY, data BLOB);`)
	require.NoError(t, err)
	require.NoError(t, createSourceCacheTable(old))
	require.NoError(t, createSourceCacheTable(old)) // running twice is how a restart sees it
	assert.Equal(t, "source_cache", sourceCacheTableName(t, old))
}

type describedPlaceFake struct{ description string }

func (f describedPlaceFake) PlacesNear(context.Context, float64, float64) ([]planner.Place, error) {
	return []planner.Place{{ID: "Q1", Name: "Castle", Description: f.description}}, nil
}

func TestSourceCache_PlaceDescriptionSurvivesTheCache(t *testing.T) {
	c := testStore.NewCachedPlaceSource(describedPlaceFake{"castle in Tavira, Portugal"}).(*cachedPlaces)
	c.source = t.Name()

	// The second call is answered from the stored row, which a failing source cannot replace.
	_, err := c.PlacesNear(context.Background(), 37.1, -7.6)
	require.NoError(t, err)
	c.fetch = func(context.Context, float64, float64) ([]planner.Place, error) { return nil, errors.New("offline") }

	places, err := c.PlacesNear(context.Background(), 37.1, -7.6)
	require.NoError(t, err)
	require.Len(t, places, 1)
	assert.Equal(t, "castle in Tavira, Portugal", places[0].Description)
}

func TestSourceCache_RowsCachedBeforeDescriptionsStillLoad(t *testing.T) {
	c := testStore.NewCachedPlaceSource(&cachePlaceFake{}).(*cachedPlaces)
	c.source = t.Name()

	// A row written before Description existed has no such key.
	raw, err := compressJSON([]map[string]any{{"ID": "Q1", "Name": "Old row"}})
	require.NoError(t, err)
	_, err = testStore.db.Exec(`REPLACE INTO source_cache (key, fetched_at, data) VALUES (?, ?, ?);`,
		c.key(37.1, -7.6), time.Now().Unix(), raw)
	require.NoError(t, err)

	places, err := c.PlacesNear(context.Background(), 37.1, -7.6)
	require.NoError(t, err)
	require.Len(t, places, 1)
	assert.Equal(t, "Old row", places[0].Name)
	assert.Empty(t, places[0].Description)
}
