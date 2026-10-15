import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import vm from 'node:vm';

const read = name => readFileSync(new URL(`../../../public/redesign/${name}`, import.meta.url), 'utf8');
const sources = [read('writers-map-leaflet.js'), read('writers-map.js')];
const drawCalls = ['clear', 'clusters', 'control', 'fit', 'flyTo', 'line', 'markers', 'popup'];
const eoxTiles = 'https://tiles.maps.eox.at/wmts/1.0.0/s2cloudless_3857/default/GoogleMapsCompatible/{z}/{y}/{x}.jpg';

// --- MapLibre engine ---

test('loads the pinned MapLibre assets, and not Leaflet, when a map shell appears', () => {
  const app = browser({ maplibre: null });
  app.run();
  app.mount();
  assert.equal(app.assets('SCRIPT').length, 1);
  assert.equal(app.assets('SCRIPT')[0].src, 'https://cdn.jsdelivr.net/npm/maplibre-gl@5.24.0/dist/maplibre-gl.js');
  assert.match(app.assets('SCRIPT')[0].integrity, /^sha384-/);
  assert.equal(app.assets('LINK')[0].href, 'https://cdn.jsdelivr.net/npm/maplibre-gl@5.24.0/dist/maplibre-gl.css');
  assert.match(app.assets('LINK')[0].integrity, /^sha384-/);
  assert.equal(app.assets('LINK')[0].crossOrigin, 'anonymous');
});

test('draws a 3D MapLibre map with the dark default style, terrain and camera', async () => {
  const app = browser();
  app.run();
  app.mount();
  await app.flush();
  const map = app.maplibre[0];
  assert.equal(map.options.style, 'https://tiles.openfreemap.org/styles/dark');
  assert.deepEqual(Array.from(map.options.center), [-16.9, 32.65]);
  assert.deepEqual([map.options.zoom, map.options.pitch, map.options.bearing], [14.3, 55, -20]);
  const source = map.sources['terrain-dem'];
  assert.deepEqual(Array.from(source.tiles), ['https://tiles.mapterhorn.com/{z}/{x}/{y}.webp']);
  assert.equal(source.tileSize, 512);
  assert.equal(map.terrain.source, 'terrain-dem');
  assert.equal(map.layers.some(layer => layer.type === 'hillshade'), false);
  assert.equal(app.leafletMaps.length, 0);
  assert.equal(app.events[0].type, 'travelmap:ready');
  assert.equal(app.events[0].detail.engine, 'maplibre');
  assert.equal(app.events[0].detail.map, map);
});

test('MapLibre controls switch to 2D and restore 3D terrain, with navigation and scale at bottom right', async () => {
  const app = browser();
  app.run();
  app.mount();
  await app.flush();
  const map = app.maplibre[0];
  assert.deepEqual(map.controls.filter(item => !item.custom).map(item => item.position), ['bottom-right']);
  const tools = map.controls.find(item => item.control.onAdd?.().className === 'travelmap-tools').control.onAdd();
  const dimension = tools.children.find(item => item.className === 'travelmap-dimension');
  dimension.click();
  assert.equal(map.pitch, 0);
  assert.equal(map.terrain, null);
  dimension.click();
  assert.equal(map.pitch, 55);
  assert.equal(map.terrain.source, 'terrain-dem');
});

test('compass resets bearing and precedes locate, dimension and zoom controls', async () => {
 const app = browser(); app.run(); app.mount(); await app.flush();
 const map = app.maplibre[0];
 const tools = map.controls.find(item => item.control.onAdd?.().className === 'travelmap-tools');
 assert.equal(tools.position, 'top-right');
 const buttons = tools.control.onAdd().children;
 assert.equal(buttons[0].className, 'travelmap-compass');
 assert.equal(buttons[1].className, 'travelmap-locate');
 assert.equal(buttons[2].className, 'travelmap-dimension');
 assert.deepEqual(buttons[3].children.map(button => button.textContent), ['+', '−']);
 buttons[0].click(); assert.equal(map.ease.bearing, 0);
});

test('fixture camera shows dispersed city pins while production keeps its close view', async () => {
 const app = browser(); app.run(); app.mount('Lisbon', { mapStyleUrl: '/fixture-map/style.json' }); await app.flush();
 assert.equal(app.maplibre[0].options.zoom, 12.3);
});

test('back is hidden before selection, returns to city and restores map focus', async () => {
 const feature = { properties: { qid: 'Q1', name: 'Tower' }, geometry: { coordinates: [-16.91, 32.66] } };
 const app = browser({ features: [feature] }); app.run();
 const shell = app.mount(); await app.flush();
 const map = app.maplibre[0];
 const back = map.controls.find(item => item.control.onAdd?.().className === 'travelmap-back').control.onAdd();
 assert.equal(back.hidden, true);
 app.window.TravelMap.selectPlace('Q1');
 assert.equal(back.hidden, false);
 back.click();
 assert.equal(app.events.at(-1).type, 'travelmap:back');
 assert.deepEqual(Array.from(map.fly.center), [-16.9, 32.65]);
 assert.equal(shell.canvas.focused, true);
});

test('locate me shows a position dot and a denied request reports status without retrying', async () => {
  let request;
  let calls = 0;
  const app = browser({ geolocation: { getCurrentPosition(ok, fail) { calls++; request = { ok, fail }; } } });
  app.run();
  const shell = app.mount();
  await app.flush();
  const tools = app.maplibre[0].controls.find(item => item.control.onAdd?.().className === 'travelmap-tools').control.onAdd();
  const locate = tools.children.find(item => item.className === 'travelmap-locate');
  locate.click();
  request.ok({ coords: { latitude: 32.66, longitude: -16.91 } });
  assert.deepEqual(Array.from(app.maplibre[0].fly.center), [-16.91, 32.66]);
  assert.equal(app.maplibre[0].markers.at(-1).element.className, 'travelmap-location-dot');
  locate.click();
  request.fail({ code: 1 });
  assert.match(shell.status.textContent, /Location unavailable/);
  locate.click();
  assert.equal(calls, 2);
});

test('Leaflet controls keep locate and hide the 2D/3D switch', async () => {
  const app = browser({ webgl: false });
  app.run();
  app.mount();
  await app.flush();
  const map = app.leafletMaps[0];
  assert.equal(map.options.zoomControl, false);
  assert.equal(map.controls.filter(item => !item.custom).every(item => item.options.position === 'bottomright'), true);
  const tools = map.controls.find(item => item.element?.className === 'travelmap-tools').element;
  assert.ok(tools.children.some(item => item.className === 'travelmap-locate'));
  assert.equal(tools.children.some(item => item.className === 'travelmap-dimension'), false);
});

test('adds one dark 3D building layer below the labels when the style has no fill-extrusion layer', async () => {
  const app = browser();
  app.run();
  app.mount();
  await app.flush();
  const buildings = app.maplibre[0].layers.filter(layer => layer.type === 'fill-extrusion');
  assert.equal(buildings.length, 1);
  const [layer] = buildings;
  assert.equal(layer.source, 'openmaptiles');
  assert.equal(layer['source-layer'], 'building');
  assert.equal(layer.minzoom, 14);
  assert.equal(JSON.stringify(layer.paint['fill-extrusion-height']), '["coalesce",["get","render_height"],8]');
  assert.equal(JSON.stringify(layer.paint['fill-extrusion-base']), '["coalesce",["get","render_min_height"],0]');
  assert.ok(layer.paint['fill-extrusion-opacity'] < 1 && layer.paint['fill-extrusion-opacity'] > 0.5);
  assert.equal(app.maplibre[0].addedBefore[layer.id], 'road-label', 'above the roads, below the first label after the buildings');
});

