# Offene Arbeit

Stand: 22.09.2026

**Die eine Liste.** Hier steht alles, was noch zu tun, zu prüfen oder zu entscheiden ist — Code,
Betrieb und Schule. Einen zweiten Ort gibt es nicht. Erledigtes wird gelöscht, nicht archiviert:
Die Geschichte steht in den Commit-Nachrichten und in `git log -p docs/OFFEN.md`.

Andere Dokumente erklären (Konzept, Anleitung, der Katalog der Bugklassen in
[sweeps.md](sweeps.md)), führen aber keine eigene Offen-Liste.

---

## Was jetzt dran ist

Dieser Block nennt die Reihenfolge. Alles darunter ist die ausführliche Fassung mit
Begründungen.

**Vorrang hat die Sichtung vom 16.09.2026** (Abschnitt 9): zwölf Punkte, sieben davon erledigt.
Die Antwort der Schule vom 22.09.2026 hat drei Punkte entschieden, die jetzt zu bauen sind
(9.3 c, 9.4, 9.6). Daneben stehen zwei Bedingungen neben der Mängelliste — der Nachweis der
DSGVO-Konformität und ein Hosting- und Pflegekonzept (9.9).

**Was bei dir liegt — der Reihe nach:**

1. **Die Antwort der Schule ist da (22.09.2026) — alle drei Dinge sind gebaut.** Keine
   automatische Sperre für Lernmittel, auch keine übergehbare (9.3 c). Titel ohne Exemplare
   stehen in keinem Katalog mehr (9.4). Mehrjahresbände am Werk, das Schuljahr steckt in der
   Frist (9.6). Offen ist dein Blick auf die zwei neuen Bedienstellen (Umschalter „Mit
   Exemplaren | Ohne Exemplare" in der Titel-Verwaltung, Schalter „Mehrjahresband" im
   Buchformular).
2. **Die sechs Fragen sind beantwortet und gebaut** (21.09.2026: 4.7, 4.9, 4.11, 4.13, 4.14,
   4.21). Von 4.8 steht nur noch der Lauf im Druck-Center aus — er ändert Daten und liegt bei
   dir (Stichtag 15.07.2026, gemessen). Neu seit 4.21: In den Einstellungen unter
   „Schule" steht das Feld „Eigentumsvermerk Schülerbücherei" — leer heißt, diese Bücher
   tragen keinen Vermerk. Den Wortlaut kennt nur die Schule.
3. **Die Zahlen vom Testserver liegen vor** (21.09.2026): 8 Leser ohne Ausweisnummer, alle
   Lehrkräfte (5.16 C); kein Kollege in einer Warteschlange, in der er nie nachrückt (5.19).
   Die Messung aus 4.3 ist gelaufen (22.09.2026: kein Titel mit Wert); 4.3 ist gebaut und weg.
4. **Zwei Umbauten freigeben**, die vorbereitet, aber nicht gebaut sind, weil sie die Datenbank
   ändern: die Ausweisnummer schon beim Anlegen eines Kontos (5.16 C) und die eigene Spalte für
   die Karenz-Uhr (4.12). Beide sind entschieden, beide brauchen eine Migration, die Nummern
   bzw. Daten schreibt, die niemand zurücknimmt.
5. **Der Nachweis von Hand für die Theke ohne Netz** (Abschnitt 2): Netz kappen, Bücher aller
   Formen und zwei Ausweise scannen, 20 Minuten warten, Netz zurück, Meldungen ansehen. Dazu
   der Nachweis für den Server.
6. **15 Minuten durch die Leserdatei gehen** (5.16 A): Stimmen die Wörter, fehlt etwas?
7. **Schulbücher in neuer Auflage** (4.18). Das Feld „Auflage" und die Dublettenkontrolle beim
   Anlegen sind seit dem 17.09.2026 gebaut. Offen ist die dritte Stufe, das Werk über den
   Auflagen — sie ändert das Schema und braucht eine Freigabe — und die Frage, wer ein Werk
   anlegt.
8. **Liegt bei anderen** (Abschnitt 8): die Anfragen an Schule, Schulamt und Schulträger. Hier
   ist nichts zu tun außer nachzufragen, wenn nichts kommt.
9. **Aus dem Abgleich mit Littera** (4.19, 4.20): **Ferienkalender** — heute mahnen wir das Kind,
   dessen Frist in die Herbstferien fiel. Richtung und Form sind am 18.09.2026 entschieden
   (Ferien als iCal-Datei, Fristen rutschen mit); gebaut ist nichts, und eine Frage steht noch
   offen (Feiertage als zweite Datei oder gerechnet). Dazu die Frage, ob die Schülerbücherei eine
   Themensuche bekommt (4.20).

**Beim nächsten Aufspielen erweitert sich die Datenbank** (Migrationen bis 135). Das passiert
beim Start von allein; Daten gehen nicht verloren, nachgetragen wird nichts.

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
   sie umgesetzt ist.
4. **Die Reihenfolge** wird bei jeder Änderung mitgepflegt.

---

## Reihenfolge

1. **Der Etiketten-Lauf im Druck-Center** (4.8): Stichtag 15.07.2026, gemessen.
2. **Die zwei Migrationen nach der Freigabe**: Ausweisnummer beim Anlegen des Kontos (5.16 C,
   gemessen: 8 Lehrkräfte ohne Nummer) und die Karenz-Spalte (4.12).
3. **Abschnitt 2** Offline-Betrieb der Theke: gebaut, samt Meldungsliste und Doku. Offen sind
   nur noch die Nachweise am Stack (2.3) — Stufe 1 und 3 von Hand, Stufe 2 über die Tür.
4. **5.16** Leserdatei: gebaut. Offen ist dein Blick auf den Stand (A) und die Ausweisnummer (C).
5. **5.5** (Rückschreiben nach der Messung) und **5.19** — kleine B-Commits (5.6, 5.12 und 5.17 am 22.09.2026 erledigt).
6. Mahnverfahren: Vor dem ersten echten Bescheid **5.2**, dann **4.4** (E6) und
   **5.13** Stufe 3 (5.3).
7. Nach der Antwort zu E5 (**8.3**): **5.4**.
8. **5.10** (Gates und Werkzeuge) und Abschnitt 6 nur mit Anlass. Schlüsselwechsel,
   Littera-Import-Hülle und `cmd/seed` sind seit dem 21./22.09.2026 abgedeckt.

**Parallel auf der Schulseite:** Abschnitte 7 und 8 — zuerst S3 (7.3), das Littera-Backup (7.2), die
Anfragen E1, E2, E5 (8.1–8.3), B3 und B4 (8.5) und ein Termin für die Abnahmen (7.7). Einen echten LUSD-Import erst nach der
Littera-Übernahme (7.2).

