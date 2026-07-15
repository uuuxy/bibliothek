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

Ziel ist, kritische Invarianten von 🔴/🟡 nach 🟢 zu schieben. Stand: 2026-07-15.

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
| Ausleih-Limit `max_ausleihen_schueler` | 🟡 (Checkout-Pfad, per Test belegt) | `action_helpers_test.go:47` |
| Doppel-Scan desselben Exemplars → sauberer Konflikt | 🟡 `ErrConflict` + `SELECT … FOR UPDATE` | `loan_checkout_cases.go:22`, `loan.go:106` |

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
| **[G1] Adress-/Kontaktdaten** aus LUSD importiert, Zweck Rechnung/Mahnung, bei Anonymisierung gelöscht | 🟢 **entschieden (B)** — Import + Löschung umgesetzt | `lusd_apply.go`, `schema.sql:127` |

---

## 3. Buch: Titel (`buecher_titel`) & Exemplar (`buecher_exemplare`)

| Invariante | Durchsetzung | Fundstelle |
|---|---|---|
| ISBN eindeutig (wo gesetzt) | 🟢 UNIQUE | `schema.sql:246` |
| Exemplar-Barcode eindeutig | 🟢 UNIQUE NOT NULL | `schema.sql:291` |
| Exemplar hängt an existierendem Titel | 🟢 FK ON DELETE CASCADE | `schema.sql:290` |
| **[G4]** `grade_level`, `stock` in gültigem Bereich | 🟡 nur in Go (Import-Parser) | `import_verarbeitung_zeilen.go` |
| **[G3]** Physischer Zustand „Verloren/…" als Zustandsautomat | 🔴 Freitext `zustand_notiz` | `schema.sql:292` |
| **[G2]** `cover_status`, `medientyp` in gültiger Menge | 🔴 kein CHECK | `schema.sql:252,260` |

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
| **[G2]** Status-Lebenszyklus `wartend → abholbereit` (3-Tage-Fenster) | 🔴 Freitext, kein CHECK | `loan_return.go:38` |
| Bereitgestelltes Exemplar existiert | 🟢 FK ON DELETE SET NULL | `schema.sql:500` |

**Hinweis:** Verwendete Werte im Code: `wartend`, `abholbereit`. Der **vollständige** erlaubte
Satz (inkl. Ablauf/Abholung aus Migration 022) muss vor einem CHECK in Phase 0 gepinnt werden.

---

## 6. Klassensatz-Reservierung (`klassensatz_reservierungen`)

| Invariante | Durchsetzung | Fundstelle |
|---|---|---|
| Hängt an existierendem Titel | 🟢 FK CASCADE | `schema.sql:511` |
| Lebenszyklus offen/erledigt | 🟡 Boolean `erledigt` | `schema.sql:516` |
| **[G4]** `anzahl > 0` | 🔴 kein CHECK (nur DEFAULT 1) | `schema.sql:513` |

---

## 7. Bestellung (`bestellungen_verlauf` / `_positionen`)

| Invariante | Durchsetzung | Fundstelle |
|---|---|---|
| Position hängt an existierender Bestellung | 🟢 FK CASCADE | `schema.sql:481` |
| Nur Positionen mit Menge > 0 werden bestellt | 🟡 Go-Guard | `order_handler.go:78` |
| **[G4]** `menge > 0`, `einzelpreis ≥ 0` auf DB-Ebene | 🔴 kein CHECK | `schema.sql:485` |

---

## 8. Gerät (`geraete`) & Inventur

| Invariante | Durchsetzung | Fundstelle |
|---|---|---|
| Gesperrtes/ausgesondertes Gerät leiht nicht | 🟡 | `device_service.go:84` |
| **[G2]** `inventur_status` ∈ {`ausstehend`,`erfasst`} | 🔴 Freitext, kein CHECK | `schema.sql:296` |

---

## 9. Auth & Rollen

| Invariante | Durchsetzung | Fundstelle |
|---|---|---|
| `benutzer.rolle` ∈ Enum | 🟢 `benutzer_rolle` ENUM (kleingeschr.) | `schema.sql:15` |
| `benutzer_rollen.rolle` ∈ Menge | 🟢 CHECK (GROSS + `HELFER`) | `schema.sql:119` |
| **[G5]** Ein einziges Rollen-Vokabular | 🔴 **zwei divergierende** Definitionen | siehe Lücken-Register |
| Login-Rate-Limit je echter Client-IP (nicht Proxy) | 🟢/🟡 `pkg/clientip` + `TRUSTED_PROXIES` | `middleware_ratelimit.go` |

---

