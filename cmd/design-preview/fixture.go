package main

import (
	"context"
	"errors"
	"strings"
	"time"

	app "weatherservice/internal/application"
	"weatherservice/internal/planner"
)

// fixtureWeather is a deterministic stand-in for the geolocation/weather/wave reporters.
// It always uses the local map origin, so a browser cannot accidentally contact a production map.
type fixtureWeather struct {
	mapURL string
	delay  time.Duration
}

func (f fixtureWeather) GenerateReport(ctx context.Context, query string) (*app.GeneralWeatherInfo, error) {
	if err := fixtureDelay(ctx, f.delay); err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(query)
	parts := strings.Split(raw, ",")
	city := strings.TrimSpace(parts[0])
	region := ""
	if len(parts) > 1 {
		region = strings.ToLower(strings.TrimSpace(parts[1]))
	}
	switch strings.ToLower(city) {
	case "":
		return nil, errors.New("fixture: empty destination")
	case "error":
		return nil, errors.New("fixture search failure")
	}

	// Ordinary destinations share Lisbon's country and coordinates. The photo fixtures below have
	// their own identities, because the photo source is asked about the resolved identity.
	country, lat, lon := "pt", 38.7223, -9.1393
	for _, name := range []string{"Lisbon", "Porto", "Nophoto", "Photoerror", "Brokenimage", "Longguide"} {
		if strings.EqualFold(city, name) {
			city = name
		}
	}
	// Real places with real coordinates, for the opt-in -live-photos run. The fixture photo
	// source has no photograph for them.
	for _, place := range []struct {
		name, country string
		lat, lon      float64
	}{
		{"Tokyo", "jp", 35.6895, 139.6917}, {"Tavira", "pt", 37.1264, -7.6506}, {"Porto", "pt", 41.1494, -8.6108},
	} {
		if strings.EqualFold(city, place.name) {
			city, country, lat, lon = place.name, place.country, place.lat, place.lon
		}
	}
	if strings.EqualFold(city, "Paris") {
		city = "Paris"
		switch region {
		case "us", "usa", "texas", "tx", "united states":
			country, lat, lon = "us", 33.6609, -95.5555 // Paris, Texas
		default:
			country, lat, lon = "fr", 48.8589, 2.3200
		}
	}
	condition := "Clear"
	if strings.EqualFold(city, "Porto") {
		condition = "Partly cloudy"
	}
	return &app.GeneralWeatherInfo{
		City:     city,
		Country:  country,
		Lat:      lat,
		Lon:      lon,
		Weather:  app.Weather{Temperature: 22, FeelsLike: 22, Humidity: 60, Wind: 10, Condition: condition},
		Waves:    app.Waves{Height: 1.2},
		EmbedURL: f.mapURL,
	}, nil
}

func (f fixtureWeather) GenerateReportForPlace(ctx context.Context, place app.PlaceSelection) (*app.GeneralWeatherInfo, error) {
	info, err := f.GenerateReport(ctx, place.Name+", "+place.CountryCode)
	if err != nil {
		return nil, err
	}
	info.Country = strings.ToLower(place.CountryCode)
	info.Lat, info.Lon = place.Lat, place.Lon
	return info, nil
}

type fixtureVideos struct{ scenario string }

func (f fixtureVideos) GenerateReport(_ context.Context, query string) (*app.VideosStream, error) {
	city := strings.TrimSpace(strings.Split(strings.TrimPrefix(query, "Turistic places in "), ",")[0])
	if strings.EqualFold(city, "Porto") || f.scenario == "videos-0" || f.scenario == "fallback" {
		return &app.VideosStream{}, nil
	}
	videos := app.VideosStream{
		{Title: "Fixture travel film", VideoID: "dQw4w9WgXcQ"},
		{Title: "Fixture neighborhood walk", VideoID: "9bZkp7q19f0"},
		{Title: "Fixture food tour", VideoID: "kJQP7kiw5Fk"},
		{Title: "Fixture museum tour", VideoID: "YQHsXMglF9A"},
		{Title: "Fixture evening view", VideoID: "fJ9rUzIMcZQ"},
		{Title: "Fixture harbor tour", VideoID: "9EcjWd-O4jI"},
	}
	if f.scenario == "videos-1" {
		videos = videos[:1]
	}
	if f.scenario == "videos-10" {
		for _, id := range []string{"aqz-KE-bpKQ", "M7lc1UVf-VE", "jNQXAC9IVRw", "LXb3EKWsInQ"} {
			videos = append(videos, app.GeneralVideoStreamInfo{Title: "Fixture extra travel film", VideoID: id})
		}
	}
	return &videos, nil
}

type fixtureTrips struct {
	delay    time.Duration
	scenario string
}