---

## 2. Offline-Betrieb der Theke — der Nachweis steht aus

Gebaut sind alle drei Stufen (15./16.09.2026): Die Theke hält die Buch-Barcode-Liste im Browser
und nimmt jede Scan-Form offline an, der Sync schickt an `POST /api/action/nachbuchen`, und was
nicht durchging, steht als Meldungsliste am Band. Wie sich das verhält, steht in
[FACHKONZEPT.md](FACHKONZEPT.md) 18.4 und im [Handbuch](HANDBUCH.md).

Zwei Dinge sind offen.

**Der Nachweis von Hand (2.3).** Gebaut heißt nicht geprüft: Am 16.09.2026 stand zweimal
„gebaut", weil Commits vorlagen — angeschlossen war beide Male nichts. Stufe 1 und 3 gehören
von Hand in den echten Chrome, Stufe 2 über die Tür.

**Der Server nimmt eine Nummer nur in der Schreibweise an, in der sie gespeichert ist.**
`internal/service/omnibox_service.go` vergleicht die Vorsilben mit `strings.HasPrefix` gegen
`"A-"`, `"S-"`, `"L-"`, `"B-"`, `"G-"`, und `GetLeserByBarcode` schlägt mit
`WHERE barcode_id = $1` exakt nach. Die Theke vereinheitlicht seit dem 16.09.2026 selbst
(`frontend/src/lib/scanEinordnen.js`, `normalisiereScan`) — nur wenn hinter der Vorsilbe eine
Ziffer steht, sonst würde aus der Suche nach „s-bahn" ein Ausweis. Ein anderer Aufrufer
(Skript, zweite Oberfläche, direkter Aufruf) läuft weiterhin ins Leere. Ob das am Server
geheilt wird, ist zu entscheiden: Eine Suche über `upper(barcode_id)` nutzt den vorhandenen
Index nicht mehr, und der hält die Eindeutigkeit der Ausweisnummern.

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

Entschieden am 21.09.2026: über das Druck-Center („Fehlende Etiketten" → „Altbestand
aufräumen", mit Vorschau und Stichtag); `scripts/repair_altbestand_etiketten.sql` ist gelöscht.

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

### 4.12 Karenz gegen Lesehistorie

Die Uhr vor der Anonymisierung rechnet ab der letzten Rückgabe über `ausleihen.schueler_id`, die
der Lesehistorie-Lauf trennt. Ist die Karenz länger eingestellt als die Lesehistorie, wird früher
anonymisiert als eingestellt. Es gibt zwei Lesehistorie-Fristen: Schülerbücherei (Vorgabe 90 Tage)
und Lernmittel (`lesehistorie_lernmittel_tage`, Vorgabe 730); mit den Vorgaben (Karenz 90) ohne
Wirkung. **Frage:** Soll die Einstellung Karenz ≤ beide Lesehistorie-Fristen erzwingen, oder
speichert die Uhr ihren Zeitpunkt selbst (eigene Spalte)?

**Entschieden am 16.09.2026: eigene Spalte, per Trigger gepflegt.** Nicht erzwingen — die
Kopplung liefe in die falsche Richtung: Um eine längere Karenz zu bekommen, müsste die Schule
die Lesehistorie verlängern, also mehr Personendaten länger aufbewahren. Die beiden Fristen
beantworten verschiedene Fragen und dürfen sich nicht gegenseitig binden.

**Gebaut wird erst nach deinem Ja (17.09.2026)** — die Umsetzung ist eine Migration mit zwei
Triggern und einer Rückfüllung über den ganzen Bestand. Der Plan steht, der Nachweis ist
beschrieben; was fehlt, ist die Freigabe für einen Eingriff ins Schema.

Stattdessen bekommt die Leserzeile eine Spalte „letzter Vorgang", die kein anderer Löschlauf
anfasst; gepflegt von Triggern auf `ausleihen` (Rückgabe) und `schadensfaelle` (bezahlt oder
storniert), nach dem Muster von `konto_hat_leserzeile` — an der einen Stelle, an der kein
Schreibweg vorbeikommt. `repository.KarenzUhr` (seit 22.09.2026 die eine Formulierung für
Löschuhr und Wächter) rechnet dann `GREATEST(AbgangSeit, letzter_vorgang_am)` ohne Unterabfragen. Nachweis: ein PG-Test mit Karenz > Lesehistorie, der beweist, dass der
Lesehistorie-Lauf den Anonymisierungs-Zeitpunkt NICHT verschiebt — am alten Stand rot.

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

**Wie Littera es macht** (Handbuch, am 17.09.2026 gelesen): Jede Auflage ist ein eigener Titel mit
eigenem Feld „Auflage" („z. B. 3. Aufl."). Beim Anlegen läuft eine Dublettenkontrolle über die
ISBN — ohne ISBN über Verfasser und Haupttitel — und bietet an, statt eines neuen Titels ein
weiteres EXEMPLAR anzulegen. Verbunden werden Auflagen über eine Verweisung „Früherer Titel",
damit die Recherche beide findet. Ein Zusammenführen von Titeln gibt es nicht, eine Ebene über dem
Titel auch nicht. Das ist allerdings die BIBLIOTHEKS-Software; die Lernmittelverwaltung ist bei
Littera ein eigenes Programm. Littera beantwortet also „finde beide Auflagen", nicht „wie viele
Mathe 7 haben wir".

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

**Offen: Wer legt das Werk an?** (Am 17.09.2026 ohne Meinung geblieben.) **Vorschlag:** Der
Vorschlag entsteht automatisch, das Ja kommt von einem Menschen. Vollautomatisch über den Namen zu
gruppieren verbindet früher oder später zwei „Deutschbuch 7" verschiedener Verlage; rein von Hand
pflegt es niemand, und der Bedarf bleibt falsch. Auf dem Weg über die Nachbestellung ist der
Vorschlag praktisch sicher, weil er von einem konkreten Titel ausgeht.

**Stand:**

1. **„Auflage" als Feld am Titel — gebaut am 17.09.2026** (Migration 126, `553efd7c`): Spalte,
   Feld in der Maske, Anzeige in Katalogkarte und Buch-Akte; der Listenimport leert sie nicht.
2. **Dublettenkontrolle beim Anlegen — gebaut am 17.09.2026** (`e5fa346c`): in der Maske beim
   Anlegen und Ändern, über die ISBN in jeder Schreibweise, ohne ISBN über Titel und Autor; eine
   gefüllte Auflage hebt den Verdacht auf. Die Importe (Excel, Liste, ISBN-Abruf, Littera)
   verlassen sich weiter auf `ON CONFLICT (isbn)`, also zeichengleich — das ist 5.5.
