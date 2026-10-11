package api

import (
	"context"
	"net/http"
	"net/http/httptest"
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

func TestWordPress_ListPosts_SendsQueryAndDecodesPage(t *testing.T) {
	var got *http.Request
	server := wordpressServer(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.Header().Set("X-WP-TotalPages", "3")
		_, _ = w.Write([]byte(wpPostsJSON))
	})
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	posts, more, err := testWordPressClient(server).ListPosts(context.Background(), wordpressAPIBase(server), since, 2)

	require.NoError(t, err)
	assert.True(t, more)
	assert.Equal(t, "/wp-json/wp/v2/posts", got.URL.Path)
	assert.Equal(t, UserAgent, got.Header.Get("User-Agent"))
	query := got.URL.Query()
	assert.Equal(t, "2", query.Get("page"))
	assert.Equal(t, "100", query.Get("per_page"))
	assert.Equal(t, "2026-09-01T00:00:00Z", query.Get("modified_after"))
	assert.Equal(t, "modified", query.Get("orderby"))
	assert.Equal(t, "asc", query.Get("order"))
	assert.Equal(t, "id,link,title,modified_gmt,content", query.Get("_fields"))
	require.Len(t, posts, 2)
	assert.Equal(t, WordPressPost{
		ID:       11,
		Link:     "https://blog.example/funchal-3-days/",
		Title:    "Funchal – 3 days",
		Modified: time.Date(2026, 9, 30, 8, 15, 0, 0, time.UTC),
		Content:  "<p>Synthetic body one.</p>",
	}, posts[0])
	assert.Equal(t, int64(12), posts[1].ID)
}

func TestWordPress_ListPosts_LastPageHasNoMore(t *testing.T) {
	server := wordpressServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-WP-TotalPages", "3")
		_, _ = w.Write([]byte(wpPostsJSON))
	})

	posts, more, err := testWordPressClient(server).ListPosts(context.Background(), wordpressAPIBase(server), time.Time{}, 3)

	require.NoError(t, err)
	assert.Len(t, posts, 2)
	assert.False(t, more)
}

func TestWordPress_ListPosts_ZeroTimeOmitsModifiedAfter(t *testing.T) {
	var query map[string][]string
	server := wordpressServer(t, func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		_, _ = w.Write([]byte(`[]`))
	})

	posts, more, err := testWordPressClient(server).ListPosts(context.Background(), wordpressAPIBase(server), time.Time{}, 1)

	require.NoError(t, err)
	assert.Empty(t, posts)
	assert.False(t, more)
	assert.NotContains(t, query, "modified_after")
}

func TestWordPress_ListPosts_400PastLastPageEndsCleanly(t *testing.T) {
	server := wordpressServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(wpPastLastPageJSON))
	})

	posts, more, err := testWordPressClient(server).ListPosts(context.Background(), wordpressAPIBase(server), time.Time{}, 4)

	require.NoError(t, err)
	assert.Empty(t, posts)
	assert.False(t, more)
}

func TestWordPress_ListPosts_400OnFirstPageIsError(t *testing.T) {
	server := wordpressServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	_, _, err := testWordPressClient(server).ListPosts(context.Background(), wordpressAPIBase(server), time.Time{}, 1)

	assert.ErrorContains(t, err, "status 400")
}

func TestWordPress_PostIDs_RequestsOnlyIDs(t *testing.T) {
	var query map[string][]string
	server := wordpressServer(t, func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		w.Header().Set("X-WP-TotalPages", "2")
		_, _ = w.Write([]byte(`[{"id":11},{"id":12}]`))
	})

	ids, more, err := testWordPressClient(server).PostIDs(context.Background(), wordpressAPIBase(server), 1)

	require.NoError(t, err)
	assert.Equal(t, []int64{11, 12}, ids)
	assert.True(t, more)
	assert.Equal(t, []string{"id"}, query["_fields"])
	assert.Equal(t, []string{"100"}, query["per_page"])
	assert.Equal(t, []string{"1"}, query["page"])
}

func TestWordPress_PostIDs_400PastLastPageEndsCleanly(t *testing.T) {
	server := wordpressServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	ids, more, err := testWordPressClient(server).PostIDs(context.Background(), wordpressAPIBase(server), 3)

	require.NoError(t, err)
	assert.Empty(t, ids)
	assert.False(t, more)
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

	_, _, listErr := client.ListPosts(context.Background(), base, time.Time{}, 1)
	_, _, idsErr := client.PostIDs(context.Background(), base, 1)
	_, contentErr := client.PostContent(context.Background(), base, 42)

	assert.ErrorIs(t, listErr, ErrBlocked)
	assert.ErrorIs(t, idsErr, ErrBlocked)
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

	_, _, err := testWordPressClient(server).ListPosts(context.Background(), wordpressAPIBase(server), time.Time{}, 1)

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
		_, _, err := client.PostIDs(context.Background(), wordpressAPIBase(server), page)
		require.NoError(t, err)
	}

	assert.Equal(t, int32(1), robotsCalls.Load())
}

func TestWordPress_MissingRobotsAllowsEverything(t *testing.T) {
	server := wordpressServerWithRobots(t, http.StatusNotFound, "", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id":7}]`))
	})

	ids, _, err := testWordPressClient(server).PostIDs(context.Background(), wordpressAPIBase(server), 1)

	require.NoError(t, err)
	assert.Equal(t, []int64{7}, ids)
}

func TestWordPress_RobotsForbiddenIsBlocked(t *testing.T) {
	server := wordpressServerWithRobots(t, http.StatusForbidden, "", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	})

	_, _, err := testWordPressClient(server).PostIDs(context.Background(), wordpressAPIBase(server), 1)

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

	_, _, firstErr := client.PostIDs(context.Background(), wordpressAPIBase(server), 1)
	_, _, secondErr := client.PostIDs(context.Background(), wordpressAPIBase(server), 1)

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

	_, _, err := client.PostIDs(context.Background(), wordpressAPIBase(server), 1)
	require.NoError(t, err)
	_, _, err = client.PostIDs(context.Background(), wordpressAPIBase(server), 2)
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
