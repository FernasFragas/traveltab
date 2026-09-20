# Notes: Task 9 (Schedule) and Task 10 (Stays)

Files owned by this lane: `planner/weather.go`, `planner/weather_test.go`, `planner/stays.go`,
`planner/stays_test.go`. Written test-first, with the RED runs kept below.

## How it was built

1. Added `Schedule`, `PickStays` and `BookingURL` with their contract signatures and zero-value
   bodies, so a failing run would be an assertion failure and not a compile error.
2. Wrote every test listed in [todo.md](todo.md) for both tasks, ran them and kept the output (RED).
3. Made them pass, then split the logic into small named steps and re-ran the focused tests.

## Task 9: Schedule days around the forecast

`Schedule(groups, pool, fc, start, today)` in `planner/weather.go`:

- `dateDays` — day *i* is `start + i` days at midnight in the forecast's time zone (UTC with no
  forecast or an unknown zone). The date keeps `start`'s calendar day, so a trip asked for the 1st
  never slides to the 2nd because of a time-zone conversion.
- `Certain` is 0–6 whole calendar days after `today` (`ForecastTrustDays`). `Forecast` is filled in
  whenever the forecast covers the date, even for a day that is not certain (days 8–16 show the
  weather, just marked less certain). `Rainy` is `Certain` **and** a rain value is present **and**
  it is at least `RainyDayMM`.
- `assignGroups` — only the certain days take part. Their groups are sorted by how many Indoor or
  Mixed stops they hold, the most sheltered go to the rainy dates in date order, and the rest fall
  back onto the dry dates in their original order. Days that are not certain keep their group.
- `topUpRainyDays` — while a rainy day has fewer than `MinStopsPerDay` sheltered stops, it drops its
  least famous outdoor stop and takes the **nearest unused** Indoor or Mixed place from `pool`
  (distance to the nearest stop of that day, ties by fame then ID). Every place already planned,
  and every place dropped, stays marked as used, so no place is ever planned twice. A changed day is
  re-ordered with `WalkingLoop`, which also refreshes `WalkKM`.

### RED (assertion failures against the zero-value body)

`CGO_ENABLED=1 go test -race -count=1 -run 'Schedule' ./planner/` — 13 failures, one per test:

```
--- FAIL: TestSchedule_DatesFollowTheStartDate (0.00s)
    weather_test.go:62: Error: "[]" should have 3 item(s), but has 0
--- FAIL: TestSchedule_DatesUseTheForecastTimeZone (0.00s)
    weather_test.go:72: Error: "[]" should have 1 item(s), but has 0
--- FAIL: TestSchedule_RainyDayGetsTheMostIndoorGroup (0.00s)
    weather_test.go:80: Error: "[]" should have 2 item(s), but has 0
--- FAIL: TestSchedule_RainyDayIsToppedUpToThreeIndoorStops (0.00s)
    weather_test.go:91: Error: "[]" should have 1 item(s), but has 0
--- FAIL: TestSchedule_TopUpUsesTheNearestUnusedIndoorPlaces (0.00s)
    weather_test.go:104: Error: "[]" should have 1 item(s), but has 0
--- FAIL: TestSchedule_NoPlaceAppearsTwice (0.00s)
    weather_test.go:116: Error: "[]" should have 2 item(s), but has 0
--- FAIL: TestSchedule_FiveMMIsRainy (0.00s)
    weather_test.go:129: Error: "[]" should have 1 item(s), but has 0
--- FAIL: TestSchedule_JustUnderFiveMMIsDry (0.00s)
    weather_test.go:135: Error: "[]" should have 1 item(s), but has 0
--- FAIL: TestSchedule_MissingRainIsNotRainy (0.00s)
    weather_test.go:145: Error: "[]" should have 1 item(s), but has 0
--- FAIL: TestSchedule_DaysSevenOrMoreOutAreNotCertain (0.00s)
    weather_test.go:155: Error: "[]" should have 2 item(s), but has 0
--- FAIL: TestSchedule_DaysSevenOrMoreOutAreNotRearranged (0.00s)
    weather_test.go:167: Error: "[]" should have 2 item(s), but has 0
--- FAIL: TestSchedule_DryDaysKeepTheirStops (0.00s)
    weather_test.go:176: Error: "[]" should have 2 item(s), but has 0
--- FAIL: TestSchedule_NilForecastKeepsTheOrderWithNoWeather (0.00s)
    weather_test.go:187: Error: "[]" should have 2 item(s), but has 0
FAIL	weatherservice/planner	0.332s
```

