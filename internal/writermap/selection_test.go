package writermap_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"weatherservice/internal/writermap"
)

var selectionPlaces = []string{"Madeira", "Açores", "Lisboa", "Pico Ruivo"}

func TestSelect_KeepsOnlyPostsWhoseTitleOrSlugNamesAPlace(t *testing.T) {
	blog := writermap.BlogCandidates{Host: "a", Hits: []writermap.SearchHit{
		{ID: 1, Title: "Two weeks in MADEIRA", Slug: "two-weeks"},
		{ID: 2, Title: "Best beaches", Slug: "praias-dos-acores"},
		{ID: 3, Title: "Pack light", Slug: "pack-light"}, // the body mentions Lisboa, the title and slug do not
		{ID: 4, Title: "Visita aos Açores", Slug: "visita"},
		{ID: 5, Title: "Hiking Pico Ruivo", Slug: "hiking"},
	}}

	picks := writermap.Select([]writermap.BlogCandidates{blog}, selectionPlaces, 100)

	assert.ElementsMatch(t, []int64{1, 2, 4, 5}, ids(picks, "a"))
}

func TestSelect_DropsDuplicateHitsFromSeveralSearchTerms(t *testing.T) {
	hit := writermap.SearchHit{ID: 1, Title: "Madeira", Slug: "madeira"}
	blog := writermap.BlogCandidates{Host: "a", Hits: []writermap.SearchHit{hit, hit}}

	picks := writermap.Select([]writermap.BlogCandidates{blog}, selectionPlaces, 100)

	assert.Equal(t, []int64{1}, ids(picks, "a"))
}

func TestSelect_SplitsProportionallyWithLargestRemainder(t *testing.T) {
	blogs := []writermap.BlogCandidates{blogOf("a", 100), blogOf("b", 33), blogOf("c", 33)}

	picks := writermap.Select(blogs, selectionPlaces, 100)

	assert.Len(t, picks, 100)
	// ideal shares 60.24 / 19.88 / 19.88: floors 60, 19, 19, then the two leftovers go to b and c
	assert.Len(t, ids(picks, "a"), 60)
	assert.Len(t, ids(picks, "b"), 20)
	assert.Len(t, ids(picks, "c"), 20)
}

func TestSelect_GivesEveryBlogWithADestinationPostAtLeastOne(t *testing.T) {
	blogs := []writermap.BlogCandidates{blogOf("big", 500), blogOf("tiny", 1), blogOf("none", 0)}

	picks := writermap.Select(blogs, selectionPlaces, 100)

	assert.Len(t, picks, 100)
	assert.Len(t, ids(picks, "tiny"), 1)
	assert.Len(t, ids(picks, "big"), 99)
	assert.Empty(t, ids(picks, "none"))
}

func TestSelect_NeverGivesABlogMoreThanItsCandidates(t *testing.T) {
	blogs := []writermap.BlogCandidates{blogOf("a", 1), blogOf("b", 100), blogOf("c", 100)}

	picks := writermap.Select(blogs, selectionPlaces, 100)

	assert.Len(t, picks, 100)
	assert.Len(t, ids(picks, "a"), 1)
}

func TestSelect_RanksByCommentsThenNewest(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	blog := writermap.BlogCandidates{
		Host: "a",
		Hits: []writermap.SearchHit{
			{ID: 1, Title: "Madeira", Published: base.Add(1 * time.Hour)},
			{ID: 2, Title: "Madeira", Published: base.Add(2 * time.Hour)},
			{ID: 3, Title: "Madeira", Published: base.Add(3 * time.Hour)},
			{ID: 4, Title: "Madeira", Published: base.Add(4 * time.Hour)},
		},
		Comments: map[int64]int{1: 9, 2: 5, 3: 5},
	}

	picks := writermap.Select([]writermap.BlogCandidates{blog}, selectionPlaces, 3)

	assert.Equal(t, []int64{1, 3, 2}, ids(picks, "a"), "comments first, newest breaks the tie, no-comment post 4 is cut")
}

func TestSelect_BlogWithoutCommentsTakesItsNewest(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	blog := writermap.BlogCandidates{Host: "a"}
	for id := int64(1); id <= 5; id++ {
		blog.Hits = append(blog.Hits, writermap.SearchHit{ID: id, Title: "Lisboa", Published: base.Add(time.Duration(id) * time.Hour)})
	}

	picks := writermap.Select([]writermap.BlogCandidates{blog}, selectionPlaces, 2)

	assert.Equal(t, []int64{5, 4}, ids(picks, "a"))
}

func TestSelect_FewerCandidatesThanTheCapKeepsThemAll(t *testing.T) {
	blogs := []writermap.BlogCandidates{blogOf("a", 30), blogOf("b", 20)}

	picks := writermap.Select(blogs, selectionPlaces, 100)

	assert.Len(t, picks, 50)
}

func TestSelect_NoPlacesOrNoBlogsPicksNothing(t *testing.T) {
	assert.Empty(t, writermap.Select(nil, selectionPlaces, 100))
	assert.Empty(t, writermap.Select([]writermap.BlogCandidates{blogOf("a", 3)}, nil, 100))
}

func TestSelect_NoRoomLeftPicksNothing(t *testing.T) {
	blogs := []writermap.BlogCandidates{blogOf("a", 3)}

	assert.Empty(t, writermap.Select(blogs, selectionPlaces, 0))
	assert.Empty(t, writermap.Select(blogs, selectionPlaces, -5))
}

func TestSelect_MoreBlogsThanTheCapKeepsTheCapAndTheBiggestBlogs(t *testing.T) {
	var blogs []writermap.BlogCandidates
	for i := 0; i < 5; i++ {
		blogs = append(blogs, blogOf(fmt.Sprintf("b%d", i), i+1))
	}

	picks := writermap.Select(blogs, selectionPlaces, 3)

	assert.Len(t, picks, 3)
	assert.Empty(t, ids(picks, "b0"))
	assert.Len(t, ids(picks, "b4"), 1)
}

// blogOf returns a blog with n destination hits.
func blogOf(host string, n int) writermap.BlogCandidates {
	blog := writermap.BlogCandidates{Host: host}
	for id := int64(1); id <= int64(n); id++ {
		blog.Hits = append(blog.Hits, writermap.SearchHit{ID: id, Title: "Guide to Madeira", Slug: "guide"})
	}
	return blog
}

func ids(picks []writermap.Pick, host string) []int64 {
	var out []int64
	for _, pick := range picks {
		if pick.Host == host {
			out = append(out, pick.PostID)
		}
	}
	return out
}
