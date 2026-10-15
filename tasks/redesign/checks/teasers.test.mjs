import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';
class Node {
  constructor() { this.children = []; this.textContent = ''; this.dataset = {}; }
  appendChild(child) { this.children.push(child); return child; }
  replaceChildren(...children) { this.children = children; }
  querySelector() { return null; }
}
const setup = (hasStay = true) => {
  const preview = new Node(); const stays = new Node(); const firstStay = new Node(); firstStay.querySelector = selector => ({textContent: selector === ".stay-name" ? "My hotel" : "Hotel · 5-star property"}); const listeners = {};
  const document = { body: {addEventListener: (name, fn) => listeners[name] = fn}, addEventListener: (name, fn) => listeners[name] = fn, createElement: () => new Node(), querySelector: selector => selector === '[data-teaser-day]' ? preview : selector === '[data-teaser-stays]' ? stays : selector === '#stays .stay-item' ? (hasStay ? firstStay : null) : selector === '#stays .stay-note' ? {textContent:'No matching places to stay were returned.'} : null };
  vm.runInNewContext(readFileSync('public/redesign/teasers.js', 'utf8'), { document });
  return { preview, stays, emit: (name, detail) => listeners[name]({detail}) };
};
const names = node => [node.textContent, ...node.children.flatMap(names)];
test('suggested day uses three highest ranked notable places', () => {
  const app = setup();
  app.emit('travelmap:places-loaded', { features: [1,4,2,3].map(rank => ({properties:{name:'Place '+rank,sitelinks:rank}})) });
  const text = names(app.preview);
  assert.ok(text.includes('Suggested day'));
  assert.deepEqual(text.filter(s => s.startsWith('Place')), ['Place 4','Place 3','Place 2']);
});
test('generated day replaces suggestion and later place loads preserve the plan', () => {
  const app = setup();
  app.emit('travelmap:plan-updated',{ days:[{number:1,stops:[{name:'My stop'}]}] });
  app.emit('travelmap:places-loaded',{features:[]});
  const text = names(app.preview);
  assert.ok(text.includes('Day 1')); assert.ok(text.includes('My stop'));
});

test('generated plan updates the hotel preview from the rendered stays', () => {
  const app = setup(); app.emit('htmx:afterSwap', {});
  assert.ok(names(app.stays).includes('My hotel'));
  assert.ok(names(app.stays).includes('Hotel · 5-star property'));
});

test('generated empty stays replaces the planning prompt', () => {
  const app = setup(false); app.emit('htmx:afterSwap', {});
  assert.equal(app.stays.textContent, 'No matching places to stay were returned.');
});
