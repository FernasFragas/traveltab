package main

import (
	"context"
	"log"
	"os"
	"time"
	guidedata "weatherservice/guides"
	"weatherservice/internal/adapters/api"
	"weatherservice/internal/adapters/httpserver"
	"weatherservice/internal/adapters/sqlite"
	"weatherservice/internal/application"
	"weatherservice/internal/config"
	"weatherservice/internal/guides"
	"weatherservice/internal/writermap"
	"weatherservice/internal/writermap/writermaptest"
	"weatherservice/writerdata"
)

const port = ":8080"

func main() {

	weatherServiceSecrets := config.LoadEnvKey()

	reporters := application.NewWeatherReporters(
		api.NewWeatherAPI(weatherServiceSecrets.OpenWeatherAPIKey),
		api.NewOpenMateoAPI(),
	)

	videoStreamReporters := application.NewVideoStreamReporters(
		api.NewYoutubeAPI(weatherServiceSecrets.YoutubeAPIKey),
	)

	// DB_PATH should point at the mounted volume in production so data survives deploys.
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "weatherservice.db"
	}

	storage, err := sqlite.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := storage.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()
	server := httpserver.NewAppServer(reporters, videoStreamReporters, storage)
	wireWritersMap(server)
	basePlaces := storage.NewCachedBasePlaceSource(api.NewWikidataNearbyAPI(nil))
	wireBaseLayer(server, basePlaces)
	wireWriterSync(storage, basePlaces.(application.BasePlaceRefresher))
	server.SetPlaceSuggester(api.NewOpenMeteoGeocodingAPI(nil))

	// The trip planner reads free, keyless sources. Each one goes through the SQLite cache, so a
	// city is fetched once and still plans when a source is down. The command wires adapters into application interfaces; application and planner packages stay independent of them.
	server.SetTripPlanner(application.NewTripPlanner(
		storage.NewCachedPlaceSource(api.NewWikimediaAPI(nil)),
		storage.NewCachedForecastSource(api.NewOpenMeteoForecastAPI(nil)),
		storage.NewCachedStaySource(api.NewOverpassAPI(nil, overpassURLs(weatherServiceSecrets))),
	))

	// Destination photos come from Wikidata and Commons, keyless like the planner sources. The
	// cache keeps each photograph's identity, so a namesake city never receives another's picture.
	server.SetDestinationPhotoSource(storage.NewCachedDestinationPhotoSource(api.NewDestinationPhotoAPI(nil)))

	// Reviewed city guides are compiled in from guides/guides.json and answered from memory. Only
	// entries a person has checked against Wikivoyage are published, and nothing at request time
	// generates a guide or calls a model.
	book, err := guides.ParseBook(guidedata.JSON)
	if err != nil {
		log.Fatal(err)
	}

	server.SetCityGuideSource(book)

	err = server.Listen(port)
	if err != nil {
		log.Fatal(err)
	}
}

func wireWritersMap(server *httpserver.Server) {
	if config.LoadFeatureFlags().WritersMap {
		server.SetWriterMapSource(writermaptest.NewMadeiraSource(), true)
	}
}

func wireBaseLayer(server *httpserver.Server, basePlaces application.BasePlaceSource) {
	if config.LoadFeatureFlags().WritersMap {
		server.SetBasePlaceSource(basePlaces, true)
	}
}

// wireWriterSync starts the writers index sync in the background. The monthly run also refreshes
// every cached destination's base places; the summary step (PR 9) plugs into job.SummaryStep.
func wireWriterSync(storage *sqlite.Store, refresher application.BasePlaceRefresher) {
	if !config.LoadFeatureFlags().WritersSync {
		return
	}
	data, err := writerdata.Files.ReadFile("sources.json")
	if err != nil {
		log.Fatal(err)
	}
	sources, err := writermap.ParseSources(data)
	if err != nil {
		log.Fatal(err)
	}
	client := api.NewWordPressClient(nil, api.NewHostLimiter())
	job := writermap.NewSyncJob(sources, writerIndexStore{storage}, wordPressPosts{client})
	job.BlockedErrors = []error{api.ErrBlocked, api.ErrDisallowed}
	job.Destinations, job.Refresher = storage, refresher
	go job.Schedule(context.Background())
}

// writerIndexStore adapts the SQLite writers index to writermap.SyncIndex.
type writerIndexStore struct{ store *sqlite.Store }

func (s writerIndexStore) UpsertPost(ctx context.Context, post writermap.IndexedPost) error {
	return s.store.UpsertWriterPost(ctx, sqlite.WriterPost{
		Host: post.Host, PostID: post.PostID, URL: post.URL, Title: post.Title,
		Language: post.Language, Modified: post.Modified, Extracted: post.Extracted,
	})
}

func (s writerIndexStore) DeletePost(ctx context.Context, host string, postID int64) error {
	return s.store.DeleteWriterPost(ctx, host, postID)
}

func (s writerIndexStore) PurgeHost(ctx context.Context, host string) error {
	return s.store.PurgeWriterHost(ctx, host)
}

func (s writerIndexStore) PostIDs(ctx context.Context, host string) ([]int64, error) {
	return s.store.WriterPostIDs(ctx, host)
}

func (s writerIndexStore) SyncState(ctx context.Context, host string) (writermap.SyncState, error) {
	state, err := s.store.GetWriterSyncState(ctx, host)
	return writermap.SyncState(state), err
}

func (s writerIndexStore) SaveSyncState(ctx context.Context, state writermap.SyncState) error {
	return s.store.SaveWriterSyncState(ctx, sqlite.WriterSyncState(state))
}

// wordPressPosts adapts the WordPress client to writermap.PostLister.
type wordPressPosts struct{ client *api.WordPressClient }

func (w wordPressPosts) ListPosts(ctx context.Context, apiBase string, modifiedAfter time.Time, page int) ([]writermap.SourcePost, bool, error) {
	posts, more, err := w.client.ListPosts(ctx, apiBase, modifiedAfter, page)
	out := make([]writermap.SourcePost, len(posts))
	for i, post := range posts {
		out[i] = writermap.SourcePost{ID: post.ID, Link: post.Link, Title: post.Title, Modified: post.Modified, Content: post.Content}
	}
	return out, more, err
}

func (w wordPressPosts) PostIDs(ctx context.Context, apiBase string, page int) ([]int64, bool, error) {
	return w.client.PostIDs(ctx, apiBase, page)
}

// overpassURLs uses OVERPASS_URLS when it is set, and the public servers otherwise.
func overpassURLs(secrets *config.WeatherServiceKeys) []string {
	if len(secrets.OverpassURLs) > 0 {
		return secrets.OverpassURLs
	}

	return api.DefaultOverpassURLs
}
