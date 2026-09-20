package planner

import (
	"math"
	"sort"
)

const (
	// kmeansRounds caps the clustering loop; it settles well before this on city-sized pools.
	kmeansRounds = 50
	// compactRadiusKM is how far apart two stops can sit and still feel like the same walk.
	compactRadiusKM = 1.5
	// PullReachKM is how far a short day may reach for a stop from another day. Beyond it
	// the day keeps the stops it has: two stops together beat three stops across town.
	PullReachKM = 3.0
)

// GroupDays splits the most famous ranked places into one walking day each. It takes
// the top SunnyPoolSize places, keeps the ones that are both famous and close together,
// clusters them by location, aims every day at MinStopsPerDay to MaxStopsPerDay stops —
// a day keeps 2 stops rather than reach past PullReachKM for a third — and puts each day
// in walking order.
func GroupDays(ranked []Place, days int) []Group {
	pool := ranked
	if len(pool) > SunnyPoolSize {
		pool = pool[:SunnyPoolSize]
	}
	if len(pool) == 0 {
		return nil
	}
	k := dayCount(len(pool), days)
	clusters, centers := clusterPlaces(selectCompact(pool, k*MaxStopsPerDay), k)
	clusters = balanceClusters(clusters, centers)
	fillSingletons(clusters, pool)
	groups := make([]Group, 0, len(clusters))
	for _, stops := range clusters {
		if len(stops) > 0 {
			groups = append(groups, WalkingLoop(stops))
		}
	}
	sort.SliceStable(groups, func(i, j int) bool {
		return moreFamous(groups[i].Stops[0], groups[j].Stops[0])
	})
	return groups
}

// fillSingletons gives an isolated day a nearby highlight that compact selection left out.
// It stays inside the sunny top-20 pool and the same reach limit as balancing. A truly
// isolated place remains alone rather than adding a long cross-city walk.
func fillSingletons(clusters [][]Place, pool []Place) {
	used := make(map[string]bool)
	for _, stops := range clusters {
		for _, stop := range stops {
			used[stop.ID] = true
		}
	}
	for i, stops := range clusters {
		if len(stops) != 1 {
			continue
		}
		best := -1
		distance := PullReachKM
		for j, candidate := range pool {
			if used[candidate.ID] {
				continue
			}
			d := DistanceKM(stops[0], candidate)
			if d <= distance && (best < 0 || d < distance || moreFamous(candidate, pool[best])) {
				best, distance = j, d
			}
		}
		if best >= 0 {
			clusters[i] = append(clusters[i], pool[best])
			used[pool[best].ID] = true
		}
	}
}

// dayCount is how many days a pool of this size can fill: never more than asked, never
// more than there are stops for, and always one while there is a place to visit.
func dayCount(size, days int) int {
	k := size / MinStopsPerDay
	if k > days {
		k = days
	}
	if k < 1 {
		k = 1
	}
	return k
}

// selectCompact keeps at most slots places, dropping the rest by fame and closeness
// together instead of fame alone. It always keeps the most famous place, then the
// highest-ranked places that sit near the ones already kept; once a day's worth of stops
// sit together it opens the next area around the most famous place left. Kept places stay
// in the order they came in, so the least famous stop of a day is still the last one.
func selectCompact(pool []Place, slots int) []Place {
	if len(pool) <= slots {
		return pool
	}
	kept := make([]bool, len(pool))
	for taken := 0; taken < slots; {
		seed := mostFamousLeft(pool, kept)
		if seed < 0 {
			break // every place is already kept
		}
		kept[seed] = true
		taken++
		area := []Place{pool[seed]}
		for len(area) < MaxStopsPerDay && taken < slots {
			next := mostFamousNear(pool, kept, area)
			if next < 0 {
				break
			}
			kept[next] = true
			taken++
			area = append(area, pool[next])
		}
	}
	compact := make([]Place, 0, slots)
	for i, p := range pool {
		if kept[i] {
			compact = append(compact, p)
		}
	}
	return compact
}

// mostFamousLeft is the index of the most famous place not kept yet, ties by ID, or -1
// when every place is kept.
func mostFamousLeft(pool []Place, kept []bool) int {
	best := -1
	for i, p := range pool {
		if !kept[i] && (best < 0 || moreFamous(p, pool[best])) {
			best = i
		}
	}
	return best
}

// mostFamousNear is the index of the most famous place left within a walk of the area,
// or -1 when every place is kept. It widens its reach only when nothing qualifies; the
// reach doubles, so the search always ends — a few rounds cover any city.
func mostFamousNear(pool []Place, kept []bool, area []Place) int {
	if mostFamousLeft(pool, kept) < 0 {
		return -1
	}
	for reach := compactRadiusKM; ; reach *= 2 {
		best := -1
		for i, p := range pool {
			if kept[i] || nearestKM(p, area) > reach {
				continue
			}
			if best < 0 || moreFamous(p, pool[best]) {
				best = i
			}
		}
		if best >= 0 {
			return best
		}
	}
}

