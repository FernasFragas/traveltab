package writermap_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"weatherservice/internal/adapters/api"
	"weatherservice/internal/adapters/sqlite"
	"weatherservice/internal/writermap"
	"weatherservice/writerdata"
)

const (
	blogHost  = "blog.example"
	blogAPI   = "https://blog.example/wp-json/wp/v2/posts"
	otherHost = "other.example"
	otherAPI  = "https://other.example/wp-json/wp/v2/posts"
	postText  = "SECRET-POST-TEXT"
)

var (
	crawlTime   = time.Date(2026, 10, 11, 14, 0, 0, 0, time.UTC) // outside the off-peak window
	monthlyTime = time.Date(2026, 11, 1, 3, 0, 0, 0, time.UTC)   // first off-peak window of November
)

func TestParseSources_ReadsEmbeddedAllowlist(t *testing.T) {
	data, err := writerdata.Files.ReadFile("sources.json")
	require.NoError(t, err)

	sources, err := writermap.ParseSources(data)

	require.NoError(t, err)
	require.NotEmpty(t, sources)
	assert.Equal(t, writermap.Source{
		Host: "almadeviajante.com", Name: "Alma de Viajante",
		APIBase:  "https://almadeviajante.com/wp-json/wp/v2/posts",
		Language: "pt", Status: writermap.StatusActive,
	}, sources[0])
}

func TestParseSources_RejectsInvalidJSON(t *testing.T) {
	_, err := writermap.ParseSources([]byte(`{`))

	assert.Error(t, err)
}

func TestSyncJob_InitialCrawlIndexesEveryPostAndCompletes(t *testing.T) {
	wp := newFakeWordPress(2)
	wp.addBlog(blogAPI, testPosts(5)...)
	index := newFakeIndex()
	job := newTestJob(index, wp, crawlTime, activeSource(blogHost, blogAPI))

	require.NoError(t, job.Run(context.Background()))

	assert.Len(t, index.postsFor(blogHost), 5)
	post := index.post(blogHost, 3)
	assert.Equal(t, "https://blog.example/3", post.URL)
	assert.Equal(t, "Post 3", post.Title)
	assert.Equal(t, "en", post.Language)
	assert.Contains(t, foldedPhrases(post), "pico ruivo")
	state := index.state(blogHost)
	assert.Equal(t, writermap.StatusActive, state.Status)
	assert.Equal(t, crawlTime, state.LastFull)
	assert.True(t, state.LastDelta.IsZero())
	assert.Equal(t, testModified(5).Format(time.RFC3339Nano), state.Cursor)
	assert.True(t, wp.blog(blogAPI).listedAfter[0].IsZero(), "a first crawl lists every post")
}

func TestSyncJob_InterruptedCrawlResumesFromCursor(t *testing.T) {
	wp := newFakeWordPress(2)
	wp.addBlog(blogAPI, testPosts(5)...)
	wp.blog(blogAPI).listErr = func(call int) error {
		if call == 2 {
			return errors.New("connection reset")
		}
		return nil
	}
	index := newFakeIndex()
	job := newTestJob(index, wp, crawlTime, activeSource(blogHost, blogAPI))

	require.Error(t, job.Run(context.Background()))

	interrupted := index.state(blogHost)
	assert.Equal(t, writermap.StatusPartial, interrupted.Status)
	assert.True(t, interrupted.LastFull.IsZero())
	assert.Equal(t, testModified(2).Format(time.RFC3339Nano), interrupted.Cursor, "cursor saved after page 1")
	assert.Len(t, index.postsFor(blogHost), 2)

	wp.blog(blogAPI).listErr = nil
	upsertsBefore := index.upsertCount()
	require.NoError(t, job.Run(context.Background()))

	listed := wp.blog(blogAPI).listedAfter
	assert.Equal(t, testModified(2).Add(-time.Second), listed[2], "resume lists from the saved cursor")
	assert.Len(t, index.postsFor(blogHost), 5)
	assert.LessOrEqual(t, index.upsertCount()-upsertsBefore, 4, "resume does not restart the crawl")
	assert.Equal(t, writermap.StatusActive, index.state(blogHost).Status)
	assert.Equal(t, crawlTime, index.state(blogHost).LastFull)
}

