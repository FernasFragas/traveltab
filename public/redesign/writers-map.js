(function () {
    'use strict';

    if (window.TravelMap) return;

    // Shell: data-writers-map, data-map-canvas, mapLat/mapLon/mapUrl, optional mapDestination,
    // mapStyleUrl, mapTerrainUrl, mapEngine, mapTileUrl/mapAttribution (fallback), and the
    // contract's named slots. Engines: MapLibre here, Leaflet in writers-map-leaflet.js.
    var maplibreCSS = 'https://cdn.jsdelivr.net/npm/maplibre-gl@5.24.0/dist/maplibre-gl.css';
    var maplibreJS = 'https://cdn.jsdelivr.net/npm/maplibre-gl@5.24.0/dist/maplibre-gl.js';
    var pollDelay = 3000;
    var defaultStyle = 'https://tiles.openfreemap.org/styles/dark';
    var defaultTerrain = 'https://tiles.mapterhorn.com/{z}/{x}/{y}.webp';
    var buildingLow = '#2c3044';
    var buildingHigh = '#46414d';
    // Colours sampled from the Electroverse map (Lisbon), applied over OpenFreeMap dark layer ids, plus
    // the road weight and label size that go with them. Layer ids a style lacks are skipped. Only the
    // default style is changed. Minor roads are thin and dim at z10 to z13, so towns do not drown in lines.
    var minorRoadWidth = ['interpolate', ['exponential', 1.55], ['zoom'], 10, 0.4, 13, 1.2, 20, 20];
    var minorRoadOpacity = ['interpolate', ['linear'], ['zoom'], 10, 0.35, 14, 0.9];
    // Primary and trunk roads keep their weight; secondary and tertiary share their layers, so they thin by class.
    function byClass(main, other) { return ['match', ['get', 'class'], ['primary', 'trunk'], main, other]; }
    var majorRoadWidth = ['interpolate', ['exponential', 1.3], ['zoom'], 10, byClass(2, 0.7), 13, byClass(3.6, 1.4), 20, 20];
    var majorCasingWidth = ['interpolate', ['exponential', 1.3], ['zoom'], 10, byClass(3, 1.2), 13, byClass(4.8, 2), 20, 23];
    var majorRoadOpacity = ['interpolate', ['linear'], ['zoom'], 10, byClass(1, 0.4), 14, 1];
    var majorSubtleWidth = ['interpolate', ['linear'], ['zoom'], 6, 0, 8, byClass(1.5, 0.5)];
    var palette = [
        [['background', 'landcover_ice_shelf', 'landcover_glacier', 'landuse_residential', 'road_area_pier', 'road_pier'], { 'background-color': '#202a32', 'fill-color': '#202a32', 'line-color': '#202a32' }],
        [['water', 'waterway'], { 'fill-color': '#000e29', 'line-color': '#000e29' }],
        [['landcover_wood', 'landuse_park'], { 'fill-color': '#29382f', 'fill-pattern': null }],
        [['building'], { 'fill-color': '#2c3044', 'fill-outline-color': '#232b3d' }],
        [['highway_path', 'railway_transit', 'railway_minor', 'railway', 'aeroway-taxiway'], { 'line-color': '#2c3550' }],
        [['highway_minor', 'highway_major_subtle', 'highway_motorway_subtle'], { 'line-color': '#404860' }],
        [['highway_minor'], { 'line-width': minorRoadWidth, 'line-opacity': minorRoadOpacity }],
        [['highway_major_inner'], { 'line-width': majorRoadWidth, 'line-opacity': majorRoadOpacity }],
        [['highway_major_casing'], { 'line-width': majorCasingWidth, 'line-opacity': majorRoadOpacity }],
        [['highway_major_subtle'], { 'line-width': majorSubtleWidth }],
        [['highway_major_inner'], { 'line-color': '#4c576c' }],
        [['highway_motorway_inner', 'aeroway-runway'], { 'line-color': '#5a6274' }],
        [['highway_major_casing', 'highway_motorway_casing', 'aeroway-runway-casing'], { 'line-color': '#101c2e' }],
        [['highway_name_other', 'highway_name_motorway'], { 'text-color': '#9aa3b5', 'text-halo-color': '#202a32' }],
        [['water_name'], { 'text-color': '#6f7f9c', 'text-halo-color': '#000e29' }],
        [['place_other', 'place_suburb', 'place_village', 'place_town', 'place_city', 'place_city_large', 'place_state',
            'place_country_other', 'place_country_minor', 'place_country_major'], { 'text-color': '#e0e0e0', 'text-halo-color': '#101820' }],
        [['place_other', 'place_suburb', 'place_village', 'place_town', 'place_city'], { 'text-size': 11 }]
    ];
    // Opens centred on the destination at about a 200 m scale bar (zoom 14.3 at mid latitudes); the
    // view is not fitted to the pins, so places farther out are reached by panning.
    var camera = { zoom: 14.3, pitch: 55, bearing: -20 };
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
        if (engine && engine.draw && engine.draw.flyTo) engine.draw.flyTo(feature.geometry.coordinates, 15);
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
        installControls();
        document.dispatchEvent(new CustomEvent('travelmap:ready', {
            detail: { map: engine.map, destination: context.destination, engine: engine.name }
        }));
        loadPlaces(currentRequest);
    }

    function installControls() {
        var tools = document.createElement('div');
        tools.className = 'travelmap-tools';
        if (engine.name === 'maplibre') {
            var compass = document.createElement('button');
            compass.type = 'button';
            compass.className = 'travelmap-compass';
            compass.textContent = 'N';
            compass.setAttribute('aria-label', 'Reset bearing');
            compass.addEventListener('click', function () { engine.map.easeTo({ bearing: 0 }); });
            tools.appendChild(compass);
        }
        var locate = document.createElement('button');
        locate.type = 'button';
        locate.className = 'travelmap-locate';
        locate.setAttribute('aria-label', 'Locate me');
        locate.textContent = '⌖';
        locate.addEventListener('click', function () {
            var geolocation = window.navigator && window.navigator.geolocation;
            if (!geolocation) { showStatus(shell, 'Location unavailable.'); locate.disabled = true; return; }
            geolocation.getCurrentPosition(function (position) {
                var point = [position.coords.longitude, position.coords.latitude];
                var dot = document.createElement('span');
                dot.className = 'travelmap-location-dot';
                context.draw.markers('location', [{ lngLat: point, element: dot, title: 'Your location' }]);
                context.draw.flyTo(point, 15);
            }, function () {
                showStatus(shell, 'Location unavailable.');
                locate.disabled = true;
            }, { enableHighAccuracy: true, timeout: 10000 });
        });
        tools.appendChild(locate);
        if (engine.set3D) {
            var dimension = document.createElement('button');
            dimension.type = 'button';
            dimension.className = 'travelmap-dimension';
            dimension.textContent = '2D';
            dimension.setAttribute('aria-label', 'Switch to 2D map');
            var threeD = true;
            dimension.addEventListener('click', function () {
                threeD = !threeD;
                engine.set3D(threeD);
                dimension.textContent = threeD ? '2D' : '3D';
                dimension.setAttribute('aria-label', 'Switch to ' + (threeD ? '2D' : '3D') + ' map');
            });
            tools.appendChild(dimension);
        }
        ['+', '−'].forEach(function (label, index) {
            var zoom = document.createElement('button');
            zoom.type = 'button'; zoom.textContent = label;
            zoom.setAttribute('aria-label', index ? 'Zoom out' : 'Zoom in');
            zoom.addEventListener('click', function () { if (index) engine.map.zoomOut(); else engine.map.zoomIn(); });
            tools.appendChild(zoom);
        });
        if (window.TravelMapLayers) tools.appendChild(window.TravelMapLayers.createMenu());
        context.draw.control('map-tools', tools, 'top-right');
    }

    function unavailable(currentRequest) {
        context = { destination: shell.dataset.mapDestination || shell.dataset.mapUrl,
            features: [], shell: shell, selectPlace: selectPlace };
        showStatus(shell, 'Map unavailable.');
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
                var threeD = true;
                var settled = false;
                var failedTiles = {};
                var timer = setTimeout(function () { fail(new Error('Map style timed out')); }, loadTimeout);
                map.on('style.load', function () { addTerrain(); addPalette(); addBuildings(); drawing.restoreLines(); });
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
                    map.addControl(new window.maplibregl.ScaleControl({ unit: 'metric' }), 'bottom-right');
                    resolve({ name: 'maplibre', map: map, draw: drawing.draw, clearAll: drawing.clearAll,
                        set3D: function (enabled) {
                            threeD = enabled;
                            map.setPitch(enabled ? camera.pitch : 0);
                            map.setTerrain(enabled && map.getSource('terrain-dem') ? { source: 'terrain-dem', exaggeration: 1 } : null);
                        },
                        reset: function (lat, lon) { drawing.clearAll(); threeD = true; map.resize(); map.jumpTo({ center: [lon, lat], zoom: camera.zoom, pitch: camera.pitch, bearing: camera.bearing }); if (map.getSource('terrain-dem')) map.setTerrain({ source: 'terrain-dem', exaggeration: 1 }); },
                        destroy: function () { drawing.clearAll(); map.remove(); } });
                });

                function addTerrain() {
                    var url = root.dataset.mapTerrainUrl;
                    if (url === '') return;
                    if (!map.getSource('terrain-dem')) map.addSource('terrain-dem', {
                        type: 'raster-dem', tiles: [(url || defaultTerrain).replace(/%7B([zxy])%7D/gi, '{$1}')],
                        tileSize: 512, encoding: 'terrarium', attribution: '© Mapterhorn'
                    });
                    if (threeD) map.setTerrain({ source: 'terrain-dem', exaggeration: 1 });
                }

                // The default style only: each layer takes the properties of its own type (a fill
                // gets fill-color, a line line-color, ...), so one row can cover several types.
                // Label size is a layout property, everything else here is paint.
                function addPalette() {
                    if (root.dataset.mapStyleUrl) return;
                    var types = {};
                    (map.getStyle().layers || []).forEach(function (layer) { types[layer.id] = layer.type; });
                    var prefixes = { background: 'background-', fill: 'fill-', line: 'line-', symbol: 'text-' };
                    var layoutProperties = ['text-size'];
                    palette.forEach(function (row) {
                        row[0].forEach(function (id) {
                            var type = types[id];
                            if (!type) return;
                            Object.keys(row[1]).forEach(function (property) {
                                if (property.indexOf(prefixes[type]) !== 0) return;
                                if (layoutProperties.indexOf(property) !== -1) map.setLayoutProperty(id, property, row[1][property]);
                                else map.setPaintProperty(id, property, row[1][property]);
                            });
                        });
                    });
                    (map.getStyle().layers || []).forEach(function (layer) {
                        if (layer.type === 'symbol' && /^poi(?:_|$)/.test(layer.id) && !/^poi_(?:major|landmark)(?:_|$)/.test(layer.id)) {
                            map.setLayoutProperty(layer.id, 'visibility', 'none');
                        }
                    });
                }

                // The dark style has no 3D buildings, so draw them from its vector source. A style
                // that already extrudes buildings, or this layer from an earlier style.load, is kept.
                function addBuildings() {
                    var style = map.getStyle();
                    var layers = style.layers || [];
                    if (layers.some(function (layer) { return layer.type === 'fill-extrusion'; })) return;
                    var sources = style.sources || {};
                    var name = sources.openmaptiles ? 'openmaptiles' : Object.keys(sources).filter(function (id) { return sources[id].type === 'vector'; })[0];
                    if (!name) return;
                    var below;
                    var start = layers.findIndex(function (layer) { return layer['source-layer'] === 'building'; });
                    layers.slice(Math.max(start, 0)).some(function (layer) { if (layer.type === 'symbol') { below = layer.id; return true; } });
                    map.addLayer({ id: 'buildings-3d', type: 'fill-extrusion', source: name, 'source-layer': 'building', minzoom: 14,
                        paint: {
                            'fill-extrusion-color': ['interpolate', ['linear'], ['coalesce', ['get', 'render_height'], 0], 0, buildingLow, 80, buildingHigh],
                            'fill-extrusion-height': ['coalesce', ['get', 'render_height'], 8],
                            'fill-extrusion-base': ['coalesce', ['get', 'render_min_height'], 0],
                            'fill-extrusion-opacity': 0.85
                        } }, below);
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
            installControls();
            addLayers();
            document.dispatchEvent(new CustomEvent('travelmap:ready', {
                detail: { map: created.map, destination: context.destination, engine: created.name }
            }));
        }).catch(function () { showStatus(shell, 'Map unavailable.'); });
    }

    function newMaplibreDrawing(map) {
        var groups = new Map();
        function group(id) {
            if (!groups.has(id)) groups.set(id, { markers: [], line: null, clusters: null, control: null, popup: null });
            return groups.get(id);
        }
        function clearMarkers(entry) { entry.markers.forEach(function (marker) { marker.remove(); }); entry.markers = []; }
        function clearPopup(entry) {
            var popup = entry.popup;
            entry.popup = null; // the close handler ignores a popup the layer replaced or cleared
            if (popup) popup.remove();
        }
        function clearLine(entry, id) {
            if (map.getLayer('draw-line-' + id)) map.removeLayer('draw-line-' + id);
            if (map.getSource('draw-line-' + id)) map.removeSource('draw-line-' + id);
            entry.line = null;
        }
        function clearControl(entry) { if (entry.control) map.removeControl(entry.control); entry.control = null; }
        function clearClusters(entry, id) {
            if (!entry.clusters) return;
            var prefix = 'draw-clusters-' + id;
            ['label', 'single-count', 'single', 'count', 'group'].forEach(function (part) {
                if (map.getLayer(prefix + '-' + part)) map.removeLayer(prefix + '-' + part);
            });
            if (map.getSource(prefix)) map.removeSource(prefix);
            map.off('click', prefix + '-group', entry.clusters.groupClick);
            map.off('click', prefix + '-single', entry.clusters.singleClick);
            map.off('idle', entry.clusters.updateList);
            map.off('moveend', entry.clusters.updateList);
            if (entry.clusters.control) map.removeControl(entry.clusters.control);
            entry.clusters = null;
        }
        function clear(id) {
            var entry = groups.get(id);
            if (!entry) return;
            clearMarkers(entry);
            clearLine(entry, id);
            clearClusters(entry, id);
            clearControl(entry);
            clearPopup(entry);
            groups.delete(id);
        }
        function drawLine(id, entry) {
            if (!entry.line || map.getLayer('draw-line-' + id)) return;
            try {
                map.addSource('draw-line-' + id, { type: 'geojson', data: {
                    type: 'Feature', geometry: { type: 'LineString', coordinates: entry.line.coords }
                } });
                var paint = { 'line-color': entry.line.style.color || '#ff8a65', 'line-width': entry.line.style.width || 3 };
                if (entry.line.style.dash) paint['line-dasharray'] = entry.line.style.dash;
                map.addLayer({ id: 'draw-line-' + id, type: 'line', source: 'draw-line-' + id, paint: paint });
            } catch (error) {
                // The style is reloading: style.load calls restoreLines, which draws this line again.
                if (map.getSource('draw-line-' + id)) map.removeSource('draw-line-' + id);
            }
        }
        function drawClusters(id, entry) {
            if (!entry.clusters || map.getSource('draw-clusters-' + id)) return;
            var prefix = 'draw-clusters-' + id;
            var cluster = entry.clusters;
            map.addSource(prefix, { type: 'geojson', data: { type: 'FeatureCollection', features: cluster.features }, cluster: true, clusterRadius: 50 });
            map.addLayer({ id: prefix + '-group', type: 'circle', source: prefix, filter: ['has', 'point_count'],
                paint: { 'circle-color': '#9197a1', 'circle-radius': 22,
                    'circle-stroke-color': '#fff', 'circle-stroke-width': 2 } });
            map.addLayer({ id: prefix + '-count', type: 'symbol', source: prefix, filter: ['has', 'point_count'],
                layout: { 'text-field': ['get', 'point_count_abbreviated'], 'text-size': 13 },
                paint: { 'text-color': '#202a32' } });
            map.addLayer({ id: prefix + '-single', type: 'circle', source: prefix, filter: ['!', ['has', 'point_count']],
                paint: { 'circle-color': cluster.style.color || '#a582d4', 'circle-radius': 14, 'circle-opacity': cluster.style.pinElement ? 0 : 1,
                    'circle-stroke-color': '#fff', 'circle-stroke-width': 2 } });
            map.addLayer({ id: prefix + '-single-count', type: 'symbol', source: prefix, filter: ['!', ['has', 'point_count']],
                layout: { 'text-field': ['to-string', ['get', 'writer_count']], 'text-size': 12 },
                paint: { 'text-color': '#fff', 'text-opacity': cluster.style.pinElement ? 0 : 1 } });
            if (!cluster.style.pinElement && cluster.style.labelled && cluster.style.labelled.length) {
                map.addLayer({ id: prefix + '-label', type: 'symbol', source: prefix, minzoom: 15,
                    filter: ['all', ['!', ['has', 'point_count']], ['in', ['get', 'qid'], ['literal', cluster.style.labelled]]],
                    layout: { 'text-field': ['get', 'name'], 'text-size': 12, 'text-anchor': 'left', 'text-offset': [2, 0] },
                    paint: { 'text-color': '#fff', 'text-halo-color': '#202a32', 'text-halo-width': 2 } });
            }
        }
        return {
            clearAll: function () { Array.from(groups.keys()).forEach(clear); },
            restoreLines: function () { groups.forEach(function (entry, id) { drawLine(id, entry); drawClusters(id, entry); }); },
            draw: {
                clusters: function (id, features, style, onClick) {
                    var entry = group(id);
                    clearClusters(entry, id);
                    var prefix = 'draw-clusters-' + id;
                    var cluster = { features: features || [], style: style || {}, control: null };
                    cluster.groupClick = function (event) {
                        var feature = event.features && event.features[0];
                        if (!feature) return;
                        map.getSource(prefix).getClusterExpansionZoom(feature.properties.cluster_id).then(function (zoom) {
                            map.flyTo({ center: feature.geometry.coordinates, zoom: zoom });
                        }).catch(function () { /* cluster changed while zooming */ });
                    };
                    cluster.singleClick = function (event) {
                        var feature = event.features && event.features[0];
                        if (feature) onClick(feature);
                    };
                    cluster.updateList = function () {
                        if (!cluster.clusterButtons || !map.queryRenderedFeatures || !map.getLayer(prefix + '-group')) return;
                        if (cluster.style.pinElement) {
                            var visible = {};
                            map.queryRenderedFeatures({ layers: [prefix + '-single'] }).forEach(function (feature) { visible[feature.properties.qid] = true; });
                            (cluster.pins || []).forEach(function (pin) {
                                pin.element.hidden = !visible[pin.qid];
                                var label = pin.element.querySelector('.map-pin-label');
                                if (label) label.hidden = map.getZoom() < 15;
                            });
                        }
                        cluster.clusterButtons.replaceChildren();
                        var seen = {};
                        map.queryRenderedFeatures({ layers: [prefix + '-group'] }).forEach(function (feature) {
                            if (seen[feature.properties.cluster_id]) return;
                            seen[feature.properties.cluster_id] = true;
                            var button = document.createElement('button');
                            button.type = 'button';
                            button.textContent = 'Zoom to ' + feature.properties.point_count + ' writer places';
                            button.addEventListener('click', function () { cluster.groupClick({ features: [feature] }); });
                            cluster.clusterButtons.appendChild(button);
                        });
                    };
                    entry.clusters = cluster;
                    drawClusters(id, entry);
                    if (cluster.style.pinElement) {
                        cluster.pins = cluster.features.map(function (feature) {
                            var element = cluster.style.pinElement(feature);
                            element.hidden = true;
                            element.addEventListener('click', function () { onClick(feature, element); });
                            var marker = new window.maplibregl.Marker({ element: element, anchor: 'center' }).setLngLat(feature.geometry.coordinates).addTo(map);
                            entry.markers.push(marker);
                            return { qid: feature.properties.qid, element: element };
                        });
                    }
                    map.on('click', prefix + '-group', cluster.groupClick);
                    map.on('click', prefix + '-single', cluster.singleClick);
                    cluster.list = document.createElement('div');
                    cluster.list.className = 'travelmap-cluster-list';
                    cluster.clusterButtons = document.createElement('div');
                    cluster.list.appendChild(cluster.clusterButtons);
                    var placeButtons = document.createElement('div');
                    cluster.features.forEach(function (feature) {
                        var button = document.createElement('button');
                        button.type = 'button';
                        button.textContent = 'Open ' + feature.properties.name;
                        button.setAttribute('data-qid', feature.properties.qid);
                        button.addEventListener('click', function () {
                            map.jumpTo({ center: feature.geometry.coordinates, zoom: Math.max(map.getZoom(), 15) });
                            onClick(feature);
                        });
                        placeButtons.appendChild(button);
                    });
                    cluster.list.appendChild(placeButtons);
                    cluster.control = { onAdd: function () { return cluster.list; }, onRemove: function () { cluster.list.remove(); } };
                    map.addControl(cluster.control, 'bottom-left');
                    map.on('idle', cluster.updateList);
                    map.on('moveend', cluster.updateList);
                    cluster.updateList();
                },
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
                        entry.markers.push(new window.maplibregl.Marker({ element: item.element, opacityWhenCovered: '1' })
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
                popup: function (id, lngLat, element, options) {
                    var entry = group(id);
                    clearPopup(entry);
                    var popup = new window.maplibregl.Popup({ closeButton: false, closeOnClick: false, closeOnMove: false,
                        offset: 18, className: 'travelmap-popup', maxWidth: '260px' }).setDOMContent(element).setLngLat(lngLat);
                    // A click on a pin also reaches the map, so MapLibre's own closeOnClick would close the popup
                    // the pin just opened.
                    var closeOnMapClick = function (event) {
                        var target = event.originalEvent && event.originalEvent.target;
                        if (!target || !target.closest || !target.closest('.maplibregl-marker')) popup.remove();
                    };
                    map.on('click', closeOnMapClick);
                    entry.popup = popup;
                    popup.on('close', function () {
                        map.off('click', closeOnMapClick);
                        if (entry.popup !== popup) return;
                        entry.popup = null;
                        if (options && options.onClose) options.onClose();
                    });
                    popup.addTo(map);
                },
                clear: clear,
                fit: function (bounds, options) {
                    map.fitBounds(bounds, { padding: options && options.padding || 0, maxZoom: options && options.maxZoom,
                        animate: options && options.animate });
                },
                flyTo: function (lngLat, zoom) { map.flyTo({ center: lngLat, zoom: zoom }); }
            }
        };
    }

    // While the server is still loading places it answers with the pins so far and loading: true,
    // so draw them and ask again; the last answer carries the writer matches.
    function loadPlaces(currentRequest) {
        return fetch(shell.dataset.mapUrl, { headers: { Accept: 'application/geo+json' } }).then(function (response) {
            if (!response.ok) throw new Error('Places unavailable');
            return response.json();
        }).then(function (data) {
            if (currentRequest !== requestNumber) return;
            var loading = data.loading === true;
            if (engine && context.features.length) removeLayers();
            context.features = Array.isArray(data.features) ? data.features : [];
            context.itineraries = Array.isArray(data.itineraries) ? data.itineraries : [];
            var fullMap = shell.querySelector('.map-action');
            var city = context.features.find(function (feature) {
                return feature.properties.photo_url && feature.properties.name.toLowerCase() === String(shell.dataset.mapDestination).replace(/-[^-]*$/, '').replace(/-/g, ' ');
            });
            if (fullMap && city && !fullMap.querySelector('.map-action-photo')) {
                var thumbnail = document.createElement('img'); thumbnail.className = 'map-action-photo';
                thumbnail.src = city.properties.photo_url; thumbnail.alt = ''; thumbnail.loading = 'lazy';
                var icon = fullMap.querySelector('.bi-map');
                if (icon) {
                    thumbnail.addEventListener('error', function () { thumbnail.replaceWith(icon); });
                    icon.replaceWith(thumbnail);
                }
            }
            if (engine) addLayers();
            document.dispatchEvent(new CustomEvent('travelmap:places-loaded', { detail: { features: context.features } }));
            showStatus(shell, !engine ? 'Map unavailable.' :
                loading ? 'Loading more places\u2026' :
                data.complete === false ? 'Some map sources are unavailable.' : '');
            if (loading) setTimeout(function () { if (currentRequest === requestNumber) loadPlaces(currentRequest); }, pollDelay);
        }).catch(function () {
            if (currentRequest !== requestNumber) return;
            if (engine) addLayers();
            showStatus(shell, engine ? 'Places are unavailable. The map is still available.' : 'Map and places are unavailable.');
        });
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
