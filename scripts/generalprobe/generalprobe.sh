#!/usr/bin/env bash
# Generalprobe: eine Littera-Sicherung in eine frische, abgeschottete Datenbank übernehmen und
# durch die Anwendung prüfen — Neuaufbau aus den Migrationen, Übernahme, Theke, Mahnwesen,
# Nachtsicherung und Wiederherstellung nach docs/resilience_and_recovery.md 2a. Bedienung:
# docs/SCRIPTS.md, Abschnitt 1b.
#
#   scripts/generalprobe/generalprobe.sh QUELLE [--gruppe 'Untergruppe=Ziel'] … [--behalten]
#
#   QUELLE      littera_sav.mdb (Access, gelesen mit mdb-export) oder ein Verzeichnis mit den
#               CSVs derselben Form, etwa aus der SQL-Server-Sicherung ausgegeben
#   --gruppe    stellt eine Zuordnung nach, die die Bücherei in Littera vornähme, bevor sie die
#               Sicherung zieht („Undefinierte Untergruppe=Schüler") — nur in der Kopie des
#               Exports, nur für die Probe
#   --behalten  lässt Stack und Arbeitsverzeichnis stehen. Dort liegen Personendaten: danach
#               selbst löschen (die Befehle stehen am Ende der Ausgabe)
#
# Personendaten: Export, Protokoll und Dumps liegen nur im Arbeitsverzeichnis (mktemp, 700) und
# werden am Ende gelöscht, der Stack mit `down -v`. Ausgegeben werden Zählungen, Nummern und
# Gruppenbezeichnungen, nie ein Name. Nichts verlässt den Rechner: Das Backend hat keinen Weg ins
# Internet (geprüft, bevor Daten hineinkommen), die Nachtsicherung läuft ohne S3 und ohne Mail.
#
# Rückgabe: 0 = jede Prüfung bestanden, 1 = eine Prüfung abgewichen oder die Probe abgebrochen.
set -euo pipefail

HIER="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WURZEL="$(cd "$HIER/../.." && pwd)"
COMPOSE=(docker compose -f "$HIER/docker-compose.yml")
DB="postgresql://postgres:probe-nur-lokal@127.0.0.1:5436"
NETZ=bibliothek-probe_intern
SCHLUESSEL=probe-backup-schluessel-nur-lokal-32-zeichen
TABELLEN=(Titel Exemplar Verlag Medienart Personen Personen_Zuordnung Leser Leser_UG Verleih
	Schlagworte Schlag_zuord Verweise_Schlagworte Verweis_Zu_Schlagworte Interessenskreis IntZuMed)
[ -d /opt/homebrew/opt/libpq/bin ] && PATH="/opt/homebrew/opt/libpq/bin:$PATH"

QUELLE="" BEHALTEN=0 GRUPPEN=()
while [ $# -gt 0 ]; do
	case "$1" in
	--gruppe) GRUPPEN+=("$2"); shift 2 ;;
	--behalten) BEHALTEN=1; shift ;;
	-h | --help) sed -n '2,24p' "$0"; exit 0 ;;
	*) QUELLE="$1"; shift ;;
	esac
done
[ -n "$QUELLE" ] && [ -e "$QUELLE" ] || { echo "Aufruf: $0 QUELLE [--gruppe 'Untergruppe=Ziel'] [--behalten]" >&2; exit 1; }
QUELLE="$(cd "$(dirname "$QUELLE")" && pwd)/$(basename "$QUELLE")"

ABWEICHUNGEN=()
ok() { printf '  ok          %s\n' "$*"; }
abweichung() { printf '  ABWEICHUNG  %s\n' "$*"; ABWEICHUNGEN+=("$*"); }
pruefe() { if [ "$1" = "$2" ]; then ok "$3"; else abweichung "$3 (ist $1, soll $2)"; fi; }
abbruch() { printf '\nABBRUCH: %s\n' "$*"; exit 1; }
schritt() { printf '\n== %s\n' "$*"; }
sql() { PGOPTIONS='--client-min-messages=warning' psql -X -q -tA "$DB/${2:-bibliothek}" -c "$1"; }

ARBEIT="$(mktemp -d "${TMPDIR:-/tmp}/generalprobe.XXXXXX")"
chmod 700 "$ARBEIT"
aufraeumen() {
	docker rm -f probe-restore-backend >/dev/null 2>&1 || true
	if [ "$BEHALTEN" = 1 ]; then
		printf '\nStehen gelassen (Personendaten): %s und der Stack. Löschen mit:\n' "$ARBEIT"
		printf '  %s down -v && rm -rf %s\n' "${COMPOSE[*]}" "$ARBEIT"
		return
	fi
	"${COMPOSE[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
	rm -rf "$ARBEIT"
	printf '\nAufgeräumt: Stack samt Volumes und Arbeitsverzeichnis gelöscht.\n'
}
trap aufraeumen EXIT

