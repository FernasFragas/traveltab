#!/usr/bin/env node
// Run against the default fixture scenario. All requests have bounded timeouts.
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';

const base = process.env.PREVIEW_URL || 'http://127.0.0.1:8087';
const mapOrigin = process.env.PREVIEW_MAP_URL || 'http://127.0.0.1:8088';
const tomorrow = new Date();
tomorrow.setUTCDate(tomorrow.getUTCDate() + 1);
const start = tomorrow.toISOString().slice(0, 10);
const headers = {'HX-Request': 'true'};
const request = async (path, status = 200, options = {}) => {
  const res = await fetch(new URL(path, base), {...options, signal: AbortSignal.timeout(15000)});
  const body = await res.text();
  assert.equal(res.status, status, `${path}: ${res.status}: ${body.slice(0, 200)}`);
  return {res, body};
};
const voidElements = new Set(['area', 'base', 'br', 'col', 'embed', 'hr', 'img', 'input', 'link', 'meta', 'param', 'source', 'track', 'wbr']);
// Check the fragment's actual root structure, not IDs appearing somewhere in nested markup.
function planRoots(body) {
  const stack = [], roots = [];
  const tokens = body.match(/<!--[\s\S]*?-->|<\/?[a-z][^>"']*(?:(?:"[^"]*"|'[^']*')[^>"']*)*>/gi) || [];
  for (const token of tokens) {
    if (token.startsWith('<!--')) continue;
    const name = token.match(/^<\/?([a-z][\w-]*)/i)[1].toLowerCase();
    if (token.startsWith('</')) {
      assert.equal(stack.pop(), name, `unbalanced ${token}`);
      continue;
    }
    if (!stack.length) roots.push(token);
    if (!voidElements.has(name) && !token.endsWith('/>')) stack.push(name);
  }
  assert.equal(stack.length, 0, 'fragment must close every root');
  assert.equal(roots.length, 3, 'plan response must contain exactly three sibling roots');
  for (const [index, id] of ['trip-card', 'itinerary', 'stays'].entries()) {
    assert.match(roots[index], new RegExp(`\\bid="${id}"`));
    if (index) assert.match(roots[index], /\bhx-swap-oob="outerHTML"/);
    else assert.doesNotMatch(roots[index], /\bhx-swap-oob=/);
  }
}
const initial = (await request('/')).body;
assert.match(initial, /Lisbon/);
assert.match(initial, /name="city" value="Lisbon"/);
assert.match(initial, /name="country" value="pt"/);
assert.match(initial, /src="\/images\/destinations\/lisbon-alfama.jpg"/);
assert.match(initial, /dQw4w9WgXcQ/);
assert.ok(initial.includes(`src="${new URL('/', mapOrigin).href}"`), 'map iframe must use configured fixture origin');

