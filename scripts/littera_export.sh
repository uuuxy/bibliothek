#!/bin/sh
# Gibt die Tabellen einer Littera-Sicherung als CSV aus, in der Form, die cmd/littera-altbestand
# liest. Die Liste der Tabellen steht nur hier; scripts/littera_uebernahme.sh und die Generalprobe
# rufen dieses Skript.
#
#   scripts/littera_export.sh QUELLE ZIEL
#
#   QUELLE  littera.mdb aus der Datensicherung von Littera (gelesen mit mdb-export) oder ein
#           Verzeichnis mit den CSVs derselben Form
#   ZIEL    Verzeichnis für die CSVs
#
# POSIX-sh statt bash: Das Skript läuft auch im Container des Programms (Alpine, ohne bash).
set -eu

TABELLEN="Titel Exemplar Verlag Medienart Personen Personen_Zuordnung Leser Leser_UG Verleih
Schlagworte Schlag_zuord Verweise_Schlagworte Verweis_Zu_Schlagworte Interessenskreis IntZuMed"
# Ohne Ersatzetiketten und ohne Karten des Herstellers fehlen diese beiden Tabellen; die
# Übernahme rechnet dann mit den Nummern aus Littera.
WAHLWEISE="FremdLeserNummer FremdBarcode"

[ $# -eq 2 ] || {
	echo "Aufruf: $0 QUELLE ZIEL" >&2
	exit 1
}
QUELLE=$1
ZIEL=$2
klein() { echo "$1" | tr '[:upper:]' '[:lower:]'; }

mkdir -p "$ZIEL"
if [ -d "$QUELLE" ]; then
	cp "$QUELLE"/*.csv "$ZIEL"/
else
	command -v mdb-export >/dev/null || {
		echo "mdb-export fehlt (mdbtools)" >&2
		exit 1
	}
	vorhanden=$(mdb-tables -1 "$QUELLE")
	for t in $TABELLEN $WAHLWEISE; do
		if printf '%s\n' "$vorhanden" | grep -qx "$t"; then
			mdb-export "$QUELLE" "$t" >"$ZIEL/$(klein "$t").csv"
		fi
	done
fi
for t in $TABELLEN; do
	[ -s "$ZIEL/$(klein "$t").csv" ] || {
		echo "Tabelle $t fehlt in der Sicherung" >&2
		exit 1
	}
done
echo "$(find "$ZIEL" -name '*.csv' | wc -l | tr -d ' ') Tabellen"
