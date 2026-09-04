# Befund-Register

**Zweck:** Ein Ort für Dinge, die beim Arbeiten auffallen, aber nicht sofort angefasst
werden. Ohne diese Liste gibt es nur zwei Umgangsformen mit einem Fund — sofort
reparieren oder vergessen — und beide sind falsch.

Ergänzt [`invarianten.md`](invarianten.md): Dort steht, was immer wahr sein _muss_.
Hier steht, was aufgefallen ist und noch nicht entschieden oder erledigt wurde.

**Erledigtes wird gelöscht, nicht abgehakt** — es steht vollständig in
`git log -p docs/befunde.md`. Stände vor früheren Kürzungen: `2e09ec14` (31.08.),
`47a09f70` (01.09.), `9f91784b` (04.09.).

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

---

## Offen — abarbeitbar

Abgearbeitetes steht in der Git-Historie dieser Datei (`git log -p docs/befunde.md`),
nicht hier.

- **Buchcover im Bestellbedarf: Anzeige nachziehen, Datenlage trägt sie** (04.09.2026,
  Peter). Im Bestellbedarf steht statt des Covers ein Platzhalter; das Bild erscheint nur
  über `CoverPeek` auf Anforderung. Vier Geschwister derselben Seite zeigen es längst in
  der Zeile: `OrderSearch`, `OrderCart`, `WareneingangTable`, `BestellDetailPositionen`.

  **Auf dem Zielsystem gemessen** (13.060 Titel), nachdem eine erste Messung auf der
  lokalen Seed-DB mit 2,1 % zu einer falschen Empfehlung geführt hatte:

  | Bestand         | `FOUND` | `PENDING` (ohne ISBN) | `NOT_FOUND` | `FAILED` | Cover-Quote |
  | --------------- | ------- | --------------------- | ----------- | -------- | ----------- |
  | Schülerbücherei | 7192    | 2787                  | 1820        | 683      | **57,6 %**  |
  | Lernmittel      | 300     | 213                   | 53          | 12       | **51,9 %**  |

  Über die Hälfte der Titel hat ein Cover — die Anzeige in der Zeile lohnt sich also.
  `PENDING` ist **exakt** die Menge ohne ISBN (3000 Titel): Der Sync-Query verlangt
  `isbn <> ''`, sie werden nie versucht. Ursache laut Peter: rund zehn Jahre alte
  Littera-Daten ohne ISBN; die aktuellen Titel kamen als Klassensätze aus einem anderen
  Programm.

  **Der Sync selbst ist in Ordnung** und war zwischenzeitlich zu Unrecht als
  „Endlosschleife ohne Backoff" kritisiert: 8 Worker mit globaler Drossel von 500 ms je
  Titel, `NOT_FOUND` wird **nicht** wiederholt (nur `FAILED` = Abruffehler, und das ist
  richtig), dazu Überlappungsschutz und Panik-Netz.

  **Vor dem Bauen zu klären** (Abfrage steht in der Sitzung vom 04.09.): Wie hoch ist die
  Cover-Quote SPEZIFISCH bei den Titeln des Bestellbedarfs? Die Liste zeigt Titel mit
  niedrigem Bestand — auf dem Zielsystem „179 von 247 ohne ein einziges Exemplar" —, und
  das könnten überproportional die ISBN-losen Altdaten sein. Ist die Quote dort niedrig,
  bleibt `CoverPeek` richtig; ist sie wie im Gesamtbestand, gehört das Cover in die Zeile.

  **Wenn gebaut wird, dann als Bauteil:** Fünf Cover-Breiten im Haus
  (`w-7`/`w-8`/`w-10`/`w-12`/`w-16`), und die Fallback-Kette aus `coverKandidaten` steht in
  **vier** Dateien kopiert. Muss eine Komponente werden, kein Snippet — sie braucht State
  pro Instanz (Kandidaten-Index, `naturalWidth`-Prüfung gegen das 1×1-GIF des Proxys).
  M3-Mass: `list-item-leading-image` 56 px, `leading-space` 16 px.

  Für Titel ohne ISBN gibt es `inventur.SucheTextDNB` (Freitext, heute von
  `order_service.go` genutzt) und „Cover ändern" von Hand in der Buchakte. **Nicht
  automatisch verdrahten:** Freitext auf „Mathematik" liefert hunderte Treffer; den ersten
  zu nehmen hängt ein falsches Cover an ein Buch, und das ist schlimmer als keines. Beim
  DNB-Signaturvorschlag wurde deshalb Vorschlag-plus-Bestätigung gebaut (22a10b1).

  Offen bleibt Peters Wunsch für **Klassensatz-Reservierungen** (`reservation.go` liefert
  weder `cover_url` noch `isbn`) und **Wünsche & Meldungen** (`anliegen.go` hat `isbn`,
  kein `cover_url`).

