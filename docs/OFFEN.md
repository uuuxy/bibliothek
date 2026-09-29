# Offene Arbeit

Stand: 29.09.2026

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
3. **Der Nachweis von Hand für die Theke ohne Netz** (Abschnitt 2, Stufe 1 und 3 im echten
   Chrome) — zurückgestellt am 24.09.2026. Stufe 2 (die Tür per curl) mache ich am lokalen
   Stack, wenn der Nachweis ansteht.

**Im Code:** Die Festlegung vom 28.09.2026 — bis zu den drei Antworten nur, was einen Termin
hat — ist am 29.09.2026 für die Punkte unter 1. aufgehoben. `Cache-Control` (5.29) bleibt
zurückgestellt; am 29.09.2026 war es nicht abschätzbar. Einen Termin hat Node 26 ab dem
28. Oktober 2026 nach der Regel „immer die aktive LTS"
([PFLEGEKONZEPT.md](PFLEGEKONZEPT.md), Abschnitt 4). Die CI steht seit dem 28.09.2026 fest auf
`ubuntu-24.04`, der Wechsel auf Ubuntu 26 hat damit keinen Termin mehr (5.10). Die Reihenfolge
(freigegeben am 23.09.2026; die Stellung von 5.3 ist der Vorschlag vom 24.09.2026, die
Reihenfolge unter 1. die vom 29.09.2026):

1. Aus der Gruppe „kann still jemandem schaden" (entschieden am 28.09.2026) ist nur 5.29 offen,
   zurückgestellt (siehe oben). Es folgen der kleine Umbau 8.8 (Abholfrist), entschieden am
   28.09.2026, der Eigentumsvermerk je Exemplar aus Littera (4.24) und zusätzlich 12
   wöchentliche Stände der Sicherung (5.30). Dann 5.18 (Klassen als Stammdaten, mit Frage-Runde
   zur Oberfläche), dann 5.21 (Palettenfarben, Bildschirm für Bildschirm).
2. **5.3** — muss stehen, bevor ein echter Bescheid übergeben wird; echte Bescheide gibt es erst
   im Echtbetrieb.
3. Nach der Antwort zu 8.3: **5.4**.
4. **5.10** (Gates und Werkzeuge) und Abschnitt 6 nur mit Anlass.

Vor dem Echtstart außerdem: 5.29 (Antworten ohne `Cache-Control`), 5.31 (`update.sh` für den
Schulserver) und der Eingang für die Seite der Lieferanten (4.23).

**In der Doku:** Pflegekonzept und Datenschutz-Nachweis (9.9) stehen als Entwurf. Im
Pflegekonzept sind seit dem 28.09.2026 die Antworten zu seinen vier offenen Stellen eingetragen,
die Vorlage für das Blatt liegt in [blatt_vorlage.md](blatt_vorlage.md). Es folgen die
Arbeitsnotizen ins Repository und die Probe durch die Vertretung.

Mit der Littera-Übernahme (7.2) kommen die Littera-Schlagworte aus 4.20 und der Eigentumsvermerk je
Exemplar aus 4.24.

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

### 4.20 Littera-Schlagworte übernehmen

**Freigegeben am 23.09.2026:** die **Littera-Schlagworte** (MAB 710) beim nächsten Einspielen
des Backups mitnehmen (7.2). Heute liest der Import sie, leitet das Fach ab und verwirft sie.
Vorher messen, wie viele Titel welche tragen. Die Schlagworte selbst (Migration 138, Pflege mit
Verweisen, Portal-Filter) stehen in [FACHKONZEPT.md](FACHKONZEPT.md).

### 4.22 Datenweg beim Wechsel auf ein anderes Programm

