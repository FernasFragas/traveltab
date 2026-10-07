package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadFeatureFlags_WritersFlagsDefaultOff(t *testing.T) {
	t.Setenv("ENV", "production")
	for _, name := range []string{"WRITERS_MAP", "WRITERS_SYNC", "WRITERS_AI"} {
		t.Setenv(name, "")
	}

	flags := LoadFeatureFlags()

	assert.Equal(t, FeatureFlags{}, flags)
}

func TestLoadFeatureFlags_ReadsWritersFlags(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("WRITERS_MAP", "1")
	t.Setenv("WRITERS_SYNC", "1")
	t.Setenv("WRITERS_AI", "1")

	flags := LoadFeatureFlags()

	assert.Equal(t, FeatureFlags{WritersMap: true, WritersSync: true, WritersAI: true}, flags)
}

func TestLoadFeatureFlags_WritersFlagsStayOffForOtherValues(t *testing.T) {
	t.Setenv("ENV", "production")
	for _, value := range []string{"true", "yes", "on", "0", "false", "enabled", " "} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("WRITERS_MAP", value)

			flags := LoadFeatureFlags()

			assert.False(t, flags.WritersMap)
		})
	}
}
