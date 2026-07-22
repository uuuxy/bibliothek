# Master-Fahrplan: Offene Punkte bis Go-Live

> Offene Punkte Stand **2026-07-11** · gestrafft 2026-07-22 (erledigte Historie → [CHANGELOG.md](CHANGELOG.md)).
> Radar-Referenz: [`docs/api_inventar.md`](api_inventar.md) (neu erzeugen mit `./scripts/api_inventar.sh`).

## 🎯 Aktuell Offen & Nächste Schritte

### 1. Ausstehende Verifikationen (Admin-Flows)
> **Blockiert: aktuell kein LUSD-Zugriff.** Vorbereitung ist fertig:
> [`abnahme_checkliste.md`](abnahme_checkliste.md) — damit ist jede Abnahme ein ~10-Minuten-Durchlauf,
> sobald eine echte Exportdatei vorliegt.
- [ ] **LUSD-Import**: Manuelle Abnahme mit einer echten LUSD-Exportdatei durch das Sekretariat.
- [ ] **Schuljahres-Versetzung**: Manuelle Abnahme mit einem echten Klassensatz vor dem Wechsel (⏰ Deadline Schuljahreswechsel; braucht kein LUSD).
- [ ] **Klassensatz-Reservierungen**: Abnahme des "Erledigen"-Ablaufs mit einer echten Anfrage (braucht kein LUSD).

### 2. Kritischer Pfad Go-Live (wartet auf Pete)
- [ ] **Littera-MySQL-Dump + 3 Ausweis-Probe-Scans** besorgen → dann: Migrations-Tool auf echtes Littera-Schema (Titel/Exemplare mit Zugangsdatum+Preis, Leser↔LUSD-Matching, **offene Ausleihen** — ohne die startet das System mit „alles verfügbar", obwohl tausende LMF-Bücher verliehen sind).
- [ ] **Zielumgebung klären**: Server/Domain, Prod-Secrets, echter Schul-IMAP (`IMAP_HOST`) und **SMTP-Zugangsdaten** (ohne sie versendet das Mahnwesen nichts — `mail_settings_config` ist leer).
- [ ] **OPAC-Produktentscheidung**: LMF-Schulbücher erscheinen in der öffentlichen Katalogsuche — rausfiltern ja/nein? (Umsetzung: Fünfzeiler.)

### 3. Testing & Infrastruktur
- [ ] **Restore-Probe**: Datenbank-Restore-Probe gegen eine Wegwerf-DB in der Zielumgebung durchführen. Dabei den dokumentierten Cover-Reset beachten ([DEPLOYMENT.md §6](DEPLOYMENT.md)).

### 4. Offene Betreiber-Entscheidungen
> Detail + Begründung im [Invarianten-Katalog](invarianten.md) (§ Restarbeit) — hier nur als Go-Live-Merker:
- [ ] **`helfer`-Rechte entscheiden**: Die Rolle ist mit den Default-Rechten funktionsunfähig (jeder Kiosk-Scan → 403). `view_students`/`view_books` öffnen Schülerdaten — fachliche/datenschutzrechtliche Entscheidung, keine Code-Änderung.
- [ ] **Branch-Protection**: Push auf `main` umgeht die PR-Pflicht per Admin-Bypass — Regel ernst nehmen oder abschaffen.
- [ ] **Meldebestand** je LMF-Titel: ob der Default 5 gepflegt wird, ist eine Betreiber-Annahme, kein Beschluss.

### 5. Phase 3: Ausbau & Betrieb (Zukunft)
- [ ] **API-Versionierung**: Einführung von `/api/v1` inkl. Rest-Sprachvereinheitlichung (z.B. `/api/books` statt `/api/buecher`).
- [ ] **Mandantenfähigkeit (RLS)**: Tenant-Claim in Auth-Middleware, `tenant_id`-Migrationen (Dry-Run-Prozess).

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

> Abgeschlossene Phasen, Bugfixes und Radar-Zahlen: siehe [CHANGELOG.md](CHANGELOG.md).