Aufgefallen beim Datenschutz-Nachweis (28.09.2026), am Code nachgesehen. Für das Ende der Pflege
sieht das Pflegekonzept den Wechsel auf ein Kaufprogramm vor (Abschnitt 8). Mitnehmen lassen sich
heute die nächtliche Sicherung, eine vollständige PostgreSQL-Datenbank, und die Bestandsliste als
CSV (`GET /api/admin/books/export`, Einstellungen → Datenverwaltung: je Exemplar Titel, Autor,
Verlag, ISBN, Jahr, Kategorie, Barcode, Zustand). Leser, Ausleihen, Signaturen und Schlagworte
gibt das Programm in keiner Form aus, die ein anderes Programm einliest; ein Wechsel bräuchte
eine Umsetzung aus der Sicherung. Littera führt unter Dienstprogramme einen Ex- und Import der
Schlagworte. **Frage:** eine Gesamtausgabe bauen (Katalog mit Signaturen und Schlagworten,
Exemplare, Leser, offene Ausleihen) oder beim Ende der Pflege aus der Sicherung umsetzen?
Verwandt im Parkdeck (6.3): den Schlagwortkatalog als Datei aus- und einlesen.

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

### 4.24 Eigentumsvermerk je Exemplar aus Littera

Littera führt das Eigentum am Exemplar, in der Spalte `Exemplar.Eigentumsvermerk` (Freitext, 50
Zeichen). Die Übernahme (`internal/littera`) liest sie nicht, und
[littera_schema_befund.md](littera_schema_befund.md) nennt sie nicht unter „Was NICHT übernommen
wird" — entschieden ist das also nicht.

