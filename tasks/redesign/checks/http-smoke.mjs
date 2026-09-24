#!/usr/bin/env node
// Run against the default fixture scenario. All requests have bounded timeouts.
import assert from 'node:assert/strict';

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
console.log('preview HTTP smoke passed: initial data, 1/3/5-day plans, response roots/OOB, 400/500, Porto/Coimbra, shared page, ICS/KML, local map');
