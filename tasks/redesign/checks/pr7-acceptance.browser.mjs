#!/usr/bin/env node
// PR 7 acceptance gaps: 7b (no duplicate layers, base-only destination), 7d (storage, no server call on save,
// only /plan carries include) and 7g (View on map via map-dayplan.js, no inline handler).
// Start an isolated Brave/Chromium on CDP_PORT with about:blank, then run this check.
import assert from 'node:assert/strict';
import { connect, startPreview, sleep } from './lib/browser.mjs';

for (const [engine, scenario] of [['maplibre', 'writers-map'], ['leaflet', 'writers-map-leaflet']]) {
const preview = await startPreview({ scenario });
let browser;
try {
  browser = await connect();
  const hook = `
    window.__previewErrors = [];
    const originalError = console.error.bind(console);
    console.error = (...args) => { window.__previewErrors.push(args.map(String).join(' ')); originalError(...args); };
    window.addEventListener('error', event => window.__previewErrors.push(event.message));
    window.addEventListener('unhandledrejection', event => window.__previewErrors.push(String(event.reason)));`;
  await browser.call('Page.addScriptToEvaluateOnNewDocument', { source: hook });
  const requests = () => browser.events.filter(e => e.method === 'Network.responseReceived').map(e => e.params.response.url);
  const count = sel => browser.evaluate(`document.querySelectorAll('${sel}').length`);
  const chip = label => `[...document.querySelectorAll('[data-map-slot="layers"] button')].find(b => b.textContent.includes("${label}"))`;
  const generate = async () => {
    await browser.evaluate("document.querySelector('#trip-card form').requestSubmit()");
    await browser.until("document.querySelectorAll('[data-plan-stop]').length > 0");
  };

  for (const width of [375, 1440]) {
    await browser.viewport(width, 900);

    // 7b: base-only destination (Porto), and the map destination, leave no duplicate layers.
    for (const slug of ['porto-pt', 'madeira-pt']) {
      await browser.navigate(`${preview.url}/trip/${slug}`);
      await browser.until(`Boolean(document.querySelector('[data-map-canvas] .${engine === 'maplibre' ? 'maplibregl-canvas' : 'leaflet-pane'}') && document.querySelector('#trip-card form'))`);
      assert.equal(await count('.plan-pin'), 0, `${slug}: no plan pins before a plan`);
      await generate();
      await browser.until("document.querySelectorAll('.plan-pin').length > 0");
      const stops = await count('[data-plan-stop]');
      assert.equal(await count('.plan-pin'), stops, `${slug}: one pin per stop`);
      for (let i = 0; i < 2; i++) { // regenerate twice, with a different length to force a redraw
        await browser.evaluate(`document.querySelector('#trip-days').value = '${i ? 3 : 2}'`);
        await generate();
        await sleep(150);
        const now = await count('[data-plan-stop]');
        assert.equal(await count('.plan-pin'), now, `${slug}: pins equal stops after regenerate ${i + 1}`);
        assert.ok(await count('.plan-day-toggle') <= 1, `${slug}: one day toggle`);
      }
      await browser.evaluate('window.TravelMap.updatePlan([])');
      assert.equal(await count('.plan-pin'), 0, `${slug}: cleared`);
      assert.equal(await count('.plan-day-toggle'), 0, `${slug}: toggle cleared`);
      console.log(`PASS 7b ${engine} ${slug} ${width}px: ${stops} stops, no duplicates after regenerate, clear`);
    }

    // 7d: saves make no server call; only the plan request carries include.
    await browser.navigate(`${preview.url}/trip/madeira-pt`);
    await browser.evaluate('(() => { try { localStorage.clear(); } catch {} })()');
    await browser.navigate(`${preview.url}/trip/madeira-pt`);
    await browser.until("document.querySelectorAll('[data-map-slot=\"list\"] button').length > 0");
    await browser.evaluate("[...document.querySelectorAll('[data-map-slot=\"list\"] button')].find(b => b.textContent === 'Pico Ruivo').click()");
    await browser.until("document.querySelector('.map-place-panel [data-save-qid]')");
    await sleep(300);
    const before = requests().length;
    await browser.evaluate("document.querySelector('.map-place-panel [data-save-qid]').click()");
    await sleep(500);
    assert.equal(requests().length, before, `saving made requests: ${requests().slice(before)}`);
    await browser.evaluate("document.querySelector('#trip-card form').requestSubmit()");
    await browser.until("document.querySelectorAll('.plan-pin').length > 0");
    const withInclude = requests().slice(before).filter(u => u.includes('include='));
    assert.equal(withInclude.length, 1, JSON.stringify(withInclude));
    assert.match(new URL(withInclude[0]).pathname, /^\/plan$/);
    assert.ok(requests().filter(u => u.includes('include=')).every(u => new URL(u).pathname === '/plan'), 'include only on /plan');
    assert.match(withInclude[0], /include=Q473169/);
    console.log(`PASS 7d ${engine} ${width}px: save made 0 requests; only ${new URL(withInclude[0]).pathname} carried include`);

    // 7e in the preview: five saved places and a one-day plan -> the day holds four, one is "also suggested".
    await browser.navigate(`${preview.url}/trip/madeira-pt`);
    await browser.evaluate('(() => { try { localStorage.clear(); } catch {} })()');
    await browser.navigate(`${preview.url}/trip/madeira-pt`);
    await browser.until("document.querySelectorAll('[data-map-slot=\"list\"] button').length >= 5");
    const names = await browser.evaluate("[...document.querySelectorAll('[data-map-slot=\"list\"] button')].slice(0, 5).map(b => b.textContent)");
    for (const name of names) {
      await browser.evaluate(`[...document.querySelectorAll('[data-map-slot="list"] button')].find(b => b.textContent === ${JSON.stringify(name)}).click()`);
      await browser.until(`document.querySelector('.map-place-panel h3')?.textContent === ${JSON.stringify(name)} && document.querySelector('.map-place-panel [data-save-qid]')`);
      await sleep(100);
      await browser.evaluate("document.querySelector('.map-place-panel [data-save-qid]').click()");
    }
    assert.match(await browser.evaluate("document.querySelector('[data-map-slot=\"saved\"]').textContent"), /5 saved places/);
    await browser.evaluate("document.querySelector('#trip-days').value = '1'");
    await generate();
    const planned = await browser.evaluate("[...document.querySelectorAll('[data-plan-stop]')].map(e => e.dataset.name)");
    const also = await browser.evaluate("[...document.querySelectorAll('[data-also-suggested]')].length");
    assert.equal(planned.length, 4, JSON.stringify(planned));
    assert.ok(names.slice(0, 4).every(n => planned.includes(n)), JSON.stringify({ names, planned }));
    assert.equal(also, 1, 'one place is also suggested');
    console.log(`PASS 7e preview ${engine} ${width}px: saved places in the day (${planned.join(', ')}), ${also} also suggested`);
    await browser.evaluate('(() => { try { localStorage.clear(); } catch {} })()');

    // 7g: View on map presses the plan chip only when needed, scrolls to the map, no inline handler.
    assert.equal(await count('[data-view-on-map][onclick]'), 0);
    await browser.until(`${chip("day plan")}.getAttribute('aria-pressed') === 'true'`);
    await browser.evaluate(`${chip("day plan")}.click()`);
    assert.equal(await count('.plan-pin'), 0, 'plan chip off');
    await browser.evaluate('window.scrollTo(0, document.documentElement.scrollHeight)');
    const scrolled = await browser.evaluate('window.scrollY');
    await browser.evaluate("document.querySelector('[data-view-on-map]').click()");
    await browser.until("document.querySelectorAll('.plan-pin').length > 0");
    assert.equal(await browser.evaluate(`${chip("day plan")}.getAttribute('aria-pressed')`), 'true');
    await browser.until(`window.scrollY < ${scrolled}`);
    await browser.evaluate("document.querySelector('[data-view-on-map]').click()");
    await sleep(200);
    assert.equal(await browser.evaluate(`${chip("day plan")}.getAttribute('aria-pressed')`), 'true', 'already pressed stays pressed');
    assert.ok(await count('.plan-pin') > 0);
    assert.deepEqual(await browser.evaluate('window.__previewErrors'), []);
    console.log(`PASS 7g ${engine} ${width}px: View on map presses chip once, scrolls ${scrolled} -> ${await browser.evaluate('window.scrollY')}`);
  }

  // 7d: blocked storage hides the buttons and the saved line.
  const blocker = await browser.call('Page.addScriptToEvaluateOnNewDocument', { source: `
    Object.defineProperty(window, 'localStorage', { get() { throw new DOMException('blocked', 'SecurityError'); } });` });
  await browser.navigate(`${preview.url}/trip/madeira-pt`);
  await browser.until("document.querySelectorAll('[data-map-slot=\"list\"] button').length > 0");
  assert.equal(await browser.evaluate('(() => { try { localStorage.length; return false; } catch { return true; } })()'), true, 'storage is blocked');
  await browser.evaluate("[...document.querySelectorAll('[data-map-slot=\"list\"] button')].find(b => b.textContent === 'Pico Ruivo').click()");
  await browser.until("document.querySelector('.map-place-panel [data-save-qid]')");
  await sleep(300);
  assert.equal(await browser.evaluate("document.querySelector('.map-place-panel [data-save-qid]').hidden"), true, 'save button hidden');
  assert.equal(await browser.evaluate("document.querySelector('[data-map-slot=\"saved\"]').children.length"), 0, 'saved line hidden');
  await browser.evaluate("document.querySelector('.map-place-panel [data-save-qid]').click()");
  assert.equal(await browser.evaluate("document.querySelector('[data-map-slot=\"saved\"]').children.length"), 0);
  await browser.evaluate("document.querySelector('#trip-card form').requestSubmit()");
  await browser.until("document.querySelectorAll('[data-plan-stop]').length > 0");
  assert.deepEqual(await browser.evaluate('window.__previewErrors'), []);
  await browser.call('Page.removeScriptToEvaluateOnNewDocument', { identifier: blocker.identifier });
  console.log(`PASS 7d ${engine} blocked storage: buttons and saved line hidden, plan still works, console clean`);
} finally {
  await browser?.close?.();
  preview.stop();
}
}
