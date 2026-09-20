<div id="trip-card" class="trip-card">
    <div class="trip-card-head">
        <h3 class="trip-card-title"><i class="bi bi-calendar2-week"></i> Plan my trip</h3>
        <p class="trip-card-subtitle">Famous places day by day, with museums saved for the rain.</p>
    </div>

    <form class="trip-form"
          hx-get="/plan"
          hx-target="#trip-card"
          hx-swap="outerHTML"
          hx-indicator="#trip-spinner">
        <input type="hidden" name="city" value="{{ .City }}">
        <input type="hidden" name="country" value="{{ .Country }}">
        <input type="hidden" name="lat" value="{{ .Lat }}">
        <input type="hidden" name="lon" value="{{ .Lon }}">

        <label class="trip-field">
            <span class="trip-field-label">Starting</span>
            <input type="date" name="start" value="{{ .Start }}" min="{{ .MinStart }}" class="form-control" required>
        </label>

        <label class="trip-field">
            <span class="trip-field-label">Days</span>
            <select name="days" class="form-select">
                {{- $selected := .Days }}
                {{- range .DayOptions }}
                <option value="{{ . }}"{{ if eq . $selected }} selected{{ end }}>{{ . }}</option>
                {{- end }}
            </select>
        </label>

        <button type="submit" class="btn btn-primary trip-submit">Plan my trip</button>
        <span id="trip-spinner" class="trip-spinner"><i class="fas fa-spinner fa-spin"></i></span>
    </form>

    {{ if .Error }}
    <p class="trip-error" role="alert"><i class="bi bi-exclamation-circle"></i> {{ .Error }}</p>
    {{ end }}

    {{ with .Plan }}
    {{ if .Note }}<p class="trip-note">{{ .Note }}</p>{{ end }}

    <ol class="trip-days">
        {{ range .Days }}
        <li class="trip-day{{ if .Rainy }} trip-day-rainy{{ end }}">
            <div class="trip-day-head">
                <h4 class="trip-day-title">Day {{ .Number }} · {{ .Date }}</h4>
                <div class="trip-badges">
                    {{ if .Rainy }}<span class="trip-badge trip-badge-rain"><i class="bi bi-cloud-rain"></i> Rainy day</span>{{ end }}
                    {{ if .RainMM }}<span class="trip-badge">{{ .RainMM }}</span>{{ end }}
                    {{ if not .HasForecast }}<span class="trip-badge trip-badge-soft">No forecast data for this day</span>
                    {{ else if not .Certain }}<span class="trip-badge trip-badge-soft">Forecast is less certain this far ahead</span>{{ end }}
                    <span class="trip-badge"><i class="bi bi-signpost-split"></i> {{ .WalkKM }} km walk</span>
                </div>
            </div>

            <ul class="trip-stops">
                {{ range .Stops }}
                <li class="trip-stop">
                    {{ if .Image }}
                    <figure class="trip-stop-figure">
                        <img class="trip-stop-photo" src="{{ .ImageURL 400 }}" alt="{{ .Name }}" loading="lazy">
                        <figcaption class="trip-stop-credit">
                            <a href="{{ .ImagePageURL }}" target="_blank" rel="noopener noreferrer">Photo: Wikimedia Commons</a>
                        </figcaption>
                    </figure>
                    {{ end }}
                    <span class="trip-stop-name">{{ .Name }}</span>
                    {{ if eq .Kind "indoor" }}<span class="trip-stop-kind">Indoors</span>
                    {{ else if eq .Kind "outdoor" }}<span class="trip-stop-kind">Outdoors</span>
                    {{ else if eq .Kind "mixed" }}<span class="trip-stop-kind">Indoor &amp; outdoor</span>{{ end }}
                </li>
                {{ end }}
            </ul>
        </li>
        {{ end }}
    </ol>

    {{ if or .Stays .StaysNote .BookingURL }}
    {{ template "stay_card" . }}
    {{ end }}
    {{ end }}

    <script>
        // HTMX only swaps successful answers, so let the card show its own problems too.
        (function () {
            if (window.tripCardErrorSwapWired) {
                return;
            }
            window.tripCardErrorSwapWired = true;

            document.body.addEventListener('htmx:beforeSwap', function (event) {
                var target = event.detail.target;
                if (target && target.id === 'trip-card' && event.detail.xhr.status >= 400) {
                    event.detail.shouldSwap = true;
                    event.detail.isError = false;
                }
            });
        })();
    </script>
</div>
