# Kommandozeilen-Skripte und Werkzeuge

---

## 1. LITTERA-Import (`cmd/littera_migration`)

Migriert Altbestände aus LITTERA-Exporten in die neue Datenbankstruktur.

- **Funktionsweise:** Verarbeitet CSV-Dumps der LITTERA-Software (Titelinformationen + Barcodes physischer Exemplare).
- **Build-Tag:** Benötigt `unixODBC`. Standard-Build schließt dieses Tool aus — kein ODBC auf dem Server nötig:
  ```bash
  go build -tags odbc ./cmd/littera_migration/...
  ```
- **Architektur:** Transaktionaler Import — Buchtitel (`buecher_titel`) und Exemplare (`buecher_exemplare`) werden atomar angelegt.

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
