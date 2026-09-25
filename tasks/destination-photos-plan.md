# Destination photos for every city search

Status: implemented 2026-09-24; see [results](destination-photos-results.md). Updated 2026-09-24.

## Objective and required behavior

Show a photograph of the resolved destination in the hero area beside its weather information. Searching Paris must display Paris; searching another city must resolve a photograph for that city without adding it to a local manifest first.

This applies to every successful destination search, initial page load with a destination, and shared-trip page. “Search” means submitting the existing destination form; this task does not add autocomplete or requests on every keystroke.

**Live requests are required when needed.** After weather/geocoding resolves the destination, invoke the photo resolver regardless of whether the existing weather/video cache was hit. If no usable photo is cached, or its metadata has expired, request the external provider during that same user request and include the result in the rendered response. Do not require a background job, a previously generated itinerary, a curated entry, or a second search to discover a city's photo.

A valid cached photo may accelerate a repeat search. An expired photo may be retained when its live refresh fails. Never reuse a different destination's photo. Provider outages and places without suitable published photographs prevent a literal guarantee of a real photo for every city; those cases must show an explicit, stable unavailable state while the rest of the destination remains usable.

## Current implementation

| Area | Current behavior and required change |
| --- | --- |
| `internal/adapters/httpserver/presentation_assets/manifest.json` | Contains only Lisbon. Keep this verified local asset as an optional override, not the catalogue of supported cities. |
| `internal/adapters/httpserver/presentation.go` | `newPresentation` performs only local city/country matching. Keep formatting separate from network lookup and merge the resolved photo into the view. |
| `internal/adapters/httpserver/server.go` | Both initial and HTMX responses construct presentation from `TemplateData`. Resolve photos for both cache hits and fresh data. |
| `internal/adapters/httpserver/trip.go` | Shared pages construct presentation separately; use the same resolver here. |
| City cache | `checkDatabase` looks up city name alone. Validate cached identity against the newly resolved country and coordinates before using it; a namesake city's cached data must not determine the photograph or weather. |
| `internal/adapters/api/wikimedia.go` | Existing Wikimedia client supports planner landmarks and P18 image filenames. It does not provide a destination-photo service or full attribution metadata. Avoid running the expensive planner discovery pipeline just to get a hero. |
| `internal/adapters/sqlite/sourcecache.go` | Existing source cache provides reusable storage patterns and stale-on-error behavior. Its current coordinate-only interface and single TTL need adaptation for destination identity and shorter negative caching. |
| `views/weather_display.go.tpl` | Renders `.Presentation.Hero`; an image error only hides the image. Add an explicit failure state and remove orphaned credits/captions. |
| `cmd/design-preview/` | Deterministic fixture server. Add an injected photo fixture; keep automated checks independent of live providers. |

## Proposed design

Use a keyless Wikimedia-backed destination-photo source, injected through an application interface. Resolve identity from the successful weather/geocoding response, never from unvalidated search text alone.

Flow: submitted search → resolved city/country/coordinates → validate existing page-cache identity → curated override or photo cache → live lookup on miss/expiry → photo plus attribution → full-page or HTMX response.

### Contract

Add `internal/application/destinationphoto.go` with these proposed types:

```go
type DestinationIdentity struct {
    City, Country string // resolved name and normalized ISO country code
    Lat, Lon      float64
}

type DestinationPhoto struct {
    URL, Alt, Credit, CreditURL string
    License, LicenseURL         string
    Width, Height               int
    SourceID                    string // selected city entity/article
}

type DestinationPhotoSource interface {
    Photo(context.Context, DestinationIdentity) (*DestinationPhoto, error)
}
```

Use `(nil, nil)` only for a completed lookup with no suitable image. Return errors for timeouts, malformed responses, rate limits, and upstream failures. Keep the HTTP-only `imageView`; map this model into it. Photo metadata remains separate from weather/video cache JSON.

### Lookup and image selection

