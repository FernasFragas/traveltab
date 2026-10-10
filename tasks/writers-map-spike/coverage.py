"""Writers-map spike, part 2: Madeira coverage and precision (tasks/todo.md 2a).

1. Rebuilds the October 5 deep-run place list: OSM gazetteer + Wikidata aliases (as in
   ../blog-overlap/gaz.py and wikidata.py), Madeira posts from the 12 blogs (as in
   fetch_deep.py) and the same matcher (match.py). Post bodies stay in memory.
2. Checks which places mentioned by >=3 blogs the Wikidata nearby query keeps after the
   planner's type filter (internal/planner/placetypes_gen.go) and the ~200 sitelink cap.
3. Draws 40 random first mentions for a hand precision check.

Committed output (names, QIDs, blog hosts, post IDs only): madeira-coverage.json.
Context snippets for the hand check go to the cache directory only (argv[1]); delete it
after the review.
"""
import html
import json
import random
import re
import sys
import time
import unicodedata
import urllib.error
import urllib.parse
import urllib.request
import urllib.robotparser
from collections import defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parent
REPO = ROOT.parent.parent
OUT = ROOT / "madeira-coverage.json"
UA = "TravelTab-research/0.1 (blog overlap check)"
LIMITER_SECONDS = 1.5
BLOGS = ["almadeviajante.com", "www.viajecomigo.com", "www.vortexmag.net", "viagensasolta.com", "www.portugalist.com",
         "www.nomadicmatt.com", "theplanetd.com", "www.earthtrekkers.com", "www.saltinourhair.com", "travellemming.com",
         "www.thecrowdedplanet.com", "www.adventurouskate.com"]
TERMS = ["Madeira", "Funchal", "levada"]
BBOX = "32.62,-17.28,32.88,-16.65"
OVERPASS = ["https://overpass-api.de/api/interpreter", "https://overpass.private.coffee/api/interpreter",
            "https://overpass.kumi.systems/api/interpreter"]
SEL = ['["tourism"~"attraction|museum|viewpoint|gallery|zoo|aquarium|theme_park"]',
       '["historic"~"castle|monument|fort|palace|church|monastery|tower|city_gate|ruins|archaeological_site"]',
       '["natural"~"beach|peak|volcano|crater|cave_entrance|waterfall|cliff|bay|cape"]',
       '["leisure"~"park|garden|nature_reserve"]', '["water"="lake"]', '["amenity"~"marketplace|place_of_worship"]',
       '["place"~"neighbourhood|quarter|suburb|village|town"]', '["waterway"="waterfall"]',
       '["man_made"~"bridge|lighthouse"]', '["bridge"="yes"]["wikidata"]["man_made"="bridge"]']
DEST = {"lisboa", "lisbon", "porto", "oporto", "madeira", "funchal", "acores", "azores", "sao miguel", "ponta delgada",
        "portugal", "sao miguel island", "madeira island"}
CAP = 200


