package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
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
	// Rings of nearbyRingStep km are loaded nearest first, nearbyPageSize items per query. 100 items
	// took 8-26 s in dense rings and 3-9 MB as plain rows; 50 plus a separate label query is steadier.
	nearbyRingStep = 5
	nearbyPageSize = 50
	// nearbyMinSitelinks keeps Câmara de Lobos (2 sitelinks) and drops the long tail of 1-sitelink items.
	nearbyMinSitelinks = 2
	// A ring with fewer items than nearbyThinRing means nothing notable lies farther out.
	nearbyThinRing = 5
	// Measured 2026-10-12 (items with 2+ sitelinks within 5 km): Funchal 144, Tokyo 690, Kyoto 766,
	// Lisbon 797, New York 2,509, London 4,752, Paris 9,783. Above nearbyDenseAbove a city loads
	// nearbyDenseRings rings of nearbyDenseRing km (8 km at most) instead of 5 km rings to the radius.
	nearbyDenseAbove = 1500
	nearbyDenseRing  = 1.0
	nearbyDenseRings = 8
	nearbyNormal     = "normal"
	nearbyDense      = "dense"
	// nearbyPhotoWidth is the thumbnail width for the place panel, which shows a small photo.
	nearbyPhotoWidth = 640
)

// WikidataNearbyAPI finds nearby entities once, then selects base pins from that gazetteer.
type WikidataNearbyAPI struct {
	client *http.Client
}

var (
	_ application.BasePlaceSource = (*WikidataNearbyAPI)(nil)
	_ writermap.PagedBaseSource   = (*WikidataNearbyAPI)(nil)
)

func NewWikidataNearbyAPI(client *http.Client) *WikidataNearbyAPI {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &WikidataNearbyAPI{client: client}
}

// Places returns all nearby named entities for matching and up to 200 visitable base pins.
func (a *WikidataNearbyAPI) Places(ctx context.Context, dest writermap.Area) (writermap.BasePlaces, error) {
	loaded, err := a.PlacesFrom(ctx, dest, writermap.BaseProgress{}, nil)
	if err != nil {
		return writermap.BasePlaces{}, err
	}
	return loaded.BasePlaces, nil
}

// PlacesFrom loads 5 km rings nearest first, one page at a time, and stops once 200 visitable
// pins are loaded, a ring comes back thin, or the area radius is covered. Pages are sequential
// so Wikidata is never hit in parallel.
func (a *WikidataNearbyAPI) PlacesFrom(ctx context.Context, dest writermap.Area, from writermap.BaseProgress, onPage func(writermap.BaseProgress)) (writermap.BaseProgress, error) {
	radius := dest.RadiusKM
	if radius <= 0 {
		radius = wikidataNearbyRadius
	}
	progress := from
	for !progress.Done {
		step, limit := float64(nearbyRingStep), radius
		if progress.Mode == nearbyDense {
			step, limit = nearbyDenseRing, min(radius, nearbyDenseRing*nearbyDenseRings)
		}
		if progress.Mode == "" && progress.PageRows == 0 && (progress.Ring > 0 || progress.Offset > 0) {
			// The first page is out; now decide, once, how the rest loads.
			dense, err := a.isDense(ctx, dest.Lat, dest.Lon)
			if err != nil {
				return progress, err
			}
			progress.Mode = nearbyNormal
			if dense {
				progress.Mode, progress.Ring, progress.Offset = nearbyDense, 0, 0
			}
			continue
		}
		inner := float64(progress.Ring) * step
		if inner >= limit {
			progress.Done = true
			break
		}
		outer := min(inner+step, limit)
		if progress.PageRows == 0 {
			unnamed, fetched, err := a.pageItems(ctx, dest.Lat, dest.Lon, inner, outer, progress.Offset)
			if err != nil {
				return progress, err
			}
			progress.Unnamed, progress.PageRows = unnamed, fetched
			if len(unnamed) > 0 {
				// Show the page's pins now, without names, while the label query runs.
				progress.Pins = nearbyPins(append(slices.Clone(progress.Gazetteer), unnamed...))
				if onPage != nil {
					onPage(progress)
				}
			}
		}
		if len(progress.Unnamed) > 0 {
			rows, err := a.pageLabels(ctx, progress.Unnamed)
			if err != nil {
				return progress, err
			}
			page := nearbyPlaces(rows)
			if err := a.addPhotoCredits(ctx, page.Gazetteer); err != nil {
				if ctx.Err() != nil {
					return progress, ctx.Err()
				}
				for i := range page.Gazetteer {
					page.Gazetteer[i].Photo = ""
					page.Gazetteer[i].PhotoCredit = ""
					page.Gazetteer[i].PhotoURL = ""
				}
			}
			progress.Gazetteer = nearbyMerge(progress.Gazetteer, page.Gazetteer)
		}
		fetched := progress.PageRows
		progress.Unnamed, progress.PageRows = nil, 0
		progress.Pins = nearbyPins(progress.Gazetteer)
		progress.Offset += fetched
		ringEnded := fetched < nearbyPageSize
		thin := ringEnded && progress.Offset < nearbyThinRing
		if ringEnded {
			progress.Ring++
			progress.Offset = 0
		}
		progress.Done = len(progress.Pins) >= wikidataNearbyLimit || thin ||
			float64(progress.Ring)*step >= limit && ringEnded
		if progress.Done && progress.Mode == "" {
			progress.Mode = nearbyNormal
		}
		if onPage != nil {
			onPage(progress)
		}
	}
	if progress.Mode == "" {
		progress.Mode = nearbyNormal
	}
	return progress, nil
}

