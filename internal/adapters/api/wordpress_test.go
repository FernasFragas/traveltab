package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Synthetic WordPress responses: no real post text.
const (
	wpPostsJSON = `[
  {"id":11,"link":"https://blog.example/funchal-3-days/","title":{"rendered":"Funchal &#8211; 3 days"},"modified_gmt":"2026-09-30T08:15:00","content":{"rendered":"<p>Synthetic body one.</p>"}},
  {"id":12,"link":"https://blog.example/porto-moniz/","title":{"rendered":"Porto Moniz"},"modified_gmt":"2026-10-01T10:00:00","content":{"rendered":"<p>Synthetic body two.</p>"}}
]`
	wpPastLastPageJSON = `{"code":"rest_post_invalid_page_number","message":"The page number requested is larger than the number of pages available.","data":{"status":400}}`
)

func TestWordPress_FirstPage400IsError(t *testing.T) {
	server := wordpressServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	_, _, err := testWordPressClient(server).SearchPosts(context.Background(), wordpressAPIBase(server), "madeira", 1)

	assert.ErrorContains(t, err, "status 400")
}

func TestWordPress_PostContent_FetchesByID(t *testing.T) {
	var got *http.Request
	server := wordpressServer(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		_, _ = w.Write([]byte(`{"content":{"rendered":"<p>Synthetic body.</p>"}}`))
	})

	content, err := testWordPressClient(server).PostContent(context.Background(), wordpressAPIBase(server), 42)

	require.NoError(t, err)
	assert.Equal(t, "<p>Synthetic body.</p>", content)
	assert.Equal(t, "/wp-json/wp/v2/posts/42", got.URL.Path)
	assert.Equal(t, "content", got.URL.Query().Get("_fields"))
}

func TestWordPress_403ReturnsErrBlocked(t *testing.T) {
	server := wordpressServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	client := testWordPressClient(server)
	base := wordpressAPIBase(server)

	_, _, searchErr := client.SearchPosts(context.Background(), base, "madeira", 1)
	_, contentErr := client.PostContent(context.Background(), base, 42)

	assert.ErrorIs(t, searchErr, ErrBlocked)
	assert.ErrorIs(t, contentErr, ErrBlocked)
}

func TestWordPress_OtherStatusIsErrorWithoutRetry(t *testing.T) {
	var calls atomic.Int32
	server := wordpressServer(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := testWordPressClient(server).PostContent(context.Background(), wordpressAPIBase(server), 42)

	assert.ErrorContains(t, err, "status 500")
	assert.NotErrorIs(t, err, ErrBlocked)
	assert.Equal(t, int32(1), calls.Load())
}

func TestWordPress_TimeoutRetriesOnceWithBackoff(t *testing.T) {
	var calls atomic.Int32
	server := wordpressServer(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			<-r.Context().Done() // hang until the client gives up
			return
		}
		_, _ = w.Write([]byte(`{"content":{"rendered":"<p>Synthetic body.</p>"}}`))
	})
	client := testWordPressClient(server)
	client.client.Timeout = 50 * time.Millisecond
	var backoffs []time.Duration
	client.sleep = func(_ context.Context, d time.Duration) error {
		backoffs = append(backoffs, d)
		return nil
	}

	content, err := client.PostContent(context.Background(), wordpressAPIBase(server), 42)

	require.NoError(t, err)
	assert.Equal(t, "<p>Synthetic body.</p>", content)
	assert.Equal(t, int32(2), calls.Load())
	assert.Equal(t, []time.Duration{wordpressRetryDelay}, backoffs)
}

func TestWordPress_TimeoutTwiceReturnsError(t *testing.T) {
	var calls atomic.Int32
	server := wordpressServer(t, func(_ http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		<-r.Context().Done()
	})
	client := testWordPressClient(server)
	client.client.Timeout = 50 * time.Millisecond

	_, err := client.PostContent(context.Background(), wordpressAPIBase(server), 42)

	require.Error(t, err)
	assert.Equal(t, int32(2), calls.Load())
}

func TestWordPress_CancelledContextIsNotRetried(t *testing.T) {
	var calls atomic.Int32
	server := wordpressServer(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(`{"content":{"rendered":""}}`))
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := testWordPressClient(server).PostContent(ctx, wordpressAPIBase(server), 42)

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, int32(0), calls.Load())
}

