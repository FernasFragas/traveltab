package httpserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
	"weatherservice/internal/application"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// photoStub is a destination photo source that records who asked. It never touches the network.
type photoStub struct {
	mu    sync.Mutex
	calls []application.DestinationIdentity
	// answer decides the result; the default is a photograph named after the identity.
	answer func(application.DestinationIdentity) (*application.DestinationPhoto, error)
}

func (p *photoStub) Photo(_ context.Context, id application.DestinationIdentity) (*application.DestinationPhoto, error) {
	p.mu.Lock()
	p.calls = append(p.calls, id)
	p.mu.Unlock()

	if p.answer != nil {
		return p.answer(id)
	}

	return stubPhoto(id), nil
}

func (p *photoStub) callCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return len(p.calls)
}

func stubPhoto(id application.DestinationIdentity) *application.DestinationPhoto {
	name := strings.ReplaceAll(id.City, " ", "_")

	return &application.DestinationPhoto{
		URL:       "https://upload.wikimedia.org/photos/" + name + "-" + id.Country + ".jpg",
		Alt:       "View of " + id.City,
		Credit:    "Photographer of " + id.City,
		CreditURL: "https://commons.wikimedia.org/wiki/File:" + name + ".jpg",
		License:   "CC BY-SA 4.0", LicenseURL: "https://creativecommons.org/licenses/by-sa/4.0",
		Width: 1280, Height: 720, SourceID: "Q1",
	}
}

func photoURLFor(city, country string) string {
	return "https://upload.wikimedia.org/photos/" + strings.ReplaceAll(city, " ", "_") + "-" + strings.ToUpper(country) + ".jpg"
}

// photoInfo is what the weather reporters resolve a search to.
func photoInfo(city, country string, lat, lon float64) *application.GeneralWeatherInfo {
	info := createTestGeneralWeatherInfoFor(city)
	info.Country, info.Lat, info.Lon = country, lat, lon

	return info
}

const (
	parisLat, parisLon = 48.8589, 2.3200
	texasLat, texasLon = 33.6609, -95.5555
)

type photoRoute struct {
	name string
	path func(city string) string
	htmx bool
}

// photoRoutes are the three ways a destination is rendered: the full page, the HTMX search
// fragment and the shared trip page. The trip page's slug carries the country.
var photoRoutes = []photoRoute{
	{"initial page", func(city string) string { return "/?city_name=" + city }, false},
	{"htmx search", func(city string) string { return "/process-form/?city_name=" + city }, true},
	{"shared trip", func(city string) string { return "/trip/" + strings.ToLower(city) + "-fr" }, false},
}

func (r photoRoute) request(city string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, r.path(city), nil)
	if r.htmx {
		req.Header.Set("HX-Request", "true")
	}

	return req
}

// weatherQuery is what the handler asks the weather reporter for on this route.
func (r photoRoute) weatherQuery(city string) string {
	if strings.HasPrefix(r.path(city), "/trip/") {
		return strings.ToLower(city) + ", fr"
	}

	return city
}

func newPhotoServer(t *testing.T, source application.DestinationPhotoSource) (*Server, *MockWeatherReporter, *MockVideoStreamReporter) {
	t.Helper()

	server, weather, videos := createTestServer(gomock.NewController(t))
	server.now = func() time.Time { return planNow }
	server.SetTripPlanner(fakeTripPlanner{})

	if source != nil {
		server.SetDestinationPhotoSource(source)
	}

	return server, weather, videos
}

func freshVideos(videos *MockVideoStreamReporter, info *application.GeneralWeatherInfo) {
	videos.EXPECT().GenerateReport(gomock.Any(), "Turistic places in "+info.City+", "+info.Country).Return(createTestVideosStream(), nil)
}

func TestDestinationPhoto_UncachedSearchRendersItsPhotoInTheSameResponse(t *testing.T) {
	for i, route := range photoRoutes {
		t.Run(route.name, func(t *testing.T) {
			city := fmt.Sprintf("Photoville%d", i)
			forgetCity(t, city)
			source := &photoStub{}
			server, weather, videos := newPhotoServer(t, source)
			info := photoInfo(city, "fr", parisLat, parisLon)
			weather.EXPECT().GenerateReport(gomock.Any(), route.weatherQuery(city)).Return(info, nil)
			freshVideos(videos, info)

			status, body := doRequest(t, server, route.request(city))

			require.Equal(t, http.StatusOK, status)
			assert.Contains(t, body, `src="`+photoURLFor(city, "fr")+`"`)
			assert.Contains(t, body, `alt="View of `+city+`"`)
			assert.Contains(t, body, `data-photo-state="ready"`)
			assert.Contains(t, body, "Photo: Photographer of "+city)
			assert.Contains(t, body, `href="https://commons.wikimedia.org/wiki/File:`+city+`.jpg"`)
			assert.Contains(t, body, `href="https://creativecommons.org/licenses/by-sa/4.0"`)
			assert.Contains(t, body, ">CC BY-SA 4.0</a>")
			assert.NotContains(t, body, `src="/images/destinations/lisbon-alfama.jpg"`)
			require.Equal(t, 1, source.callCount(), "one lookup per response")
			assert.Equal(t, application.DestinationIdentity{City: city, Country: "FR", Lat: parisLat, Lon: parisLon}, source.calls[0])
		})
	}
}

