package guides

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// roundTripFunc mirrors the fake-transport pattern used in internal/adapters/api (e.g.
// openweather_test.go): no network is hit, every request is recorded, and the stub answers
// whatever the test needs.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// newStubWikivoyageAPI answers every request with the given body and status, and records the
// requests it receives.
func newStubWikivoyageAPI(status int, body string) (*WikivoyageAPI, *[]*http.Request) {
	var requests []*http.Request

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req)

		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{},
		}, nil
	})}

	return NewWikivoyageAPI(client), &requests
}

const lisbonWikivoyageResponse = `{"batchcomplete":"","query":{"pages":{"19798":{"pageid":19798,"ns":0,"title":"Lisbon","extract":"Lisbon is the capital of Portugal. The historic Alfama district sits below São Jorge Castle.","revisions":[{"revid":5364237,"parentid":5364227}]}}}}`

const missingPageResponse = `{"batchcomplete":"","query":{"pages":{"-1":{"ns":0,"title":"Nowhereville","missing":""}}}}`

func TestFetchWikivoyageText_ReturnsPlainTextAndRevision(t *testing.T) {
	api, requests := newStubWikivoyageAPI(http.StatusOK, lisbonWikivoyageResponse)

	page, err := api.FetchText(context.Background(), "Lisbon")
	require.NoError(t, err)

	require.Len(t, *requests, 1)
	req := (*requests)[0]
	assert.Equal(t, "Lisbon", req.URL.Query().Get("titles"))
	assert.Equal(t, "extracts|revisions", req.URL.Query().Get("prop"))
	assert.Equal(t, "1", req.URL.Query().Get("explaintext"))
	assert.NotEmpty(t, req.Header.Get("User-Agent"))

	assert.Contains(t, page.Text, "Alfama")
	assert.Equal(t, int64(5364237), page.Revision)
	assert.Equal(t, "Lisbon", page.Title)
}

func TestFetchWikivoyageText_ReturnsErrorForAMissingPage(t *testing.T) {
	api, _ := newStubWikivoyageAPI(http.StatusOK, missingPageResponse)

	_, err := api.FetchText(context.Background(), "Nowhereville")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Nowhereville")
}

func TestFetchWikivoyageText_ReturnsErrorOnHTTPFailure(t *testing.T) {
	api, _ := newStubWikivoyageAPI(http.StatusInternalServerError, `oops`)

	_, err := api.FetchText(context.Background(), "Lisbon")
	require.Error(t, err)
}
