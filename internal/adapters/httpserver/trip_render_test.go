package httpserver

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"weatherservice/internal/planner"
)

// stayPhotos are the stops of the rendered plan: three with a photo and one without.
var stayPhotos = []planner.Place{
	place("Q193386", "Belem Tower", "Torre de Belem.jpg", 38.6916, -9.2159),
	place("Q208420", "Jeronimos Monastery", "Mosteiro dos Jeronimos.jpg", 38.6979, -9.2065),
	place("Q1770963", "Tile Museum", "Museu do Azulejo.jpg", 38.7254, -9.1146),
	place("Q1543442", "Fado Museum", "", 38.7118, -9.1272),
}

// sixStays is what PickStays returns at most: six named places, some without stars or a website.
var sixStays = []planner.Stay{
	{Name: "Hotel Alpha", Kind: "hotel", Stars: 4, Website: "https://alpha.example", Lat: 38.71, Lon: -9.13},
	{Name: "Hostel Bravo", Kind: "hostel", Lat: 38.712, Lon: -9.131},
	{Name: "Guest House Charlie", Kind: "guest_house", Stars: 3, Website: "https://charlie.example", Lat: 38.713, Lon: -9.132},
	{Name: "Hotel Delta", Kind: "hotel", Stars: 5, Lat: 38.714, Lon: -9.133},
	{Name: "Hotel Echo", Kind: "hotel", Lat: 38.715, Lon: -9.134},
	{Name: "Hostel Foxtrot", Kind: "hostel", Stars: 2, Lat: 38.716, Lon: -9.135},
}

const testBookingURL = "https://www.booking.com/searchresults.html?ss=Lisbon&checkin=2026-03-11&checkout=2026-03-14"

// planWithStays renders a full plan: one day of stops and the places to stay under it.
func planWithStays(stays []planner.Stay, staysNote string) func(planner.Request) *planner.Plan {
	return func(req planner.Request) *planner.Plan {
		return &planner.Plan{
			Request: req,
			Days: []planner.Day{{
				Date:    req.Start,
				Certain: true,
				WalkKM:  2.5,
				Stops:   stayPhotos,
			}},
			Center:     stayPhotos[0],
			Stays:      stays,
			StaysNote:  staysNote,
			BookingURL: testBookingURL,
		}
	}
}

// renderPlan runs the real route and templates and returns the rendered fragment.
func renderPlan(t *testing.T, build func(planner.Request) *planner.Plan) string {
	t.Helper()

	server := planServer(t, fakeTripPlanner{build: build})
	status, body := doRequest(t, server, planRequest(nil))
	require.Equal(t, fiber.StatusOK, status)

	return body
}

func TestTripCard_ListsEveryStay(t *testing.T) {
	body := renderPlan(t, planWithStays(sixStays, ""))

	for _, stay := range sixStays {
		assert.Contains(t, body, stay.Name)
	}

	assert.Equal(t, len(sixStays), countOccurrences(body, `class="stay"`))
	// The icon says what kind of place it is.
	assert.Equal(t, 3, countOccurrences(body, "stay-icon-hotel"))
	assert.Equal(t, 2, countOccurrences(body, "stay-icon-hostel"))
	assert.Equal(t, 1, countOccurrences(body, "stay-icon-guest_house"))
	// Only the places that have one get a website link.
	assert.Contains(t, body, `href="https://alpha.example"`)
	assert.Contains(t, body, `href="https://charlie.example"`)
	assert.Equal(t, 2, countOccurrences(body, "stay-website"))
}

func TestTripCard_ShowsStarsOnlyWhenKnown(t *testing.T) {
	body := renderPlan(t, planWithStays(sixStays, ""))

	// Alpha, Charlie, Delta and Foxtrot have stars; Bravo and Echo don't.
	assert.Equal(t, 4, countOccurrences(body, `class="stay-stars"`))
	for _, stars := range []string{"★ 4", "★ 3", "★ 5", "★ 2"} {
		assert.Contains(t, body, stars)
	}
	assert.NotContains(t, body, "★ 0")
}

func TestTripCard_ShowsTheStaysNoteInsteadOfAList(t *testing.T) {
	body := renderPlan(t, planWithStays(nil, "Places to stay are unavailable right now"))

	assert.Contains(t, body, "Places to stay are unavailable right now")
	assert.NotContains(t, body, `class="stay"`)
}

func TestTripCard_HasACheckPricesLink(t *testing.T) {
	body := renderPlan(t, planWithStays(sixStays, ""))

	assert.Contains(t, body, "Check prices")
	assert.Contains(t, body, "www.booking.com/searchresults.html")
	assert.Contains(t, body, "checkin=2026-03-11")
	assert.Contains(t, body, "checkout=2026-03-14")
}

func TestTripCard_EveryPhotoLinksToItsCommonsPage(t *testing.T) {
	body := renderPlan(t, planWithStays(sixStays, ""))

	withPhotos := 0
	for _, stop := range stayPhotos {
		if stop.Image == "" {
			continue
		}
		withPhotos++

		assert.Contains(t, body, stop.ImageURL(400))
		assert.Contains(t, body, `href="`+stop.ImagePageURL()+`"`)
	}

	// Every photo carries its credit, and the stop without one carries none.
	assert.Equal(t, withPhotos, countOccurrences(body, "Photo: Wikimedia Commons"))
}

func TestTripCard_LabelsStopsByWeatherSuitability(t *testing.T) {
	body := renderPlan(t, func(req planner.Request) *planner.Plan {
		return &planner.Plan{Request: req, Days: []planner.Day{{Date: req.Start, Certain: true, Stops: []planner.Place{
			{Name: "Museum", Kind: planner.Indoor},
			{Name: "Park", Kind: planner.Outdoor},
			{Name: "Palace and gardens", Kind: planner.Mixed},
		}}}}
	})
	for _, label := range []string{"Indoors", "Outdoors", "Indoor &amp; outdoor"} {
		assert.Contains(t, body, label)
	}
}

func TestTripCard_MissingWeatherIsNotShownAsAnUncertainForecast(t *testing.T) {
	for _, forecast := range []*planner.DayForecast{nil, {Date: planNow}} {
		body := renderPlan(t, func(req planner.Request) *planner.Plan {
			return &planner.Plan{Request: req, Days: []planner.Day{{Date: req.Start, Forecast: forecast}}}
		})
		assert.Contains(t, body, "No forecast data for this day")
		assert.NotContains(t, body, "less certain")
	}
}

func TestFooter_ShowsAllFourDataCredits(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server, mockWeather, _ := createTestServer(ctrl)
	server.now = func() time.Time { return planNow }
	seedPlannerCity(t, "Creditville")
	mockWeather.EXPECT().GenerateReport(gomock.Any(), "Creditville").
		Return(createTestGeneralWeatherInfoFor("Creditville"), nil).Times(2)

	// "&" is written as an HTML entity, so that is how it reaches the page.
	credits := []string{
		"Places: Wikipedia &amp; Wikidata",
		"Photos: Wikimedia Commons",
		"Weather: Open-Meteo (CC BY 4.0)",
		"Map data © OpenStreetMap contributors",
	}

	for _, htmx := range []bool{false, true} {
		req := httptest.NewRequest("GET", "/?city_name=Creditville", nil)
		if htmx {
			req.Header.Set("HX-Request", "true")
		}

		status, body := doRequest(t, server, req)
		require.Equal(t, fiber.StatusOK, status)

		for _, credit := range credits {
			assert.Contains(t, body, credit, "htmx=%v", htmx)
		}
	}
}
