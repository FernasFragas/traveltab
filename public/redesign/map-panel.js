(function () {
    'use strict';

    // Pins for the notable (base) and writers layers, and the place panel they open.
    if (!window.TravelMap) return;

    var labelledPlaces = 5;
    var shownMonograms = 3;
    var writerPinZ = 1000;

    window.TravelMap.registerLayer('notable', pinLayer(false));
    window.TravelMap.registerLayer('writers', pinLayer(true));
    document.addEventListener('travelmap:place-selected', loadPanel);
    document.body.addEventListener('htmx:afterSwap', function (event) {
        var monograms = event.detail.target.querySelector('.map-panel-monograms');
        if (monograms) drawMonograms(monograms, event.detail.target.querySelectorAll('[data-writer-host]'));
    });
    document.body.addEventListener('click', function (event) {
        var close = event.target.closest('.map-panel-close');
        var slot = close && close.closest('[data-map-slot="panel"]');
        if (!slot) return;
        var empty = document.createElement('p');
        empty.textContent = 'Pick a place on the map';
        slot.replaceChildren(empty);
    });

    function pinLayer(writers) {
        var group = null;
        return {
            add: function (map, context) {
                var labelled = biggestPlaces(context.features);
                group = window.L.layerGroup();
                context.features.forEach(function (feature) {
                    var place = feature.properties || {};
                    if (!place.qid || (place.writer_count > 0) !== writers) return;
                    var marker = window.L.marker([feature.geometry.coordinates[1], feature.geometry.coordinates[0]], {
                        icon: pinIcon(place, writers),
                        title: place.name,
                        zIndexOffset: writers ? writerPinZ : 0
                    });
                    marker.on('click', function () { context.selectPlace(place.qid); });
                    if (labelled.has(place.qid)) {
                        marker.bindTooltip(place.name, { permanent: true, direction: 'right', className: 'map-pin-label' });
                    }
                    group.addLayer(marker);
                });
                group.addTo(map);
            },
            remove: function (map) {
                if (group) map.removeLayer(group);
                group = null;
            }
        };
    }

    function loadPanel(event) {
        var shell = document.querySelector('[data-writers-map]');
        var slot = shell && shell.querySelector('[data-map-slot="panel"]');
        var place = event.detail.place && event.detail.place.properties;
        if (!slot || !place || !place.panel_url || !window.htmx) return;
        window.htmx.ajax('GET', place.panel_url, slot);
    }

    function drawMonograms(root, links) {
        var writers = new Map();
        Array.prototype.forEach.call(links, function (link) {
            if (!writers.has(link.dataset.writerHost)) writers.set(link.dataset.writerHost, link.dataset.blogName || link.dataset.writerHost);
        });
        root.replaceChildren();
        Array.from(writers).slice(0, shownMonograms).forEach(function (writer) {
            var monogram = document.createElement('span');
            monogram.className = 'map-panel-monogram';
            monogram.textContent = initials(writer[1]);
            monogram.title = writer[1];
            monogram.style.backgroundColor = hostColour(writer[0]);
            root.appendChild(monogram);
        });
        if (writers.size > shownMonograms) {
            var more = document.createElement('span');
            more.className = 'map-panel-monogram map-panel-monogram-more';
            more.textContent = '+' + (writers.size - shownMonograms);
            root.appendChild(more);
        }
    }

    function biggestPlaces(features) {
        return new Set(features
            .filter(function (feature) { return feature.properties && feature.properties.qid; })
            .sort(function (a, b) { return (b.properties.sitelinks || 0) - (a.properties.sitelinks || 0); })
            .slice(0, labelledPlaces)
            .map(function (feature) { return feature.properties.qid; }));
    }

    function pinIcon(place, writers) {
        return window.L.divIcon({
            className: 'map-pin ' + (writers ? 'map-pin-writer' : 'map-pin-notable'),
            html: writers ? '<span>' + Number(place.writer_count) + '</span>' : '',
            iconSize: writers ? [28, 28] : [16, 16],
            tooltipAnchor: writers ? [14, 0] : [8, 0]
        });
    }

    function initials(name) {
        return name.trim().split(/\s+/).slice(0, 2).map(function (word) { return word.charAt(0); }).join('').toUpperCase();
    }

    function hostColour(host) {
        var hash = 0;
        for (var i = 0; i < host.length; i++) hash = (hash * 31 + host.charCodeAt(i)) % 360;
        return 'hsl(' + hash + ' 55% 40%)';
    }
})();
