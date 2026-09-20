package sqlite

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// Store owns the SQLite connection used by city, source-cache and analytics persistence.
type Store struct{ db *sql.DB }

// Open initializes a connection and creates tables for both new and existing databases.
func Open(dataSourceName string) (*Store, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}
	store := &Store{db: db}
	if err := store.initialize(); err != nil {
		_ = db.Close()
		return nil, err
	}
	log.Println("Database initialized successfully (with compressed data blob).")
	return store, nil
}

func (s *Store) initialize() error {
	if err := s.db.Ping(); err != nil {
		return fmt.Errorf("error pinging database: %w", err)
	}
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS city_data (city TEXT PRIMARY KEY, data BLOB);`); err != nil {
		return fmt.Errorf("error creating table: %w", err)
	}
	if err := s.createVisitsTable(); err != nil {
		return err
	}
	return createSourceCacheTable(s.db)
}

// createSourceCacheTable adds the table the planner's source cache stores its gzipped JSON in.
// It takes the connection so an existing database file can be migrated without touching db.
func createSourceCacheTable(conn *sql.DB) error {
	createTableSQL := `CREATE TABLE IF NOT EXISTS source_cache (
        key TEXT PRIMARY KEY,
        fetched_at INTEGER,
        data BLOB
    );`

	if _, err := conn.Exec(createTableSQL); err != nil {
		return fmt.Errorf("error creating source_cache table: %w", err)
	}

	return nil
}

// compressJSON marshals a value to JSON and gzips it, the shape every blob in the database has.
func compressJSON(value any) ([]byte, error) {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("error marshaling data to JSON: %w", err)
	}

	var compressedBuf bytes.Buffer
	gzWriter := gzip.NewWriter(&compressedBuf)
	if _, err := gzWriter.Write(jsonData); err != nil {
		gzwCloseErr := gzWriter.Close() // Try to close even on write error
		log.Printf("Gzip writer close error after write error: %v", gzwCloseErr)
		return nil, fmt.Errorf("error compressing JSON data: %w", err)
	}
	if err := gzWriter.Close(); err != nil { // Close flushes the compressed data
		return nil, fmt.Errorf("error closing gzip writer: %w", err)
	}

	return compressedBuf.Bytes(), nil
}

// decompressBlob gunzips a blob read from the database.
func decompressBlob(compressedData []byte) ([]byte, error) {
	gzReader, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		// Data might not be compressed? Or corrupted.
		return nil, fmt.Errorf("error reading compressed data: %w", err)
	}
	defer func() { _ = gzReader.Close() }()

	decompressed, err := io.ReadAll(gzReader)
	if err != nil {
		return nil, fmt.Errorf("error decompressing data: %w", err)
	}

	return decompressed, nil
}

// SaveCityData serializes the provided data map to JSON, compresses it,
// and saves/updates it in the database as a BLOB.
func (s *Store) SaveCityData(city string, data map[string]any) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("database is not initialized")
	}

	// 1. Marshal the data map into JSON and compress it with gzip.
	compressedData, err := compressJSON(data)
	if err != nil {
		return fmt.Errorf("error compressing data for city %s: %w", city, err)
	}

	cityName := strings.ToLower(city)

	// 2. Insert or replace compressed BLOB data based on the primary key (city).
	insertSQL := `REPLACE INTO city_data (city, data) VALUES (?, ?);`

	// Execute the SQL statement.
	_, err = s.db.Exec(insertSQL, cityName, compressedData)
	if err != nil {
		return fmt.Errorf("error saving compressed data for city %s: %w", city, err)
	}

	log.Printf("Successfully saved compressed data for city: %s", city)
	return nil
}

// GetCityData retrieves the compressed BLOB data for a specific city,
// decompresses it, and returns the original JSON data as a string.
// It returns sql.ErrNoRows if the city is not found.
func (s *Store) GetCityData(city string) (string, error) {
	if s == nil || s.db == nil {
		return "", fmt.Errorf("database is not initialized")
	}

	querySQL := `SELECT data FROM city_data WHERE city = ?;`
	cityName := strings.ToLower(city)

	// 1. Query the compressed BLOB data.
	var compressedData []byte
	err := s.db.QueryRow(querySQL, cityName).Scan(&compressedData)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", sql.ErrNoRows // Not an error, just cache miss
		}
		// For other errors, wrap them.
		return "", fmt.Errorf("error querying compressed data for city %s: %w", city, err)
	}

	// Handle case where blob might be empty/null in DB
	if len(compressedData) == 0 {
		log.Printf("Warning: Found empty data blob for city %s. Returning empty string.", city)
		return "", nil // Or perhaps return sql.ErrNoRows? Decide based on expected behavior.
	}

	// 2. Decompress the data using gzip.
	decompressedJSON, err := decompressBlob(compressedData)
	if err != nil {
		// Data might not be compressed? Or corrupted.
		log.Printf("Error decompressing data for city %s (data might be uncompressed or corrupt): %v", city, err)
		return "", fmt.Errorf("error decompressing data for city %s: %w", city, err)
	}

	log.Printf("Successfully retrieved and decompressed data for city: %s", city)
	// 3. Return the original JSON data as a string.
	return string(decompressedJSON), nil
}

// Close releases the store's connection. The caller owns the store lifetime.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}
