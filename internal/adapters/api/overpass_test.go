package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"weatherservice/internal/planner"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	firstOverpassURL  = "https://first.example/api/interpreter"
	secondOverpassURL = "https://second.example/api/interpreter"
)

// secondServerBody names its hotel after the server, so a test can tell which
// server the result came from.
const secondServerBody = `{"elements":[{"type":"node","lat":37.1,"lon":-7.6,"tags":{"tourism":"hotel","name":"Second server hotel"}}]}`

// overpassReply is one canned answer for one request.
type overpassReply struct {
	status  int
	body    string
	headers map[string]string
	err     error
}

// overpassCall records what the client sent to a server.
type overpassCall struct {
	url    string
	data   string
	method string
	header http.Header
}

// overpassStub answers each server with the next reply queued for it. It also
// replaces the sleeper, so no test ever really waits.
func overpassStub(t *testing.T, servers []string, replies map[string][]overpassReply) (*OverpassAPI, *[]overpassCall, *[]time.Duration) {
	t.Helper()

	var calls []overpassCall
	var slept []time.Duration

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		form, err := url.ParseQuery(string(body))
		require.NoError(t, err)

		key := req.URL.String()
		calls = append(calls, overpassCall{
			url:    key,
			data:   form.Get("data"),
			method: req.Method,
			header: req.Header.Clone(),
		})

		queued := replies[key]
		require.NotEmpty(t, queued, "the client asked %s more often than the test planned for", key)
		reply := queued[0]
		replies[key] = queued[1:]

		if reply.err != nil {
			return nil, reply.err
		}

		header := http.Header{}
		for name, value := range reply.headers {
			header.Set(name, value)
		}

		return &http.Response{
			StatusCode: reply.status,
			Body:       io.NopCloser(strings.NewReader(reply.body)),
			Header:     header,
		}, nil
	})

	api := NewOverpassAPI(&http.Client{Transport: transport}, servers)
	api.sleep = func(d time.Duration) { slept = append(slept, d) }

	return api, &calls, &slept
}

func overpassFixture(t *testing.T) string {
	t.Helper()

	body, err := os.ReadFile("testdata/overpass/tavira.json")
	require.NoError(t, err)

	return string(body)
}

// taviraStays runs the happy path against the recorded Tavira answer.
func taviraStays(t *testing.T) []planner.Stay {
	t.Helper()

	api, _, _ := overpassStub(t, []string{firstOverpassURL}, map[string][]overpassReply{
		firstOverpassURL: {{status: http.StatusOK, body: overpassFixture(t)}},
	})

	stays, err := api.StaysNear(context.Background(), 37.1251, -7.6501)
	require.NoError(t, err)

	return stays
}

func TestOverpass_ReturnsStaysFromTheFirstServer(t *testing.T) {
	api, calls, _ := overpassStub(t, []string{firstOverpassURL, secondOverpassURL}, map[string][]overpassReply{
		firstOverpassURL:  {{status: http.StatusOK, body: overpassFixture(t)}},
		secondOverpassURL: {{status: http.StatusOK, body: secondServerBody}},
	})

	stays, err := api.StaysNear(context.Background(), 37.1251, -7.6501)

	require.NoError(t, err)
	require.Len(t, stays, 2)
	assert.Equal(t, "English hotel", stays[0].Name)
	require.Len(t, *calls, 1, "the second server must not be asked when the first one answers")
	assert.Equal(t, firstOverpassURL, (*calls)[0].url)
}

func TestOverpass_RetriesOnceBeforeMovingOn(t *testing.T) {
	api, calls, _ := overpassStub(t, []string{firstOverpassURL, secondOverpassURL}, map[string][]overpassReply{
		firstOverpassURL: {
			{status: http.StatusGatewayTimeout, body: "gateway timeout"},
			{status: http.StatusOK, body: overpassFixture(t)},
		},
		secondOverpassURL: {{status: http.StatusOK, body: secondServerBody}},
	})

	stays, err := api.StaysNear(context.Background(), 37.1251, -7.6501)

	require.NoError(t, err)
	require.Len(t, stays, 2)
	assert.Equal(t, "English hotel", stays[0].Name, "the retry on the first server should have won")
	require.Len(t, *calls, 2)
	assert.Equal(t, firstOverpassURL, (*calls)[0].url)
	assert.Equal(t, firstOverpassURL, (*calls)[1].url)
}

func TestOverpass_FallsBackAfterTwoFailures(t *testing.T) {
	api, calls, _ := overpassStub(t, []string{firstOverpassURL, secondOverpassURL}, map[string][]overpassReply{
		firstOverpassURL: {
			{status: http.StatusGatewayTimeout, body: "gateway timeout"},
			{status: http.StatusGatewayTimeout, body: "gateway timeout"},
		},
		secondOverpassURL: {{status: http.StatusOK, body: secondServerBody}},
	})

	stays, err := api.StaysNear(context.Background(), 37.1251, -7.6501)

	require.NoError(t, err)
	require.Len(t, stays, 1)
	assert.Equal(t, "Second server hotel", stays[0].Name)
	require.Len(t, *calls, 3)
	assert.Equal(t, secondOverpassURL, (*calls)[2].url)
}

