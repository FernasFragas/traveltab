# Security review: attack surface and fix plan

Initial review: 2026-10-09. Code-review update: 2026-10-10. **Status re-check: 2026-10-15**
(branch `fix-v4` at `9e6e45e` plus the uncommitted working tree). Scope: the web app
(`internal/adapters/httpserver`, active API clients, SQLite caches, templates, browser JS)
and its deployment (`Dockerfile`, `fly.toml`). Dependency and supply-chain checks are covered
by [dependency-security-plan.md](dependency-security-plan.md).

**Method:** the initial report recorded probes against the local design preview
(`cmd/design-preview`, which uses the real `NewAppServer`). Those observations are retained
as **previously reported local results**; this update did not rerun them or independently
verify their output. The update inspected current source, installed middleware source and
existing tests. No tests, load tests or live-site probes were run for this update.

**2026-10-15 re-check:** every finding below was re-read against current source. **None of the
twelve fixes has landed**; line references were refreshed. New since 2026-10-10: the writers' map
routes (`/map/:dest.geojson`, `/map/:dest/:qid`), the WordPress sync, CDN-loaded map libraries and
`Cache-Control: no-cache` on static files. They add **finding 13** and four rows to *Existing
protections*. This re-check also ran no security probes.

Severity is a remediation priority, not a measured exploitability score. Production reachability,
proxy configuration, deployed environment variables and resource impact remain unverified.

## Summary

