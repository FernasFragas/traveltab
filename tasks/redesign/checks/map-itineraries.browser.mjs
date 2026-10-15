#!/usr/bin/env node
// Start an isolated Brave/Chromium on CDP_PORT with about:blank, then run this check.
// M5: one writer itinerary on both map engines at 375 and 1440 px.
import assert from 'node:assert/strict';
import { connect, showLayer, startPreview } from './lib/browser.mjs';

let browser;
try {
  browser = await connect();
  await browser.call('Page.addScriptToEvaluateOnNewDocument', { source: `
    window.__previewErrors = [];
    window.__itineraryMap = null;
    window.__mapEngine = '';
    document.addEventListener('travelmap:ready', event => {
      window.__itineraryMap = event.detail.map;
      window.__mapEngine = event.detail.engine;
    });
    const originalError = console.error.bind(console);
    console.error = (...args) => { window.__previewErrors.push(args.map(String).join(' ')); originalError(...args); };
    window.addEventListener('error', event => window.__previewErrors.push(event.message));
    window.addEventListener('unhandledrejection', event => window.__previewErrors.push(String(event.reason)));
  ` });
  for (const [engine, scenario] of [['maplibre', 'writers-map'], ['leaflet', 'writers-map-leaflet']]) {
    const preview = await startPreview({ scenario });
    try {
      for (const width of [375, 1440]) {
        await browser.viewport(width, 900);
        await browser.navigate(`${preview.url}/trip/madeira-pt`);
        await browser.evaluate('(() => { try { localStorage.clear(); sessionStorage.clear(); } catch {} })()');
        await browser.navigate(`${preview.url}/trip/madeira-pt`);
        await browser.until(`window.__mapEngine === '${engine}'`, 30000);
        await showLayer(browser, 'itineraries');
        await browser.until(`document.querySelectorAll('.itinerary-pin').length > 0`, 30000);
        browser.errors.length = 0;
        const count = selector => browser.evaluate(`document.querySelectorAll('${selector}').length`);
        const chip = `[...document.querySelectorAll('[data-map-slot="layers"] button')].find(b => b.getAttribute('aria-label') === "Itineraries")`;
        const lineCount = engine === 'maplibre'
          ? "Number(Boolean(window.__itineraryMap?.getLayer('draw-line-itineraries')))"
          : "[...document.querySelectorAll('.leaflet-overlay-pane path')].filter(p => p.getAttribute('stroke') === '#a582d4').length";
        const first = await count('.itinerary-pin');
        const look = await browser.evaluate(`({ colour: getComputedStyle(document.querySelector('.itinerary-pin')).backgroundColor,
          numbers: [...document.querySelectorAll('.itinerary-pin')].map(p => p.textContent).join(','),
          buttons: [...document.querySelectorAll('.itinerary-pin')].every(p => p.tagName === 'BUTTON' && p.title),
          lines: ${lineCount}, chip: ${chip}.hidden === false && getComputedStyle(${chip}).display !== 'none',
          overflow: document.documentElement.scrollWidth })`);
        assert.equal(look.colour, 'rgb(165, 130, 212)', JSON.stringify(look));
        assert.equal(look.numbers, Array.from({ length: first }, (_, i) => i + 1).join(','));
        assert.ok(look.buttons, JSON.stringify(look));
        assert.equal(look.lines, 1, JSON.stringify(look));
        assert.ok(look.chip, JSON.stringify(look));
        assert.ok(look.overflow <= width, JSON.stringify(look));

        await browser.evaluate("document.querySelector('.itinerary-menu > summary').click()");
        assert.equal(await browser.evaluate("document.querySelector('.itinerary-menu').open"), true);
        const options = await browser.evaluate("[...document.querySelectorAll('.itinerary-picker option')].map(o => o.textContent)");
        assert.equal(options.length, 2, JSON.stringify(options));
        await browser.evaluate("(() => { const s = document.querySelector('.itinerary-picker'); s.value = '1'; s.dispatchEvent(new Event('change')); })()");
        const second = await count('.itinerary-pin');
        assert.notEqual(second, first);
        assert.equal(await browser.evaluate(lineCount), 1);
        assert.equal(await count('.itinerary-picker'), 1);

        await browser.evaluate("document.querySelector('.itinerary-pin').click()");
        await browser.until("document.querySelector('.itinerary-popup a')");
        const link = await browser.evaluate("(() => { const a = document.querySelector('.itinerary-popup a'); return [a.href, a.rel, document.querySelector('.itinerary-popup').textContent]; })()");
        assert.match(link[0], /^https:\/\/viagensasolta\.com\//);
        assert.equal(link[1], 'noopener noreferrer');
        assert.match(link[2], /Route by viagensasolta\.com/);

        if (engine === 'maplibre') {
          await browser.evaluate("(() => { window.__styleReloaded = false; window.__itineraryMap.once('style.load', () => { window.__styleReloaded = true; }); window.__itineraryMap.setStyle(document.querySelector('[data-writers-map]').dataset.mapStyleUrl); return true; })()");
          await browser.until("window.__styleReloaded && window.__itineraryMap?.getLayer('draw-line-itineraries') && window.__itineraryMap?.isStyleLoaded() && window.__itineraryMap?.areTilesLoaded()", 30000);
          assert.equal(await browser.evaluate(lineCount), 1);
          assert.equal(await count('.itinerary-pin'), second);
          assert.equal(await count('.itinerary-picker'), 1);
        }

        await browser.evaluate(`${chip}.click()`);
        assert.equal(await count('.itinerary-pin'), 0);
        assert.equal(await browser.evaluate(lineCount), 0);
        assert.equal(await count('.itinerary-picker'), 0);
        assert.equal(await count('.itinerary-popup'), 0);
        await browser.evaluate(`${chip}.click()`);
        await browser.until("document.querySelectorAll('.itinerary-pin').length > 0");
        assert.equal(await count('.itinerary-pin'), second);
        assert.equal(await browser.evaluate(lineCount), 1);
        assert.equal(await count('.itinerary-picker'), 1);
        assert.deepEqual(await browser.evaluate('window.__previewErrors'), []);

        await browser.navigate(`${preview.url}/trip/porto-pt`);
        await browser.until(`window.__mapEngine === '${engine}'`, 30000);
        assert.equal(await count('.itinerary-pin'), 0);
        assert.equal(await count('.itinerary-picker'), 0);
        assert.equal(await browser.evaluate(`getComputedStyle(${chip}).display`), 'none');
        assert.deepEqual(await browser.evaluate('window.__previewErrors'), []);
        assert.deepEqual(browser.errors, []);
        console.log(`PASS ${engine} ${width}px: ${first}/${second} accessible stop dots, one line, picker, popup, toggle, empty route, console clean`);
      }
    } finally {
      preview.stop();
    }
  }
} finally {
  if (browser) await browser.close();
}
