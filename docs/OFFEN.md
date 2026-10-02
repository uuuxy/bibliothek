# Offene Arbeit

Stand: 02.10.2026

**Die eine Liste.** Hier steht alles, was noch zu tun, zu prüfen oder zu entscheiden ist — Code,
Betrieb und Schule. Einen zweiten Ort gibt es nicht. Erledigtes wird gelöscht, nicht archiviert:
Die Geschichte steht in den Commit-Nachrichten und in `git log -p docs/OFFEN.md`.

Andere Dokumente erklären (Konzept, Anleitung, der Katalog der Bugklassen in
[sweeps.md](sweeps.md)), führen aber keine eigene Offen-Liste.

---

## Was jetzt dran ist

**Die Sichtung vom 16.09.2026** (Abschnitt 9): Die zwei Bedingungen aus 9.9 — DSGVO-Nachweis
sowie Hosting- und Pflegekonzept — sind am 23.09.2026 zurückgestellt. Das Pflegekonzept ist am
24.09.2026 umentschieden, der DSGVO-Nachweis am 28.09.2026; beide liegen als Entwurf vor.

**Entschieden am 28.09.2026: Neuaufbau am Schulserver.** Der Echtbetrieb beginnt mit einer leeren
Datenbank und der Littera-Übernahme; `tabula_rasa.sql` und `repair_titel_dubletten.sql` sind
entfernt, das Aufräumen vor einem zweiten Littera-Lauf und der Etiketten-Lauf für den Altbestand
entfallen.

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
     darunter bleiben im Bild. Dann den Titel leeren und „Speichern" drücken. Erwartet: Die
     Meldung oben rechts liegt nicht über dem Knopf.

**Im Code:** Die Festlegung vom 28.09.2026 — bis zu den drei Antworten nur, was einen Termin
hat — ist am 29.09.2026 für die Punkte unter 1. aufgehoben. Einen Termin hat Node 26 ab dem
28. Oktober 2026 nach der Regel „immer die aktive LTS"
([PFLEGEKONZEPT.md](PFLEGEKONZEPT.md), Abschnitt 4). Die CI steht seit dem 28.09.2026 fest auf
`ubuntu-24.04`, der Wechsel auf Ubuntu 26 hat damit keinen Termin mehr (5.10). Die Reihenfolge
(freigegeben am 23.09.2026; die Stellung von 5.3 ist der Vorschlag vom 24.09.2026, die
Reihenfolge unter 1. die vom 29.09.2026).

1. Es folgt 5.21 (Palettenfarben, Bildschirm für Bildschirm). Zu 4.24 ist am 01.10.2026
   entschieden: kein dritter Eigentümer im Programm, die Frage geht an die Bücherei (oben
   unter 2.). Zu 4.28 (Anmelden ohne Mailserver) steht die Entscheidung aus; erst klären.
2. **5.3** — muss stehen, bevor ein echter Bescheid übergeben wird; echte Bescheide gibt es erst
   im Echtbetrieb.
3. Nach der Antwort zu 8.3: **5.4**.
4. **5.10** (Gates und Werkzeuge) und Abschnitt 6 nur mit Anlass.

Vor dem Echtstart außerdem: 5.31 (`update.sh` für den Schulserver) und der Eingang für die Seite
der Lieferanten (4.23).

**In der Doku:** Pflegekonzept und Datenschutz-Nachweis (9.9) stehen als Entwurf. Im
Pflegekonzept sind seit dem 28.09.2026 die Antworten zu seinen vier offenen Stellen eingetragen,
die Vorlage für das Blatt liegt in [blatt_vorlage.md](blatt_vorlage.md). Es folgen die
Arbeitsnotizen ins Repository und die Probe durch die Vertretung.

Mit der Littera-Übernahme (7.2) kommen das Eigentum je Exemplar (4.24) und alle Schlagworte und
Interessenkreise der Titel; ob Littera Verweise zwischen Schlagworten führt, zeigt erst die
Sicherung von 2026 (4.20).

**Parallel auf der Schulseite:** Abschnitte 7 und 8 — zuerst der Schulserver samt Speicher außer
Haus (7.3), das Passwort der Littera-Sicherungen (7.2), die Anfragen E1, E2 und zu den
Zahlungswegen (8.1–8.3), B3 und B4 (8.5) und ein Termin für die Abnahmen, sobald der
Schulserver steht (7.7). Einen echten LUSD-Import erst nach der
Littera-Übernahme (7.2).

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

