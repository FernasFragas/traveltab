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

test('Add to plan saves once, submits the plan, then opens the updated itinerary', () => {
  const app = browser();
  app.run();
  app.click(app.addButton);
  assert.equal(app.store['traveltab.savedQids'], '["Q1"]');
  assert.equal(app.submissions, 1);
  const plan = { path: '/plan', parameters: {} };
  app.fire('htmx:configRequest', plan);
  assert.deepEqual([...plan.parameters.include], ['Q1']);
  app.fire('htmx:afterSwap', { target: { id: 'trip-card' } });
  assert.equal(app.location.hash, '#itinerary');
  app.click(app.addButton);
  assert.equal(app.store['traveltab.savedQids'], '["Q1"]');
});

test('saving shows a polite toast and Undo removes just that place', () => {
 const app = browser(); app.run(); app.click(app.button); app.click(app.button2);
 assert.equal(app.toast.attrs['aria-live'], 'polite');
 assert.equal(app.toast.children[0].textContent, 'Added to your saved places · ');
 app.click(app.toast.children[1]);
 assert.equal(app.store['traveltab.savedQids'], '["Q1"]');
 assert.equal(app.toast.hidden, true);
});
test('header saved list has an empty state, names and Escape restores focus', () => {
 const app = browser(); app.run(); app.click(app.toggle);
 assert.equal(app.list.children[0].textContent, 'No saved places yet');
 app.fire('travelmap:places-loaded', { features: [{ properties: { qid: 'Q1', name: 'Tower' } }] }); app.click(app.button); app.click(app.toggle);
 assert.equal(app.list.children[0].textContent, 'Tower');
 app.key('Escape'); assert.equal(app.toggle.focused, true);
});
test('toast expires and a replacement cancels the previous timer', () => {
 const app = browser(); app.run(); app.click(app.button); app.click(app.button2);
 assert.equal(app.timers.size, 1); app.expire(); assert.equal(app.toast.hidden, true);
});

test('saved list retains destination names and waits for its QID after a cross-city search', () => {
 const app = browser(); app.run();
 app.fire('travelmap:places-loaded', { features: [{ properties: { qid: 'Q1', name: 'Tower' } }] });
 app.click(app.button);
 app.shell.dataset.mapDestination = 'porto-pt';
 app.fire('travelmap:places-loaded', { features: [] });
 assert.equal(app.list.children[0].textContent, 'Tower');
 app.click(app.list.children[0]);
 assert.equal(app.searchSubmissions, 1);
 assert.equal(app.fields['#city_name'].value, 'funchal');
 app.fire('travelmap:places-loaded', { features: [] });
 assert.deepEqual(app.selected, []);
 app.fire('travelmap:places-loaded', { features: [{ properties: { qid: 'Q1', name: 'Tower' } }] });
 assert.deepEqual(app.selected, ['Q1']);
});
test('legacy saved entries without destination data are explicitly unavailable', () => {
 const app = browser({ store: { 'traveltab.savedQids': '["Q1"]' } }); app.run();
 assert.equal(app.list.children[0].disabled, true);
 assert.match(app.list.children[0].textContent, /destination unavailable/);
});

function browser(options = {}) {
  const store = options.store || {};
  const handlers = {};
  const button = fakeNode('button', { saveQid: 'Q1' });
  const button2 = fakeNode('button', { saveQid: 'Q2' });
  const addButton = fakeNode('button', { addQid: 'Q1' });
  button.textContent = button2.textContent = 'Save to plan';
  const slot = fakeNode('div');
  const toast = fakeNode('div'); const toggle = fakeNode('button'); toggle.setAttribute('aria-controls', 'saved-places-list'); const list = fakeNode('div');
  const timers = new Map(); let timerID = 0;
  const location = { hash: '' };
  let submissions = 0, searchSubmissions = 0;
  const selected = [];
  const shell = { dataset: { mapDestination: 'funchal-pt', mapLat: '32.65', mapLon: '-16.91' } };
  const fields = Object.fromEntries(['#city_name', '#country_code', '#place_lat', '#place_lon'].map(id => [id, { value: '' }]));
  const searchForm = { requestSubmit() { searchSubmissions++; } };
  const form = { requestSubmit() { submissions++; } };
  const document = {
    addEventListener(name, fn) { handlers[name] = fn; },
    body: { addEventListener(name, fn) { handlers[name] = fn; } },
    createElement: (tag) => fakeNode(tag),
    querySelector: (selector) => (selector in fields ? fields[selector] : selector === '[data-writers-map]' ? shell : selector === '#destination-search' ? searchForm : selector === '[data-map-slot="saved"]' ? slot : selector === '.trip-form' ? form : selector === '.tt-saved-toggle' ? toggle : selector === '#saved-places-list' ? list : selector === '.writers-map-toast' ? toast : null),
    querySelectorAll: (selector) => (selector === '[data-save-qid]' ? [button, button2] : selector === '[data-add-qid]' ? [addButton] : [])
  };
  const localStorage = {
    getItem(key) { if (options.blocked) throw new Error('blocked'); return key in store ? store[key] : null; },
    setItem(key, value) { if (options.blocked) throw new Error('blocked'); store[key] = value; }
  };
  return {
    store, button, button2, addButton, slot, location, toast, toggle, list, timers, selected, shell, fields,
    get searchSubmissions() { return searchSubmissions; },
    key(key) { handlers.keydown({ key, preventDefault() {} }); }, expire() { for (const fn of timers.values()) fn(); timers.clear(); },
    get submissions() { return submissions; },
    run() { vm.runInNewContext(source, { window: { localStorage, location, TravelMap: { selectPlace(qid) { selected.push(qid); } } }, document, console, clearTimeout: id => timers.delete(id), setTimeout: (fn, delay) => { if (!delay) { fn(); return; } timers.set(++timerID, fn); return timerID; } }); },
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
    focus() { this.focused = true; },
    setAttribute(name, value) { this.attrs[name] = value; },
    getAttribute(name) { return this.attrs[name]; },
    append(...nodes) { this.children.push(...nodes); },
    replaceChildren(...nodes) { this.children = nodes; },
    closest(selector) {
      if (selector === '.tt-saved-toggle') return this.attrs['aria-controls'] === 'saved-places-list' ? this : null;
      if (selector === '[data-saved-undo]') return 'data-saved-undo' in this.attrs ? this : null;
      if (selector === '[data-saved-place]') return 'data-saved-place' in this.attrs ? this : null;
      if (selector === '[data-save-qid]') return 'saveQid' in this.dataset ? this : null;
      if (selector === '[data-add-qid]') return 'addQid' in this.dataset ? this : null;
      if (selector === '[data-saved-clear]') return 'data-saved-clear' in this.attrs ? this : null;
      return null;
    }
  };
  return node;
}