func TestWordPress_RobotsDisallowStopsFetches(t *testing.T) {
	var apiCalls atomic.Int32
	server := wordpressServerWithRobots(t, http.StatusOK, "User-agent: *\nAllow: /\n\nUser-agent: TravelTab\nDisallow: /wp-json/\n", func(w http.ResponseWriter, _ *http.Request) {
		apiCalls.Add(1)
		_, _ = w.Write([]byte(`[]`))
	})

	_, _, err := testWordPressClient(server).SearchPosts(context.Background(), wordpressAPIBase(server), "madeira", 1)

	assert.ErrorIs(t, err, ErrDisallowed)
	assert.Equal(t, int32(0), apiCalls.Load())
}

func TestWordPress_RobotsIsCachedPerHost(t *testing.T) {
	var robotsCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			robotsCalls.Add(1)
			_, _ = w.Write([]byte("User-agent: *\nDisallow: /wp-admin/\n"))
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(server.Close)
	client := testWordPressClient(server)

	for page := 1; page <= 3; page++ {
		_, _, err := client.SearchPosts(context.Background(), wordpressAPIBase(server), "madeira", page)
		require.NoError(t, err)
	}

	assert.Equal(t, int32(1), robotsCalls.Load())
}

func TestWordPress_MissingRobotsAllowsEverything(t *testing.T) {
	server := wordpressServerWithRobots(t, http.StatusNotFound, "", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id":7}]`))
	})

	hits, _, err := testWordPressClient(server).SearchPosts(context.Background(), wordpressAPIBase(server), "madeira", 1)

	require.NoError(t, err)
	require.Len(t, hits, 1)
	assert.Equal(t, int64(7), hits[0].ID)
}

func TestWordPress_RobotsForbiddenIsBlocked(t *testing.T) {
	server := wordpressServerWithRobots(t, http.StatusForbidden, "", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	})

	_, _, err := testWordPressClient(server).SearchPosts(context.Background(), wordpressAPIBase(server), "madeira", 1)

	assert.ErrorIs(t, err, ErrBlocked)
}

func TestWordPress_RobotsServerErrorIsNotCached(t *testing.T) {
	var robotsCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			if robotsCalls.Add(1) == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(server.Close)
	client := testWordPressClient(server)

	_, _, firstErr := client.SearchPosts(context.Background(), wordpressAPIBase(server), "madeira", 1)
	_, _, secondErr := client.SearchPosts(context.Background(), wordpressAPIBase(server), "madeira", 1)

	assert.ErrorContains(t, firstErr, "status 503")
	assert.NoError(t, secondErr)
	assert.Equal(t, int32(2), robotsCalls.Load())
}

func TestWordPress_EveryRequestWaitsForTheHostLimiter(t *testing.T) {
	server := wordpressServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	})
	limiter, slept := frozenLimiter()
	client := NewWordPressClient(server.Client(), limiter)

	_, _, err := client.SearchPosts(context.Background(), wordpressAPIBase(server), "madeira", 1)
	require.NoError(t, err)
	_, _, err = client.SearchPosts(context.Background(), wordpressAPIBase(server), "madeira", 2)
	require.NoError(t, err)

	// robots.txt, page 1, page 2
	assert.Equal(t, []time.Duration{0, MinHostInterval, 2 * MinHostInterval}, *slept)
}

func TestRobotsRules_Allowed(t *testing.T) {
	robots := `# comment
User-agent: Googlebot
Disallow: /

User-agent: *
Disallow: /wp-admin/
Allow: /wp-admin/admin-ajax.php
Disallow: /*?s=
Disallow: /*.pdf$
`
	rules := parseRobots(robots)

	cases := map[string]bool{
		"/wp-json/wp/v2/posts?page=1":  true,
		"/wp-admin/options.php":        false,
		"/wp-admin/admin-ajax.php":     true,
		"/search/?s=funchal":           false,
		"/guide.pdf":                   false,
		"/guide.pdf?download=1":        true,
		"/wp-json/wp/v2/posts/42":      true,
		"/wp-json/wp/v2/posts?s=noise": false,
	}
	for path, want := range cases {
		assert.Equal(t, want, rules.allowed(path), path)
	}
}

func TestParseRobots_OwnGroupReplacesWildcardGroup(t *testing.T) {
	rules := parseRobots("User-agent: *\nDisallow: /\n\nUser-agent: Other\nUser-agent: traveltab\nDisallow: /private/\n")

	assert.True(t, rules.allowed("/wp-json/wp/v2/posts"))
	assert.False(t, rules.allowed("/private/x"))
}

func TestParseRobots_EmptyDisallowAllowsEverything(t *testing.T) {
	rules := parseRobots("User-agent: *\nDisallow:\n")

	assert.True(t, rules.allowed("/wp-json/wp/v2/posts"))
}

