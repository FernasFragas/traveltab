import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import vm from 'node:vm';

const source = readFileSync(new URL('../../../public/redesign/map-panel.js', import.meta.url), 'utf8');

test('registers the notable and writers pin layers with the map core', () => {
  const app = browser();
  app.run();
  assert.deepEqual([...app.layers.keys()], ['notable', 'writers']);
});

test('draws writer pins orange with their count, above blue notable pins', () => {
  const app = browser();
  app.run();
  const map = app.addLayers();
  const writers = map.markers.filter(marker => marker.options.icon.className.includes('map-pin-writer'));
  const notable = map.markers.filter(marker => marker.options.icon.className.includes('map-pin-notable'));
  assert.deepEqual(writers.map(marker => marker.options.title), ['Pico Ruivo', 'Cabo Girão']);
  assert.equal(writers[0].options.icon.html, '<span>2</span>');
  assert.deepEqual(notable.map(marker => marker.options.title), ['Funchal', 'Seixal']);
  assert.ok(writers.every(marker => marker.options.zIndexOffset > notable[0].options.zIndexOffset));
});

test('labels only the biggest places', () => {
  const app = browser({ features: [...features(), ...extraPlaces(6)] });
  app.run();
  const map = app.addLayers();
  const labelled = map.markers.filter(marker => marker.tooltip).map(marker => marker.tooltip.text);
  assert.equal(labelled.length, 5);
  assert.ok(labelled.includes('Funchal'));
  assert.ok(!labelled.includes('Seixal'));
});

test('selects a place when its pin is clicked', () => {
  const app = browser();
  app.run();
  const map = app.addLayers();
  map.markers.find(marker => marker.options.title === 'Seixal').emit('click');
  assert.deepEqual(app.selected, ['Q3']);
});

test('removing a layer takes its pins off the map and adding it again leaves one copy', () => {
  const app = browser();
  app.run();
  const map = app.addLayers();
  app.layers.get('writers').remove(map, app.context);
  assert.equal(map.groups.length, 1);
  app.layers.get('writers').add(map, app.context);
  assert.equal(map.groups.length, 2);
});

test('loads the selected place into the panel slot', () => {
  const app = browser();
  app.run();
  app.select('Q1');
  assert.deepEqual(app.requests, [['GET', '/map/madeira-pt/Q1?lat=1&lon=2', app.panel]]);
});

