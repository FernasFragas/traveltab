# PR 1 — Contract Gaps Before Review: TDD Evidence

**Source plan:** [`tasks/todo.md`](../../tasks/todo.md) PR 1, follow-up agreed on October 8, 2026, before the owner review freezes the contract. It closes three gaps that would have blocked PR 3 and PR 4.

**Status:** done. Only the base-place ports have code; the `:dest` rules and the panel template owner are contract and task-list changes, tested later by 3b.

## User journeys

- As a **map developer** (PR 4), I want a base-place interface and a fake that counts calls, so that the Wikidata adapter, its cache and suggestion warming can be built and tested separately.
- As the **monthly job** (5d), I want to refresh a destination's base places through an interface, so that the job doesn't depend on the SQLite adapter.
- As a **map developer** (PR 3), I want map URLs to carry the slug and coordinates, so that serving the map never needs a geocoding call.

## Decisions

- **`:dest`** is the `/trip/:slug` slug plus `?lat=…&lon=…`. A bad slug returns 404; missing or invalid coordinates return 400 with `/plan`'s rules. Recorded under **Destination in map URLs** in [the contract](../writers-map-contract.md).
- **Base-place ports:** `application.BasePlaceSource` (`Places`) and `application.BasePlaceRefresher` (`Refresh`), two interfaces rather than one, because the Wikidata adapter (4a) has nothing to refresh; only the cache decorator (4b) does.
- **Panel template:** 3b creates a minimal `views/place_panel.go.tpl`; 7a builds it out.
- Also noted in the tasks: 3d checks the satellite imagery terms first, and 4a reuses `planner.PlaceTypes` for visitable classes.

## Task report

### Fake base-place source
- **Summary:** `writermaptest.FakeBaseSource` implements both ports, returns copies of configured places or an error, and counts `Places` and `Refresh` calls under a mutex.
- **RED (compile)**, `go vet ./internal/writermap/writermaptest/`:
  ```
  vet: internal/writermap/writermaptest/fake_test.go:159:13: undefined: FakeBaseSource
  ```
- **GREEN:** `ok  	weatherservice/internal/writermap/writermaptest	0.488s`
- **Refactor:** the place-copying loop in `cloneResult` moved into `clonePlaces`, shared by both fakes. Tests still pass.
- **Can fail** (each change made alone, then reverted), `go test ./internal/writermap/writermaptest/ -run …`:
  - returning `f.Result` uncopied → `--- FAIL: TestFakeBaseSource_EditingAResultLeavesTheFakeUnchanged`
  - removing `f.placesCalls++` → `--- FAIL: TestFakeBaseSource_PlacesReturnsTheConfiguredPlacesAndCountsCalls`
  - removing `f.refreshCalls++` → `--- FAIL: TestFakeBaseSource_RefreshCountsCalls`

## Guarantees

| What is guaranteed | Test | Type | Result |
| --- | --- | --- | --- |
| The fake implements `BasePlaceSource` and `BasePlaceRefresher` | compile-time assertions in `fake.go` | unit | PASS (builds) |
| `Places` returns the configured places and counts the call | `TestFakeBaseSource_PlacesReturnsTheConfiguredPlacesAndCountsCalls` | unit | PASS |
| Editing a returned place leaves the fake unchanged | `TestFakeBaseSource_EditingAResultLeavesTheFakeUnchanged` | unit | PASS |
| A configured error returns an empty list and the error | `TestFakeBaseSource_ReturnsConfiguredError` | unit | PASS |
| `Refresh` counts the call | `TestFakeBaseSource_RefreshCountsCalls` | unit | PASS |

## Coverage and full suite

`go test -cover ./internal/writermap/... ./internal/application/`:
```
ok  	weatherservice/internal/writermap	(cached)	coverage: [no statements]
ok  	weatherservice/internal/writermap/writermaptest	0.493s	coverage: 98.3% of statements
ok  	weatherservice/internal/application	(cached)	coverage: 44.2% of statements
```
`internal/application` was already below 80%; this change adds only interface declarations there, with no statements.

`make test lint build`: all packages `ok`, `golangci-lint` reported `0 issues.`, and the build succeeded.

## Known gaps

- The `:dest` 404/400 rules have no test until 3b adds the routes.
- Whether a free satellite imagery source fits the terms is still open (3d).
