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
