# Master-Fahrplan: Radar-Analyse & Konsolidierung

> Stand: **2026-07-08** · Ursprung: Radar-Analyse vom 07.07. (Routen-Abgleich, Komponenten-Nutzung,
> Test-Inventar, Middleware-Audit). Lebendes Dokument — Phase 1 und Phase 2 sind abgeschlossen,
> aktueller Arbeitsvorrat steht unter „Nächste Schritte".
> Radar-Referenz: [`dokumentation/api_inventar.md`](api_inventar.md) (neu erzeugen mit `./scripts/api_inventar.sh`).

---

# ✅ ERLEDIGT (Kurzprotokoll mit Befunden)

## ⚡ Vorab-Bug (2026-07-07, `3efb88f`)
„Sperre aufheben" im Omnibox-Block-Alert rief das nicht existierende `POST /api/schueler/{id}/update`
→ umgestellt auf `PATCH /api/admin/students/{id}/lock` (wie `StudentLockModal`). War der einzige
Geister-Aufruf im gesamten Abgleich; E2E-Flow „sperren/entsperren" steht noch aus (s. u.).

## 🧹 Phase 1 — Dead Code & Cleanup (2026-07-07, `b33e05c` + `28b3add` + `098194d`)
- **11 tote Go-Handler + Routen gelöscht** inkl. verwaister Helfer (`service.ReceiveItem`,
  beide `UndoReturn`-Repo-Methoden, `RecentTransaction`): `bestellungen/receive`,
  `transactions/recent`, 2 Mail-501-Stubs, Import-Dreifach-Cluster (`import/students`,
  `students/import`, `schueler/import-lusd`), Signatur-PUT/DELETE, `ausleihen/{id}/rueckgabe`.
- **Undo-Return-Feature bewusst gestrichen** (Handler + `UndoToast` + Store — war nie verdrahtet).
- **16 tote Svelte-Dateien gelöscht** (GlobalScanner/KioskMode-Cluster, LusdPreviewModal,
  ClassPrintStation, StudentEditModal, OfflineQueueBanner, SvelteKit-Reste unter `inventur/routes/`,
  3 Inventur-Komponenten). `triggerStudentScan` blieb — hat aktive Schreiber (BookBorrowers*).
- **`scripts/api_inventar.sh`** erzeugt das Routen/Aufrufer-Inventar als Radar-Referenz.
- Bewusst **nicht** gelöscht (Entscheidungsfälle → Phase 3): `PromoteStudentsHandler`,
  Klassensatz-„erledigen"-Handler, `api/lusd.go` (preview/import — getesteter Flow, verwaist).

## 🧪 Phase 2 — Die Festung (T1–T7)

**T1 — Wareneingang/Bestellwesen (07.07., `5160156`):** Go-Tests `BulkReceiveOrder`
(`received_items` + `etikett_gedruckt`-Vertrag, 404-Wortlaut); 13 orderStore-Vitest-Tests
(Warenkorb-Dedup `titel_id`/ISBN, Summen, Submit-Gate, **Out-of-Order-Such-Race**).

**T2 — Berichts-Datumsgrenzen (07.07., `5160156`):** Datums-Helfer nach `lib/utils/dates.js`
extrahiert; Regressionstests gegen den Zeitzonen-Bug (Monatsletzter, Schaltjahre).

**T3 — Auth-Lebenszyklus (07.07., `fc36fb1`):** Session-Refresh-Loop im `authStore` verdrahtet
(30-min-Tick; Server erneuert ab <50% Restlaufzeit; 401→Logout, Netzfehler≠Logout).
5 Go-Tests `RefreshTokenHandler`, 3 Vitest-Tests. *Rest offen: Login-Handler-Tests (IMAP mocken).*

