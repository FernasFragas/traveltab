#!/usr/bin/env node
// Map-first journey in the deterministic design preview, at phone and desktop widths.
// Attach to an isolated Brave/Chromium on CDP_PORT with an about:blank tab.
import assert from 'node:assert/strict';
import { mkdir, writeFile } from 'node:fs/promises';
import { connect, startPreview } from './lib/browser.mjs';

const out = process.env.OUTPUT_DIR || 'docs/tdd/artifacts/map-first';
const preview = await startPreview({ scenario: 'writers-map' });
let browser;

try {
  await mkdir(out, { recursive: true });
  browser = await connect();
  await browser.call('Page.addScriptToEvaluateOnNewDocument', { source: `
    window.__previewErrors = [];
    window.addEventListener('error', event => window.__previewErrors.push(event.message));
    window.addEventListener('unhandledrejection', event => window.__previewErrors.push(String(event.reason)));
  ` });
  for (const width of [375, 1440]) {
    await browser.viewport(width, 900);
    await browser.navigate(`${preview.url}/`);
    await browser.evaluate('localStorage.clear(); sessionStorage.clear()');
    await browser.navigate(`${preview.url}/`);
    await browser.evaluate("document.querySelector('#city_name').value = 'Funchal'; document.querySelector('#city_name').focus()");
    await browser.key('Enter', 'Enter', 13);
    await browser.until("document.querySelector('[data-writers-map]')?.dataset.mapDestination === 'funchal-pt'");
    await browser.until("Boolean(document.querySelector('.travelmap-cluster-list button[data-qid=\"Q25444\"], .map-pin[title=\"Funchal\"]'))");
    await browser.until("document.querySelectorAll('.map-pin-notable').length > 0");
    assert.ok(await browser.evaluate("!!document.querySelector('.map-pin-nature') && !!document.querySelector('.map-pin-notable .bi-bank') && !!document.querySelector('.map-pin-notable .bi-cup-hot')"), 'fixture shows nature, museum and food pin kinds');
    await browser.settle();
    assert.equal(await browser.evaluate("document.querySelector('#overview').hidden"), false);
    assert.ok(await browser.evaluate('document.documentElement.scrollWidth <= innerWidth'));
    await writeFile(`${out}/funchal-map-${width}.png`, await browser.shot());

    await browser.evaluate(`(() => {
      const pin = document.querySelector('.travelmap-cluster-list button[data-qid="Q25444"], .map-pin-writer[title="Funchal"]');
      pin.focus(); pin.click();
    })()`);
    await browser.until("Boolean(document.querySelector('.map-popup-details'))");
    await browser.evaluate("document.querySelector('.map-popup-details').click()");
    await browser.until("document.querySelector('.map-place-panel h3')?.textContent.trim() === 'Funchal'");
    if (width === 375) {
      await browser.until("document.querySelector('[data-map-slot=panel]').classList.contains('is-sheet')");
    }
    assert.ok(await browser.evaluate("document.querySelector('.map-panel-writers')?.textContent.includes('Writers')"));
    await browser.until("document.querySelector('.map-panel-photo img')?.naturalWidth > 0");
    assert.ok(await browser.evaluate("document.querySelector('.map-panel-mentioned').textContent.includes('Featured in 2') && document.querySelector('.map-panel-mentioned').textContent.includes('Earth Trekkers')"));
    assert.ok(await browser.evaluate("document.querySelector('.map-panel-hours').open && document.querySelector('.map-panel-hours p').textContent.includes('09:00')"));
    assert.equal(await browser.evaluate("document.querySelectorAll('.map-panel-writers li').length"), 2);
    if (width === 375) {
      const uncovered = await browser.evaluate(`(() => {
        const panel = document.querySelector('.map-place-panel');
        const rect = panel.querySelector('.map-panel-title').getBoundingClientRect();
        return [0.25, 0.5, 0.75].every(x => [0.25, 0.5, 0.75].every(y =>
          panel.contains(document.elementFromPoint(rect.left + rect.width * x, rect.top + rect.height * y))));
      })()`);
      assert.ok(uncovered, 'map pins must stay behind the phone place sheet');
    }
    await writeFile(`${out}/funchal-panel-${width}.png`, await browser.shot());
    await browser.evaluate("document.querySelector('.map-panel-save').click()");
    await browser.until("!document.querySelector('.writers-map-toast').hidden");
    assert.ok(await browser.evaluate(`(() => {
      const undo = document.querySelector('[data-saved-undo]'); const r = undo.getBoundingClientRect();
      return undo === document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2);
    })()`), 'Undo toast is visible and clickable above the phone sheet');
    await browser.evaluate("document.querySelector('[data-saved-undo]').click()");
    assert.equal(await browser.evaluate("document.querySelector('.map-panel-save').getAttribute('aria-pressed')"), 'false');
    await browser.evaluate("document.querySelector('.map-panel-save').click(); document.querySelector('.tt-saved-toggle').click()");
    await browser.until("document.querySelector('[data-saved-place=Q25444]')");
    assert.equal(await browser.evaluate("document.querySelector('[data-saved-place=Q25444]').textContent"), 'Funchal');
    await browser.key('Escape', 'Escape', 27);
    assert.equal(await browser.evaluate("document.activeElement.classList.contains('tt-saved-toggle')"), true);
    await browser.evaluate("document.querySelector('.tt-saved-toggle').click(); document.querySelector('[data-saved-place=Q25444]').click()");
    await browser.until("document.querySelector('.map-place-panel h3')?.textContent.trim() === 'Funchal'");
    await browser.evaluate("document.querySelector('[data-add-qid]').click()");
    await browser.until("location.hash === '#itinerary' && !document.querySelector('#itinerary').hidden && document.querySelectorAll('[data-plan-stop]').length > 0");
    assert.ok(await browser.evaluate("[...document.querySelectorAll('[data-plan-stop]')].some(stop => stop.dataset.qid === 'Q25444')"));
    assert.ok(await browser.evaluate('document.documentElement.scrollWidth <= innerWidth'));
    assert.deepEqual(await browser.evaluate('window.__previewErrors'), []);
    assert.deepEqual(browser.errors, []);
    await writeFile(`${out}/funchal-itinerary-${width}.png`, await browser.shot());
    console.log(`PASS Funchal ${width}px: search, writer pin, panel, Add to plan, itinerary, no overflow or console errors`);
  }
} finally {
  await browser?.close?.();
  preview.stop();
}
