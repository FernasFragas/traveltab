package writermap

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchDestination_NestedNamesAndCase(t *testing.T) {
	base := BasePlaces{Gazetteer: []Place{
		{QID: "Q1", Names: []string{"Alcântara"}},
		{QID: "Q2", Names: []string{"São Pedro de Alcântara"}},
		{QID: "Q3", Names: []string{"Praia"}},
	}}
	post := IndexedPost{Host: "writer.test", PostID: 1, URL: "https://writer.test/1", Title: "Lisbon walk"}
	rows := []MatchRow{
		{Post: post, Phrase: NamePhrase{Folded: "lisbon", OriginalCase: true, Start: 0, End: 6}},
		{Post: post, Phrase: NamePhrase{Folded: "sao pedro de alcantara", OriginalCase: true, Start: 20, End: 42}},
		{Post: post, Phrase: NamePhrase{Folded: "alcantara", OriginalCase: true, Start: 33, End: 42}},
		{Post: post, Phrase: NamePhrase{Folded: "praia", OriginalCase: false, Start: 50, End: 55}},
	}

	got := MatchDestination(base, rows, []string{"Lisbon"}, nil, MatchOverrides{})

	assert.Empty(t, got.Mentions["Q1"])
	require.Len(t, got.Mentions["Q2"], 1)
	assert.Equal(t, []Position{{Start: 20, End: 42}}, got.Mentions["Q2"][0].Positions)
	assert.Empty(t, got.Mentions["Q3"])
}

func TestMatchDestination_OverridesMergeAndRemoveAliases(t *testing.T) {
	base := BasePlaces{Pins: []Place{{QID: "Q1", Names: []string{"Igreja e Torre dos Clérigos"}}, {QID: "Q2", Names: []string{"Igreja dos Clérigos"}}, {QID: "Q3", Names: []string{"Torre dos Clérigos"}}}, Gazetteer: []Place{
		{QID: "Q1", Names: []string{"Igreja e Torre dos Clérigos"}},
		{QID: "Q2", Names: []string{"Igreja dos Clérigos"}},
		{QID: "Q3", Names: []string{"Torre dos Clérigos"}},
		{QID: "Q4", Names: []string{"Teleférico do Funchal", "Monte"}},
	}}
	post := IndexedPost{Host: "writer.test", PostID: 2}
	rows := []MatchRow{
		{Post: post, Phrase: NamePhrase{Folded: "porto", OriginalCase: true, Start: 0, End: 5}},
		{Post: post, Phrase: NamePhrase{Folded: "igreja dos clerigos", OriginalCase: true, Start: 10, End: 29}},
		{Post: post, Phrase: NamePhrase{Folded: "torre dos clerigos", OriginalCase: true, Start: 35, End: 53}},
		{Post: post, Phrase: NamePhrase{Folded: "monte", OriginalCase: true, Start: 60, End: 65}},
	}
	overrides := MatchOverrides{
		Merges:        []PlaceMerge{{CanonicalName: "Igreja e Torre dos Clérigos", Names: []string{"Igreja dos Clérigos", "Torre dos Clérigos"}}},
		AliasRemovals: []AliasRemoval{{PlaceName: "Teleférico do Funchal", Aliases: []string{"Monte"}}},
	}

	got := MatchDestination(base, rows, []string{"Porto"}, nil, overrides)

	require.Len(t, got.Mentions["Q1"], 1)
	assert.Len(t, got.Mentions["Q1"][0].Positions, 2)
	assert.Empty(t, got.Mentions["Q2"])
	assert.Empty(t, got.Mentions["Q3"])
	assert.Empty(t, got.Mentions["Q4"])
	assert.Equal(t, 1, got.WriterCounts["Q1"])
	assert.Equal(t, []string{"Q1"}, placeIDs(got.Places))
}

