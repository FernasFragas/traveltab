import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import vm from 'node:vm';

const source = readFileSync(new URL('../../../public/redesign/map-dayplan.js', import.meta.url), 'utf8');

test('registers one layer named plan with the core', () => {
  const app = browser();
  app.run();
  assert.deepEqual(app.registered, ['plan']);
});

test('does nothing when the core or the map shell is missing', () => {
  const noCore = browser({ core: false });
  noCore.run();
  assert.deepEqual(noCore.registered, []);

  const noShell = browser({ shell: false });
  noShell.run();
  noShell.swap([[['A', 1, 2]]]);
  assert.equal(noShell.updates.length, 0);
});

test('reads stop coordinates after a swap and reports them to the core', () => {
  const app = browser();
  app.run();
  app.swap([[['Tower', 38.69, -9.21], ['Museum', 38.72, -9.11]], [['Park', 38.7, -9.1]]]);
  assert.deepEqual(JSON.parse(JSON.stringify(app.updates.at(-1))), [
    { number: 1, stops: [{ name: 'Tower', lat: 38.69, lon: -9.21 }, { name: 'Museum', lat: 38.72, lon: -9.11 }] },
    { number: 2, stops: [{ name: 'Park', lat: 38.7, lon: -9.1 }] }
  ]);
});

