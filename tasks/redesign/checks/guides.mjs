#!/usr/bin/env node
// Browser acceptance for reviewed city guides against the deterministic design preview.
//
//   HTMX_PATH=/tmp/htmx.js PREVIEW_URL=http://127.0.0.1:8087 CDP_PORT=9227 node <this file>
//
// Run the default scenario. Set FALLBACK_PREVIEW_URL to a second preview started with
// -scenario fallback to add the "no guide anywhere" check. The preview embeds the same reviewed
// guides/guides.json the site serves, so this proves the page's guide states, attribution,
// escaping and layout; it says nothing about Wikivoyage itself (the review is in
// tasks/guides-review.md). Screenshots and JSON go to tasks/artifacts/guides/ (or OUTPUT_DIR).
import assert from 'node:assert/strict';
import { readFile, mkdir, writeFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';

const base = process.env.PREVIEW_URL || 'http://127.0.0.1:8087';
const fallbackBase = process.env.FALLBACK_PREVIEW_URL || '';
assert.ok(['127.0.0.1', 'localhost'].includes(new URL(base).hostname), 'Use a local fixture');
assert.ok(process.env.HTMX_PATH, 'Set HTMX_PATH to the pinned 1.9.11 script');
const htmx = await readFile(process.env.HTMX_PATH);
const template = await readFile(new URL('../../../views/index.go.tpl', import.meta.url), 'utf8');
const hash = template.match(/htmx.org@1\.9\.11" integrity="sha384-([^"]+)"/)?.[1];
assert.equal(createHash('sha384').update(htmx).digest('base64'), hash, 'Pinned HTMX integrity');
const guides = JSON.parse(await readFile(new URL('../../../guides/guides.json', import.meta.url), 'utf8'));
const output = new URL(process.env.OUTPUT_DIR || '../../artifacts/guides/', import.meta.url);
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


const screenshotOf = async (name, selector) => {
  // Scroll the element to the top of the viewport (below the sticky header) and capture the viewport.
  await evaluate(`(() => { const el = document.querySelector(${JSON.stringify(selector)}); const y = el.getBoundingClientRect().top + scrollY - 90; window.scrollTo({ top: Math.max(0, y), behavior: 'instant' }); })()`);
  await new Promise(resolve => setTimeout(resolve, 120));
  const shot = await call('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false });
  await writeFile(new URL(`${name}.png`, output), Buffer.from(shot.data, 'base64'));
};

// What the page shows for the guide, read in the browser.
const state = () => evaluate(`(() => {
  const visible = node => !!node && getComputedStyle(node).display !== 'none' && node.getBoundingClientRect().width > 0;
  const card = document.querySelector('.guide-card');
  const links = [...document.querySelectorAll('.guide-attribution a')].map(a => ({ text: a.textContent.trim(), href: a.getAttribute('href'), target: a.target, rel: a.rel }));
  const rect = card?.getBoundingClientRect();
  return {
    title: document.querySelector('.destination-title')?.textContent.trim() || null,
    country: document.querySelector('.trip-form')?.elements.country?.value || null,
    cards: document.querySelectorAll('.guide-card').length,
    state: card?.dataset.guideState || null,
    heading: document.querySelector('.guide-title')?.textContent.trim() || null,
    paragraphs: [...document.querySelectorAll('.guide-paragraph')].map(p => p.textContent.trim()),
    attribution: document.querySelector('.guide-attribution')?.textContent.replace(/\\s+/g, ' ').trim() || null,
    review: document.querySelector('.guide-review')?.textContent.trim() || null,
    links,
    searchLink: document.querySelector('.guide-link') ? { href: document.querySelector('.guide-link').getAttribute('href'), text: document.querySelector('.guide-link').textContent.trim(), height: Math.round(document.querySelector('.guide-link').getBoundingClientRect().height) } : null,
    inOverview: !!document.querySelector('#overview .guide-card'),
    afterGrid: !!document.querySelector('#overview .tt-overview-grid + .guide-card'),
    visible: visible(card),
    box: rect ? { width: Math.round(rect.width), left: Math.round(rect.left), right: Math.round(rect.right) } : null,
    overflow: document.documentElement.scrollWidth, viewport: innerWidth,
    sections: ['overview', 'itinerary', 'stays', 'videos'].map(id => document.querySelectorAll('section#' + id).length),
    tripCards: document.querySelectorAll('#trip-card').length,
    planner: !!document.querySelector('.trip-form'), weather: !!document.querySelector('.weather-metrics')
  };
})()`);

// The guide each destination must show. Keys match guides/guides.json; the country tells the two Parises apart.
const keyOf = ({ title, country }) => `${title.toLowerCase()}-${country}`;
const shipped = s => guides[keyOf(s)]?.reviewed ? guides[keyOf(s)] : null;
const isFixtureOnly = s => s.title === 'Longguide';
const assertCoherent = (s, context) => {
  assert.equal(s.cards, 1, `${context}: exactly one guide card`);
  assert.equal(s.heading, `About ${s.title}`, `${context}: the guide is about the destination shown`);
  if (isFixtureOnly(s)) { assert.equal(s.state, 'reviewed', context); return; }
  const entry = shipped(s);
  if (!entry) {
    assert.equal(s.state, 'missing', `${context}: ${s.title}/${s.country} has no reviewed guide`);
    assert.equal(s.attribution, null, `${context}: no attribution without a guide`);
    assert.deepEqual(s.paragraphs.length, 1, `${context}: only the one missing-state sentence`);
    return;
  }
  assert.equal(s.state, 'reviewed', `${context}: ${s.title}/${s.country} has a reviewed guide`);
  assert.equal(s.paragraphs.join('\n\n'), entry.intro.split('\n\n').map(p => p.replace(/\s+/g, ' ').trim()).join('\n\n'), `${context}: the reviewed text, unchanged`);
  assert.match(s.attribution, new RegExp(`revision ${entry.wikivoyage_revision}\\b`), `${context}: this city's revision`);
  assert.match(s.attribution, new RegExp(`Wikivoyage: ${entry.wikivoyage_title}`), `${context}: this city's article`);
};
const assertLayout = (s, width, context) => {
  assert.ok(s.visible, `${context}: card visible`);
  assert.ok(s.inOverview && s.afterGrid, `${context}: the card follows the overview grid inside #overview`);
  assert.deepEqual(s.sections, [1, 1, 1, 1], `${context}: each section id exactly once`);
  assert.equal(s.tripCards, 1, `${context}: one #trip-card`);
  assert.ok(s.weather && s.planner, `${context}: weather and planner render`);
  assert.ok(s.overflow <= width, `${context}: horizontal overflow ${s.overflow} > ${width}`);
  assert.ok(s.box.left >= 0 && s.box.right <= width, `${context}: card inside the viewport (${s.box.left}..${s.box.right})`);
};
const assertReviewed = (s, width, context) => {
  assertCoherent(s, context); assertLayout(s, width, context);
  assert.equal(s.state, 'reviewed', context);
  assert.equal(s.links.length, 4, `${context}: article, revision, authors and license links`);
  const [page, revision, history, license] = s.links;
  assert.match(page.href, /^https:\/\/en\.wikivoyage\.org\/wiki\//); assert.match(revision.href, /^https:\/\/en\.wikivoyage\.org\/w\/index\.php\?oldid=\d+&title=/);
  assert.match(history.href, /action=history/); assert.equal(license.href, 'https://creativecommons.org/licenses/by-sa/4.0/'); assert.equal(license.text, 'CC BY-SA 4.0');
  for (const link of s.links) { assert.equal(link.target, '_blank'); assert.match(link.rel, /noopener/); assert.match(link.rel, /noreferrer/); }
  assert.match(s.review, /^Summarised and reworded\. Checked against that revision on \d{4}-\d{2}-\d{2}\.$/, `${context}: says it was adapted and when it was checked`);
};
// forced: the preview has no guides at all (the fallback scenario), so shipped guides do not apply.
const assertMissing = (s, width, context, forced = false) => {
  if (forced) { assert.equal(s.cards, 1, context); assert.equal(s.heading, `About ${s.title}`, context); assert.equal(s.attribution, null, context); }
  else assertCoherent(s, context);
  assertLayout(s, width, context);
  assert.equal(s.state, 'missing', context);
  assert.match(s.paragraphs[0], /^We don't have a reviewed guide for .+ yet\. Guides appear here only after a person has checked them against their source\.$/);
  assert.ok(s.searchLink && s.searchLink.href.startsWith('https://en.wikivoyage.org/w/index.php?search='), `${context}: Wikivoyage search link`);
  assert.ok(s.searchLink.height >= 44, `${context}: the link is a 44px target (got ${s.searchLink.height})`);
  assert.equal(s.links.length, 0, `${context}: no attribution links`);
};
const assertCard = (s, width, context) => (s.state === 'reviewed' ? assertReviewed : assertMissing)(s, width, context);

// WCAG contrast of every guide text colour against the card's own background.
const contrast = () => evaluate(`(() => {
  const parse = value => value.match(/[\\d.]+/g).slice(0, 3).map(Number);
  const lum = ([r, g, b]) => { const f = c => { c /= 255; return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4; }; return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b); };
  const ratio = (a, b) => { const [x, y] = [lum(a), lum(b)].sort((p, q) => q - p); return (x + 0.05) / (y + 0.05); };
  const background = parse(getComputedStyle(document.querySelector('.guide-card')).backgroundColor);
  const out = {};
  for (const selector of ['.guide-eyebrow', '.guide-badge', '.guide-title', '.guide-paragraph', '.guide-source', '.guide-attribution a', '.guide-link']) {
    const node = document.querySelector(selector);
    if (node) out[selector] = Math.round(ratio(parse(getComputedStyle(node).color), background) * 100) / 100;
  }
  return out;
})()`);

const search = async term => {
  await evaluate(`document.querySelector('#city_name').value = ${JSON.stringify(term)}; document.querySelector('#destination-search').requestSubmit()`);
};
// Wait for the swap itself: two searches can share a title (Paris, Paris, Texas), so the old card
// node is remembered and the new one must be a different element.
const searchAndSettle = async (term, title) => {
  await evaluate("window.previousGuideCard = document.querySelector('.guide-card')");
  await search(term);
  await until(`document.querySelector('.destination-title')?.textContent.trim() === ${JSON.stringify(title)} && !!document.querySelector('.guide-card') && document.querySelector('.guide-card') !== window.previousGuideCard`);
};
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

    // 1. Initial page load: Lisbon, a reviewed guide.
    await navigate(`${base}/?guides=${Date.now()}`);
    const lisbon = await state();
    assertReviewed(lisbon, width, 'initial Lisbon');
    record.steps.lisbon = lisbon;
    record.steps.contrast = await contrast();
    for (const [selector, ratio] of Object.entries(record.steps.contrast)) assert.ok(ratio >= 4.5, `contrast of ${selector} is ${ratio}, below 4.5:1`);
    await screenshotOf(`${width}-1-lisbon-reviewed`, '#overview .guide-card');
    await screenshot(`${width}-1-lisbon-top`);
    check(width, `initial page shows Lisbon's reviewed guide with attribution; text contrast ${Math.min(...Object.values(record.steps.contrast))}:1 or better`);

    // 2. HTMX searches replace the guide together with the destination.
    await searchAndSettle('Paris', 'Paris');
    const paris = await state(); assertReviewed(paris, width, 'search Paris');
    assert.notEqual(paris.paragraphs[0], lisbon.paragraphs[0], "Lisbon's text must not follow the search");
    record.steps.paris = paris;
    await screenshotOf(`${width}-2-paris-reviewed`, '#overview .guide-card');
    await searchAndSettle('Paris, Texas', 'Paris');
    const texas = await state(); assert.equal(texas.country, 'us');
    assertMissing(texas, width, 'search Paris, Texas');
    assert.ok(!texas.paragraphs.join(' ').includes(paris.paragraphs[0].slice(0, 40)), "Paris, Texas must not show Paris, France's guide");
    record.steps.texas = texas;
    await screenshotOf(`${width}-3-paris-texas-missing`, '#overview .guide-card');
    await searchAndSettle('Porto, Portugal', 'Porto');
    const porto = await state(); assertReviewed(porto, width, 'search Porto'); record.steps.porto = porto;
    await searchAndSettle('Coimbra, Portugal', 'Coimbra');
    const coimbra = await state(); assertMissing(coimbra, width, 'search Coimbra'); record.steps.coimbra = coimbra;
    await screenshotOf(`${width}-4-coimbra-missing`, '#overview .guide-card');
    record.steps.missingContrast = await contrast();
    for (const [selector, ratio] of Object.entries(record.steps.missingContrast)) assert.ok(ratio >= 4.5, `missing-state contrast of ${selector} is ${ratio}`);
    check(width, 'Lisbon → Paris → Paris, Texas → Porto → Coimbra: the guide, its attribution and the missing state follow the destination');

    // 3. A rapid sequence: nothing may pair one city's title with another city's guide.
    await evaluate(`(() => {
      window.snapshots = [];
      const read = () => ({
        title: document.querySelector('.destination-title')?.textContent.trim() || null,
        country: document.querySelector('.trip-form')?.elements.country?.value || null,
        cards: document.querySelectorAll('.guide-card').length,
        state: document.querySelector('.guide-card')?.dataset.guideState || null,
        heading: document.querySelector('.guide-title')?.textContent.trim() || null,
        first: document.querySelector('.guide-paragraph')?.textContent.trim().slice(0, 60) || null,
        attribution: document.querySelector('.guide-attribution')?.textContent.replace(/\\s+/g, ' ').trim() || null
      });
      window.snapshotObserver?.disconnect();
      window.snapshotObserver = new MutationObserver(() => window.snapshots.push(read()));
      window.snapshotObserver.observe(document.querySelector('#content-area'), { childList: true, subtree: true, attributes: true });
      const form = document.querySelector('#destination-search');
      for (const term of ['Paris', 'Porto, Portugal', 'Paris, Texas', 'Coimbra', 'Porto, Portugal']) {
        document.querySelector('#city_name').value = term; form.requestSubmit();
      }
    })()`);
    await new Promise(resolve => setTimeout(resolve, 1500));
    await until('!!document.querySelector(".guide-card")');
    const snapshots = await evaluate('window.snapshots');
    await evaluate('window.snapshotObserver.disconnect()');
    assert.ok(snapshots.length > 0, 'the rapid sequence must swap the page');
    for (const snap of snapshots.filter(item => item.title && item.cards)) {
      assert.equal(snap.cards, 1, 'one guide card at every moment');
      assert.equal(snap.heading, `About ${snap.title}`, `rapid sequence: ${snap.title}'s card is about ${snap.heading}`);
      const entry = guides[keyOf(snap)];
      if (entry?.reviewed) assert.match(snap.attribution || '', new RegExp(`revision ${entry.wikivoyage_revision}\\b`), `rapid sequence: ${snap.title}/${snap.country} shows its own revision`);
      else assert.equal(snap.state, 'missing', `rapid sequence: ${snap.title}/${snap.country} has no guide`);
    }
    const settled = await state(); assertCard(settled, width, 'rapid sequence settled'); assert.equal(settled.title, 'Porto');
    record.steps.rapid = { observations: snapshots.length, settled: `${settled.title}/${settled.country}` };
    check(width, `rapid sequence: ${snapshots.length} observations, every card matches its own destination; settled on ${settled.title}`);

    // 4. Shared trip pages and reload, both Parises.
    for (const [slug, country, kind] of [['paris-fr', 'fr', 'reviewed'], ['paris-us', 'us', 'missing']]) {
      await navigate(`${base}/trip/${slug}`);
      const shared = await state(); assert.equal(shared.country, country);
      assertCard(shared, width, `shared ${slug}`); assert.equal(shared.state, kind);
      record.steps[slug] = shared;
      await screenshotOf(`${width}-5-shared-${slug}`, '#overview .guide-card');
    }
    await call('Page.reload', { ignoreCache: true });
    await until("document.readyState === 'complete' && !!document.querySelector('.destination-title')");
    assertCard(await state(), width, 'shared reload');
    check(width, 'shared trip pages and reload: Paris, France has its guide, Paris, Texas the missing state');

    // 5. Planning replaces the planner card and the itinerary/stays sections only; Back/Forward keep the guide.
    await navigate(`${base}/?back-forward=${Date.now()}`);
    await evaluate(`localStorage.removeItem('htmx-history-cache'); window.restores = 0; document.body.addEventListener('htmx:historyRestore', () => window.restores++);`);
    await searchAndSettle('Paris', 'Paris');
    const beforePlan = await state();
    await evaluate("document.querySelector('.trip-form').requestSubmit()");
    await until("document.querySelectorAll('.itinerary-day').length === 3 && !document.querySelector('.trip-submit').disabled");
    const planned = await state();
    assertReviewed(planned, width, 'after planning');
    assert.deepEqual(planned.paragraphs, beforePlan.paragraphs, 'planning leaves the guide untouched');
    await evaluate('history.back()');
    await until('window.restores === 1');
    const back = await state(); assertReviewed(back, width, 'after Back'); assert.equal(back.title, 'Paris');
    assert.deepEqual(back.paragraphs, beforePlan.paragraphs);
    await evaluate('history.forward()');
    await until('window.restores === 2');
    const forward = await state(); assertReviewed(forward, width, 'after Forward');
    assert.equal(await evaluate("document.querySelectorAll('.itinerary-day').length"), 3, 'Forward restores the plan');
    check(width, 'plan → Back → Forward: one guide card, same text, planner sections intact');

    // 6. Awkward content: a token wider than the phone, a long article title, markup characters.
    await navigate(`${base}/?city_name=Longguide&awkward=${Date.now()}`);
    const awkward = await state();
    assert.equal(awkward.state, 'reviewed'); assertLayout(awkward, width, 'Longguide');
    assert.ok(awkward.paragraphs[0].includes('<b>markup characters</b>'), 'the text is shown literally, not as markup');
    assert.equal(await evaluate("document.querySelectorAll('.guide-body b, .guide-body script').length"), 0, 'no markup elements were created from guide text');
    record.steps.longguide = awkward;
    await screenshotOf(`${width}-6-awkward-content`, '#overview .guide-card');
    check(width, `awkward fixture content wraps: no overflow (${awkward.overflow}px page width in a ${width}px viewport), text stays literal`);

    results.push(record);
  }

  if (fallbackBase) {
    for (const [width, height] of [[375, 812], [1440, 1000]]) {
      await call('Emulation.setDeviceMetricsOverride', { width, height, deviceScaleFactor: 1, mobile: width <= 768 });
      await navigate(`${fallbackBase}/?fallback=${Date.now()}`);
      const none = await state();
      assertMissing(none, width, 'fallback scenario, Lisbon', true);
      await screenshotOf(`${width}-7-fallback-scenario`, '#overview .guide-card');
      results.push({ fallback: width, state: none.state });
      check(width, 'fallback scenario: even Lisbon shows the missing state');
    }
  }

  assert.deepEqual(errors, [], 'Unexpected JavaScript exceptions');
  await writeFile(new URL('browser-results.json', output), JSON.stringify({ browser, htmx: '1.9.11', base, fallbackBase, results, errors }, null, 2));
  console.log('PASS: city guide browser acceptance, no JavaScript exceptions');
  console.log(`Evidence: ${output.pathname}`);
} finally {
  await call('Fetch.disable').catch(() => {});
  ws.close();
}
