#!/usr/bin/env node
// PR 7 together: 7f chips toggle the 7a and 7b layers; 7d saves from 7a's real panel and the plan request carries them.
// Start an isolated Brave/Chromium on CDP_PORT with about:blank, then run this check.
import assert from 'node:assert/strict';
import { connect, openPlacePanel, startPreview } from './lib/browser.mjs';

for (const [engine, scenario] of [['maplibre', 'writers-map'], ['leaflet', 'writers-map-leaflet']]) {
const preview = await startPreview({ scenario });
let browser;
try {
  browser = await connect();
  await browser.call('Page.addScriptToEvaluateOnNewDocument', { source: `
    window.__previewErrors = [];
    const originalError = console.error.bind(console);
    console.error = (...args) => { window.__previewErrors.push(args.map(String).join(' ')); originalError(...args); };
    window.addEventListener('error', event => window.__previewErrors.push(event.message));
    window.addEventListener('unhandledrejection', event => window.__previewErrors.push(String(event.reason)));
    window.__planURLs = [];
    const open = XMLHttpRequest.prototype.open;
    XMLHttpRequest.prototype.open = function (method, url, ...rest) { if (String(url).includes('/plan')) window.__planURLs.push(String(url)); return open.call(this, method, url, ...rest); };
  ` });
  for (const width of [375, 1440]) {
    await browser.viewport(width, 900);
    await browser.navigate(`${preview.url}/trip/madeira-pt`);
    await browser.evaluate('(() => { try { localStorage.clear(); sessionStorage.clear(); } catch {} })()');
    await browser.navigate(`${preview.url}/trip/madeira-pt`);
    await browser.until("document.querySelectorAll('.map-pin').length > 0 && document.querySelectorAll('[data-map-slot=\"layers\"] button').length > 0");
    const count = sel => browser.evaluate(`document.querySelectorAll('${sel}').length`);
    const chip = label => `[...document.querySelectorAll('[data-map-slot="layers"] button')].find(b => b.textContent.includes("${label}"))`;
    const chips = await browser.evaluate(`[...document.querySelectorAll('[data-map-slot="layers"] button')].map(b => [b.textContent.trim(), b.getAttribute('aria-pressed'), b.disabled, b.hidden])`);

    // 7f toggles 7a's layers.
    const writers = await count('.map-pin-writer');
    const notable = await count('.map-pin-notable');
    await browser.evaluate("document.querySelectorAll('.map-pin-notable').forEach(pin => { pin.__kept = true; })");
    await browser.evaluate(`${chip("Writers")}.click()`);
    assert.equal(await count('.map-pin-writer'), 0, 'writers chip off');
    assert.equal(await browser.evaluate("[...document.querySelectorAll('.map-pin-notable')].every(pin => pin.__kept)"), true, 'notable pins not redrawn');
    assert.equal(await count('.map-pin-notable'), notable);
    assert.equal(await browser.evaluate(`${chip("Writers")}.getAttribute('aria-pressed')`), 'false');
    await browser.evaluate(`${chip("Notable")}.click()`);
    assert.equal(await count('.map-pin-notable'), 0, 'notable chip off');
    await browser.evaluate(`${chip("Writers")}.click()`);
    await browser.evaluate(`${chip("Notable")}.click()`);
    assert.equal(await count('.map-pin-writer'), writers);
    assert.equal(await count('.map-pin-notable'), notable);

    // 7d saves from 7a's real panel.
    await openPlacePanel(browser, 'Pico Ruivo');
    await browser.until("document.querySelector('.map-place-panel [data-save-qid]')");
    await browser.evaluate("document.querySelector('.map-place-panel [data-save-qid]').click()");
    const saved = await browser.evaluate("({ button: document.querySelector('.map-place-panel [data-save-qid]').textContent.trim(), line: document.querySelector('[data-map-slot=\"saved\"]').textContent.trim() })");
    assert.equal(saved.button, 'Saved', JSON.stringify(saved));
    assert.match(saved.line, /1 saved place/, JSON.stringify(saved));

    // Reload keeps the save; reopening the panel shows it as saved.
    await browser.navigate(`${preview.url}/trip/madeira-pt`);
    await browser.until("document.querySelectorAll('.map-pin').length > 0");
    await openPlacePanel(browser, 'Pico Ruivo');
    await browser.until("document.querySelector('.map-place-panel [data-save-qid]')");
    await browser.until("document.querySelector('.map-place-panel [data-save-qid]').textContent.trim() === 'Saved'");

    // Plan request carries the save; 7f enables the plan chip; plan chip toggles 7b's pins.
    assert.equal(await browser.evaluate(`${chip("day plan")}.disabled`), true, 'plan chip disabled before a plan');
    await browser.evaluate("document.querySelector('#trip-card form').requestSubmit()");
    await browser.until("document.querySelectorAll('.plan-pin').length > 0");
    const planURL = await browser.evaluate('window.__planURLs.at(-1)');
    assert.match(planURL, /Q473169/, planURL);
    await browser.until(`!${chip("day plan")}.disabled`);
    await browser.evaluate(`${chip("day plan")}.click()`);
    assert.equal(await count('.plan-pin'), 0, 'plan chip off');
    await browser.evaluate(`${chip("day plan")}.click()`);
    assert.ok(await count('.plan-pin') > 0, 'plan chip on');

    // Every chip off leaves an empty map with no error; turning them back on restores the pins.
    const visibleChips = `[...document.querySelectorAll('[data-map-slot="layers"] button')].filter(b => !b.hidden && !b.disabled)`;
    await browser.evaluate(`${visibleChips}.forEach(b => { if (b.getAttribute('aria-pressed') === 'true') b.click(); })`);
    assert.equal(await count('.map-pin') + await count('.plan-pin'), 0, 'all chips off');
    await browser.evaluate(`${visibleChips}.forEach(b => { if (b.getAttribute('aria-pressed') === 'false') b.click(); })`);
    assert.equal(await count('.map-pin-writer'), writers);
    assert.ok(await count('.plan-pin') > 0);

    // Opening a panel (an htmx swap) keeps the chosen day (7b fix) and doesn't duplicate layers.
    const before = await count('.plan-pin');
    await browser.evaluate("document.querySelectorAll('.plan-day-toggle button')[1].click()");
    const dayOne = await count('.plan-pin');
    await openPlacePanel(browser, 'Funchal');
    await browser.until("document.querySelector('.map-place-panel h3')?.textContent === 'Funchal'");
    assert.equal(await count('.plan-pin'), dayOne, 'day survives panel swap');
    assert.ok(dayOne < before);
    assert.equal(await count('.map-pin-writer'), writers, 'no duplicate writer pins');

    const overflow = await browser.evaluate('document.documentElement.scrollWidth');
    assert.ok(overflow <= width, `overflow ${overflow}`);
    assert.deepEqual(await browser.evaluate('window.__previewErrors'), []);
    console.log(`PASS ${engine} ${width}px`, JSON.stringify({ chips, writers, notable, saved, planURL, before, dayOne, overflow }));
  }
} finally {
  await browser?.close?.();
  preview.stop();
}
}
