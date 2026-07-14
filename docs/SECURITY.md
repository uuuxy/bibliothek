# Security and Privacy Concept (GDPR)

This documentation describes the system-wide mechanisms for maintaining the security and privacy of the library management software.

> Last updated: 2026-06-24 (Session Audit: deeply scanned all 46 files)

---

## 🛡️ Authentication & Session Management

### JWT (JSON Web Tokens)
- **Algorithm Pinning:** The server exclusively accepts HMAC-signed tokens (HS256). The `alg=none` vulnerability (CVE class) is thus prevented — a token without a signature is rejected.
- **Blacklist (fail-closed):** Logged out tokens are registered in a database blacklist. If the blacklist query is unreachable (DB error), the request is rejected (HTTP 500), not allowed through. "Fail-open" behavior is excluded.
- **Lifespan:** 12 hours; re-login is required afterwards.
- **Cookie Attributes:** `HttpOnly` (no JS access), `SameSite=Lax`, additionally `Secure` in production (via `COOKIE_SECURE=true`).

### Brute-Force Protection (Login)
- **Key:** `lower(email)|ip` — locks an account for an IP address (5 failed attempts / 15 min).
- **Why not just IP?** On a school network NAT, all devices share an IP. If only the IP were blocked, a single failed attempt would lock out the entire school. The composite key (`email|ip`) isolates the affected account on that IP while still protecting against targeted account attacks.
- **Global Rate Limiter:** Additionally 50 requests/s/IP across all endpoints (Map+Mutex, no external cache needed).

---

## 🔒 Authorization (RBAC)

### RequirePermission Middleware
- All sensitive endpoints are secured via `RequirePermission` or `RequireRoles`.
- **No transient 403 caching:** If the database is unreachable during the permission check (network error, timeout), HTTP 500 is returned and **not** written to the permission cache. A temporary DB failure thus does not lock out legitimate users for 60 seconds.
- **Stable Denial:** Only `pgx.ErrNoRows` (permission definitely not present) is cached and evaluated as 403.

### Role Concept
- `admin`: Full access (`["*"]`). Permissions are loaded directly from `role_permissions` upon login.
- `lehrer`: Granular rights — every permission must be explicitly enabled by an admin.
- `mitarbeiter`: Basic rights for front desk operations.
- All Enum values in the database are **lowercase** (`admin`, `lehrer`, `mitarbeiter`). SQL comparisons use `LOWER(rolle::text)` to avoid casing errors (Bugfix: `LEHRER` Enum led to HTTP 500 in the Omnibox).

---

## 🛡️ Protection against Injection Attacks

### SQL Injection
- All database interactions are exclusively executed via parameterized queries (`$1`, `$2`, …) using `jackc/pgx/v5`. String concatenation in SQL statements does not exist.

### CSV Formula Injection (CWE-1236)
- **Attack Vector:** Book titles or author names starting with `=`, `+`, `-`, `@`, `\t`, `\r`, `\n` can be interpreted as formulas in CSV files (Excel/LibreOffice executes these upon opening).
- **Protection:** `pkg/csvutil.SanitizeRow()` prefixes all cells starting with one of these characters with an apostrophe (OWASP recommendation). Used for all CSV exports (`inventur/export_csv.go`, order exports).

### XSS (Cross-Site Scripting)
- Svelte 5 automatically escapes all template variables.
- `{@html}` is not used anywhere in the frontend.
- SVG icons are hardcoded constants, not user-controlled values.

---

## 📁 File Uploads

### Photo Uploads (Student Photos)
- **Decompression Bomb Protection:** `pkg/imageutil.GuardImageDimensions()` reads only the image header via `image.DecodeConfig` (without full decoding). Images over 50 megapixels are rejected before `image.Decode` allocates the full pixel matrix (protection against RAM exhaustion via crafted images).
- **MIME Check:** Via real decoding, not just file extension.
- **Encryption:** Photos are AES-256-GCM encrypted and stored as `BYTEA` in the database — no plaintext on the filesystem.
- **Path Traversal:** All path operations use `filepath.Base` + `filepath.Clean` + Prefix-Guard.

### Cover Uploads
- 10 MB body limit, 0600 file permissions.
- Also `GuardImageDimensions` before full decode.

---

## 📧 Email Security (SMTP/IMAP)

