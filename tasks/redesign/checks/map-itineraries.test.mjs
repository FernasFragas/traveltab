import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import vm from 'node:vm';

const source = readFileSync(new URL('../../../public/redesign/map-itineraries.js', import.meta.url), 'utf8');

const FEATURES = [['Q1', 'Tower', 38.1, -9.1], ['Q2', 'Museum', 38.2, -9.2], ['Q3', 'Park', 38.3, -9.3]].map(([qid, name, lat, lon]) =>
  ({ type: 'Feature', geometry: { type: 'Point', coordinates: [lon, lat] }, properties: { qid, name } }));
const ROUTE_A = { writer_host: 'a.example', post_url: 'https://a.example/trip/', post_title: 'Trip A', language: 'en', days: [{ qids: ['Q1', 'Q2'] }, { qids: ['Q3'] }] };
const ROUTE_B = { writer_host: 'b.example', post_url: 'https://b.example/roteiro/', post_title: 'Roteiro B', language: 'pt', days: [{ qids: ['Q3', 'Q1'] }] };

test('registers one layer named itineraries with the core', () => {
  const app = browser();
  app.run();
  assert.deepEqual(app.registered, ['itineraries']);
});

test('does nothing without the core', () => {
  const app = browser({ core: false });
  app.run();
  assert.deepEqual(app.registered, []);
});

test('draws purple numbered pins in order, across days, and a line linking them', async () => {
  const app = browser({ itineraries: [ROUTE_A] });
  app.run();
  await app.mount();
  const pins = app.pins();
  assert.equal(pins.length, 3);
  assert.deepEqual(JSON.parse(JSON.stringify(pins.map(p => p.point))), [[38.1, -9.1], [38.2, -9.2], [38.3, -9.3]]);
  assert.match(pins[0].icon.html, /itinerary-pin/);
  assert.match(pins[2].icon.html, />3</);
  assert.equal(app.lines().length, 1);
  assert.deepEqual(JSON.parse(JSON.stringify(app.lines()[0].points)), [[38.1, -9.1], [38.2, -9.2], [38.3, -9.3]]);
});

test('labels the line as the writer\'s route and links the post', async () => {
  const app = browser({ itineraries: [ROUTE_A] });
  app.run();
  await app.mount();
  assert.equal(app.lines()[0].tooltip, 'Route by a.example');
  const link = app.pins()[0].popup.children.find(c => c.tagName === 'A');
  assert.equal(link.href, 'https://a.example/trip/');
  assert.equal(link.textContent, 'Trip A');
  assert.equal(link.attrs.rel, 'noopener noreferrer');
});

test('ignores a post link that is not http or https', async () => {
  const app = browser({ itineraries: [{ ...ROUTE_A, post_url: 'javascript:alert(1)' }] });
  app.run();
  await app.mount();
  assert.equal(app.pins()[0].popup.children.find(c => c.tagName === 'A'), undefined);
});

test('skips stops missing from the places and draws no line for fewer than two stops', async () => {
  const app = browser({ itineraries: [{ ...ROUTE_A, days: [{ qids: ['Q1', 'Q404'] }] }] });
  app.run();
  await app.mount();
  assert.equal(app.pins().length, 1);
  assert.equal(app.lines().length, 0);
});

test('with no itineraries draws nothing, shows no picker and leaves the shell unmarked', async () => {
  const app = browser({ itineraries: [] });
  app.run();
  await app.mount();
  assert.equal(app.pins().length, 0);
  assert.equal(app.picker(), undefined);
  assert.equal('itineraries' in app.shell.dataset, false);
});

test('marks the shell when itineraries exist, even while the layer is off', async () => {
  const app = browser({ itineraries: [ROUTE_A] });
  app.run();
  await app.ready();
  assert.equal(app.shell.dataset.itineraries, '');
});

test('one itinerary shows no picker; several show one that switches the route', async () => {
  const single = browser({ itineraries: [ROUTE_A] });
  single.run();
  await single.mount();
  assert.equal(single.picker(), undefined);

  const app = browser({ itineraries: [ROUTE_A, ROUTE_B] });
  app.run();
  await app.mount();
  const picker = app.picker();
  assert.deepEqual(JSON.parse(JSON.stringify(picker.children.map(o => o.textContent))), ['a.example: Trip A', 'b.example: Roteiro B']);
  assert.equal(app.pins().length, 3);
  picker.value = '1';
  picker.change();
  assert.equal(app.pins().length, 2);
  assert.equal(app.maps[0].groups.length, 1);
  assert.equal(app.lines()[0].tooltip, 'Route by b.example');
});

