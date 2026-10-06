package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type WeatherServiceKeys struct {
	OpenWeatherAPIKey string
	StormGlassAPIKey  string
	MateoMaticsAuths  MateoMaticsSecrets
	YoutubeAPIKey     string
	MarkcorpsAPIKey   string
	AmadeusID         string
	AmadeusSecret     string
	GeoapifyAPIKey    string
	OverpassURLs      []string
	WritersMap        bool
	WritersSync       bool
	WritersAI         bool
	LLMGatewayURL     string
	LLMGatewayKey     string
}

type MateoMaticsSecrets struct {
	Username string
	Password string
}

func LoadEnvKey() (weatherServiceKeys *WeatherServiceKeys) {
	weatherServiceKeys = &WeatherServiceKeys{}

	// Load .env file only in development
	if os.Getenv("ENV") != "production" {
		err := godotenv.Load()
		if err != nil {
			log.Println("Error loading .env file", err.Error())
		}
	}

	weatherServiceKeys.OpenWeatherAPIKey = os.Getenv("WEATHER_API_KEY")

	//weatherServiceKeys.StormGlassAPIKey = os.Getenv("STORMGLASS_API_KEY")

	//weatherServiceKeys.MateoMaticsAuths.Username = os.Getenv("MATEOMATICS_USERNAME")
	//	weatherServiceKeys.MateoMaticsAuths.Password = os.Getenv("MATEOMATICS_PASSWORD")

	weatherServiceKeys.YoutubeAPIKey = os.Getenv("YOUTUBE_NEW")

	//	weatherServiceKeys.MarkcorpsAPIKey = os.Getenv("MARKCORPS_API_KEY")

	//weatherServiceKeys.AmadeusID = os.Getenv("AMADEUS_ID")
	//weatherServiceKeys.AmadeusSecret = os.Getenv("AMADEUS_CLIENT_SECRET")

	weatherServiceKeys.GeoapifyAPIKey = os.Getenv("GEOAPIFY")

	// Empty unless someone points the planner at their own Overpass servers; cmd/web then
	// falls back to the public ones.
	weatherServiceKeys.OverpassURLs = splitList(os.Getenv("OVERPASS_URLS"))
	weatherServiceKeys.WritersMap = envBool("WRITERS_MAP")
	weatherServiceKeys.WritersSync = envBool("WRITERS_SYNC")
	weatherServiceKeys.WritersAI = envBool("WRITERS_AI")
	weatherServiceKeys.LLMGatewayURL = os.Getenv("LLM_GATEWAY_URL")
	weatherServiceKeys.LLMGatewayKey = os.Getenv("LLM_GATEWAY_KEY")

	return weatherServiceKeys
}

// envBool accepts the conventional true values used by deployment environments.
func envBool(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// splitList reads a comma-separated setting, dropping the spaces around each entry and any
// empty ones, so "a, b," and an unset variable both give what they look like.
func splitList(value string) []string {
	var list []string

	for _, part := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			list = append(list, trimmed)
		}
	}

	return list
}
