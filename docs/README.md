# Bibliothek – School Library Software

A web-based management software for school libraries. The system supports the processing of book and hardware rentals using an integrated barcode scanner concept.

---

## Tech Stack

| Component | Technology |
|---|---|
| Backend | Go 1.26.4+, `net/http`, `pgx/v5` |
| Frontend | Svelte 5 (Runes), Tailwind CSS, Vite |
| Database | PostgreSQL 15/16 |
| Real-time | Server-Sent Events (SSE) |
| Deployment | Docker Compose, Caddy (Reverse Proxy) |

---

## Main Features

- **Central Omnibox (Scanner Dispatcher):** A single input field processes barcode scans and assigns actions based on prefixes (`S-` student, `L-` teacher, `B-` book, `G-` device).
- **Deadline Calculation:** Accounts for LMF books (deadline July 31), special inventory (CDs, DVDs, audiobooks), and summer reading clubs.
- **Audit Trail:** Append-only event logging for administrative actions.
- **Privacy Features:** Automated deletion routines for graduating students, AES-256 encryption for student photos.
- **LUSD Interface:** Import of student data from the LUSD system.
- **Hardware Management:** Rental of laptops/tablets including accessory checklists.
- **Print Center:** Generation of barcode labels and student ID cards.
- **Role-Based Access Control (RBAC):** Roles for admin, teachers (configurable permissions), and staff.

---

## Documentation

> **Note:** This is the English documentation. Für die deutsche Version, siehe [README.de.md](README.de.md).

| Document | Content |
|---|---|
| [ARCHITECTURE.md](ARCHITECTURE.md) | Layered architecture, concurrency model, database design, frontend |
| [SECURITY.md](SECURITY.md) | Security concept, GDPR, protective measures |
| [DEPLOYMENT.md](DEPLOYMENT.md) | Production deployment, environment variables, Caddy, backups |
| [INSTALL.md](INSTALL.md) | Local setup |
| [SCRIPTS.md](SCRIPTS.md) | CLI tools and migrations |
| [CHANGELOG.md](CHANGELOG.md) | Change history |
| [resilience_and_recovery.md](resilience_and_recovery.md) | Backup encryption, restore test, disaster recovery |
| [backup_cron.md](backup_cron.md) | Backup cronjob setup |
| [master_fahrplan.md](master_fahrplan.md) | Status document: completed / open / backlog |
| [api_inventar.md](api_inventar.md) | Generated route inventory (`scripts/api_inventar.sh`) |
| [archive/](archive/) | Completed plans and checklists (e.g., MySQL migration, audit sweeps) |

---

## Quickstart (Local)

### Prerequisites
- Go 1.26.4+
- Node.js (npm)
- PostgreSQL (local or via Docker)

### With Docker
```bash
docker compose -f docker-compose.local.yml up -d
```
Backend: `http://localhost:8084` · DB: `localhost:5434`

### Manual

**1. Environment Variables**
```bash
cp .env.example .env
# Adjust DATABASE_URL, JWT_SECRET (≥32 chars), APP_ENCRYPTION_KEY (32 Bytes)
```

**2. Start Backend**
```bash
go run main.go
# Automatically runs database migrations
```

**3. Start Frontend**
```bash
cd frontend
npm install
npm run dev
# → http://localhost:5173
```

---

## System Architecture (Overview)

```
Middleware (Rate-Limit → Auth → CSRF → RBAC)
        │
        ▼
Handler (api/) → Service (internal/service/) → Repository (repository/)
        │                                               │
        ▼                                               ▼
SSE Broker (Real-time)                        PostgreSQL (pgx/v5)
```

Details: [ARCHITECTURE.md](ARCHITECTURE.md)

---

## Security

- JWT HMAC-only (no `alg=none`)
- Brute-force protection: `email|ip` composite key
- CSRF: Double-Submit Cookie
- AES-256-GCM for student photos
- SMTP with TLS certificate validation
- CSV Formula Injection protection (OWASP CWE-1236)
- Decompression bomb guard for image uploads
- Production secret guard (server will not start with default secrets)

Details: [SECURITY.md](SECURITY.md)
