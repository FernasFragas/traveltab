package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
	"weatherservice/internal/application"
	"weatherservice/internal/planner"

	"golang.org/x/net/html"
)

const (
	commonsAPIURL = "https://commons.wikimedia.org/w/api.php"

	// destinationPhotoTimeout bounds one whole lookup, every request in it included.
	destinationPhotoTimeout = 4 * time.Second

	// cityCandidateLimit and imageCandidateLimit keep discovery bounded: at most five entities
	// are examined and at most three image files are resolved for the one that is selected.
	cityCandidateLimit  = 5
	imageCandidateLimit = 3

	// photoThumbWidth is the thumbnail width requested from Commons, a little above the widest
	// hero the page draws. photoMinWidth rejects originals too small to fill it.
	photoThumbWidth = 1280
	photoMinWidth   = 800
	// landscapeRatio is the width/height a photograph must reach to be preferred for the wide hero.
	landscapeRatio = 1.2

	// nearbyRadiusM and nearbyPages bound the geographic fallback search (10 km is the API's limit).
	nearbyRadiusM = 10000
	nearbyPages   = 50

	// ambiguousKM is how far apart two equally plausible entities for one name may be before the
	// lookup refuses to choose between them.
	ambiguousKM = 10

	maxCreditRunes = 120

	wikidataGlobeEarth = "http://www.wikidata.org/entity/Q2"
)

// settlementTypes are the "instance of" classes that mark an entity as a place people live in.
// Any entity with a population statement counts as well, so national subclasses of these need
// not be listed.
var settlementTypes = []string{
	"Q515", "Q1549591", "Q3957", "Q532", "Q5119", "Q486972", "Q5084", "Q15284", "Q1637706",
	"Q702492", "Q13217644", "Q1093829",
}

// unwantedFileName rejects Commons files that depict a symbol or a diagram of a place instead of
// the place. It is checked against the file title, so it is deliberately word-based.
var unwantedFileName = regexp.MustCompile(`(?i)(^|[^a-z])(flag|flagge|bandera|drapeau|bandeira|logo|map|mapa|carte|karte|locator|location|coat[ _-]?of[ _-]?arms|seal|emblem|blason|wappen|escudo|brasao|icon|diagram|plan|montage|collage)([^a-z]|$)`)

var photoMIMETypes = []string{"image/jpeg", "image/png", "image/webp"}

// DestinationPhotoAPI finds a photograph of a city in Wikidata and Wikimedia Commons. It needs no
// key. The city entity comes from the name, and is accepted only when its coordinates and country
// agree with the resolved destination, so a namesake city never lends its picture.
type DestinationPhotoAPI struct {
	wiki    *WikimediaAPI
	timeout time.Duration

	mu        sync.Mutex
	countries map[string]string // country entity ID -> upper-case ISO code, "" when it has none
}

// NewDestinationPhotoAPI returns a photo source over the Wikimedia APIs. A nil client means a
// default one with the Wikimedia timeout; each lookup is also bounded by destinationPhotoTimeout.
func NewDestinationPhotoAPI(client *http.Client) *DestinationPhotoAPI {
	return &DestinationPhotoAPI{wiki: NewWikimediaAPI(client), timeout: destinationPhotoTimeout, countries: make(map[string]string)}
}

// cityMatch is a Wikidata entity accepted as the destination.
type cityMatch struct {
	ID         string
	Lat, Lon   float64
	DistanceKM float64
	Populated  bool
	Sitelinks  int
	Images     []string // P18 file names, best statements first
	Article    string   // English Wikipedia title, "" when it has none
}

