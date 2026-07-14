# Command Line Scripts and Tools

---

## 1. LITTERA Import (`cmd/littera_migration`)

Migrates legacy data from LITTERA exports into the new database structure.

- **Operation:** Processes CSV dumps of the LITTERA software (title information + barcodes of physical copies).
- **Build Tag:** Requires `unixODBC`. The default build excludes this tool — no ODBC required on the server:
  ```bash
  go build -tags odbc ./cmd/littera_migration/...
  ```
- **Architecture:** Transactional import — book titles (`buecher_titel`) and copies (`buecher_exemplare`) are created atomically.

---

## 2. Photo Migration (`cmd/migrate-fotos`)

Migrates unencrypted image files from the filesystem into the database.

- **Operation:** Iterates over a directory containing student photos, validates and encrypts them (AES-256-GCM), and stores them as `BYTEA` in `schueler_fotos`.
- **Purpose:** Consolidation of the infrastructure (no separate photo directory) + data security.

---

## 3. Database Backup (`scripts/backup.sh` / `jobs/backup.go`)

Periodic database backups.

- **Manual:** `./scripts/backup.sh`
- **Automatic:** Daily at 02:30 AM via internal scheduler (`jobs/cron.go`)
- **Pipeline:** `pg_dump → gzip → AES-GCM encryption (random nonce) → 0600 on disk`
- **Rotation:** Oldest files are deleted after the retention window expires.

---

## 4. Deployment (`scripts/deploy.sh`)

Automates production deployment on the Hetzner server.

```bash
./scripts/deploy.sh
```

Executes:
1. `git pull` (fetch latest state)
2. `docker compose up -d --build` (rebuild containers, zero-downtime for other services)
3. Checks if Caddy configuration contains the domain block, appends it if necessary

---

## 5. Concurrency Load Test (`cmd/stresstest`)

Simulates race conditions for parallel barcode scans.

```bash
go run cmd/stresstest/main.go -port 8084
```

- Fires dozens of simultaneous requests against `/api/action` via `sync.Cond` + goroutines
- Purpose: Verification of transaction safety (FOR UPDATE + Unique Partial Index)

---

## 6. Package Utilities (`pkg/`)

### `pkg/csvutil`
CSV Formula Injection protection (OWASP CWE-1236):
```go
import "bibliothek/pkg/csvutil"

safeRow := csvutil.SanitizeRow([]string{titel, autor, ...})
```
Prefixes cells starting with `= + - @ \t \r \n` with an apostrophe.

### `pkg/imageutil`
Decompression Bomb Guard:
```go
import "bibliothek/pkg/imageutil"

if err := imageutil.GuardImageDimensions(r.Body, 50_000_000); err != nil {
    // Image too large or invalid
}
```
Reads only the image header (`image.DecodeConfig`) — without full RAM allocation. Limit: 50 Megapixels.
