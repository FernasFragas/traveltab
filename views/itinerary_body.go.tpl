<div class="itinerary-body">
    <div class="itinerary-head">
        <div>
            <h2 id="itinerary-title" class="itinerary-title" tabindex="-1">Your itinerary</h2>
            <p class="itinerary-subtitle">Day-by-day stops and walking directions.</p>
        </div>
        {{ if . }}
            {{ with .Plan }}
            {{ if or .ICSURL .KMLURL }}
            <div class="itinerary-actions">
                {{ if .ICSURL }}<a class="itinerary-action" href="{{ .ICSURL }}"><i class="bi bi-calendar-plus" aria-hidden="true"></i>Calendar export</a>{{ end }}
                {{ if .KMLURL }}<a class="itinerary-action" href="{{ .KMLURL }}"><i class="bi bi-map" aria-hidden="true"></i>Map export</a>{{ end }}
            </div>
            {{ end }}
            {{ end }}
        {{ end }}
    </div>

    {{ if . }}
        {{ if .Plan }}
            {{ with .Plan }}
            {{ if .Note }}<p class="itinerary-note">{{ .Note }}</p>{{ end }}

            {{ if .Days }}
            <div class="itinerary-days">
                {{ range $index, $day := .Days }}
                <details class="itinerary-day" {{ if eq $index 0 }}open{{ end }}>
                    <summary class="itinerary-day-summary">
                        <span class="itinerary-day-marker" aria-hidden="true">{{ $day.Number }}</span>
                        <span class="itinerary-day-title">
                            <span class="itinerary-day-number">Day {{ $day.Number }}</span>
                            <span class="itinerary-day-date"> · {{ $day.Date }}</span>
                        </span>
                        <span class="itinerary-status">
                            {{ if $day.Rainy }}
                            <span class="itinerary-chip itinerary-chip-rain"><i class="bi bi-cloud-rain" aria-hidden="true"></i>Rain{{ if $day.RainMM }} · {{ $day.RainMM }}{{ end }}</span>
                            {{ else if $day.RainMM }}
                            <span class="itinerary-chip"><i class="bi bi-cloud-drizzle" aria-hidden="true"></i>{{ $day.RainMM }} rain</span>
                            {{ else if $day.HasForecast }}
                            <span class="itinerary-chip"><i class="bi bi-cloud-sun" aria-hidden="true"></i>Rain not expected</span>
                            {{ end }}
                            {{ if not $day.HasForecast }}
                            <span class="itinerary-chip itinerary-chip-muted">No forecast data for this day</span>
                            {{ else if not $day.Certain }}
                            <span class="itinerary-chip itinerary-chip-muted">Forecast less certain</span>
                            {{ end }}
                        </span>
                        <span class="itinerary-walk"><i class="bi bi-signpost-split" aria-hidden="true"></i><span>{{ $day.WalkKM }} km walk</span></span>
                        <i class="bi bi-chevron-down itinerary-indicator" aria-hidden="true"></i>
                    </summary>
                    <div class="itinerary-day-body">
                        <div class="itinerary-stop-row">
                            <ol class="itinerary-stops">
                                {{ range $day.Stops }}
                                <li class="itinerary-stop">
                                    <figure class="itinerary-stop-media">
                                        <span class="itinerary-stop-fallback" aria-hidden="true"><i class="bi bi-image"></i></span>
                                        {{ if .Image }}
                                        <img class="itinerary-stop-photo" src="{{ .ImageURL 400 }}" alt="{{ .Name }}" width="400" loading="lazy" decoding="async"
                                             onerror="this.closest('.itinerary-stop-media').classList.add('itinerary-stop-media-failed'); this.hidden = true;">
                                        <figcaption class="itinerary-stop-credit">
                                            <a href="{{ .ImagePageURL }}" target="_blank" rel="noopener noreferrer">Photo: Wikimedia Commons</a>
                                        </figcaption>
                                        {{ end }}
                                        <span class="itinerary-stop-number" aria-hidden="true"></span>
                                    </figure>
                                    <div class="itinerary-stop-details">
                                        <span class="itinerary-stop-name">{{ .Name }}</span>
                                        {{ with .Kind }}<span class="itinerary-stop-kind">{{ if eq . "indoor" }}Indoors{{ else if eq . "outdoor" }}Outdoors{{ else if eq . "mixed" }}Indoor &amp; outdoor{{ else }}{{ . }}{{ end }}</span>{{ end }}
                                    </div>
                                </li>
                                {{ end }}
                            </ol>
                            {{ if $day.MapsURL }}
                            <a class="itinerary-directions" href="{{ $day.MapsURL }}" target="_blank" rel="noopener noreferrer">
                                <i class="bi bi-map" aria-hidden="true"></i>
                                Walking directions
                            </a>
                            {{ end }}
                        </div>
                    </div>
                </details>
                {{ end }}
            </div>
            {{ else }}
            <p class="itinerary-note">No itinerary stops were returned for this plan.</p>
            {{ end }}
            {{ end }}
        {{ else if .Error }}
            <p class="itinerary-note">No itinerary is currently shown. Resolve the form error and try again.</p>
        {{ else }}
            <p class="itinerary-empty">Generate a trip plan to see your day-by-day itinerary.</p>
        {{ end }}
    {{ else }}
        <p class="itinerary-unavailable">Trip planning is unavailable until a destination is loaded.</p>
    {{ end }}
</div>
