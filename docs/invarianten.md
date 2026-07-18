# Invarianten-Katalog

**Zweck:** Eine einzige Quelle der Wahrheit dafür, *was im System immer wahr sein muss* — und
auf welcher Ebene das heute durchgesetzt wird. Der Katalog ist die Grundlage für DB-Constraints,
Tests und Code-Reviews. Er wird gepflegt, nicht einmalig geschrieben.

**Methode — die entscheidende Frage je Invariante ist nicht „testen wir sie?", sondern
„auf welcher Ebene ist sie durchgesetzt?":**

| Ebene | Bedeutung | Umgehbar? |
|---|---|---|
| 🟢 **DB** | CHECK / UNIQUE / FK / Enum / NOT NULL | Nein — strukturell unmöglich |
| 🟡 **Code** | Go-Handler/Service-Logik | Ja, sobald ein zweiter Schreibpfad die Prüfung auslässt |
| 🔴 **Doku** | nur im Kommentar/Konzept | Ja — reine Hoffnung |

Ziel ist, kritische Invarianten von 🔴/🟡 nach 🟢 zu schieben. Stand: 2026-07-15
(Lücken-Register G1–G6 abgearbeitet; die 🟢-Invarianten sind in CI gegen echtes
Postgres abgesichert).

---

## 1. Ausleihe (`ausleihen`)

