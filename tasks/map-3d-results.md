# Spike results: sharper map and a 3D view

**Date:** 2026-10-11. **Spec:** [map-3d-spike.md](map-3d-spike.md). No production code was changed.

## Recommendation: quick fix now, MapLibre later

- **Do the quick fix now.** It is a few lines in `writers-map.js`, it removes the **grey map at z19**, and it removes the **blurry satellite** by switching to sharp OSM tiles above z14.
- **Plan MapLibre as its own PR.** The 3D look works from free data, with no keys. But the PR 7 layers call Leaflet directly, so it is a **rewrite of the layer files**, not a swap.
- **Not "MapLibre now":** it costs a rewrite plus new risks (WebGL on old phones, OpenFreeMap has no SLA, 1.1 MB of JS), and the quick fix is needed anyway as the no-WebGL fallback.

## Diagnosis (numbers, Leaflet + live EOX, Brave 155 headless, 1440x900)

**The suspected cause was half right.** The grey comes from z19, not from failed requests.

| Finding | Evidence |
| --- | --- |
| EOX serves **z0 to z18**, then **404 at z19** | Lisbon and Funchal tiles, z8-z18 all 200; z19 404 with header `x-mapcache-error: invalid TILEMATRIX 19` |
| The 404 body is a **valid transparent PNG** | 1225 bytes, 256x256 RGBA, so the browser decodes it as an image |
| So `tileerror` **never fires** and the OSM fallback never runs | z19: 35 of 35 tiles HTTP 404, 0 `tileerror`, 0 OSM requests, screen fully **grey** |
| z12, 14, 15, 17, 18 over both cities | **100% HTTP 200**, 0 broken tiles, 0 grey (24 to 35 tiles each) |
| Imagery is **blurry above z14** | 10 m Sentinel pixels; tile size falls from 22 KB (z13) to 2.4 KB (z18) as it is upscaled. Funchal z17 is a smear |
| No rate limiting seen | 400 tiles in 3.4 s, all 200 |

- **Unproven:** an occasional grey tile from a transient EOX error or timeout. I did not reproduce one. The per-layer swap in `tileerror` would produce gaps in that case.
- Screenshots: `/private/tmp/claude-501/-Users-fernando-personal-projects-traveltab/3db1f481-7924-450e-abf9-93fbf9871511/scratchpad/map3d/shots/leaflet-before-{lisbon,funchal}-z{12,14,15,17,18,19}.png`.

## Quick fix measured

Scratch copy `/private/tmp/claude-501/-Users-fernando-personal-projects-traveltab/3db1f481-7924-450e-abf9-93fbf9871511/scratchpad/map3d/leaflet.html`. Both variants: EOX `maxNativeZoom: 14`, and a **per-tile** swap to OSM (`tile.src` replaced on error) instead of replacing the whole layer.

| Variant | z12 | z14 | z15 | z17 | z19 |
| --- | --- | --- | --- | --- | --- |
| Before | sharp | OK | blurry | blurry | **grey** |
| A: `maxNativeZoom 14` only | OK | OK | upscaled (still soft) | soft, 2-4 tiles requested | **imagery, not grey** (still soft) |
| B: A, plus **OSM raster above z14** | EOX | EOX | OSM, sharp, labelled | OSM, sharp | OSM, sharp |

