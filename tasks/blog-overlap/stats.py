"""Numbers for the write-up, from claims.jsonl + consensus.json. Run in the run dir."""
import json, collections, random, sys
claims = [json.loads(l) for l in open("claims.jsonl")]
good = [c for c in claims if c["quote_ok"] and c["place_ok"]]
print(f"claims total: {len(claims)}")
print(f"verified (quote_ok && place_ok): {len(good)} ({len(good)/max(1,len(claims)):.1%})")
print(f"  quote_ok only: {sum(c['quote_ok'] for c in claims)} | place_ok only: {sum(c['place_ok'] for c in claims)}")
print(f"  quote_ok && !place_ok: {sum(c['quote_ok'] and not c['place_ok'] for c in claims)}")
print(f"  !quote_ok && place_ok: {sum(c['place_ok'] and not c['quote_ok'] for c in claims)}")

print("\nby type:", dict(collections.Counter(c["type"] for c in good).most_common()))
print("by stance:", dict(collections.Counter(c["stance"] for c in good).most_common()))
print("\nby type (all claims, incl. unverified):", dict(collections.Counter(c["type"] for c in claims).most_common()))

# dedupe multi-place sentences: same blog+quote repeated across places
print(f"\nduplicate (blog,quote) pairs among verified: {len(good)-len({(c['blog'],c['quote']) for c in good})}")
print(f"distinct (blog,quote) verified: {len({(c['blog'],c['quote']) for c in good})}")

by_place = collections.defaultdict(list)
for c in good: by_place[c["place"]].append(c)
b = {p: len({c["blog"] for c in cs}) for p, cs in by_place.items()}
print(f"\nplaces with >=1 verified claim: {len(by_place)}")
print(f"  from >=2 blogs: {sum(1 for v in b.values() if v>=2)}")
print(f"  from >=3 blogs: {sum(1 for v in b.values() if v>=3)}")
print(f"  single-blog: {sum(1 for v in b.values() if v==1)}")

try:
    cons = json.load(open("consensus.json"))
    st = sum(1 for e in cons.values() if e["statements"])
    st3 = sum(1 for e in cons.values() if any(len(s["blogs"])>=3 for s in e["statements"]))
    dis = sum(1 for e in cons.values() if e["disagreements"])
    print(f"\nplaces with >=1 agreed statement (>=2 blogs): {st}  <-- HEADLINE")
    print(f"places with >=1 statement backed by >=3 blogs: {st3}")
    print(f"places with a disagreement: {dis}")
    allst = [(e, s) for e in cons.values() for s in e["statements"]]
    print(f"total agreed statements: {len(allst)}")
    allst.sort(key=lambda es: -len(es[1]["blogs"]))
    print("\ntop statements:")
    for e, s in allst[:15]:
        print(f"  [{len(s['blogs'])} blogs] {s['text_en']}")
    # single-blog places with a usable tip
    tips = [p for p, e in cons.items() if len(e["blogs"])==1 and any(c["type"] in ("tip","access","timing","cost","warning") for c in e["claims"] if False)]
    print(f"\nsingle-blog places: {sum(1 for e in cons.values() if len(e['blogs'])==1)}")
except FileNotFoundError:
    print("\n(consensus.json not found yet)")

if len(sys.argv) > 1 and sys.argv[1] == "sample":
    random.seed(7)
    n = min(50, len(good))
    for i, c in enumerate(random.sample(good, n), 1):
        print(f"\n--- {i}. blog={c['blog']} place={c['place']} type={c['type']}/{c['stance']}")
        print(f"    claim_en: {c['claim_en']}")
        print(f"    quote   : {c['quote']}")
        print(f"    post    : {c['post']}")
