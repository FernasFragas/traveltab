(function () {
    'use strict';

    // Draws one writer's itinerary at a time: purple numbered pins in order and a line between them.
    // Routes come from the GeoJSON's top-level `itineraries`; stops are looked up by QID in its features.
    if (!window.TravelMap || window.TravelMap.itinerariesLayerRegistered) return;
    window.TravelMap.itinerariesLayerRegistered = true;

    var loads = {};
    var map = null;
    var shell = null;
    var group = null;
    var picker = null;
    var selected = 0;
    var drawNumber = 0;

    window.TravelMap.registerLayer('itineraries', { add: add, remove: remove });
    document.addEventListener('travelmap:ready', function () {
        var current = document.querySelector('[data-writers-map]');
        if (current) load(current);
    });

    function add(nextMap, context) {
        map = nextMap;
        shell = context && context.shell;
        if (!shell) return;
        var number = ++drawNumber;
        load(shell).then(function (data) {
            if (number === drawNumber && map) draw(data);
        });
    }

    function remove() {
        drawNumber++;
        clear();
        map = null;
        shell = null;
    }

    function clear() {
        if (group) group.remove();
        if (picker) picker.remove();
        group = null;
        picker = null;
    }

    // Fetches the GeoJSON once per URL and marks the shell when it has routes (the chip shows only then).
    function load(target) {
        var url = target.dataset.mapUrl;
        if (!url) return Promise.resolve({ features: [], itineraries: [] });
        if (!loads[url]) {
            loads[url] = fetch(url, { headers: { Accept: 'application/geo+json' } }).then(function (response) {
                if (!response.ok) throw new Error('Itineraries unavailable');
                return response.json();
            }).then(function (data) {
                return {
                    features: Array.isArray(data.features) ? data.features : [],
                    itineraries: Array.isArray(data.itineraries) ? data.itineraries : []
                };
            }).catch(function () {
                delete loads[url];
                return { features: [], itineraries: [] };
            });
        }
        return loads[url].then(function (data) {
            if (data.itineraries.length) target.dataset.itineraries = '';
            return data;
        });
    }

    function draw(data) {
        clear();
        if (!map || !data.itineraries.length) return;
        if (selected >= data.itineraries.length) selected = 0;
        var itinerary = data.itineraries[selected];
        var stops = stopsOf(itinerary, data.features);
        group = window.L.layerGroup().addTo(map);
        if (stops.length > 1) {
            group.addLayer(window.L.polyline(stops.map(function (stop) { return [stop.lat, stop.lon]; }), {
                className: 'itinerary-line'
            }).bindTooltip('Route by ' + itinerary.writer_host));
        }
        stops.forEach(function (stop, index) {
            group.addLayer(window.L.marker([stop.lat, stop.lon], {
                title: 'Stop ' + (index + 1) + ': ' + stop.name,
                icon: window.L.divIcon({
                    className: 'itinerary-pin-wrap',
                    iconSize: [26, 26],
                    html: '<span class="itinerary-pin">' + (index + 1) + '</span>'
                })
            }).bindPopup(popup(itinerary, stop)));
        });
        if (data.itineraries.length > 1) addPicker(data);
    }

    function stopsOf(itinerary, features) {
        var byQID = {};
        features.forEach(function (feature) {
            var coordinates = feature.geometry && feature.geometry.coordinates;
            if (feature.properties && coordinates) byQID[feature.properties.qid] = feature;
        });
        var stops = [];
        (itinerary.days || []).forEach(function (day) {
            (day.qids || []).forEach(function (qid) {
                var feature = byQID[qid];
                if (!feature) return;
                var lon = Number(feature.geometry.coordinates[0]);
                var lat = Number(feature.geometry.coordinates[1]);
                if (!Number.isFinite(lat) || !Number.isFinite(lon) || (lat === 0 && lon === 0) ||
                    Math.abs(lat) > 90 || Math.abs(lon) > 180) return;
                stops.push({ name: feature.properties.name || '', lat: lat, lon: lon });
            });
        });
        return stops;
    }

    function popup(itinerary, stop) {
        var box = document.createElement('div');
        box.className = 'itinerary-popup';
        var name = document.createElement('strong');
        name.textContent = stop.name;
        box.appendChild(name);
        var route = document.createElement('div');
        route.textContent = 'Route by ' + itinerary.writer_host;
        box.appendChild(route);
        if (/^https?:\/\//i.test(itinerary.post_url || '')) {
            var link = document.createElement('a');
            link.href = itinerary.post_url;
            link.textContent = itinerary.post_title || itinerary.post_url;
            link.setAttribute('target', '_blank');
            link.setAttribute('rel', 'noopener noreferrer');
            box.appendChild(link);
        }
        return box;
    }

    function addPicker(data) {
        var slot = shell && shell.querySelector('[data-map-slot="layers"]');
        if (!slot) return;
        picker = document.createElement('select');
        picker.className = 'itinerary-picker';
        picker.setAttribute('aria-label', "Choose a writer's itinerary");
        data.itineraries.forEach(function (itinerary, index) {
            var option = document.createElement('option');
            option.value = String(index);
            option.textContent = itinerary.writer_host + ': ' + (itinerary.post_title || itinerary.post_url);
            picker.appendChild(option);
        });
        picker.value = String(selected);
        picker.addEventListener('change', function () {
            selected = Number(picker.value) || 0;
            draw(data);
        });
        slot.appendChild(picker);
    }
})();
