package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"weatherservice/internal/application"
	"weatherservice/internal/writermap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var madeiraArea = writermap.Area{Name: "Madeira", Country: "Portugal", Lat: 32.76, Lon: -16.96, RadiusKM: 60}

type basePlaceSourceFake struct {
	mu      sync.Mutex
	calls   int
	result  writermap.BasePlaces
	err     error
	start   chan struct{}
	release chan struct{}
}

func (f *basePlaceSourceFake) Places(context.Context, writermap.Area) (writermap.BasePlaces, error) {
	f.mu.Lock()
	f.calls++
	result, err := f.result, f.err
	start, release := f.start, f.release
	f.mu.Unlock()
	if start != nil {
		select {
		case start <- struct{}{}:
		default:
		}
	}
	if release != nil {
		<-release
	}
	return result, err
}

func (f *basePlaceSourceFake) set(result writermap.BasePlaces, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.result, f.err = result, err
}

func (f *basePlaceSourceFake) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func basePlaceCacheForTest(t *testing.T) (*Store, application.BasePlaceSource, *basePlaceSourceFake) {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "baseplaces.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	fake := &basePlaceSourceFake{}
	return store, store.NewCachedBasePlaceSource(fake), fake
}

func basePlaces(t *testing.T, source application.BasePlaceSource, area writermap.Area) writermap.BasePlaces {
	t.Helper()
	places, err := source.Places(context.Background(), area)
	require.NoError(t, err)
	return places
}

func TestWikidataNearbyCache_CachesPinsAndGazetteerWithoutExpiry(t *testing.T) {
	store, source, fake := basePlaceCacheForTest(t)
	want := writermap.BasePlaces{
		Pins:      []writermap.Place{{QID: "Q1", Names: []string{"Pico Ruivo"}}},
		Gazetteer: []writermap.Place{{QID: "Q1"}, {QID: "Q2", Names: []string{"Câmara de Lobos"}}},
	}
	fake.set(want, nil)
	assert.Equal(t, want, basePlaces(t, source, madeiraArea))
	fake.set(writermap.BasePlaces{Pins: []writermap.Place{{QID: "Q3"}}}, nil)
	assert.Equal(t, want, basePlaces(t, source, madeiraArea))
	assert.Equal(t, want, basePlaces(t, store.NewCachedBasePlaceSource(fake), madeiraArea), "another decorator uses the persisted row")
	assert.Equal(t, 1, fake.count())
}

func TestWikidataNearbyCache_RefreshOverwritesTheRow(t *testing.T) {
	_, source, fake := basePlaceCacheForTest(t)
	first := writermap.BasePlaces{Pins: []writermap.Place{{QID: "Q1"}}, Gazetteer: []writermap.Place{{QID: "Q1"}}}
	second := writermap.BasePlaces{Pins: []writermap.Place{{QID: "Q2"}}, Gazetteer: []writermap.Place{{QID: "Q2"}, {QID: "Q3"}}}
	fake.set(first, nil)
	assert.Equal(t, first, basePlaces(t, source, madeiraArea))
	fake.set(second, nil)
	refresher := source.(application.BasePlaceRefresher)
	require.NoError(t, refresher.Refresh(context.Background(), madeiraArea))
	assert.Equal(t, second, basePlaces(t, source, madeiraArea))
	assert.Equal(t, 2, fake.count())
}

func TestWikidataNearbyCache_ColdErrorReturnsEmptyAndIsNotCached(t *testing.T) {
	_, source, fake := basePlaceCacheForTest(t)
	providerErr := errors.New("Wikidata unavailable")
	fake.set(writermap.BasePlaces{Pins: []writermap.Place{{QID: "Q1"}}}, providerErr)
	got, err := source.Places(context.Background(), madeiraArea)
	assert.ErrorIs(t, err, providerErr)
	assert.Empty(t, got.Pins)
	assert.Empty(t, got.Gazetteer)
	fake.set(writermap.BasePlaces{Pins: []writermap.Place{{QID: "Q2"}}}, nil)
	assert.Equal(t, "Q2", basePlaces(t, source, madeiraArea).Pins[0].QID)
	assert.Equal(t, 2, fake.count())
}

