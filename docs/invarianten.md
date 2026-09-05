# Invarianten-Katalog

**Zweck:** Eine einzige Quelle der Wahrheit dafür, _was im System immer wahr sein muss_ — und
auf welcher Ebene das heute durchgesetzt wird. Der Katalog ist die Grundlage für DB-Constraints,
Tests und Code-Reviews. Er wird gepflegt, nicht einmalig geschrieben.

**Methode — die entscheidende Frage je Invariante ist nicht „testen wir sie?", sondern
„auf welcher Ebene ist sie durchgesetzt?":**

| Ebene       | Bedeutung                             | Umgehbar?                                               |
| ----------- | ------------------------------------- | ------------------------------------------------------- |
| 🟢 **DB**   | CHECK / UNIQUE / FK / Enum / NOT NULL | Nein — strukturell unmöglich                            |
| 🟡 **Code** | Go-Handler/Service-Logik              | Ja, sobald ein zweiter Schreibpfad die Prüfung auslässt |
| 🔴 **Doku** | nur im Kommentar/Konzept              | Ja — reine Hoffnung                                     |

Ziel ist, kritische Invarianten von 🔴/🟡 nach 🟢 zu schieben. Stand: 2026-09-02
(Lücken-Register G1–G6 abgearbeitet; die 🟢-Invarianten sind in CI gegen echtes
Postgres abgesichert).

> **Nachgeprüft am 11.08.2026**, Zeile für Zeile gegen Code und laufende Datenbank: Alle
> 40 genannten Dateien existieren, alle genannten Migrationen ebenfalls, und die
> stichprobenartig nachgemessenen Verhaltenszusagen halten — `mahnstufe` wird tatsächlich
> nur an **einer** Stelle geschrieben (`api/mahnwesen_bulk.go`), die Bestätigung ist über
> `WHERE bestaetigt_am IS NULL` wirklich atomar, HELFER hat in der laufenden Datenbank
> exakt die zwei dokumentierten Rechte, und `GET /api/scan` ist wirklich weg. Korrigiert
> wurde eine Fundstelle (Lieferant, siehe unten) und das Rollen-Vokabular in Abschnitt 9.

> **Fundstellen werden benannt, nicht gezählt.** Die Spalte nennt Constraint-, Index-
> und Spaltennamen — keine Zeilennummern. Bis zum 06.08.2026 stand dort `schema.sql:NNN`;
> alle 21 Verweise zeigten nach dem Wachstum der Datei auf etwas anderes als gemeint
> (`check_return_date` war als `:370` notiert und stand auf `:499`). Die Invarianten selbst
> stimmten — nur der Weg dorthin war falsch, und das fällt beim Lesen nicht auf.
> `docs/invarianten_fundstellen_test.go` prüft jetzt, dass jeder genannte Name in
> `schema.sql` wirklich existiert.
>
> **Migrationsnummern sind ebenfalls keine haltbare Fundstelle** (11.08.2026). Sie
> existieren als Datei weiter, auch wenn eine spätere Migration ihr Werk zurücknimmt —
> eine Existenzprüfung bliebe also grün. Genau so ist es hier passiert: Der Lieferanten-
> Eintrag nannte `migrations/058`, doch Migration 066 hatte Spalte **und** Index längst
> ersetzt (`ist_standard` → `ist_hauptlieferant`). Die Invariante galt, ihr Beleg zeigte
> ins Leere. Deshalb gilt auch hier: **das lebende Objekt beim Namen nennen**, dann trägt
> das Gate die Prüfung. Migrationsnummern nur als Zusatz zur Herkunft.

---

## 1. Ausleihe (`ausleihen`)

