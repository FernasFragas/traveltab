package planner

import (
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	stayRadiusKM = StayRadiusM / 1000.0

	// MaxTravelers is the most the plan form offers; DefaultTravelers is used when a request
	// carries none, or one outside 1..MaxTravelers.
	MaxTravelers     = 8
	DefaultTravelers = 2
)

// PickStays keeps the named places to stay within StayRadiusM of the plan center. The ones a
// traveller can look up come first: a website, then stars, then distance, then name.
func PickStays(center Place, stays []Stay) []Stay {
	type nearby struct {
		stay Stay
		km   float64
	}
	kept := make([]nearby, 0, len(stays))
	for _, s := range stays {
		km := DistanceKM(center, Place{Lat: s.Lat, Lon: s.Lon})
		if strings.TrimSpace(s.Name) == "" || km > stayRadiusKM {
			continue
		}
		kept = append(kept, nearby{stay: s, km: km})
	}
	sort.SliceStable(kept, func(i, j int) bool {
		a, b := kept[i], kept[j]
		switch {
		case (a.stay.Website != "") != (b.stay.Website != ""):
			return a.stay.Website != ""
		case a.stay.Stars != b.stay.Stars:
			return a.stay.Stars > b.stay.Stars
		case a.km != b.km:
			return a.km < b.km
		default:
			return a.stay.Name < b.stay.Name
		}
	})
	if len(kept) > MaxStays {
		kept = kept[:MaxStays]
	}
	out := make([]Stay, len(kept))
	for i := range kept {
		out[i] = kept[i].stay
	}
	return out
}

// BookingURL builds a Booking.com search for the city with the trip's dates and number of
// adults filled in. travelers outside 1..MaxTravelers count as DefaultTravelers.
func BookingURL(city, country string, start time.Time, days, travelers int) string {
	if travelers < 1 || travelers > MaxTravelers {
		travelers = DefaultTravelers
	}

	where := make([]string, 0, 2)
	for _, part := range []string{city, country} {
		if part != "" {
			where = append(where, part)
		}
	}
	return "https://www.booking.com/searchresults.html?ss=" + url.QueryEscape(strings.Join(where, ", ")) +
		"&checkin=" + start.Format(time.DateOnly) +
		"&checkout=" + start.AddDate(0, 0, days).Format(time.DateOnly) +
		"&group_adults=" + strconv.Itoa(travelers)
}
