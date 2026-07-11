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

### 2. Testing & Infrastruktur
- [ ] **Restore-Probe**: Datenbank-Restore-Probe gegen eine Wegwerf-DB in der Zielumgebung durchführen.

### 3. Phase 3: Ausbau & Betrieb (Zukunft)
- [ ] **API-Versionierung**: Einführung von `/api/v1` inkl. Rest-Sprachvereinheitlichung (z.B. `/api/books` statt `/api/buecher`)
- [ ] **Mandantenfähigkeit (RLS)**: Tenant-Claim in Auth-Middleware, `tenant_id`-Migrationen (Dry-Run-Prozess).

## ISBN-Katalogisierung & Altersstufen-Automatik — UMGESETZT (11.07.)

### 1. Datenquellen-Entscheidung (revidiert)
* **DNB SRU direkt statt Lobid:** Der ursprüngliche Lobid-Plan ("spart MARC21-Parsing-Boilerplate") war beim Code-Abgleich überholt — der MARC21-XML-Parser existierte längst und ist produktiv (`inventur/metadaten_anbieter.go`, Fallback-Kette DNB → Google Books → OpenLibrary). Lobid bleibt als auskommentiertes Backup im Code.
* **Inhaltlicher Hebel bestätigt:** Die DNB liefert die Altersstufen strukturiert — MARC **655** (Genre: "Kinderbuch", "Kinderbücher bis 11 Jahre", "Jugendbücher ab 12 Jahre") und **653** mit Präfix `(Zielgruppe)` (z. B. "ab 10 Jahre"). Live gegen die echte API verifiziert.

### 2. Umgesetzter Workflow (Neuanschaffungen)
1. ISBN-Scan/Eingabe im Buchformular → `GET /api/lookup/{isbn}` (bestehende Route).
2. `sucheDNB` extrahiert zusätzlich 655-Genres + 653-Zielgruppe.
3. `leiteBibKategorieAb` (metadaten_helfer.go) mappt auf die Signatur-Kategorien der Schülerbücherei aus `signatur_optionen.js`: **Manga > Comic > Jugendbuch > Kinderbuch** (Genre-Treffer), Fallback über die Altersgrenze (ab 12 = Jugendbuch, darunter Kinderbuch). Kein Treffer = kein Vorschlag — Sachbücher/Romane bleiben Handarbeit.
4. Response enthält `zielgruppe` + `bibKategorie`; IsbnFeld/BuchFormular befüllen das leere Signatur-Pflichtfeld mit `BIB {Kategorie}` vor. Eine vorhandene Signatur wird nie überschrieben (Guard-Muster).

### 3. LMF-Import (Littera-Altbestand) — UMGESETZT (11.07.)
* **Erkennung:** Littera kennzeichnet Lernmittelfreiheit uneinheitlich — Signatur-Präfix (`LMF Bio 7`, MAB 700), Standort-Feld (MAB **108a**: "LMF", "LMF/Bibliothek") oder CSV-Kategorie-Token ("Buch LMF Ma 6/Gri"). `internal/service/import_lmf.go` erkennt alle drei Varianten (Token an Wortgrenze, "Filmfest" schlägt nicht an).
* **Verarbeitung:** LMF-Token wird aus der Signatur geschnitten (reine Fach-Signatur wie auf dem Rücken-Etikett bleibt), der Titel bekommt das Projekt-Präfix **`LMF-`** — daran erkennen Leihfristen (loan_rules), Statistik und Massenverlängerung den Schulbuch-Bestand bereits heute. Eingebaut in alle drei Import-Pfade (ParseLitteraXML, ImportDynamic, ImportLitteraBestand); in ImportDynamic vor dem Titel-Matching, damit beide Pässe denselben Schlüssel nutzen.
* **🐛 Bestandsbug gefunden & behoben:** Der XML-Import hat die Signatur **noch nie übernommen** — `feld.MAB` wurde getrimmt, der Switch-Case verglich aber mit `"700 "` (trailing Space, toter Case). Realdaten-Probelauf gegen das echte `katalogisat.xml`: 13.708 Titel, davon jetzt **100 % mit Signatur** (vorher 0) und **480 LMF-geflaggt**.
* **Datenlage:** `katalogisat.xml` (14.858 Titel, saubere Felder, keine Barcodes) + `clean_import.csv` (30.658 Exemplare mit Barcodes, ~48 % LMF, aber verrutschte Spalten aus der PDF-Konvertierung). Empfohlene Import-Reihenfolge: **XML zuerst** (Titel + Signaturen), **dann CSV** (Exemplare/Barcodes, matcht per ISBN/Titel).

### 4. Abgrenzung & Randfälle
* **API-Ausfall/Lücke:** Kennt keine der drei Quellen die ISBN (z. B. Schenkung), greift die manuelle Eingabe im Svelte-Formular; das Signatur-Pflichtfeld bleibt der menschliche Kontrollpunkt.
* **Offen (bewusst nicht gebaut):** Kein `ist_schulbuch`-Schemafeld — die etablierte `LMF-`-Titel-Konvention ist projektweit verdrahtet (loan_rules, stats, ausleihe, Frontend). Ein Schema-Refactor wäre eine eigene Entscheidung nach dem Go-Live.

---

## ✅ Kürzlich Erledigt (Go-Live Ready)

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
