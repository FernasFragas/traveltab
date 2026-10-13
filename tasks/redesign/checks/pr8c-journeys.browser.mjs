#!/usr/bin/env node
// PR 8c: keyboard journeys and screenshots for two writer cities at phone and desktop widths.
// Start an isolated Brave/Chromium on CDP_PORT with about:blank before running this check.
import assert from 'node:assert/strict';
import { mkdir, writeFile } from 'node:fs/promises';
import { connect, startPreview } from './lib/browser.mjs';

const out = process.env.OUTPUT_DIR || 'docs/tdd/artifacts/8c';
const preview = await startPreview({ scenario: 'writers-map' });
let browser;

const focusIs = selector => `document.activeElement?.matches(${JSON.stringify(selector)})`;
const press = key => browser.key(key, key === ' ' ? 'Space' : key, ({ Enter: 13, Escape: 27, ArrowDown: 40, ArrowUp: 38, ' ': 32 })[key], 0, key === ' ' ? ' ' : key === 'Enter' ? '\r' : undefined);
const tabUntil = async (selector, max = 100) => {
  for (let i = 0; i < max; i++) {
    if (await browser.evaluate(focusIs(selector))) return;
    await browser.tab();
  }
  throw new Error(`Tab never reached ${selector}; active=${await browser.evaluate('document.activeElement?.outerHTML')}`);
};
const revealMap = async () => {
  for (let tabs = 0; tabs < 100; tabs++) {
    const visible = await browser.evaluate(`(() => {
      const box = document.querySelector('[data-map-canvas]').getBoundingClientRect();
      return box.top < innerHeight && box.bottom > 0;
    })()`);
    if (visible) return;
    await browser.tab();
  }
  throw new Error('Tab navigation did not reveal the map');
};
const shot = async (name, width) => {
  await browser.settle();
  const height = await browser.evaluate('document.documentElement.scrollHeight');
  await writeFile(`${out}/${name}-${width}.png`, await browser.shot({ x: 0, y: 0, width, height }));
};