def main():
    cache = Path(sys.argv[1])
    cache.mkdir(parents=True, exist_ok=True)
    overrides = json.loads((REPO / "writerdata/overrides.json").read_text())
    area = next(a for a in overrides["areas"] if a["name"] == "Madeira")
    places = gazetteer(cache)
    mentions, samples, hosts = crawl_and_match(places, overrides["stoplist"])
    consensus = [k for k, b in mentions.items() if len(b) >= 3]
    nearby = nearby_places(cache, area)
    kept = type_filter(nearby, load_place_types())
    capped = set(sorted(kept, key=lambda q: (-nearby[q]["sitelinks"], q))[:CAP])
    rows = []
    for k in sorted(consensus, key=lambda k: (-len(mentions[k]), places[k]["name"])):
        qids = sorted(places[k]["qids"])
        rows.append({"name": places[k]["name"], "osm_kind": places[k]["kind"], "qids": qids, "blogs": len(mentions[k]),
                     "in_radius": any(q in nearby for q in qids), "type_kept": any(q in kept for q in qids),
                     "in_cap": any(q in capped for q in qids),
                     "types": sorted({t for q in qids if q in nearby for t in nearby[q]["types"]})})
    random.seed(7)
    sample = random.sample(samples, min(40, len(samples)))
    (cache / "precision-sample.json").write_text(json.dumps(sample, ensure_ascii=False, indent=1))
    c = lambda n: sum(len(b) >= n for b in mentions.values())
    summary = {"run": time.strftime("%Y-%m-%d"), "hosts": hosts, "posts": sum(h["kept"] for h in hosts.values()),
               "gazetteer_places": len(places), "mentioned": len(mentions), "ge3": c(3), "ge4": c(4),
               "single": len(mentions) - c(2), "nearby_radius_km": area["radius_km"], "nearby_places": len(nearby),
               "nearby_type_kept": len(kept), "nearby_cap": CAP,
               "consensus_in_radius": sum(r["in_radius"] for r in rows),
               "consensus_type_kept": sum(r["type_kept"] for r in rows),
               "consensus_in_cap": sum(r["in_cap"] for r in rows),
               "precision_sample": [{k: s[k] for k in ("name", "blog", "post_id", "matched")} for s in sample],
               "consensus": rows}
    OUT.write_text(json.dumps(summary, ensure_ascii=False, indent=1))
    print(json.dumps({k: v for k, v in summary.items() if k not in ("consensus", "precision_sample")}, indent=1))


def gazetteer(cache):
    """OSM places with a wikidata tag in the Madeira box, plus Wikidata en/pt labels and aliases,
    merged by folded name as match.py does. Returns {folded name: {name, kind, qids, aliases}}."""
    path = cache / "gazetteer.json"
    if path.exists():
        raw = json.loads(path.read_text())
    else:
        raw = overpass_places()
        add_wikidata_aliases(raw)
        path.write_text(json.dumps(raw, ensure_ascii=False))
    merged = {}
    for qid, p in raw.items():
        m = merged.setdefault(fold(p["name"]).lower(), {"name": p["name"], "kind": p["kind"], "qids": set(), "aliases": set()})
        m["qids"].add(qid)
        m["aliases"] |= set(p["aliases"])
    return merged


def overpass_places():
    body = "".join(f'nwr["wikidata"]["name"]{s}({BBOX});' for s in SEL)
    query = f"[out:json][timeout:90];({body});out tags;"
    for attempt in range(3):
        for endpoint in OVERPASS:
            try:
                data = json.load(request(endpoint, urllib.parse.urlencode({"data": query}).encode(), timeout=120))
                break
            except Exception as exc:
                print(f"overpass {endpoint}: {exc}", file=sys.stderr)
                time.sleep(5)
        else:
            continue
        break
    else:
        raise SystemExit("Overpass unavailable")
    places = {}
    for element in data["elements"]:
        tags = element["tags"]
        p = places.setdefault(tags["wikidata"], {"name": tags["name"], "aliases": set(), "kind": ""})
        p["kind"] = p["kind"] or next((f"{k}={tags[k]}" for k in ("tourism", "historic", "natural", "leisure", "amenity",
                                                                      "place", "man_made", "waterway", "water") if k in tags), "")
        for key in ("name", "name:en", "name:pt", "alt_name", "short_name", "official_name"):
            p["aliases"] |= {v.strip() for v in tags.get(key, "").split(";") if v.strip()}
    print(f"gazetteer: {len(places)} places", file=sys.stderr)
    return {q: dict(p, aliases=sorted(p["aliases"])) for q, p in places.items()}


