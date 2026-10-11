package writermap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// Source statuses. The allowlist uses active, blocked and removed; partial (a crawl or monthly
// update stopped part-way and will resume from its cursor) only appears in the sync state.
const (
	StatusActive  = "active"
	StatusPartial = "partial"
	StatusBlocked = "blocked"
	StatusRemoved = "removed"
)

// The off-peak window, in UTC hours [offPeakStart, offPeakEnd), when monthly updates run.
const (
	offPeakStart = 2
	offPeakEnd   = 6
)

// ErrSyncRunning means another run holds the single-run lock.
var ErrSyncRunning = errors.New("writers sync already running")

// SyncJob keeps the writers index up to date. A source's first crawl starts at once (and
// resumes after a restart); afterwards each source gets one update per calendar month, in the
// off-peak window: changed posts since the cursor, then a deleted-posts sweep. When any source
// finished a crawl or update, cached destinations are refreshed and the summary step runs.
// Only names, positions and post metadata are indexed or logged, never post text.
type SyncJob struct {
	sources []Source
	index   SyncIndex
	posts   PostLister

	// Destinations and Refresher re-run the base query of every cached destination; the step
	// is skipped when either is nil.
	Destinations CachedDestinations
	Refresher    DestinationRefresher
	// SummaryStep is the hook for the summary job (PR 9); nil skips it.
	SummaryStep func(ctx context.Context) error
	// BlockedErrors are lister errors that mark a source blocked so it is not requested again.
	BlockedErrors []error

	Now   func() time.Time
	Sleep func(ctx context.Context, d time.Duration) error
	Logf  func(format string, args ...any)

	running sync.Mutex
}

// NewSyncJob returns a job with the real clock and the standard logger.
func NewSyncJob(sources []Source, index SyncIndex, posts PostLister) *SyncJob {
	return &SyncJob{
		sources: sources,
		index:   index,
		posts:   posts,
		Now:     time.Now,
		Sleep:   sleepContext,
		Logf:    log.Printf,
	}
}

// Source is one allowlisted writer site from writerdata/sources.json.
type Source struct {
	Host     string `json:"host"`
	Name     string `json:"name"`
	APIBase  string `json:"api_base"`
	Language string `json:"language"`
	Status   string `json:"status"`
}

// ParseSources reads the allowlist file.
func ParseSources(data []byte) ([]Source, error) {
	var sources []Source
	if err := json.Unmarshal(data, &sources); err != nil {
		return nil, fmt.Errorf("parse writer sources: %w", err)
	}
	return sources, nil
}

// SyncIndex stores indexed posts and per-source sync progress. Every write is an upsert or a
// delete in place.
type SyncIndex interface {
	UpsertPost(ctx context.Context, post IndexedPost) error
	DeletePost(ctx context.Context, host string, postID int64) error
	PurgeHost(ctx context.Context, host string) error
	PostIDs(ctx context.Context, host string) ([]int64, error)
	SyncState(ctx context.Context, host string) (SyncState, error)
	SaveSyncState(ctx context.Context, state SyncState) error
}

// IndexedPost is the metadata and extracted positions kept for one post.
type IndexedPost struct {
	Host      string
	PostID    int64
	URL       string
	Title     string
	Language  string
	Modified  time.Time
	Extracted ExtractedPost
}

// SyncState is one source's status and progress. Cursor is the latest indexed modified time.
type SyncState struct {
	Host      string
	Status    string
	LastFull  time.Time
	LastDelta time.Time
	Cursor    string
}

// PostLister reads a writer site one page at a time. ListPosts returns posts modified after
// modifiedAfter (all posts when it is zero), oldest change first; PostIDs lists every post id.
type PostLister interface {
	ListPosts(ctx context.Context, apiBase string, modifiedAfter time.Time, page int) ([]SourcePost, bool, error)
	PostIDs(ctx context.Context, apiBase string, page int) ([]int64, bool, error)
}

// SourcePost is one listed post. Content is processed in memory and dropped.
type SourcePost struct {
	ID       int64
	Link     string
	Title    string
	Modified time.Time
	Content  string
}

// CachedDestinations lists the destinations whose base places are cached.
type CachedDestinations interface {
	CachedDestinations(ctx context.Context) ([]Area, error)
}

// DestinationRefresher re-runs one destination's base query and replaces its cached places.
type DestinationRefresher interface {
	Refresh(ctx context.Context, dest Area) error
}

