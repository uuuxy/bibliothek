# Offene Arbeit

Stand: 16.10.2026

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
  Datenbankschicht. ([ARCHITEKTUR.md](ARCHITEKTUR.md) 5.2.2)
- [x] Türen in Themenordner, dritte Freigabe: Stufe 4 ist seit dem 12.10.2026 freigegeben,
  je Thema ein Schritt, zuerst die vier Themen mit dem wenigsten im Weg (Inventur, Benutzer,
  Portal, System).
- [x] `repository/` nach Themen teilen: nein, entschieden am 13.10.2026 nach der Messung.
  Ohne Umbau ließen sich 48 der 151 Dateien einem Thema zuordnen; was mehrere Themen
  brauchen (Protokoll, Einstellungen, Titel, Leser), bliebe ein gemeinsames Paket mit 72 %
  der Zeilen, und 21 Tabellen bekämen Schreiber in zwei Paketen. Die Zahlen stehen in
  [ARCHITEKTUR.md](ARCHITEKTUR.md), Kapitel 9 (A25). Gebaut ist dafür nichts; die Entscheidung lässt
  sich ändern, ohne etwas zurückzunehmen.
- [ ] Sicherung außer Haus: Alte Sicherungen löscht das Programm nur auf dem Server, am
  Speicher außer Haus nie. Vorschlag: Es löscht dort nach derselben Regel (die jüngsten 14
  Nächte, dazu je Woche eine für 12 Wochen); eine Löschregel, die jemand am Speicher von Hand
  einstellt, braucht es dann nicht. (7.3, R8)
- [x] Etiketten auf die Breite der Zelle kürzen: ja, entschieden am 13.10.2026 und gebaut.
  Gekürzt wird nur, was über den Rand seiner Zelle lief; was passt, steht da wie vorher, der
  Schulname auch am kleinen Bogen ganz. Die Obergrenze in Zeichen gilt weiter: Ein langer
  Titel zeigt am breiten Bogen so viele Zeichen wie vorher. Soll er dort mehr zeigen,
  entfällt die Obergrenze an einer Stelle (`kuerzeAufEtikett` in `pdf/etikett_buch.go`).
- [x] Zwei zugelieferte Pull Requests vom 13.10.2026 (734 und 735): entschieden am
  13.10.2026, nach dem Vorschlag. 734 ist übernommen; was am zusammengeführten Stand geprüft
  ist, steht im Merge-Commit. 735 ist geschlossen; die Messung steht am Pull Request.
- [x] Vier zugelieferte Pull Requests (726 bis 729): geschlossen am 12.10.2026, nach dem
  Vorschlag. Die Begründung steht je am Pull Request. Soll es den Knopf zum Leeren der Suche
  geben, entsteht er neu nach der Seite von Material 3.
