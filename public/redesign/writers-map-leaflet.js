(function () {
    'use strict';

    // Fallback engine for the writers' map: Leaflet with the raster basemap (EOX imagery, OSM on a
    // tile error), zoom capped at 18. writers-map.js calls create() when WebGL is missing or the
    // MapLibre style, tiles or terrain fail. Drawing follows docs/writers-map-contract.md.
    var leafletCSS = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.css';
    var leafletJS = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.js';
    var imageryTiles = 'https://tiles.maps.eox.at/wmts/1.0.0/s2cloudless_3857/default/GoogleMapsCompatible/{z}/{y}/{x}.jpg';
    var imageryAttribution = 'EOxCloudless <a href="https://cloudless.eox.at">cloudless.eox.at</a> by <a href="https://eox.at">EOX IT Services GmbH</a> (Contains modified Copernicus Sentinel data 2016 &amp; 2017), <a href="https://creativecommons.org/licenses/by/4.0/">CC BY 4.0</a>';
    var osmTiles = 'https://tile.openstreetmap.org/{z}/{x}/{y}.png';
    var osmAttribution = '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors';
    var maxZoom = 18;
    var startZoom = 15; // about a 200 m scale bar at mid latitudes
    var leafletPromise = null;

    window.TravelMapEngines = window.TravelMapEngines || {};
    window.TravelMapEngines.leaflet = { create: create };

    // Resolves to { name, map, draw, reset(lat, lon, shell), clearAll(), destroy() }.
    function create(canvas, shell, latitude, longitude) {
        return loadLeaflet().then(function () {
            var map = window.L.map(canvas, { scrollWheelZoom: false, maxZoom: maxZoom, zoomControl: false }).setView([latitude, longitude], startZoom);
            window.L.control.scale({ imperial: false, position: 'bottomright' }).addTo(map);
            var tileLayer = null;
            var drawing = newDrawing(map);
            setTiles();
            return {
                name: 'leaflet',
                map: map,
                draw: drawing.draw,
                clearAll: drawing.clearAll,
                reset: function (nextLatitude, nextLongitude, nextShell) {
                    shell = nextShell;
                    drawing.clearAll();
                    map.invalidateSize();
                    map.setView([nextLatitude, nextLongitude], startZoom);
                    if (tileLayer) map.removeLayer(tileLayer);
                    setTiles();
                },
                destroy: function () {
                    drawing.clearAll();
                    map.remove();
                }
            };

            function setTiles() {
                var tileURL = (shell.dataset.mapTileUrl || imageryTiles).replace(/%7B([zxy])%7D/gi, '{$1}');
                var layer = window.L.tileLayer(tileURL, {
                    maxZoom: maxZoom,
                    attribution: shell.dataset.mapAttribution || (tileURL === osmTiles ? osmAttribution : imageryAttribution)
                });
                tileLayer = layer;
                if (tileURL !== osmTiles) layer.on('tileerror', function () {
                    if (tileLayer !== layer) return;
                    map.removeLayer(layer);
                    tileLayer = window.L.tileLayer(osmTiles, { maxZoom: maxZoom, attribution: osmAttribution }).addTo(map);
                });
                layer.addTo(map);
            }
        });
    }

    // The drawing API of the contract, on Leaflet. Calls with the same id replace that group.
    function newDrawing(map) {
        var groups = new Map();
        var positions = { 'top-left': 'topleft', 'top-right': 'topright', 'bottom-left': 'bottomleft', 'bottom-right': 'bottomright' };

        function group(id) {
            if (!groups.has(id)) groups.set(id, { markers: [], line: null, control: null, popup: null, labelUpdate: null });
            return groups.get(id);
        }

        function clearMarkers(entry) {
            entry.markers.forEach(function (marker) { map.removeLayer(marker); });
            entry.markers = [];
        }

        function clearLine(entry) {
            if (entry.line) map.removeLayer(entry.line);
            entry.line = null;
        }

        function clearControl(entry) {
            if (entry.control) entry.control.remove();
            entry.control = null;
        }

        function clearPopup(entry) {
            var popup = entry.popup;
            entry.popup = null; // the remove handler ignores a popup the layer replaced or cleared
            if (popup) map.removeLayer(popup);
        }

        function clear(id) {
            var entry = groups.get(id);
            if (!entry) return;
            clearMarkers(entry);
            if (entry.labelUpdate) map.off('zoomend', entry.labelUpdate);
            clearLine(entry);
            clearControl(entry);
            clearPopup(entry);
            groups.delete(id);
        }

        return {
            clearAll: function () { Array.from(groups.keys()).forEach(clear); },
            draw: {
                clusters: function (id, features, style, onClick) {
                    var entry = group(id);
                    if (entry.labelUpdate) map.off('zoomend', entry.labelUpdate);
                    var labels = [];
                    this.markers(id, (features || []).map(function (feature) {
                        var pin = document.createElement('button');
                        pin.type = 'button';
                        pin.className = 'map-pin map-pin-writer';
                        pin.setAttribute('data-qid', feature.properties.qid);
                        var quill = document.createElement('i'); quill.className = 'bi bi-feather'; quill.setAttribute('aria-hidden', 'true'); pin.appendChild(quill);
                        if (style && style.labelled && style.labelled.indexOf(feature.properties.qid) !== -1) {
                            var label = document.createElement('span');
                            label.className = 'map-pin-label';
                            label.textContent = feature.properties.name;
                            label.setAttribute('aria-hidden', 'true');
                            pin.appendChild(label);
                            labels.push(label);
                        }
                        return { lngLat: feature.geometry.coordinates, element: pin, title: feature.properties.name,
                            onClick: function () { onClick(feature, pin); }, z: 1000 };
                    }));
                    entry.labelUpdate = function () {
                        labels.forEach(function (label) { label.hidden = map.getZoom() < 15; });
                    };
                    map.on('zoomend', entry.labelUpdate);
                    entry.labelUpdate();
                },
                markers: function (id, items) {
                    var entry = group(id);
                    clearMarkers(entry);
                    (items || []).forEach(function (item) {
                        var anchor = document.createElement('div');
                        anchor.appendChild(item.element);
                        label(item);
                        if (item.onClick) item.element.addEventListener('click', item.onClick);
                        entry.markers.push(window.L.marker([item.lngLat[1], item.lngLat[0]], {
                            icon: window.L.divIcon({ html: anchor, className: 'travelmap-marker', iconSize: [0, 0] }),
                            keyboard: false,
                            zIndexOffset: item.z || 0
                        }).addTo(map));
                    });
                },
                line: function (id, coords, style) {
                    var entry = group(id);
                    clearLine(entry);
                    style = style || {};
                    entry.line = window.L.polyline(coords.map(function (point) { return [point[1], point[0]]; }), {
                        color: style.color, weight: style.width, dashArray: style.dash ? style.dash.join(' ') : null
                    }).addTo(map);
                },
                control: function (id, element, position) {
                    var entry = group(id);
                    clearControl(entry);
                    var control = window.L.control({ position: positions[position] || 'topright' });
                    control.onAdd = function () { return element; };
                    if (window.L.DomEvent) window.L.DomEvent.disableClickPropagation(element);
                    entry.control = control.addTo(map);
                },
                popup: function (id, lngLat, element, options) {
                    var entry = group(id);
                    clearPopup(entry);
                    // A click on a pin also reaches the map, so Leaflet's own closeOnClick would close the
                    // popup the pin just opened.
                    var popup = window.L.popup({ closeButton: false, autoClose: false, closeOnClick: false, offset: [0, -18],
                        className: 'travelmap-popup', maxWidth: 260, autoPanPaddingTopLeft: [56, 8], autoPanPaddingBottomRight: [8, 8] }).setLatLng([lngLat[1], lngLat[0]]).setContent(element);
                    var closeOnMapClick = function (event) {
                        var target = event && event.originalEvent && event.originalEvent.target;
                        if (!target || !target.closest || !target.closest('.leaflet-marker-icon')) map.removeLayer(popup);
                    };
                    map.on('click', closeOnMapClick);
                    entry.popup = popup;
                    popup.on('remove', function () {
                        map.off('click', closeOnMapClick);
                        if (entry.popup !== popup) return;
                        entry.popup = null;
                        if (options && options.onClose) options.onClose();
                    });
                    popup.openOn(map);
                },
                clear: clear,
                fit: function (bounds, options) {
                    var padding = options && options.padding || 0;
                    map.fitBounds([[bounds[0][1], bounds[0][0]], [bounds[1][1], bounds[1][0]]], { padding: [padding, padding], maxZoom: options && options.maxZoom,
                        animate: options && options.animate });
                },
                flyTo: function (lngLat, zoom) {
                    map.flyTo([lngLat[1], lngLat[0]], zoom === undefined ? map.getZoom() : zoom);
                }
            }
        };
    }

    function label(item) {
        if (!item.title) return;
        item.element.setAttribute('title', item.title);
        item.element.setAttribute('aria-label', item.title);
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
        leafletPromise.catch(function () { leafletPromise = null; });
        return leafletPromise;
    }
})();
