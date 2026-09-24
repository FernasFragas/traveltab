#!/usr/bin/env node
// Browser acceptance against the deterministic design preview.
import assert from 'node:assert/strict';
import { readFile, mkdir, writeFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';

const base = process.env.PREVIEW_URL || 'http://127.0.0.1:8087';
const scenario = process.env.SCENARIO || 'default';
assert.ok(['default', 'fallback'].includes(scenario));
assert.ok(['127.0.0.1', 'localhost'].includes(new URL(base).hostname));
assert.ok(process.env.HTMX_PATH, 'Set HTMX_PATH to the pinned 1.9.11 script');
const htmx = await readFile(process.env.HTMX_PATH);
const template = await readFile(new URL('../../../views/index.go.tpl', import.meta.url), 'utf8');
const hash = template.match(/htmx.org@1\.9\.11" integrity="sha384-([^"]+)"/)?.[1];
assert.equal(createHash('sha384').update(htmx).digest('base64'), hash);
const output = new URL(`../../artifacts/redesign/final-verification/${scenario}/`, import.meta.url);
await mkdir(output, { recursive: true });
const tabs = await (await fetch(`http://127.0.0.1:${process.env.CDP_PORT || 9227}/json/list`, { signal: AbortSignal.timeout(5000) })).json();
const tab = tabs.find(t => t.type === 'page');
assert.ok(tab, 'Start isolated Chromium with an about:blank tab');
const ws = new WebSocket(tab.webSocketDebuggerUrl);
await new Promise((resolve, reject) => {
  const timer = setTimeout(() => reject(new Error('CDP connection timeout')), 5000);
  ws.addEventListener('open', () => { clearTimeout(timer); resolve(); }, { once: true });
  ws.addEventListener('error', reject, { once: true });
});
let sequence = 0;
const pending = new Map();
const errors = [];
const failedResources = [];
let planMode = 'normal';
const call = (method, params = {}) => new Promise((resolve, reject) => {
  const id = ++sequence;
  const timer = setTimeout(() => { pending.delete(id); reject(new Error(`CDP timeout: ${method}`)); }, 15000);
  pending.set(id, { resolve, reject, timer });
  ws.send(JSON.stringify({ id, method, params }));
});
ws.addEventListener('message', event => {
  const message = JSON.parse(event.data);
  if (message.id) {
    const item = pending.get(message.id);
    if (!item) return;
    pending.delete(message.id); clearTimeout(item.timer);
    message.error ? item.reject(message.error) : item.resolve(message.result);
  } else if (message.method === 'Runtime.exceptionThrown') {
    errors.push(message.params.exceptionDetails.exception?.description || JSON.stringify(message.params.exceptionDetails));
  } else if (message.method === 'Network.loadingFailed') {
    failedResources.push(message.params.errorText);
  } else if (message.method === 'Fetch.requestPaused') {
    const { requestId, request } = message.params;
    const action = request.url === 'https://unpkg.com/htmx.org@1.9.11'
      ? call('Fetch.fulfillRequest', { requestId, responseCode: 200, body: htmx.toString('base64'), responseHeaders: [
        { name: 'Content-Type', value: 'text/javascript' }, { name: 'Access-Control-Allow-Origin', value: '*' }] })
      : new URL(request.url).pathname === '/plan' && planMode === 'fail'
        ? call('Fetch.failRequest', { requestId, errorReason: 'InternetDisconnected' })
      : new URL(request.url).pathname === '/plan' && planMode === 'delay'
        ? new Promise(resolve => setTimeout(resolve, 800)).then(() => call('Fetch.continueRequest', { requestId }))
      : call('Fetch.continueRequest', { requestId });
    action.catch(error => errors.push(String(error)));
  }
});
const evaluate = async expression => {
  const result = await call('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true });
  assert.equal(result.exceptionDetails, undefined, JSON.stringify(result.exceptionDetails));
  return result.result.value;
};
const until = async expression => {
  const deadline = Date.now() + 15000;
  while (Date.now() < deadline) {
    if (await evaluate(expression)) return;
    await new Promise(resolve => setTimeout(resolve, 50));
  }
  throw new Error(`Timed out: ${expression}`);
};
const screenshot = async name => {
  await evaluate(`(async () => {
    const frame = document.querySelector('.map-embed');
    if (!frame || frame.dataset.evidenceMapLoaded === 'true') return;
    await new Promise((resolve, reject) => {
      const timer = setTimeout(() => reject(Error('Fixture map iframe did not load')), 5000);
      frame.addEventListener('load', () => { clearTimeout(timer); frame.dataset.evidenceMapLoaded = 'true'; resolve(); }, { once: true });
      frame.loading = 'eager';
      frame.src = frame.src;
    });
  })()`);
  await evaluate("window.scrollTo({top: 0, behavior: 'instant'})");
  await until('window.scrollY === 0');
  const viewport = await call('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false });
  await writeFile(new URL(`${name}-viewport.png`, output), Buffer.from(viewport.data, 'base64'));
  await evaluate(`(() => {
    window.evidenceHidden = [...document.querySelectorAll('.tt-bottom-navigation, .tt-skip-link')]
      .map(node => [node, node.style.visibility]);
    window.evidenceHidden.forEach(([node]) => { node.style.visibility = 'hidden'; });
  })()`);
  try {
    const full = await call('Page.captureScreenshot', { format: 'png', captureBeyondViewport: true });
    await writeFile(new URL(`${name}-full.png`, output), Buffer.from(full.data, 'base64'));
  } finally {
    await evaluate(`window.evidenceHidden.forEach(([node, visibility]) => { node.style.visibility = visibility; }); delete window.evidenceHidden;`);
  }
};
const inspect = () => evaluate(`(() => {
  const rect = selector => document.querySelector(selector)?.getBoundingClientRect().toJSON();
  const days = [...document.querySelectorAll('.itinerary-day')];
  return {
    title: document.querySelector('.destination-title')?.textContent.trim(),
    htmx: window.htmx?.version, planner: !!document.querySelector('.trip-form'),
    photo: document.querySelector('.destination-image')?.naturalWidth || 0,
    videos: document.querySelectorAll('[data-video-activate]').length,
    sections: ['overview','itinerary','stays','videos'].map(id => document.querySelectorAll('#' + id).length),
    overflow: document.documentElement.scrollWidth, viewport: innerWidth,
    map: rect('.tt-overview-map'), form: rect('.tt-overview-planner'),
    days: days.length, openDays: days.map(day => day.open),
    stays: document.querySelectorAll('.stay-item').length,
    note: document.querySelector('.stay-note')?.textContent.trim() || '',
    error: document.querySelector('.trip-error')?.textContent.trim() || '',
    searchError: document.querySelector('#destination-search-error')?.textContent.trim() || '',
    idsUnique: [...document.querySelectorAll('[id]')].every(node => document.querySelectorAll('#' + CSS.escape(node.id)).length === 1)
  };
})()`);
const results = [];
try {
  await call('Page.enable'); await call('Runtime.enable'); await call('Network.enable');
  await call('Network.setCacheDisabled', { cacheDisabled: true });
  await call('Fetch.enable', { patterns: [{ urlPattern: 'https://unpkg.com/htmx.org@1.9.11' }, { urlPattern: '*/plan*' }] });
  const browser = (await call('Browser.getVersion')).product;
  for (const [width, height] of [[375,812],[768,1024],[1100,1000],[1440,1000]]) {
    await call('Emulation.setDeviceMetricsOverride', { width, height, deviceScaleFactor: 1, mobile: width <= 768 });
    await call('Page.navigate', { url: `${base}/?browser-acceptance=${Date.now()}` });
    await until("document.readyState === 'complete' && window.htmx?.version === '1.9.11' && !!document.querySelector('.destination-title')");
    const initial = await inspect();
    assert.equal(initial.title, 'Lisbon'); assert.ok(initial.photo > 0, 'Lisbon hero must load');
    assert.deepEqual(initial.sections, [1,1,1,1]); assert.equal(initial.idsUnique, true);
    assert.equal(initial.planner, true); assert.ok(initial.videos > 0 || scenario === 'fallback');
    assert.ok(initial.overflow <= width, `${width}px initial overflow: ${initial.overflow}`);
    if (width === 375) assert.ok(initial.form.y < initial.map.y, 'Mobile form precedes map');
    await screenshot(`${width}-initial`);
    await evaluate("document.querySelector('.trip-form').requestSubmit()");
    await until("document.querySelectorAll('.itinerary-day').length === 3 && document.querySelector('.trip-submit')?.disabled === false");
    assert.equal(await evaluate("document.activeElement?.id"), 'itinerary-title', 'Planning focuses the current itinerary heading');
    const planned = await inspect();
    assert.equal(planned.days, 3); assert.deepEqual(planned.openDays, [true,false,false]);
    assert.ok(planned.overflow <= width, `${width}px planned overflow: ${planned.overflow}`);
    if (scenario === 'fallback') {
      assert.equal(planned.stays, 0); assert.match(planned.note, /unavailable/i);
      assert.equal(await evaluate("document.querySelectorAll('.itinerary-stop-photo').length"), 0);
      assert.match(await evaluate("document.querySelector('.itinerary-days').textContent"), /No forecast data/);
    } else assert.ok(planned.stays > 0);
    await screenshot(`${width}-planned`);
    results.push({ width, height, initial, planned });
    console.log(`PASS ${scenario} ${width}×${height}: data, layout, planning, screenshots`);
  }
  await call('Emulation.setDeviceMetricsOverride', { width: 375, height: 812, deviceScaleFactor: 1, mobile: true });
  await call('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-reduced-motion', value: 'reduce' }] });
  assert.equal(await evaluate("matchMedia('(prefers-reduced-motion: reduce)').matches"), true);
  await call('Emulation.setPageScaleFactor', { pageScaleFactor: 2 });
  assert.ok((await inspect()).overflow <= 375, '200% zoom must not cause overflow');
  await call('Emulation.setPageScaleFactor', { pageScaleFactor: 1 });
  assert.equal(await evaluate("(() => { const d = document.querySelectorAll('.itinerary-day'); d[1].querySelector('summary').click(); return d[0].open && d[1].open; })()"), true);
  await evaluate("document.querySelector('.trip-form').requestSubmit()");
  await until("document.querySelectorAll('.itinerary-day').length === 3 && document.querySelector('#trip-status')?.textContent.includes('updated')");
  assert.deepEqual((await inspect()).openDays, [true,false,false], 'Regeneration reopens first day');
  if (scenario === 'default') {
    const sharedURL = await evaluate('location.href');
    assert.match(sharedURL, /\/trip\/lisbon-pt\?days=3&from=/);
    const links = await evaluate(`(() => ({
      calendar: document.querySelector('.itinerary-action[href*=".ics"]')?.href,
      map: document.querySelector('.itinerary-action[href*=".kml"]')?.href,
      booking: document.querySelector('.stay-prices')?.href,
      directions: [...document.querySelectorAll('.itinerary-directions')].map(a => a.href),
      start: document.querySelector('.trip-form').elements.start.value
    }))()`);
    assert.ok(links.calendar && links.map && links.booking && links.directions.length === 3);
    assert.ok(links.directions.every(url => url.startsWith('https://www.google.com/maps/dir/')));
    const booking = new URL(links.booking);
    assert.equal(booking.searchParams.get('checkin'), links.start);
    const checkout = new Date(`${links.start}T00:00:00Z`);
    checkout.setUTCDate(checkout.getUTCDate() + 3);
    assert.equal(booking.searchParams.get('checkout'), checkout.toISOString().slice(0, 10));
    for (const url of [links.calendar, links.map]) {
      const download = await fetch(url);
      assert.equal(download.status, 200, `Export must download: ${url}`);
    }
    await call('Page.navigate', { url: sharedURL });
    await until("document.readyState === 'complete' && window.htmx?.version === '1.9.11' && document.querySelectorAll('.itinerary-day').length === 3");
    assert.equal((await inspect()).title, 'Lisbon', 'Shared page reload keeps selected trip');
    assert.equal(await evaluate("document.querySelector('.trip-form').elements.start.value"), links.start);
    await evaluate("document.querySelector('.trip-select').focus()");
    await call('Input.dispatchKeyEvent', { type: 'keyDown', key: 'Tab', code: 'Tab', windowsVirtualKeyCode: 9, nativeVirtualKeyCode: 9 });
    await call('Input.dispatchKeyEvent', { type: 'keyUp', key: 'Tab', code: 'Tab', windowsVirtualKeyCode: 9 });
    assert.equal(await evaluate("document.activeElement?.classList.contains('trip-submit')"), true, 'Keyboard Tab reaches submit');
    assert.equal(await evaluate("getComputedStyle(document.activeElement).outlineStyle"), 'solid', 'Keyboard focus ring is visible');
    planMode = 'delay';
    const loading = await evaluate(`(() => { document.querySelector('.trip-form').requestSubmit(); return {
      status: document.querySelector('#trip-status').textContent,
      disabled: document.querySelector('.trip-submit').disabled,
      busy: document.querySelector('.trip-form').getAttribute('aria-busy')
    }; })()`);
    assert.deepEqual(loading, { status: 'Planning your trip…', disabled: true, busy: 'true' });
    await until("document.querySelector('#trip-status')?.textContent.includes('updated') && !document.querySelector('.trip-submit')?.disabled");
    planMode = 'fail';
    await evaluate("document.querySelector('.trip-form').requestSubmit()");
    await until("document.querySelector('#trip-status')?.textContent.includes('did not complete') && !document.querySelector('.trip-submit')?.disabled");
    assert.equal(await evaluate("document.querySelector('.trip-form').getAttribute('aria-busy')"), 'false');
    planMode = 'normal';
    await evaluate("document.querySelector('.trip-form').elements.days.value = '4'; document.querySelector('.trip-form').requestSubmit()");
    await until("!!document.querySelector('.trip-error')");
    const failed = await inspect();
    assert.equal(failed.days, 0); assert.equal(failed.stays, 0); assert.ok(failed.error);
    await screenshot('375-plan-error');
    await evaluate("document.querySelector('.trip-form').elements.lat.value = 'invalid'; document.querySelector('.trip-form').requestSubmit()");
    await until("document.querySelector('.trip-error')?.textContent.includes('Search') || document.querySelector('.trip-error')?.textContent.includes('destination')");
    const invalid = await inspect();
    assert.equal(invalid.days, 0); assert.equal(invalid.stays, 0); assert.ok(invalid.error);
    await screenshot('375-validation-error');
    await evaluate("document.querySelector('#city_name').value = 'Porto'; document.querySelector('#destination-search').requestSubmit()");
    await until("document.querySelector('.destination-title')?.textContent.trim() === 'Porto'");
    const porto = await inspect(); assert.equal(porto.videos, 0); assert.equal(porto.days, 0);
    await screenshot('375-porto-empty-videos');
    await evaluate("document.querySelector('#city_name').value = 'error'; document.querySelector('#destination-search').requestSubmit()");
    await until("document.querySelector('#destination-search-error')?.textContent.trim().length > 0");
    assert.equal((await inspect()).title, 'Porto', 'Search failure retains current destination');
  }
  assert.deepEqual(errors, [], 'Unexpected JavaScript exceptions');
  await writeFile(new URL('browser-results.json', output), JSON.stringify({ browser, htmx: '1.9.11', scenario, base, results, errors, failedResources }, null, 2));
  console.log(`PASS ${scenario}: interactions, reduced motion, 200% zoom, no JavaScript exceptions`);
  console.log(`Evidence: ${output.pathname}`);
} finally {
  await call('Fetch.disable').catch(() => {});
  ws.close();
}
