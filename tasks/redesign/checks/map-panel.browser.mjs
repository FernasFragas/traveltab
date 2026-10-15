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
        about: document.querySelector('.map-default-guide p')?.textContent,
        plan: document.querySelector('.map-default-plan')?.getAttribute('href')
      }))()`);
      assert.equal(initial.weather, undefined);
      assert.ok(initial.about?.length > 0);
      assert.equal(initial.plan, '#itinerary');
      assert.equal(await browser.evaluate("document.querySelectorAll('h1').length"), 1);
      assert.match(await browser.evaluate("document.querySelector('.writers-map-card > h2')?.textContent || ''"), /Map/);

      await openPlacePanel(browser, 'Funchal');
      await browser.until("document.querySelector('.map-place-panel')");
      if (width === 375) {
        await browser.until("document.activeElement.matches('.map-place-panel h3')");
        assert.ok(await browser.evaluate("!document.querySelector('.is-sheet') && !document.querySelector('.map-sheet-handle')"));
        assert.equal(await browser.evaluate("document.querySelector('[data-map-slot=panel]').getAttribute('aria-modal')"), null);
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
      assert.equal(panel.add.trim(), 'Add to your plan');
      assert.equal(panel.save.trim(), 'Save');
      assert.equal(panel.share.trim(), 'Share');
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
      await browser.key('Escape', 'Escape', 27);
      assert.ok(await browser.evaluate("Boolean(document.querySelector('.map-default-panel'))"));
      assert.deepEqual(await browser.evaluate('window.__previewErrors'), []);
      console.log(`PASS ${width}px: inline place card, share and return to city`);
    } finally {
      preview.stop();
    }
  }
} finally {
  await browser.close();
}
