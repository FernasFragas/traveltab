#!/usr/bin/env node
// Screenshots AND assertions for every named scenario in tasks/redesign/preview.md: the 12 data
// variants, the routes in the scenario table, the destination photo fixtures and the city guide
// fixtures, at 375 and 1440 px, initial and planned states where they apply.
//
//   CDP_PORT=9327 HTMX_PATH=/tmp/htmx.js [OUTPUT_DIR=dir] node <this file>
//
// The script starts (and stops) its own preview processes on free localhost ports, one per
// scenario, and attaches to an isolated Chromium/Brave with its own debugging port and temporary
// profile. Third-party images (stop photos, video thumbnails) are answered with a generated local
// gradient image and the YouTube embed is blocked, so nothing depends on a live service and the
// screenshots are repeatable. Every scenario asserts its expected text and state; any mismatch
// fails the run (non-zero exit). Screenshots are WebP (quality 80) to keep the evidence small.
import assert from 'node:assert/strict';
import { readFile, mkdir, writeFile } from 'node:fs/promises';
import { connect, startPreview, sleep, repoRoot } from './lib/browser.mjs';

const output = new URL(process.env.OUTPUT_DIR ? `file://${process.env.OUTPUT_DIR.replace(/\/?$/, '/')}` : '../../artifacts/redesign/final-verification-2/scenarios/', import.meta.url);
await mkdir(output, { recursive: true });
const guides = JSON.parse(await readFile(new URL('guides/guides.json', `file://${repoRoot}`), 'utf8'));
const tomorrow = new Date(Date.now() + 86400000).toISOString().slice(0, 10);
const plus = (date, days) => { const d = new Date(`${date}T00:00:00Z`); d.setUTCDate(d.getUTCDate() + days); return d.toISOString().slice(0, 10); };
const WIDTHS = [[375, 812], [1440, 1000]];

const b = await connect();
const { evaluate, until, call } = b;
const record = [];
let checks = 0;
const ok = (cond, message) => { checks++; assert.ok(cond, message); };
const eq = (actual, expected, message) => { checks++; assert.deepEqual(actual, expected, message); };
const has = (text, re, message) => { checks++; assert.match(text ?? '', re, message); };

// --- Reading the page -----------------------------------------------------------------------------
const READ = `(() => {
  const txt = el => el ? el.textContent.replace(/\\s+/g, ' ').trim() : null;
  const shown = el => !!el && getComputedStyle(el).display !== 'none' && el.getClientRects().length > 0;
  const figure = document.querySelector('.destination-photo'); const img = document.querySelector('.destination-image');
  const days = [...document.querySelectorAll('.itinerary-day')].map(d => ({ label: txt(d.querySelector('.itinerary-day-title')),
    chips: [...d.querySelectorAll('.itinerary-chip')].map(txt), walk: txt(d.querySelector('.itinerary-walk')), stops: d.querySelectorAll('.itinerary-stop').length,
    photos: d.querySelectorAll('.itinerary-stop-photo').length, failedPhotos: d.querySelectorAll('.itinerary-stop-media-failed').length, open: d.open, directions: !!d.querySelector('.itinerary-directions') }));
  const stays = [...document.querySelectorAll('.stay-item')].map(li => ({ name: txt(li.querySelector('.stay-name')), kind: txt(li.querySelector('.stay-kind')), stars: txt(li.querySelector('.stay-stars')), website: !!li.querySelector('.stay-website'), image: !!li.querySelector('.stay-image') }));
  const cards = [...document.querySelectorAll('.video-card')];
  const guide = document.querySelector('.guide-card');
  const form = document.querySelector('.trip-form');
  return {
    url: location.pathname + location.search,
    title: txt(document.querySelector('.destination-title')), country: form?.elements.country?.value || null,
    tagline: txt(document.querySelector('.destination-tagline')), countryLabel: txt(document.querySelector('.destination-country')),
    weather: [...document.querySelectorAll('.weather-metric')].map(m => txt(m.querySelector('.weather-value')) + ' ' + txt(m.querySelector('.weather-label'))),
    hero: figure ? { state: figure.dataset.photoState, imgPresent: !!img, loaded: !!img && img.complete && img.naturalWidth > 0, src: img?.getAttribute('src') || null, alt: img?.getAttribute('alt') ?? null,
      credit: txt(document.querySelector('.destination-photo-credit')), creditShown: shown(document.querySelector('.destination-photo-credit')), caption: txt(document.querySelector('.destination-photo-caption')),
      fallbackShown: shown(document.querySelector('.destination-photo-fallback')), fallbackText: txt(document.querySelector('.destination-photo-fallback')), imgShown: shown(img), frame: figure.getBoundingClientRect().height } : null,
    mapFrame: document.querySelector('.map-embed')?.getAttribute('title') || null, mapSrc: document.querySelector('.map-embed')?.getAttribute('src') || null,
    guide: guide ? { state: guide.dataset.guideState, heading: txt(document.querySelector('.guide-title')), paragraphs: [...document.querySelectorAll('.guide-paragraph')].map(txt), attribution: txt(document.querySelector('.guide-attribution')), review: txt(document.querySelector('.guide-review')),
      links: [...document.querySelectorAll('.guide-attribution a')].map(a => a.getAttribute('href')), search: document.querySelector('.guide-link')?.getAttribute('href') || null, count: document.querySelectorAll('.guide-card').length } : null,
    form: form ? { start: form.elements.start.value, days: form.elements.days.value, ends: txt(document.querySelector('#trip-end-value')), submitDisabled: document.querySelector('.trip-submit').disabled, busy: form.getAttribute('aria-busy') } : null,
    tripError: txt(document.querySelector('.trip-error')), searchError: txt(document.querySelector('#destination-search-error')), tripStatus: txt(document.querySelector('#trip-status')), searchStatus: txt(document.querySelector('#destination-search-status')),
    itineraryEmpty: txt(document.querySelector('.itinerary-empty')), itineraryNote: txt(document.querySelector('.itinerary-note')), days,
    exports: { ics: document.querySelector('.itinerary-action[href*=".ics"]')?.getAttribute('href') || null, kml: document.querySelector('.itinerary-action[href*=".kml"]')?.getAttribute('href') || null },
    stays, stayNote: txt(document.querySelector('.stay-note')), booking: document.querySelector('.stay-prices')?.getAttribute('href') || null,
    videoTotal: document.querySelectorAll('[data-video-activate]').length, videoShown: cards.filter(c => shown(c)).length, videoNote: txt(document.querySelector('.video-note')), videoCount: txt(document.querySelector('.video-count')),
    videoTitles: cards.map(c => txt(c.querySelector('.video-title'))), expandButton: document.querySelector('[data-video-expand]') ? { label: txt(document.querySelector('[data-video-expand]')), expanded: document.querySelector('[data-video-expand]').getAttribute('aria-expanded') } : null,
    failedThumbs: document.querySelectorAll('.video-thumbnail-failed').length,
    sections: ['overview', 'itinerary', 'stays', 'videos'].map(id => document.querySelectorAll('section#' + id).length),
    overflow: document.documentElement.scrollWidth, viewport: innerWidth,
    idsUnique: [...document.querySelectorAll('[id]')].every(n => document.querySelectorAll('#' + CSS.escape(n.id)).length === 1)
  };
})()`;
const read = () => evaluate(READ);

