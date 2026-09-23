# 1. Einführung und Ziele

Stand: 23.09.2026

---

## 1.1 Aufgabenstellung

**Bibliothek** ist die Verwaltungssoftware der Schulbibliothek einer Gesamtschule. Sie
löst eine Windows-Altanwendung (**Littera**) ab und deckt den vollständigen Betrieb ab:
Ausleihe und Rückgabe am Scanner-Tresen, Medienkatalog, Lernmittelverwaltung (LMF),
Mahnwesen, Vormerkungen, Inventur, Bestellwesen, Geräteausleihe, Leserdatei und die
datenschutzrechtlich vorgeschriebenen Löschroutinen.

**Das ist kein Produkt.** Es ist für **einen** konkreten Betrieb gebaut; die Entscheidungen
darin sind entsprechend konkret (eine feste Lernmittel-Stichtagsregel, ein einziger
Mailserver als Anmeldequelle, Barcodes eines Altbestands, die nicht neu geklebt werden
können). Wo diese Dokumentation eine Entscheidung „falsch" erscheinen lässt, lohnt der
Blick in [Kapitel 9](09-architekturentscheidungen.md): fast immer ist die Randbedingung der
Grund, nicht der Geschmack.

### Die tragenden fachlichen Anforderungen

| #  | Anforderung                                                                                                                                  | Fundstelle                                       |
| -- | -------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------ |
| F1 | **Ein Eingabefeld für alle Scans** (Omnibox). Ohne Präfix wird in der Reihenfolge Buch → Ausweis → Volltextsuche aufgelöst.                   | `internal/service/omnibox_service.go`, FACHKONZEPT §1 |
| F2 | **Fristen** je Medienart: Lernmittel auf den Stichtag 31. Juli bzw. den Klassentermin des LMF-Plans, Freihand rollierend, Lehrkraft ein Jahr. | `internal/service/loan_rules.go`, `pkg/lmf`, `pkg/lmfplan` |
| F3 | **Bis zu 8 Kiosk-Stationen gleichzeitig** am selben Bestand, ohne Doppelbuchung und ohne Phantom-Erfolg.                                      | `migrations/033_unique_active_loan.sql`, `sse/`  |
| F4 | **Mahnwesen**: Mahnstufe steigt ausschließlich beim PDF-Druck (dem physischen Verwaltungsakt), nie beim Mailversand; Listen gehen an die Klassenleitung, nie an Schüler. | `api/mahnwesen_bulk.go`                          |
| F5 | **Schülerdaten unter DSGVO**: Löschfristen, Karenz, Anonymisierung, verschlüsselte Fotos, Auskunftsrecht, Audit-Trail.                        | `jobs/cron_dsgvo*.go`, `internal/crypto`, [SECURITY.md](../SECURITY.md) |
| F6 | **Öffentliche Seiten ohne Anmeldung** (Katalog `/katalog`, Monitor `/monitor`) — Titeldaten ja, Personendaten nie.                            | `api/opac.go`, `api/monitor.go`, FACHKONZEPT §16 |
| F7 | **Kollegiums-Portal** mit Selbstanmeldung über das Schulpostfach, Klassensatz-Reservierung, Wünsche und Meldungen.                            | `auth/selbstanmeldung.go`, `frontend/src/lib/KollegiumPortal.svelte` |
| F8 | **Altbestandsübernahme aus Littera**: Titel, Exemplare, Personen, offene Ausleihen — verlustfrei und nachweisbar.                             | `internal/littera`, `internal/uebernahme`, `cmd/littera-altbestand` |
| F9 | **Bestellwesen** bis zum Wareneingang, inklusive Bestätigungslink für Händler, die selbst etikettieren.                                       | `api/bestellbestaetigung_*.go`                   |
| F10| **Der Betrieb muss merken, wenn eine Funktion still nichts tut** (fehlende Einstellung, fehlendes Geheimnis, fehlgeschlagene Restore-Probe).  | `api/betriebsbereitschaft.go`, FACHKONZEPT §15   |

---

## 1.2 Qualitätsziele

Die fünf Ziele stehen in der Reihenfolge, in der bei einem Konflikt entschieden wird. Das
ist keine Rhetorik: Q1 vor Q5 heißt konkret, dass eine Buchung lieber mit **409 Conflict**
scheitert als „irgendwie" durchgeht, und Q2 vor Q4, dass eine Löschroutine auch dann läuft,
wenn dadurch eine Statistik ihre Zahlenbasis verliert.

