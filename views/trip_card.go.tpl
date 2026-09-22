<div id="trip-card" class="trip-card">
    <div class="trip-card-head">
        <h3 class="trip-card-title">Plan your trip</h3>
        <p class="trip-card-subtitle">Famous places day by day, with museums saved for the rain.</p>
    </div>

    <form class="trip-form"
          aria-describedby="trip-help"
          hx-get="/plan"
          hx-target="#trip-card"
          hx-swap="outerHTML"
          hx-indicator="#trip-spinner">
        <input type="hidden" name="city" value="{{ .City }}">
        <input type="hidden" name="country" value="{{ .Country }}">
        <input type="hidden" name="lat" value="{{ .Lat }}">
        <input type="hidden" name="lon" value="{{ .Lon }}">

        <div class="trip-date-group">
            <label class="trip-field trip-start" for="trip-start">
                <span class="trip-field-label">Start</span>
                <input id="trip-start" class="trip-input" type="date" name="start" value="{{ .Start }}" min="{{ .MinStart }}" required autocomplete="off">
            </label>

            <div class="trip-field trip-end">
                <span class="trip-field-label">Ends</span>
                <output id="trip-end-value" class="trip-end" for="trip-start trip-days" data-trip-end aria-live="polite">{{ .EndLabel }}</output>
            </div>
        </div>

        <label class="trip-field trip-days" for="trip-days">
            <span class="trip-field-label">Days</span>
            <select id="trip-days" class="trip-select" name="days">
                {{- $selected := .Days }}
                {{- range .DayOptions }}
                <option value="{{ . }}"{{ if eq . $selected }} selected{{ end }}>{{ . }}</option>
                {{- end }}
            </select>
        </label>

        <button type="submit" class="trip-submit">
            Generate plan
            <i class="bi bi-arrow-right" aria-hidden="true"></i>
        </button>

        <p id="trip-help" class="trip-help">
            Pick a start date and how many days to plan. The last itinerary day updates automatically.
        </p>

        <span id="trip-spinner" class="trip-spinner htmx-indicator" role="status" aria-live="polite">
            <i class="bi bi-arrow-repeat" aria-hidden="true"></i>
            <span class="tt-visually-hidden">Planning…</span>
        </span>
    </form>

    {{ if .Error }}
    <p class="trip-error" role="alert"><i class="bi bi-exclamation-circle" aria-hidden="true"></i> {{ .Error }}</p>
    {{ end }}
</div>
