package httpserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPlanRoute_LinksToItsOwnExportsAndMapsRoutes proves /plan's own response links to the
// export routes with a slug and query the export handlers can actually resolve - not just
// that each piece works in isolation.
func TestPlanRoute_LinksToItsOwnExportsAndMapsRoutes(t *testing.T) {
	server := planServer(t, fakeTripPlanner{})

	status, body := doRequest(t, server, planRequest(nil))
	require.Equal(t, 200, status)

	// html/template correctly escapes "&" to "&amp;" inside an HTML attribute.
	assert.Contains(t, body, `/trip/lisbon-portugal.ics?days=3&amp;from=2026-03-11`)
	assert.Contains(t, body, `/trip/lisbon-portugal.kml?days=3&amp;from=2026-03-11`)
	assert.Contains(t, body, "https://www.google.com/maps/dir/?")
}
