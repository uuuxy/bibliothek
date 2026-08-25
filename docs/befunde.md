# Befund-Register

**Zweck:** Ein Ort für Dinge, die beim Arbeiten auffallen, aber nicht sofort angefasst
werden. Ohne diese Liste gibt es nur zwei Umgangsformen mit einem Fund — sofort
reparieren oder vergessen — und beide sind falsch, wenn das System kurz vor dem
Pilotbetrieb steht.

Ergänzt [`invarianten.md`](invarianten.md): Dort steht, was immer wahr sein _muss_.
Hier steht, was aufgefallen ist und noch nicht entschieden oder erledigt wurde.

---

## Die Einordnung

Vor jedem Fund steht dieselbe Frage — **nicht** „ist das hässlich?", sondern
**„kann das still jemandem schaden?"**:

|       | Kategorie                                                                                                                                                                                            | Umgang                                                               |
| ----- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| **A** | Kann **stillschweigend** ein falsches Ergebnis für einen echten Menschen erzeugen: doppelte Mahnung an Eltern, Daten beim falschen Empfänger, verlorene Eingabe, ein Gate, das nicht rot werden kann | **Sofort**, eigener Commit, eigener Test, eigene Deploy-Entscheidung |
| **B** | Fehler, der sich **laut** meldet, oder Unordnung ohne Wirkung nach außen: totes Codestück, wackeliger Test, Doppelung                                                                                | **Hier notieren**, gebündelt nach dem Pilotstart                     |
| **C** | „Wenn ich schon mal hier bin" — Umbenennungen, Stilfragen, Refactorings ohne Anlass                                                                                                                  | **Nicht** vor dem Pilotstart                                         |

Zwei Regeln dazu:

1. **Ein Fund = ein Commit.** Kein Anhängen verwandter Aufräumarbeiten. Was beim
   Reparieren zusätzlich auffällt, kommt in diese Liste, nicht in denselben Commit.
2. **Kategorie A wird belegt, nicht behauptet.** Ein Fund dieser Klasse braucht einen
   Test, der mit dem alten Code rot wird. Ohne diese Gegenprobe ist unklar, ob
   überhaupt etwas repariert wurde.

---

## Offen

