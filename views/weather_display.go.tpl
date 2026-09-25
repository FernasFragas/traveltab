{{ if .GeneralInfo }}
<section class="destination-banner" aria-label="Destination and current weather">
    <figure class="destination-photo" data-photo-state="{{ if .Presentation.Hero }}ready{{ else }}unavailable{{ end }}">
        {{ with .Presentation.Hero }}
        <img class="destination-image" src="{{ .URL }}" alt="{{ .Alt }}" width="{{ .Width }}" height="{{ .Height }}" fetchpriority="high" onload="this.closest('.destination-photo').dataset.photoState = 'ready'" onerror="this.closest('.destination-photo').dataset.photoState = 'failed'">
        <figcaption class="destination-photo-credit"><a href="{{ .CreditURL }}" target="_blank" rel="noopener noreferrer">Photo: {{ .Credit }}</a>{{ if .LicenseURL }} · <a href="{{ .LicenseURL }}" target="_blank" rel="noopener noreferrer">{{ .License }}</a>{{ else }} · {{ .License }}{{ end }}</figcaption>
        {{ end }}
        {{ with .Presentation.PhotoCaption }}<p class="destination-photo-caption">{{ . }}</p>{{ end }}
        <p class="destination-photo-fallback"><i class="bi bi-image" aria-hidden="true"></i> Photo unavailable</p>
    </figure>
    <div class="destination-information">
        <h1 class="destination-title">{{ .GeneralInfo.City }}</h1>
        {{ with .Presentation.Tagline }}<p class="destination-tagline">{{ . }}</p>{{ end }}
        {{ template "weather_card" . }}
        {{ with .Presentation.CountryLabel }}<p class="destination-country">{{ . }}</p>{{ end }}
    </div>
</section>
{{ end }}
