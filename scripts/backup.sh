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

# Verschlüsselungs-Helfer (docs/resilience_and_recovery.md 1b)
# shellcheck source=scripts/backup_krypto.sh
source "$(dirname "$0")/backup_krypto.sh"

# Name des laufenden Datenbank-Containers (siehe docker-compose.yml)
CONTAINER_NAME="bibliothek-db"
# PostgreSQL Benutzername / Datenbank — Defaults wie im Compose (${VAR:-...})
DB_USER="${POSTGRES_USER:-postgres}"
DB_NAME="${POSTGRES_DB:-bibliothek}"
# Verzeichnis, in dem die Backups gespeichert werden sollen
BACKUP_DIR="$(dirname "$0")/../backups"

# Aufbewahrung: verschlüsselt eine Woche, Klartext nur zwei Tage.
#
# Der Unterschied ist der ganze Punkt (Befund A5): Ein `.enc` ist ohne Passphrase
# wertlos und darf deshalb liegen bleiben. Ein Klartext-Dump entsteht hier nur noch als
# Notnagel, wenn die Verschlüsselung nicht möglich war — er ist eine lesbare Kopie des
# gesamten Bestands und bekommt die kurze Lunte.
RETENTION_ENC_TAGE=7
RETENTION_KLARTEXT_TAGE=2

# ==========================================

# Stelle sicher, dass das Backup-Verzeichnis existiert
mkdir -p "$BACKUP_DIR"

# Aktuelles Datum für den Dateinamen (z.B. 2026-06-11)
TIMESTAMP=$(date +"%Y-%m-%d")
BACKUP_BASIS="$BACKUP_DIR/bibliothek_backup_$TIMESTAMP.sql.gz"

echo "Starte Backup für Datenbank '$DB_NAME' aus Container '$CONTAINER_NAME'..."

# umask im Subshell, nicht chmod danach: Läge zwischen Anlegen und Rechtesetzen eine
# Lücke, wäre die Datei genau während des Schreibens für alle lesbar.
#
# pipefail ebenfalls im Subshell: Ohne ihn liefert die Pipe den Status des LETZTEN
# Glieds, und gzip gelingt auch dann, wenn pg_dump vorher abgebrochen ist. Das Skript
# meldete damit "Backup erfolgreich" und legte eine gzip-Datei mit einer Fehlermeldung
# darin ab — ein Backup, dessen Ausfall man erst beim Wiederherstellen bemerkt.
if krypto_moeglich; then
  # Der Regelweg: pg_dump → gzip → Verschlüsselung, alles in EINER Pipe. Der Klartext
  # berührt die Platte nie, nicht einmal für Sekunden.
  BACKUP_FILE="${BACKUP_BASIS}.enc"
  if (umask 077; set -o pipefail; \
      docker exec "$CONTAINER_NAME" pg_dump -U "$DB_USER" -d "$DB_NAME" \
      | gzip | krypto_pipe > "$BACKUP_FILE") \
     && pruefe_enc_datei "$BACKUP_FILE" && pruefe_enc_rundweg "$BACKUP_FILE"; then
    echo "Backup erfolgreich (verschlüsselt, Rückweg geprüft): $BACKUP_FILE"
    echo "  Wiederherstellen: docker compose exec backend ./restore-backup <datei> <ziel.sql>"
  else
    echo "FEHLER: Backup fehlgeschlagen (Dump, Verschlüsselung oder Rückweg-Prüfung)!"
    # Die angefangene Datei nicht liegen lassen: Sie sieht wie ein Backup aus.
    rm -f "$BACKUP_FILE"
    exit 1
  fi
else
  # Notnagel. Kein Abbruch: Ein lesbares Backup ist besser als keines — aber es wird
  # benannt, was es ist, und es verschwindet nach zwei Tagen.
  echo "WARNUNG: Verschlüsselung nicht möglich ($KRYPTO_GRUND)."
  echo "WARNUNG: Es entsteht ein UNVERSCHLÜSSELTER Dump mit allen Klarnamen und Adressen."
  echo "WARNUNG: Er wird nach ${RETENTION_KLARTEXT_TAGE} Tagen gelöscht — bitte vorher selbst entsorgen."
  BACKUP_FILE="$BACKUP_BASIS"
  if (umask 077; set -o pipefail; \
      docker exec "$CONTAINER_NAME" pg_dump -U "$DB_USER" -d "$DB_NAME" | gzip > "$BACKUP_FILE"); then
    echo "Backup erfolgreich (unverschlüsselt): $BACKUP_FILE"
  else
    echo "FEHLER: Backup fehlgeschlagen!"
    rm -f "$BACKUP_FILE"
    exit 1
  fi
fi

# -- Aufräum-Logik (Retention Policy) --
echo "Räume alte Backups auf (verschlüsselt: ${RETENTION_ENC_TAGE} Tage, Klartext: ${RETENTION_KLARTEXT_TAGE} Tage)..."
find "$BACKUP_DIR" -type f -name "*.sql.gz.enc" -mtime "+${RETENTION_ENC_TAGE}" -exec rm -f {} \;
find "$BACKUP_DIR" -type f -name "*.sql.gz" -mtime "+${RETENTION_KLARTEXT_TAGE}" -exec rm -f {} \;

KLARTEXT_UEBRIG=$(find "$BACKUP_DIR" -type f -name "*.sql.gz" 2>/dev/null | wc -l | tr -d ' ')
if [ "$KLARTEXT_UEBRIG" -gt 0 ]; then
  echo "WARNUNG: ${KLARTEXT_UEBRIG} unverschlüsselte/r Dump/s liegt/liegen noch in ${BACKUP_DIR}/ (Klarnamen im Klartext)."
fi

echo "Backup-Prozess abgeschlossen."
