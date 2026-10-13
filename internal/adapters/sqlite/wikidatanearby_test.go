package sqlite

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
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

func TestWikidataNearbyCache_IgnoresRowsFromBeforeThePhotoURLFix(t *testing.T) {
	store, source, fake := basePlaceCacheForTest(t)
	stale := writermap.BasePlaces{Pins: []writermap.Place{{QID: "Q1", PhotoURL: "https://commons.wikimedia.org/wiki/File:Pico_Ruivo.jpg"}}}
	fresh := writermap.BasePlaces{Pins: []writermap.Place{{QID: "Q1", PhotoURL: "https://upload.wikimedia.org/wikipedia/commons/thumb/a/aa/Pico_Ruivo.jpg/640px-Pico_Ruivo.jpg"}}}
	fake.set(stale, nil)
	basePlaces(t, source, madeiraArea)
	_, err := store.db.Exec(`UPDATE source_cache SET key = replace(key, 'wikidata:v3:', 'wikidata:v1:') WHERE key LIKE 'wikidata:v3:%'`)
	require.NoError(t, err)
	fake.set(fresh, nil)

	assert.Equal(t, fresh, basePlaces(t, source, madeiraArea), "a v1 row is not served; the source is asked again")
	areas, err := store.CachedDestinations(context.Background())
	require.NoError(t, err)
	assert.Len(t, areas, 1, "only the new row is listed for the monthly sync")
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

// pagedFake is a paged source whose pages are scripted. Each page adds one place; the load
// finishes after total pages. step receives the page about to be loaded so a test can block or fail it.
type pagedFake struct {
	mu      sync.Mutex
	froms   []writermap.BaseProgress
	total   int
	step    func(page int) error
	started chan struct{}
	ctxErrs []error
	mode    string
}

func (f *pagedFake) Places(context.Context, writermap.Area) (writermap.BasePlaces, error) {
	return writermap.BasePlaces{}, errors.New("paged source must be loaded with PlacesFrom")
}

func (f *pagedFake) PlacesFrom(ctx context.Context, _ writermap.Area, from writermap.BaseProgress, onPage func(writermap.BaseProgress)) (writermap.BaseProgress, error) {
	f.mu.Lock()
	f.froms = append(f.froms, from)
	f.mu.Unlock()
	if f.started != nil {
		select {
		case f.started <- struct{}{}:
		default:
		}
	}
	progress := from
	for progress.Ring < f.total {
		if f.step != nil {
			if err := f.step(progress.Ring); err != nil {
				f.mu.Lock()
				f.ctxErrs = append(f.ctxErrs, ctx.Err())
				f.mu.Unlock()
				return progress, err
			}
		}
		qid := "Q" + string(rune('1'+progress.Ring))
		place := writermap.Place{QID: qid, Names: []string{qid}}
		progress.Gazetteer = append(progress.Gazetteer, place)
		progress.Pins = append(progress.Pins, place)
		progress.Ring++
		progress.Mode = f.mode
		progress.Done = progress.Ring == f.total
		if onPage != nil {
			onPage(progress)
		}
	}
	f.mu.Lock()
	f.ctxErrs = append(f.ctxErrs, ctx.Err())
	f.mu.Unlock()
	return progress, nil
}

func (f *pagedFake) fromCalls() []writermap.BaseProgress {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]writermap.BaseProgress(nil), f.froms...)
}

func pagedCacheForTest(t *testing.T, fake *pagedFake) (*Store, *cachedBasePlaces) {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "paged.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	c := store.NewCachedBasePlaceSource(fake).(*cachedBasePlaces)
	c.retryDelay = 0
	return store, c
}

func waitDone(t *testing.T, c *cachedBasePlaces, area writermap.Area) writermap.BaseProgress {
	t.Helper()
	var progress writermap.BaseProgress
	require.Eventually(t, func() bool {
		var err error
		progress, err = c.Partial(context.Background(), area)
		require.NoError(t, err)
		return progress.Done
	}, 5*time.Second, 5*time.Millisecond)
	return progress
}

