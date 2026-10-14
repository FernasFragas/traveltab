import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import vm from 'node:vm';

const source = readFileSync(new URL('../../../public/redesign/map-layers.js', import.meta.url), 'utf8');
const NAMES = ['notable', 'writers', 'plan', 'itineraries'];
const LABELS = ["Notable places", "Writers' pins", 'Your plan', "Writers' itineraries"];

test('upgrades the static chips into buttons with aria-pressed and keeps labels', () => {
  const app = browser();
  app.run();
  const chips = app.chips();
  assert.deepEqual(chips.map(c => c.tagName), ['BUTTON', 'BUTTON', 'BUTTON', 'BUTTON']);
  assert.deepEqual(chips.map(c => c.textContent), LABELS);
  assert.equal(chips[0].attrs['type'], 'button');
  assert.equal(chips[0].attrs['aria-pressed'], 'true');
  assert.match(chips[0].className, /writers-map-layer-notable/);
});

test('upgrading again after an HTMX swap does not duplicate chips', () => {
  const app = browser();
  app.run();
  app.document.emit('htmx:afterSwap');
  assert.equal(app.chips().length, 4);
});

test('a chip toggles only its own layer', () => {
  const app = browser({ registered: NAMES });
  app.run();
  app.calls.length = 0;
  app.chip('writers').click();
  assert.deepEqual(app.calls, [['writers', false]]);
  assert.equal(app.chip('writers').attrs['aria-pressed'], 'false');
  app.chip('writers').click();
  assert.deepEqual(app.calls.at(-1), ['writers', true]);
  assert.equal(app.chip('writers').attrs['aria-pressed'], 'true');
});

test('layers menu toggles the same flags and stays in sync with chips', () => {
  const app = browser({ registered: NAMES });
  app.run();
  const menu = app.window.TravelMapLayers.createMenu();
  const writers = menu.children.find(item => item.className.includes('writers-map-layer-writers'));
  writers.click();
  assert.deepEqual(app.calls.at(-1), ['writers', false]);
  assert.equal(app.chip('writers').attrs['aria-pressed'], 'false');
  app.chip('writers').click();
  assert.equal(writers.attrs['aria-pressed'], 'true');
});

test('all chips off is fine and never throws', () => {
  const app = browser({ registered: NAMES });
  app.run();
  app.document.emit('travelmap:plan-updated', { days: [{ places: [1] }] });
  for (const n of NAMES) app.chip(n).click();
  for (const n of NAMES) assert.equal(app.chip(n).attrs['aria-pressed'], 'false');
});

test('unregistered layers never throw on click or ready', () => {
  const app = browser({ registered: [] });
  app.run();
  assert.doesNotThrow(() => { app.chip('notable').click(); app.document.emit('travelmap:ready'); });
  assert.equal(app.chip('notable').attrs['aria-pressed'], 'false');
});

test('plan chip is disabled until a plan with days exists, then enabled', () => {
  const app = browser({ registered: NAMES });
  app.run();
  assert.equal(app.chip('plan').disabled, true);
  app.document.emit('travelmap:plan-updated', { days: [] });
  assert.equal(app.chip('plan').disabled, true);
  app.document.emit('travelmap:plan-updated', { days: [{ places: [] }] });
  assert.equal(app.chip('plan').disabled, false);
  app.document.emit('travelmap:plan-updated', { days: [] });
  assert.equal(app.chip('plan').disabled, true);
});

test('itineraries chip is hidden until that layer is registered', () => {
  const app = browser({ registered: ['notable', 'writers'] });
  app.run();
  assert.equal(app.chip('itineraries').hidden, true);
  app.registered.add('itineraries');
  app.document.emit('travelmap:ready');
  assert.equal(app.chip('itineraries').hidden, false);
});

test('remembered state is applied once the layer is registered', () => {
  const app = browser({ registered: [], stored: { notable: false } });
  app.run();
  assert.equal(app.chip('notable').attrs['aria-pressed'], 'false');
  app.registered.add('notable');
  app.document.emit('travelmap:ready');
  assert.deepEqual(app.calls.at(-1), ['notable', false]);
});

test('toggles are remembered in sessionStorage', () => {
  const app = browser({ registered: NAMES });
  app.run();
  app.chip('notable').click();
  assert.deepEqual(JSON.parse(app.store['travelmap.layers']), { notable: false });
});

test('blocked sessionStorage does not break the chips', () => {
  const app = browser({ registered: NAMES, blockedStorage: true });
  app.run();
  assert.doesNotThrow(() => app.chip('notable').click());
  assert.equal(app.chip('notable').attrs['aria-pressed'], 'false');
});

function element(tagName) {
  return {
    tagName: tagName.toUpperCase(), attrs: {}, children: [], className: '', textContent: '', hidden: false, disabled: false, listeners: {},
    setAttribute(k, v) { this.attrs[k] = String(v); },
    getAttribute(k) { return this.attrs[k] ?? null; },
    addEventListener(n, cb) { this.listeners[n] = cb; },
    appendChild(child) { this.children.push(child); },
    click() { if (!this.disabled) this.listeners.click?.({ currentTarget: this }); },
    replaceChild(next, old) { this.children[this.children.indexOf(old)] = next; },
    querySelectorAll(sel) { return sel === '.writers-map-layer' ? this.children.filter(child => child.className.includes('writers-map-layer')) : []; }
  };
}

function browser({ registered = NAMES, stored = null, blockedStorage = false } = {}) {
  const reg = new Set(registered);
  const calls = [];
  const store = stored ? { 'travelmap.layers': JSON.stringify(stored) } : {};
  const slot = element('DIV');
  NAMES.forEach((n, i) => {
    const span = element('SPAN');
    span.className = 'writers-map-layer writers-map-layer-' + n;
    span.textContent = LABELS[i];
    slot.children.push(span);
  });
  const handlers = {};
  const document = {
    readyState: 'complete',
    createElement: element,
    querySelectorAll: sel => (sel === '[data-map-slot="layers"]' ? [slot] : []),
    addEventListener(n, cb) { (handlers[n] ??= []).push(cb); },
    emit(n, detail) { (handlers[n] || []).forEach(cb => cb({ detail })); }
  };
  const window = { TravelMap: { setLayerEnabled(name, on) { if (!reg.has(name)) throw new Error('Unknown map layer: ' + name); calls.push([name, on]); } } };
  const sessionStorage = blockedStorage
    ? { getItem() { throw new Error('blocked'); }, setItem() { throw new Error('blocked'); } }
    : { getItem: k => store[k] ?? null, setItem: (k, v) => { store[k] = v; } };
  const context = vm.createContext({ document, window, sessionStorage, JSON });
  const chips = () => slot.children;
  return {
    document, window, calls, store, registered: reg, chips,
    chip: n => chips().find(c => c.className.includes('writers-map-layer-' + n)),
    run: () => vm.runInContext(source, context)
  };
}
