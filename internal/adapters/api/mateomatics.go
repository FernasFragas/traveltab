package api

/*
import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"weatherservice/internal/application"
)

const mateoMaticsAuthenticationURL = "https://login.meteomatics.com/api/v1/token"
const mateioMaticsAPIURL = "https://api.meteomatics.com"

type MateoMaticAPI struct {
	username string
	password string
	token    string

	client *http.Client
}

func NewMateoMaticAPI(username, password string) *MateoMaticAPI {
	m := &MateoMaticAPI{
		username: username,
		password: password,
		client:   http.DefaultClient,
	}

	if m.getOAuthToken() != nil {
		return nil
	}

	return m
}

func (api *MateoMaticAPI) getOAuthToken() error {
	if api.client == nil {
		return fmt.Errorf("client not initialized")
	}

	// Encode username and password in Base64 for Basic Auth
	auth := api.username + ":" + api.password
	authEncoded := base64.StdEncoding.EncodeToString([]byte(auth))

	// Create HTTP request
	req, err := http.NewRequest("GET", mateoMaticsAuthenticationURL, nil)
	if err != nil {
		return err
	}

	// Add Authorization header
	req.Header.Set("Authorization", "Basic "+authEncoded)

	// Make HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	// Parse JSON token response
	var result struct {
		AccessToken string `json:"access_token"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return err
	}

	api.token = result.AccessToken

	return nil

}

func (api *MateoMaticAPI) FetchReportData(ctx context.Context, city string) (*application.InformationReporter[application.Waves], error) {
	apiURL, err := api.setupQueryParams(city)
	if err != nil {
		return nil, err
	}

	resp, err := api.client.Get(apiURL)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	return nil, nil
}

func (api *MateoMaticAPI) FetchGeneralInfo(ctx context.Context) (*application.GeneralInfo, error) {
	if api.client == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	// TODO: Implement actual API call
	return &application.GeneralInfo{
		City:    "San Francisco",
		Country: "United States",
		Lon:     122.4194,
		Lat:     37.7749,
	}, nil
}

func (api *MateoMaticAPI) setupQueryParams(city string) (string, error) {
	if api.client == nil {
		return "", fmt.Errorf("client not initialized")
	}

	time := "yesterdayT00:00Z--todayT12:00Z:PT3H"

	format := "json"
	interest := "max_individual_wave_height:m"

	queryParams := url.Values{}
	queryParams.Add("access_token", api.token)
	queryParams.Add("units", "metric")
	queryParams.Add("APPID", api.key)

	apiUrl := fmt.Sprintf("%s/%s/%s/%s/%s?%s", mateioMaticsAPIURL, time, interest, city, format, queryParams.Encode())

	return apiUrl, nil
}*/
