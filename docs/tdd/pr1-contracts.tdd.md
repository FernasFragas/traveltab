# PR 1 — Contracts, Data and Flags: TDD Evidence

**Source plan:** [`tasks/todo.md`](../../tasks/todo.md) PR 1, subtasks 1a–1d. Step 1e is not started.

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

The existing tests from 1c and 1d still pass: fixture shape, panel summary present or absent, cancellation, fixture not changed by the fake, fake summarizer, and flags default off or read as on.

**Full suite:** `make test lint build` passes, and golangci-lint reports `0 issues.`

## Coverage

`go test -cover ./internal/writermap/ ./internal/config/ ./writerdata/`:

| Package | Before | After |
| --- | ---: | ---: |
| `internal/writermap` | 90.9% | 97.6% |
| `internal/config` | 87.0% | 100.0% |
| `writerdata` | no statements | no statements; 8 data tests |

`writermap` dropped from 98.2% to 97.6% when `cloneResult` lost its copy loops: fewer statements, and the uncovered branch is the same. In `config`, the `.env` test now covers the development branch too.

## Known gaps

- **No recorded RED for the original 1a–1d.** That evidence can't be recovered.
- **Branch mixes PR 1 and PR 2.** Splitting it needs commits, which agents don't make (`AGENTS.md` §6). The owner decides.
- **Step 1e** (fields for the mockup) is open. It changes the contract and needs its own RED → GREEN cycle.