func TestDestinationPhoto_IsResolvedEvenWhenThePageDataIsCached(t *testing.T) {
	for i, route := range photoRoutes {
		t.Run(route.name, func(t *testing.T) {
			city := fmt.Sprintf("Cachedphoto%d", i)
			info := photoInfo(city, "fr", parisLat, parisLon)
			require.NoError(t, testStore.SaveCityData(city, map[string]any{
				"GeneralInfo": info, "Videos": application.VideosStream{{Title: "Cached film", VideoID: "cached12345"}},
			}))
			source := &photoStub{}
			server, weather, _ := newPhotoServer(t, source) // no video expectation: a hit must not fetch videos
			weather.EXPECT().GenerateReport(gomock.Any(), route.weatherQuery(city)).Return(info, nil)

			status, body := doRequest(t, server, route.request(city))

			require.Equal(t, http.StatusOK, status)
			assert.Contains(t, body, "Cached film", "the cached page data is used")
			assert.Contains(t, body, `src="`+photoURLFor(city, "fr")+`"`, "the photo is still looked up for this search")
			assert.Equal(t, 1, source.callCount())
		})
	}
}

func TestDestinationPhoto_NamesakeCacheEntriesNeverDetermineTheDestination(t *testing.T) {
	texas := photoInfo("Namesake", "us", texasLat, texasLon)
	texas.Temperature = 99

	// Every subtest gets its own city key. A cache miss saves the fresh page in the background, so
	// two subtests sharing one key could see each other's writes and turn a miss into a hit.
	n := 0

	for name, cached := range map[string]*application.GeneralWeatherInfo{
		"another country":       texas,
		"same country, far":     photoInfo("Namesake", "fr", 43.30, 5.37), // Marseille, about 660 km away
		"no country":            photoInfo("Namesake", "", parisLat, parisLon),
		"no coordinates":        photoInfo("Namesake", "fr", 0, 0),
		"another name":          photoInfo("Someplace", "fr", parisLat, parisLon),
		"legacy full-name form": photoInfo("Namesake", "france", parisLat, parisLon),
	} {
		cached.Temperature = 99

		for _, route := range photoRoutes {
			n++
			city := fmt.Sprintf("Namesake%d", n)

			t.Run(name+"/"+route.name, func(t *testing.T) {
				entry := *cached
				if entry.City == "Namesake" {
					entry.City = city
				}

				forgetCity(t, city)
				require.NoError(t, testStore.SaveCityData(city, map[string]any{
					"GeneralInfo": entry, "Videos": application.VideosStream{{Title: "Texas film", VideoID: "texas123456"}},
				}))
				source := &photoStub{}
				server, weather, videos := newPhotoServer(t, source)
				fresh := photoInfo(city, "fr", parisLat, parisLon)
				fresh.Temperature = 17
				weather.EXPECT().GenerateReport(gomock.Any(), route.weatherQuery(city)).Return(fresh, nil)
				freshVideos(videos, fresh) // the mismatch is a cache miss: fresh videos are fetched

				status, body := doRequest(t, server, route.request(city))

				require.Equal(t, http.StatusOK, status)
				assert.NotContains(t, body, "Texas film", "cached namesake data is not shown")
				assert.Contains(t, body, ">17<", "the fresh weather is displayed")
				assert.NotContains(t, body, ">99<")
				assert.Contains(t, body, `src="`+photoURLFor(city, "fr")+`"`)
				require.Equal(t, 1, source.callCount())
				assert.Equal(t, "FR", source.calls[0].Country)
				assert.Equal(t, parisLat, source.calls[0].Lat)
			})
		}
	}
}