**T4 — Mahnwesen (07.07., `d659759`):** Test fand echten Bug — **Slice-Reallokation verschluckte
Medien gleichnamiger Schüler** in allen drei Mahnlisten-Queries (Pointer in Slice-Elemente).
Fix: index-basierter `klassenGrouper`; Scan-Fehler werden nicht mehr verschluckt.

**T5 — E2E-Gerüst Playwright (08.07., `346e1ce`):** `npm run test:e2e` (frontend/) gegen den
lokalen Docker-Stack (`docker compose -f docker-compose.local.yml up -d --build`; Backend :8084,
Postgres :5434, Mock-IMAP akzeptiert jedes Passwort). 3 Smoke-Flows, mehrfach stabil grün (~1,6 s):
UI-Login/Logout · Lieferant anlegen + Berichte-Datumsvalidierung · Schüler per API seeden →
Omnibox-Scan → Konto. `uiLogin`-Fixture mit Fill-Guards (Svelte-Mount-Race); Vitest excludet `e2e/`.

**T6 — Inventur-Rechte (07.07., `9ddd050`):** Fehlalarm der Benennung — RBAC war längst injiziert
(`RequirePermission("view_books"/"edit_books")`); Felder umbenannt zu `RequireViewBooks`/`RequireEditBooks`.
(`GET /uploads/` bleibt unauthentifiziert — ausschließlich Buchcover-WebPs.)

**T7 — Betriebspflichten:** ✅ **Migration 035 real getestet** (08.07., lokale DB): Wiederanmeldung
einer soft-gelöschten `lusd_id` legt frischen aktiven Datensatz an; zweiter *aktiver* scheitert
korrekt an `uniq_schueler_lusd_id_active`. ⏳ Nur in Zielumgebung: Restore-Probe gegen Wegwerf-DB;
Prod-Secrets (`ENFORCE_PROD_SECRETS`, `BACKUP_ENCRYPTION_KEY` — ohne den läuft **kein** Backup).

---

# 🔧 NÄCHSTE SCHRITTE (aktueller Arbeitsvorrat, in dieser Reihenfolge)

1. ~~**Session-Restore beim SPA-Boot**~~ ✅ **erledigt 08.07.** (`daf19f2`): `GET /api/auth/me` +
   `restoreSession()`-Boot-Check mit `sessionChecked`-Gate; Logout invalidiert jetzt auch serverseitig.
   E2E beweist beide Richtungen (Reload bleibt eingeloggt / bleibt ausgeloggt). **Der E2E-Bau fand
   dabei zwei weitere echte Bugs, beide gefixt:** (a) `fix(auth) d2ecf4c` — JWTs ohne `jti` waren bei
   zwei Logins in derselben Sekunde byte-identisch, ein Logout widerrief beide Sessions;
   (b) `fix(api) 2c54ce6` — die erste Mutation nach dem Login lief ohne CSRF-Token in einen 403
   (Cookie wurde nie initial beschafft, jetzt Bootstrap über `GET /api/csrf-token`). Außerdem:
   Heartbeat-Overlay erscheint nicht mehr bei transientem SSE-`onerror` (Druckdialog!), sondern
   erst nach dem dokumentierten 25s-Timeout.
2. **Bot-PR-Triage** (10 offene PRs, Stand 08.07. — Review/Merge ist Peters Entscheidung):
   zuerst **#200 (HIGH, Path Traversal)**, dann #194 (`os.OpenRoot`, backup_email) und
   #197 (PGPASSWORD-Leak — schleppt CI-/go.mod-Änderungen mit, genau prüfen).
   **Achtung:** Bolt-Order-PRs #195/#196 sind gegen den Stand *vor* dem orders→bestellungen-Refactor
   gebaut → vermutlich konfliktbehaftet/obsolet. Palette-PRs (a11y) risikoarm.