- [x] Drei Updates von Dependabot (731, 732, 733): übernommen am 12.10.2026, nach dem
  Vorschlag; was am zusammengeführten Stand geprüft ist, steht in den drei Merge-Commits. Der
  Schritt mit upload-artifact läuft nur nach einem roten Browser-Test und ist ungeprüft. Das
  vierte Update (730, zehn Pakete der Oberfläche) ist übernommen, mit typescript-eslint
  8.71.2; was geprüft ist, steht im Merge-Commit.
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
- [ ] Inventur: eine unbekannte Nummer scannen („Zu diesem Barcode gibt es kein Exemplar")
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
- [ ] Am Testserver einmal den Build-Cache leeren: `docker builder prune -af`. Dort liegen
  12,8 GB, die Platte ist zu 82 % belegt (gelesen am 13.10.2026). Images, Container und
  Volumes fasst der Befehl nicht an; das Update danach baut jede Schicht neu. Ab dem
  übernächsten Update leert `update.sh` den Build-Cache selbst (Schritt 7).
- [ ] Theke ohne Netz: der Nachweis von Hand im echten Chrome (Stufe 1 und 3), zurückgestellt
  am 24.09.2026. Stufe 2 über die Tür ist seit dem 08.10.2026 belegt. (2.3)

### Termine

- [ ] 19. Oktober 2026: CodeQL wechselt bei GitHub das Abbild; den Lauf danach ansehen. (5.10)
- [ ] Ab dem 28. Oktober 2026: Node 26, nach der Regel „immer die aktive LTS"
  ([PFLEGEKONZEPT.md](PFLEGEKONZEPT.md), Abschnitt 4).

### Umbau: Lesbarkeit nach Googles Regelwerk für Go

Entschieden am 13.10.2026: Wer neu in den Code kommt, soll sich ohne langes Lesen
zurechtfinden; die Zeit dafür spielt keine Rolle. Die Regeln für Namen, Sichtbarkeit und
Kommentare stehen in [STIL.md](STIL.md). Umgestellt wird Paket für Paket, von unten nach oben,
je Paket alles in einem Zug. Was „fertig" heißt und wie belegt wird, steht unter 5.66.

- [x] Die Regeln stehen auf einer Seite ([STIL.md](STIL.md)); eine Prüfung hält je Paket fest,
  was ihnen noch nicht folgt, und lässt den Bestand nur sinken (`stil_ratsche_test.go`).
- [x] Die Pakete unter `pkg/`.
- [x] Unter `internal/` die neun Pakete, die `repository` nicht einbinden.
- [x] `repository/`.
- [x] Die übrigen acht Pakete unter `internal/` ohne `internal/service` (Reihenfolge in 5.66).
- [x] `internal/service`.
- [x] `api/tuer`.
- [x] Die zwölf Themen unter `api/`.
- [x] `api/`.
- [x] Das Wurzelpaket.
- [x] Die Testdateien: Namen und Kommentare; Vergleiche mit dem Paket `testing` statt testify.
- [x] Fehlertexte: klein und ohne Satzzeichen; den Satz für die Oberfläche schreibt die Tür
  ([STIL.md](STIL.md), Abschnitt 11).
- [x] Zustand der Paketebene: Was sich ändert, legt der Rufer an ([STIL.md](STIL.md),
  Abschnitt 12).
- [x] Die Oberfläche, Stufe 1: Datei- und Ordnernamen deutsch ([STIL.md](STIL.md), Abschnitt 13;
  `frontend-hygiene-sprache.test.js`), bis auf zwei geteilte Bauteile (5.66).
- [x] Die Oberfläche, Stufe 2: Namen innerhalb der Dateien (5.66), bis auf `apiDelete`, das wie
  die HTTP-Methode heißt.
- [x] Die Oberfläche, Stufe 3: Props, die ein Bauteil vom Rufer bekommt (5.66).
- [x] [ARCHITEKTUR.md](ARCHITEKTUR.md) kürzen.

Was nach dem Abgleich vom 16.10.2026 noch von Googles Regelwerk abweicht, in der Reihenfolge des
Gewichts:

- [ ] Schnittstellen beim Nutzer statt beim Hersteller: `New…` in `repository/` und
  `internal/service` liefert den Typ. Noch zwei Stufen, je Stufe deine Freigabe; das Gate steht.
  (5.71)
- [x] Der Kontext der Anfrage reist bis zur Abfrage mit; kein Kontext als Feld. (5.71)
- [x] Öffentlich nur, was ein anderes Paket nennt (68 Namen), mit einer Prüfung, die das hält.
  (5.71)
- [ ] Lokale Namen in Tests deutsch und `t.Helper()` in den Hilfsfunktionen, je mit Ratsche.
  (5.71)
- [x] Leerimporte nur in `main` und in Tests. (5.71)
- [ ] Oberfläche: `stores/omnibox.svelte.js` (862 Zeilen) nach seinen Sachen teilen, in
  noch vier Stufen, je Stufe deine Freigabe. (5.72)

### Später: kleine Fehler und Aufräumarbeit

Ohne feste Reihenfolge. Jede Zeile bündelt, was unter ihrer Nummer in den Einzelheiten steht.
Was davon fertig ist, wird dort gelöscht und fällt aus der Zeile; abgehakt wird sie, wenn unter
der Nummer nichts mehr dazu offen ist.

- [ ] **Gates und Werkzeuge (5.10):** Der erste Lauf von `release.yml` auf Ubuntu 26, die
  Excel-Bibliothek und gosec auf einem unveröffentlichten Stand.
- [x] **Schichtung des Backends (5.62):** abgeschlossen am 13.10.2026. Die sechs Dateien mit
  Tür und Regel sind gelesen und getrennt, beide Importe fragen den Titelbestand mit einer
  Methode. Was an der Tür bleibt und warum, steht in [ARCHITEKTUR.md](ARCHITEKTUR.md) 5.2.2
  (Tür und Abfrage); die Größen misst der Anhang von Kapitel 5.
- [x] **Auflagen-Zeile im Schulbuch-Export (5.63):** erledigt am 16.10.2026. Gemessen: In
  Großbuchstaben wurde die Zeile vierzeilig; sie kürzt jetzt nach gezählten Zeilen.
- [ ] **Umfang senken (5.67):** SQL-Testdaten über `internal/testdaten`.
- [ ] **Vorgeschichte in Kommentaren der Oberfläche (5.70):** 448 Kommentarzeilen mit Datum;
  eine Ratsche wie im Go-Code, dann Datei für Datei.
- [x] **Meldungen in Satzschreibung und auf Deutsch (5.68):** erledigt am 16.10.2026. Der Text
  eines Fehlers ohne Meldung geht unter keinem Status mehr hinaus (`apierrors`), und die
  Meldungs-Ratsche prüft die festen Sätze jedes Fehlerstatus außer 500, auch 502 und 503.
- [x] **Tippfehler in TRUSTED_PROXIES (5.69):** erledigt am 15.10.2026. Ein Eintrag, der weder
  Netz noch Adresse ist, steht als Warnung im Log des Starts und auf der Seite
  „Betriebsbereitschaft" (Bereich „Vertraute Proxys").
