# Planner implementation notes

Task 1: contract copied from plan.md, formatted with gofmt. Geo tests passed with `CGO_ENABLED=1 go test -race -count=1 ./planner/`. Assertion RED against zero bodies:

```text
--- FAIL: TestDistanceKM_LisbonToPortoIsAbout274Km (0.00s)
--- FAIL: TestWalkingLoop_StartsAtTheMostFamousStop (0.00s)
--- FAIL: TestWalkingLoop_BreaksFameTiesByLowerID (0.00s)
--- FAIL: TestWalkingLoop_VisitsEveryStopOnce (0.00s)
--- FAIL: TestWalkingLoop_WalkKMIncludesTheWayBack (0.00s)
--- FAIL: TestWalkingLoop_SingleStopWalksZeroKM (0.00s)
--- FAIL: TestMedoid_PicksThePlaceClosestToAllOthers (0.00s)
--- FAIL: TestMedoid_BreaksTiesByLowerID (0.00s)
--- FAIL: TestImageURL_BuildsACommonsThumbnailURL (0.00s)
--- FAIL: TestImagePageURL_LinksToTheCommonsFilePage (0.00s)
```

The same-place distance and empty-image tests naturally pass against zero-value stubs; these are boundary guards, not independent RED evidence. All remaining Task 1 tests failed before implementation. Full original output: `/tmp/planner-task1-red.txt`.
