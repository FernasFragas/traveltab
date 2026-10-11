#!/usr/bin/env node
// Start isolated Chromium/Brave on CDP_PORT, then run this live-provider check.
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
    await browser.until("document.querySelector('.leaflet-tile-loaded') !== null");
    await browser.evaluate(`(() => {
      const old = document.querySelector('[data-writers-map]');
      const next = old.cloneNode(false);
      next.innerHTML = '<div data-map-canvas></div><div data-map-slot="status"></div><div data-map-slot="list"></div>';
      delete next.dataset.mapTileUrl;
      delete next.dataset.mapAttribution;
      old.replaceWith(next);
      window.TravelMap.mount(next);
    })()`);
    await browser.until("[...document.querySelectorAll('.leaflet-tile-loaded')].some(t => t.src.includes('tiles.maps.eox.at'))", 20000);
    const state = await browser.evaluate(`(() => {
      const map = document.querySelector('.leaflet-container');
      const credit = map.querySelector('.leaflet-control-attribution');
      const scale = map.querySelector('.leaflet-control-scale');
      const creditRect = credit.getBoundingClientRect();
      const scaleRect = scale.getBoundingClientRect();
      return {
        src: map.querySelector('.leaflet-tile-loaded[src*="tiles.maps.eox.at"]').src,
        credit: credit.textContent,
        creditVisible: getComputedStyle(credit).display !== 'none' && credit.getBoundingClientRect().width > 0,
        controlsOverlap: scaleRect.left < creditRect.right && scaleRect.right > creditRect.left &&
          scaleRect.top < creditRect.bottom && scaleRect.bottom > creditRect.top,
        scaleRect: { x: scaleRect.x, y: scaleRect.y, width: scaleRect.width, height: scaleRect.height },
        creditRect: { x: creditRect.x, y: creditRect.y, width: creditRect.width, height: creditRect.height },
        overflow: document.documentElement.scrollWidth,
        viewport: innerWidth,
        errors: window.__previewErrors
      };
    })()`);
    assert.match(state.credit, /EOxCloudless.*EOX IT Services GmbH.*2016 & 2017.*CC BY 4.0/);
    assert.ok(state.creditVisible && state.overflow <= state.viewport, JSON.stringify(state));
    assert.equal(state.controlsOverlap, false, JSON.stringify(state));
    assert.match(state.src, /s2cloudless_3857/);
    assert.deepEqual(state.errors, []);
    assert.deepEqual(browser.errors, []);
    console.log(`PASS ${width}px: live EOX imagery, required credit visible, no overflow or console errors`);
  }
} finally {
  if (browser) await browser.close();
  preview.stop();
}
