// Package weatherservice package provides a simple API to fetch weather reports for a given city.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"weatherservice/internal/application"
)

const openWeatherMapWebhookURL = "https://api.openweathermap.org/data/2.5/weather?"
const openWeatherGeoLocationURL = "http://api.openweathermap.org/geo/1.0/direct?"

type WeatherAPI struct {
	client *http.Client

	key string
}

func NewWeatherAPI(key string) *WeatherAPI {
	return &WeatherAPI{
		client: http.DefaultClient,
		key:    key,
	}
}

func (api *WeatherAPI) FetchReportData(ctx context.Context, city ...string) (*application.DataToReport[application.GeneralWeatherInfo], error) {
	toSearch := strings.Split(city[0], ",")
	if len(toSearch) == 1 {
		return nil, fmt.Errorf("city and country are required")
	}
	toSearch[0] = strings.TrimSpace(toSearch[0])
	toSearch[1] = strings.TrimSpace(toSearch[1])
	if toSearch[1] == "" {
		toSearch[1] = "PT"
	}

	weather, err := api.fetchLocationInfo(ctx, toSearch[0], toSearch[1])
	if err != nil {
		return nil, err
	}

	var condition string
	if len(weather.Weather) > 0 {
		condition = weather.Weather[0].Description
	}

	return &application.DataToReport[application.GeneralWeatherInfo]{
		Data: application.GeneralWeatherInfo{
			City:    weather.Name,
			Country: weather.Sys.Country,
			Lon:     weather.Coordinates.Lon,
			Lat:     weather.Coordinates.Las,
			Weather: application.Weather{
				Temperature: weather.Temperature.Temperature,
				FeelsLike:   weather.Temperature.FeelsLike,
				Wind:        weather.Wind.Speed,
				Humidity:    weather.Temperature.Humidity,
				Condition:   condition,
			},
		},
	}, nil
}

func (api *WeatherAPI) FetchGeneralInfo(ctx context.Context, city ...string) (*application.DataToReport[application.GeneralWeatherInfo], error) {
	toSearch := strings.Split(city[0], ",")
	toSearch[0] = strings.TrimSpace(toSearch[0])
	toSearch[1] = strings.TrimSpace(toSearch[1])

	generalInfo, err := api.fetchLocationInfo(ctx, toSearch[0], toSearch[1])
	if err != nil {
		return nil, err
	}

	return &application.DataToReport[application.GeneralWeatherInfo]{
		Data: application.GeneralWeatherInfo{
			City:    generalInfo.Name,
			Country: generalInfo.Sys.Country,
			Lon:     generalInfo.Coordinates.Lon,
			Lat:     generalInfo.Coordinates.Las,
		},
	}, nil
}

type openWeatherRequestParams struct {
	City    string
	Country string
	Lat     float64
	Lon     float64
}

func (api *WeatherAPI) fetchLocationInfo(ctx context.Context, city string, country string) (*weatherData, error) {
	if api.client == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	coordinates, err := api.fetchCoordinates(ctx, city, country)
	if err != nil {
		return nil, err
	}

	weather, err := api.fetchWeather(ctx, *coordinates)
	if err != nil {
		return nil, err
	}

	weather.Name = city

	return weather, nil
}

func (api *WeatherAPI) fetchCoordinates(ctx context.Context, city string, country string) (*Coordinates, error) {
	requestParamsForCoordinates := openWeatherRequestParams{
		City:    city,
		Country: country,
	}

	queryParams, err := api.setupQueryParams(ctx, requestParamsForCoordinates)
	if err != nil {
		return nil, err
	}

	apiUrl := fmt.Sprintf("%s%s", openWeatherGeoLocationURL, queryParams.Encode())

	resp, err := api.client.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	var coordinates GeocodingResponse

	err = json.NewDecoder(resp.Body).Decode(&coordinates)
	if err != nil {
		return nil, err
	}

	if len(coordinates) == 0 {
		return nil, fmt.Errorf("no location found for %s, %s", city, country)
	}

	return &Coordinates{
		Lon: coordinates[0].Lon,
		Las: coordinates[0].Lat,
	}, nil
}

func (api *WeatherAPI) fetchWeather(ctx context.Context, coordinates Coordinates) (*weatherData, error) {
	queryParams, err := api.setupQueryParams(ctx, openWeatherRequestParams{
		Lat: coordinates.Las,
		Lon: coordinates.Lon,
	})
	if err != nil {
		return nil, err
	}

	apiUrl := fmt.Sprintf("%s%s", openWeatherMapWebhookURL, queryParams.Encode())

	resp, err := api.client.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	var weather weatherData

	err = json.NewDecoder(resp.Body).Decode(&weather)
	if err != nil {
		return nil, err
	}

	return &weather, nil
}

// checkResponse turns a non-200 OpenWeather response into an error carrying the API's message.
func checkResponse(resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	var apiErr struct {
		Message string `json:"message"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&apiErr)

	// Proxies and outages can answer with HTML, which has no message to decode.
	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(resp.StatusCode)
	}

	return fmt.Errorf("openweather request failed with status %d: %s", resp.StatusCode, apiErr.Message)
}

func (api *WeatherAPI) setupQueryParams(ctx context.Context, requestParams openWeatherRequestParams) (url.Values, error) {
	if api.client == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	queryParams := url.Values{}
	if requestParams.City != "" && requestParams.Country != "" && requestParams.Lat == 0 && requestParams.Lon == 0 {
		locationToSearch := fmt.Sprintf("%s,%s", requestParams.City, requestParams.Country)

		queryParams.Add("q", locationToSearch)
	} else if requestParams.Lat != 0 && requestParams.Lon != 0 {
		queryParams.Add("lat", fmt.Sprintf("%f", requestParams.Lat))
		queryParams.Add("lon", fmt.Sprintf("%f", requestParams.Lon))
	} else {
		return nil, fmt.Errorf("invalid request parameters")
	}

	queryParams.Add("units", "metric")
	queryParams.Add("APPID", api.key)

	return queryParams, nil
}

// unexported
type weatherData struct {
	Coordinates Coordinates `json:"coord"`
	Weather     weatherInfo `json:"weather"`
	Temperature temperature `json:"main"`
	Wind        wind        `json:"wind"`
	Sys         sys         `json:"sys"`
	Name        string      `json:"name"`
}

type Coordinates struct {
	Lon float64 `json:"lon"`
	Las float64 `json:"lat"`
}

type weatherInfo []struct {
	WeatherState string `json:"main"`
	Description  string `json:"description"`
}

type temperature struct {
	Temperature float64 `json:"temp"`
	FeelsLike   float64 `json:"feels_like"`
	TempMin     float64 `json:"temp_min"`
	TempMax     float64 `json:"tem_max"`
	Humidity    float64 `json:"humidity"`
}

type sys struct {
	Country string  `json:"country"`
	Sunrise float64 `json:"sunrise"`
	Sunset  float64 `json:"sunset"`
}

type wind struct {
	Speed float64 `json:"speed"`
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
