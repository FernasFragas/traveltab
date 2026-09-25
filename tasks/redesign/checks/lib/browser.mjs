// Shared helpers for the final-verification-2 browser checks (a11y-audit.mjs, scenarios.mjs).
// Node 24 only: built-in WebSocket and fetch. No packages.
//
// The checks start their own deterministic design preview processes (one per scenario, on free
// localhost ports) and attach to an ISOLATED Chromium/Brave that you started with its own
// --remote-debugging-port and a temporary --user-data-dir. They never touch another browser.
import assert from 'node:assert/strict';
import { spawn, execFileSync } from 'node:child_process';
import { readFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { createServer } from 'node:net';
import { inflateSync, deflateSync, crc32 } from 'node:zlib';
import { fileURLToPath } from 'node:url';

export const repoRoot = fileURLToPath(new URL('../../../../', import.meta.url));
export const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));

const freePort = () => new Promise((resolve, reject) => {
  const server = createServer();
  server.on('error', reject);
  server.listen(0, '127.0.0.1', () => { const { port } = server.address(); server.close(() => resolve(port)); });
});

// Builds (once) and starts the design preview for one scenario. Returns { url, mapUrl, stop }.
let builtBinary = null;
export const startPreview = async ({ scenario = 'default', slow = false } = {}) => {
  if (!builtBinary) {
    builtBinary = process.env.PREVIEW_BIN || `${process.env.TMPDIR || '/tmp'}/traveltab-preview-${process.pid}`;
    if (!process.env.PREVIEW_BIN) execFileSync('go', ['build', '-o', builtBinary, './cmd/design-preview'], { cwd: repoRoot, stdio: 'inherit' });
  }
  const [port, mapPort] = [await freePort(), await freePort()];
  const args = ['-addr', `127.0.0.1:${port}`, '-map-addr', `127.0.0.1:${mapPort}`, '-scenario', scenario];
  if (slow) args.push('-slow');
  const child = spawn(builtBinary, args, { cwd: repoRoot, stdio: 'ignore' });
  const url = `http://127.0.0.1:${port}`;
  const deadline = Date.now() + 15000;
  for (;;) {
    try { if ((await fetch(`${url}/styles.css`, { signal: AbortSignal.timeout(2000) })).status === 200) break; } catch { /* not up yet */ }
    if (Date.now() > deadline) { child.kill(); throw new Error(`preview ${scenario} did not start`); }
    await sleep(100);
  }
  return { url, mapUrl: `http://127.0.0.1:${mapPort}`, scenario, stop: () => child.kill('SIGKILL') };
};

// Hosts whose images the preview's pages reference (stop photos, video thumbnails) and the video embed.
const thirdParty = ['commons.wikimedia.org', 'upload.wikimedia.org', 'i.ytimg.com', 'www.youtube-nocookie.com'];

// A small generated PNG standing in for a third-party photograph: a sky-to-ground gradient
// ('synthetic') or pure white ('white', the worst case behind light overlay text).
const syntheticCache = new Map();
const syntheticPNG = mode => {
  if (syntheticCache.has(mode)) return syntheticCache.get(mode);
  const [width, height] = [320, 240];
  const raw = Buffer.alloc((width * 3 + 1) * height);
  for (let y = 0; y < height; y++) for (let x = 0; x < width; x++) {
    const o = y * (width * 3 + 1) + 1 + x * 3; const t = y / height;
    const rgb = mode === 'white' ? [255, 255, 255] : y > height * 0.7 ? [86 + (x % 40), 96, 80] : [Math.round(140 + 90 * t), Math.round(180 + 50 * t), Math.round(215 + 30 * t)];
    raw[o] = rgb[0]; raw[o + 1] = rgb[1]; raw[o + 2] = rgb[2];
  }
  const chunk = (type, data) => { const head = Buffer.alloc(8); head.writeUInt32BE(data.length, 0); head.write(type, 4); const tail = Buffer.alloc(4); tail.writeUInt32BE(crc32(Buffer.concat([head.subarray(4), data])), 0); return Buffer.concat([head, data, tail]); };
  const ihdr = Buffer.alloc(13); ihdr.writeUInt32BE(width, 0); ihdr.writeUInt32BE(height, 4); ihdr[8] = 8; ihdr[9] = 2;
  const png = Buffer.concat([Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]), chunk('IHDR', ihdr), chunk('IDAT', deflateSync(raw)), chunk('IEND', Buffer.alloc(0))]);
  syntheticCache.set(mode, png);
  return png;
};

