# Offene Arbeit

Stand: 23.09.2026

**Die eine Liste.** Hier steht alles, was noch zu tun, zu prüfen oder zu entscheiden ist — Code,
Betrieb und Schule. Einen zweiten Ort gibt es nicht. Erledigtes wird gelöscht, nicht archiviert:
Die Geschichte steht in den Commit-Nachrichten und in `git log -p docs/OFFEN.md`.

Andere Dokumente erklären (Konzept, Anleitung, der Katalog der Bugklassen in
[sweeps.md](sweeps.md)), führen aber keine eigene Offen-Liste.

---

## Was jetzt dran ist

**Die Sichtung vom 16.09.2026** (Abschnitt 9): offen ist die Frage 9.3 c. Die zwei Bedingungen
aus 9.9 — DSGVO-Nachweis sowie Hosting- und Pflegekonzept — sind am 23.09.2026 zurückgestellt.

**Was bei dir liegt — der Reihe nach:**

1. **Der Etiketten-Lauf im Druck-Center** (4.8): Stichtag 15.07.2026, gemessen. Er ändert Daten.
2. **Der Nachweis von Hand für die Theke ohne Netz** (Abschnitt 2): Netz kappen, Bücher aller
   Formen und zwei Ausweise scannen, 20 Minuten warten, Netz zurück, Meldungen ansehen. Dazu
   der Nachweis für den Server.
3. **Kleine Fragen:** 5.18 (Klasse umbenennen statt löschen?), 5.1 (wer bucht eine Überweisung
   ein), 5.16 (Ausweisnummer leeren),
   5.19 (Auskunft und Vormerken für Kollegen), 9.3 c (Bücherei-Sperre wegen überfälliger
   Schulbücher; Vorschlag: so lassen), 5.5 (Jahrgang am Titel: „unbekannt" statt Vorgabe 5 bis
   10?), 5.5 (Google Books nur noch für Cover?).
4. **Liegt bei anderen** (Abschnitt 8): die Anfragen an Schule, Schulamt und Schulträger, dazu
   der Wortlaut des Eigentumsvermerks der Schülerbücherei (Einstellungen → Schule; leer heißt,
   diese Bücher tragen keinen Vermerk). Hier ist nichts zu tun außer nachzufragen, wenn nichts
   kommt.

**Im Code, in dieser Reihenfolge** (freigegeben am 23.09.2026, die Reihenfolge ist meine):

1. **4.19** Ferienkalender, in den fünf Stufen dort — er ändert Fristen, also Stufe für Stufe.
2. **4.18** Werk über den Auflagen — ändert die Nachbestell-Liste, also zuletzt.
3. **5.21** Palettenfarben nebenher, Bildschirm für Bildschirm.
4. Mahnverfahren nach der Antwort zu E6 (**4.4**): **5.13** Stufe 3 (5.3); nach E5 (**8.3**): **5.4**.
5. **5.10** (Gates und Werkzeuge) und Abschnitt 6 nur mit Anlass.

Von 4.20 (Schlagworte) bleiben nur die Littera-Schlagworte; sie kommen mit dem Backup (7.2).

**Parallel auf der Schulseite:** Abschnitte 7 und 8 — zuerst S3 (7.3), das Littera-Backup (7.2),
die Anfragen E1, E2, E5 (8.1–8.3), B3 und B4 (8.5) und ein Termin für die Abnahmen (7.7). Einen
echten LUSD-Import erst nach der Littera-Übernahme (7.2).

**Was liegen bleiben darf:** die übrigen B-Punkte in Abschnitt 5, die Beobachtungen in 6 und die
Betriebspunkte in 7. Keiner davon schadet still; sie werden gebündelt erledigt.

---

## So wird die Liste geführt

Vor jedem Fund steht dieselbe Frage — nicht „ist das hässlich?", sondern **„kann das still
jemandem schaden?"**

|       | Kategorie                                                                                                                                                                                            | Umgang                                                               |
| ----- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| **A** | Kann **stillschweigend** ein falsches Ergebnis für einen echten Menschen erzeugen: doppelte Mahnung an Eltern, Daten beim falschen Empfänger, verlorene Eingabe, ein Gate, das nicht rot werden kann | **Sofort**, eigener Commit, eigener Test, eigene Deploy-Entscheidung |
| **B** | Fehler, der sich **laut** meldet, oder Unordnung ohne Wirkung nach außen: totes Codestück, wackeliger Test, Doppelung                                                                                | Hier notieren, gebündelt abarbeiten                                  |
| **C** | „Wenn ich schon mal hier bin" — Umbenennungen, Stilfragen, Refactorings ohne Anlass                                                                                                                  | Nur mit Anlass und Zeit                                              |

1. **Ein Fund = ein Commit.** Was beim Reparieren zusätzlich auffällt, kommt hierher, nicht in
   denselben Commit.
1. **Kategorie A wird belegt, nicht behauptet:** ein Test, der am alten Code rot wird. Bis ein
   Fund nachgestellt ist, heißt er „Verdacht".
2. **Neues kommt nur hierher** — Funde, offene Fragen, Betriebspunkte. Kein Issue, kein anderes
   Dokument. Eine Frage steht hier, bevor die Antwort kommt.
3. **Erledigt heißt:** hier löschen — Datum und Begründung stehen in der Commit-Nachricht, ein
   Archiv gibt es nicht. Eine Antwort bekommt „Entschieden am …" und fällt weg, sobald
   sie umgesetzt ist. Die Nummer eines gelöschten Punkts wird nicht wieder vergeben —
   Kommentare im Code nennen sie als Herkunft.
4. **Die Reihenfolge** wird bei jeder Änderung mitgepflegt.

---

---

## 2. Offline-Betrieb der Theke — der Nachweis steht aus

Gebaut sind alle drei Stufen (15./16.09.2026): Die Theke hält die Buch-Barcode-Liste im Browser
und nimmt jede Scan-Form offline an, der Sync schickt an `POST /api/action/nachbuchen`, und was
nicht durchging, steht als Meldungsliste am Band. Wie sich das verhält, steht in
[FACHKONZEPT.md](FACHKONZEPT.md) 18.4 und im [Handbuch](HANDBUCH.md).

Offen ist **der Nachweis (2.3):** Stufe 1 und 3 gehören von Hand in den echten Chrome, Stufe 2
über die Tür.

### 2.3 Nachweis am Stack (je Stufe, echter Chrome)

- Stufe 1: DevTools-Drosselung 20 s, Schüler laden, Buch, Escape → IndexedDB trägt den Schüler.
  Offline „zurückgeben" im Profil → Rückgabe. Lehrkraft laden, Buch → Handapparat. IndexedDB
  blockiert → „NICHT gespeichert". Zwei Sicherungen einspielen → ein Stapel. Zwei parallele
  Anfragen mit einem Schlüssel gegen `/api/action` → eine Ausleihe.
- Stufe 2: curl mit Cookie gegen `/nachbuchen`: Umbuchung, Doppelscan, gesperrter Ausweis,
  Rückgabe vor älterer Ausleihe (409 `veraltet`), abgebrochener Online-Versand mit gleichem
  Schlüssel (Ausleihe nachgeholt), verloren gemeldetes Buch (Forderung endet, Hinweis in der
  Meldung), zwei parallele Aufrufe auf ein Exemplar, ein Online-Scan parallel zum Nachbuchen
  desselben Kindes (kein Deadlock); `docker stop` der Datenbank → 503 → Eintrag bleibt;
  `pg_stat_activity` beim Nachbuchen von 200 Einträgen (eine Verbindung je Eintrag).
- Stufe 3: Netz aus, Bücher aller Formen und zwei Ausweise scannen, ein Band, kein Vollbild,
  20 Minuten warten ohne Sperre, Netz an → Sperre; anmelden, nachgebucht, Meldungen an einem
  zweiten Browser; Format-1-Sicherung eines anderen Rechners einspielen.

### 2.4 Nicht im Umfang