test('adds no building layer when the style already has a fill-extrusion layer', async () => {
  const app = browser({ style: { layers: [{ id: 'building-3d', type: 'fill-extrusion' }, { id: 'labels', type: 'symbol' }] } });
  app.run();
  app.mount();
  await app.flush();
  assert.equal(app.maplibre[0].layers.some(layer => layer.type === 'fill-extrusion'), false);
});

test('adds no building layer when the style has no vector source', async () => {
  const app = browser({ style: { sources: { raster: { type: 'raster' } } } });
  app.run();
  app.mount();
  await app.flush();
  assert.equal(app.maplibre[0].layers.some(layer => layer.type === 'fill-extrusion'), false);
});

test('running style.load again does not duplicate the building layer', async () => {
  const app = browser();
  app.run();
  app.mount();
  await app.flush();
  app.maplibre[0].emit('style.load');
  app.maplibre[0].emit('style.load');
  assert.equal(app.maplibre[0].layers.filter(layer => layer.type === 'fill-extrusion').length, 1);
});

test('applies the Electroverse-like palette to the OpenFreeMap layers on style.load of the default style', async () => {
  const app = browser({ style: openFreeMapStyle() });
  app.run();
  app.mount();
  await app.flush();
  const { paints } = app.maplibre[0];
  assert.equal(paints.background['background-color'], '#202a32');
  assert.equal(paints.water['fill-color'], '#000e29');
  assert.equal(paints.landuse_park['fill-color'], '#29382f');
  assert.equal(paints.landcover_wood['fill-color'], '#29382f');
  assert.equal(paints.landcover_wood['fill-pattern'], null, 'the wood pattern would hide the colour');
  assert.equal(paints.highway_minor['line-color'], '#404860');
  assert.equal(paints.highway_major_inner['line-color'], '#4c576c');
  assert.equal(paints.highway_motorway_inner['line-color'], '#5a6274');
  assert.equal(paints.building['fill-color'], '#2c3044');
  assert.equal(paints.place_city['text-color'], '#e0e0e0');
  assert.equal(paints.place_city['text-halo-color'], '#101820');
  const [layer] = app.maplibre[0].layers.filter(item => item.type === 'fill-extrusion');
  assert.equal(JSON.stringify(layer.paint['fill-extrusion-color']), '["interpolate",["linear"],["coalesce",["get","render_height"],0],0,"#2c3044",80,"#46414d"]');
});

test('thins and dims minor roads, and secondary and tertiary roads, at z10 to z13, and leaves primaries and motorways readable', async () => {
  const app = browser({ style: openFreeMapStyle() });
  app.run();
  app.mount();
  await app.flush();
  const { paints } = app.maplibre[0];
  const json = value => JSON.stringify(value);
  const weight = (major, other) => ['match', ['get', 'class'], ['primary', 'trunk'], major, other];
  assert.equal(json(paints.highway_minor['line-width']), '["interpolate",["exponential",1.55],["zoom"],10,0.4,13,1.2,20,20]');
  assert.equal(json(paints.highway_minor['line-opacity']), '["interpolate",["linear"],["zoom"],10,0.35,14,0.9]');
  assert.equal(json(paints.highway_major_inner['line-width']),
    json(['interpolate', ['exponential', 1.3], ['zoom'], 10, weight(2, 0.7), 13, weight(3.6, 1.4), 20, 20]));
  assert.equal(json(paints.highway_major_casing['line-width']),
    json(['interpolate', ['exponential', 1.3], ['zoom'], 10, weight(3, 1.2), 13, weight(4.8, 2), 20, 23]));
  assert.equal(json(paints.highway_major_subtle['line-width']), json(['interpolate', ['linear'], ['zoom'], 6, 0, 8, weight(1.5, 0.5)]));
  for (const id of ['highway_major_inner', 'highway_major_casing']) {
    assert.equal(json(paints[id]['line-opacity']), json(['interpolate', ['linear'], ['zoom'], 10, weight(1, 0.4), 14, 1]));
  }
  assert.equal(paints.highway_motorway_inner['line-width'], undefined);
  assert.equal(paints.highway_motorway_inner['line-opacity'], undefined);
  assert.equal(paints.highway_motorway_casing['line-width'], undefined);
});

test('sets the place label size through layout properties, and not for the large cities', async () => {
  const app = browser({ style: openFreeMapStyle() });
  app.run();
  app.mount();
  await app.flush();
  const { layouts, paints } = app.maplibre[0];
  assert.equal(layouts.place_city['text-size'], 11);
  assert.equal(layouts.place_town['text-size'], 11);
  assert.equal(layouts.place_suburb['text-size'], 11);
  assert.equal(layouts.place_city_large, undefined);
  assert.equal(paints.place_city['text-size'], undefined, 'text-size is a layout property');
  assert.equal(paints.place_town['text-color'], '#e0e0e0');
  assert.equal(paints.place_suburb['text-halo-color'], '#101820');
});

test('applies no layout overrides to a custom style URL', async () => {
  const app = browser({ style: openFreeMapStyle() });
  app.run();
  app.mount('Madeira', { mapStyleUrl: '/map-fixtures/style.json' });
  await app.flush();
  assert.deepEqual(app.maplibre[0].layoutCalls, []);
});

test('hides generic POIs but retains major landmarks and geographic labels', async () => {
  const app = browser({ style: openFreeMapStyle() });
  app.run();
  app.mount();
  await app.flush();
  const { layouts } = app.maplibre[0];
  assert.equal(layouts.poi.visibility, 'none');
  assert.equal(layouts.poi_z14.visibility, 'none');
  for (const id of ['poi_major', 'place_city', 'highway_name_other', 'water_name']) {
    assert.equal(layouts[id]?.visibility, undefined);
  }
});

test('the four pin colours contrast with the map background by at least 3:1', async () => {
  const app = browser({ style: openFreeMapStyle() });
  app.run();
  app.mount();
  await app.flush();
  const background = app.maplibre[0].paints.background['background-color'];
  const luminance = hex => {
    const rgb = hex.match(/[a-f\d]{2}/gi).map(value => parseInt(value, 16) / 255);
    const linear = rgb.map(value => value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4);
    return linear[0] * 0.2126 + linear[1] * 0.7152 + linear[2] * 0.0722;
  };
  for (const pin of ['#cf795b', '#459b6b', '#a582d4', '#a56824']) {
    const values = [luminance(pin), luminance(background)].sort((a, b) => b - a);
    assert.ok((values[0] + 0.05) / (values[1] + 0.05) >= 3, `${pin} lacks contrast against ${background}`);
  }
});

test('leaves a layer id the style does not have alone', async () => {
  const app = browser({ style: { layers: [{ id: 'water', type: 'fill' }, { id: 'labels', type: 'symbol' }] } });
  app.run();
  app.mount();
  await app.flush();
  assert.deepEqual(Object.keys(app.maplibre[0].paints), ['water']);
});

test('applies no palette overrides to a custom style URL', async () => {
  const app = browser({ style: openFreeMapStyle() });
  app.run();
  app.mount('Madeira', { mapStyleUrl: '/map-fixtures/style.json' });
  await app.flush();
  assert.deepEqual(app.maplibre[0].paintCalls, []);
});

