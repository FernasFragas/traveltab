# Writers' map in 3D (MapLibre)

**Status:** M0–M6 done and re-verified 2026-10-11, merged in #36 and #37 (evidence: `docs/tdd/map-3d.tdd.md`). M7 done except the **owner real-phone check**, which is pending. Tasks: [map-3d-todo.md](map-3d-todo.md). Spike: [map-3d-results.md](map-3d-results.md).

**Goal:** replace the Leaflet satellite map with a **3D vector map**: tilted camera, terrain and extruded buildings, like [Electroverse](https://electroverse.com/map/38.7221,-9.2516,12.3/). It must be **sharp at every zoom with no grey tiles**, built from **free sources only**.

**When:** after PR 7 merges, and **before PR 8** (8a–8c verify the final UI). It edits PR 7's map files, which no open task owns.

## Owner decisions (2026-10-11)
- **No quick fix** to today's Leaflet map; go straight to MapLibre.
- **Base: vector**: OpenFreeMap (Liberty style, built-in 3D buildings). No satellite in the 3D view.
- **Terrain: Mapterhorn**, 512 px tiles, no hillshade (Madeira 10 m, Azores 30 m; first view about 1.3 MB).
- **Free hosts with no uptime guarantee are accepted**, with a **fallback to today's Leaflet + EOX map** when:
  - WebGL is missing, or
  - the style, tiles or terrain fail to load.
  
  The fallback caps zoom at **18**, because EOX has no z19 and that's where the grey tiles came from.
- **Weight accepted:** MapLibre is about 275 KB compressed, **lazy-loaded** only when the map card is on screen.

## Why it's not a swap
Every PR 7 layer receives the **Leaflet map** and calls `L.*` directly (spike: **Migration scope**). The `TravelMap` API (`mount`, `registerLayer`, `setLayerEnabled`, `selectPlace`, `updatePlan`, the three events) can stay. The layers can't.

**Design:** the core exposes a **small drawing API** that works the same on both engines, so each layer is written **once** and runs on MapLibre or the Leaflet fallback:

| Drawing API (in `context.draw`) | MapLibre | Leaflet fallback |
|---|---|---|
| `markers(id, [{lngLat, element, title, onClick}])` / `clear(id)` | `maplibregl.Marker` with an HTML element | `L.marker` + `divIcon` |
| `line(id, coords, style)` | GeoJSON source + line layer | `L.polyline` |
| `control(id, element, position)` | custom `IControl` | `L.control` |
| `fit(bounds)`, `flyTo(lngLat, zoom)` | native | native |

- HTML markers keep today's look, focus and accessibility, and they **survive style reloads**. Lines are re-added by the core after a style change.
- Layers never touch `maplibregl` or `L`. The contract's `add(map, context)` keeps its shape, but layers use `context.draw`.

## Pinned versions and sources
- **MapLibre GL JS 5.24.0**, the UMD build (6.x is ESM-only; a later move needs a module loader).
- **Style:** `https://tiles.openfreemap.org/styles/liberty`, attribution "OpenFreeMap © OpenMapTiles Data from OpenStreetMap" (MapLibre shows it).
- **Terrain:** Mapterhorn raster-dem, attribution "© Mapterhorn" + sources (Madeira IGP CC BY 4.0, Copernicus GLO-30).
- **Camera:** pitch about 55°, bearing about −20°, zoom 12.3 on the destination. The user can rotate and tilt.

## Risks (from the spike)
- **Real phones are untested.** Terrain at an angle is heavy, so a real-device check is a task (M7).
- **Building heights come from OSM**, so Funchal and small towns often look flat. Accepted.
- **Outages:** if a free host goes down, users get the fallback. Self-hosting PMTiles stays the escape hatch.
- **CSP (security review item 4)** must allow OpenFreeMap and Mapterhorn (`connect-src`, `img-src`, `worker-src blob:`). M7 notes this in `docs/security-review.md`.
- **Contract change:** the contract is frozen, so M0 needs **owner review** first, as 1f did.
