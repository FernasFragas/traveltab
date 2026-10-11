package writermap

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"weatherservice/internal/planner"
)

const maxMapPlaces = 200

// MatchRow is one indexed phrase occurrence joined to its source post.
type MatchRow struct {
	Post   IndexedPost
	Phrase NamePhrase
}

// MatchOverrides contains the reviewed name rules from writerdata/overrides.json.
type MatchOverrides struct {
	AliasRemovals []AliasRemoval `json:"alias_removals"`
	Merges        []PlaceMerge   `json:"merges"`
	Stoplist      []string       `json:"stoplist"`
}

type AliasRemoval struct {
	PlaceName string   `json:"place_name"`
	Aliases   []string `json:"aliases"`
}

type PlaceMerge struct {
	CanonicalName string   `json:"canonical_name"`
	Names         []string `json:"names"`
}

// MatchDestination turns nearby places and indexed rows into pins, mentions and routes.
// destinationNames are the destination phrases used to establish post or day context.
func MatchDestination(base BasePlaces, rows []MatchRow, destinationNames []string, sources []Source, overrides MatchOverrides) DestinationResult {
	places := matchingPlaces(base.Gazetteer, overrides)
	aliases := placeAliases(places, destinationNames, overrides)
	context := foldedSet(destinationNames)
	byPost := make(map[PostRef][]MatchRow)
	posts := make(map[PostRef]IndexedPost)
	for _, row := range rows {
		ref := postRef(row.Post)
		byPost[ref] = append(byPost[ref], row)
		posts[ref] = row.Post
	}
	blogNames := make(map[string]string, len(sources))
	for _, source := range sources {
		blogNames[source.Host] = source.Name
	}
	mentions := make(map[string][]Mention)
	writerHosts := make(map[string]map[string]bool)
	for ref, postRows := range byPost {
		post := posts[ref]
		for qid, positions := range matchPost(postRows, aliases, context) {
			mentions[qid] = append(mentions[qid], Mention{
				WriterHost: post.Host, BlogName: blogNames[post.Host], PostRef: ref,
				PostURL: post.URL, PostTitle: post.Title, Language: post.Language, Positions: positions,
			})
			if writerHosts[qid] == nil {
				writerHosts[qid] = make(map[string]bool)
			}
			writerHosts[qid][post.Host] = true
		}
	}
	counts := make(map[string]int, len(writerHosts))
	for qid, hosts := range writerHosts {
		counts[qid] = len(hosts)
	}
	for qid := range mentions {
		sort.Slice(mentions[qid], func(i, j int) bool {
			a, b := mentions[qid][i], mentions[qid][j]
			if a.WriterHost != b.WriterHost {
				return a.WriterHost < b.WriterHost
			}
			return a.PostRef < b.PostRef
		})
	}
	postList := make([]IndexedPost, 0, len(posts))
	for _, post := range posts {
		postList = append(postList, post)
	}
	return DestinationResult{
		Places: rankedPlaces(base.Pins, places, counts), Mentions: mentions,
		WriterCounts: counts, Itineraries: AssembleItineraries(postList, mentions),
	}
}

// MatchLookupPhrases returns the exact folded keys the index should fetch, including context.
func MatchLookupPhrases(gazetteer []Place, destinationNames []string, overrides MatchOverrides) []string {
	places := matchingPlaces(gazetteer, overrides)
	aliases := placeAliases(places, destinationNames, overrides)
	set := foldedSet(destinationNames)
	for name := range aliases {
		set[name] = true
	}
	phrases := make([]string, 0, len(set))
	for name := range set {
		phrases = append(phrases, name)
	}
	sort.Strings(phrases)
	return phrases
}

type placeName struct {
	QID   string
	Words int
}

type matchedSpan struct {
	QID string
	Position
}

func matchPost(rows []MatchRow, aliases map[string][]placeName, context map[string]bool) map[string][]Position {
	var contextRows []MatchRow
	for _, row := range rows {
		if context[row.Phrase.Folded] {
			contextRows = append(contextRows, row)
		}
	}
	if len(contextRows) == 0 {
		return nil
	}
	var spans []matchedSpan
	for _, row := range rows {
		phrase := row.Phrase
		for _, alias := range aliases[phrase.Folded] {
			if alias.Words == 1 && !phrase.OriginalCase {
				continue
			}
			if !inDestinationContext(phrase, row.Post.Extracted.Days, contextRows) {
				continue
			}
			spans = append(spans, matchedSpan{QID: alias.QID, Position: Position{Start: phrase.Start, End: phrase.End}})
		}
	}
	result := make(map[string][]Position)
	for _, span := range spans {
		contained := false
		for _, row := range rows {
			other := row.Phrase
			if other.Start <= span.Start && other.End >= span.End && other.End-other.Start > span.End-span.Start {
				contained = true
				break
			}
		}
		if !contained {
			result[span.QID] = append(result[span.QID], span.Position)
		}
	}
	for qid, positions := range result {
		sort.Slice(positions, func(i, j int) bool { return positions[i].Start < positions[j].Start })
		unique := positions[:0]
		for _, position := range positions {
			if len(unique) == 0 || unique[len(unique)-1] != position {
				unique = append(unique, position)
			}
		}
		result[qid] = unique
	}
	return result
}

