#!/usr/bin/env node
// Browser acceptance for destination photos against the deterministic design preview.
//
//   HTMX_PATH=/tmp/htmx.js PREVIEW_URL=http://127.0.0.1:8087 CDP_PORT=9227 node <this file>
//
// Run the default scenario. Set SLOW_PREVIEW_URL to a second preview started with -slow to add the
// overlapping-request check. The fixture photo source is local, so this proves the page's photo
// behaviour and says nothing about Wikimedia; live lookups are checked separately.
import assert from 'node:assert/strict';
import { readFile, mkdir, writeFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';

const base = process.env.PREVIEW_URL || 'http://127.0.0.1:8087';
const slowBase = process.env.SLOW_PREVIEW_URL || '';
assert.ok(['127.0.0.1', 'localhost'].includes(new URL(base).hostname), 'Use a local fixture');
assert.ok(process.env.HTMX_PATH, 'Set HTMX_PATH to the pinned 1.9.11 script');
const htmx = await readFile(process.env.HTMX_PATH);
const template = await readFile(new URL('../../../views/index.go.tpl', import.meta.url), 'utf8');
const hash = template.match(/htmx.org@1\.9\.11" integrity="sha384-([^"]+)"/)?.[1];
assert.equal(createHash('sha384').update(htmx).digest('base64'), hash, 'Pinned HTMX integrity');
const output = new URL(process.env.OUTPUT_DIR || '../../artifacts/destination-photos/fixture/', import.meta.url);
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
const errors = [];
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
  } else if (message.method === 'Runtime.exceptionThrown') {
    errors.push(message.params.exceptionDetails.exception?.description || JSON.stringify(message.params.exceptionDetails));
  } else if (message.method === 'Fetch.requestPaused') {
    const { requestId, request } = message.params;
    const action = request.url === 'https://unpkg.com/htmx.org@1.9.11'
      ? call('Fetch.fulfillRequest', { requestId, responseCode: 200, body: htmx.toString('base64'), responseHeaders: [
        { name: 'Content-Type', value: 'text/javascript' }, { name: 'Access-Control-Allow-Origin', value: '*' }] })
      : call('Fetch.continueRequest', { requestId });
    action.catch(error => errors.push(String(error)));
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
    await new Promise(resolve => setTimeout(resolve, 50));
  }
  throw new Error(`Timed out: ${expression}`);
};
const screenshot = async name => {
  await evaluate("window.scrollTo({top: 0, behavior: 'instant'})");
  const shot = await call('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false });
  await writeFile(new URL(`${name}.png`, output), Buffer.from(shot.data, 'base64'));
};

// What the page shows for the destination, read in the browser.
const state = () => evaluate(`(() => {
  const figure = document.querySelector('.destination-photo');
  const img = document.querySelector('.destination-image');
  const visible = node => !!node && getComputedStyle(node).display !== 'none' && node.getBoundingClientRect().width > 0;
  const rect = figure?.getBoundingClientRect();
  return {
    title: document.querySelector('.destination-title')?.textContent.trim() || null,
    country: document.querySelector('.trip-form')?.elements.country?.value || null,
    state: figure?.dataset.photoState || null,
    src: img?.src || null,
    alt: img?.alt || null,
    loaded: !!img && img.complete && img.naturalWidth > 0,
    naturalWidth: img?.naturalWidth || 0,
    imageVisible: visible(img),
    credit: document.querySelector('.destination-photo-credit')?.textContent.trim() || null,
    creditVisible: visible(document.querySelector('.destination-photo-credit')),
    caption: document.querySelector('.destination-photo-caption')?.textContent.trim() || null,
    captionVisible: visible(document.querySelector('.destination-photo-caption')),
    fallbackVisible: visible(document.querySelector('.destination-photo-fallback')),
    fallbackText: document.querySelector('.destination-photo-fallback')?.textContent.trim() || null,
    images: document.querySelectorAll('.destination-image').length,
    credits: document.querySelectorAll('.destination-photo-credit').length,
    frame: rect ? { width: Math.round(rect.width), height: Math.round(rect.height) } : null,
    overflow: document.documentElement.scrollWidth, viewport: innerWidth,
    weather: !!document.querySelector('.weather-metrics'), planner: !!document.querySelector('.trip-form')
  };
})()`);

// The photo a destination must show. The country tells the two Paris cities apart.
const expected = ({ title, country }) => {
  if (title === 'Lisbon') return { src: /\/images\/destinations\/lisbon-alfama\.jpg$/, credit: /Qiqritiq/ };
  if (title === 'Paris') return country === 'us'
    ? { src: /\/fixture-photos\/paris-texas\.png$/, credit: /Paris, Texas/ }
    : { src: /\/fixture-photos\/paris\.png$/, credit: /\(Paris\)/ };
  if (title === 'Porto') return { src: /\/fixture-photos\/porto\.png$/, credit: /\(Porto\)/ };
  if (title === 'Brokenimage') return { src: /\/fixture-photos\/missing\.png$/, credit: /broken image/ };
  return null; // no photo: Nophoto, Photoerror, Coimbra
};
// Every observation of the page must associate the destination with its own photo and credit.
const assertCoherent = (s, context) => {
  const want = expected(s);
  if (!want) {
    assert.equal(s.state, 'unavailable', `${context}: ${s.title} has no photo`);
    assert.equal(s.images, 0, `${context}: no stale photo may remain`);
    assert.equal(s.credits, 0, `${context}: no stale credit may remain`);
    return;
  }
  assert.match(s.src, want.src, `${context}: ${s.title}/${s.country} photo`);
  assert.match(s.credit, want.credit, `${context}: ${s.title}/${s.country} credit`);
  assert.equal(s.images, 1, `${context}: exactly one hero`);
  assert.equal(s.credits, 1, `${context}: exactly one credit`);
};
const assertShown = (s, width, context) => {
  assertCoherent(s, context);
  assert.equal(s.state, 'ready', `${context}: photo state`);
  assert.equal(s.loaded, true, `${context}: image must have actually loaded (complete and naturalWidth > 0)`);
  assert.ok(s.imageVisible && s.creditVisible, `${context}: image and credit visible`);
  assert.equal(s.fallbackVisible, false, `${context}: fallback hidden while the photo shows`);
  assert.ok(s.alt && s.alt.length > 3, `${context}: alt text`);
  assert.ok(s.weather && s.planner, `${context}: weather and planner render`);
  assert.ok(s.overflow <= width, `${context}: horizontal overflow ${s.overflow} > ${width}`);
};
const assertUnavailable = (s, width, context) => {
  assert.equal(s.state, 'unavailable', `${context}: state`);
  assert.equal(s.images, 0); assert.equal(s.credits, 0);
  assert.equal(s.fallbackVisible, true, `${context}: explicit unavailable message`);
  assert.equal(s.fallbackText, 'Photo unavailable');
  assert.equal(s.captionVisible, false, `${context}: no orphaned caption`);
  assert.ok(s.weather && s.planner, `${context}: weather and planner still render`);
  assert.ok(s.overflow <= width, `${context}: overflow`);
};
const assertFailed = (s, width, context) => {
  assert.equal(s.state, 'failed', `${context}: state`);
  assert.equal(s.imageVisible, false, `${context}: broken image hidden`);
  assert.equal(s.creditVisible, false, `${context}: credit hidden with the failed image`);
  assert.equal(s.captionVisible, false, `${context}: caption hidden with the failed image`);
  assert.equal(s.fallbackVisible, true, `${context}: fallback revealed`);
  assert.equal(s.fallbackText, 'Photo unavailable');
  assert.ok(s.weather && s.planner, `${context}: weather and planner still render`);
  assert.ok(s.overflow <= width, `${context}: overflow`);
};
const search = async term => {
  await evaluate(`document.querySelector('#city_name').value = ${JSON.stringify(term)}; document.querySelector('#destination-search').requestSubmit()`);
};
const searchAndSettle = async (term, title, extra = 'true') => {
  await search(term);
  await until(`document.querySelector('.destination-title')?.textContent.trim() === ${JSON.stringify(title)} && ${extra}`);
};
const loaded = 'document.querySelector(".destination-image")?.complete && document.querySelector(".destination-image").naturalWidth > 0';
const navigate = async url => {
  await call('Page.navigate', { url });
  await until("document.readyState === 'complete' && window.htmx?.version === '1.9.11' && !!document.querySelector('.destination-title')");
};
const results = [];
const check = (width, label) => console.log(`PASS ${width}px: ${label}`);
try {
  await call('Page.enable'); await call('Runtime.enable'); await call('Network.enable');
  await call('Network.setCacheDisabled', { cacheDisabled: true });
  await call('Fetch.enable', { patterns: [{ urlPattern: 'https://unpkg.com/htmx.org@1.9.11' }] });
  const browser = (await call('Browser.getVersion')).product;
  console.log(`Environment: ${browser}`);

  for (const [width, height] of [[375, 812], [1440, 1000]]) {
    await call('Emulation.setDeviceMetricsOverride', { width, height, deviceScaleFactor: 1, mobile: width <= 768 });
    const record = { width, height, steps: {} };

    // 1. Initial page load: Lisbon, the curated photograph.
    await navigate(`${base}/?destination-photos=${Date.now()}`);
    await until(loaded);
    const lisbon = await state();
    assertShown(lisbon, width, 'initial Lisbon');
    const frame = lisbon.frame;
    record.steps.lisbon = lisbon;
    await screenshot(`${width}-1-lisbon-initial`);
    check(width, 'initial page loads the curated Lisbon photo');

    // 2. HTMX search: Paris replaces Lisbon's photo, credit and caption.
    await searchAndSettle('Paris', 'Paris', loaded);
    const paris = await state();
    assertShown(paris, width, 'search Paris');
    assert.equal(paris.country, 'fr'); assert.equal(paris.caption, null, "Lisbon's caption must not follow the search");
    assert.deepEqual(paris.frame, frame, 'hero frame keeps its size between destinations');
    record.steps.paris = paris;
    await screenshot(`${width}-2-paris`);

    // 3. Another city, then the namesake with another country.
    await searchAndSettle('Porto, Portugal', 'Porto', loaded);
    const porto = await state();
    assertShown(porto, width, 'search Porto');
    assert.notEqual(porto.src, paris.src);
    record.steps.porto = porto;
    await screenshot(`${width}-3-porto`);
    await searchAndSettle('Paris, Texas', 'Paris', loaded);
    const texas = await state();
    assertShown(texas, width, 'search Paris, Texas');
    assert.equal(texas.country, 'us'); assert.notEqual(texas.src, paris.src, 'Paris, Texas must not show Paris, France');
    record.steps.texas = texas;
    await screenshot(`${width}-4-paris-texas`);
    check(width, 'Lisbon → Paris → Porto → Paris, Texas: each photo, credit and identity replaced together');

    // 4. Explicit unavailable states keep layout, weather and planner.
    for (const [term, title, why] of [['Nophoto', 'Nophoto', 'lookup found no photo'], ['Photoerror', 'Photoerror', 'provider failure'], ['Coimbra, Portugal', 'Coimbra', 'unlisted destination']]) {
      await searchAndSettle(term, title);
      const none = await state();
      assertUnavailable(none, width, why);
      assert.deepEqual(none.frame, frame, `${why}: hero frame keeps its size`);
      record.steps[title.toLowerCase()] = none;
      if (title === 'Nophoto') await screenshot(`${width}-5-unavailable`);
    }
    check(width, 'no-photo, provider-failure and unlisted destinations show the explicit unavailable state');

    // 5. A rapid sequence: nothing may pair one city with another's photo at any moment.
    await evaluate(`(() => {
      window.snapshots = [];
      const read = () => ({
        title: document.querySelector('.destination-title')?.textContent.trim() || null,
        country: document.querySelector('.trip-form')?.elements.country?.value || null,
        state: document.querySelector('.destination-photo')?.dataset.photoState || null,
        src: document.querySelector('.destination-image')?.src || null,
        credit: document.querySelector('.destination-photo-credit')?.textContent.trim() || null,
        images: document.querySelectorAll('.destination-image').length,
        credits: document.querySelectorAll('.destination-photo-credit').length
      });
      window.snapshotObserver?.disconnect();
      window.snapshotObserver = new MutationObserver(() => window.snapshots.push(read()));
      window.snapshotObserver.observe(document.querySelector('#content-area'), { childList: true, subtree: true, attributes: true });
      const form = document.querySelector('#destination-search');
      for (const term of ['Paris', 'Porto, Portugal', 'Paris, Texas', 'Nophoto', 'Porto, Portugal']) {
        document.querySelector('#city_name').value = term; form.requestSubmit();
      }
    })()`);
    await new Promise(resolve => setTimeout(resolve, 1500));
    await until(loaded + ' || document.querySelector(".destination-photo")?.dataset.photoState === "unavailable"');
    const snapshots = await evaluate('window.snapshots');
    await evaluate('window.snapshotObserver.disconnect()');
    assert.ok(snapshots.length > 0, 'the rapid sequence must swap the page');
    for (const snapshot of snapshots.filter(item => item.title)) assertCoherent(snapshot, `rapid sequence (${snapshot.title}/${snapshot.country})`);
    const settled = await state();
    assertCoherent(settled, 'rapid sequence settled');
    if (settled.state !== 'unavailable') assertShown(settled, width, 'rapid sequence settled');
    record.steps.rapid = { observations: snapshots.length, settled: `${settled.title}/${settled.country}` };
    check(width, `rapid sequence: ${snapshots.length} observations, all coherent; settled on ${settled.title}/${settled.country}`);

    // 6. Shared trip pages and reload, both Parises.
    for (const [slug, country, file] of [['paris-fr', 'fr', 'paris.png'], ['paris-us', 'us', 'paris-texas.png']]) {
      await navigate(`${base}/trip/${slug}`);
      await until(loaded);
      const shared = await state();
      assertShown(shared, width, `shared ${slug}`);
      assert.equal(shared.country, country); assert.ok(shared.src.endsWith(`/fixture-photos/${file}`));
      record.steps[slug] = shared;
    }
    await screenshot(`${width}-6-shared-paris-texas`);
    await call('Page.reload', { ignoreCache: true });
    await until("document.readyState === 'complete' && !!document.querySelector('.destination-title')");
    await until(loaded);
    assertShown(await state(), width, 'shared reload');
    check(width, 'shared trip pages and reload show the matching Paris photo');

    // 7. Back/Forward: plan Paris, then restore the destination before planning and return.
    await navigate(`${base}/?back-forward=${Date.now()}`);
    await evaluate(`localStorage.removeItem('htmx-history-cache'); window.restores = 0; document.body.addEventListener('htmx:historyRestore', () => window.restores++);`);
    await searchAndSettle('Paris', 'Paris', loaded);
    const beforePlan = await state();
    await evaluate("document.querySelector('.trip-form').requestSubmit()");
    await until("document.querySelectorAll('.itinerary-day').length === 3 && !document.querySelector('.trip-submit').disabled");
    assertShown(await state(), width, 'after planning');
    const plannedURL = await evaluate('location.href');
    assert.match(plannedURL, /\/trip\/paris-fr\?days=3/);
    await evaluate('history.back()');
    await until('window.restores === 1');
    await until(loaded);
    const back = await state();
    assertShown(back, width, 'after Back');
    assert.equal(back.src, beforePlan.src); assert.equal(back.title, 'Paris');
    await evaluate('history.forward()');
    await until('window.restores === 2');
    await until(loaded);
    const forward = await state();
    assertShown(forward, width, 'after Forward');
    assert.equal(forward.src, beforePlan.src);
    assert.equal(await evaluate("document.querySelectorAll('.itinerary-day').length"), 3, 'Forward restores the plan');
    check(width, 'plan → Back → Forward keeps the Paris photo loaded with one hero and one credit');

    // 8. A broken image URL: the page reveals the fallback and hides the credit.
    await navigate(`${base}/?city_name=Brokenimage&broken=${Date.now()}`);
    await until('document.querySelector(".destination-photo")?.dataset.photoState === "failed"');
    const brokenInitial = await state();
    assertFailed(brokenInitial, width, 'initial page, broken image');
    assert.deepEqual(brokenInitial.frame, frame, 'failed photo keeps the hero frame');
    record.steps.brokenInitial = brokenInitial;
    await screenshot(`${width}-7-broken-image-initial`);
    await navigate(`${base}/?failure-swap=${Date.now()}`);
    await searchAndSettle('Brokenimage', 'Brokenimage', 'document.querySelector(".destination-photo")?.dataset.photoState === "failed"');
    assertFailed(await state(), width, 'HTMX swap, broken image');
    await screenshot(`${width}-8-broken-image-swapped`);
    await searchAndSettle('Porto, Portugal', 'Porto', loaded);
    assertShown(await state(), width, 'a good photo replaces the failure state');
    await searchAndSettle('Brokenimage', 'Brokenimage', 'document.querySelector(".destination-photo")?.dataset.photoState === "failed"');
    // The failure state is an attribute, so it survives a history snapshot: plan, Back, and it must persist.
    await evaluate(`localStorage.removeItem('htmx-history-cache'); window.restores = 0; document.body.addEventListener('htmx:historyRestore', () => window.restores++);`);
    await evaluate("document.querySelector('.trip-form').requestSubmit()");
    await until("document.querySelectorAll('.itinerary-day').length === 3 && !document.querySelector('.trip-submit').disabled");
    await evaluate('history.back()');
    await until('window.restores === 1');
    await until('document.querySelector(".destination-photo")?.dataset.photoState === "failed"');
    assertFailed(await state(), width, 'history restore of a failed image');
    check(width, 'broken image: fallback revealed, credit and caption hidden, layout stable, survives swaps and history');

    results.push(record);
  }

  if (slowBase) {
    // Overlapping requests on a slow preview: an earlier city's response must never paint its
    // photo over a later search, nor mix into it.
    await call('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1000, deviceScaleFactor: 1, mobile: false });
    await navigate(`${slowBase}/?slow=${Date.now()}`);
    await evaluate(`window.snapshots = [];
      const read = () => ({ title: document.querySelector('.destination-title')?.textContent.trim() || null,
        country: document.querySelector('.trip-form')?.elements.country?.value || null,
        state: document.querySelector('.destination-photo')?.dataset.photoState || null,
        src: document.querySelector('.destination-image')?.src || null,
        credit: document.querySelector('.destination-photo-credit')?.textContent.trim() || null,
        images: document.querySelectorAll('.destination-image').length,
        credits: document.querySelectorAll('.destination-photo-credit').length });
      new MutationObserver(() => window.snapshots.push(read())).observe(document.querySelector('#content-area'), { childList: true, subtree: true, attributes: true });`);
    await search('Paris');
    await new Promise(resolve => setTimeout(resolve, 400));
    await search('Porto, Portugal');
    await new Promise(resolve => setTimeout(resolve, 7000));
    const snapshots = await evaluate('window.snapshots');
    for (const snapshot of snapshots.filter(item => item.title)) assertCoherent(snapshot, `slow overlap (${snapshot.title})`);
    const settled = await state();
    assertCoherent(settled, 'slow overlap settled');
    results.push({ slow: { observations: snapshots.length, settled: settled.title, titles: [...new Set(snapshots.map(item => item.title))] } });
    console.log(`PASS slow overlap: ${snapshots.length} observations coherent; settled on ${settled.title}; titles seen ${[...new Set(snapshots.map(item => item.title))].join(', ')}`);
  }

  assert.deepEqual(errors, [], 'Unexpected JavaScript exceptions');
  await writeFile(new URL('browser-results.json', output), JSON.stringify({ browser, htmx: '1.9.11', base, slowBase, results, errors }, null, 2));
  console.log('PASS: destination photo browser acceptance, no JavaScript exceptions');
  console.log(`Evidence: ${output.pathname}`);
} finally {
  await call('Fetch.disable').catch(() => {});
  ws.close();
}
