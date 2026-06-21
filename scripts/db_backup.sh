#!/bin/bash
# db_backup.sh
# Automatisches Backup für die Bibliotheks-Datenbank mit 7-Tage-Rotation.

# Lade Umgebungsvariablen falls .env existiert
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

DB_USER="${POSTGRES_USER:-postgres}"
DB_NAME="${POSTGRES_DB:-bibliothek}"
DB_HOST="${POSTGRES_HOST:-localhost}"
DB_PORT="${POSTGRES_PORT:-5432}"

BACKUP_DIR="backups"
DATE=$(date +"%Y-%m-%d_%H-%M-%S")
BACKUP_FILE="$BACKUP_DIR/backup_${DB_NAME}_${DATE}.sql.gz"

mkdir -p "$BACKUP_DIR"

echo "Erstelle Backup für Datenbank $DB_NAME..."
pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" "$DB_NAME" | gzip > "$BACKUP_FILE"

if [ $? -eq 0 ]; then
    echo "Backup erfolgreich: $BACKUP_FILE"
    
    # Rotation: Lösche Backups, die älter als 7 Tage sind
    echo "Führe Rotation durch (lösche Backups älter als 7 Tage)..."
    find "$BACKUP_DIR" -type f -name "backup_${DB_NAME}_*.sql.gz" -mtime +7 -exec rm {} \;
    echo "Rotation abgeschlossen."
else
    echo "Fehler beim Erstellen des Backups!"
    exit 1
fi
