# Map look v2: TDD evidence

Sources: [plan](../../tasks/map-look-v2-plan.md) and [task list](../../tasks/map-look-v2-todo.md), L1–L11.

## Journeys

- As a traveller, I want to recognise TravelTab at a glance, so that the product feels like one brand.
- As a traveller, I want layer pills above the map and controls on it, so that the map stays clear.
- As a traveller, I want to tell places, writers' pins and my plan apart by colour, shape and icon, so that I know what I am looking at.
- As a traveller, I want a place's story, writers, hours and directions in one panel, so that I can decide without leaving the map.
- As a traveller on a phone, I want a half-height place sheet, so that I still see the map.
- As a traveller, I want save confirmation, Undo and a way back to saved places.

## Working tree and scope

The owner requested implementation with agents on 2026-10-14. Implementation proceeded on the
existing uncommitted map-first working tree, preserving its edits; no commits, pushes or history
changes were made. One implementation agent owned the production files for L1–L10 in order.
Supporting agents reviewed the code, added isolated behavior tests, updated browser checks and
ran browser verification. The root agent integrated documentation and final checks.

Two ownership lists omit files needed for the stated behavior: writer itinerary route styling
is in `public/redesign/map-itineraries.js`, and the map fixture is in
`cmd/design-preview/writersmap.go`. Necessary changes there remain within the requested features.
The panel handler in `internal/adapters/httpserver/writermap.go` also passes the existing source
allowlist to its template so route authors can show blog names without a matching mention.
The frozen GeoJSON contract is unchanged.

## Behavior evidence

The implementation agent retained chronological RED/GREEN output in temporary logs. Initial
static checks use source assertions; they establish markup and styling contracts, while runtime
tests and browser journeys establish interactions.

| Behavior | RED command and actual excerpt | GREEN evidence |
| --- | --- | --- |
| Two-line brand, search without arrow, saved slots, panel cards and full-map card | `node --test tasks/redesign/checks/map-look-v2.test.mjs`: `tests 4`, `pass 0`, `fail 4` | Final suite result recorded below |
| Panel bookmark, writer card, directions and post tile | `go test ./internal/adapters/httpserver/ -run 'TestPlacePanel_(StoryAndDirectionsCards\|ShowsPartsInMockupOrder)'`: `FAIL weatherservice/internal/adapters/httpserver 0.583s`; missing new cards and old tablist present | Final focused/package result recorded below |
| Save confirmation, Undo, list focus and replacement toast timer | `node --test tasks/redesign/checks/map-saved.test.mjs`: `tests 12`, `pass 9`, `fail 3` | Same command: `tests 12`, `pass 12`, `fail 0` |
| Selecting a second pin clears the earlier ring | `node --test --test-name-pattern='second pin' tasks/redesign/checks/map-look-v2-behavior.test.mjs`: `tests 1`, `pass 0`, `fail 1`; assertion `true !== false` | `node --test tasks/redesign/checks/map-look-v2-behavior.test.mjs`: `tests 7`, `pass 7`, `fail 0` |
| Source names are available even without a matching place mention | `go test ./internal/adapters/httpserver/ -run TestPlacePanel_HasExistingWriterSourceNamesForItineraryCard`: missing Earth Trekkers source attributes, `FAIL weatherservice/internal/adapters/httpserver 0.607s` | `go test ./internal/adapters/httpserver/ -run 'TestPlacePanel\|TestIndex_BrandAndSearchField'`: `ok weatherservice/internal/adapters/httpserver 0.374s` |
| Keyboard focus reaches Hours after Directions in the phone sheet | `node --test --test-name-pattern='Hours summary' tasks/redesign/checks/map-look-v2-behavior.test.mjs`: one intended failure, “Tab from Directions must reach the following Hours summary” | Same command: one test, one pass, zero failures |

Six additional runtime tests in `map-look-v2-behavior.test.mjs` first ran green after corrections
had already landed. A retrospective negative control replayed the original working-tree
`map-panel.js`, reconstructed from HEAD and the captured pre-task diff, without changing
production files:

```sh
env MAP_PANEL_SOURCE=/tmp/map-look-v2-baseline-panel.js node --test --test-name-pattern='writer card|first pin callout' tasks/redesign/checks/map-look-v2-behavior.test.mjs
```

