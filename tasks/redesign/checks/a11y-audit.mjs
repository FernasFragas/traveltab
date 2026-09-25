#!/usr/bin/env node
// Contrast and accessibility audit of the CURRENT UI, run in a real browser against the offline
// design preview. It measures instead of assuming: text and icon contrast is read from the
// pixels behind every line of text (so photo overlays, gradients and chips are covered), focus
// rings from before/after screenshots, and names/landmarks from the browser's accessibility tree.
//
//   CDP_PORT=9327 HTMX_PATH=/tmp/htmx.js [AXE_PATH=/path/to/axe.min.js] [OUTPUT_DIR=dir] node <this file>
//
// The script starts (and stops) its own preview processes on free localhost ports and attaches to
// an isolated Chromium/Brave that has its own debugging port and temporary profile. AXE_PATH is
// optional: axe-core is never a repo dependency; download it to a scratch directory to add it.
// Exit status is non-zero if any check fails. Evidence: <OUTPUT_DIR>/a11y-results.json + shots.
import assert from 'node:assert/strict';
import { readFile, mkdir, writeFile } from 'node:fs/promises';
import { connect, startPreview, decodePNG, luminance, ratio, composite, round, sleep, repoRoot } from './lib/browser.mjs';
import { collectText, collectControls, collectComponents, collectStructure, collectTargets } from './lib/collect.mjs';

const output = new URL(process.env.OUTPUT_DIR ? `file://${process.env.OUTPUT_DIR.replace(/\/?$/, '/')}` : '../../artifacts/redesign/final-verification-2/a11y/', import.meta.url);
await mkdir(output, { recursive: true });
const axeSource = process.env.AXE_PATH ? await readFile(process.env.AXE_PATH, 'utf8') : null;

// AUDIT_ONLY=contrast-quick,focus,targets runs a subset (used by the negative controls); default is everything.
const only = (process.env.AUDIT_ONLY || '').split(',').filter(Boolean);
const want = name => !only.length || only.includes(name);
const results = []; // the requirement matrix rows this script owns
const findings = []; // every measured pair that failed, with numbers
// `limitation` marks a failure that is deliberate and documented (a design trade-off); it is reported
// as its own status and does not fail the run. An unexpected pass of a documented limitation fails it.
const record = (id, ok, detail, limitation = null) => {
  const status = ok ? (limitation ? 'fail' : 'pass') : limitation ? 'limitation' : 'fail';
  results.push({ id, status, detail, ...(limitation ? { limitation } : {}) });
  console.log(`${status === 'pass' ? 'PASS' : status === 'limitation' ? 'KNOWN LIMITATION' : 'FAIL'} ${id}: ${detail}${limitation ? ` [${limitation}]` : ''}`);
};
const pass = (id, detail) => record(id, true, detail);
const shots = [];
const axResults = {};
const focusResults = {};

const preview = await startPreview({ scenario: 'default' });
const fallbackPreview = await startPreview({ scenario: 'fallback' });
const wideVideos = await startPreview({ scenario: 'videos-10' });
const wideStays = await startPreview({ scenario: 'stays-6' });
const uncertain = await startPreview({ scenario: 'forecast-uncertain' });
const b = await connect();
const { evaluate, until, call } = b;

// NEGATIVE_CONTROL=muted|border|focus|targets serves a deliberately broken copy of /styles.css, to prove
// that the audit notices each kind of regression. The run is expected to FAIL.
if (process.env.NEGATIVE_CONTROL) {
  const broken = {
    muted: css => css.replace('--tt-muted: #666158', '--tt-muted: #9a948a'),
    border: css => css.replace('--tt-border-control: #8F887B', '--tt-border-control: #DED8CC'),
    focus: css => `${css}\n:focus-visible, .itinerary-day-summary:focus-visible { outline: none !important; }\n.tt-search:focus-within { border-color: #DED8CC !important; box-shadow: none !important; }`,
    targets: css => `${css}\n.tt-footer-links a { width: 28px; height: 28px; }\n.tt-header-navigation a { min-height: 20px; }`
  }[process.env.NEGATIVE_CONTROL];
  assert.ok(broken, 'unknown NEGATIVE_CONTROL');
  const css = broken(await readFile(new URL('public/styles.css', `file://${repoRoot}`), 'utf8'));
  b.state.overrides.set('/styles.css', { responseCode: 200, body: Buffer.from(css).toString('base64'), responseHeaders: [{ name: 'Content-Type', value: 'text/css' }, { name: 'Cache-Control', value: 'no-store' }] });
  console.log(`NEGATIVE CONTROL ${process.env.NEGATIVE_CONTROL}: serving a broken /styles.css; failures below are expected`);
}

const WIDTHS = [[375, 812], [1440, 1000]];
const dedupeBy = list => { const seen = new Map(); for (const p of list) { const key = `${p.kind}|${p.el}|${p.fg}|${p.required}`; if (!seen.has(key) || seen.get(key).ratio > p.ratio) seen.set(key, p); } return [...seen.values()]; };
const px = (img, x, y) => { const i = (y * img.width + x) * 4; return [img.data[i], img.data[i + 1], img.data[i + 2]]; };

const hideText = `(() => { const s = document.createElement('style'); s.id = 'audit-hide';
  s.textContent = '*, *::before, *::after, ::placeholder { color: transparent !important; -webkit-text-fill-color: transparent !important; text-shadow: none !important; text-decoration-color: transparent !important; caret-color: transparent !important; } .bi::before { color: transparent !important; }';
  document.head.appendChild(s); })()`;
const showText = "document.getElementById('audit-hide')?.remove()";

// --- Text and icon contrast from the real pixels behind each line -------------------------------
// Returns every pair measured; `worst` is the lowest ratio against the lightest and darkest
// background pixel under the text, so a busy photo cannot hide a failure.
const contrastPass = async (label, width, height, { settleImages = true, keepFocus = false } = {}) => {
  // A programmatic focus ring (the itinerary heading after planning) is measured by the focus audit, not here.
  if (!keepFocus) await evaluate('document.activeElement && document.activeElement.blur && document.activeElement.blur()');
  const docHeight = await evaluate('Math.ceil(document.documentElement.scrollHeight)');
  await b.viewport(width, Math.min(docHeight, 15000));
  await sleep(150);
  if (settleImages) await until("[...document.images].every(i => i.complete)", 10000).catch(() => {});
  await evaluate('document.fonts.ready.then(() => true)');
  const texts = await evaluate(collectText);
  await evaluate(hideText); await sleep(120);
  const image = decodePNG(await b.shot());
  await evaluate(showText);
  await b.viewport(width, height);
  const pairs = [];
  for (const t of texts) {
    if (t.covered) continue;
    let lightest = null, darkest = null;
    // An icon's element box is taller than its glyph; sample the glyph's own centred square.
    const rects = t.kind === 'icon' ? t.rects.map(r => { const side = Math.min(t.size * 0.7, r.w, r.h); return { x: r.x + (r.w - side) / 2, y: r.y + (r.h - side) / 2, w: side, h: side }; }) : t.rects;
    for (const r of rects) {
      const [x0, x1] = [Math.max(0, Math.ceil(r.x + 1)), Math.min(image.width, Math.floor(r.x + r.w - 1))];
      const [y0, y1] = [Math.max(0, Math.ceil(r.y + 1)), Math.min(image.height, Math.floor(r.y + r.h - 1))];
      for (let y = y0; y < y1; y++) for (let x = x0; x < x1; x++) {
        const p = px(image, x, y); const l = luminance(p);
        if (!lightest || l > lightest.l) lightest = { l, p };
        if (!darkest || l < darkest.l) darkest = { l, p };
      }
    }
    if (!lightest) continue; // off-screen or clipped
    const against = bg => ratio(composite(t.rgb, t.alpha, bg), bg);
    const [a, z] = [against(lightest.p), against(darkest.p)];
    const worst = Math.min(a, z);
    const required = t.kind === 'icon' ? 3 : t.large ? 3 : 4.5;
    pairs.push({ label, width, kind: t.kind, el: t.el, text: t.text, size: t.size, weight: t.weight, required, ratio: round(worst),
      fg: t.rgb.map(Math.round), alpha: round(t.alpha), bgLight: lightest.p, bgDark: darkest.p, disabled: t.disabled, iconOnlyControl: !!t.iconOnlyControl });
  }
  return pairs;
};

