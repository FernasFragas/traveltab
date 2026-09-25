// In-page collectors for the accessibility audit. Each export is a JavaScript expression that
// runs inside the page (Runtime.evaluate) and returns plain JSON. Kept as strings so Node can
// ship them to the isolated browser without any bundling.

// Every piece of visible text (and icon) with its computed colour, size and line boxes. The
// caller hides all text, screenshots the page, and reads the real background pixels beneath
// each box, so gradients, translucent overlays and photographs are measured, not assumed.
export const collectText = `(() => {
  const canvas = document.createElement('canvas'); canvas.width = canvas.height = 1;
  const ctx = canvas.getContext('2d', { willReadFrequently: true });
  const parse = value => {
    ctx.clearRect(0, 0, 1, 1); ctx.fillStyle = '#000'; ctx.fillStyle = value; ctx.fillRect(0, 0, 1, 1);
    const d = ctx.getImageData(0, 0, 1, 1).data; return { rgb: [d[0], d[1], d[2]], a: d[3] / 255 };
  };
  const opacityOf = el => { let o = 1; for (let n = el; n && n.nodeType === 1; n = n.parentElement) o *= Number(getComputedStyle(n).opacity); return o; };
  const describe = el => {
    const id = el.id ? '#' + el.id : '';
    const cls = [...el.classList].slice(0, 2).map(c => '.' + c).join('');
    return el.tagName.toLowerCase() + id + cls;
  };
  const shown = el => { const cs = getComputedStyle(el); return cs.visibility !== 'hidden' && cs.display !== 'none'; };
  const large = (size, weight) => size >= 24 || (size >= 18.66 && weight >= 700);
  const controlType = el => el.closest('input, select, textarea') ? 'control' : 'text';
  const rectOf = r => ({ x: r.left + scrollX, y: r.top + scrollY, w: r.width, h: r.height });
  const out = [];
  const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
  for (let node = walker.nextNode(); node; node = walker.nextNode()) {
    const text = node.textContent.replace(/\\s+/g, ' ').trim();
    if (!text) continue;
    const el = node.parentElement;
    if (!el || ['SCRIPT', 'STYLE', 'NOSCRIPT', 'OPTION', 'TEMPLATE'].includes(el.tagName) || el.closest('.tt-visually-hidden, iframe, [hidden]')) continue;
    if (!shown(el)) continue;
    const range = document.createRange(); range.selectNodeContents(node);
    const rects = [...range.getClientRects()].filter(r => r.width > 1 && r.height > 1).map(rectOf);
    if (!rects.length) continue;
    const cs = getComputedStyle(el); const opacity = opacityOf(el);
    if (opacity === 0) continue;
    const c = parse(cs.color); const size = parseFloat(cs.fontSize); const weight = Number(cs.fontWeight);
    out.push({ kind: 'text', el: describe(el), text: text.slice(0, 60), rgb: c.rgb, alpha: c.a * opacity, size, weight, large: large(size, weight),
      rects, disabled: !!el.closest(':disabled, [aria-disabled="true"]'), link: !!el.closest('a') });
  }
  // Text drawn by CSS (the itinerary stop numbers). Icon-font glyphs are the .bi elements below.
  for (const el of document.body.querySelectorAll('*')) {
    if (el.classList.contains('bi') || !shown(el) || el.closest('.tt-visually-hidden, iframe, [hidden]')) continue;
    for (const pseudo of ['::before', '::after']) {
      const cs = getComputedStyle(el, pseudo); const content = cs.content;
      if (!content || ['none', 'normal', '""', "''"].includes(content) || cs.display === 'none') continue;
      const r = el.getBoundingClientRect(); const opacity = opacityOf(el);
      if (r.width < 2 || r.height < 2 || opacity === 0) continue;
      const c = parse(cs.color); const size = parseFloat(cs.fontSize); const weight = Number(cs.fontWeight);
      if (cs.counterIncrement === 'none' && !/^"/.test(content) && !/counter\\(/.test(content)) continue;
      out.push({ kind: 'text', el: describe(el) + pseudo, text: 'CSS content ' + content.slice(0, 20), rgb: c.rgb, alpha: c.a * opacity, size, weight, large: large(size, weight), rects: [rectOf(r)], disabled: false, link: false });
    }
  }
  for (const el of document.body.querySelectorAll('.bi')) {
    if (!shown(el) || el.closest('.tt-visually-hidden, [hidden]')) continue;
    const r = el.getBoundingClientRect(); const opacity = opacityOf(el);
    if (r.width < 2 || r.height < 2 || opacity === 0) continue;
    const cs = getComputedStyle(el); const c = parse(cs.color);
    // An icon hidden behind an opaque image (a fallback glyph under a loaded photo) is not visible.
    const centre = document.elementFromPoint(Math.min(Math.max(r.left + r.width / 2, 0), innerWidth - 1), Math.min(Math.max(r.top + r.height / 2, 0), innerHeight - 1));
    const covered = !!centre && centre !== el && !el.contains(centre);
    const control = el.closest('a, button');
    const visibleText = control && [...control.childNodes].some(n => (n.nodeType === 3 && n.textContent.trim()) || (n.nodeType === 1 && !n.classList.contains('bi') && n.textContent.trim() && !n.classList.contains('tt-visually-hidden')));
    out.push({ kind: 'icon', el: describe(el) + '.' + [...el.classList].find(k => k.startsWith('bi-')), text: el.className.replace(/\\s+/g, ' '), rgb: c.rgb, alpha: c.a * opacity, size: parseFloat(cs.fontSize), weight: 400,
      large: true, rects: [rectOf(r)], disabled: !!el.closest(':disabled'), link: false, covered, iconOnlyControl: !!control && !visibleText, ariaHidden: !!el.closest('[aria-hidden="true"]') });
  }
  return out;
})()`;

