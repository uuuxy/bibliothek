#!/bin/bash

# ==========================================
# PostgreSQL Docker Backup Script
# ==========================================

# -- Konfiguration --
# .env aus dem Repo-Root laden, falls vorhanden (POSTGRES_USER/POSTGRES_DB)
ENV_FILE="$(dirname "$0")/../.env"
if [ -f "$ENV_FILE" ]; then
  export $(grep -v '^#' "$ENV_FILE" | xargs)
fi

# Name des laufenden Datenbank-Containers (siehe docker-compose.yml)
CONTAINER_NAME="bibliothek-db"
# PostgreSQL Benutzername / Datenbank — Defaults wie im Compose (${VAR:-...})
DB_USER="${POSTGRES_USER:-postgres}"
DB_NAME="${POSTGRES_DB:-bibliothek}"
# Verzeichnis, in dem die Backups gespeichert werden sollen
BACKUP_DIR="$(dirname "$0")/../backups"

# ==========================================

# Stelle sicher, dass das Backup-Verzeichnis existiert
mkdir -p "$BACKUP_DIR"

# Aktuelles Datum für den Dateinamen (z.B. 2026-06-11)
TIMESTAMP=$(date +"%Y-%m-%d")
BACKUP_FILE="$BACKUP_DIR/bibliothek_backup_$TIMESTAMP.sql.gz"

echo "Starte Backup für Datenbank '$DB_NAME' aus Container '$CONTAINER_NAME'..."

# Führe pg_dump im Container aus und komprimiere den Output direkt mit gzip.
#
# umask im Subshell, nicht chmod danach: Der Dump ist unverschlüsselt und enthält jeden
# Schülernamen, jede Adresse und jede Ausleihe im Klartext. Ein nachgereichtes chmod
# ließe die Datei genau während des Schreibens für alle lesbar liegen.
# Das VERSCHLÜSSELTE Backup erzeugt der nächtliche Job (jobs/backup.go); dieses Skript
# bleibt bewusst unverschlüsselt, damit "zcat | psql" ohne Schlüssel funktioniert
# (docs/resilience_and_recovery.md, Abschnitt 1b/2b).
# pipefail ebenfalls im Subshell: Ohne ihn liefert die Pipe den Status von gzip, und
# gzip gelingt auch dann, wenn pg_dump vorher abgebrochen ist. Das Skript meldete
# damit "Backup erfolgreich" und legte eine gzip-Datei mit einer Fehlermeldung darin
# ab — ein Backup, dessen Ausfall man erst beim Wiederherstellen bemerkt.
if (umask 077; set -o pipefail; docker exec "$CONTAINER_NAME" pg_dump -U "$DB_USER" -d "$DB_NAME" | gzip > "$BACKUP_FILE"); then
  echo "Backup erfolgreich: $BACKUP_FILE"
else
  echo "FEHLER: Backup fehlgeschlagen!"
  # Die angefangene Datei nicht liegen lassen: Sie sieht wie ein Backup aus.
  rm -f "$BACKUP_FILE"
  exit 1
fi

# -- Aufräum-Logik (Retention Policy) --
# Lösche alle Dateien im Backup-Ordner, die älter als 7 Tage sind und auf .sql.gz enden
echo "Räume alte Backups auf (älter als 7 Tage)..."
find "$BACKUP_DIR" -type f -name "*.sql.gz" -mtime +7 -exec rm -f {} \;

echo "Backup-Prozess abgeschlossen."
