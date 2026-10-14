<article class="map-place-panel" data-place-qid="{{ .Place.QID }}" data-place-area="{{ .Place.Area }}">
    <div class="map-panel-media">
        {{ if .Place.PhotoURL }}<figure class="map-panel-photo"><img src="{{ .Place.PhotoURL }}" alt="{{ .Name }}" loading="lazy" onerror="this.closest('.map-panel-photo').hidden = true">{{ if .Place.PhotoCredit }}<figcaption>{{ .Place.PhotoCredit }}</figcaption>{{ end }}</figure>{{ end }}
        <button type="button" class="map-panel-close" aria-label="Close place details"><i class="bi bi-x-lg" aria-hidden="true"></i></button>
    </div>
    <div class="map-panel-title"><h3>{{ .Name }}</h3><button type="button" class="map-panel-bookmark" data-save-qid="{{ .Place.QID }}" aria-label="Save to visit"><i class="bi bi-bookmark" aria-hidden="true"></i></button></div>
    {{ if or .Place.Kind .Place.Area }}<p class="map-panel-meta">{{ .Place.Kind }}{{ if and .Place.Kind .Place.Area }} · {{ end }}{{ .Place.Area }}</p>{{ end }}
    {{ range .WriterSources }}<span data-writer-source-host="{{ .Host }}" data-writer-source-name="{{ .Name }}" hidden></span>{{ end }}
    <div class="map-panel-writer-card"><i class="bi bi-feather" aria-hidden="true"></i><div class="map-panel-mentioned">{{ if .Mentions }}<span class="map-panel-monograms" aria-hidden="true"></span><span>Mentioned by {{ .WriterCount }} writer{{ if ne .WriterCount 1 }}s{{ end }}</span>{{ else }}<span>No writer mentions yet</span>{{ end }}</div>{{ if .Mentions }}<button type="button" data-writer-posts aria-label="Go to writers' posts">›</button>{{ end }}</div>
    {{ if .Summary }}<p class="map-panel-summary">{{ .Summary.EnglishText }}</p>{{ else if .Place.Description }}<p class="map-panel-description">{{ .Place.Description }}</p>{{ end }}
    <div class="map-panel-actions">
        <button type="button" class="map-panel-add" data-add-qid="{{ .Place.QID }}">+ Add to plan</button>
        <button type="button" class="map-panel-save" data-save-qid="{{ .Place.QID }}">Save to visit</button>
        <button type="button" class="map-panel-share" data-map-share>Share</button>
    </div>
    <p class="map-panel-feedback" role="status" aria-live="polite"></p>
    <div class="map-panel-links">
        {{ if .Place.WikipediaURL }}<a class="map-panel-wikipedia" href="{{ .Place.WikipediaURL }}" target="_blank" rel="noopener noreferrer"><strong>Wikipedia</strong><span>View on Wikipedia</span><i class="bi bi-box-arrow-up-right" aria-hidden="true"></i></a>{{ end }}
        <a class="map-panel-directions" href="https://www.openstreetmap.org/directions?route=%3B{{ .Place.Lat }}%2C{{ .Place.Lon }}" target="_blank" rel="noopener noreferrer"><strong>Directions</strong><span>Open in OpenStreetMap</span><i class="bi bi-box-arrow-up-right" aria-hidden="true"></i></a>
    </div>
    {{ if .Place.OpeningHours }}<details class="map-panel-hours" open><summary><i class="bi bi-clock" aria-hidden="true"></i> Hours <span aria-hidden="true">▾</span></summary><p>{{ .Place.OpeningHours }}</p></details>{{ end }}
    {{ if .Mentions }}
    <section class="map-panel-writers">
        <h4>Writers' posts ({{ len .Mentions }}) ›</h4>
        <ul>
            {{ range .Mentions }}<li><a href="{{ .PostURL }}" title="{{ .PostTitle }}" data-writer-host="{{ .WriterHost }}" data-blog-name="{{ .BlogName }}" data-language="{{ .Language }}" target="_blank" rel="noopener noreferrer"><span class="map-panel-post-tile" aria-hidden="true"><i class="bi bi-feather"></i></span><span class="map-panel-post-title">{{ .PostTitle }}</span> <i class="bi bi-box-arrow-up-right" aria-hidden="true"></i></a><span class="map-panel-blog">{{ .BlogName }}</span> <span class="map-panel-language">{{ .Language }}</span></li>
            {{ end }}
        </ul>
    </section>
    {{ end }}
</article>
