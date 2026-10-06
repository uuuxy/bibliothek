# Offene Arbeit

Stand: 06.10.2026

**Der Fahrplan.** Oben steht, was als Nächstes getan wird, in der Reihenfolge der Arbeit: je
Schritt eine Zeile mit Kästchen. Die Nummer in Klammern führt zu den Einzelheiten weiter unten.
Einen zweiten Ort für Offenes gibt es nicht; andere Dokumente erklären (Konzept, Anleitung, der
Katalog der Bugklassen in [sweeps.md](sweeps.md)), führen aber keine eigene Offen-Liste. Wie die
Liste geführt wird, steht am Ende.

---

## Fahrplan

### Etappe 1: als Nächstes

- [ ] **Theke: zweiter Scan desselben Buchs.** Er leiht das eben zurückgegebene Buch nicht
  wieder aus, und eine Ausleihe meldet sich mit eigener Farbe und eigenem Ton. Zuerst das Bild,
  nach dem Ja der Bau. Kann still schaden: Das Buch steht im Regal und bleibt auf einem Konto.
  (4.32)
- [ ] **Standort: Ein neues Exemplar erbt ihn,** wenn alle übrigen Exemplare des Titels
  denselben tragen. Entschieden am 06.10.2026. (5.53)
- [ ] **Titel speichern:** Speichern zwei Plätze denselben Titel, gewinnt bei den Feldern des
  Titels ohne Meldung der zweite; die Eingabe des ersten ist weg. Die Maske schickt künftig nur
  die geänderten Felder. (5.5)
- [x] **PRs auf GitHub:** 21 durchgesehen am 06.10.2026. Zwei sind hereingeholt (701, 702), 19
  geschlossen; aus sieben davon sind die Tests übernommen (d1dba610).
- [x] **PR 722** durchgesehen und geschlossen am 06.10.2026: Die Prüfung beim Löschen einer
  Sachgruppe (`api/systematik_handler.go`) würde durch ihn höchstens 1 ms schneller.
- [ ] **Doku-Ordner:** Vorschlag, welche der 36 Dateien in `docs/` zusammengelegt oder gelöscht
  werden; 19 davon kamen seit September 2026 dazu.

### Etappe 2: Farben auf Material-3-Rollen (5.21)

114 Stellen tragen noch Palettenfarben (gezählt am 06.10.2026 mit dem Muster der Ratsche).
Bildschirm für Bildschirm, je Portion ein Commit, am gerenderten Bildschirm geprüft:

- [x] Buchakte mit der Liste der Ausleiher (20)
- [x] „Klassen & Bücher" (50)
- [x] Signaturen (24)
- [x] Gemeinsame Bauteile, erster Teil (43): Knopf, Schalter, Status-Chip, Dialog,
  Cover-Vorschau
- [x] Meldungen und Sprechblasen (10)
- [x] Dialog „Klassenversand" (16)
- [x] Suchpille mit dem Scanfeld der Theke (14)
- [x] Druck-Center (5). Quittung und nachgebildetes Etikett sind Papier und bleiben (21).
- [x] Monitor (25)
- [x] Berechtigungen (18)
- [x] System und Einzelstellen (54)
- [x] Dunkle Flächen (16): Sucher der Kamera, Leiste des Ausweisdrucks, die zwei Schleier
  hinter den Alarmen der Theke

Offen: 48 Farbverläufe der selbstgebauten Cover-Platzhalter (6.2), 16 Stellen der
Initialen-Kachel, 29 der Karte im Ausweis-Designer und 21 auf Papier.

### Etappe 3: vor dem Echtstart

Der Echtbetrieb beginnt am Schulserver mit einer leeren Datenbank und der Littera-Übernahme
(entschieden am 28.09.2026).

- [ ] Die Littera-Übernahme überträgt die Auflage. (5.5)
- [ ] Generalprobe der Übernahme mit der Sicherung von 2026, sobald sie sich öffnen lässt:
  Ausweisnummern, offene Ausleihen, Standorte, Verweise der Schlagworte. (7.2, 4.20)
- [ ] `update.sh` für den Schulserver: nur Releases, Images frisch. (5.31)
- [ ] Eingang des Servers: Von außen ist nur die Seite der Lieferanten erreichbar. (4.23)
- [ ] Arbeitsnotizen der Entwicklung ins Repository, entlang dem Pflegekonzept. (9.9)
- [ ] Am Schulserver einrichten: Sicherung außer Haus (7.3), Uptime-Signal (7.5),
  Eigentumsvermerk der Etiketten (4.24), Frist bis zum Sperrbildschirm (9.9). Danach nachsehen:
  Admin-Konten, Verbindung zur DNB (7.8).
- [ ] Abnahmen mit echten Daten. (7.7)
- [ ] Wiederherstellung an einem fremden Ziel proben, allein mit dem Pflegekonzept. (7.4)

### Bei dir

**Entscheiden:**

- [ ] Bekommt die Bestandsliste (CSV) eine Spalte „Standort"? Vorschlag: ja, zusammen mit dem
  Erben. (5.53)
- [ ] Die gemeinsamen Bauteile stehen auf M3-Rollen und sehen überall etwas anders aus:
  umrandete Knöpfe mit dunklerem Rand, „Löschen" rötlich getönt ohne Rand, Schalter im
  Aus-Zustand grauer, Dialoge mit rundem Schließen-Knopf. Vorschlag: bleibt so. (5.21)

**Am Testserver ausprobieren,** nach `git pull` und `./update.sh` (7.10):

- [ ] Portal: „Problem melden" unter dem Suchfeld
- [ ] Buchakte: „Standort ändern", auch mit dem Handscanner
- [ ] Theke: Schnellrückgabe mit einem Stapel
- [ ] Die übrigen Proben mit dem Handscanner
- [ ] Maske „Buch bearbeiten"
- [ ] Mahnwesen: Mahnbriefe und Liste drucken

**Erledigen:**

- [ ] SonarQube-Scan starten. (5.10)
- [ ] PR-Pflicht im Regelwerk für `main` entfernen. (7.6)
- [ ] Das Blatt mit den zwei Schlüsseln ausfüllen. (9.9)
- [ ] Theke ohne Netz: der Nachweis von Hand im echten Chrome, zurückgestellt am 24.09.2026.
  (2.3)

### Termine

- [ ] 19. Oktober 2026: CodeQL wechselt bei GitHub das Abbild; den Lauf danach ansehen. (5.10)
- [ ] Ab dem 28. Oktober 2026: Node 26, nach der Regel „immer die aktive LTS"
  ([PFLEGEKONZEPT.md](PFLEGEKONZEPT.md), Abschnitt 4).

### Später: kleine Fehler und Aufräumarbeit

Ohne feste Reihenfolge, gebündelt. Die Einzelheiten stehen unter der Nummer; was erledigt ist,
wird dort gelöscht.

- **Jahrgang am Titel (5.5):** „Klasse" und „von … bis" werden eine Angabe, „unbekannt" eine
  eigene (entschieden am 24.09.2026). Davor: je Titel festlegen, welche Spanne gilt.