// Photo returns an attributed photograph of the destination, (nil, nil) when it has a matching
// city entity but no image that can be shown or no unambiguous entity at all, and an error when a
// provider could not answer.
func (a *DestinationPhotoAPI) Photo(ctx context.Context, id application.DestinationIdentity) (*application.DestinationPhoto, error) {
	normalized, ok := application.NewDestinationIdentity(id.City, id.Country, id.Lat, id.Lon)
	if !ok {
		return nil, fmt.Errorf("destination photo: unusable identity %+v", id)
	}

	id = normalized

	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	candidates, err := a.searchCities(ctx, id.City)
	if err != nil {
		return nil, err
	}

	city, err := a.matchCity(ctx, id, candidates)
	if err != nil {
		return nil, err
	}

	if city == nil {
		// The name search returns only the best-known entities for a name, so a smaller
		// namesake can be missing. Articles near the coordinates find it.
		if candidates, err = a.nearbyCities(ctx, id); err != nil {
			return nil, err
		}

		if city, err = a.matchCity(ctx, id, candidates); err != nil {
			return nil, err
		}
	}

	if city == nil {
		return nil, nil
	}

	return a.photoFor(ctx, id, *city)
}

// cityCandidate is an entity a search proposed, with the names the search matched it by.
type cityCandidate struct {
	ID    string
	Names []string
}

// searchCities returns the entities whose label or alias, in any language, equals the name up to
// case and diacritics. That is how "Lisboa" and "München" find Lisbon and Munich.
func (a *DestinationPhotoAPI) searchCities(ctx context.Context, name string) ([]cityCandidate, error) {
	params := url.Values{
		"action":   {"wbsearchentities"},
		"search":   {name},
		"language": {"en"},
		"type":     {"item"},
		"limit":    {fmt.Sprint(cityCandidateLimit)},
		"format":   {"json"},
	}

	var response struct {
		Search []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
			Match struct {
				Text string `json:"text"`
			} `json:"match"`
		} `json:"search"`
		mediaWikiError
	}

	if err := a.wiki.get(ctx, wikidataAPIURL, params, &response); err != nil {
		return nil, fmt.Errorf("wikidata search: %w", err)
	}

	if err := response.err("wikidata search"); err != nil {
		return nil, err
	}

	want := application.FoldName(name)

	var found []cityCandidate

	for _, hit := range response.Search {
		names := []string{hit.Label, hit.Match.Text}
		if hit.ID != "" && slices.ContainsFunc(names, func(n string) bool { return application.FoldName(n) == want }) {
			found = append(found, cityCandidate{ID: hit.ID, Names: names})
		}
	}

	return found, nil
}

// nearbyCities returns the entities of English Wikipedia articles within 10 km of the destination
// whose title, before any ", region" or "(disambiguation)", is the city name.
func (a *DestinationPhotoAPI) nearbyCities(ctx context.Context, id application.DestinationIdentity) ([]cityCandidate, error) {
	params := url.Values{
		"action":        {"query"},
		"generator":     {"geosearch"},
		"ggscoord":      {fmt.Sprintf("%f|%f", id.Lat, id.Lon)},
		"ggsradius":     {fmt.Sprint(nearbyRadiusM)},
		"ggslimit":      {fmt.Sprint(nearbyPages)},
		"prop":          {"pageprops"},
		"ppprop":        {"wikibase_item"},
		"format":        {"json"},
		"formatversion": {"2"},
	}

	var response struct {
		Query struct {
			Pages []struct {
				Title     string `json:"title"`
				PageProps struct {
					WikibaseItem string `json:"wikibase_item"`
				} `json:"pageprops"`
			} `json:"pages"`
		} `json:"query"`
		mediaWikiError
	}

	if err := a.wiki.get(ctx, wikipediaAPIURL, params, &response); err != nil {
		return nil, fmt.Errorf("wikipedia nearby articles: %w", err)
	}

	if err := response.err("wikipedia nearby articles"); err != nil {
		return nil, err
	}

	want := application.FoldName(id.City)

	var found []cityCandidate

	for _, page := range response.Query.Pages {
		base, _, _ := strings.Cut(page.Title, ",")
		base, _, _ = strings.Cut(base, " (")

		if page.PageProps.WikibaseItem != "" && application.FoldName(base) == want && len(found) < cityCandidateLimit {
			found = append(found, cityCandidate{ID: page.PageProps.WikibaseItem, Names: []string{base}})
		}
	}

	return found, nil
}

