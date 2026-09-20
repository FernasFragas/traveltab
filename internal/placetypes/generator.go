package placetypes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"sort"

	"weatherservice/internal/adapters/api"
	"weatherservice/internal/planner"
)

// city is one of the places the type list is learned from.
type city struct {
	Name     string
	Lat, Lon float64
}

// cities are the 20 cities from the task: five the validation covered, and fifteen more that
// bring in other cultures and other kinds of place.
var cities = []city{
	{"Lisbon", 38.7223, -9.1393},
	{"Porto", 41.1579, -8.6291},
	{"Tavira", 37.1264, -7.6506},
	{"Funchal", 32.6669, -16.9241},
	{"Kyoto", 35.0116, 135.7681},
	{"Paris", 48.8566, 2.3522},
	{"Rome", 41.9028, 12.4964},
	{"Barcelona", 41.3874, 2.1686},
	{"London", 51.5074, -0.1278},
	{"New York", 40.7128, -74.0060},
	{"Tokyo", 35.6762, 139.6503},
	{"Istanbul", 41.0082, 28.9784},
	{"Prague", 50.0755, 14.4378},
	{"Amsterdam", 52.3676, 4.9041},
	{"Berlin", 52.5200, 13.4050},
	{"Seville", 37.3891, -5.9845},
	{"Florence", 43.7696, 11.2558},
	{"Vienna", 48.2082, 16.3738},
	{"Marrakesh", 31.6295, -7.9811},
	{"Mexico City", 19.4326, -99.1332},
}

// summarySize is how many kept and dropped types the review summary prints.
const summarySize = 30

// classEntry is what one Wikidata type looks like from here: its English label and the types it
// is a subclass of.
type classEntry struct {
	Label   string   `json:"label"`
	Parents []string `json:"parents"`
}

// cache is what the tool keeps between runs: the types of the places found in each city, and
// the part of the class graph it has walked.
type cache struct {
	// Cities maps a city name to the P31 types of each place found around it, by place ID.
	Cities map[string]map[string][]string `json:"cities"`
	// Classes maps a type ID to its label and its subclass-of parents.
	Classes map[string]classEntry `json:"classes"`

	path string
}

// Run does the whole job: fetch, apply the rules, write the file and print the summary.
func Run(ctx context.Context, cachePath, out string) error {
	client := api.NewWikimediaAPI(nil)

	store, err := loadCache(cachePath)
	if err != nil {
		return err
	}

	log.Printf("cache: %s (%d cities, %d types already fetched)", cachePath, len(store.Cities), len(store.Classes))

	counts, err := collectTypes(ctx, client, store)
	if err != nil {
		return err
	}

	rules := DefaultRules()

	graph, labels, err := walkClasses(ctx, client, rules, counts, store)
	if err != nil {
		return err
	}

	types := rules.Classify(counts, labels, graph)

	source, err := Render(types)
	if err != nil {
		return err
	}

	if err := os.WriteFile(out, source, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}

	log.Printf("wrote %s", out)

	printSummary(types)

	return nil
}

// collectTypes counts how many places had each P31 type, over all 20 cities. A place that shows
// up in two cities is counted once.
func collectTypes(ctx context.Context, client *api.WikimediaAPI, store *cache) (map[string]int, error) {
	for _, c := range cities {
		if _, ok := store.Cities[c.Name]; ok {
			continue
		}

		log.Printf("fetching places around %s (%.4f, %.4f)", c.Name, c.Lat, c.Lon)

		places, err := client.PlacesNear(ctx, c.Lat, c.Lon)
		if err != nil {
			return nil, fmt.Errorf("places near %s: %w", c.Name, err)
		}

		found := make(map[string][]string, len(places))

		for _, place := range places {
			if len(place.Types) > 0 {
				found[place.ID] = place.Types
			}
		}

		log.Printf("%s: %d places, %d of them with types", c.Name, len(places), len(found))

		store.Cities[c.Name] = found

		if err := store.save(); err != nil {
			return nil, err
		}
	}

	counts := map[string]int{}
	seen := map[string]bool{}

	for _, c := range cities {
		for placeID, types := range store.Cities[c.Name] {
			if seen[placeID] {
				continue
			}

			seen[placeID] = true

			for _, typeID := range types {
				counts[typeID]++
			}
		}
	}

	log.Printf("%d places carry %d different types", len(seen), len(counts))

	return counts, nil
}

