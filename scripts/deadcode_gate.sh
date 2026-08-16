#!/usr/bin/env bash
# Gate gegen unerreichbaren Go-Code (freie Funktionen und Methoden).
#
# Anlass (16.08.2026): UpsertBookTitle stand fünf Wochen tot im Repository und
# driftete dabei vom Live-Pfad weg. `deadcode` (x/tools, Erreichbarkeit ab main)
# findet solche Leichen — AUSSER sie hängen an einem Interface: Sobald ein Typ in
# ein Interface gewandelt wird, gilt seine ganze Methodentabelle als erreichbar.
# Für DIESE Klasse gibt es das Schwester-Gate tote_tueren_test.go (läuft in go test).
#
# Baseline: scripts/deadcode_baseline.txt hält die BEGRÜNDETEN Ausnahmen
# (Testinfrastruktur, episodische Littera-Werkzeuge). Jede neue Zeile im
# deadcode-Output, die nicht in der Baseline steht, macht das Gate rot.
#
# Reparatur bei Rot: Code zurückbauen (Regelfall) oder die Zeile mit Begründung
# (Kommentar darüber) in die Baseline aufnehmen. Zeilennummern sind absichtlich
# normalisiert — die Baseline nennt Datei und Funktion, nicht die Zeile.
set -euo pipefail
cd "$(dirname "$0")/.."

IST="$(mktemp)"
SOLL="$(mktemp)"
trap 'rm -f "$IST" "$SOLL"' EXIT

# node_modules ausklammern: das npm-Paket `flatted` liefert eine Go-Datei mit,
# die ohne eigenes go.mod in unser Modul fiele.
go run golang.org/x/tools/cmd/deadcode@v0.48.0 \
  $(go list ./... | grep -v node_modules) \
  | sed -E 's/:[0-9]+:[0-9]+:/:/' | sort > "$IST"
grep -v '^#' scripts/deadcode_baseline.txt | grep -v '^$' | sort > "$SOLL"

NEU="$(comm -23 "$IST" "$SOLL")"
WEG="$(comm -13 "$IST" "$SOLL")"

if [ -n "$NEU" ]; then
  echo "✗ Neuer toter Code (nicht in der Baseline):"
  echo "$NEU"
  echo "→ Zurückbauen oder mit Begründung in scripts/deadcode_baseline.txt aufnehmen."
  exit 1
fi
if [ -n "$WEG" ]; then
  echo "✗ Baseline-Einträge, die es nicht mehr gibt (Baseline aufräumen):"
  echo "$WEG"
  exit 1
fi
echo "✓ deadcode: keine neuen Leichen, Baseline deckungsgleich."
