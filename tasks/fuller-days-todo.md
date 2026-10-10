# Fuller itinerary days — Tasks

Source of truth for progress. Read [the plan](fuller-days-plan.md) first.

**Status:** not started (2026-10-10).

**How to read this file**
- Tasks run in order F1 → F4 and F6, by one agent: they share `days.go`. F5 waits for Writers' Map 7e.
- **Owns** = the only files the task may edit. None overlap with the Writers' Map tasks in [todo.md](todo.md).
- TDD per [AGENTS.md](../AGENTS.md) §5: see each test fail for the intended reason before changing `days.go`.
- No commits or pushes. The owner commits.

**User journey:** *As a traveller, I want every day of my trip to have several nearby things to do, so that no day feels empty.*

---

### F1. Record the baseline
- [ ] Done

**Owns:** `docs/tdd/fuller-days.tdd.md` (new)

- Table of stops per day for Lisbon, Kyoto and Tavira, 1–5 days, from the recorded fixtures. Use a temporary test or a scratch copy, and don't commit it.
- Expected today: Lisbon 5 days `4 3 4 3 2`, Kyoto 5 days `4 3 4 2 2`.

**Verify:** the table is in the TDD doc, with the command that produced it.

### F2. Fill short days from the full ranked list
- [ ] Done

**Owns:** `internal/planner/days.go`, `internal/planner/days_test.go`

- [ ] RED `TestGroupDays_FillsAShortDayFromBeyondTheTop20`: a 2-stop day gains place #25, which sits 0.5 km away.
- [ ] RED `TestGroupDays_TopsUpTwoStopDaysNotOnlySingletons`
- [ ] RED `TestGroupDays_FillPrefersFamousPlacesWithinAShortWalk`: within 1.5 km wins over a more famous place 2.5 km away.
- [ ] Tighten `TestGroupDays_EveryDayHasThreeToFourStopsUnlessNothingIsInReach`: also no unused ranked place within reach.
- [ ] GREEN: `fillShortDays(clusters, ranked)` replaces `fillSingletons`. Measure from the nearest stop with `nearestKM`.
- [ ] Update the doc comments on `GroupDays` and `PullReachKM`.

**Verify:** `go test ./internal/planner/ -run TestGroupDays`

### F3. Swap lone days for another area
- [ ] Done

**Owns:** `internal/planner/days.go`, `internal/planner/days_test.go`

- [ ] RED `TestGroupDays_SwapsALoneStopForAnotherArea`
- [ ] RED `TestGroupDays_DropsALoneDayWhenNoAreaCanFillIt`: fewer groups are returned.
- [ ] Keep green: `AlwaysKeepsTheMostFamousPlace`, `OnePlaceGivesOneDay`, `NoPlaceAppearsTwice`, `SameInputGivesTheSameDays`, `LisbonLikeDaysWalkUnder15Km`.
- [ ] GREEN: `replaceLoneDays(clusters, ranked)` after the fill. The most famous place's day is never dropped, and a one-place town keeps its plan.

**Verify:** `go test ./internal/planner/ -run TestGroupDays`

### F4. City regressions
- [ ] Done

**Owns:** `internal/planner/cities_test.go`

- [ ] `TestCities_EveryDayHasAtLeastTwoStops`: Lisbon, Kyoto and Tavira, 1–5 days.
- [ ] `TestCities_KyotoArashiyamaDayIncludesTenryuji` (`Q1759226`).
- [ ] Replace `TestCities_KyotoOnlyStrandsAnIsolatedStop` with the 2-stop minimum, because isolated stops are now swapped.
- [ ] `TestCities_TaviraFiveDaysGetsFewerDaysAndANote` still passes.

**Verify:** `go test ./internal/planner/ -run TestCities`

### F5. Respect included places · *after Writers' Map 7e*
- [ ] Done

**Owns:** `internal/planner/days.go`, `internal/planner/days_test.go`

- [ ] RED: an included place is never swapped out by `replaceLoneDays`, and its day is never dropped.
- [ ] GREEN: treat `Include` places as used in `fillShortDays` and keep them in `replaceLoneDays`.

**Verify:** `go test ./internal/planner/...`

### F6. Wrap up
- [ ] Done

**Owns:** `docs/tdd/fuller-days.tdd.md`, `tasks/fuller-days-plan.md` (status line), this file (checkboxes)

- [ ] After table next to the F1 baseline: no day below 2 stops.
- [ ] `go test -cover ./internal/planner/...` ≥80%.
- [ ] `make test lint build` green.
- [ ] TDD doc: journeys, RED/GREEN excerpts per behaviour, a guarantees table, coverage and gaps.
- [ ] List the changed files for the owner.

**Verify:** `make test lint build`
