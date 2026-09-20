# Task 6 notes: SQLite source cache

Files owned: `sourcecache.go`, `sourcecache_test.go`, `database.go` (table + shared gzip helpers), this file.

## RED (before any implementation)

`sourcecache.go` was a stub whose `cached[T].get` returned zero values, and `InitDB` did not
create `source_cache`. All 8 tests from the todo list failed with assertion failures (not
compile errors).

```
$ CGO_ENABLED=1 go test -race -count=1 -run 'SourceCache|InitDB' .
--- FAIL: TestSourceCache_ServesAFreshEntryFromTheCache (0.00s)
    sourcecache_test.go:56: "[]" should have 1 item(s), but has 0
--- FAIL: TestSourceCache_RefreshesAnExpiredEntry (0.00s)
    sourcecache_test.go:61: "[]" should have 1 item(s), but has 0
--- FAIL: TestSourceCache_ServesOldDataWhenTheRefreshFails (0.00s)
    sourcecache_test.go:67: "[]" should have 1 item(s), but has 0
--- FAIL: TestSourceCache_ReturnsTheErrorWhenNothingIsCached (0.00s)
    sourcecache_test.go:76: Target error should be in err chain:
                            expected: "offline"
                            in chain:
--- FAIL: TestSourceCache_SharesEntriesForNearbyCoordinates (0.00s)
    sourcecache_test.go:80: "[]" should have 1 item(s), but has 0
--- FAIL: TestSourceCache_ExpiresForecastsAfterThreeHours (0.00s)
    sourcecache_test.go:95: Expected value not to be nil.
--- FAIL: TestSourceCache_KeepsSourcesApart (0.00s)
    sourcecache_test.go:104: "[]" should have 1 item(s), but has 0
--- FAIL: TestInitDB_CreatesTheSourceCacheTable (0.00s)
    sourcecache_test.go:113: Received unexpected error:
                            sql: no rows in result set
FAIL    weatherservice  0.392s
```

### Second RED, for the extra case in `TestInitDB_CreatesTheSourceCacheTable`

The todo says the test covers "an existing database file opens and gets the table", so the test
also builds a pre-planner database file (only `city_data`) and runs the migration on it. With
`createSourceCacheTable` added as a stub returning `nil`, that part failed on its own:

```
$ CGO_ENABLED=1 go test -race -count=1 -run 'InitDB' .
--- FAIL: TestInitDB_CreatesTheSourceCacheTable (0.00s)
    sourcecache_test.go:120:
        	Error Trace:	sourcecache_test.go:115
        	            				sourcecache_test.go:120
        	Error:      	Received unexpected error:
        	            	sql: no rows in result set
FAIL    weatherservice  0.400s
```

## Decisions

- `createSourceCacheTable(d *sql.DB)` takes the connection instead of using the global `db` like
  `createVisitsTable()` does, so the migration can be tested on a throwaway file without swapping
  the package-global `db` (a swap would race with the background city save in `server.go`).
- Key is `"<source>:<lat>:<lon>"` with lat/lon at 3 decimals (~110 m), so nearby coordinates share
  an entry and sources never mix.
- `fetched_at` is stored as Unix seconds; TTL check is `now - fetched_at < ttl`, so an entry that
  is exactly `ttl` old is refreshed (the forecast test expects "call 2" at exactly 3 h).
- A refresh failure with an entry in the table logs and returns the old data; only an empty cache
  returns the error (wrapped with `%w`).
- A missing/corrupt row, or a database that is not initialised, is treated as a cache miss, so the
  cache never breaks a source; write failures are logged only.
- gzip encode/decode moved into `compressJSON` / `decompressBlob` in `database.go` and reused by
  `SaveCityData`, `GetCityData` and the cache.

## GREEN

```
$ CGO_ENABLED=1 go test -race -count=1 -v -run 'SourceCache|InitDB' .
--- PASS: TestSourceCache_ServesAFreshEntryFromTheCache (0.00s)
--- PASS: TestSourceCache_RefreshesAnExpiredEntry (0.00s)
--- PASS: TestSourceCache_ServesOldDataWhenTheRefreshFails (0.00s)
    (logs: Refreshing TestSourceCache_ServesOldDataWhenTheRefreshFails:38.722:-9.139 failed,
           serving data cached at 2026-09-19T12:00:00Z: offline)
--- PASS: TestSourceCache_ReturnsTheErrorWhenNothingIsCached (0.00s)
--- PASS: TestSourceCache_SharesEntriesForNearbyCoordinates (0.00s)
--- PASS: TestSourceCache_ExpiresForecastsAfterThreeHours (0.00s)
--- PASS: TestSourceCache_KeepsSourcesApart (0.00s)
--- PASS: TestInitDB_CreatesTheSourceCacheTable (0.00s)
ok      weatherservice  1.301s

$ CGO_ENABLED=1 go test -race -count=1 .     # whole root package
ok      weatherservice  1.526s

$ golangci-lint run .
0 issues.

$ CGO_ENABLED=1 go build ./...               # clean
```

`make lint` fails only on `gofmt -l` reporting `api/wikimedia.go`, another lane's file (Task 3);
the root package is formatted and lints clean.

## Refactor

- `SaveCityData` and `GetCityData` now call the shared `compressJSON` / `decompressBlob` helpers,
  so the cache stores blobs exactly the way city data is stored instead of copying the gzip code.

## Known limit

The tests share the package database through `TestMain` and drive a fake clock, so they are
written for `-count=1` (what the Makefile and the task's commands use). Under `-count=2` the
second pass would read the entry the first pass wrote at a later fake time.
