#!/usr/bin/env node
// Start an isolated Brave/Chromium on CDP_PORT with about:blank, then run this check.
import assert from 'node:assert/strict';
import { mkdir, writeFile } from 'node:fs/promises';
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
    await browser.until("document.querySelector('.leaflet-container') && document.querySelector('[data-map-slot=\"list\"] button')");
    browser.errors.length = 0; // Prior page teardown can report after navigation starts.
    const layout = await browser.evaluate(`(() => {
      const shell = document.querySelector('[data-writers-map]');
      const rect = element => {
        const { x, y, width, height } = element.getBoundingClientRect();
        return { x, y, width, height };
      };
      const info = shell.querySelector('[aria-describedby]');
      const tooltip = info && document.getElementById(info.getAttribute('aria-describedby'));
      return {
        card: rect(shell), canvas: rect(shell.querySelector('[data-map-canvas]')),
        panel: rect(shell.querySelector('[data-map-slot="panel"]')),
        layers: rect(shell.querySelector('[data-map-slot="layers"]')),
        title: shell.querySelector('h3')?.textContent.trim(),
        info: info?.getAttribute('aria-label'),
        tooltip: info && document.getElementById(info.getAttribute('aria-describedby'))?.textContent.trim(),
        tooltipRect: tooltip && rect(tooltip),
        statusHidden: shell.querySelector('[data-map-slot="status"]').hidden,
        statusText: shell.querySelector('[data-map-slot="status"]').textContent.trim(),
        overflow: document.documentElement.scrollWidth,
        zoom: !!shell.querySelector('.leaflet-control-zoom'),
        scale: !!shell.querySelector('.leaflet-control-scale')
      };
    })()`);
    assert.equal(layout.title, "Writers' Map");
    assert.equal(layout.info, 'About map layers');
    assert.match(layout.tooltip, /notable places.*writers.*day plan.*itineraries/i);
    assert.ok(layout.tooltipRect.x >= 0 && layout.tooltipRect.x + layout.tooltipRect.width <= width, JSON.stringify(layout));
    const tooltipState = () => browser.evaluate(`(() => {
      const button = document.querySelector('.writers-map-info');
      const tooltip = document.getElementById(button.getAttribute('aria-describedby'));
      return { expanded: button.getAttribute('aria-expanded'), visible: getComputedStyle(tooltip).visibility };
    })()`);
    assert.deepEqual(await tooltipState(), { expanded: 'false', visible: 'hidden' });
    await browser.evaluate("document.querySelector('.writers-map-info').focus()");
    assert.equal((await tooltipState()).visible, 'visible');
    await browser.evaluate("document.querySelector('.writers-map-info').dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))");
    assert.equal((await tooltipState()).visible, 'hidden');
    await browser.evaluate("document.querySelector('.writers-map-info').blur()");
    assert.equal((await tooltipState()).visible, 'hidden');
    await browser.evaluate("document.querySelector('.writers-map-info').click()");
    assert.deepEqual(await tooltipState(), { expanded: 'true', visible: 'visible' });
    await browser.key('Escape', 'Escape', 27);
    assert.deepEqual(await tooltipState(), { expanded: 'false', visible: 'hidden' });
    assert.equal(layout.statusText, 'Some map sources are unavailable.');
    assert.ok(!layout.statusHidden && layout.zoom && layout.scale, JSON.stringify(layout));
    assert.ok(layout.layers.width > 0 && layout.layers.y < layout.canvas.y, JSON.stringify(layout));
    assert.ok(layout.overflow <= width, JSON.stringify(layout));
    if (width === 1440) {
      assert.ok(layout.card.width > 1000, JSON.stringify(layout));
      assert.ok(layout.canvas.x < layout.panel.x && Math.abs(layout.canvas.y - layout.panel.y) < 3, JSON.stringify(layout));
      assert.ok(layout.canvas.width / (layout.canvas.width + layout.panel.width) > 0.62, JSON.stringify(layout));
      assert.ok(Math.abs(layout.canvas.height - layout.panel.height) < 3, JSON.stringify(layout));
      const tallPanel = await browser.evaluate(`(() => {
        const shell = document.querySelector('[data-writers-map]');
        const panel = shell.querySelector('[data-map-slot="panel"]');
        panel.innerHTML = '<div style="height:600px">Tall place details</div>';
        const result = { canvas: shell.querySelector('[data-map-canvas]').getBoundingClientRect().height,
          panel: panel.getBoundingClientRect().height };
        panel.innerHTML = '<p>Pick a place on the map</p>';
        return result;
      })()`);
      assert.ok(Math.abs(tallPanel.canvas - tallPanel.panel) < 3, JSON.stringify(tallPanel));
    } else {
      assert.ok(layout.panel.y >= layout.canvas.y + layout.canvas.height, JSON.stringify(layout));
    }
    assert.deepEqual(browser.errors, []);
    assert.deepEqual(await browser.evaluate('window.__previewErrors'), []);
    if (process.env.OUTPUT_DIR) {
      await mkdir(process.env.OUTPUT_DIR, { recursive: true });
      await writeFile(`${process.env.OUTPUT_DIR}/${width}.png`, await browser.shot({
        x: 0, y: Math.floor(layout.card.y), width, height: Math.ceil(layout.card.height)
      }));
    }
    await browser.evaluate("document.querySelector('#city_name').value = 'Porto'; document.querySelector('#destination-search').requestSubmit()");
    await browser.until("document.querySelector('[data-writers-map]')?.dataset.mapDestination === 'porto-pt' && document.querySelector('[data-map-slot=\"status\"]')?.hidden");
    const swapped = await browser.evaluate(`(() => {
      const shell = document.querySelector('[data-writers-map]');
      const canvas = shell.querySelector('[data-map-canvas]').getBoundingClientRect();
      const panel = shell.querySelector('[data-map-slot="panel"]').getBoundingClientRect();
      return { width: shell.getBoundingClientRect().width, canvasY: canvas.y, canvasRight: canvas.right,
        panelY: panel.y, panelX: panel.x, overflow: document.documentElement.scrollWidth };
    })()`);
    assert.ok(swapped.overflow <= width, JSON.stringify(swapped));
    if (width === 1440) {
      assert.ok(swapped.width > 1000 && swapped.panelX >= swapped.canvasRight, JSON.stringify(swapped));
    } else {
      assert.ok(swapped.panelY > swapped.canvasY, JSON.stringify(swapped));
    }
    console.log(`PASS ${width}px: writers' map card layout`);
  }
} finally {
  if (browser) await browser.close();
  preview.stop();
}