func TestWikidataNearbyCache_Partial_ServesThePinsLoadedSoFarWhileLoading(t *testing.T) {
	release := make(chan struct{})
	fake := &pagedFake{total: 3, started: make(chan struct{}, 1)}
	fake.step = func(page int) error {
		if page == 1 {
			<-release
		}
		return nil
	}
	_, c := pagedCacheForTest(t, fake)

	first, err := c.Partial(context.Background(), madeiraArea)
	require.NoError(t, err)
	<-fake.started
	require.Eventually(t, func() bool {
		p, _ := c.Partial(context.Background(), madeiraArea)
		return len(p.Pins) == 1
	}, 5*time.Second, 5*time.Millisecond, "the first page's pins are served while page 2 is still loading")
	assert.False(t, first.Done)
	close(release)
	final := waitDone(t, c, madeiraArea)
	assert.Len(t, final.Pins, 3)
}

func TestWikidataNearbyCache_Partial_StoresAPartialRowAfterEachPageButNeverAsComplete(t *testing.T) {
	release := make(chan struct{})
	fake := &pagedFake{total: 2}
	fake.step = func(page int) error {
		if page == 1 {
			<-release
		}
		return nil
	}
	store, c := pagedCacheForTest(t, fake)
	_, err := c.Partial(context.Background(), madeiraArea)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		var n int
		_ = store.db.QueryRow(`SELECT COUNT(*) FROM source_cache WHERE key LIKE 'wikidatapartial:%'`).Scan(&n)
		return n == 1
	}, 5*time.Second, 5*time.Millisecond)

	var complete int
	require.NoError(t, store.db.QueryRow(`SELECT COUNT(*) FROM source_cache WHERE key LIKE 'wikidata:v3:%'`).Scan(&complete))
	assert.Zero(t, complete, "a half-loaded city is not cached as complete")
	areas, err := store.CachedDestinations(context.Background())
	require.NoError(t, err)
	assert.Empty(t, areas, "the monthly sync does not list a partial row")
	close(release)
	waitDone(t, c, madeiraArea)
}

func TestWikidataNearbyCache_Partial_FinishedLoadIsCachedCompleteAndPartialRowIsRemoved(t *testing.T) {
	fake := &pagedFake{total: 2}
	store, c := pagedCacheForTest(t, fake)
	_, err := c.Partial(context.Background(), madeiraArea)
	require.NoError(t, err)

	final := waitDone(t, c, madeiraArea)

	assert.Len(t, final.Pins, 2)
	var complete, partial int
	require.Eventually(t, func() bool {
		require.NoError(t, store.db.QueryRow(`SELECT COUNT(*) FROM source_cache WHERE key LIKE 'wikidata:v3:%'`).Scan(&complete))
		require.NoError(t, store.db.QueryRow(`SELECT COUNT(*) FROM source_cache WHERE key LIKE 'wikidatapartial:%'`).Scan(&partial))
		return complete == 1 && partial == 0
	}, 5*time.Second, 5*time.Millisecond)
	got, err := c.Places(context.Background(), madeiraArea)
	require.NoError(t, err)
	assert.Len(t, got.Pins, 2)
	assert.Len(t, fake.fromCalls(), 1, "a later read is a cache hit")
}

func TestWikidataNearbyCache_Partial_LoadContinuesAfterTheRequestIsCancelled(t *testing.T) {
	release := make(chan struct{})
	fake := &pagedFake{total: 2}
	fake.step = func(page int) error {
		if page == 1 {
			<-release
		}
		return nil
	}
	_, c := pagedCacheForTest(t, fake)
	ctx, cancel := context.WithCancel(context.Background())
	_, err := c.Partial(ctx, madeiraArea)
	require.NoError(t, err)
	cancel()
	close(release)

	final := waitDone(t, c, madeiraArea)

	assert.Len(t, final.Pins, 2)
	fake.mu.Lock()
	defer fake.mu.Unlock()
	for _, err := range fake.ctxErrs {
		assert.NoError(t, err, "the load's context is not the request's")
	}
}

