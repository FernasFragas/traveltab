package sqlite

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testStore *Store
var db *sql.DB

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "traveltab-sqlite-test-")
	if err != nil {
		log.Fatal(err)
	}
	testStore, err = Open(filepath.Join(dir, "test.db"))
	if err != nil {
		log.Fatal(err)
	}
	db = testStore.db
	code := m.Run()
	_ = testStore.Close()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func TestOpen_CreatesTablesAndReopensExistingDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "init.db")
	for i := 0; i < 2; i++ {
		store, err := Open(path)
		require.NoError(t, err)
		for _, table := range []string{"city_data", "visits", "source_cache"} {
			var name string
			err := store.db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
			assert.NoError(t, err, "table %s should exist", table)
		}
		require.NoError(t, store.Close())
	}
	store, err := Open(filepath.Join(t.TempDir(), "missing", "invalid.db"))
	assert.Error(t, err)
	assert.Nil(t, store)
}

func TestOpen_StoresHaveIndependentConnections(t *testing.T) {
	first, err := Open(filepath.Join(t.TempDir(), "first.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = first.Close() })
	second, err := Open(filepath.Join(t.TempDir(), "second.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = second.Close() })
	require.NoError(t, first.SaveCityData("Lisbon", map[string]any{"name": "first"}))
	_, err = second.GetCityData("Lisbon")
	assert.ErrorIs(t, err, sql.ErrNoRows)
	require.NoError(t, second.Close())
	data, err := first.GetCityData("Lisbon")
	require.NoError(t, err)
	assert.Contains(t, data, "first")
}