3. ~~**Vierter E2E-Flow**~~ ✅ **erledigt 08.07.** (`093968f`): sperren → Block-Alert →
   „Sperre dauerhaft aufheben" → Ausleihe läuft durch (deckt den Vorab-Bugfix E2E ab).
   **Beifang — echter Prod-Bug gefixt** (`e94a6fb`): Eine NULL-wert-Zeile in
   `system_einstellungen` machte über querySettings' string-Scan **jeden Checkout zum 500**
   (pgx bricht ab, rows.Err() schlägt durch). Fix: `coalesce(wert,'')` + Regressionstest.
4. ~~**Login-Handler-Tests**~~ ✅ **erledigt 08.07.** (`f0058b3`): 5 Tests — Validierung,
   401 unbekannt/403 deaktiviert, Cookie+LoginShape, Brute-Force-Limiter (429 ohne DB-Zugriff).
5. ~~**FE ruft 501-Stub**~~ ✅ **erledigt 08.07.** (`49c7abc`): Eltern-Mail-Button samt
   Store-Pfad, Stub-Handler und Route entfernt — Versand bleibt aus Datenschutzgründen deaktiviert.

---

# 🚀 Phase 3: Produktentscheidungen & Ausbau (nach den nächsten Schritten)

1. **LUSD-Import-Konsolidierung:** Aktiv ist nur `/api/import/lusd` (LusdImportModal);
   `/api/lusd/preview`+`import` (api/lusd.go) sind der *getestete* Preview→Commit-Flow, aber seit
   Löschung des toten Modals verwaist. Entscheiden: besseren Flow anbinden oder endgültig streichen.
2. **Klassensatz-Reservierung „erledigen"** — UI-Lücke schließen (Handler existiert und ist korrekt).
3. **Schuljahres-Versetzung** (`students/promote`): UI bauen oder Handler streichen — **Deadline
   Schuljahreswechsel!**
4. **API-Versionierung `/api/v1` + Rest-Sprachvereinheitlichung** — ein Paket, jetzt mit E2E-Netz.
5. **Mandantenfähigkeit (RLS)** — Tenant-Claim in Auth-Middleware, `tenant_id`-Migrationen mit dem
   etablierten Dry-Run-Prozess.

---

# 🛑 Das Parkdeck (unverändert — fassen wir NICHT an)

| Thema | Warum geparkt |
|---|---|
| **Mandantenfähigkeit / RLS** | Erst als Phase-3-Punkt 5, nicht früher |
| **Rest-Vereinheitlichung API-Sprache** (`/api/books` vs. `/api/buecher` …) | Nur als Paket mit `/api/v1` (Phase 3.4) |
| **Integer-Cent-Refactor** (Go `float64`, DB `NUMERIC(10,2)`) | Bewusste, dokumentierte Nicht-Entscheidung |
| **Bundle-Splitting** (720-kB-Chunk) | Performance-Feinschliff, kein Stabilitätsthema |
| **TypeScript-Migration** | JSDoc-Typedefs reichen aktuell |
| **Verschmelzung `inventur/` ins Haupt-API** | Rechte sind angeglichen (T6); Struktur bleibt |
| **Edge-to-Edge-Feinschliff restlicher Views** | UI-Refactoring abgeschlossen; kein Re-Opening ohne Anlass |

---

## Radar-Zahlen (Stand 08.07., nach Phase 1+2)

| Metrik | Radar 07.07. | Jetzt |
|---|---|---|
| Geister-Aufrufe (FE ohne Backend-Route) | 1 | **0** |
| Tote Go-Handler | 11 + 3 Fälle | **0** (3 bewusste Entscheidungsfälle dokumentiert) |
| Tote Svelte-Dateien | 13–16 | **0** |
| Svelte-4-Konstrukte | 0 | 0 (Runes-Migration vollständig) |
| Go-Testdateien / FE-Testdateien / E2E-Flows | 25 / 1 / 0 | **30 / 4 / 3** |
| Bekannte offene UX-Defekte | — | 1 (501-Stub-Aufruf Mahn-Mail) |
