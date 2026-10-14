(function () {
    'use strict';

    // Draws the itinerary's stops as turquoise numbered pins, with a day toggle.
    // Stops come from data-plan-day / data-plan-stop in the plan markup (no endpoint).
    if (!window.TravelMap || window.TravelMap.planLayerRegistered) return;
    window.TravelMap.planLayerRegistered = true;

    var days = [];
    var daysKey = '[]';
    var activeDay = 0; // 0 shows every day
    var drawAPI = null;
    var windowContext = null;
    var openPin = null;
    var routeIds = [];

    window.TravelMap.registerLayer('plan', { add: add, remove: remove });
    document.addEventListener('travelmap:plan-updated', function (event) {
        var next = (event.detail && event.detail.days) || [];
        var nextKey = JSON.stringify(next);
        if (nextKey === daysKey) return;
        days = next;
        daysKey = nextKey;
        activeDay = 0;
        draw();
    });
    // "View on map": press the Your day plan chip (7f keeps its own state) and scroll to the map.
    document.addEventListener('click', function (event) {
        if (!event.target.closest || !event.target.closest('[data-view-on-map]')) return;
        var chip = document.querySelector('.writers-map-layer-plan');
        if (chip && chip.getAttribute('aria-pressed') !== 'true') chip.click();
        window.location.hash = '#overview';
        var mapElement = document.querySelector('[data-writers-map]');
        if (mapElement) mapElement.scrollIntoView({ behavior: 'smooth', block: 'start' });
    });
    document.body.addEventListener('htmx:afterSwap', report);
    document.body.addEventListener('htmx:historyRestore', report);
    document.addEventListener('keydown', function (event) {
        if (event.key !== 'Escape' || !openPin) return;
        var pin = openPin;
        drawAPI.clear('pin-popup');
        openPin = null;
        pin.focus({ preventScroll: true });
    }, true);
    report();

    function report() {
        if (!document.querySelector('[data-writers-map]')) return;
        window.TravelMap.updatePlan(readDays());
    }

    function readDays() {
        var result = [];
        document.querySelectorAll('[data-plan-day]').forEach(function (dayElement) {
            var stops = [];
            dayElement.querySelectorAll('[data-plan-stop]').forEach(function (stopElement) {
                var lat = Number(stopElement.dataset.lat);
                var lon = Number(stopElement.dataset.lon);
                if (!Number.isFinite(lat) || !Number.isFinite(lon) || (lat === 0 && lon === 0) ||
                    Math.abs(lat) > 90 || Math.abs(lon) > 180) return;
                var distance = stopElement.querySelector('[data-stop-distance]');
                var photo = stopElement.querySelector('.itinerary-stop-photo');
                stops.push({ name: stopElement.dataset.name || '', lat: lat, lon: lon,
                    qid: stopElement.dataset.qid || '', kind: stopElement.dataset.kind || '',
                    description: stopElement.dataset.description || '', distance: distance ? distance.textContent.trim() : '',
                    photoURL: photo ? photo.src : undefined });
            });
            result.push({ number: Number(dayElement.dataset.planDay), stops: stops });
        });
        return result;
    }

    function add(map, context) {
        windowContext = context;
        drawAPI = context.draw;
        draw();
    }

    function remove() {
        clear();
        drawAPI = null;
        windowContext = null;
    }

    function clear() {
        if (!drawAPI) return;
        drawAPI.clear('pin-popup');
        openPin = null;
        routeIds.forEach(function (id) { drawAPI.clear(id); });
        routeIds = [];
        drawAPI.clear('plan');
    }

    function draw() {
        clear();
        if (!drawAPI) return;
        var shown = days.filter(function (day) { return day.stops.length; });
        if (!shown.length) return;
        var pins = [];
        shown.forEach(function (day) {
            if (activeDay && day.number !== activeDay) return;
            if (day.stops.length > 1 && drawAPI.line) {
                var routeId = 'plan-route-' + day.number;
                drawAPI.line(routeId, day.stops.map(function (stop) { return [stop.lon, stop.lat]; }),
                    { color: '#a56824', width: 3, dash: [1, 2] });
                routeIds.push(routeId);
            }
            day.stops.forEach(function (stop, index) {
                var pin = document.createElement('button');
                pin.type = 'button';
                pin.className = 'plan-pin';
                pin.textContent = String(index + 1);
                pin.style.cssText = 'display:flex;align-items:center;justify-content:center;' +
                    'color:#fff;border:2px solid #fff;font:600 13px/1 sans-serif;' +
                    'box-shadow:0 1px 4px rgba(0,0,0,.4);cursor:pointer';
                pin.style.background = '#a56824';
                pin.setAttribute('aria-label', 'Day ' + day.number + ', stop ' + (index + 1) + ': ' + stop.name);
                if (stop.photoURL) {
                    var photo = document.createElement('img');
                    photo.className = 'plan-pin-photo';
                    photo.src = stop.photoURL;
                    photo.alt = '';
                    photo.loading = 'lazy';
                    pin.appendChild(photo);
                }
                pins.push({
                    lngLat: [stop.lon, stop.lat],
                    element: pin,
                    title: 'Day ' + day.number + ': ' + stop.name,
                    onClick: function () { showPopup(stop, day.number, index + 1, pin); },
                    z: 30
                });
            });
        });
        drawAPI.markers('plan', pins);
        if (shown.length > 1) addToggle(shown);
    }

    function showPopup(stop, dayNumber, stopNumber, pin) {
        var box = document.createElement('div');
        box.className = 'map-popup plan-popup';
        box.setAttribute('role', 'dialog');
        box.setAttribute('aria-label', stop.name);
        var name = document.createElement('strong');
        name.className = 'map-popup-name';
        name.textContent = stop.name;
        box.appendChild(name);
        var facts = document.createElement('ul');
        facts.className = 'plan-popup-facts';
        facts.appendChild(fact('bi-calendar3', 'Day ' + dayNumber));
        facts.appendChild(fact('bi-geo-alt-fill', 'Stop ' + stopNumber));
        if (stop.distance) {
            var walk = fact('bi-person-walking', stop.distance.replace(/\s*from previous$/, ''));
            var from = document.createElement('small');
            from.textContent = 'from previous';
            walk.appendChild(from);
            facts.appendChild(walk);
        }
        box.appendChild(facts);
        var known = stop.qid && windowContext.features && windowContext.features.some(function (feature) {
            return feature.properties && feature.properties.qid === stop.qid;
        });
        if (known) {
            var details = document.createElement('button');
            details.type = 'button';
            details.className = 'map-popup-details';
            details.textContent = 'View details';
            details.appendChild(icon('bi-arrow-right'));
            details.addEventListener('click', function () {
                drawAPI.clear('pin-popup');
                openPin = null;
                windowContext.selectPlace(stop.qid);
            });
            box.appendChild(details);
        }
        drawAPI.popup('pin-popup', [stop.lon, stop.lat], box, { onClose: function () { openPin = null; } });
        openPin = pin;
    }

    function addToggle(shown) {
        var box = document.createElement('div');
        box.className = 'plan-day-toggle';
        box.setAttribute('role', 'group');
        box.setAttribute('aria-label', 'Show day plan pins');
        [{ label: 'All', number: 0 }].concat(shown.map(function (day) {
            return { label: 'Day ' + day.number, number: day.number };
        })).forEach(function (item) {
            var button = document.createElement('button');
            button.type = 'button';
            button.textContent = item.label;
            button.setAttribute('aria-pressed', String(item.number === activeDay));
            button.addEventListener('click', function () {
                activeDay = item.number;
                draw();
            });
            box.appendChild(button);
        });
        drawAPI.control('plan', box, 'top-left');
    }

    function fact(iconClass, label) {
        var item = document.createElement('li');
        item.appendChild(icon(iconClass));
        var value = document.createElement('span');
        value.textContent = label;
        item.appendChild(value);
        return item;
    }

    function icon(iconClass) {
        var element = document.createElement('i');
        element.className = 'bi ' + iconClass;
        element.setAttribute('aria-hidden', 'true');
        return element;
    }
})();