| Fund                                                              | Warum offen                                                                                                                                                                                                                                                                                                                                                                                                  |
| ----------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `golang.org/x/crypto/openpgp` gilt als unwartbar (`GO-2026-5932`) | Kein Fix verfügbar (`Fixed in: N/A`), transitive Abhängigkeit, **kein** Aufrufer im eigenen Code (`govulncheck`: „your code doesn't appear to call these"). Beobachten, nicht jagen.                                                                                                                                                                                                                         |
| `pgtest_support_test.go` liegt echt dupliziert in fünf Paketen    | Sauber wäre ein `internal/pgtest`-Paket; fasst die Test-Infrastruktur von fünf Paketen auf einmal an und ist nur gegen echtes PostgreSQL beweisbar. Steht als Befund in `sonar-project.properties`.                                                                                                                                                                                                          |
| Etikettenraster stehen an zwei Stellen                            | Maßgeblich ist `api/label_formats.go`; das Frontend führt seit dem 24.08.2026 genau EINE Kopie (`src/lib/etikettformate.js`, vorher vier Fundstellen). Der Umbau auf die Server-Liste — wie bei der Lieferantenseite gemacht — wäre eine Verhaltensänderung an einem täglich benutzten Bildschirm. Bis dahin hält `etikettformate-konsistenz.test.js` die beiden deckungsgleich, Spalten und Zeilen einzeln. |
| Tote Druck-CSS für Raster, die niemand mehr trägt                 | `druck-grundlagen.css` führt `.a4_grid`, `.card_printer`, `.print-labels-grid` und `.print-label-box`. Keine dieser Klassen steht noch in einem Markup — die Buch-Etiketten entstehen serverseitig als PDF. Rund 45 Zeilen CSS, die aussehen, als steuerten sie den Etikettendruck. Vor dem Löschen am gebauten Bundle nachsehen, nicht nur greppen.                                                         |
| `DELETE /api/schueler/deleted/{id}` hat keinen Aufrufer           | Das sofortige endgültige Löschen aus dem Papierkorb existiert als Route (Recht `manage_students_admin`), aber kein Bildschirm ruft sie; der nächtliche Löschjob räumt nach Frist. Entweder ein Knopf im Papierkorb (mit Rückfrage) oder die Route streichen — Fund des Raster-Durchgangs 24.08.2026.                                                                                                         |

---

## Textfelder waren die letzte M3-Lücke — ein Bauteil, eine Ratsche (25.08.2026)

Ausgangsfrage war klein: neun native Datumsfelder auf M3 ziehen. Beim Öffnen der ersten
Datei stand das Geburtsdatum zwischen Vorname und Nachname im **selben alten Slate-Stil**
— ein Datumsfeld allein im M3-Rahmen hätte das Formular uneinheitlicher gemacht.
Gemessen: **81 handgebaute `<input>` in 47 Dateien**, sieben Radien, vier Fokusfarben,
drei Flächen; dazu zwei Bauteile für dieselbe Sache (`SettingField` M3/rounded-sm,
`InputField` Slate). Buttons, Select und Suchfelder waren längst normiert, Textfelder
nur in der Höhe.

**Umgesetzt (Peters Entscheidung: „Formularfelder komplett auf M3"):**

- `components/ui/Feld.svelte` — 36 px, `rounded-xl` (Radius-Regel 55a1d4b0: Karten +
  Felder 12 px), `text-sm`, Rahmen/Fläche/Fokus exakt das Rezept von `Select.svelte`.
  `label` → 3-Zeilen-Subgrid; ohne `label` nur das Feld mit Pflicht-`aria-label`.
  `ungueltig`, `hint`, `feld` (Input-Klassen), `element` (bind:this), Rest per Spread.
- `SettingField` und `InputField` gelöscht; 78 Fundstellen in 46 Dateien umgestellt,
  vier davon als `ui/Suchfeld` (Autocomplete-Rolle), das Suchfeld selbst auf die
  M3-Rollen (stand als letztes Feld noch auf slate/blue).
- Ratsche `frontend-hygiene-felder.test.js` auf **0** mit drei begründeten Ausnahmen:
  Omnibox (Scan-Pille), `AusweisGueltigkeit` (zusammengesetzte Pille „Gültig bis 31.07.
  [Jahr]"), `ClassAssignmentSelector` (unsichtbares Tipp-Feld im Chip-Kasten).
- Farb-Ratsche 2611 → 2285, Dateigrößen-Bestand um 7 Dateien kürzer.

**Sichtbare Nebenwirkung, bewusst:** Einstellungsfelder schreiben jetzt 14 statt 16 px,
wie Select daneben. **Korrigiert nach Peters Blick:** Das Scan-Feld der Inventur war mit
der Migration auf 36 px gerutscht („deutlich kleiner als in Ausleihe") — falsch
eingeordnet, es ist das Werkzeug der Seite wie die Omnibox. Jetzt die 48-px-Suchpille
(`9c484c67`).

**Ein A-Fund beim Beweisen:** `SettingField` hatte `type="number"` als STANDARD, `Feld`
hat `"text"`. Nach der Umstellung liefen zwölf Einstellungsfelder und die Anzahl der
Klassensatz-Reservierung als String ans Backend — „cannot unmarshal string into … anzahl
of type int". Die Unit-Tests und 26 E2E-Specs blieben grün; gefunden hat es
`lehrer-reservierung.spec.js` im vollen Lauf. Fix: `type="number"` an allen 13 Stellen,
plus Wächter in `frontend-hygiene-felder.test.js` (min/max/step ohne type="number" ist rot).

**Raster-Gegenprobe (Peter: „sollten wir Daniels Raster nochmal drüber laufen lassen?"):**
Zwei Prüfungen, die kein Unit-Test leistet. (1) Attribut-Parität: jedes alte `<input>`
gegen sein neues `<Feld>`/`<Suchfeld>` gepaart — 80 Felder, 62 Dateien, keine verlorenen
Handler, Bindungen, `required`, `maxlength`, Datalists. (2) `e2e/feld-roundtrip.spec.js`:
sieben Schreibpfade ohne bisherigen E2E-Beweis bis in die Datenbank — Buch anlegen mit
Bestand/Zähldatum/Standort, Abgangsjahr, Rückgabedatum, Exemplar-Barcode, Anliegen anlegen
und erledigen mit Notiz, Einstellungs-Zahlenfeld. Alle grün; der einzige 400 unterwegs war
meine erfundene ISBN. Nebenbefund (B): Die sieben Bestell-Reiter brechen bei 1280 px in
zwei Zeilen um — M3 sieht dafür scrollbare Tabs vor.

**Nachgezogen (Peter: „wäre gut oder?"):** `vorlaufend`/`nachlaufend`-Snippets am Feld
(Inhalt links/rechts IM Feld, als Überlagerung wie bei Suchfeld — das Feld bleibt das
gerahmte 36-px-Element, das die Höhen-Ratsche misst). Damit sind die drei handgebauten
Subgrid-Hüllen (Einheit „Stück", ISBN-Knöpfe) und die Ausnahme `AusweisGueltigkeit`
aufgelöst; es bleiben zwei Ausnahmen (Omnibox-Pille, Chip-Kasten).

**Offen (C):** Tabellen-Inline-Felder (Rückgabedatum, Exemplar-Barcode) sind mit 36 px
höher als ihre Zeile — eine `size="sm"`-Variante wie beim Button wäre konsequent, ist
aber ohne Bedienbefund nicht gebaut.

## Orte — Entscheidungsrunde 24.08.2026 (Peter: „mach mal")

Sechs Ort-Fragen aus diesem Register, in einem Durchgang entschieden und umgesetzt:

| Fund                                                             | Entscheidung / Umsetzung                                                                                                                                                                                                                                                                                      |
| ---------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Versetzung + LUSD unter Einstellungen → Datenverwaltung          | Eigene Einstellungs-Kategorie **„Schuljahreswechsel"** (sichtbar mit `import_students` oder `manage_students_admin`). Peters Entscheidung nach einem Zwischenstand als Schülerdatei-Reiter: „gehört definitiv ins System-Menü unter Einstellungen". Datenverwaltung behält Import/Export/Offline-Sicherungen. |
| „Mahnwesen" in Gruppe Kiosk                                      | Nach **Verwaltung**, direkt unter Schülerdatei — Schülerarbeit, keine Thekenarbeit. Mitarbeiter sehen es (view_students); bleibt so.                                                                                                                                                                          |
| „Schulklassen" unter Verwaltung; Namenskollision mit „Jahrgänge" | Nach **Bibliothek**, umbenannt in **„Klassensätze"** (Route/id unverändert).                                                                                                                                                                                                                                  |
| Eingehende Anliegen melden sich nirgends                         | `GET /api/anliegen/anzahl` (view_orders); Badge an „Bestellungen" zählt Reservierungen + Anliegen.                                                                                                                                                                                                            |
| Kein Weg zum Klassensatz-Ausweisdruck                            | `KlassenDruckEinstieg` springt mit markierter Klasse in die Schülerdatei — als eigener Druck-Center-Reiter **„Klassenweise drucken"** (als Block über dem Ausweis-Designer wirkte er „hingeklatscht", Peter 24.08.).                                                                                          |
| Zwei Vorgaben für dasselbe Raster                                | **Zweckform L4760** überall (Server hatte es schon, Druck-Center stand auf Avery 3475).                                                                                                                                                                                                                       |

## Beispiel-Plugin abgeklemmt — und warum NICHT mehr (23.08.2026)

Beim Doku-Durchgang fiel auf, dass `plugins/vorlage` in Produktion mitläuft: zwei
Log-Zeilen bei jeder Rückgabe am Tresen, eine aus `DispatchEvent`, eine aus dem
Beispiel selbst. Demo-Code im Betrieb, an der meistbenutzten Stelle des Systems.

**Mein erster Entschluss war, das ganze System zu löschen.** Die Zahlen schienen
eindeutig: 128 Zeilen, genau EIN Ereignistyp, fünf Aufrufstellen die alle dasselbe
Ereignis feuern, ein Zuhörer der nur loggt, ein Frontend-Teil (`Extension.svelte`) der
nie angeschlossen wurde, und in 15 Monaten kein einziges echtes Plugin. Ich hatte die
Dispatch-Aufrufe bereits entfernt, als Peter sagte: „überlege wirklich gründlich".

**Zwei Dinge hatte ich übersprungen.**

Erstens tragen die fünf Aufrufstellen eine Entscheidung: Alle sitzen unmittelbar NACH
`tx.Commit`, direkt neben dem Audit-Eintrag — „nur melden, was wirklich und dauerhaft
passiert ist". (Das relativiert sich: Dieselben fünf Punkte findet man über
`LogRueckgabe` wieder. Das Wissen ginge nicht verloren, nur die Arbeit wäre erneut zu
tun.)

Zweitens — und das ist entscheidend — steht `plugins/` in diesem Register selbst als
**Kategorie C**, und die eigene Regel dazu lautet „nicht vor dem Pilotstart". Der Pilot
läuft seit heute Mittag. Ich war dabei, Rückgabe-, Ausleih- und Gerätepfade chirurgisch
anzufassen, um 128 Zeilen loszuwerden, die zur Laufzeit nichts kosten — gegen die
ausdrückliche Regel dieses Projekts, mitten im Pilotbetrieb.

**Gemacht wurde deshalb nur das, was ein Fehler ist:** `vorlage.Init()` aus `main.go`
entfernt. Ohne Zuhörer kehrt `DispatchEvent` still zurück, beide Log-Zeilen sind weg.
Eine Zeile statt einer Operation.

**Was das kostet:** zwei Einträge in `scripts/deadcode_baseline.txt` (`RegisterHook`,
`Init`), mit befristeter Begründung und Ablaufdatum „nach dem Pilot".

**Und wer hält die Entscheidung?** Nicht der Kommentar in `main.go` — der wäre genau
die Bauform, an der dieses Projekt heute schon dreimal hängengeblieben ist. Das
deadcode-Gate wirkt in beide Richtungen: Schließt jemand `vorlage.Init()` wieder an,
werden die Baseline-Einträge erreichbar und der Lauf meldet „Baseline-Einträge, die es
nicht mehr gibt". Am Rückbau geprüft.

**Nach dem Pilot zu entscheiden.** Stand heute spricht weiterhin alles fürs Löschen —
aber das ist dann eine Entscheidung mit Zeit, nicht eine nebenbei.

---

## Die offenen Punkte nachgeprüft (23.08.2026) — und meine Kritik am Register zurückgenommen

Ich hatte behauptet, dieses Register führe Erledigtes als offen weiter und trenne
„gefunden" nicht von „noch offen". **Das stimmt nicht.** Jeder Abschnitt nennt im Kopf
die Commits der Abarbeitung und darunter eine eigene Liste „bewusst offen". Was mich
getäuscht hat: Die Fundtabellen selbst tragen keine Status-Spalte, gelesen ohne den Kopf
darüber sehen sie aus wie offene Punkte. Das ist ein Lesbarkeits-, kein Pflegeproblem.

Die drei Stellen, an denen ich „veraltet" gerufen hatte, stehen im Abarbeitungs-Kopf
namentlich: `restore-backup` im Image (`5a55147f`), Go-/Node-Versionen und
PATCH-Geburtsdatum (`63d09011`). Ich hatte den Kopf nicht gelesen.

### Was tatsächlich offen ist — Punkt für Punkt am Code geprüft

| Punkt                                              | Nachgemessen am 23.08.2026                                                                              | Gilt noch |
| -------------------------------------------------- | ------------------------------------------------------------------------------------------------------- | --------- |
| `GO-2026-5932` in `golang.org/x/crypto`            | `govulncheck -show verbose`: `Found in v0.55.0`, `Fixed in: N/A`, kein Aufrufer im eigenen Code         | ✅ ja     |
| `pgtest_support_test.go` in fünf Paketen           | `find` zählt 5 (`api`, `repository`, `db`, `cmd/migrate`, `internal/littera`)                           | ✅ ja     |
| Etikettenraster an mehreren Stellen                | Maßgeblich `api/label_formats.go`; Kopien in `stores/labels.svelte.js` und `LabelLayoutOptionen.svelte` | ✅ ja     |
| Zwei Vorgaben für dasselbe Raster                  | `api/labels.go:19` → `avery_3475`, `api/pdf_service.go:131` → `zweckform_l4760`                         | ✅ ja     |
| Paritätstest ohne COMMENTs/Seeds                   | `conname` steht inzwischen drin (Zeile 40–43), `obj_description`/`col_description` kommen nirgends vor  | ✅ ja     |
| LMF hängt an der Namenskonvention                  | `pkg/lmf`: `~ '^lmf[ -]'` auf Titel und Signatur; kein `ist_lernmittel` im Schema                       | ✅ ja     |
| LUSD-Namensschlüssel nur `lower+trim`              | `repository/lusd_bestand.go:40` — genau das, nichts weiter                                              | ✅ ja     |
| Dependabot kann `golang:`/`node:` unabhängig heben | `.github/dependabot.yml`: der `docker`-Block hat kein `ignore`                                          | ✅ ja     |
| Altbackups vor 21.08. unlesbar                     | Betreiber-Hinweis steht in `resilience_and_recovery.md:23`                                              | ✅ ja     |
| `sonar.projectVersion` nicht gesetzt               | kommt weder in `sonar-project.properties` noch in `sonar_scan.sh` vor                                   | ✅ ja     |

**Ergebnis: kein einziger Eintrag war überholt.** Das Register ist genauer als meine
Kritik daran — und der Beleg dafür hat eine Stunde gekostet, was die richtige
Reihenfolge gewesen wäre, bevor ich es behauptet habe.

### Ein Fund, der NICHT im Register stand

`docs/ARCHITECTURE.md` führt unter „Komponenten-Regeln" ohne jede Einschränkung:
**„≤ 200 Zeilen pro `.svelte`-Datei"**. Gemessen: **43 von 206 Dateien brechen sie**, die
größte mit 412 Zeilen (`EtikettenNachdruck.svelte`), gefolgt von `LusdImportView` (394)
und `StatsDashboard` (391). Geprüft hatte das nie jemand — deshalb stand es auch nicht
hier.

Eine Regel, die ein Fünftel des Baums verletzt und nichts davon merkt, ist keine Regel,
sondern eine Absichtserklärung. Sie ist jetzt eine Ratsche
(`frontend-hygiene-dateigroesse.test.js`): Der Bestand von 43 ist eingefroren, eine neue
Datei über 200 Zeilen ist rot, und eine geduldete Datei, die WEITER wächst, ebenfalls —
sonst wäre die Ausnahme ein Freibrief. Wer eine Datei unter 200 bringt, muss sie
austragen; die Liste ist eine Arbeitsliste, keine Duldung.

Beide Zweige am Rückbau rot gesehen. Beim ersten Lauf meldete das Gate den ganzen
Bestand als gewachsen — `split('\n')` zählt eine Zeile mehr als `wc -l`. Auch ein
Detektor, der zu viel meldet, ist kaputt: Er wird abgeschaltet.

---

## Nachtrag: die erste Erklärung war falsch (23.08.2026)

Ein neuer Test war allein grün und in der vollen Suite rot — eine Klasse stand danach
als `7a` da, angelegt worden war `07a`. Meine Erklärung dafür lautete: „Der
Versetzungslauf schreibt in seinem Test jede Schülerzeile und zieht meine mit." Sie
klang plausibel, passte zum Symptom, und ich habe sie so ins Register geschrieben,
**ohne sie zu messen**.

Sie war falsch. Zwei Messungen genügten:

- Das `api`-Paket kennt **kein** `t.Parallel()` — Tests laufen nacheinander.
- `resetBestandsdaten` macht ein `TRUNCATE … schueler` — fremde Zeilen sind ohnehin weg.

Die echte Ursache ist eine Tabelle, die niemand zurücksetzt: `klassen`. Der Trigger
`trg_schueler_klasse_vokabular` (Migration 079) schlägt dort die kanonische Schreibweise
nach. Steht `7a` schon drin, wird ein später eingefügtes `07a` still als `7a`
gespeichert; ist die Tabelle leer, bleibt `07a` stehen und wird selbst zur kanonischen
Form. **Die Schreibweise einer Klasse hing also davon ab, welcher Test vorher lief.**

Behoben, indem `klassen` in beiden `resetBestandsdaten`-Helfern (`api`, `repository`)
mit geleert wird. Befüllt wird die Tabelle von keiner Migration — sie entsteht allein
durch den Trigger, ist also reines Ableitungsprodukt und gehört in den Reset. Der
strenge Vergleich auf `07a` steht wieder im Test und hält die Aussage: Wer `klassen`
aus dem Reset nimmt, macht die Suite rot.

**Was daran zählt:** In diesem Fall zeigte die Kopplung in die harmlose Richtung — ein
Test wurde grundlos rot. Die gefährliche Richtung ist die andere: ein Gate, das fremdes
Vokabular still **grün** hält. Genau dafür gibt es die Regel, Befunde am Live-Pfad zu
messen; sie gilt für die eigenen Erklärungen genauso wie für fremde Fehlerberichte.
Eine Notiz in diesem Register ist eine Zusicherung wie jede andere — und diese hier hat
keine 24 Stunden gehalten.

---

## Die 73 Schreibstellen des Frontends gelesen (23.08.2026) — zwei A-Funde

Anlass war eine Frage, keine Meldung: Die 206 Svelte-Dateien waren bis dahin nur über
Detektoren gelaufen, nie gelesen. Statt alle zu lesen, wurde die Fläche eingegrenzt auf
die Stellen, an denen etwas **geschrieben** wird — dort saßen alle bisherigen Funde.

**Die Eingrenzung war beim ersten Versuch falsch.** `grep "method: 'PUT'"` fand 23
Dateien. Die Komfort-Hülle (`apiClient.put(…)`, `apiPut(…)`) trägt aber kein `method:` —
real sind es **73 Schreibstellen in rund 40 Dateien**. Beide Funde unten sitzen in der
Hälfte, die die erste Sonde nicht sah. Dieselbe Lehre wie bei der
[statischen Inventur](../CLAUDE.md): Die Sonde muss man messen, bevor man ihr glaubt.

### Fund 1 (A): Stammdaten des Schülers ließen sich nicht löschen

|                         |                                                                                                                                                                                                     |
| ----------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Was behauptet wurde** | „Änderungen gespeichert"                                                                                                                                                                            |
| **Was stimmte**         | Die geleerte Adresse und die Eltern-Mail kamen nie an. Beim nächsten Öffnen der Akte stand der alte Wert wieder da.                                                                                 |
| **Warum**               | `useStudentEditForm.svelte.js` baute `strasse: formData.strasse ǀǀ null`. Im Backend sind die Felder `*string`; JSON-null wird nil, und nil heißt dort „nicht mitgeschickt, Spalte in Ruhe lassen". |
| **Wen es trifft**       | Genau die Angaben, deren Entfernung jemand verlangen kann — Postanschrift und Elternkontakt.                                                                                                        |

**Die Gegenprobe am laufenden Handler zeigte die zweite Hälfte:** Ein LEERER String kam
mit 200 durch und räumte Vorname, Nachname, Ausweisnummer und Klasse weg; aus der
geleerten Klasse leitete `calculateAbgaengerJahr` noch ein Abgängerjahr ab. Beim
Anlegen sind dieselben Felder `validate:"required"` — beim Ändern war die Regel nie
nachgezogen worden.

Dass daraus nie ein Schaden entstand, war **Zufall und keine Regel**: Das Formular
schickte null statt "". Hätte man nur die eine Hälfte repariert, wäre die andere scharf
geworden. Deshalb beide: leer = löschen (NULL) bei den optionalen Feldern, 400 mit
Begründung bei den Pflichtfeldern. `geburtsdatum` bleibt bewusst bei `ǀǀ null` — sonst
wäre jeder Altdatensatz ohne Geburtsdatum nicht mehr speicherbar.

Commit `77ee14b9`. Gates: `api/schueler_feld_leeren_pg_test.go` (beide Richtungen, gegen
echtes PostgreSQL) und `frontend/src/lib/useStudentEditForm.test.js` (hält die Nutzlast
fest, weil der Rückfall auf dieser Seite passiert). Beide am Rückbau rot gesehen.

### Fund 2 (A): Das Geräte-Formular hob still die Defekt-Markierung auf

|                         |                                                                                                                                                                                                                                                 |
| ----------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Was behauptet wurde** | „Gerät gespeichert"                                                                                                                                                                                                                             |
| **Was stimmte**         | Das defekte Gerät war danach wieder ausleihbar.                                                                                                                                                                                                 |
| **Warum**               | Im Handler stand `istAusleihbar := true`, wenn das Feld fehlt. Der Bearbeiten-Dialog kennt die Defekt-Markierung gar nicht — sie liegt auf einem eigenen Knopf in der Liste. Ein fehlendes Feld war damit kein „unverändert", sondern ein Wert. |
| **Wen es trifft**       | Das nächste Kind bekommt ein kaputtes Tablet.                                                                                                                                                                                                   |

Gegenrichtung derselben Stelle: Die **Seriennummer** schickt das Formular mit, der
Handler reichte sie nicht ans Repository weiter — eine Korrektur war folgenlos, wieder
mit Erfolgsmeldung. Beide Felder sind jetzt Zeiger: nil heißt „nicht angefasst",
"" heißt „gelöscht".

Commit `5c08a47e`. Gates: `api/geraet_bearbeiten_pg_test.go` — das Formular darf die
Markierung nicht aufheben, der Defekt-Knopf darf die Stammdaten nicht anfassen.

### Die Bugklasse dahinter

Beide Funde sind **dasselbe Missverständnis über Schweigen**: Ein nicht mitgeschicktes
Feld heißt im einen Handler „unverändert" und im anderen „nimm den Vorgabewert". Sender
und Empfänger waren sich jeweils nur zufällig einig.

Daraus folgt für jeden PUT/PATCH **eine zweite Frage**, die der Payload↔Struct-Vergleich
nicht stellt:

1. Welches Feld schickt das Formular, das der Handler nicht schreibt?
2. Welches Feld schickt es **nicht** — und was macht der Handler mit dem Schweigen?

Frage 2 ist die blinde Hälfte (vgl. Abschnitt „Audit-Raster").

### Gate-Härtung nebenbei

Das Einstellungs-Gate (`einstellungen-kategorien.test.js`) hielt eine **handgepflegte
Kopie** der 23 Schlüssel aus `repository/system_settings_patch.go`. Deckungsgleich —
aber ein neuer Schlüssel im Go-Struct hätte es nie erreicht, und seit dem strengen
Decoder hieße das 400 beim ersten Speicherversuch. Liest jetzt die Go-Datei, mit
Gegenprobe am Detektor selbst: Beim ersten Versuch kam eine leere Liste heraus (die
Tags tragen `,omitempty`), und ein Gate über eine leere Liste ist grün, ohne zu prüfen.
Der Wächter meldete es sofort. Commit `b9c54936`.

### Was sauber war

Buch-Rundlauf (der Listen-SELECT liefert alle Felder; `GetBookByID` ist nicht der
Live-Pfad und bedient nur Cover-Wege mit COALESCE-Schreiber), Mail-Passwort (zwei
getrennte Anweisungen, nicht nur ein Kommentar), Hauptlieferant, Sperr-Endpunkt,
CSRF-Ausnahmen für die zwei rohen `fetch`, Systematik, Mail-Vorlagen, Exemplar-Status,
Rechte-Matrix, Klassenzuordnung, Bestellungen, Etiketten, Vormerkungen.

### Was NICHT behauptet wird

Die restlichen 183 Svelte-Dateien sind weiterhin **nicht gelesen**, sondern über
Detektoren gelaufen. Das ist eine Entscheidung, keine Lücke: Beide Funde saßen auf
Schreibpfaden, und dort ist jetzt jede Stelle einzeln angesehen. Anzeigecode dieselbe
Aufmerksamkeit zu geben hieße, sie an der Stelle mit dem geringsten Ertrag auszugeben.

---

## Kollegiums-Portal: zwei Aufgaben, eine Fläche (23.08.2026)

Peters Frage zum Bildschirmfoto: „entspricht das Material 3 oder ist das nicht etwas
unübersichtlich? unten diese Suche und oben der OPAC?" — Ja, unübersichtlich, und aus
drei benennbaren Gründen. Verletzt waren nicht die Bauteile, sondern die
Informationsarchitektur.

1. **Zwei ungleiche Aufgaben auf einer Fläche ohne Gliederung.** Oben ein NAMENLOSES
   Suchfeld (täglich), unten ein benannter Abschnitt „Wünsche & Meldungen" (selten).
2. **Ein 340-px-Poster in der Mitte**, das wörtlich wiederholte, was einen Zentimeter
   darüber schon im Platzhalter stand („Titel, Autor oder ISBN eingeben").
3. **Die Formularfelder trugen die Pillenform der Suche.** „Welches Buch?" sah aus wie
   ein zweites Suchfeld; die Segmented Buttons darüber standen so dicht am Suchfeld,
   dass sie wie dessen Filterchips wirkten.

**Umgesetzt nach der Frage-Runde (alle drei Empfehlungen angenommen):** zwei M3-Reiter
(„Bücher & Klassensätze" / „Meine Anliegen" mit Zähler offener Anliegen) · statt des
Posters steht dort jetzt, **was gerade läuft** — die Warteschlange der Klassensätze und
die eigenen Anliegen · Formularfelder auf `SettingField`. Die Regel im Haus heißt damit:
**Pille = suchen, Rahmen = eingeben.**

**Nachtrag 25.08.2026 (abends) — Startfläche nur noch Reservierungen.** Mit vier Reitern
war der Auszug „Deine Anliegen" auf Reiter 1 der Inhalt eines Geschwister-Reiters (M3:
„each tab contains distinct content"), und „Offene Klassensätze" hieß wie der Reiter
„Klassensätze", der etwas anderes meint (Zuordnung Klasse → Bücher). Jetzt: ein Wort,
eine Bedeutung — „Deine Reservierungen"; Überschrift nur bei Einträgen, ein Leerzustand,
keine Trennlinie. Peters Frage: „sollte unter den oberen Reitern … liegen?"

**Nachtrag 25.08.2026 — vier Reiter statt zwei.** Der dritte Reiter „Lernmittel" (Klassensätze
und Bestand nach Jahrgang gestapelt, mit eigenen Überschriften) sah am Live-Bildschirm
„hingeklatscht" aus, und „Bücher & Klassensätze" hieß fast so wie der Abschnitt darin.
Jetzt vier gleichrangige Aufgaben: **Suchen & Reservieren · Klassensätze · Bestand nach
Jahrgang · Meine Anliegen**; der Reiter ist die Überschrift, Beitexte entfallen
(`a6c99ed3`). Am Rechteumfang ändert das nichts — weiterhin nur `create_reservations`,
die Lese-Sichten brauchen nur die Anmeldung (`FACHKONZEPT.md`, `4b46f1bd`).

**Vier Dinge kamen beim Bauen dazu:**

- **Reiter waren viermal von Hand gebaut** (Medienkatalog, Bestellwesen, Buch-Akte,
  Inventur-Startseite) und liefen auseinander — dieselbe Geschichte wie die zehn
  Suchfeld-Kopien. Erst `components/ui/Reiter.svelte`, dann der fünfte Reiter; die vier
  Bestandsfälle stehen als Ratsche eingefroren (`frontend-hygiene-reiter.test.js`).
- **Die Warteschlange kannte den Titel nicht.** Der Kommentar am Struct nannte „Titel,
  Klasse, Menge", geliefert wurde nur die `titel_id`. Neben einem Suchtreffer fiel das
  nicht auf (das Buch steht daneben); auf der Startfläche stand „Klasse 8B · 28 Stück"
  ohne jeden Hinweis, worum es geht. LEFT JOIN ergänzt, PG-Test am Rückbau rot gesehen.
- **Drei Abrufe für einen Zustand.** Zähler am Reiter, Startfläche und Anliegen-Reiter
  hätten dieselbe Liste dreimal geholt — nach dem Absenden hätte der Zähler den alten
  Stand gezeigt. Jetzt hält das Portal die Liste, die zwei Bauteile bekommen sie.
- **`KollegiumPortal.svelte` hatte 389 Zeilen** (Vorgabe: 200). Die Trefferkarte ist
  jetzt ein eigenes Bauteil; die Datei steht bei 288.

### Und derselbe Layoutfehler ein zweites Mal — eine Stunde später

Im neuen Anliegen-Formular hing „Klasse / Kurs" wieder eine Zeile tiefer als
„Anmerkung". Diesmal nicht wegen einer umbrechenden Beschriftung, sondern wegen einer
`<div class="sm:col-span-2">`-**Hülle** um ein Feld: Die Hülle ist selbst das
Rasterelement und spannt EINE Zeile, während ihre Nachbarn drei spannen.

Das ist der eigentliche Befund des Tages: **Mein frisch gebautes Gate hat es nicht
gefangen.** Die Schwelle stand bei 40 px, der Abstand betrug 46. Eine geratene Schwelle,
die zufällig für den ersten Fall passte.

Behoben in zwei Richtungen:

- Die Spaltenbreite gehört ans FELD (`<SettingField class="sm:col-span-2" />`), nicht an
  eine Hülle. `feld-huellen.test.js` schließt die Ursache aus — auch dort, wo kein
  e2e-Test hinsieht.
- Die Messung im Browser ist jetzt **kalibriert statt geraten**: echte Reihenabstände
  104 px (Einstellungen) und 124 px (Portal), aufgetretene Fehler 20 px und 46 px →
  Schwelle 60. Beide Fehlerarten am Rückbau rot gesehen, die Prüfung deckt jetzt auch
  Mail und das Portal ab.

Nebenbei: Der Nulllauf-Wächter von `control-hoehen.spec.js` schlug an (13 Felder statt

> 15. — die Einstellungen zeigen seit dem Umbau EINE Kategorie statt sieben Abschnitten.
>     Die Schwelle zu senken hätte die Aussage verkleinert; der Test läuft jetzt drei
>     Kategorien ab und misst wieder mehr als vorher.

---

## Selbstprüfung nach dem Deploy: die Mail-Kategorie war nur umhüllt (23.08.2026)

Auf Peters Bitte, die eigene Arbeit zu prüfen, habe ich alle Kategorien im Browser
angesehen statt sie für fertig zu halten. Befund gegen mich selbst:

**Q3 („outlined statt Unterstrich") war nur zur Hälfte umgesetzt.** Die drei
eingebetteten Altbauteile — `MailConfig`, `MailTemplates`, `DataManagement` — habe ich
in `KategorieRahmen` gehüllt, aber nicht angefasst. In der Kategorie „Mail" standen
deshalb **Unterstrich-Felder (Material 2) direkt neben den neuen outlined-Feldern**,
dazu ein schwarzer Speichern-Knopf neben blauen, eine Zwischenüberschrift GRÖSSER als
der Kategorietitel darüber, und zwei Zeilen Erklärtext, wo Peters Punkt 4 einen Satz
verlangt. Ein halber Umbau sieht schlechter aus als gar keiner: Vorher war die Seite
einheitlich M2, danach war sie gemischt.

`MailConfig` ist jetzt umgestellt (SettingField, Button, M3-Rollen, ein Satz je
Abschnitt, Zwischenüberschriften kleiner als der Kategorietitel). Palette-Ratsche
2823 → 2781.

**Offen und bewusst nicht in diesem Zug:** `MailTemplates` und `DataManagement` tragen
weiter eigene Bauteile. Sie sind größer und stehen jeweils allein in ihrer Kategorie,
mischen sich also nicht sichtbar mit neuen Feldern — anders als MailConfig, das mit den
Vorlagen dieselbe Kategorie teilt.

**Merksatz:** Wer eine Seite neu strukturiert und Altbauteile nur einhüllt, hat die
Inkonsistenz nicht geerbt, sondern erzeugt.

---

## Verrutschtes Feld auf flasch3 (23.08.2026)

Peters Bildschirmfoto nach dem Deploy: In „Datenschutz & Sitzung" stand das Feld
„Lesehistorie Schülerbücherei (Tage)" **eine Zeilenhöhe tiefer** als die drei daneben.

Ursache war nicht die lange Beschriftung, sondern dass jedes Feld ein eigener
Flex-Block war und sich nur an sich selbst ausrichtete: Bricht eine Beschriftung auf
zwei Zeilen um, rutscht ihr Eingabefeld mit. Behoben mit `grid-rows-subgrid` in
`SettingField` — alle Felder einer Reihe teilen sich jetzt dieselben drei Zeilen
(Beschriftung, Feld, Hinweis), gleich wie lang eine Beschriftung ist. Steht ein Feld
allein, läuft subgrid ins Leere und `display:grid` stapelt wie vorher; der Aufrufer
muss nichts dazutun.

**Nicht** durch Kürzen der Beschriftung: Das hätte denselben Fehler beim nächsten
schmaleren Fenster oder der nächsten längeren Beschriftung wieder erzeugt.

**Gate:** `e2e/einstellungen-kategorien.spec.js` misst im BROWSER über fünf Kategorien.
Regel: Zwei Eingabefelder stehen entweder auf derselben Höhe (eine Reihe) oder deutlich
auseinander (verschiedene Reihen, gemessen 104 px). Ein Abstand von einer einzelnen
Zeilenhöhe dazwischen IST der Fehler. Am alten Stand rot gesehen — mit genau Peters
Symptom: „(y=274) und (y=254) stehen 20 px auseinander".

Dabei mitgefunden: Der e2e-Helfer `einstellungsKategorie()` ankerte auf `^Titel` — damit
traf „Mahnwesen" auch „Mahnwesen-Routing". Jetzt `^Titel ` mit Leerzeichen.

---

## Raster-Durchgang über das GANZE Programm (23.08.2026) — und zwei Fragen mehr

Peters Auftrag: „wende Daniels Schema nochmal komplett auf das ganze Programm an —
erweitere es ggf." Die bisherigen Durchgänge liefen über _Commits_ (21./22.08.) oder über
_einen Umbau_ (Einstellungen). Dieser lief über den Baum: 45.000 Zeilen Go, 206
Svelte-Dateien, 89 Migrationen, 32 Tabellen.

### Die zwei neuen Fragen

Beide stammen aus echten Vorfällen dieses Projekts, und beide hätten den A5-Befund
**vorher** gefunden statt in einer externen Bewertung:

**9. Ausleitung — wo verlässt eine Kopie der Daten die Anwendung?**
Datei, Mail, Export, Protokoll, Log, Fremdsystem. Für jede Stelle: Wer darf sie lesen, wie
lange lebt sie, ist sie verschlüsselt? Die acht alten Fragen sehen alle _nach innen_ — auf
Schreibpfade, Zustände, Sichtbarkeiten. A5 (unverschlüsselte Dumps, 30 Tage) und der
A-Fund unten liegen beide außerhalb davon.

**10. Rückweg — ist der Weg zurück begehbar, und ist er am Ergebnis bewiesen?**
Liegt das Werkzeug dort, wo man es im Ernstfall braucht? Wurde der Rückweg an der fertigen
Datei geprüft oder nur am Vorgang? Aus drei Vorfällen: `restore-backup` fehlte im Image,
der `pg_dump`-17-Pin machte Prod-Backups monatelang nicht einspielbar, `/app/backups`
fehlte im Image und kein Backup entstand.

### Kategorie A — beide erledigt, je ein Commit, Gate am Rückbau rot gesehen

| Frage                       | Fund                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  | Erledigt als                                                                                                                                                                                                            |
| --------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **9 Ausleitung**            | `littera_import.log` bekam bei jeder Kollision den echten Datenwert. Postgres schreibt in `DETAIL` den Inhalt der Zeile (`Key (email)=(erika.mustermann@…) already exists.`), `uebernahme.BeschreibeFehler` hängte ihn wörtlich an; zusätzlich schrieb `personenlauf.mailadresse` die kollidierende Adresse selbst als Kennung. Ziel ist eine unverschlüsselte Datei im Arbeitsverzeichnis, ohne Frist. Trifft genau den Littera-Lauf, der als Nächstes ansteht. Am laufenden Postgres nachgestellt.                  | `428e4a06` — Wertteil geschwärzt, Form und Diagnosewert (SQLSTATE/Constraint/Spalte) bleiben; Lesernummer statt Adresse. Zwei Gates.                                                                                    |
| **3 zwei Wahrheitsquellen** | Die Rechte-Oberfläche zeigt „Statistiken anzeigen" (`view_stats`), das Menü blendet danach ein — `GET /api/statistiken` verlangte `view_students`. In der Vorgabe stimmen beide je Rolle **zufällig** überein, deshalb fiel es nie auf. Getrennt gewinnt still der Server: `view_stats` AUS entzog nichts (die Route antwortet weiter, nur der Menüpunkt verschwindet), `view_stats` AN ohne `view_students` zeigte einen Punkt, der jedes Mal 403 lieferte. `view_stats` war das EINZIGE Recht im System ohne Route. | `96d936ea` — Route verlangt `view_stats` (fachlich richtig: Antwort ist PII-Stufe 0), Matrix mitgezogen, `api/rechte_paritaet_test.go` hält beide Listen zusammen. Drei Rückbauten rot gesehen, zwei davon am Detektor. |

### Kategorie B — alle zehn abgearbeitet (23.08.2026)

Peters Ansage: „arbeite bitte alles ab … und Dinge, die du noch nicht abgedeckt hast,
bitte auch." Je Fund ein Commit, jedes Gate am Rückbau rot gesehen:

| Fund                                                                           | Erledigt als                                                                                                                                                                               |
| ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Aussonderung ohne `ist_ausleihbar = false`                                     | `a78327ec` — beide Zweige, dazu ein Gate über den ganzen Baum (jede aussondernde Anweisung muss beide Spalten setzen, die Gegenrichtung ebenso)                                            |
| Sortier-Meldung zählte die Eingabe statt der Wirkung · zwei stille e2e-Wächter | `c9c903ac` — `RowsAffected` + 404 bei null Treffern; `expect(a.or(b))` vor der Verzweigung, dazu eine Ratsche gegen das Muster                                                             |
| PDF-Datumsangaben in Container-Zeit (samt 14-Tage-Zahlungsfrist)               | `b28d7a7e` — Zeitzone in `pkg/schulzeit` gehoben (eine Quelle, kein Nachbau), Gate unter TZ=Pacific/Midway + Ratsche gegen rohes `time.Now()` in `pdf/`                                    |
| „leer" hieß beim Ändern dasselbe wie beim Anlegen (Bestand, Titel, Autor)      | `7c2d3d2f` — `stock` ist ein Zeiger (nil = nicht anfassen), leerer Titel/Autor wird 400 statt Platzhalter                                                                                  |
| `migrate-fotos` sagte nur, man könne aufräumen                                 | `0805d958` — löscht selbst, aber erst nach Gegenprobe (zurückgelesen, entschlüsselt, verglichen); `FOTOS_BEHALTEN=1` schaltet ab                                                           |
| Rollen-Vokabular in drei Schreibweisen                                         | `caf439be` — Gate gegen das Enum in `schema.sql`, beide Richtungen rot gesehen (u. a. der historische LEHRER-Fall)                                                                         |
| Drei Tabellen ohne Lebenszyklus                                                | `d07a56e7` — `lehrer_anliegen` bekommt eine Frist (365 Tage nach Erledigung, einstellbar, 0 = aus); die anderen zwei bleiben **bewusst** unbefristet (Bestandskartei-Nachweis, Belegwesen) |
| `DisallowUnknownFields` nirgends                                               | `1a293dee` — streng dort, wo die Nutzlast aus einer geschlossenen Liste entsteht (Einstellungen); dazu ein Paritäts-Gate Kategorien ↔ Patch-Struct                                         |

**Warum nicht global streng?** Mehrere Endpunkte nehmen bewusst das ganze Objekt
entgegen, das sie vorher ausgeliefert haben — samt der Felder, die nur der Server füllt
(`id`, `verfuegbar`, `gesamt`, `sortOrder` beim Buch). Dort wäre Strenge kein Gewinn,
sondern ein 400 auf einem Bildschirm, der heute funktioniert. Die Grenze verläuft nicht
zwischen „wichtig" und „unwichtig", sondern zwischen geschlossener und offener
Schlüsselmenge.

### Nachgezogen: die Flächen, die der Durchgang offen gelassen hatte

| Fläche                     | Ergebnis                                                                                                                                                                                                                                                                                                                                                                                             |
| -------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Objektbindung (IDOR)**   | Kein neuer Fund. Die zwei schwächsten Rollen erreichen zusammen drei schreibende Routen; die Theke ist in der PII-Matrix ausdrücklich als „bewusst ohne Objektbindung" vermerkt (jedes Buch auf jeden Ausweis — das IST die Theke), `ListEigeneAnliegen` bindet an `claims.UserID`, die öffentlichen Bestell-Links an den gehashten Token.                                                           |
| **`mailservice/`**         | **Ein Fund, gefixt (`5499d68e`).** Über `baueMailNachricht` stand „req.To und req.Subject müssen bereits sanitiert sein" — neun Aufrufer, keiner tat es. Geprüft war nur der SMTP-Umschlag; die Kopfzeilen der Nachricht nicht. Die Regel steht jetzt an einer Stelle und gilt für beide Wege.                                                                                                       |
| **`sse/`**                 | Kein Fund. Begrenzter Puffer, nicht-blockierendes Senden, Abmeldung per `defer`, RLock deckt das Senden mit ab; der Strom trägt nur IDs und Buchtitel (Matrix-Zeile Stufe 0).                                                                                                                                                                                                                        |
| **`plugins/`**             | ~~`RegisterHook` wird nirgends aufgerufen: null Zuhörer, tote Erweiterungsstelle.~~ **Falsch — am Startpfad widerlegt** (`main.go` rief `vorlage.Init()`). Das BEISPIEL-Plugin lief in Produktion mit und schrieb bei jeder Rückgabe zwei Zeilen ins Log (Titel, Barcode, Bearbeiter-UUID). **Erledigt 23.08.2026:** Der Init-Aufruf ist raus, der Erweiterungspunkt bleibt. Begründung siehe unten. |
| **Frontend (206 Dateien)** | Kein offener Fund über fünf Detektoren: `catch`-Blöcke (36 begründet, 7 stumm, alle um `scanner.stop()`), schreibende Aufrufe ohne Auswertung der Antwort (2, beide mit begründendem Kommentar), Rollen-Literale, Rechte-Schlüssel, gesendete Nutzlast-Schlüssel. Die zwei Paritäts-Gates von heute (Rechte, Rollen) prüfen den Frontend-Baum jetzt bei jedem Lauf mit.                              |

### Kategorie B — die Fundliste im Original

| Frage                   | Fund                                                                                                                                                                                                                                                                                                                                                                                                                                             | Beleg                                                                                                               |
| ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------- |
| 1 Konvention            | `DisallowUnknownFields` kommt im ganzen Projekt **nicht** vor. „Das Frontend schickt genau die Felder, die der Server kennt" ist damit reine Verabredung — die Klasse, die am 23.08. früh die Mail-Kategorie traf (`488f51d9`). Ein Abgleich aller gesendeten Schlüssel gegen alle `json`-Tags fand keinen offenen Fall, hat aber eine blinde Hälfte: ein Schlüssel, den ein ANDERES Struct kennt, fällt nicht auf.                              | `api/http_utils.go:19`, `inventur/endpunkte_buecher_aktualisieren.go:27`                                            |
| 1 Konvention            | `syncBookStock` setzt beim Aussondern `ist_ausgesondert = true` **ohne** `ist_ausleihbar = false` — die drei anderen Aussonderungswege setzen beides. Alle heutigen Leser prüfen beide Spalten; ein neuer, der nur `ist_ausleihbar` prüft, verleiht ein ausgesondertes Exemplar.                                                                                                                                                                 | `inventur/db_books_update.go:134-140` vs. `repository/audit_books.go:139`, `damage.go:228`, `book_inventory.go:73`  |
| 2 Spezialwert           | `PUT /api/books/{id}` ist ein Voll-Objekt-Schreibpfad ohne Untergrenze: fehlendes `stock` wird 0, und `syncBookStock` sondert dann den GESAMTEN Bestand aus — im Rückfallzweig auch **ausgeliehene** Exemplare. Die Warnung im Formular greift bei `NaN` nicht (`NaN < 5` ist `false`). Der eine echte Aufrufer lädt vorher das volle Objekt und bricht bei Ladefehler ab — abgesichert, aber nur durch einen Kommentar.                         | `inventur/endpunkte_buecher_aktualisieren.go:100`, `db_books_update.go:132-170`, `AdminBuchAktionen.svelte:15`      |
| 2 Spezialwert           | Beim Aktualisieren wird ein geleerter Titel still zu „Unbekannter Titel", ein geleerter Autor zu „Unbekannter Autor" — und ein geleertes Cover löst eine Fremdabfrage aus, die es neu füllt. Beim Anlegen ist das richtig; beim Ändern heißt „leer" etwas anderes.                                                                                                                                                                               | `inventur/endpunkte_buecher_aktualisieren.go:105,126-131`                                                           |
| 3 zwei Wahrheitsquellen | Das Rollen-Vokabular steht in **drei** Schreibweisen: DB-Enum klein (`'admin'`), Go-Konstanten groß (`RoleAdmin = "ADMIN"`), Frontend vergleicht exakt klein (`rolle === 'admin'`, ~10 Stellen). Zusammengehalten wird das nur von `EqualFold`/`UPPER()` im Go-Code. Wer die Antwort je auf die Go-Konstante normalisiert, lässt zehn Admin-Bedienelemente lautlos verschwinden — dieselbe Klasse wie der tote `"LEHRER"`-Zweig (Migration 069). | `schema.sql:35`, `auth/jwt.go:22-30`, `App.svelte:158` u. a.                                                        |
| 5 stille Fehler         | `handleReorderBooks` meldet „erfolgreich N Bücher sortiert" mit `len(input.BookIDs)` statt `RowsAffected` — bei unbekannten IDs ändert sich nichts, und die Meldung behauptet das Gegenteil.                                                                                                                                                                                                                                                     | `inventur/reorder_handler.go:58`                                                                                    |
| 6 Zeit                  | Die Datumsangaben in den PDFs nehmen `time.Now()` in Container-Zeit (UTC) statt der Schulzeitzone — inklusive der **14-Tage-Zahlungsfrist** auf dem Schadensbescheid. Zwischen 22 und 24 Uhr UTC trägt das Schreiben den Vortag. Dieselbe Klasse wie der Mahnketten-TZ-Befund, nur im gedruckten Dokument statt im Test.                                                                                                                         | `pdf/schadensfall.go:60,103`, `pdf/rechnung.go:102`, `pdf/kontoauszug.go:84` vs. `service.TagesEndeInSchulzeitzone` |
| 7 Gate-Ehrlichkeit      | Zwei `if (await …isVisible())`-Wächter — genau das Muster, das ein Gate still überspringen lässt.                                                                                                                                                                                                                                                                                                                                                | `frontend/e2e/admin-lusd.spec.js:51,132`                                                                            |
| 8 Lebenszyklus          | Drei Tabellen haben **keinen** Löschpfad und keine Frist: `lehrer_anliegen` (Freitext + Klasse, Bezug auf die Lehrkraft), `inventur_sessions`, `bestellungen_verlauf` (Lieferantenname/-adresse). Wachstum ist unkritisch; die Frage ist die Aufbewahrung.                                                                                                                                                                                       | `schema.sql`, kein `DELETE FROM` im Baum                                                                            |
| 9 Ausleitung            | `cmd/migrate-fotos` **sagt** „Du kannst das Verzeichnis `uploads/fotos` jetzt sicher löschen" — löscht es aber nicht. `/uploads/` ist bewusst ohne Anmeldung lesbar, und die Dateinamen waren die Barcode-IDs vom Schülerausweis, also vollständig aufzählbar. Der Code legt das Verzeichnis seit dem 08.08. nicht mehr an; ob auf dem Schulserver noch Altbestand liegt, sagt nur ein Blick ins Volume.                                         | `cmd/migrate-fotos/main.go:80`, `inventur/api_routen.go:72`, `api/router.go:110`                                    |

### Geprüft, kein Befund

Routen-Schutz (Gate greift, Allowlist begründet) · PII-Matrix (beidseitig, Recht gegen
Registrierung) · ignorierte Schreibfehler (`errcheck` mit `check-blank` schließt die Klasse
im ganzen Go-Baum) · „unbegrenzte Listen" (53 Kandidaten, alle durch reale Mengen begrenzt
oder mit `LIMIT`) · zerstörende Migrationen (20 Dateien, jede Spalte nachweislich tot) ·
`/uploads/` (Fotos liegen verschlüsselt in der DB, Verzeichnis bewusst nicht mehr angelegt)
· Frontend-`catch` (36 mit begründendem Kommentar, 7 stumm — alle um `scanner.stop()`
herum, ohne Wirkung) · übersprungene Tests (alle bedingt und begründet) · Rechte-Strings
(jetzt gegatet) · Etiketten-POST-Weg (Server füllt Signatur und Jahr selbst nach).

### Was der Durchgang über Gates gezeigt hat

Beide A-Funde waren **Paare, die nur zufällig einig waren** — nicht Fehler, die jemand
gemacht hat, sondern Übereinstimmungen, die niemand hielt. Das ist die Bauform, die sich
lohnt zu suchen: zwei Listen, zwei Schreibweisen, zwei Wege zu demselben Zustand. Von den
inzwischen sechs mechanischen Gates dieses Projekts (Routen-Schutz, PII-Matrix,
Einstellungs-Parität, Kategorien, Schema-Parität, Rechte-Parität) entstand jedes aus genau
so einem Paar.

**Nachgezogen am selben Tag** (Abschnitt „Nachgezogen" oben): Objektbindung, die drei
kleinen Pakete und die Svelte-Fläche. Ein weiterer Fund kam dabei heraus — die
Mail-Kopfzeilen. Die 206 Svelte-Dateien sind weiterhin nicht Zeile für Zeile gelesen,
sondern über fünf Detektoren gelaufen; das ist die ehrliche Auskunft über die Tiefe
dieser Fläche.

---

## Raster-Durchgang über den Einstellungs-Umbau (23.08.2026) — vier Funde, drei davon jetzt Gates

Peters Frage: „sollten wir Daniels Schema auf die heutigen Dinge nochmal anwenden — und
kann man das nicht bei der Programmierung direkt berücksichtigen?" Beides ja, mit einer
Einschränkung, die sich an diesem Durchgang gut zeigen lässt.

**Wann das Raster lohnt:** wenn sich die FORM eines Schreibpfads ändert — neuer Endpunkt,
neuer Rumpf, andere Speicher-Granularität. Genau das war heute der Fall. Bei Kosmetik
(die Sonar-Restliste vom selben Tag) ist es Leerlauf: Dort war nichts zu holen, und der
Durchgang hätte nur Zeit gekostet.

**Die Funde:**

| Rasterfrage                          | Fund                                                                                                                                                                                                                                                     | Erledigt als                                                                                                                                                                                    |
| ------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Stille Fehler                        | `PUT /api/einstellungen` mit leerem Rumpf antwortete **200 „ok"** und schrieb einen Audit-Eintrag `UPDATE_SETTINGS` — eine protokollierte Änderung, die nie stattfand, und einer kaputten Oberfläche die Bescheinigung, sie habe gespeichert.            | 400 + kein Audit-Eintrag; e2e am Draht, am alten Verhalten rot gesehen (200 statt 400).                                                                                                         |
| Spezialwert / Doppeljob              | Eine getippte **0** in „Tage / Buch" wurde serverseitig still zur Vorgabe 21. Vor heute egal, seit heute ein Wortbruch: Die Seite verspricht „was im Feld steht, wird gespeichert".                                                                      | `sammleZahlen` kennt je Feld ein `min` und meldet „Tage / Buch (mindestens 1)". Wo die 0 „aus" BEDEUTET (Lesehistorie, Sperrbildschirm, Tage bis Sperre), steht `min: 0`.                       |
| Konvention statt Regel               | Die ganze Zusicherung „eine Kategorie ändert nichts in den anderen" beruht darauf, dass jede der sieben Kategorien nur ihre eigenen Felder in den Patch legt — eine Verabredung zwischen sieben Dateien, die nichts erzwang. Der e2e prüft **ein** Paar. | `einstellungen-kategorien.test.js`: kein Feld in zwei Kategorien, kein Feld ohne Kategorie.                                                                                                     |
| Zwei Wahrheitsquellen                | Jeder Kategorietitel steht zweimal (Liste links, Überschrift der Detailfläche — und aus ihr entsteht die Knopfbeschriftung „<Name> speichern").                                                                                                          | Dritter Fall im selben Test.                                                                                                                                                                    |
| Blinde Hälfte (Feld ohne Nachfüller) | Kein Fund — Schreib- und Lesepfad sind deckungsgleich (22 Schlüssel). Aber nichts hielt das fest; so verschwand am 17.08. `alarm_empfaenger`.                                                                                                            | `system_settings_paritaet_test.go`: jeder schreibbare Schlüssel muss von `applyEinstellung` zurückgelesen werden, keiner doppelt. Ohne Datenbank, Millisekunden. Beide Fehlerarten rot gesehen. |

**Und die eigentliche Antwort auf „kann man das direkt berücksichtigen":** Von den acht
Rasterfragen sind drei **mechanisch prüfbar**, sobald man die Struktur kennt — Schreibpfad
gegen Lesepfad, Feld gegen Kategorie, Name gegen Name. Die sind ab heute Gates und stellen
sich von selbst. Die anderen fünf (Spezialwert, wer sieht was, stille Fehler,
Zeit/Reihenfolge, Lebenszyklus) sind **Urteilsfragen**: Ein Gate kann dort immer nur den
EINEN Fall festhalten, den jemand vorher gefunden hat. Genau so ist die Ratschen-Liste
dieses Projekts gewachsen — jede Ratsche ist ein Rasterfund von gestern.

**Ein Nebenergebnis über Detektoren:** Der erste Entwurf des Kategorien-Gates durchsuchte
die ganze Datei nach Schlüsselnamen. Die Gegenprobe („Schreibweg entfernen, Lesezugriff
stehen lassen") blieb **grün** — der Detektor konnte Lesen nicht von Schreiben
unterscheiden und behauptete damit mehr, als er prüfte. Jetzt liest er nur den
`speichereKategorie(…)`-Aufruf. Ein Gate ist erst fertig, wenn auch der DETEKTOR eine
Gegenprobe hinter sich hat, nicht nur der Code.

---

## Einstellungen als M3-Kategorienliste (23.08.2026) — und der Befund darunter

Peters Design-Prüfung der Einstellungsseite: „strukturell nein, optisch halb". Vier
Punkte, alle vier umgesetzt — der wichtigste war aber keiner von den vieren.

**Der Befund unter dem Design.** Der eine Speichern-Knopf am Seitenende war nicht nur
unordentlich, er war die URSACHE der drei Leer-Regeln, über die Peter gestolpert ist
(Schule: leer = unverändert · Öffentliche Adresse: leer = abschalten · Datenschutz:
0 = aus, leer = unverändert). Weil immer das ganze Formular auf einmal ging, brauchte
jede Sektion eine eigene Notbremse gegen das Überschreiben der anderen.

Und die Bremse war unvollständig. `SaveSettings` schrieb **elf Schlüssel bei jedem
Aufruf**, gebildet aus einem `SystemEinstellungen`-Struct, in dem ein nicht
mitgeschicktes Feld als `false`/`0`/`""` ankommt. Solange die Oberfläche immer alles
schickte, fiel das nicht auf. Ein Speichern je Kategorie hätte mit einem Klick in
„Datenschutz & Sitzung" den Ferien-Leseclub und die Bestellbedarf-Warnung
ausgeschaltet, die Preiserfassung abgeschaltet und fünf Fristen auf die Vorgabe
zurückgesetzt — **mit grüner Erfolgsmeldung**. Dieselbe Bugklasse wie das
Upsert-Blanking beim Import: Ein Schreibpfad, der „nicht mitgeschickt" nicht von
„leer" unterscheiden kann.

Deshalb ging der Umbau vom Backend aus: `repository.EinstellungenPatch` (alle Felder
Zeiger, nil = unangetastet) ist jetzt der Rumpf von `PUT /api/einstellungen`; die
Oberfläche schickt ausschließlich die Felder der Kategorie, die gerade gespeichert
wird. Damit fällt die Notbremse weg, und mit ihr die drei Regeln: **Was im Feld steht,
wird gespeichert** — ein geleertes Schulfeld wird auch geleert (vorher ließ sich ein
falscher Eigentumsvermerk nie wieder entfernen), ein leer geräumtes ZAHLENfeld ist
keine 0 und kein „lass es wie es war", sondern eine gemeldete Lücke.

| Peters Punkt                                                                    | Umgesetzt als                                                                                                                                                                                                                                                                                                                              |
| ------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| „Allgemein" ist ein Sammelbecken; Reiter sind für 3–5 gleichgewichtige Bereiche | Kategorienliste (Symbol, Titel, eine Zeile Beitext) + Detailfläche, ab `lg` zweispaltig, darunter list-detail mit Zurück-Pfeil. Zehn Kategorien statt sechs Reitern.                                                                                                                                                                       |
| Ein globaler Knopf für sieben fremde Sektionen                                  | Ein Speichern je Kategorie, beschriftet mit ihrem Namen. Schalter, die eine Liste steuern (Mahnwesen-Routing), speichern weiterhin je Zeile — deshalb hat diese Kategorie gar keinen Knopf.                                                                                                                                                |
| Unterstrich-Felder sind Material 2                                              | `SettingField` ist outlined (Rahmen `outline-variant`, im Fokus `primary`, Radius 4 px, Fläche weiß). Höhe bleibt 36 px wie jedes andere Bedienelement. Nebenbei: Der Hilfetext hing IM `<label>` — ein Screenreader las „Öffentliche Adresse Leer = keine Bestätigungs-Links verschicken" als NAMEN des Feldes; jetzt `aria-describedby`. |
| Zu viel Prosa                                                                   | Ein Satz Supporting Text je Kategorie, Details in einem nativen `<details>`-Aufklapper.                                                                                                                                                                                                                                                    |

**Zwei Abweichungen von der Vorschlagsliste, bewusst:**

- **Berechtigungen bleiben draußen.** Sie standen in der Frage-Runde als Kategorie —
  sie sind aber seit `66a58b06` (16.08.) ein eigener Menüpunkt, und die Drift-Warnung
  der Betriebsbereitschaft zeigt dorthin. Zurück in die Einstellungen wäre die
  Rücknahme einer Entscheidung von vor einer Woche und eine zweite Tür zum selben
  Bildschirm.
- **„Datenverwaltung" ist dazugekommen** (in der Liste vergessen) und
  **„Erreichbarkeit & Alarme" ist neu**: Öffentliche Adresse und Alarm-Empfänger
  tragen beide die Regel „leer = aus" und dürfen deshalb nicht bei den Schul-Stammdaten
  stehen, wo leer schlicht leer heißt. Eine Kategorie, eine Regel.

**Gates (alle am Rückbau rot gesehen):**

- `TestSettingsRoundtrip_KategorieSpeichernLaesstDenRestInRuhe` (echtes Postgres):
  voller Ausgangsstand → eine Kategorie speichern → die anderen sechs stehen noch.
  Gegen das alte Verhalten rot mit „Preise=false, frist_buch_tage 21 statt 28".
- `e2e/einstellungen-kategorien.spec.js`: dieselbe Aussage am DRAHT. Der Go-Test
  beginnt beim Patch-Objekt; ob die OBERFLÄCHE nur ihre eigenen Felder hineinlegt,
  sieht er nicht — und genau dort saß der Fehler. Rot gesehen mit einer Kategorie, die
  wie früher das ganze Formular schickt.
- Zweiter e2e: leer geräumtes Zahlenfeld speichert gar nichts und nennt die
  BESCHRIFTUNG des fehlenden Feldes, nicht den API-Schlüssel.

**Nebenbefund, kein Design:** Als Schulname steht auf dem Testserver
„Philipp-Reis-Schule.de" (lokal „Philipp-Reis-Schule, Friedrichsdorf"). Das ist ein
DATENwert, kein Code — er landet so auf jedem Buchetikett und als Kopfzeile des
Schülerausweises. Zu korrigieren in Einstellungen → Schule.

---

## SonarQube-Lauf 23.08.2026 — was der Scan sagt und was er nicht sagt

Erster Lauf auf dem neuen Server (26.7). Endstand nach der Abarbeitung: **0 Bugs,
0 Vulnerabilities, Ratings A/A/A, Coverage 58,3 %, Duplikate 0,5 %, 32 offene Smells,
Quality Gate OK** (`http://localhost:9000/dashboard?id=bibliothek`).

**Drei Messfehler, die wichtiger waren als die Funde:**

1. **Das Gate war eine Attrappe.** „Sonar way" prüft ausschließlich _neuen_ Code; die
   New-Code-Periode ist `PREVIOUS_VERSION`, und es gab keine vorherige Analyse. Ergebnis:
   null Bedingungen, Status OK — eine Aussage über nichts. Ab der zweiten Analyse greift es
   (und meldete sofort zwei echte Punkte).
2. **Security-Rating E hing an einer Kommentarzeile.** Zwei BLOCKER `secrets:S6698` in
   `scripts/pruefe_secrets.sh`: Das „Passwort" ist das Platzhalterwort `PASSWORT` in einem
   Erklärtext — ausgerechnet in dem Skript, das falsch konfigurierte Geheimnisse _findet_.
   Begründet ausgeschlossen (e4). **Falle dabei:** Die erste Fassung der Begründung zitierte
   die beanstandete Zeile wörtlich — der Detektor meldete daraufhin die Begründung. Wer
   einen Secrets-Fehlalarm dokumentiert, darf das Muster nicht abschreiben.
3. **Svelte ist im Scan gar nicht enthalten.** Sprachverteilung: `go=70.117`, `js=5.148`,
   `web=619`, `css=874`. Im Repo liegen **195 `.svelte`-Dateien mit rund 26.000 Zeilen** —
   SonarQube hat keinen Svelte-Parser. Was dieser Server über das Frontend aussagt, gilt für
   die 49 produktiven `.js`-Dateien, **nicht für die Komponenten**. Steht so auch in
   `sonar-project.properties`, damit die Zahl niemanden in Sicherheit wiegt.

**Behoben:** `klassifiziereZeileName` 8 → 7 Parameter (der achte stammte vom A2-Fix desselben
Tages; jetzt Modus statt Schlüssel+Flag, symmetrisch zu `bestandsSchluessel`) ·
`settingsWerte.js` `String(unknown)` → Typ-Zweige · `menue-fuehrt-irgendwohin.spec.js`:
Kommentar versprach „auf den Navigationszustand warten", darunter stand `waitForTimeout(600)`
— eine feste Frist, nach der ein langsam rendernder Bildschirm einen gesunden Menüpunkt als
tot gemeldet hätte (am Rückbau rot gesehen) · Frontend-Coverage (`@vitest/coverage-v8`, lcov,
an `sonar_scan.sh` gebunden mit Abbruch bei roten Tests) — schließt den offenen Punkt oben.

**Restliste abgearbeitet (23.08. abends, 9 Funde — reine Lesbarkeit, kein Verhalten):**
`go:S1192` 3× — die Selbstprüfung sagte vier Bereichen wörtlich „in <env> ist das richtig
so." und dreien „Datenbankverbindung prüfen und die Seite neu laden."; jetzt
`befundNichtImEchtbetrieb()` + `abhilfeDbNeuLaden`, damit die Seite in EINEM Tonfall spricht.
Dritter: `apierrors.Internal("Sachgruppe konnte nicht geändert werden")` an Begin/Update/Commit
→ `fehlerSachgruppeAendern` (für den Benutzer ist es derselbe Vorgang, der nicht stattfand). ·
`go:S107` 2× — `processSingleBatchItem` reichte acht Werte durch die Stapelschleife, darunter
**zwei gleichartige `bool`** (`darfGrundSehen`/`darfOverride`), bei denen ein Dreher stumm
den Sperrgrund preisgäbe → `batchKontext`; `AnliegenRepository.Create` nahm **sieben Strings
in Reihe** (Klasse und ISBN vertauschbar ohne Compilerwort) → `NeuesAnliegen`; der
PG-Test las die beiden Felder bisher gar nicht zurück und hätte den Dreher nicht gesehen —
jetzt nagelt er die Spaltenzuordnung fest (am Rückbau rot gesehen: ISBN landete als
Klasse). · JS-Stil 4× —
`toHaveLength` statt `.length).toBe` (mit Meldung), ein `push()` statt zwei, `String.raw`
statt `'\\$&'` (Äquivalenz am Regex-Source geprüft), `replaceAll` statt `split().join()`.
Gates: `go build`/`go test ./...`, `golangci-lint` 0, eslint --max-warnings 0, svelte-check
0/0, vitest 213, `npm run build`. Damit stehen noch 23 Smells offen — die 20 `go:S3776`
und die drei begründeten JS-Ausnahmen unten.

**Bewusst offen:**

| Fund                                                       | Warum offen                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| ---------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 20× `go:S3776` (Cognitive Complexity > 15)                 | Alle im Go-Backend, **null im Frontend** — die Ratsche vom 13.07. hält. Mehrere Handler stehen seit Mai/Juni über der Schwelle; Grund ist die Zählweise (ein `http.HandlerFunc`-Closure ist eine zusätzliche Verschachtelungsebene, siehe Projektgedächtnis). Ein Aufsplitten nur für die Zahl macht die Handler nicht lesbarer. Lohnend sind allenfalls die drei größten: `OverrideDueDateHandler` (30), `behandleAbgaenger` (23), `ErledigeAnliegenHandler`/`PatchStudentHandler` (21). |
| 1× `javascript:S2925` (`kontrast.spec.js:44`)              | 700 ms Setzzeit vor der Kontrastmessung. Eine beobachtbare Bedingung gäbe es nur je Bildschirm (Selektor-Tabelle) — ehrlicher wäre eine Stabilitäts-Schleife wie in den Schwester-Specs. Eigene kleine Runde.                                                                                                                                                                                                                                                                             |
| 1× `javascript:S6551` (`escapeHtml.js`)                    | `String(wert)` ist dort genau die Absicht: Der Helfer soll jeden Wert sicher machen, auch einen versehentlich übergebenen.                                                                                                                                                                                                                                                                                                                                                                |
| 1× `javascript:S8783` (`schuelerprofil-sperre.spec.js:75`) | `hover({force:true})` auf einem **absichtlich deaktivierten** Knopf — ohne `force` wartet Playwright ewig auf Aktionierbarkeit. Genau der Testzweck (Tooltip am gesperrten Knopf).                                                                                                                                                                                                                                                                                                        |
| `sonar.projectVersion` nicht gesetzt                       | Die New-Code-Periode `PREVIOUS_VERSION` hat damit keinen Bezugspunkt; „neuer Code" heißt derzeit „seit dem letzten Scan". Mit `--define sonar.projectVersion=$(git describe --tags)` hieße es „seit dem letzten Release" — passt zum Release-Workflow, aber ändert die Gate-Semantik. Entscheidung offen.                                                                                                                                                                                 |

---

## Raster-Durchgang über die Etiketten-Commits (24.08.2026) — zwei Funde, eine neue Frage

Auf Peters Ansage über die 15 Commits des Tages (Ausweisdruck-Fixes, Klebebogen statt
A4, Rückbau). Der Anlass erfüllt die Regel „nur bei Formwechsel eines Schreibpfads":
Das zentral gespeicherte Design hat ein neues Vokabular (`printMode: card|etikett`,
`etikettFormat`) und mit dem Etiketten-PDF eine neue Ausleitung.

### Die Funde (beide sofort, je ein Commit, je rot gesehen)

**1. Wer lädt den geteilten Zustand? (`3fd1a5a5`)** Der Schülerdatei-Druckpfad
ENTSCHEIDET am zentralen `printMode`, geladen hat ihn dort aber nur
`StudentBatchPrint` — die Komponente, die das Ergebnis des eigenen Ladens abräumt
(`{#if !etikettModus}`). Ging nur zufällig gut; die E2E-Spec sah nichts, weil sie den
Designer zuerst besucht (Grün aus Umgebungsgunst). Beim Fixen die Kehrseite gleich
mitgefunden: Nachladen bei jedem Mount überschreibt frische, noch nicht fertig
gespeicherte Designer-Änderungen mit dem alten Serverstand → Sitzungs-Merker, geladen
wird beim ersten Bedarf.

**2. Der Auto-Save verwarf die letzte Änderung beim Verlassen (`37abcbe0`).** Der
Effekt-Abbau löscht den Entprell-Timer — beim Tippen ist das die Entprellung, beim
Verlassen des Designers ging die letzte Änderung verloren („Speichert…" stand noch
da). Vorbestand seit Bau des Designers; seit dem Umschalter verliert derselbe Abriss
die Betriebsart des Stapeldrucks. `onDestroy` schickt Ausstehendes jetzt sofort ab.
Restlücke benannt: Browser-Schließen binnen 800 ms verliert weiterhin (sendBeacon
nicht gebaut, Randlage).

### Geprüft, kein Befund

Payload↔Struct beidseitig (Request und Design-Blob gepaart) · Spezialwerte
(Startposition 0→1, leere Klasse lässt Zeile weg, `'a4'`-Altwert wird beim Lesen
migriert und beim nächsten Speichern getilgt) · privilegiertes Feld (`muster`
harmlos, IDs statt Namen ist gerade die Absicherung) · stille Fehler (Servertext
erreicht die Theke, Popup-Block gemeldet, Barcode-Ausfall je Etikett verkraftet) ·
Zeit (keine Datumsangaben auf dem Etikett) · Lebenszyklus (PDF wird gestreamt, nie
serverseitig abgelegt; Gelöschte ausgeschlossen, PG-getestet) · Ausleitung (Stufe 1,
Matrix-Zeile, Grenze 600, keine PII in URL/Dateinamen) · Rückweg (Design-Blob
round-trip-stabil). Kleinigkeit notiert, nicht angefasst: die `LabelHeight >= 30`-
Schwelle steht jetzt 6× in zwei Dateien (Bauform der Datei, C-Klasse).

### Die neue Frage — Vorschlag als Nr. 11

**Geteilter Zustand: Wer lädt ihn — und überlebt der Lader das Ergebnis?** Für jeden
Bildschirm, der zentralen Zustand LIEST: Welcher Code lädt ihn auf diesem Pfad, was
passiert im Fenster davor, was bei Fehlschlag — und hängt der Lader an einer
Bedingung, die sein eigenes Ergebnis umlegt? Die zehn bisherigen Fragen sehen auf
Schreibpfade und Ausgänge; diese sieht auf die Ankunft. Beide heutigen Funde und die
Ausweis-Design-Persistenz-Historie (localStorage-Sackgasse, Musterstadt-Kopf) sind
dieselbe Stelle. Wie 9 und 10 ein Urteil, kein Gate.

Stand: 2026-08-24

## Raster-Durchgang über die Löschwege (23./24.08.2026) — fünf Funde, einer zurückgenommen

Peters Auftrag: „prüfe doch alles am Code … mach es gleich richtig!" Fläche: die neun
Commits vom 23.08. (Löschrouten-Wächter, Verlust-Löschen, Titel-Löschen), davon acht
Produktivdateien in `api/`, `inventur/`, `jobs/`, `repository/`.

Anlass war ein Fund, der schon beim Zusammenstellen der Fragen auffiel — deshalb steht er
mit in dieser Liste, obwohl er vor dem Durchgang behoben wurde.

### Kategorie A — behoben, je ein Commit, Gate am Rückbau rot gesehen

| Frage                              | Fund                                                                                                                                                                                                                                                                                                                                                                                                                                          | Erledigt als                                                                                                                                                                                 |
| ---------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **9 Ausleitung**                   | Die Protokollspur für gelöschte verliehene Bücher trug den KLARNAMEN des Entleihers, aber keine `schueler_id`. Das Prädikat der Lesehistorie-Befristung verlangt `details ? 'schueler_id'` — die Zeile fiel durch und wäre erst nach der Audit-Aufbewahrung verschwunden: 24 Monate statt 90 Tage. Sie war damit auch für die Art.-15-Auskunft unsichtbar (die liest `details->>'schueler_id'`).                                              | `a7bdeb46` — Spur schreibt `schueler_id`, Job tilgt zusätzlich `entleiher`. Zwei Gates, beide mit Gegenprobe am Detektor.                                                                    |
| **1 Konvention / 2 Spezialwert**   | Prädikat und Zahlen waren nur per Konvention gepaart. Fünf der sieben Bedingungen überstehen ein Vertauschen folgenlos — aber nur, weil Addition kommutativ ist. Die Audit-Aufbewahrung ist nicht symmetrisch: `months=>24,days=>0` → Grenze 2024-08-23, vertauscht `months=>0,days=>24` → 2026-07-30. Das ganze Prüfprotokoll bis auf 24 Tage, nächtlich und unumkehrbar.                                                                    | `659b3b5b` — `Loeschbedingung{Where, Args}`; die Zahl kann nur noch dorthin, wo ihr Name steht. Dazu ein Gate auf die EINHEIT (Postgres rechnet die Grenze aus, Vergleich mit dem Kalender). |
| **6 Zeit**                         | `AbgaengerStichjahr` nahm das Jahr aus der lokalen Zeit und verglich mit einem in UTC gebauten 30. Januar. Unter Europe/Berlin lieferte der 30.01. um 00:30 das Vorjahr. Richtung harmlos, aber das Ergebnis hing an der Container-Zeitzone statt am Schulkalender.                                                                                                                                                                           | `f2f1ae22` — Rechnung in `pkg/schulzeit`. Gate prüft die Invariante: derselbe Zeitpunkt, vier Zeitzonen, ein Ergebnis.                                                                       |
| **1 Konvention / 4 wer sieht was** | `audit_aufbewahrung_monate` war NIRGENDS setzbar — nicht im Frontend, nicht im Patch-Vokabular, nicht im Seed, in keiner Migration. Der Dateikopf beschrieb eine einstellbare Frist samt „Untergrenze 6"; real war sie fest verdrahtet, und die Selbstprüfung meldete „Frist 24 Monate", als wäre sie konfiguriert. Die drei Schwester-Fristen SIND einstellbar.                                                                              | `3aea221d` — zu Ende gebaut (Struct, Patch, Kategorie, Untergrenze 6 sichtbar statt still). Zweite Wahrheitsquelle (roher SQL-Leser) beseitigt. Zwei neue Gates, siehe unten.                |
| **5 stille Fehler**                | Der Fehlbestand meldete `${geloescht} Exemplare endgültig gelöscht` als **success** — auch bei 0. Schlimmer: Die Liste wurde lokal um ALLE ANGEFRAGTEN IDs bereinigt, unabhängig davon, was der Server entfernt hat. Wer fünf auswählte, von denen zwei nicht mehr als Verlust gebucht waren, sah alle fünf verschwinden — zwei nur auf seinem Bildschirm. Der Teilfall war im PG-Test schon abgedeckt; nur die Rückmeldung daran war falsch. | `4ff5958a` — Endpunkt liefert die tatsächlich gelöschten IDs, Oberfläche entfernt genau diese und meldet den Unterschied.                                                                    |

### Zurückgenommen — der Befund stimmt, der Fix war eine Kompetenzüberschreitung

**10 Rückweg:** `/api/audit` liefert die Spalte `details` nicht aus. Übrig bleibt „DELETE
auf ausleihen, Datensatz \<UUID\>" — und diese UUID gehört zu einem Exemplar, das im
selben Vorgang gelöscht wurde. Die Protokollspur für ein gelöschtes verliehenes Buch ist
über die Anwendung **nicht lesbar**; wer das zurückgebrachte Buch auf dem Tresen hat,
findet niemanden. Der Rückweg existiert im Datensatz und ist unbegehbar.

Der Fix (`3213ac1e`) wurde mit `9812af4d` zurückgenommen: `docs/PII_MATRIX.de.md` stuft
die Route ausdrücklich als **Stufe 0 — „Audit ohne details; IDs statt Namen"** ein. Das
Fehlen der Spalte war keine Lücke, sondern eine dokumentierte Datenminimierung, auf die
der VVT-Entwurf verweist. Mit `details` kann die Antwort Klarnamen tragen. Das ist eine
Entscheidung des Betreibers, keine Fehlerbehebung.

**Offen, zur Entscheidung:** Statt die Klasse der ganzen Route zu heben — ein
zweckgebundener Weg für den einen Fall, der ihn braucht („Buch liegt auf dem Tresen,
Barcode unbekannt: war er mal vergeben?"), mit eigener Einstufung und eigenem Recht.

### Was der Durchgang über die Gates gezeigt hat

- Das PII-Gate blieb bei der Reklassifizierung **grün, und zwar zu Recht**: Es prüft
  Route, Recht und DASS eine Zeile existiert — ausdrücklich nicht, was die Antwort
  wirklich enthält („von Hand am Handler zu verifizieren"). Die Einstufung ist damit eine
  Zusage, die nur ein Dokument behauptet. Bleibt als offener Punkt.
- Das Einstellungs-Paritätsgate sieht Oberfläche ↔ Patch-Struct, aber **nicht**, was der
  Server liest. Genau dort saß der Waisen-Schlüssel. Zwei neue Gates schließen das:
  gelesene Schlüssel müssen einstellbar sein, und neue rohe Leser von
  `system_einstellungen` sind eingefroren (fünf begründete Ausnahmen, die sich selbst
  prüfen — der erste Entwurf der Liste enthielt drei erfundene Schlüssel).
- Der Wächter aus `d3188b2a` hätte den Ausleitungs-Fund NIE gemeldet: Er stellt per
  Konstruktion dieselbe Frage wie der Job. Eine geteilte Wahrheitsquelle schützt vor
  Auseinanderlaufen, nicht vor einer Lücke, die BEIDE haben.

### Geprüft, kein Befund

| Frage | Fläche                          | Ergebnis                                                                                                                                                             |
| ----- | ------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 3     | Audit-Frist, Abgänger-Stichjahr | je genau eine Quelle (nach dem Fix oben)                                                                                                                             |
| 4     | `audit_logs`-Recht              | in der Vorgabe ADMIN-only, die drei anderen Rollen `false`                                                                                                           |
| 4     | Alarm-Mail                      | Befund-Texte tragen nur Routinenname, Zahl und Frist — keine PII                                                                                                     |
| 7     | CI                              | setzt `TEST_DATABASE_URL`, läuft über `./...` und **bricht ab, wenn ein Test wegen fehlender Umgebung übersprungen wurde** — die fünf neuen PG-Gates laufen wirklich |
| 8     | `inventur_verluste`             | nur bibliografische Daten, kein Personenbezug; verwaiste Zeilen (`exemplar_id NULL`) bleiben lesbar, das Frontend filtert sie aus „offen" und gated den Knopf        |
| 8     | neue Protokollspur              | liegt in `audit_log` → Aufbewahrung (jetzt einstellbar), PII nach 90 Tagen getilgt                                                                                   |
| 4     | Art.-15-Auskunft                | liest `tabelle='ausleihen' AND details->>'schueler_id'` — die Spur ist seit `a7bdeb46` auch dort sichtbar                                                            |

## Prüfung 22.08.2026 (Daniel-Raster über alle Commits seit 21.08.) — Stand nach der Abarbeitung

Sechs unabhängige, nur lesende Durchgänge nach dem abstrahierten Raster des externen
DB-Prüfberichts (Konvention statt Regel · Spezialwert/Doppeljob · zwei Wahrheitsquellen ·
wer sieht was · stille Fehler · Zeit/Reihenfolge · Gate-Ehrlichkeit · Lebenszyklus) über
die ~70 Commits vom 21./22.08. Jeder Fund unten ist **am Code verifiziert** (Datei:Zeile),
die HOCH-Funde zusätzlich am Live-Pfad bzw. per Probe gegen echtes Postgres.

**Abarbeitung 22.08. abends (je Fund ein Commit, Gate am alten Code rot gesehen):**
A1/A2 `bdb48ca7` · A3 `6d01f27a` · A4 `b5b50a2e` · A5 `4cf559cc` · A6 `729a7271` · A7 `f7e39361` ·
A8 `9fa3eae4` · B-Betrieb `5a55147f` (restore-backup im Image, S3-Durchreichung, Probe-Startlauf,
Stderr-Scrub, scrypt-Texte, Doku) · B-DB `8c9c8042` (Migration 085, conname-Parität,
Selbstprüfung „DSGVO-Löschroutinen", Restore-409, Append-only-Ratsche) · B-Rest `63d09011`
(Art.-15-Angaben aus den Fristen, LUSD-Import-Audit, PATCH-Geburtsdatum, Release-Wächter +
Tag-Ruleset, actionlint, trivy-Pin, Jules-Nacharbeit, Doku) · LUSD-Modus-Wechsel `968ee01a`.

**Bewusst offen (gelistet, entschieden „nicht jetzt"):**

- Paritätstest vergleicht weiterhin keine COMMENTs/Seeds (nur Kosmetik; 085 deckt die
  strukturellen Lücken) — Prod-Check der vier Indexe nach dem Deploy: `SELECT indexname FROM
pg_indexes WHERE indexname IN ('idx_schueler_deleted_at','idx_ausleihen_rueckgabe_am')`.
- 082-Dedupe der Vormerkungen (neuerer `abholbereit` verloren) — auf Prod bereits gelaufen,
  nicht rückholbar; nur relevant, falls eine weitere gewachsene DB hinzukommt.
- Lernmittel-/Schülerbücherei-Frist hängt an der `LMF-`-Namenskonvention; kein
  `ist_lernmittel`-Flag (Produktentscheidung, siehe LMF-Memory).
- LUSD: Namensschlüssel nur `lower+trim` (Umlaut-/Bindestrich-Varianten gelten als
  verschiedene Menschen → „mehrdeutig"/neu) — sicher, aber nicht klug; ID-Modus mit gemischter
  Datei behandelt ID-lose Zeilen weiterhin nur als Zähler.
- Jules: sieben Testdateien > 200 Zeilen; Export-CSV-„breaks stream"-Test bleibt schwach.
- Release: Go/Node-Versionen sind jetzt dokumentiert, aber Dependabot kann `golang:`/`node:`
  im Dockerfile weiter unabhängig heben (kein `ignore`).
- Altbackups vor 21.08. sind unlesbar; Re-Encrypt-Tool bewusst nicht gebaut (Pilot, keine
  schützenswerten Altbestände) — dafür Betreiber-Hinweis in resilience_and_recovery.md.

### Kategorie A — kann still jemandem schaden (ALLE ERLEDIGT 22.08.)

| Fund                                                                                                                                                                                                                                                                                                       | Nachweis                                                                                                                                                                     | Fix-Idee                                                                                                                                                                                    |
| ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **LUSD Nur-Name: bestätigter Bestandsschüler, dessen Name ZWEIMAL in der Datei steht, wird Abgänger und anonymisiert.** Der „Mehrdeutig“-Zweig setzt `gesehen` nicht; `sammleAbgaenger` hält ihn für „nicht im Export“. Vorschau zeigt ihn unter „Mehrdeutig (wird nicht angefasst)“ UND unter „Abgänger“. | `api/lusd_klassifizierung.go:59-63` + `:155-163`; Probe gegen PG: 1/13 → Schwelle greift nicht, Apply ohne Bestätigung, danach `ist_abgaenger=true, vorname='Abgänger'`      | im Default-Zweig `gesehen[rec.namensschluessel()] = true` vor `continue`; Regressionstest mit Bestandstreffer + zwei Dateizeilen                                                            |
| **LUSD Nur-Name: gesperrter Abgänger wird von Namensvetter „reaktiviert“** — Fünftklässler landet auf dem Datensatz (Schulden, Sperre, Lesehistorie) eines anderen Kindes.                                                                                                                                 | `api/lusd_klassifizierung.go:136-143` (`idx.abgaenger[key]` nur über Name)                                                                                                   | im Nur-Name-Modus Abgänger-Treffer als „Rückkehrer-Kandidat“ melden statt zuordnen                                                                                                          |
| **LUSD-Abgänger-Anonymisierung setzt kein `anonymized_at`; Purge läuft VOR der Cron-Tilgung und tilgt `audit_logs` nie** → LUSD-ID (LUSD_ID_NACHGETRAGEN) überlebt 24 Monate in `audit_logs.details`; Vormerkungen bleiben bis zum Purge.                                                                  | `api/lusd_apply.go:319-349` (kein `anonymized_at`), `repository/audit_users.go:168-203` (nur `audit_log`), `jobs/cron.go:43-45` (Reihenfolge), Purge-Cutoff 30.01. vs. 360 d | `anonymisiereAbgaenger` setzt `anonymized_at` + `ANON-`-Barcode; Purge tilgt `audit_logs` vor dem DELETE; PG-Test für den Purge-Pfad                                                        |
| **Leeres Zahlenfeld in „Datenschutz & Sitzung“ schaltet Befristung/Sperre still auf 0 = aus.** `bind:value` liefert `null`, `Number(null) \|\| 0` → 0; Backend nimmt 0.                                                                                                                                    | `frontend/src/lib/SystemSettingsAllgemein.svelte:86-89`, `SettingField.svelte:42`, `repository/system_settings_datenschutz.go:49-57`                                         | „Aus“ als eigener Schalter + Zahl ≥ 1; leer ⇒ Vorgabe; Selbstprüfung meldet „Befristung aus“                                                                                                |
| **Lesehistorie lebt im `audit_log` weiter** (CHECKOUT/RETURN mit `details.schueler_id`, 24 Monate) — Art.-13/VVT sagen „Zuordnung automatisch entfernt“; Art.-15-Auskunft liest diese Einträge nicht einmal.                                                                                               | `repository/audit_books.go` (`details["schueler_id"]`), `jobs/cron_dsgvo_lesehistorie.go` fasst nur `ausleihen` an, `api/dsgvo_auskunft.go:255` nur `tabelle='schueler'`     | Job tilgt `details - 'schueler_id'` für `tabelle='ausleihen'` nach derselben Frist; Auskunft um Ausleih-Einträge ergänzen                                                                   |
| **Sperrbildschirm: Druckvorschau (Strg+P) zeigt die Seite dahinter** (`no-print` am Overlay), **kein Fokus-Fang/`inert`** (Tab verlässt die Sperre, Screenreader liest dahinter), **Kamera-Scanner bucht weiter**.                                                                                         | `Sperrbildschirm.svelte:33`, `styles/druck-grundlagen.css:38-42`, `App.svelte` rendert App unter dem Overlay, `idleLock.svelte.js` fasst `showCamera` nicht an               | App-Fläche bei Sperre nicht rendern (löst Druck + Fokus + SR zusammen); `thekeLeeren()` stoppt Kamera                                                                                       |
| **Login-Handler-Kontext 10 s < IMAP-Frist 15 s** → korrektes, langsames Login scheitert am DB-Lookup, zählt als Fehlversuch (401 + Sperre). Umgekehrt: ≥ 15 s-Tarpit des Mailservers macht jedes falsche Passwort zum 503 ohne Zählung.                                                                    | `auth/handlers.go:127`, `auth/imap.go:192,215-216`, `:264`, `selbstanmeldung.go:118-123`                                                                                     | Handler-ctx an `AuthenticateIMAP` durchreichen, EINE Frist; Klassifikation aus der IMAP-Antwort (NO = Passwort), nicht aus der Zeit; Test mit Mini-IMAP-Listener (sofort NO / verzögert NO) |
| **Bulk-Mahnmail: SMTP-Hänger je Klasse bis 70 s, Ausfälle zählen als „übersprungen (keine E-Mail hinterlegt)“**, End-Audit mit totem `r.Context()` schweigt.                                                                                                                                               | `api/mail_sender.go:84` (kein ctx), `api/mahnwesen_bulk_mail.go:288,324-327,128,374`                                                                                         | „fehlgeschlagen“ getrennt zählen; Audit mit `context.Background()`+Frist; nach erstem Versandfehler abbrechen                                                                               |

### Kategorie B — Typografie unter der M3-Skala (25.08.2026, gemessen)

Im Browser gemessen über 31 Ansichten: Tabellen-Nebentext (Klasse, Barcode, Datum, Status)
liegt in 8 Tabellen auf 12 px statt M3 body-medium 14 (~10.000 Zellen); Kennungen auf 11 px;
`Button size="sm"` = 12 px (M3 kennt keinen); StatusChip 11 px; 25 Stellen mit Gewicht 800/900;
Reiter in drei Höhen. Vollständiger Bericht mit Reihenfolge und Gate-Vorschlag:
[`m3_typografie_audit_2026-08-25.md`](m3_typografie_audit_2026-08-25.md).
Bewusst NICHT Befund: 36-px-Controls, dichte Tabellen.

### Kategorie B — laut oder ohne Außenwirkung (erledigt bis auf die oben gelisteten Punkte)

| Fund                                                                                                                                                                                                                                                                                                                                                                             | Nachweis                                                                                                                                                                                              |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `restore-backup` liegt **nicht** im Image, Doku sagt „liegt im Image“ — im Ernstfall auf dem Schulserver ohne Go nicht baubar                                                                                                                                                                                                                                                    | `Dockerfile:67,103-107`, `docs/resilience_and_recovery.md:76`                                                                                                                                         |
| Selbstprüfung rät „S3_* in der .env setzen“ — Compose reicht `S3_*` nicht durch (gleiche Klasse wie BACKUP_ENCRYPTION_KEY)                                                                                                                                                                                                                                                       | `api/betriebsbereitschaft.go:380`, `docker-compose.yml` ohne `S3_`, `.env.example:122-126`                                                                                                            |
| Altbackups (SHA-256) seit 5265698c **unlesbar**, bis 02:30 des Folgetags gibt es kein lesbares Backup; Doku widerspricht sich (sed-Reparatur für Dateien, die nicht mehr entschlüsselbar sind), `backup.go:297-299` behauptet „alte Backups bleiben lesbar“                                                                                                                      | `jobs/backup_krypto.go:94-97`, `docs/resilience_and_recovery.md:20-23` vs. `:95-100`                                                                                                                  |
| IMAP `Logout()` ohne Frist (go-imap `Timeout=0`) — schweigender Server hängt den Login-Handler                                                                                                                                                                                                                                                                                   | `auth/imap.go:202-208`                                                                                                                                                                                |
| Restore-Probe: „per Neustart erneut proben“ stimmt nicht (nur So 03:30) → bis 7 Tage „kritisch“ nach Behebung; psql-Stderr mit Datenkontext (`COPY schueler, line …`) landet in DB/Seite/Alarm-Mail                                                                                                                                                                              | `api/betriebsbereitschaft.go:185-186`, `jobs/cron.go:116`, `restore_probe_hilfen.go:87`                                                                                                               |
| Veraltete SHA-256-Texte nach scrypt-Umstellung                                                                                                                                                                                                                                                                                                                                   | `betriebsbereitschaft.go:238`, `backup_status.go:30`, `cron.go:75-77`, `backup.go:297-299`                                                                                                            |
| Art.-15-Auskunft nennt pauschal lit. e, Speicherdauer ohne 90/730 Tage, keine Einwilligung für die Schülerbücherei — zwei Wahrheitsquellen zu SECURITY.md/VVT                                                                                                                                                                                                                    | `api/dsgvo_auskunft.go:121-129`                                                                                                                                                                       |
| VVT sagt „Protokoll ohne IP-Adresse“ — `audit_logs.ip_adresse` wird geschrieben                                                                                                                                                                                                                                                                                                  | `repository/audit.go:132`, `api/littera_import.go`, `api/mahnwesen_bulk_mail.go:371`                                                                                                                  |
| Lernmittel-/Schülerbücherei-Frist hängt an der `LMF-`-Namenskonvention; keine Gegenprobe „Lernmittel-Fach ohne Kennung“                                                                                                                                                                                                                                                          | `jobs/cron_dsgvo_lesehistorie.go:71-75`, `pkg/lmf/lmf.go:43-45`                                                                                                                                       |
| DSGVO-Jobs (Anonymisierung, Lesehistorie) scheitern still (nur Log) — exakt die Klasse, die zweimal monatelang ins Leere lief; Selbstprüfung hat keinen DSGVO-Bereich                                                                                                                                                                                                            | `jobs/cron_dsgvo.go:102,112,159,172,183`, `api/betriebsbereitschaft.go`                                                                                                                               |
| Paritäts-Ratsche sieht nur die älteste Baseline: DBs aus schema.sql vom 14.06.–21.08. fehlen `idx_ausleihen_ausgeliehen_am/rueckgabe_am`, `idx_buecher_titel_erstellt_am`, `idx_schueler_deleted_at` (082 heilt bewusst nur die Gegenrichtung); Test vergleicht keine Constraint-NAMEN (Kommentar behauptet es), keine COMMENTs (084-Kommentar fehlt in schema.sql), keine Seeds | `migrations/082:14-15`, `db/migrations_schema_paritaet_pg_test.go:33-41`; Prod prüfen: `SELECT indexname FROM pg_indexes WHERE indexname IN ('idx_schueler_deleted_at','idx_ausleihen_rueckgabe_am')` |
| Papierkorb-Restore stellt anonymisierte Zeilen wieder her (gesperrt, Name „Anonym“); 082-Dedupe der Vormerkungen löscht den NEUEREN Eintrag auch wenn er `abholbereit` ist                                                                                                                                                                                                       | `api/student_deleted.go:86-90`, `migrations/082`                                                                                                                                                      |
| Append-only auf `audit_log(s)` ist seit 083 reine Konvention ohne Ratsche (App läuft als `postgres`)                                                                                                                                                                                                                                                                             | `docker-compose.yml:101`, Schreibtüren `audit_users.go:176`, `cron_dsgvo.go:154/168`, `cron_audit_retention.go:37/43`                                                                                 |
| LUSD: Modus-Wechsel Nur-Name → Name+Geburtsdatum dupliziert den Bestand (Bestand ohne Datum nur in `ohneSchluessel`; Datum wird nie nachgetragen); ID-Modus mit gemischter Datei macht ID-lose Zeile zum Abgänger; kein Audit-Eintrag für den Import; PATCH blankt das Pflicht-Geburtsdatum; Parser loggt Namen bei 400                                                          | `api/lusd_bestand.go:75-76`, `lusd_apply.go:155-166`, `lusd_klassifizierung.go:87-90`, `student_update.go:145-150`, `lusd_parser.go:204`                                                              |
| Mahnketten-TZ-Fix ist Test-Reparatur, aber am DST-Ende (24.10., 22–23 UTC) wieder rot: `time.Now().AddDate` in Runner-TZ statt Schulzeitzone                                                                                                                                                                                                                                     | `api/mahnwesen_kette_pg_test.go:153` vs. `internal/service/loan_rules.go:162`                                                                                                                         |
| Release: v-Tag ist ungeschützter Kanal (jeder Push-Collaborator; kein main-/CI-Check), `latest` hat zwei Schreiber (Doku behauptet einen), Release-Notes versprechen ein Image, das ein zweiter Workflow erst baut; ghcr-Image ist öffentlich (Kommentar sagt privat); kein actionlint; `trivy-action@master`; Go-Version 1.26.5 in docs vs. 1.26.6; Node 22 (Image) vs. 24 (CI) | `.github/workflows/release.yml:16-18,49,54`, `docker-publish.yml:13,17-25`, `security-scan.yml:169`, `docs/README.md:11,67`, `ci.yml:157,195`                                                         |
| update.sh 4b: leerer GIT_COMMIT = nur Warnung                                                                                                                                                                                                                                                                                                                                    | `update.sh:275-279`                                                                                                                                                                                   |
| Jules-Batches: `TestMigriereFoto_VerschluesselungFehlgeschlagen` belegt seine Behauptung nicht; Dubletten (`TestKuerze`, `TestSammleSignaturUpdatesDynamic`, DeleteBooks-Mock); 7 Testdateien > 200 Zeilen; Nil-Deref im Fehlerzweig (`schreiber_bestand_test.go:113-124`); DeleteBooks-Test schreibt `inventur/uploads/` ins Repo                                               | `cmd/migrate-fotos/main_test.go:174-193` u. a.; Cruft: keiner, `go test -race -count=3` + `TZ=Pacific/Midway` grün                                                                                    |

### Geprüft, kein Befund (Auszug)

SMTP-Frist deckt alle Phasen (Hänger-Test echt, am Rückbau rot); keine Geheimnisse in argv/Log (PGPASSFILE); scrypt 32 MB ohne mem_limit; Backup-Alter eine Quelle; Cron-Zeiten UTC ohne Überlappung; Restore-Probe gegen echtes pg_dump/psql; Dockerfile-Pin 16 ≥ Prod 15; 080–084 idempotent, Migrationslauf aus sechs Baselines bricht nirgends; `ANON-<uuid>` kollisionsfrei; LUSD-Apply atomar mit Advisory-Lock, Upsert-Blanking gepaart, Handanlagen nie Abgänger; Dependabot-PRs laufen durch ci+security-scan; 3b80f7f1 `npm ci` konsistent; Jules-Merges ohne Cruft, nur zwei bewusste Produktions-Nähte.

---

## Erledigt (2026-08-06) — Etiketten nach Rückmeldung Naacher

Zwei Wünsche aus einem Telefonat, beide am fertigen PDF abgesichert:

| Fund                                 | Was es behauptete           | Was stimmte                                                                                                                                                                             |
| ------------------------------------ | --------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Große Lernmittel-Etiketten           | „ein Etikett je Seite (A6)" | Auf einem A4-Drucker kam damit **ein** Etikett je Blatt heraus. Vier passen ohne jede Skalierung darauf — A6 ist exakt ein Viertel von A4. Jetzt 2×2 mit Schnittlinien.                 |
| Kleine Etiketten im Lieferanten-Link | „Bogen wie im Druck-Center" | Der Weg hatte `"zweckform_l4760"` **fest verdrahtet**, während das Druck-Center längst drei Raster anbietet. Wer andere Bögen im Drucker hatte, bekam einen Ausdruck, der danebenliegt. |

**Der Fund kam aus der Rückbau-Probe, nicht aus dem Code.** Der erste Test zur
Formatwahl rief den Generator direkt auf und blieb grün, als der Lieferanten-Weg
probeweise wieder auf das feste Raster gesetzt wurde — er sah die Durchreichung gar
nicht. Erst ein Test über den echten Endpunkt (`bestellbestaetigung_format_pg_test.go`)
wurde rot. Genau die Bugklasse dieses Projekts: ein Gate, das eine Stelle prüft, die
niemand kaputtmacht.

Dazu ein drittes Fundstück am Rand: Die Rasterdaten stehen an **vier** Stellen
(`api/label_formats.go`, `LabelLayoutOptionen.svelte`, `stores/labels.svelte.js`, dazu
zwei verschiedene Vorgaben — `avery_3475` im Druck-Center, `zweckform_l4760` im
Lieferanten-Weg). Der Umbau des Druck-Centers auf die Server-Liste wäre eine
Verhaltensänderung an einem täglich benutzten Bildschirm und ist **nicht** gemacht;
stattdessen hält `etikettformate-konsistenz.test.js` die Kopien deckungsgleich.

Stand: 2026-08-06

---

## Erledigt (2026-08-06) — Audit-Nachlese

Ein externes Audit meldete 16 Punkte. Nachgeprüft haben sich davon acht bestätigt, drei
in anderer Ursache als gemeldet, zwei waren falsch. Vier weitere Funde kamen erst beim
Nachprüfen dazu — sie standen in keiner Meldung.

**Der teuerste Fund stand nicht auf der Liste:** `internal/crypto` las den
Master-Schlüssel vorrangig aus `ENCRYPTION_KEY` und erst danach aus
`APP_ENCRYPTION_KEY`. Geprüft wird beim Start aber nur der zweite Name — auf Länge, auf
Hex-Form und (mit `ENFORCE_PROD_SECRETS`) gegen die bekannten Default-Werte. Ein
gesetztes `ENCRYPTION_KEY` hätte alle drei Prüfungen umgangen und still gewonnen; ver-
und entschlüsselt worden wäre mit einem anderen Schlüssel als dem geprüften.
Schülerfotos und das gespeicherte SMTP-Passwort wären damit nicht falsch, sondern
**weg**. Dieselbe Machart wie „zwei Türen zum selben Zustand": ein zweiter Weg zu einem
Wert, der nur an einem Weg abgesichert ist.

| Fund                                                                                            | Was es behauptete                         | Was stimmte                                                                                                                                                                                                   |
| ----------------------------------------------------------------------------------------------- | ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `SECURITY.md`: „docker-compose.yml erzwingt per `${VAR:?}`, dass **alle** Secrets gesetzt sind" | Absicherung durch Compose                 | Nur `POSTGRES_PASSWORD` und `IMAP_HOST`. `JWT_SECRET` und `APP_ENCRYPTION_KEY` fallen auf die **committeten** Defaults zurück; einzige Absicherung ist der Code-Guard, und der ist standardmäßig aus.         |
| `starttlsWennMoeglich`                                                                          | „verifiziertes STARTTLS"                  | Bot der Server keines an, ging die Mahnung im Klartext raus — mit Schülernamen und Elternadressen. Ein MITM konnte das durch Streichen der Erweiterung selbst herbeiführen.                                   |
| Cover-Proxy `/api/images/cover`                                                                 | öffentlicher Bild-Cache                   | Einziger unauthentifizierter Pfad, der eine ausgehende Verbindung **und** ein volles `image.Decode` auslöst — ohne Größengrenze, ohne Dimensionsprüfung, ohne Bremse, mit dem Query-Parameter als Dateinamen. |
| `maskiereToken` „in beiden Logzeilen"                                                           | Token nirgends im Log                     | Die Panic-Recovery schrieb den rohen Pfad. Beim ersten Fix schlicht übersehen.                                                                                                                                |
| `ReadHeaderTimeout` am Server                                                                   | Anfragen sind zeitlich begrenzt           | Nur die Kopfzeilen. Ein Rumpf durfte beliebig langsam kommen; der Kontext-Deadline der `TimeoutMiddleware` bricht kein blockierendes `Read` ab.                                                               |
| `docker-compose.local.yml`                                                                      | Entwicklungsstack, „nicht laxer als Prod" | Der DB-Port war auf `127.0.0.1` gebunden und begründet — das Backend eine Zeile darunter auf `0.0.0.0`, mit `IMAP_HOST=mock`, das jedes Passwort durchwinkt.                                                  |
| `SCRIPTS.md`: Backup-Pipeline „→ AES-GCM → 0600"                                                | gilt für alle Backups                     | Galt nur für den nächtlichen Job. `scripts/backup.sh` **und** `./update.sh` legten unverschlüsselte Volldumps mit Standardrechten ab — letzterer war überhaupt nicht dokumentiert.                            |
| `scripts/backup.sh` meldete „Backup erfolgreich"                                                | pg_dump lief durch                        | Ohne `pipefail` lieferte die Pipe den Status von `gzip`, und `gzip` gelingt auch dann, wenn `pg_dump` abgebrochen ist.                                                                                        |
| `Caddyfile` im Repo-Root                                                                        | Vorlage für den Reverse-Proxy             | Zeigte auf `bibliothek-backend-local:8083` — ein Container dieses Namens existiert nur lokal und lauscht auf 8084.                                                                                            |
| `useStudentEditForm({ student, … })`                                                            | Formular des ausgewählten Schülers        | Destrukturieren nimmt einen Schnappschuss. `save()` hätte das PATCH an die **zuvor** geöffnete ID geschickt — mit Erfolgsmeldung.                                                                             |
| `bind:this` auf `let` in `UnifiedInventory`                                                     | `$effect` setzt den Fokus ins Scanfeld    | Ohne `$state` ist die Zuweisung nicht reaktiv: Der Effekt lief einmal, bevor das Feld existierte, und nie wieder. Jeder Scan wäre ins Leere gelaufen — ohne Fehlermeldung.                                    |

**Zwei Meldungen waren falsch**, und beide auf dieselbe Weise: Sie beschrieben eine
Grenze, die es gab, als fehlend. Die XLSX-Importe haben `http.MaxBytesReader`
(100/20 MB) und hängen hinter `RequirePermission("manage_inventory")`. Und der
Klartext-Versand hätte die **SMTP-Zugangsdaten** nie preisgegeben: `smtp.PlainAuth`
verweigert die Übertragung über eine unverschlüsselte Verbindung von sich aus. Der
Inhalt war das Problem, nicht die Anmeldung — bei einem Relay ohne Zugangsdaten fiel
diese Bremse allerdings ganz weg.

**Merksatz dieses Tages:** Ein gemeldeter Befund ist eine Behauptung wie jede andere.
Drei der 16 stimmten in der Wirkung, aber nicht in der Ursache — wer sie so repariert
hätte, wie sie dastanden, hätte die falsche Stelle angefasst und den Fund für erledigt
gehalten.

Stand: 2026-08-06

---

## Erledigt (2026-08-04) — Littera-Schreibpfad

Fünf Funde, alle derselben Machart wie am 30.07.: Etwas behauptete etwas, und die
Behauptung war nie gegen die Wirklichkeit gehalten worden. Nur diesmal war die
Wirklichkeit nicht der Code, sondern **das Buch im Regal**.

| Fund                                | Was es behauptete                          | Was stimmte                                                                                                                                                                                                                          |
| ----------------------------------- | ------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `barcode_id` aus der Exemplarnummer | „Auf dem Etikett steht die Exemplarnummer" | Der Scanner liest eine EAN-13: aufgedruckt `105785`, gescannt `1057850039567`. Keines der 61.520 Exemplare wäre auffindbar gewesen — gemerkt hätte man es an der Theke.                                                              |
| Schülerausweis `[0395] 37`          | Der Aufdruck ist der Scanwert              | Gescannt kommt `B97601826457`, die Nummer des Kartenherstellers. Sie steht in keinem Stammdatenfeld, nur in Litteras `FremdLeserNummer`.                                                                                             |
| „10.002 Titel mit Autor (93 %)"     | Autorenabdeckung                           | Mitgezählt waren Standortvermerke, die als Personen erfasst sind: `Buchbestand Bibliothek` allein auf 6.711 Titeln. Bei 7.131 Titeln stünde ein Regalvermerk in der Autorenangabe. Echt sind 9.029 (84 %).                           |
| Geburtsdaten aus Littera            | Jahrgänge der Personen                     | Gos Jahrhundertgrenze liegt bei 69: 69 Lehrkräfte der Jahrgänge 1946–1968 kamen als **2046–2068** an.                                                                                                                                |
| Präfixlose Omnibox-Auflösung        | Buch → Schüler → Suche                     | Lehrkräfte stehen in `benutzer`, nicht in `schueler`. Ein gescannter Lehrerausweis lief bis in die Volltextsuche. Die passende Abfrage gab es längst — sie hing allein hinter `L-` und ließ sich als rohes SQL am Pool nicht testen. |

**Merksatz dieses Tages:** Aufdruck, Datenbankspalte und Scanwert sind drei verschiedene
Dinge. Für jede Barcode-Quelle einmal real in einen Texteditor scannen, bevor irgendetwas
nach `barcode_id` geschrieben wird.

Dazu, aus demselben Lauf und nicht kleiner: **`npm ci` lief gegen das committete Lockfile
gar nicht durch** (`typescript@^7` gegen `typescript-eslint@8.65`, das `<6.1.0` verlangt).
Lokal war ESLint grün, weil `node_modules` älter war als das Lockfile — auf einem frischen
Klon oder in CI wäre es rot gewesen. Ein grünes Gate beweist nichts über das, was im Repo
steht, wenn die Umgebung abgedriftet ist.

Stand: 2026-08-04

## Erledigt (2026-07-30)

Alle fünf Funde eines Tages waren dieselbe Art Fehler — etwas behauptete zu
funktionieren, und nichts prüfte die Behauptung nach:

| Fund                                   | Was es behauptete                 | Was stimmte                                                             |
| -------------------------------------- | --------------------------------- | ----------------------------------------------------------------------- |
| `expect([200, 400, 500])` im Mail-Test | prüft den Testversand             | konnte nicht rot werden; schickte Felder, die die API nie gelesen hat   |
| CSRF-Ausnahme für `/api/admin` & Co.   | „Inventur-Modul hat eigenes CSRF" | dieses System gab es im Code nicht; sechs Admin-Mutationen ungeschützt  |
| SMTP-Einstellungen im Admin-Bereich    | steuern den Mailversand           | wurden nur vom Test-Knopf gelesen; echte Mails nahmen die Umgebung      |
| Diagnose bei SMTP-Fehlern              | steht im Formular                 | jede 500 wurde zu „interner Datenbankfehler" eingedampft                |
| Ergebnis des Mailversands              | Nachricht zugestellt              | war das Ergebnis der Verabschiedung; Abbruch danach galt als Fehlschlag |

**Merksatz:** Der Fundort ist immer die Stelle, an der eine Zusicherung steht, die
niemand prüft — ein Kommentar, ein Testname, eine Eingabemaske. Wer dort sucht,
findet; wer im Code sucht, findet Schönheitsfehler.

### Nachtrag desselben Tages: die Liste selbst abgearbeitet

| Fund                                                | Was es behauptete                             | Was stimmte                                                                                                                                                                                                                      |
| --------------------------------------------------- | --------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `schadensfall.spec.js` fiel unter Last um           | Test prüft den Elternbrief                    | `popup.url()` wurde gelesen, bevor die Navigation übernommen hatte — allein grün, im vollen Lauf manchmal rot. Jetzt wird die Adresse abgewartet und die Route zusätzlich am Inhaltstyp geprüft.                                 |
| `SendTemplateMail`                                  | Vorlagen-Versand des Systems                  | kein einziger Aufrufer; die Vorlagen werden anderswo direkt geladen. Entfernt.                                                                                                                                                   |
| `inventur`-Backup-Modul samt Benachrichtigungs-Mail | zweites Backup-System mit Mail bei Änderungen | `NewAPIHandler` hat das Feld nie gesetzt — unerreichbar. Gesichert wird über `jobs.BackupJob`. Die `.env.example` lud dazu ein, `BACKUP_EMAIL_TO` zu setzen und auf Mails zu warten, die nie kommen konnten. Ersatzlos entfernt. |

Der dritte Fund kam aus der Prüfung des zweiten: Der Eintrag stand hier als
„bewusst so entschieden" — und die Begründung hielt der Nachfrage nicht stand. Auch
eine Notiz in dieser Liste ist eine Zusicherung, die jemand prüfen muss.

Stand: 2026-07-30
