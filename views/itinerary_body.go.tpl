{{/* Compatibility extraction for tasks 01–03; task 04 owns the accordion redesign. */}}
<h2 class="tt-section-title">Your itinerary</h2>
{{ if and . .Plan }}
    {{ with .Plan }}
    {{ if .Note }}<p class="trip-note">{{ .Note }}</p>{{ end }}

    {{ if or .ICSURL .KMLURL }}
    <div class="trip-export">
        {{ if .ICSURL }}<a class="btn btn-outline-secondary trip-export-link" href="{{ .ICSURL }}"><i class="bi bi-calendar-plus"></i> Add to calendar</a>{{ end }}
        {{ if .KMLURL }}<a class="btn btn-outline-secondary trip-export-link" href="{{ .KMLURL }}"><i class="bi bi-map"></i> Download for Google My Maps</a>{{ end }}
    </div>
    {{ end }}

    <ol class="trip-days">
        {{ range .Days }}
        <li class="trip-day{{ if .Rainy }} trip-day-rainy{{ end }}">
            <div class="trip-day-head">
                <h4 class="trip-day-title">Day {{ .Number }} · {{ .Date }}</h4>
                <div class="trip-badges">
                    {{ if .Rainy }}<span class="trip-badge trip-badge-rain"><i class="bi bi-cloud-rain"></i> Rainy day</span>{{ end }}
                    {{ if .RainMM }}<span class="trip-badge">{{ .RainMM }}</span>{{ end }}
                    {{ if not .HasForecast }}<span class="trip-badge trip-badge-soft">No forecast data for this day</span>
                    {{ else if not .Certain }}<span class="trip-badge trip-badge-soft">Forecast is less certain this far ahead</span>{{ end }}
                    <span class="trip-badge"><i class="bi bi-signpost-split"></i> {{ .WalkKM }} km walk</span>
                </div>
                {{ if .MapsURL }}
                <a class="trip-day-maps" href="{{ .MapsURL }}" target="_blank" rel="noopener noreferrer">
                    <i class="bi bi-map"></i> Open Day {{ .Number }} in Google Maps
                </a>
                {{ end }}
            </div>

            <ul class="trip-stops">
                {{ range .Stops }}
                <li class="trip-stop">
                    {{ if .Image }}
                    <figure class="trip-stop-figure">
                        <img class="trip-stop-photo" src="{{ .ImageURL 400 }}" alt="{{ .Name }}" loading="lazy">
                        <figcaption class="trip-stop-credit">
                            <a href="{{ .ImagePageURL }}" target="_blank" rel="noopener noreferrer">Photo: Wikimedia Commons</a>
                        </figcaption>
                    </figure>
                    {{ end }}
                    <span class="trip-stop-name">{{ .Name }}</span>
                    {{ if eq .Kind "indoor" }}<span class="trip-stop-kind">Indoors</span>
                    {{ else if eq .Kind "outdoor" }}<span class="trip-stop-kind">Outdoors</span>
                    {{ else if eq .Kind "mixed" }}<span class="trip-stop-kind">Indoor &amp; outdoor</span>{{ end }}
                </li>
                {{ end }}
            </ul>
        </li>
        {{ end }}
    </ol>

    {{ end }}

{{ else }}
<p class="tt-section-subtitle">Generate a trip plan to see your day-by-day itinerary.</p>
{{ end }}
