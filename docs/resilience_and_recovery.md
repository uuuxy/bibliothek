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

- Schlüssel: `BACKUP_ENCRYPTION_KEY` (≥ 32 Zeichen). Ableitung via **scrypt**
  (N=2¹⁵, r=8, p=1) mit einem 16-Byte-Salt pro Datei — speicherhart, damit eine
  entwendete Backup-Datei nicht mit hoher Rate offline durchprobiert werden kann.
  Dateiformat versioniert (`BKDF`+`0x02`+Salt+Nonce+Ciphertext). Der frühere schwache
  SHA-256-Weg ist **ganz entfernt**: Dateien ohne die `BKDF`-Kennung werden abgelehnt,
  nicht mehr schwach entschlüsselt (`internal/backupkrypto`).
  **Folge für den Betrieb:** Backups von **vor dem 21.08.2026** (Deploy von 5265698c) sind
  **nicht mehr entschlüsselbar** — lokal wie auf S3. Nach diesem Deploy gibt es bis zum
  nächsten 02:30-UTC-Lauf **kein lesbares Backup**; deshalb direkt nach dem Deploy einen
  manuellen Lauf anstoßen (`docker compose exec backend ./main` kennt keinen Schalter —
  kürzester Weg: `docker compose exec bibliothek-db pg_dump -U postgres bibliothek | gzip >
  backups/manuell_$(date +%F).sql.gz` und die Datei nach Eingang des ersten scrypt-Backups
  löschen, sie ist unverschlüsselt). Alte `.enc`-Dateien und S3-Kopien entsorgen.
- Rotation: die letzten **14** Backups bleiben erhalten.
- Optionaler Offsite-Upload nach S3, falls `S3_ENDPOINT`/`S3_ACCESS_KEY`/`S3_SECRET_KEY`/`S3_BUCKET` gesetzt sind.

> ⚠️ **Wichtig:** Diese Dateien sind verschlüsselt. `zcat`/`gunzip`/`psql` funktionieren darauf **nicht**
> direkt — sie müssen zuerst mit dem `restore-backup`-Tool entschlüsselt werden (siehe Abschnitt 2a).
> Ohne den originalen `BACKUP_ENCRYPTION_KEY` ist ein verschlüsseltes Backup **nicht** wiederherstellbar.

### 1b. Die beiden Shell-Wege — seit 23.08.2026 ebenfalls verschlüsselt

Bis zum 23.08.2026 legten beide Skripte **unverschlüsselte** Dumps ab (7 bzw. 30 Tage) —
jeder Schülername, jede Adresse, jede Ausleihe im Klartext, geschützt allein durch `0600`.
Wer den Datenträger, ein Datei-Backup des Servers oder das Verzeichnis in die Hand bekam,
las alles ohne Passphrase. Das war Befund **A5** der Datenschutz-Bewertung
(`docs/datenschutz_offene_punkte.md`).

Beide verschlüsseln jetzt über **dieselbe Ableitung wie der nächtliche Job** — Werkzeug
`cmd/encrypt-backup` im Backend-Container, Helfer `scripts/backup_krypto.sh`. Der
Schlüssel bleibt dabei im Container: `docker exec` reicht ihn nicht durch, das Werkzeug
liest ihn aus der eigenen Umgebung. Auf dem Host taucht er weder in der Prozessliste noch
in einer Variablen auf.

**`scripts/backup.sh`** — Ad-hoc-Sicherung, `backups/bibliothek_backup_<DATUM>.sql.gz.enc`,
7-Tage-Rotation, `pg_dump` per `docker exec` im DB-Container. Die Verschlüsselung sitzt
IN der Pipe (`pg_dump | gzip | encrypt-backup`), der Klartext berührt die Platte nie.
Seit dem 06.08.2026 mit `pipefail`: Ohne ihn lieferte die Pipe den Status des letzten
Glieds, und `gzip` gelingt auch dann, wenn `pg_dump` abgebrochen ist — das Skript meldete
„Backup erfolgreich" und legte eine gzip-Datei mit einer Fehlermeldung darin ab.

**`./update.sh`** — legt vor **jedem** Deploy `backups/backup_<ZEITSTEMPEL>.sql.gz` an und
nennt diese Datei in seiner Rollback-Anleitung. Diese eine Datei entsteht **bewusst im
Klartext**: Sie ist der Rückweg für genau das Zeitfenster, in dem der neue Container nicht
hochkommt — und in dem damit auch das Verschlüsselungswerkzeug nicht erreichbar wäre.
Ist der Deploy gesund (Schritt 4/4b bestanden), verschlüsselt **Schritt 5** sie und löscht
den Klartext. Offen liegt sie damit für die Dauer eines Deploys statt 30 Tage.

**Der Rückweg wird bewiesen.** Beide Wege schicken die fertige `.enc`-Datei durch das
`restore-backup` im Container, bevor sie Erfolg melden oder einen Klartext-Dump löschen
(`pruefe_enc_rundweg`). Eine Formprüfung allein reichte nicht: Eine beim Schreiben
abgeschnittene Datei trägt ihre `BKDF`-Kennung und sieht vollständig aus.

