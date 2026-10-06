"""Fetch destination posts from WordPress travel blogs (identified UA, polite delay). Writes posts/<blog>__<dest>.json."""
import json, urllib.request, urllib.parse, time, sys, os, unicodedata, html, re
UA = "TravelTab-research/0.1 (blog overlap check)"
BLOGS = {"almadeviajante.com":"pt","www.viajecomigo.com":"pt","www.vortexmag.net":"pt","viagensasolta.com":"pt",
 "www.portugalist.com":"en","www.nomadicmatt.com":"en","theplanetd.com":"en","www.earthtrekkers.com":"en",
 "www.saltinourhair.com":"en","travellemming.com":"en","www.thecrowdedplanet.com":"en","www.adventurouskate.com":"en",
 "www.nomadicsamuel.com":"en","travelingwithjosh.com":"en"}
TERMS = {"Lisbon":["Lisbon","Lisboa"],"Porto":["Porto"],"Madeira":["Madeira","Funchal"],"Azores":["Azores","Açores","Sao Miguel","São Miguel"]}
fold = lambda s: unicodedata.normalize("NFKD", s).encode("ascii","ignore").decode().lower()
def get(url):
    req = urllib.request.Request(url, headers={"User-Agent": UA})
    return json.load(urllib.request.urlopen(req, timeout=40))
def main():
  for blog in BLOGS:
    for dest, terms in TERMS.items():
        fn = f"posts/{blog}__{dest}.json"
        if os.path.exists(fn): continue
        seen, keep, err = set(), [], None
        for term in terms:
            rows = []
            for page in (1, 2, 3):  # relevance search ranks body mentions too; dig until 10 title matches
                try: batch = get(f"https://{blog}/wp-json/wp/v2/posts?search={urllib.parse.quote(term)}&per_page=50&page={page}&_fields=id,link,title,content")
                except Exception as ex: err = str(ex); break
                rows += batch; time.sleep(1.5)
                if len(batch) < 50 or len(rows) >= 150: break
            for r in rows:
                title = html.unescape(re.sub("<[^>]+>","",r["title"]["rendered"]))
                if r["id"] in seen or not any(fold(t) in fold(title) for t in terms): continue
                if dest=="Porto" and ("porto santo" in fold(title) or "porto moniz" in fold(title)): continue
                seen.add(r["id"]); keep.append({"id":r["id"],"link":r["link"],"title":title,"html":r["content"]["rendered"]})
            time.sleep(1.5)
        keep = keep[:10]
        json.dump({"error":err,"posts":keep}, open(fn,"w"), ensure_ascii=False)
        print(f"{blog:28} {dest:8} {len(keep):3} {err or ''}", flush=True)

if __name__ == "__main__":
    main()
