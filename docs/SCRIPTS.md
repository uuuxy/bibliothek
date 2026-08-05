# Kommandozeilen-Skripte und Werkzeuge

---

## 1. LITTERA-Altbestand (`cmd/littera-altbestand`)

Überträgt den Littera-Altbestand nach PostgreSQL. Quelle sind `mdb-export`-CSVs, nicht
die Access-Datei selbst: Deren ODBC-Treiber gibt es nur unter Windows.

```bash
# 1. Export erzeugen (mdbtools, plattformunabhängig)
for t in Titel Exemplar Verlag Medienart Personen Personen_Zuordnung Leser Leser_UG Verleih; do
  mdb-export littera_sav.mdb "$t" > "littera-export/$(echo "$t" | tr 'A-Z' 'a-z').csv"
done

# 2. Trockenlauf – liest und berichtet, schreibt nichts
go run ./cmd/littera-altbestand -csv ./littera-export -trocken

# 3. Übernahme
go run ./cmd/littera-altbestand -csv ./littera-export -db "$DATABASE_URL"
```

**Was übernommen wird**, steuern `-bestand` (Vorgabe an), `-personen` und `-ausleihen`
(Vorgabe aus). Die Vorgabe ist Absicht: Der geprüfte Export ist ein Stand von 2010
(Schüler-Geburtsjahre 1989–2001, letzte Ausleihe 2010). Personen und Ausleihen daraus
legen Schüler an, die heute Mitte dreißig sind, und melden ein Viertel des Bestands als
verliehen. Für einen aktuellen Export sind beide Schalter richtig.

**Barcodes:** `-barcodes littera` (Vorgabe) schreibt den Wert, den ein Scanner vom
Buchetikett liest — eine **EAN-13**, nicht die daneben gedruckte Exemplarnummer:

```
1 0 5 7 8 5   0   0 3 9 5   6   7      Exemplar-Nr. 105785 → 1057850039567
└─ Exemplarnr └─ Bibl.-Nr ─┘   └ Prüfziffer
   6-stellig       0395
```

Gemessen an zwei echten Büchern der Schule, Prüfziffer jeweils verifiziert (siehe
`littera.EtikettBarcode`). Der Bestand bleibt damit ohne Neubeklebung scannbar. Eine
zweite Etikettengeneration gibt es nicht mehr: Die alten Littera-Aufkleber (Zeichenkette
für eine Barcode-Schrift, `8 *pkpööp#-c.bc-*`) sitzen auf Büchern, die nicht mehr im
Regal stehen — bestätigt am 04.08.2026.
`-barcodes neu` vergibt stattdessen frische `B-XXXXX` aus `barcode_seq` — derselben
Sequenz, aus der die Anwendung ihre Barcodes zieht — und setzt voraus, dass jedes Buch
ein neues Etikett bekommt.

Trägt Litteras Tabelle `FremdBarcode` für ein Exemplar ein Ersatzetikett, gewinnt das
gegen die gerechnete EAN-13. Dasselbe gilt für Schülerausweise: `FremdLeserNummer` hält
die Nummer des Kartenherstellers (`B97601826457`), und die steht in keinem Stammdatenfeld.
Beide Dateien sind optional; fehlen sie, rechnet der Import mit den Littera-Nummern.

**Härtung:** Ein Savepoint je Datensatz (`internal/uebernahme`), Postgres-Fehler nach
SQLSTATE eingeordnet, Abwertungen (verworfene ISBN, gekürztes Feld) getrennt von
Ausfällen protokolliert nach `littera_import.log`. Nach jedem Abschnitt wird die gemeldete
Zahl gegen den tatsächlichen Zeilenzuwachs gehalten.

**Wiederholung:** Ein zweiter Lauf wird abgelehnt — es gibt keinen natürlichen Schlüssel,
an dem Postgres die Dublette erkennen könnte. Zum Aufräumen:

```sql
DELETE FROM ausleihen a USING buecher_exemplare e
  WHERE a.exemplar_id = e.id AND e.erweiterte_eigenschaften ? 'littera_id';
DELETE FROM buecher_exemplare WHERE erweiterte_eigenschaften ? 'littera_id';
DELETE FROM buecher_titel     WHERE erweiterte_eigenschaften ? 'littera_id';
DELETE FROM ausleihen WHERE schueler_id IN (SELECT id FROM schueler WHERE lusd_id LIKE 'littera:%');
DELETE FROM schueler  WHERE lusd_id LIKE 'littera:%';
DELETE FROM benutzer  WHERE rolle = 'lehrer' AND email LIKE '%@littera.invalid';
```

**Lehrkräfte** werden mit `aktiv = true` angelegt — die Ausweis-Abfrage der Omnibox
filtert darauf, eine inaktive Lehrkraft wäre am Scanner unauffindbar. Den Login sperrt
statt dessen die Adresse: Anmeldung geht ausschließlich über IMAP gegen den Schul-Mail-
server, und `littera-4908@littera.invalid` gibt es dort nicht. `-lehrer-inaktiv` kehrt das
um, macht die Karten aber wertlos.

**Rückgabewerte:** 0 vollständig · 1 abgebrochen · 2 unvollständig (Details im Protokoll).

> Das frühere `cmd/littera_migration` ist entfallen. Es fragte `SELECT TitelID, Titel,
> Autor, ISBN, Verlag, Jahr, Signatur FROM TITEL` ab — von diesen Spalten existiert in
> einer echten Littera-Datei einzig `ISBN`. Es ist nie gegen echte Daten gelaufen.

