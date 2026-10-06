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

// Summarizer makes an English summary from transient, attributed passages for one place.
type Summarizer interface {
	Summarize(ctx context.Context, place writermap.Place, passages []writermap.Passage) (string, error)
}
