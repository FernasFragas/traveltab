# Writers' map in 3D (MapLibre) — Tasks

Source of truth for progress. Read [the plan](map-3d-plan.md) and [the spike results](map-3d-results.md) first.

**Status:** M0–M6 complete and re-verified 2026-10-11 (see the Verification run in `docs/tdd/map-3d.tdd.md`). M7 is done except the **owner real-phone check**, which is pending.

**How to read this file**
- Order: **M0 (owner review) → M1 and M2 in parallel → M3, M4, M5 in parallel → M6 → M7**.
- **Owns** = the only files the task may edit. Starts after PR 7 is merged, because it edits PR 7's map files.
- TDD per [AGENTS.md](../AGENTS.md) §5. Browser changes need checks plus a runtime check in `cmd/design-preview` (clean console, 375 and 1440 px).
- **Tests never hit the network:** the preview serves a local style, glyphs, sprite, vector and terrain fixtures.
- **Free sources only**, with no keys. No commits or pushes; the owner commits.

**User journeys**
- *As a traveller, I want a sharp 3D map of the destination, so that I can see the terrain and streets at any zoom.*
- *As a traveller on an old phone or a bad network, I still want a working map.*
- *As the owner, I want the map to keep working if a free tile host goes down.*

---

### M0. Contract change: the drawing API · **owner review**
- [x] Done — approved by the owner 2026-10-11 (`docs/writers-map-contract.md`, **Browser extension API** and **Template hooks**)

**Owns:** `docs/writers-map-contract.md`

- **Browser extension API:** add `context.draw` (markers, clear, line, control, fit, flyTo), `context.engine` (`"maplibre"` or `"leaflet"`), and the rule that layers must not call `maplibregl`/`L`.
- `travelmap:ready` `detail.map` becomes opaque (engine-specific). Layers use `context.draw`.
- Layers must be idempotent; the core re-adds lines after a style reload.
- **Fallback rules:** no WebGL, or a style/tile/terrain load error → Leaflet + EOX, max zoom 18.

**Verify:** owner approves the contract diff. *(Approved 2026-10-11.)*

### M1. Core: MapLibre engine, drawing API and Leaflet fallback · *after M0*
- [x] Done

