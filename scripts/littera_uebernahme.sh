#!/usr/bin/env bash
# Littera-Übernahme am Server: eine Sicherung aus Littera mit Büchern, Lesern und Ausleihen in die
# Datenbank des Programms übernehmen. Ablauf und Voraussetzungen: docs/DEPLOYMENT.md, Abschnitt 9.
#
#   scripts/littera_uebernahme.sh SICHERUNG [--ja] [Schalter der Übernahme …]
#
#   SICHERUNG  littera.mdb aus der Datensicherung von Littera, oder ein Verzeichnis mit den CSVs
#   --ja       ohne Rückfrage übernehmen, wenn der Probelauf durchgeht
#   weitere    gehen an cmd/littera-altbestand, etwa -kein-standort "…" (docs/SCRIPTS.md, Abschnitt 1)
#
# Erst ein Probelauf, der nur liest; hält er an, schreibt das Skript nichts. Übernommen wird nur in
# eine frisch angelegte Datenbank: Stehen schon Bücher, Schüler oder Ausleihen darin, etwa aus
# einem LUSD-Import, hält es vorher an, denn die Übernahme legte jeden Schüler ein zweites Mal an.
#
# Gelesen und geschrieben wird in einem Wegwerf-Container aus dem Image des Programms, mit dessen
# Verbindung zur Datenbank; mdb-export und das Übernahme-Programm liegen im Image, der Server
# braucht nur Docker. Die CSVs löscht das Skript am Ende; das Protokoll bleibt zur Durchsicht und
# trägt Namen.
#
# Rückgabe wie cmd/littera-altbestand: 0 = vollständig übernommen, 1 = angehalten oder abgebrochen,
# 2 = übernommen, aber Datensätze fehlen (FEHLER-Zeilen im Protokoll).
set -euo pipefail

WURZEL="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$WURZEL"
# docker compose liest COMPOSE_FILE aus der Umgebung; so lässt sich das Skript gegen einen
# Wegwerf-Stack prüfen, ohne eine zweite Fassung.
COMPOSE=(docker compose)

case "${1:-}" in
-h | --help)
	sed -n '2,21p' "$0"
	exit 0
	;;
"" | -*)
	echo "Aufruf: $0 SICHERUNG [--ja] [Schalter der Übernahme …]" >&2
	exit 1
	;;
esac
[ -e "$1" ] || {
	echo "Nicht gefunden: $1" >&2
	exit 1
}
QUELLE="$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
shift
JA=0 SCHALTER=()
for a in "$@"; do
	if [ "$a" = --ja ]; then JA=1; else SCHALTER+=("$a"); fi
done

abbruch() {
	printf '\nANGEHALTEN: %s\n' "$*" >&2
	exit 1
}
schritt() { printf '\n== %s\n' "$*"; }

ARBEIT="$(mktemp -d "${TMPDIR:-/tmp}/littera-uebernahme.XXXXXX")"
chmod 700 "$ARBEIT"
# Die CSVs tragen alle Personendaten; das Protokoll bleibt nur, wenn ein Lauf es geschrieben hat.
aufraeumen() {
	rm -rf "$ARBEIT/export"
	[ -e "$ARBEIT/littera_import.log" ] || rm -rf "$ARBEIT"
}
trap aufraeumen EXIT

# Ein Wegwerf-Container aus dem Image des Programms, mit dessen Umgebung (DATABASE_URL) und Netzen.
# Er läuft unter dem Benutzer des Aufrufers, damit er ins Arbeitsverzeichnis schreiben darf.
im_programm() {
	"${COMPOSE[@]}" run --rm --no-deps -T --user "$(id -u):$(id -g)" \
		-v "$QUELLE:/uebernahme/quelle:ro" \
		-v "$WURZEL/scripts/littera_export.sh:/uebernahme/littera_export.sh:ro" \
		-v "$ARBEIT:/uebernahme/arbeit" -w /uebernahme/arbeit \
		backend "$@"
}
uebernahme() {
	im_programm /app/littera-altbestand -csv export -personen -ausleihen \
		${SCHALTER[@]+"${SCHALTER[@]}"} "$@"
}

schritt "Programm und Datenbank"
command -v docker >/dev/null || abbruch "docker fehlt"
im_programm test -x /app/littera-altbestand ||
	abbruch "Das Programm enthält die Übernahme noch nicht. Erst die neue Fassung einspielen (./update.sh), dann erneut."
# Bücher, Schüler und Ausleihen bringt die Übernahme selbst mit. Ein Konto des ersten Admins
# stört nicht, es ist kein Schüler.
# shellcheck disable=SC2016 # $DATABASE_URL und $1 gelten erst im Container.
belegt="$(im_programm sh -c 'psql "$DATABASE_URL" -X -q -tA -c "$1"' sh \
	"SELECT (SELECT count(*) FROM buecher_titel) || ' Titel, ' ||
	        (SELECT count(*) FROM buecher_exemplare) || ' Exemplare, ' ||
	        (SELECT count(*) FROM schueler) || ' Schüler, ' ||
	        (SELECT count(*) FROM ausleihen) || ' Ausleihen'")" ||
	abbruch "Die Datenbank antwortet nicht. Läuft das Programm (docker compose up -d)?"
[ "$belegt" = "0 Titel, 0 Exemplare, 0 Schüler, 0 Ausleihen" ] ||
	abbruch "In der Datenbank stehen schon $belegt. Die Übernahme läuft nur in eine frisch angelegte Datenbank (docs/DEPLOYMENT.md, Abschnitt 9)."
echo "  Datenbank ist leer, Übernahme im Programm vorhanden"

schritt "Sicherung lesen"
im_programm sh /uebernahme/littera_export.sh /uebernahme/quelle export ||
	abbruch "Die Sicherung ließ sich nicht lesen."

schritt "Probelauf (liest nur, schreibt nichts)"
uebernahme -trocken ||
	abbruch "Der Probelauf hält an, Grund in der Ausgabe oben. Leser ohne Gruppe in Littera einer Gruppe zuordnen, neu sichern und erneut starten."

if [ "$JA" != 1 ]; then
	printf '\nJetzt in die Datenbank übernehmen? Zum Bestätigen ja eingeben: '
	antwort=""
	read -r antwort </dev/tty || true
	[ "$antwort" = ja ] || abbruch "Nicht übernommen."
fi

schritt "Übernahme"
code=0
uebernahme || code=$?
case "$code" in
0) echo "Vollständig übernommen." ;;
2) echo "Übernommen, aber nicht vollständig: Die fehlenden Datensätze stehen als FEHLER im Protokoll." ;;
*) echo "Abgebrochen (Rückgabe $code), Grund in der Ausgabe oben." ;;
esac
if [ -e "$ARBEIT/littera_import.log" ]; then
	printf 'Protokoll (enthält Namen): %s\nNach der Durchsicht löschen: rm -r %s\n' "$ARBEIT/littera_import.log" "$ARBEIT"
fi
# Für einen zweiten Versuch wird die Sicherung noch gebraucht; nach einer Übernahme nennt das
# Skript, wann sie gelöscht wird.
if [ "$code" = 0 ] || [ "$code" = 2 ]; then
	printf 'Die Sicherung trägt alle Schülerdaten: nach den Abnahmen am Server und auf dem USB-Stick löschen (%s).\n' "$QUELLE"
fi
exit "$code"
