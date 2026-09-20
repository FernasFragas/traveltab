package planner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stayCenter is the plan center in every PickStays test. 0.001 degrees of latitude is about 111 m.
var stayCenter = Place{ID: "Q1", Name: "Castle", Lat: 38.71, Lon: -9.14}

func stayNames(stays []Stay) []string {
	out := make([]string, len(stays))
	for i := range stays {
		out[i] = stays[i].Name
	}
	return out
}

func TestPickStays_DropsPlacesFartherThan1Km(t *testing.T) {
	near := Stay{Name: "Near Hotel", Kind: "hotel", Lat: 38.714, Lon: -9.14}
	far := Stay{Name: "Far Hotel", Kind: "hotel", Lat: 38.73, Lon: -9.14}
	assert.Equal(t, []string{"Near Hotel"}, stayNames(PickStays(stayCenter, []Stay{near, far})))
}

func TestPickStays_DropsUnnamedPlaces(t *testing.T) {
	named := Stay{Name: "Named Hostel", Kind: "hostel", Lat: 38.71, Lon: -9.14}
	unnamed := Stay{Kind: "hostel", Lat: 38.71, Lon: -9.14}
	assert.Equal(t, []string{"Named Hostel"}, stayNames(PickStays(stayCenter, []Stay{unnamed, named})))
}

func TestPickStays_PutsPlacesWithAWebsiteFirst(t *testing.T) {
	// The hotel without a website wins on every other key: more stars, nearer, earlier name.
	noSite := Stay{Name: "Alfa Hotel", Kind: "hotel", Lat: 38.71, Lon: -9.14, Stars: 5}
	withSite := Stay{Name: "Zulu Hostel", Kind: "hostel", Lat: 38.715, Lon: -9.14, Website: "https://zulu.example"}
	assert.Equal(t, []string{"Zulu Hostel", "Alfa Hotel"}, stayNames(PickStays(stayCenter, []Stay{noSite, withSite})))
}

func TestPickStays_ThenSortsByStars(t *testing.T) {
	fewer := Stay{Name: "Alfa Hotel", Kind: "hotel", Lat: 38.71, Lon: -9.14, Website: "https://alfa.example", Stars: 2}
	more := Stay{Name: "Zulu Hotel", Kind: "hotel", Lat: 38.715, Lon: -9.14, Website: "https://zulu.example", Stars: 4}
	assert.Equal(t, []string{"Zulu Hotel", "Alfa Hotel"}, stayNames(PickStays(stayCenter, []Stay{fewer, more})))
}

func TestPickStays_ThenSortsByDistance(t *testing.T) {
	farther := Stay{Name: "Alfa Hotel", Kind: "hotel", Lat: 38.718, Lon: -9.14, Website: "https://alfa.example", Stars: 3}
	nearer := Stay{Name: "Zulu Hotel", Kind: "hotel", Lat: 38.711, Lon: -9.14, Website: "https://zulu.example", Stars: 3}
	assert.Equal(t, []string{"Zulu Hotel", "Alfa Hotel"}, stayNames(PickStays(stayCenter, []Stay{farther, nearer})))
}

func TestPickStays_ReturnsAtMostSix(t *testing.T) {
	var stays []Stay
	for i := 0; i < 8; i++ {
		stays = append(stays, Stay{Name: string(rune('A'+i)) + " Hotel", Kind: "hotel", Lat: 38.71 + float64(i)/10000, Lon: -9.14})
	}
	assert.Len(t, PickStays(stayCenter, stays), MaxStays)
}

func TestBookingURL_CheckoutIsStartPlusDays(t *testing.T) {
	got := BookingURL("Lisbon", "Portugal", time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), 3)
	assert.Equal(t, "https://www.booking.com/searchresults.html?ss=Lisbon%2C+Portugal&checkin=2026-03-01&checkout=2026-03-04", got)
}

func TestBookingURL_EscapesCityAndCountry(t *testing.T) {
	got := BookingURL("São Paulo", "Brazil", time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), 1)
	require.Contains(t, got, "ss=S%C3%A3o+Paulo%2C+Brazil")
	assert.Equal(t, "https://www.booking.com/searchresults.html?ss=S%C3%A3o+Paulo%2C+Brazil&checkin=2026-03-01&checkout=2026-03-02", got)
}