Anmeldung ohne Server · Schülerdaten auf dem Rechner · Umbuchen im Online-Weg · Geräte offline ·
scannende Person im Protokoll (nachgebucht wird unter dem beim Sync angemeldeten Konto; steht in
der Doku).

## 4. Entscheidungen

Die Nummern bleiben fest; beantwortete Fragen fallen weg, sobald sie umgesetzt sind.

### 4.4 E6: Nach der Übergabe an die Schulaufsicht

Bleibt der Schüler gesperrt und die Forderung offen, bis das Sekretariat „bezahlt laut
Finanzbericht" bucht — oder gilt die Übergabe schulseitig als erledigt? **Vorschlag (Konzept):**
Sperre bleibt, Löschblockade fällt. **Seit dem 22.09.2026 gilt „Sperre bleibt" nur noch für
die Schülerbücherei:** Für Lernmittel darf keine Forderung eine Ausleihe abweisen (9.3 c). **Wann:** sobald ein erster echter Bescheid absehbar ist;
blockiert 5.3. Einzelheiten in [mittel_konzept.md](mittel_konzept.md), Abschnitt 6.

### 4.8 Etiketten-Altbestand nachtragen — gemessen, der Lauf steht aus

Der Lauf geht über das Druck-Center („Fehlende Etiketten" → „Altbestand aufräumen", mit
Vorschau und Stichtag).

**Gemessen am Testserver am 21.09.2026** (Exemplare ohne Etikett-Vermerk, nicht ausgesondert,
nach `erworben_am`): 30.654 Exemplare ohne `B-`-Nummer, alle am 15.07.2026 — dem Tag der
Littera-Übernahme. Jede `B-`-Nummer liegt danach: 23.07. (8, davon 1 im Zulauf), 31.07. (7),
02.08. (1), 10.09. (2, beide im Zulauf), 16.09. (9). Der Lauf vergleicht
`erworben_am <= Stichtag`; **der Stichtag 15.07.2026 trifft genau den Altbestand** und lässt
die 27 Neuzugänge offen.

**Der Lauf selbst:** Stichtag 15.07.2026 eintragen; die Vorschau muss 30.654 zeigen. Den ganzen
Lauf nimmt nichts zurück (einzelne Exemplare holt „Etikett zurücksetzen" in der
Nachdruck-Liste zurück). Am Schulserver vorher dieselbe Zählung wiederholen — die Zahlen oben
gelten für den Testserver:

```sql
SELECT (barcode_id LIKE 'B-%') AS b_nummer, (bestellstatus IS NOT NULL) AS im_zulauf,
       erworben_am::date AS tag, count(*)
FROM buecher_exemplare
WHERE etikett_gedruckt = false AND ist_ausgesondert = false
GROUP BY 1, 2, 3 ORDER BY 3, 1;
```

**Wann:** vor Abnahme-Flow 4. Kommt eine neue Littera-Übernahme mit Neuaufbau (7.2), erledigt
sich der Punkt — der Import setzt den Vermerk seit dem 16.08.2026 selbst.

### 4.18 Neue Auflage eines Schulbuchs — ein Werk über den Auflagen

**Der Fall (gefragt am 17.09.2026):** Ein Schulbuch wird nachbestellt, es gibt es aber nur noch in
der nächsten Auflage — neue ISBN, also ein neuer Titel. Es geht ausdrücklich NUR um Schulbücher.

**Was heute passiert.** `buecher_titel.isbn` ist UNIQUE; die neue Auflage wird zwangsläufig eine
zweite Titelzeile, und das ist richtig so — es sind zwei verschiedene Bücher. Nur hängen
`meldebestand` und der Gesamtbestand am TITEL (`api/reorders.go`): Die alte Zeile behält den
Meldebestand 95 und zählt ihre 65 Restexemplare, die neue startet mit der Vorgabe 5 und hat 30.
Die Nachbestell-Liste steht damit für dasselbe Buch zweimal da, und beide Zahlen sind falsch.
Dasselbe gilt für den Klassensatz (`class_books`), Vormerkungen und Reservierungen — sie alle
hängen am Titel.

**Entschieden am 17.09.2026:**

- **Der Bedarf rechnet am Werk**, nicht an der Auflage: „95 Stück Mathe 7, egal welche Auflage."
  Die Suche zeigt einen Treffer mit der Gesamtzahl und darunter die Aufschlüsselung je Auflage.
- **Die Ausgabe warnt**, wenn eine Klasse gemischte Auflagen bekommt. Still darf das nicht
  passieren: verschiedene Auflagen heißen verschiedene Seitenzahlen.
- **Zusammengelegt wird nichts.** Zwei Titelzeilen zu einer zu machen und die Exemplare umzuhängen
  verliert die Auflage — und genau die ist bei Schulbüchern die Information, die zählt. Jedes
  Exemplar bleibt an seiner Auflage.
- **Die Form:** eine schmale Tabelle `werke` (Id, Name) und eine NULLBARE Spalte `werk_id` am
  Titel; gruppiert wird über `COALESCE(werk_id, id)`. Jeder Titel ohne Werk ist damit sein eigenes
  Werk — es gibt keinen Zwischenzustand „halber Bestand gepflegt", und jeder Lesepfad, den die
  Frage nicht betrifft, bleibt unverändert. Am Werk rechnen Meldebestand, Bedarf und
  Nachbestellung; an der Auflage bleiben Exemplar, Etikett, Ausleihe und Ausgabe.

**Entschieden am 23.09.2026: Der Besteller legt das Werk an.** Der Vorschlag entsteht
automatisch beim Nachbestellen, das Ja gibt der Besteller. Vollautomatisch über den Namen zu
gruppieren verbindet früher oder später zwei „Deutschbuch 7" verschiedener Verlage; rein von Hand
pflegt es niemand, und der Bedarf bleibt falsch. Auf dem Weg über die Nachbestellung ist der
Vorschlag praktisch sicher, weil er von einem konkreten Titel ausgeht.

**Freigegeben am 23.09.2026, nicht gebaut:** das Werk samt Migration, Gruppierung im Bedarf
und Warnung in der Ausgabe. Ändert das Schema und rechnet die Nachbestell-Liste anders — in
Stufen mit Nachweis.

### 4.19 Frist fällt in die Ferien

**Der Fall.** Ausleihe am letzten Schultag vor den Herbstferien, Frist 21 Tage: fällig mitten in
den Ferien. Das Kind kann nicht zurückgeben, steht danach in der Mahnliste und ab einem
überfälligen Medium mit gesperrtem Ausweis an der Theke (`MaxOverdueItems`, Vorgabe 1).

**Was heute passiert.** `calculateDueDate` (`internal/service/loan_rules.go`) rechnet
Kalendertage: Stichtag bei Lernmitteln, feste Tageszahl sonst. Kein Kalender geht ein. Der einzige
Behelf ist der Ferien-Leseclub — ein festes Zieldatum für ALLE Ausleihen, von Hand ein- und
auszuschalten.

**Woher kämen die Tage bei uns?** Das ist die eigentliche Frage. Gebaut ist bisher nur, was der
LMF-Plan braucht:

- **Feiertage Hessens:** gerechnet, kein Pflegeaufwand (`pkg/lmfplan/feiertage.go`; die gesetzlichen
  Feiertage Hessens samt Fronleichnam).
- **Sommerferien:** Programmtabelle nach dem KMK-Beschluss bis 2030 plus eigene Einträge der
  Schule (Einstellungen → LUSD & Versetzung → Sommerferien). Läuft die Tabelle aus, warnt die
  Betriebsbereitschaft zwei Jahre vorher.
- **Bewegliche Ferientage, pädagogische Tage, Brückentage:** `lmf_plan_freie_tage` — sie hängen
  aber am einzelnen Plan (`plan_id`), nicht am Schuljahr.
- **Herbst-, Weihnachts- und Osterferien gibt es nirgends**, Öffnungstage der Bibliothek auch
  nicht. Genau diese Ferien sind die, in die eine 21-Tage-Frist fällt.

