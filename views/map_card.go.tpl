{{ if .WriterMap -}}
<div class="map-card writers-map-card" data-writers-map data-map-destination="{{ .WriterMap.Destination }}" data-map-lat="{{ .WriterMap.Lat }}" data-map-lon="{{ .WriterMap.Lon }}" data-map-url="{{ .WriterMap.URL }}"{{ with .WriterMap.TileURL }} data-map-tile-url="{{ . }}"{{ end }}{{ with .WriterMap.Attribution }} data-map-attribution="{{ . }}"{{ end }}{{ with .WriterMap.StyleURL }} data-map-style-url="{{ . }}"{{ end }}{{ if .WriterMap.TerrainOff }} data-map-terrain-url=""{{ else }}{{ with .WriterMap.TerrainURL }} data-map-terrain-url="{{ . }}"{{ end }}{{ end }}{{ with .WriterMap.Engine }} data-map-engine="{{ . }}"{{ end }}>
    <h2 class="tt-visually-hidden">Map places</h2>
    <div class="writers-map-header">
        <h3>Writers' Map <button class="writers-map-info" type="button" aria-label="About map layers" aria-describedby="writers-map-tooltip" aria-expanded="false"><i class="bi bi-info-circle" aria-hidden="true"></i></button></h3>
        <span class="writers-map-tooltip" id="writers-map-tooltip" role="tooltip">The map has four layers: notable places, writers' pins, your day plan, and writers' itineraries.</span>
        <div data-map-slot="layers" class="writers-map-layers" aria-label="Map layers">
            <span class="writers-map-layer writers-map-layer-notable">Notable places</span>
            <span class="writers-map-layer writers-map-layer-writers">Writers' pins</span>
            <span class="writers-map-layer writers-map-layer-itineraries">Writers' itineraries</span>
            <span class="writers-map-layer writers-map-layer-plan">Your plan</span>
        </div>
    </div>
    <div data-map-slot="status" role="status" hidden></div>
    <div class="writers-map-body">
        <div data-map-canvas></div>
        <div data-map-slot="panel">
            <div class="map-default-panel">
                {{ with .Presentation.Hero }}<figure class="map-default-photo"><img src="{{ .URL }}" alt="{{ .Alt }}" width="{{ .Width }}" height="{{ .Height }}" onerror="this.closest('.map-default-photo').hidden = true"><figcaption class="map-default-photo-credit"><a href="{{ .CreditURL }}" target="_blank" rel="noopener noreferrer">Photo: {{ .Credit }}</a>{{ if .LicenseURL }} · <a href="{{ .LicenseURL }}" target="_blank" rel="noopener noreferrer">{{ .License }}</a>{{ else }} · {{ .License }}{{ end }}</figcaption></figure>{{ end }}
                <h2>{{ .GeneralInfo.City }}</h2>
                {{ with .Presentation.Location }}<p class="map-default-location">{{ . }}</p>{{ end }}
                {{ if .GeneralInfo }}<div class="map-default-weather"><i class="bi bi-thermometer-half" aria-hidden="true"></i><strong>{{ .GeneralInfo.Weather.Temperature }}°C</strong><span>{{ .GeneralInfo.Weather.Condition }}</span></div>{{ end }}
                <section class="map-default-guide">
                    <h3>About {{ .GeneralInfo.City }}</h3>
                    {{ with .Presentation.Guide }}
                        {{ if .Available }}
                            {{ range $i, $paragraph := .Paragraphs }}{{ if eq $i 0 }}<p>{{ $paragraph }}</p>{{ end }}{{ end }}
                            <p class="map-default-credit">Adapted from <a href="{{ .SourcePageURL }}" target="_blank" rel="noopener noreferrer">Wikivoyage</a> ({{ .LicenseName }}).</p>
                        {{ else }}<p>We don't have a reviewed guide for this destination yet.</p>{{ end }}
                    {{ end }}
                </section>
                <a class="map-default-plan" href="#itinerary">Plan your trip <i class="bi bi-arrow-right" aria-hidden="true"></i></a>
            </div>
        </div>
    </div>
    <div class="writers-map-toast" role="status" aria-live="polite" hidden></div>
    <div hx-get="{{ .WriterMap.URL }}" hx-trigger="load" hx-swap="none" hidden></div>
    <div class="map-footer">
        {{ with .Presentation.MapURL }}<a class="map-action" href="{{ . }}" target="_blank" rel="noopener noreferrer">{{ with $.Presentation.Hero }}<img class="map-action-photo" src="{{ .URL }}" alt="" loading="lazy">{{ else }}<i class="bi bi-map" aria-hidden="true"></i>{{ end }}<span><strong>View full map</strong><small>Open {{ $.GeneralInfo.City }} on a larger map</small></span><i class="bi bi-box-arrow-up-right" aria-hidden="true"></i></a>{{ end }}
    </div>
</div>
{{ else -}}
<div class="map-card">
    {{ if .GeneralInfo.EmbedURL }}
    <iframe src="{{ .GeneralInfo.EmbedURL }}" class="map-embed" title="Live Waze map of {{ .GeneralInfo.City }}" loading="lazy" allowfullscreen></iframe>
    {{ else }}
    <p class="map-unavailable">The live map is unavailable. Open the larger map to explore this destination.</p>
    {{ end }}
    <div class="map-footer">
        <a class="map-attribution" href="https://www.waze.com/" target="_blank" rel="noopener noreferrer">Map by Waze</a>
        {{ with .Presentation.MapURL }}<a class="map-action" href="{{ . }}" target="_blank" rel="noopener noreferrer">{{ with $.Presentation.Hero }}<img class="map-action-photo" src="{{ .URL }}" alt="" loading="lazy">{{ else }}<i class="bi bi-map" aria-hidden="true"></i>{{ end }}<span><strong>View full map</strong><small>Open {{ $.GeneralInfo.City }} on a larger map</small></span><i class="bi bi-box-arrow-up-right" aria-hidden="true"></i></a>{{ end }}
    </div>
</div>
{{- end }}
