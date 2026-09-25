// Package guides fetches Wikivoyage city text, asks a local Ollama model for a short intro
// grounded in that text, validates it names no place the source doesn't mention, and writes the
// results to guides/guides.json for a human to review. Generation is used only by cmd/guides and
// never runs in the web server. cmd/web reads the reviewed result through Book, which answers
// from memory and publishes only entries a person has marked reviewed.
package guides

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"weatherservice/internal/adapters/api"
)

const (
	wikivoyageAPIURL  = "https://en.wikivoyage.org/w/api.php"
	wikivoyageTimeout = 30 * time.Second
)

// WikivoyagePage is the plain text of one Wikivoyage article and the revision it was read at.
type WikivoyagePage struct {
	Title    string // the article title after redirects
	Text     string
	Revision int64
}

// WikivoyageAPI fetches plain-text Wikivoyage articles from the MediaWiki Action API. It needs
// no key.
type WikivoyageAPI struct {
	client  *http.Client
	baseURL string
}

// NewWikivoyageAPI returns a client for the Wikivoyage API. A nil client means a default one
// with a 30 second timeout.
func NewWikivoyageAPI(client *http.Client) *WikivoyageAPI {
	if client == nil {
		client = &http.Client{Timeout: wikivoyageTimeout}
	}

	return &WikivoyageAPI{client: client, baseURL: wikivoyageAPIURL}
}

// wikivoyagePage is one page in the Action API's response.
type wikivoyagePage struct {
	Title     string  `json:"title"`
	Missing   *string `json:"missing"`
	Extract   string  `json:"extract"`
	Revisions []struct {
		RevID int64 `json:"revid"`
	} `json:"revisions"`
}

// wikivoyageResponse is the shape of a query&prop=extracts|revisions response.
type wikivoyageResponse struct {
	Query struct {
		Pages map[string]wikivoyagePage `json:"pages"`
	} `json:"query"`
}

// FetchText returns the plain text of a Wikivoyage article and its current revision ID. It
// follows redirects (e.g. "Marrakesh" to the article actually titled "Marrakech").
func (a *WikivoyageAPI) FetchText(ctx context.Context, title string) (WikivoyagePage, error) {
	req, err := a.buildRequest(ctx, title)
	if err != nil {
		return WikivoyagePage{}, err
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return WikivoyagePage{}, fmt.Errorf("fetching wikivoyage page %q: %w", title, err)
	}

	body, err := readBody(resp)
	if err != nil {
		return WikivoyagePage{}, err
	}

	return parseWikivoyageResponse(title, body)
}

// buildRequest is split out so the tests can inspect exactly what gets sent.
func (a *WikivoyageAPI) buildRequest(ctx context.Context, title string) (*http.Request, error) {
	q := url.Values{}
	q.Set("action", "query")
	q.Set("prop", "extracts|revisions")
	q.Set("explaintext", "1")
	q.Set("rvprop", "ids")
	q.Set("redirects", "1")
	q.Set("titles", title)
	q.Set("format", "json")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("building wikivoyage request for %q: %w", title, err)
	}

	req.Header.Set("User-Agent", api.UserAgent)

	return req, nil
}

// parseWikivoyageResponse turns the API body into a WikivoyagePage, or an error naming why the
// page could not be used (missing, empty, or malformed).
func parseWikivoyageResponse(title string, body []byte) (WikivoyagePage, error) {
	var parsed wikivoyageResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return WikivoyagePage{}, fmt.Errorf("decoding wikivoyage response for %q: %w", title, err)
	}

	for _, page := range parsed.Query.Pages {
		if page.Missing != nil {
			return WikivoyagePage{}, fmt.Errorf("wikivoyage page %q not found", title)
		}

		if page.Extract == "" {
			return WikivoyagePage{}, fmt.Errorf("wikivoyage page %q has no extract", title)
		}

		var revision int64
		if len(page.Revisions) > 0 {
			revision = page.Revisions[0].RevID
		}

		return WikivoyagePage{Title: page.Title, Text: page.Extract, Revision: revision}, nil
	}

	return WikivoyagePage{}, fmt.Errorf("wikivoyage page %q: no pages in response", title)
}

// readBody is a small helper kept separate so FetchText's HTTP handling stays short.
func readBody(resp *http.Response) ([]byte, error) {
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading wikivoyage response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wikivoyage: status %d: %s", resp.StatusCode, body)
	}

	return body, nil
}
