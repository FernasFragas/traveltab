#!/usr/bin/env node
// Check the M1 core on the deterministic preview with an isolated CDP browser.
import assert from 'node:assert/strict';
import { connect, startPreview } from './lib/browser.mjs';

const preview = await startPreview();
let browser;
try {
  browser = await connect();
  for (const width of [375, 1440]) {
    await browser.viewport(width, 900);
    for (const name of ['maplibre', 'leaflet']) {
      await browser.navigate(`${preview.url}/trip/lisbon-pt`);
      await browser.evaluate(`(() => {
        window.__mapConsoleErrors = [];
        window.__mapReadyEngine = '';
        document.addEventListener('travelmap:ready', event => { window.__mapReadyEngine = event.detail.engine; });
        const original = console.error.bind(console);
        console.error = (...args) => { window.__mapConsoleErrors.push(args.map(String).join(' ')); original(...args); };
        ['notable', 'writers', 'plan', 'itineraries'].forEach(name => {
          try { window.TravelMap.setLayerEnabled(name, false); } catch (_) {}
        });
        window.TravelMap.registerLayer('core-probe', { add: map => { window.__mapObject = map; }, remove() {} });
        const shell = document.createElement('section');
        shell.dataset.writersMap = '';
        shell.dataset.mapLat = '32.65';
        shell.dataset.mapLon = '-16.9';
        shell.dataset.mapDestination = 'Madeira';
        shell.dataset.mapUrl = '/map/madeira.geojson';
        shell.dataset.mapStyleUrl = ${JSON.stringify(preview.mapUrl + '/fixture-map/style.json')};
        shell.dataset.mapTerrainUrl = ${JSON.stringify(preview.mapUrl + '/fixture-map/terrain/{z}/{x}/{y}.png')};
        shell.dataset.mapTileUrl = ${JSON.stringify(preview.mapUrl + '/fixture-tiles/{z}/{x}/{y}.png')};
        if (${JSON.stringify(name)} === 'leaflet') {
          const getContext = HTMLCanvasElement.prototype.getContext;
          HTMLCanvasElement.prototype.getContext = function (type, ...args) {
            return type === 'webgl' || type === 'webgl2' ? null : getContext.call(this, type, ...args);
          };
        }
        shell.innerHTML = '<div data-map-canvas></div><div data-map-slot="status"></div><div data-map-slot="list"></div>';
        document.querySelector('.tt-content').appendChild(shell);
        const originalFetch = window.fetch.bind(window);
        window.fetch = (url, options) => String(url).startsWith('/map/')
          ? Promise.resolve({ ok: true, json: () => Promise.resolve({ complete: true, features: [] }) })
          : originalFetch(url, options);
        window.TravelMap.mount(shell);
        shell.scrollIntoView();
      })()`);
      await browser.until(`window.__mapReadyEngine === ${JSON.stringify(name)} && document.querySelector('[data-map-canvas] .${name === 'maplibre' ? 'maplibregl-canvas' : 'leaflet-pane'}') !== null`, 30000);
      if (name === 'maplibre') {
        await browser.until(`window.__mapObject?.querySourceFeatures('fixture', { sourceLayer: 'transportation' }).length > 0`, 15000);
        await browser.until(`performance.getEntriesByType('resource').some(item => item.name.includes('/fixture-map/terrain/'))`, 15000);
      }
      const state = await browser.evaluate(`(() => {
        const shell = document.querySelector('[data-writers-map]');
        const canvas = shell.querySelector('[data-map-canvas]');
        return { width: Math.round(canvas.getBoundingClientRect().width),
          height: Math.round(canvas.getBoundingClientRect().height),
          overflow: document.documentElement.scrollWidth, viewport: innerWidth,
          style: shell.dataset.mapStyleUrl, terrain: shell.dataset.mapTerrainUrl,
          engine: window.__mapReadyEngine, errors: window.__mapConsoleErrors };
      })()`);
      assert.ok(state.width > 0 && state.width <= width, JSON.stringify(state));
      assert.ok(state.height >= 320, JSON.stringify(state));
      assert.ok(state.overflow <= state.viewport, JSON.stringify(state));
      assert.match(state.style, /\/fixture-map\/style\.json$/);
      assert.match(state.terrain, /\/fixture-map\/terrain\//);
      assert.equal(state.engine, name);
      assert.deepEqual(state.errors, []);
      assert.deepEqual(browser.errors, []);
      console.log(`PASS ${name} ${width}px: canvas, local fixtures, layout, console`);
    }
  }
} finally {
  if (browser) await browser.close();
  preview.stop();
}