- **Buchakte (5.5):** „Exemplar löschen" an einem bestellten Exemplar schreibt einen Abgang ohne
  Zugang; ausgesonderte und bestellte Exemplare heißen dort „Gesperrt".
- **Druck-Center (5.5, 5.45):** Ein Ladefehler steht als „kein Exemplar" da, die Vorschau zeigt
  immer denselben Bogen, und bei vielen Exemplaren wird die Seite sehr lang.
- **Überläufe (5.45):** Bestellwesen 16 px, Signaturen bei 1280 px, ein langer Name in der
  Leserakte.
- **Auskunft (5.19):** Zwei Einträge über die Anlage eines Kontos fehlen, sobald das Konto
  gelöscht ist; zu klären, welche Rohdaten aufs Blatt gehören.
- **Protokoll und Tilgung (5.35):** Einträge, die einen Leser nur über seine Forderung meinen;
  die Frist für die Löschspur von Forderung und Vormerkung; am Testserver alte Einträge zu schon
  gelöschten Lesern.
- **Gates und Werkzeuge (5.10):** Lücken in Prüfregeln und Tests, der Wechsel auf Ubuntu 26.
- **Zwei Helfer (5.5):** Beträge und Fehlertexte schreiben 27 Stellen selbst.

### Nur mit Anlass: kein Schritt

Bekannt und beschrieben. Gebaut wird erst, wenn der dort genannte Anlass eintritt.

- 5.5 Die kurze Nummer der Littera-Etiketten an der Theke: entscheidet sich mit der Generalprobe.
- 5.5 Listenimport: Trägt eine Zeile die Ausweisnummer eines Lesers als Buchnummer, nennt die
  Meldung weder Zeile noch Weg.
- 5.25 Eine Forderung für ein Gerät lässt sich nicht anlegen.
- 5.45 Bedienung am Tablet und in Fenstern unter 1280 px.
- 5.49 Versetzung, wenn eine Klasse mit einer Zahl beginnt, die kein Jahrgang ist.
- 5.54 Klassensätze aus den Ausleihen für ET1 bis ET3, weitere Hinweise an der Theke.
- Abschnitt 6: Beobachtungen und Kategorie C.

---

## Einzelheiten

Zum Nachschlagen: Beleg, Messwert und nächster Schritt je Punkt. Die Nummern bleiben fest und
werden nicht neu vergeben; Kommentare im Code nennen sie als Herkunft.

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

**Vor dem Bau muss feststehen:** der Name, unter dem der Server im Internet erreichbar ist, und
dass die Geräte der Schule ihn unter demselben Namen erreichen, denn das Zertifikat gilt für den
Namen; die Freigabe von Port 443 von außen auf den Server und der Netzabschnitt, in dem er dann
steht; und mit welcher Absenderadresse Anfragen aus dem Schulnetz am Server ankommen. Daran
erkennt der Eingang das Schulnetz. Kommen Anfragen von außen und aus der Schule mit derselben
Adresse an, kann er sie nicht unterscheiden.

Auch im Schulnetz braucht der Server Verbindungen nach außen: den Mailserver der Schule (die
Anmeldung läuft über das Postfach, `auth/imap.go`), DNB, Google Books und OpenLibrary für
Titeldaten und Cover (`pkg/coverquelle`), für Updates GitHub, Docker Hub und die Paketquellen
(`Dockerfile`, `update.sh`).

### 4.24 Eigentum je Exemplar

Die Übernahme ordnet den Littera-Eigentumsvermerk je Exemplar nach einer festen Liste zu
(`vermerkeLittera` in `internal/littera/eigentum.go`): „Land Hessen" wird Land, auch an den
11.160 Exemplaren ohne LMF-Signatur (nach dem Leitfaden „Lernmittelfreiheit in Hessen", Ziffern
2.1 und 9.4.2; Begründung in 35ae1cf9), „Hochtaunuskreis" wird Schulträger. Nach welcher Regel
Etikett, Bestandsbücher und Schadensersatz das Eigentum lesen, steht in
[FACHKONZEPT.md](FACHKONZEPT.md).

**Ohne Zuordnung** sind die Exemplare mit den fünf seltenen Vermerken (Schule 355, Bibliothek
157, Förderverein 86, Info Schulprojekt 31, Dauerleihgabe 4). Sie kommen nur als Wortlaut mit,
und es gilt die Faustregel. Die Zuordnung steht an einer Stelle (`vermerkeLittera`); nach der
Übernahme lassen sich einzelne Titel auch in der Buchakte setzen (_Eigentum ändern_).
**Entschieden am 01.10.2026:** kein dritter Eigentümer im Programm. Bücher Dritter behalten den
Littera-Wortlaut, den die Buchakte schon zeigt; Ersatz für ein verlorenes Buch liefe dann über
den Schulträger, bei 90 Büchern ein seltener Fall.

**Eigentumsvermerk auf neuen Etiketten** (entschieden am 28.09.2026): Neue Etiketten der
Schülerbücherei tragen den Wortlaut der alten Littera-Etiketten. Er wird beim Einrichten unter
Einstellungen → Schule eingetragen; das Feld ist heute leer, neue Etiketten tragen also keinen
Vermerk.

### 4.32 Theke: ein Stapel vom Rückgabetisch

Seit dem 06.10.2026 gibt es die Schnellrückgabe wie in Littera: ein Knopf neben dem Scanfeld.
Solange sie an ist, nimmt jeder Scan nur zurück, kein Leser wird geladen, und die Meldung nennt,
bei wem das Buch war. Sie endet mit einem zweiten Klick, mit Escape, mit einem gescannten
Ausweis oder gewählten Leser und mit dem Leeren der Theke (Vorgabe fünf Minuten ohne
Bedienung, Abmelden, Neuladen).

Offen bleibt der gewöhnliche Betrieb. Am Code gelesen und am 06.10.2026 am lokalen Stack
nachgestellt, jeder Schritt an der Tabelle der Ausleihen belegt:

- Eine Rückgabe ohne geladenen Leser lädt den Leser des Buchs (`verarbeiteRueckgabe` in
  `stores/omnibox.svelte.js`). Wer einen Stapel ohne Schnellrückgabe scannt oder nach dem
  Leeren der Theke weiterscannt, hat ab dem zweiten Buch einen Leser geladen.
- Ein freies Exemplar wird an den geladenen Leser ausgeliehen (`HandleUnifiedCheckout` in
  `internal/service/loan_checkout.go`). Das trifft ein Buch, das nicht verliehen war, und den
  zweiten Scan desselben Buchs: Der erste gibt zurück, der zweite leiht wieder aus.
- Ausleihe und Rückgabe melden sich gleich, grün und mit demselben Ton; nur der Text der
  Meldung unterscheidet sie (`verarbeiteRueckgabe`, `verarbeiteAusleihe`). Ein Buch eines
  anderen Lesers meldet sich orange mit Warnton.

Folge: ein Buch im Regal, das auf dem Konto eines Kindes steht und später gemahnt wird.
Kategorie A.

