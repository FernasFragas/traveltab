# MapLibre core and offline preview fixtures (M1–M2)

Source: [map-3d-todo.md](../../tasks/map-3d-todo.md), M1 and M2, and the approved drawing API in [writers-map-contract.md](../writers-map-contract.md).

## Journeys

- As a traveller, I want a sharp 3D destination map, so I can see terrain and streets at any zoom.
- As a traveller without WebGL or with a failed map source, I still want a working map and place list.
- As a developer, I want a local style, vector tiles and terrain, so browser checks do not depend on map hosts.

## RED and GREEN

| Behaviour | Command | RED excerpt | GREEN excerpt |
| --- | --- | --- | --- |
| Core engine, drawing API, fallback and reuse | `node --test tasks/redesign/checks/writers-map-core.test.mjs` | `ℹ tests 28`, `ℹ pass 0`, `ℹ fail 28`; `ReferenceError: leafletPromise is not defined` from the incomplete core | `ℹ tests 29`, `ℹ pass 29`, `ℹ fail 0` |
| Style error after MapLibre is ready | `node --test tasks/redesign/checks/writers-map-core.test.mjs --test-name-pattern='a style error after the map is ready switches to Leaflet'` | `✖ a style error after the map is ready switches to Leaflet`; actual calls were only `['add', 'maplibre']` | `✔ a style error after the map is ready switches to Leaflet` in the final 29-test run |
| A tile host fails after earlier success | `node --test --test-name-pattern='a source outage after earlier successful tiles' tasks/redesign/checks/writers-map-core.test.mjs` | `✖ a source outage after earlier successful tiles falls back to Leaflet`; `0 !== 1` fallback maps | `✔ a source outage after earlier successful tiles falls back to Leaflet`; `ℹ pass 1`, `ℹ fail 0` |
| Local style, glyphs, sprite, vector tile and terrain | `go test ./cmd/design-preview/ -run 'TestPreview_ServesALocalMapStyle\|TestPreview_ServesGlyphsAndSprite\|TestPreview_ServesAVectorTileOverFunchal\|TestPreview_ServesAFlatTerrainTile' -count=1` | A post-handoff negative check temporarily disabled the fixture route: all four tests failed on `text/html` responses instead of the expected asset content types. The route was restored immediately. | `ok  weatherservice/cmd/design-preview  0.390s` |
| Both engines in the preview at mobile and desktop widths | `CDP_PORT=9227 node tasks/redesign/checks/writers-map-core.browser.mjs` | A stricter first check expected vector worker requests in page resource timing and timed out. A screenshot showed the scene rendered; the check now queries loaded vector features directly. | `PASS maplibre 375px`, `PASS leaflet 375px`, `PASS maplibre 1440px`, `PASS leaflet 1440px` (canvas, local fixtures, layout, console) |

## Guarantees

| What is guaranteed | Test name | Level | Result |
| --- | --- | --- | --- |
| Pinned, lazy MapLibre load; camera and terrain | `loads the pinned MapLibre assets`, `draws a 3D MapLibre map` | unit | PASS |
| WebGL absence, style errors, tile errors and script errors use Leaflet at max zoom 18 | `falls back to Leaflet with maxZoom 18 when WebGL is missing`, style, tile and script fallback tests | unit | PASS |
| Drawing calls replace their IDs on either engine; lines return after style reload | `context.draw has the same calls on both engines`, drawing and style reload tests | unit | PASS |
| HTMX swaps preserve the map and discard stale responses | `reuses the map across swaps and drops an old response`, `a swap clears what the layers drew` | unit | PASS |
| Preview serves a local style, vector features, flat DEM, glyph and sprite stubs | `TestPreview_ServesALocalMapStyle`, `TestPreview_ServesAVectorTileOverFunchal`, `TestPreview_ServesAFlatTerrainTile`, `TestPreview_ServesGlyphsAndSprite` | integration | PASS |
| MapLibre loads local transportation features and terrain; Leaflet works with WebGL disabled; no page exceptions or overflow at 375 and 1440 px | `writers-map-core.browser.mjs` | browser | PASS |

`go test -cover ./cmd/design-preview/` reported `coverage: 86.8% of statements`. `PATH=/private/tmp:$PATH make test lint build` passed, including `golangci-lint run ./...` with `0 issues.` The installed linter was built with Go 1.26; this run used golangci-lint v2.14.0 built with the project's Go 1.27.2 toolchain in `/private/tmp`.

M3–M5 still need to move the existing pin, day-plan and itinerary layers from direct Leaflet calls to `context.draw`. The M1 browser check disables those layers to test the core in isolation. The full writers-map preview is therefore not the final integrated map yet. The browser check uses local map data fixtures but loads the pinned MapLibre and Leaflet libraries from their CDNs; the Go fixture tests use no external network.
