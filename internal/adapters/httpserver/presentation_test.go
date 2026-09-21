package httpserver

import (
	"bytes"
	"html/template"
	"image"
	_ "image/jpeg"
	"math"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

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
