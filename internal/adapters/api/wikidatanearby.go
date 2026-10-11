package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"weatherservice/internal/application"
	"weatherservice/internal/planner"
	"weatherservice/internal/writermap"
)

const (
	wikidataNearbyURL    = "https://query.wikidata.org/sparql"
	wikidataNearbyRadius = 40
	wikidataNearbyLimit  = 200
)

// WikidataNearbyAPI finds nearby entities once, then selects base pins from that gazetteer.
type WikidataNearbyAPI struct {
	client *http.Client
}

var _ application.BasePlaceSource = (*WikidataNearbyAPI)(nil)

func NewWikidataNearbyAPI(client *http.Client) *WikidataNearbyAPI {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &WikidataNearbyAPI{client: client}
}

// Places returns all nearby named entities for matching and up to 200 visitable base pins.
func (a *WikidataNearbyAPI) Places(ctx context.Context, dest writermap.Area) (writermap.BasePlaces, error) {
	radius := dest.RadiusKM
	if radius <= 0 {
		radius = wikidataNearbyRadius
	}

	rows, err := a.nearby(ctx, dest.Lat, dest.Lon, radius)
	if err != nil {
		return writermap.BasePlaces{}, err
	}
	result := nearbyPlaces(rows)
	if err := a.addPhotoCredits(ctx, result.Gazetteer); err != nil {
		if ctx.Err() != nil {
			return writermap.BasePlaces{}, ctx.Err()
		}
		for i := range result.Gazetteer {
			result.Gazetteer[i].Photo = ""
			result.Gazetteer[i].PhotoCredit = ""
			result.Gazetteer[i].PhotoURL = ""
		}
	}
	byQID := make(map[string]writermap.Place, len(result.Gazetteer))
	for _, place := range result.Gazetteer {
		byQID[place.QID] = place
	}
	for i := range result.Pins {
		result.Pins[i] = byQID[result.Pins[i].QID]
	}
	return result, nil
}

type nearbyBindingValue struct {
	Value string `json:"value"`
}

type nearbyBinding struct {
	Item, Coord, Type, TypeLabel       nearbyBindingValue
	EnLabel, PtLabel, EnAlias, PtAlias nearbyBindingValue
	Description, AreaLabel, Image      nearbyBindingValue
	Sitelinks, Article                 nearbyBindingValue
	OpenDayLabel, OpenTime, CloseTime  nearbyBindingValue
}

func (a *WikidataNearbyAPI) nearby(ctx context.Context, lat, lon, radius float64) ([]nearbyBinding, error) {
	query := fmt.Sprintf(`SELECT ?item ?coord ?type ?typeLabel ?enLabel ?ptLabel ?enAlias ?ptAlias ?description ?areaLabel ?image ?sitelinks ?article ?openDayLabel ?openTime ?closeTime WHERE {
  SERVICE wikibase:around {
    ?item wdt:P625 ?coord .
    bd:serviceParam wikibase:center "Point(%g %g)"^^geo:wktLiteral ; wikibase:radius "%g" .
  }
  OPTIONAL { ?item wdt:P31 ?type . OPTIONAL { ?type rdfs:label ?typeLabel FILTER(LANG(?typeLabel) = "en") } }
  OPTIONAL { ?item rdfs:label ?enLabel FILTER(LANG(?enLabel) = "en") }
  OPTIONAL { ?item rdfs:label ?ptLabel FILTER(LANG(?ptLabel) = "pt") }
  OPTIONAL { ?item skos:altLabel ?enAlias FILTER(LANG(?enAlias) = "en") }
  OPTIONAL { ?item skos:altLabel ?ptAlias FILTER(LANG(?ptAlias) = "pt") }
  OPTIONAL { ?item schema:description ?description FILTER(LANG(?description) = "en") }
  OPTIONAL { ?item wdt:P131/rdfs:label ?areaLabel FILTER(LANG(?areaLabel) = "en") }
  OPTIONAL { ?item wdt:P18 ?image }
  OPTIONAL { ?item wikibase:sitelinks ?sitelinks }
  OPTIONAL { ?article schema:about ?item ; schema:isPartOf <https://en.wikipedia.org/> }
  OPTIONAL { ?item p:P3025 ?hoursStatement . ?hoursStatement ps:P3025 ?openDay ; pq:P8626 ?openTimeItem ; pq:P8627 ?closeTimeItem .
    OPTIONAL { ?openDay rdfs:label ?openDayLabel FILTER(LANG(?openDayLabel) = "en") }
    OPTIONAL { ?openTimeItem rdfs:label ?openTime FILTER(LANG(?openTime) = "en") }
    OPTIONAL { ?closeTimeItem rdfs:label ?closeTime FILTER(LANG(?closeTime) = "en") }
  }
}`, lon, lat, radius)
	endpoint, _ := url.Parse(wikidataNearbyURL)
	params := endpoint.Query()
	params.Set("query", query)
	params.Set("format", "json")
	endpoint.RawQuery = params.Encode()

	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("wikidata nearby request: %w", err)
		}
		req.Header.Set("User-Agent", UserAgent)
		req.Header.Set("Accept", "application/sparql-results+json")
		resp, err := a.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("wikidata nearby: %w", err)
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			delay := nearbyRetryAfter(resp.Header.Get("Retry-After"))
			_ = resp.Body.Close()
			if attempt == 0 && delay > 0 {
				timer := time.NewTimer(delay)
				select {
				case <-timer.C:
				case <-ctx.Done():
					timer.Stop()
					return nil, ctx.Err()
				}
				continue
			}
			return nil, fmt.Errorf("wikidata nearby: status %d", resp.StatusCode)
		}
		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("wikidata nearby: status %d", resp.StatusCode)
		}
		var decoded struct {
			Results struct {
				Bindings []nearbyBinding `json:"bindings"`
			} `json:"results"`
		}
		err = json.NewDecoder(resp.Body).Decode(&decoded)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("wikidata nearby decode: %w", err)
		}
		return decoded.Results.Bindings, nil
	}
	return nil, fmt.Errorf("wikidata nearby: retries exhausted")
}

