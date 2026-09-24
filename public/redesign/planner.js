(function () {
    'use strict';

    if (window.travelTabPlannerWired) {
        return;
    }
    window.travelTabPlannerWired = true;

    var weekdays = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
    var months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
    var pendingValues = null;
    var focusScheduled = false;

    function ensureStatusRegion() {
        var element = document.getElementById('trip-status');
        if (element) return element;

        element = document.createElement('p');
        element.id = 'trip-status';
        element.className = 'tt-visually-hidden';
        element.setAttribute('role', 'status');
        element.setAttribute('aria-live', 'polite');
        document.body.appendChild(element);
        return element;
    }

    var tripStatus = ensureStatusRegion();

    function announce(message) {
        if (!tripStatus) return;
        tripStatus.textContent = message;
    }

    function updateEndLabel(form) {
        if (!form) return;
        var start = form.elements.start;
        var days = form.elements.days;
        var output = form.querySelector('[data-trip-end]');
        if (!start || !days || !output) return;

        var value = String(start.value || '');
        var dayCount = Number(days.value);
        var match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
        if (!match || !Number.isFinite(dayCount) || dayCount < 1) {
            output.textContent = '';
            return;
        }

        var year = Number(match[1]);
        var month = Number(match[2]);
        var day = Number(match[3]);
        var date = new Date(Date.UTC(year, month - 1, day));
        if (Number.isNaN(date.getTime()) || date.getUTCFullYear() !== year ||
            date.getUTCMonth() !== month - 1 || date.getUTCDate() !== day) {
            output.textContent = '';
            return;
        }

        date.setUTCDate(date.getUTCDate() + dayCount - 1);
        output.textContent = weekdays[date.getUTCDay()] + ', ' + date.getUTCDate() + ' ' + months[date.getUTCMonth()];
    }

    function captureFormValues(form) {
        return {
            city: form.elements.city ? form.elements.city.value : '',
            country: form.elements.country ? form.elements.country.value : '',
            lat: form.elements.lat ? form.elements.lat.value : '',
            lon: form.elements.lon ? form.elements.lon.value : '',
            start: form.elements.start ? form.elements.start.value : '',
            days: form.elements.days ? form.elements.days.value : ''
        };
    }

    function restoreFormValues(form, values) {
        if (!form || !values) return;
        ['city', 'country', 'lat', 'lon'].forEach(function (name) {
            if (form.elements[name]) form.elements[name].value = values[name] || '';
        });
        if (form.elements.start) form.elements.start.value = values.start || '';
        if (form.elements.days) {
            var option = form.elements.days.querySelector('option[value="' + values.days + '"]');
            if (option) form.elements.days.value = values.days;
        }
        updateEndLabel(form);
    }

    function focusItineraryHeading() {
        var heading = document.getElementById('itinerary-title');
        if (!heading) return;
        var reduce = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
        heading.focus({ preventScroll: true });
        heading.scrollIntoView({ block: 'start', behavior: reduce ? 'auto' : 'smooth' });
    }

    function scheduleFocusItinerary() {
        if (focusScheduled) return;
        focusScheduled = true;
        window.setTimeout(function () {
            focusScheduled = false;
            focusItineraryHeading();
        }, 0);
    }

    function setPending(form, pending) {
        if (!form) return;
        form.dataset.tripPending = pending ? 'true' : 'false';
        form.setAttribute('aria-busy', pending ? 'true' : 'false');
        var submit = form.querySelector('.trip-submit');
        if (submit) submit.disabled = pending;
    }

    function isTripForm(element) {
        return element instanceof Element && element.classList.contains('trip-form');
    }

    document.body.addEventListener('htmx:historyRestore', function () {
        // HTMX can snapshot the form before afterRequest clears its loading state.
        setPending(document.querySelector('.trip-form'), false);
        pendingValues = null;
        tripStatus = ensureStatusRegion();
        announce('');
    });

    document.addEventListener('submit', function (event) {
        if (!isTripForm(event.target)) return;
        if (event.target.dataset.tripPending === 'true') {
            event.preventDefault();
            event.stopPropagation();
            return;
        }
        pendingValues = captureFormValues(event.target);
        setPending(event.target, true);
        announce('Planning your trip…');
    }, true);

    document.addEventListener('input', function (event) {
        if (!(event.target instanceof Element)) return;
        var form = event.target.closest('.trip-form');
        if (form) updateEndLabel(form);
    });

    document.addEventListener('change', function (event) {
        if (!(event.target instanceof Element)) return;
        var form = event.target.closest('.trip-form');
        if (form) updateEndLabel(form);
    });

    document.body.addEventListener('htmx:beforeSwap', function (event) {
        var target = event.detail.target;
        var xhr = event.detail.xhr;
        if (target && target.id === 'trip-card' && xhr && xhr.status >= 400) {
            event.detail.shouldSwap = true;
            event.detail.isError = false;
        }
    });

    document.body.addEventListener('htmx:afterRequest', function (event) {
        var form = event.detail.requestConfig && event.detail.requestConfig.elt;
        if (isTripForm(form)) setPending(form, false);
        if (event.detail.failed) announce('Planning did not complete. Please check the form and try again.');
    });

    document.body.addEventListener('htmx:afterSwap', function (event) {
        var target = event.detail.target;
        var xhr = event.detail.xhr;
        if (!target || target.id !== 'trip-card' || !xhr || xhr.status < 400 || !pendingValues) return;
        restoreFormValues(target.querySelector('.trip-form'), pendingValues);
    });

    document.body.addEventListener('htmx:afterSettle', function (event) {
        var target = event.detail.target;
        var xhr = event.detail.xhr;
        if (!target || target.id !== 'trip-card' || !xhr || xhr.status < 200 || xhr.status >= 400) return;
        scheduleFocusItinerary();
        announce('Your itinerary has been updated.');
    });

    ['htmx:sendError', 'htmx:timeout'].forEach(function (name) {
        document.body.addEventListener(name, function (event) {
            var form = event.detail.requestConfig && event.detail.requestConfig.elt;
            if (isTripForm(form)) setPending(form, false);
            announce('Planning did not complete. Please try again.');
        });
    });
})();
