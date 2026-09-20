package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"weatherservice/internal/planner"
)

// DefaultOverpassURLs are the public Overpass servers, tried in this order.
var DefaultOverpassURLs = []string{
	"https://overpass-api.de/api/interpreter",
	"https://overpass.private.coffee/api/interpreter",
}

const (
	// overpassTimeout is how long one attempt against one server may take.
	overpassTimeout = 10 * time.Second
	// overpassMaxRetryAfter caps how long a Retry-After header may hold us up.
	overpassMaxRetryAfter = 5 * time.Second
	// overpassAttempts is one try plus one retry per server.
	overpassAttempts = 2
)

// OverpassAPI reads places to stay from OpenStreetMap through Overpass.
type OverpassAPI struct {
	client *http.Client
	urls   []string
	sleep  func(time.Duration)
}

var _ planner.StaySource = (*OverpassAPI)(nil)

// NewOverpassAPI returns a stay source. A nil client means a default client
// with a 10 s timeout, and empty urls mean DefaultOverpassURLs.
func NewOverpassAPI(client *http.Client, urls []string) *OverpassAPI {
	if client == nil {
		client = &http.Client{Timeout: overpassTimeout}
	}

	if len(urls) == 0 {
		urls = DefaultOverpassURLs
	}

	return &OverpassAPI{client: client, urls: urls, sleep: time.Sleep}
}

// overpassResponse is the slice of the Overpass answer we use. Nodes carry
// lat/lon directly, while ways and relations carry a center.
type overpassResponse struct {
	Elements []struct {
		Type   string  `json:"type"`
		Lat    float64 `json:"lat"`
		Lon    float64 `json:"lon"`
		Center *struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		} `json:"center"`
		Tags map[string]string `json:"tags"`
	} `json:"elements"`
}

// StaysNear returns the hotels, hostels and guest houses within StayRadiusM of
// the point. Each server gets one retry before the next one is tried.
func (api *OverpassAPI) StaysNear(ctx context.Context, lat, lon float64) ([]planner.Stay, error) {
	query := overpassQuery(lat, lon)

	var failures []error
	for _, server := range api.urls {
		body, err := api.askServer(ctx, server, query)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", server, err))
			continue
		}

		var decoded overpassResponse
		if err := json.Unmarshal(body, &decoded); err != nil {
			failures = append(failures, fmt.Errorf("%s: decode: %w", server, err))
			continue
		}

		stays := make([]planner.Stay, 0, len(decoded.Elements))
		for _, element := range decoded.Elements {
			lat, lon := element.Lat, element.Lon
			if element.Center != nil {
				lat, lon = element.Center.Lat, element.Center.Lon
			}

			if stay, ok := stayFromTags(element.Tags, lat, lon); ok {
				stays = append(stays, stay)
			}
		}

		return stays, nil
	}

	return nil, fmt.Errorf("overpass: every server failed: %w", errors.Join(failures...))
}

// overpassQuery builds the Overpass QL query. The [out:json] setting is what
// makes the server answer with JSON instead of XML.
func overpassQuery(lat, lon float64) string {
	return fmt.Sprintf(
		`[out:json];nwr["tourism"~"^(hotel|hostel|guest_house)$"](around:%d,%s,%s); out center tags;`,
		planner.StayRadiusM,
		strconv.FormatFloat(lat, 'f', -1, 64),
		strconv.FormatFloat(lon, 'f', -1, 64),
	)
}

// askServer sends the query to one server, retrying once on 429, 5xx, a
// timeout or a transport error.
func (api *OverpassAPI) askServer(ctx context.Context, server, query string) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt < overpassAttempts; attempt++ {
		body, wait, err := api.attempt(ctx, server, query)
		if err == nil {
			return body, nil
		}

		lastErr = err
		if attempt == overpassAttempts-1 {
			break
		}

		if wait > 0 && api.sleep != nil {
			api.sleep(wait)
		}
	}

	return nil, lastErr
}

// attempt makes one request. It returns how long the server asked us to wait
// before a retry, already capped at overpassMaxRetryAfter.
func (api *OverpassAPI) attempt(ctx context.Context, server, query string) ([]byte, time.Duration, error) {
	ctx, cancel := context.WithTimeout(ctx, overpassTimeout)
	defer cancel()

	form := url.Values{"data": {query}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, 0, fmt.Errorf("request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, retryAfter(resp.Header.Get("Retry-After")), fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("read body: %w", err)
	}

	return body, 0, nil
}

// retryAfter reads a Retry-After header given in seconds, capped at
// overpassMaxRetryAfter. Anything it cannot read means no wait.
func retryAfter(header string) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(header))
	if err != nil || seconds <= 0 {
		return 0
	}

	wait := time.Duration(seconds) * time.Second
	if wait > overpassMaxRetryAfter {
		return overpassMaxRetryAfter
	}

	return wait
}

// stayFromTags turns OpenStreetMap tags into a planner.Stay. Places without a
// name are dropped, because the card has nothing to show for them.
func stayFromTags(tags map[string]string, lat, lon float64) (planner.Stay, bool) {
	name := tags["name:en"]
	if name == "" {
		name = tags["name"]
	}
	if name == "" {
		return planner.Stay{}, false
	}

	website := tags["website"]
	if website == "" {
		website = tags["contact:website"]
	}

	return planner.Stay{
		Name:    name,
		Kind:    tags["tourism"],
		Lat:     lat,
		Lon:     lon,
		Website: website,
		Stars:   parseStars(tags["stars"]),
	}, true
}

// parseStars reads the leading digits of an OpenStreetMap stars tag, so "4S"
// and "4" both mean four stars. Anything else means unknown.
func parseStars(value string) int {
	digits := 0
	for digits < len(value) && value[digits] >= '0' && value[digits] <= '9' {
		digits++
	}

	if digits == 0 {
		return 0
	}

	stars, err := strconv.Atoi(value[:digits])
	if err != nil {
		return 0
	}

	return stars
}
