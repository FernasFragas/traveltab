package main

import (
	"log"
	"os"
	guidedata "weatherservice/guides"
	"weatherservice/internal/adapters/api"
	"weatherservice/internal/adapters/httpserver"
	"weatherservice/internal/adapters/sqlite"
	"weatherservice/internal/application"
	"weatherservice/internal/config"
	"weatherservice/internal/guides"
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

	// Destination photos come from Wikidata and Commons, keyless like the planner sources. The
	// cache keeps each photograph's identity, so a namesake city never receives another's picture.
	server.SetDestinationPhotoSource(storage.NewCachedDestinationPhotoSource(api.NewDestinationPhotoAPI(nil)))

	// Reviewed city guides are compiled in from guides/guides.json and answered from memory. Only
	// entries a person has checked against Wikivoyage are published, and nothing at request time
	// generates a guide or calls a model.
	book, err := guides.ParseBook(guidedata.JSON)
	if err != nil {
		log.Fatal(err)
	}

	server.SetCityGuideSource(book)

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
