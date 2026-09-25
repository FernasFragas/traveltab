package httpserver

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"weatherservice/internal/application"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// cacheResponseStorage is immutable so background cache saves cannot change the
// response under test or outlive a mock expectation. No production database is used.
type cacheResponseStorage struct {
	raw string
	err error
}

func (s cacheResponseStorage) GetCityData(city string) (string, error) {
	return s.raw, s.err
}

func (cacheResponseStorage) SaveCityData(string, map[string]any) error { return nil }
func (cacheResponseStorage) RecordSitemapSlug(string) error            { return nil }
func (cacheResponseStorage) ListSitemapSlugs() ([]string, error)       { return nil, nil }
func (cacheResponseStorage) SaveVisit(string, string, string) error    { return nil }
func (cacheResponseStorage) GetVisitStats(time.Time) (*application.VisitStats, error) {
	return &application.VisitStats{}, nil
}

func TestCheckDatabase_RejectsFailedReads(t *testing.T) {
	storageFailure := errors.New("storage unavailable")
	for _, tc := range []struct {
		name string
		err  error
		want error
	}{
		{"miss", sql.ErrNoRows, sql.ErrNoRows},
		{"wrapped miss", fmt.Errorf("lookup: %w", sql.ErrNoRows), sql.ErrNoRows},
		{"storage failure", storageFailure, storageFailure},
		{"wrapped storage failure", fmt.Errorf("lookup: %w", storageFailure), storageFailure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := &Server{storage: cacheResponseStorage{raw: `{"GeneralInfo":{"City":"Stale"}}`, err: tc.err}}
			data, err := server.checkDatabase("Cachetown")
			require.ErrorIs(t, err, tc.want)
			assert.Equal(t, TemplateData{}, data)
		})
	}
}

func TestCheckDatabase_RejectsInvalidJSONWithoutPartialData(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
	}{
		{"syntax error", `{"GeneralInfo":`},
		{"partial decode", `{"GeneralInfo":{"City":"Stale"},"Videos":"not a video list"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := &Server{storage: cacheResponseStorage{raw: tc.raw}}
			data, err := server.checkDatabase("Cachetown")
			require.Error(t, err)
			assert.Equal(t, TemplateData{}, data)
			var syntaxErr *json.SyntaxError
			var typeErr *json.UnmarshalTypeError
			assert.True(t, errors.As(err, &syntaxErr) || errors.As(err, &typeErr), "decoding error must remain inspectable")
		})
	}
}

func TestDestinationRoutes_CacheFallback(t *testing.T) {
	info := createTestGeneralWeatherInfoFor("Cachetown")
	info.Country = "pt"
	cached, err := json.Marshal(TemplateData{
		GeneralInfo: *info,
		Videos:      application.VideosStream{{Title: "Cached travel video", VideoID: "cached12345"}},
	})
	require.NoError(t, err)

	for _, route := range []struct {
		name    string
		path    string
		query   string
		htmx    bool
		planned bool
	}{
		{name: "destination", path: "/?city_name=Cachetown", query: "Cachetown"},
		{name: "destination fragment", path: "/process-form/?city_name=Cachetown", query: "Cachetown", htmx: true},
		{name: "shared trip", path: "/trip/cachetown-pt?days=3&from=2026-03-11", query: "cachetown, pt", planned: true},
	} {
		for _, tc := range []struct {
			name string
			raw  string
			err  error
			hit  bool
		}{
			{name: "miss", err: sql.ErrNoRows},
			{name: "storage failure", err: errors.New("storage unavailable")},
			{name: "malformed JSON", raw: `{"GeneralInfo":`},
			{name: "partially decoded JSON", raw: `{"GeneralInfo":{"City":"Stale"},"Videos":"invalid"}`},
			{name: "valid cache hit", raw: string(cached), hit: true},
		} {
			t.Run(route.name+"/"+tc.name, func(t *testing.T) {
				ctrl := gomock.NewController(t)
				weather := NewMockWeatherReporter(ctrl)
				videos := NewMockVideoStreamReporter(ctrl)
				weather.EXPECT().GenerateReport(gomock.Any(), route.query).Return(info, nil)
				wantVideo := "Cached travel video"
				if !tc.hit {
					videos.EXPECT().GenerateReport(gomock.Any(), "Turistic places in Cachetown, pt").Return(createTestVideosStream(), nil)
					wantVideo = "Test Video 1"
				}
				server := NewAppServer(weather, videos, cacheResponseStorage{raw: tc.raw, err: tc.err})
				server.now = func() time.Time { return planNow }
				server.SetTripPlanner(fakeTripPlanner{})
				req := httptest.NewRequest(http.MethodGet, route.path, nil)
				if route.htmx {
					req.Header.Set("HX-Request", "true")
				}
				status, body := doRequest(t, server, req)
				require.Equal(t, http.StatusOK, status)
				assert.True(t, strings.Contains(body, `name="city" value="Cachetown"`), "planner must contain the actual city")
				assert.True(t, strings.Contains(body, `name="country" value="pt"`), "planner must contain the actual country")
				assert.True(t, strings.Contains(body, wantVideo), "destination must contain video %q", wantVideo)
				assert.False(t, strings.Contains(body, "Stale"), "partially decoded cache data must not render")
				if route.planned {
					assert.Contains(t, body, "Day 1")
					assert.Contains(t, body, "Belem Tower")
				}
			})
		}
	}
}
