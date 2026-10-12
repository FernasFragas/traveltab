import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import vm from 'node:vm';

const source = readFileSync(new URL('../../../public/redesign/map-dayplan.js', import.meta.url), 'utf8');

test('registers one layer named plan and reports valid stops after swaps', () => {
  const app = browser();
  app.run();
  assert.deepEqual(app.registered, ['plan']);
  app.swap([[['Tower', 38.69, -9.21], ['Invalid', 0, 0]]]);
  assert.deepEqual(JSON.parse(JSON.stringify(app.updates.at(-1))), [
    { number: 1, stops: [{ name: 'Tower', lat: 38.69, lon: -9.21 }] }
  ]);
});

test('does nothing when the core or map shell is missing', () => {
  const noCore = browser({ core: false });
  noCore.run();
  assert.deepEqual(noCore.registered, []);
  const noShell = browser({ shell: false });
  noShell.run();
  noShell.swap([[['A', 1, 2]]]);
  assert.equal(noShell.updates.length, 0);
});

test('uses drawing API for green, numbered, focusable pins and a day control', () => {
  const app = browser();
  app.run();
  app.mount();
  app.swap([[['Tower', 38.69, -9.21], ['Museum', 38.72, -9.11]], [['Park', 38.7, -9.1]]]);
  assert.deepEqual(JSON.parse(JSON.stringify(app.pins().map(pin => pin.lngLat))), [[-9.21, 38.69], [-9.11, 38.72], [-9.1, 38.7]]);
  assert.deepEqual(Array.from(app.pins(), pin => pin.element.textContent), ['1', '2', '1']);
  assert.ok(app.pins().every(pin => pin.element.tagName === 'BUTTON' && pin.element.className === 'plan-pin'));
  assert.ok(app.pins().every(pin => pin.element.style.background === '#2e7d32'));
  assert.match(app.pins()[0].title, /Day 1: Tower/);
  assert.deepEqual(Array.from(app.control().buttons(), button => button.textContent), ['All', 'Day 1', 'Day 2']);
  assert.equal(app.control().position, 'top-right');
  assert.deepEqual(app.drawCalls().map(call => call.method).slice(-2), ['markers', 'control']);
});

test('generate, regenerate, day toggle, clear and enable leave one current set', () => {
  const app = browser();
  app.run();
  app.mount();
  const plan = [[['A', 1, 2]], [['B', 3, 4], ['C', 5, 6]]];
  app.swap(plan);
  assert.equal(app.pins().length, 3);
  app.control().buttons()[2].click();
  assert.equal(app.pins().length, 2);
  app.swap(plan);
  assert.equal(app.pins().length, 2);
  assert.deepEqual(app.control().buttons().map(button => button.attrs['aria-pressed']), ['false', 'false', 'true']);
  app.swap([[['D', 7, 8]], [['E', 9, 10]]]);
  assert.deepEqual(Array.from(app.pins(), pin => pin.title), ['Day 1: D', 'Day 2: E']);
  assert.deepEqual(app.control().buttons().map(button => button.attrs['aria-pressed']), ['true', 'false', 'false']);
  app.swap([]);
  assert.equal(app.pins().length, 0);
  assert.equal(app.control(), null);
  app.swap(plan);
  app.setEnabled(false);
  assert.equal(app.pins().length, 0);
  assert.equal(app.control(), null);
  app.setEnabled(true);
  assert.equal(app.pins().length, 3);
  assert.equal(app.control().buttons().length, 3);
});

test('re-adding on another map context draws once and clears the old context', () => {
  const app = browser();
  app.run();
  app.mount();
  app.swap([[['A', 1, 2]]]);
  const old = app.context();
  app.remount();
  assert.equal(old.markers.size, 0);
  assert.equal(app.pins().length, 1);
  assert.equal(app.context().markers.size, 1);
});