func TestSyncJob_CrawlWithEqualTimestampsAcrossPagesIndexesEveryPost(t *testing.T) {
	wp := newFakeWordPress(2)
	posts := testPosts(5)
	for i := range posts {
		posts[i].Modified = testModified(1)
	}
	wp.addBlog(blogAPI, posts...)
	index := newFakeIndex()
	job := newTestJob(index, wp, crawlTime, activeSource(blogHost, blogAPI))

	require.NoError(t, job.Run(context.Background()))

	assert.Len(t, index.postsFor(blogHost), 5)
}

func TestSyncJob_MonthlyRunReindexesEditedPosts(t *testing.T) {
	wp := newFakeWordPress(2)
	wp.addBlog(blogAPI, testPosts(5)...)
	index := newFakeIndex()
	clock := &testClock{now: crawlTime}
	job := newTestJobWithClock(index, wp, clock, activeSource(blogHost, blogAPI))
	require.NoError(t, job.Run(context.Background()))

	edited := testPost(3)
	edited.Title = "Post 3 edited"
	edited.Content = "<p>" + postText + " Now about Cabo Girão.</p>"
	edited.Modified = time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	wp.put(blogAPI, edited)
	added := testPost(6)
	added.Modified = time.Date(2026, 10, 21, 9, 0, 0, 0, time.UTC)
	wp.put(blogAPI, added)
	callsBefore := len(wp.blog(blogAPI).listedAfter)
	clock.set(monthlyTime)
	require.NoError(t, job.Run(context.Background()))

	post := index.post(blogHost, 3)
	assert.Equal(t, "Post 3 edited", post.Title)
	assert.Contains(t, foldedPhrases(post), "cabo girao")
	assert.NotContains(t, foldedPhrases(post), "pico ruivo")
	assert.Len(t, index.postsFor(blogHost), 6)
	listed := wp.blog(blogAPI).listedAfter
	assert.Equal(t, testModified(5).Add(-time.Second), listed[callsBefore], "monthly run lists changes since the cursor")
	state := index.state(blogHost)
	assert.Equal(t, monthlyTime, state.LastDelta)
	assert.Equal(t, crawlTime, state.LastFull)
	assert.Equal(t, added.Modified.Format(time.RFC3339Nano), state.Cursor)
}

func TestSyncJob_MonthlyRunDropsDeletedPosts(t *testing.T) {
	wp := newFakeWordPress(2)
	wp.addBlog(blogAPI, testPosts(5)...)
	index := newFakeIndex()
	clock := &testClock{now: crawlTime}
	job := newTestJobWithClock(index, wp, clock, activeSource(blogHost, blogAPI))
	require.NoError(t, job.Run(context.Background()))

	wp.remove(blogAPI, 2)
	clock.set(monthlyTime)
	require.NoError(t, job.Run(context.Background()))

	assert.Equal(t, []int64{1, 3, 4, 5}, index.postIDs(blogHost))
	assert.Equal(t, writermap.StatusActive, index.state(blogHost).Status)
}

func TestSyncJob_SweepConfirmsMissingIDsBeforeDeleting(t *testing.T) {
	wp := newFakeWordPress(2)
	wp.addBlog(blogAPI, testPosts(6)...)
	index := newFakeIndex()
	clock := &testClock{now: crawlTime}
	job := newTestJobWithClock(index, wp, clock, activeSource(blogHost, blogAPI))
	require.NoError(t, job.Run(context.Background()))

	// Post 1 was really deleted. Post 6 is deleted after the first sweep page is read, so post 4
	// shifts onto page 1 and the first sweep never sees it, although it still exists.
	wp.remove(blogAPI, 1)
	blog := wp.blog(blogAPI)
	blog.afterIDsPage = func(b *fakeBlog, page int) {
		if page == 1 && b.idsCalls == 1 {
			delete(b.posts, 6)
		}
	}
	clock.set(monthlyTime)
	require.NoError(t, job.Run(context.Background()))

	assert.Equal(t, []int64{2, 3, 4, 5, 6}, index.postIDs(blogHost), "4 kept; 6 goes next month")
}

