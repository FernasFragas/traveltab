package guides

// City is one of the starter cities guides.json is generated for.
type City struct {
	// Name is the app's display name for the city, and the Wikivoyage article title unless
	// wikivoyageTitles overrides it below.
	Name string
	// Country is the two-letter code the app already carries elsewhere (Request.Country).
	Country string
}

// Cities are the same 20 starter cities cmd/placetypes uses. Order and names match
// internal/placetypes/generator.go; country codes are added here for guide keys.
var Cities = []City{
	{"Lisbon", "pt"},
	{"Porto", "pt"},
	{"Tavira", "pt"},
	{"Funchal", "pt"},
	{"Kyoto", "jp"},
	{"Paris", "fr"},
	{"Rome", "it"},
	{"Barcelona", "es"},
	{"London", "gb"},
	{"New York", "us"},
	{"Tokyo", "jp"},
	{"Istanbul", "tr"},
	{"Prague", "cz"},
	{"Amsterdam", "nl"},
	{"Berlin", "de"},
	{"Seville", "es"},
	{"Florence", "it"},
	{"Vienna", "at"},
	{"Marrakesh", "ma"},
	{"Mexico City", "mx"},
}

// wikivoyageTitles overrides the Wikivoyage article title when it differs from the city's
// display name. "New York" alone is a short disambiguation-style page on Wikivoyage; the full
// article lives at "New York City".
var wikivoyageTitles = map[string]string{
	"New York": "New York City",
}

// WikivoyageTitle returns the Wikivoyage article title to fetch for this city.
func (c City) WikivoyageTitle() string {
	if title, ok := wikivoyageTitles[c.Name]; ok {
		return title
	}

	return c.Name
}

// Slug returns this city's guides.json key.
func (c City) Slug() string {
	return Slug(c.Name, c.Country)
}