func TestWikidataNearbyCache_Partial_FailedLoadResumesFromTheStoredPage(t *testing.T) {
	var mu sync.Mutex
	fail := true
	fake := &pagedFake{total: 3}
	fake.step = func(page int) error {
		mu.Lock()
		defer mu.Unlock()
		if page == 1 && fail {
			return errors.New("wikidata 502")
		}
		return nil
	}
	_, c := pagedCacheForTest(t, fake)
	now := time.Now()
	c.cache.now = func() time.Time { return now }
	_, err := c.Partial(context.Background(), madeiraArea)
	require.NoError(t, err)
	var failed writermap.BaseProgress
	require.Eventually(t, func() bool {
		failed, err = c.Partial(context.Background(), madeiraArea)
		return err != nil
	}, 5*time.Second, 5*time.Millisecond)
	assert.ErrorContains(t, err, "wikidata 502")
	assert.Len(t, failed.Pins, 1, "the page loaded before the failure is still served")

	mu.Lock()
	fail = false
	mu.Unlock()
	now = now.Add(time.Minute)
	resumed, err := c.Partial(context.Background(), madeiraArea)
	require.NoError(t, err)
	assert.Len(t, resumed.Pins, 1, "the retry starts from the stored page")
	final := waitDone(t, c, madeiraArea)

	assert.Len(t, final.Pins, 3)
	froms := fake.fromCalls()
	require.Len(t, froms, basePlaceAttempts+1, "the first load tried its attempts, the second load resumed once")
	assert.Equal(t, 1, froms[basePlaceAttempts].Ring, "the second load resumed after the first stored page")
}

func TestWikidataNearbyCache_Partial_RetriesAFailedPageBeforeGivingUp(t *testing.T) {
	var mu sync.Mutex
	failures := 2
	fake := &pagedFake{total: 3}
	fake.step = func(page int) error {
		mu.Lock()
		defer mu.Unlock()
		if page == 1 && failures > 0 {
			failures--
			return errors.New("wikidata 504")
		}
		return nil
	}
	_, c := pagedCacheForTest(t, fake)

	final, err := c.Places(context.Background(), madeiraArea)

	require.NoError(t, err, "two transient failures are absorbed by the retries")
	assert.Len(t, final.Pins, 3)
	froms := fake.fromCalls()
	require.Len(t, froms, 3)
	assert.Equal(t, 1, froms[1].Ring, "each retry resumes after the last stored page")
	assert.Equal(t, 1, froms[2].Ring)
}

func TestWikidataNearbyCache_Partial_FreshFailureIsNotRetriedStraightAway(t *testing.T) {
	fake := &pagedFake{total: 2}
	fake.step = func(int) error { return errors.New("wikidata 429") }
	_, c := pagedCacheForTest(t, fake)
	now := time.Now()
	c.cache.now = func() time.Time { return now }
	_, err := c.Partial(context.Background(), madeiraArea)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		_, err = c.Partial(context.Background(), madeiraArea)
		return err != nil
	}, 5*time.Second, 5*time.Millisecond)

	for i := 0; i < 3; i++ {
		_, err = c.Partial(context.Background(), madeiraArea)
		assert.Error(t, err)
	}

	assert.Len(t, fake.fromCalls(), basePlaceAttempts, "polls inside the backoff do not start another load")
}

func TestWikidataNearbyCache_Partial_OneLoadPerDestination(t *testing.T) {
	release := make(chan struct{})
	fake := &pagedFake{total: 2, started: make(chan struct{}, 1)}
	fake.step = func(page int) error {
		if page == 0 {
			<-release
		}
		return nil
	}
	_, c := pagedCacheForTest(t, fake)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = c.Partial(context.Background(), madeiraArea)
		}()
	}
	wg.Wait()
	<-fake.started
	close(release)
	waitDone(t, c, madeiraArea)

	assert.Len(t, fake.fromCalls(), 1)
}

