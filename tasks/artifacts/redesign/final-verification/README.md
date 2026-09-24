# Redesign browser acceptance evidence

Tested 2026-09-24 against the working tree, using Brave Chromium via CDP, the deterministic design preview, and the exact HTMX 1.9.11 script pinned by SRI in `views/index.go.tpl`. The tested browser identified itself in [default results](default/browser-results.json) and [fallback results](fallback/browser-results.json). The preview ran without live providers or SQLite.

## Reproduce

Follow the [browser prerequisites](../../../redesign/preview.md#browser-prerequisites) to obtain pinned HTMX and start an isolated Brave or Chromium `about:blank` tab with debugging port 9237. In separate terminals, run the preview and check from the repository root. Stop the preview before starting the next scenario:

```sh
go run ./cmd/design-preview -addr 127.0.0.1:8097 -map-addr 127.0.0.1:8098
```

```sh
HTMX_PATH=/tmp/traveltab-htmx-1.9.11.js PREVIEW_URL=http://127.0.0.1:8097 CDP_PORT=9237 node tasks/redesign/checks/browser-smoke.mjs
```

Then, again in separate terminals:

```sh
go run ./cmd/design-preview -addr 127.0.0.1:8099 -map-addr 127.0.0.1:8100 -scenario fallback
```

```sh
HTMX_PATH=/tmp/traveltab-htmx-1.9.11.js PREVIEW_URL=http://127.0.0.1:8099 CDP_PORT=9237 SCENARIO=fallback node tasks/redesign/checks/browser-smoke.mjs
```

The script refuses unpinned HTMX, waits for page readiness and the local lazy map iframe, and fails on unexpected JavaScript exceptions. It creates viewport and full-page screenshots for initial and planned states at 375×812, 768×1024, 1100×1000, and 1440×1000. The default run additionally captures the 500 plan error, 400 validation error, and Porto's empty-video state.

The default JSON records one `net::ERR_INTERNET_DISCONNECTED` resource failure. This was injected deliberately for the network-error assertion; there were zero uncaught JavaScript exceptions. The fallback JSON has no failed resources.

## Result matrix

| Requirement | Evidence and result |
| --- | --- |
| Initial destination, hero, HTMX, unique section IDs and element IDs | **Pass** at all four widths in both scenarios. Lisbon title, loaded local hero, planner and pinned HTMX were asserted. |
| No page-level horizontal overflow | **Pass** at all four widths, initial and planned, both scenarios; also at mobile 200% page scale. Measured `document.documentElement.scrollWidth <= innerWidth`. |
| Mobile form before map | **Pass** at 375 px by bounding boxes. |
| Three-day planning and accordion | **Pass** at all four widths. Three days render; only day 1 starts open; another day opens independently; regeneration reopens day 1. The current itinerary heading receives focus after planning. |
| Fallback data | **Pass** at all four widths. No stop photos, absent forecasts, temporarily unavailable stays, and no videos appear with the expected text; booking remains available. |
| Shared page, links and downloads | **Pass** in the default browser run. The generated Lisbon URL reloads with three days and the selected start date. Calendar and map links return 200. Three walking links target Google Maps directions. Booking check-in equals start date and checkout is three days after it. |
| Error states | **Pass** in the default browser run. Four-day fixture planning returns the visible 500 error; invalid latitude returns the visible 400 error. Each clears stale itinerary and stays. Porto replaces Lisbon, shows no videos, and clears the plan. An intentional search error retains Porto and shows an error. |
| Slow request, network failure and visible keyboard focus | **Pass** in the default browser run. A delayed /plan request announces loading, marks the form busy and disables submit; simulated disconnection announces failure, clears busy state and unlocks submit. Native Tab moves from the day selector to Generate plan and its computed outline is solid. |
| Reduced motion and browser exceptions | **Pass** for emulated reduced-motion media query and zero uncaught JavaScript exceptions in both runs. The script did not measure every transition's actual duration. |
| Keyboard use, full history cycle, video player activation | Covered by the separate [interaction regression evidence](interaction-results.md), not by these screenshots. |
| Contrast ratios and every named scenario variant | **Unverified by this run.** The HTTP fixture and other tests cover some response shapes, but these visual browser states still need direct evidence. |
| Real map, Wikimedia stop/stay photographs, YouTube playback, booking destination | **Unverified external services.** The map shown here is expressly a local fixture. Export status and booking URL/date checks do not establish third-party rendering or successful import/reservation. |

## Reference comparison

Compared [desktop planned screenshot](default/1440-planned-full.png) with the central website in [the reference image](../../../../goalimg.png), and [mobile planned viewport](default/375-planned-viewport.png) with the phone screen. The poster margins, promotional lettering, phone hardware and staged photographic background are outside the application and excluded.

| Section | Observed comparison |
| --- | --- |
| Shell and navigation | Both have a centered light shell, compact brand/search header, four section links and warm orange accents. The implemented header uses current section names where the poster shows broader site navigation; it has no profile icon. |
| Destination | Both use a Lisbon photo and weather metrics. The implemented desktop hero is taller and uses a different image/copy and live fixture values. Mobile title is legible over the photo. |
| Overview | The two-column desktop composition and mobile form-before-map order match the target. The fixture map is visibly labeled rather than displaying the poster's street map. The form has separate start/end/day controls instead of the poster's compressed date-range control. |
| Itinerary | Both present day accordions, weather/walk context and four stops across on desktop; first day is open. The fixture's external stop imagery was not proven to load, so placeholders appear in captures. Current stop cards are taller, making the planned page denser vertically than the poster. |
| Stays and videos | Both have card grids and a video section. Fixture stays show neutral image placeholders and honest missing star/website data; target uses photographs and denser cards. Current video cards show thumbnails where available, with fallback art for a failed thumbnail. |
| Mobile | The bottom navigation, form, map, accordion and stays stack without horizontal overflow. The target phone uses a narrower art-directed crop; these are 375 px browser captures rather than a phone-device mockup. |

The full-page CDP capture can duplicate or reposition `position: fixed` controls, including the bottom navigation and offscreen skip link, after browser focus and scrolling. The script temporarily hides those two controls **only during full-page captures**, then restores their inline styles. Viewport captures leave the DOM unmodified and show their true placement. Use full-page images for section content and viewport images for fixed controls. The script asserts focus before scrolling to the top for screenshots.
