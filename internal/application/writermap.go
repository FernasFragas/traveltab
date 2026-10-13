package application

import (
	"context"

	"weatherservice/internal/writermap"
)

// WriterMapSource serves already indexed destination data. Implementations must not contact
// writer sites while either method is serving a request.
type WriterMapSource interface {
	Destination(ctx context.Context, dest writermap.Area) (*writermap.DestinationResult, error)
	Place(ctx context.Context, dest writermap.Area, qid string) (*writermap.Place, *writermap.Summary, error)
}

// ProgressiveWriterMapSource also serves a destination that is still loading: the pins found so
// far with no writer matches, and loading true until the last ring is in.
type ProgressiveWriterMapSource interface {
	WriterMapSource
	DestinationProgress(ctx context.Context, dest writermap.Area) (result *writermap.DestinationResult, loading bool, err error)
}

// BasePlaceSource returns capped map pins and the nearby gazetteer, loaded in 5 km rings.
// When they cannot be fetched and nothing is cached, it returns an empty result with an error.
type BasePlaceSource interface {
	Places(ctx context.Context, dest writermap.Area) (writermap.BasePlaces, error)
}

// BasePlaceRefresher re-runs a destination's base query and replaces its cached places.
// The cache decorator implements it for the monthly job.
type BasePlaceRefresher interface {
	Refresh(ctx context.Context, dest writermap.Area) error
}

// Summarizer makes an English summary from transient, attributed passages for one place.
type Summarizer interface {
	Summarize(ctx context.Context, place writermap.Place, passages []writermap.Passage) (string, error)
}