func TestSyncJob_FailedSweepDeletesNothing(t *testing.T) {
	wp := newFakeWordPress(2)
	wp.addBlog(blogAPI, testPosts(5)...)
	index := newFakeIndex()
	clock := &testClock{now: crawlTime}
	job := newTestJobWithClock(index, wp, clock, activeSource(blogHost, blogAPI))
	require.NoError(t, job.Run(context.Background()))

	wp.remove(blogAPI, 1)
	wp.blog(blogAPI).idsErr = func(page int) error {
		if page == 2 {
			return errors.New("status 500")
		}
		return nil
	}
	clock.set(monthlyTime)
	require.Error(t, job.Run(context.Background()))

	assert.Equal(t, []int64{1, 2, 3, 4, 5}, index.postIDs(blogHost))
	assert.Equal(t, writermap.StatusPartial, index.state(blogHost).Status)
	assert.True(t, index.state(blogHost).LastDelta.IsZero())
}

func TestSyncJob_EmptySweepDeletesNothing(t *testing.T) {
	wp := newFakeWordPress(2)
	wp.addBlog(blogAPI, testPosts(3)...)
	index := newFakeIndex()
	clock := &testClock{now: crawlTime}
	job := newTestJobWithClock(index, wp, clock, activeSource(blogHost, blogAPI))
	require.NoError(t, job.Run(context.Background()))

	for id := int64(1); id <= 3; id++ {
		wp.remove(blogAPI, id)
	}
	clock.set(monthlyTime)
	require.Error(t, job.Run(context.Background()))

	assert.Equal(t, []int64{1, 2, 3}, index.postIDs(blogHost))
}

func TestSyncJob_PartialMonthlyRunResumesInNextWindow(t *testing.T) {
	wp := newFakeWordPress(2)
	wp.addBlog(blogAPI, testPosts(5)...)
	index := newFakeIndex()
	clock := &testClock{now: crawlTime}
	job := newTestJobWithClock(index, wp, clock, activeSource(blogHost, blogAPI))
	require.NoError(t, job.Run(context.Background()))
	wp.remove(blogAPI, 1)
	wp.blog(blogAPI).idsErr = func(int) error { return errors.New("status 500") }
	clock.set(monthlyTime)
	require.Error(t, job.Run(context.Background()))

	wp.blog(blogAPI).idsErr = nil
	clock.set(monthlyTime.Add(24 * time.Hour))
	require.NoError(t, job.Run(context.Background()))

	assert.Equal(t, []int64{2, 3, 4, 5}, index.postIDs(blogHost))
	assert.Equal(t, monthlyTime.Add(24*time.Hour), index.state(blogHost).LastDelta)
}

func TestSyncJob_MonthlyRunWaitsForNewMonthAndOffPeak(t *testing.T) {
	wp := newFakeWordPress(2)
	wp.addBlog(blogAPI, testPosts(3)...)
	index := newFakeIndex()
	clock := &testClock{now: crawlTime}
	job := newTestJobWithClock(index, wp, clock, activeSource(blogHost, blogAPI))
	require.NoError(t, job.Run(context.Background()))
	calls := wp.blog(blogAPI).listCalls

	clock.set(time.Date(2026, 10, 12, 3, 0, 0, 0, time.UTC)) // off-peak, same month
	require.NoError(t, job.Run(context.Background()))
	clock.set(time.Date(2026, 11, 1, 14, 0, 0, 0, time.UTC)) // new month, peak hours
	require.NoError(t, job.Run(context.Background()))
	assert.Equal(t, calls, wp.blog(blogAPI).listCalls, "nothing is due")

	clock.set(monthlyTime)
	require.NoError(t, job.Run(context.Background()))
	assert.Greater(t, wp.blog(blogAPI).listCalls, calls)
}

