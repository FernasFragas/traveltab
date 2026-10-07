package config

import "os"

// FeatureFlags switches features on or off per deployment. All flags default to off.
type FeatureFlags struct {
	WritersMap  bool
	WritersSync bool
	WritersAI   bool
}

func LoadFeatureFlags() FeatureFlags {
	loadDotEnv()

	return FeatureFlags{
		WritersMap:  envFlag("WRITERS_MAP"),
		WritersSync: envFlag("WRITERS_SYNC"),
		WritersAI:   envFlag("WRITERS_AI"),
	}
}

// envFlag is on only when the variable is exactly "1"; anything else, including unset, is off.
func envFlag(name string) bool {
	return os.Getenv(name) == "1"
}
