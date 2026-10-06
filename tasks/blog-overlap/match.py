import json, glob, re, html, unicodedata, collections, random, sys
fold = lambda s: unicodedata.normalize("NFKD", s).encode("ascii","ignore").decode()
gaz = json.load(open("gazetteer.json"))
DEST = {"lisboa","lisbon","porto","oporto","madeira","funchal","acores","azores","sao miguel","ponta delgada","portugal","sao miguel island","madeira island"}
STOP = {"miradouro","igreja","capela","jardim","praia","museu","castelo","convento","parque","mercado","ribeira","baixa","centro","cathedral",
 "church","garden","beach","museum","viewpoint","market","park","lagoa","ponta","faja","vitoria","estrela","risco","old town","downtown",
 "city centre","city center","historic centre","the cathedral","main square","botanical garden","jardim botanico","cable car","teleferico",
 "santa maria","sao pedro","sao joao","sao jose","santo antonio","nossa senhora","old market","fish market","farmers market","bridge","lighthouse","farol"}
def text(h):
    h = re.sub(r"(?is)<(script|style)[^>]*>.*?</\1>", " ", h)
    return re.sub(r"\s+"," ", fold(html.unescape(re.sub(r"<[^>]+>", " ", h))))
def pats(aliases):
    out = []
    for a in set(aliases):
        f = fold(a).strip()
        if len(f) < 5 or f.lower() in DEST or f.lower() in STOP: continue
        out.append(re.compile(r"\b"+re.escape(f)+r"\b", re.I if " " in f else 0))
    return out
report, samples = {}, []
for dest, places in gaz.items():
    merged = {}  # dedupe by folded name
    for q, p in places.items():
        k = fold(p["name"]).lower(); m = merged.setdefault(k, {"name": p["name"], "kind": p["kind"], "aliases": set()}); m["aliases"] |= set(p["aliases"])
    P = {k: pats(m["aliases"]) for k, m in merged.items()}
    blogs_cov, mentions, nposts = set(), collections.defaultdict(set), 0
    for fn in sorted(glob.glob(f"posts/*__{dest}.json")):
        blog = fn.split("/")[1].split("__")[0]
        for post in json.load(open(fn))["posts"]:
            nposts += 1; blogs_cov.add(blog); t = fold(post["title"]) + " . " + text(post["html"])
            spans = [(m.start(), m.end(), k) for k, ps in P.items() for p in ps for m in p.finditer(t)]
            keep = [s for s in spans if not any(o[2]!=s[2] and o[0]<=s[0] and o[1]>=s[1] and (o[1]-o[0])>(s[1]-s[0]) for o in spans)]
            for s,e,k in keep:
                if blog not in mentions[k]: samples.append((dest, merged[k]["name"], blog, t[max(0,s-70):e+70]))
                mentions[k].add(blog)
    top = sorted(mentions.items(), key=lambda kv: -len(kv[1]))
    c = lambda n: sum(len(b)>=n for b in mentions.values())
    report[dest] = {"blogs": sorted(blogs_cov), "posts": nposts, "mentioned": len(mentions), "ge2": c(2), "ge3": c(3), "ge4": c(4), "single": len(mentions)-c(2),
        "top": [(merged[k]["name"], merged[k]["kind"], len(b), sorted(b)) for k,b in top[:30]]}
json.dump(report, open("report.json","w"), ensure_ascii=False, indent=1)
random.seed(7); json.dump(random.sample(samples, 40), open("sample.json","w"), ensure_ascii=False, indent=1)
for d, r in report.items():
    print(f"\n== {d}: {len(r['blogs'])} blogs, {r['posts']} posts, {r['mentioned']} places | >=2:{r['ge2']} >=3:{r['ge3']} >=4:{r['ge4']} single:{r['single']}")
    for n,k,cnt,b in r["top"]: print(f"  {cnt:2}  {n}  [{k}]")
