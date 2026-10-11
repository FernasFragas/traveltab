#!/usr/bin/env node
// Start an isolated Brave/Chromium on CDP_PORT with about:blank, then run this check.
// 7c: writers' itineraries on the Madeira fixture, at 375 and 1440 px.
import assert from 'node:assert/strict';
import { connect, startPreview } from './lib/browser.mjs';

const preview = await startPreview({ scenario: 'writers-map' });
let browser;
try {
  browser = await connect();
  await browser.call('Page.addScriptToEvaluateOnNewDocument', { source: `
    window.__previewErrors = [];
    const originalError = console.error.bind(console);
    console.error = (...args) => { window.__previewErrors.push(args.map(String).join(' ')); originalError(...args); };
    window.addEventListener('error', event => window.__previewErrors.push(event.message));
    window.addEventListener('unhandledrejection', event => window.__previewErrors.push(String(event.reason)));
  ` });
  for (const width of [375, 1440]) {
    await browser.viewport(width, 900);
    await browser.navigate(`${preview.url}/trip/madeira-pt`);
    await browser.evaluate('(() => { try { localStorage.clear(); sessionStorage.clear(); } catch {} })()');
    await browser.navigate(`${preview.url}/trip/madeira-pt`);
    await browser.until("document.querySelectorAll('.itinerary-pin').length > 0");
    browser.errors.length = 0;
    const count = sel => browser.evaluate(`document.querySelectorAll('${sel}').length`);
    const chip = "[...document.querySelectorAll('[data-map-slot=\"layers\"] button')].find(b => b.textContent.includes(\"itineraries\"))";

    // Pins are purple and numbered 1..n, with a line between them.
    const first = await count('.itinerary-pin');
    const look = await browser.evaluate(`({ colour: getComputedStyle(document.querySelector('.itinerary-pin')).backgroundColor,
      numbers: [...document.querySelectorAll('.itinerary-pin')].map(p => p.textContent).join(','),
      lines: document.querySelectorAll('path.itinerary-line').length,
      chip: ${chip}.hidden === false && getComputedStyle(${chip}).display !== 'none',
      overflow: document.documentElement.scrollWidth })`);
    assert.equal(look.colour, 'rgb(123, 63, 184)', JSON.stringify(look));
    assert.equal(look.numbers, Array.from({ length: first }, (_, i) => i + 1).join(','));
    assert.equal(look.lines, 1);
    assert.ok(look.chip, 'chip visible');
    assert.ok(look.overflow <= width, JSON.stringify(look));

    // Two itineraries: the picker switches one route at a time without duplicates.
    const options = await browser.evaluate("[...document.querySelectorAll('.itinerary-picker option')].map(o => o.textContent)");
    assert.equal(options.length, 2, JSON.stringify(options));
    await browser.evaluate("(() => { const s = document.querySelector('.itinerary-picker'); s.value = '1'; s.dispatchEvent(new Event('change')); })()");
    const second = await count('.itinerary-pin');
    assert.notEqual(second, first);
    assert.equal(await count('path.itinerary-line'), 1);
    assert.equal(await count('.itinerary-picker'), 1);

    // The route is labelled and links the post.
    await browser.evaluate("document.querySelector('.itinerary-pin').click()");
    await browser.until("document.querySelector('.itinerary-popup a')");
    const link = await browser.evaluate("(() => { const a = document.querySelector('.itinerary-popup a'); return [a.href, a.rel, document.querySelector('.itinerary-popup').textContent]; })()");
    assert.match(link[0], /^https:\/\/viagensasolta\.com\//);
    assert.equal(link[1], 'noopener noreferrer');
    assert.match(link[2], /Route by viagensasolta\.com/);

    // The chip toggles the layer and the picker with it.
    await browser.evaluate(`${chip}.click()`);
    assert.equal(await count('.itinerary-pin'), 0);
    assert.equal(await count('.itinerary-picker'), 0);
    await browser.evaluate(`${chip}.click()`);
    await browser.until("document.querySelectorAll('.itinerary-pin').length > 0");
    assert.equal(await count('.itinerary-pin'), second);
    assert.equal(await count('.itinerary-picker'), 1);
    assert.deepEqual(await browser.evaluate('window.__previewErrors'), []);

    // A destination without itineraries hides the chip and draws nothing.
    await browser.navigate(`${preview.url}/trip/porto-pt`);
    await browser.until("document.querySelectorAll('.map-pin').length > 0");
    assert.equal(await count('.itinerary-pin'), 0);
    assert.equal(await count('.itinerary-picker'), 0);
    assert.equal(await browser.evaluate(`getComputedStyle(${chip}).display`), 'none');
    assert.deepEqual(await browser.evaluate('window.__previewErrors'), []);
    assert.deepEqual(browser.errors, []);
    console.log(`PASS ${width}px: ${first}/${second} purple numbered pins, line, picker, post link, chip toggle, hidden when none, console clean`);
  }
} finally {
  if (browser) await browser.close();
  preview.stop();
}
