#!/usr/bin/env node
// Start an isolated Brave/Chromium on CDP_PORT with about:blank, then run this check.
import assert from 'node:assert/strict';
import { connect, startPreview } from './lib/browser.mjs';

const browser = await connect();
try {
  for (const [engine, scenario] of [['maplibre', 'writers-map'], ['leaflet', 'writers-map-leaflet']]) {
    const preview = await startPreview({ scenario });
    try {
      for (const width of [375, 1440]) {
        await browser.viewport(width, 900);
        await browser.navigate(`${preview.url}/trip/madeira-pt`);
        await browser.until("document.querySelector('[data-map-canvas] .maplibregl-canvas, [data-map-canvas] .leaflet-pane')");
        browser.errors.length = 0;
        const top = await browser.evaluate(`(() => {
          const banner = document.querySelector('#overview .destination-banner');
          const box = banner?.getBoundingClientRect();
          return { footer: getComputedStyle(document.querySelector('.tt-footer')).display,
            header: document.querySelector('.tt-header').getBoundingClientRect().bottom,
            layers: document.querySelector('[data-map-slot="layers"]').getBoundingClientRect().top,
            banner: banner ? { top: box.top, bottom: box.bottom,
              title: banner.querySelector('.destination-title')?.textContent.trim(),
              weather: !!banner.querySelector('.weather-metrics'), photo: !!banner.querySelector('.destination-photo') } : null };
        })()`);
        assert.equal(top.footer, 'none', `the creator footer is replaced on the map page: ${JSON.stringify(top)}`);
        assert.ok(top.banner && top.banner.title && top.banner.weather && top.banner.photo, `city banner with photo, name and weather: ${JSON.stringify(top)}`);
        assert.ok(top.banner.top >= top.header - 1 && top.banner.bottom <= top.layers, `banner sits between the header and the map: ${JSON.stringify(top)}`);
        await browser.evaluate("window.scrollTo({ top: scrollY + document.querySelector('.writers-map-card').getBoundingClientRect().top, behavior: 'instant' })");
        const layout = await browser.evaluate(`(() => {
          const box = element => {
            const { x, y, width, height, bottom, right } = element.getBoundingClientRect();
            return { x, y, width, height, bottom, right };
          };
          return {
            header: box(document.querySelector('.tt-header')),
            canvas: box(document.querySelector('[data-map-canvas]')),
            panel: box(document.querySelector('[data-map-slot="panel"]')),
            nav: box(document.querySelector('.tt-bottom-navigation')),
            teasers: box(document.querySelector('.overview-teasers')),
            layers: box(document.querySelector('[data-map-slot="layers"]')),
            mapHeader: box(document.querySelector('.writers-map-header')),
            status: (() => { const element = document.querySelector('[data-map-slot="status"]'); return element.hidden ? null : box(element); })(),
            controls: Array.from(document.querySelectorAll('.travelmap-tools button')).map(box),
            overflow: document.documentElement.scrollWidth
          };
        })()`);
        assert.ok(layout.overflow <= width, JSON.stringify(layout));
        for (const control of layout.controls) {
          assert.ok(control.width >= 44 && control.height >= 44, JSON.stringify(control));
          assert.ok(control.y >= layout.canvas.y - 1 && control.bottom <= layout.canvas.bottom + 1, JSON.stringify(layout));
        }
        if (width === 1440) {
          assert.ok(layout.canvas.y >= layout.layers.bottom && layout.canvas.height >= 480 && layout.canvas.height <= 520, JSON.stringify(layout));
          assert.ok(layout.panel.width >= 490 && layout.panel.width <= 530, JSON.stringify(layout));
          assert.ok(layout.panel.x >= layout.canvas.right + 12, JSON.stringify(layout));
          assert.ok(layout.panel.y <= layout.layers.y + 2 && layout.panel.bottom >= layout.teasers.bottom - 2, JSON.stringify(layout));
          assert.ok(layout.teasers.y >= layout.canvas.bottom && layout.teasers.y <= layout.canvas.bottom + 24, JSON.stringify(layout));
          const scroll = await browser.evaluate(`(() => {
            const panel = document.querySelector('[data-map-slot="panel"]');
            const canvas = document.querySelector('[data-map-canvas]');
            const before = canvas.getBoundingClientRect().height;
            panel.innerHTML = '<div style="height:1800px">Long details</div>';
            return { canvas: canvas.getBoundingClientRect().height, panel: panel.getBoundingClientRect().height,
              overflow: getComputedStyle(panel).overflowY, scrollHeight: panel.scrollHeight, before };
          })()`);
          assert.equal(scroll.canvas, scroll.before);
          assert.ok(scroll.scrollHeight > scroll.panel && scroll.overflow === 'auto', JSON.stringify(scroll));
        } else {
          assert.ok(layout.canvas.y >= layout.header.bottom - 2, JSON.stringify(layout));
          assert.ok(layout.canvas.height >= 190 && layout.canvas.height <= 220, JSON.stringify(layout));
          assert.ok(layout.panel.y >= layout.canvas.bottom && layout.panel.y <= layout.canvas.bottom + 12, JSON.stringify(layout));
          assert.ok(layout.teasers.y >= layout.panel.bottom, JSON.stringify(layout));
          assert.ok(layout.layers.y >= layout.header.bottom - 2 && layout.layers.bottom <= layout.canvas.y, JSON.stringify(layout));
          if (layout.status) assert.ok(layout.status.y >= layout.layers.bottom, `map status covers layer controls: ${JSON.stringify(layout)}`);
        }
        assert.deepEqual(browser.errors, []);
        console.log(`PASS ${engine} ${width}px: short map with inline place card and teasers`);
      }
    } finally {
      preview.stop();
    }
  }
} finally {
  await browser.close();
}
