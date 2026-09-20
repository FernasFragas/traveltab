# Notes: Task 4 (Open-Meteo forecast) and Task 5 (Overpass stays)

Written while building [Task 4](todo.md#task-4-open-meteo-daily-forecast-source) and
[Task 5](todo.md#task-5-overpass-stay-source-with-fallback-servers). Both followed the
RED → GREEN → REFACTOR loop from [plan.md](plan.md#how-every-task-is-built-tdd).

Focused command used throughout:

```
CGO_ENABLED=1 go test -race -count=1 -run 'Forecast|Overpass' ./api/
```

---

## Task 4: Open-Meteo daily forecast source

### Files

- `api/openmeteo_forecast.go`
- `api/openmeteo_forecast_test.go`
- `api/testdata/openmeteo/forecast_kyoto.json` (already present, unchanged)

### Tests (9, all from todo.md)

| Test | Size | Status |
|---|---|---|
| `TestForecast_ReturnsSixteenDays` | small | pass |
| `TestForecast_ReturnsTheCityTimeZone` | small | pass |
| `TestForecast_DatesAreMidnightInTheCityTimeZone` | small | pass |
| `TestForecast_KeepsTheRainAmount` | small | pass |
| `TestForecast_NullRainBecomesNil` | small | pass |
| `TestForecast_ErrorsOnNon200` | small | pass |
| `TestForecast_ErrorsOnAnUnknownTimeZone` | small | pass |
| `TestForecast_AsksForDailyRainInTheLocalTimeZone` | small | pass |
| `TestForecastLive_ReturnsKyoto` | large, opt-in | pass with `LIVE_API_TESTS=1` |

No test was added or dropped: the list matches todo.md exactly. The tests were rewritten
from a one-line-per-test scaffold into the repo's Arrange/Act/Assert style before the
implementation was written, and RED was re-confirmed afterwards (output below).

### RED evidence

Against the zero-value stub (`DailyForecast` returning `nil, nil`). Every failure is an
assertion, not a compile error.

```
--- FAIL: TestForecast_ReturnsSixteenDays (0.00s)
    openmeteo_forecast_test.go:46: Error: Expected value not to be nil.
--- FAIL: TestForecast_ReturnsTheCityTimeZone (0.00s)
    openmeteo_forecast_test.go:56: Error: Expected value not to be nil.
--- FAIL: TestForecast_DatesAreMidnightInTheCityTimeZone (0.00s)
    openmeteo_forecast_test.go:66: Error: Expected value not to be nil.
--- FAIL: TestForecast_KeepsTheRainAmount (0.00s)
    openmeteo_forecast_test.go:77: Error: Expected value not to be nil.
--- FAIL: TestForecast_NullRainBecomesNil (0.00s)
    openmeteo_forecast_test.go:89: Error: Expected value not to be nil.
--- FAIL: TestForecast_ErrorsOnNon200 (0.00s)
    openmeteo_forecast_test.go:99: Error: An error is expected but got nil.
--- FAIL: TestForecast_ErrorsOnAnUnknownTimeZone (0.00s)
    openmeteo_forecast_test.go:107: Error: An error is expected but got nil.
--- FAIL: TestForecast_AsksForDailyRainInTheLocalTimeZone (0.00s)
    openmeteo_forecast_test.go:116: Error: "[]" should have 1 item(s), but has 0
FAIL
FAIL	weatherservice/api	0.394s
```

### What the implementation does

- `GET https://api.open-meteo.com/v1/forecast` with `latitude`, `longitude`,
  `daily=precipitation_sum`, `timezone=auto` and `forecast_days=16`
  (`planner.ForecastMaxDays`), plus the shared `User-Agent`.
- Decodes `precipitation_sum` into `[]*float64`, so Open-Meteo's `null` becomes a `nil`
  `RainMM` rather than `0`. Each kept value is copied into a fresh variable, so no two
  days share a pointer.
- `time/tzdata` is imported blank, so `time.LoadLocation` resolves IANA names inside the
  slim Docker image, which ships no system time zone database.
- Dates are parsed with `time.ParseInLocation`, giving midnight in the city's own zone.
- Non-200 and unknown time zones are errors. All errors are wrapped with `%w`.
- `NewOpenMeteoForecastAPI(nil)` gives a default client with a 30 s timeout, matching the
  Wikimedia client's default.
- The existing marine client, `api/openmateo.go`, was not touched.

---

## Task 5: Overpass stay source with fallback servers

### Files

- `api/overpass.go`
- `api/overpass_test.go` (new)
- `api/testdata/overpass/tavira.json` (already present, unchanged)

### Tests (13, all from todo.md)

| Test | Size | Status |
|---|---|---|
| `TestOverpass_ReturnsStaysFromTheFirstServer` | small | pass |
| `TestOverpass_RetriesOnceBeforeMovingOn` | small | pass |
| `TestOverpass_FallsBackAfterTwoFailures` | small | pass |
| `TestOverpass_ErrorNamesEveryFailedServer` | small | pass |
| `TestOverpass_WaitsForRetryAfterUpToFiveSeconds` | small | pass |
| `TestOverpass_SkipsUnnamedPlaces` | small | pass |
| `TestOverpass_PrefersTheEnglishName` | small | pass |
| `TestOverpass_ReadsTheWebsiteFromContactWebsite` | small | pass |
| `TestOverpass_ReadsStarsLike4SAsFour` | small | pass |
| `TestOverpass_StarsAreZeroWhenMissing` | small | pass |
| `TestOverpass_UsesTheWayCenterForCoordinates` | small | pass |
| `TestOverpass_SendsTheQueryAndUserAgent` | small | pass |
| `TestOverpassLive_FindsHotelsInTavira` | large, opt-in | pass with `LIVE_API_TESTS=1` |

### RED evidence

Against the compiling stub (`StaysNear` returning `nil, nil`). Every failure is an
assertion.

```
--- FAIL: TestOverpass_ReturnsStaysFromTheFirstServer (0.00s)
    overpass_test.go:126: Error: "[]" should have 2 item(s), but has 0
--- FAIL: TestOverpass_RetriesOnceBeforeMovingOn (0.00s)
    overpass_test.go:144: Error: "[]" should have 2 item(s), but has 0
--- FAIL: TestOverpass_FallsBackAfterTwoFailures (0.00s)
    overpass_test.go:163: Error: "[]" should have 1 item(s), but has 0
--- FAIL: TestOverpass_ErrorNamesEveryFailedServer (0.00s)
    overpass_test.go:183: Error: An error is expected but got nil.
--- FAIL: TestOverpass_WaitsForRetryAfterUpToFiveSeconds (0.00s)
    overpass_test.go:199: Error: "[]" should have 2 item(s), but has 0
--- FAIL: TestOverpass_SkipsUnnamedPlaces (0.00s)
    overpass_test.go:206: Error: "[]" should have 2 item(s), but has 0
            Messages: the unnamed hostel must be skipped
--- FAIL: TestOverpass_PrefersTheEnglishName (0.00s)
    overpass_test.go:216: Error: Should NOT be empty, but was []
--- FAIL: TestOverpass_ReadsTheWebsiteFromContactWebsite (0.00s)
    overpass_test.go:223: Error: Should NOT be empty, but was []
--- FAIL: TestOverpass_ReadsStarsLike4SAsFour (0.00s)
    overpass_test.go:230: Error: Should NOT be empty, but was []
--- FAIL: TestOverpass_StarsAreZeroWhenMissing (0.00s)
    overpass_test.go:237: Error: "[]" should have 2 item(s), but has 0
--- FAIL: TestOverpass_UsesTheWayCenterForCoordinates (0.00s)
    overpass_test.go:245: Error: "[]" should have 2 item(s), but has 0
--- FAIL: TestOverpass_SendsTheQueryAndUserAgent (0.00s)
    overpass_test.go:259: Error: "[]" should have 1 item(s), but has 0
FAIL
FAIL	weatherservice/api	0.406s
```

### Retry and fallback order

For each server in order, at most two attempts, then the next server:

```
server 1  attempt 1  ──ok──▶ done
             │ 429 / 5xx / timeout / transport error
             ▼  sleep(min(Retry-After, 5s))   ← injected sleeper, 0 means no sleep
server 1  attempt 2  ──ok──▶ done
             │ fail
             ▼
server 2  attempt 1 … attempt 2
             │ fail
             ▼
error: "overpass: every server failed: <url>: …; <url>: …"   (errors.Join, %w)
```

- `sleep func(time.Duration)` is a field on `OverpassAPI`, set to `time.Sleep` by the
  constructor and replaced by the tests' recorder. **No test ever really sleeps**, and
  `TestOverpass_WaitsForRetryAfterUpToFiveSeconds` asserts the recorded duration is
  exactly `5s` for a `Retry-After: 30` header.
- `Retry-After` is read as seconds and capped at 5 s. Anything unparseable, zero or
  negative means no wait, so the retry is immediate. No extra backoff was invented,
  because the task doesn't ask for one.
- Every attempt gets its own 10 s `context.WithTimeout`, so a hung server can't eat the
  whole request budget.
- Non-retryable statuses (for example 400) still consume the server's two attempts, which
  is slightly wasteful but keeps the loop to a single rule. Worth revisiting only if the
  wasted call ever shows up in production.

### Parsing

`stayFromTags` is the only place tags become a `planner.Stay`:

- Name: `name:en` first, then `name`. **No name means the place is dropped** — the card
  has nothing to show for it. In the Tavira fixture, this removes the unnamed hostel.
- Website: `website` first, then `contact:website`.
- Stars: the leading digits of the `stars` tag, so `"4S"` and `"4"` both give `4`, and a
  missing or unreadable tag gives `0`.
- Coordinates: `lat`/`lon` for nodes, `center` for ways and relations.

---

## Live tests

Both opt-in live tests were run for real, once, and passed:

```
$ LIVE_API_TESTS=1 CGO_ENABLED=1 go test -count=1 -v -run 'ForecastLive|OverpassLive' ./api/
--- PASS: TestForecastLive_ReturnsKyoto (0.27s)
--- PASS: TestOverpassLive_FindsHotelsInTavira (0.39s)
ok  	weatherservice/api	1.044s
```

They skip when `LIVE_API_TESTS` is unset, so the normal suite stays offline.

---

## One deviation from the spec, for review

**Task 5's query needs an `[out:json];` prefix.** todo.md gives the query as:

```
nwr["tourism"~"^(hotel|hostel|guest_house)$"](around:1000,LAT,LON); out center tags;
```

Overpass defaults to **XML** output. Without a leading `[out:json];` setting the server
answers with XML, which the JSON decoder cannot read, so the client would fail against
every real server. There is no header or form field that changes this; the setting has to
be inside the query. The client therefore sends:

```
[out:json];nwr["tourism"~"^(hotel|hostel|guest_house)$"](around:1000,37.1251,-7.6501); out center tags;
```

Everything after the prefix is byte-for-byte the spec's query, with `1000` coming from
`planner.StayRadiusM`. `TestOverpass_SendsTheQueryAndUserAgent` asserts this exact string,
and `TestOverpassLive_FindsHotelsInTavira` confirms the real server accepts it and returns
named hotels. **This is a spec wording fix, not a contract change** — no `planner` type,
interface or exported signature moved.

---

## Contract check

`planner/types.go` and `api/useragent.go` were read, not edited. The exported surface
matches plan.md:

- `func NewOpenMeteoForecastAPI(client *http.Client) *OpenMeteoForecastAPI`
- `func NewOverpassAPI(client *http.Client, urls []string) *OverpassAPI`
- `var DefaultOverpassURLs []string`

Both types satisfy their interfaces (`planner.ForecastSource`, `planner.StaySource`).

## Lint

`gofmt -l` is clean on all four owned files, and `golangci-lint run ./api/` reports
**0 issues**. `make lint` currently fails on `api/wikimedia.go` needing gofmt — that file
belongs to Task 3's lane and was left alone.
