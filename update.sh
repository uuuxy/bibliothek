#!/usr/bin/env bash
# ==============================================================================
# update.sh — Zero-Data-Loss Update Script for Schulbibliothek
# ==============================================================================
# Ablauf:
#   1. pg_dump Backup mit Zeitstempel → ./backups/
#   2. Code holen: am Schulserver das neueste Release (UPDATE_STAND=release), am Testserver
#      git pull (main, auch ohne Eintrag)
#   3. Frische Images: Datenbank-Image holen, Backend mit --pull bauen, dann starten
#   4. Bei Fehler: Abbruch + Rollback-Anleitung
#   5. Nach erfolgreichem Deploy: Vorab-Sicherung verschlüsseln, Klartext löschen
#   6. Alte Backups aufräumen: vordeploy_ älter als 30 Tage, Klartext älter als 2 Tage.
#      Gelöscht wird nur hier, bei einem Lauf — eine Uhr, die ohne Lauf löscht, gibt es nicht.
#   7. Docker-Build-Cache wird geleert
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
# Nachgestellt: Ein flaches Skript, das sich waehrend des Laufs um drei Zeilen verlaengert,
# stirbt mit "unexpected EOF while looking for matching quote" und fuehrt seine letzten
# Schritte nicht mehr aus. Dieselbe Datei in { } gefasst laeuft vollstaendig durch.
#
# Der Grund: `{ ... }` ist ein zusammengesetzter Befehl. Bash parst ihn komplett, bevor es
# die erste Zeile darin ausfuehrt — ab da ist die Datei auf der Platte gleichgueltig. Das
# `exit` steht innerhalb der Klammer: Stuende es dahinter, kehrte bash nach dem Block zum
# Lesen der (inzwischen veraenderten) Datei zurueck und straeuchelte genau dort.
#
# Ein Deploy-Skript aendert sich immer erst fuer den naechsten Lauf: Der Lauf, der eine neue
# Fassung zieht, fuehrt noch die alte aus.
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

# Welcher Stand eingespielt wird, steht als UPDATE_STAND in der .env: release am Schulserver, das
# neueste Release-Tag vX.Y.Z (ein Release entsteht nur mit grünen Pflichtprüfungen,
# scripts/tag-gate.sh); main am Testserver, der Zweig, wie er ist. Ohne Eintrag gilt main, so
# läuft eine .env ohne die Zeile weiter wie bisher.
UPDATE_STAND="$(sed -n "s/^UPDATE_STAND=[\"']*\([^\"' #]*\).*/\1/p" "$(cd "$(dirname "$0")" && pwd)/.env" 2>/dev/null | tail -1 || true)"
UPDATE_STAND="${UPDATE_STAND:-main}"
ZIEL_TAG=""