func TestMatchDestination_ContextTypesRankingAndCounts(t *testing.T) {
	base := BasePlaces{
		Pins: []Place{{QID: "Q1", Names: []string{"Cabo Girão"}, Sitelinks: 20}},
		Gazetteer: []Place{
			{QID: "Q1", Names: []string{"Cabo Girão"}, Sitelinks: 20},
			{QID: "Q2", Names: []string{"São Jorge"}, Types: []string{"Q1131296"}, Sitelinks: 1},
			{QID: "Q3", Names: []string{"Monte"}, Types: []string{"Q1131296"}},
			{QID: "Q4", Names: []string{"Ponta do Sol"}, Types: []string{"Q13217644"}},
			{QID: "Q5", Names: []string{"Teleférico do Funchal"}, Types: []string{"Q13409250"}},
			{QID: "Q6", Names: []string{"Forbidden"}, Types: []string{"Q515"}},
		},
	}
	posts := []IndexedPost{{Host: "a.test", PostID: 1}, {Host: "a.test", PostID: 2}, {Host: "b.test", PostID: 1}}
	var rows []MatchRow
	for _, post := range posts {
		rows = append(rows, MatchRow{Post: post, Phrase: NamePhrase{Folded: "madeira", OriginalCase: true, Start: 0, End: 7}})
		for i, name := range []string{"sao jorge", "monte", "ponta do sol", "teleferico do funchal", "forbidden"} {
			rows = append(rows, MatchRow{Post: post, Phrase: NamePhrase{Folded: name, OriginalCase: true, Start: 20 + i*30, End: 20 + i*30 + len(name)}})
		}
	}
	braga := IndexedPost{Host: "c.test", PostID: 1}
	rows = append(rows, MatchRow{Post: braga, Phrase: NamePhrase{Folded: "bom jesus do monte", OriginalCase: true, Start: 0, End: 18}})
	rows = append(rows, MatchRow{Post: braga, Phrase: NamePhrase{Folded: "monte", OriginalCase: true, Start: 13, End: 18}})

	got := MatchDestination(base, rows, []string{"Madeira", "Funchal"}, nil, MatchOverrides{})

	assert.Equal(t, 2, got.WriterCounts["Q2"])
	assert.Equal(t, 2, got.WriterCounts["Q3"])
	assert.Equal(t, 2, got.WriterCounts["Q4"])
	assert.Equal(t, 2, got.WriterCounts["Q5"])
	assert.Len(t, got.Mentions["Q3"], 3)
	assert.Empty(t, got.Mentions["Q6"])
	assert.Equal(t, []string{"Q2", "Q3", "Q4", "Q5", "Q1"}, placeIDs(got.Places))
}

func TestMatchDestination_DayContextAndStoplist(t *testing.T) {
	base := BasePlaces{Gazetteer: []Place{{QID: "Q1", Names: []string{"Monte"}}, {QID: "Q2", Names: []string{"Praia"}}}}
	post := IndexedPost{Host: "writer.test", PostID: 1, Extracted: ExtractedPost{Days: []DaySection{{Number: 1, Start: 10, End: 1000}, {Number: 2, Start: 1000, End: 2000}}}}
	rows := []MatchRow{
		{Post: post, Phrase: NamePhrase{Folded: "madeira", OriginalCase: true, Start: 15, End: 22}},
		{Post: post, Phrase: NamePhrase{Folded: "monte", OriginalCase: true, Start: 30, End: 35}},
		{Post: post, Phrase: NamePhrase{Folded: "monte", OriginalCase: true, Start: 1520, End: 1525}},
		{Post: post, Phrase: NamePhrase{Folded: "praia", OriginalCase: true, Start: 40, End: 45}},
	}

	got := MatchDestination(base, rows, []string{"Madeira"}, nil, MatchOverrides{Stoplist: []string{"Praia"}})

	require.Len(t, got.Mentions["Q1"], 1)
	assert.Equal(t, []Position{{Start: 30, End: 35}}, got.Mentions["Q1"][0].Positions)
	assert.Empty(t, got.Mentions["Q2"])
}

func TestMatchDestination_SaoVicenteIgnoresPassingLisbon(t *testing.T) {
	base := BasePlaces{Gazetteer: []Place{{QID: "Q1", Names: []string{"São Vicente"}}}}
	post := IndexedPost{Host: "writer.test", PostID: 1, Title: "18 Things to do in Madeira, Portugal"}
	rows := []MatchRow{
		{Post: post, Phrase: NamePhrase{Folded: "lisbon", OriginalCase: true, Start: 50, End: 56}},
		{Post: post, Phrase: NamePhrase{Folded: "sao vicente", OriginalCase: true, Start: 2000, End: 2011}},
	}

	got := MatchDestination(base, rows, []string{"Lisbon"}, nil, MatchOverrides{})

	assert.Empty(t, got.Mentions["Q1"])
}

