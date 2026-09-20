package placetypes

import (
	"fmt"
	"go/format"
	"sort"
	"strconv"
	"strings"

	"weatherservice/internal/planner"
)

// MaxDepth is how many subclass-of (P279) hops the walk follows before giving up. Wikidata's
// class tree is deep and full of cycles, so the walk is bounded in both ways.
const MaxDepth = 10

// ClassGraph maps a Wikidata type ID to the types it is a subclass of (P279).
type ClassGraph map[string][]string

// TypeInfo is everything the tool knows about one Wikidata type: how often places in the sample
// cities had it, its English label, and the Kind the rules give it ("" when it isn't kept).
type TypeInfo struct {
	ID    string
	Label string
	Count int
	Kind  planner.Kind
}

// Rules decide what a Wikidata type means for a trip plan.
type Rules struct {
	// Roots maps a root type to its Kind. A type that is a root, or reaches one by going up
	// subclass-of, takes that Kind.
	Roots map[string]planner.Kind
	// Deny holds types that drop a place. They match the exact type only, never through
	// subclasses: Wikidata's tree is messy, and a broad exclusion threw out Belém Tower and
	// nature parks during validation.
	Deny map[string]bool
}

// indoorRoots are the types you visit under a roof.
var indoorRoots = []string{
	"Q33506",   // museum
	"Q1007870", // art gallery
	"Q2281788", // public aquarium
	"Q24354",   // theatre building
	"Q16970",   // church building
	"Q2977",    // cathedral
	"Q163687",  // basilica
	"Q32815",   // mosque
	"Q34627",   // synagogue
	"Q16560",   // palace
	"Q44613",   // monastery
	"Q1060829", // concert hall
	"Q153562",  // opera house
	"Q200764",  // bookstore
	"Q132510",  // market
	// Q330284 marketplace is the *space* a market runs in, and Wikidata hangs it off retail
	// environment, not off Q132510 market, so the two branches never meet. Without it, covered
	// market halls typed only as "market hall" (Q2080521, a subclass of marketplace) are missed
	// entirely — La Boqueria and the Mercato Centrale among them. Its only subclasses in the
	// sample are market hall and market square; the retail junk (shops, malls, restaurants)
	// hangs off marketplace's *parent*, which the upward walk never reaches from below.
	"Q330284", // marketplace
}

// outdoorRoots are the types you visit in the open air.
var outdoorRoots = []string{
	"Q22698",   // park
	"Q1107656", // garden
	"Q167346",  // botanical garden
	"Q174782",  // square
	"Q6017969", // scenic viewpoint
	"Q40080",   // beach
	"Q12518",   // tower
	"Q4989906", // monument
	"Q179700",  // statue
	"Q23413",   // castle
	"Q57821",   // fortification
	"Q839954",  // archaeological site
	"Q43501",   // zoo
	"Q39715",   // lighthouse
	"Q79007",   // street
	"Q123705",  // neighborhood
	"Q39614",   // cemetery
	"Q845945",  // Shinto shrine
	"Q5393308", // Buddhist temple
	"Q8502",    // mountain
	"Q23442",   // island
}

// denyTypes drop a place outright, by exact type only.
var denyTypes = []string{
	"Q494721",   // city of Japan
	"Q1549591",  // big city
	"Q137773",   // ward of Japan
	"Q1025961",  // capital of Japan
	"Q27554677", // former capital
	"Q1220959",  // building of public administration
	"Q515",      // city
}

// DefaultRules are the rules from the validation run: the indoor and outdoor roots, and the
// exact types that are never kept. Bridge (Q12280) and library (Q7075) are deliberately not
// roots: road bridges and libraries rank high but aren't stops.
func DefaultRules() Rules {
	rules := Rules{
		Roots: make(map[string]planner.Kind, len(indoorRoots)+len(outdoorRoots)),
		Deny:  make(map[string]bool, len(denyTypes)),
	}

	for _, id := range indoorRoots {
		rules.Roots[id] = planner.Indoor
	}

	for _, id := range outdoorRoots {
		rules.Roots[id] = planner.Outdoor
	}

	for _, id := range denyTypes {
		rules.Deny[id] = true
	}

	return rules
}

