package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"weatherservice/internal/application"
)

const openMateoURl = "https://marine-api.open-meteo.com/v1/marine"

type OpenMateoAPI struct {
	client *http.Client
}

func NewOpenMateoAPI() *OpenMateoAPI {
	return &OpenMateoAPI{
		client: http.DefaultClient,
	}
}

func (api *OpenMateoAPI) FetchReportData(ctx context.Context, city ...string) (*application.DataToReport[application.GeneralWeatherInfo], error) {
	if api.client == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	coordinates := strings.Split(city[0], ",")

	req, err := http.NewRequest("GET", openMateoURl, nil)
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
	q.Add("latitude", strconv.FormatFloat(lat, 'f', -1, 64))
	q.Add("longitude", strconv.FormatFloat(lon, 'f', -1, 64))
	q.Add("hourly", "wave_height")
	req.URL.RawQuery = q.Encode()

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var response OpenMateoResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	waveHeight := response.Hourly.WaveHeight[len(response.Hourly.WaveHeight)-1]

	return &application.DataToReport[application.GeneralWeatherInfo]{
		Data: application.GeneralWeatherInfo{
			Waves: application.Waves{
				Height: waveHeight,
			},
		},
	}, nil
}

func (api *OpenMateoAPI) FetchGeneralInfo(ctx context.Context, city ...string) (*application.DataToReport[application.GeneralWeatherInfo], error) {
	//not implemented
	return nil, nil
}

/*
 "latitude": 52.52,
  "longitude": 13.419,
  "elevation": 44.812,
  "generationtime_ms": 2.2119,
  "utc_offset_seconds": 0,
  "timezone": "Europe/Berlin",
  "timezone_abbreviation": "CEST",
  "hourly": {
    "time": ["2022-07-01T00:00", "2022-07-01T01:00", "2022-07-01T02:00", ...],
    "wave_height": [1, 1.7, 1.7, 1.5, 1.5, 1.8, 2.0, 1.9, 1.3, ...]
  },
  "hourly_units": {
    "wave_height": "m"
  },
*/

type OpenMateoResponse struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Hourly    struct {
		Time       []string  `json:"time"`
		WaveHeight []float64 `json:"wave_height"`
	} `json:"hourly"`
	HourlyUnits struct {
		WaveHeight string `json:"wave_height"`
	} `json:"hourly_units"`
}