**Richtung vom 06.10.2026:** eine Sperre für dasselbe Buch in den ersten Sekunden nach seiner
Rückgabe und ein eigener Ton und eine eigene Farbe für die Ausleihe. Verworfen ist der Weg,
einem Leser, der nur durch eine Rückgabe erscheint, nichts auszuleihen: An der Theke wird oft
ein Buch des Kindes gescannt, damit sein Konto erscheint, und danach ausgeliehen. Die Farbe ist
eine sichtbare Änderung: vor dem Bau ein Vorschlag nach den M3-Seiten. Der Bau braucht die
Proben des Scanner-Pfads; `frontend/e2e/theke-schnellrueckgabe.spec.js` hält in seiner
Gegenprobe den heutigen Stand fest.

---

## 5. Abarbeitbar (Kategorie B)

### 5.5 Bestand, Katalog, Druck

- Listenimport gegen den Nummern-Wächter (Migration 131): Trägt eine Zeile der Datei die
  Ausweisnummer eines Lesers als Buch-Barcode, lehnt der Wächter ab und der ganze Import
  bricht mit der rohen Datenbankmeldung ab (`ON CONFLICT DO NOTHING` fängt nur den Index,
  nicht die Ausnahme). Laut, also richtig — nur die Meldung nennt weder Zeile noch Weg.
  Kategorie C, bis es einmal vorkommt.
