# Deployment Guide

> Last updated: 2026-06-24

---

## Overview

The system consists of:
- **Go Backend** (Port 8083 production / 8084 local)
- **PostgreSQL 15/16**
- **Caddy** as Reverse-Proxy (TLS termination)
- **Docker Compose** as orchestration

---

## 1. Environment Variables (Required)

All secrets are provided via environment variables. **Never commit secrets to the `.env` of the repo.**

| Variable | Description | Requirement |
|---|---|---|
| `DATABASE_URL` | PostgreSQL DSN | Required |
| `JWT_SECRET` | HMAC signature key | Required, ≥ 32 characters |
| `APP_ENCRYPTION_KEY` | AES-256 key for student photos | Required, exactly 32 bytes |
| `APP_ENV` | Environment (`production` / `local`) — controls Cookie-Secure & Swagger | Default: `production` |
| `ENFORCE_PROD_SECRETS` | Hard start-refusal with default secrets | Default: `false` (Testing phase) |
| `COOKIE_SECURE` | `true` behind TLS proxy (Caddy) | Default: `false` |
| `PORT` | HTTP port of the backend | Required |
| `SMTP_HOST` | SMTP server | Optional (Dunning process) |
| `SMTP_PORT` | SMTP port | Default: 587 |
| `SMTP_USER` | SMTP username | Optional |
| `SMTP_PASSWORD` | SMTP password | Optional |
| `SMTP_FROM` | Sender address | Optional |
| `SMTP_ALLOW_INSECURE_TLS` | Disable TLS certificate check | Only for legacy SMTP servers |
| `INITIAL_ADMIN_EMAIL` | Email of the initial admin | Default: pflasch@philipp-reis-schule.de |
| `SENTRY_DSN` | Sentry Error Tracking | Optional |

---

## 2. Production Deployment (Hetzner/Docker)

### 2.1 Create `.env` file

Create a `.env` file on the server (not in the repo):

```bash
# /opt/bibliothek/.env
POSTGRES_PASSWORD=<secure-password>
JWT_SECRET=<at-least-32-chars-secret-jwt-key>
APP_ENCRYPTION_KEY=<exactly-32-bytes-aes-key>
APP_ENV=production
ENFORCE_PROD_SECRETS=true   # enable only for real prod deploy
COOKIE_SECURE=true
SMTP_HOST=smtp.example.com
SMTP_USER=user@example.com
SMTP_PASSWORD=<smtp-password>
SMTP_FROM=bibliothek@schule.de
```

### 2.2 Secret Guard (Toggleable)

The hard start-refusal is **decoupled** from `APP_ENV` and is controlled via the dedicated switch `ENFORCE_PROD_SECRETS`:

| Phase | `ENFORCE_PROD_SECRETS` | Behavior |
|---|---|---|
| Test/Pilot operation | `false` (Default) | Stack starts even with default secrets — comfortable testing |
| Real Prod Deploy | `true` | Server **refuses to start** if a known default for `JWT_SECRET` or `APP_ENCRYPTION_KEY` is active |

> **Why decoupled from `APP_ENV`?** `APP_ENV=local` would simultaneously deactivate the cookie `Secure` flag and publicly expose the Swagger docs — undesirable on an internet-accessible test server. With `ENFORCE_PROD_SECRETS`, `APP_ENV=production` remains (secure cookies, no Swagger), while secret hardening can be toggled independently.

Error message with `ENFORCE_PROD_SECRETS=true` + Default Secret:
```
FATAL: JWT_SECRET uses a known default value. Set your own secret
JWT_SECRET (≥32 chars) — or ENFORCE_PROD_SECRETS=false during the testing phase.
```

**Checklist before the first real Prod Deploy:** Set `ENFORCE_PROD_SECRETS=true` and provide real values for `JWT_SECRET`, `APP_ENCRYPTION_KEY`, `POSTGRES_PASSWORD` as well as `COOKIE_SECURE=true` (behind Caddy HTTPS).

### 2.3 Start Docker Compose

```bash
cd /path/to/bibliothek
docker compose --env-file .env up -d --build
```

