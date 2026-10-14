# Map-first destination page: TDD evidence

Source: `tasks/map-first-plan.md` and `tasks/map-first-todo.md` M1–M9. The intended journeys are exploring a full-height map, distinguishing TravelTab places from the base map, opening a place without leaving the map, adding it to a plan, and using the map on a phone with a peek card and sheet.

## Behavior evidence

| Behavior | RED command and observation | GREEN command and observation |
| --- | --- | --- |
| Default dark style hides generic POIs, keeps labels and raises untagged buildings | `node --test tasks/redesign/checks/writers-map-core.test.mjs`: 3 intended failures for 0 m fallback, old background and visible POI | Same command: 56/56 pass after M1 |
| Preview style includes a generic POI fixture | `go test ./cmd/design-preview/ -run TestPreview_MapStyleHidesGenericPOIs`: failed because the POI fixture layer was absent | `go test ./cmd/design-preview/ -run 'TestPreview_(MapStyle|ServesALocalMapStyle)'`: pass |
| Tab views and product name | `go test ./internal/adapters/httpserver/ -run 'TestTripPage_HasATitleAndDescription|TestTripPage_UnplannedPageStillHasATitle|TestMainPage_KeepsItsPlainTitle|TestICS_OneEventPerStopAtTheDefaultTimes' -count=1`: old title and `DESCRIPTION:Part of your TravelTab trip` failed | Same command: `ok weatherservice/internal/adapters/httpserver` |
| Planner OOB update preserves the tab shell | `go test ./internal/adapters/httpserver/ -run TestRedesignPlanResponseRoots -count=1`: expected `itinerary-results`, got `itinerary`; Stays marker and hidden state missing | Same command: `ok weatherservice/internal/adapters/httpserver` |
| Planner keeps the Itinerary URL | `NAVIGATION_PATH=/tmp/traveltab-m2-before-plan-navigation.js node tasks/redesign/checks/hero-phone.browser.mjs`: at 375 px, actual hash `''`, expected `#itinerary` | `node tasks/redesign/checks/hero-phone.browser.mjs`: 375 and 1440 px passed with direct hash, tab switches, persistent map, Back and plan hash; console clean |
| Map controls | `node --test tasks/redesign/checks/writers-map-core.test.mjs`: 3 intended control failures; `node --test tasks/redesign/checks/map-layers.test.mjs`: `createMenu` absent | Core tests 59/59 and layer tests 11/11 pass after M4 |
| Planner returns to the map to show a stop | `node --test tasks/redesign/checks/map-dayplan.test.mjs`: View on map had `actual: '#itinerary'`, `expected: '#overview'` | Same command: `pass 8`, `fail 0` |
| New search opens the new destination's Overview | `CDP_PORT=9228 node tasks/redesign/checks/hero-phone.browser.mjs`: new search retained the previous tab | Same command: 375 and 1440 px passed with the Overview visible |
| History restoration keeps the selected panel usable | `PREVIEW_URL=http://127.0.0.1:18087 CDP_PORT=9228 node tasks/redesign/checks/interaction-regressions.mjs`: HTMX `afterSwap` tried to process a missing panel target | Same command: passed after guarding the target |
| Phone map status leaves the layer controls usable | `CDP_PORT=9227 node tasks/redesign/checks/writers-map-layout.browser.mjs`: `map status covers layer controls`, with status `y=142–200` and controls `y=136–244` at 375 px | Same command: MapLibre and Leaflet passed at 375 and 1440 px after moving the status below the map controls |
| Map and guide heading hierarchy | `CDP_PORT=9227 node tasks/redesign/checks/map-panel.browser.mjs`: `0 !== 1` for map-page `h1`; `go test ./internal/adapters/httpserver/ -run TestGuideCard_ShowsTheReviewedGuideWithItsAttributionOnEveryRoute -count=1`: expected guide `h2`, found `h3` | Same browser command: 375 and 1440 px passed; same Go command: `ok weatherservice/internal/adapters/httpserver` |
| Phone sheet stays above native map markers | `CDP_PORT=9230 OUTPUT_DIR=/private/tmp/traveltab-map-first-sheet-red-webgl node tasks/redesign/checks/pr8c-journeys.browser.mjs`: `map pins must stay behind the phone place sheet` | Same command after giving the map canvas its own lower stacking context: Funchal 375 and 1440 px passed |

The agents' RED excerpts for the layout, panel and cluster changes were not retained in this
integration record. Their GREEN checks and runtime observations follow; this is a TDD evidence gap.

## Guarantees