func TestOverpass_ErrorNamesEveryFailedServer(t *testing.T) {
	api, _, _ := overpassStub(t, []string{firstOverpassURL, secondOverpassURL}, map[string][]overpassReply{
		firstOverpassURL: {
			{status: http.StatusGatewayTimeout, body: "gateway timeout"},
			{status: http.StatusGatewayTimeout, body: "gateway timeout"},
		},
		secondOverpassURL: {
			{err: errors.New("dial tcp: connection refused")},
			{err: errors.New("dial tcp: connection refused")},
		},
	})

	_, err := api.StaysNear(context.Background(), 37.1251, -7.6501)

	require.Error(t, err)
	assert.ErrorContains(t, err, firstOverpassURL)
	assert.ErrorContains(t, err, secondOverpassURL)
}

func TestOverpass_WaitsForRetryAfterUpToFiveSeconds(t *testing.T) {
	api, _, slept := overpassStub(t, []string{firstOverpassURL}, map[string][]overpassReply{
		firstOverpassURL: {
			{status: http.StatusTooManyRequests, body: "slow down", headers: map[string]string{"Retry-After": "30"}},
			{status: http.StatusOK, body: overpassFixture(t)},
		},
	})

	stays, err := api.StaysNear(context.Background(), 37.1251, -7.6501)

	require.NoError(t, err)
	require.Len(t, stays, 2)
	assert.Equal(t, []time.Duration{5 * time.Second}, *slept, "Retry-After must be honored but capped at 5s")
}

func TestOverpass_SkipsUnnamedPlaces(t *testing.T) {
	stays := taviraStays(t)

	require.Len(t, stays, 2, "the unnamed hostel must be skipped")
	for _, stay := range stays {
		assert.NotEmpty(t, stay.Name)
		assert.NotEqual(t, "hostel", stay.Kind)
	}
}

func TestOverpass_PrefersTheEnglishName(t *testing.T) {
	stays := taviraStays(t)

	require.NotEmpty(t, stays)
	assert.Equal(t, "English hotel", stays[0].Name)
}

func TestOverpass_ReadsTheWebsiteFromContactWebsite(t *testing.T) {
	stays := taviraStays(t)

	require.NotEmpty(t, stays)
	assert.Equal(t, "https://example.com", stays[0].Website)
}

func TestOverpass_ReadsStarsLike4SAsFour(t *testing.T) {
	stays := taviraStays(t)

	require.NotEmpty(t, stays)
	assert.Equal(t, 4, stays[0].Stars)
}

func TestOverpass_StarsAreZeroWhenMissing(t *testing.T) {
	stays := taviraStays(t)

	require.Len(t, stays, 2)
	assert.Equal(t, "Guest house", stays[1].Name)
	assert.Equal(t, 0, stays[1].Stars)
}

func TestOverpass_UsesTheWayCenterForCoordinates(t *testing.T) {
	stays := taviraStays(t)

	require.Len(t, stays, 2)
	assert.Equal(t, "guest_house", stays[1].Kind)
	assert.InDelta(t, 37.125, stays[1].Lat, 0.00001)
	assert.InDelta(t, -7.649, stays[1].Lon, 0.00001)
}

func TestOverpass_SendsTheQueryAndUserAgent(t *testing.T) {
	api, calls, _ := overpassStub(t, []string{firstOverpassURL}, map[string][]overpassReply{
		firstOverpassURL: {{status: http.StatusOK, body: overpassFixture(t)}},
	})

	_, err := api.StaysNear(context.Background(), 37.1251, -7.6501)

	require.NoError(t, err)
	require.Len(t, *calls, 1)
	sent := (*calls)[0]
	assert.Equal(t, http.MethodPost, sent.method)
	assert.Equal(t, "application/x-www-form-urlencoded", sent.header.Get("Content-Type"))
	assert.Equal(t, UserAgent, sent.header.Get("User-Agent"))
	assert.Equal(t,
		`[out:json];nwr["tourism"~"^(hotel|hostel|guest_house)$"](around:1000,37.1251,-7.6501); out center tags;`,
		sent.data)
}

func TestOverpassLive_FindsHotelsInTavira(t *testing.T) {
	if os.Getenv("LIVE_API_TESTS") != "1" {
		t.Skip("set LIVE_API_TESTS=1 to run the live Overpass check")
	}

	stays, err := NewOverpassAPI(nil, nil).StaysNear(context.Background(), 37.1251, -7.6501)

	require.NoError(t, err)
	require.NotEmpty(t, stays)
	for _, stay := range stays {
		assert.NotEmpty(t, stay.Name)
	}
}
