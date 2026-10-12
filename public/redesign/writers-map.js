(function () {
    'use strict';

    if (window.TravelMap) return;

    // Shell: data-writers-map, data-map-canvas, mapLat/mapLon/mapUrl, optional mapDestination,
    // mapStyleUrl, mapTerrainUrl, mapEngine, mapTileUrl/mapAttribution (fallback), and the
    // contract's named slots. Engines: MapLibre here, Leaflet in writers-map-leaflet.js.
    var maplibreCSS = 'https://cdn.jsdelivr.net/npm/maplibre-gl@5.24.0/dist/maplibre-gl.css';
    var maplibreJS = 'https://cdn.jsdelivr.net/npm/maplibre-gl@5.24.0/dist/maplibre-gl.js';
    var defaultStyle = 'https://tiles.openfreemap.org/styles/liberty';
    var defaultTerrain = 'https://tiles.mapterhorn.com/{z}/{x}/{y}.webp';
    var camera = { zoom: 12.3, pitch: 55, bearing: -20 };
    var loadTimeout = 20000;
    var tileFailureLimit = 3;
    var layers = new Map();
    var engine = null;
    var canvas = null;
    var shell = null;
    var context = null;
    var maplibrePromise = null;
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
        if (engine) { removeLayers(); engine.clearAll(); }
        context = null;
        canvas.hidden = true;
        document.body.appendChild(canvas);
        shell = null;
    });
    document.body.addEventListener('htmx:afterSwap', function () { mount(); });
    document.body.addEventListener('htmx:historyRestore', function () { mount(); });
    document.body.addEventListener('click', function (event) {
        var info = event.target.closest('.writers-map-info');
        var open = document.querySelector('.writers-map-info[aria-expanded="true"]');
        if (open && open !== info) closeInfo(open);
        if (!info) return;
        var expanded = info.getAttribute('aria-expanded') !== 'true';
        info.setAttribute('aria-expanded', String(expanded));
        if (expanded) info.removeAttribute('data-dismissed');
        else info.setAttribute('data-dismissed', '');
    });
    document.body.addEventListener('focusin', function (event) {
        if (event.target.matches('.writers-map-info')) event.target.removeAttribute('data-dismissed');
    });
    document.body.addEventListener('focusout', function (event) {
        if (!event.target.matches('.writers-map-info')) return;
        event.target.setAttribute('aria-expanded', 'false');
        event.target.removeAttribute('data-dismissed');
    });
    document.body.addEventListener('keydown', function (event) {
        if (event.key !== 'Escape') return;
        var info = document.activeElement;
        if (!info || !info.matches('.writers-map-info')) info = document.querySelector('.writers-map-info[aria-expanded="true"]');
        if (info) closeInfo(info);
    });
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
        showStatus(shell, '');
        if (engine) {
            placeholder.replaceWith(canvas);
            canvas.hidden = false;
            engine.reset(latitude, longitude, shell);
            ready(currentRequest);
            return;
        }
        canvas = placeholder;
        if (!window.IntersectionObserver) {
            startMap(latitude, longitude, currentRequest);
        } else {
            whenVisible(canvas).then(function () {
                if (currentRequest === requestNumber) startMap(latitude, longitude, currentRequest);
            });
        }
    }

    function registerLayer(name, layer) {
        if (!name || !layer || typeof layer.add !== 'function' || typeof layer.remove !== 'function' || layers.has(name)) {
            throw new Error('Invalid or duplicate map layer: ' + name);
        }
        layers.set(name, { layer: layer, enabled: true });
        if (engine && context) addLayer(name, layer);
    }

    function setLayerEnabled(name, enabled) {
        var entry = layers.get(name);
        if (!entry) throw new Error('Unknown map layer: ' + name);
        enabled = Boolean(enabled);
        if (entry.enabled === enabled) return;
        entry.enabled = enabled;
        if (!engine || !context) return;
        if (enabled) addLayer(name, entry.layer);
        else removeLayer(name, entry.layer);
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

    function whenVisible(element) {
        if (!window.IntersectionObserver) return Promise.resolve();
        return new Promise(function (resolve) {
            var observer = new window.IntersectionObserver(function (entries) {
                if (!entries.some(function (entry) { return entry.isIntersecting; })) return;
                observer.disconnect();
                resolve();
            });
            observer.observe(element);
        });
    }

    function startMap(latitude, longitude, currentRequest) {
        var forceLeaflet = shell.dataset.mapEngine === 'leaflet';
        var webgl = document.createElement('canvas').getContext('webgl2') || document.createElement('canvas').getContext('webgl');
        var create = forceLeaflet || !webgl ? createLeaflet : createMaplibre;
        return create(canvas, shell, latitude, longitude, currentRequest).then(function (created) {
            if (currentRequest !== requestNumber) { created.destroy(); return; }
            engine = created;
            ready(currentRequest);
        }).catch(function () {
            if (currentRequest !== requestNumber) return;
            if (create === createLeaflet) return unavailable(currentRequest);
            return createLeaflet(canvas, shell, latitude, longitude).then(function (created) {
                if (currentRequest !== requestNumber) { created.destroy(); return; }
                engine = created;
                ready(currentRequest);
            }).catch(function () { unavailable(currentRequest); });
        });
    }

    function ready(currentRequest) {
        context = {
            engine: engine.name, destination: shell.dataset.mapDestination || shell.dataset.mapUrl,
            features: [], shell: shell, selectPlace: selectPlace, draw: engine.draw
        };
        document.dispatchEvent(new CustomEvent('travelmap:ready', {
            detail: { map: engine.map, destination: context.destination, engine: engine.name }
        }));
        loadPlaces(currentRequest);
    }

    function unavailable(currentRequest) {
        context = { destination: shell.dataset.mapDestination || shell.dataset.mapUrl,
            features: [], shell: shell, selectPlace: selectPlace };
        showStatus(shell, 'Map unavailable. Browse places below.');
        loadPlaces(currentRequest);
    }

    function loadMaplibre() {
        if (maplibrePromise) return maplibrePromise;
        var stylesheet = document.createElement('link');
        stylesheet.rel = 'stylesheet';
        stylesheet.href = maplibreCSS;
        stylesheet.integrity = 'sha384-uTttxo/aOKbdE5RlD/SPzSDoDmNvGlUYPjONi2MN/b7c9HPSvW07OIuyP7uL6jxK';
        stylesheet.crossOrigin = 'anonymous';
        document.head.appendChild(stylesheet);
        if (window.maplibregl) return maplibrePromise = Promise.resolve();
        maplibrePromise = new Promise(function (resolve, reject) {
            var script = document.createElement('script');
            script.src = maplibreJS;
            script.integrity = 'sha384-5+cfbwT0iiub6VsQAdn6yz16nr6sDiQoHx6tm4O8OVYXHYOxcffFmCJBL0dgdvGp';
            script.crossOrigin = 'anonymous';
            script.onload = function () { window.maplibregl ? resolve() : reject(new Error('MapLibre unavailable')); };
            script.onerror = reject;
            document.head.appendChild(script);
        });
        maplibrePromise.catch(function () { maplibrePromise = null; });
        return maplibrePromise;
    }

    function createLeaflet(target, root, latitude, longitude) {
        return window.TravelMapEngines.leaflet.create(target, root, latitude, longitude);
    }

    function createMaplibre(target, root, latitude, longitude, currentRequest) {
        return loadMaplibre().then(function () {
            return new Promise(function (resolve, reject) {
                var map;
                try {
                    map = new window.maplibregl.Map({ container: target,
                        style: root.dataset.mapStyleUrl || defaultStyle,
                        center: [longitude, latitude], zoom: camera.zoom, pitch: camera.pitch, bearing: camera.bearing });
                } catch (error) { reject(error); return; }
                var drawing = newMaplibreDrawing(map);
                var settled = false;
                var failedTiles = {};
                var timer = setTimeout(function () { fail(new Error('Map style timed out')); }, loadTimeout);
                map.on('style.load', function () { addTerrain(); drawing.restoreLines(); });
                map.on('sourcedata', function (event) {
                    if (event && event.sourceId && event.tile) failedTiles[event.sourceId] = 0;
                });
                map.on('error', function (event) {
                    if (currentRequest !== requestNumber || !event || !event.error) return;
                    if (!settled) { fail(event.error); return; }
                    if (!event.sourceId) { switchToLeaflet(map, currentRequest); return; }
                    failedTiles[event.sourceId] = (failedTiles[event.sourceId] || 0) + 1;
                    if (failedTiles[event.sourceId] >= tileFailureLimit) switchToLeaflet(map, currentRequest);
                });
                map.on('load', function () {
                    if (settled) return;
                    settled = true;
                    clearTimeout(timer);
                    map.addControl(new window.maplibregl.NavigationControl(), 'top-right');
                    map.addControl(new window.maplibregl.ScaleControl({ unit: 'metric' }), 'bottom-left');
                    resolve({ name: 'maplibre', map: map, draw: drawing.draw, clearAll: drawing.clearAll,
                        reset: function (lat, lon) { drawing.clearAll(); map.resize(); map.jumpTo({ center: [lon, lat], zoom: camera.zoom, pitch: camera.pitch, bearing: camera.bearing }); },
                        destroy: function () { drawing.clearAll(); map.remove(); } });
                });

                function addTerrain() {
                    var url = root.dataset.mapTerrainUrl;
                    if (url === '') return;
                    if (!map.getSource('terrain-dem')) map.addSource('terrain-dem', {
                        type: 'raster-dem', tiles: [(url || defaultTerrain).replace(/%7B([zxy])%7D/gi, '{$1}')],
                        tileSize: 512, encoding: 'terrarium', attribution: '© Mapterhorn'
                    });
                    map.setTerrain({ source: 'terrain-dem', exaggeration: 1 });
                }

                function fail(error) {
                    if (settled) return;
                    settled = true;
                    clearTimeout(timer);
                    map.remove();
                    reject(error);
                }
            });
        });
    }

    function switchToLeaflet(map, currentRequest) {
        if (!engine || engine.map !== map || currentRequest !== requestNumber) return;
        removeLayers();
        engine.destroy();
        engine = null;
        createLeaflet(canvas, shell, Number(shell.dataset.mapLat), Number(shell.dataset.mapLon)).then(function (created) {
            if (currentRequest !== requestNumber) { created.destroy(); return; }
            engine = created;
            context.engine = created.name;
            context.draw = created.draw;
            addLayers();
            document.dispatchEvent(new CustomEvent('travelmap:ready', {
                detail: { map: created.map, destination: context.destination, engine: created.name }
            }));
        }).catch(function () { showStatus(shell, 'Map unavailable. Browse places below.'); });
    }

    function newMaplibreDrawing(map) {
        var groups = new Map();
        function group(id) {
            if (!groups.has(id)) groups.set(id, { markers: [], line: null, control: null });
            return groups.get(id);
        }
        function clearMarkers(entry) { entry.markers.forEach(function (marker) { marker.remove(); }); entry.markers = []; }
        function clearLine(entry, id) {
            if (map.getLayer('draw-line-' + id)) map.removeLayer('draw-line-' + id);
            if (map.getSource('draw-line-' + id)) map.removeSource('draw-line-' + id);
            entry.line = null;
        }
        function clearControl(entry) { if (entry.control) map.removeControl(entry.control); entry.control = null; }
        function clear(id) {
            var entry = groups.get(id);
            if (!entry) return;
            clearMarkers(entry);
            clearLine(entry, id);
            clearControl(entry);
            groups.delete(id);
        }
        function drawLine(id, entry) {
            if (!entry.line || map.getLayer('draw-line-' + id)) return;
            try {
                map.addSource('draw-line-' + id, { type: 'geojson', data: {
                    type: 'Feature', geometry: { type: 'LineString', coordinates: entry.line.coords }
                } });
                var paint = { 'line-color': entry.line.style.color || '#a64027', 'line-width': entry.line.style.width || 3 };
                if (entry.line.style.dash) paint['line-dasharray'] = entry.line.style.dash;
                map.addLayer({ id: 'draw-line-' + id, type: 'line', source: 'draw-line-' + id, paint: paint });
            } catch (error) {
                // The style is reloading: style.load calls restoreLines, which draws this line again.
                if (map.getSource('draw-line-' + id)) map.removeSource('draw-line-' + id);
            }
        }
        return {
            clearAll: function () { Array.from(groups.keys()).forEach(clear); },
            restoreLines: function () { groups.forEach(function (entry, id) { drawLine(id, entry); }); },
            draw: {
                markers: function (id, items) {
                    var entry = group(id);
                    clearMarkers(entry);
                    (items || []).forEach(function (item) {
                        if (item.title) {
                            item.element.setAttribute('title', item.title);
                            item.element.setAttribute('aria-label', item.title);
                        }
                        if (item.onClick) item.element.addEventListener('click', item.onClick);
                        item.element.style.zIndex = String(item.z || 0);
                        entry.markers.push(new window.maplibregl.Marker({ element: item.element })
                            .setLngLat(item.lngLat).addTo(map));
                    });
                },
                line: function (id, coords, style) {
                    var entry = group(id);
                    clearLine(entry, id);
                    entry.line = { coords: coords, style: style || {} };
                    drawLine(id, entry);
                },
                control: function (id, element, position) {
                    var entry = group(id);
                    clearControl(entry);
                    entry.control = { onAdd: function () { return element; }, onRemove: function () { element.remove(); } };
                    map.addControl(entry.control, position || 'top-right');
                },
                clear: clear,
                fit: function (bounds, options) { map.fitBounds(bounds, { padding: options && options.padding || 0 }); },
                flyTo: function (lngLat, zoom) { map.flyTo({ center: lngLat, zoom: zoom }); }
            }
        };
    }

    function loadPlaces(currentRequest) {
        return fetch(shell.dataset.mapUrl, { headers: { Accept: 'application/geo+json' } }).then(function (response) {
            if (!response.ok) throw new Error('Places unavailable');
            return response.json();
        }).then(function (data) {
            if (currentRequest !== requestNumber) return;
            if (engine && context.features.length) removeLayers();
            context.features = Array.isArray(data.features) ? data.features : [];
            renderList(shell, context.features);
            if (engine) addLayers();
            showStatus(shell, !engine ? 'Map unavailable. Browse places below.' :
                data.complete === false ? 'Some map sources are unavailable.' : '');
        }).catch(function () {
            if (currentRequest !== requestNumber) return;
            renderList(shell, []);
            if (engine) addLayers();
            showStatus(shell, engine ? 'Places are unavailable. The map is still available.' : 'Map and places are unavailable.');
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
        layers.forEach(function (entry, name) { if (entry.enabled) addLayer(name, entry.layer); });
    }

    function removeLayers() {
        layers.forEach(function (entry, name) { if (entry.enabled) removeLayer(name, entry.layer); });
    }

    function addLayer(name, layer) {
        try { layer.add(engine.map, context); }
        catch (error) { console.warn('Map layer ' + name + ': ' + error.message); }
    }

    function removeLayer(name, layer) {
        try { layer.remove(engine.map, context); }
        catch (error) { console.warn('Map layer ' + name + ': ' + error.message); }
    }

    function closeInfo(info) {
        info.setAttribute('aria-expanded', 'false');
        info.setAttribute('data-dismissed', '');
    }
})();
