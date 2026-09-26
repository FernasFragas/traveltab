#!/usr/bin/env node
// Run with HTMX_PATH=/private/tmp/traveltab-repair-htmx-1.9.11.js CDP_PORT=9227 node tasks/city-autocomplete-check.mjs
// An isolated Brave/Chromium debug tab is required; the preview starts here.
import assert from 'node:assert/strict';
import { connect, startPreview } from './redesign/checks/lib/browser.mjs';

const preview = await startPreview();
const browser = await connect();
const { evaluate, until, viewport, navigate, key } = browser;

try {
  for (const width of [375, 1440]) {
    await viewport(width, width === 375 ? 812 : 1000);
    await navigate(preview.url);
    await evaluate("document.querySelector('#city_name').focus(); document.querySelector('#city_name').value = 'paris'; document.querySelector('#city_name').dispatchEvent(new Event('input', { bubbles: true }))");
    await until("document.querySelectorAll('#destination-suggestions [role=option]').length === 3 && document.querySelector('#city_name').getAttribute('aria-expanded') === 'true'");
    assert.equal(await evaluate("document.documentElement.scrollWidth <= innerWidth"), true);
    assert.deepEqual(await evaluate("[...document.querySelectorAll('#destination-suggestions [role=option]')].map(e => [e.dataset.value, e.dataset.countryCode, e.querySelector('small')?.textContent])"), [
      ['Paris, France', 'FR', 'Île-de-France'], ['Paris, United States', 'US', 'Texas'], ['Paris, United States', 'US', 'Tennessee']
    ]);
    await key('ArrowDown', 'ArrowDown', 40);
    await key('ArrowDown', 'ArrowDown', 40);
    assert.equal(await evaluate("document.querySelector('#city_name').getAttribute('aria-activedescendant')"), 'destination-option-1');
    await key('Enter', 'Enter', 13);
    await until("document.querySelector('.destination-title')?.textContent.trim() === 'Paris' && document.querySelector('.trip-form')?.elements.country.value === 'us'");
    assert.deepEqual(await evaluate("[document.querySelector('#city_name').value, document.querySelector('#country_code').value, document.querySelector('#city_name').getAttribute('aria-expanded')]"), ['Paris, United States', 'US', 'false']);
    assert.equal(await evaluate("document.querySelector('.trip-form').elements.lat.value"), '33.66094');

    await evaluate("document.querySelector('#city_name').focus(); document.querySelector('#city_name').value = 'paris'; document.querySelector('#city_name').dispatchEvent(new Event('input', { bubbles: true }))");
    await until("document.querySelectorAll('#destination-suggestions [role=option]').length === 3 && document.querySelector('#city_name').getAttribute('aria-expanded') === 'true'");
    await evaluate("document.querySelector('#destination-option-2').click()");
    await until("document.querySelector('.trip-form')?.elements.lat.value === '36.302'");
    assert.equal(await evaluate("document.querySelector('#country_code').value"), 'US');

    await evaluate("document.querySelector('#city_name').focus(); document.querySelector('#city_name').value = 'lis'; document.querySelector('#city_name').dispatchEvent(new Event('input', { bubbles: true }))");
    assert.equal(await evaluate("document.querySelector('#country_code').value"), '');
    await until("document.querySelector('#destination-suggestions [role=option]')?.dataset.value === 'Lisbon, Portugal' && document.querySelector('#city_name').getAttribute('aria-expanded') === 'true'");
    await key('Escape', 'Escape', 27);
    assert.equal(await evaluate("document.querySelector('#destination-suggestions').hidden && document.querySelector('#city_name').getAttribute('aria-expanded') === 'false'"), true);

    await evaluate("document.querySelector('#city_name').value = 'lisb'; document.querySelector('#city_name').dispatchEvent(new Event('input', { bubbles: true }))");
    await until("document.querySelector('#destination-suggestions [role=option]')?.dataset.value === 'Lisbon, Portugal' && document.querySelector('#city_name').getAttribute('aria-expanded') === 'true'");
    await evaluate("document.querySelector('#destination-suggestions [role=option]').click()");
    await until("document.querySelector('.destination-title')?.textContent.trim() === 'Lisbon' && document.querySelector('.trip-form')?.elements.country.value === 'pt'");
    assert.equal(await evaluate("document.querySelector('#country_code').value"), 'PT');

    await evaluate("document.querySelector('#city_name').focus(); document.querySelector('#city_name').value = 'paris'; document.querySelector('#city_name').dispatchEvent(new Event('input', { bubbles: true }))");
    await until("document.querySelectorAll('#destination-suggestions [role=option]').length === 3 && document.querySelector('#city_name').getAttribute('aria-expanded') === 'true'");
    await evaluate("document.querySelector('#destination-option-0').click()");
    await until("document.querySelector('.destination-title')?.textContent.trim() === 'Paris' && document.querySelector('.trip-form')?.elements.country.value === 'fr'");
    assert.equal(await evaluate("document.querySelector('#country_code').value"), 'FR');

    browser.state.overrides.set('/suggest', { responseCode: 200, body: '', responseHeaders: [{ name: 'Content-Type', value: 'text/html' }] });
    await evaluate("document.querySelector('#city_name').focus(); document.querySelector('#city_name').value = 'Lisbon, Portugal'; document.querySelector('#city_name').dispatchEvent(new Event('input', { bubbles: true }))");
    await until("document.querySelector('#country_code').value === '' && document.querySelector('#destination-suggestions').hidden");
    await new Promise(resolve => setTimeout(resolve, 400));
    assert.equal(await evaluate("document.querySelectorAll('#destination-suggestions [role=option]').length"), 0);
    await key('Enter', 'Enter', 13);
    try {
      await until("document.querySelector('.destination-title')?.textContent.trim() === 'Lisbon'");
    } catch (error) {
      console.error(await evaluate("({ title: document.querySelector('.destination-title')?.textContent.trim(), value: document.querySelector('#city_name')?.value, code: document.querySelector('#country_code')?.value, error: document.querySelector('#destination-search-error')?.textContent, url: location.href, expanded: document.querySelector('#city_name')?.getAttribute('aria-expanded'), focus: document.activeElement?.id })"));
      throw error;
    }
    browser.state.overrides.delete('/suggest');
    console.log(`PASS ${width}px autocomplete, Paris namesakes, Lisbon, keyboard and empty-suggestions fallback`);
  }
  assert.deepEqual(browser.errors, []);
} finally {
  await browser.close();
  preview.stop();
}
