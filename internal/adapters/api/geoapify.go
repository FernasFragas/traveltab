package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"weatherservice/internal/application"
)

const geoapifySearchURL = "https://api.geoapify.com/v2/place-details?"

type GeoapifyAPI struct {
	client *http.Client
	key    string
}

func NewGeoapifyAPI(key string) *GeoapifyAPI {
	return &GeoapifyAPI{
		client: http.DefaultClient,
		key:    key,
	}
}

func (api *GeoapifyAPI) FetchReportData(ctx context.Context, categories ...string) (*application.DataToReport[application.Coordinates], error) {
	if api.client == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	categoriesWithLonLat := strings.Split(categories[0], ",")
	categoriesWithoutLonLat := categoriesWithLonLat[2:]

	categoriesToSearch := api.filterCategories(categoriesWithoutLonLat)

	queryParams := url.Values{}
	queryParams.Add("apiKey", api.key)
	queryParams.Add("lon", categoriesWithLonLat[0])
	queryParams.Add("lat", categoriesWithLonLat[1])
	//queryParams.Add("id", "id%3D514d368a517c511e40594bfd7b574ec84740f00103f90135335d1c00000000920313416e61746f6d697363686573204d757365756d")
	queryParams.Add("features", categoriesToSearch)

	apiUrl := fmt.Sprintf("%s%s", geoapifySearchURL, queryParams.Encode())

	resp, err := api.client.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}
		return nil, fmt.Errorf("failed to fetch data from geoapify: %s, because: %s", resp.Status, string(body))
	}

	var geoapifyResponse GeoapifyResponse
	err = json.NewDecoder(resp.Body).Decode(&geoapifyResponse)
	if err != nil {
		return nil, err
	}

	fmt.Println(geoapifyResponse)

	return &application.DataToReport[application.Coordinates]{
		Data: application.Coordinates{
			CoordinatesWithName: []application.CoordinatesWithName{
				{
					Name:        "Geoapify",
					Coordinates: []interface{}{},
				},
			},
		},
	}, nil

}

func (api *GeoapifyAPI) FetchGeneralInfo(ctx context.Context, city ...string) (*application.DataToReport[application.Coordinates], error) {
	return nil, nil
}

func (api *GeoapifyAPI) filterCategories(categories []string) string {
	filteredCategories := []string{}

	for _, category := range categories {
		// Get individual features for this category
		features := api.availableCategories(category)
		filteredCategories = append(filteredCategories, features...)
	}

	return strings.Join(filteredCategories, ",")
}

func (api *GeoapifyAPI) availableCategories(category string) []string {
	switch category {
	case "restaurants":
		return []string{
			"radius_1000.restaurant",
		}
	case "tourism":
		return []string{
			"radius_1000.tourism",
		}
	case "entertainment":
		return []string{
			"radius_1000.entertainment",
		}
	case "parks":
		return []string{
			"radius_1000.park",
		}
	case "playground":
		return []string{
			"radius_1000.playground",
		}
	default:
		return []string{}
	}
}

type GeoapifyResponse struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
}

