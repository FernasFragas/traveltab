package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkcorps_ResolvesCityAndMapsHotels(t *testing.T) {
	api := NewMarkcorpsAPI("test-key")
	requests := 0
	api.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		assert.Equal(t, "test-key", req.URL.Query().Get("api_key"))
		switch req.URL.Path {
		case "/mapping":
			assert.Equal(t, "Funchal", req.URL.Query().Get("name"))
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[{"document_id":"city-42"}]`))}, nil
		case "/city":
			assert.Equal(t, "city-42", req.URL.Query().Get("name"))
			assert.Equal(t, "EUR", req.URL.Query().Get("cur"))
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[{"name":"Harbor Hotel","telephone":"+351123"}]`))}, nil
		default:
			t.Fatalf("unexpected Markcorps request: %s", req.URL)
			return nil, nil
		}
	})}
	report, err := api.FetchReportData(context.Background(), "Funchal")
	require.NoError(t, err)
	require.Len(t, report.Data, 1)
	assert.Equal(t, "Harbor Hotel", report.Data[0].HotelName)
	assert.Equal(t, "+351123", report.Data[0].HotelURL)
	assert.Equal(t, 2, requests)
}

func TestMarkcorps_ReportsLocationAndHotelFailures(t *testing.T) {
	api := &MarkcorpsAPI{client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/mapping" {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[{"document_id":"city-42"}]`))}, nil
		}
		return nil, errors.New("hotel provider unavailable")
	})}}
	_, err := api.FetchReportData(context.Background(), "Funchal")
	assert.ErrorContains(t, err, "hotel provider unavailable")
	api.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("location provider unavailable")
	})}
	_, err = api.FetchReportData(context.Background(), "Funchal")
	assert.ErrorContains(t, err, "location provider unavailable")
}
