# Offene Arbeit

Stand: 08.10.2026

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
- [x] **PR 725** durchgesehen am 08.10.2026, nicht übernommen; die Idee ist gebaut: Im Warenkorb
  des Bestellwesens ist „Menge verringern" bei Menge 1 gesperrt, in der Form der anderen
  gesperrten Symbolknöpfe.
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
- [x] Jahrgang am Titel: Der Umbau wie in 5.5 ist freigegeben (entschieden am 08.10.2026).
  „Klasse" entfällt, „von" und „bis" werden Auswahlfelder, ohne Eintrag gilt „unbekannt"; die
  Klassen-Einträge des Testservers verfallen. Die Maske wird als Bild gezeigt, bevor sie
  sichtbar wird.
- [x] Auskunft: Die Protokolleinträge auf dem Blatt nennen in Worten, worum es ging (Buch,
  Grund, Betrag), ohne die Kennungen des Programms und ohne das Konto, das gebucht hat
  (entschieden am 08.10.2026). (5.19)

**Fertig gebaut — von dir am Testserver anzusehen,** nach `git pull` und `./update.sh` (7.10):

- [ ] Portal: „Problem melden" unter dem Suchfeld
- [ ] Buchakte: „Standort ändern", auch mit dem Handscanner
- [x] Theke: Schnellrückgabe mit einem Stapel (ausprobiert am 07.10.2026 mit zwei Büchern)
- [ ] Die übrigen Proben mit dem Handscanner
- [ ] Maske „Buch bearbeiten"
- [ ] Mahnwesen: Mahnbriefe und Liste drucken
- [ ] Ausweise aus der Leserdatei am Kartendrucker drucken
- [ ] Inventur: mehrere Bücher schnell hintereinander scannen
- [ ] Leserakte: im Reiter „Stammdaten & Adresse" über „Bearbeiten" ändern und speichern
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
- [ ] Medienkatalog, Titel-Verwaltung: einen Titel mit der Tab-Taste ansteuern und mit der
  Eingabetaste öffnen
