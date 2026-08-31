#!/usr/bin/env bash
# ==============================================================================
# update.sh — Zero-Data-Loss Update Script for Schulbibliothek
# ==============================================================================
# Ablauf:
#   1. pg_dump Backup mit Zeitstempel → ./backups/
#   2. git pull (neuesten Code holen)
#   3. docker compose up -d --build (rebuild & restart)
#   4. Bei Fehler: Abbruch + Rollback-Anleitung
#   5. Nach erfolgreichem Deploy: Vorab-Sicherung verschlüsseln, Klartext löschen
#   6. Backups älter als 30 Tage werden automatisch gelöscht
#   7. Docker-Build-Cache älter als 7 Tage wird aufgeräumt
# ==============================================================================
set -euo pipefail

# ── Die Klammer um alles ───────────────────────────────────────────────────────
#
# Dieses Skript zieht in Schritt 2 per `git pull` neuen Code — und ueberschreibt dabei
# SICH SELBST, waehrend es laeuft. Bash liest ein Skript nicht auf einmal ein, sondern
# haeppchenweise und merkt sich dabei eine Position in der DATEI. Wird die Datei
# unterwegs laenger oder kuerzer, zeigt diese Position anschliessend irgendwohin, und
# bash fuehrt Bruchstuecke aus oder bricht mit einem Syntaxfehler ab — mitten im Deploy,
# nach dem Backup, vor dem Health-Check.
#
# Am 11.08.2026 nachgestellt: Ein flaches Skript, das sich waehrend des Laufs um drei
# Zeilen verlaengert, stirbt mit "unexpected EOF while looking for matching quote" und
# fuehrt seine letzten Schritte nicht mehr aus. Dieselbe Datei in { } gefasst laeuft
# vollstaendig durch.
#
# Der Grund: `{ ... }` ist EIN zusammengesetzter Befehl. Bash parst ihn komplett, bevor
# es die erste Zeile darin ausfuehrt — ab da ist die Datei auf der Platte gleichgueltig.
# Das `exit` steht bewusst INNERHALB der Klammer: Stuende es dahinter, kehrte bash nach
# dem Block zum Lesen der (inzwischen veraenderten) Datei zurueck und straeuchelte
# genau dort. Auch das nachgestellt.
#
# Sichtbar wurde das Problem harmlos: Der Deploy vom 11.08. lief noch mit der ALTEN
# Fassung dieses Skripts — die neue kam ja erst mit dem pull —, deshalb blieb
# GIT_COMMIT im Image leer. Ein Deploy-Skript aendert sich immer erst fuer den
# NAECHSTEN Lauf. Genau dafuer meldet Schritt 4b ein Image ohne Commit als Warnung und
# nicht als Fehler.
{

# ── Konfiguration ─────────────────────────────────────────────────────────────
COMPOSE_FILE="$(cd "$(dirname "$0")" && pwd)/docker-compose.yml"
BACKUP_DIR="$(cd "$(dirname "$0")" && pwd)/backups"
BACKUP_RETENTION_DAYS=30
# Klartext-Dumps bekommen die kurze Lunte (Befund A5): Nach einem erfolgreichen Deploy
# wird die Vorab-Sicherung in Schritt 5 ohnehin verschlüsselt. Was hier länger als zwei
# Tage liegen bleibt, stammt aus einem FEHLGESCHLAGENEN Deploy — dann ist die Datei
# gebraucht worden oder war nie nötig, aber in keinem Fall soll sie einen Monat lang
# jeden Schülernamen im Klartext auf der Platte halten.
KLARTEXT_RETENTION_DAYS=2
# Build-Cache-Schichten, die seit dieser Zeit niemand mehr angefasst hat, fliegen raus.
# 168h = 7 Tage — lang genug, dass der Cache des letzten Updates erhalten bleibt.
BUILD_CACHE_RETENTION_HOURS=168

DB_CONTAINER="bibliothek-db"
DB_USER="postgres"
DB_NAME="bibliothek"
# EIN Name für den Backend-Container. Er stand hier bis zum 23.08.2026 erst in Schritt 4
# (APP_CONTAINER) — als Schritt 5 dazukam, hätte ein zweiter Platz für denselben Wert
# schlicht darauf gewartet, auseinanderzulaufen. scripts/backup_krypto.sh liest ihn als
# BACKEND_CONTAINER, deshalb wird er hier gesetzt, BEVOR der Helfer eingebunden wird.
APP_CONTAINER="bibliothek-backend"
BACKEND_CONTAINER="${APP_CONTAINER}"

# Verschlüsselungs-Helfer (scripts/backup_krypto.sh, docs/resilience_and_recovery.md 1b).
#
# Bewusst NICHT hart eingebunden: Mit `set -e` bräche ein `source` auf eine fehlende
# Datei den ganzen Deploy ab — wegen eines Helfers, der nur das Aufräumen betrifft.
# Fehlt er, sagt krypto_moeglich das in Schritt 5, und die Sicherung bleibt im Klartext
# liegen (mit der kurzen Frist unten).
KRYPTO_HELFER="$(cd "$(dirname "$0")" && pwd)/scripts/backup_krypto.sh"
if [ -f "${KRYPTO_HELFER}" ]; then
    # shellcheck source=scripts/backup_krypto.sh
    source "${KRYPTO_HELFER}"
else
    krypto_moeglich() { KRYPTO_GRUND="scripts/backup_krypto.sh fehlt"; return 1; }
fi

TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
BACKUP_FILE="${BACKUP_DIR}/backup_${TIMESTAMP}.sql.gz"

# ── Farben für Ausgabe ────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

log_info()    { echo -e "${BLUE}[INFO]${NC}  $*"; }
log_ok()      { echo -e "${GREEN}[OK]${NC}    $*"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $*" >&2; }
log_step()    { echo -e "\n${BOLD}══════════════════════════════════════════════════${NC}"; echo -e "${BOLD}  $*${NC}"; echo -e "${BOLD}══════════════════════════════════════════════════${NC}"; }

# ── Rollback-Anleitung ausgeben ───────────────────────────────────────────────
print_rollback_instructions() {
    echo ""
    log_error "╔══════════════════════════════════════════════════════════════╗"
    log_error "║           UPDATE FEHLGESCHLAGEN — ROLLBACK-ANLEITUNG        ║"
    log_error "╚══════════════════════════════════════════════════════════════╝"
    echo ""
    echo -e "${YELLOW}Das letzte erfolgreiche Backup:${NC}"
    echo -e "  ${BOLD}${BACKUP_FILE}${NC}"
    echo ""
    echo -e "${YELLOW}Schritte zum manuellen Rollback:${NC}"
    echo ""
    echo -e "  ${BOLD}1. Container stoppen:${NC}"
    echo "     docker compose down"
    echo ""
    echo -e "  ${BOLD}2. Altes Image zurücksetzen (falls Git-Pull durchgeführt):${NC}"
    echo "     git stash  # oder: git reset --hard HEAD@{1}"
    echo ""
    echo -e "  ${BOLD}3. Backup einspielen:${NC}"
    echo "     # DB-Container starten:"
    echo "     docker compose up -d postgres-db"
    echo "     # Warten bis DB bereit ist:"
    echo "     docker compose exec postgres-db pg_isready -U ${DB_USER} -d ${DB_NAME}"
    echo "     # Backup einspielen:"
    echo "     gunzip -c \"${BACKUP_FILE}\" | docker exec -i ${DB_CONTAINER} psql -U ${DB_USER} -d ${DB_NAME}"
    echo ""
    echo -e "  ${BOLD}4. App wieder starten:${NC}"
    echo "     docker compose up -d --build"
    echo ""
    echo -e "  ${BOLD}5. Danach: die Vorab-Sicherung entsorgen${NC}"
    echo "     # Sie ist UNVERSCHLÜSSELT und enthält jeden Schülernamen, jede Adresse,"
    echo "     # jede Ausleihe im Klartext. Nach geglücktem Rollback löschen:"
    echo "     shred -u \"${BACKUP_FILE}\"   # oder: rm -f"
    echo "     # Andernfalls wird sie nach ${KLARTEXT_RETENTION_DAYS} Tagen automatisch gelöscht."
    echo ""
    echo -e "${YELLOW}Alle verfügbaren Backups:${NC}"
    ls -lh "${BACKUP_DIR}"/*.sql.gz "${BACKUP_DIR}"/*.sql.gz.enc 2>/dev/null || echo "  (keine Backups gefunden)"
    echo ""
    echo -e "${YELLOW}Verschlüsselte Sicherungen (.enc) zuerst öffnen:${NC}"
    echo "     docker compose exec backend ./restore-backup <datei>.sql.gz.enc /tmp/dump.sql"
    echo ""
}

# ── Schritt 0: Voraussetzungen prüfen ────────────────────────────────────────
log_step "Schritt 0: Voraussetzungen prüfen"

if ! command -v docker &>/dev/null; then
    log_error "Docker ist nicht installiert oder nicht im PATH."
    exit 1
fi

if ! docker info &>/dev/null; then
    log_error "Docker-Daemon ist nicht erreichbar. Bitte starten: sudo systemctl start docker"
    exit 1
fi

if ! docker ps --filter "name=${DB_CONTAINER}" --filter "status=running" --format '{{.Names}}' | grep -q "${DB_CONTAINER}"; then
    log_warn "DB-Container '${DB_CONTAINER}' läuft nicht. Backup wird übersprungen."
    SKIP_BACKUP=true
else
    SKIP_BACKUP=false
fi

mkdir -p "${BACKUP_DIR}"
log_ok "Voraussetzungen erfüllt."

# ── Schritt 1: Datenbank-Backup ───────────────────────────────────────────────
log_step "Schritt 1: Datenbank-Backup erstellen"

if [ "${SKIP_BACKUP}" = "true" ]; then
    log_warn "Backup übersprungen (DB-Container läuft nicht)."
else
    log_info "Erstelle Backup → ${BACKUP_FILE}"

    # Diese eine Datei entsteht bewusst im KLARTEXT — und wird in Schritt 5 wieder
    # verschlüsselt.
    #
    # Sie ist der Rückweg für genau das Zeitfenster, in dem alles schiefgehen kann:
    # zwischen `docker compose up` und einem gesunden Container. Verschlüsselte man sie
    # schon hier, hinge der Rollback an einem Werkzeug, das im selben Container liegt,
    # der gerade nicht startet — und an einer Passphrase, deren Verlust man erst im
    # Ernstfall bemerkt. Der dokumentierte Weg `gunzip | psql`
    # (print_rollback_instructions, docs/resilience_and_recovery.md 2b) bleibt damit
    # unangetastet.
    #
    # Ist der Deploy gesund, verliert sie ihren Zweck: Schritt 5 verschlüsselt sie über
    # denselben Weg wie der nächtliche Job und löscht den Klartext. Offen liegt sie damit
    # nur noch für die Dauer eines Deploys statt 30 Tage (Befund A5).
    #
    # umask im Subshell, NICHT chmod danach: Mit einem nachgereichten chmod läge die
    # Datei zwischen Anlegen und Rechtesetzen für alle lesbar auf der Platte, und genau
    # in dieser Zeit wird sie beschrieben.
    if (umask 077; docker exec "${DB_CONTAINER}" pg_dump -U "${DB_USER}" -d "${DB_NAME}" --no-password \
        | gzip > "${BACKUP_FILE}"); then

        BACKUP_SIZE="$(du -sh "${BACKUP_FILE}" | cut -f1)"
        log_ok "Backup erfolgreich: ${BACKUP_FILE} (${BACKUP_SIZE})"
    else
        log_error "pg_dump ist fehlgeschlagen! Update wird abgebrochen."
        rm -f "${BACKUP_FILE}"
        exit 1
    fi
fi

# ── Schritt 2: Neuesten Code holen ────────────────────────────────────────────
log_step "Schritt 2: Code aktualisieren (git pull)"

if [ -d "$(dirname "$0")/.git" ]; then
    cd "$(dirname "$0")"
    log_info "Führe git pull aus..."
    if ! git pull; then
        log_error "git pull fehlgeschlagen!"
        print_rollback_instructions
        exit 1
    fi
    log_ok "Code aktualisiert."
else
    log_warn ".git-Verzeichnis nicht gefunden — git pull übersprungen."
    log_warn "Bitte Code manuell aktualisieren, bevor du dieses Skript ausführst."
fi

# ── Schritt 3: Container neu bauen und starten ────────────────────────────────
log_step "Schritt 3: Docker-Container neu bauen und starten"

log_info "Führe docker compose up -d --build aus..."

# Der Commit wandert als Build-Argument ins Image (Dockerfile: ARG/ENV GIT_COMMIT) und
# wird in Schritt 4b dagegen geprüft. Ohne .git bleibt er leer — dann sagt die Prüfung
# "unbekannt", statt einen falschen Stand zu behaupten.
GIT_COMMIT="$(git rev-parse HEAD 2>/dev/null || echo '')"
export GIT_COMMIT

if docker compose -f "${COMPOSE_FILE}" up -d --build; then
    log_ok "Container erfolgreich neu gestartet."
else
    log_error "docker compose up ist fehlgeschlagen!"
    print_rollback_instructions
    exit 1
fi

# ── Schritt 4: Health-Check ───────────────────────────────────────────────────
log_step "Schritt 4: Warte auf Gesundheitsprüfung"

HEALTH_TIMEOUT=120

# Zwei Quellen, weil keine für sich allein genügt:
#
#  1. Der Docker-Healthcheck. Die genauere Auskunft — er läuft im Container —, aber er
#     braucht Anlauf, und ein Container ohne Healthcheck liefert hier gar nichts.
#  2. Die Anwendung selbst über /health. Sie beantwortet die Frage, auf die es ankommt:
#     Kommt jemand rein?
#
# Am 06.08.2026 hat dieser Schritt zwei einwandfreie Updates hintereinander als
# FEHLGESCHLAGEN gemeldet — samt Rollback-Anleitung —, während die Anwendung längst lief
# und /health von aussen mit 200 antwortete. Die Prüfung hing allein an Quelle 1, und der
# Hinweistext nannte obendrein einen Container ("bibliothek-web"), den es nicht gibt: Wer
# der Anleitung folgte, bekam eine leere Ausgabe und stand vor einem Rollback, den niemand
# brauchte. Ein Alarm, der bei jedem gesunden Lauf losgeht, erzieht nur dazu, ihn zu
# überhören — und dann fehlt er, wenn er zählt.
docker_health() {
    local status
    status="$(docker inspect --format='{{if .State.Health}}{{.State.Health.Status}}{{else}}ohne-healthcheck{{end}}' \
        "${APP_CONTAINER}" 2>/dev/null | head -1)"
    echo "${status:-unbekannt}"
}

# HTTP-Gegenprobe im Container: Der Port steht in dessen eigener Umgebung, damit hier
# keine zweite Wahrheit über die Portnummer entsteht.
app_antwortet() {
    local port
    port="$(docker inspect --format='{{range .Config.Env}}{{println .}}{{end}}' "${APP_CONTAINER}" 2>/dev/null |
        sed -n 's/^PORT=//p' | head -1)"
    [ -n "${port}" ] || return 1
    docker exec "${APP_CONTAINER}" wget --no-verbose --tries=1 --spider \
        "http://127.0.0.1:${port}/health" >/dev/null 2>&1
}

log_info "Warte auf die Anwendung (max. ${HEALTH_TIMEOUT} Sekunden)..."
WAIT=0
while true; do
    STATUS="$(docker_health)"
    if [ "${STATUS}" = "healthy" ]; then
        log_ok "Anwendung ist healthy und läuft."
        break
    fi
    if app_antwortet; then
        log_ok "Anwendung antwortet auf /health (Docker-Status: ${STATUS})."
        break
    fi

    sleep 3
    WAIT=$((WAIT + 3))
    if [ ${WAIT} -ge ${HEALTH_TIMEOUT} ]; then
        log_error "Anwendung meldet sich nach ${HEALTH_TIMEOUT}s weder als healthy noch über /health."
        log_error "Zustand:  docker inspect --format='{{json .State.Health}}' ${APP_CONTAINER}"
        log_error "Logs:     docker logs ${APP_CONTAINER} --tail 50"
        echo ""
        docker logs "${APP_CONTAINER}" --tail 20 2>&1 | sed 's/^/    /'
        print_rollback_instructions
        exit 1
    fi
    log_info "  ... noch ${WAIT}s gewartet (Docker-Status: ${STATUS})"
done

# ── Schritt 4b: Läuft auch WIRKLICH der neue Stand? ──────────────────────────
#
# Gesund heisst nicht aktuell. Am 11.08.2026 stand auf diesem Server `git pull` durch,
# `./update.sh` aber nicht: Das Arbeitsverzeichnis zeigte den neuen Commit, das laufende
# Image war zehn Stunden alt und lieferte den Stand von vorgestern aus. Beide Prüfungen
# aus Schritt 4 meldeten dabei völlig zu Recht "gesund" — sie beantworten eine andere
# Frage. Aufgefallen ist es nur, weil jemand von Hand den Namen der ausgelieferten
# Bundle-Datei verglichen hat.
#
# Diese Prüfung stellt die richtige Frage: Läuft der Container, der aus DEM Commit
# gebaut wurde, der gerade im Arbeitsverzeichnis liegt?
log_step "Schritt 4b: Läuft der Container aus diesem Commit?"

ERWARTET="$(git rev-parse HEAD 2>/dev/null || echo '')"
IM_IMAGE="$(docker exec "${APP_CONTAINER}" printenv GIT_COMMIT 2>/dev/null || echo '')"

if [ -z "${ERWARTET}" ]; then
    log_warn "Kein git-Arbeitsverzeichnis — Abgleich nicht möglich."
elif [ -z "${IM_IMAGE}" ]; then
    # Seit dem Deploy vom 11.08.2026 trägt jedes regulär gebaute Image seinen Commit.
    # Ein leerer Wert heißt: von Hand ohne GIT_COMMIT gebaut (oder ein Stand von vor
    # dem 11.08.) — genau der stille „alter Container"-Fall, den dieser Schritt fangen
    # soll (Prüfung 22.08.2026). Früher nur eine Warnung, die das Skript grün beendete.
    log_error "Das Image trägt keinen Commit (GIT_COMMIT leer) — es wurde nicht über update.sh gebaut."
    log_error "Erneut bauen mit:  GIT_COMMIT=\$(git rev-parse HEAD) docker compose -f \"${COMPOSE_FILE}\" build --no-cache && docker compose -f \"${COMPOSE_FILE}\" up -d"
    exit 1
elif [ "${IM_IMAGE}" = "${ERWARTET}" ]; then
    log_ok "Container läuft aus ${ERWARTET:0:7} — Arbeitsverzeichnis und Image stimmen überein."
else
    log_error "Der Container läuft NICHT aus dem aktuellen Stand."
    log_error "  Arbeitsverzeichnis: ${ERWARTET:0:7}"
    log_error "  laufendes Image:    ${IM_IMAGE:0:7}"
    log_error "Der Build hat den neuen Stand nicht übernommen. Die Anwendung ist erreichbar"
    log_error "und gesund — sie zeigt nur etwas anderes, als hier im Verzeichnis liegt."
    log_error "Erneut versuchen mit:  docker compose -f \"${COMPOSE_FILE}\" build --no-cache && docker compose -f \"${COMPOSE_FILE}\" up -d"
    exit 1
fi

# ── Schritt 5: Vorab-Sicherung verschlüsseln ──────────────────────────────────
#
# Erst HIER, nicht in Schritt 1: Ab dieser Zeile ist bewiesen, dass der neue Container
# läuft, gesund ist und aus diesem Commit stammt (Schritt 4/4b). Damit ist der Rollback
# vom Tisch, die Klartext-Datei hat ihren Zweck erfüllt — und der Container, der das
# Verschlüsselungswerkzeug trägt, ist nachweislich da.
#
# Die Reihenfolge in verschluessele_datei ist der Kern: verschlüsseln, Ergebnis prüfen,
# ERST DANN den Klartext löschen. Andersherum stünde am Ende eines halb geglückten
# Laufs gar keine Sicherung mehr.
log_step "Schritt 5: Vorab-Sicherung verschlüsseln"

if [ "${SKIP_BACKUP}" = "true" ]; then
    log_info "Keine Vorab-Sicherung vorhanden (Schritt 1 übersprungen)."
elif [ ! -f "${BACKUP_FILE}" ]; then
    log_warn "Vorab-Sicherung ${BACKUP_FILE} nicht mehr gefunden — übersprungen."
elif krypto_moeglich; then
    log_info "Verschlüssele ${BACKUP_FILE} …"
    if verschluessele_datei "${BACKUP_FILE}"; then
        log_ok "Verschlüsselt: ${KRYPTO_ZIEL} (Klartext gelöscht)."
        log_info "Öffnen mit: docker compose exec backend ./restore-backup <datei> /tmp/dump.sql"
    else
        log_warn "Verschlüsselung fehlgeschlagen — die Sicherung bleibt im KLARTEXT liegen:"
        log_warn "  ${BACKUP_FILE}"
        log_warn "  Sie enthält jeden Schülernamen und jede Adresse im Klartext und wird"
        log_warn "  in ${KLARTEXT_RETENTION_DAYS} Tagen automatisch gelöscht."
    fi
else
    log_warn "Verschlüsselung nicht möglich (${KRYPTO_GRUND})."
    log_warn "Die Vorab-Sicherung bleibt im KLARTEXT liegen: ${BACKUP_FILE}"
    log_warn "Sie wird in ${KLARTEXT_RETENTION_DAYS} Tagen automatisch gelöscht."
fi

# ── Schritt 6: Alte Backups aufräumen ─────────────────────────────────────────
log_step "Schritt 6: Alte Backups aufräumen (verschlüsselt: ${BACKUP_RETENTION_DAYS} Tage, Klartext: ${KLARTEXT_RETENTION_DAYS} Tage)"

# Zwei Fristen, zwei Muster. `backup_*.sql.gz` trifft NICHT die `.enc`-Dateien — deren
# Name endet auf `.enc`, und `find -name` vergleicht den ganzen Namen.
DELETED=$(find "${BACKUP_DIR}" -name "backup_*.sql.gz.enc" -mtime "+${BACKUP_RETENTION_DAYS}" -print -delete 2>/dev/null | wc -l | tr -d ' ')
DELETED_KLAR=$(find "${BACKUP_DIR}" -name "backup_*.sql.gz" -mtime "+${KLARTEXT_RETENTION_DAYS}" -print -delete 2>/dev/null | wc -l | tr -d ' ')

if [ "${DELETED}" -gt 0 ] || [ "${DELETED_KLAR}" -gt 0 ]; then
    log_ok "${DELETED} verschlüsselte/s und ${DELETED_KLAR} unverschlüsselte/s Backup/s gelöscht."
else
    log_info "Keine alten Backups zum Löschen gefunden."
fi

REMAINING=$(find "${BACKUP_DIR}" -name "backup_*.sql.gz.enc" 2>/dev/null | wc -l | tr -d ' ')
REMAINING_KLAR=$(find "${BACKUP_DIR}" -name "backup_*.sql.gz" 2>/dev/null | wc -l | tr -d ' ')
log_info "${REMAINING} verschlüsselte/s Backup/s verbleiben in ${BACKUP_DIR}/"
if [ "${REMAINING_KLAR}" -gt 0 ]; then
    log_warn "${REMAINING_KLAR} UNVERSCHLÜSSELTE/S Backup/s liegt/liegen dort ebenfalls (Klarnamen im Klartext)."
fi

# ── Schritt 7: Docker-Build-Cache aufräumen ───────────────────────────────────
log_step "Schritt 7: Docker-Build-Cache aufräumen (älter als ${BUILD_CACHE_RETENTION_HOURS}h)"

# Jedes Update baut ein neues Image, und der BuildKit-Cache wächst dabei unbegrenzt
# weiter — niemand räumt ihn von allein ab. Am 09.08.2026 stand er bei 14,47 GB und
# füllte die 38-GB-Platte zu 82 %. Läuft sie voll, steht der ganze Server, samt der
# fremden Dienste hinter demselben Caddy.
#
# Der Zeitfilter ist Absicht: Der Cache des GERADE gebauten Images bleibt liegen, das
# nächste Update ist also weiterhin schnell. Nur Schichten, die seit einer Woche
# niemand angefasst hat, fallen weg.
#
# Bewusst NICHT `docker system prune` und erst recht kein `docker volume prune`: Auf
# diesem Server liegen fremde benannte Volumes (school-calendar, inkl. deren Postgres
# und Backups), die Docker als "unbenutzt" führt. Ein Volume-Prune löschte sie.
FREI_VORHER="$(df -Pk / | awk 'NR==2 {print $4}')"

# Ein fehlgeschlagenes Aufräumen darf ein geglücktes Update NICHT zum Fehler machen —
# deshalb hier keine Abbruchbedingung, sondern nur eine Warnung.
if docker builder prune -af --filter "until=${BUILD_CACHE_RETENTION_HOURS}h" >/dev/null 2>&1; then
    FREI_NACHHER="$(df -Pk / | awk 'NR==2 {print $4}')"
    BEFREIT_MB=$(( (FREI_NACHHER - FREI_VORHER) / 1024 ))
    if [ "${BEFREIT_MB}" -gt 0 ]; then
        log_ok "Build-Cache aufgeräumt: ${BEFREIT_MB} MB frei geworden."
    else
        log_info "Build-Cache aufgeräumt: nichts Altes vorhanden."
    fi
else
    log_warn "Build-Cache konnte nicht aufgeräumt werden — das Update selbst ist davon unberührt."
fi
log_info "Speicher auf /: $(df -h / | awk 'NR==2 {print $4}') frei ($(df -h / | awk 'NR==2 {print $5}') belegt)"

# ── Fertig ────────────────────────────────────────────────────────────────────
echo ""
log_ok "══════════════════════════════════════════════════"
log_ok "  UPDATE ERFOLGREICH ABGESCHLOSSEN"
log_ok "══════════════════════════════════════════════════"
echo ""
log_info "Anwendung läuft lokal auf Port 8083 und ist erreichbar unter: https://flasch3.herzog-dupont.de"
echo ""

exit 0
}
