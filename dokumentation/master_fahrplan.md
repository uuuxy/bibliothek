# Master-Fahrplan: Radar-Analyse & Konsolidierung

> Stand: 2026-07-07 · Basis: vollständiger Abgleich aller 113 registrierten Go-Routen
> gegen sämtliche `/api/`-Aufrufer im Frontend, Komponenten-Nutzungsanalyse,
> Test-Inventar (25 Go-Testdateien / 1 FE-Testdatei / 0 E2E) und Middleware-Audit.

---

## ⚡ VORAB: Ein echter Laufzeit-Bug (heute fixen, vor allem anderen)

**`OmniboxBlockAlert.svelte:31` ruft `POST /api/schueler/{id}/update` auf — diese Route
existiert im Backend nicht.** Der Button „Sperre aufheben" im Omnibox-Block-Alert läuft
seit der Schüler-API-Konsolidierung ins 404-Leere. Fix: auf das bestehende
`PATCH /api/schueler/{id}` umstellen (Payload `is_manually_blocked`/`block_reason` bleibt).
Das ist der einzige im Radar gefundene *aktive* Defekt — alles andere ist tot, nicht kaputt.

---

# 🛑 STOP: Das Parkdeck (fassen wir vorerst NICHT an)

| Thema | Warum geparkt |
|---|---|
| **Mandantenfähigkeit / RLS** | Null Stabilitätsgewinn heute; erst sinnvoll auf getestetem, totem-Code-freiem Fundament (→ Phase 3) |
| **Rest-Vereinheitlichung der API-Sprache** (`/api/books` vs. `/api/buecher`, `/api/students/*` vs. `/api/schueler/*`) | Bestellungen sind migriert; der Rest ist reine Kosmetik mit Breaking-Change-Risiko. Kommt als Paket mit `/api/v1`-Versionierung |
| **Integer-Cent-Refactor** (Geld ist Go-seitig `float64`, DB exakt `NUMERIC(10,2)`) | Bewusste, dokumentierte Nicht-Entscheidung — bleibt so |
| **Bundle-Splitting** (720-kB-Chunk-Warnung im Vite-Build) | Performance-Feinschliff, kein Stabilitätsthema |
| **TypeScript-Migration / `any[]`-Typisierung** | Nice-to-have; JSDoc-Typedefs im orderStore reichen aktuell |
| **Verschmelzung des `inventur/`-Moduls ins Haupt-API** | Zweites Routing-Universum ist hässlich, aber funktional. Nur der Rechte-Unterschied wird behandelt (→ Phase 2, Punkt T6) — die Struktur bleibt |
| **Edge-to-Edge-Feinschliff restlicher Views** | UI-Refactoring ist offiziell abgeschlossen; kein Re-Opening ohne Anlass |

---

# 🧹 Phase 1: Dead Code & Cleanup (die nächsten 1–2 Tage)

Regel: Vor jedem Löschen ein `grep` über das Gesamt-Repo (inkl. `loadtest.js`, Doku zählt nicht) — dann löschen, bauen, Tests grün, committen. Kleine Commits pro Cluster.

## 1a. Go-Handler ohne Frontend-Aufrufer (löschen)

| Route | Handler | Anmerkung |
|---|---|---|
| `POST /api/bestellungen/receive` | `ReceiveItemHandler` | Vom Bulk-Receive-Flow ersetzt — der bekannte Altlast-Fund |
| `GET /api/transactions/recent` | `GetRecentTransactionsHandler` | Kein Aufrufer im gesamten Frontend |
| `POST /api/mail/send-notification/{schuelerID}` | `PostSendNotificationHandler` | Nur die `send-overdue-notification`-Variante wird genutzt |
| `POST /api/mail/send-bulk-overdue` | `PostSendBulkOverdueHandler` | Kein Aufrufer |
| `POST /api/import/students` | `ImportStudentsHandler` | Import-Altlasten-Cluster: aktiv sind nur |
| `POST /api/students/import` | `ImportStudentsLUSDHandler` | `/api/import/lusd`, `/api/lusd/preview`, |
| `POST /api/schueler/import-lusd` | `PostSchuelerImportLusdHandler` | `/api/lusd/import` und `/api/import/littera` |
| `PUT /api/signatures/{id}` + `DELETE /api/signatures/{id}` | Signatur-Mutationen | FE nutzt nur `GET/POST /api/signatures` |

## 1b. Go-Handler ohne Aufrufer — aber ERST ENTSCHEIDEN, dann löschen

