#!/usr/bin/env bash
# Automatisiertes PostgreSQL Backup-Skript für die Bibliothek
# Wird typischerweise via Cronjob jede Nacht ausgeführt.

# Beende sofort bei Fehlern
set -e

# 1. Umgebungsvariablen aus der .env-Datei im Projekt-Root laden
ENV_FILE="$(dirname "$0")/../.env"
if [ -f "$ENV_FILE" ]; then
    # Lade alle Variablen, die nicht auskommentiert sind
    export $(grep -v '^#' "$ENV_FILE" | xargs)
fi

# 2. Datenbank-Konfiguration (Fallbacks, falls in .env nicht explizit gesetzt)
DB_HOST=${DB_HOST:-"localhost"}
DB_PORT=${DB_PORT:-"5432"}
DB_USER=${DB_USER:-"postgres"}
DB_NAME=${DB_NAME:-"bibliothek"}
# pg_dump erwartet das Passwort in der Umgebungsvariable PGPASSWORD
export PGPASSWORD=${DB_PASSWORD:-${PGPASSWORD:-""}}

# 3. Backup-Einstellungen
BACKUP_DIR="$(dirname "$0")/../backups"
TIMESTAMP=$(date +"%Y-%m-%d_%H-%M-%S")
BACKUP_FILE="$BACKUP_DIR/backup_${DB_NAME}_${TIMESTAMP}.sql.gz"
RETENTION_DAYS=14

# Sicherstellen, dass das Backup-Verzeichnis existiert
mkdir -p "$BACKUP_DIR"

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Starte Backup für Datenbank '$DB_NAME' (Host: $DB_HOST)..."

# 4. Dump erstellen und direkt komprimieren (-F p ist plain text, wird direkt durch gzip geschleift)
pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -F p | gzip > "$BACKUP_FILE"

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Backup erfolgreich erstellt: $BACKUP_FILE"

# 5. Aufräumen: Lösche Backups, die älter als RETENTION_DAYS (14 Tage) sind
echo "[$(date +'%Y-%m-%d %H:%M:%S')] Lösche alte Backups (älter als $RETENTION_DAYS Tage)..."
find "$BACKUP_DIR" -type f -name "backup_${DB_NAME}_*.sql.gz" -mtime +$RETENTION_DAYS -exec rm -f {} \;

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Backup-Prozess abgeschlossen."
