# Lane C notes: export the trip (.ics, .kml, Google Maps link)

A first agent burned significant budget on this lane (102 tool calls, ~250K tokens) without
persisting any real code: everything in this diff was written by the coordinating session
directly, after finding the first attempt had left only a single unused `net/url` import in
`trip.go` (fixed — see `HANDOVER.md`).

## C1: `.ics` calendar export

**Files:** `internal/adapters/httpserver/export_ics.go`, `export_ics_test.go`, plus
`resolveTripPlan` added to `trip.go` (shared by C1 and C2).

**Route:** `GET /trip/:slug.ics?days=N&from=YYYY-MM-DD`. Confirmed live that Fiber v2's router
correctly captures the bare slug for a `:slug.ics` pattern without conflicting with the plain
`/trip/:slug` route (checked with a throwaway harness before writing any real code).

**Tests first (RED):** all 6 named in `tasks/todo-v2.md`'s C1 section. `TestICS_404sWithoutAFullPlan`
passed immediately against the always-404 stub (a genuine negative control, not weak RED); the
other 5 failed on real assertions.

**Bugs the tests caught, not review:**
- A first draft indexed `stopTimes[stopIndex]` *before* checking `stopIndex >= len(stopTimes)`,
  which would panic on any day with more stops than time slots. Caught by re-reading the code
  before running it, fixed by checking bounds first.
- `TestICS_UsesTheForecastTimeZone`'s first version assumed EST for 2026-03-11; the real code
  gave 14:00 UTC, not the expected 15:00, because 2026-03-11 is already inside US DST (EDT,
  UTC-4). The **code was right, the test's assumption was wrong** — fixed by moving the
  assertion to a January date, which needs no knowledge of the exact 2026 DST transition day.
- Line folding (`fold()`) failed its own test: `"76" is not less than or equal to "75"`. The bug
  was real — a continuation line's `" " + 75-byte chunk` is 76 bytes, one over RFC 5545's limit.
  Fixed by giving continuation chunks a 74-byte budget, so the mandatory leading space brings
  them to exactly 75.

**Design:** all times are converted to UTC and written with a trailing `Z` (`DTSTART:...Z`), per
the original v2 spec's explicit instruction ("write event times in UTC") — this needs no
`VTIMEZONE` block, which a hand-rolled generator would otherwise need to get right for every
zone's DST rules. `Day.Date` already carries the correct `*time.Location` (set in
`internal/planner/weather.go`'s `dateDays`), so no plumbing was needed through the planner to
recover the timezone.

**Real-world validation**, not just the Go test suite: generated a real `.ics` from a live
`go run ./cmd/web` (Lisbon, 2 days, real Wikimedia/Overpass/Open-Meteo data) and parsed it with
Python's `icalendar` library — all 8 events parsed correctly, including UTF-8 names ("Belém
Tower", "Jerónimos Monastery") and DST-correct offsets (Lisbon is UTC+1 in late September).

## C2: `.kml` export

**Files:** `internal/adapters/httpserver/export_kml.go`, `export_kml_test.go`.

**Route:** `GET /trip/:slug.kml?days=N&from=YYYY-MM-DD`, same resolution helper as C1. One
`<Folder>` per day, one `<Placemark>` per stop, KML's `lon,lat` coordinate order (the opposite of
`planner.Place`'s `Lat, Lon` fields — a real detail the tests pin explicitly).

**Tests first (RED):** all 5 from the spec, same pattern as C1.

**Real-world validation:** the same live Lisbon run's `.kml` output parsed cleanly with
`xml.etree.ElementTree`, with all coordinates in valid lat/lon range and UTF-8 names intact.

## C3: "Open Day N in Google Maps" link

**Files:** `internal/adapters/httpserver/googlemaps.go`, `googlemaps_test.go`, plus a `MapsURL`
field added to `tripDayView` and wired in `views/trip_card.go.tpl`.

**Design:** first stop = origin, last stop = destination, everything between = a `|`-separated
waypoint list. `MaxStopsPerDay` is 4, so this is at most 2 waypoints — comfortably inside
Google's documented mobile limit. A single-stop day gets a plain search link instead of
directions, since there's no route to walk. No API key needed either way.

**Tests first (RED):** all 5 named in the spec, covering the 3-stop case, the waypoint-count
boundary at `MaxStopsPerDay`, the 2-stop (no-waypoints) case, and the 1-stop (search-link) case.

## Wiring: `exportURLs`

`newTripPlanView` (in `trip.go`) now sets `ICSURL`/`KMLURL` on the plan view directly from
`plan.Request` — no signature change needed elsewhere, since `Slug`, the day count and the start
date are all already on `Request`. One real bug caught before it shipped: the first draft
appended `.ics`/`.kml` **after** the query string (`/trip/lisbon-pt?days=3&from=....ics`), which
doesn't match a route at all. Fixed to put the extension on the path segment, before the query
(`/trip/lisbon-pt.ics?days=3&from=...`).

**Integration test:** `TestPlanRoute_LinksToItsOwnExportsAndMapsRoutes` proves `/plan`'s own
response actually links to `.ics`/`.kml`/Maps URLs with a slug and query the export handlers can
resolve — not just that each piece works in isolation. Its first run failed because the
assertion checked for a raw `&` in the URL, when `html/template` correctly escapes it to `&amp;`
inside an HTML attribute; that was the test's bug, not the application's, fixed by asserting the
escaped form.

## Verification

- `CGO_ENABLED=1 go test -race -count=1 ./...` — all packages pass (this and every other lane's
  tests, including `internal/guides`).
- `golangci-lint run ./...` — 0 issues.
- `make build` and layering check (`go list -deps ./internal/planner/... | grep adapters`,
  same for `application`) — both clean.
- Live, hand-verified against a real running app with real API data (not just fixtures): `.ics`
  and `.kml` both generated and independently parsed by real libraries (Python `icalendar` and
  `xml.etree.ElementTree`), and the "Open Day 1 in Google Maps" link confirmed present on the
  rendered `/trip/lisbon-pt` page.

## Not done / left for a human

- No real import into Google Calendar, Apple Calendar, Outlook or Google My Maps was performed —
  no such accounts were available in this environment. The `.ics` and `.kml` were validated by
  parsing them with real, independent libraries instead, which catches structural and encoding
  errors but not every quirk a specific calendar app's importer might have.
- Nothing was committed.
