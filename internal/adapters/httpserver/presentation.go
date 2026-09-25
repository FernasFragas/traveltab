package httpserver

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strings"

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
	MapURL, ConditionIcon               string
	Videos                              []videoView
	Guide                               guideView // reviewed city guide, or the explicit "no guide yet" state
}

type stayCardView struct {
	Name, Kind, Website string
	Stars               int
	Image               *imageView
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
