package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"
	"weatherservice/internal/application"
)

const openMeteoGeocodingURL = "https://geocoding-api.open-meteo.com/v1/search"

type OpenMeteoGeocodingAPI struct {
	client *http.Client
}

var _ application.PlaceSuggester = (*OpenMeteoGeocodingAPI)(nil)

func NewOpenMeteoGeocodingAPI(client *http.Client) *OpenMeteoGeocodingAPI {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &OpenMeteoGeocodingAPI{client: client}
}

func (api *OpenMeteoGeocodingAPI) Suggest(ctx context.Context, query string) ([]application.PlaceSuggestion, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openMeteoGeocodingURL, nil)
	if err != nil {
		return nil, fmt.Errorf("open-meteo geocoding request: %w", err)
	}
	q := req.URL.Query()
	q.Set("name", query)
	q.Set("count", "6")
	q.Set("language", "en")
	q.Set("format", "json")
	req.URL.RawQuery = q.Encode()
	req.Header.Set("User-Agent", UserAgent)

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("open-meteo geocoding: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open-meteo geocoding: unexpected status %d", resp.StatusCode)
	}
	var decoded struct {
		Results []struct {
			Name        string  `json:"name"`
			Country     string  `json:"country"`
			CountryCode string  `json:"country_code"`
			Admin1      string  `json:"admin1"`
			Latitude    float64 `json:"latitude"`
			Longitude   float64 `json:"longitude"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("open-meteo geocoding: decode: %w", err)
	}

	places := make([]application.PlaceSuggestion, 0, len(decoded.Results))
	seen := make(map[[3]string]bool)
	for _, result := range decoded.Results {
		place := application.PlaceSuggestion{
			Name: result.Name, Region: result.Admin1,
			Country: result.Country, CountryCode: result.CountryCode,
			Lat: result.Latitude, Lon: result.Longitude,
		}
		key := [3]string{place.Name, place.Region, place.Country}
		if place.Name == "" || place.Country == "" || place.CountryCode == "" || seen[key] ||
			math.IsNaN(place.Lat) || math.IsNaN(place.Lon) || math.IsInf(place.Lat, 0) || math.IsInf(place.Lon, 0) ||
			place.Lat < -90 || place.Lat > 90 || place.Lon < -180 || place.Lon > 180 || place.Lat == 0 && place.Lon == 0 {
			continue
		}
		seen[key] = true
		places = append(places, place)
	}
	return places, nil
}
