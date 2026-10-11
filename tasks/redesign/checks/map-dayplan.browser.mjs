#!/usr/bin/env node
// Start an isolated Brave/Chromium on CDP_PORT with about:blank, then run this check.
// Generates, regenerates and clears a day plan on a map destination at 375 and 1440 px.
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
    await browser.until("Boolean(document.querySelector('.leaflet-container') && document.querySelector('#trip-card form'))");
    browser.errors.length = 0;
    const pins = () => browser.evaluate("document.querySelectorAll('.plan-pin').length");
    const generate = async () => {
      await browser.evaluate("document.querySelector('#trip-card form').requestSubmit()");
      await browser.until("document.querySelectorAll('[data-plan-stop]').length > 0");
    };
    assert.equal(await pins(), 0);
    await generate();
    await browser.until("document.querySelectorAll('.plan-pin').length > 0");
    const first = await pins();
    assert.equal(first, await browser.evaluate("document.querySelectorAll('[data-plan-stop]').length"));
    await generate();
    await browser.until(`document.querySelectorAll('.plan-pin').length === ${first}`);
    const layout = await browser.evaluate(`({ overflow: document.documentElement.scrollWidth,
      green: getComputedStyle(document.querySelector('.plan-pin')).backgroundColor,
      toggle: [...document.querySelectorAll('.plan-day-toggle button')].map(b => b.textContent) })`);
    assert.ok(layout.overflow <= width, JSON.stringify(layout));
    assert.equal(layout.green, 'rgb(46, 125, 50)');
    assert.deepEqual(layout.toggle.slice(0, 2), ['All', 'Day 1']);
    await browser.evaluate("document.querySelectorAll('.plan-day-toggle button')[1].click()");
    assert.ok((await pins()) < first);
    await browser.evaluate("window.TravelMap.updatePlan([])");
    assert.equal(await pins(), 0);
    assert.deepEqual(await browser.evaluate('window.__previewErrors'), []);
    assert.deepEqual(browser.errors, []);
    console.log(`PASS ${width}px: ${first} green numbered pins, regenerate without duplicates, day toggle, clear, console clean`);
  }
} finally {
  if (browser) await browser.close();
  preview.stop();
}
