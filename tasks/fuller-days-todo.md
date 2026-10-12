# Fuller itinerary days — Tasks

Source of truth for progress. Read [the plan](fuller-days-plan.md) first.

**Status:** F1–F7 complete on 2026-10-12. The owner's follow-up caps each complete daily walking loop at 15 km; recorded fixture days have 3–4 stops and stay within the cap. `make test lint build` passes with a temporary Go 1.27-compatible linter.

**How to read this file**
- Tasks run in order F1 → F6, by **one agent**: they share `days.go`, and F5 reuses F2's `fillShortDays`.
- **Owns** = the only files the task may edit. None overlap with the open Writers' Map tasks in [todo.md](todo.md) (PRs 8–9 don't touch `internal/planner/`).
- Start from an up-to-date `main` (`git fetch` and check `git status -sb`).
- TDD per [AGENTS.md](../AGENTS.md) §5: see each test fail for the intended reason before changing `days.go`.
- No commits or pushes. The owner commits.

**User journey:** *As a traveller, I want every day of my trip to have several nearby things to do, so that no day feels empty.*

---

### F1. Record the baseline
- [x] Done

**Owns:** `docs/tdd/fuller-days.tdd.md` (new)

- Table of stops per day for Lisbon, Kyoto and Tavira, 1–5 days, from the recorded fixtures. Use a temporary test and delete it afterwards; don't commit it. This one works (save as `internal/planner/zz_baseline_scratch_test.go`):

  ```go
  package planner_test

  import (
    "fmt"
    "testing"
  )

  func TestZZBaselineScratch(t *testing.T) {
    for _, c := range []city{lisbon, kyoto, tavira} {
      for d := 1; d <= 5; d++ {
        p := cityPlan(t, c, d)
        s := ""
        for _, day := range p.Days {
          s += fmt.Sprintf("%d ", len(day.Stops))
        }
        fmt.Printf("%s %d: %s\n", c.name, d, s)
      }
    }
  }
  ```

  Run: `go test -v -run TestZZBaselineScratch -count=1 ./internal/planner/`
- Expected today (measured 2026-10-12): Lisbon 5 days `4 3 4 3 2`; Kyoto 3 days `4 2 4`, 4 days `4 2 4 2`, 5 days `4 3 4 2 2`; Tavira 2+ days `4 3`. The full table is in [the plan](fuller-days-plan.md).

**Verify:** the table is in the TDD doc, with the command that produced it.

### F2. Fill short days from the full ranked list
- [x] Done

**Owns:** `internal/planner/days.go`, `internal/planner/days_test.go`

- [x] RED `TestGroupDays_FillsAShortDayFromBeyondTheTop20`: a 2-stop day gains place #25, which sits 0.5 km away.
- [x] RED `TestGroupDays_TopsUpTwoStopDaysNotOnlySingletons`
- [x] RED `TestGroupDays_FillPrefersFamousPlacesWithinAShortWalk`: within 1.5 km wins over a more famous place 2.5 km away.
- [x] Tighten `TestGroupDays_EveryDayHasThreeToFourStopsUnlessNothingIsInReach`: also no unused ranked place within reach.
- [x] GREEN: `fillShortDays(clusters, ranked)` replaces `fillSingletons`. Measure from the nearest stop with `nearestKM`.
- [x] Update the doc comments on `GroupDays` and `PullReachKM`.

**Verify:** `go test ./internal/planner/ -run TestGroupDays`

### F3. Swap lone days for another area
- [x] Done

**Owns:** `internal/planner/days.go`, `internal/planner/days_test.go`

- [x] RED `TestGroupDays_SwapsALoneStopForAnotherArea`
- [x] RED `TestGroupDays_DropsALoneDayWhenNoAreaCanFillIt`: fewer groups are returned.
- [x] Keep green: `AlwaysKeepsTheMostFamousPlace`, `OnePlaceGivesOneDay`, `NoPlaceAppearsTwice`, `SameInputGivesTheSameDays`, `LisbonLikeDaysWalkUnder15Km`.
- [x] GREEN: `replaceLoneDays(clusters, ranked)` after the fill. The most famous place's day is never dropped, and a one-place town keeps its plan.

