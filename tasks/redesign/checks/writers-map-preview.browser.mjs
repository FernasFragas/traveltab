#!/usr/bin/env node
// Start an isolated Brave/Chromium on CDP_PORT with about:blank, then run this check.
import assert from 'node:assert/strict';
import { connect, startPreview } from './lib/browser.mjs';

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
    // Places answer late on purpose, so every wait below is proven to wait for the places and not to pass by luck.
    const originalFetch = window.fetch.bind(window);
    window.fetch = (input, init) => /geo\\+json/.test(JSON.stringify(init?.headers || {}))
      ? new Promise(resolve => setTimeout(resolve, 400)).then(() => originalFetch(input, init)) : originalFetch(input, init);
  ` });
  for (const width of [375, 1440]) {
    await browser.viewport(width, 900);
    await browser.navigate(`${preview.url}/trip/madeira-pt`);
    await browser.until(`document.querySelector('.map-pin') && document.querySelector('[data-map-canvas] .${engine === 'maplibre' ? 'maplibregl-canvas' : 'leaflet-pane'}')`);
    if (engine === 'leaflet') await browser.until("document.querySelector('.leaflet-tile-loaded') !== null");
    browser.errors.length = 0; // Prior page teardown can report after the next navigation starts.
    const madeira = await browser.evaluate(`(() => {
      const shell = document.querySelector('[data-writers-map]');
      const pin = shell.querySelector('.map-pin');
      window.__mapCanvas = shell.querySelector('[data-map-canvas]');
      pin.focus();
      const card = shell.getBoundingClientRect();
      const canvas = window.__mapCanvas.getBoundingClientRect();
      const panel = shell.querySelector('[data-map-slot="panel"]').getBoundingClientRect();
      const info = shell.querySelector('.writers-map-info');
      return { destination: shell.dataset.mapDestination, text: pin.title, focused: document.activeElement === pin, listSlot: shell.querySelector('[data-map-slot="list"]') !== null,
        tileURL: shell.dataset.mapTileUrl, styleURL: shell.dataset.mapStyleUrl,
        width: Math.round(window.__mapCanvas.getBoundingClientRect().width),
        height: Math.round(window.__mapCanvas.getBoundingClientRect().height), overflow: document.documentElement.scrollWidth,
        card: { x: card.x, width: card.width }, canvas: { x: canvas.x, y: canvas.y, right: canvas.right, bottom: canvas.bottom, width: canvas.width, height: canvas.height },
        panel: { x: panel.x, y: panel.y, width: panel.width, height: panel.height },
        info: { label: info?.getAttribute('aria-label'), description: document.getElementById(info?.getAttribute('aria-describedby'))?.textContent.trim() },
        layers: [...shell.querySelectorAll('[data-map-slot="layers"] > button')].map(el => el.textContent.trim()),
        emptyPanel: shell.querySelector('[data-map-slot="panel"]')?.textContent.trim(),
        panelVisible: getComputedStyle(shell.querySelector('[data-map-slot="panel"]')).display !== 'none',
        peekCount: shell.querySelectorAll('[data-map-panel-peek]').length };
    })()`);
    assert.equal(madeira.destination, 'madeira-pt');
    assert.ok(madeira.text.length > 0 && madeira.focused && !madeira.listSlot, JSON.stringify(madeira));
    assert.match(madeira[engine === 'maplibre' ? 'styleURL' : 'tileURL'], engine === 'maplibre'
      ? /^http:\/\/127\.0\.0\.1:\d+\/fixture-map\/style\.json$/
      : /^http:\/\/127\.0\.0\.1:\d+\/fixture-tiles\/%7bz%7d\/%7bx%7d\/%7by%7d\.png$/i);
    assert.ok(madeira.width > 0 && madeira.width <= width && madeira.height >= 320, JSON.stringify(madeira));
    assert.ok(madeira.overflow <= width, JSON.stringify(madeira));
    assert.equal(madeira.info.label, 'About map layers');
    assert.match(madeira.info.description || '', /notable places.*writers.*day plan.*itineraries/i);
    assert.deepEqual(madeira.layers, width === 375 ? ['Notable', 'Writers', 'Itineraries', 'Your plan'] : ['Notable places', "Writers' pins", "Writers' itineraries", 'Your plan']);
    assert.match(madeira.emptyPanel, /About Madeira/);
    assert.match(madeira.emptyPanel, /Plan your trip/);
    if (width === 1440) {
      assert.ok(madeira.card.width >= 1000, JSON.stringify(madeira));
      assert.ok(madeira.panel.width >= 380 && madeira.panel.width <= 420, JSON.stringify(madeira));
      assert.ok(madeira.panel.x >= madeira.canvas.right && Math.abs(madeira.panel.height - madeira.canvas.height) <= 1, JSON.stringify(madeira));
    } else {
      assert.ok(madeira.canvas.width >= madeira.card.width - 20 && madeira.canvas.width <= madeira.card.width);
      assert.equal(madeira.panelVisible, false);
      assert.equal(madeira.peekCount, 0);
    }
    await browser.tab(true);
    await browser.tab();
    assert.equal(await browser.evaluate("document.activeElement === document.querySelector('.map-pin')"), true);

    // The status is already '' before the places arrive, so a swap with no status message is done only once its pins are drawn.
    for (const [query, slug, status, loaded] of [
      ['Porto', 'porto-pt', '', "document.querySelector('.map-pin') !== null"],
      ['Coimbra', 'coimbra-pt', 'Some map sources are unavailable.', 'true']
    ]) {
      await browser.evaluate(`(() => { document.querySelector('#city_name').value = ${JSON.stringify(query)}; document.querySelector('#destination-search').requestSubmit(); })()`);
      await browser.until(`document.querySelector('[data-writers-map]')?.dataset.mapDestination === ${JSON.stringify(slug)}`);
      await browser.until(`document.querySelector('[data-map-slot="status"]')?.textContent === ${JSON.stringify(status)} && ${loaded}`);
      const state = await browser.evaluate(`(() => ({ maps: document.querySelectorAll('[data-map-canvas]').length,
        sameCanvas: document.querySelector('[data-map-canvas]') === window.__mapCanvas,
        list: [...document.querySelectorAll('.map-pin')].map(b => b.title),
        overflow: document.documentElement.scrollWidth }))()`);
      assert.equal(state.maps, 1, JSON.stringify(state));
      assert.ok(state.sameCanvas && state.overflow <= width, JSON.stringify(state));
      if (slug === 'porto-pt') {
        assert.deepEqual(state.list, ['Clérigos Tower']);
        await browser.evaluate("document.querySelector('.map-pin').focus()");
        await browser.tab(true);
        await browser.tab();
        assert.equal(await browser.evaluate("document.activeElement === document.querySelector('.map-pin')"), true);
      }
    }
    assert.deepEqual(browser.errors, []);
    assert.deepEqual(await browser.evaluate('window.__previewErrors'), []);
    console.log(`PASS ${engine} ${width}px: writer data, base-only and unavailable swaps, one map, focusable pins, no place list, local fixtures`);
  }
} finally {
  if (browser) await browser.close();
  preview.stop();
}
}
