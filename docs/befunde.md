# Befund-Register

**Zweck:** Ein Ort für Dinge, die beim Arbeiten auffallen, aber nicht sofort angefasst
werden. Ohne diese Liste gibt es nur zwei Umgangsformen mit einem Fund — sofort
reparieren oder vergessen — und beide sind falsch.

Ergänzt [`invarianten.md`](invarianten.md): Dort steht, was immer wahr sein _muss_.
Hier steht, was aufgefallen ist und noch nicht entschieden oder erledigt wurde.

**Erledigtes wird gelöscht, nicht abgehakt** — es steht vollständig in
`git log -p docs/befunde.md`. Stände vor früheren Kürzungen: `2e09ec14` (31.08.),
`47a09f70` (01.09.), `9f91784b` (04.09.), `e2bd4b72` (05.09.), `36c7bdce` (05.09. abends: Abgänger + LMF-Plan gebaut).

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

**Reihenfolge:** die Overlays auf `Modal.svelte`, alles Weitere beim nächsten fachlichen
Anfassen.

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

- **Dritte Definition für das Ende eines Bildungsgangs** (05.09.2026, B; seit dem Bau der
  Abgängerliste am selben Tag nur noch EINE Restdefinition). Versetzung, Klassenleitungs-
  Zuordnung und Abgängerliste lesen jetzt `repository.AbschlussklasseSQL` (H ab 9, R ab 10,
  sonst ab 13; Regel-Tabelle `TestAbschlussklasseSQL_Regel`, Paar-Gate
  `TestAbgaengerliste_UndVersetzungSehenDieselbeMenge`). Übrig: `calculateAbgaengerJahr` in
  `api/student_create.go` (h → 9, ≥ 11 → 13, sonst 10 — auch G) rechnet das Anzeigejahr
  „Abgang <Jahr>" der Akte; ein 10G steht dort mit dem falschen Jahr. Anzeige, editierbar,
  kein Filter — Zwilling in Go bewusst nicht gebaut (Vollprobe fehlte). Beim nächsten
  Anfassen der Akte: Jahr aus der SQL-Regel rechnen lassen (eine Abfrage) statt in Go.
  `internal/ausweis/gueltigkeit.go` bleibt eigene Sache (Ausweis-Gültigkeit ≠ Abgang).

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

