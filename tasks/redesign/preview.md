# Redesign fixture preview

Updated 2026-09-24. The preview runs the real HTTP handlers and templates with deterministic
providers. Fixture package tests, including HTTP rendering for every named data variant, and the
default HTTP smoke suite pass. Browser acceptance and external
provider verification are recorded separately in [project repair results](../project-fix-results.md).
The fixture does not establish that live maps, Wikimedia photography, YouTube playback, or
booking destinations work.

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

No SQLite file, `.env`, credentials, or live provider API is read. The no-op storage returns
`sql.ErrNoRows`, so each destination request explicitly takes the fresh-data path. The map is a
local document with a dashed border and the title **FIXTURE MAP — NOT LIVE WAZE**. Its CSP permits
its inline styles and blocks other resources. Production templates retain their provider
attribution; inspect the iframe itself for the fixture marker.

Stop the process with Ctrl-C when done. Use separate unused ports if another preview is running.

## Stable destinations and routes

- **Lisbon** is the default, with country code `pt`, the local production hero, coordinates
  38.7223, -9.1393, six videos and four stays.
- **Porto** has a neutral hero and always has zero videos, including under video-count variants.
- **Coimbra** is a second video-bearing destination for delegated-handler regression checks.
- Search **error** to return the intentional retargeted destination-search failure.
- Plan **1, 3, or 5 days** for success. Plan **4 days** for an intentional 500; submit `days=0`
  directly to `/plan` for an intentional 400 validation response.

Use tomorrow's UTC `YYYY-MM-DD` date in the form and URLs. The fixture builds dates from the
request and remains usable as time passes. For example, substitute that date for `<tomorrow>`:

| Scenario | URL or action |
| --- | --- |
| Initial Lisbon | `http://127.0.0.1:8087/` |
| Porto / empty videos | Search `Porto, Portugal` |
| Coimbra / replacement videos | Search `Coimbra, Portugal` |
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
| `photos-missing` | Every itinerary stop lacks an image. The Lisbon destination hero remains curated; search Porto to exercise a neutral destination hero as well. |
| `forecast-absent` | Every day has a nil forecast, with no rain certainty. |
| `forecast-uncertain` | Every day has a non-nil forecast and rain estimate, with certainty false. |
| `stays-0` | No nearby stays, with a usable destination/date booking link. |
| `stays-1` | Exactly one known-star, website-bearing stay. |
| `stays-6` | Six stays, including no-star and no-website variants and a long label. |
| `stays-unavailable` | No stays plus an explicit temporary-unavailability message; booking remains usable. |
| `videos-0` | No videos. |
| `videos-1` | One valid video. |
| `videos-10` | Ten distinct valid-format video IDs for expansion and layout checks. |
| `fallback` | Missing stop photos, absent forecasts, unavailable stays, zero videos. |

Default stays already include properties without stars/websites and a long name. Photo-bearing
stops use Wikimedia file names; their successful network loading is separate from missing-photo
layout. Video IDs support deterministic activation checks; they are not evidence of playable
external media. The same fixed attraction examples appear for each fixture destination.

## Automated fixture checks

```sh
go test ./cmd/design-preview
```

This verifies explicit cache misses, reporter query normalization, scenario counts and semantics,
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

On 2026-09-24, `go test -count=1 ./cmd/design-preview`, `make lint`, and the default HTTP smoke
suite passed. Localhost and linter package loading required running outside the workstation
sandbox. The fixture only checks generated URLs and markup; live map, photo, video, and booking
service reachability and actual playback/import behavior remain separate acceptance checks.

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
existing evidence. For the fallback run, restart the preview with `-scenario fallback` and set
`SCENARIO=fallback` on the browser-smoke command. See the [recorded result matrix](../artifacts/redesign/final-verification/README.md)
for the tested configuration and external-service limits.
