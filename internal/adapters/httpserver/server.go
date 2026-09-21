package httpserver

import (
	"database/sql"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"weatherservice/internal/application"

	"github.com/gofiber/contrib/fgprof"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/template/html/v2"
)

type Server struct {
	storage application.Storage
	app     *fiber.App

	weatherReporters     application.Reporter[application.GeneralWeatherInfo]
	videoStreamReporters application.Reporter[application.VideosStream]

	tripPlanner    application.TripPlanner
	now            func() time.Time
	newCityLimiter *newCityLimiter
}

type TemplateData struct {
	Query       string
	Videos      application.VideosStream
	GeneralInfo application.GeneralWeatherInfo
}

var store = session.New()

func init() {
	// Register the application.VideosStream type with gob
	gob.Register(application.VideosStream{})
	gob.Register(application.GeneralWeatherInfo{})
}

func NewAppServer(weatherReporters application.Reporter[application.GeneralWeatherInfo], videoStreamReporters application.Reporter[application.VideosStream], storage application.Storage) *Server {
	app := fiber.New(fiber.Config{
		Views: html.New("./views", ".go.tpl"),
	})

	app.Use(fgprof.New())

	// Use the session middleware
	app.Use(func(ctx *fiber.Ctx) error {
		sess, err := store.Get(ctx)
		if err != nil {
			return err
		}
		ctx.Locals("session", sess)
		return ctx.Next()
	})

	server := &Server{
		storage:              storage,
		app:                  app,
		weatherReporters:     weatherReporters,
		videoStreamReporters: videoStreamReporters,
		now:                  time.Now,
		newCityLimiter:       newNewCityLimiter(defaultNewCitiesPerMinute),
	}

	// Serve static files from the "public" directory
	app.Static("/", "./public")

	app.Get("/", server.trackVisit, server.listGeneralInfo)

	app.Get("/process-form/", server.trackVisit, server.listGeneralInfo)

	app.Get("/stats", server.showStats)

	app.Get("/plan", server.planTrip)

	app.Get("/trip/:slug.ics", server.exportICS)

	app.Get("/trip/:slug.kml", server.exportKML)

	app.Get("/trip/:slug", server.trackVisit, server.tripPage)

	app.Get("/sitemap.xml", server.sitemap)

	return server
}

func (s *Server) Listen(port string) error {
	return s.app.Listen(port)
}

func (s *Server) listGeneralInfo(ctx *fiber.Ctx) error {
	cityAndCountry := ctx.FormValue("city_name") // retrieves the name passed in the form
	if cityAndCountry == "" {
		cityAndCountry = "Lisbon, Portugal"
	}

	generalInfo, err := s.weatherReporters.GenerateReport(ctx.Context(), cityAndCountry)
	if err != nil {
		log.Printf("Error Retriving New Weather data information with error %s", err)
		return destinationSearchError(ctx)
	}

	city := generalInfo.City

	// --- BEGIN Cache Check ---
	var data TemplateData

	if data, err = s.checkDatabase(city); err != nil {
		log.Printf("Error Retriving search data information with error %s", err)

		log.Printf("Cache miss for city: %s. Fetching fresh data.", city)

		data, err = s.retireveFreshInformation(ctx, generalInfo, city)
		if err != nil {
			log.Printf("Error Retriving fresh data information with error %s", err)
		}
	}

	// Check if it's an HTMX request
	if ctx.Get("HX-Request") == "true" {
		// Render only the content fragment for HTMX requests
		return ctx.Render("content_fragment", fiber.Map{
			"Presentation": newPresentation(data.GeneralInfo, data.Videos),
			"Query":        city,
			"GeneralInfo":  data.GeneralInfo,
			"Videos":       data.Videos,
			"Trip":         s.tripCardFor(data.GeneralInfo),
		})
	}

	// Render the full page for regular requests
	return ctx.Render("index", fiber.Map{
		"Presentation": newPresentation(data.GeneralInfo, data.Videos),
		"Query":        city,
		"GeneralInfo":  data.GeneralInfo,
		"Videos":       data.Videos,
		"Trip":         s.tripCardFor(data.GeneralInfo),
	})
}

// Search errors swap only the persistent header message, retaining the last
// useful destination. Navigation enables HTMX's error swap for this target.
func destinationSearchError(ctx *fiber.Ctx) error {
	if ctx.Get("HX-Request") == "true" {
		ctx.Set("HX-Retarget", "#destination-search-error")
		ctx.Set("HX-Reswap", "innerHTML")
	}
	return ctx.Status(fiber.StatusInternalServerError).SendString("We could not load that destination. Check the city name and try again.")
}

func (s *Server) checkDatabase(city string) (TemplateData, error) {
	var data TemplateData

	if city == "" {
		return TemplateData{}, fmt.Errorf("city is required")
	}

	if s.storage == nil {
		return TemplateData{}, fmt.Errorf("database is not initialized")
	}
	cachedJSON, err := s.storage.GetCityData(city)
	if err != nil && err != sql.ErrNoRows {
		// Handle potential DB errors (other than not found)
		log.Printf("Error checking cache for city %s: %v", city, err)
		// Decide how to handle this - maybe proceed to fetch fresh data, or return an error
		// For now, let's proceed to fetch fresh data, but log the error.
	} else if err == nil {
		// Cache hit!
		log.Printf("Cache hit for city: %s", city)

		if unmarshalErr := json.Unmarshal([]byte(cachedJSON), &data); unmarshalErr != nil {
			log.Printf("Error unmarshaling cached data for city %s: %v", city, unmarshalErr)
			// Data in DB is corrupted? Proceed to fetch fresh data.
		}
	} else {
		return TemplateData{}, err
	}

	return data, nil
}

func (s *Server) retireveFreshInformation(ctx *fiber.Ctx, generalInfo *application.GeneralWeatherInfo, city string) (TemplateData, error) {
	ctx.Status(fiber.StatusOK)

	videos, err := s.videoStreamReporters.GenerateReport(ctx.Context(), fmt.Sprintf("Turistic places in %s, %s", city, generalInfo.Country))
	if err != nil {
		videos = &application.VideosStream{}
	}

	data := TemplateData{
		GeneralInfo: *generalInfo,
		Videos:      *videos,
	}

	// Save the fetched data to the database
	go func(cityToSave string, dataToSave TemplateData) {
		dtToSave := map[string]any{
			"GeneralInfo": dataToSave.GeneralInfo,
			"Videos":      dataToSave.Videos,
		}

		if s.storage == nil {
			return
		}
		if err := s.storage.SaveCityData(cityToSave, dtToSave); err != nil {
			log.Printf("Error saving data for city %s to DB: %v", cityToSave, err)
		}
	}(city, data)

	return data, nil
}
