package httpserver

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strings"
	"time"

	"weatherservice/internal/application"
	"weatherservice/internal/planner"
)

// The manifest ships in the binary; Docker already copies its public image files.
// Decoration is rebuilt on render and never changes cached provider JSON.
//
//go:embed presentation_assets/manifest.json
var presentationManifestJSON []byte

type imageView struct {
	URL, Alt, Credit, CreditURL string
	License, LicenseURL         string
	Width, Height               int
}

type imageAsset struct {
	imageView
	Source, Changes string
}

type destinationAsset struct {
	City, Country, Tagline, PhotoCaption, CountryLabel string
	Hero                                               *imageAsset
}

type stayAsset struct {
	City, Country, Name string
	Lat, Lon            float64
	Image               imageAsset
}

type presentationManifest struct {
	Destinations []destinationAsset
	Stays        []stayAsset
}

var curatedPresentation = func() presentationManifest {
	var manifest presentationManifest
	if err := json.Unmarshal(presentationManifestJSON, &manifest); err != nil {
		panic("invalid embedded presentation manifest: " + err.Error())
	}
	return manifest
}()

type videoView struct {
	Title, VideoID, ThumbnailURL, WatchURL string
}

type presentationView struct {
	Hero                                *imageView
	PhotoUnavailable                    bool // no photograph could be shown: say so instead of an empty frame
	Tagline, PhotoCaption, CountryLabel string
	Description                         string // the city's Wikidata description; shown only when there is no Tagline
	Location                            string // "Region, Country", or the country alone when there is no region
	MapURL, ConditionIcon, WindKmh      string
	Forecast                            []forecastRow // empty when the forecast is unavailable: the card then shows current weather only
	Videos                              []videoView
	Guide                               guideView // reviewed city guide, or the explicit "no guide yet" state
}

// HeroLine is the line under the city name: the curated tagline, or else the Wikidata
// description, or "" when there is neither, so the template renders no element.
func (v presentationView) HeroLine() string {
	if tagline := strings.TrimSpace(v.Tagline); tagline != "" {
		return tagline
	}

	return strings.TrimSpace(v.Description)
}

// forecastRow is one day of the card's forecast. Temperatures are whole degrees Celsius.
type forecastRow struct {
	Day, Icon, Condition, High, Low string
}

type stayCardView struct {
	Name, Kind, Website string
	Stars               int
	Image               *imageView
}

// SetForecastSource switches the card's daily forecast on. Without a source, or when it fails,
// the card shows current weather only.
func (s *Server) SetForecastSource(source planner.ForecastSource) {
	s.forecastSource = source
}

