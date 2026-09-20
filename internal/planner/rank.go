package planner

import "sort"

// boostRank is the place a boosted entry is ranked as, counting from the most famous.
const boostRank = 10

// rankEntry pairs a place with the fame value it is sorted by. Boosted places borrow
// their fame from the rest of the list, so Place.Sitelinks is never changed.
type rankEntry struct {
	place   Place
	fame    int
	boosted bool
}

// Rank keeps the candidates worth visiting, sets their Kind and sorts them by fame.
// A place listed in boosts skips the type filter and takes the boosted Kind. Types
// maps a Wikidata "instance of" ID to the Kind it implies; a Deny type drops the place.
func Rank(candidates []Place, types, boosts map[string]Kind) []Place {
	var plain, boosted []rankEntry
	for _, p := range candidates {
		if kind, ok := boosts[p.ID]; ok {
			p.Kind = kind
			boosted = append(boosted, rankEntry{place: p, boosted: true})
			continue
		}
		kind, ok := kindOf(p.Types, types)
		if !ok {
			continue
		}
		p.Kind = kind
		plain = append(plain, rankEntry{place: p, fame: p.Sitelinks})
	}
	sortRanked(plain)
	all := make([]rankEntry, 0, len(plain)+len(boosted))
	all = append(all, plain...)
	for _, b := range boosted {
		b.fame = boostFame(plain, b.place.Sitelinks)
		all = append(all, b)
	}
	sortRanked(all)
	if len(all) > CandidatePoolSize {
		all = all[:CandidatePoolSize]
	}
	out := make([]Place, len(all))
	for i := range all {
		out[i] = all[i].place
	}
	return out
}

// kindOf reads a place's types and says how it behaves in the rain. It reports false
// when a Deny type is present, or when no type is known.
func kindOf(placeTypes []string, types map[string]Kind) (Kind, bool) {
	var indoor, outdoor bool
	for _, t := range placeTypes {
		switch types[t] {
		case Deny:
			return "", false
		case Indoor:
			indoor = true
		case Outdoor:
			outdoor = true
		case Mixed:
			indoor, outdoor = true, true
		}
	}
	switch {
	case indoor && outdoor:
		return Mixed, true
	case indoor:
		return Indoor, true
	case outdoor:
		return Outdoor, true
	}
	return "", false
}

// boostFame gives a boosted place the fame of the boostRank-th place, or of the last
// one when the list is shorter. With nothing to compare against it keeps its own.
func boostFame(plain []rankEntry, own int) int {
	if len(plain) == 0 {
		return own
	}
	i := boostRank - 1
	if i >= len(plain) {
		i = len(plain) - 1
	}
	return plain[i].fame
}

// sortRanked orders by fame, then puts boosted places first, then by ID, so that the
// result never depends on the input order.
func sortRanked(s []rankEntry) {
	sort.SliceStable(s, func(i, j int) bool {
		switch a, b := s[i], s[j]; {
		case a.fame != b.fame:
			return a.fame > b.fame
		case a.boosted != b.boosted:
			return a.boosted
		default:
			return a.place.ID < b.place.ID
		}
	})
}
