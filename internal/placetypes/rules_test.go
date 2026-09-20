package placetypes

import (
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"weatherservice/internal/planner"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeGraph is a small made-up class tree. The roots and the deny types are the real Wikidata
// IDs, so the tests exercise the real rule lists; everything below them is invented, and no
// test touches the network.
func fakeGraph() ClassGraph {
	graph := ClassGraph{
		// A museum of the town: one hop to the museum root.
		"Q900001": {"Q33506"},
		// A castle that is also a museum: it reaches an indoor and an outdoor root.
		"Q900002": {"Q33506", "Q23413"},
		// A subclass of a denied type, and nothing else.
		"Q900003": {"Q137773"},
		// A subclass of a denied type that is also a park.
		"Q900004": {"Q137773", "Q22698"},
		// A bridge is a built structure and nothing a visitor stops at.
		"Q12280":  {"Q811979"},
		"Q811979": {},
		// A cycle that still reaches a root.
		"Q900010": {"Q900011"},
		"Q900011": {"Q900012"},
		"Q900012": {"Q900010", "Q22698"},
		// A cycle that reaches nothing at all.
		"Q900020": {"Q900021"},
		"Q900021": {"Q900020"},
		// The two market branches, with their real Wikidata parents. "market" (Q132510) is the
		// gathering of traders; "marketplace" (Q330284) is the space it runs in, and it sits
		// under retail environment, not under market. A market hall therefore reaches a root
		// only through marketplace, while a food market reaches one through market.
		"Q132510":   {"Q37654", "Q15275719"},
		"Q28142754": {"Q132510"},
		"Q330284":   {"Q39659371", "Q13226383", "Q2221906"},
		"Q2080521":  {"Q240854", "Q330284", "Q18760388"},
		"Q13033698": {"Q174782", "Q330284"},
		// The roots themselves sit under general types, never under each other.
		"Q33506":  {"Q811979"},
		"Q23413":  {"Q811979"},
		"Q22698":  {"Q811979"},
		"Q174782": {"Q811979"},
		"Q137773": {"Q811979"},
	}

	// A chain whose root sits exactly MaxDepth hops up, and one whose root sits a hop further.
	addChain(graph, "Q9100", MaxDepth, "Q22698")
	addChain(graph, "Q9200", MaxDepth+1, "Q22698")

	return graph
}

// addChain adds hops links from <prefix>01 up to root, so root is exactly hops hops away.
func addChain(graph ClassGraph, prefix string, hops int, root string) {
	for i := 1; i < hops; i++ {
		graph[fmt.Sprintf("%s%02d", prefix, i)] = []string{fmt.Sprintf("%s%02d", prefix, i+1)}
	}

	graph[fmt.Sprintf("%s%02d", prefix, hops)] = []string{root}
}

func TestRules_RootTypeGetsItsKind(t *testing.T) {
	rules := DefaultRules()

	kind, ok := rules.KindOf("Q33506", fakeGraph())
	assert.True(t, ok, "museum is an indoor root")
	assert.Equal(t, planner.Indoor, kind)

	kind, ok = rules.KindOf("Q5393308", fakeGraph())
	assert.True(t, ok, "Buddhist temple is an outdoor root even with no entry in the graph")
	assert.Equal(t, planner.Outdoor, kind)
}

func TestRules_SubclassOfARootGetsItsKind(t *testing.T) {
	rules := DefaultRules()

	kind, ok := rules.KindOf("Q900001", fakeGraph())

	assert.True(t, ok, "a subclass of museum is kept")
	assert.Equal(t, planner.Indoor, kind)
}

func TestRules_TypeReachingIndoorAndOutdoorIsMixed(t *testing.T) {
	rules := DefaultRules()

	kind, ok := rules.KindOf("Q900002", fakeGraph())

	assert.True(t, ok, "a castle museum is kept")
	assert.Equal(t, planner.Mixed, kind, "it reaches both an indoor and an outdoor root")
}

func TestRules_DenyMatchesOnlyTheExactType(t *testing.T) {
	rules := DefaultRules()
	graph := fakeGraph()

	kind, ok := rules.KindOf("Q137773", graph)
	assert.True(t, ok, "ward of Japan is a known type")
	assert.Equal(t, planner.Deny, kind)

	kind, ok = rules.KindOf("Q900004", graph)
	assert.True(t, ok, "a subclass of ward of Japan that is also a park is kept")
	assert.Equal(t, planner.Outdoor, kind, "deny never travels down to subclasses")

	kind, ok = rules.KindOf("Q900003", graph)
	assert.False(t, ok, "a subclass of ward of Japan reaches no root, so it is simply unknown")
	assert.NotEqual(t, planner.Deny, kind, "it must not inherit the deny")
}

func TestRules_BridgeAloneIsNotKept(t *testing.T) {
	rules := DefaultRules()
	graph := fakeGraph()

	_, ok := rules.KindOf("Q12280", graph)
	assert.False(t, ok, "a bridge on its own is not a stop")

	kind, ok := rules.KindOf("Q4989906", graph)
	assert.True(t, ok, "but a monument is, so a bridge that is also a monument still counts")
	assert.Equal(t, planner.Outdoor, kind)
}

func TestRules_SurvivesACycleInTheClassGraph(t *testing.T) {
	rules := DefaultRules()
	graph := fakeGraph()

	kind, ok := rules.KindOf("Q900010", graph)
	assert.True(t, ok, "the walk gets past the cycle and finds the park")
	assert.Equal(t, planner.Outdoor, kind)

	_, ok = rules.KindOf("Q900020", graph)
	assert.False(t, ok, "a cycle that reaches no root ends the walk instead of hanging")
}

func TestRules_StopsAtDepthTen(t *testing.T) {
	rules := DefaultRules()
	graph := fakeGraph()

	kind, ok := rules.KindOf("Q910001", graph)
	assert.True(t, ok, "a root exactly MaxDepth hops up is still found")
	assert.Equal(t, planner.Outdoor, kind)

	_, ok = rules.KindOf("Q920001", graph)
	assert.False(t, ok, "a root one hop further is out of reach")
}

func TestRules_MarketplaceIsAnIndoorRootOfItsOwn(t *testing.T) {
	rules := DefaultRules()
	graph := fakeGraph()

	kind, ok := rules.KindOf("Q330284", graph)
	assert.True(t, ok, "marketplace is a root, because it hangs off retail environment, not off market")
	assert.Equal(t, planner.Indoor, kind)

	kind, ok = rules.KindOf("Q2080521", graph)
	assert.True(t, ok, "a market hall reaches a root only through marketplace")
	assert.Equal(t, planner.Indoor, kind, "a covered market is a rainy-day stop")

	kind, ok = rules.KindOf("Q13033698", graph)
	assert.True(t, ok, "a market square is kept either way")
	assert.Equal(t, planner.Mixed, kind, "it is both a square and a marketplace")

	kind, ok = rules.KindOf("Q28142754", graph)
	assert.True(t, ok, "food market is kept")
	assert.Equal(t, planner.Indoor, kind,
		"it comes in under the market root already, so it needs no root of its own")
}

func TestRules_ClassifyAlwaysListsEveryRootAndDenyType(t *testing.T) {
	rules := DefaultRules()

	types := rules.Classify(map[string]int{"Q900001": 4}, map[string]string{"Q900001": "town museum"}, fakeGraph())

	byID := make(map[string]TypeInfo, len(types))
	for _, info := range types {
		byID[info.ID] = info
	}

	assert.Equal(t, TypeInfo{ID: "Q900001", Label: "town museum", Count: 4, Kind: planner.Indoor}, byID["Q900001"],
		"a counted type keeps its label and count")
	assert.Equal(t, planner.Outdoor, byID["Q5393308"].Kind, "Buddhist temple is listed even with no count")
	assert.Equal(t, planner.Deny, byID["Q137773"].Kind, "ward of Japan is listed even with no count")
	assert.Equal(t, 0, byID["Q137773"].Count)
}

func TestRender_WritesAGofmtedMapSortedByID(t *testing.T) {
	types := []TypeInfo{
		{ID: "Q1007870", Label: "art gallery", Count: 12, Kind: planner.Indoor},
		{ID: "Q33506", Label: "museum", Count: 340, Kind: planner.Indoor},
		{ID: "Q137773", Label: "ward of Japan", Count: 11, Kind: planner.Deny},
		{ID: "Q22698", Label: "park", Count: 88, Kind: planner.Outdoor},
	}

	src, err := Render(types)
	require.NoError(t, err)

	out := string(src)

	formatted, err := format.Source(src)
	require.NoError(t, err, "the output must be valid Go")
	assert.Equal(t, string(formatted), out, "the output must already be gofmt-ed")

	_, err = parser.ParseFile(token.NewFileSet(), "placetypes_gen.go", src, parser.AllErrors)
	require.NoError(t, err)

	assert.Contains(t, out, "// Code generated by cmd/placetypes; DO NOT EDIT.")
	assert.Contains(t, out, "package planner")
	assert.Contains(t, out, "var PlaceTypes = map[string]Kind{")

	assert.Equal(t, []string{"Q22698", "Q33506", "Q137773", "Q1007870"}, mapOrder(t, out),
		"entries are sorted by ID")

	assert.Contains(t, lineWith(t, out, `"Q33506":`), "Indoor,")
	assert.Contains(t, lineWith(t, out, `"Q33506":`), "// museum (340)")
	assert.Contains(t, lineWith(t, out, `"Q22698":`), "Outdoor,")
	assert.Contains(t, lineWith(t, out, `"Q137773":`), "Deny,")
	assert.Contains(t, lineWith(t, out, `"Q137773":`), "// ward of Japan (11)")
}

// mapOrder returns the type IDs in the order the generated map lists them.
func mapOrder(t *testing.T, src string) []string {
	t.Helper()

	var ids []string

	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, `"Q`) {
			continue
		}

		ids = append(ids, strings.Trim(strings.SplitN(line, ":", 2)[0], `"`))
	}

	return ids
}

// lineWith returns the one line of src holding want.
func lineWith(t *testing.T, src, want string) string {
	t.Helper()

	for _, line := range strings.Split(src, "\n") {
		if strings.Contains(line, want) {
			return line
		}
	}

	require.Failf(t, "line not found", "no line holds %q", want)

	return ""
}