// Schedule runs once at startup, then at the start of every off-peak window, until ctx ends.
func (j *SyncJob) Schedule(ctx context.Context) {
	for {
		if err := j.Run(ctx); err != nil {
			j.Logf("writers sync: %v", err)
		}
		if err := j.Sleep(ctx, untilOffPeak(j.Now())); err != nil {
			return
		}
	}
}

// Run syncs every source that is due, one at a time, then refreshes cached destinations and
// runs the summary step if any source finished. It returns ErrSyncRunning if a run is active.
func (j *SyncJob) Run(ctx context.Context) error {
	if !j.running.TryLock() {
		return ErrSyncRunning
	}
	defer j.running.Unlock()

	offPeak := inOffPeak(j.Now())
	var errs []error
	finished := false
	for _, source := range j.sources {
		outcome, done, err := j.syncSource(ctx, source, offPeak)
		if err != nil {
			j.Logf("writers sync %s: %s: %v", source.Host, outcome, err)
			errs = append(errs, fmt.Errorf("%s: %w", source.Host, err))
		} else {
			j.Logf("writers sync %s: %s", source.Host, outcome)
		}
		finished = finished || done
	}
	if finished {
		errs = append(errs, j.refreshDestinations(ctx))
		if j.SummaryStep != nil {
			errs = append(errs, j.SummaryStep(ctx))
		}
	}
	return errors.Join(errs...)
}

// syncSource handles one source and describes the outcome for the progress line. done is
// true when a crawl or monthly update finished.
func (j *SyncJob) syncSource(ctx context.Context, source Source, offPeak bool) (outcome string, done bool, err error) {
	switch source.Status {
	case StatusRemoved:
		if err := j.index.PurgeHost(ctx, source.Host); err != nil {
			return "purge failed", false, err
		}
		return "removed, rows purged", false, nil
	case StatusBlocked:
		return "skipped, blocked in the allowlist", false, nil
	}
	state, err := j.index.SyncState(ctx, source.Host)
	if err != nil {
		return "sync state unavailable", false, err
	}
	if state.Status == StatusBlocked {
		return "skipped, blocked", false, nil
	}
	crawl := state.LastFull.IsZero()
	due := crawl || offPeak && (state.Status == StatusPartial || monthlyDue(state, j.Now()))
	if !due {
		return "not due", false, nil
	}

	state.Status = StatusPartial
	indexed, err := j.indexChanges(ctx, source, &state)
	deleted := 0
	if err == nil && !crawl {
		deleted, err = j.sweepDeleted(ctx, source)
	}
	if err != nil && j.blocked(err) {
		state.Status = StatusBlocked
		if saveErr := j.index.SaveSyncState(ctx, state); saveErr != nil {
			return "blocked", false, saveErr
		}
		return fmt.Sprintf("blocked after %d posts, not requested again", indexed), false, nil
	}
	if err != nil {
		return fmt.Sprintf("stopped after %d posts, will resume", indexed), false, err
	}

	state.Status = StatusActive
	if crawl {
		state.LastFull = j.Now()
		outcome = fmt.Sprintf("crawled %d posts", indexed)
	} else {
		state.LastDelta = j.Now()
		outcome = fmt.Sprintf("updated %d posts, removed %d", indexed, deleted)
	}
	if err := j.index.SaveSyncState(ctx, state); err != nil {
		return outcome, false, err
	}
	return outcome, true, nil
}

// indexChanges indexes posts modified after the cursor, one page at a time, saving the cursor
// after each page. Each page starts one second before the cursor (re-indexing is an idempotent
// upsert), so posts sharing the cursor's second are not skipped and an edit mid-crawl cannot
// shift an unread post onto an earlier page. If the cursor did not move past that overlap, the
// next page of the same listing is read instead. A post re-listed unchanged in the same run is
// not indexed twice.
func (j *SyncJob) indexChanges(ctx context.Context, source Source, state *SyncState) (int, error) {
	cursor, err := parseCursor(state.Cursor)
	if err != nil {
		return 0, err
	}
	after := overlap(cursor)
	indexed := make(map[int64]time.Time)
	for page := 1; ; {
		posts, more, err := j.posts.ListPosts(ctx, source.APIBase, after, page)
		if err != nil {
			return len(indexed), err
		}
		for _, post := range posts {
			if modified, ok := indexed[post.ID]; ok && modified.Equal(post.Modified) {
				continue
			}
			if err := j.index.UpsertPost(ctx, IndexedPost{
				Host: source.Host, PostID: post.ID, URL: post.Link, Title: post.Title,
				Language: source.Language, Modified: post.Modified,
				Extracted: ExtractPost(post.Title, post.Content),
			}); err != nil {
				return len(indexed), err
			}
			indexed[post.ID] = post.Modified
			if post.Modified.After(cursor) {
				cursor = post.Modified
			}
		}
		state.Cursor = formatCursor(cursor)
		if err := j.index.SaveSyncState(ctx, *state); err != nil {
			return len(indexed), err
		}
		if !more {
			return len(indexed), nil
		}
		if next := overlap(cursor); next.After(after) {
			after, page = next, 1
		} else {
			page++
		}
	}
}