- „Klasse" neben der Spanne: Der Titel führt zwei Jahrgangsangaben, „Klasse" (`grade_level`)
  und „von … bis" (`jahrgang_von/bis`). Inventur nach Klasse und die Mehrjahresband-Frist lesen
  nur die Spanne; Titel-Tabelle, Klassenzuweisung und Listenfilter lesen die Klasse;
  Portal-Filter und die Suche im Medienkatalog (`trifftJahrgang` in
  `frontend/src/inventur/lib/startseiten_api.js`) lesen beide. Die Zusammenlegung (Migration
  135) ist zurückgenommen: Aus Klasse N wurde die Spanne N bis N, und das trifft die Daten
  nicht. Gemessen am Testserver: 153 Titel mit Klasse, 129 davon Klasse 6–13 bei der Vorgabe 5
  bis 10, 89 dieser 129 Lernmittel; 13.057 von 13.062 Titeln tragen die Vorgabe (01.10.2026).
  Mehrjährige Bände tragen ein einziges Jahr („Natur und Technik - Biologie 7 - 10" und
  „Pontes Gesamtband": Klasse 7), Klasse und Signatur widersprechen sich („Forum Geschichte
  4 (Schulbuch Klasse 9)": Signatur Ges9, Klasse 10). Woher die Werte stammen, ist nicht
  belegt; eine Klasse 5 aus einem älteren Listenimport ist von einer gepflegten nicht zu
  unterscheiden. Katalogsuche und Schulbuchliste (`jahrgangText`) werten die Vorgabe 5 bis 10
  nicht als Jahrgang; eine bewusst gepflegte Spanne 5 bis 10 fällt damit bis zum Umbau
  ebenfalls heraus. Nächster Schritt: je Titel entscheiden, welche Spanne gilt (Liste per
  Einzeiler unten), dann die Spalte mit genau diesen Werten ablösen. Dabei mitentscheiden: die
  Spalte „klasse" des Listenimports und der Klassenvorschlag der ISBN-Suche, der auch aus
  „Band 2", „Level 9" und jeder Zahl von 5 bis 13 im Titel eine Klasse macht.
  **Entschieden am 24.09.2026, im selben Umbau:** „Jahrgang unbekannt" wird eine eigene Vorgabe
  (NULL) statt 5 bis 10 — heute ist beides nicht zu unterscheiden, und wer das Mehrjahresband
  (Migration 134) an einem Titel mit der Vorgabe anhakt, bekommt die 10. Die Leser der Spanne
  (Inventur, Portal-Filter, Katalogsuche) lernen „unbekannt" mit. Vorher am Testserver messen.

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
  Wert aus dem Ladezeitpunkt"). Die Eingabe des ersten Platzes geht ohne Meldung verloren;
  der Punkt steht deshalb im Fahrplan in Etappe 1.
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
  Schlüssel steht in `sonar-project.properties`. Letzter Scan am 04.10.2026 (MQR-Modus):
  Zuverlässigkeit 0 Meldungen, Sicherheit 0, Wartbarkeit 114, Abdeckung 77,4 %, Quality Gate OK.
  Die 114 sind am 05.10.2026 bearbeitet: 101 im Code, 13 als begründete Ausnahme in
  `sonar-project.properties` (e12 bis e21). **Offen: der Scan nach diesen Commits;** bis dahin
  zeigt der Server weiter 114. Nicht vorab messbar waren die zwei Meldungen zu `S6594` in
  `frontend/scripts/druck-sektionen-gate.mjs` und die Wirkung der Ausnahmen. Der Stand vom
  03.10.2026 liegt auf dem Server unter `Bibliothek4a`; wie die Abdeckung gemessen wird, steht
  in [SCRIPTS.md](SCRIPTS.md), „Warum die Coverage niedriger aussieht, als sie ist".
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
- **Ein Browser-Test war dreimal rot.** `e2e/feld-roundtrip.spec.js` („Buch
  anlegen: Bestand und Zähldatum kommen in der DB an") fand am 05.10.2026 in zwei vollen
  Läufen und am 06.10.2026 in einem Lauf über neun Dateien am lokalen Stack den neuen Titel
  nicht binnen 10 s; einzeln lief die Datei danach jedes Mal grün. Beim zweiten Mal war die Maske zu und die Liste stand da („Bücher (10618)"),
  der neue Titel fehlte in der Ansicht. Die lokale Datenbank trägt Hunderte Test-Titel aus
  früheren Läufen.

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

### 5.21 Palettenfarben auf M3-Rollen

Stand 06.10.2026: 114 Fundstellen mit Tailwind-Palettenfarben (`slate`, `blue`, `emerald` …),
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
Cover-Platzhalter; das gehört zu 6.2 (Cover über `ui/BuchCover`). `avatarKachel.js` (16) sind die
Verläufe der Initialen-Kachel in der Leserakte, je Name eine Farbe; sie bleiben.

Im Ausweis-Designer bleiben 29 Stellen: die Farben der Karte (`themes` in
`designer/Toolbar.svelte`, die Vorgaben in `idDesignerStore.svelte.js`) und die Platzhalter
auf der gezeichneten Karte (`CanvasElement.svelte`, `CardFace.svelte`). Die Farbe der Karte
steht als Klassenliste im zentral gespeicherten Entwurf und wird gedruckt; die Karte bleibt
weiß, auch wenn die Oberfläche ihr Farbschema wechselt.

Papier bleibt, 21 Stellen: Die Quittung (`StudentPrintReceipt`, 12) wird nur gedruckt, und
`labels/LabelPreview` (9) bildet Bogen und Etikett nach. Wie bei der Karte gilt: Papier bleibt
weiß, auch wenn die Oberfläche ihr Farbschema wechselt. Die übrigen Druckblätter tragen feste
Farbwerte im eigenen Stylesheet (`utils/listenDruck.js`).

Dunkle Bereiche (Flur-Monitor, Sucher der Kamera, Leiste des Ausweisdrucks) stehen im dunklen
Schema: Die Klasse `schema-dunkel` in `rollen.css` gibt den Rollen ihre dunklen Töne.

Beim Umstellen aufgefallen, jeweils am Code nachgesehen:

- Buchakte, Liste der Ausleiher: Eine überfällige Ausleihe ist nur an der Farbe des Datums zu
  erkennen (`BorrowersListe.svelte`). Die Leserakte setzt für dieselbe Ausleihe ein Zeichen und
  für Screenreader das Wort „Überfällig" dazu (`AusleiheRueckgabe.svelte`).
- Buchakte, Exemplarkarte (`BookExemplarCard.svelte`): „Barcode scannen" ist 24 px hoch
  (gemessen am 06.10.2026). Stift, Drucker und Papierkorb sind 14 px große Symbole ohne
  Knopffläche (`.icon-btn`), drei davon erklären sich über `title` statt `data-tip` (am Code
  gelesen). „Interne ID generieren" bricht bei 1280 px Fensterbreite im Knopf in zwei Zeilen um.
- Einstellungen, „E-Mail Routing für Mahnungen": Die Oberfläche sagt „Mapping" (leere Liste,
  Meldung nach dem Löschen, Sprechblase am Papierkorb; `SystemSettingsRouting.svelte`); ein
  deutsches Wort wäre „Zuordnung".
- „Rollen & Rechte": Bei 1280 px Fensterbreite bleiben der Erklärung eines Rechts 152 px
  neben den vier Schaltern; die Sätze laufen über bis zu sieben Zeilen (`PermissionsEditor.svelte`,
  gemessen am 06.10.2026).
- Signaturen: Welche Signatur in der linken Liste gewählt ist, sagt nur die Farbe der Zeile
  (`SignaturenView.svelte`); für Screenreader trägt die gewählte Zeile kein Merkmal.
- Klassensätze: Der Knopf „Bücher verwalten" heißt für Screenreader „Klasse bearbeiten"
  (`KlassenKarte.svelte`, `aria-label`); der Name enthält das sichtbare Wort nicht.
- Dialog „Klasse & Bücher zuweisen": Die Kacheln des Büchergitters tragen 28 px Rundung, die
  Stufe der Dialoge (Karten: 12 px), und vergrößern sich beim Wählen; die Zeile „BÜCHER FINDEN"
  steht in Versalien (`ClassAssignmentBookGrid.svelte`, gemessen am 06.10.2026).
- Titel-Verwaltung: Ein Titel lässt sich in der Liste nur mit der Maus öffnen. Der Klick hängt
  an der Zeile (`BookTableZeile.svelte`, `onclick` am `<tr>`), die Zeile nimmt keinen Fokus. Die
  Leserdatei öffnet die Akte über den Namen als Knopf.
- Titel-Verwaltung: Der Knopf „Retry Cover" trägt eine englische Beschriftung.
- „LUSD & Versetzung": Die Flächen für Fehler, Erfolg, Hinweis und Warnung stehen in
  `LusdImportView` und `PromoteStudentsView` von Hand, wie an rund 40 weiteren Stellen der
  Anwendung (gezählt am 02.10.2026: getönte Fläche und Rundung in einer Klassenliste); ein
  gemeinsames Bauteil dafür gibt es nicht.
- Statistik: Die Balkenfarben des Diagramms (`StatsTrendChart`) sind feste Werte, keine Rollen.
- Benutzerliste: Der Zustand eines Kontos steht in zwei Formen, „Aktiv" und „Inaktiv" als Punkt
  mit Wort, „Zugang beantragt" als Pille (`UserManagementTable`).
- Bestellhistorie: Die Zelle „Lieferant" trägt `max-w-0` ohne volle Breite an der Spalte
  (`BestellHistorieTabelle.svelte`). Dieselbe Form ließ im Fehlbestandsbericht der Inventur dem
  Titel 191 von 918 px (gemessen und behoben am 05.10.2026). Name und Kundennummer tragen
  `truncate` und keine Sprechblase (M3, Text truncation: „Don't truncate content without
  providing users another way to see it"). Am Code gelesen, im Browser nicht gemessen.
- Inventur: Die Wörter „Inventur-Scope" (Überschrift des Start-Dialogs) und „aus dem aktuellen
  Scope" (Rückfrage vor dem Abschluss) stehen so in der Oberfläche; ein deutsches Wort wäre
  „Umfang" oder „Bereich".
- Inventur: Ein unbekannter Barcode zeigt am Scanner den rohen Fehlertext „exemplar für
  inventur-scan nicht ladbar: no rows in result set" (`GetExemplarForInventoryScan` hüllt
  `pgx.ErrNoRows` ein, `ladeExemplarFuerScan` gibt ihn mit 404 unverändert weiter). Der Status
  stimmt, der Satz nicht.
- Leserdatei: Die Leiste des Ausweisdrucks (`students/AuswahlAktionsleiste`) ist eine eigene,
  dunkle Leiste neben `ui/AuswahlLeiste` (Schlagwort-Pflege). Vor dem Zusammenlegen zu klären:
  wohin der Hinweis „ohne Ablaufjahr" und das Feld „Ab Feld" kommen — beides passt nicht in
  die 64 px hohe Leiste.
- Flur-Monitor: Das Symbol der Kennzeile („Buch des Monats", „Neu eingetroffen", „Beliebt
  diese Woche") steht über dem Wort statt daneben (`monitor/Folie*.svelte`; das Symbol ist ein
  Blockelement in einem `<span>`). Gesehen am 06.10.2026.
- Bestellwesen, nach der Umstellung gegen die M3-Seiten gehalten (05.10.2026): Die Zahl im
  eingeklappten Bestellstreifen steht von Hand auf `primary` mit 24 px
  (`BestellWorkspace.svelte`; M3, Badges: Farbe „Error", 16dp; dafür gibt es
  `ui/Zaehlerpille`). „PDF-Bestellliste" ist ein Link in eigener Knopfform: 12 px Rundung, Fläche
  und Rand zugleich, 34 px hoch (`OrderRecommendations.svelte`; `ui/Button` kann kein Link
  sein). „Speichern", „Abbrechen", „Bearbeiten" und „Löschen" der Lieferanten sind Wörter ohne
  Knopffläche (`LieferantZeile.svelte`). Zwei Hinweissätze unter „Bestellung auslösen" stehen
  in `text-label-small`, der leere Warenkorb in einem gestrichelten Kasten (`OrderCart.svelte`).
  Der Spaltenkopf der Bedarfsliste steht von Hand in Versalien neben dem Kopf von `ui/Tabelle`.
  „Lade …" steht an drei Stellen als pulsierender Text (`BestellDetail`, `BestellHistorie`,
  `KlassensatzReservierungen`), sonst `ui/Ladekreis`. `WareneingangView.svelte` trägt `bg-white`;
  Weiß und Schwarz zählt die Ratsche nicht (74 Stellen in `.svelte`-Dateien).
- Ausweis-Designer: Die Knöpfe der Textausrichtung tragen englische Hinweise („left",
  „center", „right"; `PropertiesText.svelte`). Die zwei Umschalter der Werkzeugleiste
  (`ToolbarDruck.svelte`, `Toolbar.svelte`) stehen von Hand in gleicher Form, nicht aus
  `ui/Segmente`. Die zwei Farbwähler der Eigenschaften sind 32 und 36 px hoch.

### 5.25 Eine Forderung für ein Gerät lässt sich nicht anlegen

Die Datenbank sieht sie vor (`check_damage_item`: genau eines von `exemplar_id` und
`geraet_id`), die Rechnung an die Eltern kann sie drucken (`queryRechnungItems`), aber der
einzige Schreiber `meldeSchaden` (`repository/schaden_melden.go`) nimmt nur ein Buch-Exemplar:
Er sondert das Exemplar aus und legt die Forderung mit `exemplar_id` an. Fehlt bei der Rückgabe
Zubehör oder ist ein Gerät kaputt, gibt es keinen Weg zur Forderung; das FACHKONZEPT (Abschnitt
5) behauptete bis zum 24.09.2026 einen. Gesperrt würde wie heute (Schülerbücherei und
Geräte).

Beim Bau mitnehmen (bis zum 01.10.2026 als 5.37 geführt, am Code gelesen): Der Bescheid-Dialog
listet auch eine Forderung ohne Exemplar (`topf` leer). Sie steht richtig gesperrt da, darunter
aber der Satz „Buch der Schülerbücherei — gehört nicht auf den Bescheid des Landes."
(`frontend/src/lib/components/mahnwesen/BescheidPositionen.svelte`). Die Zeile darüber zeigt
„ohne ISBN" und, weil ein Gerät keinen Buchpreis hat, „kein Preis hinterlegt — Betrag bitte
eintragen" neben dem gesperrten Feld. Heute nicht zu sehen: Ohne Schreiber gibt es keine
Forderung für ein Gerät.

### 5.31 `update.sh` für den Schulserver: nur Releases, Images frisch

**Entschieden am 28.09.2026, nicht gebaut.** Vor dem Echtstart; gebaut wird, sobald der
Schulserver feststeht.

- **Nur Releases:** Der Schulserver bekommt nur Releases, der Testserver folgt `main` als
  Vorstufe. Heute holt `update.sh` mit `git pull` den neuesten Stand des Zweigs und fragt nicht
  ab, ob dessen Prüfläufe grün sind. Ein Release entsteht nur, wenn alle Pflicht-Prüfungen des
  Commits grün sind (`scripts/tag-gate.sh`).
- **Die Postgres-Nebenversion am Server:** `update.sh` ruft `docker compose up -d --build` auf
  und holt das Image `postgres:18-alpine` nie neu; der Datenbank-Container bleibt auf der
  Nebenversion des Images, das beim Anlegen vorlag. **Entschieden: bei jedem Update holen;** die
  Datenbank startet dann bei einer neuen Nebenversion während des Updates neu. Eine
  Nebenversion braucht weder Sicherung noch Neuaufbau, und die Datenbank sortiert unter musl
  ohne Sprachregeln (`datlocprovider` = c), ein neues Alpine im Image ändert die Reihenfolge der
  Indizes also nicht. Die Hauptversion bleibt im Repo festgeschrieben.
- **Die Alpine-Pakete im Backend-Image:** Das `Dockerfile` holt Sicherheitskorrekturen nur über
  `apk --no-cache upgrade`. Der Build-Cache hält diese Schicht fest, solange die Zeilen davor
  gleich bleiben, und `update.sh` baut ohne `--pull` und ohne `--no-cache`. Der Trivy-Scan der
  CI prüft ein frisch gebautes Image, nicht das am Server. **Entschieden: `--pull --no-cache`;**
  jedes Update baut dann alles neu (in der CI 93 Sekunden).
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

- **Druck-Center, Schritt 2** (`LabelBarcodeSchritt.svelte`, `max-h-40`): Der Kasten bleibt
  als Auswahlliste; darüber stehen ein Kästchen für alle und ein Feld für die Nummer. Offen:
  Bei einem Titel mit 409 Exemplaren ist die Seite durch die Vorschau 13.462 px hoch (gemessen
  am 02.10.2026), und „A4-Bogen drucken" steht unter beiden Spalten (`LabelPrinter.svelte`).
  Jede Zeile nennt „(Neuwertig)", wenn das Exemplar keine Zustandsnotiz trägt, auch ein
  bestelltes.
- **Bestellwesen:** Die Seite läuft 16 px über (852 von 868 px, gemessen am 02.10.2026).
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
  1118 px nichts. Bei 1024 bis rund 1035 px ist auch die
  Reiterzeile der Akte 12 px zu schmal, sobald „Gebühren & Schäden" eine Zahl trägt (gemessen am
  05.10.2026: 540 von 528 px); sie lässt sich dann seitlich schieben.
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
  (49 px), in „Ausleihen & Vormerkungen" endet er an der Oberkante der Überschrift (gemessen am
  03.10.2026 bei 1280 × 900 px an einem Testleser). Namen mit Leerzeichen oder Bindestrich
  brechen um.
