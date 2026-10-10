// CDP_PORT=9437 node tasks/redesign/checks/vendor-assets.mjs
// Start an isolated Chromium first. The helper runs the deterministic design preview.
import assert from 'node:assert/strict';
import { mkdir, writeFile } from 'node:fs/promises';
import { connect, startPreview } from './lib/browser.mjs';

const output = new URL('../../../docs/tdd/artifacts/dependabot-vendor-assets/', import.meta.url);
await mkdir(output, { recursive: true });
const preview = await startPreview();
let browser;
try {
  browser = await connect();
  const consoleErrors = [];
  // Log and console events complement the helper's JavaScript exception checks.
  const tabs = await (await fetch(`http://127.0.0.1:${process.env.CDP_PORT || 9227}/json/list`)).json();
  const socket = new WebSocket(tabs.find(tab => tab.type === 'page').webSocketDebuggerUrl);
  await new Promise((resolve, reject) => {
    socket.addEventListener('open', resolve, { once: true });
    socket.addEventListener('error', reject, { once: true });
  });
  socket.addEventListener('message', event => {
    const { method, params } = JSON.parse(event.data);
    if (method === 'Runtime.consoleAPICalled' && params.type === 'error') consoleErrors.push(params.args);
    if (method === 'Log.entryAdded' && params.entry.level === 'error') consoleErrors.push(params.entry.text);
  });
  socket.send(JSON.stringify({ id: 1, method: 'Runtime.enable' }));
  socket.send(JSON.stringify({ id: 2, method: 'Log.enable' }));
  try {
    const results = [];
    for (const [width, height] of [[375, 812], [1440, 1000]]) {
      await browser.viewport(width, height);
      await browser.navigate(preview.url);
      const state = await browser.evaluate(`(async () => {
        await document.fonts.load('16px bootstrap-icons');
        const paths = [
          'bootstrap/dist/css/bootstrap.min.css', 'bootstrap-icons/font/bootstrap-icons.min.css',
          'bootstrap-icons/font/fonts/bootstrap-icons.woff2', 'bootstrap-icons/font/fonts/bootstrap-icons.woff',
          'htmx.org/dist/htmx.min.js'
        ];
        const assets = await Promise.all(paths.map(async path => {
          const response = await fetch('/vendor/' + path);
          return { path, status: response.status, size: (await response.arrayBuffer()).byteLength };
        }));
        const rect = selector => document.querySelector(selector).getBoundingClientRect().toJSON();
        return { assets, overflow: document.documentElement.scrollWidth, viewport: innerWidth,
          bootstrap: !![...document.styleSheets].find(sheet => sheet.href?.endsWith('/vendor/bootstrap/dist/css/bootstrap.min.css'))?.cssRules.length,
          icons: document.fonts.check('16px bootstrap-icons'),
          iconFamily: getComputedStyle(document.querySelector('.bi'), '::before').fontFamily,
          htmx: document.querySelector('script[src*="htmx"]').getAttribute('src'),
          form: rect('.tt-overview-planner'), map: rect('.tt-overview-map') };
      })()`);
      for (const asset of state.assets) {
        assert.equal(asset.status, 200, asset.path);
        assert.ok(asset.size > 0, asset.path);
      }
      assert.equal(state.bootstrap, true);
      assert.equal(state.icons, true);
      assert.match(state.iconFamily, /bootstrap-icons/);
      assert.equal(state.htmx, '/vendor/htmx.org/dist/htmx.min.js');
      assert.ok(state.overflow <= width);
      if (width === 375) assert.ok(state.form.y < state.map.y);
      await browser.evaluate("document.querySelector('.trip-form').requestSubmit()");
      await browser.until("document.querySelectorAll('.itinerary-day').length === 3 && !document.querySelector('.trip-submit').disabled");
      assert.ok(await browser.evaluate(`document.documentElement.scrollWidth <= ${width}`));
      await browser.evaluate("window.scrollTo({top: 0, behavior: 'instant'})");
      await writeFile(new URL(`${width}.png`, output), await browser.shot());
      results.push({ width, ...state });
      console.log(`PASS ${width}px: local CSS, fonts, htmx, planning, no overflow`);
    }
    assert.deepEqual(browser.errors, []);
    assert.deepEqual(consoleErrors, []);
    await writeFile(new URL('results.json', output), JSON.stringify({ browser: browser.product, results, exceptions: browser.errors, consoleErrors }, null, 2) + '\n');
    console.log('PASS: no JavaScript exceptions or console errors');
  } finally {
    socket.close();
  }
} finally {
  if (browser) await browser.close();
  preview.stop();
}