| Invariante | Durchsetzung | Fundstelle |
|---|---|---|
| Höchstens **eine aktive** Ausleihe je Exemplar/Gerät | 🟢 partieller Unique-Index | `schema.sql:380` |
| Genau **ein Entleiher** (Schüler XOR Benutzer) oder beide NULL (anonymisiert) | 🟢 `check_loan_borrower` | `schema.sql:357` |
| Genau **ein Objekt** (Exemplar XOR Gerät) | 🟢 `check_loan_item` | `schema.sql:364` |
| Rückgabe nie vor Ausleihe | 🟢 `check_return_date` | `schema.sql:370` |
| Gesperrte/manuell blockierte Schüler leihen nicht | 🟡 mit Override + Audit | `loan_checkout_validation.go:33` |
| Überfällig-Automatik: ≥ `MaxOverdueItems` sperrt | 🟡 | `loan_checkout_validation.go:56` |
| Ausleih-Limit `max_ausleihen_schueler` (LMF + eigene Rückgabe ausgenommen) | 🟡 **jetzt getestet** (88,9 %) | `loan_checkout.go:55`, `loan_checkout_test.go` |
| Abholbereit reserviertes Exemplar geht nicht an Dritte | 🟡 **jetzt getestet** (90,9 %) | `loan_checkout.go:72`, `loan_checkout_test.go` |
| Doppel-Scan desselben Exemplars → sauberer Konflikt (409 statt 500) | 🟢 Unique-Index + 🟡 `mapLoanCreateErr` **100 % getestet** | `loan_checkout_cases.go:19`, `loan.go:106` |
| Lehrkraft (Handapparat) → Jahresfrist, nur aktive Lehrer | 🟡 **100 % getestet** | `loan_checkout_validation.go:105` |
| **Mahnstufe steigt NUR beim PDF-Druck** (physischer Verwaltungsakt), NIE beim Mail-Versand (Massen- wie Einzelversand = „Friendly Reminder"). PDF-Lauf schreibt `mahnstufe` + liest die PDF-Daten in DERSELBEN Tx (Papier == DB) | 🟡 nur `mahnwesen_bulk.go` schreibt `mahnstufe`; alle Mail-Pfade bewusst nicht | `api/mahnwesen_bulk.go`, `api/mahnwesen_bulk_mail.go`, `api/mahnwesen_mail.go` |

**Bewertung:** Sehr robust. Die datenkritischen Invarianten sind bereits auf DB-Ebene. Die
Geschäftsregeln (Sperre/Limit/Overdue) liegen bewusst im Code (brauchen Kontext + Override) —
Risiko nur, falls je ein *zweiter* Checkout-Pfad entsteht, der die Validierung nicht aufruft.

---

## 2. Schüler (`schueler`)

| Invariante | Durchsetzung | Fundstelle |
|---|---|---|
| Barcode eindeutig | 🟢 UNIQUE NOT NULL | `schema.sql:129` |
| Name + Geburtsdatum eindeutig | 🟢 Unique-Index (coalesce GebDat) | `schema.sql:153` |
| LUSD-ID eindeutig **unter aktiven** (Soft-Delete gibt sie frei) | 🟢 partieller Unique-Index | `schema.sql:156` |
| `abgaenger_jahr` immer gesetzt | 🟢 NOT NULL | `schema.sql:134` |
| **Gesperrt ⇒ Sperrgrund vorhanden** (kein grundloser „Zombie-Sperre"-Zustand; Personal sieht immer das *warum*). Automatische Sperr-Pfade setzen den Grund mit; `is_manually_blocked` läuft separat | 🟢 `chk_schueler_block_reason` | `schema.sql`, Migration 047, `lusd_apply.go`, `student_promotion.go` |
| **[G1] Adress-/Kontaktdaten** aus LUSD importiert, Zweck Rechnung/Mahnung, bei Anonymisierung gelöscht | 🟢 **entschieden (B)** — Import + Löschung umgesetzt | `lusd_apply.go`, `schema.sql:127` |
| **DSGVO-Retention-Kette schliesst:** Abgänger mit offenen Vorgängen → gesperrt (Name bleibt); nach Rückgabe/Bezahlung → nach Karenzzeit endgültig gelöscht | 🟡 `abgaenger_jahr` aufs Abgangsjahr + Cronjob `PurgeAbgaenger` (echte Löschung, nicht Soft-Delete) | `lusd_apply.go`, `jobs/cron.go`, `repository/audit_users.go` |
| Manuelles Löschen: Soft-Delete (Papierkorb, `DeleteStudent`) vs. endgültig (`PurgeStudent`, Recht `manage_users`) sind getrennt | 🟡 Restore hebt Lösch-Sperre auf; Purge blockiert bei offenen Vorgängen | `api/student_deleted.go`, `repository/audit_users.go` |

**Hinweis (Retention):** Der frühere Zustand hatte eine tote Kette — `sperreAbgaenger`
setzte `abgaenger_jahr` nicht (blieb auf Anlege-Default Jahr+5), also erfasste der
Cronjob den Abgänger nie; und der Job rief `DeleteStudent` (Soft-Delete), sodass die PII
selbst im Erfolgsfall nur im Papierkorb lag. Beides behoben (Tests:
`TestAbgaengerRetentionKette`, `TestRunGDPRDeleteAbgaenger_Deleted`).

---

## 3. Buch: Titel (`buecher_titel`) & Exemplar (`buecher_exemplare`)

| Invariante | Durchsetzung | Fundstelle |
|---|---|---|
| ISBN eindeutig (wo gesetzt) | 🟢 UNIQUE | `schema.sql:246` |
| Exemplar-Barcode eindeutig | 🟢 UNIQUE NOT NULL | `schema.sql:291` |
| Exemplar hängt an existierendem Titel | 🟢 FK ON DELETE CASCADE | `schema.sql:290` |
| **[G4]** `grade_level` 0–13, `stock` ≥ 0 | 🟢 `chk_grade_level_bereich`, `chk_stock_nonneg` | `migrations/039`, `migrations/040` |
| **[G3]** Aussonderungs-Grund strukturiert: im Umlauf = NULL, ausgesondert = genau ein Wert aus {VERLUST, BESCHAEDIGUNG, AUSSORTIERT, BESTANDSKORREKTUR} | 🟢 `chk_aussonderung_grund` | `migrations/043` |
| **[G2]** `cover_status` ∈ {PENDING, FOUND, FAILED, NOT_FOUND} | 🟢 `chk_cover_status` | `migrations/041` |
| `medientyp` — **bewusst ohne CHECK**: offenes, per Formular frei eingebbares Vokabular | 🟡 Formular | `migrations/040` (Begründung im Kopf) |

---

## 4. Schaden & Gebühr (`schadensfaelle`)

| Invariante | Durchsetzung | Fundstelle |
|---|---|---|
| Betrag ≥ 0 | 🟢 `check_positive_amount` | `schema.sql:397` |
| Genau ein Verantwortlicher (Schüler XOR Benutzer) oder beide NULL | 🟢 `check_damage_responsible` | `schema.sql:409` |
| Genau ein betroffenes Objekt | 🟢 `check_damage_item` | `schema.sql:416` |
| Stornierung revisionssicher (wer/wann/warum) | 🟡 Spalten `storniert_*` + Audit | `audit_system.go:10` |

---

## 5. Vormerkung (`vormerkungen`)

| Invariante | Durchsetzung | Fundstelle |
|---|---|---|
| Ein Schüler merkt einen Titel höchstens einmal vor | 🟢 `UNIQUE(titel_id, schueler_id)` | `schema.sql:502` |
| **[G2]** Status ∈ {`wartend`, `abholbereit`} | 🟢 `chk_vormerkung_status` | `migrations/040` |
| Bereitgestelltes Exemplar existiert | 🟢 FK ON DELETE SET NULL | `schema.sql:500` |

**Hinweis:** Das Vokabular ist bewusst zweiwertig — erfüllte Vormerkungen werden
**gelöscht**, nicht auf einen Endstatus gesetzt (geprüft vor Migration 040).

---

## 6. Klassensatz-Reservierung (`klassensatz_reservierungen`)

| Invariante | Durchsetzung | Fundstelle |
|---|---|---|
| Hängt an existierendem Titel | 🟢 FK CASCADE | `schema.sql:511` |
| Lebenszyklus offen/erledigt | 🟡 Boolean `erledigt` | `schema.sql:516` |
| **[G4]** `anzahl ≥ 1` | 🟢 `chk_ksr_anzahl_positiv` | `migrations/039` |

---

## 7. Bestellung (`bestellungen_verlauf` / `_positionen`)

| Invariante | Durchsetzung | Fundstelle |
|---|---|---|
| Position hängt an existierender Bestellung | 🟢 FK CASCADE | `schema.sql:481` |
| Nur Positionen mit Menge > 0 werden bestellt | 🟡 Go-Guard | `order_handler.go:78` |
| **[G4]** `menge ≥ 1`, `einzelpreis ≥ 0`, `gesamtbetrag ≥ 0`, `anzahl_exemplare ≥ 0` | 🟢 4 CHECKs | `migrations/039` |
| Bestellbedarf meint **Lernmittel**: Freihandbestand sind bewusste Einzelstücke (Prüf-/Leseexemplare) und wird nie „aufgefüllt" | 🟡 Default `?type=lmf` + Test | `reorders.go`, `reorders_test.go` |

**Hinweis zum Bestellbedarf:** Ohne die LMF-Vorauswahl bestand die Liste zu ~99% aus
Titeln, die niemand nachbestellen will (gemessen: 12.079 von 12.707 Titeln), weil alle
Titel den Default-Meldebestand 5 tragen, der Median aber bei 1 Exemplar liegt. **Offen
für den Betreiber:** ob `meldebestand` je Lernmittel-Titel gepflegt wird — der Default 5
ist eine Annahme, kein Beschluss.

---

## 8. Gerät (`geraete`) & Inventur

| Invariante | Durchsetzung | Fundstelle |
|---|---|---|
| Gesperrtes/ausgesondertes Gerät leiht nicht | 🟡 | `device_service.go:84` |
| Inventur-Fortschritt ist **session-gebunden** (nicht global): parallele Inventuren überschreiben sich nicht | 🟢 `inventur_sessions` + `inventur_erfassungen`, partieller Unique-Index je Scope | `migrations/045`, `repository/inventur_session_repo_test.go` |
| Ein aktuell **verliehenes** Buch gilt bei der Inventur nie als Verlust | 🟡 Scope-Bedingung (`NOT EXISTS` aktive Ausleihe) | `repository/inventur_session_finish.go` |

---

## 9. Auth & Rollen

| Invariante | Durchsetzung | Fundstelle |
|---|---|---|
| `benutzer.rolle` ∈ Enum (inkl. `helfer` seit Migration 042) | 🟢 `benutzer_rolle` ENUM (kleingeschr.) | `schema.sql:15`, `migrations/042` |
| **[G5]** **Genau eine** Quelle für die Rolle eines Benutzers: `benutzer.rolle` | 🟢 Legacy-Tabelle entfernt + Test verhindert Rückkehr | `migrations/044`, `rollen_vokabular_pg_test.go` |
| Welche Rechte eine Rolle hat: `role_permissions` (GROSS; Middleware mappt per `UPPER()`) | 🟡 konfigurierbar (bewusst) | `permission_middleware.go:83` |
| Login-Rate-Limit je echter Client-IP (nicht Proxy) | 🟢/🟡 `pkg/clientip` + `TRUSTED_PROXIES` | `middleware_ratelimit.go` |

---

## 10. Migrationen & Prozess

| Invariante | Durchsetzung | Fundstelle |
|---|---|---|
| Seed-Liste == alle `migrations/*.sql` | 🟢 CI-Drift-Guard (Test schlägt bei Abweichung fehl) | `db/migrations_drift_test.go` |
| Jede Migration atomar (eigene TX) | 🟢 Runner | `db/migrations.go:146` |

---

## Lücken-Register (priorisiert)

| # | Lücke | Schwere | Soll-Durchsetzung | Blockiert durch |
|---|---|---|---|---|
| ~~**G1**~~ | **ERLEDIGT (Entscheidung B, 2026-07-15):** LUSD importiert jetzt Anschrift + `eltern_email` (optional). Zweck Rechnung/Mahnung; Anonymisierung bei Abgang löscht die Daten. Kommentar korrigiert. **Offen:** Rechtsgrundlage/Aufbewahrung im Verarbeitungsverzeichnis dokumentieren (Betreiber). | erledigt | `lusd_apply.go`, `lusd_parser.go`, `schema.sql:127` | — |
| ~~**G4a**~~ | ~~`stock`, `meldebestand`, `einkaufspreis`, `menge`, `einzelpreis`, `gesamtbetrag`, `anzahl` ohne DB-Wertebereich~~ **ERLEDIGT (Migration 039):** Non-Negativitäts-/Positivitäts-CHECKs, in CI gegen echtes PG geprüft. | 🟢 erledigt | `migrations/039_wertebereich_constraints.sql` | — |
| ~~**G2**~~ | **ERLEDIGT:** `vormerkungen.status` (Migration 040), `cover_status` (Migration 041 — die vermutete inkonsistente Schreibung war ein Grep-Artefakt aus JSON-Responses; Vokabular ist durchgängig GROSS). **Dauerhaft ohne CHECK (Beschluss):** `medientyp` — offenes, frei eingebbares Vokabular. *Hinweis:* `inventur_status` (früher `chk_inventur_status`) entfiel mit Migration 045 — der Inventur-Zustand ist jetzt session-gebunden statt eine globale Spalte. | 🟢 erledigt | `migrations/040`, `migrations/041`, `migrations/045` | — |
| ~~**G4b**~~ | **ERLEDIGT:** `grade_level` = 0–13 (0 = unkategorisiert, 5–13 kooperative Gesamtschule inkl. Oberstufe), NULL erlaubt. Deckt sich mit App-Validierung. **Nebenbefund gefixt:** `parseKlassenStufe` klemmte fälschlich bei 10 → Jahrgang 11–13 wurde beim Import als 5 einsortiert; jetzt 5–13. | 🟢 erledigt | `migrations/040`, `import_verarbeitung_zeilen.go` | — |
| ~~**G3**~~ | **ERLEDIGT (Migration 043):** `aussonderung_grund` {VERLUST, BESCHAEDIGUNG, AUSSORTIERT, BESTANDSKORREKTUR} + `chk_aussonderung_grund` (im Umlauf = NULL, ausgesondert = genau ein Wert). Backfill aus `zustand_notiz`-Markern, alle **8** Schreibpfade angepasst, gegen echtes PG + e2e verifiziert. *(Audit 16.07.: der 8. Pfad — `UpdateCopyStatus`, einziger mit parametrisiertem `ist_ausgesondert` — war zunächst übersehen; Lehre: nach Grep-Filtern mit `WHERE`-Ausschluss immer ungefiltert gegenprüfen.)* Bewusst kein Status „Ausgeliehen" — Ausleihzustand lebt allein in `ausleihen` (Unique-Index Migration 033). | 🟢 erledigt | `migrations/043_aussonderung_grund.sql` | — |
| ~~**G5**~~ | **ERLEDIGT:** Handapparat-Bug behoben (Laufzeit liest `benutzer.rolle`), Rolle `helfer` erreichbar (Migration 042), Legacy-Tabelle `benutzer_rollen` entfernt (Migration 044) samt Bootstrap-Befüllung in `db/seed.go`. Ein Test verhindert ihre Rückkehr. **Wichtig war die Reihenfolge:** Migrationen laufen vor `InitPermissions` — ein verbliebenes `CREATE TABLE IF NOT EXISTS` hätte die Tabelle direkt nach dem DROP als leere Ruine neu angelegt. | 🟢 erledigt | `migrations/044`, `db/seed.go` | — |
| ~~**G6**~~ | **ERLEDIGT:** Seed-Liste vervollständigt (038–043) + CI-Drift-Guard: Test vergleicht `migrations/*.sql` gegen die Seed-Liste in `schema.sql` und schlägt bei jeder Abweichung fehl. | 🟢 erledigt | `db/migrations_drift_test.go` | — |

---

## Fahrplan

- ✅ **Phase 0 — Katalog vervollständigen.** *(dieses Dokument)*
- ✅ **Phase 1 — G1 entscheiden (Governance).** Entscheidung B umgesetzt (Import + Löschung bei
  Anonymisierung). **Offen beim Betreiber:** Rechtsgrundlage/Aufbewahrung im Verarbeitungsverzeichnis.
- ✅ **Phase 2 — Constraints nachrüsten.** Migrationen 039–043: 12 CHECKs + ENUM-Wert, jede gegen
  echtes PG 15/16 verifiziert (Verletzung provoziert → Fehler erwartet; gültige Werte akzeptiert).
- ✅ **Phase 3 — Prozess härten (G6).** `db/migrations_drift_test.go` läuft in CI.
- ✅ **Phase 4 — In Tests überführen.** Die 🟡-Ausleihregeln als Unit-Tests
  (`loan_checkout_test.go`); die 🟢-Invarianten als Integrationstests gegen echtes
  Postgres 15 im CI-Service-Container (`db/constraints_*_pg_test.go`): jede Verletzung
  wird provoziert und muss am erwarteten Constraint scheitern, jeder gültige Wert muss
  durchgehen. Ohne DB überspringen sie sich — `TestDBTestsLaufenInCI` stellt sicher,
  dass das **in CI** nicht unbemerkt passiert.

## Testebenen — wofür welche

| Ebene | Beantwortet | Beispiel |
|---|---|---|
| Unit (pgxmock) | Stimmt die Go-Logik? | Ausleihlimit, Vormerkkonflikt |
| **DB-Integration (echtes PG)** | Hält die DB die Invariante wirklich? | `chk_aussonderung_grund` lehnt NULL ab |
| e2e (Playwright) | Funktioniert der Weg durch die App? | Inventur-Verlust bucht korrekt aus |

Die mittlere Ebene ist nicht ersetzbar: pgxmock kennt keine Constraints, und e2e läuft nur
Happy-Paths. Genau in dieser Lücke sass der NULL-Bug in Migration 043.

## Restarbeit (Stand 2026-07-16)

**Code:** *(keine offenen Punkte aus dem Katalog)*

Die Rolle `helfer` hat inzwischen eine e2e-Spec (`helfer-kiosk.spec.js`): Rolle
vergebbar, Weiche in den Kiosk, kein Zugriff auf fremde Bereiche. Ob ein Helfer im
Kiosk **scannen** darf, schreibt sie bewusst nicht fest — siehe Punkt 4 unten.

**Betreiber (nur der Betreiber kann sie erledigen):**
1. ~~Oberstufen-Diagnose-Query~~ **ERLEDIGT (2026-07-16):** gegen die lokale DB
   ausgeführt — alle 12.707 Titel haben `grade_level = NULL`, kein genutzter
   Import-Pfad befüllt das Feld. Der Clamp-Bug war real, hatte aber nie
   Datenwirkung. Kein Repair nötig.
2. Echten LUSD-Export einmal hochladen — Log nennt die erkannten Adressspalten.
3. DSGVO-Verarbeitungsverzeichnis: Rechtsgrundlage + Aufbewahrung der Adressdaten.
4. Branch-Protection: Push auf `main` umgeht die PR-Pflicht per Admin-Bypass — Regel
   ernst nehmen (PR-Workflow) oder abschaffen.
5. **`helfer`-Rechte entscheiden — die Rolle ist mit den Default-Rechten funktionsunfähig.**
   Belegt: `db/seed.go` seedet für `HELFER` **alle** Rechte auf `false`. Der Router
   (`Router.svelte:37`) zwingt Helfer aber auf genau zwei Ansichten, die beide Rechte
   brauchen:

   | Ansicht | Endpunkt | benötigtes Recht | Default |
   |---|---|---|---|
   | Kiosk (Omnibox) | `POST /api/action`, `GET /api/search` | `view_students` | ❌ false |
   | Katalog | `GET /api/books` | `view_books` | ❌ false |

   Ein Helfer könnte sich also anmelden, landet im Kiosk — und **jeder Scan liefert 403**.
   Das ist bewusst nicht von mir entschieden worden: `view_students` öffnet Schülerdaten,
   und ob eine Hilfskraft (Schüler/Eltern?) die sehen darf, ist eine fachliche und
   datenschutzrechtliche Entscheidung des Betreibers — kein Implementierungsdetail.
   Ist die Antwort „ja, für den Kiosk", genügt es, die beiden Rechte im
   PermissionManager zu aktivieren (konfigurierbar, keine Code-Änderung nötig).