# Nur die Zeilen der Übernahme zeigen, die keine Personendaten tragen können: Zählungen,
# Abgleiche, Gruppen. Der Grund eines Abbruchs kann einen Namen nennen — er bleibt in der Datei.
zeige_lauf() {
	sed -E 's/^[0-9]{4}\/[0-9]{2}\/[0-9]{2} [0-9:]{8} //' "$1" |
		sed -E 's/^(⚠  Die Übernahme endete vorzeitig):.*/\1 (Ursache in der Ausgabe des Laufs)/' |
		grep -E '^(Gelesen|TROCKENLAUF|Übernehme|FEHLER: Leser ohne|━|Bestand |Schlagworte |Personen |Ausleihen |⚠|ℹ|  (Titel mit|davon|Verlage|Schlagworte:|Interessenkreise:|Fach:|Standorte:|Vermerke am Titel|Sonderstandorte der|Leser:|Ausleihen:|→|Lesergruppe|Warnungen|Fehler \(NICHT)|  +(geschrieben|Interessenkreise|✓|⚠|Fach aus|Standort:|[0-9]+ Ausleihen tragen))' || true
}

schritt "Werkzeuge"
for w in docker go python3 psql pg_dump; do command -v "$w" >/dev/null || abbruch "$w fehlt"; done
docker info >/dev/null 2>&1 || abbruch "Docker läuft nicht"
[ -d "$QUELLE" ] || command -v mdb-export >/dev/null || abbruch "mdb-export fehlt (brew install mdbtools)"
pg_dump --version | grep -q ' 18\.' || abbruch "pg_dump muss Version 18 sein (wie der Server): $(pg_dump --version)"
STAND="$(git -C "$WURZEL" rev-parse --short HEAD)"
echo "  Stand $STAND, Quelle $(basename "$QUELLE"), Arbeitsverzeichnis $ARBEIT"
[ -z "$(git -C "$WURZEL" status --porcelain)" ] ||
	echo "  Achtung: Der Arbeitsbaum ist nicht sauber — das Image enthält Änderungen, die in keinem Commit stehen."

schritt "Neuaufbau: Image bauen, leere Datenbank, Migrationen beim Start"
"${COMPOSE[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
docker build --build-arg GIT_COMMIT="$(git -C "$WURZEL" rev-parse HEAD)" -t bibliothek-probe:latest "$WURZEL" >"$ARBEIT/build.log" 2>&1 ||
	abbruch "docker build scheiterte (build.log)"
"${COMPOSE[@]}" up -d >"$ARBEIT/up.log" 2>&1 || abbruch "Stack startet nicht (Port 5436 belegt?)"
for _ in $(seq 1 60); do
	[ "$(docker exec bibliothek-werkzeug-probe curl -s -o /dev/null -w '%{http_code}' http://bibliothek-backend-probe:8086/health || true)" = 200 ] && break
	sleep 2
done
pruefe "$(docker exec bibliothek-werkzeug-probe curl -s -o /dev/null -w '%{http_code}' http://bibliothek-backend-probe:8086/health || true)" 200 "Backend antwortet"
pruefe "$(sql 'SELECT count(*) FROM schema_migrations')" "$(find "$WURZEL/migrations" -maxdepth 1 -name '*.sql' | wc -l | tr -d ' ')" "alle Migrationen eingetragen"

schritt "Abschottung, bevor Daten hineinkommen"
if docker exec bibliothek-backend-probe sh -c 'wget -T 5 -q -O /dev/null https://www.google.com' >/dev/null 2>&1; then
	abbruch "Das Backend erreicht das Internet"
fi
ok "Backend ohne Weg ins Internet"
pruefe "$(docker inspect -f '{{range $n, $_ := .NetworkSettings.Networks}}{{$n}} {{end}}' bibliothek-backend-probe | xargs)" "$NETZ" "Backend nur am Netz $NETZ"