// clusterPlaces runs a deterministic k-means and returns the stops of each cluster, in
// ranked order, alongside the centre they settled around.
func clusterPlaces(pool []Place, k int) ([][]Place, []Place) {
	centers := seedCenters(pool, k)
	clusters := assignToCenters(pool, centers)
	for round := 0; round < kmeansRounds; round++ {
		next := recenter(clusters, centers)
		if samePoints(next, centers) {
			break
		}
		centers = next
		clusters = assignToCenters(pool, centers)
	}
	return clusters, centers
}

// seedCenters picks the most famous place, then the place farthest from everything picked so
// far, farthest-point style. Ties go to the lower ID, so the seeds never vary.
func seedCenters(pool []Place, k int) []Place {
	first := pool[0]
	for _, p := range pool {
		if moreFamous(p, first) {
			first = p
		}
	}
	centers := []Place{first}
	for len(centers) < k {
		best, farthest := Place{}, -1.0
		for _, p := range pool {
			if hasPlaceID(centers, p.ID) {
				continue
			}
			if d := nearestKM(p, centers); d > farthest || d == farthest && p.ID < best.ID {
				best, farthest = p, d
			}
		}
		if farthest < 0 {
			break // every place is already a seed
		}
		centers = append(centers, best)
	}
	return centers
}

// assignToCenters puts every place with its closest centre, keeping the ranked order inside each
// cluster so the last stop is always the least famous one.
func assignToCenters(pool, centers []Place) [][]Place {
	clusters := make([][]Place, len(centers))
	for _, p := range pool {
		closest, distance := 0, math.Inf(1)
		for i, c := range centers {
			if d := DistanceKM(p, c); d < distance {
				closest, distance = i, d
			}
		}
		clusters[closest] = append(clusters[closest], p)
	}
	return clusters
}

// recenter moves each centre to the middle of its cluster. An empty cluster keeps the
// centre it had, so it still has somewhere to pull stops towards.
func recenter(clusters [][]Place, centers []Place) []Place {
	next := make([]Place, len(centers))
	for i, stops := range clusters {
		if len(stops) == 0 {
			next[i] = centers[i]
			continue
		}
		var lat, lon float64
		for _, p := range stops {
			lat, lon = lat+p.Lat, lon+p.Lon
		}
		next[i] = Place{Lat: lat / float64(len(stops)), Lon: lon / float64(len(stops))}
	}
	return next
}

// balanceClusters aims every day at MinStopsPerDay to MaxStopsPerDay stops. Short days
// pull the nearest stop from a day that can spare one, as long as it is within
// PullReachKM; a day with nothing that close keeps the stops it has. Days that are still
// too long drop their lowest-ranked stops.
func balanceClusters(clusters [][]Place, centers []Place) [][]Place {
	for moved := true; moved; {
		moved = false
		for i := range clusters {
			for len(clusters[i]) < MinStopsPerDay && pullNearest(clusters, centers, i) {
				moved = true
			}
		}
	}
	for i := range clusters {
		if len(clusters[i]) > MaxStopsPerDay {
			clusters[i] = clusters[i][:MaxStopsPerDay]
		}
	}
	return clusters
}

// pullNearest moves the stop closest to cluster i out of whichever other cluster has one
// to spare, as long as it sits within PullReachKM of cluster i's centre. It reports false
// when no cluster can give up a stop that close.
//
// A cluster left on its own may also take from a cluster sitting at exactly MinStopsPerDay,
// because two days of two stops read better than a day card showing a single stop. The
// donor still ends up with two, so the rescue can never strand anyone in turn.
func pullNearest(clusters [][]Place, centers []Place, i int) bool {
	keeps := MinStopsPerDay
	if len(clusters[i]) < 2 {
		keeps = MinStopsPerDay - 1
	}

	from, at, distance := -1, -1, math.Inf(1)
	for j := range clusters {
		if j == i || len(clusters[j]) <= keeps {
			continue
		}
		for s, p := range clusters[j] {
			if d := DistanceKM(p, centers[i]); d <= PullReachKM && d < distance {
				from, at, distance = j, s, d
			}
		}
	}
	if from < 0 {
		return false
	}
	clusters[i] = append(clusters[i], clusters[from][at])
	clusters[from] = append(clusters[from][:at], clusters[from][at+1:]...)
	return true
}

func hasPlaceID(places []Place, id string) bool {
	for _, p := range places {
		if p.ID == id {
			return true
		}
	}
	return false
}

// nearestKM is the distance from p to the closest of the given places.
func nearestKM(p Place, others []Place) float64 {
	best := math.Inf(1)
	for _, q := range others {
		if d := DistanceKM(p, q); d < best {
			best = d
		}
	}
	return best
}

// samePoints reports whether two sets of centres sit on the same coordinates.
func samePoints(a, b []Place) bool {
	for i := range a {
		if a[i].Lat != b[i].Lat || a[i].Lon != b[i].Lon {
			return false
		}
	}
	return true
}