// Form-control text and placeholders are measured from computed colours (the control's own fill
// over its parent's), because a select's chevron and a date field's picker icon sit inside the
// same box and would otherwise read as background.
export const collectControls = `(() => {
  const canvas = document.createElement('canvas'); canvas.width = canvas.height = 1;
  const ctx = canvas.getContext('2d', { willReadFrequently: true });
  const parse = value => { ctx.clearRect(0, 0, 1, 1); ctx.fillStyle = '#000'; ctx.fillStyle = value; ctx.fillRect(0, 0, 1, 1); const d = ctx.getImageData(0, 0, 1, 1).data; return { rgb: [d[0], d[1], d[2]], a: d[3] / 255 }; };
  const over = (top, bottom) => top.rgb.map((c, i) => c * top.a + bottom.rgb[i] * (1 - top.a));
  const backdrop = el => {
    const layers = [];
    for (let n = el; n; n = n.parentElement) { const c = parse(getComputedStyle(n).backgroundColor); if (c.a > 0) layers.push(c); if (c.a === 1) break; }
    let rgb = { rgb: [255, 255, 255], a: 1 };
    for (const layer of layers.reverse()) rgb = { rgb: over(layer, rgb), a: 1 };
    return rgb;
  };
  const out = [];
  for (const el of document.querySelectorAll('input:not([type=hidden]), select, textarea')) {
    const r = el.getBoundingClientRect(); if (r.width < 2) continue;
    const cs = getComputedStyle(el); const bg = backdrop(el);
    const id = el.id ? '#' + el.id : '';
    out.push({ el: el.tagName.toLowerCase() + id, kind: 'value', rgb: parse(cs.color).rgb, bg: bg.rgb, size: parseFloat(cs.fontSize), disabled: el.disabled });
    if (el.getAttribute('placeholder') !== null) {
      const p = getComputedStyle(el, '::placeholder');
      out.push({ el: el.tagName.toLowerCase() + id, kind: 'placeholder', rgb: parse(p.color).rgb, bg: bg.rgb, size: parseFloat(cs.fontSize), disabled: el.disabled });
    }
  }
  return out;
})()`;

// Non-text UI components: the boundary a person needs to find a field or button, measured against
// what is around it. Returns computed colours; the caller does the maths.
export const collectComponents = `(() => {
  const canvas = document.createElement('canvas'); canvas.width = canvas.height = 1;
  const ctx = canvas.getContext('2d', { willReadFrequently: true });
  const parse = value => { ctx.clearRect(0, 0, 1, 1); ctx.fillStyle = '#000'; ctx.fillStyle = value; ctx.fillRect(0, 0, 1, 1); const d = ctx.getImageData(0, 0, 1, 1).data; return { rgb: [d[0], d[1], d[2]], a: d[3] / 255 }; };
  const over = (top, bottom) => top.rgb.map((c, i) => c * top.a + bottom.rgb[i] * (1 - top.a));
  const backdrop = el => {
    const layers = [];
    for (let n = el; n; n = n.parentElement) { const c = parse(getComputedStyle(n).backgroundColor); if (c.a > 0) layers.push(c); if (c.a === 1) break; }
    let rgb = { rgb: [255, 255, 255], a: 1 };
    for (const layer of layers.reverse()) rgb = { rgb: over(layer, rgb), a: 1 };
    return rgb.rgb;
  };
  const selectors = ['.trip-input', '.trip-select', '.tt-search', '.trip-submit', '.tt-button', '.tt-menu-button', '.itinerary-action', '.itinerary-directions',
    '.video-expand', '.video-activate', '.itinerary-day-summary', '.tt-search-submit', '.itinerary-day-marker', '.itinerary-chip', '.itinerary-stop-number'];
  const out = [];
  for (const selector of selectors) {
    const el = document.querySelector(selector); if (!el) continue;
    const r = el.getBoundingClientRect(); if (r.width < 2) continue;
    const cs = getComputedStyle(el);
    const parent = el.parentElement ? backdrop(el.parentElement) : [255, 255, 255];
    const border = parse(cs.borderTopColor);
    const own = parse(cs.backgroundColor);
    out.push({ selector, border: parseFloat(cs.borderTopWidth) > 0 ? border.rgb : null, fill: own.a > 0 ? over(own, { rgb: parent, a: 1 }) : null, parent, opacity: Number(cs.opacity) });
  }
  return out;
})()`;

