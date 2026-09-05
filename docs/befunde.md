# Befund-Register

**Zweck:** Ein Ort für Dinge, die beim Arbeiten auffallen, aber nicht sofort angefasst
werden. Ohne diese Liste gibt es nur zwei Umgangsformen mit einem Fund — sofort
reparieren oder vergessen — und beide sind falsch.

Ergänzt [`invarianten.md`](invarianten.md): Dort steht, was immer wahr sein _muss_.
Hier steht, was aufgefallen ist und noch nicht entschieden oder erledigt wurde.

**Erledigtes wird gelöscht, nicht abgehakt** — es steht vollständig in
`git log -p docs/befunde.md`. Stände vor früheren Kürzungen: `2e09ec14` (31.08.),
`47a09f70` (01.09.), `9f91784b` (04.09.), `e2bd4b72` (05.09.).

---

## Die Einordnung

Vor jedem Fund steht dieselbe Frage — **nicht** „ist das hässlich?", sondern
**„kann das still jemandem schaden?"**:

|       | Kategorie                                                                                                                                                                                            | Umgang                                                               |
| ----- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| **A** | Kann **stillschweigend** ein falsches Ergebnis für einen echten Menschen erzeugen: doppelte Mahnung an Eltern, Daten beim falschen Empfänger, verlorene Eingabe, ein Gate, das nicht rot werden kann | **Sofort**, eigener Commit, eigener Test, eigene Deploy-Entscheidung |
| **B** | Fehler, der sich **laut** meldet, oder Unordnung ohne Wirkung nach außen: totes Codestück, wackeliger Test, Doppelung                                                                                | **Hier notieren**, gebündelt abarbeiten                              |
| **C** | „Wenn ich schon mal hier bin" — Umbenennungen, Stilfragen, Refactorings ohne Anlass                                                                                                                  | Nur mit Anlass und Zeit                                              |

Zwei Regeln dazu:

1. **Ein Fund = ein Commit.** Kein Anhängen verwandter Aufräumarbeiten. Was beim
   Reparieren zusätzlich auffällt, kommt in diese Liste, nicht in denselben Commit.
2. **Kategorie A wird belegt, nicht behauptet.** Ein Fund dieser Klasse braucht einen
   Test, der mit dem alten Code rot wird. Ohne diese Gegenprobe ist unklar, ob
   überhaupt etwas repariert wurde.

**Reihenfolge:** erst die Rückkehr zur ursprünglichen Bedeutung von „Abgänger" (Entscheidung 2),
dann der Wächter für Dauer-Abgänger, dann die Overlays auf `Modal.svelte`, alles Weitere beim nächsten fachlichen Anfassen.

---

## Offen — abarbeitbar

- **Zwei Definitionen von „derselbe Mensch"** (05.09.2026, B). Der Unique-Index
  `unique_schueler_name_gebdatum` vergleicht Vor- und Nachname roh (case-sensitiv, keine
  Normalform); der LUSD-Schlüssel rechnet seit 3848c9f6 in der Normalform `suchnorm`. Folge:
  Die Datenbank lässt „Anna Müller" und „Anna Mueller" mit gleichem Geburtsdatum als zwei
  Zeilen zu (Handanlage), der Import sieht darin einen mehrdeutigen Schlüssel und fasst beide
  nicht an. Kein stiller Schaden — die Mehrdeutig-Meldung fängt es —, aber der Index
  verspricht weniger, als der Import annimmt. Option: Index auf `suchnorm(vorname),
suchnorm(nachname), geburtsdatum` umstellen (Migration). Dann würde die Handanlage einer
  Schreibvariante an der Datenbank abgewiesen, und die Maske muss das erklären — deshalb
  eine Produktfrage, kein Reflex.

- **Drei Definitionen für das Ende eines Bildungsgangs** (05.09.2026, B).
  `internal/ausweis/gueltigkeit.go` (H 9, Mittelstufe 10, Oberstufe 13, kennt E/Q),
  `calculateAbgaengerJahr` in `api/student_create.go` (h 9; Stufe ≥ 11 → 13; alles andere
  10 — auch G) und die Versetzung in `api/student_promotion.go` (10+h, 11+r, ≥ 14). Folge:
  Ein 10G steht in der Akte mit „Abgang <dieses Jahr>", die Versetzung setzt ihn nach 11G.
  Kein stiller Schaden — der Wert ist Anzeige und editierbar, die Löschung rechnet mit dem
  tatsächlichen Abgangsjahr —, aber drei Wahrheiten für eine Frage. Beim Bau der
  Abschlussklassen-Liste (Entscheidung 2): eine Definition, drei Verbraucher.

