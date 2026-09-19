# ✈🌤️ TravelTab

**Live:** https://traveltabbypatronfragas.com/

TravelTab is a one-page travel dashboard. Search for a destination such as `Lisbon, Portugal` and it shows you:

- **Current weather**: temperature, feels-like, humidity, wind and conditions
- **Wave height** for coastal destinations
- **A live map** centred on the city
- **Travel videos** from YouTube about what to see and do there
- **Nearby hotels** with rating, address, photos, reviews and links (removed for now)

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
| Hotels         | [Google Places API (New)](https://developers.google.com/maps/documentation/places/web-service/op-overview): nearby search within 9 km, plus photos | Yes |
| Map            | Waze embed iframe                                                                          | No         |

The `api/` package also has clients for Amadeus, Geoapify, Foursquare, Makcorps, Meteomatics and Stormglass. `cmd/web` does not use them, apart from Foursquare, which powers the itinerary endpoint (see [Work in progress](#work-in-progress)).

## How it works

```
Browser ──HTMX GET /process-form/?city_name=…──▶ Fiber server
                                                   │
                        1. OpenWeather: geocode city + current weather
                        2. Open-Meteo: wave height at those coordinates
                        3. SQLite cache lookup by city
                             ├─ hit  → use the cached page data
                             └─ miss → YouTube + Google Places (and photos)
                                       → saved to SQLite in the background
                                                   │
Browser ◀──── HTML fragment (content_fragment) ────┘
```

- A full page load (`GET /`) renders `index.go.tpl`.
- The search form uses HTMX. When a request has the `HX-Request: true` header, the server returns only `content_fragment.go.tpl`, and HTMX swaps it into `#content-area` without reloading the page.
- Each provider is wrapped in a small generic interface (`Reporter[T]` / `ReporterProvider[T]` in `server.go` and `reporters.go`). The server depends only on these interfaces, so you can swap or mock a provider without changing the handlers.
- Hotel photos are fetched concurrently with one goroutine per photo.

### Routes

| Method | Path                  | Description                                                                 |
|--------|-----------------------|-----------------------------------------------------------------------------|
| GET    | `/`                   | Full page. Optional `city_name` query parameter; defaults to `Lisbon, Portugal`. |
| GET    | `/process-form/`      | Same handler as `/`. Returns only the HTML fragment for HTMX requests.      |
| POST   | `/generate-itinerary` | Itinerary builder (work in progress, not linked from the UI).               |
| GET    | `/*`                  | Static assets from `public/`.                                               |

## Project layout

```
.
├── cmd/
│   ├── web/main.go        # Entry point: wires API clients into reporters and starts the server on :8080
│   └── console/main.go    # Empty placeholder for a CLI
├── api/                   # Third-party API clients (OpenWeather, Open-Meteo, YouTube, Google Places, …)
├── server.go              # Fiber app, routes, handlers, HTMX handling
├── reporters.go           # Domain types and reporters that combine provider data
├── database.go            # SQLite cache (gzip-compressed JSON, keyed by lower-cased city)
├── env.go                 # Loads API keys from the environment / .env
├── views/                 # Go HTML templates (*.go.tpl)
├── public/                # Static assets: CSS, logo, weather backgrounds
├── Dockerfile             # Multi-stage build (CGO enabled for SQLite)
└── fly.toml               # Fly.io app configuration
```

## Running locally

### Prerequisites

- Go 1.23+
- A C toolchain (`gcc` / Xcode Command Line Tools), because `go-sqlite3` needs CGO
- API keys for OpenWeather, YouTube Data API v3 and Google Places API (New)

### 1. Configure environment variables

Create a `.env` file in the project root. It is git-ignored.

```dotenv
WEATHER_API_KEY=your-openweather-key
YOUTUBE_NEW=your-youtube-data-api-key
PLACES_API_NEW=your-google-places-api-key

# Only needed for the work-in-progress itinerary endpoint
FOURSQUARE_API_KEY=your-foursquare-key
```

| Variable             | Used for                                                                 |
|----------------------|---------------------------------------------------------------------------|
| `WEATHER_API_KEY`    | OpenWeather weather and geocoding                                         |
| `YOUTUBE_NEW`        | YouTube video search                                                      |
| `PLACES_API_NEW`     | Google Places nearby hotel search and hotel photos                        |
| `FOURSQUARE_API_KEY` | Itinerary places search                                                   |
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
fly secrets set WEATHER_API_KEY=… YOUTUBE_NEW=… PLACES_API_NEW=… FOURSQUARE_API_KEY=… ENV=production
fly deploy
```

`fly.toml` keeps one machine always running and mounts a volume (`weather_data_inc`) at `/data`.

## Tests

```bash
go test ./...
```

The tests stub provider APIs and do not need API keys. Run `go test -race ./...`
to include the race detector.

### Load testing

The [k6 guide](loadtests/README.md) covers smoke, load and stress profiles against
the real handlers with simulated providers and a temporary SQLite database.
The [recorded local run](loadtests/validation.md) passed at up to 50 concurrent
users: 22,759 requests and zero HTTP failures. This does not establish production
capacity or measure external provider performance.

## Work in progress

- **Itinerary builder:** `POST /generate-itinerary` takes a city, start and end dates, and one or more categories. It uses Foursquare to find matching places around the city and renders them on a map (`itinerary_*.go.tpl`). The backend is still there, but the form has been removed from the page for now.
- **Console client:** `cmd/console` is an empty placeholder.

## Author

Made with ❤️ by **Fernando Fragateiro**

[GitHub](https://github.com/FernasFragas) · [LinkedIn](https://pt.linkedin.com/in/fernando-paulo-fragateiro-the1) · [Medium](https://medium.com/@patronfragas) · [Website](https://www.fernandofragateiro.com)