3. **Das Werk** samt Migration, Gruppierung im Bedarf und Warnung in der Ausgabe — **nicht
   gebaut.** Ändert das Schema und rechnet die Nachbestell-Liste anders. Das geht in Stufen mit
   Nachweis und erst nach deiner Freigabe.

### 4.19 Frist fällt in die Ferien

**Der Fall.** Ausleihe am letzten Schultag vor den Herbstferien, Frist 21 Tage: fällig mitten in
den Ferien. Das Kind kann nicht zurückgeben, steht danach in der Mahnliste und ab einem
überfälligen Medium mit gesperrtem Ausweis an der Theke (`MaxOverdueItems`, Vorgabe 1).

**Was heute passiert.** `calculateDueDate` (`internal/service/loan_rules.go`) rechnet
Kalendertage: Stichtag bei Lernmitteln, feste Tageszahl sonst. Kein Kalender geht ein. Der einzige
Behelf ist der Ferien-Leseclub — ein festes Zieldatum für ALLE Ausleihen, von Hand ein- und
auszuschalten.

**Wie Littera es macht** (Handbuch 5.4.18, Kapitel *Grundeinstellungen für Verleih* und *Verleih*,
gelesen am 17.09.2026): zwei Einstellungen. *Öffnungstage* sind die Wochentage, an denen die
Bibliothek geöffnet hat. *Schließtage* sind einzelne Tage mit Bezeichnung, wahlweise als Zeitraum
eingetragen („Sommerferien, Inventur, Betriebsurlaub"), dazu ein Knopf, der die gesetzlichen
Feiertage eines Landes für ein Kalenderjahr übernimmt — jedes Jahr neu zu drücken. Fällt die Frist
auf einen Schließtag, „errechnet LITTERA automatisch den nächsten möglichen Rückgabetermin, trägt
diesen dann ein und stellt den Zeitraum zwischen errechnetem und tatsächlich möglichem
Rückgabetermin mahn- und gebührenfrei". Einstellbar ist, ob bis zum nächsten Öffnungstag oder bis
zum nächsten gleichen Wochentag verschoben wird.

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

Eine Tabelle `ferien_schliesszeiten` gab es; Migration 102 hat sie am 06.09.2026 ausgebaut, weil
sie nie einen Schreiber bekam und die automatische Mahnpause vor nichts Realem schützte. Das galt
dem Mahnwesen. Hier geht es um den Schreibpfad der Frist — um die Zahl, die auf dem Kontoauszug
steht und die das Mahnwesen später liest.

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
- **Ein Warner**, wenn der Kalender ausläuft: in der Betriebsbereitschaft, wo schon die
  Ferientabelle steht. Ein Kalender, der still endet, rechnet ab dem ersten fehlenden Tag wieder
  falsch, ohne dass es jemand merkt.

**Beim Mitrutschen gilt dieselbe Ausnahmeliste wie beim LMF-Plan:** nur offene Ausleihen, nur nach
hinten, und nicht angefasst werden von Hand gesetzte Fristen, Lernmittel mit Termin aus dem Plan
und die Fristen des Ferien-Leseclubs. Jede Verschiebung steht im Protokoll, und die Meldung nennt
die Zahl der betroffenen Ausleihen.

**Offen — zwei Dateien oder eine?** Vorgeschlagen war je eine Datei für Ferien und für Feiertage.
Für die Ferien ist das richtig. Bei den Feiertagen rate ich ab: Die rechnet das Programm bereits
exakt für Hessen (`pkg/lmfplan/feiertage.go`, Fronleichnam eingeschlossen), ohne Pflege und ohne
Ablaufdatum. Eine hochgeladene Datei daneben wäre eine zweite Quelle für denselben Zustand — die
teuerste Bugklasse dieses Projekts: Wer die Datei eines anderen Bundeslandes erwischt, verschiebt
Fristen auf einen Tag, an dem die Schule offen hat, und niemand sieht warum. *Vorschlag:* die
zweite Datei annehmen, aber nur als **Gegenprobe** — das Programm vergleicht sie mit seiner
Rechnung und meldet Abweichungen, statt sie zu übernehmen. Das kostet wenig und fängt genau den
Fall, in dem unsere Rechnung falsch wäre.

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

**Reihenfolge, wenn freigegeben:** (1) Kalender-Tabelle samt Übernahme der Sommerferien,
(2) iCal-Upload mit Vorschau, (3) Frist rechnet gegen den Kalender, (4) Mitrutschen bei Nachtrag,
(5) Warner in der Betriebsbereitschaft.

**Nicht gebaut.** Ändert einen Schreibpfad und braucht eine Migration; Gate am Rückweg, vorher rot
gesehen.

### 4.20 „Ich will was Gruseliges" — der Bestand hat kein Thema

**Was heute da ist.** Am Titel stehen Fach (Systematik), Signatur, Jahrgang, Schulzweig und die
Beschreibung, die im Volltext mitsucht. Das Fach ist die Schulfach-Systematik: für Lernmittel
richtig, für die Schülerbücherei nicht — „Jugendliteratur" ist kein Thema.

**Wie Littera es macht:** Schlagworte (frei, mit Verweisen für Schreibvarianten und Pseudonyme)
und daneben Interessenkreise — „eine thematische Gliederung des Belletristikbereiches (z. B.
Krimi, Heimat…)". Beides ist in der Recherche filterbar und als Liste auswertbar.

**Was dabei heute verloren geht:** Der Littera-Altbestand bringt Schlagworte mit (MAB 710). Der
Import liest sie, leitet daraus das Fach ab und wirft sie danach weg
(`internal/service/import_service.go`); gespeichert wird keines. Wer sie haben will, spielt den
Import gegen das frische Backup neu ein (7.2).

**Fragen:** Braucht die Schülerbücherei die Themensuche im öffentlichen Katalog — und wer pflegt
sie bei Neuzugängen? Ein Feld, das nach dem Import nie wieder gefüllt wird, kennt nach zwei Jahren
nur noch die Vergangenheit.

**Vorschlag:** Erst am Backup messen, wie viele Titel überhaupt brauchbare Schlagworte tragen;
darunter erübrigt sich die Frage. Trägt der Altbestand, dann ein geschlossenes Vokabular von
zwölf bis zwanzig Wörtern statt freier Schlagworte — als Auswahl am Titel und als Filter im
Katalog und im Portal. Freie Schlagworte brauchen Normdatenpflege (Littera hat dafür ein eigenes
Modul); die haben wir nicht.

**Stand 22.09.2026 — entschieden: bauen.** Die Quelle für das Vokabular ist die DNB, nicht
der Altbestand: Jeder DNB-Satz trägt drei geschlossene Vokabulare, die der Code heute
verwirft — die Warengruppe des Buchhandels (653 `(VLB-WN)`, z. B. „1250 Kinderbücher bis
11 Jahre", „1120 Belletristik/Spannung"), die Thema-Kategorien (655 `$2 gatbeg`, z. B.
„Spannung") und die Sachgruppe (082/084, K = Kinder- und Jugendliteratur). Die freien
Verlagsschlagworte in 653 („Gruselgeschichte", aber auch „Minecraft Buch Kinder") sind
Suchmaschinentext und werden nicht gespeichert; der Klappentext (856 Inhaltstext) wird
nicht übernommen. Google Books fällt als Themenquelle aus: ohne API-Key antwortet es mit
429 (Tageskontingent geteilt), und seine Kategorien sind englische Grobklassen.

Stufen: (1) ohne Migration — der DNB-Leser liest `(Lesealter)` neben `(Zielgruppe)` und
liefert aus den drei Vokabularen einen Themen-Vorschlag; (2) Migration — ein Feld „Thema"
am Titel mit geschlossenem Vokabular (Vorschlag aus der ISBN-Suche, Auswahl in der Maske,
Filter im Katalog und im Portal). Vor Stufe 2 zu klären: die Wortliste (zwölf bis zwanzig),
ein Thema je Titel oder mehrere, und wer es bei Titeln ohne DNB-Treffer pflegt.

---

## 5. Abarbeitbar (Kategorie B)

### 5.1 Schäden und Benutzer

- **Was „Bezahlt" bedeutet** (17.09.2026, Folge aus 9.3 e): Der Knopf in der Schülerakte
  verbucht heute eine Barzahlung am Tresen. Für Lernmittel des Landes sieht die Arbeitshilfe
  Bargeld nur als Ausnahme vor — mit Quittung und Weiterleitung binnen 14 Tagen. Die Briefe
  nennen inzwischen das Konto; ein eingehender Betrag kommt also in aller Regel als
  Überweisung, und niemand an der Theke sieht ihn. Zu klären mit der Schule: Wer bucht eine
  Zahlung ein, die auf dem Kontoauszug steht? Keine Bauarbeit, bevor das beantwortet ist —
  eine erfundene Antwort steht sonst als Vorgang in der Akte.

### 5.2 Bescheid — vor dem ersten echten Bescheid

- Das Kassenjahr ist das Jahr der Frist (ein Dezember-Brief zählt ins Folgejahr).
- Die Frist ist serverseitig unbegrenzt; eine vergangene Frist macht den Bescheid sofort
  übergabefähig.
- Der Nachdruck liest Bank, Aufsicht, Schulanschrift, Geschäftszeichen und Schulleitung live aus
  den Einstellungen; das Gate `TestBescheidNachdruck_BleibtDerselbeBrief` ändert nur die
  Schüleranschrift. Ein Schnappschuss ist eine Schema-Erweiterung.
- `scripts/tabula_rasa.sql` leert `schadensersatz_nummern` nicht; die Bescheide fallen über
  `TRUNCATE … schueler … CASCADE` mit. Nach Tabula rasa sind alle Bescheide weg, der Nummernkreis
  läuft weiter. Vorher festlegen, ob genau das gewollt ist (Nummern nie recyceln).
- Der Elternbrief je Schadensfall (`GET /api/schadensfaelle/{id}/pdf`, `api/pdf.go`) prüft nicht,
  ob die Forderung auf einem Bescheid steht — dann stünde dieselbe Forderung auf zwei Briefen mit
  zwei Fristen. Den Zahlungsweg nennt er seit dem 17.09.2026 richtig (9.3 e). Die Oberfläche
  öffnet ihn seit dem 15.09.2026 nicht mehr (Mahnverfahren Stufe 1); über die Adresse bleibt er
  erreichbar. Fällt mit dem Entfernen der Altbriefe (5.4) weg, sonst vorher denselben Filter wie
  bei der Ersatzforderung.

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
- **Altbriefe entfernen** (der Zahlungsweg ist seit 17.09.2026 in Ordnung, 9.3 e — es geht nur
  noch darum, ob es sie neben dem Bescheid überhaupt weiter geben soll): Elternbrief
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

- Zugangsdatum beim Anlegen außerhalb des Bestellwegs (Rasterdurchgang 22.09.2026, Frage 6):
  Handanlage, Sammelimport und Bestand-Nachziehen lassen `erworben_am` auf der Vorgabe
  `CURRENT_DATE`, der Listenimport schreibt sie selbst — alle vier in der UTC-Sitzung der
  Datenbank. Zwischen Mitternacht und zwei Uhr ist das der Vortag, und der INSERT-Trigger aus
  Migration 129 übernimmt ihn als Zugang; Migration 130 hat nur den UPDATE-Zweig (Wareneingang)
  auf die Schulzeit gestellt. Fix: Vorgabe der Spalte auf `(now() AT TIME ZONE 'Europe/Berlin')::date`
  (Migration) und im Listenimport `schulzeit.SQLHeute`. Nicht gebaut, weil Migration.
- Listenimport gegen den Nummern-Wächter (Migration 131): Trägt eine Zeile der Datei die
  Ausweisnummer eines Lesers als Buch-Barcode, lehnt der Wächter ab und der ganze Import
  bricht mit der rohen Datenbankmeldung ab (`ON CONFLICT DO NOTHING` fängt nur den Index,
  nicht die Ausnahme). Laut, also richtig — nur die Meldung nennt weder Zeile noch Weg.
  Kategorie C, bis es einmal vorkommt.
- ISBN: Seit Migration 133 (22.09.2026) bringt die Datenbank jede geschriebene ISBN an jeder Tür
  in EINE Schreibweise; Import-Zuordnung, Schnellanlage und die fünf Suchfelder vergleichen die
  Normalform. Der Altbestand ist noch nicht zurückgeschrieben: erst am Server messen, ob zwei
  Altzeilen sich nur in der Schreibweise unterscheiden (Einzeiler unten), dann eine Migration,
  die `isbn = isbn_normalform(isbn)` setzt. Ein CHECK auf den Jahrgang fehlt, „Jahrgang
  unbekannt" ist von der Vorgabe nicht zu unterscheiden — und seit dem Mehrjahresband (9.6)
  hängt eine Frist an „bis": Wer den Schalter auf einem Titel mit der Vorgabe 5 bis 10
  umlegt, bekommt die 10. Ein CHECK allein löst das nicht; eine Vorgabe „unbekannt" (NULL)
  bräuchte die drei Leser (Mahnwesen „Jahrgang", Inventur, Portal-Filter) mit.
- Ein Titel, dessen Exemplare alle im Zulauf sind, steht seit 9.4 im Katalog und in der
  Theken-Trefferliste (Zulauf zählt als vorhanden) — der Bestandssatz dort sagt aber
  „Keine Exemplare", weil er nur zählt, was im Regal oder verliehen ist. Richtig wäre
  „2 bestellt": eine dritte Zahl in `bestandSatz` und in den zwei Suchabfragen. Kein
  Schaden, nur eine Auskunft, die den Kollegen ins Regal schickt; beim nächsten Anfassen
  der Trefferliste.

  ```sql
  SELECT isbn_normalform(isbn) AS normalform, count(*) AS titel, string_agg(isbn, ' | ') AS schreibweisen
  FROM buecher_titel WHERE isbn IS NOT NULL GROUP BY 1 HAVING count(*) > 1 ORDER BY 2 DESC;
  ```

### 5.6 Schüler und LUSD

- **Ein Schüler wird Lehrkraft (oder umgekehrt) — weiter nicht möglich, und das ist Absicht.**
  Seit Migration 123 stehen alle Leser in einer Tabelle, die Art lässt sich aber nur zwischen
  Lehrkraft und LiV umstellen. Über die Schüler-Grenze verbietet es die Datenbank
  (`chk_leser_nur_schueler_werden_abgaenger`), weil ein Schüler aus der LUSD kommt und dort
  wieder auftauchen würde. Der Fall ist selten (ein ehemaliger Schüler kommt als LiV zurück);
  heute legt man dafür einen zweiten Leser an. Wenn er öfter vorkommt, ist es ein eigener,
  kleiner Umzugspfad wie Migration 072 — kein Auswahlfeld.
  Der Littera-Lauf übergeht weiter Praktikanten, Sekretariat und „Im Ausland"
  (`internal/littera/leser.go`).

### 5.7 bis 5.9 Bestellwesen · LMF und Statistik · Oberfläche

Nichts offen (Stand 21.09.2026). Die Nummern bleiben vergeben, weil Kommentare im Code auf sie
als Herkunft eines Fundes verweisen.

### 5.10 Gates und Werkzeuge

- Die Schema-Gegenrichtung ist blind für UNIQUE, Teilindizes und RESTRICT; die Schema-Parität
  vergleicht Funktionen nur am Namen.
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

### 5.11 Doku

Nichts offen (Stand 16.09.2026). Die Nummer bleibt, weil die Reihenfolge oben auf sie verweist.

### 5.13 Mahnverfahren: Stufe 3 (nach 4.4)

Das Modell seit dem 15.09.2026 (Stufen 1 und 2): Die Reiter „Alle ·
Akut fällig · Eskaliert" fragen „Wer hat Bücher zu spät?", der Reiter „Schadensersatz" fragt „Wer
schuldet Geld?". Solange das Buch als ausgeliehen gilt, steht das Kind links; sobald ein Verlust
oder Schaden gebucht ist, rechts — mit genau einem Stand und einem nächsten Schritt je Zeile. Der
Bescheid entsteht direkt aus den überfälligen Büchern; der Brief bucht ihren Verlust.

- **Stufe 3**: Folgen der Übergabe (5.3): Übergabe-PDF und Sammelliste für das Schulamt; E6 (4.4).

### 5.14 Fallengelassene Verdachte (15.09.2026)

Nachgestellt und widerlegt — damit der nächste Durchgang sie nicht noch einmal prüft:
`EmpfaengerFuerBescheid` ohne `deleted_at IS NULL` · Rückkehr storniert die Forderung, das Buch
bleibt ausgesondert · „Pool ODER Tx" nur behauptet · Rechte-Asymmetrie an den Buch-Routen
(`adminH` IST `RequireEditBooks`) · Migration 115 droppt `idx_lmf_termine_plan`.

### 5.15 Durchgang vom 15.09.2026

Nichts offen (Stand 22.09.2026): Nummernkreis und Bewegungsstempel sind mit den Migrationen 131
und 132 Regeln der Datenbank, die zwei Beobachtungen stehen in 6.1. Die Nummer bleibt vergeben,
weil Kommentare im Code auf sie als Herkunft eines Fundes verweisen.

### 5.16 Leserdatei und Rolle Leitung — was noch offen ist

Gebaut ist alles: Lesertabelle, Rolle Leitung, Theke und Leserdatei, eine Maske für jeden, die
Pflicht-Schul-E-Mail und das Zusammenführen von Kollegen.

**A. 15 Minuten: einmal durch die Leserdatei gehen.** Stimmen die Wörter, fehlt etwas? Was ein
Kollege bewusst NICHT in seiner Akte hat: Kontoauszug, Ersatzforderung und DSGVO-Auskunft — sie
gehören der Schülerarbeit und lesen die Sicht `schueler`, ausgeblendet statt kaputt.

**B. Entschieden, damit die Frage nicht wiederkommt:** Die E-Mail eines Kollegen steht am Konto
(`benutzer.email`, `UNIQUE lower(email)`) und nicht in `leser.eltern_email`; in der Akte wird sie
vom Konto gelesen und ist nachtragbar, solange keine da ist. Die Leitung sieht den Menüpunkt
„Einstellungen" und darf darin LUSD & Versetzung, Datenverwaltung, LMF-Aktionen und Lieferanten
bedienen; verschlossen sind Schule, Fristen und Mailversand (`manage_settings`) sowie Benutzer &
Rechte (`manage_users`). Der Nummernkreis bleibt unangetastet: Ohne Netz ist die Vorsilbe die
einzige Information, an der die Theke einen Buchscan von einem Ausweisscan unterscheidet.

**C. Offen: Die Ausweisnummer soll beim Anlegen des Kontos entstehen.** Heute entsteht eine
Nummer an drei Stellen — „Neuer Leser", LUSD-Import, Littera-Übernahme. Ein Konto vergibt keine
(`konto_hat_leserzeile` legt die Leserzeile ohne Ausweis an), und der Druck warnt nicht: Ohne
Nummer kommt ein kaputtes Bild und eine leere Zeile auf die Karte. Entschieden am 16.09.2026,
gebaut wird nach der Freigabe — die Umsetzung braucht eine Migration, die allen Lesern ohne
Nummer eine nachträgt. **Gemessen am Testserver am 21.09.2026:** 8 von 41 Lesern ohne Nummer,
alle 8 Lehrkräfte (8 von 9); von den 32 Schülern keiner. Die Zählung, vor der Migration auf
jeder Anlage zu wiederholen:

```sql
SELECT count(*) FILTER (WHERE barcode_id IS NULL) AS ohne_nummer,
       count(*)                                   AS leser_gesamt
FROM leser WHERE deleted_at IS NULL;
```

Bei der Umsetzung gilt: **ein Generator** (`GetNextSequence` über `leser.barcode_id`, Vorsilbe
`A-`) — eine zweite Vergabe in SQL wäre der Fehler aus Migration 068 in neuer Form. Der Hinweis
am Feld „Ausweisnummer" („Leer lassen, solange kein Ausweis gedruckt ist") stimmt dann nicht mehr
und fällt.

### 5.18 Klassen ohne Löschweg

Aus der Schärfung von Frage 12 (16.09.2026) bleibt eine Beobachtung: Für `klassen` gibt es im
ganzen Go-Code kein `DELETE` — eine vertippte Klasse steht ab dann in jeder Auswahlliste.

### 5.19 Lesepfade gegen die Sicht `schueler` — eine eigene Achse

Die Ratsche aus Frage 13 führt neun SCHREIBpfade gegen die Sicht, je mit Begründung. Gelesen wird
gegen sie in 29 Dateien; die Lese-Ratsche zählt sie seit dem 21.09.2026. An der Datenbank nachgestellt: Steht eine
Vormerkung für einen Kollegen, findet die Abfrage, die beim Rückgabe-Vorgang den Nächsten
bedient, null Kandidaten — die Zeile ist da, die Warteschlange geht über sie hinweg. Ob es solche
Zeilen gibt, zeigen zwei Zählungen:

```
SELECT count(*) FROM vormerkungen v JOIN leser l ON l.id = v.schueler_id WHERE l.art <> 'schueler';
SELECT count(*) FROM schadensfaelle sf JOIN leser l ON l.id = sf.schueler_id WHERE l.art <> 'schueler';
```

**Gemessen am Testserver am 21.09.2026: beide 0.** Es ist Vorsorge und keine Reparatur. Auf
einer Anlage mit anderem Bestand vorher neu zählen.

**Die Durchsicht ist abgeschlossen (22.09.2026).** Alle 29 Dateien, die gegen die Sicht
lesen, stehen in `docs/lesepfade_gegen_sicht_test.go` mit Begründung; die Liste der Ungeprüften
ist leer, eine neue Datei ist rot. Vier Funde sind behoben (Vormerkung nur für Schüler,
Suchleiste erkennt den Kollegen-Ausweis, Tresen-Auskunft nennt den Kollegen, Passbild-Upload
für Kollegen) — je mit einem PG-Test, der am alten Stand rot war.

**Zwei Fragen daraus:**

- **DSGVO-Auskunft** (`api/dsgvo_auskunft.go`): Für einen Kollegen gibt es sie nicht (5.16 A
  nennt das als Absicht). Auch eine Lehrkraft kann Auskunft über ihre Daten verlangen — soll
  die Auskunft für jeden Leser gehen?
- **Vormerken** lässt sich nur für Schüler — so bietet es die Oberfläche an, und seit dem
  21.09.2026 lehnt es auch die Tür ab. Soll ein Kollege vormerken können, ist das ein eigener
  Umbau über vier Lesepfade der Warteschlange, kein Schalter.

**Eine Kleinigkeit:**

- Ein negativer Listenpreis wird von der Datenbank abgelehnt; die Antwort sagt nicht, was erlaubt
  ist. Das Schwesterfeld derselben Migration nennt seinen Bereich.

### 5.20 Aus der Durchsicht von PR 631 (21.09.2026) — was offen bleibt

Der PR (nur Doku, 603 Zeilen) ist nicht übernommen und am 21.09.2026 mit Begründung
geschlossen. Was daraus am Code hielt, ist am selben Tag einzeln auf `main` gebaut
(Cover-Rezept, Anfrage-Log in Kommentar und arc42, Schlüssel-Probe in der
Betriebsbereitschaft, Stand-Gate rekursiv). Offen bleibt:

- **Anfrage-Log mit Dauer und Anfragekennung:** nicht gebaut, nur die Doku auf den Ist-Stand
  gezogen. Mehr Logzeilen am Schulserver sind eine Betriebsfrage.
- **Totalverlust des Servers ist nicht beschrieben** (gehört zu 7.4). Der Entwurf im PR ist
  nach eigener Angabe unerprobt und scheitert in Schritt 5: Das Backend hat nur benannte
  Volumes, die Sicherung vom zweiten Ort liegt also nicht im Container, und die entschlüsselte
  Datei entsteht in einem Wegwerf-Container und ist danach fort. Schreiben und an einem fremden
  Ziel durchspielen, nicht herleiten.
- **Zu 7.3:** Der zweite Ort muss kein S3 sein — ein zweiter Rechner per Kopierbefehl oder
  eine getauschte Platte tun dasselbe ohne Vertragsfrage. Eine Betriebsentscheidung.

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
- Die Sperrprüfung liest aus dem Pool, während die Checkout-Transaktion mit `FOR UPDATE` offen ist
  (drei Abfragen über eine zweite Verbindung). Bei `MaxConns = 50` ohne Wirkung; beim Nachbuchen
  vieler Ausleihen (Abschnitt 2) beobachten.
- `pg_dump` und `psql` werden im Backup und in der Restore-Probe per Namen über `PATH` aufgelöst:
  keine Shell, keine Injection, im Container ist `PATH` fest.
- `golang.org/x/crypto/openpgp` (`GO-2026-5932`): kein Fix verfügbar, transitiv, kein Aufrufer im
  eigenen Code.
- Designer: Wer den Browser binnen 800 ms schließt, verliert die letzte Auto-Save-Änderung;
  `sendBeacon` bewusst nicht gebaut.
- Migration 082 (Dedupe der Vormerkungen) verlor den neueren `abholbereit`-Eintrag; auf Prod
  gelaufen, nur relevant für eine weitere gewachsene Datenbank.
- Die Rate-Limiter-Maps räumen erst ab 5.000 Einträgen; nur mit vielen frischen Adressen ein
  CPU-Thema.
- ZAP am 05.09.2026: `style-src 'unsafe-inline'`, `csrf_token` ohne HttpOnly, „Suspicious
  Comments" — alle drei dokumentierte Entscheidungen.
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
- 17 Bestandsstellen bauen ihr Cover selbst (Liste in `frontend-hygiene-cover.test.js`, darunter
  `KlassenBuchKachel` im Portal). Umstellen beim fachlichen Anfassen, nicht in einem Rutsch.
- 3.000 Titel ohne ISBN: `inventur.SucheTextDNB` nur mit Bestätigung durch einen Menschen
  verdrahten.
- `scripts/backup.sh` exportiert die ganze `.env` — als einziger Punkt hier mit leichtem
  Sicherheitsbezug zuerst.
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
- Tabellen-Bauteil: gebaut (`4a55c52f`); offen allenfalls eine Sichtabnahme.

### 6.3 Parkdeck (bewusste Nicht-Entscheidungen)

Integer-Cent statt float64 · Bundle-Splitting · TypeScript-Migration · `inventur/` ins Haupt-API
verschmelzen · `cmd/migrate` (MySQL) löschen — seine PG-Tests sichern mit `internal/uebernahme`
geteilten Code · API-Versionierung · Mandantenfähigkeit (RLS) · Trennlinien-Durchgang (26 Dateien
mit `divide-y`, nur als eigener Durchgang mit Messung im Browser) · Zugangsbuch-Ausdruck je
Schulhalbjahr und Topf · Bestandskartei-Ausdruck zum 15.3. und 15.9. (beides nennt
[mittel_konzept.md](mittel_konzept.md), Abschnitt 7.1, als Verfahrensvorgabe; nicht gebaut).

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
deletions" anlassen. Am 13.09.2026 trägt das Ruleset noch `pull_request`; Pushes gehen über den
Admin-Bypass.

### 7.7 Abnahmen

Ablauf in [abnahme_checkliste.md](abnahme_checkliste.md), vorher ein Backup.

- Flows 1–3 mit dem Sekretariat: LUSD-Import, Versetzung (vor dem Schuljahreswechsel),
  Klassensatz erledigen. Dabei um Geburtsdatum und Eintrittsdatum im LUSD-Bericht bitten. Vorher 5.6. Ein
  LUSD-Import mit echten Schülern erst nach der Littera-Übernahme (7.2).
- Flow 4 (Altbestand-Etiketten, nicht umkehrbar) erst nach 4.8.
- Flow 5: Selbstanmeldung einer Lehrkraft.
- Danach: Ergebnis hier eintragen; bei Parser-Auffälligkeiten die echte LUSD-Datei anonymisiert als
  Testfixture sichern.

### 7.8 Am Server nachsehen (lesend, Einzeiler)

- Laut Serverlesung vom 13.09.2026 läuft `a841e6cd` (Container-Start 12:59 UTC); `/health` nennt
  keinen Commit.
- „Neuer Text" im Ausweis-Layout: laut Serverlesung vom 13.09.2026 in keinem Wert von
  `system_einstellungen`. Bestätigen, dann erledigt.
- Sind die Admin-Konten deaktiviert? Ist `/app/uploads/fotos` leer? Gibt es Lehrkräfte mit
  Platzhalter-Mail `@lehrer-umzug.invalid`? Braucht `repair_fach_kategorie.sql` einen zweiten
  Lauf?

## 8. Schule, Schulamt, Schulträger

### 8.1 E1: Schulamts- und Schulnummer

Für die Referenznummer der Bescheide; die Felder stehen in den Einstellungen und sind am
13.09.2026 leer. **Erst nach 5.2 eintragen:** Mit den Nummern entstehen echte Bescheide.

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

Zwölf Punkte, jeder am Code geprüft. Sieben sind erledigt und stehen deshalb nicht mehr hier:
Strichcode-Scan mit Handgerät und Kamera, Littera-Barcodes, alte Ausdrucke, Restwert beim
Melden, Listenpreis am Titel, Zugangs- und Abgangsbuch, Sortierung und Filter der Leserdatei.
Die Arbeit steht in den Commits vom 17.09.2026. Am 22.09.2026 hat die Schule die drei
Rückfragen vom 21.09.2026 beantwortet; die Antworten stehen wörtlich bei 9.3 c, 9.4 und 9.6.
Offen ist, was hier folgt.

Zwei Quellen liegen dem zugrunde: `~/Downloads/Arbeitshilfe_Mahnschreiben.pdf` (Erlass vom
17.12.2014, Az. 674.100.002-00178) und `~/Downloads/Ablauf Mahnverfahren.pdf` (die
Anforderungsliste, abgeglichen in [mittel_konzept.md](mittel_konzept.md) Abschnitt 3).

### 9.3 Vorgaben des Landes (Protokoll 1)

**9.3 c) Sperrung bei offener Bearbeitung — entschieden und gebaut am 22.09.2026.**

Die Antwort der Schule, wörtlich: „a) Dies bezieht sich nur auf die Lernmittel wegen der
rechtlichen Grundlage der Lernmittelfreiheit in Hessen. b) Für die Lernmittel darf es keinerlei
automatische ‚Sperrung' geben, auch nicht eine Sperrung, die bestimmte Personen aufheben
können. Für die Schulbibliothek gilt dies nicht."