**Entschieden am 18.09.2026:**

- **Öffnungstage gibt es nicht** — die Bibliothek hat keine festen Öffnungszeiten. Damit besteht
  der Kalender aus Ferien, Feiertagen und Wochenenden; die Hälfte, die Littera „Öffnungstage"
  nennt, entfällt ersatzlos.
- **Laufende Ausleihen rutschen mit.** Wer im September ausgeliehen hat und im Oktober die
  Herbstferien nachgetragen bekommt, soll nicht gemahnt werden, weil die Eintragung zu spät kam.
  Das Muster dafür steht schon: Beim Veröffentlichen des LMF-Plans folgen die offenen
  Schulbuch-Ausleihen dem Termin ihrer Klasse (`api/lmf_termine_frist.go`), und die Meldung nennt
  die Zahl.
- **Die Ferien kommen als Datei** (iCal), nicht als Tipparbeit: sechs Zeiträume je Schuljahr von
  Hand einzutragen, macht niemand zweimal. Die Datei holt die Schule selbst, z. B. bei
  `schulferien.org/deutschland/ical/`; hochgeladen wird sie in den Einstellungen. Zur Laufzeit
  fragt der Server nichts ab — dieselbe Linie wie bei der Sommerferien-Tabelle.
- **Ein Warner**, wenn der Kalender ausläuft. Ein Kalender, der still endet, rechnet ab dem
  ersten fehlenden Tag wieder falsch, ohne dass es jemand merkt.

**Beim Mitrutschen gilt dieselbe Ausnahmeliste wie beim LMF-Plan:** nur offene Ausleihen, nur nach
hinten, und nicht angefasst werden von Hand gesetzte Fristen, Lernmittel mit Termin aus dem Plan
und die Fristen des Ferien-Leseclubs. Jede Verschiebung steht im Protokoll, und die Meldung nennt
die Zahl der betroffenen Ausleihen.

**Entschieden am 22.09.2026 — zwei Dateien, und die Datei gewinnt.** Für Ferien und für
Feiertage je eine iCal-Datei. Die gerechneten Feiertage Hessens (`pkg/lmfplan/feiertage.go`)
bleiben die Vorbelegung für jedes Jahr, zu dem nichts hochgeladen ist; für ein Jahr aus der Datei
gilt die Datei. Damit hat jedes Jahr genau eine Quelle statt zweier nebeneinander — dasselbe
Muster wie bei den Sommerferien heute (eigener Eintrag schlägt Programmtabelle). Die Gefahr
bleibt, dass jemand die Datei eines anderen Bundeslandes erwischt und Fristen auf einen Tag
schiebt, an dem die Schule offen hat; dagegen steht die Vorschau (Falle 2) und die Gegenprobe des
Programms: Weichen die Feiertage der Datei von der Rechnung ab, nennt die Vorschau jeden
abweichenden Tag, bevor übernommen wird.

**Entschieden am 22.09.2026 — wo der auslaufende Kalender gemeldet wird:** als **Band an der
Theke**, für die, die den Kalender ändern dürfen, sobald er in weniger als der längsten Leihfrist
endet — dort entsteht die falsche Frist; und als **Mail an die Leitung**, weil der, der das
Hochladen vergisst, die Einstellungen nicht von selbst aufsucht. Die Betriebsbereitschaft nennt
den Kalender wie heute schon die Ferientabelle, ist aber nicht der meldende Weg.

**Was beim Bauen die Fallen sind** (vorab notiert, weil sie still danebengehen):

1. **`DTEND` ist bei ganztägigen Terminen exklusiv.** Ein Ferienzeitraum „bis 31.10." steht in der
   Datei als `DTEND:20261101`. Wer das direkt übernimmt, hängt einen Tag an oder verliert einen —
   und der erste Schultag wird zum Ferientag. Gate genau darauf.
2. **Eine Vorschau vor dem Übernehmen**, wie beim LUSD-Import: „Diese 6 Zeiträume werden
   eingetragen." Die Quelle ist austauschbar (`schulferien.org` ist eine von mehreren und nicht
   amtlich), und auch der Dateiname ist keine Prüfung — er heißt zwar „Ferien Hessen", ist aber
   in zwei Sekunden umbenannt. Geprüft wird der Inhalt, von einem Menschen an der Vorschau.
   **Die Maschine kann dabei eine harte Probe beisteuern:** Jede Ferien-Datei enthält die
   Sommerferien, und die kennen wir für Hessen bis 2030 aus der Programmtabelle. Weichen sie ab,
   stimmt das Bundesland oder das Jahr nicht — dann warnt der Upload, statt die Fristen der
   ganzen Schule auf fremde Termine zu schieben.
3. **Die Sommerferien gibt es schon** — als JSON-Liste unter dem Schlüssel `sommerferien`
   (`pkg/lmfplan/ferien_einstellung.go`), gelesen von Planer und Selbstprüfung. Der neue Kalender
   muss sie aufnehmen, nicht neben ihnen stehen, sonst gibt es zwei Listen von Sommerferien. Der
   Upload ist dann ein Weg in EINE Tabelle, das Tippen von Hand der andere.
4. **Nur Datei, kein Adressfeld.** Eine URL, die der Server selbst abruft, wäre ein Abruf nach
   außen aus dem Schulnetz heraus — der Grund, aus dem schon die Ferientabelle im Programm steht.

**Freigegeben am 23.09.2026, in dieser Reihenfolge:** (1) Kalender-Tabelle samt Übernahme der Sommerferien,
(2) iCal-Upload mit Vorschau (Ferien und Feiertage, Gegenprobe gegen die Rechnung), (3) Frist
rechnet gegen den Kalender, (4) Mitrutschen bei Nachtrag, (5) Band an der Theke und Mail an die
Leitung, wenn der Kalender ausläuft.

**Nicht gebaut.** Ändert einen Schreibpfad und braucht eine Migration; Gate am Rückweg, vorher rot
gesehen.

### 4.20 Schlagworte am Titel — frei eintragbar wie in Littera

**Entschieden am 23.09.2026:** Schlagworte werden frei eingetragen, mehrere je Titel, mit
Vorschlägen aus dem Bestand — wie Littera („Schlagworte (Wertehilfe)"). Zusammengehalten wird
die Liste wie dort durch Pflege: Umbenennen und Zusammenführen ändern alle Titel auf einmal,
Verweise leiten Schreibweisen auf ein Wort („Tierfantasy" → „Fantasy"). Eine Liste, nicht zwei:
Die 15 bis 20 Wörter, die im Portal als Filter stehen, markiert die Pflegeseite. Das Feld am
Titel steht seit Migration 138; selbst gesetzt und änderbar sind dort die Grenzen: 30 Wörter je
Titel, 80 Zeichen je Wort, Vorschlagsliste die 500 häufigsten.

**Offen (freigegeben am 23.09.2026):** die **Littera-Schlagworte** (MAB 710) beim nächsten
Einspielen des Backups mitnehmen (7.2). Heute liest der Import sie, leitet das Fach ab und
verwirft sie. Vorher messen, wie viele Titel welche tragen.

**Aus Littera nur, wenn die Bücherei es braucht** (entschieden am 23.09.2026): die
Schlagwortliste drucken, den Schlagwortkatalog in eine Datei schreiben und aus einer lesen. Ein
Littera-Verweis kann mehreren Wörtern zugeordnet sein; unsere zeigen auf eins und stellen die
Eingabe am Titel richtig.

Nicht geplant: Verweise für Autoren (Pseudonyme). Der Autor ist ein Textfeld am Titel, ohne
eigenen Personensatz; das wäre ein eigener Umbau.

---

## 5. Abarbeitbar (Kategorie B)

### 5.1 Schäden und Benutzer

- **Was „Bezahlt" bedeutet** (17.09.2026, aus dem Abgleich der Anforderungsliste,
  [mittel_konzept.md](mittel_konzept.md) Abschnitt 3): Der Knopf in der Schülerakte
  verbucht heute eine Barzahlung am Tresen. Für Lernmittel des Landes sieht die Arbeitshilfe
  Bargeld nur als Ausnahme vor — mit Quittung und Weiterleitung binnen 14 Tagen. Die Briefe
  nennen inzwischen das Konto; ein eingehender Betrag kommt also in aller Regel als
  Überweisung, und niemand an der Theke sieht ihn. Zu klären mit der Schule: Wer bucht eine
  Zahlung ein, die auf dem Kontoauszug steht? Keine Bauarbeit, bevor das beantwortet ist —
  eine erfundene Antwort steht sonst als Vorgang in der Akte.

