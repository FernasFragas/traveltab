package sqlite

import (
	"context"
	"fmt"
	"log"
	"time"
	"weatherservice/internal/application"
)

// A photograph is metadata about a Commons file, which rarely changes, so it stays fresh for a
// month. Learning that a city has no usable photograph is worth much less: someone may add one,
// and the answer must not outlive a bad afternoon, so it is trusted for an hour only.
const (
	destinationPhotoTTL   = 30 * 24 * time.Hour
	destinationNoPhotoTTL = time.Hour

	// destinationPhotoKeyVersion changes when the entry layout or the selection rules do, so old
	// entries are simply never read.
	destinationPhotoKeyVersion = "v1"
)

// destinationPhotoEntry is one cached answer. It repeats the identity it was stored for, so a hit
// is validated against the request instead of being trusted because its key matched. A nil Photo
// is a confirmed "no suitable photograph"; an error is never stored.
type destinationPhotoEntry struct {
	City, Country string
	Lat, Lon      string // rounded like the key
	Photo         *application.DestinationPhoto
}

// cachedDestinationPhotos wraps a photo source with the source_cache table. Unlike the planner's
// sources it needs two lifetimes, and a lookup runs live in the request whenever the cache cannot
// answer: an empty database, an expired or unreadable entry, or an entry for another destination.
type cachedDestinationPhotos struct {
	cache  *cached[destinationPhotoEntry]
	source application.DestinationPhotoSource
}

// NewCachedDestinationPhotoSource wraps a photo source with an identity-aware cache: 30 days for a
// photograph, one hour for "none found". An expired photograph is refreshed live and kept when
// the refresh fails or finds nothing; an error is returned but never cached.
func (s *Store) NewCachedDestinationPhotoSource(src application.DestinationPhotoSource) application.DestinationPhotoSource {
	return &cachedDestinationPhotos{
		cache:  &cached[destinationPhotoEntry]{db: s.db, source: "destphoto", now: time.Now},
		source: src,
	}
}

// destinationPhotoKey is the key and the identity a valid entry must carry. The key holds the name folded,
// the country and the coordinates to 3 decimals (about 110 m), never the name alone.
func destinationPhotoKey(id application.DestinationIdentity) (string, destinationPhotoEntry) {
	entry := destinationPhotoEntry{
		City:    application.FoldName(id.City),
		Country: id.Country,
		Lat:     fmt.Sprintf("%.3f", id.Lat),
		Lon:     fmt.Sprintf("%.3f", id.Lon),
	}

	return fmt.Sprintf("destphoto:%s:%s:%s:%s:%s", destinationPhotoKeyVersion, entry.City, entry.Country, entry.Lat, entry.Lon), entry
}

func (c *cachedDestinationPhotos) Photo(ctx context.Context, id application.DestinationIdentity) (*application.DestinationPhoto, error) {
	normalized, ok := application.NewDestinationIdentity(id.City, id.Country, id.Lat, id.Lon)
	if !ok {
		return nil, fmt.Errorf("destination photo: unusable identity %+v", id)
	}

	id = normalized
	key, want := destinationPhotoKey(id)

	old, fetchedAt, hit := c.cache.load(key)
	if hit && !validDestinationEntry(old, want) {
		log.Printf("Ignoring cached destination photo %s: it does not describe this destination", key)
		hit = false
	}

	if hit {
		ttl := destinationPhotoTTL
		if old.Photo == nil {
			ttl = destinationNoPhotoTTL
		}

		if c.cache.now().Sub(fetchedAt) < ttl {
			return old.Photo, nil
		}
	}

	fresh, err := c.source.Photo(ctx, id)
	if err == nil && fresh != nil && !fresh.Complete() {
		err = fmt.Errorf("destination photo for %s has incomplete attribution", key)
	}

	switch {
	case err != nil && hit && old.Photo != nil:
		log.Printf("Refreshing %s failed, serving the photo cached at %s: %v", key, fetchedAt.Format(time.RFC3339), err)
		return old.Photo, nil
	case err != nil:
		return nil, err
	case fresh == nil && hit && old.Photo != nil:
		log.Printf("Refreshing %s found no photo, keeping the one cached at %s", key, fetchedAt.Format(time.RFC3339))
		return old.Photo, nil
	}

	want.Photo = fresh
	c.cache.store(key, want)

	return fresh, nil
}

// validDestinationEntry accepts an entry only when it is for this very identity and, if it holds a photo,
// that photo has all the attribution the page must show.
func validDestinationEntry(entry, want destinationPhotoEntry) bool {
	if entry.City != want.City || entry.Country != want.Country || entry.Lat != want.Lat || entry.Lon != want.Lon {
		return false
	}

	return entry.Photo == nil || entry.Photo.Complete()
}