Gebaut als Weiche in `pruefeSchuelerAusleihbarMit`
(`internal/service/loan_checkout_validation.go`): Bei einem Lernmittel entfallen die zwei
Automatiken (offene Forderung, Überfällig-Automatik); die zwei Schalter am Leser bleiben.
Theke und Nachbuchung laufen über dieselbe Funktion. Gates: Mock-Test ohne erwartete Abfrage
und PG-Test am Live-Pfad der Theke, beide am Rückbau rot gesehen.

**Offen, bei dir:** Ein Kind mit überfälligen Lernmitteln, das ein Buch der Schülerbücherei
will — die Automatik weist es heute ab, weil sie Medien zählt, nicht Töpfe. Die Antwort der
Schule verbietet nur die Abweisung des Lernmittels; ob die Bücherei wegen überfälliger
Schulbücher zumacht, entscheidet die Schule selbst. Vorschlag: so lassen, es ist eine
Einstellung (`MaxOverdueItems`).

**9.3 e) Mahnwesen — drei begründete Abweichungen.** Die Punkte 1 bis 3 der Anforderungsliste
(Abwertung, Beschädigungsgrad, wählbare Preisgrundlage) sind am 17.09.2026 gebaut. Es bleiben
drei Stellen, an denen das Programm bewusst etwas anderes tut:

