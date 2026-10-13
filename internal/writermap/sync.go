package writermap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
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

// TopUpsPerDay is the most destination top-ups started per UTC day; the rest wait for the next.
const TopUpsPerDay = 20

// perDestinationSince is when each destination got its own PostsPerDestination. A source whose
// last run is older picked from the shared 100-post cap, so it is due at once, without waiting for
// the monthly window. It must stay before any run of this code, or that run would be due again.
var perDestinationSince = time.Date(2026, 10, 13, 20, 0, 0, 0, time.UTC)

// busyRetry is how long Schedule waits when a top-up holds the lock, before trying the run again.
const busyRetry = time.Minute

// ErrSyncRunning means another run holds the single-run lock.
var ErrSyncRunning = errors.New("writers sync already running")

// SyncJob keeps the writers index at no more than MaxPosts posts, picked afresh on every run:
// for each active blog it searches the destination names, keeps the hits whose title or slug
// names a known place, and ranks them by recent comments. Each destination gets up to
// PostsPerDestination posts, shared across blogs by Select; the index holds their union.
// Only the chosen posts are fetched and indexed; indexed posts that no destination chose are
// deleted. Rows of a blocked blog, or of a blog with no destination candidates left, stay and
// count inside MaxPosts; a union over what is left loses its lowest-ranked posts. If the total is
// still over MaxPosts after the run (a blog's store failed and kept its old rows), the
// lowest-ranked rows of the blogs updated in this run are deleted until it is exactly MaxPosts.
// Between runs, TopUp picks posts for a destination cached for the first time.
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
	// MaxPosts is the ceiling on indexed posts across all destinations and blogs (MaxIndexedPosts).
	MaxPosts int
	// RecomputeDestination replaces a destination's stored writers' map result after its top-up
	// indexed posts; nil skips it.
	RecomputeDestination func(ctx context.Context, area Area) error

	Now   func() time.Time
	Sleep func(ctx context.Context, d time.Duration) error
	Logf  func(format string, args ...any)

	running sync.Mutex
	// Guarded by running: the last monthly run's comment counts per blog, and the search hits
	// of the indexed posts, which rank them when a top-up trims.
	tallies map[string]map[int64]int
	seen    []BlogCandidates

	queueMu sync.Mutex
	queue   []Area
	queued  map[string]bool // every destination ever queued, so each is topped up once
	wake    chan struct{}
}

