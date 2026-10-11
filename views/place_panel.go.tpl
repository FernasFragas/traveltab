<article class="map-place-panel">
    <div class="map-panel-media">
        {{ if .Place.PhotoURL }}<figure class="map-panel-photo"><img src="{{ .Place.PhotoURL }}" alt="{{ .Name }}">{{ if .Place.PhotoCredit }}<figcaption>{{ .Place.PhotoCredit }}</figcaption>{{ end }}</figure>{{ end }}
        <button type="button" class="map-panel-close" aria-label="Close place details"><i class="bi bi-x-lg" aria-hidden="true"></i></button>
    </div>
    <h3>{{ .Name }}</h3>
    {{ if or .Place.Kind .Place.Area }}<p class="map-panel-meta">{{ .Place.Kind }}{{ if and .Place.Kind .Place.Area }} · {{ end }}{{ .Place.Area }}</p>{{ end }}
    {{ if .Mentions }}
    <div class="map-panel-mentioned">
        <span class="map-panel-monograms" aria-hidden="true"></span>
        <span>Mentioned by {{ .WriterCount }} writer{{ if ne .WriterCount 1 }}s{{ end }}</span>
    </div>
    {{ else }}
    <p class="map-panel-mentioned">No writer mentions here yet</p>
    {{ end }}
    {{ if .Summary }}<section class="map-panel-summary"><h4>AI summary</h4><p>{{ .Summary.EnglishText }}</p></section>{{ end }}
    {{ if .Place.Description }}<p class="map-panel-description">{{ .Place.Description }}</p>{{ end }}
    {{ if .Place.OpeningHours }}<p class="map-panel-hours"><i class="bi bi-clock" aria-hidden="true"></i> {{ .Place.OpeningHours }}</p>{{ end }}
    {{ if .Place.WikipediaURL }}<a class="map-panel-wikipedia" href="{{ .Place.WikipediaURL }}" target="_blank" rel="noopener noreferrer">View on Wikipedia <i class="bi bi-box-arrow-up-right" aria-hidden="true"></i></a>{{ end }}
    <button type="button" class="map-panel-save" data-save-qid="{{ .Place.QID }}">Save to plan</button>
    {{ if .Mentions }}
    <details class="map-panel-writers">
        <summary>View writers' posts ({{ len .Mentions }})</summary>
        <ul>
            {{ range .Mentions }}<li><a href="{{ .PostURL }}" title="{{ .PostTitle }}" data-writer-host="{{ .WriterHost }}" data-blog-name="{{ .BlogName }}" data-language="{{ .Language }}" target="_blank" rel="noopener noreferrer"><span class="map-panel-post-title">{{ .PostTitle }}</span> <i class="bi bi-box-arrow-up-right" aria-hidden="true"></i></a><span class="map-panel-blog">{{ .BlogName }}</span> <span class="map-panel-language">{{ .Language }}</span></li>
            {{ end }}
        </ul>
    </details>
    {{ end }}
</article>
