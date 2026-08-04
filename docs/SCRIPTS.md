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

**Barcodes:** `-barcodes littera` (Vorgabe) übernimmt Litteras Exemplarnummer — genau die
Nummer, die auf den vorhandenen Etiketten steht (belegt in `littera.BarcodeInhalt`:
61.520 von 61.520 aus der Druckzeichenkette rekonstruiert). Der Bestand bleibt damit ohne
Neubeklebung scannbar, sofern die Lesegeräte den Zifferninhalt liefern — das ist an einem
echten Buch zu prüfen. `-barcodes neu` vergibt stattdessen frische `B-XXXXX` aus
`barcode_seq`, derselben Sequenz, aus der die Anwendung ihre Barcodes zieht.

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
