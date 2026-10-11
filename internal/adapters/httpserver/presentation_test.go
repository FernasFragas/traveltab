package httpserver

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"image"
	_ "image/jpeg"
	"math"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/template/html/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"weatherservice/internal/application"
	"weatherservice/internal/planner"
)

func TestPresentationDestinationIdentity(t *testing.T) {
	for _, tc := range []struct {
		city, country string
		found         bool
	}{
		{"Lisbon", "Portugal", true}, {" lisbon ", " PT ", true}, {"LISBON", "portugal", true},
		{"Porto", "Portugal", false}, {"Lisbon", "United States", false}, {"Lisbon", "", false}, {"", "PT", false},
	} {
		t.Run(tc.city+"/"+tc.country, func(t *testing.T) {
			view := newPresentation(application.GeneralWeatherInfo{City: tc.city, Country: tc.country}, nil)
			if tc.found {
				require.NotNil(t, view.Hero)
				assert.Equal(t, "/images/destinations/lisbon-alfama.jpg", view.Hero.URL)
				assert.Equal(t, "Portugal", view.CountryLabel)
			} else {
				assert.Nil(t, view.Hero)
				assert.Empty(t, view.Tagline)
				assert.Empty(t, view.PhotoCaption)
				assert.Empty(t, view.CountryLabel)
			}
		})
	}
}

func TestPresentationManifestAssets(t *testing.T) {
	for _, destination := range curatedPresentation.Destinations {
		require.NotNil(t, destination.Hero)
		asset := destination.Hero
		assert.NotEmpty(t, asset.Source)
		assert.NotEmpty(t, asset.License)
		assert.NotEmpty(t, asset.LicenseURL)
		assert.NotEmpty(t, asset.Credit)
		assert.NotEmpty(t, asset.CreditURL)
		assert.NotEmpty(t, asset.Alt)
		assert.True(t, strings.HasPrefix(asset.URL, "/images/destinations/"))
		file, err := os.Open("public" + asset.URL)
		require.NoError(t, err)
		defer func() { _ = file.Close() }()
		config, _, err := image.DecodeConfig(file)
		require.NoError(t, err)
		assert.Equal(t, asset.Width, config.Width)
		assert.Equal(t, asset.Height, config.Height)
	}
}

func TestPresentationStayIdentity(t *testing.T) {
	req := planner.Request{City: "Lisbon", Country: "PT"}
	stay := planner.Stay{Name: "Verified Property", Kind: "hotel", Lat: 38.72, Lon: -9.13, Stars: 4, Website: "https://example.org"}
	asset := stayAsset{City: "Lisbon", Country: "Portugal", Name: stay.Name, Lat: stay.Lat, Lon: stay.Lon, Image: imageAsset{imageView: imageView{URL: "/images/stays/verified.jpg"}}}
	manifest := presentationManifest{Stays: []stayAsset{asset}}
	require.NotNil(t, manifest.stayImage(req, stay))
	wrongCity := req
	wrongCity.City = "Porto"
	assert.Nil(t, manifest.stayImage(wrongCity, stay))
	wrongCountry := req
	wrongCountry.Country = "US"
	assert.Nil(t, manifest.stayImage(wrongCountry, stay))
	wrongLocation := stay
	wrongLocation.Lat += 0.001
	assert.Nil(t, manifest.stayImage(req, wrongLocation))
	wrongName := stay
	wrongName.Name = "Another Property"
	assert.Nil(t, manifest.stayImage(req, wrongName))
	manifest.Stays = append(manifest.Stays, asset)
	assert.Nil(t, manifest.stayImage(req, stay), "ambiguous curated records cannot select a photo")
	view := newTripPlanView(&planner.Plan{Request: req, Stays: []planner.Stay{stay}})
	require.Len(t, view.StayCards, 1)
	assert.Equal(t, stay.Name, view.StayCards[0].Name)
	assert.Equal(t, stay.Kind, view.StayCards[0].Kind)
	assert.Equal(t, stay.Stars, view.StayCards[0].Stars)
	assert.Equal(t, stay.Website, view.StayCards[0].Website)
	assert.Nil(t, view.StayCards[0].Image, "no unverified production property photos")
	assert.Equal(t, []planner.Stay{stay}, view.Stays)
}