- **Leserakte, Autor und Nummer des Exemplars:** Der Autor steht nur in der Sprechblase am
  Titel, die Nummer in Fenstern bis rund 1580 px ebenfalls (darüber hat sie ihre Spalte;
  gemessen bei ausgeklappter Seitenleiste). Die Sprechblase erscheint beim Zeigen mit der
  Maus; an einem Tablet ohne Maus sind beide Angaben in der Akte nicht zu sehen.
  Vorleseprogramme bekommen sie als unsichtbaren Text.
- **Leserakte, doppelte Beschriftung:** Unter dem Reiter „Stammdaten & Adresse" steht dieselbe
  Überschrift noch einmal; im Reiter „Gebühren & Schäden" heißt die Liste seit dem 01.10.2026
  „Forderungen".

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

### 5.53 Standort am Exemplar: was offen ist

Der Standort steht am Exemplar und wird in der Buchakte über „Standort ändern" gesetzt; das
Feld am Titel gibt es nicht mehr. Offen sind zwei Punkte:

- **Neue Exemplare.** Entschieden am 06.10.2026: Ein neues Exemplar erbt den Standort, wenn
  alle übrigen Exemplare des Titels im Bestand denselben tragen. Nicht gebaut: Heute kommt es
  ohne Standort an, zu sehen an der Zahl hinter dem Standort („Regal 11 (30)" bei einem Bestand
  von 35). Ein Exemplar kommt auf drei Wegen zu einem vorhandenen Titel: Bestand in der
  Titelmaske erhöht (`gleicheExemplareAn` in `inventur/db_books_update.go`), Bestellung
  (`BulkInsertCopiesTx` in `repository/book_inventory.go`) und Listenimport
  (`internal/service/import_dynamic.go`). Die Regel gehört an eine Stelle.
- **Bestandsliste.** Die Bestandsliste als CSV ([PFLEGEKONZEPT.md](PFLEGEKONZEPT.md),
  Abschnitt 8) nennt je Exemplar die Signatur, den Standort nicht. Nächster Schritt: eine
  Spalte „Standort" in `inventur/export_csv.go` und im Pflegekonzept.