const (
	wpSearchJSON = `[
  {"id":21,"link":"https://blog.example/madeira-tips/","slug":"madeira-tips","title":{"rendered":"Madeira &amp; Porto Santo"},"date_gmt":"2026-09-30T08:15:00","date":"2026-09-30T09:15:00"},
  {"id":22,"link":"https://blog.example/old/","slug":"old","title":{"rendered":"Old one"},"date":"2020-01-02T03:04:05"}
]`
	wpCommentsPage1JSON = `[{"post":11},{"post":11},{"post":12}]`
	wpCommentsPage2JSON = `[{"post":12},{"post":13}]`
)

func TestWordPress_SearchPosts_ReturnsIDsTitlesAndSlugs(t *testing.T) {
	var got *http.Request
	server := wordpressServer(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.Header().Set("X-WP-TotalPages", "2")
		_, _ = w.Write([]byte(wpSearchJSON))
	})

	hits, more, err := testWordPressClient(server).SearchPosts(context.Background(), wordpressAPIBase(server), "Madeira", 1)

	require.NoError(t, err)
	assert.True(t, more)
	assert.Equal(t, "/wp-json/wp/v2/posts", got.URL.Path)
	assert.Equal(t, UserAgent, got.Header.Get("User-Agent"))
	query := got.URL.Query()
	assert.Equal(t, "Madeira", query.Get("search"))
	assert.Equal(t, "date", query.Get("orderby"))
	assert.Equal(t, "desc", query.Get("order"))
	assert.Equal(t, "100", query.Get("per_page"))
	assert.Equal(t, "1", query.Get("page"))
	assert.Equal(t, "id,link,slug,title,date_gmt,date", query.Get("_fields"))
	require.Len(t, hits, 2)
	assert.Equal(t, WordPressHit{
		ID:        21,
		Link:      "https://blog.example/madeira-tips/",
		Slug:      "madeira-tips",
		Title:     "Madeira & Porto Santo",
		Published: time.Date(2026, 9, 30, 8, 15, 0, 0, time.UTC),
	}, hits[0])
	assert.Equal(t, time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC), hits[1].Published)
}

func TestWordPress_SearchPosts_PastLastPageEndsCleanly(t *testing.T) {
	server := wordpressServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(wpPastLastPageJSON))
	})

	hits, more, err := testWordPressClient(server).SearchPosts(context.Background(), wordpressAPIBase(server), "x", 3)

	require.NoError(t, err)
	assert.False(t, more)
	assert.Empty(t, hits)
}

func TestWordPress_RecentCommentCounts_TalliesCommentsPerPost(t *testing.T) {
	var requests []*http.Request
	server := wordpressServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r)
		w.Header().Set("X-WP-TotalPages", "9")
		if r.URL.Query().Get("page") == "1" {
			_, _ = w.Write([]byte(wpCommentsPage1JSON))
			return
		}
		_, _ = w.Write([]byte(wpCommentsPage2JSON))
	})

	counts, err := testWordPressClient(server).RecentCommentCounts(context.Background(), wordpressAPIBase(server), 2)

	require.NoError(t, err)
	assert.Equal(t, map[int64]int{11: 2, 12: 2, 13: 1}, counts)
	require.Len(t, requests, 2, "stops at maxPages although more pages exist")
	assert.Equal(t, "/wp-json/wp/v2/comments", requests[0].URL.Path)
	assert.Equal(t, "post", requests[0].URL.Query().Get("_fields"))
	assert.Equal(t, "100", requests[0].URL.Query().Get("per_page"))
	assert.Equal(t, UserAgent, requests[0].Header.Get("User-Agent"))
}

func TestWordPress_RecentCommentCounts_StopsAtTheLastPage(t *testing.T) {
	var calls atomic.Int32
	server := wordpressServer(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("X-WP-TotalPages", "1")
		_, _ = w.Write([]byte(wpCommentsPage1JSON))
	})

	counts, err := testWordPressClient(server).RecentCommentCounts(context.Background(), wordpressAPIBase(server), 5)

	require.NoError(t, err)
	assert.Equal(t, map[int64]int{11: 2, 12: 1}, counts)
	assert.Equal(t, int32(1), calls.Load())
}

func TestWordPress_RecentCommentCounts_NotFoundMeansNoComments(t *testing.T) {
	server := wordpressServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	counts, err := testWordPressClient(server).RecentCommentCounts(context.Background(), wordpressAPIBase(server), 3)

	require.NoError(t, err)
	assert.NotNil(t, counts)
	assert.Empty(t, counts)
}

