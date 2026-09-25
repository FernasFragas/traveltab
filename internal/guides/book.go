package guides

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"weatherservice/internal/application"
)

// reviewDateLayout is the layout of Entry.ReviewedAt.
const reviewDateLayout = "2006-01-02"

// Publishable reports why an entry may not be shown on the site, or nil when it may. An entry is
// shown only when a person reviewed it against a recorded Wikivoyage revision: the attribution
// the license requires (article title and revision) and the review date must all be present.
func (e Entry) Publishable() error {
	switch {
	case !e.Reviewed:
		return errors.New("not reviewed")
	case strings.TrimSpace(e.Intro) == "":
		return errors.New("empty intro")
	case e.WikivoyageRevision <= 0:
		return errors.New("no Wikivoyage revision")
	case !validArticleTitle(e.WikivoyageTitle):
		return errors.New("no valid Wikivoyage article title")
	}

	if _, err := time.Parse(reviewDateLayout, e.ReviewedAt); err != nil {
		return fmt.Errorf("reviewed_at %q is not a YYYY-MM-DD date", e.ReviewedAt)
	}

	return nil
}

// validArticleTitle accepts a non-blank title with no control characters and no leading or
// trailing space. It ends up in a link, where the HTTP adapter escapes it.
func validArticleTitle(title string) bool {
	if title == "" || strings.TrimSpace(title) != title {
		return false
	}

	for _, r := range title {
		if unicode.IsControl(r) {
			return false
		}
	}

	return true
}

// Book is the read side of guides/guides.json: the reviewed guides, ready to look up by city and
// country. It is immutable after construction and answers from memory only, so a request can
// consult it without any risk of generation, a model call or I/O.
type Book struct {
	published map[string]application.CityGuide
}

// NewBook keeps the publishable entries of guides, keyed by slug. Unreviewed or incomplete
// entries are left out here, once, rather than checked on every request.
func NewBook(guides map[string]Entry) *Book {
	book := &Book{published: make(map[string]application.CityGuide, len(guides))}

	for slug, entry := range guides {
		if entry.Publishable() != nil {
			continue
		}

		book.published[slug] = application.CityGuide{
			Intro:          strings.TrimSpace(entry.Intro),
			SourceTitle:    entry.WikivoyageTitle,
			SourceRevision: entry.WikivoyageRevision,
			ReviewedOn:     entry.ReviewedAt,
		}
	}

	return book
}

// ParseBook decodes the contents of a guides.json file into a Book.
func ParseBook(data []byte) (*Book, error) {
	guides := map[string]Entry{}
	if err := json.Unmarshal(data, &guides); err != nil {
		return nil, fmt.Errorf("decoding guides: %w", err)
	}

	return NewBook(guides), nil
}

// Guide implements application.CityGuideSource. A nil Book has no guides.
func (b *Book) Guide(city, country string) (application.CityGuide, bool) {
	if b == nil || strings.TrimSpace(city) == "" || strings.TrimSpace(country) == "" {
		return application.CityGuide{}, false
	}

	guide, ok := b.published[Slug(city, country)]

	return guide, ok
}

// Len is how many guides the book can show.
func (b *Book) Len() int {
	if b == nil {
		return 0
	}

	return len(b.published)
}