test('View on map activates the plan chip and scrolls to the shell', () => {
  const app = browser();
  app.run();
  app.viewOnMap();
  assert.equal(app.chipPressed(), 'true');
  assert.equal(app.scrolls(), 1);
});

function browser({ core = true, shell = true } = {}) {
  const registered = [];
  const updates = [];
  const listeners = new Map();
  let layer = null;
  let enabled = true;
  let days = [];
  let drawing = drawContext();
  let scrollCount = 0;
  let chip = element('button');
  chip.setAttribute('aria-pressed', 'false');
  chip.addEventListener('click', () => chip.setAttribute('aria-pressed', 'true'));
  const shellElement = { scrollIntoView() { scrollCount++; } };
  const document = {
    body: { addEventListener: (n, cb) => listeners.set(n, cb) },
    createElement: tag => element(tag),
    querySelector: sel => sel === '[data-writers-map]' && shell ? shellElement : sel === '.writers-map-layer-plan' ? chip : null,
    querySelectorAll: sel => sel === '[data-plan-day]' ? days : [],
    addEventListener(n, cb) { listeners.set('doc:' + n, cb); }
  };
  document.dispatchEvent = event => listeners.get('doc:' + event.type)?.(event);
  const window = {};
  if (core) window.TravelMap = {
    registerLayer(name, value) { registered.push(name); layer = value; },
    setLayerEnabled(name, on) {
      if (on === enabled) return;
      enabled = on;
      if (on) layer.add({}, { draw: drawing.draw });
      else layer.remove({}, { draw: drawing.draw });
    },
    updatePlan(value) {
      updates.push(value);
      document.dispatchEvent({ type: 'travelmap:plan-updated', detail: { days: value } });
    }
  };
  const vmContext = vm.createContext({ document, window, CustomEvent: class {} });
  return {
    registered, updates,
    run: () => vm.runInContext(source, vmContext),
    mount() { layer.add({}, { draw: drawing.draw }); },
    remount() {
      layer.remove({}, { draw: drawing.draw });
      drawing = drawContext();
      layer.add({}, { draw: drawing.draw });
    },
    context: () => drawing,
    swap(dayStops) {
      days = dayStops.map((stops, i) => ({
        dataset: { planDay: String(i + 1) },
        querySelectorAll: () => stops.map(([name, lat, lon]) => ({ dataset: { name, lat: String(lat), lon: String(lon) } }))
      }));
      listeners.get('htmx:afterSwap')?.({});
    },
    setEnabled: on => window.TravelMap.setLayerEnabled('plan', on),
    pins: () => drawing.markers.get('plan') || [],
    control: () => drawing.controls.get('plan') || null,
    drawCalls: () => drawing.calls,
    viewOnMap() { listeners.get('doc:click')?.({ target: { closest: selector => selector === '[data-view-on-map]' } }); },
    chipPressed: () => chip.attrs['aria-pressed'],
    scrolls: () => scrollCount
  };
}

function drawContext() {
  const markers = new Map();
  const controls = new Map();
  const calls = [];
  return {
    markers, controls, calls,
    draw: {
      markers(id, items) { calls.push({ method: 'markers', id }); markers.set(id, items); },
      control(id, element, position) { calls.push({ method: 'control', id }); controls.set(id, { element, position, buttons: () => element.children }); },
      clear(id) { calls.push({ method: 'clear', id }); markers.delete(id); controls.delete(id); }
    }
  };
}

function element(tagName) {
  const listeners = new Map();
  return {
    tagName: tagName.toUpperCase(), children: [], className: '', textContent: '', style: {}, attrs: {},
    setAttribute(n, v) { this.attrs[n] = v; },
    getAttribute(n) { return this.attrs[n]; },
    appendChild(child) { this.children.push(child); return child; },
    addEventListener(n, cb) { listeners.set(n, cb); },
    click() { listeners.get('click')?.({ preventDefault() {}, stopPropagation() {} }); }
  };
}