const planURL = days => `/plan?city=Lisbon&country=pt&lat=38.7223&lon=-9.1393&start=${start}&days=${days}`;
for (const days of [1, 3, 5]) {
  const {res, body} = await request(planURL(days), 200, {headers});
  planRoots(body);
  assert.match(body, /Fixture Boutique Hotel/);
  assert.match(body, /A Very Long Destination Attraction Name/);
  assert.equal(res.headers.get('HX-Push-Url'), `/trip/lisbon-pt?days=${days}&from=${start}`);
  assert.equal((body.match(/class="itinerary-day"/g) || []).length, days);
}
for (const [days, status, message] of [[0, 400, /Pick between 1 and 5 days/], [4, 500, /could not plan your trip/]]) {
  const {body} = await request(planURL(days), status, {headers});
  planRoots(body);
  assert.match(body, message);
  assert.doesNotMatch(body, /Fixture Boutique Hotel|Belém Tower/);
}
const searchFailure = await request('/process-form/?city_name=error', 500, {headers});
assert.equal(searchFailure.res.headers.get('HX-Retarget'), '#destination-search-error');
assert.match(searchFailure.body, /could not load that destination/);
const porto = (await request('/process-form/?city_name=Porto%2C%20Portugal', 200, {headers})).body;
assert.match(porto, /name="city" value="Porto"/);
assert.match(porto, /No travel videos are available for this destination right now\./);
assert.doesNotMatch(porto, /data-video-id=/);
const coimbra = (await request('/process-form/?city_name=Coimbra%2C%20Portugal', 200, {headers})).body;
assert.match(coimbra, /name="city" value="Coimbra"/);
assert.match(coimbra, /data-video-id="dQw4w9WgXcQ"/);
// Destination photos: the fixture photo source answers by resolved identity and never calls Wikimedia.
const photoBase = new URL('/fixture-photos/', mapOrigin).href;
const heroOf = body => body.match(/<img class="destination-image" src="([^"]+)"/)?.[1];
const photoState = body => body.match(/class="destination-photo" data-photo-state="([a-z]+)"/)?.[1];
assert.equal(photoState(initial), 'ready');
assert.equal(heroOf(initial), '/images/destinations/lisbon-alfama.jpg', 'Lisbon keeps its curated local photo');
const paris = (await request('/process-form/?city_name=Paris', 200, {headers})).body;
assert.equal(heroOf(paris), `${photoBase}paris.png`);
assert.match(paris, /alt="View of Paris"/);
assert.match(paris, /Photo: Fixture Photographer \(Paris\)/);
assert.match(paris, /name="country" value="fr"/);
assert.equal(photoState(paris), 'ready');
const texas = (await request('/process-form/?city_name=Paris%2C%20Texas', 200, {headers})).body;
assert.equal(heroOf(texas), `${photoBase}paris-texas.png`, 'A namesake city gets its own photo');
assert.match(texas, /name="country" value="us"/);
assert.equal(heroOf((await request('/trip/paris-fr')).body), `${photoBase}paris.png`);
assert.equal(heroOf((await request('/trip/paris-us')).body), `${photoBase}paris-texas.png`);
assert.equal(heroOf(porto), `${photoBase}porto.png`, 'Porto now has a fixture photo');
for (const [city, why] of [['Coimbra', 'no photo for an unlisted city'], ['Nophoto', 'completed lookup without a photo'], ['Photoerror', 'provider failure']]) {
  const body = (await request(`/process-form/?city_name=${city}`, 200, {headers})).body;
  assert.equal(photoState(body), 'unavailable', why);
  assert.equal(heroOf(body), undefined, why);
  assert.match(body, /Photo unavailable/);
  assert.doesNotMatch(body, /destination-photo-credit/);
  assert.match(body, /class="weather-metrics"/, `${why}: weather stays usable`);
  assert.match(body, /id="trip-card"/, `${why}: planning stays usable`);
}
const broken = (await request('/process-form/?city_name=Brokenimage', 200, {headers})).body;
assert.equal(heroOf(broken), `${photoBase}missing.png`);
const imageBytes = {};
for (const name of ['paris', 'paris-texas', 'porto']) {
  const res = await fetch(`${photoBase}${name}.png`, {signal: AbortSignal.timeout(15000)});
  assert.equal(res.status, 200);
  assert.equal(res.headers.get('content-type'), 'image/png');
  imageBytes[name] = Buffer.from(await res.arrayBuffer());
}
assert.ok(!imageBytes.paris.equals(imageBytes['paris-texas']) && !imageBytes.paris.equals(imageBytes.porto), 'fixture photos differ per city');
assert.equal((await fetch(`${photoBase}missing.png`, {signal: AbortSignal.timeout(15000)})).status, 404, 'the broken-image fixture must answer 404');

