package writermap

import (
	"sort"
	"strings"
	"time"
)

// MaxIndexedPosts is the most posts the writers index holds in total, across all blogs.
const MaxIndexedPosts = 100

// Select chooses at most limit posts across blogs. A candidate is a destination post: its title
// or URL slug names one of the known places (accent- and case-insensitive). Each blog's share is
// proportional to its destination posts (largest remainder, at least one for any blog that has
// one, never more than it has). Within a blog, posts with more recent comments come first, then
// the newest. The result lists each blog's picks in rank order.
func Select(blogs []BlogCandidates, places []string, limit int) []Pick {
	if limit <= 0 {
		return nil
	}
	known := newPlaceMatcher(places)
	ranked := make([][]SearchHit, len(blogs))
	for i, blog := range blogs {
		ranked[i] = rankDestinationPosts(blog, known)
	}
	quotas := shares(ranked, hostsOf(blogs), limit)
	var picks []Pick
	for i, blog := range blogs {
		for _, hit := range ranked[i][:quotas[i]] {
			picks = append(picks, Pick{Host: blog.Host, PostID: hit.ID})
		}
	}
	return picks
}

// CountDestinationPosts is how many distinct hits of the blog name a known place in their title
// or URL slug: the candidates Select would rank.
func CountDestinationPosts(blog BlogCandidates, places []string) int {
	return len(rankDestinationPosts(blog, newPlaceMatcher(places)))
}

// Trim returns the n lowest-ranked of the given picks, lowest first. Rank is the one Select uses
// inside a blog (more recent comments, then newest), applied across blogs. Picks of a blog that
// is not in blogs rank lowest of all, and fewer than n come back when there are fewer picks.
func Trim(blogs []BlogCandidates, places []string, picks []Pick, n int) []Pick {
	if n <= 0 {
		return nil
	}
	known := newPlaceMatcher(places)
	type entry struct {
		pick     Pick
		hit      SearchHit
		comments int
	}
	hits := make(map[Pick]entry)
	for _, blog := range blogs {
		for _, hit := range rankDestinationPosts(blog, known) {
			pick := Pick{Host: blog.Host, PostID: hit.ID}
			hits[pick] = entry{pick: pick, hit: hit, comments: blog.Comments[hit.ID]}
		}
	}
	entries := make([]entry, len(picks))
	for i, pick := range picks {
		entries[i] = hits[pick]
		entries[i].pick = pick
	}
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.comments != b.comments {
			return a.comments < b.comments
		}
		if !a.hit.Published.Equal(b.hit.Published) {
			return a.hit.Published.Before(b.hit.Published)
		}
		if a.pick.PostID != b.pick.PostID {
			return a.pick.PostID < b.pick.PostID
		}
		return a.pick.Host < b.pick.Host
	})
	dropped := make([]Pick, 0, min(n, len(entries)))
	for _, e := range entries[:min(n, len(entries))] {
		dropped = append(dropped, e.pick)
	}
	return dropped
}

// BlogCandidates is one blog's search hits and recent-comment counts, by post ID. A blog with
// no readable comments has a nil Comments map.
type BlogCandidates struct {
	Host     string
	Hits     []SearchHit
	Comments map[int64]int
}

// SearchHit is one search result: metadata only, never post text.
type SearchHit struct {
	ID        int64
	Link      string
	Slug      string
	Title     string
	Published time.Time
}

// Pick is one chosen post.
type Pick struct {
	Host   string
	PostID int64
}

// rankDestinationPosts keeps the hits that name a place, once each, best first.
func rankDestinationPosts(blog BlogCandidates, known placeMatcher) []SearchHit {
	seen := make(map[int64]bool, len(blog.Hits))
	var posts []SearchHit
	for _, hit := range blog.Hits {
		if seen[hit.ID] || !known.names(hit.Title, hit.Slug) {
			continue
		}
		seen[hit.ID] = true
		posts = append(posts, hit)
	}
	sort.SliceStable(posts, func(i, j int) bool {
		a, b := posts[i], posts[j]
		if ca, cb := blog.Comments[a.ID], blog.Comments[b.ID]; ca != cb {
			return ca > cb
		}
		if !a.Published.Equal(b.Published) {
			return a.Published.After(b.Published)
		}
		return a.ID > b.ID
	})
	return posts
}

func hostsOf(blogs []BlogCandidates) []string {
	hosts := make([]string, len(blogs))
	for i, blog := range blogs {
		hosts[i] = blog.Host
	}
	return hosts
}

// shares splits limit across the blogs by their candidate counts. When everything fits, each
// blog keeps all of its candidates. Otherwise each gets its proportional floor (at least 1),
// trimmed back if those minimums overshoot, then the leftover goes by largest remainder.
func shares(ranked [][]SearchHit, hosts []string, limit int) []int {
	counts := make([]int, len(ranked))
	total := 0
	for i, posts := range ranked {
		counts[i] = len(posts)
		total += len(posts)
	}
	if total <= limit {
		return counts
	}
	quotas := make([]int, len(ranked))
	ideal := make([]float64, len(ranked))
	used := 0
	for i, n := range counts {
		if n == 0 {
			continue
		}
		ideal[i] = float64(limit) * float64(n) / float64(total)
		quotas[i] = max(1, int(ideal[i]))
		used += quotas[i]
	}
	byRemainder := func(order []int, remainder func(i int) float64) {
		sort.SliceStable(order, func(a, b int) bool {
			i, j := order[a], order[b]
			if ra, rb := remainder(i), remainder(j); ra != rb {
				return ra > rb
			}
			if counts[i] != counts[j] {
				return counts[i] > counts[j]
			}
			return hosts[i] < hosts[j]
		})
	}
	// Trim the most over-served blogs while the minimums overshoot the limit.
	for used > limit {
		var order []int
		for i, q := range quotas {
			if q > 0 {
				order = append(order, i)
			}
		}
		byRemainder(order, func(i int) float64 { return float64(quotas[i]) - ideal[i] })
		for _, i := range order {
			if used > limit && (quotas[i] > 1 || used-limit >= 1 && len(order) > limit) {
				quotas[i]--
				used--
			}
		}
	}
	// Hand out the leftover by largest remainder, never above a blog's candidates.
	for used < limit {
		var order []int
		for i, q := range quotas {
			if q < counts[i] {
				order = append(order, i)
			}
		}
		byRemainder(order, func(i int) float64 { return ideal[i] - float64(quotas[i]) })
		for _, i := range order {
			if used < limit {
				quotas[i]++
				used++
			}
		}
	}
	return quotas
}

// placeMatcher recognises known place names inside titles and slugs, as whole words.
type placeMatcher struct {
	folded   map[string]bool
	maxWords int
}

func newPlaceMatcher(places []string) placeMatcher {
	matcher := placeMatcher{folded: make(map[string]bool, len(places))}
	for _, place := range places {
		words := strings.Fields(foldName(place))
		if len(words) == 0 {
			continue
		}
		matcher.folded[strings.Join(words, " ")] = true
		matcher.maxWords = max(matcher.maxWords, len(words))
	}
	return matcher
}

// names reports whether the title or the slug contains a known place name.
func (m placeMatcher) names(title, slug string) bool {
	return m.contains(title) || m.contains(strings.ReplaceAll(slug, "-", " "))
}

func (m placeMatcher) contains(text string) bool {
	words := strings.FieldsFunc(foldName(text), func(r rune) bool { return !isWordRune(r) })
	for start := range words {
		for n := 1; n <= m.maxWords && start+n <= len(words); n++ {
			if m.folded[strings.Join(words[start:start+n], " ")] {
				return true
			}
		}
	}
	return false
}
