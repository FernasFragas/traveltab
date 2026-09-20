# Notes: Task 7 (Rank + boosts) and Task 8 (Group days)

Owned files: `planner/rank.go`, `planner/boosts.go`, `planner/rank_test.go`,
`planner/days.go`, `planner/days_test.go`.

Focused command used throughout:
`CGO_ENABLED=1 go test -race -count=1 -run 'Rank|Boosts|GroupDays' ./planner/`

---

## Task 7: Rank places, with the type filter and boost list

All 13 tests from todo.md were already in `planner/rank_test.go` and match the list
name for name. Nothing was added or removed.

### RED

Run against the zero-value stubs (`func Rank(...) []Place { return nil }` and
`var Boosts = map[string]Kind{}`). Every one of the 13 failed on an assertion; none
failed to compile, and none passed.

```text
--- FAIL: TestRank_KeepsAPlaceWithAnIndoorType (0.00s)
    rank_test.go:17: Error: "[]" should have 1 item(s), but has 0
--- FAIL: TestRank_DropsAPlaceWithOnlyUnknownTypes (0.00s)
    rank_test.go:22: Error: "[]" should have 1 item(s), but has 0
--- FAIL: TestRank_DenyTypeBeatsIndoorType (0.00s)
    rank_test.go:27: Error: "[]" should have 1 item(s), but has 0
--- FAIL: TestRank_IndoorPlusOutdoorTypesMakeMixed (0.00s)
    rank_test.go:32: Error: "[]" should have 1 item(s), but has 0
--- FAIL: TestRank_SortsByFameThenByID (0.00s)
    rank_test.go:37: Error: "[]" should have 3 item(s), but has 0
--- FAIL: TestRank_ReturnsAtMostSixtyPlaces (0.00s)
    rank_test.go:45: Error: "[]" should have 60 item(s), but has 0
--- FAIL: TestRank_SetsKindOnEveryPlace (0.00s)
    rank_test.go:49: Error: "[]" should have 2 item(s), but has 0
--- FAIL: TestRank_BoostedPlaceSkipsTheTypeFilter (0.00s)
    rank_test.go:56: Error: "[]" should have 1 item(s), but has 0
--- FAIL: TestRank_BoostedPlaceGetsTheBoostKind (0.00s)
    rank_test.go:61: Error: "[]" should have 1 item(s), but has 0
--- FAIL: TestRank_BoostedPlaceRanksLikeTheTenthPlace (0.00s)
    rank_test.go:71: Error: "[]" should have 60 item(s), but has 0
--- FAIL: TestRank_BoostWithFewerThanTenPlacesRanksLikeTheLastPlace (0.00s)
    rank_test.go:81: Error: "[]" should have 3 item(s), but has 0
--- FAIL: TestRank_DoesNotChangeTheInput (0.00s)
    rank_test.go:88: Error: "[]" should have 2 item(s), but has 0
--- FAIL: TestBoosts_HasTheFiveStartingPlaces (0.00s)
    rank_test.go:92: Error: Not equal:
        expected: map[string]planner.Kind{"Q11650434":"indoor", "Q168001":"outdoor",
                  "Q2063403":"indoor", "Q23579173":"outdoor", "Q652806":"indoor"}
        actual  : map[string]planner.Kind{}
FAIL	weatherservice/planner	0.341s
```

### GREEN — how it works

- `kindOf(placeTypes, types)` is the one "which kind is this place" function. It walks
  the place's types in slice order (never a map), returns early on a `Deny` type, and
  otherwise folds Indoor/Outdoor/Mixed into a single Kind. No known type means the
  place is dropped.
- Boosted places (`boosts[p.ID]`) skip `kindOf` entirely and take the boosted Kind.
- **`Place.Sitelinks` is never written.** Sorting happens on `rankEntry{place, fame,
  boosted}`, an internal struct. A boosted place's `fame` is borrowed from the plain
  (non-boosted) list: the fame of the `boostRank`-th entry (10th), or of the last entry
  when the list is shorter, or its own when there is no plain list at all.
