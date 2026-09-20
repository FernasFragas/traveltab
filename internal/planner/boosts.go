package planner

// Boosts lists Wikidata places that are worth a visit even when their types say
// otherwise. A listed place skips the type filter, takes the Kind given here, and is
// ranked as if it were the tenth most famous place around.
var Boosts = map[string]Kind{
	"Q652806":   Indoor,
	"Q168001":   Outdoor,
	"Q2063403":  Indoor,
	"Q23579173": Outdoor,
	"Q11650434": Indoor,
}