// --- Form-control text and component boundaries (computed colours) -----------------------------
const controlPass = async (label, width) => {
  const out = [];
  for (const c of await evaluate(collectControls)) {
    out.push({ label, width, kind: 'control-' + c.kind, el: c.el, required: 4.5, ratio: round(ratio(c.rgb, c.bg)), fg: c.rgb, bgLight: c.bg, disabled: c.disabled });
  }
  return out;
};
// A component is findable if either its border or its fill stands out from what is around it. Text
// fields must pass on their own; a button whose text label already identifies it is advisory.
const FIELD_COMPONENTS = new Set(['.trip-input', '.trip-select', '.tt-search']);
const componentPass = async (label, width) => {
  const out = [];
  for (const c of await evaluate(collectComponents)) {
    const border = c.border ? ratio(c.border, c.parent) : 0, fill = c.fill ? ratio(c.fill, c.parent) : 0;
    if (!c.border && !c.fill) continue; // identified by its icon or label alone; those are measured as icons/text
    out.push({ label, width, kind: FIELD_COMPONENTS.has(c.selector) ? 'component-field' : 'component-labelled', el: c.selector, required: 3, ratio: round(Math.max(border, fill)),
      border: round(border), fill: round(fill), fg: (c.border || c.fill || []).map(Math.round), bgLight: c.parent.map(Math.round) });
  }
  return out;
};

const allPairs = [];
const audit = async (label, width, height, { components = false } = {}) => {
  const pairs = [...await contrastPass(label, width, height), ...await controlPass(label, width), ...(components ? await componentPass(label, width) : [])];
  allPairs.push(...pairs);
  return pairs;
};

// --- Accessibility tree: names, roles, landmarks -----------------------------------------------
const axTree = async () => {
  await call('Accessibility.enable');
  const { nodes } = await call('Accessibility.getFullAXTree');
  return nodes.filter(n => !n.ignored).map(n => ({ id: n.nodeId, role: n.role?.value, name: (n.name?.value || '').trim(), backend: n.backendDOMNodeId,
    props: Object.fromEntries((n.properties || []).map(p => [p.name, p.value?.value])) }));
};
const describeBackend = async backend => {
  try {
    const { object } = await call('DOM.resolveNode', { backendNodeId: backend });
    const r = await call('Runtime.callFunctionOn', { objectId: object.objectId, returnByValue: true, functionDeclaration: 'function () { return this.outerHTML ? this.outerHTML.slice(0, 120) : String(this.nodeName); }' });
    return r.result.value;
  } catch { return '(unresolved)'; }
};
const NAMED_ROLES = new Set(['link', 'button', 'textbox', 'searchbox', 'combobox', 'listbox', 'spinbutton', 'checkbox', 'radio', 'switch', 'slider', 'img', 'image', 'Iframe', 'iframe', 'DisclosureTriangle', 'menuitem', 'tab', 'date', 'input', 'DateTime', 'Date']);

// --- Focus order and visible focus --------------------------------------------------------------
const waitScrollSettled = async () => {
  let last = -1;
  for (let i = 0; i < 40; i++) { const y = await evaluate('scrollY'); if (y === last) return; last = y; await sleep(60); }
};
const crop = (img, r, pad) => {
  const x0 = Math.max(0, Math.floor(r.x - pad)), y0 = Math.max(0, Math.floor(r.y - pad));
  const x1 = Math.min(img.width, Math.ceil(r.x + r.w + pad)), y1 = Math.min(img.height, Math.ceil(r.y + r.h + pad));
  return { x0, y0, x1, y1 };
};
const tomorrow = new Date(Date.now() + 86400000).toISOString().slice(0, 10);
const focusAudit = async (label, width, height, url) => {
  await b.navigate(url);
  await b.viewport(width, height);
  await evaluate("document.activeElement?.blur(); window.scrollTo({ top: 0, behavior: 'instant' })");
  const stops = [];
  const seen = new Set();
  let repeats = 0;
  for (let i = 0; i < 250; i++) {
    await b.tab();
    await waitScrollSettled();
    const info = await evaluate(`(() => {
      const el = document.activeElement; if (!el || el === document.body || el === document.documentElement) return null;
      window.__focused = el; el.__tid = el.__tid || (window.__tid = (window.__tid || 0) + 1);
      const r = el.getBoundingClientRect(); const cs = getComputedStyle(el);
      const sample = [[r.left + Math.min(r.width / 2, 24), r.top + Math.min(r.height / 2, 12)], [r.left + r.width / 2, r.top + r.height / 2]]
        .map(([x, y]) => [Math.min(Math.max(x, 1), innerWidth - 1), Math.min(Math.max(y, 1), innerHeight - 1)]);
      const hits = sample.map(([x, y]) => document.elementFromPoint(x, y));
      const obscured = !hits.some(h => h && (el === h || el.contains(h) || h.contains(el)));
      return { tid: el.__tid, el: el.tagName.toLowerCase() + (el.id ? '#' + el.id : '') + (el.classList.length ? '.' + [...el.classList].slice(0, 2).join('.') : ''),
        text: (el.getAttribute('aria-label') || el.textContent || el.value || '').replace(/\\s+/g, ' ').trim().slice(0, 40),
        rect: { x: r.left, y: r.top, w: r.width, h: r.height }, page: { x: r.left + scrollX, y: r.top + scrollY }, outline: cs.outlineStyle + ' ' + cs.outlineWidth + ' ' + cs.outlineColor, boxShadow: cs.boxShadow === 'none' ? null : cs.boxShadow,
        inViewport: r.bottom > 0 && r.top < innerHeight && r.right > 0 && r.left < innerWidth, obscured, tabindex: el.getAttribute('tabindex'), focusVisible: el.matches(':focus-visible'), focus: el.matches(':focus'),
        withinCard: !!el.closest('.map-card:focus-within') };
    })()`);
    if (!info) break;
    if (seen.has(info.tid)) { if (stops[stops.length - 1].tid === info.tid && ++repeats < 8) { stops[stops.length - 1].segments = (stops[stops.length - 1].segments || 1) + 1; continue; } break; }
    seen.add(info.tid); repeats = 0;
    // Before/after screenshots of the element's surroundings: what actually changes on focus?
    const focused = decodePNG(await b.shot());
    await evaluate('window.__focused.blur()'); await sleep(60);
    const blurred = decodePNG(await b.shot());
    await evaluate('window.__focused.focus({ preventScroll: true })');
    const box = crop(focused, info.rect, 14);
    let changed = 0, maxChange = 1;
    for (let y = box.y0; y < box.y1; y++) for (let x = box.x0; x < box.x1; x++) {
      const [f, u] = [px(focused, x, y), px(blurred, x, y)];
      if (Math.abs(f[0] - u[0]) + Math.abs(f[1] - u[1]) + Math.abs(f[2] - u[2]) > 12) { changed++; maxChange = Math.max(maxChange, ratio(f, u)); }
    }
    stops.push({ ...info, changedPixels: changed, indicatorContrast: round(maxChange) });
    if (stops.length === 1) {
      shots.push(`${label}-focus-skip-link.png`);
      await writeFile(new URL(`${label}-focus-skip-link.png`, output), await b.shot());
    }
  }
  return stops;
};