// matchCity loads the candidates, which already carry the destination's name, and returns the one
// entity that is the destination: it is a place people live in, lies within application.DestinationMatchKM and belongs to the
// destination's country. nil means none qualifies or the choice is ambiguous.
func (a *DestinationPhotoAPI) matchCity(ctx context.Context, id application.DestinationIdentity, candidates []cityCandidate) (*cityMatch, error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	ids := make([]string, len(candidates))
	for i, candidate := range candidates {
		ids[i] = candidate.ID
	}

	entities, err := a.wiki.GetEntities(ctx, ids, "claims|sitelinks")
	if err != nil {
		return nil, err
	}

	var matches []cityMatch

	for _, candidate := range candidates {
		entity, ok := entities[candidate.ID]
		if !ok {
			continue
		}

		match, ok := entityLocation(entity, id)
		if !ok {
			continue
		}

		inCountry, err := a.inCountry(ctx, entity, id.Country)
		if err != nil {
			return nil, err
		}

		if !inCountry {
			continue
		}

		match.ID = candidate.ID
		matches = append(matches, match)
	}

	return chooseCity(matches), nil
}

// entityLocation checks that the entity is a populated place near the destination.
func entityLocation(entity Entity, id application.DestinationIdentity) (cityMatch, bool) {
	match := cityMatch{Populated: len(entity.Claims["P1082"]) > 0, Sitelinks: len(entity.Sitelinks)}

	if !match.Populated && !slices.ContainsFunc(bestClaims(entity.Claims["P31"]), func(claim EntityClaim) bool {
		return slices.Contains(settlementTypes, claimItemID(claim))
	}) {
		return match, false
	}

	located := false

	for _, claim := range bestClaims(entity.Claims["P625"]) {
		var point struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Globe     string  `json:"globe"`
		}

		if json.Unmarshal(claim.MainSnak.DataValue.Value, &point) != nil || point.Globe != wikidataGlobeEarth {
			continue
		}

		match.Lat, match.Lon, located = point.Latitude, point.Longitude, true

		break
	}

	if !located {
		return match, false
	}

	match.DistanceKM = planner.DistanceKM(planner.Place{Lat: id.Lat, Lon: id.Lon}, planner.Place{Lat: match.Lat, Lon: match.Lon})
	if match.DistanceKM > application.DestinationMatchKM {
		return match, false
	}

	for _, claim := range bestClaims(entity.Claims["P18"]) {
		var file string
		if json.Unmarshal(claim.MainSnak.DataValue.Value, &file) == nil && file != "" {
			match.Images = append(match.Images, file)
		}
	}

	var link struct {
		Title string `json:"title"`
	}

	_ = json.Unmarshal(entity.Sitelinks["enwiki"], &link)
	match.Article = link.Title

	return match, true
}

// inCountry reports whether one of the entity's countries has the ISO code. An entity that names
// no country cannot be confirmed and is refused.
func (a *DestinationPhotoAPI) inCountry(ctx context.Context, entity Entity, iso string) (bool, error) {
	for _, claim := range bestClaims(entity.Claims["P17"]) {
		qid := claimItemID(claim)
		if qid == "" {
			continue
		}

		code, err := a.countryCode(ctx, qid)
		if err != nil {
			return false, err
		}

		if code == iso {
			return true, nil
		}
	}

	return false, nil
}

