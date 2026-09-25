package main

import (
	"database/sql"
	"time"
	"weatherservice/internal/application"
)

type silentAnalytics struct{}

func (silentAnalytics) SaveCityData(string, map[string]any) error { return nil }
func (silentAnalytics) GetCityData(string) (string, error) {
	return "", sql.ErrNoRows
}
func (silentAnalytics) RecordSitemapSlug(string) error         { return nil }
func (silentAnalytics) ListSitemapSlugs() ([]string, error)    { return []string{}, nil }
func (silentAnalytics) SaveVisit(string, string, string) error { return nil }
func (silentAnalytics) GetVisitStats(time.Time) (*application.VisitStats, error) {
	return &application.VisitStats{Daily: []application.DailyStats{}, TopCities: []application.CityStats{}}, nil
}
