# 3b — Map shell and fixture endpoints

Source: [tasks/todo.md](../../tasks/todo.md), PR 3 subtask 3b, and the approved [writers' map contract](../writers-map-contract.md).

## Journeys

- As a traveler with `WRITERS_MAP=1`, I want a map shell with a keyboard list and place details, so I can explore the destination.
- As a traveler with the flag off, I want the existing Waze card and its larger map link.
- As a traveler following a map URL, I want invalid slugs and coordinates rejected and place text safely escaped.

## Red and green

| Behavior | Command | RED excerpt | GREEN excerpt |
| --- | --- | --- | --- |
| Flagged shell and endpoints | `go test ./internal/adapters/httpserver/ -run TestWriterMap -count=1` | `server.SetWriterMapSource undefined` | `ok  weatherservice/internal/adapters/httpserver  0.505s` |
| Compiled fixture panel | `go test ./internal/adapters/httpserver/ -run TestWriterMap_FixturePanel -count=1` | `undefined: writermaptest.NewMadeiraSource` | `ok  weatherservice/internal/adapters/httpserver  0.394s` (run with `-run TestWriterMap`) |
| GeoJSON media type | `go test ./internal/adapters/httpserver/ -run TestWriterMap_FlaggedPageAndEndpoints -count=1` | `expected: "application/geo+json; charset=utf-8"` / `actual: "application/json"` | `ok  weatherservice/internal/adapters/httpserver  0.380s` (run with `-run TestWriterMap`) |

## Guarantees

| What is guaranteed | Test name | Level | Result |
| --- | --- | --- | --- |
| Flagged page has all slots, map URL and larger map link; flag off keeps Waze | `TestWriterMap_PageFlagStates` | Integration | PASS |
| Shared `/trip/:slug` page uses the flagged shell | `TestWriterMap_TripPageUsesTheFlaggedShell` | Integration | PASS |
| GeoJSON has coordinates in longitude, latitude order, writer count, panel URL and completion status | `TestWriterMap_FlaggedPageAndEndpoints` | Integration | PASS |
| Panel fragment serves a place and omits absent summary | `TestWriterMap_FlaggedPageAndEndpoints` | Integration | PASS |
| Bad slug, coordinates and unknown QID are rejected | `TestWriterMap_InvalidRequests` | Integration | PASS |
| Place text is HTML escaped | `TestWriterMap_PanelEscapesSourceText` | Integration | PASS |
| Embedded Madeira fixture serves writer links and an AI summary | `TestWriterMap_FixturePanel` | Integration | PASS |

`go test -cover ./internal/adapters/httpserver/... ./internal/writermap/writermaptest/...` returned `coverage: 92.2% of statements` for `httpserver` and `coverage: 92.1% of statements` for `writermaptest`.

The design-preview browser run used `node /private/tmp/writers-map-3b.browser.mjs` with the current preview and an injected map shell. It returned `PASS 375px: map shell layout, list focus, status, console` and `PASS 1440px: map shell layout, list focus, status, console`. The full 3c preview scenario remains a separate task.

The installed `golangci-lint` 2.4.0 first stopped `make test lint build`: `the Go language version (go1.26) used to build golangci-lint is lower than the targeted Go version (1.27.2)`. Rebuilding that version with Go 1.27 still failed to decode Go 1.27's export data. A temporary `golangci-lint` 2.14.0 was built with Go 1.27.2 in `/private/tmp`. `PATH=/private/tmp:$PATH make test lint build` then exited 0: all Go packages passed, lint reported `0 issues.`, and `go build -o bin/traveltab ./cmd/web` succeeded. `go vet ./...` also exited 0. The local installed linter still needs updating for an unmodified `make lint` command to work.
