package httpserver

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"weatherservice/internal/planner"
)

// newExportTestServer builds a server on the fixed clock, wired to answer a weather lookup for
// the given city/country (as ParseSlug would produce them: lower-case) and to plan with
// tripPlanner. Exports never touch videos or the city_data cache, so only the weather mock is
// primed.
func newExportTestServer(t *testing.T, ctrl *gomock.Controller, city, country string, tripPlanner fakeTripPlanner) (*Server, *MockWeatherReporter) {
	t.Helper()

	server, mockWeather, _ := createTestServer(ctrl)
	server.now = func() time.Time { return planNow }
	server.SetTripPlanner(tripPlanner)
	mockWeather.EXPECT().GenerateReport(gomock.Any(), city+", "+country).
		Return(createTestGeneralWeatherInfoFor(city), nil).AnyTimes()

	return server, mockWeather
}

func TestICS_OneEventPerStopAtTheDefaultTimes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, _ := newExportTestServer(t, ctrl, "lisbon", "pt", fakeTripPlanner{})

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/trip/lisbon-pt.ics?days=3&from=2026-03-11", nil))
	require.Equal(t, 200, status)

	// threeDayPlan gives 2, 2 and 1 stops across its three days: 5 stops, 5 VEVENTs.
	assert.Equal(t, 5, strings.Count(body, "BEGIN:VEVENT"))
	assert.Equal(t, 5, strings.Count(body, "END:VEVENT"))
	assert.Contains(t, body, "SUMMARY:Belem Tower")
	assert.Contains(t, body, "SUMMARY:Jeronimos Monastery")
	assert.Contains(t, body, "SUMMARY:Castle of Sao Jorge")

	// Day 1 (dry, req.Start = 2026-03-11) starts its first stop at 10:00 and its second at
	// 12:00 local time. threeDayPlan's days carry no explicit location, so they default to
	// UTC, and 10:00 local is 10:00 UTC.
	assert.Contains(t, body, "DTSTART:20260311T100000Z")
	assert.Contains(t, body, "DTSTART:20260311T120000Z")
}

func TestICS_UsesTheForecastTimeZone(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	server, _ := newExportTestServer(t, ctrl, "newyork", "us", fakeTripPlanner{build: func(req planner.Request) *planner.Plan {
		// The fake ignores req.Start and returns a January date instead, firmly outside any
		// US DST transition, so this test doesn't depend on knowing 2026's exact "spring
		// forward" day. The query's own from= only needs to be valid (not in the past) for
		// parseDaysAndStart to accept the request at all; planNow is 2026-03-10.
		date := time.Date(2026, 1, 15, 0, 0, 0, 0, loc)
		return &planner.Plan{
			Request: req,
			Days: []planner.Day{{
				Date:  date,
				Stops: []planner.Place{place("Q1", "Central Park", "", 40.7829, -73.9654)},
			}},
		}
	}})

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/trip/newyork-us.ics?days=1&from=2026-03-11", nil))
	require.Equal(t, 200, status)

	// 10:00 America/New_York on 2026-01-15 (EST, UTC-5) is 15:00 UTC.
	assert.Contains(t, body, "DTSTART:20260115T150000Z")
}

func TestICS_EscapesCommasAndSemicolonsInPlaceNames(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, _ := newExportTestServer(t, ctrl, "lisbon", "pt", fakeTripPlanner{build: func(req planner.Request) *planner.Plan {
		return &planner.Plan{
			Request: req,
			Days: []planner.Day{{
				Date:  req.Start,
				Stops: []planner.Place{place("Q1", "Monastery, Hall; Garden\\Path", "", 38.7, -9.1)},
			}},
		}
	}})

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/trip/lisbon-pt.ics?days=1&from=2026-03-11", nil))
	require.Equal(t, 200, status)
	assert.Contains(t, body, `SUMMARY:Monastery\, Hall\; Garden\\Path`)
}

func TestICS_WrapsLongLinesPerRFC5545(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	longName := strings.Repeat("A very long place name that keeps going ", 4)
	server, _ := newExportTestServer(t, ctrl, "lisbon", "pt", fakeTripPlanner{build: func(req planner.Request) *planner.Plan {
		return &planner.Plan{
			Request: req,
			Days: []planner.Day{{
				Date:  req.Start,
				Stops: []planner.Place{place("Q1", longName, "", 38.7, -9.1)},
			}},
		}
	}})

	status, body := doRequest(t, server, httptest.NewRequest("GET", "/trip/lisbon-pt.ics?days=1&from=2026-03-11", nil))
	require.Equal(t, 200, status)

	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		assert.LessOrEqual(t, len(line), 75, "line exceeds the 75-octet RFC 5545 limit: %q", line)
	}
	// A folded continuation line starts with a single space.
	assert.Contains(t, body, "\r\n ")
}

func TestICS_404sWithoutAFullPlan(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, _ := createTestServer(ctrl)
	server.now = func() time.Time { return planNow }
	server.SetTripPlanner(fakeTripPlanner{})
	mockWeather.EXPECT().GenerateReport(gomock.Any(), gomock.Any()).Return(createTestGeneralWeatherInfoFor("Lisbon"), nil).AnyTimes()

	for _, path := range []string{
		"/trip/lisbon-pt.ics",                        // no days/from at all
		"/trip/lisbon-pt.ics?days=3",                 // missing from
		"/trip/lisbon-pt.ics?days=0&from=2026-03-11", // invalid days
		"/trip/notaslug.ics?days=3&from=2026-03-11",  // unparsable slug
	} {
		status, _ := doRequest(t, server, httptest.NewRequest("GET", path, nil))
		assert.Equal(t, 404, status, "expected 404 for %s", path)
	}
}

func TestICS_ContentTypeIsTextCalendar(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, _ := newExportTestServer(t, ctrl, "lisbon", "pt", fakeTripPlanner{})

	req := httptest.NewRequest("GET", "/trip/lisbon-pt.ics?days=3&from=2026-03-11", nil)
	resp, err := server.app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, "text/calendar; charset=utf-8", resp.Header.Get("Content-Type"))
}
