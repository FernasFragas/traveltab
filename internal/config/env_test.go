package config

import (
	"os"
	"reflect"
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

func TestLoadEnvKey_WritersFlagsDefaultOffAndGatewayEmpty(t *testing.T) {
	t.Setenv("ENV", "production")
	for _, name := range []string{"WRITERS_MAP", "WRITERS_SYNC", "WRITERS_AI", "LLM_GATEWAY_URL", "LLM_GATEWAY_KEY"} {
		t.Setenv(name, "")
	}

	keys := LoadEnvKey()
	assert.False(t, keys.WritersMap)
	assert.False(t, keys.WritersSync)
	assert.False(t, keys.WritersAI)
	assert.Empty(t, keys.LLMGatewayURL)
	assert.Empty(t, keys.LLMGatewayKey)
}

func TestLoadEnvKey_ReadsWritersFlagsAndGateway(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("WRITERS_MAP", "1")
	t.Setenv("WRITERS_SYNC", "true")
	t.Setenv("WRITERS_AI", "on")
	t.Setenv("LLM_GATEWAY_URL", "https://gateway.example")
	t.Setenv("LLM_GATEWAY_KEY", "secret")

	keys := LoadEnvKey()
	assert.True(t, keys.WritersMap)
	assert.True(t, keys.WritersSync)
	assert.True(t, keys.WritersAI)
	assert.Equal(t, "https://gateway.example", keys.LLMGatewayURL)
	assert.Equal(t, "secret", keys.LLMGatewayKey)
}

func TestLoadEnvKey_WritersFlagsStayOffForOtherValues(t *testing.T) {
	t.Setenv("ENV", "production")
	for _, value := range []string{"0", "false", "no", "off", "enabled", " "} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("WRITERS_MAP", value)

			keys := LoadEnvKey()

			assert.False(t, keys.WritersMap)
		})
	}
}
