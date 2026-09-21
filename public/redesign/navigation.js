(function () {
    'use strict';
    if (window.travelTabNavigationWired) return;
    window.travelTabNavigationWired = true;

    var sectionIDs = ['overview', 'itinerary', 'stays', 'videos'];
    var menuButton = document.querySelector('.tt-menu-button');
    var headerNavigation = document.getElementById('header-navigation');
    var mobile = window.matchMedia('(max-width: 639px)');
    var search = document.getElementById('destination-search');
    var searchError = document.getElementById('destination-search-error');
    var searchStatus = document.getElementById('destination-search-status');
    var sections = [];
    var offset = 76;
    var frame = null;
    var navigationObserver = null;

    function setActive(id) {
        document.querySelectorAll('[data-tt-section]').forEach(function (link) {
            if (link.dataset.ttSection === id) {
                link.setAttribute('aria-current', 'location');
            } else {
                link.removeAttribute('aria-current');
            }
        });
    }

    function updateActive() {
        frame = null;
        var active = 'overview';
        sections.forEach(function (section) {
            if (section.getBoundingClientRect().top <= offset + 8) active = section.id;
        });
        // The last sections may be too short to reach the top of the viewport.
        if (window.scrollY > 0 && window.scrollY + window.innerHeight >= document.documentElement.scrollHeight - 3 && sections.length) {
            active = sections[sections.length - 1].id;
        }
        setActive(active);
    }

    function scheduleActive() {
        if (frame === null) frame = window.requestAnimationFrame(updateActive);
    }

    function measureNavigation() {
        var navigation = document.querySelector('.tt-section-navigation');
        offset = navigation ? Math.ceil(navigation.getBoundingClientRect().height) + 16 : 76;
        document.documentElement.style.setProperty('--tt-anchor-offset', offset + 'px');
        scheduleActive();
    }

    function refreshSections() {
        sections = sectionIDs.map(function (id) { return document.getElementById(id); }).filter(Boolean);
        if (navigationObserver) navigationObserver.disconnect();
        var navigation = document.querySelector('.tt-section-navigation');
        if (navigation && window.ResizeObserver) {
            navigationObserver = new ResizeObserver(measureNavigation);
            navigationObserver.observe(navigation);
        }
        measureNavigation();
    }

    function closeMenu(returnFocus) {
        if (!menuButton || !headerNavigation) return;
        menuButton.setAttribute('aria-expanded', 'false');
        menuButton.setAttribute('aria-label', 'Open section menu');
        headerNavigation.hidden = mobile.matches;
        if (returnFocus) menuButton.focus();
    }

    function configureMenu() {
        if (!menuButton || !headerNavigation) return;
        menuButton.hidden = !mobile.matches;
        closeMenu(false);
    }

    if (menuButton && headerNavigation) {
        menuButton.addEventListener('click', function () {
            var open = menuButton.getAttribute('aria-expanded') === 'true';
            menuButton.setAttribute('aria-expanded', String(!open));
            menuButton.setAttribute('aria-label', open ? 'Open section menu' : 'Close section menu');
            headerNavigation.hidden = open;
        });
    }
    if (mobile.addEventListener) mobile.addEventListener('change', configureMenu);
    else mobile.addListener(configureMenu);

    document.addEventListener('keydown', function (event) {
        if (event.key === 'Escape' && menuButton && menuButton.getAttribute('aria-expanded') === 'true') closeMenu(true);
    });
    document.addEventListener('click', function (event) {
        if (!(event.target instanceof Element)) return;
        var link = event.target.closest('a[data-tt-section], .tt-skip-link');
        if (link && !event.defaultPrevented && event.button === 0 && !event.metaKey && !event.ctrlKey && !event.shiftKey && !event.altKey) {
            var id = link.getAttribute('href').slice(1);
            var target = document.getElementById(id);
            if (target) {
                closeMenu(false);
                setActive(id);
                target.setAttribute('tabindex', '-1');
                target.focus({ preventScroll: true });
            }
        } else if (menuButton && menuButton.getAttribute('aria-expanded') === 'true' && !event.target.closest('.tt-header')) {
            closeMenu(false);
        }
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
            closeMenu(false);
        }
        refreshSections();
    });
    document.body.addEventListener('htmx:afterSettle', refreshSections);
    document.body.addEventListener('htmx:historyRestore', refreshSections);
    window.addEventListener('scroll', scheduleActive, { passive: true });
    window.addEventListener('resize', measureNavigation, { passive: true });
    window.addEventListener('hashchange', scheduleActive);
    configureMenu();
    refreshSections();
})();
