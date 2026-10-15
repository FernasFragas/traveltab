import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';
import { test } from 'node:test';
const window = {};
vm.runInNewContext(readFileSync('public/redesign/opening-hours.js', 'utf8'), { window });
const hours = window.TravelOpeningHours;
test('simple range and closed weekdays', () => {
 assert.equal(hours('Tu-Su 10:00-18:30; Mo off', new Date(2026, 9, 15)), '10:00 – 18:30');
 assert.equal(hours('Tu-Su 10:00-18:30; Mo off', new Date(2026, 9, 12)), 'Closed today');
 assert.equal(hours('Mo-Fr 09:00-17:00', new Date(2026, 9, 18)), 'Closed today');
});
test('multiple ranges, unknown syntax and empty input', () => {
 assert.equal(hours('Mo-Su 09:00-12:00,13:00-18:00'), '09:00 – 12:00, 13:00 – 18:00');
 assert.equal(hours('by appointment'), 'by appointment');
 assert.equal(hours('Mo-Su 25:00-28:00'), 'Mo-Su 25:00-28:00');
 assert.equal(hours(''), '');
});