## 10. Migrationen & Prozess

| Invariante | Durchsetzung | Fundstelle |
|---|---|---|
| Seed-Liste == alle `migrations/*.sql` | 🔴 **[G6]** `038` fehlt in der Liste | `schema.sql:534` |
| Jede Migration atomar (eigene TX) | 🟢 Runner | `db/migrations.go:146` |

---

## Lücken-Register (priorisiert)

| # | Lücke | Schwere | Soll-Durchsetzung | Blockiert durch |
|---|---|---|---|---|
| ~~**G1**~~ | **ERLEDIGT (Entscheidung B, 2026-07-15):** LUSD importiert jetzt Anschrift + `eltern_email` (optional). Zweck Rechnung/Mahnung; Anonymisierung bei Abgang löscht die Daten. Kommentar korrigiert. **Offen:** Rechtsgrundlage/Aufbewahrung im Verarbeitungsverzeichnis dokumentieren (Betreiber). | erledigt | `lusd_apply.go`, `lusd_parser.go`, `schema.sql:127` | — |
| **G4a** | ~~`stock`, `meldebestand`, `einkaufspreis`, `menge`, `einzelpreis`, `gesamtbetrag`, `anzahl` ohne DB-Wertebereich~~ **ERLEDIGT (Migration 039):** Non-Negativitäts-/Positivitäts-CHECKs, gegen echtes PG verifiziert. | 🟢 erledigt | `migrations/039_wertebereich_constraints.sql` |
| **G2** | ~~Status-Freitextfelder ohne CHECK~~ **TEILWEISE ERLEDIGT (Migration 040):** `vormerkungen.status` {wartend,abholbereit}, `inventur_status` {ausstehend,erfasst}+NULL — gegen echtes PG verifiziert. **Offen bewusst:** `cover_status` (inkonsistente Groß-/Kleinschreibung im Code → erst bereinigen), `medientyp` (offenes Vokabular, freie Formulareingabe → kein CHECK). | 🟢 Kern erledigt | `migrations/040_status_constraints.sql` |
| ~~**G4b**~~ | **ERLEDIGT:** `grade_level` = 0–13 (0 = unkategorisiert, 5–13 kooperative Gesamtschule inkl. Oberstufe), NULL erlaubt. Deckt sich mit App-Validierung. **Nebenbefund gefixt:** `parseKlassenStufe` klemmte fälschlich bei 10 → Jahrgang 11–13 wurde beim Import als 5 einsortiert; jetzt 5–13. | 🟢 erledigt | `migrations/040`, `import_verarbeitung_zeilen.go` |
| **G3** | Buch-Zustand als Freitext statt Zustandsautomat. | Mittel | Statusspalte + erlaubte Übergänge | Produkt-Entscheidung |
| **G5** | Zwei Rollen-Vokabulare (Enum kleingeschr. vs. CHECK GROSS + `HELFER`). **DEFER (Senior-Entscheidung):** Code nutzt durchgängig GROSS (`jwt.go`, `benutzer_rollen`); Konsolidierung fasst Login-/JWT-/Authz-Kette an + Datenmigration → nicht im Constraint-Durchlauf. | Mittel (nicht dringend) | Auth-Kette + Datenmigration, separater Vorgang | Bewusst zurückgestellt |
| **G6** | `038_signatur_konsolidierung.sql` fehlt in der Seed-Liste. **Kein** Fresh-Install-Breaker (reines idempotentes Daten-UPDATE), aber Prozessregel verletzt. | Niedrig | CI-Check Liste↔Dateien | — |

---

## Fahrplan

- **Phase 0 — Katalog vervollständigen.** *(dieses Dokument — erledigt für die Kern-Entitäten)*
- **Phase 1 — G1 entscheiden (Governance).** Rechtsgrundlage/Zweck der Adress-/Kontaktdaten
  festhalten, Aufbewahrung/Löschung klären, Kommentar korrigieren. **Vor** jeder `schueler`-Migration.
- **Phase 2 — Constraints nachrüsten (G4 → G2 → G5).** Je Lücke eine kleine idempotente Migration:
  erst `SELECT` auf Bestandsverletzer, dann `CHECK`/Enum/Unique. Datenkorruption vor Kosmetik.
- **Phase 3 — Prozess härten (G6).** CI-Diff `migrations/*.sql` ↔ Seed-Liste.
- **Phase 4 — In Tests überführen.** 🟡-Invarianten als table-driven Handler-Tests (echter HTTP-Pfad),
  🟢-Invarianten mit einem Test, der die Verletzung provoziert und den DB-Fehler erwartet.
