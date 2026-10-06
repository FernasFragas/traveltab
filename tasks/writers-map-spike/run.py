"""Ephemeral names-only benchmark for the writers-map index spike.

Fetched post bodies stay in memory. The SQLite database contains only post IDs,
matched QIDs, normalized names, and character offsets; no URLs/titles/body text.
"""
import html
import json
import re
import sqlite3
import time
import unicodedata
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parent
DB = ROOT / "spike.sqlite"
NAMES = ROOT / "wikidata-names.json"
UA = "TravelTab-research/0.1 (blog overlap check)"
HOSTS = ["www.viajecomigo.com", "www.saltinourhair.com"]
LIMITER_SECONDS = 1.5

def request(url, data=None, timeout=90):
    req = urllib.request.Request(url, data=data, headers={
        "User-Agent": UA,
        "Content-Type": "application/x-www-form-urlencoded" if data else "application/json",
    })
    try:
        return urllib.request.urlopen(req, timeout=timeout)
    except urllib.error.HTTPError as exc:
        if exc.code in (401, 403, 429):
            raise RuntimeError(f"source returned HTTP {exc.code}; stopping without retry")
        raise

def fold(value):
    return unicodedata.normalize("NFKD", value).encode("ascii", "ignore").decode().casefold()

def fetch_madeira_places():
    if NAMES.exists():
        return json.loads(NAMES.read_text())
    # Visitable classes used for this one-off candidate list. Airports and
    # administrative municipalities are explicitly excluded.
    query = '''SELECT ?place ?placeLabel ?alias ?coord WHERE {
      SERVICE wikibase:around {
        ?place wdt:P625 ?coord .
        bd:serviceParam wikibase:center "Point(-16.95 32.73)"^^geo:wktLiteral;
          wikibase:radius "40"; wikibase:distance ?distance .
      }
      ?place wdt:P31/wdt:P279* ?class .
      VALUES ?class { wd:Q570116 wd:Q839954 wd:Q473972 wd:Q473972
        wd:Q33506 wd:Q46169 wd:Q22698 wd:Q8502 wd:Q23442 wd:Q39816
        wd:Q179049 wd:Q173387 wd:Q207326 wd:Q17350442 wd:Q839954 }
      FILTER NOT EXISTS { ?place wdt:P31/wdt:P279* wd:Q1248784 }
      FILTER NOT EXISTS { ?place wdt:P31/wdt:P279* wd:Q15284 }
      OPTIONAL { ?place skos:altLabel ?alias . FILTER(LANG(?alias) IN ("en", "pt")) }
      SERVICE wikibase:label { bd:serviceParam wikibase:language "en,pt". }
    }'''
    url = "https://query.wikidata.org/sparql?" + urllib.parse.urlencode({"query": query, "format": "json"})
    with request(url, timeout=120) as response:
        rows = json.load(response)["results"]["bindings"]
    places = {}
    for row in rows:
        qid = row["place"]["value"].rsplit("/", 1)[-1]
        entry = places.setdefault(qid, set())
        entry.add(row.get("placeLabel", {}).get("value", ""))
        entry.add(row.get("alias", {}).get("value", ""))
    # A QID label service emits whichever language is available; query labels
    # directly to ensure en/pt aliases are represented.
    ids = list(places)
    for start in range(0, len(ids), 50):
        params = urllib.parse.urlencode({"action": "wbgetentities", "ids": "|".join(ids[start:start+50]),
            "props": "labels|aliases", "languages": "en|pt", "format": "json"})
        with request("https://www.wikidata.org/w/api.php?" + params) as response:
            entities = json.load(response)["entities"]
        for qid, entity in entities.items():
            for value in entity.get("labels", {}).values():
                places[qid].add(value["value"])
            for aliases in entity.get("aliases", {}).values():
                places[qid].update(x["value"] for x in aliases)
        time.sleep(2)
    cleaned = {}
    for qid, names in places.items():
        for name in names:
            name = name.strip()
            if len(name) >= 5 and not name.isnumeric():
                cleaned.setdefault(fold(name), []).append((qid, name))
    NAMES.write_text(json.dumps(cleaned, ensure_ascii=False, separators=(",", ":")))
    print(f"Wikidata candidate set: {len(places)} QIDs, {len(cleaned)} folded names", flush=True)
    return cleaned

def compile_patterns(aliases):
    return [(norm, qid, raw, re.compile(r"(?<!\w)" + re.escape(norm) + r"(?!\w)", re.I if " " in norm else 0))
            for norm, values in aliases.items() for qid, raw in values]