func TestSameDestination(t *testing.T) {
	base := *photoInfo("Lisbon", "pt", 38.7223, -9.1393)

	for name, tc := range map[string]struct {
		cached application.GeneralWeatherInfo
		want   bool
	}{
		"identical":              {base, true},
		"case and diacritics":    {*photoInfo("LISBON ", "PT", 38.7223, -9.1393), true},
		"geocoder noise":         {*photoInfo("Lisbon", "pt", 38.75, -9.15), true},
		"country name for code":  {*photoInfo("Lisbon", "portugal", 38.7223, -9.1393), true},
		"another country":        {*photoInfo("Lisbon", "us", 38.7223, -9.1393), false},
		"same country far away":  {*photoInfo("Lisbon", "pt", 41.15, -8.61), false},
		"just beyond the radius": {*photoInfo("Lisbon", "pt", 38.7223+0.23, -9.1393), false},
		"another name":           {*photoInfo("Porto", "pt", 38.7223, -9.1393), false},
		"missing country":        {*photoInfo("Lisbon", "", 38.7223, -9.1393), false},
		"missing coordinates":    {*photoInfo("Lisbon", "pt", 0, 0), false},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, sameDestination(tc.cached, base))
		})
	}

	assert.False(t, sameDestination(base, *photoInfo("Lisbon", "", 38.7223, -9.1393)))
	assert.False(t, sameDestination(base, *photoInfo("Lisbon", "pt", 0, 0)))
}

func TestDestinationPhoto_KeepsACompatibleLegacyCacheEntry(t *testing.T) {
	// An entry written before photos existed carries the same country and coordinates; it is used.
	info := photoInfo("Legacyville", "fr", parisLat, parisLon)
	require.NoError(t, testStore.SaveCityData("Legacyville", map[string]any{
		"GeneralInfo": info, "Videos": application.VideosStream{{Title: "Legacy film", VideoID: "legacy12345"}},
	}))
	server, weather, _ := newPhotoServer(t, &photoStub{})
	weather.EXPECT().GenerateReport(gomock.Any(), "Legacyville").Return(photoInfo("Legacyville", "fr", parisLat+0.01, parisLon), nil)

	status, body := doRequest(t, server, httptest.NewRequest(http.MethodGet, "/?city_name=Legacyville", nil))

	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, "Legacy film")
}

func TestDestinationPhoto_CuratedLisbonOverridesWithoutALookup(t *testing.T) {
	for _, route := range photoRoutes[:2] {
		t.Run(route.name, func(t *testing.T) {
			source := &photoStub{}
			server, weather, _ := newPhotoServer(t, source)
			info := photoInfo("Lisbon", "pt", 38.7223, -9.1393)
			require.NoError(t, testStore.SaveCityData("Lisbon", map[string]any{"GeneralInfo": info, "Videos": application.VideosStream{}}))
			weather.EXPECT().GenerateReport(gomock.Any(), gomock.Any()).Return(info, nil)

			status, body := doRequest(t, server, route.request("Lisbon"))

			require.Equal(t, http.StatusOK, status)
			assert.Contains(t, body, `src="/images/destinations/lisbon-alfama.jpg"`)
			assert.Contains(t, body, "Above the rooftops of Alfama")
			assert.Zero(t, source.callCount(), "the verified local photo needs no lookup")
			assert.Contains(t, body, `data-photo-state="ready"`)
		})
	}
}

