# Offene Arbeit

Stand: 05.10.2026

**Die eine Liste.** Hier steht alles, was noch zu tun, zu prüfen oder zu entscheiden ist — Code,
Betrieb und Schule. Einen zweiten Ort gibt es nicht. Erledigtes wird gelöscht, nicht archiviert:
Die Geschichte steht in den Commit-Nachrichten und in `git log -p docs/OFFEN.md`.

Andere Dokumente erklären (Konzept, Anleitung, der Katalog der Bugklassen in
[sweeps.md](sweeps.md)), führen aber keine eigene Offen-Liste.

---

## Was jetzt dran ist

**Die Sichtung vom 16.09.2026** (Abschnitt 9): Für die zwei Bedingungen aus 9.9 — DSGVO-Nachweis
sowie Hosting- und Pflegekonzept — liegen Entwürfe vor; offen ist, was bei der Schule liegt.

**Entschieden am 28.09.2026: Neuaufbau am Schulserver.** Der Echtbetrieb beginnt mit einer leeren
Datenbank und der Littera-Übernahme (7.2).

**Was bei dir liegt — der Reihe nach:**

1. **Drei Fragen** (28.09.2026). Ist eine davon ein Nein, hilft kein weiterer Code; sind alle
   drei ein Ja, bleibt vor dem Echtstart überschaubare Arbeit im Code.
   - **Schulträger:** Gibt es den Schulserver, ab wann, in welchem Netz (8.5, B5)? Heute ist er
     nur geplante Zielumgebung (Abschnitt 7), und echte Schülerdaten — auch die aus Littera —
     gehören nur dorthin. Dazu: Hat der Schulträger eine Vorlage für das IT-Sicherheitskonzept?
     Dann kommt der Teil des Programms in seiner Form. Außerdem: ob seine IT Betriebssystem und
     Docker pflegt, wie im Pflegekonzept vorgesehen (Abschnitt 9); wie die Pflege den Server
     erreicht, vor Ort oder über einen Fernzugang; und was die Seite für die Lieferanten von ihm
     braucht: einen Namen im Internet, die Freigabe von Port 443 und die Angabe, wie Anfragen aus
     dem Schulnetz am Server ankommen (4.23); und ob er einen Speicher für die Sicherungen außer
     Haus stellt (7.3).
   - **Wer die Sichtung gemacht hat** (Abschnitt 9): Reichen Datenschutz-Nachweis und
     Pflegekonzept als die zwei Bedingungen, werden die drei begründeten Abweichungen im
     Mahnwesen akzeptiert, und wer wird Vertretung?
   - **Littera:** das Kennwort, das bei der Einrichtung von Littera für dessen SQL Server
     vergeben wurde — zu fragen bei dem, der Littera eingerichtet hat, nicht bei der Bücherei.
     Damit öffnen sich womöglich die zwei Sicherungen vom 01. und 02.09.2026, und ein neues
     Backup ist nicht nötig. Kennt es niemand, bleibt die Littera-Hotline (7.2).
2. **Die übrigen Anfragen** (Abschnitt 8), soweit noch nicht gestellt:
   - **Sekretariat:** die Schulnummer für die Bescheide (8.1); wie Ersatz für Bücher der
     Schülerbücherei bisher bezahlt wurde und wer eine Zahlung einbucht (8.3, erster Schritt).
   - **Schulamt:** das Kürzel des Schulamtsbereichs und welches Kassenjahr in die Referenznummer
     gehört (8.1); der Mail-Erlass vom 11.06.2018 und das aktuelle Musterschreiben — aus dem von
     2014 stammen heute Zahlstelle und Bankverbindung im Bescheid (8.2).
   - **Datenschutzbeauftragter der Schule:** beteiligen und schriftlich festhalten, ob eine
     Datenschutz-Folgenabschätzung nötig ist (B4); das Foto auf dem Ausweis (B3; beides 8.5).
   - **Schulträger**, nach der Antwort der Schule: die Zahlungswege der Schülerbücherei (8.3,
     zweiter Schritt).
   - **An einem Buch selbst:** den Eigentumsvermerk auf den alten Littera-Etiketten der
     Schülerbücherei ablesen. Entschieden am 28.09.2026: Neue Etiketten tragen denselben
     Wortlaut; er wird beim Einrichten unter Einstellungen → Schule eingetragen (heute leer, also
     kein Vermerk). Littera führt den Vermerk je Exemplar: In der Medienliste vom 12.06.2026
     tragen über 13.000 Exemplare das Land, 2.942 den Schulträger, einige hundert andere
     Eigentümer (Schule, Förderverein, Bibliothek), rund 50.000 keinen. Zum Ablesen ein Buch
     nehmen, das dort den Schulträger trägt. Der Vermerk je Exemplar kommt mit (4.24).
   - **Bücherei:** wem die Bücher mit den Littera-Vermerken „Philipp-Reis-Schule", „Bibliothek",
     „Förderverein", „Info Schulprojekt" und „Dauerleihgabe" gehören — rund 630 Exemplare, bis
     dahin ohne Zuordnung (4.24).
3. **Der Nachweis von Hand für die Theke ohne Netz** (Abschnitt 2, Stufe 1 und 3 im echten
   Chrome) — zurückgestellt am 24.09.2026. Stufe 2 (die Tür per curl) mache ich am lokalen
   Stack, wenn der Nachweis ansteht.
