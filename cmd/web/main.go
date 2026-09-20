package main

import (
	"log"
	"os"
	"weatherservice/internal/adapters/api"
	"weatherservice/internal/adapters/httpserver"
	"weatherservice/internal/adapters/sqlite"
	"weatherservice/internal/application"
	"weatherservice/internal/config"
)

const port = ":8080"

func main() {

	weatherServiceSecrets := config.LoadEnvKey()

	reporters := application.NewWeatherReporters(
		api.NewWeatherAPI(weatherServiceSecrets.OpenWeatherAPIKey),
		api.NewOpenMateoAPI(),
	)

	videoStreamReporters := application.NewVideoStreamReporters(
		api.NewYoutubeAPI(weatherServiceSecrets.YoutubeAPIKey),
	)

	// DB_PATH should point at the mounted volume in production so data survives deploys.
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "weatherservice.db"
	}

	storage, err := sqlite.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := storage.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()
	server := httpserver.NewAppServer(reporters, videoStreamReporters, storage)

	// The trip planner reads free, keyless sources. Each one goes through the SQLite cache, so a
	// city is fetched once and still plans when a source is down. The command wires adapters into application interfaces; application and planner packages stay independent of them.
	server.SetTripPlanner(application.NewTripPlanner(
		storage.NewCachedPlaceSource(api.NewWikimediaAPI(nil)),
		storage.NewCachedForecastSource(api.NewOpenMeteoForecastAPI(nil)),
		storage.NewCachedStaySource(api.NewOverpassAPI(nil, overpassURLs(weatherServiceSecrets))),
	))

	err = server.Listen(port)
	if err != nil {
		log.Fatal(err)
	}
}

// overpassURLs uses OVERPASS_URLS when it is set, and the public servers otherwise.
func overpassURLs(secrets *config.WeatherServiceKeys) []string {
	if len(secrets.OverpassURLs) > 0 {
		return secrets.OverpassURLs
	}

	return api.DefaultOverpassURLs
}