test('disabling removes pins and picker; enabling restores one route with no duplicates', async () => {
  const app = browser({ itineraries: [ROUTE_A, ROUTE_B] });
  app.run();
  await app.mount();
  app.setEnabled(false);
  assert.equal(app.pins().length, 0);
  assert.equal(app.picker(), undefined);
  app.setEnabled(true);
  await app.settle();
  assert.equal(app.pins().length, 3);
  assert.equal(app.maps[0].groups.length, 1);
  assert.equal(app.slot.children.filter(c => c.tagName === 'SELECT').length, 1);
});

test('a failed fetch draws nothing and does not throw', async () => {
  const app = browser({ fail: true });
  app.run();
  await app.mount();
  assert.equal(app.pins().length, 0);
  assert.equal('itineraries' in app.shell.dataset, false);
});

function browser({ core = true, itineraries = [], fail = false } = {}) {
  const registered = [];
  const maps = [];
  const listeners = new Map();
  let layer = null;
  let enabled = true;
  let currentMap = null;
  const slot = element('div');
  const shell = { dataset: { mapUrl: '/map.geojson' }, querySelector: sel => sel === '[data-map-slot="layers"]' ? slot : null };
  const document = {
    createElement: tag => element(tag),
    querySelector: sel => sel === '[data-writers-map]' ? shell : null,
    addEventListener: (n, cb) => listeners.set(n, cb)
  };
  const group = () => ({
    layers: [],
    addLayer(l) { this.layers.push(l); return this; },
    addTo(map) { map.groups.push(this); this.map = map; return this; },
    remove() { this.map.groups = this.map.groups.filter(g => g !== this); }
  });
  const L = {
    divIcon: options => options,
    marker: (point, options) => ({ point, icon: options.icon, title: options.title, bindPopup(el) { this.popup = el; return this; } }),
    polyline: (points) => ({ points, line: true, bindTooltip(text) { this.tooltip = text; return this; } }),
    layerGroup: group
  };
  const window = { L };
  if (core) {
    window.TravelMap = {
      registerLayer(name, l) { registered.push(name); layer = l; },
      setLayerEnabled(name, on) {
        if (on === enabled) return;
        enabled = on;
        if (on) layer.add(currentMap, { shell }); else layer.remove(currentMap, { shell });
      }
    };
  }
  const fetch = async () => {
    if (fail) throw new Error('offline');
    return { ok: true, json: async () => ({ features: FEATURES, itineraries }) };
  };
  const vmContext = vm.createContext({ document, window, fetch });
  const settle = () => new Promise(resolve => setImmediate(resolve));
  return {
    registered, maps, shell, slot, settle,
    run: () => vm.runInContext(source, vmContext),
    async mount() {
      currentMap = { groups: [] };
      maps.push(currentMap);
      layer.add(currentMap, { shell });
      await settle();
    },
    async ready() {
      listeners.get('travelmap:ready')({ detail: {} });
      await settle();
    },
    setEnabled: on => window.TravelMap.setLayerEnabled('itineraries', on),
    all: () => maps[0].groups.flatMap(g => g.layers),
    pins() { return this.all().filter(l => !l.line); },
    lines() { return this.all().filter(l => l.line); },
    picker: () => slot.children.find(c => c.tagName === 'SELECT')
  };
}

function element(tagName) {
  const listeners = new Map();
  return {
    tagName: tagName.toUpperCase(), children: [], dataset: {}, className: '', textContent: '', value: '',
    attrs: {}, setAttribute(n, v) { this.attrs[n] = v; },
    appendChild(c) { this.children.push(c); c.parent = this; return c; },
    remove() { if (this.parent) this.parent.children = this.parent.children.filter(c => c !== this); },
    addEventListener(n, cb) { listeners.set(n, cb); },
    change() { listeners.get('change')({ target: this }); }
  };
}
