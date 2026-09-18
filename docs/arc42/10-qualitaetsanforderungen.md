# 10. Qualitätsanforderungen

Stand: 17.09.2026

---

## 10.1 Qualitätsbaum

```
Qualität
├── Q1 Korrektheit (Funktionale Eignung + Zuverlässigkeit)   ← höchste Priorität
│   ├── Keine Doppelbuchung bei parallelen Stationen                    → S1, S2
│   ├── Kein Phantom-Erfolg: ein Fehler ist ein Fehler                  → S3
│   ├── Fristen sind fachlich richtig (LMF, Freihand, Lehrkraft)        → S4
│   ├── Wiederherstellbarkeit nachgewiesen, nicht behauptet             → S5
│   └── Eine abgebrochene Abfrage liefert keine halbe Liste              → S24
├── Q2 Datenschutz (Sicherheit + Compliance)
│   ├── Öffentliche Pfade tragen keine Personendaten                    → S6
│   ├── Rechte greifen serverseitig, nicht nur in der Oberfläche        → S7
│   ├── Löschfristen laufen unbeaufsichtigt und in richtiger Folge      → S8
│   └── Kein Passwort, kein Klartext-Geheimnis im Betrieb               → S9
├── Q3 Betriebstransparenz (Analysierbarkeit)
│   ├── Eine untätige Funktion meldet sich selbst                       → S10
│   └── „Läuft der Container aus diesem Commit?" ist beantwortbar        → S11
├── Q4 Bedienbarkeit (Benutzbarkeit + Barrierefreiheit)
│   ├── Ein Scan, eine Reaktion — auch bei alten Aufdrucken             → S12
│   ├── Netzausfall hält den Betrieb nicht an                           → S13
│   ├── Bedienbar ohne Maus, WCAG 2.1 AA                                → S14
│   └── Sichtschutz am Mehrplatzrechner                                 → S15
└── Q5 Änderbarkeit (Wartbarkeit)
    ├── Ein behobener Fehler bleibt behoben                             → S16
    ├── Zwei Wahrheiten laufen nicht auseinander                        → S17
    └── Ein Deploy unterbricht den Betrieb nicht dauerhaft              → S18
```

---

## 10.2 Qualitätsszenarien

Spalte **Nachweis** nennt das Gate, das die Zusage verliert, wenn sie nicht mehr gilt. Wo
kein Gate steht, ist die Zusage nur so gut wie die Sorgfalt — das ist dann ausdrücklich
vermerkt.

### Q1 Korrektheit

| ID  | Auslöser / Situation                                                                                 | Erwartete Reaktion                                                                                                 | Nachweis                                                                                            |
| --- | ---------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------- |
| S1  | Zwei Stationen scannen **dasselbe Exemplar** für **zwei verschiedene** Leser, im selben Moment        | Eine Buchung gelingt, die andere endet mit **409 Conflict** — nie zwei aktive Ausleihen                             | `uniq_ausleihen_aktiv_exemplar` (DB) + `repository/loan_konflikt_pg_test.go`, `damage_race_test.go` |
| S2  | Derselbe Scan kommt zweimal an (Netz, Doppeldruck, Nachbuchen)                                        | Die gespeicherte Antwort wird zurückgegeben; **kein** zweiter Schreibvorgang, **keine** zweite Sperrmeldung          | `repository/idempotenz_uebernahme_pg_test.go`, `api/idempotenz_fristen_test.go`                     |
| S3  | Ein `UPDATE` trifft null Zeilen (Sicht, falsche ID, Wettlauf)                                          | Fehler oder 404 — **niemals** „erfolgreich"                                                                         | `phantom_erfolg_test.go`, `docs/schreibpfade_gegen_sicht_test.go`                                   |
| S4  | Ein Lernmittel wird nach dem Rückgabetermin der Klasse ausgeliehen                                     | Frist ist der Stichtag des **folgenden** Schuljahres, nicht ein vergangener Termin                                  | `internal/service/loan_rules_test.go`, `repository/lmf_termine_frist_pg_test.go`                    |
| S5  | Das jüngste verschlüsselte Backup soll wiederhergestellt werden                                         | Wöchentliche Probe spielt es in eine Wegwerf-Datenbank ein; Fehlschlag wird **Befund der Selbstprüfung**             | `jobs/restore_probe_pg_test.go`, `jobs/backup_drill_pg_test.go`, `docs/rueckweg_anleitungen_test.go` |
| S24 | Eine Verbindung bricht mitten in einer Ergebnisliste ab                                                 | Fehler — die Liste darf **nicht** still gekürzt zurückkommen                                                        | `rows.Err()`-Konvention + `golangci-lint`; **kein** eigenes Gate                                     |

### Q2 Datenschutz