// Structure: headings, ids, ARIA references, images, iframes, forms, link targets.
export const collectStructure = `(() => {
  const visible = el => { const r = el.getBoundingClientRect(); const cs = getComputedStyle(el); return cs.display !== 'none' && cs.visibility !== 'hidden' && !el.closest('[hidden]') && (r.width > 0 || r.height > 0); };
  const headings = [...document.querySelectorAll('h1,h2,h3,h4,h5,h6,[role=heading]')].filter(visible).map(h => ({ level: h.getAttribute('aria-level') ? Number(h.getAttribute('aria-level')) : Number(h.tagName[1]), text: h.textContent.replace(/\\s+/g, ' ').trim().slice(0, 50) }));
  const ids = [...document.querySelectorAll('[id]')].map(n => n.id);
  const dupIds = ids.filter((id, i) => ids.indexOf(id) !== i);
  const refs = [];
  for (const attr of ['aria-labelledby', 'aria-describedby', 'aria-controls', 'for']) {
    for (const el of document.querySelectorAll('[' + attr + ']')) for (const id of el.getAttribute(attr).split(/\\s+/).filter(Boolean)) {
      if (!document.getElementById(id)) refs.push(attr + '=' + id + ' on ' + el.tagName.toLowerCase());
    }
  }
  const images = [...document.querySelectorAll('img')].map(img => ({ src: img.getAttribute('src').slice(0, 60), alt: img.getAttribute('alt'), hidden: !visible(img), inButton: !!img.closest('button'), inLink: !!img.closest('a'), complete: img.complete, natural: img.naturalWidth }));
  const iframes = [...document.querySelectorAll('iframe')].map(f => ({ src: f.getAttribute('src').slice(0, 60), title: f.getAttribute('title') }));
  const controls = [...document.querySelectorAll('input:not([type=hidden]), select, textarea')].map(c => ({
    id: c.id, type: c.type, name: c.name, labelled: (c.labels ? c.labels.length : 0), autocomplete: c.getAttribute('autocomplete'), required: c.required,
    describedBy: c.getAttribute('aria-describedby'), labelText: c.labels && c.labels[0] ? c.labels[0].textContent.replace(/\\s+/g, ' ').trim() : null }));
  const external = [...document.querySelectorAll('a[target=_blank]')].map(a => ({ href: a.getAttribute('href').slice(0, 50), rel: a.rel, name: (a.getAttribute('aria-label') || a.textContent).replace(/\\s+/g, ' ').trim().slice(0, 40) }));
  const positiveTabindex = [...document.querySelectorAll('[tabindex]')].filter(n => Number(n.getAttribute('tabindex')) > 0).map(n => n.tagName.toLowerCase());
  const liveRegions = [...document.querySelectorAll('[role=status],[role=alert],[aria-live],output')].map(n => ({ el: n.tagName.toLowerCase() + (n.id ? '#' + n.id : '') + '.' + n.className.toString().split(' ')[0], role: n.getAttribute('role'), live: n.getAttribute('aria-live'), atomic: n.getAttribute('aria-atomic'), text: n.textContent.trim().slice(0, 60) }));
  return { lang: document.documentElement.lang, title: document.title, viewportMeta: document.querySelector('meta[name=viewport]')?.content, headings, dupIds, refs, images, iframes, controls, external, positiveTabindex, liveRegions,
    h1Count: headings.filter(h => h.level === 1).length };
})()`;

// Interactive targets and their sizes.
export const collectTargets = `(() => {
  const out = [];
  const nodes = document.querySelectorAll('a[href], button, input:not([type=hidden]), select, textarea, summary, [tabindex]:not([tabindex="-1"])');
  for (const el of nodes) {
    if (el.closest('.tt-visually-hidden, [hidden]')) continue;
    const cs = getComputedStyle(el); if (cs.display === 'none' || cs.visibility === 'hidden') continue;
    const r = el.getBoundingClientRect(); if (r.width < 1 || r.height < 1) continue;
    if (el.classList.contains('tt-skip-link')) continue; // measured when focused
    // An inline link inside a sentence is exempt from WCAG 2.5.8; a standalone link is not.
    let sentence = false;
    if (el.tagName === 'A' && cs.display === 'inline') {
      const parent = el.parentElement; const own = el.textContent.length;
      sentence = parent.textContent.replace(/\\s+/g, ' ').trim().length > own + 1;
    }
    out.push({ el: el.tagName.toLowerCase() + (el.id ? '#' + el.id : '') + '.' + [...el.classList].slice(0, 2).join('.'), text: (el.getAttribute('aria-label') || el.textContent).replace(/\\s+/g, ' ').trim().slice(0, 40),
      w: Math.round(r.width * 10) / 10, h: Math.round(r.height * 10) / 10, x: r.left + scrollX, y: r.top + scrollY, sentence });
  }
  return out;
})()`;
