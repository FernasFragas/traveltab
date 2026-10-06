import json, urllib.request, urllib.parse, time, sys
EPS = ["https://overpass-api.de/api/interpreter","https://overpass.private.coffee/api/interpreter","https://overpass.kumi.systems/api/interpreter"]
UA = "TravelTab-research/0.1 (blog overlap check)"
BBOX = {"Lisbon":"38.69,-9.24,38.80,-9.08","Porto":"41.10,-8.70,41.19,-8.55",
        "Madeira":"32.62,-17.28,32.88,-16.65","Azores":"37.69,-25.88,37.92,-25.12"}
SEL = ['["tourism"~"attraction|museum|viewpoint|gallery|zoo|aquarium|theme_park"]',
 '["historic"~"castle|monument|fort|palace|church|monastery|tower|city_gate|ruins|archaeological_site"]',
 '["natural"~"beach|peak|volcano|crater|cave_entrance|waterfall|cliff|bay|cape"]',
 '["leisure"~"park|garden|nature_reserve"]','["water"="lake"]','["amenity"~"marketplace|place_of_worship"]',
 '["place"~"neighbourhood|quarter|suburb|village|town"]','["waterway"="waterfall"]','["man_made"~"bridge|lighthouse"]','["bridge"="yes"]["wikidata"]["man_made"="bridge"]']
out = json.load(open("gazetteer.json")) if len(sys.argv)>1 else {}
for dest, bb in BBOX.items():
    if dest in out: continue
    body = "".join(f'nwr["wikidata"]["name"]{s}({bb});' for s in SEL)
    q = f"[out:json][timeout:90];({body});out tags;"
    for ep in EPS:
        try:
            req = urllib.request.Request(ep, data=urllib.parse.urlencode({"data": q}).encode(), headers={"User-Agent": UA})
            data = json.load(urllib.request.urlopen(req, timeout=120)); break
        except Exception as ex: print(dest, ep, ex, file=sys.stderr); time.sleep(5)
    else: continue
    places = {}
    for e in data["elements"]:
        t = e["tags"]; p = places.setdefault(t["wikidata"], {"name": t["name"], "aliases": set(), "kind": ""})
        p["kind"] = p["kind"] or next((f"{k}={t[k]}" for k in ("tourism","historic","natural","leisure","amenity","place","man_made","waterway","water") if k in t), "")
        for k in ("name","name:en","name:pt","alt_name","short_name","official_name"):
            for v in t.get(k,"").split(";"):
                if v.strip(): p["aliases"].add(v.strip())
    out[dest] = {q: dict(p, aliases=sorted(p["aliases"])) for q,p in places.items()}
    print(dest, len(places), file=sys.stderr)
    json.dump(out, open("gazetteer.json","w"), ensure_ascii=False, indent=1); time.sleep(3)
