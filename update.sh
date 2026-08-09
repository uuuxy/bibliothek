#!/usr/bin/env bash
# ==============================================================================
# update.sh — Zero-Data-Loss Update Script for Schulbibliothek
# ==============================================================================
# Ablauf:
#   1. pg_dump Backup mit Zeitstempel → ./backups/
#   2. git pull (neuesten Code holen)
#   3. docker compose up -d --build (rebuild & restart)
#   4. Bei Fehler: Abbruch + Rollback-Anleitung
#   5. Backups älter als 30 Tage werden automatisch gelöscht
#   6. Docker-Build-Cache älter als 7 Tage wird aufgeräumt
# ==============================================================================
set -euo pipefail

# ── Konfiguration ─────────────────────────────────────────────────────────────
COMPOSE_FILE="$(cd "$(dirname "$0")" && pwd)/docker-compose.yml"
BACKUP_DIR="$(cd "$(dirname "$0")" && pwd)/backups"
BACKUP_RETENTION_DAYS=30
# Build-Cache-Schichten, die seit dieser Zeit niemand mehr angefasst hat, fliegen raus.
# 168h = 7 Tage — lang genug, dass der Cache des letzten Updates erhalten bleibt.
BUILD_CACHE_RETENTION_HOURS=168

DB_CONTAINER="bibliothek-db"
DB_USER="postgres"
DB_NAME="bibliothek"

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
    echo "     docker compose up -d db"
    echo "     # Warten bis DB bereit ist:"
    echo "     docker compose exec db pg_isready -U ${DB_USER} -d ${DB_NAME}"
    echo "     # Backup einspielen:"
    echo "     gunzip -c \"${BACKUP_FILE}\" | docker exec -i ${DB_CONTAINER} psql -U ${DB_USER} -d ${DB_NAME}"
    echo ""
    echo -e "  ${BOLD}4. App wieder starten:${NC}"
    echo "     docker compose up -d --build"
    echo ""
    echo -e "${YELLOW}Alle verfügbaren Backups:${NC}"
    ls -lh "${BACKUP_DIR}"/*.sql.gz 2>/dev/null || echo "  (keine Backups gefunden)"
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

    # umask im Subshell, NICHT chmod danach: Dieses Backup ist ein unverschlüsselter
    # Volldump — jeder Schülername, jede Adresse, jede Ausleihe im Klartext. Mit einem
    # nachgereichten chmod läge die Datei zwischen Anlegen und Rechtesetzen für alle
    # lesbar auf der Platte, und genau in dieser Zeit wird sie beschrieben.
    # Verschlüsseln wäre der stärkere Schritt, würde aber den dokumentierten
    # Rollback-Weg brechen (gunzip | psql, siehe print_rollback_instructions und
    # docs/resilience_and_recovery.md 2b) — das verschlüsselte Backup liefert der
    # nächtliche Job, siehe jobs/backup.go.
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

if docker compose -f "${COMPOSE_FILE}" up -d --build; then
    log_ok "Container erfolgreich neu gestartet."
else
    log_error "docker compose up ist fehlgeschlagen!"
    print_rollback_instructions
    exit 1
fi

# ── Schritt 4: Health-Check ───────────────────────────────────────────────────
log_step "Schritt 4: Warte auf Gesundheitsprüfung"

APP_CONTAINER="bibliothek-backend"
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

# ── Schritt 5: Alte Backups aufräumen ─────────────────────────────────────────
log_step "Schritt 5: Alte Backups aufräumen (älter als ${BACKUP_RETENTION_DAYS} Tage)"

DELETED=$(find "${BACKUP_DIR}" -name "backup_*.sql.gz" -mtime "+${BACKUP_RETENTION_DAYS}" -print -delete 2>/dev/null | wc -l | tr -d ' ')

if [ "${DELETED}" -gt 0 ]; then
    log_ok "${DELETED} altes Backup/s gelöscht."
else
    log_info "Keine alten Backups zum Löschen gefunden."
fi

REMAINING=$(find "${BACKUP_DIR}" -name "backup_*.sql.gz" 2>/dev/null | wc -l | tr -d ' ')
log_info "${REMAINING} Backup/s verbleiben in ${BACKUP_DIR}/"

# ── Schritt 6: Docker-Build-Cache aufräumen ───────────────────────────────────
log_step "Schritt 6: Docker-Build-Cache aufräumen (älter als ${BUILD_CACHE_RETENTION_HOURS}h)"

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