// countryCode reads a country entity's ISO 3166-1 alpha-2 code (P297). Codes never change, so
// they are kept for the life of the process.
func (a *DestinationPhotoAPI) countryCode(ctx context.Context, qid string) (string, error) {
	a.mu.Lock()
	code, ok := a.countries[qid]
	a.mu.Unlock()

	if ok {
		return code, nil
	}

	params := url.Values{
		"action":   {"wbgetclaims"},
		"entity":   {qid},
		"property": {"P297"},
		"format":   {"json"},
	}

	var response struct {
		Claims map[string][]EntityClaim `json:"claims"`
		mediaWikiError
	}

	if err := a.wiki.get(ctx, wikidataAPIURL, params, &response); err != nil {
		return "", fmt.Errorf("wikidata country %s: %w", qid, err)
	}

	if err := response.err("wikidata country " + qid); err != nil {
		return "", err
	}

	for _, claim := range bestClaims(response.Claims["P297"]) {
		var value string
		if json.Unmarshal(claim.MainSnak.DataValue.Value, &value) == nil {
			code = strings.ToUpper(strings.TrimSpace(value))

			break
		}
	}

	a.mu.Lock()
	a.countries[qid] = code
	a.mu.Unlock()

	return code, nil
}

// chooseCity picks the most established of the matching entities: populated places first, then the
// one with the most language editions, then the closest. Two comparable entities more than
// ambiguousKM apart are two different places sharing a name, so it declines to choose.
func chooseCity(matches []cityMatch) *cityMatch {
	if len(matches) == 0 {
		return nil
	}

	sort.SliceStable(matches, func(i, j int) bool {
		a, b := matches[i], matches[j]

		switch {
		case a.Populated != b.Populated:
			return a.Populated
		case a.Sitelinks != b.Sitelinks:
			return a.Sitelinks > b.Sitelinks
		case a.DistanceKM != b.DistanceKM:
			return a.DistanceKM < b.DistanceKM
		default:
			return a.ID < b.ID
		}
	})

	if len(matches) > 1 && matches[0].Populated == matches[1].Populated &&
		matches[0].Sitelinks < 2*matches[1].Sitelinks &&
		planner.DistanceKM(planner.Place{Lat: matches[0].Lat, Lon: matches[0].Lon}, planner.Place{Lat: matches[1].Lat, Lon: matches[1].Lon}) > ambiguousKM {
		return nil
	}

	return &matches[0]
}

// photoFor resolves the entity's own photographs first and its Wikipedia article's page image
// only when none of those can be shown, examining imageCandidateLimit files at most.
func (a *DestinationPhotoAPI) photoFor(ctx context.Context, id application.DestinationIdentity, city cityMatch) (*application.DestinationPhoto, error) {
	files := city.Images
	if len(files) > imageCandidateLimit-1 {
		files = files[:imageCandidateLimit-1]
	}

	photo, err := a.usablePhoto(ctx, id, city, files)
	if photo != nil || err != nil || city.Article == "" {
		return photo, err
	}

	file, err := a.articleImage(ctx, city.Article)
	if err != nil || file == "" || slices.Contains(files, file) {
		return nil, err
	}

	return a.usablePhoto(ctx, id, city, []string{file})
}

// articleImage is the free image English Wikipedia picked to illustrate the article.
func (a *DestinationPhotoAPI) articleImage(ctx context.Context, title string) (string, error) {
	params := url.Values{
		"action":        {"query"},
		"titles":        {title},
		"prop":          {"pageimages"},
		"piprop":        {"name"},
		"redirects":     {"1"},
		"format":        {"json"},
		"formatversion": {"2"},
	}

	var response struct {
		Query struct {
			Pages []struct {
				PageImage string `json:"pageimage"`
			} `json:"pages"`
		} `json:"query"`
		mediaWikiError
	}

	if err := a.wiki.get(ctx, wikipediaAPIURL, params, &response); err != nil {
		return "", fmt.Errorf("wikipedia page image: %w", err)
	}

	if err := response.err("wikipedia page image"); err != nil {
		return "", err
	}

	for _, page := range response.Query.Pages {
		if page.PageImage != "" {
			return page.PageImage, nil
		}
	}

	return "", nil
}

