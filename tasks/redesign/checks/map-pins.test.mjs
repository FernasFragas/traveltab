import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import vm from 'node:vm';

const dayplan = readFileSync(new URL('../../../public/redesign/map-dayplan.js', import.meta.url), 'utf8');
const panel = readFileSync(new URL('../../../public/redesign/map-panel.js', import.meta.url), 'utf8');

test('notable categories use icons, writers use clusters, and selection shows a lazy photo callout', () => {
  const layers = new Map();
  const listeners = new Map();
  const calls = [];
  const element = tag => ({ tagName: tag.toUpperCase(), children: [], attrs: {}, style: {},
    className: '', textContent: '', setAttribute(name, value) { this.attrs[name] = value; },
    appendChild(child) { this.children.push(child); return child; }, addEventListener() {} });
  const document = {
    body: { addEventListener() {} }, createElement: element, querySelector: () => null,
    addEventListener(name, cb) { listeners.set(name, cb); }
  };
  const window = { TravelMap: { registerLayer(name, value) { layers.set(name, value); } } };
  vm.runInNewContext(panel, { document, window, Map, Set });
  const kinds = ['viewpoint', 'museum', 'restaurant', 'garden', 'trail', 'church', 'unknown'];
  const features = kinds.map((kind, index) => ({ geometry: { coordinates: [-16.9, 32.6] },
    properties: { qid: 'Q' + index, name: kind, kind, writer_count: 0, sitelinks: 7 - index } }));
  features.push({ geometry: { coordinates: [-16.91, 32.61] },
    properties: { qid: 'Qwriter', name: 'Writer place', kind: 'viewpoint', writer_count: 2, sitelinks: 8 } });
  const context = { features, draw: {
    markers(id, markers) { calls.push({ id, markers }); },
    clusters(id, points, style, onClick) { calls.push({ id, points, style, onClick }); },
    clear(id) { calls.push({ clear: id }); }
  } };
  layers.get('notable').add({}, context);
  layers.get('writers').add({}, context);
  const notable = calls.find(call => call.id === 'notable').markers;
  assert.deepEqual(notable.map(marker => marker.element.children[0].className),
    ['bi bi-binoculars', 'bi bi-bank', 'bi bi-cup-hot', 'bi bi-tree', 'bi bi-signpost-split', 'bi bi-building', 'bi bi-geo-alt']);
  const writers = calls.find(call => call.id === 'writers');
  assert.equal(writers.points.length, 1);
  assert.equal(writers.points[0].properties.writer_count, 2);
  assert.ok(writers.style.labelled.includes('Qwriter'));
  listeners.get('travelmap:place-selected')({ detail: { place: {
    geometry: { coordinates: [-16.91, 32.61] }, properties: { qid: 'Qwriter', name: 'Writer place', photo_url: '/photo.jpg' }
  } } });
  const photo = calls.find(call => call.id === 'selected-callout').markers[0].element.children[0];
  assert.equal(photo.tagName, 'IMG');
  assert.equal(photo.loading, 'lazy');
});

test('day plan draws orange numbered stops, a dashed route and lazy photo callouts', () => {
  const listeners = new Map();
  const drawn = { markers: [], lines: [] };
  let layer;
  const element = tag => ({ tagName: tag.toUpperCase(), children: [], style: {}, attrs: {},
    setAttribute(name, value) { this.attrs[name] = value; },
    appendChild(child) { this.children.push(child); return child; },
    addEventListener() {} });
  const stops = [
    { dataset: { name: 'Tower', lat: '32.65', lon: '-16.9', qid: 'Q1' },
      querySelector: selector => selector === '.itinerary-stop-photo' ? { src: '/tower.jpg' } : null },
    { dataset: { name: 'Garden', lat: '32.66', lon: '-16.91', qid: 'Q2' }, querySelector: () => null }
  ];
  const document = {
    body: { addEventListener(name, cb) { listeners.set(name, cb); } },
    createElement: element,
    querySelector: selector => selector === '[data-writers-map]' ? {} : null,
    querySelectorAll: selector => selector === '[data-plan-day]' ? [{ dataset: { planDay: '1' }, querySelectorAll: () => stops }] : [],
    addEventListener(name, cb) { listeners.set(name, cb); }
  };
  const window = { TravelMap: {
    registerLayer(_name, value) { layer = value; },
    updatePlan(days) { listeners.get('travelmap:plan-updated')({ detail: { days } }); }
  } };
  vm.runInNewContext(dayplan, { document, window, JSON });
  layer.add({}, { draw: {
    clear() {}, markers(_id, items) { drawn.markers = items; },
    line(id, coords, style) { drawn.lines.push({ id, coords, style }); }, popup() {}, control() {}
  }, features: [], selectPlace() {} });
  assert.deepEqual(Array.from(drawn.markers, item => item.element.style.background), ['#a56824', '#a56824']);
  assert.equal(drawn.markers[0].element.children[0].tagName, 'IMG');
  assert.equal(drawn.markers[0].element.children[0].loading, 'lazy');
  assert.deepEqual(JSON.parse(JSON.stringify(drawn.lines[0].coords)), [[-16.9, 32.65], [-16.91, 32.66]]);
  assert.deepEqual(JSON.parse(JSON.stringify(drawn.lines[0].style)), { color: '#a56824', width: 3, dash: [1, 2] });
});

test('pin icons, plan numbers, cluster counts and labels have sufficient contrast', () => {
 const luminance = hex => {
  const values = hex.slice(1).match(/../g).map(value => parseInt(value, 16) / 255).map(value => value <= .04045 ? value / 12.92 : ((value + .055) / 1.055) ** 2.4);
  return values[0] * .2126 + values[1] * .7152 + values[2] * .0722;
 };
 const contrast = (a, b) => { const x = luminance(a), y = luminance(b); return (Math.max(x, y) + .05) / (Math.min(x, y) + .05); };
 for (const color of ['#cf795b', '#459b6b', '#a582d4', '#a56824']) {
  assert.ok(contrast(color, '#202a32') >= 3, color + ' against map');
  assert.ok(contrast(color, '#ffffff') >= 3, color + ' against icon');
 }
 assert.ok(contrast('#a56824', '#ffffff') >= 4.5, 'plan numbers');
 assert.ok(contrast('#9197a1', '#202a32') >= 4.5, 'cluster count');
 assert.ok(contrast('#202a32', '#ffffff') >= 4.5, 'label text');
});