### SMTP STARTTLS
- **Certificate Check Active:** `ServerName` is set, `MinVersion: TLS 1.2` enforced. `InsecureSkipVerify` was previously set to `true` — a MITM attacker could have read SMTP credentials and the entire email content (including personal data for dunning processes). **Fixed.**
- **Opt-out for internal/legacy servers:** Environment variable `SMTP_ALLOW_INSECURE_TLS=true` allows disabling the certificate check if needed (with explicit warning in the log).
- **Header Injection:** Attachment filenames are sanitized against CRLF injection.

### IMAP
- Implicit TLS (Port 993), `MinVersion: TLS 1.2`, ServerName verification, timeouts.

---

## 🔏 CSRF Protection

- **Method:** Double-Submit Cookie with constant-time comparison.
- **Attention (fixed):** `sync-covers` and `import-bestand` are globally registered endpoints under `/api/admin/…`. Due to an overly broad exemption rule for `/api/admin/*`, these were temporarily without CSRF protection. The exemption was removed — both endpoints now pass through the global CSRF check. The frontend already sends the token correctly (no frontend changes needed).

---

## 🐳 Production Safeguards (Secret Guard)

### Problem
If `JWT_SECRET` or `APP_ENCRYPTION_KEY` use the committed development defaults, anyone with repo access can forge admin JWTs (complete takeover) or decrypt AES-encrypted student photos.

### Solution (`main.go/loadConfig`)
The server **refuses to start** if the switch `ENFORCE_PROD_SECRETS=true` is set and known default secrets are detected:
```go
enforceProdSecrets := strings.ToLower(os.Getenv("ENFORCE_PROD_SECRETS")) == "true"
if enforceProdSecrets {
    knownDefaultSecrets := map[string]bool{
        "super-secret-default-key-at-least-32-bytes": true,
        "super-secure-aes-key-32-chars-ok":           true,
        "supergeheim_lokal":                          true,
    }
    // … log.Fatalf on match
}
```

**Consciously toggleable via switch (decoupled from `APP_ENV`):**
- Testing/Pilot phase: `ENFORCE_PROD_SECRETS=false` (Default) → Stack also starts with defaults.
- Real Prod Deploy: `ENFORCE_PROD_SECRETS=true` → Hard start-refusal with default secrets.

The decoupling from `APP_ENV` is intentional: `APP_ENV=local` would otherwise simultaneously deactivate the cookie `Secure` flag and publicly expose Swagger. This way, `APP_ENV=production` remains (secure cookies, no Swagger), while secret hardening is switched separately.

### Minimum Requirements
- `JWT_SECRET`: ≥ 32 characters
- `APP_ENCRYPTION_KEY`: exactly 32 bytes (or 64 Hex characters)
- In production: `docker-compose.yml` enforces via `${VAR:?ErrorMessage}` that all secrets are set

---

## 🔒 Privacy and GDPR Compliance

### Automated Deletion Routines
The application executes automated cronjobs (`jobs/cron.go`):

- **Loan Anonymization (`RunGDPRAnonymizeLoans`):** Removes `bearbeiter_id` from loans returned more than 14 days ago.
- **Graduate Deletion (`RunGDPRDeleteAbgaenger`):** Hard delete of student records (`ist_abgaenger = true`) after a grace period (30 days into the new school year), provided there are no open loans or unpaid damages. Historical loan data is anonymized (`schueler_id = NULL`).

### Data Encryption
- Student photos: AES-256-GCM encrypted as `BYTEA` in the database. No plaintext on the filesystem.
- DB Backups: `pg_dump → gzip → AES-GCM` (random nonce), 0600 file permissions, rotation.

### Address Data (GDPR vs. Dunning)
Address columns (`strasse`, `plz`, `ort`) are required for the dunning process (postal letters) and are **consciously present**. Migration 003 originally contained a `RAISE EXCEPTION` guard that would have blocked address columns — this was removed since the data is technically essential.

---

## 🛡️ Network Security & Security Headers

Restrictive HTTP headers in `api/middleware.go`:
- `frame-ancestors 'none'` — prevents clickjacking via iFrame
- `form-action 'self'` — forms only to own API
- `script-src 'self'` — no external script loading
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`

---

## 📋 Audit Trail

- All administrative actions and book movements are logged in `audit_logs` (Append-Only).
- Auditing happens **after** the transaction commit (no rollback risk).