schritt "Export"
mkdir "$ARBEIT/export"
if [ -d "$QUELLE" ]; then
	cp "$QUELLE"/*.csv "$ARBEIT/export/"
else
	for t in "${TABELLEN[@]}" FremdLeserNummer FremdBarcode; do
		if mdb-tables -1 "$QUELLE" | grep -qx "$t"; then
			mdb-export "$QUELLE" "$t" >"$ARBEIT/export/$(echo "$t" | tr 'A-Z' 'a-z').csv"
		fi
	done
fi
for t in "${TABELLEN[@]}"; do
	[ -s "$ARBEIT/export/$(echo "$t" | tr 'A-Z' 'a-z').csv" ] || abbruch "Tabelle $t fehlt im Export"
done
echo "  $(ls "$ARBEIT/export" | wc -l | tr -d ' ') Tabellen"

schritt "Übernahme-Werkzeuge bauen"
(cd "$WURZEL" && go build -o "$ARBEIT/littera-altbestand" ./cmd/littera-altbestand &&
	go build -o "$ARBEIT/restore-backup" ./cmd/restore-backup &&
	go build -o "$ARBEIT/nachtsicherung" scripts/generalprobe/nachtsicherung.go) || abbruch "go build scheiterte"
ok "littera-altbestand, restore-backup, nachtsicherung"

uebernahme() { (cd "$ARBEIT" && ./littera-altbestand -csv export -personen -ausleihen "$@"); }
FINGERABDRUCK="SELECT (SELECT count(*) FROM buecher_titel) || ' ' || (SELECT count(*) FROM buecher_exemplare) || ' ' ||
	(SELECT count(*) FROM leser) || ' ' || (SELECT count(*) FROM benutzer) || ' ' || (SELECT count(*) FROM ausleihen)"

schritt "Trockenlauf mit -personen -ausleihen"
code=0; uebernahme -trocken >"$ARBEIT/trocken.out" 2>&1 || code=$?
zeige_lauf "$ARBEIT/trocken.out"
if [ "$code" = 1 ]; then
	echo "  → Lesergruppen ohne Zuordnung. Der echte Lauf muss anhalten, bevor er etwas schreibt:"
	vorher="$(sql "$FINGERABDRUCK")"
	code=0; uebernahme -db "$DB/bibliothek?sslmode=disable" >"$ARBEIT/halt.out" 2>&1 || code=$?
	pruefe "$code" 1 "echter Lauf hält an"
	pruefe "$(sql "$FINGERABDRUCK")" "$vorher" "nichts geschrieben (Titel, Exemplare, Leser, Konten, Ausleihen)"
	rm -f "$ARBEIT/littera_import.log"
	[ ${#GRUPPEN[@]} -gt 0 ] || abbruch "Gruppen in Littera zuordnen, oder die Zuordnung mit --gruppe nachstellen"
	python3 "$HIER/probe_host.py" zuordnen "$ARBEIT/export" "${GRUPPEN[@]}"
	code=0; uebernahme -trocken >"$ARBEIT/trocken2.out" 2>&1 || code=$?
	pruefe "$code" 0 "Trockenlauf nach der Zuordnung"
elif [ "$code" != 0 ]; then
	abbruch "Trockenlauf endete mit $code"
elif [ ${#GRUPPEN[@]} -gt 0 ]; then
	echo "  --gruppe wird nicht gebraucht: Jede Gruppe ist zugeordnet."
fi

schritt "Übernahme"
start=$SECONDS
code=0; uebernahme -db "$DB/bibliothek?sslmode=disable" >"$ARBEIT/lauf.out" 2>&1 || code=$?
zeige_lauf "$ARBEIT/lauf.out"
echo "  Dauer $((SECONDS - start)) s, Rückgabe $code (0 vollständig, 2 mit FEHLER-Zeilen)"
[ "$code" != 1 ] || abbruch "Übernahme abgebrochen"
if grep -q "ABGLEICH FEHLGESCHLAGEN" "$ARBEIT/lauf.out"; then abweichung "Abgleich an der Datenbank"; else ok "alle Abgleiche an der Datenbank"; fi
pruefe "$(sed -nE 's/.*nicht übernommen ([0-9]+)$/\1/p' "$ARBEIT/lauf.out" | tail -1)" 0 "jeder Leser übernommen"
pruefe "$(sed -nE 's/.*ohne Entleiher ([0-9]+),.*/\1/p' "$ARBEIT/lauf.out" | tail -1)" 0 "keine Ausleihe ohne Entleiher"
# Litteras Nichtsortierzeichen („¬Die¬ schwarze Katze") gehören weder zum Titel noch zum Namen.
pruefe "$(sql "SELECT count(*) FROM buecher_titel WHERE concat(titel, untertitel, autor) LIKE '%¬%'")" 0 \
	"kein Titel trägt ein Nichtsortierzeichen (im Export: $(grep -c '¬' "$ARBEIT/export/titel.csv" || true) Zeilen)"
