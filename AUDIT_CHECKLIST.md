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

- [x] **SSE-Broker** (`sse/`) — sauber: zentraler Event-Loop, `RLock`/`Lock` verhindern send-on-closed, **non-blocking** Broadcast (langsamer Client blockiert andere nicht), gepufferter Channel, Heartbeat + Context-Abbruch. Nur benigne Schwäche: bei Server-Shutdown können `register`/`unregister`-Sends blockieren, nachdem `Start` via ctx endet — Prozess terminiert aber ohnehin. Kein Runtime-Leak.
- [x] **Background-Jobs** (`backup.go`, `cron.go`) — Backup: pg_dump→gzip→AES-GCM (Zufalls-Nonce)→0600, Fehler je Stufe behandelt, Rotation. GDPR-Jobs konservativ & auditiert (Löschung nur ohne offene Ausleihen/Schäden, 30-Tage-Karenz; Fehler löscht eher zu wenige). Notiz (nicht-kritisch): Backup hält Dump komplett im RAM (2:30 Uhr ok); Cron ohne globales SkipIfStillRunning, aber kritischer Cover-Sync hat atomaren Re-Entrancy-Guard.
- [x] **Cover-Sync unter Last** — bereits in früherer Session überarbeitet: Worker-Pool (8), Re-Entrancy-Guard, FAILED-Retry, lokale WebP-Speicherung. Provider-Rate-Limits werden über FAILED-Status + Retry abgefedert.
- [x] **Migrations-Hygiene** — doppelte Präfixe (003/008/021/022 je 2×) sortieren deterministisch und sind inhaltlich unabhängig (keine Reihenfolge-Abhängigkeit). Der eigentliche Bug (schema_migrations-Seed-Mismatch) ist in `1c79b2c` behoben. Verbleibt: Stil-Smell (künftige Migrationen mit eindeutigem Sequenz-Präfix).
- [x] **Rate-Limit/Brute-Force-Tuning** — **Gefunden+behoben:** der Login-Brute-Force-Limiter schlüsselte rein auf die IP (5 Fehlversuche/15 min). Bei Schul-NAT (alle Geräte hinter einer IP) hätten 5 Fehlversuche eines Nutzers die ganze Schule ausgesperrt. Schlüssel auf `email|ip` umgestellt (Check nach E-Mail-Extraktion) → sperrt nur das betroffene Konto auf dieser IP, schützt weiter gegen Account-Brute-Force. Globaler Request-Limiter (50/s/IP) unverändert.

---

## 🔵 Priorität 4 — Frontend (153 Komponenten)

- [x] **Routen-/View-Guards** — Views werden client-seitig per `activeTab` umgeschaltet, aber JEDE Datenabfrage ist backend-seitig permission-gated (`RequirePermission`). Eine erzwungene View zeigt ohne Recht nur 403 — kein Datenleck. Backend ist die Autorität (korrekt).
- [x] **XSS** — KEIN `{@html}`, kein `innerHTML`/`eval` im gesamten Frontend; Svelte escaped alle Interpolationen automatisch. SVG-Icons sind hartkodierte Konstanten. Kein XSS-Sink.
- [x] **Offline-Queue (PWA)** — robust: Items nur bei Erfolg/permanentem 4xx (außer 429) entfernt, bei 5xx/Netzfehler erhalten + späterer Retry; Idempotenz-Key (`item.id`) wird mitgesendet → Backend-Dedup beim Replay (Doppelausführung verhindert, datenseitig zusätzlich durch Migration 033 abgesichert); `beforeunload`-Warnung. Minor-Nit: Failsafe entfernt Item bei fehlendem Batch-Index — Backend liefert aber stets alle Indizes.
- [x] **API-Fehlerbehandlung** — zentral via `apiFetch`/`handleSmartResponse` (Toasts bei Fehler); frühere stille Fehler (Adresse/ISBN) bereits behoben.
- [x] **Svelte-5-Stores** — Runes laufen single-threaded (JS-Eventloop) → keine klassischen Data-Races; SSE-Reconnect mit Guards (`isLoggedIn`, Timeout). Unauffällig.

---

## ⚪ Priorität 5 — Qualität / Infrastruktur

- [~] **Testabdeckung** — weiterhin Lücken (db/repository/auth ohne Tests); diese Session ergänzt: `pkg/csvutil`, `pkg/imageutil` (Bomb-Schutz). Kritische Pfade (Loan, RBAC, Import) bräuchten noch Tests — offen als Daueraufgabe.
- [x] **golangci-lint vollständig** — **0 issues** über das ganze Repo (vorher 11). Behoben: 2 eigene ST1005 (Fehlertexte klein), 3× `os.Setenv` (Test), `fmt.Sscanf`/`file.Close`/`io.Copy`/`resp.Body.Close` (Tools), 1 ineffassign; stray `tmp/export_csv.go` entfernt + `tmp/` ge-gitignored.
- [x] **ODBC-Abhängigkeit** — nur vom Einmal-Tool `cmd/littera_migration` genutzt (Littera-ODBC-Quelle), nicht vom Server. Hinter Build-Tag `//go:build odbc` versteckt → `go build ./...`/CI/Lint laufen jetzt ohne unixODBC; Tool bei Bedarf `-tags odbc`.
- [~] **Dockerfile/Compose** — Build kopiert migrations/+schema.sql, CGO für WebP, Postgres-Healthcheck. Offene OPERATIVE Härtung (bewusst nicht auto-geändert, deployment-abhängig): Container läuft als root (kein `USER`); compose-Default-Fallback-Secrets müssen in Prod via echte Env-Vars überschrieben werden; kein Backend-Healthcheck.

---

### Legende
`[ ]` offen · `[x]` tief gescannt (+ ggf. behoben, Commit in Klammern)
