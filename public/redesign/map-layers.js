/* Legend chips: toggle the Writers' Map layers (notable, writers, plan, itineraries). */
(function () {
    'use strict';

    var STORAGE_KEY = 'travelmap.layers';
    var LAYERS = ['notable', 'writers', 'plan', 'itineraries'];
    var state = readState();
    var hasPlan = false;

    document.addEventListener('travelmap:ready', sync);
    document.addEventListener('htmx:afterSwap', sync);
    document.addEventListener('travelmap:plan-updated', function (event) {
        var days = event && event.detail && event.detail.days;
        hasPlan = Array.isArray(days) && days.length > 0;
        sync();
    });
    if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', sync);
    else sync();

    function sync() {
        Array.prototype.forEach.call(document.querySelectorAll('[data-map-slot="layers"]'), function (slot) {
            upgrade(slot);
            Array.prototype.forEach.call(slot.querySelectorAll('.writers-map-layer'), refresh);
        });
    }

    function upgrade(slot) {
        Array.prototype.forEach.call(slot.querySelectorAll('.writers-map-layer'), function (chip) {
            if (chip.tagName !== 'SPAN') return;
            var button = document.createElement('button');
            button.className = chip.className;
            button.textContent = chip.textContent;
            button.setAttribute('type', 'button');
            button.setAttribute('aria-pressed', 'true');
            button.addEventListener('click', function () { toggle(button); });
            slot.replaceChild(button, chip);
        });
    }

    function refresh(chip) {
        var name = layerName(chip);
        if (!name) return;
        var registered = apply(name);
        chip.setAttribute('aria-pressed', isOn(name) ? 'true' : 'false');
        if (name === 'plan') chip.disabled = !hasPlan;
        if (name === 'itineraries') chip.hidden = !registered;
    }

    function toggle(chip) {
        var name = layerName(chip);
        if (!name || chip.disabled) return;
        state[name] = !isOn(name);
        writeState();
        refresh(chip);
    }

    function apply(name) {
        try {
            window.TravelMap.setLayerEnabled(name, isOn(name));
            return true;
        } catch (e) {
            return false;
        }
    }

    function isOn(name) {
        return state[name] !== false;
    }

    function layerName(chip) {
        for (var i = 0; i < LAYERS.length; i++) {
            if (String(chip.className).indexOf('writers-map-layer-' + LAYERS[i]) !== -1) return LAYERS[i];
        }
        return '';
    }

    function readState() {
        try {
            var parsed = JSON.parse(sessionStorage.getItem(STORAGE_KEY));
            return parsed && typeof parsed === 'object' ? parsed : {};
        } catch (e) {
            return {};
        }
    }

    function writeState() {
        try {
            sessionStorage.setItem(STORAGE_KEY, JSON.stringify(state));
        } catch (e) { /* storage blocked: state stays in memory */ }
    }
})();
