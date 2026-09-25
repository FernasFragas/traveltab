package httpserver

import (
	"context"
	"errors"
	"log"
	"net/url"
	"weatherservice/internal/application"
	"weatherservice/internal/planner"

	"github.com/gofiber/fiber/v2"
)

// SetDestinationPhotoSource switches live destination photos on. Without a source, only curated
// photographs are shown and every other destination gets the photo-unavailable state.
func (s *Server) SetDestinationPhotoSource(source application.DestinationPhotoSource) {
	s.photoSource = source
}

// photoLookup is a photograph lookup running for one response. It starts as soon as the
// destination is known, so it overlaps the cache read and the video request, and the response
// always waits for it before rendering: nothing can paint a photo after the page is sent, and a
// later search can never receive an earlier city's photo.
type photoLookup struct {
	done  chan struct{}
	photo *application.DestinationPhoto
}

// startPhotoLookup begins looking for a photograph of the freshly resolved destination. It
// returns nil when none is needed: no source is wired in, a curated photo overrides it, or the
// destination has no usable identity (an unusable identity is never searched for by name alone).
func (s *Server) startPhotoLookup(ctx context.Context, info application.GeneralWeatherInfo) *photoLookup {
	if s.photoSource == nil || newPresentation(info, nil).Hero != nil {
		return nil
	}

	identity, ok := application.NewDestinationIdentity(info.City, info.Country, info.Lat, info.Lon)
	if !ok {
		log.Printf("No destination photo lookup for %q: country %q and coordinates %f,%f do not identify it", info.City, info.Country, info.Lat, info.Lon)
		return nil
	}

	lookup := &photoLookup{done: make(chan struct{})}

	go func() {
		defer close(lookup.done)

		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("Destination photo lookup for %s panicked: %v", info.City, recovered)
				lookup.photo = nil
			}
		}()

		photo, err := s.photoSource.Photo(ctx, identity)
		if err != nil {
			// The rest of the destination is still worth showing.
			log.Printf("Error looking up a photo of %s, %s: %v", identity.City, identity.Country, err)
			return
		}

		lookup.photo = photo
	}()

	return lookup
}

// wait blocks until the lookup finishes and returns its photograph, nil when it found none.
func (l *photoLookup) wait() *application.DestinationPhoto {
	if l == nil {
		return nil
	}

	<-l.done

	return l.photo
}

// destinationPresentation is the one presentation builder behind the initial page, the HTMX
// search fragment and the shared trip page. A curated photograph wins; otherwise the live
// lookup's photograph is used; with neither, the view says so explicitly. The city guide is
// chosen here too, for the same resolved destination, so the initial page, the HTMX fragment and
// the shared trip page cannot disagree about which guide (if any) belongs to the city shown.
func (s *Server) destinationPresentation(info application.GeneralWeatherInfo, videos application.VideosStream, lookup *photoLookup) presentationView {
	view := newPresentation(info, videos)
	view.Guide = s.guideFor(info)

	if view.Hero != nil {
		return view
	}

	view.Hero = heroFromPhoto(lookup.wait())
	view.PhotoUnavailable = view.Hero == nil

	return view
}

// heroFromPhoto maps a photograph to the template's view, or nil when it lacks required
// attribution or its links are not plain web addresses. The lookup validates hosts already; this
// keeps an unsafe URL from any source out of an src or href.
func heroFromPhoto(photo *application.DestinationPhoto) *imageView {
	if photo == nil || !photo.Complete() || !webURL(photo.URL) || !webURL(photo.CreditURL) {
		return nil
	}

	hero := &imageView{
		URL: photo.URL, Alt: photo.Alt, Credit: photo.Credit, CreditURL: photo.CreditURL,
		License: photo.License, Width: photo.Width, Height: photo.Height,
	}
	if webURL(photo.LicenseURL) {
		hero.LicenseURL = photo.LicenseURL
	}

	return hero
}

func webURL(raw string) bool {
	parsed, err := url.Parse(raw)

	return err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Host != "" && parsed.User == nil
}

var errNamesakeCache = errors.New("cached city is another destination")

// loadDestination returns the page data for a resolved destination: the cached page when it is
// for this very destination, otherwise fresh data. The cache is keyed by name, so a cached
// "Paris" may be Paris, Texas; it is used only when its country and coordinates agree with what
// this search resolved. fromCache is false when the data is fresh.
func (s *Server) loadDestination(ctx *fiber.Ctx, freshInfo *application.GeneralWeatherInfo) (data TemplateData, fromCache bool) {
	city := freshInfo.City

	data, err := s.checkDatabase(city)
	if err == nil && !sameDestination(data.GeneralInfo, *freshInfo) {
		err = errNamesakeCache
		log.Printf("Cached data for %q describes %q, %f,%f, not the searched %q, %f,%f", city, data.GeneralInfo.Country, data.GeneralInfo.Lat, data.GeneralInfo.Lon, freshInfo.Country, freshInfo.Lat, freshInfo.Lon)
	}

	if err == nil {
		return data, true
	}

	log.Printf("Error Retriving search data information with error %s", err)
	log.Printf("Cache miss for city: %s. Fetching fresh data.", city)

	data, err = s.retireveFreshInformation(ctx, freshInfo, city)
	if err != nil {
		log.Printf("Error Retriving fresh data information with error %s", err)
	}

	return data, false
}

// sameDestination reports whether a cached page describes the freshly resolved destination:
// the same name, the same country and coordinates within application.DestinationMatchKM.
// Records without a country or coordinates cannot be verified and count as different.
func sameDestination(cached, fresh application.GeneralWeatherInfo) bool {
	if application.FoldName(cached.City) != application.FoldName(fresh.City) ||
		countryIdentity(cached.Country) == "" || countryIdentity(cached.Country) != countryIdentity(fresh.Country) {
		return false
	}

	usable := func(info application.GeneralWeatherInfo) bool {
		return validCoordinates(info.Lat, info.Lon) && (info.Lat != 0 || info.Lon != 0)
	}

	return usable(cached) && usable(fresh) &&
		planner.DistanceKM(planner.Place{Lat: cached.Lat, Lon: cached.Lon}, planner.Place{Lat: fresh.Lat, Lon: fresh.Lon}) <= application.DestinationMatchKM
}