func sameIdentity(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// OpenWeather returns ISO country codes; cached/manual requests may carry names.
// Only explicitly curated aliases are equivalent, so namesake cities stay separate.
func countryIdentity(country string) string {
	country = strings.ToLower(strings.TrimSpace(country))
	if country == "pt" {
		return "portugal"
	}
	return country
}

func newPresentation(info application.GeneralWeatherInfo, videos application.VideosStream) presentationView {
	view := presentationView{
		MapURL:        cityMapURL(info.Lat, info.Lon),
		ConditionIcon: conditionIcon(info.Condition),
		WindKmh:       windKmh(info.Wind),
		Videos:        presentationVideos(videos),
	}
	for _, destination := range curatedPresentation.Destinations {
		if !sameIdentity(info.City, destination.City) || countryIdentity(info.Country) != countryIdentity(destination.Country) {
			continue
		}
		view.Tagline, view.PhotoCaption, view.CountryLabel = destination.Tagline, destination.PhotoCaption, destination.CountryLabel
		if destination.Hero != nil {
			photo := destination.Hero.imageView
			view.Hero = &photo
		}
		break
	}
	return view
}

func validCoordinates(lat, lon float64) bool {
	return !math.IsNaN(lat) && !math.IsNaN(lon) && !math.IsInf(lat, 0) && !math.IsInf(lon, 0) && lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

func cityMapURL(lat, lon float64) string {
	if !validCoordinates(lat, lon) {
		return ""
	}
	return "https://www.google.com/maps/search/?" + url.Values{
		"api": {"1"}, "query": {fmt.Sprintf("%.6f,%.6f", lat, lon)},
	}.Encode()
}

var youtubeVideoID = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

func presentationVideos(videos application.VideosStream) []videoView {
	result := make([]videoView, 0, len(videos))
	for _, video := range videos {
		if !youtubeVideoID.MatchString(video.VideoID) {
			continue
		}
		result = append(result, videoView{
			Title: video.Title, VideoID: video.VideoID,
			ThumbnailURL: "https://i.ytimg.com/vi/" + video.VideoID + "/hqdefault.jpg",
			WatchURL:     "https://www.youtube.com/watch?v=" + video.VideoID,
		})
	}
	return result
}

// Coordinates plus destination and name identify a property. Exact matching is
// deliberately conservative: changed/ambiguous identities get no photograph.
func (manifest presentationManifest) stayImage(req planner.Request, stay planner.Stay) *imageView {
	if strings.TrimSpace(stay.Name) == "" || !validCoordinates(stay.Lat, stay.Lon) {
		return nil
	}
	var match *imageView
	for _, asset := range manifest.Stays {
		if sameIdentity(req.City, asset.City) && countryIdentity(req.Country) == countryIdentity(asset.Country) &&
			sameIdentity(stay.Name, asset.Name) && stay.Lat == asset.Lat && stay.Lon == asset.Lon {
			if match != nil {
				return nil // duplicate identity is ambiguous
			}
			photo := asset.Image.imageView
			match = &photo
		}
	}
	return match
}

func presentationStays(req planner.Request, stays []planner.Stay) []stayCardView {
	result := make([]stayCardView, 0, len(stays))
	for _, stay := range stays {
		result = append(result, stayCardView{
			Name: stay.Name, Kind: stay.Kind, Stars: stay.Stars, Website: stay.Website,
			Image: curatedPresentation.stayImage(req, stay),
		})
	}
	return result
}

func conditionIcon(condition string) string {
	condition = strings.ToLower(condition)
	for _, group := range []struct {
		words []string
		icon  string
	}{
		{[]string{"thunder", "storm"}, "cloud-lightning-rain"},
		{[]string{"snow", "sleet", "ice", "blizzard"}, "cloud-snow"},
		{[]string{"rain", "drizzle", "shower"}, "cloud-rain"},
		{[]string{"fog", "mist", "haze", "smoke", "dust"}, "cloud-fog2"},
		{[]string{"partly", "few clouds", "scattered clouds"}, "cloud-sun"},
		{[]string{"cloud", "overcast"}, "cloud"},
		{[]string{"clear", "sun"}, "sun"},
		{[]string{"wind"}, "wind"},
	} {
		for _, word := range group.words {
			if strings.Contains(condition, word) {
				return "bi-" + group.icon
			}
		}
	}
	return "bi-thermometer-half"
}

const (
	// presentationLookupTimeout bounds the region and forecast lookups, so a slow provider
	// cannot hold the page back; both are cached and optional.
	presentationLookupTimeout = 3 * time.Second
	// regionMatchDegrees is how far a geocoder result may sit from the resolved coordinates
	// (about 55 km) and still be the same city.
	regionMatchDegrees = 0.5
	forecastRowCount   = 4
)

// addLocationAndForecast fills the parts of the view that come from live lookups: the
// "Region, Country" line and the forecast rows. Both are optional and fail quietly.
func (s *Server) addLocationAndForecast(view *presentationView, info application.GeneralWeatherInfo) {
	ctx, cancel := context.WithTimeout(context.Background(), presentationLookupTimeout)
	defer cancel()

	region, country := s.regionAndCountry(ctx, info)
	if country == "" {
		country = view.CountryLabel
	}
	if country == "" {
		country = strings.ToUpper(strings.TrimSpace(info.Country))
	}
	view.Location = locationLabel(region, country)

	if s.forecastSource != nil {
		if forecast, err := s.forecastSource.DailyForecast(ctx, info.Lat, info.Lon); err == nil {
			view.Forecast = forecastRows(forecast)
		}
	}
}

// regionAndCountry finds the geocoder's admin1 and country name for the resolved city: the
// suggestion in the same country that sits closest to its coordinates. Empty when unknown.
func (s *Server) regionAndCountry(ctx context.Context, info application.GeneralWeatherInfo) (region, country string) {
	city := strings.TrimSpace(info.City)
	if s.placeSuggester == nil || s.suggestions == nil || city == "" {
		return "", ""
	}

	key := strings.ToLower(city)
	places, ok := s.suggestions.get(key)
	if !ok {
		var err error
		if places, err = s.placeSuggester.Suggest(ctx, city); err != nil {
			return "", ""
		}
		s.suggestions.put(key, places)
	}

	best, bestDistance := -1, regionMatchDegrees
	for i, place := range places {
		if countryIdentity(place.CountryCode) != countryIdentity(info.Country) && countryIdentity(place.Country) != countryIdentity(info.Country) {
			continue
		}
		if distance := math.Max(math.Abs(place.Lat-info.Lat), math.Abs(place.Lon-info.Lon)); distance <= bestDistance {
			best, bestDistance = i, distance
		}
	}
	if best < 0 {
		return "", ""
	}

	return places[best].Region, places[best].Country
}

// windKmh converts OpenWeather's metres per second to whole kilometres per hour.
func windKmh(metresPerSecond float64) string {
	return fmt.Sprintf("%.0f", metresPerSecond*3.6)
}

func locationLabel(region, country string) string {
	region, country = strings.TrimSpace(region), strings.TrimSpace(country)
	if region == "" || sameIdentity(region, country) {
		return country
	}
	if country == "" {
		return region
	}

	return region + ", " + country
}

// forecastRows turns the first days of the forecast that have a high, a low and a weather code
// into card rows. A day missing any of them is skipped rather than shown as "0°".
func forecastRows(forecast *planner.Forecast) []forecastRow {
	if forecast == nil {
		return nil
	}

	var rows []forecastRow
	for _, day := range forecast.Days {
		if day.MaxC == nil || day.MinC == nil || day.WeatherCode == nil {
			continue
		}
		rows = append(rows, forecastRow{
			Day: day.Date.Format("Mon"), Icon: weatherCodeIcon(*day.WeatherCode), Condition: weatherCodeLabel(*day.WeatherCode),
			High: fmt.Sprintf("%.0f", *day.MaxC), Low: fmt.Sprintf("%.0f", *day.MinC),
		})
		if len(rows) == forecastRowCount {
			break
		}
	}

	return rows
}

// weatherCodeIcon and weatherCodeLabel describe an Open-Meteo (WMO) weather code. The
// itinerary rows use them too, so a day reads the same on the card and in the plan.
func weatherCodeIcon(code int) string {
	return weatherCodeInfo(code).icon
}

func weatherCodeLabel(code int) string {
	return weatherCodeInfo(code).label
}

type weatherCodeDescription struct{ icon, label string }

func weatherCodeInfo(code int) weatherCodeDescription {
	switch code {
	case 0:
		return weatherCodeDescription{"bi-sun", "Clear sky"}
	case 1:
		return weatherCodeDescription{"bi-cloud-sun", "Mostly clear"}
	case 2:
		return weatherCodeDescription{"bi-cloud-sun", "Partly cloudy"}
	case 3:
		return weatherCodeDescription{"bi-cloud", "Overcast"}
	case 45, 48:
		return weatherCodeDescription{"bi-cloud-fog2", "Fog"}
	case 51, 53, 55:
		return weatherCodeDescription{"bi-cloud-drizzle", "Drizzle"}
	case 56, 57, 66, 67:
		return weatherCodeDescription{"bi-cloud-sleet", "Freezing rain"}
	case 61, 63:
		return weatherCodeDescription{"bi-cloud-rain", "Rain"}
	case 65:
		return weatherCodeDescription{"bi-cloud-rain-heavy", "Heavy rain"}
	case 71, 73, 75, 77:
		return weatherCodeDescription{"bi-cloud-snow", "Snow"}
	case 80, 81:
		return weatherCodeDescription{"bi-cloud-rain", "Rain showers"}
	case 82:
		return weatherCodeDescription{"bi-cloud-rain-heavy", "Heavy showers"}
	case 85, 86:
		return weatherCodeDescription{"bi-cloud-snow", "Snow showers"}
	case 95, 96, 99:
		return weatherCodeDescription{"bi-cloud-lightning-rain", "Thunderstorm"}
	}

	return weatherCodeDescription{"bi-thermometer-half", "Unknown"}
}
