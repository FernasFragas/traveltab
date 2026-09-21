package httpserver

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/net/html"
	"weatherservice/internal/application"
)

func TestRedesignPlanResponseRoots(t *testing.T) {
	for _, tc := range []struct {
		name             string
		planner          fakeTripPlanner
		overrides        map[string]string
		status           int
		pushURL, message string
	}{
		{name: "success", planner: fakeTripPlanner{build: planWithStays(sixStays, "")}, status: 200, pushURL: "/trip/lisbon-portugal?days=3&from=2026-03-11"},
		{name: "validation", overrides: map[string]string{"days": "9"}, status: 400, pushURL: "/trip/lisbon-portugal", message: badDaysMessage},
		{name: "provider", planner: fakeTripPlanner{err: errors.New("offline")}, status: 500, pushURL: "/trip/lisbon-portugal", message: planFailMessage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := planServer(t, tc.planner)
			response, err := server.app.Test(planRequest(tc.overrides), 5000)
			require.NoError(t, err)
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			assert.Equal(t, tc.status, response.StatusCode)
			assert.Equal(t, tc.pushURL, response.Header.Get("HX-Push-Url"))
			roots := responseRoots(t, string(body))
			require.Len(t, roots, 3, "planner must replace form, itinerary, and stays")
			for i, id := range []string{"trip-card", "itinerary", "stays"} {
				assert.Equal(t, id, nodeAttribute(roots[i], "id"))
				assert.Equal(t, 1, strings.Count(string(body), `id="`+id+`"`))
				if i > 0 {
					assert.Equal(t, "outerHTML", nodeAttribute(roots[i], "hx-swap-oob"))
				}
			}
			if tc.message != "" {
				assert.Contains(t, string(body), tc.message)
				assert.NotContains(t, string(body), "Hotel Alpha")
				assert.NotContains(t, string(body), "Belem Tower")
				assert.NotContains(t, string(body), ".ics?")
			} else {
				assert.Contains(t, string(body), "Hotel Alpha")
				assert.Contains(t, string(body), "Belem Tower")
			}
		})
	}
}

func TestRedesignSearchFailureRetainsDestination(t *testing.T) {
	ctrl := gomock.NewController(t)
	server, weather, _ := createTestServer(ctrl)
	weather.EXPECT().GenerateReport(gomock.Any(), gomock.Any()).Return(nil, errors.New("secret provider detail"))
	req := httptest.NewRequest(http.MethodGet, "/process-form/?city_name=missing", nil)
	req.Header.Set("HX-Request", "true")
	response, err := server.app.Test(req, 5000)
	require.NoError(t, err)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Equal(t, 500, response.StatusCode)
	assert.Equal(t, "#destination-search-error", response.Header.Get("HX-Retarget"))
	assert.Equal(t, "innerHTML", response.Header.Get("HX-Reswap"))
	assert.Contains(t, string(body), "Check the city name and try again")
	assert.NotContains(t, string(body), "secret provider detail")
	assert.NotContains(t, string(body), "content-area")
}

func TestRedesignPresentationAcrossPageRoutes(t *testing.T) {
	ctrl := gomock.NewController(t)
	server, weather, _ := createTestServer(ctrl)
	server.now = func() time.Time { return planNow }
	server.SetTripPlanner(fakeTripPlanner{build: planWithStays(sixStays, "")})
	info := createTestGeneralWeatherInfo()
	info.Country = "PT"
	require.NoError(t, testStore.SaveCityData("Lisbon", map[string]any{
		"GeneralInfo": info, "Videos": application.VideosStream{},
	}))
	weather.EXPECT().GenerateReport(gomock.Any(), gomock.Any()).Return(info, nil).Times(3)
	for _, route := range []string{"/", "/process-form/?city_name=Lisbon", "/trip/lisbon-pt?days=3&from=2026-03-11"} {
		req := httptest.NewRequest(http.MethodGet, route, nil)
		if strings.HasPrefix(route, "/process-form/") {
			req.Header.Set("HX-Request", "true")
		}
		status, body := doRequest(t, server, req)
		require.Equal(t, 200, status, route)
		assert.Contains(t, body, `src="/images/destinations/lisbon-alfama.jpg"`)
		assert.Contains(t, body, "Hillside streets, tiled facades, and life by the river.")
		assert.Contains(t, body, "query=38.722300%2C-9.139300")
		assert.NotContains(t, body, "hx-swap-oob")
		for _, id := range []string{"overview", "itinerary", "stays", "videos"} {
			assert.Equal(t, 1, strings.Count(body, `id="`+id+`"`), route)
		}
		if strings.HasPrefix(route, "/trip/") {
			assert.Contains(t, body, "Hotel Alpha")
		}
	}
}

func responseRoots(t *testing.T, body string) []*html.Node {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(body))
	require.NoError(t, err)
	var roots []*html.Node
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "body" {
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if child.Type == html.ElementNode {
					roots = append(roots, child)
				}
			}
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(doc)
	return roots
}

func nodeAttribute(node *html.Node, key string) string {
	for _, attribute := range node.Attr {
		if attribute.Key == key {
			return attribute.Val
		}
	}
	return ""
}
