# Changelog

---

## 2026-07-08 — E2E Testing, Session Restore & Bugfixes

- **E2E Testing (Playwright)**: Initial smoke flows implemented for core functions (Login/Logout, Date validation, Omnibox scan to student account) including local Docker test stack.
- **Session Restore**: SPA boot restores session via `GET /api/auth/me`. Server-side logout is now fully implemented.
- **Bugfixes**:
  - JWTs without `jti` collided during logins in the same second (fixed).
  - Missing CSRF token after initial login led to 403 (now bootstrapped via `GET /api/csrf-token`).
  - A NULL value in `system_einstellungen` caused a 500 error at checkout (fixed with `coalesce`).
  - 501 stub for parent email dispatch removed from UI (data privacy).
- **PR Triage**: Old PRs cleaned up and consolidated (6 merged, 10 closed).

---

## 2026-07-07 — Dead Code Cleanup & Stabilization

- **Dead Code Cleanup (Phase 1)**: 11 dead Go handlers and routes, as well as 16 unused Svelte files (GlobalScanner, KioskMode, etc.) deleted. Unwired "Undo-Return" feature completely removed.
- **Dunning Process Bugfix**: A slice reallocation bug that swallowed media of students with the same name was fixed via an index-based grouper.
- **Auth Lifecycle**: Session refresh loop in frontend (`authStore`) implemented via `POST /api/auth/refresh` (30-minute tick).
- **Inventory Permissions**: RBAC guards for `RequireViewBooks` / `RequireEditBooks` were consolidated by name.
- **Goods Receipt / Ordering**: Extensive unit and frontend tests (13 Vitest tests) added for the shopping cart.
- **Report Date Boundaries**: Date helpers outsourced (`lib/utils/dates.js`) and timezone/leap-year bugs fixed via regression tests.

---

## 2026-06-25 — Secret Guard via Toggle (Test Mode Unlocked)

**Request:** The hard start-refusal should be off during the testing/pilot phase and only turned on "as needed" for real prod deploys.

**Problem with previous approach:** The guard was tied to `APP_ENV` (`APP_ENV=local` deactivated it). This would have undesirable side effects on the test server because `APP_ENV=local` additionally deactivates the `Secure` cookie flag and exposes the Swagger docs. Furthermore, `docker-compose.yml` blocked startup via `${VAR:?}` before the app even ran.

**Solution — Decoupling via dedicated switch `ENFORCE_PROD_SECRETS`:**
- `main.go/loadConfig`: Guard now checks `ENFORCE_PROD_SECRETS=true` instead of `APP_ENV`. `APP_ENV` remains `production` (secure cookies, no Swagger), regardless of secret hardening.
- `docker-compose.yml`: `${VAR:?}` → convenient `${VAR:-…}` defaults; new pass-through `ENFORCE_PROD_SECRETS=${ENFORCE_PROD_SECRETS:-false}`. Stack starts in testing phase without configuration.
- `.env.example`: `APP_ENV` + `ENFORCE_PROD_SECRETS` documented.

**Before real prod deploy** in `.env`: `ENFORCE_PROD_SECRETS=true` + real `JWT_SECRET` (≥32 chars), `APP_ENCRYPTION_KEY` (32 Bytes), `POSTGRES_PASSWORD`, `COOKIE_SECURE=true`.

Files: `main.go`, `docker-compose.yml`, `.env.example`

---

## 2026-06-25 — Class Set Printing Relinked

`ClassPrintStation.svelte` (mass printing of all ID cards for a class) existed but was orphaned (nowhere imported) since a toolbar refactoring. Link restored: "Print class set" button in `StudentDirectoryToolbar.svelte` (new `onprintclass` callback) + render branch in `StudentDirectory.svelte`. Commit: `6e16e7e`

---

## 2026-06-24 — Cover Download Fixed (DNB Bot Barrier)

**Symptom:** Covers were no longer fetched/saved across the board (only sporadically). Even manual ISBN refresh in "Edit Title" returned correct metadata but no cover.

**Root Cause 1 — wrong User-Agent triggers DNB bot barrier (Anubis):**
The DNB-MVB cover API (`portal.dnb.de/opac/mvb/cover`) lies behind the bot barrier "Anubis". This triggers specifically on **browser-like** User-Agents (`Mozilla/5.0 … Chrome …`) and responds with **HTTP 200 + HTML Challenge** instead of the image. The code was previously "upgraded" to exactly such a Chrome UA. Verified: with the simple program UA `Inventur/1.0`, DNB returns the real JPEG (or a clean 404). Fix: `coverFetchUserAgent` reverted to `Inventur/1.0`.

