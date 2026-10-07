# Writers' Map Contracts

**Status (2026-10-08):** the Go types and ports exist; the HTTP routes, plan-stop flag, browser API and template
hooks below are planned and not yet served.

This document is the integration boundary for the writers' map. Go models live in
`internal/writermap`; application ports live in `internal/application/writermap.go`. All
coordinates use decimal degrees (latitude, then longitude). QIDs are Wikidata item IDs.

## Source ports

- `WriterMapSource.Destination(ctx, area)` returns the destination's cached base places and writer
  matches, ordered for display, along with source status and whether the result is complete.
  Writer matches are `Mentions`, keyed by QID: one entry per post with writer host, blog name
  (the source's `name` in `writerdata/sources.json`), post reference, URL, title, language and the
  matched character positions. Mentions never carry post text. A place carries `area`, its Wikidata
  P131 (located in) label, when known.
- `Itineraries` are writer routes, one entry per post. Each of its `days` has `qids` (stops in
  first-mention order), `distance_km` (straight-line distance between consecutive stops, as in the
  planner) and `walk_minutes` (that distance at `writermap.WalkSpeedKMH`, 5 km/h, rounded). The
  writer's walk card shows one day.
- Each `SourceStatuses` entry has a `status` of `active` (the last sync finished), `partial` (the
  last sync stopped part-way), `blocked` (403 or `robots.txt`; no more requests) or `removed`
  (taken down; rows purged). `checked_at` is left out when the source was never checked. The
  allowlist in `writerdata/sources.json` uses `active`, `blocked` and `removed`; `partial` only
  appears at runtime.
- `WriterMapSource.Place(ctx, area, qid)` returns panel place data and an optional summary. A
  missing summary is represented by `nil`; this is a normal state. An unknown QID returns an error
  wrapping `writermap.ErrPlaceNotFound`, never a `nil` place without an error.
- `BasePlaceSource.Places(ctx, area)` returns the destination's Wikidata base places (the
  base layer and the gazetteer for matching). When they cannot be fetched and nothing is cached, it
  returns an empty list with an error the caller can log; the map still shows tiles and the day
  plan. The Wikidata adapter and its cache decorator both implement it.
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
| `writers` | string array | Those writer hosts, for the "All writers" filter; empty for base places |
| `has_summary` | boolean | Whether the place has an AI summary, for the Summaries layer |
| `panel_url` | string | URL to fetch panel fragment |

The top-level `complete` boolean indicates whether all configured sources were checked. A result
can contain useful features while incomplete.

## Place panel

`GET /map/:dest/:qid?lat=…&lon=…` returns an HTML fragment with the place name, *kind · area* when known,
optional description and attributed photo, writer count, optional AI summary, and writer links.
Each writer link carries `href`, `title`, `host`, `blog_name` and `language`; the blog name is
shown under the post title and the language as a label. The summary block is
omitted entirely when no summary exists and is labelled **AI summary** when present. No source post
text or excerpts appear in the fragment. The save control exposes `data-save-qid="Q…"`.

## Plan stops

Each stop in the trip plan's itinerary carries `writers_pick`, true when the stop's QID has a
writer count of 1 or more. The itinerary card shows it as a **Writers' pick** chip. Stops without
a QID, or with no writer mentions, are `false`.

## Browser extension API

The map core creates `window.TravelMap` and provides:

```js
TravelMap.registerLayer(name, { add(map, context), remove(map, context) })
```

Layer names are unique. `add` and `remove` are called as a layer is enabled or disabled. The core
dispatches these `CustomEvent`s on `document`:

- `travelmap:ready` — map initialized; `detail` contains `{ map, destination }`.
- `travelmap:place-selected` — a place is selected; `detail` contains `{ qid, place }`.
- `travelmap:plan-updated` — the trip plan changed; `detail` contains `{ days }`.

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