DB_CONTAINER="bibliothek-db"
DB_USER="postgres"
DB_NAME="bibliothek"
# Ein Name für den Backend-Container, für alle Schritte: Ein zweiter Platz für denselben
# Wert liefe auseinander. scripts/backup_krypto.sh liest ihn als BACKEND_CONTAINER, deshalb
# steht er hier, bevor der Helfer eingebunden wird.
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
# Eigenes Präfix: Das nächtliche Backup heißt backup_<Zeit>.sql.gz.enc (jobs/backup.go, im
# Volume des Containers). Trüge die Vorab-Sicherung dasselbe Muster, wählte die
# Restore-Anleitung per `ls -t backups/backup_*` die letzte Deploy-Sicherung statt des
# Nachtbackups, und alles seit dem letzten Deploy wäre nach dem Restore still weg.
BACKUP_FILE="${BACKUP_DIR}/vordeploy_${TIMESTAMP}.sql.gz"
# Der Stand VOR diesem Update — der Rückweg der Rollback-Anleitung. Gelesen in Schritt 0
# aus dem laufenden Image; nur wenn das nicht geht, in Schritt 2 aus dem Arbeitsverzeichnis.
# VORHER_QUELLE sagt, woher er stammt ("image" oder "arbeitsverzeichnis").
VORHER_COMMIT=""
VORHER_QUELLE=""

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
    # Der Rückweg muss zum alten Code und zu den alten Daten führen: `git stash` wäre nach
    # einem sauberen Pull wirkungslos, ein Dump ohne dropdb und ohne ON_ERROR_STOP liefe in
    # die schon migrierte Datenbank (psql endet trotzdem mit 0). Das Gate ist
    # docs/rueckweg_anleitungen_test.go.
    echo -e "  ${BOLD}1. Backend stoppen (die Datenbank läuft weiter):${NC}"
    echo "     docker compose stop backend"
    echo ""
    echo -e "  ${BOLD}2. Code auf den Stand VOR diesem Update zurücksetzen:${NC}"
    if [ -n "${VORHER_COMMIT}" ]; then
        echo "     git reset --hard ${VORHER_COMMIT}"
        if [ "${VORHER_QUELLE}" = "arbeitsverzeichnis" ]; then
            echo "     # Achtung: Der laufende Stand war nicht lesbar, dieser Commit stammt aus dem"
            echo "     # Arbeitsverzeichnis. Lief git pull schon vor diesem Skript, ist er bereits"
            echo "     # der neue Stand — dann zeigt git reflog den Commit davor."
        fi
    else
        echo "     # Kein git pull gelaufen — der Code ist noch der alte, nichts zu tun."
    fi
    echo ""
    echo -e "  ${BOLD}3. Datenbank auf den Stand der Vorab-Sicherung zurücksetzen:${NC}"
    echo "     # Die Migrationen des neuen Codes sind womöglich schon gelaufen — deshalb die"
    echo "     # Datenbank NEU anlegen statt in die migrierte hineinzuspielen. ON_ERROR_STOP"
    echo "     # bricht beim ersten Fehler ab, statt mit Exit 0 eine halbe Datenbank zu lassen."
    echo "     docker exec ${DB_CONTAINER} dropdb -U ${DB_USER} --force --if-exists ${DB_NAME}"
    echo "     docker exec ${DB_CONTAINER} createdb -U ${DB_USER} ${DB_NAME}"
    echo "     gunzip -c \"${BACKUP_FILE}\" | docker exec -i ${DB_CONTAINER} psql -v ON_ERROR_STOP=1 -U ${DB_USER} -d ${DB_NAME}"
    echo ""
    echo -e "  ${BOLD}4. Den ALTEN Code bauen und starten:${NC}"
    echo "     docker compose up -d --build"
    echo ""
    echo -e "  ${BOLD}5. Danach: die Vorab-Sicherung entsorgen${NC}"
    echo "     # Sie ist UNVERSCHLÜSSELT und enthält jeden Schülernamen, jede Adresse,"
    echo "     # jede Ausleihe im Klartext. Nach geglücktem Rollback löschen:"
    echo "     shred -u \"${BACKUP_FILE}\"   # oder: rm -f"
    echo "     # Sonst löscht sie erst ein späterer Lauf von update.sh oder scripts/backup.sh,"
    echo "     # frühestens nach ${KLARTEXT_RETENTION_DAYS} Tagen."
    echo ""
    echo -e "${YELLOW}Alle verfügbaren Backups:${NC}"
    ls -lh "${BACKUP_DIR}"/*.sql.gz "${BACKUP_DIR}"/*.sql.gz.enc 2>/dev/null || echo "  (keine Backups gefunden)"
    echo ""
    # Die Dateien liegen auf dem Host, das Werkzeug im Image: Die Datei geht über die
    # Standardeingabe hinein. Ein Wegwerf-Container statt docker exec, weil Schritt 1 das
    # Backend anhält.
    echo -e "${YELLOW}Verschlüsselte Sicherungen (.enc) zuerst öffnen:${NC}"
    echo "     read -rp \"Datei: \" ENC"
    echo "     docker compose run --rm --no-deps -T --entrypoint ./restore-backup backend /dev/stdin < \"\$ENC\" > wiederherstellung.sql"
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

# Das Release wird vor der Sicherung bestimmt: Gibt es keines, bricht das Update ab, bevor es
# etwas getan hat.
case "${UPDATE_STAND}" in
    main)
        log_info "Stand: main, der Zweig, wie er ist."
        ;;
    release)
        if ! git -C "$(dirname "$0")" fetch --quiet --tags --force origin; then
            log_error "Die Releases ließen sich nicht holen (git fetch --tags)."
            exit 1
        fi
        ZIEL_TAG="$(git -C "$(dirname "$0")" tag --list 'v*' --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -1 || true)"
        if [ -z "${ZIEL_TAG}" ]; then
            log_error "Kein Release gefunden (Tag vX.Y.Z). Ein Release entsteht mit scripts/tag-gate.sh."
            exit 1
        fi
        # Nur vorwärts: Migrationen laufen nie zurück, und alter Code träfe auf ein neueres Schema.
        # Liegt das Release vor dem Arbeitsverzeichnis, geht es zurück nur über die Vorab-Sicherung.
        if ! git -C "$(dirname "$0")" merge-base --is-ancestor HEAD "${ZIEL_TAG}"; then
            log_error "Das Release ${ZIEL_TAG} liegt nicht nach dem Stand im Arbeitsverzeichnis ($(git -C "$(dirname "$0")" rev-parse --short HEAD))."
            log_error "Zurück geht es nur über eine Sicherung (DEPLOYMENT.md §6); dieses Update tut nichts."
            exit 1
        fi
        log_info "Stand: Release ${ZIEL_TAG}."
        ;;
    *)
        log_error "UPDATE_STAND=${UPDATE_STAND} in der .env ist unbekannt; erlaubt sind release und main."
        exit 1
        ;;
esac

if ! docker ps --filter "name=${DB_CONTAINER}" --filter "status=running" --format '{{.Names}}' | grep -q "${DB_CONTAINER}"; then
    log_warn "DB-Container '${DB_CONTAINER}' läuft nicht. Backup wird übersprungen."
    SKIP_BACKUP=true
else
    SKIP_BACKUP=false
fi

# Der Rückweg der Rollback-Anleitung ist der Commit, aus dem das laufende Image gebaut wurde
# (GIT_COMMIT, dieselbe Quelle wie Schritt 4b). `git rev-parse HEAD` vor dem Pull in
# Schritt 2 ist nach dem vorgezogenen `git pull` aus DEPLOYMENT.md §2.4 schon der neue
# Stand, und ORIG_HEAD hilft nicht: Der Pull in Schritt 2, der nichts mehr findet, setzt es
# auf den neuen Stand (docs/rueckweg_anleitungen_test.go,
# TestRueckweg_RollbackNenntDenLaufendenStand).
VORHER_COMMIT="$(docker exec "${APP_CONTAINER}" printenv GIT_COMMIT 2>/dev/null || echo '')"
if [ -n "${VORHER_COMMIT}" ]; then
    VORHER_QUELLE="image"
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
log_step "Schritt 2: Code aktualisieren"

if [ -d "$(dirname "$0")/.git" ]; then
    cd "$(dirname "$0")"
    if [ -z "${VORHER_COMMIT}" ]; then
        # Rückfall: kein laufender Container, oder sein Image trägt keinen Commit. Dann
        # bleibt nur das Arbeitsverzeichnis — richtig nur, wenn vorher kein git pull lief.
        VORHER_COMMIT="$(git rev-parse HEAD 2>/dev/null || echo '')"
        VORHER_QUELLE="arbeitsverzeichnis"
    fi
    if [ "${UPDATE_STAND}" = "release" ]; then
        log_info "Wechsle auf ${ZIEL_TAG}... (Stand vor dem Update: ${VORHER_COMMIT}, aus: ${VORHER_QUELLE})"
        if ! git checkout --quiet --detach "${ZIEL_TAG}"; then
            log_error "git checkout ${ZIEL_TAG} fehlgeschlagen!"
            print_rollback_instructions
            exit 1
        fi
    else
        log_info "Führe git pull aus... (Stand vor dem Update: ${VORHER_COMMIT}, aus: ${VORHER_QUELLE})"
        if ! git pull; then
            log_error "git pull fehlgeschlagen!"
            print_rollback_instructions
            exit 1
        fi
    fi
    log_ok "Code aktualisiert."
else
    log_warn ".git-Verzeichnis nicht gefunden — git pull übersprungen."
    log_warn "Bitte Code manuell aktualisieren, bevor du dieses Skript ausführst."
fi

# ── Schritt 3: Container neu bauen und starten ────────────────────────────────
log_step "Schritt 3: Docker-Container neu bauen und starten"

log_info "Hole das Datenbank-Image, baue das Backend mit frischen Basis-Images und starte..."

# Der Commit wandert als Build-Argument ins Image (Dockerfile: ARG/ENV GIT_COMMIT) und
# wird in Schritt 4b dagegen geprüft. Ohne .git bleibt er leer — dann sagt die Prüfung
# "unbekannt", statt einen falschen Stand zu behaupten.
GIT_COMMIT="$(git rev-parse HEAD 2>/dev/null || echo '')"
export GIT_COMMIT

# Frisch in beiden Fällen: Das Datenbank-Image kommt in der neuesten Nebenversion (sie braucht
# weder Sicherung noch Neuaufbau, PFLEGEKONZEPT.md 4), der Bau holt die Basis-Images neu
# (--pull). Sonst blieben beide auf dem Stand, der beim ersten Aufbau vorlag.
if docker compose -f "${COMPOSE_FILE}" pull postgres-db \
    && docker compose -f "${COMPOSE_FILE}" build --pull \
    && docker compose -f "${COMPOSE_FILE}" up -d; then
    log_ok "Container erfolgreich neu gestartet."
else
    log_error "Holen, Bauen oder Starten ist fehlgeschlagen (docker compose pull, build, up)!"
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
# Beide Quellen, nicht nur die erste: Hinge die Prüfung allein am Healthcheck, meldete sie
# ein Update als fehlgeschlagen, während /health längst mit 200 antwortet. Ein Alarm, der bei
# jedem gesunden Lauf losgeht, erzieht dazu, ihn zu überhören — und dann fehlt er, wenn er
# zählt.
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
# Gesund heisst nicht aktuell: Nach `git pull` ohne `./update.sh` zeigt das
# Arbeitsverzeichnis den neuen Commit, während ein altes Image läuft, und beide Prüfungen
# aus Schritt 4 melden zu Recht "gesund" — sie beantworten eine andere Frage. Diese Prüfung
# fragt: Läuft der Container, der aus dem Commit gebaut wurde, der gerade im
# Arbeitsverzeichnis liegt?
log_step "Schritt 4b: Läuft der Container aus diesem Commit?"

ERWARTET="$(git rev-parse HEAD 2>/dev/null || echo '')"
IM_IMAGE="$(docker exec "${APP_CONTAINER}" printenv GIT_COMMIT 2>/dev/null || echo '')"

if [ -z "${ERWARTET}" ]; then
    log_warn "Kein git-Arbeitsverzeichnis — Abgleich nicht möglich."
elif [ -z "${IM_IMAGE}" ]; then
    # Jedes über update.sh gebaute Image trägt seinen Commit. Ein leerer Wert heißt: von
    # Hand ohne GIT_COMMIT gebaut — genau der stille „alter Container"-Fall, den dieser
    # Schritt fangen soll; deshalb ein Fehler und keine Warnung.
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
        log_info "Öffnen mit: docker compose run --rm --no-deps -T --entrypoint ./restore-backup backend /dev/stdin < \"${KRYPTO_ZIEL}\" > wiederherstellung.sql"
    else
        log_warn "Verschlüsselung fehlgeschlagen — die Sicherung bleibt im KLARTEXT liegen:"
        log_warn "  ${BACKUP_FILE}"
        log_warn "  Sie enthält jeden Schülernamen und jede Adresse im Klartext. Nicht mehr"
        log_warn "  gebraucht: selbst löschen (shred -u). Sonst löscht sie erst ein späterer"
        log_warn "  Lauf, frühestens nach ${KLARTEXT_RETENTION_DAYS} Tagen."
    fi
else
    log_warn "Verschlüsselung nicht möglich (${KRYPTO_GRUND})."
    log_warn "Die Vorab-Sicherung bleibt im KLARTEXT liegen: ${BACKUP_FILE}"
    log_warn "Nicht mehr gebraucht: selbst löschen (shred -u). Sonst löscht sie erst ein"
    log_warn "späterer Lauf, frühestens nach ${KLARTEXT_RETENTION_DAYS} Tagen."
fi

# ── Schritt 6: Alte Backups aufräumen ─────────────────────────────────────────
log_step "Schritt 6: Alte Backups aufräumen (verschlüsselt: ${BACKUP_RETENTION_DAYS} Tage, Klartext: ${KLARTEXT_RETENTION_DAYS} Tage)"

# Zwei Fristen, zwei Muster. `*.sql.gz` trifft NICHT die `.enc`-Dateien — deren Name
# endet auf `.enc`, und `find -name` vergleicht den ganzen Namen.
# Verschlüsselt: nur die eigenen (vordeploy_, und backup_ für ältere Vorab-Sicherungen
# unter dem früheren Namen); scripts/backup.sh legt im selben Ordner mit eigener Frist ab.
# Klartext: jeder im Ordner, gleich von wem (Gate: docs/backup_ablage_test.go).
DELETED=$(find "${BACKUP_DIR}" \( -name "vordeploy_*.sql.gz.enc" -o -name "backup_*.sql.gz.enc" \) -mtime "+${BACKUP_RETENTION_DAYS}" -print -delete 2>/dev/null | wc -l | tr -d ' ')
DELETED_KLAR=$(find "${BACKUP_DIR}" -name "*.sql.gz" -mtime "+${KLARTEXT_RETENTION_DAYS}" -print -delete 2>/dev/null | wc -l | tr -d ' ')

if [ "${DELETED}" -gt 0 ] || [ "${DELETED_KLAR}" -gt 0 ]; then
    log_ok "${DELETED} verschlüsselte/s und ${DELETED_KLAR} unverschlüsselte/s Backup/s gelöscht."
else
    log_info "Keine alten Backups zum Löschen gefunden."
fi

REMAINING=$(find "${BACKUP_DIR}" -name "*.sql.gz.enc" 2>/dev/null | wc -l | tr -d ' ')
REMAINING_KLAR=$(find "${BACKUP_DIR}" -name "*.sql.gz" 2>/dev/null | wc -l | tr -d ' ')
log_info "${REMAINING} verschlüsselte/s Backup/s verbleiben in ${BACKUP_DIR}/"
if [ "${REMAINING_KLAR}" -gt 0 ]; then
    log_warn "${REMAINING_KLAR} UNVERSCHLÜSSELTE/S Backup/s liegt/liegen dort ebenfalls (Klarnamen im Klartext)."
fi

# ── Schritt 7: Docker-Build-Cache leeren ──────────────────────────────────────
log_step "Schritt 7: Docker-Build-Cache leeren"

# Jedes Update baut ein neues Image, und der Build-Cache wächst dabei weiter. Läuft die
# Platte voll, steht der ganze Server, samt der fremden Dienste hinter demselben Caddy.
#
# Geleert wird ganz, ohne Zeitfilter: Mit `--filter until=…` löscht Docker je Lauf nur die
# äußerste Schicht jeder Kette und führt ihre Eltern danach als eben benutzt. Das nächste
# Update baut dafür jede Schicht neu; das laufende Image bleibt, bis das neue steht.
#
# Kein `docker system prune` und kein `docker volume prune`: Auf diesem Server liegen fremde
# benannte Volumes (school-calendar mit Postgres und Backups), die Docker als unbenutzt führt.
FREI_VORHER="$(df -Pk / | awk 'NR==2 {print $4}')"

# Scheitert das Leeren, bleibt das Update geglückt: eine Warnung, kein Abbruch.
if docker builder prune -af >/dev/null 2>&1; then
    FREI_NACHHER="$(df -Pk / | awk 'NR==2 {print $4}')"
    BEFREIT_MB=$(( (FREI_NACHHER - FREI_VORHER) / 1024 ))
    if [ "${BEFREIT_MB}" -gt 0 ]; then
        log_ok "Build-Cache geleert: ${BEFREIT_MB} MB frei geworden."
    else
        log_info "Build-Cache geleert: Es war nichts zu löschen."
    fi
else
    log_warn "Build-Cache konnte nicht geleert werden — das Update selbst ist davon unberührt."
fi
log_info "Speicher auf /: $(df -h / | awk 'NR==2 {print $4}') frei ($(df -h / | awk 'NR==2 {print $5}') belegt)"

# ── Fertig ────────────────────────────────────────────────────────────────────
echo ""
log_ok "══════════════════════════════════════════════════"
log_ok "  UPDATE ERFOLGREICH ABGESCHLOSSEN"
log_ok "══════════════════════════════════════════════════"
echo ""
log_info "Anwendung läuft lokal auf Port 8083 (Stand: ${ZIEL_TAG:-main $(git rev-parse --short HEAD 2>/dev/null || echo unbekannt)})."
echo ""

exit 0
}
