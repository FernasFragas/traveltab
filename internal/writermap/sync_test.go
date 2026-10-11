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
	"strings"
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

func TestSyncJob_IndexesOnlyTheSelectedPosts(t *testing.T) {
	wp := newFakeWordPress()
	wp.addBlog(blogAPI, testPosts(130)...)
	wp.put(blogAPI, bodyOnlyPost(500)) // the body names Madeira, the title and slug do not
	index := newFakeIndex()
	job := newTestJob(index, wp, crawlTime, activeSource(blogHost, blogAPI))

	require.NoError(t, job.Run(context.Background()))

	assert.Len(t, index.postsFor(blogHost), writermap.MaxIndexedPosts)
	assert.Equal(t, writermap.MaxIndexedPosts, wp.blog(blogAPI).fetched, "only chosen posts are fetched")
	assert.NotContains(t, index.postIDs(blogHost), int64(500))
	post := index.post(blogHost, 130)
	assert.Equal(t, "https://blog.example/130", post.URL)
	assert.Equal(t, "Post 130 in Madeira", post.Title)
	assert.Equal(t, "en", post.Language)
	assert.Contains(t, foldedPhrases(post), "pico ruivo")
	state := index.state(blogHost)
	assert.Equal(t, writermap.StatusActive, state.Status)
	assert.Equal(t, crawlTime, state.LastFull)
}

func TestSyncJob_SearchesEachTermOncePerBlogAndFetchesEachPostOnce(t *testing.T) {
	wp := newFakeWordPress()
	wp.addBlog(blogAPI, testPosts(3)...)
	wp.put(blogAPI, namedPost(4, "Madeira and Lisboa", "two-places"))
	index := newFakeIndex()
	job := newTestJob(index, wp, crawlTime, activeSource(blogHost, blogAPI))
	job.Areas = []writermap.Area{{Name: "Madeira"}, {Name: "madeira"}, {Name: "Lisboa"}}
	job.Destinations = fakeDestinations{{Name: "Funchal"}, {Name: "LISBOA"}}

	require.NoError(t, job.Run(context.Background()))

	assert.ElementsMatch(t, []string{"Madeira", "Lisboa", "Funchal"}, wp.blog(blogAPI).searched)
	assert.Equal(t, 4, wp.blog(blogAPI).fetched, "post 4 matches two terms and is fetched once")
	assert.Equal(t, []int64{1, 2, 3, 4}, index.postIDs(blogHost))
}

func TestSyncJob_PlaceNamesFromTheMatcherCountAsDestinationNames(t *testing.T) {
	wp := newFakeWordPress()
	wp.addBlog(blogAPI, testPosts(1)...)
	hike := namedPost(2, "Hiking to Pico Ruivo", "hiking")
	hike.Content = "<p>" + postText + " Our week in Madeira.</p>" // found by the Madeira search
	wp.put(blogAPI, hike)
	index := newFakeIndex()
	job := newTestJob(index, wp, crawlTime, activeSource(blogHost, blogAPI))
	job.PlaceNames = func(context.Context) ([]string, error) { return []string{"Pico Ruivo"}, nil }

	require.NoError(t, job.Run(context.Background()))

	assert.Equal(t, []int64{1, 2}, index.postIDs(blogHost), "post 2 is found by Madeira and its title names Pico Ruivo")
}

func TestSyncJob_PlaceNamesErrorStopsTheRunBeforeAnyRequest(t *testing.T) {
	wp := newFakeWordPress()
	wp.addBlog(blogAPI, testPosts(2)...)
	job := newTestJob(newFakeIndex(), wp, crawlTime, activeSource(blogHost, blogAPI))
	job.PlaceNames = func(context.Context) ([]string, error) { return nil, errors.New("database locked") }

	require.ErrorContains(t, job.Run(context.Background()), "database locked")

	assert.Empty(t, wp.blog(blogAPI).searched)
}

func TestSyncJob_ListingCachedDestinationsFailsTheRunBeforeAnyRequest(t *testing.T) {
	wp := newFakeWordPress()
	wp.addBlog(blogAPI, testPosts(2)...)
	job := newTestJob(newFakeIndex(), wp, crawlTime, activeSource(blogHost, blogAPI))
	job.Destinations = failingDestinations{}

	require.ErrorContains(t, job.Run(context.Background()), "list cached destinations")

	assert.Empty(t, wp.blog(blogAPI).searched)
}

