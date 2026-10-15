<article class="map-place-panel" data-place-qid="{{ .Place.QID }}" data-place-area="{{ .Place.Area }}">
    {{ if .Place.PhotoURL }}<div class="map-panel-media" data-carousel><figure class="map-panel-photo"><img src="{{ .Place.PhotoURL }}" alt="{{ .Name }}" loading="lazy" onerror="this.closest('.map-panel-photo').hidden = true">{{ if .Place.PhotoCredit }}<figcaption>{{ .Place.PhotoCredit }}</figcaption>{{ end }}</figure></div>{{ end }}
    <div class="map-panel-title"><h3>{{ .Name }}</h3><button type="button" class="map-panel-bookmark" data-save-qid="{{ .Place.QID }}" aria-label="Save to visit"><i class="bi bi-bookmark" aria-hidden="true"></i></button></div>
    {{ if or .Place.Kind .Place.Area .City }}<p class="map-panel-meta">{{ if .Place.Kind }}<i class="bi bi-geo-alt" data-kind-icon="{{ .Place.Kind }}" aria-hidden="true"></i> {{ .KindLabel }}{{ end }}{{ if or .Place.Area .City }} <span><i class="bi bi-geo-alt" aria-hidden="true"></i> {{ .Place.Area }}{{ if and .Place.Area .City }}, {{ end }}{{ .City }}</span>{{ end }}</p>{{ end }}
    {{ range .WriterSources }}<span data-writer-source-host="{{ .Host }}" data-writer-source-name="{{ .Name }}" hidden></span>{{ end }}
    {{ if .Summary }}<p class="map-panel-summary">{{ .Summary.EnglishText }}</p>{{ else if .Place.Description }}<p class="map-panel-description">{{ .Place.Description }}</p>{{ end }}
    <div class="map-panel-actions">
        <button type="button" class="map-panel-add" data-add-qid="{{ .Place.QID }}"><i class="bi bi-plus-circle" aria-hidden="true"></i> Add to your plan</button>
        <button type="button" class="map-panel-save" data-save-qid="{{ .Place.QID }}" aria-label="Save to visit"><i class="bi bi-bookmark" aria-hidden="true"></i> <span data-save-label>Save</span></button>
        <button type="button" class="map-panel-share" data-map-share><i class="bi bi-share" aria-hidden="true"></i> Share</button>
    </div>
    <p class="map-panel-feedback" role="status" aria-live="polite"></p>
    <div class="map-panel-links">
        {{ if .Place.WikipediaURL }}<a class="map-panel-wikipedia" href="{{ .Place.WikipediaURL }}" target="_blank" rel="noopener noreferrer"><span class="map-panel-link-icon" aria-hidden="true">W</span><span><strong>View on Wikipedia</strong><small>History, photos and more</small></span><i class="bi bi-box-arrow-up-right" aria-hidden="true"></i></a>{{ end }}
        <a class="map-panel-directions" href="https://www.google.com/maps/dir/?api=1&amp;destination={{ .Place.Lat }},{{ .Place.Lon }}" target="_blank" rel="noopener noreferrer"><span class="map-panel-link-icon" aria-hidden="true"><i class="bi bi-geo-alt-fill"></i></span><span><strong>Get directions</strong><small>Open in Google Maps</small></span><i class="bi bi-box-arrow-up-right" aria-hidden="true"></i></a>
    </div>
    <section class="map-panel-facts" aria-label="Place facts" hidden>
        {{ if .Place.OpeningHours }}<div class="map-panel-hours" data-opening-hours="{{ .Place.OpeningHours }}"><h4><i class="bi bi-clock" aria-hidden="true"></i> Hours</h4><strong>{{ .Place.OpeningHours }}</strong></div>{{ end }}
        <div class="map-panel-best-time" data-best-time-kind="{{ .Place.Kind }}" hidden><h4><i class="bi bi-sun" aria-hidden="true"></i> Best time to visit</h4><strong></strong><small></small></div>
    </section>
    <section class="map-panel-writers">
        <header><h4><i class="bi bi-pencil-square" aria-hidden="true"></i> From our writers</h4>{{ if .Mentions }}<button type="button" data-writers-toggle aria-expanded="false" aria-controls="writer-posts-{{ .Place.QID }}">See all ›</button>{{ end }}</header>
        {{ if .Mentions }}{{ with index .Mentions 0 }}<blockquote><p>“{{ .PostTitle }}”</p><cite>— <a href="{{ .PostURL }}" target="_blank" rel="noopener noreferrer">{{ .BlogName }} <i class="bi bi-box-arrow-up-right" aria-hidden="true"></i></a></cite></blockquote>{{ end }}
        <ul id="writer-posts-{{ .Place.QID }}" hidden>
            {{ range .Mentions }}<li><a href="{{ .PostURL }}" title="{{ .PostTitle }}" data-writer-host="{{ .WriterHost }}" data-blog-name="{{ .BlogName }}" data-language="{{ .Language }}" target="_blank" rel="noopener noreferrer"><span class="map-panel-post-title">{{ .PostTitle }}</span> <i class="bi bi-box-arrow-up-right" aria-hidden="true"></i></a><span class="map-panel-blog">{{ .BlogName }}</span> <span class="map-panel-language">{{ .Language }}</span></li>{{ end }}
        </ul>{{ else }}<p>No writer mentions yet</p>{{ end }}
    </section>
</article>
