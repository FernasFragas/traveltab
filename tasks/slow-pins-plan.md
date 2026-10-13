# Slow map pins: investigation plan

**Status:** plan only, nothing measured yet. Tasks: [slow-pins-todo.md](slow-pins-todo.md).

**Goal:** find out **where the time goes** between the map section appearing and the pins showing, and name the cause with numbers. Fixing it is a separate plan, written once the cause is known.

**Independent of the remaining Writers' Map work.** This plan changes no production code. It owns only `tasks/slow-pins-results.md` and temporary scratch files that are deleted at the end.

## How pins get on screen today

The steps run **one after another**. Each must finish before the next starts:

1. **Wait until visible.** `whenVisible` (`public/redesign/writers-map.js:166`) waits for the map canvas to scroll into view.
2. **Load MapLibre from the CDN.** `loadMaplibre` adds `maplibre-gl@5.24.0` JS and CSS from jsDelivr (`writers-map.js:9-10`).
3. **Load the map style.** `createMaplibre` waits for the map's `load` event: style, sprites, glyphs, first tiles, the terrain DEM (`addTerrain`) and 3D buildings (`addBuildings`).
   - If it fails or takes longer than **`loadTimeout` = 20 s** (`writers-map.js:47`), `startMap` falls back to **Leaflet** and starts again.
4. **Fetch the pins.** Only then does `ready()` call `loadPlaces` (`writers-map.js:208`), which fetches `/map/<dest>.geojson`.
5. **Server.** `writerMapGeoJSON` (`internal/adapters/httpserver/writermap.go`) calls `MapSource.Destination` (`internal/writermap/source.go:78`):
   - **cached** → read from SQLite.
   - **not cached** → `recompute`: Wikidata SPARQL query (`WikidataNearbyAPI.Places`, 30 s client timeout, waits for `Retry-After` on 429), then `WriterMatchRows` and `sourceStatuses`.
6. **Draw.** Each pin layer (`notable`, `writers` in `map-panel.js`, plus `itineraries`, `plan`) creates one **DOM marker per place** (`maplibregl.Marker`).

## Suspects, most likely first

1. **Waterfall: the GeoJSON fetch waits for the map engine.** The pins can't start loading until steps 1–3 finish, even though the fetch needs nothing from the map. Pin time = CDN + style + tiles + terrain + **then** the fetch.
2. **Cold server cache.** The first request for a destination runs a live Wikidata query. That can take seconds, or much longer when Wikidata rate-limits.
3. **MapLibre timeout → Leaflet fallback.** A slow or failing style/tile/terrain source costs up to **20 s** before Leaflet even starts.
4. **Heavy first render.** Terrain DEM and 3D buildings must load before `load` fires.
5. **Too many DOM markers.** Hundreds of DOM markers with labels can be slow to add and to move while panning.
6. **Lazy start.** If the map sits below the fold, nothing starts until the user scrolls. This is by design, but it feels like slow pins.

## How we'll measure

- **Browser timeline:** `performance.mark` at each step boundary (scratch edit, not committed), read from the console. Plus the DevTools Network waterfall.
- **Server timing:** `curl -w '%{time_total}'` on `/map/<dest>.geojson`, cold (empty SQLite) and warm.
- **Same destinations every time:** Lisbon, Kyoto, Madeira (Funchal). Desktop 1440 px and mobile 375 px with "Fast 4G" throttling.
- **Both environments:** `cmd/design-preview` (local fixtures, no network) separates our code from third-party latency; the real app (`cmd/web`) shows what users see.

## Done when

- `tasks/slow-pins-results.md` has a **timing table per step** (visible → MapLibre loaded → map `load` → fetch start → response → pins drawn) for each destination and environment.
- Each suspect is marked **confirmed / ruled out**, with the numbers.
- One recommended fix (or a short ranked list) to plan next.
- All scratch instrumentation is removed; `git status` shows only the results doc and these plan files.
