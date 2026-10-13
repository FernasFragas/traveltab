package writermap

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapSource_DestinationMatchesStoresAndServesStored(t *testing.T) {
	ctx := context.Background()
	base, index, store := sourceTestBase(), sourceTestIndex(), newFakeDestinationStore()
	source := NewMapSource(base, index, store, fakeSyncStates{}, sourceTestSources(), Overrides{})
	area := Area{Name: "Funchal", Country: "pt", Lat: 32.65, Lon: -16.91}

	first, err := source.Destination(ctx, area)
	require.NoError(t, err)
	second, err := source.Destination(ctx, area)
	require.NoError(t, err)

	assert.Equal(t, 1, first.WriterCounts["Q1"])
	require.Len(t, first.Mentions["Q1"], 1)
	assert.Equal(t, "Writer", first.Mentions["Q1"][0].BlogName)
	require.Len(t, first.Itineraries, 1)
	assert.Equal(t, []ItineraryDay{{QIDs: []string{"Q1"}}}, first.Itineraries[0].Days)
	assert.Equal(t, first, second)
	assert.Equal(t, 1, base.calls)
	assert.Equal(t, 1, index.calls)
	assert.Contains(t, index.phrases, "monte")
	assert.Contains(t, index.phrases, "funchal")
	assert.Equal(t, *first, store.results[area])
}

func TestMapSource_ContextIncludesOverrideAreasAroundTheDestination(t *testing.T) {
	ctx := context.Background()
	post := IndexedPost{Host: "writer.test", PostID: 2, Title: "Funchal Madeira"}
	index := &fakeMatchIndex{rows: []MatchRow{
		{Post: post, Phrase: NamePhrase{Folded: "funchal", OriginalCase: true, Start: 0, End: 7}},
		{Post: post, Phrase: NamePhrase{Folded: "madeira", OriginalCase: true, Start: 8, End: 15}},
		{Post: post, Phrase: NamePhrase{Folded: "monte", OriginalCase: true, Start: 20, End: 25}},
	}}
	overrides := Overrides{Areas: []Area{
		{Name: "Madeira", Lat: 32.75, Lon: -16.96, RadiusKM: 70},
		{Name: "São Miguel", Lat: 37.78, Lon: -25.50, RadiusKM: 50},
	}}
	source := NewMapSource(sourceTestBase(), index, newFakeDestinationStore(), fakeSyncStates{}, sourceTestSources(), overrides)

	got, err := source.Destination(ctx, Area{Name: "Funchal", Lat: 32.65, Lon: -16.91})

	require.NoError(t, err)
	assert.Equal(t, 1, got.WriterCounts["Q1"])
	assert.Contains(t, index.phrases, "madeira")
	assert.NotContains(t, index.phrases, "sao miguel")
}

func TestMapSource_ContextNamesIncludeDestinationAliases(t *testing.T) {
	source := NewMapSource(nil, nil, nil, nil, nil, Overrides{
		MatchOverrides: MatchOverrides{DestinationAliases: map[string][]string{"Lisbon": {"Lisboa"}}},
	})

	assert.Equal(t, []string{"Lisbon", "Lisboa"}, source.contextNames(Area{Name: "Lisbon"}))
	assert.Equal(t, []string{"lisbon", "Lisboa"}, source.contextNames(Area{Name: "lisbon"}))
}

func TestMapSource_ReportsSourceStatusesAndCompleteness(t *testing.T) {
	ctx := context.Background()
	checked := time.Date(2026, 10, 1, 3, 0, 0, 0, time.UTC)
	later := checked.Add(24 * time.Hour)
	sources := []Source{
		{Host: "done.test", Status: StatusActive},
		{Host: "never.test", Status: StatusActive},
		{Host: "blocked.test", Status: StatusBlocked},
		{Host: "removed.test", Status: StatusRemoved},
	}
	states := fakeSyncStates{"done.test": {Host: "done.test", Status: StatusActive, LastFull: checked, LastDelta: later}}
	source := NewMapSource(sourceTestBase(), sourceTestIndex(), newFakeDestinationStore(), states, sources, Overrides{})

	got, err := source.Destination(ctx, Area{Name: "Funchal"})

	require.NoError(t, err)
	assert.False(t, got.Complete)
	require.Len(t, got.SourceStatuses, 4)
	assert.Equal(t, SourceStatus{Host: "done.test", Status: StatusActive, CheckedAt: &later}, got.SourceStatuses[0])
	assert.Equal(t, SourceStatus{Host: "never.test", Status: StatusActive}, got.SourceStatuses[1])
	assert.Equal(t, SourceStatus{Host: "blocked.test", Status: StatusBlocked}, got.SourceStatuses[2])
	assert.Equal(t, SourceStatus{Host: "removed.test", Status: StatusRemoved}, got.SourceStatuses[3])

	states["never.test"] = SyncState{Host: "never.test", Status: StatusActive, LastFull: checked}
	complete, err := NewMapSource(sourceTestBase(), sourceTestIndex(), newFakeDestinationStore(), states, sources, Overrides{}).
		Destination(ctx, Area{Name: "Funchal"})
	require.NoError(t, err)
	assert.True(t, complete.Complete)
}

