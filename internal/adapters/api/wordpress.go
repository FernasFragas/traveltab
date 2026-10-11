package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	wordpressPerPage    = 100
	wordpressRetryDelay = 5 * time.Second
	wordpressListFields = "id,link,title,modified_gmt,content"
	wordpressTimeLayout = "2006-01-02T15:04:05"
	robotsAgent         = "traveltab"
	robotsMaxBytes      = 512 << 10
)

var (
	// ErrBlocked means a writer site answered 403. Mark the source blocked and stop requesting it.
	ErrBlocked = errors.New("wordpress: blocked by host (403)")
	// ErrDisallowed means the site's robots.txt disallows the request for TravelTab.
	ErrDisallowed = errors.New("wordpress: disallowed by robots.txt")

	errBadRequest = errors.New("status 400")
)

// WordPressClient reads public WordPress REST posts politely: every request, robots.txt included,
// waits for the shared host limiter, and robots.txt is checked (once per host) before any fetch.
// Post content is returned to the caller only; it is never logged or put in errors.
type WordPressClient struct {
	client  *http.Client
	limiter *HostLimiter
	sleep   func(context.Context, time.Duration) error // backoff before the timeout retry

	mu     sync.Mutex
	robots map[string]robotsRules
}

// NewWordPressClient takes the limiter shared with every other job that fetches writer sites.
func NewWordPressClient(client *http.Client, limiter *HostLimiter) *WordPressClient {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &WordPressClient{
		client:  client,
		limiter: limiter,
		sleep:   sleepContext,
		robots:  make(map[string]robotsRules),
	}
}

// WordPressPost is one listed post. Content is rendered HTML for in-memory processing only.
type WordPressPost struct {
	ID       int64
	Link     string
	Title    string
	Modified time.Time // UTC
	Content  string
}

// ListPosts returns one page (1-based, 100 posts) of posts modified after modifiedAfter, oldest
// change first. A zero modifiedAfter lists every post. more is false on the last page, and a page
// past the last one returns no posts and no error.
func (c *WordPressClient) ListPosts(ctx context.Context, apiBase string, modifiedAfter time.Time, page int) (posts []WordPressPost, more bool, err error) {
	params := url.Values{"_fields": {wordpressListFields}, "orderby": {"modified"}, "order": {"asc"}}
	if !modifiedAfter.IsZero() {
		params.Set("modified_after", modifiedAfter.UTC().Format(time.RFC3339))
	}
	var items []struct {
		ID          int64  `json:"id"`
		Link        string `json:"link"`
		ModifiedGMT string `json:"modified_gmt"`
		Title       struct {
			Rendered string `json:"rendered"`
		} `json:"title"`
		Content struct {
			Rendered string `json:"rendered"`
		} `json:"content"`
	}
	header, err := c.listPage(ctx, apiBase, page, params, &items)
	if err != nil || len(items) == 0 {
		return nil, false, err
	}
	posts = make([]WordPressPost, len(items))
	for i, item := range items {
		modified, _ := time.Parse(wordpressTimeLayout, item.ModifiedGMT)
		posts[i] = WordPressPost{
			ID:       item.ID,
			Link:     item.Link,
			Title:    html.UnescapeString(item.Title.Rendered),
			Modified: modified,
			Content:  item.Content.Rendered,
		}
	}
	return posts, morePages(header, page, len(items)), nil
}

