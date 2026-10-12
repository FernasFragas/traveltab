{{ if .WriterMap -}}
<div class="map-card writers-map-card" data-writers-map data-map-destination="{{ .WriterMap.Destination }}" data-map-lat="{{ .WriterMap.Lat }}" data-map-lon="{{ .WriterMap.Lon }}" data-map-url="{{ .WriterMap.URL }}"{{ with .WriterMap.TileURL }} data-map-tile-url="{{ . }}"{{ end }}{{ with .WriterMap.Attribution }} data-map-attribution="{{ . }}"{{ end }}{{ with .WriterMap.StyleURL }} data-map-style-url="{{ . }}"{{ end }}{{ if .WriterMap.TerrainOff }} data-map-terrain-url=""{{ else }}{{ with .WriterMap.TerrainURL }} data-map-terrain-url="{{ . }}"{{ end }}{{ end }}{{ with .WriterMap.Engine }} data-map-engine="{{ . }}"{{ end }}>
    <div class="writers-map-header">
        <h3>Writers' Map <button class="writers-map-info" type="button" aria-label="About map layers" aria-describedby="writers-map-tooltip" aria-expanded="false"><i class="bi bi-info-circle" aria-hidden="true"></i></button></h3>
        <span class="writers-map-tooltip" id="writers-map-tooltip" role="tooltip">The map has four layers: notable places, writers' pins, your day plan, and writers' itineraries.</span>
        <div data-map-slot="layers" class="writers-map-layers" aria-label="Map layers">
            <span class="writers-map-layer writers-map-layer-notable">Notable places</span>
            <span class="writers-map-layer writers-map-layer-writers">Writers' pins</span>
            <span class="writers-map-layer writers-map-layer-plan">Your day plan</span>
            <span class="writers-map-layer writers-map-layer-itineraries">Writers' itineraries</span>
        </div>
    </div>
    <div data-map-slot="status" role="status" hidden></div>
    <div class="writers-map-body">
        <div data-map-canvas></div>
        <div data-map-slot="panel"><p>Pick a place on the map</p></div>
    </div>
    <div data-map-slot="saved"></div>
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