test('running style.load again reapplies the same palette without adding anything', async () => {
  const app = browser({ style: openFreeMapStyle() });
  app.run();
  app.mount();
  await app.flush();
  const map = app.maplibre[0];
  const once = map.paintCalls.length;
  assert.ok(once > 0);
  map.emit('style.load');
  map.emit('style.load');
  assert.equal(map.paintCalls.length, once * 3);
  assert.deepEqual(map.paintCalls.slice(once, once * 2), map.paintCalls.slice(0, once));
  assert.equal(map.layers.filter(layer => layer.type === 'fill-extrusion').length, 1);
});

test('takes the style and terrain from the shell, and an empty terrain URL turns terrain off', async () => {
  const app = browser();
  app.run();
  app.mount('Madeira', { mapStyleUrl: '/map-fixtures/style.json', mapTerrainUrl: '/map-fixtures/dem/{z}/{x}/{y}.png' });
  await app.flush();
  assert.equal(app.maplibre[0].options.style, '/map-fixtures/style.json');
  assert.deepEqual(Array.from(app.maplibre[0].sources['terrain-dem'].tiles), ['/map-fixtures/dem/{z}/{x}/{y}.png']);

  const plain = browser();
  plain.run();
  plain.mount('Madeira', { mapTerrainUrl: '' });
  await plain.flush();
  assert.equal(plain.maplibre[0].terrain, undefined);
  assert.equal(plain.maplibre[0].sources['terrain-dem'], undefined);
});

// --- Fallback ---

test('falls back to Leaflet with maxZoom 18 when WebGL is missing', async () => {
  const app = browser({ webgl: false });
  app.run();
  app.mount();
  await app.flush();
  assert.equal(app.maplibre.length, 0);
  assert.equal(app.leafletMaps[0].options.maxZoom, 18);
  assert.equal(app.leafletMaps[0].tiles[0].options.maxZoom, 18);
  assert.equal(app.events[0].detail.engine, 'leaflet');
  assert.equal(app.events[0].detail.map, app.leafletMaps[0]);
});

test('data-map-engine="leaflet" forces the fallback', async () => {
  const app = browser();
  app.run();
  app.mount('Madeira', { mapEngine: 'leaflet' });
  await app.flush();
  assert.equal(app.maplibre.length, 0);
  assert.equal(app.events[0].detail.engine, 'leaflet');
});

test('a style load error falls back to Leaflet and adds the layers once', async () => {
  const app = browser({ maplibre: { ready: false } });
  app.run();
  const added = [];
  app.window.TravelMap.registerLayer('pins', layerSpy(added));
  app.mount();
  await app.flush();
  app.maplibre[0].emit('error', { error: new Error('style 404') });
  await app.flush();
  assert.equal(app.maplibre[0].removed, true);
  assert.equal(app.leafletMaps.length, 1);
  assert.deepEqual(added.filter(call => call[0] === 'add').map(call => call[1]), ['leaflet']);
  assert.equal(added.filter(call => call[0] === 'remove').length, 0);
  assert.deepEqual(app.events.filter(event => event.type === 'travelmap:ready').map(event => event.detail.engine), ['leaflet']);
});

test('a style that never loads falls back to Leaflet', async () => {
  const app = browser({ maplibre: { ready: false } });
  app.run();
  app.mount();
  await app.flush();
  assert.equal(app.leafletMaps.length, 0);
  app.timers.at(-1).fn();
  await app.flush();
  assert.equal(app.leafletMaps.length, 1);
  assert.equal(app.events[0].detail.engine, 'leaflet');
});

test('repeated tile failures after the layers were added remove them, mount Leaflet and add them again', async () => {
  const app = browser();
  app.run();
  const calls = [];
  app.window.TravelMap.registerLayer('pins', layerSpy(calls));
  app.mount();
  await app.flush();
  assert.deepEqual(calls.map(call => call.slice(0, 2)), [['add', 'maplibre']]);
  const tileError = { sourceId: 'terrain-dem', error: new Error('tile 404') };
  app.maplibre[0].emit('error', tileError);
  app.maplibre[0].emit('error', tileError);
  assert.equal(app.leafletMaps.length, 0, 'two failed tiles are not an outage');
  app.maplibre[0].emit('error', tileError);
  await app.flush();
  assert.deepEqual(calls.map(call => call.slice(0, 2)), [['add', 'maplibre'], ['remove', 'maplibre'], ['add', 'leaflet']]);
  assert.deepEqual(app.events.filter(event => event.type === 'travelmap:ready').map(event => event.detail.engine), ['maplibre', 'leaflet']);
  assert.equal(app.leafletMaps[0].options.maxZoom, 18);
});

test('a style error after the map is ready switches to Leaflet', async () => {
  const app = browser();
  app.run();
  const calls = [];
  app.window.TravelMap.registerLayer('pins', layerSpy(calls));
  app.mount();
  await app.flush();
  app.maplibre[0].emit('error', { error: new Error('style reload failed') });
  await app.flush();
  assert.deepEqual(calls.map(call => call.slice(0, 2)), [['add', 'maplibre'], ['remove', 'maplibre'], ['add', 'leaflet']]);
  assert.equal(app.maplibre[0].removed, true);
});

test('a source outage after earlier successful tiles falls back to Leaflet', async () => {
  const app = browser();
  app.run();
  app.mount();
  await app.flush();
  for (let i = 0; i < 2; i++) app.maplibre[0].emit('error', { sourceId: 'openmaptiles', error: new Error('tile 404') });
  app.maplibre[0].emit('sourcedata', { sourceId: 'openmaptiles', tile: {} });
  for (let i = 0; i < 2; i++) app.maplibre[0].emit('error', { sourceId: 'openmaptiles', error: new Error('tile 404') });
  assert.equal(app.leafletMaps.length, 0, 'one successful tile resets the failure count');
  app.maplibre[0].emit('error', { sourceId: 'openmaptiles', error: new Error('tile 404') });
  await app.flush();
  assert.equal(app.leafletMaps.length, 1);
});

