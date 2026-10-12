# Panel photo still broken — Tasks

Read [the plan](panel-photo-cache-plan.md) first. TDD per [AGENTS.md](../AGENTS.md) §5. No commits or pushes. Tests never hit the network.

**User journey:** *As a traveller, I want each notable place's photo to show in the panel, so that I can recognise the place before I go.*

---

### D1. Owner decision: the fix option · **owner**
- [x] Choose A, B or C (or a mix). **Owner chose B + C (2026-10-12):** tie the versions, and clean up the old rows at startup.

### P1. Invalidate the destination cache · *after D1*
- [x] Done

**Owns:** `internal/adapters/sqlite/writerdestination.go`, `internal/adapters/sqlite/writerdestination_test.go`, and `internal/adapters/sqlite/wikidatanearby.go` (version constant only, if B)

- [x] RED (Prove-It): seed a `writermap:v1` row whose place has a Commons page as `PhotoURL`. `WriterDestination` must report it as **missing**, so the source recomputes the result.
- [x] RED (if B): bumping the base-place version changes the destination key.
- [x] GREEN: bump the key (A), or build it from the shared version (B). `WriterDestinations` lists the new prefix only.

**Verify:** `go test ./internal/adapters/sqlite/ -run 'TestWriterDestination'`; `go test -cover ./internal/adapters/sqlite/` ≥80%.

### P2. Remove stale cache rows · *after D1; parallel with P1*
- [x] Done

**Owns:** the store's startup migration file in `internal/adapters/sqlite/` and its test

- [x] RED: a database holding `wikidata:v1%` and `writermap:v1%` rows has none after `Open`. Other keys (`forecast:`, `places:`, `destphoto:v1`) are untouched. A second run changes nothing.
- [x] GREEN: one idempotent `DELETE`.

**Verify:** `go test ./internal/adapters/sqlite/ -run TestOpen`.

### P3. Guard: the panel never renders a page URL as an image · *parallel with P1*
- [x] Done

**Owns:** `internal/adapters/httpserver/writermap.go` (panel and GeoJSON photo fields only), `internal/adapters/httpserver/placepanel_test.go`

- [x] RED: a place whose `PhotoURL` is a Commons page renders **no** `<img>` and no credit, and the GeoJSON leaves the photo fields out.
- [x] GREEN: accept the two image hosts already allowed by the API (`upload.wikimedia.org` and `thumb.wikimedia.org`), so any stale row that slips through degrades to "no photo" instead of a broken image.

**Verify:** `go test ./internal/adapters/httpserver/ -run TestPlacePanel`.

### P4. Runtime check and evidence · *after P1–P3*
- [x] Done

**Owns:** `docs/tdd/panel-photo-cache.tdd.md` (new)

- [x] Restart `go run ./cmd/web` with `WRITERS_MAP=1`. Open Lisbon, then click *Condes de Castro Guimarães Palace*. The photo loads from Wikimedia's `thumb.wikimedia.org` image host. Clean console at 375 and 1440 px.
- [x] Record whether the live recompute finished within the 30 s timeout (plan, side effect 1). It exceeded 35 s; the evidence links PR 8 resilience and latency work.
- [x] Note the **release step** in `docs/maintenance.md`: after deploy, visit each destination so production rebuilds its destinations.
- [x] `make test lint build` (using a Go 1.27-compatible `golangci-lint` from `/private/tmp`).

**Done when:** the palace photo shows locally after a restart with no manual database edits, the regression tests pass, and the full suite is green.
