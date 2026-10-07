package writermaptest

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"weatherservice/internal/application"
	"weatherservice/internal/writermap"
)

// The fakes must keep matching the ports they stand in for.
var (
	_ application.WriterMapSource    = (*FakeSource)(nil)
	_ application.BasePlaceSource    = (*FakeBaseSource)(nil)
	_ application.BasePlaceRefresher = (*FakeBaseSource)(nil)
	_ application.Summarizer         = (*FakeSummarizer)(nil)
)

var ErrFakeUnavailable = errors.New("fake writer map source unavailable")

// FakeSource serves deterministic data for previews and offline tests.
type FakeSource struct {
	Result      writermap.DestinationResult
	Summaries   map[string]writermap.Summary
	Delay       time.Duration
	Partial     bool
	Unavailable bool
	Empty       bool
}

func (f *FakeSource) Destination(ctx context.Context, _ writermap.Area) (*writermap.DestinationResult, error) {
	if err := f.wait(ctx); err != nil {
		return nil, err
	}
	if f.Unavailable {
		return nil, ErrFakeUnavailable
	}
	if f.Empty {
		return &writermap.DestinationResult{
			Places:         []writermap.Place{},
			WriterCounts:   map[string]int{},
			Itineraries:    []writermap.Itinerary{},
			SourceStatuses: f.Result.SourceStatuses,
			Complete:       true,
		}, nil
	}
	result := cloneResult(f.Result)
	if f.Partial {
		result.Complete = false
	}
	return &result, nil
}

func (f *FakeSource) Place(ctx context.Context, _ writermap.Area, qid string) (*writermap.Place, *writermap.Summary, error) {
	if err := f.wait(ctx); err != nil {
		return nil, nil, err
	}
	if f.Unavailable {
		return nil, nil, ErrFakeUnavailable
	}
	if f.Empty {
		return nil, nil, writermap.ErrPlaceNotFound
	}
	for _, place := range f.Result.Places {
		if place.QID == qid {
			placeCopy := place
			placeCopy.Names = append([]string(nil), place.Names...)
			if summary, ok := f.Summaries[qid]; ok {
				summaryCopy := summary
				return &placeCopy, &summaryCopy, nil
			}
			return &placeCopy, nil, nil
		}
	}
	return nil, nil, fmt.Errorf("%w: %s", writermap.ErrPlaceNotFound, qid)
}

// FakeBaseSource returns configured base places or an error, and counts calls so tests can
// check deduplication and warming. It is safe for concurrent use.
type FakeBaseSource struct {
	Result       []writermap.Place
	Err          error
	mu           sync.Mutex
	placesCalls  int
	refreshCalls int
}

func (f *FakeBaseSource) Places(_ context.Context, _ writermap.Area) ([]writermap.Place, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.placesCalls++
	if f.Err != nil {
		return []writermap.Place{}, f.Err
	}
	return clonePlaces(f.Result), nil
}

func (f *FakeBaseSource) Refresh(_ context.Context, _ writermap.Area) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refreshCalls++
	return f.Err
}

func (f *FakeBaseSource) PlacesCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.placesCalls
}

func (f *FakeBaseSource) RefreshCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.refreshCalls
}

// FakeSummarizer returns configured prose or an error without retaining input passages.
type FakeSummarizer struct {
	Text  string
	Err   error
	mu    sync.Mutex
	Calls int
}

func (f *FakeSummarizer) Summarize(_ context.Context, _ writermap.Place, _ []writermap.Passage) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls++
	return f.Text, f.Err
}

func (f *FakeSource) wait(ctx context.Context) error {
	if f.Delay <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(f.Delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// cloneResult copies the places and their names, so callers can edit them. The other fields
// are shared with the fake and must be treated as read-only.
func cloneResult(in writermap.DestinationResult) writermap.DestinationResult {
	out := in
	out.Places = clonePlaces(in.Places)
	return out
}

// clonePlaces copies places and their names, so callers can edit them.
func clonePlaces(in []writermap.Place) []writermap.Place {
	out := append([]writermap.Place(nil), in...)
	for i := range out {
		out[i].Names = append([]string(nil), in[i].Names...)
	}
	return out
}
