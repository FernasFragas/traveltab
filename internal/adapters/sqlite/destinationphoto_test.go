package sqlite

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
	"weatherservice/internal/adapters/api"
	"weatherservice/internal/application"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	parisFR = application.DestinationIdentity{City: "Paris", Country: "FR", Lat: 48.8589, Lon: 2.3200}
	parisTX = application.DestinationIdentity{City: "Paris", Country: "US", Lat: 33.6609, Lon: -95.5555}
)

// photoFake answers with what is set and counts its calls. It names the photo after the call, so
// a test can tell a cached answer from a live one.
type photoFake struct {
	calls  int
	err    error
	none   bool
	broken bool
	seen   []application.DestinationIdentity
}

func (f *photoFake) Photo(_ context.Context, id application.DestinationIdentity) (*application.DestinationPhoto, error) {
	f.calls++
	f.seen = append(f.seen, id)

	switch {
	case f.err != nil:
		return nil, f.err
	case f.none:
		return nil, nil
	}

	photo := &application.DestinationPhoto{
		URL: "https://upload.wikimedia.org/" + id.Country + "/" + id.City + ".jpg", Alt: "View of " + id.City,
		Credit: "Ada", CreditURL: "https://commons.wikimedia.org/wiki/File:" + id.City + ".jpg",
		License: "CC BY-SA 4.0", Width: 1280, Height: 800, SourceID: "Q1",
	}
	if f.broken {
		photo.Credit = ""
	}

	return photo, nil
}