func TestDestinationPhoto_ShowsAnExplicitUnavailableStateWithoutBreakingTheDestination(t *testing.T) {
	for name, tc := range map[string]struct {
		source application.DestinationPhotoSource
		info   func(city string) *application.GeneralWeatherInfo
	}{
		"no source configured": {nil, func(c string) *application.GeneralWeatherInfo { return photoInfo(c, "fr", parisLat, parisLon) }},
		"no suitable photo": {&photoStub{answer: func(application.DestinationIdentity) (*application.DestinationPhoto, error) { return nil, nil }},
			func(c string) *application.GeneralWeatherInfo { return photoInfo(c, "fr", parisLat, parisLon) }},
		"provider failure": {&photoStub{answer: func(application.DestinationIdentity) (*application.DestinationPhoto, error) {
			return nil, errors.New("wikimedia returned status 429")
		}}, func(c string) *application.GeneralWeatherInfo { return photoInfo(c, "fr", parisLat, parisLon) }},
		"unusable identity": {&photoStub{}, func(c string) *application.GeneralWeatherInfo { return photoInfo(c, "france", parisLat, parisLon) }},
		"unsafe photo URL": {&photoStub{answer: func(id application.DestinationIdentity) (*application.DestinationPhoto, error) {
			p := stubPhoto(id)
			p.URL = "javascript:alert(1)"
			return p, nil
		}}, func(c string) *application.GeneralWeatherInfo { return photoInfo(c, "fr", parisLat, parisLon) }},
		"unsafe credit link": {&photoStub{answer: func(id application.DestinationIdentity) (*application.DestinationPhoto, error) {
			p := stubPhoto(id)
			p.CreditURL = "data:text/html,x"
			return p, nil
		}}, func(c string) *application.GeneralWeatherInfo { return photoInfo(c, "fr", parisLat, parisLon) }},
		"missing attribution": {&photoStub{answer: func(id application.DestinationIdentity) (*application.DestinationPhoto, error) {
			p := stubPhoto(id)
			p.Credit = ""
			return p, nil
		}}, func(c string) *application.GeneralWeatherInfo { return photoInfo(c, "fr", parisLat, parisLon) }},
	} {
		for i, route := range photoRoutes {
			t.Run(name+"/"+route.name, func(t *testing.T) {
				city := fmt.Sprintf("Nophoto%s%d", strings.ReplaceAll(name, " ", ""), i)
				forgetCity(t, city)
				server, weather, videos := newPhotoServer(t, tc.source)
				info := tc.info(city)
				weather.EXPECT().GenerateReport(gomock.Any(), route.weatherQuery(city)).Return(info, nil)
				freshVideos(videos, info)

				status, body := doRequest(t, server, route.request(city))

				require.Equal(t, http.StatusOK, status, "a photo failure never fails the destination")
				assert.Contains(t, body, `data-photo-state="unavailable"`)
				assert.Contains(t, body, "Photo unavailable")
				assert.NotContains(t, body, `class="destination-image"`)
				assert.NotContains(t, body, "destination-photo-credit")
				assert.Contains(t, body, `<h1 class="destination-title">`+city+`</h1>`)
				assert.Contains(t, body, "weather-metrics", "weather remains usable")
				assert.Contains(t, body, `id="trip-card"`, "planning remains usable")
			})
		}
	}
}

func TestDestinationPhoto_ProviderTextIsEscaped(t *testing.T) {
	source := &photoStub{answer: func(id application.DestinationIdentity) (*application.DestinationPhoto, error) {
		p := stubPhoto(id)
		p.Credit = `<img src=x onerror=alert(1)>`
		p.License = `"><script>alert(2)</script>`
		p.Alt = `x" onerror="alert(3)`
		return p, nil
	}}
	server, weather, videos := newPhotoServer(t, source)
	city := "Escapeville"
	forgetCity(t, city)
	info := photoInfo(city, "fr", parisLat, parisLon)
	weather.EXPECT().GenerateReport(gomock.Any(), city).Return(info, nil)
	freshVideos(videos, info)

	status, body := doRequest(t, server, httptest.NewRequest(http.MethodGet, "/?city_name="+city, nil))

	require.Equal(t, http.StatusOK, status)
	assert.NotContains(t, body, "<img src=x")
	assert.NotContains(t, body, "<script>alert(2)")
	assert.NotContains(t, body, `onerror="alert(3)`)
	assert.Contains(t, body, "&lt;img src=x onerror=alert(1)&gt;")
}

func TestDestinationPhoto_EachSearchReplacesThePreviousDestinationsPhotoAndCredit(t *testing.T) {
	source := &photoStub{}
	server, weather, videos := newPhotoServer(t, source)

	// Lisbon (curated) -> Paris -> another city -> Lisbon, as consecutive HTMX searches.
	lisbon := photoInfo("Lisbon", "pt", 38.7223, -9.1393)
	require.NoError(t, testStore.SaveCityData("Lisbon", map[string]any{"GeneralInfo": lisbon, "Videos": application.VideosStream{}}))

	sequence := []struct {
		city, country string
		lat, lon      float64
		wantPhoto     string
	}{
		{"Lisbon", "pt", 38.7223, -9.1393, "/images/destinations/lisbon-alfama.jpg"},
		{"Sequenceparis", "fr", parisLat, parisLon, photoURLFor("Sequenceparis", "fr")},
		{"Sequencetokyo", "jp", 35.6895, 139.6917, photoURLFor("Sequencetokyo", "jp")},
		{"Lisbon", "pt", 38.7223, -9.1393, "/images/destinations/lisbon-alfama.jpg"},
	}

	for _, step := range sequence[1:3] {
		forgetCity(t, step.city)
	}

	for _, step := range sequence {
		info := photoInfo(step.city, step.country, step.lat, step.lon)
		weather.EXPECT().GenerateReport(gomock.Any(), step.city).Return(info, nil)

		if step.city != "Lisbon" {
			freshVideos(videos, info)
		}

		req := httptest.NewRequest(http.MethodGet, "/process-form/?city_name="+step.city, nil)
		req.Header.Set("HX-Request", "true")
		status, body := doRequest(t, server, req)

		require.Equal(t, http.StatusOK, status)
		assert.Equal(t, 1, strings.Count(body, `class="destination-image"`), "exactly one hero per fragment")
		assert.Contains(t, body, `src="`+step.wantPhoto+`"`)

		for _, other := range sequence {
			if other.wantPhoto != step.wantPhoto {
				assert.NotContains(t, body, other.wantPhoto, "%s must not carry %s's photo", step.city, other.city)
			}
		}

		if step.city != "Lisbon" {
			assert.Equal(t, 1, strings.Count(body, "destination-photo-credit"))
			assert.Contains(t, body, "Photographer of "+step.city)
		}
	}
}