- All tiles loaded in every case (0 broken). Variant B loses satellite above z14 but gains street detail and names.
- Request count drops at high zoom under A (z17: 4 tiles instead of 28).
- **Also fix:** `maxZoom` could stay 19 only with OSM; with EOX alone use `maxZoom: 18`. Detect the z19 404 by tile size or by capping the zoom, since `tileerror` will not fire.
- **OSM use must follow the [tile policy](https://operations.osmfoundation.org/policies/tiles/):** valid Referer, attribution, no prefetch. Fine for the on-demand high-zoom case; no SLA.
- Screenshots: `/private/tmp/claude-501/-Users-fernando-personal-projects-traveltab/3db1f481-7924-450e-abf9-93fbf9871511/scratchpad/map3d/shots/leaflet-after14-*.png` (A), `leaflet-after-osm14-*.png` (B).

## Sources

All checked on 2026-10-11.

| Source | Licence | Attribution | Limits | Key | Madeira / Azores |
| --- | --- | --- | --- | --- | --- |
| **MapLibre GL JS** 5.24.0 (jsDelivr) | BSD-3 | auto in the control | n/a | no | n/a |
| **OpenFreeMap** vector tiles + Liberty style ([repo](https://github.com/hyperknot/openfreemap), [ToS](https://openfreemap.org/tos/)) | Project MIT; data OSM (ODbL) via OpenMapTiles | **Required.** "OpenFreeMap © OpenMapTiles Data from OpenStreetMap" (MapLibre adds it) | Repo: "no limits on the number of map views or requests". **No SLA**; donation funded; ToS can discontinue any time; ToS bans automated collection | no | Planet-wide. Seen working in Lisbon, Funchal, Ponta Delgada. Building heights come from OSM, so uneven outside big cities |
| **AWS Terrain Tiles** terrarium ([registry](https://registry.opendata.aws/terrain-tiles/), [attribution](https://github.com/tilezen/joerd/blob/master/docs/attribution.md)) | Open data; per-source terms | Per-source credit list (EU-DEM/Copernicus for Europe, SRTM/USGS, ETOPO1). **Unclear** which source covers Madeira and the Azores; the page does not say | None stated | no (public S3) | Global, worked in both. Page does not state the resolution there |
| **Mapterhorn** terrain ([data access](https://mapterhorn.com/data-access/), [attribution.json](https://download.mapterhorn.com/attribution.json)) | Code BSD-3; data per source: **Madeira CC BY 4.0 (IGP, 10 m)**, mainland Portugal CC BY 4.0 (DGT, 0.5 m), global Copernicus GLO-30 (30 m) | "© Mapterhorn" plus the source list | **None stated**; run on Cloudflare donated infrastructure; no SLA | no | Madeira 10 m. **No Azores-specific source in the catalogue**; falls back to GLO-30 |
| **EOX s2cloudless 2016** ([licence](https://cloudless.eox.at/license-non-commercial)) | CC BY 4.0 | EOX credit already in `writers-map.js` | 4096 px request max; no numeric quota; no SLA; **max zoom 18** | no | Global; Funchal and Lisbon verified |
| **OSM standard tiles** ([policy](https://operations.osmfoundation.org/policies/tiles/)) | ODbL | OSM credit visible on the map | No heavy use; no prefetch or offline; Referer and cache headers required; blocks without notice; no SLA | no | Global |

- **Unclear:** OpenFreeMap's ToS names no licence for the service itself and no attribution rule; the repo README does. Contact info@openfreemap.org if the owner wants it in writing.
- **Excluded by the no-billing-key rule:** Mapbox, MapTiler, Stadia, Esri (not re-evaluated; same reasons as task 3e).
- **Self-hosting escape hatch:** OpenFreeMap publishes weekly planet files; Mapterhorn publishes PMTiles. Not needed now.

## Prototype findings

Scratch page `/private/tmp/claude-501/-Users-fernando-personal-projects-traveltab/3db1f481-7924-450e-abf9-93fbf9871511/scratchpad/map3d/ml.html` (MapLibre 5.24.0 UMD, jsDelivr). Camera pitch 58, bearing -20, Lisbon `38.7221,-9.2516` z12.3 and Funchal.

- **Works with no keys:** OpenFreeMap Liberty style, with its **fill-extrusion `building-3d` layer already in the style** (zoom 14+, `render_height`). Terrain from a DEM source with `setTerrain`. Sky and fog. Numbered pins (circle + symbol layers) and a dashed line, redrawn from one GeoJSON source.
- **Sharp at high zoom:** z16 and z18 over Lisbon are vector, with no grey and no blur. Funchal and Azores also render.
- **Hybrid (EOX under the vector layers, fading out at z14.5 to 16):** works, but Liberty's roads and labels over satellite are **too busy**; it would need a custom style. Satellite above z14 is still blurry, so it only helps the low-zoom look.
- **Console:** 0 errors. Warnings only: "Expected value to be of type number, but found null" (the Liberty style, harmless), and a WebGL "GPU stall" notice from `preserveDrawingBuffer` (used only for screenshots; leave it off in production).
- **Terrain request cost is the gotcha.** Terrarium 256 px tiles with a hillshade layer on a second source: **88 DEM tiles, 7.7 MB**. With `tileSize: 512` and no hillshade: **15 tiles, 1.4 MB**. Mapterhorn (512 px webp): **13 to 20 tiles, 0.85 MB**.
- **Not tried:** a pin popup or tooltip, touch gestures, and a real GPU (see Risks).

Screenshots, all in `/private/tmp/claude-501/-Users-fernando-personal-projects-traveltab/3db1f481-7924-450e-abf9-93fbf9871511/scratchpad/map3d/shots/`:

| View | File |
| --- | --- |
| Lisbon z12.3, 1440 | `ml-lisbon-z12-1440.png` |
| Lisbon z16.2 (Rossio, 3D buildings), 1440 | `ml-lisbon-z16-1440.png` |
| Lisbon z18.3, 1440 | `ml-lisbon-z18-1440.png` |
| Funchal z12.3, 1440 | `ml-funchal-z12-1440.png` |
| Funchal z16.2, 1440 | `ml-funchal-z16-1440.png` |
| Hybrid Funchal z14.2 / Lisbon z13 | `ml-funchal-hybrid-z14-1440.png`, `ml-lisbon-hybrid-z13-1440.png` |
| 375 px | `ml-lisbon-z12-375.png`, `ml-funchal-z16-375.png` |
| Mapterhorn terrain, Funchal / Azores | `ml-mh-p=funchal&.png`, `ml-azores.png` |

## Performance (first view, cache disabled)

Measured in headless Brave with **software WebGL (SwiftShader)**, so GPU time is pessimistic. Network and size numbers are real.

| Metric | Leaflet today (Lisbon z12) | MapLibre, AWS DEM 512 | MapLibre, Mapterhorn |
| --- | --- | --- | --- |
| Library JS, compressed | 42 KB (+3.5 KB CSS) | **275 KB** (+10 KB CSS); 1.06 MB raw | same |
| Requests on first view | 38 | 30 | 28 |
| Bytes on first view | 0.59 MB total | 1.9 MB total | 1.33 MB total |
| `load` event | n/a (tiles done in 0.6 s) | 0.9 s | 0.6 to 0.9 s |
| Idle (all tiles drawn) | 0.6 s | 4.3 s (software GL) | 3.7 s |
| 375 px | n/a | 24 requests, 1.35 MB, idle 2.4 s | n/a |

- **About 6.5x the JS and 2 to 3x the bytes** of today. Lazy-load it only when the map card is on screen, as Leaflet already is.
- **WebGL fallback is needed.** MapLibre has no canvas fallback. Keep the Leaflet path (with the quick fix) when `webgl2`/`webgl` is unavailable. This is also why the quick fix is not throwaway work.
- Mid and low-end phones were not measured. Terrain at pitch 58 is the heavy part; test on a real device before committing.
- MapLibre 6.x (latest 6.14.0) is **ESM-only** in `dist/` (no UMD file). 5.24.0 is the UMD line and works with the current plain-script loading. A 6.x move needs a module loader.

## Migration scope (MapLibre)

**Thin adapter: yes for the `TravelMap` API, no for the layers.** `mount`, `registerLayer`, `setLayerEnabled`, `selectPlace`, `updatePlan`, the three events, and the data-attribute shell can stay identical. But every layer's `add(map, context)` receives the Leaflet map and calls `L.*` directly, and `travelmap:ready` hands out `map`. A Leaflet-compatible shim over MapLibre would be larger than rewriting the four layer files.

| File | What changes under MapLibre |
| --- | --- |
| `writers-map.js` | **Core rewrite.** Replace `loadLeaflet`, `L.map`, scale control, `L.tileLayer` with MapLibre load and `new Map`. Basemap and fallback become a style (raster EOX layer or vector). Keep htmx swap and canvas re-parenting (call `map.resize()` instead of `invalidateSize`), the info popover, list, status, `loadPlaces`. Keep a Leaflet code path for no-WebGL |
| `map-panel.js` | **Rewrite `pinLayer`.** `L.marker`, `divIcon`, `bindTooltip`, `layerGroup`, `zIndexOffset` become `maplibregl.Marker` with HTML elements (closest to today) or a symbol layer. Permanent labels need a symbol layer or custom tooltip. Panel loading and monograms unchanged |
| `map-dayplan.js` | **Rewrite `draw`.** Numbered markers become HTML markers or a symbol layer. `L.control` day toggle becomes `map.addControl` with a custom control; `L.DomEvent` goes away |
| `map-itineraries.js` | **Rewrite.** `L.polyline` becomes a GeoJSON line layer; numbered `divIcon` markers as above |
| `map-layers.js` | **Unchanged.** It only calls `TravelMap.setLayerEnabled`; layer add/remove must stay idempotent |
| `map-saved.js` | **Unchanged**, per a grep: no Leaflet calls |
| CSS | `map-panel.css` (`.map-pin.leaflet-marker-icon`) and `writers-map-layout.css` (two `.leaflet-*` rules for control position) need `maplibregl-*` equivalents. Add the MapLibre stylesheet. Check the attribution box placement at 375 px |
| Browser checks | `writers-map-core`, `map-panel`, `map-dayplan`, `writers-map-layout`, `writers-map-preview`, `pr7-acceptance`, `writers-map-imagery-live`, `a11y-audit` reference Leaflet classes or `window.L`; the `*.test.mjs` files stub `L`. Selectors and stubs need updating. The preview fixture tiles need a vector or raster style stub, since MapLibre also fetches a style, glyphs and a sprite |
| `docs/writers-map-contract.md` | `TravelMap` signatures can stay. **Change:** `travelmap:ready` `detail.map` and the `add(map, context)` argument become a MapLibre map. Add what layers may rely on (`context.features`, `selectPlace`) |
| `registerLayer({add, remove})` | The contract stays. Layers must also **re-add after a style change** (MapLibre drops custom sources and layers on `setStyle`), so a re-add hook is needed |

- **Cross-cutting:** MapLibre fetches its style, sprite and glyphs from OpenFreeMap at runtime (a new third-party dependency in the CSP `connect-src`/`img-src`/`worker-src`; check `internal` server headers). Pin accessibility (keyboard focus, titles) needs the same care as today.
- **Order that limits risk:** adapter and core first, then one layer at a time behind the existing checks, with Leaflet kept as the fallback.

## Risks

- [ ] **No SLA on the free hosts.** OpenFreeMap, Mapterhorn, EOX and OSM can all go away or throttle. Mitigation: keep the Leaflet and EOX path as the fallback; self-hosting files exist for OpenFreeMap and Mapterhorn.
- [ ] **WebGL missing or weak** on old phones. Needs a fallback and a real-device check.
- [ ] **Weight:** 275 KB JS plus 2 to 3x tile bytes on first view; terrain is the heaviest part.
- [ ] **Building heights are OSM-only.** Funchal and small towns will often show flat or missing buildings; I only inspected Lisbon at z16/18 closely.
- [ ] **Azores terrain** has no dedicated source in Mapterhorn; it falls back to 30 m. Looks coarse at close zoom.
- [ ] **Satellite stays 2016, 10 m.** MapLibre does not fix that; the sharp options are vector or OSM.
- [ ] **Testing cost:** about eight browser checks and the fixture server change.
- [ ] **ToS ambiguity** for OpenFreeMap (no explicit licence or attribution in the ToS page).
- [ ] **Unproven:** the owner's occasional grey tiles other than z19. If they persist after the quick fix, capture a HAR on their network.

## Owner decisions needed

1. **Approve the quick fix PR now?** If yes, which above z14: **A** satellite upscaled (soft, keeps the look) or **B** OSM raster (sharp, names, loses satellite)? Recommendation: **B**, with `maxZoom` aligned to the real limit.
2. **Is a 3D view worth a rewrite PR** (four layer files, about eight browser checks)? Recommendation: yes, after the quick fix, as its own PR.
3. **Accept free hosts with no SLA** (OpenFreeMap, Mapterhorn) with Leaflet as the fallback, or self-host the data later?
4. **Satellite or vector for the 3D base?** Vector is sharp and cheap; satellite is faster to understand but blurry above z14.
5. **Accept the 6.5x JS size** and a **no-WebGL fallback to Leaflet**, or drop the fallback and require WebGL?
6. **Are the grey tiles you see at z19, or elsewhere?** If elsewhere, tell me where and on which network.