| Route | Frage an uns selbst |
|---|---|
| `DELETE /api/ausleihen/{id}/rueckgabe` (`UndoReturnHandler`) | Gehörte zum nie fertig verdrahteten Undo-Feature (siehe toter `UndoToast` unten). Feature streichen → beides löschen. Feature wollen → Ticket für Phase 3 |
| `POST /api/students/promote` (`PromoteStudentsHandler`) | Schuljahres-Versetzung: fachlich wichtig, aber ohne UI unerreichbar. Vor dem Schuljahreswechsel entscheiden: UI bauen (Phase 3) oder löschen |
| `PUT /api/reservierungen/klassensatz/{id}/erledigen` | Die Reservierungs-*Anzeige* wird genutzt, aber keine Reservierung kann je erledigt werden — hier fehlt UI, der Handler ist korrekt. **Nicht löschen**, UI-Lücke in Phase 3 schließen |

## 1c. Tote Svelte-Dateien (löschen)

- `lib/GlobalScanner.svelte` + `lib/KioskMode.svelte` — Cluster: nirgends importiert; der Kiosk-Tab wird anders gerendert. Danach prüfen, ob `appState.triggerStudentScan` (Inventur-Store + App-Effect) seinen letzten Schreiber verloren hat → dann auch das Feld und den `$effect` in `App.svelte` entfernen
- `lib/UndoToast.svelte` + `lib/undoToastStore.svelte.js` — nutzen nur einander (siehe 1b, UndoReturn)
- `lib/LusdPreviewModal.svelte` — vom neuen LUSD-Flow ersetzt
- `lib/ClassPrintStation.svelte` — Klassensatz-Druck lebt jetzt im DruckCenter
- `lib/StudentEditModal.svelte`, `lib/OfflineQueueBanner.svelte` — keine Importe
- `inventur/lib/components/StartseitenHeader.svelte`, `admin/KlassenNamenEditor.svelte`, `admin/BuchAuswahlListe.svelte` — keine Importe
- `inventur/routes/+layout.svelte`, `+page.js`, `admin/+page.js`, `settings/+page.svelte`, `scanner/+page.svelte` — SvelteKit-Konventionsreste in einer Vite-SPA. **Achtung:** `+page.svelte` und `admin/+page.svelte` sind über `MediaCatalog.svelte` aktiv eingebunden — die bleiben! Wegen der verwechselbaren `+page`-Namen jede Datei einzeln per Import-Grep verifizieren

## 1d. Abschluss Phase 1

- `go build ./... && go vet ./... && go test ./...`, `npm run build && npm test`
- Routen-Inventar einmal neu ziehen (Skript aus dieser Analyse) und als `dokumentation/api_inventar.md` einchecken — das ist ab jetzt unsere Radar-Referenz

---

# 🧪 Phase 2: Die Festung bauen (Testing der blinden Flecken)

Befund: Solide Go-Tests für Ausleih-Regeln, RBAC-Middleware, LUSD-Parser, Backup-Roundtrip.
Aber: **das komplette Bestellwesen, der Etikettendruck und der Auth-Lebenszyklus sind ungetestet**, im Frontend existiert genau eine Testdatei (`authStore.test.js`), E2E-Tests gibt es nicht.

Reihenfolge nach Risiko × Änderungsfrequenz:

**T1 — Wareneingang → Etikettendruck (der frisch umgebaute Pfad, höchstes Risiko):**
- Go: Handler-Test `BulkReceiveOrderHandler` — insbesondere das neue `received_items`-Payload mit `etikett_gedruckt` (der SQL-JOIN auf `buecher_titel` ist jung), Leerlauf-Fall (bereits freigegeben → 404), Teilmengen
- FE: `orderStore`-Unit-Tests — jetzt isoliert testbar: Warenkorb-Dedup (`titel_id` vs. `id` vs. ISBN — der frühere Duplikat-Bug!), `total`/`totalQty`, Submit-Payload mit/ohne `attachBarcodes`, Such-Race (Sequenznummer, mit Fake-Timern)
- FE: `printQueue`-Übergabe Wareneingang → DruckCenter (Komponententest oder E2E, siehe T5)

**T2 — Bestellberichte-Datumsgrenzen:** Regressionstest für den Zeitzonen-Fix (Monatsletzter!) plus Go-Test, dass `von`/`bis` im Berichts-SQL inklusiv sind. Falsche Abrechnungssummen sind Vertrauenskiller Nr. 1.

**T3 — Auth-Lebenszyklus:** Login → Refresh → Logout → Heartbeat-Timeout (Handler-Ebene; die Middleware ist getestet, der Flow nicht).

