package writermap

import "sort"

// AssembleItineraries orders matched places within each indexed day section.
func AssembleItineraries(posts []IndexedPost, mentions map[string][]Mention) []Itinerary {
	var itineraries []Itinerary
	for _, post := range posts {
		if !post.Extracted.IsItinerary {
			continue
		}
		ref := postRef(post)
		itinerary := Itinerary{WriterHost: post.Host, PostRef: ref, PostURL: post.URL, PostTitle: post.Title, Language: post.Language}
		days := append([]DaySection(nil), post.Extracted.Days...)
		sort.Slice(days, func(i, j int) bool { return days[i].Start < days[j].Start })
		for _, day := range days {
			first := make(map[string]int)
			for qid, found := range mentions {
				for _, mention := range found {
					if mention.PostRef != ref {
						continue
					}
					for _, position := range mention.Positions {
						previous, found := first[qid]
						if position.Start >= day.Start && position.Start < day.End && (!found || position.Start < previous) {
							first[qid] = position.Start
						}
					}
				}
			}
			if len(first) == 0 {
				continue
			}
			qids := make([]string, 0, len(first))
			for qid := range first {
				qids = append(qids, qid)
			}
			sort.Slice(qids, func(i, j int) bool {
				if first[qids[i]] != first[qids[j]] {
					return first[qids[i]] < first[qids[j]]
				}
				return qids[i] < qids[j]
			})
			itinerary.Days = append(itinerary.Days, ItineraryDay{QIDs: qids})
		}
		if len(itinerary.Days) > 0 {
			itineraries = append(itineraries, itinerary)
		}
	}
	sort.Slice(itineraries, func(i, j int) bool { return itineraries[i].PostRef < itineraries[j].PostRef })
	return itineraries
}
