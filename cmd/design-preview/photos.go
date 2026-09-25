package main

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	app "weatherservice/internal/application"
)

// fixturePhotos is a deterministic stand-in for the Wikimedia photo source. It never reaches the
// network: every photograph is a generated image served from the local fixture origin, so a
// browser check cannot depend on, or be mistaken for evidence about, the live provider.
//
// Which outcome a search gets depends on the resolved destination, like the real source:
//
//	Paris (FR)   an attributed photograph      Paris (US)   a different photograph (Paris, Texas)
//	Porto (PT)   an attributed photograph      Nophoto      a completed lookup with no photo
//	Photoerror   a provider failure            Brokenimage  metadata whose image URL answers 404
//
// Every other destination has no photograph. The photos-missing and fallback scenarios make
// every destination photo-free, keeping a deliberately missing-photo run available.
type fixturePhotos struct {
	origin   string // the local fixture origin, ending in "/"
	scenario string
	calls    atomic.Int64
	mu       sync.Mutex
	seen     []app.DestinationIdentity
}

// Photo records the request so tests can count lookups and inspect the identities asked for.
func (f *fixturePhotos) Photo(_ context.Context, id app.DestinationIdentity) (*app.DestinationPhoto, error) {
	f.calls.Add(1)
	f.mu.Lock()
	f.seen = append(f.seen, id)
	f.mu.Unlock()

	if f.scenario == "photos-missing" || f.scenario == "fallback" {
		return nil, nil
	}

	switch strings.ToLower(id.City) + "/" + id.Country {
	case "paris/FR":
		return f.photo(id, "paris", "Fixture Photographer (Paris)"), nil
	case "paris/US":
		return f.photo(id, "paris-texas", "Fixture Photographer (Paris, Texas)"), nil
	case "porto/PT":
		return f.photo(id, "porto", "Fixture Photographer (Porto)"), nil
	case "brokenimage/PT":
		return f.photo(id, "missing", "Fixture Photographer (broken image)"), nil
	case "photoerror/PT":
		return nil, errors.New("fixture photo provider failure")
	}

	return nil, nil
}

func (f *fixturePhotos) photo(id app.DestinationIdentity, name, credit string) *app.DestinationPhoto {
	return &app.DestinationPhoto{
		URL:       f.origin + "fixture-photos/" + name + ".png",
		Alt:       "View of " + id.City,
		Credit:    credit,
		CreditURL: "https://example.com/fixture-photos/" + name,
		License:   "CC BY-SA 4.0", LicenseURL: "https://creativecommons.org/licenses/by-sa/4.0/",
		Width: fixturePhotoWidth, Height: fixturePhotoHeight, SourceID: "Q-" + name,
	}
}

const (
	fixturePhotoWidth  = 1280
	fixturePhotoHeight = 720
)

// fixturePalettes give each generated photograph its own sky and ground, so two cities are
// visibly different in a screenshot.
var fixturePalettes = map[string][2]color.RGBA{
	"paris":       {{0x7d, 0x9d, 0xc8, 0xff}, {0x4f, 0x5b, 0x6e, 0xff}},
	"paris-texas": {{0xe8, 0xa2, 0x5d, 0xff}, {0x8f, 0x4a, 0x2a, 0xff}},
	"porto":       {{0x5f, 0xb3, 0xb0, 0xff}, {0x2f, 0x6b, 0x73, 0xff}},
}

var (
	fixturePhotoOnce  sync.Once
	fixturePhotoBytes = map[string][]byte{}
)

// fixturePhotoHandler serves /fixture-photos/NAME.png. Unknown names, including the deliberately
// broken "missing", answer 404.
func fixturePhotoHandler(w http.ResponseWriter, r *http.Request) {
	fixturePhotoOnce.Do(func() {
		for name, palette := range fixturePalettes {
			fixturePhotoBytes[name] = renderFixturePhoto(palette[0], palette[1])
		}
	})

	name := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/fixture-photos/"), ".png")

	data, ok := fixturePhotoBytes[name]
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(data)
}

// renderFixturePhoto paints a sky gradient over a ground band with a stepped skyline, all from
// two colours, so the same bytes come out every run.
func renderFixturePhoto(sky, ground color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, fixturePhotoWidth, fixturePhotoHeight))
	horizon := fixturePhotoHeight * 2 / 3

	for y := 0; y < fixturePhotoHeight; y++ {
		for x := 0; x < fixturePhotoWidth; x++ {
			c := lerp(sky, color.RGBA{0xff, 0xff, 0xff, 0xff}, float64(y)/float64(horizon)*0.5)

			// Buildings of stepped height stand on the horizon.
			height := 40 + (x/64*37)%120
			if y >= horizon-height {
				c = lerp(ground, color.RGBA{0, 0, 0, 0xff}, float64(x%64)/256)
			}

			if y >= horizon {
				c = ground
			}

			img.SetRGBA(x, y, c)
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err) // encoding an in-memory image cannot fail
	}

	return buf.Bytes()
}

func lerp(a, b color.RGBA, t float64) color.RGBA {
	mix := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t) }

	return color.RGBA{mix(a.R, b.R), mix(a.G, b.G), mix(a.B, b.B), 0xff}
}
