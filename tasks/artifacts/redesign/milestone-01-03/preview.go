// Run from the repository root with go run ./tasks/artifacts/redesign/milestone-01-03.
// This milestone fixture uses real routes and a labeled local map; it is not a live-provider check.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"weatherservice/internal/adapters/httpserver"
	"weatherservice/internal/adapters/sqlite"
	app "weatherservice/internal/application"
	"weatherservice/internal/planner"
)

type weather struct{}

func (weather) GenerateReport(_ context.Context, query string) (*app.GeneralWeatherInfo, error) {
	city := strings.TrimSpace(strings.Split(query, ",")[0])
	if strings.EqualFold(city, "error") {
		return nil, errors.New("fixture search failure")
	}
	return &app.GeneralWeatherInfo{City: city, Country: "Portugal", Lat: 38.7223, Lon: -9.1393, Weather: app.Weather{Temperature: 22, FeelsLike: 22, Humidity: 60, Wind: 10, Condition: "Clear"}, Waves: app.Waves{Height: 1.2}, EmbedURL: "http://127.0.0.1:8088/"}, nil
}

type videos struct{}

func (videos) GenerateReport(context.Context, string) (*app.VideosStream, error) {
	v := app.VideosStream{}
	return &v, nil
}

type trips struct{}

func (trips) Plan(_ context.Context, r planner.Request) (*planner.Plan, error) {
	if r.Days == 4 {
		return nil, errors.New("fixture planning failure")
	}
	p := &planner.Plan{Request: r, BookingURL: planner.BookingURL(r.City, r.Country, r.Start, r.Days), Stays: []planner.Stay{{Name: "Fixture Hotel", Kind: "hotel", Stars: 4, Website: "https://example.com"}, {Name: "Fixture Hostel", Kind: "hostel"}}}
	for i := 0; i < r.Days; i++ {
		rain := 0.2
		if i == 1 {
			rain = 12.4
		}
		date := r.Start.AddDate(0, 0, i)
		p.Days = append(p.Days, planner.Day{Date: date, Certain: true, Rainy: i == 1, Forecast: &planner.DayForecast{Date: date, RainMM: &rain}, WalkKM: 2.5, Stops: []planner.Place{{ID: "Q193386", Name: "Belém Tower", Kind: planner.Outdoor}, {ID: "Q208420", Name: "Jerónimos Monastery", Kind: planner.Mixed}, {ID: "Q1770963", Name: "National Tile Museum", Kind: planner.Indoor}}})
	}
	return p, nil
}
func main() {
	go func() {
		log.Print(http.ListenAndServe("127.0.0.1:8088", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!doctype html><html lang="en"><title>Fixture map</title><body style="margin:0;display:grid;place-items:center;min-height:100vh;background:#eeeadd;color:#666158;font:14px system-ui"><p>Fixture map — live provider not loaded</p></body></html>`))
		})))
	}()

	dir, err := os.MkdirTemp("", "traveltab-redesign-preview-")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	store, err := sqlite.Open(filepath.Join(dir, "fixture.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	s := httpserver.NewAppServer(weather{}, videos{}, store)
	s.SetTripPlanner(trips{})
	log.Fatal(s.Listen("127.0.0.1:8087"))
}