func TestSyncJob_BlockedOrDisallowedHostIsMarkedAndSkipped(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"403": func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) },
		"robots disallow": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/robots.txt" {
				_, _ = fmt.Fprint(w, "User-agent: *\nDisallow: /\n")
				return
			}
			t.Errorf("disallowed path fetched: %s", r.URL.Path)
		},
	} {
		t.Run(name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				handler(w, r)
			}))
			t.Cleanup(server.Close)
			blocked := activeSource("blocked.example", server.URL+"/wp-json/wp/v2/posts")
			wp := newFakeWordPress(2)
			wp.addBlog(otherAPI, testPosts(2)...)
			lister := routedLister{
				blocked.APIBase: clientLister{api.NewWordPressClient(server.Client(), api.NewHostLimiter())},
				otherAPI:        wp,
			}
			index := newFakeIndex()
			job := writermap.NewSyncJob([]writermap.Source{blocked, activeSource(otherHost, otherAPI)}, index, lister)
			job.Now = func() time.Time { return crawlTime }
			job.Logf = func(string, ...any) {}
			job.BlockedErrors = []error{api.ErrBlocked, api.ErrDisallowed}

			require.NoError(t, job.Run(context.Background()))

			assert.Equal(t, writermap.StatusBlocked, index.state("blocked.example").Status)
			assert.Len(t, index.postsFor(otherHost), 2, "other sources still sync")

			before := requests.Load()
			require.NoError(t, job.Run(context.Background()))
			assert.Equal(t, before, requests.Load(), "blocked host is not requested again")
			assert.Equal(t, writermap.StatusBlocked, index.state("blocked.example").Status)
		})
	}
}

func TestSyncJob_RemovedSourceIsPurgedAndAllowlistBlockedIsSkipped(t *testing.T) {
	wp := newFakeWordPress(2)
	wp.addBlog(blogAPI, testPosts(2)...)
	wp.addBlog(otherAPI, testPosts(2)...)
	index := newFakeIndex()
	sources := []writermap.Source{activeSource(blogHost, blogAPI), activeSource(otherHost, otherAPI)}
	job := newTestJob(index, wp, crawlTime, sources...)
	require.NoError(t, job.Run(context.Background()))
	calls, removedCalls := wp.blog(otherAPI).listCalls, wp.blog(blogAPI).listCalls

	sources[0].Status = writermap.StatusRemoved
	sources[1].Status = writermap.StatusBlocked
	job = newTestJob(index, wp, monthlyTime, sources...)
	require.NoError(t, job.Run(context.Background()))

	assert.Empty(t, index.postsFor(blogHost))
	assert.Equal(t, writermap.SyncState{Host: blogHost}, index.state(blogHost), "sync state purged too")
	assert.Len(t, index.postsFor(otherHost), 2, "allowlist-blocked source keeps its rows")
	assert.Equal(t, calls, wp.blog(otherAPI).listCalls, "allowlist-blocked source is not requested")
	assert.Equal(t, removedCalls, wp.blog(blogAPI).listCalls, "removed source is not requested")
}

func TestSyncJob_ConcurrentRunIsRejected(t *testing.T) {
	wp := newFakeWordPress(2)
	wp.addBlog(blogAPI, testPosts(2)...)
	entered, release := make(chan struct{}), make(chan struct{})
	wp.blog(blogAPI).onList = func() {
		close(entered)
		<-release
	}
	index := newFakeIndex()
	job := newTestJob(index, wp, crawlTime, activeSource(blogHost, blogAPI))
	done := make(chan error)
	go func() { done <- job.Run(context.Background()) }()
	<-entered

	err := job.Run(context.Background())

	assert.ErrorIs(t, err, writermap.ErrSyncRunning)
	close(release)
	require.NoError(t, <-done)
	assert.Len(t, index.postsFor(blogHost), 2)
}

