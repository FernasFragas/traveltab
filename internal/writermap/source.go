package writermap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"weatherservice/internal/planner"
)

// MapSource serves destinations from cached base places, the writers index and the matcher.
// Each destination's result is stored with no expiry and replaced by Recompute after the
// monthly sync. It reads only local stores and the base place source, never a writer site.
type MapSource struct {
	base      BaseSource
	index     MatchIndex
	store     DestinationStore
	states    SyncStates
	sources   []Source
	overrides Overrides

	// Summaries reads stored place summaries; nil means none exist yet (until PR 9).
	Summaries SummaryReader
}

// NewMapSource composes the real writers' map source.
func NewMapSource(base BaseSource, index MatchIndex, store DestinationStore, states SyncStates, sources []Source, overrides Overrides) *MapSource {
	return &MapSource{base: base, index: index, store: store, states: states, sources: sources, overrides: overrides}
}

// BaseSource returns a destination's base pins and matching gazetteer.
type BaseSource interface {
	Places(ctx context.Context, dest Area) (BasePlaces, error)
}

// PartialBaseSource is a BaseSource that loads in the background and can report what it has.
type PartialBaseSource interface {
	BaseSource
	// Partial starts the destination's load if none is running, then returns the places loaded so
	// far; Done is true once the result is complete. After a failed load it returns what was loaded
	// with the error, and retries on a later call.
	Partial(ctx context.Context, dest Area) (BaseProgress, error)
}

// PagedBaseSource loads a destination's base places a page at a time, nearest ring first, so the
// pins found so far can be shown and a failed load can resume where it stopped.
type PagedBaseSource interface {
	// PlacesFrom continues from the given progress (zero value starts fresh) and calls onPage
	// with the progress after each stored page. On failure it returns the progress so far and
	// the error.
	PlacesFrom(ctx context.Context, dest Area, from BaseProgress, onPage func(BaseProgress)) (BaseProgress, error)
}

// BaseProgress is a partly or fully loaded set of base places plus where the load stopped.
type BaseProgress struct {
	BasePlaces
	Ring   int  `json:"ring"`   // index of the next 5 km ring to load
	Offset int  `json:"offset"` // items already loaded from that ring
	Done   bool `json:"done"`
	// Mode is how the destination loads, decided once from its density after the first page:
	// "normal" (5 km rings out to the radius) or "dense" (eight 1 km rings). Empty until decided.
	Mode string `json:"mode,omitempty"`
	// Unnamed is the current page as the page query returned it (coordinates, sitelinks, types, no
	// names) while its label query is pending; visitable ones are already in Pins. PageRows is the
	// page's row count, which moves Offset once the labels are in. A resume asks only for the labels.
	Unnamed  []Place `json:"unnamed,omitempty"`
	PageRows int     `json:"page_rows,omitempty"`
}

// MatchIndex returns indexed occurrences of folded phrases, with each post's day sections.
type MatchIndex interface {
	WriterMatchRows(ctx context.Context, phrases []string) ([]MatchRow, error)
}

// DestinationStore keeps one result per destination, upserted in place.
type DestinationStore interface {
	WriterDestination(ctx context.Context, dest Area) (DestinationResult, bool, error)
	SaveWriterDestination(ctx context.Context, dest Area, result DestinationResult) error
	WriterDestinations(ctx context.Context) ([]Area, error)
}

// SyncStates reads each source's latest sync progress.
type SyncStates interface {
	SyncState(ctx context.Context, host string) (SyncState, error)
}

// SummaryReader returns a place's stored summary, or nil when it has none.
type SummaryReader interface {
	Summary(ctx context.Context, qid string) (*Summary, error)
}

// Overrides is writerdata/overrides.json: the match rules plus named areas, such as islands,
// whose names also count as destination context for any destination inside them.
type Overrides struct {
	MatchOverrides
	Areas []Area `json:"areas"`
}

// ParseOverrides reads the reviewed overrides file.
func ParseOverrides(data []byte) (Overrides, error) {
	var overrides Overrides
	if err := json.Unmarshal(data, &overrides); err != nil {
		return Overrides{}, fmt.Errorf("parse writer overrides: %w", err)
	}
	return overrides, nil
}

// Destination returns the stored result, computing and storing it on the first request.
// A base place failure is returned and nothing is stored, so the next request tries again.
func (s *MapSource) Destination(ctx context.Context, dest Area) (*DestinationResult, error) {
	result, ok, err := s.store.WriterDestination(ctx, dest)
	if err != nil {
		return nil, err
	}
	if ok {
		return &result, nil
	}
	return s.recompute(ctx, dest)
}