// isDense counts, up to one more than the threshold, the items with the sitelinks floor within
// 5 km of the centre. A count stops early, so it stays cheap in the biggest cities.
func (a *WikidataNearbyAPI) isDense(ctx context.Context, lat, lon float64) (bool, error) {
	var counted struct {
		Results struct {
			Bindings []struct {
				N nearbyBindingValue `json:"n"`
			} `json:"bindings"`
		} `json:"results"`
	}
	if err := a.sparql(ctx, nearbyDensityQuery(lat, lon), &counted); err != nil {
		return false, err
	}
	if len(counted.Results.Bindings) == 0 {
		return false, nil
	}
	n, err := strconv.Atoi(counted.Results.Bindings[0].N.Value)
	return err == nil && n > nearbyDenseAbove, nil
}

func nearbyDensityQuery(lat, lon float64) string {
	return fmt.Sprintf(`SELECT (COUNT(?item) AS ?n) WHERE { { SELECT ?item WHERE {
  SERVICE wikibase:around {
    ?item wdt:P625 ?coord .
    bd:serviceParam wikibase:center "Point(%g %g)"^^geo:wktLiteral ; wikibase:radius "%d" .
  }
  ?item wikibase:sitelinks ?sl . FILTER(?sl >= %d)
} LIMIT %d } }`, lon, lat, nearbyRingStep, nearbyMinSitelinks, nearbyDenseAbove+1)
}

type nearbyBindingValue struct {
	Value string `json:"value"`
}

// nearbyBinding is one place: the page query supplies Coord and Sitelinks, the label query the
// rest. Multi-valued fields (types, names, hours) arrive joined by separators.
type nearbyBinding struct {
	Item, Coord, Type, TypeLabel       nearbyBindingValue
	EnLabel, PtLabel, EnAlias, PtAlias nearbyBindingValue
	Description, AreaLabel, Image      nearbyBindingValue
	Sitelinks, Article                 nearbyBindingValue
	Hours                              nearbyBindingValue
}