func TestSyncJob_RefreshesCachedDestinationsThenRunsSummaryStep(t *testing.T) {
	wp := newFakeWordPress(2)
	wp.addBlog(blogAPI, testPosts(3)...)
	index := newFakeIndex()
	job := newTestJob(index, wp, crawlTime, activeSource(blogHost, blogAPI))
	var steps []string
	funchal, lisbon := writermap.Area{Name: "Funchal"}, writermap.Area{Name: "Lisbon"}
	job.Destinations = fakeDestinations{funchal, lisbon}
	job.Refresher = refresherFunc(func(_ context.Context, dest writermap.Area) error {
		steps = append(steps, "refresh "+dest.Name)
		if dest.Name == "Funchal" {
			return errors.New("wikidata down")
		}
		return nil
	})
	job.SummaryStep = func(context.Context) error {
		steps = append(steps, fmt.Sprintf("summaries after %d posts", len(index.postsFor(blogHost))))
		return nil
	}

	err := job.Run(context.Background())

	require.ErrorContains(t, err, "wikidata down")
	assert.Equal(t, []string{"refresh Funchal", "refresh Lisbon", "summaries after 3 posts"}, steps)
}

func TestSyncJob_RefreshListErrorStillRunsSummaryStep(t *testing.T) {
	wp := newFakeWordPress(2)
	wp.addBlog(blogAPI, testPosts(1)...)
	job := newTestJob(newFakeIndex(), wp, crawlTime, activeSource(blogHost, blogAPI))
	job.Destinations = failingDestinations{}
	job.Refresher = refresherFunc(func(context.Context, writermap.Area) error { return nil })
	summarized := false
	job.SummaryStep = func(context.Context) error { summarized = true; return nil }

	require.ErrorContains(t, job.Run(context.Background()), "list cached destinations")
	assert.True(t, summarized)
}

func TestSyncJob_NothingDueSkipsRefreshAndSummaryStep(t *testing.T) {
	wp := newFakeWordPress(2)
	wp.addBlog(blogAPI, testPosts(1)...)
	index := newFakeIndex()
	job := newTestJob(index, wp, crawlTime, activeSource(blogHost, blogAPI))
	require.NoError(t, job.Run(context.Background()))
	called := false
	job.SummaryStep = func(context.Context) error { called = true; return nil }
	job.Destinations = fakeDestinations{{Name: "Funchal"}}
	job.Refresher = refresherFunc(func(context.Context, writermap.Area) error { called = true; return nil })

	require.NoError(t, job.Run(context.Background()))

	assert.False(t, called)
}

func TestSyncJob_LogsOneLinePerHostWithoutPostText(t *testing.T) {
	wp := newFakeWordPress(2)
	wp.addBlog(blogAPI, testPosts(3)...)
	index := newFakeIndex()
	sources := []writermap.Source{
		activeSource(blogHost, blogAPI),
		{Host: otherHost, APIBase: otherAPI, Status: writermap.StatusRemoved},
	}
	job := newTestJob(index, wp, crawlTime, sources...)
	var lines []string
	job.Logf = func(format string, args ...any) { lines = append(lines, fmt.Sprintf(format, args...)) }

	require.NoError(t, job.Run(context.Background()))

	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], blogHost)
	assert.Contains(t, lines[0], "3 posts")
	assert.Contains(t, lines[1], otherHost)
	for _, line := range lines {
		assert.NotContains(t, line, postText)
		assert.NotContains(t, line, "Post 1")
	}
}

func TestSyncJob_ScheduleRunsAtStartupThenEachOffPeakWindow(t *testing.T) {
	wp := newFakeWordPress(2)
	wp.addBlog(blogAPI, testPosts(1)...)
	wp.blog(blogAPI).listErr = func(call int) error {
		if call == 1 {
			return errors.New("connection reset")
		}
		return nil
	}
	index := newFakeIndex()
	clock := &testClock{now: crawlTime}
	job := newTestJobWithClock(index, wp, clock, activeSource(blogHost, blogAPI))
	ctx, cancel := context.WithCancel(context.Background())
	var sleeps []time.Duration
	job.Sleep = func(_ context.Context, d time.Duration) error {
		sleeps = append(sleeps, d)
		clock.set(clock.Now().Add(d))
		if len(sleeps) == 2 {
			cancel()
			return context.Canceled
		}
		return nil
	}

	job.Schedule(ctx)

	assert.Equal(t, []time.Duration{12 * time.Hour, 24 * time.Hour}, sleeps)
	assert.Equal(t, 2, wp.blog(blogAPI).listCalls, "startup run failed, the 02:00 run resumed")
	assert.Len(t, index.postsFor(blogHost), 1)
}

