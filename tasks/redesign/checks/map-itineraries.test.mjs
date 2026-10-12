import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import vm from 'node:vm';

const source = readFileSync(new URL('../../../public/redesign/map-itineraries.js', import.meta.url), 'utf8');
const stylesheet = readFileSync(new URL('../../../public/redesign/map-itineraries.css', import.meta.url), 'utf8');
const features = [['Q1', 'Tower', 38.1, -9.1], ['Q2', 'Museum', 38.2, -9.2], ['Q3', 'Park', 38.3, -9.3]].map(([qid, name, lat, lon]) =>
  ({ geometry: { coordinates: [lon, lat] }, properties: { qid, name } }));
const routeA = { writer_host: 'a.example', post_url: 'https://a.example/trip/', post_title: 'Trip A', days: [{ qids: ['Q1', 'Q2'] }, { qids: ['Q3'] }] };
const routeB = { writer_host: 'b.example', post_url: 'https://b.example/roteiro/', post_title: 'Roteiro B', days: [{ qids: ['Q3', 'Q1'] }] };

test('registers one itinerary layer and does nothing without the core', () => {
  const app = browser(); app.run(); assert.deepEqual(app.registered, ['itineraries']);
  const absent = browser({ core: false }); absent.run(); assert.deepEqual(absent.registered, []);
});

test('draws numbered buttons and one styled route through context.draw', async () => {
  const app = browser({ itineraries: [routeA] }); app.run(); await app.mount();
  assert.deepEqual(app.ids(), ['itineraries']);
  assert.deepEqual(JSON.parse(JSON.stringify(app.pins().map(p => p.lngLat))), [[-9.1, 38.1], [-9.2, 38.2], [-9.3, 38.3]]);
  assert.deepEqual(JSON.parse(JSON.stringify(app.pins().map(p => p.element.textContent))), ['1', '2', '3']);
  assert.equal(app.pins()[0].element.tagName, 'BUTTON');
  assert.equal(app.pins()[0].element.className, 'itinerary-pin');
  assert.equal(app.pins()[0].title, 'Stop 1: Tower');
  assert.deepEqual(JSON.parse(JSON.stringify(app.line().coords)), [[-9.1, 38.1], [-9.2, 38.2], [-9.3, 38.3]]);
  assert.deepEqual(JSON.parse(JSON.stringify(app.line().style)), { color: '#b794f6', width: 3, dash: [6, 6] });
  assert.deepEqual(app.calls().filter(c => c !== 'clear'), ['line', 'markers']);
});

