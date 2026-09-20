package application

import "time"

type DailyStats struct {
	Day            string `json:"day"`
	UniqueVisitors int    `json:"unique_visitors"`
	PageViews      int    `json:"page_views"`
}

type CityStats struct {
	City     string `json:"city"`
	Searches int    `json:"searches"`
}

type VisitStats struct {
	Since          time.Time    `json:"since"`
	UniqueVisitors int          `json:"unique_visitors"`
	PageViews      int          `json:"page_views"`
	Searches       int          `json:"searches"`
	Daily          []DailyStats `json:"daily"`
	TopCities      []CityStats  `json:"top_cities"`
}