// NewSyncJob returns a job with the real clock and the standard logger.
func NewSyncJob(sources []Source, index SyncIndex, posts PostSource) *SyncJob {
	return &SyncJob{
		sources: sources,
		index:   index,
		posts:   posts,

		MaxPosts: MaxIndexedPosts,
		Now:      time.Now,
		queued:   make(map[string]bool),
		wake:     make(chan struct{}, 1),
		Sleep:    sleepContext,
		Logf:     log.Printf,
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
// While a top-up holds the lock, it tries again a minute later.
func (j *SyncJob) Schedule(ctx context.Context) {
	for {
		err := j.Run(ctx)
		if errors.Is(err, ErrSyncRunning) {
			if err := j.Sleep(ctx, busyRetry); err != nil {
				return
			}
			continue
		}
		if err != nil {
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
// run that failed before saving), one last run before per-destination picks, or, in the off-peak
// window, one whose last run was in an earlier month. One due source re-picks every blog.
func (j *SyncJob) due(active []trackedSource) bool {
	offPeak := inOffPeak(j.Now())
	for _, entry := range active {
		if entry.state.LastFull.Before(perDestinationSince) || offPeak && monthlyDue(entry.state, j.Now()) {
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
	byTerm := make([][]BlogCandidates, len(terms))
	var live []trackedSource
	for _, entry := range active {
		perTerm, err := j.candidates(ctx, entry.source, terms)
		if err != nil && j.blocked(err) {
			errs = append(errs, j.markBlocked(ctx, entry, err))
			held = append(held, entry.source.Host)
			continue
		}
		if err != nil {
			j.Logf("writers sync %s: stopped, %v", entry.source.Host, err)
			return false, false, fmt.Errorf("%s: %w", entry.source.Host, err)
		}
		blog := BlogCandidates{Host: entry.source.Host}
		if merged := MergeBlogs(perTerm); len(merged) > 0 {
			blog = merged[0]
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
		for i := range perTerm {
			byTerm[i] = append(byTerm[i], perTerm[i])
		}
	}
	destinations := make([]DestinationCandidates, len(terms))
	for i, term := range terms {
		destinations[i] = DestinationCandidates{Name: term, Blogs: byTerm[i]}
	}
	j.tallies, j.seen = make(map[string]map[int64]int, len(blogs)), blogs
	for _, blog := range blogs {
		j.tallies[blog.Host] = blog.Comments
	}
	reserved := 0
	for _, host := range held {
		kept, err := j.index.PostIDs(ctx, host)
		if err != nil {
			return false, false, fmt.Errorf("%s: %w", host, err)
		}
		reserved += len(kept)
	}
	picks := SelectPerDestination(destinations, known, PostsPerDestination, max(0, j.MaxPosts-reserved))
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
	if total <= j.MaxPosts {
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
	for _, pick := range Trim(blogs, known, candidates, total-j.MaxPosts) {
		if err := j.index.DeletePost(ctx, pick.Host, pick.PostID); err != nil {
			return trimmed, fmt.Errorf("%s: %w", pick.Host, err)
		}
		trimmed++
	}
	j.Logf("writers sync: trimmed %d posts to stay within %d", trimmed, j.MaxPosts)
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
// It returns the blog's hits per term, in term order, all with the same comment counts.
func (j *SyncJob) candidates(ctx context.Context, source Source, terms []string) ([]BlogCandidates, error) {
	perTerm := make([]BlogCandidates, len(terms))
	for i, term := range terms {
		hits, _, err := j.posts.SearchPosts(ctx, source.APIBase, term, 1)
		if err != nil {
			return nil, err
		}
		perTerm[i] = BlogCandidates{Host: source.Host, Hits: hits}
	}
	comments, err := j.posts.RecentCommentCounts(ctx, source.APIBase, commentPages)
	if err != nil {
		return nil, err
	}
	for i := range perTerm {
		perTerm[i].Comments = comments
	}
	return perTerm, nil
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
	indexed, err = j.indexPosts(ctx, source, ids)
	if err != nil {
		return indexed, 0, err
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

// indexPosts fetches the posts and indexes their extracted names; the text is not kept. It returns
// how many were indexed.
func (j *SyncJob) indexPosts(ctx context.Context, source Source, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	posts, err := j.posts.PostsByID(ctx, source.APIBase, ids)
	if err != nil {
		return 0, err
	}
	indexed := 0
	for _, post := range posts {
		if err := j.index.UpsertPost(ctx, IndexedPost{
			Host: source.Host, PostID: post.ID, URL: post.Link, Title: post.Title,
			Language: source.Language, Modified: post.Modified,
			Extracted: ExtractPost(post.Title, post.Content),
		}); err != nil {
			return indexed, err
		}
		indexed++
	}
	return indexed, nil
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

// QueueTopUp asks for a top-up of a destination cached for the first time. It never blocks. Each
// destination (by name and country) is queued once per process, however often it is asked for.
func (j *SyncJob) QueueTopUp(area Area) {
	key := foldName(area.Name) + "\x00" + strings.ToLower(area.Country)
	j.queueMu.Lock()
	if j.queued[key] {
		j.queueMu.Unlock()
		return
	}
	j.queued[key] = true
	j.queue = append(j.queue, area)
	j.queueMu.Unlock()
	select {
	case j.wake <- struct{}{}:
	default:
	}
}

// WorkTopUps runs the queued top-ups one at a time until ctx ends, at most TopUpsPerDay per UTC
// day; once the day's budget is spent, the queue waits for the next day.
func (j *SyncJob) WorkTopUps(ctx context.Context) {
	var day time.Time
	started := 0
	for {
		if today := j.Now().UTC().Truncate(24 * time.Hour); !today.Equal(day) {
			day, started = today, 0
		}
		if started >= TopUpsPerDay {
			if err := j.Sleep(ctx, day.Add(24*time.Hour).Sub(j.Now())); err != nil {
				return
			}
			continue
		}
		area, ok := j.nextTopUp()
		if !ok {
			select {
			case <-ctx.Done():
				return
			case <-j.wake:
			}
			continue
		}
		started++
		if err := j.TopUp(ctx, area); err != nil {
			j.Logf("writers top-up %s: %v", area.Name, err)
		}
	}
}

func (j *SyncJob) nextTopUp() (Area, bool) {
	j.queueMu.Lock()
	defer j.queueMu.Unlock()
	if len(j.queue) == 0 {
		return Area{}, false
	}
	area := j.queue[0]
	j.queue = j.queue[1:]
	return area, true
}

// TopUp picks posts for one destination between monthly runs. It searches every active blog for
// the destination's name, ranks the hits by the last monthly run's comment counts (newest first
// when there are none yet), and indexes the up to PostsPerDestination picks not indexed yet. A
// blog whose search fails is skipped. Over MaxPosts, the lowest-ranked posts of the active blogs,
// whatever their destination, are deleted. If posts were added, the destination's stored result
// is recomputed. It waits for a running sync or top-up.
func (j *SyncJob) TopUp(ctx context.Context, area Area) error {
	j.running.Lock()
	defer j.running.Unlock()

	active, _, errs := j.activeSources(ctx)
	_, known, err := j.destinationNames(ctx)
	if err != nil {
		return errors.Join(append(errs, err)...)
	}
	known = append(known, area.Name)
	var blogs []BlogCandidates
	var live []trackedSource
	for _, entry := range active {
		hits, _, err := j.posts.SearchPosts(ctx, entry.source.APIBase, area.Name, 1)
		if err != nil && j.blocked(err) {
			errs = append(errs, j.markBlocked(ctx, entry, err))
			continue
		}
		if err != nil {
			j.Logf("writers top-up %s %s: skipped, %v", area.Name, entry.source.Host, err)
			errs = append(errs, fmt.Errorf("%s: %w", entry.source.Host, err))
			continue
		}
		blogs = append(blogs, BlogCandidates{Host: entry.source.Host, Hits: hits, Comments: j.tallies[entry.source.Host]})
		live = append(live, entry)
	}
	picks := Select(blogs, known, PostsPerDestination)
	j.seen = MergeBlogs(append(j.seen, blogs...))
	added := 0
	for _, entry := range live {
		stored, err := j.index.PostIDs(ctx, entry.source.Host)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", entry.source.Host, err))
			continue
		}
		indexed := make(map[int64]bool, len(stored))
		for _, id := range stored {
			indexed[id] = true
		}
		var ids []int64
		for _, pick := range picks {
			if pick.Host == entry.source.Host && !indexed[pick.PostID] {
				ids = append(ids, pick.PostID)
			}
		}
		n, err := j.indexPosts(ctx, entry.source, ids)
		added += n
		if err != nil && j.blocked(err) {
			errs = append(errs, j.markBlocked(ctx, entry, err))
			continue
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", entry.source.Host, err))
		}
	}
	trimmed, err := j.trimToCeiling(ctx, live, known)
	errs = append(errs, err)
	j.Logf("writers top-up %s: indexed %d posts, trimmed %d", area.Name, added, trimmed)
	if added > 0 && j.RecomputeDestination != nil {
		errs = append(errs, j.RecomputeDestination(ctx, area))
	}
	return errors.Join(errs...)
}

// trimToCeiling deletes the lowest-ranked indexed posts of the given blogs, across all
// destinations, while the index holds more than MaxPosts. Rows of every other blog count but
// stay. Posts never seen in a search this process rank lowest. It logs how many each blog lost.
func (j *SyncJob) trimToCeiling(ctx context.Context, live []trackedSource, known []string) (int, error) {
	total := 0
	var candidates []Pick
	isLive := make(map[string]bool, len(live))
	for _, entry := range live {
		isLive[entry.source.Host] = true
	}
	for _, source := range j.sources {
		if source.Status == StatusRemoved {
			continue
		}
		stored, err := j.index.PostIDs(ctx, source.Host)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", source.Host, err)
		}
		total += len(stored)
		if isLive[source.Host] {
			for _, id := range stored {
				candidates = append(candidates, Pick{Host: source.Host, PostID: id})
			}
		}
	}
	if total <= j.MaxPosts {
		return 0, nil
	}
	lost := make(map[string]int)
	trimmed := 0
	for _, pick := range Trim(j.seen, known, candidates, total-j.MaxPosts) {
		if err := j.index.DeletePost(ctx, pick.Host, pick.PostID); err != nil {
			return trimmed, fmt.Errorf("%s: %w", pick.Host, err)
		}
		lost[pick.Host]++
		trimmed++
	}
	hosts := make([]string, 0, len(lost))
	for host, n := range lost {
		hosts = append(hosts, fmt.Sprintf("%s %d", host, n))
	}
	sort.Strings(hosts)
	j.Logf("writers top-up: trimmed %d posts to stay within %d (%s)", trimmed, j.MaxPosts, strings.Join(hosts, ", "))
	return trimmed, nil
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