// walkClasses follows subclass-of upward from every type the cities had, and from the roots and
// denied types, MaxDepth levels deep. It returns the class graph and the English labels.
func walkClasses(
	ctx context.Context,
	client *api.WikimediaAPI,
	rules Rules,
	counts map[string]int,
	store *cache,
) (ClassGraph, map[string]string, error) {
	frontier := make([]string, 0, len(counts))

	for id := range counts {
		frontier = append(frontier, id)
	}

	for id := range rules.Roots {
		frontier = append(frontier, id)
	}

	for id := range rules.Deny {
		frontier = append(frontier, id)
	}

	frontier = unique(frontier)
	visited := map[string]bool{}

	for depth := 0; depth <= MaxDepth && len(frontier) > 0; depth++ {
		var unknown []string

		for _, id := range frontier {
			visited[id] = true

			if _, ok := store.Classes[id]; !ok {
				unknown = append(unknown, id)
			}
		}

		if len(unknown) > 0 {
			log.Printf("depth %d: fetching %d of %d types", depth, len(unknown), len(frontier))

			sort.Strings(unknown)

			entities, err := client.GetEntities(ctx, unknown, "claims|labels")
			if err != nil {
				return nil, nil, fmt.Errorf("walking subclass-of at depth %d: %w", depth, err)
			}

			for _, id := range unknown {
				entity := entities[id]
				store.Classes[id] = classEntry{
					Label:   entity.Labels["en"].Value,
					Parents: entity.ItemIDs("P279"),
				}
			}

			if err := store.save(); err != nil {
				return nil, nil, err
			}
		}

		var next []string

		for _, id := range frontier {
			for _, parent := range store.Classes[id].Parents {
				if !visited[parent] {
					next = append(next, parent)
				}
			}
		}

		frontier = unique(next)
	}

	graph := make(ClassGraph, len(store.Classes))
	labels := make(map[string]string, len(store.Classes))

	for id, entry := range store.Classes {
		graph[id] = entry.Parents
		labels[id] = entry.Label
	}

	log.Printf("class graph: %d types", len(graph))

	return graph, labels, nil
}

// printSummary prints the numbers a human reviews before the generated file is merged.
func printSummary(types []TypeInfo) {
	var kept, dropped []TypeInfo

	denied := 0

	for _, t := range types {
		if t.Kept() {
			kept = append(kept, t)

			continue
		}

		if t.Kind == planner.Deny {
			denied++
		}

		dropped = append(dropped, t)
	}

	fmt.Printf("\n%d types seen: %d kept, %d dropped (%d of them denied outright)\n",
		len(types), len(kept), len(dropped), denied)

	printTypes(fmt.Sprintf("Top %d kept types", summarySize), kept)
	printTypes(fmt.Sprintf("Top %d dropped types", summarySize), dropped)
}

// printTypes prints the most common types of a list, most common first.
func printTypes(title string, types []TypeInfo) {
	byCount := make([]TypeInfo, len(types))
	copy(byCount, types)

	sort.SliceStable(byCount, func(i, j int) bool {
		if byCount[i].Count != byCount[j].Count {
			return byCount[i].Count > byCount[j].Count
		}

		return byCount[i].ID < byCount[j].ID
	})

	if len(byCount) > summarySize {
		byCount = byCount[:summarySize]
	}

	fmt.Printf("\n%s\n", title)

	for _, t := range byCount {
		kind := string(t.Kind)
		if kind == "" {
			kind = "unknown"
		}

		fmt.Printf("  %5d  %-9s %-10s %s\n", t.Count, kind, t.ID, comment(t.Label))
	}
}

// loadCache reads the cache file, or starts an empty one when it isn't there yet.
func loadCache(path string) (*cache, error) {
	store := &cache{
		Cities:  map[string]map[string][]string{},
		Classes: map[string]classEntry{},
		path:    path,
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return store, nil
	}

	if err != nil {
		return nil, fmt.Errorf("reading the cache %s: %w", path, err)
	}

	if err := json.Unmarshal(data, store); err != nil {
		log.Printf("cache %s is unreadable (%v), starting again", path, err)

		return &cache{
			Cities:  map[string]map[string][]string{},
			Classes: map[string]classEntry{},
			path:    path,
		}, nil
	}

	return store, nil
}

// save writes the cache, so an interrupted run doesn't have to fetch everything again.
func (c *cache) save() error {
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("encoding the cache: %w", err)
	}

	if err := os.WriteFile(c.path, data, 0o644); err != nil {
		return fmt.Errorf("writing the cache %s: %w", c.path, err)
	}

	return nil
}

// unique drops repeats, keeping the first of each.
func unique(ids []string) []string {
	seen := make(map[string]bool, len(ids))
	out := make([]string, 0, len(ids))

	for _, id := range ids {
		if seen[id] {
			continue
		}

		seen[id] = true

		out = append(out, id)
	}

	return out
}
