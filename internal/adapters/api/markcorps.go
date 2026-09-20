package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"weatherservice/internal/application"
)

const markcorpsHotelsCityIDsAPIURL = "https://api.makcorps.com/mapping?"
const markcorpsHotelsAPIURL = "https://api.makcorps.com/city?"

type MarkcorpsAPI struct {
	client *http.Client

	apiKey string
}

func NewMarkcorpsAPI(apiKey string) *MarkcorpsAPI {
	return &MarkcorpsAPI{
		client: http.DefaultClient,
		apiKey: apiKey,
	}
}

func (api *MarkcorpsAPI) FetchReportData(ctx context.Context, city ...string) (*application.DataToReport[application.Hotels], error) {
	if api.client == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	locations, err := api.fetchLocationData(ctx, city[0])
	if err != nil {
		return nil, err
	}

	c := (*locations)[0]

	apiUrl, err := api.setupQueryParams(map[string]string{
		"name":     c.DocumentID,
		"cur":      "EUR",
		"rooms":    "1",
		"adults":   "2",
		"checkin":  "2025-12-25",
		"checkout": "2025-12-26",
	})
	if err != nil {
		return nil, err
	}

	resp, err := api.client.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var location apiHotels

	err = json.NewDecoder(resp.Body).Decode(&location)
	if err != nil {
		return nil, err
	}

	hotels := make(application.Hotels, len(location))
	for i, hotel := range location {
		hotels[i] = application.Hotel{
			HotelName: hotel.Name,
			HotelURL:  hotel.Telephone,
		}
	}

	return &application.DataToReport[application.Hotels]{
		Data: hotels,
	}, nil
}

func (api *MarkcorpsAPI) fetchLocationData(ctx context.Context, city string) (*[]location, error) {
	if api.client == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	apiUrl, err := api.setupQueryParams(map[string]string{
		"name": city,
	})
	if err != nil {
		return nil, err
	}

	resp, err := api.client.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var location []location

	err = json.NewDecoder(resp.Body).Decode(&location)
	if err != nil {
		return nil, err
	}

	return &location, nil
}

func (api *MarkcorpsAPI) setupQueryParams(paramsToPass map[string]string) (string, error) {
	params := url.Values{}
	params.Add("api_key", api.apiKey)
	for key, value := range paramsToPass {
		params.Add(key, value)
	}

	var apiUrl string
	if len(paramsToPass) == 1 {
		apiUrl = fmt.Sprintf("%s%s", markcorpsHotelsCityIDsAPIURL, params.Encode())
	} else {
		apiUrl = fmt.Sprintf("%s%s", markcorpsHotelsAPIURL, params.Encode())
	}

	return apiUrl, nil
}

func (api *MarkcorpsAPI) FetchGeneralInfo(ctx context.Context, city string) (*application.DataToReport[application.Hotels], error) {
	if api.client == nil {
		return nil, fmt.Errorf("client not initialized")
	}
	return nil, nil
}

// location represents the main location data structure
type location struct {
	LookbackServlet interface{} `json:"lookbackServlet"`
	Autobroadened   string      `json:"autobroadened"`
	Title           string      `json:"title"`
	Type            string      `json:"type"`
	DocumentID      string      `json:"document_id"`
	Scope           string      `json:"scope"`
	Name            string      `json:"name"`
	DataType        string      `json:"data_type"`
	Details         details     `json:"details"`
	Value           int         `json:"value"`
	Coords          string      `json:"coords"`
}

// details contains the detailed information about a location
type details struct {
	Placetype            int    `json:"placetype"`
	ParentName           string `json:"parent_name"`
	Address              string `json:"address,omitempty"`
	GrandparentName      string `json:"grandparent_name"`
	GrandparentID        int    `json:"grandparent_id"`
	ParentID             int    `json:"parent_id"`
	GrandparentPlaceType int    `json:"grandparent_place_type"`
	HighlightedName      string `json:"highlighted_name"`
	Name                 string `json:"name"`
	ParentPlaceType      int    `json:"parent_place_type"`
	ParentIDs            []int  `json:"parent_ids"`
	GeoName              string `json:"geo_name"`
	RacEnabled           bool   `json:"rac_enabled,omitempty"`
}

type apiHotels []hotel

type geocode struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type reviews struct {
	Rating float64 `json:"rating"`
	Count  int     `json:"count"`
}

type hotel struct {
	Geocode   geocode `json:"geocode"`
	Telephone string  `json:"telephone"`
	Name      string  `json:"name"`
	HotelID   int     `json:"hotelId"`
	Reviews   reviews `json:"reviews"`
	Vendor1   string  `json:"vendor1"`
	Price1    string  `json:"price1"`
}
