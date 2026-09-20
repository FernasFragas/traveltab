// Command placetypes regenerates internal/planner/placetypes_gen.go from Wikimedia data.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"
	"weatherservice/internal/placetypes"
)

func main() {
	log.SetFlags(0)

	cachePath := flag.String("cache", filepath.Join(os.TempDir(), "traveltab-placetypes.json"),
		"where the fetched city types and class graph are kept between runs")
	out := flag.String("out", filepath.Join("internal", "planner", "placetypes_gen.go"), "the file to write")
	timeout := flag.Duration("timeout", 30*time.Minute, "how long the whole run may take")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()

	if err := placetypes.Run(ctx, *cachePath, *out); err != nil {
		log.Fatalf("placetypes: %v", err)
	}
}
