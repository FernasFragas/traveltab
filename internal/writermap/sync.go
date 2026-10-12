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

// Source statuses used by the allowlist and the sync state.
const (
	StatusActive  = "active"
	StatusBlocked = "blocked"
	StatusRemoved = "removed"
)

// The off-peak window, in UTC hours [offPeakStart, offPeakEnd), when monthly updates run.
const (
	offPeakStart = 2
	offPeakEnd   = 6
)

// commentPages is how many pages of recent comments (100 each) are tallied per blog.
const commentPages = 10

// ErrSyncRunning means another run holds the single-run lock.
var ErrSyncRunning = errors.New("writers sync already running")

// SyncJob keeps the writers index at no more than MaxIndexedPosts posts, picked afresh on every
// run: for each active blog it searches the destination names, keeps the hits whose title or
// slug names a known place, ranks them by recent comments, and Select shares the cap across blogs.
// Only the chosen posts are fetched and indexed; indexed posts that were not chosen are deleted.
// Rows of a blocked blog, or of a blog with no destination candidates left, stay and count
// inside the cap; the other blogs share what is left. If the total is still over MaxIndexedPosts
// after the run (a blog's store failed and kept its old rows), the lowest-ranked rows of the
// blogs updated in this run are deleted until it is exactly the cap.
// The first run for a source starts at once; afterwards a run is due once per calendar month, in
// the off-peak window. When a run finishes, cached destinations are refreshed, and the summary
// step runs if any post was added or removed. Only names, positions and post metadata are
// indexed or logged, never post text.
type SyncJob struct {
	sources []Source
	index   SyncIndex
	posts   PostSource

	// Areas are the destination areas (writerdata/overrides.json). Their names, with the cached
	// destinations' names, are the search terms and count as known places.
	Areas []Area
	// PlaceNames lists more known place names (the matcher's names for the cached destinations);
	// nil means the destination names alone.
	PlaceNames func(ctx context.Context) ([]string, error)
	// Destinations and Refresher re-run the base query of every cached destination; the step
	// is skipped when either is nil. Destinations also adds search terms.
	Destinations CachedDestinations
	Refresher    DestinationRefresher
	// SummaryStep is the hook for the summary job (PR 9); nil skips it.
	SummaryStep func(ctx context.Context) error
	// BlockedErrors are source errors that mark a source blocked so it is not requested again.
	BlockedErrors []error

	Now   func() time.Time
	Sleep func(ctx context.Context, d time.Duration) error
	Logf  func(format string, args ...any)

	running sync.Mutex
}

