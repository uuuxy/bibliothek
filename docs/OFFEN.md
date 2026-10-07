# Offene Arbeit

Stand: 07.10.2026

**Der Fahrplan.** Oben steht, was als Nächstes getan wird, in der Reihenfolge der Arbeit: je
Schritt eine Zeile mit Kästchen. Die Nummer in Klammern führt zu den Einzelheiten weiter unten.
Einen zweiten Ort für Offenes gibt es nicht; andere Dokumente erklären (Konzept, Anleitung, der
Katalog der Bugklassen in [sweeps.md](sweeps.md)), führen aber keine eigene Offen-Liste. Wie die
Liste geführt wird, steht am Ende.

---

## Fahrplan

### Etappe 1: als Nächstes

- [x] **Theke: zweiter Scan desselben Buchs.** Eine Ausleihe klingt seit dem 06.10.2026
  anders als eine Rückgabe. Eine Sperre und eine eigene Farbe gibt es nicht.
- [x] **Standort: Ein neues Exemplar erbt ihn,** wenn alle übrigen Exemplare des Titels
  denselben tragen. Gebaut am 06.10.2026.
- [x] **Titel speichern:** Die Maske schickt seit dem 06.10.2026 nur die geänderten Felder,
  der Server schreibt nur diese. Zwei Plätze mit demselben Titel behalten beide, was sie an
  verschiedenen Feldern speichern. Ändern beide dasselbe Feld, gilt der spätere Eintrag.
- [x] **PRs auf GitHub:** 21 durchgesehen am 06.10.2026. Zwei sind hereingeholt (701, 702), 19
  geschlossen; aus sieben davon sind die Tests übernommen (d1dba610).
- [x] **PR 722** durchgesehen und geschlossen am 06.10.2026: Die Prüfung beim Löschen einer
  Sachgruppe (`api/systematik_handler.go`) würde durch ihn höchstens 1 ms schneller.
- [x] **PRs 723 und 724** durchgesehen und geschlossen am 07.10.2026, nichts übernommen.
- [x] **Doku-Ordner:** Aus 36 Markdown-Dateien in `docs/` sind am 07.10.2026 20 geworden. Die
  Architektur ist eine Datei (`ARCHITEKTUR.md`); das LUSD-Messprotokoll, die Vorlage für das
  Blatt bei der Schule, die Liste „Datenschutz — offene Punkte" und das zweite README sind in
  `LUSD.md`, im Pflegekonzept, im Datenschutz-Nachweis und im Haupt-README aufgegangen.

### Etappe 2: vor dem Echtstart

Der Echtbetrieb beginnt am Schulserver mit einer leeren Datenbank und der Littera-Übernahme
(entschieden am 28.09.2026).

