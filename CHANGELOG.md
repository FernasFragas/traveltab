# Changelog

All notable changes to TravelTab are recorded in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). The project has no version tags yet, so releases are grouped by date.

## [Unreleased]

### Added

- Deterministic design preview with named data variants, HTTP smoke checks, browser regression checks, and recorded four-width screenshots; see [verification results](tasks/project-fix-results.md).
- **Weather-aware trip planner.** A "Plan my trip" card under the weather takes a start date and 1–5 days, and returns a day-by-day plan of the city's famous places: museums on rainy days, viewpoints on dry ones. Places come from Wikipedia and Wikidata, ranked by how many Wikipedia languages cover them, with photos from Wikimedia Commons. Rain comes from Open-Meteo, in mm per day, and only the next 7 days are rearranged, because forecasts aren't reliable further out.
- **"Where to stay" card.** Places to stay from OpenStreetMap within 1 km of the middle of your plan, with a Booking search link carrying your dates. No prices: those need a paid API.
- `GET /plan`, which returns the plan as an HTMX fragment.
- **Source cache** (`source_cache` table): places and places to stay for 30 days, forecasts for 3 hours. A failed refresh can serve an older entry when one exists; cold requests still depend on the provider.
- `OVERPASS_URLS` to override the Overpass servers, which are tried in order.
- `cmd/placetypes`, an offline tool that generates the list of 421 Wikidata place types (indoor, outdoor or never) from 20 cities.
- **A photograph of every searched destination.** The hero beside the weather is now looked up live from Wikidata and Wikimedia Commons, keyless, for each successful search: on the initial page, in the HTMX fragment and on shared `/trip/:slug` pages. The city entity must match the resolved name, country and coordinates (within 25 km), so Paris, France and Paris, Texas get their own pictures; the photo carries its author, license and Commons link, and a failed lookup or a broken image shows an explicit "Photo unavailable" state. Lisbon keeps its verified local photo. Photo metadata is cached for 30 days per destination identity, "no photo" for one hour; errors are never cached. New `application.DestinationPhotoSource`, `api.DestinationPhotoAPI` and `Store.NewCachedDestinationPhotoSource`; see [results](tasks/destination-photos-results.md).
- Design preview: deterministic destination photo fixtures (Paris, Paris Texas, Porto, no photo, provider failure, broken image), an opt-in `-live-photos` flag, and browser checks in `tasks/redesign/checks/destination-photos.mjs` and `destination-photos-live.mjs`.
- **City guides.** The destination page shows a short reviewed intro for the 20 starter cities, adapted from Wikivoyage under CC BY-SA 4.0 with its article, revision, authors and license linked, in the overview section, on the initial page, in the HTMX fragment and on shared `/trip/:slug` pages. Every other city gets an explicit "no reviewed guide yet" card with a Wikivoyage search link. Guides are matched by the resolved city and country and read from `guides/guides.json`, compiled into the binary, so serving a page never calls Ollama or the network. New `application.CityGuideSource`, `guides.Book`, `views/guide_card.go.tpl` and `public/redesign/guide.css`; see the [review](tasks/guides-review.md) and [results](tasks/guides-results.md).
- `guides/guides.json` entries record `wikivoyage_title`, `reviewed`, `reviewed_at` and `review_note`, all optional so older files load. Only reviewed entries are published; the generator never sets `reviewed`, so a regenerated entry stays hidden until reviewed.
- Design preview: reviewed-guide fixtures (Lisbon, Porto, Paris, a namesake, a city with none, awkward content), the `fallback` scenario without guides, and browser checks in `tasks/redesign/checks/guides.mjs`.
- GitHub Actions CI workflow that runs gofmt, golangci-lint, the tests with the race detector, the build and the Docker image build on every push to `main` and every pull request.

- Accessibility audit and scenario checks: `tasks/redesign/checks/a11y-audit.mjs` measures text and icon contrast from rendered pixels, focus rings, names, landmarks, keyboard use, zoom, reduced motion and target sizes, with negative controls; `scenarios.mjs` screenshots and asserts every named preview scenario at 375 and 1440 px. Evidence and requirement matrix in [final-verification-2](tasks/artifacts/redesign/final-verification-2/README.md).

### Removed

- **The Google Places hotels feature**, along with its photo fetching. It cost money per request.
- **The Foursquare itinerary endpoint** (`POST /generate-itinerary`), its templates and its form. The new planner replaces it.
- The `PLACES_API_NEW` and `FOURSQUARE_API_KEY` environment variables.

### Changed

