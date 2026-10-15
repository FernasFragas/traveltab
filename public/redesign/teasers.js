(function () {
    'use strict';
    var days = [];
    document.addEventListener('travelmap:places-loaded', function (event) {
        if (days.length && days[0].stops.length) return;
        var places = (event.detail.features || []).slice().sort(function (a, b) {
            return (b.properties.sitelinks || 0) - (a.properties.sitelinks || 0);
        }).slice(0, 3).map(function (feature) {
            return { name: feature.properties.name, photoURL: feature.properties.photo_url };
        });
        render('Suggested day', places);
    });
    document.addEventListener('travelmap:plan-updated', function (event) {
        days = event.detail.days || [];
        if (days.length && days[0].stops.length) render('Day 1', days[0].stops.slice(0, 3));
    });

    document.body.addEventListener('htmx:afterSwap', function () {
        var target = document.querySelector('[data-teaser-stays]');
        var stay = document.querySelector('#stays .stay-item');
        if (!target) return;
        if (!stay) {
            var note = document.querySelector('#stays .stay-note');
            if (note) target.textContent = note.textContent.trim();
            return;
        }
        var card = document.createElement('div'); card.className = 'teaser-stay';
        var image = stay.querySelector('.stay-image');
        if (image && image.src) card.appendChild(image.cloneNode(true));
        var details = document.createElement('div');
        var title = document.createElement('h4'); title.textContent = stay.querySelector('.stay-name').textContent;
        details.appendChild(title);
        var meta = stay.querySelector('.stay-meta');
        if (meta) { var text = document.createElement('p'); text.textContent = meta.textContent.trim(); details.appendChild(text); }
        card.appendChild(details); target.replaceChildren(card);
    });

    function render(label, stops) {
        var preview = document.querySelector('[data-teaser-day]');
        if (!preview) return;
        var heading = document.createElement('p'); heading.className = 'teaser-day-label'; heading.textContent = label;
        var list = document.createElement('ol'); list.className = 'teaser-stops';
        stops.forEach(function (stop, index) {
            var item = document.createElement('li');
            if (stop.photoURL) {
                var photo = document.createElement('img'); photo.src = stop.photoURL; photo.alt = ''; photo.loading = 'lazy'; item.appendChild(photo);
            }
            var text = document.createElement('span');
            var time = document.createElement('small'); time.textContent = ['Morning', 'Afternoon', 'Evening'][index]; text.appendChild(time);
            var name = document.createElement('span'); name.textContent = stop.name; text.appendChild(name);
            item.appendChild(text); list.appendChild(item);
        });
        preview.replaceChildren(heading, list);
    }
})();
