package httpserver

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// As a traveller, I want the page's scripts and styles to update when the site changes,
// so that I don't keep running yesterday's copy from my browser cache.
func TestStatic_AssetsAreRevalidatedOnEveryLoad(t *testing.T) {
	server := NewAppServer(nil, nil, testStore)

	for _, path := range []string{"/redesign/search.js", "/redesign/search.css", "/styles.css"} {
		resp, err := server.app.Test(httptest.NewRequest("GET", path, nil), 5000)
		require.NoError(t, err)
		_ = resp.Body.Close()

		require.Equal(t, 200, resp.StatusCode, path)
		assert.Equal(t, "no-cache", resp.Header.Get("Cache-Control"), path)
	}
}
