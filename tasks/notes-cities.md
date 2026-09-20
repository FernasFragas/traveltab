# City acceptance test notes (Task 16)

The agent that wrote `planner/cities_test.go` was cut off by a session limit before it filed its notes. This file was written afterwards by the coordinating session, from what is verifiable on disk. **It is not the original RED evidence**, which was never saved.

## What exists

- `planner/cities_test.go` (418 lines, package `planner_test`), with the 7 tests the spec asks for:
  `TestCities_LisbonIncludesBelemTowerAndJeronimos`, `TestCities_LisbonSkipsBridgesAndTheNationalLibrary`,
  `TestCities_LisbonRainyDayIsMostlyIndoor`, `TestCities_LisbonHasAtMostSixNamedStays`,
  `TestCities_TaviraFiveDaysGetsFewerDaysAndANote`, `TestCities_KyotoRainyDayHasThreeIndoorStops`,
  `TestCities_KyotoHasNoWards`.
- `planner/testdata/cities/{lisbon,tavira,kyoto}/` — 692 KB of recorded responses (geosearch, pageprops, sitelinks, entities, forecast, Overpass). Well under the 2 MB the spec allows.
- All 7 pass offline: `CGO_ENABLED=1 go test -race -count=1 -run Cities ./planner/`.

## Evidence that the tests bite (mutation check, run 2026-09-20)

`PlaceTypes` was temporarily emptied in `planner/placetypes_gen.go` and the tests re-run:

```text
--- FAIL: TestCities_LisbonIncludesBelemTowerAndJeronimos (0.01s)
--- FAIL: TestCities_LisbonRainyDayIsMostlyIndoor (0.00s)
--- FAIL: TestCities_TaviraFiveDaysGetsFewerDaysAndANote (0.00s)
--- FAIL: TestCities_KyotoRainyDayHasThreeIndoorStops (0.01s)
```

The type list was restored immediately and the suite is green again.

**The other three tests pass with an empty type list**, because they assert that something is *absent* (bridges, the National Library, Kyoto's wards) or that a count is at most six. With no places at all, those hold trivially. They are guard tests, in the same sense Task 1 and Task 8 used the term: useful against regressions, but not independent evidence on their own. Each would be stronger with a positive control in the same test ("the plan has stops, and none of them is a bridge").

## Completion pass (2026-09-20)

The three previously vacuous tests now require a full three-day plan and known-good landmarks.
A temporary Go overlay emptied `PlaceTypes` without modifying repository files. All three failed
on the new positive controls (one day returned instead of three):

- `TestCities_LisbonSkipsBridgesAndTheNationalLibrary`
- `TestCities_LisbonHasAtMostSixNamedStays`
- `TestCities_KyotoHasNoWards`

The real generated map passes the city suite. A new Kyoto regression first failed because its
Arashiyama day had one stop and omitted Q23579173. Grouping now fills such singletons from unused
nearby top-20 candidates, so the bamboo grove joins Arashiyama and the regression passes.
The earlier empty-map findings above describe the original tests, not the strengthened version.

Fixture limitation: Kyoto stays were recorded around an older city-center coordinate. An empty
stay result in that fixture does not establish live hotel availability at the computed medoid.
The separate stay-source, radius and fallback tests cover those behaviors.