**T4 — Mahnwesen:** `GET /api/mahnwesen/pdf` und `POST /api/mahnwesen/senden` — hier fließt Geld (`float64`-Summen!) und es geht Post an Eltern raus.

**T5 — E2E-Gerüst (Playwright), genau 3 Smoke-Flows, nicht mehr:**
1. Omnibox: Schüler scannen → Buch ausleihen → zurückgeben
2. Bestellung anlegen → Wareneingang einbuchen → Druckempfehlung erscheint → Etiketten laden
3. Schüler anlegen → sperren → Sperre aufheben (deckt den Vorab-Bugfix ab)

**T6 — Rechte-Angleichung Inventur-Modul (Tech-Debt mit Sicherheitsbezug):**
Das `inventur/`-Modul fährt ein eigenes Rechtemodell: `RequireAuth` (= *jeder* eingeloggte Benutzer, auch Helfer) für `GET /api/books` & Co., `RequireAdminAuth` (Rolle statt Permission) für Schreibzugriffe. Das Haupt-API nutzt überall `RequirePermission`-RBAC. Angleichen: `view_books` für Lesen, `edit_books` für Schreiben — und einen Permission-Middleware-Test dafür. (`GET /uploads/` ist unauthentifiziert, enthält aber ausschließlich Buchcover-WebPs — dokumentieren, nicht ändern.)

**T7 — Betriebs-Pflichten aus dem Go-Live-Plan (kein Code, aber Teil der Festung):**
Migration 035 (`lusd_id`-Unique-Index) real dry-runnen + LUSD-Re-Import testen; einmalige Restore-Probe gegen Wegwerf-DB; Prod-Secrets-Checkliste (`ENFORCE_PROD_SECRETS`, `BACKUP_ENCRYPTION_KEY` — ohne den läuft **kein** Backup).

---

# 🚀 Phase 3: Zukünftige Features (erst wenn Phase 1 & 2 grün sind)

> Neuer Befund aus dem Inventar (2026-07-07): `POST /api/auth/refresh` hat keinen
> Frontend-Aufrufer — die SPA erneuert Tokens nie, Sessions laufen einfach ab.
> Entweder in `apiFetch` bei 401 einen Refresh-Versuch einbauen oder das bewusst
> so dokumentieren (Kiosk-Betrieb mit Heartbeat). Gehört zu T3 (Auth-Lebenszyklus).

In dieser Reihenfolge:

1. **LUSD-Import-Konsolidierung:** Nach Phase 1 ist nur noch `/api/import/lusd` (LusdImportModal) aktiv. `/api/lusd/preview` + `/api/lusd/import` (api/lusd.go) sind durch die Löschung des toten `LusdPreviewModal` verwaist — aber das ist der *getestete* Preview→Commit-Flow (`lusd_parser_test.go`). Entscheidung: den besseren Flow wieder anbinden und den einfachen Import ablösen, oder umgekehrt. Bis dahin bleibt api/lusd.go bewusst stehen
2. **Klassensatz-Reservierung „erledigen"** — die in 1b identifizierte UI-Lücke schließen
3. **Schuljahres-Versetzung** (`students/promote`): UI bauen oder Handler endgültig streichen — Deadline ist der Schuljahreswechsel
4. **API-Versionierung `/api/v1` + Rest-Sprachvereinheitlichung** — ein einziges großes, gut getestetes Migrations-Paket (wie die orders→bestellungen-Migration, jetzt mit E2E-Netz)
5. **Mandantenfähigkeit (Row-Level Security)** — erst jetzt: RLS-Policies auf einem Schema ohne tote Tabellen-Zugriffspfade, Tenant-Claim in der Auth-Middleware, `tenant_id`-Spalten-Migrationen mit dem in T7 etablierten Migrations-Dry-Run-Prozess

---

## Radar-Zahlen (Referenz)

| Metrik | Wert |
|---|---|
| Registrierte Routen (alle Registrierungsorte) | 113 + Subtree-Mounts (`/api/books*`, `/api/admin/*`, `/api/lookup/*`) |
| Go-Handler ohne Frontend-Aufrufer | 11 sicher tot, 3 Entscheidungsfälle |
| Frontend-Aufrufe ohne Backend-Route | 1 (= der Vorab-Bug) |
| Tote Svelte-Dateien | 13–16 (5 davon einzeln zu verifizieren) |
| Svelte-4-Konstrukte (`export let`, `$:`, Dispatcher, `on:`) | **0** — Runes-Migration vollständig ✅ |
| Go-Testdateien / FE-Testdateien / E2E | 25 / 1 / 0 |
