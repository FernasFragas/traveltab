# Fuller itinerary days: TDD evidence

**Source:** `tasks/fuller-days-plan.md`, tasks F1–F6 in `tasks/fuller-days-todo.md`.

**User journeys**

- As a traveller, I want several nearby things to do each day, so that no day feels empty.
- As a traveller who saved places, I want those places kept in the plan and their days filled with nearby highlights when available.

## Baseline

Before production changes, `go test -v -run TestZZBaselineScratch -count=1 ./internal/planner/` passed and printed the stops per day below. The temporary test was removed after the measurement.

| City | 1 day | 2 days | 3 days | 4 days | 5 days |
| --- | --- | --- | --- | --- | --- |
| Lisbon | 4 | 4 4 | 4 4 4 | 4 3 4 3 | 4 3 4 3 2 |
| Kyoto | 4 | 4 4 | 4 2 4 | 4 2 4 2 | 4 3 4 2 2 |
| Tavira | 4 | 4 3 | 4 3 | 4 3 | 4 3 |

## RED: city and saved-place regressions

`go test ./internal/planner/ -run 'TestCities_(EveryDayHasAtLeastTwoStops|KyotoArashiyamaDayIncludesTenryuji)$' -count=1` exited 1. The two-stop minimum passed on the recorded cities, while the new nearby-place check failed:

```text
--- FAIL: TestCities_KyotoArashiyamaDayIncludesTenryuji (0.01s)
cities_test.go:189: Error: []string{"Q2859566", "Q23579173"} does not contain "Q1759226"
Messages: Tenryū-ji should join the bamboo grove's Arashiyama day
FAIL weatherservice/internal/planner 0.340s
```

`go test ./internal/planner/ -run '^TestBuild_FillsAnIncludedDayFromBeyondTheTop20$' -count=1` exited 1. The saved place had no added stop because its reachable neighbours fell beyond the top 20:

```text
--- FAIL: TestBuild_FillsAnIncludedDayFromBeyondTheTop20 (0.00s)
extra elements in list A: "Q921", "Q922"
listB: []string{"Q901"}
FAIL weatherservice/internal/planner 0.277s
```

## RED and GREEN: group and replace walking days

For a short day, `go test ./internal/planner/ -run 'TestGroupDays_(FillsAShortDayFromBeyondTheTop20|TopsUpTwoStopDaysNotOnlySingletons)$' -count=1` failed before the change: `Q25` was absent from the plan and the Q12/Q16 day had 2 stops where the test expected 3. A direct fill test also failed to compile with `undefined: fillShortDays` before that helper existed.

After `fillShortDays` was added, `go test ./internal/planner/ -run 'TestGroupDays_(FillsAShortDayFromBeyondTheTop20|TopsUpTwoStopDaysNotOnlySingletons|FillPrefersFamousPlacesWithinAShortWalk|FillMeasuresFromTheNearestStop|EveryDayHasThreeToFourStopsUnlessNothingIsInReach)$' -count=1` gave `ok weatherservice/internal/planner 0.418s`.

For lone days, `go test ./internal/planner/ -run 'TestGroupDays_(SwapsALoneStopForAnotherArea|DropsALoneDayWhenNoAreaCanFillIt)$' -count=1` failed before the change: the swap test still contained `L1` and omitted `B1`; the drop test returned 2 groups where 1 was expected. After `replaceLoneDays` was added, `go test ./internal/planner/ -run 'TestGroupDays' -count=1` gave `ok weatherservice/internal/planner 0.428s`.

## GREEN: city and saved-place regressions

`go test ./internal/planner/ -run TestCities -count=1` gave `ok weatherservice/internal/planner 0.474s`. `go test ./internal/planner/ -run '^TestBuild_FillsAnIncludedDayFromBeyondTheTop20$' -count=1` gave `ok weatherservice/internal/planner 0.360s`. The existing Build tests also passed: `go test ./internal/planner/ -run '^TestBuild_' -count=1` gave `ok weatherservice/internal/planner 0.260s`.

## After

With the temporary city test again, `go test -v -run TestZZBaselineScratch -count=1 ./internal/planner/` passed and printed these stops per day. The temporary test was removed after the measurement.

| City | 1 day | 2 days | 3 days | 4 days | 5 days |
| --- | --- | --- | --- | --- | --- |
| Lisbon | 4 | 4 4 | 4 4 4 | 4 3 4 3 | 4 3 4 3 2 |
| Kyoto | 4 | 4 4 | 4 3 4 | 4 2 4 3 | 4 3 4 2 3 |
| Tavira | 4 | 4 3 | 4 3 | 4 3 | 4 3 |

