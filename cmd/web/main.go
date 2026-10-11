package main

import (
	"context"
	"log"
	"os"
	"slices"
	guidedata "weatherservice/guides"
	"weatherservice/internal/adapters/api"
	"weatherservice/internal/adapters/httpserver"
	"weatherservice/internal/adapters/sqlite"
	"weatherservice/internal/application"
	"weatherservice/internal/config"
	"weatherservice/internal/guides"
	"weatherservice/internal/writermap"
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
	basePlaces := storage.NewCachedBasePlaceSource(api.NewWikidataNearbyAPI(nil))
	sources, overrides := loadWriterData()
	writerSource := writermap.NewMapSource(basePlaces, storage, storage, writerIndexStore{storage}, sources, overrides)
	wireWritersMap(server, writerSource)
	wireBaseLayer(server, basePlaces)
	wireWriterSync(storage, basePlaces, sources, overrides, writerSource)
	server.SetPlaceSuggester(api.NewOpenMeteoGeocodingAPI(nil))

	// The trip planner reads free, keyless sources. Each one goes through the SQLite cache, so a
	// city is fetched once and still plans when a source is down. The command wires adapters into application interfaces; application and planner packages stay independent of them.
	cachedForecast := storage.NewCachedForecastSource(api.NewOpenMeteoForecastAPI(nil))
	server.SetForecastSource(cachedForecast)
	server.SetTripPlanner(application.NewTripPlanner(
		storage.NewCachedPlaceSource(api.NewWikimediaAPI(nil)),
		cachedForecast,
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

// wireWritersMap serves the real source, built from the base places, the writers index and the
// matcher. The Madeira fixture stays in writermaptest for the design preview.
func wireWritersMap(server *httpserver.Server, source application.WriterMapSource) {
	if config.LoadFeatureFlags().WritersMap {
		server.SetWriterMapSource(source, true)
	}
}

func wireBaseLayer(server *httpserver.Server, basePlaces application.BasePlaceSource) {
	if config.LoadFeatureFlags().WritersMap {
		server.SetBasePlaceSource(basePlaces, true)
	}
}

// wireWriterSync starts the writers index sync in the background. Each run picks at most 100
// destination posts. The search terms are the override areas and the cached destinations; the
// known place names add the matcher's names for those destinations' base places. The monthly run
// also refreshes every cached destination's base places, then recomputes the stored writers' map
// results; the summary step (PR 9) runs after the recompute.
func wireWriterSync(storage *sqlite.Store, basePlaces application.BasePlaceSource, sources []writermap.Source, overrides writermap.Overrides, writerSource *writermap.MapSource) {
	if !config.LoadFeatureFlags().WritersSync {
		return
	}
	client := api.NewWordPressClient(nil, api.NewHostLimiter())
	job := writermap.NewSyncJob(sources, writerIndexStore{storage}, wordPressPosts{client})
	job.BlockedErrors = []error{api.ErrBlocked, api.ErrDisallowed}
	job.Areas = overrides.Areas
	job.Destinations, job.Refresher = storage, basePlaces.(application.BasePlaceRefresher)
	job.PlaceNames = func(ctx context.Context) ([]string, error) {
		return cachedPlaceNames(ctx, storage, basePlaces, overrides)
	}
	job.SummaryStep = writerSource.Recompute
	go job.Schedule(context.Background())
}

// cachedPlaceNames returns the matcher's place names for every cached destination, so a post
// whose title names any of them counts as a destination post.
func cachedPlaceNames(ctx context.Context, storage *sqlite.Store, basePlaces application.BasePlaceSource, overrides writermap.Overrides) ([]string, error) {
	destinations, err := storage.CachedDestinations(ctx)
	if err != nil {
		return nil, err
	}
	var contextNames []string
	for _, area := range overrides.Areas {
		contextNames = append(contextNames, area.Name)
	}
	var names []string
	for _, destination := range destinations {
		base, err := basePlaces.Places(ctx, destination)
		if err != nil {
			log.Printf("writers sync: place names for %s: %v", destination.Name, err)
			continue
		}
		names = append(names, writermap.MatchLookupPhrases(base.Gazetteer, append(slices.Clone(contextNames), destination.Name), overrides.MatchOverrides)...)
	}
	return names, nil
}

// loadWriterData reads the embedded allowlist and reviewed overrides.
func loadWriterData() ([]writermap.Source, writermap.Overrides) {
	data, err := writerdata.Files.ReadFile("sources.json")
	if err != nil {
		log.Fatal(err)
	}
	sources, err := writermap.ParseSources(data)
	if err != nil {
		log.Fatal(err)
	}
	data, err = writerdata.Files.ReadFile("overrides.json")
	if err != nil {
		log.Fatal(err)
	}
	overrides, err := writermap.ParseOverrides(data)
	if err != nil {
		log.Fatal(err)
	}
	return sources, overrides
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

// wordPressPosts adapts the WordPress client to writermap.PostSource.
type wordPressPosts struct{ client *api.WordPressClient }

func (w wordPressPosts) SearchPosts(ctx context.Context, apiBase, term string, page int) ([]writermap.SearchHit, bool, error) {
	hits, more, err := w.client.SearchPosts(ctx, apiBase, term, page)
	out := make([]writermap.SearchHit, len(hits))
	for i, hit := range hits {
		out[i] = writermap.SearchHit{ID: hit.ID, Link: hit.Link, Slug: hit.Slug, Title: hit.Title, Published: hit.Published}
	}
	return out, more, err
}

func (w wordPressPosts) RecentCommentCounts(ctx context.Context, apiBase string, maxPages int) (map[int64]int, error) {
	return w.client.RecentCommentCounts(ctx, apiBase, maxPages)
}

func (w wordPressPosts) PostsByID(ctx context.Context, apiBase string, ids []int64) ([]writermap.SourcePost, error) {
	posts, err := w.client.PostsByID(ctx, apiBase, ids)
	out := make([]writermap.SourcePost, len(posts))
	for i, post := range posts {
		out[i] = writermap.SourcePost{ID: post.ID, Link: post.Link, Title: post.Title, Modified: post.Modified, Content: post.Content}
	}
	return out, err
}

// overpassURLs uses OVERPASS_URLS when it is set, and the public servers otherwise.
func overpassURLs(secrets *config.WeatherServiceKeys) []string {
	if len(secrets.OverpassURLs) > 0 {
		return secrets.OverpassURLs
	}

	return api.DefaultOverpassURLs
}