func inDestinationContext(phrase NamePhrase, days []DaySection, contextRows []MatchRow) bool {
	if len(days) == 0 {
		return true
	}
	day := dayAt(phrase.Start, days)
	for _, row := range contextRows {
		if dayAt(row.Phrase.Start, days) == day || dayAt(row.Phrase.Start, days) == -1 {
			return true
		}
	}
	return false
}

func dayAt(position int, days []DaySection) int {
	for i, day := range days {
		if position >= day.Start && position < day.End {
			return i
		}
	}
	return -1
}

func matchingPlaces(gazetteer []Place, overrides MatchOverrides) []Place {
	places := append([]Place(nil), gazetteer...)
	removed := make(map[string]bool)
	for _, merge := range overrides.Merges {
		canonical := -1
		for i, place := range places {
			if hasName(place, merge.CanonicalName) {
				canonical = i
				break
			}
		}
		if canonical < 0 {
			continue
		}
		for _, name := range merge.Names {
			for i, place := range places {
				if i != canonical && hasName(place, name) {
					places[canonical].Names = append(places[canonical].Names, place.Names...)
					removed[place.QID] = true
				}
			}
			places[canonical].Names = append(places[canonical].Names, name)
		}
	}
	kept := places[:0]
	for _, place := range places {
		if removed[place.QID] || deniedPlace(place) {
			continue
		}
		kept = append(kept, place)
	}
	return kept
}

func placeAliases(places []Place, destinationNames []string, overrides MatchOverrides) map[string][]placeName {
	stop := foldedSet(overrides.Stoplist)
	for name := range foldedSet(destinationNames) {
		stop[name] = true
	}
	aliases := make(map[string][]placeName)
	for _, place := range places {
		for _, name := range place.Names {
			folded := foldName(name)
			if utf8.RuneCountInString(folded) < 5 || stop[folded] || removedAlias(place, folded, overrides.AliasRemovals) {
				continue
			}
			aliases[folded] = append(aliases[folded], placeName{QID: place.QID, Words: len(strings.Fields(folded))})
		}
	}
	return aliases
}

func removedAlias(place Place, folded string, removals []AliasRemoval) bool {
	for _, removal := range removals {
		if !hasName(place, removal.PlaceName) {
			continue
		}
		for _, alias := range removal.Aliases {
			if alias == "*" || foldName(alias) == folded {
				return true
			}
		}
	}
	return false
}

func hasName(place Place, name string) bool {
	for _, candidate := range place.Names {
		if foldName(candidate) == foldName(name) {
			return true
		}
	}
	return false
}

func foldedSet(names []string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, name := range names {
		if folded := foldName(name); folded != "" {
			set[folded] = true
		}
	}
	return set
}

func deniedPlace(place Place) bool {
	for _, typ := range place.Types {
		if planner.PlaceTypes[typ] == planner.Deny {
			return true
		}
	}
	return false
}

func rankedPlaces(pins, gazetteer []Place, counts map[string]int) []Place {
	byQID := make(map[string]Place)
	allowed := make(map[string]bool, len(gazetteer))
	for _, place := range gazetteer {
		allowed[place.QID] = true
	}
	for _, place := range pins {
		if allowed[place.QID] {
			byQID[place.QID] = place
		}
	}
	for _, place := range gazetteer {
		if counts[place.QID] > 0 {
			byQID[place.QID] = place
		}
	}
	places := make([]Place, 0, len(byQID))
	for _, place := range byQID {
		places = append(places, place)
	}
	sort.Slice(places, func(i, j int) bool {
		a, b := places[i], places[j]
		if counts[a.QID] != counts[b.QID] {
			return counts[a.QID] > counts[b.QID]
		}
		if a.Sitelinks != b.Sitelinks {
			return a.Sitelinks > b.Sitelinks
		}
		return a.QID < b.QID
	})
	if len(places) > maxMapPlaces {
		places = places[:maxMapPlaces]
	}
	return places
}

func postRef(post IndexedPost) PostRef {
	return PostRef(fmt.Sprintf("%s:%d", post.Host, post.PostID))
}
