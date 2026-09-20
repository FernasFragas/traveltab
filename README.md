# ✈🌤️ TravelTab

**Live:** https://traveltabbypatronfragas.com/

TravelTab is a one-page travel dashboard. Search for a destination such as `Lisbon, Portugal` and it shows you:

- **Current weather**: temperature, feels-like, humidity, wind and conditions
- **Wave height** for coastal destinations
- **A live map** centred on the city
- **Travel videos** from YouTube about what to see and do there
- **A day-by-day trip plan** that follows the forecast: museums on rainy days, viewpoints on dry ones
- **Where to stay** for that plan: places to stay near the middle of your days, with a booking search link

When no city is given, the page opens on Lisbon, Portugal.

This started as a RESTful JSON weather API, a personal project for practising Go. It grew into a server-rendered web app that combines several third-party APIs into one page.

## Tech stack

| Layer      | Choice                                                                 |
|------------|------------------------------------------------------------------------|
| Language   | Go 1.23                                                                |
| Web        | [Fiber v2](https://gofiber.io/) with `html/template` views (`views/*.go.tpl`) |
| Frontend   | [HTMX](https://htmx.org/), Bootstrap 5, Font Awesome, flag-icons       |
| Cache      | SQLite (`mattn/go-sqlite3`) storing gzip-compressed JSON per city      |
| Hosting    | Docker on [Fly.io](https://fly.io/) (region `mad`)                     |

## Data sources

| Feature        | Provider                                                                                   | Key needed |
|----------------|--------------------------------------------------------------------------------------------|------------|
| Weather and geocoding | [OpenWeather](https://openweathermap.org/api) (Current Weather and Geocoding APIs)  | Yes        |
| Wave height    | [Open-Meteo Marine API](https://open-meteo.com/en/docs/marine-weather-api)                 | No         |
| Videos         | [YouTube Data API v3](https://developers.google.com/youtube/v3) (up to 10 results)         | Yes        |
| Map            | Waze embed iframe                                                                          | No         |
| Places to visit | [Wikipedia geosearch](https://www.mediawiki.org/wiki/API:Geosearch) + the [Wikidata entity API](https://www.wikidata.org/w/api.php): landmarks within 10 km, ranked by how many Wikipedia languages cover them | No |
| Place photos   | [Wikimedia Commons](https://commons.wikimedia.org/)                                        | No         |
| Trip forecast  | [Open-Meteo Forecast API](https://open-meteo.com/en/docs): daily rain in mm, 16 days       | No         |
| Places to stay | [OpenStreetMap](https://www.openstreetmap.org/) via the [Overpass API](https://wiki.openstreetmap.org/wiki/Overpass_API) | No |
| Booking hand-off | A plain Booking.com search link with your dates                                          | No         |

**Everything the trip planner uses is free and needs no key.** Only the weather, videos and map on the main page use keyed or embedded providers.

The `internal/adapters/api/` package also has unused clients for Amadeus, Geoapify, Makcorps, Meteomatics and Stormglass. `cmd/web` does not wire them up.

## How it works

```
Browser ──HTMX GET /process-form/?city_name=…──▶ Fiber server
                                                   │
                        1. OpenWeather: geocode city + current weather
                        2. Open-Meteo: wave height at those coordinates
                        3. SQLite cache lookup by city
                             ├─ hit  → use the cached page data
                             └─ miss → YouTube videos
                                       → saved to SQLite in the background
                                                   │
Browser ◀──── HTML fragment (content_fragment) ────┘
```

- A full page load (`GET /`) renders `index.go.tpl`.
- The search form uses HTMX. When a request has the `HX-Request: true` header, the server returns only `content_fragment.go.tpl`, and HTMX swaps it into `#content-area` without reloading the page.
- Each provider is wrapped in a small generic interface (`Reporter[T]` / `ReporterProvider[T]` in `internal/application/reporters.go`). The server depends only on these interfaces, so you can swap or mock a provider without changing the handlers.

## How the planner works

Pick a start date and 1–5 days in the **Plan my trip** card, and `GET /plan` returns a new fragment:

```
PlacesNear   Wikipedia geosearch within 10 km (7 smaller searches when it caps at 500),
             then Wikidata for fame, types and photos
    │
Rank         Keep what a visitor would actually go to, using a generated list of 421 place
    │        types (indoor / outdoor / never). A short boost list rescues popular places
    │        that fame alone ranks too low, like the Oceanário.
GroupDays    Group nearby places into walking loops, aiming for 3–4 stops per day.
    │        A day may keep 2 nearby stops; too few highlights → fewer days with a note.
Schedule     Open-Meteo decides which day gets what: 5 mm of rain or more makes an indoor
    │        day. Only the next 7 days are rearranged, because rain forecasts are not
    │        reliable further out; days 8–16 show the forecast marked "less certain".
Medoid       The stop closest to all the others becomes the centre of the plan.
    │
PickStays    OpenStreetMap places to stay within 1 km of that centre, plus a Booking
             search link with your dates.
```

When something is unavailable, the page degrades instead of failing:

- **Any source fails** → the SQLite cache keeps serving what it already has (places and stays for 30 days, forecasts for 3 hours).
- **Overpass fails** → a second public server is tried, then the old cached list, and only then does the stay card disappear with a short message.
- **The forecast fails** → the plan still loads with an explanatory note. Missing rain data is labelled separately from an uncertain forecast.
- **A cached city** previously measured about 3 ms. Cold requests now fetch the forecast alongside places and run up to four Wikimedia lookups concurrently; Overpass delays still affect the first request.

### Routes

| Method | Path                  | Description                                                                 |
|--------|-----------------------|-----------------------------------------------------------------------------|
| GET    | `/`                   | Full page. Optional `city_name` query parameter; defaults to `Lisbon, Portugal`. |
| GET    | `/process-form/`      | Same handler as `/`. Returns only the HTML fragment for HTMX requests.      |
| GET    | `/plan`               | The trip plan fragment. Takes `city`, `country`, `lat`, `lon`, `start` (`YYYY-MM-DD`) and `days` (1–5). |
| GET    | `/stats`              | Visit statistics as JSON. Hidden (404) unless `STATS_TOKEN` is set and matches `?token=`. |
| GET    | `/*`                  | Static assets from `public/`.                                               |

## Project layout

```
.
├── cmd/
│   ├── web/               # Production wiring and HTTP entry point
│   ├── loadtest-server/   # Fixture-backed load-test entry point
│   ├── placetypes/        # Offline generator flags and entry point
│   └── console/           # Console placeholder
├── internal/
│   ├── application/      # Report services, shared data, trip service and storage interfaces
│   ├── planner/          # Ranking, grouping, weather scheduling, geo helpers and stays
│   │   └── testdata/      # Recorded city acceptance fixtures
│   ├── placetypes/       # Offline type classification and generation
│   ├── config/           # Environment and .env loading
│   └── adapters/
│       ├── api/          # External HTTP providers and their test fixtures
│       ├── httpserver/   # Fiber routes, HTMX rendering, sessions and request analytics
│       └── sqlite/       # SQLite store, compressed city/source caches and visit queries
├── views/                # Go HTML templates (*.go.tpl)
├── public/               # Static CSS, logo and weather backgrounds
├── Dockerfile            # Multi-stage build (CGO enabled for SQLite)
└── fly.toml              # Fly.io app configuration
```

`cmd/web` wires the adapters together. Application services and planning logic do not import
Fiber, SQLite or provider clients. The HTTP adapter receives `application.Storage`; the SQLite
adapter owns its database connection and supplies cached planner sources. See the
[package boundaries and migration map](docs/architecture.md).

## Running locally

### Prerequisites

- Go 1.23+
- A C toolchain (`gcc` / Xcode Command Line Tools), because `go-sqlite3` needs CGO
- API keys for OpenWeather and YouTube Data API v3. The trip planner needs none.

### 1. Configure environment variables

Create a `.env` file in the project root. It is git-ignored.

```dotenv
WEATHER_API_KEY=your-openweather-key
YOUTUBE_NEW=your-youtube-data-api-key
```

| Variable             | Used for                                                                 |
|----------------------|---------------------------------------------------------------------------|
| `WEATHER_API_KEY`    | OpenWeather weather and geocoding                                         |
| `YOUTUBE_NEW`        | YouTube video search                                                      |
| `OVERPASS_URLS`      | Optional. Comma-separated Overpass servers to try, in order. Falls back to the built-in public list |
| `DB_PATH`            | Optional. Where the SQLite file lives. Defaults to `weatherservice.db` in the working directory |
| `STATS_TOKEN`        | Optional. Enables `/stats` for requests with a matching `?token=`          |
| `ENV`                | Set to `production` to skip loading `.env` and read only real environment variables |

### 2. Run

```bash
go run ./cmd/web
```

Open http://localhost:8080. The SQLite cache file `weatherservice.db` is created in the working directory on first run.

### Run with Docker

```bash
docker build -t traveltab .
docker run --rm -p 8080:8080 --env-file .env -e ENV=production traveltab
```

## Deployment

The live site runs on Fly.io and is built from the `Dockerfile`:

```bash
fly secrets set WEATHER_API_KEY=… YOUTUBE_NEW=… ENV=production
fly deploy
```

`fly.toml` keeps one machine always running and mounts a volume (`weather_data_inc`) at `/data`.

## Tests

```bash
go test ./...
```

Every test runs offline: provider APIs are stubbed and the city tests replay recorded
responses from `internal/planner/testdata/cities/`. No API keys are needed. Run
`go test -race ./...` to include the race detector, or `make test`, which does both.

Opt-in extras:

| Command | What it does |
|---|---|
| `LIVE_API_TESTS=1 go test -run Live ./internal/adapters/api/` | Calls the real Wikimedia, Open-Meteo and Overpass APIs |
| `RECORD_FIXTURES=1 go test -run Cities ./internal/planner/` | Re-records the Lisbon, Tavira and Kyoto fixtures |
| `go run ./cmd/placetypes` | Regenerates `internal/planner/placetypes_gen.go` from 20 cities (~9 minutes) |

### Load testing

The [k6 guide](loadtests/README.md) covers smoke, load and stress profiles against
the real handlers with simulated providers and a temporary SQLite database.
The [recorded local run](loadtests/validation.md) passed at up to 50 concurrent
users: 22,759 requests and zero HTTP failures. This does not establish production
capacity or measure external provider performance.

## Credits

Place data from [Wikipedia](https://www.wikipedia.org/) and [Wikidata](https://www.wikidata.org/) ·
photos from [Wikimedia Commons](https://commons.wikimedia.org/), each with its own licence ·
weather from [Open-Meteo](https://open-meteo.com/) (CC BY 4.0) ·
places to stay from [OpenStreetMap](https://www.openstreetmap.org/copyright) contributors (ODbL).

## Work in progress

- **Console client:** `cmd/console` is an empty placeholder.
- **First visit to a new city:** source lookups overlap where possible, but public API latency, especially Overpass retries, still affects cold requests.

## Author

Made with ❤️ by **Fernando Fragateiro**

[GitHub](https://github.com/FernasFragas) · [LinkedIn](https://pt.linkedin.com/in/fernando-paulo-fragateiro-the1) · [Medium](https://medium.com/@patronfragas) · [Website](https://www.fernandofragateiro.com)
