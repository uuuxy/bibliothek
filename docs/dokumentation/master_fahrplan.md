# Master-Fahrplan: Radar-Analyse & Konsolidierung

> Stand: **2026-07-11** · Lebendes Dokument.
> Radar-Referenz: [`dokumentation/api_inventar.md`](api_inventar.md) (neu erzeugen mit `./scripts/api_inventar.sh`).

## 🎯 Aktuell Offen & Nächste Schritte

### 1. Ausstehende Verifikationen (Admin-Flows)
> **Blockiert: aktuell kein LUSD-Zugriff** (Stand 11.07.). Vorbereitung ist fertig:
> [`abnahme_checkliste.md`](abnahme_checkliste.md) — damit ist jede Abnahme ein ~10-Minuten-Durchlauf,
> sobald eine echte Exportdatei vorliegt.
- [ ] **LUSD-Import**: Manuelle Abnahme mit einer echten LUSD-Exportdatei durch das Sekretariat.
- [ ] **Schuljahres-Versetzung**: Manuelle Abnahme mit einem echten Klassensatz vor dem Wechsel (⏰ Deadline Schuljahreswechsel; braucht kein LUSD).
- [ ] **Klassensatz-Reservierungen**: Abnahme des "Erledigen"-Ablaufs mit einer echten Anfrage (braucht kein LUSD).