func TestWikidataNearbyCache_Places_WaitsForAPagedLoadAndReturnsTheFinishedResult(t *testing.T) {
	fake := &pagedFake{total: 3}
	_, c := pagedCacheForTest(t, fake)

	got, err := c.Places(context.Background(), madeiraArea)

	require.NoError(t, err)
	assert.Len(t, got.Pins, 3)
}

func TestWikidataNearbyCache_Places_CancelledWaiterLeavesTheLoadRunning(t *testing.T) {
	release := make(chan struct{})
	fake := &pagedFake{total: 2, started: make(chan struct{}, 1)}
	fake.step = func(page int) error {
		if page == 0 {
			<-release
		}
		return nil
	}
	_, c := pagedCacheForTest(t, fake)
	ctx, cancel := context.WithCancel(context.Background())
	errc := make(chan error, 1)
	go func() {
		_, err := c.Places(ctx, madeiraArea)
		errc <- err
	}()
	<-fake.started
	cancel()
	require.ErrorIs(t, <-errc, context.Canceled)
	close(release)

	final := waitDone(t, c, madeiraArea)

	assert.Len(t, final.Pins, 2)
	assert.Len(t, fake.fromCalls(), 1)
}

func TestWikidataNearbyCache_Partial_UnpagedSourceLoadsInTheBackground(t *testing.T) {
	_, source, fake := basePlaceCacheForTest(t)
	fake.set(writermap.BasePlaces{Pins: []writermap.Place{{QID: "Q1"}}, Gazetteer: []writermap.Place{{QID: "Q1"}}}, nil)

	final := waitDone(t, source.(*cachedBasePlaces), madeiraArea)

	assert.Len(t, final.Pins, 1)
	assert.Equal(t, 1, fake.count())
}

func TestWikidataNearbyCache_Refresh_PagedSourceReplacesTheCompleteRow(t *testing.T) {
	fake := &pagedFake{total: 2}
	_, c := pagedCacheForTest(t, fake)
	_, err := c.Places(context.Background(), madeiraArea)
	require.NoError(t, err)
	fake.total = 3

	require.NoError(t, c.Refresh(context.Background(), madeiraArea))

	got, err := c.Places(context.Background(), madeiraArea)
	require.NoError(t, err)
	assert.Len(t, got.Pins, 3)
	assert.Zero(t, fake.fromCalls()[1].Ring, "a refresh starts from the first ring")
}

func TestWikidataNearbyCache_FailureLogsDoNotPrintTheWikidataError(t *testing.T) {
	var logged strings.Builder
	log.SetOutput(&logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	fake := &pagedFake{total: 2}
	fake.step = func(int) error {
		return errors.New(`Get "https://query.wikidata.org/sparql?query=SELECT+secret": timeout`)
	}
	_, c := pagedCacheForTest(t, fake)

	_, err := c.Places(context.Background(), madeiraArea)

	require.Error(t, err)
	assert.Contains(t, logged.String(), "attempt 1 failed")
	assert.NotContains(t, logged.String(), "query.wikidata.org")
	assert.NotContains(t, logged.String(), "SELECT")
}

func TestWikidataNearbyCache_Partial_StoredProgressKeepsTheLoadMode(t *testing.T) {
	fake := &pagedFake{total: 3, mode: "dense"}
	fake.step = func(page int) error {
		if page == 1 && len(fake.fromCalls()) == 1 {
			return errors.New("wikidata 502")
		}
		return nil
	}
	_, c := pagedCacheForTest(t, fake)

	got, err := c.Places(context.Background(), madeiraArea)

	require.NoError(t, err)
	assert.Len(t, got.Pins, 3)
	froms := fake.fromCalls()
	require.Len(t, froms, 2)
	assert.Equal(t, "dense", froms[1].Mode, "the retry resumed with the stored mode")
}