- Sort order, a total order so the result never depends on input order:
  1. `fame` descending
  2. boosted before plain at equal fame (this is what makes the boost land *at* rank 10
     rather than just after it)
  3. `place.ID` ascending
- Truncated to `CandidatePoolSize` (60).

Note on rule 2: with ID-only tie-breaking, a boost borrowing the 10th place's fame
would sort *behind* the real 10th place and land at index 10, failing
`TestRank_BoostedPlaceRanksLikeTheTenthPlace`. Boosted-wins-ties puts it exactly at
index 9 and keeps the order deterministic.

### Result

13/13 pass. `Rank` copies each `Place` by value, so `TestRank_DoesNotChangeTheInput`
holds; the input `Types` slices are only read.

---

## Task 8: Group places into days

New files `planner/days.go` and `planner/days_test.go`. All 10 tests from todo.md,
by name.

### RED

`func GroupDays(ranked []Place, days int) []Group { return nil }` first, then the
tests. 8 of the 10 failed on assertions:

```text
--- FAIL: TestGroupDays_NeverMixesTwoFarApartAreas (0.00s)
    days_test.go:65: Error: "[]" should have 2 item(s), but has 0
--- FAIL: TestGroupDays_EveryDayHasThreeToFourStops (0.00s)
    days_test.go:78: Error: "[]" should have 5 item(s), but has 0
--- FAIL: TestGroupDays_SmallTownGetsFewerDays (0.00s)
    days_test.go:91: Error: "[]" should have 2 item(s), but has 0
--- FAIL: TestGroupDays_OnePlaceGivesOneDay (0.00s)
    days_test.go:97: Error: "[]" should have 1 item(s), but has 0
--- FAIL: TestGroupDays_NoPlaceAppearsTwice (0.00s)
    days_test.go:122: Error: Should NOT be empty, but was map[]
--- FAIL: TestGroupDays_OrdersEachDayAsAWalkingLoop (0.00s)
    days_test.go:128: Error: "[]" should have 4 item(s), but has 0
--- FAIL: TestGroupDays_SameInputGivesTheSameDays (0.00s)
    days_test.go:137: Error: Should NOT be empty, but was []
--- FAIL: TestGroupDays_LisbonLikeDaysWalkUnder15Km (0.00s)
    days_test.go:146: Error: "[]" should have 4 item(s), but has 0
FAIL	weatherservice/planner	0.391s
```

`TestGroupDays_NoPlacesGivesNoDays` and `TestGroupDays_UsesOnlyTheTop20` pass
vacuously against a `return nil` stub — they are **boundary guards**, in the same sense
as Task 1's same-place-distance and empty-image tests, not independent RED evidence.
The other 8 are the real RED.

### GREEN — the four steps

1. **Number of days** — `dayCount(size, days)`: `min(days, size/MinStopsPerDay)`, floored
   at 1 while there is at least one place. 7 places with 5 days asked gives 2.
2. **Clustering** — `clusterPlaces`: deterministic k-means.
   - `seedCenters`: seed 0 is the most famous place (`moreFamous` from `geo.go`), then
     farthest-point — the place with the largest distance to the nearest seed so far,
     ties to the lower ID. Places already chosen are skipped.
   - `assignToCenters`: each place goes to its closest centre; a distance tie goes to the
     lower centre index. The pool is walked in ranked order, so **every cluster stays in
     ranked order** and its last element is always its least famous stop.
   - `recenter`: plain mean of lat/lon. An empty cluster keeps its previous centre so it
     still has somewhere to pull stops towards.
   - Stops as soon as the centres stop moving; `kmeansRounds` (50) is a safety cap.
3. **Balancing** — `balanceClusters`:
   - Short days (< `MinStopsPerDay`) call `pullNearest`, which takes the stop closest to
     that day's centre out of any *other* day that has more than `MinStopsPerDay` stops.
     A donor never drops below 3, so the number of short days only ever decreases and the
     loop terminates.
   - Days still longer than `MaxStopsPerDay` are truncated: because clusters are in
     ranked order, `clusters[i][:MaxStopsPerDay]` *is* "drop the lowest-ranked stops".