**Root Cause 2 — HEAD heuristic blocked fallbacks + download error cleared URL:**
`beendeSuche` checked cover availability via HEAD. Since Anubis returns HTTP 200, the cover was falsely considered "available" → Google/OpenLibrary fallbacks were skipped → the real download received HTML → `image.Decode` failed → `CoverURL` was unconditionally overwritten with `""` (valid URL was lost).

**Fix:** `beendeSuche` → new `aufloeseCover`: no more HEAD heuristic. Instead, cover sources are **actually downloaded** in priority order (DNB primary → Google → OpenLibrary with `?default=false`); the first decodable source wins. Additionally, Content-Type guard in downloader (rejects `text/html`/`json` responses immediately). Affects both manual ISBN refresh and bulk background sync.

Files: `inventur/cover_downloader.go`, `inventur/metadaten_client.go`

---

## 2026-06-24 — Deep Audit, UI Redesign & Docker Hardening

This session included a complete static code audit of all 46 files, a global UI redesign (153 Svelte components), and production hardening of the deployment infrastructure.

---

### Bugfixes

#### Omnibox / Search (HTTP 500)
- **`LEHRER` Enum Casing**: `buecher_suche` query used `rolle = 'LEHRER'` (UPPERCASE), but DB stores lowercase. Fix: `LOWER(rolle::text) = 'lehrer'`. Commit: `7271bc2`
- **Schema Drift**: Missing columns (`buecher_titel.untertitel`, `ziel_jahrgang`) in older production databases caused 500s. New migration `032_reconcile_titel_columns.sql` idempotently aligns all columns.
- **Schema Migration Seed**: `schema.sql` had 13 missing entries + 1 phantom entry (`007_performance_indexes.sql`). Result: Migrations ran again on deployment, Migration 030 crashed with "column already exists". Fixed in `1c79b2c`.
- **Migration 030 Idempotency**: No longer fails if `ziel_jahrgang` already exists (handles both cases).

#### Book Click in Omnibox
- Clicking a book in Omnibox results opened the general media catalog instead of the book detail page. Fix: `handleSelectBook` in `Router.svelte` now uses `book_detail` tab.

#### Print Bug (ID Card vs. Loan Receipt)
- "Print ID card" falsely displayed the open loan list. Fix: CSS class `.print-receipt-section` in `StudentPrintReceipt.svelte` + `body[data-print-mode] .print-receipt-section { display: none !important }` in `app.css`.

#### Role Permissions (Teacher)
- Login did not return real permissions from `role_permissions` — teachers saw nothing or everything. Fix in `auth/handlers.go`: permissions are loaded directly from DB after login; Admin gets `["*"]`.

---

### Security

#### RBAC Permission Cache
- Transient DB errors during `RequirePermission` were cached as 403. A short DB outage locked out legitimate users for 60 seconds. Fix: only `pgx.ErrNoRows` → cache + 403; DB error → 500, no caching. Commit: `8da7ee9`

#### Brute-Force Limiter
- Key was pure IP. On school NAT, 5 failed attempts would have locked out the entire school. Fix: key changed to `lower(email)|ip`.

#### SMTP STARTTLS (MITM)
- `InsecureSkipVerify: true` was set — all SMTP connections were vulnerable to MITM (credentials, mail content/personal data). Fix: `ServerName` + `MinVersion: TLS 1.2`. Opt-out via `SMTP_ALLOW_INSECURE_TLS=true`. Attachment filename sanitized against CRLF injection.

#### CSRF Vulnerability
- `sync-covers` and `import-bestand` were located under `/api/admin/*` and thus exempted from the global CSRF check. Fix: exemption removed.

#### CSV Formula Injection (CWE-1236)
- CSV exports wrote book titles/authors verbatim. Malicious strings (e.g., `=HYPERLINK(…)`) could be executed as formulas in Excel. Fix: new `pkg/csvutil.SanitizeRow` (Apostrophe prefix).

