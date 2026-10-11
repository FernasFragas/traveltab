# PR 7 integration (October 11, 2026)

Source: `tasks/todo.md` PR 7, "Integrator for PR 7". Browser: isolated headless Brave on `CDP_PORT=9227`, temp profile, preview with the `writers-map` scenario (fixture data only).

## Journeys
- As a traveler, I generate, regenerate and clear a plan on a writer destination and a base-only one, and no layer is duplicated.
- As a traveler, I save places; they survive a reload, cost no request, and only `/plan` carries `include`.
- As a traveler in a private window (blocked storage), I see no Save buttons or saved line.
- As a traveler, **View on map** turns on Your day plan and scrolls to the map, without an inline handler.

## Changes
| File | Change |
| --- | --- |
| `views/itinerary_body.go.tpl`, `public/redesign/map-dayplan.js` | inline `onclick` of **View on map** moved to a delegated click in `map-dayplan.js` (CSP without `'unsafe-inline'`) |
| `internal/adapters/httpserver/itinerary_test.go` | asserts no `onclick` |
| `public/redesign/planner.css` | form grid 3 to 4 columns so Travelers sits beside Start/Ends/Days at 640px and up; one column on phones |
| `cmd/design-preview/fixture.go`, `fixture_test.go` | daily `MaxC`/`MinC`/`WeatherCode` on days 1-3 (day 4+ none); fixture plan honours `Include` (4 stops a day, overflow in `AlsoSuggested`) |
| `tasks/redesign/checks/pr7-acceptance.browser.mjs` | new browser check |
| `tasks/redesign/checks/writers-map-preview.browser.mjs` | layer list now counts only chip buttons (7c added the picker to the slot) |
| `tasks/redesign/checks/lib/browser.mjs` | ignores `Invalid InterceptionId.` (a request cancelled by navigation; made `writers-map-core` fail about 1 run in 2 at 1440 px, 5 of 5 pass after) |

## RED / GREEN
- Inline handler. `go test ./internal/adapters/httpserver/ -run TestItinerary_HasExport -count=1`: RED `... should not contain "onclick"` (FAIL). GREEN after the move: `ok weatherservice/internal/adapters/httpserver 0.275s` (`-run Itinerary`).
- Fixture temperature. `go test ./cmd/design-preview/ -run TestFixtureTrips_PlanDays`: RED `day 1 forecast incomplete: &{... MaxC:<nil> MinC:<nil> WeatherCode:<nil>}`. GREEN: `ok ... coverage: 81.0%`.
- Fixture include. `-run SeedsIncluded`: RED `stops=[{ID:fixture-1 ...` (saved places missing). GREEN: `ok weatherservice/cmd/design-preview 1.227s coverage: 81.5% of statements` (was 78.2%).
- Browser acceptance (`node tasks/redesign/checks/pr7-acceptance.browser.mjs`), GREEN on its first complete run after fixing the check's own request filter (RED was `8 !== 1`, the check counted earlier requests):
```text
PASS 7b porto-pt 375px: 10 stops, no duplicates after regenerate, clear
PASS 7b madeira-pt 375px: 10 stops, no duplicates after regenerate, clear
PASS 7d 375px: save made 0 requests; only /plan carried include
PASS 7e preview 375px: saved places in the day (Madeira, Funchal, Pico Ruivo, Cabo Girao), 1 also suggested
PASS 7g 375px: View on map presses chip once, scrolls 4181 -> 2957
... same at 1440px ...
PASS 7d blocked storage: buttons and saved line hidden, plan still works, console clean
```

## What is guaranteed
| Guarantee | Test | Level | Result |
| --- | --- | --- | --- |
| No duplicate plan layers on generate, regenerate, clear; works on base-only Porto | `pr7-acceptance.browser.mjs`, `map-dayplan.browser.mjs` | integration | PASS (375, 1440) |
| Save makes 0 requests; `include` only on `/plan` | `pr7-acceptance.browser.mjs` | integration | PASS |
| Saves survive reload | `pr7-integration.browser.mjs`, `map-saved.test.mjs` | integration/unit | PASS |
| Blocked storage hides buttons and line | `pr7-acceptance.browser.mjs` | integration | PASS |
| Saved places are in the plan; the 5th of 5 on a one-day plan is "also suggested" | `pr7-acceptance.browser.mjs`, `TestFixtureTrips_PlanSeedsIncludedPlacesAndListsOverflow` | integration/unit | PASS |
| View on map presses the chip only when not pressed, scrolls up, no inline handler | `pr7-acceptance.browser.mjs`, `TestItinerary_HasExportToCalendarAndViewOnMapButtons` | integration | PASS |
| Fixture days 1-3 have a temperature, the last none | `TestFixtureTrips_PlanDaysCarryTemperatureExceptTheLast` | unit | PASS |

## Full suite
- `make test build`: all packages `ok`, `go build -o bin/traveltab ./cmd/web`.
- `node --test tasks/redesign/checks/*.test.mjs`: `tests 56 pass 56 fail 0`.
- Browser checks, PASS at 375 and 1440 px: `map-dayplan`, `map-itineraries`, `map-panel`, `writers-map-core` (5 of 5 reruns), `writers-map-layout`, `writers-map-preview`, `pr7-integration`, `pr7-acceptance`.
- `gofmt -l internal cmd`: no output. `go vet ./...`: no output.
- `make lint`: `Error: can't load config: the Go language version (go1.26) used to build golangci-lint is lower than the targeted Go version (1.27.2)` (known; not reinstalled).

## Preview journey (fixture, 375 / 700 / 1440 px, console clean in every run)
Hero + weather card with 4-day forecast and sunset; pins and the place panel (Pico Ruivo, AI summary, writers); four legend chips plus the itinerary picker (Madeira only; Porto has none); Save to plan with "1 saved place will be included"; itinerary rows with `23°`/`19°`/`21°`, "1.1 km from previous"; Booking link `group_adults=2`; View on map. No horizontal overflow.

## Mockup gaps (`docs/ideas/Funchal Travel Guide Dashboard.png`)
- Hero: no photo in the fixture ("Photo unavailable"), shows "PT" instead of "Madeira, Portugal" (fixture has no region), no one-line description.
- Plan form sits below the map in a separate card; in the mockup it is a full-width card with an icon, an End date (ours: read-only Ends + Days) and "2 travelers" wording.
- Map is a flat fixture tile, not satellite; no "Want to visit (3)" or theme toggle in the header.
- Itinerary rows: stop cards (photo grid) instead of the mockup's list rows with a short description; day shows condition text, not just icon + temperature.
- Panel: monograms are initials, not avatars.

## Known gaps
- `make lint` unavailable here.
- `<img onerror="...">` inline handlers remain in the itinerary stop photos (pre-existing pattern, also blocked by a strict CSP); not changed.
- Open, not built: Wikidata-description fallback for the hero; short stop description; `include`/`travelers` on the `/trip/:slug` re-plan and the `.ics`/`.kml` exports.
