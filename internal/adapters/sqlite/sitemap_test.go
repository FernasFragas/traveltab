package sqlite

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTemp(t *testing.T, path string) *Store {
	t.Helper()
	store, err := Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestSitemapSlugs_EmptyStore(t *testing.T) {
	store := openTemp(t, filepath.Join(t.TempDir(), "empty.db"))

	slugs, err := store.ListSitemapSlugs()

	require.NoError(t, err)
	assert.NotNil(t, slugs)
	assert.Empty(t, slugs)
}

func TestSitemapSlugs_PopulatedSortedDedupedAndPersisted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "populated.db")
	store := openTemp(t, path)
	for _, slug := range []string{"porto-pt", "lisbon-pt", "porto-pt", "", "faro-pt"} {
		require.NoError(t, store.RecordSitemapSlug(slug))
	}

	slugs, err := store.ListSitemapSlugs()
	require.NoError(t, err)
	assert.Equal(t, []string{"faro-pt", "lisbon-pt", "porto-pt"}, slugs)

	require.NoError(t, store.Close())
	reopened := openTemp(t, path)
	slugs, err = reopened.ListSitemapSlugs()
	require.NoError(t, err)
	assert.Equal(t, []string{"faro-pt", "lisbon-pt", "porto-pt"}, slugs, "slugs survive a restart")
}

func TestSitemapSlugs_ConcurrentWritesKeepEveryEntry(t *testing.T) {
	store := openTemp(t, filepath.Join(t.TempDir(), "concurrent.db"))
	const writers = 40

	var wg sync.WaitGroup
	errs := make(chan error, writers*2)
	for i := 0; i < writers; i++ {
		wg.Add(2)
		go func() { // distinct slugs must all survive
			defer wg.Done()
			errs <- store.RecordSitemapSlug(fmt.Sprintf("city-%02d-pt", i))
		}()
		go func() { // the same slug written by many callers is stored once
			defer wg.Done()
			errs <- store.RecordSitemapSlug("shared-pt")
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	slugs, err := store.ListSitemapSlugs()
	require.NoError(t, err)
	assert.Len(t, slugs, writers+1)
	assert.Contains(t, slugs, "shared-pt")
}

func TestSitemapSlugs_MigratesLegacyIndexEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	store := openTemp(t, path)
	require.NoError(t, store.SaveCityData(legacySitemapIndexKey, map[string]any{"slugs": []string{"tavira-pt", "lisbon-pt", "", "lisbon-pt"}}))
	require.NoError(t, store.SaveCityData("Lisbon", map[string]any{"name": "kept"}))
	require.NoError(t, store.Close())

	upgraded := openTemp(t, path)
	slugs, err := upgraded.ListSitemapSlugs()
	require.NoError(t, err)
	assert.Equal(t, []string{"lisbon-pt", "tavira-pt"}, slugs, "existing URLs are preserved")

	var count int
	require.NoError(t, upgraded.db.QueryRow(`SELECT COUNT(*) FROM city_data WHERE city = ?`, legacySitemapIndexKey).Scan(&count))
	assert.Zero(t, count, "the legacy row is removed once migrated")
	_, err = upgraded.GetCityData("Lisbon")
	assert.NoError(t, err, "real city rows are untouched")

	// A slug recorded after the upgrade and a second reopen still list together.
	require.NoError(t, upgraded.RecordSitemapSlug("faro-pt"))
	require.NoError(t, upgraded.Close())
	again := openTemp(t, path)
	slugs, err = again.ListSitemapSlugs()
	require.NoError(t, err)
	assert.Equal(t, []string{"faro-pt", "lisbon-pt", "tavira-pt"}, slugs)
}

func TestSitemapSlugs_UnreadableLegacyEntryDoesNotBlockOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrupt.db")
	store := openTemp(t, path)
	_, err := store.db.Exec(`REPLACE INTO city_data (city, data) VALUES (?, ?)`, legacySitemapIndexKey, []byte("not gzip"))
	require.NoError(t, err)
	require.NoError(t, store.Close())

	reopened := openTemp(t, path)
	slugs, err := reopened.ListSitemapSlugs()
	require.NoError(t, err)
	assert.Empty(t, slugs)
}