- **Elf Overlays bauen ihren Dialog selbst, statt `Modal.svelte` zu benutzen**
  (04.09.2026). `Modal.svelte` trug Rahmen **und** Schatten — die Bauform, die M3 bei
  keinem seiner 84 Bauteile kennt; behoben, wirkt auf elf Dialoge. Diese elf wiederholen
  den Fehler in eigenem `fixed inset-0`: `StudentLockModal`, `DamageReportModal`,
  `StudentProfileDeleteModal`, `StudentGebuehrenCard`, `WebcamCapture`,
  `OmniboxBlockAlert`, `OmniboxVormerkungAlert`, `OmniboxChecklistDialog`,
  `MahnwesenTable`, `IsbnLookupDialog`, `StrichcodeScanner`.

  Quelltext-Zählung nach den Fixes: **40 Kandidaten**, überwiegend in diesen Overlays,
  der Rest Buchcover (Bild, kein Bauteil) und Etikettenvorschau (simuliert Papier).
  **Nicht pauschal ändern:** `OmniboxBlockAlert` trägt `border-4 border-rose-500` als
  Alarmsignal an der Theke — dort IST der Rahmen die Aussage. Einzelfallprüfung am
  Bildschirm, eigener Durchgang. `e2e/m3-bauform.spec.js` sieht die Fälle nicht (Dialoge
  sind zu); seine Öffnerliste umfasst zwei.

## Offen — Entscheidung nötig (Peter)

Was einem Menschen zur Entscheidung vorgelegt wird, gehört HIER hin, bevor die Antwort
kommt — ein Vorschlag, der nur im Gespräch steht, überlebt die Sitzung nicht (04.09.2026:
drei vorgelegte Geschmacksfragen waren so schon verloren und mussten neu erzählt werden).

Rest aus dem Rasterdurchgang 02.09.2026 (LUSD-Umbenennung, 0aa07f57). Reihenfolge = Abarbeitung;
Peter überlegt beide (03.09.).

1. **Handanlagen als Paar-Kandidaten.** Ein von Hand angelegter, nie LUSD-bestätigter Schüler,
   den die LUSD anders schreibt („Anna Müller" ↔ „Anna Mueller"), wird nicht als Paar
   vorgeschlagen, sondern Neuanlage + „nicht im Export". Rückweg heute: Zusammenführen über
   die Akte. Bauen hieße: `NichtImExport`-Zeilen als Kandidaten (`WarAbgaenger=false`) — und
   die Rubrik „nicht im Export" bedeutet dann etwas anderes.
2. **Karenz ab Rückgabe statt ab Abgang.** `abgaenger_seit` stempelt beim Abgang, auch mit
   offenen Büchern; wer erst nach Ablauf der Karenz zurückgibt, wird in der Folgenacht
   anonymisiert — das Reparaturfenster fehlt dieser Gruppe. Alternative: Uhr bei Rückgabe des
   letzten Vorgangs neu starten (dann PG-Test dieser Konstellation).

