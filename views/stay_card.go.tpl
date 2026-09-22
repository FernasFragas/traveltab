<div class="stay-body">
    <div class="stay-heading">
        <div>
            <h2 id="stays-title" class="tt-section-title">Where to stay</h2>
            <p class="tt-section-subtitle">Named places to stay close to your itinerary.</p>
        </div>
        {{ if and . .BookingURL }}
        <a class="tt-button stay-prices" href="{{ .BookingURL }}" target="_blank" rel="noopener noreferrer">
            Check prices
            <i class="bi bi-arrow-up-right" aria-hidden="true"></i>
        </a>
        {{ end }}
    </div>

    {{ if not . }}
    <p class="stay-note">Generate a trip plan to find places to stay.</p>
    {{ else }}
    {{ if .StaysNote }}
    <p class="stay-note">{{ .StaysNote }}</p>
    {{ else if not .StayCards }}
    <p class="stay-note">No matching places to stay were returned. Check prices to search for accommodation.</p>
    {{ else }}
    <ul class="stay-grid">
        {{ range .StayCards }}
        <li class="stay-item">
            <div class="stay-media">
                <span class="stay-placeholder" aria-hidden="true"><i class="bi bi-building"></i></span>
                {{ with .Image }}
                <img class="stay-image" src="{{ .URL }}" alt="{{ .Alt }}" width="{{ .Width }}" height="{{ .Height }}" loading="lazy"
                     onerror="var media = this.closest('.stay-media'); if (media) { media.classList.add('stay-media-failed'); } this.hidden = true;">
                {{ if .Credit }}
                {{ if .CreditURL }}
                <a class="stay-credit" href="{{ .CreditURL }}" target="_blank" rel="noopener noreferrer">Photo: {{ .Credit }}</a>
                {{ else }}
                <span class="stay-credit">Photo: {{ .Credit }}</span>
                {{ end }}
                {{ end }}
                {{ end }}
            </div>
            <div class="stay-details">
                <h3 class="stay-name">{{ .Name }}</h3>
                {{ if or .Kind (gt .Stars 0) }}
                <p class="stay-meta">
                    {{ if .Kind }}
                    <span class="stay-kind">
                        {{ if eq .Kind "hotel" }}Hotel
                        {{ else if eq .Kind "hostel" }}Hostel
                        {{ else if eq .Kind "guest_house" }}Guest house
                        {{ else }}{{ .Kind }}{{ end }}
                    </span>
                    {{ end }}
                    {{ if gt .Stars 0 }}
                    <span class="stay-stars">{{ .Stars }}-star property</span>
                    {{ end }}
                </p>
                {{ end }}
                {{ if .Website }}
                <a class="stay-website" href="{{ .Website }}" target="_blank" rel="noopener noreferrer">
                    Website
                    <i class="bi bi-arrow-up-right" aria-hidden="true"></i>
                </a>
                {{ end }}
            </div>
        </li>
        {{ end }}
    </ul>
    {{ end }}
    {{ end }}
</div>