Gezählt in der Sicherung von 2010 am 29.09.2026 (`mdb-export`, nur Zählungen, Signatur des
Exemplars: `Sig1` beginnt mit „LMF" oder nicht), 61.580 Exemplare:

| Vermerk                                                                | LMF-Signatur | andere |
| ---------------------------------------------------------------------- | -----------: | -----: |
| Land                                                                   |       10.193 | 19.882 |
| Schulträger                                                            |          516 |  7.268 |
| Schule                                                                 |          112 |    581 |
| Förderverein                                                           |            0 |    171 |
| weitere (Verein, Projekt, Bibliothek, Dauerleihgabe, ein Personenname) |            0 |     76 |
| keiner                                                                 |       13.909 |  8.872 |

Das Programm leitet das Eigentum im Altbestand aus `ist_lernmittel` ab (`ExemplarTopfSQL`), und
die Übernahme setzt `ist_lernmittel` aus der LMF-Signatur (`pkg/lmf.Zerlege`). Nach dieser Regel
gehörten die 19.882 Exemplare, die Littera dem Land zuschreibt, dem Schulträger, und die 516
umgekehrt. Daran hängen der Vermerk auf einem neuen Etikett und, ob für ein verlorenes Buch ein
Bescheid an das Land geht (5.4, „Topf einer Forderung"). In der Medienliste vom 12.06.2026 tragen
16.878 von 67.109 Exemplaren einen Vermerk; die Sicherung von 2026 ist ohne Kennwort nicht lesbar
(7.2).

**Entschieden am 29.09.2026:** Der Vermerk kommt je Exemplar mit, als eigenes Feld am Exemplar,
zunächst ohne Wirkung auf Topf und Etikett. Gebaut wird es mit der Übernahme (7.2), denn sie läuft
einmal. Beim Bau zu klären: der eine Personenname, ein Personendatum in einem Bestandsfeld. Offen
bei der Schule, bevor der Vermerk eine Wirkung bekommt: was „Land" an einem Buch ohne
LMF-Signatur bedeutet.

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
- **Topf einer Forderung, eine Regel:** Der Bescheid nimmt `ist_lernmittel` am Titel, das
  Eigentum den Topf der Bestellung (`ExemplarTopfSQL`). Beim Altbestand ist das dasselbe (Littera
  führt das Eigentum anders, 4.24); bei
  einem aus dem anderen Topf bestellten Exemplar ginge das Geld an den, dem das Buch nicht gehört.
  Vorher am Testserver zählen (lokal 0):
  `SELECT count(*) FROM buecher_exemplare e JOIN buecher_titel t ON t.id = e.titel_id JOIN bestellungen_verlauf bv ON bv.id = e.bestellung_id WHERE bv.mittel <> CASE WHEN t.ist_lernmittel THEN 'land' ELSE 'schultraeger' END;`
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

  ```sql
  SELECT grade_level, jahrgang_von, jahrgang_bis, ist_lernmittel, signatur, titel
  FROM buecher_titel WHERE grade_level BETWEEN 1 AND 13
  ORDER BY ist_lernmittel DESC, signatur NULLS LAST, titel;
  ```

- **ISBN-10 und ISBN-13 desselben Buchs:** Die Normalform trennt beide bewusst (Migration 133),
  die Littera-Übernahme behält eine gültige ISBN-10. Seit dem 25.09.2026 rechnet die Bestelltür
  um und schlägt den Titel unter der anderen Länge vor (4.18, Stufe 4; `isbnutil.AndereForm`,
  der Zwilling von `isbnFormen.js`). Nur die Schreibweise vergleichen weiter die Markierung
  „Vorhanden" der Bestellsuche (`sammleExistierendeISBNs`) und die Dublettenkontrolle der
  Maske: Ein DNB-Treffer, dessen ISBN-10 im Katalog steht, heißt in der Trefferliste „Neu",
  erst der Klick führt zur Frage. Gemessen am Testserver am 23.09.2026 (lesend): 100 Titel mit
  ISBN-10, 9.743 mit ISBN-13, 4 Paare mit gleichem Kern — alle aus der Littera-Übernahme vom
  15.07.2026, ohne Exemplare. Unter `3499500252` und `9783499500251` stehen zwei verschiedene
  Bücher („Heinrich Mann" und „Frédéric Chopin", rororo): Die ISBN-10 trägt ein falsches
  Prüfzeichen (richtig wäre `3499500256`; die Prüfung `KlaereISBN` der Übernahme gibt es seit dem
  04.08.2026), und die Rechnung führt von ihr trotzdem auf die ISBN-13. Deshalb wird
  vorgeschlagen, nicht still zusammengeführt.
- **Etikett-Knopf bei jeder echten Nummer; ein Nachdruck gleicht dem alten Etikett**
  (entschieden am 28.09.2026, nicht gebaut). Der Knopf an der Exemplarkarte übergibt das
  Exemplar ans Druck-Center wie Wareneingang und Nachdruck, steht aber nur bei Nummern mit `B-`
  (`BookExemplarCard.svelte`, seit Juni 2026 ohne Begründung); auch `LMF-` bekommt keinen.
  Künftig: bei jeder Nummer außer den Platzhaltern `AUTO-` und `SYS-`. Die Platzhalter-Regel
  steht in derselben Datei schon zweimal (Farbe der Nummer, „Barcode scannen"); mit dem
  Etikett-Knopf wird sie eine Regel für alle drei Stellen. Littera-Exemplare tragen nach der
  Übernahme den EAN-13 ihres Etiketts als `barcode_id` (seit 1b594d1e richtig,
  `etikettAmBuch` in `internal/littera/schreiber_barcodes.go`) und die kurze Nummer in
  `erweiterte_eigenschaften` als `littera_exemplarnr`. Das Buchetikett setzt jeden Wert als
  Code 128 (oder als QR, wenn im Druck-Center gewählt) und druckt ihn unter „Exemplar-Nr."
  aus (`api/label_pdf.go`); ein Nachdruck sähe damit anders aus als das alte Etikett
  (13 Ziffern statt „58968", anderer Strichcode), scannt aber gleich. Dazu bauen: Ist die
  Nummer ein Littera-EAN (`dekodiereLitteraEtikett` in `internal/service/littera_etikett.go`),
  druckt das Etikett EAN-13 und die kurze Nummer. Ein Ersatzetikett aus Littera
  (`FremdBarcode`) bleibt Code 128. Die Übernahme setzt `etikett_gedruckt = true` außer bei
  einer neu vergebenen Nummer, die Nachdruck-Liste läuft also nicht voll.
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

### 5.18 Klassen als Stammdaten — wie die Lesergruppen in Littera

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

**Littera** (Handbuch, „Lesergruppen"): Klassen sind Stammdaten unter „Stammdaten →
Lesergruppen", mit Kurzbezeichnung und Bezeichnung unter einer Obergruppe; am Leser ist die
Klasse ein Pflichtfeld und wird nur aus dieser Liste gewählt. Gepflegt wird an der einen Stelle
(für die gleich gebaute Systematik: „einzelne Gruppen löschen, bearbeiten oder ergänzen"). Beim
Import aus der Schulverwaltung entstehen die Gruppen selbst.

**Entschieden am 24.09.2026, nicht gebaut — Stammdaten-Seite wie in Littera:** Die Tabelle
`klassen` wird die eine Liste, und alle Auswahllisten lesen sie statt `SELECT DISTINCT` über die
Verweise. Eine Pflegeseite (Einstellungen → LUSD & Versetzung) zeigt jede Klasse mit der Zahl der
Schüler, Klassensätze und Zuordnungen: „umbenennen in …" zieht über `ON UPDATE CASCADE` alles mit;
gibt es das Ziel schon, werden die Verweise dorthin umgehängt (Zusammenführen), und die alte
Klasse fällt weg; löschen nur ohne Verweis. Der LUSD-Import legt neue Klassen weiter selbst an.
Wer Schüler umhängt, ändert ihre LMF-Termine und Klassensätze mit — die Rückfrage nennt die
Zahlen. In Stufen, vorher eine Frage-Runde zur Oberfläche.

**Erweitert am 28.09.2026:** nicht nur Klassen, sondern alle Lesergruppen wie in Littera, je mit
Kürzel, Bezeichnung und Art (Schüler oder Kollegium) — auch Fachbereiche, Praktikanten, U-plus,
„Im Ausland". Anlass: Die Übernahme der 42 Konten ohne Schüler-/Lehrkraft-Gruppe (7.2). Im Schema
steht dafür schon eine Tabelle `lesergruppen` (`kuerzel`, `bezeichnung`), die kein Go-Code liest
oder schreibt; beim Bau wird sie die eine Liste oder fällt weg (vorher `count(*)` am Testserver).

**Reihenfolge entschieden am 28.09.2026:** nach dem Umstieg, an der Stelle im Plan oben. Der
Umstieg hängt nicht daran: Seit dem 28.09.2026 übernimmt der Lauf jeden Leser und schreibt die
Littera-Gruppe als Warnung ins Protokoll ([SCRIPTS.md](SCRIPTS.md), Abschnitt 1).

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

Stand 24.09.2026: 1426 Fundstellen mit Tailwind-Palettenfarben (`slate`, `blue`, `emerald` …),
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

### 5.29 API-Antworten tragen kein `Cache-Control`

Aufgefallen beim Rasterdurchgang vom 28.09.2026 (Frage 9 an der Datenschutz-Auskunft), nicht
Teil der Änderungen vom 24.09.2026. Den Kopf setzen nur Buchcover, Barcodes, die Barcode-Liste
der Theke, der Ereignisstrom und das Ausweisfoto — dieses mit `no-store`, „da die Bilder sensibel
sind" (`internal/crypto/upload_helpers.go`). Der Caddy-Block der Bibliothek (`update_caddy.sh`)
setzt keinen. Listen und Akten mit Namen und Adressen und die Auskunft als PDF darf der Browser
damit in seinem Festplatten-Cache ablegen, auf Rechnern, die mehrere Personen benutzen.
Abhilfe: `Cache-Control: no-store` als Vorgabe für `/api/`, wo der Handler nichts Eigenes setzt;
vorher nachsehen, ob die Theke ohne Netz auf dem Browser-Cache aufbaut.

### 5.30 Aufbewahrung der Sicherungen: zusätzlich 12 wöchentliche

**Entschieden am 28.09.2026, nicht gebaut.** Heute bleiben die letzten 14 Nachtsicherungen
(`rotateBackups(backupDir, 14)` in `jobs/backup.go`). Ein Fehler, der still Daten verändert — ein
falscher Import, ein Löschlauf, ein Reparaturskript — und erst nach den sechs Wochen der
Sommerferien auffällt, steckt dann in jeder vorhandenen Sicherung. Künftig bleiben dazu 12
wöchentliche Stände; gelöschte Personen stehen damit bis zu etwa drei Monate in den Sicherungen.
Vor dem Echtstart. Beim Bau mitziehen: die Auskunft (`dsgvoSicherungen`, `api/dsgvo_sicherungen_test.go`
wird rot), das VVT (Löschfristen der Tätigkeit 1),
[resilience_and_recovery.md](resilience_and_recovery.md) 1a, SECURITY („Rotation"),
PFLEGEKONZEPT 3.2 und den Datenschutz-Nachweis (Abschnitte 4 und 9); die S3-Kopie braucht
dieselbe Regel (7.3). Beim Bau klären (nicht nachgestellt): Eine zurückgespielte ältere Sicherung
bringt auch Personen zurück, die seit ihrem Stand von Hand endgültig gelöscht wurden; die
Löschläufe nach Frist greifen in der nächsten Nacht wieder, eine Löschung von Hand nicht.
Die Sicherung vor einem Update bleibt bei 30 Tagen (entschieden am 28.09.2026): Sie liegt nur
auf dem Server (die Kopie außer Haus nimmt nur die Nachtsicherung, `uploadBackupToS3` in
`jobs/backup.go`), 30 bis etwa 60 Tage liegen innerhalb der drei Monate oben, und bis die
wöchentlichen Stände gebaut sind, ist sie nach 14 Nächten der einzige Stand von vor dem Update
(`rotateBackups(backupDir, 14)`).
Dass das Löschen an einem Lauf hängt, gehört zu 5.31.

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

### 5.34 Die Suche im Medienkatalog nach einer Klasse trifft jede ISBN mit dieser Ziffer

Gefunden am 29.09.2026 bei der Durchsicht von PR #687, an `buecherSuchen`
(`frontend/src/inventur/lib/startseiten_api.js`, Medienkatalog → Suche & Filter) nachgestellt.
Jeder Suchbegriff wird auch als Teilstück der ISBN gesucht, eine einzelne Ziffer eingeschlossen.
„Klasse 7", „Jg. 8" und „9" treffen damit jedes Buch mit einer ISBN-13 (sie beginnen mit 978),
„Klasse 5" jedes, dessen ISBN eine 5 enthält — neben den Büchern des Jahrgangs. Gemessen an vier
Büchern (Jahrgang 5, Jahrgang 9, zwei ohne Jahrgang, drei mit ISBN): „Klasse 7" fand die drei mit
ISBN, richtig wären keine. Der Fehler ist sichtbar, die Liste ist zu lang. Vorschlag: Eine Zahl,
die als Jahrgang gelesen wird (höchstens zwei Ziffern), sucht nicht in der ISBN; die Suche nach
einer ganzen ISBN und ihren Schreibweisen (`isbnFormen`) bleibt. Der Test in
`startseiten_api.test.js` hält den Fehltreffer bewusst nicht fest.

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

### 6.2 Kategorie C

- Die Akte eines Kollegen ohne Ausweisnummer sagt am gesperrten Ausweisdruck „die Nummer steht
  in „Benutzer & Rechte""; ohne Konto hat er dort keinen Eintrag. Die Nummer kommt mit dem
  freigeschalteten Zugang (`StudentProfileActions.svelte`, `data-tip`).
- Browser-Gates: Die M3- und axe-Gates öffnen die Planer-Dialoge nicht, axe misst nur den
  Anfangszustand; kein Screenreader-Durchgang; der Ausweis-Designer geht nur per Maus.
- 16 Bestandsstellen bauen ihr Cover selbst (Liste in `frontend-hygiene-cover.test.js`, darunter
  `KlassenBuchKachel` im Portal). Umstellen beim fachlichen Anfassen, nicht in einem Rutsch.
- 3.000 Titel ohne ISBN: `inventur.SucheTextDNB` nur mit Bestätigung durch einen Menschen
  verdrahten.
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
- `github.com/jung-kurt/gofpdf` ist seit 2021 archiviert und steckt in 16 Dateien; gepflegt wird
  der Ableger `github.com/phpdave11/gofpdf`, den maroto mitbringt. Neue PDFs (5.3, 5.4) nicht
  mehr auf dem archivierten; die 16 beim fachlichen Anfassen umstellen, mit den PDF-Gates.
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
Personensatz).

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
**Die Sicherungen vom 01. und 02.09.2026** (nachgesehen am 28.09.2026): In `~/Downloads` und
auf dem Schreibtisch liegen seit dem 09.09.2026 zwei Littera-Sicherungen
(`littera_sicherung_01_09_2026_14_04_50.7z` und `littera_sicherung_02_09_2026_13_29_22.7z`,
856 KB und 602 KB). Jede enthält eine `.bak` (16,7 MB und 13,4 MB) und ist verschlüsselt.
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
neun Tabellen aus Abschnitt 1 von [SCRIPTS.md](SCRIPTS.md) als CSV ausgeben, Spaltennamen und
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
Speicher setzen, die der Aufbewahrung aus 5.30 folgt, oder die Rotation im Code auf den Speicher
ausdehnen. **Entschieden am 28.09.2026:** zuerst beim Schulträger fragen, ob er einen Speicher
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
deletions" anlassen. Am 23.09.2026 trägt das Ruleset noch `pull_request`; Pushes gehen über den
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

### 8.8 Die Abholfrist bei Vormerkungen

Ein vorgemerktes Buch liegt drei Tage bereit, gerechnet ab dem Zeitpunkt, zu dem es zugeteilt
wird: bei der Rückgabe (`INTERVAL '3 days'` in `internal/service/loan_return.go`) oder beim
Nachrücken, wenn der Vorige es nicht abgeholt hat (`repository/vormerkung_nachruecken.go`). Danach verfällt die Vormerkung beim nächsten
stündlichen Lauf, und das Buch geht an den Nächsten in der Warteschlange. Wochenende und Ferien
zählen mit: Ein Buch, das freitags um 10 Uhr zurückkommt, liegt bis Montag 10 Uhr bereit; kommt
es in den letzten drei Tagen vor den Herbstferien zurück, verfällt die Vormerkung in den Ferien.

Die Leihfrist verschiebt seit dem 24.09.2026 ein Ende an einem Wochenende, Feiertag oder in den
Ferien auf den nächsten Schultag (`Tagesfrist` in `internal/service/loan_rules.go`); die
Abholfrist nicht. Littera führt eine „Maximale Reservierungsdauer" in Tagen, die die Schule
einstellt (Stammdaten → Einstellungen → Verleih); Öffnungs- und Schließtage nennt das Handbuch
nur für die Leihfrist.

In der Sicherung von 2010 steht dort 0 (Tabelle `Einstellungen`, Spalte `MaxResDauer`,
nachgesehen am 28.09.2026).

**Entschieden am 28.09.2026: drei Tage, das Ende fällt wie bei der Leihfrist auf den nächsten
Schultag;** keine neue Einstellung. Folge: Kommt ein Buch kurz vor den Ferien zurück, liegt es bis
nach den Ferien bereit. Nicht gebaut: Die drei Tage stehen zweimal im Code (`INTERVAL '3 days'`
in `loan_return.go` und in `bedieneNaechstenWartenden`, `vormerkung_nachruecken.go`). Gesetzt wird
die Frist bei der Rückgabe und über `bedieneNaechstenWartenden` im Verfall-Lauf, beim Löschen
einer Vormerkung von Hand und bei der Spuren-Tilgung. `Tagesfrist` liegt in `internal/service`
und braucht die Sommerferien aus den Einstellungen; `repository` erreicht es nicht. Beim Bau die
Regel eine Schicht tiefer, mit Test über ein Wochenende, über Ferien und über den Verfall-Lauf.

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
- **Hosting- und Programmpflegekonzept.** Der Entwurf steht seit dem 24.09.2026, ergänzt am
  28.09.2026 um die Aufbewahrung der Sicherungen, den Datenweg beim Wechsel (4.22) und die
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
