package config

import (
	"bytes"
	"log"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_LoadEnvKey(t *testing.T) {
	// Production mode skips .env, so the test only sees the values set here.
	t.Setenv("ENV", "production")
	t.Setenv("WEATHER_API_KEY", "weather-key")
	t.Setenv("YOUTUBE_NEW", "youtube-key")
	t.Setenv("PLACES_API_NEW", "places-key")
	t.Setenv("GEOAPIFY", "geoapify-key")
	t.Setenv("FOURSQUARE_API_KEY", "foursquare-key")

	keys := LoadEnvKey()

	assert.Equal(t, "weather-key", keys.OpenWeatherAPIKey)
	assert.Equal(t, "youtube-key", keys.YoutubeAPIKey)
	assert.Equal(t, "geoapify-key", keys.GeoapifyAPIKey)
}

func TestLoadEnvKey_ReadsOverpassURLs(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("OVERPASS_URLS", "https://overpass.one/api/interpreter, https://overpass.two/api/interpreter")

	keys := LoadEnvKey()

	assert.Equal(t, []string{
		"https://overpass.one/api/interpreter",
		"https://overpass.two/api/interpreter",
	}, keys.OverpassURLs)
}

func TestLoadEnvKey_OverpassURLsAreEmptyWhenUnset(t *testing.T) {
	t.Setenv("ENV", "production")
	// Setenv first, so the original value is restored after the test.
	t.Setenv("OVERPASS_URLS", "https://overpass.one/api/interpreter")
	require.NoError(t, os.Unsetenv("OVERPASS_URLS"))

	keys := LoadEnvKey()

	assert.Empty(t, keys.OverpassURLs)
}

func TestLoadEnvKey_NoLongerReadsPlacesOrFoursquareKeys(t *testing.T) {
	keys := reflect.TypeOf(WeatherServiceKeys{})
	for _, field := range []string{"GooglePlacesAPIKey", "FoursquareAPIKey"} {
		_, exists := keys.FieldByName(field)
		assert.False(t, exists, "obsolete field %s remains", field)
	}
}

func TestLoadEnvKey_GatewayEmptyByDefault(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("LLM_GATEWAY_URL", "")
	t.Setenv("LLM_GATEWAY_KEY", "")

	keys := LoadEnvKey()

	assert.Empty(t, keys.LLMGatewayURL)
	assert.Empty(t, keys.LLMGatewayKey)
}

func TestLoadEnvKey_ReadsGateway(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("LLM_GATEWAY_URL", "https://gateway.example")
	t.Setenv("LLM_GATEWAY_KEY", "secret")

	keys := LoadEnvKey()

	assert.Equal(t, "https://gateway.example", keys.LLMGatewayURL)
	assert.Equal(t, "secret", keys.LLMGatewayKey)
}

func TestWeatherServiceKeys_HoldsNoFeatureFlags(t *testing.T) {
	keys := reflect.TypeOf(WeatherServiceKeys{})
	for _, field := range []string{"WritersMap", "WritersSync", "WritersAI"} {
		_, exists := keys.FieldByName(field)
		assert.False(t, exists, "feature flag %s belongs in FeatureFlags", field)
	}
}

// Changes the working directory and the log output for the whole process: never t.Parallel().
func TestLoaders_LogAMissingDotEnvOnlyOnce(t *testing.T) {
	dotEnvOnce = sync.Once{}
	t.Cleanup(func() { dotEnvOnce = sync.Once{} })
	workDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { _ = os.Chdir(workDir) })
	t.Setenv("ENV", "development")
	var logs bytes.Buffer
	previousOutput := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previousOutput) })

	LoadEnvKey()
	LoadFeatureFlags()

	assert.Equal(t, 1, strings.Count(logs.String(), "Error loading .env file"))
}