**Fristen und Ausnahmefall.** `.enc` rotiert wie gehabt (7 bzw. 30 Tage). Was als Klartext
liegen bleibt — weil ein Deploy fehlschlug oder die Verschlüsselung nicht möglich war —
wird nach **2 Tagen** gelöscht, und beide Skripte sagen bei jedem Lauf, wie viele solcher
Dateien noch da sind. Ist die Verschlüsselung nicht möglich (Container aus, Schlüssel
nicht gesetzt, altes Image), bricht `scripts/backup.sh` **nicht** ab: Ein lesbares Backup
ist besser als keines. Es benennt den Zustand und setzt die kurze Frist.

> ⚠️ Ein Klartext-Dump bleibt der gesamte Bestand in lesbarer Form. Solange einer in
> `backups/` liegt, gehört das Verzeichnis zum schutzbedürftigen Bestand.

Beispiel-Crontab für den Ad-hoc-Weg:

```bash
0 2 * * * /Pfad/zu/Bibliothek/scripts/backup.sh >> /Pfad/zu/Bibliothek/backups/backup.log 2>&1
```

## 2. Wiederherstellung (Recovery)

Allgemeine Schritte: (1) Anwendung stoppen, (2) Backup auswählen, (3) Datenbank neu erstellen und einspielen,
(4) Anwendung neu starten. Das Einspielen unterscheidet sich je nach Backup-Format.

> Der Backup-Restore-Round-Trip (Verschlüsselung ↔ Entschlüsselung) ist durch automatisierte Tests
> abgesichert: `go test ./jobs/ -run TestBackupRestore`. Vor einem produktiven Go-Live sollte zusätzlich
> **einmal** ein echtes Restore in eine Wegwerf-Datenbank durchgespielt werden (siehe 2e).