- [x] Die Littera-Übernahme überträgt die Auflage („2. Aufl.", gebaut am 07.10.2026).
- [x] Die Littera-Übernahme nimmt die Nichtsortierzeichen aus Titel und Verfasser („¬Die¬
  schwarze Katze", gebaut am 07.10.2026).
- [x] Titel, Untertitel, Autor und Verlag stehen mit einem Leerzeichen zwischen den Wörtern,
  an jeder Tür: „La  Peste" aus Littera wird „La Peste", und die Suche nimmt den Suchtext in
  derselben Form (gebaut am 07.10.2026).
- [ ] Generalprobe der Übernahme mit der Sicherung von 2026, sobald sie sich öffnen lässt:
  Ausweisnummern, offene Ausleihen, Standorte, Verweise der Schlagworte, Sperren und Salden.
  (7.2, 4.20)
- [ ] `update.sh` für den Schulserver: nur Releases, Images frisch. (5.31)
- [ ] Eingang des Servers: Von außen ist nur die Seite der Lieferanten erreichbar. (4.23)
- [x] Arbeitsnotizen der Entwicklung ins Repository, entlang dem Pflegekonzept (abgeschlossen
  am 07.10.2026; wo was steht, nennt [PFLEGEKONZEPT.md](PFLEGEKONZEPT.md) 7.2).
- [ ] Am Schulserver einrichten: Sicherung außer Haus (7.3), Uptime-Signal (7.5),
  Eigentumsvermerk der Etiketten (4.24), Frist bis zum Sperrbildschirm (9.9). Danach nachsehen:
  Admin-Konten, Verbindung zur DNB (7.8).
- [ ] Abnahmen mit echten Daten. (7.7)
- [ ] Wiederherstellung an einem fremden Ziel proben, allein mit dem Pflegekonzept. (7.4)

### Bei dir

**Entscheiden:**

- [x] Die Bestandsliste (CSV) nennt je Exemplar den Standort (entschieden und gebaut am
  07.10.2026).
- [x] Theke: Ein Scan, der eintrifft, solange die vorige Buchung läuft, wird eingereiht und
  danach gebucht; dieselbe Nummer fällt weiter weg (entschieden und gebaut am 07.10.2026).
- [x] Die Titelliste steht nach dem Titel, das Ziehen der Zeilen ist entfernt (entschieden
  und gebaut am 07.10.2026).
- [x] Titelliste: Ein Artikel am Anfang zählt beim Ordnen mit, „Die schwarze Katze" steht
  unter D; Litteras Sortiertitel kommt nicht mit (entschieden am 07.10.2026: bleibt so).
- [x] Theke: Ein gescheiterter Scan gibt immer den Fehlerton, auch außerhalb der
  Schnellrückgabe (entschieden und gebaut am 07.10.2026).
- [x] Theke: Ein Scan bei offener Rückfrage (Sperre, Vormerkung, Zubehör) lässt sie stehen,
  wird nicht gebucht und gibt den Fehlerton (entschieden und gebaut am 07.10.2026).
- [x] „Mahnbriefe drucken" verlangt dasselbe Recht wie der Mahnversand (`create_orders`,
  entschieden und gebaut am 07.10.2026).
- [x] Das Aussehen nach der Umstellung der Farben auf M3-Rollen bleibt so (entschieden am
  06.10.2026). Eigene Farben je Fach gibt es nicht.
- [x] Feld „Signatur" nach der Übernahme: Vorschläge, die Seite „Signaturen" und die Auswahl
  der Inventur fassen nach der Regaladresse zusammen, dem Teil vor „ / "; am Titel bleibt die
  ganze Aufschrift (entschieden und gebaut am 07.10.2026).
- [x] Bestellung mit dem Hinweis „Mail nicht versendet", die den Händler auf anderem Weg
  erreicht hat (Telefon, eigene Mail): „Auf anderem Weg bestellt" entfernt den Hinweis nach
  einer Rückfrage, ohne zu senden. An einer Bestellung mit Bestätigungs-Link wird stattdessen
  die Zusage des Händlers nachgetragen (entschieden und gebaut am 07.10.2026).
- [x] Buchakte: Kopf und Reiter zählen den Bestand, ein bestelltes Exemplar steht im Kopf als
  „1 bestellt"; die Zahl „Exemplare" im Kopf ist entfallen (entschieden und gebaut am
  07.10.2026).

**Fertig gebaut — von dir am Testserver anzusehen,** nach `git pull` und `./update.sh` (7.10):

- [ ] Portal: „Problem melden" unter dem Suchfeld
- [ ] Buchakte: „Standort ändern", auch mit dem Handscanner
- [x] Theke: Schnellrückgabe mit einem Stapel (ausprobiert am 07.10.2026 mit zwei Büchern)
- [ ] Die übrigen Proben mit dem Handscanner
- [ ] Maske „Buch bearbeiten"
- [ ] Mahnwesen: Mahnbriefe und Liste drucken
- [ ] Ausweise aus der Leserdatei am Kartendrucker drucken
- [ ] Inventur: mehrere Bücher schnell hintereinander scannen
- [ ] Leserakte: „Stammdaten bearbeiten" speichern
- [ ] Buchakte: Status eines gesperrten Exemplars öffnen und speichern
- [ ] Bestandsliste (Einstellungen → Datenverwaltung): Spalte „Standort"; gefüllt bei
  Exemplaren, die einen Standort tragen (Buchakte, „Standort ändern")
- [ ] Medienkatalog: Titel-Verwaltung und „Suche & Filter" stehen nach dem Titel
- [ ] Theke: Ausweis und Bücher ohne Pause hintereinander scannen
- [x] Theke: bei offenem Fenster „Ausleihe blockiert" oder „Achtung! Vorgemerkt!" noch ein
  Buch scannen (ausprobiert am 07.10.2026 mit dem Handscanner: Das Fenster bleibt stehen, rot
  und Fehlerton)
- [ ] Theke: eine unbekannte Nummer scannen
- [ ] Buchakte: Kopf und Reiter „Exemplare" bei einem Titel mit einem bestellten oder
  ausgesonderten Exemplar („1 von 2 verfügbar", daneben „1 bestellt", am Reiter die 2)
- [ ] Druck-Center: Vorschau bei einem Titel mit mehr Exemplaren, als auf einen Bogen passen
  (ein Bogen, darunter „Bogen 1 von …")
- [ ] Bestellhistorie: Spalte „Stand". Scheitert der Versand einer Bestellmail oder ist kein
  Mailserver eingetragen, steht dort „Mail nicht versendet" und in der Bestellung „Erneut
  senden"; bei einem Händler, der nicht Hauptlieferant ist, daneben „Auf anderem Weg bestellt"

**Erledigen:**

- [x] SonarQube-Scan starten.
- [ ] PR-Pflicht im Regelwerk für `main` entfernen. (7.6)
- [ ] Am Testserver zählen, wie viele gesperrte Exemplare „verloren" in der Notiz tragen. (5.5)
- [ ] Das Blatt mit den zwei Schlüsseln ausfüllen. (9.9)
- [ ] Theke ohne Netz: der Nachweis von Hand im echten Chrome, zurückgestellt am 24.09.2026.
  (2.3)

### Termine

- [ ] 19. Oktober 2026: CodeQL wechselt bei GitHub das Abbild; den Lauf danach ansehen. (5.10)
- [ ] Ab dem 28. Oktober 2026: Node 26, nach der Regel „immer die aktive LTS"
  ([PFLEGEKONZEPT.md](PFLEGEKONZEPT.md), Abschnitt 4).

### Später: kleine Fehler und Aufräumarbeit

Ohne feste Reihenfolge. Jede Zeile bündelt, was unter ihrer Nummer in den Einzelheiten steht.
Was davon fertig ist, wird dort gelöscht und fällt aus der Zeile; abgehakt wird sie, wenn unter
der Nummer nichts mehr dazu offen ist.

- [ ] **Jahrgang am Titel (5.5):** „Klasse" und „von … bis" werden eine Angabe, „unbekannt" eine
  eigene (entschieden am 24.09.2026). Davor: je Titel festlegen, welche Spanne gilt.
- [x] **Buchakte (5.5):** Ein bestelltes Exemplar, das nie eintraf, steht nach „Exemplar löschen"
  nicht mehr im Abgangsbuch; ausgesonderte und bestellte Exemplare heißen dort seit dem
  07.10.2026 „Ausgesondert" und „Bestellt".
- [ ] **Nie eingetroffene Exemplare (5.5):** Sie zählen noch an zwei Stellen als Abgang, in der
  Zahl „aus dem Katalog gelöscht" unter dem Abgangsbuch und in der Verlustquote der Statistik.
- [ ] **Druck-Center (5.45):** Die Auswahlliste nennt auch ein bestelltes Exemplar
  „(Neuwertig)". Die Vorschau zeichnet seit dem 07.10.2026 nur den ersten Bogen.
- [ ] **Überläufe (5.45):** Bestellwesen 40 px, Signaturen bei 1280 px, ein langer Name in der
  Leserakte.
- [ ] **Auskunft (5.19):** Zwei Einträge über die Anlage eines Kontos fehlen, sobald das Konto
  gelöscht ist; zu klären, welche Rohdaten aufs Blatt gehören.
- [ ] **Protokoll und Tilgung (5.35):** Einträge, die einen Leser nur über seine Forderung meinen;
  am Testserver alte Einträge zu schon gelöschten Lesern.
- [ ] **Gates und Werkzeuge (5.10):** Lücken in Prüfregeln und Tests, der Wechsel auf Ubuntu 26.
- [ ] **Zwei Helfer (5.5):** Beträge und Fehlertexte schreiben 27 Stellen selbst.
- [ ] **Masken, die ihren ganzen Stand zurückschicken (5.5):** Benutzer, Gerät, Lieferant und die
  Kategorien der Einstellungen; am Titel und am Leser ist es behoben.
- [x] **Bestellung ohne Versandstand (5.5):** Scheitert die Mail an den Lieferanten, steht es an
  der Bestellung und in der Bestellhistorie („Mail nicht versendet"), und die Bestellung lässt
  sich erneut senden (entschieden und gebaut am 07.10.2026, Migration 161).
- [ ] **Spur im Protokoll (5.57):** die übrigen ändernden Routen lesen und je Tür festlegen, ob
  sie einen Eintrag schreibt. Klassenleitungen, Mail-Vorlagen, Lieferanten und die
  Verlängerung der Lernmittel einer Klasse schreiben ihn seit dem 07.10.2026.
- [ ] **Beim Umstellen der Farben aufgefallen (5.21):** In der Inventur zeigt ein unbekannter
  Barcode einen technischen Fehlertext; die Titelliste öffnet einen Titel nur mit der Maus;
  englische Wörter in der Oberfläche; eine überfällige Ausleihe ist in der Buchakte nur an der
  Farbe zu erkennen; in der Bestellhistorie die Spalte „Lieferant" im Browser nachmessen.

### Nur mit Anlass: kein Schritt

Bekannt und beschrieben. Gebaut wird erst, wenn der dort genannte Anlass eintritt.

- 5.5 Die kurze Nummer der Littera-Etiketten an der Theke: entscheidet sich mit der Generalprobe.
- 5.5 Listenimport: Trägt eine Zeile die Ausweisnummer eines Lesers als Buchnummer, nennt die
  Meldung weder Zeile noch Weg.
- 5.45 Bedienung am Tablet und in Fenstern unter 1280 px.

Beobachtungen und Kleinigkeiten ohne geplanten Schritt stehen seit dem 07.10.2026 nicht mehr
hier, sondern in [ARCHITEKTUR.md](ARCHITEKTUR.md) 11.5 „Bekannte Grenzen" (bis dahin Abschnitt 6
und die Punkte 5.25, 5.49 und 5.54).

---

## Einzelheiten

Zum Nachschlagen: Beleg, Messwert und nächster Schritt je Punkt. Die Nummern bleiben fest und
werden nicht neu vergeben; Kommentare im Code nennen sie als Herkunft.

## 2. Offline-Betrieb der Theke — der Nachweis steht aus

Offen ist **der Nachweis (2.3):** Stufe 1 und 3 gehören von Hand in den echten Chrome, Stufe 2
über die Tür (`POST /api/action/nachbuchen`). Wie sich die Theke ohne Verbindung verhält, steht
in [FACHKONZEPT.md](FACHKONZEPT.md) 18.4 und im [Handbuch](HANDBUCH.md).

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
[Architektur 8.8](ARCHITEKTUR.md#88-echtzeit-und-offline), „Offline"). Das Uptime-Signal von außen (7.5)
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
setzen. Mit dem Bau [Architektur 7](ARCHITEKTUR.md#7-verteilungssicht) (dort „:80/:443 öffentlich") und
[Architektur 9](ARCHITEKTUR.md#9-architekturentscheidungen) nachziehen.

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
- **Eine Maske schickt alle Felder zurück, auch die, die niemand angefasst hat** (Raster,
  Frage 18; [sweeps.md](sweeps.md), „Absoluter Wert aus dem Ladezeitpunkt"). Am Titel
  (06.10.2026) und an der Leserakte (07.10.2026) behoben. Am 07.10.2026 am Code gelesen,
  mit derselben Form:
  - **Benutzer:** `PUT /api/benutzer/{id}` schreibt Name, E-Mail, Rolle, „aktiv" und die
    Ausweisnummer der Leserzeile (`UpdateUser` in `repository/user.go`); die Maske füllt
    sich aus der Zeile der Liste (`benutzerFormularAus`), die beim Öffnen der Seite und nach
    jedem eigenen Speichern lädt. Hat inzwischen jemand das Konto deaktiviert, die Rolle
    geändert oder in der Leserakte eine Ausweisnummer eingetragen, schreibt das Speichern den
    alten Stand zurück. Dafür muss die Tür Teil-Änderungen annehmen (heute sind alle Felder
    Pflicht). Kategorie B.
  - **Gerät:** „Bearbeiten" und „Defekt melden" schicken Modell, Zubehör und Notiz aus der
    Zeile der Liste (`GeraeteVerwaltung.svelte`). Kategorie C.
  - **Lieferant:** „Hauptlieferant" geht aus der Zeile mit; Stammdaten und Hauptlieferant
    schreibt die Tür in zwei Schritten ohne gemeinsame Transaktion
    (`handleUpdateSupplier`). Kategorie C.
  - **Einstellungen:** Jede Kategorie schickt alle ihre Felder (`speichereKategorie`).
    Kategorie C.
- **Status-Editor, Altbestand:** Bis zum 07.10.2026 öffnete der Editor ein gesperrtes
  Exemplar als „Verloren", wenn die Notiz das Wort enthielt, und das Speichern sonderte es
  mit dem Grund VERLUST aus. Ob das am Testserver Exemplare getroffen hat, zeigt (lesend):
  `docker exec bibliothek-db psql -U postgres -d bibliothek -c "SELECT count(*) FILTER (WHERE NOT ist_ausgesondert) AS gesperrt, count(*) FILTER (WHERE ist_ausgesondert AND aussonderung_grund = 'VERLUST') AS als_verlust FROM buecher_exemplare WHERE NOT ist_ausleihbar AND zustand_notiz ILIKE '%verloren%';"`
  „gesperrt" sind Exemplare, die der Fehler noch hätte treffen können; „als_verlust" sind die,
  bei denen nachzusehen ist, ob das Buch wirklich fehlt.
- Geräte: Eine doppelte Seriennummer meldet „Barcode ist bereits an ein anderes Gerät
  vergeben" (`CreateGeraet` in `repository/geraete.go` liest jede Eindeutigkeits-Verletzung
  als Barcode; am 07.10.2026 am Code gelesen, nicht nachgestellt). Kategorie C.
- **Ein bestelltes Exemplar, das nie eintraf, zählt noch an zwei Stellen als Abgang** (gefunden
  am 07.10.2026, am Code gelesen, nicht nachgestellt). In den Zeilen des Abgangsbuchs und in
  seiner Zahl der Abgänge ohne Zeitpunkt steht es seit dem 07.10.2026 nicht mehr
  (`sqlIstAbgang` in `repository/abgangsbuch.go`). Die Zahl „aus dem Katalog gelöscht" darunter
  zählt weiter jedes Exemplar, das mit seinem Titel gelöscht oder in der Inventur als Verlust
  endgültig gelöscht wurde: Die drei Schreiber der Spur vermerken nicht, ob es je im Bestand
  war (`repository/audit_books.go`, `inventur/db_books_delete_spur.go`,
  `repository/inventur_verlust_aktionen.go`). Verlustquote und Wiederbeschaffungswert der
  Statistik zählen jedes Exemplar mit dem Grund VERLUST oder BESCHAEDIGUNG
  (`queryBestandKennzahlen` in `api/stats.go`); ein bestelltes Exemplar, das im Status-Editor
  als „Verloren" ausgesondert wird, bekommt VERLUST. Abhilfe: Die Schreiber vermerken es an
  der Spur, die Statistik nimmt die Grenze des Abgangsbuchs. Kategorie B.
- Zwei Schreibweisen stehen neben ihrem Helfer (gezählt am 06.10.2026). Einen Betrag in Euro
  schreiben sieben Stellen selbst: sechs mit `toLocaleString` und `+ ' €'`
  (`useFehlbestand.svelte.js`, `StudentBescheideCard`, `BestellHistorie`, `BestellDetail`,
  `BescheidDialog`, `BescheideTabelle`), eine mit `toFixed` (`OrderCart.svelte`), alle mit
  gewöhnlichem statt geschütztem Leerzeichen; dafür gibt es `formatEuro` (`utils/format.js`).
  Den Text eines gefangenen Fehlers (`e instanceof Error ? e.message : String(e)`) schreiben 20
  Stellen in 12 Dateien selbst; `fehlertext` (`utils/fehlertext.js`) rufen vier Dateien.
  Kategorie B.

### 5.10 Gates und Werkzeuge

- Die Schema-Gegenrichtung ist blind für UNIQUE, Teilindizes und RESTRICT.
- Kein Gate gegen unbegrenzte Listen-Endpunkte.
- Kein Rückweg für ältere Sicherungen beim Wechsel des `BACKUP_ENCRYPTION_KEY`.
- `TestHandlerFormulierenKeinNeuesSQL` (`api/schichtung_test.go`) sieht ein `UPDATE` mit
  Tabellenkürzel nicht: Das Muster verlangt `UPDATE <Tabelle> SET`, und `UPDATE ausleihen a SET`
  trifft es nicht. Fünf Anweisungen dieser Form stehen in `api/ausleihe.go`,
  `api/etiketten_offen.go`, `api/mail_routes.go`, `api/student_promotion.go` und
  `api/supplier_handler.go` (gezählt am 07.10.2026); die Dateien stehen wegen anderer
  Anweisungen in der Liste. Ein neuer Handler, dessen einzige Anweisung so aussieht, bliebe
  unbemerkt. Kategorie B.
- `e2e/kontrast.spec.js` misst den Medienkatalog nicht in jedem Lauf mit seinen Kacheln
  (gefunden am 02.10.2026, lokal mit 8.600 Titeln). `warteAufStabilenBaum` gilt als stabil,
  sobald zwei Zählungen im Abstand von 100 ms gleich sind; kommt die Titelliste später, misst
  der Test die Seite ohne Kacheln und geht weiter. Belegt an einer Kachel, die den Klick auf
  den nächsten Menüpunkt scheitern ließ, solange der Test ihn per Teilwort traf: fünf von
  sechs Läufen rot, einer grün, die Kacheln standen dort also noch nicht. Abhilfe: je Seite
  auf ein Merkmal des Inhalts warten (Kachel, Tabellenzeile). Dieselbe Form des Wartens steht
  in `typo-rollen.spec.js`, `control-hoehen.spec.js` und `icon-trefferflaechen.spec.js`, dort
  nicht nachgemessen. Kategorie B.
- CI, Schritt „Install Playwright" (`.github/workflows/ci.yml`): Am 07.10.2026 endete der Job
  `e2e` rot, bevor ein Test lief (Lauf 37670559983, Commit f2009b9b). Der erste Versuch von
  `npx playwright install --with-deps chromium` blieb beim Lesen der Paketlisten stehen (ab
  18:58:06 Uhr UTC keine Ausgabe mehr, davor „Ign" für `azure.archive.ubuntu.com`) und lief
  in die Frist von 600 s. Sein `apt-get` (Prozess 8370) lief danach weiter und hielt
  `/var/lib/apt/lists/lock`; Versuch 2 und 3 scheiterten binnen einer Sekunde an dieser Sperre.
  `DPkg::Lock::Timeout` hat auf sie nicht gewartet. Abhilfe: vor einem neuen Versuch warten,
  bis kein `apt-get` mehr läuft, oder den übrig gebliebenen beenden. Kategorie B.
- `e2e/suchpille-einheitlich.spec.js` war am 07.10.2026 in der CI einmal rot und in der
  Wiederholung für denselben Commit grün (Lauf 37662472141, Commit 4b6b0311; lokal 54 von 54
  grün): Auf der Seite „Signaturen" maß die Spec die Pille ohne Fokus, obwohl sie das Feld
  davor fokussiert. `fokussiertMessen` wartet auf zwei gleiche Messungen und prüft nicht, ob
  das Feld den Fokus noch trägt; was ihn dort nimmt, ist nicht gefunden. Abhilfe: nach der
  Messung prüfen, dass das Feld `document.activeElement` ist, sonst neu fokussieren. Kategorie B.
- `e2e/feld-roundtrip.spec.js` („Buch anlegen") war am 07.10.2026 lokal in einem Lauf über 21
  Specs einmal rot und allein wiederholt grün (6 von 6): Nach „Speichern" wartet die Spec zehn
  Sekunden darauf, dass der neue Titel in der Liste steht; die lokale Datenbank trägt 12.909
  Titel. Am Stack nachgestellt: Das Buch wird gespeichert und steht in der Liste. Kategorie C.
- 46 Specs klicken Menüpunkte per `page.getByTitle('<Name>')`, 94 Stellen (gezählt am
  06.10.2026). Das trifft jedes Element, dessen `title` den Namen enthält, auch die Kachel
  eines Buchs. `e2e/abgaenger-management.spec.js` legt Titel „Abgänger Buch …" an und räumt
  sie nicht ab. Die zwei Klicks auf „Abgänger" (`schueler-profil-klick.spec.js`) kommen heute
  von der Theke, wo keine Kachel steht; vom Katalog aus träfen sie auch die Kachel. Abhilfe:
  `menuepunkt` aus `e2e/helpers.js` an allen Stellen (heute in drei Specs), und die Spec räumt
  ihren Titel ab. Kategorie B.
- Browser-Tests lassen Daten liegen (in der CI ist die Datenbank je Lauf frisch; lokale
  Zahlen tragen die Reste mit). Am 06.10.2026 einzeln gestartet und an der lokalen Datenbank
  gezählt: `e2e/bestellung-detail.spec.js` bestellt drei Exemplare am ersten Titel des
  Katalogs und nimmt nur den Lieferanten wieder weg; der Teardown löscht die Bestellung, die
  drei Exemplare bleiben „im Zulauf" ohne Bestellung. `e2e/abgaenger-management.spec.js` lässt
  einen Titel „Abgänger Buch …" mit einem Exemplar liegen, `e2e/zugangsbuch.spec.js` einen
  Titel mit zwei Exemplaren. Am Code gelesen: Der Wareneingang-Test in
  `e2e/scrollbereiche.spec.js` legt acht Titel mit je einem Exemplar im Zulauf an und räumt
  sie nicht ab. Abhilfe je Spec: eigener Titel, Aufräumen über die Kennung. Kategorie B.
- Code, den kein Go-Test ausführt (gemessen am 07.10.2026 mit der ganzen Suite und `-coverpkg`
  über alle Pakete: 85,3 % der Anweisungen; lokal zählt `./...` das Go-Paket mit, das npm unter
  `frontend/node_modules/flatted` ablegt, mit ihm sind es 84,9 %). Unter 50 % liegen, ohne
  `cmd/`, `main.go` und Dateien mit weniger als 20 Anweisungen, acht Dateien:
  `api/littera_import.go` 0,7 % (Littera- und Bestandsdatei hochladen; die Regeln in
  `internal/littera` 88,6 %), `api/schueler_etiketten.go` 2,3 %,
  `internal/service/cover_service.go` 18,2 %, `api/ausweis_layout.go` 33,3 %, `db/seed.go`
  35,5 %, `repository/mail_settings.go` 38,1 %, `api/orders_handler.go` 44,2 % (mehrere
  Exemplare im Wareneingang buchen 4,3 %, die Bestellsuche 13,3 %), `api/geraete.go` 49,0 %.
  Ob Browser-Tests diesen Code erreichen, ist nicht gemessen. Anlass: Das Nachziehen der
  Tests für fünf Routen am 03.10.2026 fand drei Fehler (zwei Abweisungen beim Zusammenführen
  ohne Grund, ein unlesbares Bild als Störung gemeldet, eine Antwort des Foto-Uploads, die
  kein JSON war).
  Abhilfe je Route: ein Test mit Datenbank und eine Gegenprobe je Zusicherung, Muster in
  `api/inventur_verlust_aktionen_pg_test.go`. Kategorie B.
- `beforeEach(() => attrappe.mockReset())` steht in 16 Testdateien der Oberfläche an 19
  Stellen, in einer davon mit `mockClear` (gezählt am 06.10.2026). Die Kurzform gibt die
  Attrappe zurück, und Vitest ruft eine Funktion, die ein Hook zurückgibt, nach dem Test als
  Aufräumer auf: Jeder Test ruft die Attrappe danach noch einmal. Wirft oder scheitert sie dann (`mockRejectedValue`), wird der Test rot, obwohl seine
  Erwartungen stimmen; nachgestellt an `klassensatzReservierung.svelte.test.js`. Abhilfe: der
  Rumpf des Hooks in geschweiften Klammern. Kategorie B.
- **Der Wechsel auf Ubuntu 26 als Runner.** Seit dem 28.09.2026 laufen alle zehn Jobs fest auf
  `ubuntu-24.04` statt auf `ubuntu-latest`, das ab dem 19. Oktober 2026 auf Ubuntu 26 zeigt
  (actions/runner-images#14748). Den Wechsel selbst legen, mit einem eigenen Lauf gegen das
  neue Abbild — brechen kann etwa der Postgres-Client oder die Chromium-Abhängigkeiten von
  Playwright; spätestens, wenn GitHub `ubuntu-24.04` abkündigt. Nicht darunter: CodeQL läuft
  in der Standard-Einrichtung von GitHub (Repository-Einstellung, keine Workflow-Datei) auf
  `ubuntu-latest` und wechselt am 19. Oktober 2026 mit; der Hinweis darauf steht an jedem
  CodeQL-Lauf (gesehen am 28.09.2026). Bricht die Analyse dort, wird der CodeQL-Lauf rot.
- **gosec: acht Regeln global ausgenommen** (gemessen mit v2.29.0 am 28.09.2026, ohne
  `-exclude`): G706 (38 Stellen in 20 Dateien, nachgezählt am 07.10.2026), G704 (6), G703 (5),
  G120 (5), G124 (4), G404 (4), G115 (3), G101 (1); der Grund je Regel steht in
  `.github/workflows/security-scan.yml`.
  Eine neue Stelle dieser Regeln meldet gosec nicht. Abhilfe: je Stelle ein `#nosec` mit Grund,
  dann die Regel aus `-exclude` nehmen — außerhalb von G706 sind es 28 Stellen in 14 Dateien.
  Nur mit Anlass.
- **Ein Browser-Test war viermal rot.** `e2e/feld-roundtrip.spec.js` („Buch anlegen: Bestand
  und Zähldatum kommen in der DB an") fand am 05.10.2026 in zwei vollen Läufen und am
  06.10.2026 in einem Lauf über neun Dateien und in einem vollen Lauf am lokalen Stack den
  neuen Titel nicht binnen 10 s; einzeln lief die Datei danach jedes Mal grün. Beim zweiten
  Mal war die Maske zu und die Liste stand da („Bücher (10618)"), der neue Titel fehlte in
  der Ansicht. Die lokale Datenbank trägt 12.455 Titel (gezählt am 06.10.2026), darunter die
  Reste früherer Testläufe.

### 5.19 Lesepfade gegen die Sicht `schueler` — was offen bleibt

**Rohdaten der Protokolleinträge:** Die Rohdaten der Protokolleinträge
(`details`) stehen nur in der abgerufenen Auskunft, nicht auf dem Blatt; das Gate
`TestDsgvoPDF_DrucktJedeAngabeDerAuskunft` führt sie als begründete Ausnahme, seit dem
24.09.2026 auch die Details der Kontoereignisse. Offen ist, was davon aufs Blatt gehört.
Nachgesehen am 24.09.2026: Die bearbeitende Person steht in eigenen Spalten (`bearbeiter_id`,
`admin_id`), die die Auskunft nicht ausgibt; Freitexte in den Details — etwa der Grund einer
Sperre (`LESER_GESPERRT`, `LESER_ENTSPERRT`; bis zum 24.09.2026 auch `OVERRIDE_BLOCK`) —
können aber andere Personen nennen.

**Zwei Einträge über die Anlage eines Kontos** (gefunden am 29.09.2026) tragen seine Kennung
nicht als `ziel_id` und fehlen deshalb unter den früheren Zugangskonten der Auskunft, sobald
das Konto gelöscht ist.

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

### 5.21 Beim Umstellen der Farben aufgefallen

Jeweils am Code nachgesehen:

- Buchakte, Liste der Ausleiher: Eine überfällige Ausleihe ist nur an der Farbe des Datums zu
  erkennen (`BorrowersListe.svelte`). Die Leserakte setzt für dieselbe Ausleihe ein Zeichen und
  für Screenreader das Wort „Überfällig" dazu (`AusleiheRueckgabe.svelte`).
- Einstellungen, „E-Mail Routing für Mahnungen": Die Oberfläche sagt „Mapping" (leere Liste,
  Meldung nach dem Löschen, Sprechblase am Papierkorb; `SystemSettingsRouting.svelte`); ein
  deutsches Wort wäre „Zuordnung".
- Titel-Verwaltung: Ein Titel lässt sich in der Liste nur mit der Maus öffnen. Der Klick hängt
  an der Zeile (`BookTableZeile.svelte`, `onclick` am `<tr>`), die Zeile nimmt keinen Fokus. Die
  Leserdatei öffnet die Akte über den Namen als Knopf.
- Titel-Verwaltung: Der Knopf „Retry Cover" trägt eine englische Beschriftung.
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
- Ausweis-Designer: Die Knöpfe der Textausrichtung tragen englische Hinweise („left",
  „center", „right"; `PropertiesText.svelte`).

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

Gefunden am 29.09.2026. Die Tilgung nimmt Name und Freitext neben der Kennung des Lesers aus
beiden Protokollen (`repository/protokoll_personenbezug.go`). Sie erreicht nicht:

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

### 5.45 Listen in einem Kasten mit eigenem Scrollen

Eine Liste zeigt alle Zeilen, gescrollt wird der Bereich der Seite
(`e2e/scrollbereiche.spec.js`); so stehen die Ausleihliste der Leserakte, die Positionen im
Wareneingang und die Exemplare in der Maske „Buch bearbeiten". Kategorie B. Offen:

- **Druck-Center, Schritt 2** (`LabelBarcodeSchritt.svelte`, `max-h-40`): Der Kasten bleibt
  als Auswahlliste; darüber stehen ein Kästchen für alle und ein Feld für die Nummer. Offen:
  Jede Zeile nennt „(Neuwertig)", wenn das Exemplar keine Zustandsnotiz trägt, auch ein
  bestelltes.
- **Bestellwesen:** Die Seite läuft 40 px über (857 von 817 px bei 1710 × 952 px, bei
  1280 × 720 px sind es 60 px; gemessen am 06.10.2026). Darin scrollt die Bedarfsliste in
  einem eigenen Kasten (`OrderRecommendations.svelte`, `max-h-[calc(100vh-19rem)]`).
- **Signaturen bei 1280 px:** Liste und Regal sind zusammen 1.010 px breit, Platz sind 960 px
  (gemessen am 06.10.2026). Die Spalte „verliehen" endet 18 px hinter dem Fensterrand, die
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
- **Leserakte, langer Name:** Ein Name aus einem Wort von 22 Zeichen ragt 72 px aus der
  Leserkarte in die rechte Spalte (gemessen am 06.10.2026 bei 1280 × 900 px an einem
  Testleser). Er liegt dort unter den Reitern auf der ersten Zeile des Inhalts: In
  „Stammdaten & Adresse" verdeckt er den Anfang der Überschrift (gesehen am 06.10.2026), in
  „Gebühren & Schäden" ebenso, in „Ausleihen & Vormerkungen" endet er an der Oberkante der
  Überschrift (gemessen am 03.10.2026). Namen mit Leerzeichen oder Bindestrich brechen um.
- **Leserakte, Autor und Nummer des Exemplars:** Der Autor steht nur in der Sprechblase am
  Titel, die Nummer in Fenstern bis rund 1580 px ebenfalls (darüber hat sie ihre Spalte;
  gemessen bei ausgeklappter Seitenleiste). Die Sprechblase erscheint beim Zeigen mit der
  Maus; an einem Tablet ohne Maus sind beide Angaben in der Akte nicht zu sehen.
  Vorleseprogramme bekommen sie als unsichtbaren Text.
- **Leserakte, doppelte Beschriftung:** Unter dem Reiter „Stammdaten & Adresse" steht dieselbe
  Überschrift noch einmal; im Reiter „Gebühren & Schäden" heißt die Liste seit dem 01.10.2026
  „Forderungen".

### 5.57 Spur im Protokoll: welche Tür schreibt einen Eintrag

Raster, Frage 19. Eine grobe Messung über sechs Pakete (07.10.2026) nennt 60 von 97 ändernden
Routen, deren Anmeldung keine Funktion nennt, die in `audit_log` oder `audit_logs` schreibt.
Sie sieht Routen über eine Variable nicht (die Buchung der Theke) und keine Trigger; die Liste
ist ein Suchvorrat, kein Befund. Gelesen und mit Eintrag versehen sind die Rechte-Matrix, die
Zuordnung der Klassenleitungen, die Mail-Vorlagen, die Lieferanten und die Verlängerung der
Lernmittel einer Klasse; der Rest ist nicht gelesen. Nächster Schritt: je Tür festlegen, ob
sie einen Eintrag schreibt. Was darin steht, regelt `api/verwaltung_protokoll.go`: Bearbeiter,
Gegenstand und die Namen der geänderten Felder, keine Mailadresse. Kategorie B.

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
gegenüber Littera ([datenschutz/nachweis.md](datenschutz/nachweis.md), Abschnitt 10, B7). Die
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

**Standorte vor dem Lauf lesen.** Der Trockenlauf listet jeden Vermerk am Titel und jeden
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

**Sperren und Salden aus Littera (gefunden am 07.10.2026).** Die Übernahme liest sie nicht: Wer
in Littera gesperrt ist, kommt ohne Sperre an
([littera_schema_befund.md](littera_schema_befund.md), „Was NICHT übernommen wird"). In der
Sicherung von 2010 sind es 5 Sperren und 4 Salden ungleich 0. An der Sicherung von 2026 zählen
(`Leser.GesperrtAm`, `Leser.Saldo`), danach entscheiden, ob Nachtragen von Hand genügt.

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
Scanner der Schule schnell genug tippt (höchstens 50 ms je Zeichen), zeigt nur das Gerät. An
einem offenen Fenster der Theke ist es am 07.10.2026 belegt: Der Scan wurde erkannt.

- Sperrbildschirm (Sperrfrist dafür unter Einstellungen kurz stellen): ein Buch scannen.
  Erwartet: „Scan erkannt", kein Fehlversuch; danach schließt das getippte Passwort auf.
- Theke: Leser scannen, einen Reiter der Akte anklicken, in der Akte nach unten rollen, ein
  Buch scannen. Erwartet: Die Suchleiste steht noch im Fenster, das Buch ist gebucht.
- Theke, ein gescheiterter Scan: eine Nummer scannen, die das Programm nicht kennt. Erwartet:
  die rote Meldung unter dem Scanfeld, dazu roter Blitz und Fehlerton.
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

**Die Inventur mit dem Handscanner.** Eine Inventur für eine Signatur
starten und fünf Bücher so schnell hintereinander scannen, wie es geht. Erwartet: Die Zahl
„erfasst" steht danach auf 5; auch ein Scan, der kommt, solange der Drehkreis im Feld steht,
zählt.

**Die Leserakte.** Reiter „Stammdaten & Adresse", „Bearbeiten", die Eltern-E-Mail ändern,
„Speichern". Erwartet: Die Maske schließt, die Akte zeigt die neue Adresse, Klasse und
Ausweisnummer stehen wie vorher.

**Der Status eines Exemplars.** In der Buchakte ein Exemplar auf „Gesperrt" stellen und in die
Notiz „CD verloren" schreiben, speichern, den Status noch einmal zum Ändern öffnen. Erwartet:
Die Auswahl steht auf „Gesperrt", nicht auf „Verloren".

**Das Portal ansehen:** Im Reiter „Reservieren & Melden" steht
„Problem melden" links in einer eigenen Zeile unter dem Suchfeld. Anklicken: Das Formular
öffnet darunter ohne zweite Überschrift und beginnt an derselben linken Kante wie der Knopf.
„Abbrechen" führt zurück auf den Knopf.

**Der Standort an der Buchakte.** Am Testserver trägt noch kein
Exemplar einen Standort. Einen Titel mit mehreren Exemplaren öffnen, Reiter „Exemplare": zwei
Exemplare ankreuzen, „Standort ändern", einen Standort eintragen. Erwartet: Die zwei Karten
nennen ihn, im Kopf der Akte steht er hinter der Signatur mit der Zahl 2, und die
Titel-Verwaltung zeigt dasselbe in der Spalte „Standort". Den Dialog noch einmal öffnen:
Das Feld schlägt den Standort von eben vor. Mit dem Handscanner: den Dialog öffnen und ein
Buch scannen. Erwartet: Das Feld bleibt leer, der Dialog bleibt offen, nichts ändert sich.

**Die Maske „Buch bearbeiten" ansehen:**
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

**Der Ausweisdruck am Kartendrucker.** In der Leserdatei zwei Leser ankreuzen, „Ausweise
drucken". Erwartet: Jede Karte trägt nur den Ausweis, oben steht keine Reiterzeile, und die
Nummer unter dem Strichcode steht auf der Karte. Dasselbe mit „Testdruck Vorderseite" im
Druck-Center, Reiter „Schülerausweise". Gemessen ist das in der Druckansicht des Browsers
(`e2e/ausweis-druckseite.spec.js`), am Kartendrucker noch nicht.

**Das Mahnwesen ansehen:** Kinder anhaken, „Mahnbriefe drucken". Es
kommt je Kind der Brief an die Eltern mit Anschrift; die Liste zeigt danach
„1× gemahnt, zuletzt …". Seit dem 05.10.2026 druckt „Liste drucken" die Liste als Tabelle, je
Buch eine Zeile; einmal ausdrucken und ansehen.

---

## 9. Pflegekonzept und Datenschutz-Nachweis

### 9.9 Was an den zwei Entwürfen offen ist

- **Nachweis der DSGVO-Konformität.** Der Entwurf steht:
  [datenschutz/nachweis.md](datenschutz/nachweis.md). Offen ist die Beschlussfassung
  (Abschnitt 10 des Entwurfs, B1 bis B7).
  **Die Frist bis zum Sperrbildschirm:** Der Entwurf nennt die Vorgabe des Programms, 15
  Minuten (Abschnitt 5, „Zugang"). Gewünscht sind an der Schule 8 Stunden ohne Bedienung
  (01.10.2026). Das Feld nimmt 0 bis 1440 Minuten, 480 sind am Stack nachgestellt; die Vorgabe
  bleibt 15, der Wert wird am Schulserver einmal unter Einstellungen → Datenschutz & Sitzung
  eingetragen. Der Nachweis nennt dann die Zahl der Schule. Mit 480 Minuten greift die Sperre an einem
  Schultag nicht; für den unbeaufsichtigten Platz bleibt das Leeren der Theke nach 5 Minuten.
  Das gehört zu B4.
- **Hosting- und Programmpflegekonzept.** Der Entwurf steht:
  [PFLEGEKONZEPT.md](PFLEGEKONZEPT.md). Offen:
  1. **Das Blatt bei der Schule** (Abschnitt 7.3 des Entwurfs) — ausfüllen bei dir, Vorlage im
     Anhang des Entwurfs. Vorschlag: die zwei Schlüssel in einem Passwortmanager
     und als Papier im verschlossenen Umschlag im Tresor der Schule, nie per E-Mail.
  2. **Die Probe:** Die Vertretung macht die Wiederherstellung an einem fremden Ziel (7.4) allein
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
| **C** | „Wenn ich schon mal hier bin" — Umbenennungen, Stilfragen, Refactorings ohne Anlass                                                                                                                  | Nur mit Anlass und Zeit; steht in ARCHITEKTUR.md 11.5, nicht hier    |

1. **Ein Fund = ein Commit.** Was beim Reparieren zusätzlich auffällt, kommt hierher, nicht in
   denselben Commit.
2. **Kategorie A wird belegt, nicht behauptet:** ein Test, der am alten Code rot wird. Bis ein
   Fund nachgestellt ist, heißt er „Verdacht".
3. **Neues kommt nur hierher** — Funde, offene Fragen, Betriebspunkte: als Zeile im Fahrplan,
   wenn es ein Schritt ist, und mit Beleg unter einer Nummer in den Einzelheiten. Kein Issue,
   kein anderes Dokument. Eine Frage steht hier, bevor die Antwort kommt. Eine Beobachtung
   ohne Schritt (Kategorie C) kommt nach [ARCHITEKTUR.md](ARCHITEKTUR.md) 11.5.
4. **Erledigt heißt:** im Fahrplan abhaken, in den Einzelheiten löschen — Datum und Begründung
   stehen in der Commit-Nachricht, ein Archiv gibt es nicht. Eine Antwort bekommt „Entschieden am …" und fällt weg, sobald
   sie umgesetzt ist. Die Nummer eines gelöschten Punkts wird nicht wieder vergeben —
   Kommentare im Code nennen sie als Herkunft.
5. **Die Reihenfolge** im Fahrplan wird bei jeder Änderung mitgepflegt.
