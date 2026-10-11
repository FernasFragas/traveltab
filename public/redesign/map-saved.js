(function () {
    'use strict';

    // "Save to plan": saved QIDs live in localStorage only. Saving makes no
    // server call; the QIDs ride along on the next /plan request as `include`.
    var storageKey = 'traveltab.savedQids';
    var saved = read();

    document.body.addEventListener('click', function (event) {
        if (saved === null) return;
        var clear = event.target.closest('[data-saved-clear]');
        if (clear) {
            event.preventDefault();
            saved = [];
            write();
            render();
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
    });
    document.body.addEventListener('htmx:configRequest', function (event) {
        if (saved === null || saved.length === 0 || event.detail.path.split('?')[0] !== '/plan') return;
        event.detail.parameters.include = saved.slice();
    });
    document.body.addEventListener('htmx:afterSwap', render);
    document.body.addEventListener('htmx:historyRestore', render);
    // The map core renders the panel in JS; wait a tick so its buttons exist.
    document.addEventListener('travelmap:ready', defer);
    document.addEventListener('travelmap:place-selected', defer);
    render();

    function defer() {
        setTimeout(render, 0);
    }

    function render() {
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
        if (saved === null) {
            button.hidden = true;
            return;
        }
        var on = saved.indexOf(button.dataset.saveQid) >= 0;
        button.hidden = false;
        button.textContent = on ? 'Saved' : 'Save to plan';
        button.setAttribute('aria-pressed', String(on));
    }

    // Returns null when storage is blocked, which hides the feature.
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