// commonsPage is one file's Commons image information.
type commonsPage struct {
	Title     string `json:"title"`
	Missing   bool   `json:"missing"`
	ImageInfo []struct {
		ThumbURL    string `json:"thumburl"`
		ThumbWidth  int    `json:"thumbwidth"`
		ThumbHeight int    `json:"thumbheight"`
		URL         string `json:"url"`
		Width       int    `json:"width"`
		Height      int    `json:"height"`
		Description string `json:"descriptionurl"`
		MIME        string `json:"mime"`
		Ext         map[string]struct {
			Value json.RawMessage `json:"value"`
		} `json:"extmetadata"`
	} `json:"imageinfo"`
}

// usablePhoto asks Commons about the files together and returns the best one that can be shown,
// preferring landscape images and keeping the files' order otherwise.
func (a *DestinationPhotoAPI) usablePhoto(ctx context.Context, id application.DestinationIdentity, city cityMatch, files []string) (*application.DestinationPhoto, error) {
	if len(files) == 0 {
		return nil, nil
	}

	titles := make([]string, len(files))
	for i, file := range files {
		titles[i] = "File:" + strings.ReplaceAll(file, "_", " ")
	}

	params := url.Values{
		"action":                {"query"},
		"titles":                {strings.Join(titles, "|")},
		"prop":                  {"imageinfo"},
		"iiprop":                {"url|size|mime|extmetadata"},
		"iiurlwidth":            {fmt.Sprint(photoThumbWidth)},
		"iiextmetadatafilter":   {"Artist|LicenseShortName|LicenseUrl|NonFree|Restrictions"},
		"iiextmetadatalanguage": {"en"},
		"redirects":             {"1"},
		"format":                {"json"},
		"formatversion":         {"2"},
	}

	var response struct {
		Query struct {
			Normalized []titleChange `json:"normalized"`
			Redirects  []titleChange `json:"redirects"`
			Pages      []commonsPage `json:"pages"`
		} `json:"query"`
		mediaWikiError
	}

	if err := a.wiki.get(ctx, commonsAPIURL, params, &response); err != nil {
		return nil, fmt.Errorf("commons image information: %w", err)
	}

	if err := response.err("commons image information"); err != nil {
		return nil, err
	}

	pages := make(map[string]commonsPage, len(response.Query.Pages))
	for _, page := range response.Query.Pages {
		pages[page.Title] = page
	}

	var first, landscape *application.DestinationPhoto

	for _, title := range titles {
		title = followTitle(response.Query.Redirects, followTitle(response.Query.Normalized, title))

		photo, ratio := commonsPhoto(pages[title], id, city)
		if photo == nil {
			continue
		}

		if first == nil {
			first = photo
		}

		if ratio >= landscapeRatio && landscape == nil {
			landscape = photo
		}
	}

	if landscape != nil {
		return landscape, nil
	}

	return first, nil
}

type titleChange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func followTitle(changes []titleChange, title string) string {
	for _, change := range changes {
		if change.From == title {
			return change.To
		}
	}

	return title
}