| ID  | Auslöser / Situation                                                                                   | Erwartete Reaktion                                                                                     | Nachweis                                                                       |
| --- | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| S6  | Jemand ruft ohne Anmeldung `/katalog`, `/monitor`, `/uploads/…` auf                                     | Titeldaten, Cover, Verfügbarkeit — **nie** Ausleiher, Namen, Klassen                                    | `api/pii_matrix_test.go` + `api/pii_antwort_gate_pg_test.go` (Kanarienwerte je Stufe) |
| S7  | Eine Rolle ohne Fachrecht ruft eine Schreibroute direkt per HTTP auf (Menü umgangen)                     | **403** mit der Begründung des Rechte-Wächters                                                          | `api/rechte_schreibwege_pg_test.go` (alle 87 Schreibrouten), `api/routes_authz_coverage_test.go` |
| S8  | Ein Abgänger hat keine offenen Vorgänge mehr                                                             | Sperre + Karenz (Vorgabe 90 Tage) → Anonymisierung → Hard-Delete ab 30. Januar des Folgejahres, **in dieser Folge** | `jobs/cron_dsgvo_karenz_pg_test.go`, `jobs/cron_dsgvo_abgaenger_pg_test.go`, `jobs/loeschpraedikat_ratsche_test.go` |
| S9  | Der Server startet mit einem Beispiel-Geheimnis aus dem Repository                                       | **Start verweigert** (außerhalb local/development/test); nur ein ausdrückliches `ENFORCE_PROD_SECRETS=false` lässt ihn los, mit Warnung | `api/prod_geheimnisse.go` + Selbstprüfung (**eine** gemeinsame Liste), `scripts/pruefe_secrets.sh` |
| S19 | Ein Mailserver bietet kein STARTTLS an (oder ein Angreifer streicht es aus der EHLO-Antwort)              | **Versand abgebrochen** — Mahntexte mit Schülernamen gehen nicht im Klartext über das Netz               | `mailservice/versand_test.go`, `internal/smtptest`                             |
| S20 | Ein Wert aus einer Anfrage enthält einen Zeilenumbruch und landet im Log                                  | Ein einziger JSON-Eintrag mit `\n` **innerhalb** des Strings — kein zweiter, gefälschter Eintrag         | am laufenden Server nachgemessen (11.08.2026); `slog`-JSON-Handler als Standard |

### Q3 Betriebstransparenz

| ID  | Auslöser / Situation                                                                            | Erwartete Reaktion                                                                                               | Nachweis                                                     |
| --- | ----------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------ |
| S10 | Eine fertige Funktion tut still nichts, weil eine Einstellung fehlt (SMTP, Selbstanmeldung, Backup-Schlüssel) | Selbstprüfung führt sie als Befund; kritische Befunde gehen **per Mail** an die Admins — 3 Minuten nach dem Start und danach täglich | `api/betriebsbereitschaft*.go`, `frontend/e2e/betriebsbereitschaft.spec.js` |
| S11 | Nach einem Deploy: läuft der Container aus dem aktuellen Commit?                                 | `update.sh` vergleicht `GIT_COMMIT` im Image mit `git rev-parse HEAD`; „gesund" genügt **nicht** als Antwort       | `update.sh` Schritt 4b                                       |
| S21 | Ein Testlauf überspringt die DB-Integrationstests (kein `TEST_DATABASE_URL`)                      | Der Lauf **sagt es** (Skip-Bilanz), statt grün auszusehen                                                         | `scripts/git-hooks/pre-push`, CI-Schritt „Skip-Bilanz"        |

### Q4 Bedienbarkeit

| ID  | Auslöser / Situation                                                                       | Erwartete Reaktion                                                                                                         | Nachweis                                                                 |
| --- | ------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| S12 | Gescannt wird ein Ausweis mit dem **alten** Aufdruck (Prüfzeichen im Strichcode)            | Der Scan trifft — über den zweiten Versuch ohne Prüfzeichen. Ein unbekannter Scan darf nicht wie „Scanner tot" aussehen     | `internal/service/omnibox_altes_etikett_pg_test.go`, `frontend/e2e/barcode-lesbar.spec.js` |
| S13 | Das Netz fällt während des Ausleihbetriebs aus                                              | Scans laufen weiter in die lokale Warteschlange; nach Rückkehr bucht die Nachbuch-Tür die Wirklichkeit, Abweichungen bleiben als Meldung stehen | `frontend/src/lib/stores/offlineSync.test.js`, `api/nachbuchen_*_pg_test.go` |
| S14 | Eine Hauptansicht wird mit Tastatur und Screenreader bedient                                 | Keine axe-Verstöße im Anfangszustand; keine Fokusfalle; Tabellen semantisch korrekt                                        | `frontend/e2e/barrierefreiheit-axe.spec.js`, `…-dialog.spec.js`           |
| S15 | Der Thekenrechner bleibt unbedient stehen                                                    | Nach 5 min ist der geladene Leser weg, nach 15 min der Sperrbildschirm; die Sitzung selbst läuft weiter                     | `frontend/src/lib/stores/idleLock.test.js` (gestellte Uhr, am Rückbau rot gesehen), `frontend/e2e/sperrbildschirm.spec.js` |
| S22 | Ein Menüpunkt ist für eine Rolle sichtbar                                                    | Der Klick führt **irgendwohin** — nicht wortlos zurück an die Theke                                                        | `frontend/e2e/menue-fuehrt-irgendwohin.spec.js` (Helfer, Mitarbeiter, Lehrkraft) |

