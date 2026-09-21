<section class="stay-card">
    <div class="stay-card-head">
        <h4 class="stay-card-title"><i class="bi bi-house-door"></i> Where to stay</h4>
        {{ if and . .BookingURL }}
        <a class="btn btn-primary stay-prices" href="{{ .BookingURL }}" target="_blank" rel="noopener noreferrer">Check prices</a>
        {{ end }}
    </div>

    {{ if not . }}
    <p class="stay-note">Generate a trip plan to find places to stay.</p>
    {{ else if .StaysNote }}
    <p class="stay-note">{{ .StaysNote }}</p>
    {{ else }}
    <ul class="stay-list">
        {{ range .Stays }}
        <li class="stay">
            <i class="stay-icon stay-icon-{{ .Kind }} bi {{ if eq .Kind "hotel" }}bi-building{{ else if eq .Kind "hostel" }}bi-people{{ else }}bi-house-heart{{ end }}" aria-hidden="true"></i>
            <span class="stay-name">{{ .Name }}</span>
            {{ if gt .Stars 0 }}<span class="stay-stars" aria-label="{{ .Stars }} stars">★ {{ .Stars }}</span>{{ end }}
            {{ if .Website }}<a class="stay-website" href="{{ .Website }}" target="_blank" rel="noopener noreferrer">Website</a>{{ end }}
        </li>
        {{ end }}
    </ul>
    {{ end }}
</section>