func TestWikidataNearbyCache_FailedRefreshPreservesTheCachedRow(t *testing.T) {
	_, source, fake := basePlaceCacheForTest(t)
	want := writermap.BasePlaces{Pins: []writermap.Place{{QID: "Q1"}}}
	fake.set(want, nil)
	basePlaces(t, source, madeiraArea)
	providerErr := errors.New("Wikidata unavailable")
	fake.set(writermap.BasePlaces{}, providerErr)
	err := source.(application.BasePlaceRefresher).Refresh(context.Background(), madeiraArea)
	assert.ErrorIs(t, err, providerErr)
	assert.Equal(t, want, basePlaces(t, source, madeiraArea))
	assert.Equal(t, 2, fake.count())
}

func TestWikidataNearbyCache_StoresEmptyResults(t *testing.T) {
	_, source, fake := basePlaceCacheForTest(t)
	basePlaces(t, source, madeiraArea)
	basePlaces(t, source, madeiraArea)
	assert.Equal(t, 1, fake.count())
}

func TestWikidataNearbyCache_KeepsDestinationsAndRadiiSeparate(t *testing.T) {
	_, source, fake := basePlaceCacheForTest(t)
	for _, area := range []writermap.Area{
		madeiraArea,
		{Name: "Madeira", Country: "Portugal", Lat: 32.76, Lon: -16.96, RadiusKM: 40},
		{Name: "Madeira", Country: "Portugal", Lat: 32.80, Lon: -16.96, RadiusKM: 60},
		{Name: "Porto Santo", Country: "Portugal", Lat: 32.76, Lon: -16.96, RadiusKM: 60},
	} {
		basePlaces(t, source, area)
	}
	assert.Equal(t, 4, fake.count())
}

func TestWikidataNearbyCache_ConcurrentCallersShareOneQuery(t *testing.T) {
	_, source, fake := basePlaceCacheForTest(t)
	want := writermap.BasePlaces{Pins: []writermap.Place{{QID: "Q1"}}}
	fake.set(want, nil)
	fake.start = make(chan struct{}, 1)
	fake.release = make(chan struct{})
	const callers = 20
	var wg sync.WaitGroup
	results := make([]writermap.BasePlaces, callers)
	errs := make([]error, callers)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = source.Places(context.Background(), madeiraArea)
		}()
	}
	<-fake.start
	close(fake.release)
	wg.Wait()
	for i := range results {
		assert.NoError(t, errs[i])
		assert.Equal(t, want, results[i])
	}
	assert.Equal(t, 1, fake.count())
}

func TestStore_CachedDestinations_RefreshesTheSameRows(t *testing.T) {
	store, source, fake := basePlaceCacheForTest(t)
	porto := writermap.Area{Name: "Porto", Country: "Portugal", Lat: 41.1496, Lon: -8.611, RadiusKM: 40}
	basePlaces(t, source, madeiraArea)
	basePlaces(t, source, porto)
	_, err := store.db.Exec(`INSERT INTO source_cache (key, fetched_at, data) VALUES ('planner:museums', 0, x'00')`)
	require.NoError(t, err)

	areas, err := store.CachedDestinations(context.Background())
	require.NoError(t, err)
	require.Len(t, areas, 2, "only base-place rows are destinations")
	for _, area := range areas {
		require.NoError(t, source.(application.BasePlaceRefresher).Refresh(context.Background(), area))
	}

	var rows int
	require.NoError(t, store.db.QueryRow(`SELECT COUNT(*) FROM source_cache`).Scan(&rows))
	assert.Equal(t, 3, rows, "refreshing a listed area overwrites its row instead of adding one")
	assert.Equal(t, 4, fake.count())
	assert.ElementsMatch(t, []float64{madeiraArea.RadiusKM, porto.RadiusKM}, []float64{areas[0].RadiusKM, areas[1].RadiusKM})
}

func TestStore_CachedDestinations_EmptyCache(t *testing.T) {
	store, _, _ := basePlaceCacheForTest(t)
	areas, err := store.CachedDestinations(context.Background())
	require.NoError(t, err)
	assert.Empty(t, areas)
}
