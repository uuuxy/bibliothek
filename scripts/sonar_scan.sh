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

# Schritt 1: Coverage erzeugen.
#
# ZWEI Messfehler lauern hier, beide gemessen am 06.08.2026:
#
#  1. Die auf TEST_DATABASE_URL gegateten *_pg_test.go (58 Dateien!) ueberspringen sich
#     ohne Datenbank STILL. Ihr Code zaehlt dann als ungedeckt, und die Gesamtabdeckung
#     faellt von 45,2 % auf 32,5 % — ohne dass sich am Code irgendetwas geaendert haette.
#     Das Skript sagt darum jetzt an, in welchem Modus es misst, statt eine Zahl
#     auszuliefern, deren Zustandekommen niemand sieht.
#
#  2. frontend/node_modules/flatted/golang/pkg/flatted/flatted.go ist eine fremde
#     Go-Datei in einem JS-Paket. `go list ./...` fuehrt sie als
#     "bibliothek/frontend/node_modules/..." — Go kennt node_modules nicht als
#     Sonderfall. Sie landete mit 115 ungedeckten Zeilen im Profil und drueckte die
#     Quote. sonar-project.properties schliesst sie vom SCAN aus, aber nicht aus dem
#     COVERAGE-Report, den derselbe Scan hochlaedt.
if [ -n "${TEST_DATABASE_URL:-}" ]; then
	echo "[1/3] Erzeuge Go-Coverage — MIT PostgreSQL-Integrationstests."
else
	echo "[1/3] Erzeuge Go-Coverage — OHNE PostgreSQL-Integrationstests." >&2
	echo "      TEST_DATABASE_URL ist nicht gesetzt; 58 *_pg_test.go-Dateien ueberspringen" >&2
	echo "      sich und zaehlen als ungedeckt (rund 13 Prozentpunkte weniger)." >&2
	echo "      Echte Zahlen (siehe docs/SCRIPTS.md):" >&2
	echo "        docker run -d --name biblio-test-pg -e POSTGRES_PASSWORD=test \\" >&2
	echo "          -e POSTGRES_DB=bibliothek_test -p 55432:5432 postgres:16-alpine" >&2
	echo "        export TEST_DATABASE_URL=postgres://postgres:test@localhost:55432/bibliothek_test?sslmode=disable" >&2
fi

# go list statt ./... — nur so bleibt die Fremddatei aus node_modules draussen.
PAKETE="$(go list ./... | grep -v '/node_modules/')"
# shellcheck disable=SC2086 # Paketliste soll in Woerter zerfallen
go test $PAKETE -coverprofile=coverage.out

# Schritt 1b: Frontend-Coverage. Aus demselben Grund an den Scan gebunden wie die
# Go-Abdeckung: Ohne lcov-Bericht zaehlt SonarQube JEDE Frontend-Zeile als ungedeckt (0 %,
# nicht „unbekannt"). Der Bericht deckt die reinen .js-Dateien ab; .svelte kann SonarQube
# nicht parsen und steht deshalb weder im Bericht noch in der Analyse.
echo "[2/3] Erzeuge Frontend-Coverage (vitest)."
( cd frontend && npm run test:coverage >/dev/null ) || {
	echo "Frontend-Tests sind rot — Abbruch, sonst laedt der Scan eine Abdeckung hoch," >&2
	echo "die zu einem kaputten Stand gehoert. Erst 'cd frontend && npm test' klaeren." >&2
	exit 1
}

# Schritt 3: Analyse. sonar-project.properties liefert projectKey, Ausschluesse und die
# Pfade zu beiden Coverage-Berichten — deshalb stehen hier nur Host und Quellen.
echo "[3/3] Fuehre Analyse aus (${HOST_URL})..."
SONAR_TOKEN="$SONAR_TOKEN" sonar-scanner \
	-Dsonar.sources=. \
	-Dsonar.host.url="$HOST_URL"

echo "Fertig. Ergebnis: ${HOST_URL}/dashboard?id=bibliothek"
