{{/* City guide card. Input: the presentation's guideView. Two states only: a reviewed intro with its
     Wikivoyage attribution, or a plain "no reviewed guide yet". All text is escaped; nothing here
     is markup from the guide file. */}}
<article class="guide-card" data-guide-state="{{ if .Available }}reviewed{{ else }}missing{{ end }}" aria-labelledby="guide-title">
    {{ if .Available }}
    <header class="guide-header">
        <p class="guide-eyebrow"><i class="bi bi-journal-text" aria-hidden="true"></i> City guide</p>
        <span class="guide-badge"><i class="bi bi-patch-check" aria-hidden="true"></i> Reviewed</span>
    </header>
    <h3 id="guide-title" class="guide-title">About {{ .City }}</h3>
    <div class="guide-body">
        {{ range .Paragraphs }}<p class="guide-paragraph">{{ . }}</p>{{ end }}
    </div>
    <footer class="guide-source">
        <p class="guide-attribution">Adapted from Wikivoyage: <a href="{{ .SourcePageURL }}" target="_blank" rel="noopener noreferrer">{{ .SourceTitle }}</a> (<a href="{{ .SourceRevisionURL }}" target="_blank" rel="noopener noreferrer">revision {{ .SourceRevision }}</a>), by <a href="{{ .SourceHistoryURL }}" target="_blank" rel="noopener noreferrer">Wikivoyage contributors</a>, licensed under <a href="{{ .LicenseURL }}" target="_blank" rel="noopener noreferrer">{{ .LicenseName }}</a>.</p>
        <p class="guide-review">Summarised and reworded. Checked against that revision on {{ .ReviewedOn }}.</p>
    </footer>
    {{ else }}
    <header class="guide-header">
        <p class="guide-eyebrow"><i class="bi bi-journal-text" aria-hidden="true"></i> City guide</p>
    </header>
    <h3 id="guide-title" class="guide-title">About {{ if .City }}{{ .City }}{{ else }}this destination{{ end }}</h3>
    <div class="guide-body">
        <p class="guide-paragraph guide-missing">We don't have a reviewed guide for {{ if .City }}{{ .City }}{{ else }}this destination{{ end }} yet. Guides appear here only after a person has checked them against their source.</p>
    </div>
    {{ with .SearchURL }}
    <p class="guide-actions"><a class="guide-link" href="{{ . }}" target="_blank" rel="noopener noreferrer">Search Wikivoyage for {{ $.City }} <i class="bi bi-arrow-up-right" aria-hidden="true"></i></a></p>
    {{ end }}
    {{ end }}
</article>