4. **Output** — `WalkingLoop` per day (from `geo.go`, not reimplemented), empty days
   dropped, days sorted by their most famous stop (which `WalkingLoop` puts at index 0).

### Determinism

No map is iterated anywhere in `rank.go` or `days.go` — maps are only read by key. Every
min/max scan uses a strict comparison plus an ID or index tie-break, so results do not
depend on slice order. Verified with
`go test -race -count=3 -shuffle=on -run 'Rank|Boosts|GroupDays' ./planner/`.

### Reading of "pass their farthest stops to nearby groups with room"

Implemented as a **pull** (short days take the nearest sparable stop) rather than a
**push** (full days shove their farthest stop at any day with room). The push reading
was tried on paper and is much worse: with the Lisbon fixture at 4 days it sends the
Parque das Nações tower to the Belém day, giving a ~22 km walking day and failing
`TestGroupDays_LisbonLikeDaysWalkUnder15Km`. The pull reading satisfies both "each group
gets 3–4 stops" and "nearby groups", and only ever moves a stop to rescue a day that
would otherwise be under 3 stops.

### What the Lisbon fixture actually produces

20 places in four walkable areas (Belém, Baixa, Alfama, Parque das Nações), five each:

```
days=2   15.59km [Q01 Q05 Q04 Q02]      2.05km [Q03 Q08 Q16 Q12]
days=3    4.53km [Q01 Q09 Q05 Q13]      1.97km [Q02 Q06 Q04 Q07]   2.05km [Q03 Q08 Q16 Q12]
days=4    2.49km [Q01 Q09 Q05]          1.97km [Q02 Q06 Q04 Q07]   2.05km [Q03 Q08 Q16 Q12]   9.70km [Q13 Q17 Q19]
days=5    2.49km [Q01 Q09 Q05]          1.97km [Q02 Q06 Q04 Q07]   1.06km [Q03 Q08 Q20]      13.63km [Q12 Q16 Q18]   9.70km [Q13 Q17 Q19]
```

### Known gap (for the human, not changed here)

**At 2 days the Lisbon fixture gives a 15.6 km day.** With 2 days there are 8 stop slots
for 20 places, so 12 places must be dropped and no day has room for a hand-off. The spec
says the overflow is dropped **by rank**, so the surviving 4 are simply the 4 most
famous — Torre de Belém, the castle, Praça do Comércio and the monastery — which are
spread over ~6 km of city. This is exactly the "one huge day" risk in plan.md's risk
table, and it is what the spec asks for, so it was left alone.

A fix would be to make the drop compactness-aware: keep the most famous stop, then the
`MaxStopsPerDay - 1` highest-ranked stops nearest to it, instead of the top N by fame
alone. That changes the wording "drop their lowest-ranked stops" in todo.md's Task 8,
so it needs the human's OK. It would also change `Build`'s output for short trips, which
Task 14 depends on.

### Contract

No contract problem found. `Rank` and `GroupDays` match plan.md's signatures exactly,
`Place.Sitelinks` is never mutated, and `geo.go`'s `DistanceKM`, `WalkingLoop` and
`moreFamous` are used as-is.

---

## Verification