// NewSyncJob returns a job with the real clock and the standard logger.
func NewSyncJob(sources []Source, index SyncIndex, posts PostSource) *SyncJob {
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

// SyncState is one source's status and progress. LastFull is when its last run finished. LastDelta
// and Cursor are no longer written; the SQLite schema still holds them.
type SyncState struct {
	Host      string
	Status    string
	LastFull  time.Time
	LastDelta time.Time
	Cursor    string
}

// PostSource reads a writer site. SearchPosts returns one page of search hits (metadata only),
// RecentCommentCounts tallies the newest comments per post ID (empty when the blog shows none),
// and PostsByID fetches full posts, used only for the posts chosen for the index.
type PostSource interface {
	SearchPosts(ctx context.Context, apiBase, term string, page int) ([]SearchHit, bool, error)
	RecentCommentCounts(ctx context.Context, apiBase string, maxPages int) (map[int64]int, error)
	PostsByID(ctx context.Context, apiBase string, ids []int64) ([]SourcePost, error)
}

// SourcePost is one fetched post. Content is processed in memory and dropped.
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

// Run syncs the writers index if any source is due, then refreshes cached destinations and
// runs the summary step if a post was added or removed. It returns ErrSyncRunning if a run is
// active.
func (j *SyncJob) Run(ctx context.Context) error {
	if !j.running.TryLock() {
		return ErrSyncRunning
	}
	defer j.running.Unlock()

	active, held, errs := j.activeSources(ctx)
	if !j.due(active) {
		for _, source := range active {
			j.Logf("writers sync %s: not due", source.source.Host)
		}
		return errors.Join(errs...)
	}
	finished, changed, err := j.syncSources(ctx, active, held)
	errs = append(errs, err)
	if finished {
		errs = append(errs, j.refreshDestinations(ctx))
		if changed && j.SummaryStep != nil {
			errs = append(errs, j.SummaryStep(ctx))
		}
	}
	return errors.Join(errs...)
}

// trackedSource pairs a source with its stored state.
type trackedSource struct {
	source Source
	state  SyncState
}

// activeSources purges removed sources, skips blocked ones, and returns the rest with their
// state. The hosts of the blocked ones are returned too: their rows stay and count in the cap.
func (j *SyncJob) activeSources(ctx context.Context) (active []trackedSource, held []string, errs []error) {
	for _, source := range j.sources {
		switch source.Status {
		case StatusRemoved:
			if err := j.index.PurgeHost(ctx, source.Host); err != nil {
				j.Logf("writers sync %s: purge failed: %v", source.Host, err)
				errs = append(errs, fmt.Errorf("%s: %w", source.Host, err))
				continue
			}
			j.Logf("writers sync %s: removed, rows purged", source.Host)
			continue
		case StatusBlocked:
			j.Logf("writers sync %s: skipped, blocked in the allowlist", source.Host)
			held = append(held, source.Host)
			continue
		}
		state, err := j.index.SyncState(ctx, source.Host)
		if err != nil {
			j.Logf("writers sync %s: sync state unavailable: %v", source.Host, err)
			errs = append(errs, fmt.Errorf("%s: %w", source.Host, err))
			continue
		}
		if state.Status == StatusBlocked {
			j.Logf("writers sync %s: skipped, blocked", source.Host)
			held = append(held, source.Host)
			continue
		}
		active = append(active, trackedSource{source: source, state: state})
	}
	return active, held, errs
}

// due reports whether any source needs a run: one that has never finished (this includes a
// run that failed before saving), or, in the off-peak window, one whose last run was in an
// earlier month. The cap is shared, so one due source re-picks every blog.
func (j *SyncJob) due(active []trackedSource) bool {
	offPeak := inOffPeak(j.Now())
	for _, entry := range active {
		if entry.state.LastFull.IsZero() || offPeak && monthlyDue(entry.state, j.Now()) {
			return true
		}
	}
	return false
}

// syncSources gathers every blog's candidates first, so the cap is applied to the whole set,
// then fetches and indexes the chosen posts. A failure while gathering stops the run before any
// write, so a half-seen set never purges rows. finished is true when at least one source
// completed; changed is true when a post was added or removed. held are the hosts whose rows
// stay as they are (blocked blogs); a blog found blocked or with no destination posts joins them.
func (j *SyncJob) syncSources(ctx context.Context, active []trackedSource, held []string) (finished, changed bool, err error) {
	terms, known, err := j.destinationNames(ctx)
	if err != nil {
		return false, false, err
	}
	var errs []error
	var blogs []BlogCandidates
	var live []trackedSource
	for _, entry := range active {
		blog, err := j.candidates(ctx, entry.source, terms)
		if err != nil && j.blocked(err) {
			errs = append(errs, j.markBlocked(ctx, entry, err))
			held = append(held, entry.source.Host)
			continue
		}
		if err != nil {
			j.Logf("writers sync %s: stopped, %v", entry.source.Host, err)
			return false, false, fmt.Errorf("%s: %w", entry.source.Host, err)
		}
		if CountDestinationPosts(blog, known) == 0 {
			kept, err := j.index.PostIDs(ctx, entry.source.Host)
			if err != nil {
				return false, false, fmt.Errorf("%s: %w", entry.source.Host, err)
			}
			if len(kept) > 0 {
				j.Logf("writers sync %s: warning: no destination posts matched, keeping %d indexed posts, retrying next run", entry.source.Host, len(kept))
				held = append(held, entry.source.Host)
				continue
			}
		}
		blogs, live = append(blogs, blog), append(live, entry)
	}
	reserved := 0
	for _, host := range held {
		kept, err := j.index.PostIDs(ctx, host)
		if err != nil {
			return false, false, fmt.Errorf("%s: %w", host, err)
		}
		reserved += len(kept)
	}
	picks := Select(blogs, known, max(0, MaxIndexedPosts-reserved))
	var updated []string
	for _, entry := range live {
		indexed, deleted, err := j.store(ctx, entry, picks)
		if err != nil && j.blocked(err) {
			errs = append(errs, j.markBlocked(ctx, entry, err))
			continue
		}
		if err != nil {
			j.Logf("writers sync %s: stopped, %v", entry.source.Host, err)
			errs = append(errs, fmt.Errorf("%s: %w", entry.source.Host, err))
			continue
		}
		j.Logf("writers sync %s: indexed %d posts, removed %d", entry.source.Host, indexed, deleted)
		finished, changed = true, changed || indexed > 0 || deleted > 0
		updated = append(updated, entry.source.Host)
	}
	trimmed, err := j.trimOverflow(ctx, blogs, known, picks, updated, active, held)
	errs = append(errs, err)
	return finished, changed || trimmed > 0, errors.Join(errs...)
}

// trimOverflow brings the total back to MaxIndexedPosts when it is over, for example after one
// blog's store failed and kept its old rows while the others got new picks. It deletes the
// lowest-ranked rows of the blogs updated in this run; the rows of every other blog stay.
func (j *SyncJob) trimOverflow(ctx context.Context, blogs []BlogCandidates, known []string, picks []Pick, updated []string, active []trackedSource, held []string) (int, error) {
	hosts := make(map[string]bool)
	for _, entry := range active {
		hosts[entry.source.Host] = true
	}
	for _, host := range held {
		hosts[host] = true
	}
	total := 0
	for host := range hosts {
		stored, err := j.index.PostIDs(ctx, host)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", host, err)
		}
		total += len(stored)
	}
	if total <= MaxIndexedPosts {
		return 0, nil
	}
	var candidates []Pick
	for _, host := range updated {
		stored, err := j.index.PostIDs(ctx, host)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", host, err)
		}
		inIndex := make(map[int64]bool, len(stored))
		for _, id := range stored {
			inIndex[id] = true
		}
		for _, pick := range picks {
			if pick.Host == host && inIndex[pick.PostID] {
				candidates = append(candidates, pick)
			}
		}
	}
	trimmed := 0
	for _, pick := range Trim(blogs, known, candidates, total-MaxIndexedPosts) {
		if err := j.index.DeletePost(ctx, pick.Host, pick.PostID); err != nil {
			return trimmed, fmt.Errorf("%s: %w", pick.Host, err)
		}
		trimmed++
	}
	j.Logf("writers sync: trimmed %d posts to stay within %d", trimmed, MaxIndexedPosts)
	return trimmed, nil
}

