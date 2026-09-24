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
- GitHub Actions CI workflow that runs gofmt, golangci-lint, the tests with the race detector, the build and the Docker image build on every push to `main` and every pull request.

### Removed

- **The Google Places hotels feature**, along with its photo fetching. It cost money per request.
- **The Foursquare itinerary endpoint** (`POST /generate-itinerary`), its templates and its form. The new planner replaces it.
- The `PLACES_API_NEW` and `FOURSQUARE_API_KEY` environment variables.

### Changed

- Redesigned the destination, planner, itinerary, stays, and video sections across desktop and mobile layouts.
- Moved application code into `internal/`, separating application services, planner logic,
  configuration, offline generation, HTTP provider clients, Fiber handlers and SQLite persistence.
- SQLite stores now own their connections and are passed to the HTTP adapter through an application
  interface; command entry points compose the dependencies.

- Every source the planner uses is free and needs no API key.
- Forecast and place fetches overlap; Wikimedia batches run with at most four workers per lookup.
- Isolated day groups can use nearby unused highlights from the top 20, keeping Kyoto’s bamboo grove with Arashiyama.

### Fixed

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
