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

	// tooManyNewCitiesMessage is shown instead of a plan when a burst of never-before-seen
	// cities has used up this minute's share of /trip/:slug's automatic planning.
	tooManyNewCitiesMessage = "This city is new to us and we're planning several new places right now. Press \"Plan my trip\" in a moment."
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
	ICSURL     string // "" when the plan can't be identified by a slug (e.g. no country)
	KMLURL     string
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
	MapsURL     string // a walking-directions link through the day's stops, "" with none
}

// planTrip answers GET /plan with the trip card fragment. It also carries an HX-Push-Url
// header with the shareable /trip/:slug link for whatever was just planned, which HTMX reads
// on its own and uses to update the browser's address bar - no JavaScript of ours involved.
func (s *Server) planTrip(ctx *fiber.Ctx) error {
	if s.tripPlanner == nil {
		return ctx.SendStatus(fiber.StatusNotFound)
	}

	req, problem := s.parsePlanRequest(ctx)
	card := s.newTripCard(req)

	if problem != "" {
		card.Error = problem
		s.setPushURL(ctx, req, false)
		ctx.Status(fiber.StatusBadRequest)
		return ctx.Render("trip_card", card)
	}

	plan, err := s.tripPlanner.Plan(ctx.Context(), req)
	if err != nil || plan == nil {
		log.Printf("Error planning a trip for %s with error %v", req.City, err)
		card.Error = planFailMessage
		s.setPushURL(ctx, req, false)
		ctx.Status(fiber.StatusInternalServerError)
		return ctx.Render("trip_card", card)
	}

	card.Plan = newTripPlanView(plan)
	s.setPushURL(ctx, req, true)

	return ctx.Render("trip_card", card)
}

// setPushURL points the browser's address bar at the canonical /trip/:slug link for req, with
// the days/from query only when withDates is true. It sets nothing when req doesn't even
// identify a city, since there is no shareable page to point at.
func (s *Server) setPushURL(ctx *fiber.Ctx, req planner.Request, withDates bool) {
	if req.City == "" || req.Country == "" {
		return
	}

	url := "/trip/" + Slug(req.City, req.Country)
	if withDates && !req.Start.IsZero() && req.Days > 0 {
		url += fmt.Sprintf("?days=%d&from=%s", req.Days, req.Start.Format(tripDateLayout))
	}
	ctx.Set("HX-Push-Url", url)
}

