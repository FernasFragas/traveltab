(function () {
    'use strict';
    // Deliberately supports only simple weekly OSM rules; preserve everything else.
    window.TravelOpeningHours = function (raw, today) {
        if (!raw) return '';
        var days = ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa'];
        var day = (today || new Date()).getDay();
        var result = 'Closed today';
        var rules = raw.split(';');
        for (var i = 0; i < rules.length; i++) {
            var rule = rules[i].trim().match(/^((?:Mo|Tu|We|Th|Fr|Sa|Su)(?:-(?:Mo|Tu|We|Th|Fr|Sa|Su))?) (off|(?:\d{2}:\d{2}-\d{2}:\d{2})(?:,\s*\d{2}:\d{2}-\d{2}:\d{2})*)$/);
            if (!rule) return raw;
            var range = rule[1].split('-').map(function (name) { return days.indexOf(name); });
            var first = range[0], last = range.length === 2 ? range[1] : first;
            var matches = first <= last ? day >= first && day <= last : day >= first || day <= last;
            if (rule[2] !== 'off') {
                var valid = rule[2].split(/[-,]\s*/).every(function (time) {
                    var parts = time.split(':').map(Number);
                    return parts[0] <= 24 && parts[1] < 60 && (parts[0] < 24 || parts[1] === 0);
                });
                if (!valid) return raw;
            }
            if (matches) result = rule[2] === 'off' ? 'Closed today' : rule[2].replace(/-/g, ' – ').replace(/,\s*/g, ', ');
        }
        return result;
    };
})();