| Prio | Qualitätsziel                          | Was damit konkret gemeint ist                                                                                                                                     | Wie es nachgewiesen wird                                                                                     |
| ---- | -------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| Q1   | **Korrektheit unter Nebenläufigkeit**  | Acht Stationen scannen gleichzeitig. Kein Exemplar hat zwei aktive Ausleihen, kein Doppelscan erzeugt eine zweite Buchung, kein Fehler wird als Erfolg quittiert. | Partielle Unique-Indizes (DB-Ebene), Idempotenz-Keys, `*_pg_test.go` gegen echtes Postgres, `phantom_erfolg_test.go` |
| Q2   | **Datenschutz-Konformität**            | Schülerdaten werden fristgerecht getilgt, Fotos verschlüsselt, öffentliche Seiten tragen keine Personendaten, jede Route ist nach PII-Stufe eingeordnet.          | `api/pii_matrix_test.go` hält [PII_MATRIX.de.md](../PII_MATRIX.de.md) deckungsgleich mit dem Code; DSGVO-Cronjobs mit PG-Tests |
| Q3   | **Betriebstransparenz**                | Eine eingerichtete, aber nicht in Betrieb befindliche Funktion muss sich melden — nicht schweigen.                                                                 | Selbstprüfung der Betriebsbereitschaft + Bereitschafts-Wächter (täglich per Mail), `/health`                 |
| Q4   | **Bedienbarkeit am Tresen**            | Ein Scan, eine Reaktion, ohne Maus. Der Ausfall des Netzes darf den Betrieb nicht anhalten.                                                                        | Offline-Warteschlange (IndexedDB) + Nachbuch-Tür, Tastaturbedienung, WCAG 2.1 AA mit axe-Gate                |
| Q5   | **Änderbarkeit ohne Rückfall**         | Ein behobener Fehler bleibt behoben. Jede Bugklasse bekommt einen Detektor (eine „Ratsche"), nicht nur einen Fix.                                                  | [sweeps.md](../sweeps.md), Drift-Gates (Swagger, Migrationen, Compose-Variablen), Deadcode-Gate              |

> **Die Gate-Regel des Projekts:** Der Anspruch ist nicht „es gibt Tests", sondern *jedes
> Gate muss man einmal rot gesehen haben*. Ein Gate, das seine Aussage nicht verlieren kann,
> prüft nichts. Deshalb führt `scripts/git-hooks/pre-push` auch eine **Skip-Bilanz**: Es
> sagt, was es *nicht* geprüft hat (die `*_pg_test.go` überspringen sich ohne
> `TEST_DATABASE_URL` still, mit grünem „ok" daneben).

### Was ausdrücklich **kein** Ziel ist

- **Mandantenfähigkeit.** Eine Schule, eine Datenbank, ein Stack. Es gibt keine
  Schul-ID und keinen Mandanten-Schlüssel.
- **Konfigurierbarkeit als Produktmerkmal.** Was das Kollegium darf, steht fest in
  `db/seed.go` und ist kein Schalter je Schule (Entscheidung vom 16.09.2026).
- **Skalierung über einen Host hinaus.** Der Entwurf zielt auf ~1.900 Leser und acht
  gleichzeitige Arbeitsplätze, nicht auf horizontale Skalierung. Der Prozess hält
  In-Memory-Zustand (SSE-Broker, Rechte-Cache, Rate-Limit-Zähler); mehrere Instanzen
  hinter einem Load Balancer wären ein Umbau, nicht eine Konfiguration
  ([Kapitel 11](11-risiken-und-technische-schulden.md)).
- **Mehrsprachigkeit.** Die Oberfläche, die Fachsprache und der Code sind deutsch.

---

## 1.3 Stakeholder

| Rolle                                | Erwartung an das System                                                                                                            | Was daraus architektonisch folgt                                                                                     |
| ------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| **Bibliotheksleitung**               | Überblick, Mahnwesen, Bestellung, Statistik ohne Klarnamen; Systempflege nicht übernehmen müssen                                   | Eigene Rolle `leitung` = Admin **minus** `manage_users`/`manage_settings`, abgeleitet statt abgeschrieben (Migration 122) |
| **Bibliotheks-Mitarbeiter**          | Tagesgeschäft an der Theke, schnell, ohne Systemwissen                                                                             | Omnibox als einziges Eingabefeld; Fehler sprechen deutsch und sagen die nächste Handlung                             |
| **Helfer** (Eltern, Hilfskräfte)     | Ausleihe/Rückgabe und die Frage „habt ihr Band 3 noch da?" beantworten — ohne Zugriff auf Personendaten                            | Genau zwei Rechte (`perform_actions`, `view_books`), Grenze zu Personendaten über `view_students`                     |
| **Kollegium** (~160 Lehrkräfte)      | Selbst anmelden können, Klassensätze reservieren, LMF-Termine sehen, Wünsche melden                                                 | Selbstanmeldung über die Schuldomain mit **Freischaltung durch die Bibliothek**; Portal hängt am Recht `create_reservations`, nicht an der Rolle |
| **Schüler und Eltern**               | Sehen, ob ein Buch da ist — ohne Konto                                                                                             | Öffentlicher Katalog und Monitor, die ausschließlich Titeldaten liefern (PII-Stufe 0)                                |
| **Sekretariat / Schulleitung**        | Schülerdaten müssen ohne Doppelerfassung hereinkommen; Versetzung und Abgang müssen funktionieren                                   | LUSD-Import mit drei Zuordnungsstufen, Umbenennungs-Paarung und Karenzzeit ([LUSD.md](../LUSD.md))                   |
| **Datenschutzbeauftragte(r)**        | Nachweisbare Fristen, Rechtsgrundlagen, Verarbeitungsverzeichnis, Auskunft                                                          | PII-Matrix je Route als Gate, automatische Löschroutinen, VVT- und Art.-13-Entwürfe unter `docs/datenschutz/`         |
| **Schulträger**                      | Nachweispflichten (Zugangs-/Abgangsbuch, Bestandsnachweis zum Stichtag), getrennte Töpfe Land/Träger                                | Bestandsbücher als eigene Reiter mit Druckblatt; Mittelherkunft an der Bestellung ([mittel_konzept.md](../mittel_konzept.md)) |
| **Betreiber/Entwickler** (eine Person) | Ein Deploy darf nichts still zerstören; ein Fehler muss sich selbst melden; die Doku muss die Entscheidung tragen, nicht nur das Ergebnis | Gates, Drift-Tests, Selbstprüfung, Bereitschafts-Wächter, `OFFEN.md` als einzige Liste, Commit-Historie als Doku      |
| **Land (Lernmittelfreiheit)**         | Leihbücher nach Erlass behandeln: Fristen, Ersatzwert-Staffel, Bescheid statt Barzahlung                                            | `pkg/ersatzwert` (Staffel als Vorschlag mit Herleitung), `api/bescheid_*.go`, Gate `docs/ersatzwert_staffel_test.go`  |

---

## 1.4 Umfang in Zahlen (gemessen 23.09.2026)

| Gegenstand                        | Umfang                                    |
| --------------------------------- | ----------------------------------------- |
| Go-Produktivcode                  | 67.833 Zeilen (ohne das generierte `docs/docs.go`) |
| Go-Tests                          | 87.933 Zeilen in 627 Testdateien           |
| Svelte/JavaScript (`frontend/src`) | 64.894 Zeilen, davon 293 `.svelte`-Dateien |
| e2e-Spezifikationen (Playwright)  | 109 Dateien                                |
| Registrierte HTTP-Routen          | 216 (davon 78 Operationen Swagger-annotiert) |
| Datenbank-Migrationen             | 147                                        |
| Tabellen / Sichten in `schema.sql`| 44 Tabellen, 2 Sichten (`schueler`, `view_buecher_bestand`) |

Die Befehle, mit denen diese Zahlen in zehn Sekunden neu erhoben werden, stehen in
[Kapitel 5](05-bausteinsicht.md#anhang-die-zahlen-selbst-nachmessen). Das ist Absicht: Eine
gepflegte Behauptung altert, ein Messbefehl nicht. Die vorige Fassung im README lag bei den
Testzeilen um 47 % daneben, weil sie gepflegt statt gemessen war.
