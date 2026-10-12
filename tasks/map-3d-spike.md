# Spike: sharper, reliable map with a 3D view

**Status:** started 2026-10-11. Results go in [map-3d-results.md](map-3d-results.md).

**Problem (owner, 2026-10-11)**
- Some tiles stay **grey**, as if they never loaded.
- Zooming in **loses resolution**.
- Wanted: a 3D view like [Electroverse](https://electroverse.com/map/38.7221,-9.2516,12.3/), with a pitched camera, terrain and extruded buildings.

**Likely causes today** (`public/redesign/writers-map.js`)
- EOX Sentinel-2 has **10 m pixels**, so real detail ends around **z13–14**. Leaflet allows z19 with no `maxNativeZoom`, so it asks for tiles that don't exist.
- The `tileerror` handler swaps the **whole layer** to OSM on the first failed tile, which leaves grey gaps while it switches.

**Rules:** **free sources only**: no paid keys and no billing-linked tokens (see `docs/maintenance.md` basemap section). No production code changes in this spike.

## Questions to answer
1. **Quick fix:** do `maxNativeZoom` plus per-tile fallback (instead of swapping the whole layer) remove the grey tiles and the blur in today's Leaflet map? Measure it.
2. **3D stack:** can **MapLibre GL JS** (open source) give the Electroverse look from free data?
   - **Vector base with building heights:** OpenFreeMap (no key) or another free OpenMapTiles host
   - **Terrain:** free DEM tiles (e.g. AWS Terrain Tiles / Terrarium, Mapterhorn)
   - **Satellite at low zoom:** EOX, switching to vector detail at high zoom, or a hybrid
   - For each source: licence, attribution, request limits, uptime, coverage of Madeira and the Azores.
3. **Cost of migration:** today's layers (7a pins and panel, 7b day plan, 7c itineraries, 7f chips, the `TravelMap` API in `docs/writers-map-contract.md`) are built on Leaflet. What changes to move them to MapLibre? Is a Leaflet-compatible bridge possible?
4. **Performance:** bundle size, first render, phones (375 px), and whether a WebGL fallback is needed.

## Deliverables
- [x] `tasks/map-3d-results.md`, key point first:
  - a recommendation: quick fix only, quick fix now + MapLibre later, or MapLibre now
  - a table of sources (licence · attribution · limits · key needed)
  - migration effort per PR 7 file (scope only, no hours)
  - risks
- [x] A **throwaway prototype** outside the repo (in the scratchpad): a MapLibre page centred on Lisbon (38.7221, -9.2516, z12.3) and Funchal, with pitch, terrain, 3D buildings and a few fixture pins. Screenshots at 375 and 1440 px.
- [x] Before/after screenshots of the quick fix at z12, z15 and z17.
- [x] **Owner decisions needed**, listed at the end.
