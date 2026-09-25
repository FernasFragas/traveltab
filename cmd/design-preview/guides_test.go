package main

import (
	"html"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	guidedata "weatherservice/guides"
	"weatherservice/internal/guides"
)

// shippedGuide returns the guide the real site would show for a city, from the embedded file.
func shippedGuide(t *testing.T, city, country string) (firstParagraph, revision string) {
	t.Helper()

	book, err := guides.ParseBook(guidedata.JSON)
	if err != nil {
		t.Fatal(err)
	}

	guide, ok := book.Guide(city, country)
	if !ok {
		t.Fatalf("no reviewed guide shipped for %s, %s", city, country)
	}

	first := strings.SplitN(guide.Intro, "\n\n", 2)[0]

	return html.EscapeString(strings.Join(strings.Fields(first), " ")), "revision " + strconv.FormatInt(guide.SourceRevision, 10)
}

func TestFixtureGuidesAnswerByCityAndCountry(t *testing.T) {
	source := newFixtureGuides("default")

	for _, tc := range []struct {
		city, country string
		want          bool
	}{
		{"Lisbon", "pt", true},
		{"Porto", "PT", true},
		{"Paris", "fr", true},
		{"Paris", "us", false}, // Paris, Texas
		{"Coimbra", "pt", false},
		{"Nophoto", "pt", false},
		{"Longguide", "pt", true},
		{"Longguide", "us", false},
	} {
		if _, ok := source.Guide(tc.city, tc.country); ok != tc.want {
			t.Errorf("Guide(%q, %q) = %v, want %v", tc.city, tc.country, ok, tc.want)
		}
	}

	if got := source.calls.Load(); got != 8 {
		t.Errorf("calls = %d, want 8", got)
	}

	for _, city := range []string{"Lisbon", "Paris", "Longguide"} {
		if _, ok := newFixtureGuides("fallback").Guide(city, "pt"); ok {
			t.Errorf("the fallback scenario must have no guides, got one for %s", city)
		}
	}
}

func TestPreviewServerWiresTheGuideFixture(t *testing.T) {
	server, cleanup := newPreviewServer(testOrigin, false, "default")
	defer cleanup()

	if server == nil {
		t.Fatal("no server")
	}
}

// The Fiber server exposes no test transport, so the guide states are checked over HTTP against a
// child process, like the other route tests.
func TestPreviewGuideRoutes(t *testing.T) {
	base, client := startPreviewProcess(t, "default")

	get := func(path string) string {
		t.Helper()

		res, err := client.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}

		defer func() { _ = res.Body.Close() }()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}

		if res.StatusCode != http.StatusOK {
			t.Fatalf("%s: status=%d", path, res.StatusCode)
		}

		return string(body)
	}

	lisbon, lisbonRevision := shippedGuide(t, "Lisbon", "pt")
	paris, parisRevision := shippedGuide(t, "Paris", "fr")
	porto, _ := shippedGuide(t, "Porto", "pt")

	for _, tc := range []struct {
		name, path string
		present    []string
		absent     []string
	}{
		{"Lisbon initial page", "/", []string{`data-guide-state="reviewed"`, lisbon, lisbonRevision, "Adapted from Wikivoyage", "CC BY-SA 4.0", `href="https://en.wikivoyage.org/wiki/Lisbon"`}, []string{`data-guide-state="missing"`}},
		{"Lisbon HTMX fragment", "/process-form/?city_name=Lisbon", []string{`data-guide-state="reviewed"`, lisbon}, nil},
		{"shared Lisbon", "/trip/lisbon-pt", []string{`data-guide-state="reviewed"`, lisbon}, nil},
		{"Porto", "/process-form/?city_name=Porto%2C%20Portugal", []string{`data-guide-state="reviewed"`, porto}, []string{lisbon}},
		{"Paris, France", "/process-form/?city_name=Paris", []string{`data-guide-state="reviewed"`, paris, parisRevision}, []string{`data-guide-state="missing"`}},
		{"Paris, Texas", "/process-form/?city_name=Paris%2C%20Texas", []string{`data-guide-state="missing"`, "We don't have a reviewed guide for Paris yet."}, []string{paris, "Adapted from Wikivoyage"}},
		{"shared Paris, France", "/trip/paris-fr", []string{`data-guide-state="reviewed"`, paris}, nil},
		{"shared Paris, Texas", "/trip/paris-us", []string{`data-guide-state="missing"`}, []string{paris}},
		{"no reviewed guide", "/process-form/?city_name=Coimbra%2C%20Portugal", []string{`data-guide-state="missing"`, `href="https://en.wikivoyage.org/w/index.php?search=Coimbra"`, "Search Wikivoyage for Coimbra"}, []string{`data-guide-state="reviewed"`, lisbon}},
		{"awkward fixture content", "/process-form/?city_name=Longguide", []string{`data-guide-state="reviewed"`, "FIXTURE ONLY.", "UnbrokenUnbroken", "&lt;b&gt;markup characters&lt;/b&gt; &amp; an ampersand."}, []string{"<b>markup characters</b>"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := get(tc.path)

			for _, text := range tc.present {
				if !strings.Contains(body, text) {
					t.Errorf("missing %q", text)
				}
			}

			for _, text := range tc.absent {
				if strings.Contains(body, text) {
					t.Errorf("unexpected %q", text)
				}
			}

			// The guide never disturbs the section anchors the shortcuts and planner rely on.
			for _, id := range []string{"overview", "itinerary", "stays", "videos"} {
				if strings.Count(body, `<section id="`+id+`"`) != 1 {
					t.Errorf("section %s not present exactly once", id)
				}
			}
		})
	}
}

func TestPreviewFallbackScenarioHasNoGuides(t *testing.T) {
	base, client := startPreviewProcess(t, "fallback")

	res, err := client.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = res.Body.Close() }()

	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), `data-guide-state="missing"`) || strings.Contains(string(body), "Adapted from Wikivoyage") {
		t.Fatal("the fallback scenario must show the missing guide state for Lisbon")
	}
}