func TestDestinationPhoto_RapidConcurrentSearchesKeepEachPhotoWithItsCity(t *testing.T) {
	// A slow source makes lookups overlap, so a response could receive another search's photo if
	// any state were shared.
	source := &photoStub{answer: func(id application.DestinationIdentity) (*application.DestinationPhoto, error) {
		time.Sleep(time.Duration(len(id.City)%4) * 15 * time.Millisecond)
		return stubPhoto(id), nil
	}}
	server, weather, videos := newPhotoServer(t, source)

	// Fiber fills its render key list lazily on an app's first render, which races when the first
	// renders are concurrent, so render one destination first.
	warm := photoInfo("Rapidwarm", "fr", parisLat, parisLon)
	forgetCity(t, "Rapidwarm")
	weather.EXPECT().GenerateReport(gomock.Any(), "Rapidwarm").Return(warm, nil)
	freshVideos(videos, warm)
	status, _ := doRequest(t, server, httptest.NewRequest(http.MethodGet, "/?city_name=Rapidwarm", nil))
	require.Equal(t, http.StatusOK, status)

	cities := []string{"Rapida", "Rapidbee", "Rapidcity", "Rapiddelta", "Rapide", "Rapidfox"}
	for _, city := range cities {
		forgetCity(t, city)
		info := photoInfo(city, "fr", parisLat, parisLon)
		weather.EXPECT().GenerateReport(gomock.Any(), city).Return(info, nil)
		freshVideos(videos, info)
	}

	var wg sync.WaitGroup

	results := make([]string, len(cities))

	for i, city := range cities {
		wg.Add(1)

		go func() {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodGet, "/process-form/?city_name="+city, nil)
			req.Header.Set("HX-Request", "true")

			resp, err := server.app.Test(req, 5000)
			if err != nil {
				results[i] = "error: " + err.Error()
				return
			}
			defer func() { _ = resp.Body.Close() }()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				results[i] = "error: " + err.Error()
				return
			}

			results[i] = string(body)
		}()
	}

	wg.Wait()

	for i, city := range cities {
		assert.Contains(t, results[i], `src="`+photoURLFor(city, "fr")+`"`, city)

		for _, other := range cities {
			if other != city {
				assert.NotContains(t, results[i], photoURLFor(other, "fr"), "%s must not show %s's photo", city, other)
			}
		}
	}
}

func TestDestinationPhoto_TemplateOnlyRevealsFailureStateFromTheImageEvents(t *testing.T) {
	source := &photoStub{}
	server, weather, videos := newPhotoServer(t, source)
	city := "Eventville"
	forgetCity(t, city)
	info := photoInfo(city, "fr", parisLat, parisLon)
	weather.EXPECT().GenerateReport(gomock.Any(), city).Return(info, nil)
	freshVideos(videos, info)

	_, body := doRequest(t, server, httptest.NewRequest(http.MethodGet, "/?city_name="+city, nil))

	// One image, one handler attribute per event, on the element itself: nothing to register
	// twice after an HTMX swap or a history restore.
	start := strings.Index(body, `<figure class="destination-photo"`)
	end := strings.Index(body, "</figure>")
	require.True(t, start >= 0 && end > start)
	figure := body[start:end]
	assert.Equal(t, 1, strings.Count(figure, "onerror="))
	assert.Equal(t, 1, strings.Count(figure, "onload="))
	assert.Contains(t, figure, `dataset.photoState = 'failed'`)
	assert.Equal(t, 1, strings.Count(figure, `class="destination-photo-fallback"`))
}