test('the itineraries legend dot matches the route line colour', () => {
  const css = readFileSync(new URL('../../../public/redesign/writers-map-layout.css', import.meta.url), 'utf8');
  assert.match(css, /\.writers-map-layer-itineraries \{ --layer-color: #b794f6; \}/);
});

test('a pin opens route details and a safe post link; switching closes them', async () => {
  const app = browser({ itineraries: [routeA, routeB] }); app.run(); await app.mount();
  app.pins()[0].element.click();
  assert.match(app.popup().text(), /Route by a\.example/);
  const link = app.popup().children.find(c => c.tagName === 'A');
  assert.equal(link.href, 'https://a.example/trip/');
  assert.equal(link.textContent, 'Trip A');
  assert.equal(link.attrs.rel, 'noopener noreferrer');
  app.picker().value = '1'; app.picker().change();
  assert.equal(app.popup(), undefined);
  assert.equal(app.pins().length, 2);
  assert.deepEqual(JSON.parse(JSON.stringify(app.line().coords)), [[-9.3, 38.3], [-9.1, 38.1]]);
});

test('a non-http URL gets no post link', async () => {
  const app = browser({ itineraries: [{ ...routeA, post_url: 'javascript:alert(1)' }] }); app.run(); await app.mount();
  app.pins()[0].element.click();
  assert.equal(app.popup().children.find(c => c.tagName === 'A'), undefined);
});

test('missing and invalid stops are skipped, and one stop has no line', async () => {
  const invalid = { geometry: { coordinates: [0, 0] }, properties: { qid: 'Q0', name: 'Invalid' } };
  const app = browser({ itineraries: [{ ...routeA, days: [{ qids: ['Q1', 'Q404', 'Q0'] }] }], features: [...features, invalid] });
  app.run(); await app.mount(); assert.equal(app.pins().length, 1); assert.equal(app.line(), undefined);
});

test('empty or failed route data draws nothing and leaves the chip hidden', async () => {
  for (const options of [{ itineraries: [] }, { fail: true }]) {
    const app = browser(options); app.run(); await app.mount();
    assert.equal(app.pins().length, 0);
    assert.equal(app.picker(), undefined);
    assert.equal('itineraries' in app.shell.dataset, false);
  }
});

test('the ready event marks the shell even when the layer is off', async () => {
  const app = browser({ itineraries: [routeA] }); app.run(); await app.ready();
  assert.equal(app.shell.dataset.itineraries, '');
});

test('one route has no picker; switching routes never duplicates markers, lines or the picker', async () => {
  const single = browser({ itineraries: [routeA] }); single.run(); await single.mount(); assert.equal(single.picker(), undefined);
  const app = browser({ itineraries: [routeA, routeB] }); app.run(); await app.mount();
  assert.deepEqual(app.picker().children.map(o => o.textContent), ['a.example: Trip A', 'b.example: Roteiro B']);
  for (const index of ['1', '0', '1']) {
    app.picker().value = index; app.picker().change();
    assert.equal(app.ids().length, 1);
    assert.equal(app.pins().length, index === '1' ? 2 : 3);
    assert.ok(app.line());
    assert.equal(app.slot.children.filter(c => c.tagName === 'SELECT').length, 1);
  }
});

test('disable clears every drawing and the popup; re-enable restores one route', async () => {
  const app = browser({ itineraries: [routeA, routeB] }); app.run(); await app.mount();
  app.pins()[0].element.click(); app.setEnabled(false);
  assert.equal(app.ids().length, 0);
  assert.equal(app.picker(), undefined);
  assert.equal(app.popup(), undefined);
  app.setEnabled(true); await app.settle();
  assert.equal(app.pins().length, 3);
  assert.ok(app.line());
  assert.equal(app.ids().length, 1);
});

function browser({ core = true, itineraries = [], features: places = features, fail = false } = {}) {
  const registered = [], listeners = new Map(), groups = new Map(), calls = [];
  let layer, enabled = true;
  const slot = element('div'), canvas = element('div');
  const shell = { dataset: { mapUrl: '/map.geojson' }, querySelector: sel =>
    sel === '[data-map-slot="layers"]' ? slot : sel === '[data-map-canvas]' ? canvas : null };
  const document = { createElement: element, querySelector: sel => sel === '[data-writers-map]' ? shell : null,
    addEventListener: (name, cb) => listeners.set(name, cb) };
  const draw = {
    clear(id) { calls.push('clear'); groups.delete(id); },
    line(id, coords, style) { calls.push('line'); const group = groups.get(id) || {}; group.line = { coords, style }; groups.set(id, group); },
    markers(id, items) { calls.push('markers'); const group = groups.get(id) || {};
      items.forEach(item => { if (item.onClick) item.element.addEventListener('click', item.onClick); });
      group.markers = items; groups.set(id, group); }
  };
  const window = {};
  if (core) window.TravelMap = {
    registerLayer(name, value) { registered.push(name); layer = value; },
    setLayerEnabled(name, on) { if (on === enabled) return; enabled = on;
      if (on) layer.add({}, { shell, draw }); else layer.remove({}, { shell, draw }); }
  };
  const fetch = async () => { if (fail) throw new Error('offline'); return { ok: true, json: async () => ({ features: places, itineraries }) }; };
  const vmContext = vm.createContext({ document, window, fetch });
  const settle = () => new Promise(resolve => setImmediate(resolve));
  return { registered, shell, slot, settle, run: () => vm.runInContext(source, vmContext),
    async mount() { layer.add({}, { shell, draw }); await settle(); },
    async ready() { listeners.get('travelmap:ready')({ detail: {} }); await settle(); },
    setEnabled: on => window.TravelMap.setLayerEnabled('itineraries', on),
    calls: () => calls, ids: () => [...groups.keys()],
    pins: () => groups.get('itineraries')?.markers || [],
    line: () => groups.get('itineraries')?.line,
    picker: () => slot.children.find(c => c.tagName === 'SELECT'),
    popup: () => canvas.children.find(c => c.className === 'itinerary-popup') };
}

function element(tagName) {
  const listeners = new Map();
  return { tagName: tagName.toUpperCase(), children: [], dataset: {}, className: '', textContent: '', value: '', style: {}, attrs: {},
    setAttribute(name, value) { this.attrs[name] = value; },
    appendChild(child) { this.children.push(child); child.parent = this; return child; },
    remove() { if (this.parent) this.parent.children = this.parent.children.filter(child => child !== this); },
    addEventListener(name, cb) { listeners.set(name, cb); },
    click() { listeners.get('click')?.({ target: this }); },
    change() { listeners.get('change')?.({ target: this }); },
    text() { return this.textContent + this.children.map(child => child.text()).join(''); } };
}

test('numbered pins use the brighter purple, and the white numbers stay readable on it', () => {
  const fill = /\.itinerary-pin \{[^}]*background: (#[0-9a-f]{6});/.exec(stylesheet)[1];
  assert.equal(fill, '#8e5bd6');
  const luminance = hex => [1, 3, 5].map(i => parseInt(hex.slice(i, i + 2), 16) / 255)
    .map(v => v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4)
    .reduce((sum, v, i) => sum + v * [0.2126, 0.7152, 0.0722][i], 0);
  assert.ok(1.05 / (luminance(fill) + 0.05) >= 4.5, 'white numbers need 4.5:1');
});