// pageItems loads one page of a ring: the most-linked items first, with coordinates, sitelinks and
// types but no names. fetched is the number of page rows, which tells the caller whether the ring
// has more pages. Names come from pageLabels, a second small query, because label joins make a
// combined page slow and megabytes large in dense cities.
func (a *WikidataNearbyAPI) pageItems(ctx context.Context, lat, lon, inner, outer float64, offset int) (places []writermap.Place, fetched int, err error) {
	var page struct {
		Results struct {
			Bindings []nearbyBinding `json:"bindings"`
		} `json:"results"`
	}
	if err := a.sparql(ctx, nearbyPageQuery(lat, lon, inner, outer, offset), &page); err != nil {
		return nil, 0, err
	}
	byQID := make(map[string]*writermap.Place)
	var order []string
	// OFFSET and LIMIT count the inner query's rows, one per item and coordinate; the type join
	// repeats them, so count each (item, coordinate) once.
	subrows := make(map[[2]string]bool)
	for _, row := range page.Results.Bindings {
		subrows[[2]string{row.Item.Value, row.Coord.Value}] = true
		qid := nearbyQID(row.Item.Value)
		lon, lat, ok := nearbyPoint(row.Coord.Value)
		if qid == "" || !ok {
			continue
		}
		place := byQID[qid]
		if place == nil {
			place = &writermap.Place{QID: qid, Lat: lat, Lon: lon}
			byQID[qid] = place
			order = append(order, qid)
		}
		if count, err := strconv.Atoi(row.Sitelinks.Value); err == nil {
			place.Sitelinks = count
		}
		place.Types = nearbyAppendUnique(place.Types, nearbyQID(row.Type.Value))
	}
	for _, qid := range order {
		places = append(places, *byQID[qid])
	}
	return places, len(subrows), nil
}

// pageLabels fills in names, descriptions, photos and hours for a page's places. The rows carry
// each place's coordinates and sitelinks from the page query.
func (a *WikidataNearbyAPI) pageLabels(ctx context.Context, places []writermap.Place) ([]nearbyBinding, error) {
	qids := make([]string, len(places))
	byQID := make(map[string]writermap.Place, len(places))
	for i, place := range places {
		qids[i] = place.QID
		byQID[place.QID] = place
	}
	var labels struct {
		Results struct {
			Bindings []nearbyBinding `json:"bindings"`
		} `json:"results"`
	}
	if err := a.sparql(ctx, nearbyLabelQuery(qids), &labels); err != nil {
		return nil, err
	}
	rows := make([]nearbyBinding, 0, len(labels.Results.Bindings))
	for _, row := range labels.Results.Bindings {
		place, ok := byQID[nearbyQID(row.Item.Value)]
		if !ok {
			continue
		}
		row.Coord.Value = fmt.Sprintf("Point(%v %v)", place.Lon, place.Lat)
		row.Sitelinks.Value = strconv.Itoa(place.Sitelinks)
		rows = append(rows, row)
	}
	return rows, nil
}

// nearbyPageQuery asks for one page of items between two radii, most sitelinks first. Item order
// is stable, so OFFSET pages never repeat or skip an item.
func nearbyPageQuery(lat, lon, inner, outer float64, offset int) string {
	innerFilter := ""
	if inner > 0 {
		innerFilter = fmt.Sprintf("FILTER(?dist > %g)", inner)
	}
	return fmt.Sprintf(`SELECT ?item ?coord ?sitelinks ?type WHERE {
  { SELECT ?item ?coord ?sitelinks WHERE {
    SERVICE wikibase:around {
      ?item wdt:P625 ?coord .
      bd:serviceParam wikibase:center "Point(%g %g)"^^geo:wktLiteral ; wikibase:radius "%g" ; wikibase:distance ?dist .
    }
    ?item wikibase:sitelinks ?sitelinks . FILTER(?sitelinks >= %d) %s
  } ORDER BY DESC(?sitelinks) ?item LIMIT %d OFFSET %d }
  OPTIONAL { ?item wdt:P31 ?type }
}`, lon, lat, outer, nearbyMinSitelinks, innerFilter, nearbyPageSize, offset)
}

