# 1e — Fields the Mockup Needs: TDD Evidence

**Source plan:** [`tasks/todo.md`](../../tasks/todo.md) PR 1, subtask 1e, with the scope agreed on October 8, 2026: blog names come from a new `name` field in `writerdata/sources.json`, distance and walking time are per itinerary day, and the plan-stop `writers_pick` flag is part of 1e.

**Status:** done. The Go and fixture changes are tested. The GeoJSON `writers` and `has_summary` properties and the plan-stop `writers_pick` flag are contract-only: no Go type builds GeoJSON or plan-stop views yet (PR 3 and 7g do), so they are recorded in [`docs/writers-map-contract.md`](../writers-map-contract.md) without tests.

## User journeys

- As a **visitor**, I want the panel to show where a place is (*kind · area*), so that I know which part of the destination it's in.
- As a **visitor**, I want each writer's post listed under the blog's name, not its host, so that I recognise who wrote it.
- As a **visitor**, I want a writer's walk to show its distance and walking time for one day, so that I can judge whether it fits my day.
- As a **map developer**, I want the Madeira fixture to carry all these fields with consistent values, so that I can build the panel and walk card on the fake source.

## Task report

### Blog names in the source list
- **Summary:** every source in `writerdata/sources.json` has a non-empty `name`.
- **RED**, `go test ./writerdata/ -run TestSources_EverySourceHasABlogName`:
  ```
      embed_test.go:50:
          Error:      	Should NOT be empty, but was
          Test:       	TestSources_EverySourceHasABlogName
          Messages:   	lisbonlisboaportugal.com
  FAIL	weatherservice/writerdata	0.322s
  ```
- **GREEN** (names added): `ok  	weatherservice/writerdata	0.363s`

### `area`, `blog_name` and per-day itinerary fields in the types
- **Summary:** `Place.Area` encodes as `area`, `Mention.BlogName` as `blog_name`, and `Itinerary.Days` is now `[]ItineraryDay` with `qids`, `distance_km` and `walk_minutes`. `writermap.WalkSpeedKMH` (5 km/h) converts distance into minutes.
- **RED (compile)**, `go test ./internal/writermap/`:
  ```
  internal/writermap/destinationdata_test.go:32:65: unknown field Area in struct literal of type Place
  internal/writermap/destinationdata_test.go:41:56: unknown field BlogName in struct literal of type Mention
  internal/writermap/destinationdata_test.go:50:33: undefined: ItineraryDay
  FAIL	weatherservice/internal/writermap [build failed]
  ```
- **GREEN:** `ok  	weatherservice/internal/writermap	0.314s`

### The Madeira fixture carries the new fields
- **Summary:** every place has an area; every mention's `blog_name` equals its host's name in `sources.json`; each day's `distance_km` matches `planner.DistanceKM` between consecutive stops (within 0.05 km) and `walk_minutes` matches that distance at 5 km/h.
- **RED (compile)**, `go vet ./internal/writermap/writermaptest/`:
  ```
  vet: internal/writermap/writermaptest/fake_test.go:223:28: place.Area undefined (type writermap.Place has no field or method Area)
  ```
- **GREEN** (types added, fixture filled): `ok  	weatherservice/internal/writermap/writermaptest	0.424s`
- **Can fail:** with the first `walk_minutes` in the fixture changed to 1, `go test ./internal/writermap/writermaptest/ -run DayDistance` failed with `expected: 138` / `actual  : 1`. The fixture was then restored.

## Guarantees

| What is guaranteed | Test | Type | Result |
| --- | --- | --- | --- |
| Every source has a blog name | `TestSources_EverySourceHasABlogName` | unit | PASS |
| A place's area is encoded as `area` | `TestPlace_EncodesArea` | unit | PASS |
| A mention's blog name is encoded as `blog_name` | `TestMention_EncodesBlogName` | unit | PASS |
| An itinerary day encodes `qids`, `distance_km`, `walk_minutes` | `TestItinerary_DayEncodesStopsDistanceAndWalkTime` | unit | PASS |
| Every fixture place has an area | `TestMadeiraFixture_EveryPlaceHasAnArea` | unit | PASS |
| Fixture blog names match `sources.json` | `TestMadeiraFixture_BlogNamesMatchTheSourcesFile` | unit | PASS |
| Fixture day distance and walk time match the stops | `TestMadeiraFixture_DayDistanceAndWalkTimeMatchTheStops` | unit | PASS |

## Coverage and full suite

`go test -cover ./internal/writermap/... ./writerdata/`:
```
ok  	weatherservice/internal/writermap	0.242s	coverage: [no statements]
ok  	weatherservice/internal/writermap/writermaptest	0.366s	coverage: 97.6% of statements
ok  	weatherservice/writerdata	0.299s	coverage: [no statements]
```

`make test lint build`: all packages `ok`, `golangci-lint` reported `0 issues.`, and `go build -o bin/traveltab ./cmd/web` succeeded.

## Known gaps

- **Contract-only fields:** GeoJSON `writers` and `has_summary` and the plan-stop `writers_pick` flag have no code yet; PR 3, 7a, 7f and 7g must test them when they build those views.
- **Fixture areas are hand-written** from Madeira's municipalities, not fetched from Wikidata P131, and the blog names were taken from the sites' known titles without a fresh check.
- **Walking time on an island:** Madeira's writer days span many kilometres (one is over 11 km straight-line), so the walk card may need a "drive" wording for long days. That is a UI decision for 7c.