| Check | Result |
|---|---|
| `CGO_ENABLED=1 go test -race -count=1 -run 'Rank\|Boosts\|GroupDays' ./planner/` | ok — 23/23 |
| `CGO_ENABLED=1 go test -race -count=1 ./planner/` | ok (whole package, including the other lane's tests) |
| `gofmt -l planner/` | clean |
| `make lint` | one pre-existing errcheck issue in `cmd/loadtest-server/main.go`, not an owned file |
| `make test` | `weatherservice/planner` ok, `weatherservice/api` ok; the root package fails in `trip_render_test.go` (`TestFooter_ShowsAllFourDataCredits`), another lane's in-progress work |

Raw RED output kept at `task7-red.txt` and `task8-red.txt` in the session scratchpad.

---

## Task 8, second pass: compact stop selection (decided 2026-09-20)

The human's decision in todo.md ("Choosing the stops"): when there are more places than
slots (`days × MaxStopsPerDay`), drop by **fame and closeness together**, not fame alone.
This replaces the "Known gap" above. Owned files: `planner/days.go`, `planner/days_test.go`.

### RED — the two new tests

Written first and run against the fame-alone implementation:

```text
=== RUN   TestGroupDays_ShortTripsStayCompact
    days_test.go:170:
        	Error Trace:	/Users/fernando/personal-projects/traveltab/planner/days_test.go:170
        	Error:      	"15.589043535479846" is not less than "8"
        	Test:       	TestGroupDays_ShortTripsStayCompact
        	Messages:   	a short-trip day walks too far: [Q01 Q05 Q04 Q02]
    days_test.go:175:
        	Error Trace:	/Users/fernando/personal-projects/traveltab/planner/days_test.go:175
        	Error:      	Should be false
        	Test:       	TestGroupDays_ShortTripsStayCompact
        	Messages:   	a day mixes two far-apart areas: [Q01 Q05 Q04 Q02]
--- FAIL: TestGroupDays_ShortTripsStayCompact (0.00s)
=== RUN   TestGroupDays_AlwaysKeepsTheMostFamousPlace
--- PASS: TestGroupDays_AlwaysKeepsTheMostFamousPlace (0.00s)
FAIL
FAIL	weatherservice/planner	0.434s
```

`TestGroupDays_ShortTripsStayCompact` is the real RED: both of its assertions fail, and
the message is exactly the reported bug — one 15.6 km day holding Torre de Belém, the
monastery, Praça do Comércio and the castle.

`TestGroupDays_AlwaysKeepsTheMostFamousPlace` passes straight away, so it is a **guard**
(same sense as `TestGroupDays_NoPlacesGivesNoDays`): the old rule kept the most famous
place by accident, being fame-ordered, and the guard makes sure the new rule still does.

### GREEN — `selectCompact`

One new step in `GroupDays`, between the top-20 cut and the clustering:

```go
k := dayCount(len(pool), days)
clusters, centers := clusterPlaces(selectCompact(pool, k*MaxStopsPerDay), k)
```

`selectCompact(pool, slots)` returns the whole pool untouched when it already fits. When
it doesn't, it grows the kept set one **area** at a time:

1. The seed of an area is `mostFamousLeft` — the most famous place not kept yet, ties by
   ID. The very first seed is therefore always the most famous place of all.
2. The area then takes `mostFamousNear`: the most famous place left within
   `compactRadiusKM` (1.5 km) of *any* stop already in the area. Reach, not rank, is the
   filter; rank only picks the winner among what is in reach.
3. Nothing in reach: the reach **doubles** and the search runs again. Distances are
   finite, so this always ends — a few rounds cover any city. This is the "widening only
   when nothing qualifies" half of the rule, and it keeps thinly spread cities working:
   an area still fills with its nearest stops rather than with famous strangers.
4. An area stops at `MaxStopsPerDay` (a day's worth) and the next area opens around the
   most famous place left, until `slots` places are kept.

The kept places are returned **in the order they came in**, so everything downstream still
holds: clusters stay in ranked order, and `clusters[i][:MaxStopsPerDay]` in
`balanceClusters` still means "drop the lowest-ranked stops".

Why areas rather than one growing blob: with a single region the 4-day Lisbon plan filled
its 16 slots with Belém plus the whole centre and dropped Parque das Nações — including
the aquarium, the third most famous place. Capping an area at a day's worth and reseeding
from the most famous place left keeps each day compact *and* keeps the famous outlying
areas in the trip.

### Determinism

`mostFamousLeft` and `mostFamousNear` scan the pool by index and compare with `moreFamous`
(fame desc, then ID asc), a total order, so neither depends on input order. `kept` is a
`[]bool` parallel to the pool, never a map. Verified with
`CGO_ENABLED=1 go test -race -count=3 -shuffle=on -run GroupDays ./planner/`.

### What the Lisbon fixture produces now

```
days=1    4.53km [Q01 Q09 Q05 Q13]
days=2    4.53km [Q01 Q09 Q05 Q13]   1.97km [Q02 Q06 Q04 Q07]
days=3    4.53km [Q01 Q09 Q05 Q13]   1.97km [Q02 Q06 Q04 Q07]   2.05km [Q03 Q08 Q16 Q12]
days=4    2.49km [Q01 Q09 Q05]       2.90km [Q02 Q06 Q04 Q10]   2.05km [Q03 Q08 Q16 Q12]  10.14km [Q07 Q15 Q13]
days=5    2.49km [Q01 Q09 Q05]       1.97km [Q02 Q06 Q04 Q07]   1.06km [Q03 Q08 Q20]      13.63km [Q12 Q16 Q18]   9.70km [Q13 Q17 Q19]
```

Worst day at 2 days: **15.59 km → 4.53 km**. The 3-day plan is now three one-area days.
At 5 days nothing changed at all: 20 places into 5 × 4 slots is not an overflow, so
`selectCompact` returns the pool untouched.

### Remaining gap: the long days at 4 and 5 days (for the human, not changed here)

The 10.14 km and 13.63/9.70 km days above are **not** selection any more — at 4 days the
compact rule happens to keep the same 16 places fame alone would. They come from
**`pullNearest` in the balancing step**: a cluster left with two stops pulls the nearest
sparable stop, and "nearest" can still be 5 km away when the only day with a spare stop is
in another part of town (Belém's art museum lands on a Baixa day). todo.md's balancing rule
("short days pull the nearest sparable stop from a day that has more than the minimum") is
exactly what is implemented, so fixing this means changing that rule — e.g. refusing a pull
beyond a distance cap and letting the day have two stops instead. That needs the human's OK,
so it was left alone. All days stay under the 15 km the tests require.

### Verification

| Check | Result |
|---|---|
| `CGO_ENABLED=1 go test -race -count=1 -run GroupDays ./planner/` | ok — **12/12** (the 10 old tests unchanged, both new ones green) |
| `CGO_ENABLED=1 go test -race -count=3 -shuffle=on -run GroupDays ./planner/` | ok |
| `CGO_ENABLED=1 go test -race -count=1 ./planner/` | only `TestBuild_*` fails — `planner/build.go` is still the other lane's zero-value stub |
| `gofmt -l planner/` | clean |
| `golangci-lint run ./planner/...` | 0 issues |

---

## Task 8, third pass: the pull reach limit (decided 2026-09-20)

The human's decision in todo.md ("Balancing"): **a pull may only reach 3 km.** When no
sparable stop is that close, the day keeps the stops it has, so `MinStopsPerDay` is a
target rather than a guarantee. This closes the "Remaining gap" above. Owned files:
`planner/days.go`, `planner/days_test.go`, this note.

### RED — the new test

`TestGroupDays_LongTripsStayWalkable` asks the same of the 4- and 5-day plans that
`TestGroupDays_ShortTripsStayCompact` asks of the 2-day one: every day under 8 km.
Written first and run against the unlimited-reach implementation:

```text
=== RUN   TestGroupDays_LongTripsStayWalkable
    days_test.go:185:
        	Error Trace:	/Users/fernando/personal-projects/traveltab/planner/days_test.go:185
        	Error:      	"10.139265199454751" is not less than "8"
        	Test:       	TestGroupDays_LongTripsStayWalkable
        	Messages:   	a day of the 4-day plan walks too far: [Q07 Q15 Q13]
    days_test.go:185:
        	Error Trace:	/Users/fernando/personal-projects/traveltab/planner/days_test.go:185
        	Error:      	"13.626475122761732" is not less than "8"
        	Test:       	TestGroupDays_LongTripsStayWalkable
        	Messages:   	a day of the 5-day plan walks too far: [Q12 Q16 Q18]
    days_test.go:185:
        	Error Trace:	/Users/fernando/personal-projects/traveltab/planner/days_test.go:185
        	Error:      	"9.699663726468449" is not less than "8"
        	Test:       	TestGroupDays_LongTripsStayWalkable
        	Messages:   	a day of the 5-day plan walks too far: [Q13 Q17 Q19]
--- FAIL: TestGroupDays_LongTripsStayWalkable (0.00s)
FAIL	weatherservice/planner	0.459s
```

All three failures are the reported bug: the 10.1 km day on the 4-day trip and the
13.6 km and 9.7 km days on the 5-day trip, each one a day that had been handed a stop
from another part of town.

### GREEN — `PullReachKM`

One constant beside the other tuning numbers in `days.go`, and one clause in
`pullNearest`:

```go
// PullReachKM is how far a short day may reach for a stop from another day. Beyond it
// the day keeps the stops it has: two stops together beat three stops across town.
PullReachKM = 3.0
...
if d := DistanceKM(p, centers[i]); d <= PullReachKM && d < distance {
```

The reach is measured from the short day's centre, which is what "nearest" already meant
there. Nothing else changed: donors still have to hold more than `MinStopsPerDay`, the
pull still takes the nearest candidate, and over-long days are still truncated by rank.
`GroupDays`'s signature is untouched.

### The one existing test that had to change

`TestGroupDays_EveryDayHasThreeToFourStops` asserted the old guarantee and failed:

```text
--- FAIL: TestGroupDays_EveryDayHasThreeToFourStops (0.00s)
    days_test.go:93: "2" is not greater than or equal to "3"
                     day too short: [Q12 Q16]
    days_test.go:93: "2" is not greater than or equal to "3"
                     day too short: [Q13 Q17]
```

Those two days are the decision working as intended: nothing sparable sits within 3 km of
Parque das Nações or of Belém's museums. The test is now
`TestGroupDays_EveryDayHasThreeToFourStopsUnlessNothingIsInReach` (the old name is a
prefix, so `-run TestGroupDays_EveryDayHasThreeToFourStops` still selects it) and states
the new rule instead of the old guarantee:

- no day is empty, and no day exceeds `MaxStopsPerDay`;
- a day below `MinStopsPerDay` is allowed **only** when every other day that could spare a
  stop keeps all of them more than `PullReachKM` away;
- at least 3 of the 5 days still reach `MinStopsPerDay`, so a plan of short days
  everywhere would fail.

No other test was touched or weakened. The 11 others, including
`TestGroupDays_LisbonLikeDaysWalkUnder15Km` and `TestGroupDays_ShortTripsStayCompact`,
pass unchanged.

### What the Lisbon fixture produces now

```
days=1    4.53km [Q01 Q09 Q05 Q13]
days=2    4.53km [Q01 Q09 Q05 Q13]   1.97km [Q02 Q06 Q04 Q07]
days=3    4.53km [Q01 Q09 Q05 Q13]   1.97km [Q02 Q06 Q04 Q07]   2.05km [Q03 Q08 Q16 Q12]
days=4    2.49km [Q01 Q09 Q05]       1.97km [Q02 Q06 Q04 Q07]   2.05km [Q03 Q08 Q16 Q12]   0.00km [Q13]
days=5    2.49km [Q01 Q09 Q05]       1.97km [Q02 Q06 Q04 Q07]   1.06km [Q03 Q08 Q20]       0.80km [Q12 Q16]   1.05km [Q13 Q17]
```

Worst day, against the table in the previous section:

| Days | Before | After |
|---|---|---|
| 1 | 4.53 km | 4.53 km (unchanged) |
| 2 | 4.53 km | 4.53 km (unchanged) |
| 3 | 4.53 km | 4.53 km (unchanged) |
| 4 | **10.14 km** | **2.49 km** |
| 5 | **13.63 km** | **2.49 km** |

1–3 days never used a pull, so they are untouched. The 4- and 5-day plans are now made of
one-area days only.

### Side effect the human should see: 1-stop days

Refusing a far pull can leave a day with **one** stop, not just two. On the 4-day Lisbon
fixture the k-means puts Belém's art museum (Q13) in a cluster of its own; the only stops
within 3 km are on the Belém day, which holds exactly `MinStopsPerDay` and so may not give
one up, and the day ends as `[Q13]` alone. The same shows on the real recorded fixtures:

| City, days | Before | After |
|---|---|---|
| Lisbon, 5 | 3 stops / 14.55 km | **2 stops / 2.53 km** |
| Kyoto, 3 | 3 stops / 9.71 km | **1 stop / 0.00 km** |
| Kyoto, 4 | 3 stops / 12.49 km and 3 / 9.71 km | **2 stops / 1.00 km** and **1 stop / 0.00 km** |
| Kyoto, 5 | 3 stops / 12.49 km | **2 stops / 1.00 km** |
| Tavira, any | — | unchanged |

Every city test still passes (`Schedule` tops a rainy day back up to three indoor stops
from the pool, so `TestCities_KyotoRainyDayHasThreeIndoorStops` is unaffected), but a day
card showing a single stop is a product question, not a test question. Two ways out, both
needing the human's OK because they change todo.md's balancing rule again:

1. **Let a donor at exactly `MinStopsPerDay` give one up when the receiver would otherwise
   be left with a single stop** — `3 + 1` becomes `2 + 2`. Same total shortfall, no 1-stop
   day, still nothing dragged across town.
2. **Merge an isolated day into its nearest neighbour when that neighbour has room.**
   Better-looking days, but the trip then has fewer days than asked, which
   `TestGroupDays_LisbonLikeDaysWalkUnder15Km` and `TestGroupDays_OrdersEachDayAsAWalkingLoop`
   currently forbid (both require exactly 4 groups for a 4-day trip).

### Verification

| Check | Result |
|---|---|
| `CGO_ENABLED=1 go test -race -count=1 -run GroupDays ./planner/` | ok — **13/13** (12 existing, 1 new) |
| `CGO_ENABLED=1 go test -race -count=1 ./planner/` | ok — whole package, including `TestCities_*` and `TestBuild_*` |
| `CGO_ENABLED=1 go test -race -count=3 -shuffle=on ./planner/` | ok |
| `gofmt -l planner/` | clean |
| `golangci-lint run ./planner/...` | 0 issues |
| `go build ./...` | ok |


## Follow-up: no day left on its own (2026-09-20, coordinating session)

The 3 km reach limit fixed the long days but left single-stop days (`[Q13]` alone on the 4-day
Lisbon fixture, and the same on Kyoto's real data at 3 and 4 days). Option (1) from the note above
was implemented: `pullNearest` lets a day with fewer than 2 stops take from a donor sitting at
exactly `MinStopsPerDay`. The donor keeps two, so a rescue cannot strand anyone in turn.

RED first, from `TestGroupDays_NoDayIsLeftWithASingleStop`:

```text
--- FAIL: TestGroupDays_NoDayIsLeftWithASingleStop (0.00s)
    days_test.go:217: "1" is not greater than or equal to "2"
        a day of the 4-day plan is on its own: [Q13]
```

Lisbon fixture afterwards — every day has 2–4 stops and no day walks over 5 km:

```text
days=2 | 4 stops 4.53km | 4 stops 1.97km
days=3 | 4 stops 4.53km | 4 stops 1.97km | 4 stops 2.05km
days=4 | 2 stops 2.16km | 4 stops 1.97km | 4 stops 2.05km | 2 stops 2.13km
days=5 | 3 stops 2.49km | 4 stops 1.97km | 3 stops 1.06km | 2 stops 0.80km | 2 stops 1.05km
```

`go test -race -count=3 -shuffle=on ./planner/` passes, including every `TestCities_*` and
`TestBuild_*`. `golangci-lint run ./planner/...` reports 0 issues.


## Completion: nearby highlights rescue Kyoto (2026-09-20)

`TestCities_KyotoUsesNearbyHighlightsInsteadOfASingleStopDay` failed on the recorded three-day
plan: Arashiyama had one stop and nearby bamboo grove Q23579173 was absent. The grove ranked in
the sunny top 20 but was omitted during compact selection. After normal balancing, remaining
singletons now take the nearest unused top-20 highlight within `PullReachKM`, with fame as a tie
breaker. No place repeats and reach limits remain unchanged. Grouping and city race tests pass.
A genuinely isolated stop still stays alone when there is no nearby candidate.