func nearbyPlaces(rows []nearbyBinding) writermap.BasePlaces {
	places := make(map[string]*writermap.Place)
	order := make([]string, 0)
	for _, row := range rows {
		qid := nearbyQID(row.Item.Value)
		lon, lat, ok := nearbyPoint(row.Coord.Value)
		if qid == "" || !ok {
			continue
		}
		place := places[qid]
		if place == nil {
			place = &writermap.Place{QID: qid, Lat: lat, Lon: lon}
			places[qid] = place
			order = append(order, qid)
		}
		place.Names = nearbyAppendUnique(place.Names, row.EnLabel.Value, row.PtLabel.Value, row.EnAlias.Value, row.PtAlias.Value)
		place.Types = nearbyAppendUnique(place.Types, nearbyQID(row.Type.Value))
		if place.Kind == "" {
			place.Kind = row.TypeLabel.Value
		}
		if place.Description == "" {
			place.Description = row.Description.Value
		}
		if place.Area == "" {
			place.Area = row.AreaLabel.Value
		}
		if place.Photo == "" && row.Image.Value != "" {
			place.Photo = nearbyImageFile(row.Image.Value)
		}
		if count, err := strconv.Atoi(row.Sitelinks.Value); err == nil {
			place.Sitelinks = count
		}
		if place.WikipediaURL == "" {
			place.WikipediaURL, _ = safeURL(row.Article.Value, "en.wikipedia.org")
		}
		if place.OpeningHours == "" && row.OpenTime.Value != "" && row.CloseTime.Value != "" {
			place.OpeningHours = nearbyHours(row.OpenDayLabel.Value, row.OpenTime.Value, row.CloseTime.Value)
		}
	}

	result := writermap.BasePlaces{Gazetteer: make([]writermap.Place, 0, len(order))}
	for _, qid := range order {
		place := *places[qid]
		if len(place.Names) == 0 {
			continue
		}
		result.Gazetteer = append(result.Gazetteer, place)
		if nearbyVisitable(place.Types) {
			result.Pins = append(result.Pins, place)
		}
	}
	sort.Slice(result.Pins, func(i, j int) bool {
		if result.Pins[i].Sitelinks != result.Pins[j].Sitelinks {
			return result.Pins[i].Sitelinks > result.Pins[j].Sitelinks
		}
		return result.Pins[i].QID < result.Pins[j].QID
	})
	if len(result.Pins) > wikidataNearbyLimit {
		result.Pins = result.Pins[:wikidataNearbyLimit]
	}
	return result
}

func nearbyVisitable(types []string) bool {
	visitable := false
	for _, qid := range types {
		kind, found := planner.PlaceTypes[qid]
		if found && kind == planner.Deny {
			return false
		}
		if found {
			visitable = true
		}
	}
	return visitable
}

