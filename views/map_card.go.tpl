{{ if .WriterMap -}}
<div class="map-card" data-writers-map data-map-destination="{{ .WriterMap.Destination }}" data-map-lat="{{ .WriterMap.Lat }}" data-map-lon="{{ .WriterMap.Lon }}" data-map-url="{{ .WriterMap.URL }}"{{ with .WriterMap.TileURL }} data-map-tile-url="{{ . }}"{{ end }}{{ with .WriterMap.Attribution }} data-map-attribution="{{ . }}"{{ end }}>
    <h3>Writers' Map</h3>
    <div data-map-slot="layers"></div>
    <div data-map-slot="status" role="status" hidden></div>
    <div data-map-canvas></div>
    <div data-map-slot="panel"><p>Pick a place on the map</p></div>
    <div data-map-slot="saved"></div>
    <div data-map-slot="list"></div>
    <div hx-get="{{ .WriterMap.URL }}" hx-trigger="load" hx-swap="none" hidden></div>
    <div class="map-footer">
        {{ with .Presentation.MapURL }}<a class="map-action" href="{{ . }}" target="_blank" rel="noopener noreferrer">View larger map <i class="bi bi-arrow-up-right" aria-hidden="true"></i></a>{{ end }}
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
        {{ with .Presentation.MapURL }}<a class="map-action" href="{{ . }}" target="_blank" rel="noopener noreferrer">View larger map <i class="bi bi-arrow-up-right" aria-hidden="true"></i></a>{{ end }}
    </div>
</div>
{{- end }}