### 5.3 Folgen der Übergabe (nach 4.4)

`POST /api/bescheide/{id}/uebergeben` setzt heute nur Status und Zeitpunkt
(`repository/bescheid.go`). Offen:

- Übergabe-PDF (Original und Sammelliste).
- Die Gates der Übergabe-Folgen.

### 5.4 Schadensersatz Teil A, Etappen 3 und 4 (nach 8.3)

Konzept: [mittel_konzept.md](mittel_konzept.md), Abschnitt 4.7.

- **Kreis-Rechnung:** zweite Variante des Renderers (Nummernkreis `SB-Jahr-lfd`, Frist, Tabelle,
  Hinweis auf Ersatzbeschaffung, Zahlungsweg aus den Einstellungen; ohne Referenznummer im
  Landesformat, ohne Rechtsbehelfsbelehrung). Solange die Bankverbindung des Schulträgers fehlt,
  steht sichtbar „(Bankverbindung des Schulträgers nicht hinterlegt)". Warnung in der
  Betriebsbereitschaft. Topf in die Referenznummer, sonst kollidieren Land und Kreis an der
  UNIQUE-Spalte. Heute lehnt der Server jeden Topf außer `land` mit 409 ab.
- **Altbriefe entfernen** (es geht darum, ob es sie neben dem Bescheid überhaupt weiter geben
  soll): Elternbrief
  `pdf/schadensfall.go` ← `api/pdf.go`
  (`GenerateDamagePDFHandler`) ← Route in `api/routes_students.go` ← `useStudentProfile.svelte.js`;
  Rechnung `pdf/rechnung.go` ← `api/print.go` ← `GET /api/print/rechnung/{schueler_id}` in
  `api/routes_system.go` ← Knopf in `StudentProfileActions.svelte`; dazu
  `api/print_rechnung_pg_test.go` und `elternbrief_generiert*`. Der Eltern-Mahnbrief bleibt.
- **Doku:** FACHKONZEPT Abschnitt 3 (Mahnwesen ohne Bescheid) und 14 (PDF-Rechnung, Barzahlung am
  Tresen); SECURITY und VVT-Entwurf mit dem Zweck „Schadensersatz-Bescheid". Den VVT-Satz
  vorziehen, bevor die Schule den Entwurf beschließt (8.5).
- **Release** beim Abschluss.

### 5.5 Bestand, Katalog, Druck

- Listenimport gegen den Nummern-Wächter (Migration 131): Trägt eine Zeile der Datei die
  Ausweisnummer eines Lesers als Buch-Barcode, lehnt der Wächter ab und der ganze Import
  bricht mit der rohen Datenbankmeldung ab (`ON CONFLICT DO NOTHING` fängt nur den Index,
  nicht die Ausnahme). Laut, also richtig — nur die Meldung nennt weder Zeile noch Weg.
  Kategorie C, bis es einmal vorkommt.
- Jahrgang am Titel: Ein CHECK fehlt, „Jahrgang unbekannt" ist von der Vorgabe 5 bis 10 nicht
  zu unterscheiden — und seit dem Mehrjahresband (Migration 134) hängt eine Frist an „bis": Wer
  den Schalter auf einem Titel mit der Vorgabe umlegt, bekommt die 10. Ein CHECK allein löst
  das nicht; eine Vorgabe „unbekannt" (NULL) bräuchte die drei Leser (Mahnwesen „Jahrgang",
  Inventur, Portal-Filter) mit.
