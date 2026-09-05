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

**Reihenfolge:** erst Peters Entscheidungen (Punkt 2 dort ist der einzige mit
Verfallsdatum), dann die Overlays auf `Modal.svelte`, alles Weitere beim nächsten
fachlichen Anfassen.

---

## Offen — abarbeitbar

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
  - **Zwei Backend-Felder für die Arbeitslisten** warten auf Entscheidung 6 unten:
    `GetKlassensatzReservierungen` liefert `cover_url`, aber kein `isbn` (Ausweichquelle);
    `anliegen.go` liefert ISBN, aber kein `cover_url`. Je ein Zweizeiler — ohne
    Verbraucher aber nicht bauen.
  - **Portal** ohne Cover: `AnliegenWidget`, `PortalSchulbuecher`, `PortalLernmittel`.
  - **3.000 Titel ohne ISBN** (`PENDING`, werden vom Sync nie versucht): Datenfrage, keine
    Reparatur — zehn Jahre alte Littera-Daten. `inventur.SucheTextDNB` (Freitext) **nur mit
    Bestätigung durch einen Menschen** verdrahten: Freitext auf „Mathematik" liefert
    hunderte Treffer, den ersten zu nehmen hängt ein falsches Cover an ein Buch. Vorbild:
    DNB-Signaturvorschlag (22a10b1).

## Offen — Entscheidung nötig (Peter)

Was einem Menschen zur Entscheidung vorgelegt wird, gehört HIER hin, bevor die Antwort
kommt — ein Vorschlag, der nur im Gespräch steht, überlebt die Sitzung nicht.

Rest aus dem Rasterdurchgang 02.09.2026 (LUSD-Umbenennung, 0aa07f57):

1. **Handanlagen als Paar-Kandidaten.** Ein von Hand angelegter, nie LUSD-bestätigter Schüler,
   den die LUSD anders schreibt („Anna Müller" ↔ „Anna Mueller"), wird nicht als Paar
   vorgeschlagen, sondern Neuanlage + „nicht im Export". Rückweg heute: Zusammenführen über
   die Akte. Bauen hieße: `NichtImExport`-Zeilen als Kandidaten (`WarAbgaenger=false`) — und
   die Rubrik „nicht im Export" bedeutet dann etwas anderes.
2. **Karenz ab Rückgabe statt ab Abgang.** `abgaenger_seit` stempelt beim Abgang, auch mit
   offenen Büchern; wer erst nach Ablauf der Karenz zurückgibt, wird in der Folgenacht
   anonymisiert — das Reparaturfenster fehlt dieser Gruppe. Alternative: Uhr bei Rückgabe des
   letzten Vorgangs neu starten (dann PG-Test dieser Konstellation).
3. **`GET /api/search` hängt an `perform_actions`, zwei Aufrufer liegen außerhalb der Theke.**
   Die Etiketten-Titelsuche im Druck-Center (Menüpunkt: `view_students`) und die Schülersuche
   im Vormerkungs-Reiter der Buchakte (`view_books`) rufen dieselbe Route wie die
   Theken-Omnibox. In der Werksvorgabe fällt das nicht auf, weil jede Rolle mit
   `view_books`/`view_students` auch `perform_actions` trägt — entzieht ein Admin einer Rolle
   das Theken-Recht, meldet die Suche auf diesen Seiten nur noch „Suche nicht möglich" (laut,
   daher B). Optionen: (a) Oder-Rechte-Middleware (`perform_actions` ODER `view_books` ODER
   `view_students`) — dann sähe eine Rolle mit bloßem Katalogrecht die Kiosk-Sicht auf
   Schüler (Name, Klasse, Ausweis; PII-Matrix Stufe 1), das ist eine
   Datenschutz-Entscheidung; (b) beide Felder an `perform_actions` koppeln und ohne das
   Recht ausblenden; (c) so lassen und in der Rechte-Oberfläche beim Theken-Recht darauf
   hinweisen.

Geschmacksfragen aus dem Design-Rundgang 04.09.2026 — nichts davon kann jemandem schaden,
deshalb wird nichts davon eigenmächtig entschieden:

4. **Zweites Feld neben der Suchpille im Bestellwesen.** Im Warenkorb steht neben der
   Suchpille ein zweites Feld „Titel suchen & hinzufügen" (`OrderSearch.svelte`). Es sucht
   NICHT die Liste, sondern legt in den Warenkorb — ein Formularfeld in einem eigenen Bereich,
   also kein Verstoß gegen „eine Suchleiste je Seite". Offen ist allein, ob die Nähe der
   beiden Felder trotzdem stört; dann Umbau.
5. **Die Klassennamen in den Klassensätzen sind sehr groß.** Gemessen am Quelltext
   (05.09.2026): `KlassenKarte.svelte` setzt den Namen in der weiten Fassung auf
   `text-2xl font-bold text-slate-800`, also 24 px fett — die kompakte Fassung derselben
   Karte nimmt `text-base font-medium text-on-surface` (16 px). M3 gäbe einer Listenzeile
   title-medium (16 px); 24 px ist headline-small. Zweiter Punkt derselben Zeile: die weite
   Fassung greift auf `slate`-Farben statt auf die Token. Entscheidung: auf 16 px
   zurücknehmen (dann sind beide Fassungen bis auf die Dichte gleich) oder als bewusste
   Kachel-Überschrift so lassen.
6. **Cover in den beiden Arbeitslisten — wohin?** (05.09.2026) Klassensatz-Reservierungen und
   Wünsche & Meldungen laufen beide über `ArbeitsZeile.svelte`, die M3-„list item"-Zeile mit
   FÜHRENDEM ELEMENT: links ein 48-px-Kreis mit der Klasse (Entscheidung vom 26.08.: „welche
   Klasse will welches Buch" ist die Frage, mit der die Bibliothek diese Listen überfliegt).
   Ein Cover davorzusetzen hieße zwei führende Elemente in einer Zeile, und M3 kennt genau
   eins. Möglichkeiten: (a) so lassen — in einer Arbeitsliste ist das Buchbild Zierde, der
   Titel steht ja da; (b) Cover STATT Klassenkreis, Klasse wandert in den Nebentext (dann ist
   die Liste nach Büchern zu überfliegen, nicht nach Klassen); (c) Cover klein hinter dem
   Titel als Anreißer. Empfehlung: (a) — diese Listen werden abgearbeitet, nicht gestöbert.
   Bis zur Entscheidung bleiben die zwei Backend-Felder oben ungebaut.

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
| Jules-Erbe                                                                     | Go-Testdateien > 200 Zeilen aus Jules-PRs; Export-CSV-„breaks stream"-Test schwach.                                                                                                                                                                                                                              |
| Klone: Go `dupl` 8 Paare, Frontend `jscpd` 0,41 %                              | Unter der Schwelle.                                                                                                                                                                                                                                                                                              |

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
