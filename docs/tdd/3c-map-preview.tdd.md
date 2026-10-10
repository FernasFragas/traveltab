# 3c — Writers' map design preview

Source: [tasks/todo.md](../../tasks/todo.md), PR 3 subtask 3c.

## Journeys

- As a traveler viewing Madeira, I want writer places and a local map fixture so I can inspect the map without live data sources.
- As a traveler viewing Porto, I want base places even when no writer has mentioned them.
- As a traveler viewing Coimbra while sources are unavailable, I want the map and an explicit status.
- As a keyboard user, I want to reach the place list after destination swaps at mobile and desktop widths.

## Red and green

| Behavior | Command | RED excerpt | GREEN excerpt |
| --- | --- | --- | --- |
| Preview sources and local tiles | `go test ./cmd/design-preview -run TestWriterMapPreview -count=1` | `undefined: newWriterMapPreviewSource`; `undefined: fixtureTileHandler` | `ok weatherservice/cmd/design-preview 0.763s` |
| Browser swaps and tile placeholders | `CDP_PORT=9339 node tasks/redesign/checks/writers-map-preview.browser.mjs` | `data-map-tile-url` was `.../fixture-tiles/%7bz%7d/%7bx%7d/%7by%7d.png`, failing the placeholder assertion | `PASS 375px: writer data, base-only and unavailable swaps, one map, keyboard list, local tiles`; `PASS 1440px: writer data, base-only and unavailable swaps, one map, keyboard list, local tiles` |
| Unavailable sources without a console error | `go test ./cmd/design-preview -run TestWriterMapPreview -count=1` and the browser command above | `Destination error = fake writer map source unavailable`; `/map/coimbra-pt.geojson...: status=503 body=Service Unavailable`; browser console logged `Response Status Error Code 503` | `ok weatherservice/cmd/design-preview 0.508s`; both browser viewport checks passed with no page console errors |

## Guarantees

| What is guaranteed | Test name | Level | Result |
| --- | --- | --- | --- |
| Madeira has writer data, Porto has a base place, Coimbra has an empty incomplete result with unavailable source statuses | `TestWriterMapPreview_Scenarios` | Unit | PASS |
| Preview routes render all three destinations and GeoJSON states | `TestWriterMapPreview_RoutesAndLocalTiles` | Integration | PASS |
| Valid XYZ tile paths return PNG and invalid paths return 404 | `TestWriterMapPreview_RoutesAndLocalTiles` | Unit | PASS |
| Base-only result has no mentions; panels and selected coordinates resolve correctly | `TestWriterMapPreview_BaseOnlyJSON`, `TestWriterMapPreview_PanelsAndSelectedDestination` | Unit | PASS |
| At 375 and 1440 px, two swaps retain one map, load a local tile, keep layout inside the viewport, and allow keyboard focus on the list | `writers-map-preview.browser.mjs` | Browser | PASS |

`go test -cover ./cmd/design-preview/...` returned `coverage: 80.4% of statements`.
`go test -cover ./internal/adapters/httpserver/...` returned `coverage: 91.8% of statements`.
`PATH=/private/tmp:$PATH make test lint build` exited 0: all Go packages passed, lint reported `0 issues.`, and the web app built. The `/private/tmp` linter is used because the installed 2.4.0 binary cannot read Go 1.27 export data.

The preview uses the production Leaflet CDN, so the browser run requires that script to load. The fixture verifies local GeoJSON and tile loading, not live map provider availability.
