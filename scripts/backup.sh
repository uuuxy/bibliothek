#!/bin/bash

# ==========================================
# PostgreSQL Docker Backup Script
# ==========================================

# -- Konfiguration --
# Name des laufenden Datenbank-Containers
CONTAINER_NAME="bibliothek-db"
# PostgreSQL Benutzername (default: postgres)
DB_USER="postgres"
# Name der Datenbank
DB_NAME="bibliothek"
# Verzeichnis, in dem die Backups gespeichert werden sollen
BACKUP_DIR="$(dirname "$0")/../backups"

# ==========================================

# Stelle sicher, dass das Backup-Verzeichnis existiert
mkdir -p "$BACKUP_DIR"

# Aktuelles Datum für den Dateinamen (z.B. 2026-06-11)
TIMESTAMP=$(date +"%Y-%m-%d")
BACKUP_FILE="$BACKUP_DIR/bibliothek_backup_$TIMESTAMP.sql.gz"

echo "Starte Backup für Datenbank '$DB_NAME' aus Container '$CONTAINER_NAME'..."

# Führe pg_dump im Container aus und komprimiere den Output direkt mit gzip
docker exec "$CONTAINER_NAME" pg_dump -U "$DB_USER" -d "$DB_NAME" | gzip > "$BACKUP_FILE"

if [ $? -eq 0 ]; then
  echo "Backup erfolgreich: $BACKUP_FILE"
else
  echo "FEHLER: Backup fehlgeschlagen!"
  exit 1
fi

# -- Aufräum-Logik (Retention Policy) --
# Lösche alle Dateien im Backup-Ordner, die älter als 7 Tage sind und auf .sql.gz enden
echo "Räume alte Backups auf (älter als 7 Tage)..."
find "$BACKUP_DIR" -type f -name "*.sql.gz" -mtime +7 -exec rm -f {} \;

echo "Backup-Prozess abgeschlossen."
