/* Legend chips: toggle the Writers' Map layers (notable, writers, plan, itineraries). */
(function () {
    'use strict';

    var STORAGE_KEY = 'travelmap.layers';
    var LAYERS = ['notable', 'writers', 'plan', 'itineraries'];
    var state = readState();
    var hasPlan = false;
    var menus = [];

    window.TravelMapLayers = { createMenu: createMenu };

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
        menus.forEach(function (menu) {
            Array.prototype.forEach.call(menu.querySelectorAll('.writers-map-layer'), refresh);
        });
    }

    function createMenu() {
        var menu = document.createElement('details');
        menu.className = 'travelmap-layers-menu';
        var summary = document.createElement('summary');
        summary.textContent = 'Layers';
        menu.appendChild(summary);
        var labels = ['Notable places', "Writers' pins", 'Your plan', "Writers' itineraries"];
        LAYERS.forEach(function (name, index) {
            var button = document.createElement('button');
            button.type = 'button';
            button.className = 'writers-map-layer writers-map-layer-' + name;
            button.textContent = labels[index];
            button.addEventListener('click', function () { toggle(button); });
            menu.appendChild(button);
        });
        menus = [menu];
        Array.prototype.forEach.call(menu.querySelectorAll('.writers-map-layer'), refresh);
        return menu;
    }

    function upgrade(slot) {
        Array.prototype.forEach.call(slot.querySelectorAll('.writers-map-layer'), function (chip) {
            if (chip.tagName !== 'SPAN') return;
            var button = document.createElement('button');
            button.className = chip.className;
            button.textContent = chip.textContent;
            button.setAttribute('aria-label', chip.textContent);
            button.setAttribute('type', 'button');
            button.setAttribute('aria-pressed', 'true');
            button.addEventListener('click', function () { toggle(button); });
            slot.replaceChild(button, chip);
        });
    }

    function refresh(chip) {
        var name = layerName(chip);
        if (!name) return;
        var labels = { notable: ['Notable places', 'Notable'], writers: ["Writers' pins", 'Writers'], itineraries: ["Writers' itineraries", 'Itineraries'], plan: ['Your plan', 'Your plan'] };
        chip.setAttribute('aria-label', labels[name][0]);
        chip.textContent = window.innerWidth <= 600 ? labels[name][1] : labels[name][0];
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
        Array.prototype.forEach.call(document.querySelectorAll('[data-map-slot="layers"]'), function (slot) {
            Array.prototype.forEach.call(slot.querySelectorAll('.writers-map-layer'), function (peer) {
                if (layerName(peer) === name) peer.setAttribute('aria-pressed', isOn(name) ? 'true' : 'false');
            });
        });
        menus.forEach(function (menu) {
            Array.prototype.forEach.call(menu.querySelectorAll('.writers-map-layer'), function (peer) {
                if (layerName(peer) === name) peer.setAttribute('aria-pressed', isOn(name) ? 'true' : 'false');
            });
        });
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
