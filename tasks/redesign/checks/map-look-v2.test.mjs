import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {test} from 'node:test';
const read = path => readFileSync(new URL('../../../' + path, import.meta.url), 'utf8');
test('brand has accessible two-line wordmark and search submits without a button', () => {
 const html = read('views/index.go.tpl');
 assert.match(html, /aria-label="TravelTabByPatronFragas"/);
 assert.match(html, /tt-brand-peaks/);
 assert.match(html, />TravelTab<.*>ByPatronFragas</s);
 assert.doesNotMatch(html, /tt-search-submit|bi-signpost-split-fill/);
});
test('saved places have a header list and polite undo toast', () => {
 assert.match(read('views/index.go.tpl'), /aria-label="Saved places"/);
 const js = read('public/redesign/map-saved.js');
 assert.match(js, /No saved places yet/);
 assert.match(js, /Added to your saved places/);
 assert.match(js, /data-saved-undo/);
});
test('panel presents story and cards without tabs or peek', () => {
 const html = read('views/place_panel.go.tpl');
 assert.doesNotMatch(html, /role="tab|map-panel-tabs/);
 assert.match(html, /From our writers/);
 assert.match(html, /google.com\/maps\/dir/);
 assert.match(html, /map-panel-bookmark/);
 assert.match(html, /map-panel-facts/);
 assert.doesNotMatch(read('views/map_card.go.tpl'), /data-map-panel-peek/);
});
test('map uses card layout, full-map link and updated palette', () => {
 assert.match(read('views/map_card.go.tpl'), /View full map/);
 assert.match(read('public/redesign/map-pins.css'), /map-pin-nature/);
 assert.match(read('public/redesign/writers-map.js'), /Reset bearing/);
 assert.match(read('public/redesign/map-dayplan.js'), /#a56824/);
});

test('v3 chrome groups zoom, exposes back and uses compact attribution', () => {
 const js = read('public/redesign/writers-map.js');
 assert.match(js, /travelmap-back/);
 assert.match(js, /travelmap-zoom/);
 assert.match(js, /bi-crosshair/);
 assert.match(js, /compact: true/);
 assert.doesNotMatch(js, /TravelMapLayers.createMenu/);
});

// As a traveller on a phone, I want the small map uncluttered: credits one tap away, no duplicate name.
test('compact attribution starts collapsed once the map loads', () => {
 const js = read('public/redesign/writers-map.js');
 assert.match(js, /function collapseAttribution\(/);
 // Collapse like MapLibre's own minimise: drop the class, keep <details open>, so its toggle keeps working.
 assert.match(js, /classList\.remove\('maplibregl-compact-show'\)/);
 assert.doesNotMatch(js, /removeAttribute\('open'\)/);
 assert.match(js, /map\.on\('load', function \(\) \{[^]*?collapseAttribution\(target\)/);
});
test('the selected pin hides its name label because the callout shows it', () => {
 assert.match(read('public/redesign/map-pins.css'), /\.map-pin\.is-selected \.map-pin-label\s*\{\s*display:\s*none/);
});
