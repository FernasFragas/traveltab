import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import vm from 'node:vm';

const read = name => readFileSync(new URL(`../../../public/redesign/${name}`, import.meta.url), 'utf8');
const sources = [read('writers-map-leaflet.js'), read('writers-map.js')];
const drawCalls = ['clear', 'control', 'fit', 'flyTo', 'line', 'markers'];
const eoxTiles = 'https://tiles.maps.eox.at/wmts/1.0.0/s2cloudless_3857/default/GoogleMapsCompatible/{z}/{y}/{x}.jpg';

// --- MapLibre engine ---

test('loads the pinned MapLibre assets, and not Leaflet, when a map shell appears', () => {
  const app = browser({ maplibre: null });
  app.run();
  app.mount();
  assert.equal(app.assets('SCRIPT').length, 1);
  assert.equal(app.assets('SCRIPT')[0].src, 'https://cdn.jsdelivr.net/npm/maplibre-gl@5.24.0/dist/maplibre-gl.js');
  assert.match(app.assets('SCRIPT')[0].integrity, /^sha384-/);
  assert.equal(app.assets('LINK')[0].href, 'https://cdn.jsdelivr.net/npm/maplibre-gl@5.24.0/dist/maplibre-gl.css');
  assert.match(app.assets('LINK')[0].integrity, /^sha384-/);
  assert.equal(app.assets('LINK')[0].crossOrigin, 'anonymous');
});

test('draws a 3D MapLibre map with the default style, terrain and camera', async () => {
  const app = browser();
  app.run();
  app.mount();
  await app.flush();
  const map = app.maplibre[0];
  assert.equal(map.options.style, 'https://tiles.openfreemap.org/styles/liberty');
  assert.deepEqual(Array.from(map.options.center), [-16.9, 32.65]);
  assert.deepEqual([map.options.zoom, map.options.pitch, map.options.bearing], [12.3, 55, -20]);
  const source = map.sources['terrain-dem'];
  assert.deepEqual(Array.from(source.tiles), ['https://tiles.mapterhorn.com/{z}/{x}/{y}.webp']);
  assert.equal(source.tileSize, 512);
  assert.equal(map.terrain.source, 'terrain-dem');
  assert.equal(map.layers.some(layer => layer.type === 'hillshade'), false);
  assert.equal(app.leafletMaps.length, 0);
  assert.equal(app.events[0].type, 'travelmap:ready');
  assert.equal(app.events[0].detail.engine, 'maplibre');
  assert.equal(app.events[0].detail.map, map);
});

test('takes the style and terrain from the shell, and an empty terrain URL turns terrain off', async () => {
  const app = browser();
  app.run();
  app.mount('Madeira', { mapStyleUrl: '/map-fixtures/style.json', mapTerrainUrl: '/map-fixtures/dem/{z}/{x}/{y}.png' });
  await app.flush();
  assert.equal(app.maplibre[0].options.style, '/map-fixtures/style.json');
  assert.deepEqual(Array.from(app.maplibre[0].sources['terrain-dem'].tiles), ['/map-fixtures/dem/{z}/{x}/{y}.png']);

  const plain = browser();
  plain.run();
  plain.mount('Madeira', { mapTerrainUrl: '' });
  await plain.flush();
  assert.equal(plain.maplibre[0].terrain, undefined);
  assert.equal(plain.maplibre[0].sources['terrain-dem'], undefined);
});

// --- Fallback ---

test('falls back to Leaflet with maxZoom 18 when WebGL is missing', async () => {
  const app = browser({ webgl: false });
  app.run();
  app.mount();
  await app.flush();
  assert.equal(app.maplibre.length, 0);
  assert.equal(app.leafletMaps[0].options.maxZoom, 18);
  assert.equal(app.leafletMaps[0].tiles[0].options.maxZoom, 18);
  assert.equal(app.events[0].detail.engine, 'leaflet');
  assert.equal(app.events[0].detail.map, app.leafletMaps[0]);
});