func TestSyncJob_PurgesPostsThatDropOut(t *testing.T) {
	wp := newFakeWordPress()
	wp.addBlog(blogAPI, testPosts(130)...)
	wp.blog(blogAPI).comments = map[int64]int{}
	for id := int64(1); id <= 100; id++ {
		wp.blog(blogAPI).comments[id] = 10 // posts 1-100 are the most commented
	}
	index := newFakeIndex()
	clock := &testClock{now: crawlTime}
	job := newTestJobWithClock(index, wp, clock, activeSource(blogHost, blogAPI))
	require.NoError(t, job.Run(context.Background()))
	require.Equal(t, int64(1), index.postIDs(blogHost)[0])

	wp.blog(blogAPI).comments = map[int64]int{}
	for id := int64(31); id <= 130; id++ {
		wp.blog(blogAPI).comments[id] = 10 // month 2: posts 31-130 lead
	}
	clock.set(monthlyTime)
	require.NoError(t, job.Run(context.Background()))

	ids := index.postIDs(blogHost)
	assert.Len(t, ids, writermap.MaxIndexedPosts)
	assert.Equal(t, int64(31), ids[0], "posts 1-30 were purged")
	assert.Equal(t, int64(130), ids[len(ids)-1], "posts 101-130 were indexed")
	assert.Equal(t, monthlyTime, index.state(blogHost).LastFull)
}

func TestSyncJob_SharesTheCapAcrossBlogsByTheirDestinationPosts(t *testing.T) {
	wp := newFakeWordPress()
	wp.addBlog(blogAPI, testPosts(150)...)
	wp.addBlog(otherAPI, testPosts(50)...)
	index := newFakeIndex()
	job := newTestJob(index, wp, crawlTime, activeSource(blogHost, blogAPI), activeSource(otherHost, otherAPI))

	require.NoError(t, job.Run(context.Background()))

	assert.Len(t, index.postsFor(blogHost), 75)
	assert.Len(t, index.postsFor(otherHost), 25)
}

func TestSyncJob_BlockedBlogsRowsStayAndCountInsideTheCap(t *testing.T) {
	const blockedHost = "blocked.example"
	for name, block := range map[string]func(*fakeIndex, *writermap.Source){
		"blocked in the allowlist": func(_ *fakeIndex, s *writermap.Source) { s.Status = writermap.StatusBlocked },
		"blocked by an earlier run": func(i *fakeIndex, s *writermap.Source) {
			require.NoError(t, i.SaveSyncState(context.Background(), writermap.SyncState{Host: s.Host, Status: writermap.StatusBlocked}))
		},
	} {
		t.Run(name, func(t *testing.T) {
			wp := newFakeWordPress()
			wp.addBlog(blogAPI, testPosts(150)...)
			wp.addBlog(otherAPI, testPosts(150)...)
			index := newFakeIndex()
			seedRows(t, index, blockedHost, 30)
			blocked := activeSource(blockedHost, "https://blocked.example/wp-json/wp/v2/posts")
			block(index, &blocked)
			job := newTestJob(index, wp, crawlTime, blocked, activeSource(blogHost, blogAPI), activeSource(otherHost, otherAPI))

			require.NoError(t, job.Run(context.Background()))

			assert.Len(t, index.postsFor(blockedHost), 30, "the blocked blog keeps its rows")
			assert.Len(t, index.postsFor(blogHost), 35)
			assert.Len(t, index.postsFor(otherHost), 35)
		})
	}
}

