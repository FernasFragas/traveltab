# Dark 3D map

**Source:** `tasks/map-3d-plan.md`, owner decision 2026-10-12 (always dark; OpenFreeMap `dark` style; the core adds 3D buildings). Under M7 in `tasks/map-3d-todo.md`.

**Journeys**
- As a traveller, I want the map always dark, so that it matches the owner's chosen look.
- As a traveller, I want 3D buildings on the dark map, so that the city still reads in 3D.
- As the owner, I want a custom style that already has buildings left alone, so that none are doubled.

## Behaviours

### 1. Default style is the dark one
`node --test tasks/redesign/checks/writers-map-core.test.mjs`

RED: `actual: 'https://tiles.openfreemap.org/styles/liberty'`, `expected: 'https://tiles.openfreemap.org/styles/dark'`. GREEN: passes.

### 2. The core adds one dark building layer, once
- Same command. RED: `actual: 0`, `expected: 1` (no `fill-extrusion` layer after `style.load`). GREEN: passes.
- `buildings-3d`: source `openmaptiles`, source-layer `building`, minzoom 14, height from `render_height`, base from `render_min_height`, colour ramp `#2e3238` (0 m) to `#4a4f58` (80 m), opacity 0.85. Inserted before the first symbol layer after the style's `building` layer.
- No layer when the style has a `fill-extrusion` layer or no vector source; two more `style.load` events leave one layer.

### 3. Fixture style is dark
`go test ./cmd/design-preview/ -run MapStyleIsDark`

RED: `land colour #ece6d3 has luminance 0.90, want at most 0.08` (and water, roads, buildings). GREEN: land `#0c0c0c`, water `#1b1b1d`, secondary road `#2e2e31`, primary road `#4a4a4f`, buildings `#3d4047`. Coverage `cmd/design-preview` 86.8%.

### 4. Browser check on the dark fixture
`writers-map-core.browser.mjs` (new): a `fill-extrusion` layer exists and three canvas pixels (screenshot) have every channel at most 64. Result: PASS maplibre and leaflet at 375 and 1440 px, console clean. (Written after the fixture change, so no RED run; the pixel threshold sits far from the old light land `#ece6d3`.)

## What is guaranteed

| Guarantee | Test | Level | Result |
| --- | --- | --- | --- |
| Default style URL is OpenFreeMap dark | `draws a 3D MapLibre map with the dark default style...` | unit | PASS |
| One building layer after `style.load`, below labels | `adds one dark 3D building layer below the labels...` | unit | PASS |
| None when the style has fill-extrusion | `adds no building layer when the style already has...` | unit | PASS |
| None without a vector source | `adds no building layer when the style has no vector source` | unit | PASS |
| No duplicate on repeated `style.load` | `running style.load again does not duplicate...` | unit | PASS |
| Fixture style is dark | `TestPreview_MapStyleIsDark` | unit | PASS |
| Canvas dark, extrusion present, both engines | `writers-map-core.browser.mjs` | integration | PASS |
| Live dark style, one `buildings-3d`, background `rgb(12,...)` | `writers-map-imagery-live.mjs` | live | PASS |
| Leaflet fallback unchanged (EOX imagery) | live check leaflet 375/1440, existing fallback tests | unit, live | PASS |

## Contrast
Measured WCAG ratios (non-text 3:1 is the target for lines): itinerary line `#7b3fb8` is 3.02 on land `#0c0c0c` and 2.66 on water `#1b1b1d`; day-plan default line `#a64027` is 3.15 and 2.77. Buildings `#2e3238` to `#4a4f58` are 1.5 to 2.4 on land, which is meant to be a quiet backdrop. Pins, chips, popups and controls have their own light backgrounds, so DOM contrast is unchanged: `a11y-audit.mjs` map pins, chips and axe checks pass on maplibre, and its "below threshold" pairs are the same as before the change (min 1.91 baseline, 1.92 now; page elements outside the map).

## Verification run
- `node --test tasks/redesign/checks/*.test.mjs`: 77 pass.
- Browser checks on both engines, 375 and 1440 px (own CDP port, SwiftShader WebGL): writers-map-core, map-panel, map-dayplan, map-itineraries, writers-map-layout, writers-map-preview, pr7-integration, hero-phone PASS. pr7-acceptance passed on rerun (one first-run failure: a Mapterhorn terrain tile request landed in the "saving makes no requests" window, a timing flake from the map flying to the selected place).
- `scenarios.mjs`: PASS, 2816 assertions.
- `a11y-audit.mjs`: all pre-existing and dark-map checks pass; the run ends with `SyntaxError: Unexpected end of input` and two `writers-map-popup-leaflet` FAILs, which come from the concurrent popup work (it added those assertions to the audit), not from this change.
- `writers-map-imagery-live.mjs`: PASS, live requests to OpenFreeMap, Mapterhorn, EOX; about 1.15 MB over 28 to 31 map responses per first view. Screenshots (Funchal): `funchal-dark-z12-{375,1440}.png`, `funchal-dark-z16-{375,1440}.png` in the session scratchpad `dark/shots/`.
- `make test build`, `go vet ./...`, `gofmt -l internal cmd` clean; `make lint` with `PATH=/private/tmp:$PATH`: 0 issues.

## Known gaps
- Itinerary and default route line colours are set in `map-itineraries.js` and `writers-map.js` (not changed): contrast is just at 3:1 on land and under it on water.
- Real-phone check is still the owner's (M7).
