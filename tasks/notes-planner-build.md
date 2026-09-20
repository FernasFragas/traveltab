# Notes: Task 14 (`Build`)

Files owned by this lane: `planner/build.go`, `planner/build_test.go`. Written test-first, with the
RED run kept below.

## How it was built

1. Added `Build` with its contract signature and a `return nil, nil` body, so a failing run would be
   an assertion failure and not a compile error.
2. Wrote all 12 tests listed for Task 14 in [todo.md](todo.md), ran them, kept the output (RED).
3. Made them pass with the seven ordered steps and nothing else, then checked the tests bite by
   mutating the implementation (see "Mutation checks" below).

## What `Build` does

`Build(ctx, req, places, forecast, stays, now)` in `planner/build.go`, in the spec's order:

1. `req.Days` outside 1..`MaxDays` is an error; nothing is called.
2. Start `forecast.DailyForecast(...)` alongside `places.PlacesNear(req.Lat, req.Lon)`.
   A places error ends the plan, wrapped with `%w`, and cancels the forecast lookup.
3. `Rank(candidates, PlaceTypes, Boosts)` then `GroupDays(ranked, req.Days)`. No groups is an error.
   Fewer groups than days adds the short-trip note.
4. Read the forecast result. An error is logged and the forecast becomes `nil`, so the plan is
   made without weather and gets an explanatory note. A start more than `ForecastMaxDays` days
   after today adds the far-trip note.
5. `today` is `now.In(forecastLocation(fc))` — the city's zone, UTC when there is no forecast — and
   `Schedule(groups, ranked, fc, req.Start, today)` dates the days. The full ranked list is the pool,
   so a rainy day has indoor places to swap in.
6. `Medoid` of every stop in the plan is the center. `stays.StaysNear(center)`; an error is logged
   and sets `StaysNote`, otherwise `PickStays(center, ...)` fills `Stays`.
7. `BookingURL(req.City, req.Country, req.Start, req.Days)`.

