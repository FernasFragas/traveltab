# PR 1 — Contracts, Data and Flags: TDD Evidence

**Source plan:** [`tasks/todo.md`](../../tasks/todo.md) PR 1, subtasks 1a–1d. Step 1e is in [its own report](1e-mockup-fields.tdd.md).

**Status:** 1a–1d were written before the TDD rules in [`AGENTS.md`](../../AGENTS.md). Their first RED runs were not recorded. This report covers the follow-up from the October 7, 2026 review: it fixes one test that could not fail, decides the not-found rule, and adds the missing tests. Every test that passed on its first run was checked by breaking its behaviour temporarily.

## User journeys

- As a **map developer**, I want a fake writer map source with fixed Madeira data and slow, partial, unavailable and empty modes, so that I can build and test the map UI without the real index.
- As a **map developer**, I want asking for an unknown place to return a clear "not found" error, so that a panel request can't crash on a missing place.
- As the **owner**, I want the writer source list and overrides checked against the maintenance rules, so that a bad edit fails CI instead of reaching the monthly sync.
- As the **owner**, I want the writers' flags off unless explicitly set to on, so that nothing launches by accident.

## Task report

### Fix the `partial` test (1c)
- **Summary:** the test used a fixture that was already `complete: false`, so it passed even with the `Partial` switch ignored. It now starts from a complete copy.
- **RED** (switch disabled with `if false && f.Partial`), `go test ./internal/writermap/ -run 'TestFakeSourceSwitches/partial'`:
  ```
  --- FAIL: TestFakeSourceSwitches/partial (0.00s)
          Error:      Should be false
  ```
- **GREEN** (real code): `ok  weatherservice/internal/writermap`
- Also removed an assertion that checked the test's own fixture copy rather than the fake.

### Not-found rule for `Place` (1b, 1c)
- **Decision:** an unknown QID, and any QID on an empty destination, returns an error wrapping `writermap.ErrPlaceNotFound`. Before, the empty case returned `nil, nil, nil` and the unknown case returned an unwrapped error. Recorded in [`docs/writers-map-contract.md`](../writers-map-contract.md).
- **RED (compile)**, `go test ./internal/writermap/`:
  ```
  internal/writermap/fake_test.go:99:25: undefined: ErrPlaceNotFound
  FAIL    weatherservice/internal/writermap [build failed]
  ```
- **RED (runtime)**, with the sentinel declared but `Place` unchanged:
  ```
  --- FAIL: TestFakeSourcePlace_UnknownQIDReturnsNotFound (0.00s)
          Error:      Target error should be in err chain:
  --- FAIL: TestFakeSourcePlace_EmptyDestinationReturnsNotFound (0.00s)
          Error:      Target error should be in err chain:
  ```
- **GREEN:** `ok  weatherservice/internal/writermap`
- **Refactor:** moved `ErrPlaceNotFound` from `fake.go` to `types.go` (since renamed `destinationdata.go`), because real sources will return it too. Tests still pass.

### Unavailable and delay paths (1c)
- These tests passed on their first run, because the behaviour already existed.
- **Can fail:** with the `Unavailable` check removed from `Place`, `TestFakeSourcePlace_UnavailableReturnsError` fails. With the timer path returning `context.DeadlineExceeded`, `TestFakeSource_DelayElapsesThenReturnsData` fails.

### Source data and overrides (1a)
- **Summary:** new `writerdata/embed_test.go` reads the embedded files and checks the rules in [`docs/maintenance.md`](../maintenance.md).
- These tests passed on their first run, because the committed data is valid.
- **Can fail:** a broken copy of the data (status `paused`, language `fr`, date `5 Oct`, `/wp/v2/pages`, a duplicate host, `radius_km: 0`, hide QID `Pico`) failed all seven rule tests. Renaming `stoplist` to `stop_list` failed `TestOverrides_DecodeWithKnownFieldsOnly` with `json: unknown field "stop_list"`. The original files were then restored.

### Flags stay off (1d)
- This test passed on its first run.
- **Can fail:** with `envBool` changed to treat any non-empty value as on, all six subtests of the stay-off test (now `TestLoadFeatureFlags_WritersFlagsStayOffForOtherValues`) failed. `env.go` was then restored unchanged.

