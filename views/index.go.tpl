<!doctype html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
    {{ if .PageTitle }}<title>{{ .PageTitle }}</title>{{ else }}<title>✈🌤️TravelTab</title>{{ end }}
    {{ if .PageDescription }}<meta name="description" content="{{ .PageDescription }}">{{ end }}
    {{ if .CanonicalURL }}<link rel="canonical" href="{{ .CanonicalURL }}">{{ end }}
    <meta name="theme-color" content="#F7F3EB">
    <link rel="icon" href="/traveltab.png" type="image/png">
    <!-- Bootstrap utilities support the existing planner until task 04 replaces it. -->
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.2.3/dist/css/bootstrap.min.css" rel="stylesheet" integrity="sha384-rbsA2VBKQhggwzxH7pPCaAqO46MgnOM80zW1RWuH61DGLwZJEdK2Kadq2F9CUG65" crossorigin="anonymous">
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.3/font/bootstrap-icons.min.css">
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="/styles.css">
    <link rel="stylesheet" href="/redesign/destination.css">
    <link rel="stylesheet" href="/redesign/planner.css">
    <link rel="stylesheet" href="/redesign/discovery.css">
    <link rel="stylesheet" href="/redesign/search.css">
    <link rel="stylesheet" href="/redesign/guide.css">
    <script defer src="https://unpkg.com/htmx.org@1.9.11" integrity="sha384-0gxUXCCR8yv9FM2b+U3FDbsKthCI66oH5IA9fHppQq9DDMHuMauqq1ZHBpJxQ0J0" crossorigin="anonymous"></script>
    <script defer src="/redesign/navigation.js"></script>
    <script defer src="/redesign/planner.js"></script>
    <script defer src="/redesign/discovery.js"></script>
    <script defer src="/redesign/search.js"></script>
</head>
<body>
    <a class="tt-skip-link" href="#overview">Skip to trip overview</a>
    <div class="tt-shell">
        <header class="tt-header">
            <a class="tt-brand" href="/" aria-label="TravelTab home"><i class="bi bi-signpost-split-fill" aria-hidden="true"></i>TravelTab</a>
            <button class="tt-menu-button tt-button tt-button-quiet" type="button" aria-expanded="false" aria-controls="header-navigation" aria-label="Open section menu" hidden>
                <i class="bi bi-list" aria-hidden="true"></i>
            </button>
            <nav id="header-navigation" class="tt-header-navigation" aria-label="Main navigation">
                <a href="#overview" data-tt-section="overview" aria-current="location">Overview</a>
                <a href="#itinerary" data-tt-section="itinerary">Itinerary</a>
                <a href="#stays" data-tt-section="stays">Stays</a>
                <a href="#videos" data-tt-section="videos">Videos</a>
            </nav>
            <div class="tt-search-area">
                <form id="destination-search" class="tt-search" action="/process-form/" method="get" role="search"
                      hx-get="/process-form/" hx-target="#content-area" hx-swap="innerHTML" hx-indicator="#destination-search-spinner">
                    <label class="tt-visually-hidden" for="city_name">Search destination, including country</label>
                    <i class="bi bi-search" aria-hidden="true"></i>
                    <input id="city_name" type="search" name="city_name" placeholder="Search destinations…" required autocomplete="off" aria-describedby="destination-search-error" role="combobox" aria-autocomplete="list" aria-expanded="false" aria-controls="destination-suggestions"
                           hx-get="/suggest" hx-trigger="input delay:250ms" hx-target="#destination-suggestions" hx-swap="innerHTML" hx-sync="this:replace" hx-params="city_name">
                    <input id="country_code" type="hidden" name="country_code" value="">
                    <input id="place_lat" type="hidden" name="place_lat" value="">
                    <input id="place_lon" type="hidden" name="place_lon" value="">
                    <button type="submit" class="tt-search-submit" aria-label="Search destination"><i class="bi bi-arrow-right" aria-hidden="true"></i></button>
                    <span id="destination-search-spinner" class="tt-search-spinner htmx-indicator" role="status"><span class="tt-spinner" aria-hidden="true"></span><span class="tt-visually-hidden">Finding destination…</span></span>
                </form>
                <ul id="destination-suggestions" class="tt-suggestions" role="listbox" aria-label="Suggested destinations" hidden></ul>
                <p id="destination-search-error" class="tt-error" role="alert" aria-atomic="true"></p>
                <p id="destination-search-status" class="tt-visually-hidden" role="status" aria-atomic="true"></p>
            </div>
        </header>

        <main id="content-area" class="tt-content">
            {{ template "content_fragment" . }}
        </main>

        <footer class="tt-footer">
            <div class="tt-footer-main">
                <a class="tt-brand tt-footer-brand" href="/">TravelTab</a>
                <p>Made by Fernando Fragateiro</p>
                <nav class="tt-footer-links" aria-label="Creator links">
                    <a href="https://github.com/FernasFragas" target="_blank" rel="noopener noreferrer" aria-label="GitHub (opens in new tab)"><i class="bi bi-github" aria-hidden="true"></i></a>
                    <a href="https://pt.linkedin.com/in/fernando-paulo-fragateiro-the1" target="_blank" rel="noopener noreferrer" aria-label="LinkedIn (opens in new tab)"><i class="bi bi-linkedin" aria-hidden="true"></i></a>
                    <a href="https://medium.com/@patronfragas" target="_blank" rel="noopener noreferrer" aria-label="Medium blog (opens in new tab)"><i class="bi bi-medium" aria-hidden="true"></i></a>
                    <a href="https://www.fernandofragateiro.com" target="_blank" rel="noopener noreferrer" aria-label="Fernando's website (opens in new tab)"><i class="bi bi-globe" aria-hidden="true"></i></a>
                </nav>
            </div>
            <p class="tt-credits">Places: Wikipedia &amp; Wikidata · City guides: Wikivoyage (CC BY-SA 4.0) · Photos: Wikimedia Commons · Weather: Open-Meteo (CC BY 4.0) · Map data © OpenStreetMap contributors</p>
        </footer>
    </div>
    <nav class="tt-bottom-navigation" aria-label="Quick section navigation">
        {{ template "shell_section_links" . }}
    </nav>
</body>
</html>