There is no planning logic of its own: the only helpers are `shortTrip` (the note's wording) and
`planStops` (flattens the days for `Medoid`). The four note texts sit together in one `const` block
at the top of the file.

### Decisions worth knowing

- **Notes compose.** They are collected in a slice and joined with a space, so a far small
  town gets "Tavira has enough highlights for 1 day, so here's a 1-day plan. Forecast appears closer
  to your trip".
- **`BookingURL` uses `req.Days`, not `len(plan.Days)`.** The link is the stay the traveller asked
  for; a short town plan does not shorten the booking. Easy to flip if the human prefers the other
  reading.
- **A failed forecast explains the fallback.** The note says "Weather is unavailable right now,
  so the days are not ordered by the forecast." The failure is also logged with `log.Printf`.
- **`Certain` is not `Forecast != nil`.** Per HANDOVER, the UI must gate weather on `Forecast != nil`.
  `Build` does not touch either field; it just hands `Schedule` the right `today`.

## RED (assertion failures against the `return nil, nil` body)

`CGO_ENABLED=1 go test -race -count=1 -run Build ./planner/` — 12 failures, one per test, all
assertion failures (no compile errors):

```
--- FAIL: TestBuild_PlansTheRequestedNumberOfDays (0.00s)
    build_test.go:156: Error: Expected value not to be nil.
--- FAIL: TestBuild_RejectsZeroOrSixDays (0.00s)
    build_test.go:176: Error: An error is expected but got nil.  Messages: days=0
--- FAIL: TestBuild_FailsWhenPlacesCannotBeLoaded (0.00s)
    build_test.go:188: Error: An error is expected but got nil.
--- FAIL: TestBuild_FailsWhenNoPlaceSurvivesRanking (0.00s)
    build_test.go:204: Error: An error is expected but got nil.
--- FAIL: TestBuild_SmallTownNoteNamesTheCity (0.00s)
    build_test.go:216: Error: Expected value not to be nil.
--- FAIL: TestBuild_PlansWithoutWeatherWhenTheForecastFails (0.00s)
    build_test.go:228: Error: Expected value not to be nil.
--- FAIL: TestBuild_SaysTheForecastComesLaterForFarTrips (0.00s)
    build_test.go:248: Error: Expected value not to be nil.
--- FAIL: TestBuild_StillPlansWhenStaysFail (0.00s)
    build_test.go:266: Error: Expected value not to be nil.
--- FAIL: TestBuild_PicksStaysAroundThePlanCenter (0.00s)
    build_test.go:285: Error: Expected value not to be nil.
--- FAIL: TestBuild_SetsTheBookingLink (0.00s)
    build_test.go:300: Error: Expected value not to be nil.
--- FAIL: TestBuild_TodayIsTakenInTheCityTimeZone (0.00s)
    build_test.go:317: Error: Expected value not to be nil.
--- FAIL: TestBuild_SameInputGivesTheSamePlan (0.00s)
    build_test.go:339: Error: Expected value not to be nil.
FAIL	weatherservice/planner	0.301s
```

## GREEN

`CGO_ENABLED=1 go test -race -count=1 -run Build ./planner/` — all 12 pass. The whole package,
`go vet ./planner/...`, `golangci-lint run ./planner/...` (0 issues) and
`CGO_ENABLED=1 go test -race -count=1 ./...` are green too.

## The test doubles

- `buildPlaces`, `buildForecast`, `buildStays` are small structs with canned data or a canned error.
  The place and stay ones record the coordinates they were asked about, which is how
  "stays are looked up around the center, not the request" is checked.
- `useTestPlaceTypes(t)` swaps `PlaceTypes` for `{T_museum: Indoor, T_park: Outdoor}` and restores it
  with `t.Cleanup`, so nothing here depends on Task 13's generated list. It is global state, so **no
  test in this file calls `t.Parallel()`**.
- Fixtures: `buildCity` (4 museums north, 4 parks 11 km south — exactly two walking days),
  `buildSmallTown` (4 places in one cluster — one day), `buildStarCity` (one place ringed by four,
  so the medoid is the middle one whichever ring places survive the day's stop limit).

`buildCity` started as 12 places. Task 8's `selectCompact` landed mid-lane and, with six places per
cluster, it filled the first day from the museums and then seeded the second day from the two
leftover museums, dragging a museum into the park day. Eight places (one day's worth per cluster)
keeps the fixture saying what it means whichever way `GroupDays` is tuned.

## Mutation checks (the tests bite)

Each change below was applied to `build.go`, the focused run confirmed the named failure, and the
file was restored:

| Mutation | Test that caught it |
|---|---|
| `today := now` instead of `now.In(forecastLocation(fc))` | `TestBuild_TodayIsTakenInTheCityTimeZone` |
| stays looked up at `req.Lat/Lon` instead of the center | `TestBuild_PicksStaysAroundThePlanCenter` |
| `>= ForecastMaxDays` instead of `>` | `TestBuild_SaysTheForecastComesLaterForFarTrips` |
| `%v` instead of `%w` on the places error | `TestBuild_FailsWhenPlacesCannotBeLoaded` |
| one day dropped from `Schedule`'s result | 3 tests |

The first version of `TestBuild_PicksStaysAroundThePlanCenter` put the request on the same
coordinates as the center, so the second mutation slipped through. The request now sits ~5 km away.

## For the human

**The note's wording.** todo.md writes the short-trip note as
`"<City> has enough highlights for N days, so here's an N-day plan."`. Rendered literally with `N=1`
that reads "for 1 days, so here's an 1-day plan". The implementation keeps every word of the spec but
makes the article and the plural agree with the number, which for `N` in 1..4 means:

- `Tavira has enough highlights for 1 day, so here's a 1-day plan.`
- `Tavira has enough highlights for 2 days, so here's a 2-day plan.`

The four note strings are consts at the top of `build.go`, so this is a one-line change if the human
wants the literal text back. **Task 16 should assert against whatever is decided here.**

## Completion follow-up (2026-09-20)

The working tree already contained three new tests for concurrent source loading and forecast
failure notes, but their implementation was unfinished. Preserved those tests, updated the older
contradictory empty-note assertion, and added a regression for canceling the forecast on a place
source failure. The source interfaces, requested booking duration, and other note behavior are
unchanged. A buffered forecast-result channel allows an early place error to return immediately;
a deferred context cancellation stops the abandoned request.

**RED:** Initially isolated the Build tests with an explicit Go file list because the concurrent
Wikimedia work temporarily did not compile. All failures below were assertions against the old
Build implementation, not compilation failures:

```
TestBuild_SaysWeatherIsUnavailableWhenTheForecastFails: expected weather note, actual ""
TestBuild_KeepsTheOtherNotesWhenTheWeatherIsMissing: weather note missing after short-trip note
TestBuild_LoadsPlacesAndTheForecastAtTheSameTime: placesSawForecast was false
TestBuild_PlansWithoutWeatherWhenTheForecastFails: expected weather note, actual ""
TestBuild_CancelsTheForecastWhenPlacesFail: parallel forecast lookup was not canceled
```

**GREEN:** `go test -race -count=1 -run Build ./planner/` passes after implementing the note and
parallel lookup. The parent agent runs the full suite, lint and build after all lanes integrate.