| Nr. | Verlangt                                  | Stand                                                                                     |
| --- | ----------------------------------------- | ----------------------------------------------------------------------------------------- |
| 5   | Versand per Post, E-Mail oder App         | Nur Post — Schriftform, dazu die Datenschutz-Entscheidung A3 vom 22.08.2026.               |
| 6   | Zahlung ohne Rückgabe → Buch **gelöscht** | Wird ausgesondert statt gelöscht — die Bestandskartei muss den Abgang nachweisen.          |
| 7   | Mahnfrist **sechs** Wochen                | Vier Wochen — Arbeitshilfe und Verfahrensbeschreibung nennen vier, mit Datum.              |

Offen bleibt daneben: Der Knopf „Bezahlt" in der Schülerakte bedeutet heute Barzahlung am
Tresen; für Lernmittel ist das die Ausnahme mit Quittung, nicht der Regelweg (5.1). Und der
Zahlungsweg für ein Buch der Schülerbücherei ist nicht entschieden — die Briefe schreiben
dort „(Bankverbindung des Schulträgers nicht hinterlegt)" (E5, 8.3).

### 9.4 Titel ohne Exemplare — entschieden und gebaut am 22.09.2026

Die Antwort der Schule, wörtlich: „Ein Titel/Werk ohne (verliehene oder verfügbare) Exemplare
im Bestand sollte aus unserer Sicht nicht im Katalog erscheinen, da dies zu Verwirrungen
führen könnte. Vielleicht wäre eine Inventur hilfreich, um den Bestand genau zu erfassen und
den Titeln entweder Exemplare zuzuweisen oder sie zu entfernen?"