4. **Am Testserver mit dem Handscanner probieren**, nach `git pull` und `./update.sh`. Die
   Browser-Tests tippen die Zeichen blind wie ein Scanner; ob der Scanner der Schule schnell
   genug tippt (höchstens 50 ms je Zeichen), zeigt nur das Gerät.
   - Sperrbildschirm (Sperrfrist dafür unter Einstellungen kurz stellen): ein Buch scannen.
     Erwartet: „Scan erkannt", kein Fehlversuch; danach schließt das getippte Passwort auf.
   - Theke: Leser scannen, einen Reiter der Akte anklicken, in der Akte nach unten rollen, ein
     Buch scannen. Erwartet: Die Suchleiste steht noch im Fenster, das Buch ist gebucht.
   - Buchmaske: einen Titel halb ausfüllen, sperren lassen, aufschließen. Erwartet: Die
     Eingaben stehen noch da.
   - „Neues Buch" öffnen und ein Buch scannen, ohne ins Feld zu klicken. Erwartet: Die ISBN
     steht im Feld, Titel und Autor sind eingetragen. Ein zweites Buch scannen, wieder ohne
     Klick. Erwartet: ISBN, Titel und Autor sind die des zweiten.
   - Buchmaske, ohne Scanner: einen Titel mit vielen Exemplaren öffnen (der größte am
     Testserver hat 383) und nach unten rollen. Erwartet: „Speichern", das Cover und die Knöpfe
     darunter bleiben im Bild. Dann ins Feld ISBN „12345" tippen und „Speichern" drücken.
     Erwartet: Die Meldung oben rechts („ungültiges ISBN-Format") liegt nicht über dem Knopf.
   - Druck-Center, Buch-Etiketten: einen Titel mit mehr als fünf Exemplaren wählen, den Haken
     „Alle … Exemplare" entfernen, ins Feld „Nummer eingeben oder scannen" klicken und ein Buch
     dieses Titels scannen, einmal mit einem Littera-Etikett und einmal mit einem eigenen.
     Erwartet: Das Feld ist wieder leer, und in der Vorschau steht genau dieses Etikett.
5. **Die Maske „Buch bearbeiten" in der neuen Reihenfolge ansehen** (gebaut am 03.10.2026):
   zuerst die ISBN und die Angaben zum Buch, darunter die Gruppe „An der Schule" mit der Wahl
   Bibliothek oder Lernmittel, „Andere Auflagen" als letzte Angabe dieser Gruppe, Bestand und
   Zähldatum unter „Exemplare". Das Feld „Beschreibung / Klappentext" gibt es seit dem
   03.10.2026 nicht mehr; „Speichern" bleibt am neuen Bibliotheksbuch bedienbar und führt ohne
   Signatur zum Feld. Seit dem 03.10.2026 ist die ISBN freiwillig, Pflicht ist der Titel (Stern
   an der Beschriftung): einen Titel ohne ISBN öffnen, die Signatur ändern und speichern; ein
   Medium ohne ISBN neu anlegen und danach noch einmal mit demselben Titel und Autor — die
   Maske fragt dann „Ist es dasselbe Medium?". Seit dem 03.10.2026 sind die zehn- und die
   dreizehnstellige ISBN dieselbe Nummer: ein Buch mit der zehnstelligen ISBN vom Titelblatt
   anlegen — gespeichert steht die dreizehnstellige im Feld — und danach in „Neues Buch"
   seinen Strichcode scannen; die Maske fragt „Vorhandenen Titel öffnen?".
6. **Ein Termin für die Abnahmen** mit dem Sekretariat, sobald der Schulserver steht (7.7). Ein
   LUSD-Import mit echten Schülern kommt erst nach der Littera-Übernahme (7.2).
7. **Das Mahnwesen ansehen** (gebaut am 04.10.2026): Kinder anhaken, „Mahnbriefe drucken". Es
   kommt je Kind der Brief an die Eltern mit Anschrift; die Liste zeigt danach
   „1× gemahnt, zuletzt …". Zu entscheiden ist, ob die zwei Listen im Druck-Menü bleiben (4.30);
   danach bekommt der Knopf zum Druck-Menü seine Form (5.51).

**Im Code,** in dieser Reihenfolge:

1. **5.21** (Palettenfarben, Bildschirm für Bildschirm).
2. **5.46** (Portal: ein Weg für Wunsch und Meldung) — die Frage ist am 05.10.2026 vorgelegt.
3. Nach der Antwort zu 8.3: **5.4**.
4. **5.10** (Gates und Werkzeuge) und Abschnitt 6 nur mit Anlass.

Einen Termin hat Node 26 ab dem 28. Oktober 2026 nach der Regel „immer die aktive LTS"
([PFLEGEKONZEPT.md](PFLEGEKONZEPT.md), Abschnitt 4). Vor dem Echtstart außerdem: 5.31
(`update.sh` für den Schulserver), der Eingang für die Seite der Lieferanten (4.23) und die
Auflage in der Littera-Übernahme (5.5).

**In der Doku:** Pflegekonzept und Datenschutz-Nachweis stehen als Entwurf. Es folgen die
Arbeitsnotizen ins Repository und die Probe durch die Vertretung (9.9).

Mit der Littera-Übernahme (7.2) kommen das Eigentum je Exemplar (4.24) und alle Schlagworte und
Interessenkreise der Titel; ob Littera Verweise zwischen Schlagworten führt, zeigt erst die
Sicherung von 2026 (4.20).

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
2. **Kategorie A wird belegt, nicht behauptet:** ein Test, der am alten Code rot wird. Bis ein
   Fund nachgestellt ist, heißt er „Verdacht".
3. **Neues kommt nur hierher** — Funde, offene Fragen, Betriebspunkte. Kein Issue, kein anderes
   Dokument. Eine Frage steht hier, bevor die Antwort kommt.
4. **Erledigt heißt:** hier löschen — Datum und Begründung stehen in der Commit-Nachricht, ein
   Archiv gibt es nicht. Eine Antwort bekommt „Entschieden am …" und fällt weg, sobald
   sie umgesetzt ist. Die Nummer eines gelöschten Punkts wird nicht wieder vergeben —
   Kommentare im Code nennen sie als Herkunft.
5. **Die Reihenfolge** wird bei jeder Änderung mitgepflegt.

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
  Offline „zurückgeben" im Profil → Rückgabe. Lehrkraft laden, Buch scannen → Ausleihe auf die
  Lehrkraft. IndexedDB blockiert → „NICHT gespeichert". Zwei Sicherungen einspielen → ein
  Stapel. Zwei parallele Anfragen mit einem Schlüssel gegen `/api/action` → eine Ausleihe.
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

**Heute** setzt die Übergabe nur Status und Zeitpunkt (`Uebergebe`, `repository/bescheid.go`).
Die Forderung bleibt offen, und weil die Unterlagen eine Rückmeldung nur für eingegangenes Geld
vorsehen, meist für immer: Der Abgänger wird nie anonymisiert oder gelöscht
(`PredikatAnonymisierung`), Löschen von Hand lehnt der Server ab, die Selbstprüfung meldet ihn
nach einem Jahr. Die Oberfläche sagt dagegen schon: „Der Fall liegt bei der Aufsicht; die Schule
veranlasst nichts mehr." (`bescheidStatus.js`)

**Die Unterlagen der Schule beantworten die Frage** (nachgelesen am 24.09.2026). Arbeitshilfe,
Abschnitt 2: „Das weitere Verfahren wird im Staatlichen Schulamt geführt … Weitere Schritte sind
durch die Schule nicht zu veranlassen." Anforderungsliste (`Ablauf Mahnverfahren.pdf`), Punkt 7,
zum Ausdruck für das Schulamt: „Der Saldo der betroffenen Leser wird entsprechend bereinigt."
Littera kennt keine Übergabe; dort bucht man einen offenen Betrag von Hand aus, um den Leser
löschen zu können. **Entschieden am 24.09.2026: Mit der Übergabe ist der Fall für die Schule
erledigt**; zu melden bleibt eine spätere Rückgabe, das kann die Theke schon. Umbau: 5.3.

### 4.20 Littera-Schlagworte übernehmen — die Verweise

Die Übernahme aus der Sicherung zählt die Verweise zwischen Schlagworten
(`Verweise_Schlagworte`, `Verweis_Zu_Schlagworte`), übernimmt sie aber nicht
(`internal/littera/schlagworte.go`); in der Sicherung von 2010 sind beide leer. Nennt die
Generalprobe mit der Sicherung von 2026 welche, an diesen Daten die Form ablesen und die
Übernahme bauen.

### 4.23 Erreichbarkeit von außen

**Entschieden am 28.09.2026: Von außen ist nur die Seite für die Lieferanten erreichbar,** alles
andere nur aus dem Schulnetz. Lehrkräfte erreichen das Programm vorerst nicht von zu Hause
(ebenfalls entschieden am 28.09.2026); der Katalog ohne Anmeldung (`/api/public/opac/…`) ist nur
in der Schule durchsuchbar. Caddy holt das Zertifikat selbst bei Let's Encrypt wie am
Testserver (`Caddyfile`: keine `tls`-Zeile). Ohne HTTPS ginge im Betrieb nicht einmal die
Anmeldung: Das Sitzungs-Cookie wird nur über HTTPS gesetzt (`ermittleCookieSecure` in
`main.go`). Mit einem Zertifikat, dem die Browser nicht vertrauen, stünde an jedem Gerät eine
Warnung, und die Theke ließe sich bei einem Netzausfall nicht neu laden (Service Worker,
[arc42/08](arc42/08-querschnittliche-konzepte.md), „Offline"). Das Uptime-Signal von außen (7.5)
ruft `/health` ab.
Littera lief im Schulnetz; Recherche, Reservierung und Verlängerung über das Internet gab es dort
nur mit dem gesondert lizenzierten Zusatzmodul web.OPAC (Littera-Handbuch, „Einstellungen für den
web.OPAC").

**Was die Seite von außen braucht** (am Code nachgesehen am 28.09.2026): den Pfad
`/bestellung/<token>` mit den Dateien der Oberfläche, `/api/public/bestellung/…` (Abruf,
Etiketten, Bestätigung) und für das Uptime-Signal `/health`, das nur `healthy` oder `unhealthy`
meldet. Mehr nicht: Das CSRF-Cookie für die Bestätigung setzt schon der erste Abruf
(`refreshCSRFCookie` in `api/csrf.go`), der Aufruf von `/api/auth/me` beim Laden
(`restoreSession`) darf scheitern, und die Seite zeigt keine Personendaten — Titel, ISBN, Menge,
Schule, Lieferant, Topf, Kundennummer (`api/bestellbestaetigung_public.go`).

**Zu bauen, vor dem Echtstart:** Der Eingang (Caddy) lässt von außen nur diese Pfade durch; was
nicht ausdrücklich freigegeben ist, bleibt zu. In den Einstellungen die öffentliche Adresse
setzen. Mit dem Bau [arc42/07](arc42/07-verteilungssicht.md) (dort „:80/:443 öffentlich") und
[arc42/09](arc42/09-architekturentscheidungen.md) nachziehen.

**Vom Schulträger** (Frage oben): der Name, unter dem der Server im Internet erreichbar ist, und
dass die Geräte der Schule ihn unter demselben Namen erreichen, denn das Zertifikat gilt für den
Namen; die Freigabe von Port 443 von außen auf den Server und der Netzabschnitt, in dem er dann
steht; und mit welcher Absenderadresse Anfragen aus dem Schulnetz am Server ankommen. Daran
erkennt der Eingang das Schulnetz. Kommen Anfragen von außen und aus der Schule mit derselben
Adresse an, kann er sie nicht unterscheiden.

Auch im Schulnetz braucht der Server Verbindungen nach außen: den Mailserver der Schule (die
Anmeldung läuft über das Postfach, `auth/imap.go`), DNB, Google Books und OpenLibrary für
Titeldaten und Cover (`pkg/coverquelle`), für Updates GitHub, Docker Hub und die Paketquellen
(`Dockerfile`, `update.sh`). Eine Aufstellung für den Schulträger entsteht nicht (entschieden am
28.09.2026).

### 4.24 Eigentum je Exemplar

Die Übernahme ordnet den Littera-Eigentumsvermerk je Exemplar nach einer festen Liste zu
(`vermerkeLittera` in `internal/littera/eigentum.go`): „Land Hessen" wird Land, auch an den
11.160 Exemplaren ohne LMF-Signatur (nach dem Leitfaden „Lernmittelfreiheit in Hessen", Ziffern
2.1 und 9.4.2; Begründung in 35ae1cf9), „Hochtaunuskreis" wird Schulträger. Nach welcher Regel
Etikett, Bestandsbücher und Schadensersatz das Eigentum lesen, steht in
[FACHKONZEPT.md](FACHKONZEPT.md).

**Offen bei der Schule:** wem die Exemplare mit den fünf seltenen Vermerken gehören (Schule 355,
Bibliothek 157, Förderverein 86, Info Schulprojekt 31, Dauerleihgabe 4). Bis dahin kommen sie
nur als Wortlaut mit, und es gilt die Faustregel. Die Zuordnung steht an einer Stelle
(`vermerkeLittera`); nach der Übernahme lassen sich einzelne Titel auch in der Buchakte setzen
(_Eigentum ändern_).

**Zur Frage an die Bücherei (30.09.2026):** Eine hessische Schule ist eine nichtrechtsfähige
Anstalt und schließt Rechtsgeschäfte „mit Wirkung für den ermächtigenden Rechtsträger" ab
(Hessisches Schulgesetz). „Philipp-Reis-Schule", „Bibliothek" und „Info Schulprojekt" heißen also
Schulträger oder Land, je nachdem, wessen Geld es war; „Förderverein" und „Dauerleihgabe" können
Dritten gehören. Gefragt wird je Vermerk: aus welchem Geld — Schulträger, Land, Förderverein —
oder geliehen? **Entschieden am 01.10.2026:** kein dritter Eigentümer im Programm. Bücher Dritter
behalten den Littera-Wortlaut, den die Buchakte schon zeigt; Ersatz für ein verlorenes Buch liefe
dann über den Schulträger, bei 90 Büchern ein seltener Fall.

### 4.30 Was das Mahnwesen neben dem Mahnbrief druckt

Seit dem 04.10.2026 kommt aus der Auswahl ein Papier: der Mahnbrief an die Eltern. Daneben
drucken drei Wege weiter je Kind ein Blatt in Du-Form, keiner zählt:

- „Mahnliste einer Klasse" im Druck-Menü (`GET /api/print/mahnung/klasse/{klasse}`, Überschrift
  „Erinnerung: Rückgabe von Bibliotheksbüchern", `pdf.GenerateMahnliste`),
- „Übersichtsliste" im Druck-Menü (`GET /api/mahnwesen/pdf`, Überschrift „Mahnung –
  Schulbibliothek", `generateMahnPDF`),
- der Anhang der Mail an die Klassenleitung („Alle anmahnen"), dasselbe Blatt wie die
  Übersichtsliste.

**Zu entscheiden:** Bleiben die zwei Listen im Druck-Menü? Der Anhang der Mail hängt am Mahnlauf
und ist nicht gemeint.

Nachgesehen und am lokalen Stack ausprobiert am 05.10.2026:

- Beide Einträge drucken je Kind ein Blatt, keine Liste: der eine für eine Klasse, der andere
  für alle Klassen.
- Im Blatt je Klasse steht unter „Fällig seit" das Ausleihdatum (`queryMahnungSchueler` in
  `api/print.go` liest `ausgeliehen_am`). Probe: gedruckt 04.09.2026, die Frist lief am
  24.09.2026 ab.
- „Diese Seite drucken" zeigt Reiter, Suchfeld und Knöpfe mit: `print:hidden` wirkt an einem
  Element mit `flex` nicht, weil `designer/PrintPreview.svelte` jedes `.flex` im Druck auf
  `display: block !important` stellt. Buchtitel stehen nicht in der Liste.
- Littera hat einen Mahnbrief und daneben die „Liste der verliehenen Medien", die mit „nur
  Überfällige drucken" zur Mahnliste wird und sich auf Klassen einschränken lässt (Handbuch,
  „Verliehene Medien").

Vorgelegt am 05.10.2026, Antwort offen: a) eine Liste — der Knopf heißt „Liste drucken" und
druckt, was in der Liste steht, als Tabelle (Klasse, Kind, Buch, fällig seit, wie oft
gemahnt), die zwei Blätter entfallen (empfohlen); b) kein Druck neben dem Mahnbrief; c) es
bleibt bei drei Einträgen, mit berichtigtem Datum. Mit a und b entfällt 5.51.

Am 04.10.2026 gebaut und am selben Tag zurückgenommen, weil nicht bestellt:

- **Brief ab 18 an die Person selbst.** Der Mahnbrief geht immer an „Eltern von …". Die Regel
  hat nur der Bescheid; die Arbeitshilfe des Landes nennt sie für das Schreiben mit der
  Zahlungsaufforderung („bei den volljährigen Schülerinnen und Schülern oder den
  Erziehungsberechtigten der minderjährigen"). Littera kennt keine Altersregel. Gelesen, nicht
  nachgestellt: Auch die Ersatzforderung (`pdf/rechnung.go`) und der Brief zum Schadensfall
  (`pdf/schadensfall.go`) sprechen immer die Erziehungsberechtigten an.
- **Klasse auf dem Brief.** Das frühere Blatt aus der Auswahl nannte die Klasse, der Brief nennt
  sie nicht. Die Briefe einer Klasse liegen im Druck beieinander.

---

## 5. Abarbeitbar (Kategorie B)

### 5.3 Die Übergabe schließt die Forderung ab (nach 4.4)

**Zurückgestellt am 02.10.2026:** Zurzeit wird nichts an die Schulaufsicht übergeben. Der Punkt
wird nicht vorgeschlagen und nicht in Reihenfolgen genannt, bis eine Übergabe ansteht.

E6 ist am 24.09.2026 bejaht (4.4). Fertig sein muss es spätestens mit der Antwort zu E1 (8.1) —
ab dann gibt es echte Bescheide und vier Wochen später die erste Übergabe. Eine Stufe, jeder Punkt mit einem Test, der am Rückbau rot
wird; das Modell der Stufen 1 und 2 beschreibt das [Handbuch](HANDBUCH.md).

- **Abschließen:** Bis dahin zählt eine übergebene Forderung an der Theke weiter als Hinweis:
  Sie hält Bücherei und Gerät an, übergehbar (FACHKONZEPT §2.2). `Uebergebe` setzt in derselben
  Transaktion `ist_bezahlt` an den offenen Forderungen, wie Zahlung und Storno — die 17 Dateien, die nach offenen Forderungen fragen,
  folgen von selbst, die Karenz startet über den Trigger aus Migration 137 (`aktualisiert_am`
  mitsetzen). Dazu ein Kennzeichen je Forderung: Ein Brief kann bezahlte und offene Positionen
  mischen.
- **Rückfrage:** „Übergeben" bucht heute ohne Rückfrage (`BescheideTabelle.svelte`). Übergeben
  wird nur, wenn nicht gezahlt wurde — gezahlt wird aber aufs Konto des Landes, das die Theke
  nicht sieht (wer es einbucht: 8.3). Der Dialog nennt Referenznummer und Betrag und fragt nach
  dem Zahlungseingang.
- **Späte Rückgabe:** `VerbucheRueckkehr` muss übergebene Forderungen ausdrücklich mitnehmen,
  sonst entfällt „Schulaufsicht informieren" (PG-Test). `entferneSchuelerPIIUndLoesche` löscht
  heute jede erledigte Forderung mit dem Leser; eine übergebene bleibt ohne Person stehen wie der
  Bescheid, sonst findet die Theke nach der Löschung nichts.
- **Anzeige:** Akte, Auskunft und Protokoll zeigten sonst „bezahlt" — dritter Zustand „an die
  Schulaufsicht übergeben"; ebenso die Abhilfe in `pruefeEhemaligeOffen` und der Theken-Satz „Die
  Forderung bleibt bis dahin offen".
- **Übergabe-PDF:** Original-Nachdruck und Sammelliste für das Schulamt.
- Vorher am Testserver zählen (lokal 0):
  `SELECT count(*) FROM schadensfaelle f JOIN schadensersatz_bescheide b ON b.id = f.bescheid_id WHERE b.status = 'uebergeben' AND NOT f.ist_bezahlt;`

**Bis dahin (Verdacht, am Code gelesen):** Die Akte bietet „Bezahlt" und „Stornieren" auch auf
einem übergebenen Bescheid, die Selbstprüfung rät nach einem Jahr dazu — und nach einem Storno
löst eine spätere Rückgabe „Schulaufsicht informieren" nicht mehr aus. Wirkt erst mit echten
Bescheiden.

### 5.4 Schadensersatz Teil A, Etappen 3 und 4 (nach 8.3)

Konzept: [mittel_konzept.md](mittel_konzept.md), Abschnitt 4.7.

- **Kreis-Rechnung:** zweite Variante des Renderers (Nummernkreis `SB-Jahr-lfd`, Frist, Tabelle,
  Hinweis auf Ersatzbeschaffung, Zahlungsweg aus den Einstellungen; ohne Referenznummer im
  Landesformat, ohne Rechtsbehelfsbelehrung). Solange die Bankverbindung des Schulträgers fehlt,
  steht sichtbar „(Bankverbindung des Schulträgers nicht hinterlegt)". Warnung in der
  Betriebsbereitschaft. Topf in die Referenznummer, sonst kollidieren Land und Kreis an der
  UNIQUE-Spalte. Heute lehnt der Server jeden Topf außer `land` mit 409 ab.
- **Zahlungen nach 8.3**, wie Littera sie führte (Zahlungsart, Beleg, Auswertung nach Zeitraum
  und Zahlungsart): Zahlungsart beim Knopf „Bezahlt", Quittung als PDF, beide Wege auf der
  Rechnung, Liste der Barzahlungen je Zeitraum und Topf — auch für die Ausnahme beim Land (Bargeld
  binnen 14 Tagen weiterleiten, [mittel_konzept.md](mittel_konzept.md) 1.1). Der Hinweis
  „bereits bezahlt" nennt dann, wann und wie; an ihm zeigt sich eine doppelte Zahlung.
- **Altbriefe entfernen** (es geht darum, ob es sie neben dem Bescheid überhaupt weiter geben
  soll): Elternbrief `pdf/schadensfall.go` ← `api/pdf.go` (`GenerateDamagePDFHandler`) ←
  Route in `api/routes_students.go` — die Oberfläche ruft ihn seit dem 15.09.2026 nicht mehr
  auf (`33e92c44`), erreichbar ist er nur noch über die Adresse; Rechnung `pdf/rechnung.go` ←
  `api/print.go` ← `GET /api/print/rechnung/{schueler_id}` in `api/routes_system.go` ← Knopf
  in `students/SchuelerDokumente.svelte`; dazu `api/print_rechnung_pg_test.go` und
  `elternbrief_generiert*`. Der Eltern-Mahnbrief bleibt.
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
- „Klasse" neben der Spanne (22.09.2026): Zwei Jahrgangsangaben am Titel, „Klasse"
  (`grade_level`) und „von … bis" (`jahrgang_von/bis`). Inventur nach Klasse und die
  Mehrjahresband-Frist lesen nur die Spanne; Titel-Tabelle,
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
  **Entschieden am 24.09.2026, im selben Umbau:** „Jahrgang unbekannt" wird eine eigene Vorgabe
  (NULL) statt 5 bis 10 — heute ist beides nicht zu unterscheiden, und wer das
  Mehrjahresband (Migration 134) an einem Titel mit der Vorgabe anhakt, bekommt die 10.
  Die zwei Leser der Spanne (Inventur, Portal-Filter) lernen „unbekannt" mit; die Ansicht
  „Jahrgang" des Mahnwesens, der dritte, ist seit dem 04.10.2026 ausgebaut. Vorher am
  Testserver messen.
  Ein vierter Leser ist die Suche im Medienkatalog (`trifftJahrgang` in
  `frontend/src/inventur/lib/startseiten_api.js`): Sie liest Klasse oder Spanne. Seit dem
  01.10.2026 zählt die Vorgabe 5 bis 10 dort nicht mehr als Jahrgang, wie schon in der
  Schulbuchliste (`jahrgangText`); vorher traf „Klasse 5" bis „Klasse 10" jeden Titel ohne
  gepflegte Spanne. Gemessen am Testserver am 01.10.2026: 13.057 von 13.062 Titeln tragen die
  Vorgabe, 153 davon eine Klasse. Eine bewusst gepflegte Spanne 5 bis 10 ist bis zum Umbau
  davon nicht zu unterscheiden; mit ihm liest die Suche „unbekannt" statt der Vorgabe.

  ```sql
  SELECT grade_level, jahrgang_von, jahrgang_bis, ist_lernmittel, signatur, titel
  FROM buecher_titel WHERE grade_level BETWEEN 1 AND 13
  ORDER BY ist_lernmittel DESC, signatur NULLS LAST, titel;

  SELECT jahrgang_von, jahrgang_bis, count(*) FROM buecher_titel GROUP BY 1, 2 ORDER BY 3 DESC;
  ```

- **Die kurze Nummer der alten Littera-Etiketten lässt sich an der Theke nicht eintippen**
  (gefunden am 30.09.2026, am Code gelesen, nicht nachgestellt). Nach der Übernahme ist die
  Nummer eines Littera-Exemplars der EAN-13 seines Etiketts (`5896800039556`); lesbar steht auf
  dem Etikett nur „Exemplar-Nr.: 58968". Liest der Scanner das Etikett nicht mehr, findet die
  Theke das Buch über die getippte kurze Nummer nicht: `resolveOhnePraefix`
  (`internal/service/omnibox_service.go`) sucht die Nummer genau und rechnet nur 13-stellige
  Scans zurück; die kurze Nummer steht in `erweiterte_eigenschaften` als `littera_exemplarnr`
  und wird nirgends gelesen. Ist `FremdLeserNummer` im frischen Backup leer (7.2), tragen die
  Ausweise die Littera-Lesernummer, und beide Nummernkreise beginnen bei 1: Eine getippte kurze
  Buchnummer kann dann einen Leser laden. Bis dahin: den Titel suchen (die 13 Ziffern beginnen
  mit der kurzen Nummer) oder im Druck-Center unter „Fehlende Etiketten", Stufe „Alle", nach der
  kurzen Nummer suchen und das Etikett nachdrucken; der Nachdruck trägt die volle Nummer als
  Strichcode und als Text. Entscheiden, sobald feststeht, ob `FremdLeserNummer` gefüllt ist.
- **Speichern zwei Plätze denselben Titel, gilt bei den Feldern des Titels der zweite Stand**
  (gefunden am 01.10.2026, am Stack nachgestellt). `PUT /api/books/{id}` schreibt alle Felder
  zurück, auch die, die niemand angefasst hat: Die Signatur, die Platz 1 gespeichert hat, ist
  nach dem Speichern von Platz 2 wieder die alte, beide Antworten sind 200. Der Bestand ist
  seit dem 01.10.2026 ausgenommen; er geht nur geändert und mit der Zahl vom Öffnen mit.
  Abhilfe: Die Maske nennt die Felder, die sie geändert hat, und der Server schreibt nur
  diese. So halten es Googles Regeln für Schnittstellen (AIP-134: „only fields declared in
  the field mask are updated"). Ein Vergleich des ganzen Stands lehnte dagegen auch ab, wenn
  dazwischen nur ein Cover nachgeladen wurde. Dieselbe Form an Leser, Gerät, Benutzer und
  Einstellungen ist nicht durchgesehen (Raster, Frage 18; [sweeps.md](sweeps.md), „Absoluter
  Wert aus dem Ladezeitpunkt").
- Die Buchakte führt ausgesonderte und bestellte Exemplare als „Gesperrt" (Reiter
  „Exemplare", `BookExemplarCard.svelte`; die Buchmaske listet seit dem 02.10.2026 nur den
  Bestand). „Exemplar löschen" antwortet dort an einem ausgesonderten „exemplar nicht
  gefunden oder bereits ausgebucht". An einem bestellten sondert es aus, was nie eingetroffen
  ist (`DeleteCopy`): Das Exemplar steht danach im Abgangsbuch, ohne je im Zugangsbuch
  gestanden zu haben. Kategorie B.
- Die Übernahme aus Littera überträgt die Auflage nicht (gefunden am 03.10.2026, am Code
  gelesen und an der Sicherung gezählt): `sqlTitelEinfuegen` in `internal/littera` schreibt die
  Spalte `auflage` nicht, das Feld am Titel gibt es seit dem 17.09.2026. In der Sicherung von
  2010 tragen 3.182 von 10.732 Titeln eine Angabe in `Titel.Auflage` („1. Aufl.", „2. Aufl."),
  663 davon stehen nach der Übernahme ohne ISBN da. Ohne die Auflage sind zwei Ausgaben
  desselben Buchs im Katalog nicht zu unterscheiden. Vor dem Echtstart, mit 7.2. Kategorie B.
- Druck-Center, Buch-Etiketten: Scheitert das Laden der Exemplare eines Titels, steht dort „Zu
  diesem Titel gibt es kein Exemplar, das ein Etikett bekommen kann." (`loadExistingCopies` in
  `stores/labels.svelte.js` leert die Liste bei jeder Fehlantwort; am Code gelesen im
  Rasterdurchgang vom 03.10.2026). Der Satz legt nahe, neue Barcodes zu erzeugen. Abhilfe: der
  Ladefehler als eigener Zustand (`ui/LadeFehler`). Kategorie B.
- Druck-Center, Buch-Etiketten: Die Vorschau zeichnet für jedes Etikettenformat dasselbe Blatt,
  drei Spalten mit Etiketten von 42,3 × 25,4 mm (`LabelPreview.svelte`). Das ist der Bogen
  „Zweckform L4760" in zwei Dritteln der Größe. Für „Avery 3475" (3 × 8) und „Kleine Barcodes"
  (4 × 13) zeigt sie damit nicht den gewählten Bogen; gedruckt wird nach
  `api/label_formats.go`, die Überschrift der Vorschau nennt das gewählte Format (gemessen am
  03.10.2026).
- Zwei Schreibweisen stehen neben ihrem Helfer (gefunden am 05.10.2026 beim Umbau zu den
  Meldungen zur Wartbarkeit). Einen Betrag in Euro schreiben sechs Stellen selbst
  (`toLocaleString` und `+ ' €'`: `useFehlbestand.svelte.js`, `StudentBescheideCard`,
  `BestellHistorie`, `BestellDetail`, `BescheidDialog`, `BescheideTabelle`), mit gewöhnlichem
  statt geschütztem Leerzeichen; dafür gibt es `formatEuro` (`utils/format.js`). Den Text eines
  gefangenen Fehlers (`e instanceof Error ? e.message : String(e)`) schreiben 21 Stellen in 13
  Dateien selbst; `fehlertext` (`utils/fehlertext.js`) rufen bisher drei. Kategorie B.

### 5.10 Gates und Werkzeuge

- Die Schema-Gegenrichtung ist blind für UNIQUE, Teilindizes und RESTRICT.
- Kein Gate gegen unbegrenzte Listen-Endpunkte.
- Kein Rückweg für ältere Sicherungen beim Wechsel des `BACKUP_ENCRYPTION_KEY`.
- `TestHandlerFormulierenKeinNeuesSQL` (`api/schichtung_test.go`) sieht ein `UPDATE` mit
  Tabellenkürzel nicht: Das Muster verlangt `UPDATE <Tabelle> SET`, und `UPDATE ausleihen a SET`
  trifft es nicht. Gefunden am 04.10.2026: Die Ratsche hielt `api/mahnwesen_bulk.go` für frei
  von SQL, als dessen einzige Anweisung ein Kürzel bekam. Drei Anweisungen dieser Form stehen
  in `api/ausleihe.go`, `api/etiketten_offen.go` und `api/student_promotion.go`; die Dateien
  stehen wegen anderer Anweisungen in der Liste. Ein neuer Handler, dessen einzige Anweisung
  so aussieht, bliebe unbemerkt. Kategorie B.
- `e2e/kontrast.spec.js` misst den Medienkatalog nicht in jedem Lauf mit seinen Kacheln
  (gefunden am 02.10.2026, lokal mit 8.600 Titeln). `warteAufStabilenBaum` gilt als stabil,
  sobald zwei Zählungen im Abstand von 100 ms gleich sind; kommt die Titelliste später, misst
  der Test die Seite ohne Kacheln und geht weiter. Belegt an einer Kachel, die den Klick auf
  den nächsten Menüpunkt scheitern ließ, solange der Test ihn per Teilwort traf: fünf von
  sechs Läufen rot, einer grün, die Kacheln standen dort also noch nicht. Abhilfe: je Seite
  auf ein Merkmal des Inhalts warten (Kachel, Tabellenzeile). Dieselbe Form des Wartens steht
  in `typo-rollen.spec.js`, `control-hoehen.spec.js` und `icon-trefferflaechen.spec.js`, dort
  nicht nachgemessen. Kategorie B.
- 44 Specs klicken Menüpunkte per `page.getByTitle('<Name>')`, 87 Stellen (gezählt am
  03.10.2026). Das trifft jedes Element, dessen `title` den Namen enthält, auch die Kachel
  eines Buchs. `e2e/abgaenger-management.spec.js` legt Titel „Abgänger Buch …" an und räumt
  sie nicht ab (lokal 133, der älteste an erster Stelle des Katalogs). Die zwei Klicks auf
  „Abgänger" (`schueler-profil-klick.spec.js`) kommen heute von der Theke, wo keine Kachel
  steht; vom Katalog aus brächen sie ab wie der Kontrast-Test bis zum 02.10.2026. Abhilfe:
  `menuepunkt` aus `e2e/helpers.js` an allen Stellen, und die Spec räumt ihren Titel ab.
  Kategorie B.
- Browser-Tests lassen Daten liegen (gezählt am 02.10.2026 in der lokalen Datenbank; in der CI
  ist die Datenbank je Lauf frisch). `e2e/bestellung-detail.spec.js` bestellt je Lauf drei
  Exemplare am ersten Titel des Katalogs und nimmt nur den Lieferanten wieder weg; der
  Teardown löscht die Bestellung, die Exemplare bleiben „im Zulauf" ohne Bestellung.
  `e2e/abgaenger-management.spec.js` räumt nichts ab (133 Titel „Abgänger Buch …"), der
  Wareneingang-Test in `e2e/scrollbereiche.spec.js` lässt je Lauf acht Titel mit je einem
  Exemplar im Zulauf liegen, `e2e/zugangsbuch.spec.js` je Lauf einen Titel mit zwei
  Exemplaren (74 Titel). Von `e2e/leserdatei.spec.js` stehen aus der Zeit vor dem
  02.10.2026 noch 84 Titel, 83 Ausleihen, 86 Leser und 81 Konten. Ein voller Lauf am
  02.10.2026 ließ 94 Titel und 438 Exemplare zurück. Lokale Zahlen tragen diese Reste mit.
  Abhilfe je Spec: eigener Titel, Aufräumen über die Kennung. Kategorie B.
- Code, den kein Go-Test ausführt (gemessen am 03.10.2026 mit der ganzen Suite und `-coverpkg`
  über alle Pakete: 84,2 % der Anweisungen; die Messung je Paket rechnet die Datenbank-Tests
  aus `api/` nicht für `repository/` an und nennt 78,9 %). Unter 50 % liegen, ohne `cmd/`,
  `main.go` und Dateien mit weniger als 20 Anweisungen: `api/littera_import.go` 0,7 % (Littera-
  und Bestandsdatei hochladen; die Regeln in `internal/littera` 82,7 %),
  `internal/service/cover_service.go` 18,2 %, `internal/littera/altbestand.go` 4,4 %,
  `api/klassen_mapping.go` 27,0 %, `api/schueler_etiketten.go` 2,3 %,
  `inventur/endpunkte_cover_retry.go` 26,5 %, `api/geraete.go` 49,0 %, `db/seed.go` 35,5 %,
  `api/ausweis_layout.go` 33,3 %, `api/user_admin_loeschen.go` 48,3 %,
  `repository/mail_settings.go` 38,1 %. Ob Browser-Tests diesen Code erreichen, ist nicht
  gemessen. Anlass: Das Nachziehen der Tests für fünf Routen am 03.10.2026 fand drei Fehler
  (zwei Abweisungen beim Zusammenführen ohne Grund, ein unlesbares Bild als Störung gemeldet,
  eine Antwort des Foto-Uploads, die kein JSON war). Abhilfe je Route: ein Test mit Datenbank
  und eine Gegenprobe je Zusicherung, Muster in `api/inventur_verlust_aktionen_pg_test.go`.
  Kategorie B.
- `beforeEach(() => attrappe.mockReset())` steht in 16 Testdateien der Oberfläche, in einer
  davon mit `mockClear` (gefunden am 05.10.2026). Die Kurzform gibt die Attrappe zurück, und Vitest ruft eine Funktion, die ein
  Hook zurückgibt, nach dem Test als Aufräumer auf: Jeder Test ruft die Attrappe danach noch
  einmal. Wirft oder scheitert sie dann (`mockRejectedValue`), wird der Test rot, obwohl seine
  Erwartungen stimmen; nachgestellt an `klassensatzReservierung.svelte.test.js`. Abhilfe: der
  Rumpf des Hooks in geschweiften Klammern. Kategorie B.
- SonarQube läuft von Hand über `scripts/sonar_scan.sh` gegen das Projekt `Bibliothek5`; der
  Schlüssel steht in `sonar-project.properties`. Letzter Scan am 04.10.2026, der vierte des
  Tages, in der Einteilung der Übersicht (MQR-Modus): Zuverlässigkeit 0 Meldungen (Note A),
  Sicherheit 0 (A), Wartbarkeit 114 (A), Abdeckung 77,4 % von 38.650 Zeilen. Am 03.10.2026
  waren es 209 Meldungen, davon 75 `go:S3776` und zehn mit Auswirkung auf die Zuverlässigkeit
  (Note C); beide Gruppen und zehn Meldungen zur Wartbarkeit sind behoben. Die übrigen 114
  sind am 05.10.2026 einzeln gelesen und bearbeitet: im Code 20 in Go (c8f05392, bbb30013,
  baf75962, 4c126aaa), 80 in JavaScript und die eine in Python (fd29f5a9 bis 096f08a6); 13 in
  JavaScript stehen als begründete Ausnahme in `sonar-project.properties` (e12: `S7776` in
  `.svelte.js`, e13 bis e21: `S2925` in neun Specs). Der Scan nach diesen Commits steht aus;
  bis dahin zeigt der Server weiter 114. Nicht vorab messbar waren die zwei Meldungen zu
  `S6594` in `frontend/scripts/druck-sektionen-gate.mjs` und die Wirkung der Ausnahmen; die
  übrigen Regeln sind mit ESLint nachgestellt und treffen die geänderten Stellen nicht mehr.
  `komplexitaet_ratsche_test.go`
  lässt keine Produktionsfunktion über 15 zu. Das Projekt ist am 04.10.2026 neu angelegt, der Stand vom
  03.10.2026 liegt auf dem Server unter `Bibliothek4a`. Das Quality Gate vergleicht mit dem
  ersten Scan des Projekts und steht auf OK mit drei Bedingungen; vom neuen Code sind 94,1 %
  getestet (3 von 47 Zeilen offen: zwei im Fehlerausgang von `ZaehleMahnungTx`, eine in
  `scripts/generalprobe/probe_host.py`). Der dritte Scan stand an dieser Bedingung auf Fehler
  (57,7 %, Gesamtabdeckung 72,0 %): Er rechnete einem Go-Paket nur die Tests an, die in ihm
  selbst liegen. Seit dem 04.10.2026 misst das Skript mit `-coverpkg` über alle Pakete
  ([SCRIPTS.md](SCRIPTS.md), „Warum die Coverage niedriger aussieht, als sie ist").
- **Der Wechsel auf Ubuntu 26 als Runner.** Seit dem 28.09.2026 laufen alle zehn Jobs fest auf
  `ubuntu-24.04` statt auf `ubuntu-latest`, das ab dem 19. Oktober 2026 auf Ubuntu 26 zeigt
  (actions/runner-images#14748). Den Wechsel selbst legen, mit einem eigenen Lauf gegen das
  neue Abbild — brechen kann etwa der Postgres-Client oder die Chromium-Abhängigkeiten von
  Playwright; spätestens, wenn GitHub `ubuntu-24.04` abkündigt. Nicht darunter: CodeQL läuft
  in der Standard-Einrichtung von GitHub (Repository-Einstellung, keine Workflow-Datei) auf
  `ubuntu-latest` und wechselt am 19. Oktober 2026 mit; der Hinweis darauf steht an jedem
  CodeQL-Lauf (gesehen am 28.09.2026). Bricht die Analyse dort, wird der CodeQL-Lauf rot.
- **gosec: acht Regeln global ausgenommen** (gemessen mit v2.29.0 am 28.09.2026, ohne
  `-exclude`): G706 (36 Stellen in 18 Dateien), G704 (6), G703 (5), G120 (5), G124 (4), G404
  (4), G115 (3), G101 (1); der Grund je Regel steht in `.github/workflows/security-scan.yml`.
  Eine neue Stelle dieser Regeln meldet gosec nicht. Abhilfe: je Stelle ein `#nosec` mit Grund,
  dann die Regel aus `-exclude` nehmen — außerhalb von G706 sind es 28 Stellen in 14 Dateien.
  Nur mit Anlass.
- **Ein Browser-Test war im vollen Lauf einmal rot.** `e2e/feld-roundtrip.spec.js` („Buch
  anlegen: Bestand, Zähldatum, Standort kommen in der DB an") fand am 05.10.2026 im vollen
  Lauf am lokalen Stack den neuen Titel nicht binnen 10 s; einzeln lief die Datei danach
  zweimal grün. Die lokale Datenbank trägt Hunderte Test-Titel aus früheren Läufen.

### 5.19 Lesepfade gegen die Sicht `schueler` — was offen bleibt

**Offen aus dem Umbau der Auskunft (gebaut am 24.09.2026):** Die Rohdaten der Protokolleinträge
(`details`) stehen nur in der abgerufenen Auskunft, nicht auf dem Blatt; das Gate
`TestDsgvoPDF_DrucktJedeAngabeDerAuskunft` führt sie als begründete Ausnahme, seit dem
24.09.2026 auch die Details der Kontoereignisse. Offen ist, was davon aufs Blatt gehört.
Nachgesehen am 24.09.2026: Die bearbeitende Person steht in eigenen Spalten (`bearbeiter_id`,
`admin_id`), die die Auskunft nicht ausgibt; Freitexte in den Details — etwa der Grund einer
Sperre (`LESER_GESPERRT`, `LESER_ENTSPERRT`; bis zum 24.09.2026 auch `OVERRIDE_BLOCK`) —
können aber andere Personen nennen.

**Beim Bau der Auskunft über gelöschte Konten gefunden (29.09.2026), nicht gebaut:** Zwei
Einträge über die Anlage eines Kontos tragen seine Kennung nicht als `ziel_id` und fehlen
deshalb unter den früheren Zugangskonten der Auskunft, sobald das Konto gelöscht ist.

- `SELBSTANMELDUNG` trägt das Konto nur als `admin_id`; die Spalte geht beim Löschen des
  Kontos auf NULL (`auth/selbstanmeldung.go`).
- `KOLLEGIUMSKONTO_NACHGETRAGEN` (Schul-E-Mail in der Akte nachgetragen) trägt die Leserkennung,
  nicht die des Kontos; die Auskunft zeigt ihn unter den Verwaltungseingriffen, nicht beim Konto
  (`api/student_update.go`).

Abhilfe in beiden Fällen: `ziel_id` in die Details, für den Nachtrag dazu eine Beschriftung im
PDF (`dsgvoKontoAktion`).

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

Stand 05.10.2026: 651 Fundstellen mit Tailwind-Palettenfarben (`slate`, `blue`, `emerald` …),
gehalten von der Ratsche `frontend/src/lib/frontend-hygiene-farben.test.js`; Neues entsteht nur
noch auf Rollen. Umstellen ist eine Umgestaltung, keine Umbenennung: Die Palette führt sechs
Textgraustufen, M3 zwei Rollen. Für „in Ordnung" und „Achtung" gibt es die eigenen Rollen
`success` und `warning` in `styles/rollen.css` (M3, „Define custom color roles"). Vorgehen:
Bildschirm für Bildschirm, die größten zuerst, je Portion ein Commit, am gerenderten Bildschirm
geprüft. Das Muster steht in Buchformular und Bestellfenster: Zustände über ui/StatusChip, Cover
über ui/BuchCover, Rückmeldung beim Zeigen über den State-Layer statt `hover:bg-*`, ein Fehler
über den Fehlerzustand des Feldes statt eines farbigen Kastens. Für die übrige Anwendung
freigegeben am 23.09.2026.

`inventur/lib/bookHelpers.js` (48) sind Farbverläufe je Fach für selbstgebaute
Cover-Platzhalter; das gehört zu 6.2 (Cover über `ui/BuchCover`).

Beim Umstellen aufgefallen, jeweils am Code nachgesehen:

- Titel-Verwaltung: Ein Titel lässt sich in der Liste nur mit der Maus öffnen. Der Klick hängt
  an der Zeile (`BookTableZeile.svelte`, `onclick` am `<tr>`), die Zeile nimmt keinen Fokus. Die
  Leserdatei öffnet die Akte über den Namen als Knopf.
- Titel-Verwaltung: Der Knopf „Retry Cover" trägt eine englische Beschriftung.
- „LUSD & Versetzung": Die Flächen für Fehler, Erfolg, Hinweis und Warnung stehen in
  `LusdImportView` und `PromoteStudentsView` von Hand, wie an rund 40 weiteren Stellen der
  Anwendung (gezählt am 02.10.2026: getönte Fläche und Rundung in einer Klassenliste); ein
  gemeinsames Bauteil dafür gibt es nicht.
- Statistik: Die Sprechblase des Diagramms (`StatsTrendChart`) trägt wie die Sprechblasen der
  ganzen Anwendung (`actions/tooltip.js`) `bg-slate-900`, und `rollen.css` kennt die Rolle dafür
  nicht (M3, Color roles: „Inverse surface: Background fills for elements which contrast against
  surface"). Die Balkenfarben des Diagramms sind feste Werte, keine Rollen.
- Benutzerliste: Der Zustand eines Kontos steht in zwei Formen, „Aktiv" und „Inaktiv" als Punkt
  mit Wort, „Zugang beantragt" als Pille (`UserManagementTable`).
- Inventur: Im Fehlbestandsbericht ist der Titel bei 1280 px nach rund 95 px abgeschnitten,
  während die Spalte „Signatur" etwa dreimal so breit ist (gesehen am 02.10.2026; die Titelzelle
  trägt `max-w-0` ohne volle Breite an der Spalte). Der Bericht wird zum Absuchen des Regals
  ausgedruckt; der Ausdruck kürzt ebenso, nach rund 24 Zeichen, und rechts bleibt ein Viertel
  der Seite leer (gemessen am 05.10.2026).
- Inventur: Die Wörter „Inventur-Scope" (Überschrift des Start-Dialogs) und „aus dem aktuellen
  Scope" (Rückfrage vor dem Abschluss) stehen so in der Oberfläche; ein deutsches Wort wäre
  „Umfang" oder „Bereich".
- Inventur: Ein unbekannter Barcode zeigt am Scanner den rohen Fehlertext „exemplar für
  inventur-scan nicht ladbar: no rows in result set" (`GetExemplarForInventoryScan` hüllt
  `pgx.ErrNoRows` ein, `ladeExemplarFuerScan` gibt ihn mit 404 unverändert weiter). Der Status
  stimmt, der Satz nicht.
- Leserdatei: Die Leiste des Ausweisdrucks (`students/AuswahlAktionsleiste`) ist dunkel in
  Palettenfarben; für markierte Zeilen gibt es `ui/AuswahlLeiste` (Schlagwort-Pflege). Beim
  Umstellen zu klären: wohin der Hinweis „ohne Ablaufjahr" und das Feld „Ab Feld" kommen —
  beides passt nicht in die 64 px hohe Leiste.
- Die Ratsche zählt eine Palettenfarbe an einer einzelnen Rahmenseite nicht
  (`border-l-amber-500`): drei Stellen, `system/BackupAlert` (2) und
  `students/DeletedStudentList` (1), gezählt am 05.10.2026.
- Mahnwesen: Scheitert das Blatt je Klasse, steht die Meldung in einem eigenen Kasten oben
  rechts (`globalErrorToast` in `stores/mahnwesenPdf.svelte.js`); die übrigen Meldungen
  derselben Datei gehen über `toastStore`. Entfällt mit dem Blatt (4.30).

### 5.22 Fremdrückgabe über Kreuz verklemmt sich — seit Migration 137

Der Rückgabe-Trigger `trg_leser_stempel_rueckgabe` (Karenz-Uhr) sperrt bei einer Rückgabe die
Leserzeile des Ausleihers — im Normalfall immer, denn die Rückgabe liegt nach dem letzten
Stempel —, und zwar NACH der Ausleihe. Die Fremdrückgabe an der Theke sperrt vorher das Kind der
offenen Sitzung. Geben zwei Kinder an zwei Theken zugleich je das Buch des anderen ab, wartet jede
Transaktion auf die andere, und Postgres bricht nach einer Sekunde eine ab (40P01). Nachgestellt
am 23.09.2026 mit den Schritten des Codes; ohne den Trigger läuft derselbe Ablauf durch
(`TEST_DATABASE_URL=… go test -tags raster -run TestRaster_Fremdrueckgabe ./repository/`). Das
Nachbuchen zweier Theken, die über Kreuz umbuchen, hat dieselbe Folge von Sperren (nicht eigens
nachgestellt).

Wirkung: An der Theke erscheint eine Fehlermeldung, erneutes Scannen bucht. Beim Nachbuchen
wird der Eintrag „wiederholen" und läuft in der nächsten Runde durch. Keine Daten gehen verloren.
Nebenfolge derselben Sperre: Eine Rückgabe wartet, solange ein anderer Vorgang die Leserzeile
hält (etwa ein LUSD-Lauf, der diesen Schüler ändert).

**Entschieden am 24.09.2026: so lassen.** Die Abhilfe ohne Verklemmung wäre der Stempel in einer
eigenen Tabelle statt an der Leserzeile — eine Migration an der Karenz-Uhr (Löschuhr, Wächter,
DSGVO-Auskunft lesen ihn). Sie kommt beim nächsten Umbau der Karenz-Uhr mit; bis dahin steht der
Punkt hier, damit dieser Umbau ihn findet.

### 5.25 Eine Forderung für ein Gerät lässt sich nicht anlegen

Die Datenbank sieht sie vor (`check_damage_item`: genau eines von `exemplar_id` und
`geraet_id`), die Rechnung an die Eltern kann sie drucken (`queryRechnungItems`), aber der
einzige Schreiber `meldeSchaden` (`repository/schaden_melden.go`) nimmt nur ein Buch-Exemplar:
Er sondert das Exemplar aus und legt die Forderung mit `exemplar_id` an. Fehlt bei der Rückgabe
Zubehör oder ist ein Gerät kaputt, gibt es keinen Weg zur Forderung; das FACHKONZEPT (Abschnitt
5) behauptete bis zum 24.09.2026 einen. Gesperrt würde nach 4.4 wie heute (Schülerbücherei und
Geräte).

Beim Bau mitnehmen (bis zum 01.10.2026 als 5.37 geführt, am Code gelesen): Der Bescheid-Dialog
listet auch eine Forderung ohne Exemplar (`topf` leer). Sie steht richtig gesperrt da, darunter
aber der Satz „Buch der Schülerbücherei — gehört nicht auf den Bescheid des Landes."
(`frontend/src/lib/components/mahnwesen/BescheidPositionen.svelte`). Die Zeile darüber zeigt
„ohne ISBN" und, weil ein Gerät keinen Buchpreis hat, „kein Preis hinterlegt — Betrag bitte
eintragen" neben dem gesperrten Feld. Heute nicht zu sehen: Ohne Schreiber gibt es keine
Forderung für ein Gerät.

### 5.31 `update.sh` für den Schulserver: nur Releases, Images frisch

**Entschieden am 28.09.2026, nicht gebaut.** Vor dem Echtstart; gebaut wird, wenn die drei
Fragen oben beantwortet sind.

- **Nur Releases:** Der Schulserver bekommt nur Releases, der Testserver folgt `main` als
  Vorstufe. Heute holt `update.sh` mit `git pull` den neuesten Stand des Zweigs und fragt nicht
  ab, ob dessen Prüfläufe grün sind. Ein Release entsteht nur, wenn alle Pflicht-Prüfungen des
  Commits grün sind (`scripts/tag-gate.sh`).
- **Die Postgres-Nebenversion am Server** (gefunden beim Pflegekonzept, 24.09.2026).
  `update.sh` ruft `docker compose up -d --build` auf und holt das Image `postgres:18-alpine`
  nie neu; der Datenbank-Container bleibt auf der Nebenversion des Images, das beim Anlegen
  vorlag (Major-Wechsel 31.08.2026). Nebenversionen mit Sicherheitskorrekturen erscheinen
  vierteljährlich; aktuell ist 18.6 (postgresql.org, abgerufen am 24.09.2026). Der Port ist nur
  an `127.0.0.1` gebunden, das begrenzt das Risiko. Die Lücke besteht unabhängig vom Messwert —
  auch bei 18.6 käme die nächste Nebenversion nicht an; die Zahl zeigt nur, wie weit der Server
  zurückliegt. Gemessen am Testserver am 25.09.2026: 18.6, also aktuell. **Entschieden am
  28.09.2026 nach dem Vorschlag vom 25.09.2026: bei jedem Update holen;** die Datenbank startet
  dann bei einer neuen Nebenversion während des Updates neu. Postgres rät zu solchen Updates
  („The community considers performing minor upgrades to be less risky than continuing to run
  an old minor version", postgresql.org/support/versioning); eine Nebenversion braucht weder
  Sicherung noch Neuaufbau. Die Datenbank sortiert unter musl ohne Sprachregeln (lokal
  gemessen: `datlocprovider` = c), ein neues Alpine im Image ändert also die Reihenfolge der
  Indizes nicht. Die CI testet bei jedem Lauf gegen denselben Tag `postgres:18-alpine`. Die
  Hauptversion bleibt im Repo festgeschrieben.
- **Die Alpine-Pakete im Backend-Image** (gefunden am 25.09.2026). Das `Dockerfile` holt
  Sicherheitskorrekturen nur über `apk --no-cache upgrade`. Der Build-Cache hält diese Schicht
  fest, solange die Zeilen davor gleich bleiben, und `update.sh` baut ohne `--pull` und ohne
  `--no-cache`; das Aufräumen in Schritt 7 entfernt nur Schichten, die eine Woche lang niemand
  benutzt hat. Lokal gemessen: Image vom 24.09.2026, die `apk upgrade`-Schicht darin drei Wochen
  alt. Der Trivy-Scan der CI prüft ein frisch gebautes Image, nicht das am Server. Gemessen am
  Testserver am 25.09.2026: 3 Tage — der Server liegt kaum zurück, die Lücke bleibt. **Entschieden
  am 28.09.2026 nach dem Vorschlag vom 25.09.2026: `--pull --no-cache`;** jedes Update baut dann
  alles neu. Der Sicherheitsscan der CI baut ohnehin ohne Zwischenspeicher (`docker build` auf
  einem frischen Runner) und braucht dafür 93 Sekunden (Lauf vom 25.09.2026, Schritt „Build
  Docker image for scanning"); ein Takt-Stempel im `Dockerfile` spart ein, zwei Minuten und
  braucht eigene Mechanik. Die Regel „mindestens einmal im Monat ein Update, auch ohne neue
  Funktionen" steht seit dem 28.09.2026 im Pflegekonzept (Abschnitt 4).
- **Das Löschen hängt am Lauf** (gefunden am 28.09.2026). `update.sh` und `scripts/backup.sh`
  löschen alte Sicherungen nur, wenn sie laufen. Kommt kein Update mehr, bleibt die letzte
  Vorab-Sicherung für immer; ein Klartext-Rest nach einem misslungenen Update bleibt bis zum
  ersten Lauf eines der beiden Skripte nach zwei Tagen. Die Meldungen sagen das seit d0391f6d.
  Vorschlag, nicht nachgestellt: Mit fertigen Images bleibt das alte Image am Server. Läuft
  der alte Container, verschlüsselt Schritt 1 die Sicherung sofort, und der Rückweg
  entschlüsselt sie mit dem alten Image (`docker run --rm`); die Vorab-Sicherung läge dann nie
  unverschlüsselt in `backups/`. Läuft er nicht, bleibt der Klartext-Weg. Für das Löschen nach
  der Uhr bräuchte es einen Lauf, der nicht am Update hängt.

### 5.35 Protokolleinträge zu Lesern, die die Tilgung noch nicht erreicht

Gefunden beim Bau des Gates aus 5.10 am 29.09.2026. Behoben ist am selben Tag die Löschspur
eines Titels: Name und Freitext neben der Kennung des Lesers (`schuldner`, `beschreibung`,
`betrifft`) fallen jetzt mit der Anonymisierung und dem endgültigen Löschen
(`api/titel_loeschspur_tilgung_pg_test.go`). Offen:

- **Einträge zu Lesern, die schon endgültig gelöscht sind.** Der Nachtlauf räumt nur Einträge
  zu anonymisierten Lesern, die noch in der Tabelle stehen. Wer vor dem Einspielen endgültig
  gelöscht wurde, behält in der Löschspur Name oder Freitext bis zur Audit-Aufbewahrung. Der
  Schulserver beginnt leer (Neuaufbau), betroffen ist nur der Testserver. Erst messen, dann wie
  Migration 147 bereinigen:
  `docker exec bibliothek-db psql -U postgres -d bibliothek -c "SELECT count(*) FROM audit_log a WHERE a.details ?| ARRAY['schuldner','beschreibung','betrifft'] AND a.details ? 'schueler_id' AND NOT EXISTS (SELECT 1 FROM leser l WHERE l.id::text = lower(a.details->>'schueler_id'));"`
- **Einträge, die einen Leser nur über seine Forderung meinen.** Das Stornieren einer Forderung
  schreibt `grund` in die Datensatz-Historie (`tabelle = 'schadensfaelle'`, Kennung der
  Forderung, ohne `schueler_id`), ebenso das Stornieren bei der Rückgabe
  (`repository/bescheid_rueckkehr.go`). Die Tilgung findet solche Einträge nicht. Ob in diesem
  Grund Personenbezug steht, ist nicht nachgestellt.
- **Frage: Gilt für die Löschspur von Forderung und Vormerkung die Lesehistorie-Frist?** Die
  Ausleihspur eines gelöschten Titels verliert Kennung und Namen nach der Lesehistorie-Frist
  der Schülerbücherei (`tabelle = 'ausleihen'`; das Exemplar gibt es nicht mehr, es zählt als
  Nicht-Lernmittel). Die Spuren von Forderung und Vormerkung behalten die Kennung bis zur
  Anonymisierung, sonst bis zur Audit-Aufbewahrung (24 Monate). Die Kommentare in beiden
  Löschwegen hatten 90 Tage angenommen; zu den Nachbuch-Meldungen steht in
  `repository/loeschfristen.go`: „länger als die Lesehistorie darf nichts den Schüler an ein Buch
  binden".

### 5.45 Listen in einem Kasten mit eigenem Scrollen

Eine Liste zeigt alle Zeilen, gescrollt wird der Bereich der Seite
(`e2e/scrollbereiche.spec.js`); so stehen die Ausleihliste der Leserakte, die Positionen im
Wareneingang und die Exemplare in der Maske „Buch bearbeiten". Kategorie B. Offen:

- **Weitere Kästen derselben Form, durchgesehen am 02.10.2026** (1280 × 900, lokale Mengen;
  Suchmuster: `max-h-` und `overflow-y-auto` in einer Zeile, 13 Treffer, die übrigen sind
  Auswahllisten, Vorschläge, Dialoge und das Eigenschaften-Feld des Designers). Zu
  entscheiden ist je Kasten, ob er bleibt:
  - Druck-Center, Schritt 2 (`LabelBarcodeSchritt.svelte`, `max-h-40`): Der Kasten bleibt
    als Auswahlliste; seit dem 02.10.2026 stehen darüber ein Kästchen für alle und ein Feld
    für die Nummer. Offen: Bei einem Titel mit 409 Exemplaren ist die Seite durch die
    Vorschau 13.462 px hoch (gemessen am 02.10.2026), und „A4-Bogen drucken" steht unter
    beiden Spalten (`LabelPrinter.svelte`). Jede Zeile nennt „(Neuwertig)", wenn das Exemplar
    keine Zustandsnotiz trägt, auch ein bestelltes.
  - Signaturen (`SignaturenView.svelte`, `max-h-112`): Die Liste links zeigt zwölf von 762
    Signaturen (448 px von 27.432 px) und endet in einer halben Zeile; die Seite scrollt
    daneben selbst (852 von 1.876 px), weil darunter die Sachgruppen folgen. Liste und Regal
    sind Auswahl und Detail (M3, List-detail: „Use the list-detail layout for quickly
    accessing details of an item from a long list of content").
  - Bestellwesen (`OrderRecommendations.svelte`, `BestellWorkspace.svelte`): Bedarf (596 von
    3.900 px bei 60 geladenen von 560 Titeln) und Bestellspalte (811 von 1.488 px bei sechs
    Positionen) scrollen je für sich. Die Bestellspalte ist so entschieden und gesichert
    (`e2e/bestellung-erreichbar.spec.js`). Die Seite läuft dabei 16 px über (852 von 868 px).
  - `LabelSettings.svelte` (`max-h-48`) ist die Vorschlagsliste unter dem Suchfeld, kein
    Kasten im Seitenfluss.
- **Signaturen bei 1280 px:** Liste und Regal sind zusammen 1.000 px breit, Platz sind 960 px
  (gemessen am 02.10.2026). Die Spalte „verliehen" endet 8 px hinter dem Fensterrand, die
  Seite bekommt eine waagerechte Scrollleiste. Die rechte Spalte des Rasters ist `1fr` ohne
  untere Grenze 0 (`lg:grid-cols-[20rem_1fr]`).
- **Die Exemplare in „Buch bearbeiten" bei Mengen wie an der Schule** (am Testserver lesend
  gezählt am 02.10.2026): 2.253 Titel haben Exemplare, mindestens die Hälfte davon eines, 90 %
  höchstens 58; über 100 Exemplare haben 68 Titel, der größte 383. Lokal mit 403 Exemplaren
  öffnet die Maske in 0,3 s, „Speichern", Cover und Knöpfe bleiben im Bild, die Seite ist
  28.377 px hoch. In der Liste lässt sich nicht suchen (`BuchExemplareListe.svelte`); ein
  einzelnes Exemplar findet dort nur die Suche des Browsers. Anlass zum Bauen: Jemand sucht in
  der Maske ein bestimmtes Exemplar.
- **Der Abstand des Rahmens liegt um den Scrollbereich, nicht in ihm** (alle Seiten, gemessen
  am 02.10.2026 bei 1710 × 952 px): `Anwendungsrahmen.svelte` gibt der Arbeitsfläche 24 px
  oben und unten und 32 px an den Seiten, gescrollt wird erst das Element darin
  (`Router.svelte`, `overflow-y-auto`). Die Scrollleiste sitzt deshalb 32 px vom Fensterrand,
  und über und unter dem Inhalt bleiben beim Scrollen je 24 px stehen. Betrifft jedes
  Bauteil, das beim Scrollen stehen bleibt (`sticky top-0`).
- **Leserakte bei 1024 bis 1065 px Fensterbreite:** Seit dem 02.10.2026 beginnt die
  Navigation unter 1280 px eingeklappt (`Sidebar.svelte`); der Titel in der Ausleihliste hat
  damit ab 1066 px mindestens 140 px (`e2e/ausleihliste-zeilen.spec.js`). Darunter, bis zur
  Grenze von 1024 px, an der die Akte Leserkarte (320 px) und Inhalt nebeneinanderstellt,
  bleiben ihm 98 bis 139 px (gemessen am 02.10.2026 mit zwölf Ausleihen). Wer die
  Navigation unter 1258 px von Hand ausklappt, lässt dem Titel weniger als 140 px, unter
  1118 px nichts. Nicht gebaut: die Zeile zweizeilig.
- **Eingeklappte Navigation:** Sie zeigt nur Symbole, bis zu 18; der Name steht im `title`
  des Knopfs und erscheint beim Zeigen mit der Maus. M3, Navigation rail, Guidelines: „All
  navigation items require a one word label text" und „The collapsed nav rail … should
  contain 3–7 navigation items". Ausgeklappt schiebt sie in schmalen Fenstern den Inhalt
  zusammen; M3: „A navigation rail can be expanded by default on larger screen sizes, or can
  be expanded over content on smaller screen sizes". Anlass zum Bauen: Die Anwendung wird
  an einem Tablet oder in Fenstern unter 1280 px bedient.
- **Leserakte, langer Name:** Ein Name aus einem Wort von 22 Zeichen ragt 73 px aus der
  Leserkarte in die rechte Spalte. Er liegt dort unter den Reitern auf der ersten Zeile des
  Inhalts: In „Gebühren & Schäden" und „Stammdaten & Adresse" verdeckt er deren Anfang
  (49 px), in „Ausleihen & Historie" endet er an der Oberkante der Überschrift (gemessen am
  03.10.2026 bei 1280 × 900 px an einem Testleser). Namen mit Leerzeichen oder Bindestrich
  brechen um.
- **Leserakte, Autor und Nummer des Exemplars:** Der Autor steht nur in der Sprechblase am
  Titel, die Nummer in Fenstern bis rund 1580 px ebenfalls (darüber hat sie ihre Spalte;
  gemessen bei ausgeklappter Seitenleiste). Die Sprechblase erscheint beim Zeigen mit der
  Maus; an einem Tablet ohne Maus sind beide Angaben in der Akte nicht zu sehen.
  Vorleseprogramme bekommen sie als unsichtbaren Text.
- **Leserakte, zwei Beschriftungen:** Der Reiter heißt „Ausleihen & Historie", zeigt aber nur
  die laufenden Ausleihen und die Vormerkungen. Unter dem Reiter „Stammdaten & Adresse" steht
  dieselbe Überschrift noch einmal; im Reiter „Gebühren & Schäden" heißt die Liste seit dem
  01.10.2026 „Forderungen".

### 5.46 Portal: ein Weg für Wunsch und Meldung

Wunsch vom 01.10.2026: Im Portal, Reiter „Meine Anliegen", stehen über dem Formular zwei Knöpfe,
„Buchwunsch" und „Etwas stimmt nicht". Beide führen zum selben Formular und zum selben Absenden;
es soll einer sein. Am Code nachgesehen am 03.10.2026 (`portal/AnliegenWidget.svelte`): Die Wahl
ändert die Beschriftung des ersten Feldes („Welches Buch?" oder „Worum geht es?"), dessen
Beispieltext und die Meldung nach dem Absenden. Am Server bestimmt sie den Betreff der Mail beim
Erledigen (`api/anliegen.go`), in der Liste der Bibliothek das Abzeichen „Wunsch" oder „Meldung"
(`bestellungen/AnliegenListe.svelte`). Die zwei Knöpfe sind von Hand gebaut, nicht aus
`ui/Segmente`.

Vor dem Bauen zu klären: Fällt die Unterscheidung ganz weg — ein Formular, ein Abzeichen, ein
Betreff —, oder bleibt sie für die Bibliothek und wird nur anders gewählt? Kategorie B.

Vorgelegt am 05.10.2026, Antwort offen: a) sie fällt ganz weg — ein Formular ohne Vorauswahl,
kein Abzeichen, der Betreff „Ihr Anliegen ist erledigt" (empfohlen); b) sie bleibt, gewählt
wird mit `ui/Segmente`. Sortiert oder gefiltert wird nach der Art nirgends; außer den drei
genannten Stellen liest sie die Auskunft über ein Konto (`api/dsgvo_pdf_konto.go`).

### 5.49 Versetzung und eine Klasse, deren Zahl kein Jahrgang ist

Die Versetzung liest die Zahl am Anfang der Klasse (`promoteStudentsQuery` und
`leseKlassenlehrerVersetzung` in `api/student_promotion.go`). Nachgestellt am 04.10.2026 am
Router, in der Vorschau (`dry_run`):

- Trägt ein aktiver Schüler eine Klasse, deren Zahl am Anfang größer ist als 2.147.483.647 (eine
  Buchnummer im Feld Klasse, „9783123456789"), antwortet die Versetzung mit 500 und versetzt
  niemanden. Die Meldung nennt weder die Klasse noch den Schüler.
- Jede Zahl ab 13 gilt als Abschlussklasse (`repository.AbschlussklasseSQL`): Der Schüler der
  Klasse „2147483647A" wird in der Vorschau als Abgänger gezählt (`archived_count` 1).

Die lesenden Türen (Jahrgangs-Auswahl und Jahrgangsfilter der Leserdatei, LMF-Planer,
Abgängerliste) antworten auch mit einer solchen Klasse (`klassenZahlSQL` in
`repository/abschlussklasse.go`, `api/klasse_lange_ziffernfolge_pg_test.go`); offen ist nur die
Versetzung. Die Klassen der Schule lösen keinen der beiden Fälle aus. Nächster Schritt mit
Anlass: Die Versetzung nimmt Klassen aus, deren Zahl nicht zwischen 1 und 13 liegt, und nennt
sie in der Vorschau. Kategorie B.

### 5.51 Der Knopf zum Druck-Menü im Mahnwesen trägt kein Wort

Seit dem 04.10.2026 öffnet im Mahnwesen ein umrandeter Knopf mit Drucker-Symbol und ohne
Beschriftung das Druck-Menü (`MahnwesenDruckMenue.svelte`); vorher war es der Split-Button
„Mahnbriefe ▾". Verglichen am selben Tag mit den anderen Seiten:

- Druck-Knöpfe tragen sonst Symbol und Wort, in 15 Dateien („Liste drucken", „Drucken",
  „Als PDF", „Nachdruck", „Ausdrucken", „A4-Bogen drucken" …). Ein Drucker ohne Wort steht sonst
  nur als Zeilenaktion in einer Liste (`BookExemplarCard.svelte`,
  `BestellDetailPositionen.svelte`).
- Menüs öffnen sonst über einen Split-Button („Ausweis drucken ▾" in der Schülerakte) oder über
  ⋮ (Bedarf, Zeile im LMF-Plan, Schlagworte).
- „Neu laden" daneben (`MahnwesenAktionen.svelte`) trägt ebenfalls kein Wort; auf den anderen
  Seiten heißt der Knopf „Aktualisieren" oder „Neu prüfen".

M3, Icon buttons (Guidelines): „Default icon buttons can open other elements, such as a menu" —
die Bauform ist erlaubt. Dieselbe Seite: „Icons visually communicate the button's action. Their
meaning should be clear and unambiguous." Der Drucker öffnet hier ein Menü, und zwei der drei
Wege darin laden ein PDF. M3, Menus (Guidelines): Ein Menü öffnet aus „an icon, button, or text
field".

Nächster Schritt, nach 4.30 (die Antwort bestimmt, was hinter dem Knopf steht): ein umrandeter
Knopf mit Drucker, Wort und Pfeil in der Form von „Liste drucken" in der Buchakte; „Neu laden"
bekommt im selben Zug sein Wort. Die Form ist ein Vorschlag, entschieden ist sie nicht.
Kategorie B.

---

## 6. Beobachten und Kategorie C (nur mit Anlass)

### 6.1 Beobachtungen

- Das Feld ISBN der Buchmaske nimmt jede Nummer mit 10 bis 13 Zeichen an (`validiereISBN`),
  also auch elf- und zwölfstellige. **Entschieden am 03.10.2026:** Das bleibt so; das Feld
  trägt auch die Nummer einer DVD oder CD. Die Datenbank bringt nur eine ISBN in ihre
  Normalform (10 oder 13 Stellen, `isbn_normalform`); jede andere Nummer bleibt, wie sie
  geschrieben wurde, mit Bindestrichen und Leerzeichen. Die Dublettenkontrolle der Maske
  vergleicht solche Nummern deshalb zusätzlich ohne Trennzeichen (`titelMitISBN`), der
  UNIQUE-Index tut es nicht. Gemessen am Testserver am 03.10.2026: 65 Titel mit einem Wert
  ohne ISBN-Form.
- Der Medienkatalog lädt in beiden Reitern die ganze Titelliste (`GET /api/books`, ohne
  Grenze): „Suche & Filter" bei jedem Öffnen, weil dort im Browser gesucht wird, die
  Titel-Verwaltung beim Öffnen und bei leerem Suchfeld, also auch bei jedem Wechsel zwischen den
  Reitern. Gezeigt werden je 50 Titel. Die Liste geht mit gzip gepackt hinaus
  (`api/middleware_kompression.go`). Gemessen am 04.10.2026 am lokalen Stack: 9.738 Titel,
  4,87 MB, über die Leitung 0,43 MB. Über den Proxy des Testservers kommen die Antworten gepackt
  an (gemessen am 04.10.2026 vom Server aus: Skript der Oberfläche 347.994 statt 1.091.670 Byte,
  öffentliche Katalogsuche 3.017 statt 8.389 Byte, je mit `Content-Encoding: gzip`). An einem
  Arbeitsplatz, dessen Virenscanner HTTPS filtert, ist das nicht abzulesen: Er entpackt die
  Antwort, und es bleibt der Kopf `x-content-encoding-over-network`. Die lokale Zahl trägt
  Test-Titel mit; Größe und Dauer am Server über das Schulnetz sind nicht gemessen (im Browser:
  F12, Netzwerk, Zeile `books`).
  Anlass zum Bauen: Der Katalog öffnet am Server trotz Packen spürbar verzögert. Dann beantwortet
  der Server einen unveränderten Bestand mit 304 statt mit der Liste, wie bei den Buchnummern
  der Theke (`api/buchbarcodes_handler.go`).
- Breite der Textfelder. **Entschieden am 03.10.2026:** Textfelder folgen Material 3 (Text
  fields, Guidelines: „Text fields shouldn’t span the full width of a large screen"); die
  Hausregel, die Fläche zu nutzen, gilt den Flächen der Seite, nicht den Feldern. In der Maske
  „Buch bearbeiten" enden die Felder seitdem bei 704 px; die Exemplare darunter behalten die
  Breite. Am Bild abgenommen am 03.10.2026; der freie Streifen zwischen Feldern und Cover im
  breiten Fenster bleibt (gemessen bei 1710 px: Felder bis 1016 px). Offen: die übrigen
  Masken — sie sind nicht durchgesehen. Der Wert steht bisher nur in `BuchFormular.svelte`;
  mit der zweiten Maske gehört er an eine Stelle.
- Die Meldungen der Anwendung (`ToastContainer.svelte`) erscheinen oben rechts, bis 384 px
  breit, 5 s lang, und halten ihre Standzeit an, solange der Mauszeiger auf ihnen ruht. Was
  dort steht, ist in dieser Zeit verdeckt (gemessen am 02.10.2026 bei 1280 × 720: Meldung bei
  x 975 bis 1256, y 24 bis 68). In der Maske „Buch bearbeiten" steht „Speichern" deshalb
  hinter der Überschrift statt am rechten Rand (`e2e/scrollbereiche.spec.js`); unter rund
  1070 px Fensterbreite reicht eine Meldung in voller Breite trotzdem bis an den Knopf. M3,
  Snackbar, Guidelines: „Snackbars should be placed at the bottom of a UI, in front of the
  main content" und „avoid positioning the snackbar in a way that completely obscures
  actionable elements". Am unteren Rand stehen die Auswahlleisten der Listen
  (`ui/AuswahlLeiste`, `AuswahlAktionsleiste`); dort müssten die Meldungen ausweichen.
- Das Zugangsbuch baut alle Zugänge des Zeitraums auf einmal auf
  (`components/bestand/Bestandsbuch.svelte`; die Abfrage in `repository/zugangsbuch.go` hat
  keine Grenze). Gemessen am 02.10.2026 an der lokalen Datenbank: 34.621 Zeilen im laufenden
  Halbjahr, 5,4 MB, Abruf 0,1 s, Aufbau im Browser 4,2 s. Die lokalen Zugangsdaten stammen aus
  Importen und Testläufen und sagen nichts über die Schule; ein neues Exemplar übernimmt sein
  Erwerbsdatum als Zugangsdatum (`stempel_zugang_am`). Anlass zum Bauen: ein Halbjahr mit
  mehreren tausend Zugängen am Server. Die Felder je Topf über der Liste bauen dabei nichts
  neu auf: Eine Liste mit Zeilen bleibt im Dokument und wird aus- und eingeblendet (gemessen am
  02.10.2026 mit 38.810 Zeilen: ausblenden 0,1 s, einblenden 0,8 bis 0,9 s).
- Bei 390 px Breite ist die Bestellspalte (Bestellwesen, das Fenster vor dem Warenkorb) 68 px
  breit, auch das Eingabefeld der Schlagworte; die Chips ragen darüber hinaus (gemessen am
  30.09.2026 im echten Chrome, schon vor der zweiten Vorschlagszeile so). Unterhalb von `lg` legt
  `BestellWorkspace.svelte` die Spalten untereinander (`grid-cols-1`); woher die Breite kommt, ist
  nicht nachgesehen. Anlass zum Bauen: Bestellen soll am Telefon gehen.
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
- Von Hand lässt sich eine ausgeschiedene `A-`-Nummer wieder eintragen, in der Akte wie in
  „Benutzer & Rechte" — gewollt für die alte Karte eines Schülers, der zurückkommt (Migration
  146). Die Maske sagt dabei nicht, dass die Nummer schon einmal vergeben war; nur der Generator
  und die Littera-Übernahme lesen `ausweisnummern_ausgeschieden`. Anlass zum Bauen: eine alte
  Karte, die auf diesem Weg an eine andere Person gerät.
- Cover-Dateien ohne Titel (`uploads/cover_auto_…`): Eine ISBN-Abfrage, nach der nicht
  gespeichert wird, lässt eine Datei je ISBN liegen (der Name kommt seit dem 02.10.2026 aus
  ISBN und Inhalt, eine Wiederholung legt keine weitere ab), und ein ersetztes Cover die alte
  Datei. Was vorher entstand, liegt weiter dort; gemessen ist die Menge an keinem Server.
  Im Zwischenspeicher des Cover-Abrufs (`uploads/covers`) liegen dazu die Ersatzbilder von
  Google Books, die bis zum 02.10.2026 als Cover abgelegt wurden (je 1.118 Byte); abgerufen
  werden sie nicht mehr. Anlass zum Bauen: Der Ordner wird merklich groß.
- Ein Titel ohne gespeichertes Cover zeigt, was Google Books oder OpenLibrary liefern. Für
  Schulbücher ist das wenig (gemessen am 02.10.2026 an sieben ISBN der Reihen Deutschbuch,
  Lambacher Schweizer, Green Line und Mensch und Politik): Google Books hat eines, OpenLibrary
  keines, der Cover-Dienst der DNB fünf. Die DNB fragt nur der Cover-Abgleich des Servers
  (beim Start und alle sechs Stunden); bis er einen Titel erreicht hat, steht dort die
  Initiale. Anlass zum Bauen: Nach der Littera-Übernahme fehlen in den Katalogen merklich
  Cover, die es bei der DNB gibt.
- Der Cover-Abgleich fragt eine ISBN, die kein Katalogdienst kennt, alle sechs Stunden und bei
  jedem Start neu (am Code gelesen am 02.10.2026): `processCover`
  (`internal/service/cover_service.go`) setzt `FAILED`, sobald die Abfrage einen Fehler
  liefert, und „nicht gefunden" ist dort ein Fehler; `NOT_FOUND` gibt es nur für einen Treffer
  ohne Cover. Je Titel sind das bis zu fünf Anfragen an DNB, Google Books und OpenLibrary,
  gedrosselt auf zwei Titel je Sekunde. Wie viele Titel es trifft, zeigt
  `SELECT cover_status, count(*) FROM buecher_titel GROUP BY 1` am Server. Anlass zum Bauen:
  Ein Dienst sperrt die Adresse der Schule, oder der Lauf dauert merklich.
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
- Das Druck-Center hängt im Menü an `view_students`, seine Buch-Etiketten brauchen nur
  `view_books` und `edit_books`. Ab Werk hat jede Rolle mit `edit_books` auch `view_students`;
  wer die Rechte anders verteilt, erreicht die Buch-Etiketten nicht (Sammelpunkt wie
  „Einstellungen"). Der Etikett-Knopf der Buchakte fragt deshalb beides ab.
- Dieselbe Form bei der Klassenauswahl (Rasterdurchgang 30.09.2026): `GET /api/klassen` verlangt
  `view_students`, gewählt wird die Klasse aber auch in den Klassensätzen und der
  LMF-Verlängerung (`edit_books`) und im Mahnwesen-Routing (`manage_settings`). Ab Werk hat jede
  Rolle mit einem dieser Rechte auch `view_students`. Wer die Rechte anders verteilt, sieht dort
  „Klassen nicht geladen", und der Rat „Bitte neu öffnen" hilft ihm nicht.
- Ausfallmatrix A3 und B4; A3 erst nach S3 (7.3).
- Anmeldungen stehen nicht im Protokoll (am Code nachgesehen am 28.09.2026): `LoginHandler` in
  `auth/handlers.go` schreibt keinen Eintrag, nur die Selbstanmeldung
  (`auth/selbstanmeldung.go`). Nach einem Missbrauch lässt sich nicht nachsehen, wann und von wo
  ein Konto angemeldet war; der Datenschutz-Nachweis (Abschnitte 8 und 9) sagt das so. Ein
  Protokoll der Anmeldungen wären neue Personendaten des Personals (Zeitpunkt, Netzadresse) mit
  eigener Frist. Anlass zum Bauen: ein Vorfall oder eine Frage des Datenschutzbeauftragten.
- Skripte, die Titel löschen, lassen Werke zurück (Rasterdurchgang 25.09.2026):
  `e2e_altlasten.sql`, `entferne_demo_daten.sql` und `seed_demo.sql` löschen Titel per DELETE;
  das erreicht `werke` nicht, der Verweis zeigt vom Titel zum Werk. Zurück bleiben Werke ohne
  Titel, die keine Ansicht zeigt, oder mit einem einzigen Titel, der überall wie ein Titel ohne
  weitere Auflage erscheint (Titelmaske: unter „Andere Auflagen" steht keine Liste). Kein Schaden; die
  Ratsche `auflagen_schreibpfad_ratsche_test.go` liest keine Skripte.
- Der Katalogisat-Import legt Einträge über den Titeltext zusammen, und ein zweiter Lauf schreibt
  die Angaben des zuletzt passenden Eintrags darüber (gemessen am 30.09.2026 am Export vom Juni
  2026, zweimal in eine leere Datenbank). Der Upsert (`queueTitelUpsert` in
  `BulkUpsertBookTitles`) sucht einen Titel über die ISBN, sonst über den Titeltext. Der erste
  Lauf macht aus 13.708 Einträgen 11.302 Titel: 226 sind echte Dubletten (gleiche ISBN, gleicher
  Text), 432 teilen die ISBN mit einem Eintrag anderen Texts (4190700703809 steht an neun
  verschiedenen DVDs), 1.748 den Titeltext bei anderer oder fehlender ISBN — „Harry Potter und der
  Feuerkelch" steht als Buch, Taschenbuch und DVD in der Datei und wird ein Titel. Beim zweiten
  Lauf gilt ein solcher Eintrag als Aktualisierung des vorhandenen Titels: Titeltext, Autor,
  Verlag, Jahr, Signatur und Jahrgangsspanne kommen aus dem Eintrag, sobald er einen Wert hat
  (`qUpdate`), der letzte gewinnt. Geändert werden so 789 Titel — Jahr 664, Verlag 359, Signatur
  347, Autor 220, Jahrgang von 65 und bis 62 (mit den Interessenkreisen, die die Spanne aus allen
  Werten bilden), Titeltext 38, dazu Leerstellen: ISBN 33, Fach 61, Schlagworte an einem. Der
  Buchtitel „Harry Potter und der Feuerkelch" trägt danach Signatur und Jahr des DVD-Eintrags
  („DvD/D", 2005). Die Übernahme aus der Sicherung (`internal/littera`), mit der der Echtbetrieb
  beginnt (entschieden am 28.09.2026), gleicht nicht über den Titeltext ab; der Import aus CSV und
  Excel (`internal/service/import_dynamic.go`) tut es ebenfalls, dort nicht gemessen. Der Katalog
  am Testserver stammt aus diesem Import (13.705 der 13.708 Einträge finden dort ihren Titel,
  gezählt am 30.09.2026). Anlass zum Bauen: Das Katalogisat wird wieder ein Weg in den
  Echtbetrieb, oder ein gepflegter Katalog soll es erneut einlesen.
- Nach einem Rückbau auf einen älteren Stand behält ein Browser die neuere Startseite: Sie geht
  mit `Last-Modified` und ohne `Cache-Control` hinaus (`http.ServeFileFS` in `api/router.go`),
  und auf die Rückfrage mit dem jüngeren Datum antwortet der ältere Stand mit 304 (gemessen am
  01.10.2026: 304 mit späterem, 200 mit früherem Datum). Die Seite verlangt dann ein Bundle,
  das der Server nicht hat, bis jemand ohne Zwischenspeicher neu lädt. Bei einem Update nach
  vorn tritt es nicht auf. Abhilfe mit Anlass: `Cache-Control: no-cache` und ein ETag aus dem
  Namen des Bundles für die Startseite. Ein Fenster, das über den Rückbau hinweg offen bleibt,
  arbeitet mit dem neueren Programm weiter: Sperrt es nach Inaktivität, lässt es sich nicht
  aufschließen, weil dem älteren Stand die Wege zum Sperren und Aufschließen fehlen (am
  01.10.2026 am Code gelesen, nicht nachgestellt); es hilft nur das Neuladen ohne
  Zwischenspeicher.
- Der LMF-Planer erfährt nur über die Live-Leitung, dass ein anderer Platz den Plan geändert
  hat (`fremdesSignal`). War die Leitung unterbrochen — kein Netz, oder die Sperre nach
  Inaktivität, hinter der sie ruht —, fehlt der Hinweis, und wer danach speichert,
  überschreibt die fremde Änderung. Am Code gelesen am 01.10.2026, nicht nachgestellt. Seit
  dem 01.10.2026 bleibt der Planer hinter der Sperre mit seinen ungespeicherten Änderungen
  stehen; vorher gingen sie mit der Sperre verloren. Abhilfe mit Anlass: nach dem
  Wiederaufbau der Leitung und nach dem Aufschließen den Stand am Server vergleichen.
- Ungespeichertes übersteht seit dem 01.10.2026 die Sperre nach Inaktivität, aber nicht, was
  ihr vorausgeht oder folgen kann (Rasterdurchgang vom 02.10.2026, Frage 15). Nach fünf Minuten
  ohne Bedienung leert sich die Theke und baut die Akte des Lesers ab (am Stack nachgestellt),
  samt einem offenen Dialog — Schaden melden, Stammdaten bearbeiten, Rückgabedatum ändern — und
  dem, was darin getippt war. „Abmelden und als andere Person anmelden" am Sperrbildschirm
  verwirft, was dahinter ungespeichert steht, ohne es zu sagen (am Code gelesen). Ob etwas
  ungespeichert ist, weiß die Anwendung nur beim LMF-Planer (`uiStore.verlassenSperre`), und
  gefragt wird nur beim Wechsel des Menüpunkts und beim Schließen des Fensters. Das Leeren der
  Theke ist gewollt (A4 im Datenschutz-Nachweis). Anlass zum Bauen: eine verlorene Eingabe an
  der Theke.
- Das Schließen-Symbol der Maske „Buch bearbeiten" und „Neues Buch" verwirft, was getippt und
  nicht gespeichert ist, ohne Rückfrage: `onClose` schaltet nur zurück zur Titelliste
  (`inventur/routes/admin/+page.svelte`). Seit dem 02.10.2026 steht „Speichern" im Kopf neben
  dem Symbol. M3, Dialogs, Guidelines: „When someone dismisses a full-screen dialog, a basic
  dialog should appear to confirm that they want to discard the unsaved changes." Dafür müsste
  die Maske wissen, ob etwas geändert ist (wie `uiStore.verlassenSperre` beim LMF-Planer).
- Die Live-Leitung (`GET /events`) verlangt nur eine Anmeldung, und ihre Meldung „action"
  trägt zu jeder Buchung die Kennung des Lesers, die Buchnummer und den Titel
  (`broadcastActionEvent`). Jedes angemeldete Konto bekommt sie, auch das Kollegium ohne
  Leserecht; einen Namen trägt sie nicht (PII-Matrix, Stufe 0). Gelesen am 01.10.2026, nicht
  nachgestellt. Zuhörer sind nur die Theke und die Abgänger-Seite. Frage mit Anlass: Braucht
  die Meldung die Kennung des Lesers für jeden Empfänger?

### 6.2 Kategorie C

- Reste des Worts „Schülerdatei" nach der Umbenennung in „Leserdatei" (16.09.2026), gefunden am
  01.10.2026: das Recht „Schülerdatei anzeigen" samt Beschreibung und der Hinweis darauf in der
  Vormerk-Liste (`permissionMetadata.js`, `BookVormerkungenTab.svelte`), das Etikett der
  Reiterleiste (`StudentDirectory.svelte`), „Öffnet die Schülerdatei …" im Druck-Einstieg
  (`KlassenDruckEinstieg.svelte`), „Schülerakte" in zwei Erklärtexten (`permissionMetadata.js`,
  `DatenschutzKategorie.svelte`) und die Überschrift „Gelöschte Schüler (Papierkorb)", unter der
  auch das Kollegium steht (`DeletedStudentList.svelte`). Beim Umbenennen die E2E-Specs
  mitziehen.
- Drei Knöpfe tun bei leerem Feld nichts und sagen es nicht (gefunden am 01.10.2026 beim
  Durchgang über die Speichern-Wege der Oberfläche, am Code gelesen): die Nummer eines
  Exemplars speichern (`saveBarcode` in `BookExemplarCard.svelte`), eine Sachgruppe „Sichern"
  (`speichereBearbeitung` in `SystematikVerwaltung.svelte`) und „Anmelden" mit leerer Adresse
  oder leerem Passwort (`authStore.handleLogin`). Verloren geht nichts. Den Knopf in dem
  Zustand sperren oder das Feld nennen.
- Die Maske „Buch bearbeiten" speichert nur per Klick auf „Speichern": Sie ist kein Formular,
  die Eingabetaste und ein Tastenkürzel lösen nichts aus (`BuchFormular.svelte`, am Code
  gelesen am 02.10.2026). In der Reihenfolge der Tabulatortaste steht „Speichern" seit dem
  02.10.2026 vor den Feldern; wer mit der Tastatur ausfüllt, erreicht den Knopf nach dem
  letzten Feld nur rückwärts.
- Elf Dialoge sperren „Abbrechen", solange ihre Anfrage läuft (etwa
  `StudentProfileDeleteModal.svelte`, `BescheidDialog.svelte`, `PapierkorbLoeschenDialog.svelte`),
  dazu sechs Stellen in Formularen und Listen (`PromoteStudentsView.svelte` zweimal,
  `GeraeteVerwaltung.svelte`, `AnliegenListe.svelte`, `KlassensatzReservierungen.svelte`,
  `AusleiheRueckgabe.svelte`; gezählt am 03.10.2026). M3 Dialogs, Guidelines: „Disable confirming
  actions until a choice is made. Dismissive actions are never disabled." Abbrechen bricht die
  laufende Anfrage am Server nicht ab. In der Lösch-Rückfrage der Benutzerliste schließt der
  Knopf den Dialog wie Escape, und das Ergebnis der Anfrage steht auf der Seite; zu entscheiden
  wäre, ob das für die übrigen ebenso gilt. Voraussetzung je Stelle: Erfolg und Ablehnung werden
  außerhalb des Dialogs gemeldet.
- Die Akte eines Kollegen ohne Ausweisnummer sagt am gesperrten Ausweisdruck „die Nummer steht
  in „Benutzer & Rechte""; ohne Konto hat er dort keinen Eintrag. Die Nummer kommt mit dem
  freigeschalteten Zugang (`StudentProfileActions.svelte`, `data-tip`).
- Browser-Gates: Die M3- und axe-Gates öffnen die Planer-Dialoge nicht, axe misst nur den
  Anfangszustand; kein Screenreader-Durchgang; der Ausweis-Designer geht nur per Maus.
- Symbol-Knöpfe haben 32 × 32 px ohne größere Trefferfläche (`.icon-btn` in
  `styles/komponenten.css`, Gate `e2e/icon-trefferflaechen.spec.js` mit 32 px als Untergrenze).
  M3, Icon buttons: „Extra small and small icon buttons must have a target size of 48x48dp or
  larger to be accessible." `CLAUDE.md` nennt 48 px Trefferfläche als Hausmaß; gemessen wird
  sie nirgends. In Tabellenzeilen stehen bis zu drei Symbole ohne Abstand nebeneinander
  (Ausleihliste der Leserakte: verlängern, Schaden melden, zurückgeben). Anlass zum Bauen:
  Fehlklicks an der Theke oder Bedienung am Tablet.
- 11 Bestandsstellen bauen ihr Cover selbst (Liste in `frontend-hygiene-cover.test.js`, darunter
  `KlassenBuchKachel` im Portal). Umstellen beim fachlichen Anfassen, nicht in einem Rutsch.
- 3.000 Titel ohne ISBN: `inventur.SucheTextDNB` nur mit Bestätigung durch einen Menschen
  verdrahten.
- Titel aus der DNB tragen deren Platzhalter für eine fehlende Zählung („Deutschbuch [...]
  Gymnasium 5."), und der Zusatz zum Sachtitel (MARC 245 $b) steht im Titel statt im Feld
  Untertitel („Lambacher Schweizer Mathematik 6. Ausgabe Hessen Schulbuch mit Medien Klasse
  6"; an Sätzen der DNB gelesen am 02.10.2026). Die Maske zeigt den Titel vor dem Speichern.
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
- `github.com/jung-kurt/gofpdf` ist seit 2021 archiviert und steckt in 17 Dateien (ohne Tests,
  gezählt am 30.09.2026); gepflegt wird der Ableger `github.com/phpdave11/gofpdf`, den maroto
  mitbringt. Neue PDFs (5.3, 5.4) nicht mehr auf dem archivierten; die 17 beim fachlichen Anfassen
  umstellen, mit den PDF-Gates.
- Etikettenraster doppelt (`api/label_formats.go` und `etikettformate.js`), gehalten von
  `etikettformate-konsistenz.test.js`; am 31.08.2026 entschieden geparkt.
- Reste des Nie-verdrahtet-Sweeps: `abgaenger_jahr` in der Aktivlisten-Antwort ohne
  Konsument; bei den Geräten `ActionEvent.GeraetID` ohne Broadcast und mit Null-Zeitstempel.
- Die Prüfung der UUID-Pfadparameter (`ValidateUUIDParamsMiddleware`) sitzt in
  `RequirePermission`. Eine Route mit `{id}`, `{schueler_id}` oder `{ausleihe_id}` unter
  `RequireAuthenticated` liefe an ihr vorbei, und die Kennung ginge ungeprüft an die Datenbank.
  Heute gibt es keine: Am 04.10.2026 antworteten alle 65 Routen mit UUID-Platzhalter am echten
  Router auf eine Kennung, die keine UUID ist, mit 400. Kein Gate hält das;
  `api/uuid_pfadparameter_test.go` prüft die Namen der Platzhalter, nicht die Hülle der Route.
  Anlass zum Bauen: die erste Route mit UUID-Platzhalter ohne `RequirePermission`.
- `javascript:S6551` und `javascript:S8783`: begründete Dauer-Ausnahmen. Die Begründung zu
  S6551 steht als Kommentar in `settingsWerte.js`; S8783 nennt im Repository keine Stelle
  (nachgesehen am 02.10.2026).
- `auth.Claims.BarcodeID` liest niemand mehr; die Ausweisnummer kommt seit Migration 125
  als LEFT JOIN aus der Leserzeile in die Sitzung, nur damit das Feld gefüllt bleibt.
- Tabellen-Inline-Felder mit 36 px: eine `size="sm"`-Variante von `Feld` erst bei Bedienbefund.
- `LabelHeight >= 30` steht zweimal (`api/label_pdf.go`, `api/schueler_etikett_pdf.go`).
- Zwei Normalformen für Namen (`repository.Suchnorm`, `normName` in `api/lusd_paarung.go`); beim
  Anfassen der Paarung zusammenführen.
- Der Paritätstest vergleicht keine COMMENTs und Seeds.
- Erbe der PR-Zulieferungen: Go-Testdateien über 200 Zeilen, ein schwacher Export-CSV-Test.
- Klone: Go 9 Gruppen (05.09.2026), Frontend 0,41 %.
- 50 Handler-Dateien in `api/` formulieren rohes SQL neben `repository/` (gezählt am 24.09.2026);
  der Bestand ist seit dem 07.08.2026 eingefroren (`handlerMitSQL` in `api/schichtung_test.go`).
  Umstellen beim fachlichen Anfassen einer Datei, nicht in einem Rutsch.
- Exemplarkarte der Buchakte (`BookExemplarCard.svelte`): Die vier Symbolknöpfe sind 14 px groß
  statt 32 px (`.icon-btn`), drei erklären sich per `title` statt `data-tip`; die Buchakte fehlt
  in `icon-trefferflaechen.spec.js` und `icon-tooltips.spec.js`. Ein 32-px-Knopf bricht die
  Kopfzeile bei 1280 px um (gemessen am 24.09.2026: Karte 70 → 100 px) — die Knöpfe brauchen
  eine eigene Zeile; eine Layoutfrage, nicht einzeln.

### 6.3 Parkdeck (bewusste Nicht-Entscheidungen)

Integer-Cent statt float64 · Bundle-Splitting · TypeScript-Migration · `inventur/` ins Haupt-API
verschmelzen · `cmd/migrate` (MySQL) löschen — seine PG-Tests sichern mit `internal/uebernahme`
geteilten Code · API-Versionierung · Mandantenfähigkeit (RLS) · Trennlinien-Durchgang (25 Dateien
mit `divide-y`, nur als eigener Durchgang mit Messung im Browser) · Zugangsbuch-Ausdruck je
Schulhalbjahr und Topf · Bestandskartei-Ausdruck zum 15.3. und 15.9. (beides nennt
[mittel_konzept.md](mittel_konzept.md), Abschnitt 7.1, als Verfahrensvorgabe; nicht gebaut) ·
ein Schüler wird Lehrkraft (die Datenbank verbietet es, `chk_leser_nur_schueler_werden_abgaenger`;
heute ein zweiter Leser, bei Häufung ein Umzugspfad wie Migration 072) ·
Schlagwortliste drucken, Schlagwortkatalog als Datei aus- und einlesen (wie Littera, nur wenn die
Bücherei es braucht; 23.09.2026) · Verweise für Autoren (der Autor ist ein Textfeld, kein
Personensatz) · Statistik nach Zweigen (braucht den Zweig an der Ausleihe ohne Namen, weil eine
Ausleihe der Bücherei seit dem 29.09.2026 den Namen nach einem Tag verliert; nur mit eigener
Frage).

---

## 7. Betrieb (liegt bei der Schulseite)

Geplante Zielumgebung ist der Schulserver; heute ist der Hetzner-Server die einzige Instanz. Beim
Umzug gilt dieser Abschnitt dort erneut — ebenso das, was am Hetzner-Server schon erfüllt ist
(Commit-Geschichte, 13.09.2026). Fest auf den Testserver eingetragen sind dabei die Adresse in
der Schlussmeldung von `update.sh` und `DOMAIN` in `scripts/deploy.sh`.

### 7.2 Frisches Littera-Backup

`littera_sav.mdb` ist ein Stand von 2010. Ohne die offenen Ausleihen startet das System mit „alles
verfügbar"; Ausweisnummern und Exemplare nach 2010 fehlen, rund 1.350 Titel haben keine Signatur.
Anforderungen in [littera_schema_befund.md](littera_schema_befund.md). Vorher B7 (8.5). Die
teuerste offene Position vor dem Echtstart — früh bei der Schule anfragen.
**Reihenfolge:** Littera-Personen und -Ausleihen im selben Lauf übernehmen, **bevor** ein echter
LUSD-Import läuft. Der Littera-Personenlauf erkennt Schüler aus der LUSD nicht und legt sie ein
zweites Mal an ([SCRIPTS.md](SCRIPTS.md), Abschnitt 0). Das Geburtsdatum im Backup ist die Brücke
für den späteren LUSD-Abgleich — vor dem Lauf prüfen. Die Übernahme läuft am Schulserver auf
einer neu angelegten Datenbank (entschieden am 28.09.2026); muss sie wiederholt werden, wird die
Datenbank neu angelegt ([SCRIPTS.md](SCRIPTS.md), Abschnitt 1).
**Vor dem Personenlauf:** Im frischen Backup nachsehen, ob die Tabelle `FremdLeserNummer` gefüllt
ist (im Stand von 2010 ist sie leer). Sie trägt die Nummern, die die Ausweise beim Scannen liefern;
nur mit ihr funktionieren die vorhandenen Ausweise ohne Neudruck. **Ist sie leer, muss jeder Ausweis
neu gedruckt werden:** Kein Ausweis liefert beim Scannen die Lesernummer. Der Lauf trägt dann
die Lesernummer als Ausweisnummer ein; mit einem Buch kollidiert sie nach dem Neuaufbau nicht, weil
Bücher den 13-stelligen EAN-13 tragen (in der Generalprobe eine Nummer doppelt, zwischen zwei
Lesern — der zweite bekam eine `A-`-Nummer). Ist sie gefüllt, stehen die einzelnen Personen ohne Karte
mit „keine Karte in FremdLeserNummer" im Protokoll des Laufs.
**Die Sicherungen vom 01. und 02.09.2026** (nachgesehen am 28.09.2026): In `~/Downloads`
liegen seit dem 09.09.2026 zwei Littera-Sicherungen
(`littera_sicherung_01_09_2026_14_04_50.7z` und `littera_sicherung_02_09_2026_13_29_22.7z`,
856 KB und 602 KB); auf dem Schreibtisch liegt nur eine Kopie der ersten (nachgesehen am
02.10.2026). Jede enthält eine `.bak` (16,7 MB und 13,4 MB) und ist verschlüsselt.
Name und Form sind nach dem Littera-Handbuch („Datensicherung mit SQL Server-Datenbank",
lokaler Server) die Sicherung der **SQL-Server-Fassung**: Die `.bak` ist dann eine
SQL-Server-Sicherung, keine Access-Datei, und `mdb-export` liest sie nicht. Zum Zurückspielen
nennt das Handbuch: „Man benötigt das Kennwort welches bei der Installation des Servers erfasst
wurde" — das kennt, wer Littera an der Schule eingerichtet hat. Ob es die `.7z` selbst öffnet oder
erst beim Zurückspielen mit `SqlServerRestore.exe` gebraucht wird, sagt das Handbuch nicht. Kennt
es niemand, bleibt die Littera-Hotline. Ohne Kennwort blieben nur die Auswertungen von Littera —
Leserliste (mit dem Leserdatenaustausch, laut Handbuch lizenzabhängig), Medienliste, Liste der
verliehenen Medien —, die sich teils als Datei ausgeben lassen („Export des Druckbildes"). Ob sie
die Nummern tragen, die die Übernahme braucht, ist nicht geprüft; die Übernahme liest die Tabellen
der Datenbank, ein Weg über Auswertungen hieße einen neuen Importer (nachgelesen am 28.09.2026).
**Nächste Schritte:** (1) dieses
Kennwort erfragen; (2) die `.bak` auf dem eigenen Rechner in einen SQL Server einspielen und die
dreizehn Tabellen aus Abschnitt 1 von [SCRIPTS.md](SCRIPTS.md) als CSV ausgeben, Spaltennamen und
Datumsformate gegen den Importer prüfen, der bisher nur `mdb-export` kennt; (3) lokal und nur
lesend messen, nicht auf dem Testserver, am einfachsten mit der Generalprobe über das
CSV-Verzeichnis ([SCRIPTS.md](SCRIPTS.md), Abschnitt 1b): `FremdLeserNummer` gefüllt, offene Ausleihen, Titel mit
Schlagworten (4.20). Ob es die volle Datenbank ist, zeigt erst das Einspielen; dass die jüngere
Datei kleiner ist, ist ungeklärt.
**Etiketten vor dem Lauf prüfen:** `LITTERA_CSV_DIR=… go test ./internal/littera/` gegen die
ausgegebenen Tabellen. Der Test vergleicht seit dem 28.09.2026 den vollen EAN-13 jedes Etiketts
mit der Rechnung (`pruefeEtiketten`); bis dahin prüfte er nur die Nummer und sah nicht, dass die
Übernahme für Nummern unter sechs Stellen falsch rechnete (1b594d1e). Exemplare, deren Etikett
schon ein anderes trägt (2010: 8), stehen nach dem Lauf unter „Fehlende Etiketten"
([SCRIPTS.md](SCRIPTS.md), Abschnitt 1).

**Lesergruppen ohne Zuordnung vor dem Umstiegstag klären.** Seit dem 28.09.2026 übernimmt der
Lauf jeden Leser ([SCRIPTS.md](SCRIPTS.md), Abschnitt 1); eine Lesergruppe ohne Zuordnung hält
ihn an, bevor er schreibt. In der Sicherung von 2010 ist das „Undefinierte Untergruppe" (5
Personen, 21 Ausleihen). Mit dem frischen Backup die Generalprobe fahren ([SCRIPTS.md](SCRIPTS.md),
Abschnitt 1b; mit der Sicherung von 2010 am 28.09.2026 bestanden, 15.612 von 15.615 Ausleihen);
nennt sie Gruppen, setzt die Bücherei diese Personen in Littera in ihre Gruppe, bevor die
Sicherung für den Umstieg gezogen wird.

### 7.3 S3-Auslagerung der Backups

`S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY` und `S3_BUCKET` sind leer (13.09.2026); alles
liegt auf einer Platte. Nur EU oder Schulträger. Der Code ist fertig bis auf das Löschen
(nachgesehen am 28.09.2026): `uploadBackupToS3` in `jobs/backup.go` lädt jede Nachtsicherung
hoch, die Rotation gilt nur dem lokalen Verzeichnis. Ohne Löschregel am Speicher bliebe dort jede
Sicherung mit allen Personen ihres Stands unbegrenzt liegen. Beim Einrichten eine Löschregel am
Speicher setzen, die der Aufbewahrung der Nachtsicherung folgt (die jüngsten 14, dazu je
Kalenderwoche eine für 12 Wochen; `jobs.BehalteNaechte`, `jobs.BehalteWochen`), oder die Rotation
im Code auf den Speicher ausdehnen. **Entschieden am 28.09.2026:** zuerst beim Schulträger fragen, ob er einen Speicher
außer Haus stellt (Frage oben); ein Speicher des Schulträgers braucht keinen Vertrag mit einem
Dritten. Einen anderen Kopierweg als S3 gibt es im Programm nicht.

### 7.4 Manuelle Restore-Probe an einem fremden Ziel

Die automatische Wochenprobe im Container lief am 13.09.2026 erfolgreich; sie ersetzt die
manuelle Probe nicht. Anleitung: [resilience_and_recovery.md](resilience_and_recovery.md),
Abschnitt 2e — dabei die neuen Befehle erproben, die bisher nur am Text geprüft sind. Sinnvoll
nach S3 oder am Schulserver. Machen soll sie die Vertretung allein mit dem Pflegekonzept (9.9);
sie wartet also auf den Schulserver und auf die benannte Vertretung.

### 7.5 Externes Uptime-Signal

Fällt der Server ganz aus, meldet es niemand. Ein externer Monitor ruft alle 5 Minuten `/health`
ab ([DEPLOYMENT.md](DEPLOYMENT.md), Abschnitt 7.0). Etwa fünf Minuten Aufwand; beim Umzug neu
einrichten. Am Schulserver geht das erst, wenn der Eingang aus 4.23 gebaut ist, der `/health`
von außen durchlässt, und Port 443 frei ist.

### 7.6 Ruleset `main`

PR-Pflicht entfernen (Solo-Entscheidung 30.07.2026), „Block force pushes" und „Restrict
deletions" anlassen. Am 03.10.2026 trägt das Ruleset noch `pull_request`; Pushes gehen über den
Admin-Bypass.

### 7.7 Abnahmen

Ablauf in [abnahme_checkliste.md](abnahme_checkliste.md), vorher ein Backup. **Der Termin hängt am
Schulserver** (nachgesehen am 28.09.2026): Die Abnahme prüft echte Daten (Kopf der Checkliste),
echte Schülerdaten gehören nur auf den Schulserver, und der LUSD-Import kommt erst nach der
Littera-Übernahme, die dort läuft (7.2).

- Flows 1–3 mit dem Sekretariat: LUSD-Import, Versetzung (vor dem Schuljahreswechsel),
  Klassensatz erledigen. Dabei um Geburtsdatum und Eintrittsdatum im LUSD-Bericht bitten. Ein
  LUSD-Import mit echten Schülern erst nach der Littera-Übernahme (7.2).
- Flow 4 (Altbestand-Etiketten) entfällt mit dem Neuaufbau (entschieden am 28.09.2026): Die
  Littera-Übernahme setzt den Etikett-Vermerk selbst (`internal/littera/schreiber_bestand.go`).
- Flow 5: Selbstanmeldung einer Lehrkraft.
- Danach: Ergebnis hier eintragen; bei Parser-Auffälligkeiten die echte LUSD-Datei anonymisiert als
  Testfixture sichern.

### 7.8 Am Server nachsehen (lesend, Einzeiler)

- „Neuer Text" im Ausweis-Layout: laut Serverlesung vom 13.09.2026 in keinem Wert von
  `system_einstellungen`. Bestätigen, dann erledigt.
- Sind die Admin-Konten deaktiviert? Ist `/app/uploads/fotos` leer? Gibt es Lehrkräfte mit
  Platzhalter-Mail `@lehrer-umzug.invalid`? Braucht `repair_fach_kategorie.sql` einen zweiten
  Lauf?
- Tragen Titel die Nichtsortierzeichen der DNB? Sie umschließen den Artikel am Anfang
  („Der kleine Hobbit"), sind nicht zu sehen und standen bis zum 02.10.2026 in jedem Titel,
  der mit Artikel aus einer ISBN-Abfrage oder einer Bestellung per ISBN entstand; seitdem
  entfernt sie das Einlesen. Bei 0 erledigt, sonst bereinigt eine Migration nach dem Muster
  von 154:
  `docker exec bibliothek-db psql -U postgres -d bibliothek -c "SELECT count(*) FROM buecher_titel WHERE titel ~ U&'[\0098\009C]';"`
- Erreicht der Server die DNB? Ohne sie bringt die ISBN-Abfrage der Buchmaske keine Angaben
  und der Cover-Abgleich kein Cover; gespeichert wird trotzdem. Am Abbild vom 01.10.2026 geprüft, Ausgabe „erreichbar":
  `docker exec bibliothek-backend sh -c 'wget -q -T 8 -O /dev/null "https://services.dnb.de/sru/dnb?version=1.1&operation=explain" && echo erreichbar || echo nicht erreichbar'`

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

### 8.3 E5: Zahlungswege der Schülerbücherei — und wer eine Zahlung einbucht

Für die Schülerbücherei nennt der Brief keinen Weg, sondern „(Bankverbindung des Schulträgers
nicht hinterlegt)" (`pdf/zahlungsweg.go`). Es fehlen Konto und Verwendungszweck oder
Kassenzeichen, und ob die Schule Geld des Schulträgers bar annehmen darf (in öffentlichen Kassen
meist nur eine eingerichtete Zahlstelle) und wie es zu ihm kommt. **In zwei Schritten fragen**
(Vorschlag vom 24.09.2026):

1. **Die Schule** (Büchereileitung, Sekretariat): Wie wurde Ersatz für Bücher der
   Schülerbücherei bisher bezahlt, wohin ging das Geld? Wer bucht eine Zahlung ein, die auf dem
   Kontoauszug oder im Finanzbericht steht — für beide Töpfe? „Bezahlt" verbucht heute eine
   Barzahlung am Tresen. Keine Bauarbeit vor der Antwort; eine erfundene stünde als Vorgang in
   der Akte.
2. **Der Schulträger** (Fachbereich Schule und Betreuung des Hochtaunuskreises), als Vorschlag
   zum Bestätigen, an die bisherige Praxis angepasst: „Wir bieten beides an: Überweisung auf Ihr
   Konto mit der Rechnungsnummer als Verwendungszweck, und Barzahlung gegen Quittung. Das Bargeld
   geben wir einmal im Monat mit einer Liste an Sie weiter. Ist das zulässig? Welches Konto und
   welches Kassenzeichen gelten?"

**Engpass:** Ohne Antwort kein 5.4.

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
- **B7** Löschkonzept gegenüber Littera — vor der Littera-Übernahme (7.2). Dazu gehört, wie lange
  die Littera-Sicherung vom Umstiegstag aufgehoben wird (entschieden am 28.09.2026: hier statt als
  eigene Frage nach einem Rückweg; für das Ende der Pflege gibt es seit dem 24.09.2026 keinen
  Rückweg zu Littera).

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

### 9.9 Zwei Bedingungen neben der Mängelliste

**Entschieden am 24.09.2026 für das Pflegekonzept, am 28.09.2026 für den DSGVO-Nachweis:**
Beides wird jetzt erarbeitet, das Pflegekonzept als Wartungshandbuch, weil die Antworten von
Schule und Schulträger den Echtstart bestimmen und beide Dokumente sie beeinflussen.

Die Einschätzung am Ende des Protokolls nennt zwei Punkte, die in keinem der zwölf Mängel
stehen:

> „Ein Nachweis der DSVGO-Konformität liegt nicht vor.
> Hosting- und Programmpflegekonzepte sind nicht geplant. Dies könnte ein Ausschlusskriterium
> sein."

- **Nachweis der DSGVO-Konformität.** Der Entwurf steht seit dem 28.09.2026:
  [datenschutz/nachweis.md](datenschutz/nachweis.md) — eine Übersicht zum Weitergeben mit den
  Unterlagen (VVT-Entwurf, Datenschutzhinweis, PII-Matrix), den Löschfristen, dem Test hinter
  jeder Zusage, dem Ablauf bei einer Datenpanne, den bekannten Lücken und dem, was bei der
  Schule liegt. Offen ist die Beschlussfassung der Schule (8.5, B1–B7).
  **Die Frist bis zum Sperrbildschirm:** Der Entwurf nennt die Vorgabe des Programms, 15
  Minuten (Abschnitt 5, „Zugang"). Gewünscht sind an der Schule 8 Stunden ohne Bedienung
  (01.10.2026). Das Feld nimmt 0 bis 1440 Minuten, 480 sind am Stack nachgestellt; die Vorgabe
  bleibt 15, der Wert wird am Schulserver einmal unter Einstellungen → Datenschutz & Sitzung
  eingetragen. Der Nachweis nennt dann die Zahl der Schule. Mit 480 Minuten greift die Sperre an einem
  Schultag nicht; für den unbeaufsichtigten Platz bleibt das Leeren der Theke nach 5 Minuten.
  Das gehört zur Beteiligung des Datenschutzbeauftragten (8.5, B4).
- **Hosting- und Programmpflegekonzept.** Der Entwurf steht seit dem 24.09.2026, ergänzt am
  28.09.2026 um die Aufbewahrung der Sicherungen, den Datenweg beim Wechsel und die
  Kontakte für eine Datenpanne: [PFLEGEKONZEPT.md](PFLEGEKONZEPT.md) — mit den drei am
  24.09.2026 beantworteten Fragen
  (Betrieb, Pflege mit Vertretung, Ende der Pflege), den wiederkehrenden Aufgaben mit Takt, den
  zwei Handgriffen der Vertretung und der Messung, ob jemand anderes das Programm weiterführen
  kann; am 28.09.2026 mit den Antworten zu den vier offenen Stellen des Entwurfs (für welche
  Schulen, Fehler melden, Betriebssystem und Docker, nur Releases) und der Vorlage für das Blatt.
  Offen:
  1. **Das Blatt bei der Schule** (Abschnitt 7.3 des Entwurfs) — ausfüllen bei dir, Vorlage in
     [blatt_vorlage.md](blatt_vorlage.md). Vorschlag: die zwei Schlüssel in einem Passwortmanager
     und als Papier im verschlossenen Umschlag im Tresor der Schule, nie per E-Mail.
  2. **Die Arbeitsnotizen der Entwicklung** (am 24.09.2026 219 Einträge) entlang der Gliederung
     des Entwurfs ins Repository — bei mir.
  3. **Die Probe:** Die Vertretung macht die Wiederherstellung an einem fremden Ziel (7.4) allein
     mit dem Dokument.

**Beides ist Voraussetzung für ein „nutzbar", nicht Beiwerk.**