func TestMatchDestination_TitleContextKeepsDistantPlace(t *testing.T) {
	base := BasePlaces{Gazetteer: []Place{{QID: "Q1", Names: []string{"São Vicente"}}}}
	post := IndexedPost{Host: "writer.test", PostID: 1, Title: "Lisbon guide"}
	rows := []MatchRow{
		{Post: post, Phrase: NamePhrase{Folded: "lisbon", OriginalCase: true, Start: 0, End: 6}},
		{Post: post, Phrase: NamePhrase{Folded: "sao vicente", OriginalCase: true, Start: 2000, End: 2011}},
	}

	got := MatchDestination(base, rows, []string{"Lisbon"}, nil, MatchOverrides{})

	require.Len(t, got.Mentions["Q1"], 1)
}

func TestMatchDestination_TitleBoundaryUsesRenderedRunes(t *testing.T) {
	base := BasePlaces{Gazetteer: []Place{{QID: "Q1", Names: []string{"São Vicente"}}}}
	post := IndexedPost{Host: "writer.test", PostID: 1, Title: "Madeira &amp; Funchal"}
	rows := []MatchRow{
		{Post: post, Phrase: NamePhrase{Folded: "lisbon", OriginalCase: true, Start: 19, End: 25}},
		{Post: post, Phrase: NamePhrase{Folded: "sao vicente", OriginalCase: true, Start: 2000, End: 2011}},
	}

	got := MatchDestination(base, rows, []string{"Lisbon"}, nil, MatchOverrides{})

	assert.Empty(t, got.Mentions["Q1"])
}

func TestMatchDestination_NearbyContextUsesRuneDistanceBothWays(t *testing.T) {
	base := BasePlaces{Gazetteer: []Place{{QID: "Q1", Names: []string{"São Vicente"}}}}
	post := IndexedPost{Host: "writer.test", PostID: 1, Title: "A guide"}
	rows := []MatchRow{
		{Post: post, Phrase: NamePhrase{Folded: "lisbon", OriginalCase: true, Start: 30, End: 36}},
		{Post: post, Phrase: NamePhrase{Folded: "sao vicente", OriginalCase: true, Start: 336, End: 347}},
		{Post: post, Phrase: NamePhrase{Folded: "sao vicente", OriginalCase: true, Start: 500, End: 511}},
		{Post: post, Phrase: NamePhrase{Folded: "lisbon", OriginalCase: true, Start: 811, End: 817}},
		{Post: post, Phrase: NamePhrase{Folded: "sao vicente", OriginalCase: true, Start: 2000, End: 2011}},
	}

	got := MatchDestination(base, rows, []string{"Lisbon"}, nil, MatchOverrides{})

	require.Len(t, got.Mentions["Q1"], 1)
	assert.Equal(t, []Position{{Start: 336, End: 347}, {Start: 500, End: 511}}, got.Mentions["Q1"][0].Positions)
}

func TestMatchDestination_ContextJustBeyondWindowDoesNotCount(t *testing.T) {
	base := BasePlaces{Gazetteer: []Place{{QID: "Q1", Names: []string{"São Vicente"}}}}
	post := IndexedPost{Host: "writer.test", PostID: 1, Title: "A guide"}
	rows := []MatchRow{
		{Post: post, Phrase: NamePhrase{Folded: "lisbon", OriginalCase: true, Start: 30, End: 36}},
		{Post: post, Phrase: NamePhrase{Folded: "sao vicente", OriginalCase: true, Start: 1037, End: 1048}},
	}

	got := MatchDestination(base, rows, []string{"Lisbon"}, nil, MatchOverrides{})

	assert.Empty(t, got.Mentions["Q1"])
}

