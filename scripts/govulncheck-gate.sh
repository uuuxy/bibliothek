#!/bin/bash
#
# govulncheck mit einer Ausnahmeliste, die sich nicht totstellen kann.
#
# Warum es dieses Skript gibt (17.09.2026): `govulncheck ./...` kennt keine Ausnahme und
# wird rot, sobald irgendeine Schwachstelle einen Aufruf unseres Codes trifft. Am
# 16.09.2026 erschien GO-2026-6452 (excelize) — für ALLE Versionen ab 0, ohne eine mit
# Fix. Damit war kein Push mehr möglich, und zwar für niemanden, obwohl am Programm nichts
# faul war: Der verwundbare Weg ist bei uns nachweislich unerreichbar
# (security/vuln-ausnahmen.json nennt den Nachweis).
#
# Die Alternative wäre gewesen, `--no-verify` zur Gewohnheit zu machen. Dann prüft
# irgendwann niemand mehr etwas. Deshalb dieses Skript: EINE Ausnahme, benannt, begründet,
# mit Nachweis und Wiedervorlage — und alles andere weiterhin rot.
#
# Drei Arten, wie das Gate absichtlich rot wird:
#   1. Eine Schwachstelle trifft unseren Code und steht NICHT in der Liste.
#   2. Eine Ausnahme ist über ihre Wiedervorlage hinaus (niemand hat sie nachgeprüft).
#   3. Eine Ausnahme wird gar nicht mehr gemeldet — dann gibt es einen Fix, und die
#      Ausnahme gehört gelöscht, damit die Liste nicht verrottet.
#
# Aufruf:  ./scripts/govulncheck-gate.sh
set -uo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

AUSNAHMEN="security/vuln-ausnahmen.json"

if ! command -v govulncheck >/dev/null 2>&1; then
    echo "⚠ govulncheck nicht installiert — Scan übersprungen."
    echo "  go install golang.org/x/vuln/cmd/govulncheck@latest"
    exit 0
fi

BERICHT="$(mktemp)"
trap 'rm -f "$BERICHT"' EXIT

# -format json liefert immer Exit 0; über rot oder grün entscheidet die Auswertung unten.
if ! govulncheck -format json ./... > "$BERICHT" 2>/dev/null; then
    echo "✗ govulncheck konnte nicht laufen."
    exit 1
fi

python3 - "$BERICHT" "$AUSNAHMEN" <<'PY'
import json, sys, datetime

bericht, ausnahmendatei = sys.argv[1], sys.argv[2]

# govulncheck schreibt einen Strom von JSON-Objekten, kein Array.
roh = open(bericht, encoding="utf-8").read()
dec, i, objekte = json.JSONDecoder(), 0, []
while i < len(roh):
    while i < len(roh) and roh[i].isspace():
        i += 1
    if i >= len(roh):
        break
    obj, i = dec.raw_decode(roh, i)
    objekte.append(obj)

# Titel je ID, damit die Meldung sagt, worum es geht.
titel = {}
for o in objekte:
    if "osv" in o and isinstance(o["osv"], dict):
        titel[o["osv"].get("id", "")] = (o["osv"].get("summary") or "").strip()

# "Betrifft uns" heißt: Die Spur endet in einer FUNKTION, die unser Code erreicht.
# Ein Fund nur auf Modulebene ist eine Abhängigkeit, die wir nicht aufrufen — genauso
# hat `govulncheck ./...` es bisher gehandhabt.
betroffen, nur_modul = {}, set()
for o in objekte:
    f = o.get("finding")
    if not f:
        continue
    spur = f.get("trace") or []
    id_ = f.get("osv", "")
    if spur and spur[0].get("function"):
        stelle = "%s.%s" % (spur[0].get("package", "?"), spur[0]["function"])
        betroffen.setdefault(id_, set()).add(stelle)
    else:
        nur_modul.add(id_)
nur_modul -= set(betroffen)

with open(ausnahmendatei, encoding="utf-8") as fh:
    ausnahmen = {a["id"]: a for a in json.load(fh)["ausnahmen"]}

heute = datetime.date.today()
fehler = []

for id_, stellen in sorted(betroffen.items()):
    a = ausnahmen.get(id_)
    if not a:
        fehler.append(
            "✗ %s (%s) trifft unseren Code: %s\n"
            "  Keine Ausnahme dafür. Entweder beheben (Abhängigkeit heben, Aufruf meiden)\n"
            "  oder — wenn das Programm nachweislich nicht betroffen ist — mit Grund,\n"
            "  Nachweis und Wiedervorlage in %s eintragen."
            % (id_, titel.get(id_, "ohne Titel"), ", ".join(sorted(stellen)), ausnahmendatei)
        )
        continue
    try:
        frist = datetime.date.fromisoformat(a["wiedervorlage"])
    except (KeyError, ValueError):
        fehler.append("✗ %s: Ausnahme ohne gültige Wiedervorlage (JJJJ-MM-TT)." % id_)
        continue
    if not a.get("grund") or not a.get("nachweis"):
        fehler.append("✗ %s: Ausnahme ohne Grund oder ohne Nachweis." % id_)
        continue
    if heute > frist:
        fehler.append(
            "✗ %s: Die Ausnahme ist seit %s überfällig. Bitte nachprüfen, ob es inzwischen\n"
            "  eine heile Fassung gibt, und die Wiedervorlage neu setzen — oder die Ausnahme\n"
            "  löschen." % (id_, a["wiedervorlage"])
        )
        continue
    print("• %s — bekannt und begründet, Wiedervorlage %s" % (id_, a["wiedervorlage"]))
    print("  %s" % titel.get(id_, ""))
    # Je Nachweis eine Zeile — und BEWUSST nicht im Format "datei.go: Text":
    # actions/setup-go registriert in der CI einen Problem-Matcher für Go, der solche
    # Zeilen für Compiler-Meldungen hält und daraus eine rote Markierung an einem
    # grünen Job macht (17.09.2026 genau so passiert).
    for n in a["nachweis"]:
        print("  Nachweis: %s" % n)

for id_ in sorted(set(ausnahmen) - set(betroffen)):
    fehler.append(
        "✗ %s steht als Ausnahme in %s, wird aber nicht mehr gemeldet.\n"
        "  Dann gibt es einen Fix: Ausnahme löschen, damit die Liste etwas aussagt."
        % (id_, ausnahmendatei)
    )

for id_ in sorted(nur_modul):
    print("· %s — in einer Abhängigkeit, von unserem Code nicht aufgerufen." % id_)

if fehler:
    print()
    print("\n\n".join(fehler))
    sys.exit(1)

print("✓ govulncheck: keine ungeklärte Schwachstelle.")
PY