// DestinationProgress is Destination for a map that can show pins while base places load. It
// returns the stored result, or, while a background load is running, the pins found so far with
// no writer matches and loading true. Matching runs once, when the load is done. A load that
// failed after finding pins serves them with loading false; the next visit resumes it.
func (s *MapSource) DestinationProgress(ctx context.Context, dest Area) (*DestinationResult, bool, error) {
	result, ok, err := s.store.WriterDestination(ctx, dest)
	if err != nil {
		return nil, false, err
	}
	if ok {
		return &result, false, nil
	}
	partial, ok := s.base.(PartialBaseSource)
	if !ok {
		result, err := s.recompute(ctx, dest)
		return result, false, err
	}
	progress, loadErr := partial.Partial(ctx, dest)
	if loadErr == nil && progress.Done {
		result, err := s.recompute(ctx, dest)
		return result, false, err
	}
	if loadErr != nil && len(progress.Pins) == 0 {
		return nil, false, loadErr
	}
	return &DestinationResult{
		Places: progress.Pins, WriterCounts: map[string]int{},
		Itineraries: []Itinerary{}, SourceStatuses: []SourceStatus{},
	}, loadErr == nil, nil
}

// Place returns one pin of the destination and its summary, if one is stored.
func (s *MapSource) Place(ctx context.Context, dest Area, qid string) (*Place, *Summary, error) {
	result, _, err := s.DestinationProgress(ctx, dest)
	if err != nil {
		return nil, nil, err
	}
	for _, place := range result.Places {
		if place.QID != qid {
			continue
		}
		if s.Summaries == nil {
			return &place, nil, nil
		}
		summary, err := s.Summaries.Summary(ctx, qid)
		if err != nil {
			return nil, nil, err
		}
		return &place, summary, nil
	}
	return nil, nil, fmt.Errorf("%w: %s", ErrPlaceNotFound, qid)
}

// Recompute replaces every stored destination from the current index. The monthly sync calls
// it after refreshing base places. A failed destination keeps its previous result.
func (s *MapSource) Recompute(ctx context.Context) error {
	areas, err := s.store.WriterDestinations(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, area := range areas {
		if _, err := s.recompute(ctx, area); err != nil {
			errs = append(errs, fmt.Errorf("recompute %s: %w", area.Name, err))
		}
	}
	return errors.Join(errs...)
}

// RecomputeDestination replaces one destination's stored result from the current index. A top-up
// calls it after indexing the destination's posts.
func (s *MapSource) RecomputeDestination(ctx context.Context, dest Area) error {
	_, err := s.recompute(ctx, dest)
	return err
}

func (s *MapSource) recompute(ctx context.Context, dest Area) (*DestinationResult, error) {
	base, err := s.base.Places(ctx, dest)
	if err != nil {
		return nil, err
	}
	names := s.contextNames(dest)
	rows, err := s.index.WriterMatchRows(ctx, MatchLookupPhrases(base.Gazetteer, names, s.overrides.MatchOverrides))
	if err != nil {
		return nil, err
	}
	result := MatchDestination(base, rows, names, s.sources, s.overrides.MatchOverrides)
	result.SourceStatuses, result.Complete, err = s.sourceStatuses(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.store.SaveWriterDestination(ctx, dest, result); err != nil {
		return nil, err
	}
	return &result, nil
}

// contextNames are the destination's own name plus every override area that contains it.
func (s *MapSource) contextNames(dest Area) []string {
	names := []string{dest.Name}
	point := planner.Place{Lat: dest.Lat, Lon: dest.Lon}
	for _, area := range s.overrides.Areas {
		if planner.DistanceKM(point, planner.Place{Lat: area.Lat, Lon: area.Lon}) <= area.RadiusKM {
			names = append(names, area.Name)
		}
	}
	return names
}

// sourceStatuses reports each allowlisted source. The result is complete when every active
// source has finished its latest sync.
func (s *MapSource) sourceStatuses(ctx context.Context) ([]SourceStatus, bool, error) {
	statuses := make([]SourceStatus, 0, len(s.sources))
	complete := true
	for _, source := range s.sources {
		status := SourceStatus{Host: source.Host, Status: source.Status}
		if source.Status == StatusActive {
			state, err := s.states.SyncState(ctx, source.Host)
			if err != nil {
				return nil, false, err
			}
			if state.Status != "" {
				status.Status = state.Status
			}
			status.CheckedAt = latest(state.LastFull, state.LastDelta)
			complete = complete && state.Status == StatusActive
		}
		statuses = append(statuses, status)
	}
	return statuses, complete, nil
}

func latest(a, b time.Time) *time.Time {
	if b.After(a) {
		a = b
	}
	if a.IsZero() {
		return nil
	}
	return &a
}