That replay reported four tests, two passing and two failing: the original writer card showed
“Mentioned by 3 writers” instead of itinerary precedence, and the original initial popup lacked
the photo. The current-file run
`node --test tasks/redesign/checks/map-look-v2-behavior.test.mjs` passed all six tests. This is
retrospective negative-control evidence, not chronological pre-edit RED. The tests cover actual
itinerary QIDs, blog-name deduplication, mention/none fallback, callout photos and omission,
the callout panel action, and sheet dragging.
The seventh test, for clearing the previous selection ring, was then added and observed failing
before its fix; its RED/GREEN evidence is chronological.
An eighth source-name precedence test then passed. Removing only the source-name preference
in a temporary copy produced its intended failure (“Alpha Travels, b.example” rather than
“Official Alpha, Official Beta”); this is a retrospective mutation control.

Final unit verification: `node --test tasks/redesign/checks/*.test.mjs` reported `tests 142`,
`pass 142`, `fail 0`. The saved-place file now has 14 passing tests, including cross-city metadata
and waiting through partial map responses. The compass and palette command
`node --test tasks/redesign/checks/writers-map-core.test.mjs tasks/redesign/checks/map-pins.test.mjs`
passed 65 tests, including numeric icon and text contrast. Those added tests were verification
after the initial structural RED, rather than independently observed pre-edit failures.

Retained command transcripts and output are in [the logs directory](artifacts/map-look-v2/logs/),
including [implementation evidence](artifacts/map-look-v2/logs/map-look-v2-implementation-evidence.md).

## Guarantees

| What is guaranteed | Test name | Level | Result |
| --- | --- | --- | --- |
| Brand accessible name, wordmark and search form | `TestIndex_BrandAndSearchField` | Integration | Pass |
| Summary/description and optional panel blocks, directions and post cards | `TestPlacePanel_*` | Integration | Pass |
| Existing allowlist names are available for itinerary cards | `TestPlacePanel_HasExistingWriterSourceNamesForItineraryCard` | Integration | Pass |
| Itinerary QIDs take precedence; names deduplicate; mentions/none fallback | `map-look-v2-behavior.test.mjs` writer card cases | Unit with fake DOM | Pass |
| Initial callout photo, area, action and missing-photo handling | `map-look-v2-behavior.test.mjs` first pin callout cases | Unit with fake DOM | Pass |
| Only the latest selected pin retains a ring | `map-look-v2-behavior.test.mjs` second pin case | Unit with fake DOM | Pass |
| Save, Undo, toast timers, list state and Escape focus | `map-saved.test.mjs` | Unit with fake DOM | Pass |
| Compass bearing reset and top-right control order | `writers-map-core.test.mjs` | Unit with fake engine | Pass |
| Pin/category palette and numeric contrast | `map-pins.test.mjs`, `writers-map-core.test.mjs` | Unit | Pass |
| Search, callout, panel, save/Undo/list, and Add to plan journeys | `pr8c-journeys.browser.mjs` | Browser at 375 and 1440 px | Pass |
| Sheet geometry, handle keyboard actions and Escape | `map-panel.browser.mjs` | Browser at 375 and 1440 px | Pass |
| Plan pins, popups, day toggles and regeneration | `map-dayplan.browser.mjs` | Browser, MapLibre and Leaflet at both widths | Pass |

## Coverage and final checks

`go test -cover ./internal/adapters/httpserver/...` returned:

```text
ok  weatherservice/internal/adapters/httpserver 1.154s coverage: 93.0% of statements
```

The preview package in `go test -cover ./internal/adapters/httpserver/... ./cmd/design-preview/...`
returned:

```text
ok  weatherservice/cmd/design-preview 1.166s coverage: 87.3% of statements
```

That combined run still found an obsolete HTTP assertion for the removed summary tab; after
fixing it, the standalone HTTP coverage command above passed. Both touched Go packages exceed 80%.

`PATH=/private/tmp/traveltab-lint-bin:$PATH make test lint build` exited 0. Excerpts:

```text
ok  weatherservice/internal/adapters/httpserver 3.140s
golangci-lint run ./...
0 issues.
go build -o bin/traveltab ./cmd/web
```

The temporary linter is the existing Go 1.27-compatible binary from map-first verification.
The gate also installed locked browser dependencies and ran all Go packages with the race detector.
Earlier failures were the new header test's missing weather fixture and obsolete expectations
for the saved slot and AI summary tab label; the final gate includes their corrections.