test('data-map-engine="leaflet" forces the fallback', async () => {
  const app = browser();
  app.run();
  app.mount('Madeira', { mapEngine: 'leaflet' });
  await app.flush();
  assert.equal(app.maplibre.length, 0);
  assert.equal(app.events[0].detail.engine, 'leaflet');
});

test('a style load error falls back to Leaflet and adds the layers once', async () => {
  const app = browser({ maplibre: { ready: false } });
  app.run();
  const added = [];
  app.window.TravelMap.registerLayer('pins', layerSpy(added));
  app.mount();
  await app.flush();
  app.maplibre[0].emit('error', { error: new Error('style 404') });
  await app.flush();
  assert.equal(app.maplibre[0].removed, true);
  assert.equal(app.leafletMaps.length, 1);
  assert.deepEqual(added.filter(call => call[0] === 'add').map(call => call[1]), ['leaflet']);
  assert.equal(added.filter(call => call[0] === 'remove').length, 0);
  assert.deepEqual(app.events.filter(event => event.type === 'travelmap:ready').map(event => event.detail.engine), ['leaflet']);
});

test('a style that never loads falls back to Leaflet', async () => {
  const app = browser({ maplibre: { ready: false } });
  app.run();
  app.mount();
  await app.flush();
  assert.equal(app.leafletMaps.length, 0);
  app.timers.at(-1).fn();
  await app.flush();
  assert.equal(app.leafletMaps.length, 1);
  assert.equal(app.events[0].detail.engine, 'leaflet');
});

test('repeated tile failures after the layers were added remove them, mount Leaflet and add them again', async () => {
  const app = browser();
  app.run();
  const calls = [];
  app.window.TravelMap.registerLayer('pins', layerSpy(calls));
  app.mount();
  await app.flush();
  assert.deepEqual(calls.map(call => call.slice(0, 2)), [['add', 'maplibre']]);
  const tileError = { sourceId: 'terrain-dem', error: new Error('tile 404') };
  app.maplibre[0].emit('error', tileError);
  app.maplibre[0].emit('error', tileError);
  assert.equal(app.leafletMaps.length, 0, 'two failed tiles are not an outage');
  app.maplibre[0].emit('error', tileError);
  await app.flush();
  assert.deepEqual(calls.map(call => call.slice(0, 2)), [['add', 'maplibre'], ['remove', 'maplibre'], ['add', 'leaflet']]);
  assert.deepEqual(app.events.filter(event => event.type === 'travelmap:ready').map(event => event.detail.engine), ['maplibre', 'leaflet']);
  assert.equal(app.leafletMaps[0].options.maxZoom, 18);
});

test('a style error after the map is ready switches to Leaflet', async () => {
  const app = browser();
  app.run();
  const calls = [];
  app.window.TravelMap.registerLayer('pins', layerSpy(calls));
  app.mount();
  await app.flush();
  app.maplibre[0].emit('error', { error: new Error('style reload failed') });
  await app.flush();
  assert.deepEqual(calls.map(call => call.slice(0, 2)), [['add', 'maplibre'], ['remove', 'maplibre'], ['add', 'leaflet']]);
  assert.equal(app.maplibre[0].removed, true);
});

test('a source outage after earlier successful tiles falls back to Leaflet', async () => {
  const app = browser();
  app.run();
  app.mount();
  await app.flush();
  for (let i = 0; i < 2; i++) app.maplibre[0].emit('error', { sourceId: 'openmaptiles', error: new Error('tile 404') });
  app.maplibre[0].emit('sourcedata', { sourceId: 'openmaptiles', tile: {} });
  for (let i = 0; i < 2; i++) app.maplibre[0].emit('error', { sourceId: 'openmaptiles', error: new Error('tile 404') });
  assert.equal(app.leafletMaps.length, 0, 'one successful tile resets the failure count');
  app.maplibre[0].emit('error', { sourceId: 'openmaptiles', error: new Error('tile 404') });
  await app.flush();
  assert.equal(app.leafletMaps.length, 1);
});

