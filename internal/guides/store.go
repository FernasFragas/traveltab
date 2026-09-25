package guides

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"
)

// Entry is one city's row in guides/guides.json.
//
// The generator writes the first three fields, plus WikivoyageTitle. The rest record a human
// review and are added by hand after reading the intro against its source (see
// tasks/guides-review.md); the generator never sets them, so a regenerated entry is unreviewed
// until someone reviews the new text.
type Entry struct {
	Intro              string    `json:"intro"`
	WikivoyageRevision int64     `json:"wikivoyage_revision"`
	GeneratedAt        time.Time `json:"generated_at"`

	// WikivoyageTitle is the article title after redirects ("Marrakech" for the "Marrakesh"
	// request). It is what the attribution links to.
	WikivoyageTitle string `json:"wikivoyage_title,omitempty"`
	// Reviewed is true only for an intro a person checked against WikivoyageRevision. Only
	// reviewed entries are ever shown on the site.
	Reviewed bool `json:"reviewed,omitempty"`
	// ReviewedAt is the review date, YYYY-MM-DD.
	ReviewedAt string `json:"reviewed_at,omitempty"`
	// ReviewNote says what was checked and what was changed.
	ReviewNote string `json:"review_note,omitempty"`
}

// LoadGuides reads guides/guides.json, or returns an empty map when the file doesn't exist yet.
func LoadGuides(path string) (map[string]Entry, error) {
	data, err := readGuidesFile(path)
	if err != nil {
		return nil, err
	}

	guides := map[string]Entry{}
	if data == nil {
		return guides, nil
	}

	if err := json.Unmarshal(data, &guides); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", path, err)
	}

	return guides, nil
}

// WriteGuides writes guides, keyed by slug, as indented JSON so the diff a human reviews stays
// readable.
func WriteGuides(path string, guides map[string]Entry) error {
	data, err := json.MarshalIndent(guides, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding guides: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

// readGuidesFile is split out so LoadGuides' "doesn't exist yet" branch is easy to see.
func readGuidesFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	return data, nil
}