- Reviewed all 20 generated city intros against their Wikivoyage revisions and rewrote them: 26 wrong or misleading statements corrected (for example Mexico City's "22 million" residents, which is the urban area) and 56 unsupported or dated ones removed. See [the review](tasks/guides-review.md).
- The page credits line now names Wikivoyage for city guides.
- Redesigned the destination, planner, itinerary, stays, and video sections across desktop and mobile layouts.
- Moved application code into `internal/`, separating application services, planner logic,
  configuration, offline generation, HTTP provider clients, Fiber handlers and SQLite persistence.
- SQLite stores now own their connections and are passed to the HTTP adapter through an application
  interface; command entry points compose the dependencies.
- `/sitemap.xml` slugs now live in a `sitemap_slugs` table behind `Storage.RecordSitemapSlug` and `ListSitemapSlugs`, replacing the `__sitemap_index__` city-cache entry that concurrent writers could overwrite. On startup, an existing entry is migrated into the table and removed, so sitemap URLs are unchanged.
- Moved the `city-country` slug format shared by `/trip/:slug` and `guides/guides.json` into `internal/slug`; `guides.Slug`, `httpserver.Slug` and `httpserver.ParseSlug` are thin wrappers, and URLs and guide keys are unchanged.

- A cached city page is now used only when its country and coordinates match the destination the search resolved. Before, the cache was keyed by name alone, so a cached Paris, Texas could stand in for Paris, France; a mismatch now fetches fresh data. Compatible entries, including those written before this change, keep working.
- The destination hero no longer hides itself on an image error: it reveals the "Photo unavailable" state and hides the orphaned credit and caption.
- Every source the planner uses is free and needs no API key.
- Forecast and place fetches overlap; Wikimedia batches run with at most four workers per lookup.
- Isolated day groups can use nearby unused highlights from the top 20, keeping Kyoto’s bamboo grove with Arashiyama.

### Fixed

- Accessibility defects found by the audit: form-field borders (date, days, search) were 1.26 to 1.39:1 and now use the new `--tt-border-control` token (3.15 to 3.46:1); weather icons 2.48:1 now 3.44:1; focus rings on photo credits (2.87:1 and 2.08:1 over the dark overlay), the first itinerary day and stop credits were low-contrast or clipped and are now inset and visible; keyboard focus entering the map frame now scrolls it into view and rings the map card; focused controls no longer sit under the fixed bottom bar on phones; the header nav marks the current section with an underline, not colour alone; the "Ends" date has an accessible name; the brand link, search field and search button are at least 44px; focus stays on Generate plan after a failed plan.
- A flaky test (`TestDestinationPhoto_NamesakeCacheEntriesNeverDetermineTheDestination`): its subtests shared one city key, so a background cache save from one could turn the next one's expected cache miss into a hit. Each subtest now has its own key.
- Malformed or unreadable city-cache entries now trigger a fresh fetch instead of a broken destination or shared-trip page.
- Planner form state survives HTMX history restoration, and video activation reads the correct video ID attribute.
- Planner input rejects missing cities, non-finite coordinates and coordinates outside geographic bounds.
- Forecast outages and missing daily values have explicit messages; stops show indoor/outdoor labels.
- Planner card contrast on the light page background.
- Docker’s Go toolchain matches the target architecture, including ARM hosts; local secrets and databases stay outside the build context.

## 2026-09-19

### Added

- Visit analytics. Requests to `/` and `/process-form/` are recorded in a new `visits` SQLite table. Bots are skipped, and visitors are stored as an anonymous hash of IP and user agent, never the raw IP.
- `GET /stats` endpoint that returns unique visitors, page views, searches, a daily breakdown and the top 10 searched cities as JSON. It is only reachable with the `STATS_TOKEN` environment variable (`/stats?token=…&days=7`) and returns 404 otherwise.
- `DB_PATH` environment variable for the SQLite file location. On Fly.io it points at the mounted volume (`/data/weatherservice.db`), so data survives deploys.
- `Makefile` with `help`, `fmt`, `lint`, `test`, `build`, `run` and `clean` targets.
- Tests for analytics and environment loading.

### Changed

- Docker images now use Debian bookworm instead of bullseye.
- Errors from photo fetching and database close are now logged instead of ignored.
- README rewritten with the tech stack, data sources, routes, project layout and setup instructions.

### Fixed

- OpenWeather requests now return a clear error when the API responds with a non-200 status or when geocoding finds no location, instead of failing while decoding the response.
- Server and OpenWeather tests.

## 2026-01-05

### Removed

- Itinerary form from the page. The `POST /generate-itinerary` backend is still there for later work.

## 2025-08

### Added

- Itinerary builder: pick a city, dates and categories, and see matching places from Foursquare on a map.

### Changed

- Better search parameters for weather lookups.

### Fixed

- Database cache.

## 2025-04

### Added

- Hotel photos, fetched concurrently, and more review details on the hotels card.
- SQLite cache that stores gzip-compressed page data per city, so repeat searches skip the YouTube and Google Places calls.
- Deployment to Fly.io with Docker and a persistent volume.
- Logo, loading spinner, social links and a Waze map embed.
- Mobile layout.

### Changed

- Data from providers is now fetched concurrently.
- Hotels now come only from the Google Places API (New).
- White background and general style updates.

### Fixed

- Image display and YouTube embed bugs.

## 2025-03

### Added

- Weather page v2 with weather, map, travel videos (YouTube) and hotel information on one page.
- Hotels card.
- Wave height from Open-Meteo.

## 2025-02

### Changed

- Major refactor into a server-rendered web app with `cmd/web`, a Fiber server and Go HTML templates.
- Weather now comes from OpenWeather.

### Added

- Wave data from Meteomatics.

## 2023-03

### Added

- First version: a RESTful JSON weather API in Go with data from NOAA, an HTML view and API tests.