func TestPresentationEndLabel(t *testing.T) {
	server := &Server{now: func() time.Time { return planNow }}
	for _, tc := range []struct {
		start string
		days  int
		end   string
	}{
		{"2026-03-11", 1, "Wed, 11 Mar"}, {"2026-03-31", 3, "Thu, 2 Apr"},
		{"2026-12-31", 2, "Fri, 1 Jan"}, {"2028-02-28", 2, "Tue, 29 Feb"},
	} {
		start, err := time.Parse(tripDateLayout, tc.start)
		require.NoError(t, err)
		card := server.newTripCard(planner.Request{Start: start, Days: tc.days})
		assert.Equal(t, tc.end, card.EndLabel)
		assert.Equal(t, tc.start, card.Start)
		assert.Equal(t, tc.days, card.Days)
	}
}

func TestPresentationVideosValidateIDsAndEscapeTitles(t *testing.T) {
	assert.Empty(t, presentationVideos(nil))
	videos := presentationVideos(application.VideosStream{
		{Title: `<img src=x onerror=alert(1)>`, VideoID: "aB_12345-xY"},
		{VideoID: "short"}, {VideoID: "aB_12345-xY/../../"}, {VideoID: `12345"67890`}, {VideoID: "abcdefghij&"},
	})
	require.Len(t, videos, 1)
	assert.Equal(t, "https://i.ytimg.com/vi/aB_12345-xY/hqdefault.jpg", videos[0].ThumbnailURL)
	assert.Equal(t, "https://www.youtube.com/watch?v=aB_12345-xY", videos[0].WatchURL)
	tpl := template.Must(template.New("video").Parse(`<a href="{{.WatchURL}}">{{.Title}}</a>`))
	var output bytes.Buffer
	require.NoError(t, tpl.Execute(&output, videos[0]))
	assert.NotContains(t, output.String(), "<img")
	assert.Contains(t, output.String(), "&lt;img")
}

func TestPresentationSafeMapURL(t *testing.T) {
	link, err := url.Parse(cityMapURL(38.7223, -9.1393))
	require.NoError(t, err)
	assert.Equal(t, "https", link.Scheme)
	assert.Equal(t, "www.google.com", link.Host)
	assert.Equal(t, "38.722300,-9.139300", link.Query().Get("query"))
	for _, coords := range [][2]float64{{math.NaN(), 0}, {0, math.Inf(1)}, {91, 0}, {0, -181}} {
		assert.Empty(t, cityMapURL(coords[0], coords[1]))
	}
}

func TestPresentationCurrentConditionIcons(t *testing.T) {
	for condition, icon := range map[string]string{
		"thunderstorm with rain": "bi-cloud-lightning-rain", "light snow": "bi-cloud-snow",
		"drizzle": "bi-cloud-rain", "mist": "bi-cloud-fog2", "overcast clouds": "bi-cloud",
		"few clouds": "bi-cloud-sun", "clear sky": "bi-sun", "wind": "bi-wind", "unknown": "bi-thermometer-half",
	} {
		assert.Equal(t, icon, conditionIcon(condition))
	}
}

func TestLocationLabel_RegionThenCountry(t *testing.T) {
	assert.Equal(t, "Madeira, Portugal", locationLabel("Madeira", "Portugal"))
}

func TestLocationLabel_NoRegionShowsCountryOnly(t *testing.T) {
	assert.Equal(t, "Portugal", locationLabel("", "Portugal"))
	assert.Equal(t, "Portugal", locationLabel("  ", "Portugal"))
}

func TestLocationLabel_RegionSameAsCountryIsNotRepeated(t *testing.T) {
	assert.Equal(t, "Singapore", locationLabel("Singapore", "Singapore"))
}

func TestForecastRows_FourDaysWithDayIconHighLow(t *testing.T) {
	days := make([]planner.DayForecast, 6)
	for i := range days {
		max, min, code := 24.6+float64(i), 17.4, 1
		days[i] = planner.DayForecast{
			Date: time.Date(2026, 10, 11+i, 0, 0, 0, 0, time.UTC),
			MaxC: &max, MinC: &min, WeatherCode: &code,
		}
	}

	rows := forecastRows(&planner.Forecast{Days: days})

	require.Len(t, rows, 4)
	assert.Equal(t, forecastRow{Day: "Sun", Icon: "bi-cloud-sun", Condition: "Mostly clear", High: "25", Low: "17"}, rows[0])
	assert.Equal(t, "Mon", rows[1].Day)
	assert.Equal(t, "26", rows[1].High)
}