func TestSyncJob_BlogBlockedDuringTheRunKeepsItsRowsInsideTheCap(t *testing.T) {
	const blockedHost = "blocked.example"
	const blockedAPI = "https://blocked.example/wp-json/wp/v2/posts"
	wp := newFakeWordPress()
	wp.addBlog(blockedAPI, testPosts(5)...)
	wp.blog(blockedAPI).searchErr = api.ErrBlocked
	wp.addBlog(blogAPI, testPosts(150)...)
	wp.addBlog(otherAPI, testPosts(150)...)
	index := newFakeIndex()
	seedRows(t, index, blockedHost, 30)
	job := newTestJob(index, wp, crawlTime, activeSource(blockedHost, blockedAPI), activeSource(blogHost, blogAPI), activeSource(otherHost, otherAPI))
	job.BlockedErrors = []error{api.ErrBlocked}

	require.NoError(t, job.Run(context.Background()))

	assert.Equal(t, writermap.StatusBlocked, index.state(blockedHost).Status)
	assert.Len(t, index.postsFor(blockedHost), 30)
	assert.Len(t, index.postsFor(blogHost), 35)
	assert.Len(t, index.postsFor(otherHost), 35)
}

func TestSyncJob_SuddenEmptySearchKeepsRowsWarnsAndRetries(t *testing.T) {
	wp := newFakeWordPress()
	wp.addBlog(blogAPI, testPosts(150)...)
	wp.addBlog(otherAPI) // the search returns nothing this run
	index := newFakeIndex()
	seedRows(t, index, otherHost, 30)
	require.NoError(t, index.SaveSyncState(context.Background(), writermap.SyncState{Host: otherHost, Status: writermap.StatusActive, LastFull: crawlTime}))
	clock := &testClock{now: monthlyTime}
	job := newTestJobWithClock(index, wp, clock, activeSource(blogHost, blogAPI), activeSource(otherHost, otherAPI))
	var lines []string
	job.Logf = func(format string, args ...any) { lines = append(lines, fmt.Sprintf(format, args...)) }

	require.NoError(t, job.Run(context.Background()))

	assert.Len(t, index.postsFor(otherHost), 30, "the rows are kept")
	assert.Len(t, index.postsFor(blogHost), 70, "and count inside the cap")
	assert.Equal(t, crawlTime, index.state(otherHost).LastFull, "not finished, so the next run retries")
	assert.Equal(t, monthlyTime, index.state(blogHost).LastFull)
	assert.Contains(t, strings.Join(lines, "\n"), "writers sync "+otherHost+": warning")

	for id := int64(1); id <= 150; id++ {
		wp.put(otherAPI, testPost(id))
	}
	require.NoError(t, job.Run(context.Background()))

	assert.Equal(t, monthlyTime, index.state(otherHost).LastFull)
	assert.Len(t, index.postsFor(otherHost), 50)
	assert.Len(t, index.postsFor(blogHost), 50)
}

func TestSyncJob_BlogThatNeverHadPostsAndFindsNoneIsJustSkipped(t *testing.T) {
	wp := newFakeWordPress()
	wp.addBlog(blogAPI, testPosts(150)...)
	wp.addBlog(otherAPI)
	index := newFakeIndex()
	job := newTestJob(index, wp, crawlTime, activeSource(blogHost, blogAPI), activeSource(otherHost, otherAPI))
	var lines []string
	job.Logf = func(format string, args ...any) { lines = append(lines, fmt.Sprintf(format, args...)) }

	require.NoError(t, job.Run(context.Background()))

	assert.Len(t, index.postsFor(blogHost), writermap.MaxIndexedPosts)
	assert.Empty(t, index.postsFor(otherHost))
	assert.Equal(t, crawlTime, index.state(otherHost).LastFull, "nothing to retry")
	assert.NotContains(t, strings.Join(lines, "\n"), "warning")
}

func TestSyncJob_UsesCommentCountsToRankAndNoCommentsNewest(t *testing.T) {
	wp := newFakeWordPress()
	wp.addBlog(blogAPI, testPosts(120)...)
	wp.addBlog(otherAPI, testPosts(120)...)
	wp.blog(blogAPI).comments = map[int64]int{1: 50}
	index := newFakeIndex()
	job := newTestJob(index, wp, crawlTime, activeSource(blogHost, blogAPI), activeSource(otherHost, otherAPI))

	require.NoError(t, job.Run(context.Background()))

	assert.Contains(t, index.postIDs(blogHost), int64(1), "the only commented post is kept although it is the oldest")
	assert.NotContains(t, index.postIDs(otherHost), int64(1))
	assert.Contains(t, index.postIDs(otherHost), int64(120), "without comments the newest are taken")
}

