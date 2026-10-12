# Maintenance notes

Setup, routes, providers, and generator commands are in the [README](../README.md).
Package boundaries are in [architecture.md](architecture.md) and the UI contracts are in
[design.md](design.md). The redesign is implemented, and the
[September 2026 results](../tasks/project-fix-results.md) record its verification and limits.
Open work is listed under [Next up](#next-up).

## Dependency and vulnerability checks

Dependabot checks Go modules, GitHub Actions, Docker base images and the three browser
libraries every Monday. Minor and patch Go/npm updates are grouped; major updates stay
separate. Version updates have a cooldown (14 days for Go/npm major releases, seven for
minor releases and three for patches; seven days for Actions and Docker). Security updates
are not delayed by cooldown. Review and merge every PR manually; auto-merge is not enabled.

| Tool | Coverage | Runs |
|---|---|---|
| govulncheck | Vulnerable Go code reachable from the application | CI after tests; also Mondays 06:00 UTC on `main` |
| npm audit | Browser libraries in `package-lock.json` (HIGH and CRITICAL) | CI after govulncheck; also Mondays 06:00 UTC on `main` |
| Browser checks | `interaction-regressions.mjs` and `browser-smoke.mjs` in headless Chrome against the design preview | PRs that change `package.json` or `package-lock.json` (`browser.yml`); screenshots are uploaded on failure |
| Trivy | Debian packages and Go dependencies in the built image | CI after the Docker build; also Mondays 06:00 UTC on `main`. Pushes and weekly runs upload to the Security tab |

For a failed scan, read the advisory and its fixed versions, update the affected module or
image, then rerun the failing scanner and `make test lint build`. `go mod tidy` alone does
not upgrade dependencies. For standard-library findings, update both the `toolchain` in
`go.mod` and the builder tag in `Dockerfile`. The runtime image applies available Debian
updates during its build; rebuild with `--pull --no-cache` when a cached layer predates a
security fix. Trivy fails on fixable HIGH/CRITICAL findings. Do not suppress a fixable
finding; an exception in `.trivyignore` needs a specific advisory and a reason why it has
no fix and does not affect the application.

Run the scanners locally:

```sh
go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
npm audit --audit-level=high
docker build -t traveltab .
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock aquasec/trivy:0.70.0 image --severity CRITICAL,HIGH --ignore-unfixed --exit-code 1 traveltab
```

Trivy 0.70.0 matches the pinned action's default scanner. The Trivy action and the SARIF
uploader use full commit hashes with release comments so Dependabot can update them.
PR scans do not upload SARIF; fork and Dependabot PRs can use their read-only tokens.

The root `package.json` pins Bootstrap, bootstrap-icons and htmx exactly, and
`package-lock.json` records an integrity hash for each package. `make assets` (run by
`make test` and `make run`), CI and the Docker build install them with
`npm ci --ignore-scripts`: npm rejects a package whose hash doesn't match, and package
install scripts never run. `npm run vendor` then copies only the five files the page loads
into `public/vendor/` (git-ignored build output), which the existing `public/` route serves.
The rest of `node_modules/` is never published, and the image does not contain it. A
Dependabot browser-library PR needs no manual step. `TestNewAppServer_ServesBrowserLibraries`
fails if a file the page loads is missing, and `TestNewAppServer_HidesOtherPackageFiles`
fails if package files become public. If a major release moves a file, `npm run vendor`
fails; update the path in the `vendor` script in `package.json`, `views/index.go.tpl` and
the test. Server tests only prove the files are served; the browser checks prove the
planner, history and video interactions still work with the new version. `npm ci` makes installs reproducible but does not check for known vulnerabilities,
and the image keeps no npm manifests, so Trivy cannot see these libraries. `npm audit` in CI
and Dependabot alerts cover them through `package-lock.json`. Google Fonts remains external and is not tracked.

A Go minor upgrade (for example 1.28) arrives through the Docker builder tag. Raise the
`go` and `toolchain` lines in `go.mod` by hand in the same PR, run `go mod tidy`, and verify
both CI and Docker use the intended toolchain. The initial rollout uses Go 1.27.2 because
the plan's 1.27.1 has [standard-library vulnerabilities](https://pkg.go.dev/vuln/GO-2026-6617).

Owner rollout steps: in Settings → Code security, enable Dependency graph, Dependabot
alerts and Dependabot security updates. After merging, check Insights → Dependency graph
→ Dependabot for all four ecosystems and any configuration errors. Confirm Trivy
results appear under Security → Code scanning. These hosted checks require the owner to
merge/push the changes; local validation cannot establish their result.

The weekly run rescans `main` with fresh vulnerability databases and a freshly built image;
it does not rebuild or redeploy the running Fly.io machine. GitHub disables scheduled
workflows after 60 days without repository activity; re-enable CI under Actions if that
happens.

## Accessibility limits

- **Focus order below 900px.** The planner is shown before the map through CSS `order`, but the
  DOM keeps the map first for the desktop reading order, so keyboard focus visits the map card
  (three stops) before the form. Reordering the DOM would move the mismatch to desktop.
- **The map frame is a third-party document.** Its own focus indication and keyboard behaviour
  are unverified until the live check; the page scrolls it into view and rings the card.
- **Not announced:** the start of a destination search; the hidden spinner text stays in the
  reading order. Below the 320px WCAG reflow width (375px at 200% is 188px) the page scrolls
  horizontally.
- Re-run `tasks/redesign/checks/a11y-audit.mjs` after any colour or layout change; it fails on
  regressions (see the [reproduce steps](../tasks/artifacts/redesign/final-verification-2/README.md#reproduce)).

## Planner decisions and limits

- Wikipedia coverage alone ranks administrative areas and other non-attractions highly.
  The generated Wikidata type filter is therefore required; unknown types are excluded.
  Classification uses exact deny types because broad ancestor exclusions can remove real
  landmarks. Review changes to `internal/placetypes/rules.go` and the generated map together.
- Indoor/outdoor labels are approximations. Station buildings can classify as outdoor,
  and market squares can classify as mixed. Mixed counts as sheltered during rain scheduling.
- Wikipedia geosearch splits a capped 10 km search into seven smaller searches. A smaller
  search can still hit the 500-result cap; there is no recursive splitting.
- Walking distances are straight-line estimates, not street routing. Compact grouping targets
  three or four stops, but isolated places can remain alone and sparse cities return fewer days.
- Requested dates represent local calendar days. Forecast certainty describes the date horizon;
  it does not prove that forecast data exists. Check rain-data availability separately.
- Shared plans are rebuilt from current data. With the same place data, grouping is deterministic;
  refreshed forecasts can change day order. Links do not preserve a historical snapshot.
- Source cache refresh failures serve stale entries when available. Cold requests still depend on
  public API latency, particularly Overpass retries. Parallel Wikimedia lookups can also time out.

## Destination photo limits

- A lookup needs a Wikidata entity that matches the resolved name, country and coordinates.
  Places the search ranks below its top five names are found through a nearby-article fallback;
  a country or territory whose ISO code differs from its Wikidata country (for example Puerto Rico
  against the United States) is refused, and so is a genuinely ambiguous match. Each refusal shows
  "Photo unavailable" rather than a guess.
- During a Wikimedia outage every uncached search waits up to the four-second lookup deadline
  before showing "Photo unavailable" (in parallel with the video request); errors are not cached,
  and there is no circuit breaker yet.
- Only metadata is cached. Images load from Commons in the browser, and some are large (a
  1.7 MB PNG was selected for "Lisboa"). Caching image bytes is a separate enhancement if that
  matters.
- The photo is the entity's own Wikidata image, which is sometimes a landmark rather than a
  skyline, and the caption is generic ("View of <city>").

## City guide limits

- **Only the 20 starter cities have a guide.** Every other search gets the "no reviewed guide yet"
  card with a Wikivoyage search link. Adding a city means adding it to `internal/guides/city.go`,
  generating it (`go run ./cmd/guides -only=...`) and reviewing it by hand; see
  [the review notes](../tasks/guides-review.md#re-reviewing).
- **A guide is matched by the resolved name and country.** OpenWeather decides the name, so a
  search that resolves to a local spelling ("Lisboa", "Wien") shows the missing state, not a guess.
- **The review was done by an AI from free sources.** A person should skim the intros before a
  wider launch. Wikivoyage pages move on: five had newer revisions by the review date, and the
  intros are pinned to the reviewed revision (linked as a permalink), not to today's page.
- **Regeneration hides a guide.** A city whose Wikivoyage revision changed is regenerated as an
  unreviewed entry, replacing the reviewed text in the file (git keeps the old one) until it is
  reviewed again.
- The site does not check the guide file for factual accuracy; the tests check its shape
  (attribution fields, review date, length and that it round-trips through the generator's writer).

## Writers' map

**Status (2026-10-11):** the map, the WordPress client, the capped sync job and the matcher exist.
The sync runs when `WRITERS_SYNC=1` and the map shows its results when `WRITERS_MAP=1`. A first
live run (see [post-limit-results.md](../tasks/post-limit-results.md#coverage-l4)) indexed 100
posts. The summary pass (`WRITERS_AI=1`) is separate and not covered by this section.

After deploying the panel photo cache fix, visit each destination whose map must be ready to
serve so its current-version base places and map result are rebuilt. The monthly writers sync
refreshes only destinations already present under the current cache version, and an already-current
sync is not forced by restarting with `WRITERS_SYNC=1`. Check that each cold map request completes
before relying on it; a local Lisbon recompute exceeded 35 seconds.

### Basemap sources (tasks 3e and M1–M7)

The default 3D map lazy-loads **MapLibre GL JS 5.24.0** (UMD JavaScript and CSS from jsDelivr).
Its vector basemap is the always-dark [OpenFreeMap dark style](https://tiles.openfreemap.org/styles/dark)
(owner decision 2026-10-12; no OS or theme switching), which loads its tiles, glyphs and sprite from OpenFreeMap. MapLibre shows the style attribution,
“OpenFreeMap © OpenMapTiles Data from OpenStreetMap” (unchanged). That style has no 3D buildings, so
the core adds one `fill-extrusion` layer (`buildings-3d`) on every `style.load` from the style's
OpenMapTiles `building` layer (`render_height`, from zoom 14, `#2c3044` to `#46414d`, opacity
0.85) below the first label layer after the buildings. It adds nothing if the style already has a
`fill-extrusion` layer, such as a custom `data-map-style-url`. On every `style.load` of the default
style (not a custom `data-map-style-url`) the core also recolours the OpenFreeMap layers from the
`palette` table in `writers-map.js` (land `#18263b`, water `#000e29`, parks `#203838`, roads
`#404860` to `#5a6274`, labels `#e0e0e0`), a navy palette sampled by eye-dropper from the Electroverse
map; colours only, none of their style, tiles or fonts. The same table also sets road weight and
label size (paint and layout properties, default style only): minor roads, and secondary and
tertiary roads (which share the primary layers, so they thin by `class`), are thinner and dimmer
between z10 and z13; primaries, trunks and motorways keep their weight; town and neighbourhood
labels are 11 px. A layer id the style lacks is skipped, so an OpenFreeMap layer rename just
leaves that layer in the stock dark colours. The preview fixture style
(`cmd/design-preview/mapstyle.go`) carries the same colours and the same minor-road weight; it has
no glyphs and so no label layers to mirror. With terrain on, MapLibre can leave labels unplaced
after a bare `jumpTo` at pitch 0 until the next camera change or `map.resize()`; screenshot scripts
that jump the camera themselves should call `resize()` after the jump. The terrain source is
`https://tiles.mapterhorn.com/{z}/{x}/{y}.webp`, 512-pixel Terrarium-encoded DEM tiles. Its on-map
credit is “© Mapterhorn”; its source data includes Madeira IGP (CC BY 4.0) and Copernicus GLO-30.
Neither provider needs an application key. Both are external dependencies without an application
SLA; review their terms and attribution before changing the style or terrain host. The map shell's
`data-map-style-url` and `data-map-terrain-url` select local fixtures or replacement hosts.

If WebGL is unavailable, or the MapLibre style, tiles or terrain fail, the card switches to its
Leaflet 1.9.4 raster fallback. An explicit `data-map-engine="leaflet"` forces that engine for the
preview. The fallback uses EOX imagery and switches to OpenStreetMap standard tiles on an image
tile error; it caps zoom at 18. The raster fallback is also subject to the OSM tile policy below.

If OpenFreeMap becomes unavailable long term, self-host a compatible OpenMapTiles vector archive
as PMTiles on static storage that supports HTTPS, CORS and byte-range requests. Add the PMTiles
protocol to the MapLibre loader, point a locally hosted copy of the dark style at the archive,
and serve its glyphs and sprite from an allowed origin. Then set `data-map-style-url` to that style,
keep the OpenStreetMap/OpenMapTiles attribution, update the CSP origins and test the 375 px and
1440 px views. `data-map-style-url` alone cannot read a PMTiles archive with the current loader;
the protocol integration is a code change. Terrain can be replaced independently through
`data-map-terrain-url` with compatible 512-pixel Terrarium tiles.

#### Leaflet raster fallback (task 3e)

The owner accepted this source and its older-image tradeoff on 2026-10-11.

The fallback uses EOX's **2016 EOxCloudless Sentinel-2** WMTS layer, `s2cloudless_3857`:
`https://tiles.maps.eox.at/wmts/1.0.0/s2cloudless_3857/default/GoogleMapsCompatible/{z}/{y}/{x}.jpg`.
EOX [licenses this year under CC BY 4.0](https://cloudless.eox.at/license-non-commercial),
which permits commercial use with credit. Its required visible map credit is
“EOxCloudless https://cloudless.eox.at by EOX IT Services GmbH (Contains modified Copernicus
Sentinel data 2016 & 2017)”. The map credit links to EOX and the
[CC BY 4.0 licence](https://creativecommons.org/licenses/by/4.0/). No API key or billing account is
needed for EOX's [direct WMTS integration](https://cloudless.eox.at/documentation/usage).
EOX states a 4096-pixel maximum request size; the map requests 256-pixel tiles. No numeric tile
request quota is published on those pages. The layer is fixed to 2016 imagery, so it cannot show
recent construction. Its advertised coverage is global; only Funchal was checked directly.

The choice was checked on 2026-10-11 against these alternatives:

| Source | Licence and access | Decision |
| --- | --- | --- |
| [Esri World Imagery](https://developers.arcgis.com/rest/basemap-styles/service-data/) | [Token required](https://developers.arcgis.com/documentation/mapping-and-location-services/mapping/basemaps/introduction-static-basemap-tiles-service/); [2 million free tiles, then billed](https://developers.arcgis.com/pricing/). Esri and data-provider attribution required. | Excluded by the no billing-linked key rule. |
| [EOX 2018–2025](https://cloudless.eox.at/license-non-commercial) | Free direct WMTS for noncommercial use under CC BY-NC-SA 4.0; commercial use requires a separate licence. Same 4096-pixel request limit; no numeric tile quota published. | Excluded because this site's commercial status need not be assumed. |
| EOX 2016 | CC BY 4.0, direct WMTS without a key or fee, with the exact EOX credit above; same request limit. | Selected despite older, lower-detail imagery. |

The map shell's `data-map-tile-url` and `data-map-attribution` override the fallback together for
fixtures or a later provider. An explicit OSM URL gets the OSM credit if no credit is supplied.
If a tile fails, the browser switches to
[OpenStreetMap standard tiles](https://operations.osmfoundation.org/policies/tiles/) and their own
credit. OSM is a best-effort fallback with no SLA; browser requests must preserve the Referer and
normal cache headers, and must not prefetch large areas. On 2026-10-11, EOX's live WMTS capabilities
listed `s2cloudless_3857` with the `GoogleMapsCompatible` matrix, and individual Funchal tiles at
zoom 12 and 14 returned 256×256 JPEG imagery. The automated preview retains local fixture tiles;
the optional `tasks/redesign/checks/writers-map-imagery-live.mjs` browser check verifies the live
3D and raster sources and their on-map credit.

The writers' map reads its allowlist from the embedded `writerdata/sources.json` file and reviewed
matching policy from `writerdata/overrides.json`. Source status is `active`, `blocked` or `removed`;
the date in the source file records the last check. Add or change a source only after confirming its
public WordPress API and language. The API base must be the site's `/wp-json/wp/v2/posts` endpoint.

Sync and summary requests use an identified TravelTab user agent, honor each site's `robots.txt`,
and share a per-host limiter of at least 1.5 seconds between requests. If a site returns 403, mark it
`blocked` and stop requesting it. Do not retry through alternate identities or endpoints to work
around a block. A robots disallow also stops fetches. Record status changes and their check date in
the embedded source data.

The sync runs once at startup (a source that never finished is due at once) and then, off peak,
once per calendar month when `WRITERS_SYNC=1`. It updates post, name, day, destination and
summary rows in place. It retains names, positions, post metadata and generated summaries; raw post
text and HTML are processed in memory and discarded after indexing or summarization. Never store,
log or commit post text, excerpts or fetched response bodies. The optional monthly summary pass is
gated by `WRITERS_AI=1`; it sends mention passages to the configured LLM Gateway and keeps only the
English summary, source post references, model and generation time.

**The 100-post cap** (`writermap.MaxIndexedPosts`). The index holds at most 100 posts in total, across
all blogs. Every run picks the set again from scratch; there is no cursor crawl or deleted-posts
sweep any more. How the posts are picked:
1. Search terms are the destination names: the override areas (`writerdata/overrides.json`) plus
   every destination the app has cached (a city page or map visit caches it). One `search=` request
   (first page, newest first) per term per blog.
2. A search hit is a **destination post** only if its title or URL slug names a known place (the
   matcher's names for the cached destinations, accent- and case-insensitive). Hits that only mention
   a place in the body are dropped.
3. Each blog's share of the 100 is proportional to its destination posts (largest remainder, at least
   one for any blog that has one, never more than it has).
4. Within a blog, posts are ranked by how many of the blog's newest 1,000 comments they have
   (up to 10 pages), then the newest first. A blog without readable comments (404) takes its newest.
5. Only the chosen posts are fetched (`include=`) and indexed. Indexed posts that were not chosen
   are deleted, so a post that drops out of the top is purged on the next monthly run.

**Blocked and empty blogs.** A blog that is blocked (allowlist or 403) keeps its already-indexed
posts, and they count inside the 100: the other blogs share what is left, so the total never exceeds
the cap. A blog that has indexed posts but whose search suddenly returns nothing (no error) keeps its
rows too, counts inside the cap, logs `writers sync <host>: warning: search returned no posts ...`, and is not
marked finished, so the next run retries it. A blog that never had posts and finds none is just
skipped. To remove a blog's rows for good, use the takedown below.

For a takedown, mark the source `removed` in `writerdata/sources.json`. The next sync purges that
host's indexed rows and removes its post references from summaries; recompute affected destination
results and summaries. To hide one generated summary, add its Wikidata QID to
`summary_hide_qids` in `writerdata/overrides.json`; the next summary pass suppresses it. Keep the
flags off until the corresponding feature has been reviewed and is ready to launch.

## Next up

In priority order. Updated 2026-10-11. Done work (redesign, destination photos, sitemap storage,
city guides, slug helpers, city autocomplete) is recorded in the [changelog](../CHANGELOG.md).

1. **Writers' map.** Progress lives in the task files, not here: [tasks/todo.md](../tasks/todo.md)
   (PRs 1–9), [post-limit-todo.md](../tasks/post-limit-todo.md) (100-post index) and
   [map-3d-todo.md](../tasks/map-3d-todo.md) (MapLibre 3D map).
2. **Close the verification gaps.** Live checks of the map embed, Wikimedia photos, YouTube
   playback and the Booking landing page in a normal browser (see the
   [external checks](../tasks/artifacts/redesign/final-verification/external-results.md)), plus
   the manual autocomplete checks in [its plan](../tasks/city-autocomplete-plan.md#verification).
3. **Export interoperability.** Import the ICS and KML exports into real calendar and map apps.
   Parser checks don't prove importer compatibility. Event times are defaults, not real visit
   schedules.
4. **Indexing after deployment.** Check the public trip URLs, canonical tags and sitemap in
   search-engine diagnostics. Local tests can't show this.

## Historical verification boundaries

The September 19–20, 2026 planner work used recorded Lisbon, Tavira, and Kyoto fixtures for
repeatable offline acceptance checks. The Kyoto stay fixture uses an older center coordinate,
so it does not establish hotel availability around every computed itinerary center.

That work also checked real generated ICS/KML files with independent parsers, but did not import
them into Google Calendar, Apple Calendar, Outlook, or Google My Maps. Historical browser checks
used fixture providers and did not verify the production map or booking landing page.

Historical Lisbon Wikimedia measurements improved from 18.4–18.6 seconds with one worker to
6.3–7.8 seconds with four; one parallel attempt timed out. These are observations from that run,
not performance guarantees. For repeatable load testing and measured scope, use the
[load-test guide](../loadtests/README.md) and its [recorded results](../loadtests/validation.md).