- [ ] **Statistik bei Lesefehler (5.73):** Ein gescheitertes Lesen sieht aus wie ein leerer
  Bestand, bei den Ladenhütern sogar wie eine gute Nachricht.

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
(`erneuereCSRFCookie` in `api/csrf.go`), der Aufruf von `/api/auth/me` beim Laden
(`restoreSession`) darf scheitern, und die Seite zeigt keine Personendaten — Titel, ISBN, Menge,
Schule, Lieferant, Topf, Kundennummer (`api/bestellung/bestellbestaetigung_oeffentlich.go`).

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

- **Ubuntu 26 als Runner: der erste Lauf von `release.yml`.** Alle zehn Jobs der vier
  Workflows laufen auf `ubuntu-26.04`; `ci.yml`, `security-scan.yml` und `docker-publish.yml`
  sind dort grün (zuletzt am Stand 9687c973). `release.yml` läuft erst mit dem nächsten v-Tag;
  den Lauf dann ansehen. Es braucht dort nur `git`, `gh` und `scripts/tag-gate.sh`. Nicht
  darunter: CodeQL läuft in der Standard-Einrichtung von GitHub (Repository-Einstellung, keine
  Workflow-Datei) auf `ubuntu-latest` und wechselt am 19. Oktober 2026 mit; der Hinweis darauf
  steht an jedem CodeQL-Lauf (gesehen am 28.09.2026). Bricht die Analyse dort, wird der
  CodeQL-Lauf rot.
- **excelize auf einem unveröffentlichten Stand.** Eingesetzt ist
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
  (CVE-2026-107212; `pkg/xlsxgrenze/zeilennummer_test.go`). Das Programm
  liest mit der Bibliothek nur, an drei Stellen hinter Anmeldung und Fachrecht
  (`inventur/excel_import.go`, `internal/lusd/quelle.go`, `api/system/littera_import.go`), alle
  durch `xlsxgrenze.LeseMappe`; die Schranke dort bleibt als zweite Lage. Offen: auf die
  veröffentlichte Fassung heben, sobald sie erscheint (Dependabot schlägt sie vor). Kommt
  vorher eine weitere Meldung, wird der Sicherheits-Scan rot; der Handgriff steht in
  [PFLEGEKONZEPT.md](PFLEGEKONZEPT.md), Abschnitt 5. Fällt an einem der drei Importe etwas
  auf, zuerst gegen v2.11.0 gegenprüfen. Kategorie B.
