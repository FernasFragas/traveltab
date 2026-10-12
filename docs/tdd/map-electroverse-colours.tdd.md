# Electroverse-like map colours

**Source:** owner request 2026-10-12 (make the 3D map colours more like Electroverse, Lisbon view). Under M7 in `tasks/map-3d-todo.md`.

**Journeys**
- As the owner, I want the writers' map to use the navy palette of Electroverse, so that it looks like the map I like.
- As the owner, I want a custom `data-map-style-url` left alone, so that a replacement style keeps its own colours.
- As a traveller, I want pins and routes still readable on the new base.

## Reference sample
Electroverse map at 38.73133,-9.26065,z11.4 (and z16.2 Lisbon centre for buildings), isolated headless Brave (CDP 9261, temp profile). Colours only were read from screenshot pixels (mode of each area); no style JSON, tiles, fonts or code. Screenshots: `scratchpad/ev/electroverse-{1440,375,z16-1440}.png`.

| Element | Electroverse sample | Our override |
| --- | --- | --- |
| Land | `#18263b` | `#18263b` (background, residential, ice) |
| Water | `#000e29` | `#000e29` (water, waterway) |
| Park / wood | `#203838` (z16 `#354b44`) | `#203838` (wood pattern removed) |
| Minor roads | `#404860` | `#404860` |
| Major roads | `#4c576c` | `#4c576c` |
| Motorways, runways (z16 streets `#5a6274`) | `#5a6274` | `#5a6274` |
| Buildings | `#2c3044` to `#46414d` | `#2c3044` to `#46414d` (2D and 3D) |
| Place labels | `#e0e0e0`, halo `#101820` | same |
| Street labels | not sampled | `#9aa3b5`, halo land |

## Behaviours
### 1. Palette applied on style.load, default style only
`node --test tasks/redesign/checks/writers-map-core.test.mjs`
RED: `actual: [], expected: [ 'water' ]` (no `setPaintProperty` calls) and the palette test failing on `paints.background`. GREEN: 98 pass in all `*.test.mjs`.
- Custom `data-map-style-url`: zero paint calls. Repeated `style.load`: same calls again, still one `buildings-3d`. Layer ids the style lacks are skipped.
- A second RED/GREEN: the wood layer's `fill-pattern` hid its colour (flat green missing at Lisbon); RED `actual: undefined, expected: null`, GREEN after `'fill-pattern': null`.
### 2. Fixture style has the palette
`go test ./cmd/design-preview/ -run Electroverse`
RED: `land background-color = #0c0c0c, want #18263b` (and water, roads, building). GREEN: ok. Coverage `cmd/design-preview` 86.8%. The old dark-luminance test limit for land moved 0.08 to 0.2 (navy land has luminance 0.14).
### 3. Browser pixels
`writers-map-core.browser.mjs`: top-left canvas pixel within 6 per channel of land, and land and water each over 5% of the canvas. Result: PASS maplibre and leaflet at 375 and 1440 px, console clean. Measured pixels were exactly `(24,38,59)` and `(0,14,41)`. (No RED run: written after the fixture change.)

## What is guaranteed
| Guarantee | Test | Level | Result |
| --- | --- | --- | --- |
| Palette on default style load | `applies the Electroverse-like palette...` | unit | PASS |
| Absent layer ids skipped | `leaves a layer id the style does not have alone` | unit | PASS |
| Custom style URL untouched | `applies no palette overrides to a custom style URL` | unit | PASS |
| Repeated style.load stable | `running style.load again reapplies...` | unit | PASS |
| Fixture colours | `TestPreview_MapStyleUsesElectroversePalette` | unit | PASS |
| Canvas land/water pixels | `writers-map-core.browser.mjs` | integration | PASS |
| Live style background `#18263b`, one `buildings-3d` | `writers-map-imagery-live.mjs` | live | PASS |

## Contrast (WCAG, pin or line colour vs base)
| Colour | Land `#18263b` | Water `#000e29` | Park `#203838` | Road `#4c576c` | Old land `#0c0c0c` |
| --- | --- | --- | --- | --- | --- |
| Writer pin `#f35a2c` | 4.57 | 5.76 | 3.75 | 2.19 | 5.88 |
| Notable pin `#2685ee` | 4.11 | 5.18 | 3.37 | 1.96 | 5.28 |
| Plan pin `#2e7d32` | 2.97 | 3.74 | 2.43 | 1.42 | 3.82 |
| Itinerary pin `#7b3fb8` | 2.35 | 2.96 | 1.93 | 1.12 | 3.02 |
| Itinerary line `#b794f6` | 6.22 | 7.84 | 5.09 | 2.97 | 7.99 |
| Route line `#ff8a65` | 6.58 | 8.29 | 5.39 | 3.14 | 8.45 |
| Day-plan line `#a64027` | 2.45 | 3.09 | 2.01 | 1.17 | 3.15 |

Below 3:1 on land: plan pin, itinerary pin and the day-plan default line (it was 3.15 on the old land). Pin colours were not changed (owner decision). Every pin has a white 2 px ring, so the pin as a whole still stands out; the figures are for the fill alone. Buildings (`#2c3044` to `#46414d`) are 1.1 to 1.9 on land, a quiet backdrop as before.

## Verification run (Brave headless, SwiftShader, 375 and 1440 px, both engines)
- `node --test tasks/redesign/checks/*.test.mjs`: 98 pass.
- PASS: writers-map-core, map-panel, map-dayplan, map-itineraries, writers-map-layout, pr7-acceptance, pr7-integration, hero-phone. `writers-map-preview` is flaky at 1440 maplibre (Porto pin list empty right after the swap): it failed 4 of 7 runs with the palette code and 1 of 2 with the palette call removed, so it is not from this change; it passed fully once (4 PASS).
- `a11y-audit.mjs`: 31 passed, 0 failed. `scenarios.mjs`: 2816 assertions pass.
- `writers-map-imagery-live.mjs` (live OpenFreeMap, owner-approved): PASS, 31 map responses, about 1.2 MB per first view.
- `make test build`: pass.
- Side by side (scratchpad `ev/`): `side-by-side-1440.png`, `side-by-side-375.png`, from `electroverse-{1440,375}.png` and `ours-{1440,375}.png`; ours flat (pitch 0) at 38.73133, -9.26065, z11.4.

## Known gaps
- OpenFreeMap has fewer place labels and green areas around Lisbon than Electroverse (Mapbox data), so the look matches in colour, not in label density. Roads are a little heavier.
- Electroverse's UI chrome (purple header, charger pins) is not copied; colours of the basemap only.
- `make lint` was not run.