func TestSyncJob_RunWaitsForNewMonthAndOffPeak(t *testing.T) {
	wp := newFakeWordPress()
	wp.addBlog(blogAPI, testPosts(3)...)
	index := newFakeIndex()
	clock := &testClock{now: crawlTime}
	job := newTestJobWithClock(index, wp, clock, activeSource(blogHost, blogAPI))
	require.NoError(t, job.Run(context.Background()))
	calls := len(wp.blog(blogAPI).searched)

	clock.set(time.Date(2026, 10, 12, 3, 0, 0, 0, time.UTC)) // off-peak, same month
	require.NoError(t, job.Run(context.Background()))
	clock.set(time.Date(2026, 11, 1, 14, 0, 0, 0, time.UTC)) // new month, peak hours
	require.NoError(t, job.Run(context.Background()))
	assert.Equal(t, calls, len(wp.blog(blogAPI).searched), "nothing is due")

	clock.set(monthlyTime)
	require.NoError(t, job.Run(context.Background()))
	assert.Greater(t, len(wp.blog(blogAPI).searched), calls)
}

func TestSyncJob_DueRunSearchesEveryActiveBlogEvenWhenOnlyOneIsDue(t *testing.T) {
	wp := newFakeWordPress()
	wp.addBlog(blogAPI, testPosts(3)...)
	wp.addBlog(otherAPI, testPosts(3)...)
	index := newFakeIndex()
	clock := &testClock{now: crawlTime}
	job := newTestJobWithClock(index, wp, clock, activeSource(blogHost, blogAPI))
	require.NoError(t, job.Run(context.Background()))

	job = newTestJobWithClock(index, wp, clock, activeSource(blogHost, blogAPI), activeSource(otherHost, otherAPI))
	require.NoError(t, job.Run(context.Background())) // the new blog has never synced

	assert.Len(t, wp.blog(blogAPI).searched, 2, "the cap is shared, so every blog is re-picked together")
	assert.Len(t, index.postsFor(otherHost), 3)
}

func TestSyncJob_FailedRunWritesNothingAndRetriesNextWindow(t *testing.T) {
	wp := newFakeWordPress()
	wp.addBlog(blogAPI, testPosts(3)...)
	wp.addBlog(otherAPI, testPosts(3)...)
	wp.blog(otherAPI).searchErr = errors.New("status 500")
	index := newFakeIndex()
	clock := &testClock{now: crawlTime}
	job := newTestJobWithClock(index, wp, clock, activeSource(blogHost, blogAPI), activeSource(otherHost, otherAPI))

	require.ErrorContains(t, job.Run(context.Background()), "status 500")

	assert.Empty(t, index.postsFor(blogHost), "a half-seen cap is never applied")
	assert.True(t, index.state(blogHost).LastFull.IsZero())

	wp.blog(otherAPI).searchErr = nil
	require.NoError(t, job.Run(context.Background()))
	assert.Len(t, index.postsFor(blogHost), 3)
	assert.Len(t, index.postsFor(otherHost), 3)
}

func TestSyncJob_CommentErrorFailsTheRun(t *testing.T) {
	wp := newFakeWordPress()
	wp.addBlog(blogAPI, testPosts(3)...)
	wp.blog(blogAPI).commentsErr = errors.New("status 500")
	index := newFakeIndex()
	job := newTestJob(index, wp, crawlTime, activeSource(blogHost, blogAPI))

	require.ErrorContains(t, job.Run(context.Background()), "status 500")

	assert.Empty(t, index.postsFor(blogHost))
}