func (a *WikidataNearbyAPI) addPhotoCredits(ctx context.Context, places []writermap.Place) error {
	// Commons accepts multiple file titles in one request. Resolve each file only once.
	files := make(map[string][]int)
	for i := range places {
		if places[i].Photo != "" {
			files[places[i].Photo] = append(files[places[i].Photo], i)
		}
	}
	if len(files) == 0 {
		return nil
	}
	titles := make([]string, 0, len(files))
	for file := range files {
		titles = append(titles, "File:"+file)
	}
	sort.Strings(titles)
	for start := 0; start < len(titles); start += 50 {
		end := min(start+50, len(titles))
		var response struct {
			Query struct {
				Pages []commonsPage `json:"pages"`
			} `json:"query"`
			mediaWikiError
		}
		params := url.Values{"action": {"query"}, "titles": {strings.Join(titles[start:end], "|")}, "prop": {"imageinfo"}, "iiprop": {"url|extmetadata"}, "iiextmetadatafilter": {"Artist|LicenseShortName|NonFree|Restrictions"}, "format": {"json"}, "formatversion": {"2"}}
		if err := NewWikimediaAPI(a.client).get(ctx, commonsAPIURL, params, &response); err != nil {
			return fmt.Errorf("wikidata nearby photo credit: %w", err)
		}
		if err := response.err("wikidata nearby photo credit"); err != nil {
			return err
		}
		for _, page := range response.Query.Pages {
			file := strings.TrimPrefix(page.Title, "File:")
			if len(page.ImageInfo) == 0 {
				continue
			}
			info := page.ImageInfo[0]
			artist := nearbyMetadata(info.Ext, "Artist")
			license := nearbyMetadata(info.Ext, "LicenseShortName")
			photo, photoOK := safeURL(info.URL, "upload.wikimedia.org", "thumb.wikimedia.org")
			pageURL, pageOK := safeURL(info.Description, "commons.wikimedia.org")
			if !photoOK || !pageOK || artist == "" || license == "" || strings.EqualFold(nearbyMetadata(info.Ext, "NonFree"), "true") || nearbyMetadata(info.Ext, "Restrictions") != "" {
				continue
			}
			for _, i := range files[file] {
				places[i].Photo = photo
				places[i].PhotoCredit = artist + " · " + license
				places[i].PhotoURL = pageURL
			}
		}
	}
	for i := range places {
		if places[i].PhotoCredit == "" {
			places[i].Photo = ""
		}
	}
	return nil
}

func nearbyQID(value string) string {
	value = strings.TrimPrefix(value, "http://www.wikidata.org/entity/")
	value = strings.TrimPrefix(value, "https://www.wikidata.org/entity/")
	if len(value) < 2 || value[0] != 'Q' {
		return ""
	}
	if _, err := strconv.Atoi(value[1:]); err != nil {
		return ""
	}
	return value
}

func nearbyImageFile(value string) string {
	value = strings.TrimPrefix(value, "http://commons.wikimedia.org/wiki/Special:FilePath/")
	value = strings.TrimPrefix(value, "https://commons.wikimedia.org/wiki/Special:FilePath/")
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return ""
	}
	return strings.ReplaceAll(decoded, "_", " ")
}

func nearbyPoint(value string) (lon, lat float64, ok bool) {
	if _, err := fmt.Sscanf(value, "Point(%f %f)", &lon, &lat); err != nil {
		return 0, 0, false
	}
	return lon, lat, true
}

func nearbyAppendUnique(existing []string, values ...string) []string {
	for _, value := range values {
		if value == "" {
			continue
		}
		found := false
		for _, prior := range existing {
			if value == prior {
				found = true
				break
			}
		}
		if !found {
			existing = append(existing, value)
		}
	}
	return existing
}

func nearbyHours(day, open, close string) string {
	if strings.EqualFold(day, "all days of the week") {
		day = "daily"
	}
	open = strings.TrimSuffix(open, ":00")
	close = strings.TrimSuffix(close, ":00")
	open = strings.TrimPrefix(open, "0")
	if open == "" || close == "" {
		return ""
	}
	if day == "" {
		return "Open " + open + " – " + close
	}
	return "Open " + day + " · " + open + " – " + close
}

func nearbyMetadata(ext map[string]struct {
	Value json.RawMessage `json:"value"`
}, key string) string {
	var raw string
	_ = json.Unmarshal(ext[key].Value, &raw)
	return htmlText(raw, maxCreditRunes)
}

func nearbyRetryAfter(header string) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(header)); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if date, err := http.ParseTime(header); err == nil {
		return max(time.Until(date), 0)
	}
	return 0
}