// ====================================================================================================
try {
  console.log(`Environment: ${b.product}; axe-core ${axeSource ? axeSource.match(/axe v([\d.]+)/)?.[1] : 'not used'}`);

  // ---- 1. Contrast of every state, both widths ---------------------------------------------------
  const planDays = async (days = 3) => {
    await evaluate(`(() => { const f = document.querySelector('.trip-form'); f.elements.days.value = '${days}'; f.requestSubmit(); })()`);
    await until(`document.querySelectorAll('.itinerary-day').length === ${days} && !document.querySelector('.trip-submit').disabled`);
    await evaluate("document.querySelectorAll('.itinerary-day').forEach(d => { d.open = true; })");
    await b.settle();
    await sleep(400); await waitScrollSettled(); // planning focuses the itinerary heading and scrolls to it after a tick
  };
  const search = async (term, title) => {
    await evaluate("window.__prev = document.querySelector('.destination-banner')");
    await evaluate(`document.querySelector('#city_name').value = ${JSON.stringify(term)}; document.querySelector('#destination-search').requestSubmit()`);
    await until(`document.querySelector('.destination-title')?.textContent.trim() === ${JSON.stringify(title)} && document.querySelector('.destination-banner') !== window.__prev`);
    await b.settle();
  };
  const heroState = () => evaluate("document.querySelector('.destination-photo')?.dataset.photoState");

  const structureByState = {};
  if (want('contrast') || want('contrast-quick')) for (const [width, height] of WIDTHS) {
    b.state.imageMode = 'synthetic';
    await b.viewport(width, height);
    const states = [
      ['lisbon-initial', async () => { await b.navigate(`${preview.url}/?a=${Date.now()}`); }],
      ['lisbon-planned-5d-open', async () => { await planDays(5); }],
      ['porto-photo', async () => { await b.navigate(`${preview.url}/?city_name=Porto,%20Portugal&a=${Date.now()}`); assert.equal(await heroState(), 'ready'); }],
      ['porto-planned', async () => { await planDays(3); }],
      ['coimbra-no-photo-no-guide', async () => { await b.navigate(`${preview.url}/?city_name=Coimbra,%20Portugal&a=${Date.now()}`); assert.equal(await heroState(), 'unavailable'); }],
      ['brokenimage-failed', async () => { await b.navigate(`${preview.url}/?city_name=Brokenimage&a=${Date.now()}`); await until("document.querySelector('.destination-photo')?.dataset.photoState === 'failed'"); }],
      ['photoerror', async () => { await b.navigate(`${preview.url}/?city_name=Photoerror&a=${Date.now()}`); assert.equal(await heroState(), 'unavailable'); }],
      ['paris-texas', async () => { await b.navigate(`${preview.url}/trip/paris-us`); }],
      ['longguide', async () => { await b.navigate(`${preview.url}/?city_name=Longguide&a=${Date.now()}`); }],
      ['plan-error-500', async () => { await b.navigate(`${preview.url}/?a=${Date.now()}`); await evaluate("document.querySelector('.trip-form').elements.days.value = '4'; document.querySelector('.trip-form').requestSubmit()"); await until("!!document.querySelector('.trip-error')"); }],
      ['plan-error-400', async () => { await b.navigate(`${preview.url}/?a=${Date.now()}`); await evaluate("document.querySelector('.trip-form').elements.lat.value = 'invalid'; document.querySelector('.trip-form').requestSubmit()"); await until("!!document.querySelector('.trip-error')"); }],
      ['search-error', async () => { await b.navigate(`${preview.url}/?a=${Date.now()}`); await evaluate("document.querySelector('#city_name').value = 'error'; document.querySelector('#destination-search').requestSubmit()"); await until("document.querySelector('#destination-search-error')?.textContent.trim().length > 0"); }],
      ['submit-disabled', async () => { await b.navigate(`${preview.url}/?a=${Date.now()}`); await evaluate("document.querySelector('.trip-submit').disabled = true"); }],
      ['fallback-planned', async () => { await b.navigate(`${fallbackPreview.url}/?a=${Date.now()}`); await planDays(5); }],
      ['forecast-uncertain-planned', async () => { await b.navigate(`${uncertain.url}/?a=${Date.now()}`); await planDays(3); }],
      ['stays-6-planned', async () => { await b.navigate(`${wideStays.url}/?a=${Date.now()}`); await planDays(3); }],
      ['videos-10-expanded', async () => { await b.navigate(`${wideVideos.url}/?a=${Date.now()}`); await evaluate("document.querySelector('[data-video-expand]').click()"); }],
      ['video-activated', async () => { await b.navigate(`${preview.url}/?a=${Date.now()}`); await evaluate("document.querySelector('[data-video-activate]').click()"); await until("!!document.querySelector('.video-embed')"); }],
    ];
    if (width === 375) states.push(['mobile-menu-open', async () => { await b.navigate(`${preview.url}/?a=${Date.now()}`); await evaluate("document.querySelector('.tt-menu-button').click()"); }]);
    // Last: every third-party image request fails, so each image-error fallback (stop, thumbnail) shows.
    states.push(['images-blocked-planned', async () => { b.state.imageMode = 'block'; await b.navigate(`${preview.url}/?a=${Date.now()}`); await planDays(3); }]);
    for (const [name, setup] of states.filter(([n]) => want('contrast') || ['lisbon-initial', 'lisbon-planned-5d-open'].includes(n))) {
      await setup();
      const label = `${name}@${width}`;
      const pairs = await audit(label, width, height, { components: ['lisbon-planned-5d-open', 'lisbon-initial', 'plan-error-500', 'submit-disabled'].includes(name) });
      const bad = pairs.filter(p => !p.disabled && !(p.kind === 'icon' && !p.iconOnlyControl) && p.ratio < p.required);
      structureByState[label] = await evaluate(collectStructure);
      console.log(`  ${label}: ${pairs.length} pairs, ${bad.length} below threshold, min ${Math.min(...pairs.map(p => p.ratio / p.required * 4.5)).toFixed(2)} (normalised to 4.5)`);
      if (name === 'lisbon-planned-5d-open' || name === 'coimbra-no-photo-no-guide' || name === 'brokenimage-failed' || name === 'plan-error-500') {
        await writeFile(new URL(`${label}.png`, output), await b.shot()); shots.push(`${label}.png`);
      }
    }
  }

  b.state.imageMode = 'synthetic';
  // Worst case behind light overlay text: a pure-white photograph behind the hero credit,
  // caption, mobile title, stop-photo credits and video play badges.
  b.state.imageMode = 'white';
  if (want('contrast')) for (const [width, height] of WIDTHS) {
    await b.viewport(width, height);
    await b.navigate(`${preview.url}/?city_name=Porto,%20Portugal&white=${Date.now()}`);
    // Local fixture photos are served by the map origin; blank the hero itself as well.
    await evaluate(`(async () => { const img = document.querySelector('.destination-image'); const c = document.createElement('canvas'); c.width = 64; c.height = 36; const x = c.getContext('2d'); x.fillStyle = '#fff'; x.fillRect(0, 0, 64, 36); img.src = c.toDataURL(); await new Promise(r => img.decode().then(r, r)); })()`);
    await evaluate("(() => { const f = document.querySelector('.trip-form'); f.requestSubmit(); })()");
    await until("document.querySelectorAll('.itinerary-day').length === 3 && !document.querySelector('.trip-submit').disabled");
    await evaluate("document.querySelectorAll('.itinerary-day').forEach(d => { d.open = true; })");
    await b.settle();
    const label = `worst-case-white-photos@${width}`;
    const pairs = await audit(label, width, height);
    const overlays = pairs.filter(p => /credit|caption|destination-title|play-badge/.test(p.el) || /play-fill/.test(p.el));
    console.log(`  ${label}: ${pairs.length} pairs; overlay pairs ${overlays.length}, min overlay ratio ${Math.min(...overlays.map(p => p.ratio))}`);
    await writeFile(new URL(`${label}.png`, output), await b.shot()); shots.push(`${label}.png`);
  }
  b.state.imageMode = 'synthetic';

  // Every visible icon is held to 3:1, decorative or not. A bordered button whose text label already
  // identifies it ('component-labelled') is reported as advisory, not a failure.
  const failing = allPairs.filter(p => !p.disabled && p.kind !== 'component-labelled' && p.ratio < p.required);
  const advisory = dedupeBy(allPairs.filter(p => p.kind === 'component-labelled' && p.ratio < p.required));
  const dedupe = dedupeBy;
  for (const p of dedupe(failing)) findings.push({ ...p, area: 'contrast' });
  const textPairs = allPairs.filter(p => p.kind === 'text' || p.kind === 'control-value' || p.kind === 'control-placeholder');
  const lowest = [...allPairs].sort((p, q) => p.ratio / p.required - q.ratio / q.required).slice(0, 12);
  if (allPairs.length) record('contrast.text', dedupe(failing.filter(p => p.kind === 'text' || p.kind.startsWith('control'))).length === 0,
    `${textPairs.length} text/control-value pairs measured across ${new Set(allPairs.map(p => p.label)).size} states; ${dedupe(failing.filter(p => p.kind === 'text' || p.kind.startsWith('control'))).length} distinct failing (see findings)`);
  if (allPairs.length) record('contrast.ui-components-and-icons', dedupe(failing.filter(p => p.kind === 'icon' || p.kind.startsWith('component'))).length === 0,
    `${allPairs.filter(p => p.kind === 'icon' || p.kind.startsWith('component')).length} icon/component pairs; ${dedupe(failing.filter(p => p.kind === 'icon' || p.kind.startsWith('component'))).length} distinct failing (see findings)`);
  if (allPairs.length) {
    results[results.length - 1].advisory = advisory.map(p => `${p.el} ${p.ratio}:1 (border ${p.border}, fill ${p.fill})`);
    console.log('Advisory (labelled buttons with low-contrast borders):', results[results.length - 1].advisory.join('; '));
  }
  console.log('Lowest margin pairs:'); for (const p of lowest) console.log(`  ${p.ratio}:1 (need ${p.required}) ${p.label} ${p.kind} ${p.el} "${p.text || ''}"`);

  // ---- 2. Structure: landmarks, headings, names, alt text, forms, iframes ---------------------------
  const structFailures = [];
  const seenHeadings = new Set();
  for (const [label, s] of Object.entries(structureByState)) {
    if (s.lang !== 'en') structFailures.push(`${label}: html lang=${s.lang}`);
    if (!s.title) structFailures.push(`${label}: no title`);
    if (/maximum-scale|user-scalable=no/.test(s.viewportMeta || '')) structFailures.push(`${label}: viewport blocks zoom`);
    if (s.h1Count !== 1) structFailures.push(`${label}: ${s.h1Count} h1`);
    if (s.headings[0]?.level !== 1) structFailures.push(`${label}: first heading is h${s.headings[0]?.level}`);
    s.headings.forEach((h, i) => { if (i && h.level > s.headings[i - 1].level + 1) structFailures.push(`${label}: heading jumps h${s.headings[i - 1].level} -> h${h.level} "${h.text}"`); });
    if (s.dupIds.length) structFailures.push(`${label}: duplicate ids ${s.dupIds}`);
    if (s.refs.length) structFailures.push(`${label}: dangling ARIA refs ${s.refs}`);
    if (s.positiveTabindex.length) structFailures.push(`${label}: positive tabindex`);
    for (const i of s.images) if (i.alt === null) structFailures.push(`${label}: img without alt attribute ${i.src}`); else if (!i.hidden && !i.alt && !i.inButton) structFailures.push(`${label}: visible img with empty alt outside a button ${i.src}`);
    for (const f of s.iframes) if (!f.title) structFailures.push(`${label}: iframe without title ${f.src}`);
    for (const c of s.controls) if (!c.labelled) structFailures.push(`${label}: control without a label ${c.id || c.name}`);
    for (const a of s.external) if (!/noopener/.test(a.rel)) structFailures.push(`${label}: target=_blank without noopener ${a.href}`);
    for (const h of s.headings) seenHeadings.add(`h${h.level} ${h.text}`);
  }
  if (Object.keys(structureByState).length && want('contrast')) record('structure.headings-lang-ids-refs', structFailures.length === 0, structFailures.length ? structFailures.slice(0, 8).join(' | ') : `${Object.keys(structureByState).length} states: one h1, no skipped levels, lang=en, no duplicate ids, no dangling ARIA references, no positive tabindex`);
  structFailures.forEach(f => findings.push({ area: 'structure', detail: f }));

  // Accessibility tree at both widths on the planned page: landmarks and accessible names.
  if (want('structure')) for (const [width, height] of WIDTHS) {
    await b.viewport(width, height);
    await b.navigate(`${preview.url}/?a=${Date.now()}`);
    await planDays(3);
    await evaluate("document.querySelector('[data-video-expand]')?.click()");
    const tree = await axTree();
    const landmarks = tree.filter(n => ['banner', 'navigation', 'main', 'contentinfo', 'search', 'region', 'form', 'complementary'].includes(n.role)).map(n => `${n.role}:${n.name || '(unnamed)'}`);
    const unnamed = [];
    for (const n of tree.filter(n => NAMED_ROLES.has(n.role) && !n.name && n.backend)) unnamed.push({ role: n.role, html: await describeBackend(n.backend) });
    const nameless = unnamed.filter(u => !/type="date"/.test(u.html) || true);
    axResults[width] = { landmarks, unnamed: nameless, nodes: tree.length,
      names: Object.fromEntries(tree.filter(n => ['textbox', 'searchbox', 'combobox', 'button', 'DisclosureTriangle', 'Iframe', 'iframe'].includes(n.role)).slice(0, 60).map((n, i) => [`${i}:${n.role}`, n.name])) };
    const nav = landmarks.filter(l => l.startsWith('navigation:'));
    const problems = [];
    if (landmarks.filter(l => l.startsWith('banner:')).length !== 1) problems.push('banner count');
    if (landmarks.filter(l => l.startsWith('main:')).length !== 1) problems.push('main count');
    if (landmarks.filter(l => l.startsWith('contentinfo:')).length !== 1) problems.push('contentinfo count');
    if (new Set(nav).size !== nav.length || nav.some(l => l.endsWith('(unnamed)'))) problems.push(`navigation landmarks not uniquely named: ${nav}`);
    if (!landmarks.some(l => l.startsWith('search:'))) problems.push('no search landmark');
    if (nameless.length) problems.push(`${nameless.length} controls without an accessible name: ${nameless.map(u => u.role + ' ' + u.html.slice(0, 70)).join('; ')}`);
    record(`landmarks-and-names@${width}`, problems.length === 0, problems.length ? problems.join(' | ') : `${landmarks.length} landmarks (${landmarks.join(', ')}); every link, button, field, image and frame has a name`);
    if (problems.length) findings.push({ area: 'landmarks', width, detail: problems.join(' | ') });
    // Names of the form controls, as the browser computes them.
    const fieldNames = tree.filter(n => ['textbox', 'searchbox', 'combobox', 'date', 'Date', 'input'].includes(n.role)).map(n => `${n.role}:${n.name}`);
    axResults[width].fields = fieldNames;
    const endOutput = tree.find(n => n.role === 'status' && /,/.test(n.name || ''));
    const output_ = await evaluate("(() => { const o = document.querySelector('#trip-end-value'); return { text: o.textContent.trim(), name: o.getAttribute('aria-label') || o.getAttribute('aria-labelledby') }; })()");
    axResults[width].endOutput = output_;
    if (axeSource) {
      await evaluate(axeSource);
      const axe = await evaluate(`axe.run(document, { resultTypes: ['violations', 'incomplete'] }).then(r => ({ version: axe.version, violations: r.violations.map(v => ({ id: v.id, impact: v.impact, help: v.help, nodes: v.nodes.map(n => n.target.join(' ') + ' :: ' + n.html.slice(0, 90)) })), incomplete: r.incomplete.map(v => ({ id: v.id, nodes: v.nodes.length })), passes: r.passes.length }))`);
      axResults[width].axe = axe;
      record(`axe-core@${width}`, axe.violations.length === 0, axe.violations.length ? axe.violations.map(v => `${v.id}(${v.nodes.length})`).join(', ') : `axe-core ${axe.version}: 0 violations, ${axe.passes} passes, ${axe.incomplete.length} incomplete (${axe.incomplete.map(i => i.id).join(', ')})`);
      axe.violations.forEach(v => findings.push({ area: 'axe', width, id: v.id, impact: v.impact, nodes: v.nodes.slice(0, 5) }));
    }
  }

  // ---- 3. Focus order, visible focus, skip link --------------------------------------------------
  if (want('focus')) for (const [width, height] of WIDTHS) {
    const stops = await focusAudit(`focus-${width}`, width, height, `${preview.url}/trip/lisbon-pt?days=3&from=${tomorrow}`);
    focusResults[width] = stops;
    const problems = [];
    if (!/tt-skip-link/.test(stops[0]?.el || '')) problems.push(`first stop is ${stops[0]?.el}, not the skip link`);
    if (stops[0] && (stops[0].rect.y < 0 || stops[0].rect.x < 0 || !stops[0].inViewport)) problems.push('skip link is not on screen when focused');
    for (const s of stops) {
      if (!s.inViewport) problems.push(`${s.el} "${s.text}" focused off screen`);
      if (s.obscured) problems.push(`${s.el} "${s.text}" is covered by other content when focused`);
      if (s.changedPixels < 20) problems.push(`${s.el} "${s.text}" shows no visible change on focus (${s.changedPixels} px)`);
      else if (s.indicatorContrast < 3) problems.push(`${s.el} "${s.text}" focus indicator contrast ${s.indicatorContrast}:1`);
      if (s.tabindex && Number(s.tabindex) > 0) problems.push(`${s.el} positive tabindex`);
    }
    // Visual order: after the skip link, a stop that sits well above the previous one AND does not start in a
    // column to its right is out of visual order (a two-column layout reads left column, then right).
    const order = [];
    for (let i = 2; i < stops.length; i++) {
      const [p, c] = [{ ...stops[i - 1].rect, ...stops[i - 1].page }, { ...stops[i].rect, ...stops[i].page }];
      const inLaterColumn = c.x > p.x + p.w / 2;
      if (c.y < p.y - 120 && !inLaterColumn && !/tt-section-navigation/.test(stops[i].el) && stops[i].inViewport) order.push(`${stops[i - 1].el} "${stops[i - 1].text}" -> ${stops[i].el} "${stops[i].text}"`);
    }
    focusResults[width + 'order'] = order;
    const orderLimitation = width === 375 ? 'Below 900px the planner is shown before the map through CSS order (the design puts the form first on phones), while the DOM keeps the map first for the two-column desktop reading order. Reordering the DOM would only move the mismatch to desktop.' : null;
    record(`focus-order-matches-visual@${width}`, order.length === 0, order.length ? `${order.length} jumps against the visual order: ${order.join(' | ')}` : 'no tab stop moves backwards up the page without moving to a later column', orderLimitation);
    if (order.length) findings.push({ area: 'focus-order', width, detail: order.join(' | ') });
    record(`focus-visible@${width}`, problems.length === 0, problems.length ? `${problems.length} problems: ${problems.slice(0, 6).join(' | ')}` : `${stops.length} tab stops (${stops.reduce((n, s) => n + (s.segments || 0), 0)} extra date-field segments): skip link first, none covered, every one shows a focus change of at least 3:1 (weakest ${Math.min(...stops.map(s => s.indicatorContrast))}:1)`);
    problems.forEach(p => findings.push({ area: 'focus', width, detail: p }));
  }

  // Skip link: Enter moves focus to the overview and scrolls it into view, contrast when focused.
  if (want('focus')) {
    await b.viewport(375, 812);
    await b.navigate(`${preview.url}/?skip=${Date.now()}`);
    await evaluate("document.activeElement.blur()");
    await b.tab();
    await sleep(120);
    const focusedSkip = await evaluate("(() => { const el = document.activeElement; const r = el.getBoundingClientRect(); return { skip: el.classList.contains('tt-skip-link'), top: r.top, bottom: r.bottom, left: r.left, right: r.right }; })()");
    const pairs = await contrastPass('skip-link-focused@375', 375, 812, { keepFocus: true });
    const skipPair = pairs.find(p => /tt-skip-link/.test(p.el));
    await b.key('Enter', 'Enter', 13, 0, '\r'); await sleep(400);
    const after = await evaluate("({ active: document.activeElement.id, top: document.getElementById('overview').getBoundingClientRect().top })");
    record('skip-link', focusedSkip.skip && focusedSkip.top >= 0 && skipPair.ratio >= 4.5 && after.active === 'overview' && after.top < 200,
      `first Tab lands on it at ${Math.round(focusedSkip.top)}..${Math.round(focusedSkip.bottom)}px from the top, text contrast ${skipPair?.ratio}:1; Enter focuses #${after.active} with its top at ${Math.round(after.top)}px`);
    await b.viewport(375, 812);
  }

  // ---- 4. Accordion keyboard, live regions, error alerts ----------------------------------------------
  if (want('behaviour')) {
    await b.viewport(1440, 1000);
    await b.navigate(`${preview.url}/?acc=${Date.now()}`);
    await planDays(3);
    await evaluate("document.querySelectorAll('.itinerary-day').forEach((d, i) => { d.open = i === 0; })");
    const kb = [];
    await evaluate("document.querySelectorAll('.itinerary-day-summary')[1].focus()");
    await b.key('Enter', 'Enter', 13, 0, '\r'); await sleep(100); kb.push(await evaluate("document.querySelectorAll('.itinerary-day')[1].open"));
    await b.key('Enter', 'Enter', 13, 0, '\r'); await sleep(100); kb.push(await evaluate("document.querySelectorAll('.itinerary-day')[1].open"));
    await b.key(' ', 'Space', 32, 0, ' '); await sleep(100); kb.push(await evaluate("document.querySelectorAll('.itinerary-day')[1].open"));
    await b.key(' ', 'Space', 32, 0, ' '); await sleep(100); kb.push(await evaluate("document.querySelectorAll('.itinerary-day')[1].open"));
    const tree = await axTree();
    const summaries = tree.filter(n => n.role === 'DisclosureTriangle' || n.role === 'disclosure-triangle');
    const expandedProps = summaries.map(s => s.props.expanded);
    const independent = await evaluate("[...document.querySelectorAll('.itinerary-day')].map(d => d.open)");
    record('accordion-keyboard', JSON.stringify(kb) === '[true,false,true,false]' && summaries.length === 3 && expandedProps.every(v => typeof v === 'boolean') && JSON.stringify(independent) === '[true,false,false]',
      `Enter opens/closes, Space opens/closes (${JSON.stringify(kb)}); day 1 open by default, days independent (${JSON.stringify(independent)}); ${summaries.length} summaries exposed with expanded=${JSON.stringify(expandedProps)}; names ${JSON.stringify(summaries.map(s => s.name.slice(0, 30)))}`);
    // Video expansion button (a disclosure button) by keyboard.
  }
  if (want('behaviour')) {
    await b.viewport(1440, 1000);
    await b.navigate(`${wideVideos.url}/?vid=${Date.now()}`);
    await evaluate("document.querySelector('[data-video-expand]').focus()");
    await b.key('Enter', 'Enter', 13, 0, '\r'); await sleep(100);
    const opened = await evaluate("({ expanded: document.querySelector('[data-video-expand]').getAttribute('aria-expanded'), hidden: document.querySelector('#video-extra').hidden, label: document.querySelector('[data-video-expand-label]').textContent })");
    await b.key(' ', 'Space', 32, 0, ' '); await sleep(100);
    const closed = await evaluate("({ expanded: document.querySelector('[data-video-expand]').getAttribute('aria-expanded'), hidden: document.querySelector('#video-extra').hidden, label: document.querySelector('[data-video-expand-label]').textContent })");
    record('video-expand-keyboard', opened.expanded === 'true' && !opened.hidden && closed.expanded === 'false' && closed.hidden && opened.label === 'Show fewer videos',
      `Enter expands (aria-expanded=${opened.expanded}, "${opened.label}"), Space collapses (aria-expanded=${closed.expanded}, "${closed.label}")`);
  }
  if (want('behaviour')) {
    // Live regions: watch every status/alert region while a search, a plan and two failures run.
    await b.viewport(1440, 1000);
    await b.navigate(`${preview.url}/?live=${Date.now()}`);
    await evaluate(`(() => {
      window.__live = [];
      const watch = el => { new MutationObserver(() => window.__live.push([el.id || el.className, el.textContent.trim()])).observe(el, { childList: true, characterData: true, subtree: true }); };
      ['destination-search-error', 'destination-search-status', 'trip-status'].forEach(id => watch(document.getElementById(id)));
      new MutationObserver(records => { for (const r of records) for (const n of r.addedNodes) if (n.nodeType === 1) { const alert = n.matches?.('[role=alert]') ? n : n.querySelector?.('[role=alert]'); if (alert) window.__live.push(['inserted-alert', alert.textContent.trim()]); } }).observe(document.body, { childList: true, subtree: true });
    })()`);
    await evaluate("document.querySelector('#city_name').value = 'Porto'; document.querySelector('#destination-search').requestSubmit()");
    await until("document.querySelector('.destination-title')?.textContent.trim() === 'Porto'"); await sleep(200);
    await evaluate("document.querySelector('.trip-form').requestSubmit()");
    await until("document.querySelectorAll('.itinerary-day').length === 3 && !document.querySelector('.trip-submit').disabled"); await sleep(300);
    await evaluate("document.querySelector('.trip-form').elements.days.value = '4'; document.querySelector('.trip-form').requestSubmit()");
    await until("!!document.querySelector('.trip-error')"); await sleep(200);
    await evaluate("document.querySelector('#city_name').value = 'error'; document.querySelector('#destination-search').requestSubmit()");
    await until("document.querySelector('#destination-search-error')?.textContent.trim().length > 0"); await sleep(200);
    const live = await evaluate('window.__live');
    const texts = live.map(l => l[1]);
    const expectations = ['Destination updated. Your overview is ready.', 'Planning your trip…', 'Your itinerary has been updated.', 'fixture', 'Planning did not complete'];
    const missing = [];
    if (!texts.includes('Destination updated. Your overview is ready.')) missing.push('search success announcement');
    if (!texts.includes('Planning your trip…')) missing.push('planning-started announcement');
    if (!texts.includes('Your itinerary has been updated.')) missing.push('plan-complete announcement');
    if (!live.some(l => l[0] === 'inserted-alert')) missing.push('plan error inserted as role=alert');
    if (!live.some(l => l[0] === 'destination-search-error' && l[1])) missing.push('search error text in the alert region');
    const roles = await evaluate(collectStructure);
    const regions = roles.liveRegions.map(r => `${r.el}[${r.role || r.live}]`);
    record('live-regions', missing.length === 0, missing.length ? `missing: ${missing.join(', ')}` : `announcements observed in order: ${JSON.stringify(texts)}; regions ${JSON.stringify(regions)}`);
    if (missing.length) findings.push({ area: 'live-regions', detail: missing.join(', ') });
    results[results.length - 1].observed = live;
  }

  // ---- 5. Reduced motion ---------------------------------------------------------------------------------
  if (want('motion')) {
    await b.viewport(375, 812);
    await b.navigate(`${preview.url}/?motion=${Date.now()}`);
    await planDays(3);
    const measure = () => evaluate(`(() => {
      const secs = value => Math.max(0, ...String(value).split(',').map(v => { v = v.trim(); return v.endsWith('ms') ? parseFloat(v) / 1000 : parseFloat(v) || 0; }));
      let maxTransition = 0, maxAnimation = 0, worst = '';
      for (const el of document.querySelectorAll('body, body *')) {
        const cs = getComputedStyle(el);
        const t = secs(cs.transitionDuration) ; const a = cs.animationName !== 'none' ? secs(cs.animationDuration) : 0;
        if (t > maxTransition) { maxTransition = t; worst = 'transition ' + el.className; }
        if (a > maxAnimation) { maxAnimation = a; worst = 'animation ' + el.className; }
      }
      return { maxTransition, maxAnimation, worst, scrollBehavior: getComputedStyle(document.documentElement).scrollBehavior, matches: matchMedia('(prefers-reduced-motion: reduce)').matches };
    })()`);
    const normal = await measure();
    await call('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-reduced-motion', value: 'reduce' }] });
    const reduced = await measure();
    // Focus scrolling after planning must not animate when reduced motion is on.
    await evaluate("document.querySelector('.trip-form').requestSubmit()");
    await until("document.querySelector('#trip-status')?.textContent.includes('updated')");
    await call('Emulation.setEmulatedMedia', { features: [] });
    const ok = normal.maxAnimation > 0.1 && reduced.matches && reduced.maxTransition <= 0.001 && reduced.maxAnimation <= 0.001 && reduced.scrollBehavior === 'auto';
    record('reduced-motion', ok, `normal: longest transition ${normal.maxTransition}s, longest animation ${normal.maxAnimation}s, scroll-behavior ${normal.scrollBehavior}; with prefers-reduced-motion: reduce: transition ${reduced.maxTransition}s, animation ${reduced.maxAnimation}s, scroll-behavior ${reduced.scrollBehavior}`);
  }

  // ---- 6. 200% zoom, 320px reflow and clipped content -------------------------------------------------------
  if (want('zoom')) {
    const zoomRows = [];
    const zoomProblems = [], informational = [];
    // Browser zoom Z shrinks the CSS viewport to width/Z with Z device pixels per CSS pixel.
    for (const [name, width, height, scale] of [['1280px at 200%', 640, 400, 2], ['1440px at 200%', 720, 500, 2], ['1440px at 400% (360 CSS px)', 360, 250, 4], ['1280px at 400% (WCAG 1.4.10, 320 CSS px)', 320, 256, 4], ['375px at 200% (188 CSS px, below the 320px WCAG 1.4.10 reflow width: informational)', 188, 406, 2]]) {
      await call('Emulation.setDeviceMetricsOverride', { width, height, deviceScaleFactor: scale, mobile: false });
      await b.navigate(`${preview.url}/?zoom=${Date.now()}`);
      await planDays(3);
      await evaluate("document.querySelectorAll('.itinerary-day').forEach(d => { d.open = true; })");
      const m = await evaluate(`(() => {
        const clipped = [];
        for (const el of document.querySelectorAll('body *')) {
          const cs = getComputedStyle(el);
          if (['hidden', 'clip'].includes(cs.overflowX) && el.scrollWidth > el.clientWidth + 1 && el.clientWidth > 0 && !el.matches('.tt-visually-hidden, .tt-visually-hidden *, .destination-banner, .destination-photo, .itinerary-stop-media, .stay-media, .video-thumbnail, .map-card, .itinerary-stop, .itinerary-days, .video-player') ) clipped.push(el.className.toString().slice(0, 40) + ' ' + el.scrollWidth + '>' + el.clientWidth);
        }
        return { scrollWidth: document.documentElement.scrollWidth, innerWidth, clipped: clipped.slice(0, 6), dpr: devicePixelRatio, submitVisible: !!document.querySelector('.trip-submit') };
      })()`);
      const overflowing = await evaluate(`(() => [...document.querySelectorAll('body *')].filter(el => { const r = el.getBoundingClientRect(); return r.right > innerWidth + 1 && !el.closest('.tt-visually-hidden') && getComputedStyle(el).position !== 'fixed'; }).slice(0, 5).map(el => el.tagName.toLowerCase() + '.' + el.className.toString().slice(0, 30) + ' right=' + Math.round(el.getBoundingClientRect().right)))()`);
      zoomRows.push({ name, cssWidth: width, ...m, overflowing });
      if (width === 188) { informational.push(`${name}: scrollWidth ${m.scrollWidth} vs ${m.innerWidth}${overflowing.length ? ' (' + overflowing.join(', ') + ')' : ''}`); continue; }
      if (m.scrollWidth > m.innerWidth) zoomProblems.push(`${name}: horizontal scroll ${m.scrollWidth} > ${m.innerWidth}${overflowing.length ? ' (' + overflowing.join(', ') + ')' : ''}`);
      if (m.clipped.length) zoomProblems.push(`${name}: clipped content ${m.clipped.join('; ')}`);
      if (scale === 2 && width === 720) { await writeFile(new URL('zoom-200-1440.png', output), await b.shot()); shots.push('zoom-200-1440.png'); }
      if (width === 320) { await writeFile(new URL('reflow-320.png', output), await b.shot()); shots.push('reflow-320.png'); }
    }
    record('zoom-200-and-reflow', zoomProblems.length === 0, zoomProblems.length ? zoomProblems.join(' | ') : `no horizontal scroll or clipped text at ${zoomRows.map(r => `${r.cssWidth}px (${r.name}: scrollWidth ${r.scrollWidth})`).join(', ')}`);
    zoomProblems.forEach(p => findings.push({ area: 'zoom', detail: p }));
    results[results.length - 1].rows = zoomRows;
    results[results.length - 1].informational = informational;
  }

  // ---- 7. Target sizes -------------------------------------------------------------------------------------------------
  if (want('targets')) {
    const targetProblems = [], smallInline = [], creditsUnder44 = [], videoTitleEquivalent = [], report = {};
    for (const [width, height] of WIDTHS) {
      await b.viewport(width, height);
      await b.navigate(`${preview.url}/?targets=${Date.now()}`);
      await planDays(3);
      await evaluate("document.querySelector('[data-video-expand]')?.click()");
      const targets = await evaluate(collectTargets);
      report[width] = targets.length;
      // WCAG 2.5.8 spacing exception: an undersized target passes if a 24px circle centred on it touches no other target.
      const isolated = t => { const [cx, cy] = [t.x + t.w / 2, t.y + t.h / 2]; return !targets.some(o => o !== t && o.x < cx + 12 && o.x + o.w > cx - 12 && o.y < cy + 12 && o.y + o.h > cy - 12); };
      for (const t of targets) {
        const below24 = (t.w < 24 || t.h < 24) && !isolated(t);
        const below44 = t.h < 44 || t.w < 44;
        // A video card's title link duplicates its 44px thumbnail button and "Watch on YouTube" link (WCAG 2.5.8 equivalent-control exception).
        if (/^a\.video-title/.test(t.el)) { if (t.h < 44) videoTitleEquivalent.push(`${width}: ${t.w}x${t.h}`); continue; }
        if (t.sentence) { if (below24) smallInline.push(`${width}: ${t.el} "${t.text}" ${t.w}x${t.h} (inline in a sentence)`); continue; }
        // Photo credit links sit on a small image and carry licence text: the WCAG 2.5.8 minimum (24px) applies.
        // Every other control is held to the 44px the stylesheets set for buttons, links and fields.
        if (/^Photo: /.test(t.text) && t.el.startsWith('a.')) { if (below24) targetProblems.push(`${width}: credit ${t.el} "${t.text}" ${t.w}x${t.h} is under 24px`); else if (below44) creditsUnder44.push(`${width}: "${t.text}" ${t.w}x${t.h}`); continue; }
        if (below44) targetProblems.push(`${width}: ${t.el} "${t.text}" ${t.w}x${t.h}`);
      }
    }
    const uniq = [...new Set(targetProblems)];
    record('touch-targets', uniq.length === 0, uniq.length ? `${uniq.length} standalone targets under 44px: ${uniq.slice(0, 8).join(' | ')}` : `${JSON.stringify(report)} interactive targets measured: every control is at least 44x44px except ${new Set(creditsUnder44).size} photo-credit links (at least 24px, spacing-exempt below that) and ${new Set(videoTitleEquivalent).size} video-title sizes that duplicate 44px controls; ${new Set(smallInline).size} inline sentence links are under 24px and exempt (WCAG 2.5.8)`);
    uniq.forEach(p => findings.push({ area: 'targets', detail: p }));
    results[results.length - 1].inlineUnder24 = [...new Set(smallInline)];
    results[results.length - 1].creditsUnder44 = [...new Set(creditsUnder44)];
    results[results.length - 1].videoTitleUnder24 = [...new Set(videoTitleEquivalent)];
  }

  // ---- 8. Hover states of interactive colours -----------------------------------------------------------------------
  if (want('hover')) {
    await b.viewport(1440, 1000);
    await b.navigate(`${preview.url}/?hover=${Date.now()}`);
    await planDays(3);
    const selectors = ['.tt-header-navigation a', '.tt-section-navigation a', '.tt-brand', '.trip-submit', '.itinerary-action', '.itinerary-directions', '.itinerary-day-summary', '.stay-prices', '.stay-website', '.video-title', '.video-watch', '.map-action', '.tt-footer-links a', '.guide-attribution a', '.destination-photo-credit a'];
    const rows = [];
    for (const selector of selectors) {
      const target = await evaluate(`(() => { const el = document.querySelector(${JSON.stringify(selector)}); if (!el) return null; window.scrollTo({ top: Math.max(0, el.getBoundingClientRect().top + scrollY - 220), behavior: 'instant' }); const r = el.getBoundingClientRect(); return { x: r.left + Math.min(r.width / 2, 40), y: r.top + r.height / 2 }; })()`);
      if (!target) { rows.push({ selector, missing: true }); continue; }
      await waitScrollSettled(); await sleep(100);
      const moved = await evaluate(`(() => { const el = document.querySelector(${JSON.stringify(selector)}); const r = el.getBoundingClientRect(); return { x: r.left + Math.min(r.width / 2, 40), y: r.top + r.height / 2 }; })()`);
      await call('Input.dispatchMouseEvent', { type: 'mouseMoved', x: moved.x, y: moved.y });
      await sleep(120);
      await call('Input.dispatchMouseEvent', { type: 'mouseMoved', x: moved.x + 1, y: moved.y });
      await sleep(260); // longer than the 160ms transitions
      const hovered = await evaluate(`(() => {
        const el = document.querySelector(${JSON.stringify(selector)}); const canvas = document.createElement('canvas'); canvas.width = canvas.height = 1; const ctx = canvas.getContext('2d', { willReadFrequently: true });
        const parse = v => { ctx.clearRect(0, 0, 1, 1); ctx.fillStyle = '#000'; ctx.fillStyle = v; ctx.fillRect(0, 0, 1, 1); const d = ctx.getImageData(0, 0, 1, 1).data; return { rgb: [d[0], d[1], d[2]], a: d[3] / 255 }; };
        const layers = []; for (let n = el; n; n = n.parentElement) { const c = parse(getComputedStyle(n).backgroundColor); if (c.a > 0) layers.push(c); if (c.a === 1) break; }
        let bg = [255, 255, 255]; for (const l of layers.reverse()) bg = l.rgb.map((c, i) => c * l.a + bg[i] * (1 - l.a));
        return { hover: el.matches(':hover'), color: parse(getComputedStyle(el).color).rgb, bg, size: parseFloat(getComputedStyle(el).fontSize) };
      })()`);
      const over = /credit/.test(selector);
      rows.push({ selector, hover: hovered.hover, ratio: round(ratio(hovered.color, hovered.bg)), color: hovered.color, bg: hovered.bg, overPhoto: over });
    }
    await call('Input.dispatchMouseEvent', { type: 'mouseMoved', x: 5, y: 5 });
    const bad = rows.filter(r => !r.missing && !r.overPhoto && (r.ratio < 4.5 || !r.hover));
    record('hover-states', bad.length === 0, bad.length ? bad.map(r => `${r.selector} ${r.ratio}:1 hover=${r.hover}`).join(' | ') : `${rows.filter(r => !r.missing).length} hovered controls: lowest text ratio ${Math.min(...rows.filter(r => !r.missing && !r.overPhoto).map(r => r.ratio))}:1 (photo credits are measured on the overlay in the pixel pass)`);
    results[results.length - 1].rows = rows;
    bad.forEach(r => findings.push({ area: 'hover', ...r }));
  }

  // ---- 8b. The current section is not shown by colour alone ---------------------------------------------------------
  if (want('current')) {
    const rows = [];
    for (const [width, height, open] of [[1440, 1000, false], [375, 812, true]]) {
      await b.viewport(width, height);
      await b.navigate(`${preview.url}/?current=${Date.now()}`);
      if (open) await evaluate("document.querySelector('.tt-menu-button').click()");
      const nav = await evaluate(`(() => {
        const read = (root) => { const links = [...document.querySelectorAll(root + ' a[data-tt-section]')].filter(a => a.getBoundingClientRect().width > 0); const cur = links.find(a => a.hasAttribute('aria-current')); const other = links.find(a => !a.hasAttribute('aria-current'));
          if (!cur || !other) return null; const pick = a => { const cs = getComputedStyle(a); return { color: cs.color, decoration: cs.textDecorationLine, weight: cs.fontWeight, borderBottom: cs.borderBottomWidth === '0px' ? 'none' : cs.borderBottomWidth + ' ' + cs.borderBottomStyle + ' ' + cs.borderBottomColor, borderTop: cs.borderTopWidth === '0px' ? 'none' : cs.borderTopWidth + ' ' + cs.borderTopStyle + ' ' + cs.borderTopColor }; };
          return { root, cur: pick(cur), other: pick(other) }; };
        return ['.tt-header-navigation', '.tt-section-navigation', '.tt-bottom-navigation'].map(read).filter(Boolean);
      })()`);
      for (const n of nav) {
        const nonColour = ['decoration', 'weight', 'borderBottom', 'borderTop'].filter(k => n.cur[k] !== n.other[k]);
        rows.push({ width, nav: n.root, nonColourCues: nonColour, current: n.cur, other: n.other });
      }
    }
    const bad = rows.filter(r => r.nonColourCues.length === 0);
    record('current-section-not-colour-only', bad.length === 0, bad.length ? `${bad.map(r => `${r.width}px ${r.nav}`).join(', ')}: current link differs only by colour` : `${rows.length} navigation bars: the current section also differs by ${rows.map(r => `${r.nav.replace('.tt-', '')} ${r.nonColourCues.join('/')}`).join('; ')}`);
    results[results.length - 1].rows = rows;
    bad.forEach(r => findings.push({ area: 'colour-only', width: r.width, detail: `${r.nav} shows the current section by colour only` }));
  }

  // ---- 9. Menu, focus after an error, control names, text spacing ---------------------------------------------------
  if (want('misc')) {
    await b.viewport(375, 812);
    await b.navigate(`${preview.url}/?menu=${Date.now()}`);
    await evaluate("document.querySelector('.tt-menu-button').focus()");
    const closed = await evaluate("({ expanded: document.querySelector('.tt-menu-button').getAttribute('aria-expanded'), hidden: document.getElementById('header-navigation').hidden, label: document.querySelector('.tt-menu-button').getAttribute('aria-label') })");
    await b.key('Enter', 'Enter', 13, 0, '\r'); await sleep(100);
    const open = await evaluate("({ expanded: document.querySelector('.tt-menu-button').getAttribute('aria-expanded'), hidden: document.getElementById('header-navigation').hidden, label: document.querySelector('.tt-menu-button').getAttribute('aria-label') })");
    await b.tab(); await sleep(60);
    const firstLink = await evaluate("document.activeElement.textContent.trim()");
    await b.key('Escape', 'Escape', 27); await sleep(100);
    const escaped = await evaluate("({ expanded: document.querySelector('.tt-menu-button').getAttribute('aria-expanded'), hidden: document.getElementById('header-navigation').hidden, focus: document.activeElement.className })");
    record('mobile-menu-keyboard', closed.expanded === 'false' && closed.hidden && open.expanded === 'true' && !open.hidden && firstLink === 'Overview' && escaped.expanded === 'false' && escaped.hidden && /tt-menu-button/.test(escaped.focus),
      `375px: closed (aria-expanded=${closed.expanded}, hidden=${closed.hidden}); Enter opens it (aria-expanded=${open.expanded}, label "${open.label}"), the next Tab reaches "${firstLink}"; Escape closes it and returns focus to the menu button`);
  }
  if (want('misc')) {
    await b.viewport(1440, 1000);
    await b.navigate(`${preview.url}/?names=${Date.now()}`);
    const named = await evaluate(`(() => { const o = document.querySelector('#trip-end-value'); const label = o.getAttribute('aria-labelledby'); return { labelledby: label, name: label ? label.split(' ').map(id => document.getElementById(id)?.textContent.trim()).join(' ') : '' }; })()`);
    const { root } = await call('DOM.getDocument');
    const { nodeId } = await call('DOM.querySelector', { nodeId: root.nodeId, selector: '#trip-end-value' });
    const partial = await call('Accessibility.getPartialAXTree', { nodeId, fetchRelatives: false });
    const axName = (partial.nodes[0]?.name?.value || '').trim();
    record('end-date-output-named', axName === 'Ends', `the "Ends" read-only date is exposed as ${JSON.stringify(partial.nodes[0]?.role?.value)} named ${JSON.stringify(axName)} (aria-labelledby=${named.labelledby})`);
    if (axName !== 'Ends') findings.push({ area: 'names', detail: `#trip-end-value has accessible name ${JSON.stringify(axName)}; its visible label is "Ends"` });
  }
  if (want('misc')) {
    // Focus after a failed plan: the form controls are replaced, so where does the next Tab go?
    await b.viewport(1440, 1000);
    await b.navigate(`${preview.url}/?errfocus=${Date.now()}`);
    await evaluate("document.querySelector('.trip-form').elements.days.value = '4'; document.querySelector('.trip-submit').focus()");
    await b.key('Enter', 'Enter', 13, 0, '\r');
    await until("!!document.querySelector('.trip-error')"); await sleep(300);
    const afterSwap = await evaluate("({ tag: document.activeElement.tagName, inTrip: !!document.activeElement.closest('#trip-card'), cls: document.activeElement.className })");
    await b.tab(); await sleep(150);
    const nextStop = await evaluate("({ tag: document.activeElement.tagName, cls: document.activeElement.className, inOverview: !!document.activeElement.closest('#overview'), skip: document.activeElement.classList.contains('tt-skip-link'), text: document.activeElement.textContent.trim().slice(0, 30) })");
    record('focus-after-plan-error', afterSwap.inTrip || (nextStop.inOverview && !nextStop.skip),
      `after a keyboard-submitted failing plan focus is on <${afterSwap.tag.toLowerCase()}> (inside the form card: ${afterSwap.inTrip}); the next Tab reaches ${nextStop.tag.toLowerCase()}.${nextStop.cls.split(' ')[0]} "${nextStop.text}" (inside #overview: ${nextStop.inOverview})`);
  }
  if (want('misc')) {
    // WCAG 1.4.12 text spacing: override spacing and confirm nothing overflows or is clipped.
    const rows = [], problems = [];
    for (const [width, height] of WIDTHS) {
      await b.viewport(width, height);
      await b.navigate(`${preview.url}/trip/lisbon-pt?days=3&from=${tomorrow}`);
      await evaluate("document.querySelectorAll('.itinerary-day').forEach(d => { d.open = true; })");
      await evaluate("(() => { const s = document.createElement('style'); s.id = 'spacing'; s.textContent = '* { line-height: 1.5 !important; letter-spacing: 0.12em !important; word-spacing: 0.16em !important; } p { margin-bottom: 2em !important; }'; document.head.appendChild(s); })()");
      await sleep(200);
      const m = await evaluate(`(() => {
        const clipped = [];
        for (const el of document.querySelectorAll('body *')) {
          const cs = getComputedStyle(el);
          if (el.closest('.tt-visually-hidden') || !el.textContent.trim()) continue;
          const hidesY = ['hidden', 'clip'].includes(cs.overflowY), hidesX = ['hidden', 'clip'].includes(cs.overflowX);
          if ((hidesY && el.scrollHeight > el.clientHeight + 1 && el.clientHeight > 0 && !el.matches('.destination-banner, .itinerary-days, .map-card, .destination-photo, .itinerary-stop, .itinerary-stop-media')) || (hidesX && el.scrollWidth > el.clientWidth + 1 && el.clientWidth > 0 && !el.matches('.destination-banner, .itinerary-days, .map-card, .destination-photo, .itinerary-stop, .itinerary-stop-media'))) clipped.push(el.className.toString().slice(0, 40) + ' ' + el.scrollWidth + 'x' + el.scrollHeight + ' in ' + el.clientWidth + 'x' + el.clientHeight);
        }
        return { scrollWidth: document.documentElement.scrollWidth, innerWidth, clipped: clipped.slice(0, 6) };
      })()`);
      rows.push({ width, ...m });
      if (m.scrollWidth > m.innerWidth) problems.push(`${width}: horizontal scroll ${m.scrollWidth} > ${m.innerWidth}`);
      if (m.clipped.length) problems.push(`${width}: clipped ${m.clipped.join('; ')}`);
      if (width === 375) { await writeFile(new URL('text-spacing-375.png', output), await b.shot()); shots.push('text-spacing-375.png'); }
    }
    record('text-spacing', problems.length === 0, problems.length ? problems.join(' | ') : `with line-height 1.5, letter-spacing .12em, word-spacing .16em and 2em paragraph spacing: no horizontal scroll or clipped text (${JSON.stringify(rows.map(r => [r.width, r.scrollWidth]))})`);
    problems.forEach(p => findings.push({ area: 'text-spacing', detail: p }));
  }

  assert.deepEqual(b.errors, [], 'Unexpected JavaScript exceptions');
  pass('js-exceptions', 'no uncaught JavaScript exceptions during the audit');
} finally {
  await writeFile(new URL('a11y-results.json', output), JSON.stringify({ browser: b.product, generated: new Date().toISOString(), axe: axeSource ? axeSource.match(/axe v([\d.]+)/)?.[1] : null, results, findings, shots,
    contrastLowest: [...allPairs].sort((p, q) => p.ratio / p.required - q.ratio / q.required).slice(0, 40), pairCount: allPairs.length, states: [...new Set(allPairs.map(p => p.label))], axResults, focusResults, errors: b.errors }, null, 2));
  await writeFile(new URL('contrast-pairs.json', output), JSON.stringify(allPairs, null, 1));
  await b.close();
  for (const p of [preview, fallbackPreview, wideVideos, wideStays, uncertain]) p.stop();
}
const failed = results.filter(r => r.status === 'fail');
console.log(`\n${results.length - failed.length} passed, ${failed.length} failed`);
for (const f of failed) console.log(`  FAIL ${f.id}: ${f.detail}`);
process.exit(failed.length ? 1 : 0);
