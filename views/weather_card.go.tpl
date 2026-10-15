{{ if .GeneralInfo }}
<div class="weather-card weather-metrics" role="group" aria-label="Current weather">
    <div class="weather-current">
        <i class="bi {{ if .Presentation.ConditionIcon }}{{ .Presentation.ConditionIcon }}{{ else }}bi-thermometer-half{{ end }} weather-icon weather-icon-main" aria-hidden="true"></i>
        <dl class="weather-reading weather-main">
            <dt class="weather-label">{{ if .GeneralInfo.Weather.Condition }}{{ .GeneralInfo.Weather.Condition }}{{ else }}Temperature{{ end }}</dt>
            <dd class="weather-value weather-temperature">{{ printf "%.0f" .GeneralInfo.Weather.Temperature }}<span class="weather-unit">°C</span></dd>
            <dd class="weather-feels">Feels like {{ printf "%.0f" .GeneralInfo.Weather.FeelsLike }}°C</dd>
        </dl>
    </div>
    <div class="weather-strip-details"><dl class="weather-details">
        <div class="weather-detail"><i class="bi bi-droplet-fill weather-icon-water" aria-hidden="true"></i><dt class="weather-detail-label">Humidity</dt><dd>{{ .GeneralInfo.Weather.Humidity }}%</dd></div>
        <div class="weather-detail"><i class="bi bi-wind weather-icon-water" aria-hidden="true"></i><dt class="weather-detail-label">Wind</dt><dd>{{ .Presentation.WindKmh }} km/h</dd></div>
        <div class="weather-detail"><i class="bi bi-water weather-icon-water" aria-hidden="true"></i><dt class="weather-detail-label">Waves</dt><dd>{{ .GeneralInfo.Waves.Height }} m</dd></div>
        {{ with .GeneralInfo.Weather.Sunset }}<div class="weather-detail"><i class="bi bi-sunset" aria-hidden="true"></i><dt class="weather-detail-label">Sunset</dt><dd>{{ . }}</dd></div>{{ end }}
    </dl>
    </div>
    <a class="tt-button weather-plan" href="#itinerary" data-tt-section="itinerary">Plan your trip ↗</a>
    <button type="button" class="weather-forecast-toggle" popovertarget="weather-popover" aria-label="Weather details and forecast"><span>Forecast </span>›</button>
    <div id="weather-popover" class="weather-popover" popover aria-label="Weather details and forecast">
        <h2>Weather and forecast</h2>
        <dl class="weather-popover-details">
            <div><dt>Humidity</dt><dd>{{ .GeneralInfo.Weather.Humidity }}%</dd></div>
            <div><dt>Wind</dt><dd>{{ .Presentation.WindKmh }} km/h</dd></div>
            <div><dt>Waves</dt><dd>{{ .GeneralInfo.Waves.Height }} m</dd></div>
            {{ with .GeneralInfo.Weather.Sunset }}<div><dt>Sunset</dt><dd>{{ . }}</dd></div>{{ end }}
        </dl>
    {{ with .Presentation.Forecast }}
    <ul class="weather-forecast" aria-label="Forecast">
        {{ range . }}
        <li class="weather-forecast-day" title="{{ .Condition }}">
            <span class="weather-forecast-name">{{ .Day }}</span>
            <i class="bi {{ .Icon }}" aria-hidden="true"></i><span class="tt-visually-hidden">{{ .Condition }}</span>
            <span class="weather-forecast-range"><strong>{{ .High }}°</strong> {{ .Low }}°</span>
        </li>
        {{ end }}
    </ul>
    {{ end }}    </div>
</div>
{{ end }}