func TestMapSource_PlaceReturnsPinAndOptionalSummary(t *testing.T) {
	ctx := context.Background()
	source := NewMapSource(sourceTestBase(), sourceTestIndex(), newFakeDestinationStore(), fakeSyncStates{}, sourceTestSources(), Overrides{})
	area := Area{Name: "Funchal"}

	place, summary, err := source.Place(ctx, area, "Q1")
	require.NoError(t, err)
	assert.Equal(t, "Monte", place.Names[0])
	assert.Nil(t, summary)

	source.Summaries = fakeSummaries{"Q1": {QID: "Q1", EnglishText: "A hilltop parish."}}
	_, summary, err = source.Place(ctx, area, "Q1")
	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, "A hilltop parish.", summary.EnglishText)

	_, _, err = source.Place(ctx, area, "Q404")
	assert.ErrorIs(t, err, ErrPlaceNotFound)
}

func TestMapSource_BaseErrorIsReturnedAndNotStored(t *testing.T) {
	ctx := context.Background()
	store := newFakeDestinationStore()
	base := &fakeBaseSource{err: errors.New("wikidata down")}
	source := NewMapSource(base, sourceTestIndex(), store, fakeSyncStates{}, sourceTestSources(), Overrides{})

	_, err := source.Destination(ctx, Area{Name: "Funchal"})

	require.Error(t, err)
	assert.Empty(t, store.results)
}

func TestMapSource_RecomputeReplacesEveryStoredDestination(t *testing.T) {
	ctx := context.Background()
	index := &fakeMatchIndex{}
	store := newFakeDestinationStore()
	source := NewMapSource(sourceTestBase(), index, store, fakeSyncStates{}, sourceTestSources(), Overrides{})
	area := Area{Name: "funchal", Country: "pt", Lat: 32.65, Lon: -16.91}
	before, err := source.Destination(ctx, area)
	require.NoError(t, err)
	require.Zero(t, before.WriterCounts["Q1"])

	index.rows = sourceTestIndex().rows
	require.NoError(t, source.Recompute(ctx))
	after, err := source.Destination(ctx, area)

	require.NoError(t, err)
	assert.Equal(t, 1, after.WriterCounts["Q1"])
	assert.Len(t, store.results, 1)
}

func TestMapSource_RecomputeDestinationReplacesOnlyThatDestination(t *testing.T) {
	ctx := context.Background()
	index := &fakeMatchIndex{}
	store := newFakeDestinationStore()
	source := NewMapSource(sourceTestBase(), index, store, fakeSyncStates{}, sourceTestSources(), Overrides{})
	funchal := Area{Name: "funchal", Country: "pt", Lat: 32.65, Lon: -16.91}
	other := Area{Name: "funchal", Country: "pt", Lat: 32.66, Lon: -16.92}
	_, err := source.Destination(ctx, funchal)
	require.NoError(t, err)
	_, err = source.Destination(ctx, other)
	require.NoError(t, err)

	index.rows = sourceTestIndex().rows
	require.NoError(t, source.RecomputeDestination(ctx, funchal))

	recomputed, err := source.Destination(ctx, funchal)
	require.NoError(t, err)
	assert.Equal(t, 1, recomputed.WriterCounts["Q1"])
	untouched, err := source.Destination(ctx, other)
	require.NoError(t, err)
	assert.Zero(t, untouched.WriterCounts["Q1"], "the other stored destination waits for the monthly recompute")
}

func TestMapSource_ServesWithoutHTTPRequests(t *testing.T) {
	ctx := context.Background()
	transport := &countingTransport{}
	previous := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = previous })
	source := NewMapSource(sourceTestBase(), sourceTestIndex(), newFakeDestinationStore(), fakeSyncStates{}, sourceTestSources(), Overrides{})
	area := Area{Name: "Funchal"}

	_, err := source.Destination(ctx, area)
	require.NoError(t, err)
	_, _, err = source.Place(ctx, area, "Q1")
	require.NoError(t, err)

	assert.Zero(t, transport.calls)
}

