#!/usr/bin/env node
// Start an isolated Brave/Chromium on CDP_PORT with about:blank, then run this check.
import assert from 'node:assert/strict';
import { writeFile } from 'node:fs/promises';
import { connect, startPreview } from './lib/browser.mjs';

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
  for (const scenario of ['writers-map', 'writers-map-leaflet']) {
    const preview = await startPreview({ scenario });
    try {
      for (const width of [375, 1440]) {
        await browser.viewport(width, 900);
        await browser.navigate(`${preview.url}/trip/madeira-pt`);
        await browser.until("document.querySelectorAll('.map-pin').length > 0");
        const pins = await browser.evaluate(`(() => ({
          engine: document.querySelector('${scenario === 'writers-map-leaflet' ? '.leaflet-pane' : '.maplibregl-canvas'}') ? '${scenario === 'writers-map-leaflet' ? 'leaflet' : 'maplibre'}' : '',
          writers: [...document.querySelectorAll('.map-pin-writer')].map(pin => pin.querySelector('span').textContent),
          notable: document.querySelectorAll('.map-pin-notable').length,
          labels: document.querySelectorAll('.map-pin-label').length,
          writerZ: Math.min(...[...document.querySelectorAll('.map-pin-writer')].map(pin => Number((pin.closest('.leaflet-marker-icon') || pin).style.zIndex))),
          notableZ: Math.max(...[...document.querySelectorAll('.map-pin-notable')].map(pin => Number((pin.closest('.leaflet-marker-icon') || pin).style.zIndex))),
          focusable: [...document.querySelectorAll('.map-pin')].every(pin => pin.tagName === 'BUTTON' && pin.title && pin.getAttribute('aria-label') === pin.title)
        }))()`);
        assert.equal(pins.engine, scenario === 'writers-map-leaflet' ? 'leaflet' : 'maplibre', JSON.stringify(pins));
        assert.ok(pins.writers.length > 0 && pins.writers.every(count => Number(count) > 0), JSON.stringify(pins));
        assert.ok(pins.notable > 0 && pins.labels === 5, JSON.stringify(pins));
        assert.ok(pins.writerZ > pins.notableZ, JSON.stringify(pins));
        assert.ok(pins.focusable, JSON.stringify(pins));

        await browser.tab();
        const focus = await browser.evaluate(`(() => {
          const pin = [...document.querySelectorAll('.map-pin')].find(b => b.title === 'Pico Ruivo');
          pin.focus();
          return { active: document.activeElement === pin, outline: getComputedStyle(pin).outlineStyle };
        })()`);
        assert.ok(focus.active && focus.outline !== 'none', JSON.stringify(focus));
        await browser.evaluate("[...document.querySelectorAll('.map-pin')].find(b => b.title === 'Pico Ruivo').click()");
        await browser.until("document.querySelector('.map-place-panel .map-panel-monogram')");
        const panel = await browser.evaluate(`(() => {
          const root = document.querySelector('.map-place-panel');
          root.querySelector('.map-panel-writers summary').click();
          const box = root.getBoundingClientRect();
          return {
            name: root.querySelector('h3').textContent,
            monograms: [...root.querySelectorAll('.map-panel-monogram')].map(m => m.textContent),
            summary: Boolean(root.querySelector('.map-panel-summary')),
            posts: [...root.querySelectorAll('.map-panel-writers a')].filter(a => a.offsetParent && a.href).length,
            width: box.width, overflow: document.documentElement.scrollWidth
          };
        })()`);
        assert.equal(panel.name, 'Pico Ruivo');
        assert.equal(panel.monograms.length, 2, JSON.stringify(panel));
        assert.ok(panel.summary && panel.posts === 2, JSON.stringify(panel));
        assert.ok(panel.overflow <= width, JSON.stringify(panel));
        if (process.env.SHOT_DIR) {
          const card = await browser.evaluate("(() => { const r = document.querySelector('[data-writers-map]').getBoundingClientRect(); return { x: r.x, y: r.y + scrollY, width: r.width, height: r.height }; })()");
          await writeFile(`${process.env.SHOT_DIR}/map-panel-${scenario}-${width}.png`, await browser.shot(card));
        }

        await browser.evaluate("document.querySelector('.map-panel-close').click()");
        assert.equal(await browser.evaluate("document.querySelector('[data-map-slot=\"panel\"]').textContent.trim()"), 'Pick a place on the map');
        assert.deepEqual(await browser.evaluate('window.__previewErrors'), []);
        console.log(`ok ${scenario} ${width}px`, JSON.stringify({ pins, panel }));
      }
    } finally {
      preview.stop();
    }
  }
} finally {
  await browser?.close?.();
}
