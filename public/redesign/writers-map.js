(function () {
    'use strict';

    if (window.TravelMap) return;

    // Shell: data-writers-map, data-map-canvas, mapLat/mapLon/mapUrl, optional
    // mapDestination/mapTileUrl/mapAttribution, and the contract's named slots.
    var leafletCSS = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.css';
    var leafletJS = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.js';
    var imageryTiles = 'https://tiles.maps.eox.at/wmts/1.0.0/s2cloudless_3857/default/GoogleMapsCompatible/{z}/{y}/{x}.jpg';
    var imageryAttribution = 'EOxCloudless <a href="https://cloudless.eox.at">cloudless.eox.at</a> by <a href="https://eox.at">EOX IT Services GmbH</a> (Contains modified Copernicus Sentinel data 2016 &amp; 2017), <a href="https://creativecommons.org/licenses/by/4.0/">CC BY 4.0</a>';
    var osmTiles = 'https://tile.openstreetmap.org/{z}/{x}/{y}.png';
    var osmAttribution = '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors';
    var layers = new Map();
    var map = null;
    var canvas = null;
    var shell = null;
    var context = null;
    var tileLayer = null;
    var leafletPromise = null;
    var requestNumber = 0;

    window.TravelMap = {
        mount: mount,
        registerLayer: registerLayer,
        setLayerEnabled: setLayerEnabled,
        selectPlace: selectPlace,
        updatePlan: updatePlan
    };

    document.body.addEventListener('htmx:beforeSwap', function (event) {
        if (!shell || !canvas || !event.detail.target.contains(shell)) return;
        requestNumber++;
        canvas.hidden = true;
        document.body.appendChild(canvas);
        shell = null;
    });
    document.body.addEventListener('htmx:afterSwap', function () { mount(); });
    document.body.addEventListener('htmx:historyRestore', function () { mount(); });
    mount();

    function mount(nextShell) {
        nextShell = nextShell || document.querySelector('[data-writers-map]');
        if (!nextShell || nextShell === shell) return;

        var latitude = Number(nextShell.dataset.mapLat);
        var longitude = Number(nextShell.dataset.mapLon);
        var placeholder = nextShell.querySelector('[data-map-canvas]');
        if (!placeholder || !nextShell.dataset.mapUrl || !Number.isFinite(latitude) ||
            !Number.isFinite(longitude) || Math.abs(latitude) > 90 || Math.abs(longitude) > 180) {
            showStatus(nextShell, 'Map unavailable.');
            return;
        }

        shell = nextShell;
        var currentRequest = ++requestNumber;
        showStatus(shell, 'Loading places…');
        loadLeaflet().then(function () {
            if (currentRequest !== requestNumber) return;
            showMap(placeholder, latitude, longitude);
            var destination = shell.dataset.mapDestination || shell.dataset.mapUrl;
            context = { destination: destination, features: [], shell: shell, selectPlace: selectPlace };
            addLayers();
            document.dispatchEvent(new CustomEvent('travelmap:ready', { detail: { map: map, destination: destination } }));
            return loadPlaces(currentRequest);
        }).catch(function () {
            if (currentRequest !== requestNumber) return;
            context = {
                destination: shell.dataset.mapDestination || shell.dataset.mapUrl,
                features: [], shell: shell, selectPlace: selectPlace
            };
            showStatus(shell, 'Map unavailable. Browse places below.');
            return loadPlaces(currentRequest);
        });
    }

    function registerLayer(name, layer) {
        if (!name || !layer || typeof layer.add !== 'function' || typeof layer.remove !== 'function' || layers.has(name)) {
            throw new Error('Invalid or duplicate map layer: ' + name);
        }
        layers.set(name, { layer: layer, enabled: true });
        if (map && context) layer.add(map, context);
    }

    function setLayerEnabled(name, enabled) {
        var entry = layers.get(name);
        if (!entry) throw new Error('Unknown map layer: ' + name);
        enabled = Boolean(enabled);
        if (entry.enabled === enabled) return;
        entry.enabled = enabled;
        if (!map || !context) return;
        if (enabled) entry.layer.add(map, context);
        else entry.layer.remove(map, context);
    }

    function selectPlace(qid) {
        if (!context) return;
        var feature = context.features.find(function (item) { return item.properties && item.properties.qid === qid; });
        if (!feature) return;
        document.dispatchEvent(new CustomEvent('travelmap:place-selected', {
            detail: { qid: qid, place: feature }
        }));
    }

    function updatePlan(days) {
        document.dispatchEvent(new CustomEvent('travelmap:plan-updated', { detail: { days: days } }));
    }

    function loadLeaflet() {
        if (leafletPromise) return leafletPromise;
        var stylesheet = document.createElement('link');
        stylesheet.rel = 'stylesheet';
        stylesheet.href = leafletCSS;
        stylesheet.integrity = 'sha256-p4NxAoJBhIIN+hmNHrzRCf9tD/miZyoHS5obTRR9BMY=';
        stylesheet.crossOrigin = 'anonymous';
        document.head.appendChild(stylesheet);
        if (window.L) {
            leafletPromise = Promise.resolve();
            return leafletPromise;
        }

        leafletPromise = new Promise(function (resolve, reject) {
            var script = document.createElement('script');
            script.src = leafletJS;
            script.integrity = 'sha256-20nQCchB9co0qIjJZRGuk2/Z9VM+kNiyxNV1lvTlZBo=';
            script.crossOrigin = 'anonymous';
            script.onload = function () { window.L ? resolve() : reject(new Error('Leaflet unavailable')); };
            script.onerror = reject;
            document.head.appendChild(script);
        });
        return leafletPromise;
    }

    function showMap(placeholder, latitude, longitude) {
        if (map) {
            removeLayers();
            placeholder.replaceWith(canvas);
            canvas.hidden = false;
            map.invalidateSize();
            map.setView([latitude, longitude], 12);
            if (tileLayer) map.removeLayer(tileLayer);
        } else {
            canvas = placeholder;
            map = window.L.map(canvas, { scrollWheelZoom: false }).setView([latitude, longitude], 12);
            window.L.control.scale({ imperial: false }).addTo(map);
        }
        var tileURL = (shell.dataset.mapTileUrl || imageryTiles).replace(/%7B([zxy])%7D/gi, '{$1}');
        var layer = window.L.tileLayer(tileURL, {
            maxZoom: 19,
            attribution: shell.dataset.mapAttribution || (tileURL === osmTiles ? osmAttribution : imageryAttribution)
        });
        tileLayer = layer;
        if (tileURL !== osmTiles) layer.on('tileerror', function () {
            if (tileLayer !== layer) return;
            map.removeLayer(layer);
            tileLayer = window.L.tileLayer(osmTiles, { maxZoom: 19, attribution: osmAttribution }).addTo(map);
        });
        layer.addTo(map);
    }

    function loadPlaces(currentRequest) {
        return fetch(shell.dataset.mapUrl, { headers: { Accept: 'application/geo+json' } }).then(function (response) {
            if (!response.ok) throw new Error('Places unavailable');
            return response.json();
        }).then(function (data) {
            if (currentRequest !== requestNumber) return;
            if (map) removeLayers();
            context.features = Array.isArray(data.features) ? data.features : [];
            renderList(shell, context.features);
            if (map) addLayers();
            showStatus(shell, !map ? 'Map unavailable. Browse places below.' :
                data.complete === false ? 'Some map sources are unavailable.' : '');
        }).catch(function () {
            if (currentRequest !== requestNumber) return;
            renderList(shell, []);
            showStatus(shell, map ? 'Places are unavailable. The map is still available.' : 'Map and places are unavailable.');
        });
    }

    function renderList(root, features) {
        var slot = root.querySelector('[data-map-slot="list"]');
        if (!slot) return;
        slot.replaceChildren();
        if (!features.length) {
            var empty = document.createElement('p');
            empty.textContent = 'No places to list yet.';
            slot.appendChild(empty);
            return;
        }
        var list = document.createElement('ul');
        list.setAttribute('aria-label', 'Places on the map');
        features.forEach(function (feature) {
            if (!feature.properties || !feature.properties.qid) return;
            var item = document.createElement('li');
            var button = document.createElement('button');
            button.type = 'button';
            button.textContent = feature.properties.name || feature.properties.qid;
            button.addEventListener('click', function () { selectPlace(feature.properties.qid); });
            item.appendChild(button);
            list.appendChild(item);
        });
        slot.appendChild(list);
    }

    function showStatus(root, message) {
        var slot = root.querySelector('[data-map-slot="status"]');
        if (!slot) return;
        slot.setAttribute('role', 'status');
        slot.setAttribute('aria-live', 'polite');
        slot.textContent = message;
        slot.hidden = !message;
    }

    function addLayers() {
        layers.forEach(function (entry) { if (entry.enabled) entry.layer.add(map, context); });
    }

    function removeLayers() {
        layers.forEach(function (entry) { if (entry.enabled) entry.layer.remove(map, context); });
    }
})();