Every recorded day has at least 2 stops. Kyoto's 3-day plan gains a stop, as does one day in each of its 4- and 5-day plans.

## Guarantees

| What is guaranteed | Test name | Type | Result |
| --- | --- | --- | --- |
| Short days draw nearby places below rank 20 and top up two-stop days | `TestGroupDays_FillsAShortDayFromBeyondTheTop20`, `TestGroupDays_TopsUpTwoStopDaysNotOnlySingletons` | unit | PASS |
| A compact walk wins over a more famous distant stop, using the nearest stop on the day | `TestGroupDays_FillPrefersFamousPlacesWithinAShortWalk`, `TestGroupDays_FillMeasuresFromTheNearestStop` | unit | PASS |
| A lone day is rebuilt elsewhere, or dropped when no other area can fill it | `TestGroupDays_SwapsALoneStopForAnotherArea`, `TestGroupDays_DropsALoneDayWhenNoAreaCanFillIt` | unit | PASS |
| The most famous place and one-place towns remain planned | `TestGroupDays_KeepsAnIsolatedMostFamousPlace`, `TestGroupDays_OnePlaceGivesOneDay` | unit | PASS |
| Recorded city days have at least two stops; Kyoto's Arashiyama day includes Tenryū-ji | `TestCities_EveryDayHasAtLeastTwoStops`, `TestCities_KyotoArashiyamaDayIncludesTenryuji` | integration | PASS |
| An included day fills from below rank 20 while an isolated included place remains planned | `TestBuild_FillsAnIncludedDayFromBeyondTheTop20`, `TestBuild_PlansIncludedPlacesWhenNothingElseIsNearby` | integration | PASS |

## Coverage and gaps

`go test -cover ./internal/planner/...` gave `ok weatherservice/internal/planner 0.485s coverage: 96.7% of statements`.

At this stage, some days remained at two stops: Lisbon's 5-day plan, and Kyoto's 4- and 5-day plans. The nearest unused ranked place was 4.68 km from Lisbon's thin day and 3.63 km from Kyoto's, beyond the then-specified 3 km `PullReachKM` limit. The owner's 15 km follow-up below resolves these fixture cases. The one-place-town and isolated included-place exceptions still permit a single-stop day.

## Final verification

The first `make test lint build` run outside the sandbox passed the complete `go test -race -count=1 ./...` stage, including `internal/planner`, `cmd/design-preview` and the adapter packages. The sandbox run had been unable to access Go's build cache or bind local `httptest` listeners. The outside run stopped at lint because the installed `golangci-lint` v2.4.0 was built with Go 1.26:

```text
Error: can't load config: the Go language version (go1.26) used to build golangci-lint is lower than the targeted Go version (1.27.2)
make: *** [lint] Error 3
```

Rebuilding v2.4.0 with Go 1.27.2 passed the version check but could not decode Go 1.27 export data. A temporary `golangci-lint` v2.14.0 built with Go 1.27.2 was installed outside the repository in `/private/tmp/traveltab-tools`. Then `PATH=/private/tmp/traveltab-tools:$PATH make test lint build` exited 0: all packages passed the race-test stage, lint reported `0 issues.`, and `go build -o bin/traveltab ./cmd/web` succeeded.

Separately, `go vet ./...` passed with no output, and `gofmt -l internal/planner/days.go internal/planner/days_test.go internal/planner/build.go internal/planner/build_test.go internal/planner/cities_test.go` returned no files. The repository's installed linter remains v2.4.0; the compatible binary used for this gate is temporary.

## Follow-up: 15 km maximum walk per day

**Source:** owner's follow-up to this task: change the walking limit to 15 km, clarified as the **total walking loop for each day**. As a traveller, I want a fuller day whose complete walk, including the return, is no longer than 15 km.

### RED

`go test ./internal/planner/ -run '^TestGroupDays_FillUsesFifteenKilometreReach$' -count=1` failed with `expected: []string{"Q1", "Q2", "Q3"}` and `actual: []string{"Q1", "Q2"}` under the former 3 km pull limit. The test was renamed to `TestGroupDays_FillsBeyondThreeKilometresWithinTheDailyLimit` after the owner clarified that 15 km applies to the complete loop. Raising the reach alone made planner regressions fail: Kyoto's bamboo-grove day disappeared and several synthetic Lisbon days exceeded their prior 8 km expectation. The initial day-cluster balancing reach therefore remains 3 km; filling can search farther only when the complete loop fits.

