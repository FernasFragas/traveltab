"""Fetch every Madeira-related post per blog (no 10-post cap). Writes deep/<blog>.json.
Keeps a post when its title names Madeira/Funchal or a multi-word Madeira gazetteer place,
or its body says "Madeira" (capitalized) at least 3 times."""
import json, urllib.request, urllib.parse, time, os, re, html, unicodedata, sys
UA = "TravelTab-research/0.1 (blog overlap check)"
BLOGS = ["almadeviajante.com","www.viajecomigo.com","www.vortexmag.net","viagensasolta.com","www.portugalist.com","www.nomadicmatt.com",
         "theplanetd.com","www.earthtrekkers.com","www.saltinourhair.com","travellemming.com","www.thecrowdedplanet.com","www.adventurouskate.com"]
TERMS = ["Madeira", "Funchal", "levada"]
fold = lambda s: unicodedata.normalize("NFKD", s).encode("ascii","ignore").decode().lower()
gaz = json.load(open("gazetteer.json"))["Madeira"]
PLACE_TITLES = {fold(a) for p in gaz.values() for a in p["aliases"] if " " in a and len(a) >= 8}
def get(url):
    for attempt in range(3):
        try: return json.load(urllib.request.urlopen(urllib.request.Request(url, headers={"User-Agent": UA}), timeout=60))
        except urllib.error.HTTPError as ex:
            if ex.code == 400: return []   # past last page
            if attempt == 2: raise
        except Exception:
            if attempt == 2: raise
        time.sleep(5)
os.makedirs("deep", exist_ok=True)
for blog in BLOGS:
    fn = f"deep/{blog}.json"
    if os.path.exists(fn): continue
    seen, keep, scanned, err = set(), [], 0, None
    for term in TERMS:
        for page in range(1, 30):
            try: rows = get(f"https://{blog}/wp-json/wp/v2/posts?search={urllib.parse.quote(term)}&per_page=100&page={page}&_fields=id,link,title,content")
            except Exception as ex: err = str(ex); break
            for r in rows:
                if r["id"] in seen: continue
                seen.add(r["id"]); scanned += 1
                title = html.unescape(re.sub("<[^>]+>", "", r["title"]["rendered"])); ft = fold(title)
                body = r["content"]["rendered"]
                if "porto santo" in ft and "madeira" not in ft: continue
                if ("madeira" in ft or "funchal" in ft or any(p in ft for p in PLACE_TITLES)
                        or len(re.findall(r"\bMadeira\b", body)) >= 3):
                    keep.append({"id": r["id"], "link": r["link"], "title": title, "html": body})
            time.sleep(1.5)
            if len(rows) < 100: break
    json.dump({"error": err, "scanned": scanned, "posts": keep}, open(fn, "w"), ensure_ascii=False)
    print(f"{blog:28} scanned {scanned:4}  kept {len(keep):3} {err or ''}", flush=True)
