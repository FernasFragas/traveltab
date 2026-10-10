import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import vm from 'node:vm';

const source = readFileSync(new URL('../../../public/redesign/writers-map.js', import.meta.url), 'utf8');

test('loads pinned Leaflet assets when a map shell appears', () => {
  const app = browser({ preloaded: false });
  app.run();
  assert.equal(app.window.TravelMap !== undefined, true);
  app.mount();
  assert.equal(app.assets('SCRIPT')[0].src, 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.js');
  assert.match(app.assets('SCRIPT')[0].integrity, /^sha256-/);
  assert.match(app.assets('LINK')[0].integrity, /^sha256-/);
  assert.equal(app.assets('LINK')[0].crossOrigin, 'anonymous');
});

test('uses the freely licensed satellite tiles and their required credit by default', async () => {
  const app = browser();
  app.run();
  app.mount();
  await app.flush();
  const tile = app.maps[0].tiles[0];
  assert.equal(tile.url, 'https://tiles.maps.eox.at/wmts/1.0.0/s2cloudless_3857/default/GoogleMapsCompatible/{z}/{y}/{x}.jpg');
  assert.match(tile.options.attribution, /EOxCloudless.*EOX IT Services GmbH.*Contains modified Copernicus Sentinel data 2016 &amp; 2017/);
  assert.match(tile.options.attribution, /https:\/\/creativecommons.org\/licenses\/by\/4.0\//);
});

test('keeps configured tiles and attribution and falls back to OSM after an imagery error', async () => {
  const app = browser();
  app.run();
  const shell = app.mount();
  shell.root.dataset.mapTileUrl = '/tiles/{z}/{x}/{y}.png';
  shell.root.dataset.mapAttribution = 'Configured tiles';
  await app.flush();
  assert.equal(app.maps[0].tiles[0].url, '/tiles/{z}/{x}/{y}.png');
  assert.equal(app.maps[0].tiles[0].options.attribution, 'Configured tiles');
  app.maps[0].tiles[0].emit('tileerror');
  assert.equal(app.maps[0].tiles.at(-1).url, 'https://tile.openstreetmap.org/{z}/{x}/{y}.png');
  assert.match(app.maps[0].tiles.at(-1).options.attribution, /OpenStreetMap/);
});

test('credits OSM when its URL is configured without an explicit credit', async () => {
  const app = browser();
  app.run();
  const shell = app.mount();
  shell.root.dataset.mapTileUrl = 'https://tile.openstreetmap.org/{z}/{x}/{y}.png';
  await app.flush();
  assert.match(app.maps[0].tiles[0].options.attribution, /OpenStreetMap/);
});

test('renders a keyboard list, emits selection, and shows partial data status', async () => {
  const app = browser();
  app.run();
  const shell = app.mount();
  assert.equal(shell.status.textContent, 'Loading places…');
  app.loadLeaflet();
  await app.flush();
  assert.equal(app.maps.length, 1);
  assert.equal(app.maps[0].tiles[0].url, 'https://tiles.maps.eox.at/wmts/1.0.0/s2cloudless_3857/default/GoogleMapsCompatible/{z}/{y}/{x}.jpg');
  assert.equal(app.events[0].type, 'travelmap:ready');
  assert.equal(shell.status.textContent, 'Some map sources are unavailable.');
  const button = shell.list.children[0].children[0].children[0];
  assert.equal(button.textContent, 'Pico do Arieiro');
  button.click();
  assert.equal(app.events.at(-1).type, 'travelmap:place-selected');
  assert.equal(app.events.at(-1).detail.qid, 'Q1');
});

test('reuses the map across swaps and drops an old response', async () => {
  const app = browser({ delayFirstFetch: true });
  app.run();
  const first = app.mount();
  app.loadLeaflet();
  await app.flush();
  app.body.emit('htmx:beforeSwap', { target: { contains: node => node === first.root } });
  const second = app.mount('Paris');
  await app.flush();
  app.resolveFirstFetch();
  await app.flush();
  assert.equal(app.maps.length, 1);
  assert.equal(app.maps[0].views.length, 2);
  assert.equal(second.canvas.replacement, first.canvas);
  assert.equal(app.assets('LINK').length, 1);
  assert.equal(app.events.filter(event => event.type === 'travelmap:ready').length, 2);
  assert.equal(second.list.children[0].children[0].children[0].textContent, 'Pico do Arieiro');
  assert.equal(first.list.children.length, 0);
});

test('toggles registered layers and reports unavailable places', async () => {
  const app = browser({ failFetch: true });
  app.run();
  const calls = [];
  app.window.TravelMap.registerLayer('places', {
    add: (_, context) => calls.push(['add', context.destination]),
    remove: () => calls.push(['remove'])
  });
  const shell = app.mount();
  app.loadLeaflet();
  await app.flush();
  assert.deepEqual(calls, [['add', 'Madeira']]);
  app.window.TravelMap.setLayerEnabled('places', false);
  assert.deepEqual(calls.at(-1), ['remove']);
  assert.equal(shell.status.textContent, 'Places are unavailable. The map is still available.');
  app.window.TravelMap.updatePlan([{ qids: ['Q1'] }]);
  assert.equal(app.events.at(-1).type, 'travelmap:plan-updated');
  assert.throws(() => app.window.TravelMap.registerLayer('places', { add() {}, remove() {} }), /duplicate/);
});

test('keeps the place list available when Leaflet cannot load', async () => {
  const app = browser({ preloaded: false });
  app.run();
  const shell = app.mount();
  app.assets('SCRIPT')[0].onerror(new Error('offline'));
  await app.flush();
  assert.equal(shell.list.children[0].children[0].children[0].textContent, 'Pico do Arieiro');
  assert.equal(shell.status.textContent, 'Map unavailable. Browse places below.');
  shell.list.children[0].children[0].children[0].click();
  assert.equal(app.events.at(-1).detail.qid, 'Q1');
});

function browser({ failFetch = false, preloaded = true, delayFirstFetch = false } = {}) {
  const nodes = [];
  const events = [];
  let resolveFirstFetch;
  let fetchCount = 0;
  const listeners = new Map();
  const maps = [];
  const body = element('BODY');
  body.addEventListener = (name, callback) => listeners.set(name, callback);
  body.emit = (name, detail) => listeners.get(name)({ detail });
  const makeShell = name => {
    const root = element('SECTION');
    root.dataset = { mapLat: '32.65', mapLon: '-16.9', mapUrl: '/map/' + name.toLowerCase() + '.geojson', mapDestination: name };
    const canvas = element('DIV');
    const list = element('DIV');
    const status = element('DIV');
    root.querySelector = selector => ({
      '[data-map-canvas]': canvas,
      '[data-map-slot="list"]': list,
      '[data-map-slot="status"]': status
    })[selector] || null;
    return { root, canvas, list, status };
  };
  const document = {
    body,
    head: { appendChild(node) { nodes.push(node); } },
    createElement: element,
    querySelector() { return null; },
    dispatchEvent(event) { events.push(event); }
  };
  const window = {
    L: {
      map(canvas) {
        const map = {
          canvas,
          views: [],
          tiles: [],
          setView(point) { this.views.push(point); return this; },
          invalidateSize() {},
          removeLayer() {}
        };
        maps.push(map);
        return map;
      },
      control: { scale() { return { addTo() {} }; } },
      tileLayer(url, options) { return {
        url, options,
        on(name, callback) { this.listeners ??= new Map(); this.listeners.set(name, callback); return this; },
        emit(name) { this.listeners?.get(name)?.(); },
        addTo(map) { map.tiles.push(this); return this; }
      }; }
    }
  };
  if (!preloaded) delete window.L;
  const context = vm.createContext({
    document, window, fetch: () => {
      fetchCount++;
      if (failFetch) return Promise.reject(new Error('unavailable'));
      const response = { ok: true, json: () => Promise.resolve({ complete: false, features: [
        { properties: { qid: 'Q1', name: 'Pico do Arieiro' } }
      ] }) };
      if (delayFirstFetch && fetchCount === 1) return new Promise(resolve => { resolveFirstFetch = () => resolve(response); });
      return Promise.resolve(response);
    },
    CustomEvent: class { constructor(type, options) { this.type = type; this.detail = options.detail; } }
  });
  return {
    window, events, maps, body,
    assets: tagName => nodes.filter(node => node.tagName === tagName),
    run: () => vm.runInContext(source, context),
    mount(name = 'Madeira') { const shell = makeShell(name); window.TravelMap.mount(shell.root); return shell; },
    loadLeaflet() {},
    resolveFirstFetch: () => resolveFirstFetch(),
    flush: async () => { await new Promise(resolve => setImmediate(resolve)); }
  };
}

function element(tagName) {
  const listeners = new Map();
  return {
    tagName: tagName.toUpperCase(),
    children: [],
    setAttribute(name, value) { this[name] = value; },
    addEventListener(name, callback) { listeners.set(name, callback); },
    click() { listeners.get('click')(); },
    appendChild(child) { this.children.push(child); },
    replaceChildren() { this.children = []; },
    replaceWith(node) { this.replacement = node; }
  };
}