// nearbyLabelQuery fetches one grouped row per item, so a page stays small however many aliases
// or types an item has.
func nearbyLabelQuery(qids []string) string {
	values := make([]string, len(qids))
	for i, qid := range qids {
		values[i] = "wd:" + qid
	}
	return fmt.Sprintf(`SELECT ?item
  (GROUP_CONCAT(DISTINCT STR(?typeValue); separator="\u001F") AS ?type)
  (SAMPLE(?typeLabelValue) AS ?typeLabel)
  (GROUP_CONCAT(DISTINCT STR(?enLabelValue); separator="\u001F") AS ?enLabel)
  (GROUP_CONCAT(DISTINCT STR(?ptLabelValue); separator="\u001F") AS ?ptLabel)
  (GROUP_CONCAT(DISTINCT STR(?enAliasValue); separator="\u001F") AS ?enAlias)
  (GROUP_CONCAT(DISTINCT STR(?ptAliasValue); separator="\u001F") AS ?ptAlias)
  (SAMPLE(?descriptionValue) AS ?description)
  (SAMPLE(?areaLabelValue) AS ?areaLabel)
  (SAMPLE(?imageValue) AS ?image)
  (SAMPLE(?articleValue) AS ?article)
  (GROUP_CONCAT(DISTINCT IF(BOUND(?openTimeValue) && BOUND(?closeTimeValue), CONCAT(COALESCE(STR(?openDayLabelValue), ""), "\u001E", STR(?openTimeValue), "\u001E", STR(?closeTimeValue)), ""); separator="\u001F") AS ?hours)
WHERE {
  VALUES ?item { %s }
  OPTIONAL { ?item wdt:P31 ?typeValue . OPTIONAL { ?typeValue rdfs:label ?typeLabelValue FILTER(LANG(?typeLabelValue) = "en") } }
  OPTIONAL { ?item rdfs:label ?enLabelValue FILTER(LANG(?enLabelValue) = "en") }
  OPTIONAL { ?item rdfs:label ?ptLabelValue FILTER(LANG(?ptLabelValue) = "pt") }
  OPTIONAL { ?item skos:altLabel ?enAliasValue FILTER(LANG(?enAliasValue) = "en") }
  OPTIONAL { ?item skos:altLabel ?ptAliasValue FILTER(LANG(?ptAliasValue) = "pt") }
  OPTIONAL { ?item schema:description ?descriptionValue FILTER(LANG(?descriptionValue) = "en") }
  OPTIONAL { ?item wdt:P131/rdfs:label ?areaLabelValue FILTER(LANG(?areaLabelValue) = "en") }
  OPTIONAL { ?item wdt:P18 ?imageValue }
  OPTIONAL { ?articleValue schema:about ?item ; schema:isPartOf <https://en.wikipedia.org/> }
  OPTIONAL { ?item p:P3025 ?hoursStatement . ?hoursStatement ps:P3025 ?openDay ; pq:P8626 ?openTimeItem ; pq:P8627 ?closeTimeItem .
    OPTIONAL { ?openDay rdfs:label ?openDayLabelValue FILTER(LANG(?openDayLabelValue) = "en") }
    OPTIONAL { ?openTimeItem rdfs:label ?openTimeValue FILTER(LANG(?openTimeValue) = "en") }
    OPTIONAL { ?closeTimeItem rdfs:label ?closeTimeValue FILTER(LANG(?closeTimeValue) = "en") }
  }
} GROUP BY ?item`, strings.Join(values, " "))
}

