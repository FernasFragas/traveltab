(function () {
    'use strict';

    // "Save to plan": saved QIDs live in localStorage only. Saving makes no
    // server call; the QIDs ride along on the next /plan request as `include`.
    var storageKey = 'traveltab.savedQids';
    var saved = read();
    var planning = false;
    var toastTimer;
    var undoQid;
    var places = readPlaces();
    var pendingPlace;
    document.addEventListener('travelmap:places-loaded', function (event) {
        var shell = document.querySelector('[data-writers-map]');
        (event.detail.features || []).forEach(function (feature) {
            places[feature.properties.qid] = { name: feature.properties.name || (places[feature.properties.qid] && places[feature.properties.qid].name), destination: shell && shell.dataset.mapDestination, lat: shell && shell.dataset.mapLat, lon: shell && shell.dataset.mapLon };
        });
        try { window.localStorage.setItem('traveltab.savedPlaces', JSON.stringify(places)); } catch (error) { /* QID storage remains the source of truth */ }
        if (pendingPlace && (event.detail.features || []).some(function (feature) { return feature.properties.qid === pendingPlace; })) { window.TravelMap.selectPlace(pendingPlace); pendingPlace = null; }
        renderList();
    });
    document.body.addEventListener('keydown', function (event) {
        if (event.key !== 'Escape') return;
        var list = document.querySelector('#saved-places-list');
        var toggle = document.querySelector('.tt-saved-toggle');
        if (list && !list.hidden) { event.preventDefault(); list.hidden = true; toggle.setAttribute('aria-expanded', 'false'); toggle.focus(); }
    });

    document.body.addEventListener('click', function (event) {
        if (saved === null) return;
        var toggle = event.target.closest('.tt-saved-toggle');
        var list = document.querySelector('#saved-places-list');
        if (toggle && list) { list.hidden = !list.hidden; toggle.setAttribute('aria-expanded', String(!list.hidden)); renderList(); return; }
        var item = event.target.closest('[data-saved-place]');
        if (item) {
            var selected = item.getAttribute('data-saved-place');
            var info = places[selected];
            var shell = document.querySelector('[data-writers-map]');
            window.location.hash = '#overview';
            if (info && info.destination && shell && info.destination !== shell.dataset.mapDestination) {
                pendingPlace = selected;
                var input = document.querySelector('#city_name');
                var country = document.querySelector('#country_code');
                var destination = info.destination.split('-');
                country.value = destination.pop(); input.value = destination.join(' ');
                document.querySelector('#place_lat').value = info.lat;
                document.querySelector('#place_lon').value = info.lon;
                document.querySelector('#destination-search').requestSubmit();
            } else window.TravelMap.selectPlace(selected);
            list.hidden = true; document.querySelector('.tt-saved-toggle').setAttribute('aria-expanded', 'false'); return;
        }
        var undo = event.target.closest('[data-saved-undo]');
        if (undo) { saved = saved.filter(function (qid) { return qid !== undoQid; }); write(); render(); hideToast(); return; }
        var clear = event.target.closest('[data-saved-clear]');
        if (clear) {
            event.preventDefault();
            saved = [];
            write();
            render();
            return;
        }
        var add = event.target.closest('[data-add-qid]');
        if (add) {
            var addQid = add.dataset.addQid;
            if (saved.indexOf(addQid) < 0) saved.push(addQid);
            write();
            render();
            var form = document.querySelector('.trip-form');
            if (form) {
                planning = true;
                form.requestSubmit();
            } else window.location.hash = '#itinerary';
            return;
        }
        var button = event.target.closest('[data-save-qid]');
        if (!button) return;
        var qid = button.dataset.saveQid;
        var index = saved.indexOf(qid);
        if (index >= 0) saved.splice(index, 1);
        else saved.push(qid);
        write();
        render();
        if (index < 0 && saved !== null) showToast(qid);
    });
    document.body.addEventListener('htmx:configRequest', function (event) {
        if (saved === null || saved.length === 0 || event.detail.path.split('?')[0] !== '/plan') return;
        event.detail.parameters.include = saved.slice();
    });
    document.body.addEventListener('htmx:afterSwap', function (event) {
        render();
        if (planning && event.detail.target.id === 'trip-card') {
            planning = false;
            window.location.hash = '#itinerary';
        }
    });
    document.body.addEventListener('htmx:historyRestore', render);
    // The map core renders the panel in JS; wait a tick so its buttons exist.
    document.addEventListener('travelmap:ready', defer);
    document.addEventListener('travelmap:place-selected', defer);
    render();

    function defer() {
        setTimeout(render, 0);
    }

    function renderList() {
        var list = document.querySelector('#saved-places-list');
        if (!list) return;
        var items = (saved || []).map(function (qid) {
            var item = document.createElement('button'); item.type = 'button';
            item.setAttribute('data-saved-place', qid); item.textContent = places[qid] ? places[qid].name : ('Saved place ' + qid + ' · destination unavailable');
            item.disabled = !places[qid];
            return item;
        });
        if (!items.length) { var empty = document.createElement('p'); empty.textContent = 'No saved places yet'; items.push(empty); }
        list.replaceChildren.apply(list, items);
    }

    function hideToast() {
        clearTimeout(toastTimer);
        var toast = document.querySelector('.writers-map-toast');
        if (toast) toast.hidden = true;
    }

    function showToast(qid) {
        var toast = document.querySelector('.writers-map-toast');
        if (!toast) return;
        clearTimeout(toastTimer); undoQid = qid;
        var text = document.createElement('span'); text.textContent = 'Added to your saved places · ';
        var undo = document.createElement('button'); undo.type = 'button'; undo.textContent = 'Undo'; undo.setAttribute('data-saved-undo', '');
        toast.replaceChildren(text, undo); toast.setAttribute('aria-live', 'polite'); toast.hidden = false;
        toastTimer = setTimeout(hideToast, 5000);
    }

    function render() {
        renderList();
        var addButtons = document.querySelectorAll('[data-add-qid]');
        for (var j = 0; j < addButtons.length; j++) addButtons[j].hidden = saved === null;
        var buttons = document.querySelectorAll('[data-save-qid]');
        for (var i = 0; i < buttons.length; i++) paint(buttons[i]);
        var slot = document.querySelector('[data-map-slot="saved"]');
        if (!slot) return;
        if (saved === null || saved.length === 0) {
            slot.replaceChildren();
            return;
        }
        var text = document.createElement('span');
        text.textContent = saved.length + (saved.length === 1 ? ' saved place will be included ' : ' saved places will be included ');
        var clear = document.createElement('a');
        clear.setAttribute('href', '#');
        clear.setAttribute('data-saved-clear', '');
        clear.textContent = 'Clear';
        var line = document.createElement('p');
        line.className = 'writers-map-saved';
        line.append(text, clear);
        slot.replaceChildren(line);
    }

    function paint(button) {
        var label = button.querySelector('[data-save-label]') || button;
        if (!button.dataset.saveLabel) button.dataset.saveLabel = label.textContent;
        if (saved === null) {
            button.hidden = true;
            return;
        }
        var on = saved.indexOf(button.dataset.saveQid) >= 0;
        button.hidden = false;
        if (button.className !== 'map-panel-bookmark') label.textContent = on ? 'Saved' : button.dataset.saveLabel;
        else button.setAttribute('aria-label', on ? 'Remove from saved places' : 'Save to visit');
        button.setAttribute('aria-pressed', String(on));
    }

    // Returns null when storage is blocked, which hides the feature.
    function readPlaces() {
        try { var value = JSON.parse(window.localStorage.getItem('traveltab.savedPlaces')); return value && typeof value === 'object' ? value : {}; }
        catch (error) { return {}; }
    }

    function read() {
        try {
            var storage = window.localStorage;
            var raw = storage.getItem(storageKey);
            storage.setItem(storageKey, raw === null ? '[]' : raw);
            var list = raw === null ? [] : JSON.parse(raw);
            return Array.isArray(list) ? list : [];
        } catch (error) {
            return probe();
        }
    }

    function write() {
        try {
            window.localStorage.setItem(storageKey, JSON.stringify(saved));
        } catch (error) {
            saved = null;
            render();
        }
    }

    // Distinguishes corrupt JSON (start empty) from blocked storage (null).
    function probe() {
        try {
            window.localStorage.setItem(storageKey, '[]');
            return [];
        } catch (error) {
            return null;
        }
    }
})();
