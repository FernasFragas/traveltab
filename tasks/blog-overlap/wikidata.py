"""Add Wikidata en/pt labels and aliases to gazetteer.json, then apply the manual alias fixes found in the precision check."""
import json, urllib.request, urllib.parse, time, urllib.error
UA = "TravelTabResearch/0.1 (https://github.com/FernasFragas/traveltab; one-off overlap check) python-urllib"
gaz = json.load(open("gazetteer.json"))
qids = sorted({q for d in gaz.values() for q in d})
wd = {}
for i in range(0, len(qids), 50):
    url = "https://www.wikidata.org/w/api.php?" + urllib.parse.urlencode({"action":"wbgetentities","ids":"|".join(qids[i:i+50]),"props":"labels|aliases","languages":"en|pt","format":"json"})
    for attempt in range(6):
        try:
            r = json.load(urllib.request.urlopen(urllib.request.Request(url, headers={"User-Agent":UA}), timeout=60))
            if "entities" in r: break
            time.sleep(10*(attempt+1))
        except urllib.error.HTTPError as ex:
            time.sleep(int(ex.headers.get("Retry-After") or 10*(attempt+1)))
    else: raise SystemExit("gave up at batch %d" % i)
    for q, e in r["entities"].items():
        wd[q] = sorted({v["value"] for v in e.get("labels",{}).values()} | {a["value"] for l in e.get("aliases",{}).values() for a in l})
    time.sleep(2)
for d in gaz.values():
    for q, v in d.items():
        v["aliases"] = sorted(set(v["aliases"]) | set(wd.get(q, [])))
        # Manual fixes from the 40-match precision sample (see ../blog-overlap-results.md).
        if v["name"] == "Aeroporto de Santana": v["aliases"] = [a for a in v["aliases"] if a != "Ribeira Grande"]
        if v["name"] == "Nossa Senhora da Conceição": v["aliases"] = []
        if v["name"] in ("Igreja dos Clérigos", "Torre dos Clérigos"): v["name"] = "Igreja e Torre dos Clérigos"
json.dump(gaz, open("gazetteer.json","w"), ensure_ascii=False, indent=1)