func TestForecastRows_SkipsDaysWithoutTemperatures(t *testing.T) {
	max, min, code := 20.0, 12.0, 61
	rows := forecastRows(&planner.Forecast{Days: []planner.DayForecast{
		{Date: time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC), MaxC: &max},
		{Date: time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC), MaxC: &max, MinC: &min, WeatherCode: &code},
	}})

	require.Len(t, rows, 1)
	assert.Equal(t, "Mon", rows[0].Day)
	assert.Equal(t, "bi-cloud-rain", rows[0].Icon)
}

func TestForecastRows_NoForecastMeansNoRows(t *testing.T) {
	assert.Empty(t, forecastRows(nil))
	assert.Empty(t, forecastRows(&planner.Forecast{}))
}

func TestWeatherCode_IconAndLabel(t *testing.T) {
	for code, want := range map[int][2]string{
		0: {"bi-sun", "Clear sky"}, 3: {"bi-cloud", "Overcast"}, 45: {"bi-cloud-fog2", "Fog"},
		53: {"bi-cloud-drizzle", "Drizzle"}, 65: {"bi-cloud-rain-heavy", "Heavy rain"},
		73: {"bi-cloud-snow", "Snow"}, 81: {"bi-cloud-rain", "Rain showers"}, 95: {"bi-cloud-lightning-rain", "Thunderstorm"},
		999: {"bi-thermometer-half", "Unknown"},
	} {
		assert.Equal(t, want[0], weatherCodeIcon(code), code)
		assert.Equal(t, want[1], weatherCodeLabel(code), code)
	}
}

type fakeSuggester struct {
	places []application.PlaceSuggestion
	err    error
	calls  int
}

func (f *fakeSuggester) Suggest(context.Context, string) ([]application.PlaceSuggestion, error) {
	f.calls++
	return f.places, f.err
}

type fakeForecast struct {
	forecast *planner.Forecast
	err      error
}

func (f fakeForecast) DailyForecast(context.Context, float64, float64) (*planner.Forecast, error) {
	return f.forecast, f.err
}

var funchal = application.GeneralWeatherInfo{City: "Funchal", Country: "pt", Lat: 32.65, Lon: -16.91}

func TestDestinationPresentation_ShowsRegionAndCountryFromTheGeocoder(t *testing.T) {
	s := &Server{suggestions: newSuggestionCache(), placeSuggester: &fakeSuggester{places: []application.PlaceSuggestion{
		{Name: "Funchal", Region: "Other", Country: "Portugal", CountryCode: "PT", Lat: 40, Lon: -8},
		{Name: "Funchal", Region: "Madeira", Country: "Portugal", CountryCode: "PT", Lat: 32.6669, Lon: -16.9241},
	}}}

	view := s.destinationPresentation(funchal, nil, nil)

	assert.Equal(t, "Madeira, Portugal", view.Location)
}

func TestDestinationPresentation_NoRegionShowsCountryOnly(t *testing.T) {
	s := &Server{suggestions: newSuggestionCache(), placeSuggester: &fakeSuggester{places: []application.PlaceSuggestion{
		{Name: "Funchal", Country: "Portugal", CountryCode: "PT", Lat: 32.6669, Lon: -16.9241},
	}}}

	assert.Equal(t, "Portugal", s.destinationPresentation(funchal, nil, nil).Location)
}

func TestDestinationPresentation_GeocoderDownFallsBackToTheCountryCode(t *testing.T) {
	s := &Server{suggestions: newSuggestionCache(), placeSuggester: &fakeSuggester{err: errors.New("down")}}

	assert.Equal(t, "PT", s.destinationPresentation(funchal, nil, nil).Location)
}

