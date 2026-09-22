(function () {
    'use strict';

    if (window.travelTabDiscoveryWired) {
        return;
    }
    window.travelTabDiscoveryWired = true;

    var validVideoID = /^[A-Za-z0-9_-]{11}$/;

    function isDiscoveryNode(node) {
        return node instanceof Element && Boolean(node.closest('.stay-body, .video-body'));
    }

    function activateVideo(button) {
        var player = button.closest('[data-video-player]');
        if (!player || player.dataset.videoLoaded === 'true') {
            return;
        }

        var videoID = button.dataset.videoID || '';
        if (!validVideoID.test(videoID)) {
            return;
        }

        var title = button.dataset.videoTitle || 'TravelTab video';
        var frame = document.createElement('iframe');
        frame.className = 'video-embed';
        frame.src = 'https://www.youtube-nocookie.com/embed/' + encodeURIComponent(videoID) +
            '?autoplay=1&rel=0';
        frame.title = 'Play video: ' + title;
        frame.allow = 'accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share';
        frame.allowFullScreen = true;
        player.dataset.videoLoaded = 'true';
        player.replaceChildren(frame);
        player.focus({ preventScroll: true });
    }

    function toggleVideos(button) {
        var controls = button.getAttribute('aria-controls');
        var expanded = !controls ? null : document.getElementById(controls);
        if (!expanded || !expanded.hasAttribute('data-video-extra')) {
            return;
        }

        var willExpand = button.getAttribute('aria-expanded') !== 'true';
        expanded.hidden = !willExpand;
        button.setAttribute('aria-expanded', String(willExpand));

        var label = button.querySelector('[data-video-expand-label]');
        if (label) {
            label.textContent = willExpand ? 'Show fewer videos' : 'View all videos';
        }

        var icon = button.querySelector('.bi');
        if (icon) {
            icon.classList.toggle('bi-chevron-down', !willExpand);
            icon.classList.toggle('bi-chevron-up', willExpand);
        }
    }

    document.body.addEventListener('click', function (event) {
        if (!(event.target instanceof Element) || !isDiscoveryNode(event.target)) {
            return;
        }

        var activation = event.target.closest('[data-video-activate]');
        if (activation) {
            activateVideo(activation);
            return;
        }

        var expansion = event.target.closest('[data-video-expand]');
        if (expansion) {
            toggleVideos(expansion);
        }
    });
})();