func TestWordPress_PostsByID_FetchesOnlyTheGivenIDs(t *testing.T) {
	var got *http.Request
	server := wordpressServer(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		_, _ = w.Write([]byte(wpPostsJSON))
	})

	posts, err := testWordPressClient(server).PostsByID(context.Background(), wordpressAPIBase(server), []int64{11, 12})

	require.NoError(t, err)
	assert.Equal(t, "/wp-json/wp/v2/posts", got.URL.Path)
	assert.Equal(t, "11,12", got.URL.Query().Get("include"))
	assert.Equal(t, "id,link,title,modified_gmt,content", got.URL.Query().Get("_fields"))
	require.Len(t, posts, 2)
	assert.Equal(t, "Funchal – 3 days", posts[0].Title)
	assert.Equal(t, "<p>Synthetic body two.</p>", posts[1].Content)
}

func TestWordPress_PostsByID_BatchesOfAHundred(t *testing.T) {
	var includes []string
	server := wordpressServer(t, func(w http.ResponseWriter, r *http.Request) {
		includes = append(includes, r.URL.Query().Get("include"))
		_, _ = w.Write([]byte(`[]`))
	})
	ids := make([]int64, 250)
	for i := range ids {
		ids[i] = int64(i + 1)
	}

	_, err := testWordPressClient(server).PostsByID(context.Background(), wordpressAPIBase(server), ids)

	require.NoError(t, err)
	require.Len(t, includes, 3)
	assert.Len(t, strings.Split(includes[0], ","), 100)
	assert.Len(t, strings.Split(includes[2], ","), 50)
}

func TestWordPress_PostsByID_NoIDsMakesNoRequest(t *testing.T) {
	var calls atomic.Int32
	server := wordpressServer(t, func(http.ResponseWriter, *http.Request) { calls.Add(1) })

	posts, err := testWordPressClient(server).PostsByID(context.Background(), wordpressAPIBase(server), nil)

	require.NoError(t, err)
	assert.Empty(t, posts)
	assert.Equal(t, int32(0), calls.Load())
}

func TestWordPress_NewReads_RobotsDisallowReturnsErrDisallowed(t *testing.T) {
	var apiCalls atomic.Int32
	server := wordpressServerWithRobots(t, http.StatusOK, "User-agent: *\nDisallow: /wp-json/\n", func(w http.ResponseWriter, _ *http.Request) {
		apiCalls.Add(1)
		_, _ = w.Write([]byte(`[]`))
	})
	for name, call := range newReadCalls(server) {
		t.Run(name, func(t *testing.T) {
			assert.ErrorIs(t, call(), ErrDisallowed)
		})
	}
	assert.Equal(t, int32(0), apiCalls.Load())
}

func TestWordPress_NewReads_403ReturnsErrBlocked(t *testing.T) {
	server := wordpressServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	for name, call := range newReadCalls(server) {
		t.Run(name, func(t *testing.T) {
			assert.ErrorIs(t, call(), ErrBlocked)
		})
	}
}

func TestWordPress_NewReads_OtherStatusIsError(t *testing.T) {
	server := wordpressServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	for name, call := range newReadCalls(server) {
		t.Run(name, func(t *testing.T) {
			err := call()
			require.Error(t, err)
			assert.Contains(t, err.Error(), fmt.Sprint(http.StatusInternalServerError))
		})
	}
}

// newReadCalls runs SearchPosts, RecentCommentCounts and PostsByID against server.
func newReadCalls(server *httptest.Server) map[string]func() error {
	client, base, ctx := testWordPressClient(server), wordpressAPIBase(server), context.Background()
	return map[string]func() error{
		"SearchPosts": func() error { _, _, err := client.SearchPosts(ctx, base, "madeira", 1); return err },
		"RecentCommentCounts": func() error {
			_, err := client.RecentCommentCounts(ctx, base, 1)
			return err
		},
		"PostsByID": func() error { _, err := client.PostsByID(ctx, base, []int64{1}); return err },
	}
}

func wordpressServer(t *testing.T, api http.HandlerFunc) *httptest.Server {
	return wordpressServerWithRobots(t, http.StatusNotFound, "", api)
}

func wordpressServerWithRobots(t *testing.T, robotsStatus int, robots string, api http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			w.WriteHeader(robotsStatus)
			_, _ = w.Write([]byte(robots))
			return
		}
		api(w, r)
	}))
	t.Cleanup(server.Close)
	return server
}

// testWordPressClient returns a client whose limiter and backoff never sleep.
func testWordPressClient(server *httptest.Server) *WordPressClient {
	limiter, _ := frozenLimiter()
	client := NewWordPressClient(server.Client(), limiter)
	client.sleep = func(context.Context, time.Duration) error { return nil }
	return client
}

func wordpressAPIBase(server *httptest.Server) string {
	return server.URL + "/wp-json/wp/v2/posts"
}