func TestSyncJob_DefaultSleepStopsWhenContextEnds(t *testing.T) {
	job := writermap.NewSyncJob(nil, newFakeIndex(), newFakeWordPress(1))
	job.Now = func() time.Time { return crawlTime }
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	job.Schedule(ctx) // returns instead of sleeping until 02:00
}

func TestSyncJob_TwoUnchangedRunsKeepRowCountsAndFileSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "writers.db")
	store, err := sqlite.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	wp := newFakeWordPress(2)
	posts := testPosts(5)
	posts[1].Content = "<h2>Day 1</h2><p>Pico Ruivo.</p><h2>Day 2</h2><p>Cabo Girão.</p>"
	wp.addBlog(blogAPI, posts...)
	clock := &testClock{now: crawlTime}
	job := writermap.NewSyncJob([]writermap.Source{activeSource(blogHost, blogAPI)}, storeIndex{store}, wp)
	job.Now = clock.Now
	job.Logf = func(string, ...any) {}
	require.NoError(t, job.Run(context.Background()))

	clock.set(monthlyTime)
	require.NoError(t, job.Run(context.Background()))
	counts, size := tableCounts(t, path), fileSize(t, path)
	clock.set(time.Date(2026, 12, 1, 3, 0, 0, 0, time.UTC))
	require.NoError(t, job.Run(context.Background()))

	assert.Equal(t, counts, tableCounts(t, path))
	assert.Equal(t, size, fileSize(t, path))
	assert.Equal(t, 5, counts["writer_posts"])
	assert.Positive(t, counts["writer_names"])
	assert.Positive(t, counts["writer_days"])
	assert.Equal(t, 1, counts["writer_sync"])
}

// --- test helpers ---

func newTestJob(index writermap.SyncIndex, posts writermap.PostLister, now time.Time, sources ...writermap.Source) *writermap.SyncJob {
	return newTestJobWithClock(index, posts, &testClock{now: now}, sources...)
}

func newTestJobWithClock(index writermap.SyncIndex, posts writermap.PostLister, clock *testClock, sources ...writermap.Source) *writermap.SyncJob {
	job := writermap.NewSyncJob(sources, index, posts)
	job.Now = clock.Now
	job.Logf = func(string, ...any) {}
	return job
}

func activeSource(host, apiBase string) writermap.Source {
	return writermap.Source{Host: host, Name: host, APIBase: apiBase, Language: "en", Status: writermap.StatusActive}
}

func testPosts(n int) []writermap.SourcePost {
	posts := make([]writermap.SourcePost, n)
	for i := range posts {
		posts[i] = testPost(int64(i + 1))
	}
	return posts
}

func testPost(id int64) writermap.SourcePost {
	return writermap.SourcePost{
		ID:       id,
		Link:     fmt.Sprintf("https://blog.example/%d", id),
		Title:    fmt.Sprintf("Post %d", id),
		Modified: testModified(int(id)),
		Content:  "<p>" + postText + " We walked up Pico Ruivo.</p>",
	}
}

func testModified(id int) time.Time {
	return time.Date(2026, 1, 1, id, 0, 0, 0, time.UTC)
}

func foldedPhrases(post writermap.IndexedPost) []string {
	var folded []string
	for _, phrase := range post.Extracted.Phrases {
		folded = append(folded, phrase.Folded)
	}
	return folded
}

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) set(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = now
}

// fakeWordPress serves posts like the WordPress REST API: listings ordered by modified time
// (oldest first) with a strict modified_after, and id sweeps newest id first.
type fakeWordPress struct {
	mu      sync.Mutex
	perPage int
	blogs   map[string]*fakeBlog
}

type fakeBlog struct {
	posts        map[int64]writermap.SourcePost
	listCalls    int
	idsCalls     int
	listedAfter  []time.Time
	listErr      func(call int) error
	idsErr       func(page int) error
	afterIDsPage func(b *fakeBlog, page int)
	onList       func()
}

