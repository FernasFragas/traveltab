# Place panel photo fix

**Source:** owner bug report, no plan task. Cabo Espichel Lighthouse (Lisbon): the credit showed, the `<img>` was broken.

**Journey:** As a reader, I want the place panel photo to load, so that I see the place and its credit.

## Cause

`addPhotoCredits` in `internal/adapters/api/wikidatanearby.go` put the original file URL in `Photo` and the Commons description page (HTML) in `PhotoURL`. `views/place_panel.go.tpl` uses `PhotoURL` as the `<img src>`. The contract says `photo` is the Commons file identifier and `photo_url` the image URL. `planner/geo.go` also reads `Photo` as a file name, so the old value broke that too. The fixture had correct values, so tests passed.

## Fix

- Imageinfo call adds `iiurlwidth=640`. `PhotoURL` is `thumburl`, or `url` when no thumb is returned, checked with `safeURL` (upload/thumb.wikimedia.org). The Commons page is still required to be a valid commons URL, but is not stored.
- `Photo` stays the Commons file identifier. `Photo` and `PhotoCredit` are cleared when no usable `PhotoURL` is found.
- Cache key `wikidata:v1:` becomes `wikidata:v2:` (key format and the `CachedDestinations` filter). Cached v1 rows are never read, so each destination is re-fetched with correct URLs on its next request. v1 rows stay in `source_cache` unused, and the monthly sync only lists v2 rows, so a destination is rebuilt when first requested, not before.
- No contract fields added. JS (`map-panel.js`, `writers-map*.js`) does not read `photo`/`photo_url`; no change.

## RED

`go test ./internal/adapters/api/ -run WikidataNearby_Places` before the fix:

```
TestWikidataNearby_Places_CreditsCommonsImage
  expected: "Pico Ruivo.jpg"
  actual  : "https://upload.wikimedia.org/wikipedia/commons/a/aa/Pico_Ruivo.jpg"
  expected: ".../thumb/a/aa/Pico_Ruivo.jpg/640px-Pico_Ruivo.jpg"
  actual  : "https://commons.wikimedia.org/wiki/File:Pico_Ruivo.jpg"
TestWikidataNearby_Places_RequestsAThumbnail   expected: "640"  actual: ""
TestWikidataNearby_Places_FallsBackToTheOriginalWhenNoThumbnail  FAIL
TestWikidataNearbyCache_IgnoresRowsFromBeforeThePhotoURLFix      FAIL (stale v1 row served)
```

`TestPlacePanel_PhotoSrcIsAnImageURLNotACommonsPage` passed on its first run: the template was already correct and only needed correct data. It guards the template side.

## GREEN

`go test -cover ./internal/adapters/api/ ./internal/adapters/sqlite/ ./internal/adapters/httpserver/`:

```
ok  weatherservice/internal/adapters/api         coverage: 82.9% of statements
ok  weatherservice/internal/adapters/sqlite      coverage: 81.7% of statements
ok  weatherservice/internal/adapters/httpserver  coverage: 92.5% of statements
```

`go vet ./...`, `gofmt -l internal cmd`, `make test build`, `make lint` (0 issues) all clean.

| Guaranteed | Test | Kind | Result |
|---|---|---|---|
| `PhotoURL` is the thumb image URL, `Photo` the file identifier | `TestWikidataNearby_Places_CreditsCommonsImage` | unit (recorded Commons response) | pass |
| Imageinfo asks for a 640 px thumbnail | `TestWikidataNearby_Places_RequestsAThumbnail` | unit | pass |
| Original URL used when no thumb | `TestWikidataNearby_Places_FallsBackToTheOriginalWhenNoThumbnail` | unit | pass |
| Old v1 cache rows are not served or listed | `TestWikidataNearbyCache_IgnoresRowsFromBeforeThePhotoURLFix` | integration (temp SQLite) | pass |
| Panel `<img src>` is the upload.wikimedia.org image, no Commons page | `TestPlacePanel_PhotoSrcIsAnImageURLNotACommonsPage` | integration (httptest) | pass |

## Live check (owner approved a few requests)

Temporary test in the api package (deleted after) called `Places` for Lisbon (38.72, -9.14): 200 pins. First four with photos, each fetched, all `200 image/jpeg`:

- Belém Tower, `Photo` = `Belem Tower - April 2019 (2).jpg`, `PhotoURL` = `https://thumb.wikimedia.org/wikipedia/commons/thumb/b/b5/...960px-...jpg?utm_source=...`
- Jerónimos Monastery, Pena Palace, Padrão dos Descobrimentos: same shape.

Commons returns a 960 px thumb for the 640 request (it rounds to standard sizes) on host `thumb.wikimedia.org`, which `safeURL` allows.

## Known gaps

- Browser rendering of the panel was not checked in `cmd/design-preview`; the change is data only, template unchanged.
- Old v1 rows stay in the DB unused.