- **gosec auf einem unveröffentlichten Stand.** Der Sicherheits-Prüflauf und der Hook vor dem
  Push bauen gosec vom Entwicklungsstand des 05.10.2026
  (`v2.29.1-0.20261005092323-d2b649ec0182`, mit `golang.org/x/tools` 0.51); die Fassung steht
  in `scripts/gosec-gate.sh`. Die jüngste
  veröffentlichte Fassung 2.29.0 liest die Paketdaten von Go 1.27.2 nicht. Am Stand des
  Tages meldet der neue Stand mit denselben Ausnahmen nichts (489 Dateien, 61 Vermerke
  `#nosec`). Offen: auf die nächste veröffentlichte Fassung heben, sobald sie erscheint; sie
  kann neue Regeln mitbringen, deshalb mit eigenem Commit. Kategorie B.

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
  jedes Update baut dann alles neu (in der CI 93 Sekunden). `update.sh` leert nach jedem
  Update den Build-Cache ganz (Schritt 7): Das folgende Update baut
  damit schon jede Schicht neu, auch die mit `apk upgrade`. Offen bleibt `--pull` für die
  Basis-Images.
- **Das Löschen hängt am Lauf** (gefunden am 28.09.2026). `update.sh` und `scripts/backup.sh`
  löschen alte Sicherungen nur, wenn sie laufen. Kommt kein Update mehr, bleibt die letzte
  Vorab-Sicherung für immer; ein Klartext-Rest nach einem misslungenen Update bleibt bis zum
  ersten Lauf eines der beiden Skripte nach zwei Tagen. Die Meldungen sagen das.
  Vorschlag, nicht nachgestellt: Mit fertigen Images bleibt das alte Image am Server. Läuft
  der alte Container, verschlüsselt Schritt 1 die Sicherung sofort, und der Rückweg
  entschlüsselt sie mit dem alten Image (`docker run --rm`); die Vorab-Sicherung läge dann nie
  unverschlüsselt in `backups/`. Läuft er nicht, bleibt der Klartext-Weg. Für das Löschen nach
  der Uhr bräuchte es einen Lauf, der nicht am Update hängt.

### 5.66 Lesbarkeit nach Googles Regelwerk für Go

**Fertig sind alle 79 Pakete und ihre Tests.** `stil_bestand_test.go` ist leer; ein neuer Fund
im Programmcode oder ein neuer Name in einem Test macht `stil_ratsche_test.go` rot, testify und
`reflect.DeepEqual` sperrt der Linter ([STIL.md](STIL.md), Abschnitt 10).

**Ein Paket ist fertig, wenn:**

1. jeder Name, den es erklärt, [STIL.md](STIL.md) folgt (Fachwort, Verb, Kürzel, keine
   Wiederholung des Pakets) und jede Datei wie ihre Sache heißt;
2. öffentlich nur ist, was ein anderes Paket beim Namen nennt oder was in der Signatur eines
   solchen Namens steht;
3. seine Kommentare in ein bis zwei Sätzen das Warum nennen, ohne Datum und Vorgeschichte;
4. die Tests in seinem Ordner dieselben Namen und denselben Stil der Kommentare tragen;
5. seine Zeile in `stil_bestand_test.go` entfernt ist.

**Beleg je Paket:** Umbenannt wird mit dem Werkzeug der Sprache (`gopls rename`), das jede
Nennung über die Typprüfung findet. Danach laufen die ganze Go-Suite an der Datenbank, der
Linter, die Prüfung auf toten Code und gosec. Adressen, die Texte der Datenbankabfragen und die
Feldnamen in JSON sind vor und nach dem Schritt dieselben (Vergleich aller Zeichenketten des
Programmcodes). Ein Commit, der nur Kommentare ändert, wird am Code ohne Kommentare verglichen.
Eine Attrappe, die eine Schnittstelle nur in den Testdateien eines anderen Pakets erfüllt,
zieht das Werkzeug nicht mit (`tuertest.EinstellungenAttrappe`); `go vet ./...` baut auch die
Tests und meldet die Stelle. Drei Stellen sieht auch `go vet ./...` nicht oder erst am Fehler:

- Dateien hinter einem Build-Tag (`repository/raster_*_pg_test.go`) und Skripte mit
  `//go:build ignore` (`scripts/generalprobe/nachtsicherung.go`): `bedingte_dateien_test.go`
  übersetzt sie in jedem Lauf der Suite, `sichtbarkeit_ratsche_test.go` zählt die Skripte als
  Nutzer.