test('draws one monogram per writer with a colour from the host, and +N past three', () => {
  const app = browser();
  app.run();
  app.swapPanel([
    ['travel.example', 'Salt in our Hair'], ['travel.example', 'Salt in our Hair'],
    ['viagens.example', 'Viagens à Solta'], ['c.example', 'Casa'], ['d.example', 'Dias'], ['e.example', 'Ermo']
  ]);
  const monograms = app.monograms.children;
  assert.deepEqual(monograms.map(item => item.textContent), ['SI', 'VÀ', 'C', '+2']);
  assert.match(monograms[0].style.backgroundColor, /^hsl\(\d+ /);
  assert.equal(monograms[0].title, 'Salt in our Hair');
  const again = browser();
  again.run();
  again.swapPanel([['travel.example', 'Salt in our Hair']]);
  assert.equal(again.monograms.children[0].style.backgroundColor, monograms[0].style.backgroundColor);
});

test('close returns the panel to its empty state', () => {
  const app = browser();
  app.run();
  app.swapPanel([['travel.example', 'Salt in our Hair']]);
  app.body.emit('click', { target: element('button', { className: 'map-panel-close', closestOf: app.panel }) });
  assert.equal(app.panel.children.length, 1);
  assert.equal(app.panel.children[0].textContent, 'Pick a place on the map');
});

test('does nothing without the map core', () => {
  const app = browser({ core: false });
  assert.doesNotThrow(() => app.run());
  assert.equal(app.layers.size, 0);
});

function features() {
  return [
    feature('Q1', 'Pico Ruivo', 2, 45),
    feature('Q2', 'Funchal', 0, 92),
    feature('Q3', 'Seixal', 0, 14),
    feature('Q4', 'Cabo Girão', 1, 31)
  ];
}

function extraPlaces(count) {
  return Array.from({ length: count }, (_, index) => feature('Q9' + index, 'Place ' + index, 0, 40 + index));
}

function feature(qid, name, writers, sitelinks) {
  return {
    type: 'Feature',
    geometry: { type: 'Point', coordinates: [-16.9, 32.7] },
    properties: { qid, name, writer_count: writers, sitelinks, panel_url: '/map/madeira-pt/' + qid + '?lat=1&lon=2' }
  };
}

function browser(options = {}) {
  const layers = new Map();
  const selected = [];
  const requests = [];
  const listeners = new Map();
  const body = target();
  const panel = element('div');
  const shell = element('div', { slots: { panel } });
  const context = { features: options.features || features(), selectPlace: qid => selected.push(qid), shell };
  const window = {
    L: leaflet(),
    htmx: { ajax: (method, url, slot) => { requests.push([method, url, slot]); return Promise.resolve(); } }
  };
  if (options.core !== false) {
    window.TravelMap = { registerLayer: (name, layer) => layers.set(name, layer) };
  }
  const document = {
    body,
    addEventListener: (type, fn) => listeners.set(type, fn),
    querySelector: selector => selector === '[data-writers-map]' ? shell : null,
    createElement: tag => element(tag)
  };
  const app = {
    layers, selected, requests, body, panel, context, monograms: null,
    run: () => vm.runInNewContext(source, { window, document, ...window }),
    addLayers() {
      const map = fakeMap();
      layers.forEach(layer => layer.add(map, context));
      return map;
    },
    select(qid) {
      const place = context.features.find(item => item.properties.qid === qid);
      listeners.get('travelmap:place-selected')({ detail: { qid, place } });
    },
    swapPanel(writers) {
      app.monograms = element('span', { className: 'map-panel-monograms' });
      panel.children = [app.monograms, ...writers.map(([host, blog]) =>
        element('a', { dataset: { writerHost: host, blogName: blog } }))];
      body.emit('htmx:afterSwap', { detail: { target: panel } });
    }
  };
  return app;
}

function leaflet() {
  return {
    divIcon: icon => icon,
    layerGroup: () => {
      const group = { markers: [], addLayer(marker) { group.markers.push(marker); }, addTo(map) { map.add(group); return group; } };
      return group;
    },
    marker: (latlng, options) => {
      const marker = target();
      Object.assign(marker, { latlng, options, bindTooltip(text, tooltipOptions) { marker.tooltip = { text, options: tooltipOptions }; return marker; } });
      marker.on = (type, fn) => { marker.addEventListener(type, fn); return marker; };
      return marker;
    }
  };
}

function fakeMap() {
  const map = {
    groups: [],
    get markers() { return map.groups.flatMap(group => group.markers); },
    add(group) { map.groups.push(group); },
    removeLayer(group) { map.groups = map.groups.filter(item => item !== group); }
  };
  return map;
}

function target() {
  const handlers = new Map();
  return {
    addEventListener(type, fn) { handlers.set(type, [...(handlers.get(type) || []), fn]); },
    emit(type, event = {}) { (handlers.get(type) || []).forEach(fn => fn(event)); }
  };
}

function element(tag, options = {}) {
  const node = {
    tagName: tag.toUpperCase(),
    className: options.className || '',
    dataset: options.dataset || {},
    style: {},
    title: '',
    textContent: '',
    children: [],
    appendChild(child) { node.children.push(child); return child; },
    replaceChildren(...children) { node.children = children; },
    closest: selector => selector === '[data-map-slot="panel"]' ? options.closestOf || null :
      selector === '.map-panel-close' && node.className === 'map-panel-close' ? node : null,
    querySelector(selector) {
      if (selector === '[data-map-slot="panel"]') return options.slots && options.slots.panel || null;
      return node.querySelectorAll(selector)[0] || null;
    },
    querySelectorAll(selector) {
      if (selector === '.map-panel-monograms') return node.children.filter(child => child.className === 'map-panel-monograms');
      if (selector === '[data-writer-host]') return node.children.filter(child => child.dataset.writerHost);
      return [];
    }
  };
  return node;
}
