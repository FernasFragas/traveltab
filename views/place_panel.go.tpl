<article class="map-place-panel">
    <h3>{{ .Name }}</h3>
    {{ if or .Place.Kind .Place.Area }}<p>{{ .Place.Kind }}{{ if and .Place.Kind .Place.Area }} · {{ end }}{{ .Place.Area }}</p>{{ end }}
    {{ if .Place.Description }}<p>{{ .Place.Description }}</p>{{ end }}
    {{ if .Place.PhotoURL }}<figure><img src="{{ .Place.PhotoURL }}" alt="{{ .Name }}">{{ if .Place.PhotoCredit }}<figcaption>{{ .Place.PhotoCredit }}</figcaption>{{ end }}</figure>{{ end }}
    <p>Mentioned by {{ .WriterCount }} writers</p>
    {{ if .Summary }}<section><h4>AI summary</h4><p>{{ .Summary.EnglishText }}</p></section>{{ end }}
    {{ if .Place.OpeningHours }}<p>{{ .Place.OpeningHours }}</p>{{ end }}
    {{ if .Place.WikipediaURL }}<p><a href="{{ .Place.WikipediaURL }}" target="_blank" rel="noopener noreferrer">Wikipedia</a></p>{{ end }}
    {{ if .Mentions }}<ul>{{ range .Mentions }}<li><a href="{{ .PostURL }}" title="{{ .PostTitle }}" data-writer-host="{{ .WriterHost }}" data-blog-name="{{ .BlogName }}" data-language="{{ .Language }}" target="_blank" rel="noopener noreferrer">{{ .PostTitle }}</a><span>{{ .BlogName }} · {{ .Language }}</span></li>{{ end }}</ul>{{ end }}
    <button type="button" data-save-qid="{{ .Place.QID }}">Save to plan</button>
</article>