### Q5 Änderbarkeit

| ID  | Auslöser / Situation                                                                              | Erwartete Reaktion                                                                                     | Nachweis                                                                    |
| --- | ------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------- |
| S16 | Eine bereits behobene Bugklasse kehrt in neuem Code zurück                                         | Die zuständige Ratsche wird rot — **und** ihre bekannte Blindheit steht in der Landkarte                | [sweeps.md](../sweeps.md) samt „Landkarte der Ratschen"                     |
| S17 | Zwei Orte behaupten dasselbe (Swagger vs. Annotation, Migrationsliste vs. `schema.sql`, Compose vs. Code, Dokumentzahl vs. Rechenfunktion) | Drift wird rot                                                          | `docs/swagger_drift_test.go`, `db/migrations_drift_test.go`, `docs/compose_variablen_test.go`, `docs/ersatzwert_staffel_test.go`, `docs/stand_angaben_test.go` |
| S18 | Ein Deploy während laufender Kiosk-Verbindungen                                                    | Geordnetes Herunterfahren innerhalb von 10 s, **kein** `os.Exit(1)`; Migrationen laufen beim Start des neuen Containers | `sse/sse_test.go`, `main_test.go`                                           |
| S23 | Toter Code bleibt liegen (eine geschriebene, getestete und nie gerufene Funktion)                   | Deadcode-Gate wird rot                                                                                 | `scripts/deadcode_gate.sh` + Baseline                                       |

---

## 10.3 Leistungsanforderungen (Größenordnung)

Der Entwurf zielt auf **eine** Schule. Die Zahlen sind die Auslegungsgrundlage, nicht ein
gemessenes Limit:

| Kennzahl                            | Auslegung                                                                                |
| ----------------------------------- | ---------------------------------------------------------------------------------------- |
| Leser                               | ~1.900 (LUSD-Simulation mit 1.890 Schülern über drei Schuljahre, 40 Prüfungen)           |
| Gleichzeitige Arbeitsplätze         | bis 8 Kiosk-Stationen mit dauerhafter SSE-Verbindung                                     |
| Antwortzeit eines Scans             | „ein Scan, eine Reaktion" — die Theke soll nicht warten; Fristen der Middleware begrenzen die Bearbeitungsdauer |
| Globales Rate-Limit                 | 50 Anfragen/s/IP, Auslieferungspfade ausgenommen                                          |
| Nachbuch-Portion                    | 1–50 Einträge je Aufruf (mehr wäre eine zu lange offene Transaktion)                      |
| Cover-Sync                          | Worker-Pool 8, alle 6 Stunden und auf Anforderung                                         |
| Lastprüfung                         | `loadtest.js` / `loadtest_advanced.js` (k6), `cmd/stresstest`                              |

**Nicht Teil der Zusage:** horizontale Skalierung, Mehrmandantenfähigkeit, Hochverfügbarkeit
ohne Wartungsfenster. Ein Deploy ist ein kurzer, geordneter Abriss der Verbindungen
(10-Sekunden-Fenster), kein Zero-Downtime-Rollout.

---

## 10.4 Was diese Zusagen **nicht** abdecken

Ehrlichkeit über die Grenzen gehört zur Qualitätszusage, sonst ist sie nur Werbung:

- **Ratschen sind lexikalisch, nicht semantisch.** Das Coverage-Gate prüft, dass ein Recht
  an einer Route **dransteht** — nicht, ob es das **richtige** ist. Die PII-Matrix vergleicht
  bei Nicht-GET-Routen Text gegen Text. Zu jeder Ratsche steht ihre Blindheit in der
  Landkarte in [sweeps.md](../sweeps.md).
- **PG-Tests laufen lokal nur, wenn eine Datenbank steht.** In CI immer, im pre-push-Hook
  nur bei laufendem Stack-Postgres — sonst **still** übersprungen. Dagegen steht die
  Skip-Bilanz (S21), nicht ein Zwang.
- **Die Sperrreihenfolge (A7) ist Konvention.** Kein Gate erzwingt sie.
- **Barrierefreiheit ist im Anfangszustand gemessen.** Zustände nach mehreren
  Interaktionsschritten sind nur teilweise abgedeckt; Umfang und bekannte Lücken stehen in
  [FACHKONZEPT.md §19](../FACHKONZEPT.md).
- **Swagger deckt 63 von 206 Routen ab.** Das vollständige Verzeichnis ist
  [api_inventar.md](../api_inventar.md) — generiert, nicht gepflegt.
