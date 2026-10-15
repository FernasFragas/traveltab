(function () {
    'use strict';

    // Pins for the notable (base) and writers layers, the popup a pin opens, and the place panel
    // its Details button opens.
    if (!window.TravelMap) return;

    var labelledPlaces = 5;
    // A pin whose name is still loading has no name in the feed; never show its QID.
    var pendingName = 'Loading name\u2026';
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
        if (event.key !== 'Escape') return;
        if (selectedPlace && !openPopup) { closeSelection(); return; }
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
        initializePanel(target);
        if (focusPanel) revealPanel(target);
    });
    document.body.addEventListener('click', function (event) {
        var posts = event.target.closest('[data-writers-toggle]');
        if (posts) {
            var list = posts.closest('.map-panel-writers').querySelector('ul');
            list.hidden = !list.hidden;
            posts.setAttribute('aria-expanded', String(!list.hidden));
            posts.textContent = list.hidden ? 'See all ›' : 'Show less ‹';
            return;
        }
        var share = event.target.closest('[data-map-share]');
        if (share) { sharePlace(share); return; }
    });
    document.addEventListener('travelmap:back', closeSelection);

    function closeSelection() {
        var slot = document.querySelector('[data-map-slot="panel"]');
        if (!slot) return;
        var empty = document.createElement('p');
        empty.textContent = 'Pick a place on the map';
        slot.replaceChildren(defaultPanel ? defaultPanel.cloneNode(true) : empty);
        selectedPlace = null;
        focusPanel = false;
        if (selectedPin && selectedPin.classList) selectedPin.classList.remove('is-selected');
        if (pinContext) pinContext.draw.clear('selected-callout');
        closePopup();
        var map = document.querySelector('[data-map-canvas]');
        if (map && map.focus) { map.tabIndex = -1; map.focus({ preventScroll: true }); }
        document.dispatchEvent(new CustomEvent('travelmap:closed'));
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
        if (isPhone()) {
            focusPanel = Boolean(place.panel_url);
            closePopup();
            context.selectPlace(place.qid);
            return;
        }
        var details = document.createElement('button');
        details.type = 'button';
        details.className = 'map-popup-details';
        details.textContent = '›';
        details.setAttribute('aria-label', 'Details');
        details.addEventListener('click', function () {
            selectedPin = pin;
            selectedPlace = { properties: place, geometry: feature.geometry };
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
        if (place.photo_url) {
            var thumbnail = document.createElement('img'); thumbnail.src = place.photo_url; thumbnail.alt = ''; thumbnail.loading = 'lazy'; thumbnail.className = 'map-popup-photo'; hideWhenBroken(thumbnail); box.appendChild(thumbnail);
        }
        var name = document.createElement('strong');
        name.className = 'map-popup-name';
        name.textContent = place.name || pendingName;
        box.appendChild(name);
        if (place.kind || place.area) { var area = document.createElement('span'); area.className = 'map-popup-meta'; area.textContent = place.kind_label || place.kind || place.area; box.appendChild(area); }
        if (place.panel_url && window.fetch && window.DOMParser) {
            window.fetch(place.panel_url).then(function (response) { return response.text(); }).then(function (html) {
                var article = new window.DOMParser().parseFromString(html, 'text/html').querySelector('.map-place-panel');
                if (!article || !article.dataset.placeArea || place.kind) return;
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

    function revealPanel(slot) {
        if (isPhone()) {
            slot.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
            slot.classList.add('is-highlighted');
            window.setTimeout(function () { slot.classList.remove('is-highlighted'); }, 1400);
        }
        focusPlacePanel(slot);
    }

    function initializePanel(slot) {
        Array.prototype.forEach.call(slot.querySelectorAll('[data-kind-icon]'), function (icon) {
            icon.className = 'bi bi-' + iconForKind(icon.dataset.kindIcon);
        });
        var hours = slot.querySelector('[data-opening-hours]');
        if (hours && window.TravelOpeningHours) hours.querySelector('strong').textContent = window.TravelOpeningHours(hours.dataset.openingHours);
        var best = slot.querySelector('[data-best-time-kind]');
        var tip = best && window.TravelBestTime && window.TravelBestTime(best.dataset.bestTimeKind);
        if (tip) { best.hidden = false; best.querySelector('strong').textContent = tip.value; best.querySelector('small').textContent = tip.tip; }
        var facts = slot.querySelector('.map-panel-facts');
        if (facts) facts.hidden = !hours && !tip;
        Array.prototype.forEach.call(slot.querySelectorAll('[data-carousel]'), initializeCarousel);
    }

    function initializeCarousel(media) {
        var photos = media.querySelectorAll('.map-panel-photo');
        if (photos.length < 2 || media.querySelector('[data-carousel-step]')) return;
        var current = 0;
        var badge = document.createElement('span');
        badge.className = 'map-panel-photo-count';
        var live = document.createElement('span');
        live.className = 'tt-visually-hidden'; live.setAttribute('aria-live', 'polite');
        media.appendChild(badge); media.appendChild(live);
        function update() {
            Array.prototype.forEach.call(photos, function (photo, index) { photo.hidden = index !== current; });
            badge.textContent = (current + 1) + ' / ' + photos.length;
            live.textContent = (current + 1) + ' of ' + photos.length;
        }
        [-1, 1].forEach(function (step) {
            var button = document.createElement('button'); button.type = 'button';
            button.className = 'map-panel-photo-arrow'; button.dataset.carouselStep = String(step);
            button.setAttribute('aria-label', step < 0 ? 'Previous photo' : 'Next photo');
            button.textContent = step < 0 ? '‹' : '›';
            button.addEventListener('click', function () { current = (current + step + photos.length) % photos.length; update(); });
            media.appendChild(button);
        });
        update();
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
        label.textContent = place.name;
        var kind = document.createElement('small'); kind.className = 'map-selected-kind'; kind.textContent = place.kind_label || place.kind || '';
        label.appendChild(kind);
        var arrow = document.createElement('span'); arrow.textContent = '›';
        button.appendChild(label);
        button.appendChild(arrow);
        button.addEventListener('click', function () { focusPanel = true; pinContext.selectPlace(place.qid); });
        pinContext.draw.markers('selected-callout', [{ lngLat: feature.geometry.coordinates, element: button,
            title: place.name, z: 2000 }]);
    }

    function iconForKind(kind) {
        var icons = {
            viewpoint: 'binoculars', cliff: 'binoculars', mountain: 'binoculars', waterfall: 'binoculars',
            museum: 'bank', gallery: 'bank',
            restaurant: 'fork-knife', cafe: 'fork-knife', market: 'fork-knife',
            park: 'tree', garden: 'tree', forest: 'tree', 'nature reserve': 'tree',
            trail: 'signpost-split', path: 'signpost-split', levada: 'signpost-split',
            church: 'building', cathedral: 'building', monastery: 'building'
        };
        return icons[String(kind || '').toLowerCase()] || 'geo-alt';
    }

    function hideWhenBroken(image) {
        image.addEventListener('error', function () { image.hidden = true; });
    }
})();
