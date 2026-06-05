# MySQL → PostgreSQL Migration

Dieses Dokument beschreibt das isolierte Migrations-Skript unter [cmd/migrate/main.go](../cmd/migrate/main.go).

## Zweck

Das Skript überführt eine ältere MySQL-Datenbank in das aktuelle PostgreSQL-Schema der Bibliothek. Es ist für einen einmaligen Import oder für kontrollierte Testläufe gedacht und bricht bei fehlerhaften Datensätzen nicht komplett ab.

## Aufruf

```bash
MYSQL_DSN="user:pass@tcp(host:3306)/olddb?parseTime=true" \
PG_DSN="postgres://user:pass@host:5432/newdb" \
go run ./cmd/migrate --dry-run
```

### Optionen

- `--dry-run`: Validiert die Daten und schreibt nur in `migration_errors.log`.
- `--batch N`: Importiert Titel in Blöcken der Größe `N`.

## Umgebungsvariablen

- `MYSQL_DSN`: Verbindungszeichenfolge für die Quell-Datenbank.
- `PG_DSN`: Verbindungszeichenfolge für die Ziel-Datenbank.

Die Verbindungen werden am Ende des Laufs sauber geschlossen.

## Erwartete MySQL-Struktur

Das Skript erwartet eine ältere Medientabelle mit folgenden Feldern:

- `id`
- `titel`
- `untertitel`
- `autor`
- `isbn`
- `verlag`
- `erscheinungsjahr`
- `beschreibung`
- `medientyp`
- `standort`
- `regal`
- `notizen`
- `anzahl`
- `erstellt_am`

Falls die Quellstruktur anders heißt, muss nur die SELECT-Abfrage in [cmd/migrate/main.go](../cmd/migrate/main.go) angepasst werden.

## Daten-Mapping

Die Migration schreibt in die Zieltabellen `buecher_titel` und `buecher_exemplare`.

### Titel

- `titel`, `untertitel`, `autor`, `isbn`, `verlag`, `erscheinungsjahr`, `beschreibung`, `medientyp` werden direkt zugeordnet.
- Freitextfelder wie `standort`, `regal` und `notizen` werden in `erweiterte_eigenschaften` als JSONB gespeichert.
- Die Legacy-ID aus MySQL wird zusätzlich als `mysql_id` in `erweiterte_eigenschaften` abgelegt.

### Exemplare

- Das Feld `anzahl` bestimmt die Anzahl der physisch angelegten Exemplare.
- Für jedes Exemplar wird ein neuer Barcode im Format `B-[number]` erzeugt.
- Die Barcodes laufen sequenziell ab dem höchsten vorhandenen Barcode in PostgreSQL weiter.

## Validierung

Das Skript prüft während des Imports:

- ISBN-10-Prüfziffern
- ISBN-13-Prüfziffern
- doppelte ISBNs im laufenden Import
- gültige Barcode-Formate

Ungültige Einträge werden nicht importiert, sondern in `migration_errors.log` protokolliert.

## Fehlerbehandlung

- Ein fehlerhafter Datensatz stoppt den Gesamtlauf nicht.
- Die Migration sammelt Warnungen und Fehler in `migration_errors.log`.
- Am Ende wird eine Zusammenfassung mit importierten Titeln, Exemplaren und Fehlern ausgegeben.
- Wenn Fehler aufgetreten sind, beendet sich das Skript mit Exit-Code `2`.

## Empfohlener Ablauf

1. Erst `--dry-run` ausführen.
2. `migration_errors.log` prüfen.
3. Wenn die Validierung sauber ist, den echten Lauf ohne `--dry-run` starten.
4. Danach die Zieltabellen stichprobenartig prüfen.

## Hinweise

- Das Skript ist bewusst isoliert gehalten und verändert keine Laufzeitlogik der Webanwendung.
- Die Barcode-Logik orientiert sich an der bestehenden Bibliotheks-Konvention `B-00001`, `B-00002`, usw.
- Das Skript ist als Startpunkt für eine kontrollierte Datenübernahme gedacht, nicht als generischer ETL-Ersatz.