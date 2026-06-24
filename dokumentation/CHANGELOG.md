# Changelog

---

## 2026-06-24 — Tiefen-Audit, UI-Redesign & Docker-Härtung

Diese Session umfasste einen vollständigen statischen Code-Audit aller 46 Dateien, ein globales UI-Redesign (153 Svelte-Komponenten) sowie eine Produktions-Härtung der Deployment-Infrastruktur.

---

### Bugfixes

#### Omnibox / Suche (HTTP 500)
- **`LEHRER`-Enum-Casing**: `buecher_suche`-Query nutzte `rolle = 'LEHRER'` (UPPERCASE), die DB speichert aber lowercase. Fix: `LOWER(rolle::text) = 'lehrer'`. Commit: `7271bc2`
- **Schema-Drift**: Fehlende Spalten (`buecher_titel.untertitel`, `ziel_jahrgang`) in älteren Produktionsdatenbanken verursachten 500er. Neue Migration `032_reconcile_titel_columns.sql` gleicht alle Spalten idempotent an.
- **Schema-Migrations-Seed**: `schema.sql` hatte 13 fehlende Einträge + 1 Phantom-Eintrag (`007_performance_indexes.sql`). Folge: Migrationen liefen beim Deployment erneut, Migration 030 crashte mit "column already exists". Behoben in `1c79b2c`.
- **Migration 030 Idempotenz**: Schlägt nicht mehr fehl wenn `ziel_jahrgang` bereits existiert (beide Fälle behandelt).

#### Buchklick in der Omnibox
- Klick auf ein Buch in den Omnibox-Ergebnissen öffnete den allgemeinen Medienkatalog statt der Buchdetailseite. Fix: `handleSelectBook` in `Router.svelte` nutzt jetzt `book_detail`-Tab.

#### Print-Bug (Ausweis vs. Ausleihquittung)
- "Ausweis drucken" zeigte fälschlicherweise auch die offene Ausleihliste. Fix: CSS-Klasse `.print-receipt-section` in `StudentPrintReceipt.svelte` + `body[data-print-mode] .print-receipt-section { display: none !important }` in `app.css`.

#### Rollen-Berechtigungen (Lehrer)
- Login lieferte keine echten Permissions aus `role_permissions` — Lehrer sahen nichts oder alles. Fix in `auth/handlers.go`: Berechtigungen werden direkt nach Login aus der DB geladen; Admin erhält `["*"]`.

---

### Sicherheit

#### RBAC Permission-Cache
- Transiente DB-Fehler bei `RequirePermission` wurden als 403 gecacht. Ein kurzfristiger DB-Ausfall sperrte legitime Nutzer für 60 Sekunden aus. Fix: nur `pgx.ErrNoRows` → cachen + 403; DB-Fehler → 500, kein Caching. Commit: `8da7ee9`

#### Brute-Force-Limiter
- Schlüssel war reine IP. An Schul-NAT (alle Geräte hinter einer IP) hätten 5 Fehlversuche die gesamte Schule ausgesperrt. Fix: Schlüssel auf `lower(email)|ip` geändert.

#### SMTP STARTTLS (MITM)
- `InsecureSkipVerify: true` war gesetzt — alle SMTP-Verbindungen waren gegen MITM angreifbar (Credentials, Mailinhalt/Personendaten). Fix: `ServerName` + `MinVersion: TLS 1.2`. Opt-out via `SMTP_ALLOW_INSECURE_TLS=true`. Attachment-Dateiname gegen CRLF-Injection bereinigt.

#### CSRF-Lücke
- `sync-covers` und `import-bestand` lagen unter `/api/admin/*` und waren dadurch von der globalen CSRF-Prüfung ausgenommen. Fix: Ausnahme entfernt.

#### CSV-Formel-Injection (CWE-1236)
- CSV-Exporte schrieben Buchtitel/Autoren verbatim. Böswillige Strings (z. B. `=HYPERLINK(…)`) könnten in Excel als Formel ausgeführt werden. Fix: neues `pkg/csvutil.SanitizeRow` (Apostroph-Präfix).

#### Decompression-Bomb-Schutz
- `image.Decode` allokiert die vollständige Pixelmatrix vor der Dimensionsprüfung. Ein präpariertes 50.000×50.000px-JPEG würde den RAM erschöpfen. Fix: neues `pkg/imageutil.GuardImageDimensions` (liest nur Header via `image.DecodeConfig`, Limit: 50 MP). Beide Upload-Pfade (Foto + Cover) nutzen es.

