#!/usr/bin/env node
// Runs the 3a core against a hand-built shell on the real design-preview page (default scenario,
// flag off, so the page has the core loaded but no map shell of its own).
// Start an isolated Brave/Chromium on CDP_PORT with about:blank, then run this file.
import assert from 'node:assert/strict';
import { connect, startPreview } from './lib/browser.mjs';

const preview = await startPreview();
let browser;
try {
  browser = await connect();
  for (const width of [375, 1440]) {
    await browser.viewport(width, 900);
    await browser.navigate(`${preview.url}/trip/lisbon-pt`);
    await browser.evaluate(`(() => {
      window.__mapConsoleErrors = [];
      const originalConsoleError = console.error.bind(console);
      console.error = (...args) => {
        window.__mapConsoleErrors.push(args.map(String).join(' '));
        originalConsoleError(...args);
      };
      const host = document.querySelector('.tt-content');
      const shell = document.createElement('section');
      shell.dataset.writersMap = '';
      shell.dataset.mapLat = '32.65';
      shell.dataset.mapLon = '-16.9';
      shell.dataset.mapDestination = 'Madeira';
      shell.dataset.mapUrl = '/map/madeira.geojson';
      shell.innerHTML = '<div data-map-canvas></div><div data-map-slot="status"></div><div data-map-slot="list"></div>';
      host.appendChild(shell);
      const originalFetch = window.fetch.bind(window);
      window.fetch = (url, options) => String(url).startsWith('/map/')
        ? Promise.resolve({ ok: true, json: () => Promise.resolve({ complete: false, features: [
            { properties: { qid: 'Q1', name: 'Pico do Arieiro' } }
          ] }) })
        : originalFetch(url, options);
      window.L = {
        map: () => ({ setView() { return this; }, invalidateSize() {}, removeLayer() {} }),
        control: { scale: () => ({ addTo() {} }) },
        tileLayer: () => ({ addTo() {} })
      };
      // Test the core alone: the fake places have no geometry for the PR 7 pin layers.
      ['notable', 'writers', 'plan'].forEach(name => window.TravelMap.setLayerEnabled(name, false));
      window.TravelMap.mount();
    })()`);
    await browser.until("document.querySelector('[data-map-slot=\"list\"] button') !== null");
    const state = await browser.evaluate(`(() => {
      const shell = document.querySelector('[data-writers-map]');
      const canvas = shell.querySelector('[data-map-canvas]');
      const listButton = shell.querySelector('[data-map-slot="list"] button');
      listButton.focus();
      return {
        width: Math.round(canvas.getBoundingClientRect().width),
        height: Math.round(canvas.getBoundingClientRect().height),
        overflow: document.documentElement.scrollWidth,
        viewport: innerWidth,
        status: shell.querySelector('[data-map-slot="status"]').textContent,
        focused: document.activeElement === listButton,
        consoleErrors: window.__mapConsoleErrors
      };
    })()`);
    assert.ok(state.width > 0 && state.width <= width, JSON.stringify(state));
    assert.ok(state.height >= 320, JSON.stringify(state));
    assert.ok(state.overflow <= state.viewport, JSON.stringify(state));
    assert.equal(state.status, 'Some map sources are unavailable.');
    assert.equal(state.focused, true);
    assert.deepEqual(state.consoleErrors, []);
    assert.deepEqual(browser.errors, []);
    console.log(`PASS ${width}px: map shell layout, keyboard list, status, console`);
  }
} finally {
  if (browser) await browser.close();
  preview.stop();
}
