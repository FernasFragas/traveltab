{{ if .WriterMap }}<h1 class="tt-visually-hidden">{{ .GeneralInfo.City }} travel map</h1>{{ else }}{{ template "weather_display" . }}{{ end }}

<section id="overview" class="tt-section tt-view" data-tt-view tabindex="-1" aria-label="Overview">
    {{ if .WriterMap }}{{ template "weather_display" . }}{{ end }}
    <div class="tt-overview-map">{{ template "map_card" . }}</div>
    {{ if not .WriterMap }}{{ template "guide_card" .Presentation.Guide }}{{ end }}
</section>

<section id="itinerary" class="tt-section tt-view" data-tt-view tabindex="-1" aria-label="Your itinerary" hidden>
    {{ if .Trip }}
        {{ template "trip_card" .Trip }}
    {{ else }}
        <div class="tt-empty-state">
            <h2 class="tt-section-title">Plan your trip</h2>
            <p>Search for a destination to start planning. Trip planning is unavailable until we can locate your destination.</p>
        </div>
    {{ end }}
    <div id="itinerary-results">{{ template "itinerary_body" .Trip }}</div>
</section>

<section id="stays" class="tt-section tt-view" data-tt-view tabindex="-1" aria-label="Where to stay" hidden>
    {{ if .Trip }}
        {{ template "stay_card" .Trip.Plan }}
    {{ else }}
        {{ template "stay_card" .Trip }}
    {{ end }}
</section>

<section id="videos" class="tt-section tt-view" data-tt-view tabindex="-1" aria-label="Explore through video" hidden>
    {{ template "video" . }}
</section>
