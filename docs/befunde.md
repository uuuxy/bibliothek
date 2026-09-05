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

## Reihenfolge — womit weitermachen (05.09.2026)

Diese Liste ordnet nur, sie beschreibt nichts doppelt; die Sache selbst steht jeweils
unten. Was oben steht, blockiert oder verfällt; was unten steht, wartet ohne Schaden.

1. **Peters fünf Entscheidungen** (Abschnitt „Entscheidung nötig"). Punkt 2 (Karenz ab
   Rückgabe) ist der einzige mit Verfallsdatum: Er betrifft Schüler, die nach dem Abgang
   zurückgeben.
2. **Cover in den Arbeitslisten** — steht jetzt als Frage 6 bei dir, nicht als Bauauftrag:
   Das Backend-Feld fehlt gar nicht, die Zeile hat nur schon ein führendes Element.
3. **Die elf Overlays auf `Modal.svelte`** — ein Durchgang am Bildschirm, Einzelfall für
   Einzelfall, nicht als Suchen-und-Ersetzen. Die Ratsche unten nennt die Dateien.
4. **16 Cover-Stellen** und Kategorie C: nur beim nächsten fachlichen Anfassen.

Erledigt am 05.09.: Release `v1.9.6` gezogen (39 Commits seit v1.9.5), der Jules-Stapel
aus 32 PRs abgearbeitet (19 übernommen), `monitorTakt.test.js` entwackelt und die
Bauform-Ratsche gebaut.

**Getrennte Arbeitsverzeichnisse** für parallele Sitzungen bleiben offen, sind aber gerade
gegenstandslos: Am 05.09. läuft nur eine Sitzung auf dem Verzeichnis. Sobald wieder zwei
gleichzeitig arbeiten, ist es der erste Handgriff — zwei Sitzungen auf einem Arbeitsbaum
haben am 04.09. einen E2E-Lauf entwertet und einen Commit blockiert.

Nicht auf dieser Liste, weil sie bei Peter liegt: der Abschnitt „Außerhalb dieses Registers".

---

## Offen — abarbeitbar

Abgearbeitetes steht in der Git-Historie dieser Datei (`git log -p docs/befunde.md`),
nicht hier.

- **Buchcover: Rest nach dem Bestellbedarf** (04.09.2026). Der Bestellbedarf zeigt seit
  9cf11044 das Cover in der Zeile (`ui/BuchCover.svelte`, lazy, `naturalWidth`-Prüfung,
  Initiale als Platzhalter; CoverPeek bleibt die Großansicht). Auf dem Zielsystem tragen
  5.724 von 8.706 Titeln ohne Exemplar ein Cover — die Anzeige lohnt. Offen:

  - **16 Bestandsstellen** bauen ihr Cover noch selbst (Liste in
    `frontend-hygiene-cover.test.js`, eingefroren). Umstellen beim nächsten fachlichen
    Anfassen — nicht in einem Rutsch, das sind täglich benutzte Bildschirme.
  - **Die beiden Arbeitslisten hängen NICHT am Backend** (korrigiert 05.09.2026, die
    Notiz vom 04.09. war falsch): `GetKlassensatzReservierungen` selektiert
    `coalesce(t.cover_url,'')` und der Handler reicht die Struktur unverändert
    durch — das Feld ist seit Langem da und wird im Frontend nur nicht gelesen.
    Fehlend ist dort allein `isbn` (die Ausweichquelle) und bei `anliegen.go`
    `cover_url` (ISBN hat es). Beides ist ein Zweizeiler — gebaut wird es aber
    erst mit der Entscheidung unten, sonst entsteht ein Feld ohne Verbraucher.
    **Portal**: `AnliegenWidget`, `PortalSchulbuecher`, `PortalLernmittel` ohne Cover.
  - **3.000 Titel ohne ISBN** (`PENDING`, werden vom Sync nie versucht): Datenfrage, keine
    Reparatur — zehn Jahre alte Littera-Daten. `inventur.SucheTextDNB` existiert
    (Freitext), aber **nur mit Bestätigung durch einen Menschen** verdrahten: Freitext auf
    „Mathematik" liefert hunderte Treffer, den ersten zu nehmen hängt ein falsches Cover
    an ein Buch. Vorbild: DNB-Signaturvorschlag (22a10b1). Der Sync selbst ist in Ordnung
    (Drossel 500 ms, `NOT_FOUND` wird nicht wiederholt) — die Kritik vom 04.09. war falsch.

