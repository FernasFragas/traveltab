(function () {
    'use strict';
    var rules = [
        { kinds: /museum|gallery|palace|monastery|church|cathedral|tower|castle|fort|monument/i, value: 'Morning', tip: 'Tip: quieter early in the day' },
        { kinds: /viewpoint|miradouro|lookout|bridge/i, value: 'Sunset', tip: 'Tip: best light at golden hour' },
        { kinds: /park|garden|beach/i, value: 'Morning', tip: 'Tip: cooler and calmer early' },
        { kinds: /market/i, value: 'Morning', tip: 'Tip: freshest stalls in the morning' }
    ];
    window.TravelBestTime = function (kind) {
        return rules.find(function (rule) { return rule.kinds.test(kind || ''); }) || null;
    };
})();