- [ ] Buchakte, Reiter „Ausleiher": das Zeichen neben dem Datum einer überfälligen Ausleihe
- [ ] Inventur: eine unbekannte Nummer scannen („Zu diesem Barcode gibt es kein Exemplar.")
- [ ] Statistiken: „Gesamtbestand" und „aktive Exemplare" zählen bestellte Exemplare nicht
  mehr mit
- [ ] Leserakte einer Lehrkraft mit Dauerleihe über den Browser drucken (Strg+P; einen Knopf
  dafür hat die Akte nicht): Die Ausleih-Quittung nennt in der Zeile „ohne Frist" statt eines
  Datums
- [ ] Bestellwesen in einem kleinen Fenster (1366 × 700): Bedarfsliste und Bestellspalte enden
  am unteren Rand und scrollen in sich; „Bestellung auslösen" bleibt im Bild
- [ ] Signaturen bei 1280 px Breite: Das Regal steht neben der Liste, lange Titel brechen um
- [ ] Medienkatalog → Geräte: ein Gerät mit Zustandsnotiz anlegen; die Notiz steht danach in
  der Liste
- [ ] Leserakte eines Schülers mit offenem Schadensfall, „Ersatzforderung" drucken: Die
  Beträge stehen mit Komma („12,50 EUR")
- [ ] Bestellwesen, „Titel suchen & hinzufügen" bei leerem Warenkorb: Die Trefferliste ist
  ganz zu sehen, mit ihrem unteren Rand; die Linie links der Bestellspalte reicht bis zum
  unteren Rand der Seite

**Erledigen:**

- [x] SonarQube-Scan starten.
- [ ] PR-Pflicht im Regelwerk für `main` entfernen. (7.6)
- [x] Am Testserver gezählt am 08.10.2026: Kein gesperrtes Exemplar trägt „verloren" in der
  Notiz. Mit dieser Notiz als Verlust ausgebucht ist eines, die Nummer 2424.
- [ ] Am Testserver nachsehen, ob das Buch mit der Nummer 2424 („Mathematik heute", Band 6)
  fehlt; es ist seit dem 17.09.2026 als Verlust ausgebucht. Liegt es im Regal: in der Buchakte
  den Status des Exemplars zurückstellen.
- [x] Am Testserver gezählt am 08.10.2026: Jedes Exemplar im Bestand trägt ein Zugangsdatum;
  ohne Datum sind nur die 26 bestellten im Zulauf.
- [ ] Das Blatt mit den zwei Schlüsseln ausfüllen. (9.9)
- [ ] Theke ohne Netz: der Nachweis von Hand im echten Chrome (Stufe 1 und 3), zurückgestellt
  am 24.09.2026. Stufe 2 über die Tür ist seit dem 08.10.2026 belegt. (2.3)

### Termine

- [ ] 19. Oktober 2026: CodeQL wechselt bei GitHub das Abbild; den Lauf danach ansehen. (5.10)
- [ ] Ab dem 28. Oktober 2026: Node 26, nach der Regel „immer die aktive LTS"
  ([PFLEGEKONZEPT.md](PFLEGEKONZEPT.md), Abschnitt 4).

### Später: kleine Fehler und Aufräumarbeit

Ohne feste Reihenfolge. Jede Zeile bündelt, was unter ihrer Nummer in den Einzelheiten steht.
Was davon fertig ist, wird dort gelöscht und fällt aus der Zeile; abgehakt wird sie, wenn unter
der Nummer nichts mehr dazu offen ist.

- [ ] **Jahrgang am Titel (5.5):** „Klasse" und „von … bis" werden eine Angabe, „unbekannt" eine
  eigene (entschieden am 24.09.2026). Der Bauplan in drei Stufen ist seit dem 08.10.2026
  freigegeben; gebaut wird vor der Übernahme am Schulserver, die Maske zuerst als Bild.
- [x] **Buchakte (5.5):** Ein bestelltes Exemplar, das nie eintraf, steht nach „Exemplar löschen"
  nicht mehr im Abgangsbuch; ausgesonderte und bestellte Exemplare heißen dort seit dem
  07.10.2026 „Ausgesondert" und „Bestellt".
- [x] **Nie eingetroffene Exemplare (5.5):** Sie zählen seit dem 07.10.2026 auch in der Zahl
  „aus dem Katalog gelöscht" unter dem Abgangsbuch und in der Statistik nicht mehr mit; die
  Statistik zählt ein bestelltes Exemplar erst mit dem Eintreffen zum Bestand.
- [x] **Druck-Center (5.45):** Ein bestelltes Exemplar ohne Notiz heißt in der Auswahlliste
  seit dem 08.10.2026 „(Bestellt)". Die Vorschau zeichnet seit dem 07.10.2026 nur den ersten
  Bogen.
- [x] **Überläufe (5.45):** Bestellwesen, Signaturen bei 1280 px und ein langer Name in der
  Leserakte sind seit dem 08.10.2026 behoben.
- [ ] **Auskunft (5.19):** Die Protokolleinträge auf dem Blatt nennen in Worten, worum es ging
  (entschieden am 08.10.2026). Der Bau steht aus.
- [x] **Protokoll und Tilgung (5.35):** Am Testserver am 08.10.2026 gezählt: kein alter Eintrag
  zu einem gelöschten Leser, keine Stornierung ohne die Kennung des Lesers, kein Eintrag über
  ein früheres Konto. Zu bereinigen ist nichts.
- [ ] **Gates und Werkzeuge (5.10):** zwei Lücken in Prüfregeln, der erste Lauf von
  `release.yml` auf Ubuntu 26, die Excel-Bibliothek auf einem unveröffentlichten Stand.
- [x] **Ausweis aus der Leserakte (5.5):** Lässt sich das Ausweis-Design nicht laden, drucken
  Leserakte und Leserdatei seit dem 08.10.2026 nicht und sagen es beim Druck.
- [x] **Bestellsuche (5.5):** Antwortet die DNB nicht, steht es seit dem 08.10.2026 oben in der
  Trefferliste, auch ohne Treffer aus dem eigenen Katalog.
- [x] **Zwei Helfer (5.5):** Beträge in Euro und Fehlertexte kommen seit dem 07.10.2026 aus
  ihren Helfern, die Regel „überfällig" ebenso; eine Ratsche hält die ersten beiden fest.
- [x] **Masken, die ihren ganzen Stand zurückschicken (5.5):** Am Titel, am Leser, am Benutzer,
  am Gerät, am Lieferanten und seit dem 08.10.2026 in den Kategorien der Einstellungen schickt
  die Maske nur, was geändert wurde.
- [x] **Bestellung ohne Versandstand (5.5):** Scheitert die Mail an den Lieferanten, steht es an
  der Bestellung und in der Bestellhistorie („Mail nicht versendet"), und die Bestellung lässt
  sich erneut senden (entschieden und gebaut am 07.10.2026, Migration 161).
- [x] **Spur im Protokoll (5.57):** Aussondern über den Status in der Buchakte, die
  Bestandskorrektur und den Abschluss einer Inventur steht seit dem 08.10.2026 mit der Person
  im Protokoll. Die übrigen Türen sind seit dem 08.10.2026 durchgesehen; Schlagworte
  zusammenführen, umleiten und löschen schreiben seitdem einen Eintrag.
- [x] **Beim Umstellen der Farben aufgefallen (5.21):** In der Bestellhistorie bekommt die
  Spalte „Lieferant" seit dem 08.10.2026 die übrige Breite.

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

Offen ist **der Nachweis (2.3):** Stufe 1 und 3 gehören von Hand in den echten Chrome. Stufe 2
über die Tür (`POST /api/action/nachbuchen`) ist am 08.10.2026 am lokalen Stack belegt. Wie sich
die Theke ohne Verbindung verhält, steht in [FACHKONZEPT.md](FACHKONZEPT.md) 18.4 und im
[Handbuch](HANDBUCH.md).

### 2.3 Nachweis am Stack (je Stufe, echter Chrome)

- Stufe 1: DevTools-Drosselung 20 s, Schüler laden, Buch, Escape → IndexedDB trägt den Schüler.
  Offline „zurückgeben" im Profil → Rückgabe. Lehrkraft laden, Buch scannen → Ausleihe auf die
  Lehrkraft. IndexedDB blockiert → „NICHT gespeichert". Zwei Sicherungen einspielen → ein
  Stapel. Zwei parallele Anfragen mit einem Schlüssel gegen `/api/action` → eine Ausleihe.
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
- **Jahrgang am Titel: eine Angabe, „unbekannt" als eigener Wert** (entschieden am 24.09.2026,
  gemessen am 08.10.2026). Der Titel führt zwei Angaben, „Klasse" (`grade_level`) und „von …
  bis" (`jahrgang_von/bis`); die Spanne trägt an jedem Titel ohne Angabe die Vorgabe 5 bis 10.

  **Was die Vorgabe bewirkt:**
  - Die Inventur „Nach Fach / Klasse" (`InventoryStartModal`, `repository.InventurScope`)
    trifft für jede Klasse von 5 bis 10 den ganzen Bestand im Regal und für 11 bis 13 nichts:
    am Testserver je Klasse 34.667 bis 34.757 Exemplare, höchstens 90 davon an Titeln mit
    eigener Spanne. Beim Abschluss stehen die nicht gescannten als Verlust im Bericht
    (`FinishInventurSession`). Eine Inventur mit Fach und Klasse 5 wurde dort am 23.07.2026
    begonnen und nicht abgeschlossen.
  - Der Filter „Jahrgang" der Schulbücher im Portal (`lernmittelFilterSQL`) zeigt für 5 bis 10
    jedes Lernmittel mit der Vorgabe, für 11 bis 13 keines davon.
  - Der Kopf der Buchakte nennt „Jahrgang 5–10" (`BookAkteMeta`).
  - Ein Mehrjahresband mit der Vorgabe bleibt bis zum Ende von Jahrgang 10.
  - Katalogsuche (`trifftJahrgang`) und Schulbuchliste (`jahrgangText`) werten 5 bis 10 als
    „keine Angabe"; eine gepflegte Spanne 5 bis 10 fällt dort heraus.

  **Die Klasse** lesen die Titel-Tabelle, die Klassenzuweisung und die Katalogsuche. Es
  schreiben sie die Maske (Vorgabe 5 bei einem neuen Buch), der Listenimport (Spalte „klasse")
  und der Vorschlag der ISBN-Abfrage, der auch aus „Band 2", „Level 9" und jeder Zahl von 5
  bis 13 im Titel eine Klasse macht (`automatischeKategorisierung`).

  **Testserver (08.10.2026, lesend):** 13.056 von 13.062 Titeln tragen die Vorgabe. 155 tragen
  eine Klasse: 22 die 5, 133 eine von 6 bis 13; 129 dieser 133 bei der Vorgabe, 90 davon
  Lernmittel. In 57 der 133 steht die Zahl im Titel, bei 62 der 129 nennt die Signatur
  dieselbe Zahl; mehrjährige Bände tragen ein einziges Jahr („Natur und Technik - Biologie 7
  - 10": Klasse 7). Woher die Werte stammen, ist nicht belegt.

  **Nach der Littera-Übernahme** (Sicherung von 2010, gezählt mit dem Code der Übernahme):
  7.467 von 10.732 Titeln bekommen eine Spanne, 257 aus der Signatur (alle ein Jahrgang) und
  7.210 aus dem Interessenkreis: 11 bis 13 an 2.982 Titeln, 5 bis 13 an 2.414, 5 bis 10 an
  1.723, 5 bis 6 an 91. 3.265 Titel bekommen keine. Die Klasse steht an den 257 und weicht nie
  von der Spanne ab. Die 1.723 Titel mit 5 bis 10 wären nach der Übernahme von den 3.265 ohne
  Angabe nicht zu unterscheiden, und eine spätere Migration könnte sie nicht mehr trennen. Der
  Umbau gehört deshalb vor die Übernahme am Schulserver.

  **Andere Programme:** Littera führt am Titel keine Klasse; die Schule trug die Stufe als
  Interessenkreis ein (eine Liste, mehrere je Titel: „Sekundarstufe 1", „Sekundarstufe 2",
  „Förderstufe") und bei Lernmitteln in die Signatur („LMF Bio 7"). MARC 21, Feld 521, nennt
  die Klassenstufe als Text, „9-12" oder „7 & up"; ONIX, die Meldeform des Buchhandels, kennt
  „genau", „von", „bis" und „von … bis". Follett Destiny führt ein Auswahlfeld „Interest
  Level" nach Klassenstufen. Koha führt die Zielgruppe als Auswahl aus acht Stufen („Primary",
  „Adolescent" …; MARC 21, Feld 008, Stelle 22) und sucht die Klassenstufe im Text von Feld
  521. Eine einzelne Zahl neben einer Spanne führt keines.

  **Bauplan, freigegeben am 08.10.2026:**
  1. Datenbank: Die Spanne darf fehlen (beide Werte oder keiner; 1 bis 13, „von" nicht über
     „bis"). Die Migration setzt die Vorgabe 5 bis 10 auf „unbekannt" und entfernt die Spalte
     der Klasse. Die fünf Titel mit Klasse und eigener Spanne verlieren nichts, die Klasse
     liegt bei allen in der Spanne. Die Klassen bei der Vorgabe verfallen (129 von 6 bis 13,
     dazu 21 mit der 5, der Vorgabe der Maske): Der Schulserver beginnt mit leerer Datenbank,
     dort gibt es sie nicht.
  2. Server: Übernahme, Katalog-Import, Listenimport (die Spalte „klasse" wird die Spanne N
     bis N) und Maske schreiben nur die Spanne; ohne Angabe bleibt sie leer. Inventur und
     Portal-Filter treffen nur Titel mit Spanne. Der Vorschlag der ISBN-Abfrage liest nur
     noch „Klasse N", „Klassen N–M" und „Jahrgangsstufe N". Der Filter `?gradeLevel=` der
     Titelliste, den keine Seite ruft, entfällt.
  3. Oberfläche: Die Maske verliert das Feld „Klasse"; „von" und „bis" werden Auswahlfelder
     mit „–" für keine Angabe und 5 bis 13 (M3, Menus: „Select menus"; ein Schieberegler
     scheidet aus, M3, Sliders: „Changes made with sliders must take effect immediately").
     Titel-Tabelle und Klassenzuweisung zeigen die Spanne („7", „7–10", „–"); der Kopf der
     Buchakte nennt den Jahrgang nur, wenn er feststeht.

  Je Stufe ein Test, der am alten Stand rot ist, und die volle Suite; danach die Generalprobe
  der Übernahme. Zum Nachzählen am Testserver:

  ```sql
  SELECT jahrgang_von, jahrgang_bis, count(*) FROM buecher_titel GROUP BY 1, 2 ORDER BY 3 DESC;

  SELECT grade_level, jahrgang_von, jahrgang_bis, ist_lernmittel, signatur, titel
  FROM buecher_titel WHERE grade_level BETWEEN 1 AND 13
  ORDER BY ist_lernmittel DESC, signatur NULLS LAST, titel;
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

### 5.10 Gates und Werkzeuge

- Die Schema-Gegenrichtung ist blind für UNIQUE, Teilindizes und RESTRICT.
- Kein Gate gegen unbegrenzte Listen-Endpunkte.
- **Ubuntu 26 als Runner: der erste Lauf von `release.yml`.** Seit dem 08.10.2026 laufen alle
  zehn Jobs der vier Workflows fest auf `ubuntu-26.04`; der Name steht für actionlint in
  `.github/actionlint.yaml`. Die ersten Läufe von `ci.yml`, `security-scan.yml` und
  `docker-publish.yml` sind angesehen (am Stand 3bae78b9 grün bis auf actionlint, das den
  Namen nicht kannte). Offen: `release.yml` läuft erst mit dem nächsten v-Tag; den Lauf dann
  ansehen. Es braucht dort nur `git`, `gh` und `scripts/tag-gate.sh`. Nicht darunter: CodeQL
  läuft in der Standard-Einrichtung von GitHub (Repository-Einstellung, keine Workflow-Datei)
  auf `ubuntu-latest` und wechselt am 19. Oktober 2026 mit; der Hinweis darauf steht an jedem
  CodeQL-Lauf (gesehen am 28.09.2026). Bricht die Analyse dort, wird der CodeQL-Lauf rot.
- **excelize auf einem unveröffentlichten Stand.** Eingesetzt ist seit dem 08.10.2026
  `v2.11.1-0.20260910071107-696050fbf14e`, der Entwicklungsstand der Bibliothek vom
  10.09.2026. Er enthält die Korrekturen zu den neun Meldungen vom 07.10.2026 (GitHub,
  Dependabot Nr. 21 bis 29, CVE-2026-107217 bis CVE-2026-107225: drei „high", sechs „medium");
  eine veröffentlichte Fassung damit gibt es nicht, die jüngste ist v2.11.0 vom 06.07.2026.
  Mit dem Stand ändert sich sonst nur `richardlehane/mscfb` (1.0.7 auf 1.0.8). Belegt: Die
  ganze Go-Suite ist mit ihm grün, und der Absturz auf dem Auslagerungs-Weg, den v2.11.0 noch
  hat, ist weg (`pkg/xlsxgrenze/negativer_sharedstring_test.go`). 38 Excel-Dateien vom
  Entwicklungsrechner, darunter Klassen- und Schülerlisten und eine Medienliste mit 559
  Blättern, liest er wie v2.11.0: je Blatt dieselbe Zahl der Zeilen und Zellen und dieselbe
  Prüfsumme über die Werte, roh und formatiert (verglichen am 08.10.2026). Das Programm liest
  mit der Bibliothek nur, an drei Stellen hinter Anmeldung und Fachrecht
  (`inventur/excel_import.go`, `api/lusd_parser_quelle.go`, `api/littera_import.go`), alle
  durch `xlsxgrenze.MitMappe`; die Schranke dort bleibt als zweite Lage. Offen: auf die
  veröffentlichte Fassung heben, sobald sie erscheint (Dependabot schlägt sie vor). Fällt bis
  dahin an einem der drei Importe etwas auf, zuerst gegen v2.11.0 gegenprüfen. Kategorie B.
- **gosec: acht Regeln global ausgenommen** (gemessen mit v2.29.0 am 28.09.2026, ohne
  `-exclude`): G706 (38 Stellen in 20 Dateien, nachgezählt am 07.10.2026), G704 (6), G703 (5),
  G120 (5), G124 (4), G404 (4), G115 (3), G101 (1); der Grund je Regel steht in
  `.github/workflows/security-scan.yml`.
  Eine neue Stelle dieser Regeln meldet gosec nicht. Abhilfe: je Stelle ein `#nosec` mit Grund,
  dann die Regel aus `-exclude` nehmen — außerhalb von G706 sind es 28 Stellen in 14 Dateien.
  Nur mit Anlass.

### 5.19 Lesepfade gegen die Sicht `schueler` — was offen bleibt

**Protokolleinträge auf dem Blatt:** Das Blatt nennt je Eintrag den Zeitpunkt und die Aktion
in der Schreibweise des Programms („CHECKOUT", „LESER_GESPERRT", „PURGE_STUDENT"), in der
Datensatz-Historie dazu „USER" oder „SYSTEM" (`dsgvoAuditAbschnitt`,
`dsgvoVerwaltungAbschnitt`). Worum es ging, steht nur in den Rohdaten (`details`) der
abgerufenen Auskunft, und die bekommt die Person nicht; das Gate
`TestDsgvoPDF_DrucktJedeAngabeDerAuskunft` führt die vier Stellen als begründete Ausnahme.

Was die Rohdaten neben der Kennung des Lesers tragen (gezählt am 08.10.2026 an
`repository/protokoll_personenbezug.go`, seinem Test und den Einträgen der lokalen Datenbank):

- Angaben zur Person: den Grund einer Sperre, Entsperrung oder Stornierung (Freitext, kann
  andere nennen), die Ausweisnummern beim Zusammenführen, die LUSD-ID, Name und Klasse bei
  einer Änderung der Leserzeile, Betrag und Referenznummer eines Bescheids;
- bei Ausleihe und Rückgabe nur die Kennung des Exemplars, keinen Titel;
- Kennungen des Programms (Exemplar, Forderung, Bescheid, aufgelöste Leserzeile);
- die Kennung des Kontos, das gebucht hat (`bearbeiter_id`, `benutzer_id`).

Aus der Datensatz-Historie liest die Auskunft nur Einträge zur Leserzeile und zu Ausleihen
(`dsgvoQueryAuditEintraege`). Einträge anderer Tabellen, die den Leser nennen, nennt sie
nicht: die Stornierung einer Forderung (trägt die Kennung seit dem 08.10.2026) und die Spur
einer Forderung oder Vormerkung, die mit ihrem Titel gelöscht wurde. Stornierung und
Forderung stehen im Abschnitt „Schadensfälle", solange es die Forderung gibt.

Nachgelesen am 08.10.2026: DSGVO Art. 15 Abs. 3 verlangt eine Kopie der Daten, Abs. 4 nimmt
die Rechte anderer aus, Art. 12 Abs. 1 verlangt eine verständliche Form. EuGH C-487/21
(04.05.2023): Die Kopie ist „eine originalgetreue und verständliche Reproduktion aller dieser
Daten". EuGH C-579/21 (22.06.2023): Zur Auskunft gehören Informationen über Abfragen, die
„sich auf den Zeitpunkt und die Zwecke dieser Vorgänge beziehen"; die Namen der Beschäftigten,
die sie ausgeführt haben, nur, wenn es unerlässlich ist. Im Littera-Handbuch steht keine
Auskunft an Betroffene. Google nennt im Export von „Meine Aktivitäten" je Eintrag das Produkt,
den Zeitpunkt und den Vorgang in Worten („High level summary of the user activity", etwa „Nach
… gesucht"); unter den dort genannten Feldern ist keine Kennung. Die Seite von OCLC zu
BIBLIOTHECAplus und der DSGVO nennt keine Auskunft an Leser; dort wurden die Protokolldateien
gekürzt, damit „dort keine personenbezogenen Daten verbleiben".

**Entschieden am 08.10.2026:** Das Blatt nennt je Eintrag den
Vorgang in Worten („Ausleihe", „Sperre von Hand") und die Angaben zur Person: bei Ausleihe und
Rückgabe Titel und Nummer des Buchs, sonst Grund, Betrag, Referenznummer, Ausweisnummern und
LUSD-ID. Nicht aufs Blatt kommen die Kennungen des Programms und das Konto, das gebucht hat.
Ein Freitext steht da, wie er gespeichert ist, wie heute schon der Sperrgrund und der Grund
einer Stornierung in den Abschnitten darüber; wer das Blatt aushändigt, liest es vorher. Der
Abschnitt nennt dann jeden Eintrag, der den Leser nennt, gleich aus welcher Tabelle. Das Gate
verlangt danach jede dieser Angaben auf dem Blatt und führt nur noch die Kennungen als
Ausnahme.

**Auf einer anderen Anlage vorher zählen:** Vormerkungen und Schadensfälle an einem Kollegen
sehen die Lesepfade gegen die Sicht nicht — die Warteschlange geht über eine solche Vormerkung
hinweg. Am Testserver waren beide Zählungen 0 (zuletzt am 08.10.2026); neue Vormerkungen für
Kollegen lehnt die Tür seit dem 21.09.2026 ab.

```
SELECT count(*) FROM vormerkungen v JOIN leser l ON l.id = v.schueler_id WHERE l.art <> 'schueler';
SELECT count(*) FROM schadensfaelle sf JOIN leser l ON l.id = sf.schueler_id WHERE l.art <> 'schueler';
```

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

### 5.45 Listen in einem Kasten mit eigenem Scrollen

Eine Liste zeigt alle Zeilen, gescrollt wird der Bereich der Seite
(`e2e/scrollbereiche.spec.js`); so stehen die Ausleihliste der Leserakte, die Positionen im
Wareneingang und die Exemplare in der Maske „Buch bearbeiten". Kategorie B. Offen:

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
- **Leserakte, Autor und Nummer des Exemplars:** Der Autor steht nur in der Sprechblase am
  Titel, die Nummer in Fenstern bis rund 1580 px ebenfalls (darüber hat sie ihre Spalte;
  gemessen bei ausgeklappter Seitenleiste). Die Sprechblase erscheint beim Zeigen mit der
  Maus; an einem Tablet ohne Maus sind beide Angaben in der Akte nicht zu sehen.
  Vorleseprogramme bekommen sie als unsichtbaren Text.
- **Leserakte, doppelte Beschriftung:** Unter dem Reiter „Stammdaten & Adresse" steht dieselbe
  Überschrift noch einmal; im Reiter „Gebühren & Schäden" heißt die Liste seit dem 01.10.2026
  „Forderungen".

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
Abschnitt 2e — dabei die Befehle aus 2a erproben, die bisher nur am Text geprüft sind. Sinnvoll
nach S3 oder am Schulserver. Machen soll sie die Vertretung allein mit dem Pflegekonzept (9.9);
sie wartet also auf den Schulserver und auf die benannte Vertretung.

Der Totalverlust des Servers steht seit dem 08.10.2026 in Abschnitt 2f, durchgespielt an einem
frischen Klon auf dem Entwicklungsrechner. Er setzt eine Sicherung voraus, die nicht auf dem
Server lag (7.3).

Für die Probe von 2a: Der Abschnitt setzt `pg_dump`, `dropdb`, `createdb`, `psql` und ein
gebautes `restore-backup` auf dem Rechner voraus, an dem die Befehle laufen. Ob der Zielserver
sie hat, zeigt die Probe; die Form über die Container (`docker compose exec`, die Datei über
die Standardeingabe) ist in 2f erprobt.

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
