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
# ==============================================================================
set -euo pipefail

# ── Konfiguration ─────────────────────────────────────────────────────────────
COMPOSE_FILE="$(cd "$(dirname "$0")" && pwd)/docker-compose.yml"
BACKUP_DIR="$(cd "$(dirname "$0")" && pwd)/backups"
BACKUP_RETENTION_DAYS=30

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

    if docker exec "${DB_CONTAINER}" pg_dump -U "${DB_USER}" -d "${DB_NAME}" --no-password \
        | gzip > "${BACKUP_FILE}"; then

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

log_info "Warte bis der Web-Container healthy ist (max. 60 Sekunden)..."
WAIT=0
until docker inspect --format='{{.State.Health.Status}}' bibliothek-web 2>/dev/null | grep -q "healthy"; do
    sleep 3
    WAIT=$((WAIT + 3))
    if [ ${WAIT} -ge 60 ]; then
        log_error "Web-Container ist nach 60 Sekunden nicht healthy!"
        log_error "Prüfe Logs: docker logs bibliothek-web --tail 50"
        print_rollback_instructions
        exit 1
    fi
    log_info "  ... noch ${WAIT}s gewartet"
done

log_ok "Anwendung ist healthy und läuft."

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

# ── Fertig ────────────────────────────────────────────────────────────────────
echo ""
log_ok "══════════════════════════════════════════════════"
log_ok "  UPDATE ERFOLGREICH ABGESCHLOSSEN"
log_ok "══════════════════════════════════════════════════"
echo ""
log_info "Anwendung läuft unter: http://$(hostname -I | awk '{print $1}'):8081"
echo ""
