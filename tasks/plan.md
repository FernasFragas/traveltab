# Implementation Plan: Weather-aware trip planner (MVP)

_Written 2026-09-19 from [the one-pager](../docs/ideas/weather-aware-trip-planner.md) and [its validation](../docs/ideas/weather-aware-trip-planner-validation.md). Tasks are in [todo.md](todo.md)._

## Overview

A "Plan my trip" card under the weather. You pick a start date and 1–5 days, and get a day-by-day plan of famous places: museums on rainy days, viewpoints on dry ones, plus the best area to stay. Everything comes from free, keyless sources: Wikipedia and Wikidata, Open-Meteo, and OpenStreetMap (Overpass).

**Built so agents can work in parallel:** one small task defines the shared types and interfaces. After that, up to 10 tasks run at the same time, each owning its own files.

**Built test-first:** every task lists its failing tests before any code. See [How every task is built (TDD)](#how-every-task-is-built-tdd).

**Not in this plan:** v2 (shareable links, export, AI intros). It gets its own breakdown after the MVP ships.

## Architecture Decisions

- **New package `planner/`** (import path `weatherservice/planner`) for the types, interfaces and the plain-Go planning logic. It never imports the root package or `api/`, so it has no cycles and its tests need no network or SQLite.
- **API clients live in `api/`, one file per source**, and implement the `planner` interfaces. They follow the existing client pattern (`client *http.Client` field, `New…` constructor), and their tests stub HTTP with the `roundTripFunc` type from `api/openweather_test.go`.
- **The cache lives in the root package** (`sourcecache.go`), because `database.go` owns the global `db`. It wraps any source and keeps serving old data when a refresh fails.
- **Planning is a chain of plain functions**: `Rank → GroupDays → Schedule → Medoid → PickStays`, joined by `Build`. Each step is one task, and time is passed in (`today`, `now`), so tests are deterministic.
- **The root package never imports `api/`**, because `api/` already imports the root package (for `DataToReport` and friends). `cmd/web/main.go` is the only place that wires API clients, cache and server together.
- **The UI talks to an interface** (`TripPlanner`), not to `planner` directly. So the card can be built against a fake before the real planner exists.
- **Old code goes first.** Removing the Google Places, Foursquare and itinerary code before the UI work means the parallel tasks never fight over `server.go`.

## Shared contract (Task 1)

Every other task codes against this. **Changing it needs the human's OK**, because parallel tasks depend on it.

```go
package planner // planner/types.go

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
	ID        string   // Wikidata ID, e.g. "Q193386"
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
```

```go
// planner/geo.go (Task 1)
func DistanceKM(a, b Place) float64
func WalkingLoop(stops []Place) Group     // starts at the most famous stop, nearest-neighbour order, back to the start
func Medoid(places []Place) Place         // the place with the smallest total distance to the others
func (p Place) ImageURL(width int) string // Commons Special:FilePath thumbnail, "" when there's no image
func (p Place) ImagePageURL() string      // the Commons file page, for photo credits
```

```go
// api/useragent.go (Task 1)
const UserAgent = "TravelTab/1.0 (https://traveltabbypatronfragas.com; https://github.com/FernasFragas/traveltab)"
```

```go
// planner/placetypes_gen.go (Task 1 creates an empty placeholder, Task 13 generates the real list)
var PlaceTypes = map[string]Kind{}
```

**Signatures later tasks fill in** (fixed here so tasks can call each other before they're merged):

| Function or value | File | Task |
|---|---|---|
| `func Rank(candidates []Place, types, boosts map[string]Kind) []Place` | `planner/rank.go` | 7 |
| `var Boosts map[string]Kind` | `planner/boosts.go` | 7 |
| `func GroupDays(ranked []Place, days int) []Group` | `planner/days.go` | 8 |
| `func Schedule(groups []Group, pool []Place, fc *Forecast, start, today time.Time) []Day` | `planner/weather.go` | 9 |
| `func PickStays(center Place, stays []Stay) []Stay` | `planner/stays.go` | 10 |
| `func BookingURL(city, country string, start time.Time, days int) string` | `planner/stays.go` | 10 |
| `var PlaceTypes map[string]Kind` (empty placeholder, then generated) | `planner/placetypes_gen.go` | 1, then 13 |
| `func Build(ctx context.Context, req Request, places PlaceSource, forecast ForecastSource, stays StaySource, now time.Time) (*Plan, error)` | `planner/build.go` | 14 |
| `func NewWikimediaAPI(client *http.Client) *WikimediaAPI` (nil means a default client with a 30 s timeout) | `api/wikimedia.go` | 3 |
| `func (a *WikimediaAPI) GetEntities(ctx context.Context, ids []string, props string) (map[string]Entity, error)` | `api/wikimedia.go` | 3 (the place-type tool reuses it) |
| `func NewOpenMeteoForecastAPI(client *http.Client) *OpenMeteoForecastAPI` | `api/openmeteo_forecast.go` | 4 |
| `func NewOverpassAPI(client *http.Client, urls []string) *OverpassAPI` and `var DefaultOverpassURLs []string` | `api/overpass.go` | 5 |
| `func NewCachedPlaceSource(src planner.PlaceSource) planner.PlaceSource`, plus `NewCachedForecastSource` and `NewCachedStaySource` | `sourcecache.go` (root) | 6 |
| `type TripPlanner interface { Plan(ctx context.Context, req planner.Request) (*planner.Plan, error) }` and `func (s *Server) SetTripPlanner(p TripPlanner)` | `trip.go` (root) | 11 |
| `func NewTripPlanner(places planner.PlaceSource, forecast planner.ForecastSource, stays planner.StaySource) TripPlanner` | `tripplanner.go` (root) | 15 |

## Parallel lanes

```
Phase 0   [1 Contract]  [2 Remove old code]                                  ← 2 agents
              │                  │
Phase 1   Sources:  [3 Wikipedia/Wikidata] [4 Forecast] [5 Overpass] [6 Cache]
          Planner:  [7 Rank + boosts] [8 Group days] [9 Weather] [10 Stays]
          UI:       [11 Planner route + card] → [12 Stay card + credits]     ← up to 10 agents
              │
Phase 2   [13 Place-type tool] (needs 3)    [14 Build] (needs 7–10)          ← 2 agents
              │
Phase 3   [15 Wire it up] (needs 3–6, 11–14)   [16 City acceptance tests] (needs 3–5, 14; red until 13 lands)
              │
Phase 4   [17 README + CHANGELOG] (needs 15)
```

## Who owns which files

Parallel tasks never share a file. The shared files are edited in a fixed order.

| File | Tasks, in order |
|---|---|
| `server.go`, `reporters.go`, `server_test.go` | 2 → 11 |
| `views/content_fragment.go.tpl`, `views/index.go.tpl` | 2 → 11 → 12 |
| `views/trip_card.go.tpl` | 11 → 12 |
| `cmd/web/main.go`, `env.go`, `env_test.go` | 2 → 15 |
| `database.go` | 6 |
| `planner/placetypes_gen.go` | 1 (placeholder) → 13 |
| `README.md`, `CHANGELOG.md` | 17 only |
| `planner/*`, `api/*` (new files) | one task per file, listed in each task |

## Rules for every agent

- **Only touch the files your task lists.** If you need something outside them, or a change to the contract, stop and ask.
- **Build test-first** (see below). No code without a test that failed first.
- **No network in tests.** Stub HTTP with `roundTripFunc` and keep recorded responses under `testdata/`. Live checks are opt-in with `LIVE_API_TESTS=1`.
- **Match the surrounding code:** `log.Printf` for logs, errors wrapped with `%w`, Fiber handlers like the ones in `server.go`, tests named `TestThing_Behavior` with testify's `assert` and `require`.
- **Before you finish:** `make lint` and `make test` pass. `make test` runs with `-race` and needs CGO for SQLite.
- **Delivery** (branches, PRs, merge order) is handled by the human.

## How every task is built (TDD)

```
 RED                        GREEN                    REFACTOR
 Write the task's tests  →  Least code that makes  →  Clean up, re-running
 and watch them fail        them pass                 the focused tests
```

- **RED must be an assertion failure.** In Go, a missing function is a compile error, and that doesn't count. Add the function with its contract signature and a zero-value body first, run the tests, and watch them fail. Keep the failing output in your notes.
- **A test that passes straight away proves nothing.** The one exception is a *guard test* for something that must keep working through a change (Task 2's old cache rows). Label those as guards.
- **Tests are the spec.** Each task in todo.md lists its tests by name. Names read like behavior (`TestSchedule_FiveMMIsRainy`), and each test covers one concept, with Arrange, Act, Assert.
- **Check results, not calls.** Assert on returned data and rendered HTML. Where call counts matter for cost (no extra API requests), keep the check at the HTTP boundary.
- **Test doubles, most preferred first:** the real code, then a fake (a small struct with canned data), then a stub (`roundTripFunc`). Don't add new gomock mocks.
- **Bugs follow the Prove-It pattern:** first a test that reproduces the bug and fails, then the fix, then the full suite.
- **Don't re-run unchanged code.** Run the focused tests during the loop and the full suite once at the end. Run again only after an edit.

**Test sizes by layer:**

| Layer | Tasks | Size | What it may use |
|---|---|---|---|
| Planning logic (`planner/`) | 1, 7, 8, 9, 10, 14 | **Small** | Nothing: no I/O, no network, no database |
| API clients (`api/`) | 3, 4, 5 | **Small** | Stubbed HTTP with recorded responses in `testdata/` |
| Place-type rules | 13 | **Small** | A fake class graph in the test |
| Cache | 6 | **Medium** | The package's SQLite test database |
| Route and templates | 2, 11, 12, 15 | **Medium** | An in-process Fiber app and `./views` |
| City acceptance tests | 16 | **Medium** | Recorded responses on disk, no network |
| Live APIs and the browser | Checkpoints, opt-in tests | **Large** | The real internet and a real browser, by hand or with `LIVE_API_TESTS=1` |

Most tests are small. There's about one medium test per boundary and a few large checks at the checkpoints.

**Task 16 is written ahead of time:** its city tests are written after `Build` exists but **before Task 13's type list lands**. With the empty placeholder they fail (RED), and Task 13 turns them green.

## Task List

Full details, acceptance criteria and verification steps are in [todo.md](todo.md).

### Phase 0: Foundation
- [ ] Task 1: Planner contract and geo helpers
- [ ] Task 2: Remove the old hotels and itinerary code

### Checkpoint A: Foundation
- [ ] `make test` and `make build` pass
- [ ] The site still shows weather and videos, with no hotels card

### Phase 1: Parallel pieces
- [ ] Task 3: Wikipedia + Wikidata place source
- [ ] Task 4: Open-Meteo daily forecast source
- [ ] Task 5: Overpass stay source with fallback servers
- [ ] Task 6: SQLite source cache that keeps old data on errors
- [ ] Task 7: Rank places, with the type filter and boost list
- [ ] Task 8: Group places into days
- [ ] Task 9: Schedule days around the forecast
- [ ] Task 10: Pick places to stay and build the booking link
- [ ] Task 11: Planner route and "Plan my trip" card (with a fake planner)
- [ ] Task 12: Stay card, photo credits and data credits

### Checkpoint B: Pieces
- [ ] `make lint` and `make test` pass on `main` with all Phase 1 work merged
- [ ] Every task noted its RED output, and no test is skipped or disabled
- [ ] The card renders a fake plan on the real page

### Phase 2: Joining the logic
- [ ] Task 13: Place-type tool and the generated type list (human review)
- [ ] Task 14: `Build`, which runs the whole planning chain

### Phase 3: Integration
- [ ] Task 15: Wire the real planner into the app
- [ ] Task 16: City acceptance tests for Lisbon, Tavira and Kyoto (written before Task 13 lands)

### Checkpoint C: End to end
- [ ] Every "Done when" item from the one-pager passes on a local run
- [ ] Human review before the docs task

### Phase 4: Polish
- [ ] Task 17: README "How the planner works" and CHANGELOG

### Checkpoint D: Complete
- [ ] All acceptance criteria met, CI green, ready to deploy

## Risks and Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| The contract needs to change mid-way | Medium | Keep it small; changes go through the human; affected tasks rebase |
| The place-type list lets junk in or drops good places | High | Task 13's output gets a human review; Task 16 asserts known-good and known-bad places for 3 cities |
| Public APIs are flaky while recording fixtures | Medium | Record once, commit the files, retry by hand; tests never hit the network |
| Both Overpass servers are down in production | Medium | Retries and fallback (Task 5), old cached data (Task 6), hide the stay card (Task 14) |
| Grouping into days gives odd days (zig-zag walks, one huge day) | Medium | Task 8 tests with a Lisbon-like layout; Task 16 checks real data |
| Merge conflicts in shared files | Medium | File-ownership table above; shared files edited in a fixed order |
| Wikimedia rate limits | Low | `User-Agent` on every request, 30-day cache, split searches only when capped |
| Tests that pass without testing anything | Medium | RED must be an assertion failure and noted per task; Checkpoint B checks the notes |

## Decisions (2026-09-19)

- ✅ **Route:** `GET /plan` returns an HTMX fragment for the MVP. v2 moves plans to `/trip/:city`.
- ✅ **Delivery** (branches, PRs, merge order) is handled by the human, not the task list.
- ✅ **v2 gets broken down after the MVP ships.**

## Open Questions

None right now.