func (f fixtureTrips) Plan(ctx context.Context, req planner.Request) (*planner.Plan, error) {
	if err := fixtureDelay(ctx, f.delay); err != nil {
		return nil, err
	}
	if req.Days == 4 {
		return nil, errors.New("fixture planning failure")
	}
	if req.Days < 1 || req.Days > planner.MaxDays {
		return nil, errors.New("fixture: days out of range")
	}
	places := fixturePlaces()
	stays := fixtureStays()
	if f.scenario == "photos-missing" || f.scenario == "fallback" {
		for i := range places {
			places[i].Image = ""
		}
	}
	switch f.scenario {
	case "stays-0", "stays-unavailable", "fallback":
		stays = nil
	case "stays-1":
		stays = stays[:1]
	case "stays-6":
		stays = append(stays, planner.Stay{Name: "Fixture Riverside Hotel", Kind: "hotel"}, planner.Stay{Name: "Fixture Garden Hostel", Kind: "hostel"})
	}
	p := &planner.Plan{
		Request:    req,
		BookingURL: planner.BookingURL(req.City, req.Country, req.Start, req.Days),
		Stays:      stays,
	}

	if f.scenario == "stays-unavailable" || f.scenario == "fallback" {
		p.StaysNote = "Nearby stays are temporarily unavailable. You can still search Booking.com."
	}

	// Keep the requested day count: this is an interface fixture, not the production algorithm.
	for i := 0; i < req.Days; i++ {
		day := planner.Day{Date: req.Start.AddDate(0, 0, i), WalkKM: 3.1 + float64(i)}
		switch i {
		case 0:
			rain := 0.4
			day.Certain = true
			day.Forecast = &planner.DayForecast{Date: day.Date, RainMM: &rain}
			day.Stops = places[:4]
		case 1:
			rain := 12.4
			day.Certain = true
			day.Forecast = &planner.DayForecast{Date: day.Date, RainMM: &rain}
			day.Rainy = true
			day.Stops = []planner.Place{places[0], places[2], places[4]}
		case 2:
			rain := 2.1
			day.Forecast = &planner.DayForecast{Date: day.Date, RainMM: &rain}
			day.Stops = []planner.Place{places[5], places[6], places[4]}
		default:
			// Deliberately leaves Forecast nil, so a five-day plan exercises absent forecasts.
		}
		switch f.scenario {
		case "forecast-absent", "fallback":
			day.Forecast, day.Certain, day.Rainy = nil, false, false
		case "forecast-uncertain":
			rain := 2.1
			day.Forecast = &planner.DayForecast{Date: day.Date, RainMM: &rain}
			day.Certain, day.Rainy = false, false
		}
		p.Days = append(p.Days, day)
	}
	return p, nil
}

// fixturePlaces covers a photo-bearing stop, a mixed stop, an indoor stop, and deliberately
// long labels. Wikimedia Image values are stable Commons file names, not local production assets.
func fixturePlaces() []planner.Place {
	return []planner.Place{
		{ID: "fixture-1", Name: "Belém Tower", Kind: planner.Outdoor, Image: "Torre de Belém 1.jpg", Lat: 38.6916, Lon: -9.216},
		{ID: "fixture-2", Name: "Jerónimos Monastery", Kind: planner.Mixed, Lat: 38.6979, Lon: -9.2065},
		{ID: "fixture-3", Name: "A Very Long Destination Attraction Name That Must Wrap Without Losing Accessibility", Kind: planner.Indoor, Lat: 38.7139, Lon: -9.1394},
		{ID: "fixture-4", Name: "Time Out Market Lisboa", Kind: planner.Mixed, Lat: 38.7071, Lon: -9.1459},
		{ID: "fixture-5", Name: "National Tile Museum", Kind: planner.Indoor, Lat: 38.7373, Lon: -9.1129},
		{ID: "fixture-6", Name: "Alfama viewpoint", Kind: planner.Outdoor, Lat: 38.7118, Lon: -9.1293},
		{ID: "fixture-7", Name: "Praça do Comércio", Kind: planner.Outdoor, Lat: 38.7075, Lon: -9.1364},
	}
}

// fixtureStays includes a starred property, an unstarred hostel, a websiteless guest house,
// and a long-name guest house.
func fixtureStays() []planner.Stay {
	return []planner.Stay{
		{Name: "Fixture Boutique Hotel", Kind: "hotel", Stars: 4, Website: "https://example.com/hotel", Lat: 38.7100, Lon: -9.1400},
		{Name: "Fixture Hostel", Kind: "hostel", Lat: 38.7110, Lon: -9.1410},
		{Name: "Fixture Guest House", Kind: "guest_house", Lat: 38.7120, Lon: -9.1420},
		{Name: "Another Unusually Long Fixture Accommodation Name To Test Compact Stay Wrapping", Kind: "guest_house", Lat: 38.7130, Lon: -9.1430},
	}
}

// Delay is cancellable so tests can verify both wiring paths without sleeping two seconds.
func fixtureDelay(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func validScenario(name string) bool {
	switch name {
	case "default", "photos-missing", "forecast-absent", "forecast-uncertain", "stays-0", "stays-1", "stays-6", "stays-unavailable", "videos-0", "videos-1", "videos-10", "fallback":
		return true
	default:
		return false
	}
}
