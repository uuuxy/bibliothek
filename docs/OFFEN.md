# Offene Arbeit

Stand: 13.10.2026

**Der Fahrplan.** Oben steht, was als Nächstes getan wird, in der Reihenfolge der Arbeit: je
Schritt eine Zeile mit Kästchen. Die Nummer in Klammern führt zu den Einzelheiten weiter unten.
Einen zweiten Ort für Offenes gibt es nicht; andere Dokumente erklären (Konzept, Anleitung, der
Katalog der Bugklassen in [sweeps.md](sweeps.md)), führen aber keine eigene Offen-Liste. Wie die
Liste geführt wird, steht am Ende.

---

## Fahrplan

### Etappe 1: vor dem Echtstart

Der Echtbetrieb beginnt am Schulserver mit einer leeren Datenbank und der Littera-Übernahme
(entschieden am 28.09.2026). Steht an einem Schritt R7, R8 oder R9, verkleinert er das Risiko
dieser Nummer aus [ARCHITEKTUR.md](ARCHITEKTUR.md) 11.1.

- [ ] Generalprobe der Übernahme mit der Sicherung von 2026, sobald sie sich öffnen lässt:
  Ausweisnummern, offene Ausleihen, Standorte, Verweise der Schlagworte, Sperren und Salden.
  Danach entscheidet sich, ob die Theke die kurze Nummer der Littera-Etiketten annehmen muss
  ([ARCHITEKTUR.md](ARCHITEKTUR.md) 11.5). (7.2, 4.20)
- [ ] `update.sh` für den Schulserver: nur Releases, Images frisch. (5.31)
- [ ] Eingang des Servers: Von außen ist nur die Seite der Lieferanten erreichbar. (4.23)
- [ ] Am Schulserver einrichten: Sicherung außer Haus (7.3, R8), Uptime-Signal (7.5, R9),
  Eigentumsvermerk der Etiketten (4.24), Frist bis zum Sperrbildschirm (9.9). Danach nachsehen:
  Admin-Konten, Verbindung zur DNB (7.8).
- [ ] Abnahmen mit echten Daten. (7.7)
- [ ] Wiederherstellung an einem fremden Ziel proben, allein mit dem Pflegekonzept. (7.4, R7)

### Bei dir

**Entscheiden:**

- [x] Türen in Themenordner: Der Plan ist seit dem 12.10.2026 freigegeben. Geteilt wird nur
  `api/`, je Menüpunkt der Oberfläche ein Ordner; die Abfragen bleiben in der einen
  Datenbankschicht. (5.65)
- [x] Türen in Themenordner, dritte Freigabe: Stufe 4 ist seit dem 12.10.2026 freigegeben,
  je Thema ein Schritt, zuerst die vier Themen mit dem wenigsten im Weg (Inventur, Benutzer,
  Portal, System). (5.65)
- [ ] Sicherung außer Haus: Alte Sicherungen löscht das Programm nur auf dem Server, am
  Speicher außer Haus nie. Vorschlag: Es löscht dort nach derselben Regel (die jüngsten 14
  Nächte, dazu je Woche eine für 12 Wochen); eine Löschregel, die jemand am Speicher von Hand
  einstellt, braucht es dann nicht. (7.3, R8)
- [x] Vier zugelieferte Pull Requests (726 bis 729): geschlossen am 12.10.2026, nach dem
  Vorschlag. Die Begründung steht je am Pull Request. Soll es den Knopf zum Leeren der Suche
  geben, entsteht er neu nach der Seite von Material 3.
- [x] Drei Updates von Dependabot (731, 732, 733): übernommen am 12.10.2026, nach dem
  Vorschlag; was am zusammengeführten Stand geprüft ist, steht in den drei Merge-Commits. Der
  Schritt mit upload-artifact läuft nur nach einem roten Browser-Test und ist ungeprüft. Das
  vierte Update (730, zehn Pakete der Oberfläche) ist rot und steht unter 5.10.
- [x] gosec vor dem Push: Der Hook fährt gosec seit dem 09.10.2026 mit, über dasselbe Skript
  wie der Sicherheits-Lauf (`scripts/gosec-gate.sh`). Gebaut nach dem Vorschlag; zurücknehmen
  lässt es sich mit dem Rückbau dieses einen Commits.
- [x] Ändern eines Lesers: `repository/` nennt seit dem 09.10.2026 die Spalten, die sich
  ändern lassen (`repository.LeserAenderung`), die Tür reicht nur Werte. Gebaut nach dem
  Vorschlag; die Anweisung ist dieselbe wie vorher.

**Fertig gebaut — von dir am Testserver anzusehen,** nach `git pull` und `./update.sh` (7.10):

- [ ] Portal: „Problem melden" unter dem Suchfeld
- [ ] Buchakte: „Standort ändern", auch mit dem Handscanner
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
- [ ] Leserakte eines Schülers mit Ausleihen, „DSGVO-Auskunft": Die Abschnitte 8 und 9 nennen
  jeden Vorgang in Worten, bei Ausleihe und Rückgabe mit Buchtitel und Nummer; die Beträge
  stehen mit Komma
- [ ] Jahrgang am Titel: „Neues Buch" öffnen, die zwei Felder zum Jahrgang stehen leer, ein
  Feld „Klasse" gibt es nicht; eine 7 in „von" steht sofort auch in „bis". Die Titel-Verwaltung
  führt die Spalte „Jahrgang". Im Portal unter „Schulbücher" zeigt der Filter „Jahrgang" nur
  Bücher, an denen ein Jahrgang eingetragen ist; die laufende Inventur nach Klasse erwartet
  nur noch solche Bücher (7.10)
