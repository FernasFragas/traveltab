package writerdata

import (
	"bytes"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSources_StatusIsActiveBlockedOrRemoved(t *testing.T) {
	for _, s := range loadSources(t) {
		assert.Contains(t, []string{"active", "blocked", "removed"}, s.Status, s.Host)
	}
}

func TestSources_APIBaseIsTheHostsWordPressPostsEndpoint(t *testing.T) {
	for _, s := range loadSources(t) {
		assert.Equal(t, "https://"+s.Host+"/wp-json/wp/v2/posts", s.APIBase, s.Host)
	}
}

func TestSources_LanguageIsEnglishOrPortuguese(t *testing.T) {
	for _, s := range loadSources(t) {
		assert.Contains(t, []string{"en", "pt"}, s.Language, s.Host)
	}
}

func TestSources_CheckedAtIsADate(t *testing.T) {
	for _, s := range loadSources(t) {
		_, err := time.Parse(time.DateOnly, s.CheckedAt)
		assert.NoError(t, err, s.Host)
	}
}

func TestSources_HostsAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range loadSources(t) {
		assert.False(t, seen[s.Host], "duplicate host %s", s.Host)
		seen[s.Host] = true
	}
}

func TestOverrides_DecodeWithKnownFieldsOnly(t *testing.T) {
	var o overrides
	decodeStrict(t, "overrides.json", &o)
	assert.NotEmpty(t, o.Stoplist)
}

func TestOverrides_AreasHaveAPositiveRadius(t *testing.T) {
	var o overrides
	decodeStrict(t, "overrides.json", &o)
	for _, area := range o.Areas {
		assert.Positive(t, area.RadiusKM, area.Name)
	}
}

func TestOverrides_SummaryHideListHoldsWikidataQIDs(t *testing.T) {
	var o overrides
	decodeStrict(t, "overrides.json", &o)
	qid := regexp.MustCompile(`^Q[1-9][0-9]*$`)
	for _, id := range o.SummaryHideQIDs {
		assert.Regexp(t, qid, id)
	}
}

func loadSources(t *testing.T) []source {
	t.Helper()
	var sources []source
	decodeStrict(t, "sources.json", &sources)
	require.NotEmpty(t, sources)
	return sources
}

// decodeStrict fails on fields the struct does not know, so a typo in a key is caught.
func decodeStrict(t *testing.T, name string, v any) {
	t.Helper()
	data, err := Files.ReadFile(name)
	require.NoError(t, err)
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	require.NoError(t, decoder.Decode(v))
}

type source struct {
	Host      string `json:"host"`
	APIBase   string `json:"api_base"`
	Language  string `json:"language"`
	Status    string `json:"status"`
	CheckedAt string `json:"checked_at"`
}

type overrides struct {
	AliasRemovals []struct {
		PlaceName string   `json:"place_name"`
		Aliases   []string `json:"aliases"`
	} `json:"alias_removals"`
	Merges []struct {
		CanonicalName string   `json:"canonical_name"`
		Names         []string `json:"names"`
	} `json:"merges"`
	Areas []struct {
		Name     string  `json:"name"`
		Lat      float64 `json:"lat"`
		Lon      float64 `json:"lon"`
		RadiusKM float64 `json:"radius_km"`
	} `json:"areas"`
	SummaryHideQIDs []string `json:"summary_hide_qids"`
	Stoplist        []string `json:"stoplist"`
}