Unberührt bleibt: Ein Titel ist Bibliothek oder Lernmittel, mit allen Exemplaren. Die
Exemplare eines Schulbuchs, die in der Bücherei stehen, werden wie Lernmittel verliehen.

Am Testserver sind die Standorte aus Littera erst mit einer Übernahme aus der
Littera-Datenbank zu sehen (7.2). Kategorie B.

### 5.54 Klassensätze aus den Ausleihen und Hinweise an der Theke

Am 03.10.2026 auf eine Frage hin am Code gelesen. Ein Auftrag dazu liegt nicht vor; gebaut
wird nur mit Anlass.

- Für ET1 bis ET3 entsteht kein Klassensatz „aus Ausleihen": Die Übersicht zählt nur Klassen,
  deren Name mit einer Ziffer beginnt (`GetClassGroups` in `inventur/datenbank_klassen.go`).
  Gezählt werden Ausleihen auf die Ausweise der Kinder; ein Stapel auf dem Ausweis der Lehrkraft
  erscheint nicht bei der Klasse.
- Die Theke warnt, wenn ein Kind eine andere Auflage bekommt als seine Klasse, und wenn ein Buch
  auf ein anderes Kind verbucht ist. Sie warnt nicht, wenn ein Kind ein Buch bekommt, das nicht
  zu seinem Jahrgang gehört, oder ein zweites Exemplar eines Titels, den es schon hat (am Code
  gelesen am 03.10.2026, nicht nachgestellt). Die Kachel eines Klassensatzes nennt die Zahl der
  Leser, nicht, wem das Buch fehlt.

Kategorie B.

---

## 6. Beobachten und Kategorie C (nur mit Anlass)

### 6.1 Beobachtungen

- Der Medienkatalog lädt in beiden Reitern die ganze Titelliste (`GET /api/books`, ohne
  Grenze), „Suche & Filter" bei jedem Öffnen, die Titel-Verwaltung beim Öffnen und bei leerem
  Suchfeld; gezeigt werden je 50 Titel. Gemessen am 04.10.2026 am lokalen Stack: 9.738 Titel,
  4,87 MB, gepackt über die Leitung 0,43 MB (`api/middleware_kompression.go`); auch über den
  Proxy des Testservers kommen die Antworten gepackt an. Größe und Dauer am Server über das
  Schulnetz sind nicht gemessen (im Browser: F12, Netzwerk, Zeile `books`). Anlass zum Bauen:
  Der Katalog öffnet am Server spürbar verzögert. Dann beantwortet der Server einen
  unveränderten Bestand mit 304 statt mit der Liste, wie bei den Buchnummern der Theke
  (`api/buchbarcodes_handler.go`).
