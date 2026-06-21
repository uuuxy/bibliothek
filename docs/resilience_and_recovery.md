# System-Resilienz und Wiederherstellung

Dieses Dokument beschreibt die Backup-Strategien, Wiederherstellungsverfahren sowie die in der Bibliothek implementierten Sicherheitsmechanismen wie Soft-Deletes und Audit-Logs.

## 1. Backups

Zur Sicherung der Datenbank wird das Skript `scripts/db_backup.sh` verwendet.
Es sichert die PostgreSQL-Datenbank in eine komprimierte `.sql.gz`-Datei und behält eine Historie von 7 Tagen (Rotation).

### Automatisierung (Cronjob)
Es wird empfohlen, das Backup-Skript täglich automatisiert auf dem Server auszuführen.
Beispiel für einen Crontab-Eintrag (Ausführung täglich um 02:00 Uhr):

```bash
0 2 * * * /Pfad/zu/Bibliothek/scripts/db_backup.sh >> /Pfad/zu/Bibliothek/backups/backup.log 2>&1
```

*(Hinweis: Stellen Sie sicher, dass in dem Verzeichnis, von dem das Skript aufgerufen wird, eine `.env` Datei mit den Datenbank-Zugangsdaten wie `POSTGRES_USER` und `POSTGRES_DB` liegt, falls nicht die Standardwerte genutzt werden sollen.)*

## 2. Wiederherstellung (Recovery)

Falls die Datenbank aus einem Backup wiederhergestellt werden muss, befolgen Sie diese Schritte:

1. Stoppen Sie die Anwendung, um parallele Zugriffe auf die Datenbank zu verhindern.
2. Suchen Sie das aktuellste Backup im Ordner `backups/`.
3. Stellen Sie die Datenbank über `psql` / `zcat` oder `gunzip` wieder her.

**Beispiel-Befehl:**
```bash
# Alte Datenbank leeren/löschen und neu erstellen (Achtung: Dies löscht alle aktuellen Daten!)
dropdb -U postgres bibliothek
createdb -U postgres bibliothek

# Backup einspielen
zcat backups/backup_bibliothek_YYYY-MM-DD_HH-MM-SS.sql.gz | psql -U postgres -d bibliothek
```

4. Starten Sie die Anwendung neu.

## 3. Soft-Deletes und Datenintegrität

Die Bibliothek implementiert für zentrale Entitäten wie **Schüler** sogenannte *Soft-Deletes*.

- Beim "Löschen" eines Schülers (z. B. durch einen Administrator oder den DSGVO-Job) wird der Datensatz nicht physisch aus der Datenbank entfernt.
- Stattdessen wird die Spalte `deleted_at` auf den aktuellen Zeitstempel gesetzt.
- Sämtliche regulären Lesezugriffe (Such-APIs, Export-Jobs, Laufzettel) filtern diese Datensätze automatisch heraus (`WHERE deleted_at IS NULL`).
- **Vorteil**: Ehemalige Buchausleihen und historische Transaktionen behalten ihre Integrität (Foreign Keys bleiben gültig). Sollte ein Schüler versehentlich gelöscht worden sein, reicht es aus, `deleted_at` in der Datenbank manuell auf `NULL` zu setzen.

## 4. Audit-Logs für kritische Aktionen

Neben der Ausleihen- und Rückgaben-Historie werden kritische administrative Eingriffe im System revisionssicher protokolliert.

- Die Tabelle `audit_logs` speichert dabei unter anderem:
  - Wer (Admin-ID) hat die Aktion durchgeführt?
  - Wann (Zeitstempel) wurde die Aktion ausgeführt?
  - Was (Aktion, z.B. `OVERRIDE_BLOCK`, `RECEIVE_ITEM`, `DELETE_STUDENT`) wurde getan?
  - Details im JSON-Format für tiefergehende Analysen.
- Dies stellt sicher, dass manuelle Sperr-Aufhebungen oder Wareneingänge jederzeit nachvollzogen werden können.
