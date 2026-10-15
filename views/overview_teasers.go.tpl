<div class="overview-teasers">
    <article class="overview-teaser teaser-day">
        <div class="teaser-heading"><i class="bi bi-calendar-week" aria-hidden="true"></i><h3><span class="teaser-desktop-title">Plan your days</span><span class="teaser-phone-title">Plan your day in {{ .GeneralInfo.City }}</span></h3><a href="#itinerary" data-tt-section="itinerary" aria-label="View your itinerary"><span class="teaser-phone-title">View all </span>›</a></div>
        <p class="teaser-intro">Build your ideal itinerary with notable places, writers' picks and your saved spots.</p>
        <div class="teaser-day-preview" data-teaser-day>
        {{ if and .Trip .Trip.Plan .Trip.Plan.Days }}
            <p class="teaser-day-label">Day 1</p><ol class="teaser-stops">
            {{ range $i, $stop := (index .Trip.Plan.Days 0).Stops }}{{ if lt $i 3 }}<li>{{ with $stop.ImageURL 160 }}<img src="{{ . }}" alt="" loading="lazy">{{ end }}<span><small>{{ if eq $i 0 }}Morning{{ else if eq $i 1 }}Afternoon{{ else }}Evening{{ end }}</small>{{ $stop.Name }}</span></li>{{ end }}{{ end }}
            </ol>
        {{ else }}
            <p class="teaser-day-label">Suggested day</p><ol class="teaser-stops" data-suggested-day><li>Explore the notable places on the map.</li></ol>
        {{ end }}
        </div>
    </article>
    <article class="overview-teaser teaser-stays">
        <div class="teaser-heading"><i class="bi bi-building" aria-hidden="true"></i><h3>Where to stay</h3><a href="#stays" data-tt-section="stays">View all →</a></div>
        <p class="teaser-intro">Handpicked hotels near the best of {{ .GeneralInfo.City }}.</p>
        <div data-teaser-stays>
        {{ if and .Trip .Trip.Plan }}{{ if .Trip.Plan.StayCards }}{{ with index .Trip.Plan.StayCards 0 }}
        <div class="teaser-stay">{{ with .Image }}<img src="{{ .URL }}" alt="{{ .Alt }}" loading="lazy">{{ end }}<div><h4>{{ .Name }}</h4><p>{{ .Kind }}{{ if gt .Stars 0 }} · {{ .Stars }}-star property{{ end }}</p></div></div>
        {{ end }}{{ else }}<p>No stays available yet</p>{{ end }}{{ else }}<p>Generate a plan to see places to stay</p>{{ end }}
        </div>
    </article>
    {{ with .Presentation.Videos }}
    <article class="overview-teaser teaser-videos">
        <div class="teaser-heading"><i class="bi bi-play-circle-fill" aria-hidden="true"></i><h3>Travel videos</h3><a href="#videos" data-tt-section="videos" aria-label="View travel videos">›</a></div>
        <p class="teaser-intro">Watch destination guides, neighborhood walks and local experiences.</p>
        <div class="teaser-video-list">{{ range $i, $video := . }}{{ if lt $i 2 }}<a class="teaser-video" href="#videos" data-tt-section="videos"><span><img src="{{ .ThumbnailURL }}" alt="" loading="lazy"><i class="bi bi-play-circle-fill" aria-hidden="true"></i></span><strong>{{ .Title }}</strong></a>{{ end }}{{ end }}</div>
    </article>
    {{ end }}
</div>
