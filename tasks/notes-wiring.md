# Task 15 notes — wire the real planner into the app

**Current status:** final race tests, lint, build, native ARM Docker and mobile browser checks pass.
The full route also survives all three providers failing with fresh or expired caches. See
[completion evidence](notes-completion.md). The runs and limitations below are historical.

Lane: wiring. Files owned: `tripplanner.go`, `tripplanner_test.go`, `env.go`, `env_test.go`,
`cmd/web/main.go`, this file.

## What was built

- **`tripplanner.go`** — `NewTripPlanner(places, forecast, stays)` returns a `TripPlanner` that
  calls `planner.Build` with `time.Now()`. It is the only thing between the `/plan` route and the
  planning chain, and it keeps the root package free of `weatherservice/api`.
- **`env.go`** — `WeatherServiceKeys.OverpassURLs []string`, read from `OVERPASS_URLS`
  (comma-separated). `splitList` trims each entry and drops empty ones, so an unset or empty
  variable gives an empty slice rather than `[""]`.
- **`cmd/web/main.go`** — builds the three API clients, wraps each with the Task 6 cache, and calls
  `server.SetTripPlanner(...)`. `overpassURLs` falls back to `api.DefaultOverpassURLs` when
  `OVERPASS_URLS` is unset. `main.go` is the only file that imports both `weatherservice` and
  `weatherservice/api`.

Independence from Task 13: the test fakes use **boosted** Wikidata IDs (`Q652806`, `Q2063403`,
`Q11650434`, `Q168001`), because a boosted place skips the type filter. The tests therefore pass
whether `planner/placetypes_gen.go` is the empty placeholder or the generated list.

## RED evidence

### First run — the contract signatures exist with zero-value bodies

`NewTripPlanner` returned a planner whose `Plan` returned `(nil, nil)`, and `OverpassURLs` was
declared but never filled.

```
$ CGO_ENABLED=1 go test -count=1 -run 'TripPlanner|LoadEnvKey|PlanRoute' .
--- FAIL: TestLoadEnvKey_ReadsOverpassURLs (0.00s)
    env_test.go:34:
        	Error:      	Not equal:
        	            	expected: []string{"https://overpass.one/api/interpreter", "https://overpass.two/api/interpreter"}
        	            	actual  : []string(nil)
--- FAIL: TestTripPlanner_PlansFromTheGivenSources (0.00s)
    tripplanner_test.go:119:
        	Error:      	Expected value not to be nil.
--- FAIL: TestPlanRoute_WithTheRealTripPlanner (0.00s)
    tripplanner_test.go:142:
        	Error:      	Not equal:
        	            	expected: 200
        	            	actual  : 500
FAIL	weatherservice	0.534s
```

`TestLoadEnvKey_OverpassURLsAreEmptyWhenUnset` passed on this run, which proves nothing on its own.

### Second run — the naive implementation, so the unset test fails too

`weatherServiceKeys.OverpassURLs = strings.Split(os.Getenv("OVERPASS_URLS"), ",")`:

```
$ CGO_ENABLED=1 go test -count=1 -run 'LoadEnvKey' .
--- FAIL: TestLoadEnvKey_ReadsOverpassURLs (0.00s)
    env_test.go:34:
        	            	expected: []string{"https://overpass.one/api/interpreter", "https://overpass.two/api/interpreter"}
        	            	actual  : []string{"https://overpass.one/api/interpreter", " https://overpass.two/api/interpreter"}
--- FAIL: TestLoadEnvKey_OverpassURLsAreEmptyWhenUnset (0.00s)
    env_test.go:48:
        	Error:      	Should be empty, but was []
FAIL	weatherservice	0.509s
```

All four tests have now failed on an assertion before the code that makes them pass existed.

## GREEN

```
$ CGO_ENABLED=1 go test -race -count=1 -run 'TripPlanner|LoadEnvKey|PlanRoute' .
ok  	weatherservice	1.723s
```

| Test | Size | Status |
|---|---|---|
| `TestTripPlanner_PlansFromTheGivenSources` | Small | PASS |
| `TestLoadEnvKey_ReadsOverpassURLs` | Small | PASS |
| `TestLoadEnvKey_OverpassURLsAreEmptyWhenUnset` | Small | PASS |
| `TestPlanRoute_WithTheRealTripPlanner` | Medium | PASS |

`TestPlanRoute_WithTheRealTripPlanner` is the one integration test across the whole chain: the real
`GET /plan` route → `NewTripPlanner` → `planner.Build` → the templates, with fake sources and no
network. It asserts on the rendered HTML: the day heading and date, every stop, the rain badge, both
places to stay and the Booking link.