- Wandelt ein Paket die Zeile eines anderen in seinen eigenen Typ um (`AbgaengerZeile(z)` in
  `api/mahnwesen`, `auskunft.Stammdaten(*zeile)`), müssen die Felder beider Typen gleich
  heißen: Ein umbenanntes Feld zieht das Feld des Zwillings mit, das Feld in JSON bleibt.
- Kommentare im Text einer Abfrage (`-- …`) stehen in einer Zeichenkette. Sie werden von Hand
  gesucht und als eigener Commit nachgezogen, weil sich dabei der Text der Abfrage ändert.

**Namen als Text:** Dokumente nennen 405 Namen des Programmcodes, Testdateien 524 in
Zeichenketten (Listen, an denen Prüfungen hängen, und Fehlermeldungen), die Liste des toten
Codes 86, die Kommentare für die Schnittstellenbeschreibung rund 50 Typen. Dort meldet der
Compiler nichts. Jeder umbenannte Name wird im ganzen Repository gesucht und im selben Commit
nachgezogen; eine Prüfung, die ihn als Text führt, wird danach einmal rot gesehen. Die Suche
nimmt auch den Go-Code mit: Kommentare anderer Pakete nennen den Namen, und das Werkzeug zieht
nur das erste Wort über der Erklärung nach. Eine Prüfung, die einen Aufruf an seinem Namen
erkennt, bekommt vor dem Umbenennen ein Merkmal, das den Namen überlebt
(`repository/einstellungen_lesefehler_ratsche_test.go` nimmt den Typ des Ergebnisses). Einen
umbenannten Dateinamen in Dokumenten und Kommentaren meldet `go test ./docs/`. Steht der
Kommentar über dem Feld eines Anfrage- oder Antworttyps, steht er auch in `docs/docs.go`:
danach `swag init -g main.go -o docs`. Führt ein anderes Paket einen eigenen Bezeichner
desselben Namens (einen Erzeuger oder ein Feld), bleibt er: Die Suche als Text gilt deshalb
nur im Ordner des Pakets und für die Form `paket.Name`.

**Was seinen Namen behält:** Adressen, Feldnamen in JSON, Tabellen und Spalten
([STIL.md](STIL.md), Abschnitt 9).

**Danach, je als eigener Schritt:**

- Die geteilten Bauteile `CoverPeek` und `TabelleSortKopf` heißen noch englisch: CLAUDE.md und
  der Text eines Hooks in `.claude/settings.json` (nicht eingecheckt) zählen die Bauteile mit
  Namen auf. Beide Stellen zusammen mit dem Umbenennen ändern, etwa `CoverVorschau` und
  `TabelleSortierKopf`.

### 5.67 Umfang senken: was sich löschen lässt

Gemessen am 15.10.2026 an den eingecheckten Dateien (rund 362.000 Zeilen, davon 130.400 Go-Tests
und 79.700 Go-Programm): Toten Code gibt es kaum. In Go hält ihn `scripts/deadcode_gate.sh` bei
null, auf der Oberfläche ist jede Datei von `main.js` aus erreichbar. Die eine Route ohne Aufrufer,
der Elternbrief je Schadensfall, ist entfernt. Der Zuwachs steckt in Tests, die je eine Zusage
halten; sie zu löschen hieße, Regeln zu lockern. Offen:

- **SQL-Testdaten:** 730 Anweisungen mit 3.443 Zeilen legen Titel, Exemplare, Schüler, Leser,
  Konten und Ausleihen selbst an, obwohl `internal/testdaten` dafür Helfer hat. Wo einer passt,
  wird aus im Schnitt 4,6 Zeilen eine; zusammen 2.000 bis 2.700 Zeilen weniger. Paket für Paket
  zusammen mit dem Umbau nach [STIL.md](STIL.md).

### 5.70 Vorgeschichte in Kommentaren der Oberfläche

Gezählt am 16.10.2026: 448 Kommentarzeilen mit Datum in 219 der 459 Svelte- und JS-Dateien
unter `frontend/src` (ohne Tests; Zeilen in `//`, `/* */` und `<!-- -->`), etwa „Hier lag bis
zum 16.09.2026 ein Vollbild …" in `App.svelte` oder „stand hier bis zum 24.08.2026" in
`menue.js`. Die Regel steht in `CLAUDE.md`, Abschnitt 4: kein Datum, keine Vorgeschichte. Im Go-Code hält sie `stil_ratsche_test.go` mit
leerem Bestand; für die Oberfläche fehlt eine Prüfung. Vorschlag: in
`frontend/src/lib/frontend-hygiene-sprache.test.js` je Datei die Kommentarzeilen mit Datum als
Bestand einfrieren, der nur sinken darf (das Datum einer Quelle ausgenommen, wie im Go-Code),
dann Datei für Datei die Kommentare auf das Warum kürzen, nach jeder Datei `npx svelte-check`
und `npx vitest run`.