// tripPage answers GET /trip/:slug: it works like the home page, but the slug fills the search
// instead of a form post, and when ?days=N&from=YYYY-MM-DD are both present and valid, the
// plan is already in the page on first load - no click needed. An unparsable slug is a 404; an
// unseen city behind a well-formed slug goes through the same fetch-then-cache path "/" uses.
func (s *Server) tripPage(ctx *fiber.Ctx) error {
	slug := ctx.Params("slug")
	city, country, ok := ParseSlug(slug)
	if !ok {
		return ctx.SendStatus(fiber.StatusNotFound)
	}

	generalInfo, err := s.weatherReporters.GenerateReport(ctx.Context(), city+", "+country)
	if err != nil {
		log.Printf("Error retrieving weather data for /trip/%s with error %s", slug, err)
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	var data TemplateData
	newCity := false
	if data, err = s.checkDatabase(generalInfo.City); err != nil {
		newCity = true
		if data, err = s.retireveFreshInformation(ctx, generalInfo, generalInfo.City); err != nil {
			log.Printf("Error retrieving fresh data for /trip/%s with error %s", slug, err)
		}
	}

	s.recordSitemapSlug(slug)

	days, planned := s.slugPlanDays(ctx)

	return ctx.Render("index", fiber.Map{
		"Query":           data.GeneralInfo.City,
		"GeneralInfo":     data.GeneralInfo,
		"Videos":          data.Videos,
		"Trip":            s.tripCardForSlug(ctx, data.GeneralInfo, newCity),
		"PageTitle":       tripPageTitle(data.GeneralInfo.City, days, planned),
		"PageDescription": tripPageDescription(data.GeneralInfo.City, days, planned),
		"CanonicalURL":    "/trip/" + slug,
	})
}

// slugPlanDays is the day count a /trip/:slug page would plan for right now: the query's own
// days value when the whole days/from pair is valid, defaultTripDays otherwise. planned says
// whether that pair was actually valid, for the title and description to know whether they are
// describing a plan that is really on the page.
func (s *Server) slugPlanDays(ctx *fiber.Ctx) (days int, planned bool) {
	days, _, ok := parseDaysAndStart(ctx, "days", "from", s.today())
	if !ok {
		return defaultTripDays, false
	}
	return days, true
}

// tripCardForSlug builds the trip card /trip/:slug shows: prefilled from the slug's city, and
// already carrying the plan when a valid days/from pair is on the query string. It returns nil
// (as the template sees it) when no planner is wired in, exactly like the home page.
func (s *Server) tripCardForSlug(ctx *fiber.Ctx, info application.GeneralWeatherInfo, newCity bool) any {
	if s.tripPlanner == nil {
		return nil
	}

	req := planner.Request{City: info.City, Country: info.Country, Lat: info.Lat, Lon: info.Lon}

	days, start, ok := parseDaysAndStart(ctx, "days", "from", s.today())
	if !ok {
		return s.newTripCard(req)
	}
	req.Days, req.Start = days, start
	card := s.newTripCard(req)

	if newCity && s.newCityLimiter != nil && !s.newCityLimiter.Allow(s.clock()) {
		card.Error = tooManyNewCitiesMessage
		return card
	}

	plan, err := s.tripPlanner.Plan(ctx.Context(), req)
	if err != nil || plan == nil {
		log.Printf("Error planning a trip for %s with error %v", req.City, err)
		card.Error = planFailMessage
		return card
	}

	card.Plan = newTripPlanView(plan)
	return card
}

// resolveTripPlan turns a /trip/:slug export request's slug, days and from into a Plan. It
// reports ok=false for anything that can't produce a full plan to export - an unparsable slug,
// a missing or invalid days/from pair, or the planner itself failing - so an export route can
// 404 instead of serving an empty or broken file.
func (s *Server) resolveTripPlan(ctx *fiber.Ctx) (*planner.Plan, bool) {
	if s.tripPlanner == nil {
		return nil, false
	}

	city, country, ok := ParseSlug(ctx.Params("slug"))
	if !ok {
		return nil, false
	}

	days, start, ok := parseDaysAndStart(ctx, "days", "from", s.today())
	if !ok {
		return nil, false
	}

	generalInfo, err := s.weatherReporters.GenerateReport(ctx.Context(), city+", "+country)
	if err != nil {
		log.Printf("Error resolving coordinates for an export of %s, %s with error %v", city, country, err)
		return nil, false
	}

	req := planner.Request{
		City:    generalInfo.City,
		Country: generalInfo.Country,
		Lat:     generalInfo.Lat,
		Lon:     generalInfo.Lon,
		Days:    days,
		Start:   start,
	}

	plan, err := s.tripPlanner.Plan(ctx.Context(), req)
	if err != nil || plan == nil {
		log.Printf("Error planning an export for %s with error %v", req.City, err)
		return nil, false
	}

	return plan, true
}

// parseDaysAndStart reads a day count and a start date from ctx's query, under the given key
// names. It reports ok=false for anything missing, malformed or in the past, with no message:
// callers that fall back silently (like /trip/:slug) just skip the plan; /plan's own parsing
// builds its own user-facing text from the individual checks instead.
func parseDaysAndStart(ctx *fiber.Ctx, dayKey, startKey string, today time.Time) (days int, start time.Time, ok bool) {
	days, err := strconv.Atoi(ctx.Query(dayKey))
	if err != nil || days < 1 || days > planner.MaxDays {
		return 0, time.Time{}, false
	}

	start, err = time.ParseInLocation(tripDateLayout, ctx.Query(startKey), time.UTC)
	if err != nil || start.Before(today) {
		return 0, time.Time{}, false
	}

	return days, start, true
}

// tripPageTitle is the <title> a /trip/:slug page carries: a planned trip names its length,
// an unplanned one just invites planning one.
func tripPageTitle(city string, days int, planned bool) string {
	if !planned {
		return fmt.Sprintf("Plan a trip to %s — TravelTab", city)
	}
	return fmt.Sprintf("%d %s in %s — TravelTab", days, dayUnit(days), city)
}

// tripPageDescription is the page's meta description, matching tripPageTitle's two cases.
func tripPageDescription(city string, days int, planned bool) string {
	if !planned {
		return fmt.Sprintf("Plan a weather-aware, day-by-day trip to %s, with places to stay nearby.", city)
	}
	// "%d-day trip" is a hyphenated adjective, so it stays singular even for 3 days -
	// "a 3-day trip", never "a 3-days trip".
	return fmt.Sprintf("A day-by-day plan for a %d-day trip to %s, with weather-aware stops and places to stay.", days, city)
}

func dayUnit(days int) string {
	if days == 1 {
		return "day"
	}
	return "days"
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

// clock is the full current instant, on the same test-controllable source as today().
func (s *Server) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
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

	view.ICSURL, view.KMLURL = exportURLs(plan.Request)

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
			MapsURL:     googleMapsWalkingURL(day.Stops),
		})
	}

	return view
}

// exportURLs are the .ics and .kml links for req: the same slug and days/from query
// resolveTripPlan reads, with the extension on the path itself, matching the routes
// registered in server.go ("/trip/:slug.ics", "/trip/:slug.kml"). Both are "" when req
// doesn't carry enough to build a slug (no country) or a date (no start, no days).
func exportURLs(req planner.Request) (ics, kml string) {
	if req.City == "" || req.Country == "" || req.Start.IsZero() || req.Days <= 0 {
		return "", ""
	}

	slug := Slug(req.City, req.Country)
	query := fmt.Sprintf("?days=%d&from=%s", req.Days, req.Start.Format(tripDateLayout))

	return "/trip/" + slug + ".ics" + query, "/trip/" + slug + ".kml" + query
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
