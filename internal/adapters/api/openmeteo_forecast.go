package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	// tzdata is embedded so IANA time zone names resolve inside the slim
	// Docker image, which ships no system time zone database.
	_ "time/tzdata"

	"weatherservice/internal/planner"
)

const openMeteoForecastURL = "https://api.open-meteo.com/v1/forecast"

// openMeteoForecastTimeout is the default timeout when no client is given.
const openMeteoForecastTimeout = 30 * time.Second

// OpenMeteoForecastAPI reads daily rain totals from Open-Meteo. It is separate
// from OpenMateoAPI, which reads the marine forecast.
type OpenMeteoForecastAPI struct {
	client *http.Client
}

var _ planner.ForecastSource = (*OpenMeteoForecastAPI)(nil)

// NewOpenMeteoForecastAPI returns a forecast source. A nil client means a
// default client with a 30 s timeout.
func NewOpenMeteoForecastAPI(client *http.Client) *OpenMeteoForecastAPI {
	if client == nil {
		client = &http.Client{Timeout: openMeteoForecastTimeout}
	}

	return &OpenMeteoForecastAPI{client: client}
}

// openMeteoForecastResponse is the slice of the Open-Meteo answer we use.
// Rain totals are pointers, because Open-Meteo sends null for missing days.
type openMeteoForecastResponse struct {
	Timezone string `json:"timezone"`
	Daily    struct {
		Time             []string   `json:"time"`
		PrecipitationSum []*float64 `json:"precipitation_sum"`
	} `json:"daily"`
}

// DailyForecast returns up to ForecastMaxDays of daily rain totals, dated at
// midnight in the city's own time zone.
func (api *OpenMeteoForecastAPI) DailyForecast(ctx context.Context, lat, lon float64) (*planner.Forecast, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openMeteoForecastURL, nil)
	if err != nil {
		return nil, fmt.Errorf("open-meteo forecast request: %w", err)
	}

	q := req.URL.Query()
	q.Set("latitude", strconv.FormatFloat(lat, 'f', -1, 64))
	q.Set("longitude", strconv.FormatFloat(lon, 'f', -1, 64))
	q.Set("daily", "precipitation_sum")
	q.Set("timezone", "auto")
	q.Set("forecast_days", strconv.Itoa(planner.ForecastMaxDays))
	req.URL.RawQuery = q.Encode()
	req.Header.Set("User-Agent", UserAgent)

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("open-meteo forecast: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open-meteo forecast: unexpected status %d", resp.StatusCode)
	}

	var decoded openMeteoForecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("open-meteo forecast: decode: %w", err)
	}

	location, err := time.LoadLocation(decoded.Timezone)
	if err != nil {
		return nil, fmt.Errorf("open-meteo forecast: unknown time zone %q: %w", decoded.Timezone, err)
	}

	forecast := &planner.Forecast{
		Timezone: decoded.Timezone,
		Days:     make([]planner.DayForecast, 0, len(decoded.Daily.Time)),
	}

	for i, day := range decoded.Daily.Time {
		date, err := time.ParseInLocation("2006-01-02", day, location)
		if err != nil {
			return nil, fmt.Errorf("open-meteo forecast: bad date %q: %w", day, err)
		}

		var rain *float64
		if i < len(decoded.Daily.PrecipitationSum) && decoded.Daily.PrecipitationSum[i] != nil {
			mm := *decoded.Daily.PrecipitationSum[i]
			rain = &mm
		}

		forecast.Days = append(forecast.Days, planner.DayForecast{Date: date, RainMM: rain})
	}

	return forecast, nil
}
