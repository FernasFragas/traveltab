<div class="video-body">
    <div class="video-heading">
        <div>
            <h2 id="videos-title" class="tt-section-title">Explore through video</h2>
            <p class="tt-section-subtitle">Short travel films and tours for the area.</p>
        </div>
    </div>

    {{ $videos := .Presentation.Videos }}
    {{ with .Presentation }}{{ $videos = .Videos }}{{ end }}
    {{ $count := len $videos }}

    {{ if not $videos }}
    <p class="video-note">No travel videos are available for this destination right now.</p>
    {{ else }}
    <ul class="video-grid">
        {{ range $index, $video := $videos }}
        {{ if lt $index 4 }}
        <li class="video-card">
            <div class="video-player" data-video-player tabindex="-1">
                <button class="video-activate" type="button" data-video-activate
                        data-video-id="{{ $video.VideoID }}" data-video-title="{{ $video.Title }}"
                        aria-label="Play {{ $video.Title }} on this page">
                    <span class="video-thumbnail">
                        <span class="video-thumbnail-fallback" aria-hidden="true"><i class="bi bi-play-btn"></i></span>
                        <img class="video-thumbnail-image" src="{{ $video.ThumbnailURL }}" alt="" width="480" height="360" loading="lazy"
                             onerror="var thumbnail = this.closest('.video-thumbnail'); if (thumbnail) { thumbnail.classList.add('video-thumbnail-failed'); } this.hidden = true;">
                        <span class="video-play-badge"><i class="bi bi-play-fill" aria-hidden="true"></i></span>
                    </span>
                </button>
            </div>
            <div class="video-details">
                <a class="video-title" href="{{ $video.WatchURL }}" target="_blank" rel="noopener noreferrer">{{ $video.Title }}</a>
                <a class="video-watch" href="{{ $video.WatchURL }}" target="_blank" rel="noopener noreferrer">
                    Watch on YouTube
                    <i class="bi bi-arrow-up-right" aria-hidden="true"></i>
                </a>
            </div>
        </li>
        {{ end }}
        {{ end }}
    </ul>

    {{ if gt $count 4 }}
    <div class="video-expansion">
        <div class="video-extra" id="video-extra" data-video-extra hidden>
            <ul class="video-grid">
                {{ range $index, $video := $videos }}
                {{ if ge $index 4 }}
                <li class="video-card">
                    <div class="video-player" data-video-player tabindex="-1">
                        <button class="video-activate" type="button" data-video-activate
                                data-video-id="{{ $video.VideoID }}" data-video-title="{{ $video.Title }}"
                                aria-label="Play {{ $video.Title }} on this page">
                            <span class="video-thumbnail">
                                <span class="video-thumbnail-fallback" aria-hidden="true"><i class="bi bi-play-btn"></i></span>
                                <img class="video-thumbnail-image" src="{{ $video.ThumbnailURL }}" alt="" width="480" height="360" loading="lazy"
                                     onerror="var thumbnail = this.closest('.video-thumbnail'); if (thumbnail) { thumbnail.classList.add('video-thumbnail-failed'); } this.hidden = true;">
                                <span class="video-play-badge"><i class="bi bi-play-fill" aria-hidden="true"></i></span>
                            </span>
                        </button>
                    </div>
                    <div class="video-details">
                        <a class="video-title" href="{{ $video.WatchURL }}" target="_blank" rel="noopener noreferrer">{{ $video.Title }}</a>
                        <a class="video-watch" href="{{ $video.WatchURL }}" target="_blank" rel="noopener noreferrer">
                            Watch on YouTube
                            <i class="bi bi-arrow-up-right" aria-hidden="true"></i>
                        </a>
                    </div>
                </li>
                {{ end }}
                {{ end }}
            </ul>
        </div>
        <div class="video-more-row">
            <p class="video-count">Showing the first 4 of {{ $count }} videos.</p>
            <button class="video-expand" type="button" data-video-expand
                    aria-expanded="false" aria-controls="video-extra">
                <span data-video-expand-label>View all videos</span>
                <i class="bi bi-chevron-down" aria-hidden="true"></i>
            </button>
        </div>
    </div>
    {{ end }}
    {{ end }}
</div>