### Second check: the tests really do test the behaviour

Two throwaway mutations of the finished code, to prove the tests aren't passing by luck:

| Mutation | Tests that failed |
|---|---|
| Skip `topUpRainyDays` | `TestSchedule_RainyDayIsToppedUpToThreeIndoorStops`, `TestSchedule_TopUpUsesTheNearestUnusedIndoorPlaces` |
| Never rearrange the certain days | `TestSchedule_RainyDayGetsTheMostIndoorGroup` |

Both mutations were undone straight away.

## Task 10: Pick places to stay and build the booking link

`planner/stays.go`:

- `PickStays(center, stays)` drops places with no name and anything farther than `StayRadiusM` from
  the center, then sorts with `sort.SliceStable` on website, stars (highest first), distance and
  name, and returns at most `MaxStays`.
- `BookingURL(city, country, start, days)` returns
  `https://www.booking.com/searchresults.html?ss=<city, country>&checkin=…&checkout=…`, with the
  search text query-escaped and checkout at `start + days`. An empty city or country is left out
  instead of leaving a stray comma.

### RED (assertion failures against the zero-value body)

`CGO_ENABLED=1 go test -race -count=1 -run 'PickStays|BookingURL' ./planner/` — 8 failures:

```
--- FAIL: TestPickStays_DropsPlacesFartherThan1Km (0.00s)
    stays_test.go:25: expected: []string{"Near Hotel"}  actual: []string{}
--- FAIL: TestPickStays_DropsUnnamedPlaces (0.00s)
    stays_test.go:31: expected: []string{"Named Hostel"}  actual: []string{}
--- FAIL: TestPickStays_PutsPlacesWithAWebsiteFirst (0.00s)
    stays_test.go:38: expected: []string{"Zulu Hostel", "Alfa Hotel"}  actual: []string{}
--- FAIL: TestPickStays_ThenSortsByStars (0.00s)
    stays_test.go:44: expected: []string{"Zulu Hotel", "Alfa Hotel"}  actual: []string{}
--- FAIL: TestPickStays_ThenSortsByDistance (0.00s)
    stays_test.go:50: expected: []string{"Zulu Hotel", "Alfa Hotel"}  actual: []string{}
--- FAIL: TestPickStays_ReturnsAtMostSix (0.00s)
    stays_test.go:58: Error: "[]" should have 6 item(s), but has 0
--- FAIL: TestBookingURL_CheckoutIsStartPlusDays (0.00s)
    stays_test.go:63: expected: "https://www.booking.com/searchresults.html?ss=Lisbon%2C+Portugal&checkin=2026-03-01&checkout=2026-03-04"  actual: ""
--- FAIL: TestBookingURL_EscapesCityAndCountry (0.00s)
    stays_test.go:68: Error: "" does not contain "ss=S%C3%A3o+Paulo%2C+Brazil"
FAIL	weatherservice/planner	0.283s
```

In each sort test the loser wins on every other key, so only the key under test can explain the
order.

## GREEN

```
CGO_ENABLED=1 go test -race -count=1 -run 'Schedule|PickStays|BookingURL' ./planner/   # ok, 21 tests
CGO_ENABLED=1 go test -race -count=1 ./planner/                                        # ok
golangci-lint run ./planner/...                                                        # 0 issues
make test                                                                              # all packages ok
```

`make lint` stops on `api/wikimedia.go` needing gofmt, which belongs to another lane. Every file in
this lane is gofmt-clean.

## Decisions worth knowing

- **Dates keep the requested calendar day.** `Schedule` reads `start`'s year, month and day and
  rebuilds midnight in the forecast's zone, rather than converting the instant, which in a western
  zone would move a UTC-midnight start back a day.
- **`Certain` is purely about the date**, as `types.go` says, so it stays true with no forecast at
  all. The UI should check `Forecast != nil` before showing weather for a day.
- **Dry days keep their order.** Only rainy dates pull groups out of order; the leftovers go back
  onto the dry dates in the order they started in, so a plan changes as little as the weather needs.
- **Swapped-out stops are not reused.** A stop dropped from a rainy day is not offered to another
  day, which keeps "no place appears twice" true for the whole plan.
- **A day with fewer than three stops and nothing outdoor to trade** takes an extra sheltered place
  instead of a swap, so a short day still reaches three stops when the pool allows.

## Contract

No problems found. `Schedule`, `PickStays` and `BookingURL` match the signatures in
[plan.md](plan.md), and only `Group`, `Day`, `Place`, `Forecast`, `Stay` and the constants were
needed. Nothing in the contract was changed.