### Move the flags out of `WeatherServiceKeys` (1d)
- **Summary:** `WritersMap`, `WritersSync` and `WritersAI` are switches, not keys, so they moved to a new `config.FeatureFlags`, loaded by `LoadFeatureFlags()` in `internal/config/flags.go`. `LLMGatewayURL` and `LLMGatewayKey` stay in `WeatherServiceKeys`: they are a credential and its endpoint. Loading `.env` moved into `loadDotEnv()`, which both loaders call.
- **RED (compile)**, `go test ./internal/config/`:
  ```
  internal/config/flags_test.go:26:11: undefined: LoadFeatureFlags
  FAIL    weatherservice/internal/config [build failed]
  ```
- **RED (runtime)**, with an empty `FeatureFlags` type and a loader that returned it:
  ```
  --- FAIL: TestWeatherServiceKeys_HoldsNoFeatureFlags (0.00s)
          Messages:   feature flag WritersMap belongs in FeatureFlags
  --- FAIL: TestLoadFeatureFlags_ReadsWritersFlags (0.00s)
  ```
- **GREEN:** `ok  weatherservice/internal/config  coverage: 87.0% of statements`

### Second review fixes
- **Flags accept only `"1"`.** This matches the docs (`WRITERS_SYNC=1`) and the repo's `RECORD_FIXTURES == "1"`. The parser moved from `env.go` to `flags.go`, its only user, and was renamed `envFlag`.
  - **RED**, `go test ./internal/config/`: `--- FAIL: TestLoadFeatureFlags_WritersFlagsStayOffForOtherValues/true`, and the same for `/yes` and `/on`.
  - **GREEN:** `ok  weatherservice/internal/config`
- **`.env` is loaded once.** `loadDotEnv` uses `sync.Once`, so a missing file is logged once, not once per loader.
  - **RED (compile):** `undefined: dotEnvOnce`.
  - **RED (runtime)**, with the variable declared but unused: `TestLoaders_LogAMissingDotEnvOnlyOnce`, `expected: 1`, `actual: 2`.
  - **GREEN:** `ok  weatherservice/internal/config  coverage: 100.0% of statements`
- **`cloneResult` trimmed.** It now copies only places and their names; the other fields are shared and read-only, as its comment says. This was a refactor with tests green before and after. A new test, `TestFakeSource_ReplacingAResultPlaceLeavesTheFakeUnchanged`, passed on its first run. With the places copy removed, both copy tests failed. With the names copy removed, `TestFakeSource_EditingAResultNameLeavesTheFakeUnchanged` failed.
- **`Passage` lost its JSON tags**, so passages aren't easy to serialize by accident. Nothing marshalled them.
- **Tests split and reordered.** The fixture, panel and summarizer tests that checked several things are now one test per behaviour. Every earlier assertion is kept, except `Calls == 2`: each summarizer test now has its own summarizer and checks `Calls == 1`. `fake_test.go` and `writerdata/embed_test.go` now go top-down, tests first and helpers last. `embed_test.go` was a pure reorder, with the same lines when sorted.
- **Spike script.** `run.py` now checks `robots.txt` before each host: 404 allows everything, a disallow skips the host, and 401/403/429 stop the run. Four offline cases passed (disallow all, disallow `/wp-json/`, allow, no file) with the network call replaced. The script was not rerun.

### Fake moved to its own package
- **Summary:** `FakeSource`, `FakeSummarizer` and the Madeira fixture moved from `internal/writermap` to `internal/writermap/writermaptest`, following the `net/http/httptest` pattern, so callers write `writermaptest.FakeSource`. Not `testdata/`: the Go tool ignores `.go` files there. Refactor only, no behaviour change.
- **Before and after:** `go test ./internal/writermap/... -v` showed 21 passing tests and subtests both times. Afterwards: `ok  weatherservice/internal/writermap/writermaptest`.

