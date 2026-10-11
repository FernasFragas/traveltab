package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAmadeus_FetchesTokenAndMapsTheFirstTenHotels(t *testing.T) {
	hotels := make([]Hotel, 11)
	for i := range hotels {
		hotels[i] = Hotel{Name: "Hotel " + string(rune('A'+i)), HotelID: "ID" + string(rune('A'+i))}
	}
	payload, err := json.Marshal(Response{Data: hotels})
	require.NoError(t, err)
	requests := 0
	api := &AmadeusAPI{AmadeusID: "test-id", AmadeusSecret: "test-secret"}
	api.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		switch req.URL.Path {
		case "/v1/security/oauth2/token":
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			form, err := url.ParseQuery(string(body))
			require.NoError(t, err)
			assert.Equal(t, http.MethodPost, req.Method)
			assert.Equal(t, "client_credentials", form.Get("grant_type"))
			assert.Equal(t, "test-id", form.Get("client_id"))
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"access_token":"session-token"}`))}, nil
		case "/v1/reference-data/locations/hotels/by-geocode":
			assert.Equal(t, "Bearer session-token", req.Header.Get("Authorization"))
			assert.Equal(t, "32.76", req.URL.Query().Get("latitude"))
			assert.Equal(t, "-16.96", req.URL.Query().Get("longitude"))
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(payload)))}, nil
		default:
			t.Fatalf("unexpected Amadeus request: %s", req.URL)
			return nil, nil
		}
	})}
	api.token, err = api.fetchAuthToken(context.Background())
	require.NoError(t, err)
	report, err := api.FetchReportData(context.Background(), "32.76,-16.96")
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Len(t, report.Data, 10)
	assert.Equal(t, "Hotel A", report.Data[0].HotelName)
	assert.Equal(t, "IDJ", report.Data[9].HotelURL)
	assert.Equal(t, 2, requests)
}

func TestAmadeus_ReportsInvalidCoordinatesAndTransportErrors(t *testing.T) {
	api := &AmadeusAPI{client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("upstream unavailable")
	})}}
	_, err := api.FetchReportData(context.Background(), "bad,-16.96")
	assert.Error(t, err)
	_, err = api.FetchReportData(context.Background(), "32.76,bad")
	assert.Error(t, err)
	_, err = api.FetchReportData(context.Background(), "32.76,-16.96")
	assert.ErrorContains(t, err, "upstream unavailable")
	_, err = api.fetchAuthToken(context.Background())
	assert.ErrorContains(t, err, "upstream unavailable")
}
