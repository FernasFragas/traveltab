package httpserver

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"weatherservice/internal/application"
	"weatherservice/internal/slug"
)

// guideStub is a city guide source over an in-memory map, keyed like guides.json. It records every
// lookup so tests can see which city and country a response asked about.
type guideStub struct {
	mu     sync.Mutex
	guides map[string]application.CityGuide
	asked  [][2]string
}

func newGuideStub() *guideStub {
	return &guideStub{guides: map[string]application.CityGuide{}}
}

func (g *guideStub) add(city, country string, guide application.CityGuide) *guideStub {
	g.guides[slug.Build(city, country)] = guide

	return g
}

func (g *guideStub) Guide(city, country string) (application.CityGuide, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.asked = append(g.asked, [2]string{city, country})
	guide, ok := g.guides[slug.Build(city, country)]

	return guide, ok
}

func reviewedGuide(text string) application.CityGuide {
	return application.CityGuide{
		Intro:          text,
		SourceTitle:    "Paris",
		SourceRevision: 5354829,
		ReviewedOn:     "2026-09-24",
	}
}

const parisGuideText = "Paris sits on the Seine.\n\nIt has twenty arrondissements."

func newGuideServer(t *testing.T, source application.CityGuideSource) (*Server, *MockWeatherReporter, *MockVideoStreamReporter) {
	t.Helper()

	server, weather, videos := newPhotoServer(t, &photoStub{})
	if source != nil {
		server.SetCityGuideSource(source)
	}

	return server, weather, videos
}

func TestGuideCard_ShowsTheReviewedGuideWithItsAttributionOnEveryRoute(t *testing.T) {
	for i, route := range photoRoutes {
		t.Run(route.name, func(t *testing.T) {
			city := "Paris"
			if i > 0 {
				city = fmt.Sprintf("Paris%d", i) // distinct cache keys; the guide is for "Paris" only
			}

			// The guide is filed under this route's own resolved city, so the test proves the
			// lookup used the resolved identity rather than a fixed name.
			forgetCity(t, city)
			source := newGuideStub().add(city, "FR", reviewedGuide(parisGuideText))
			server, weather, videos := newGuideServer(t, source)
			info := photoInfo(city, "FR", parisLat, parisLon)
			weather.EXPECT().GenerateReport(gomock.Any(), route.weatherQuery(city)).Return(info, nil)
			freshVideos(videos, info)

			status, body := doRequest(t, server, route.request(city))

			require.Equal(t, http.StatusOK, status)
			assert.Contains(t, body, `data-guide-state="reviewed"`)
			assert.Contains(t, body, `<h3 id="guide-title" class="guide-title">About `+city+`</h3>`)
			assert.Contains(t, body, `<p class="guide-paragraph">Paris sits on the Seine.</p>`)
			assert.Contains(t, body, `<p class="guide-paragraph">It has twenty arrondissements.</p>`)

			// Wikivoyage attribution: article, revision, authors and license, all links.
			assert.Contains(t, body, `href="https://en.wikivoyage.org/wiki/Paris"`)
			assert.Contains(t, body, `href="https://en.wikivoyage.org/w/index.php?oldid=5354829&amp;title=Paris"`)
			assert.Contains(t, body, `href="https://en.wikivoyage.org/w/index.php?action=history&amp;title=Paris"`)
			assert.Contains(t, body, `href="https://creativecommons.org/licenses/by-sa/4.0/"`)
			assert.Contains(t, body, ">CC BY-SA 4.0</a>")
			assert.Contains(t, body, "revision 5354829")
			assert.Contains(t, body, "Summarised and reworded. Checked against that revision on 2026-09-24.")
			assert.NotContains(t, body, `data-guide-state="missing"`)
			assert.Equal(t, 1, strings.Count(body, `id="guide-title"`), "one guide card, one unique id")

			require.Len(t, source.asked, 1, "one in-memory lookup per response")
			assert.Equal(t, [2]string{city, "FR"}, source.asked[0], "asked about the resolved city and country")
		})
	}
}

func TestGuideCard_SitsInTheOverviewWithoutChangingTheSectionIDs(t *testing.T) {
	forgetCity(t, "Paris")
	server, weather, videos := newGuideServer(t, newGuideStub().add("Paris", "FR", reviewedGuide(parisGuideText)))
	info := photoInfo("Paris", "FR", parisLat, parisLon)
	weather.EXPECT().GenerateReport(gomock.Any(), "Paris").Return(info, nil)
	freshVideos(videos, info)

	_, body := doRequest(t, server, photoRoutes[1].request("Paris")) // the HTMX fragment

	overview := body[strings.Index(body, `<section id="overview"`):strings.Index(body, `<section id="itinerary"`)]
	assert.Contains(t, overview, `class="guide-card"`, "the card is inside the overview section")

	for _, id := range []string{"overview", "itinerary", "stays", "videos"} {
		assert.Equal(t, 1, strings.Count(body, `<section id="`+id+`"`), "section %s", id)
	}

	assert.Equal(t, 1, strings.Count(body, `id="trip-card"`))
}

