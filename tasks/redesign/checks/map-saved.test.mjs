import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import vm from 'node:vm';

const source = readFileSync(new URL('../../../public/redesign/map-saved.js', import.meta.url), 'utf8');

test('toggles a save button to Saved and back, and remembers it in storage', () => {
  const app = browser();
  app.run();
  app.click(app.button);
  assert.equal(app.button.textContent, 'Saved');
  assert.equal(app.button.attrs['aria-pressed'], 'true');
  assert.equal(app.store['traveltab.savedQids'], '["Q1"]');
  app.click(app.button);
  assert.equal(app.button.textContent, 'Save to plan');
  assert.equal(app.store['traveltab.savedQids'], '[]');
});

test('saves survive a reload', () => {
  const first = browser();
  first.run();
  first.click(first.button);
  const second = browser({ store: first.store });
  second.run();
  assert.equal(second.button.textContent, 'Saved');
  assert.equal(second.lineText(), '1 saved place will be included');
});

test('saving makes no server call and shows the count line; clear empties the list', () => {
  const app = browser();
  app.run();
  app.click(app.button);
  assert.equal(app.lineText(), '1 saved place will be included');
  app.click(app.button2);
  assert.equal(app.lineText(), '2 saved places will be included');
  const event = app.click(app.slot.children[0].children[1]);
  assert.equal(event.defaultPrevented, true);
  assert.equal(app.slot.children.length, 0);
  assert.equal(app.button.textContent, 'Save to plan');
  assert.equal(app.store['traveltab.savedQids'], '[]');
});

test('only the plan request carries the saved QIDs as include', () => {
  const app = browser();
  app.run();
  app.click(app.button);
  app.click(app.button2);
  const plan = { path: '/plan', parameters: { city: 'Funchal' } };
  app.fire('htmx:configRequest', plan);
  assert.deepEqual([...plan.parameters.include], ['Q1', 'Q2']);
  const other = { path: '/process-form/', parameters: {} };
  app.fire('htmx:configRequest', other);
  assert.equal('include' in other.parameters, false);
});

test('adds nothing to the plan request when nothing is saved', () => {
  const app = browser();
  app.run();
  const plan = { path: '/plan', parameters: {} };
  app.fire('htmx:configRequest', plan);
  assert.equal('include' in plan.parameters, false);
});

test('blocked storage hides the buttons and shows no line', () => {
  const app = browser({ blocked: true });
  app.run();
  assert.equal(app.button.hidden, true);
  app.click(app.button);
  assert.equal(app.slot.children.length, 0);
  const plan = { path: '/plan', parameters: {} };
  app.fire('htmx:configRequest', plan);
  assert.equal('include' in plan.parameters, false);
});

test('buttons from a later swap are bound, and corrupt storage is ignored', () => {
  const app = browser({ store: { 'traveltab.savedQids': 'not json' } });
  app.run();
  app.fire('htmx:afterSwap', {});
  app.click(app.button);
  assert.equal(app.button.textContent, 'Saved');
});

test('buttons rendered by the map core on place selection are painted or hidden', () => {
  const blocked = browser({ blocked: true });
  blocked.run();
  blocked.button.hidden = false;
  blocked.fire('travelmap:place-selected', { qid: 'Q1' }, true);
  assert.equal(blocked.button.hidden, true);
  const app = browser({ store: { 'traveltab.savedQids': '["Q1"]' } });
  app.run();
  app.button.textContent = 'Save to plan';
  app.fire('travelmap:place-selected', { qid: 'Q1' }, true);
  assert.equal(app.button.textContent, 'Saved');
});

function browser(options = {}) {
  const store = options.store || {};
  const handlers = {};
  const button = fakeNode('button', { saveQid: 'Q1' });
  const button2 = fakeNode('button', { saveQid: 'Q2' });
  button.textContent = button2.textContent = 'Save to plan';
  const slot = fakeNode('div');
  const document = {
    addEventListener(name, fn) { handlers[name] = fn; },
    body: { addEventListener(name, fn) { handlers[name] = fn; } },
    createElement: (tag) => fakeNode(tag),
    querySelector: (selector) => (selector === '[data-map-slot="saved"]' ? slot : null),
    querySelectorAll: (selector) => (selector === '[data-save-qid]' ? [button, button2] : [])
  };
  const localStorage = {
    getItem(key) { if (options.blocked) throw new Error('blocked'); return key in store ? store[key] : null; },
    setItem(key, value) { if (options.blocked) throw new Error('blocked'); store[key] = value; }
  };
  return {
    store, button, button2, slot,
    run() { vm.runInNewContext(source, { window: { localStorage }, document, console, setTimeout: (fn) => fn() }); },
    fire(name, detail) { handlers[name]({ detail }); },
    lineText() { return slot.children[0].children[0].textContent.trim(); },
    click(target) {
      const event = { target, defaultPrevented: false, preventDefault() { this.defaultPrevented = true; } };
      handlers.click(event);
      return event;
    }
  };
}

function fakeNode(tag, dataset = {}) {
  const node = {
    tag, dataset, attrs: {}, children: [], hidden: false, textContent: '',
    setAttribute(name, value) { this.attrs[name] = value; },
    append(...nodes) { this.children.push(...nodes); },
    replaceChildren(...nodes) { this.children = nodes; },
    closest(selector) {
      if (selector === '[data-save-qid]') return 'saveQid' in this.dataset ? this : null;
      if (selector === '[data-saved-clear]') return 'data-saved-clear' in this.attrs ? this : null;
      return null;
    }
  };
  return node;
}