- „Klasse" neben der Spanne (22.09.2026): Zwei Jahrgangsangaben am Titel, „Klasse"
  (`grade_level`) und „von … bis" (`jahrgang_von/bis`). Mahnwesen „nach Jahrgang", Inventur
  nach Klasse und die Mehrjahresband-Frist lesen nur die Spanne; Titel-Tabelle,
  Klassenzuweisung und Listenfilter lesen die Klasse, der Portal-Filter liest beide. Die
  Zusammenlegung (Migration 135) ist zurückgenommen: Sie machte aus Klasse N die Spanne N
  bis N, und das trifft die Daten nicht. Lesend gemessen auf dem Testserver: 153 Titel mit
  Klasse, 129 davon Klasse 6–13 bei der Vorgabe 5 bis 10, 89 dieser 129 Lernmittel.
  Mehrjährige Bände tragen ein einziges Jahr („Natur und Technik - Biologie 7 - 10" und
  „Pontes Gesamtband": Klasse 7), Klasse und Signatur widersprechen sich („Forum Geschichte
  4 (Schulbuch Klasse 9)": Signatur Ges9, Klasse 10). Woher die Werte stammen, ist nicht
  belegt. Seit dem 22.09.2026 rät der Listenimport keine Klasse mehr (früher: erste Zahl im
  Titel, sonst 5) und schreibt ohne Spalte „klasse" NULL; eine Klasse 5 aus einem älteren
  Listenimport ist von einer gepflegten nicht zu unterscheiden. Nächster Schritt: je Titel
  entscheiden, welche Spanne gilt (Liste per Einzeiler
  unten), dann die Spalte mit genau diesen Werten ablösen. Dabei mitentscheiden: die Spalte
  „klasse" des Listenimports und der Klassenvorschlag der ISBN-Suche, der auch aus
  „Band 2", „Level 9" und jeder Zahl von 5 bis 13 im Titel eine Klasse macht.

  ```sql
  SELECT grade_level, jahrgang_von, jahrgang_bis, ist_lernmittel, signatur, titel
  FROM buecher_titel WHERE grade_level BETWEEN 1 AND 13
  ORDER BY ist_lernmittel DESC, signatur NULLS LAST, titel;
  ```

- **ISBN-10 und ISBN-13 desselben Buchs:** Die Normalform trennt beide bewusst (Migration 133),
  die Littera-Übernahme behält eine gültige ISBN-10. Die Bestelltür (`findeLokalenTitel`), die
  Markierung „Vorhanden" der Bestellsuche (`sammleExistierendeISBNs`) und die
  Dublettenkontrolle der Maske vergleichen nur die Normalform: Ein Titel mit ISBN-10 wird beim
  Bestellen per Strichcode (EAN-13) nicht gefunden und entsteht ein zweites Mal. Die Suche im
  Browser rechnet schon um (`isbnFormen.js`), am Server fehlt das Gegenstück. Gemessen am
  Testserver am 23.09.2026 (lesend): 100 Titel mit ISBN-10, 9.743 mit ISBN-13, 4 Paare mit
  gleichem Kern — alle aus der Littera-Übernahme vom 15.07.2026, ohne Exemplare; eines davon
  sind zwei verschiedene Bücher unter derselben Nummer („Heinrich Mann" und „Frédéric Chopin",
  rororo, `3499500252`/`9783499500251`). Ein Abgleich über beide Formen darf deshalb
  vorschlagen, nicht still zusammenführen.
- **Frage: Google Books als Quelle für Titeldaten.** `SucheNachISBN` fragt der Reihe nach DNB,
  Google Books und OpenLibrary; Google liefert also Titel, Autor und Verlag, wenn die DNB den
  Titel nicht kennt. Nach einer Notiz vom 22.09.2026 soll Google nur noch als Rückfall für das
  Cover dienen. Gilt das — dann aus der Reihe nehmen?

### 5.10 Gates und Werkzeuge

- Die Schema-Gegenrichtung ist blind für UNIQUE, Teilindizes und RESTRICT.
- Kein Gate gegen unbegrenzte Listen-Endpunkte.
- **Wiedervorlage 17. November 2026: `GO-2026-6452` (excelize).** Am 16.09.2026 erschien eine
  Schwachstelle ohne heile Fassung — der Eintrag führt alle Versionen ab 0 und nennt keine
  mit Fix. Sie trifft uns nicht: Der gewöhnliche Weg hat den Bereichsschutz seit v2.11.0,
  und der ungeschützte Auslagerungs-Weg ist unerreichbar, weil `xlsxgrenze.Optionen()` beide
  Entpackgrenzen gleich setzt (nachgemessen, vier Tests). Solange das so ist, steht sie als
  begründete Ausnahme in `security/vuln-ausnahmen.json`; das Gate
  (`scripts/govulncheck-gate.sh`) wird von allein rot, sobald die Wiedervorlage abläuft, die
  Ausnahme überflüssig wird oder irgendeine andere Schwachstelle unseren Code trifft. Zu tun:
  nachsehen, ob excelize inzwischen eine Fassung mit Fix hat — dann Ausnahme löschen und
  heben.
- Kein Rückweg für ältere Sicherungen beim Wechsel des `BACKUP_ENCRYPTION_KEY`.
- `e2e/icon-trefferflaechen.spec.js` misst die Bestellhistorie, legt aber keine Bestellung an:
  Allein oder ohne eine `bestell…`-Spec davor läuft es in die Zeitüberschreitung (lokal am
  23.09.2026, Bestellhistorie leer; der globale Teardown löscht die E2E-Bestellungen). In der
  vollen Suite legt eine alphabetisch frühere Spec sie an. Nach dem Muster von `seedBenutzer`
  selbst anlegen.

### 5.13 Mahnverfahren: Stufe 3 (nach 4.4)

Folgen der Übergabe (5.3): Übergabe-PDF und Sammelliste für das Schulamt; E6 (4.4). Das Modell
der Stufen 1 und 2 beschreibt das [Handbuch](HANDBUCH.md).

### 5.16 Leserdatei und Rolle Leitung — was noch offen ist

**Entschieden, damit die Frage nicht wiederkommt:** Ein Kollege hat bewusst keinen
Kontoauszug, keine Ersatzforderung und keine DSGVO-Auskunft in seiner Akte — sie gehören der
Schülerarbeit und lesen die Sicht `schueler`, ausgeblendet statt kaputt. Die E-Mail eines Kollegen steht am Konto
(`benutzer.email`, `UNIQUE lower(email)`) und nicht in `leser.eltern_email`; in der Akte wird sie
vom Konto gelesen und ist nachtragbar, solange keine da ist. Die Leitung sieht den Menüpunkt
„Einstellungen" und darf darin LUSD & Versetzung, Datenverwaltung, LMF-Aktionen und Lieferanten
bedienen; verschlossen sind Schule, Fristen und Mailversand (`manage_settings`) sowie Benutzer &
Rechte (`manage_users`). Der Nummernkreis bleibt unangetastet: Ohne Netz ist die Vorsilbe die
einzige Information, an der die Theke einen Buchscan von einem Ausweisscan unterscheidet.

**Offen, eine Frage:** Seit Migration 136 hat jedes aktive Konto eine Ausweisnummer. Die
Verwaltung kann sie an einem Kollegen weiter leeren (`TestAusweisnummerLeeren`, entschieden
am 16.09.2026, als der Hinweis am Feld noch „Leer lassen" sagte) — dann fehlt sie wieder, und
der Druck liefert eine leere Zeile. Soll Leeren dort eine neue Nummer ziehen statt keine?

### 5.18 Eine vertippte Klasse wieder loswerden — Rückfrage

Am 23.09.2026 entschieden: „Klasse löschen möglich machen". Beim Bauen stellte sich heraus, dass
die Beschreibung, auf der die Entscheidung stand, nicht stimmte:

- Die Auswahllisten lesen **nicht** die Tabelle `klassen`, sondern die Verweise:
  `GET /api/klassen` ist `SELECT DISTINCT klasse FROM schueler`, dazu Klassensätze und
  Zuordnungen. `klassen` ist ein Vokabular, das Trigger selbst füllen; im Go-Code liest es
  niemand.
- Gemessen am Testserver (23.09.2026, lesend): 108 Klassen im Vokabular, 30 ohne jeden Verweis
  (05A–08D aus Migration 079). Die sieht niemand; ein Löschen änderte nichts Sichtbares.
- Sichtbar ist eine vertippte Klasse, solange Schüler, ein Klassensatz, eine Zuordnung, eine
  Reservierung oder ein LMF-Termin sie tragen. Löschen verweigert dann die Datenbank
  (`ON DELETE RESTRICT` an sechs Tabellen).

**Vorschlag:** Statt „Löschen" ein **Umbenennen** an einer Stelle (Einstellungen → LUSD &
Versetzung): Liste der Klassen mit Zahl der Schüler, Klassensätze und Zuordnungen; „umbenennen
in …" zieht über `ON UPDATE CASCADE` alles mit, und gibt es das Ziel schon, werden die Verweise
dorthin umgehängt und die alte Klasse fällt weg. Eine Klasse ohne Verweis lässt sich dort auch
löschen. Achtung: Wer Schüler umhängt, ändert ihre LMF-Termine und Klassensätze mit — die
Rückfrage nennt die Zahlen.

### 5.19 Lesepfade gegen die Sicht `schueler` — was offen bleibt

**Zwei Fragen:**

- **DSGVO-Auskunft** (`api/dsgvo_auskunft.go`): Für einen Kollegen gibt es sie nicht (5.16
  nennt das als Absicht). Auch eine Lehrkraft kann Auskunft über ihre Daten verlangen — soll
  die Auskunft für jeden Leser gehen?
- **Vormerken** lässt sich nur für Schüler — so bietet es die Oberfläche an, und seit dem
  21.09.2026 lehnt es auch die Tür ab. Soll ein Kollege vormerken können, ist das ein eigener
  Umbau über vier Lesepfade der Warteschlange, kein Schalter.

**Auf einer anderen Anlage vorher zählen:** Vormerkungen und Schadensfälle an einem Kollegen
sehen die Lesepfade gegen die Sicht nicht — die Warteschlange geht über eine solche Vormerkung
hinweg. Am Testserver waren am 21.09.2026 beide Zählungen 0; neue Vormerkungen für Kollegen
lehnt die Tür seit dem 21.09.2026 ab.

```
SELECT count(*) FROM vormerkungen v JOIN leser l ON l.id = v.schueler_id WHERE l.art <> 'schueler';
SELECT count(*) FROM schadensfaelle sf JOIN leser l ON l.id = sf.schueler_id WHERE l.art <> 'schueler';
```

### 5.20 Aus der Durchsicht von PR 631 (21.09.2026) — was offen bleibt

- **Anfrage-Log mit Dauer und Anfragekennung:** nicht gebaut, nur die Doku auf den Ist-Stand
  gezogen. Mehr Logzeilen am Schulserver sind eine Betriebsfrage.
- **Totalverlust des Servers ist nicht beschrieben** (gehört zu 7.4). Der Entwurf im PR ist
  nach eigener Angabe unerprobt und scheitert in Schritt 5: Das Backend hat nur benannte
  Volumes, die Sicherung vom zweiten Ort liegt also nicht im Container, und die entschlüsselte
  Datei entsteht in einem Wegwerf-Container und ist danach fort. Schreiben und an einem fremden
  Ziel durchspielen, nicht herleiten.
- **Zu 7.3:** Der zweite Ort muss kein S3 sein — ein zweiter Rechner per Kopierbefehl oder
  eine getauschte Platte tun dasselbe ohne Vertragsfrage. Eine Betriebsentscheidung.

### 5.21 Palettenfarben auf M3-Rollen

Stand 23.09.2026: 1471 Fundstellen mit Tailwind-Palettenfarben (`slate`, `blue`, `emerald` …),
gehalten von der Ratsche `frontend/src/lib/frontend-hygiene-farben.test.js`; Neues entsteht nur
noch auf Rollen. Umstellen ist eine Umgestaltung, keine Umbenennung: Die Palette führt sechs
Textgraustufen, M3 zwei Rollen, und für „in Ordnung" kennt M3 keine Farbe. Vorschlag:
Bildschirm für Bildschirm, die größten zuerst, je Portion ein Commit, am gerenderten Bildschirm
geprüft. Das Muster steht in Buchformular und Bestellfenster: Zustände über ui/StatusChip, Cover
über ui/BuchCover, Rückmeldung beim Zeigen über den State-Layer statt `hover:bg-*`, ein Fehler
über den Fehlerzustand des Feldes statt eines farbigen Kastens. Für die übrige Anwendung
freigegeben am 23.09.2026.

Dazu gehört die Leiste des Ausweisdrucks in der Leserdatei (`students/AuswahlAktionsleiste`,
dunkel in Palettenfarben): Seit dem 23.09.2026 gibt es für markierte Zeilen `ui/AuswahlLeiste`
(Schlagwort-Pflege). Beim Umstellen zu klären: wohin der Hinweis „ohne Ablaufjahr" und das Feld
„Ab Feld" kommen — beides passt nicht in die 64 px hohe Leiste.

---

## 6. Beobachten und Kategorie C (nur mit Anlass)

### 6.1 Beobachtungen

- Der Stand-Merker der Barcode-Liste (Anzahl + `max(aktualisiert_am)`) rechnet mit dem Beginn
  der Transaktion: Ändert eine lange Transaktion ein Etikett und committet nach einem kürzeren
  Schreiber, bleibt es bei 304. Nachgestellt hinter dem Build-Tag `raster`
  (`TEST_DATABASE_URL=… go test -tags raster -run TestRaster_ ./repository/`). Heute ändert nur
  `UpdateCopyBarcode` einen Barcode, als kurzer Einzelbefehl — scharf wird es erst mit einem
  Schreiber, der in einer langen Transaktion umetikettiert (Durchgang 15.09.2026).
- Ein Ausweis aus dem Altbestand ohne Vorsilbe (gemessen `B97601826457`) ist ohne Netz „unklar"
  und wird abgewiesen. Hängt am Ausweis-Neudruck und entscheidet sich mit dem frischen
  Littera-Backup (7.2).
- Band „Keine Verbindung": Der Herzschlag-Wächter (`App.svelte`, 25 s ohne `ping`) unterscheidet
  nicht zwischen „kein Ping gekommen" und „der Tab selbst stand" (Standby, eingefrorener
  Hintergrund-Tab). Beim Aufwachen wäre der Herzschlag alt und das Band stünde bis zum nächsten
  Ping, höchstens 15 s. Nicht nachgestellt.
- Schreibweise der Nummern (entschieden am 23.09.2026: am Server nichts bauen). Der Server
  schlägt Nummern exakt nach (`GetLeserByBarcode`, `GetCopyByBarcode`); die Theke
  vereinheitlicht vorher (`normalisiereScan`, nur mit Ziffer hinter der Vorsilbe) — beim
  Buchen und in der Offline-Warteschlange. Roh fragt nur die Vorschau beim Tippen
  (`/api/search`). Gemessen am Testserver am 23.09.2026: alle gespeicherten Vorsilben groß
  (40 Leser, 4.130 Exemplare, keine klein). **Anlass zum Bauen:** ein zweiter Aufrufer, der
  rohe Nummern schickt — dann die Eingabe am Server vereinheitlichen (nicht `upper(barcode_id)`,
  das nimmt den Index), mit einer gemeinsamen Fall-Tabelle für Go und JS.
- Die Sperrprüfung liest aus dem Pool, während die Checkout-Transaktion mit `FOR UPDATE` offen ist
  (drei Abfragen über eine zweite Verbindung). Bei `MaxConns = 50` ohne Wirkung; beim Nachbuchen
  vieler Ausleihen (Abschnitt 2) beobachten.
- `golang.org/x/crypto/openpgp` (`GO-2026-5932`): kein Fix verfügbar, transitiv, kein Aufrufer im
  eigenen Code.
- Designer: Wer den Browser binnen 800 ms schließt, verliert die letzte Auto-Save-Änderung;
  `sendBeacon` bewusst nicht gebaut.
- Migration 082 (Dedupe der Vormerkungen) verlor den neueren `abholbereit`-Eintrag; auf Prod
  gelaufen, nur relevant für eine weitere gewachsene Datenbank.
- Die Rate-Limiter-Maps räumen erst ab 5.000 Einträgen; nur mit vielen frischen Adressen ein
  CPU-Thema.
- `startGDPRWorker` (`main.go`) ruft nur die Leihen-Anonymisierung und die Abgänger-Löschung;
  `RunGDPRAnonymizeOldData` läuft allein im Cron. Keine Wirkung erkennbar, die Doku beschreibt
  beide Wege.
- Portal: Das Menü „Mein Portal" verlangt `create_reservations`, `GET /api/lmf-termine` nur eine
  Sitzung — bewusst, der Inhalt ist PII-Stufe 0.
- Ausfallmatrix A3 und B4; A3 erst nach S3 (7.3).

### 6.2 Kategorie C

- Die Ferientabelle 2027–2030 (`pkg/lmfplan/ferien.go`) ist eine ungeprüfte Abschrift; der
  Horizont-Test wird ab 2029 rot. Beim nächsten KMK-Beschluss gegen die Quelle prüfen.
- Browser-Gates: Die M3- und axe-Gates öffnen die Planer-Dialoge nicht, axe misst nur den
  Anfangszustand; kein Screenreader-Durchgang; der Ausweis-Designer geht nur per Maus.
- 16 Bestandsstellen bauen ihr Cover selbst (Liste in `frontend-hygiene-cover.test.js`, darunter
  `KlassenBuchKachel` im Portal). Umstellen beim fachlichen Anfassen, nicht in einem Rutsch.
- 3.000 Titel ohne ISBN: `inventur.SucheTextDNB` nur mit Bestätigung durch einen Menschen
  verdrahten.
- Die Altersangabe der DNB (653 „(Zielgruppe)ab 10 Jahre", `MetadatenErgebnis.Zielgruppe`) wird
  gelesen und nicht gespeichert: Es gibt keine Spalte und keinen Leser. Anlass zum Bauen: ein
  Leser, etwa ein Filter im Portal.
- `TestEtikettenkette_ZaehlerFolgtDenFilternDerListe` schickt `?bis=time.Now()` in der Zone des
  Testprozesses; unter `TZ=Pacific/Midway` ist das der Vortag und der Zähler nennt 0. Kein
  Produktfehler — im Betrieb kommt dieses Datum aus dem Browser in Berlin. Beim nächsten Anfassen
  auf `schulzeit.Zone()` umstellen (gefunden am 18.09.2026, als die volle Suite einmal unter einer
  fremden Prozesszone lief).
- Zwei Regexe für die LMF-Kennung (`pkg/lmf/lmf.go`, `internal/service/import_lmf.go`).
- `migrations/110_schadensersatz_bescheide.sql` nennt drei Barzahlungs-Briefe, es sind zwei;
  eingespielte Migrationen bleiben unverändert.
- Knopfzeile über den Reitern (Mahnwesen): kommt aus dem gemeinsamen Seitengerüst; Anlass wäre ein
  Rundgang über das Gerüst.
- Drei handgebaute Pillen-Gruppen in `StatsDashboard` statt `ui/Segmente.svelte`.
- Schriftstärke der Chips: `ui/ChipFeld`, `ui/FilterChips` und `ui/Segmente` schreiben
  `font-medium`, das im Haus 400 ist (`styles/theme-mass.css`; an `FilterChips` im Browser
  gemessen am 23.09.2026, die beiden anderen tragen dieselbe Klasse). M3 nennt für label-large
  500, die Knöpfe tragen `font-semibold` (500). Alle drei zusammen entscheiden, nicht einzeln.
- Etikettenraster doppelt (`api/label_formats.go` und `etikettformate.js`), gehalten von
  `etikettformate-konsistenz.test.js`; am 31.08.2026 entschieden geparkt.
- Reste des Nie-verdrahtet-Sweeps: `inventur_sessions.gestartet_von` wird nie angezeigt;
  `abgaenger_jahr` in der Aktivlisten-Antwort ohne Konsument; bei den Geräten
  `ActionEvent.GeraetID` ohne Broadcast und mit Null-Zeitstempel.
- Cognitive Complexity: 32 Funktionen über 15 ohne Tests (Messung 05.09.2026); lohnend allenfalls
  `OverrideDueDateHandler` und `behandleAbgaenger`.
- `javascript:S6551` und `javascript:S8783`: begründete Dauer-Ausnahmen.
- `auth.Claims.BarcodeID` liest niemand mehr; die Ausweisnummer kommt seit Migration 125
  als LEFT JOIN aus der Leserzeile in die Sitzung, nur damit das Feld gefüllt bleibt.
- Tabellen-Inline-Felder mit 36 px: eine `size="sm"`-Variante von `Feld` erst bei Bedienbefund.
- `LabelHeight >= 30` steht zweimal (`api/label_pdf.go`, `api/schueler_etikett_pdf.go`).
- Zwei Normalformen für Namen (`repository.Suchnorm`, `normName` in `api/lusd_paarung.go`); beim
  Anfassen der Paarung zusammenführen.
- Der Paritätstest vergleicht keine COMMENTs und Seeds.
- Erbe der PR-Zulieferungen: Go-Testdateien über 200 Zeilen, ein schwacher Export-CSV-Test.
- Klone: Go 9 Gruppen (05.09.2026), Frontend 0,41 %.
- 52 Handler-Dateien in `api/` formulieren rohes SQL neben `repository/`; der Bestand ist seit dem
  07.08.2026 eingefroren (`handlerMitSQL` in `api/schichtung_test.go`). Umstellen beim fachlichen
  Anfassen einer Datei, nicht in einem Rutsch.

### 6.3 Parkdeck (bewusste Nicht-Entscheidungen)

Integer-Cent statt float64 · Bundle-Splitting · TypeScript-Migration · `inventur/` ins Haupt-API
verschmelzen · `cmd/migrate` (MySQL) löschen — seine PG-Tests sichern mit `internal/uebernahme`
geteilten Code · API-Versionierung · Mandantenfähigkeit (RLS) · Trennlinien-Durchgang (25 Dateien
mit `divide-y`, nur als eigener Durchgang mit Messung im Browser) · Zugangsbuch-Ausdruck je
Schulhalbjahr und Topf · Bestandskartei-Ausdruck zum 15.3. und 15.9. (beides nennt
[mittel_konzept.md](mittel_konzept.md), Abschnitt 7.1, als Verfahrensvorgabe; nicht gebaut) ·
ein Schüler wird Lehrkraft (die Datenbank verbietet es, `chk_leser_nur_schueler_werden_abgaenger`;
heute ein zweiter Leser, bei Häufung ein Umzugspfad wie Migration 072) · der Littera-Personenlauf
übergeht Praktikanten, Sekretariat und „Im Ausland" (`internal/littera/leser.go`).

---

## 7. Betrieb (liegt bei der Schulseite)

Geplante Zielumgebung ist der Schulserver; heute ist der Hetzner-Server die einzige Instanz. Beim
Umzug gilt dieser Abschnitt dort erneut — ebenso das, was am Hetzner-Server schon erfüllt ist
(Commit-Geschichte, 13.09.2026).

### 7.2 Frisches Littera-Backup

`littera_sav.mdb` ist ein Stand von 2010. Ohne die offenen Ausleihen startet das System mit „alles
verfügbar"; Ausweisnummern und Exemplare nach 2010 fehlen, rund 1.350 Titel haben keine Signatur.
Anforderungen in [littera_schema_befund.md](littera_schema_befund.md). Vorher B7 (8.5). Die
teuerste offene Position vor dem Echtstart — früh bei der Schule anfragen.
**Reihenfolge:** Littera-Personen und -Ausleihen im selben Lauf übernehmen, **bevor** ein echter
LUSD-Import läuft. Der Littera-Personenlauf erkennt Schüler aus der LUSD nicht und legt sie ein
zweites Mal an ([SCRIPTS.md](SCRIPTS.md), Abschnitt 0). Das Geburtsdatum im Backup ist die Brücke
für den späteren LUSD-Abgleich — vor dem Lauf prüfen.
**Vor dem Personenlauf:** Im frischen Backup nachsehen, ob die Tabelle `FremdLeserNummer` gefüllt
ist (im Stand von 2010 ist sie leer). Sie trägt die Nummern, die die Ausweise beim Scannen liefern;
nur mit ihr funktionieren die vorhandenen Ausweise ohne Neudruck. **Ist sie leer, muss jeder Ausweis
neu gedruckt werden:** Kein Ausweis liefert beim Scannen die Lesernummer, und Littera zählt Bücher
und Leser getrennt, beide ab 1 — fast jede Lesernummer ist auf dem Server schon die Nummer eines
Buchs, der Lauf vergibt dann `L-`-Nummern. Ist sie gefüllt, stehen die einzelnen Personen ohne Karte
mit „keine Karte in FremdLeserNummer" im Protokoll des Laufs.
**Am 17.09.2026 aufgefallen:** In `~/Downloads` und auf dem Schreibtisch liegen seit dem
09.09.2026 zwei Littera-Sicherungen vom 01. und 02.09.2026 (856 KB und 602 KB). Ob das die
Datensicherung aus dem laufenden Littera ist, ist UNGEPRÜFT — für eine Vollsicherung wären
100 MB+ zu erwarten, die Größe spricht eher für einen Teilexport. Zum Nachsehen fehlt auf dem
Rechner ein Entpacker für `.7z`.

**Rückweg zu Littera:** Bücher und Schüler behalten ihre Littera-Nummer, Lehrkräfte nicht. Soll der
Rückweg offen bleiben, vor dem Lauf nachtragen und das Littera-Backup vom Umstiegstag aufheben.

### 7.3 S3-Auslagerung der Backups

`S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY` und `S3_BUCKET` sind leer (13.09.2026); alles
liegt auf einer Platte. Nur EU oder Schulträger. Der Code ist fertig.

### 7.4 Manuelle Restore-Probe an einem fremden Ziel

Die automatische Wochenprobe im Container lief am 13.09.2026 erfolgreich; sie ersetzt die
manuelle Probe nicht. Anleitung: [resilience_and_recovery.md](resilience_and_recovery.md),
Abschnitt 2e — dabei die neuen Befehle erproben, die bisher nur am Text geprüft sind. Sinnvoll
nach S3 oder am Schulserver.

### 7.5 Externes Uptime-Signal

Fällt der Server ganz aus, meldet es niemand. Ein externer Monitor ruft alle 5 Minuten `/health`
ab ([DEPLOYMENT.md](DEPLOYMENT.md), Abschnitt 7.0). Etwa fünf Minuten Aufwand; beim Umzug neu
einrichten.

### 7.6 Ruleset `main`

PR-Pflicht entfernen (Solo-Entscheidung 30.07.2026), „Block force pushes" und „Restrict
deletions" anlassen. Am 23.09.2026 trägt das Ruleset noch `pull_request`; Pushes gehen über den
Admin-Bypass.

### 7.7 Abnahmen

Ablauf in [abnahme_checkliste.md](abnahme_checkliste.md), vorher ein Backup.

- Flows 1–3 mit dem Sekretariat: LUSD-Import, Versetzung (vor dem Schuljahreswechsel),
  Klassensatz erledigen. Dabei um Geburtsdatum und Eintrittsdatum im LUSD-Bericht bitten. Ein
  LUSD-Import mit echten Schülern erst nach der Littera-Übernahme (7.2).
- Flow 4 (Altbestand-Etiketten, nicht umkehrbar) erst nach 4.8.
- Flow 5: Selbstanmeldung einer Lehrkraft.
- Danach: Ergebnis hier eintragen; bei Parser-Auffälligkeiten die echte LUSD-Datei anonymisiert als
  Testfixture sichern.

### 7.8 Am Server nachsehen (lesend, Einzeiler)

- „Neuer Text" im Ausweis-Layout: laut Serverlesung vom 13.09.2026 in keinem Wert von
  `system_einstellungen`. Bestätigen, dann erledigt.
- Sind die Admin-Konten deaktiviert? Ist `/app/uploads/fotos` leer? Gibt es Lehrkräfte mit
  Platzhalter-Mail `@lehrer-umzug.invalid`? Braucht `repair_fach_kategorie.sql` einen zweiten
  Lauf?

## 8. Schule, Schulamt, Schulträger

### 8.1 E1: Schulamts- und Schulnummer

Für die Referenznummer der Bescheide; die Felder stehen in den Einstellungen und sind am
13.09.2026 leer. Mit den Nummern entstehen echte Bescheide; was davor zu richten war (Frist,
Nachdruck, Briefdatum), ist seit dem 23.09.2026 gebaut.

**Beim Schulamt mitfragen — das Kassenjahr:** Das Programm nimmt das Jahr der Frist, ein
Dezember-Brief mit Frist im Januar zählt also ins Folgejahr. Die Arbeitshilfe nennt das
Kassenjahr als Teil der Referenznummer, sagt aber nicht, welches Jahr gemeint ist, und verweist
„im Zweifelsfall" ans Schulamt. Die Nummer lässt sich nach dem Brief nicht mehr ändern.

### 8.2 E2: E-Mail-Erlass vom 11.06.2018 und aktuelles Musterschreiben

Die vorliegenden Unterlagen sind von 2014 und nennen eine inzwischen aufgelöste Stelle. Gebaut
ist nach dem Muster von 2014; der Text ist an einer Stelle austauschbar.

### 8.3 E5: Zahlungsweg der Schülerbücherei

Kreiskasse mit Kassenzeichen, Budgetkonto oder bar mit Quittung? Zuständig ist der Fachbereich
Schule und Betreuung des Hochtaunuskreises. **Engpass:** Ohne Antwort kein 5.4.

### 8.4 D2: Getrennte Kundenkonten beim Händler?

Führt der Händler getrennte Kundenkonten für Lernmittel und Schülerbücherei? Das Feld
„Kundennummer Schülerbücherei" am Lieferanten bleibt bis zur Antwort leer.

### 8.5 Datenschutz Teil B

Einzelheiten und Entwürfe in [datenschutz_offene_punkte.md](datenschutz_offene_punkte.md),
Abschnitt B, und in [datenschutz/](datenschutz/).

- **B1/B2** VVT und Datenschutzhinweis beschließen; die Entwürfe haben noch Platzhalter. Vorher den
  Zweck „Schadensersatz-Bescheid" ergänzen (5.4).
- **B3** Foto auf dem Schülerausweis: Erlass oder Einwilligung — vor dem ersten Ausweisdruck mit
  Foto.
- **B4** Schulischen DSB beteiligen, Schwellwertanalyse DSFA schriftlich — vor B1/B2.
- **B5** IT-Sicherheitskonzept mit dem Schulträger, Netzplatzierung.
- **B6** Rolle des Wartenden regeln (AVV oder dienstlich).
- **B7** Löschkonzept gegenüber Littera — vor der Littera-Übernahme (7.2).

Zuerst B3 und B4 anstoßen.

### 8.6 Barrierefreiheit

Gilt für das System die Pflicht zur Barrierefreiheit — mit Erklärung zur Barrierefreiheit und
barrierefreien PDFs (HTML-Druckweg oder begründete Ausnahme)? Bis zur Antwort geparkt; was die
Gates heute prüfen, steht in [FACHKONZEPT.md](FACHKONZEPT.md), Abschnitt 19.

---

## 9. Sichtung vom 16.09.2026

Zwölf Punkte, jeder am Code geprüft. Offen ist nur, was hier folgt; das Übrige steht in den
Commits vom 17. und 22.09.2026, die drei begründeten Abweichungen im Mahnwesen (nur Post, nie
löschen, vier statt sechs Wochen) in [mittel_konzept.md](mittel_konzept.md) Abschnitt 3.

Zwei Quellen liegen dem zugrunde: `~/Downloads/Arbeitshilfe_Mahnschreiben.pdf` (Erlass vom
17.12.2014, Az. 674.100.002-00178) und `~/Downloads/Ablauf Mahnverfahren.pdf` (die
Anforderungsliste, abgeglichen in [mittel_konzept.md](mittel_konzept.md) Abschnitt 3).

### 9.3 Vorgaben des Landes (Protokoll 1)

**9.3 c) Sperrung bei offener Bearbeitung — eine Frage bei dir.** Nach der Antwort der Schule
vom 22.09.2026 darf ein Lernmittel nie automatisch gesperrt werden; für die Schulbibliothek gilt
das nicht. Offen: Ein Kind mit überfälligen Lernmitteln, das ein Buch der Schülerbücherei will —
die Automatik weist es heute ab, weil sie Medien zählt, nicht Töpfe. Ob die Bücherei wegen
überfälliger Schulbücher zumacht, entscheidet die Schule selbst. Vorschlag: so lassen, es ist
eine Einstellung (`MaxOverdueItems`).

### 9.9 Zwei Bedingungen neben der Mängelliste

**Entschieden am 23.09.2026: zurückgestellt.** Beides bleibt liegen, bis es ansteht; dann gelten
die Schritte und Fragen unten.

Die Einschätzung am Ende des Protokolls nennt zwei Punkte, die in keinem der zwölf Mängel
stehen:

> „Ein Nachweis der DSVGO-Konformität liegt nicht vor.
> Hosting- und Programmpflegekonzepte sind nicht geplant. Dies könnte ein Ausschlusskriterium
> sein."

- **Nachweis der DSGVO-Konformität.** Das Material liegt vor und ist vollständiger, als der Satz
  vermuten lässt: VVT-Entwurf und Datenschutzhinweis ([datenschutz/](datenschutz/)), die
  PII-Matrix über jede Route ([PII_MATRIX.de.md](PII_MATRIX.de.md)), die Löschfristen samt
  nächtlichem Job. Was fehlt, ist ein Dokument, das man weitergeben kann — und die
  Beschlussfassung der Schule (8.5, B1–B7). **Nächster Schritt, bei mir:** das Dokument aus
  diesem Material zusammenstellen; Arbeit an der Doku, keine Bauarbeit.
- **Hosting- und Programmpflegekonzept.** Hier fehlt wirklich etwas. Betrieb, Sicherung,
  Wiederherstellung und Aktualisierung sind beschrieben — aber als Anleitung für den Betreiber
  (DEPLOYMENT.md, resilience_and_recovery.md, SCRIPTS.md), nicht als Konzept, das jemand prüft.
  **Nächster Schritt, bei mir:** ein Entwurf mit dem, was der Code beantwortet — wie eine
  Aktualisierung zur Schule kommt (Release, Image, `update.sh`), Sicherung, Wiederherstellung.
  **Vier Fragen bei dir**, die sich nicht aus dem Code beantworten lassen:
  1. Wer betreibt das Programm — die Schule, der Schulträger oder du?
  2. Wer pflegt es?
  3. Was gilt, wenn die Pflege endet — Datenausgabe, Rückweg zu Littera (7.2)?
  4. Soll es über die eigene Schule hinaus eingesetzt werden?

**Beides ist Voraussetzung für ein „nutzbar", nicht Beiwerk.**
