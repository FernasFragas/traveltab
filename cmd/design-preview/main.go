// design-preview runs TravelTab's actual HTTP server and templates against deterministic,
// clearly labeled local data. It is preparation for redesign verification, not evidence that
// live maps, photography, or YouTube playback work.
//
// Usage:
//
//	go run ./cmd/design-preview [-addr 127.0.0.1:8087] [-map-addr 127.0.0.1:8088] [-slow]
package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8087", "HTTP listen address for the preview server")
	mapAddr := flag.String("map-addr", "127.0.0.1:8088", "HTTP listen address for the unmistakable local map fixture")
	slow := flag.Bool("slow", false, "delay search and planning responses by two seconds")
	scenario := flag.String("scenario", "default", "fixture scenario (see tasks/redesign/preview.md)")
	flag.Parse()
	if !validScenario(*scenario) {
		log.Fatalf("unknown fixture scenario %q; see tasks/redesign/preview.md", *scenario)
	}

	mapOrigin := "http://" + *mapAddr + "/"
	mapServer := &http.Server{
		Addr:              *mapAddr,
		Handler:           http.HandlerFunc(fixtureMapHandler),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Printf("fixture map listening on %s", mapOrigin)
		if err := mapServer.ListenAndServe(); err != nil && !errorIs(err, http.ErrServerClosed) {
			log.Printf("fixture map stopped: %v", err)
		}
	}()

	server, cleanup := newPreviewServer(mapOrigin, *slow, *scenario)
	defer func() { cleanup(); _ = mapServer.Close() }()
	log.Printf("fixture preview listening on http://%s/", *addr)
	if err := server.Listen(*addr); err != nil && !errorIs(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func fixtureMapHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>FIXTURE MAP — NOT LIVE WAZE</title><style>
html,body{margin:0;min-height:100vh;background:#f0e6d2;color:#5b442f;font:16px/1.5 system-ui,sans-serif}
body{display:grid;place-items:center;padding:24px;box-sizing:border-box}
figure{max-width:min(720px,90vw);border:8px dashed #a64027;background:#fffdfa;padding:24px;text-align:center;box-shadow:0 8px 24px rgba(0,0,0,.15)}
h1{margin:0 0 8px;font-size:26px;letter-spacing:.03em;color:#a64027}
p{margin:0}
small{display:block;margin-top:12px;color:#7a6a59}
</style></head><body><main><figure><h1>FIXTURE MAP</h1><p>This is local test content. It is not a live map provider.</p><small>No Waze tiles, routing, or provider behavior are verified here.</small></figure></main></body></html>`))
}

func errorIs(err error, target error) bool {
	return errors.Is(err, target)
}