// --- Navigation helpers ----------------------------------------------------------------------------------
const search = async (term, title) => {
  await evaluate("window.__prev = document.querySelector('.destination-banner')");
  await evaluate(`document.querySelector('#city_name').value = ${JSON.stringify(term)}; document.querySelector('#destination-search').requestSubmit()`);
  await until(`document.querySelector('.destination-title')?.textContent.trim() === ${JSON.stringify(title)} && document.querySelector('.destination-banner') !== window.__prev`);
  await settleHero();
};
// The hero settles as ready, unavailable or failed; a broken image needs its error event.
const settleHero = async () => {
  await until("(() => { const f = document.querySelector('.destination-photo'); const i = document.querySelector('.destination-image'); return !f || f.dataset.photoState !== 'ready' || !i || i.complete; })()", 10000);
  await b.settle();
};
const plan = async (days = 3) => {
  await evaluate(`(() => { const f = document.querySelector('.trip-form'); f.elements.days.value = '${days}'; f.requestSubmit(); })()`);
  await until(`document.querySelectorAll('.itinerary-day').length === ${days} && !document.querySelector('.trip-submit').disabled`);
  await b.settle();
  await sleep(350); // planning focuses the itinerary heading and scrolls to it on the next tick
};
const load = async url => { await b.navigate(url); await settleHero(); };

// --- Screenshots ----------------------------------------------------------------------------------------------
const files = [];
const forceMap = () => evaluate(`(async () => {
  const frame = document.querySelector('.map-embed'); if (!frame || frame.dataset.evidenceMapLoaded === 'true') return;
  await new Promise((resolve, reject) => { const timer = setTimeout(() => reject(Error('fixture map iframe did not load')), 8000);
    frame.addEventListener('load', () => { clearTimeout(timer); frame.dataset.evidenceMapLoaded = 'true'; resolve(); }, { once: true }); frame.loading = 'eager'; frame.src = frame.src; });
})()`);
const write = async (name, buffer) => { await mkdir(new URL('./', new URL(name, output)), { recursive: true }); await writeFile(new URL(name, output), buffer); files.push(name); };
// Full-page capture; the fixed bottom bar and the off-screen skip link would repeat, so they are hidden for it.
const shoot = async (name, { viewport = false } = {}) => {
  await forceMap();
  await until("(window.scrollTo({ top: 0, behavior: 'instant' }), window.scrollY < 1)");
  await evaluate('document.activeElement && document.activeElement.blur && document.activeElement.blur()');
  if (viewport) await write(`${name}-viewport.webp`, Buffer.from((await call('Page.captureScreenshot', { format: 'webp', quality: 80 })).data, 'base64'));
  await evaluate("window.__hidden = [...document.querySelectorAll('.tt-bottom-navigation, .tt-skip-link')].map(n => [n, n.style.visibility]); window.__hidden.forEach(([n]) => { n.style.visibility = 'hidden'; })");
  try { await write(`${name}-full.webp`, Buffer.from((await call('Page.captureScreenshot', { format: 'webp', quality: 80, captureBeyondViewport: true })).data, 'base64')); }
  finally { await evaluate('window.__hidden.forEach(([n, v]) => { n.style.visibility = v; })'); }
};

