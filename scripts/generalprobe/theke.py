"""Theke der Generalprobe (scripts/generalprobe/generalprobe.sh).

Läuft in einem Container am Netz „intern" und spricht mit dem Backend so, wie die Oberfläche es
tut: GET /api/csrf-token, POST /login, POST /api/action. Gibt nur Status, Typ, Nummern, Zeiten
und Zählungen aus, nie ein Leserfeld. Ergebnis für die Prüfungen an der Datenbank:
/probe/theke.json. Rückgabe 1, wenn eine Prüfung hier schon scheitert.
"""
import http.cookiejar
import json
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid
import zlib

BASE = "http://bibliothek-backend-probe:8086"
cj = http.cookiejar.CookieJar()
op = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj))
fehler = []


def req(methode, pfad, body=None, roh=False):
    daten = json.dumps(body).encode() if body is not None else None
    r = urllib.request.Request(BASE + pfad, data=daten, method=methode)
    if daten is not None:
        r.add_header("Content-Type", "application/json")
    if methode != "GET":
        r.add_header("X-CSRF-Token", next((c.value for c in cj if c.name == "csrf_token"), ""))
    t = time.time()
    try:
        antwort = op.open(r, timeout=180)
        code, inhalt = antwort.status, antwort.read()
    except urllib.error.HTTPError as e:
        code, inhalt = e.code, e.read()
    ms = round((time.time() - t) * 1000)
    if roh:
        return code, inhalt, ms
    try:
        return code, json.loads(inhalt or b"null"), ms
    except ValueError:
        return code, None, ms


def aktion(query, leser=None):
    body = {"query": query, "idempotency_key": str(uuid.uuid4())}
    if leser:
        body["active_leser_id"] = leser
    code, d, ms = req("POST", "/api/action", body)
    d = d if isinstance(d, dict) else {}
    return {"code": code, "typ": d.get("type"), "leihe": bool(d.get("loan_id")),
            "meldung": d.get("error") or "", "ms": ms}


def pruefe(bedingung, text):
    print(f"  {'ok ' if bedingung else 'ABWEICHUNG'}  {text}")
    if not bedingung:
        fehler.append(text)


def pdf_text(inhalt):
    text = b""
    for m in re.finditer(rb"stream\r?\n(.*?)\r?\nendstream", inhalt or b"", re.S):
        try:
            text += zlib.decompress(m.group(1))
        except zlib.error:
            text += m.group(1)
    return text


stich = json.load(open("/probe/stichprobe.json", encoding="utf-8"))
leser = json.load(open("/probe/leser.json", encoding="utf-8"))
erg = {}

req("GET", "/api/csrf-token")
code, _, _ = req("POST", "/login", {"email": "probe-admin@test.local", "password": "probe"})
pruefe(code == 200, f"Anmeldung (IMAP-Attrappe): HTTP {code}")

print("\n1) Etikettenwerte scannen, ohne aktiven Leser — das Buch ist bekannt und nicht verliehen")
for s in stich["frei"]:
    a = aktion(s["scan"])
    erkannt = a["code"] == 400 and "nicht ausgeliehen" in a["meldung"]
    pruefe(erkannt and "Transaktionszustand" not in a["meldung"],
           f"Nr {s['nummer']:>7} Etikett {s['scan']}: HTTP {a['code']}, {a['ms']} ms")

print("\n2) Ausleihe und Rückgabe")
buch = stich["frei"][-1]
a = aktion(leser["ausweis"])
pruefe(a["code"] == 200 and a["typ"] == "student", f"Ausweis scannen: HTTP {a['code']} {a['typ']}")
# loan_id kommt nur bei der Rückgabe (für „Rückgängig"); ob die Ausleihe steht, prüft das Skript
# danach an der Datenbank.
a = aktion(buch["scan"], leser["id"])
pruefe(a["code"] == 200 and a["typ"] == "ausleihe", f"Nr {buch['nummer']} ausleihen: HTTP {a['code']} {a['typ']}")
a = aktion(buch["scan"])
pruefe(a["code"] == 200 and a["typ"] == "rueckgabe", f"Nr {buch['nummer']} zurück: HTTP {a['code']} {a['typ']}")
if stich["verliehen"]:
    alt = stich["verliehen"]
    a = aktion(alt["scan"])
    pruefe(a["code"] == 200 and a["typ"] == "rueckgabe",
           f"Nr {alt['nummer']} (in Littera verliehen) zurück: HTTP {a['code']} {a['typ']}")

print("\n3) Buchliste für die Theke ohne Netz")
code, d, ms = req("GET", "/api/action/buchbarcodes")
liste = set((d or {}).get("barcodes") or []) if isinstance(d, dict) else set()
drin = sum(1 for s in stich["frei"] if s["scan"] in liste)
pruefe(code == 200 and drin == len(stich["frei"]),
       f"HTTP {code}, {len(liste)} Nummern, {ms} ms; Stichprobe enthalten: {drin}/{len(stich['frei'])}")
erg["buchliste"] = len(liste)

print("\n4) Etikett nachdrucken")
for s in (stich["frei"][0], stich["frei"][-1]):
    code, pdf, ms = req("POST", "/api/print/labels", {"formatId": "zweckform_l4760", "startPosition": 1,
                        "isQR": False, "items": [{"BarcodeID": s["scan"], "Titel": "Probe", "Autor": ""}]}, roh=True)
    steht = s["scan"].encode() in pdf_text(pdf)
    pruefe(code == 200 and (pdf or b"")[:5] == b"%PDF-" and steht,
           f"Nr {s['nummer']}: HTTP {code}, PDF mit der Nummer {s['scan']}: {steht}")

print("\n5) Mahnwesen und Mahnbriefe")
code, d, ms = req("GET", "/api/mahnwesen")
klassen = (d or {}).get("klassen") or [] if isinstance(d, dict) else []
schueler = sum(len(k.get("schueler") or []) for k in klassen)
pruefe(code == 200 and klassen, f"Mahnwesen: HTTP {code}, {len(klassen)} Klassen, {schueler} Schüler, {ms} ms")
# Die Tür der Oberfläche („Mahnbriefe drucken"): Sie nimmt die Ausleihen und zählt die Mahnung.
ohne_pdf = 0
for k in klassen:
    ausleihen = [m["ausleihe_id"] for s in k.get("schueler") or [] for m in s.get("medien") or []]
    c, p, _ = req("POST", "/api/admin/mahnungen/bulk-print", {"ausleih_ids": ausleihen}, roh=True)
    ohne_pdf += c != 200 or not (p or b"").startswith(b"%PDF-")
pruefe(ohne_pdf == 0, f"Mahnbriefe: {len(klassen) - ohne_pdf} von {len(klassen)} Klassen als PDF")
erg["mahnwesen"] = {"klassen": len(klassen), "schueler": schueler}

print("\n6) Leserdatei und Katalog")
code, _, ms = req("GET", "/api/schueler")
pruefe(code == 200, f"Leserdatei: HTTP {code}, {ms} ms")
for q in ("Deutsch", "Mathematik 7"):
    code, _, ms = req("GET", "/api/public/opac/suche?q=" + urllib.parse.quote(q))
    pruefe(code == 200, f"Katalog „{q}“: HTTP {code}, {ms} ms")

erg["fehler"] = fehler
json.dump(erg, open("/probe/theke.json", "w", encoding="utf-8"))
sys.exit(1 if fehler else 0)