## L10/L11 completion: broken images, a flaky check, full run

Funchal screenshots in the preview showed broken images wherever a place photo failed to load
(the preview blocks external images; a real outage looks the same). Each fix was written
test-first:

| Behaviour | RED | GREEN |
| --- | --- | --- |
| Full-map card shows the destination photo and gives the map icon back when it fails | `node --test --test-name-pattern='full-map card' tasks/redesign/checks/writers-map-core.test.mjs`: `actual: undefined`, `expected: 'map-action-photo'` (the old code removed the icon) | `node --test tasks/redesign/checks/writers-map-core.test.mjs`: `tests 63`, `pass 63`, `fail 0` |
| Popup thumbnail and selected-pin photo hide themselves when they fail | `node --test --test-name-pattern='fails to load' tasks/redesign/checks/map-look-v2-behavior.test.mjs`: `pass 0`, `fail 1`, `actual: undefined`, `expected: true` | `node --test tasks/redesign/checks/map-look-v2-behavior.test.mjs`: `tests 11`, `pass 11`, `fail 0` |
| Panel photo hides its figure when it fails | `go test ./internal/adapters/httpserver/ -run TestPlacePanel_PhotoHidesItself -count=1`: `--- FAIL: TestPlacePanel_PhotoHidesItselfWhenItFailsToLoad` | `go test ./internal/adapters/httpserver/ -run 'TestPlacePanel' -count=1`: `ok weatherservice/internal/adapters/httpserver` |

**Flaky check fixed.** `map-dayplan.browser.mjs` failed in 2 of 4 runs: once `Timed out:
document.querySelectorAll('.plan-pin').length === 10`, once with focus on
`<h2 id="itinerary-title">` instead of the pin after Escape. Cause: `generate()` waited only for
the stops, but the planner focuses the itinerary heading after `htmx:afterSettle` plus a
`setTimeout 0`, so that focus could land after the check had moved on. `generate()` now also
waits for the heading to have focus. After the change: 5 runs, 5 passes, 4 `PASS` lines each
(MapLibre and Leaflet at 375 and 1440 px). No product code changed for this.

**Full run** on 2026-10-14, isolated headless Brave (Chrome/155) with SwiftShader WebGL:

| Check | Result |
| --- | --- |
| `PATH=/private/tmp/traveltab-lint-bin:$PATH make test lint build` | exit 0; `0 issues.` |
| `node --test tasks/redesign/checks/*.test.mjs` | `tests 145`, `pass 145`, `fail 0` |
| `go test -cover ./internal/adapters/httpserver/... ./cmd/design-preview/...` | 93.0% and 87.3% |
| `http-smoke`, `browser-smoke`, `interaction-regressions`, `guides`, `destination-photos` | exit 0 |
| `hero-phone`, `map-itineraries`, `map-panel`, `pr7-acceptance`, `pr7-integration`, `pr8c-journeys`, `writers-map-core`, `writers-map-layout`, `writers-map-preview` (`*.browser.mjs`), `vendor-assets` | exit 0 |
| `map-dayplan.browser.mjs` | exit 0 after the fix above (5/5) |
| `a11y-audit.mjs` | exit 0 run on its own; 31 `PASS`; 3145 text and 1072 icon/component pairs, 0 failing. In the long run it hit `CDP timeout: Page.captureScreenshot` before its first check (browser load, not a finding) |
| `scenarios.mjs` | `PASS: 2816 assertions across 110 screenshots, no JavaScript exceptions` |

**Against the mockups.** [Desktop](artifacts/map-look-v2/funchal-place-1440.png) and
[phone](artifacts/map-look-v2/funchal-place-375.png) with Funchal selected, next to
`../../phone.png` and `../../desktop.png`: brand, header bookmark, pill band, rounded map and panel
cards, dark control column, teardrop pins, selected callout, writer card, actions, link cards,
Hours, writers' posts, full-map card, half-height phone sheet and text bottom nav all match in
structure. Remaining visual differences, not fixed here:
- Writer card text is one purple line; the mockup has a dark bold title over a grey subtitle.
- Link cards have no icon tile ("W", map), and the ↗ sits under the text, not at the right.
- Locate, zoom and compass use text glyphs (`⌖`, `+`, `−`, `N`) rather than the mockup's icons.
- The phone pill row shows a thin scrollbar under it.
- The writers' itinerary picker (an existing map-itineraries control) sits on the map's top left.
- Fixture photos are external, so the preview shows the photo-less states.

