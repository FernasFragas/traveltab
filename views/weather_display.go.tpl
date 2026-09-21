{{ if .GeneralInfo }}
<section class="destination-banner" aria-label="Destination and current weather">
    <figure class="destination-photo">
        {{ with .Presentation.Hero }}
        <img class="destination-image" src="{{ .URL }}" alt="{{ .Alt }}" width="{{ .Width }}" height="{{ .Height }}" fetchpriority="high" onerror="this.hidden = true">
        <figcaption class="destination-photo-credit"><a href="{{ .CreditURL }}" target="_blank" rel="noopener noreferrer">Photo: {{ .Credit }}</a>{{ if .LicenseURL }} · <a href="{{ .LicenseURL }}" target="_blank" rel="noopener noreferrer">{{ .License }}</a>{{ end }}</figcaption>
        {{ end }}
        {{ with .Presentation.PhotoCaption }}<p class="destination-photo-caption">{{ . }}</p>{{ end }}
    </figure>
    <div class="destination-information">
        <h1 class="destination-title">{{ .GeneralInfo.City }}</h1>
        {{ with .Presentation.Tagline }}<p class="destination-tagline">{{ . }}</p>{{ end }}
        {{ template "weather_card" . }}
        {{ with .Presentation.CountryLabel }}<p class="destination-country">{{ . }}</p>{{ end }}
    </div>
</section>
{{ end }}