- **Zwei Alarm-Overlays behalten Rahmen UND Schatten — bewusst** (05.09.2026, C). Von den
  elf selbstgebauten Overlays haben neun am 05.09. ihren Rahmen abgegeben und behalten die
  Erhebung: In M3 entscheidet die Rolle, welcher der beiden Teile weicht, und ein Dialog ist
  eine erhobene Fläche (level3, kein outline-Token) — dieselbe Antwort, die `Modal.svelte`
  am 04.09. bekommen hat. Alle neun liegen auf einem Abdunkler, keiner verliert seine Kante.

  `OmniboxBlockAlert` und `OmniboxVormerkungAlert` bleiben, wie sie sind: `border-4
  border-rose-500` ist dort kein Dekor, sondern das Signal, das einen Schüler an der Ausleihe
  stoppt. Ein Alarm wird nicht leiser gemacht, um eine Gestaltungsregel zu erfüllen. Sie
  stehen als benannte Ausnahme in `frontend-hygiene-bauform.test.js` (Bestand jetzt 25 statt
  34; der Rest sind Cover-Bilder, die Etikettenvorschau und Flächen, die kein Dialog sind).

  Offen bleibt, dass diese Dialoge ihr MARKUP weiterhin selbst bauen, statt `ui/Modal.svelte`
  zu benutzen. Das VERHALTEN ist seit 05.09. zusammengeführt: `use:escapeSchliesst` liegt in
  `components/ui/`, wird von Modal und von fünfzehn Overlays benutzt und ist über
  `escapeSchliesst.test.js` geratscht — wer ein neues Overlay ohne Tastaturweg baut, wird rot.

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
   **Empfehlung (05.09.): nicht bauen — messen.** Bei der Abnahme „LUSD-Import" die Vorschau
   auf echten Daten laufen lassen und jedes Paar „neu" + „nicht im Export" ansehen: WARUM
   matcht es nicht — zweiter Vorname, Umlaut, Bindestrich, Tippfehler? Der Schlüssel ist heute
   `lower + trim` (`repository.LusdNamensSchluessel`, EINE Stelle); die billigste Regel, die den
   echten Grund trifft, kommt dorthin. Kandidaten aus „nicht im Export" hätten die
   Paarungs-Signale (Eintritt, Anschrift) meist gar nicht: schwache Vorschläge, und eine
   Rubrik, die zweierlei bedeutet.