3. **`GET /api/search` hängt an `perform_actions`, drei Aufrufer liegen außerhalb der Theke.**
   Die Etiketten-Titelsuche im Druck-Center (Menüpunkt: `view_students`), die Schülersuche im
   Vormerkungs-Reiter der Buchakte (`view_books`) und die globale Suchleiste rufen dieselbe
   Route wie die Theken-Omnibox. In der Werksvorgabe fällt das nicht auf, weil jede Rolle mit
   `view_books`/`view_students` auch `perform_actions` trägt — entzieht ein Admin einer Rolle
   das Theken-Recht, meldet die Suche auf diesen Seiten nur noch „Suche nicht möglich" (laut,
   daher B). Die Suchleiste blendet sich seit 03.09. abends ohne `perform_actions` aus; die
   beiden anderen Felder nicht. Optionen: (a) Oder-Rechte-Middleware (`perform_actions` ODER
   `view_books` ODER `view_students`) — dann sähe eine Rolle mit bloßem Katalogrecht die
   Kiosk-Sicht auf Schüler (Name, Klasse, Ausweis; PII-Matrix Stufe 1), das ist eine
   Datenschutz-Entscheidung; (b) beide Felder wie die Suchleiste an `perform_actions` koppeln;
   (c) so lassen und in der Rechte-Oberfläche beim Theken-Recht darauf hinweisen.

Aus dem Design-Rundgang 04.09.2026 (Peter vorgelegt, Antwort steht aus). Alles
Geschmacksfragen — nichts davon kann jemandem schaden, deshalb wird nichts davon
eigenmächtig entschieden.

4. **Zweites Feld neben der Suchpille im Bestellwesen.** Im Warenkorb steht neben der
   Suchpille ein zweites Feld „Titel suchen & hinzufügen". Es sucht NICHT die Liste,
   sondern legt in den Warenkorb — ein Formularfeld in einem eigenen Bereich, also kein
   Verstoß gegen „eine Suchleiste je Seite" (04.09.). Offen ist allein, ob die Nähe der
   beiden Felder trotzdem stört; dann Umbau.
5. **Die Klassennamen in den Klassensätzen sind sehr groß.** Noch **nicht gemessen** —
   der Punkt steht bisher nur als Eindruck hier, ohne Zahl aus dem Browser.

## Beobachten (nichts zu tun)

| Fund                                                                                       | Warum nur beobachten                                                                                                                                                         |
| ------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `golang.org/x/crypto/openpgp` gilt als unwartbar (`GO-2026-5932`)                          | Kein Fix verfügbar (`Fixed in: N/A`), transitiv, **kein** Aufrufer im eigenen Code (`govulncheck`, zuletzt 31.08. bestätigt). Dependabot meldet sich, falls sich das ändert. |
| Designer-Restlücke: Browser-Schließen binnen 800 ms verliert die letzte Auto-Save-Änderung | `onDestroy` schickt seit `37abcbe0` sofort; `sendBeacon` bewusst nicht gebaut (Randlage).                                                                                    |
| 082-Dedupe der Vormerkungen verlor den neueren `abholbereit`-Eintrag                       | Auf Prod gelaufen, nicht rückholbar; nur relevant, falls je eine weitere **gewachsene** DB migriert wird.                                                                    |

## Kategorie C — bewusst nicht ohne Anlass

Einzeiler; die ausführlichen Begründungen stehen in der Git-Historie dieser Datei.