- **Elf Overlays bauen ihr Dialog-Markup selbst statt `ui/Modal.svelte`** (05.09.2026, C).
  Das Verhalten ist zusammengeführt (`use:escapeSchliesst`, geratscht in
  `escapeSchliesst.test.js` — dort steht die Dateiliste), die Bauform ist es (Rahmen ab,
  Erhebung bleibt). Offen ist allein das Markup: ein Durchgang am Bildschirm, Einzelfall
  für Einzelfall, kein Suchen-und-Ersetzen. Ausnahme bleibt: `OmniboxBlockAlert` und
  `OmniboxVormerkungAlert` behalten `border-4 border-rose-500` — der Rahmen ist dort das
  Signal, das einen Schüler an der Ausleihe stoppt, kein Dekor. Sie stehen benannt in
  `frontend-hygiene-bauform.test.js`.

- **Buchcover: Rest nach dem Bestellbedarf** (04.09.2026). Der Bestellbedarf zeigt das
  Cover in der Zeile (`ui/BuchCover.svelte`); auf dem Zielsystem tragen 5.724 von 8.706
  Titeln ohne Exemplar ein Cover — die Anzeige lohnt. Offen:

  - **16 Bestandsstellen** bauen ihr Cover noch selbst (Liste in
    `frontend-hygiene-cover.test.js`, eingefroren). Umstellen beim nächsten fachlichen
    Anfassen — nicht in einem Rutsch, das sind täglich benutzte Bildschirme.
  - **Arbeitslisten bleiben ohne Cover** (entschieden 05.09.2026): Das führende Element der
    `ArbeitsZeile` ist der Klassenkreis, nach dem Klassensatz-Reservierungen und Wünsche &
    Meldungen überflogen werden — M3 kennt genau eines. Die zwei fehlenden Backend-Felder
    (`isbn` in `GetKlassensatzReservierungen`, `cover_url` in `anliegen.go`) werden deshalb
    nicht gebaut.
  - **Portal** ohne Cover: `AnliegenWidget`, `PortalSchulbuecher`, `PortalLernmittel`.
  - **3.000 Titel ohne ISBN** (`PENDING`, werden vom Sync nie versucht): Datenfrage, keine
    Reparatur — zehn Jahre alte Littera-Daten. `inventur.SucheTextDNB` (Freitext) **nur mit
    Bestätigung durch einen Menschen** verdrahten: Freitext auf „Mathematik" liefert
    hunderte Treffer, den ersten zu nehmen hängt ein falsches Cover an ein Buch. Vorbild:
    DNB-Signaturvorschlag (22a10b1).

## Offen — Entscheidung nötig (Peter)

Was einem Menschen zur Entscheidung vorgelegt wird, gehört HIER hin, bevor die Antwort
kommt — ein Vorschlag, der nur im Gespräch steht, überlebt die Sitzung nicht.

1. **Abgänger mit nie geschlossenem Vorgang bleiben unbegrenzt** (05.09.2026). Ein offenes
   Buch oder ein unbezahlter Schadensfall schützt vor Anonymisierung und Löschung — richtig,
   solange die Bibliothek mahnt oder abrechnet. Schließt niemand den Vorgang (Buch kommt nie
   zurück und wird nie als Verlust gebucht), bleibt der Datensatz mit Namen und Anschrift auf
   Dauer, und keine Routine und kein Wächter meldet das.
   **Empfehlung: kein Automatismus, sondern ein Befund in der Betriebsbereitschaft** —
   „n Abgänger mit offenen Vorgängen seit mehr als 365 Tagen" als Warnung mit Abhilfe
   (Verlust buchen und Rechnung stellen oder stornieren; danach greifen Karenz und Löschung
   von selbst). Was zu tun ist, entscheidet ein Mensch; der Wächter hält nur fest, DASS es
   zu tun ist. Kosten: eine Zählung im Wächter, ein Test, eine Zeile im Handbuch.

