package config

import (
	"log"
	"os"
	"time"
)

// FeatureFlags switches features on or off per deployment. All flags default to off.
type FeatureFlags struct {
	WritersMap  bool
	WritersSync bool
	WritersAI   bool
	// WritersResyncBefore makes the writers sync run at startup, without waiting for the monthly
	// window, when a source's last run is older (WRITERS_RESYNC_BEFORE, "2026-10-13" or RFC 3339).
	// Zero is off.
	WritersResyncBefore time.Time
}

func LoadFeatureFlags() FeatureFlags {
	loadDotEnv()

	return FeatureFlags{
		WritersMap:          envFlag("WRITERS_MAP"),
		WritersSync:         envFlag("WRITERS_SYNC"),
		WritersAI:           envFlag("WRITERS_AI"),
		WritersResyncBefore: envTime("WRITERS_RESYNC_BEFORE"),
	}
}

// envFlag is on only when the variable is exactly "1"; anything else, including unset, is off.
func envFlag(name string) bool {
	return os.Getenv(name) == "1"
}

// envTime reads a date ("2026-10-13", midnight UTC) or an RFC 3339 time. Unset is zero; a value
// that parses as neither is logged and also zero.
func envTime(name string) time.Time {
	value := os.Getenv(name)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.DateOnly, time.RFC3339} {
		if t, err := time.Parse(layout, value); err == nil {
			return t.UTC()
		}
	}
	log.Printf("%s=%q is not a date (2026-10-13) or RFC 3339 time; ignored", name, value)
	return time.Time{}
}