| Posten                                                                         | Warum geparkt                                                                                                                                                                                                                                                                                                    |
| ------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Knopfzeile über den Reitern (Mahnwesen, 04.09.)                                | Sie kommt aus dem gemeinsamen Seitengerüst. Daran zu drehen entscheidet über ALLE Seiten — Anlass wäre ein Rundgang über das Gerüst, nicht eine Seite.                                                                                                                                                           |
| 3 handgebaute Pillen-Gruppen in `StatsDashboard` (`pills`)                     | Zweite Fassung von `ui/Segmente.svelte`. Tauschen beim nächsten fachlichen Anfassen der Statistik.                                                                                                                                                                                                               |
| Etikettenraster doppelt (`api/label_formats.go` ↔ `src/lib/etikettformate.js`) | 31.08. ENTSCHIEDEN geparkt: Server-Umbau bräuchte neuen Endpunkt + async-Ladezustand in drei Verbrauchern des täglichen Druckbildschirms — neuer Ausfallmodus gegen null Risiko heute. `etikettformate-konsistenz.test.js` hält die Drift. Wieder anfassen bei viertem Verbraucher oder konfigurierbarem Format. |
| Reste des Nie-verdrahtet-Sweeps (01.09.)                                       | `inventur_sessions.gestartet_von` nie angezeigt (Produktfrage); `abgaenger_jahr` in der Aktivlisten-Antwort ohne Konsument; Geräte-Torso (`ActionEvent.GeraetID`, kein Broadcast, Null-Zeitstempel) wartet auf den Geräte-Ausbau.                                                                                |
| 20× `go:S3776` (Cognitive Complexity)                                          | Zählweise rechnet den Handler-Closure als Ebene. Lohnend allenfalls `OverrideDueDateHandler` (30), `behandleAbgaenger` (23).                                                                                                                                                                                     |
| `javascript:S6551`, `javascript:S8783`                                         | Begründete Dauer-Ausnahmen.                                                                                                                                                                                                                                                                                      |
| Tabellen-Inline-Felder (Rückgabedatum, Exemplar-Barcode) 36 px                 | `size="sm"`-Variante von `Feld` erst bei Bedienbefund.                                                                                                                                                                                                                                                           |
| `LabelHeight >= 30` steht 6× in zwei Dateien                                   | Schwellwert-Doppelung ohne Wirkung.                                                                                                                                                                                                                                                                              |
| LUSD-Namensschlüssel nur `lower+trim`                                          | Umlaut-/Bindestrich-Varianten gelten als verschiedene Menschen — sicher, aber nicht klug.                                                                                                                                                                                                                        |
| Paritätstest vergleicht keine COMMENTs/Seeds                                   | Kosmetik; Struktur ist gedeckt.                                                                                                                                                                                                                                                                                  |
| Jules-Erbe                                                                     | Sieben Testdateien > 200 Zeilen; Export-CSV-„breaks stream"-Test schwach.                                                                                                                                                                                                                                        |
| Klone: Go `dupl` 8 Paare, Frontend `jscpd` 0,41 %                              | Unter der Schwelle.                                                                                                                                                                                                                                                                                              |

## Außerhalb dieses Registers (Betrieb, liegt bei Peter)

Seit dem 01.09.2026 ist dieser Abschnitt die EINE Liste der offenen Betriebs-Punkte —
der frühere `master_fahrplan.md` ist aufgelöst (seine Code-Punkte waren erledigt, der
Rest stand doppelt; Littera-Details jetzt in
[littera_schema_befund.md](littera_schema_befund.md), Historie in `git log`):

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
- **GitHub**: PR-Pflicht abschaffen (Solo-Entscheidung 30.07.), „Block force pushes"
  und „Restrict deletions" anlassen.
- Datenwert Schulname/„Neuer Text" auf Live korrigieren.

**Parkdeck** (bewusste Nicht-Entscheidungen, nur mit Anlass wieder anfassen):
Integer-Cent-Refactor (float64/NUMERIC) · Bundle-Splitting (720-kB-Chunk) ·
TypeScript-Migration (null TS-Dateien) · Verschmelzung `inventur/` ins Haupt-API ·
`cmd/migrate` (MySQL) löschen — hat keine Datenquelle mehr, aber seine PG-Tests
sichern mit `internal/uebernahme` geteilten Code · Zukunftsideen API-Versionierung
(`/api/v1`) und Mandantenfähigkeit (RLS).

Stand: 2026-09-04