**Verify:** `go test ./internal/planner/ -run TestGroupDays`

### F4. City regressions
- [x] Done

**Owns:** `internal/planner/cities_test.go`

- [x] `TestCities_EveryDayHasAtLeastTwoStops`: Lisbon, Kyoto and Tavira, 1–5 days (covers Kyoto's thin days on 3-, 4- and 5-day trips).
- [x] `TestCities_KyotoArashiyamaDayIncludesTenryuji` (`Q1759226`).
- [x] Replace `TestCities_KyotoOnlyStrandsAnIsolatedStop` with the 2-stop minimum, because isolated stops are now swapped.
- [x] `TestCities_TaviraFiveDaysGetsFewerDaysAndANote` still passes.

**Verify:** `go test ./internal/planner/ -run TestCities`

### F5. Fill days built around saved places · *after F2*
- [x] Done

**Owns:** `internal/planner/build.go` (`seedGroups` and `nearestUnused` only), `internal/planner/build_test.go`

Saved places (`Request.Include`, Writers' Map 7e) never reach `GroupDays`: `seedGroups` (`build.go`) clusters them into days and fills each with `nearestUnused` from the top-20 pool only.

- [x] RED `TestBuild_FillsAnIncludedDayFromBeyondTheTop20`: an included place whose only neighbours within reach rank below 20 gets a day of 2+ stops.
- [x] Keep green, already covering the "never drop or swap" rule: `TestBuild_PlansIncludedPlacesWhenNothingElseIsNearby` (an isolated included place keeps its own day) and `TestBuild_RainDoesNotDropAnIncludedPlace`.
- [x] Keep green, the other 7e tests: `WithoutIncludeGivesTheUnseededPlan`, `IncludedPlacesAppearInTheDays`, `IncludedPlacesThatAreCloseShareADay`, `IncludedPlaceAlsoInTheCandidatesIsPlannedOnce`, `MoreIncludedPlacesThanDaysCanHoldAreAlsoSuggested`, `ADayOfIncludedPlacesHoldsAtMostFourStops`.
- [x] GREEN: in `seedGroups`, fill the included days with `fillShortDays` from `rest` (all ranked places not included) instead of `nearestUnused` from `pool`. Don't call `replaceLoneDays` on included days.
- [x] Delete `nearestUnused` (now unused).

**Verify:** `go test ./internal/planner/...`

### F6. Wrap up
- [x] Done

**Owns:** `docs/tdd/fuller-days.tdd.md`, `tasks/fuller-days-plan.md` (status line), this file (checkboxes)

- [x] After table next to the F1 baseline: no day below 2 stops.
- [x] `go test -cover ./internal/planner/...` ≥80%.
- [x] `make test lint build` green (temporary Go 1.27-compatible linter on `PATH`).
- [x] TDD doc: journeys, RED/GREEN excerpts per behaviour, a guarantees table, coverage and gaps.
- [x] List the changed files for the owner.

**Verify:** `make test lint build`

### F7. Cap the complete daily walk at 15 km · owner follow-up
- [x] Done

**Owns:** `internal/planner/days.go`, `days_test.go`, `build.go`, `build_test.go`, `cities_test.go`, `weather.go`, `weather_test.go`, `docs/tdd/fuller-days.tdd.md`, and the status/owner follow-up in the task files.

- [x] RED: a farther place can fill a short day, but a candidate that makes the return walk longer than 15 km cannot.
- [x] RED: long initial groups, included-place groups, rainy swaps and recorded city days can exceed 15 km.
- [x] GREEN: enforce the cap in grouping, saved-place grouping, filling and rainy swaps; suggest saved places that cannot fit.
- [x] Verify 1–5 day fixture trips for Lisbon, Kyoto and Tavira, package coverage ≥80%, and `make test lint build`.