test('keeps the EOX imagery and its credit on the fallback, and swaps to OSM on a tile error', async () => {
  const app = browser({ webgl: false });
  app.run();
  const shell = app.mount();
  await app.flush();
  const tile = app.leafletMaps[0].tiles[0];
  assert.equal(tile.url, eoxTiles);
  assert.match(tile.options.attribution, /EOxCloudless.*EOX IT Services GmbH.*Contains modified Copernicus Sentinel data 2016 &amp; 2017/);
  assert.match(tile.options.attribution, /https:\/\/creativecommons.org\/licenses\/by\/4.0\//);
  tile.emit('tileerror');
  assert.equal(app.leafletMaps[0].tiles.at(-1).url, 'https://tile.openstreetmap.org/{z}/{x}/{y}.png');
  assert.match(app.leafletMaps[0].tiles.at(-1).options.attribution, /OpenStreetMap/);
  assert.equal(app.leafletMaps[0].tiles.at(-1).options.maxZoom, 18);
  assert.ok(shell);
});

test('keeps configured fallback tiles and attribution, and credits OSM when its URL is configured', async () => {
  const app = browser({ webgl: false });
  app.run();
  const shell = app.mount();
  shell.root.dataset.mapTileUrl = '/tiles/%7Bz%7D/%7Bx%7D/%7By%7D.png';
  shell.root.dataset.mapAttribution = 'Configured tiles';
  await app.flush();
  assert.equal(app.leafletMaps[0].tiles[0].url, '/tiles/{z}/{x}/{y}.png');
  assert.equal(app.leafletMaps[0].tiles[0].options.attribution, 'Configured tiles');

  const osm = browser({ webgl: false });
  osm.run();
  osm.mount().root.dataset.mapTileUrl = 'https://tile.openstreetmap.org/{z}/{x}/{y}.png';
  await osm.flush();
  assert.match(osm.leafletMaps[0].tiles[0].options.attribution, /OpenStreetMap/);
});

test('loads the pinned Leaflet assets only for the fallback', () => {
  const app = browser({ webgl: false, leaflet: null });
  app.run();
  app.mount();
  assert.equal(app.assets('SCRIPT')[0].src, 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.js');
  assert.match(app.assets('SCRIPT')[0].integrity, /^sha256-/);
  assert.match(app.assets('LINK')[0].integrity, /^sha256-/);
  assert.equal(app.assets('LINK')[0].crossOrigin, 'anonymous');
});

// --- Drawing API ---

test('context.draw has the same calls on both engines', async () => {
  const shapes = {};
  for (const webgl of [true, false]) {
    const app = browser({ webgl });
    app.run();
    let draw;
    app.window.TravelMap.registerLayer('probe', { add: (_, context) => { draw = context.draw; }, remove() {} });
    app.mount();
    await app.flush();
    shapes[webgl ? 'maplibre' : 'leaflet'] = Object.keys(draw).sort();
  }
  assert.deepEqual(shapes.maplibre, drawCalls);
  assert.deepEqual(shapes.leaflet, drawCalls);
});

for (const engine of ['maplibre', 'leaflet']) {
  test(`${engine}: markers, line and control draw, replace by id, and clear`, async () => {
    const { app, draw } = await drawn(engine);
    const clicks = [];
    const items = ids => ids.map((id, i) => ({
      lngLat: [-16.9 + i / 100, 32.65], element: app.element('BUTTON'), title: `Stop ${id}`, z: i, onClick: () => clicks.push(id)
    }));
    draw.markers('plan', items([1, 2, 3]));
    draw.markers('plan', items([1, 2]));
    draw.markers('other', items([9]));
    assert.equal(app.visible().markers, 3, 'replace by id leaves two plus one');
    const first = app.markerElements()[0];
    assert.equal(first.title, 'Stop 1');
    assert.equal(first['aria-label'], 'Stop 1');
    first.click();
    assert.deepEqual(clicks, [1]);

    draw.line('plan', [[-16.9, 32.65], [-16.8, 32.66]], { color: '#a64027', width: 4, dash: [2, 1] });
    draw.line('plan', [[-16.9, 32.65], [-16.7, 32.7]], { color: '#a64027', width: 4 });
    draw.control('plan', app.element('DIV'), 'top-right');
    draw.control('plan', app.element('DIV'), 'top-right');
    assert.deepEqual(app.visible(), { markers: 3, lines: 1, controls: 1 });

    draw.clear('plan');
    draw.clear('unknown');
    assert.deepEqual(app.visible(), { markers: 1, lines: 0, controls: 0 });
  });

  test(`${engine}: fit and flyTo move the camera`, async () => {
    const { app, draw } = await drawn(engine);
    draw.fit([[-17, 32.6], [-16.8, 32.7]], { padding: 40 });
    draw.flyTo([-16.9, 32.65], 15);
    assert.deepEqual(JSON.parse(JSON.stringify(app.camera())), { fit: [[-17, 32.6], [-16.8, 32.7]], padding: 40, fly: [-16.9, 32.65], zoom: 15 });
  });

  test(`${engine}: a layer that clears its ids on remove gets the same result from add after remove`, async () => {
    const app = browser({ webgl: engine === 'maplibre' });
    app.run();
    const layer = {
      add: (_, context) => {
        context.draw.markers('pins', [{ lngLat: [-16.9, 32.65], element: app.element('BUTTON'), title: 'A', z: 1 }]);
        context.draw.line('pins', [[-16.9, 32.65], [-16.8, 32.7]], { color: '#000', width: 2 });
      },
      remove: (_, context) => context.draw.clear('pins')
    };
    app.window.TravelMap.registerLayer('pins', layer);
    app.mount();
    await app.flush();
    const first = app.visible();
    app.window.TravelMap.setLayerEnabled('pins', false);
    assert.deepEqual(app.visible(), { markers: 0, lines: 0, controls: 0 });
    app.window.TravelMap.setLayerEnabled('pins', true);
    assert.deepEqual(app.visible(), first);
    assert.deepEqual(first, { markers: 1, lines: 1, controls: 0 });
  });
}

test('a style reload re-adds lines without duplicating markers', async () => {
  const { app, draw } = await drawn('maplibre');
  draw.markers('plan', [{ lngLat: [-16.9, 32.65], element: app.element('BUTTON'), title: 'A', z: 1 }]);
  draw.line('plan', [[-16.9, 32.65], [-16.8, 32.7]], { color: '#a64027', width: 4 });
  app.maplibre[0].dropCustomLayers();
  assert.equal(app.visible().lines, 0);
  app.maplibre[0].emit('style.load');
  app.maplibre[0].emit('style.load');
  assert.deepEqual(app.visible(), { markers: 1, lines: 1, controls: 0 });
  assert.equal(app.maplibre[0].sources['terrain-dem'] !== undefined, true);
});

test('a line drawn while the style is reloading waits for style.load instead of throwing', async () => {
  const { app, draw } = await drawn('maplibre');
  app.maplibre[0].styleLoading = true;
  assert.doesNotThrow(() => draw.line('plan', [[-16.9, 32.65], [-16.8, 32.7]], { color: '#a64027', width: 4 }));
  assert.equal(app.visible().lines, 0);
  app.maplibre[0].styleLoading = false;
  app.maplibre[0].emit('style.load');
  assert.equal(app.visible().lines, 1);
});

// --- Layers ---

test('a layer that throws in add is logged and skipped, and the others still draw', async () => {
  const app = browser();
  app.run();
  const added = [];
  app.window.TravelMap.registerLayer('old', { add() { throw new TypeError('L is not defined'); }, remove() {} });
  app.window.TravelMap.registerLayer('pins', layerSpy(added));
  app.mount();
  await app.flush();
  assert.deepEqual(added.map(call => call[0]), ['add']);
  assert.match(app.warnings[0], /old.*L is not defined/);
  app.window.TravelMap.setLayerEnabled('old', false);
  assert.equal(app.events.at(-1).type, 'travelmap:ready');
});

test('gives layers the engine, destination, features and shell in the context', async () => {
  const app = browser();
  app.run();
  let seen;
  app.window.TravelMap.registerLayer('probe', { add: (_, context) => { seen = context; }, remove() {} });
  const shell = app.mount();
  await app.flush();
  assert.equal(seen.engine, 'maplibre');
  assert.equal(seen.destination, 'Madeira');
  assert.equal(seen.shell, shell.root);
  assert.equal(seen.features[0].properties.qid, 'Q1');
  assert.equal(typeof seen.selectPlace, 'function');
});

// --- Core behaviour kept from before ---

test('hides loading status, renders a keyboard list, and shows partial data status', async () => {
  const app = browser();
  app.run();
  const shell = app.mount();
  assert.equal(shell.status.textContent, '');
  assert.equal(shell.status.hidden, true);
  await app.flush();
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
  await app.flush();
  app.body.emit('htmx:beforeSwap', { target: { contains: node => node === first.root } });
  const second = app.mount('Paris');
  await app.flush();
  app.resolveFirstFetch();
  await app.flush();
  assert.equal(app.maplibre.length, 1);
  assert.equal(app.maplibre[0].jumps.length, 1);
  assert.equal(second.canvas.replacement, first.canvas);
  assert.equal(app.assets('LINK').length, 1);
  assert.equal(app.events.filter(event => event.type === 'travelmap:ready').length, 2);
  assert.equal(second.list.children[0].children[0].children[0].textContent, 'Pico do Arieiro');
  assert.equal(first.list.children.length, 0);
});

test('a swap clears what the layers drew before the map is reused', async () => {
  const app = browser();
  app.run();
  const layer = {
    add: (_, context) => context.draw.markers('pins', [{ lngLat: [0, 0], element: app.element('BUTTON'), title: 'A' }]),
    remove: (_, context) => context.draw.clear('pins')
  };
  app.window.TravelMap.registerLayer('pins', layer);
  const first = app.mount();
  await app.flush();
  app.body.emit('htmx:beforeSwap', { target: { contains: node => node === first.root } });
  app.mount('Paris');
  await app.flush();
  assert.equal(app.visible().markers, 1);
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
  await app.flush();
  assert.deepEqual(calls, [['add', 'Madeira']]);
  app.window.TravelMap.setLayerEnabled('places', false);
  assert.deepEqual(calls.at(-1), ['remove']);
  assert.equal(shell.status.textContent, 'Places are unavailable. The map is still available.');
  app.window.TravelMap.updatePlan([{ qids: ['Q1'] }]);
  assert.equal(app.events.at(-1).type, 'travelmap:plan-updated');
  assert.throws(() => app.window.TravelMap.registerLayer('places', { add() {}, remove() {} }), /duplicate/);
});

test('keeps the place list available when neither engine can load', async () => {
  const app = browser({ maplibre: null, leaflet: null, webgl: true });
  app.run();
  const shell = app.mount();
  app.assets('SCRIPT')[0].onerror(new Error('offline'));
  await app.flush();
  app.assets('SCRIPT')[1].onerror(new Error('offline'));
  await app.flush();
  assert.equal(shell.list.children[0].children[0].children[0].textContent, 'Pico do Arieiro');
  assert.equal(shell.status.textContent, 'Map unavailable. Browse places below.');
  shell.list.children[0].children[0].children[0].click();
  assert.equal(app.events.at(-1).detail.qid, 'Q1');
});

test('falls back to Leaflet when the MapLibre script cannot load', async () => {
  const app = browser({ maplibre: null, leaflet: null });
  app.run();
  app.mount();
  app.assets('SCRIPT')[0].onerror(new Error('offline'));
  await app.flush();
  assert.equal(app.assets('SCRIPT')[1].src, 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.js');
  app.window.L = leafletStub(app.leafletMaps);
  app.assets('SCRIPT')[1].onload();
  await app.flush();
  assert.equal(app.events[0].detail.engine, 'leaflet');
});

// --- Test doubles ---

function layerSpy(calls) {
  return {
    add: (map, context) => calls.push(['add', context.engine, map]),
    remove: (map, context) => calls.push(['remove', context.engine, map])
  };
}

async function drawn(engine) {
  const app = browser({ webgl: engine === 'maplibre' });
  app.run();
  let draw;
  app.window.TravelMap.registerLayer('probe', { add: (_, context) => { draw = context.draw; }, remove() {} });
  app.mount();
  await app.flush();
  return { app, draw };
}

// options: webgl (default true); maplibre: null (not loaded yet) or { ready: false } (never loads on its own);
// leaflet: null (not loaded yet).
function browser({ failFetch = false, delayFirstFetch = false, webgl = true, maplibre = {}, leaflet = {} } = {}) {
  const nodes = [];
  const events = [];
  const warnings = [];
  const timers = [];
  const maplibreMaps = [];
  const leafletMaps = [];
  const listeners = new Map();
  let resolveFirstFetch;
  let fetchCount = 0;
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
    createElement(tag) {
      const node = element(tag);
      if (tag === 'canvas') node.getContext = type => webgl && type === 'webgl2' ? {} : null;
      return node;
    },
    querySelector() { return null; },
    dispatchEvent(event) { events.push(event); }
  };
  const window = {};
  if (maplibre) window.maplibregl = maplibreStub(maplibreMaps, maplibre.ready !== false);
  if (leaflet) window.L = leafletStub(leafletMaps);
  const context = vm.createContext({
    document, window,
    console: { warn: (...args) => warnings.push(args.map(String).join(' ')), error() {}, log() {} },
    setTimeout: (fn, ms) => { timers.push({ fn, ms }); return timers.length; },
    clearTimeout() {},
    fetch: () => {
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
  const engineOf = () => (maplibreMaps.length && !maplibreMaps.at(-1).removed ? 'maplibre' : 'leaflet');
  const app = {
    window, events, warnings, timers, body,
    maplibre: maplibreMaps, leafletMaps,
    assets: tagName => nodes.filter(node => node.tagName === tagName),
    run: () => sources.forEach(source => vm.runInContext(source, context)),
    mount(name = 'Madeira', dataset = {}) { const shell = makeShell(name); Object.assign(shell.root.dataset, dataset); window.TravelMap.mount(shell.root); return shell; },
    resolveFirstFetch: () => resolveFirstFetch(),
    flush: async () => { await new Promise(resolve => setImmediate(resolve)); await new Promise(resolve => setImmediate(resolve)); },
    element,
    visible() {
      if (engineOf() === 'maplibre') {
        const map = maplibreMaps.at(-1);
        return {
          markers: map.markers.filter(marker => !marker.removed).length,
          lines: map.layers.filter(layer => layer.id.startsWith('draw-line')).length,
          controls: map.controls.filter(control => control.custom).length
        };
      }
      const map = leafletMaps.at(-1);
      return {
        markers: map.layers.filter(layer => layer.kind === 'marker').length,
        lines: map.layers.filter(layer => layer.kind === 'line').length,
        controls: map.controls.filter(control => control.custom).length
      };
    },
    markerElements() {
      if (engineOf() === 'maplibre') return maplibreMaps.at(-1).markers.filter(marker => !marker.removed).map(marker => marker.element);
      return leafletMaps.at(-1).layers.filter(layer => layer.kind === 'marker').map(layer => layer.element);
    },
    camera() {
      if (engineOf() === 'maplibre') {
        const map = maplibreMaps.at(-1);
        return { fit: map.fit[0], padding: map.fit[1].padding, fly: map.fly.center, zoom: map.fly.zoom };
      }
      const map = leafletMaps.at(-1);
      return { fit: [map.fit[0][0].slice().reverse(), map.fit[0][1].slice().reverse()], padding: map.fit[1].padding[0], fly: map.fly[0].slice().reverse(), zoom: map.fly[1] };
    }
  };
  return app;
}

function maplibreStub(maps, auto) {
  class Map {
    constructor(options) {
      Object.assign(this, { options, handlers: {}, sources: {}, layers: [], markers: [], controls: [], jumps: [], fit: null, fly: null });
      maps.push(this);
      if (auto) setImmediate(() => { this.emit('style.load'); this.emit('load'); });
    }
    on(name, callback) { (this.handlers[name] ??= []).push(callback); return this; }
    emit(name, event) { (this.handlers[name] || []).slice().forEach(callback => callback(event)); }
    addSource(id, source) { if (this.styleLoading) throw new Error('Style is not done loading.'); this.sources[id] = source; }
    getSource(id) { return this.sources[id]; }
    removeSource(id) { delete this.sources[id]; }
    addLayer(layer) { this.layers.push(layer); }
    getLayer(id) { return this.layers.find(layer => layer.id === id); }
    removeLayer(id) { this.layers = this.layers.filter(layer => layer.id !== id); }
    dropCustomLayers() { this.layers = []; Object.keys(this.sources).forEach(id => delete this.sources[id]); }
    setTerrain(terrain) { this.terrain = terrain; }
    setSky() {}
    addControl(control, position) { this.controls.push({ control, position, custom: Boolean(control.onAdd && !control.native) }); }
    removeControl(control) { this.controls = this.controls.filter(item => item.control !== control); }
    fitBounds(bounds, options) { this.fit = [bounds, options]; }
    flyTo(options) { this.fly = options; }
    jumpTo(options) { this.jumps.push(options); }
    resize() {}
    remove() { this.removed = true; }
  }
  class Marker {
    constructor(options) { this.element = options.element; }
    setLngLat(lngLat) { this.lngLat = lngLat; return this; }
    addTo(map) { map.markers.push(this); return this; }
    remove() { this.removed = true; return this; }
  }
  class Control { constructor() { this.native = true; } }
  return { Map, Marker, NavigationControl: Control, ScaleControl: Control, AttributionControl: Control };
}

function leafletStub(maps) {
  const control = options => ({
    options, custom: true,
    addTo(map) { this.map = map; this.element = this.onAdd(map); map.controls.push(this); return this; },
    remove() { this.map.controls = this.map.controls.filter(item => item !== this); return this; }
  });
  control.scale = () => ({ addTo(map) { map.controls.push({ custom: false }); return this; } });
  return {
    map(canvas, options) {
      const map = {
        canvas, options, views: [], tiles: [], layers: [], controls: [], fit: null, fly: null,
        setView(point, zoom) { this.views.push([point, zoom]); return this; },
        invalidateSize() {},
        addLayer(layer) { this.layers.push(layer); return this; },
        removeLayer(layer) { this.layers = this.layers.filter(item => item !== layer); },
        fitBounds(bounds, options) { this.fit = [bounds, options]; },
        flyTo(point, zoom) { this.fly = [point, zoom]; },
        getZoom() { return 12; },
        remove() { this.removed = true; }
      };
      maps.push(map);
      return map;
    },
    control,
    divIcon: options => options,
    marker(point, options) {
      return {
        kind: 'marker', point, options, element: options.icon.html.children[0],
        addTo(map) { map.layers.push(this); return this; }
      };
    },
    polyline(points, options) {
      return { kind: 'line', points, options, addTo(map) { map.layers.push(this); return this; } };
    },
    DomEvent: { disableClickPropagation() {} },
    tileLayer(url, options) { return {
      url, options,
      on(name, callback) { this.listeners ??= new Map(); this.listeners.set(name, callback); return this; },
      emit(name) { this.listeners?.get(name)?.(); },
      addTo(map) { map.tiles.push(this); return this; }
    }; }
  };
}

function element(tagName) {
  const listeners = new Map();
  return {
    tagName: tagName.toUpperCase(),
    children: [],
    style: {},
    dataset: {},
    setAttribute(name, value) { this[name] = value; },
    addEventListener(name, callback) { listeners.set(name, callback); },
    click() { listeners.get('click')(); },
    appendChild(child) { this.children.push(child); },
    replaceChildren() { this.children = []; },
    replaceWith(node) { this.replacement = node; },
    remove() { this.removed = true; }
  };
}
