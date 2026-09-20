package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"weatherservice/internal/planner"
)

const (
	wikipediaAPIURL = "https://en.wikipedia.org/w/api.php"
	wikidataAPIURL  = "https://www.wikidata.org/w/api.php"

	// geosearchLimit is the most results one geosearch returns. Hitting it exactly means the
	// search was capped and there are probably more places around.
	geosearchLimit = 500
	// splitSearchRadiusKM and splitRingKM place the smaller searches used when a search is capped:
	// one at the centre and splitRingSearches around it, together covering the whole 10 km circle.
	splitSearchRadiusKM = 5
	splitRingKM         = 8.66
	splitRingSearches   = 6

	// entityBatchSize is how many pages or entities one request asks about.
	entityBatchSize = 50
	// wikimediaConcurrency is how many lookups may be in flight at once. The batches are
	// independent, so they need not wait for each other, but Wikimedia asks anonymous clients to
	// go easy: four at a time is a polite amount of company.
	wikimediaConcurrency = 4
	// detailedPlaces is how many of the most famous places get types, an image and a label.
	detailedPlaces = 300

	wikimediaTimeout = 30 * time.Second
)

// EntityLabel is one label of a Wikidata entity.
type EntityLabel struct {
	Value string `json:"value"`
}

// EntityClaim is one statement about a Wikidata entity.
type EntityClaim struct {
	MainSnak struct {
		DataValue struct {
			Value json.RawMessage `json:"value"`
		} `json:"datavalue"`
	} `json:"mainsnak"`
}

// Entity is a Wikidata item, holding whichever props were asked for.
type Entity struct {
	ID        string                     `json:"id"`
	Labels    map[string]EntityLabel     `json:"labels"`
	Sitelinks map[string]json.RawMessage `json:"sitelinks"`
	Claims    map[string][]EntityClaim   `json:"claims"`
}

// ItemIDs returns the entity IDs a property points at, such as the "instance of" (P31) types.
func (e Entity) ItemIDs(property string) []string {
	var ids []string

	for _, claim := range e.Claims[property] {
		var item struct {
			ID string `json:"id"`
		}

		if err := json.Unmarshal(claim.MainSnak.DataValue.Value, &item); err != nil || item.ID == "" {
			continue
		}

		ids = append(ids, item.ID)
	}

	return ids
}

// StringValue returns the first plain string a property holds, such as the image (P18) file name.
func (e Entity) StringValue(property string) string {
	for _, claim := range e.Claims[property] {
		var value string

		if err := json.Unmarshal(claim.MainSnak.DataValue.Value, &value); err == nil && value != "" {
			return value
		}
	}

	return ""
}

// WikimediaAPI finds famous places with the English Wikipedia and Wikidata APIs. It needs no key.
type WikimediaAPI struct {
	client *http.Client
}

// NewWikimediaAPI returns a client for the Wikimedia APIs. A nil client means a default one with
// a 30 second timeout.
func NewWikimediaAPI(client *http.Client) *WikimediaAPI {
	if client == nil {
		client = &http.Client{Timeout: wikimediaTimeout}
	}

	return &WikimediaAPI{client: client}
}

// geoPage is one Wikipedia page found by a geosearch.
type geoPage struct {
	PageID int     `json:"pageid"`
	Title  string  `json:"title"`
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
}

// PlacesNear returns every candidate within planner.SearchRadiusKM, unfiltered and with Kind
// empty, most famous first.
func (a *WikimediaAPI) PlacesNear(ctx context.Context, lat, lon float64) ([]planner.Place, error) {
	pages, err := a.searchPages(ctx, lat, lon)
	if err != nil {
		return nil, err
	}

	if len(pages) == 0 {
		return nil, nil
	}

	return a.describePages(ctx, pages)
}

// searchPages looks for pages within the search radius, splitting the search into smaller ones
// when the first search comes back capped.
func (a *WikimediaAPI) searchPages(ctx context.Context, lat, lon float64) ([]geoPage, error) {
	pages, err := a.geosearch(ctx, lat, lon, planner.SearchRadiusKM)
	if err != nil {
		return nil, err
	}

	if len(pages) < geosearchLimit {
		return pages, nil
	}

	log.Printf("wikimedia: geosearch around %.4f,%.4f hit the %d result cap, splitting into %d smaller searches",
		lat, lon, geosearchLimit, splitRingSearches+1)

	centre := planner.Place{Lat: lat, Lon: lon}
	centres := [][2]float64{{lat, lon}}

	for i := 0; i < splitRingSearches; i++ {
		ringLat, ringLon := destination(lat, lon, splitRingKM, float64(i)*360/splitRingSearches)
		centres = append(centres, [2]float64{ringLat, ringLon})
	}

	// The smaller searches do not depend on each other, so they run together. Their results are
	// merged in centre order, which is the order the searches used to run in, so which of two
	// searches answers first can never change what comes out.
	found, err := inParallel(ctx, centres, func(ctx context.Context, c [2]float64) ([]geoPage, error) {
		return a.geosearch(ctx, c[0], c[1], splitSearchRadiusKM)
	})
	if err != nil {
		return nil, err
	}

	var merged []geoPage

	seen := make(map[int]bool)

	for _, fromCentre := range found {
		for _, page := range fromCentre {
			if seen[page.PageID] {
				continue
			}

			if planner.DistanceKM(centre, planner.Place{Lat: page.Lat, Lon: page.Lon}) > planner.SearchRadiusKM {
				continue
			}

			seen[page.PageID] = true

			merged = append(merged, page)
		}
	}

	return merged, nil
}