`docker-compose.yml` provides convenient defaults for all secrets (`${VAR:-…}`) so the stack starts during the testing phase without further configuration. Production security is handled by the code guard (`ENFORCE_PROD_SECRETS=true`), not the compose file.

### 2.4 Deployment Script

The script `scripts/deploy.sh` automates the process:
```bash
./scripts/deploy.sh
```

Executes: `git pull` → `docker compose up -d --build` → reload Caddy if necessary.

---

## 3. Caddy Reverse Proxy

Bibliothek runs behind Caddy as a TLS proxy in the Docker network `caddy_global_net`.

### Caddyfile entry
```caddyfile
flasch3.herzog-dupont.de {
    reverse_proxy bibliothek-backend:8083
}
```

### Zero-Downtime Reload
```bash
# If Caddy runs as a Docker container:
docker exec caddy caddy reload -c /etc/caddy/Caddyfile

# If Caddy runs as a systemd service:
systemctl reload caddy
```

**Important:** `restart` instead of `reload` would sever active connections of other services.

---

## 4. Local Development (docker-compose.local.yml)

```bash
docker compose -f docker-compose.local.yml up -d
```

- Backend: `http://localhost:8084`
- PostgreSQL: `localhost:5434`
- `APP_ENV=local` → Default secrets from `docker-compose.local.yml` are allowed
- `COOKIE_SECURE=false` → no TLS needed

The local compose file already contains valid development secrets (≥32 chars), which are intentionally allowed in the repo — they apply **only** for `APP_ENV=local`.

---

## 5. Database Migrations

Migrations run **automatically on server start** (`database.RunMigrations`). Manual intervention is only necessary in case of problems.

### Migration Directory: `migrations/`

| File | Content |
|---|---|
| `030_ziel_jahrgang.sql` | LMF multi-level deadlines; idempotent (both cases: column exists / does not exist) |
| `032_reconcile_titel_columns.sql` | Idempotent alignment of all `buecher_titel` columns (fixes schema drift from old deployments) |
| `033_unique_active_loan.sql` | Deduplication of existing duplicates + unique partial indices for active loans |

### Add a new migration
1. Create file `migrations/NNN_description.sql` (NNN = next number, no name conflict)
2. Add hash in `schema.sql` under `schema_migrations` (will be checked automatically on next start)

---

## 6. Backup & Recovery

Automatic backup cronjob daily at 02:30 AM (configurable in `jobs/cron.go`):

```
pg_dump → gzip → AES-GCM encryption (random nonce) → 0600 on disk
```

Backup rotation: oldest files are deleted after the retention window expires.

Manual backup:
```bash
./scripts/backup.sh
```

### Backup Scope: Database only (conscious decision, 11.07.2026)

The `uploads/` volume (book covers as local WebP files) is **intentionally not** backed up:

- **Student photos** are stored encrypted in the database (`schueler_fotos.foto_encrypted`) and are thus covered by pg_dump — no personal data is lost.
- **Covers are reproducible**: The cover sync job (every 6 h + on server start, `internal/service/cover_service.go`) throttles reloading of PENDING/FAILED titles (2 titles/s) from DNB/Google/OpenLibrary. **Attention:** The job skips titles with status `FOUND` and a dead `/uploads/` path — after a restore without volume, reset once, then the next run heals everything:
  ```sql
  UPDATE buecher_titel SET cover_status = 'PENDING'
  WHERE cover_url LIKE '/uploads/%';
  ```
- **Labels/PDFs** are generated on-demand and never persisted.

If you want to avoid reloading after a restore (e.g., offline operation), you can additionally backup the volume with `docker run --rm -v bibliothek_uploads:/data alpine tar czf - /data` — it is not mandatory.

---

## 7. Health Check & Monitoring

The Docker container contains a built-in Health Check:
```yaml
healthcheck:
  test: ["CMD-SHELL", "pg_isready -U postgres -d bibliothek"]
  interval: 5s
  timeout: 5s
  retries: 5
```

Optional: Sentry integration for error tracking via `SENTRY_DSN`.