type Feature struct {
	Type       string                 `json:"type"`
	Geometry   Geometry               `json:"geometry"`
	Properties Properties             `json:"properties"`
	Bbox       []float64              `json:"bbox,omitempty"`
	Center     []float64              `json:"center,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

type Geometry struct {
	Type        string        `json:"type"`
	Coordinates []interface{} `json:"coordinates"`
}

type Properties struct {
	Datasource    Datasource `json:"datasource"`
	Country       string     `json:"country"`
	CountryCode   string     `json:"country_code"`
	State         string     `json:"state"`
	County        string     `json:"county"`
	City          string     `json:"city"`
	Postcode      string     `json:"postcode"`
	District      string     `json:"district"`
	Suburb        string     `json:"suburb"`
	Street        string     `json:"street"`
	Housenumber   string     `json:"housenumber"`
	Lon           float64    `json:"lon"`
	Lat           float64    `json:"lat"`
	Distance      float64    `json:"distance"`
	ResultType    string     `json:"result_type"`
	Formatted     string     `json:"formatted"`
	AddressLine1  string     `json:"address_line1"`
	AddressLine2  string     `json:"address_line2"`
	Category      string     `json:"category"`
	Timezone      Timezone   `json:"timezone"`
	PlusCode      string     `json:"plus_code"`
	PlusCodeShort string     `json:"plus_code_short"`
	Rank          Rank       `json:"rank"`
	PlaceID       string     `json:"place_id"`
	Bbox          []float64  `json:"bbox"`
	StreetNumber  string     `json:"street_number"`
	HouseNumber   string     `json:"house_number"`
	Road          string     `json:"road"`
	Neighbourhood string     `json:"neighbourhood"`
	Quarter       string     `json:"quarter"`
	Hamlet        string     `json:"hamlet"`
	Village       string     `json:"village"`
	Town          string     `json:"town"`
	Municipality  string     `json:"municipality"`
	CityDistrict  string     `json:"city_district"`
	StateDistrict string     `json:"state_district"`
	ISO31662      string     `json:"ISO3166-2"`
	StateCode     string     `json:"state_code"`
}

type Datasource struct {
	Sourcename  string `json:"sourcename"`
	Attribution string `json:"attribution"`
	License     string `json:"license"`
	URL         string `json:"url"`
	Raw         Raw    `json:"raw"`
}

type Raw struct {
	Name          string `json:"name"`
	Street        string `json:"street"`
	City          string `json:"city"`
	State         string `json:"state"`
	Country       string `json:"country"`
	CountryCode   string `json:"country_code"`
	Postcode      string `json:"postcode"`
	District      string `json:"district"`
	Suburb        string `json:"suburb"`
	Housenumber   string `json:"housenumber"`
	Lon           string `json:"lon"`
	Lat           string `json:"lat"`
	Formatted     string `json:"formatted"`
	AddressLine1  string `json:"address_line1"`
	AddressLine2  string `json:"address_line2"`
	Category      string `json:"category"`
	ResultType    string `json:"result_type"`
	Rank          Rank   `json:"rank"`
	PlaceID       string `json:"place_id"`
	Bbox          Bbox   `json:"bbox"`
	StreetNumber  string `json:"street_number"`
	HouseNumber   string `json:"house_number"`
	Road          string `json:"road"`
	Neighbourhood string `json:"neighbourhood"`
	Quarter       string `json:"quarter"`
	Hamlet        string `json:"hamlet"`
	Village       string `json:"village"`
	Town          string `json:"town"`
	Municipality  string `json:"municipality"`
	CityDistrict  string `json:"city_district"`
	StateDistrict string `json:"state_district"`
	ISO31662      string `json:"ISO3166-2"`
	StateCode     string `json:"state_code"`
}

type Timezone struct {
	Name             string `json:"name"`
	OffsetSTD        string `json:"offset_STD"`
	OffsetSTDSeconds int    `json:"offset_STD_seconds"`
	OffsetDST        string `json:"offset_DST"`
	OffsetDSTSeconds int    `json:"offset_DST_seconds"`
	AbbreviationSTD  string `json:"abbreviation_STD"`
	AbbreviationDST  string `json:"abbreviation_DST"`
}

type Rank struct {
	Importance          float64 `json:"importance"`
	Popularity          float64 `json:"popularity"`
	Confidence          float64 `json:"confidence"`
	ConfidenceCityLevel float64 `json:"confidence_city_level"`
	MatchType           string  `json:"match_type"`
}

type Bbox struct {
	Lon1 float64 `json:"lon1"`
	Lat1 float64 `json:"lat1"`
	Lon2 float64 `json:"lon2"`
	Lat2 float64 `json:"lat2"`
}