def plain_text(markup):
    markup = re.sub(r"(?is)<(script|style)[^>]*>.*?</\1>", " ", markup)
    return re.sub(r"\s+", " ", html.unescape(re.sub(r"<[^>]+>", " ", markup))).strip()

def init_db():
    DB.unlink(missing_ok=True)
    db = sqlite3.connect(DB)
    db.execute("PRAGMA journal_mode=DELETE")
    db.execute("PRAGMA page_size=4096")
    db.executescript('''
      CREATE TABLE name_occurrences(
        post_id TEXT NOT NULL,
        qid TEXT NOT NULL,
        phrase TEXT NOT NULL,
        position INTEGER NOT NULL,
        original_case INTEGER NOT NULL,
        PRIMARY KEY(post_id, qid, position, phrase)
      ) WITHOUT ROWID;
      CREATE INDEX name_phrase_lookup ON name_occurrences(phrase, post_id);
      CREATE INDEX name_qid_lookup ON name_occurrences(qid, post_id);
    ''')
    return db

def main():
    aliases = fetch_madeira_places()
    patterns = compile_patterns(aliases)
    db = init_db()
    total_posts, total_rows = 0, 0
    per_host = {}
    for host in HOSTS:
        endpoint = f"https://{host}/wp-json/wp/v2/posts?per_page=100&page=1&_fields=id,content"
        with request(endpoint) as response:
            total_pages = int(response.headers.get("X-WP-TotalPages", "1"))
            first = json.load(response)
        print(f"{host}: {total_pages} pages", flush=True)
        host_posts, host_rows = 0, 0
        for page in range(1, total_pages + 1):
            rows = first if page == 1 else None
            if rows is None:
                time.sleep(LIMITER_SECONDS)
                url = f"https://{host}/wp-json/wp/v2/posts?per_page=100&page={page}&_fields=id,content"
                with request(url) as response:
                    rows = json.load(response)
            pending = []
            for post in rows:
                text = plain_text(post["content"]["rendered"])
                folded = fold(text)
                spans = []
                for norm, qid, raw, pattern in patterns:
                    for match in pattern.finditer(folded):
                        spans.append((match.start(), match.end(), qid, norm, raw, text[match.start():match.end()]))
                spans.sort(key=lambda x: (x[0], -(x[1] - x[0])))
                kept = []
                for span in spans:
                    if any(prev[0] <= span[0] and prev[1] >= span[1] for prev in kept):
                        continue
                    kept.append(span)
                    pending.append((f"{host}:{post['id']}", span[2], span[3], span[0], int(span[5] == span[4])))
                host_posts += 1
            db.executemany("INSERT OR IGNORE INTO name_occurrences VALUES(?,?,?,?,?)", pending)
            db.commit()
            host_rows += len(pending)
            print(f"{host}: page {page}/{total_pages}; {host_posts} posts, {host_rows} name rows", flush=True)
            if page != total_pages:
                time.sleep(LIMITER_SECONDS)
        per_host[host] = {"posts": host_posts, "name_occurrences": host_rows}
        total_posts += host_posts
        total_rows += host_rows
    db.execute("PRAGMA optimize")
    db.commit()
    db.close()
    size = DB.stat().st_size
    db = sqlite3.connect(DB)
    # Warm the query plan once, then report median of 101 indexed phrase lookups.
    frequent = db.execute("SELECT phrase FROM name_occurrences GROUP BY phrase ORDER BY count(*) DESC LIMIT 1").fetchone()
    phrase = frequent[0] if frequent else "madeira"
    times = []
    for _ in range(101):
        start = time.perf_counter_ns()
        db.execute("SELECT post_id,qid,position FROM name_occurrences WHERE phrase=? LIMIT 200", (phrase,)).fetchall()
        times.append((time.perf_counter_ns() - start) / 1e6)
    times.sort()
    lookup_ms = times[len(times)//2]
    db.close()
    print(json.dumps({"places": len({qid for values in aliases.values() for qid, _ in values}), "aliases": len(aliases),
        "total_posts": total_posts, "rows": total_rows, "per_host": per_host,
        "sqlite_bytes": size, "mb_decimal": size / 1_000_000,
        "rows_per_1000_posts": total_rows * 1000 / total_posts if total_posts else 0,
        "projection_20400_mb": size * 20400 / total_posts / 1_000_000 if total_posts else 0,
        "lookup_p50_ms": lookup_ms, "lookup_phrase": phrase}, ensure_ascii=False))

if __name__ == "__main__":
    main()