// commonsPhoto turns a file into a photograph of the destination, or nil when it is not one that
// can be shown: not a bitmap photograph, too small or portrait, named like a symbol or map,
// non-free or restricted, or missing an author, a license or trustworthy URLs.
func commonsPhoto(page commonsPage, id application.DestinationIdentity, city cityMatch) (*application.DestinationPhoto, float64) {
	if page.Missing || len(page.ImageInfo) == 0 || unwantedFileName.MatchString(strings.TrimPrefix(page.Title, "File:")) {
		return nil, 0
	}

	info := page.ImageInfo[0]

	if !slices.Contains(photoMIMETypes, info.MIME) || info.Width < photoMinWidth || info.Height <= 0 || info.Width < info.Height {
		return nil, 0
	}

	meta := func(key string) string {
		var value string
		if err := json.Unmarshal(info.Ext[key].Value, &value); err != nil {
			return ""
		}

		return strings.TrimSpace(value)
	}

	// The thumbnail is absent when the original is narrower than the request; use the original.
	imageURL, width, height := info.ThumbURL, info.ThumbWidth, info.ThumbHeight
	if imageURL == "" {
		imageURL, width, height = info.URL, info.Width, info.Height
	}

	credit := htmlText(meta("Artist"), maxCreditRunes)
	license := htmlText(meta("LicenseShortName"), maxCreditRunes)

	licenseURL, _ := safeURL(meta("LicenseUrl"))
	imageURL, imageOK := safeURL(imageURL, "upload.wikimedia.org", "thumb.wikimedia.org")
	creditURL, creditOK := safeURL(info.Description, "commons.wikimedia.org")

	if !imageOK || !creditOK || width <= 0 || height <= 0 || credit == "" || license == "" ||
		strings.EqualFold(meta("NonFree"), "true") || meta("Restrictions") != "" {
		return nil, 0
	}

	return &application.DestinationPhoto{
		URL:        imageURL,
		Alt:        "View of " + id.City,
		Credit:     credit,
		CreditURL:  creditURL,
		License:    license,
		LicenseURL: licenseURL,
		Width:      width,
		Height:     height,
		SourceID:   city.ID,
	}, float64(info.Width) / float64(info.Height)
}

// safeURL accepts an absolute https URL, on one of the hosts when any are given. Provider text
// becomes an href or src, so anything else, scripts and data URLs included, is dropped.
func safeURL(raw string, hosts ...string) (string, bool) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return "", false
	}

	if len(hosts) > 0 && !slices.Contains(hosts, strings.ToLower(parsed.Hostname())) {
		return "", false
	}

	return parsed.String(), true
}

// htmlText reduces provider HTML, such as an author credit with links, to its text. The result is
// plain text for the template to escape; the markup is never passed on.
func htmlText(markup string, maxRunes int) string {
	var text strings.Builder

	tokens := html.NewTokenizer(strings.NewReader(markup))
	skipping := 0

	for {
		switch tokens.Next() {
		case html.ErrorToken:
			return truncate(strings.Join(strings.Fields(text.String()), " "), maxRunes)
		case html.TextToken:
			if skipping == 0 {
				text.Write(tokens.Text())
			}
		case html.StartTagToken, html.EndTagToken, html.SelfClosingTagToken:
			tag := tokens.Token()

			switch {
			case tag.Data != "script" && tag.Data != "style":
				text.WriteByte(' ') // tags such as br and li separate words
			case tag.Type == html.StartTagToken:
				skipping++
			case tag.Type == html.EndTagToken && skipping > 0:
				skipping--
			}
		}
	}
}

func truncate(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}

	return strings.TrimSpace(string([]rune(s)[:maxRunes])) + "…"
}

// bestClaims keeps a property's preferred statements when it has any, and otherwise its normal
// ones. Deprecated statements are never used.
func bestClaims(claims []EntityClaim) []EntityClaim {
	var preferred, normal []EntityClaim

	for _, claim := range claims {
		switch claim.Rank {
		case "preferred":
			preferred = append(preferred, claim)
		case "deprecated":
		default:
			normal = append(normal, claim)
		}
	}

	if len(preferred) > 0 {
		return preferred
	}

	return normal
}

func claimItemID(claim EntityClaim) string {
	var item struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(claim.MainSnak.DataValue.Value, &item); err != nil {
		return ""
	}

	return item.ID
}

// mediaWikiError is embedded in a response to catch the error object MediaWiki sends with a 200.
type mediaWikiError struct {
	Error *struct {
		Code string `json:"code"`
		Info string `json:"info"`
	} `json:"error"`
}

func (e mediaWikiError) err(what string) error {
	if e.Error == nil {
		return nil
	}

	return errors.New(what + ": " + e.Error.Code + ": " + e.Error.Info)
}