> **Kein Platzhalter in den Befehlen.** Dateiname und Schlüssel stehen in Shell-Variablen.
> Eine Anleitung mit spitzen Klammern wurde am 06.08.2026 wörtlich eingefügt und legte die
> Produktion lahm ([SECURITY.md](SECURITY.md), Abschnitt „`APP_ENCRYPTION_KEY` wechseln").
> Hier wöge derselbe Fehler schwerer: Wer `dropdb` ausführt und **erst danach** merkt, dass
> der Restore-Befehl nicht läuft, steht vor einer leeren Datenbank.
>
> **Deshalb die Reihenfolge unten: erst entschlüsseln und prüfen, dann löschen.**

### 2a. Verschlüsseltes `.sql.gz.enc`-Backup (Abschnitt 1a)

Alles in **derselben** Shell-Sitzung, damit `$KEY` und `$DUMP` erhalten bleiben.

```bash
# 0. Restore-Tool: liegt seit 22.08.2026 im Image (`docker compose exec backend ./restore-backup …`).
#    Außerhalb des Containers (Entwicklungsrechner mit Go) einmalig bauen:
go build -o restore-backup ./cmd/restore-backup

# 1. Backup auswählen — neuestes verschlüsseltes Backup
ENC=$(ls -t backups/backup_*.sql.gz.enc | head -1)
echo "Verwende: $ENC"

# 2. Schlüssel setzen (der ORIGINALE aus der Zeit des Backups)
read -rsp "BACKUP_ENCRYPTION_KEY: " KEY; echo

# 3. Entschlüsseln in eine Datei — noch wird nichts gelöscht
DUMP=wiederherstellung.sql
BACKUP_ENCRYPTION_KEY="$KEY" ./restore-backup "$ENC" "$DUMP"

# 4. Gegenprobe VOR dem Löschen: hat die Datei Inhalt und sieht sie aus wie ein pg_dump?
ls -lh "$DUMP"
head -5 "$DUMP"
grep -c "CREATE TABLE" "$DUMP"     # muss deutlich > 0 sein

# (Ein früherer Schritt 4b entfernte `SET transaction_timeout` aus pg_dump-17-Dumps von
# vor dem 22.08.2026 — diese Dateien sind seit dem scrypt-Umstieg ohnehin nicht mehr
# entschlüsselbar, der Schritt ist gegenstandslos. Neue Backups kommen von Client 16.)
```

Erst wenn Schritt 4 plausibel aussieht, die Datenbank ersetzen:

```bash
# 5. Sicherheitsnetz: aktuellen Stand wegsichern (der Rückweg, falls der Restore misslingt)
pg_dump -U postgres bibliothek > vor-restore.sql
ls -lh vor-restore.sql

# 6. Datenbank neu anlegen und einspielen
dropdb -U postgres bibliothek
createdb -U postgres bibliothek
psql -U postgres -d bibliothek -f "$DUMP"
```

### 2b. Backups aus den Shell-Wegen (Abschnitt 1b)

**Der Regelfall — `.enc`:** identisch zu Abschnitt 2a, nur der Dateiname unterscheidet
sich (`bibliothek_backup_<DATUM>.sql.gz.enc` bzw. `backup_<ZEITSTEMPEL>.sql.gz.enc`).

**Der Klartext-Fall — `.sql.gz`:** Solche Dateien entstehen nur noch im Ausnahmefall
(fehlgeschlagener Deploy, Verschlüsselung nicht möglich). Sie lassen sich weiterhin direkt
einspielen — das ist ihr Zweck:

```bash
GZ=$(ls -t backups/*.sql.gz | head -1)
echo "Verwende: $GZ"
zcat "$GZ" | head -5                 # Gegenprobe: echter SQL-Text, keine Fehlermeldung

pg_dump -U postgres bibliothek > vor-restore.sql   # Rückweg
dropdb -U postgres bibliothek
createdb -U postgres bibliothek
zcat "$GZ" | psql -U postgres -d bibliothek
```

> Die Gegenprobe mit `head` ist hier nicht Zierde: `scripts/backup.sh` legte vor dem
> 06.08.2026 ohne `pipefail` auch dann eine gzip-Datei an, wenn `pg_dump` abgebrochen war —
> darin steht dann eine Fehlermeldung statt eines Dumps (Abschnitt 1b).

> Nach getaner Arbeit **löschen** (`shred -u`). Die Skripte tun das nach 2 Tagen von
> selbst, aber bis dahin liegt der ganze Bestand lesbar da.

### 2c. Der Rückweg

Misslingt der Restore, führt `vor-restore.sql` aus Schritt 5 zurück auf den Stand von
vorher:

```bash
dropdb -U postgres bibliothek
createdb -U postgres bibliothek
psql -U postgres -d bibliothek -f vor-restore.sql
```

Danach die Anwendung neu starten. Ist auch das nicht möglich, bleibt das nächstältere
verschlüsselte Backup — es liegen 14 Stück vor (Abschnitt 1a).

### 2d. Aufräumen — erst nach bestätigter Wiederherstellung

`wiederherstellung.sql` und `vor-restore.sql` sind **unverschlüsselte** Dumps mit jedem
Schülernamen, jeder Adresse und jeder Ausleihe im Klartext (dieselbe Warnung wie in
Abschnitt 1b). Sie sind Arbeitsmaterial, kein Backup.

Solange die Wiederherstellung nicht bestätigt ist, `vor-restore.sql` **behalten** — es ist
der einzige Weg zurück. Läuft die Anwendung wieder und ist stichprobenartig geprüft:

```bash
shred -u wiederherstellung.sql vor-restore.sql 2>/dev/null \
  || rm -f wiederherstellung.sql vor-restore.sql
```

### 2e. Restore-Probe vor Go-Live (dringend empfohlen)

Ein Backup, das nie zurückgespielt wurde, ist kein verlässliches Backup.

> **Automatisch läuft das bereits wöchentlich** (`jobs/restore_probe.go`, So 03:30 UTC):
> Der Job entschlüsselt das jüngste Backup, spielt es in eine Wegwerf-Datenbank
> (`bibliothek_restore_probe_wegwerf`) ein, zählt die Tabellen und meldet das Ergebnis
> als Befund der Betriebsbereitschafts-Seite (fehlgeschlagen oder älter als 9 Tage =
> kritisch = tägliche Alarm-Mail). Die manuelle Probe unten bleibt trotzdem sinnvoll:
> Sie prüft zusätzlich den kompletten Weg am **echten Zielsystem** (fremder Server,
> `restore-backup`-Tool, Cover-Reset) — nicht nur, dass die Datei sich einspielen lässt.

Diese manuelle Probe fasst die Produktivdatenbank **nicht** an:

```bash
ENC=$(ls -t backups/backup_*.sql.gz.enc | head -1)
read -rsp "BACKUP_ENCRYPTION_KEY: " KEY; echo

createdb -U postgres bibliothek_restore_test
BACKUP_ENCRYPTION_KEY="$KEY" ./restore-backup "$ENC" \
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

Neben der Ausleihen- und Rückgaben-Historie werden kritische administrative Eingriffe im System protokolliert (append-only als Konvention — kein Bedien- oder Codepfad ändert Einträge, außer der DSGVO-PII-Tilgung; kein Trigger-Zwang, siehe Migration 083 / FACHKONZEPT §10).

- Die Tabelle `audit_logs` speichert dabei unter anderem:
  - Wer (Admin-ID) hat die Aktion durchgeführt?
  - Wann (Zeitstempel) wurde die Aktion ausgeführt?
  - Was (Aktion, z.B. `OVERRIDE_BLOCK`, `RECEIVE_ITEM`, `DELETE_STUDENT`) wurde getan?
  - Details im JSON-Format für tiefergehende Analysen.
- Dies stellt sicher, dass manuelle Sperr-Aufhebungen oder Wareneingänge jederzeit nachvollzogen werden können.