func newFakeWordPress(perPage int) *fakeWordPress {
	return &fakeWordPress{perPage: perPage, blogs: map[string]*fakeBlog{}}
}

func (f *fakeWordPress) addBlog(apiBase string, posts ...writermap.SourcePost) {
	blog := &fakeBlog{posts: map[int64]writermap.SourcePost{}}
	for _, post := range posts {
		blog.posts[post.ID] = post
	}
	f.blogs[apiBase] = blog
}

func (f *fakeWordPress) blog(apiBase string) *fakeBlog {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.blogs[apiBase]
}

func (f *fakeWordPress) put(apiBase string, post writermap.SourcePost) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.blogs[apiBase].posts[post.ID] = post
}

func (f *fakeWordPress) remove(apiBase string, id int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.blogs[apiBase].posts, id)
}

func (f *fakeWordPress) ListPosts(_ context.Context, apiBase string, modifiedAfter time.Time, page int) ([]writermap.SourcePost, bool, error) {
	f.mu.Lock()
	blog := f.blogs[apiBase]
	blog.listCalls++
	blog.listedAfter = append(blog.listedAfter, modifiedAfter)
	onList := blog.onList
	f.mu.Unlock()
	if onList != nil {
		onList()
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if blog.listErr != nil {
		if err := blog.listErr(blog.listCalls); err != nil {
			return nil, false, err
		}
	}
	after := modifiedAfter.Truncate(time.Second)
	var matching []writermap.SourcePost
	for _, post := range blog.posts {
		if modifiedAfter.IsZero() || post.Modified.After(after) {
			matching = append(matching, post)
		}
	}
	sort.Slice(matching, func(i, j int) bool {
		if !matching[i].Modified.Equal(matching[j].Modified) {
			return matching[i].Modified.Before(matching[j].Modified)
		}
		return matching[i].ID < matching[j].ID
	})
	items, more := paginate(matching, page, f.perPage)
	return items, more, nil
}

func (f *fakeWordPress) PostIDs(_ context.Context, apiBase string, page int) ([]int64, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	blog := f.blogs[apiBase]
	blog.idsCalls++
	if blog.idsErr != nil {
		if err := blog.idsErr(page); err != nil {
			return nil, false, err
		}
	}
	var ids []int64
	for id := range blog.posts {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] > ids[j] })
	items, more := paginate(ids, page, f.perPage)
	if blog.afterIDsPage != nil {
		blog.afterIDsPage(blog, page)
	}
	return items, more, nil
}

func paginate[T any](all []T, page, perPage int) ([]T, bool) {
	start := (page - 1) * perPage
	if start >= len(all) {
		return nil, false
	}
	end := min(start+perPage, len(all))
	return all[start:end], end < len(all)
}

// routedLister sends each API base to its own lister.
type routedLister map[string]writermap.PostLister

func (r routedLister) ListPosts(ctx context.Context, apiBase string, after time.Time, page int) ([]writermap.SourcePost, bool, error) {
	return r[apiBase].ListPosts(ctx, apiBase, after, page)
}

func (r routedLister) PostIDs(ctx context.Context, apiBase string, page int) ([]int64, bool, error) {
	return r[apiBase].PostIDs(ctx, apiBase, page)
}

// clientLister adapts the real WordPress client, as cmd/web does.
type clientLister struct{ client *api.WordPressClient }

func (c clientLister) ListPosts(ctx context.Context, apiBase string, after time.Time, page int) ([]writermap.SourcePost, bool, error) {
	posts, more, err := c.client.ListPosts(ctx, apiBase, after, page)
	out := make([]writermap.SourcePost, len(posts))
	for i, post := range posts {
		out[i] = writermap.SourcePost{ID: post.ID, Link: post.Link, Title: post.Title, Modified: post.Modified, Content: post.Content}
	}
	return out, more, err
}

func (c clientLister) PostIDs(ctx context.Context, apiBase string, page int) ([]int64, bool, error) {
	return c.client.PostIDs(ctx, apiBase, page)
}

type fakeIndex struct {
	mu      sync.Mutex
	posts   map[string]map[int64]writermap.IndexedPost
	states  map[string]writermap.SyncState
	upserts int
}