### Compile-time port check (1c)
- **Summary:** `writermaptest/fake.go` now asserts `_ application.WriterMapSource = (*FakeSource)(nil)` and `_ application.Summarizer = (*FakeSummarizer)(nil)`, so a fake that drifts from its port fails the build.
- **Can fail:** with the `area` argument removed from `FakeSource.Place`, `go build` failed: `*FakeSource does not implement application.WriterMapSource (wrong type for method Place)`. With the passages argument removed from `Summarize`, it failed the same way for `application.Summarizer`. The file was then restored, and `make test lint build` passed.

### Third review fixes
- **Empty result shares its source statuses.** This matches `cloneResult`'s rule that only places are copied, and the one 200-character line is split. A new test, `TestFakeSource_EmptyDestinationKeepsSourceStatuses`, passed before and after the change and failed with the statuses line removed.
- **`fake.go` is top-down:** `FakeSource`, then `FakeSummarizer`, then the helpers `wait` and `cloneResult`. A pure reorder, with the same lines when sorted.
- **The `.env` test restores the previous log output**, not `os.Stderr`, and a comment says it must not run in parallel.
- **Task docs no longer tell agents to merge, open PRs or rebase** (`tasks/todo.md` 1d, `tasks/plan.md` Rules for Agents). The owner does those, per `AGENTS.md` §6.
- `make test lint build` passed. Coverage: `config` 100.0%, `writermaptest` 97.6%.

### Fourth review fixes
- **Fixture QIDs were all wrong.** The 15 IDs in `madeira.json` belonged to unrelated Wikidata items (Q799 "wisdom", Q190136 "dendrite", Q206626 "2009–10 UEFA Europa League", …). Each one was replaced with the item found by a Wikidata name search and confirmed by its description and its coordinates inside Madeira. For towns, the settlement or city-seat item was preferred where one exists. Earlier sections of this report quote the old IDs as history.
  - **Guard:** `TestMadeiraFixture_PlacesMatchWikidata` (`writermaptest/wikidata_test.go`) checks each place against `testdata/wikidata-places.json`: the English label must be one of its names, and its coordinates must be within 0.05°. `RECORD_FIXTURES=1` refreshes the file from the live API, the same convention as `internal/planner`. `make test` stays offline.
  - **RED**, `RECORD_FIXTURES=1 go test ./internal/writermap/writermaptest/ -run PlacesMatchWikidata` with the old IDs:
    ```
    Error:  []string{"Madeira", "Madeira Island"} does not contain "wisdom"
    Error:  []string{"Funchal"} does not contain "dendrite"
    ```
  - **GREEN:** the new IDs re-recorded, then `ok  weatherservice/internal/writermap/writermaptest` both online and offline. "25 Fontes Falls", Wikidata's English label, was added to that place's names.
- **`SourceStatus.CheckedAt` is `*time.Time`.** `omitempty` can't leave out a zero `time.Time` on Go 1.23.
  - **RED:** `TestSourceStatus_NeverCheckedLeavesOutCheckedAt`, actual `"checked_at":"0001-01-01T00:00:00Z"`.
  - **GREEN:** `ok  weatherservice/internal/writermap`.
- **Source statuses are defined.** The contract lists `active`, `partial`, `blocked` and `removed`, with `partial` only appearing at runtime. `TestMadeiraFixture_SourceStatusesUseContractValues` passed on its first run, because the fixture already complied, and failed with `paused`.
- `make test lint build` passed after one errcheck fix in the recorder (`defer func() { _ = response.Body.Close() }()`, matching the repo).

## Test specification

