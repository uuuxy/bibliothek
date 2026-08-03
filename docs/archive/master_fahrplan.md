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
- [ ] **Zielumgebung klären**: Server/Domain, Prod-Secrets, echter Schul-IMAP (`IMAP_HOST`) und **SMTP-Zugangsdaten** (ohne sie versendet das Mahnwesen nichts). *Seit 30.07.2026 gilt für Mail eine einzige Quelle: die Einstellung in der Oberfläche (`mail_settings_config`). Die `SMTP_*`-Variablen werden beim ersten Start einmalig übernommen und sind danach nur noch Rückfall — ein Serverwechsel braucht keinen Container-Neustart mehr.*
- [x] **OPAC-Produktentscheidung** (30.07.2026): LMF-Schulbücher sind aus der öffentlichen Katalogsuche ausgefiltert — sie werden klassensatzweise zugeteilt, nicht recherchiert. Nur die öffentliche Suche filtert; Verwaltung, Inventur, Bestellwesen und Klassensatz-Reservierung finden sie unverändert (`api/opac.go`, E2E-Beleg in `opac-suche.spec.js`).

### 3. Testing & Infrastruktur
- [ ] **Restore-Probe**: Datenbank-Restore-Probe gegen eine Wegwerf-DB in der Zielumgebung durchführen. Dabei den dokumentierten Cover-Reset beachten ([DEPLOYMENT.md §6](DEPLOYMENT.md)).

### 4. Offene Betreiber-Entscheidungen
> Detail + Begründung im [Invarianten-Katalog](invarianten.md) (§ Restarbeit) — hier nur als Go-Live-Merker:
- [x] **`helfer`-Katalogzugriff** (30.07.2026): **Ja.** Ein Helfer an der Theke ist die erste Anlaufstelle für „Habt ihr Band 3 noch da?" und musste dafür bisher jede Frage weiterreichen. Rein lesend; die Grenze zu Personendaten zieht weiterhin `view_students`. Umgesetzt per Migration 055 (die Vorgabe in `seed.go` allein hätte die laufende Installation nicht erreicht).
- [ ] **Branch-Protection**: Push auf `main` umgeht die PR-Pflicht per Admin-Bypass. **Entscheidung 30.07.2026: PR-Pflicht abschaffen** (Solo-Entwicklung, die Regel ist derzeit Zeremonie). Empfehlung dazu: „Block force pushes" und „Restrict deletions" eingeschaltet lassen — sie kosten im Alltag nichts und sind der einzige Schutz gegen ein versehentliches Überschreiben der Historie. *Auszuführen in den GitHub-Einstellungen, keine Code-Änderung.*
- [x] **Meldebestand** je LMF-Titel (30.07.2026): Wird **nicht** gepflegt und bekommt keine eigene Oberfläche — der Bestellbedarf richtet sich nach der konfigurierbaren Schwelle. Das Feld war allerdings nicht folgenlos: Die Bestellliste für den Händler rechnete ihre Menge daraus und konnte negativ werden. Behoben, die Menge folgt jetzt dem Soll-Bestand (`api/reorders_pdf.go`).

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