func newFakeIndex() *fakeIndex {
	return &fakeIndex{posts: map[string]map[int64]writermap.IndexedPost{}, states: map[string]writermap.SyncState{}}
}

func (f *fakeIndex) UpsertPost(_ context.Context, post writermap.IndexedPost) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.posts[post.Host] == nil {
		f.posts[post.Host] = map[int64]writermap.IndexedPost{}
	}
	f.posts[post.Host][post.PostID] = post
	f.upserts++
	return nil
}

func (f *fakeIndex) DeletePost(_ context.Context, host string, postID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.posts[host], postID)
	return nil
}

func (f *fakeIndex) PurgeHost(_ context.Context, host string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.posts, host)
	delete(f.states, host)
	return nil
}

func (f *fakeIndex) PostIDs(_ context.Context, host string) ([]int64, error) {
	return f.postIDs(host), nil
}

func (f *fakeIndex) SyncState(_ context.Context, host string) (writermap.SyncState, error) {
	return f.state(host), nil
}

func (f *fakeIndex) SaveSyncState(_ context.Context, state writermap.SyncState) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.states[state.Host] = state
	return nil
}

func (f *fakeIndex) postsFor(host string) map[int64]writermap.IndexedPost {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.posts[host]
}

func (f *fakeIndex) post(host string, id int64) writermap.IndexedPost {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.posts[host][id]
}

func (f *fakeIndex) postIDs(host string) []int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	var ids []int64
	for id := range f.posts[host] {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func (f *fakeIndex) state(host string) writermap.SyncState {
	f.mu.Lock()
	defer f.mu.Unlock()
	if state, ok := f.states[host]; ok {
		return state
	}
	return writermap.SyncState{Host: host}
}

func (f *fakeIndex) upsertCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.upserts
}

type fakeDestinations []writermap.Area

func (f fakeDestinations) CachedDestinations(context.Context) ([]writermap.Area, error) {
	return f, nil
}

type failingDestinations struct{}

func (failingDestinations) CachedDestinations(context.Context) ([]writermap.Area, error) {
	return nil, errors.New("database locked")
}

type refresherFunc func(context.Context, writermap.Area) error

func (f refresherFunc) Refresh(ctx context.Context, dest writermap.Area) error { return f(ctx, dest) }

// storeIndex adapts the SQLite store, as cmd/web does.
type storeIndex struct{ store *sqlite.Store }

func (s storeIndex) UpsertPost(ctx context.Context, post writermap.IndexedPost) error {
	return s.store.UpsertWriterPost(ctx, sqlite.WriterPost{
		Host: post.Host, PostID: post.PostID, URL: post.URL, Title: post.Title,
		Language: post.Language, Modified: post.Modified, Extracted: post.Extracted,
	})
}

func (s storeIndex) DeletePost(ctx context.Context, host string, postID int64) error {
	return s.store.DeleteWriterPost(ctx, host, postID)
}

func (s storeIndex) PurgeHost(ctx context.Context, host string) error {
	return s.store.PurgeWriterHost(ctx, host)
}

func (s storeIndex) PostIDs(ctx context.Context, host string) ([]int64, error) {
	return s.store.WriterPostIDs(ctx, host)
}

func (s storeIndex) SyncState(ctx context.Context, host string) (writermap.SyncState, error) {
	state, err := s.store.GetWriterSyncState(ctx, host)
	return writermap.SyncState(state), err
}

func (s storeIndex) SaveSyncState(ctx context.Context, state writermap.SyncState) error {
	return s.store.SaveWriterSyncState(ctx, sqlite.WriterSyncState(state))
}

func tableCounts(t *testing.T, path string) map[string]int {
	t.Helper()
	conn, err := sql.Open("sqlite3", path)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()
	counts := map[string]int{}
	for _, table := range []string{"writer_posts", "writer_names", "writer_days", "writer_sync"} {
		var n int
		require.NoError(t, conn.QueryRow("SELECT COUNT(*) FROM "+table).Scan(&n))
		counts[table] = n
	}
	return counts
}

func fileSize(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	return info.Size()
}