| # | What is guaranteed | Test | Type | Result |
| --- | --- | --- | --- | --- |
| 1 | `Partial` marks a complete result incomplete | `TestFakeSourceSwitches/partial` | unit | PASS |
| 2 | Unknown QID returns `ErrPlaceNotFound`, with nil place and nil summary | `TestFakeSourcePlace_UnknownQIDReturnsNotFound` | unit | PASS |
| 3 | Empty destination returns `ErrPlaceNotFound` for any QID | `TestFakeSourcePlace_EmptyDestinationReturnsNotFound` | unit | PASS |
| 4 | Unavailable source returns `ErrFakeUnavailable` from `Place` | `TestFakeSourcePlace_UnavailableReturnsError` | unit | PASS |
| 5 | A slow source returns data once the delay passes | `TestFakeSource_DelayElapsesThenReturnsData` | unit | PASS |
| 6 | Source status is `active`, `blocked` or `removed` | `TestSources_StatusIsActiveBlockedOrRemoved` | unit | PASS |
| 7 | API base is `https://<host>/wp-json/wp/v2/posts` | `TestSources_APIBaseIsTheHostsWordPressPostsEndpoint` | unit | PASS |
| 8 | Source language is `en` or `pt` | `TestSources_LanguageIsEnglishOrPortuguese` | unit | PASS |
| 9 | Source check date is `YYYY-MM-DD` | `TestSources_CheckedAtIsADate` | unit | PASS |
| 10 | Each host appears once | `TestSources_HostsAreUnique` | unit | PASS |
| 11 | Overrides contain only known keys | `TestOverrides_DecodeWithKnownFieldsOnly` | unit | PASS |
| 12 | Every area has a positive radius | `TestOverrides_AreasHaveAPositiveRadius` | unit | PASS |
| 13 | Summary hide list holds only Wikidata QIDs | `TestOverrides_SummaryHideListHoldsWikidataQIDs` | unit | PASS |
| 14 | Writers' flags are off for anything but `"1"`: `true`, `yes`, `on`, `0`, `false`, unknown values and blank | `TestLoadFeatureFlags_WritersFlagsStayOffForOtherValues` | unit | PASS |
| 15 | Flags are read into `FeatureFlags`: off by default, on for `1` | `TestLoadFeatureFlags_WritersFlagsDefaultOff`, `TestLoadFeatureFlags_ReadsWritersFlags` | unit | PASS |
| 16 | `WeatherServiceKeys` holds no feature flags | `TestWeatherServiceKeys_HoldsNoFeatureFlags` | unit | PASS |
| 17 | A missing `.env` is logged once when both loaders run | `TestLoaders_LogAMissingDotEnvOnlyOnce` | unit | PASS |
| 18 | Editing or replacing a result place leaves the fake unchanged | `TestFakeSource_EditingAResultNameLeavesTheFakeUnchanged`, `TestFakeSource_ReplacingAResultPlaceLeavesTheFakeUnchanged` | unit | PASS |
| 19 | The fakes implement `WriterMapSource` and `Summarizer` | compile-time assertions in `writermaptest/fake.go` | build | PASS |
| 20 | An empty destination still reports source statuses | `TestFakeSource_EmptyDestinationKeepsSourceStatuses` | unit | PASS |
| 21 | Every fixture QID is the Wikidata item it names, at its real location | `TestMadeiraFixture_PlacesMatchWikidata` | unit (recorded) | PASS |
| 22 | A never-checked source leaves out `checked_at`; a checked one encodes it | `TestSourceStatus_NeverCheckedLeavesOutCheckedAt`, `TestSourceStatus_CheckedEncodesCheckedAt` | unit | PASS |
| 23 | Fixture source statuses use only contract values | `TestMadeiraFixture_SourceStatusesUseContractValues` | unit | PASS |

The existing tests from 1c and 1d still pass: fixture shape, panel summary present or absent, cancellation, fixture not changed by the fake, fake summarizer, and flags default off or read as on.

**Full suite:** `make test lint build` passes, and golangci-lint reports `0 issues.`

## Coverage

`go test -cover ./internal/writermap/ ./internal/config/ ./writerdata/`:

| Package | Before | After |
| --- | ---: | ---: |
| `internal/writermap` (fake now in `writermaptest`) | 90.9% | 97.6% |
| `internal/config` | 87.0% | 100.0% |
| `writerdata` | no statements | no statements; 8 data tests |

`writermap` dropped from 98.2% to 97.6% when `cloneResult` lost its copy loops: fewer statements, and the uncovered branch is the same. In `config`, the `.env` test now covers the development branch too.

## Known gaps

- **No recorded RED for the original 1a–1d.** That evidence can't be recovered.
- **Branch mixes PR 1 and PR 2.** Splitting it needs commits, which agents don't make (`AGENTS.md` §6). The owner decides.
- **Fixture sitelink counts are illustrative**, not recorded from Wikidata. Nothing ranks by them yet; 4a should record real counts if it uses this fixture.
- **Step 1e** (fields for the mockup) is open. It changes the contract and needs its own RED → GREEN cycle.