1. Preserve the local Lisbon override when identity matches. Every other resolved destination is eligible for live lookup.
2. In a new `internal/adapters/api/destinationphoto.go`, find a bounded set of candidate city entities/articles using the resolved name and country. Validate name/aliases, geographic coordinates and country before selecting one. Prefer a matching city entity and its P18 photograph; use its verified Wikipedia article's free page image as a secondary candidate. Never accept the first name-only search result.
3. Define and test a conservative coordinate-distance threshold, initially 25 km, together with name and country matching. Retrieve country ISO codes from entity metadata as needed. Reject ambiguous matches rather than silently choosing another place. Cover local names/diacritics and namesakes, including within one country.
4. Resolve candidate files through Commons image information. Request a bounded thumbnail around 1280 px wide, dimensions, author, source page and license metadata. Prefer landscape photography; reject flags, logos, maps, unsupported media and missing required attribution. Do not guess thumbnail URLs from filenames.
5. Bound discovery to at most five city candidates and three image candidates. Use one total lookup deadline, initially four seconds, and the existing identifying `UserAgent`. Honor cancellation and HTTP/API errors; avoid unbounded retries. Record measured cold-search latency before adjusting this budget.
6. Convert provider HTML metadata to plain text with an HTML parser; never pass it through `template.HTML`. Validate image/source/license URLs before rendering. A caption can say “View of <city>”; do not invent landmarks or copy Lisbon's caption to other cities.

