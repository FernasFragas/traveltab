(function () {
    'use strict';

    if (window.travelTabSearchWired) return;
    window.travelTabSearchWired = true;

    var form = document.getElementById('destination-search');
    var input = document.getElementById('city_name');
    var countryCode = document.getElementById('country_code');
    var placeLat = document.getElementById('place_lat');
    var placeLon = document.getElementById('place_lon');
    var list = document.getElementById('destination-suggestions');
    if (!form || !input || !countryCode || !placeLat || !placeLon || !list) return;

    var active = -1;
    var picked = false;

    function options() {
        return Array.from(list.querySelectorAll('[role="option"]'));
    }

    function setActive(index) {
        var items = options();
        active = index;
        items.forEach(function (item, i) {
            item.setAttribute('aria-selected', String(i === index));
        });
        if (index >= 0 && items[index]) {
            input.setAttribute('aria-activedescendant', items[index].id);
            items[index].scrollIntoView({ block: 'nearest' });
        } else {
            input.removeAttribute('aria-activedescendant');
        }
    }

    function close() {
        list.hidden = true;
        input.setAttribute('aria-expanded', 'false');
        setActive(-1);
    }

    function openIfAvailable() {
        if (picked || options().length === 0 || document.activeElement !== input) {
            close();
            return;
        }
        list.hidden = false;
        input.setAttribute('aria-expanded', 'true');
        setActive(-1);
    }

    function pick(item) {
        if (!item) return;
        picked = true;
        input.value = item.dataset.value || '';
        countryCode.value = item.dataset.countryCode || '';
        placeLat.value = item.dataset.lat || '';
        placeLon.value = item.dataset.lon || '';
        close();
        form.requestSubmit();
    }

    input.addEventListener('input', function () {
        countryCode.value = '';
        placeLat.value = '';
        placeLon.value = '';
        picked = false;
        close();
    });
    input.addEventListener('focus', openIfAvailable);
    input.addEventListener('keydown', function (event) {
        if (event.key === 'Escape') {
            if (!list.hidden) {
                event.preventDefault();
                close();
            }
            return;
        }
        if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
            var items = options();
            if (!items.length) return;
            event.preventDefault();
            list.hidden = false;
            input.setAttribute('aria-expanded', 'true');
            setActive(event.key === 'ArrowDown' ? (active + 1) % items.length : (active - 1 + items.length) % items.length);
        } else if (event.key === 'Enter' && !list.hidden && active >= 0) {
            event.preventDefault();
            pick(options()[active]);
        } else if (event.key === 'Enter') {
            event.preventDefault();
            form.requestSubmit();
        }
    });
    list.addEventListener('click', function (event) {
        var item = event.target instanceof Element ? event.target.closest('[role="option"]') : null;
        if (item && list.contains(item)) pick(item);
    });
    document.addEventListener('click', function (event) {
        if (!form.contains(event.target) && !list.contains(event.target)) close();
    });
    form.addEventListener('submit', close);
    document.body.addEventListener('htmx:beforeSwap', function (event) {
        if (event.detail.target !== list) return;
        var sent = new URL(event.detail.xhr.responseURL, location.href).searchParams.get('city_name');
        if (picked || sent !== input.value) event.detail.shouldSwap = false;
    });
    document.body.addEventListener('htmx:afterSwap', function (event) {
        if (event.detail.target === list) openIfAvailable();
    });
})();
