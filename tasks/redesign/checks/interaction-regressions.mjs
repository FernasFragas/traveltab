// Run the default design preview (six Lisbon videos; empty Porto) and an isolated Chromium.
// HTMX_PATH is the exact pinned local script; external requests are blocked (no playback proof).
// PREVIEW_URL=http://127.0.0.1:8087 CDP_PORT=9227 HTMX_PATH=/tmp/htmx.js node <this file>
// For negative-control runs, PLANNER_PATH and DISCOVERY_PATH can supply local JS variants.
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';

const base = process.env.PREVIEW_URL || 'http://127.0.0.1:8087';
assert.ok(['127.0.0.1', 'localhost'].includes(new URL(base).hostname), 'Use a local fixture');
assert.ok(process.env.HTMX_PATH, 'HTMX_PATH must point to HTMX 1.9.11');
const script = await readFile(process.env.HTMX_PATH);
// Optional local variants prove that the assertions fail against each original bug.
const overrides = new Map();
for (const [name, path] of [['planner.js', process.env.PLANNER_PATH], ['discovery.js', process.env.DISCOVERY_PATH]]) {
  if (path) overrides.set(`/redesign/${name}`, (await readFile(path)).toString('base64'));
}
const template = await readFile(new URL('../../../views/index.go.tpl', import.meta.url), 'utf8');
const integrity = template.match(/htmx.org@1\.9\.11" integrity="sha384-([^"]+)"/)?.[1];
assert.ok(integrity, 'Template must pin HTMX 1.9.11 with SRI');
assert.equal(createHash('sha384').update(script).digest('base64'), integrity, 'Pinned HTMX integrity');
const tabs = await (await fetch(`http://127.0.0.1:${process.env.CDP_PORT || 9227}/json/list`, { signal: AbortSignal.timeout(5000) })).json();
const tab = tabs.find(tab => tab.type === 'page');
assert.ok(tab, 'Start a separate Chromium instance with an about:blank tab');
const ws = new WebSocket(tab.webSocketDebuggerUrl);
await new Promise((resolve, reject) => {
  const timer = setTimeout(() => reject(new Error('CDP connection timeout')), 5000);
  ws.addEventListener('open', () => { clearTimeout(timer); resolve(); }, { once: true });
  ws.addEventListener('error', error => { clearTimeout(timer); reject(error); }, { once: true });
});
let id = 0;
const pending = new Map();
const errors = [];
const call = (method, params = {}) => new Promise((resolve, reject) => {
  const messageID = ++id;
  const timer = setTimeout(() => { pending.delete(messageID); reject(new Error(`CDP timeout: ${method}`)); }, 15000);
  pending.set(messageID, { resolve, reject, timer });
  ws.send(JSON.stringify({ id: messageID, method, params }));
});
ws.addEventListener('message', event => {
  const message = JSON.parse(event.data);
  if (message.id) {
    const item = pending.get(message.id);
    if (!item) return;
    pending.delete(message.id);
    clearTimeout(item.timer);
    message.error ? item.reject(message.error) : item.resolve(message.result);
  } else if (message.method === 'Runtime.exceptionThrown') {
    errors.push(message.params.exceptionDetails);
  } else if (message.method === 'Fetch.requestPaused') {
    const { requestId, request } = message.params;
    const url = new URL(request.url);
    let action;
    if (['127.0.0.1', 'localhost'].includes(url.hostname) && overrides.has(url.pathname)) {
      action = call('Fetch.fulfillRequest', { requestId, responseCode: 200, body: overrides.get(url.pathname),
        responseHeaders: [{ name: 'Content-Type', value: 'text/javascript' }] });
    } else if (['127.0.0.1', 'localhost'].includes(url.hostname)) {
      action = call('Fetch.continueRequest', { requestId });
    } else if (url.href === 'https://unpkg.com/htmx.org@1.9.11') {
      action = call('Fetch.fulfillRequest', { requestId, responseCode: 200, body: script.toString('base64'),
        responseHeaders: [{ name: 'Content-Type', value: 'text/javascript' }, { name: 'Access-Control-Allow-Origin', value: '*' }] });
    } else {
      action = call('Fetch.failRequest', { requestId, errorReason: 'BlockedByClient' });
    }
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
const activateVideo = async (keyboard = false) => {
  const expected = await evaluate(`(() => {
    const button = document.querySelector('[data-video-activate]');
    if (!button) throw Error('Fixture must have a playable video');
    window.testVideoButton = button;
    window.testVideoPlayer = button.closest('[data-video-player]');
    button.focus();
    return { id: button.dataset.videoId, title: button.dataset.videoTitle };
  })()`);
  if (keyboard) {
    await until('document.activeElement === window.testVideoButton');
    await call('Input.dispatchKeyEvent', { type: 'keyDown', key: 'Enter', code: 'Enter', windowsVirtualKeyCode: 13, nativeVirtualKeyCode: 13, text: '\r' });
    await call('Input.dispatchKeyEvent', { type: 'keyUp', key: 'Enter', code: 'Enter', windowsVirtualKeyCode: 13 });
  } else await evaluate('window.testVideoButton.click()');
  const player = await evaluate(`({src: testVideoPlayer.querySelector('iframe')?.src, title: testVideoPlayer.querySelector('iframe')?.title, count: testVideoPlayer.querySelectorAll('iframe').length})`);
  assert.equal(player.src, `https://www.youtube-nocookie.com/embed/${expected.id}?autoplay=1&rel=0`, 'Valid activation must create its iframe');
  assert.equal(player.title, `Play video: ${expected.title}`);
  assert.equal(player.count, 1);
  assert.equal(await evaluate(`(() => {
    testVideoPlayer.appendChild(testVideoButton);
    testVideoButton.click(); testVideoButton.click();
    testVideoButton.remove();
    return testVideoPlayer.querySelectorAll('iframe').length;
  })()`), 1, 'Repeated activation must not duplicate an existing player');
};
const assertRestored = async (planned, expectedURL, values) => {
  assert.deepEqual(await evaluate(`({
    disabled: document.querySelector('.trip-submit').disabled,
    pending: document.querySelector('.trip-form').dataset.tripPending,
    busy: document.querySelector('.trip-form').getAttribute('aria-busy'),
    status: document.querySelector('#trip-status').textContent
  })`), { disabled: false, pending: 'false', busy: 'false', status: '' });
  assert.equal(await evaluate('location.href'), expectedURL);
  assert.equal(await evaluate('document.querySelector(".destination-title").textContent.trim()'), 'Lisbon');
  assert.equal(await evaluate('document.querySelectorAll(".itinerary-day").length'), planned ? 3 : 0);
  assert.equal(await evaluate('document.querySelectorAll(".stay-item").length > 0'), planned);
  assert.deepEqual(await evaluate('Object.fromEntries(new FormData(document.querySelector(".trip-form")))'), values);
};
const travelHistory = async direction => {
  const previous = await evaluate('window.restores');
  await evaluate(`history.${direction}()`);
  await until(`window.restores === ${previous + 1}`);
};
const submit = async duplicate => {
  const loading = await evaluate(`(() => {
    const form = document.querySelector('.trip-form');
    form.requestSubmit();
    ${duplicate ? 'form.requestSubmit();' : ''}
    return {status: document.querySelector('#trip-status').textContent, disabled: form.querySelector('.trip-submit').disabled};
  })()`);
  assert.deepEqual(loading, { status: 'Planning your trip…', disabled: true });
  await until(`document.querySelectorAll('.itinerary-day').length === 3 &&
    document.querySelector('#trip-status').textContent === 'Your itinerary has been updated.' &&
    document.activeElement?.id === 'itinerary-title' && !document.querySelector('.trip-submit').disabled`);
};

try {
  await call('Page.enable');
  await call('Runtime.enable');
  await call('Network.enable');
  await call('Network.setCacheDisabled', { cacheDisabled: true });
  await call('Fetch.enable', { patterns: [{ urlPattern: '*' }] });
  await call('Page.navigate', { url: `${base}/?interaction-regression=${Date.now()}` });
  await until('document.readyState === "complete" && !!window.htmx && !!window.travelTabPlannerWired && !!window.travelTabDiscoveryWired');
  assert.equal(await evaluate('htmx.version'), '1.9.11');
  console.log(`Environment: ${(await call('Browser.getVersion')).product}; HTMX ${await evaluate('htmx.version')}`);
  await activateVideo();
  console.log('PASS: valid initial video, title, expected embed URL, repeated activation');
  await evaluate(`
    localStorage.removeItem('htmx-history-cache');
    window.planRequests = 0;
    window.restores = 0;
    document.body.addEventListener('htmx:beforeRequest', event => {
      if (event.detail.elt.matches('.trip-form')) window.planRequests++;
    });
    document.body.addEventListener('htmx:historyRestore', () => window.restores++);
  `);
  const initialURL = await evaluate('location.href');
  const values = await evaluate('Object.fromEntries(new FormData(document.querySelector(".trip-form")))');
  assert.equal(values.country, 'pt', 'Fixture must use the ISO country code');
  await submit(true);
  assert.equal(await evaluate('window.planRequests'), 1, 'Synchronous duplicate submissions must produce one request');
  const plannedURL = await evaluate('location.href');
  const expectedURL = new URL('/trip/lisbon-pt', base);
  expectedURL.searchParams.set('days', values.days);
  expectedURL.searchParams.set('from', values.start);
  assert.equal(plannedURL, expectedURL.href);
  await travelHistory('back');
  await assertRestored(false, initialURL, values);
  await travelHistory('forward');
  await assertRestored(true, plannedURL, values);
  await travelHistory('back');
  await assertRestored(false, initialURL, values);
  await submit(true);
  assert.equal(await evaluate('window.planRequests'), 2, 'Later legitimate submit must produce another request');
  assert.equal(await evaluate('location.href'), plannedURL);
  console.log('PASS: generate → Back → Forward → Back → generate; URL, city, form, itinerary, stays, live status and focus');

  await evaluate(`document.querySelector('#city_name').value = 'Porto'; document.querySelector('#destination-search').requestSubmit()`);
  await until('document.querySelector(".destination-title").textContent.trim() === "Porto"');
  assert.equal(await evaluate('document.querySelectorAll("[data-video-activate]").length'), 0);
  assert.match(await evaluate('document.querySelector(".video-note").textContent'), /No travel videos/);
  await evaluate(`document.querySelector('#city_name').value = 'Coimbra'; document.querySelector('#destination-search').requestSubmit()`);
  await until('document.querySelector(".destination-title").textContent.trim() === "Coimbra"');
  assert.equal(await evaluate('document.querySelector(".trip-form").elements.city.value'), 'Coimbra');
  assert.equal(await evaluate('document.querySelectorAll(".itinerary-day").length'), 0);
  assert.equal(await evaluate('document.querySelectorAll(".stay-item").length'), 0);
  assert.equal(await evaluate(`(() => {
    const button = document.querySelector('[data-video-activate]');
    const original = button.dataset.videoId;
    const player = button.closest('[data-video-player]');
    for (const invalid of ['', 'too-short', 'invalid<script>', '../abcdefgh']) { button.dataset.videoId = invalid; button.click(); }
    button.dataset.videoId = original;
    return player.querySelectorAll('iframe').length;
  })()`), 0, 'Invalid video IDs must never create a player');
  await activateVideo(true);
  assert.equal(await evaluate(`(() => {
    const button = document.querySelector('[data-video-expand]');
    if (!button) throw Error('Use default fixture with more than four videos');
    const extra = document.getElementById(button.getAttribute('aria-controls'));
    if (!extra.hidden || button.getAttribute('aria-expanded') !== 'false') return false;
    button.click();
    const expanded = !extra.hidden && button.getAttribute('aria-expanded') === 'true' && button.textContent.includes('Show fewer videos');
    button.click();
    return expanded && extra.hidden && button.getAttribute('aria-expanded') === 'false' && button.textContent.includes('View all videos');
  })()`), true, 'Video expansion and collapse must agree with ARIA and labels');
  console.log('PASS: Porto empty state; delegated keyboard activation after Coimbra swap; invalid IDs; video expansion');

  // Save a real HTMX history entry without a planner, then restore it. Only fixture DOM is changed.
  // Normal search does not push history; use a pushing HTMX link to record the missing form.
  await evaluate(`
    document.querySelector('.trip-form').remove();
    const link = document.createElement('button');
    link.setAttribute('hx-get', '/process-form/?city_name=Coimbra');
    link.setAttribute('hx-target', '#content-area');
    link.setAttribute('hx-push-url', '/?without-planner-next');
    document.body.appendChild(link); htmx.process(link); link.click(); link.remove();
  `);
  await until('location.search === "?without-planner-next" && !!document.querySelector(".trip-form")');
  await travelHistory('back');
  assert.equal(await evaluate('document.querySelector(".trip-form")'), null);
  assert.equal(await evaluate('document.querySelectorAll("#trip-status").length'), 1);
  assert.equal(await evaluate('document.querySelector("#trip-status").textContent'), '');
  assert.deepEqual(errors, [], 'No unexpected browser exceptions');
  console.log('PASS: actual history restoration without planner; no browser exceptions');
} finally {
  await call('Fetch.disable').catch(() => {});
  for (const item of pending.values()) { clearTimeout(item.timer); item.reject(new Error('CDP closed')); }
  pending.clear();
  ws.close();
}
