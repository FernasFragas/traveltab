import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import vm from 'node:vm';

const source = readFileSync(process.env.MAP_PANEL_SOURCE || new URL('../../../public/redesign/map-panel.js', import.meta.url), 'utf8');

test('writer card uses real itinerary day QIDs before mention count and deduplicates blog names', () => {
  const app = panel({ mentions: 'Mentioned by 3 writers', routes: [route('a.example', ['Q1']), route('a.example', ['Q1', 'Q2']), route('b.example', ['Q2'])] });
  app.select();
  app.swap();
  assert.match(app.mentioned.textContent, /Featured in 2 writers' itineraries/);
  assert.match(app.mentioned.textContent, /Including Alpha Travels$/);
  assert.equal(app.card.querySelector('[data-writer-posts]').getAttribute('aria-label'), "Go to writers' posts");
});

test('writer card preserves mentions when only other places feature in itineraries', () => {
  const app = panel({ mentions: 'Mentioned by 3 writers', routes: [route('a.example', ['Q2'])] });
  app.select();
  app.swap();
  assert.equal(app.mentioned.textContent, 'Mentioned by 3 writers');
});

test('writer itinerary names use existing source metadata before mention aliases or hosts', () => {
  const app = panel({ routes: [route('a.example', ['Q1']), route('b.example', ['Q1'])], writerSources: [['a.example', 'Official Alpha'], ['b.example', 'Official Beta']] });
  app.select();
  app.swap();
  assert.match(app.mentioned.textContent, /Including Official Alpha, Official Beta$/);
  assert.doesNotMatch(app.mentioned.textContent, /Alpha Travels|b\.example/);
});

test('writer card preserves no mentions state when no itinerary includes the place', () => {
  const app = panel({ mentions: 'No writer mentions yet', routes: [] });
  app.select();
  app.swap();
  assert.equal(app.mentioned.textContent, 'No writer mentions yet');
  assert.equal(app.card.querySelector('[data-writer-posts]'), null);
});

test('first pin callout shows the photo and area before its action opens the panel', () => {
  const app = panel();
  const pin = app.groups.get('notable')[0];
  pin.onClick();
  assert.equal(app.requests.length, 0, 'pin selection leaves panel loading to the callout action');
  const callout = app.popups.get('pin-popup').element;
  assert.ok(callout.querySelector('img'), 'first callout includes the existing place photo');
  assert.match(callout.textContent, /Old Town/);
  const action = callout.querySelector('button');
  assert.ok(action);
  action.click();
  assert.equal(app.requests.length, 1);
});

test('dragging the phone handle expands once and updates its accessible action', () => {
  const app = panel({ phone: true });
  app.select();
  app.swap();
  const handle = app.slot.querySelector('.map-sheet-handle');
  handle.dispatch('pointerdown', { clientY: 400, pointerId: 1 });
  handle.dispatch('pointerup', { clientY: 250, pointerId: 1 });
  assert.equal(handle.getAttribute('aria-expanded'), 'true');
  assert.equal(handle.getAttribute('aria-label'), 'Collapse place details');
  handle.click();
  assert.equal(handle.getAttribute('aria-expanded'), 'true', 'the click synthesized after dragging does not undo expansion');
});

test('a callout photo that fails to load is hidden instead of showing a broken image', () => {
  const app = panel();
  app.groups.get('notable')[0].onClick();
  const thumbnail = app.popups.get('pin-popup').element.querySelector('img');
  thumbnail.dispatch('error');
  assert.equal(thumbnail.hidden, true, 'popup thumbnail');
  app.select();
  const photo = app.groups.get('selected-callout')[0].element.querySelector('img');
  photo.dispatch('error');
  assert.equal(photo.hidden, true, 'selected pin photo');
});

test('a place without a photo opens a usable callout without an empty image', () => {
  const app = panel({ photo: false });
  app.groups.get('notable')[0].onClick();
  const callout = app.popups.get('pin-popup').element;
  assert.equal(callout.querySelector('img'), null);
  assert.match(callout.textContent, /Museum/);
  callout.querySelector('button').click();
  assert.equal(app.requests.length, 1);
});

test('opening a second pin callout leaves only the latest pin selected', () => {
  const app = panel({ secondPlace: true });
  const [first, second] = app.groups.get('notable');
  first.onClick();
  assert.ok(first.element.className.split(' ').includes('is-selected'));
  second.onClick();
  assert.ok(second.element.className.split(' ').includes('is-selected'));
  assert.equal(first.element.className.split(' ').includes('is-selected'), false);
});

test('selected callout gets its area from the loaded panel without adding a GeoJSON field', () => {
 const app = panel({ featureArea: '' }); app.select(); app.swap();
 assert.match(app.groups.get('selected-callout')[0].element.textContent, /Old Town/);
});

function panel({ featureArea = 'Old Town', mentions = 'No writer mentions yet', routes = [], phone = false, photo = true, secondPlace = false, writerSources = [] } = {}) {
  const listeners = new Map();
  const bodyListeners = new Map();
  const layers = new Map();
  const groups = new Map();
  const popups = new Map();
  const requests = [];
  const mentioned = node('div', 'map-panel-mentioned');
  mentioned.textContent = mentions;
  const card = node('div', 'map-panel-writer-card');
  card.appendChild(mentioned);
  const article = node('article', 'map-place-panel');
  article.dataset.placeArea = 'Old Town';
  article.appendChild(card);
  const link = node('a');
  link.setAttribute('data-writer-host', 'a.example');
  link.dataset = { writerHost: 'a.example', blogName: 'Alpha Travels' };
  article.appendChild(link);
  writerSources.forEach(([host, name]) => {
    const source = node('span');
    source.setAttribute('data-writer-source-host', host);
    source.dataset = { writerSourceHost: host, writerSourceName: name };
    article.appendChild(source);
  });
  const slot = node('div');
  slot.appendChild(article);
  const shell = node('div');
  shell.querySelector = () => slot;
  const feature = { properties: { qid: 'Q1', name: 'Museum', area: featureArea, kind: 'museum', photo_url: photo ? '/museum.jpg' : '', writer_count: 0, panel_url: '/map/city/Q1' }, geometry: { coordinates: [1, 2] } };
  const draw = { markers: (id, markers) => groups.set(id, markers), clear: id => { groups.delete(id); popups.delete(id); }, popup: (id, coordinates, element) => popups.set(id, { element }) };
  const features = [feature];
  if (secondPlace) features.push({ ...feature, properties: { ...feature.properties, qid: 'Q2', name: 'Second museum' } });
  const context = { features, itineraries: routes, draw, selectPlace: () => select() };
  const window = { innerWidth: phone ? 375 : 1440, TravelMap: { registerLayer: (name, layer) => layers.set(name, layer) }, htmx: { ajax: (...args) => requests.push(args) } };
  const document = { body: { addEventListener: (name, callback) => bodyListeners.set(name, callback) }, addEventListener: (name, callback) => listeners.set(name, callback), createElement: tag => node(tag), querySelector: selector => selector === '[data-writers-map]' ? shell : null };
  vm.runInNewContext(source, { window, document, setTimeout, URL });
  layers.get('notable').add({}, context);
  function select() { listeners.get('travelmap:place-selected')({ detail: { place: feature } }); }
  return { mentioned, card, groups, popups, requests, slot, select, swap: () => bodyListeners.get('htmx:afterSwap')({ detail: { target: slot } }) };
}

function route(host, qids) {
  return { writer_host: host, days: [{ qids }] };
}

function node(tag, className = '') {
  const attrs = {};
  const handlers = new Map();
  let ownText = '';
  const element = {
    tagName: tag.toUpperCase(), className, children: [], dataset: {}, style: {},
    get textContent() { return ownText + this.children.map(child => child.textContent).join(''); },
    set textContent(value) { ownText = value; this.children = []; },
    appendChild(child) { this.children.push(child); return child; },
    prepend(child) { this.children.unshift(child); },
    setAttribute(name, value) { attrs[name] = value; },
    removeAttribute(name) { delete attrs[name]; },
    getAttribute(name) { return attrs[name]; },
    addEventListener(name, callback) { handlers.set(name, callback); },
    click() { handlers.get('click')?.(); },
    dispatch(name, event) { handlers.get(name)?.(event); },
    setPointerCapture() {},
    focus() {},
    querySelector(selector) { return this.querySelectorAll(selector)[0] || null; },
    querySelectorAll(selector) {
      const matches = child => selector[0] === '.' ? child.className.split(' ').includes(selector.slice(1)) : selector[0] === '[' ? child.getAttribute(selector.slice(1, -1)) !== undefined : child.tagName.toLowerCase() === selector;
      return this.children.flatMap(child => [...(matches(child) ? [child] : []), ...child.querySelectorAll(selector)]);
    }
  };
  element.classList = {
    add(...names) { element.className = [...new Set([...element.className.split(' '), ...names])].join(' '); },
    remove(...names) { element.className = element.className.split(' ').filter(value => !names.includes(value)).join(' '); },
    toggle(name, force) {
      const on = force === undefined ? !element.className.split(' ').includes(name) : force;
      element.className = element.className.split(' ').filter(value => value !== name).concat(on ? [name] : []).join(' ');
      return on;
    }
  };
  return element;
}

test('phone sheet keeps Hours summary in the keyboard focus cycle after Directions', () => {
  let onKey;
  let focused;
  const handle = { focus() { focused = 'handle'; } };
  const directions = {};
  const summary = {};
  const sheet = { querySelectorAll(selector) { return selector.includes('summary') ? [handle, directions, summary] : [handle, directions]; } };
  const document = { activeElement: directions, querySelector: () => sheet,
    body: { addEventListener() {} }, addEventListener(name, fn) { if (name === 'keydown') onKey = fn; } };
  vm.runInNewContext(source, { document, window: { TravelMap: { registerLayer() {} } } });
  let prevented = false;
  onKey({ key: 'Tab', preventDefault() { prevented = true; } });
  assert.equal(prevented, false, 'Tab from Directions must reach the following Hours summary');
  document.activeElement = summary;
  onKey({ key: 'Tab', preventDefault() { prevented = true; } });
  assert.equal(prevented, true);
  assert.equal(focused, 'handle', 'Tab from the final summary wraps to the sheet handle');
});