func (a *WikimediaAPI) geosearch(ctx context.Context, lat, lon, radiusKM float64) ([]geoPage, error) {
	params := url.Values{
		"action":        {"query"},
		"list":          {"geosearch"},
		"gscoord":       {fmt.Sprintf("%f|%f", lat, lon)},
		"gsradius":      {strconv.FormatFloat(radiusKM*1000, 'f', 0, 64)},
		"gslimit":       {strconv.Itoa(geosearchLimit)},
		"format":        {"json"},
		"formatversion": {"2"},
	}

	var response struct {
		Query struct {
			GeoSearch []geoPage `json:"geosearch"`
		} `json:"query"`
	}

	if err := a.get(ctx, wikipediaAPIURL, params, &response); err != nil {
		return nil, fmt.Errorf("wikipedia geosearch: %w", err)
	}

	return response.Query.GeoSearch, nil
}

// describePages turns pages into places: their Wikidata ID, then how famous they are, and then
// the types, image and English label of the most famous ones.
func (a *WikimediaAPI) describePages(ctx context.Context, pages []geoPage) ([]planner.Place, error) {
	ids, err := a.wikidataIDs(ctx, pages)
	if err != nil {
		return nil, err
	}

	places := make([]planner.Place, 0, len(pages))
	seen := make(map[string]bool, len(pages))

	for _, page := range pages {
		id := ids[page.PageID]
		if id == "" || seen[id] {
			continue
		}

		seen[id] = true

		places = append(places, planner.Place{ID: id, Name: page.Title, Lat: page.Lat, Lon: page.Lon})
	}

	if len(places) == 0 {
		return nil, nil
	}

	if err := a.addSitelinks(ctx, places); err != nil {
		return nil, err
	}

	sort.SliceStable(places, func(i, j int) bool {
		if places[i].Sitelinks != places[j].Sitelinks {
			return places[i].Sitelinks > places[j].Sitelinks
		}

		return places[i].ID < places[j].ID
	})

	if err := a.addDetails(ctx, places); err != nil {
		return nil, err
	}

	return places, nil
}

// wikidataIDs maps each page ID to its Wikidata ID. Pages without one are left out.
func (a *WikimediaAPI) wikidataIDs(ctx context.Context, pages []geoPage) (map[int]string, error) {
	pageIDs := make([]string, 0, len(pages))
	for _, page := range pages {
		pageIDs = append(pageIDs, strconv.Itoa(page.PageID))
	}

	found, err := inParallel(ctx, batches(pageIDs, entityBatchSize), a.pagePropsBatch)
	if err != nil {
		return nil, err
	}

	ids := make(map[int]string, len(pages))
	for _, batch := range found {
		for pageID, id := range batch {
			ids[pageID] = id
		}
	}

	return ids, nil
}

// pagePropsBatch asks Wikipedia for the Wikidata ID of one batch of pages.
func (a *WikimediaAPI) pagePropsBatch(ctx context.Context, batch []string) (map[int]string, error) {
	params := url.Values{
		"action":        {"query"},
		"prop":          {"pageprops"},
		"ppprop":        {"wikibase_item"},
		"pageids":       {strings.Join(batch, "|")},
		"format":        {"json"},
		"formatversion": {"2"},
	}

	var response struct {
		Query struct {
			Pages []struct {
				PageID    int `json:"pageid"`
				PageProps struct {
					WikibaseItem string `json:"wikibase_item"`
				} `json:"pageprops"`
			} `json:"pages"`
		} `json:"query"`
	}

	if err := a.get(ctx, wikipediaAPIURL, params, &response); err != nil {
		return nil, fmt.Errorf("wikipedia page properties: %w", err)
	}

	ids := make(map[int]string, len(batch))

	for _, page := range response.Query.Pages {
		if page.PageProps.WikibaseItem != "" {
			ids[page.PageID] = page.PageProps.WikibaseItem
		}
	}

	return ids, nil
}

