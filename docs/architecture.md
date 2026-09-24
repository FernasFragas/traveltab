# Package structure

Command entry points live in `cmd/`; reusable application code is private to this module under
`internal/`. Templates and static assets remain in `views/` and `public/`, relative to the process
working directory. Run the commands from the repository root, as before.

| Package | Responsibility | Internal dependencies |
|---|---|---|
| `internal/planner` | Trip types, ranking, grouping, scheduling, geometry and stay selection | None |
| `internal/application` | Weather/video report services, trip service, shared models and ports | `planner` |
| `internal/config` | Environment and `.env` configuration | None |
| `internal/adapters/api` | External HTTP provider clients, including Wikimedia, Open-Meteo and Overpass | `application`, `planner` |
| `internal/adapters/httpserver` | Fiber routes, sessions, HTML/HTMX rendering and request analytics | `application`, `planner` |
| `internal/adapters/sqlite` | Database lifecycle, city/source caching, visit persistence and queries | `application`, `planner` |
| `internal/placetypes` | Offline Wikimedia collection, classification rules and Go map generation | `adapters/api`, `planner` |
| `internal/guides` | Offline Wikivoyage collection, a local Ollama call and generation of `guides/guides.json`. Used only by `cmd/guides`; nothing in `internal/adapters/httpserver` or `views/` reads its output yet | none of the above (standalone) |

```mermaid
flowchart TD
    Command[cmd/web] --> Config[config]
    Command --> HTTP[adapters/httpserver]
    Command --> API[adapters/api]
    Command --> SQLite[adapters/sqlite]
    Command --> App[application]
    HTTP --> App
    API --> App
    SQLite --> App
    App --> Planner[planner]
    HTTP --> Planner
    API --> Planner
    SQLite --> Planner
```

`application.Storage` is the dashboard's persistence interface. `sqlite.Open(path)` returns a
store that implements it. Each store owns its connection, and `Store.Close()` releases it.
The store's `NewCachedPlaceSource`, `NewCachedForecastSource` and `NewCachedStaySource` methods
wrap planner sources using that same connection. There is no production package-global database.

Application services and planner logic do not import adapters. The HTTP adapter does not open
databases or construct provider clients. `cmd/web` loads configuration, opens storage, constructs
providers and passes them to services and handlers. `cmd/loadtest-server` follows the same
wiring with fixture reporters. `cmd/placetypes` handles flags and calls `placetypes.Run`.
`cmd/guides` is a separate command with no place in this dependency graph: it talks to
Wikivoyage and a local Ollama server and writes `guides/guides.json`, and nothing else imports
it or reads its output — deliberately, so a human reviews the generated text before anything
wires it in.

`internal/adapters/httpserver` also holds shareable-link and export code: `slug.go`
(`Slug`/`ParseSlug`, the `city-country` `/trip/:slug` path segment), `export_ics.go`/
`export_kml.go` (the `.ics`/`.kml` routes), `googlemaps.go` (the per-day walking-directions
link), `sitemap.go` (`/sitemap.xml`, built from cached cities only) and `ratelimit.go` (caps how
many never-before-seen cities `/trip/:slug` will plan per minute). `sitemap.go` currently stores
its city list under a reserved key in the existing city-cache table, because `application.Storage`
has no dedicated "list cached cities" method — a real one would let it drop that workaround.

## Presentation and assets

`index.go.tpl` owns the document shell and calls `content_fragment.go.tpl` for the destination
body. Search requests render that same fragment. `plan_response.go.tpl` replaces the trip form
and uses out-of-band HTMX swaps for the itinerary and stays. All four section anchors exist
before a plan is generated.

`presentation.go` decorates HTTP views from an embedded image manifest. Images live in
`public/images/`; destination matching includes city and country, and optional stay imagery
also requires an unambiguous property identity. Decoration does not alter provider/domain types
or cached JSON. The production Docker image includes templates and public assets.

The [design reference](design.md) records template inputs, the planner response contract and
style boundaries. The [credential-free preview](../tasks/redesign/preview.md) exercises real routes
and templates. See [verification results](../tasks/project-fix-results.md) for tested behavior
and [maintenance notes](maintenance.md) for remaining work.

Tests live with their packages. HTTP integration tests use a temporary SQLite store and real
root templates; source and city fixtures remain next to their packages.

## Verification

```sh
make test
make lint
make build
docker build -t traveltab .

# Optional live checks retain their explicit opt-in.
LIVE_API_TESTS=1 go test -run Live ./internal/adapters/api/
RECORD_FIXTURES=1 go test -run Cities ./internal/planner/
```

`go run ./cmd/placetypes` now writes `internal/planner/placetypes_gen.go` by default.