func TestDestinationPresentation_NoGeocoderKeepsTheCuratedCountry(t *testing.T) {
	s := &Server{suggestions: newSuggestionCache()}
	lisbon := application.GeneralWeatherInfo{City: "Lisbon", Country: "pt", Lat: 38.72, Lon: -9.14}

	assert.Equal(t, "Portugal", s.destinationPresentation(lisbon, nil, nil).Location)
}

func TestDestinationPresentation_ReusesTheSuggestionCache(t *testing.T) {
	suggester := &fakeSuggester{places: []application.PlaceSuggestion{{Name: "Funchal", Region: "Madeira", Country: "Portugal", CountryCode: "PT", Lat: 32.66, Lon: -16.92}}}
	s := &Server{suggestions: newSuggestionCache(), placeSuggester: suggester}

	s.destinationPresentation(funchal, nil, nil)
	s.destinationPresentation(funchal, nil, nil)

	assert.Equal(t, 1, suggester.calls)
}

func TestDestinationPresentation_ForecastRowsWhenAvailable(t *testing.T) {
	max, min, code := 22.0, 16.0, 0
	s := &Server{suggestions: newSuggestionCache(), forecastSource: fakeForecast{forecast: &planner.Forecast{Days: []planner.DayForecast{
		{Date: time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC), MaxC: &max, MinC: &min, WeatherCode: &code},
	}}}}

	view := s.destinationPresentation(funchal, nil, nil)

	require.Len(t, view.Forecast, 1)
	assert.Equal(t, "22", view.Forecast[0].High)
}

func TestDestinationPresentation_ForecastDownMeansNoRows(t *testing.T) {
	s := &Server{suggestions: newSuggestionCache(), forecastSource: fakeForecast{err: errors.New("down")}}

	assert.Empty(t, s.destinationPresentation(funchal, nil, nil).Forecast)
}

func TestWindKmh_ConvertsMetresPerSecond(t *testing.T) {
	assert.Equal(t, "13", windKmh(3.6))
	assert.Equal(t, "0", windKmh(0))
}

func TestWeatherDisplay_ShowsLocationHeroDetailsAndForecast(t *testing.T) {
	html := renderWeatherDisplay(t, presentationView{
		Tagline: "Garden island", Location: "Madeira, Portugal", WindKmh: "13",
		Forecast: []forecastRow{{Day: "Sun", Icon: "bi-sun", Condition: "Clear sky", High: "25", Low: "17"}},
	}, application.Weather{Temperature: 24, FeelsLike: 25, Wind: 3.5, Humidity: 60, Condition: "clear sky", Sunset: "19:30"})

	for _, want := range []string{"Madeira, Portugal", "Garden island", "Feels like 25", "13", "km/h", "Humidity", "Sunset", "19:30", "weather-forecast", "Sun", "25°", "17°"} {
		assert.Contains(t, html, want)
	}
}

func TestWeatherDisplay_NoForecastAndNoSunsetRenderNoEmptyRow(t *testing.T) {
	html := renderWeatherDisplay(t, presentationView{Location: "Portugal"}, application.Weather{Temperature: 24, Condition: "clear sky"})

	assert.NotContains(t, html, "weather-forecast")
	assert.NotContains(t, html, "Sunset")
}

func renderWeatherDisplay(t *testing.T, view presentationView, weather application.Weather) string {
	t.Helper()
	engine := html.New("./views", ".go.tpl")
	require.NoError(t, engine.Load())
	var out bytes.Buffer
	require.NoError(t, engine.Render(&out, "weather_display", map[string]any{
		"Presentation": view,
		"GeneralInfo":  application.GeneralWeatherInfo{City: "Funchal", Weather: weather},
	}))
	return out.String()
}

func TestPresentationHeroLine_TaglineWinsOverTheWikidataDescription(t *testing.T) {
	for name, tc := range map[string]struct {
		view presentationView
		want string
	}{
		"tagline and description": {presentationView{Tagline: "Garden island", Description: "island in the Atlantic"}, "Garden island"},
		"description only":        {presentationView{Description: " island in the Atlantic "}, "island in the Atlantic"},
		"tagline only":            {presentationView{Tagline: "Garden island"}, "Garden island"},
		"neither":                 {presentationView{}, ""},
		"blank description":       {presentationView{Description: "  "}, ""},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.view.HeroLine())
		})
	}
}
