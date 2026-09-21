{{ if .GeneralInfo }}
<div class="weather-metrics" role="group" aria-label="Current weather">
    <div class="weather-metric weather-condition">
        <i class="bi {{ if .Presentation.ConditionIcon }}{{ .Presentation.ConditionIcon }}{{ else }}bi-thermometer-half{{ end }} weather-icon" aria-hidden="true"></i>
        <dl class="weather-reading">
            <dt class="weather-label">{{ if .GeneralInfo.Weather.Condition }}{{ .GeneralInfo.Weather.Condition }}{{ else }}Temperature{{ end }}</dt>
            <dd class="weather-value">{{ .GeneralInfo.Weather.Temperature }}<span class="weather-unit">°C</span></dd>
        </dl>
    </div>
    <div class="weather-metric">
        <i class="bi bi-droplet-fill weather-icon weather-icon-water" aria-hidden="true"></i>
        <dl class="weather-reading">
            <dt class="weather-label">Humidity</dt>
            <dd class="weather-value">{{ .GeneralInfo.Weather.Humidity }}<span class="weather-unit">%</span></dd>
        </dl>
    </div>
    <div class="weather-metric">
        <i class="bi bi-water weather-icon weather-icon-water" aria-hidden="true"></i>
        <dl class="weather-reading">
            <dt class="weather-label">Wave height</dt>
            <dd class="weather-value">{{ .GeneralInfo.Waves.Height }}<span class="weather-unit"> m</span></dd>
        </dl>
    </div>
</div>
{{ end }}
