# Writers' Map Contracts

This document is the integration boundary for the writers' map. Go models live in
`internal/writermap`; application ports live in `internal/application/writermap.go`. All
coordinates use decimal degrees (latitude, then longitude). QIDs are Wikidata item IDs.

## Source ports

- `WriterMapSource.Destination(ctx, area)` returns the destination's cached base places and writer
  matches, ordered for display, along with source status and whether the result is complete.
- `WriterMapSource.Place(ctx, area, qid)` returns panel place data and an optional summary. A
  missing summary is represented by `nil`; this is a normal state.
- `Summarizer.Summarize(ctx, place, passages)` returns English prose. Passage text is transient:
  callers must not log it or persist it. Only the generated summary and source post references may
  be retained.

## GeoJSON endpoint

`GET /map/:dest.geojson` returns a GeoJSON `FeatureCollection`. Each feature is a Point whose
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

## Place panel

`GET /map/:dest/:qid` returns an HTML fragment with the place name, optional description and
attributed photo, writer count, optional AI summary, and writer links. Each writer link carries
`href`, `title`, `host`, and `language`; the language is shown as a label. The summary block is
omitted entirely when no summary exists and is labelled **AI summary** when present. No source post
text or excerpts appear in the fragment. The save control exposes `data-save-qid="Q…"`.

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