func TestGuideCard_MissingGuideIsADesignedStateThatSaysSo(t *testing.T) {
	for name, source := range map[string]application.CityGuideSource{
		"no source wired in":            nil,
		"source without this city":      newGuideStub().add("Lisbon", "PT", reviewedGuide("Lisbon text.")),
		"a namesake in another country": newGuideStub().add("Paris", "US", reviewedGuide("Paris, Texas text.")),
	} {
		for i, route := range photoRoutes {
			t.Run(name+"/"+route.name, func(t *testing.T) {
				city := fmt.Sprintf("Noguide%d", i)
				forgetCity(t, city)
				server, weather, videos := newGuideServer(t, source)
				info := photoInfo(city, "FR", parisLat, parisLon)
				weather.EXPECT().GenerateReport(gomock.Any(), route.weatherQuery(city)).Return(info, nil)
				freshVideos(videos, info)

				status, body := doRequest(t, server, route.request(city))

				require.Equal(t, http.StatusOK, status)
				assert.Contains(t, body, `data-guide-state="missing"`)
				assert.Contains(t, body, "We don't have a reviewed guide for "+city+" yet.")
				assert.Contains(t, body, `href="https://en.wikivoyage.org/w/index.php?search=`+city+`"`)
				assert.NotContains(t, body, `data-guide-state="reviewed"`)
				assert.NotContains(t, body, "Adapted from Wikivoyage")
				assert.NotContains(t, body, "Paris, Texas text.")
				assert.NotContains(t, body, "Lisbon text.")
			})
		}
	}
}

func TestGuideCard_NeverAppearsForADifferentCityOrCountry(t *testing.T) {
	// The source holds Paris, France. Searching for Paris, Texas (same name, other country) and
	// for another French city must both get the missing state, never Paris's text.
	source := newGuideStub().add("Paris", "FR", reviewedGuide(parisGuideText))

	for name, tc := range map[string]struct {
		city, country string
		lat, lon      float64
		want          string
	}{
		"Paris, France":   {"Paris", "FR", parisLat, parisLon, "reviewed"},
		"Paris, Texas":    {"Paris", "US", texasLat, texasLon, "missing"},
		"Lyon, France":    {"Lyon", "FR", 45.764, 4.8357, "missing"},
		"country by name": {"Paris", "France", parisLat, parisLon, "missing"},
	} {
		t.Run(name, func(t *testing.T) {
			forgetCity(t, tc.city)
			server, weather, videos := newGuideServer(t, source)
			info := photoInfo(tc.city, tc.country, tc.lat, tc.lon)
			weather.EXPECT().GenerateReport(gomock.Any(), tc.city).Return(info, nil)
			freshVideos(videos, info)

			_, body := doRequest(t, server, photoRoutes[0].request(tc.city))

			assert.Contains(t, body, `data-guide-state="`+tc.want+`"`)

			if tc.want == "missing" {
				assert.NotContains(t, body, "Paris sits on the Seine.")
			}
		})
	}
}

func TestGuideCard_ACachedPageStillGetsTheGuideOfTheSearchedDestination(t *testing.T) {
	// A cached "Guidecity" belongs to another country; the searched one is Guidecity, France.
	// The cache is refused (see loadDestination), and the guide is chosen from what the search
	// resolved, not from the cache row.
	cached := photoInfo("Guidecity", "US", texasLat, texasLon)
	require.NoError(t, testStore.SaveCityData("Guidecity", map[string]any{
		"GeneralInfo": cached, "Videos": application.VideosStream{{Title: "Texas film", VideoID: "texas123456"}},
	}))
	source := newGuideStub().add("Guidecity", "FR", reviewedGuide("The French one."))
	server, weather, videos := newGuideServer(t, source)
	fresh := photoInfo("Guidecity", "FR", parisLat, parisLon)
	weather.EXPECT().GenerateReport(gomock.Any(), "Guidecity").Return(fresh, nil)
	freshVideos(videos, fresh)

	_, body := doRequest(t, server, photoRoutes[0].request("Guidecity"))

	assert.Contains(t, body, "The French one.")
	assert.NotContains(t, body, "Texas film")
}