---

### 5.71 Abgleich mit Googles Regelwerk für Go: was noch abweicht

Gemessen am 16.10.2026 mit einem erweiterten Satz von golangci-lint (revive, errorlint,
containedctx, contextcheck, thelper, ireturn und weitere; die Konfiguration ist nicht
eingecheckt) und mit `go/types` über alle Pakete. Was schon gilt, steht mit seiner Prüfung in
[STIL.md](STIL.md); hier steht der Rest, je mit der Stelle bei Google (gelesen am 16.10.2026).
Funktions- und Dateilänge sind kein Punkt: Google setzt keine Höchstlänge für Funktionen, und für
Dateien gilt „it is usually not a good idea to have a single file with many thousands of lines"
(Best Practices, „Package size"); die größte Go-Datei hat 629 Zeilen, die Komplexität hält
`komplexitaet_ratsche_test.go`.

- **Schnittstellen beim Hersteller.** Decisions, „Interfaces": „Avoid creating interfaces until
  a real need exists", „The consumer of the interface should define it (not the package
  implementing the interface), ensuring it includes only the methods they actually use" und
  „Functions should take interfaces as arguments but return concrete types." Die Regel steht in
  [STIL.md](STIL.md) 14; `ireturn` hält sie, die fünf Erzeuger, die noch eine Schnittstelle
  liefern, stehen als Ausnahmen in `.golangci.yml`. Einen Bedarf schafft hier nur eine Attrappe
  im Test; welcher Typ jede erfüllt, ist mit `go/types` über alle Pakete samt Tests gemessen.
  Die Schnittstelle fällt, der Typ `pg…Abfragen` bekommt ihren Namen, der Erzeuger liefert ihn.
  Freigegeben ist ein Plan in Stufen, je Schnittstelle ein Commit, je Stufe ein Push und deine
  Freigabe für die nächste:
  3. Attrappe als Feld eines Dienstes: `LeserAbfragen`, `KatalogAbfragen`, `AusleihAbfragen` und
     `ProtokollAbfragen` in `internal/service`, `ProtokollAbfragen` auch in `jobs`; die Tests
     setzen die Felder direkt (`&ausleihDienst{leserAbfragen: …}`). Ebenso `service.AusleihDienst`
     am Feld `ausleihe` von `OmniboxDienst` (Attrappen in `omnibox_test.go` und
     `littera_etikett_routing_test.go`); [ARCHITEKTUR.md](ARCHITEKTUR.md) 5.4.1 nennt in
     `ausleihe.go` die Schnittstelle. `NachbuchDienst` bucht über sein Feld
     `ausleihe *ausleihDienst`; wird der Typ öffentlich, zieht das Feld mit. Kleine Schnittstellen in beiden
     Paketen, benannt nach dem Vorgang wie in [STIL.md](STIL.md) 14; so stoßen sie nicht mit den
     Feldern und Parametern zusammen, die schon `leserAbfragen` heißen.
  4. Abschluss: keine Ausnahme mehr in `.golangci.yml`; STIL.md Abschnitt 2 nennt `…Abfragen`
     ohne `pg`.

  Es bleiben `Datenbank`, `KontoSchreiber` und `SpurenDatenbank` (Parameter von Funktionen in
  `repository/`) und `tuer.Mailversand` (mehrere Umsetzungen). Nachweis je Stufe: `go build`,
  `go vet`, golangci-lint wie in CI, die volle Suite mit `TEST_DATABASE_URL`, `go test -list`
  vorher und nachher gleich, die Funde von `ireturn` sinken um genau die der Stufe. Ein
  nil-Zeiger in einer Schnittstelle ist nicht nil: Je Stufe jede Prüfung auf nil an den
  geänderten Parametern und Feldern lesen.
