# Panel photo cache fix

Source: [panel-photo-cache-plan.md](../../tasks/panel-photo-cache-plan.md) and
[panel-photo-cache-todo.md](../../tasks/panel-photo-cache-todo.md), P1–P4. The owner chose B + C:
couple the destination cache key to the base-place version and remove v1 rows at startup.

**Journey:** As a traveller, I want each notable place's photo to show in the panel, so that I can
recognise the place before I go. A stale Commons page URL must not appear as an image or leave an
orphaned credit.

## RED → GREEN

| Behaviour | Command | RED excerpt | GREEN excerpt |
| --- | --- | --- | --- |
| A v1 destination row is a miss, and the destination key uses the base-place version | `go test ./internal/adapters/sqlite/ -run 'TestWriterDestination' -count=1` | `--- FAIL: TestWriterDestination_IgnoresLegacyCommonsPage (0.01s)`; `Should be false`; `Should be empty, but was [{lisbon pt 38.7 -9.1 0}]`. `--- FAIL: TestWriterDestination_KeyUsesBasePlaceVersion (0.00s)`; expected `writermap:v2:"lisbon":"pt":38.7:-9.1:0`, actual `writermap:v1:"lisbon":"pt":38.7:-9.1:0` | `ok weatherservice/internal/adapters/sqlite 0.382s` |
| Opening an old database deletes only the v1 photo cache rows, including on a second open | `go test ./internal/adapters/sqlite/ -run TestOpen_RemovesLegacyPhotoCacheRows -count=1` | `--- FAIL: TestOpen_RemovesLegacyPhotoCacheRows (0.01s)`; actual keys still included `wikidata:v1:funchal`, `wikidata:v1:lisbon`, `writermap:v1:funchal`, and `writermap:v1:lisbon` | `ok weatherservice/internal/adapters/sqlite 0.400s` |
| A Commons page URL gives neither panel image nor credit and omits GeoJSON photo fields | `go test ./internal/adapters/httpserver/ -run TestPlacePanel_InvalidPhotoURLHasNoPhotoOrCredit -count=1` | `--- FAIL: TestPlacePanel_InvalidPhotoURLHasNoPhotoOrCredit (0.01s)`; assertions found the page image, credit, and three GeoJSON photo fields | `ok weatherservice/internal/adapters/httpserver 0.393s` |
| A valid Wikimedia thumbnail host keeps the image in both responses | `go test ./internal/adapters/httpserver/ -run TestPlacePanel_ThumbWikimediaPhotoAppearsInPanelAndGeoJSON -count=1` | `--- FAIL: TestPlacePanel_ThumbWikimediaPhotoAppearsInPanelAndGeoJSON (0.01s)`; panel assertions found no image or credit, and GeoJSON photo values were nil | `ok weatherservice/internal/adapters/httpserver 0.544s` |

The thumbnail-host case came from the live palace response: Wikimedia returned a real image on
`thumb.wikimedia.org`. The adapter already accepts both `thumb.wikimedia.org` and
`upload.wikimedia.org`, so the panel guard uses the same two image hosts.

| What is guaranteed | Test name | Level | Result |
| --- | --- | --- | --- |
| Old destination result cannot hide recomputation; cache keys share the base-place version | `TestWriterDestination_IgnoresLegacyCommonsPage`, `TestWriterDestination_KeyUsesBasePlaceVersion` | SQLite integration | pass |
| Startup removes legacy rows while preserving other rows and their bytes | `TestOpen_RemovesLegacyPhotoCacheRows` | SQLite integration | pass |
| A Commons page URL does not reach the panel or map photo fields | `TestPlacePanel_InvalidPhotoURLHasNoPhotoOrCredit` | HTTP integration | pass |
| Both supported Wikimedia image hosts can supply a panel photo | `TestPlacePanel_PhotoSrcIsAnImageURLNotACommonsPage`, `TestPlacePanel_ThumbWikimediaPhotoAppearsInPanelAndGeoJSON` | HTTP integration | pass |

## Coverage and full gate

- `go test -cover ./internal/adapters/sqlite/`: `coverage: 81.6% of statements`.
- `go test -cover ./internal/adapters/httpserver/...`: `coverage: 92.7% of statements`.
- `PATH=/private/tmp/traveltab-tools:$PATH make test lint build`: all Go packages passed; `golangci-lint run ./...` reported `0 issues.`; `go build -o bin/traveltab ./cmd/web` passed. The installed `golangci-lint` 2.4.0 was built with Go 1.26 and could not lint this Go 1.27.2 module, so this run used 2.14.0 built with Go 1.27.2 in `/private/tmp`.

## Runtime check

Copied the local database to `/private/tmp/traveltab-panel-photo-runtime.db`, then started
`WRITERS_MAP=1 DB_PATH=/private/tmp/traveltab-panel-photo-runtime.db go run ./cmd/web`. No rows
were manually changed. Startup removed the old `wikidata:v1` and `writermap:v1` rows. The first
`curl -m 35` Lisbon GeoJSON request ended with `Operation timed out after 35005 milliseconds with
0 bytes received`; the server later stored `wikidata:v2` and `writermap:v2` rows. The next
GeoJSON request returned `HTTP 200, total 0.003216s`, and the palace panel returned HTTP 200.

The palace's `photo_url` was a `https://thumb.wikimedia.org/wikipedia/commons/thumb/.../960px-...jpg`
URL with the credit `Diego Delso · CC BY-SA 4.0`. `curl -I -L` returned `HTTP/2 200` and
`content-type: image/jpeg`. The live browser check against `/trip/lisbon-pt` opened the palace
panel at 375 and 1440 px: `naturalWidth: 960`, the expected credit, no horizontal overflow, and
`errors: []` at both widths. The fixture check
`CDP_PORT=9240 node tasks/redesign/checks/map-panel.browser.mjs` passed MapLibre and Leaflet at
both widths with clean page consoles.

**Known gap:** a cold Lisbon recompute did not finish within 35 seconds, even though it eventually
populated the cache. Before release, warm and verify required destinations as described in
[maintenance.md](../maintenance.md). The remaining cold-path resilience and latency work belongs
with [PR 8a–8b](../../tasks/todo.md#pr-8--verify-and-launch--needs-pr-6-pr-7) and
[upstream deadlines](../security-review.md#10-upstream-deadlines-and-cancellation-gaps-medium).
