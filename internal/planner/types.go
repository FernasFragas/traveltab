package planner

import (
	"context"
	"time"
)

// Kind says how a place behaves in the rain.
type Kind string

const (
	Indoor  Kind = "indoor"
	Outdoor Kind = "outdoor"
	Mixed   Kind = "mixed"
	Deny    Kind = "deny" // only used in type lists: a place with this type is never kept
)

// Numbers from the one-pager, in one place so they're easy to tune.
const (
	SearchRadiusKM    = 10
	StayRadiusM       = 1000
	RainyDayMM        = 5.0
	ForecastTrustDays = 7
	ForecastMaxDays   = 16
	MaxDays           = 5
	MinStopsPerDay    = 3
	MaxStopsPerDay    = 4
	SunnyPoolSize     = 20
	CandidatePoolSize = 60
	MaxStays          = 6
)

type Place struct {
	ID        string // Wikidata ID, e.g. "Q193386"
	Name      string
	Lat, Lon  float64
	Sitelinks int      // how many Wikimedia sites have a page about it
	Types     []string // Wikidata "instance of" (P31) IDs
	Image     string   // Wikimedia Commons file name, "" when there is none
	Kind      Kind     // set by Rank; empty in source results
}

type DayForecast struct {
	Date   time.Time // midnight in the city's time zone
	RainMM *float64  // nil when Open-Meteo has no value
}

type Forecast struct {
	Timezone string // IANA name, e.g. "Europe/Lisbon"
	Days     []DayForecast
}

type Stay struct {
	Name     string // English name when OpenStreetMap has one
	Kind     string // "hotel", "hostel" or "guest_house"
	Lat, Lon float64
	Website  string
	Stars    int // 0 when unknown
}

// Group is a set of stops close to each other, in walking order.
type Group struct {
	Stops  []Place
	WalkKM float64
}

type Day struct {
	Date     time.Time
	Stops    []Place
	WalkKM   float64
	Rainy    bool
	Forecast *DayForecast // nil when the date is past the forecast window
	Certain  bool         // true when the date is within ForecastTrustDays of today
}

type Request struct {
	City, Country string
	Lat, Lon      float64
	Start         time.Time
	Days          int // 1..MaxDays
}

type Plan struct {
	Request    Request
	Days       []Day
	Note       string // small-town or forecast message, "" when none
	Center     Place
	Stays      []Stay
	StaysNote  string // set when places to stay couldn't be loaded
	BookingURL string
}

type PlaceSource interface {
	// PlacesNear returns every candidate within SearchRadiusKM, unfiltered and with Kind empty.
	PlacesNear(ctx context.Context, lat, lon float64) ([]Place, error)
}

type ForecastSource interface {
	DailyForecast(ctx context.Context, lat, lon float64) (*Forecast, error)
}

type StaySource interface {
	StaysNear(ctx context.Context, lat, lon float64) ([]Stay, error)
}