| What is guaranteed | Test name | Level | Result |
| --- | --- | --- | --- |
| POI visibility and building fallback | `writers-map-core.test.mjs` style cases | Unit | Pass |
| Local preview map style contains the generic POI case | `TestPreview_MapStyleHidesGenericPOIs` | Integration | Pass |
| Brand title and ICS description | Trip page and ICS focused tests above | Integration | Pass |
| Planner response keeps tab shell | `TestRedesignPlanResponseRoots` | Integration | Pass |
| Hash views, Back and mounted map | `hero-phone.browser.mjs` | Browser | Pass |
| Locate, 2D/3D and shared layer menu | `writers-map-core.test.mjs`, `map-layers.test.mjs` | Unit | Pass |
| Desktop full-height map and 380–420 px panel; phone map fits above navigation | `writers-map-layout.browser.mjs` | Browser, both engines at 375 and 1440 px | Pass |
| Place panel, phone peek and sheet, and action buttons | `map-panel.test.mjs`, `map-panel.browser.mjs` | Unit and browser at 375 and 1440 px | Pass |
| Category icons, native writer clusters, and selected/plan photo callouts | `map-pins.test.mjs`, `writers-map-core.test.mjs`, `pr7-integration.browser.mjs` | Unit and browser, both engines at 375 and 1440 px | Pass |
| Funchal search → pin → panel → Add to plan → Itinerary shows the stop | `pr8c-journeys.browser.mjs` | Browser at 375 and 1440 px | Pass |
| Planned stop popup, Escape, Details and View on map | `map-dayplan.browser.mjs` | Browser, both engines at 375 and 1440 px | Pass |
| Preview navigation, layout, controls and console remain usable | `browser-smoke.mjs`, `hero-phone.browser.mjs`, `writers-map-core.browser.mjs`, `writers-map-preview.browser.mjs`, `pr7-acceptance.browser.mjs` | Browser | Pass |
| All named offline preview data variants and routes survive the tab layout | `scenarios.mjs` | Browser at 375 and 1440 px | 2816 assertions, 110 screenshots, no JavaScript exceptions |
| Live dark basemap and 3D buildings load in both cities | Temporary `startPreview({ liveTiles: true })` browser capture at zoom 13 and 16 | Browser, desktop | Four captures; `loaded: true`, no browser errors |

`node --test tasks/redesign/checks/*.test.mjs` returned 122 passing tests and no failures.
`go test -cover ./internal/adapters/httpserver/... ./cmd/design-preview/...` returned 93.2% and 87.0% statement coverage respectively. `go vet ./...` and `make build` passed.
`PATH=/private/tmp/traveltab-lint-bin:$PATH make test lint build` passed: all Go packages
passed the race tests, `golangci-lint run ./...` reported `0 issues.`, and the web build
completed. The temporary linter was built with Go 1.27 because the machine's installed
`golangci-lint` v2.4.0 was built with Go 1.26.4 and rejected this module's Go 1.27.2 target.

Fixture screenshots: [phone map](artifacts/map-first/funchal-map-375.png),
[desktop map](artifacts/map-first/funchal-map-1440.png),
[phone panel](artifacts/map-first/funchal-panel-375.png),
[desktop panel](artifacts/map-first/funchal-panel-1440.png), and itinerary views in the same directory.
The live-tile preview was captured at [Funchal z13](artifacts/map-first/live-funchal-z13.png),
[Funchal z16](artifacts/map-first/live-funchal-z16.png),
[Lisbon z13](artifacts/map-first/live-lisbon-z13.png) and
[Lisbon z16](artifacts/map-first/live-lisbon-z16.png). The temporary browser check called
`startPreview({ scenario: 'writers-map', liveTiles: true })`, set zoom to 13 and 16, and observed
`loaded: true` with no browser errors for each capture. These checks show the live OpenFreeMap
style and 3D buildings; the place data remains the deterministic preview fixture.
The local vector tile fixture contains a small Funchal scene, so these screenshots do not establish
the density or performance of live Funchal and Lisbon map tiles. The fixture also reports a partial
writer source, which displays the source-status message.

## M7 full check run

Run on 2026-10-14 against an isolated headless Brave (Chrome/155) on `CDP_PORT=9227`, started
with `--use-angle=swiftshader --enable-unsafe-swiftshader`. Without those flags MapLibre never
creates a canvas, and `a11y-audit.mjs` and `map-dayplan.browser.mjs` time out waiting for
`.maplibregl-canvas`; with them both pass, so that was the environment, not the code.

Four older checks still asserted the pre-map-first layout. Each failed on the old expectation,
which was updated to the decided design (no product code changed):

