package application

import (
	"context"
	"fmt"
	"strings"
)

const weatherEmbedURL = "https://embed.waze.com/iframe?zoom=10&lat=%f&lon=%f"

type Reporter[T any] interface {
	GenerateReport(ctx context.Context, localization string) (*T, error)
}

type ReporterProvider[T any] interface {
	FetchReportData(ctx context.Context, _ ...string) (*DataToReport[T], error)
	FetchGeneralInfo(ctx context.Context, _ ...string) (*DataToReport[T], error)
}

type ReporterPublisher[T any] interface {
	PublishReportData(ctx context.Context, weather *GeneralWeatherInfo) error
}

type DataToReport[T any] struct {
	Data T
}

type GeneralWeatherInfo struct {
	Weather
	Waves
	City     string  `json:"city"`
	Country  string  `json:"country"`
	Lon      float64 `json:"lon"`
	Lat      float64 `json:"lat"`
	EmbedURL string  `json:"embed_url"`
}

type WeatherReporters struct {
	weatherApi ReporterProvider[GeneralWeatherInfo]
	wavesApi   ReporterProvider[GeneralWeatherInfo]
}

func NewWeatherReporters(weatherApi ReporterProvider[GeneralWeatherInfo], wavesApi ReporterProvider[GeneralWeatherInfo]) *WeatherReporters {
	return &WeatherReporters{
		weatherApi: weatherApi,
		wavesApi:   wavesApi,
	}
}

type Weather struct {
	Temperature float64 `json:"temperature"`
	FeelsLike   float64 `json:"feels_like"`
	Wind        float64 `json:"wind"`
	Humidity    float64 `json:"humidity"`
	Condition   string  `json:"condition"`
}

type Waves struct {
	Height float64 `json:"height"`
}

func (s *WeatherReporters) GenerateReport(ctx context.Context, city string) (*GeneralWeatherInfo, error) {
	weatherInfo, err := s.weatherApi.FetchReportData(ctx, city)
	if err != nil {
		return nil, err
	}
	return s.finishWeatherReport(ctx, weatherInfo)
}

// GenerateReportForPlace uses the chosen coordinates when the weather provider supports them.
func (s *WeatherReporters) GenerateReportForPlace(ctx context.Context, place PlaceSelection) (*GeneralWeatherInfo, error) {
	provider, ok := s.weatherApi.(interface {
		FetchReportDataForPlace(context.Context, PlaceSelection) (*DataToReport[GeneralWeatherInfo], error)
	})
	if !ok {
		return nil, fmt.Errorf("weather provider does not support selected coordinates")
	}
	weatherInfo, err := provider.FetchReportDataForPlace(ctx, place)
	if err != nil {
		return nil, err
	}
	return s.finishWeatherReport(ctx, weatherInfo)
}

func (s *WeatherReporters) finishWeatherReport(ctx context.Context, weatherInfo *DataToReport[GeneralWeatherInfo]) (*GeneralWeatherInfo, error) {

	cityCoordinates := fmt.Sprintf("%f,%f", weatherInfo.Data.Lat, weatherInfo.Data.Lon)

	waveInfo, err := s.wavesApi.FetchReportData(ctx, cityCoordinates)
	if err != nil {
		return nil, err
	}

	return &GeneralWeatherInfo{
		City:     weatherInfo.Data.City,
		Country:  strings.ToLower(weatherInfo.Data.Country),
		Lat:      weatherInfo.Data.Lat,
		Lon:      weatherInfo.Data.Lon,
		Waves:    waveInfo.Data.Waves,
		Weather:  weatherInfo.Data.Weather,
		EmbedURL: fmt.Sprintf(weatherEmbedURL, weatherInfo.Data.Lat, weatherInfo.Data.Lon),
	}, nil

}

func NewVideoStreamReporters(youtubeApi ReporterProvider[VideosStream]) *VideoStreamReporters {
	return &VideoStreamReporters{
		youtubeApi: youtubeApi,
	}
}

type VideoStreamReporters struct {
	youtubeApi ReporterProvider[VideosStream]
}

type GeneralVideoStreamInfo struct {
	Title   string
	VideoID string
}

type VideosStream []GeneralVideoStreamInfo

func (s *VideoStreamReporters) GenerateReport(ctx context.Context, city string) (*VideosStream, error) {
	videos, err := s.youtubeApi.FetchReportData(ctx, city)
	if err != nil {
		return nil, err
	}
	return &videos.Data, nil
}

type Hotels []Hotel

type Hotel struct {
	HotelName    string
	HotelURL     string
	HotelPrice   string
	HotelRating  float64
	HotelAddress string
	HotelMapURL  string
	ContactPhone string
	PriceRange   string
	HotelPhotos  []string
	HotelReviews []HotelReview
}

type HotelReview struct {
	AuthorName string
	Text       string
	Rating     float64
}

type FlightsInfo []FlightInfo

type FlightInfo struct {
	DepartureDate  string
	ArrivalDate    string
	Airline        string
	FlightNumber   string
	FlightDuration string
	Price          float64
	Origin         string
	Destination    string
	FlightMapURL   string
}

// Geocoding API Response Structs
type GeocodingResponse []GeocodingResult

type GeocodingResult struct {
	Name       string            `json:"name"`
	LocalNames map[string]string `json:"local_names,omitempty"`
	Lat        float64           `json:"lat"`
	Lon        float64           `json:"lon"`
	Country    string            `json:"country"`
	State      string            `json:"state,omitempty"`
}

type Coordinates struct {
	CoordinatesWithName []CoordinatesWithName
}

type CoordinatesWithName struct {
	Name             string
	Coordinates      []interface{}
	CoordinatesFloat []float64
}