test('draws green numbered pins per day', () => {
  const app = browser();
  app.run();
  app.mount();
  app.swap([[['Tower', 38.69, -9.21], ['Museum', 38.72, -9.11]], [['Park', 38.7, -9.1]]]);
  const pins = app.pins();
  assert.equal(pins.length, 3);
  assert.match(pins[0].icon.html, /plan-pin/);
  assert.match(pins[0].icon.html, />1</);
  assert.match(pins[1].icon.html, />2</);
  assert.match(pins[2].icon.html, />1</);
  assert.match(pins[0].icon.html, /#2e7d32/);
});

test('regenerating and clearing leave no duplicate pins', () => {
  const app = browser();
  app.run();
  app.mount();
  app.swap([[['A', 1, 2], ['B', 3, 4]]]);
  app.swap([[['A', 1, 2], ['B', 3, 4]]]);
  assert.equal(app.pins().length, 2);
  assert.equal(app.maps[0].groups.length, 1);
  app.swap([]);
  assert.equal(app.pins().length, 0);
  assert.equal(app.maps[0].controls.filter(c => c.attached).length, 0);
});

test('day toggle shows one day or all days', () => {
  const app = browser();
  app.run();
  app.mount();
  app.swap([[['A', 1, 2]], [['B', 3, 4], ['C', 5, 6]]]);
  const buttons = app.maps[0].controls.find(c => c.attached).buttons();
  assert.deepEqual(buttons.map(b => b.textContent), ['All', 'Day 1', 'Day 2']);
  buttons[2].click();
  assert.equal(app.pins().length, 2);
  buttons[1].click();
  assert.equal(app.pins().length, 1);
  buttons[0].click();
  assert.equal(app.pins().length, 3);
});

test('keeps the selected day across unrelated swaps and an unchanged plan', () => {
  const app = browser();
  app.run();
  app.mount();
  const plan = [[['A', 1, 2]], [['B', 3, 4], ['C', 5, 6]]];
  app.swap(plan);
  app.maps[0].controls.find(c => c.attached).buttons()[2].click();
  assert.equal(app.pins().length, 2);
  app.swap(plan);
  assert.equal(app.pins().length, 2);
  const pressed = app.maps[0].controls.find(c => c.attached).buttons().map(b => b.attrs['aria-pressed']);
  assert.deepEqual(pressed, ['false', 'false', 'true']);
});

test('resets to all days when the plan changes', () => {
  const app = browser();
  app.run();
  app.mount();
  app.swap([[['A', 1, 2]], [['B', 3, 4]]]);
  app.maps[0].controls.find(c => c.attached).buttons()[2].click();
  assert.equal(app.pins().length, 1);
  app.swap([[['A', 1, 2]], [['B', 3, 4]], [['C', 5, 6]]]);
  assert.equal(app.pins().length, 3);
});

test('disabling the layer removes pins and enabling restores them', () => {
  const app = browser();
  app.run();
  app.mount();
  app.swap([[['A', 1, 2]]]);
  app.setEnabled(false);
  assert.equal(app.pins().length, 0);
  app.setEnabled(true);
  assert.equal(app.pins().length, 1);
});

test('skips stops without usable coordinates', () => {
  const app = browser();
  app.run();
  app.mount();
  app.swap([[['A', 0, 0], ['B', 'x', 4], ['C', 5, 6]]]);
  assert.equal(app.pins().length, 1);
});

function browser({ core = true, shell = true } = {}) {
  const registered = [];
  const updates = [];
  const maps = [];
  const listeners = new Map();
  let layer = null;
  let enabled = true;
  let days = [];
  let currentMap = null;

  const stopEl = ([name, lat, lon]) => ({ dataset: { name, lat: String(lat), lon: String(lon) } });
  const document = {
    body: { addEventListener: (n, cb) => listeners.set(n, cb) },
    createElement: tag => element(tag),
    querySelector: sel => (shell && sel === '[data-writers-map]' ? {} : null),
    querySelectorAll: sel => sel === '[data-plan-day]' ? days : [],
    addEventListener(n, cb) { listeners.set('doc:' + n, cb); }
  };
  document.dispatchEvent = event => listeners.get('doc:' + event.type)?.(event);
  const L = {
    divIcon: options => options,
    marker: (point, options) => ({ point, icon: options.icon }),
    layerGroup: () => ({
      layers: [],
      addLayer(l) { this.layers.push(l); return this; },
      removeLayer(l) { this.layers = this.layers.filter(x => x !== l); return this; },
      clearLayers() { this.layers = []; return this; },
      addTo(map) { map.groups.push(this); this.map = map; return this; },
      remove() { this.map.groups = this.map.groups.filter(g => g !== this); }
    }),
    control: options => ({
      options, attached: false,
      addTo(map) { this.el = this.onAdd(map); this.attached = true; map.controls.push(this); return this; },
      remove() { this.attached = false; },
      buttons() { return this.el.children; }
    }),
    DomEvent: { disableClickPropagation() {} }
  };
  const window = { L };
  if (core) {
    window.TravelMap = {
      registerLayer(name, l) { registered.push(name); layer = l; },
      setLayerEnabled(name, on) {
        if (on === enabled) return;
        enabled = on;
        if (on) layer.add(currentMap, {}); else layer.remove(currentMap, {});
      },
      updatePlan(d) {
        updates.push(d);
        document.dispatchEvent({ type: 'travelmap:plan-updated', detail: { days: d } });
      }
    };
  }
  const vmContext = vm.createContext({ document, window, CustomEvent: class {} });
  return {
    registered, updates, maps,
    run: () => vm.runInContext(source, vmContext),
    mount() {
      currentMap = { groups: [], controls: [] };
      maps.push(currentMap);
      layer.add(currentMap, {});
    },
    swap(dayStops) {
      days = dayStops.map((stops, i) => {
        const els = stops.map(stopEl);
        return { dataset: { planDay: String(i + 1) }, querySelectorAll: () => els };
      });
      listeners.get('htmx:afterSwap')?.({});
    },
    setEnabled: on => window.TravelMap.setLayerEnabled('plan', on),
    pins: () => maps[0].groups.flatMap(g => g.layers)
  };
}

function element(tagName) {
  const listeners = new Map();
  return {
    tagName: tagName.toUpperCase(), children: [], dataset: {}, className: '', textContent: '', style: {},
    attrs: {}, setAttribute(n, v) { this.attrs[n] = v; }, appendChild(c) { this.children.push(c); return c; },
    addEventListener(n, cb) { listeners.set(n, cb); },
    click() { listeners.get('click')({ preventDefault() {}, stopPropagation() {} }); }
  };
}