// Attaches to the first about:blank page of the isolated browser.
export const connect = async ({ cdpPort = process.env.CDP_PORT || 9227, htmxPath = process.env.HTMX_PATH } = {}) => {
  assert.ok(htmxPath, 'Set HTMX_PATH to the pinned 1.9.11 script');
  const htmx = await readFile(htmxPath);
  const template = await readFile(new URL('views/index.go.tpl', `file://${repoRoot}`), 'utf8');
  const integrity = template.match(/htmx.org@1\.9\.11" integrity="sha384-([^"]+)"/)?.[1];
  assert.equal(createHash('sha384').update(htmx).digest('base64'), integrity, 'Pinned HTMX integrity');
  const tabs = await (await fetch(`http://127.0.0.1:${cdpPort}/json/list`, { signal: AbortSignal.timeout(5000) })).json();
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
  const errors = [];
  const events = [];
  // imageMode decides what third-party image and video hosts answer, so no run depends on them:
  // 'synthetic' (a generated gradient photo, the default), 'white' (worst case behind light
  // overlay text) or 'block' (the request fails, showing every image-error fallback).
  const state = { imageMode: 'synthetic', overrides: new Map() };
  const call = (method, params = {}, timeout = 30000) => new Promise((resolve, reject) => {
    const id = ++sequence;
    const timer = setTimeout(() => { pending.delete(id); reject(new Error(`CDP timeout: ${method}`)); }, timeout);
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
    } else if (message.method === 'Fetch.requestPaused') {
      const { requestId, request } = message.params;
      const url = new URL(request.url);
      let action;
      if (request.url === 'https://unpkg.com/htmx.org@1.9.11') {
        action = call('Fetch.fulfillRequest', { requestId, responseCode: 200, body: htmx.toString('base64'), responseHeaders: [
          { name: 'Content-Type', value: 'text/javascript' }, { name: 'Access-Control-Allow-Origin', value: '*' }] });
      } else if (state.overrides.has(url.pathname) && ['127.0.0.1', 'localhost'].includes(url.hostname)) {
        action = call('Fetch.fulfillRequest', { requestId, ...state.overrides.get(url.pathname) });
      } else if (thirdParty.includes(url.hostname)) {
        action = url.hostname === 'www.youtube-nocookie.com' || state.imageMode === 'block'
          ? call('Fetch.failRequest', { requestId, errorReason: 'BlockedByClient' })
          : call('Fetch.fulfillRequest', { requestId, responseCode: 200, body: syntheticPNG(state.imageMode).toString('base64'), responseHeaders: [
            { name: 'Content-Type', value: 'image/png' }, { name: 'Access-Control-Allow-Origin', value: '*' }] });
      } else {
        action = call('Fetch.continueRequest', { requestId });
      }
      action.catch(error => errors.push(String(error)));
    } else if (message.method === 'Network.loadingFailed' || message.method === 'Network.responseReceived') {
      events.push(message);
    }
  });
  const evaluate = async expression => {
    const result = await call('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true });
    assert.equal(result.exceptionDetails, undefined, JSON.stringify(result.exceptionDetails));
    return result.result.value;
  };
  const until = async (expression, timeout = 15000) => {
    const deadline = Date.now() + timeout;
    while (Date.now() < deadline) {
      if (await evaluate(expression)) return;
      await sleep(50);
    }
    throw new Error(`Timed out: ${expression}`);
  };
  await call('Page.enable'); await call('Runtime.enable'); await call('Network.enable');
  await call('Network.setCacheDisabled', { cacheDisabled: true });
  await call('Fetch.enable', { patterns: [{ urlPattern: 'https://unpkg.com/htmx.org@1.9.11' }, { urlPattern: 'http://127.0.0.1:*/*' },
    ...thirdParty.map(host => ({ urlPattern: `https://${host}/*` }))] });
  const product = (await call('Browser.getVersion')).product;
  const viewport = (width, height, extra = {}) => call('Emulation.setDeviceMetricsOverride', { width, height, deviceScaleFactor: 1, mobile: width <= 768, ...extra });
  const ready = () => until("document.readyState === 'complete' && window.htmx?.version === '1.9.11' && !!document.querySelector('.destination-title, .tt-empty-state, .trip-card')");
  // Waits until the hero image (when there is one) has either loaded or failed, and fonts are ready.
  const settle = async () => {
    await evaluate('document.fonts.ready.then(() => true)');
    await until("[...document.images].filter(i => i.loading !== 'lazy').every(i => i.complete)");
    await sleep(80);
  };
  const navigate = async url => { await call('Page.navigate', { url }); await ready(); await settle(); };
  const key = async (name, code, vk, modifiers = 0, text) => {
    await call('Input.dispatchKeyEvent', { type: 'keyDown', key: name, code, windowsVirtualKeyCode: vk, nativeVirtualKeyCode: vk, modifiers, ...(text ? { text } : {}) });
    await call('Input.dispatchKeyEvent', { type: 'keyUp', key: name, code, windowsVirtualKeyCode: vk, nativeVirtualKeyCode: vk, modifiers });
  };
  const tab_ = (shift = false) => key('Tab', 'Tab', 9, shift ? 8 : 0);
  const loadFonts = async () => evaluate('document.fonts.ready.then(() => document.fonts.size)');
  return { call, evaluate, until, viewport, navigate, ready, settle, key, tab: tab_, errors, events, state, product, loadFonts,
    shot: async (clip, format = 'png') => Buffer.from((await call('Page.captureScreenshot', { format, ...(clip ? { clip: { ...clip, scale: 1 } } : {}), captureBeyondViewport: !!clip })).data, 'base64'),
    close: async () => { await call('Fetch.disable').catch(() => {}); ws.close(); } };
};

// --- PNG decoding (8-bit RGB/RGBA, non-interlaced: what Chromium's screenshots are) ---
export const decodePNG = buffer => {
  assert.equal(buffer.subarray(1, 4).toString(), 'PNG');
  let offset = 8, width = 0, height = 0, depth = 0, colour = 0; const idat = [];
  while (offset < buffer.length) {
    const length = buffer.readUInt32BE(offset); const type = buffer.subarray(offset + 4, offset + 8).toString();
    const data = buffer.subarray(offset + 8, offset + 8 + length);
    if (type === 'IHDR') { width = data.readUInt32BE(0); height = data.readUInt32BE(4); depth = data[8]; colour = data[9]; assert.equal(data[12], 0, 'interlaced PNG'); }
    if (type === 'IDAT') idat.push(data);
    offset += length + 12;
  }
  assert.equal(depth, 8); assert.ok(colour === 2 || colour === 6, `PNG colour type ${colour}`);
  const channels = colour === 6 ? 4 : 3, stride = width * channels;
  const raw = inflateSync(Buffer.concat(idat)); const out = Buffer.alloc(width * height * 4);
  let prev = Buffer.alloc(stride);
  for (let y = 0; y < height; y++) {
    const filter = raw[y * (stride + 1)]; const line = Buffer.from(raw.subarray(y * (stride + 1) + 1, (y + 1) * (stride + 1)));
    for (let x = 0; x < stride; x++) {
      const a = x >= channels ? line[x - channels] : 0, b = prev[x], c = x >= channels ? prev[x - channels] : 0;
      let add = 0;
      if (filter === 1) add = a; else if (filter === 2) add = b; else if (filter === 3) add = (a + b) >> 1;
      else if (filter === 4) { const p = a + b - c, pa = Math.abs(p - a), pb = Math.abs(p - b), pc = Math.abs(p - c); add = pa <= pb && pa <= pc ? a : pb <= pc ? b : c; }
      line[x] = (line[x] + add) & 255;
    }
    for (let x = 0; x < width; x++) {
      out[(y * width + x) * 4] = line[x * channels]; out[(y * width + x) * 4 + 1] = line[x * channels + 1]; out[(y * width + x) * 4 + 2] = line[x * channels + 2];
      out[(y * width + x) * 4 + 3] = channels === 4 ? line[x * channels + 3] : 255;
    }
    prev = line;
  }
  return { width, height, data: out };
};

// --- WCAG 2.x contrast ---
const channel = c => { c /= 255; return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4; };
export const luminance = ([r, g, b]) => 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b);
export const ratio = (a, b) => { const [hi, lo] = [luminance(a), luminance(b)].sort((p, q) => q - p); return (hi + 0.05) / (lo + 0.05); };
export const composite = (fg, alpha, bg) => fg.map((c, i) => c * alpha + bg[i] * (1 - alpha));
export const round = (n, places = 2) => Math.round(n * 10 ** places) / 10 ** places;
