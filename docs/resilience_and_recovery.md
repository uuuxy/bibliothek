# System-Resilienz und Wiederherstellung

Dieses Dokument beschreibt die Backup-Strategien, Wiederherstellungsverfahren sowie die in der Bibliothek implementierten Sicherheitsmechanismen wie Soft-Deletes und Audit-Logs.

## 1. Backups

Es gibt **zwei** Backup-Wege. Maßgeblich für die Wiederherstellung ist, welches Format vorliegt.

### 1a. Automatisches verschlüsseltes Backup (Primär, Produktion)
Der eingebaute Scheduler (`jobs.RunDatabaseBackup`) läuft täglich um **02:30 UTC** und erzeugt
**AES-256-GCM-verschlüsselte**, gzip-komprimierte `pg_dump`-Dateien:

```
backups/backup_<ZEITSTEMPEL>.sql.gz.enc
```

- Schlüssel: `BACKUP_ENCRYPTION_KEY` (≥ 32 Zeichen), Ableitung via SHA-256.
- Rotation: die letzten **14** Backups bleiben erhalten.
- Optionaler Offsite-Upload nach S3, falls `S3_ENDPOINT`/`S3_ACCESS_KEY`/`S3_SECRET_KEY`/`S3_BUCKET` gesetzt sind.

> ⚠️ **Wichtig:** Diese Dateien sind verschlüsselt. `zcat`/`gunzip`/`psql` funktionieren darauf **nicht**
> direkt — sie müssen zuerst mit dem `restore-backup`-Tool entschlüsselt werden (siehe Abschnitt 2a).
> Ohne den originalen `BACKUP_ENCRYPTION_KEY` ist ein verschlüsseltes Backup **nicht** wiederherstellbar.

### 1b. Manuelles unverschlüsseltes Skript (Optional)
`scripts/backup.sh` erzeugt eine unverschlüsselte `.sql.gz`-Datei (`backups/bibliothek_backup_<DATUM>.sql.gz`,
7-Tage-Rotation) — führt `pg_dump` per `docker exec` im DB-Container aus (z. B. für schnelle Ad-hoc-Sicherungen).
Beispiel-Crontab:

```bash
0 2 * * * /Pfad/zu/Bibliothek/scripts/backup.sh >> /Pfad/zu/Bibliothek/backups/backup.log 2>&1
```

## 2. Wiederherstellung (Recovery)

Allgemeine Schritte: (1) Anwendung stoppen, (2) Backup auswählen, (3) Datenbank neu erstellen und einspielen,
(4) Anwendung neu starten. Das Einspielen unterscheidet sich je nach Backup-Format.

> Der Backup-Restore-Round-Trip (Verschlüsselung ↔ Entschlüsselung) ist durch automatisierte Tests
> abgesichert: `go test ./jobs/ -run TestBackupRestore`. Vor einem produktiven Go-Live sollte zusätzlich
> **einmal** ein echtes Restore in eine Wegwerf-Datenbank durchgespielt werden (siehe 2c).

### 2a. Verschlüsseltes `.sql.gz.enc`-Backup (Abschnitt 1a)

```bash
# 0. Restore-Tool bauen (einmalig)
go build -o restore-backup ./cmd/restore-backup

# 1. Datenbank leeren/neu anlegen (ACHTUNG: löscht alle aktuellen Daten!)
dropdb -U postgres bibliothek
createdb -U postgres bibliothek

# 2. Entschlüsseln + dekomprimieren + direkt einspielen
BACKUP_ENCRYPTION_KEY="<originaler-schluessel>" \
  ./restore-backup backups/backup_<ZEITSTEMPEL>.sql.gz.enc | psql -U postgres -d bibliothek

# Alternativ: erst in eine Datei entschlüsseln, dann einspielen
BACKUP_ENCRYPTION_KEY="<…>" ./restore-backup backups/backup_<…>.sql.gz.enc wiederherstellung.sql
psql -U postgres -d bibliothek -f wiederherstellung.sql
```

### 2b. Unverschlüsseltes `.sql.gz`-Backup (Abschnitt 1b)

```bash
dropdb -U postgres bibliothek
createdb -U postgres bibliothek
zcat backups/bibliothek_backup_<…>.sql.gz | psql -U postgres -d bibliothek
```

### 2c. Restore-Probe vor Go-Live (dringend empfohlen)
Ein Backup, das nie zurückgespielt wurde, ist kein verlässliches Backup. Einmal gefahrlos verifizieren:

```bash
createdb -U postgres bibliothek_restore_test
BACKUP_ENCRYPTION_KEY="<…>" ./restore-backup backups/backup_<…>.sql.gz.enc \
  | psql -U postgres -d bibliothek_restore_test
# Stichprobe, danach Wegwerf-DB entfernen:
psql -U postgres -d bibliothek_restore_test -c "SELECT count(*) FROM schueler;"
dropdb -U postgres bibliothek_restore_test
```

Anschließend die Anwendung neu starten.

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