func TestMatchDestination_ContextInsidePlaceCounts(t *testing.T) {
	base := BasePlaces{Gazetteer: []Place{{QID: "Q1", Names: []string{"Lisbon Cathedral"}}}}
	post := IndexedPost{Host: "writer.test", PostID: 1, Title: "A guide"}
	rows := []MatchRow{
		{Post: post, Phrase: NamePhrase{Folded: "lisbon", OriginalCase: true, Start: 1200, End: 1206}},
		{Post: post, Phrase: NamePhrase{Folded: "lisbon cathedral", OriginalCase: true, Start: 1200, End: 1215}},
	}

	got := MatchDestination(base, rows, []string{"Lisbon"}, nil, MatchOverrides{})

	require.Len(t, got.Mentions["Q1"], 1)
}

func TestMatchDestination_DayContextDoesNotUseIntroduction(t *testing.T) {
	base := BasePlaces{Gazetteer: []Place{{QID: "Q1", Names: []string{"São Vicente"}}}}
	post := IndexedPost{Host: "writer.test", PostID: 1, Title: "A guide", Extracted: ExtractedPost{Days: []DaySection{{Number: 1, Start: 100, End: 2000}}}}
	rows := []MatchRow{
		{Post: post, Phrase: NamePhrase{Folded: "lisbon", OriginalCase: true, Start: 20, End: 26}},
		{Post: post, Phrase: NamePhrase{Folded: "sao vicente", OriginalCase: true, Start: 1500, End: 1511}},
	}

	got := MatchDestination(base, rows, []string{"Lisbon"}, nil, MatchOverrides{})

	assert.Empty(t, got.Mentions["Q1"])
}

func TestMatchDestination_SameDayContextKeepsDistantPlace(t *testing.T) {
	base := BasePlaces{Gazetteer: []Place{{QID: "Q1", Names: []string{"São Vicente"}}}}
	post := IndexedPost{Host: "writer.test", PostID: 1, Title: "A guide", Extracted: ExtractedPost{Days: []DaySection{{Number: 1, Start: 100, End: 2000}}}}
	rows := []MatchRow{
		{Post: post, Phrase: NamePhrase{Folded: "lisbon", OriginalCase: true, Start: 120, End: 126}},
		{Post: post, Phrase: NamePhrase{Folded: "sao vicente", OriginalCase: true, Start: 1500, End: 1511}},
	}

	got := MatchDestination(base, rows, []string{"Lisbon"}, nil, MatchOverrides{})

	require.Len(t, got.Mentions["Q1"], 1)
}

func TestMatchLookupPhrases_UsesAliasesAndDestinationContext(t *testing.T) {
	places := []Place{{QID: "Q1", Names: []string{"Teleférico do Funchal", "Monte"}}, {QID: "Q2", Names: []string{"Praia"}}}
	overrides := MatchOverrides{AliasRemovals: []AliasRemoval{{PlaceName: "Teleférico do Funchal", Aliases: []string{"Monte"}}}, Stoplist: []string{"Praia"}}

	got := MatchLookupPhrases(places, []string{"Madeira", "Funchal"}, overrides)

	assert.Equal(t, []string{"funchal", "madeira", "teleferico do funchal"}, got)
}

func TestMatchDestination_CapsRankedPlaces(t *testing.T) {
	base := BasePlaces{Pins: make([]Place, 200), Gazetteer: make([]Place, 201)}
	for i := range base.Gazetteer {
		place := Place{QID: fmt.Sprintf("Q%d", i+1), Names: []string{fmt.Sprintf("Place %d", i+1)}, Sitelinks: 201 - i}
		base.Gazetteer[i] = place
		if i < 200 {
			base.Pins[i] = place
		}
	}
	post := IndexedPost{Host: "writer.test", PostID: 1}
	rows := []MatchRow{
		{Post: post, Phrase: NamePhrase{Folded: "madeira", OriginalCase: true, Start: 0, End: 7}},
		{Post: post, Phrase: NamePhrase{Folded: "place 201", OriginalCase: true, Start: 10, End: 19}},
	}

	got := MatchDestination(base, rows, []string{"Madeira"}, nil, MatchOverrides{})

	require.Len(t, got.Places, 200)
	assert.Equal(t, "Q201", got.Places[0].QID)
	assert.NotContains(t, placeIDs(got.Places), "Q200")
}

func placeIDs(places []Place) []string {
	ids := make([]string, len(places))
	for i, place := range places {
		ids[i] = place.QID
	}
	return ids
}
