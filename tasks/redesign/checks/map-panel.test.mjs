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
  const writers = map.markers('writers');
  const notable = map.markers('notable');
  assert.deepEqual(writers.map(marker => marker.title), ['Pico Ruivo', 'Cabo Girão']);
  assert.equal(writers[0].element.children[0].textContent, '2');
  assert.ok(writers.every(marker => marker.element.className.includes('map-pin-writer')));
  assert.deepEqual(notable.map(marker => marker.title), ['Funchal', 'Seixal']);
  assert.ok(notable.every(marker => marker.element.className.includes('map-pin-notable')));
  assert.ok(writers.every(marker => marker.z > notable[0].z));
  assert.ok(map.drawn.every(call => call.method === 'markers'));
});

test('labels only the biggest places', () => {
  const app = browser({ features: [...features(), ...extraPlaces(6)] });
  app.run();
  const map = app.addLayers();
  const labelled = [...map.markers('notable'), ...map.markers('writers')]
    .flatMap(marker => marker.element.children.filter(child => child.className === 'map-pin-label').map(child => child.textContent));
  assert.equal(labelled.length, 5);
  assert.ok(labelled.includes('Funchal'));
  assert.ok(!labelled.includes('Seixal'));
});

test('selects a place when its pin is clicked', () => {
  const app = browser();
  app.run();
  const map = app.addLayers();
  map.markers('notable').find(marker => marker.title === 'Seixal').onClick();
  assert.deepEqual(app.selected, ['Q3']);
});

test('pins have accessible titles, keyboard focus and a visible focus style', () => {
  const app = browser();
  app.run();
  const marker = app.addLayers().markers('writers')[0];
  assert.equal(marker.element.tagName, 'BUTTON');
  assert.equal(marker.element.type, 'button');
  assert.equal(marker.element.title, marker.title);
  assert.equal(marker.element.getAttribute('aria-label'), marker.title);
  assert.match(readFileSync(new URL('../../../public/redesign/map-panel.css', import.meta.url), 'utf8'), /\.map-pin:focus-visible/);
});

test('removing a layer takes its pins off the map and adding it again leaves one copy', () => {
  const app = browser();
  app.run();
  const map = app.addLayers();
  app.layers.get('writers').remove(map, app.context);
  assert.equal(map.markers('writers').length, 0);
  assert.equal(map.markers('notable').length, 2);
  app.layers.get('writers').add(map, app.context);
  assert.equal(map.markers('writers').length, 2);
  assert.deepEqual(map.drawn.slice(-2).map(call => call.method), ['clear', 'markers']);
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
  const map = fakeMap();
  const context = { features: options.features || features(), selectPlace: qid => selected.push(qid), shell, draw: map.draw };
  const window = {
    L: new Proxy({}, { get() { throw new Error('pin layer must not access Leaflet'); } }),
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

function fakeMap() {
  const groups = new Map();
  const drawn = [];
  return {
    drawn,
    markers: id => groups.get(id) || [],
    draw: {
      markers(id, items) { drawn.push({ method: 'markers', id }); groups.set(id, items); },
      clear(id) { drawn.push({ method: 'clear', id }); groups.delete(id); }
    }
  };
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
    attributes: {},
    textContent: '',
    children: [],
    appendChild(child) { node.children.push(child); return child; },
    setAttribute(name, value) { node.attributes[name] = value; if (name === 'title') node.title = value; },
    getAttribute(name) { return node.attributes[name]; },
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