// destinationNames returns the search terms (area and cached destination names, once each) and
// every name that makes a post a destination post.
func (j *SyncJob) destinationNames(ctx context.Context) (terms, known []string, err error) {
	names := make([]string, 0, len(j.Areas))
	for _, area := range j.Areas {
		names = append(names, area.Name)
	}
	if j.Destinations != nil {
		cached, err := j.Destinations.CachedDestinations(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("list cached destinations: %w", err)
		}
		for _, area := range cached {
			names = append(names, area.Name)
		}
	}
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		if folded := foldName(name); folded != "" && !seen[folded] {
			seen[folded] = true
			terms = append(terms, name)
		}
	}
	known = append(known, terms...)
	if j.PlaceNames != nil {
		more, err := j.PlaceNames(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("list place names: %w", err)
		}
		known = append(known, more...)
	}
	return terms, known, nil
}

// candidates searches one blog for each term (first page only) and tallies its recent comments.
func (j *SyncJob) candidates(ctx context.Context, source Source, terms []string) (BlogCandidates, error) {
	blog := BlogCandidates{Host: source.Host}
	for _, term := range terms {
		hits, _, err := j.posts.SearchPosts(ctx, source.APIBase, term, 1)
		if err != nil {
			return BlogCandidates{}, err
		}
		blog.Hits = append(blog.Hits, hits...)
	}
	comments, err := j.posts.RecentCommentCounts(ctx, source.APIBase, commentPages)
	if err != nil {
		return BlogCandidates{}, err
	}
	blog.Comments = comments
	return blog, nil
}

// store indexes the source's chosen posts, deletes its indexed posts that were not chosen, and
// saves the finished state. It returns how many posts were indexed and deleted.
func (j *SyncJob) store(ctx context.Context, entry trackedSource, picks []Pick) (indexed, deleted int, err error) {
	source := entry.source
	var ids []int64
	chosen := make(map[int64]bool)
	for _, pick := range picks {
		if pick.Host == source.Host {
			ids = append(ids, pick.PostID)
			chosen[pick.PostID] = true
		}
	}
	if len(ids) > 0 {
		posts, err := j.posts.PostsByID(ctx, source.APIBase, ids)
		if err != nil {
			return 0, 0, err
		}
		for _, post := range posts {
			if err := j.index.UpsertPost(ctx, IndexedPost{
				Host: source.Host, PostID: post.ID, URL: post.Link, Title: post.Title,
				Language: source.Language, Modified: post.Modified,
				Extracted: ExtractPost(post.Title, post.Content),
			}); err != nil {
				return indexed, 0, err
			}
			chosen[post.ID] = true
			indexed++
		}
	}
	stored, err := j.index.PostIDs(ctx, source.Host)
	if err != nil {
		return indexed, 0, err
	}
	for _, id := range stored {
		if chosen[id] {
			continue
		}
		if err := j.index.DeletePost(ctx, source.Host, id); err != nil {
			return indexed, deleted, err
		}
		deleted++
	}
	state := entry.state
	state.Status, state.LastFull = StatusActive, j.Now()
	return indexed, deleted, j.index.SaveSyncState(ctx, state)
}

// markBlocked records that a source refused us, so it is not requested again.
func (j *SyncJob) markBlocked(ctx context.Context, entry trackedSource, cause error) error {
	j.Logf("writers sync %s: blocked, not requested again: %v", entry.source.Host, cause)
	state := entry.state
	state.Status = StatusBlocked
	return j.index.SaveSyncState(ctx, state)
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
