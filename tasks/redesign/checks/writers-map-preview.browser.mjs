#!/usr/bin/env node
// Start an isolated Brave/Chromium on CDP_PORT with about:blank, then run this check.
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
    await browser.until("document.querySelector('[data-map-slot=\"list\"] button') && document.querySelector('.leaflet-container')");
    await browser.until("document.querySelector('.leaflet-tile-loaded') !== null");
    browser.errors.length = 0; // Prior page teardown can report after the next navigation starts.
    const madeira = await browser.evaluate(`(() => {
      const shell = document.querySelector('[data-writers-map]');
      const list = shell.querySelector('[data-map-slot="list"] button');
      window.__mapCanvas = shell.querySelector('[data-map-canvas]');
      list.focus();
      return { destination: shell.dataset.mapDestination, text: list.textContent, focused: document.activeElement === list,
        tileURL: shell.dataset.mapTileUrl, width: Math.round(window.__mapCanvas.getBoundingClientRect().width),
        height: Math.round(window.__mapCanvas.getBoundingClientRect().height), overflow: document.documentElement.scrollWidth };
    })()`);
    assert.equal(madeira.destination, 'madeira-pt');
    assert.ok(madeira.text.length > 0 && madeira.focused, JSON.stringify(madeira));
    assert.match(madeira.tileURL, /^http:\/\/127\.0\.0\.1:\d+\/fixture-tiles\/%7bz%7d\/%7bx%7d\/%7by%7d\.png$/i);
    assert.ok(madeira.width > 0 && madeira.width <= width && madeira.height >= 320, JSON.stringify(madeira));
    assert.ok(madeira.overflow <= width, JSON.stringify(madeira));
    await browser.tab(true);
    await browser.tab();
    assert.equal(await browser.evaluate("document.activeElement === document.querySelector('[data-map-slot=\"list\"] button')"), true);

    for (const [query, slug, status] of [
      ['Porto', 'porto-pt', ''], ['Coimbra', 'coimbra-pt', 'Some map sources are unavailable.']
    ]) {
      await browser.evaluate(`(() => { document.querySelector('#city_name').value = ${JSON.stringify(query)}; document.querySelector('#destination-search').requestSubmit(); })()`);
      await browser.until(`document.querySelector('[data-writers-map]')?.dataset.mapDestination === ${JSON.stringify(slug)}`);
      await browser.until(`document.querySelector('[data-map-slot="status"]')?.textContent === ${JSON.stringify(status)}`);
      const state = await browser.evaluate(`(() => ({ maps: document.querySelectorAll('.leaflet-container').length,
        sameCanvas: document.querySelector('[data-map-canvas]') === window.__mapCanvas,
        list: [...document.querySelectorAll('[data-map-slot="list"] button')].map(b => b.textContent),
        overflow: document.documentElement.scrollWidth }))()`);
      assert.equal(state.maps, 1, JSON.stringify(state));
      assert.ok(state.sameCanvas && state.overflow <= width, JSON.stringify(state));
      if (slug === 'porto-pt') {
        assert.deepEqual(state.list, ['Clérigos Tower']);
        await browser.evaluate("document.querySelector('[data-map-slot=\"list\"] button').focus()");
        await browser.tab(true);
        await browser.tab();
        assert.equal(await browser.evaluate("document.activeElement === document.querySelector('[data-map-slot=\"list\"] button')"), true);
      }
    }
    assert.deepEqual(browser.errors, []);
    assert.deepEqual(await browser.evaluate('window.__previewErrors'), []);
    console.log(`PASS ${width}px: writer data, base-only and unavailable swaps, one map, keyboard list, local tiles`);
  }
} finally {
  if (browser) await browser.close();
  preview.stop();
}
