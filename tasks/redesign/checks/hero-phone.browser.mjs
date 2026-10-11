#!/usr/bin/env node
// Hero line on phones: below the photo and above the weather card, readable on the page surface; desktop unchanged.
// Start an isolated Brave/Chromium on CDP_PORT with about:blank, then run this check.
import assert from 'node:assert/strict';
import { connect, startPreview } from './lib/browser.mjs';

// Desktop (1440 px) geometry of the unchanged hero, measured before the phone change (Lisbon).
const DESKTOP_BEFORE = {"photo":{"top":96,"bottom":414,"left":149,"right":1277,"width":1128},"banner":{"top":95,"bottom":415,"left":148,"right":1278,"width":1130},"tag":{"top":297,"bottom":321,"left":181,"right":637,"width":456},"card":{"top":112,"bottom":363,"left":893,"right":1261,"width":368},"title_":{"top":189,"bottom":250,"left":181,"right":637,"width":456},"tagVisible":true,"capVisible":true,"clamp":"none","colour":"rgb(255, 253, 250)"};

const preview = await startPreview({ scenario: 'default' });
let browser;
try {
  browser = await connect();
  await browser.call('Page.addScriptToEvaluateOnNewDocument', { source: `
    window.__previewErrors = [];
    const originalError = console.error.bind(console);
    console.error = (...args) => { window.__previewErrors.push(args.map(String).join(' ')); originalError(...args); };
    window.addEventListener('error', event => window.__previewErrors.push(event.message));` });
  const search = async (term, title) => {
    await browser.evaluate("window.__prev = document.querySelector('.destination-banner')");
    await browser.evaluate(`document.querySelector('#city_name').value = ${JSON.stringify(term)}; document.querySelector('#destination-search').requestSubmit()`);
    await browser.until(`document.querySelector('.destination-title')?.textContent.trim() === ${JSON.stringify(title)} && document.querySelector('.destination-banner') !== window.__prev`);
    await browser.settle();
  };
  const read = () => browser.evaluate(`(() => {
    const rect = el => { if (!el) return null; const r = el.getBoundingClientRect(); return { top: Math.round(r.top + scrollY), bottom: Math.round(r.bottom + scrollY), left: Math.round(r.left), right: Math.round(r.right), width: Math.round(r.width) }; };
    const visible = el => !!el && getComputedStyle(el).display !== 'none' && el.getBoundingClientRect().height > 0;
    const tag = document.querySelector('.destination-tagline');
    const cap = document.querySelector('.destination-photo-caption');
    const colour = tag && getComputedStyle(tag).color;
    const surface = getComputedStyle(document.querySelector('.weather-card')).backgroundColor;
    return {
      title: document.querySelector('.destination-title').textContent.trim(),
      text: tag ? tag.textContent.trim() : null, tagVisible: visible(tag), capVisible: visible(cap), colour, surface,
      banner: rect(document.querySelector('.destination-banner')), photo: rect(document.querySelector('.destination-photo')),
      tag: rect(tag), card: rect(document.querySelector('.weather-card')), title_: rect(document.querySelector('.destination-title')),
      clamp: tag ? getComputedStyle(tag).webkitLineClamp : null,
      overflowX: document.documentElement.scrollWidth - document.documentElement.clientWidth };
  })()`);

  await browser.viewport(375, 800);
  await browser.navigate(`${preview.url}/`);
  const phone = {};
  for (const [name, term, title] of [['Lisbon'], ['Porto', 'Porto, Portugal', 'Porto'], ['Nophoto', 'Nophoto', 'Nophoto']]) {
    if (term) await search(term, title);
    phone[name] = await read();
    const s = phone[name];
    assert.equal(s.overflowX, 0, `${name} 375: no horizontal overflow`);
    assert.equal(s.capVisible, false, `${name} 375: photo caption stays hidden`);
    if (name === 'Nophoto') {
      assert.equal(s.text, null, 'Nophoto 375: no tagline element');
      assert.equal(s.card.top - s.photo.bottom, 0, `Nophoto 375: no gap between photo and card ${JSON.stringify(s)}`);
      continue;
    }
    assert.ok(s.text, `${name} 375: has a hero line`);
    assert.equal(s.tagVisible, true, `${name} 375: hero line visible ${JSON.stringify(s)}`);
    assert.ok(s.tag.top >= s.photo.bottom, `${name} 375: below the photo ${JSON.stringify(s)}`);
    assert.ok(s.tag.bottom <= s.card.top, `${name} 375: above the weather card ${JSON.stringify(s)}`);
    assert.notEqual(s.colour, 'rgb(255, 253, 250)', `${name} 375: dark text on the page surface, not the overlay white`);
    assert.ok(s.tag.right <= 375 && s.tag.left >= 0, `${name} 375: inside the viewport`);
    assert.equal(s.clamp, 'none', `${name} 375: no line clamp`);
    console.log(`PASS 375 ${name}: "${s.text}" photo.bottom ${s.photo.bottom} <= tagline ${s.tag.top}-${s.tag.bottom} <= card.top ${s.card.top}`);
  }
  assert.equal(phone.Porto.text, 'city in northern Portugal');

  await browser.viewport(1440, 900);
  await browser.navigate(`${preview.url}/`);
  const desktop = await read();
  const geometry = { photo: desktop.photo, banner: desktop.banner, tag: desktop.tag, card: desktop.card, title_: desktop.title_, tagVisible: desktop.tagVisible, capVisible: desktop.capVisible, clamp: desktop.clamp, colour: desktop.colour };
  console.log('DESKTOP', JSON.stringify(geometry));
  assert.equal(desktop.overflowX, 0, '1440: no horizontal overflow');
  assert.equal(desktop.tagVisible, true, '1440: tagline visible');
  assert.equal(desktop.colour, 'rgb(255, 253, 250)', '1440: tagline stays light on the photo');
  assert.equal(desktop.clamp, 'none', '1440: no line clamp');
  assert.ok(desktop.tag.bottom <= desktop.photo.bottom, '1440: tagline over the photo');
  assert.deepEqual(geometry, DESKTOP_BEFORE, '1440: hero geometry unchanged from before the phone change');
  assert.deepEqual(await browser.evaluate('window.__previewErrors'), [], 'console clean');
  assert.deepEqual(browser.errors, [], 'no page errors');
  console.log('PASS 1440: hero geometry identical to the baseline; console clean');
} finally {
  await browser?.close?.();
  preview.stop();
}