- Breite der Textfelder: Textfelder folgen Material 3 (Text fields, Guidelines: „Text fields
  shouldn’t span the full width of a large screen"; entschieden am 03.10.2026). Umgesetzt ist das
  in der Maske „Buch bearbeiten" (Felder bis 704 px); die übrigen Masken sind nicht
  durchgesehen. Der Wert steht bisher nur in `BuchFormular.svelte`; mit der zweiten Maske gehört
  er an eine Stelle.
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
- Die Leiste der markierten Exemplare in der Buchakte trägt mit „Standort ändern" vier Knöpfe
  und ist 692 px breit. Sie passt bis zu einem Fenster von 768 px. Bei 640 px ragt das × um
  10 px über die Leiste, bei 390 px sind „Eigentum ändern", „Löschen" und das × abgeschnitten
  (gemessen am 06.10.2026 im Browser). Mit drei Knöpfen war sie 553 px breit (gerechnet aus den
  Knopfbreiten) und bei 390 px schon abgeschnitten. Material 3, Toolbars: „If there's not
  enough space for all items, put them in an overflow menu in the trailing slot";
  `ui/AuswahlLeiste` hat kein solches Menü. Anlass zum Bauen: Die Buchakte soll am Telefon oder
  an einem kleinen Tablet bedient werden.
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
- Die Rate-Limiter-Maps räumen erst ab 5.000 Einträgen; nur mit vielen frischen Adressen ein
  CPU-Thema.
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
  die Angaben des zuletzt passenden Eintrags darüber (`queueTitelUpsert` in
  `BulkUpsertBookTitles` sucht über die ISBN, sonst über den Titeltext). Gemessen am 30.09.2026
  am Export vom Juni 2026: Aus 13.708 Einträgen werden 11.302 Titel (226 echte Dubletten, 432
  mit gleicher ISBN und anderem Text, 1.748 mit gleichem Text bei anderer oder fehlender ISBN) —
  „Harry Potter und der Feuerkelch" steht als Buch, Taschenbuch und DVD in der Datei und wird
  ein Titel. Ein zweiter Lauf ändert 789 Titel; der letzte passende Eintrag gewinnt. Der Import
  aus CSV und Excel (`internal/service/import_dynamic.go`) gleicht ebenso ab, dort nicht
  gemessen; die Übernahme aus der Sicherung (`internal/littera`), mit der der Echtbetrieb
  beginnt, tut es nicht. Der Katalog am Testserver stammt aus diesem Import. Anlass zum Bauen:
  Das Katalogisat wird wieder ein Weg in den Echtbetrieb, oder ein gepflegter Katalog soll es
  erneut einlesen.
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
- Das Anfrage-Log nennt weder Dauer noch Anfragekennung (aus der Durchsicht von PR 631 am
  21.09.2026): nicht gebaut, die Doku steht auf dem Ist-Stand. Mehr Logzeilen am Schulserver
  sind eine Betriebsfrage.
- Barrierefreiheit: Ob das System eine Erklärung zur Barrierefreiheit und barrierefreie PDFs
  braucht (HTML-Druckweg oder begründete Ausnahme), ist nicht geklärt; bis dahin geparkt. Was
  die Gates heute prüfen, steht in [FACHKONZEPT.md](FACHKONZEPT.md), Abschnitt 19.

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
- Mahnwesen: „Neu laden" ist ein Symbolknopf ohne Wort (`MahnwesenAktionen.svelte`); auf den
  anderen Seiten heißt der Knopf „Aktualisieren" oder „Neu prüfen".
- `ui/Menue` kann einen Kopf über den Einträgen und Gruppen-Überschriften (`kopf`,
  `ueberschriftDavor`); seit dem 05.10.2026 nutzt beides kein Aufrufer mehr.
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
- `github.com/jung-kurt/gofpdf` ist seit 2021 archiviert und steckt in 17 Dateien (ohne Tests,
  gezählt am 30.09.2026); gepflegt wird der Ableger `github.com/phpdave11/gofpdf`, den maroto
  mitbringt. Neue PDFs nicht mehr auf dem archivierten; die 17 beim fachlichen Anfassen
  umstellen, mit den PDF-Gates.
- Reste des Nie-verdrahtet-Sweeps: `abgaenger_jahr` in der Aktivlisten-Antwort ohne
  Konsument; bei den Geräten `ActionEvent.GeraetID` ohne Broadcast und mit Null-Zeitstempel.
- Die Prüfung der UUID-Pfadparameter (`ValidateUUIDParamsMiddleware`) sitzt in
  `RequirePermission`. Eine Route mit `{id}`, `{schueler_id}` oder `{ausleihe_id}` unter
  `RequireAuthenticated` liefe an ihr vorbei, und die Kennung ginge ungeprüft an die Datenbank.
  Heute gibt es keine: Am 04.10.2026 antworteten alle 65 Routen mit UUID-Platzhalter am echten
  Router auf eine Kennung, die keine UUID ist, mit 400. Kein Gate hält das;
  `api/uuid_pfadparameter_test.go` prüft die Namen der Platzhalter, nicht die Hülle der Route.
  Anlass zum Bauen: die erste Route mit UUID-Platzhalter ohne `RequirePermission`.
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
- `docs/datenschutz_offene_punkte.md` heißt wie eine zweite Offen-Liste: Teil A ist dort
  abgehakt statt gelöscht. Vorgeschlagen am 02.10.2026 und nicht
  entschieden: umbenennen und die Erledigt-Spalte streichen; 17 Dateien verweisen auf den Namen.
- Vorschläge an einem Textfeld zeichnet der Browser (`datalist`): am Feld „Signatur", im
  Dialog „Standort ändern", in der Schlagwort-Pflege und in `ui/ChipFeld` (sechs Stellen,
  gezählt am 06.10.2026). Die Liste sieht je Browser anders aus und folgt nicht Material 3;
  im Haus ist sie einheitlich. Anlass zum Bauen: Die Vorschläge sollen aussehen wie die Menüs
  der Anwendung.

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
Anforderungen in [littera_schema_befund.md](littera_schema_befund.md). Vorher das Löschkonzept
gegenüber Littera ([datenschutz_offene_punkte.md](datenschutz_offene_punkte.md), B7). Die
teuerste offene Position vor dem Echtstart.
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
wurde". Ob es die `.7z` selbst öffnet oder erst beim Zurückspielen mit `SqlServerRestore.exe`
gebraucht wird, sagt das Handbuch nicht. Ohne Kennwort blieben nur die Auswertungen von Littera —
Leserliste (mit dem Leserdatenaustausch, laut Handbuch lizenzabhängig), Medienliste, Liste der
verliehenen Medien —, die sich teils als Datei ausgeben lassen („Export des Druckbildes"). Ob sie
die Nummern tragen, die die Übernahme braucht, ist nicht geprüft; die Übernahme liest die Tabellen
der Datenbank, ein Weg über Auswertungen hieße einen neuen Importer (nachgelesen am 28.09.2026).
**Nächste Schritte, sobald das Kennwort vorliegt:** (1) die `.bak` auf dem eigenen Rechner in
einen SQL Server einspielen und die dreizehn Tabellen aus Abschnitt 1 von
[SCRIPTS.md](SCRIPTS.md) als CSV ausgeben, Spaltennamen und Datumsformate gegen den Importer
prüfen, der bisher nur `mdb-export` kennt; (2) lokal und nur
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

**Standorte vor dem Lauf lesen (5.53).** Der Trockenlauf listet jeden Vermerk am Titel und jeden
Sonderstandort mit seiner Zahl ([SCRIPTS.md](SCRIPTS.md), Abschnitt 1). An der Sicherung von 2026
prüfen: (1) Die Zahl der Vermerke liegt in der Größe der Titelliste vom Juni 2026 (678 Einträge);
bei null hat `Personen_Zuordnung.Flags` in der SQL-Server-Fassung eine andere Form. (2) Was kein
Standort ist, mit `-kein-standort` ausnehmen: Verfasser ohne Komma und, falls vorhanden, Nummer
und Name eines Lesers aus einem Zeitschriften-Rundlauf. Ein Name, der hier durchrutscht, steht
danach an der Exemplarkarte, in der Titel-Verwaltung und in den Vorschlägen von „Standort
ändern", und eine Änderung schreibt ihn ins Protokoll. (3) Entscheiden, ob „Buchbestand
Bibliothek" und „Bibliothek" als Standort mitkommen, falls sie noch an Titeln stehen (2010 an
7.471 Titeln; sie nennen den gewöhnlichen Platz). Die Generalprobe zeigt nur die Zahlen; die
Listen liest, wer den Trockenlauf selbst startet.

### 7.3 S3-Auslagerung der Backups

`S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY` und `S3_BUCKET` sind leer (13.09.2026); alles
liegt auf einer Platte. Nur ein Speicher in der EU. Der Code ist fertig bis auf das Löschen
(nachgesehen am 28.09.2026): `uploadBackupToS3` in `jobs/backup.go` lädt jede Nachtsicherung
hoch, die Rotation gilt nur dem lokalen Verzeichnis. Ohne Löschregel am Speicher bliebe dort jede
Sicherung mit allen Personen ihres Stands unbegrenzt liegen. Beim Einrichten eine Löschregel am
Speicher setzen, die der Aufbewahrung der Nachtsicherung folgt (die jüngsten 14, dazu je
Kalenderwoche eine für 12 Wochen; `jobs.BehalteNaechte`, `jobs.BehalteWochen`), oder die Rotation
im Code auf den Speicher ausdehnen. Einen anderen Kopierweg als S3 gibt es im Programm nicht.
Ein zweiter Rechner per Kopierbefehl oder eine getauschte Platte leisten dasselbe ohne
Vertragsfrage, außerhalb des Programms; das ist eine Betriebsentscheidung.

### 7.4 Manuelle Restore-Probe an einem fremden Ziel

Die automatische Wochenprobe im Container lief am 13.09.2026 erfolgreich; sie ersetzt die
manuelle Probe nicht. Anleitung: [resilience_and_recovery.md](resilience_and_recovery.md),
Abschnitt 2e — dabei die neuen Befehle erproben, die bisher nur am Text geprüft sind. Sinnvoll
nach S3 oder am Schulserver. Machen soll sie die Vertretung allein mit dem Pflegekonzept (9.9);
sie wartet also auf den Schulserver und auf die benannte Vertretung.

**Der Totalverlust des Servers ist nicht beschrieben** (aus der Durchsicht von PR 631 am
21.09.2026). Der Entwurf im PR ist nach eigener Angabe unerprobt und scheitert in Schritt 5:
Das Backend hat nur benannte Volumes, die Sicherung vom zweiten Ort liegt also nicht im
Container, und die entschlüsselte Datei entsteht in einem Wegwerf-Container und ist danach
fort. Schreiben und an einem fremden Ziel durchspielen, nicht herleiten.

### 7.5 Externes Uptime-Signal

Fällt der Server ganz aus, meldet es niemand. Ein externer Monitor ruft alle 5 Minuten `/health`
ab ([DEPLOYMENT.md](DEPLOYMENT.md), Abschnitt 7.0). Etwa fünf Minuten Aufwand; beim Umzug neu
einrichten. Am Schulserver geht das erst, wenn der Eingang aus 4.23 gebaut ist, der `/health`
von außen durchlässt, und Port 443 frei ist.

### 7.6 Ruleset `main`

PR-Pflicht entfernen (Solo-Entscheidung 30.07.2026), „Block force pushes" und „Restrict
deletions" anlassen. Am 06.10.2026 trägt das Ruleset noch `pull_request`; Pushes gehen über den
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

- Sind die Admin-Konten deaktiviert?
- Erreicht der Server die DNB? Ohne sie bringt die ISBN-Abfrage der Buchmaske keine Angaben
  und der Cover-Abgleich kein Cover; gespeichert wird trotzdem. Am Abbild vom 01.10.2026 geprüft, Ausgabe „erreichbar":
  `docker exec bibliothek-backend sh -c 'wget -q -T 8 -O /dev/null "https://services.dnb.de/sru/dnb?version=1.1&operation=explain" && echo erreichbar || echo nicht erreichbar'`

### 7.10 Am Testserver ausprobieren

Nach `git pull` und `./update.sh`.

**Mit dem Handscanner.** Die Browser-Tests tippen die Zeichen blind wie ein Scanner; ob der
Scanner der Schule schnell genug tippt (höchstens 50 ms je Zeichen), zeigt nur das Gerät.

- Sperrbildschirm (Sperrfrist dafür unter Einstellungen kurz stellen): ein Buch scannen.
  Erwartet: „Scan erkannt", kein Fehlversuch; danach schließt das getippte Passwort auf.
- Theke: Leser scannen, einen Reiter der Akte anklicken, in der Akte nach unten rollen, ein
  Buch scannen. Erwartet: Die Suchleiste steht noch im Fenster, das Buch ist gebucht.
- Theke, Schnellrückgabe: den Knopf neben dem Scanfeld anklicken und einen Stapel scannen, in
  dem zwei verliehene Bücher, ein Buch aus dem Regal und eines doppelt liegen. Erwartet: Die
  verliehenen sind zurück und die Meldung nennt den Leser; das Buch aus dem Regal und der
  zweite Scan melden sich rot mit Fehlerton; kein Konto erscheint. Danach einen Ausweis
  scannen. Erwartet: Der Knopf ist aus, das Konto steht da.
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

**Das Portal ansehen** (gebaut am 06.10.2026): Im Reiter „Reservieren & Melden" steht
„Problem melden" links in einer eigenen Zeile unter dem Suchfeld. Anklicken: Das Formular
öffnet darunter ohne zweite Überschrift und beginnt an derselben linken Kante wie der Knopf.
„Abbrechen" führt zurück auf den Knopf.

**Der Standort an der Buchakte** (gebaut am 06.10.2026). Am Testserver trägt noch kein
Exemplar einen Standort. Einen Titel mit mehreren Exemplaren öffnen, Reiter „Exemplare": zwei
Exemplare ankreuzen, „Standort ändern", einen Standort eintragen. Erwartet: Die zwei Karten
nennen ihn, im Kopf der Akte steht er hinter der Signatur mit der Zahl 2, und die
Titel-Verwaltung zeigt dasselbe in der Spalte „Standort". Den Dialog noch einmal öffnen:
Das Feld schlägt den Standort von eben vor. Mit dem Handscanner: den Dialog öffnen und ein
Buch scannen. Erwartet: Das Feld bleibt leer, der Dialog bleibt offen, nichts ändert sich.

**Die Maske „Buch bearbeiten" in der neuen Reihenfolge ansehen** (gebaut am 03.10.2026):
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

**Das Mahnwesen ansehen** (gebaut am 04.10.2026): Kinder anhaken, „Mahnbriefe drucken". Es
kommt je Kind der Brief an die Eltern mit Anschrift; die Liste zeigt danach
„1× gemahnt, zuletzt …". Seit dem 05.10.2026 druckt „Liste drucken" die Liste als Tabelle, je
Buch eine Zeile; einmal ausdrucken und ansehen.

---

## 9. Pflegekonzept und Datenschutz-Nachweis

### 9.9 Was an den zwei Entwürfen offen ist

- **Nachweis der DSGVO-Konformität.** Der Entwurf steht:
  [datenschutz/nachweis.md](datenschutz/nachweis.md). Offen ist die Beschlussfassung
  ([datenschutz_offene_punkte.md](datenschutz_offene_punkte.md), Teil B).
  **Die Frist bis zum Sperrbildschirm:** Der Entwurf nennt die Vorgabe des Programms, 15
  Minuten (Abschnitt 5, „Zugang"). Gewünscht sind an der Schule 8 Stunden ohne Bedienung
  (01.10.2026). Das Feld nimmt 0 bis 1440 Minuten, 480 sind am Stack nachgestellt; die Vorgabe
  bleibt 15, der Wert wird am Schulserver einmal unter Einstellungen → Datenschutz & Sitzung
  eingetragen. Der Nachweis nennt dann die Zahl der Schule. Mit 480 Minuten greift die Sperre an einem
  Schultag nicht; für den unbeaufsichtigten Platz bleibt das Leeren der Theke nach 5 Minuten.
  Das gehört zu Teil B, B4.
- **Hosting- und Programmpflegekonzept.** Der Entwurf steht:
  [PFLEGEKONZEPT.md](PFLEGEKONZEPT.md). Offen:
  1. **Das Blatt bei der Schule** (Abschnitt 7.3 des Entwurfs) — ausfüllen bei dir, Vorlage in
     [blatt_vorlage.md](blatt_vorlage.md). Vorschlag: die zwei Schlüssel in einem Passwortmanager
     und als Papier im verschlossenen Umschlag im Tresor der Schule, nie per E-Mail.
  2. **Die Arbeitsnotizen der Entwicklung** (am 24.09.2026 219 Einträge) entlang der Gliederung
     des Entwurfs ins Repository — bei mir.
  3. **Die Probe:** Die Vertretung macht die Wiederherstellung an einem fremden Ziel (7.4) allein
     mit dem Dokument.

---

## So wird die Liste geführt

**Kästchen stehen nur im Fahrplan.** Abgehakt wird im selben Commit, der den Schritt erledigt.
Abgehakte Zeilen bleiben stehen, bis ihre Etappe fertig ist; dann fällt die Etappe weg. In den
Einzelheiten wird Erledigtes gelöscht.

**Hier steht nur Arbeit, die einer von uns beiden tun kann:** am Programm, an den Servern, auf
GitHub. Fragen an Dritte und das Warten auf ihre Antworten stehen hier nicht.

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
3. **Neues kommt nur hierher** — Funde, offene Fragen, Betriebspunkte: als Zeile im Fahrplan,
   wenn es ein Schritt ist, und mit Beleg unter einer Nummer in den Einzelheiten. Kein Issue,
   kein anderes Dokument. Eine Frage steht hier, bevor die Antwort kommt.
4. **Erledigt heißt:** im Fahrplan abhaken, in den Einzelheiten löschen — Datum und Begründung
   stehen in der Commit-Nachricht, ein Archiv gibt es nicht. Eine Antwort bekommt „Entschieden am …" und fällt weg, sobald
   sie umgesetzt ist. Die Nummer eines gelöschten Punkts wird nicht wieder vergeben —
   Kommentare im Code nennen sie als Herkunft.
5. **Die Reihenfolge** im Fahrplan wird bei jeder Änderung mitgepflegt.