test('keeps the EOX imagery and its credit on the fallback, and swaps to OSM on a tile error', async () => {
  const app = browser({ webgl: false });
  app.run();
  const shell = app.mount();
  await app.flush();
  const tile = app.leafletMaps[0].tiles[0];
  assert.equal(tile.url, eoxTiles);
  assert.match(tile.options.attribution, /EOxCloudless.*EOX IT Services GmbH.*Contains modified Copernicus Sentinel data 2016 &amp; 2017/);
  assert.match(tile.options.attribution, /https:\/\/creativecommons.org\/licenses\/by\/4.0\//);
  tile.emit('tileerror');
  assert.equal(app.leafletMaps[0].tiles.at(-1).url, 'https://tile.openstreetmap.org/{z}/{x}/{y}.png');
  assert.match(app.leafletMaps[0].tiles.at(-1).options.attribution, /OpenStreetMap/);
  assert.equal(app.leafletMaps[0].tiles.at(-1).options.maxZoom, 18);
  assert.ok(shell);
});

test('keeps configured fallback tiles and attribution, and credits OSM when its URL is configured', async () => {
  const app = browser({ webgl: false });
  app.run();
  const shell = app.mount();
  shell.root.dataset.mapTileUrl = '/tiles/%7Bz%7D/%7Bx%7D/%7By%7D.png';
  shell.root.dataset.mapAttribution = 'Configured tiles';
  await app.flush();
  assert.equal(app.leafletMaps[0].tiles[0].url, '/tiles/{z}/{x}/{y}.png');
  assert.equal(app.leafletMaps[0].tiles[0].options.attribution, 'Configured tiles');

  const osm = browser({ webgl: false });
  osm.run();
  osm.mount().root.dataset.mapTileUrl = 'https://tile.openstreetmap.org/{z}/{x}/{y}.png';
  await osm.flush();
  assert.match(osm.leafletMaps[0].tiles[0].options.attribution, /OpenStreetMap/);
});

test('loads the pinned Leaflet assets only for the fallback', () => {
  const app = browser({ webgl: false, leaflet: null });
  app.run();
  app.mount();
  assert.equal(app.assets('SCRIPT')[0].src, 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.js');
  assert.match(app.assets('SCRIPT')[0].integrity, /^sha256-/);
  assert.match(app.assets('LINK')[0].integrity, /^sha256-/);
  assert.equal(app.assets('LINK')[0].crossOrigin, 'anonymous');
});

// --- Drawing API ---

test('context.draw has the same calls on both engines', async () => {
  const shapes = {};
  for (const webgl of [true, false]) {
    const app = browser({ webgl });
    app.run();
    let draw;
    app.window.TravelMap.registerLayer('probe', { add: (_, context) => { draw = context.draw; }, remove() {} });
    app.mount();
    await app.flush();
    shapes[webgl ? 'maplibre' : 'leaflet'] = Object.keys(draw).sort();
  }
  assert.deepEqual(shapes.maplibre, drawCalls);
  assert.deepEqual(shapes.leaflet, drawCalls);
});

for (const engine of ['maplibre', 'leaflet']) {
  test(`${engine}: markers, line and control draw, replace by id, and clear`, async () => {
    const { app, draw } = await drawn(engine);
    const clicks = [];
    const items = ids => ids.map((id, i) => ({
      lngLat: [-16.9 + i / 100, 32.65], element: app.element('BUTTON'), title: `Stop ${id}`, z: i, onClick: () => clicks.push(id)
    }));
    draw.markers('plan', items([1, 2, 3]));
    draw.markers('plan', items([1, 2]));
    draw.markers('other', items([9]));
    assert.equal(app.visible().markers, 3, 'replace by id leaves two plus one');
    const first = app.markerElements()[0];
    assert.equal(first.title, 'Stop 1');
    assert.equal(first['aria-label'], 'Stop 1');
    first.click();
    assert.deepEqual(clicks, [1]);

    draw.line('plan', [[-16.9, 32.65], [-16.8, 32.66]], { color: '#a64027', width: 4, dash: [2, 1] });
    draw.line('plan', [[-16.9, 32.65], [-16.7, 32.7]], { color: '#a64027', width: 4 });
    draw.control('plan', app.element('DIV'), 'top-right');
    draw.control('plan', app.element('DIV'), 'top-right');
    assert.deepEqual(app.visible(), { markers: 3, lines: 1, controls: 1 });

    draw.clear('plan');
    draw.clear('unknown');
    assert.deepEqual(app.visible(), { markers: 1, lines: 0, controls: 0 });
  });

  test(`${engine}: fit and flyTo move the camera`, async () => {
    const { app, draw } = await drawn(engine);
    draw.fit([[-17, 32.6], [-16.8, 32.7]], { padding: 40 });
    draw.flyTo([-16.9, 32.65], 15);
    assert.deepEqual(JSON.parse(JSON.stringify(app.camera())), { fit: [[-17, 32.6], [-16.8, 32.7]], padding: 40, fly: [-16.9, 32.65], zoom: 15 });
  });

  test(`${engine}: popup shows an element at a point, replaces by id, and clear closes it`, async () => {
    const { app, draw } = await drawn(engine);
    const [first, second, other] = [app.element('DIV'), app.element('DIV'), app.element('DIV')];
    draw.popup('pin', [-16.9, 32.65], first);
    draw.popup('pin', [-16.8, 32.7], second);
    draw.popup('other', [-16.7, 32.6], other);
    assert.deepEqual(app.popups().map(popup => popup.content), [second, other], 'replace by id leaves one per id');
    assert.deepEqual(JSON.parse(JSON.stringify(app.popups()[0].lngLat)), [-16.8, 32.7]);
    draw.clear('pin');
    draw.clear('unknown');
    assert.deepEqual(app.popups().map(popup => popup.content), [other]);
  });

  test(`${engine}: onClose runs when the engine closes the popup, not on replace or clear`, async () => {
    const { app, draw } = await drawn(engine);
    const closed = [];
    draw.popup('pin', [-16.9, 32.65], app.element('DIV'), { onClose: () => closed.push('first') });
    draw.popup('pin', [-16.9, 32.65], app.element('DIV'), { onClose: () => closed.push('second') });
    assert.deepEqual(closed, []);
    app.closePopupFromEngine();
    assert.deepEqual(closed, ['second']);
    assert.equal(app.popups().length, 0);
    draw.popup('pin', [-16.9, 32.65], app.element('DIV'), { onClose: () => closed.push('third') });
    draw.clear('pin');
    assert.deepEqual(closed, ['second']);
  });

  test(`${engine}: opens on the destination at the 200 m view and does not fit the camera to the pins`, async () => {
    const places = (...rows) => rows.map(([qid, lon, lat, writers]) => ({ geometry: { coordinates: [lon, lat] }, properties: { qid, name: qid, writer_count: writers } }));
    const fitOf = app => engine === 'maplibre' ? app.maplibre[0].fit : app.leafletMaps[0].fit;
    for (const features of [places(['Q1', -17.2, 32.7, 2], ['Q2', -16.7, 32.8, 1]), places(['Q3', -16.0, 32.9, 0]), []]) {
      const app = browser({ webgl: engine === 'maplibre', features });
      app.run();
      app.mount();
      await app.flush();
      assert.equal(fitOf(app), null, 'the view stays on the destination; pins outside it are reached by panning');
    }
    const app = browser({ webgl: engine === 'maplibre', features: places(['Q1', -17.2, 32.7, 2]) });
    app.run();
    app.mount();
    await app.flush();
    if (engine === 'maplibre') {
      assert.deepEqual(Array.from(app.maplibre[0].options.center), [-16.9, 32.65]);
      assert.equal(app.maplibre[0].options.zoom, 14.3);
    } else {
      assert.deepEqual(JSON.parse(JSON.stringify(app.leafletMaps[0].views[0])), [[32.65, -16.9], 15]);
    }
  });

  test(`${engine}: a layer that clears its ids on remove gets the same result from add after remove`, async () => {
    const app = browser({ webgl: engine === 'maplibre' });
    app.run();
    const layer = {
      add: (_, context) => {
        context.draw.markers('pins', [{ lngLat: [-16.9, 32.65], element: app.element('BUTTON'), title: 'A', z: 1 }]);
        context.draw.line('pins', [[-16.9, 32.65], [-16.8, 32.7]], { color: '#000', width: 2 });
      },
      remove: (_, context) => context.draw.clear('pins')
    };
    app.window.TravelMap.registerLayer('pins', layer);
    app.mount();
    await app.flush();
    const first = app.visible();
    app.window.TravelMap.setLayerEnabled('pins', false);
    assert.deepEqual(app.visible(), { markers: 0, lines: 0, controls: 0 });
    app.window.TravelMap.setLayerEnabled('pins', true);
    assert.deepEqual(app.visible(), first);
    assert.deepEqual(first, { markers: 1, lines: 1, controls: 0 });
  });
}

test('maplibre: a click on the map closes the popup, but a click on a pin (which also reaches the map) does not', async () => {
  const { app, draw } = await drawn('maplibre');
  const closed = [];
  const map = app.maplibre[0];
  const click = onMarker => map.emit('click', { originalEvent: { target: { closest: selector => onMarker && selector === '.maplibregl-marker' ? {} : null } } });
  draw.popup('pin', [-16.9, 32.65], app.element('DIV'), { onClose: () => closed.push('closed') });
  click(true);
  assert.equal(app.popups().length, 1);
  click(false);
  assert.equal(app.popups().length, 0);
  assert.deepEqual(closed, ['closed']);
  draw.popup('pin', [-16.9, 32.65], app.element('DIV'));
  draw.clear('pin');
  assert.equal((map.handlers.click || []).length, 0, 'no click handler is left behind');
});

test('leaflet: a click on the map closes the popup, but a click on a pin (which Leaflet also sends to the map) does not', async () => {
  const { app, draw } = await drawn('leaflet');
  const closed = [];
  const map = app.leafletMaps[0];
  draw.popup('pin', [-16.9, 32.65], app.element('DIV'), { onClose: () => closed.push('closed') });
  assert.equal(app.popups().length, 1);
  map.emit('click', { originalEvent: { target: { closest: selector => selector === '.leaflet-marker-icon' ? {} : null } } });
  assert.equal(app.popups().length, 1);
  map.emit('click', { originalEvent: { target: { closest: () => null } } });
  assert.equal(app.popups().length, 0);
  assert.deepEqual(closed, ['closed']);
  draw.popup('pin', [-16.9, 32.65], app.element('DIV'));
  draw.clear('pin');
  assert.equal((map.handlers.click || []).length, 0, 'no click handler is left behind');
});

test('maplibre: pins stay fully opaque when terrain or 3D buildings cover them', async () => {
  const { app, draw } = await drawn('maplibre');
  draw.markers('pins', [{ lngLat: [-16.9, 32.65], element: app.element('BUTTON'), title: 'A' }]);
  assert.equal(app.maplibre[0].markers.at(-1).options.opacityWhenCovered, '1');
});

test('maplibre: a line drawn without a colour uses the bright default for the dark map', async () => {
  const { app, draw } = await drawn('maplibre');
  draw.line('plain', [[-16.9, 32.65], [-16.8, 32.7]]);
  assert.equal(app.maplibre[0].layers.find(layer => layer.id === 'draw-line-plain').paint['line-color'], '#ff8a65');
});

test('a style reload re-adds lines without duplicating markers', async () => {
  const { app, draw } = await drawn('maplibre');
  draw.markers('plan', [{ lngLat: [-16.9, 32.65], element: app.element('BUTTON'), title: 'A', z: 1 }]);
  draw.line('plan', [[-16.9, 32.65], [-16.8, 32.7]], { color: '#a64027', width: 4 });
  app.maplibre[0].dropCustomLayers();
  assert.equal(app.visible().lines, 0);
  app.maplibre[0].emit('style.load');
  app.maplibre[0].emit('style.load');
  assert.deepEqual(app.visible(), { markers: 1, lines: 1, controls: 0 });
  assert.equal(app.maplibre[0].sources['terrain-dem'] !== undefined, true);
});

test('MapLibre clusters writer places with a count and expands a clicked cluster', async () => {
  const { app, draw } = await drawn('maplibre');
  const features = [
    { type: 'Feature', geometry: { type: 'Point', coordinates: [-16.9, 32.7] }, properties: { qid: 'Q1', name: 'First' } },
    { type: 'Feature', geometry: { type: 'Point', coordinates: [-16.91, 32.71] }, properties: { qid: 'Q2', name: 'Second' } }
  ];
  const clicked = [];
  draw.clusters('writers', features, { labelled: ['Q1'] }, feature => clicked.push(feature.properties.qid));
  const map = app.maplibre[0];
  assert.equal(map.sources['draw-clusters-writers'].cluster, true);
  assert.equal(map.sources['draw-clusters-writers'].data.features.length, 2);
  assert.deepEqual(map.layers.filter(layer => layer.source === 'draw-clusters-writers').map(layer => layer.id),
    ['draw-clusters-writers-group', 'draw-clusters-writers-count', 'draw-clusters-writers-single', 'draw-clusters-writers-single-count', 'draw-clusters-writers-label']);
  assert.equal(map.layers.find(layer => layer.id === 'draw-clusters-writers-label').minzoom, 15);
  const group = { type: 'Feature', geometry: { coordinates: [-16.9, 32.7] }, properties: { cluster_id: 7, point_count: 2 } };
  map.emitLayer('click', 'draw-clusters-writers-group', { features: [group] });
  await app.flush();
  assert.equal(map.fly.zoom, 16);
  assert.deepEqual(Array.from(map.fly.center), [-16.9, 32.7]);
  map.emitLayer('click', 'draw-clusters-writers-single', { features: [features[0]] });
  assert.deepEqual(clicked, ['Q1']);
  map.renderedFeatures = [group];
  map.emit('moveend');
  const list = map.controls.find(item => item.control.onAdd?.().className === 'travelmap-cluster-list').control.onAdd();
  assert.equal(list.children[0].children[0].textContent, 'Zoom to 2 writer places');
  list.children[0].children[0].click();
  await app.flush();
  assert.equal(map.fly.zoom, 16);
  assert.equal(list.children[1].children[0]['data-qid'], 'Q1');
  list.children[1].children[0].click();
  assert.deepEqual(clicked, ['Q1', 'Q1']);
  draw.clear('writers');
  assert.equal(map.sources['draw-clusters-writers'], undefined);
});

test('Leaflet draws writer clusters as individual markers', async () => {
  const { app, draw } = await drawn('leaflet');
  const features = [
    { type: 'Feature', geometry: { coordinates: [-16.9, 32.7] }, properties: { qid: 'Q1', name: 'First' } },
    { type: 'Feature', geometry: { coordinates: [-16.91, 32.71] }, properties: { qid: 'Q2', name: 'Second' } }
  ];
  const clicked = [];
  draw.clusters('writers', features, { labelled: ['Q1'] }, feature => clicked.push(feature.properties.qid));
  const markers = app.leafletMaps[0].layers.filter(layer => layer.kind === 'marker');
  assert.equal(markers.length, 2);
  const label = markers[0].element.children.find(child => child.className === 'map-pin-label');
  assert.equal(label.hidden, true);
  app.leafletMaps[0].zoom = 16;
  app.leafletMaps[0].emit('zoomend');
  assert.equal(label.hidden, false);
  markers[0].element.click();
  assert.deepEqual(clicked, ['Q1']);
});

test('a line drawn while the style is reloading waits for style.load instead of throwing', async () => {
  const { app, draw } = await drawn('maplibre');
  app.maplibre[0].styleLoading = true;
  assert.doesNotThrow(() => draw.line('plan', [[-16.9, 32.65], [-16.8, 32.7]], { color: '#a64027', width: 4 }));
  assert.equal(app.visible().lines, 0);
  app.maplibre[0].styleLoading = false;
  app.maplibre[0].emit('style.load');
  assert.equal(app.visible().lines, 1);
});

// --- Layers ---

test('a layer that throws in add is logged and skipped, and the others still draw', async () => {
  const app = browser();
  app.run();
  const added = [];
  app.window.TravelMap.registerLayer('old', { add() { throw new TypeError('L is not defined'); }, remove() {} });
  app.window.TravelMap.registerLayer('pins', layerSpy(added));
  app.mount();
  await app.flush();
  assert.deepEqual(added.map(call => call[0]), ['add']);
  assert.match(app.warnings[0], /old.*L is not defined/);
  app.window.TravelMap.setLayerEnabled('old', false);
  assert.equal(app.events.filter(event => event.type === 'travelmap:ready').length, 1);
});

test('gives layers the engine, destination, features and shell in the context', async () => {
  const app = browser();
  app.run();
  let seen;
  app.window.TravelMap.registerLayer('probe', { add: (_, context) => { seen = context; }, remove() {} });
  const shell = app.mount();
  await app.flush();
  assert.equal(seen.engine, 'maplibre');
  assert.equal(seen.destination, 'Madeira');
  assert.equal(seen.shell, shell.root);
  assert.equal(seen.features[0].properties.qid, 'Q1');
  assert.equal(typeof seen.selectPlace, 'function');
});

test('the full-map card shows the destination photo and restores the map icon when it fails to load', async () => {
  const app = browser({ features: [{ properties: { qid: 'Q1', name: 'Funchal', photo_url: 'https://example.org/funchal.jpg' } }] });
  app.run();
  const shell = app.mount('funchal-pt');
  await app.flush();
  const thumbnail = shell.fullMapIcon.replacement;
  assert.equal(thumbnail?.className, 'map-action-photo');
  assert.equal(thumbnail.src, 'https://example.org/funchal.jpg');
  thumbnail.dispatch('error');
  assert.equal(thumbnail.replacement, shell.fullMapIcon, 'a broken photo gives the map icon back');
});

// --- Core behaviour kept from before ---

test('hides loading status, renders no place list, and shows partial data status', async () => {
  const app = browser();
  app.run();
  const shell = app.mount();
  assert.equal(shell.status.textContent, '');
  assert.equal(shell.status.hidden, true);
  await app.flush();
  assert.equal(app.events[0].type, 'travelmap:ready');
  assert.equal(shell.status.textContent, 'Some map sources are unavailable.');
  assert.equal(shell.list.children.length, 0, 'nothing renders a place-tag row');
  assert.equal(sources[1].includes('renderList'), false);
});

test('reuses the map across swaps and drops an old response', async () => {
  const app = browser({ delayFirstFetch: true });
  app.run();
  const first = app.mount();
  await app.flush();
  app.body.emit('htmx:beforeSwap', { target: { contains: node => node === first.root } });
  const second = app.mount('Paris');
  await app.flush();
  app.resolveFirstFetch();
  await app.flush();
  assert.equal(app.maplibre.length, 1);
  assert.equal(app.maplibre[0].jumps.length, 1);
  assert.equal(second.canvas.replacement, first.canvas);
  assert.equal(app.assets('LINK').length, 1);
  assert.equal(app.events.filter(event => event.type === 'travelmap:ready').length, 2);
  assert.equal(second.list.children.length, 0);
});

test('a swap clears what the layers drew before the map is reused', async () => {
  const app = browser();
  app.run();
  const layer = {
    add: (_, context) => context.draw.markers('pins', [{ lngLat: [0, 0], element: app.element('BUTTON'), title: 'A' }]),
    remove: (_, context) => context.draw.clear('pins')
  };
  app.window.TravelMap.registerLayer('pins', layer);
  const first = app.mount();
  await app.flush();
  app.body.emit('htmx:beforeSwap', { target: { contains: node => node === first.root } });
  app.mount('Paris');
  await app.flush();
  assert.equal(app.visible().markers, 1);
});

// --- Loading more places ---

const pin = qid => ({ properties: { qid, name: qid } });
const pollTimers = app => app.timers.filter(timer => timer.ms === 3000);

test('polls while places are loading, adds the new pins and says it is loading', async () => {
  const app = browser({ pages: [
    { loading: true, complete: false, features: [pin('Q1')] },
    { loading: true, complete: false, features: [pin('Q1'), pin('Q2')] },
    { loading: false, complete: true, features: [pin('Q1'), pin('Q2'), pin('Q3')] }
  ] });
  app.run();
  const seen = [];
  app.window.TravelMap.registerLayer('probe', { add: (_, context) => seen.push(context.features.length), remove() {} });
  const shell = app.mount();
  await app.flush();
  assert.deepEqual(seen, [1]);
  assert.equal(shell.status.textContent, 'Loading more places…');
  assert.equal(shell.status.hidden, false);
  assert.equal(pollTimers(app).length, 1, 'one poll is scheduled');

  pollTimers(app)[0].fn();
  await app.flush();
  assert.deepEqual(seen, [1, 2]);
  assert.equal(shell.status.textContent, 'Loading more places…');

  pollTimers(app)[1].fn();
  await app.flush();
  assert.deepEqual(seen, [1, 2, 3]);
  assert.equal(shell.status.textContent, '');
  assert.equal(shell.status.hidden, true);
  assert.equal(pollTimers(app).length, 2, 'no poll is scheduled once loading is false');
});

test('a finished load with missing sources still reports them after polling', async () => {
  const app = browser({ pages: [
    { loading: true, complete: false, features: [] },
    { loading: false, complete: false, features: [pin('Q1')] }
  ] });
  app.run();
  const shell = app.mount();
  await app.flush();
  assert.equal(shell.status.textContent, 'Loading more places…');
  pollTimers(app)[0].fn();
  await app.flush();
  assert.equal(shell.status.textContent, 'Some map sources are unavailable.');
});

test('stops polling when the map is swapped to another destination', async () => {
  const app = browser({ pages: [{ loading: true, complete: false, features: [pin('Q1')] }] });
  app.run();
  const first = app.mount();
  await app.flush();
  const stale = pollTimers(app)[0];
  app.body.emit('htmx:beforeSwap', { target: { contains: node => node === first.root } });
  app.mount('Paris');
  await app.flush();
  const before = pollTimers(app).length;
  stale.fn();
  await app.flush();
  assert.equal(pollTimers(app).length, before, 'the old destination schedules nothing more');
});

test('toggles registered layers and reports unavailable places', async () => {
  const app = browser({ failFetch: true });
  app.run();
  const calls = [];
  app.window.TravelMap.registerLayer('places', {
    add: (_, context) => calls.push(['add', context.destination]),
    remove: () => calls.push(['remove'])
  });
  const shell = app.mount();
  await app.flush();
  assert.deepEqual(calls, [['add', 'Madeira']]);
  app.window.TravelMap.setLayerEnabled('places', false);
  assert.deepEqual(calls.at(-1), ['remove']);
  assert.equal(shell.status.textContent, 'Places are unavailable. The map is still available.');
  app.window.TravelMap.updatePlan([{ qids: ['Q1'] }]);
  assert.equal(app.events.at(-1).type, 'travelmap:plan-updated');
  assert.throws(() => app.window.TravelMap.registerLayer('places', { add() {}, remove() {} }), /duplicate/);
});

test('says the map is unavailable, without a place list, when neither engine can load', async () => {
  const app = browser({ maplibre: null, leaflet: null, webgl: true });
  app.run();
  const shell = app.mount();
  app.assets('SCRIPT')[0].onerror(new Error('offline'));
  await app.flush();
  app.assets('SCRIPT')[1].onerror(new Error('offline'));
  await app.flush();
  assert.equal(shell.list.children.length, 0);
  assert.equal(shell.status.textContent, 'Map unavailable.');
  assert.equal(shell.status.hidden, false);
});

test('falls back to Leaflet when the MapLibre script cannot load', async () => {
  const app = browser({ maplibre: null, leaflet: null });
  app.run();
  app.mount();
  app.assets('SCRIPT')[0].onerror(new Error('offline'));
  await app.flush();
  assert.equal(app.assets('SCRIPT')[1].src, 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.js');
  app.window.L = leafletStub(app.leafletMaps);
  app.assets('SCRIPT')[1].onload();
  await app.flush();
  assert.equal(app.events[0].detail.engine, 'leaflet');
});

// --- Test doubles ---

function layerSpy(calls) {
  return {
    add: (map, context) => calls.push(['add', context.engine, map]),
    remove: (map, context) => calls.push(['remove', context.engine, map])
  };
}

async function drawn(engine) {
  const app = browser({ webgl: engine === 'maplibre' });
  app.run();
  let draw;
  app.window.TravelMap.registerLayer('probe', { add: (_, context) => { draw = context.draw; }, remove() {} });
  app.mount();
  await app.flush();
  return { app, draw };
}

// options: webgl (default true); maplibre: null (not loaded yet) or { ready: false } (never loads on its own);
// leaflet: null (not loaded yet); style: the stub MapLibre style's sources and layers.
function browser({ failFetch = false, delayFirstFetch = false, webgl = true, maplibre = {}, leaflet = {}, style = {}, features = null, pages = null, geolocation = null } = {}) {
  const nodes = [];
  const events = [];
  const warnings = [];
  const timers = [];
  const maplibreMaps = [];
  const leafletMaps = [];
  const listeners = new Map();
  let resolveFirstFetch;
  let fetchCount = 0;
  const body = element('BODY');
  body.addEventListener = (name, callback) => listeners.set(name, callback);
  body.emit = (name, detail) => listeners.get(name)({ detail });
  const makeShell = name => {
    const root = element('SECTION');
    root.dataset = { mapLat: '32.65', mapLon: '-16.9', mapUrl: '/map/' + name.toLowerCase() + '.geojson', mapDestination: name };
    const canvas = element('DIV');
    const list = element('DIV');
    const status = element('DIV');
    const fullMapIcon = element('I');
    const fullMap = element('A');
    fullMap.querySelector = selector => (selector === '.bi-map' ? fullMapIcon : null);
    root.querySelector = selector => ({
      '[data-map-canvas]': canvas,
      '[data-map-slot="list"]': list,
      '[data-map-slot="status"]': status,
      '.map-action': fullMap
    })[selector] || null;
    return { root, canvas, list, status, fullMapIcon };
  };
  const document = {
    body,
    head: { appendChild(node) { nodes.push(node); } },
    createElement(tag) {
      const node = element(tag);
      if (tag === 'canvas') node.getContext = type => webgl && type === 'webgl2' ? {} : null;
      return node;
    },
    querySelector() { return null; },
    addEventListener(name, callback) { listeners.set(name, callback); },
    dispatchEvent(event) { events.push(event); listeners.get(event.type)?.(event); }
  };
  const window = { navigator: { geolocation } };
  if (maplibre) window.maplibregl = maplibreStub(maplibreMaps, maplibre.ready !== false, style);
  if (leaflet) window.L = leafletStub(leafletMaps);
  const context = vm.createContext({
    document, window,
    console: { warn: (...args) => warnings.push(args.map(String).join(' ')), error() {}, log() {} },
    setTimeout: (fn, ms) => { timers.push({ fn, ms }); return timers.length; },
    clearTimeout() {},
    fetch: () => {
      fetchCount++;
      if (failFetch) return Promise.reject(new Error('unavailable'));
      if (pages) return Promise.resolve({ ok: true, json: () => Promise.resolve(pages[Math.min(fetchCount - 1, pages.length - 1)]) });
      const response = { ok: true, json: () => Promise.resolve({ complete: false, features: features || [
        { properties: { qid: 'Q1', name: 'Pico do Arieiro' } }
      ] }) };
      if (delayFirstFetch && fetchCount === 1) return new Promise(resolve => { resolveFirstFetch = () => resolve(response); });
      return Promise.resolve(response);
    },
    CustomEvent: class { constructor(type, options) { this.type = type; this.detail = options.detail; } }
  });
  const engineOf = () => (maplibreMaps.length && !maplibreMaps.at(-1).removed ? 'maplibre' : 'leaflet');
  const app = {
    window, events, warnings, timers, body,
    maplibre: maplibreMaps, leafletMaps,
    assets: tagName => nodes.filter(node => node.tagName === tagName),
    run: () => sources.forEach(source => vm.runInContext(source, context)),
    mount(name = 'Madeira', dataset = {}) { const shell = makeShell(name); Object.assign(shell.root.dataset, dataset); window.TravelMap.mount(shell.root); return shell; },
    resolveFirstFetch: () => resolveFirstFetch(),
    flush: async () => { await new Promise(resolve => setImmediate(resolve)); await new Promise(resolve => setImmediate(resolve)); },
    element,
    visible() {
      if (engineOf() === 'maplibre') {
        const map = maplibreMaps.at(-1);
        return {
          markers: map.markers.filter(marker => !marker.removed).length,
          lines: map.layers.filter(layer => layer.id.startsWith('draw-line')).length,
          controls: map.controls.filter(control => control.custom && !['travelmap-tools', 'travelmap-back'].includes(control.control.onAdd().className)).length
        };
      }
      const map = leafletMaps.at(-1);
      return {
        markers: map.layers.filter(layer => layer.kind === 'marker').length,
        lines: map.layers.filter(layer => layer.kind === 'line').length,
        controls: map.controls.filter(control => control.custom && !['travelmap-tools', 'travelmap-back'].includes(control.element.className)).length
      };
    },
    popups() {
      if (engineOf() === 'maplibre') return maplibreMaps.at(-1).popups.filter(popup => !popup.removed).map(popup => ({ content: popup.content, lngLat: popup.lngLat }));
      return leafletMaps.at(-1).layers.filter(layer => layer.kind === 'popup').map(popup => ({ content: popup.content, lngLat: popup.latlng.slice().reverse() }));
    },
    closePopupFromEngine() {
      if (engineOf() === 'maplibre') maplibreMaps.at(-1).popups.filter(popup => !popup.removed).forEach(popup => popup.remove());
      else { const map = leafletMaps.at(-1); map.layers.filter(layer => layer.kind === 'popup').forEach(popup => map.removeLayer(popup)); }
    },
    markerElements() {
      if (engineOf() === 'maplibre') return maplibreMaps.at(-1).markers.filter(marker => !marker.removed).map(marker => marker.element);
      return leafletMaps.at(-1).layers.filter(layer => layer.kind === 'marker').map(layer => layer.element);
    },
    camera() {
      if (engineOf() === 'maplibre') {
        const map = maplibreMaps.at(-1);
        return { fit: map.fit[0], padding: map.fit[1].padding, fly: map.fly.center, zoom: map.fly.zoom };
      }
      const map = leafletMaps.at(-1);
      return { fit: [map.fit[0][0].slice().reverse(), map.fit[0][1].slice().reverse()], padding: map.fit[1].padding[0], fly: map.fly[0].slice().reverse(), zoom: map.fly[1] };
    }
  };
  return app;
}

// The OpenFreeMap dark layer ids the palette table targets.
function openFreeMapStyle() {
  const ids = ['background', 'water', 'landuse_park', 'landcover_wood', 'highway_minor', 'highway_major_inner', 'highway_major_casing', 'highway_major_subtle', 'highway_motorway_inner', 'highway_motorway_casing', 'building', 'place_city', 'place_town', 'place_suburb', 'place_city_large', 'poi', 'poi_z14', 'poi_major', 'highway_name_other', 'water_name'];
  return { layers: ids.map(id => ({ id, type: id.startsWith('place_') || id.startsWith('poi') || id.endsWith('_name_other') || id === 'water_name' ? 'symbol' : id === 'background' ? 'background' : id.startsWith('highway') ? 'line' : 'fill', ...(id === 'building' ? { 'source-layer': 'building' } : {}) })) };
}

function maplibreStub(maps, auto, style) {
  const sources = style.sources || { openmaptiles: { type: 'vector' } };
  const styleLayers = style.layers || [
    { id: 'water', type: 'fill' }, { id: 'water-label', type: 'symbol' }, { id: 'building', type: 'fill', 'source-layer': 'building' },
    { id: 'road', type: 'line' }, { id: 'road-label', type: 'symbol' }, { id: 'place-label', type: 'symbol' }
  ];
  class Map {
    constructor(options) {
      Object.assign(this, { options, addedBefore: {}, paints: {}, paintCalls: [], layouts: {}, layoutCalls: [], handlers: {}, sources: {}, layers: [], markers: [], popups: [], controls: [], jumps: [], fit: null, fly: null });
      maps.push(this);
      if (auto) setImmediate(() => { this.emit('style.load'); this.emit('load'); });
    }
    on(name, layer, callback) { (this.handlers[name + (callback ? ':' + layer : '')] ??= []).push(callback || layer); return this; }
    off(name, layer, callback) { const key = name + (callback ? ':' + layer : ''); this.handlers[key] = (this.handlers[key] || []).filter(item => item !== (callback || layer)); return this; }
    emit(name, event) { (this.handlers[name] || []).slice().forEach(callback => callback(event)); }
    emitLayer(name, layer, event) { (this.handlers[name + ':' + layer] || []).slice().forEach(callback => callback(event)); }
    addSource(id, source) { if (this.styleLoading) throw new Error('Style is not done loading.'); this.sources[id] = source; if (source.cluster) source.getClusterExpansionZoom = () => Promise.resolve(16); }
    getSource(id) { return this.sources[id]; }
    queryRenderedFeatures() { return this.renderedFeatures || []; }
    removeSource(id) { delete this.sources[id]; }
    addLayer(layer, before) { this.layers.push(layer); this.addedBefore[layer.id] = before; }
    getStyle() { return { sources, layers: [...styleLayers, ...this.layers] }; }
    getLayer(id) { return this.layers.find(layer => layer.id === id); }
    removeLayer(id) { this.layers = this.layers.filter(layer => layer.id !== id); }
    dropCustomLayers() { this.layers = []; Object.keys(this.sources).forEach(id => delete this.sources[id]); }
    setPaintProperty(id, name, value) { this.paintCalls.push([id, name, value]); (this.paints[id] ??= {})[name] = value; }
    setLayoutProperty(id, name, value) { this.layoutCalls.push([id, name, value]); (this.layouts[id] ??= {})[name] = value; }
    setTerrain(terrain) { this.terrain = terrain; }
    setPitch(pitch) { this.pitch = pitch; }
    setSky() {}
    addControl(control, position) { this.controls.push({ control, position, custom: Boolean(control.onAdd && !control.native) }); }
    removeControl(control) { this.controls = this.controls.filter(item => item.control !== control); }
    fitBounds(bounds, options) { this.fit = [bounds, options]; }
    flyTo(options) { this.fly = options; }
    easeTo(options) { this.ease = options; }
    jumpTo(options) { this.jumps.push(options); }
    getZoom() { return this.jumps.at(-1)?.zoom || this.options.zoom; }
    resize() {}
    remove() { this.removed = true; }
  }
  class Marker {
    constructor(options) { this.element = options.element; this.options = options; }
    setLngLat(lngLat) { this.lngLat = lngLat; return this; }
    addTo(map) { map.markers.push(this); return this; }
    remove() { this.removed = true; return this; }
  }
  class Popup {
    constructor(options) { this.options = options; this.handlers = {}; }
    setDOMContent(content) { this.content = content; return this; }
    setLngLat(lngLat) { this.lngLat = lngLat; return this; }
    on(name, callback) { (this.handlers[name] ??= []).push(callback); return this; }
    addTo(map) { map.popups.push(this); return this; }
    remove() { if (this.removed) return this; this.removed = true; (this.handlers.close || []).forEach(callback => callback()); return this; }
  }
  class Control { constructor() { this.native = true; } }
  return { Map, Marker, Popup, NavigationControl: Control, ScaleControl: Control, AttributionControl: Control };
}

function leafletStub(maps) {
  const control = options => ({
    options, custom: true,
    addTo(map) { this.map = map; this.element = this.onAdd(map); map.controls.push(this); return this; },
    remove() { this.map.controls = this.map.controls.filter(item => item !== this); return this; }
  });
  control.scale = options => ({ options, addTo(map) { map.controls.push({ custom: false, options }); return this; } });
  control.zoom = options => ({ options, addTo(map) { map.controls.push({ custom: false, options }); return this; } });
  return {
    map(canvas, options) {
      const map = {
        canvas, options, views: [], tiles: [], layers: [], controls: [], fit: null, fly: null, handlers: {},
        on(name, callback) { (this.handlers[name] ??= []).push(callback); return this; },
        off(name, callback) { this.handlers[name] = (this.handlers[name] || []).filter(item => item !== callback); return this; },
        emit(name, event) { (this.handlers[name] || []).slice().forEach(callback => callback(event)); },
        setView(point, zoom) { this.views.push([point, zoom]); return this; },
        invalidateSize() {},
        addLayer(layer) { this.layers.push(layer); return this; },
        removeLayer(layer) {
          if (!this.layers.includes(layer)) return;
          this.layers = this.layers.filter(item => item !== layer);
          (layer.handlers?.remove || []).forEach(callback => callback());
        },
        fitBounds(bounds, options) { this.fit = [bounds, options]; },
        flyTo(point, zoom) { this.fly = [point, zoom]; },
        getZoom() { return this.zoom || 12; },
        remove() { this.removed = true; }
      };
      maps.push(map);
      return map;
    },
    control,
    divIcon: options => options,
    marker(point, options) {
      return {
        kind: 'marker', point, options, element: options.icon.html.children[0],
        addTo(map) { map.layers.push(this); return this; }
      };
    },
    popup(options) {
      return {
        kind: 'popup', options, handlers: {},
        setLatLng(latlng) { this.latlng = latlng; return this; },
        setContent(content) { this.content = content; return this; },
        on(name, callback) { (this.handlers[name] ??= []).push(callback); return this; },
        openOn(map) { map.layers.push(this); return this; }
      };
    },
    polyline(points, options) {
      return { kind: 'line', points, options, addTo(map) { map.layers.push(this); return this; } };
    },
    DomEvent: { disableClickPropagation() {} },
    tileLayer(url, options) { return {
      url, options,
      on(name, callback) { this.listeners ??= new Map(); this.listeners.set(name, callback); return this; },
      emit(name) { this.listeners?.get(name)?.(); },
      addTo(map) { map.tiles.push(this); return this; }
    }; }
  };
}

function element(tagName) {
  const listeners = new Map();
  return {
    tagName: tagName.toUpperCase(),
    children: [],
    style: {},
    dataset: {},
    setAttribute(name, value) { this[name] = value; },
    addEventListener(name, callback) { listeners.set(name, callback); },
    click() { if (!this.disabled) listeners.get('click')(); },
    dispatch(name) { listeners.get(name)?.(); },
    appendChild(child) { this.children.push(child); },
    querySelector() { return null; },
    replaceChildren() { this.children = []; },
    replaceWith(node) { this.replacement = node; },
    focus() { this.focused = true; },
    remove() { this.removed = true; }
  };
}