// PostIDs returns one page (1-based, 100 ids) of every published post id, for the deleted-posts
// sweep. A page past the last one returns no ids and no error.
func (c *WordPressClient) PostIDs(ctx context.Context, apiBase string, page int) (ids []int64, more bool, err error) {
	var items []struct {
		ID int64 `json:"id"`
	}
	header, err := c.listPage(ctx, apiBase, page, url.Values{"_fields": {"id"}}, &items)
	if err != nil || len(items) == 0 {
		return nil, false, err
	}
	ids = make([]int64, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	return ids, morePages(header, page, len(items)), nil
}

// PostContent returns the rendered HTML of one post.
func (c *WordPressClient) PostContent(ctx context.Context, apiBase string, id int64) (string, error) {
	endpoint, err := url.Parse(apiBase)
	if err != nil {
		return "", fmt.Errorf("wordpress api base: %w", err)
	}
	endpoint = endpoint.JoinPath(strconv.FormatInt(id, 10))
	endpoint.RawQuery = url.Values{"_fields": {"content"}}.Encode()
	var post struct {
		Content struct {
			Rendered string `json:"rendered"`
		} `json:"content"`
	}
	if _, err := c.get(ctx, endpoint, &post); err != nil {
		return "", err
	}
	return post.Content.Rendered, nil
}

// listPage fetches one page into out. WordPress answers 400 for a page past the last one, which
// ends pagination cleanly (out stays empty); on page 1 a 400 is a real error.
func (c *WordPressClient) listPage(ctx context.Context, apiBase string, page int, params url.Values, out any) (http.Header, error) {
	endpoint, err := url.Parse(apiBase)
	if err != nil {
		return nil, fmt.Errorf("wordpress api base: %w", err)
	}
	params.Set("per_page", strconv.Itoa(wordpressPerPage))
	params.Set("page", strconv.Itoa(page))
	endpoint.RawQuery = params.Encode()

	header, err := c.get(ctx, endpoint, out)
	if errors.Is(err, errBadRequest) && page > 1 {
		return nil, nil
	}
	return header, err
}

// get checks robots.txt, then fetches endpoint into out, retrying once after a backoff if the
// request times out.
func (c *WordPressClient) get(ctx context.Context, endpoint *url.URL, out any) (http.Header, error) {
	if err := c.checkRobots(ctx, endpoint); err != nil {
		return nil, err
	}
	header, err := c.getOnce(ctx, endpoint, out)
	if isTimeout(err) && ctx.Err() == nil {
		if err := c.sleep(ctx, wordpressRetryDelay); err != nil {
			return nil, err
		}
		header, err = c.getOnce(ctx, endpoint, out)
	}
	return header, err
}

func (c *WordPressClient) getOnce(ctx context.Context, endpoint *url.URL, out any) (http.Header, error) {
	resp, err := c.do(ctx, endpoint.String(), endpoint.Host)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusForbidden:
		return nil, fmt.Errorf("%w: %s", ErrBlocked, endpoint.Host)
	case http.StatusBadRequest:
		return nil, fmt.Errorf("wordpress %s: %w", endpoint.Host, errBadRequest)
	default:
		return nil, fmt.Errorf("wordpress %s: status %d", endpoint.Host, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nil, fmt.Errorf("wordpress %s decode: %w", endpoint.Host, err)
	}
	return resp.Header, nil
}

// checkRobots returns ErrDisallowed when the host's robots.txt disallows endpoint. Rules are
// cached per host once fetched; a failed fetch is not cached.
func (c *WordPressClient) checkRobots(ctx context.Context, endpoint *url.URL) error {
	c.mu.Lock()
	rules, ok := c.robots[endpoint.Host]
	c.mu.Unlock()
	if !ok {
		var err error
		if rules, err = c.fetchRobots(ctx, endpoint); err != nil {
			return err
		}
		c.mu.Lock()
		c.robots[endpoint.Host] = rules
		c.mu.Unlock()
	}
	if !rules.allowed(endpoint.RequestURI()) {
		return fmt.Errorf("%w: %s", ErrDisallowed, endpoint.Host)
	}
	return nil
}

// fetchRobots reads robots.txt. A 403 blocks the host; any other 4xx means no rules.
func (c *WordPressClient) fetchRobots(ctx context.Context, endpoint *url.URL) (robotsRules, error) {
	robotsURL := url.URL{Scheme: endpoint.Scheme, Host: endpoint.Host, Path: "/robots.txt"}
	resp, err := c.do(ctx, robotsURL.String(), endpoint.Host)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode == http.StatusOK:
		body, err := io.ReadAll(io.LimitReader(resp.Body, robotsMaxBytes))
		if err != nil {
			return nil, fmt.Errorf("wordpress %s robots.txt: %w", endpoint.Host, err)
		}
		return parseRobots(string(body)), nil
	case resp.StatusCode == http.StatusForbidden:
		return nil, fmt.Errorf("%w: %s robots.txt", ErrBlocked, endpoint.Host)
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		return robotsRules{}, nil
	default:
		return nil, fmt.Errorf("wordpress %s robots.txt: status %d", endpoint.Host, resp.StatusCode)
	}
}

// do waits for the host limiter, then sends an identified GET.
func (c *WordPressClient) do(ctx context.Context, rawURL, host string) (*http.Response, error) {
	if err := c.limiter.Wait(ctx, host); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("wordpress %s request: %w", host, err)
	}
	req.Header.Set("User-Agent", UserAgent)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("wordpress %s: %w", host, err)
	}
	return resp, nil
}

// robotsRules are the Allow and Disallow lines of the group that applies to TravelTab.
type robotsRules []robotsRule

type robotsRule struct {
	allow   bool
	pattern string
}

// allowed applies the longest matching rule; Allow wins a tie and no match allows.
func (r robotsRules) allowed(path string) bool {
	best, allow := -1, true
	for _, rule := range r {
		if rule.pattern == "" || !robotsMatch(rule.pattern, path) {
			continue
		}
		if n := len(rule.pattern); n > best || (n == best && rule.allow) {
			best, allow = n, rule.allow
		}
	}
	return allow
}

// parseRobots keeps the rules for TravelTab's own group if there is one, otherwise the "*" group.
func parseRobots(text string) robotsRules {
	var own, wildcard robotsRules
	var agents []string
	ownFound, inRules := false, false
	for line := range strings.Lines(text) {
		line, _, _ = strings.Cut(line, "#")
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key, value = strings.ToLower(strings.TrimSpace(key)), strings.TrimSpace(value)
		switch key {
		case "user-agent":
			if inRules {
				agents, inRules = nil, false
			}
			agents = append(agents, strings.ToLower(value))
		case "allow", "disallow":
			inRules = true
			rule := robotsRule{allow: key == "allow", pattern: value}
			for _, agent := range agents {
				switch agent {
				case robotsAgent:
					own, ownFound = append(own, rule), true
				case "*":
					wildcard = append(wildcard, rule)
				}
			}
		}
	}
	if ownFound {
		return own
	}
	return wildcard
}

// robotsMatch matches a robots.txt path pattern with "*" wildcards and a trailing "$" anchor.
func robotsMatch(pattern, path string) bool {
	anchored := strings.HasSuffix(pattern, "$")
	parts := strings.Split(strings.TrimSuffix(pattern, "$"), "*")
	if !strings.HasPrefix(path, parts[0]) {
		return false
	}
	rest := path[len(parts[0]):]
	for i, part := range parts[1:] {
		if anchored && i == len(parts)-2 {
			return strings.HasSuffix(rest, part)
		}
		j := strings.Index(rest, part)
		if j < 0 {
			return false
		}
		rest = rest[j+len(part):]
	}
	return !anchored || rest == ""
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// morePages reads X-WP-TotalPages; without it, a non-empty page means there may be more.
func morePages(header http.Header, page, items int) bool {
	total, err := strconv.Atoi(header.Get("X-WP-TotalPages"))
	if err != nil {
		return items > 0
	}
	return page < total
}