### 4.28 Anmelden ohne den Mailserver der Schule

Die Anmeldung prüft jedes Passwort beim Mailserver der Schule (`auth/imap.go`). Ist er nicht
erreichbar, antwortet `/login` mit 503, und niemand kann sich neu anmelden
(`TestLoginHandler_MailserverAusfallIstKeinFalschesPasswort`). Wer schon angemeldet ist, arbeitet
weiter: Die Sperre nach Inaktivität geht seit Migration 155 mit dem Passwort auch ohne
Mailserver auf, über den Prüfwert der laufenden Anmeldung
([SECURITY.md](SECURITY.md), „Woran die Zugangsdaten geprüft werden").

**Offen:** Soll auch eine neue Anmeldung ohne Mailserver gehen? Dafür müsste der Prüfwert über
das Abmelden hinaus bleiben — ein dauerhafter Passwort-Hash je Konto, den es seit Migration 012
nicht gibt ([arc42/09](arc42/09-architekturentscheidungen.md), A2). Entschieden am 01.10.2026:
getrennt von der Sperre zu entscheiden. Vor einer Empfehlung zu klären: wie oft der Mailserver
der Schule ausfällt, und wie Littera und andere Programme es halten.

---

## 5. Abarbeitbar (Kategorie B)

### 5.3 Die Übergabe schließt die Forderung ab (nach 4.4)

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
  **Entschieden am 24.09.2026, im selben Umbau:** „Jahrgang unbekannt" wird eine eigene Vorgabe
  (NULL) statt 5 bis 10 — heute ist beides nicht zu unterscheiden, und wer den
  Mehrjahresband-Schalter (Migration 134) auf einem Titel mit der Vorgabe umlegt, bekommt die 10.
  Die drei Leser der Spanne (Mahnwesen „Jahrgang", Inventur, Portal-Filter) lernen „unbekannt"
  mit. Vorher am Testserver messen.
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

- **ISBN-10 und ISBN-13 desselben Buchs:** Die Normalform trennt beide bewusst (Migration 133),
  die Littera-Übernahme behält eine gültige ISBN-10. Seit dem 25.09.2026 rechnet die Bestelltür
  um und schlägt den Titel unter der anderen Länge vor (`isbnutil.AndereForm`, der Zwilling
  von `isbnFormen.js`). Nur die Schreibweise vergleichen weiter die Markierung
  „Vorhanden" der Bestellsuche (`sammleExistierendeISBNs`) und die Dublettenkontrolle der
  Maske: Ein DNB-Treffer, dessen ISBN-10 im Katalog steht, heißt in der Trefferliste „Neu",
  erst der Klick führt zur Frage. Gemessen am Testserver am 23.09.2026 (lesend): 100 Titel mit
  ISBN-10, 9.743 mit ISBN-13, 4 Paare mit gleichem Kern — alle aus der Littera-Übernahme vom
  15.07.2026, ohne Exemplare. Unter `3499500252` und `9783499500251` stehen zwei verschiedene
  Bücher („Heinrich Mann" und „Frédéric Chopin", rororo): Die ISBN-10 trägt ein falsches
  Prüfzeichen (richtig wäre `3499500256`; die Prüfung `KlaereISBN` der Übernahme gibt es seit dem
  04.08.2026), und die Rechnung führt von ihr trotzdem auf die ISBN-13. Deshalb wird
  vorgeschlagen, nicht still zusammengeführt.
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
- Jede ISBN-Abfrage legt eine Cover-Datei ab (gefunden am 01.10.2026, am lokalen Stack
  gezählt): `GET /api/lookup/{isbn}` speichert das Cover als `uploads/cover_auto_…`, auch wenn
  danach nichts gespeichert wird. `POST /api/books` schlägt wegen des leeren Listenpreises
  noch einmal nach und legt eine zweite Datei ab, die kein Titel trägt. Zwei Abfragen und zwei
  Speicherversuche derselben ISBN ergaben vier Dateien. Löschen aus der Maske
  (`DELETE /api/buecher/titel/{id}`) lässt die Datei des Titels liegen, `DELETE /api/books`
  nimmt sie mit (`sammleLokaleCoverPfade`). Der Ordner wächst; Kategorie B.
- Googles Ersatzbild gilt als Cover (gefunden am 02.10.2026, am lokalen Stack gesehen): Hat
  ein Titel kein gespeichertes Cover, fragt die Oberfläche über den eigenen Proxy erst Google
  Books, dann OpenLibrary (`coverKandidaten` in `utils/coverSrc.js`). Google antwortet auf eine
  ISBN ohne Bild mit Status 200 und einem grauen Bild „image not available" (PNG, 128 × 170 px,
  1.269 Byte). `holeUndKonvertiereCover` (`api/image_caching.go`) legt es als Cover ab, die
  Prüfung in `ui/BuchCover.svelte` sieht nur Bilder unter 10 px als leer an. Zu sehen ist dann
  das Ersatzbild statt der Initiale, und OpenLibrary wird nicht mehr gefragt. Betroffen sind
  Katalog (`BuchKarte`, `BookTableZeile`), Buch-Akte, Portal (`KlassenBuchKachel`) und das
  Bestellfenster; die Ausleihliste der Leserakte zeigt nur gespeicherte Cover
  (`nurGespeichert`). Der Cover-Abgleich des Servers fragt die Google-Books-API und ist nicht
  betroffen. Abhilfe wäre, das Ersatzbild im Proxy zu erkennen und wie einen Fehlschlag zu
  behandeln; bereits abgelegte Dateien unter `uploads/covers` blieben bis dahin liegen.
  Kategorie B.
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
- Die Exemplarliste der Buchmaske führt ausgesonderte und bestellte Exemplare mit (gesehen am
  01.10.2026 im Browser): Über der Liste steht „Exemplare (9)" neben dem Bestand 5, beide
  Arten heißen „Gesperrt", und „Exemplar löschen" an einem ausgesonderten antwortet
  „exemplar nicht gefunden oder bereits ausgebucht". Kategorie B.
- **Kein neues Buch, solange die Katalogdienste nicht antworten** (gefunden am 01.10.2026, am
  Stack nachgestellt; Kategorie B). `ergaenzeBuchMetadaten` fragt vor dem Anlegen DNB, Google
  Books und OpenLibrary, sobald Titel, Autor, Cover oder Listenpreis fehlen; bei einem von
  Hand eingetragenen Buch fehlt fast immer das Cover. Antwortet keiner der Dienste
  (nachgestellt mit Adressen, die Pakete verschlucken), meldet die Maske nach 10 s
  „Netzwerk-Timeout: Die Anfrage hat zu lange gedauert.", der Server bricht ab, gespeichert
  wird nichts; die Eingaben bleiben stehen. Mit Cover und Listenpreis antwortet dieselbe Tür
  sofort mit 201. Betrifft jeden Ausfall des Internets und einen Server, dessen Netz die
  Dienste nicht erreicht (7.8). Abhilfe: Das Anlegen wartet nicht auf die Dienste — die Maske
  hat die Angaben bei der Eingabe der ISBN schon geholt. Damit entfiele auch die zweite
  Cover-Datei aus dem Punkt darüber.
- Maske „Neues Buch": „Speichern" ist gesperrt, solange einem Bibliotheksbuch die Signatur
  fehlt; den Grund nennt das Feld Signatur („Speichern ist bis dahin gesperrt"). M3, Dialogs,
  Guidelines, zur bildschirmfüllenden Maske: „Don't disable the confirmation button" und
  „Only trigger an additional basic dialog if the action fails". Zu entscheiden wäre, ob der
  Knopf bedienbar bleibt und ein Klick zum Feld führt (Lesung 02.10.2026, Kategorie C).
- Druck-Center, Buch-Etiketten: Bei 1280 px Fensterbreite schiebt sich die A4-Vorschau (feste
  140 mm, `LabelPreview.svelte`) über den rechten Rand der Layout-Optionen; die Pfeile der
  Auswahlfelder liegen darunter (Sichtprüfung 24.09.2026). Mit 5.21 für diesen Bildschirm.

### 5.10 Gates und Werkzeuge

- Die Schema-Gegenrichtung ist blind für UNIQUE, Teilindizes und RESTRICT.
- Kein Gate gegen unbegrenzte Listen-Endpunkte.
- Kein Rückweg für ältere Sicherungen beim Wechsel des `BACKUP_ENCRYPTION_KEY`.
- `e2e/icon-trefferflaechen.spec.js` und `e2e/icon-tooltips.spec.js` messen die Bestellhistorie,
  legen aber keine Bestellung an: Allein oder ohne eine `bestell…`-Spec davor laufen sie in die
  Zeitüberschreitung (lokal am 23.09.2026 und am 25.09.2026, Bestellhistorie leer; der globale
  Teardown löscht die E2E-Bestellungen). In der
  vollen Suite legt eine alphabetisch frühere Spec sie an. Nach dem Muster von `seedBenutzer`
  selbst anlegen.
- `e2e/kontrast.spec.js` scheitert im zweiten Lauf der Suite auf derselben kleinen Datenbank
  (gefunden am 01.10.2026, an einem eigenen Stack zweimal nachgestellt):
  `page.getByTitle('Leserdatei')` trifft neben dem Menüpunkt auch die Kachel eines Buchs, dessen
  Titel so beginnt, und `e2e/leserdatei.spec.js` lässt „Leserdateibuch …" samt Exemplar und
  Ausleihe liegen. Auf frischer Datenbank läuft kontrast vor leserdatei und ist grün. Den
  Menüpunkt genau treffen oder das Buch aufräumen.
- `e2e/schlagworte-roundtrip.spec.js` hängt an der Geschwindigkeit des Rechners (gefunden am
  02.10.2026). `speichere()` wartet auf die Meldung „Buch erfolgreich gespeichert!" mit
  `.first()`; eine Meldung steht 5 s, ab dem zweiten Speichern genügt die des vorigen, und der
  nächste Schritt beginnt, während die Maske noch schließt. Sie blendet 200 ms aus und nimmt
  in dieser Zeit keine Eingabe an (`inert`); ein Klick auf den Titel holt in dieser Zeit
  dieselbe Maske zurück. Nachgestellt an beiden Ständen: Speichern, Titel sofort wieder
  anklicken, Schlagwort tippen — das Feld bleibt leer, gespeichert wird ohne das Wort. Im Test
  wurde das sichtbar, als eine Meldung über „Speichern" lag und der Klick 900 ms später kam
  (vierter Schritt rot). Von Hand ist das Fenster von 200 ms nicht zu treffen. Abhilfe: nach
  dem Speichern warten, bis die Überschrift „Buch bearbeiten" fort ist. Kategorie B.
- Gegen den Entwicklungsserver (`E2E_BASE_URL=http://localhost:5173`) sind drei Specs rot, die
  am gebauten Stand grün sind (gemessen am 02.10.2026). Die Barrierefreiheits-Prüfung des
  Mahnwesens läuft in die Zeitüberschreitung, und `e2e/zugangsbuch.spec.js` findet den Zugang
  nicht in der Tabelle des Landes — beide auch ohne Änderung an den Quellen. Die Prüfung der
  Leserdatei liegt bei 30 s an der Grenze (ein Lauf grün, einer rot; am gebauten Stand 6 s).
  Ursache beim Zugangsbuch nicht untersucht. Ein Lauf gegen den Entwicklungsserver ist
  deshalb nie ganz grün.
- Die Prüfung der Wegweiser (`api/betriebsbereitschaft_wegweiser_test.go`) liest nur die Texte
  der Selbstprüfung. Drei Hinweise der Oberfläche nennen einen Ort, den es so nicht gibt
  (gefunden am 01.10.2026): `BestelllinkHinweis.svelte` schickt für die öffentliche Adresse
  nach „Einstellungen → Allgemein → Schule", sie steht unter Einstellungen → Erreichbarkeit &
  Alarme; `WareneingangView.svelte` nennt das Zugangsbuch unter „System → Bestandsbücher",
  die Gruppe heißt Berichte; `backupStatusText.js` schreibt „Schlüssel in den Einstellungen
  unter Betriebsbereitschaft hinterlegen", dort lässt sich nichts hinterlegen, die
  Selbstprüfung nennt die `.env`. Texte berichtigen und die Prüfung auf die Texte der
  Oberfläche ausdehnen. Kategorie B.
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

Stand 02.10.2026: 1366 Fundstellen mit Tailwind-Palettenfarben (`slate`, `blue`, `emerald` …),
gehalten von der Ratsche `frontend/src/lib/frontend-hygiene-farben.test.js`; Neues entsteht nur
noch auf Rollen. Umstellen ist eine Umgestaltung, keine Umbenennung: Die Palette führt sechs
Textgraustufen, M3 zwei Rollen. Für „in Ordnung" und „Achtung" gibt es seit dem 24.09.2026 die
eigenen Rollen `success` und `warning` in `styles/rollen.css` (M3, „Define custom color
roles"). Vorschlag:
Bildschirm für Bildschirm, die größten zuerst, je Portion ein Commit, am gerenderten Bildschirm
geprüft. Das Muster steht in Buchformular und Bestellfenster: Zustände über ui/StatusChip, Cover
über ui/BuchCover, Rückmeldung beim Zeigen über den State-Layer statt `hover:bg-*`, ein Fehler
über den Fehlerzustand des Feldes statt eines farbigen Kastens. Für die übrige Anwendung
freigegeben am 23.09.2026.

Der Inventur-Bildschirm steht seit dem 24.09.2026 auf Rollen (`UnifiedInventory.svelte`, die
Scan-Rückmeldung in `inventur/ScanRueckmeldung.svelte`). Offen auf demselben Bildschirm: die
beiden Dialoge (`InventoryStartModal` 32, `InventoryFinishModal` 15) und der Fehlbestandsbericht
(`inventur/FehlbestandBericht` 14). `inventur/lib/bookHelpers.js` (48) sind Farbverläufe je Fach
für selbstgebaute Cover-Platzhalter; das gehört zu 6.2 (Cover über `ui/BuchCover`).

Beim Ansehen der Inventur am 24.09.2026 aufgefallen, jeweils am Code nachgesehen:

- Ein unbekannter Barcode zeigt am Scanner den rohen Fehlertext „exemplar für inventur-scan
  nicht ladbar: no rows in result set" (`GetExemplarForInventoryScan` hüllt `pgx.ErrNoRows` ein,
  `ladeExemplarFuerScan` gibt ihn mit 404 unverändert weiter). Der Status stimmt, der Satz nicht.

Dazu gehört die Leiste des Ausweisdrucks in der Leserdatei (`students/AuswahlAktionsleiste`,
dunkel in Palettenfarben): Seit dem 23.09.2026 gibt es für markierte Zeilen `ui/AuswahlLeiste`
(Schlagwort-Pflege). Beim Umstellen zu klären: wohin der Hinweis „ohne Ablaufjahr" und das Feld
„Ab Feld" kommen — beides passt nicht in die 64 px hohe Leiste.

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

Am 01.10.2026 und 02.10.2026 sind drei Kästen entfernt: die Ausleihliste der Leserakte
(256 px, drei Zeilen), die Positionen im Wareneingang (60 % der Fensterhöhe) und die Exemplare
in der Maske „Buch bearbeiten" (256 px, vier Zeilen). Die Listen zeigen jetzt alle Zeilen,
gescrollt wird der Bereich der Seite (`e2e/scrollbereiche.spec.js`). Kategorie B. Offen:

- **Weitere Kästen derselben Form, nicht durchgesehen:** `SignaturenView.svelte`
  (`max-h-112`), `OrderRecommendations.svelte`, `BestellWorkspace.svelte`
  (`max-h-(--rail-max)`), `LabelBarcodeSchritt.svelte` (`max-h-40`) und `LabelSettings.svelte`
  (`max-h-48`). Suchmuster: `max-h-` und `overflow-y-auto` in einer Zeile, 13 Treffer; die
  übrigen acht sind Auswahllisten, Vorschläge, Dialoge und das Eigenschaften-Feld des
  Designers. Je Bildschirm mit einer Menge wie an der Schule ansehen: Die Testdaten kennen
  höchstens drei Ausleihen je Leser, an der Schule sind es acht bis achtzehn.
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
- **Leserakte in einem Fenster unter 1200 px Breite** (gemessen am 02.10.2026, Seitenleiste
  ausgeklappt): Die Akte stellt Leserkarte (320 px) und Inhalt ab 1024 px nebeneinander
  (`StudentProfile.svelte`, `lg:grid-cols-[320px_minmax(0,1fr)]`). Der Ausleihliste bleiben
  dann 339 bis 512 px; Datum und Aktionen brauchen 290 px, das Miniaturbild 33 px. Dem Titel
  eines Lernmittels bleiben bei 1200 px 69 px und ab 1100 px abwärts nichts. Ab 1280 px sind
  es mindestens 149 px, rund zwanzig Zeichen (`e2e/ausleihliste-zeilen.spec.js` verlangt
  140 px). Zu entscheiden wäre, ob die Leserkarte unter 1280 px über den Inhalt rückt oder die
  Zeile dort zweizeilig wird.
- **Leserakte, Autor und Nummer des Exemplars:** Der Autor steht nur in der Sprechblase am
  Titel, die Nummer in Fenstern bis rund 1580 px ebenfalls (darüber hat sie ihre Spalte;
  gemessen bei ausgeklappter Seitenleiste). Die Sprechblase erscheint beim Zeigen mit der
  Maus; an einem Tablet ohne Maus sind beide Angaben in der Akte nicht zu sehen.
  Vorleseprogramme bekommen sie als unsichtbaren Text.
- **Leserakte, zwei Beschriftungen:** Der Reiter heißt „Ausleihen & Historie", zeigt aber nur
  die laufenden Ausleihen und die Vormerkungen. Unter dem Reiter „Stammdaten & Adresse" steht
  dieselbe Überschrift noch einmal; im Reiter „Gebühren & Schäden" heißt die Liste seit dem
  01.10.2026 „Forderungen".

---

## 6. Beobachten und Kategorie C (nur mit Anlass)

### 6.1 Beobachtungen

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
  weitere Auflage erscheint (Titelmaske: „Keine andere Auflage zugeordnet."). Kein Schaden; die
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
- Zwölf Dialoge und Formulare sperren „Abbrechen", solange ihre Anfrage läuft (etwa
  `StudentProfileDeleteModal.svelte`, `BescheidDialog.svelte`, `PapierkorbLoeschenDialog.svelte`;
  gezählt am 01.10.2026). M3 Dialogs, Guidelines: „Disable confirming actions until a choice is
  made. Dismissive actions are never disabled." Abbrechen bricht die laufende Anfrage am Server
  nicht ab; zu entscheiden wäre, ob der Knopf dann den Dialog schließen darf.
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
- 14 Bestandsstellen bauen ihr Cover selbst (Liste in `frontend-hygiene-cover.test.js`, darunter
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
- Das Nachschlagen (`GET /api/lookup/{isbn}`) liefert einen Untertitel (`subtitle`), den weder
  das ISBN-Feld noch der Scan im Buchformular übernimmt. Beim Anlegen trägt ihn der Server
  nach, beim Ändern nicht (`ergaenzeFehlendeMetadatenFuerAktualisierung`) — der Zwilling zu
  `8f7610aa`, der den Untertitel sonst mitnimmt.
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
- `github.com/jung-kurt/gofpdf` ist seit 2021 archiviert und steckt in 17 Dateien (ohne Tests,
  gezählt am 30.09.2026); gepflegt wird der Ableger `github.com/phpdave11/gofpdf`, den maroto
  mitbringt. Neue PDFs (5.3, 5.4) nicht mehr auf dem archivierten; die 17 beim fachlichen Anfassen
  umstellen, mit den PDF-Gates.
- Etikettenraster doppelt (`api/label_formats.go` und `etikettformate.js`), gehalten von
  `etikettformate-konsistenz.test.js`; am 31.08.2026 entschieden geparkt.
- Reste des Nie-verdrahtet-Sweeps: `inventur_sessions.gestartet_von` wird nie angezeigt;
  `abgaenger_jahr` in der Aktivlisten-Antwort ohne Konsument; bei den Geräten
  `ActionEvent.GeraetID` ohne Broadcast und mit Null-Zeitstempel.
- Cognitive Complexity: 32 Funktionen über 15 ohne Tests (Messung 05.09.2026); lohnend allenfalls
  `OverrideDueDateHandler` und `behandleAbgaenger`.
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
deletions" anlassen. Am 30.09.2026 trägt das Ruleset noch `pull_request`; Pushes gehen über den
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
- Erreicht der Server die DNB? Ohne sie lässt sich ein neues Buch ohne Cover oder Listenpreis
  nicht speichern (5.5). Am Abbild vom 01.10.2026 geprüft, Ausgabe „erreichbar":
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

**Entschieden am 23.09.2026: zurückgestellt.** Beides bleibt liegen, bis es ansteht; dann gelten
die Schritte und Fragen unten. **Am 24.09.2026 für das Pflegekonzept umentschieden: jetzt, als
Wartungshandbuch** (Vorschlag am Ende dieses Abschnitts); **am 28.09.2026 auch für den
DSGVO-Nachweis**, weil jetzt die Antworten von Schule und Schulträger den Echtstart bestimmen
und beide Dokumente sie beeinflussen.

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