func TestSyncJob_FetchErrorKeepsThatBlogsRowsAndRetries(t *testing.T) {
	wp := newFakeWordPress()
	wp.addBlog(blogAPI, testPosts(3)...)
	wp.addBlog(otherAPI, testPosts(3)...)
	index := newFakeIndex()
	clock := &testClock{now: crawlTime}
	job := newTestJobWithClock(index, wp, clock, activeSource(blogHost, blogAPI), activeSource(otherHost, otherAPI))
	require.NoError(t, job.Run(context.Background()))

	wp.blog(blogAPI).fetchErr = errors.New("status 502")
	wp.remove(otherAPI, 3)
	clock.set(monthlyTime)
	require.ErrorContains(t, job.Run(context.Background()), "status 502")

	assert.Len(t, index.postsFor(blogHost), 3, "the failed blog keeps its rows")
	assert.Equal(t, crawlTime, index.state(blogHost).LastFull, "and stays due")
	assert.Equal(t, []int64{1, 2}, index.postIDs(otherHost), "the other blog is still updated")
	assert.Equal(t, monthlyTime, index.state(otherHost).LastFull)
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
			wp := newFakeWordPress()
			wp.addBlog(otherAPI, testPosts(2)...)
			source := routedSource{
				blocked.APIBase: clientSource{api.NewWordPressClient(server.Client(), api.NewHostLimiter())},
				otherAPI:        wp,
			}
			index := newFakeIndex()
			job := writermap.NewSyncJob([]writermap.Source{blocked, activeSource(otherHost, otherAPI)}, index, source)
			job.Areas = []writermap.Area{{Name: "Madeira"}}
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

func TestSyncJob_SkipsBlockedAndRemovedSources(t *testing.T) {
	wp := newFakeWordPress()
	wp.addBlog(blogAPI, testPosts(2)...)
	wp.addBlog(otherAPI, testPosts(2)...)
	index := newFakeIndex()
	sources := []writermap.Source{activeSource(blogHost, blogAPI), activeSource(otherHost, otherAPI)}
	job := newTestJob(index, wp, crawlTime, sources...)
	require.NoError(t, job.Run(context.Background()))
	searches, removedSearches := len(wp.blog(otherAPI).searched), len(wp.blog(blogAPI).searched)

	sources[0].Status = writermap.StatusRemoved
	sources[1].Status = writermap.StatusBlocked
	job = newTestJob(index, wp, monthlyTime, sources...)
	require.NoError(t, job.Run(context.Background()))

	assert.Empty(t, index.postsFor(blogHost))
	assert.Equal(t, writermap.SyncState{Host: blogHost}, index.state(blogHost), "sync state purged too")
	assert.Len(t, index.postsFor(otherHost), 2, "allowlist-blocked source keeps its rows")
	assert.Equal(t, searches, len(wp.blog(otherAPI).searched), "allowlist-blocked source is not requested")
	assert.Equal(t, removedSearches, len(wp.blog(blogAPI).searched), "removed source is not requested")
}

func TestSyncJob_ConcurrentRunIsRejected(t *testing.T) {
	wp := newFakeWordPress()
	wp.addBlog(blogAPI, testPosts(2)...)
	entered, release := make(chan struct{}), make(chan struct{})
	wp.blog(blogAPI).onSearch = func() {
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
	wp := newFakeWordPress()
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

func TestSyncJob_SummaryStepIsSkippedWhenNothingWasIndexed(t *testing.T) {
	wp := newFakeWordPress()
	wp.addBlog(blogAPI, bodyOnlyPost(1))
	job := newTestJob(newFakeIndex(), wp, crawlTime, activeSource(blogHost, blogAPI))
	summarized := false
	job.SummaryStep = func(context.Context) error { summarized = true; return nil }

	require.NoError(t, job.Run(context.Background()))

	assert.False(t, summarized, "no post was added or removed")
}

func TestSyncJob_NothingDueSkipsRefreshAndSummaryStep(t *testing.T) {
	wp := newFakeWordPress()
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
	wp := newFakeWordPress()
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
	assert.Contains(t, lines[0], otherHost, "removed sources are handled first")
	assert.Contains(t, lines[1], blogHost)
	assert.Contains(t, lines[1], "3 posts")
	for _, line := range lines {
		assert.NotContains(t, line, postText)
		assert.NotContains(t, line, "Post 1")
	}
}

func TestSyncJob_ScheduleRunsAtStartupThenEachOffPeakWindow(t *testing.T) {
	wp := newFakeWordPress()
	wp.addBlog(blogAPI, testPosts(1)...)
	wp.blog(blogAPI).searchErr = errors.New("connection reset")
	index := newFakeIndex()
	clock := &testClock{now: crawlTime}
	job := newTestJobWithClock(index, wp, clock, activeSource(blogHost, blogAPI))
	ctx, cancel := context.WithCancel(context.Background())
	var sleeps []time.Duration
	job.Sleep = func(_ context.Context, d time.Duration) error {
		sleeps = append(sleeps, d)
		clock.set(clock.Now().Add(d))
		wp.blog(blogAPI).searchErr = nil // the 02:00 run succeeds
		if len(sleeps) == 2 {
			cancel()
			return context.Canceled
		}
		return nil
	}

	job.Schedule(ctx)

	assert.Equal(t, []time.Duration{12 * time.Hour, 24 * time.Hour}, sleeps)
	assert.Len(t, index.postsFor(blogHost), 1, "startup run failed, the 02:00 run retried")
}

func TestSyncJob_DefaultSleepStopsWhenContextEnds(t *testing.T) {
	job := writermap.NewSyncJob(nil, newFakeIndex(), newFakeWordPress())
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
	wp := newFakeWordPress()
	posts := testPosts(5)
	posts[1].Content = "<h2>Day 1</h2><p>Pico Ruivo.</p><h2>Day 2</h2><p>Cabo Girão.</p>"
	wp.addBlog(blogAPI, posts...)
	clock := &testClock{now: crawlTime}
	job := writermap.NewSyncJob([]writermap.Source{activeSource(blogHost, blogAPI)}, storeIndex{store}, wp)
	job.Areas = []writermap.Area{{Name: "Madeira"}}
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

func newTestJob(index writermap.SyncIndex, posts writermap.PostSource, now time.Time, sources ...writermap.Source) *writermap.SyncJob {
	return newTestJobWithClock(index, posts, &testClock{now: now}, sources...)
}

func newTestJobWithClock(index writermap.SyncIndex, posts writermap.PostSource, clock *testClock, sources ...writermap.Source) *writermap.SyncJob {
	job := writermap.NewSyncJob(sources, index, posts)
	job.Areas = []writermap.Area{{Name: "Madeira"}}
	job.Now = clock.Now
	job.Logf = func(string, ...any) {}
	return job
}

func activeSource(host, apiBase string) writermap.Source {
	return writermap.Source{Host: host, Name: host, APIBase: apiBase, Language: "en", Status: writermap.StatusActive}
}

// seedRows stores n already-indexed posts for host, as an earlier run would have.
func seedRows(t *testing.T, index *fakeIndex, host string, n int) {
	t.Helper()
	for i := 1; i <= n; i++ {
		require.NoError(t, index.UpsertPost(context.Background(), writermap.IndexedPost{Host: host, PostID: int64(1000 + i)}))
	}
}

func testPosts(n int) []writermap.SourcePost {
	posts := make([]writermap.SourcePost, n)
	for i := range posts {
		posts[i] = testPost(int64(i + 1))
	}
	return posts
}

func testPost(id int64) writermap.SourcePost {
	post := namedPost(id, fmt.Sprintf("Post %d in Madeira", id), fmt.Sprintf("post-%d", id))
	post.Link = fmt.Sprintf("https://blog.example/%d", id)
	return post
}

// namedPost has a place-naming title; the slug is kept in Link's tail by the fake.
func namedPost(id int64, title, slug string) writermap.SourcePost {
	return writermap.SourcePost{
		ID:       id,
		Link:     "https://blog.example/" + slug,
		Title:    title,
		Modified: testModified(int(id)),
		Content:  "<p>" + postText + " We walked up Pico Ruivo.</p>",
	}
}

// bodyOnlyPost matches a search on the body, but neither its title nor its slug names a place.
func bodyOnlyPost(id int64) writermap.SourcePost {
	post := namedPost(id, "Packing list", "packing-list")
	post.Content = "<p>" + postText + " Our week in Madeira.</p>"
	return post
}

func testModified(id int) time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(id) * time.Hour)
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

// fakeWordPress serves posts like the WordPress REST API: search matches the title, slug or body,
// and every call is counted so tests can see what was requested.
type fakeWordPress struct {
	mu    sync.Mutex
	blogs map[string]*fakeBlog
}

type fakeBlog struct {
	posts       map[int64]writermap.SourcePost
	comments    map[int64]int
	searched    []string
	fetched     int
	searchErr   error
	commentsErr error
	fetchErr    error
	onSearch    func()
}

func newFakeWordPress() *fakeWordPress {
	return &fakeWordPress{blogs: map[string]*fakeBlog{}}
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

func (f *fakeWordPress) SearchPosts(_ context.Context, apiBase, term string, page int) ([]writermap.SearchHit, bool, error) {
	f.mu.Lock()
	blog := f.blogs[apiBase]
	blog.searched = append(blog.searched, term)
	onSearch := blog.onSearch
	f.mu.Unlock()
	if onSearch != nil {
		onSearch()
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if blog.searchErr != nil {
		return nil, false, blog.searchErr
	}
	var hits []writermap.SearchHit
	for _, post := range blog.posts {
		text := strings.ToLower(post.Title + " " + post.Link + " " + post.Content)
		if page == 1 && strings.Contains(text, strings.ToLower(term)) {
			slug := post.Link[strings.LastIndex(post.Link, "/")+1:]
			hits = append(hits, writermap.SearchHit{ID: post.ID, Link: post.Link, Slug: slug, Title: post.Title, Published: post.Modified})
		}
	}
	return hits, false, nil
}

func (f *fakeWordPress) RecentCommentCounts(_ context.Context, apiBase string, _ int) (map[int64]int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	blog := f.blogs[apiBase]
	return blog.comments, blog.commentsErr
}

func (f *fakeWordPress) PostsByID(_ context.Context, apiBase string, ids []int64) ([]writermap.SourcePost, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	blog := f.blogs[apiBase]
	if blog.fetchErr != nil {
		return nil, blog.fetchErr
	}
	var posts []writermap.SourcePost
	for _, id := range ids {
		if post, ok := blog.posts[id]; ok {
			posts = append(posts, post)
			blog.fetched++
		}
	}
	return posts, nil
}

// routedSource sends each API base to its own source.
type routedSource map[string]writermap.PostSource

func (r routedSource) SearchPosts(ctx context.Context, apiBase, term string, page int) ([]writermap.SearchHit, bool, error) {
	return r[apiBase].SearchPosts(ctx, apiBase, term, page)
}

func (r routedSource) RecentCommentCounts(ctx context.Context, apiBase string, maxPages int) (map[int64]int, error) {
	return r[apiBase].RecentCommentCounts(ctx, apiBase, maxPages)
}

func (r routedSource) PostsByID(ctx context.Context, apiBase string, ids []int64) ([]writermap.SourcePost, error) {
	return r[apiBase].PostsByID(ctx, apiBase, ids)
}

// clientSource adapts the real WordPress client, as cmd/web does.
type clientSource struct{ client *api.WordPressClient }

func (c clientSource) SearchPosts(ctx context.Context, apiBase, term string, page int) ([]writermap.SearchHit, bool, error) {
	hits, more, err := c.client.SearchPosts(ctx, apiBase, term, page)
	out := make([]writermap.SearchHit, len(hits))
	for i, hit := range hits {
		out[i] = writermap.SearchHit{ID: hit.ID, Link: hit.Link, Slug: hit.Slug, Title: hit.Title, Published: hit.Published}
	}
	return out, more, err
}

func (c clientSource) RecentCommentCounts(ctx context.Context, apiBase string, maxPages int) (map[int64]int, error) {
	return c.client.RecentCommentCounts(ctx, apiBase, maxPages)
}

func (c clientSource) PostsByID(ctx context.Context, apiBase string, ids []int64) ([]writermap.SourcePost, error) {
	posts, err := c.client.PostsByID(ctx, apiBase, ids)
	out := make([]writermap.SourcePost, len(posts))
	for i, post := range posts {
		out[i] = writermap.SourcePost{ID: post.ID, Link: post.Link, Title: post.Title, Modified: post.Modified, Content: post.Content}
	}
	return out, err
}

type fakeIndex struct {
	mu     sync.Mutex
	posts  map[string]map[int64]writermap.IndexedPost
	states map[string]writermap.SyncState
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
