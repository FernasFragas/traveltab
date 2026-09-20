<div class="col-12 mb-1">
    {{ template "weather_display" . }}
</div>

{{ if .Trip }}
<div class="col-12 mb-1">
    {{ template "trip_card" .Trip }}
</div>
{{ end }}

<div class="col-12 mb-1">
    {{ template "video" . }}
</div>

<!-- Footer Section -->
<footer class="col-12 site-footer">
    <p>Made with ❤️ by Fernando Fragateiro</p>
    <div class="social-links">
        <a href="https://github.com/fernafrag" target="_blank" aria-label="GitHub"><i class="fab fa-github"></i></a>
        <a href="https://linkedin.com/in/your_linkedin" target="_blank" aria-label="LinkedIn"><i class="fab fa-linkedin"></i></a>
        <a href="https://your_medium_blog.medium.com" target="_blank" aria-label="Medium Blog"><i class="fab fa-medium"></i></a>
        <a href="https://your_personal_website.com" target="_blank" aria-label="Personal Website"><i class="fas fa-globe"></i></a>
        <!-- Add other social links as needed -->
    </div>
        <p class="data-credits">
            Places: Wikipedia &amp; Wikidata · Photos: Wikimedia Commons · Weather: Open-Meteo (CC BY 4.0) · Map data © OpenStreetMap contributors
        </p>
</footer> 