2. **„Abgänger" = wer die Schule zum Schuljahresende verlässt — eine Bedeutung, keine zwei
   Blöcke** (05.09.2026, Peter). Am Code belegt: Der Initial-Commit (29.05., Ansicht
   „Abgänger-Entlastung") filterte die Abschlussklassen 9H/10R/13 mit offenen Büchern — noch
   an der Schule, weiter ausleihend, Liste zum Einsammeln vor der Entlassung. 15beae53
   (25.06., Nebensatz in einem Briefkopf-Commit) und 37d6542c (16.07., Laufzettel) stellten
   auf `ist_abgaenger = true` („fehlt im LUSD-Export") um; Anlass war ein echter Fehler
   (hartkodierte Klassennamen fanden „09H1" nicht), Wirkung ein Bedeutungswechsel, den
   Handbuch, Fachkonzept und Karenz-Arbeit seitdem als Wahrheit beschreiben. Mein erster
   Vorschlag (zwei Blöcke, Sekretariatseintrag, Tür „Abgang melden") war zu kompliziert —
   von Peter verworfen.
   **Zu bauen:** Abgänger sind die Abschlussklassen des laufenden Schuljahrs. Die Regel hat
   die Versetzung schon (`api/student_promotion.go`: 9+h, 10+r, 13), angewandt auf die
   AKTUELLE Klasse über Jahrgang und Zweig, nie über Klassennamen als Text. Ansicht und
   Laufzettel filtern darauf; Ausleihe unverändert. **Sichtbar ab dem 1. Mai** bis zur
   Versetzung (Peter, 05.09.: „vielleicht wäre der Mai gut"); fester Wert, keine
   Einstellung. Vorher zeigt die Ansicht den Hinweis „Abschlussklassen erscheinen hier ab
   Mai", damit die Tür nicht tot wirkt. Wer laut LUSD schon weg ist und noch Bücher hat, steht in der Mahnliste —
   kein eigener Block. Paar-Gate: Versetzungs-Vorschau und Abgänger-Liste liefern dieselbe
   Menge. Danach Handbuch und Fachkonzept zurückschreiben. Die Frist-Frage löst sich über 3.

3. **LMF-Plan: Rückgabe- und Ausgabetermine je Klasse statt Excel** (05.09.2026, Peters
   Vorschlag, zwei Listen der Schule gesehen). Die Schule führt eine Tabelle Wochentag,
   Datum, Stunde, Klasse(n), Besonderheiten: vor den Sommerferien „Bücherrückgabe" für alle
   Klassen, Abschlussklassen zuerst; wer keine Abschlussklasse ist, bekommt am selben Termin
   die neuen Bücher; nach den Ferien „Bücherausgabe" für die neu gebildeten Klassen (5er,
   7er „neu") plus „Nachzügler"; dazwischen Zeilen „Bücher setzen" ohne Klasse; Zeilen mit
   zwei Klassen („6F1/6F2") und freien Vermerken. Klassenzahl schwankt je Jahr. Der Plan
   geht als PDF per Mail ans Kollegium.
   **Vorschlag:** Tabelle `lmf_termine` (Datum, Stunde, Art Rückgabe/Ausgabe, 0..n Klassen
   aus dem Vokabular, freier Text), eine Seite unter Lernmittel, PDF in der heutigen Form,
   Vorbelegung aus `klassen` (Abschlussklassen zuerst, je eine Stunde, dann von Hand
   schieben). Zweiter, getrennt zu entscheidender Schritt: Der Termin einer Klasse wird die
   Rückgabefrist ihrer LMF-Bücher — heute gilt für alle der globale Stichtag `lmf_stichtag`.
   **Verteilung über das Kollegiums-Portal** (Peter, 05.09.): Der Plan wird heute per Mail
   ans Kollegium geschickt, Korrekturen als Folge-Mail („alle anderen Termine bleiben
   gleich"). Im Portal ist er immer aktuell: ein Reiter mit der ganzen Tabelle, für alle gleich,
   in der Reihenfolge des PDFs. KEINE Personalisierung nach Klassenleitung (Peter: auch
   Fachlehrer gehen mit ihren Klassen zum Büchertausch, und `klassen_lehrer_mapping` ist
   nicht für jede Klasse gefüllt); das Portal liest
   die Tabelle live, kein gespeichertes PDF (Peters Bedingung: „muss sich selbst
   aktualisieren"). Versand: PDF zum Herunterladen reicht (Peter); die App kennt nicht das
   ganze Kollegium, nur Klassenleitungen (`klassen_lehrer_mapping`) und Portal-Nutzer
   (`benutzer`, Rolle kollegium). Falls die Schule eine Verteiler-Adresse hat, reicht EINE
   Einstellung dafür — keine Adress-Sammlung in der App.
   **Entschieden (Peter, 05.09.):** (a) Der Rückgabe-Termin einer Klasse IST die
   Rückgabefrist ihrer LMF-Bücher („das wäre doch logisch"). Regel: nur Zeilen der Art
   Rückgabe setzen die Frist; Ausgabe-Zeilen nicht; Klasse ohne Termin → globaler Stichtag
   `lmf_stichtag` wie bisher; danach greift das Mahnwesen wie bei jeder Frist. (b) Die
   Abschlussklassen erscheinen unabhängig vom Plan als Abgänger (Entscheidung 2 bleibt);
   der Plan bestimmt nur, WANN ihre Bücher fällig sind. (c) Neuer Plan startet leer — eine
   Vorbelegung würde mit Platzhalter-Daten sofort Fristen setzen; stattdessen zeigt die
   Seite, welche Klassen noch keinen Termin haben.

## Beobachten (nichts zu tun)

| Fund                                                                                       | Warum nur beobachten                                                                                                                                                         |
| ------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `golang.org/x/crypto/openpgp` gilt als unwartbar (`GO-2026-5932`)                          | Kein Fix verfügbar (`Fixed in: N/A`), transitiv, **kein** Aufrufer im eigenen Code (`govulncheck`, zuletzt 31.08. bestätigt). Dependabot meldet sich, falls sich das ändert. |
| Designer-Restlücke: Browser-Schließen binnen 800 ms verliert die letzte Auto-Save-Änderung | `onDestroy` schickt seit `37abcbe0` sofort; `sendBeacon` bewusst nicht gebaut (Randlage).                                                                                    |
| 082-Dedupe der Vormerkungen verlor den neueren `abholbereit`-Eintrag                       | Auf Prod gelaufen, nicht rückholbar; nur relevant, falls je eine weitere **gewachsene** DB migriert wird.                                                                    |

## Kategorie C — bewusst nicht ohne Anlass

Einzeiler; die ausführlichen Begründungen stehen in der Git-Historie dieser Datei.

| Posten                                                                                                                                                                                                             | Warum geparkt                                                                                                                                                                                                                                                                                                    |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Knopfzeile über den Reitern (Mahnwesen, 04.09.)                                                                                                                                                                    | Sie kommt aus dem gemeinsamen Seitengerüst. Daran zu drehen entscheidet über ALLE Seiten — Anlass wäre ein Rundgang über das Gerüst, nicht eine Seite.                                                                                                                                                           |
| 3 handgebaute Pillen-Gruppen in `StatsDashboard` (`pills`)                                                                                                                                                         | Zweite Fassung von `ui/Segmente.svelte`. Tauschen beim nächsten fachlichen Anfassen der Statistik.                                                                                                                                                                                                               |
| Etikettenraster doppelt (`api/label_formats.go` ↔ `src/lib/etikettformate.js`)                                                                                                                                     | 31.08. ENTSCHIEDEN geparkt: Server-Umbau bräuchte neuen Endpunkt + async-Ladezustand in drei Verbrauchern des täglichen Druckbildschirms — neuer Ausfallmodus gegen null Risiko heute. `etikettformate-konsistenz.test.js` hält die Drift. Wieder anfassen bei viertem Verbraucher oder konfigurierbarem Format. |
| Reste des Nie-verdrahtet-Sweeps (01.09.)                                                                                                                                                                           | `inventur_sessions.gestartet_von` nie angezeigt (Produktfrage); `abgaenger_jahr` in der Aktivlisten-Antwort ohne Konsument; Geräte-Torso (`ActionEvent.GeraetID`, kein Broadcast, Null-Zeitstempel) wartet auf den Geräte-Ausbau.                                                                                |
| Cognitive Complexity: `gocognit -over 15` ohne Tests = 32 Funktionen (Messung 05.09.2026)                                                                                                                          | Zählweise rechnet den Handler-Closure als Ebene. Lohnend allenfalls `OverrideDueDateHandler` (30), `behandleAbgaenger` (25).                                                                                                                                                                                     |
| `javascript:S6551`, `javascript:S8783`                                                                                                                                                                             | Begründete Dauer-Ausnahmen.                                                                                                                                                                                                                                                                                      |
| Tabellen-Inline-Felder (Rückgabedatum, Exemplar-Barcode) 36 px                                                                                                                                                     | `size="sm"`-Variante von `Feld` erst bei Bedienbefund.                                                                                                                                                                                                                                                           |
| `LabelHeight >= 30` steht 2× in zwei Dateien (`api/label_pdf.go`, `api/schueler_etikett_pdf.go`; Messung 05.09.2026)                                                                                               | Schwellwert-Doppelung ohne Wirkung.                                                                                                                                                                                                                                                                              |
| Zwei Go-Normalformen für Namen: `repository.Suchnorm` (Zwilling von SQL `suchnorm`, LUSD-Schlüssel) und `normName` in `api/lusd_paarung.go` (eigene Umlaut-Tabelle, zusätzlich ohne Bindestrich/Leerzeichen/Punkt) | Die Paarung ist ein Vorschlag mit menschlicher Bestätigung, Unschärfe dort harmlos. Beim nächsten Anfassen der Paarung: `normName` auf `Suchnorm` aufsetzen und nur die Zeichen-Tilgung behalten.                                                                                                                |
| Paritätstest vergleicht keine COMMENTs/Seeds                                                                                                                                                                       | Kosmetik; Struktur ist gedeckt.                                                                                                                                                                                                                                                                                  |
| Jules-Erbe                                                                                                                                                                                                         | Go-Testdateien > 200 Zeilen aus Jules-PRs; Export-CSV-„breaks stream"-Test schwach.                                                                                                                                                                                                                              |
| Klone: Go `dupl -t 100` ohne Tests = 9 Gruppen (05.09.2026); Frontend `jscpd` 0,41 % (ältere Messung, Werkzeug lokal nicht installiert)                                                                            | Unter der Schwelle.                                                                                                                                                                                                                                                                                              |

## Außerhalb dieses Registers (Betrieb, liegt bei Peter)

Die EINE Liste der offenen Betriebs-Punkte. Littera-Details in
[littera_schema_befund.md](littera_schema_befund.md).

- **Frisches Littera-Backup** — littera_sav.mdb ist ein 2010er-Stand; Anforderungen
  (FremdLeserNummer/FremdBarcode) in [littera_schema_befund.md](littera_schema_befund.md).
- **Drei Sekretariats-Abnahmen** à ~10 Minuten (LUSD-Import, Versetzung ⏰ vor dem
  Schuljahreswechsel, Klassensatz-Erledigen) — Ablauf in
  [abnahme_checkliste.md](abnahme_checkliste.md).
- **Zielumgebung** (die Seite System → Betriebsbereitschaft zeigt den Ist-Zustand):
  Prod-Secrets mit `ENFORCE_PROD_SECRETS=true` (ohne `BACKUP_ENCRYPTION_KEY` läuft
  KEIN Backup) · Schul-IMAP/SMTP-Zugangsdaten · **`SELBSTANMELDUNG_DOMAIN`** setzen ·
  einmalige manuelle Restore-Probe am echten Ziel ([DEPLOYMENT.md](DEPLOYMENT.md) §6) ·
  `SENTRY_DSN` leer lassen (A6 in
  [datenschutz_offene_punkte.md](datenschutz_offene_punkte.md)) ·
  **S3-Auslagerung der Backups**.
- **GitHub**: PR-Pflicht abschaffen (Solo-Entscheidung 30.07.; das Ruleset `main` trägt am
  05.09. noch die `pull_request`-Regel), „Block force pushes" und „Restrict deletions"
  anlassen.
- Datenwert Schulname/„Neuer Text" auf Live korrigieren.

**Parkdeck** (bewusste Nicht-Entscheidungen, nur mit Anlass wieder anfassen):
Integer-Cent-Refactor (float64/NUMERIC) · Bundle-Splitting (720-kB-Chunk) ·
TypeScript-Migration (null TS-Dateien) · Verschmelzung `inventur/` ins Haupt-API ·
`cmd/migrate` (MySQL) löschen — hat keine Datenquelle mehr, aber seine PG-Tests
sichern mit `internal/uebernahme` geteilten Code · Zukunftsideen API-Versionierung
(`/api/v1`) und Mandantenfähigkeit (RLS).

Stand: 2026-09-05
