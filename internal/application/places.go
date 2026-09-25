package application

import "context"

// PlaceSuggestion is a place returned while a visitor types a destination.
type PlaceSuggestion struct {
	Name, Region, Country, CountryCode string
	Lat, Lon                           float64
}

// PlaceSuggester finds places without making the web handler depend on a provider.
type PlaceSuggester interface {
	Suggest(context.Context, string) ([]PlaceSuggestion, error)
}

// PlaceSelection carries the coordinates of an explicitly chosen suggestion.
type PlaceSelection struct {
	Name, CountryCode string
	Lat, Lon          float64
}

type PlaceWeatherReporter interface {
	GenerateReportForPlace(context.Context, PlaceSelection) (*GeneralWeatherInfo, error)
}
