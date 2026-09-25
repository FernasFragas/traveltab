#!/usr/bin/env node
// Live destination-photo check. NOT part of the offline suite: it needs internet access, the
// results depend on Wikimedia, and the fixture preview must be started with -live-photos:
//
//   go run ./cmd/design-preview -addr 127.0.0.1:8087 -map-addr 127.0.0.1:8088 -live-photos
//   HTMX_PATH=/tmp/htmx.js PREVIEW_URL=http://127.0.0.1:8087 CDP_PORT=9227 node <this file>
//
// Every search is a cold lookup (the preview has no photo cache). Weather, videos and the map
// remain fixtures; only the photograph, its credit and its license come from Wikimedia.
import assert from 'node:assert/strict';
import { readFile, mkdir, writeFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';

const base = process.env.PREVIEW_URL || 'http://127.0.0.1:8087';
assert.ok(['127.0.0.1', 'localhost'].includes(new URL(base).hostname), 'Use a local preview');
assert.ok(process.env.HTMX_PATH, 'Set HTMX_PATH to the pinned 1.9.11 script');
const htmx = await readFile(process.env.HTMX_PATH);
const template = await readFile(new URL('../../../views/index.go.tpl', import.meta.url), 'utf8');
const hash = template.match(/htmx.org@1\.9\.11" integrity="sha384-([^"]+)"/)?.[1];
assert.equal(createHash('sha384').update(htmx).digest('base64'), hash, 'Pinned HTMX integrity');
const output = new URL(process.env.OUTPUT_DIR || '../../artifacts/destination-photos/live/', import.meta.url);
await mkdir(output, { recursive: true });
const tabs = await (await fetch(`http://127.0.0.1:${process.env.CDP_PORT || 9227}/json/list`, { signal: AbortSignal.timeout(5000) })).json();
const tab = tabs.find(t => t.type === 'page');
assert.ok(tab, 'Start an isolated Chromium/Brave with an about:blank tab');
const ws = new WebSocket(tab.webSocketDebuggerUrl);
await new Promise((resolve, reject) => {
  const timer = setTimeout(() => reject(new Error('CDP connection timeout')), 5000);
  ws.addEventListener('open', () => { clearTimeout(timer); resolve(); }, { once: true });
  ws.addEventListener('error', reject, { once: true });
});
let sequence = 0;
const pending = new Map();
const call = (method, params = {}) => new Promise((resolve, reject) => {
  const id = ++sequence;
  const timer = setTimeout(() => { pending.delete(id); reject(new Error(`CDP timeout: ${method}`)); }, 20000);
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
  } else if (message.method === 'Fetch.requestPaused') {
    const { requestId, request } = message.params;
    (request.url === 'https://unpkg.com/htmx.org@1.9.11'
      ? call('Fetch.fulfillRequest', { requestId, responseCode: 200, body: htmx.toString('base64'), responseHeaders: [
        { name: 'Content-Type', value: 'text/javascript' }, { name: 'Access-Control-Allow-Origin', value: '*' }] })
      : call('Fetch.continueRequest', { requestId })).catch(() => {});
  }
});
const evaluate = async expression => {
  const result = await call('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true });
  assert.equal(result.exceptionDetails, undefined, JSON.stringify(result.exceptionDetails));
  return result.result.value;
};
const until = async (expression, timeout = 25000) => {
  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    if (await evaluate(expression)) return true;
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  return false;
};

// query -> what identity the page must end up showing. The Paris entries are the namesake pair.
const cases = [
  { query: 'Paris', title: 'Paris', country: 'fr', mustMatch: /Tour_Eiffel|Eiffel|Paris/i },
  { query: 'Paris, Texas', title: 'Paris', country: 'us', mustMatch: /Texas/i },
  { query: 'Porto, Portugal', title: 'Porto', country: 'pt', mustMatch: /Porto/i },
  { query: 'Tokyo', title: 'Tokyo', country: 'jp', mustMatch: /Shinjuku|Tokyo/i },
  { query: 'Tavira, Portugal', title: 'Tavira', country: 'pt', mustMatch: /Tavira/i },
];
const results = [];
try {
  await call('Page.enable'); await call('Runtime.enable'); await call('Network.enable');
  await call('Network.setCacheDisabled', { cacheDisabled: true });
  await call('Fetch.enable', { patterns: [{ urlPattern: 'https://unpkg.com/htmx.org@1.9.11' }] });
  const browser = (await call('Browser.getVersion')).product;
  for (const [width, height] of [[1440, 1000], [375, 812]]) {
    await call('Emulation.setDeviceMetricsOverride', { width, height, deviceScaleFactor: 1, mobile: width < 700 });
    await call('Page.navigate', { url: `${base}/?live=${Date.now()}` });
    await until("document.readyState === 'complete' && window.htmx?.version === '1.9.11'");
    for (const item of cases) {
      // Server latency for a cold search response, measured outside the browser.
      const started = performance.now();
      const res = await fetch(new URL(`/process-form/?city_name=${encodeURIComponent(item.query)}`, base), { headers: { 'HX-Request': 'true' }, signal: AbortSignal.timeout(20000) });
      const html = await res.text();
      const searchMs = Math.round(performance.now() - started);
      assert.equal(res.status, 200);
      const src = html.match(/<img class="destination-image" src="([^"]+)"/)?.[1] || null;
      // Now the same search in the browser, watching the actual image load.
      await evaluate(`document.querySelector('#city_name').value = ${JSON.stringify(item.query)}; document.querySelector('#destination-search').requestSubmit()`);
      const swapped = await until(`document.querySelector('.destination-title')?.textContent.trim() === ${JSON.stringify(item.title)} && document.querySelector('.trip-form')?.elements.country.value === ${JSON.stringify(item.country)}`);
      const shown = await until('document.querySelector(".destination-image")?.complete && document.querySelector(".destination-image").naturalWidth > 0');
      const page = await evaluate(`(() => {
        const img = document.querySelector('.destination-image');
        const links = [...document.querySelectorAll('.destination-photo-credit a')].map(a => ({ text: a.textContent.trim(), href: a.href }));
        return { state: document.querySelector('.destination-photo')?.dataset.photoState, src: img?.src || null, alt: img?.alt || null,
          naturalWidth: img?.naturalWidth || 0, naturalHeight: img?.naturalHeight || 0, credit: document.querySelector('.destination-photo-credit')?.textContent.trim() || null,
          links, overflow: document.documentElement.scrollWidth };
      })()`);
      const record = { width, query: item.query, expected: `${item.title}/${item.country}`, searchMs, swapped, imageLoaded: shown, ...page };
      results.push(record);
      const ok = swapped && shown && page.state === 'ready' && page.overflow <= width && item.mustMatch.test(decodeURIComponent(page.src || ''));
      console.log(`${ok ? 'PASS' : 'FAIL'} ${width}px ${item.query}: response ${searchMs} ms, image ${page.naturalWidth}x${page.naturalHeight}, ${page.credit}`);
      console.log(`     ${page.src}`);
      if (width === 1440 || item.title === 'Paris') {
        await evaluate("window.scrollTo({top: 0, behavior: 'instant'})");
        const shot = await call('Page.captureScreenshot', { format: 'png' });
        await writeFile(new URL(`${width}-${item.title.toLowerCase()}-${item.country}.png`, output), Buffer.from(shot.data, 'base64'));
      }
    }
  }
  await writeFile(new URL('browser-results.json', output), JSON.stringify({ browser, base, results }, null, 2));
  const failed = results.filter(r => !(r.swapped && r.imageLoaded && r.state === 'ready'));
  console.log(failed.length ? `FAILED ${failed.length} of ${results.length} live checks` : `PASS: ${results.length} live checks`);
  console.log(`Evidence: ${output.pathname}`);
  if (failed.length) process.exitCode = 1;
} finally {
  await call('Fetch.disable').catch(() => {});
  ws.close();
}