def add_wikidata_aliases(places):
    """Same as ../blog-overlap/wikidata.py, including the precision-check fixes (now in overrides.json)."""
    ids = sorted(places)
    for start in range(0, len(ids), 50):
        params = urllib.parse.urlencode({"action": "wbgetentities", "ids": "|".join(ids[start:start + 50]),
                                         "props": "labels|aliases", "languages": "en|pt", "format": "json"})
        entities = json.load(request("https://www.wikidata.org/w/api.php?" + params, timeout=60))["entities"]
        for qid, entity in entities.items():
            if qid not in places:
                continue
            extra = {v["value"] for v in entity.get("labels", {}).values()}
            extra |= {a["value"] for values in entity.get("aliases", {}).values() for a in values}
            places[qid]["aliases"] = sorted(set(places[qid]["aliases"]) | extra)
        time.sleep(2)
    for p in places.values():
        if p["name"] == "Aeroporto de Santana":
            p["aliases"] = [a for a in p["aliases"] if a != "Ribeira Grande"]
        if p["name"] == "Nossa Senhora da Conceição":
            p["aliases"] = []


def crawl_and_match(places, stoplist):
    """Fetches Madeira posts host by host (fetch_deep.py rules) and matches them in memory
    (match.py rules). Returns {place key: set of hosts}, first-mention samples and host stats."""
    stop = {fold(s).lower() for s in stoplist}
    patterns = {k: compile_patterns(p["aliases"], stop) for k, p in places.items()}
    place_titles = {fold(a).lower() for p in places.values() for a in p["aliases"] if " " in a and len(a) >= 8}
    mentions, samples, hosts = defaultdict(set), [], {}
    for host in BLOGS:
        stats = {"scanned": 0, "kept": 0, "status": "ok"}
        hosts[host] = stats
        try:
            if not allowed_by_robots(host, "/wp-json/wp/v2/posts"):
                stats["status"] = "robots.txt disallows"
                print(f"{host}: robots.txt disallows the posts API; skipping", flush=True)
                continue
        except Exception as exc:
            stats["status"] = f"robots.txt: {exc}"
            continue
        seen = set()
        try:
            for term in TERMS:
                for page in range(1, 30):
                    time.sleep(LIMITER_SECONDS)
                    rows = get_posts(host, term, page)
                    for post in rows:
                        if post["id"] in seen:
                            continue
                        seen.add(post["id"])
                        stats["scanned"] += 1
                        title = html.unescape(re.sub("<[^>]+>", "", post["title"]["rendered"]))
                        if not keep_post(title, post["content"]["rendered"], place_titles):
                            continue
                        stats["kept"] += 1
                        match_post(host, post["id"], title, post["content"]["rendered"], patterns, places, mentions, samples)
                    if len(rows) < 100:
                        break
        except Exception as exc:
            stats["status"] = f"partial: {exc}"
        print(f"{host:28} scanned {stats['scanned']:4} kept {stats['kept']:3} {stats['status']}", flush=True)
    return mentions, samples, hosts


def keep_post(title, body, place_titles):
    ft = fold(title).lower()
    if "porto santo" in ft and "madeira" not in ft:
        return False
    return ("madeira" in ft or "funchal" in ft or any(p in ft for p in place_titles)
            or len(re.findall(r"\bMadeira\b", body)) >= 3)


def match_post(host, post_id, title, body, patterns, places, mentions, samples):
    text = fold(title) + " . " + plain_text(body)
    spans = [(m.start(), m.end(), k) for k, ps in patterns.items() for p in ps for m in p.finditer(text)]
    kept = [s for s in spans if not any(o[2] != s[2] and o[0] <= s[0] and o[1] >= s[1] and (o[1] - o[0]) > (s[1] - s[0])
                                        for o in spans)]
    for start, end, k in kept:
        if host not in mentions[k]:
            samples.append({"name": places[k]["name"], "blog": host, "post_id": post_id, "matched": text[start:end],
                            "context": text[max(0, start - 70):end + 70]})
        mentions[k].add(host)