try {
  await mkdir(out, { recursive: true });
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
    for (const city of [
      { name: 'Funchal', slug: 'funchal-pt', pin: 'Pico Ruivo', qid: 'Q473169', host: 'www.saltinourhair.com', itineraryHost: 'www.earthtrekkers.com' },
      { name: 'Lisbon', slug: 'lisbon-pt', pin: 'Belém Tower', qid: 'Q215003', host: 'www.earthtrekkers.com', itineraryHost: 'www.earthtrekkers.com' }
    ]) {
      await browser.navigate(`${preview.url}/`);
      await browser.evaluate('localStorage.clear()');
      await browser.navigate(`${preview.url}/`);
      await tabUntil('#city_name');
      await browser.call('Input.insertText', { text: city.name });
      await press('Enter');
      await browser.until(`document.querySelector('[data-writers-map]')?.dataset.mapDestination === ${JSON.stringify(city.slug)} && document.querySelector('.destination-title')?.textContent.trim() === ${JSON.stringify(city.name)}`);
      await revealMap();
      try {
        await browser.until(`Boolean([...document.querySelectorAll('.map-pin')].find(p => p.title === ${JSON.stringify(city.pin)})) && document.querySelectorAll('.itinerary-pin').length > 0`);
      } catch (error) {
        console.error('MAP WAIT', city.name, JSON.stringify(await browser.evaluate(`({
          status: document.querySelector('[data-map-slot=status]')?.textContent,
          map: document.querySelector('[data-writers-map]')?.outerHTML.slice(0, 500),
          markers: document.querySelectorAll('.maplibregl-marker').length,
          errors: window.__previewErrors
        })`)));
        throw error;
      }
      const hero = await browser.evaluate(`({ title: document.querySelector('.destination-title')?.textContent.trim(), weather: document.querySelector('.weather-card')?.textContent.trim(), photo: !!document.querySelector('.destination-photo'), overflow: document.documentElement.scrollWidth })`);
      assert.equal(hero.title, city.name);
      assert.ok(hero.weather && hero.photo && hero.overflow <= width, JSON.stringify(hero));
      await shot(city.name.toLowerCase() + '-initial', width);

      await tabUntil(`.map-pin[title="${city.pin}"]`);
      await press('Enter');
      await browser.until(`document.querySelector('.map-popup[aria-label=${JSON.stringify(city.pin)}] .map-popup-details') && ${focusIs('.map-popup-details')}`);
      await press('Enter');
      try {
        await browser.until(`document.querySelector('.map-place-panel h3')?.textContent === ${JSON.stringify(city.pin)} && ${focusIs('.map-place-panel h3')}`);
      } catch (error) {
        console.error('PANEL WAIT', city.name, width, JSON.stringify(await browser.evaluate(`({
          panel: document.querySelector('[data-map-slot=panel]')?.textContent.slice(0, 200),
          active: document.activeElement?.outerHTML.slice(0, 300), errors: window.__previewErrors
        })`)));
        throw error;
      }
      await tabUntil('[data-save-qid]');
      assert.equal(await browser.evaluate("document.activeElement.dataset.saveQid"), city.qid);
      await press('Enter');
      await browser.until("document.querySelector('[data-map-slot=\"saved\"]')?.textContent.includes('1 saved place')");
      await tabUntil('.map-panel-writers summary');
      await press('Enter');
      const posts = await browser.evaluate(`(() => ({ expanded: document.querySelector('.map-panel-writers').open,
        links: [...document.querySelectorAll('.map-panel-writers a')].map(a => a.href) }))()`);
      assert.ok(posts.expanded && posts.links.some(url => url.includes(city.host)), JSON.stringify(posts));
      await shot(city.name.toLowerCase() + '-panel', width);

      await tabUntil('#trip-travelers');
      await press('ArrowDown');
      assert.equal(await browser.evaluate("document.querySelector('#trip-travelers').value"), '3');
      await tabUntil('.trip-submit');
      await press('Enter');
      await browser.until("document.querySelectorAll('[data-plan-stop]').length > 0 && document.querySelectorAll('.plan-pin').length > 0");
      assert.equal(await browser.evaluate("document.querySelector('#trip-travelers').value"), '3');
      assert.ok(await browser.evaluate(`([...document.querySelectorAll('[data-plan-stop]')].some(p => p.dataset.name === ${JSON.stringify(city.pin)}))`), 'saved place in plan');
      const day = await browser.evaluate(`(() => ({ dayPins: document.querySelectorAll('.plan-pin').length,
        stops: document.querySelectorAll('[data-plan-stop]').length,
        itineraryPins: document.querySelectorAll('.itinerary-pin').length,
        itineraryLink: document.querySelector('.itinerary-popup a')?.href,
        overflow: document.documentElement.scrollWidth }))()`);
      assert.equal(day.dayPins, day.stops);
      assert.ok(day.itineraryPins > 0 && day.overflow <= width, JSON.stringify(day));
      await shot(city.name.toLowerCase() + '-planned', width);

      await tabUntil('.itinerary-pin');
      await press('Enter');
      await browser.until("document.querySelector('.itinerary-popup a')");
      assert.ok((await browser.evaluate("document.querySelector('.itinerary-popup a').href")).includes(city.itineraryHost));
      assert.deepEqual(await browser.evaluate('window.__previewErrors'), []);
      assert.deepEqual(browser.errors, []);
      console.log(`PASS ${city.name} ${width}px: keyboard search, hero/weather, pin/panel/posts, save, 3 travelers, day layer, writer route; no overflow or browser errors`);
    }

    // A failed map source still leaves the map shell and the independent day planner usable.
    await browser.navigate(`${preview.url}/trip/coimbra-pt`);
    await revealMap();
    await browser.until("document.querySelector('[data-map-slot=\"status\"]')?.textContent.trim() === 'Some map sources are unavailable.' && document.querySelector('[data-map-canvas] .maplibregl-canvas, [data-map-canvas].leaflet-container')");
    await tabUntil('.trip-submit');
    await press('Enter');
    await browser.until("document.querySelectorAll('.plan-pin').length > 0");
    assert.ok(await browser.evaluate('document.documentElement.scrollWidth <= innerWidth'));
    assert.deepEqual(await browser.evaluate('window.__previewErrors'), []);
    await shot('coimbra-unavailable', width);
    console.log(`PASS Coimbra ${width}px: map source status visible, basemap and keyboard day plan usable`);
  }
} finally {
  await browser?.close?.();
  preview.stop();
}
