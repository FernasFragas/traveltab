(function () {
    'use strict';

    // Draws the itinerary's stops as green numbered pins, with a day toggle.
    // Stops come from data-plan-day / data-plan-stop in the plan markup (no endpoint).
    if (!window.TravelMap || window.TravelMap.planLayerRegistered) return;
    window.TravelMap.planLayerRegistered = true;

    var days = [];
    var daysKey = '[]';
    var activeDay = 0; // 0 shows every day
    var drawAPI = null;

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
        var mapElement = document.querySelector('[data-writers-map]');
        if (mapElement) mapElement.scrollIntoView({ behavior: 'smooth', block: 'start' });
    });
    document.body.addEventListener('htmx:afterSwap', report);
    document.body.addEventListener('htmx:historyRestore', report);
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
                stops.push({ name: stopElement.dataset.name || '', lat: lat, lon: lon });
            });
            result.push({ number: Number(dayElement.dataset.planDay), stops: stops });
        });
        return result;
    }

    function add(map, context) {
        drawAPI = context.draw;
        draw();
    }

    function remove() {
        clear();
        drawAPI = null;
    }

    function clear() {
        if (drawAPI) drawAPI.clear('plan');
    }

    function draw() {
        clear();
        if (!drawAPI) return;
        var shown = days.filter(function (day) { return day.stops.length; });
        if (!shown.length) return;
        var pins = [];
        shown.forEach(function (day) {
            if (activeDay && day.number !== activeDay) return;
            day.stops.forEach(function (stop, index) {
                var pin = document.createElement('button');
                pin.type = 'button';
                pin.className = 'plan-pin';
                pin.textContent = String(index + 1);
                pin.style.cssText = 'display:flex;align-items:center;justify-content:center;width:26px;height:26px;' +
                    'border-radius:50%;color:#fff;border:2px solid #fff;font:600 13px/1 sans-serif;' +
                    'box-shadow:0 1px 4px rgba(0,0,0,.4);cursor:pointer';
                pin.style.background = '#2e7d32';
                pins.push({
                    lngLat: [stop.lon, stop.lat],
                    element: pin,
                    title: 'Day ' + day.number + ': ' + stop.name,
                    z: 30
                });
            });
        });
        drawAPI.markers('plan', pins);
        if (shown.length > 1) addToggle(shown);
    }

    function addToggle(shown) {
        var box = document.createElement('div');
        box.className = 'plan-day-toggle';
        box.style.background = '#fff';
        box.setAttribute('role', 'group');
        box.setAttribute('aria-label', 'Show day plan pins');
        [{ label: 'All', number: 0 }].concat(shown.map(function (day) {
            return { label: 'Day ' + day.number, number: day.number };
        })).forEach(function (item) {
            var button = document.createElement('button');
            button.type = 'button';
            button.textContent = item.label;
            button.style.cssText = 'display:block;width:100%;padding:4px 10px;border:0;background:' +
                (item.number === activeDay ? '#c8e6c9' : '#fff') + ';color:#1b5e20;cursor:pointer;font:600 12px sans-serif';
            button.setAttribute('aria-pressed', String(item.number === activeDay));
            button.addEventListener('click', function () {
                activeDay = item.number;
                draw();
            });
            box.appendChild(button);
        });
        drawAPI.control('plan', box, 'top-right');
    }
})();
