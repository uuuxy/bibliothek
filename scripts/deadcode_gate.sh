#!/usr/bin/env bash
# Gate gegen unerreichbaren Go-Code.
#
# Nutzt golang.org/x/tools/cmd/deadcode: Das Werkzeug rechnet Erreichbarkeit ab main()
# ueber den echten Aufrufgraphen und nimmt dabei alle main-Pakete als Wurzeln, auch die
# unter cmd/. Ein grep kann das nicht — es zaehlt Textvorkommen und haelt schon die
# Erwaehnung eines Namens im eigenen Doc-Kommentar fuer eine Verwendung.
#
# Wird ein Einstiegspunkt entfernt und der Rumpf stehen gelassen, halten nur noch die
# eigenen Tests die verwaisten Funktionen am Leben — gruene Tests fuer Code, den die
# Anwendung nie erreicht, und der dabei vom Live-Pfad wegdriftet.
#
# Deshalb ohne -test: Rot ist auch Code, den nur Tests erreichen. Das Rauschen dieser Frage
# faengt die begruendete Baseline (scripts/deadcode_baseline.txt): Jeder Eintrag braucht
# eine Begruendung, jeder neue Fund macht das Gate rot (Ratsche).
#
# Grenze des Werkzeugs, per Gegenprobe belegt: deadcode sieht keine toten
# Interface-Methoden — nach der Wandlung eines Typs in ein Interface gilt seine ganze
# Methodentabelle als erreichbar. Fuer diese Klasse gibt es das Schwester-Gate
# tote_tueren_test.go (laeuft in jedem `go test ./...`).
#
# Reparatur bei Rot: Code zurueckbauen (Regelfall) oder mit Begruendung in die
# Baseline aufnehmen. Zeilennummern sind normalisiert — die Baseline nennt Datei und
# Funktion, nicht die Zeile.
#
# Aufruf: ./scripts/deadcode_gate.sh   (vom Repo-Root; pre-push [6/6] und CI)
set -euo pipefail
cd "$(dirname "$0")/.."

IST="$(mktemp)"
SOLL="$(mktemp)"
trap 'rm -f "$IST" "$SOLL"' EXIT

# Gepinnte Version statt install-Pruefung: `go run` besorgt das Werkzeug selbst,
# reproduzierbar und ohne stilles Ueberspringen. node_modules ausklammern: das
# npm-Paket `flatted` liefert eine Go-Datei mit, die ohne eigenes go.mod in unser
# Modul fiele.
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
