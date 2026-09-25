# Redesign fixture preview

Updated 2026-09-24. The preview runs the real HTTP handlers and templates with deterministic
providers. Fixture package tests, including HTTP rendering for every named data variant, and the
default HTTP smoke suite pass. Browser acceptance and external
provider verification are recorded separately in [project repair results](../project-fix-results.md).
The fixture does not establish that live maps, Wikimedia photography, YouTube playback, or
booking destinations work. Browser rendering of every named scenario and a full accessibility audit are recorded in [final-verification-2](../artifacts/redesign/final-verification-2/README.md).

## Run the preview

Run from the repository root:

```sh
go run ./cmd/design-preview -addr 127.0.0.1:8087 -map-addr 127.0.0.1:8088
```

- `-addr` selects the application listener; `-map-addr` selects the separate local map listener.
- `-slow` delays destination weather/search and planning by two seconds each. This applies to
  ordinary Lisbon/Porto requests and intentional fixture errors. A shared page or export that
  resolves weather and a plan takes about four seconds. Cancellation interrupts the delay.
- `-scenario NAME` selects a deterministic data variant for the entire process (table below).
  Restart the preview to change it. Unknown names fail startup.
- `-live-photos` replaces the fixture photo source with the real Wikimedia lookup. It is opt-in,
  not deterministic and needs internet access; see [Live photo check](#live-photo-check).

No SQLite file, `.env`, credentials, or live provider API is read (unless you pass `-live-photos`). The no-op storage returns
`sql.ErrNoRows`, so each destination request explicitly takes the fresh-data path. The map is a
local document with a dashed border and the title **FIXTURE MAP — NOT LIVE WAZE**. Its CSP permits
its inline styles and blocks other resources. Production templates retain their provider
attribution; inspect the iframe itself for the fixture marker.

Stop the process with Ctrl-C when done. Use separate unused ports if another preview is running.

## Stable destinations and routes

- **Lisbon** is the default, with country code `pt`, the local production hero, coordinates
  38.7223, -9.1393, six videos and four stays.
- **Porto** has a fixture destination photo and always has zero videos, including under video-count
  variants.
- **Coimbra** is a second video-bearing destination for delegated-handler regression checks. It has
  no destination photo, so it shows the "Photo unavailable" state.
- **Paris** (search `Paris`, or `Paris, Texas` for the namesake) and the photo-state fixtures
  described under [Destination photo fixtures](#destination-photo-fixtures).
- Search **error** to return the intentional retargeted destination-search failure.
- Plan **1, 3, or 5 days** for success. Plan **4 days** for an intentional 500; submit `days=0`
  directly to `/plan` for an intentional 400 validation response.

Use tomorrow's UTC `YYYY-MM-DD` date in the form and URLs. The fixture builds dates from the
request and remains usable as time passes. For example, substitute that date for `<tomorrow>`:

| Scenario | URL or action |
| --- | --- |
| Initial Lisbon | `http://127.0.0.1:8087/` |
| Porto / empty videos | Search `Porto, Portugal` |
| Coimbra / replacement videos, no photo | Search `Coimbra, Portugal` |
| Paris, France / Paris, Texas | Search `Paris`, `Paris, Texas`; or `/trip/paris-fr`, `/trip/paris-us` |
| Search failure | Search `error` |
| Shared three-day plan | `/trip/lisbon-pt?days=3&from=<tomorrow>` |
| Calendar download | `/trip/lisbon-pt.ics?days=1&from=<tomorrow>` |
| Map download | `/trip/lisbon-pt.kml?days=1&from=<tomorrow>` |
| Local map identity | `http://127.0.0.1:8088/` |

The generated shared/export links use the same `lisbon-pt` slug. The videos provider receives
`Turistic places in <city>, <country>` from the HTTP adapter; the fixture deliberately normalizes
this production input before matching Porto. No production provider contract is changed.

## Deterministic data variants

Every variant uses this invocation, replacing `NAME` with a table entry:

```sh
go run ./cmd/design-preview -addr 127.0.0.1:8087 -map-addr 127.0.0.1:8088 -scenario NAME
```

Plan a three-day trip after loading the page to inspect itinerary/stay variants. The initial
page is sufficient for video variants. `-slow` can be combined with any variant.

| NAME | Data and expected presentation |
| --- | --- |
| `default` | Four stays and six videos; mixed photo/no-photo stops; dry certain day 1, rainy certain day 2, uncertain non-nil forecast day 3. A five-day plan has no forecast on days 4–5. |
| `photos-missing` | Every itinerary stop lacks an image, and the photo source finds no destination photo. The Lisbon hero remains curated; search Porto or Paris to see the "Photo unavailable" state. |
| `forecast-absent` | Every day has a nil forecast, with no rain certainty. |
| `forecast-uncertain` | Every day has a non-nil forecast and rain estimate, with certainty false. |
| `stays-0` | No nearby stays, with a usable destination/date booking link. |
| `stays-1` | Exactly one known-star, website-bearing stay. |
| `stays-6` | Six stays, including no-star and no-website variants and a long label. |
| `stays-unavailable` | No stays plus an explicit temporary-unavailability message; booking remains usable. |
| `videos-0` | No videos. |
| `videos-1` | One valid video. |
| `videos-10` | Ten distinct valid-format video IDs for expansion and layout checks. |
| `fallback` | Missing stop photos and destination photos, absent forecasts, unavailable stays, zero videos, and no city guide anywhere (even Lisbon shows the "no reviewed guide yet" card). |

Default stays already include properties without stars/websites and a long name. Photo-bearing
stops use Wikimedia file names; their successful network loading is separate from missing-photo
layout. Video IDs support deterministic activation checks; they are not evidence of playable
external media. The same fixed attraction examples appear for each fixture destination.

## Destination photo fixtures

The preview injects a fixture `DestinationPhotoSource` (`cmd/design-preview/photos.go`), so the real
handlers, templates and HTMX flow render destination photos without any provider. It answers by
the *resolved* identity (name and ISO country), like the real source, and serves generated PNGs
from the local fixture origin (`-map-addr`, under `/fixture-photos/`). Nothing contacts Wikimedia.

| Search | Resolved identity | Outcome |
| --- | --- | --- |
| `Lisbon` | Lisbon, PT | Curated local photo; the source is not asked |
| `Paris` or `Paris, France` | Paris, FR | Attributed fixture photo |
| `Paris, Texas` | Paris, US | A different fixture photo, so namesakes are visibly distinct |
| `Porto` | Porto, PT | Attributed fixture photo |
| `Nophoto` | Nophoto, PT | Completed lookup with no photograph: "Photo unavailable" |
| `Photoerror` | Photoerror, PT | Provider failure: "Photo unavailable", the rest of the page works |
| `Brokenimage` | Brokenimage, PT | Valid metadata whose image URL answers 404: the image `error` event reveals "Photo unavailable" and hides the credit |
| any other city (for example `Coimbra`) | | No photograph: "Photo unavailable" |

`Tokyo` and `Tavira` carry real coordinates for the [live check](#live-photo-check) and have no
fixture photo. The `photos-missing` and `fallback` scenarios make every destination photo-free.
`fixturePhotos` counts and records its calls, and its unit tests assert that no URL points at
Wikimedia.

## City guide fixtures

The preview injects a `CityGuideSource` (`cmd/design-preview/guides.go`). It answers from the reviewed guides
embedded in the binary, through the same `guides.Book` the site uses, so screenshots show the real text and
attribution at their real length, and it reads no file and contacts nothing. It answers by the *resolved* city and
country, like the real source.

| Search | Resolved identity | Outcome |
| --- | --- | --- |
| `Lisbon` (the default page) | Lisbon, PT | Reviewed guide with attribution |
| `Porto`, `Tokyo`, `Tavira` | PT / JP / PT | Reviewed guide |
| `Paris` or `Paris, France` | Paris, FR | Reviewed guide |
| `Paris, Texas` | Paris, US | The "no reviewed guide yet" state; never Paris, France's text |
| `Coimbra`, `Nophoto` and any other city | | The "no reviewed guide yet" state with a Wikivoyage search link |
| `Longguide` | Longguide, PT | Fixture-only text for layout: a 93-character unbroken word, a long article title and markup characters shown literally |

The `fallback` scenario has no guides at all. Because the fixture guides are the shipped ones, an edit to
`guides/guides.json` changes what these pages show; the checks read the file for the expected text and revision.

## Automated fixture checks

```sh
go test ./cmd/design-preview
```

This verifies explicit cache misses, reporter query normalization, scenario counts and semantics, the city guide states (reviewed, missing, namesake, awkward content, fallback scenario),
cancellable delays, map identity/CSP, and the actual initial/shared/export HTTP routes. It also
starts a preview process for each non-default variant and checks its initial page and rendered
three-day plan over HTTP. Route tests launch and clean up their child processes on ephemeral
localhost ports.
Sandboxed environments may need permission for Go cache access and localhost listeners.

With the default preview running:

```sh
PREVIEW_URL=http://127.0.0.1:8087 \
PREVIEW_MAP_URL=http://127.0.0.1:8088 \
node tasks/redesign/checks/http-smoke.mjs
```

The smoke suite checks the real initial city/form/hero/video data, local map iframe URL and map
response separately, 1/3/5-day plans, exactly three sibling plan roots and their OOB attributes,
400/500 clearing fragments, Porto/Coimbra search, retargeted search errors, shared-page reload,
and generated ICS/KML links, MIME types, dates, event/point counts and coordinates. Expected
fixture errors are asserted as successful test cases. Run this suite with `-scenario default`.

The smoke suite also covers the city guide states above: Lisbon, Porto and Paris with their revisions, attribution links and license, Paris, Texas and Coimbra without a guide, the shared `/trip/paris-fr` and `/trip/paris-us` pages, escaping, and that `/plan` responses carry no guide card.

The smoke suite also covers the destination photo states above: Lisbon's curated photo, Paris and
Paris, Texas (initial fragment and shared `/trip/paris-fr`/`/trip/paris-us`), Porto, the
unavailable states, the broken-image URL, and that the generated images download and differ.

On 2026-09-24, `go test -count=1 ./cmd/design-preview`, `make lint`, and the default HTTP smoke
suite passed. Localhost and linter package loading required running outside the workstation
sandbox. The fixture only checks generated URLs and markup; live map, photo, video, and booking
service reachability and actual playback/import behavior remain separate acceptance checks. The
destination photo runs and their evidence are in the
[destination photo results](../destination-photos-results.md).

## Browser prerequisites

Browser checks use Node 24's built-in WebSocket and Chromium's DevTools Protocol. No frontend
package manager is required. Download the pinned HTMX script once; each check verifies its bytes
against the SRI hash in `views/index.go.tpl`:

```sh
curl -fL --max-time 30 https://unpkg.com/htmx.org@1.9.11 -o /tmp/traveltab-htmx-1.9.11.js
```

Start the preview as above and an isolated Chromium profile in another terminal. Use a dedicated
debugging port and a fresh profile, separate from your regular browser. For example:

```sh
chromium --headless --remote-debugging-port=9227 --user-data-dir=$(mktemp -d /tmp/traveltab-preview-browser-XXXX) about:blank
```

Use your installed Chromium or Brave executable if it has a different name. In a third terminal,
run the interaction and four-width acceptance checks against the default preview:

```sh
HTMX_PATH=/tmp/traveltab-htmx-1.9.11.js PREVIEW_URL=http://127.0.0.1:8087 CDP_PORT=9227 node tasks/redesign/checks/interaction-regressions.mjs
HTMX_PATH=/tmp/traveltab-htmx-1.9.11.js PREVIEW_URL=http://127.0.0.1:8087 CDP_PORT=9227 node tasks/redesign/checks/browser-smoke.mjs
```

The browser scripts attach to an `about:blank` tab. Browser smoke writes screenshots and JSON
under `tasks/artifacts/redesign/final-verification/default/`; it overwrites that scenario's
existing evidence, which is the historical September repair record, so do not re-run it casually. For the fallback run, restart the preview with `-scenario fallback` and set
`SCENARIO=fallback` on the browser-smoke command. See the [recorded result matrix](../artifacts/redesign/final-verification/README.md)
for the tested configuration and external-service limits.

## Destination photo browser checks

`destination-photos.mjs` is the offline browser acceptance for destination photos. It writes
screenshots and JSON under `tasks/artifacts/destination-photos/fixture/` (or `OUTPUT_DIR`), leaving
the September evidence alone. Use the default scenario and the isolated browser above. Run the
preview, and a second `-slow` one on other ports for the overlapping-request check, which is
optional:

```sh
go run ./cmd/design-preview -addr 127.0.0.1:8087 -map-addr 127.0.0.1:8088
go run ./cmd/design-preview -addr 127.0.0.1:8089 -map-addr 127.0.0.1:8090 -slow   # optional
HTMX_PATH=/tmp/traveltab-htmx-1.9.11.js PREVIEW_URL=http://127.0.0.1:8087 \
  SLOW_PREVIEW_URL=http://127.0.0.1:8089 CDP_PORT=9227 node tasks/redesign/checks/destination-photos.mjs
```

At 375 px and 1440 px it exercises: the initial Lisbon page; HTMX searches Lisbon → Paris → Porto →
Paris, Texas; the three unavailable states; a rapid five-search sequence observed by a
`MutationObserver` (every observation must pair a title with its own photo and credit); shared
`/trip/paris-fr` and `/trip/paris-us` pages and reload; plan → Back → Forward; and the broken
image on initial load, after an HTMX swap and after a history restore. It asserts the image
actually loaded (`complete` and `naturalWidth > 0`), the identity, exactly one hero and credit,
an unchanged hero frame, no horizontal overflow and no JavaScript exceptions. A negative control
(the old `onerror="this.hidden = true"` handler) fails it.

The existing scripts still apply: `http-smoke.mjs` and `interaction-regressions.mjs` now assert
the photo states above (Porto swaps in its fixture photo; Coimbra shows no stale photo) and the
guide states (Porto swaps in its own reviewed guide; Coimbra shows the missing state and no stale
attribution). `browser-smoke.mjs` is unchanged.

## City guide browser checks

`guides.mjs` is the offline browser acceptance for the city guide. It writes screenshots and JSON under
`tasks/artifacts/guides/` (or `OUTPUT_DIR`). Use the isolated browser above, on its own debugging port and
temporary profile (never your regular browser). Optionally start a second preview with `-scenario fallback`:

```sh
go run ./cmd/design-preview -addr 127.0.0.1:8087 -map-addr 127.0.0.1:8088
go run ./cmd/design-preview -addr 127.0.0.1:8089 -map-addr 127.0.0.1:8090 -scenario fallback   # optional
HTMX_PATH=/tmp/traveltab-htmx-1.9.11.js PREVIEW_URL=http://127.0.0.1:8087 \
  FALLBACK_PREVIEW_URL=http://127.0.0.1:8089 CDP_PORT=9227 node tasks/redesign/checks/guides.mjs
```

At 375 px and 1440 px it exercises: the initial Lisbon page; HTMX searches Lisbon → Paris → Paris, Texas → Porto →
Coimbra; a rapid five-search sequence observed by a `MutationObserver` (every observation must pair a title with
its own guide and revision); shared `/trip/paris-fr` and `/trip/paris-us` pages and reload; plan → Back → Forward;
and the awkward `Longguide` content. It asserts the reviewed text is exactly what `guides/guides.json` holds, the four
attribution links and their targets, the missing state and its 44 px link, the card's place in `#overview` after
the grid, each section ID exactly once, no horizontal overflow, WCAG text contrast of at least 4.5:1 for every guide
text colour in both states, no exceptions, and (with the second preview) the fallback scenario.

## Accessibility audit and scenario screenshots

Two scripts start their own preview processes (one per scenario, free localhost ports) and need only the isolated browser and the pinned HTMX file described above. Their evidence and a requirement matrix are in [final-verification-2](../artifacts/redesign/final-verification-2/README.md).

```sh
CDP_PORT=9227 HTMX_PATH=/tmp/traveltab-htmx-1.9.11.js node tasks/redesign/checks/scenarios.mjs
CDP_PORT=9227 HTMX_PATH=/tmp/traveltab-htmx-1.9.11.js [AXE_PATH=/scratch/axe.min.js] node tasks/redesign/checks/a11y-audit.mjs
```

- `scenarios.mjs` screenshots and **asserts** every scenario above at 375 and 1440 px: the 12 data variants (initial and planned, plus five days, expanded videos and failed third-party images for the relevant ones), the table rows (Porto, Coimbra, both Parises by search and shared URL, search failure, 500, 400, shared plan, ICS/KML responses, local map identity), the photo fixtures (Nophoto, Photoerror, Brokenimage, Tokyo, Tavira), Longguide, and the `-slow` loading states. A mismatch fails the run. Third-party images are replaced by a generated gradient and the YouTube embed is blocked. Screenshots are WebP under `final-verification-2/scenarios/`; rerunning overwrites them.
- `a11y-audit.mjs` measures contrast from pixels, focus rings, names, landmarks, keyboard use, live regions, zoom, reduced motion and target sizes, and exits non-zero on any failure. `AXE_PATH` is optional; axe-core is never a repo dependency. `NEGATIVE_CONTROL=muted|border|focus|targets` serves a deliberately broken stylesheet and must fail; `AUDIT_ONLY=contrast-quick,focus,targets` runs a subset.

## Live photo check

This is separate from the offline suite: it needs internet access, depends on Wikimedia, and is
not deterministic. Start the preview with `-live-photos` (weather, videos and the map stay
fixtures; Paris, Porto, Tokyo and Tavira have real coordinates), then:

```sh
go run ./cmd/design-preview -addr 127.0.0.1:8087 -map-addr 127.0.0.1:8088 -live-photos
HTMX_PATH=/tmp/traveltab-htmx-1.9.11.js PREVIEW_URL=http://127.0.0.1:8087 CDP_PORT=9227 \
  node tasks/redesign/checks/destination-photos-live.mjs
```

For each of Paris, Paris (Texas), Porto, Tokyo and Tavira at 1440 px and 375 px it measures the cold
search response, waits for the real image to load, and records the credit, license links and
screenshots under `tasks/artifacts/destination-photos/live/`. Provider coverage without the browser
comes from `LIVE_API_TESTS=1 go test -v -run 'DestinationPhoto.*Live' ./internal/adapters/api/ ./internal/adapters/sqlite/`.