func TestGuideCard_EscapesEverythingItRenders(t *testing.T) {
	forgetCity(t, "Escapetown")
	hostile := application.CityGuide{
		Intro:          "<script>alert('intro')</script> Fish & chips.\n\n<img src=x onerror=alert(1)>",
		SourceTitle:    `Bad "title" <b>&`,
		SourceRevision: 42,
		ReviewedOn:     "<i>2026-09-24</i>",
	}
	server, weather, videos := newGuideServer(t, newGuideStub().add("Escapetown", "FR", hostile))
	info := photoInfo("Escapetown", "FR", parisLat, parisLon)
	weather.EXPECT().GenerateReport(gomock.Any(), "Escapetown").Return(info, nil)
	freshVideos(videos, info)

	_, body := doRequest(t, server, photoRoutes[0].request("Escapetown"))

	card := body[strings.Index(body, `<article class="guide-card"`):]
	card = card[:strings.Index(card, "</article>")]

	assert.NotContains(t, card, "<script>")
	assert.NotContains(t, card, "<img")
	assert.NotContains(t, card, "<b>")
	assert.NotContains(t, card, "<i>2026")
	assert.Contains(t, card, "&lt;script&gt;alert(&#39;intro&#39;)&lt;/script&gt; Fish &amp; chips.")
	assert.Contains(t, card, "&lt;img src=x onerror=alert(1)&gt;")
	// The title is data in a link and in text, never markup.
	assert.Contains(t, card, "%22title%22", "the title is percent-encoded inside links")
	assert.Contains(t, card, "%3Cb%3E")
	assert.Contains(t, card, "Bad &#34;title&#34; &lt;b&gt;&amp;")
}

func TestGuideCard_AGuideWithoutAttributionIsNotShown(t *testing.T) {
	for name, guide := range map[string]application.CityGuide{
		"no title":    {Intro: "Some text.", SourceRevision: 1, ReviewedOn: "2026-09-24"},
		"no revision": {Intro: "Some text.", SourceTitle: "Paris", ReviewedOn: "2026-09-24"},
		"blank intro": {Intro: " \n\n ", SourceTitle: "Paris", SourceRevision: 1, ReviewedOn: "2026-09-24"},
	} {
		t.Run(name, func(t *testing.T) {
			server := &Server{guideSource: newGuideStub().add("Paris", "FR", guide)}

			view := server.guideFor(*photoInfo("Paris", "FR", parisLat, parisLon))

			assert.False(t, view.Available)
			assert.Empty(t, view.Paragraphs)
			assert.Contains(t, view.SearchURL, "search=Paris")
		})
	}
}

func TestGuideFor_BuildsSafeWikivoyageLinks(t *testing.T) {
	server := &Server{guideSource: newGuideStub().add("Mexico City", "MX", application.CityGuide{
		Intro: "Text.", SourceTitle: "New York City", SourceRevision: 5354261, ReviewedOn: "2026-09-24",
	})}

	view := server.guideFor(*photoInfo("Mexico City", "MX", 19.43, -99.13))

	require.True(t, view.Available)
	assert.Equal(t, "https://en.wikivoyage.org/wiki/New_York_City", view.SourcePageURL)
	assert.Equal(t, "https://en.wikivoyage.org/w/index.php?oldid=5354261&title=New+York+City", view.SourceRevisionURL)
	assert.Equal(t, "https://en.wikivoyage.org/w/index.php?action=history&title=New+York+City", view.SourceHistoryURL)
	assert.Equal(t, "https://creativecommons.org/licenses/by-sa/4.0/", view.LicenseURL)
	assert.Empty(t, view.SearchURL, "the search link is for the missing state only")
}

func TestGuideParagraphs(t *testing.T) {
	assert.Equal(t, []string{"One two.", "Three."}, guideParagraphs("  One\ntwo.\r\n\r\n\n  Three. \n"))
	assert.Empty(t, guideParagraphs(" \n\n "))
}

func TestGuideCard_IsNotPartOfThePlannerResponse(t *testing.T) {
	// /plan answers with trip_card plus the two out-of-band sections; the guide is not one of them,
	// so the planner's swap contract is untouched.
	server := planServer(t, fakeTripPlanner{})
	server.SetCityGuideSource(newGuideStub().add("Lisbon", "PT", reviewedGuide("Lisbon text.")))

	status, body := doRequest(t, server, planRequest(map[string]string{"start": "2026-03-12"}))

	require.Equal(t, http.StatusOK, status)
	assert.NotContains(t, body, "guide-card")
	assert.Equal(t, 2, strings.Count(body, `hx-swap-oob="outerHTML"`))
}
