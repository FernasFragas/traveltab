package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"weatherservice/internal/application"
)

const stormGlassWebhookURL = "https://api.stormglass.io/v2/weather/point"
const stormGlassStationsAPIURL = "https://api.stormglass.io/v2/tide/stations"

type StormGlassAPI struct {
	client *http.Client

	key string
}

func NewStormGlassAPI(key string) *StormGlassAPI {
	return &StormGlassAPI{
		client: http.DefaultClient,
		key:    key,
	}
}

func (api *StormGlassAPI) FetchReportData(ctx context.Context, city ...string) (*application.DataToReport[application.GeneralWeatherInfo], error) {
	if api.client == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	coordinates := strings.Split(city[0], ",")

	// Create the request
	req, err := http.NewRequest("GET", stormGlassWebhookURL, nil)
	if err != nil {
		return nil, err
	}

	lat, err := strconv.ParseFloat(coordinates[0], 64)
	if err != nil {
		return nil, err
	}

	lon, err := strconv.ParseFloat(coordinates[1], 64)
	if err != nil {
		return nil, err
	}
	//add query params
	q := req.URL.Query()
	q.Add("lat", strconv.FormatFloat(lat, 'f', -1, 64))
	q.Add("lng", strconv.FormatFloat(lon, 'f', -1, 64))
	q.Add("params", "waveHeight")

	// Calculate the timestamp for the previous hour
	oneHourAgo := time.Now().Add(-1 * time.Hour).Unix()
	q.Add("time", strconv.FormatInt(oneHourAgo, 10))

	req.URL.RawQuery = q.Encode()

	// Add the API key to the request header
	req.Header.Set("Authorization", api.key)

	// Send the request
	resp, err := api.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var data waves

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, err
	}

	var waveHeight float64
	if len(data.Hours) == 0 {
		waveHeight = 0
	} else {
		waveHeight = data.Hours[len(data.Hours)-1].WaveHeight.Noaa
	}

	return &application.DataToReport[application.GeneralWeatherInfo]{
		Data: application.GeneralWeatherInfo{
			Waves: application.Waves{
				Height: waveHeight,
			},
		},
	}, nil
}

func (api *StormGlassAPI) FetchGeneralInfo(ctx context.Context, _ string) (*application.DataToReport[application.GeneralWeatherInfo], error) {
	if api.client == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	stations, err := api.FetchStationsData(ctx)
	if err != nil {
		return nil, err
	}

	var generalInfo []application.GeneralWeatherInfo
	for _, station := range stations.Data {
		generalInfo = append(generalInfo, application.GeneralWeatherInfo{
			Country: station.Name,
			Lon:     station.Lng,
			Lat:     station.Lat,
		})
	}

	return &application.DataToReport[application.GeneralWeatherInfo]{
		Data: generalInfo[0],
	}, nil
}

func (api *StormGlassAPI) FetchStationsData(ctx context.Context) (*stations, error) {
	if api.client == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	// Create the request
	req, err := http.NewRequest("GET", stormGlassStationsAPIURL, nil)
	if err != nil {
		return nil, err
	}

	// Add the API key to the request header
	req.Header.Set("Authorization", api.key)

	// Send the request
	resp, err := api.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var data stations

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

type stations struct {
	Data []station `json:"data"`
}

type station struct {
	Lat    float64 `json:"lat"`
	Lng    float64 `json:"lng"`
	Name   string  `json:"name"`
	Source string  `json:"source"`
}

type waves struct {
	Hours []hour `json:"hours"`
}

type hour struct {
	Time           string         `json:"time"`
	AirTemperature airTemperature `json:"airTemperature"`
	WaveHeight     waveHeight     `json:"waveHeight"`
}

type airTemperature struct {
	AirTemperature string `json:"smhi"`
}

type waveHeight struct {
	Noaa  float64 `json:"noaa"`
	Meteo float64 `json:"meteo"`
}
