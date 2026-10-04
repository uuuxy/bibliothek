"""Helfer der Generalprobe auf dem Rechner (scripts/generalprobe/generalprobe.sh).

Gibt nie einen Namen aus: nur Nummern, Zählungen und Gruppenbezeichnungen.

  zuordnen   VERZ 'Untergruppe=Ziel' …   stellt eine Zuordnung in Littera nach (Kopie des Exports)
  stichprobe VERZ VERLIEHEN AUS           Etikettenwerte je Stellenzahl, dazu ein verliehenes Buch
  protokoll  LOG                          littera_import.log nach Schwere und Grund gezählt
"""
import csv
import json
import random
import re
import sys
from collections import Counter

csv.field_size_limit(10**9)

# Litteras Spalte Barcode ist der EAN-13 in EAN-Schrift-Setzform (docs/SCRIPTS.md, Abschnitt 1):
# „8 *pkpööp#-c.bc-*" — die Tastaturreihen stehen für die Ziffern 1…9, 0. Hier unabhängig von
# littera.EtikettZiffern gelesen: Die Probe soll prüfen, ob der Wert vom Etikett an der Theke
# ein Buch findet, nicht ob der Code mit sich selbst übereinstimmt.
ZIFFER = {z: (i + 1) % 10 for reihe in ("qwertzuiop", "asdfghjklö", "yxcvbnm,.-") for i, z in enumerate(reihe)}
SETZFORM = re.compile(r"^(\d) \*(.{6})#(.{4})(.)(.)\*$")


def etikett(roh):
    m = SETZFORM.match((roh or "").strip())
    if not m:
        return None
    try:
        return m.group(1) + "".join(str(ZIFFER[z]) for z in "".join(m.groups()[1:]))
    except KeyError:
        return None


def zuordnen(verz, paare):
    pfad = verz + "/leser_ug.csv"
    with open(pfad, encoding="utf-8", newline="") as f:
        zeilen = list(csv.reader(f))
    kopf = zeilen[0]
    spalte = kopf.index("Untergruppe")
    ziel = dict(p.split("=", 1) for p in paare)
    getroffen = Counter()
    for z in zeilen[1:]:
        name = z[spalte].strip()
        if name in ziel:
            z[spalte] = ziel[name]
            getroffen[name] += 1
    for name in ziel:
        if not getroffen[name]:
            sys.exit(f"FEHLER: Lesergruppe „{name}“ steht nicht in leser_ug.csv")
        print(f"  Zuordnung wie in Littera nachgestellt: „{name}“ → „{ziel[name]}“ ({getroffen[name]} Gruppe)")
    with open(pfad, "w", encoding="utf-8", newline="") as f:
        csv.writer(f).writerows(zeilen)


def stichprobe(verz, verliehen_pfad, aus):
    verliehen = {z.strip() for z in open(verliehen_pfad, encoding="utf-8") if z.strip()}
    frei, offen = {}, []
    with open(verz + "/exemplar.csv", encoding="utf-8", newline="") as f:
        for r in csv.DictReader(f):
            nr, scan = (r.get("Exemplarnummer") or "").strip(), etikett(r.get("Barcode"))
            if not nr or not scan:
                continue
            if nr in verliehen:
                offen.append({"nummer": nr, "scan": scan})
            else:
                frei.setdefault(len(nr), []).append({"nummer": nr, "scan": scan})
    zufall = random.Random(28092026)
    auswahl = [e for laenge in sorted(frei) for e in zufall.sample(frei[laenge], min(3, len(frei[laenge])))]
    ergebnis = {"frei": auswahl, "verliehen": zufall.choice(offen) if offen else None}
    json.dump(ergebnis, open(aus, "w", encoding="utf-8"))
    print(f"  Stichprobe: {len(auswahl)} freie Exemplare, Stellenzahlen {sorted(frei)}; "
          f"dazu {len(offen)} verliehene")


def kopf(grund):
    """Der Anfang eines Grundes, ohne Werte: Namen stehen nur hinter „ – " oder „: "."""
    grund = re.split(r" – |: ", grund, maxsplit=1)[0]
    grund = re.sub(r"[A-Za-z0-9._%+-]{1,64}@[A-Za-z0-9.-]{1,255}", "<mail>", grund)
    grund = re.sub(r"„[^“]*“", "„…“", grund)
    grund = re.sub(r'"[^"]*"', '"…"', grund)
    grund = re.sub(r"\([^)]*\)", "(…)", grund)
    return re.sub(r"\d+", "#", grund)[:110]


def protokoll(log):
    zeile = re.compile(r'^\[[^\]]*\] (\S+) \S+ kennung=".*?(?<!\\)" grund=(.*)$')
    zaehler, unlesbar = Counter(), 0
    for z in open(log, encoding="utf-8", errors="replace"):
        m = zeile.match(z.rstrip("\n"))
        if not m:
            unlesbar += 1
            continue
        zaehler[(m.group(1), kopf(m.group(2)))] += 1
    for (schwere, grund), n in sorted(zaehler.items(), key=lambda x: (x[0][0], -x[1])):
        print(f"  {n:>6}  {schwere:<8} {grund}")
    if unlesbar:
        print(f"  {unlesbar:>6}  Zeilen in anderer Form (nicht angezeigt)")


if __name__ == "__main__":
    befehl, argumente = sys.argv[1], sys.argv[2:]
    {"zuordnen": lambda: zuordnen(argumente[0], argumente[1:]),
     "stichprobe": lambda: stichprobe(*argumente),
     "protokoll": lambda: protokoll(*argumente)}[befehl]()
