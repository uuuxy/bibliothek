#!/bin/sh
# SonarQube-Analyse inklusive Coverage.
#
# Warum es dieses Skript gibt (04.08.2026): Der Scan laeuft von Hand — es gibt keinen
# Sonar-Job in der CI. Ein blosser `sonar-scanner`-Aufruf laedt KEINE Coverage hoch, und
# SonarQube wertet fehlende Daten nicht als "unbekannt", sondern als **0 %**. Das Quality
# Gate (Schwelle 80 % auf neuem Code) stand deshalb auf ERROR, obwohl 26 Testpakete gruen
# sind. Die Erzeugung des Reports muss an den Scan gebunden sein, sonst reisst die naechste
# vergessene Zeile das Gate wieder ein.
#
# Der Token kommt aus der Umgebung und wird als SONAR_TOKEN an den Scanner uebergeben,
# NICHT als -Dsonar.token=. Sonst steht er in der Prozessliste und ist fuer jeden
# lesbar, der auf der Maschine `ps` aufrufen kann.
#
# Aufruf:  SONAR_TOKEN=sqp_... ./scripts/sonar_scan.sh   (vom Repo-Root)
#          SONAR_HOST_URL=http://anderer:9000 SONAR_TOKEN=... ./scripts/sonar_scan.sh
set -eu

HOST_URL="${SONAR_HOST_URL:-http://localhost:9000}"

if ! command -v sonar-scanner >/dev/null 2>&1; then
	# Bewusst KEIN stilles Ueberspringen — siehe deadcode_gate.sh, gleiche Begruendung:
	# Ein Waechter, der sich beim Fehlen seines Werkzeugs abschaltet, meldet nie etwas.
	echo "sonar-scanner ist nicht installiert — der Scan kann nicht laufen." >&2
	echo "  brew install sonar-scanner" >&2
	exit 1
fi

if [ -z "${SONAR_TOKEN:-}" ]; then
	echo "SONAR_TOKEN ist nicht gesetzt — der Scanner kann den Bericht nicht hochladen." >&2
	echo "  Token in SonarQube unter My Account -> Security erzeugen, dann:" >&2
	echo "  SONAR_TOKEN=sqp_... ./scripts/sonar_scan.sh" >&2
	exit 1
fi

# Schritt 1: Coverage erzeugen. Laeuft die volle Suite; die auf TEST_DATABASE_URL
# gegateten PG-Tests ueberspringen sich ohne Datenbank von selbst und senken damit die
# gemeldete Abdeckung — wer die echten Zahlen will, setzt TEST_DATABASE_URL vorher.
echo "[1/2] Erzeuge Go-Coverage..."
go test ./... -coverprofile=coverage.out

# Schritt 2: Analyse. sonar-project.properties liefert projectKey, Ausschluesse und den
# Pfad zum Coverage-Report — deshalb stehen hier nur Host und Quellen.
echo "[2/2] Fuehre Analyse aus (${HOST_URL})..."
SONAR_TOKEN="$SONAR_TOKEN" sonar-scanner \
	-Dsonar.sources=. \
	-Dsonar.host.url="$HOST_URL"

echo "Fertig. Ergebnis: ${HOST_URL}/dashboard?id=bibliothek"