// City guides: a reviewed intro with its Wikivoyage attribution, or an explicit missing state.
// The expected revisions come from the shipped guides/guides.json, the file the site embeds.
const shippedGuides = JSON.parse(await readFile(new URL('../../../guides/guides.json', import.meta.url), 'utf8'));
const guideState = body => body.match(/class="guide-card" data-guide-state="([a-z]+)"/)?.[1];
const assertReviewedGuide = (body, key, context) => {
  const entry = shippedGuides[key];
  assert.ok(entry?.reviewed, `${context}: ${key} must be a reviewed entry`);
  assert.equal(guideState(body), 'reviewed', context);
  assert.equal((body.match(/class="guide-card"/g) || []).length, 1, `${context}: exactly one guide card`);
  assert.match(body, /Adapted from Wikivoyage/, context);
  assert.ok(body.includes(`revision ${entry.wikivoyage_revision}</a>`), `${context}: revision ${entry.wikivoyage_revision} is shown and linked`);
  assert.ok(body.includes(`oldid=${entry.wikivoyage_revision}`), `${context}: permalink to the reviewed revision`);
  assert.ok(body.includes('https://creativecommons.org/licenses/by-sa/4.0/'), `${context}: license link`);
  assert.match(body, /Wikivoyage contributors/, context);
  assert.match(body, new RegExp(`Checked against that revision on ${entry.reviewed_at}`), context);
  const overview = body.slice(body.indexOf('<section id="overview"'), body.indexOf('<section id="itinerary"'));
  assert.ok(overview.includes('class="guide-card"'), `${context}: the card is in the overview section`);
  for (const id of ['overview', 'itinerary', 'stays', 'videos']) assert.equal((body.match(new RegExp(`<section id="${id}"`, 'g')) || []).length, 1, `${context}: section ${id} once`);
  return entry;
};
const assertMissingGuide = (body, city, context) => {
  assert.equal(guideState(body), 'missing', context);
  assert.ok(body.includes(`We don't have a reviewed guide for ${city} yet.`), context);
  assert.ok(body.includes(`https://en.wikivoyage.org/w/index.php?search=${encodeURIComponent(city)}`), `${context}: Wikivoyage search link`);
  assert.doesNotMatch(body, /Adapted from Wikivoyage|guide-source/, `${context}: no attribution without a guide`);
};
const firstSentence = entry => entry.intro.split(/(?<=\.)\s/)[0].replaceAll("'", '&#39;');
const lisbonGuide = assertReviewedGuide(initial, 'lisbon-pt', 'initial Lisbon');
assert.ok(initial.includes(firstSentence(lisbonGuide)), 'the reviewed Lisbon text is rendered');
assertReviewedGuide(porto, 'porto-pt', 'HTMX Porto');
const parisGuide = assertReviewedGuide(paris, 'paris-fr', 'HTMX Paris, France');
assertMissingGuide(texas, 'Paris', 'HTMX Paris, Texas');
assert.ok(!texas.includes(firstSentence(parisGuide)), "Paris, Texas never receives Paris, France's guide");
assertReviewedGuide((await request('/trip/paris-fr')).body, 'paris-fr', 'shared Paris, France');
const sharedTexas = (await request('/trip/paris-us')).body;
assertMissingGuide(sharedTexas, 'Paris', 'shared Paris, Texas');
assert.ok(!sharedTexas.includes(firstSentence(parisGuide)), 'the shared Paris, Texas page has no Paris, France text');
assertMissingGuide(coimbra, 'Coimbra', 'HTMX Coimbra');
assertMissingGuide((await request('/process-form/?city_name=Nophoto', 200, {headers})).body, 'Nophoto', 'HTMX Nophoto');
const awkward = (await request('/process-form/?city_name=Longguide', 200, {headers})).body;
assert.equal(guideState(awkward), 'reviewed', 'awkward fixture guide');
assert.ok(awkward.includes('&lt;b&gt;markup characters&lt;/b&gt; &amp; an ampersand.') && !awkward.includes('<b>markup characters</b>'), 'guide text is escaped');
for (const days of [1, 3]) {
  assert.doesNotMatch((await request(planURL(days), 200, {headers})).body, /guide-card/, 'the planner response never carries the guide card');
}

const shared = (await request(`/trip/lisbon-pt?days=3&from=${start}`)).body;
assert.match(shared, /name="city" value="Lisbon"/);
assert.match(shared, /Fixture Boutique Hotel/);
assert.match(shared, new RegExp(`value="${start}"`));
for (const format of ['ics', 'kml']) {
  // Follow the generated download URL, including HTML attribute entity decoding.
  const link = shared.match(new RegExp(`href="([^\"]+\\.${format}\\?[^\"]+)"`));
  assert.ok(link, `${format} download link missing`);
  const {res, body} = await request(link[1].replaceAll('&amp;', '&'));
  assert.match(res.headers.get('content-disposition'), /attachment; filename="lisbon-pt\./);
  if (format === 'ics') {
    assert.match(res.headers.get('content-type'), /^text\/calendar/);
    assert.match(body, /BEGIN:VCALENDAR/);
    assert.match(body, new RegExp(`DTSTART:${start.replaceAll('-', '')}T100000Z`));
    assert.equal((body.match(/BEGIN:VEVENT/g) || []).length, 10);
    assert.match(body, /SUMMARY:Belém Tower/);
  } else {
    assert.match(res.headers.get('content-type'), /^application\/vnd.google-earth.kml\+xml/);
    assert.match(body, /<kml xmlns="http:\/\/www.opengis.net\/kml\/2.2">/);
    assert.equal((body.match(/<Placemark>/g) || []).length, 10);
    assert.match(body, /<coordinates>-9.216,38.6916<\/coordinates>/);
  }
}
const mapFixture = await request(mapOrigin);
assert.match(mapFixture.body, /FIXTURE MAP/);
assert.match(mapFixture.body, /not a live map provider/);
assert.match(mapFixture.res.headers.get('content-security-policy'), /style-src 'unsafe-inline'/);
console.log('preview HTTP smoke passed: initial data, 1/3/5-day plans, response roots/OOB, 400/500, Porto/Coimbra, destination photo states, city guide states, shared page, ICS/KML, local map');
