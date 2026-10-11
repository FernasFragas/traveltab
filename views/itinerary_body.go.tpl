<div class="itinerary-body">
    <div class="itinerary-head">
        <div>
            <h2 id="itinerary-title" class="itinerary-title" tabindex="-1">Your itinerary</h2>
            <p class="itinerary-subtitle">Day-by-day stops and walking directions.</p>
        </div>
        {{ if . }}
            {{ with .Plan }}
            {{ if or .ICSURL .KMLURL .Days }}
            <div class="itinerary-actions">
                {{ if .ICSURL }}<a class="itinerary-action" href="{{ .ICSURL }}"><i class="bi bi-calendar-plus" aria-hidden="true"></i>Export to calendar</a>{{ end }}
                {{ if .KMLURL }}<a class="itinerary-action" href="{{ .KMLURL }}"><i class="bi bi-map" aria-hidden="true"></i>Map export</a>{{ end }}
                {{ if .Days }}<button type="button" class="itinerary-action itinerary-action-primary" data-view-on-map><i class="bi bi-geo-alt" aria-hidden="true"></i>View on map</button>{{ end }}
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
                <details class="itinerary-day" data-plan-day="{{ $day.Number }}" {{ if eq $index 0 }}open{{ end }}>
                    <summary class="itinerary-day-summary">
                        <span class="itinerary-day-marker" aria-hidden="true">{{ $day.Number }}</span>
                        <span class="itinerary-day-title">
                            <span class="itinerary-day-number">Day {{ $day.Number }}</span>
                            <span class="itinerary-day-date"> · {{ $day.Date }}</span>
                        </span>
                        {{ if or $day.Temp $day.Condition }}
                        <span class="itinerary-weather">
                            {{ if $day.TempIcon }}<i class="bi {{ $day.TempIcon }}" aria-hidden="true"></i>{{ end }}
                            {{ if $day.Temp }}<span class="itinerary-day-temp" data-day-temp>{{ $day.Temp }}</span>{{ end }}
                            {{ with $day.Condition }}<span class="itinerary-day-condition">{{ . }}</span>{{ end }}
                        </span>
                        {{ end }}
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
                                {{ range $i, $stop := $day.Stops }}
                                <li class="itinerary-stop" data-plan-stop data-name="{{ $stop.Name }}" data-lat="{{ $stop.Lat }}" data-lon="{{ $stop.Lon }}">
                                    <figure class="itinerary-stop-media">
                                        <span class="itinerary-stop-fallback" aria-hidden="true"><i class="bi bi-image"></i></span>
                                        {{ if $stop.Image }}
                                        <img class="itinerary-stop-photo" src="{{ $stop.ImageURL 400 }}" alt="{{ $stop.Name }}" width="400" loading="lazy" decoding="async"
                                             onerror="this.closest('.itinerary-stop-media').classList.add('itinerary-stop-media-failed'); this.hidden = true;">
                                        <figcaption class="itinerary-stop-credit">
                                            <a href="{{ $stop.ImagePageURL }}" target="_blank" rel="noopener noreferrer">Photo: Wikimedia Commons</a>
                                        </figcaption>
                                        {{ end }}
                                        <span class="itinerary-stop-number" aria-hidden="true"></span>
                                    </figure>
                                    <div class="itinerary-stop-details">
                                        <span class="itinerary-stop-name">{{ $stop.Name }}</span>
                                        {{ with $stop.Description }}<span class="itinerary-stop-description">{{ . }}</span>{{ end }}
                                        <span class="itinerary-stop-meta">
                                            {{ with $stop.Kind }}<span class="itinerary-stop-kind">{{ if eq . "indoor" }}Indoors{{ else if eq . "outdoor" }}Outdoors{{ else if eq . "mixed" }}Indoor &amp; outdoor{{ else }}{{ . }}{{ end }}</span>{{ end }}
                                            {{ with index $day.Legs $i }}<span class="itinerary-stop-distance" data-stop-distance>{{ . }} from previous</span>{{ end }}
                                        </span>
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

            {{ if .AlsoSuggested }}
            <div class="itinerary-note itinerary-also-suggested">
                <strong>Also suggested</strong> (your days are full):
                {{ range $i, $p := .AlsoSuggested }}{{ if $i }}, {{ end }}<span data-also-suggested="{{ $p.ID }}">{{ $p.Name }}</span>{{ end }}
            </div>
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
