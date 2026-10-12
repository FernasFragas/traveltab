#!/usr/bin/env node
// Start isolated Chromium/Brave on CDP_PORT, then run this optional live-provider check.
import assert from 'node:assert/strict';
import { connect, startPreview } from './lib/browser.mjs';

for (const [engine, scenario] of [['maplibre', 'writers-map'], ['leaflet', 'writers-map-leaflet']]) {
  const preview = await startPreview({ scenario, liveTiles: engine === 'maplibre' });
  let browser;
  try {
    browser = await connect();
    await browser.call('Page.addScriptToEvaluateOnNewDocument', { source: `
      window.__previewErrors = [];
      const originalError = console.error.bind(console);
      console.error = (...args) => { window.__previewErrors.push(args.map(String).join(' ')); originalError(...args); };
      window.addEventListener('error', event => window.__previewErrors.push(event.message));
      window.addEventListener('unhandledrejection', event => window.__previewErrors.push(String(event.reason)));
      document.addEventListener('travelmap:ready', event => { window.__liveMap = event.detail.map; window.__liveEngine = event.detail.engine; });
    ` });
    for (const width of [375, 1440]) {
      const networkStart = browser.events.length;
      await browser.viewport(width, 900);
      await browser.navigate(`${preview.url}/trip/madeira-pt`);
      await browser.evaluate("document.querySelector('[data-map-canvas]')?.scrollIntoView({ block: 'center', behavior: 'instant' })");
      if (engine === 'leaflet') {
        await browser.until("document.querySelector('.leaflet-tile-loaded')");
        await browser.evaluate(`(() => {
          const old = document.querySelector('[data-writers-map]');
          const next = old.cloneNode(false);
          next.innerHTML = '<div data-map-canvas></div><div data-map-slot="status"></div><div data-map-slot="list"></div>';
          delete next.dataset.mapTileUrl;
          delete next.dataset.mapAttribution;
          old.replaceWith(next);
          window.TravelMap.mount(next);
        })()`);
      }
      await browser.until(`window.__liveEngine === ${JSON.stringify(engine)}`, 30000);
      if (engine === 'maplibre') {
        await browser.until('window.__liveMap?.isStyleLoaded() && window.__liveMap?.areTilesLoaded()', 30000);
        const state = await browser.evaluate(`(() => {
          const map = window.__liveMap;
          const canvas = document.querySelector('[data-map-canvas]');
          const credit = canvas.querySelector('.maplibregl-ctrl-attrib');
          return { style: JSON.stringify(map.getStyle().sources), terrain: map.getTerrain(),
            credit: credit?.textContent, creditVisible: !!credit && credit.getBoundingClientRect().width > 0,
            overflow: document.documentElement.scrollWidth, viewport: innerWidth,
            errors: window.__previewErrors };
        })()`);
        assert.match(state.style, /tiles\.openfreemap\.org/);
        assert.equal(state.terrain.source, 'terrain-dem');
        assert.match(state.credit, /OpenFreeMap|OpenMapTiles|OpenStreetMap/);
        assert.ok(state.creditVisible && state.overflow <= state.viewport, JSON.stringify(state));
        assert.deepEqual(state.errors, []);
        const events = browser.events.slice(networkStart);
        const responses = new Map(events.filter(e => e.method === 'Network.responseReceived' &&
          /cdn\.jsdelivr\.net\/npm\/maplibre-gl|tiles\.openfreemap\.org|tiles\.mapterhorn\.com/.test(e.params.response.url))
          .map(e => [e.params.requestId, e.params.response.url]));
        const finished = events.filter(e => e.method === 'Network.loadingFinished' && responses.has(e.params.requestId));
        const bytes = finished.reduce((total, e) => total + e.params.encodedDataLength, 0);
        console.log(`OBSERVED maplibre ${width}px: ${finished.length} page-target map responses, ${bytes} encoded bytes`);
        console.log(`PASS maplibre ${width}px: live OpenFreeMap style and Mapterhorn terrain loaded, credit visible`);
      } else {
        await browser.until("[...document.querySelectorAll('.leaflet-tile-loaded')].some(t => t.src.includes('tiles.maps.eox.at'))", 30000);
        const state = await browser.evaluate(`(() => {
          const canvas = document.querySelector('[data-map-canvas]');
          const credit = canvas.querySelector('.leaflet-control-attribution');
          return { src: canvas.querySelector('.leaflet-tile-loaded[src*="tiles.maps.eox.at"]').src,
            credit: credit.textContent, creditVisible: credit.getBoundingClientRect().width > 0,
            overflow: document.documentElement.scrollWidth, viewport: innerWidth,
            errors: window.__previewErrors };
        })()`);
        assert.match(state.src, /s2cloudless_3857/);
        assert.match(state.credit, /EOxCloudless.*EOX IT Services GmbH.*2016 & 2017.*CC BY 4.0/);
        assert.ok(state.creditVisible && state.overflow <= state.viewport, JSON.stringify(state));
        assert.deepEqual(state.errors, []);
        console.log(`PASS leaflet ${width}px: live EOX imagery and credit visible`);
      }
      assert.deepEqual(browser.errors, []);
    }
  } finally {
    if (browser) await browser.close();
    preview.stop();
  }
}