2. **Karenz ab Rückgabe statt ab Abgang.** `abgaenger_seit` stempelt beim Abgang, auch mit
   offenen Büchern; wer erst nach Ablauf der Karenz zurückgibt, wird in der Folgenacht
   anonymisiert — das Reparaturfenster fehlt dieser Gruppe. Alternative: Uhr bei Rückgabe des
   letzten Vorgangs neu starten (dann PG-Test dieser Konstellation).
   **Empfehlung (05.09.): ja — der einzige Punkt mit Schaden im Stillen** (unumkehrbar,
   nächtlich, ohne Zeugen). Und nicht als Schreiben auf `abgaenger_seit` bei der Rückgabe,
   sondern im Prädikat: `PredikatAnonymisierung` ist die EINE Quelle für Job und Wächter;
   dort zusätzlich „keine Rückgabe innerhalb der Karenz" (`NOT EXISTS … rueckgabe_am >
   NOW() - karenz`). Die Karenz läuft dann ab dem letzten abgeschlossenen Vorgang; ein
   Schreibweg an der Rückgabe wäre eine zweite Tür zum selben Zustand. PG-Test: Abgang vor
   100 Tagen, Rückgabe gestern → bleibt; Rückgabe vor 91 Tagen → fällt. Etwa eine Stunde.

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
   **Empfehlung (05.09.): keine der drei — „Recht folgt der Handlung".** (1) Die
   Etiketten-Titelsuche bekommt eine eigene Titel-Route unter `view_books`
   (`SearchTitlesFuzzy` existiert; der OPAC taugt dafür nicht, er blendet Lernmittel aus —
   genau die Etiketten, die gedruckt werden). (2) `/api/search` nimmt `perform_actions` ODER
   `manage_vormerkungen`, und der Vormerkungs-Reiter zeigt sein Suchfeld nur mit
   `manage_vormerkungen`: Wer Vormerkungen anlegen darf, darf Schüler finden — das Recht
   deckt die Personendaten bereits. Dann gibt kein Recht mehr her, als seine Handlung
   braucht, und die Datenschutz-Frage stellt sich nicht. Zwei bis drei Stunden.

Aus dem Design-Rundgang 04.09.2026 (Peter vorgelegt, Antwort steht aus). Alles
Geschmacksfragen — nichts davon kann jemandem schaden, deshalb wird nichts davon
eigenmächtig entschieden.

4. **Zweites Feld neben der Suchpille im Bestellwesen.** Im Warenkorb steht neben der
   Suchpille ein zweites Feld „Titel suchen & hinzufügen". Es sucht NICHT die Liste,
   sondern legt in den Warenkorb — ein Formularfeld in einem eigenen Bereich, also kein
   Verstoß gegen „eine Suchleiste je Seite" (04.09.). Offen ist allein, ob die Nähe der
   beiden Felder trotzdem stört; dann Umbau.
   **Empfehlung (05.09.): so lassen.** Erst ändern, wenn jemand beim Arbeiten danebengreift —
   dann ist es gemessen, nicht geraten.
5. **Die Klassennamen in den Klassensätzen sind sehr groß.** Gemessen am 05.09., im
   Quelltext: `KlassenKarte.svelte` führt den Klassennamen ZWEIMAL — `text-2xl font-bold`
   (24 px, headline-small) unter Bibliothek → Klassensätze und `text-base font-medium`
   (16 px) im Kollegiums-Portal (`kompakt`), dazu zwei verschiedene Zähler-Chips. Die Karte
   ist laut eigenem Kommentar ein ausklappbares LISTENELEMENT; dessen Headline ist in M3
   body-large, 16 px. Der Eindruck „sehr groß" ist die 24-px-Fassung.
   **Empfehlung: die kompakte Fassung für beide Orte** — eine Gestalt, und die Zwillinge sind
   weg. Eine halbe Stunde, sichtbar, deshalb deine Entscheidung.

6. **Cover in den beiden Arbeitslisten — wohin?** (05.09.2026) Klassensatz-Reservierungen und
   Wünsche & Meldungen laufen beide über `ArbeitsZeile.svelte`, die M3-„list item"-Zeile mit
   FÜHRENDEM ELEMENT: links ein 48-px-Kreis mit der Klasse. Das war deine Entscheidung vom
   26.08. mit einer Begründung, die weiter gilt — „welche Klasse will welches Buch" ist die
   Frage, mit der die Bibliothek diese Listen überfliegt. Ein Cover davorzusetzen hieße zwei
   führende Elemente in einer Zeile, und M3 kennt genau eins. Möglichkeiten: (a) so lassen —
   in einer Arbeitsliste ist das Buchbild Zierde, der Titel steht ja da; (b) Cover STATT
   Klassenkreis, Klasse wandert in den Nebentext (dann ist die Liste nach Büchern zu
   überfliegen, nicht nach Klassen); (c) Cover klein hinter dem Titel als Anreißer. Meine
   Empfehlung ist (a): Diese Listen werden abgearbeitet, nicht gestöbert. Bis das entschieden
   ist, bleiben die zwei fehlenden Backend-Felder ungebaut.

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
| Jules-Erbe                                                                     | Acht Testdateien > 200 Zeilen (neu: `inventur/lernmittel_pdf_test.go`, 257); Export-CSV-„breaks stream"-Test schwach.                                                                                                                                                                                                                                        |
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

**Empfohlene Reihenfolge (05.09.2026):** (0) System → Betriebsbereitschaft auf dem
Schulserver LESEN — zwei Minuten, und was dort rot ist, schlägt alles auf dieser Liste;
insbesondere: läuft ein Backup? (1) Frisches Littera-Backup anfordern — die einzige Pflicht,
die nur mit fremder Hilfe geht, mit Wartezeit, und sie wird ZWEIMAL gebraucht: jetzt zum
Erproben der Übernahme, am Umstellungstag noch einmal frisch. (2) Restore-Probe am Ziel —
ein Backup, das nie zurückgespielt wurde, ist eine Hoffnung. (3) S3-Auslagerung. (4) `v1.9.6`
auf den Test-Server — der Abstand zwischen Image und laufendem Stand wächst mit jedem Release,
Migrationen lieber einzeln erleben als später im Bündel. (5) Abnahmen terminieren; die
LUSD-Abnahme NACH Entscheidung 2 und MIT der Messung aus Entscheidung 1. (6) Handgriffe, wenn
man ohnehin dran ist: `SELBSTANMELDUNG_DOMAIN`, PR-Pflicht abschaffen, Schulname korrigieren.

**Parkdeck** (bewusste Nicht-Entscheidungen, nur mit Anlass wieder anfassen):
Integer-Cent-Refactor (float64/NUMERIC) · Bundle-Splitting (720-kB-Chunk) ·
TypeScript-Migration (null TS-Dateien) · Verschmelzung `inventur/` ins Haupt-API ·
`cmd/migrate` (MySQL) löschen — hat keine Datenquelle mehr, aber seine PG-Tests
sichern mit `internal/uebernahme` geteilten Code · Zukunftsideen API-Versionierung
(`/api/v1`) und Mandantenfähigkeit (RLS).

Stand: 2026-09-05
