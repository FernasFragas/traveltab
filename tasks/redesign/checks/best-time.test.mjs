import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';
import { test } from 'node:test';
const window = {};
vm.runInNewContext(readFileSync('public/redesign/best-time.js', 'utf8'), { window });
test('kind rules are labelled tips, case insensitive and hidden for unknown kinds', () => {
 assert.equal(window.TravelBestTime('Art MUSEUM').value, 'Morning');
 assert.equal(window.TravelBestTime('museum').tip, 'Tip: quieter early in the day');
 assert.equal(window.TravelBestTime('viewpoint').value, 'Sunset');
 assert.equal(window.TravelBestTime('park').tip, 'Tip: cooler and calmer early');
 assert.equal(window.TravelBestTime('market').tip, 'Tip: freshest stalls in the morning');
 assert.equal(window.TravelBestTime('restaurant'), null);
});
test('historic landmarks share the quieter-early tip', () => {
 for (const kind of ['tower', 'castle', 'fort', 'monument']) {
  assert.equal(window.TravelBestTime(kind).value, 'Morning', kind);
  assert.equal(window.TravelBestTime(kind).tip, 'Tip: quieter early in the day', kind);
 }
});
