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

test('draws purple writer quills above terracotta notable pins', () => {
  const app = browser();
  app.run();
  const map = app.addLayers();
  const writers = map.markers('writers');
  const notable = map.markers('notable');
  assert.deepEqual(writers.map(marker => marker.title), ['Pico Ruivo', 'Cabo Girão']);
  assert.equal(writers[0].element.children[0].className, 'bi bi-feather');
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

test('a pin whose name has not loaded yet is a neutral marker: no label, no QID, and it fills in on the next draw', () => {
  const nameless = feature('Q777', '', 0, 999);
  const app = browser({ features: [nameless, ...features()] });
  app.run();
  const map = app.addLayers();
  const pin = map.markers('notable').find(marker => marker.element.getAttribute('aria-label') === 'Loading name…');
  assert.ok(pin, 'a pin with the loading title exists');
  assert.equal(pin.title, 'Loading name…');
  assert.equal(pin.element.children.filter(child => child.className === 'map-pin-label').length, 0, 'the biggest place gets no label while nameless');
  const labelled = [...map.markers('notable'), ...map.markers('writers')]
    .flatMap(marker => marker.element.children.filter(child => child.className === 'map-pin-label').map(child => child.textContent));
  assert.ok(labelled.includes('Funchal'), 'the next biggest places are labelled instead');
  assert.ok(![pin.title, pin.element.title, ...labelled].some(text => text.includes('Q777')));
});

test('a pin click opens a popup with the name, the writer count and the blog monograms, and does not open the panel', async () => {
  const app = browser();
  app.run();
  const map = app.addLayers();
  map.markers('writers').find(marker => marker.title === 'Pico Ruivo').onClick();
  const popup = map.popups.get('pin-popup');
  assert.deepEqual(popup.lngLat, [-16.9, 32.7]);
  assert.equal(popup.element.attributes.role, 'dialog');
  assert.equal(popup.element.attributes['aria-label'], 'Pico Ruivo');
  assert.deepEqual(app.textOf(popup.element), ['Pico Ruivo']);
  assert.deepEqual(app.selected, []);
  await app.flush();
  assert.deepEqual(app.requests.map(request => request[0]), ['/map/madeira-pt/Q1?lat=1&lon=2']);
});

test('a popup says "1 writer" for one writer and shows the place kind for a notable pin', () => {
  const app = browser();
  app.run();
  const map = app.addLayers();
  map.markers('writers').find(marker => marker.title === 'Cabo Girão').onClick();
  assert.deepEqual(app.textOf(map.popups.get('pin-popup').element), ['Cabo Girão']);
  map.markers('notable').find(marker => marker.title === 'Funchal').onClick();
  assert.equal(map.popups.size, 1, 'one popup at a time');
  assert.deepEqual(app.textOf(map.popups.get('pin-popup').element), ['Funchal']);
  assert.equal(app.requests.length, 2, 'panel metadata comes from existing place fragments');
});

test('the popup takes focus on its Details button and has no blog text beyond names', () => {
  const app = browser();
  app.run();
  const map = app.addLayers();
  map.markers('notable').find(marker => marker.title === 'Seixal').onClick();
  const details = app.find(map.popups.get('pin-popup').element, node => node.tagName === 'BUTTON')[0];
  assert.equal(details.textContent, '›');
  assert.equal(details.getAttribute('aria-label'), 'Details');
  assert.equal(app.focused, details);
  assert.equal(app.textOf(map.popups.get('pin-popup').element).join(' ').includes('quote'), false);
});

test('Details opens the place panel and closes the popup', () => {
  const app = browser();
  app.run();
  const map = app.addLayers();
  map.markers('notable').find(marker => marker.title === 'Seixal').onClick();
  app.find(map.popups.get('pin-popup').element, node => node.tagName === 'BUTTON')[0].click();
  assert.deepEqual(app.selected, ['Q3']);
  assert.equal(map.popups.size, 0);
});

test('Escape closes the popup and returns focus to the pin', () => {
  const app = browser();
  app.run();
  const map = app.addLayers();
  const marker = map.markers('notable').find(marker => marker.title === 'Seixal');
  marker.onClick();
  app.keydown('Enter');
  assert.equal(map.popups.size, 1, 'other keys leave it open');
  app.keydown('Escape');
  assert.equal(map.popups.size, 0);
  assert.equal(app.focused, marker.element);
  app.keydown('Escape');
  assert.equal(app.focused, marker.element, 'Escape with no popup does nothing');
});

test('a popup the engine closed (click elsewhere) is not closed again and takes no focus', () => {
  const app = browser();
  app.run();
  const map = app.addLayers();
  const marker = map.markers('notable').find(marker => marker.title === 'Seixal');
  marker.onClick();
  app.focused = null;
  map.closeFromEngine('pin-popup');
  app.keydown('Escape');
  assert.equal(app.focused, null);
});

test('removing the pin layer closes its popup', () => {
  const app = browser();
  app.run();
  const map = app.addLayers();
  map.markers('notable').find(marker => marker.title === 'Seixal').onClick();
  app.layers.get('notable').remove(map, app.context);
  assert.equal(map.popups.size, 0);
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

test('writer card arrow scrolls to the posts', () => {
 const app = browser(); app.run(); let scrolled = false;
 const root = { querySelector: () => ({ scrollIntoView() { scrolled = true; } }) };
 const arrow = { closest: selector => selector === '[data-writer-posts]' ? arrow : selector === '.map-place-panel' ? root : null };
 app.body.emit('click', { target: arrow }); assert.equal(scrolled, true);
});

test('Share copies a link to the selected place when Web Share is unavailable', async () => {
  let copied = '';
  const app = browser({ navigator: { clipboard: { writeText: text => { copied = text; return Promise.resolve(); } } } });
  app.run();
  const feedback = { textContent: '' };
  const root = { dataset: { placeQid: 'Q1' }, querySelector: selector => selector === '.map-panel-feedback' ? feedback : { textContent: 'Pico Ruivo' } };
  const share = { closest: selector => selector === '[data-map-share]' ? share : selector === '.map-place-panel' ? root : null };
  app.body.emit('click', { target: share });
  await app.flush();
  assert.match(copied, /[?&]place=Q1/);
  assert.equal(feedback.textContent, 'Link copied');
});

test('does nothing without the map core', () => {
  const app = browser({ core: false });
  assert.doesNotThrow(() => app.run());
  assert.equal(app.layers.size, 0);
});

function features() {
  return [
    feature('Q1', 'Pico Ruivo', 2, 45),
    feature('Q2', 'Funchal', 0, 92, 'City'),
    feature('Q3', 'Seixal', 0, 14),
    feature('Q4', 'Cabo Girão', 1, 31)
  ];
}

function extraPlaces(count) {
  return Array.from({ length: count }, (_, index) => feature('Q9' + index, 'Place ' + index, 0, 40 + index));
}

function feature(qid, name, writers, sitelinks, kind) {
  return {
    type: 'Feature',
    geometry: { type: 'Point', coordinates: [-16.9, 32.7] },
    properties: { qid, name, kind, writer_count: writers, sitelinks, panel_url: '/map/madeira-pt/' + qid + '?lat=1&lon=2' }
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
  const focus = { current: null };
  const writersOf = [['travel.example', 'Salt in our Hair'], ['viagens.example', 'Viagens à Solta'], ['c.example', 'Casa'], ['c.example', 'Casa'], ['d.example', 'Dias']];
  const window = {
    location: { href: 'https://traveltab.example/trip/madeira-pt' },
    navigator: options.navigator || {},
    fetch: url => { requests.push([url]); return Promise.resolve({ ok: true, text: () => Promise.resolve(JSON.stringify(writersOf)) }); },
    DOMParser: class { parseFromString(text) { return { querySelector: () => null, querySelectorAll: () => JSON.parse(text).map(([host, blog]) => element('a', { dataset: { writerHost: host, blogName: blog } })) }; } },
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
    createElement: tag => element(tag, { focus })
  };
  const app = {
    layers, selected, requests, body, panel, context, monograms: null,
    get focused() { return focus.current; },
    set focused(node) { focus.current = node; },
    keydown: key => listeners.get('keydown')({ key }),
    flush: async () => { await new Promise(resolve => setImmediate(resolve)); await new Promise(resolve => setImmediate(resolve)); },
    find: (node, test) => [node, ...node.children.flatMap(child => app.find(child, test))].filter(test),
    textOf: node => app.find(node, item => item.textContent && item.tagName !== 'BUTTON' && !item.className.includes('monogram')).map(item => item.textContent),
    run: () => vm.runInNewContext(source, { window, document, ...window, URL, setTimeout }),
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
  const popups = new Map();
  const drawn = [];
  return {
    drawn, popups,
    markers: id => groups.get(id) || [],
    closeFromEngine(id) { const popup = popups.get(id); popups.delete(id); popup.options.onClose(); },
    draw: {
      markers(id, items) { drawn.push({ method: 'markers', id }); groups.set(id, items); },
      popup(id, lngLat, element, options) { drawn.push({ method: 'popup', id }); popups.set(id, { lngLat, element, options }); },
      clear(id) { drawn.push({ method: 'clear', id }); groups.delete(id); popups.delete(id); }
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
    type: '',
    attributes: {},
    handlers: {},
    textContent: '',
    children: [],
    appendChild(child) { node.children.push(child); return child; },
    addEventListener(type, fn) { node.handlers = { ...node.handlers, [type]: fn }; },
    click() { node.handlers.click(); },
    focus() { if (options.focus) options.focus.current = node; },
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