// sparql runs one query, retrying once after a 429 or 503 that says how long to wait.
func (a *WikidataNearbyAPI) sparql(ctx context.Context, query string, into any) error {
	endpoint, _ := url.Parse(wikidataNearbyURL)
	params := endpoint.Query()
	params.Set("query", query)
	params.Set("format", "json")
	endpoint.RawQuery = params.Encode()

	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
		if err != nil {
			return fmt.Errorf("wikidata nearby request: %w", err)
		}
		req.Header.Set("User-Agent", UserAgent)
		req.Header.Set("Accept", "application/sparql-results+json")
		resp, err := a.client.Do(req)
		if err != nil {
			return fmt.Errorf("wikidata nearby: %w", err)
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
					return ctx.Err()
				}
				continue
			}
			return fmt.Errorf("wikidata nearby: status %d", resp.StatusCode)
		}
		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			return fmt.Errorf("wikidata nearby: status %d", resp.StatusCode)
		}
		err = json.NewDecoder(resp.Body).Decode(into)
		_ = resp.Body.Close()
		if err != nil {
			return fmt.Errorf("wikidata nearby decode: %w", err)
		}
		return nil
	}
	return fmt.Errorf("wikidata nearby: retries exhausted")
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
		for _, names := range []string{row.EnLabel.Value, row.PtLabel.Value, row.EnAlias.Value, row.PtAlias.Value} {
			place.Names = nearbyAppendUnique(place.Names, nearbySplit(names)...)
		}
		for _, entity := range nearbySplit(row.Type.Value) {
			place.Types = nearbyAppendUnique(place.Types, nearbyQID(entity))
		}
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
		if place.OpeningHours == "" {
			var hours []string
			for _, encoded := range nearbySplit(row.Hours.Value) {
				parts := strings.Split(encoded, "\x1e")
				if len(parts) == 3 {
					hours = nearbyAppendUnique(hours, nearbyHours(parts[0], parts[1], parts[2]))
				}
			}
			place.OpeningHours = strings.Join(hours, "; ")
		}
	}

	result := writermap.BasePlaces{Gazetteer: make([]writermap.Place, 0, len(order))}
	for _, qid := range order {
		place := *places[qid]
		if len(place.Names) == 0 {
			continue
		}
		result.Gazetteer = append(result.Gazetteer, place)
	}
	result.Pins = nearbyPins(result.Gazetteer)
	return result
}

// nearbyMerge appends the places whose QID is not already in the gazetteer.
func nearbyMerge(gazetteer, page []writermap.Place) []writermap.Place {
	known := make(map[string]bool, len(gazetteer))
	for _, place := range gazetteer {
		known[place.QID] = true
	}
	for _, place := range page {
		if !known[place.QID] {
			known[place.QID] = true
			gazetteer = append(gazetteer, place)
		}
	}
	return gazetteer
}

// nearbyPins picks up to 200 visitable places, most sitelinks first.
func nearbyPins(gazetteer []writermap.Place) []writermap.Place {
	var pins []writermap.Place
	for _, place := range gazetteer {
		if nearbyVisitable(place.Types) {
			pins = append(pins, place)
		}
	}
	sort.Slice(pins, func(i, j int) bool {
		if pins[i].Sitelinks != pins[j].Sitelinks {
			return pins[i].Sitelinks > pins[j].Sitelinks
		}
		return pins[i].QID < pins[j].QID
	})
	if len(pins) > wikidataNearbyLimit {
		pins = pins[:wikidataNearbyLimit]
	}
	return pins
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
		params := url.Values{"action": {"query"}, "titles": {strings.Join(titles[start:end], "|")}, "prop": {"imageinfo"}, "iiprop": {"url|extmetadata"}, "iiurlwidth": {fmt.Sprint(nearbyPhotoWidth)}, "iiextmetadatafilter": {"Artist|LicenseShortName|NonFree|Restrictions"}, "format": {"json"}, "formatversion": {"2"}}
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
			// The thumbnail is absent when the original is narrower than the request.
			imageURL := info.ThumbURL
			if imageURL == "" {
				imageURL = info.URL
			}
			photoURL, photoOK := safeURL(imageURL, "upload.wikimedia.org", "thumb.wikimedia.org")
			if _, pageOK := safeURL(info.Description, "commons.wikimedia.org"); !photoOK || !pageOK || artist == "" || license == "" || strings.EqualFold(nearbyMetadata(info.Ext, "NonFree"), "true") || nearbyMetadata(info.Ext, "Restrictions") != "" {
				continue
			}
			for _, i := range files[file] {
				places[i].PhotoCredit = artist + " · " + license
				places[i].PhotoURL = photoURL
			}
		}
	}
	for i := range places {
		if places[i].PhotoURL == "" {
			places[i].Photo = ""
			places[i].PhotoCredit = ""
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

func nearbySplit(value string) []string {
	return strings.Split(value, "\x1f")
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
