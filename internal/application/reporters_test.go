package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeatherReporters_GenerateReportMergesWeatherAndWaves(t *testing.T) {
	weather := &fakeReportProvider[GeneralWeatherInfo]{result: &DataToReport[GeneralWeatherInfo]{Data: GeneralWeatherInfo{
		City: "Lisbon", Country: "PORTUGAL", Lat: 38.72, Lon: -9.14,
		Weather: Weather{Temperature: 22, Condition: "sunny"},
	}}}
	waves := &fakeReportProvider[GeneralWeatherInfo]{result: &DataToReport[GeneralWeatherInfo]{Data: GeneralWeatherInfo{Waves: Waves{Height: 1.5}}}}

	got, err := NewWeatherReporters(weather, waves).GenerateReport(context.Background(), "Lisbon")

	require.NoError(t, err)
	assert.Equal(t, "Lisbon", weather.query)
	assert.Equal(t, "38.720000,-9.140000", waves.query)
	assert.Equal(t, &GeneralWeatherInfo{
		City: "Lisbon", Country: "portugal", Lat: 38.72, Lon: -9.14,
		Weather: Weather{Temperature: 22, Condition: "sunny"}, Waves: Waves{Height: 1.5},
		EmbedURL: "https://embed.waze.com/iframe?zoom=10&lat=38.720000&lon=-9.140000",
	}, got)
}

func TestWeatherReporters_PropagatesSourceErrors(t *testing.T) {
	upstream := errors.New("weather unavailable")
	weather := &fakeReportProvider[GeneralWeatherInfo]{err: upstream}
	waves := &fakeReportProvider[GeneralWeatherInfo]{}
	got, err := NewWeatherReporters(weather, waves).GenerateReport(context.Background(), "Lisbon")
	assert.Nil(t, got)
	assert.ErrorIs(t, err, upstream)
	assert.Empty(t, waves.query)

	upstream = errors.New("waves unavailable")
	weather = &fakeReportProvider[GeneralWeatherInfo]{result: &DataToReport[GeneralWeatherInfo]{Data: GeneralWeatherInfo{City: "Lisbon", Lat: 38.72, Lon: -9.14}}}
	waves.err = upstream
	got, err = NewWeatherReporters(weather, waves).GenerateReport(context.Background(), "Lisbon")
	assert.Nil(t, got)
	assert.ErrorIs(t, err, upstream)
}

func TestWeatherReporters_SelectedCoordinates(t *testing.T) {
	place := PlaceSelection{Name: "Paris", CountryCode: "US", Lat: 36.302, Lon: -88.32671}
	weather := &fakeSelectedWeatherProvider{fakeReportProvider: &fakeReportProvider[GeneralWeatherInfo]{result: &DataToReport[GeneralWeatherInfo]{Data: GeneralWeatherInfo{
		City: "Paris", Country: "US", Lat: place.Lat, Lon: place.Lon,
	}}}}
	waves := &fakeReportProvider[GeneralWeatherInfo]{result: &DataToReport[GeneralWeatherInfo]{}}

	got, err := NewWeatherReporters(weather, waves).GenerateReportForPlace(context.Background(), place)

	require.NoError(t, err)
	assert.Equal(t, place, weather.selected)
	assert.Equal(t, "36.302000,-88.326710", waves.query)
	assert.Equal(t, "Paris", got.City)
	assert.Equal(t, "us", got.Country)
}

func TestWeatherReporters_SelectedCoordinatesErrors(t *testing.T) {
	place := PlaceSelection{Name: "Paris", CountryCode: "US", Lat: 36.302, Lon: -88.32671}
	waves := &fakeReportProvider[GeneralWeatherInfo]{}
	unsupported := NewWeatherReporters(&fakeReportProvider[GeneralWeatherInfo]{}, waves)
	got, err := unsupported.GenerateReportForPlace(context.Background(), place)
	assert.Nil(t, got)
	assert.ErrorContains(t, err, "does not support selected coordinates")

	upstream := errors.New("coordinates unavailable")
	weather := &fakeSelectedWeatherProvider{fakeReportProvider: &fakeReportProvider[GeneralWeatherInfo]{err: upstream}}
	got, err = NewWeatherReporters(weather, waves).GenerateReportForPlace(context.Background(), place)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, upstream)
	assert.Empty(t, waves.query)
}

func TestVideoStreamReporters_GenerateReport(t *testing.T) {
	videos := VideosStream{{Title: "A day in Lisbon", VideoID: "abc123"}}
	source := &fakeReportProvider[VideosStream]{result: &DataToReport[VideosStream]{Data: videos}}
	got, err := NewVideoStreamReporters(source).GenerateReport(context.Background(), "Lisbon")
	require.NoError(t, err)
	assert.Equal(t, "Lisbon", source.query)
	assert.Equal(t, videos, *got)

	upstream := errors.New("videos unavailable")
	source.err = upstream
	got, err = NewVideoStreamReporters(source).GenerateReport(context.Background(), "Lisbon")
	assert.Nil(t, got)
	assert.ErrorIs(t, err, upstream)
}

type fakeReportProvider[T any] struct {
	result *DataToReport[T]
	err    error
	query  string
}

func (f *fakeReportProvider[T]) FetchReportData(_ context.Context, query ...string) (*DataToReport[T], error) {
	if len(query) > 0 {
		f.query = query[0]
	}
	return f.result, f.err
}

func (f *fakeReportProvider[T]) FetchGeneralInfo(context.Context, ...string) (*DataToReport[T], error) {
	return f.result, f.err
}

type fakeSelectedWeatherProvider struct {
	*fakeReportProvider[GeneralWeatherInfo]
	selected PlaceSelection
}

func (f *fakeSelectedWeatherProvider) FetchReportDataForPlace(_ context.Context, place PlaceSelection) (*DataToReport[GeneralWeatherInfo], error) {
	f.selected = place
	return f.result, f.err
}