### 2. Kritischer Pfad Go-Live (wartet auf Pete, Stand 11.07.)
- [ ] **Littera-MySQL-Dump + 3 Ausweis-Probe-Scans** besorgen → dann: Migrations-Tool auf echtes Littera-Schema (Titel/Exemplare mit Zugangsdatum+Preis, Leser↔LUSD-Matching, **offene Ausleihen** — ohne die startet das System mit „alles verfügbar", obwohl tausende LMF-Bücher verliehen sind).
- [ ] **Zielumgebung klären**: Server/Domain, Prod-Secrets, echter Schul-IMAP (`IMAP_HOST`) und **SMTP-Zugangsdaten** (ohne sie versendet das Mahnwesen nichts — `mail_settings_config` ist leer).
- [ ] **OPAC-Produktentscheidung**: LMF-Schulbücher erscheinen in der öffentlichen Katalogsuche — rausfiltern ja/nein? (Umsetzung: Fünfzeiler.)

### 3. Testing & Infrastruktur
- [ ] **Restore-Probe**: Datenbank-Restore-Probe gegen eine Wegwerf-DB in der Zielumgebung durchführen. Dabei den dokumentierten Cover-Reset beachten ([DEPLOYMENT.md §6](DEPLOYMENT.md)).

### 4. Phase 3: Ausbau & Betrieb (Zukunft)
- [ ] **API-Versionierung**: Einführung von `/api/v1` inkl. Rest-Sprachvereinheitlichung (z.B. `/api/books` statt `/api/buecher`)
- [ ] **Mandantenfähigkeit (RLS)**: Tenant-Claim in Auth-Middleware, `tenant_id`-Migrationen (Dry-Run-Prozess).

---

## ✅ Kürzlich Erledigt (Go-Live Ready)

- **Produktions-Import-Fehler behoben (11.07. spät)** — Ursache per Prod-Log statt Verdacht geklärt:
  - **🐛 500 beim Bestands-Import** war ein **Format-Mismatch, kein DB-Fehler**: `/api/admin/import-bestand` las hart mit Semikolon und verlangte 8 Spalten; eine komma-/7-Spalten-Datei wurde als **500** statt 400 abgewiesen. Zwei divergierende Import-Pfade (starrer `ImportLitteraBestand` vs. robuster `ImportDynamic`) auf den robusten **vereinheitlicht**: Trennzeichen-Erkennung, namensbasiertes Spalten-Mapping, optionale Zustand-Spalte (`ist_ausleihbar`/`zustand_notiz`), Formatfehler → **400 mit klarer Meldung**. `ImportLitteraBestand` entfernt.
  - **🐛 XML-„Netzwerk-Timeout"** war der **10s-Client-Timeout** (apiFetch) gegen einen **N+1**: `ParseLitteraXML` machte 14.858 Einzel-Upserts (~60s gegen nicht-lokale DB). Ersetzt durch **einen gepipelineten `BulkUpsertBookTitles`** (Repo, alles-oder-nichts-TX) → 1s lokal, ~1 Rundreise statt 14.858. Upload-Timeout im Frontend für FormData auf 5min (abgestimmt auf Caddy 300s).
  - **🐛 Empty-ISBN-Duplikat-Bug**: Titel ohne ISBN wurden bei **jedem** Import neu angelegt (DB wuchs pro Lauf um Zehntausende). Jetzt Titel-Dedup in beiden Pfaden → Re-Import legt **0** Duplikate an (lokal verifiziert: Titelzahl bleibt konstant).
  - Verifiziert gegen die echten Dateien im lokalen Stack: 500→200, 400 bei kaputtem Format, Zustand=verliehen sperrt, XML idempotent.

- **Zweite Lücken-Analyse validiert (11.07. spät)**: Von 5 behaupteten E2E-Lücken hielten 2 der Prüfung stand. Littera-Import-Claim war falsch (buecher-crud.spec.js testet den echten Endpoint; dazu Realdaten-Probelauf + Go-Fixtures — bestgetesteter Import im Haus). Neu abgedeckt: **Etikettendruck** (beide PDF-Pfade: Titel-Bogen + /api/print/labels, echter PDF-Smoke) und **Mail-Template-Roundtrip** (PUT mit Restore, Platzhalter-Erhalt). Lieferanten-Anlage war schon gedeckt (bestellungen.spec.js), Fotos → Backlog. Suite: **33 Flows, 2× hintereinander grün**.
- **E2E-Nachschärfung + Produktbug im Lehrerportal (11.07. abends)**:
  - **🐛 Produktbug: Lehrerportal-Suche war komplett kaputt** — `getForm()` mutierte `$state` während des Template-Renderns (`{@const form = getForm(...)}`), Svelte 5 wirft `state_unsafe_mutation` und bricht das Rendern der Suchtreffer ab. Lehrkräfte konnten real keine Klassensätze reservieren. Fix: `ensureForm()` (nur Event-Handler) von der reinen Lese-Sicht `getForm()` getrennt. Gefunden durch den neuen `lehrer-reservierung`-E2E — der Klassensatz-Spec hatte per SQL geseedet und den UI-Pfad nie betreten.
  - **3 der 5 neuen E2E-Specs waren nicht idempotent** (fixe ISBNs/Barcodes statt `uniqueSuffix()` → zweiter Lauf kollidierte am Unique-Index) und `lmf-massenverlaengerung` assertete nur `toContain('Erfolgreich')` — das erscheint auch bei 0 verlängerten Ausleihen (und es WAREN 0: Seed-Titel „LMF Titel" ohne Bindestrich matchte `ILIKE 'LMF-%'` nie). Alle drei auf Suite-Konvention umgeschrieben, harte DB-Asserts ergänzt. Suite 2× hintereinander grün (31/31).
- **Betriebsvorbereitung (11.07. nachmittags)**:
  - **Cover-Sync gedrosselt**: Der bestehende 6-h-Job lief ungedrosselt (8 Worker, bis zu ~5 HTTP-Calls/Titel) — beim Altbestands-Backfill (~14k Titel) reales IP-Block-Risiko bei der DNB, was auch den ISBN-Lookup am Pult getroffen hätte. Jetzt global 2 Titel/s (kompletter Altbestand in ~2 h).
  - **Backup-Scope bewusst entschieden & dokumentiert** (DEPLOYMENT.md §6): nur DB im Backup; Schülerfotos liegen verschlüsselt in der DB, Cover sind reproduzierbar (inkl. dokumentiertem Einmal-Reset nach Volume-Verlust).
  - **[Runbook fürs Sekretariat](runbook_sekretariat.md)**: Tagesgeschäft, Sonderfälle, Störungen (Offline-Pufferung!, Scanner defekt → Tastatur), Wochen-Checkliste, Eskalation.
  - **PR-Triage-Nachgang**: #274 nach Verifikation gemergt (peek via timestamp-Index), #275 geschlossen (No-Op — Branch-Diff gegen main war leer), #235-Bewertung revidiert (kein echter HIGH-Fund; `filepath.Base` + ISBN-Validierung decken den Pfad ab). 0 offene PRs.
- **Lücken-Analyse & E2E-Vervollständigung (11.07.)**:
  - Validierung der 7 behaupteten Lücken gegen den echten Testbestand durchgeführt.
  - Lehrer-Portal & OPAC waren bereits abgedeckt.
  - Fehlende E2E-Specs für *LMF-Massenverlängerung*, *Abgänger-Management*, *E-Mail-Konfiguration*, *System-Logs* und *Monitor-Slideshow* implementiert.
  - Etabliertes Suite-Muster (Seeds, `uiLogin`, DB-Cleanup) konsequent fortgeführt, alle 23 Specs (inkl. der neuen 5) laufen lokal grün.

- **ISBN-Katalogisierung & Altersstufen-Automatik (11.07.)**:
  - **Datenquellen-Entscheidung (revidiert)**: DNB SRU direkt statt Lobid (MARC21-XML-Parser bereits produktiv).
  - **Workflow**: `GET /api/lookup/{isbn}` extrahiert DNB 655/653-Daten und mappt Signaturen (Manga > Comic > Jugendbuch > Kinderbuch) als Vorschlag ins Frontend-Formular.
  - **LMF-Import (Littera-Altbestand)**: Erkennung über Signatur-Präfix, MAB 108a und CSV-Kategorie. Behebung eines MAB 700 XML-Parsing-Bugs (trailing Space). Lokaler Testlauf mit 13.708 Titeln und 30.658 Exemplaren erfolgreich validiert.
  - **Abgrenzung**: Keine `ist_schulbuch`-Tabellenspalte (Projekt nutzt etablierte `LMF-`-Titelpräfix-Konvention). API-Ausfall fängt manuelle Eingabe ab.
- **E2E-Absicherung Runde 2 & CI-Hygiene (11.07.)**:
  - **Inventur-Ablauf** (`inventur.spec.js`): Signatur-Scope, Scan, Abschluss. **Fand Bug: JEDER Inventur-Abschluss war ein 500** — nicht existente SQL-Funktion `update_verfuegbar_count` brach die Finish-Transaktion ab (25P02). Behoben.
  - **Bücher-CRUD + Signatur** (`buecher-crud.spec.js`): Anlegen, Exemplare, Katalog-Suche, Littera-Import-Schutz. **Fand Bug: Create/Update-Handler verwarfen das signatur-Feld** — das Pflichtfeld des Formulars kam nie in der DB an. Behoben.
  - **Settings-Enforcement** (`settings-enforcement.spec.js`): Limit=1 → zweiter Checkout blockt sofort, Reset im finally.
  - **Papierkorb-Flow** (`papierkorb.spec.js`): löschen (Tipp-Bestätigung) → wiederherstellen; Schadensfall blockt Löschung. **Fand Bug: Papierkorb-Liste war ein 500** — timestamptz in *string gescannt. Behoben.
  - **Katalog-Suche**: in buecher-crud.spec.js integriert (Suche & Filter-Tab).
  - **Offline-Queue als Vitest-Unit** (`offlineSync.test.js`, fake-indexeddb): Idempotenz-Keys, Batch-Sync, 4xx-Dequeue, 502-Retention.
  - **Git Hook & CI-Hygiene**:
    - **Petes Entscheidung 11.07.**: Repo bleibt auf jeden Fall PRIVAT — Option „public" ist gestrichen.
    - **Sofort-Hygiene** (11.07.): `concurrency: cancel-in-progress` in ci.yml — Push-Serien verbrennen keine Minuten mehr für veraltete Läufe.
    - **Lösung (11.07.): pre-push-Git-Hook** (`scripts/git-hooks/pre-push`) — jeder `git push` läuft erst durch Go-Build+Tests, Vitest, Container-Rebuild und die volle Playwright-Suite; rot = Push blockiert. Aktivierung pro Klon einmalig: `git config core.hooksPath scripts/git-hooks`. Notausgang: `SKIP_E2E=1 git push` (nur Go+Vitest) oder `git push --no-verify`. Die GitHub-Actions-CI bleibt als Definition bestehen, falls später doch ein Self-hosted Runner kommt — bis dahin ist der Hook die verbindliche Prüfinstanz.
  - **Cleanup**: Nach erfolgreicher LUSD-Abnahme entscheiden, ob das alte `LusdImportModal` + `/api/import/lusd` gestrichen wird. Bereits erledigt (09.07.) — Code-Prüfung 11.07.: kein Treffer mehr für `LusdImportModal` oder `/api/import/lusd`, es existiert nur noch der getestete Preview-Flow (`/api/lusd/preview` + `/api/lusd/import`).
- **Fremdscan, Testing & E2E-Absicherung (10.07.)**:
  - **Fremdscan in aktiver Sitzung**: Doppel-Scan ohne Dialog implementiert (fremdes Buch in aktiver Sitzung führt direkt zur Rückgabe beim Vorbesitzer mit Info-Banner/Toast, ein zweiter Scan leiht es normal an die neue Sitzung aus). Umgesetzt in `loan_checkout_cases.go`.
  - **Kiosk-Code-Bereinigung**: Toter Kiosk-Parallelbau (`stores/kiosk.svelte.js`, `components/kiosk/`) entfernt (der Omnibox-Flow ist der produktive Ausleihe-Pfad).
  - **E2E-Tests**: Playwright-Tests für die drei neuen Admin-Flows (Versetzung, LUSD, Reservierungen) erfolgreich integriert.
  - **E2E-Alltagsflüsse**: Rückgabe, Fremdrückgabe, Mahnwesen (+PDF-Smoke), Schadensfall laufen stabil in der CI (14 Flows).
  - **E2E-Lücken-Analyse**:
    - **P1 — RBAC-Negativpfad**: Mitarbeiter → DSGVO-Auskunft/Backup-Status 403 + Badge unsichtbar; Lehrer → `/abgaenger` leakt nichts, Benutzer-Anlage 403.
    - **P2 — Kiosk-Scan-Dauerfeuer**: 3 Scans ohne Pause → alle 3 verbucht, Zähler stimmt.
    - **P3 — LUSD-Schrottdatei**: E2E für falsche Header + Binärmüll; Parser-Fehlermeldungen auf Deutsch optimiert.
    - **P4 — Massendaten**: 2.000 Schüler + 50.000 Ausleihen + 100 überfällige — Suche und Mahnwesen antworten in <2 Sekunden.
    - **P5 — Mehrplatz-Livesync**: SSE-Synchronisation bewiesen (Rückgabe an PC A aktualisiert Konto an PC B live).
    - **Race Condition Doppel-Checkout**: Auf DB-Ebene hart garantiert (Migration 033 „≤ 1 aktive Ausleihe je Exemplar", Idempotenz-Keys).
- **Backup-Wächter & DSGVO-Auskunft (09.07.)**:
  - Backup-Status-Endpoint & UI-Badge zur aktiven Überwachung implementiert.
  - DSGVO-Auskunft (Art. 15) als JSON-Export im Svelte-Schülerprofil integriert inkl. automatischer Audit-Protokollierung. Letzte Go-Live-Empfehlungspunkte damit abgeschlossen.
- **Klassensatz-Reservierung „erledigen" (09.07.)**:
  - Neue Listen-Ansicht und Abschluss-Flow im BestellWorkspace für Administratoren.
- **LUSD-Konsolidierung & Versetzung (09.07.)**:
  - `LusdImportView` und `PromoteStudentsView` inkl. sicherem Dry-Run/Preview eingebunden.
  - Harter Rollback-Schutz und 30%-Abgang-Massenbremse (409) in den Backend-Handlern umgesetzt.

---

## 🗄️ Historie & Abgeschlossene Phasen

<details>
<summary><b>Klicken zum Ausklappen der abgeschlossenen Phasen</b></summary>

### Phase 1: Dead Code & Cleanup (07.07.)
- **11 tote Go-Handler** und **16 tote Svelte-Dateien** gelöscht.
- Geister-Aufruf in Omnibox behoben (`POST /api/schueler/{id}/update` -> `PATCH /api/admin/students/{id}/lock`).
- `Undo-Return`-Feature mangels Nutzung komplett entfernt.

### Phase 2: Die Test-Festung T1-T7 (07.07. - 08.07.)
- **T1 Wareneingang**: 13 Vitest-Tests (Dedup, Race-Conditions).
- **T2 Datumsgrenzen**: Regressionstests für Schaltjahre und Zeitzonen.
- **T3 Auth-Lebenszyklus**: Session-Restore beim Boot, 30-min-Refresh-Loop, invalidierender Logout.
- **T4 Mahnwesen**: Bugfix für verschluckte Medien bei gleichnamigen Schülern (Slice-Reallokation).
- **T5 E2E-Playwright**: 3 Smoke-Flows stabil implementiert (`npm run test:e2e`).
- **T6 Inventur-Rechte**: RBAC-Benennung geglättet (`RequireViewBooks`).
- **T7 Betriebspflichten**: Migration 035 (soft-deleted `lusd_id` Wiederanmeldung) real getestet. Null-Wert in Setting-Bug gefixt.

### PR-Backlog & Triage (08.07.)
- **15 Alt-PRs abgeräumt**: 6 gemergt (Security, a11y), 9 mit Begründung geschlossen (Duplikate, Security-Downgrades). Keine offenen Alt-PRs mehr.

</details>

---

## 🛑 Das Parkdeck (Unangetastet)

| Thema | Warum geparkt |
|---|---|
| **Integer-Cent-Refactor** (Go `float64`, DB `NUMERIC(10,2)`) | Bewusste, dokumentierte Nicht-Entscheidung |
| **Bundle-Splitting** (720-kB-Chunk) | Performance-Feinschliff, kein Stabilitätsthema |
| **TypeScript-Migration** | JSDoc-Typedefs reichen aktuell |
| **Verschmelzung `inventur/` ins Haupt-API** | Rechte sind angeglichen (T6); Struktur bleibt |
| **Edge-to-Edge-Feinschliff restlicher Views** | UI-Refactoring abgeschlossen; kein Re-Opening ohne Anlass |

---

## 📊 Radar-Zahlen (Stand 08.07.)

| Metrik | Radar 07.07. | Jetzt |
|---|---|---|
| Geister-Aufrufe (FE ohne Backend-Route) | 1 | **0** |
| Tote Go-Handler | 11 + 3 Fälle | **0** (3 bewusste Entscheidungsfälle dokumentiert) |
| Tote Svelte-Dateien | 13–16 | **0** |
| Svelte-4-Konstrukte | 0 | 0 (Runes-Migration vollständig) |
| Go-Testdateien / FE-Testdateien / E2E-Flows | 25 / 1 / 0 | **30 / 4 / 3** |
| Bekannte offene UX-Defekte | — | **0** |