Gebaut als EIN Prädikat (`repository.SQLTitelHatExemplar`) an den drei Türen, über die ein
Kollegium Titel sieht: Katalogliste `GET /api/books` (Portal und Titel-Verwaltung),
Theken-Suche, Aktionssuche. Der Zulauf zählt als vorhanden (entschieden am 22.09.2026). Der
Titel bleibt in der Tabelle; die Titel-Verwaltung erreicht ihn über den Umschalter „Mit
Exemplaren | Ohne Exemplare" (`?bestand=ohne`, dasselbe Prädikat mit NOT) — das ist die
Aufräumhilfe, die die Schule mit „Inventur" meint; die eigene Inventur zählt Exemplare und
findet einen Titel ohne Exemplar deshalb nicht. Die Bestellliste führt ihn weiter (Bestand
unter jeder Schwelle). Gates am Rückbau des Prädikats rot gesehen, an allen drei Türen.

**Noch anzusehen, bei dir:** der Umschalter in der Titel-Verwaltung am Bildschirm (M3
Segmented Button, Bauteil `Segmente`). Und die E2E-Suite vor dem nächsten Push — sie lief
am 22.09.2026 nicht, weil eine zweite Sitzung den Stack hielt; eine Spec ist angepasst
(`suchfelder-eigene-tuer.spec.js` bekommt ein Exemplar).

