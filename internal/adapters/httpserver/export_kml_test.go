package httpserver

import (
	"encoding/xml"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestKML_OneFolderPerDay(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, _ := newExportTestServer(t, ctrl, "lisbon", "pt", fakeTripPlanner{})

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/trip/lisbon-pt.kml?days=3&from=2026-03-11", nil))
	require.Equal(t, 200, status)

	// threeDayPlan has 3 days.
	assert.Equal(t, 3, xmlCount(t, body, "Folder"))
	assert.Contains(t, body, "<name>Day 1</name>")
	assert.Contains(t, body, "<name>Day 2</name>")
	assert.Contains(t, body, "<name>Day 3</name>")
}

func TestKML_OnePlacemarkPerStopWithCoordinates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, _ := newExportTestServer(t, ctrl, "lisbon", "pt", fakeTripPlanner{})

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/trip/lisbon-pt.kml?days=3&from=2026-03-11", nil))
	require.Equal(t, 200, status)

	// 2 + 2 + 1 stops across the three days.
	assert.Equal(t, 5, xmlCount(t, body, "Placemark"))
	assert.Contains(t, body, "<name>Belem Tower</name>")
	// KML coordinates are lon,lat[,alt] - the opposite order from planner.Place's Lat, Lon.
	assert.Contains(t, body, "<coordinates>-9.2159,38.6916</coordinates>")
}

func TestKML_IsValidXML(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, _ := newExportTestServer(t, ctrl, "lisbon", "pt", fakeTripPlanner{})

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/trip/lisbon-pt.kml?days=3&from=2026-03-11", nil))
	require.Equal(t, 200, status)

	var doc struct {
		XMLName xml.Name `xml:"kml"`
	}
	require.NoError(t, xml.Unmarshal([]byte(body), &doc), "response is not well-formed XML")
	assert.Equal(t, "kml", doc.XMLName.Local)
}

func TestKML_404sWithoutAFullPlan(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, _ := createTestServer(ctrl)
	server.now = func() time.Time { return planNow }
	server.SetTripPlanner(fakeTripPlanner{})
	mockWeather.EXPECT().GenerateReport(gomock.Any(), gomock.Any()).Return(createTestGeneralWeatherInfoFor("Lisbon"), nil).AnyTimes()

	for _, path := range []string{
		"/trip/lisbon-pt.kml",
		"/trip/lisbon-pt.kml?days=3",
		"/trip/lisbon-pt.kml?days=0&from=2026-03-11",
		"/trip/notaslug.kml?days=3&from=2026-03-11",
	} {
		status, _ := doRequest(t, server, httptest.NewRequest("GET", path, nil))
		assert.Equal(t, 404, status, "expected 404 for %s", path)
	}
}

func TestKML_ContentTypeIsKML(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, _ := newExportTestServer(t, ctrl, "lisbon", "pt", fakeTripPlanner{})

	req := httptest.NewRequest("GET", "/trip/lisbon-pt.kml?days=3&from=2026-03-11", nil)
	resp, err := server.app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "application/vnd.google-earth.kml+xml; charset=utf-8", resp.Header.Get("Content-Type"))
}

// xmlCount counts how many <tag ...> opening elements appear in body. Good enough for these
// tests, since TestKML_IsValidXML separately proves the document is well-formed.
func xmlCount(t *testing.T, body, tag string) int {
	t.Helper()

	decoder := xml.NewDecoder(strings.NewReader(body))
	count := 0
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		if start, ok := tok.(xml.StartElement); ok && start.Name.Local == tag {
			count++
		}
	}
	return count
}
