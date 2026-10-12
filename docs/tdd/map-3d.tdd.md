# MapLibre integration (M6–M7)

Source: [map-3d-todo.md](../../tasks/map-3d-todo.md), M6 and M7. The M1–M2 core and fixture evidence is in [map-3d-m1-m2.tdd.md](map-3d-m1-m2.tdd.md); the layer RED/GREEN records are in [M3](map-3d-m3.tdd.md), [M4](map-3d-m4.tdd.md) and [M5](map-3d-m5.tdd.md).

## Journeys

- As a visitor with WebGL, I want the 3D map, its pins, panel, day plan and itineraries to work at phone and desktop widths.
- As a visitor without WebGL, I want the same controls and saved-place flow on the Leaflet fallback.
- As a keyboard user, I want named and focusable map controls and pins.
- As an operator, I want the live basemap and terrain hosts to load with visible attribution, and a documented self-hosting path if a host fails.

## RED → GREEN evidence

| Behaviour | Command | RED excerpt | GREEN excerpt |
| --- | --- | --- | --- |
| Layout check waits for a usable map after each viewport navigation | `CDP_PORT=9227 node tasks/redesign/checks/writers-map-layout.browser.mjs` | `Timed out: document.querySelector('[data-map-canvas] .maplibregl-canvas') ...` after the first width | `PASS maplibre 375px`, `PASS maplibre 1440px`, `PASS leaflet 375px`, `PASS leaflet 1440px` after `Page.bringToFront` (agent's isolated CDP 9233 run) |
| Map controls and pins have accessible names and keyboard focus | `CDP_PORT=9227 AUDIT_ONLY=map node tasks/redesign/checks/a11y-audit.mjs` | `FAIL writers-map-leaflet@375` and `@1440`: one unnamed decorative tile image was counted as a control | `PASS writers-map-maplibre@375`, `@1440`, `PASS writers-map-leaflet@375`, `@1440`; `5 passed, 0 failed` |
| Live default hosts load on both viewports | `CDP_PORT=9227 node tasks/redesign/checks/writers-map-imagery-live.mjs` | `AssertionError`: second MapLibre view still used `fixture-map/tiles` when the check remounted the existing map | `PASS maplibre 375px`, `PASS maplibre 1440px` for OpenFreeMap and Mapterhorn; `PASS leaflet 375px`, `PASS leaflet 1440px` for EOX |

## Integration checks

| What is guaranteed | Check | Kind | Result |
| --- | --- | --- | --- |
| M3 pins and panel on both engines at 375 and 1440 px | `map-panel.browser.mjs` | browser | PASS (see M3 report) |
| M4 generate, regenerate, toggle, clear and View on map on both engines and widths | `map-dayplan.browser.mjs` | browser | PASS (see M4 report) |
| M5 itinerary switching and style reload on both engines and widths | `map-itineraries.browser.mjs` | browser | PASS (see M5 report) |
| Writer data, base-only and unavailable swaps, keyboard list and one canvas | `writers-map-preview.browser.mjs` | browser | PASS: four engine/width combinations |
| Map card layout, controls, tooltip and destination swap | `writers-map-layout.browser.mjs` | browser | PASS: four engine/width combinations |
| Chips, panel Save, persisted save and plan request | `pr7-integration.browser.mjs` | browser | PASS: four engine/width combinations |
| Regenerate without duplicate pins, clear, blocked storage and View on map | `pr7-acceptance.browser.mjs` | browser | PASS: both engines, both widths, blocked storage |
| Named, focusable pins and named toggle chips without overflow | `AUDIT_ONLY=map a11y-audit.mjs` | browser accessibility tree | PASS: four engine/width combinations |
| OpenFreeMap style, Mapterhorn terrain and EOX fallback imagery with visible credit | `writers-map-imagery-live.mjs` | live browser | PASS: four engine/width combinations |

The browser checks use an isolated headless Brave CDP session and offline preview fixtures except for the explicitly live provider check. The map audit ran without the optional axe-core injection. M1–M2 Go coverage reached 86.8% for `cmd/design-preview`; M3–M5 JavaScript checks have no Go package coverage target. `PATH=/private/tmp:$PATH make test lint build` passed after integration: all Go packages passed, lint reported `0 issues.`, and `go build -o bin/traveltab ./cmd/web` completed. The full accessibility audit and first-view comparison are recorded below; the owner phone check is pending.

## First view and owner device

**First-view measurement (2026-10-11).** Headless Brave on CDP, software WebGL (SwiftShader), cache disabled, 1440x900. Preview `-scenario writers-map -live-tiles` (production defaults, live hosts), page `/trip/madeira-pt`, map scrolled into view, waited for tiles drawn (`areTilesLoaded()` for MapLibre, loaded tiles for Leaflet) plus 3 s. The forced Leaflet run disables `webgl`/`webgl2` contexts before page load, so the real fallback path runs. Script: `firstview.mjs` in the session scratchpad (CDP `Network.responseReceived` + `loadingFinished`, `encodedDataLength`). Two runs each; bytes differ by under 100 bytes.

| Engine | Map requests | Map bytes | Breakdown | Whole page, external hosts | Spike baseline (whole page) |
| --- | --- | --- | --- | --- | --- |
| MapLibre 3D | 20 | 1.12 MB | jsDelivr 2 / 288 KB; OpenFreeMap 6 / 154 KB; Mapterhorn 12 / 678 KB | 26 requests, 1.17 MB | 28 requests, 1.33 MB |
| Leaflet fallback | 14 | 0.19 MB | unpkg 2 / 47 KB; EOX 12 / 140 KB | 20 requests, 0.24 MB | 38 requests, 0.59 MB |

- "Map" counts only the map library and tile hosts. "Whole page, external" adds Google Fonts and YouTube thumbnails (6 requests, 55 KB), and excludes the local preview and `data:`/`blob:` URLs. The spike's whole-page figures included the app's own assets, which differ in the preview, so compare the map rows with the spike's map part, not the page totals.
- MapLibre is about 6x the Leaflet bytes on the map (1.12 MB against 0.19 MB), in line with the spike's "2 to 3x" on total page bytes. Mapterhorn terrain is 60% of the MapLibre bytes, as the spike said. Request counts are close to the spike (20 map + 6 fonts and thumbnails = 26 external, against 28 in the spike).
- Zero failed requests on both engines.

**Owner real-phone check: PENDING.** On a mid-range Android or iPhone, open the live site (or `go run ./cmd/design-preview -scenario writers-map -live-tiles` on the LAN) at `/trip/madeira-pt` with `WRITERS_MAP=1` and scroll to the map. Record:
1. Seconds from scrolling the map into view to sharp tiles and terrain.
2. Whether pan, pinch and tilt stay smooth (no stutter or blank frames).
3. Whether the Leaflet fallback appears by mistake (no 3D tilt, satellite look).

If it is too slow, a follow-up turns terrain off on small screens (`data-map-terrain-url=""`).

## M2. Offline map fixtures for the preview

Source: [map-3d-todo.md](../../tasks/map-3d-todo.md) M2; attributes from [writers-map-contract.md](../writers-map-contract.md) **Template hooks**. Browser evidence is in [map-3d-m1-m2.tdd.md](map-3d-m1-m2.tdd.md).

**Journeys**
- As a developer, I want the preview map to load with no network, so checks are deterministic.
- As the owner, I want production to keep the browser defaults, so `cmd/web` sets none of the new attributes.
- As a check author, I want to force the Leaflet fallback, so M6 can run every check on both engines.

**Server.** `SetWriterMapBasemap(styleURL, terrainURL, engine)` sets `data-map-style-url`, `data-map-terrain-url` and `data-map-engine`; each is rendered only when non-empty. `SetWriterMapTerrainOff()` renders `data-map-terrain-url=""` (the contract's "empty turns terrain off"). Brace characters in `data-map-terrain-url` are escaped by `html/template` to `%7b` and `%7d`, exactly like `data-map-tile-url`; the browser core restores them.

| Behaviour | RED (real output) | GREEN |
| --- | --- | --- |
| Attributes present when set, absent when not, terrain off | `go test ./internal/adapters/httpserver/ -run TestWriterMap_Shell`: `s.SetWriterMapBasemap undefined`, `s.SetWriterMapTerrainOff undefined` | `ok weatherservice/internal/adapters/httpserver` |
| Style, glyphs, sprite, vector tile, terrain, 404s | `go test ./cmd/design-preview/ -run 'TestPreview_'`: `status = 200, content type = "text/html; charset=utf-8", want "application/json"` (and the same for pbf, mvt, png); `/fixture-map/nope status = 200` | `ok weatherservice/cmd/design-preview` |
| Preview page carries style and terrain; Leaflet scenario; live tiles clear them | `missing data-map-style-url="http://127.0.0.1:8088/fixture-map/style.json"` | pass |

| Guarantee | Test | Level | Result |
| --- | --- | --- | --- |
| Every URL in the style is on the preview origin | `TestPreview_ServesALocalMapStyle` | integration | PASS |
| Glyph stub (`application/x-protobuf`), sprite JSON and PNG | `TestPreview_ServesGlyphsAndSprite` | integration | PASS |
| Funchal tile has water, 2+ roads, 3+ buildings with `render_height`; tiles elsewhere are empty | `TestPreview_ServesAVectorTileOverFunchal`, `..AwayFromFunchal` | integration | PASS |
| Terrain tile is 512 px, terrarium sea level (R=128, G=0, B=0) | `TestPreview_ServesAFlatTerrainTile` | integration | PASS |
| Bad paths return 404 | `TestPreview_RejectsUnknownOrMalformedMapAssets` | integration | PASS |
| `writers-map` sets style and terrain, no engine | `TestWriterMapPreview_SetsTheMapStyleAndTerrain` | integration | PASS |
| `writers-map-leaflet` adds `data-map-engine="leaflet"` | `TestWriterMapPreview_LeafletScenarioForcesTheFallback` | integration | PASS |
| `-live-tiles` sets none of the three | `TestWriterMapPreview_LiveTilesLeavesTheProductionBasemap` | integration | PASS |
| Attributes present when set, absent by default, terrain off | `TestWriterMap_Shell*` (3 tests) | integration | PASS |

**Fixtures** (generated in Go by `cmd/design-preview/mapstyle.go`, no binary files, no network). Served from the map origin (default `http://127.0.0.1:8088/`), CORS `*`:
- Style `/fixture-map/style.json`; vector source `tiles/{z}/{x}/{y}.mvt` (maxzoom 14), layers `water`, `transportation` (`class` primary/secondary), `building` (`render_height`, `render_min_height`).
- Glyphs `/fixture-map/glyphs/{fontstack}/{range}.pbf` (valid, empty; no text layers). Sprite `/fixture-map/sprite[@2x].json|png` (`{}` and a 1x1 PNG).
- Terrain `/fixture-map/terrain/{z}/{x}/{y}.png`: 512 px, **terrarium** encoding, flat at sea level.
- Vector tiles are computed from coordinates, so any z/x/y works; shapes sit around Funchal (32.65, -16.91).

**Force the fallback:** run the preview with `-scenario writers-map-leaflet`. It is `writers-map` plus `data-map-engine="leaflet"`; Leaflet still uses the local `fixture-tiles`. `-live-tiles` (with `writers-map`) clears style, terrain and engine so the production defaults apply.

**Coverage:** `go test -cover ./cmd/design-preview/ ./internal/adapters/httpserver/` gave 86.8% and 92.5%. Gap: glyphs have no glyph data, because the style has no text layers.

## Verification run (2026-10-11)

Independent re-run of M3 to M7 by a second session. Isolated headless Brave on CDP 9241 (own temp profile; the earlier session's browser on 9227 was not touched). Every browser check starts its own preview, so these ran against the offline fixtures. `PASS` below means the check printed one PASS line per engine and width and exited cleanly.

**Layer review (M3, M4, M5).** `grep` for `L.`, `window.L`, `maplibregl` in `map-panel.js`, `map-dayplan.js`, `map-itineraries.js`: no matches. Each layer reaches the map only through `context.draw` (`markers`, `clear`, `line`, `control`). Idempotence, replace-by-id and View on map are covered by the checks below.

**Gap found and fixed (Prove-It).** Flaky failure, 2 of 4 runs of `map-itineraries.browser.mjs` and 1 of 3 of `writers-map-core.browser.mjs`: `Error: Style is not done loading` thrown from `drawLine` (`writers-map.js`) when a layer drew a line while MapLibre was reloading its style.
- RED: new `writers-map-core.test.mjs` test "a line drawn while the style is reloading waits for style.load instead of throwing" (stub `addSource` throws while `styleLoading`): `node --test` failed with `Error: Style is not done loading.`
- GREEN: `drawLine` catches that error and leaves the line recorded; `style.load` calls `restoreLines`, which draws it. `node --test ...writers-map-core.test.mjs`: 30 pass, 0 fail.
- The itineraries browser check also raced: it waited for the old line, not for the reload. It now waits for `style.load` after `setStyle`. 5 consecutive runs: 4 PASS lines each. 5 consecutive `writers-map-core.browser.mjs` runs: 4 PASS lines each.

| Command (`CDP_PORT=9241`) | MapLibre 375 | MapLibre 1440 | Leaflet 375 | Leaflet 1440 |
| --- | --- | --- | --- | --- |
| `node --test tasks/redesign/checks/*.test.mjs` (73 tests) | PASS | PASS | PASS | PASS (engine-independent, stubbed) |
| `map-panel.browser.mjs` | PASS | PASS | PASS | PASS |
| `map-dayplan.browser.mjs` | PASS | PASS | PASS | PASS |
| `map-itineraries.browser.mjs` | PASS | PASS | PASS | PASS |
| `writers-map-core.browser.mjs` | PASS | PASS | PASS | PASS |
| `writers-map-layout.browser.mjs` | PASS | PASS | PASS | PASS |
| `writers-map-preview.browser.mjs` | PASS | PASS | PASS | PASS |
| `pr7-integration.browser.mjs` | PASS | PASS | PASS | PASS |
| `pr7-acceptance.browser.mjs` (7b, 7d incl. blocked storage, 7e, 7g) | PASS | PASS | PASS | PASS |
| `writers-map-imagery-live.mjs` (live OpenFreeMap, Mapterhorn, EOX) | PASS | PASS | PASS | PASS |
| `a11y-audit.mjs` (full, 39 states) | 27 passed, 0 failed; map rows `writers-map-maplibre` and `-leaflet` at 375 and 1440 all PASS | | | |
| `scenarios.mjs` | `PASS: 2816 assertions across 110 screenshots` | | | |
| `hero-phone.browser.mjs` | PASS at 375 (Lisbon, Porto) and 1440 | | | |

Notes: `pr7-acceptance` printed 22 PASS lines (both engines, both widths, blocked storage). The remaining Leaflet selectors in the browser checks (`.leaflet-pane`, `.leaflet-overlay-pane`) are inside Leaflet-only branches; MapLibre branches use `.maplibregl-canvas`, `[data-map-canvas]` and `getLayer`.

Go: `make test build` passed (all packages `ok`, `go build -o bin/traveltab ./cmd/web`); `go vet ./...` clean; `gofmt -l internal cmd` empty; `PATH=/private/tmp:$PATH make lint` used the golangci-lint binary in `/private/tmp` and reported `0 issues.`

