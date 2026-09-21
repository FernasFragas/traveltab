{{ template "weather_display" . }}

<nav class="tt-section-navigation" aria-label="Destination sections">
    {{ template "shell_section_links" . }}
</nav>

<section id="overview" class="tt-section" tabindex="-1" aria-labelledby="overview-title">
    <div class="tt-section-heading">
        <div>
            <h2 id="overview-title" class="tt-section-title">Overview</h2>
            <p class="tt-section-subtitle">Get oriented and plan your next few days.</p>
        </div>
    </div>
    <div class="tt-overview-grid">
        <div class="tt-overview-map">{{ template "map_card" . }}</div>
        <div class="tt-overview-planner">
            {{ if .Trip }}
                {{ template "trip_card" .Trip }}
            {{ else }}
                <div class="tt-empty-state">
                    <h3 class="tt-section-title">Plan your trip</h3>
                    <p>Search for a destination to start planning. Trip planning is unavailable until we can locate your destination.</p>
                </div>
            {{ end }}
        </div>
    </div>
</section>

<section id="itinerary" class="tt-section" tabindex="-1" aria-label="Your itinerary">
    {{ template "itinerary_body" .Trip }}
</section>

<section id="stays" class="tt-section" tabindex="-1" aria-label="Where to stay">
    {{ if .Trip }}
        {{ template "stay_card" .Trip.Plan }}
    {{ else }}
        {{ template "stay_card" .Trip }}
    {{ end }}
</section>

<section id="videos" class="tt-section" tabindex="-1" aria-label="Explore through video">
    {{ template "video" . }}
</section>