- **Tests.** Decisions, „Test helpers": Eine Hilfsfunktion ruft `t.Helper()`, damit die Meldung
  die Zeile des Tests nennt; 26 tun es nicht. Lokale Namen in Tests zählt `stil_ratsche_test.go`
  nicht: 31 englische Namen in 69 Testdateien (`bookRepo` in 19, `loanSvc` in 12, `studentRepo` in
  11, `staffID`, `activeLeserID`). Weg: je eine Prüfung, dann Datei für Datei.

### 5.72 Dateigröße der Oberfläche: der Omnibox-Store

`frontend-hygiene-dateigroesse.test.js` zählt Komponenten und JS-Module; im Bestand stehen 17
JS-Module über 200 Zeilen, das größte `stores/omnibox.svelte.js` mit 862. Googles Maßstab für
Dateien (Best Practices, „Package size"): „files should be focused enough that a maintainer can
tell which file contains something". Der Omnibox-Store trägt Eingabe, Auflösung, Warteschlange,
Sperrdialog, Leserauswahl und Zusammenführen ([ARCHITEKTUR.md](ARCHITEKTUR.md) 5.3). Freigegeben
ist ein Plan in Stufen, je Stufe ein Commit, ein Push und deine Freigabe für die nächste:

- **Stufen 4 bis 6:** Je ein Teil in eine eigene Datei, die lose gekoppelten zuerst:
  `omniboxAntwort.svelte.js` (was eine Antwort des Servers am Zustand der Theke ändert),
  `omniboxOhneNetz.svelte.js` (Schnappschuss, Ablage in der Warteschlange, Ausweis-Merker),
  `omniboxReihe.svelte.js` (`fuehreAktionAus`, die Reihe der Scans, der einzige Versand an
  `/api/action`). In `omnibox.svelte.js` bleiben der Zustand der Theke, Rückfrage und Fokus
  (`frontend-hygiene-theken-fenster.test.js` liest dort `rueckfrageOffen`), Zusammenführen,
  `gibZurueck` und die Fassade.
- **Stufe 7:** Suche und Rückmeldung werden Unterobjekte der Fassade (`omniboxStore.suche.…`,
  `omniboxStore.rueckmeldung.…`), damit der Aufruf die Datei nennt: 36 Stellen in 4 Dateien
  und 33 in 7, Tests eingeschlossen. Danach den Bestand der Größen-Ratsche austragen und
  ARCHITEKTUR.md 5.3 nachziehen.

Regeln für jeden Schnitt: Code wird verschoben, nicht umgeschrieben. Ein Zustand lebt in genau
einem Teil; die anderen bekommen Funktionen, kein geteiltes Objekt. Der Ausweis-Merker schickt
über dieselbe Funktion wie der Scan: Es bleibt ein Weg zu `/api/action`, und die Erlaubnisliste
von `frontend-hygiene-action-endpunkt.test.js` tauscht die Datei, statt zu wachsen. Bauteile
und Tests laden nur `omnibox.svelte.js` (`frontend-hygiene-omnibox-teile.test.js`), sonst
entginge ein Import am Teil vorbei `frontend-hygiene-thekenzeitgeber.test.js`. Jeder Teil
liefert sein Objektliteral mit `@satisfies` ([ARCHITEKTUR.md](ARCHITEKTUR.md) 5.3); dann
meldet svelte-check jeden Zugriff auf einen Namen, den der Teil oder die Fassade nicht hat, auch
in Bauteilen. Darauf baut Stufe 7.

Nachweis je Stufe: die Store-Tests (13 Dateien, 94 Fälle) und `scanOhneFokus.test.js` ohne
Änderung grün, svelte-check ohne Fehler und Warnung, vitest ganz, am frisch gebauten Stack
(`scripts/stack-neu.sh`) die Browser-Tests der Theke: `theke-scan-reihe`,
`theke-schnellrueckgabe`, `theke-suchleiste`, `theke-bestand-im-treffer`, `sperre-scan`,
`scanner-fokus-menue`, `kiosk-scannerfokus` (tippt blind und zählt in der Datenbank),
`escape-kurzbefehl`, `auflagen-theke`. Nach Stufe 7 eine Probe mit dem echten Handscanner an
der Theke, durch dich.

### 5.73 Statistik: ein Lesefehler sieht aus wie ein leerer Bestand

Scheitert das Lesen von Rennern, Ladenhütern oder Monatstrend, setzt `api/system/statistik.go`
den Abschnitt leer und schreibt den Fehler nur ins Log. Die Seite zeigt dann den Leerzustand; bei
den Ladenhütern steht in Grün „Keine Ladenhüter — der Bestand ist in Bewegung"
(`Statistik.svelte`, Snippet `keineLadenhueter`). Raster, Frage 5: Die Seite meldet Erfolg
anhand der leeren Liste, nicht anhand der Wirkung. Weg: Die Antwort nennt die Abschnitte, die
nicht gelesen wurden, und die Seite zeigt dort statt des Leerzustands einen Hinweis; vorher die
zuständige Seite von M3 lesen.

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
ausgegebenen Tabellen. Der Test vergleicht den vollen EAN-13 jedes Etiketts mit der Rechnung
(`pruefeEtiketten`). Exemplare, deren Etikett
schon ein anderes trägt (2010: 8), stehen nach dem Lauf unter „Fehlende Etiketten"
([SCRIPTS.md](SCRIPTS.md), Abschnitt 1).

**Lesergruppen ohne Zuordnung vor dem Umstiegstag klären.** Der Lauf übernimmt jeden
Leser ([SCRIPTS.md](SCRIPTS.md), Abschnitt 1); eine Lesergruppe ohne Zuordnung hält
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
(nachgesehen am 28.09.2026): `schreibeBackupNachS3` in `jobs/backup.go` lädt jede Nachtsicherung
hoch, die Rotation gilt nur dem lokalen Verzeichnis. Ohne Löschregel am Speicher bliebe dort jede
Sicherung mit allen Personen ihres Stands unbegrenzt liegen. Beim Einrichten eine Löschregel am
Speicher setzen, die der Aufbewahrung der Nachtsicherung folgt (die jüngsten 14, dazu je
Kalenderwoche eine für 12 Wochen; `jobs.AufbewahrteNaechte`, `jobs.AufbewahrteWochen`), oder die Rotation
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

- Liegen im Volume noch Fotos aus der Zeit vor dem 13.06.2026? Sie wären unverschlüsselt und
  stünden schon in der Datenbank: `docker compose exec backend ls -la uploads/fotos`. Liegen dort
  Dateien: `docker compose exec backend sh -c 'rm -f uploads/fotos/*.jpg'`.
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
  Erwartet: Die Meldung oben rechts („Die ISBN ist ungültig") liegt nicht über dem Knopf.
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
Zähldatum unter „Exemplare". Ein Feld „Beschreibung / Klappentext" gibt es
nicht; „Speichern" bleibt am neuen Bibliotheksbuch bedienbar und führt ohne
Signatur zum Feld. Die ISBN ist freiwillig, Pflicht ist der Titel (Stern
an der Beschriftung): einen Titel ohne ISBN öffnen, die Signatur ändern und speichern; ein
Medium ohne ISBN neu anlegen und danach noch einmal mit demselben Titel und Autor — die
Maske fragt dann „Ist es dasselbe Medium?". Die zehn- und die
dreizehnstellige ISBN sind dieselbe Nummer: ein Buch mit der zehnstelligen ISBN vom Titelblatt
anlegen — gespeichert steht die dreizehnstellige im Feld — und danach in „Neues Buch"
seinen Strichcode scannen; die Maske fragt „Vorhandenen Titel öffnen?".

**Der Ausweisdruck am Kartendrucker.** In der Leserdatei zwei Leser ankreuzen, „Ausweise
drucken". Erwartet: Jede Karte trägt nur den Ausweis, oben steht keine Reiterzeile, und die
Nummer unter dem Strichcode steht auf der Karte. Dasselbe mit „Testdruck Vorderseite" im
Druck-Center, Reiter „Schülerausweise". Gemessen ist das in der Druckansicht des Browsers
(`e2e/ausweis-druckseite.spec.js`), am Kartendrucker noch nicht.

**Das Mahnwesen ansehen:** Kinder anhaken, „Mahnbriefe drucken". Es
kommt je Kind der Brief an die Eltern mit Anschrift; die Liste zeigt danach
„1× gemahnt, zuletzt …". „Liste drucken" druckt die Liste als Tabelle, je
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
