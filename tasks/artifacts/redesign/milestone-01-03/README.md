# Tasks 01–03 implementation evidence

This is a runnable milestone, not completion of tasks 04–07 or the full reference redesign.

Run the actual HTTP handlers with deterministic data from the repository root:

```sh
go run ./tasks/artifacts/redesign/milestone-01-03
```

Open `http://127.0.0.1:8087/`. The fixture uses temporary SQLite storage, tomorrow's start date, and no credentials. Search `error` for a search failure, another city for neutral hero fallback, or choose four days for planner failure. The clearly labeled map is served locally on port 8088; it does not verify a live map provider.

`check.mjs` uses Node's built-in WebSocket and Chromium's DevTools protocol; no frontend dependencies are installed. Start a separate Chromium browser with `--headless --remote-debugging-port=9227 --user-data-dir=/tmp/traveltab-redesign-browser` and run:

```sh
node tasks/artifacts/redesign/milestone-01-03/check.mjs
```

## Verified

- `go test ./...`, JavaScript syntax check, and `git diff --check` pass.
- Real Chromium/HTMX 1.9.11 checks pass at 375×812, 768×1024, 1100×1000, and 1440×1000: no document overflow, one of every section ID, loaded local photograph, mobile form-before-map ordering, desktop equal overview columns.
- Mobile menu opens and Escape closes it; skip and section links transfer focus. Bottom navigation active state follows the current section and the footer clears the fixed bar.
- Actual plan submissions update the form plus itinerary and stays via OOB swaps. Both 400 and 500 responses clear obsolete results. Search errors preserve the last destination; a new destination replaces it.
- Unknown/long destination names and failed images retain usable city/weather layouts. Long mobile headings have a dark backing for contrast. The photo source and license remain linked.
- `results.json` records layout measurements, checked local response statuses, and zero JavaScript exceptions. The intentional 400/500/404 responses are exercised failure cases.

## Reference comparison and limits

The cream 1180px shell, compact wordmark/search, editorial headings, terracotta navigation, 58/42 hero split, and mobile photograph/weather/navigation hierarchy are implemented. The real Lisbon photograph has the requested rooftops, dome, and river, but is daytime rather than the reference's sunset image. Long city headings may grow the photo area to remain readable.

Task 04's form and itinerary redesign and task 05's stays/videos redesign remain pending: their legacy dark surfaces, form layout, non-accordion days, and existing video behavior are deliberately retained. Empty future component assets reserve the contracted loading order. The compatibility extraction moves existing results into their own sections without completing those later tasks.

The live Waze map and YouTube playback were not validated; screenshots use an explicit local map fixture and no videos. Production retains the real embed and external links. Full 200% zoom, complete back/forward journeys, accordion behavior, click-to-load video, and final whole-page visual refinement belong to tasks 04–07 and are not claimed here.

Screenshots: `initial-375.png`, `initial-768.png`, `initial-1100.png`, `initial-1440.png`, `planned-375.png`, `fallback-375.png`, `failed-photo-375.png`.
