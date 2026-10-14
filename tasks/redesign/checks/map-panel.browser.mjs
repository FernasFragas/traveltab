#!/usr/bin/env node
// Start an isolated Brave/Chromium on CDP_PORT with about:blank, then run this check.
import assert from 'node:assert/strict';
import { connect, openPlacePanel, startPreview } from './lib/browser.mjs';

const browser = await connect();
try {
  await browser.call('Page.addScriptToEvaluateOnNewDocument', { source: `
    window.__previewErrors = [];
    console.error = (...args) => window.__previewErrors.push(args.map(String).join(' '));
    window.addEventListener('error', event => window.__previewErrors.push(event.message));
    window.addEventListener('unhandledrejection', event => window.__previewErrors.push(String(event.reason)));
  ` });
  for (const width of [375, 1440]) {
    const preview = await startPreview({ scenario: 'writers-map' });
    try {
      await browser.viewport(width, 900);
      await browser.navigate(`${preview.url}/trip/madeira-pt`);
      await browser.until("document.querySelector('.map-pin[title=\"Funchal\"]')");
      const initial = await browser.evaluate(`(() => ({
        weather: document.querySelector('.map-default-weather')?.textContent,
        about: document.querySelector('.map-default-guide h3')?.textContent,
        plan: document.querySelector('.map-default-plan')?.getAttribute('href')
      }))()`);
      assert.match(initial.weather, /°C/);
      assert.equal(initial.about, 'About Madeira');
      assert.equal(initial.plan, '#itinerary');
      assert.equal(await browser.evaluate("document.querySelectorAll('h1').length"), 1);
      assert.match(await browser.evaluate("document.querySelector('.writers-map-card > h2')?.textContent || ''"), /Map/);

      await openPlacePanel(browser, 'Funchal');
      await browser.until("document.querySelector('.map-place-panel')");
      if (width === 375) {
        const sheet = await browser.evaluate(`(() => { const e = document.querySelector('[data-map-slot=panel]'); return { open: e.classList.contains('is-sheet'), height: e.getBoundingClientRect().height, role: e.getAttribute('role') }; })()`);
        assert.ok(sheet.open && sheet.height >= 400 && sheet.height <= 500);
        assert.equal(sheet.role, 'dialog');
        await browser.evaluate("document.querySelector('.map-sheet-handle').focus()");
        await browser.key('Enter', 'Enter', 13);
        assert.ok(await browser.evaluate("document.querySelector('[data-map-slot=panel]').classList.contains('is-expanded')"));
        await browser.key(' ', 'Space', 32, 0, ' ');
        assert.equal(await browser.evaluate("document.querySelector('[data-map-slot=panel]').classList.contains('is-expanded')"), false);
        const handle = await browser.evaluate("document.querySelector('.map-sheet-handle').getBoundingClientRect().toJSON()");
        const x = handle.x + handle.width / 2, y = handle.y + handle.height / 2;
        await browser.call('Input.dispatchMouseEvent', { type: 'mouseMoved', x, y });
        await browser.call('Input.dispatchMouseEvent', { type: 'mousePressed', x, y, button: 'left', clickCount: 1 });
        await browser.call('Input.dispatchMouseEvent', { type: 'mouseMoved', x, y: y - 60, button: 'left', buttons: 1 });
        await browser.call('Input.dispatchMouseEvent', { type: 'mouseReleased', x, y: y - 60, button: 'left', clickCount: 1 });
        assert.ok(await browser.evaluate("document.querySelector('[data-map-slot=panel]').classList.contains('is-expanded')"), 'dragging up expands the sheet');
        await browser.evaluate("document.querySelector('.map-sheet-handle').click()");
        await browser.evaluate("document.querySelector('.map-panel-writers')?.remove(); document.querySelector('.map-panel-directions').focus()");
        await browser.key('Tab', 'Tab', 9);
        assert.ok(await browser.evaluate("document.activeElement.matches('.map-panel-hours summary')"), 'a place without posts still reaches Hours by keyboard');
        await browser.key('Tab', 'Tab', 9);
        assert.ok(await browser.evaluate("document.activeElement.classList.contains('map-sheet-handle')"), 'Tab from Hours wraps within the sheet');
      }
      const panel = await browser.evaluate(`(() => {
        const root = document.querySelector('.map-place-panel');
        return { name: root.querySelector('h3').textContent,
          add: root.querySelector('[data-add-qid]').textContent,
          save: root.querySelector('.map-panel-save').textContent,
          share: root.querySelector('[data-map-share]').textContent,
          overflow: document.documentElement.scrollWidth };
      })()`);
      assert.equal(panel.name, 'Funchal');
      assert.equal(panel.add, '+ Add to plan');
      assert.equal(panel.save, 'Save to visit');
      assert.equal(panel.share, 'Share');
      assert.ok(panel.overflow <= width);
      await browser.evaluate(`(() => {
        window.__copied = '';
        Object.defineProperty(navigator, 'share', { configurable: true, value: undefined });
        Object.defineProperty(navigator, 'clipboard', { configurable: true,
          value: { writeText: text => { window.__copied = text; return Promise.resolve(); } } });
        document.querySelector('[data-map-share]').click();
      })()`);
      await browser.until("document.querySelector('.map-panel-feedback').textContent === 'Link copied'");
      assert.match(await browser.evaluate('window.__copied'), /[?&]place=Q25444/);
      await browser.evaluate("document.querySelector('.map-panel-close').click()");
      if (width === 375) {
        assert.ok(await browser.evaluate("!document.querySelector('[data-map-slot=panel]').classList.contains('is-sheet')"));
        assert.equal(await browser.evaluate("document.querySelector('[data-map-slot=panel]').getBoundingClientRect().height"), 0, 'closing the sheet removes it from view');
      } else assert.ok(await browser.evaluate("Boolean(document.querySelector('.map-default-panel'))"));
      assert.deepEqual(await browser.evaluate('window.__previewErrors'), []);
      console.log(`PASS ${width}px: place panel, share and phone sheet`);
    } finally {
      preview.stop();
    }
  }
} finally {
  await browser.close();
}
