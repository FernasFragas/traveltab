(function () {
    'use strict';

    // Pins for the notable (base) and writers layers, the popup a pin opens, and the place panel
    // its Details button opens.
    if (!window.TravelMap) return;

    var labelledPlaces = 5;
    // A pin whose name is still loading has no name in the feed; never show its QID.
    var pendingName = 'Loading name\u2026';
    var shownMonograms = 3;
    var writerPinZ = 1000;
    var popupId = 'pin-popup';
    var openPopup = null;
    var focusPanel = false;

    window.TravelMap.registerLayer('notable', pinLayer(false));
    window.TravelMap.registerLayer('writers', pinLayer(true));
    document.addEventListener('travelmap:place-selected', loadPanel);
    document.addEventListener('keydown', function (event) {
        if (event.key !== 'Escape' || !openPopup) return;
        var pin = openPopup.pin;
        closePopup();
        pin.focus();
    });
    document.body.addEventListener('htmx:afterSwap', function (event) {
        if (focusPanel) focusPlacePanel(event.detail.target);
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
        var id = writers ? 'writers' : 'notable';
        return {
            add: function (_map, context) {
                var labelled = biggestPlaces(context.features);
                var markers = context.features.map(function (feature) {
                    var place = feature.properties || {};
                    if (!place.qid || (place.writer_count > 0) !== writers) return null;
                    var marker = {
                        lngLat: feature.geometry.coordinates,
                        element: pinElement(place, writers, labelled.has(place.qid)),
                        title: place.name || pendingName,
                        onClick: function () { showPopup(context, feature, marker.element, id); },
                        z: writers ? writerPinZ : 0
                    };
                    return marker;
                }).filter(Boolean);
                context.draw.markers(id, markers);
            },
            remove: function (_map, context) {
                if (openPopup && openPopup.layer === id) closePopup();
                context.draw.clear(id);
            }
        };
    }

    function showPopup(context, feature, pin, layer) {
        var place = feature.properties;
        var details = document.createElement('button');
        details.type = 'button';
        details.className = 'map-popup-details';
        details.textContent = 'Details';
        details.addEventListener('click', function () {
            focusPanel = Boolean(place.panel_url);
            closePopup();
            context.selectPlace(place.qid);
        });
        context.draw.popup(popupId, feature.geometry.coordinates, popupElement(place, details), {
            onClose: function () { openPopup = null; }
        });
        openPopup = { pin: pin, layer: layer, context: context };
        details.focus({ preventScroll: true });
    }

    function closePopup() {
        var current = openPopup;
        openPopup = null;
        if (current) current.context.draw.clear(popupId);
    }

    function popupElement(place, details) {
        var box = document.createElement('div');
        box.className = 'map-popup';
        box.setAttribute('role', 'dialog');
        box.setAttribute('aria-label', place.name || pendingName);
        var name = document.createElement('strong');
        name.className = 'map-popup-name';
        name.textContent = place.name || pendingName;
        box.appendChild(name);
        var count = Number(place.writer_count);
        var meta = count > 0 ? 'Mentioned by ' + count + ' writer' + (count === 1 ? '' : 's') : place.kind;
        if (meta) {
            var line = document.createElement('p');
            line.className = 'map-popup-meta';
            line.textContent = meta;
            box.appendChild(line);
        }
        if (count > 0) {
            var monograms = document.createElement('span');
            monograms.className = 'map-panel-monograms map-popup-monograms';
            box.appendChild(monograms);
            loadMonograms(place.panel_url, monograms);
        }
        box.appendChild(details);
        return box;
    }

    // The popup lists up to three blogs, taken from the place's panel fragment.
    function loadMonograms(url, root) {
        if (!url || !window.fetch || !window.DOMParser) return;
        window.fetch(url).then(function (response) {
            if (!response.ok) throw new Error('Panel unavailable');
            return response.text();
        }).then(function (text) {
            var links = new window.DOMParser().parseFromString(text, 'text/html').querySelectorAll('[data-writer-host]');
            drawMonograms(root, links);
            var names = [];
            Array.prototype.forEach.call(links, function (link) {
                var name = link.dataset.blogName || link.dataset.writerHost;
                if (names.indexOf(name) < 0) names.push(name);
            });
            root.setAttribute('role', 'img');
            root.setAttribute('aria-label', 'Blogs: ' + names.slice(0, shownMonograms).join(', ') + (names.length > shownMonograms ? ' and more' : ''));
        }).catch(function () { /* the popup still has the name and count */ });
    }

    function focusPlacePanel(panel) {
        focusPanel = false;
        var heading = panel.querySelector('.map-place-panel h3');
        if (!heading) return;
        heading.tabIndex = -1;
        heading.focus({ preventScroll: true });
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
            .filter(function (feature) { return feature.properties && feature.properties.qid && feature.properties.name; })
            .sort(function (a, b) { return (b.properties.sitelinks || 0) - (a.properties.sitelinks || 0); })
            .slice(0, labelledPlaces)
            .map(function (feature) { return feature.properties.qid; }));
    }

    function pinElement(place, writers, labelled) {
        var pin = document.createElement('button');
        pin.type = 'button';
        pin.className = 'map-pin ' + (writers ? 'map-pin-writer' : 'map-pin-notable');
        pin.title = place.name || pendingName;
        pin.setAttribute('aria-label', place.name || pendingName);
        if (writers) {
            var count = document.createElement('span');
            count.textContent = String(Number(place.writer_count));
            pin.appendChild(count);
        }
        if (labelled) {
            var label = document.createElement('span');
            label.className = 'map-pin-label';
            label.textContent = place.name;
            label.setAttribute('aria-hidden', 'true');
            pin.appendChild(label);
        }
        return pin;
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
