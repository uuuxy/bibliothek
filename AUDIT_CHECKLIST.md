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

- [x] **File-Uploads (Foto/Cover)** — Path-Traversal überall sicher (`http.Dir` + `filepath.Base` + Clean/Prefix-Guard); MIME via echtem Decode; Fotos AES-verschlüsselt in DB (kein FS); Cover 10 MB-Limit, 0600. **Gefunden+behoben:** Decompression-Bomb-DoS (image.Decode allokiert Pixelmatrix vor Dimensionsprüfung) → zentraler `GuardImageDimensions` (Header-only, 50 MP-Limit) in Foto- und Cover-Pfad. Rest-Hinweis: `DecodeAndValidate` (Foto-Upload) hat kein eigenes Body-Limit (nur global 100 MB) — optional verschärfen.
- [x] **DSGVO-Konflikt Migration 003 ↔ Adressfeature** — verbietender RAISE-Wächter entfernt; Adressspalten bleiben erhalten (fachlich essenziell), `geburtsdatum` bleibt; Fresh-Deploy läuft ohne Abbruch. Verifiziert gegen echte DB.
- [x] **Import-Pfade (LUSD / Littera / Excel)** — Import-SQL voll parametrisiert ($1..$N); ImportDynamic atomar (eine Transaktion, kein Teil-Import); XML via Go-`encoding/xml` (kein XXE); Body-Limits gesetzt. **Gefunden+behoben:** CSV-Formel-Injection (CWE-1236) beim EXPORT (`inventur/export_csv.go`, `api/order_pdf.go` schrieben Titel/Autor verbatim) → neues `pkg/csvutil.SanitizeRow` (Apostroph-Präfix bei `= + - @`/Steuerzeichen) in beiden Exporten.
- [x] **PDF-Erzeugung (Mahnwesen / Abgänger / Schäden)** — sauber: gofpdf programmatisch (keine Template-Injection), `pdf.Output`-Fehler überall behandelt, `UnicodeTranslator` für Umlaute (kein Crash bei Sonderzeichen), Daten permission-gated. Nicht-kritisch: PDFs werden vollständig im RAM gepuffert (bei Schuldatenmengen unbedenklich).
- [x] **IMAP/SMTP** — IMAP sauber (implizit-TLS/993, MinVersion 1.2, ServerName-Verifikation, Timeouts, kein Goroutine-Leak). Mail: To via `mail.ParseAddress`, Subject CRLF-bereinigt. **Gefunden+behoben:** SMTP-STARTTLS lief mit `InsecureSkipVerify: true` (MITM konnte AUTH-Credentials + Mailinhalt/Personendaten mitlesen) → jetzt Zertifikatsprüfung gegen Host (Default sicher, Env-Opt-out `SMTP_ALLOW_INSECURE_TLS`); Attachment-Dateiname gegen Header-Injection bereinigt.
- [x] **CSRF-Lücke `/api/admin/*`** — `sync-covers` & `import-bestand` (global registriert, nicht im Inventur-Modul) waren über den `/api/admin`-Präfix von der globalen CSRF-Prüfung ausgenommen → kein CSRF-Schutz. Ausnahme für diese beiden Pfade entfernt; globale Prüfung greift jetzt. Frontend sendet den Token bereits (keine FE-Änderung nötig).

---

## 🟠 Priorität 2 — Korrektheit / Geschäftslogik

- [x] **Loan-Service komplett** — sauber & hochwertig: Fristen inkl. `ziel_jahrgang`-Mehrjahres-LMF mit sicheren Defaults; `FOR UPDATE` auf Schüler (Limit-Race) UND auf älteste wartende Vormerkung (keine Doppelzuteilung); Neuausleihe/Rückgabe/Fremdrückgabe atomar in einer Tx, Audit nach Commit; `staffRole`-Vergleich nutzt korrekt Uppercase-JWT-Rolle. Keine Korrektheitsfehler.
- [x] **Device-Service** — sauber: Verfügbarkeits-/Sperr-/Aussonderungs-Checks, `FOR UPDATE`-Lock auf aktive Geräte-Ausleihe, Checklisten-Gate für Zubehör, transaktionaler Commit. Gleiche Qualität wie Loan-Service.
- [x] **Order-/Reorder-Service** — saubere Transaktionsstruktur (Begin/Commit/SafeRollback), keine Auffälligkeiten.
- [x] **Idempotency-Keys** — TTL-Cleanup (24h-Cron) vorhanden, 5xx werden nicht gecacht. **Gefunden+strukturell behoben:** Check-then-act-TOCTOU (zwei zeitgleiche Requests mit gleichem Key konnten beide ausführen; `ON CONFLICT` dedupliziert nur die gespeicherte Antwort). Die schlimmste Folge — zwei aktive Ausleihen auf einem Exemplar — ist jetzt per DB-Unique-Index (Migration 033) ausgeschlossen. Rest-Limitierung: vollständig atomare Key-Reservierung bräuchte Schema-Änderung (pending-State) — dokumentiert.
- [x] **Mahnwesen-Bulk** — sauber: eine Transaktion (Begin/Commit/SafeRollback); die früher vermutete Asymmetrie war ein Grep-Artefakt (deutsche Kommentarwörter). PDF wird IN der Tx erzeugt → bei PDF-Fehler wird der Mahnstufen-Bump zurückgerollt. Korrekt.
- [x] **Datenintegrität aktive Ausleihen (neu, P2)** — **Gefunden+behoben:** kein Schutz gegen zwei aktive Ausleihen pro Exemplar/Gerät (FOR UPDATE kann nicht-existente Zeile nicht sperren). Migration 033: Dedup bestehender Duplikate (jüngste behalten) + partielle Unique-Indizes; Unique-Verletzung wird zu 409-Konflikt gemappt. Gegen echte DB verifiziert.
- [x] **Repository-Layer (24 Dateien)** — Queries voll parametrisiert (keine Injection), `coalesce`-NULL-Handling konsistent. **Gefunden+behoben:** 5 Dateien (audit_books, inventory_repo, reservation_repo, user, vormerkung) iterierten `rows.Next()` ohne anschließende `rows.Err()`-Prüfung → ein Verbindungsabbruch mitten in der Iteration hätte eine TEILMENGE als Erfolg geliefert (z. B. Nutzerliste/Reservierungen unvollständig; audit_books hätte einen Titel trotz aktiver Ausleihen löschen können). `rows.Err()`-Checks ergänzt.

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
