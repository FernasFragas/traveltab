// Package writermap defines the stable domain shapes shared by the writers' map.
package writermap

import (
	"errors"
	"time"
)

// ErrPlaceNotFound means the destination has no place with the requested QID.
var ErrPlaceNotFound = errors.New("place not found")

// Area is a resolved destination and the radius used to find nearby places.
// A non-positive RadiusKM lets the source use its normal city radius.
type Area struct {
	Name     string  `json:"name"`
	Country  string  `json:"country,omitempty"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	RadiusKM float64 `json:"radius_km,omitempty"`
}

// DestinationResult is the complete map payload for a destination.
type DestinationResult struct {
	Places         []Place              `json:"places"`
	Mentions       map[string][]Mention `json:"mentions,omitempty"`
	WriterCounts   map[string]int       `json:"writer_counts"`
	Itineraries    []Itinerary          `json:"itineraries"`
	SourceStatuses []SourceStatus       `json:"source_statuses"`
	Complete       bool                 `json:"complete"`
}

// Place is a Wikidata place with the names and metadata used by matching and panels.
type Place struct {
	QID         string   `json:"qid"`
	Names       []string `json:"names"`
	Lat         float64  `json:"lat"`
	Lon         float64  `json:"lon"`
	Kind        string   `json:"kind,omitempty"`
	Description string   `json:"description,omitempty"`
	Photo       string   `json:"photo,omitempty"`
	PhotoCredit string   `json:"photo_credit,omitempty"`
	PhotoURL    string   `json:"photo_url,omitempty"`
	Sitelinks   int      `json:"sitelinks,omitempty"`
}

// Mention records that a writer's post mentions a place. Post content is never included.
type Mention struct {
	WriterHost string     `json:"writer_host"`
	PostRef    PostRef    `json:"post_ref"`
	PostURL    string     `json:"post_url"`
	PostTitle  string     `json:"post_title"`
	Language   string     `json:"language"`
	Positions  []Position `json:"positions"`
}

// PostRef is a stable source reference, normally host plus WordPress post ID.
type PostRef string

// Position identifies a matched span in the plain-text title and body stream.
type Position struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// Itinerary is an ordered writer route. Days contain Wikidata QIDs in first-mention order.
type Itinerary struct {
	WriterHost string     `json:"writer_host"`
	PostRef    PostRef    `json:"post_ref"`
	PostURL    string     `json:"post_url"`
	PostTitle  string     `json:"post_title"`
	Language   string     `json:"language"`
	Days       [][]string `json:"days"`
}

// SourceStatus reports the latest known state of one allowlisted writer source.
type SourceStatus struct {
	Host      string    `json:"host"`
	Status    string    `json:"status"`
	CheckedAt time.Time `json:"checked_at,omitempty"`
}

// Summary contains the only blog-derived prose retained by the writers' map.
type Summary struct {
	QID            string    `json:"qid"`
	EnglishText    string    `json:"english_text"`
	SourcePostRefs []PostRef `json:"source_post_refs"`
	Model          string    `json:"model"`
	GeneratedAt    time.Time `json:"generated_at"`
}

// Passage is transient text passed to the summarizer and must not be persisted or logged.
type Passage struct {
	Language string
	PostRef  PostRef
	Text     string
}