// sweepDeleted drops stored posts the site no longer lists. Pages can shift while a sweep
// runs, so an id missing from one sweep is only deleted if a second sweep also misses it. A
// failed or empty sweep deletes nothing.
func (j *SyncJob) sweepDeleted(ctx context.Context, source Source) (int, error) {
	stored, err := j.index.PostIDs(ctx, source.Host)
	if err != nil || len(stored) == 0 {
		return 0, err
	}
	missing, err := j.missingIDs(ctx, source, stored)
	if err != nil {
		return 0, err
	}
	if len(missing) > 0 {
		if missing, err = j.missingIDs(ctx, source, missing); err != nil {
			return 0, err
		}
	}
	for _, id := range missing {
		if err := j.index.DeletePost(ctx, source.Host, id); err != nil {
			return 0, err
		}
	}
	return len(missing), nil
}

// missingIDs returns the candidates absent from one complete id sweep.
func (j *SyncJob) missingIDs(ctx context.Context, source Source, candidates []int64) ([]int64, error) {
	missing := make(map[int64]bool, len(candidates))
	for _, id := range candidates {
		missing[id] = true
	}
	seen := 0
	for page, more := 1, true; more; page++ {
		var ids []int64
		var err error
		if ids, more, err = j.posts.PostIDs(ctx, source.APIBase, page); err != nil {
			return nil, err
		}
		seen += len(ids)
		for _, id := range ids {
			delete(missing, id)
		}
	}
	if seen == 0 {
		return nil, errors.New("deleted-posts sweep listed no posts")
	}
	var result []int64
	for _, id := range candidates {
		if missing[id] {
			result = append(result, id)
		}
	}
	return result, nil
}

// refreshDestinations re-runs every cached destination's base query, continuing past errors.
func (j *SyncJob) refreshDestinations(ctx context.Context) error {
	if j.Destinations == nil || j.Refresher == nil {
		return nil
	}
	areas, err := j.Destinations.CachedDestinations(ctx)
	if err != nil {
		return fmt.Errorf("list cached destinations: %w", err)
	}
	var errs []error
	for _, area := range areas {
		if err := j.Refresher.Refresh(ctx, area); err != nil {
			errs = append(errs, err)
		}
	}
	j.Logf("writers sync: refreshed %d of %d cached destinations", len(areas)-len(errs), len(areas))
	return errors.Join(errs...)
}

func (j *SyncJob) blocked(err error) bool {
	for _, target := range j.BlockedErrors {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// monthlyDue reports whether the source's last finished sync was in an earlier month.
func monthlyDue(state SyncState, now time.Time) bool {
	last := state.LastFull
	if state.LastDelta.After(last) {
		last = state.LastDelta
	}
	ly, lm, _ := last.UTC().Date()
	ny, nm, _ := now.UTC().Date()
	return ly < ny || (ly == ny && lm < nm)
}

func inOffPeak(now time.Time) bool {
	hour := now.UTC().Hour()
	return hour >= offPeakStart && hour < offPeakEnd
}

// untilOffPeak returns the time until the next start of the off-peak window after now.
func untilOffPeak(now time.Time) time.Duration {
	now = now.UTC()
	next := time.Date(now.Year(), now.Month(), now.Day(), offPeakStart, 0, 0, 0, time.UTC)
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next.Sub(now)
}

func overlap(cursor time.Time) time.Time {
	if cursor.IsZero() {
		return cursor
	}
	return cursor.Add(-time.Second)
}

func parseCursor(cursor string) (time.Time, error) {
	if cursor == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, cursor)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse sync cursor: %w", err)
	}
	return parsed, nil
}

func formatCursor(cursor time.Time) string {
	if cursor.IsZero() {
		return ""
	}
	return cursor.UTC().Format(time.RFC3339Nano)
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
