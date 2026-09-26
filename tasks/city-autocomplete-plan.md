# City autocomplete: "Lisbon, Portugal" suggestions

## Context
- **Now:** the header search (`views/index.go.tpl:44-47`) is a plain text box. Users have to type `City, Country` themselves. `WeatherAPI.FetchReportData` (`internal/adapters/api/openweather.go:31`) splits on the comma and **fails** if there's no country.
- **Goal:** as the user types, show a dropdown of matching places like **Lisbon, Portugal** or **Paris, France / Paris, Texas, United States**. Picking one fills the box and runs the search.
- **Constraint:** free, keyless data only (project rule). Use the **Open-Meteo Geocoding API** (`https://geocoding-api.open-meteo.com/v1/search?name=lis&count=6&language=en`). It's the same provider as the forecast and returns `name`, `country`, `country_code`, `admin1`, `latitude`, `longitude`.

## Plan

### 1. Adapter: `internal/adapters/api/openmeteo_geocoding.go` (new)
- [ ] `OpenMeteoGeocodingAPI` with `NewOpenMeteoGeocodingAPI(client *http.Client)`. Nil client means a default with a short timeout (~5 s). Same shape as `openmeteo_forecast.go`.
- [ ] `Suggest(ctx, query string) ([]application.PlaceSuggestion, error)`. Sets `User-Agent` from `useragent.go`, `count=6`, `language=en`, `format=json`.
- [ ] Map to `PlaceSuggestion{Name, Region (admin1), Country, CountryCode}`. Drop duplicates with the same name, region and country.
- [ ] Test with a `testdata/openmeteo/geocoding_lis.json` fixture served by `httptest`, following the pattern in `openmeteo_forecast_test.go`.

### 2. Application port: `internal/application/`
- [ ] Add a `PlaceSuggestion` type and a `PlaceSuggester` interface. Keeps the httpserver independent of the adapter, like `DestinationPhotoSource`.

### 3. HTTP endpoint: `internal/adapters/httpserver/suggest.go` (new)
- [ ] `GET /suggest?city_name=…`. Trim the text before the first comma. Under 2 characters → empty response.
- [ ] Render a `views/search_suggestions.go.tpl` fragment of `<li role="option">` items:
  - main line **`Name, Country`**
  - small secondary line with `Region` (tells Paris FR from Paris TX)
  - `data-value="Name, Country"` and `data-country-code="PT"`
- [ ] If the upstream call fails, return an empty list. Autocomplete must never block a normal search.
- [ ] Small in-memory TTL cache keyed by the lowercase query (for example 10 min, capped size). Cuts Open-Meteo calls while typing.
- [ ] Add `SetPlaceSuggester` setter (same style as `SetDestinationPhotoSource`) and wire it up in `cmd/web/main.go`.
- [ ] Tests: short query, happy path renders `Lisbon, Portugal`, upstream error gives an empty 200.

### 4. Make a picked suggestion resolve reliably
- OpenWeather geocoding is only dependable with **ISO country codes**. So the box shows "Lisbon, Portugal" but the search sends the code.
- [ ] Form gets a hidden `<input name="country_code">`. It's filled when a suggestion is picked and cleared when the user edits the text.
- [ ] In `listGeneralInfo` (`server.go:97`): if `country_code` is set, search `"<city part>, <CODE>"`. Otherwise keep today's behaviour, so typed and shared URLs still work.

### 5. Front end
- [ ] `views/index.go.tpl`: add combobox ARIA to the input (`role="combobox"`, `aria-autocomplete="list"`, `aria-expanded`, `aria-controls="destination-suggestions"`). Add an empty `<ul id="destination-suggestions" role="listbox" hidden>` under the form.
- [ ] Use htmx on the input: `hx-get="/suggest" hx-trigger="input changed delay:250ms" hx-target="#destination-suggestions" hx-sync="this:replace"`. Put it on the input itself so it doesn't set off the form's own `hx-get`.
- [ ] New `public/redesign/search.js`, loaded like `discovery.js`, with a `window.travelTabSearchWired` guard:
  - show or hide the list and keep `aria-expanded` in step
  - ↑/↓ to move (`aria-activedescendant`), Enter to pick, Esc or clicking outside to close
  - picking fills the input and `country_code`, then `requestSubmit()`s the form
- [ ] Styles in `public/redesign/discovery.css` (or a new `search.css`) using the existing `tt-` tokens. The dropdown sits under `.tt-search`, full width on 375 px.

## Verification
- [ ] `go test ./...` (new adapter and handler tests pass).
- [ ] `make` run locally → type `lis` → "Lisbon, Portugal" appears → pick it → the Lisbon page loads. Type `paris` → both France and Texas show, each opens the right city.
- [ ] Keyboard only: arrows, Enter, Esc. Screen reader announces the options.
- [ ] Block `geocoding-api.open-meteo.com` → no dropdown, typing "Lisbon, Portugal" + Enter still works.
- [ ] Re-run `tasks/redesign/checks/a11y-audit.mjs` and `browser-smoke.mjs` at 375 and 1440.