// KindOf says how a type behaves in the rain. A denied type answers Deny. Otherwise the walk
// goes up subclass-of, at most MaxDepth hops and never twice through the same type, and the
// roots it reaches decide the Kind. It reports false when the type reaches no root, which means
// a place with only this type isn't worth a stop.
func (r Rules) KindOf(typeID string, graph ClassGraph) (planner.Kind, bool) {
	if r.Deny[typeID] {
		return planner.Deny, true
	}

	var indoor, outdoor bool

	visited := map[string]bool{typeID: true}
	frontier := []string{typeID}

	for depth := 0; len(frontier) > 0; depth++ {
		for _, id := range frontier {
			switch r.Roots[id] {
			case planner.Indoor:
				indoor = true
			case planner.Outdoor:
				outdoor = true
			}
		}

		if depth == MaxDepth {
			break
		}

		var next []string

		for _, id := range frontier {
			for _, parent := range graph[id] {
				if visited[parent] {
					continue
				}

				visited[parent] = true

				next = append(next, parent)
			}
		}

		frontier = next
	}

	switch {
	case indoor && outdoor:
		return planner.Mixed, true
	case indoor:
		return planner.Indoor, true
	case outdoor:
		return planner.Outdoor, true
	}

	return "", false
}

// Classify gives every counted type its label and Kind, sorted by ID. Every root and denied
// type is listed too, even when no place in the sample had it, so the generated map always
// carries the full rule list.
func (r Rules) Classify(counts map[string]int, labels map[string]string, graph ClassGraph) []TypeInfo {
	ids := make(map[string]bool, len(counts)+len(r.Roots)+len(r.Deny))

	for id := range counts {
		ids[id] = true
	}

	for id := range r.Roots {
		ids[id] = true
	}

	for id := range r.Deny {
		ids[id] = true
	}

	types := make([]TypeInfo, 0, len(ids))

	for id := range ids {
		kind, _ := r.KindOf(id, graph)
		types = append(types, TypeInfo{ID: id, Label: labels[id], Count: counts[id], Kind: kind})
	}

	SortByID(types)

	return types
}

// Kept reports whether a place with this type alone is worth a stop.
func (t TypeInfo) Kept() bool {
	return t.Kind == planner.Indoor || t.Kind == planner.Outdoor || t.Kind == planner.Mixed
}

// kindNames turns a Kind into the planner constant that names it.
var kindNames = map[planner.Kind]string{
	planner.Indoor:  "Indoor",
	planner.Outdoor: "Outdoor",
	planner.Mixed:   "Mixed",
	planner.Deny:    "Deny",
}

// Render returns the source of internal/planner/placetypes_gen.go, gofmt-ed and sorted by ID. Types with
// no Kind are left out: the map only holds what the rules decided.
func Render(types []TypeInfo) ([]byte, error) {
	sorted := make([]TypeInfo, 0, len(types))

	for _, t := range types {
		if t.Kind == "" {
			continue
		}

		sorted = append(sorted, t)
	}

	SortByID(sorted)

	var b strings.Builder

	b.WriteString("// Code generated by cmd/placetypes; DO NOT EDIT.\n\n")
	b.WriteString("package planner\n\n")
	b.WriteString("// PlaceTypes maps a Wikidata \"instance of\" (P31) type to how a place with that type\n")
	b.WriteString("// behaves in the rain. Deny types drop the place. The comment on each line is the\n")
	b.WriteString("// type's English label and how many places in the sample cities had it.\n")
	b.WriteString("var PlaceTypes = map[string]Kind{\n")

	for _, t := range sorted {
		name, ok := kindNames[t.Kind]
		if !ok {
			return nil, fmt.Errorf("type %s has unknown kind %q", t.ID, t.Kind)
		}

		fmt.Fprintf(&b, "\t%q: %s, // %s (%d)\n", t.ID, name, comment(t.Label), t.Count)
	}

	b.WriteString("}\n")

	src, err := format.Source([]byte(b.String()))
	if err != nil {
		return nil, fmt.Errorf("formatting the generated map: %w", err)
	}

	return src, nil
}

// comment keeps a label to one line, so it can't break out of the comment it sits in.
func comment(label string) string {
	if label == "" {
		return "no English label"
	}

	return strings.Join(strings.Fields(label), " ")
}

// SortByID orders types by their Wikidata number, so Q515 comes before Q1007870.
func SortByID(types []TypeInfo) {
	sort.Slice(types, func(i, j int) bool {
		a, b := types[i].ID, types[j].ID

		na, aIsNumbered := qNumber(a)
		nb, bIsNumbered := qNumber(b)

		switch {
		case aIsNumbered && bIsNumbered && na != nb:
			return na < nb
		case aIsNumbered != bIsNumbered:
			return aIsNumbered
		default:
			return a < b
		}
	})
}

// qNumber reads the number out of a Wikidata ID such as "Q33506".
func qNumber(id string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimPrefix(id, "Q"))
	if err != nil {
		return 0, false
	}

	return n, true
}
