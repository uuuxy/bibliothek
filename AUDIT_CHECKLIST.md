# 🔍 Audit-Abhakliste — Tiefen-Scan Bibliothek

> Stand: 2026-06-24. Lebendes Dokument. Abgehakt = tief gescannt (+ ggf. behoben).
> Reihenfolge = Priorität (oben = höchstes Risiko).

---

## ✅ Bereits tief gescannt & behoben (diese Session)

- [x] **Auth-Kern** — JWT (HMAC-Pinning, kein `alg=none`), Blacklist (fail-closed), Brute-Force-Limiter (IP-spoofing-sicher), Login/Refresh
- [x] **RBAC Backend** — `RBACBlockMiddleware` entfernt; Autorisierung einheitlich über `RequirePermission`/`RequireRoles` (`d259011`)
- [x] **RBAC Permission-Cache** — transiente DB-Fehler nicht mehr als 403 gecacht (`8da7ee9`)
- [x] **RBAC Ende-zu-Ende** — Login liefert echte `role_permissions`; Frontend-Menü gated darauf (`c9d9283`)
- [x] **Omnibox** — Enum-Casing `LEHRER` 500 (`7271bc2`); Schema-Drift + kaputte Migrationskette (`1c79b2c`)
- [x] **Middleware-Kette** — Reihenfolge, CSRF (Double-Submit + ConstantTime), Rate-Limiter (Map+Mutex)
- [x] **Excel-Import Goroutinen** — Semaphore vor `go` (`8da7ee9`)
- [x] **Transaktions-Bilanz** — Begin/Commit/Rollback über 22 Dateien geprüft (keine Leaks)
- [x] **Startup/Shutdown** — `main.go` Init-Kette, Graceful Shutdown, Re-Entrancy-Guard

---

## 🔴 Priorität 1 — Sicherheits-/datenkritisch

- [ ] **File-Uploads (Foto/Cover)** — Path-Traversal im Pfadbau, Größenlimits, MIME-/Magic-Byte-Validierung, Speicherort-Rechte, Resize-DoS (große Bilder)
- [x] **DSGVO-Konflikt Migration 003 ↔ Adressfeature** — verbietender RAISE-Wächter entfernt; Adressspalten bleiben erhalten (fachlich essenziell), `geburtsdatum` bleibt; Fresh-Deploy läuft ohne Abbruch. Verifiziert gegen echte DB.
- [ ] **Import-Pfade (LUSD / Littera / Excel)** — Injection über CSV/Excel-Inhalte, Teilfehler-Verhalten, Transaktionsgrenzen, Speicher bei ~20k Zeilen, Encoding/BOM
- [ ] **PDF-Erzeugung (Mahnwesen / Abgänger / Schäden)** — Template-Injection, Ressourcen bei großen Listen, ungültige/fehlende Daten, Timeouts
- [ ] **IMAP/SMTP** (`auth/imap.go`, `inventur/backup_email.go`) — Credential-Handling, TLS-Erzwingung, Timeouts, Header-/CRLF-Injection in Mail
- [ ] **CSRF-Lücke `/api/admin/*`** — `sync-covers` & `import-bestand` laufen an beiden CSRF-Systemen vorbei (SameSite-mitigiert, aber DiD schließen)

---

## 🟠 Priorität 2 — Korrektheit / Geschäftslogik

- [ ] **Loan-Service komplett** (`loan_checkout*.go`, `loan_return.go`, `loan_rules.go`) — Fristenberechnung (`ziel_jahrgang`), Fremdrückgabe, Vormerkungs-Übergabe, Sperren/Reaktivierung, Race bei gleichzeitigem Scan
- [ ] **Device-Service** (`device_service.go`) — Geräte-Ausleihe, Checklisten-Pflicht, Zustandsübergänge
- [ ] **Order-/Reorder-Service** — Mengen-/Schwellenlogik (`meldebestand`), Doppelbestellungen
- [ ] **Idempotency-Keys** (`028`, Migration) — tatsächliche Wirksamkeit, Race, TTL/Cleanup
- [ ] **Mahnwesen-Bulk** — asymmetrisches Begin=1/Commit=2/Rollback=3 verifizieren (mehrere Commit-Pfade)
- [ ] **Repository-Layer (24 Dateien)** — Scan-Konsistenz mit Tabellen, NULL-Handling, N+1-Queries, fehlende Indizes

---

## 🟡 Priorität 3 — Robustheit / Betrieb

- [ ] **SSE-Broker** (`sse/`) — Goroutine-/Channel-Leaks bei Client-Disconnect, Backpressure, langsame Clients
- [ ] **Background-Jobs** (`jobs/backup.go`, `antolin_sync.go`, `cron.go`) — Fehlerbehandlung, Überlappungsschutz, Retry, Backup-Integrität
- [ ] **Cover-Sync unter Last** — 20k Titel: Rate-Limits DNB/Google/OpenLibrary, Worker-Pool-Verhalten, FAILED-Retry-Schleifen
- [ ] **Migrations-Hygiene** — doppelte Präfixe (`003_`, `008_`, `021_`, `022_` je 2×) → fragile Reihenfolge; Idempotenz aller ADD-COLUMN-Migrationen
- [ ] **Rate-Limit/Brute-Force-Tuning** — Schwellen, Verhalten hinter Caddy, Shared-IP (Schul-NAT)

---

## 🔵 Priorität 4 — Frontend (153 Komponenten)

- [ ] **Routen-/View-Guards** — nicht nur Menü: lassen sich Views direkt aufrufen (z. B. via State-Manipulation)? Backend schützt → aber UX/Leak prüfen
- [ ] **XSS** — alle `{@html}`-Stellen, ungesäuberte Server-/Nutzerdaten
- [ ] **Offline-Queue (PWA)** — Konsistenz, Doppel-Scans, Sync-Konflikte, Service-Worker-Cache-Invalidierung
- [ ] **API-Fehlerbehandlung** — flächendeckend Toasts/Feedback (kein stilles Scheitern wie früher bei Adresse/ISBN)
- [ ] **Svelte-5-Stores** — Race/Stale-State in Runes, SSE-Reconnect-Logik

---

## ⚪ Priorität 5 — Qualität / Infrastruktur

- [ ] **Testabdeckung** — viele Pakete ohne Tests (`db`, `repository`, `auth`); kritische Pfade (Loan, RBAC, Import) absichern
- [ ] **golangci-lint vollständig** — über gesamtes Repo, nicht nur Teilpakete
- [ ] **ODBC-Abhängigkeit** (`alexbrainman/odbc`, Buildfehler `sql.h`) — wird sie überhaupt gebraucht? Sonst entfernen
- [ ] **Dockerfile/Compose** — Secrets-Handling, Healthchecks, Restart-Policy, `schema.sql`-Init vs. Migrationen bei Volume-Reuse

---

### Legende
`[ ]` offen · `[x]` tief gescannt (+ ggf. behoben, Commit in Klammern)
