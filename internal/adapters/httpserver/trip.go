package httpserver

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"
	"weatherservice/internal/application"

	"github.com/gofiber/fiber/v2"

	"weatherservice/internal/planner"
)

const (
	tripDateLayout  = "2006-01-02"
	tripDateHeading = "Mon, 2 Jan"
	defaultTripDays = 3
)

// The card speaks to travellers, so every problem has a plain message.
const (
	badDaysMessage   = "Pick between 1 and 5 days."
	badStartMessage  = "Pick a start date, like 2026-03-11."
	pastStartMessage = "That start date is in the past. Pick today or a later day."
	noCityMessage    = "Search for a city first, then plan your trip."
	planFailMessage  = "We could not plan your trip right now. Please try again in a moment."
)

// SetTripPlanner switches the planner card on. Without it the card never renders.
func (s *Server) SetTripPlanner(tripPlanner application.TripPlanner) {
	s.tripPlanner = tripPlanner
}

// tripCard is everything views/trip_card.go.tpl needs: the form, and the plan when there is one.
type tripCard struct {
	City       string
	Country    string
	Lat, Lon   float64
	Start      string
	MinStart   string
	Days       int
	DayOptions []int
	Error      string
	Plan       *tripPlanView
}

type tripPlanView struct {
	Note       string
	Days       []tripDayView
	Stays      []planner.Stay
	StaysNote  string
	BookingURL string
}

type tripDayView struct {
	Number      int
	Date        string
	Rainy       bool
	RainMM      string
	Certain     bool
	HasForecast bool
	WalkKM      string
	Stops       []planner.Place
}

// planTrip answers GET /plan with the trip card fragment.
func (s *Server) planTrip(ctx *fiber.Ctx) error {
	if s.tripPlanner == nil {
		return ctx.SendStatus(fiber.StatusNotFound)
	}

	req, problem := s.parsePlanRequest(ctx)
	card := s.newTripCard(req)

	if problem != "" {
		card.Error = problem
		ctx.Status(fiber.StatusBadRequest)
		return ctx.Render("trip_card", card)
	}

	plan, err := s.tripPlanner.Plan(ctx.Context(), req)
	if err != nil || plan == nil {
		log.Printf("Error planning a trip for %s with error %v", req.City, err)
		card.Error = planFailMessage
		ctx.Status(fiber.StatusInternalServerError)
		return ctx.Render("trip_card", card)
	}

	card.Plan = newTripPlanView(plan)

	return ctx.Render("trip_card", card)
}

// parsePlanRequest reads the query into a request, or returns the message to show instead.
func (s *Server) parsePlanRequest(ctx *fiber.Ctx) (planner.Request, string) {
	req := planner.Request{City: strings.TrimSpace(ctx.Query("city")), Country: strings.TrimSpace(ctx.Query("country"))}

	lat, latErr := strconv.ParseFloat(ctx.Query("lat"), 64)
	lon, lonErr := strconv.ParseFloat(ctx.Query("lon"), 64)
	if req.City == "" || latErr != nil || lonErr != nil ||
		math.IsNaN(lat) || math.IsInf(lat, 0) || lat < -90 || lat > 90 ||
		math.IsNaN(lon) || math.IsInf(lon, 0) || lon < -180 || lon > 180 {
		return req, noCityMessage
	}
	req.Lat, req.Lon = lat, lon

	days, err := strconv.Atoi(ctx.Query("days"))
	if err != nil || days < 1 || days > planner.MaxDays {
		return req, badDaysMessage
	}
	req.Days = days

	start, err := time.ParseInLocation(tripDateLayout, ctx.Query("start"), time.UTC)
	if err != nil {
		return req, badStartMessage
	}
	if start.Before(s.today()) {
		return req, pastStartMessage
	}
	req.Start = start

	return req, ""
}

// today is midnight on the server's clock, so date tests stay deterministic.
func (s *Server) today() time.Time {
	now := time.Now
	if s.now != nil {
		now = s.now
	}

	t := now().UTC()

	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// tripCardFor builds the card shown under the weather, or nil when no planner is set.
func (s *Server) tripCardFor(info application.GeneralWeatherInfo) any {
	if s.tripPlanner == nil {
		return nil
	}

	return s.newTripCard(planner.Request{City: info.City, Country: info.Country, Lat: info.Lat, Lon: info.Lon})
}

// newTripCard fills the form, defaulting to three days starting tomorrow.
func (s *Server) newTripCard(req planner.Request) tripCard {
	today := s.today()

	start := req.Start
	if start.IsZero() {
		start = today.AddDate(0, 0, 1)
	}

	days := req.Days
	if days < 1 || days > planner.MaxDays {
		days = defaultTripDays
	}

	options := make([]int, 0, planner.MaxDays)
	for day := 1; day <= planner.MaxDays; day++ {
		options = append(options, day)
	}

	return tripCard{
		City:       req.City,
		Country:    req.Country,
		Lat:        req.Lat,
		Lon:        req.Lon,
		Start:      start.Format(tripDateLayout),
		MinStart:   today.Format(tripDateLayout),
		Days:       days,
		DayOptions: options,
	}
}

// newTripPlanView formats a plan for the template, which then only prints strings.
func newTripPlanView(plan *planner.Plan) *tripPlanView {
	view := &tripPlanView{
		Note:       plan.Note,
		Stays:      plan.Stays,
		StaysNote:  plan.StaysNote,
		BookingURL: plan.BookingURL,
		Days:       make([]tripDayView, 0, len(plan.Days)),
	}

	for i, day := range plan.Days {
		view.Days = append(view.Days, tripDayView{
			Number:      i + 1,
			Date:        day.Date.Format(tripDateHeading),
			Rainy:       day.Rainy,
			RainMM:      rainBadge(day),
			Certain:     day.Certain,
			HasForecast: day.Forecast != nil && day.Forecast.RainMM != nil,
			WalkKM:      fmt.Sprintf("%.1f", day.WalkKM),
			Stops:       day.Stops,
		})
	}

	return view
}

// rainBadge says how much rain the forecast expects, or nothing when it has no value.
func rainBadge(day planner.Day) string {
	if day.Forecast == nil || day.Forecast.RainMM == nil {
		return ""
	}

	switch mm := *day.Forecast.RainMM; {
	case mm < 0.1:
		return ""
	case mm < 1:
		return "under 1 mm"
	default:
		return fmt.Sprintf("%.0f mm", mm)
	}
}