---

## 2. Foto-Migration (`cmd/migrate-fotos`)

Migriert unverschlüsselte Bilddateien vom Dateisystem in die Datenbank.

- **Funktionsweise:** Iteriert über ein Verzeichnis mit Schülerfotos, validiert und verschlüsselt diese (AES-256-GCM), speichert sie als `BYTEA` in `schueler_fotos`.
- **Zweck:** Konsolidierung der Infrastruktur (kein separates Foto-Verzeichnis) + Datensicherheit.

---

## 3. Datenbank-Backup (`scripts/backup.sh` / `jobs/backup.go`)

Periodische Datenbank-Backups.

- **Manuell:** `./scripts/backup.sh`
- **Automatisch:** Täglich 02:30 Uhr via internem Scheduler (`jobs/cron.go`)
- **Pipeline:** `pg_dump → gzip → AES-GCM-Verschlüsselung (Zufalls-Nonce) → 0600 auf Disk`
- **Rotation:** Älteste Dateien werden nach Ablauf des Aufbewahrungsfensters gelöscht.

---

## 4. Deployment (`scripts/deploy.sh`)

Automatisiert das Produktions-Deployment auf dem Hetzner-Server.

```bash
./scripts/deploy.sh
```

Führt aus:
1. `git pull` (aktuellsten Stand ziehen)
2. `docker compose up -d --build` (Container neu bauen, Zero-Downtime für andere Dienste)
3. Prüft ob Caddy-Konfiguration den Domain-Block enthält, hängt ihn ggf. an

---

## 5. Concurrency-Lasttest (`cmd/stresstest`)

Simuliert Race Conditions für parallele Barcode-Scans.

```bash
go run cmd/stresstest/main.go -port 8084
```

- Feuert via `sync.Cond` + Goroutinen zeitgleich Dutzende Requests gegen `/api/action`
- Zweck: Verifikation der Transaktionssicherheit (FOR UPDATE + Unique Partial Index)

---

## 6. Paket-Utilities (`pkg/`)

### `pkg/csvutil`
CSV-Formel-Injection-Schutz (OWASP CWE-1236):
```go
import "bibliothek/pkg/csvutil"

safeRow := csvutil.SanitizeRow([]string{titel, autor, ...})
```
Setzt Apostroph-Präfix bei Zellen die mit `= + - @ \t \r \n` beginnen.

### `pkg/imageutil`
Decompression-Bomb-Guard:
```go
import "bibliothek/pkg/imageutil"

if err := imageutil.GuardImageDimensions(r.Body, 50_000_000); err != nil {
    // Bild zu groß oder ungültig
}
```
Liest nur den Bild-Header (`image.DecodeConfig`) — ohne volle RAM-Allokation. Limit: 50 Megapixel.

---

## 7. Übrige Skripte in `scripts/` (Kurzübersicht)

Bis zum 05.08.2026 beschrieb dieses Dokument nur die großen Werkzeuge; die folgenden
Skripte lagen undokumentiert im Verzeichnis. Sie sind bewusst kurz gehalten — der
ausführliche Kommentar steht jeweils im Dateikopf.

### Qualitäts-Gates (lokal, es gibt dafür keinen CI-Job)

| Skript | Zweck |
|---|---|
| `api_inventar.sh` | Erzeugt `docs/api_inventar.md`: alle registrierten Go-Routen, alle `/api/`-Aufrufer im Frontend und den Abgleich in beide Richtungen (tote Handler / Geister-Aufrufe). |
| `deadcode_gate.sh` | Gate gegen unerreichbaren Go-Code (`x/tools/cmd/deadcode`), Erreichbarkeit ab allen `main`-Paketen. |
| `sonar_scan.sh` | SonarQube-Analyse **inklusive** Coverage. Ein bloßer `sonar-scanner`-Aufruf lädt keine Coverage hoch — fehlende Coverage zählt dort als 0 %. |
| `install-hooks.sh` | Installiert `scripts/git-hooks/` (pre-commit, pre-push) in `.git/hooks`. |

### Datenbank-Helfer

| Skript | Zweck | Vorsicht |
|---|---|---|
| `seed_demo.sql` | Realistischer Demo-Datensatz für Pilot und Schulung. | Nur auf Test-/Demo-Datenbanken. |
| `seed_loadtest.sql` | Datenbestand für den k6-Lasttest. | Nur auf Wegwerf-Datenbanken. |
| `tabula_rasa.sql` | Bereinigt die Datenbank für den Echtbetrieb (Bewegungsdaten raus). | **Löscht Daten.** Vorher Backup. |
| `repair_titel_dubletten.sql` | Räumt Titel-Dubletten aus dem Import auf. | Vorher Backup, Ergebnis prüfen. |
| `repair_titel_ortssuffix.sql` | Entfernt Ortssuffixe aus Titelfeldern (Import-Artefakt). | Vorher Backup. |
| `signatur_report.sql` | Report zur Signatur-Harmonisierung nach Littera-Import (Migration 038). | Nur lesend. |

### Einmal-Werkzeuge (`//go:build ignore`, per `go run` gestartet)

| Skript | Zweck |
|---|---|
| `import_isbns.go` | Nachträglicher ISBN-Import in bestehende Titel. |
| `migrate_photos.go` | Überträgt Schülerfotos aus Dateien (`<barcode>.jpg`) in die verschlüsselte Ablage. |
| `monitor_stats.sh` | Protokolliert Systemkennzahlen über ~6 Stunden (Begleitung von Lasttests). |
