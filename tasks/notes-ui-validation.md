# Planner UI completion — 2026-09-20

The route now rejects empty cities, NaN/infinite coordinates and out-of-range latitude/longitude.
The new input tests first returned 200 instead of 400, then passed after validation was added.

Rendered-card regressions first failed for missing indoor/outdoor labels and for treating absent
forecast values as merely uncertain. The cards now show `Indoors`, `Outdoors`, `Indoor & outdoor`,
and `No forecast data for this day` as appropriate. Existing uncertain-forecast tests still pass.

Brave headless, driven through its local debugging protocol, checked the real app templates
and HTMX with fixture weather, trip and stay providers. Final checks:

- Viewport 375 × 812; document width 375 before and after planning; no overflowing elements.
- Three-day form submission returned 200 and rendered day cards, rain badge, stays and dated Booking link.
- An invalid coordinate returned 400 and HTMX displayed the friendly error in the card.
- Five-day fixture returned 200 and rendered the stay-service fallback message.
- Commons photos loaded at natural width 500; HTMX loaded from the page's CDN.
- No JavaScript exceptions. The only console error was the expected 400 from the invalid-input check.
- Subtitle contrast improved from 1.72:1 to 7.41:1 by giving the planner an opaque dark background.

The fixture map was blanked deliberately; this does not validate the production map embed.
Fixture stops are repeated across days to exercise rendering; actual grouping is covered by
the recorded-city tests, including the Kyoto regression.

[Before planning](artifacts/traveltab-mobile-before.png) ·
[After planning](artifacts/traveltab-mobile-after.png) ·
[Browser results](artifacts/traveltab-ui-browser-results.json)