// --- Generic invariants every rendered page must satisfy -----------------------------------------------------
const invariants = (s, width, ctx) => {
  eq(s.sections, [1, 1, 1, 1], `${ctx}: each section id exactly once`);
  ok(s.idsUnique, `${ctx}: element ids are unique`);
  ok(s.overflow <= width, `${ctx}: horizontal overflow ${s.overflow} > ${width}`);
  ok(s.title, `${ctx}: a destination title`);
  eq(s.weather.length, 3, `${ctx}: three weather metrics`);
  ok(s.mapFrame?.startsWith('Live Waze map of '), `${ctx}: the map iframe is named`);
  ok(s.guide && s.guide.count === 1, `${ctx}: exactly one guide card`);
};
const guideKey = s => `${s.title.toLowerCase()}-${s.country}`;
const expectGuide = (s, scenario, ctx) => {
  if (s.title === 'Longguide' && scenario !== 'fallback') { eq(s.guide.state, 'reviewed', ctx); ok(s.guide.paragraphs[0].includes('<b>markup characters</b>'), `${ctx}: markup shown literally`); return; }
  const entry = scenario === 'fallback' ? null : guides[guideKey(s)]?.reviewed ? guides[guideKey(s)] : null;
  eq(s.guide.heading, `About ${s.title}`, `${ctx}: guide is about the destination shown`);
  if (!entry) {
    eq(s.guide.state, 'missing', `${ctx}: ${guideKey(s)} has no reviewed guide`);
    has(s.guide.paragraphs[0], /^We don't have a reviewed guide for .+ yet\./, ctx);
    ok(s.guide.search?.startsWith('https://en.wikivoyage.org/w/index.php?search='), `${ctx}: Wikivoyage search link`);
    eq(s.guide.links.length, 0, `${ctx}: no attribution without a guide`);
  } else {
    eq(s.guide.state, 'reviewed', `${ctx}: ${guideKey(s)} has a reviewed guide`);
    eq(s.guide.paragraphs.join('\n\n'), entry.intro.split('\n\n').map(p => p.replace(/\s+/g, ' ').trim()).join('\n\n'), `${ctx}: reviewed text unchanged`);
    has(s.guide.attribution, new RegExp(`revision ${entry.wikivoyage_revision}\\b`), ctx);
    eq(s.guide.links.length, 4, `${ctx}: article, revision, authors and licence links`);
  }
};
// Destination photo expectation by the resolved identity (cmd/design-preview/photos.go).
const photoExpectation = (s, scenario) => {
  if (s.title === 'Lisbon') return 'ready';
  if (scenario === 'photos-missing' || scenario === 'fallback') return 'unavailable';
  if (s.title === 'Brokenimage') return 'failed';
  if (['Porto', 'Paris'].includes(s.title)) return 'ready';
  return 'unavailable';
};
const expectHero = (s, scenario, ctx) => {
  const expected = photoExpectation(s, scenario);
  eq(s.hero.state, expected, `${ctx}: hero photo state`);
  if (expected === 'ready') {
    ok(s.hero.loaded && s.hero.imgShown, `${ctx}: the photo actually loaded`);
    ok(s.hero.creditShown && /^Photo: /.test(s.hero.credit), `${ctx}: credit is shown`);
    ok(!s.hero.fallbackShown, `${ctx}: no "Photo unavailable" over a real photo`);
    ok(s.hero.alt && s.hero.alt.includes(s.title), `${ctx}: alt text names the destination (${s.hero.alt})`);
    if (s.title === 'Lisbon') has(s.hero.src, /^\/images\/destinations\/lisbon-alfama\.jpg$/, `${ctx}: the curated local hero`);
    else has(s.hero.credit, new RegExp(`Fixture Photographer \\(${s.country === 'us' ? 'Paris, Texas' : s.title}\\)`), `${ctx}: credit names the right destination's photographer`);
  } else {
    ok(s.hero.fallbackShown && s.hero.fallbackText === 'Photo unavailable', `${ctx}: "Photo unavailable" is shown`);
    ok(!s.hero.creditShown, `${ctx}: no credit without a photo`);
    ok(!s.hero.imgShown, `${ctx}: no image shown`);
    ok(s.hero.frame >= 160, `${ctx}: the hero frame keeps its size (${s.hero.frame}px)`);
  }
};
const weatherFor = s => [`22°C ${s.title === 'Porto' ? 'Partly cloudy' : 'Clear'}`, '60% Humidity', '1.2 m Wave height'];

// --- What each data variant shows (preview.md "Deterministic data variants") ------------------------------------------
const VARIANTS = ['default', 'photos-missing', 'forecast-absent', 'forecast-uncertain', 'stays-0', 'stays-1', 'stays-6', 'stays-unavailable', 'videos-0', 'videos-1', 'videos-10', 'fallback'];
const videosFor = (scenario, title) => title === 'Porto' || ['videos-0', 'fallback'].includes(scenario) ? 0 : scenario === 'videos-1' ? 1 : scenario === 'videos-10' ? 10 : 6;
const expectVideos = (s, scenario, ctx) => {
  const total = videosFor(scenario, s.title);
  eq(s.videoTotal, total, `${ctx}: video count`);
  if (total === 0) { has(s.videoNote, /^No travel videos are available for this destination right now\.$/, ctx); eq(s.videoShown, 0, ctx); ok(!s.expandButton, `${ctx}: no expand button`); return; }
  eq(s.videoShown, Math.min(total, 4), `${ctx}: videos visible before expansion`);
  if (total > 4) { eq(s.expandButton?.label, 'View all videos', ctx); eq(s.expandButton?.expanded, 'false', ctx); eq(s.videoCount, `Showing the first 4 of ${total} videos.`, ctx); }
  else ok(!s.expandButton && !s.videoCount, `${ctx}: no expansion for ${total} videos`);
};
const expectInitial = (s, scenario, ctx, width) => {
  invariants(s, width, ctx);
  expectHero(s, scenario, ctx); expectGuide(s, scenario, ctx); expectVideos(s, scenario, ctx);
  eq(s.weather, weatherFor(s), `${ctx}: weather values`);
  eq(s.days.length, 0, `${ctx}: no itinerary before planning`);
  eq(s.itineraryEmpty, 'Generate a trip plan to see your day-by-day itinerary.', ctx);
  eq(s.stays.length, 0, `${ctx}: no stays before planning`); eq(s.stayNote, 'Generate a trip plan to find places to stay.', ctx);
  eq(s.form.start, tomorrow, `${ctx}: start date defaults to tomorrow`);
  eq(s.form.days, '3', `${ctx}: default day count`);
  ok(!s.form.submitDisabled && s.form.busy !== 'true', `${ctx}: form idle`);
};
const dayLabel = date => { const d = new Date(`${date}T00:00:00Z`); return `${['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'][d.getUTCDay()]}, ${d.getUTCDate()} ${['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'][d.getUTCMonth()]}`; };
const CHIPS = { rain1: 'under 1 mm rain', rain2: 'Rain · 12 mm', rain3: '2 mm rain', uncertain: 'Forecast less certain', none: 'No forecast data for this day' };
const expectPlanned = (s, scenario, days, ctx, width) => {
  invariants(s, width, ctx);
  eq(s.days.length, days, `${ctx}: ${days} days`);
  eq(s.days.map(d => d.open), s.days.map((_, i) => i === 0), `${ctx}: only day 1 is open`);
  eq(s.days.map(d => d.label), s.days.map((_, i) => `Day ${i + 1} · ${dayLabel(plus(tomorrow, i))}`), `${ctx}: day labels`);
  eq(s.days.map(d => d.walk), s.days.map((_, i) => `${(3.1 + i).toFixed(1)} km walk`), `${ctx}: walk distances`);
  eq(s.days.map(d => d.stops), [4, 3, 3, 0, 0].slice(0, days), `${ctx}: stops per day (days 4-5 of the fixture have none)`);
  // Forecast chips.
  const absent = ['forecast-absent', 'fallback'].includes(scenario);
  const expectedChips = s.days.map((_, i) => {
    if (absent) return [CHIPS.none];
    if (scenario === 'forecast-uncertain') return [CHIPS.rain3, CHIPS.uncertain];
    return [[CHIPS.rain1], [CHIPS.rain2], [CHIPS.rain3, CHIPS.uncertain]][i] || [CHIPS.none];
  });
  eq(s.days.map(d => d.chips), expectedChips, `${ctx}: forecast chips`);
  // Stop photos: only the Belém Tower stop has one; photos-missing and fallback remove it.
  const photoDays = ['photos-missing', 'fallback'].includes(scenario) ? [0, 0, 0] : [1, 1, 0];
  eq(s.days.map(d => d.photos), [...photoDays, 0, 0].slice(0, days), `${ctx}: stop photos per day`);
  eq(s.days.map(d => d.directions), [true, true, true, false, false].slice(0, days), `${ctx}: walking directions on days that have stops`);
  // Stays.
  const stayCount = { 'stays-0': 0, 'stays-1': 1, 'stays-6': 6, 'stays-unavailable': 0, fallback: 0 }[scenario] ?? 4;
  eq(s.stays.length, stayCount, `${ctx}: stay count`);
  if (['stays-unavailable', 'fallback'].includes(scenario)) eq(s.stayNote, 'Nearby stays are temporarily unavailable. You can still search Booking.com.', ctx);
  else if (stayCount === 0) eq(s.stayNote, 'No matching places to stay were returned. Check prices to search for accommodation.', ctx);
  else ok(!s.stayNote, `${ctx}: no stay note when stays are listed`);
  if (scenario === 'stays-1') { eq(s.stays[0].name, 'Fixture Boutique Hotel', ctx); eq(s.stays[0].stars, '4-star property', ctx); ok(s.stays[0].website, ctx); }
  if (scenario === 'stays-6') {
    ok(s.stays.some(x => !x.stars) && s.stays.some(x => !x.website), `${ctx}: includes no-star and no-website stays`);
    ok(s.stays.some(x => x.name.length > 60), `${ctx}: includes a long label`);
    eq(s.stays.slice(4).map(x => x.name), ['Fixture Riverside Hotel', 'Fixture Garden Hostel'], ctx);
  }
  if (scenario === 'default') eq(s.stays.map(x => x.name).slice(0, 3), ['Fixture Boutique Hotel', 'Fixture Hostel', 'Fixture Guest House'], ctx);
  // Booking stays usable in every variant, with the destination and dates.
  const booking = new URL(s.booking);
  eq(booking.searchParams.get('checkin'), tomorrow, `${ctx}: booking check-in`);
  eq(booking.searchParams.get('checkout'), plus(tomorrow, days), `${ctx}: booking check-out`);
  eq(booking.searchParams.get('ss'), `${s.title}, ${s.country}`, `${ctx}: booking search names the destination and country`);
  ok(s.exports.ics && s.exports.kml, `${ctx}: calendar and map export links`);
  eq(s.form.days, String(days), ctx); eq(s.tripError, null, `${ctx}: no error`);
};

const started = {};
// ====================================================================================================
try {
  console.log(`Environment: ${b.product}; screenshots: ${output.pathname}`);
  const previews = {};
  for (const name of VARIANTS) previews[name] = started[name] = await startPreview({ scenario: name });
  const slowPreview = started.slow = await startPreview({ scenario: 'default', slow: true });
  const base = previews.default;

  // ---- A. The 12 named data variants: initial and planned ------------------------------------------------------
  for (const scenario of VARIANTS) {
    const { url } = previews[scenario];
    for (const [width, height] of WIDTHS) {
      b.state.imageMode = 'synthetic';
      await b.viewport(width, height);
      const ctx = `${scenario}@${width}`;
      await load(`${url}/?scenario=${scenario}&t=${Date.now()}`);
      const initial = await read();
      eq(initial.title, 'Lisbon', `${ctx}: default destination`);
      expectInitial(initial, scenario, `${ctx} initial`, width);
      await shoot(`variants/${scenario}/${width}-initial`, { viewport: scenario === 'default' });
      await plan(3);
      const planned = await read();
      expectPlanned(planned, scenario, 3, `${ctx} planned 3d`, width);
      expectGuide(planned, scenario, `${ctx} planned`);
      expectVideos(planned, scenario, `${ctx} planned`);
      await shoot(`variants/${scenario}/${width}-planned-3d`, { viewport: scenario === 'default' });
      if (['videos-10', 'videos-1'].includes(scenario) && scenario === 'videos-10') {
        await evaluate("document.querySelector('[data-video-expand]').click()");
        await until("document.querySelector('[data-video-expand]').getAttribute('aria-expanded') === 'true'");
        const expanded = await read();
        eq(expanded.videoShown, 10, `${ctx}: all ten videos after expanding`); eq(expanded.expandButton.label, 'Show fewer videos', ctx);
        await shoot(`variants/${scenario}/${width}-planned-3d-videos-expanded`);
      }
      if (scenario === 'default') {
        // Five days: days 4 and 5 have no forecast (preview.md); regeneration reopens day 1 only.
        await plan(5);
        const five = await read();
        expectPlanned(five, scenario, 5, `${ctx} planned 5d`, width);
        eq(five.days.slice(3).map(d => d.chips), [[CHIPS.none], [CHIPS.none]], `${ctx}: days 4-5 have no forecast`);
        await shoot(`variants/${scenario}/${width}-planned-5d`);
        // Third-party image failure: stop photos and video thumbnails answer with a network error.
        b.state.imageMode = 'block';
        await load(`${url}/?images-blocked=${Date.now()}`);
        await plan(3);
        await evaluate("document.querySelectorAll('.itinerary-day').forEach(d => { d.open = true; })"); // lazy photos in closed days load only when opened
        await until("document.querySelectorAll('.itinerary-stop-media-failed').length === 2", 8000);
        await evaluate("document.querySelector('#videos').scrollIntoView({ behavior: 'instant' })"); // lazy thumbnails fail when they come into view
        await until("document.querySelectorAll('.video-thumbnail-failed').length === 4", 8000);
        const blocked = await read();
        eq(blocked.days.map(d => d.failedPhotos).slice(0, 2), [1, 1], `${ctx}: broken stop photos fall back to the placeholder and hide their credit`);
        eq(blocked.failedThumbs, 4, `${ctx}: the four visible video thumbnails fall back`);
        ok(blocked.videoTotal === 6 && blocked.overflow <= width, ctx);
        await shoot(`variants/${scenario}/${width}-planned-3d-third-party-images-failed`);
        b.state.imageMode = 'synthetic';
      }
      record.push({ scenario, width, states: ['initial', 'planned-3d'] });
      console.log(`PASS ${ctx}: initial + planned assertions`);
    }
  }

  // ---- B. The scenario table (default preview) --------------------------------------------------------------------
  const scenarioTable = async (width, height) => {
    const { url } = base; const ctx = width;
    b.state.imageMode = 'synthetic';
    await b.viewport(width, height);
    // Initial Lisbon: covered by the variants; here the fixed bottom bar and the HTMX search flows.
    await load(`${url}/?table=${Date.now()}`);
    // Porto / empty videos.
    await search('Porto, Portugal', 'Porto');
    let s = await read();
    expectInitial(s, 'default', `porto@${ctx} initial`, width); eq(s.videoTotal, 0, 'Porto has no videos'); eq(s.hero.state, 'ready', 'Porto has a photo');
    eq(s.weather[0], '22°C Partly cloudy', 'Porto condition');
    await shoot(`table/porto-empty-videos/${width}-initial`);
    await plan(3); s = await read(); expectPlanned(s, 'default', 3, `porto@${ctx} planned`, width); expectVideos(s, 'default', 'porto planned');
    await shoot(`table/porto-empty-videos/${width}-planned-3d`);
    // Coimbra / replacement videos, no photo.
    await search('Coimbra, Portugal', 'Coimbra');
    s = await read(); expectInitial(s, 'default', `coimbra@${ctx} initial`, width); eq(s.videoTotal, 6, 'Coimbra shows replacement videos'); eq(s.hero.state, 'unavailable', 'Coimbra has no photo');
    eq(s.days.length, 0, 'the previous plan is cleared'); ok(!/Belém/.test(await evaluate('document.body.textContent')), 'no stale itinerary');
    await shoot(`table/coimbra-no-photo/${width}-initial`);
    await plan(3); s = await read(); expectPlanned(s, 'default', 3, `coimbra@${ctx} planned`, width);
    await shoot(`table/coimbra-no-photo/${width}-planned-3d`);
    // Paris, France / Paris, Texas by search.
    await search('Paris', 'Paris'); s = await read();
    eq(s.country, 'fr', 'Paris, France'); expectInitial(s, 'default', `paris@${ctx} initial`, width); eq(s.hero.state, 'ready', 'Paris photo');
    const parisSrc = s.hero.src;
    await shoot(`table/paris-france/${width}-initial`);
    await plan(3); s = await read(); expectPlanned(s, 'default', 3, `paris@${ctx} planned`, width); await shoot(`table/paris-france/${width}-planned-3d`);
    await search('Paris, Texas', 'Paris'); s = await read();
    eq(s.country, 'us', 'Paris, Texas'); expectInitial(s, 'default', `paris-texas@${ctx} initial`, width); eq(s.hero.state, 'ready', 'Paris, Texas photo');
    ok(s.hero.src !== parisSrc, 'the two Parises have different photographs'); eq(s.guide.state, 'missing', "Paris, Texas never shows Paris, France's guide");
    await shoot(`table/paris-texas/${width}-initial`);
    await plan(3); s = await read(); expectPlanned(s, 'default', 3, `paris-texas@${ctx} planned`, width); await shoot(`table/paris-texas/${width}-planned-3d`);
    // Shared pages for both Parises (also the /trip/... entries of the table).
    for (const [slug, country] of [['paris-fr', 'fr'], ['paris-us', 'us']]) {
      await load(`${url}/trip/${slug}`); s = await read();
      eq(s.country, country, slug); expectInitial(s, 'default', `${slug}@${ctx} shared`, width);
      await shoot(`table/shared-${slug}/${width}-initial`);
    }
    // Search failure keeps the current destination and says why.
    await load(`${url}/?failure=${Date.now()}`);
    await evaluate("document.querySelector('#city_name').value = 'error'; document.querySelector('#destination-search').requestSubmit()");
    await until("document.querySelector('#destination-search-error')?.textContent.trim().length > 0");
    s = await read();
    eq(s.searchError, 'We could not load that destination. Check the city name and try again.', 'search failure message');
    eq(s.title, 'Lisbon', 'search failure keeps the current destination'); invariants(s, width, `search-failure@${ctx}`);
    ok(await evaluate("document.querySelector('#destination-search-error').getAttribute('role') === 'alert'"), 'the failure is an alert');
    await shoot(`table/search-failure/${width}-error`, { viewport: true });
    // Shared three-day plan.
    await load(`${url}/trip/lisbon-pt?days=3&from=${tomorrow}`); s = await read();
    expectPlanned(s, 'default', 3, `shared-3d@${ctx}`, width); eq(s.form.start, tomorrow, 'shared page keeps the start date'); eq(s.url, `/trip/lisbon-pt?days=3&from=${tomorrow}`, 'shared URL');
    await shoot(`table/shared-three-day-plan/${width}-planned-3d`);
    // Calendar and map downloads: the response itself, then the link area on the page.
    const ics = await fetch(`${url}/trip/lisbon-pt.ics?days=1&from=${tomorrow}`); const icsBody = await ics.text();
    has(ics.headers.get('content-type'), /^text\/calendar/, 'ICS content type'); has(ics.headers.get('content-disposition'), /attachment; filename="lisbon-pt\./, 'ICS is an attachment');
    ok(icsBody.startsWith('BEGIN:VCALENDAR'), 'ICS starts a calendar'); eq((icsBody.match(/BEGIN:VEVENT/g) || []).length, 4, 'ICS has one event per day-1 stop');
    has(icsBody, new RegExp(`DTSTART:${tomorrow.replaceAll('-', '')}T100000Z`), 'ICS first event date'); has(icsBody, /SUMMARY:Belém Tower/, 'ICS names the first stop');
    const kml = await fetch(`${url}/trip/lisbon-pt.kml?days=1&from=${tomorrow}`); const kmlBody = await kml.text();
    has(kml.headers.get('content-type'), /^application\/vnd.google-earth.kml\+xml/, 'KML content type'); eq((kmlBody.match(/<Placemark>/g) || []).length, 4, 'KML has one point per day-1 stop');
    has(kmlBody, /<coordinates>-9.216,38.6916<\/coordinates>/, 'KML first coordinates');
    await load(`${url}/trip/lisbon-pt?days=1&from=${tomorrow}`); s = await read();
    has(s.exports.ics, /\.ics\?/, 'planned page links the calendar export'); has(s.exports.kml, /\.kml\?/, 'planned page links the map export');
    eq((await fetch(new URL(s.exports.ics.replaceAll('&amp;', '&'), url))).status, 200, 'linked ICS downloads'); eq((await fetch(new URL(s.exports.kml.replaceAll('&amp;', '&'), url))).status, 200, 'linked KML downloads');
    await evaluate("document.querySelector('#itinerary').scrollIntoView({ behavior: 'instant' })");
    await write(`table/calendar-and-map-downloads/${width}-export-links.webp`, Buffer.from((await call('Page.captureScreenshot', { format: 'webp', quality: 80 })).data, 'base64'));
    // Plan errors: 500 (four days) and 400 (days=0 sent directly to /plan).
    await load(`${url}/?errors=${Date.now()}`);
    await evaluate("document.querySelector('.trip-form').elements.days.value = '4'; document.querySelector('.trip-form').requestSubmit()");
    await until("!!document.querySelector('.trip-error')"); s = await read();
    eq(s.tripError, 'We could not plan your trip right now. Please try again in a moment.', '500 message'); eq(s.days.length, 0, '500 clears the itinerary'); eq(s.stays.length, 0, '500 clears stays'); eq(s.form.days, '4', '500 keeps the submitted form value');
    eq(s.itineraryNote, 'No itinerary is currently shown. Resolve the form error and try again.', '500 itinerary note'); invariants(s, width, `plan-500@${ctx}`);
    await shoot(`table/plan-error-500/${width}-error`);
    await load(`${url}/?errors=${Date.now()}`);
    await evaluate("htmx.ajax('GET', '/plan?city=Lisbon&country=pt&lat=38.7223&lon=-9.1393&start=" + tomorrow + "&days=0', { target: '#trip-card', swap: 'outerHTML' })");
    await until("!!document.querySelector('.trip-error')"); s = await read();
    ok(s.tripError && s.tripError.length > 10, `400 shows a message (${s.tripError})`); eq(s.days.length, 0, '400 clears the itinerary'); eq(s.stays.length, 0, '400 clears stays'); invariants(s, width, `plan-400@${ctx}`);
    await shoot(`table/plan-error-400/${width}-error`);
    // Local map identity.
    await call('Page.navigate', { url: base.mapUrl }); await until("document.readyState === 'complete' && !!document.querySelector('figure')");
    const mapTitle = await evaluate('document.title'); eq(mapTitle, 'FIXTURE MAP — NOT LIVE WAZE', 'the map fixture identifies itself');
    has(await evaluate('document.body.textContent'), /This is local test content\. It is not a live map provider\./, 'map fixture text');
    await write(`table/local-map-identity/${width}-map.webp`, Buffer.from((await call('Page.captureScreenshot', { format: 'webp', quality: 80 })).data, 'base64'));
  };
  for (const [width, height] of WIDTHS) { await scenarioTable(width, height); console.log(`PASS ${width}px: scenario table (Porto, Coimbra, both Parises, shared pages, search failure, exports, 400/500, map identity)`); }

  // ---- C. Destination photo fixtures and city guide fixtures ---------------------------------------------------------------
  const FIXTURES = [
    ['Nophoto', 'nophoto-completed-lookup-none'], ['Photoerror', 'photoerror-provider-failure'], ['Brokenimage', 'brokenimage-image-404'],
    ['Longguide', 'longguide-awkward-content'], ['Tokyo', 'tokyo-reviewed-guide-no-photo'], ['Tavira', 'tavira-reviewed-guide-no-photo']
  ];
  for (const [width, height] of WIDTHS) {
    await b.viewport(width, height);
    for (const [term, dir] of FIXTURES) {
      const ctx = `${term}@${width}`;
      await load(`${base.url}/?city_name=${term}&fixture=${Date.now()}`);
      if (term === 'Brokenimage') await until("document.querySelector('.destination-photo').dataset.photoState === 'failed'", 10000);
      const s = await read();
      eq(s.title, term, ctx); invariants(s, width, ctx); expectHero(s, 'default', ctx); expectGuide(s, 'default', ctx); expectVideos(s, 'default', ctx);
      if (term === 'Brokenimage') { ok(s.hero.imgPresent && !s.hero.imgShown && !s.hero.creditShown, `${ctx}: the failed image and its credit are hidden`); ok(s.hero.fallbackShown, `${ctx}: unavailable text revealed by the image error event`); }
      if (term === 'Photoerror') { await plan(3); expectPlanned(await read(), 'default', 3, `${ctx} planned (the rest of the page works after a photo provider failure)`, width); await shoot(`fixtures/${dir}/${width}-planned-3d`); }
      else if (term === 'Brokenimage') { await shoot(`fixtures/${dir}/${width}-initial`); await plan(3); expectPlanned(await read(), 'default', 3, `${ctx} planned`, width); await shoot(`fixtures/${dir}/${width}-planned-3d`); }
      else await shoot(`fixtures/${dir}/${width}-initial`);
      if (term === 'Longguide') { const w = await evaluate("[...document.querySelectorAll('.guide-paragraph')].some(p => p.scrollWidth > p.clientWidth + 1)"); ok(!w, `${ctx}: the unbroken 93-character word wraps`); }
      console.log(`PASS ${ctx}: fixture assertions`);
    }
  }
  // Photo unavailable next to a reviewed guide, and the reverse, are covered by Nophoto/Tokyo above; the
  // "no guide anywhere" scenario is the fallback variant in section A.

  // ---- D. Slow preview: the loading states are visible -----------------------------------------------------------------------
  for (const [width, height] of WIDTHS) {
    await b.viewport(width, height);
    await load(`${slowPreview.url}/?slow=${Date.now()}`);
    await evaluate("document.querySelector('#city_name').value = 'Porto, Portugal'; document.querySelector('#destination-search').requestSubmit()");
    await until("document.querySelector('#destination-search').classList.contains('htmx-request') || document.querySelector('#destination-search').getAttribute('aria-busy') === 'true'", 3000);
    await sleep(450);
    const busySearch = await evaluate("({ busy: document.querySelector('#destination-search').getAttribute('aria-busy'), spinner: getComputedStyle(document.querySelector('#destination-search-spinner')).opacity, title: document.querySelector('.destination-title').textContent.trim() })");
    eq(busySearch.busy, 'true', `slow@${width}: the search is marked busy`); eq(busySearch.spinner, '1', `slow@${width}: the spinner is visible`); eq(busySearch.title, 'Lisbon', `slow@${width}: the old destination stays until the response`);
    await write(`slow/${width}-search-loading.webp`, Buffer.from((await call('Page.captureScreenshot', { format: 'webp', quality: 80 })).data, 'base64'));
    await until("document.querySelector('.destination-title').textContent.trim() === 'Porto'", 6000); await settleHero();
    await evaluate("document.querySelector('.trip-form').requestSubmit()");
    await until("document.querySelector('.trip-submit').disabled", 3000); await sleep(450);
    const busyPlan = await evaluate("({ status: document.querySelector('#trip-status').textContent, busy: document.querySelector('.trip-form').getAttribute('aria-busy'), disabled: document.querySelector('.trip-submit').disabled, spinner: getComputedStyle(document.querySelector('#trip-spinner')).opacity })");
    eq(busyPlan, { status: 'Planning your trip…', busy: 'true', disabled: true, spinner: '1' }, `slow@${width}: planning shows busy, disabled submit and a visible spinner`);
    await evaluate("document.querySelector('#trip-card').scrollIntoView({ block: 'center', behavior: 'instant' })");
    await write(`slow/${width}-plan-loading.webp`, Buffer.from((await call('Page.captureScreenshot', { format: 'webp', quality: 80 })).data, 'base64'));
    await until("document.querySelector('#trip-status').textContent === 'Your itinerary has been updated.' && document.querySelectorAll('.itinerary-day').length === 3 && !document.querySelector('.trip-submit').disabled", 8000);
    eq((await read()).tripStatus, 'Your itinerary has been updated.', `slow@${width}: completion is announced`);
    console.log(`PASS ${width}px: slow preview loading states`);
  }

  assert.deepEqual(b.errors, [], 'Unexpected JavaScript exceptions');
  await writeFile(new URL('scenario-results.json', output), JSON.stringify({ browser: b.product, generated: new Date().toISOString(), tomorrow, assertions: checks, scenarios: record, screenshots: files, errors: b.errors }, null, 2));
  console.log(`\nPASS: ${checks} assertions across ${files.length} screenshots, no JavaScript exceptions`);
} catch (error) {
  console.error(`FAIL: ${error.stack || error.message || JSON.stringify(error)}`);
  process.exitCode = 1;
} finally {
  for (const p of Object.values(started)) p.stop();
  await b.close();
  process.exit(process.exitCode || 0);
}