MediaWiki documents [free page-image selection](https://www.mediawiki.org/wiki/Extension:PageImages#API) and [image URLs, dimensions and extended metadata](https://www.mediawiki.org/wiki/API:Imageinfo). Extended metadata may contain HTML. These APIs are the proposed building blocks; verify actual destination matches with live samples during implementation.

### Cache and failure rules

- Use a versioned key containing normalized city, country and coordinates rounded to three decimals. Metadata must include enough identity to validate a hit. Never key photos by city name alone.
- Cache successful photo metadata for 30 days. On expiry, attempt a live refresh in the current request; keep the old photo when that refresh fails or produces no replacement.
- Cache a confirmed no-photo result for at most one hour. Do not cache transport errors as a successful no-photo result or overwrite a known photo with an error.
- Treat unreadable/malformed entries and incomplete photo metadata as misses, and fetch live. Cache read/write failure must not prevent a live result from rendering.
- The first successful uncached lookup must display its photo in the same response. A cache entry must never be a prerequisite for showing a picture.
- Initially cache metadata, not image bytes. The browser loads the selected Commons thumbnail. If image download fails, display the explicit unavailable state; metadata caching alone is not proof that image bytes remain available. A persistent image-download cache is a separate enhancement if live acceptance exposes an availability problem.

## Implementation work packages

### 1. Shared contract and destination identity

Files: new `internal/application/destinationphoto.go`; `internal/adapters/httpserver/server.go`, `trip.go`, and related cache tests.

- [x] Add the photo-source contract and identity normalization/validation.
- [x] Add `SetDestinationPhotoSource` to the HTTP server, following the planner injection pattern; a nil source remains valid for existing tests and fixture servers.
- [x] Validate page-cache hits against the freshly resolved destination. Refetch mismatched data and use the fresh `GeneralWeatherInfo` for displayed weather and photo lookup. Preserve compatible legacy cache entries and existing sitemap behavior; no destructive cache migration is needed.
- [x] Add regressions for Paris, France versus Paris, Texas, and same-country namesakes with different coordinates. Keep cache errors and old-cache compatibility covered.

Exit: no previously cached namesake can replace the resolved destination's identity in either search or shared-page rendering.

### 2. Live Wikimedia adapter

Files: new `internal/adapters/api/destinationphoto.go` and `destinationphoto_test.go`; small shared Wikimedia helper changes only if necessary.

- [x] Implement bounded identity resolution, photo selection and attribution extraction using the contract above.
- [x] Inject the HTTP client so unit tests use recorded/stubbed responses without internet access.
- [x] Test city matches, aliases, disambiguation, wrong countries/coordinates, no image, malformed responses, timeout, 429/500 errors, metadata HTML, unsafe URLs and invalid image candidates.
- [x] Keep destination-photo requests independent of itinerary generation and API keys.

Exit: an uncached destination resolves to a matching, attributed photograph or a clearly classified no-result/error.

### 3. Persistent photo cache

Files: new `internal/adapters/sqlite/destinationphoto.go` and corresponding tests; reuse `source_cache` storage patterns without changing planner cache semantics.

- [x] Add `Store.NewCachedDestinationPhotoSource` with identity-aware keys and positive/negative TTLs.
- [x] Test live lookup on misses, expiry and corrupt entries; warm hits; stale retention; negative-cache expiry; country/coordinate isolation; and cache storage failure.
- [x] Verify an empty database works without prewarming and that an upstream error does not poison subsequent searches.

Exit: caching improves repeat searches while cold searches fetch live and produce a photo in their first response.

### 4. Rendering and production wiring

Files: `cmd/web/main.go`; `internal/adapters/httpserver/presentation.go`, `server.go`, `trip.go`; `views/weather_display.go.tpl`; `public/redesign/destination.css`; image-error behavior in the existing frontend code if needed.

- [x] Wire the cached live adapter in `cmd/web`. Keep provider construction out of HTTP handlers and application models.
- [x] Add one server-side presentation builder used by initial, HTMX-search and shared-trip responses. Resolve photos after destination identity is known, including page-cache hits. Construct it once per response.
- [x] Preserve Lisbon's override and merge live photo metadata for all other cities. Failed lookup must not fail weather, planning or destination search.
- [x] Keep the existing desktop/mobile hero dimensions, crop behavior and attribution links. Add useful alt text and an explicit photo-unavailable state.
- [x] On image load failure, hide the failed image's credit/caption with it and reveal the fallback. Ensure this works after HTMX swaps and history restoration without duplicate listeners.
- [x] Confirm a successful search replaces both the previous destination's photo and metadata. Keep lookup tied to the response being rendered; no detached update may paint an earlier city's photo over a later search.

Exit: searching an uncached Paris renders a Paris photograph beside Paris weather immediately, and repeated city changes retain the correct association.

### 5. Fixture, browser acceptance and documentation

Files: `cmd/design-preview/` fixtures/tests; `tasks/redesign/checks/`; new evidence directory `tasks/artifacts/destination-photos/`; `README.md`, `docs/architecture.md`, `tasks/redesign/preview.md` and `CHANGELOG.md`.

- [x] Add deterministic Paris and another destination, an attributed photo result, no-photo response, provider failure and broken-image case. Test source calls with a counting stub; fixture tests must not call Wikimedia.
- [x] Update existing assumptions that every non-Lisbon city has an empty hero. Preserve a deliberately missing-photo scenario.
- [x] Exercise initial HTML, HTMX search, shared reload and browser Back/Forward. Test Lisbon → Paris → another city, a rapid search sequence, and an image failure at 375 px and 1440 px.
- [x] In the browser, assert the actual image loaded (`complete` and `naturalWidth > 0`), city identity, credits, stable layout and no horizontal overflow. A valid `src` alone is insufficient.
- [x] Run live cold lookups for Paris, Porto, Tokyo and a smaller town. Verify selected identity, image download and credit/license links; repeat with cache disabled/empty. Record failures honestly rather than counting fixture success as provider coverage.
- [x] Document the live lookup and TTL/failure behavior. Update the old local-manifest-only guidance, including the presentation-data section of `docs/design.md`. Link new verification separately from the historical September repair evidence.

Exit: reproducible evidence covers both live discovery and UI behavior, with any provider coverage limits recorded.

## Order and verification

Complete package 1 first. Packages 2 and 3 can then be implemented independently against the frozen interface. Integrate package 4 after both are available; finish with package 5. If delegated, assign distinct files and one integration owner for HTTP rendering and production wiring.

Run focused package tests while implementing, then the project checks:

```sh
go test -race -count=1 ./...
make lint
make build
git diff --check
```

Run `node --check` for any modified JavaScript, the existing fixture HTTP/browser checks, and the new photo scenarios using the [preview guide](redesign/preview.md). Build the Docker image if wiring or packaged assets change. Live API/browser checks are separate from the offline automated suite.

Preserve the current uncommitted repair work. Record implemented files, commands, test results, live city samples, latency, screenshots and remaining limits in `tasks/destination-photos-results.md`. This plan does not itself implement the feature or authorize deployment.