func photoCacheForTest(t *testing.T) (*cachedDestinationPhotos, *photoFake, *Store, *time.Time) {
	t.Helper()

	store, err := Open(filepath.Join(t.TempDir(), "photos.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	fake := &photoFake{}
	c := store.NewCachedDestinationPhotoSource(fake).(*cachedDestinationPhotos)
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	c.cache.now = func() time.Time { return now }

	return c, fake, store, &now
}

func photoURL(t *testing.T, c *cachedDestinationPhotos, id application.DestinationIdentity) string {
	t.Helper()

	photo, err := c.Photo(context.Background(), id)
	require.NoError(t, err)

	if photo == nil {
		return ""
	}

	return photo.URL
}

func TestDestinationPhotoCache_ColdSearchGoesLiveAndAnswersTheSameRequest(t *testing.T) {
	c, fake, _, _ := photoCacheForTest(t)

	// An empty database needs no prewarming: the first search returns the live photograph.
	assert.Equal(t, "https://upload.wikimedia.org/FR/Paris.jpg", photoURL(t, c, parisFR))
	assert.Equal(t, 1, fake.calls)
}

func TestDestinationPhotoCache_WarmHitSkipsTheProvider(t *testing.T) {
	c, fake, _, now := photoCacheForTest(t)

	photoURL(t, c, parisFR)
	*now = now.Add(29 * 24 * time.Hour)
	assert.Equal(t, "https://upload.wikimedia.org/FR/Paris.jpg", photoURL(t, c, parisFR))
	assert.Equal(t, 1, fake.calls)
}

func TestDestinationPhotoCache_SharesEntriesForTheSameDestination(t *testing.T) {
	c, fake, _, _ := photoCacheForTest(t)

	photoURL(t, c, parisFR)
	near := parisFR
	near.City, near.Country, near.Lat = " PARIS ", "fr", 48.85891 // spelling, case and 1 m of geocoder noise
	photoURL(t, c, near)
	assert.Equal(t, 1, fake.calls)
}

func TestDestinationPhotoCache_KeepsCountriesAndCoordinatesApart(t *testing.T) {
	c, fake, _, _ := photoCacheForTest(t)

	assert.Equal(t, "https://upload.wikimedia.org/FR/Paris.jpg", photoURL(t, c, parisFR))
	assert.Equal(t, "https://upload.wikimedia.org/US/Paris.jpg", photoURL(t, c, parisTX), "Paris, Texas never receives Paris, France's photo")
	assert.Equal(t, 2, fake.calls)

	// Same name and country, but another city 100 km away.
	elsewhere := parisTX
	elsewhere.Lat = 34.6
	photoURL(t, c, elsewhere)
	assert.Equal(t, 3, fake.calls)

	// The originals remain cached, side by side.
	photoURL(t, c, parisFR)
	photoURL(t, c, parisTX)
	assert.Equal(t, 3, fake.calls)
}

func TestDestinationPhotoCache_RefreshesAnExpiredPhotoLive(t *testing.T) {
	c, fake, _, now := photoCacheForTest(t)

	photoURL(t, c, parisFR)
	*now = now.Add(31 * 24 * time.Hour)
	photoURL(t, c, parisFR)
	assert.Equal(t, 2, fake.calls)

	photoURL(t, c, parisFR)
	assert.Equal(t, 2, fake.calls, "the refreshed entry is fresh again")
}

func TestDestinationPhotoCache_KeepsAnExpiredPhotoWhenTheRefreshFails(t *testing.T) {
	c, fake, _, now := photoCacheForTest(t)

	photoURL(t, c, parisFR)
	*now = now.Add(31 * 24 * time.Hour)
	fake.err = errors.New("offline")
	assert.Equal(t, "https://upload.wikimedia.org/FR/Paris.jpg", photoURL(t, c, parisFR))

	fake.err = nil
	photoURL(t, c, parisFR)
	assert.Equal(t, 3, fake.calls, "the failure was not cached as a result")
}

func TestDestinationPhotoCache_KeepsAnExpiredPhotoWhenTheRefreshFindsNothing(t *testing.T) {
	c, fake, _, now := photoCacheForTest(t)

	photoURL(t, c, parisFR)
	*now = now.Add(31 * 24 * time.Hour)
	fake.none = true
	assert.Equal(t, "https://upload.wikimedia.org/FR/Paris.jpg", photoURL(t, c, parisFR), "a known photo is not replaced by nothing")
}

func TestDestinationPhotoCache_CachesNoPhotoForOneHourOnly(t *testing.T) {
	c, fake, _, now := photoCacheForTest(t)
	fake.none = true

	assert.Empty(t, photoURL(t, c, parisFR))
	*now = now.Add(59 * time.Minute)
	assert.Empty(t, photoURL(t, c, parisFR))
	assert.Equal(t, 1, fake.calls, "the confirmed absence is reused for an hour")

	*now = now.Add(2 * time.Minute)
	fake.none = false
	assert.Equal(t, "https://upload.wikimedia.org/FR/Paris.jpg", photoURL(t, c, parisFR), "a photograph added later is found")
	assert.Equal(t, 2, fake.calls)
}

func TestDestinationPhotoCache_ErrorsAreNeitherCachedNorPoisonLaterSearches(t *testing.T) {
	c, fake, _, _ := photoCacheForTest(t)
	fake.err = errors.New("Wikimedia returned status 429")

	_, err := c.Photo(context.Background(), parisFR)
	require.ErrorIs(t, err, fake.err)

	fake.err = nil
	assert.Equal(t, "https://upload.wikimedia.org/FR/Paris.jpg", photoURL(t, c, parisFR))
	assert.Equal(t, 2, fake.calls)
}

func TestDestinationPhotoCache_AnErrorNeverOverwritesAKnownPhotoOrANegativeEntry(t *testing.T) {
	c, fake, _, now := photoCacheForTest(t)
	fake.none = true
	photoURL(t, c, parisFR)

	*now = now.Add(2 * time.Hour)
	fake.none, fake.err = false, errors.New("timeout")
	_, err := c.Photo(context.Background(), parisFR)
	require.Error(t, err, "an expired negative entry is not a photo to fall back on")
	assert.Equal(t, 2, fake.calls)
}

func TestDestinationPhotoCache_TreatsUnreadableEntriesAsMisses(t *testing.T) {
	c, fake, store, _ := photoCacheForTest(t)
	key, _ := destinationPhotoKey(parisFR)

	for name, blob := range map[string][]byte{
		"not gzip":     []byte("garbage"),
		"empty":        {},
		"not JSON":     mustGzip(t, "{"),
		"wrong shape":  mustGzip(t, `{"Photo": "text"}`),
		"empty object": mustGzip(t, `{}`),
	} {
		t.Run(name, func(t *testing.T) {
			before := fake.calls
			_, err := store.db.Exec(`REPLACE INTO source_cache (key, fetched_at, data) VALUES (?, ?, ?);`, key, c.cache.now().Unix(), blob)
			require.NoError(t, err)

			assert.Equal(t, "https://upload.wikimedia.org/FR/Paris.jpg", photoURL(t, c, parisFR))
			assert.Equal(t, before+1, fake.calls)
		})
	}
}

func TestDestinationPhotoCache_RejectsEntriesForAnotherIdentityOrWithIncompleteMetadata(t *testing.T) {
	c, fake, store, _ := photoCacheForTest(t)
	key, want := destinationPhotoKey(parisFR)
	stored := func(entry destinationPhotoEntry) {
		blob, err := compressJSON(entry)
		require.NoError(t, err)
		_, err = store.db.Exec(`REPLACE INTO source_cache (key, fetched_at, data) VALUES (?, ?, ?);`, key, c.cache.now().Unix(), blob)
		require.NoError(t, err)
	}
	complete := &application.DestinationPhoto{URL: "https://upload.wikimedia.org/x.jpg", Alt: "a", Credit: "c", CreditURL: "https://commons.wikimedia.org/w", License: "l", Width: 1, Height: 1, SourceID: "Q9"}

	for name, edit := range map[string]func(*destinationPhotoEntry){
		"another country":   func(e *destinationPhotoEntry) { e.Country = "US" },
		"another city":      func(e *destinationPhotoEntry) { e.City = "paris texas" },
		"other coordinates": func(e *destinationPhotoEntry) { e.Lat = "33.661" },
		"no credit":         func(e *destinationPhotoEntry) { e.Photo = &application.DestinationPhoto{URL: complete.URL, Alt: "a"} },
		"no license":        func(e *destinationPhotoEntry) { p := *complete; p.License = ""; e.Photo = &p },
	} {
		t.Run(name, func(t *testing.T) {
			before := fake.calls
			entry := want
			entry.Photo = complete
			edit(&entry)
			stored(entry)

			assert.Equal(t, "https://upload.wikimedia.org/FR/Paris.jpg", photoURL(t, c, parisFR), "an entry that does not describe this destination is ignored")
			assert.Equal(t, before+1, fake.calls)
		})
	}

	// A valid entry stored by hand is served without a lookup.
	before := fake.calls
	entry := want
	entry.Photo = complete
	stored(entry)
	assert.Equal(t, complete.URL, photoURL(t, c, parisFR))
	assert.Equal(t, before, fake.calls)
}

func TestDestinationPhotoCache_RejectsIncompleteLivePhotosWithoutCachingThem(t *testing.T) {
	c, fake, _, _ := photoCacheForTest(t)
	fake.broken = true

	_, err := c.Photo(context.Background(), parisFR)
	require.Error(t, err)

	fake.broken = false
	assert.NotEmpty(t, photoURL(t, c, parisFR))
	assert.Equal(t, 2, fake.calls)
}

func TestDestinationPhotoCache_StorageFailureStillReturnsTheLivePhoto(t *testing.T) {
	c, fake, store, _ := photoCacheForTest(t)
	require.NoError(t, store.Close())

	assert.Equal(t, "https://upload.wikimedia.org/FR/Paris.jpg", photoURL(t, c, parisFR))
	assert.Equal(t, "https://upload.wikimedia.org/FR/Paris.jpg", photoURL(t, c, parisFR))
	assert.Equal(t, 2, fake.calls, "nothing could be cached, so each search is live")

	// A cache without a database (as in fixture servers) is a pass-through.
	c = &cachedDestinationPhotos{cache: &cached[destinationPhotoEntry]{now: time.Now}, source: fake}
	assert.NotEmpty(t, photoURL(t, c, parisFR))
}

func TestDestinationPhotoCache_PassesNormalizedIdentityToTheSource(t *testing.T) {
	c, fake, _, _ := photoCacheForTest(t)

	photoURL(t, c, application.DestinationIdentity{City: "  Sao   Paulo ", Country: "br", Lat: -23.55, Lon: -46.63})
	require.Len(t, fake.seen, 1)
	assert.Equal(t, "Sao Paulo", fake.seen[0].City)
	assert.Equal(t, "BR", fake.seen[0].Country)
}

func TestDestinationPhotoCache_RefusesAnUnusableIdentity(t *testing.T) {
	c, fake, _, _ := photoCacheForTest(t)

	for _, id := range []application.DestinationIdentity{
		{City: "Paris", Country: "France", Lat: 48, Lon: 2},
		{City: "Paris", Country: "FR"},
		{Country: "FR", Lat: 48, Lon: 2},
	} {
		_, err := c.Photo(context.Background(), id)
		require.Error(t, err)
	}

	assert.Zero(t, fake.calls)
}

func TestDestinationPhotoCache_LeavesPlannerEntriesAlone(t *testing.T) {
	c, _, store, _ := photoCacheForTest(t)
	photoURL(t, c, parisFR)

	var keys int
	require.NoError(t, store.db.QueryRow(`SELECT COUNT(*) FROM source_cache WHERE key LIKE 'destphoto:v1:%'`).Scan(&keys))
	assert.Equal(t, 1, keys)
}

func mustGzip(t *testing.T, s string) []byte {
	t.Helper()

	blob, err := compressJSON(json.RawMessage(s))
	if err != nil {
		// Invalid JSON cannot be marshalled as a RawMessage; gzip it directly.
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, err = zw.Write([]byte(s))
		require.NoError(t, err)
		require.NoError(t, zw.Close())

		return buf.Bytes()
	}

	return blob
}

// The live check is opt-in: an empty database, the real Wikimedia APIs, one cold lookup that fills
// the cache and a warm one that must not call the provider.
func TestDestinationPhotoCacheLive(t *testing.T) {
	if os.Getenv("LIVE_API_TESTS") != "1" {
		t.Skip("set LIVE_API_TESTS=1 to call the real Wikimedia APIs")
	}

	store, err := Open(filepath.Join(t.TempDir(), "live.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	counting := &countingSource{inner: api.NewDestinationPhotoAPI(nil)}
	source := store.NewCachedDestinationPhotoSource(counting)

	for _, id := range []application.DestinationIdentity{parisFR, parisTX, {City: "Porto", Country: "PT", Lat: 41.1494, Lon: -8.6108}} {
		start := time.Now()
		cold, err := source.Photo(context.Background(), id)
		coldTime := time.Since(start)
		require.NoError(t, err)
		require.NotNil(t, cold, "%s, %s", id.City, id.Country)

		start = time.Now()
		warm, err := source.Photo(context.Background(), id)
		warmTime := time.Since(start)
		require.NoError(t, err)
		assert.Equal(t, cold, warm)
		t.Logf("%s, %s: cold %s, warm %s, %s", id.City, id.Country, coldTime.Round(time.Millisecond), warmTime.Round(time.Microsecond), cold.SourceID)
	}

	assert.Equal(t, 3, counting.calls, "each destination went live once; the warm repeats were served from the database")
}

type countingSource struct {
	inner application.DestinationPhotoSource
	calls int
}

func (c *countingSource) Photo(ctx context.Context, id application.DestinationIdentity) (*application.DestinationPhoto, error) {
	c.calls++

	return c.inner.Photo(ctx, id)
}