The cap tests then failed before the cap was implemented. Command: `go test ./internal/planner/ -run 'Test(GroupDays_(FillKeepsTheWalkingLoopWithinFifteenKilometres|OneDayKeepsItsWalkWithinFifteenKilometres)|Cities_EveryDayWalksAtMostFifteenKilometres|Build_SplitsIncludedPlacesToKeepDaysWithinFifteenKilometres|Schedule_RainyTopUpKeepsTheWalkingLoopWithinFifteenKilometres)$' -count=1`.

```text
--- FAIL: TestBuild_SplitsIncludedPlacesToKeepDaysWithinFifteenKilometres (0.00s)
        Error:      "26.68678239469353" is not less than or equal to "15"
--- FAIL: TestGroupDays_OneDayKeepsItsWalkWithinFifteenKilometres (0.00s)
        Error:      "22.23898532891175" is not less than or equal to "15"
--- FAIL: TestSchedule_RainyTopUpKeepsTheWalkingLoopWithinFifteenKilometres (0.00s)
        Error:      "22.23898532891175" is not less than or equal to "15"
    --- FAIL: TestCities_EveryDayWalksAtMostFifteenKilometres/Lisbon_5_days (0.00s)
            Messages:   day 2 has a 16.3 km walk
    --- FAIL: TestCities_EveryDayWalksAtMostFifteenKilometres/Kyoto_4_days (0.00s)
            Messages:   day 2 has a 19.2 km walk
```

An existing saved-place test also failed after enforcing the cap: `Q950` could not fit with five places over two days and appeared in `AlsoSuggested`. Its assertions now cover that outcome.

### GREEN

The same focused cap-test command gave `ok weatherservice/internal/planner 0.657s`. `go test ./internal/planner/... -count=1` gave `ok weatherservice/internal/planner 0.749s`. `go test -cover ./internal/planner/...` gave `ok weatherservice/internal/planner 0.520s coverage: 97.0% of statements`.

The temporary fixture test `go test -v ./internal/planner/ -run '^TestZZReachCityScratch$' -count=1` passed and was removed. Stops per day after the cap:

| City | 1 day | 2 days | 3 days | 4 days | 5 days | Longest loop |
| --- | --- | --- | --- | --- | --- | --- |
| Lisbon | 4 | 4 4 | 4 4 4 | 4 3 4 3 | 4 3 4 3 3 | 14.6 km |
| Kyoto | 4 | 4 4 | 4 3 4 | 3 3 4 3 | 4 3 4 3 3 | 14.4 km |
| Tavira | 4 | 4 3 | 4 3 | 4 3 | 4 3 | 9.1 km |

| What is guaranteed | Test name | Type | Result |
| --- | --- | --- | --- |
| Fill can use a place beyond 3 km, but rejects one that makes the round trip exceed 15 km | `TestGroupDays_FillsBeyondThreeKilometresWithinTheDailyLimit`, `TestGroupDays_FillKeepsTheWalkingLoopWithinFifteenKilometres` | unit | PASS |
| An initial cluster longer than 15 km is trimmed while keeping its most famous stop | `TestGroupDays_OneDayKeepsItsWalkWithinFifteenKilometres` | unit | PASS |
| Saved places on an overlong day split across available days; a place without room is suggested | `TestBuild_SplitsIncludedPlacesToKeepDaysWithinFifteenKilometres`, `TestBuild_ADayOfIncludedPlacesHoldsAtMostFourStops` | integration | PASS |
| Rain swaps cannot extend the daily loop beyond 15 km | `TestSchedule_RainyTopUpKeepsTheWalkingLoopWithinFifteenKilometres` | unit | PASS |
| Every recorded Lisbon, Kyoto and Tavira day, for requested trips of 1–5 days, stays within 15 km | `TestCities_EveryDayWalksAtMostFifteenKilometres` | integration | PASS |

`PATH=/private/tmp/traveltab-tools:$PATH make test lint build` exited 0: the complete race-test suite passed, lint reported `0 issues.`, and `go build -o bin/traveltab ./cmd/web` succeeded.

**Limit:** an included place is returned in `AlsoSuggested` when the requested days cannot fit it within 15 km. A rainy day may have fewer sheltered stops when all unused sheltered candidates would break the cap.