| Invariante                                                                                                                                                                                                                                                     | Durchsetzung                                                                              | Fundstelle                                                                     |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| Höchstens **eine aktive** Ausleihe je Exemplar/Gerät                                                                                                                                                                                                           | 🟢 partieller Unique-Index                                                                | `uniq_ausleihen_aktiv_exemplar`, `uniq_ausleihen_aktiv_geraet`                 |
| Genau **ein Entleiher** (Schüler XOR Benutzer) oder beide NULL (anonymisiert)                                                                                                                                                                                  | 🟢 CHECK                                                                                  | `check_loan_borrower`                                                          |
| Genau **ein Objekt** (Exemplar XOR Gerät)                                                                                                                                                                                                                      | 🟢 CHECK                                                                                  | `check_loan_item`                                                              |
| Rückgabe nie vor Ausleihe                                                                                                                                                                                                                                      | 🟢 CHECK                                                                                  | `check_return_date`                                                            |
| Gesperrte/manuell blockierte Schüler leihen nicht                                                                                                                                                                                                              | 🟡 mit Override + Audit                                                                   | `internal/service/loan_checkout_validation.go:48`                              |
| Überfällig-Automatik: ≥ `MaxOverdueItems` sperrt                                                                                                                                                                                                               | 🟡                                                                                        | `internal/service/loan_checkout_validation.go:107`                             |
| Ausleih-Limit `max_ausleihen_schueler` (LMF + eigene Rückgabe ausgenommen)                                                                                                                                                                                     | 🟡 **jetzt getestet** (88,9 %)                                                            | `internal/service/loan_checkout.go:55`, `loan_checkout_test.go`                |
| Abholbereit reserviertes Exemplar geht nicht an Dritte                                                                                                                                                                                                         | 🟡 **jetzt getestet** (90,9 %)                                                            | `internal/service/loan_checkout.go:73`, `loan_checkout_test.go`                |
| Doppel-Scan desselben Exemplars → sauberer Konflikt (409 statt 500)                                                                                                                                                                                            | 🟢 Unique-Index + 🟡 `mapLoanCreateErr` **100 % getestet**                                | `internal/service/loan_checkout_cases.go:19`                                   |
| Lehrkraft (Handapparat) → Jahresfrist, nur aktive Lehrer                                                                                                                                                                                                       | 🟡 **100 % getestet**                                                                     | `internal/service/loan_checkout_validation.go:162`                             |
| **Mahnstufe steigt NUR beim PDF-Druck** (physischer Verwaltungsakt), NIE beim Mail-Versand (Massen- wie Einzelversand = „Friendly Reminder"). PDF-Lauf schreibt `mahnstufe` + liest die PDF-Daten in DERSELBEN Tx (Papier == DB)                               | 🟡 nur `mahnwesen_bulk.go` schreibt `mahnstufe`; alle Mail-Pfade bewusst nicht            | `api/mahnwesen_bulk.go`, `api/mahnwesen_bulk_mail.go`, `api/mahnwesen_mail.go` |
| **Mahnliste geht an die Klassenleitung, nie an Schüler** — je Klasse eine Mail an genau eine Adresse (kein TO/CC über mehrere Betroffene)                                                                                                                      | 🟡 `versendeKlassenMahnungen` überspringt Klassen ohne hinterlegte Adresse; getestet      | `api/mahnwesen_bulk_mail.go`, `api/mahnwesen_bulk_mail_test.go`                |
| **Massenversand-Auswahl: leer heißt niemand, nicht alle.** Fehlendes `klassen`-Feld = alle (alter Vertrag), leeres Array = 400                                                                                                                                 | 🟡 `parseBulkOverdueRequest`, Zeiger-Semantik + Test                                      | `api/mahnwesen_bulk_mail.go`                                                   |
| **`override_email` ist die einzige Abweichung** von „jede Lehrkraft nur die eigene Klasse": Ein Empfänger sieht dann mehrere Klassen (Vertretung/Sekretariat). Weiterhin eine Mail je Klasse, kein Sammel-PDF — und die Adresse steht im Klartext im Audit-Log | 🟡 Handler + `bulkOverdueAudit` (zwei Einträge: Absicht vor dem Versand, Ergebnis danach) | `api/mahnwesen_bulk_mail.go`, `frontend/src/lib/Mahnwesen.svelte`              |

**Bewertung:** Sehr robust. Die datenkritischen Invarianten sind bereits auf DB-Ebene. Die
Geschäftsregeln (Sperre/Limit/Overdue) liegen bewusst im Code (brauchen Kontext + Override) —
Risiko nur, falls je ein _zweiter_ Checkout-Pfad entsteht, der die Validierung nicht aufruft.

---

## 2. Schüler (`schueler`)

| Invariante                                                                                                                                                                                                                                                                                                                                                                                 | Durchsetzung                                                                                                                                   | Fundstelle                                                                                                    |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| Barcode eindeutig **unter aktiven** (Soft-Delete gibt ihn zum Recycling frei)                                                                                                                                                                                                                                                                                                              | 🟢 NOT NULL + partieller Unique-Index                                                                                                          | `schueler.barcode_id`, `uniq_schueler_barcode_active`                                                         |
| Name + Geburtsdatum eindeutig — **nur bei bekanntem Geburtsdatum** (sonst wären namensgleiche Schüler ohne Datum Duplikate: Zwillings-Blockade, Migration 048)                                                                                                                                                                                                                             | 🟢 partieller Unique-Index                                                                                                                     | `unique_schueler_name_gebdatum`                                                                               |
| LUSD-ID eindeutig **unter aktiven** (Soft-Delete gibt sie frei)                                                                                                                                                                                                                                                                                                                            | 🟢 partieller Unique-Index                                                                                                                     | `uniq_schueler_lusd_id_active`                                                                                |
| `abgaenger_jahr` immer gesetzt                                                                                                                                                                                                                                                                                                                                                             | 🟢 NOT NULL                                                                                                                                    | `schueler.abgaenger_jahr`                                                                                     |
| **Gesperrt ⇒ Sperrgrund vorhanden** (kein grundloser „Zombie-Sperre"-Zustand; Personal sieht immer das _warum_). Automatische Sperr-Pfade setzen den Grund mit; `is_manually_blocked` läuft separat                                                                                                                                                                                        | 🟢 `chk_schueler_block_reason`                                                                                                                 | `schema.sql`, Migration 047, `lusd_apply.go`, `student_promotion.go`                                          |
| **[G1] Adress-/Kontaktdaten** aus LUSD importiert, Zweck Rechnung/Mahnung, bei Anonymisierung gelöscht                                                                                                                                                                                                                                                                                     | 🟢 **entschieden (B)** — Import + Löschung umgesetzt                                                                                           | `lusd_apply.go`, `schueler.strasse`/`plz`/`ort`/`eltern_email`                                                |
| **DSGVO-Retention-Kette schliesst:** Abgänger mit offenen Vorgängen → gesperrt (Name bleibt); ohne Vorgänge → für die Karenzzeit (`abgaenger_karenz_tage`, Uhr = spätester von Abgang `abgaenger_seit`, letzter Rückgabe, letztem Schadensabschluss) gesperrt, dann anonymisiert; nach Rückgabe/Bezahlung → nach Stichjahr endgültig gelöscht. Import, Job und Wächter teilen das Prädikat | 🟡 `abgaenger_jahr` aufs Abgangsjahr + `PredikatAnonymisierung(karenz, kulanz)` + Cronjob `PurgeAbgaenger` (echte Löschung, nicht Soft-Delete) | `lusd_apply.go`, `jobs/cron_dsgvo.go`, `repository/loeschfristen.go`, `repository/audit_users.go`             |
| **Leer heißt löschen — aber nur, wo Löschen erlaubt ist.** Optionale Stammdaten (Anschrift, Eltern-Mail) werden mit leerem Wert auf NULL gesetzt; Pflichtfelder (Vor-/Nachname, Klasse, Ausweisnummer) lehnen den leeren Wert mit 400 ab. Ein **fehlendes** Feld heißt weiterhin „unverändert“                                                                                             | 🟢 `addStrLeerbar` + Pflichtfeld-Wächter, PG-Test über den laufenden Handler                                                                   | `api/student_update.go`, `api/schueler_feld_leeren_pg_test.go`, `frontend/src/lib/useStudentEditForm.test.js` |
| Manuelles Löschen: Soft-Delete (Papierkorb, `DeleteStudent`) vs. endgültig (`PurgeStudent`, Recht `manage_students_admin`) sind getrennt                                                                                                                                                                                                                                                   | 🟡 Restore hebt Lösch-Sperre auf; Purge blockiert bei offenen Vorgängen                                                                        | `api/student_deleted.go`, `repository/audit_users.go`                                                         |

**Hinweis (Retention):** Der frühere Zustand hatte eine tote Kette — `sperreAbgaenger`
setzte `abgaenger_jahr` nicht (blieb auf Anlege-Default Jahr+5), also erfasste der
Cronjob den Abgänger nie; und der Job rief `DeleteStudent` (Soft-Delete), sodass die PII
selbst im Erfolgsfall nur im Papierkorb lag. Beides behoben (Tests:
`TestAbgaengerRetentionKette`, `TestRunGDPRDeleteAbgaenger_Deleted`).

**Hinweis (Löschen von Feldern):** Bis zum 23.08.2026 kam eine geleerte Anschrift gar
nicht erst an — das Formular schickte JSON-null, was im `*string` als nil ankommt und
„nicht mitgeschickt“ bedeutet. Die Oberfläche meldete trotzdem „gespeichert“. Die
Gegenprobe zeigte die andere Hälfte: Ein leerer String räumte damals Vorname, Nachname,
Ausweisnummer und Klasse mit 200 weg. Beide Richtungen sind jetzt geregelt und gegatet;
`geburtsdatum` bleibt bewusst unlöschbar (Schlüssel des LUSD-Abgleichs).

---

## 3. Buch: Titel (`buecher_titel`) & Exemplar (`buecher_exemplare`)

| Invariante                                                                                                                                              | Durchsetzung                                     | Fundstelle                            |
| ------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------ | ------------------------------------- |
| ISBN eindeutig (wo gesetzt)                                                                                                                             | 🟢 UNIQUE                                        | `buecher_titel.isbn`                  |
| Exemplar-Barcode eindeutig                                                                                                                              | 🟢 UNIQUE NOT NULL                               | `buecher_exemplare.barcode_id`        |
| Exemplar hängt an existierendem Titel                                                                                                                   | 🟢 FK ON DELETE CASCADE                          | `buecher_exemplare.titel_id`          |
| **[G4]** `grade_level` 0–13, `stock` ≥ 0                                                                                                                | 🟢 `chk_grade_level_bereich`, `chk_stock_nonneg` | `migrations/039`, `migrations/040`    |
| **[G3]** Aussonderungs-Grund strukturiert: im Umlauf = NULL, ausgesondert = genau ein Wert aus {VERLUST, BESCHAEDIGUNG, AUSSORTIERT, BESTANDSKORREKTUR} | 🟢 `chk_aussonderung_grund`                      | `migrations/043`                      |
| **[G2]** `cover_status` ∈ {PENDING, FOUND, FAILED, NOT_FOUND}                                                                                           | 🟢 `chk_cover_status`                            | `migrations/041`                      |
| `medientyp` — **bewusst ohne CHECK**: offenes, per Formular frei eingebbares Vokabular                                                                  | 🟡 Formular                                      | `migrations/040` (Begründung im Kopf) |

---

## 4. Schaden & Gebühr (`schadensfaelle`)

| Invariante                                                        | Durchsetzung                     | Fundstelle                                          |
| ----------------------------------------------------------------- | -------------------------------- | --------------------------------------------------- |
| Betrag ≥ 0                                                        | 🟢 CHECK                         | `check_positive_amount`                             |
| Genau ein Verantwortlicher (Schüler XOR Benutzer) oder beide NULL | 🟢 CHECK                         | `check_damage_responsible`                          |
| Genau ein betroffenes Objekt                                      | 🟢 CHECK                         | `check_damage_item`                                 |
| Stornierung revisionssicher (wer/wann/warum)                      | 🟡 Spalten `storniert_*` + Audit | `repository/audit_system.go` (`StornierungGebuehr`) |

---

## 5. Vormerkung (`vormerkungen`)

| Invariante                                         | Durchsetzung               | Fundstelle                                |
| -------------------------------------------------- | -------------------------- | ----------------------------------------- |
| Ein Schüler merkt einen Titel höchstens einmal vor | 🟢 UNIQUE                  | `vormerkungen (titel_id, schueler_id)`    |
| **[G2]** Status ∈ {`wartend`, `abholbereit`}       | 🟢 `chk_vormerkung_status` | `migrations/040`                          |
| Bereitgestelltes Exemplar existiert                | 🟢 FK ON DELETE SET NULL   | `vormerkungen.bereitgestellt_exemplar_id` |

**Hinweis:** Das Vokabular ist bewusst zweiwertig — erfüllte Vormerkungen werden
**gelöscht**, nicht auf einen Endstatus gesetzt (geprüft vor Migration 040).

---

## 6. Klassensatz-Reservierung (`klassensatz_reservierungen`)

| Invariante                   | Durchsetzung                | Fundstelle                            |
| ---------------------------- | --------------------------- | ------------------------------------- |
| Hängt an existierendem Titel | 🟢 FK CASCADE               | `klassensatz_reservierungen.titel_id` |
| Lebenszyklus offen/erledigt  | 🟡 Boolean                  | `klassensatz_reservierungen.erledigt` |
| **[G4]** `anzahl ≥ 1`        | 🟢 `chk_ksr_anzahl_positiv` | `migrations/039`                      |

---

## 7. Bestellung (`bestellungen_verlauf` / `_positionen`)

| Invariante                                                                                                                     | Durchsetzung                                               | Fundstelle                                                                      |
| ------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------- | ------------------------------------------------------------------------------- |
| Position hängt an existierender Bestellung                                                                                     | 🟢 FK CASCADE                                              | `bestellungen_positionen.bestellung_id`                                         |
| Nur Positionen mit Menge > 0 werden bestellt                                                                                   | 🟡 Go-Guard                                                | `api/order_service.go` (`verarbeiteBestellItem`)                                |
| **[G4]** `menge ≥ 1`, `einzelpreis ≥ 0`, `gesamtbetrag ≥ 0`, `anzahl_exemplare ≥ 0`                                            | 🟢 4 CHECKs                                                | `migrations/039`                                                                |
| Bestellbedarf meint **Lernmittel**: Freihandbestand sind bewusste Einzelstücke (Prüf-/Leseexemplare) und wird nie „aufgefüllt" | 🟡 Default `?type=lmf` + Test                              | `reorders.go`, `reorders_test.go`                                               |
| Bestätigung einer Bestellung gilt genau einmal                                                                                 | 🟢 `WHERE bestaetigt_am IS NULL` (409 statt Überschreiben) | `api/bestellbestaetigung_handler.go` (`bestaetigeBestellung`)                   |
| Höchstens ein **Haupt**lieferant                                                                                               | 🟢 Partieller Unique-Index                                 | `idx_lieferanten_ein_hauptlieferant`                                            |
| Ein Bestätigungs-Token gehört zu genau einer Bestellung, gespeichert nur als Hash                                              | 🟢 Partieller Unique-Index auf `bestaetigungs_token_hash`  | `migrations/063`                                                                |
| Die Etikettenseite des Links zeigt nur Exemplare DIESER Bestellung mit Vorab-Barcode                                           | 🟡 SQL-Filter + PG-Test                                    | `api/bestellbestaetigung_etiketten.go`, `bestellbestaetigung_ablauf_pg_test.go` |

**Hinweis zum Bestellbedarf (aktualisiert 05.08.2026):** Ohne die LMF-Vorauswahl bestand
die Liste zu ~99% aus Titeln, die niemand nachbestellen will (gemessen: 12.079 von 12.707
Titeln), weil alle Titel den Default-Meldebestand 5 tragen, der Median aber bei 1 Exemplar
liegt. **Entschieden:** Die frühere offene Frage („wird `meldebestand` je Titel gepflegt?")
ist gegenstandslos — der Auslöser ist jetzt die Einstellung `bestellbedarf_schwelle`
(Vorgabe 3, in der Oberfläche änderbar, abschaltbar über
`bestellbedarf_warnung_aktiv`). `meldebestand` wird nur noch informativ mitgeliefert und
löst nichts mehr aus (`api/reorders.go`).

---

## 8. Gerät (`geraete`) & Inventur

| Invariante                                                                                                                                                                                                                         | Durchsetzung                                                                      | Fundstelle                                                                    |
| ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- | ----------------------------------------------------------------------------- |
| Gesperrtes/ausgesondertes Gerät leiht nicht                                                                                                                                                                                        | 🟡                                                                                | `internal/service/device_service.go` (`ladeGeraet`)                           |
| **Ein fehlendes `ist_ausleihbar` heißt „unverändert“, nicht „ausleihbar“.** Der Bearbeiten-Dialog kennt die Defekt-Markierung nicht (eigener Knopf); ohne diese Regel gab jede Stammdaten-Korrektur ein defektes Gerät wieder frei | 🟢 Zeiger + `COALESCE` im UPDATE, PG-Test in beide Richtungen                     | `api/geraete.go`, `repository/geraete.go`, `api/geraet_bearbeiten_pg_test.go` |
| Inventur-Fortschritt ist **session-gebunden** (nicht global): parallele Inventuren überschreiben sich nicht                                                                                                                        | 🟢 `inventur_sessions` + `inventur_erfassungen`, partieller Unique-Index je Scope | `migrations/045`, `repository/inventur_session_repo_test.go`                  |
| Ein aktuell **verliehenes** Buch gilt bei der Inventur nie als Verlust                                                                                                                                                             | 🟡 Scope-Bedingung (`NOT EXISTS` aktive Ausleihe)                                 | `repository/inventur_session_finish.go`                                       |

---

## 9. Auth & Rollen

| Invariante                                                                                   | Durchsetzung                                          | Fundstelle                                      |
| -------------------------------------------------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------- |
| `benutzer.rolle` ∈ Enum — heute `admin`, `kollegium`, `mitarbeiter`, `helfer` (kleingeschr.) | 🟢 ENUM                                               | `benutzer_rolle`                                |
| **[G5]** **Genau eine** Quelle für die Rolle eines Benutzers: `benutzer.rolle`               | 🟢 Legacy-Tabelle entfernt + Test verhindert Rückkehr | `migrations/044`, `rollen_vokabular_pg_test.go` |
| Welche Rechte eine Rolle hat: `role_permissions` (GROSS; Middleware mappt per `UPPER()`)     | 🟡 konfigurierbar (bewusst)                           | `permission_middleware.go:83`                   |

> **Zum Rollen-Vokabular (11.08.2026):** `helfer` kam mit Migration 042 dazu, `kollegium`
> hieß bis Migration 069 `lehrer`. Der alte Name war doppelt belegt und bezeichnet seither
> nur noch den **Entleihertyp** (`schueler.klasse = 'lehrer'`, Handapparat) — wer nach ihm
> greppt, findet also weiterhin richtige Treffer. Migration 070 hat den Rechteumfang der
> Rolle auf `create_reservations` zurückgenommen; das ist aber eine **Vorgabe**, keine
> Invariante: Die Zeile darüber gilt, ein Administrator darf mehr erteilen. Siehe
> [FACHKONZEPT §12](FACHKONZEPT.md).
> | Login-Rate-Limit je echter Client-IP (nicht Proxy) | 🟢/🟡 `pkg/clientip` + `TRUSTED_PROXIES` | `middleware_ratelimit.go` |

---

## 10. Migrationen & Prozess

| Invariante                            | Durchsetzung                                         | Fundstelle                    |
| ------------------------------------- | ---------------------------------------------------- | ----------------------------- |
| Seed-Liste == alle `migrations/*.sql` | 🟢 CI-Drift-Guard (Test schlägt bei Abweichung fehl) | `db/migrations_drift_test.go` |
| Jede Migration atomar (eigene TX)     | 🟢 Runner                                            | `db/migrations.go:146`        |

---

## Lücken-Register (priorisiert)

| #           | Lücke                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           | Schwere     | Soll-Durchsetzung                                                    | Blockiert durch |
| ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------- | -------------------------------------------------------------------- | --------------- |
| ~~**G1**~~  | **ERLEDIGT (Entscheidung B, 2026-07-15):** LUSD importiert jetzt Anschrift + `eltern_email` (optional). Zweck Rechnung/Mahnung; Anonymisierung bei Abgang löscht die Daten. Kommentar korrigiert. ~~**Offen:** Rechtsgrundlage/Aufbewahrung im Verarbeitungsverzeichnis dokumentieren (Betreiber).~~ **ERLEDIGT: In `docs/SECURITY.md` dokumentiert.**                                                                                                                                                                                                                                                                          | erledigt    | `lusd_apply.go`, `lusd_parser.go`, `schueler.strasse`/`eltern_email` | —               |
| ~~**G4a**~~ | ~~`stock`, `meldebestand`, `einkaufspreis`, `menge`, `einzelpreis`, `gesamtbetrag`, `anzahl` ohne DB-Wertebereich~~ **ERLEDIGT (Migration 039):** Non-Negativitäts-/Positivitäts-CHECKs, in CI gegen echtes PG geprüft.                                                                                                                                                                                                                                                                                                                                                                                                         | 🟢 erledigt | `migrations/039_wertebereich_constraints.sql`                        | —               |
| ~~**G2**~~  | **ERLEDIGT:** `vormerkungen.status` (Migration 040), `cover_status` (Migration 041 — die vermutete inkonsistente Schreibung war ein Grep-Artefakt aus JSON-Responses; Vokabular ist durchgängig GROSS). **Dauerhaft ohne CHECK (Beschluss):** `medientyp` — offenes, frei eingebbares Vokabular. _Hinweis:_ `inventur_status` (früher `chk_inventur_status`) entfiel mit Migration 045 — der Inventur-Zustand ist jetzt session-gebunden statt eine globale Spalte.                                                                                                                                                             | 🟢 erledigt | `migrations/040`, `migrations/041`, `migrations/045`                 | —               |
| ~~**G4b**~~ | **ERLEDIGT:** `grade_level` = 0–13 (0 = unkategorisiert, 5–13 kooperative Gesamtschule inkl. Oberstufe), NULL erlaubt. Deckt sich mit App-Validierung. **Nebenbefund gefixt:** `parseKlassenStufe` klemmte fälschlich bei 10 → Jahrgang 11–13 wurde beim Import als 5 einsortiert; jetzt 5–13.                                                                                                                                                                                                                                                                                                                                  | 🟢 erledigt | `migrations/040`, `import_verarbeitung_zeilen.go`                    | —               |
| ~~**G3**~~  | **ERLEDIGT (Migration 043):** `aussonderung_grund` {VERLUST, BESCHAEDIGUNG, AUSSORTIERT, BESTANDSKORREKTUR} + `chk_aussonderung_grund` (im Umlauf = NULL, ausgesondert = genau ein Wert). Backfill aus `zustand_notiz`-Markern, alle **8** Schreibpfade angepasst, gegen echtes PG + e2e verifiziert. _(Audit 16.07.: der 8. Pfad — `UpdateCopyStatus`, einziger mit parametrisiertem `ist_ausgesondert` — war zunächst übersehen; Lehre: nach Grep-Filtern mit `WHERE`-Ausschluss immer ungefiltert gegenprüfen.)_ Bewusst kein Status „Ausgeliehen" — Ausleihzustand lebt allein in `ausleihen` (Unique-Index Migration 033). | 🟢 erledigt | `migrations/043_aussonderung_grund.sql`                              | —               |
| ~~**G5**~~  | **ERLEDIGT:** Handapparat-Bug behoben (Laufzeit liest `benutzer.rolle`), Rolle `helfer` erreichbar (Migration 042), Legacy-Tabelle `benutzer_rollen` entfernt (Migration 044) samt Bootstrap-Befüllung in `db/seed.go`. Ein Test verhindert ihre Rückkehr. **Wichtig war die Reihenfolge:** Migrationen laufen vor `InitPermissions` — ein verbliebenes `CREATE TABLE IF NOT EXISTS` hätte die Tabelle direkt nach dem DROP als leere Ruine neu angelegt.                                                                                                                                                                       | 🟢 erledigt | `migrations/044`, `db/seed.go`                                       | —               |
| ~~**G6**~~  | **ERLEDIGT:** Seed-Liste vervollständigt (038–043) + CI-Drift-Guard: Test vergleicht `migrations/*.sql` gegen die Seed-Liste in `schema.sql` und schlägt bei jeder Abweichung fehl.                                                                                                                                                                                                                                                                                                                                                                                                                                             | 🟢 erledigt | `db/migrations_drift_test.go`                                        | —               |

---

## Fahrplan

- ✅ **Phase 0 — Katalog vervollständigen.** _(dieses Dokument)_
- ✅ **Phase 1 — G1 entscheiden (Governance).** Entscheidung B umgesetzt (Import + Löschung bei
  Anonymisierung). ~~**Offen beim Betreiber:** Rechtsgrundlage/Aufbewahrung im Verarbeitungsverzeichnis.~~ **ERLEDIGT (in `SECURITY.md` dokumentiert).**
- ✅ **Phase 2 — Constraints nachrüsten.** Migrationen 039–043: 12 CHECKs + ENUM-Wert, jede gegen
  echtes Postgres verifiziert (damals 15/16, heute 18) (Verletzung provoziert → Fehler erwartet; gültige Werte akzeptiert).
- ✅ **Phase 3 — Prozess härten (G6).** `db/migrations_drift_test.go` läuft in CI.
- ✅ **Phase 4 — In Tests überführen.** Die 🟡-Ausleihregeln als Unit-Tests
  (`loan_checkout_test.go`); die 🟢-Invarianten als Integrationstests gegen echtes
  Postgres 18 im CI-Service-Container (Produktionsversion) (`db/constraints_*_pg_test.go`): jede Verletzung
  wird provoziert und muss am erwarteten Constraint scheitern, jeder gültige Wert muss
  durchgehen. Ohne DB überspringen sie sich — `TestDBTestsLaufenInCI` stellt sicher,
  dass das **in CI** nicht unbemerkt passiert.

## Daniels Raster — die elf Fragen, und ihre Frontend-Lesart

**Wann:** beim Formwechsel eines Schreibpfads (neuer Endpunkt, neuer Rumpf, andere
Speicher-Granularität) — nicht bei Kosmetik. Die Durchgänge samt Funden stehen in
[befunde.md](befunde.md), die Bestands-Achse (bekannte Bugklasse × ganzer Baum) in
[sweeps.md](sweeps.md). Die kanonische Liste steht hier, weil sweeps.md hierher zeigt
und die Fragen sonst nur verstreut in den Durchgangs-Protokollen stünden.

1. **Konvention statt Regel** — was hält die Zusicherung: ein Constraint oder eine Verabredung zwischen Dateien?
2. **Spezialwert** — was bedeutet leer/0/nil/fehlend auf diesem Pfad — und meint der Nachbarpfad dasselbe?
3. **Zwei Wahrheitsquellen / zwei Türen** — dieselbe Regel, Liste oder Beschriftung zweimal formuliert; Paare, die nur zufällig einig sind.
4. **Wer sieht was** — Recht an der Route (nicht nur am Menü), Objektbindung an den Angemeldeten.
5. **Stille Fehler** — meldet der Pfad Erfolg anhand der Wirkung oder anhand der Eingabe?
6. **Zeit** — Zeitzone, Reihenfolge, das Fenster zwischen zwei Schritten.
7. **Gate-Ehrlichkeit** — wurde das Gate rot gesehen? Kann es sich still überspringen?
8. **Lebenszyklus** — wie entsteht, altert und stirbt der Datensatz: Frist, Löschpfad, Papierkorb, Anonymisierung.
9. **Ausleitung** — wo verlässt eine Kopie die Anwendung (Datei, Mail, Export, Log, Fremdsystem)? Wer liest sie, wie lange lebt sie, ist sie verschlüsselt?
10. **Rückweg** — ist der Weg zurück begehbar und am Ergebnis bewiesen, nicht am Vorgang?
11. **Geteilter Zustand** — wer lädt ihn auf diesem Pfad, was gilt vor dem Laden und bei Fehlschlag, überlebt der Lader sein eigenes Ergebnis?

### Frontend-Lesart (ergänzt 31.08.2026)

Anlass: Beim Komplett-Durchgang 31.08. fielen alle drei Frontend-A-Funde unter
vorhandene Fragen — verfehlt wurden sie trotzdem, weil das Raster backend-gelesen
wurde („Schreibpfad" = Handler, „Tür" = Endpunkt). Dieselben Fragen, übersetzt:

| Frage                  | Backend-Begriff                             | Frontend-Begriff                                                                                                                                  |
| ---------------------- | ------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| 3 · zwei Türen         | zweiter Endpunkt schreibt denselben Zustand | zweiter Ort für denselben Zustand: Komponenten-State neben dem Store, localStorage neben dem Backend (Multi-PC!)                                  |
| 5 · stille Fehler      | 200 ohne Wirkung                            | Ladefehler ohne Fehlerzustand — ein catch, das alten Inhalt stehen lässt und nichts anzeigt; Erfolgsmeldung aus der Eingabe statt aus der Antwort |
| 6 · Zeit               | Zeitzone, Tx-Fenster                        | Antwort-Reihenfolge: überholt die langsame Antwort die schnelle? Sequenznummer wie im orderStore                                                  |
| 8 · Lebenszyklus       | Frist, Löschjob                             | Bediener- und Ansichtswechsel: welcher State muss beim Verlassen zurück auf Anfang — und welcher Code tut das wirklich?                           |
| 11 · geteilter Zustand | Wächter/Lader je Pfad                       | wer lädt den Store auf DIESEM Einstieg — und unmountet das Ergebnis seinen eigenen Lader?                                                         |

Die übrigen Fragen (1, 2, 4, 7, 9, 10) gelten wörtlich auch vorn — Frage 4 heißt dort:
das Recht steuert das Menü UND die Route, nie nur eines von beiden.

---

## Testebenen — wofür welche

| Ebene                          | Beantwortet                          | Beispiel                               |
| ------------------------------ | ------------------------------------ | -------------------------------------- |
| Unit (pgxmock)                 | Stimmt die Go-Logik?                 | Ausleihlimit, Vormerkkonflikt          |
| **DB-Integration (echtes PG)** | Hält die DB die Invariante wirklich? | `chk_aussonderung_grund` lehnt NULL ab |
| e2e (Playwright)               | Funktioniert der Weg durch die App?  | Inventur-Verlust bucht korrekt aus     |

Die mittlere Ebene ist nicht ersetzbar: pgxmock kennt keine Constraints, und e2e läuft nur
Happy-Paths. Genau in dieser Lücke sass der NULL-Bug in Migration 043.

## Restarbeit (Stand 2026-07-23)

**Code:** _(keine offenen Punkte aus dem Katalog)_

Die Rolle `helfer` hat inzwischen eine e2e-Spec (`helfer-kiosk.spec.js`): Rolle
vergebbar, Weiche in den Kiosk, kein Zugriff auf fremde Bereiche. Ob ein Helfer im
Kiosk **scannen** darf, schreibt sie bewusst nicht fest — siehe Punkt 4 unten.

**Betreiber (nur der Betreiber kann sie erledigen):**

1. ~~Oberstufen-Diagnose-Query~~ **ERLEDIGT (2026-07-16):** gegen die lokale DB
   ausgeführt — alle 12.707 Titel haben `grade_level = NULL`, kein genutzter
   Import-Pfad befüllt das Feld. Der Clamp-Bug war real, hatte aber nie
   Datenwirkung. Kein Repair nötig.
2. Echten LUSD-Export einmal hochladen — Log nennt die erkannten Adressspalten.
3. ~~DSGVO-Verarbeitungsverzeichnis: Rechtsgrundlage + Aufbewahrung der Adressdaten.~~ **ERLEDIGT (in `SECURITY.md` dokumentiert).**
4. Branch-Protection: Push auf `main` umgeht die PR-Pflicht per Admin-Bypass — Regel
   ernst nehmen (PR-Workflow) oder abschaffen.
5. ~~**`helfer`-Katalogzugriff**~~ **ERLEDIGT (2026-08-08 am Code und an der Datenbank
   nachgeprüft).**

   Die Kiosk-Kernfunktion ist von `view_students` auf ein eigenes `perform_actions`-Recht
   entkoppelt (`api/routes_misc.go` — `POST /api/action`, `GET /api/search`). Der frühere
   Zustand „jeder Scan → 403" ist behoben.

   Der hier als offen geführte Katalogzugriff ist **entschieden und umgesetzt**: `db/seed.go`
   seedet `HELFER` → `view_books = true` (Betreiber-Entscheidung 30.07.2026, siehe
   FACHKONZEPT §12). Der Satz „HELFER-Default `false`" stimmte zuletzt vor dieser
   Entscheidung und widersprach seither dem Fachkonzept — nachgezählt in der laufenden
   Datenbank: erteilt sind genau `perform_actions` und `view_books`, sonst nichts.

   `GET /api/scan` stand hier als Beleg und existiert seit dem 08.08.2026 nicht mehr: Der
   Endpunkt hatte im gesamten Repository keinen Aufrufer — weder Frontend noch E2E noch das
   gebaute Bundle — und wurde ausgebaut. Der Kiosk scannt über die Omnibox.