## The import rule

```
$ go list -deps . | grep weatherservice/api
(no output)
```

## Browser / end-to-end check (large, by hand)

Run against the real internet with `WEATHER_API_KEY` from `.env`, on a scratch database
(`DB_PATH` pointed outside the repo), with `planner/placetypes_gen.go` freshly generated (428 lines).

1. **`GET /?city_name=Lisbon, Portugal`** → 200. The page shows the **Plan my trip** card with
   `city="Lisbon"`, `country="pt"`, `lat="38.7078"`, `lon="-9.1366"`, starting tomorrow and with
   `min` set to today.
2. **`GET /plan?city=Lisbon&country=Portugal&lat=38.7223&lon=-9.1393&start=<tomorrow>&days=3`**
   → 200 in **24.5 s** (cold: Wikimedia, Open-Meteo and Overpass all fetched). A real plan:

   | Day | Stops | Walk |
   |---|---|---|
   | 1 · Mon, 21 Sep | Belém Tower, Padrão dos Descobrimentos, Jerónimos Monastery, Belém Palace | 3.4 km |
   | 2 · Tue, 22 Sep | Alfama, Monastery of São Vicente de Fora, Church of Santa Engrácia, Castle of Saint George | 1.7 km |
   | 3 · Wed, 23 Sep | Lisbon Cathedral, Lisbon Baixa, Santa Justa Lift, Teatro Nacional de São Carlos | 1.7 km |

   Places to stay: Andaz Lisbon, Hotel Altis Avenida, 9 Hotel Mercy, Hotel Avenida Palace,
   Bairro Alto Hotel, Montebelo. Photos and Commons credits render. No rain badge, because
   Lisbon has no rain in the forecast this week.
3. **The same request again** → 200 in **3 ms**, byte-identical. Well under the 2 s target.
   The console stays clean; the only planner log is Wikimedia splitting its geosearch, which is
   expected for a city that hits the 500-result cap.
4. `source_cache` afterwards holds `places:38.722:-9.139`, `forecast:38.722:-9.139` and
   `stays:38.711:-9.137`.

### Offline check

A real network-off test was not possible here (I can't turn this machine's network off), so it was
approximated by restarting the app with `OVERPASS_URLS=https://127.0.0.1:1/api/interpreter`:

- The **cached** Lisbon plan still came back in **3.9 ms**, identical to before, with all six places
  to stay — nothing was fetched.
- A **fresh** city (Porto, `41.1579,-8.6291`, 2 days) still planned in 8.6 s and rendered
  *Luiz I Bridge, Porto Cathedral, Palácio da Bolsa, Ponte de D. Maria Pia* / *Casa da Música,
  National Museum Soares dos Reis, Livraria Lello*, with the stay card showing
  "Places to stay are unavailable right now". The log shows the dead URL was the one used, which is
  also the proof that `OVERPASS_URLS` reaches the Overpass client.

## Full suite

```
$ CGO_ENABLED=1 go test -race -count=1 ./...
ok  	weatherservice
ok  	weatherservice/api
ok  	weatherservice/cmd/placetypes
ok  	weatherservice/planner
$ make build
go build -o bin/traveltab ./cmd/web
$ golangci-lint run . ./cmd/web/
0 issues.
```

**Re-run at the end of the session**, after Task 16's `planner/cities_test.go` landed (written at
04:40, while this lane was running): `weatherservice`, `weatherservice/api` and
`weatherservice/cmd/placetypes` stay green, and `weatherservice/planner` now fails four of Task 16's
city tests:

```
--- FAIL: TestCities_LisbonSkipsBridgesAndTheNationalLibrary
--- FAIL: TestCities_LisbonRainyDayIsMostlyIndoor
--- FAIL: TestCities_KyotoRainyDayHasThreeIndoorStops
--- FAIL: TestCities_KyotoHasNoWards
```

They belong to the Task 16 lane and are unrelated to this one: no file this lane owns is involved,
and the full suite was green at 04:38, before that file existed. `TestCities_KyotoHasNoWards` fails
on its own fixture guard ("the fixture must offer Higashiyama-ku, or this test proves nothing"),
which points at the recorded responses rather than at the wiring.

`docker build -t traveltab .` was **not run**: the docker daemon is not reachable on this machine.

**`make lint` fails on a file this lane does not own**, and it failed before this task started:

```
cmd/loadtest-server/main.go:74:20: Error return value of `os.RemoveAll` is not checked (errcheck)
```

`cmd/loadtest-server/main.go` is committed (009b8e6) and unmodified. It needs a one-line fix from
whoever owns the load-test lane before `make lint` is green.
