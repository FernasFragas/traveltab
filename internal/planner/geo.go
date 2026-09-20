package planner

import (
	"math"
	"net/url"
	"strconv"
	"strings"
)

// DistanceKM returns the great-circle distance between two coordinates.
func DistanceKM(a, b Place) float64 {
	const radians = math.Pi / 180
	dlat, dlon := (b.Lat-a.Lat)*radians, (b.Lon-a.Lon)*radians
	h := math.Pow(math.Sin(dlat/2), 2) + math.Cos(a.Lat*radians)*math.Cos(b.Lat*radians)*math.Pow(math.Sin(dlon/2), 2)
	return 6371 * 2 * math.Asin(math.Sqrt(math.Min(1, h)))
}

func moreFamous(a, b Place) bool {
	return a.Sitelinks > b.Sitelinks || a.Sitelinks == b.Sitelinks && a.ID < b.ID
}

// WalkingLoop visits the nearest unvisited stop, then returns to its starting point.
func WalkingLoop(stops []Place) Group {
	if len(stops) == 0 {
		return Group{}
	}
	remaining := append([]Place(nil), stops...)
	first := 0
	for i := range remaining {
		if moreFamous(remaining[i], remaining[first]) {
			first = i
		}
	}
	group := Group{Stops: []Place{remaining[first]}}
	remaining = append(remaining[:first], remaining[first+1:]...)
	for len(remaining) > 0 {
		last := group.Stops[len(group.Stops)-1]
		next := 0
		for i := 1; i < len(remaining); i++ {
			d, best := DistanceKM(last, remaining[i]), DistanceKM(last, remaining[next])
			if d < best || d == best && remaining[i].ID < remaining[next].ID {
				next = i
			}
		}
		group.WalkKM += DistanceKM(last, remaining[next])
		group.Stops = append(group.Stops, remaining[next])
		remaining = append(remaining[:next], remaining[next+1:]...)
	}
	group.WalkKM += DistanceKM(group.Stops[len(group.Stops)-1], group.Stops[0])
	return group
}

// Medoid returns the input place with the smallest total distance to its peers.
func Medoid(places []Place) Place {
	best := Place{}
	distance := math.Inf(1)
	for _, p := range places {
		total := 0.0
		for _, q := range places {
			total += DistanceKM(p, q)
		}
		if total < distance || total == distance && p.ID < best.ID {
			best = p
			distance = total
		}
	}
	return best
}

func (p Place) ImageURL(width int) string {
	if p.Image == "" {
		return ""
	}
	return "https://commons.wikimedia.org/wiki/Special:FilePath/" + url.PathEscape(strings.ReplaceAll(p.Image, " ", "_")) + "?width=" + strconv.Itoa(width)
}
func (p Place) ImagePageURL() string {
	if p.Image == "" {
		return ""
	}
	return "https://commons.wikimedia.org/wiki/File:" + url.PathEscape(strings.ReplaceAll(p.Image, " ", "_"))
}
