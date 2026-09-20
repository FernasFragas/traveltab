// loadtest-server runs the real TravelTab handlers with local fixture reporters.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"weatherservice/internal/adapters/httpserver"
	"weatherservice/internal/adapters/sqlite"
	app "weatherservice/internal/application"
)

type fixture[T any] struct {
	value T
	delay time.Duration
}

func (f fixture[T]) GenerateReport(ctx context.Context, _ string) (*T, error) {
	if err := wait(ctx, f.delay); err != nil {
		return nil, err
	}
	value := f.value
	return &value, nil
}

type weatherFixture struct{ delay time.Duration }

func (f weatherFixture) GenerateReport(ctx context.Context, query string) (*app.GeneralWeatherInfo, error) {
	if err := wait(ctx, f.delay); err != nil {
		return nil, err
	}
	city := strings.TrimSpace(strings.SplitN(query, ",", 2)[0])
	return &app.GeneralWeatherInfo{
		City: city, Country: "pt", Lat: 38.72, Lon: -9.14,
		Weather: app.Weather{Temperature: 22, FeelsLike: 22, Humidity: 60, Wind: 10, Condition: "Clear"},
		Waves:   app.Waves{Height: 1.2}, EmbedURL: "about:blank",
	}, nil
}

func wait(ctx context.Context, delay time.Duration) error {
	if delay == 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func run() error {
	address := flag.String("listen", "127.0.0.1:8081", "listen address")
	delay := flag.Duration("reporter-delay", 0, "simulated latency per reporter call, e.g. 50ms")
	flag.Parse()
	if *delay < 0 {
		return fmt.Errorf("reporter-delay must not be negative")
	}
	if _, err := os.Stat("views/index.go.tpl"); err != nil {
		return fmt.Errorf("run from the repository root: %w", err)
	}
	dir, err := os.MkdirTemp("", "traveltab-loadtest-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()

	weather := weatherFixture{delay: *delay}
	videos := fixture[app.VideosStream]{value: app.VideosStream{{Title: "Fixture city tour", VideoID: "fixture"}}, delay: *delay}
	storage, err := sqlite.Open(filepath.Join(dir, "fixture.db"))
	if err != nil {
		return err
	}
	defer func() { _ = storage.Close() }()
	server := httpserver.NewAppServer(weather, videos, storage)
	// Seed the working set synchronously, so the benchmark exercises warm cache
	// reads and real visit writes without racing asynchronous cache population.
	for _, city := range []string{"Lisbon", "Porto", "Faro"} {
		info, err := (weatherFixture{}).GenerateReport(context.Background(), city)
		if err != nil {
			return err
		}
		if err := storage.SaveCityData(city, map[string]any{"GeneralInfo": info, "Videos": videos.value}); err != nil {
			return err
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	result := make(chan error, 1)
	go func() { result <- server.Listen(*address) }()
	log.Printf("Fixture server: http://%s; temporary DB: %s; reporter delay: %s", *address, dir, *delay)
	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return nil
	}
}

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
