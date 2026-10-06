"""Extract per-place claims from deep/*.json posts with local Ollama. Appends to claims.jsonl (resumable).
Every claim must carry a verbatim quote; quote_ok records whether it is really in the post."""
import json, glob, re, html, unicodedata, urllib.request, sys, time, os
MODEL = os.environ.get("MODEL", "qwen3.6:35b")
fold = lambda s: unicodedata.normalize("NFKD", s).encode("ascii","ignore").decode()
norm = lambda s: re.sub(r"\s+", " ", fold(s).lower().replace("’","'").replace("“",'"').replace("”",'"')).strip(" .,\"'")
STOP = {"miradouro","igreja","capela","jardim","praia","museu","castelo","convento","parque","mercado","ribeira","centro","cathedral","church",
 "garden","beach","museum","viewpoint","market","park","lagoa","ponta","faja","risco","old town","downtown","city centre","city center",
 "botanical garden","jardim botanico","cable car","teleferico","santa maria","sao pedro","sao joao","sao jose","santo antonio","nossa senhora",
 "bridge","lighthouse","farol","madeira","funchal","portugal","madeira island"}
def plain(h):
    h = re.sub(r"(?is)<(script|style)[^>]*>.*?</\1>", " ", h)
    return re.sub(r"\s+", " ", html.unescape(re.sub(r"<[^>]+>", " ", h))).strip()
def places():
    merged = {}
    for p in json.load(open("gazetteer.json"))["Madeira"].values():
        m = merged.setdefault(p["name"], set()); m |= set(p["aliases"])
    out = {}
    for name, al in merged.items():
        ps = []
        for a in al:
            f = fold(a).strip()
            if len(f) < 5 or f.lower() in STOP: continue
            ps.append(re.compile(r"\b"+re.escape(f)+r"\b", re.I if " " in f else 0))
        if ps: out[name] = ps
    return out
P = places()
def mentions(text):
    ft = fold(text)
    spans = [(m.start(), m.end(), n) for n, ps in P.items() for p in ps for m in p.finditer(ft)]
    return [s for s in spans if not any(o[2]!=s[2] and o[0]<=s[0] and o[1]>=s[1] and o[1]-o[0]>s[1]-s[0] for o in spans)]
def passages(text, spans, width=500, cap=14000):
    wins = sorted((max(0,s-width), min(len(text),e+width)) for s,e,_ in spans)
    merged = []
    for a,b in wins:
        if merged and a <= merged[-1][1]: merged[-1][1] = max(merged[-1][1], b)
        else: merged.append([a,b])
    out = "\n[...]\n".join(text[a:b] for a,b in merged)
    return out[:cap]
SCHEMA = {"type":"object","properties":{"claims":{"type":"array","items":{"type":"object","properties":{
  "place":{"type":"string"},
  "type":{"type":"string","enum":["verdict","timing","cost","access","duration","warning","tip"]},
  "stance":{"type":"string","enum":["go","skip","mixed","neutral"]},
  "claim_en":{"type":"string"},
  "quote":{"type":"string"}},"required":["place","type","stance","claim_en","quote"]}}},"required":["claims"]}
PROMPT = """You extract travel advice from a blog post for a map of Madeira.

Places detected in this post: {places}

For each of those places, list the advice or opinions the AUTHOR gives about it:
- verdict: whether it is worth visiting, what they loved or disliked
- timing: best time of day/season, crowds
- cost: prices, fees, free entry
- access: how to get there, parking, booking, transport
- duration: how long to spend or how long a hike takes
- warning: difficulty, safety, closures, tourist traps
- tip: any other practical recommendation

Rules:
- Only claims the author actually makes in the text below. No outside knowledge.
- Skip pure history, hotel/restaurant addresses, and places only listed by name.
- "place" must be exactly one of the detected place names.
- "claim_en": one short English sentence (translate if the post is Portuguese).
- "quote": copy the exact sentence or phrase from the text that supports the claim, in its original language, without changes. Max 250 characters.
- If the author gives no advice about a place, return nothing for it. An empty list is fine.

Reply with ONLY the JSON object, no commentary, no markdown fences. "type" must be exactly one of
verdict|timing|cost|access|duration|warning|tip and "stance" exactly one of go|skip|mixed|neutral.

Post title: {title}

Text:
{text}"""
# Cloud models ignore the `format` schema: they wrap the JSON in prose and invent enum values.
# Parse tolerantly and coerce type/stance back onto the schema's enums.
TYPES = {"verdict","timing","cost","access","duration","warning","tip"}
STANCES = {"go","skip","mixed","neutral"}
TYPE_MAP = {"opinion":"verdict","review":"verdict","recommendation":"verdict","advice":"tip","recommend":"tip",
 "safety":"warning","caution":"warning","difficulty":"warning","price":"cost","fee":"cost","money":"cost",
 "booking":"access","parking":"access","transport":"access","getting there":"access","time":"duration","length":"duration"}
STANCE_MAP = {"positive":"go","recommend":"go","must go":"go","negative":"skip","avoid":"skip","dont go":"skip",
 "info":"neutral","informational":"neutral","fact":"neutral","neutral":'neutral'}
def coerce(c):
    t = str(c.get("type","")).strip().lower(); s = str(c.get("stance","")).strip().lower()
    c["type"] = t if t in TYPES else TYPE_MAP.get(t, "tip")
    c["stance"] = s if s in STANCES else STANCE_MAP.get(s, "neutral")
    c["place"] = str(c.get("place","")).strip()
    c["claim_en"] = str(c.get("claim_en","")).strip()
    c["quote"] = str(c.get("quote","")).strip()
    return c
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
    out["claims"] = [coerce(c) for c in out.get("claims", []) if isinstance(c, dict)]
    return out, r.get("prompt_eval_count",0), r.get("eval_count",0)
def main(limit=None):
    done = set()
    if os.path.exists("claims_done.txt"): done = set(open("claims_done.txt").read().split())
    n = 0
    for fn in sorted(glob.glob("deep/*.json")):
        blog = fn.split("/")[-1][:-5]
        for post in json.load(open(fn))["posts"]:
            key = f"{blog}:{post['id']}"
            if key in done: continue
            text = plain(post["html"]); spans = mentions(text)
            names = sorted({n_ for _,_,n_ in spans})
            if names:
                t0 = time.time()
                try: out, pin, pout = ask(PROMPT.format(places=", ".join(names), title=post["title"], text=passages(text, spans)))
                except Exception as ex: print("ERR", key, ex, flush=True); continue
                nt = norm(text) + " " + norm(post["title"])
                with open("claims.jsonl","a") as f:
                    for c in out.get("claims", []):
                        c.update(blog=blog, post=post["link"], place_ok=c["place"] in names, quote_ok=len(norm(c["quote"]))>=15 and norm(c["quote"]) in nt)
                        f.write(json.dumps(c, ensure_ascii=False)+"\n")
                print(f"{key:40} places={len(names):2} claims={len(out.get('claims',[])):2} in={pin} out={pout} {time.time()-t0:.0f}s", flush=True)
            open("claims_done.txt","a").write(key+"\n"); n += 1
            if limit and n >= limit: return
if __name__ == "__main__":
    main(int(sys.argv[1]) if len(sys.argv) > 1 else None)