// addSitelinks counts how many Wikimedia sites have a page about each place.
func (a *WikimediaAPI) addSitelinks(ctx context.Context, places []planner.Place) error {
	entities, err := a.GetEntities(ctx, placeIDs(places), "sitelinks")
	if err != nil {
		return err
	}

	for i := range places {
		places[i].Sitelinks = len(entities[places[i].ID].Sitelinks)
	}

	return nil
}

// addDetails fills in the types, image and English name of the most famous places. The rest keep
// their Wikipedia title and no types, because looking every one of them up costs requests.
func (a *WikimediaAPI) addDetails(ctx context.Context, places []planner.Place) error {
	detailed := places
	if len(detailed) > detailedPlaces {
		detailed = detailed[:detailedPlaces]
	}

	entities, err := a.GetEntities(ctx, placeIDs(detailed), "claims|labels")
	if err != nil {
		return err
	}

	for i := range detailed {
		entity, ok := entities[detailed[i].ID]
		if !ok {
			continue
		}

		detailed[i].Types = entity.ItemIDs("P31")
		detailed[i].Image = entity.StringValue("P18")

		if label := entity.Labels["en"].Value; label != "" {
			detailed[i].Name = label
		}
	}

	return nil
}

// GetEntities loads Wikidata entities in batches of 50, keyed by their ID. props is the
// wbgetentities props list, such as "sitelinks" or "claims|labels".
func (a *WikimediaAPI) GetEntities(ctx context.Context, ids []string, props string) (map[string]Entity, error) {
	found, err := inParallel(ctx, batches(ids, entityBatchSize), func(ctx context.Context, batch []string) (map[string]Entity, error) {
		return a.entitiesBatch(ctx, batch, props)
	})
	if err != nil {
		return nil, err
	}
	entities := make(map[string]Entity, len(ids))
	for _, batch := range found {
		for id, entity := range batch {
			entities[id] = entity
		}
	}

	return entities, nil
}

func (a *WikimediaAPI) entitiesBatch(ctx context.Context, ids []string, props string) (map[string]Entity, error) {
	params := url.Values{
		"action":    {"wbgetentities"},
		"props":     {props},
		"ids":       {strings.Join(ids, "|")},
		"languages": {"en"},
		"format":    {"json"},
	}
	var response struct {
		Entities map[string]Entity `json:"entities"`
	}
	if err := a.get(ctx, wikidataAPIURL, params, &response); err != nil {
		return nil, fmt.Errorf("wikidata entities: %w", err)
	}
	return response.Entities, nil
}

// get sends a GET with the Wikimedia User-Agent and decodes a 200 response into out.
func (a *WikimediaAPI) get(ctx context.Context, endpoint string, params url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	req.Header.Set("User-Agent", UserAgent)

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("requesting %s: %w", endpoint, err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))

		return fmt.Errorf("%s returned status %d: %s", endpoint, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding %s response: %w", endpoint, err)
	}

	return nil
}

func placeIDs(places []planner.Place) []string {
	ids := make([]string, len(places))
	for i, place := range places {
		ids[i] = place.ID
	}

	return ids
}

// batches cuts ids into slices of at most size.
func batches(ids []string, size int) [][]string {
	var out [][]string

	for start := 0; start < len(ids); start += size {
		end := start + size
		if end > len(ids) {
			end = len(ids)
		}

		out = append(out, ids[start:end])
	}

	return out
}

// inParallel bounds source traffic and waits for every worker before returning. Results and
// errors retain input order, regardless of which HTTP request completes first.
func inParallel[T, R any](ctx context.Context, inputs []T, fetch func(context.Context, T) (R, error)) ([]R, error) {
	out := make([]R, len(inputs))
	errs := make([]error, len(inputs))
	workers := min(wikimediaConcurrency, len(inputs))
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := worker; i < len(inputs); i += workers {
				if err := ctx.Err(); err != nil {
					errs[i] = err
					continue
				}
				out[i], errs[i] = fetch(ctx, inputs[i])
			}
		}()
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// destination is the point distanceKM away from lat/lon along bearingDeg, clockwise from north.
func destination(lat, lon, distanceKM, bearingDeg float64) (float64, float64) {
	const earthRadiusKM = 6371.0

	angular := distanceKM / earthRadiusKM
	bearing := bearingDeg * math.Pi / 180
	latRad := lat * math.Pi / 180
	lonRad := lon * math.Pi / 180

	destLat := math.Asin(math.Sin(latRad)*math.Cos(angular) + math.Cos(latRad)*math.Sin(angular)*math.Cos(bearing))
	destLon := lonRad + math.Atan2(
		math.Sin(bearing)*math.Sin(angular)*math.Cos(latRad),
		math.Cos(angular)-math.Sin(latRad)*math.Sin(destLat),
	)

	return destLat * 180 / math.Pi, destLon * 180 / math.Pi
}