| Check | RED (old expectation) | Update |
| --- | --- | --- |
| `http-smoke.mjs` | `The input did not match /\bid="itinerary"/` on `<div id="itinerary-results" hx-swap-oob="outerHTML">` | Plan response roots are `trip-card`, `itinerary-results`, `stays` (as in `docs/design.md`) |
| `http-smoke.mjs` | `no photo for an unlisted city: weather stays usable` (`class="weather-metrics"` exact match; the class became `weather-card weather-metrics` in 5d237cf, before this work) | Match `weather-metrics` as one of the classes |
| `guides.mjs` | `the card follows the overview grid inside #overview`, then `after planning: card visible` and `after Forward: card visible` | Guide follows `.tt-overview-map`; after planning and Forward the Itinerary tab is shown, so the guide must stay, unchanged, in the hidden Overview |
| `vendor-assets.mjs` | `Cannot read properties of null` for `.tt-overview-planner` | Dropped the "planner above map on phone" assertion; the planner now lives in the Itinerary tab |

GREEN, each run on its own:

| Check | Result |
| --- | --- |
| `node --test tasks/redesign/checks/*.test.mjs` | 122 pass, 0 fail |
| `http-smoke.mjs`, `browser-smoke.mjs`, `interaction-regressions.mjs`, `guides.mjs`, `destination-photos.mjs` (`PREVIEW_URL=http://127.0.0.1:8087`) | exit 0 |
| `a11y-audit.mjs` | exit 0; contrast: 3054 text and 1186 icon/component pairs, 0 distinct failing |
| `hero-phone`, `map-dayplan`, `map-itineraries`, `map-panel`, `pr7-acceptance`, `pr7-integration`, `pr8c-journeys`, `writers-map-core`, `writers-map-layout`, `writers-map-preview` (`*.browser.mjs`) | exit 0 |
| `vendor-assets.mjs` | `PASS 375px` / `PASS 1440px: local CSS, fonts, htmx, planning, no overflow` |
| `scenarios.mjs` | `PASS: 2816 assertions across 110 screenshots, no JavaScript exceptions` |
| `PATH=/private/tmp/traveltab-lint-bin:$PATH make test lint build` | all packages `ok`; `0 issues.`; build ok |
| `go test -cover ./internal/adapters/httpserver/... ./cmd/design-preview/...` | 93.2% and 87.0% |

Not run: `destination-photos-live.mjs` and `writers-map-imagery-live.mjs` (live network, outside
`make test`). Running the checks regenerated their screenshots under `tasks/artifacts/` and
`docs/tdd/artifacts/dependabot-vendor-assets/`, which now show the map-first layout.

## Owner follow-up: no clear cross, rounder corners, city photo

Journeys: *as a traveller, I want to see the city before I pick a place*; *as the owner, I want no
grey cross in the search box and corners that match the mockup.*

| Behaviour | RED | GREEN |
| --- | --- | --- |
| Default panel shows the city photo and credit; omitted when there is none | `go test ./internal/adapters/httpserver/ -run 'TestWriterMap_DefaultPanel' -count=1`: `does not contain "class=\"map-default-photo\""` | Same command: `ok weatherservice/internal/adapters/httpserver` |
| No grey cross in the filled search box (our × removed; the browser's `::-webkit-search-cancel-button` hidden) | `CDP_PORT=9227 node tasks/redesign/checks/hero-phone.browser.mjs`: `375: no grey cross in the filled search box`, actual `{ clear: true, nativeCancel: 'block' }`; after removing our ×, `{ clear: false, nativeCancel: 'shown' }` | Same command: `PASS 375` and `PASS 1440: … new search, no clear cross, no overflow`; `PASS: console clean` |

Corners (CSS only, checked in screenshots at 375 and 1440 px): photos and search 12 px, buttons and
map controls 10 px, layer chips are pills, and the place photo is inset in the panel instead of full
bleed. The browser had cached the old stylesheets; screenshots were retaken after
`Network.clearBrowserCache`.

**Against the mockup** ([reference](<../../TravelTab Funchal Map Experience.png>),
[desktop panel](artifacts/map-first/funchal-panel-1440.png)): header, tabs, full-height map,
docked panel order, tabs, actions and the writers' posts list match. Differences are the decided
ones (dark vector map instead of satellite, wordmark name) and fixture data: the Funchal fixture
has few places and no hero photo for the selected place, so the screenshots are sparser than the
mockup. Writer avatars and post thumbnails are deferred by the plan.

## Known gaps

The implementation uses a dark vector map rather than photorealistic tiles. Routes are straight
segments, the Leaflet fallback uses a different base style, and Photos/Nearby tabs, Save trip,
post thumbnails and PMTiles remain deferred by the source plan. The owner's mid-range phone
performance check (M8) remains open.
