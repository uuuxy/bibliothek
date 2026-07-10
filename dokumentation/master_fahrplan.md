# Master-Fahrplan: Radar-Analyse & Konsolidierung

> Stand: **2026-07-09** · Lebendes Dokument.
> Radar-Referenz: [`dokumentation/api_inventar.md`](api_inventar.md) (neu erzeugen mit `./scripts/api_inventar.sh`).

## 🎯 Aktuell Offen & Nächste Schritte

### 1. Ausstehende Verifikationen (Admin-Flows)
- [ ] **LUSD-Import**: Manuelle Abnahme mit einer echten LUSD-Exportdatei durch das Sekretariat.
- [ ] **Schuljahres-Versetzung**: Manuelle Abnahme mit einem echten Klassensatz vor dem Wechsel.
- [ ] **Klassensatz-Reservierungen**: Abnahme des "Erledigen"-Ablaufs mit einer echten Anfrage.
- [ ] **Cleanup**: Nach erfolgreicher LUSD-Abnahme entscheiden, ob das alte `LusdImportModal` + `/api/import/lusd` gestrichen wird.
- [x] **Cleanup**: Toter Kiosk-Parallelbau (`stores/kiosk.svelte.js`, `components/kiosk/`) am 10.07. entfernt — war nirgends eingebunden, der Omnibox-Flow ist der produktive Ausleihe-Pfad.

### 1b. Produktentscheidung Fremdscan in aktiver Sitzung — ENTSCHIEDEN & UMGESETZT (10.07.)
- [x] **Petes Entscheidung: Doppel-Scan ohne Dialog.** Fremdes Buch in aktiver Sitzung → NUR Rückgabe beim Vorbesitzer, Info per Toast + Banner („war auf Max verbucht — dort zurückgegeben. Erneut scannen, um es an Lisa auszuleihen"), null Klicks, Workflow läuft ungestört weiter. Zweiter Scan leiht das nun freie Buch normal an die Sitzung aus (deckt den Altlasten-Fall ab). Ohne Sitzung bleibt der Scan eine reine Rückgabe; manuelle Rücknahme (Profil-Button) unverändert. Umgesetzt in loan_checkout_cases.go (handleForeignReturn ohne Auto-Umbuchen), Omnibox-Store/-Banner, Fremdrückgabe-E2E-Test schreibt das neue Verhalten fest.

### 2. Testing & Infrastruktur
- [x] **E2E-Tests**: Playwright-Tests für die drei neuen Admin-Flows (Versetzung, LUSD, Reservierungen) erfolgreich integriert.
- [x] **E2E-Alltagsflüsse** (10.07.): Rückgabe, Fremdrückgabe, Mahnwesen (+PDF-Smoke), Schadensfall — Suite: 14 Flows, läuft jetzt auch in der CI (e2e-Job).
- [ ] **Restore-Probe**: Datenbank-Restore-Probe gegen eine Wegwerf-DB in der Zielumgebung durchführen.

#### E2E-Lücken-Analyse (10.07., extern angeregt — Bewertung am Code)
- [x] **P1 — RBAC-Negativpfad** (`permissions.spec.js`, 10.07.): Mitarbeiter → DSGVO-Auskunft/Backup-Status 403 + Badge unsichtbar; Lehrer → `/abgaenger` leakt nichts, Benutzer-Anlage 403.
- [x] **P2 — Kiosk-Scan-Dauerfeuer** (`kiosk-dauerfeuer.spec.js`, 10.07.): 3 Scans ohne Pause → alle 3 verbucht, Zähler stimmt.
- [x] **P3 — LUSD-Schrottdatei** (10.07.): E2E für falsche Header + Binärmüll; Parser-Fehlermeldungen dabei auf verständliches Deutsch umgestellt („Pflichtspalte 'lusd_id' fehlt in der CSV-Kopfzeile — ist das die richtige LUSD-Exportdatei?").
- [x] **P4 — Massendaten** (`massendaten.spec.js`, 10.07.): 2.000 Schüler + 50.000 Ausleihen + 100 überfällige — Schülerdatei-Suche und Mahnwesen antworten in <2s; Daten werden nach dem Test wieder entfernt (MASS-Präfix).
- [x] **P5 — Mehrplatz-Livesync** (`sse-livesync.spec.js`, 10.07.): Zwei Browser-Kontexte — Rückgabe an PC A entfernt das Buch ohne Reload aus dem offenen Konto an PC B (SSE bewiesen).
- [x] **Race Condition Doppel-Checkout**: bereits auf DB-Ebene hart garantiert (Migration 033 „≤ 1 aktive Ausleihe je Exemplar", dazu Idempotenz-Keys). Klassensatz-Reservierungen sind eine Merkliste ohne knappen Bestand — kein echtes Race-Gut. E2E-Race-Tests mit parallelen Browser-Kontexten wären flaky; falls überhaupt, als Go-Concurrency-Test.

#### CI-Budget (privates Repo → 2.000 Actions-Minuten/Monat)
- [ ] **Entscheidung nötig**: (a) Repo public machen ⇒ Minuten unbegrenzt kostenlos (prüfen: nichts Sensibles in Historie), (b) e2e-Job nur auf PRs + `concurrency: cancel-in-progress` (spart Push-Serien), oder (c) Self-hosted Runner auf eigenem Rechner/Server. Bis dahin frisst der Docker-Build im e2e-Job das Kontingent am schnellsten.

### 3. Phase 3: Ausbau & Betrieb (Zukunft)
- [ ] **API-Versionierung**: Einführung von `/api/v1` inkl. Rest-Sprachvereinheitlichung (z.B. `/api/books` statt `/api/buecher`)
- [ ] **Mandantenfähigkeit (RLS)**: Tenant-Claim in Auth-Middleware, `tenant_id`-Migrationen (Dry-Run-Prozess).

---

## ✅ Kürzlich Erledigt (Go-Live Ready)

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