# Titeltexte stehen mit einem Leerzeichen zwischen den Wörtern (Migration 160); Littera führt
# Titel mit zwei Leerzeichen in Folge und mit geschütztem Leerzeichen.
pruefe "$(sql "SELECT count(*) FROM buecher_titel WHERE concat_ws('|', titel, untertitel, autor, verlag) ~ ('  |' || chr(160))")" 0 \
	"kein Titel trägt Leerraum in Folge (im Export: $(python3 "$HIER/probe_host.py" leerraum "$ARBEIT/export") Titel)"
# Die Auflage unterscheidet zwei Ausgaben desselben Buchs; verglichen wird je Titel.
sql "SELECT json_build_object('id', erweiterte_eigenschaften->>'littera_id', 'auflage', coalesce(auflage, ''))
	FROM buecher_titel" >"$ARBEIT/auflagen.txt"
auflagen="$(python3 "$HIER/probe_host.py" auflage "$ARBEIT/export" "$ARBEIT/auflagen.txt")" || abbruch "Vergleich der Auflagen scheiterte"
read -r abweichend verglichen im_export <<<"$auflagen"
pruefe "$abweichend von $verglichen" "0 von $(sql 'SELECT count(*) FROM buecher_titel')" \
	"die Auflage steht an jedem Titel wie in Littera (im Export: $im_export Titel mit Auflage)"
echo "  Protokoll nach Grund (ohne Werte):"
python3 "$HIER/probe_host.py" protokoll "$ARBEIT/littera_import.log"
echo "  In der Datenbank: $(sql "SELECT count(*) || ' Titel, ' || (SELECT count(*) FROM buecher_exemplare) || ' Exemplare, ' ||
	(SELECT count(*) FROM leser WHERE art = 'schueler') || ' Schüler, ' || (SELECT count(*) FROM leser WHERE art <> 'schueler') ||
	' im Kollegium, ' || (SELECT count(*) FROM ausleihen) || ' Ausleihen, davon offen ' ||
	(SELECT count(*) FROM ausleihen WHERE rueckgabe_am IS NULL) FROM buecher_titel")"

schritt "Theke (über das Netz $NETZ, wie die Oberfläche)"
mkdir "$ARBEIT/probe"
sql "SELECT e.erweiterte_eigenschaften->>'littera_exemplarnr' FROM buecher_exemplare e
	WHERE EXISTS (SELECT 1 FROM ausleihen a WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL)" >"$ARBEIT/verliehen.txt"
python3 "$HIER/probe_host.py" stichprobe "$ARBEIT/export" "$ARBEIT/verliehen.txt" "$ARBEIT/probe/stichprobe.json"
sql "SELECT json_build_object('id', l.id, 'ausweis', l.barcode_id) FROM leser l
	WHERE l.art = 'schueler' AND NOT l.ist_gesperrt AND NOT l.ist_abgaenger AND l.deleted_at IS NULL
	  AND l.barcode_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM ausleihen a WHERE a.schueler_id = l.id AND a.rueckgabe_am IS NULL)
	ORDER BY l.barcode_id LIMIT 1" >"$ARBEIT/probe/leser.json"
cp "$HIER/theke.py" "$ARBEIT/probe/"
code=0; docker run --rm --network "$NETZ" -v "$ARBEIT/probe:/probe" python:3.13-alpine python /probe/theke.py || code=$?
pruefe "$code" 0 "Theke ohne Abweichung"
wert() { python3 -c "import json,sys; d=json.load(open('$ARBEIT/probe/$1')); print($2)"; }
BUCH="$(wert stichprobe.json "d['frei'][-1]['scan']")" LESER="$(wert leser.json "d['id']")"
pruefe "$(psql -X -q -tA "$DB/bibliothek" -v buch="$BUCH" -v leser="$LESER" <<'SQL'
SELECT count(*) FROM ausleihen a JOIN buecher_exemplare e ON e.id = a.exemplar_id
WHERE e.barcode_id = :'buch' AND a.schueler_id = :'leser' AND a.rueckgabe_am IS NOT NULL
SQL
)" 1 "Ausleihe und Rückgabe stehen in der Datenbank"
ALT="$(wert stichprobe.json "(d['verliehen'] or {}).get('scan', '')")"
if [ -n "$ALT" ]; then
	pruefe "$(psql -X -q -tA "$DB/bibliothek" -v buch="$ALT" <<'SQL'