| # | Finding | Severity | Status |
|---|---|---|---|
| 1 | Unauthenticated runtime profiler at `/debug/fgprof` | **High** | **Open.** Code + previously reported local response |
| 2 | Expensive routes bypass the new-city limiter; its budget is shared | Medium | **Open.** From code; resource impact unmeasured |
| 3 | `/suggest` has no application length cap or rate limit | Medium | **Open.** From code |
| 4 | No application security headers (CSP, frame, nosniff, referrer) | Low / hardening | **Open.** Code + previously reported local headers |
| 5 | Stats token in the query string, compared without constant time | Low | **Open.** From code |
| 6 | Session store loaded on every request but never used | Informational | **Open.** Code + previously reported absence of cookie |
| 7 | `/stats?days=` has no upper bound | Low | **Open.** From code |
| 8 | No application retention/size limit for visits and persistent caches | Medium | **Open, wider** (writers' map caches, see 13). From code; exhaustion unproven |
| 9 | Runtime image defaults to root | Low / hardening | **Open.** From Dockerfile; deployed UID unverified |
| 10 | Active upstream clients lack overall deadlines and cancellation propagation | Medium | **Open.** From code |
| 11 | OpenWeather geocoding sends the API key over HTTP | **High** | **Open.** From code; interception not demonstrated |
| 12 | Analytics visitor ID is derived from the client IP and is reversible | Medium (privacy) | **Open** (decided, not implemented). Owner decision 2026-10-11: stop using the IP, drop unique visitors |
| 13 | Writers' map routes start unbounded Wikidata work and cache rows per request | Medium | **New 2026-10-15.** From code; only when `WRITERS_MAP` is on |
| — | Existing protections and remaining verification gaps | — | See below |

**Fix order:** 1 and 11 → 2, 3, 10, 12 and 13 → 8 → 5 and 9 → 4 and 7. Item 6 is optional cleanup.
Start with these focused changes; another scanner or a distributed rate-limiting service is
not necessary to address the findings. Keep the existing dependency checks.

---

## 1. Unauthenticated runtime profiler at `/debug/fgprof` (High)

**Evidence:** `internal/adapters/httpserver/server.go:67` runs `app.Use(fgprof.New())` with no
condition. The initial review reported that
`curl 'http://127.0.0.1:8097/debug/fgprof?seconds=2'` returned **200** after **2.0 s** with a
1.9 KB profile. Installed `gofiber/contrib/fgprof@v1.0.4` routes this path to the fgprof
handler, which accepts a positive `seconds` value without an application maximum and sleeps
for that duration while profiling. It profiles runtime stacks, including off-CPU time.

**Risk:**

- **Resource consumption.** Long or overlapping requests can prolong profiling work. Actual denial-of-service impact has not been measured. `fly.toml` specifies a minimum of one machine with one shared CPU and 1 GB RAM; it does not prove the deployed machine count.
- **Information leak.** Profiles can reveal function names and source paths useful for fingerprinting. They are not evidence of secret disclosure or exact dependency-version disclosure.

**Fix plan:**

- [ ] RED: `TestNewAppServer_HidesProfiler` checks that `GET /debug/fgprof?seconds=1` returns 404.
- [ ] Remove profiler registration from the public server. If profiling is needed later, use an explicitly enabled, private diagnostic interface. Do not rely on `ENV != "production"`: an unset variable would enable it, and `fly.toml` does not declare `ENV=production`.
- [ ] GREEN, then `make test lint build`.
- [ ] After deploy, `curl -o /dev/null -w '%{http_code}' https://<site>/debug/fgprof?seconds=1` should print 404.

## 2. Incomplete rate-limit coverage and shared budget (Medium)

**Evidence:** `ratelimit.go` allows `defaultNewCitiesPerMinute = 5` across all visitors to
one server instance. Only `tripCardForSlug` uses it, for a destination cache miss with valid
planning dates. `planTrip` (`/plan`) and `resolveTripPlan` (both exports) call the planner
without that check. `/`, `/process-form/` and `/trip/:slug` fetch weather before checking
the destination cache; photo/video work can also occur before the limiter is consulted.
Provider fan-out varies: Wikimedia performs additional geographic searches when its initial
search hits the result cap, rather than always making seven searches. The writers' map routes
added since have no limiter either (finding 13).

**Risk:**

- **Shared-budget starvation.** One client can consume the five slots and prevent other visitors from automatically planning uncached cities during that window. Cities must resolve successfully and reach the limiter; arbitrary made-up names are not sufficient. Cached destinations and `/plan` remain outside this limit.
- **Quota and availability abuse.** Unrestricted routes can trigger expensive work despite that budget. Provider throttling or quota exhaustion may degrade search and planning; cached responses can remain usable.

**Fix plan:**

- [ ] RED: use fake providers to show that `/plan`, both exports and destination routes enforce an agreed request/upstream-work budget before starting expensive work.
- [ ] RED: client A exceeding its individual budget does not consume client B's allowance while the shared safety budget still has capacity. Test the shared cap separately.
- [ ] Apply a consistent per-client policy and a global bound on expensive work. Size limits from provider quotas and expected traffic; do not automatically raise the existing cap.
- [ ] Establish the trusted proxy boundary before using client-IP headers. Trust `Fly-Client-IP` only on the Fly ingress path, not arbitrary direct connections; another proxy in front changes its meaning. See [Fly request headers](https://docs.fly.io/networking/request-headers).
- [ ] Bound any per-client map by entry count as well as expiry; a TTL alone does not bound a burst of distinct keys. Allow for legitimate visitors sharing a NAT address.
- [ ] Note in `docs/maintenance.md` that the limit is in memory and per machine.

## 3. `/suggest` lacks application input and rate limits (Medium)

**Evidence:** `suggest.go` needs at least two characters but has no application maximum
(HTTP parser/proxy limits still apply). Each uncached lower-cased query calls
`placeSuggester.Suggest` (geocoding). The cache is size-limited, but deliberately distinct
queries can keep missing it.

**Risk:** distinct cache misses can generate upstream geocoding requests as quickly as the
deployment can accept them. The cache has a 128-entry cap and the geocoder a five-second
timeout, but neither limits aggregate request rate or concurrent work.

**Fix plan:**

- [ ] RED: a 200-character query returns an empty result **without** calling the suggester (use a fake suggester that counts calls).
- [ ] Cap the query at about 64 characters.
- [ ] Apply the per-client limiter from item 2 to cache misses only.

## 4. No application security headers (Low / hardening)

**Evidence:** the initial review reported that local responses from `/` set no `Content-Security-Policy`, `X-Frame-Options`,
`X-Content-Type-Options` or `Referrer-Policy`. Only the design preview's map fixture sends a CSP.

**Risk:**

- **Missing defense in depth.** A restrictive CSP could limit some consequences of an injection; its absence does not establish XSS. Browser origin restrictions still apply.
- **Clickjacking.** Without a framing restriction, another site can embed the page. Impact is limited by the current public, unauthenticated feature set.
- **Referrer policy is implicit.** Modern browsers default to `strict-origin-when-cross-origin`, which normally omits paths and queries on cross-origin requests. Missing this header does not by itself prove full-URL leakage to third parties. See [MDN Referrer-Policy](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Referrer-Policy).

**Fix plan:**

- [ ] RED: a test that `/` returns each header.
- [ ] Add Fiber's `helmet` middleware, or set the headers by hand in one middleware.
- [ ] Write a CSP that allows only what the page uses:
  - `default-src 'self'`
  - `script-src 'self'`, `connect-src 'self'`, `object-src 'none'`, `base-uri 'self'`, `form-action 'self'`
  - Google Fonts in `style-src` and `font-src`
  - Wikimedia and YouTube thumbnails in `img-src`
  - the YouTube embed and map origin in `frame-src`
  - `frame-ancestors 'none'`
  - For the writers' 3D map (hosts read from `writers-map.js` and `writers-map-leaflet.js`, the jsDelivr, unpkg, OpenFreeMap, Mapterhorn and EOX requests were observed in the browser on 2026-10-11; OSM only loads after an EOX tile error):
    - `script-src` and `style-src`: `https://cdn.jsdelivr.net` (MapLibre 5.24.0 JS and CSS), plus `https://unpkg.com` (Leaflet 1.9.4 fallback)
    - `connect-src`: `https://tiles.openfreemap.org` (style, vector tiles, glyphs, sprite) and `https://tiles.mapterhorn.com` (terrain)
    - `img-src`: `https://tiles.openfreemap.org` and `https://tiles.mapterhorn.com`, plus `https://tiles.maps.eox.at` (fallback imagery) and `https://tile.openstreetmap.org` (OSM fallback on a tile error); also `blob:` and `data:` for MapLibre
    - `worker-src blob:` (MapLibre's worker); add `child-src blob:` for older Safari
    - Recheck the actual browser requests before enforcing the policy, and drop the Leaflet and OSM origins if the fallback stops using them.
- [ ] Look for inline scripts and styles. Prefer moving them out over allowing `'unsafe-inline'`.
  - Known inline image handlers (re-checked October 15, 2026): `views/weather_display.go.tpl` (hero `onload`/`onerror`), `views/itinerary_body.go.tpl` (stop photo `onerror`), `views/stay_card.go.tpl` (`onerror`), `views/video.go.tpl` (2× `onerror`), and new since: `views/map_card.go.tpl` (city photo `onerror`) and `views/place_panel.go.tpl` (place photo `onerror`). Image `error` events don't bubble: a shared script must listen in the capture phase and also handle images that failed before it loaded.
- [ ] Account for HTMX's injected indicator styles and evaluation-dependent features; inspect `includeIndicatorStyles` and `allowEval` before enforcing the policy. The app now vendors **HTMX 2.0.11** (`package.json`), so use the [current configuration reference](https://htmx.org/reference/#config), not the 1.x docs. Include local images/fonts and any actually used data-URI assets in the resource inventory.
- [ ] Choose an explicit referrer policy compatible with the video/map embeds. Use `Cache-Control: no-store` and `Referrer-Policy: no-referrer` specifically for private stats responses. (Static files now send `Cache-Control: no-cache`; that is a freshness fix and does not cover `/stats`.)
- [ ] Run the browser checks (`browser-smoke.mjs`, `interaction-regressions.mjs`, `a11y-audit.mjs`) with the policy enabled. Add explicit capture of CSP violations/blocked resources: `lib/browser.mjs` currently records `Runtime.exceptionThrown`, which does not cover every CSP console diagnostic. Verify search, swaps, indicators, maps and video playback as well as layout.
- [ ] Optionally ship it as `Content-Security-Policy-Report-Only` for a week first.

## 5. Stats token handling (Low)

**Evidence:** `analytics.go:58–60` reads the token from `?token=` and compares it with `!=`.

**Risk:**

- **Potential token disclosure.** Query tokens can enter request logs and browser history. No actual leak was demonstrated; cross-origin referrer leakage is not assumed (see item 4).
- **Comparison hardening.** The equality check is not constant time. No practical remote token-recovery attack was demonstrated; removing credentials from URLs is the priority.

**Fix plan:**

- [ ] RED: the token is accepted in an `Authorization: Bearer` header, and a wrong token still returns 404.
- [ ] Compare with `crypto/subtle.ConstantTimeCompare`.
- [ ] Test missing/empty/malformed credentials and an unset server token; none may authorize access. Keep tokens out of logs and prevent caching of stats responses.
- [ ] Drop `?token=` support after updating whatever reads `/stats`.
- [ ] Coordinate rotation of a token previously used in URLs after migrating its consumers, because existing logs/history may retain it. This review does not rotate secrets.

## 6. Unused session store (Informational)

**Evidence:** `server.go:69–76` calls `store.Get(ctx)` on every request and stores the result in
`ctx.Locals`, but no session reader or save was found. The initial review reported no cookie on `/`.

**Risk:** no current session-based exploit established. It is unnecessary work on every request, and anyone who later
saves the session would get Fiber's default cookie flags (`Secure` and `HttpOnly` off,
`SameSite=Lax`) unless they think to change them.

**Fix plan:**

- [ ] Check for readers of `ctx.Locals("session")` (expect none).
- [ ] Remove the middleware, the `store` variable and the `gob.Register` calls, if nothing else needs them.
- [ ] `make test lint build`.

## 7. `/stats?days=` has no upper bound (Low)

**Evidence:** `analytics.go:63–66` sets a minimum but no maximum. Only token holders can reach it.

**Fix plan:**

- [ ] RED: `days=100000` is clamped to the maximum (for example 365).
- [ ] Add the clamp.

## 8. Persistent cache and analytics growth (Medium)

**Evidence:** `internal/adapters/sqlite/sourcecache.go` keys planner records by source and
coordinates rounded to three decimals. Its TTL determines freshness; it does not remove old
keys. No general retention or entry-count limit was found for `source_cache`, `city_data`
or `visits`. `trackVisit` inserts a visit before running the destination handler, including
requests whose destination lookup subsequently fails, provided the user agent passes its
simple bot filter. `cached.get` also allows concurrent misses for one key to fetch separately.

**Risk:** varied locations and repeated page visits can increase storage and SQLite write
load. Concurrent cache misses can amplify upstream work. Filling the production volume or
causing an outage has not been demonstrated. The in-memory suggestion cache is already bounded.
Since 2026-10-10 the writers' map also writes Wikidata place results to `source_cache` under
request-controlled keys (finding 13), which widens this growth path.

**Fix plan:**

- [ ] Use local fake providers and temporary SQLite to verify growth and duplicate fetches under a small, bounded workload.
- [ ] Define retention and size budgets for visits and caches; test cleanup preserves fresh entries and the intended stale-data fallback.
- [ ] Apply the request budgets from item 2 before analytics writes where appropriate, and monitor database/volume size.
- [ ] If duplicate work is material, coalesce concurrent fetches for the same cache key with cancellation and a bounded number of active fetches. Avoid adding a distributed cache without a demonstrated need.

## 9. Runtime container defaults to root (Low / hardening)

**Evidence:** the final `debian:bookworm-slim` stage in `Dockerfile` has no `USER` directive.
The image therefore defaults to root; runtime overrides were not inspected. The application
uses `/data/weatherservice.db` on a persistent Fly volume.

**Risk:** a future code-execution vulnerability would give the attacker the application's
process privileges. Root increases the ability to modify files inside the container; it does
not, by itself, establish host compromise. An unprivileged process can still read its own
environment secrets and write its permitted database, so this is impact reduction.

**Fix plan:**

- [ ] Run the final image as a dedicated unprivileged UID/GID, keeping application files read-only to that user.
- [ ] Arrange ownership of the existing mounted database, its directory and SQLite sidecar files before rollout; avoid a recursive ownership change without checking the volume contents.
- [ ] Verify the image's effective UID and startup with a representative mounted volume, including database reads/writes. Keep a rollback path for deployment failures.

## 10. Upstream deadlines and cancellation gaps (Medium)

**Evidence:** `cmd/web/main.go` wires `NewWeatherAPI` and `NewOpenMateoAPI` into the public
weather flow. Both use `http.DefaultClient` (`openweather.go:26`, `openmateo.go:21`; YouTube too,
`youtube.go:24`), which has no overall request timeout. OpenWeather
uses `client.Get`; the marine client uses `http.NewRequest` without the supplied context.
Consequently, their upstream requests do not inherit that context's cancellation. Go documents
the client timeout semantics in [net/http.Client](https://pkg.go.dev/net/http#Client).

The YouTube adapter's search and video-status `.Do()` calls do not attach a per-call context;
passing a context when constructing the service is not an explicit deadline on those calls.
Other active adapters already have timeouts (geocoding, Wikimedia, forecast, Overpass and
destination photos), so this finding does not apply equally to every provider. The newer
WordPress and Wikidata-nearby clients default to a 30 s client timeout (`wordpress.go:54`,
`wikidatanearby.go:56`).

**Risk:** slow or stalled upstream responses can occupy requests and resources for longer
than intended, multiplying item 2's availability risk. No stalled-provider experiment was run.

**Fix plan:**

- [ ] RED: local controlled transports/providers show that cancellation reaches each active client and that slow responses terminate within a bounded budget.
- [ ] Set explicit client timeouts and use context-aware requests, including YouTube's individual API calls.
- [ ] Bound the complete lookup/planning operation as well as individual requests; sequential retries can otherwise multiply per-request timeouts. Do not assume Fiber's `ctx.Context()` supplies a request deadline or client-disconnect cancellation.
- [ ] Preserve existing graceful fallbacks when optional providers time out.

## 11. OpenWeather API key transmitted over HTTP (High)

**Evidence:** `internal/adapters/api/openweather.go` defines `openWeatherGeoLocationURL` as
`http://api.openweathermap.org/geo/1.0/direct?`. `setupQueryParams` adds `APPID`, and
`fetchCoordinates` sends that URL using the HTTP client. This is part of the active city-name
weather lookup. The subsequent weather request uses HTTPS, but the preceding geocoding
request does not. Site-level `force_https` protects incoming browser traffic, not this connection.

**Risk:** an observer or attacker on the outbound network path could obtain the provider key
or tamper with the geocoding response. An HTTP-to-HTTPS redirect cannot protect the initial
request. This requires a network-path position; no interception or credential theft was performed.

**Fix plan:**

- [ ] RED: a fake transport verifies that geocoding requests use HTTPS. Use a local TLS fixture to check rejection of untrusted certificates, and reject redirects that downgrade to HTTP.
- [ ] Use the provider's HTTPS endpoint. Validate compatibility without putting a real key in test output, command history or logs.
- [ ] Review transport-error logging: HTTP errors can include the request URL and its key. Redact credentials before errors reach application logs, with a regression test using a dummy key.
- [ ] Coordinate rotation of the OpenWeather key after deploying the transport fix if the HTTP flow has been used in production; this review does not change credentials.

---

## 12. Analytics visitor ID is derived from the client IP (Medium, privacy)

**Owner decision (2026-10-11): stop using the IP for analytics.**

**Evidence:** `internal/adapters/httpserver/analytics.go` reads `Fly-Client-IP` (else `ctx.IP()`)
and stores `visitorID(ip, userAgent)`: the first 8 bytes of `sha256(ip + "|" + userAgent)`, with
**no secret salt**. `internal/adapters/sqlite/analytics.go` keeps it in `visits.visitor` and
`/stats` uses it for `COUNT(DISTINCT visitor)` (`unique_visitors`).

**Risk:**

- **Reversible.** The IPv4 space is about 4 billion addresses. With the database and a visitor's
  user agent, anyone can recover the IP by hashing every address. The IDs are pseudonymous
  personal data, not anonymous.
- **Kept with no retention limit** (see item 8), so old rows stay linkable to IPs indefinitely.

**Fix plan:**

- [x] **Decide the replacement** (owner): **A, drop per-visitor counting**, decided 2026-10-11.
  - **A. Drop per-visitor counting (recommended):** store no visitor identity at all; `/stats`
    reports visits, searches and top cities only. No IP, no cookie, no consent banner.
  - **B. Random first-party ID** in a cookie or storage: counts returning visitors, but it is an
    analytics identifier stored on the device, so it needs a consent banner in the EU.
  - **C. Hash with a secret salt that rotates daily** and is never stored: daily uniques only.
    This still processes the IP, so it does not meet the decision.
- [ ] RED: a test proves `SaveVisit` receives no value derived from the IP: two requests with
  different `Fly-Client-IP` and the same user agent store identical rows, apart from time and path.
- [ ] Remove `visitorID` and the IP read from the analytics middleware. Keep the IP read only
  where rate limiting needs it (item 2), in memory and never stored.
- [ ] **Scrub existing data (owner decision 2026-10-11: drop the column by migration):**
  - On startup the app drops `visits.visitor` if it exists, then runs `VACUUM` so the old
    bytes are not left in free pages. It is idempotent and ships in the same change as the code.
  - RED: a test opens a database with the old schema and rows, runs the migration, and asserts
    the column is gone and visits, paths and cities are kept. A second run is a no-op.
  - Release: back up first (`fly ssh sftp get /data/weatherservice.db`), deploy, confirm the
    schema, then **delete the backup**. Fly's daily volume snapshots keep the old hashes until
    they expire (5 days by default).
  - Local copies (`weatherservice.db`, `launch-check.db`) are migrated the same way on the next run.
- [ ] Update `/stats` and its consumers: remove `unique_visitors` and the tests that assert it.
- [ ] Note the change and the retention rule (item 8) in `docs/maintenance.md`.

## 13. Writers' map routes start unbounded upstream work and cache rows (Medium)

**Added 2026-10-15.** Applies only when `WRITERS_MAP` is enabled; `fly.toml` doesn't set it, so it
may be a Fly secret. Production state unverified.

**Evidence:** `GET /map/:dest.geojson` and `GET /map/:dest/:qid` (`writermap.go`) accept any city
slug that `ParseSlug` allows, plus any valid `lat`/`lon` (`mapRequest`). `destinationSoFar` asks
the base place source for that area. The SQLite cache keys it as
`wikidata:<version>:"<name>":"<country>":<lat>:<lon>:<radius>` with full-precision `%g`
coordinates (`internal/adapters/sqlite/wikidatanearby.go:25`, `basePlaceKey`). So every distinct
name or coordinate is a cache miss. That starts the 5 km ring queries against Wikidata and stores a
new row, plus partial rows while rings load. No limiter from item 2 applies, and requests skip
`trackVisit`.

**Risk:**
- **Upstream abuse.** Scripted requests with varied coordinates can drive Wikidata SPARQL traffic
  as fast as the app accepts requests, which risks provider throttling for real visitors.
- **Storage growth.** Each miss adds `source_cache` rows with no retention (item 8).
- Exploitation was not attempted. The client's 30 s timeout bounds each call, not the total.

**Fix plan:**
- [ ] RED: with a fake base place source that counts calls, show that requests for many distinct
  coordinates within one window are refused (or served cached) after an agreed budget, and that a
  coordinate a few metres away reuses the same cache entry.
- [ ] Snap request coordinates to a grid (for example 3 decimals, like the planner's source cache)
  before keying. Consider accepting only destinations the page itself rendered.
- [ ] Apply item 2's per-client and global budgets to cache misses on these routes.
- [ ] Include the writers' map keys in item 8's retention budget.

## Existing protections and limits of the evidence

These observations reduce risk but do not close an entire vulnerability class. Referenced
tests were inspected, not executed during this update.

| Area | Current evidence and limitation |
|---|---|
| SQL injection | Reviewed SQLite paths bind request-derived values through placeholders. No injection found in those paths; text searches alone are not proof for every query. |
| Overpass query injection | `overpass.go:110` formats latitude and longitude from `float64` with `strconv.FormatFloat` |
| ICS/KML slug in `Content-Disposition` | **Still open for verification.** The initial review's malformed request returned 404, which may only show that lookup failed. `internal/slug.Parse` validates the country suffix but allows other characters in the city; exporters interpolate the original slug into a quoted filename. Verify safe header construction with a successful fake plan and hostile input before calling this closed. No response-splitting exploit established. |
| Planner `days` | Bounded by `planner.MaxDays` (`trip.go:257`, `309`, `365`) |
| Template XSS | `html/template` escapes output; no `template.HTML`/`URL`/`JS` bypass found in `internal/`; no direct `innerHTML` or `eval` use found in application browser JS. HTMX does insert response HTML. `TestPresentationVideosValidateIDsAndEscapeTitles` covers video titles; this is not complete coverage of every template context. |
| Photo URLs | `heroFromPhoto` checks URL schemes, host presence and absence of URL credentials through `webURL`. `TestDestinationPhoto_ShowsAnExplicitUnavailableStateWithoutBreakingTheDestination` already includes unsafe photo and credit URLs. Extend coverage for licence links and browser rendering rather than claiming no tests exist. |
| Destination cache identity | `loadDestination` verifies name, country and coordinate proximity before reusing a city-name cache entry. Same-name cities can replace each other's entry and cause misses, but a name collision alone is not proof of poisoning. |
| Browser HTTPS | `fly.toml` sets `force_https = true`; deployed behavior not checked. Outbound HTTP is a separate issue (item 11). |
| `.env` in the image | `.dockerignore` excludes `.env` and `.env.*` |
| Static browser assets | The app serves `./public`, not all of `node_modules`. The vendor script copies five selected assets. Confirm sensitive paths and encoded traversal attempts remain inaccessible; no live filesystem-disclosure test was performed. |
| CDN map libraries (new) | `writers-map.js` loads MapLibre 5.24.0 from jsDelivr and `writers-map-leaflet.js` loads Leaflet 1.9.4 from unpkg, both with pinned versions and Subresource Integrity (`sha384`/`sha256`). A tampered CDN file is refused by the browser. The CSP in item 4 must still allow these origins. |
| Writers' post and Wikipedia links (new) | `place_panel.go.tpl` puts crawled `PostURL` and `WikipediaURL` in `href` through `html/template`, which replaces unsafe schemes such as `javascript:`. Links open with `rel="noopener noreferrer"`. Titles and blog names are escaped text. |
| Place photo URLs (new) | `validPlacePhotoURL` (`writermap.go:197`) only accepts `https` URLs without credentials on `upload.wikimedia.org` or `thumb.wikimedia.org`. Otherwise the panel and GeoJSON drop the photo. |
| WordPress crawler (new) | It fetches only hosts listed in the embedded allowlist `writerdata/sources.json` (`writermap.ParseSources`, `cmd/web/main.go:157`), through a per-host limiter, so it isn't steerable to arbitrary URLs by visitors. Response size limits weren't reviewed. |
| Dependencies and image | Current CI config includes Dependabot, govulncheck, `npm audit --audit-level=high`, Trivy and a weekly rescan. This establishes configuration, not a fresh passing scan. Trivy gates fixable High/Critical findings (`ignore-unfixed: true`); known-vulnerability scans and lockfile integrity do not establish that an update is benign. |

## Remaining verification

- [ ] **Output handling.** Exercise user/provider-controlled text and URLs in complete pages and HTMX fragments. Extend existing photo tests for unsafe licence URLs. Test CR, LF and CRLF in calendar fields: `icsEscape` escapes LF but does not explicitly normalize CR. Distinguish malformed exports from demonstrated calendar injection.
- [ ] **Provider failure handling.** Check empty/malformed responses without live upstream calls. In particular, `openmateo.go` indexes the final wave-height element without checking for an empty array; verify the reachable error path and panic containment.
- [ ] **Proxy trust and analytics.** User-agent filtering is trivially bypassed. Whether a client-IP header is spoofable depends on the ingress path; do not assume Fly-proxied requests and direct connections have identical trust. Traffic inflation also has the storage/write-load impact in item 8.
- [ ] **Static files and secrets.** Check `.env`, database, package metadata and encoded traversal paths against the actual static middleware. Inspect the final image for unintended files without exposing secret contents.
- [ ] **Feature flags in production.** `WRITERS_MAP`, `WRITERS_SYNC` and `WRITERS_AI` aren't in `fly.toml` (only `WRITERS_RESYNC_BEFORE` is). Check `fly secrets list` to know whether finding 13 and the crawler are live.
- [ ] **Deployment and repository controls.** Verify deployed UID, proxy restrictions, required PR checks/reviews, token permissions, secret access and rollback procedure. These settings cannot be established from source alone; dependency coverage is tracked in the linked plan.
- [ ] **Live follow-up.** After fixes, verify profiler denial and response headers through production ingress. Keep any load/resource tests on an isolated instance with fake providers and explicit resource limits; the local preview does not exercise production providers or proxy behavior.

Behavior fixes follow `AGENTS.md` §5 (test first), with actual results recorded in
`docs/tdd/<task>.tdd.md`. Browser changes require checks at 375 and 1440 px. Neither the
2026-10-10 update nor the 2026-10-15 re-check implements any of the proposed fixes or claims
new runtime verification.
