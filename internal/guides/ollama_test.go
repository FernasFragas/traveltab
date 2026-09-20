package guides

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newStubOllamaClient answers /api/generate with the given response body, and records the
// requests it receives. No network is hit.
func newStubOllamaClient(status int, body string) (*OllamaClient, *[]*http.Request) {
	var requests []*http.Request

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req)

		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{},
		}, nil
	})}

	return NewOllamaClient(client, "http://localhost:11434", "mistral:7b"), &requests
}

func TestOllamaClient_GenerateIntro_PostsThePromptAndModel(t *testing.T) {
	client, requests := newStubOllamaClient(http.StatusOK, `{"response":"Lisbon is a coastal capital.","done":true}`)

	intro, err := client.GenerateIntro(context.Background(), "Lisbon", "Lisbon is the capital of Portugal near the Alfama district.")
	require.NoError(t, err)

	require.Len(t, *requests, 1)
	req := (*requests)[0]
	assert.Equal(t, http.MethodPost, req.Method)
	assert.Contains(t, req.URL.String(), "/api/generate")

	var sent generateRequest
	require.NoError(t, json.NewDecoder(req.Body).Decode(&sent))
	assert.Equal(t, "mistral:7b", sent.Model)
	assert.False(t, sent.Stream)
	assert.Contains(t, sent.Prompt, "Lisbon")
	assert.Contains(t, sent.Prompt, "Alfama")

	assert.Equal(t, "Lisbon is a coastal capital.", intro)
}

func TestOllamaClient_GenerateIntro_ReturnsErrorOnHTTPFailure(t *testing.T) {
	client, _ := newStubOllamaClient(http.StatusInternalServerError, `oops`)

	_, err := client.GenerateIntro(context.Background(), "Lisbon", "some source text")
	require.Error(t, err)
}

func TestOllamaClient_GenerateIntro_ReturnsErrorOnEmptyResponse(t *testing.T) {
	client, _ := newStubOllamaClient(http.StatusOK, `{"response":"","done":true}`)

	_, err := client.GenerateIntro(context.Background(), "Lisbon", "some source text")
	require.Error(t, err)
}