### 9.6 A: Mehrjahresbände — entschieden und gebaut am 22.09.2026

Protokoll 5, zweiter Spiegelstrich: „Bei den Ausleihfristen fehlt das Jahr. Mehrjahresbände
lassen sich nicht abbilden." Die Antwort der Schule, wörtlich: „Ja, Ihre Vermutung trifft zu.
Die Einstellung ‚Mehrjahres-Band' wird am Titel/Werk hinterlegt (s. Screenshot). Littera Lm
funktioniert nicht nach der Logik ‚Vorrücken', es werden jeweils die verschiedenen Schuljahre
aufgerufen und nebeneinander gespeichert. Aus diesem Grund muss hier auch das Schuljahr
definiert werden."

Gebaut in drei Schritten am 22.09.2026: die Fristregel (Rückgabetermin der Klasse geht ein
Mehrjahresband nichts an; nach dem Termin rechnet sie vom folgenden Schuljahr aus), dann eine
Tür am Werk als zweite Jahreszahl — und noch am selben Tag ihr Ersatz, weil die Zahl neben
„Jahrgang von … bis" doppelt war (Migration 134): Am Werk steht ein Schalter „Mehrjahresband",
die Zahl ist `jahrgang_bis`, die Spalte `ziel_jahrgang` ist weg (gemessen am Testserver: ohne
Wert). Der Schalter gilt nur an einem Lernmittel und nur mit einer Spanne über mehr als einen
Jahrgang, an beiden Türen der Titel-Verwaltung und als CHECK in der Datenbank. Das Schuljahr,
das Littera getrennt führt, steckt im Datum der Frist. Gates am Rückbau rot gesehen: Fristregel,
beide Schreibpfade, CHECK.

