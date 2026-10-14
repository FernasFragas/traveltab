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
    var defaultPanel = null;
    var selectedPin = null;
    var selectedPlace = null;
    var pinContext = null;

    window.TravelMap.registerLayer('notable', pinLayer(false));
    window.TravelMap.registerLayer('writers', pinLayer(true));
    document.addEventListener('travelmap:place-selected', function (event) {
        loadPanel(event);
        showSelectedCallout(event);
    });
    document.addEventListener('travelmap:ready', function () {
        openSharedPlace();
    });
    document.addEventListener('keydown', function (event) {
        if (event.defaultPrevented) return;
        var activeSheet = document.querySelector('[data-map-slot="panel"].is-sheet');
        if (event.key === 'Tab' && activeSheet) {
            var focusable = activeSheet.querySelectorAll('button:not([hidden]), a[href], summary, [tabindex="0"]');
            if (focusable.length) {
                var first = focusable[0];
                var last = focusable[focusable.length - 1];
                if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
                else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
            }
        }
        if (event.key !== 'Escape') return;
        if (activeSheet) { closeSheet(activeSheet); return; }
        if (!openPopup) return;
        var pin = openPopup.pin;
        closePopup();
        if (pin && pin.focus) pin.focus();
    });
    document.body.addEventListener('htmx:afterSwap', function (event) {
        var target = event.detail && event.detail.target;
        if (!target) return;
        var article = target.querySelector('.map-place-panel');
        if (article && selectedPlace) {
            selectedPlace.properties.area = article.dataset.placeArea || '';
            showSelectedCallout({ detail: { place: selectedPlace } });
        }
        if (focusPanel) focusPlacePanel(target);
        var monograms = target.querySelector('.map-panel-monograms');
        if (monograms) drawMonograms(monograms, target.querySelectorAll('[data-writer-host]'));
        if (target.querySelector('.map-place-panel')) showPeek(target);
    });
    document.body.addEventListener('click', function (event) {
        var posts = event.target.closest('[data-writer-posts]');
        if (posts) { var section = posts.closest('.map-place-panel').querySelector('.map-panel-writers'); if (section) section.scrollIntoView({ behavior: 'smooth' }); return; }
        var share = event.target.closest('[data-map-share]');
        if (share) { sharePlace(share); return; }
        var close = event.target.closest('.map-panel-close');
        var slot = close && close.closest('[data-map-slot="panel"]');
        if (!slot) return;
        if (slot.classList && slot.classList.contains('is-sheet')) { closeSheet(slot); return; }
        resetPanel(slot);
    });
    function resetPanel(slot) {
        var empty = document.createElement('p');
        empty.textContent = 'Pick a place on the map';
        slot.replaceChildren(defaultPanel ? defaultPanel.cloneNode(true) : empty);
        selectedPlace = null;
        var peek = document.querySelector('[data-map-panel-peek]');
        if (peek) peek.hidden = true;
    }

    function pinLayer(writers) {
        var id = writers ? 'writers' : 'notable';
        var zoomMap = null;
        var updateLabels = null;
        return {
            add: function (map, context) {
                pinContext = context;
                var labelled = biggestPlaces(context.features);
                var matching = context.features.filter(function (feature) {
                    var place = feature.properties || {};
                    return place.qid && (place.writer_count > 0) === writers;
                });
                if (writers && context.draw.clusters) {
                    context.draw.clusters(id, matching, { color: '#a582d4', labelled: Array.from(labelled), pinElement: function (feature) { return pinElement(feature.properties, true, labelled.has(feature.properties.qid)); } }, function (feature, clickedPin) {
                        var buttons = document.querySelectorAll ? document.querySelectorAll('.travelmap-cluster-list button[data-qid]') : [];
                        var focusPin = Array.prototype.find.call(buttons, function (button) {
                            return button.getAttribute('data-qid') === feature.properties.qid;
                        }) || document.activeElement;
                        showPopup(context, feature, clickedPin || focusPin, id);
                    });
                    return;
                }
                var markers = matching.map(function (feature) {
                    var place = feature.properties;
                    var marker = {
                        lngLat: feature.geometry.coordinates,
                        element: pinElement(place, writers, labelled.has(place.qid)),
                        title: place.name || pendingName,
                        onClick: function () { showPopup(context, feature, marker.element, id); },
                        z: writers ? writerPinZ : 0
                    };
                    return marker;
                });
                context.draw.markers(id, markers);
                if (!writers && map && map.on) {
                    zoomMap = map;
                    updateLabels = function () {
                        markers.forEach(function (marker) {
                            var label = marker.element.querySelector('.map-pin-label');
                            if (label) label.hidden = map.getZoom() < 15;
                        });
                    };
                    map.on('zoomend', updateLabels);
                    updateLabels();
                }
            },
            remove: function (_map, context) {
                if (zoomMap && updateLabels) zoomMap.off('zoomend', updateLabels);
                zoomMap = null;
                updateLabels = null;
                if (openPopup && openPopup.layer === id) closePopup();
                context.draw.clear(id);
            }
        };
    }

    function showPopup(context, feature, pin, layer) {
        var place = feature.properties;
        if (selectedPin && selectedPin.classList) selectedPin.classList.remove('is-selected');
        selectedPin = pin;
        if (pin && pin.classList) pin.classList.add('is-selected');
        var details = document.createElement('button');
        details.type = 'button';
        details.className = 'map-popup-details';
        details.textContent = '›';
        details.setAttribute('aria-label', 'Details');
        details.addEventListener('click', function () {
            selectedPin = pin;
            selectedPlace = { properties: place, geometry: feature.geometry };
            focusPanel = Boolean(place.panel_url) && !isPhone();
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
        if (place.photo_url) {
            var thumbnail = document.createElement('img'); thumbnail.src = place.photo_url; thumbnail.alt = ''; thumbnail.loading = 'lazy'; thumbnail.className = 'map-popup-photo'; hideWhenBroken(thumbnail); box.appendChild(thumbnail);
        }
        var name = document.createElement('strong');
        name.className = 'map-popup-name';
        name.textContent = place.name || pendingName;
        box.appendChild(name);
        if (place.area) { var area = document.createElement('span'); area.className = 'map-popup-meta'; area.textContent = place.area; box.appendChild(area); }
        if (place.panel_url && window.fetch && window.DOMParser) {
            window.fetch(place.panel_url).then(function (response) { return response.text(); }).then(function (html) {
                var article = new window.DOMParser().parseFromString(html, 'text/html').querySelector('.map-place-panel');
                if (!article || !article.dataset.placeArea) return;
                place.area = article.dataset.placeArea;
                var area = document.createElement('span'); area.className = 'map-popup-meta'; area.textContent = place.area; box.insertBefore(area, details);
            }).catch(function () {});
        }
        box.appendChild(details);
        return box;
    }

    function focusPlacePanel(panel) {
        var heading = panel.querySelector('.map-place-panel h3');
        if (!heading) return;
        focusPanel = false;
        heading.tabIndex = -1;
        heading.focus({ preventScroll: true });
    }

    function loadPanel(event) {
        var shell = document.querySelector('[data-writers-map]');
        var slot = shell && shell.querySelector('[data-map-slot="panel"]');
        var place = event.detail.place && event.detail.place.properties;
        if (!slot || !place || !place.panel_url || !window.htmx) return;
        var markers = document.querySelectorAll ? document.querySelectorAll('.map-pin') : [];
        Array.prototype.forEach.call(markers, function (pin) { if (pin.classList) pin.classList.remove('is-selected'); });
        selectedPin = Array.prototype.find.call(markers, function (pin) { return pin.getAttribute('data-qid') === place.qid; }) || selectedPin;
        if (selectedPin && selectedPin.classList) selectedPin.classList.add('is-selected');
        var initial = slot.querySelector('.map-default-panel');
        if (initial) defaultPanel = initial.cloneNode(true);
        selectedPlace = event.detail.place;
        window.htmx.ajax('GET', place.panel_url, slot);
    }

    function showPeek(slot) {
        updateWriterCard(slot);
        if (!isPhone() || !slot.querySelector('.map-place-panel')) return;
        if (!slot.querySelector('.map-sheet-handle')) {
            var handle = document.createElement('button');
            handle.type = 'button'; handle.className = 'map-sheet-handle';
            handle.setAttribute('aria-label', 'Expand place details');
            handle.setAttribute('aria-expanded', 'false');
            handle.textContent = '━';
            handle.addEventListener('click', function () {
                if (dragged) { dragged = false; return; }
                var expanded = slot.classList.toggle('is-expanded');
                handle.setAttribute('aria-expanded', String(expanded));
                handle.setAttribute('aria-label', expanded ? 'Collapse place details' : 'Expand place details');
            });
            handle.addEventListener('keydown', function (event) {
                if (event.key !== 'Enter' && event.key !== ' ') return;
                event.preventDefault(); handle.click();
            });
            var startY;
            var dragged = false;
            handle.addEventListener('pointerdown', function (event) { startY = event.clientY; handle.setPointerCapture(event.pointerId); });
            handle.addEventListener('pointerup', function (event) {
                if (Math.abs(event.clientY - startY) < 20) return;
                dragged = true;
                var expanded = event.clientY < startY;
                slot.classList.toggle('is-expanded', expanded);
                handle.setAttribute('aria-expanded', String(expanded));
                handle.setAttribute('aria-label', expanded ? 'Collapse place details' : 'Expand place details');
            });
            slot.prepend(handle);
        }
        slot.classList.add('is-sheet');
        slot.setAttribute('role', 'dialog');
        slot.setAttribute('aria-modal', 'true');
        slot.setAttribute('aria-label', 'Place details');
        focusPlacePanel(slot);
    }

    function updateWriterCard(slot) {
        var card = slot.querySelector('.map-panel-writer-card');
        if (!card || !pinContext || !selectedPlace) return;
        var qid = selectedPlace.properties.qid;
        var routes = (pinContext.itineraries || []).filter(function (route) {
            return (route.days || []).some(function (day) { return (day.qids || []).indexOf(qid) >= 0; });
        });
        if (!routes.length) return;
        var names = [];
        routes.forEach(function (route) {
            var link = Array.prototype.find.call(slot.querySelectorAll('[data-writer-host]'), function (item) { return item.dataset.writerHost === route.writer_host; });
            var source = Array.prototype.find.call(slot.querySelectorAll('[data-writer-source-host]'), function (item) { return item.dataset.writerSourceHost === route.writer_host; });
            var name = source ? source.dataset.writerSourceName : link ? link.dataset.blogName : route.writer_host;
            if (names.indexOf(name) < 0) names.push(name);
        });
        var text = card.querySelector('.map-panel-mentioned');
        text.textContent = 'Featured in ' + routes.length + " writers' itineraries · Including " + names.join(', ');
        if (!card.querySelector('[data-writer-posts]')) {
            var arrow = document.createElement('button'); arrow.type = 'button'; arrow.textContent = '›';
            arrow.setAttribute('data-writer-posts', ''); arrow.setAttribute('aria-label', "Go to writers' posts"); card.appendChild(arrow);
        }
    }

    function closeSheet(slot) {
        slot.classList.remove('is-sheet', 'is-expanded');
        slot.removeAttribute('role');
        slot.removeAttribute('aria-modal');
        slot.removeAttribute('aria-label');
        if (selectedPin) selectedPin.focus({ preventScroll: true });
    }

    function isPhone() {
        return window.innerWidth <= 600;
    }

    function sharePlace(button) {
        var root = button.closest('.map-place-panel');
        var qid = root && root.dataset.placeQid;
        if (!qid) return;
        var url = new URL(window.location.href);
        url.searchParams.set('place', qid);
        url.hash = '#overview';
        var feedback = root.querySelector('.map-panel-feedback');
        if (window.navigator.share) {
            window.navigator.share({ title: root.querySelector('h3').textContent, url: url.href }).catch(function () {});
        } else if (window.navigator.clipboard && window.navigator.clipboard.writeText) {
            window.navigator.clipboard.writeText(url.href).then(function () { feedback.textContent = 'Link copied'; },
                function () { feedback.textContent = 'Could not copy link'; });
        } else feedback.textContent = 'Could not copy link';
    }

    function openSharedPlace() {
        if (!window.location || !window.htmx) return;
        var qid = new URL(window.location.href).searchParams.get('place');
        var shell = document.querySelector('[data-writers-map]');
        var slot = shell && shell.querySelector('[data-map-slot="panel"]');
        if (!qid || !slot) return;
        var initial = slot.querySelector('.map-default-panel');
        if (initial) defaultPanel = initial.cloneNode(true);
        window.htmx.ajax('GET', '/map/' + encodeURIComponent(shell.dataset.mapDestination) + '/' + encodeURIComponent(qid) +
            '?lat=' + encodeURIComponent(shell.dataset.mapLat) + '&lon=' + encodeURIComponent(shell.dataset.mapLon), slot);
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
        if (!writers && /^(viewpoint|cliff|mountain|waterfall|park|garden|forest|nature reserve|trail|path|levada)$/i.test(place.kind || '')) pin.className += ' map-pin-nature';
        pin.title = place.name || pendingName;
        pin.setAttribute('aria-label', place.name || pendingName);
        pin.setAttribute('data-qid', place.qid);
        if (writers) {
            var quill = document.createElement('i');
            quill.className = 'bi bi-feather';
            quill.setAttribute('aria-hidden', 'true');
            pin.appendChild(quill);
        } else {
            var icon = document.createElement('i');
            icon.className = 'bi bi-' + iconForKind(place.kind);
            icon.setAttribute('aria-hidden', 'true');
            pin.appendChild(icon);
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

    function showSelectedCallout(event) {
        if (!pinContext) return;
        pinContext.draw.clear('selected-callout');
        var feature = event.detail && event.detail.place;
        var place = feature && feature.properties;
        if (!place || !feature.geometry) return;
        var button = document.createElement('button');
        button.type = 'button';
        button.className = 'map-pin map-pin-selected';
        button.setAttribute('aria-label', 'Show ' + place.name + ' details');
        if (place.photo_url) {
        var photo = document.createElement('img');
        photo.className = 'map-pin-photo';
        photo.src = place.photo_url;
        photo.alt = '';
        photo.loading = 'lazy';
        hideWhenBroken(photo);
        button.appendChild(photo);
        }
        var label = document.createElement('span');
        label.className = 'map-selected-name';
        label.textContent = place.name + (place.area ? ' · ' + place.area : '') + ' ›';
        button.appendChild(label);
        button.addEventListener('click', function () { pinContext.selectPlace(place.qid); if (isPhone()) showPeek(document.querySelector('[data-map-slot="panel"]')); });
        pinContext.draw.markers('selected-callout', [{ lngLat: feature.geometry.coordinates, element: button,
            title: place.name, z: 2000 }]);
    }

    function iconForKind(kind) {
        var icons = {
            viewpoint: 'binoculars', cliff: 'binoculars', mountain: 'binoculars', waterfall: 'binoculars',
            museum: 'bank', gallery: 'bank',
            restaurant: 'cup-hot', cafe: 'cup-hot', market: 'cup-hot',
            park: 'tree', garden: 'tree', forest: 'tree', 'nature reserve': 'tree',
            trail: 'signpost-split', path: 'signpost-split', levada: 'signpost-split',
            church: 'building', cathedral: 'building', monastery: 'building'
        };
        return icons[String(kind || '').toLowerCase()] || 'geo-alt';
    }

    function initials(name) {
        return name.trim().split(/\s+/).slice(0, 2).map(function (word) { return word.charAt(0); }).join('').toUpperCase();
    }

    function hostColour(host) {
        var hash = 0;
        for (var i = 0; i < host.length; i++) hash = (hash * 31 + host.charCodeAt(i)) % 360;
        return 'hsl(' + hash + ' 55% 40%)';
    }

    function hideWhenBroken(image) {
        image.addEventListener('error', function () { image.hidden = true; });
    }
})();