#### Decompression Bomb Protection
- `image.Decode` allocates the full pixel matrix before dimension check. A crafted 50,000×50,000px JPEG would exhaust RAM. Fix: new `pkg/imageutil.GuardImageDimensions` (reads only header via `image.DecodeConfig`, limit: 50 MP). Both upload paths (photo + cover) use it.

#### Docker Hardening / Secret Guard
- `docker-compose.yml` had fallback defaults for secrets — deployments with forgotten secrets simply continued with repo defaults (complete JWT forgery possible).
- Fix 1: `docker-compose.yml` uses `${VAR:?}` — Docker aborts with a descriptive error message if a variable is missing.
- Fix 2: `main.go/loadConfig` refuses server start outside `APP_ENV=local` if known default secrets are detected.
- Fix 3: `docker-compose.local.yml` receives `APP_ENV=local` + valid 32+ char Dev JWT.

---

### Data Integrity

#### Unique Active Loan (Migration 033)
- No DB protection against two simultaneous active loans on the same copy (race in TOCTOU window). Fix: Dedup existing duplicates + Partial Unique Indexes (`WHERE rueckgabe_am IS NULL`). Unique violation → HTTP 409.

#### Repository: `rows.Err()` after Iteration
- 5 files iterated rows without trailing `rows.Err()` check. A connection drop was treated as success (incomplete results). Fixed.

#### Excel Import Goroutines
- Semaphore was acquired inside the goroutine instead of before → unlimited goroutine creation under load. Fixed. Commit: `8da7ee9`

#### GDPR vs. Dunning Process
- Migration 003 blocked address columns via `RAISE EXCEPTION`. Since these are essential for the dunning process (postal letters), the guard was removed.

---

### Code Quality

#### golangci-lint: 0 Issues
- Previously 11 Issues. Fixed: 2× ST1005 (German error texts lowercased), 3× `os.Setenv` in tests, multiple `errcheck` findings, 1 `ineffassign`. Stray `tmp/export_csv.go` removed, `tmp/` in `.gitignore`.

#### ODBC Dependency Isolated
- `cmd/littera_migration` required `unixODBC` → blocked `go build ./...` and lint without ODBC. Fix: `//go:build odbc` tag. Default build runs without ODBC.

---

### UI/UX Redesign (Flat & Edge-to-Edge)

Complete sweep across 153 Svelte components in 7 batches. Goal: Eliminate card anti-pattern at layout level → flat design, edge-to-edge, separation via `border-b border-gray-200`.

| Batch | Area | Commits |
|---|---|---|
| 1 | MailTemplates, OverdueWidget | `785ecb0` |
| 2 | Orders | `785ecb0` |
| 3 | Student Lists, Audit Logs | `555fa11` |
| 4 | Book File Tabs | `0853476` |
| 5 | Dashboards, Portals, Misc | `7e0c37b` |
| 6 | Inventory Module | `0eed201` |
| 7 | Settings Tabs, Typography | (currently staged) |

Batch 7 (Settings):
- `SystemSettings.svelte` — "Team & Rights" + "System" tabs no longer constrained to `max-w-3xl`
- `SystemSettingsAllgemein.svelte` — `max-w-3xl` → `max-w-5xl`, larger grid spacing
- `SettingField.svelte` — Labels `text-xs uppercase` → `text-sm font-medium text-gray-600`; Inputs → `text-lg`
- `PermissionManager.svelte` — white container removed, 263 → 150 lines (DRY: `PermissionsEditor.svelte` + `permissionMetadata.js` outsourced)

---

### New Files

| File | Purpose |
|---|---|
| `migrations/032_reconcile_titel_columns.sql` | Idempotent schema alignment for older DBs |
| `migrations/033_unique_active_loan.sql` | Unique partial indexes for active loans |
| `pkg/csvutil/csvutil.go` | CSV formula injection protection |
| `pkg/imageutil/webp.go` | Decompression bomb guard |
| `frontend/src/lib/PermissionsEditor.svelte` | DRY subcomponent for Role Toggles |
| `frontend/src/lib/permissionMetadata.js` | Extracted permissions metadata |

---

## 2026-06-20 — Initial Documentation & Deployment Setup

- Initial version of the documentation (`dokumentation/`) created
- Deployment guide for Hetzner server (`DEPLOYMENT.md`)
- Security concept documentation (`SECURITY.md`)
- Architecture overview (`ARCHITECTURE.md`)
- Goods receipt implementation plan (`implementation_plan_wareneingang.md`)
