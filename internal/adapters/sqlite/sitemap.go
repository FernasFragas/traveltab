package sqlite

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
)

// legacySitemapIndexKey is where earlier versions stored the sitemap's slug list: a fake city
// row in city_data holding {"slugs": [...]}. Open still reads that row once so existing
// /trip/:slug URLs survive the upgrade, then removes it.
const legacySitemapIndexKey = "__sitemap_index__"

func createSitemapSlugsTable(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS sitemap_slugs (
        slug TEXT PRIMARY KEY,
        added_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
    );`); err != nil {
		return fmt.Errorf("error creating sitemap_slugs table: %w", err)
	}
	return nil
}

// migrateLegacySitemapIndex copies the slugs of a legacy __sitemap_index__ city_data row into
// sitemap_slugs and deletes the row, in one transaction. It is idempotent, and an unreadable
// legacy row is logged and left in place rather than blocking startup.
func migrateLegacySitemapIndex(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error starting sitemap migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var blob []byte
	err = tx.QueryRow(`SELECT data FROM city_data WHERE city = ?;`, legacySitemapIndexKey).Scan(&blob)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("error reading legacy sitemap index: %w", err)
	}

	var legacy struct {
		Slugs []string `json:"slugs"`
	}
	raw, err := decompressBlob(blob)
	if err == nil {
		err = json.Unmarshal(raw, &legacy)
	}
	if err != nil {
		log.Printf("Skipping unreadable legacy sitemap index: %v", err)
		return nil
	}

	for _, slug := range legacy.Slugs {
		if slug == "" {
			continue
		}
		if _, err := tx.Exec(`INSERT OR IGNORE INTO sitemap_slugs (slug) VALUES (?);`, slug); err != nil {
			return fmt.Errorf("error migrating sitemap slug %s: %w", slug, err)
		}
	}
	if _, err := tx.Exec(`DELETE FROM city_data WHERE city = ?;`, legacySitemapIndexKey); err != nil {
		return fmt.Errorf("error removing legacy sitemap index: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing sitemap migration: %w", err)
	}
	return nil
}

// RecordSitemapSlug persists a /trip/:slug slug so /sitemap.xml lists it. Each slug is its own
// row and the insert is a no-op for a slug already recorded, so concurrent writers cannot
// drop each other's entries.
func (s *Store) RecordSitemapSlug(slug string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("database is not initialized")
	}
	if slug == "" {
		return nil
	}
	if _, err := s.db.Exec(`INSERT OR IGNORE INTO sitemap_slugs (slug) VALUES (?);`, slug); err != nil {
		return fmt.Errorf("error recording sitemap slug %s: %w", slug, err)
	}
	return nil
}

// ListSitemapSlugs returns every recorded slug in ascending order, or an empty slice when none
// have been recorded. It never reads city data or calls a provider.
func (s *Store) ListSitemapSlugs() ([]string, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("database is not initialized")
	}
	rows, err := s.db.Query(`SELECT slug FROM sitemap_slugs ORDER BY slug;`)
	if err != nil {
		return nil, fmt.Errorf("error listing sitemap slugs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	slugs := []string{}
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, fmt.Errorf("error scanning sitemap slug: %w", err)
		}
		slugs = append(slugs, slug)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error listing sitemap slugs: %w", err)
	}
	return slugs, nil
}