#### Docker-Härtung / Secret Guard
- `docker-compose.yml` hatte Fallback-Defaults für Secrets — Deployments mit vergessenen Secrets liefen einfach mit den Repo-Defaults weiter (vollständige JWT-Fälschung möglich).
- Fix 1: `docker-compose.yml` nutzt `${VAR:?}` — Docker bricht mit sprechender Fehlermeldung ab wenn eine Variable fehlt.
- Fix 2: `main.go/loadConfig` verweigert den Server-Start außerhalb von `APP_ENV=local` wenn bekannte Default-Secrets erkannt werden.
- Fix 3: `docker-compose.local.yml` erhält `APP_ENV=local` + gültiges 32+-Zeichen-Dev-JWT.

---

### Datenintegrität

#### Unique Active Loan (Migration 033)
- Kein DB-Schutz gegen zwei gleichzeitige aktive Ausleihen auf demselben Exemplar (Race in TOCTOU-Fenster). Fix: Dedup bestehender Duplikate + Partial Unique Indexes (`WHERE rueckgabe_am IS NULL`). Unique-Verletzung → HTTP 409.

#### Repository: `rows.Err()` nach Iteration
- 5 Dateien (`audit_books.go`, `inventory_repo.go`, `reservation_repo.go`, `user.go`, `vormerkung.go`) iterierten Rows ohne abschließende `rows.Err()`-Prüfung. Ein Verbindungsabbruch wurde als Erfolg behandelt (unvollständige Ergebnisse). Behoben.

#### Excel-Import Goroutinen
- Semaphore wurde innerhalb der Goroutine akquiriert statt davor → unbegrenzte Goroutine-Erzeugung unter Last. Behoben. Commit: `8da7ee9`

#### DSGVO vs. Mahnwesen
- Migration 003 blockierte Adressspalten per `RAISE EXCEPTION`. Da diese für das Mahnwesen (Briefversand) essenziell sind, wurde der Wächter entfernt.

---

### Code-Qualität

#### golangci-lint: 0 Issues
- Vorher 11 Issues. Behoben: 2× ST1005 (deutsche Fehlertexte kleingeschrieben), 3× `os.Setenv` in Tests, mehrere `errcheck`-Funde, 1 `ineffassign`. Stray `tmp/export_csv.go` entfernt, `tmp/` in `.gitignore`.

#### ODBC-Abhängigkeit isoliert
- `cmd/littera_migration` benötigte `unixODBC` → blockierte `go build ./...` und Lint ohne ODBC. Fix: `//go:build odbc` Tag. Standard-Build läuft ohne ODBC.

---

### UI/UX-Redesign (Flat & Edge-to-Edge)

Vollständiger Sweep über 153 Svelte-Komponenten in 7 Batches. Ziel: Karten-Anti-Pattern auf Layout-Ebene beseitigen → flaches Design, edge-to-edge, Trennung via `border-b border-gray-200`.

| Batch | Bereich | Commits |
|---|---|---|
| 1 | MailTemplates, OverdueWidget | `785ecb0` |
| 2 | Bestellungen | `785ecb0` |
| 3 | Schüler-Listen, Audit-Logs | `555fa11` |
| 4 | Buch-Akte-Tabs | `0853476` |
| 5 | Dashboards, Portale, Sonstiges | `7e0c37b` |
| 6 | Inventur-Modul | `0eed201` |
| 7 | Einstellungs-Tabs, Typografie | (aktuell staged) |

Batch 7 (Einstellungen):
- `SystemSettings.svelte` — "Team & Rechte" + "System"-Tabs nicht mehr auf `max-w-3xl` eingeengt
- `SystemSettingsAllgemein.svelte` — `max-w-3xl` → `max-w-5xl`, größere Grid-Abstände
- `SettingField.svelte` — Labels `text-xs uppercase` → `text-sm font-medium text-gray-600`; Inputs → `text-lg`
- `PermissionManager.svelte` — weißer Container entfernt, 263 → 150 Zeilen (DRY: `PermissionsEditor.svelte` + `permissionMetadata.js` ausgelagert)

---

### Neue Dateien

| Datei | Zweck |
|---|---|
| `migrations/032_reconcile_titel_columns.sql` | Idempotente Schema-Angleichung für ältere DBs |
| `migrations/033_unique_active_loan.sql` | Unique-Partial-Indizes für aktive Ausleihen |
| `pkg/csvutil/csvutil.go` | CSV-Formel-Injection-Schutz |
| `pkg/imageutil/webp.go` | Decompression-Bomb-Guard |
| `frontend/src/lib/PermissionsEditor.svelte` | DRY-Teilkomponente für Role-Toggles |
| `frontend/src/lib/permissionMetadata.js` | Extrahierte Permissions-Metadaten |

---

## 2026-06-20 — Initiale Dokumentation & Deployment-Setup

- Erste Version der Dokumentation (`dokumentation/`) erstellt
- Deployment-Guide für Hetzner-Server (`DEPLOYMENT.md`)
- Sicherheitskonzept-Dokumentation (`SECURITY.md`)
- Architektur-Übersicht (`ARCHITECTURE.md`)
- Wareneingang-Implementierungsplan (`implementation_plan_wareneingang.md`)
