#!/usr/bin/env node
// Start an isolated Brave/Chromium on CDP_PORT with about:blank, then run this check.
import assert from 'node:assert/strict';
import { writeFile } from 'node:fs/promises';
import { connect, openPlacePanel, startPreview } from './lib/browser.mjs';

let browser;
try {
  browser = await connect();
  await browser.call('Page.addScriptToEvaluateOnNewDocument', { source: `
    window.__previewErrors = [];
    const originalError = console.error.bind(console);
    console.error = (...args) => { window.__previewErrors.push(args.map(String).join(' ')); originalError(...args); };
    document.addEventListener('travelmap:ready', event => { window.__map = event.detail.map; window.__engine = event.detail.engine; });
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
        // The view opens on the destination, not fitted to the pins, so writer pins exist but may lie outside it.
        // Pan to them, as a visitor would, before the click and keyboard steps below.
        assert.ok(await browser.evaluate("document.querySelectorAll('.map-pin-writer').length") > 0, 'writer pins are drawn');
        await browser.evaluate(`(async () => {
          const data = await (await fetch(document.querySelector('[data-map-url]').dataset.mapUrl)).json();
          const points = data.features.filter(f => f.properties.writer_count > 0).map(f => f.geometry.coordinates);
          const lons = points.map(p => p[0]), lats = points.map(p => p[1]);
          const [w, e, s, n] = [Math.min(...lons), Math.max(...lons), Math.min(...lats), Math.max(...lats)];
          if (window.__engine === 'maplibre') window.__map.fitBounds([[w, s], [e, n]], { padding: 88, maxZoom: 12.3, animate: false });
          else window.__map.fitBounds([[s, w], [n, e]], { padding: [48, 48], maxZoom: 12, animate: false });
        })()`);
        const inView = `(() => {
          const canvas = document.querySelector('[data-map-canvas]').getBoundingClientRect();
          return [...document.querySelectorAll('.map-pin-writer')].filter(pin => { const r = pin.getBoundingClientRect();
            return r.width > 0 && r.left >= canvas.left && r.right <= canvas.right && r.top >= canvas.top && r.bottom <= canvas.bottom; }).length;
        })()`;
        await browser.until(`${inView} === document.querySelectorAll('.map-pin-writer').length`, 15000);
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

        assert.equal(await browser.evaluate("document.querySelector('[data-map-slot=\"list\"]')"), null, 'no place-tag list');
        assert.equal(await browser.evaluate("document.querySelectorAll('[data-map-slot=\"list\"] button, [data-map-canvas] ~ ul button').length"), 0);

        // Keyboard: Tab reaches a pin, Enter opens its popup (focus moves in), Escape closes it and returns to the pin.
        await browser.evaluate("document.querySelector('.writers-map-info').focus()");
        for (let i = 0; i < 40 && !(await browser.evaluate("document.activeElement.classList.contains('map-pin')")); i++) await browser.tab();
        assert.ok(await browser.evaluate("document.activeElement.classList.contains('map-pin')"), 'Tab reaches a pin');
        const tabbed = await browser.evaluate("document.activeElement.title");
        await browser.key('Enter', 'Enter', 13, 0, '\r');
        await browser.until("document.querySelector('.map-popup .map-popup-details')");
        const keyboardPopup = await browser.evaluate(`(() => { const box = document.querySelector('.map-popup');
          return { name: box.getAttribute('aria-label'), role: box.getAttribute('role'), focusedDetails: document.activeElement === box.querySelector('.map-popup-details'), popups: document.querySelectorAll('.map-popup').length }; })()`);
        assert.deepEqual(keyboardPopup, { name: tabbed, role: 'dialog', focusedDetails: true, popups: 1 });
        await browser.key('Escape', 'Escape', 27);
        await browser.until("document.querySelectorAll('.map-popup').length === 0"); // Leaflet fades it out first
        assert.equal(await browser.evaluate("document.activeElement.title"), tabbed, 'focus returns to the pin');

        // Mouse: a real click on the first writer pin opens the popup and it stays open.
        const clickAt = async (x, y) => {
          await browser.call('Input.dispatchMouseEvent', { type: 'mousePressed', x, y, button: 'left', clickCount: 1 });
          await browser.call('Input.dispatchMouseEvent', { type: 'mouseReleased', x, y, button: 'left', clickCount: 1 });
        };
        const centre = selector => browser.evaluate(`(() => { const el = document.querySelector(${JSON.stringify(selector)}); el.scrollIntoView({ block: 'center', behavior: 'instant' });
          const r = el.getBoundingClientRect(); return { x: r.x + r.width / 2, y: r.y + r.height / 2 }; })()`);
        const first = await centre('.map-pin-writer');
        await clickAt(first.x, first.y);
        await browser.until("document.querySelector('.map-popup .map-popup-details')");
        await new Promise(resolve => setTimeout(resolve, 400));
        const mousePopup = await browser.evaluate(`(() => { const box = document.querySelector('.map-popup'); const rect = box.getBoundingClientRect(); const canvas = document.querySelector('[data-map-canvas]').getBoundingClientRect();
          return { text: box.innerText.replace(/\\s+/g, ' '), popups: document.querySelectorAll('.map-popup').length, monograms: box.querySelectorAll('.map-panel-monogram').length,
            inside: rect.left >= canvas.left - 1 && rect.right <= canvas.right + 1, panelOpen: Boolean(document.querySelector('.map-place-panel')) }; })()`);
        assert.equal(mousePopup.popups, 1, JSON.stringify(mousePopup));
        assert.match(mousePopup.text, /Mentioned by \d+ writers?/, JSON.stringify(mousePopup));
        assert.ok(mousePopup.monograms > 0 && mousePopup.inside && !mousePopup.panelOpen, JSON.stringify(mousePopup));
        if (process.env.SHOT_DIR) {
          const card = await browser.evaluate("(() => { const r = document.querySelector('[data-writers-map]').getBoundingClientRect(); return { x: r.x, y: r.y + scrollY, width: r.width, height: r.height }; })()");
          await writeFile(`${process.env.SHOT_DIR}/map-popup-${scenario}-${width}.png`, await browser.shot(card));
        }
        // Another pin replaces it; a click on the empty map closes it.
        await browser.evaluate("document.querySelector('.map-pin-notable').click()");
        await browser.until("document.querySelectorAll('.map-popup').length === 1 && !document.querySelector('.map-popup .map-panel-monograms')");
        // Find a spot on the bare map: its own canvas or tile is the topmost element (not a pin, popup or control).
        const emptyArea = await browser.evaluate(`(() => { const canvas = document.querySelector('[data-map-canvas]'); canvas.scrollIntoView({ block: 'center', behavior: 'instant' }); const r = canvas.getBoundingClientRect();
          for (let fy = 0.9; fy > 0.1; fy -= 0.1) for (let fx = 0.1; fx < 0.95; fx += 0.1) {
            const x = r.x + r.width * fx, y = r.y + r.height * fy;
            if (document.elementFromPoint(x, y)?.matches('canvas.maplibregl-canvas, .leaflet-tile, .leaflet-container')) return { x, y };
          }
          return null; })()`);
        assert.ok(emptyArea, 'a bare spot on the map');
        await clickAt(emptyArea.x, emptyArea.y);
        await browser.until("document.querySelectorAll('.map-popup').length === 0");
        // Details opens the panel, closes the popup and moves focus to the place name.
        await browser.evaluate("document.querySelector('.map-pin-writer').click()");
        await browser.until("document.querySelector('.map-popup-details')");
        await browser.key('Enter', 'Enter', 13, 0, '\r');
        await browser.until("document.querySelector('.map-place-panel h3')");
        await browser.until("document.querySelectorAll('.map-popup').length === 0");
        assert.equal(await browser.evaluate("document.activeElement.tagName"), 'H3', 'focus moves to the place name in the panel');
        await browser.evaluate("document.querySelector('.map-panel-close').click()");

        await browser.tab();
        const focus = await browser.evaluate(`(() => {
          const pin = [...document.querySelectorAll('.map-pin')].find(b => b.title === 'Pico Ruivo');
          pin.focus();
          return { active: document.activeElement === pin, outline: getComputedStyle(pin).outlineStyle };
        })()`);
        assert.ok(focus.active && focus.outline !== 'none', JSON.stringify(focus));
        await openPlacePanel(browser, 'Pico Ruivo');
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
