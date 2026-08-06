# Master-Fahrplan: Offene Punkte bis Go-Live

> Stand **2026-08-05**. Nur was offen ist — die Historie steht in `git log`, ausführlicher
> als jede gepflegte Liste. Radar-Referenz: [`api_inventar.md`](api_inventar.md)
> (neu erzeugen mit `./scripts/api_inventar.sh`).

## 🎯 Aktuell offen

### 1. Littera-Altbestand — es fehlt nur noch die richtige Datei

Der Schreibpfad ist fertig, gehärtet und gegen echtes PostgreSQL bewiesen
([SCRIPTS.md](SCRIPTS.md), Befunde in [littera_schema_befund.md](littera_schema_befund.md)).
Was fehlt, ist die Quelle:

- [x] **Titelkatalog gelöst (Stand 05.08.2026):** In `~/Developer/littera import/` liegt
      ein MAB-Export `katalogisat.xml` von **Juni 2026** mit 13.708 Titeln, der über den
      bestehenden Import direkt nutzbar ist. Damit fehlt für den Katalog nichts mehr —
      offen sind nur noch **Leser und laufende Ausleihen** (nächster Punkt).
- [ ] **Frisches Backup aus dem laufenden Littera** (Dienstprogramme → Datensicherung,
      ~100 MB+) — für Leser, Ausweisnummern und offene Ausleihen; der MAB-Export enthält
      sie nicht. Die vorliegende `littera_sav.mdb` ist ein Stand von **2010** — belegt an
      Leser 37: letzte Bewegung 11.11.2010 in der Datei gegen letzte Ausleihe 17.06.2026
      im laufenden LITTERA 5.4. Ein Buch von 2022 trägt Exemplar-Nr. 105785, die Datei
      hört bei 61.520 auf. **Es fehlen zwölf Jahre**, alle aktuellen Schüler und alle
      laufenden Ausleihen.
- [ ] Der Export muss `FremdLeserNummer` und `FremdBarcode` enthalten. Dort liegen die
      Nummern der Kartenhersteller bzw. Ersatzetiketten; in den Stammdaten stehen sie nicht.
- [x] **Barcode-Formate geklärt** (04.08.2026): Buchetikett = EAN-13 aus Exemplar- und
      Bibliotheksnummer, Schülerausweis = Herstellernummer. Der Aufdruck unter einem
      Littera-Barcode ist **nie** der Scanwert.

### 2. Ausstehende Verifikationen (Admin-Flows)

> Blockiert: kein LUSD-Zugriff. Vorbereitung ist fertig —
> [`abnahme_checkliste.md`](abnahme_checkliste.md) macht jede Abnahme zu einem
> ~10-Minuten-Durchlauf, sobald eine echte Exportdatei vorliegt.

- [ ] **LUSD-Import**: Abnahme mit einer echten Exportdatei durch das Sekretariat.
- [ ] **Schuljahres-Versetzung**: Abnahme mit einem echten Klassensatz vor dem Wechsel
      (⏰ Deadline Schuljahreswechsel; braucht kein LUSD).
- [ ] **Klassensatz-Reservierungen**: Abnahme des „Erledigen"-Ablaufs mit einer echten Anfrage.

### 3. Zielumgebung (wartet auf Pete)

- [ ] **Prod-Secrets**: `ENFORCE_PROD_SECRETS=true` plus echte `JWT_SECRET`,
      `APP_ENCRYPTION_KEY`, `POSTGRES_PASSWORD`, `BACKUP_ENCRYPTION_KEY`.
      **Ohne `BACKUP_ENCRYPTION_KEY` läuft kein Backup** — der Job überspringt sich mit
      einer Logzeile. Nach dem Setzen einmal `/api/admin/system/backup-status` im
      Admin-Dashboard ansehen, statt der Logzeile zu vertrauen.
- [ ] **Schul-IMAP** (`IMAP_HOST`, Login) und **SMTP-Zugangsdaten** (ohne sie versendet das
      Mahnwesen nichts). Für Mail gilt seit 30.07.2026 eine einzige Quelle: die Einstellung
      in der Oberfläche (`mail_settings_config`); die `SMTP_*`-Variablen werden beim ersten
      Start einmalig übernommen und sind danach nur noch Rückfall.
- [ ] **Restore-Probe** gegen eine Wegwerf-DB in der Zielumgebung. Der automatisierte
      Round-Trip läuft (`jobs/backup_drill_pg_test.go`); was fehlt, ist der Durchlauf am
      echten Ziel inklusive Cover-Reset ([DEPLOYMENT.md](DEPLOYMENT.md) §6).
- [ ] **Branch-Protection**: Entscheidung vom 30.07.2026 ist, die PR-Pflicht abzuschaffen
      (Solo-Entwicklung). „Block force pushes" und „Restrict deletions" sollten an bleiben —
      sie kosten nichts und sind der einzige Schutz gegen ein Überschreiben der Historie.
      *Auszuführen in den GitHub-Einstellungen, keine Code-Änderung.*

### 4. Phase 3: Ausbau & Betrieb (Zukunft)

- [ ] **API-Versionierung**: `/api/v1` inkl. Sprachvereinheitlichung (`/api/books` statt
      `/api/buecher`).
- [ ] **Mandantenfähigkeit (RLS)**: Tenant-Claim in der Auth-Middleware, `tenant_id`-Migrationen.

---

## 🛑 Das Parkdeck (unangetastet)

| Thema | Warum geparkt |
|---|---|
| **Integer-Cent-Refactor** (Go `float64`, DB `NUMERIC(10,2)`) | Bewusste, dokumentierte Nicht-Entscheidung |
| **Bundle-Splitting** (720-kB-Chunk) | Performance-Feinschliff, kein Stabilitätsthema |
| **TypeScript-Migration** | Das Frontend hat null TS-Dateien; `typescript` ist nur noch Peer-Abhängigkeit der Werkzeuge |
| **Verschmelzung `inventur/` ins Haupt-API** | Rechte sind angeglichen (T6); Struktur bleibt |
| **Edge-to-Edge-Feinschliff restlicher Views** | UI-Refactoring abgeschlossen; kein Re-Opening ohne Anlass |
| **`cmd/migrate` (MySQL)** | Hat keine Datenquelle mehr — der Dump wurde nie geliefert. Löschen ist eine eigene Entscheidung, seine PG-Tests sichern mit `internal/uebernahme` geteilten Code ab |
