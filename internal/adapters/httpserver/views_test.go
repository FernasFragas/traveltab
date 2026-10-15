package httpserver

import (
	"testing"

	"github.com/gofiber/template/html/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// html/template escapes a template on its first Execute, writing to it, while the gofiber engine copies
// templates without a lock. Escaping every template before the first request keeps concurrent first renders
// read-only (a data race CI caught in TestDestinationPhoto_RapidConcurrentSearchesKeepEachPhotoWithItsCity).
// html/template refuses to Clone a template once it is escaped, so Clone failing is the sign.
func TestViews_TemplatesAreEscapedBeforeTheFirstRequest(t *testing.T) {
	server := NewAppServer(nil, nil, testStore)

	engine, ok := server.app.Config().Views.(*html.Engine)
	require.True(t, ok)
	require.NotNil(t, engine.Templates)
	for _, name := range []string{"index", "content_fragment", "place_panel", "plan_response", "search_suggestions"} {
		tmpl := engine.Templates.Lookup(name)
		require.NotNil(t, tmpl, name)
		_, err := tmpl.Clone()
		assert.Error(t, err, "%s is not escaped yet", name)
	}
}