func TestParseOverrides_ReadsMatchRulesAndAreas(t *testing.T) {
	got, err := ParseOverrides([]byte(`{
		"stoplist": ["praia"],
		"merges": [{"canonical_name": "A", "names": ["B"]}],
		"areas": [{"name": "Madeira", "lat": 32.75, "lon": -16.96, "radius_km": 70}],
		"summary_hide_qids": ["Q1"]
	}`))

	require.NoError(t, err)
	assert.Equal(t, []string{"praia"}, got.Stoplist)
	assert.Equal(t, []PlaceMerge{{CanonicalName: "A", Names: []string{"B"}}}, got.Merges)
	assert.Equal(t, []Area{{Name: "Madeira", Lat: 32.75, Lon: -16.96, RadiusKM: 70}}, got.Areas)

	_, err = ParseOverrides([]byte(`{`))
	assert.Error(t, err)
}

func sourceTestBase() *fakeBaseSource {
	monte := Place{QID: "Q1", Names: []string{"Monte"}, Lat: 32.67, Lon: -16.90}
	return &fakeBaseSource{result: BasePlaces{Pins: []Place{monte}, Gazetteer: []Place{monte}}}
}

func sourceTestIndex() *fakeMatchIndex {
	post := IndexedPost{
		Host: "writer.test", PostID: 1, URL: "https://writer.test/1", Title: "Funchal in a day",
		Extracted: ExtractedPost{IsItinerary: true, Days: []DaySection{{Number: 1, Start: 0, End: 100}}},
	}
	return &fakeMatchIndex{rows: []MatchRow{
		{Post: post, Phrase: NamePhrase{Folded: "funchal", OriginalCase: true, Start: 0, End: 7}},
		{Post: post, Phrase: NamePhrase{Folded: "monte", OriginalCase: true, Start: 20, End: 25}},
	}}
}

func sourceTestSources() []Source {
	return []Source{{Host: "writer.test", Name: "Writer", Status: StatusActive}}
}

type fakeBaseSource struct {
	result BasePlaces
	err    error
	calls  int
}

func (f *fakeBaseSource) Places(context.Context, Area) (BasePlaces, error) {
	f.calls++
	return f.result, f.err
}

type fakeMatchIndex struct {
	rows    []MatchRow
	phrases []string
	calls   int
}

func (f *fakeMatchIndex) WriterMatchRows(_ context.Context, phrases []string) ([]MatchRow, error) {
	f.calls++
	f.phrases = phrases
	return f.rows, nil
}

type fakeDestinationStore struct{ results map[Area]DestinationResult }

func newFakeDestinationStore() *fakeDestinationStore {
	return &fakeDestinationStore{results: make(map[Area]DestinationResult)}
}

func (f *fakeDestinationStore) WriterDestination(_ context.Context, dest Area) (DestinationResult, bool, error) {
	result, ok := f.results[dest]
	return result, ok, nil
}

func (f *fakeDestinationStore) SaveWriterDestination(_ context.Context, dest Area, result DestinationResult) error {
	f.results[dest] = result
	return nil
}

func (f *fakeDestinationStore) WriterDestinations(context.Context) ([]Area, error) {
	areas := make([]Area, 0, len(f.results))
	for area := range f.results {
		areas = append(areas, area)
	}
	return areas, nil
}

type fakeSyncStates map[string]SyncState

func (f fakeSyncStates) SyncState(_ context.Context, host string) (SyncState, error) {
	state, ok := f[host]
	if !ok {
		return SyncState{Host: host}, nil
	}
	return state, nil
}

type fakeSummaries map[string]Summary

func (f fakeSummaries) Summary(_ context.Context, qid string) (*Summary, error) {
	summary, ok := f[qid]
	if !ok {
		return nil, nil
	}
	return &summary, nil
}

type countingTransport struct{ calls int }

func (c *countingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	c.calls++
	return nil, errors.New("unexpected HTTP request")
}

type fakePartialBase struct {
	*fakeBaseSource
	progress BaseProgress
	err      error
	partials int
}

func (f *fakePartialBase) Partial(context.Context, Area) (BaseProgress, error) {
	f.partials++
	return f.progress, f.err
}

