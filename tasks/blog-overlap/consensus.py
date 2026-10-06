"""Group verified claims (claims.jsonl, quote_ok and place_ok) per place into statements backed by >=2 blogs.
The model proposes groups; this script keeps a statement only if its claim ids span >=2 distinct blogs.
Writes consensus.json and prints a summary."""
import json, collections, urllib.request, os
MODEL = os.environ.get("MODEL", "qwen3.6:35b")
SCHEMA = {"type":"object","properties":{"statements":{"type":"array","items":{"type":"object","properties":{
  "text_en":{"type":"string"},"claim_ids":{"type":"array","items":{"type":"integer"}}},"required":["text_en","claim_ids"]}},
  "disagreements":{"type":"array","items":{"type":"object","properties":{
  "text_en":{"type":"string"},"claim_ids":{"type":"array","items":{"type":"integer"}}},"required":["text_en","claim_ids"]}}},
  "required":["statements","disagreements"]}
PROMPT = """These are claims different travel blogs make about {place} (Madeira). Each has an id and a blog.

{claims}

1. "statements": group claims that say the SAME thing (same advice, same verdict, compatible numbers). Write one short English sentence per group and list the claim ids. Only include groups whose claims come from at least two different blogs.
2. "disagreements": groups where blogs clearly contradict each other (e.g. one says go early, another says afternoon is fine). List the ids.
Do not add information that is not in the claims. Empty lists are fine.

Reply with ONLY the JSON object {"statements":[...],"disagreements":[...]}, no commentary, no markdown fences."""
def parse(content):
    """Cloud models prepend reasoning; grab the outermost JSON object."""
    try: return json.loads(content)
    except Exception: pass
    start = content.find("{")
    while start != -1:
        depth = 0
        for i in range(start, len(content)):
            if content[i] == "{": depth += 1
            elif content[i] == "}":
                depth -= 1
                if depth == 0:
                    try: return json.loads(content[start:i+1])
                    except Exception: break
        start = content.find("{", start+1)
    raise ValueError("no JSON object in model output")
def ask(prompt):
    body = json.dumps({"model": MODEL, "stream": False, "think": False, "format": SCHEMA,
        "options": {"temperature": 0, "num_ctx": 16384}, "messages": [{"role":"user","content":prompt}]}).encode()
    r = json.load(urllib.request.urlopen(urllib.request.Request("http://localhost:11434/api/chat", data=body, headers={"Content-Type":"application/json"}), timeout=900))
    out = parse(r["message"]["content"])
    for kind in ("statements", "disagreements"):
        out[kind] = [s for s in out.get(kind, []) if isinstance(s, dict) and isinstance(s.get("claim_ids"), list)]
    return out
claims = [json.loads(l) for l in open("claims.jsonl")]
good = [c for c in claims if c["quote_ok"] and c["place_ok"]]
by_place = collections.defaultdict(list)
for c in good: by_place[c["place"]].append(c)
result = {}
for place, cs in sorted(by_place.items(), key=lambda kv: -len({c["blog"] for c in kv[1]})):
    blogs = {c["blog"] for c in cs}
    entry = {"blogs": sorted(blogs), "claims": len(cs), "statements": [], "disagreements": []}
    if len(blogs) >= 2:
        listing = "\n".join(f'{i}. [{c["blog"]}] ({c["type"]}/{c["stance"]}) {c["claim_en"]}' for i, c in enumerate(cs))
        out = ask(PROMPT.format(place=place, claims=listing))
        for kind in ("statements", "disagreements"):
            for s in out.get(kind, []):
                ids = [i for i in s["claim_ids"] if 0 <= i < len(cs)]
                sb = sorted({cs[i]["blog"] for i in ids})
                if len(sb) >= 2:
                    entry[kind].append({"text_en": s["text_en"], "blogs": sb,
                        "sources": [{"blog": cs[i]["blog"], "post": cs[i]["post"], "quote": cs[i]["quote"], "claim_en": cs[i]["claim_en"]} for i in ids]})
        print(f"{place:40} blogs={len(blogs):2} claims={len(cs):3} statements={len(entry['statements'])} disagreements={len(entry['disagreements'])}", flush=True)
    result[place] = entry
json.dump(result, open("consensus.json","w"), ensure_ascii=False, indent=1)
n = lambda f: sum(1 for e in result.values() if f(e))
print(f"\nclaims: {len(claims)} total, {len(good)} verified ({len(good)/max(1,len(claims)):.0%})")
print(f"places with >=1 verified claim: {len(result)} | from >=2 blogs: {n(lambda e: len(e['blogs'])>=2)} | from >=3 blogs: {n(lambda e: len(e['blogs'])>=3)}")
print(f"places with >=1 agreed statement (>=2 blogs): {n(lambda e: e['statements'])} | with >=1 statement backed by >=3 blogs: {n(lambda e: any(len(s['blogs'])>=3 for s in e['statements']))}")
print(f"places with a disagreement: {n(lambda e: e['disagreements'])} | single-blog places with a usable tip: {n(lambda e: len(e['blogs'])==1)}")
