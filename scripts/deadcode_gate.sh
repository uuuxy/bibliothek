#!/bin/sh
# Gate gegen unerreichbaren Go-Code.
#
# Nutzt golang.org/x/tools/cmd/deadcode: Das Werkzeug rechnet Erreichbarkeit ab main()
# ueber den echten Aufrufgraphen und nimmt dabei ALLE main-Pakete als Wurzeln, auch die
# unter cmd/. Ein grep kann das nicht — es zaehlt Textvorkommen und haelt schon die
# Erwaehnung eines Namens im eigenen Doc-Kommentar fuer eine Verwendung.
#
# Warum es dieses Gate gibt (04.08.2026): An einem Tag wurde zweimal ein Einstiegspunkt
# entfernt und der Rumpf stehen gelassen. Die verwaisten Funktionen wurden danach nur
# noch von ihren eigenen Tests am Leben gehalten — gruene Tests fuer Code, den die
# Anwendung nie erreicht. Beide Faelle hat erst dieses Werkzeug aufgedeckt.
#
# -test zaehlt Testbinaries als Einstiegspunkt mit. Gemeldet wird deshalb nur, was WEDER
# von der Anwendung NOCH von einem Test erreicht wird — die echten Leichen. Funktionen,
# die nur Tests benutzen, meldet "deadcode ./..." (ohne -test); das ist eine eigene,
# schwaechere Frage und bewusst nicht Teil dieses Gates.
#
# Aufruf: ./scripts/deadcode_gate.sh   (vom Repo-Root)
set -eu

if ! command -v deadcode >/dev/null 2>&1; then
	# Bewusst KEIN stilles Ueberspringen: Ein Waechter, der sich abschaltet, sobald sein
	# Werkzeug fehlt, meldet jahrelang gruen und hat nie etwas geprueft.
	echo "deadcode ist nicht installiert — das Gate kann nicht laufen." >&2
	echo "  go install golang.org/x/tools/cmd/deadcode@latest" >&2
	exit 1
fi

# node_modules enthaelt eine fremde Go-Datei in einem npm-Paket (flatted) und gehoert
# nicht zu diesem Projekt.
LEICHEN=$(deadcode -test ./... 2>/dev/null | grep -v node_modules || true)

if [ -z "$LEICHEN" ]; then
	echo "deadcode: kein unerreichbarer Code."
	exit 0
fi

echo "deadcode: unerreichbarer Code gefunden — entfernen oder verdrahten:" >&2
echo "$LEICHEN" >&2
exit 1
