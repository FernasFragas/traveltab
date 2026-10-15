import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
const html = readFileSync('views/index.go.tpl', 'utf8');
test('header follows brand search tabs bookmark and retains accessible brand', () => {
  assert.ok(html.includes('Search destinations, cities, or places…'));
  assert.ok(html.indexOf('class="tt-search-area"') < html.indexOf('id="header-navigation"'));
  assert.ok(html.includes('aria-label="TravelTabByPatronFragas"'));
});
test('phone navigation has four icons including a bed SVG', () => {
  const nav = html.slice(html.indexOf('<nav class="tt-bottom-navigation"'));
  for (const icon of ['bi-compass-fill', 'bi-map', '<svg', 'bi-play-circle']) assert.ok(nav.includes(icon), icon);
});

// As a traveller, I want a small cross in the search box, so that I can clear it in one tap.
test('search has a labelled clear button that starts hidden', () => {
  const form = html.slice(html.indexOf('<form id="destination-search"'), html.indexOf('</form>'));
  assert.match(form, /<button type="button" class="tt-search-clear" aria-label="Clear search" hidden><i class="bi bi-x-lg" aria-hidden="true"><\/i><\/button>/);
  assert.ok(form.indexOf('tt-search-clear') > form.indexOf('id="city_name"'), 'the cross sits after the field');
});