- [ ] Bestellwesen, einen Titel aus der Suche wählen: Das Fenster ist weiß mit Rahmen; der
  Knopf unter den Schlagworten heißt „Vorschläge aus der DNB holen" und steht wie „Abbrechen"
  in Blau. Knöpfe ohne Fläche sind im ganzen Programm blau (etwa „Abbrechen" in den Dialogen,
  „Cover ändern" am Buch); Knöpfe, die nur ein Symbol zeigen, bleiben grau
- [ ] Fünf Kästen sind weiß mit Rahmen statt grau: in der Buchakte der Status eines Exemplars
  (Reiter „Exemplare", „Status ändern"), unter Medienkatalog → Geräte „Gerät anlegen", in der
  Leserakte „Doppelter Datensatz?" und bei einer Lehrkraft der Hinweis in den Stammdaten, in
  „Neuen Leser anlegen" der Hinweis bei einer Lehrkraft
- [ ] Jahrgang aus der ISBN-Abfrage: „Neues Buch" öffnen und die ISBN 9783141096835 eingeben
  („EinFach Deutsch Unterrichtsmodelle … Klassen 8 - 10"). Erwartet: „von" zeigt 8, „bis"
  zeigt 10. Bei einem Buch für einen Jahrgang (9783060623198, „… 5. Schuljahr") zeigen beide 5
- [ ] Mahnwesen, „Mahnbriefe drucken" für einen Schüler mit so vielen Büchern über der Frist,
  dass die Tabelle nicht auf die erste Seite passt (mit der Standardvorlage ab dem neunten): Die
  Tabelle läuft auf einer Folgeseite weiter, mit „Fortsetzung: Vorname Nachname" und den
  Spaltenköpfen; keine Zeile ist über zwei Seiten verteilt
- [ ] Mahnwesen, Mahnliste an die Klassenleitung (Versand an eine eigene Adresse, Anhang
  öffnen): Das Cover steht im Seitenverhältnis des Buchs; ein langer Titel endet mit „…" vor
  der Spalte „Autor"; bei mehr als zehn Büchern eines Schülers folgt eine Seite mit
  „Fortsetzung: Name, Klasse"
- [ ] Bestellwesen, Anschreiben im Anhang der Bestellmail: Der Name der Schule steht am selben
  linken Rand wie der Brief; ab der zweiten Seite stehen die Spaltenköpfe über den Positionen

**Erledigen:**

- [ ] PR-Pflicht im Regelwerk für `main` entfernen. (7.6)
- [ ] Am Testserver nachsehen, ob das Buch mit der Nummer 2424 („Mathematik heute", Band 6)
  fehlt; es ist seit dem 17.09.2026 als Verlust ausgebucht. Liegt es im Regal: in der Buchakte
  den Status des Exemplars zurückstellen.
- [ ] Das Blatt mit den zwei Schlüsseln ausfüllen. (9.9, R7)
- [ ] Theke ohne Netz: der Nachweis von Hand im echten Chrome (Stufe 1 und 3), zurückgestellt
  am 24.09.2026. Stufe 2 über die Tür ist seit dem 08.10.2026 belegt. (2.3)

### Termine

- [ ] 19. Oktober 2026: CodeQL wechselt bei GitHub das Abbild; den Lauf danach ansehen. (5.10)
- [ ] Ab dem 28. Oktober 2026: Node 26, nach der Regel „immer die aktive LTS"
  ([PFLEGEKONZEPT.md](PFLEGEKONZEPT.md), Abschnitt 4).

### Umbau: die Türen in Themenordner

Entschieden am 12.10.2026, in Stufen (5.65). Nach jeder Stufe ist `main` grün, und der Umbau
lässt sich dort anhalten.

- [x] Stufe 0, Schutz: Die drei Prüfungen, die `api/` ohne Unterordner lasen, lesen sie mit.
  Eine neue Prüfung hält die Dateiverweise gegen die Dateien des Repositorys. Die Zahl der
  Tests und die Routen stehen als Vergleichsstand fest.
- [x] Stufe 1, Tests: Die Helfer, die Tests mehrerer Themen brauchen und selbst nichts aus
  `api/`, stehen in Paketen für Prüfhilfen.
- [x] Stufe 2, Kern: Was jede Tür braucht (Antwort schreiben, Rechte, Protokoll, Uhr), steht
  im Paket `api/tuer`, der Mailversand in `mailservice/`. Es war die erste Änderung am
  Programm; das Verhalten ist dasselbe.
- [x] Stufe 3, das erste Thema: Die Türen des LMF-Plans stehen in `api/lernmittel` (5 Dateien,
  9 Testdateien), umgezogen mit einem Werkzeug. Ein Test über den ganzen Router steht im
  Ordner seines Themas, als eigenes Testpaket. Der Aufwand je Thema ist gemessen (5.65).
- [x] Stufe 4, die übrigen elf Themen, je Thema ein Schritt: Inventur
  (`api/inventurlauf`), Benutzer (`api/benutzer`), Portal (`api/portal`), System
  (`api/system`), Mahnwesen (`api/mahnwesen`), Schaden (`api/schaden`), Katalog
  (`api/katalog`), Druck (`api/druck`), Bestellung (`api/bestellung`), Leser (`api/leser`)
  und Theke (`api/theke`). Alle zwölf Themen stehen in eigenen Paketen.
- [ ] Stufe 5, Abschluss: Eine Ratsche hält, dass kein Thema ein anderes einbindet. Tests,
  die nichts aus `api/` prüfen, ziehen zu ihrem Paket. Die Dokumente nennen die neue
  Gliederung. Eine Messung an `repository/` entscheidet, ob die Datenbankschicht folgt.

### Später: kleine Fehler und Aufräumarbeit

Ohne feste Reihenfolge. Jede Zeile bündelt, was unter ihrer Nummer in den Einzelheiten steht.
Was davon fertig ist, wird dort gelöscht und fällt aus der Zeile; abgehakt wird sie, wenn unter
der Nummer nichts mehr dazu offen ist.

- [ ] **Gates und Werkzeuge (5.10):** die Bestände zweier Ratschen sind nicht befragt (Schema:
  was die Datenbank ablehnt; Listen: was sie begrenzt), die Form-Ratsche sieht umrandete
  Karten nicht (drei mit runderer Ecke), der erste Lauf von `release.yml` auf Ubuntu 26, die
  Excel-Bibliothek und gosec auf einem unveröffentlichten Stand, drei falsche Warnungen des
  Linters im Sammel-Update der Oberfläche.
- [ ] **Schichtung des Backends (5.62):** die Regeln in Dateien, die auch eine Tür tragen, je
  Datei lesen und trennen; die Zuordnung der zwei Importe zusammenlegen.
- [ ] **Etiketten kürzen nach Zeichenzahl (5.63):** an neun Stellen messen, ob ein Text über
  den Rand des Etiketts läuft.

Was nur mit Anlass gebaut wird, steht nicht hier, sondern in
[ARCHITEKTUR.md](ARCHITEKTUR.md) 11.5 „Bekannte Grenzen".

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
Anmeldung: Der Browser schickt das Sitzungs-Cookie in der Vorgabe nur über HTTPS mit
(`ermittleCookieSecure` in `main.go`; ein ausdrückliches `COOKIE_SECURE=false` hebt das auf und
warnt beim Start). Mit einem Zertifikat, dem die Browser nicht vertrauen, stünde an jedem Gerät eine
Warnung, und die Theke ließe sich bei einem Netzausfall nicht neu laden (Service Worker,
[Architektur 8.8](ARCHITEKTUR.md#88-echtzeit-und-offline), „Offline"). Das Uptime-Signal von außen (7.5)
ruft `/health` ab.

**Was die Seite von außen braucht** (am Code nachgesehen am 28.09.2026): den Pfad
`/bestellung/<token>` mit den Dateien der Oberfläche, `/api/public/bestellung/…` (Abruf,
Etiketten, Bestätigung) und für das Uptime-Signal `/health`, das nur `healthy` oder `unhealthy`
meldet. Mehr nicht: Das CSRF-Cookie für die Bestätigung setzt schon der erste Abruf
(`refreshCSRFCookie` in `api/csrf.go`), der Aufruf von `/api/auth/me` beim Laden
(`restoreSession`) darf scheitern, und die Seite zeigt keine Personendaten — Titel, ISBN, Menge,
Schule, Lieferant, Topf, Kundennummer (`api/bestellung/bestellbestaetigung_public.go`).

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

### 5.10 Gates und Werkzeuge

- **Der Bestand der Schema-Ratsche ist nicht befragt.** Seit dem 08.10.2026 friert
  `repository/schema_gegenrichtung_sperren_pg_test.go` ein, was die Datenbank ablehnt oder
  weiterträgt: 12 Fremdschlüssel, an denen ein Löschen scheitert, 6, die ein Umbenennen
  weitertragen (Klasse, Sachgruppe), 35 Eindeutigkeitsregeln. Eine neue Regel macht den Test
  rot. Zu den vorhandenen steht erst an einer die Antwort (Sachgruppe löschen: 409 mit Satz).
  Nächster Schritt: je Eintrag den Schreiber suchen, der dagegen laufen kann, und nachstellen,
  was er bekommt; ohne eigene Behandlung ist es der neutrale Satz von `apierrors`. Zuerst die
  elf übrigen Löschsperren (Leser, Exemplar, Gerät, Klasse). Kategorie B.
- **87 Listen ohne Antwort.** Seit dem 08.10.2026 ruft `api/listen_grenze_gate_pg_test.go` jede
  GET-Route auf und verlangt zu jeder Liste in der Antwort, was sie begrenzt; eine neue Liste
  ohne Antwort macht den Test rot. Gefunden sind 92 Listen in 70 Routen. Fünf tragen ihre
  Antwort (beide Protokolle und der Papierkorb mit je 1000 Zeilen, der Katalog mit der
  Kappung bei 50000, die Regaladressen); 87 stehen als Bestand im Gate. Nächster Schritt: je
  Liste die Grenze am Code nachlesen und eintragen, zuerst die Listen, deren Tabelle mit der
  Zeit wächst (Bestellhistorie, Vormerkungen, Bescheide, Nachbuch-Meldungen, Inventuren,
  Historie eines Titels). Fehlt eine Grenze: Obergrenze in der Abfrage, Index auf der
  Sortierspalte, die Kappung in der Oberfläche ansagen. Kategorie B.
- **Die Form-Ratsche sieht eine umrandete Karte nicht.** `frontend-hygiene-layout.test.js`
  nennt 16 px Ecke auf einer Karte falsch, erkennt eine Karte aber nur an weißer Füllung
  (`bg-white`, `bg-surface-container-lowest`). Eine umrandete Karte ohne Füllung oder in
  `bg-surface` fällt durch. Drei Stellen tragen so `rounded-2xl`: in der Buchakte das Formular
  „Schüler vormerken" (`BookVormerkungenTab.svelte`), im Mahnwesen der Kasten „Keine offenen
  Schadensersatz-Fälle" (`BescheideTabelle.svelte`), im Ausweis-Designer die Auswahl
  (`designer/ToolbarAuswahl.svelte`). Nächster Schritt: die Erkennung auf den ganzen Rahmen
  erweitern und die drei auf `rounded-xl` stellen; der Sperrbildschirm trägt als Dialog 28 px
  und bleibt. Sichtbar sind je 4 px an der Ecke. Kategorie B.
- **Ubuntu 26 als Runner: der erste Lauf von `release.yml`.** Alle zehn Jobs der vier
  Workflows laufen auf `ubuntu-26.04`; `ci.yml`, `security-scan.yml` und `docker-publish.yml`
  sind dort grün (zuletzt am Stand 9687c973). `release.yml` läuft erst mit dem nächsten v-Tag;
  den Lauf dann ansehen. Es braucht dort nur `git`, `gh` und `scripts/tag-gate.sh`. Nicht
  darunter: CodeQL läuft in der Standard-Einrichtung von GitHub (Repository-Einstellung, keine
  Workflow-Datei) auf `ubuntu-latest` und wechselt am 19. Oktober 2026 mit; der Hinweis darauf
  steht an jedem CodeQL-Lauf (gesehen am 28.09.2026). Bricht die Analyse dort, wird der
  CodeQL-Lauf rot.
- **excelize auf einem unveröffentlichten Stand.** Eingesetzt ist seit dem 08.10.2026 abends
  `v2.11.1-0.20261003002531-6258dcebc4e2`, der Entwicklungsstand der Bibliothek vom
  03.10.2026. Er enthält die Korrekturen zu den neun Meldungen vom 07.10.2026 (CVE-2026-107217
  bis CVE-2026-107225) und zu den sechs vom 08.10.2026 (CVE-2026-107211 bis CVE-2026-107216,
  alle „high"); eine veröffentlichte Fassung damit gibt es nicht, die jüngste ist v2.11.0 vom
  06.07.2026. Mit dem Stand ändert sich sonst nur `xuri/efp` (0.0.1 auf 0.0.2). Belegt: Die
  ganze Go-Suite ist mit ihm grün. 38 Excel-Dateien vom Entwicklungsrechner, darunter Klassen-
  und Schülerlisten und eine Medienliste mit 559 Blättern, liest er wie der Stand vom
  10.09.2026: je Blatt dieselbe Zahl der Zeilen und Zellen und dieselbe Prüfsumme über die
  Werte, roh und formatiert (1.176 Blätter, 72.836 Zeilen, verglichen am 08.10.2026). Eine der
  sechs Meldungen trifft den eigenen Lesepfad: Eine Zeilennummer jenseits der Blattgrenze ließ
  `GetRows` jede fehlende Zeile abzählen, nach der Meldung bis zu elf Tage lang
  (CVE-2026-107212; `pkg/xlsxgrenze/zeilennummer_test.go`, am Stand davor rot). Das Programm
  liest mit der Bibliothek nur, an drei Stellen hinter Anmeldung und Fachrecht
  (`inventur/excel_import.go`, `internal/lusd/quelle.go`, `api/system/littera_import.go`), alle
  durch `xlsxgrenze.MitMappe`; die Schranke dort bleibt als zweite Lage. Offen: auf die
  veröffentlichte Fassung heben, sobald sie erscheint (Dependabot schlägt sie vor). Kommt
  vorher eine weitere Meldung, wird der Sicherheits-Scan rot; der Handgriff steht in
  [PFLEGEKONZEPT.md](PFLEGEKONZEPT.md), Abschnitt 5. Fällt an einem der drei Importe etwas
  auf, zuerst gegen v2.11.0 gegenprüfen. Kategorie B.
- **gosec auf einem unveröffentlichten Stand.** Der Sicherheits-Prüflauf und der Hook vor dem
  Push bauen seit dem 09.10.2026 gosec vom Entwicklungsstand des 05.10.2026
  (`v2.29.1-0.20261005092323-d2b649ec0182`, mit `golang.org/x/tools` 0.51); die Fassung steht
  in `scripts/gosec-gate.sh`. Die jüngste
  veröffentlichte Fassung 2.29.0 liest die Paketdaten von Go 1.27.2 nicht. Am Stand des
  Tages meldet der neue Stand mit denselben Ausnahmen nichts (489 Dateien, 61 Vermerke
  `#nosec`). Offen: auf die nächste veröffentlichte Fassung heben, sobald sie erscheint; sie
  kann neue Regeln mitbringen, deshalb mit eigenem Commit. Kategorie B.
- **Das Sammel-Update der Oberfläche vom 12.10.2026 ist rot** (Pull Request 730, zehn Pakete).
  Der Linter meldet nach dem Update drei gebundene Eigenschaften (`$bindable()`) als „assigned
  a value but only used as a type" (`@typescript-eslint/no-unused-vars`), und er läuft mit
  `--max-warnings 0`: `queryVal` in `CameraScanner.svelte`, `element` in `ui/Suchpille.svelte`
  und `showWebcam` in `StudentProfileCard.svelte`. Jedes der drei Bauteile setzt den Wert nur,
  und er geht über `bind:` an den Aufrufer; die Meldung ist falsch. Nachgestellt am 12.10.2026
  in einem eigenen Arbeitsverzeichnis mit `npm ci`: Sie kommt mit typescript-eslint 8.71.1 und
  bleibt mit 8.71.0 aus; eslint, eslint-plugin-svelte und svelte ändern daran nichts. Bei den
  Herstellern ist es als Fehler dieser Fassung gemeldet (typescript-eslint, Meldung 12980,
  Korrektur für die nächste Fassung zugesagt; eslint-plugin-svelte, Meldung 1607, offen).
  Prettier, svelte-check, Vitest und der Bau sind mit dem Update grün, die Browser-Tests liefen
  im Pull Request durch. Nächster Schritt: das Sammel-Update der folgenden Woche ansehen
  (Dependabot prüft montags). Meldet der Linter dann weiter, typescript-eslint in
  `.github/dependabot.yml` zurückhalten, bis die Korrektur erschienen ist; die drei Stellen
  bleiben, wie sie sind. Kategorie B.

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

### 5.62 Schichtung des Backends: `api/` trägt Regeln und SQL

**Entschieden am 09.10.2026: wird jetzt abgebaut,** vor dem Echtstart und in einzelnen
Schritten; bis dahin galt „beim fachlichen Anfassen einer Datei". Das Risiko steht in
[ARCHITEKTUR.md](ARCHITEKTUR.md) 11.1 unter R4, die Messwerte vom 09.10.2026 ebenfalls.

- **Die Bremse steht.** `schichtung_ratsche_test.go` weist jede SQL-Anweisung außerhalb der
  Datenbankschicht ab und führt die Dateien, die noch welche tragen, als Bestand, der nur
  sinken kann; `api/schichtung_test.go` führt die Dateien von `api/` ohne Tür als Bestand, der
  nur kleiner werden kann. Stand am 09.10.2026: keine Datei von `api/` mit SQL (am Anfang 48
  mit 177 Anweisungen), 12 Dateien ohne Tür (am Anfang 44); `api/` hat 21.319 Zeilen in 136
  Dateien (am Anfang 30.785 in 168).
- **Umzug je Thema.** Was keine Tür ist, zieht in ein eigenes Paket. Der LUSD-Import steht in
  `internal/lusd` (Tür in `api/leser/lusd.go`, Anweisungen in `repository/lusd_import.go`), die
  Selbstprüfung und die Regel zu den Start-Geheimnissen in `internal/bereitschaft`, die
  Auskunft nach Art. 15 DSGVO mit ihrem Blatt in `internal/auskunft` (Türen in
  `api/leser/dsgvo_auskunft.go` und `api/leser/dsgvo_pdf.go`), der
  Strichcode in `pkg/strichcode`, die Arten eines Lesers in `pkg/leserart`, die Töpfe einer Bestellung mit ihren Texten in
  `pkg/mitteltopf`, ihr Bestätigungs-Link in `pkg/bestelllink`, die Zuordnung der
  Kopfzeile einer Importdatei beim Importer, das Anlegen einer Bestellung, der Text der
  Bestellmail, die Kopplung der Lernmittel-Fristen an den LMF-Plan und die Vorgabe für einen
  neuen Plan in `internal/service`; in `pdf/` stehen
  der Bescheid (`bescheid.go`), die Etiketten mit ihren Bogenformaten (`etikett_*.go`), die
  Bestandsbücher (`abgangsbuch.go`, `zugangsbuch.go`), die Mahnliste (`mahnliste.go`), der
  Mahnbrief (`mahnbrief.go`) und das Bestellanschreiben (`bestellanschreiben.go`). Kein
  PDF-Erzeuger steht mehr in `api/`, und keine Datei ohne Tür trägt dort noch Regeln. Was ein Erzeuger aus `repository/`
  liest, bekommt er in `pdf/` als eigenen Typ; die Tür füllt ihn (ARCHITEKTUR 5.2.2; dort
  steht auch, warum die Auskunft ein eigenes Paket hat). Je Thema ein Commit; die
  Tests der Türen bleiben stehen und belegen, dass sich nichts ändert. Die Anlagen der Bestellmail
  (`bestellung/bestellmail_anhaenge.go`, bisher `pdf_service.go`) sind kein Erzeuger: Die Datei stellt sie
  zusammen und verschickt die Mail; sie bleibt als Helfer bei ihren Türen.
  Die Tests an der Datenbank bleiben in `api/`, weil ihre Helfer dort liegen; was sie aus einem
  umgezogenen Paket brauchen, ist dort sichtbar gemacht. Die Regeln für Tür und Abfrage stehen in
  [ARCHITEKTUR.md](ARCHITEKTUR.md) 5.2.2, der Handgriff zum Umzug einer Anweisung in 8.14.
- **Was ohne Tür bleibt.** Die 24 Dateien des Bestands sind Typen einer Anfrage oder
  Antwort, Füll-Funktionen und Helfer mehrerer Türen (1.167 Zeilen, gezählt am 13.10.2026).
  Zwei stehen in `api/`: `action_types.go` und `schueler_kiosk.go`. Drei stehen seit dem
  12.10.2026 in `api/tuer`, dem Paket mit dem, was
  jede Tür braucht: `constants.go`, `verwaltung_protokoll.go` und `tuer.go` (der Typ `Basis`
  und die Uhr). 19 stehen in den Paketen der umgezogenen Themen: Mit ihrem Thema
  umgezogen sind `lernmittel/lmf_plan_live.go`, `system/betriebsbereitschaft_alarm.go`,
  `mahnwesen/abgaenger_fenster.go` (die Grenzen der Abgänger-Saison),
  `mahnwesen/mahnwesen_mail.go`, `schaden/bescheid_absender.go`,
  `bestellung/bestellmail_anhaenge.go` (stellt die Anlagen der Bestellmail zusammen und
  verschickt sie), `bestellung/bestellmail_versand.go` (trägt auch die Rückmeldung nach
  dem Versand) und `leser/student_klasse_regel.go` (prüft eine Eingabe der zwei Türen zum
  Anlegen und Ändern und trägt deren Meldung; keine andere Schicht ruft sie), dazu kommt je
  Thema die
  Datei mit seinem Typ (elf; mit der Theke eine mehr). Der
  Bestand fällt damit nicht auf null. Die sechs Dateien mit Regeln sind am 09.10.2026
  ausgezogen; wohin, steht oben und in ARCHITEKTUR 5.2.2.
- **Was der Bestand nicht sieht.** Eine Datei bindet `net/http` nur für Statuswerte ein und
  trägt keine Tür: `theke/nachbuchen_schluessel.go` (148 Zeilen); der Mailversand, die zweite, steht
  seit dem 12.10.2026 in `mailservice/mail_sender.go`. Und
  Regeln in einer Datei, die auch eine Tür trägt: Die meisten Zeilen in Funktionen ohne einen
  Typ aus `net/http` tragen `bestellung/bestellbericht_handler.go` (367),
  `lernmittel/lmf_plan.go` (260), `schaden/bescheid_handler.go` (211; die Rechnung des
  Ersatzwerts steht seit dem 12.10.2026 in `internal/service`), `leser/dsgvo_auskunft.go` (193),
  `system/betriebsbereitschaft_handler.go` (175) und `mahnwesen/graduates_mail.go` (157). Die Messung und ihre
  Grenze stehen in ARCHITEKTUR 11.1 unter R4. Nächster Schritt: diese Dateien der Reihe nach
  lesen und trennen, was Regel und was Füll-Funktion ist.
- **Die Zuordnung der Importe steht zweimal.** Katalog-Import und Bestands-Import laden den
  Titelbestand mit einer Funktion (`repository.LadeTitelBestand`), fragen ihn aber je für sich:
  zuerst die ISBN, sonst der Titeltext (`queueTitelUpsert` in `repository/book_inventory.go`,
  `matchTitelID` in `internal/service/import_dynamic.go`). Beide Fassungen antworten gleich.
  Nächster Schritt: ein Typ für den Titelbestand mit der Frage als Methode, den beide rufen.
- **Die Türen selbst in Themenordner zu teilen:** entschieden am 12.10.2026, in Stufen. Messung
  und Plan stehen in 5.65.

**Die Abfragen außerhalb der Datenbankschicht sind umgezogen** (seit dem 10.10.2026). Zur
Datenbankschicht zählen `repository/`, `inventur/` mit eigener Schicht
([ARCHITEKTUR.md](ARCHITEKTUR.md) 5.2.3), `db/`, der Schreiber der Littera-Übernahme und die
Einmal-Werkzeuge unter `cmd/`; die Liste mit dem Grund je Paket steht in
`schichtung_ratsche_test.go`. Außerhalb davon tragen noch die zwei Dateien der Restore-Probe
in `jobs/` Anweisungen (am Anfang 28 Dateien). Die zwei bleiben: Sie legen eine
Wegwerf-Datenbank an, entfernen sie und zählen darin die Tabellen, und keine der Anweisungen
trifft eine Tabelle der Anwendung. Die Ratsche weist jede weitere ab.

**Nächster Schritt: die Regeln in den Dateien mit Tür** (oben, „Was der Bestand nicht sieht").
Eine Regel in einer Datei mit Tür erschwert das Lesen, führt aber zu keinem zweiten Weg zum
selben Zustand; deshalb kamen die Abfragen zuerst.

### 5.63 Etiketten kürzen nach Zeichenzahl

Die Listen und Briefe mit gofpdf kürzen den Text einer Zelle seit dem 09.10.2026 auf ihre
gedruckte Breite (`pdfzeichen.KuerzeAufZelle`; die Fehlerart steht in [sweeps.md](sweeps.md),
„Gedruckter Text nach Zeichenzahl gekürzt"). Nach Zeichen gekürzt lief dort ein gewöhnlicher
langer Titel bis zu 6 mm, ein Titel in Großbuchstaben bis zu 33 mm über die Nachbarzelle.

Nach Zeichen kürzen weiter die Etiketten (gezählt am 09.10.2026): `pdf/etikett_buch.go` an fünf
Stellen, `pdf/etikett_lernmittel.go` an vier. Zu tun je Stelle: mit ihrer Schrift und der Breite
des Etiketts messen, ob die zugelassene Zahl breiter Buchstaben über den Rand läuft, und dann
auf die Breite kürzen. Die Etiketten sind im Aussehen abgenommen; mit der Umstellung ändert sich,
wo ein langer Titel endet. Die Zeile mit den Auflagen im Schulbuch-Export
(`inventur/lernmittel_pdf.go`) kürzt ebenfalls nach Zeichen; sie bricht über drei Zeilen um und
braucht eine andere Messung.

### 5.65 Die Türen in Themenordner

**Entschieden am 12.10.2026: `api/` wird nach Themen geteilt, in Stufen.** Geteilt wird die
Schicht der Türen, je Menüpunkt der Oberfläche ein Paket unter `api/`. `repository/`,
`internal/service` und `pdf/` bleiben, was sie sind ([ARCHITEKTUR.md](ARCHITEKTUR.md) 5.2.2).
Der Maßstab steht in Googles Regeln für Go, Abschnitt
[Package size](https://google.github.io/styleguide/go/best-practices#package-size): In einem
Paket bleibt, was eng aneinander hängt; was für sich steht, bekommt ein eigenes. Der Plan ist
seit dem 12.10.2026 freigegeben; die Stufen stehen im Fahrplan, Stufe 0 bis 3 sind gebaut,
von Stufe 4 zehn der elf Themen.

- **Gemessen am 12.10.2026** nach Stufe 2 (Stand b3440e46), mit Typprüfung (`go/types`) über
  die 131 Produktiv- und 322 Testdateien von `api/`: Jede Datei ist einem Thema zugeordnet,
  gezählt ist jeder Name, den eine Datei aus einer anderen braucht, bei Tests auch über ihre
  Helfer.
- **Die Themen stehen für sich.** Zwischen zwölf Themen hängen neun Namen an drei Stellen. Die
  Bestellung braucht fünf Namen rund um das Buchetikett aus dem Druck (`BarcodeLabelDetail`,
  `EtikettKopf`, `etikettKopf`, `buchEtiketten`, `etikettKopfAus`), der Druck einen aus der
  Bestellung (`etikettenAus`): ein Kreis (seit dem 12.10.2026 stehen die sechs in
  `internal/buchetikett`). Der Katalog braucht zwei aus dem Schaden
  (`preisquelle`, `ersatzwertVorschlagAus`; seit dem 12.10.2026 steht die Rechnung des
  Betragsvorschlags in `internal/service/ersatzwert_vorschlag.go`), der Leser einen aus den Benutzern
  (`protokolliereKontoAnlage`; seit dem 12.10.2026 steht er als `ProtokolliereKontoAnlage` in
  `api/tuer`, weil zwei Themen ein Konto anlegen). Aus den Schichten darunter brauchen die Themen 470 Namen aus
  `repository/`, 444 davon je genau ein Thema, 53 aus `internal/service`, jeden genau ein
  Thema, 34 aus `api/tuer` und acht aus `mailservice/`.
- **Zielbild.** In kein Thema ziehen zwölf Dateien mit 1.834 Zeilen: Router, Routenlisten,
  Middleware, CSRF, Rate-Limit und Abmeldung bleiben in `api/`, mit 23 Routen ohne Tür aus
  einem Thema, darunter Anmeldung, Sitzung und die acht Einhängepunkte von `inventur/`. Die
  Routenlisten bleiben an einer Stelle: Dort steht jede Tür mit ihrem Recht, und neun
  Prüfungen lesen sie dort. Die Themen, mit den Testdateien, die nur dieses eine Thema
  brauchen:

  | Thema      | Dateien | Zeilen | Routen | Testdateien | Dateien heute in `api/` |
  | ---------- | ------- | ------ | ------ | ----------- | ----------------------- |
  | Katalog    | 21      | 3.035  | 43     | 35          | `copy_admin*.go`, `auflagen_handler.go`, `book_systematik_handler.go`, `isbn_handler.go`, `schlagwort*.go`, `signaturen_handler.go`, `systematik_handler.go`, `lookups.go`, `exemplar_*.go`, `cover.go`, `cover_quelle_bindung.go`, `image_caching.go`, `geraete.go`, `abgangsbuch_handler.go`, `zugangsbuch_handler.go`, `bestandsbuch.go`, seit dem 12.10.2026 in `api/katalog` |
  | Leser      | 17      | 3.022  | 23     | 56          | `student_*.go`, `schueler_jahrgang_filter.go`, `photo_*.go`, `lusd.go`, `klassen_mapping.go`, `dsgvo_*.go`, dazu der Kontoauszug (`print_kontoauszug.go`); seit dem 13.10.2026 in `api/leser`, ohne `klassen_mapping.go`: Die Adressen der Klassenleitungen pflegt die Kategorie „Mahnwesen-Routing" der Einstellungen, sie stehen beim Mahnwesen |
  | Bestellung | 17      | 2.824  | 23     | 24          | `bestell*.go`, `orders_handler.go`, `supplier_handler.go`, `reorders*.go`, seit dem 12.10.2026 in `api/bestellung` |
  | System     | 12      | 1.537  | 17     | 20          | `settings.go`, `backup_status.go`, `betriebsbereitschaft_*.go`, `audit_*.go`, `stats.go`, `reporting_dashboard.go`, `mail_settings.go`, `mail_routes.go`, `littera_import.go`, seit dem 12.10.2026 in `api/system` |
  | Theke      | 9       | 1.472  | 11     | 24          | `action*.go`, `search.go`, `schueler_kiosk.go`, `nachbuch*.go`, `ausleihe.go`, `vormerkungen.go`, dazu die Barcode-Liste (`buchbarcodes_handler.go`); seit dem 13.10.2026 in `api/theke`, ohne die Titelsuche des Druck-Centers und die Verlängerung der Lernmittel einer Klasse |
  | Mahnwesen  | 7       | 1.302  | 6      | 11          | `mahnwesen*.go`, `graduates*.go`, `abgaenger_fenster.go`, seit dem 12.10.2026 in `api/mahnwesen`; seit dem 13.10.2026 dazu `klassen_mapping.go` (die Adressen der Klassenleitungen) |
  | Druck      | 8       | 1.172  | 15     | 8           | `labels.go`, `etiketten_offen.go`, `buchbarcodes_handler.go`, `barcode_*.go`, `schueler_etiketten.go`, `ausweis_layout.go`, `print_kontoauszug.go` (der Kontoauszug aus `print.go`); seit dem 12.10.2026 in `api/druck`, ohne drei Dateien, die nach der rufenden Seite zogen oder ziehen: `barcode_sequence.go` zum Katalog, `buchbarcodes_handler.go` zur Theke, der Kontoauszug zu den Lesern; seit dem 13.10.2026 dazu die Titelsuche des Druck-Centers (`titel_suche.go`, bis dahin in `search.go` bei der Theke) |
  | Schaden    | 7       | 1.154  | 13     | 10          | `bescheid_*.go`, `ersatzwert_vorschlag_handler.go`, `damage*.go`, `pdf.go`, dazu die Rechnung an die Eltern aus `print.go`; seit dem 12.10.2026 in `api/schaden` |
  | Lernmittel | 5       | 1.098  | 7      | 8           | `lmf_*.go`, seit dem 12.10.2026 in `api/lernmittel`; seit dem 13.10.2026 dazu die Verlängerung der Lernmittel einer Klasse (`lmf_verlaengerung.go`, bis dahin in `ausleihe.go` bei der Theke) |
  | Benutzer   | 6       | 838    | 6      | 9           | `user_admin*.go`, seit dem 12.10.2026 in `api/benutzer` |
  | Inventur   | 5       | 777    | 9      | 5           | `inventur_*.go`, seit dem 12.10.2026 in `api/inventurlauf` |
  | Portal     | 5       | 681    | 15     | 11          | `anliegen.go`, `reservation.go`, `katalog_kollegium.go`, `opac.go`, `monitor.go`, seit dem 12.10.2026 in `api/portal` |

  Zwei Namen sind als Paket vergeben (`pkg/lmf`, `inventur/`): Der Ordner der Lernmittel-Türen
  heißt deshalb nicht `lmf`, der Ordner der Inventur-Türen `inventurlauf`. `inventur/` ist
  der Medienkatalog, und die Routenliste bindet beide Pakete ein.
- **Was jede Tür braucht, steht in `api/tuer`** (seit dem 12.10.2026, fünf Dateien mit 530
  Zeilen): Eingabe lesen und Antwort schreiben (`RespondJSON`, `DecodeAndValidate`), zehn
  Konstanten, die Rechteprüfung mit ihrem Zwischenspeicher (`BesitztRecht`,
  `RequireAuthenticated`, `PruefeBerechtigung`), das Protokoll der Verwaltung mit dem Eintrag
  zur Konto-Anlage und die Uhr. Der
  Typ `tuer.Basis` trägt Datenbank, Anmeldung, Live-Leitung und Uhr; `api.Server` bettet ihn
  ein, und der Typ eines Themas bettet einen Zeiger auf die Basis des Servers ein, damit jede
  Tür `s.DB` und `s.BesitztRecht` unter demselben Namen erreicht. Der Mailversand steht in
  `mailservice/`,
  die Adresse des Aufrufers fragen die Türen bei `pkg/clientip`. `RequirePermission` bleibt
  beim Router: Es setzt die Prüfung der Pfadparameter vor die Rechteprüfung, und deren Liste
  gehört zu den Routen. Aus `api/` selbst brauchen die Themen noch zweierlei: den Typ
  `Server`, an dem ihre Türen als Methoden hängen (108 Dateien, 340 Methoden; nach Stufe 3
  noch 103 Dateien und 324 Methoden, nach den ersten vier Themen von Stufe 4 noch 77 Dateien
  und 262 Methoden, nach sechs noch 66 Dateien und 231 Methoden, nach sieben noch 47 Dateien
  und 177 Methoden, nach neun noch 25 Dateien und 116 Methoden, nach zehn noch neun Dateien
  und 62 Methoden; mit dem Umzug bekommt jedes Thema seinen eigenen Typ),
  und die Bremse je Adresse, die neben der Middleware nur der Abruf der Cover im Katalog
  braucht; sie steht seit dem 12.10.2026 in `pkg/ratelimit`. Das Feld `IdempotenzWartezeit`
  braucht nur die Theke.
- **Die Kosten liegen bei den Tests.** 221 Testdateien brauchen Namen aus genau einem Thema und
  ziehen mit ihm um, 43 brauchen aus `api/` nur, was dort bleibt (Router, Middleware, den Typ
  `Server`), 18 mehrere Themen. 40 brauchen
  keinen Namen aus `api/`: Ratschen, die Quelltext lesen, und Tests an der Datenbank, die
  `repository/`, `internal/service` oder das Schema prüfen und hier liegen, weil ihre Helfer
  hier liegen. Ein Test eines anderen Pakets kann einen Helfer aus einer Testdatei nicht
  einbinden. Seit dem 12.10.2026 stehen deshalb 54 Helfer in Paketen für Prüfhilfen: Zeilen der
  Testdatenbank in `internal/testdaten`, Anfragen an Türen in `internal/tuertest` (dort auch
  `MailAbfangen` und `SetzeGenauEinRecht`), je zwei bis drei in `internal/pgtest`,
  `internal/pdftest` und `internal/smtptest`; zwei weitere reichten nur an eine Funktion dort
  weiter und sind entfallen. In `api/` teilen sich Testdateien noch 91 Helfer aus 51 Dateien.
  Zwölf davon brauchen Tests aus mehr als einem Thema und ziehen mit ihrer Stufe: acht hängen
  an einem oder zwei Themen, die Liste der Tabellen mit Bezug auf einen Leser gehört zu den
  Tests der Auskunft (seit dem 13.10.2026 `testdaten.DsgvoSchuelerQuellen`), und drei bauen
  den ganzen Router (`routerMitSitzung`, `routerMitSitzungUeber`,
  `protokollWelt`). Mit der Inventur zog am 12.10.2026 ein weiterer nach
  (`tuertest.AntwortOhneDB`: eine Tür ohne Datenbank rufen), mit dem Mahnwesen fünf: einen
  Anhang aus einer abgefangenen Mail lesen (`smtptest.DateiAusMail`, `smtptest.PDFAusMail`)
  und Zeilen für einen Test anlegen (`testdaten.AusleihbaresExemplar`,
  `testdaten.SchuelerMitBuch`, `testdaten.AusleiheUeberDenDienst`), vor Druck und Bestellung
  fünf: der Aufruf über einen ganzen Router mit Sitzung und CSRF-Nachweis
  (`tuertest.RouterAufruf`), eine Bestellung über die Tür (`tuertest.Bestelle`), eine
  Bestellung aus der Zeit vor den Töpfen (`testdaten.AltBestellung`), der Eigentumsvermerk der
  Schülerbücherei als Einstellung mit seinen Werten (`testdaten.SetzeVermerkSchuelerbuecherei`)
  und die Prüfung eines gedruckten Texts (`pdftest.Aufdruck`), vor den Lesern einer: den Stand
  eines Schülers lesen (`testdaten.LeseSchuelerStatus`). In das Innere eines anderen Pakets
  greift kein Test mehr: Was eine Rolle
  darf, legt `tuer.MerkeRechtFuerTest` in den Zwischenspeicher, die Mail-Konfiguration tauscht
  `mailservice.TauscheKonfigLaderFuerTest`.
- **Das erste Thema ist umgezogen (Stufe 3, 12.10.2026).** Die fünf Dateien des LMF-Plans
  stehen in `api/lernmittel`, mit neun Testdateien: den acht, die nur dieses Thema brauchen,
  und `lmf_plan_live_test.go`, das die Schreibwege des Plans in der Routenliste prüft. Der
  Typ `Tueren` trägt die 17 Methoden und hält die Basis des Servers als eingebetteten Zeiger:
  Als Kopie gäbe es Datenbank, Live-Leitung und Uhr je Thema noch einmal, und eine Uhr, die
  ein Test nach dem Bau des Routers stellt, erreichte die umgezogenen Türen nicht. Die
  Routenliste meldet die sieben Routen über den Zugang `s.lernmittel()` an. Wie ein Thema
  gebaut ist, steht in [ARCHITEKTUR.md](ARCHITEKTUR.md) 5.2.2, der Handgriff mit dem Nachweis
  in 8.14. Aus `api/` brauchte das Thema nur den Typ `Server`; der Probelauf des Werkzeugs
  meldete kein Hindernis. Die Kommentare der umgezogenen Dateien sind unverändert, bis auf
  zwei, die den Typ `Server` nannten.
- **Vier weitere Themen sind umgezogen (Stufe 4, 12.10.2026),** je Thema ein Commit, gebaut
  wie das erste:

  | Thema    | Paket              | Dateien | Testdateien | Methoden | Routen |
  | -------- | ------------------ | ------- | ----------- | -------- | ------ |
  | Inventur | `api/inventurlauf` | 5       | 5           | 12       | 9      |
  | Benutzer | `api/benutzer`     | 6       | 10          | 12       | 6      |
  | Portal   | `api/portal`       | 5       | 11          | 17       | 15     |
  | System   | `api/system`       | 12      | 20          | 25       | 17     |

  Dazu kommt je Paket die Datei mit seinem Typ. Fünf Namen standen im Weg, jeder führte zu
  einer Entscheidung. Der Helfer der Tests, der eine Tür ohne Datenbank ruft, steht bei den
  Prüfhilfen (`tuertest.AntwortOhneDB`). Den Eintrag zur Konto-Anlage schreiben zwei Themen,
  Benutzer & Rechte und die Leserdatei; er steht deshalb in `api/tuer`
  (`ProtokolliereKontoAnlage`), und kein Thema bindet dafür ein anderes ein. Vier Testdateien,
  die über den ganzen Router prüfen, stehen als Testpaket im Ordner ihres Themas. Für die
  eine, die den Router über einen Helfer aus `api/` baute, gibt es das Paket
  `internal/routertest`. Den Bereitschafts-Alarm ruft der Start am Server (`main.go`); er
  steht bei den Türen der Systempflege, und der Server reicht den Aufruf weiter
  (`Server.BereitschaftsAlarm`). Das Werkzeug liest nur `api/`: Diesen Aufruf zeigte erst der
  Bau. Ein leeres Literal des Servers in einem Test (`&Server{}`) schreibt das Werkzeug als
  `&Tueren{Basis: &tuer.Basis{}}`: Am Server war die Basis eingebettet und leer, am Typ des
  Themas wäre sie sonst nil, und das Protokoll der Verwaltung bräche ab.
- **Mahnwesen und Schaden sind umgezogen (Stufe 4, 12.10.2026),** je mit Vorarbeit in eigenen
  Commits:

  | Thema     | Paket           | Dateien | Testdateien | Methoden | Routen |
  | --------- | --------------- | ------- | ----------- | -------- | ------ |
  | Mahnwesen | `api/mahnwesen` | 7       | 15          | 13       | 6      |
  | Schaden   | `api/schaden`   | 8       | 14          | 19       | 14     |

  Beim Mahnwesen standen Helfer und Tests im Weg. Fünf Helfer, die Tests mehrerer Themen
  brauchen, stehen bei den Prüfhilfen. Zwei Testdateien trugen Tests zu Türen zweier Themen
  und sind getrennt: Das Paar-Gate Abgängerliste gegen Versetzung braucht beide Themen,
  bleibt in `api/` (`abgaenger_versetzung_paar_pg_test.go`) und liest die Liste selbst von
  der Tür; der Test zum Laufzettel zog mit dem Mahnwesen um. Die Datei mit der Tür der Liste
  hieß `mahnwesen.go` wie das Paket und heißt jetzt `mahnwesen_liste.go`.
  Beim Schaden stand eine Regel im Weg: Den Ersatzwert eines Buchs nennen vier Türen, zwei
  des Schadens und zwei des Katalogs, und gerechnet wurde beim Schaden. Die Rechnung steht
  jetzt in `internal/service/ersatzwert_vorschlag.go`. Proben je Aufrufstelle zeigten, dass
  nur eine der vier Türen daraufhin geprüft war, ob sie die Einstellung „immer der
  Kaufpreis" beachtet; `api/ersatzwert_grundlage_tueren_pg_test.go` vergleicht jetzt alle
  vier. Die frühere Datei `print.go` trug die Rechnung an die Eltern und den Kontoauszug; die Rechnung
  gehört zum Schaden (`print_rechnung.go`), der Kontoauszug steht in
  `api/leser/print_kontoauszug.go`. Die Tür der Rechnung ist eine Funktion ohne Server und steht
  in der Routenliste mit dem Paketnamen (`schaden.PrintRechnungHandler`).
- **Der Katalog ist umgezogen (Stufe 4, 12.10.2026),** mit Vorarbeit in drei Commits:

  | Thema   | Paket         | Dateien | Testdateien | Methoden | Routen |
  | ------- | ------------- | ------- | ----------- | -------- | ------ |
  | Katalog | `api/katalog` | 21      | 35          | 55       | 43     |

  Im Weg standen die Bremse je Adresse, die neben der Middleware der Abruf der Cover braucht
  (jetzt `pkg/ratelimit`), eine DNB-Attrappe, die ihre Anfragen zählt und die Tests zweier
  Themen nutzen (jetzt `tuertest.ZaehlendeDNB`), und eine Quelltext-Ratsche in einer
  Testdatei der Türen (jetzt `api/abgangsdatum_ratsche_test.go`; siehe unten, Prüfungen, die
  Dateien lesen). Der Abgleich der Cover ist eine Funktion ohne Server und steht mit dem
  Paketnamen in der Routenliste (`katalog.SyncCoversHandler`). Titelliste, Buchmaske und
  Import bleiben im Untermodul `inventur/`. Die Probe am Zugang lief hier auch mit einer
  Kopie der Basis statt des Zeigers: Bei sieben von acht Zugängen blieb die Suite grün.
  `api/zugaenge_basis_test.go` vergleicht seither je Zugang den Zeiger mit der Basis des
  Servers.
- **Druck und Bestellung sind umgezogen (Stufe 4, 12.10.2026),** mit Vorarbeit in vier
  Commits:

  | Thema      | Paket            | Dateien | Testdateien | Methoden | Routen |
  | ---------- | ---------------- | ------- | ----------- | -------- | ------ |
  | Druck      | `api/druck`      | 5       | 5           | 12       | 11     |
  | Bestellung | `api/bestellung` | 17      | 33          | 46       | 23     |

  Zwischen beiden lag der Kreis um das Buchetikett. Der Typ des Etiketts, der Kopf der
  Schule, die drei Wege zu den Daten eines Etiketts und die Funktion, die die Eingabe der
  Erzeuger füllt, stehen jetzt in `internal/buchetikett`, unter den Türen. Die Proben dazu
  zeigten eine Lücke im Vergleich der Wege: Der Test lief an einem Titel ohne Signatur und
  ohne Autor, und zwei leere Felder sind gleich; nahm die Abfrage der Bestellung eines der
  beiden nicht mehr mit, blieb die Suite grün. Der Titel des Tests trägt jetzt beide. Fünf
  Helfer der Tests zogen zu den Prüfhilfen; der Test „Eigentumsvermerk folgt dem Topf" ist
  nach Thema getrennt, der Test der Etikettenkette prüft Türen beider Themen und bleibt in
  `api/`. Zwei Dateien, die die Zuordnung beim Druck führte, ziehen nach der Seite, die sie
  ruft: die nächste freie Exemplarnummer zum Katalog (`katalog/barcode_sequence.go`), die
  Barcode-Liste der Theke mit der Theke (`buchbarcodes_handler.go`). Zum Katalog kam außerdem
  der Test der Bestandsbücher über den Router, der beim Umzug in `api/` geblieben war; das
  Paket trägt damit 22 Dateien, 37 Testdateien, 56 Methoden und 44 Routen. Die Grenze für die
  Größe des Ausweis-Designs braucht ein Test im Testpaket und ist groß geschrieben
  (`druck.MaxAusweisLayoutBytes`).
- **Die Leser sind umgezogen (Stufe 4, 13.10.2026),** mit Vorarbeit in vier Commits:

  | Thema | Paket       | Dateien | Testdateien | Methoden | Routen |
  | ----- | ----------- | ------- | ----------- | -------- | ------ |
  | Leser | `api/leser` | 17      | 57          | 52       | 21     |

  Die Türen zu den Adressen der Klassenleitungen ruft die Kategorie „Mahnwesen-Routing" der
  Einstellungen; sie stehen beim Mahnwesen (`api/mahnwesen/klassen_mapping.go`), nicht bei
  den Lesern. Zu welchem Thema eine Tür gehört, sagt die Stelle der Oberfläche, die sie ruft:
  der Menüpunkt, in den Einstellungen die Kategorie
  ([ARCHITEKTUR.md](ARCHITEKTUR.md) 5.2.2). Im Weg standen ein Helfer und eine
  Liste aus Testdateien, die in `api/` bleiben: Den Stand eines Schülers liest jetzt
  `testdaten.LeseSchuelerStatus`, die Liste der Tabellen mit Bezug auf einen Leser steht als
  `testdaten.DsgvoSchuelerQuellen` bei den Prüfhilfen. Vier Prüfungen lesen sie: Der Scan am
  Quelltext der Auskunft und der Test zum Zusammenführen bleiben in `api/`, die Rundreise und
  die Ratsche der Fremdschlüssel zogen mit den Lesern. Die Proben dazu zeigten eine Lücke:
  Fiel eine Quelle ohne Fremdschlüssel aus der Liste (`leser`, `audit_log`, `audit_logs`),
  blieb die Suite grün; ein Test in `api/dsgvo_paar_vollstaendigkeit_test.go` hält die drei
  jetzt. Sechs Tests prüfen neben Türen der Leser die eines umgezogenen Themas und bleiben in
  `api/`. Drei davon riefen einen Helfer aus einer Testdatei der Leser und rufen die Tür
  jetzt selbst, über `tuertest.RufeAlsAdmin` und `tuertest.JSONPost`. Der Test zur Dublette
  aus Name und Geburtsdatum prüft über den Router und steht als Testpaket im Ordner. Die Tür
  des Kontoauszugs ist eine Funktion ohne Server und steht mit dem Paketnamen in der
  Routenliste (`leser.PrintKontoauszugHandler`).
- **Ein Test über den ganzen Router steht im Ordner seines Themas, als eigenes Testpaket**
  (entschieden mit Stufe 3, 12.10.2026). `api/` bindet das Paket eines Themas ein. Eine
  Testdatei im Paket des Themas kann `api/` deshalb nicht einbinden, auch nicht über eine
  Prüfhilfe, die es tut: Go weist den Kreis ab (`import cycle not allowed in test`). Ein
  Testpaket im selben Ordner (`package lernmittel_test`) darf es. Vom Thema sieht es nur groß
  geschriebene Namen; ein Test, der eine Tür über ihre Adresse ruft, braucht keinen. So hält
  es auch Googles Regelwerk für Go, Abschnitt
  [Test package](https://google.github.io/styleguide/go/decisions#test-package): ein Paket mit
  der Endung `_test`, wenn Tests im selben Paket einen Kreis ergäben. Blieben solche Tests in
  `api/`, stünden die Tests eines Themas an zwei Orten. Gebaut ist es am ersten Thema:
  `api/lernmittel/lmf_plan_tuer_pg_test.go` baut den Server mit `api.NewServer`, speichert
  einen Plan über die Route und prüft, dass die Türen in die Datenbank des Servers schreiben
  und mit seiner Uhr rechnen, auch wenn sie nach dem Bau des Routers gestellt wird. Konto,
  Sitzung und Anmeldung liefert `tuertest.AdminSitzung`; denselben Helfer ruft
  `routerMitSitzungUeber` in `api/`. Gezählt am 12.10.2026: Von den acht Testdateien der
  Lernmittel brauchte keine den Router. In `api/` bauen ihn 29 Testdateien; sieben brauchen
  dazu Namen eines Themas, 22 rufen die Türen nur über ihre Adresse. An den Adressen gemessen
  rufen 20 der 29 Türen genau eines Themas und ziehen in Stufe 4 mit ihm um; 19 sind es
  nach zehn Themen (Benutzer eine, Portal zwei, System eine, Mahnwesen drei, Schaden zwei,
  Katalog zwei, Druck drei, Bestellung vier, Leser eine), zehn bauen den Router noch in
  `api/`. Einer der
  20 bleibt dort: Der Test zur Panik in einer Tür ruft eine Adresse des Drucks und prüft die
  Schichten des Routers. Die Prüfungen
  über alle Routen (Personendaten in den Antworten, Grenzen der Listen, Rechte der
  Schreibwege) bleiben in `api/`. Für die Helfer, die den Router bauen, gibt es seit dem
  12.10.2026 das Paket `internal/routertest`, das `api/` einbindet; es trägt `MitSitzung`,
  `MitSitzungUeber` (der Server bekommt einen eigenen Pool) und `MitAufruf` (ein Admin-Konto
  mit fertigem Aufruf über den Router, in `api/` heißt er `protokollWelt`). Ein Test im Paket
  `api` kann es nicht nutzen und behält seinen Helfer in
  `api/pgtest_support_test.go`: Beide rufen `tuertest.AdminSitzung` und bauen den Server. Ob
  die Tests, die in `api/` bleiben, am Ende als Testpaket stehen und der Helfer dort
  entfällt, entscheidet Stufe 5.
- **Was beim letzten Thema im Weg steht, gemessen am 13.10.2026** nach zehn Themen
  von Stufe 4. Der Probelauf des Werkzeugs über das Thema, mit den Testdateien,
  die nur dieses Thema brauchen, nennt zweierlei: einen Namen, den eine umziehende Datei aus
  `api/` braucht, und einen klein geschriebenen Namen des Themas, den eine Datei braucht, die
  bleibt.

  | Thema      | Dateien mit Tests | Namen im Weg | darunter |
  | ---------- | ----------------- | ------------ | -------- |
  | Theke      | 38                | 11           | das Feld `IdempotenzWartezeit`, zwei Fristen der Middleware, ein Test, der den Server baut, sieben Zugänge umgezogener Themen, die drei Tests rufen |

  Gemessen mit der Barcode-Liste bei der Theke (`buchbarcodes_handler.go`). Ein Test, der den
  Zugang eines umgezogenen Themas ruft, prüft zwei Themen und bleibt in `api/`; der Test der
  UUID-Eingaben ruft fünf Zugänge. Jeder Name ist eine
  Entscheidung von Hand: groß schreiben, den Helfer zu den Prüfhilfen legen, den Test in das
  Testpaket des Themas stellen oder eine geteilte Regel an einen dritten Ort ziehen. Der Rest
  eines Umzugs ist der Lauf des Werkzeugs und der Skripte. Ein Teil der Namen kommt aus den
  Testdateien, die Namen aus mehreren Themen brauchen; sie bleiben im Probelauf in `api/`.
  Die Zahl nennt `NewServer` und `Routes` mit, wo ein Test über den Router prüft: Als
  Testpaket umgezogen sind sie kein Hindernis.
- **Prüfungen, die Dateien lesen.** Jede der 43 Prüfungen, die Go-Dateien des Baums
  durchlaufen, sieht eine Datei in einem Unterordner von `api/` (belegt am 12.10.2026, je mit
  einer Probe). Beim Umzug eines Themas werden von selbst rot und ziehen mit: jede
  Ausnahmeliste, die eine Datei beim Pfad nennt (`"api/csrf.go:CSRFMiddleware"`; so der
  Bestand der XLSX-Lesestellen in `docs/xlsx_leser_grenzen_test.go`), jede Liste, die einen
  Typ mit seinem Paket nennt (die Ausnahmen in `uuid_eingaben_test.go`:
  `benutzer.CreateUserRequest.BarcodeID`), und
  Prüfungen, die ihr Thema in festen Dateien suchen: `mail_vorlagen_platzhalter_test.go` den
  Standardtext in `mahnwesen/mahnwesen_bulk.go`; `lmf_plan_live_test.go` ist mit den Lernmitteln
  umgezogen und liest die Schreibwege des LMF-Plans in der Routenliste einen Ordner höher.
  Die Routenlisten (`router.go`, `routes_*.go`) bleiben in `api/`; neun
  Prüfungen lesen sie dort beim Namen. Eine Prüfung, die den Aufruf einer Funktion am Namen
  ohne Paket erkennt, sieht ihn aus einem anderen Paket nicht mehr: `uuid_eingaben_test.go`
  fand nach Stufe 2 keine Eingabe und wurde an seiner Untergrenze rot; es kennt jetzt beide
  Formen. Umgekehrt sieht eine Prüfung, die mit ihrem Thema umzieht und den Baum ab `..`
  durchläuft, einen Ordner tiefer nur noch `api/` und bleibt grün. In Stufe 4 betrifft es eine
  Prüfung: Die Ratsche zum Abgangsdatum stand bei den Tests an den Türen des Katalogs und
  steht seit dem 12.10.2026 in `api/abgangsdatum_ratsche_test.go`, wo sie bleibt; in den
  Listen der übrigen vier Themen steht kein Baumlauf (gemessen am 12.10.2026). Sechs
  Testdateien in `api/` durchlaufen den Baum ab `..`. Zieht eine in Stufe 5 zu ihrem Paket,
  braucht sie die Wurzel von dort und eine Probe mit einem Verstoß außerhalb ihres Ordners.
  Der Handgriff steht in [ARCHITEKTUR.md](ARCHITEKTUR.md) 8.14.
- **Verweise beim Pfad.** Die Dokumente nennen Dateien aus `api/` an 307 Stellen beim Pfad, der
  Go-Code außerhalb von `api/` an 147 Stellen in 90 Dateien, die Oberfläche an 67 (gezählt am
  12.10.2026). `docs/dateiverweise_test.go` hält die Verweise der Dokumente, der Kommentare im
  Go-Code, der Oberfläche und der Skripte gegen die Dateien des Repositorys. Der Umzug schreibt
  die Pfade je Thema mit einem Skript um, und die Prüfung nennt, was das Skript ausgelassen
  hat. Zeichenketten im Go-Code liest sie nicht: Die Meldungen und Listen der Ratschen, die
  eine Datei nennen, schreibt das Skript mit um. Es lässt aus, was eine Datei nennt, wie sie
  einmal war: den datierten Prüfbericht unter `bewertung/` mit seinen Zeilennummern und die
  Ordner der Werkzeuge (`.claude/`, `.jules/`).
- **Nachweis je Stufe.** Die Tests und ihr Ausgang sind vor und nach der Stufe dieselben; neue
  kommen nur dazu, wo die Stufe sie baut. Vergleichsstand vom 13.10.2026, nach zehn Themen
  von Stufe 4: 2.276 Testfunktionen, davon 252 in `api/`, 112 in `api/leser`, 91 in
  `api/katalog`, 87 in `api/bestellung`, 44 in `api/system`, 39 in `api/mahnwesen`, 35 in
  `api/schaden`, 23 in `api/benutzer`, je 18 in `api/lernmittel` und `api/inventurlauf`, 15 in
  `api/portal`, sieben in `api/druck`, fünf in `internal/buchetikett`, 30 in `mailservice/`
  und drei in `api/tuer`, sieben übersprungen; verglichen wird am Namen des Tests, weil ein
  Test mit
  seiner Datei das Paket wechselt. Methode, Pfad und Recht jeder Route sind dieselben (211 in
  `api/`, 18 in
  `inventur/`); das hält `api/pii_matrix_test.go` gegen die PII-Matrix, belegt mit acht Proben.
  Jede Erklärung ist vor und nach der Stufe Token für Token gleich, gleich in welcher Datei
  und in welchem Paket sie steht, nachdem die neuen Namen zurückgeführt sind; jede Abweichung
  steht in der Commit-Nachricht (Stufe 2 gegen den Stand davor: 2.321 Erklärungen gleich, 26
  gewollt verschieden, acht neu, eine entfallen; Stufe 3: beim Umzug 2.080 gleich, vier
  gewollt verschieden, zwei neu, beim Test über den Router eine verschieden und drei neu;
  Stufe 4, je Thema gegen den Commit davor: Inventur 2.085 gleich, eine gewollt verschieden,
  zwei neu, Benutzer 2.088, eine, zwei, Portal 2.090, eine, zwei, System 2.091, zwei, sechs,
  Mahnwesen 2.089, drei, zwei, Schaden 2.085, eine, zwei, Katalog 2.077, vier, zwei, Druck
  2.060, drei, zwei, Bestellung 2.063, zwei, zwei, Leser 2.063, drei, zwei).
  Dazu die ganze Go-Suite an der Datenbank, golangci-lint, deadcode, gosec und die
  Browser-Suite. Wie der Vergleichsstand erhoben wird, steht in
  [ARCHITEKTUR.md](ARCHITEKTUR.md) 8.14.
- **Reihenfolge der Themen:** nach dem, was ein Thema von anderen braucht. Umgezogen sind
  Lernmittel, Inventur, Benutzer, Portal, System, Mahnwesen, Schaden, Katalog, Druck,
  Bestellung und Leser. Es folgt die Theke.
- **Was nicht dazugehört:** die Abfragen je Thema zu teilen. Aus Sicht der Türen ginge es (444
  von 470 Namen gehören zu einem Thema). `repository/` brauchen aber auch `internal/service`,
  `jobs/` und `auth/`, und dort stehen die Regeln, die es einmal geben soll (die Bedingungen der
  Löschfristen, der Topf eines Exemplars, wer als Wartender ein Buch bekommt). `inventur/` zeigt
  den Preis einer eigenen Datenbankschicht je Thema: Schreibende Anweisungen an `buecher_titel`
  stehen in sechs Dateien von `repository/` und in drei von `inventur/`. Entschieden wird das
  nach dem Umzug der Türen, an einer Messung von `repository/` wie dieser.
- **Was der Umzug nicht bringt:** weniger Code und kürzere Bauzeiten. Die Zeilen bleiben, und
  das Testprogramm von `api/` baut nach einer geänderten Zeile in 4,4 Sekunden (gemessen am
  12.10.2026). Er bringt einen Ordner je Menüpunkt statt 453 Dateien in einem, und der
  Compiler weist ab, wenn ein Thema in ein anderes greift.

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

Der Totalverlust des Servers steht in Abschnitt 2f, durchgespielt am 08.10.2026 an einem
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
deletions" anlassen. Am 08.10.2026 trägt das Ruleset noch `pull_request`; Pushes gehen über den
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

**Der Jahrgang am Titel.** Das Update nimmt jedem Titel, der „5 bis 10" trug, den Jahrgang (am
Testserver 13.056 von 13.062, gemessen am 08.10.2026; sechs Titel behalten ihre eigene Spanne)
und entfernt das frühere Feld „Klasse" (am Testserver trugen es 155 Titel; die Angabe verfällt,
in „von … bis" wird nichts übernommen).
„Neues Buch" öffnen. Erwartet: Die zwei Felder zum Jahrgang stehen leer, ein Feld „Klasse"
gibt es nicht. In „von" eine 7 tippen. Erwartet: „bis" zeigt sofort auch 7; nach Tab und 10
steht dort 10. Einen Titel mit „7 bis 7" öffnen, „bis" leeren, „Speichern". Erwartet: Die
Meldung sagt, dass „von" und „bis" zusammengehören; mit beiden geleert speichert die Maske.
Titel-Verwaltung. Erwartet: Die Spalte heißt „Jahrgang" und zeigt „7", „7–10" oder einen
Strich. Portal, Reiter
„Schulbücher", Filter „Jahrgang 7". Erwartet: nur Bücher, an denen ein Jahrgang eingetragen
ist; ohne Filter steht die Liste da wie vorher. Inventur: Die am 23.07.2026 begonnene und nicht
abgeschlossene Inventur (Fach und Klasse 5) erwartet nur noch Exemplare von Titeln, an denen
Jahrgang 5 eingetragen ist; vorher zählte dort jedes Exemplar des Fachs.

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
Abgehakte Zeilen fallen bei der nächsten Durchsicht der Liste weg, eine fertige Etappe als
Ganzes. In den Einzelheiten wird Erledigtes gelöscht.

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