**Owns:** `public/redesign/writers-map.js`, `public/redesign/writers-map-leaflet.js` (new: today's Leaflet path, zoom capped at 18), `public/redesign/writers-map.css`, `public/redesign/writers-map-layout.css` (`.leaflet-*` → engine-neutral selectors), `tasks/redesign/checks/writers-map-core.test.mjs`, `tasks/redesign/checks/writers-map-core.browser.mjs`, `views/index.go.tpl` (map script and style tags only)

- [x] RED `writers-map-core.test.mjs`: `context.draw` has the same shape on both engines (stubbed `maplibregl` and `L`).
- [x] RED: no WebGL → Leaflet engine, `maxZoom` 18.
- [x] RED: style load error → Leaflet engine, and layers re-added once.
- [x] RED: a style reload re-adds lines without duplicating markers.
- [x] GREEN:
  - lazy-load MapLibre 5.24.0 (UMD) only when the card is on screen
  - OpenFreeMap Liberty style, Mapterhorn terrain (512 px, no hillshade), pitch about 55°, bearing about −20°, zoom 12.3
  - keep the htmx swap, `mount`, `selectPlace`, `updatePlan`, and the three events
- [x] Browser check: MapLibre canvas present, and the Leaflet fallback when WebGL is disabled. Clean console.

**Verify:** `node --test tasks/redesign/checks/writers-map-core.test.mjs`; `writers-map-core.browser.mjs` at 375 and 1440 px.

### M2. Offline map fixtures for the preview · *parallel with M1*
- [x] Done

**Owns:** `cmd/design-preview/*` (fixture map assets and routes only), `cmd/design-preview/*_test.go`

- [x] RED `TestPreview_ServesALocalMapStyle`: style JSON whose sources point to the preview origin.
- [x] RED: glyphs, sprite, one vector tile and one terrain tile are served locally.
- [x] GREEN: a small fixture style (land, roads, a few buildings with `render_height`), glyph and sprite stubs, and a flat DEM tile. Keep the existing Leaflet fixture tiles for the fallback.
- [x] The preview sets the style/terrain URLs through the existing `data-map-tile-url` pattern (or new `data-map-style-url` / `data-map-terrain-url` attributes, as M0 specifies).

**Verify:** `go test -cover ./cmd/design-preview/` ≥80%.

### M3. Pins and place panel on the drawing API · *after M1, M2*
- [x] Done

**Owns:** `public/redesign/map-panel.js`, `public/redesign/map-panel.css`, `tasks/redesign/checks/map-panel.test.mjs`, `tasks/redesign/checks/map-panel.browser.mjs`

- [x] RED (test.mjs): the pin layer calls only `context.draw.markers`/`clear`, with no `L.*`.
- [x] GREEN:
  - writer pins (orange, with count) above notable pins (blue)
  - big places labelled
  - click selects the place and loads the panel
  - same look on both engines
- [x] Keyboard focus and titles on pins as today.

**Verify:** `node --test …map-panel.test.mjs`; `map-panel.browser.mjs` on both engines at 375 and 1440 px.

### M4. Day plan layer on the drawing API · *after M1, M2; parallel with M3*
- [x] Done

**Owns:** `public/redesign/map-dayplan.js`, `tasks/redesign/checks/map-dayplan.test.mjs`, `tasks/redesign/checks/map-dayplan.browser.mjs`

- [x] RED: green numbered markers and the day toggle use `context.draw.markers`/`control` only.
- [x] RED: generate, regenerate and clear leave no duplicates (as 7b acceptance).
- [x] GREEN, and keep **View on map** working (it clicks the chip).

**Verify:** `node --test …map-dayplan.test.mjs`; `map-dayplan.browser.mjs` on both engines.

### M5. Itinerary layer on the drawing API · *after M1, M2; parallel with M3, M4*
- [x] Done

**Owns:** `public/redesign/map-itineraries.js`, `public/redesign/map-itineraries.css`, `tasks/redesign/checks/map-itineraries.test.mjs`, `tasks/redesign/checks/map-itineraries.browser.mjs`

- [x] RED: the route uses `context.draw.line`, and the numbered pins use `markers`. The picker uses `control` or the `layers` slot as today.
- [x] RED: switching routes leaves one line and one set of pins, including after a style reload.
- [x] GREEN.

**Verify:** `node --test …map-itineraries.test.mjs`; `map-itineraries.browser.mjs` on both engines.

### M6. Integrate and re-run every map check · *after M3–M5*
- [x] Done

**Owns:** `tasks/redesign/checks/pr7-acceptance.browser.mjs`, `pr7-integration.browser.mjs`, `writers-map-layout.browser.mjs`, `writers-map-preview.browser.mjs`, `writers-map-imagery-live.mjs`, `a11y-audit.mjs` and `lib/` (selectors and engine switch only)

- [x] Every map browser check runs on **both engines**. Add an option to force the fallback, e.g. disable WebGL in the CDP launch.
- [x] Replace Leaflet class selectors with engine-neutral ones (`[data-map-canvas]`, marker data attributes).
- [x] `writers-map-imagery-live.mjs` checks the live OpenFreeMap style and Mapterhorn tiles load (optional live check, outside `make test`).
- [x] Accessibility audit passes. Chips, the panel, Save to plan and View on map all work as before.

**Verify:** all map checks pass on both engines at 375 and 1440 px; `make test build`.

### M7. Docs, CSP note and a real-phone check · *after M6*
- Dark map (owner decision 2026-10-12): evidence in [docs/tdd/map-3d-dark.tdd.md](../docs/tdd/map-3d-dark.tdd.md)
- Electroverse-like colours (owner request 2026-10-12): evidence in [docs/tdd/map-electroverse-colours.tdd.md](../docs/tdd/map-electroverse-colours.tdd.md)
- Labels, road weight, brighter pin colours and the preview flake (owner decisions 2026-10-12): evidence in [docs/tdd/map-labels-colours-flake.tdd.md](../docs/tdd/map-labels-colours-flake.tdd.md)
- [ ] Done — all but the owner real-phone check, which is pending (instructions in `docs/tdd/map-3d.tdd.md`, **First view and owner device**)

**Owns:** `docs/maintenance.md` (basemap section), `docs/security-review.md` (item 4 note only), `docs/tdd/map-3d.tdd.md` (new)

- [x] Update `docs/maintenance.md`:
  - sources, attributions and the pinned MapLibre version
  - fallback rules
  - how to self-host PMTiles if a host fails
- [x] Add to security item 4 the origins the CSP must allow: OpenFreeMap (style, tiles, glyphs, sprite), Mapterhorn, and `worker-src blob:` for MapLibre.
- [ ] **Owner (pending):** open the map on a real mid-range phone (Funchal, pitch and terrain on). Record first-render time and whether panning stays smooth. If it's too slow, a follow-up turns terrain off on small screens.
- [x] Evidence doc per AGENTS.md, plus first-view bytes and requests against the spike numbers.
- [x] Owner follow-up 2026-10-12: visible, framed pins, a pin popup with Details, and the place-tag list removed. Evidence: [`docs/tdd/map-pins-popup.tdd.md`](../docs/tdd/map-pins-popup.tdd.md).

**Verify:** document review; owner's phone check recorded.

**Done when:** with `WRITERS_MAP=1`, the map is MapLibre 3D, sharp with no grey tiles at every zoom. All layers work on it and on the Leaflet fallback, every map check passes on both, and the owner has checked it on a real phone.