## Owner follow-up: plan day picker, city banner above the map

Journeys: *as a traveller, I want the plan's day filter to look like the other map controls*;
*as a traveller, I want the page to open with the city's photo, name and weather.*
The owner chose to replace the creator footer with the city banner on the map page only, first
below the map and then, at their request, above it.

| Behaviour | RED | GREEN |
| --- | --- | --- |
| Day picker takes its look from CSS (dark row, plan orange for the selected day), not the old turquoise inline styles | `node --test --test-name-pattern='day control takes' tasks/redesign/checks/map-dayplan.test.mjs`: `pass 0`, `fail 1`, `no inline background on the group` | `node --test tasks/redesign/checks/map-dayplan.test.mjs`: `tests 9`, `pass 9`, `fail 0` |
| Day picker is a row at the map's top left, beside the control column (44 px targets would overflow the phone map as a column) | Same file: `actual: 'top-right'` | Same command: 9/9 |
| Map page renders the destination banner, with photo, `h2` title and weather; one `h1` | `go test ./internal/adapters/httpserver/ -run TestWriterMap_DestinationBannerFollowsTheMap -count=1`: `"100" is not less than "-1"`; no `destination-title`; no `weather-metrics` | Same command: `ok`; the package then passed 25 of 25 runs |
| The banner comes before the map | `go test ./internal/adapters/httpserver/ -run TestWriterMap_DestinationBannerPrecedesTheMap -count=1`: `--- FAIL` (`slice bounds out of range [:-1]`: no map after the banner) | Same package: `ok weatherservice/internal/adapters/httpserver` |
| On the map page the footer is hidden and the banner sits between the header and the map; scrolled to, the map still fills a screen (phone: down to the bottom nav) | `CDP_PORT=9227 node tasks/redesign/checks/writers-map-layout.browser.mjs`: first `the creator footer is replaced on the map page`, `"footer":"block"`; after the move, canvas bottom `910.58` below the nav at `835` | Same command: `PASS` MapLibre and Leaflet at 375 and 1440 px, `city banner above a full-height map` |
| "Your plan" stop popup is a card like `Elegant Travel Itinerary Map UI.png`: serif name, then Day / Stop / distance rows (with "from previous" on its own line), then "View details →"; kind and description stay in the details panel | `node --test tasks/redesign/checks/map-dayplan.test.mjs`: `fail 1`, `The input did not match the regular expression /\bplan-popup\b/` | Same command: `tests 9`, `pass 9`; `map-dayplan.browser.mjs`: `PASS` MapLibre and Leaflet at 375 and 1440 px |
| With the banner above the map, the "Added to your saved places · Undo" toast stays on screen: fixed to the viewport (centred on desktop, above the phone nav) instead of the map's bottom edge | `pr8c-journeys.browser.mjs` at 1440 px: `Undo toast is visible and clickable above the phone sheet` (toast at y=1141 in a 900 px window) | Same command: `PASS` Funchal 375 and 1440 px |

The new Go test first failed 1 run in 8 for another reason: other tests cache Lisbon, the only
city with a curated photo, in the shared store. It now uses its own SQLite store in a directory it
cleans up itself (the server's background cache save can outlive the test, which made
`t.TempDir` cleanup fail once in 20 runs). The check scrolls with `behavior: 'instant'`, because the site scrolls smoothly, and
scrolls to the map's exact top rather than with `scrollIntoView`, which stops
`--tt-anchor-offset` short (`scroll-padding-top`). `browser-smoke.mjs` failed once in a full run
(`Timed out: window.scrollY === 0` after planning, which smooth-scrolls to the itinerary heading);
its screenshot helper now re-requests the top until it holds, and passed 5 of 5 runs.

## Known limits

The map remains a dark vector map. The screenshots use deterministic preview data and do not
establish live map density, device performance or provider availability. The feature remains
subject to the existing writers' map flags.

The plan defers an internal full-screen map, official websites and street addresses, post
thumbnails, computed opening status, photo galleries, pill dropdowns and additional filters.
Profile avatars, ratings, prices, cultural-highlight chips, notes and post excerpts are omitted
as decided. The external full-map card retains its existing destination URL.