func TestMapSource_DestinationProgress_ServesPartialPinsWithoutMatchingWhileLoading(t *testing.T) {
	ctx := context.Background()
	monte := Place{QID: "Q1", Names: []string{"Monte"}, Lat: 32.67, Lon: -16.90}
	base := &fakePartialBase{fakeBaseSource: sourceTestBase(), progress: BaseProgress{BasePlaces: BasePlaces{Pins: []Place{monte}, Gazetteer: []Place{monte}}}}
	index, store := sourceTestIndex(), newFakeDestinationStore()
	source := NewMapSource(base, index, store, fakeSyncStates{}, sourceTestSources(), Overrides{})
	area := Area{Name: "Funchal", Lat: 32.65, Lon: -16.91}

	got, loading, err := source.DestinationProgress(ctx, area)

	require.NoError(t, err)
	assert.True(t, loading)
	assert.Equal(t, []Place{monte}, got.Places)
	assert.False(t, got.Complete)
	assert.Empty(t, got.WriterCounts)
	assert.Zero(t, index.calls, "no writer matching on a partial result")
	assert.Empty(t, store.results, "a partial result is not stored")
}

func TestMapSource_DestinationProgress_MatchesOnceWhenTheLoadIsDone(t *testing.T) {
	ctx := context.Background()
	monte := Place{QID: "Q1", Names: []string{"Monte"}, Lat: 32.67, Lon: -16.90}
	done := BaseProgress{BasePlaces: BasePlaces{Pins: []Place{monte}, Gazetteer: []Place{monte}}, Done: true}
	base := &fakePartialBase{fakeBaseSource: sourceTestBase(), progress: done}
	index, store := sourceTestIndex(), newFakeDestinationStore()
	source := NewMapSource(base, index, store, fakeSyncStates{}, sourceTestSources(), Overrides{})
	area := Area{Name: "Funchal", Lat: 32.65, Lon: -16.91}

	first, loading, err := source.DestinationProgress(ctx, area)
	require.NoError(t, err)
	second, _, err := source.DestinationProgress(ctx, area)
	require.NoError(t, err)

	assert.False(t, loading)
	assert.Equal(t, 1, first.WriterCounts["Q1"])
	assert.Equal(t, first, second)
	assert.Equal(t, 1, index.calls, "matching ran once; the second call read the stored result")
	assert.Equal(t, 1, base.partials)
}

func TestMapSource_DestinationProgress_FailureWithNothingLoadedIsAnError(t *testing.T) {
	base := &fakePartialBase{fakeBaseSource: sourceTestBase(), err: errors.New("wikidata down")}
	source := NewMapSource(base, sourceTestIndex(), newFakeDestinationStore(), fakeSyncStates{}, sourceTestSources(), Overrides{})

	got, _, err := source.DestinationProgress(context.Background(), Area{Name: "Funchal"})

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestMapSource_DestinationProgress_FailureWithPinsServesThemAsNotLoading(t *testing.T) {
	monte := Place{QID: "Q1", Names: []string{"Monte"}}
	base := &fakePartialBase{fakeBaseSource: sourceTestBase(), err: errors.New("wikidata down"), progress: BaseProgress{BasePlaces: BasePlaces{Pins: []Place{monte}}}}
	source := NewMapSource(base, sourceTestIndex(), newFakeDestinationStore(), fakeSyncStates{}, sourceTestSources(), Overrides{})

	got, loading, err := source.DestinationProgress(context.Background(), Area{Name: "Funchal"})

	require.NoError(t, err)
	assert.False(t, loading, "the load stopped; the next visit resumes it")
	assert.Equal(t, []Place{monte}, got.Places)
}

func TestMapSource_DestinationProgress_PlainBaseSourceBlocksLikeDestination(t *testing.T) {
	source := NewMapSource(sourceTestBase(), sourceTestIndex(), newFakeDestinationStore(), fakeSyncStates{}, sourceTestSources(), Overrides{})

	got, loading, err := source.DestinationProgress(context.Background(), Area{Name: "Funchal", Lat: 32.65, Lon: -16.91})

	require.NoError(t, err)
	assert.False(t, loading)
	assert.Equal(t, 1, got.WriterCounts["Q1"])
}

func TestMapSource_Place_FindsAPinInAPartialResult(t *testing.T) {
	monte := Place{QID: "Q1", Names: []string{"Monte"}}
	base := &fakePartialBase{fakeBaseSource: sourceTestBase(), progress: BaseProgress{BasePlaces: BasePlaces{Pins: []Place{monte}}}}
	source := NewMapSource(base, sourceTestIndex(), newFakeDestinationStore(), fakeSyncStates{}, sourceTestSources(), Overrides{})

	place, summary, err := source.Place(context.Background(), Area{Name: "Funchal"}, "Q1")

	require.NoError(t, err)
	assert.Equal(t, "Q1", place.QID)
	assert.Nil(t, summary)
}
