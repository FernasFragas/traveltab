#!/usr/bin/env node
// M2: hash-selected views, persistent map node, search without a clear cross, and phone width.
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { connect, startPreview } from './lib/browser.mjs';

const preview = await startPreview({ scenario: 'writers-map' });
let browser;
try {
  browser = await connect();
  if (process.env.NAVIGATION_PATH) browser.state.overrides.set('/redesign/navigation.js', {
    responseCode: 200, body: (await readFile(process.env.NAVIGATION_PATH)).toString('base64'),
    responseHeaders: [{ name: 'Content-Type', value: 'text/javascript' }]
  });
  for (const [width, height] of [[375, 800], [1440, 900]]) {
    await browser.viewport(width, height);
    await browser.navigate(`${preview.url}/#stays`);
    const state = () => browser.evaluate(`({
      shown: [...document.querySelectorAll('[data-tt-view]')].filter(s => !s.hidden).map(s => s.id),
      active: [...document.querySelectorAll('[data-tt-section][aria-current]')].map(a => a.dataset.ttSection),
      overflow: document.documentElement.scrollWidth - document.documentElement.clientWidth
    })`);
    assert.deepEqual(await state(), { shown: ['stays'], active: ['stays', 'stays'], overflow: 0 }, `${width}: direct #stays`);
    await browser.evaluate(`window.mapNode = document.querySelector('[data-map-canvas]'); document.querySelector('.tt-bottom-navigation [data-tt-section="overview"]').click()`);
    await browser.until('location.hash === "#overview" && !document.querySelector("#overview").hidden');
    await browser.evaluate(`document.querySelector('.tt-bottom-navigation [data-tt-section="itinerary"]').click()`);
    await browser.until('location.hash === "#itinerary" && !document.querySelector("#itinerary").hidden');
    assert.equal(await browser.evaluate('!!document.querySelector("#itinerary .trip-form")'), true, `${width}: planner is in itinerary`);
    await browser.evaluate(`document.querySelector('.tt-bottom-navigation [data-tt-section="overview"]').click()`);
    await browser.until('location.hash === "#overview" && !document.querySelector("#overview").hidden');
    assert.equal(await browser.evaluate('window.mapNode === document.querySelector("[data-map-canvas]")'), true, `${width}: map node persists`);
    await browser.evaluate('history.back()');
    await browser.until('location.hash === "#itinerary" && !document.querySelector("#itinerary").hidden');
    assert.equal((await state()).overflow, 0, `${width}: no horizontal scroll after Back`);
    await browser.evaluate(`document.querySelector('#itinerary .trip-form').requestSubmit()`);
    await browser.until('document.querySelectorAll("#itinerary .itinerary-day").length > 0');
    assert.equal(await browser.evaluate('location.hash'), '#itinerary', `${width}: planning keeps the Itinerary URL`);
    assert.equal(await browser.evaluate('!!document.querySelector("#itinerary .trip-form") && !document.querySelector("#itinerary").hidden'), true,
      `${width}: planning preserves the visible form and view`);
    await browser.evaluate(`document.querySelector('#city_name').value = 'Porto'; document.querySelector('#city_name').focus()`);
    await browser.key('Enter', 'Enter', 13);
    assert.equal(await browser.evaluate("!!document.querySelector('.tt-search-submit')"), false, 'search submits without an arrow button');
    assert.equal(await browser.evaluate("document.querySelector('.tt-brand').getAttribute('aria-label')"), 'TravelTabByPatronFragas');
    await browser.until('document.querySelector("#itinerary .trip-form")?.elements.city.value === "Porto"');
    assert.equal(await browser.evaluate('location.hash'), '#overview', `${width}: new search opens the map`);
    assert.equal(await browser.evaluate('!document.querySelector("#overview").hidden'), true, `${width}: Overview is visible after search`);
    await browser.evaluate(`document.querySelector('#city_name').value = 'Porto'; document.querySelector('#city_name').dispatchEvent(new Event('input', { bubbles: true }))`);
    assert.deepEqual(await browser.evaluate(`({ clear: !!document.querySelector('.tt-search-clear'), nativeCancel: [...document.styleSheets].filter(sheet => sheet.href?.startsWith(location.origin)).flatMap(sheet => [...sheet.cssRules]).some(rule => rule.selectorText?.includes('::-webkit-search-cancel-button') && rule.style.display === 'none') ? 'none' : 'shown' })`),
      { clear: false, nativeCancel: 'none' }, `${width}: no grey cross in the filled search box`);
    console.log(`PASS ${width}: direct hash, tabs, persistent map, Back, planning, new search, no clear cross, no overflow`);
  }
  assert.deepEqual(browser.errors, [], 'no page exceptions');
  console.log('PASS: console clean');
} finally {
  await browser?.close?.();
  preview.stop();
}