SELECT count(*) FROM ausleihen a JOIN buecher_exemplare e ON e.id = a.exemplar_id
WHERE e.barcode_id = :'buch' AND a.rueckgabe_am IS NULL
SQL
)" 0 "das in Littera verliehene Buch ist zurück"
fi

schritt "Nachtsicherung (Code des Jobs, ohne S3 und Mail)"
mkdir "$ARBEIT/sicherung"
env -i PATH="$PATH" DATABASE_URL="$DB/bibliothek?sslmode=disable" BACKUP_DIR="$ARBEIT/sicherung" \
	BACKUP_ENCRYPTION_KEY="$SCHLUESSEL" "$ARBEIT/nachtsicherung" >"$ARBEIT/sicherung.out" 2>&1 || true
ENC="$(ls -t "$ARBEIT"/sicherung/backup_*.sql.gz.enc 2>/dev/null | head -1)"
grep -q "Backup: completed successfully" "$ARBEIT/sicherung.out" && [ -n "$ENC" ] ||
	abbruch "Nachtsicherung ohne Datei (sicherung.out)"
ok "Sicherung $(basename "$ENC"), $(du -h "$ENC" | cut -f1)"

schritt "Wiederherstellung nach resilience_and_recovery.md 2a, in eine Wegwerf-Datenbank"
DUMP="$ARBEIT/wiederherstellung.sql"
BACKUP_ENCRYPTION_KEY="$SCHLUESSEL" "$ARBEIT/restore-backup" "$ENC" "$DUMP" >"$ARBEIT/restore.out" 2>&1 ||
	abbruch "restore-backup scheiterte (restore.out)"
sql "DROP DATABASE IF EXISTS bibliothek_restore" postgres
sql "CREATE DATABASE bibliothek_restore" postgres
code=0; psql -X -q -v ON_ERROR_STOP=1 "$DB/bibliothek_restore" -f "$DUMP" >"$ARBEIT/einspielen.out" 2>&1 || code=$?
pruefe "$code" 0 "Einspielen mit ON_ERROR_STOP"
# Schritt 7 der Anleitung, wörtlich: Zeilen je Tabelle im Dump gegen die Datenbank.
for t in leser buecher_titel buecher_exemplare ausleihen audit_logs; do
	im_dump=$(awk -v kopf="COPY public.$t " 'index($0, kopf) == 1 {f = 1; next} /^\\\.$/ {f = 0} f' "$DUMP" | wc -l | tr -d ' ')
	pruefe "$(sql "SELECT count(*) FROM $t" bibliothek_restore) $(sql "SELECT count(*) FROM $t")" "$im_dump $im_dump" \
		"$t: wiederhergestellt / vorher = Dump ($im_dump)"
done
aufbau="SELECT (SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public') || ' ' ||
	(SELECT count(*) FROM pg_indexes WHERE schemaname = 'public') || ' ' || (SELECT count(*) FROM pg_trigger WHERE NOT tgisinternal)"
pruefe "$(sql "$aufbau" bibliothek_restore)" "$(sql "$aufbau")" "Tabellen, Indizes, Trigger gleich"
docker run -d --rm --name probe-restore-backend --network "$NETZ" -e PORT=8087 -e APP_ENV=local \
	-e "DATABASE_URL=postgres://postgres:probe-nur-lokal@bibliothek-db-probe:5432/bibliothek_restore?sslmode=disable" \
	-e JWT_SECRET=probe-jwt-secret-nur-lokal-mindestens-32-zeichen -e 'APP_ENCRYPTION_KEY=probe-aes-key-32-zeichen-lokal!!' \
	-e COOKIE_SECURE=false -e IMAP_HOST=mock -e SMTP_HOST=127.0.0.1 -e SMTP_PORT=9 bibliothek-probe:latest >/dev/null
for _ in $(seq 1 30); do
	[ "$(docker exec bibliothek-werkzeug-probe curl -s -o /dev/null -w '%{http_code}' http://probe-restore-backend:8087/health || true)" = 200 ] && break
	sleep 2
done
pruefe "$(docker exec bibliothek-werkzeug-probe curl -s -o /dev/null -w '%{http_code}' http://probe-restore-backend:8087/health || true)" 200 \
	"das Programm startet auf der wiederhergestellten Datenbank"

schritt "Ergebnis (Stand $STAND)"
if [ ${#ABWEICHUNGEN[@]} -eq 0 ]; then
	echo "  Jede Prüfung bestanden."
	exit 0
fi
printf '  %s Abweichung(en):\n' "${#ABWEICHUNGEN[@]}"
printf '    - %s\n' "${ABWEICHUNGEN[@]}"
exit 1
