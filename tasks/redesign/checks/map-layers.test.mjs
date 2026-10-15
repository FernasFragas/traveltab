import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import vm from 'node:vm';

const source = readFileSync(new URL('../../../public/redesign/map-layers.js', import.meta.url), 'utf8');
const NAMES = ['notable', 'writers', 'plan', 'itineraries'];
const LABELS = ["Notable places", "Writers' pins", 'Your plan', "Itineraries"];

test('upgrades every chip into a button with aria-pressed and keeps labels', () => {
  const app = browser();
  app.run();
  const chips = app.chips();
  assert.deepEqual(chips.map(c => c.tagName), ['BUTTON', 'BUTTON', 'BUTTON', 'BUTTON']);
  assert.deepEqual(chips.map(c => c.textContent), LABELS);
  assert.equal(chips[0].attrs['type'], 'button');
  assert.equal(chips[0].attrs['aria-pressed'], 'true');
  assert.match(chips[0].className, /writers-map-layer-notable/);
});

// As a traveller, I want to hide the notable places too, so that I can see only the writers' pins or my plan.
test('notable places can be switched off and on again', () => {
  const app = browser({ registered: NAMES });
  app.run();
  app.chip('notable').click();
  assert.deepEqual(app.calls.at(-1), ['notable', false]);
  assert.equal(app.chip('notable').attrs['aria-pressed'], 'false');
  app.chip('notable').click();
  assert.deepEqual(app.calls.at(-1), ['notable', true]);
  assert.equal(app.chip('notable').attrs['aria-pressed'], 'true');
});

// As a traveller, I want a newly loaded destination to show only its notable places, so that the map starts uncluttered.
test('a loaded destination starts with only notable places on', () => {
  const app = browser({ registered: NAMES });
  app.run();
  assert.deepEqual(app.chips().map(c => c.attrs['aria-pressed']), ['true', 'false', 'false', 'false']);
  assert.deepEqual(app.calls, [['notable', true], ['writers', false], ['plan', false], ['itineraries', false]]);
});

test('loading another destination resets the chips to notable only', () => {
  const app = browser({ registered: NAMES });
  app.run();
  app.chip('notable').click();
  app.chip('writers').click();
  app.renderChips();
  app.document.emit('htmx:afterSwap');
  assert.deepEqual(app.chips().map(c => c.attrs['aria-pressed']), ['true', 'false', 'false', 'false']);
  assert.deepEqual(app.calls.slice(-4), [['notable', true], ['writers', false], ['plan', false], ['itineraries', false]]);
});

test('other swaps on the same destination keep the chosen chips', () => {
  const app = browser({ registered: NAMES });
  app.run();
  app.chip('writers').click();
  app.document.emit('htmx:afterSwap');
  app.document.emit('travelmap:ready');
  assert.equal(app.chip('writers').attrs['aria-pressed'], 'true');
});

test('notable is filled terracotta only while on; on overlays are white', () => {
  const css = readFileSync(new URL('../../../public/redesign/map-layers.css', import.meta.url), 'utf8');
  assert.match(css, /button\.writers-map-layer-notable\[aria-pressed="true"\]\s*\{[^}]*background:\s*var\(--tt-accent\)/);
  assert.doesNotMatch(css, /button\.writers-map-layer\[aria-pressed="true"\]\s*\{[^}]*background:\s*var\(--tt-accent\)/);
  assert.doesNotMatch(css, /writers-map-layer-base/);
});

test('phone keeps full labels and outline/filled state', () => {
  const app = browser();
  app.window.innerWidth = 375;
  app.run();
  assert.deepEqual(app.chips().map(c => c.textContent), LABELS);
  assert.match(readFileSync(new URL('../../../public/redesign/map-layers.css', import.meta.url), 'utf8'), /aria-pressed="true"/);
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
  assert.deepEqual(app.calls, [['writers', true]]);
  assert.equal(app.chip('writers').attrs['aria-pressed'], 'true');
  app.chip('writers').click();
  assert.deepEqual(app.calls.at(-1), ['writers', false]);
  assert.equal(app.chip('writers').attrs['aria-pressed'], 'false');
});

test('layers menu toggles the same flags and stays in sync with chips', () => {
  const app = browser({ registered: NAMES });
  app.run();
  const menu = app.window.TravelMapLayers.createMenu();
  const writers = menu.children.find(item => item.className.includes('writers-map-layer-writers'));
  writers.click();
  assert.deepEqual(app.calls.at(-1), ['writers', true]);
  assert.equal(app.chip('writers').attrs['aria-pressed'], 'true');
  app.chip('writers').click();
  assert.equal(writers.attrs['aria-pressed'], 'false');
});

test('all chips off is fine and never throws', () => {
  const app = browser({ registered: NAMES });
  app.run();
  app.document.emit('travelmap:plan-updated', { days: [{ places: [1] }] });
  app.chip('notable').click();
  for (const n of NAMES) assert.equal(app.chip(n).attrs['aria-pressed'], 'false');
});

test('unregistered layers never throw on click or ready', () => {
  const app = browser({ registered: [] });
  app.run();
  assert.doesNotThrow(() => { app.chip('writers').click(); app.document.emit('travelmap:ready'); });
  assert.equal(app.chip('writers').attrs['aria-pressed'], 'true');
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

test('a layer registered late still starts off', () => {
  const app = browser({ registered: [] });
  app.run();
  app.registered.add('writers');
  app.document.emit('travelmap:ready');
  assert.deepEqual(app.calls.at(-1), ['writers', false]);
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

function browser({ registered = NAMES } = {}) {
  const reg = new Set(registered);
  const calls = [];
  const slot = element('DIV');
  // The server renders the chips as spans with each destination.
  const renderChips = () => {
    slot.children = NAMES.map((n, i) => {
      const span = element('SPAN');
      span.className = 'writers-map-layer writers-map-layer-' + n;
      span.textContent = LABELS[i];
      return span;
    });
  };
  renderChips();
  const handlers = {};
  const document = {
    readyState: 'complete',
    createElement: element,
    querySelectorAll: sel => (sel === '[data-map-slot="layers"]' ? [slot] : []),
    addEventListener(n, cb) { (handlers[n] ??= []).push(cb); },
    emit(n, detail) { (handlers[n] || []).forEach(cb => cb({ detail })); }
  };
  const window = { TravelMap: { setLayerEnabled(name, on) { if (!reg.has(name)) throw new Error('Unknown map layer: ' + name); calls.push([name, on]); } } };
  const context = vm.createContext({ document, window, JSON });
  const chips = () => slot.children;
  return {
    document, window, calls, registered: reg, chips, renderChips,
    chip: n => chips().find(c => c.className.includes('writers-map-layer-' + n)),
    run: () => vm.runInContext(source, context)
  };
}

test('pill outlines are drawn inside the pill so the transparent hit-area borders leave no stray edges', () => {
  const css = readFileSync(new URL('../../../public/redesign/map-layers.css', import.meta.url), 'utf8');
  assert.match(css, /button\.writers-map-layer \{ border-color: transparent; box-shadow: inset 0 0 0 1px var\(--tt-border\); \}/);
  assert.match(css, /button\.writers-map-layer-notable\[aria-pressed="true"\] \{ box-shadow: none; \}/);
});