- **LMF-Plan: Tagesplaner statt Termin-Dialog** (Peter 05.09.2026, nach dem ersten Blick auf
  die gebaute Seite: „so ganz zufrieden bin ich nicht"). Drei Befunde am Bau von 1890a4df:

  1. **Das Anlegen ist zu klein und zu kleinteilig.** Ein Modal je Zeile; wer sechs Klassen
     ankreuzt, bekommt EINE Zeile mit sechs Klassen in EINER Stunde — gemeint ist aber: sechs
     Klassen nacheinander, Stunde 1 bis 6. Peters echter Plan 2026 (Excel „lmf termine 26",
     05.09. abends) zeigt die Form: Rückgabe Do 11.06. ab 3. Std. 9H1, 9H2, 10R1/10R2, 10R3
     (Abschlussklassen zuerst), dann JEDER Schultag Stunde 1–6, eine Klasse je Stunde, die
     Reihenfolge läuft über die Tage weiter (8H, 9R, 8R, 7R, 7H, 6F paarweise, 5F, 10G, 9G,
     8G, 7G, 6G …); Ausgabe Mo 10.08. ab 2. Std. „7G1 neu" … „5F4 neu", zuletzt „Nachzügler",
     „Aufräumen" ohne Klasse. Zwei Tippfehler im Excel (Freitag 17.06., Montag 19.06.) —
     Wochentag und Datum von Hand sind fehleranfällig. Recherche (Gymnasium Wentorf,
     Hebbelschule Kiel, Kaiserin-Friedrich Bad Homburg, Realschule Waibstadt): überall dasselbe
     Muster, je Klasse „weniger als eine Unterrichtsstunde", die Lehrkraft der Stunde bringt
     die Klasse. **Der Plan ist eine REIHENFOLGE von Klassen, die auf Schultage × Stunden
     gegossen wird** — keine Liste einzeln angelegter Zeilen.
     **Vorschlag:** Die Seite wird ein Planer mit drei Schritten. (a) Rahmen: Art, erster Tag,
     Startstunde am ersten Tag, Stunden je Tag (Vorgabe 6); Schultage rechnet das System
     selbst (Mo–Fr, `ferien_schliesszeiten` ausgespart). (b) Reihenfolge: eine Liste der
     Klassen, per Ziehen sortierbar; Vorbelegung = **Plan des Vorjahres** (liegt in
     `lmf_termine`, Klassennamen bleiben Jahr für Jahr gleich, weil die Versetzung Schüler
     verschiebt, nicht Namen), sonst Abschlussklassen zuerst, dann Jahrgang absteigend; nicht
     im Vokabular stehende Klassen („7G1 neu" vor dem LUSD-Import) frei eintippbar. Die
     Liste hat zwei Teile, „Im Plan" und „Nicht im Plan": **Die Oberstufe braucht den Ablauf
     an dieser Schule nicht, sie organisiert Rückgabe und Ausgabe selbst** (Peter 05.09.
     abends) — E/Q/12/13 stehen beim ersten Plan unten, das Vorjahr merkt es sich, und
     „Noch ohne Rückgabe-Termin" mahnt sie nicht an; ihre Lernmittel behalten den Stichtag.
     Für eine andere Schule ist die Oberstufe damit optional, nicht ausgeschlossen.
     (c) Das System verteilt die Reihenfolge auf Tage und Stunden; die Tabelle zeigt das
     Ergebnis und ist DORT editierbar: „mit voriger Zeile zusammenlegen" (10R1/10R2 teilen sich
     die Stunde), Leerzeile einfügen (Bücher setzen, Nachzügler, Aufräumen — mit Vermerk),
     Besonderheit je Zeile, Zeile entfernen; jede Änderung fließt nach. Kein Modal. Portal-
     Reiter und PDF bleiben, wie sie sind. Schema: `lmf_termine` trägt das Modell schon
     (mehrere Klassen je Termin, Termin ohne Klasse); neu nur die Reihenfolge als Attribut
     (`position`) und ggf. `stunde_bis` für Doppelstunden — im echten Plan kommen keine vor,
     also erst bei Bedarf.
  2. **Stundenraster ist Schulsache.** Heute CHECK 1–12 in der Tabelle und `STUNDEN` 1–12 im
     Frontend. Der echte Plan nutzt Stunde 1–6; das ist die Planer-Vorgabe „Stunden je Tag",
     am Plan änderbar, keine eigene Systemeinstellung. Uhrzeiten zeigt der Plan nicht — das
     Excel hatte keine, das Kollegium kennt sein Raster.
  3. **Klassen-Schema ist hart verdrahtet** — vier Stellen lesen Jahrgang und Zweig aus dem
     Namen mit Regeln DIESER Schule: `repository.AbschlussklasseSQL` (H ab 9, R ab 10, sonst
     13), `calculateAbgaengerJahr` (student_create.go), `internal/ausweis/gueltigkeit.go`
     (E/Q-Oberstufe, Gymnasialzweig bis 10), `pkg/lmf.JahrgangAusZielgruppe`. Eine andere
     Schule (Q1 statt 12, „12T1", nur „7a") hätte keine Tür, das einzugeben. Vorschlag:
     Einstellungs-Block „Klassen-Schema" unter Schule — Tabelle Zweig-Buchstabe → Bezeichnung →
     letzter Jahrgang (H 9/10, R 10, G 13, F = Förderstufe, T = Oberstufe) plus Aliase ohne
     Ziffer (E→11, Q1/Q2→12, Q3/Q4→13); Vorgabe = heutige Regel, die vier Stellen lesen die
     Einstellung. Der Tagesplaner braucht daraus nur die Reihenfolge (Abschluss zuerst).
     **Reihenfolge-Frage:** jetzt (Bibliosys hat eine Schule) oder erst mit der zweiten?
  4. **Menüplatz.** Zweimal im Jahr gebraucht, steht aber dauerhaft unter Bibliothek. NICHT in
     die Einstellungen: dort steht Konfiguration (manage_settings), der Plan ist ein Arbeits-
     dokument mit Fristfolge und PDF (edit_books). Vorschlag: eine Seite „Schuljahreswechsel"
     unter Verwaltung mit den Einmal-im-Jahr-Aufgaben als Reiter — LMF-Plan, Abgänger,
     Versetzung (heute in SystemSettings) — statt dreier Menüpunkte. Alternative: Menüpunkt nur
     in der Saison zeigen (Mai–September) — verworfen, weil „wo ist es hin?" schlimmer ist als
     ein Punkt zu viel.

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
