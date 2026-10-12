# Writers' Map Contracts

**Status:** original contract approved by the owner on 2026-10-08. The separate 1f contract
change was approved by the owner on 2026-10-10 (see **Rules for Agents** in
[the plan](../tasks/plan.md)). The PR 7 change (itineraries in the GeoJSON, layer names) was
approved by the owner on 2026-10-11. The map-3d change (task M0: map engines, the drawing API and
the fallback, in **Browser extension API** and **Template hooks**) was approved by the owner on
2026-10-11. The Go types and ports exist; the HTTP routes, plan-stop distance,
browser API and template hooks below are planned and not yet served.

This document is the integration boundary for the writers' map. Go models live in
`internal/writermap`; application ports live in `internal/application/writermap.go`. All
coordinates use decimal degrees (latitude, then longitude). QIDs are Wikidata item IDs.

## Source ports

- `WriterMapSource.Destination(ctx, area)` returns the destination's cached base places and writer
  matches, ordered for display, along with source status and whether the result is complete.
  Writer matches are `Mentions`, keyed by QID: one entry per post with writer host, blog name
  (the source's `name` in `writerdata/sources.json`), post reference, URL, title, language and the
  matched character positions. Mentions never carry post text. A place carries `area`, its Wikidata
  P131 (located in) label, plus optional `opening_hours` (from P3025/P8626/P8627) and
  `wikipedia_url` (English Wikipedia), when known.
- `Itineraries` are writer routes, one entry per post. Each of its `days` has `qids` (stops in
  first-mention order).
- Each `SourceStatuses` entry has a `status` of `active` (the last sync finished), `partial` (the
  last sync stopped part-way), `blocked` (403 or `robots.txt`; no more requests) or `removed`
  (taken down; rows purged). `checked_at` is left out when the source was never checked. The
  allowlist in `writerdata/sources.json` uses `active`, `blocked` and `removed`; `partial` only
  appears at runtime.
- `WriterMapSource.Place(ctx, area, qid)` returns panel place data and an optional summary. A
  missing summary is represented by `nil`; this is a normal state. An unknown QID returns an error
  wrapping `writermap.ErrPlaceNotFound`, never a `nil` place without an error.
- `BasePlaceSource.Places(ctx, area)` returns `BasePlaces` from one Wikidata query: `Pins` is
  filtered to visitable types and capped for the base layer; `Gazetteer` keeps the full nearby
  list and each place's P31 `Types` for writer matching. When they cannot be fetched and nothing
  is cached, it returns an empty result with an error the caller can log; the map still shows
  tiles and the day plan. The Wikidata adapter and its cache decorator both implement it.
- `BasePlaceRefresher.Refresh(ctx, area)` re-runs the base query and replaces the cached places in
  place. Only the cache decorator implements it; the monthly job calls it.
- `Summarizer.Summarize(ctx, place, passages)` returns English prose. Passage text is transient:
  callers must not log it or persist it. Only the generated summary and source post references may
  be retained.

## Destination in map URLs

`:dest` is the same `city-country` slug as `/trip/:slug` (for example `lisbon-pt`). Every map
request also carries the resolved coordinates as `?lat=…&lon=…`, as `/plan` does, so the server
builds the `Area` without a geocoding call. The slug gives the area's name and country.

- A slug that does not parse returns 404.
- Missing, non-finite or out-of-range coordinates (latitude outside ±90, longitude outside ±180)
  return 400, with the same rules as `/plan`.
- The template puts the full URL, coordinates included, in the map shell's data attributes, so
  the browser never builds it. `panel_url` in the GeoJSON carries the same query.

## GeoJSON endpoint

`GET /map/:dest.geojson?lat=…&lon=…` returns a GeoJSON `FeatureCollection`. Each feature is a Point whose
coordinates are `[longitude, latitude]`. `properties` contains:

| Field | Type | Meaning |
| --- | --- | --- |
| `qid` | string | Wikidata ID |
| `name` | string | Display name |
| `names` | string array | Matchable English and Portuguese labels/aliases |
| `kind` | string | Place kind, when known |
| `description` | string | English description, when known |
| `photo` | string | Commons image identifier, when present |
| `photo_credit` | string | Human-readable attribution, when present |
| `photo_url` | string | Image URL, when present |
| `sitelinks` | integer | Wikimedia sitelink count |
| `writer_count` | integer | Distinct writer hosts mentioning the place |
| `panel_url` | string | URL to fetch panel fragment |

The top-level `complete` boolean indicates whether all configured sources were checked. A result
can contain useful features while incomplete.

The top-level `itineraries` array carries the writers' routes (an empty array when there are none).
Each entry has `writer_host`, `post_url`, `post_title`, `language` and `days`; each day has `qids`
(stops in first-mention order, matching feature `qid`s). The post reference stays on the server.

## Place panel

`GET /map/:dest/:qid?lat=…&lon=…` returns an HTML fragment with the place name, *kind · area* when known,
optional description and attributed photo, writer count, optional AI summary, optional opening hours,
an optional English Wikipedia link, and writer links. Missing opening hours or Wikipedia URL leave
no empty row or link in the panel.
Each writer link carries `href`, `title`, `host`, `blog_name` and `language`; the blog name is
shown under the post title and the language as a label. The summary block is
omitted entirely when no summary exists and is labelled **AI summary** when present. No source post
text or excerpts appear in the fragment. The save control exposes `data-save-qid="Q…"`.

## Plan stops

Each stop in the trip plan's itinerary carries `distance_km`, the straight-line distance from the
previous stop in the same day, as calculated by `planner.DistanceKM`. The first stop has `0`.

## Browser extension API

The map core creates `window.TravelMap` and provides:

```js
TravelMap.registerLayer(name, { add(map, context), remove(map, context) })
TravelMap.setLayerEnabled(name, enabled)
TravelMap.selectPlace(qid)
TravelMap.updatePlan(days)
```

Layer names are unique. `add` and `remove` are called as a layer is enabled or disabled. The map
layers are `notable` (base places), `writers` (writer pins), `plan` (the day plan) and `itineraries`
(writers' routes); the legend chips toggle them with `TravelMap.setLayerEnabled(name, enabled)`.

### Map engines

The core draws the map with one of two engines:

- **`maplibre`** (default): MapLibre GL JS, a vector style with 3D buildings and terrain.
- **`leaflet`** (fallback): Leaflet with the raster basemap (EOX imagery, OSM on tile error),
  maximum zoom 18.

The core picks `leaflet` when WebGL is unavailable, or when the style, its tiles or the terrain
fail to load. It decides **once per mount**. If MapLibre fails after layers were added, the core
removes them, mounts the fallback and adds the enabled layers again. Layers never choose an engine.

### Layer context

`add` and `remove` receive `map` and `context`. **`map` is opaque**: it is the engine's own map
object, and layers must not call it, `maplibregl` or `L` directly. Layers draw only through
`context.draw`. `context` contains:

| Field | Meaning |
| --- | --- |
| `engine` | `"maplibre"` or `"leaflet"`, for logging and tests only; layers must not branch on it |
| `destination` | The destination slug (or the map URL when no slug is set) |
| `features` | The GeoJSON features from the map URL; empty until loaded |
| `shell` | The map shell element (for the named slots) |
| `selectPlace(qid)` | Selects a place, as `TravelMap.selectPlace` |
| `draw` | The drawing API below |

### Drawing API

`context.draw` behaves the same on both engines. Every call takes an `id` (a string the layer
owns, e.g. `"plan"`); calling it again with the same `id` **replaces** that group, so redraws
never duplicate.

| Call | Meaning |
| --- | --- |
| `markers(id, items)` | Shows HTML markers. Each item is `{ lngLat: [lon, lat], element, title, onClick, z }`. `element` is a DOM node the layer builds and styles; `title` is its accessible name; `z` orders overlapping markers (higher on top). |
| `line(id, coords, style)` | Shows a line through `[[lon, lat], …]`. `style` is `{ color, width, dash }` (`dash` is an array or omitted). |
| `control(id, element, position)` | Places a control element in a map corner (`"top-left"`, `"top-right"`, `"bottom-left"`, `"bottom-right"`). |
| `clear(id)` | Removes everything drawn under `id` (markers, line and control). Clearing an unknown `id` does nothing. |
| `fit(bounds, options)` | Fits `[[west, south], [east, north]]`, with `options.padding` in pixels. |
| `flyTo(lngLat, zoom)` | Moves the camera to `[lon, lat]`; `zoom` is optional. |

- **Markers and controls survive style reloads.** The core redraws lines after a style reload, so
  layers don't listen for engine events.
- **Remove contract:** a layer's `remove` must `clear` every `id` it drew. `add` after `remove` must
  give the same result as the first `add`.
- **Accessibility:** marker elements must be focusable buttons or links with a visible focus style;
  `title` sets the accessible name. The `list` slot stays the keyboard alternative to the map.

### Events

The core dispatches these `CustomEvent`s on `document`:

- `travelmap:ready` — map initialized; `detail` contains `{ map, destination, engine }`. `map` is
  opaque, as above.
- `travelmap:place-selected` — a place is selected; `detail` contains `{ qid, place }`.
- `travelmap:plan-updated` — the trip plan changed; `detail` contains `{ days }`.

`travelmap:ready` fires again when the core switches to the fallback.

## Template hooks

Templates provide one element for each map slot:

```html
<div data-map-slot="panel"></div>
<div data-map-slot="layers"></div>
<div data-map-slot="saved"></div>
<div data-map-slot="status"></div>
<div data-map-slot="list"></div>
```

The `list` slot is the keyboard-accessible alternative to interacting with map markers. Save
buttons use `data-save-qid` and may be handled entirely in the browser.

The map shell (`data-writers-map`) carries its configuration as data attributes. The server
fills them, and the browser never builds URLs:

| Attribute | Meaning |
| --- | --- |
| `data-map-url` | GeoJSON URL, coordinates included (required) |
| `data-map-lat`, `data-map-lon` | Destination centre (required) |
| `data-map-destination` | Destination slug |
| `data-map-style-url` | MapLibre style URL; default the OpenFreeMap Liberty style |
| `data-map-terrain-url` | Terrain (raster-dem) tile URL template; default Mapterhorn; empty turns terrain off |
| `data-map-tile-url`, `data-map-attribution` | Fallback raster tiles and their credit; default EOX imagery |
| `data-map-engine` | `leaflet` forces the fallback (tests and the design preview only) |

The design preview points the style, terrain and fallback tiles at local fixtures, so tests never
hit the network.