**Noch anzusehen, bei dir:** der Schalter in der Maske am Bildschirm.

### 9.7 Reihenfolge der drei Bauten

Beide Rückfragen vom 21.09.2026 sind beantwortet (9.3 c, 9.6); die Schule hat dazu 9.4 neu
aufgemacht. Vorschlag für die Reihenfolge, jede Stufe mit eigenem Commit und Gate:

1. **9.3 c** — gebaut am 22.09.2026.
2. **9.4** — gebaut am 22.09.2026 (drei Türen, Aufräumsicht).
3. **9.6** — gebaut am 22.09.2026 (Fristregel und Tür am Werk).

### 9.9 Zwei Bedingungen neben der Mängelliste

Die Einschätzung am Ende des Protokolls nennt zwei Punkte, die in keinem der zwölf Mängel
stehen:

> „Ein Nachweis der DSVGO-Konformität liegt nicht vor.
> Hosting- und Programmpflegekonzepte sind nicht geplant. Dies könnte ein Ausschlusskriterium
> sein."

- **Nachweis der DSGVO-Konformität.** Das Material liegt vor und ist vollständiger, als der Satz
  vermuten lässt: VVT-Entwurf, Datenschutzhinweis-Entwurf, die PII-Matrix über jede Route, die
  Löschfristen samt nächtlichem Job. Was fehlt, ist ein Dokument, das man einer prüfenden Stelle
  GIBT — und die Beschlussfassung der Schule (8.5, B1–B7). Das Zusammenstellen ist Arbeit an der
  Doku, keine Bauarbeit.
- **Hosting- und Programmpflegekonzept.** Hier fehlt wirklich etwas. Betrieb, Sicherung,
  Wiederherstellung und Aktualisierung sind beschrieben — aber als Anleitung für den Betreiber
  (DEPLOYMENT.md, resilience_and_recovery.md, SCRIPTS.md), nicht als Konzept, das jemand prüft:
  Wer betreibt das Programm, wer pflegt es, was gilt, wenn die Pflege endet, wie kommen
  Aktualisierungen zu einer Schule, die es einsetzt. Das ist zugleich die Frage, ob das Programm
  über die eigene Schule hinaus verwendbar ist — sie lässt sich nicht aus dem Code beantworten.

**Beides ist Voraussetzung für ein „nutzbar", nicht Beiwerk** — und beides steht in keinem
der vier Punkte darüber. Wer die Mängelliste abarbeitet und diese zwei Sätze überliest, hat das
Gate nicht bestanden.
