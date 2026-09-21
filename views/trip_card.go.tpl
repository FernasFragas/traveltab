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
        <span id="trip-spinner" class="trip-spinner"><i class="bi bi-arrow-repeat" aria-hidden="true"></i><span class="tt-visually-hidden">Planning…</span></span>
    </form>

    {{ if .Error }}
    <p class="trip-error" role="alert"><i class="bi bi-exclamation-circle"></i> {{ .Error }}</p>
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
