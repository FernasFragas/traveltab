(function () {
    'use strict';
    if (window.travelTabNavigationWired) return;
    window.travelTabNavigationWired = true;

    var sectionIDs = ['overview', 'itinerary', 'stays', 'videos'];
    var search = document.getElementById('destination-search');
    var searchError = document.getElementById('destination-search-error');
    var searchStatus = document.getElementById('destination-search-status');
    function currentSection() {
        var id = location.hash.slice(1);
        return sectionIDs.includes(id) ? id : 'overview';
    }

    function setActive(id) {
        document.querySelectorAll('[data-tt-section]').forEach(function (link) {
            if (link.dataset.ttSection === id) {
                link.setAttribute('aria-current', 'location');
            } else {
                link.removeAttribute('aria-current');
            }
        });
    }

    function showSection(id) {
        sectionIDs.forEach(function (sectionID) {
            var section = document.getElementById(sectionID);
            if (section) section.hidden = sectionID !== id;
        });
        setActive(id);
    }

    document.addEventListener('click', function (event) {
        if (!(event.target instanceof Element)) return;
        var link = event.target.closest('a[data-tt-section], .tt-skip-link');
        if (link && !event.defaultPrevented && event.button === 0 && !event.metaKey && !event.ctrlKey && !event.shiftKey && !event.altKey) {
            var id = link.getAttribute('href').slice(1);
            var target = document.getElementById(id);
            if (target) {
                showSection(id);
                target.setAttribute('tabindex', '-1');
                target.focus({ preventScroll: true });
            }
        }
    });

    // Keyboard focus that moves into the cross-origin map frame is invisible to the page: the browser
    // neither scrolls the frame into view nor lets CSS style it. Do both, for keyboard focus only.
    var keyboardFocus = false;
    document.addEventListener('keydown', function (event) { if (event.key === 'Tab') keyboardFocus = true; }, true);
    document.addEventListener('pointerdown', function () { keyboardFocus = false; }, true);
    window.addEventListener('blur', function () {
        var frame = document.activeElement;
        if (!keyboardFocus || !frame || frame.tagName !== 'IFRAME' || !frame.classList.contains('map-embed')) return;
        frame.scrollIntoView({ block: 'nearest', inline: 'nearest' });
        var card = frame.closest('.map-card');
        if (card) card.setAttribute('data-frame-focus', 'true');
    });
    window.addEventListener('focus', function () {
        document.querySelectorAll('.map-card[data-frame-focus]').forEach(function (card) { card.removeAttribute('data-frame-focus'); });
    });

    function isSearchEvent(event) {
        var detail = event.detail || {};
        var source = detail.requestConfig && detail.requestConfig.elt || detail.elt;
        return source === search || source instanceof Element && source.closest('#destination-search') === search;
    }

    document.body.addEventListener('htmx:beforeRequest', function (event) {
        if (!isSearchEvent(event)) return;
        searchError.textContent = '';
        if (searchStatus) searchStatus.textContent = '';
        search.setAttribute('aria-busy', 'true');
    });
    document.body.addEventListener('htmx:beforeSwap', function (event) {
        var detail = event.detail;
        if (isSearchEvent(event) && detail.target && detail.target.id === 'destination-search-error' && detail.xhr.status >= 400) {
            detail.shouldSwap = true;
            detail.isError = false;
        }
    });
    document.body.addEventListener('htmx:afterRequest', function (event) {
        if (!isSearchEvent(event)) return;
        search.removeAttribute('aria-busy');
        if (event.detail.failed && !searchError.textContent.trim()) {
            searchError.textContent = 'We could not load that destination. Please try again.';
        }
    });
    ['htmx:sendError', 'htmx:timeout'].forEach(function (name) {
        document.body.addEventListener(name, function (event) {
            if (!isSearchEvent(event)) return;
            search.removeAttribute('aria-busy');
            searchError.textContent = 'We could not reach the destination service. Please try again.';
        });
    });
    document.body.addEventListener('htmx:afterSwap', function (event) {
        if (event.detail.target && event.detail.target.id === 'content-area') {
            if (searchError) searchError.textContent = '';
            if (searchStatus) searchStatus.textContent = 'Destination updated. Your overview is ready.';
            if (location.hash !== '#overview') history.replaceState(history.state, '', location.pathname + location.search + '#overview');
            showSection('overview');
        }
    });
    document.body.addEventListener('htmx:afterSettle', function (event) {
        if (!event.detail.target || event.detail.target.id !== 'trip-card') return;
        if (location.hash !== '#itinerary') history.replaceState(history.state, '', location.pathname + location.search + '#itinerary');
        showSection('itinerary');
    });
    document.body.addEventListener('htmx:historyRestore', function () { showSection(currentSection()); });
    window.addEventListener('hashchange', function () { showSection(currentSection()); });
    showSection(currentSection());
})();