def compile_patterns(aliases, stop):
    out = []
    for alias in set(aliases):
        f = fold(alias).strip()
        if len(f) < 5 or f.lower() in DEST or f.lower() in stop:
            continue
        out.append(re.compile(r"\b" + re.escape(f) + r"\b", re.I if " " in f else 0))
    return out


def nearby_places(cache, area):
    """Every Wikidata item within the area radius, with its P31 types and sitelink count."""
    path = cache / "nearby.json"
    if path.exists():
        return json.loads(path.read_text())
    query = f'''SELECT ?place ?type ?sitelinks WHERE {{
      SERVICE wikibase:around {{
        ?place wdt:P625 ?coord .
        bd:serviceParam wikibase:center "Point({area["lon"]} {area["lat"]})"^^geo:wktLiteral;
          wikibase:radius "{area["radius_km"]}" .
      }}
      OPTIONAL {{ ?place wdt:P31 ?type . }}
      OPTIONAL {{ ?place wikibase:sitelinks ?sitelinks . }}
    }}'''
    url = "https://query.wikidata.org/sparql?" + urllib.parse.urlencode({"query": query, "format": "json"})
    rows = json.load(request(url, timeout=120))["results"]["bindings"]
    places = {}
    for row in rows:
        qid = row["place"]["value"].rsplit("/", 1)[-1]
        p = places.setdefault(qid, {"types": [], "sitelinks": int(row.get("sitelinks", {}).get("value", 0))})
        if "type" in row:
            t = row["type"]["value"].rsplit("/", 1)[-1]
            if t not in p["types"]:
                p["types"].append(t)
    path.write_text(json.dumps(places))
    print(f"nearby: {len(places)} items", file=sys.stderr)
    return places


def load_place_types():
    src = (REPO / "internal/planner/placetypes_gen.go").read_text()
    return dict(re.findall(r'"(Q\d+)":\s*(\w+),', src))


def type_filter(nearby, place_types):
    """planner.kindOf: drop on any Deny type, keep when at least one type is Indoor/Outdoor/Mixed."""
    kept = set()
    for qid, p in nearby.items():
        kinds = [place_types.get(t) for t in p["types"]]
        if "Deny" not in kinds and any(k in ("Indoor", "Outdoor", "Mixed") for k in kinds):
            kept.add(qid)
    return kept


def get_posts(host, term, page):
    url = (f"https://{host}/wp-json/wp/v2/posts?search={urllib.parse.quote(term)}"
           f"&per_page=100&page={page}&_fields=id,title,content")
    for attempt in range(3):
        try:
            return json.load(request(url, timeout=60))
        except urllib.error.HTTPError as exc:
            if exc.code == 400:
                return []  # past the last page
            if attempt == 2:
                raise
        except (TimeoutError, urllib.error.URLError):
            if attempt == 2:
                raise
        time.sleep(5)


def request(url, data=None, timeout=90):
    req = urllib.request.Request(url, data=data, headers={"User-Agent": UA})
    try:
        return urllib.request.urlopen(req, timeout=timeout)
    except urllib.error.HTTPError as exc:
        if exc.code in (401, 403, 429):
            raise RuntimeError(f"HTTP {exc.code}; stopping this source without retry")
        raise


def allowed_by_robots(host, path):
    parser = urllib.robotparser.RobotFileParser()
    try:
        parser.parse(request(f"https://{host}/robots.txt").read().decode("utf-8", "replace").splitlines())
    except urllib.error.HTTPError as exc:
        if exc.code == 404:
            return True
        raise
    return parser.can_fetch(UA, f"https://{host}{path}")


def plain_text(markup):
    markup = re.sub(r"(?is)<(script|style)[^>]*>.*?</\1>", " ", markup)
    return re.sub(r"\s+", " ", fold(html.unescape(re.sub(r"<[^>]+>", " ", markup))))


def fold(value):
    return unicodedata.normalize("NFKD", value).encode("ascii", "ignore").decode()


if __name__ == "__main__":
    main()
