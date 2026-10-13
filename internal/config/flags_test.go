package config

import (
	"bytes"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadFeatureFlags_WritersFlagsDefaultOff(t *testing.T) {
	t.Setenv("ENV", "production")
	for _, name := range []string{"WRITERS_MAP", "WRITERS_SYNC", "WRITERS_AI", "WRITERS_RESYNC_BEFORE"} {
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

func TestLoadFeatureFlags_ReadsWritersResyncBefore(t *testing.T) {
	t.Setenv("ENV", "production")
	for value, want := range map[string]time.Time{
		"2026-10-13":           time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC),
		"2026-10-13T20:00:00Z": time.Date(2026, 10, 13, 20, 0, 0, 0, time.UTC),
	} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("WRITERS_RESYNC_BEFORE", value)

			flags := LoadFeatureFlags()

			assert.True(t, want.Equal(flags.WritersResyncBefore), "got %v", flags.WritersResyncBefore)
		})
	}
}

func TestLoadFeatureFlags_InvalidWritersResyncBeforeIsOffAndLogged(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("WRITERS_RESYNC_BEFORE", "yesterday")
	var logs bytes.Buffer
	previousOutput := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previousOutput) })

	flags := LoadFeatureFlags()

	assert.True(t, flags.WritersResyncBefore.IsZero())
	assert.Contains(t, logs.String(), "WRITERS_RESYNC_BEFORE")
}
