package httpserver

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"weatherservice/internal/planner"
	"weatherservice/internal/writermap"
	"weatherservice/internal/writermap/writermaptest"
)

// TestPlanRoute_LinksToItsOwnExportsAndMapsRoutes proves /plan's own response links to the
// export routes with a slug and query the export handlers can actually resolve - not just
// that each piece works in isolation.
func TestPlanRoute_LinksToItsOwnExportsAndMapsRoutes(t *testing.T) {
	server := planServer(t, fakeTripPlanner{})

	status, body := doRequest(t, server, planRequest(nil))
	require.Equal(t, 200, status)

	// html/template correctly escapes "&" to "&amp;" inside an HTML attribute.
	assert.Contains(t, body, `/trip/lisbon-portugal.ics?days=3&amp;from=2026-03-11`)
	assert.Contains(t, body, `/trip/lisbon-portugal.kml?days=3&amp;from=2026-03-11`)
	assert.Contains(t, body, "https://www.google.com/maps/dir/?")
}

// seededPlanner plans like threeDayPlan, with the request's Include places added to day 1, so
// an export shows whether the saved places reached the planner.
func seededPlanner() fakeTripPlanner {
	return fakeTripPlanner{build: func(req planner.Request) *planner.Plan {
		plan := threeDayPlan(req)
		plan.Days[0].Stops = append(plan.Days[0].Stops, req.Include...)
		return plan
	}}
}

func exportServerWithSaves(t *testing.T, ctrl *gomock.Controller, enabled bool) *Server {
	t.Helper()

	server, _ := newExportTestServer(t, ctrl, "lisbon", "pt", seededPlanner())
	server.SetWriterMapSource(&writermaptest.FakeSource{Result: writermap.DestinationResult{Places: []writermap.Place{
		{QID: "Q1", Names: []string{"Saved One"}, Lat: 38.71, Lon: -9.14},
		{QID: "Q2", Names: []string{"Saved Two"}, Lat: 38.72, Lon: -9.15},
	}}}, enabled)

	return server
}

func TestExportURLs_CarryOnlyTheResolvedIncludes(t *testing.T) {
	req := planner.Request{
		City: "Lisbon", Country: "pt", Days: 3, Start: planNow,
		Include: []planner.Place{{ID: "Q1"}, {ID: "Q2"}},
	}

	ics, kml := exportURLs(req)

	assert.Equal(t, "/trip/lisbon-pt.ics?days=3&from=2026-03-10&include=Q1&include=Q2", ics)
	assert.Equal(t, "/trip/lisbon-pt.kml?days=3&from=2026-03-10&include=Q1&include=Q2", kml)
}

func TestExportURLs_WithoutIncludeAreUnchanged(t *testing.T) {
	ics, kml := exportURLs(planner.Request{City: "Lisbon", Country: "pt", Days: 3, Start: planNow})

	assert.Equal(t, "/trip/lisbon-pt.ics?days=3&from=2026-03-10", ics)
	assert.Equal(t, "/trip/lisbon-pt.kml?days=3&from=2026-03-10", kml)
}

func TestPlanRoute_ExportLinksCarryTheResolvedIncludes(t *testing.T) {
	var got planner.Request
	server := includeServer(t, true, &got)

	_, body := doRequest(t, server, includeRequest("Q404", "Q1", "Q2"))

	assert.Contains(t, body, `/trip/lisbon-portugal.ics?days=3&amp;from=2026-03-11&amp;include=Q1&amp;include=Q2"`)
	assert.Contains(t, body, `/trip/lisbon-portugal.kml?days=3&amp;from=2026-03-11&amp;include=Q1&amp;include=Q2"`)
	assert.NotContains(t, body, "Q404&")
}

func TestPlanRoute_PushURLStaysWithoutInclude(t *testing.T) {
	var got planner.Request
	server := includeServer(t, true, &got)

	resp, err := server.app.Test(includeRequest("Q1"), 5000)
	require.NoError(t, err)

	assert.Equal(t, "/trip/lisbon-portugal?days=3&from=2026-03-11", resp.Header.Get("HX-Push-Url"))
}

func TestICSAndKML_WithIncludeContainTheSavedPlaces(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	server := exportServerWithSaves(t, ctrl, true)

	for _, ext := range []string{"ics", "kml"} {
		status, body := doRequest(t, server, httptest.NewRequest("GET", "/trip/lisbon-pt."+ext+"?days=3&from=2026-03-11&include=Q1&include=Q2&include=Q404", nil))

		require.Equal(t, 200, status, ext)
		assert.Contains(t, body, "Saved One", ext)
		assert.Contains(t, body, "Saved Two", ext)
	}
}

func TestICS_WithoutIncludeIsUnchangedByTheSavedPlaces(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	withSaves := exportServerWithSaves(t, ctrl, true)
	plain, _ := newExportTestServer(t, ctrl, "lisbon", "pt", fakeTripPlanner{})

	_, got := doRequest(t, withSaves, httptest.NewRequest("GET", "/trip/lisbon-pt.ics?days=3&from=2026-03-11", nil))
	_, want := doRequest(t, plain, httptest.NewRequest("GET", "/trip/lisbon-pt.ics?days=3&from=2026-03-11", nil))

	assert.Equal(t, want, got)
}

func TestICS_IgnoresIncludeWhenTheWritersMapIsOff(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	server := exportServerWithSaves(t, ctrl, false)

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/trip/lisbon-pt.ics?days=3&from=2026-03-11&include=Q1", nil))

	require.Equal(t, 200, status)
	assert.NotContains(t, body, "Saved One")
}

func TestTripPage_IgnoresInclude(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	var got planner.Request
	server, mockWeather, _ := createTestServer(ctrl)
	server.now = func() time.Time { return planNow }
	server.SetTripPlanner(fakeTripPlanner{build: func(req planner.Request) *planner.Plan {
		got = req
		return threeDayPlan(req)
	}})
	server.SetWriterMapSource(&writermaptest.FakeSource{Result: writermap.DestinationResult{Places: []writermap.Place{
		{QID: "Q1", Lat: 38.71, Lon: -9.14},
	}}}, true)
	seedPlannerCity(t, "Planton")
	mockWeather.EXPECT().GenerateReport(gomock.Any(), "planton, pt").Return(createTestGeneralWeatherInfoFor("Planton"), nil)

	status, _ := doRequest(t, server, httptest.NewRequest("GET", "/trip/planton-pt?days=3&from=2026-03-11&include=Q1", nil))

	require.Equal(t, 200, status)
	assert.Equal(t, 3, got.Days, "the page did plan")
	assert.Empty(t, got.Include)
}
