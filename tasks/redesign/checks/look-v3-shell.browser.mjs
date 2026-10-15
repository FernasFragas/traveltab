import assert from 'node:assert/strict';
import { mkdir, writeFile } from 'node:fs/promises';
import { connect, startPreview, openPlacePanel } from './lib/browser.mjs';
const preview = await startPreview({scenario:'writers-map'});
const browser = await connect();
try {
  for (const width of [375,1440]) {
    await browser.viewport(width,900);
    await browser.navigate(preview.url);
    await browser.until(`document.querySelectorAll('.map-pin').length > 0`);
    const layout = await browser.evaluate(`(() => {
      const box = s => {const r=document.querySelector(s).getBoundingClientRect();return {top:r.top,bottom:r.bottom,left:r.left,right:r.right,height:r.height,width:r.width}};
      return {hero:box('.destination-banner'), title:box('.destination-title'),tagline:box('.destination-tagline'), tabs:box('.tt-header-navigation'),bookmark:box('.tt-saved-toggle'), overflow:document.documentElement.scrollWidth-document.documentElement.clientWidth};
    })()`);
    assert.equal(layout.overflow,0);
    assert.ok(layout.hero.height >= (width===375?90:150) && layout.hero.height <= (width===375?110:165),JSON.stringify(layout));
    assert.ok(layout.title.top >= layout.hero.top && layout.title.bottom <= layout.hero.bottom,JSON.stringify(layout));
    assert.ok(layout.tagline.bottom <= layout.hero.bottom,JSON.stringify(layout));
    if(width===1440) assert.ok(layout.tabs.right <= layout.bookmark.left && layout.bookmark.left-layout.tabs.right<=72,JSON.stringify(layout));
    await browser.until(`document.querySelectorAll('.teaser-stops li').length === 3`);
    if (width === 375) {
      const day = await browser.evaluate(`(() => {
        const items = [...document.querySelectorAll('.teaser-stops li')];
        const shown = el => el && el.getClientRects().length > 0;
        const r = el => el.getBoundingClientRect();
        const photos = [...document.querySelectorAll('.teaser-stops img')].filter(shown);
        const texts = items.map(li => li.querySelector(':scope > span')).filter(shown);
        return { photos: photos.length, texts: texts.length,
          photoLeftOfText: photos.length > 0 && texts.every(t => r(photos[0]).right <= r(t).left),
          oneRow: texts.length > 1 && Math.abs(r(texts[0]).top - r(texts[1]).top) < 2 };
      })()`);
      assert.deepEqual(day, { photos: 1, texts: 2, photoLeftOfText: true, oneRow: true }, 'phone day card: one photo, Morning and Afternoon in a row');
    }
    await browser.evaluate(`document.querySelector('.weather-forecast-toggle').focus();document.querySelector('.weather-forecast-toggle').click()`);
    await browser.until(`document.querySelector('#weather-popover').matches(':popover-open')`);
    await browser.key('Escape','Escape',27);
    assert.ok(await browser.evaluate(`!document.querySelector('#weather-popover').matches(':popover-open') && document.activeElement.matches('.weather-forecast-toggle')`));
    await browser.evaluate(`document.activeElement.blur(); window.scrollTo(0,0)`);
    const dir = 'tasks/artifacts/look-v3/final'; await mkdir(dir,{recursive:true});
    await writeFile(`${dir}/${width}-city.png`,await browser.shot({x:0,y:0,width,height:await browser.evaluate('Math.ceil(document.documentElement.scrollHeight)')}));
    // A full-page CDP capture can pause animation frames until the tab is foregrounded.
    await browser.call('Page.bringToFront');
    await browser.evaluate(`window.TravelMap.selectPlace('Q215003',{focusPanel:false})`);
    await browser.until(`document.querySelector('.map-place-panel')`);
    await browser.until(`document.querySelector('.map-panel-photo img')?.complete && document.querySelector('.map-panel-photo img')?.naturalWidth > 0`);
    await browser.until(`(() => {const p=document.querySelector('.map-pin.is-selected')?.getBoundingClientRect(); const m=document.querySelector('[data-map-canvas]').getBoundingClientRect(); return p && p.left>=m.left && p.right<=m.right && p.top>=m.top && p.bottom<=m.bottom})()`);
    assert.ok(await browser.evaluate(`(() => {const p=document.querySelector('.map-pin-selected')?.getBoundingClientRect(); const m=document.querySelector('[data-map-canvas]').getBoundingClientRect(); return p && p.left>=m.left && p.right<=m.right && p.top>=m.top && p.bottom<=m.bottom})()`), 'selected photo callout fits inside the map');
    assert.ok((await browser.evaluate(`document.querySelector('.map-panel-facts').textContent`)).includes('10:00 – 18:30'));
    const card = await browser.evaluate(`(() => {
      const r = s => document.querySelector(s)?.getBoundingClientRect();
      const title = document.querySelector('.map-place-panel h3'), photo = r('.map-panel-photo img'), credit = r('.map-panel-photo figcaption');
      const attribution = document.querySelector('.maplibregl-ctrl-attrib');
      const label = document.querySelector('.map-pin.is-selected .map-pin-label');
      return {
        titleLines: Math.round(title.getBoundingClientRect().height / parseFloat(getComputedStyle(title).lineHeight)),
        creditInsidePhoto: !credit || (credit.top >= photo.top && credit.bottom <= photo.bottom + 0.5),
        meta: document.querySelector('.map-panel-meta').textContent,
        bestTime: document.querySelector('.map-panel-best-time')?.hidden === false ? document.querySelector('.map-panel-best-time').textContent : '',
        quoteIsLink: !!document.querySelector('.map-panel-writers blockquote p a'),
        attributionOpen: !!attribution && attribution.classList.contains('maplibregl-compact-show'),
        selectedLabelShown: !!label && getComputedStyle(label).display !== 'none',
        notablePill: (() => { const b = document.querySelector('button.writers-map-layer-notable'); return b && b.getAttribute('aria-pressed') + ' ' + getComputedStyle(b).backgroundColor; })(),
        writersPill: getComputedStyle(document.querySelector('button.writers-map-layer-writers')).backgroundColor
      };
    })()`);
    const lines = await browser.evaluate(`(() => {
      const count = el => Math.round(el.getBoundingClientRect().height / parseFloat(getComputedStyle(el).lineHeight));
      const arrow = document.querySelector('.map-pin-selected > span:last-child');
      return { links: [...document.querySelectorAll('.map-panel-links strong')].map(count),
        truncated: [...document.querySelectorAll('.map-panel-links strong')].filter(el => el.scrollWidth > el.clientWidth).map(el => el.textContent),
        stayTitle: count(document.querySelector('.teaser-stays h3')),
        calloutArrow: arrow ? parseFloat(getComputedStyle(arrow).fontSize) : 0 };
    })()`);
    assert.deepEqual(lines.links, [1, 1], 'link card titles stay on one line: ' + JSON.stringify(lines));
    assert.deepEqual(lines.truncated, [], 'link card titles are not cut off: ' + JSON.stringify(lines));
    assert.equal(lines.stayTitle, 1, 'Where to stay fits on one line');
    assert.ok(lines.calloutArrow >= 20, 'callout arrow is a visible chevron: ' + JSON.stringify(lines));
    assert.equal(card.titleLines, 1, 'place title fits on one line: ' + JSON.stringify(card));
    assert.ok(card.creditInsidePhoto, 'photo credit overlays the photo: ' + JSON.stringify(card));
    assert.match(card.meta, /Historic Landmark/);
    assert.match(card.bestTime, /Morning.*Tip: quieter early in the day/s);
    assert.equal(card.quoteIsLink, false);
    assert.equal(card.attributionOpen, false, 'map credits start collapsed');
    assert.equal(card.selectedLabelShown, false, 'selected pin has no duplicate name label');
    assert.equal(card.notablePill, 'true rgb(166, 64, 39)', 'notable is on and filled terracotta');
    assert.equal(card.writersPill, 'rgb(255, 255, 255)');
    await browser.evaluate(`document.activeElement.blur(); window.scrollTo(0,0)`);
    const bounds = await browser.evaluate(`Object.fromEntries(['.tt-header','.destination-banner','[data-map-slot="layers"]','[data-map-canvas]','[data-map-slot="panel"]','.map-panel-facts','.map-panel-writers','.overview-teasers'].map(selector => {const element=document.querySelector(selector);if(!element)return [selector,null];const r=element.getBoundingClientRect();return [selector,{x:r.x+scrollX,y:r.y+scrollY,width:r.width,height:r.height}]}))`);
    await writeFile(`${dir}/${width}-bounds.json`,JSON.stringify(bounds,null,2));
    await writeFile(`${dir}/${width}-place.png`,await browser.shot({x:0,y:0,width,height:await browser.evaluate('Math.ceil(document.documentElement.scrollHeight)')}));
    console.log(`PASS ${width}: header, hero text bounds, forecast Escape/focus, Belém facts, readable kind, tip, quote, collapsed credits, pills, screenshots`);
  }
  assert.deepEqual(browser.errors,[]);
} finally {await browser.close();preview.stop();}
