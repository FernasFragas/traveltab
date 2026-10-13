package writermap_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"weatherservice/internal/adapters/sqlite"
	"weatherservice/internal/writermap"
	"weatherservice/internal/writermap/writermaptest"
)

// As a traveler, I want indexed places to remain searchable while an interrupted crawl is
// retried, so that a failed sync never makes the current index disappear.
func TestResilience_InterruptedSyncKeepsSearchesWorkingAndResumes(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "writers.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	wp := newFakeWordPress()
	wp.addBlog(blogAPI, testPosts(2)...)
	index := &failAfterFirstPost{SyncIndex: storeIndex{store}}
	job := newTestJob(index, wp, crawlTime, activeSource(blogHost, blogAPI))

	require.ErrorContains(t, job.Run(ctx), "interrupted crawl")
	assert.Len(t, writerPostIDs(t, store), 1)
	assert.True(t, storeHasPhrase(t, store, "pico ruivo"), "the indexed post remains searchable")
	state, err := store.GetWriterSyncState(ctx, blogHost)
	require.NoError(t, err)
	assert.True(t, state.LastFull.IsZero(), "an interrupted run is due again")

	job = newTestJob(storeIndex{store}, wp, crawlTime, activeSource(blogHost, blogAPI))
	require.NoError(t, job.Run(ctx))
	assert.Equal(t, []int64{1, 2}, writerPostIDs(t, store), "restart upserts without duplicates")
	assert.True(t, storeHasPhrase(t, store, "pico ruivo"))
	state, err = store.GetWriterSyncState(ctx, blogHost)
	require.NoError(t, err)
	assert.Equal(t, crawlTime, state.LastFull)
}

// As a traveler, I want twenty searches for the same new city to share the Wikidata load,
// so that a burst of visits does not make twenty upstream queries.
func TestResilience_TwentyColdDestinationSearchesShareOneBaseQuery(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "writers.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	cliff := writermap.Place{QID: "Q1", Names: []string{"Cliff"}, Lat: 32.66, Lon: -17}
	base := &writermaptest.FakeBaseSource{Result: writermap.BasePlaces{
		Pins: []writermap.Place{cliff}, Gazetteer: []writermap.Place{cliff},
	}}
	cached := store.NewCachedBasePlaceSource(base)
	results := &concurrentDestinationStore{results: make(map[writermap.Area]writermap.DestinationResult)}
	source := writermap.NewMapSource(cached, emptyMatchIndex{}, results, nil, nil, writermap.Overrides{})
	area := writermap.Area{Name: "Funchal", Country: "pt", Lat: 32.65, Lon: -16.91}
	const searches = 20
	var wg sync.WaitGroup
	errs := make([]error, searches)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			result, err := source.Destination(ctx, area)
			if err == nil && (result == nil || len(result.Places) != 1) {
				err = errors.New("missing base pin")
			}
			errs[i] = err
		}(i)
	}
	wg.Wait()
	for _, err := range errs {
		assert.NoError(t, err)
	}
	assert.Equal(t, 1, base.PlacesCalls())
}

func writerPostIDs(t *testing.T, store *sqlite.Store) []int64 {
	t.Helper()
	ids, err := store.WriterPostIDs(context.Background(), blogHost)
	require.NoError(t, err)
	return ids
}

func storeHasPhrase(t *testing.T, store *sqlite.Store, phrase string) bool {
	t.Helper()
	rows, err := store.WriterMatchRows(context.Background(), []string{phrase})
	require.NoError(t, err)
	return len(rows) > 0
}

type failAfterFirstPost struct {
	writermap.SyncIndex
	calls int
}

func (f *failAfterFirstPost) UpsertPost(ctx context.Context, post writermap.IndexedPost) error {
	f.calls++
	if f.calls == 2 {
		return errors.New("interrupted crawl")
	}
	return f.SyncIndex.UpsertPost(ctx, post)
}

type emptyMatchIndex struct{}

func (emptyMatchIndex) WriterMatchRows(context.Context, []string) ([]writermap.MatchRow, error) {
	return nil, nil
}

type concurrentDestinationStore struct {
	mu      sync.Mutex
	results map[writermap.Area]writermap.DestinationResult
}

func (s *concurrentDestinationStore) WriterDestination(_ context.Context, area writermap.Area) (writermap.DestinationResult, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result, ok := s.results[area]
	return result, ok, nil
}

func (s *concurrentDestinationStore) SaveWriterDestination(_ context.Context, area writermap.Area, result writermap.DestinationResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results[area] = result
	return nil
}

func (s *concurrentDestinationStore) WriterDestinations(context.Context) ([]writermap.Area, error) {
	return nil, nil
}
