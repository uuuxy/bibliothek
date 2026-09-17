# Offene Arbeit

Stand: 17.09.2026

**Die eine Liste.** Hier steht alles, was noch zu tun, zu prüfen oder zu entscheiden ist — Code,
Betrieb und Schule. Einen zweiten Ort gibt es nicht: Das Befund-Register (`docs/befunde.md`) und
die Issues #593, #594, #597, #598, #599 und #600 sind am 13.09.2026 hierher umgezogen. Erledigtes
wird gelöscht, nicht archiviert (entschieden am 15.09.2026): Die Geschichte steht in den
Commit-Nachrichten, in `git log -p docs/OFFEN.md` und in den geschlossenen Issues.

Andere Dokumente erklären (Konzept, Anleitung, der Katalog der Bugklassen in
[sweeps.md](sweeps.md)), führen aber keine eigene Offen-Liste.

---

## Was jetzt dran ist — in einfachen Worten (Stand 17.09.2026)

Mehr als diesen Block muss niemand lesen, um zu wissen, was als Nächstes kommt. Alles darunter
ist die ausführliche Fassung mit Begründungen; sie ändert nichts an dieser Reihenfolge.

**Neu am 17.09.2026: Das Medienzentrum hat das Programm gesichtet.** Ein Protokoll vom
16.09.2026 listet zwölf Punkte und schließt mit: Sind sie abgestellt, wäre das Programm für
Schulen nutzbar. Das ist damit das Gate, an dem alles andere hängt — es steht vollständig in
**Abschnitt 9**, jeder Punkt am Code nachgeprüft.

Zwei davon waren schon gelöst (der Scanner ging weder per Handgerät noch per Kamera — das war
gestern Abend und heute früh die ganze Arbeit; und Littera-Barcodes kann das Programm lesen).
Zwei sind neu und wiegen schwer: Erstens sind **alle Ausweise und Etiketten, die vor heute
gedruckt wurden, weiterhin unlesbar** — der Strichcode trug eine Ziffer zu viel. Zweitens
verlangt das Medienzentrum **Bücher, die über mehrere Jahre bei einem Kind bleiben** — genau
das, was wir vorgestern zu streichen beschlossen hatten. Diese Entscheidung ist gestoppt.

**Vier Fragen dazu stehen in 9.7** und halten den Rest auf.

**Was seither gebaut ist (17.09.2026, nachmittags):** Das Programm rechnet jetzt aus, was ein
Buch heute noch wert ist — und schreibt dazu, wie die Zahl entstanden ist. An jedem Exemplar in
der Buchakte steht der Betrag samt Begründung, ein beschädigtes Buch bekommt einen Prozentwert
für seinen Zustand (der den Betrag mindert und ein Buch nicht aus dem Verkehr zieht), und in den
Einstellungen kann die Schule wählen, ob der heutige Listenpreis oder der bezahlte Einkaufspreis
die Grundlage ist. Damit sind die drei Punkte der Anforderungsliste zur Abwertung abgearbeitet.

**Die drei Fragen dazu sind entschieden** (17.09.2026): An der Theke wird kein Wertverlust
eingetragen — sie bleibt ein Scanfeld. Für die Schülerbücherei zählt der Abschlag für
Beschädigung ebenfalls, und ersetzt wird dort der heutige Preis statt des damals bezahlten: Die
Benutzungsordnung verlangt den Neuwert, und zur Bücherei sagt das Land nichts. Dabei kam ein
Fehler mit heraus, der niemandem aufgefallen war: Der Brief rechnete Büchereibücher nach der
Staffel des Landes und nannte für ein zehn Jahre altes Buch 1,40 € statt 14,00 €.

---

**Was DU tun kannst — der Reihe nach:**

1. **Sieben Fragen beantworten** (Abschnitt 4: 4.7 bis 4.11, 4.13, 4.14). Sie sind der Rest der
   Fragerunde vom 16.09.; ohne sie bleiben sieben kleine Bauarbeiten liegen. Jede hat einen
   Vorschlag danebenstehen.
2. **Zwei Zahlen vom Server holen.** Beide stehen als fertiger Einzeiler in der Liste: Wie viele
   Titel tragen ein Ziel-Jahrgangsfeld (4.3)? Und wie viele Leser stehen ohne Ausweisnummer da
   (5.16 E)? Erst danach darf ich das Feld streichen und die Nummern nachtragen — beides ändert
   echte Daten.
3. **Zwei Umbauten freigeben**, die ich vorbereitet, aber bewusst NICHT gebaut habe, weil sie
   die Datenbank ändern: die Ausweisnummer schon beim Anlegen eines Kontos (5.16 E) und die
   eigene Spalte für die Karenz-Uhr (4.12). Beide sind entschieden, beide brauchen eine
   Migration — und die schreibt Nummern bzw. Daten, die niemand zurücknimmt.
4. **Dein Nachweis von Hand für die Theke ohne Netz** (Abschnitt 2): Netz kappen, Bücher aller
   Formen und zwei Ausweise scannen, 20 Minuten warten, Netz zurück, Meldungen ansehen. Dazu
   der Nachweis für den Server.
5. **15 Minuten durch die Leserdatei gehen** (5.16 A): Stimmen die Wörter, fehlt dir etwas?
6. **Neu am 17.09.: Schulbücher in neuer Auflage** (4.18). Die Richtung ist entschieden — der
   Bedarf rechnet über die Auflagen hinweg, die Ausgabe warnt bei gemischten Auflagen, und
   zusammengelegt wird nichts. Gebaut ist davon nichts: Der erste Schritt ist klein (das Feld
   „Auflage"), der dritte ändert das Schema und braucht deine Freigabe.
7. **Liegt bei anderen** (Abschnitt 8): die Anfragen an Schule, Schulamt und Schulträger. Hier
   ist nichts zu tun außer nachzufragen, wenn nichts kommt.

**Was in der Nacht vom 16. auf den 17.09. gebaut wurde** — alles mit Gates, alles auf `main`:

- Eine Einstellung, die außerhalb ihres Bereichs liegt, wird jetzt **abgelehnt statt still
  ersetzt**. Vorher wurde aus einer getippten 0 eine 5, gemeldet als „gespeichert" — und die
  Sperr-Automatik liess sich mit einer großen Zahl unbemerkt abschalten.
- Ein **geänderter Warenkorb** wird nicht mehr still zur alten Bestellung, wenn die Antwort
  einmal verloren ging.
- Die **Theke sagt es**, wenn ihre offenen Vorgänge so nicht ins System kommen (etwa weil die
  gerade angemeldete Person nicht buchen darf) — vorher lief der Versuch stumm jede Minute ins
  Leere. Die Buchliste für den Betrieb ohne Netz frischt sich stündlich auf.
- **Sieben Datumsangaben** rechneten in der Zeit des Servers (UTC) statt in der Zeit der Schule:
  die Volljährigkeit im Bescheid, der Mahnlauf, „heute zurückgegeben", der Stornierungsgrund und
  drei Mails — Mahnliste, Kontoauszüge und die Bestellung an den Händler trugen zwischen
  Mitternacht und 2 Uhr das Datum von gestern. Dagegen gibt es jetzt eine Ratsche.
- **Kein Passbild mehr aus dem Papierkorb**, ein **Klassenwechsel rechnet das Abgangsjahr wieder
  neu**, das **Massenlöschen von Titeln verlangt das Löschrecht**, und die Sperrmeldung eines
  Geräts **erfindet kein Kind mehr**.
- Die Tür „Defekt melden" ist **gestrichen** (sie war halb kaputt), und das API-Inventar meldete
  drei benutzte Routen zu Unrecht als tot — auch das ist behoben.
- **In der CI** wird jetzt auch die Formatierung geprüft (lief nur im Hook auf deinem Rechner),
  und ein versioniertes Image entsteht nur noch auf grüner CI. Der Suchtest, der sich auf jedem
  fremden Rechner still übersprungen hat, läuft wieder.
- **Kleinkram mit Wirkung:** Das Demo-Löschskript bricht ab, statt Forderungen echter Leser
  mitzunehmen. Ausgesonderte Exemplare bekommen kein Etikett mehr (sie verschoben auf dem Bogen
  alle folgenden). Im Kollegiums-Portal sagt ein gescheiterter erster Abruf das auch, statt
  „keine Anliegen" zu zeigen — sonst schickt jemand seinen Wunsch ein zweites Mal. Der Toast der
  Akte zeigt den Satz des Servers statt seines JSON-Rumpfs, „Erneut versuchen" gibt es nur noch,
  wenn es etwas zu wiederholen gibt, und ein Fehler beim endgültigen Löschen wird nicht mehr
  pauschal als „da ist noch etwas offen" gemeldet.

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

1. **Die sieben offenen Fragen** aus Abschnitt 4 (4.7–4.11, 4.13, 4.14) — sie halten sieben
   kleine Bauarbeiten auf und kosten zusammen eine halbe Stunde.
2. **Die zwei Messungen am Server**: Ziel-Jahrgang (4.3) und Leser ohne Ausweisnummer
   (5.16 E). Danach die beiden Migrationen, die daran hängen — und die Karenz-Spalte (4.12).
3. **Abschnitt 2** Offline-Betrieb der Theke: gebaut, samt Meldungsliste und Doku. Offen sind
   nur noch die Nachweise am Stack (2.3) — Stufe 1 und 3 von Hand, Stufe 2 über die Tür.
4. **5.16** Leserdatei: gebaut. Offen ist dein Blick auf den Stand und die Ausweisnummer (E).
5. **5.5–5.9**, **5.12** und die B-Punkte aus **5.15** — kleine B-Commits, gebündelt.
6. Mahnverfahren: Vor dem ersten echten Bescheid **5.2** und **4.5** (E4), dann **4.4** (E6) und
   **5.13** Stufe 3 (5.3).
7. Nach der Antwort zu E5 (**8.3**): **5.4**.
8. **5.10** (Gates und Werkzeuge) und Abschnitt 6 nur mit Anlass — mit EINER Ausnahme, die
   vorgezogen gehört: Tests für die Werkzeuge der Littera-Übernahme und den Schlüsselwechsel.
   Sie laufen einmal gegen echte Daten, und bis heute hat keines von ihnen einen Test.

**Parallel auf der Schulseite:** Abschnitte 7 und 8 — zuerst S3 (7.3), das Littera-Backup (7.2), die
Anfragen E1, E2, E5 (8.1–8.3), B3 und B4 (8.5) und ein Termin für die Abnahmen (7.7). Einen echten LUSD-Import erst nach der
Littera-Übernahme (7.2).

---

## 1. Sofort (Kategorie A)

Nichts offen (Stand 15.09.2026).

---

## 2. Laufende Arbeit: Offline-Betrieb der Theke

**Ziel (entschieden am 13.09.2026):** Bei einem Verbindungsabbruch geht der Betrieb an der Theke normal
weiter.

**Entschieden am 13.09.2026:**

- Offline werden alle Buchformen gespeichert: `B-…`, nur Ziffern (auch EAN-13), `LMF-…`.
- Ein offline gescannter Schülerausweis kommt in die Warteschlange; die folgenden Bücher werden
  ihm zugeordnet, der Server löst ihn beim Nachbuchen auf. Keine Schülerdaten auf dem Rechner.
  Sperren prüft der Server beim Nachbuchen und meldet Abgelehntes mit Barcode.
- (a) An der Theke ein nicht blockierendes Band statt des Vollbilds; Scannen geht weiter.
- (b) Lehrerausweis und geladene Lehrkraft werden offline wie ein Schüler zugeordnet; die Bücher
  gehen beim Nachbuchen als Handapparat an die Lehrkraft.
- (c) Beim Nachbuchen wird die Wirklichkeit gebucht: Lag das Buch noch bei jemand anderem, dort
  zurücknehmen und neu ausleihen; beim selben Schüler nichts umkehren; beides steht im
  Nachbuch-Bericht, nichts verschwindet still.
- (d) 502/503/504 gelten wie ein Netzausfall.
- Arbeitsweise: in Stufen — (1) vorhandene Fehler, (2) Server, (3) Theke, Warteschlange und Band.
  Je Stufe Rot-Test am alten Code, volle Suite mit Postgres, Nachweis am Stack und im Browser,
  dann die Freigabe.

**Entschieden am 13.09.2026, zweite Runde:**

- Nach einem Neuladen ohne Netz bleibt die Anmeldemaske der Rückfall. Die gespeicherten
  Offline-Scans bleiben auf dem Rechner und werden nach der nächsten Anmeldung mit Netz
  nachgebucht.
- Nachbuch-Meldungen liegen am Server, sichtbar nur mit dem Recht `view_students` (nicht für
  Helfer). Erledigte werden nach der Lesehistorie-Frist gelöscht, höchstens nach 30 Tagen. Offene
  erscheinen nach 14 Tagen als Warnung in der Betriebsbereitschaft. Beim Zusammenführen wandern
  sie mit.
- Gebucht wird mit dem Scan-Zeitpunkt, höchstens der Serverzeit: Ausleih- und Rückgabedatum,
  Frist, Mahnwesen und Lesehistorie rechnen ab dem Scan.
- Ohne Netz sperrt die Theke nicht; geleert wird sie weiter nach 5 Minuten. Kommt das Netz zurück
  und war länger als 15 Minuten niemand da, sperrt sie sofort.
- „Sicherung speichern" bleibt auch auf Anmeldemaske und Sperrbildschirm sichtbar; eingespielt wird
  nur angemeldet.
- Ziffernfolgen: Der Rechner hält eine Liste aller Buch-Barcodes (keine Personendaten, bei der
  Anmeldung aktualisiert). Steht eine Ziffernfolge darauf, ist sie ein Buch; sonst gilt sie als
  unklar und sperrt die Zuordnung, bis ein eindeutiger Ausweis kommt.

**Daraus abgeleitet** (Entscheidung (c) und heutiges Verhalten der Online-Theke):

- Beim Vorbesitzer wird immer zurückgenommen. Scheitert die neue Ausleihe an Sperre, Limit oder
  Vormerkung, wird nur sie als nicht gebucht gemeldet.
- Hat ein abgebrochener Online-Versand schon eine Rücknahme gebucht, holt das Nachbuchen die
  fehlende Ausleihe nach (unter Wächter und Schranken).
- `B-` und `LMF-` werden auch ohne geladene Regeln oder Buchliste gespeichert.
- `/api/action/batch` bleibt eine Version länger bestehen, für Theken-Tabs mit altem Stand.

**Maßstab:** Nicht die Zahl der Randfälle entscheidet, sondern ob an der Theke je ein Buch bei
einem Kind liegt, das im System frei ist, oder umgekehrt. Alles, was das verhindert, gehört hinein.
Alles andere kann warten, bis es im Betrieb vorkommt.

**Stufe 1 ist NICHT gebaut — richtiggestellt am 16.09.2026, 20:30 Uhr, am Stack gemessen.**
Der zweite Durchgang des Stufe-1-Nachweises (Netz gekappt, im Kiosk gescannt) endete mit zwei
Toasts „Netzwerkfehler", und der Scan war weg. Ursache, am Code belegt:

- `speichereOfflineAktion` (`stores/omnibox.svelte.js`) nimmt bis heute NUR `B-`-Barcodes in die
  Warteschlange; alles andere — Littera-Ziffern, `LMF-`, jeder Ausweis — bekommt einen nackten
  „Netzwerkfehler" und wird verworfen. Das ist Wort für Wort Punkt 1 von 2.1, also genau das, was
  Stufe 1 beheben sollte. Die sieben Commits haben die Funktion angefasst (`65f9a998` hat ihren
  Fehlerfall umgebaut), die Beschränkung aber nie entfernt.
- Die Tür, die dem Rechner sagt, welche Ziffernfolge ein Buch ist, EXISTIERT am Server
  (`GET /api/action/buchbarcodes`, `RequirePermission("perform_actions")`) und wurde am 15.09.
  noch verbessert (ausgesonderte Exemplare). **Kein einziger Aufruf im Browser** — Bugklasse „Nie
  verdrahtet". Die lokale Buch-Barcode-Liste aus der Entscheidung vom 13.09. gibt es nicht.
- Kein Gate konnte das sehen: JEDER Fall in `stores/omniboxOffline.test.js` scannt `B-10234`.
  Geprüft ist genau der eine Weg, der funktioniert.

Folge für den Betrieb: Solange das so ist, ist der Satz im Offline-Band („Scannen geht weiter,
die Buchungen folgen von selbst") ein Versprechen, das das System nicht hält. Und weil nie etwas
in die Warteschlange kommt, erscheint auch „Sicherung speichern" nie — der Bediener sieht nur den
Knopf zum Einspielen und fragt sich zu Recht, wo er denn speichern soll. Beide Beschwerden vom
16.09. haben diese EINE Ursache.

Daraus die Lehre, die teurer war als der Fehler: „gebaut" hiess hier „Commits liegen vor". Ein
Nachweis von Hand hat nie stattgefunden, und die Testfälle bestätigten nur den funktionierenden
Fall. Stufe 3 wurde am 16.09. auf diese Grundlage aufgesetzt — was dort gebaut ist (Band statt
Vollbild, keine Sperre ohne Netz), gilt weiter und ist getestet; es zeigt nur eine Warteschlange,
die noch nichts annimmt.

**Nachtrag, 16.09.2026 gegen 20:45 — Stufe 2 ist am Browser EBENSO wenig angeschlossen.**
Beim Planen des Stufe-1-Baus gemessen, weil die Frage aufkam, ob die Tür einen offline
gescannten Ausweis annimmt. Sie tut es (`NachbuchenEintrag.AusweisBarcode`), und mehr:

- `POST /api/action/nachbuchen` ist geroutet und vollständig — Idempotenz je Schlüssel,
  Uhrversatz des Theken-Rechners, Ausweis-Auflösung, die sieben Ergebniswörter,
  Meldungsliste. **Kein Aufruf im Browser.**
- `stores/offlineSync.svelte.js` schickt weiterhin an `POST /api/action/batch` — die Tür, die
  laut Entscheidung vom 13.09. nur noch „eine Version länger" für Theken-Tabs mit altem Stand
  bestehen bleibt und deren Rückbau oben schon eingeplant ist.

Damit steht am Server ein vollständiger Offline-Betrieb, von dem an der Theke nichts ankommt.
Zweimal dieselbe Form (Bugklasse „Nie verdrahtet"), und beide Male sagte die Stand-Angabe
„gebaut", weil Commits vorlagen.

**Was daraus fuer die Reihenfolge folgt.** Der Bau zerfaellt in drei Schritte, und nur der
erste bringt den Nachweis zum Laufen:

- **A — die Theke nimmt an.** Barcode-Liste im Browser halten (die Tür liefert sie mit ETag und
  gepackt), Scans offline einordnen (`B-`/`LMF-` = Buch, `A-`/`S-`/`L-` = Ausweis, Ziffern über
  `litteraEtikett.js` zurückrechnen und in der Liste nachschlagen, sonst „unklar"), alle Formen
  und den Ausweis in die Warteschlange lassen. Danach landen Scans, und „Sicherung speichern"
  erscheint überhaupt erst.
- **B — die Buchungen kommen richtig an.** Der Sync schickt an `/api/action/nachbuchen` statt an
  `/api/action/batch`, mit Scan-Zeitpunkt, Schlüssel und Ausweis-Barcode.
- **C — was nicht durchging, wird sichtbar.** Die Meldungsliste aus Migration 117.

**Offen geblieben (16.09.2026, beim Nachweis gefunden): Der SERVER nimmt eine Nummer nur in
genau der Schreibweise an, in der sie gespeichert ist.** Gemeldet wurde „s-10001 hat nicht
funktioniert"; die Ursache liegt nicht im Offline-Teil. `internal/service/omnibox_service.go`
vergleicht die Vorsilben mit `strings.HasPrefix` gegen `"A-"`, `"S-"`, `"L-"`, `"B-"`, `"G-"`,
und `GetLeserByBarcode` schlägt mit `WHERE barcode_id = $1` exakt nach. Eine Schicht, die die
Eingabe vorher vereinheitlicht, gibt es auf keiner Seite. Wer die Nummer von Hand tippt, weil
eine Karte nicht mehr lesbar ist, traf damit weder mit noch ohne Netz etwas.

Die Theke vereinheitlicht seit dem 16.09.2026 selbst (`frontend/src/lib/scanEinordnen.js`,
`normalisiereScan`) — und zwar nur, wenn hinter der Vorsilbe eine Ziffer steht, sonst würde aus
der Suche nach „s-bahn" ein Ausweis. Damit ist der Weg über die Theke geheilt, mit Netz wie
ohne. **Der Server selbst bleibt empfindlich:** Ein anderer Aufrufer (Skript, zweite Oberfläche,
direkter API-Aufruf) läuft weiter ins Leere. Ob das dort ebenfalls geheilt wird, ist eine eigene
Entscheidung — eine Suche über `upper(barcode_id)` nutzt den vorhandenen Index nicht mehr, und
der hält die Eindeutigkeit der Ausweisnummern.

**Stand 16.09.2026, spät: Schritt A und Schritt B sind gebaut** (vier Commits, je ein Rot-Test
am alten Code; volle Go-Suite mit Postgres, Frontend-Suite, Lint und `svelte-check` grün). Die
Theke hält die Buch-Barcode-Liste im Browser, ordnet jeden Scan selbst ein
(`frontend/src/lib/scanEinordnen.js`, der Zwilling des Server-Switch), nimmt alle Buchformen und
jeden Ausweis an, und der Sync schickt an `/api/action/nachbuchen` statt an den Stapel. Ein ohne
Netz gescannter Ausweis wird als NUMMER gemerkt; die folgenden Bücher tragen ihn, der Server
löst ihn beim Nachbuchen auf. Je Form ein eigener Testfall statt fünfmal `B-10234`.

**Stand 16.09.2026, nachts: Schritt C ist gebaut.** Am Band hängt ein Knopf mit der Zahl der
offenen Meldungen, dahinter die Liste mit Zeitpunkt, Buch, Person, Ergebnis und Grund,
quittierbar mit „Erledigt". Die Zahl gehört jeder Theken-Rolle, die Liste verlangt
`view_students`; aktuell bleibt beides über die SSE-Leitung, und „Theke leeren" schliesst die
Liste mit. Doku nachgezogen: Handbuch, Fachkonzept 18.4, VVT und Datenschutzhinweis (die
kurzzeitige Speicherung von Ausweis- und Buchnummern am Theken-Rechner).

**Nächster Schritt, vor allem Weiteren:** die Nachweise am Stack (2.3) — Stufe 1 und Stufe 3
von Hand im echten Chrome, Stufe 2 über die Tür. Gebaut ist alles; geprüft ist es erst, wenn es
einmal von Hand gelaufen ist.

**Stand 15.09.2026 (überholt, siehe oben):** Stufe 1 galt als gebaut — sieben Commits `0fa4b5a3`, `18b4887e`, `44615c42`,
`08312c87`, `65f9a998`, `6fe6ba8b`, `23ca498c`, je ein Rot-Test am alten Code, volle Suite mit
Postgres, Lint und Frontend-Gates grün (Einzelheiten in den Commit-Nachrichten). Abweichungen
vom Plan: Der Warteschlangen-Eintrag behält `id` als Schlüssel (der keyPath des bestehenden
IndexedDB-Schemas, keine Migration); die Absicht heißt `ausleihe`/`rueckgabe` wie die Antworttypen
des Servers; ein Stapel-Eintrag, dessen Buchung gerade läuft, bekommt 503 (nicht 409), damit der
Sync ihn liegen lässt.
**Stand abends:** Stufe 2 (der Server) ist gebaut und gepusht, die CI ist grün.
**Stand spät:** Die Tür-Funde aus dem Rasterdurchgang sind behoben — der Bewegungsstempel läuft
nie rückwärts, die Tür liest die Antwort unter dem Schlüssel und legt ihre Ergebnisse ab, sie
rechnet den Uhrversatz des Rechners heraus, die Barcode-Liste führt auch ausgesonderte Exemplare,
und die Theke hat eine Rückrechnung der Littera-Etiketten mit denselben Prüffällen wie der Server
(Einzelheiten in den Commit-Nachrichten).
**Stand 16.09.2026, abends — der Stufe-1-Nachweis ist von Hand gelaufen** (echter Chrome, Netz
gekappt). Zwei Befunde, beide echt, beide Stufe-3-Arbeit:

1. **An der Theke lässt sich nichts mehr buchen — aus ZWEI Gründen, und der zweite ist der
   eigentliche.** Sichtbar ist der Offline-Hinweis (`components/OfflineIndicator.svelte`): ein
   `fixed`-Block über der Anwendung, mit Riesenschrift und grossen Knöpfen, und bei null Vorgängen
   gleich laut wie bei fünfzig („0 Vorgänge nur auf diesem Rechner"). Blockierend ist aber etwas
   anderes: 25 Sekunden nach dem letzten Herzschlag setzt `App.svelte` `heartbeatOk` auf falsch
   und legt ein `fixed inset-0`-Vollbild mit Weichzeichner über die ganze Seite („VERBINDUNG
   VERLOREN / Reconnecting…"). DAS ist das Vollbild aus Punkt (a) des Plans. Wer nur den roten
   Balken schlanker macht, hat die Theke nicht entsperrt.
2. **Ohne Netz kommt niemand mehr herein.** Zwei Ursachen, die man auseinanderhalten muss. Der
   Sperrbildschirm nach 15 Minuten wirft die Sitzung NICHT weg (sie hält 12 Stunden und erneuert
   sich alle 30 Minuten) — aufgemacht wird er aber mit dem Passwort gegen den Schul-Mailserver,
   und ohne Netz passt der Schlüssel nicht ins Schloss (`stores/idleLock.svelte.js`, `entsperren`
   ruft `/login`). Lädt der Tab neu, kommt die volle Anmeldemaske, und die braucht den
   Mailserver ebenfalls.

**Entschieden am 16.09.2026 — der Nachweis hat die Entscheidung vom 13.09. widerlegt.** Dort
stand: „Nach einem Neuladen ohne Netz bleibt die Anmeldemaske der Rückfall." Das taugt an der
Theke nicht; „Tab neu geladen, kein Netz" hieße Nachmittag vorbei. Stattdessen gilt: Ist das
Sitzungs-Kärtchen im Browser noch gültig, arbeitet die Theke ohne Netz weiter, ohne den
Mailserver zu fragen — eingeschränkt auf das Nötige, also scannen in die Warteschlange ja,
Schülerdaten anzeigen nein. Sobald das Netz zurück ist, wird geprüft. Das erweitert den Umfang
von Stufe 3 um diesen Punkt; die Einschränkung ist der Ersatz für die Prüfung, die gerade nicht
möglich ist.

**Nächster Schritt:** Stufe 3 bauen — Band statt Block, keine Sperre ohne Netz, die eben
entschiedene Weiterarbeit mit gültiger Sitzung, Nachsenden, Meldungsliste. Der Nachweis für
Stufe 2 (Anfragen an die Tür, 2.3) steht weiterhin aus; er gehört dem, der den Umbau gebaut hat,
denn wer nicht weiß, was dort zugesagt wurde, hakt ihn nur ab.

### 2.1 Was heute fehlt (am Code gelesen 13.09., nachgeprüft 14.09.2026)

1. Offline werden nur `B-`-Barcodes gespeichert (`omnibox.svelte.js`, `speichereOfflineAktion`).
   Alles andere, auch `LMF-`, Ziffern und Ausweise, meldet „Netzwerkfehler" und wird verworfen.
2. Ein Ausweis ohne Netz wird nicht geladen; die folgenden Bücher gehen an die vorher geladene
   Person. Mit geladener Lehrkraft wird ein Buch als Rückgabe eingereiht (nur `activeStudent`
   entscheidet; `activeTeacher` wird ignoriert, obwohl der Stapel-Endpunkt `active_teacher_id`
   kennt).
3. Person und Absicht werden erst NACH der hängenden Anfrage gelesen (Timeout 10 s in
   `apiFetch.js`). Escape oder „Theke leeren" in dieser Zeit: Das Buch geht als Rückgabe ohne
   Person in die Warteschlange. Ein Scan-Zeitpunkt wird nicht festgehalten; `offlineQueue.js`
   schreibt den Zeitpunkt des Einreihens.
4. In die Warteschlange kommt jeder `TypeError`, auch einer aus der Auswertung einer gelungenen
   200-Antwort (`verarbeiteAktionsErgebnis` liegt im selben `catch`). Der Eintrag trägt denselben
   Idempotenz-Schlüssel; solange der Server-Cache den Schlüssel hält (24 h, `jobs/cron.go`),
   kommt die alte Antwort zurück, danach wird neu gebucht.
5. Beim Nachbuchen gilt als erledigt: `success`, jeder 4xx außer 429, und „Server nannte den
   Index nicht" (`!result`). Der Antworttyp wird nicht mit der Absicht verglichen. Ein zweiter
   Scan desselben Buchs beim selben Kind wird zur Rückgabe; ein Buch, das noch bei jemand anderem
   steht, wird dort nur zurückgenommen (`handleForeignReturn`, bewusst kein Umbuchen).
6. Ein liegengebliebener Eintrag (5xx, 429) wird ohne Pause sofort erneut gesendet
   (`while (navigator.onLine)` in `offlineSync.svelte.js`). Antworten ab 500 werden serverseitig
   nicht gespeichert, 4xx schon.
7. Der Nachbuch-Bericht ist ein Toast mit dem Grund des ersten abgelehnten Eintrags. Er
   verschwindet mit dem Toast.
8. Nach 25 s ohne SSE-Signal legt sich ein Vollbild über die Theke (`App.svelte`,
   `heartbeatOk`). 502/503/504 gelten nicht als offline. Es gibt drei voneinander unabhängige
   „offline"-Begriffe: Herzschlag (Vollbild), `navigator.onLine` (rosa Band oben und
   Einreih-Entscheidung), Versandfehler je Anfrage (Einreih-Entscheidung).
9. Nach 15 Minuten ohne Eingabe sperrt die Theke per `setTimeout`, ohne Netz-Abfrage; entsperren
   geht nur über `POST /login`. „Theke leeren" nach 5 Minuten.
10. „Sicherung speichern" (`OfflineIndicator.svelte`) hängt strukturell schon außerhalb von
    Anmeldemaske und Sperrbildschirm, zeigt sich aber nur bei Zähler > 0, und der Zähler wird erst
    nach der Anmeldung geladen. Nach Neuladen ohne Anmeldung: kein Knopf.
11. `enqueueOfflineAction`, `loadQueue` und `dequeueOfflineAction` schlucken jeden IndexedDB-Fehler;
    die Theke meldet danach „gespeichert" mit Erfolgston, das Band zeigt 0.
12. Idempotenz (`api/action.go`): Antwort wird nach der Arbeit mit dem Request-Context
    gespeichert; bricht der Aufrufer ab, geht das Speichern verloren. Keine Reservierung vor der
    Arbeit. Zwei Anfragen mit demselben Schlüssel: gleichzeitig fängt der Unique-Index die zweite
    (gleicher Ausleiher → bestehende Ausleihe); kommt die zweite NACH dem Commit der ersten und vor
    dem Speichern der Antwort, wird sie zur Rückgabe.
13. Rückholen eines ausgesonderten Exemplars (`holeExemplarZurueck`) läuft in einer eigenen
    Transaktion mit Commit VOR Sperre, Limit und Vormerkung. Zwilling ohne `VerbucheRueckkehr`:
    `MarkiereVerlustAlsGefunden` (Inventur).
14. Der Online-Pfad sperrt `schueler`, dann die Ausleihe (`FOR UPDATE`), nie das Exemplar. Die
    Schranken-Zählungen laufen über `s.pool`, nicht über `tx`. Die Uhr ist je Service
    (`s.heute()`), nicht je Aufruf; die Handapparat-Frist nimmt an zwei Stellen rohes `time.Now()`.
    `ausgeliehen_am` ist ein DB-Default, `rueckgabe_am` wird mit `CURRENT_TIMESTAMP` geschrieben.
    `check_return_date` (23514) ist im Ausleihpfad nicht gemappt → 500.

### 2.2 Der Bau in drei Stufen, 17 Commits

Je Stufe: Rot-Test am alten Code, volle Suite mit Postgres, Nachweis am frisch gebauten Stack und
im Browser, dann die Freigabe.

Ratschen, die jeder Commit im Blick hat: 200 Zeilen je Frontend-Datei (`App.svelte` steht auf
genau 200, `Omnibox.svelte` mit 282 im Bestand und darf nicht wachsen); kein SQL in `api/`
(`schichtung_test.go`); `offlineImport.test.js` pinnt die Signatur von `enqueueOfflineAction`;
`omniboxOffline.test.js` ist heute nur durch den Fehler aus Commit 2 grün (der `apiClient.post`-Mock
liefert `undefined`, der TypeError kommt aus `res.ok`); `offlineSync.test.js` pinnt „4xx fliegt raus".

#### Stufe 1 — vorhandene Fehler (7 Commits, je ein Fund)

1. **Schnappschuss beim Scan.** `submitAction` hält vor dem Versand fest: Absicht (Ausleihe,
   Rückgabe), Person (Schüler ODER Lehrkraft), Scan-Zeitpunkt. Der Warteschlangen-Eintrag wird ein
   Objekt (`{art, barcode, schueler_id, lehrer_id, gescannt_am, key}`), `enqueueOfflineAction`
   nimmt es entgegen. `offlineImport.test.js` wird einmal auf das Objekt umgestellt, nicht dreimal.
   Format 1 (alte Sicherungen: `{action_type, barcode_id, schueler_id, timestamp}`) bleibt lesbar.
   Rot-Test: Anfrage hängt, Store-Person wird geleert, Timeout → Eintrag trägt die Person vom Scan.
1. **Nur ein gescheiterter Versand wird eingereiht.** Der `catch` um `verarbeiteAktionsErgebnis`
   reiht nicht ein. Der Mock in `omniboxOffline.test.js` wird auf einen echten Versandfehler
   umgestellt (`apiClient.post.mockRejectedValue(new TypeError('Failed to fetch'))`); das ist der
   Beweis, dass der Test vorher die falsche Sache maß. Rot-Test: 200 mit `{type:'teacher'}` ohne
   `teacher` → Fehlerbanner, keine Warteschlange.
2. **„Buch zurückgeben" im Profil ist offline eine Rückgabe.** `onReturnClick` übergibt die
   Absicht aus Commit 1. Rot-Test: Doppelklick offline → zwei Rückgaben, keine Ausleihe.
3. **Lehrkraft geladen: Offline-Buch ist eine Handapparat-Ausleihe.** Der Eintrag trägt
   `lehrer_id`, `baueBatchPayload` sendet `active_teacher_id` (der Stapel-Endpunkt kennt es
   heute schon). Rot-Test am Payload.
4. **Ein Fehler der Warteschlange heißt nicht „gespeichert".** `enqueueOfflineAction` wirft weiter;
   die Theke meldet „NICHT gespeichert — Buch zurücklegen" mit Fehlerton. `loadQueue` liefert bei
   Fehler nicht `[]`, sondern wirft; das Band zeigt „Warteschlange nicht lesbar" statt 0.
   Rot-Test: IndexedDB wirft.
5. **Erledigt ist nur, was der Server wie gescannt gebucht hat.** Die `!result`-Failsafe fällt
   (Schweigen des Servers ist kein Erfolg). Passt der Antworttyp nicht zur Absicht (Ausleihe
   gescannt, `rueckgabe` gebucht), wird der Eintrag ausgebucht UND gemeldet, mit Barcode und
   beiden Typen; er blockiert nicht. 4xx wird weiter ausgebucht und gemeldet. 5xx und 429 bleiben
   liegen, aber die Runde endet dort, statt sofort erneut zu senden; der nächste Anlauf kommt mit
   dem nächsten `online`-Ereignis oder nach einer Minute. (Das Blockieren der Warteschlange am
   ersten `wiederholen`-Eintrag kommt erst in Stufe 3 mit der neuen Tür, die den Fall
   `bereits_ausgeliehen` kennt; vorher hätte ein legitimer Fall die Warteschlange gesperrt, ohne
   dass es einen Ort gäbe, ihn aufzulösen.)
6. **Idempotenz hält.** Die Antwort wird mit `context.WithoutCancel` gespeichert (Vorbild
   `mahnwesen_bulk_mail.go`); der Schlüssel wird vor der Arbeit reserviert (Zeile mit Marker,
   `response_data` bekommt eine unterscheidbare Form oder wird nullbar mit `status_code = 0`);
   ein zweiter Aufruf auf einen reservierten Schlüssel wartet oder antwortet 409 `in_arbeit`.
   Antworten ab 500 werden weiter nicht gespeichert. Rot-Test (Postgres): zweite Anfrage mit
   demselben Schlüssel NACH dem Commit der ersten und VOR dem Speichern der Antwort → heute
   Rückgabe, danach dieselbe Ausleihe. Der 24-Stunden-Ablauf bleibt; die Nachbuch-Tür (Stufe 2)
   deckt spätere Wiederholungen über den Wächter ab.

#### Stufe 2 — Server: gebaut am 15.09.2026

Fünf Commits, alle Gates grün, jeder mit Rot-Probe am alten Code. Was jetzt steht:

- **Rückholen als Baustein** — Umlauf und Forderung in der Transaktion des Aufrufers; der
  Fund im Fehlbestandsbericht nutzt denselben Baustein (vorher stand das UPDATE zweimal da).
- **Migration 116, Bewegungsstempel** — `ausleihen.erfasst_am` (Scan-Zeitpunkt) und
  `buecher_exemplare.letzte_bewegung_am`, von den Schreibern gesetzt. Sperrreihenfolge
  Schüler → Ausleihe → Exemplar steht in [invarianten.md](invarianten.md).
- **Migration 117, Nachbuch-Meldungen** — jede Abweichung vom Scan bleibt mit Barcode,
  Grund und Beteiligten stehen, bis jemand sie quittiert. Liste und Quittieren mit
  `view_students`, Zähler fürs Band mit `perform_actions`. Quittierte fallen nach der
  Lesehistorie-Frist, höchstens 30 Tage; offene meldet die Betriebsbereitschaft nach
  14 Tagen. Personenspalten wandern beim Zusammenführen, werden getilgt und stehen in der
  Auskunft.
- **`POST /api/action/nachbuchen`** — 1–50 Einträge und die Sendezeit des Rechners
  (`gesendet_am`, Pflicht), je Eintrag eine Transaktion. Scan-Zeitpunkte werden um den
  gemessenen Uhrversatz umgerechnet (Antwort `uhr_versatz_sekunden`), höchstens Serverzeit.
  Wächter gegen veraltete Scans. Unter dem Idempotenz-Schlüssel liest die Tür die gespeicherte
  Antwort: online schon gebucht → `bereits_gebucht`; online nur die Fremdrückgabe → Ausleihe
  ab der Rückgabe nachholen; eigene Ergebnisse werden abgelegt und einer Wiederholung
  zurückgegeben. Rücknahme beim Vorbesitzer vor dem Savepoint, Schranken über die
  Transaktion des Eintrags. Ein Serverfehler beendet den Aufruf nicht — der Eintrag
  bekommt „wiederholen" und bleibt auf dem Rechner.
- **`GET /api/action/buchbarcodes`** — die Barcodes aller Exemplare, auch ausgesonderter,
  bewusst ohne LIMIT (begründet im Kopfkommentar), mit Stand-Merker aus Anzahl und
  jüngster Änderung; unverändert antwortet der Server 304. Gepackt, weil Caddy nicht
  komprimiert. Keine Personendaten.

**Offen aus Stufe 2:** der Nachweis am Stack (2.3, Server-Hälfte) — die curl-Fälle gegen
`/nachbuchen`, `docker stop` der Datenbank während eines Laufs und `pg_stat_activity` beim
Nachbuchen von 200 Einträgen. Die PG-Tests decken die Fälle fachlich ab; der Nachweis
zeigt sie am laufenden Stack.


#### Stufe 3 — Theke (5 Commits)

13. **Ein Band statt Vollbild und rosa Leiste; ohne Netz keine Sperre.** Das Vollbild fällt
    (macht in `App.svelte` Platz), `OfflineIndicator` wird das eine Band: Zustand, Zähler,
    „Sicherung speichern", Zugang zur Meldungsliste. Ein Prädikat `verbindungFehlt` in
    `offlineSync` vereint die drei Begriffe: `navigator.onLine === false`, Herzschlag älter als
    25 s, letzter Versand mit Netzfehler, Timeout oder 502/503/504 (abzweigen vor
    `handleActionHttpError`). `idleLock` sperrt nicht, solange `verbindungFehlt`; kommt die
    Verbindung zurück und war länger als 15 Minuten niemand da, sperrt es sofort. Geleert wird
    weiter nach 5 Minuten. Der Zähler kommt direkt aus IndexedDB, auch ohne Anmeldung. Der
    Sperrbildschirm (`aria-modal`, Fokusfalle) lässt den Knopf per Tastatur erreichbar. axe-Gate
    misst den Zustand „Band sichtbar" (`context.setOffline(true)` wie in
    `abmelden-ohne-antwort.spec.js`). `idleLock.test.js` bekommt den Fall „ohne Netz keine Sperre".
14. **Sync über die Nachbuch-Tür.** Portion 25, Frist 20 s, Einträge nach Scan-Zeitpunkt; die
    Runde endet beim ersten `wiederholen`, alles andere wird ausgebucht und gemeldet;
    `bereits_gebucht` wie ein Erfolg (Typvergleich wie Stufe 1, Commit 6). Jede Portion trägt
    `gesendet_am` von derselben Uhr wie `gescannt_am`. Springt die Uhr des Rechners zwischen Scan
    und Versand (Zeitabgleich nach der Netzrückkehr), verfälscht der Sprung die Umrechnung —
    beim Bau eine Uhr wählen, die nicht springt, oder den Sprung mitschicken. Sicherungen
    (Format 2 mit Zeitstempel und Absicht; Format 1 weiter lesbar) werden gemeinsam eingespielt,
    ein `startSync()` für alle Dateien. `/api/action/batch` bleibt eine Version.
15. **Offline alle Buchformen und Ausweise.** Die Buchliste wird nach der Anmeldung geholt
    (`starteHintergrundAbrufe`, `perform_actions`, ETag) und lokal gehalten, ohne Personenbezug.
    `B-` und `LMF-` gelten auch ohne Liste; ein 13-stelliges Littera-Etikett wird zuerst
    zurückgerechnet (`frontend/src/lib/litteraEtikett.js`, gebaut, Prüffälle gemeinsam mit dem
    Server); Ziffernfolge auf der Liste = Buch (die Liste führt auch ausgesonderte); `S-`/`L-` = Ausweis
    (Merker; Person geleert; folgende Bücher tragen den Ausweis-Barcode); unklar = Zuordnung
    gesperrt bis zum nächsten eindeutigen Ausweis. Geräte offline mit Meldung abgewiesen. Nach
    Rückkehr der Verbindung: beim nächsten Buchscan den Merker-Ausweis online auflösen, dann normal
    buchen. Dieser Commit kommt NACH 14: Vorher hätte der alte Stapel-Sync einen Ausweis-Eintrag
    als Ausweis-Lookup gebucht und die folgenden Bücher als Rückgabe.
16. **Meldungsliste** hinter dem Band, nur angemeldet, nicht gesperrt, mit `view_students`;
    „Theke leeren" schließt sie (Eintrag in `thekeLeeren.js`). Quittieren; Zähler an allen
    Arbeitsplätzen über SSE. Neues Bauteil, eigene Datei.
17. **Doku:** HANDBUCH („Theke ohne Verbindung"), FACHKONZEPT 18.4, PII-Matrix, invarianten
    (Sperrreihenfolge), VVT und Datenschutzhinweis (Meldungen mit Frist; kurzzeitige Speicherung
    von Ausweis- und Buchnummern am Theken-Rechner), Rückweg-Anleitung und OFFEN.md.

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
Rückbau von `/api/action/batch` (nächste Version) · scannende Person im Protokoll (nachgebucht wird
unter dem beim Sync angemeldeten Konto; steht in der Doku).

### 2.5 Was die Prüfung am 14.09.2026 am ersten Entwurf geändert hat

- Commit 6 blockiert die Warteschlange nicht mehr in Stufe 1: Ein legitimer Fall (Ausleihe
  gescannt, Server bucht Rückgabe) hätte jeden Rechner gesperrt, ohne Ort zur Auflösung bis
  Stufe 3. Er meldet; blockiert wird erst mit der neuen Tür.
- Der Rot-Test zu Commit 7 muss die Form „zweite Anfrage nach dem Commit der ersten" haben; die
  gleichzeitige Form fängt heute schon der Unique-Index.
- Stufe 2 in der Reihenfolge 8 → 9 (Migration) → 10 (Meldungen) → 11 (Tür) → 12: Die Tür braucht
  den Bewegungsstempel und die Meldungstabelle; im ersten Entwurf kamen beide nach ihr.
- „Rücknahme bleibt, neue Ausleihe scheitert" in EINER Transaktion braucht einen Savepoint; der
  erste Entwurf hätte beim Rollback die Rücknahme mitgenommen.
- Der Wächter braucht die Ausnahme „gleicher Idempotenz-Schlüssel", sonst weist er genau den
  Eintrag ab, den die Ableitung oben („abgebrochener Online-Versand") nachholen will.
- Sperrreihenfolge festgelegt (Schüler, Ausleihe, Exemplar): Online sperrt das Exemplar heute
  nie; „Exemplar sperren → Wächter" als erster Schritt hätte gegen den Online-Scan einen Zyklus
  gebaut.
- Uhr je Aufruf statt je Service, Repository-Funktionen mit Datum, beide Handapparat-Fristen,
  23514-Mapping, Schranken über `tx`: fehlten im ersten Entwurf.
- Stufe 3: 15 (neue Formen) nach 14 (neue Tür); ein Band statt zwei; das Netz-Signal für die
  Sperre ist benannt; Format-1-Sicherungen bleiben lesbar; `App.svelte` und `Omnibox.svelte`
  stehen an der 200-Zeilen-Ratsche.
- Der Nachbuch-Bericht war nie ein Bericht, sondern ein Toast (2.1, Punkt 7).
- Antworten ab 500 werden heute schon nicht gespeichert; die befürchtete Endlosschleife am
  gecachten 5xx gibt es nicht, nur das pausenlose Wiederholen (2.1, Punkt 6).

---

## 3. Theke — vor dem Offline-Bau (Kategorie B)

Nichts offen (Stand 15.09.2026); die Nummer bleibt, weil Abschnitt 4 auf 3.4 verweist.

---

## 4. Entscheidungen

Die Nummern bleiben fest. Beantwortete Fragen wandern in den Punkt, der sie umsetzt (4.1 → Abschnitt
2, 4.2 → 3.4) oder fallen weg, sobald sie umgesetzt sind.

### 4.3 `ziel_jahrgang`: bauen oder streichen

> **Gestoppt am 17.09.2026 (siehe 9.6).** Das Medienzentrum verlangt ausdrücklich
> Mehrjahresbände. Bis die Frage neu entschieden ist, wird `ziel_jahrgang` nicht angefasst und
> die Messung unten nicht gefahren.

`ziel_jahrgang` (mehrjährige Ausleihe) wird in `internal/service/loan_rules.go` gelesen, aber von
keinem Code geschrieben; die Fristregel verzweigt auf einen Wert, der immer 0 ist. Die Frist am
Rückgabetermin ist entschieden (1.4). **Entschieden am 16.09.2026: streichen.** Spalte, die drei
Lesestellen in `repository/book_search.go` und der Zweig in der Fristregel fallen. Der Zweig ist
zugleich das Tor zur LMF-Plan-Frist (sie gilt nur bei `additionalYears == 0`) — ein Wert, den
niemand setzt, darf diese Regel nicht aushebeln können.

**Vor dem Streichen eine Zählung am Server (16.09.2026).** „Wird von keinem Code geschrieben"
gilt für den HEUTIGEN Code; im Juni gab es einen Schreiber (`INSERT INTO buecher_titel (…,
ziel_jahrgang, …)`, Commit `f8dab25a`, mit den Migrationen 029/030). Steht auf dem Server auch
nur ein Titel mit einem Wert, dann ist die mehrjährige Ausleihe dort nicht tot, sondern in
Betrieb — und das Streichen gäbe diesen Büchern beim nächsten Ausleihen eine Frist im selben
Schuljahr statt in einem späteren. Deshalb erst messen, dann bauen:

```sql
SELECT count(*) AS titel_mit_wert, min(ziel_jahrgang), max(ziel_jahrgang)
FROM buecher_titel WHERE ziel_jahrgang <> 0;
```

Ergebnis 0 → streichen wie entschieden. Ergebnis > 0 → die Frage ist eine andere und kommt
zurück auf den Tisch.

### 4.4 E6: Nach der Übergabe an die Schulaufsicht

Bleibt der Schüler gesperrt und die Forderung offen, bis das Sekretariat „bezahlt laut
Finanzbericht" bucht — oder gilt die Übergabe schulseitig als erledigt? **Vorschlag (Konzept):**
Sperre bleibt, Löschblockade fällt. **Wann:** sobald ein erster echter Bescheid absehbar ist;
blockiert 5.3. Einzelheiten in [mittel_konzept.md](mittel_konzept.md), Abschnitt 6.

### 4.5 E4: Feld „Listenpreis" am Titel

**Keine Komfortfrage mehr (17.09.2026, siehe 9.3 b).** Die Arbeitshilfe verlangt ab dem zweiten
Verleihjahr den Neupreis zum Zeitpunkt des Verlusts, die Anforderungsliste Nr. 3 beide Preise
wählbar. Das Feld gehört damit zum Abnahme-Gate des Medienzentrums.

Für die Staffel ab dem 2. Verleihjahr. Der Bescheid-Handler übergibt heute als Neupreis 0
(`api/bescheid_handler.go`); die Staffel nimmt dann ersatzweise den Kaufpreis, der Dialog zeigt
„Kaufpreis (kein Neupreis hinterlegt)". Ein Mensch bestätigt den Betrag. **Vorschlag (Konzept):**
optionales Feld; der Vorschlag nimmt den Listenpreis, sonst den Einkaufspreis, und sagt, welchen.
**Wann:** vor dem ersten echten Bescheid — sonst beantwortet der erste Bescheid für ein Buch ab
dem 2. Verleihjahr die Frage still mit „Kaufpreis". Der Staffel-Vorschlag im Schadensdialog (5.4)
übernimmt die Antwort später.

### 4.7 Sechs Umgebungsvariablen, die Compose nicht durchreicht

`ALLOWED_ORIGIN`, `RATE_LIMIT`, `SENTRY_DSN`, `IMAP_PORT`, `SMTP_ALLOW_INSECURE_TLS` und
`SMTP_ALLOW_PLAINTEXT` liest der Server; `docker-compose.yml` reicht keine davon durch, im
Container gilt also immer die eingebaute Vorgabe. Die beiden `SMTP_ALLOW_*` führte das Register
als Werkzeug-Variablen — der Server liest sie in `mailservice/versand.go`. Auf dem Server am
13.09.2026 im Container leer.
**Frage:** Je Variable: Ist die Vorgabe gewollt? Sonst durchreichen. `SENTRY_DSN` bleibt leer
(Datenschutz A6). **Danach:** Ratsche in 5.10.

### 4.8 Etiketten-Altbestand nachtragen?

30.676 echte Exemplare zählen auf dem Server als „Etikett offen" (13.09.2026). Es gibt zwei Wege
mit verschiedener Bedingung: das Druck-Center („Fehlende Etiketten" → „Altbestand aufräumen",
`POST /api/exemplare/etiketten-altbestand`, mit Vorschau und Stichtag; Abnahme-Flow 4) und
`scripts/repair_altbestand_etiketten.sql` (alles ohne `B-`, also auch jedes `LMF-`-Exemplar). Ob
das Skript schon einmal lief, ist nicht belegt. Beides ist nicht umkehrbar.
**Frage:** Nachtragen, und über welchen Weg? Tragen die LMF-Exemplare ein Etikett? **Wann:** vor
Abnahme-Flow 4; kommt eine neue Littera-Übernahme (7.2), erst danach.

### 4.9 Security-Jobs im Release-Gate

Die Pflichtliste in `.github/workflows/release.yml` enthält keinen der Security-Jobs
(govulncheck, gosec, npm audit, Trivy); ein Tag auf einen Commit mit rotem Trivy erzeugt trotzdem
das Release; für das Image siehe 5.10. **Frage:** aufnehmen oder begründet so lassen? Bei Ja prüft
`docs/umgebung_paritaet_test.go` auch `security-scan.yml`. **Wann:** vor dem nächsten Release.

### 4.10 Offene PRs (Stand 17.09.2026)

**Fünf Jules-PRs, aber nur zwei Themen.** #621, #624 und #626 schlagen dreimal dieselbe Sache vor
(CTE für die Klassensatz-Verfügbarkeit; laut Messung vom 13.09.2026 langsamer als die bestehende
Abfrage, nicht wiederholt), #620 und #625 zweimal dieselbe (`title` an gesperrten Knöpfen des
Planers). Dazu die Remote-Branch des geschlossenen PR #611
(`fix/cron-dsgvo-test-comment-5377838966258287578`). **Vorschlag:** alle schließen, Branches
löschen.

**Zwei Dependabot-PRs, seit dem 14.09.2026 offen** — sie standen bis heute in keiner Liste:
#622 (Go, sechs Pakete; `build-and-test` ist ROT) und #623 (npm, sieben Pakete, Prüfungen grün).
Auto-Merge ist aus, also entscheidet sie jemand von Hand. **Vorschlag:** #623 übernehmen; bei #622
zuerst die rote Prüfung ansehen — ein blankes Update hat den Build schon einmal gebrochen, deshalb
gilt „gezielt statt in einem Rutsch".

### 4.11 Topf auf der Bestätigungsseite und den großen Etiketten

Der Händler bekommt am selben Tag zwei gleich aussehende Links; auch zu einer
Schülerbücherei-Bestellung werden die großen Lernmittel-Etiketten „Eigentum des Landes"
angeboten. **Frage:** Nennt die Bestätigungsseite den Topf? Große Etiketten nur bei `land`?

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
Schreibweg vorbeikommt. Das Prädikat rechnet dann `GREATEST(AbgangSeit, letzter_vorgang_am)`
ohne Unterabfragen. Nachweis: ein PG-Test mit Karenz > Lesehistorie, der beweist, dass der
Lesehistorie-Lauf den Anonymisierungs-Zeitpunkt NICHT verschiebt — am alten Stand rot.

### 4.13 Abgänger-Druck und die Suche

Zwei Kommentare sichern zu „Was auf dem Bildschirm steht, steht auf dem Papier";
`GET /api/abgaenger/pdf` kennt aber nur `klasse`, die Suche filtert im Browser. **Frage:** Folgt
der Druck der Suche (neuer Parameter, dieselbe Auswahl zweimal), oder bleibt er klassenweise
(dann fallen die zwei Kommentare)? Ohne Antwort wird nichts gebaut.

### 4.14 Cover im Kollegiums-Portal (`AnliegenWidget`)

Tabelle und API tragen `isbn` und `titel_id` am Anliegen schon; nur das Wunsch-Formular schickt
sie nicht mit. Am 05.09.2026 entschieden: Arbeitslisten der Bibliothek bleiben ohne Cover.
**Frage:** Gilt das auch für die eigene Liste der Lehrkraft, oder bekommt das Wunsch-Formular
eine Buchauswahl?

### 4.16 Routen ohne Aufrufer

Laut API-Inventar (`docs/api_inventar.md`) ruft weder das Frontend noch ein Skript im Repo diese
Routen auf: `PUT /api/books/{id}/cover`, `POST /api/books/{id}/refresh-cover`,
`POST /api/buecher/exemplare/{id}/schadensnotiz` und `POST /api/buecher/exemplare/{id}/aussondern`.
Ein Grep schließt Aufrufer außerhalb des Repos nicht aus.

**Die vier Befunde (17.09.2026, am Code gelesen) — zu entscheiden ist je Route:**

1. **`PUT /api/books/{id}/cover`** (`inventur/update_cover_handler.go`) setzt eine Cover-Adresse
   von Hand, mit Herkunftsprüfung. Sie ist die Tür, die 5.12 schon nennt: Ein unbekanntes Buch
   ergibt 500 statt 404, und ein altes hochgeladenes Cover bleibt als Datei liegen. In der
   Oberfläche gibt es den Upload (`/cover-upload`) und den Sammellauf (`/admin/sync-covers`) —
   eine Adresse von Hand einzutragen, kann heute niemand. *Vorschlag: streichen.* Wer ein
   bestimmtes Bild will, lädt es hoch; das ist derselbe Zweck ohne Fremd-URL in der Datenbank.
2. **`POST /api/books/{id}/refresh-cover`** (`inventur/cover_aktualisierung.go`) holt das Cover
   EINES Buches neu bei den Katalogdiensten. Sauber gebaut (404 bei unbekanntem Buch,
   unterscheidet Netzausfall von Nicht-Treffer, eigene Tests). In der Oberfläche gibt es nur den
   Sammellauf über den ganzen Bestand. *Vorschlag: anbieten* — ein Knopf „Cover neu holen" in der
   Titel-Akte ist die kleinere Handlung als „alle Cover abgleichen", und die Tür ist fertig.
3. **`POST /api/buecher/exemplare/{id}/schadensnotiz`** (`api/copy_admin_status.go`) schreibt die
   Zustandsnotiz eines Exemplars. Kein Aufrufer; die Notiz entsteht heute nebenbei beim Melden
   eines Schadens und beim Aussondern. *Frage: Soll die Bibliothek eine Notiz am Exemplar von
   Hand ändern können — oder ist die Notiz bewusst nur eine Spur der Vorgänge?*
4. **`POST /api/buecher/exemplare/{id}/aussondern`** (`api/copy_admin_status.go`) sondert ein
   Exemplar aus, mit der richtigen Sperre („noch verliehen" → 409). Ausgesondert wird heute über
   den Status-Editor und über die Schadensmeldung. *Frage: doppelte Tür zum selben Zustand
   (Bugklasse „zwei Türen") — oder der bewusste kurze Weg?* Wenn doppelt: streichen, sonst in der
   Exemplar-Liste anbieten.

`POST /api/buecher/exemplare/{id}/defekt` ist am 16.09.2026 gestrichen — sie war zur Hälfte kaputt
(Zweig ohne Schüler in eine Spalte, die Migration 125 entfernt hat), nicht bloß ungenutzt.

### 4.17 Echte Schülerdaten auf dem Hetzner-Server?

[datenschutz_offene_punkte.md](datenschutz_offene_punkte.md) nimmt an, der Hetzner-Server trage
nie echte Schülerdaten; er ist aber die einzige laufende Instanz. Ob echte LUSD-Schülerdaten dort
liegen, ist nicht gemessen. **Nächster Schritt:** Messung (7.8), dann Doku oder Datenlage
angleichen. Bezug: 8.5 (B5, B6).

---

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

**Reihenfolge, wenn freigegeben:**

1. **„Auflage" als echtes Feld am Titel.** Heute gibt es das Wort im ganzen System nur als
   Freitext in `erweiterte_eigenschaften` aus dem Listenimport. Ohne das Feld stehen zwei Zeilen
   „Lambacher Schweizer 7" in jeder Liste, die niemand unterscheiden kann — unabhängig davon, was
   danach kommt. Kleinster Schritt, nützt sofort.
2. **Dublettenkontrolle beim Anlegen** wie bei Littera (ISBN, sonst Autor und Titel). Sie löst den
   anderen Fall — dasselbe Buch versehentlich zweimal —, der als Fund schon in 5.5 und 5.12 steht.
3. **Das Werk** samt Migration, Gruppierung im Bedarf und Warnung in der Ausgabe.

**Nicht gebaut.** Schritt 3 ändert das Schema und rechnet die Nachbestell-Liste anders. Das geht
in Stufen mit Nachweis und erst nach deiner Freigabe.

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
- **`DamageReportModal`:** Staffel-Vorschlag mit Herleitung statt Startwert 15 €, kein
  automatisches PDF-Fenster. **Vorgezogen (17.09.2026, siehe 9.3 a):** Dieser Punkt hängt an
  keiner der offenen Fragen — nur die Kreis-Rechnung hängt an 8.3.
- **Doku:** FACHKONZEPT Abschnitt 3 (Mahnwesen ohne Bescheid) und 14 (PDF-Rechnung, Barzahlung am
  Tresen); SECURITY und VVT-Entwurf mit dem Zweck „Schadensersatz-Bescheid". Den VVT-Satz
  vorziehen, bevor die Schule den Entwurf beschließt (8.5).
- **Release** beim Abschluss.

### 5.5 Bestand, Katalog, Druck

- `DeleteBooks` liest die Spuren vor der Transaktion.
- Buchetiketten haben keinen Ersatz für Zeichen außerhalb cp1252; ş und ł werden zum Punkt
  (`api/label_pdf.go`).
- Die Barcode-Höhe des Ausweises ist im Druck fest (`CardFace.svelte`).
- Die ISBN ist nur je Schreibweise eindeutig (mit oder ohne Bindestrich); ein CHECK auf den
  Jahrgang fehlt, „Jahrgang unbekannt" ist von der Vorgabe nicht zu unterscheiden. Erst Dubletten
  und Jahrgänge am Server messen (Einzeiler dafür), dann Schema.

### 5.6 Schüler und LUSD

- LUSD: Zwei Zeilen mit gleichem Namen und Geburtsdatum, aber verschiedenen Klassen werden still
  zu einer Person; die Meldung „mehrdeutig" fehlt (dokumentierte Grenze). Vor der LUSD-Abnahme als
  Hinweis in der Vorschau.
- **Ein Schüler wird Lehrkraft (oder umgekehrt) — weiter nicht möglich, und das ist Absicht.**
  Seit Migration 123 stehen alle Leser in einer Tabelle, die Art lässt sich aber nur zwischen
  Lehrkraft und LiV umstellen. Über die Schüler-Grenze verbietet es die Datenbank
  (`chk_leser_nur_schueler_werden_abgaenger`), weil ein Schüler aus der LUSD kommt und dort
  wieder auftauchen würde. Der Fall ist selten (ein ehemaliger Schüler kommt als LiV zurück);
  heute legt man dafür einen zweiten Leser an. Wenn er öfter vorkommt, ist es ein eigener,
  kleiner Umzugspfad wie Migration 072 — kein Auswahlfeld.
  Der Littera-Lauf übergeht weiter Praktikanten, Sekretariat und „Im Ausland"
  (`internal/littera/leser.go`).

### 5.7 Bestellwesen

- Der Wareneingang gruppiert nach Datum und dem aus `zustand_notiz` abgeleiteten Lieferanten statt
  nach `bestellung_id`: zwei Töpfe am selben Tag ergeben eine Gruppe, ohne Vorab-Barcode
  „Unbekannter Lieferant", das Datum ohne Schulzeitzone.
- Mail-Datum und Link-Frist stehen in Serverzeit.

### 5.8 LMF und Statistik

- Die Statistik hat keine Sequenznummer (eine langsame Antwort kann eine schnellere überholen)
  und im Browser keinen Fehlerzustand: Ein Query-Fehler ergibt eine leere Liste — protokolliert
  wird er inzwischen (`api/stats.go`), zu sehen ist er nicht.

### 5.9 Oberfläche

- Scannen fehlt im Medienkatalog und bei den Bestellungen: ein Suchfeld gibt es dort, aber keine
  Möglichkeit, einen Barcode mit dem Handgerät oder der Kamera einzulesen. Die Kamera-Erkennung
  hängt heute allein an der Theke (`Omnibox.svelte` mit `CameraScanner.svelte`); `MediaCatalog.svelte`
  und der Bestellbereich haben nur ein getipptes Feld. Peter am 17.09.2026 angemerkt — noch nicht
  entschieden, ob beide Wege (Handgerät und Kamera) an beide Stellen gehören.
- Der Stift der Katalog-Kachel: `BuchKarte.svelte` sagt „öffnet die Akte",
  `e2e/cover-aendern.spec.js` sagt „öffnet die Titel-Verwaltung". Im Browser messen, einen
  Kommentar berichtigen.
- Schülerakte: Scheitert der Abruf des Kopfes (`GET /api/schueler/{id}` in 503 oder Netzfehler),
  bleibt die Akte leer — `StudentProfile.svelte` hat nach `{:else if st.profile}` kein `{:else}`.
  Die drei Listen daneben vermerken ihren Ausfall seit dem 15.09.2026 (1.5); der Kopf ist der
  verbliebene Eintrag in `fehlerausgang.test.js`. Ein `{:else}` mit `LadeFehler` braucht Platz:
  die Datei steht an der Größen-Ratsche (238 Zeilen). Gefunden beim Bau von 1.5.

### 5.10 Gates und Werkzeuge

- Keine Ratsche „Go liest, Compose reicht nicht durch" (nach 4.7); `docs/compose_variablen_test.go`
  prüft nur die Gegenrichtung.
- Die Schema-Gegenrichtung ist blind für UNIQUE, Teilindizes und RESTRICT; die Schema-Parität
  vergleicht Funktionen nur am Namen.
- Mehrere Ratschen haben keine Zeile in der Landkarte von [sweeps.md](sweeps.md), heute noch
  `frontend-hygiene-dialoge/-ladekreis/-schalter.test.js`; Regel 7 hat keine Ratsche.
  (`docs/werkzeuge_im_image_test.go`, `docs/compose_variablen_test.go` und
  `frontend-hygiene-tabellen.test.js` stehen inzwischen drin — am 17.09.2026 nachgezählt.)
  (Die seit dem 17.09.2026 neu gebauten Ratschen tragen sich beim Bauen selbst ein — der
  Rückstand betrifft die älteren.)
- Kein Gate gegen unbegrenzte Listen-Endpunkte.
- `routes_authz_coverage_test.go` liest die `mux.Handle`-ZEILE. Eine Registrierung, die
  zwischen Adresse und Wrapper umbricht, meldet es als „hat KEINEN Autorisierungs-Wrapper",
  obwohl sie geschützt ist (am 17.09.2026 beim Einhängen des Ersatzwert-Vorschlags gesehen).
  Nur ein Fehlalarm, kein Loch: Ein Umbruch kann nie ein falsches GRÜN erzeugen, immer nur ein
  falsches Rot. Trotzdem kostet er beim nächsten Mal wieder eine Viertelstunde Suche, und die
  Meldung schickt einen in die falsche Richtung („schütze die Route" — sie ist geschützt).
  Reparatur: den Aufruf als Ausdruck über Zeilengrenzen lesen, wie es der Fehler-Kollaps-Detektor
  über den AST schon tut.
- Kein Rückweg für ältere Sicherungen beim Wechsel des `BACKUP_ENCRYPTION_KEY`;
  `cmd/rotate-encryption-key`, `cmd/littera-import` und `cmd/seed` haben keine Tests. Vor einem
  Schlüsselwechsel oder der Littera-Übernahme.

### 5.11 Doku

Nichts offen (Stand 16.09.2026). Die Nummer bleibt, weil die Reihenfolge oben auf sie verweist.

### 5.13 Mahnverfahren: Stufe 3 (nach 4.4)

Das Modell seit dem 15.09.2026 (Stufen 1 und 2): Die Reiter „Alle ·
Akut fällig · Eskaliert" fragen „Wer hat Bücher zu spät?", der Reiter „Schadensersatz" fragt „Wer
schuldet Geld?". Solange das Buch als ausgeliehen gilt, steht das Kind links; sobald ein Verlust
oder Schaden gebucht ist, rechts — mit genau einem Stand und einem nächsten Schritt je Zeile. Der
Bescheid entsteht direkt aus den überfälligen Büchern; der Brief bucht ihren Verlust.

- **Stufe 3**: Folgen der Übergabe (5.3): Übergabe-PDF und Sammelliste für das Schulamt; E6 (4.4).
- Offen aus Stufe 2: Staffelbetrag als Vorschlag im `DamageReportModal` (5.4) — der Weg über die
  Akte fragt den Betrag weiter ohne Vorschlag.

### 5.12 Review der Commits vom 11.–14.09.2026 (14.09.2026)

Rund 85 Commits, von vier getrennten Prüfungen gelesen, die A-Funde selbst am Code nachgeprüft.
Ein A-Fund (1.5), sonst B und C; die Fixes selbst waren richtig, die Funde sind Nachbarn.

- **Wächter „Ehemalige mit offenen Vorgängen" (`fa4a2113`):** Nur der Grundausdruck `AbgangSeit`
  ist mit der Löschuhr vereint; die Löschuhr rechnet zusätzlich `GREATEST(…, max(rueckgabe_am),
  max(Schadensfall))`. Ohne Außenwirkung, aber eine dritte Formulierung derselben Frage.
- **Zweiter Cover-Schreibpfad (`ddee5802` nicht mitgezogen):** `update_cover_handler.go` setzt
  `cover_url`, ohne das alte Upload-Cover zu löschen, und meldet ein unbekanntes Buch als 500.
  Gleiche Reihenfolgefrage in `cover_aktualisierung.go`, `endpunkte_cover_retry.go`,
  `cover_service.go`.
- **ISBN-Dublette (`260b1436`):** Der UNIQUE-Constraint fängt nur zeichengleiche Dubletten;
  geprüft wird auf einer bereinigten Kopie, gespeichert der Rohwert. `9783123456789` und
  `978-3-12-345678-9` sind zwei Titel. Ob das Frontend vor dem Senden bereinigt, ist ungeprüft.
- **Selbstanmeldung (`d9d84fd3`):** Ein liegengelassener Antrag liest dauerhaft „Zugang
  beantragt"; nur `aktiv = true` räumt `zugang_beantragt_am`. Kleiner geworden, seit ein
  abgelehnter Antrag samt Leserzeile gelöscht wird (16.09.2026): Es geht nur noch um Anträge,
  die weder freigeschaltet noch gelöscht werden. Ob das gewollt ist, steht nirgends.
- **Ratschen mit Umgehungsweg, heute ohne offene Stelle:** UUID-Ratsche sieht `[]struct` ohne
  `dive`, `x := T{}`, Typen fremder Pakete und `q.Get(…)` nicht; Fehlerausgang-Scanner prüft Form 2
  (`?:`) nur mit `istOkZugriff`, nicht mit der UND-Kette; Schema-Gegenrichtung führt
  `lmf_termine.art -> lmf_plaene` doppelt mit falscher Begründung (Mengenvergleich macht es
  unsichtbar); `uuidPfadParameter` hat keinen Test.

### 5.14 Rasterdurchgang 11.–15.09.2026 (15.09.2026) — nur noch die fallengelassenen Verdachte

Die drei Funde dieses Durchgangs sind erledigt (zwei fielen beim Nachprüfen am 16.09. von
selbst weg, die zwei Uhren in `UeberfaelligeAusleihen` sind am 17.09. behoben). Was bleibt,
ist die Liste der geprüften Verdachte — sie steht hier, damit der nächste Durchgang sie nicht
noch einmal findet.

Nachgestellt und **fallengelassen** — damit der nächste Durchgang sie nicht noch einmal findet:
`EmpfaengerFuerBescheid` ohne `deleted_at IS NULL` (Bescheid an ein Kind im Papierkorb —
`pruefeSchuelerLoeschbar` blockiert das Löschen bei offenen Ausleihen und unbezahlten Schäden, es
gibt also nichts für den Brief; Rest ist eine Regel in einer anderen Datei) · Rückkehr storniert
die Forderung, das Buch bleibt ausgesondert (beide Türen nehmen die Aussonderung zurück) ·
„Pool ODER Tx" nur behauptet (der Handler öffnet die Transaktion wirklich) · Rechte-Asymmetrie an
den Buch-Routen (`adminH` IST `RequireEditBooks`, nur der Name führt in die Irre) · Migration 115
droppt `idx_lmf_termine_plan` (der neue Unique-Index deckt dieselben Spalten in derselben
Reihenfolge, und alle Lesepfade filtern auf `plan_id` als Präfix).

### 5.15 Rasterdurchgang über die Stufe-2-Commits (15.09.2026, abends)

Sieben Code-Commits seit 5.14 (Idempotenz-Fristen bis Buch-Barcodes), am Code gelesen und jeder
Punkt am echten Postgres nachgestellt. Behoben am 15.09.2026 spät: vorgehende Theken-Uhr,
„Schlüssel bekannt", wiederholte Portion, Littera-Etiketten und ausgesonderte Exemplare in der
Barcode-Liste (Einzelheiten in den Commit-Nachrichten). Die Nachstellung des Stand-Merkers liegt
weiter hinter dem Build-Tag `raster`: `TEST_DATABASE_URL=… go test -tags raster -run TestRaster_
./repository/`.

Die Frage nach den Ausweis-Formen ist am 16.09.2026 entschieden und gebaut
(`frontend/src/lib/scanEinordnen.js`): Vorsilben zuerst, dann die Barcode-Liste, dann die
Rückrechnung des Littera-Etiketts, sonst „unklar". Seither bekommt jeder Ausweis die
Vorsilbe `A-`. Ein Ausweis aus dem Altbestand ohne Vorsilbe (gemessen `B97601826457`) ist ohne
Netz „unklar" und wird abgewiesen — das hängt am Ausweis-Neudruck aus 7.2 und
entscheidet sich mit dem frischen Littera-Backup.

- **B — Ausweis- und Buchnummer werden nur in der Littera-Übernahme gegeneinander geprüft**
  (Frage 3). Seit dem 15.09.2026 vergibt der Personenlauf keine Nummer, die schon ein Buch trägt.
  Wer von Hand eine Ausweisnummer ändert (Schülerakte) oder ein Buch umetikettiert
  (`UpdateCopyBarcode`), wird nicht gebremst — und die Theke löst eine Nummer ohne Vorsilbe zuerst
  als Buch auf. Auf dem Server gibt es heute keine Überschneidung (Zählung oben).
- **B, am SQL belegt, kein Schaden gefunden — der Bewegungsstempel folgt einer Auswahl, nicht einer Regel** (Fragen 1, 3 und 7;
  `repository/bewegungsstempel_pg_test.go`). Aussondern, Bestandskorrektur, Schadensmeldung und
  Soft-Delete stempeln; „Verloren" und Reaktivieren im Status-Editor (`UpdateCopyStatus`), der
  Inventur-Abschluss (`FinishInventurSession`) und `repository/damage.go` stempeln nicht. Die
  Wirkung ist nach Lesart harmlos — ein Scan vor einer Verlustbuchung erklärt den Verlust eher, als
  dass er ihm widerspricht —, aber die Auswahl steht nirgends. Der Test prüft vier Schreiber als
  feste Liste; ein neuer Schreiber ohne Stempel bliebe grün. Durchgespielt: Ein Scan vor einer
  Verlustbuchung ohne Stempel bucht die Wirklichkeit (das Kind hatte das Buch), und Status-Editor
  wie Inventur weisen verliehene Bücher ohnehin ab.
- **B, Mechanismus bestätigt, heute ohne Auslöser — der Stand-Merker der Barcode-Liste rechnet mit dem Beginn der Transaktion** (Frage 6).
  `max(aktualisiert_am)` trägt `CURRENT_TIMESTAMP`, also den Transaktionsbeginn. Ändert eine lange
  Transaktion (Import) ein Etikett und committet nach einem kürzeren Schreiber, ändern sich weder
  Anzahl noch Maximum: 304, die Theke behält die alte Nummer. Nachgestellt:
  `TestRaster_StandMerkerUebersiehtLangeTransaktion` rot. Heute ändert aber nur
  `UpdateCopyBarcode` einen Barcode, als kurzer Einzelbefehl; Einfügen und Aussondern ändern die
  Anzahl. Erst ein Schreiber, der in einer langen Transaktion umetikettiert, macht es scharf — dann
  hieße die Folge „unklar".

Geprüft und in Ordnung: Rechte an den neuen Routen (Liste und Quittieren `view_students`; Tür,
Barcodes und Anzahl `perform_actions`) · Frist der quittierten Meldungen (Job und Rückstands-Wächter
über dasselbe Prädikat, Obergrenze 30 Tage auch bei 0 oder negativer Einstellung) · keine doppelte
Meldung je Schlüssel · ein Serverfehler je Eintrag wird „wiederholen", nicht Erfolg · Rückgabe vor
der Ausleihe fängt `check_return_date` · Sperrreihenfolge Schüler → Ausleihe → Exemplar im
Nachbuchen gehalten · Barcode-Liste ohne Personendaten, Löschen ändert den Stand über die Anzahl.

### 5.16 Leserdatei und Rolle Leitung — was noch offen ist (Stand 16.09.2026)

Gebaut ist alles: die Lesertabelle, die Rolle Leitung, Theke und Leserdatei, eine Maske für
jeden, die Pflicht-Schul-E-Mail und das Zusammenführen von Kollegen. Die Geschichte dazu
steht in den Commit-Nachrichten vom 16.09.2026 (`git log --oneline --since=2026-09-15`).
Hier steht nur noch, was NICHT fertig ist.

**Das Modell in vier Sätzen** — es erklärt die offenen Punkte darunter:

- Alle Leser stehen in EINER Tabelle `leser` mit der Spalte `art` (Schüler · Lehrkraft ·
  LiV). Ausleihen darf jeder aktive Leser; die Art entscheidet nichts.
- `schueler` ist seither eine Sicht auf `leser` mit `WHERE art = 'schueler'`. Sie hält die
  alte Bedeutung für Klassenlisten, LUSD-Abgleich, Mahnlauf und Löschfristen.
- Kollegium ist keine Rolle, sondern der Grundzustand. Eine Rolle (Leitung, Mitarbeiter,
  Helfer, Admin) erhebt der Admin an der E-Mail-Adresse.
- Konten bleiben daneben und tun nur, wofür sie da sind: Anmeldung und Rechte. Jedes Konto
  zeigt auf seine Leserzeile.

---

**A. 15 Minuten: einmal durch die Leserdatei gehen.** Sieh dir an, ob die Wörter
stimmen und ob dir etwas fehlt. Was ein Kollege bewusst NICHT in seiner Akte hat, und warum:

- **Kontoauszug, Ersatzforderung, DSGVO-Auskunft** — sie gehören der Schülerarbeit und lesen
  alle die Sicht `schueler`. Ausgeblendet statt kaputt.
- **Löschen** — steht seit dem 16.09.2026 auch beim Kollegium in der Gefahrenzone. Mit dem
  Eintrag fällt sein Zugang; wird er aus dem Papierkorb zurückgeholt, kommt er ohne Zugang
  zurück, und die Schul-E-Mail in der Akte legt ihn neu an.

**B. Die vier Fragen zum Betrieb sind beantwortet** (16.09.2026) und stehen als
Entscheidung unter D.

**C. Der Nummernkreis bleibt unangetastet — bewusst.** Gemessen auf dem Testserver
(16.09.2026): Exemplare 1…122.127 (30.658 nackte Littera-Nummern, 4.065 `LMF-`, 65 `B-`),
Leser 33 Zeilen, genau EINE Nummer wäre ohne Vorsilbe doppeldeutig. Aus `littera_sav.mdb`:
Exemplare 1…61.512, Leser 0…3.531 — 1.990 von 1.991 Lesernummern sind dort zugleich
Exemplarnummern. Littera lebt damit, weil seine Scanformen sich unterscheiden. Ein
gemeinsamer Nummernkreis ohne Vorsilben wäre möglich, brächte aber nichts, solange die
Offline-Theke die Vorsilbe braucht: **Ohne Netz ist sie die einzige Information, an der die
Theke einen Buchscan von einem Ausweisscan unterscheiden kann.**

**D. Entschieden und nicht mehr zu diskutieren** (steht hier, weil die Frage sonst wiederkommt):

- Die E-Mail eines Kollegen wird NICHT in `leser.eltern_email` abgetippt. Die Spalte gehört
  dem LUSD-Import und heißt auch in der DSGVO-Auskunft „Eltern-E-Mail". Die Adresse steht
  eindeutig am Konto (`benutzer.email`, `UNIQUE lower(email)`); eine zweite Kopie wäre die
  zweite Tür zu derselben Identität. Seit dem 16.09.2026 steht sie in der Akte — gelesen vom
  Konto, und NACHTRAGBAR, solange keine da ist (dann entsteht das Konto). Steht eine da, ist
  das Feld eine Anzeige; geändert wird sie in der Benutzerverwaltung.
- Die Leitung sieht den Menüpunkt „Einstellungen" weiter und darf darin LUSD & Versetzung,
  Datenverwaltung, LMF-Aktionen und Lieferanten bedienen; verschlossen sind Schule, Fristen
  und Mailversand (`manage_settings`) sowie Benutzer & Rechte (`manage_users`). Ein Menüpunkt
  ist kein Recht, sondern ein Sammelpunkt über sechs Kategorien — was gilt, beweist die
  Antwort des Servers. Bestätigt am 16.09.2026.

**E. Offen: Wer bekommt eine Ausweisnummer — und wer nicht?** (Frage vom 16.09.2026, am Code
nachgesehen.) Eine Nummer entsteht heute an genau drei Stellen: „Neuer Leser" (`student_create.go`
vergibt IMMER eine, auch wenn das Feld leer bleibt), der LUSD-Import und die Littera-Übernahme. Ein
Konto vergibt keine: `konto_hat_leserzeile` legt die Leserzeile ausdrücklich ohne Ausweis an. Wer
also über ein KONTO in die Leserdatei gekommen ist — Selbstanmeldung, vom Admin angelegt, oder ein
Admin-Konto aus der Zeit vor Migration 125 ohne Nummer —, hat keine.

Der Druck vergibt auch keine und warnt nicht: `CardFace.svelte` setzt die Nummer in
`/api/barcode?content=…`, und ohne Nummer kommt ein kaputtes Bild und eine leere Zeile auf die
Karte. Genau das zeigt der Lehrerausweis, der die Frage ausgelöst hat.

Von Hand geht es: Das Feld „Ausweisnummer" steht in der Maske für jeden. Nur sagt sein Hinweis beim
Kollegium „Leer lassen, solange kein Ausweis gedruckt ist" — und beim Anlegen stimmt das nicht,
weil der Server dann selbst eine zieht.

**Entschieden am 16.09.2026: Die Nummer entsteht beim Anlegen des Kontos.** Wer ein Konto
bekommt, bekommt damit eine Ausweisnummer — auch wer nie an die Theke kommt. Damit gibt es den
Zustand „Leserzeile ohne Nummer" nicht mehr, und keine Karte kann ohne Barcode aus dem Drucker
kommen.

**Gebaut wird erst nach deinem Ja (17.09.2026).** Nicht weil die Entscheidung offen wäre,
sondern weil die Umsetzung eine Migration braucht: Sie trägt allen Lesern ohne Nummer eine nach,
und vergebene Nummern werden nie recycelt. Vorher die Zählung am Server:

```sql
SELECT count(*) FILTER (WHERE barcode_id IS NULL) AS ohne_nummer,
       count(*)                                   AS leser_gesamt
FROM leser WHERE deleted_at IS NULL;
```

Wichtig bei der Umsetzung: **ein Generator.** Die Nummer zieht denselben Weg wie „Neuer Leser"
(`GetNextSequence` über `leser.barcode_id`, Vorsilbe `A-`); eine zweite Vergabe in SQL wäre der
Fehler aus Migration 068 in neuer Form (zwei Generatoren, ein Nummernkreis). Dazu zwei kleine
Dinge, die daran hängen: die Leser aus der Zeit davor bekommen ihre Nummer nachgetragen
(Migration, nicht von Hand), und der Hinweis am Feld „Ausweisnummer" („Leer lassen, solange kein
Ausweis gedruckt ist") stimmt dann nicht mehr und fällt.

### 5.17 Rasterdurchgang über den 15. und 16.09.2026 (Funde vom 16.09.2026)

Die zwölf Fragen über alle Änderungen beider Tage. Die Commits vom 15.09. hatten ihren
Durchgang am selben Abend (5.15); alles ab Mitternacht — Rolle Leitung, Migrationen 121 bis
125, Leserdatei, Theken-Suche — war ungeprüft. Gates beim Durchgang: golangci-lint ohne
Befund, svelte-check 0/0, Frontend-Tests 668 grün, Go-Suite mit echtem Postgres grün.

Sechs Funde, vier davon noch am selben Tag behoben und hier gelöscht: der Test außerhalb von
Git (`4edcf1b8`), das Zusammenführen eines doppelt stehenden Kollegen (`bf36df57`), die nicht
mehr änderbare Art (`cd46fc44`, `ecd007bd`) und die veralteten Zahlen in
`docs/invarianten.md`. Die beiden, an denen eine Entscheidung hing, sind am 16.09.2026
entschieden und gebaut (keine Forderung gegen einen Kollegen; die Leserzeile geht mit dem
Konto, solange nichts an ihr hängt). Übrig ist einer.

**Übrig aus diesem Durchgang: Ein zweites Konto derselben Person erzeugt eine zweite
Leserzeile.** Der Wächter `trg_benutzer_hat_leserzeile` hängt jedem Konto ohne Leserzeile
eine an, und die Zuordnung zur vorhandenen läuft über die Schul-E-Mail, die am KONTO steht.
Wird ein Konto gelöscht und später ein neues angelegt, entsteht deshalb wieder eine zweite
Zeile. Repariert wird das mit „das ist dieselbe Person" (Zusammenführen); entstehen lassen
sollte man es trotzdem nicht. Die Waisen-Zeile einer ABGELEHNTEN Anfrage gibt es seit dem
16.09.2026 nicht mehr — sie geht mit dem Konto, solange nichts an ihr hängt.

Die übrigen Funde dieses Durchgangs sind behoben; die Einzelheiten stehen in den
Commit-Nachrichten vom 16.09.2026.

**Was der Durchgang ausdrücklich in Ordnung fand:** die Sicht-Falle (`CREATE VIEW … SELECT *`
friert die Spalten ein) hat ihr eigenes Gate (`db/sicht_schueler_vollstaendig_pg_test.go`); das
Gegenrichtungs-Inventar ist zu allen zehn Migrationen nachgezogen; die Rechte der Leitung sind
abgeleitet statt abgeschrieben (Migration 122 aus den ADMIN-Zeilen, `db/rolle_leitung_test.go`
aus der Vorgabe); Mahnlauf, LUSD-Abgleich und Löschjob lesen weiter Schüler; die Dauerleihe
hängt jetzt an der Art statt an der Tabelle; der Name eines Kontos und seiner Leserzeile werden
in einer Transaktion geschrieben, und ein stiller Null-Treffer ist dort ein Fehler.

---

### 5.18 Rasterdurchgang über die Leserdatei-Arbeit (16.09.2026, abends)

Umfang: die 22 Commits seit dem Durchgang aus 5.17, also die ganze Leserdatei-Arbeit
(Anlegen, Ändern, Löschen, Zusammenführen, Schul-E-Mail, Ausweis-Vorsilbe, keine Frist fürs
Kollegium). Keine neue Migration in diesem Zeitraum — Frage 12 hing am eingefrorenen Inventar.
Drei Funde, jeder am laufenden Pfad nachgestellt; der erste ist behoben.

Fund 1 (die Waisen-Leserzeile einer abgelehnten Zugangsanfrage) ist am 16.09.2026 behoben:
Die Leserzeile geht mit dem Konto, solange sie unberührt ist. Welche Tabellen „unberührt"
umfasst, liest die Prüfung aus dem Fremdschlüssel-Katalog der Datenbank statt aus einer
Aufzählung im Go-Code — wer eine Tabelle an `leser` hängt, bekommt die Prüfung damit
geschenkt. Der Ausweis hält die Zeile ebenfalls; ihn sieht keine Fremdschlüssel-Abfrage,
er steht als Spalte in der Zeile.

**2 · Die Dauerleihe eines Kollegen wird als überfällig gefärbt (B).** In der Akte am 16.09.
behoben; zwei weitere Ausgänge blieben: `components/BorrowersListe.svelte` und
`utils/ausleiherDruck.js` färben die Frist rot, sobald das Datum vorbei ist. Beide hängen an
derselben Abfrage (`api/copy_admin.go`, Ausleiher eines Titels), und die holt `ist_handapparat`
gar nicht erst ab. Das Gate gehört deshalb an den fertigen Inhalt, nicht an die Komponente.

`repository/ueberfaellig_regel_test.go` konnte das nicht finden: Es sucht den Vergleich
`rueckgabe_frist < CURRENT_TIMESTAMP` im SQL, aber diese Abfrage vergleicht nichts — sie liefert
das Datum aus, verglichen wird in JavaScript. Die Ratsche misst die falsche Schicht; der
Gegenpart im Frontend fehlt.

Am Schulserver gemessen (16.09.2026): 0 offene Kollegen-Ausleihen ohne das Merkmal — bei 0
offenen Kollegen-Ausleihen überhaupt. Es gibt also nichts zu reparieren, und die erste Zahl
allein hätte nichts bewiesen. Entschieden wird weiter an `ist_handapparat`, nicht an
`klasse = 'Lehrer'`: Das wäre die zweite Wahrheitsquelle von der anderen Seite.

**Raster-Frage 13 ist am 17.09.2026 aufgenommen und gebaut** — Text in `docs/invarianten.md`,
Anker in `docs/schreibpfade_gegen_sicht_test.go` (neun Schreibpfade gegen die Sicht `schueler`,
je mit Begründung; eine zehnte Zeile ist eine Frage). Die Schärfung von Frage 12 („Wer räumt weg,
was die Datenbank selbst angelegt hat?") steht dort ebenfalls; offen bleibt daraus die eine
Beobachtung: Für `klassen` gibt es im ganzen Go-Code kein `DELETE` — eine vertippte Klasse steht
ab dann in jeder Auswahlliste.

**Ausdrücklich NICHT vorgeschlagen**, damit die Frage nicht wiederkommt: eine Frage zur
Barrierefreiheit (hat ein Gate, andere Achse), zum Betrieb (drei Doku-Tests gaten das schon), zur
Eingabeform (Gate seit 13.09.) und zum Rückweg eines Releases (Down-Migrationen wären ein großes
Projekt gegen ein Risiko, das mit dem Nachtbackup bewusst angenommen ist). Geld wurde geprüft und
für unauffällig befunden: Die Summe eines Bescheids wird zweimal gerechnet, im Browser und in Go,
beide als Fließkommazahl — eine erreichbare Abweichung liess sich aber nicht konstruieren, weil
der Fehler bei zweistelligen Eingaben weit unter einem halben Cent liegt und `NUMERIC(10,2)` der
Anker ist. Bleibt als Fleck ohne Fund notiert, nicht als Arbeit.

---

## 6. Beobachten und Kategorie C (nur mit Anlass)

### 6.1 Beobachtungen

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
- Tote CSS-Klassen in `altlasten.css`; toter `logout()` in
  `frontend/src/inventur/lib/store.svelte.js`.
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
- `docs/docs.go` (Swagger) nennt noch `personenart`, `lehrer_id` und `active_teacher_id`.
  Der Generator `swag` läuft unter Go 1.27 nicht mehr durch (er stolpert über die
  Standardbibliothek); das Drift-Gate prüft nur die Endpunkte, nicht die Feldnamen. Beim
  nächsten Anfassen der API-Doku mit einer neueren `swag`-Fassung erzeugen.
- `auth.Claims.BarcodeID` liest niemand mehr; die Ausweisnummer kommt seit Migration 125
  als LEFT JOIN aus der Leserzeile in die Sitzung, nur damit das Feld gefüllt bleibt.
- Tabellen-Inline-Felder mit 36 px: eine `size="sm"`-Variante von `Feld` erst bei Bedienbefund.
- `LabelHeight >= 30` steht zweimal (`api/label_pdf.go`, `api/schueler_etikett_pdf.go`).
- Zwei Normalformen für Namen (`repository.Suchnorm`, `normName` in `api/lusd_paarung.go`); beim
  Anfassen der Paarung zusammenführen.
- Der Paritätstest vergleicht keine COMMENTs und Seeds.
- Jules-Erbe: Go-Testdateien über 200 Zeilen, ein schwacher Export-CSV-Test.
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
- Messung zu 4.17: Liegen echte Schülerdaten auf dem Server?

## 8. Schule, Schulamt, Schulträger

### 8.1 E1: Schulamts- und Schulnummer

Für die Referenznummer der Bescheide; die Felder stehen in den Einstellungen und sind am
13.09.2026 leer. **Erst nach 5.2 und 4.5 eintragen:** Mit den Nummern entstehen echte Bescheide.

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

## 9. Sichtung des Medienzentrums (16.09.2026)

Protokoll vom 16.09.2026, 09:00–11:00 Uhr (Teilnehmende CN, SW, HMP; Protokoll HMP). Die
Einschätzung schließt mit: „Wenn o.g. Mängel abgestellt sind, wäre das Programm nach aktueller
Einschätzung für Schulen nutzbar." Damit ist diese Liste das Abnahme-Gate einer fremden Stelle —
sie steht hier vollständig, mit dem Stand am Code, geprüft am 17.09.2026.

Zwei Quellen liegen dem zugrunde und sind am 17.09.2026 erstmals im Original gelesen worden:
`~/Downloads/Arbeitshilfe_Mahnschreiben.pdf` (Erlass vom 17.12.2014, Az. 674.100.002-00178) und
`~/Downloads/Ablauf Mahnverfahren.pdf` (die Anforderungsliste, abgeglichen in
[mittel_konzept.md](mittel_konzept.md) Abschnitt 3).

### 9.1 Gelöst — mit Nachweis

- **Barcode-Scan ging weder per USB-Handscanner noch per Kamera** (Protokoll 2). EINE Ursache für
  beide Geräte: `code39.Encode(inhalt, true, true)` hängte ein Prüfzeichen an, das IN den
  Strichcode-Daten steht. Aufdruck „A-10003", Strichcode „A-100037" — der Server suchte eine
  Nummer, die es nicht gibt, und ein unbekannter Scan erzeugt nur eine leere Trefferliste.
  Behoben am 17.09.2026 (`dfe14222`, Code 128), dazu die fehlende `code_39` in der Formatliste
  der Kamera (`4fe4388e`), zwei Startstellen für eine Kamera (`a840fdb8`), der stille
  Rückfall-Erkenner (`7bd631f6`) und das ein Jahr lang gecachte Strichcode-Bild (`ea0c4118`).
- **Littera-Barcodes einlesbar?** (Protokoll 3) Ja. Littera druckt die Mediennummer im Klartext
  und codiert eine EAN-13; sie wird zurückgerechnet — an der Theke
  (`internal/service/littera_etikett.go`) und ohne Netz (`frontend/src/lib/litteraEtikett.js`),
  beide gegen dieselben Prüffälle (`litteraEtikett.faelle.json`).

### 9.2 Alte Ausdrucke — GELÖST am 17.09.2026

Jeder Ausweis und jedes Etikett von VOR dem 17.09.2026 trägt Code 39 mit Prüfzeichen; das
Lesegerät liefert „B-100016", und die Nummer gibt es in keiner Tabelle. Entschieden (Peter,
17.09.2026): Alte Barcodes sollen weiter funktionieren — das war von Anfang an gefordert.

Gebaut: Findet ein Scan nichts, wird er ein zweites Mal ohne Mod-43-Prüfzeichen nachgeschlagen
(`pkg/code39`, in `ProcessQuery` an EINER Stelle, offline in `scanEinordnen.js`). Nur als
ZWEITER Versuch — bei 43 möglichen Zeichen sieht sonst jeder 43. gültige Code zufällig so aus.
Beide Seiten teilen 15 Prüffälle; die Rechnung ist an den Balken der Druck-Bibliothek
gegengeprüft. Der Kommentar in `api/barcode_generate.go` stimmt jetzt.

Neu drucken muss die Schule damit nichts. Wer es trotzdem tut, bekommt Code 128 — schmaler und
ohne Prüfzeichen.

### 9.3 Vorgaben des Landes (Protokoll 1)

**9.3 a) Restwertberechnung beim Melden — GEBAUT am 17.09.2026.** Der Dialog holt den Vorschlag
jetzt vom Server (`GET /api/buecher/exemplare/{id}/ersatzwert-vorschlag`) und zeigt die
Herleitung unter dem Feld; die feste 15,00 € ist weg. Zwei Regeln: Lernmittel nach der Staffel
der Arbeitshilfe, Bücherei-Bestand zum Neuwert (Benutzungsordnung, Konzept 1.2) — dort ohne
Altersabschlag, aber mit dem Beschädigungsgrad. Die Schuljahr-Zählung teilt sich die Quelle mit
dem Bescheid-Weg.

**An diesem Punkt ist nichts mehr offen:** Der Beschädigungsgrad in Prozent
(Anforderungsliste Nr. 2) und der Listenpreis (Nr. 3) sind am 17.09.2026 gebaut — 9.8.

**9.3 b) Der Listenpreis — GEBAUT am 17.09.2026 (Migration 127).** Er kommt aus der DNB, die
ihn in jeder ISBN-Abfrage mitliefert; welcher Preis gilt, ist zusätzlich einstellbar. Der
Befund, der dahin geführt hat, bleibt hier stehen, weil er die Entscheidung trägt: Bisher als Komfortfrage
geführt (**4.5**, E4). Die Arbeitshilfe ist eindeutig: ab dem zweiten Verleihjahr sind es 80 %
„des Neupreises des Lehrwerks **zum Zeitpunkt des Verlusts**", und die Anforderungsliste Nr. 3
verlangt beide Preise „auswählbar … welcher Preis als Berechnungsgrundlage verwendet wird".
Bis zum 17.09.2026 übergab `api/bescheid_handler.go` als Neupreis hart `0`, und der Dialog
schrieb „Kaufpreis (kein Neupreis hinterlegt)" — ehrlich, aber nicht die Vorgabe. Damit war 4.5
keine Komfortfrage, sondern Teil dieses Gates.

**9.3 c) Sperrung bei offener Bearbeitung — LMF untersagt das.**
`internal/service/loan_checkout_validation.go` (`pruefeOffeneSchaeden`) sperrt bei jedem
unbezahlten Schadensfall jede weitere Ausleihe — **ohne Ausnahme für Lernmittel**. Dasselbe im
Geräte-Pfad (`pruefeGeraetAutomatikSperren`). Übergehbar ist es nur von Hand mit Audit-Eintrag;
der Grundzustand ist die Sperre. Woher diese Regel stammt, ist nicht belegt: In der
Arbeitshilfe zum Erlass vom 17.12.2014 und in der Anforderungsliste „Mahnverfahren" steht
zur Sperre nichts (beide am 17.09.2026 im Original gelesen). Die Zeile im Konzept, die sie
als fremde Praxis auswies, war unbelegt und ist entfernt. **Frage geht an das
Medienzentrum, siehe 9.7.**

**9.3 d) Zugangs- und Abgangsbuch.** `erworben_am` trägt das echte Littera-Zugangsdatum,
`aussonderung_grund` trennt VERLUST / AUSSORTIERT / BESTANDSKORREKTUR.

**Das Abgangsdatum ist gebaut (17.09.2026, Migration 128).** `ist_ausgesondert` war ein Ja/Nein
ohne Zeitpunkt, und `letzte_bewegung_am` wird von jeder späteren Bewegung überschrieben — ein
Abgangsbuch daraus hätte rückwirkend seine Zeilen geändert. Jetzt steht `ausgesondert_am` am
Exemplar, gesetzt von einem TRIGGER: Ausgesondert wird an sechs Stellen im Code (Status-Editor,
Aussondern, Ausbuchen, Schaden melden, zweimal Bestandskorrektur der Inventur), und die siebte
hätte den Stempel vergessen. Zurückgeholt löscht ihn wieder — sonst führte das Abgangsbuch
Bücher, die im Regal stehen.

Kein Nachtragen für den Altbestand: Was vor der Migration ausgesondert wurde, hat NULL. Ein
erfundenes Abgangsdatum sähe aus wie eine Tatsache und stünde falsch in einem Bestandsnachweis;
der Ausdruck schreibt „Zeitpunkt unbekannt" hin. Gates: `api/abgangsdatum_pg_test.go` (vier
Türen, keine Verschiebung beim zweiten Update, Rückholen, kein erfundenes Datum, dazu eine
Quelltext-Ratsche gegen Schreiber am Datum) und
`inventur/abgangsdatum_bestandskorrektur_pg_test.go` (die zwei Türen im anderen Paket). Am
Trigger-losen Schema rot gesehen.

**Der Ausdruck ist gebaut (17.09.2026).** Medienkatalog → Reiter „Abgangsbuch": Zeitraum
vorbelegt mit dem laufenden Schulhalbjahr (Stichtage 15.3./15.9., dieselben wie die
Bestandskartei), beide Felder überschreibbar; zwei Abschnitte je Topf mit eigener Stückzahl;
Blatt zum Abheften über `GET /api/bestand/abgangsbuch/pdf`. Der Zeitraum wird am Server
bestimmt — die Oberfläche rechnet ihn nicht selbst nach, sonst deckte der Ausdruck einen
anderen ab als die Liste davor. Unter dem Blatt steht die Zahl der Abgänge OHNE Zeitpunkt,
statt Vollständigkeit zu behaupten. Er sitzt im Medienkatalog und nicht im Druck-Center: Er
ist ein Bestandsnachweis, kein Etikettendruck — und daneben ist Platz für das Zugangsbuch.

Gates: `api/abgangsbuch_pg_test.go` (Ränder des Halbjahres, Topf-Trennung, Zurückgeholtes,
Blatt aus dem Inhaltsstrom), `frontend/e2e/abgangsbuch.spec.js` (am Draht, am Rückbau der
Topf-Trennung rot gesehen), `pkg/schulzeit/halbjahr_test.go`.

**Offen bleibt das Zugangsbuch:** aus den Daten ableitbar
([mittel_konzept.md](mittel_konzept.md), Abschnitt 7.1: „je Schulhalbjahr ein Ausdruck der
Neuanschaffungen"), steht dort aber unter „Später, kein Teil dieses Pakets". Der Reiter im
Medienkatalog hat Platz dafür.

**9.3 e) Mahnwesen entspricht nicht den Vorgaben.** Aufgeschlüsselt gegen die Anforderungsliste:

| Nr. | Verlangt                                        | Stand am 17.09.2026                                                                       |
| --- | ----------------------------------------------- | ----------------------------------------------------------------------------------------- |
| 1   | Automatische Abwertung, zeitbasiert             | ✅ **gebaut am 17.09.2026.** Der Buchwert steht an jedem Exemplar der Buchakte, mit Herleitung. Ein ZWEITER, frei eingestellter Abwertungssatz bleibt bewusst aus (zwei Beträge für dasselbe Buch) — gerechnet wird die Staffel des Erlasses. |
| 2   | Beschädigungsgrad in Prozent, Buchwert sinkt    | ✅ **gebaut am 17.09.2026** (`zustand_abwertung_prozent`). Nicht an der Theke: Dort wird gescannt, nicht ausgefüllt — erfasst wird nach der Rückgabe in der Buchakte. |
| 3   | Einkaufspreis UND Listenpreis, wählbar          | ✅ **gebaut am 17.09.2026** (Migration 127 + Einstellung in der Kategorie Schadensersatz). |
| 5   | Versand per Post, E-Mail oder App               | Nur Post — begründet (Schriftform, Datenschutz-Entscheidung A3 vom 22.08.2026).            |
| 6   | Zahlung ohne Rückgabe → Buch **gelöscht**       | Wird ausgesondert statt gelöscht — begründet (die Bestandskartei muss den Abgang nachweisen). |
| 7   | Mahnfrist **sechs** Wochen                      | Vier Wochen — Arbeitshilfe und Verfahrensbeschreibung sagen vier, mit Datum.                |

Die Staffel selbst ist **richtig gebaut**: 1. Verleihjahr voller Kaufpreis, dann 80/60/40/20 %,
ab dem 6. Jahr 10 %. Die Arbeitshilfe sagt „Nach 5 Jahren und für jedes weitere Jahr … 10 %" —
also ab dem sechsten. Am 17.09.2026 am Original nachgelesen.

**Der Zahlungsweg der Altbriefe — GELÖST am 17.09.2026.** Die Arbeitshilfe schreibt „Es darf
kein Schulgirokonto, kein anderes Bankkonto und **keine Bargeldannahme** vorgesehen werden."
`pdf/schadensfall.go` und `pdf/rechnung.go` verlangten beide „bar in der Bibliothek".

Kein pauschales Streichen, sondern zwei Töpfe, zwei Wege (`pdf/zahlungsweg.go`): Für ein
Lernmittel nennen beide Briefe jetzt Zahlstelle und Bankverbindung des Landes — aus derselben
Einstellung wie der Bescheid, eine zweite Kontoangabe im selben Haus wäre eine zweite Wahrheit.
Für ein Buch der Schülerbücherei ist der Weg nicht entschieden (E5, 8.3): Dort steht die
beschlossene Zeile „(Bankverbindung des Schulträgers nicht hinterlegt)" und sonst nichts —
„bar gegen Quittung" ist dort ausdrücklich möglich, aber niemand hat es beschlossen, und ein
Brief, der sich einen Zahlungsweg ausdenkt, schickt Geld an die falsche Stelle.

Eine Rechnung kann beide Töpfe tragen (sie listet ALLE offenen Forderungen und sucht sich ihre
Positionen nicht aus). Dann stehen beide Wege mit ihrer Teilsumme da. Gate am fertigen PDF:
`pdf/zahlungsweg_test.go`, am alten Fuß rot gesehen.

Bar angenommenes Geld bleibt möglich — als Einzelfall mit Quittung und Weiterleitung binnen
14 Tagen, wie es die Arbeitshilfe vorsieht. Was nicht mehr geht, ist ein Brief, der es
VORSIEHT. Offen bleibt die Frage dahinter: Der Knopf „Bezahlt" in der Schülerakte bedeutet
heute Barzahlung am Tresen — für Landesmittel ist das die Ausnahme, nicht der Regelweg (5.1).

Der Elternbrief ist seit dem 15.09.2026 aus der Oberfläche heraus nicht mehr erreichbar, die
Rechnung über den Knopf in `StudentProfileActions.svelte` schon. Ob die Altbriefe ganz
weichen, entscheidet **5.4**.

### 9.4 Titel mit 0 Exemplaren (Protokoll 4)

Entschieden: Ein Titel ohne Exemplare ist ein legitimer Zustand (Anlegen ohne Bestandsangabe,
Altbestand aus Littera) — verstecken wäre falsch. Die Trefferliste muss es SAGEN.

**Hälfte 1 GEBAUT am 17.09.2026 (Daten).** Beide Suchtüren (`SearchTitles` für den
Aktionspfad, `SearchTitlesFuzzy` für die Theke) liefern jetzt `bestand` und `verfuegbar` je
Titel. Die beiden Prädikate stehen an EINER Stelle (`repository/book_bestand.go`) und sind
wörtlich die der Klassenbuch-Abfrage — zwei Auslegungen von „verfügbar" ergäben zwei Zahlen
über denselben Titel, und an der Theke entscheidet diese Zahl, ob jemand ins Regal läuft. Die
Felder sind ZEIGER: Jede andere Abfrage lässt sie nil, damit die Oberfläche nicht „0 Exemplare"
über einen Titel schreibt, dessen Bestand niemand gezählt hat. Vier Ursachen für „kein
Exemplar da" sind am echten Postgres geprüft (gar keins, alle ausgesondert, alle verliehen,
alles im Zulauf).

**Hälfte 2 GEBAUT am 17.09.2026 (Anzeige).** Die Theken-Trefferliste trägt eine vierte
Spalte: „1 von 2 verfügbar", und bei einem Titel ohne Exemplare abgesetzt „Keine
Exemplare". Nur dieser Fall wird hervorgehoben — „0 von 5 verfügbar" ist der Normalfall
im Schuljahr; sähe beides gleich aus, wäre nichts gewonnen. Die Zeile steht auch im
`aria-label`, damit sie nicht allein an der Farbe hängt.

Der Wortlaut kommt jetzt aus `utils/format.js` (`bestandSatz`). Er stand vorher zweimal
im Katalog — mit zwei verschiedenen Antworten auf den Fall „Zahl fehlt": Die Katalog-Kachel
schrieb dort „Keine Exemplare", die Klassensatz-Kachel nichts. Richtig ist nichts, denn
die Zahlen sind Zeiger und nur die Suchabfragen füllen sie.

Nachweis: `frontend/e2e/theke-bestand-im-treffer.spec.js`, am alten Bauteil rot gesehen —
der Titel stand in der Liste und sagte nichts.

**Damit ist Punkt 4 des Protokolls abgeschlossen.**

### 9.5 Schülerdatei ohne Sortierung und Filter (Protokoll 5)

**Filter: GEBAUT am 17.09.2026.** Die Leserdatei hat ein Auswahlfeld „Jahrgang" neben der
Suche; gefiltert wird auf dem Server (`?jahrgang=`), die besetzten Jahrgänge nennt
`GET /api/jahrgaenge`. Die Ableitung „Klassenname → Jahrgang" läuft über
`ausweis.AblaufJahrgang`, damit „ET" als elfter Jahrgang mitkommt und es keine zweite
Auslegung des Klassenschemas gibt.

**Sortierung: GEBAUT am 17.09.2026.** Drei Spaltenköpfe sind Knöpfe (Name, Klasse, Geliehene
Bücher), erster Klick aufsteigend, zweiter dreht um. Sortiert wird am SERVER, aus demselben
Grund wie bei Suche und Filter: Im Browser sortiert säße es hinter der Kappung bei 500 Zeilen
und ordnete die ersten 500 der Kartei-Reihenfolge um statt der ersten 500 der gewählten Spalte.

Zwei Dinge, die dabei entschieden wurden: Die erlaubten Spalten sind eine geschlossene Menge
(ein durchgereichter Spaltenname wäre eine SQL-Injektion, ein unbekannter Wert ist ein 400 mit
den erlaubten Namen). Und das **Kollegium bleibt oben**, auch beim Sortieren — diese
Reihenfolge ist der Schutz gegen die Kappung, ohne sie stünde ein Kollege unter „Z" und wäre
lautlos nicht mehr in der Liste. Sortiert wird innerhalb der Gruppen.

Der sortierbare Spaltenkopf ist ein eigenes Bauteil (`ui/TabelleSortKopf.svelte`) — der erste
der Anwendung, und die Frage kommt bei jeder weiteren Liste wieder.

Beim Bauen nebenbei gefunden und mit behoben: `StudentDirectory.svelte` stand exakt auf der
200-Zeilen-Ratsche (Markierung jetzt in `leserAuswahl.svelte.js`), und der Handler holte sich
die Klassenliste über `s.DB.Pool` statt als Parameter — ein Test ohne Datenbank stürzte darauf
mit SIGSEGV ab.

### 9.6 A: Mehrjahresbände — die Entscheidung von 4.3 kippt

Protokoll 5, zweiter Spiegelstrich: „Bei den Ausleihfristen fehlt das Jahr. Mehrjahresbände
lassen sich nicht abbilden."

Die Mechanik ist vollständig da: `ziel_jahrgang` → `AdditionalYears` →
`stichtag.AddDate(jahre, 0, 0)` in `internal/service/loan_rules.go`. Es fehlt allein die Tür,
die den Wert setzt. **4.3 hält fest: „Entschieden am 16.09.2026: streichen."** Das ist genau das
Gegenteil dessen, was die prüfende Stelle verlangt — und mit der Spalte fiele auch die Rechnung,
die es dafür schon gibt. **Die Entscheidung gehört zurück auf den Tisch, bevor die Messung aus
4.3 läuft.** Solange sie offen ist, wird `ziel_jahrgang` nicht angefasst.

### 9.7 Stand der Fragen

1. **Sperre bei offener Forderung** (9.3 c): Die Frage geht ans Medienzentrum (Text am
   17.09.2026 formuliert): Gilt das Verbot nur für Lernmittel oder für jede Ausleihe, worauf
   stützt es sich, und zählt eine übergehbare Abweisung schon als „Sperrung"? Bis zur Antwort
   bleibt es, wie es ist.
2. **Abwertung und Beschädigungsgrad** — **entschieden am 17.09.2026: so wie das Medienzentrum
   es sagt.** Umsetzung und der Konflikt, der darin steckt: 9.8.
3. **Mehrjahresbände** (9.6): Was das Medienzentrum wörtlich sagt, ist nur der eine Satz „Bei
   den Ausleihfristen fehlt das Jahr. Mehrjahresbände lassen sich nicht abbilden." Mehr steht
   nicht im Protokoll. Die Deutung geht mit an das Medienzentrum (Frage 2 des Schreibens).
   **4.3 bleibt bis dahin gestoppt.**
4. **Alte Strichcodes** — **entschieden am 17.09.2026: sie sollen weiter funktionieren, das war
   von Anfang an gefordert.** Gebaut, siehe 9.2.

### 9.8 Abwertung und Beschädigungsgrad — Bauplan (nach Entscheidung vom 17.09.2026)

Umzusetzen sind die Punkte 1, 2 und 3 der Anforderungsliste „Mahnverfahren":

1. Medien automatisch abwerten, zeitbasiert (z. B. 10 % pro Jahr) oder nutzungsbasiert.
2. Beschädigungen bei Katalogisierung und Rückgabe mit einem Prozentwert erfassen (z. B. 20 %
   durch Wasserschaden); der Buchwert sinkt dadurch.
3. Einkaufspreis UND Listenpreis hinterlegen, für das Mahnwesen auswählbar, welcher gilt.

**Der Konflikt, der in Nr. 1 steckt — und warum er auflösbar ist.** „10 % pro Jahr" ist nicht
die Staffel des Erlasses vom 17.12.2014 (100/80/60/40/20, ab dem 6. Jahr 10 %). Für Lernmittel
ist die Staffel bindend; ein zweiter, frei eingestellter Abwertungssatz ergäbe für dasselbe Buch
zwei Beträge, und im Bescheid an Eltern stünde der falsche. Aufzulösen ist das, weil die Staffel
SELBST die zeitbasierte Abwertung ist — sie wird heute nur nicht als Dauerzustand geführt,
sondern erst im Dialog gerechnet. Was fehlt, ist die Sichtbarkeit, nicht die Rechnung.

Nr. 2 ist dagegen ohne Konflikt: Die Arbeitshilfe stellt den Betrag ausdrücklich ins Ermessen
der Schule „je nach Zustand des Lehrwerks, Ausleihhäufigkeit etc." — ein erfasster
Beschädigungsgrad ist genau die Begründung für dieses Ermessen.

**Stand am 17.09.2026 (Peter hat die volle Umsetzung freigegeben):**

- **Stufe 1 GEBAUT** (Migration 127). `buecher_titel.listenpreis` (nullbar) und
  `buecher_exemplare.zustand_abwertung_prozent` (NOT NULL, 0–100). Der Listenpreis hängt an
  einer Quelle, die es schon gab: Die DNB liefert den Ladenpreis aus MARC21 020 $c in jeder
  Antwort mit (`metadaten_preis.go`), er stand dort ungenutzt — jetzt füllt er beim Anlegen
  über die ISBN das Feld. Verdrahtet über alle vier Schreibwege; Feld in der Katalog-Maske.
- **Stufe 4 GEBAUT.** In den Einstellungen unter „Schadensersatz" steht jetzt der Schalter
  „Immer mit dem Einkaufspreis rechnen". AUS (die Vorgabe, und auch der Zustand jeder Anlage,
  in der niemand etwas einstellt) heißt: ab dem zweiten Verleihjahr der Listenpreis, so wie es
  die Arbeitshilfe verlangt. AN heißt: immer der Preis, den die Schule bezahlt hat — und die
  Herleitung sagt dann „Kaufpreis (so eingestellt)" statt „kein Listenpreis hinterlegt", denn
  das wäre eine falsche Auskunft über die Datenlage. Im ersten Verleihjahr ändert der Schalter
  nichts; dort gilt ohnehin der Kaufpreis. Am Draht geprüft: Schlüssel in der Tabelle → Aufruf
  der Exemplar-Liste → Betrag und Begründung in der Antwort.
- **Stufe 2b GEBAUT.** An jedem Exemplar der Buchakte steht, was ein Ersatz heute kostet —
  mit der Herleitung darunter („3. Verleihjahr → 60 % von 41,50 € (Listenpreis), abzüglich
  20 % für den Zustand"). Die Zahl kommt aus derselben Funktion wie der Vorschlag im
  Melde-Dialog; die Oberfläche rechnet nichts. Für den ganzen Titel eine Abfrage, nicht eine
  je Karte. Und die Status-Tür antwortet mit dem NEUEN Wert: Wer den Wertverlust einträgt,
  sieht sofort, was das Buch damit noch wert ist, statt den alten Betrag daneben stehen zu
  sehen.
- **Stufe 3 GEBAUT (Exemplar-Akte).** Der Wertverlust steht im Status-Editor der Buchakte,
  neben der Notiz, und die Karte zeigt ihn an. Er reist über die BESTEHENDE Tür
  (`PUT /api/buecher/exemplare/{id}/status`) — keine zweite für denselben Zustand. Drei Regeln
  hängen an Tests: Ein fehlendes Feld lässt den Wert unangetastet (sonst löschte jeder
  Statuswechsel einen erfassten Wasserschaden), „Verfügbar" räumt die Notiz, aber NICHT den
  Wertverlust, und ein unmöglicher Wert ist ein 400 statt eines 500.
- **Stufe 2a GEBAUT.** `ersatzwert.Rechne` nimmt den Zustands-Abschlag, und alle drei
  Vorschlagswege liefern Listenpreis und Abschlag durch. Die Herleitung sagt „Listenpreis"
  statt „Neupreis" — ein Wort für eine Sache. **Geprüft** seit dem 17.09.2026: die Rechnung in
  `pkg/ersatzwert` (Abschlag nach der Staffel, Kappung 0–100, einmal gerundet) und der Satz der
  Herleitung in `api` — beide am Rückbau rot gesehen.

**Was als Nächstes dran ist:** nichts mehr an dieser Baustelle — die drei Punkte der
Anforderungsliste zur Abwertung sind abgearbeitet, und die drei offenen Fragen sind am
17.09.2026 entschieden (siehe unten).

**Die vier Stufen im Einzelnen:**

1. **Migration.** `buecher_titel.listenpreis` (Neupreis zum heutigen Tag, nullbar) und
   `buecher_exemplare.zustand_abwertung_prozent` (0–100, Vorgabe 0). Zwei Spalten, keine
   Trigger, keine Rückfüllung — Altbestand startet mit 0 und leerem Listenpreis.
2. **Der Buchwert wird sichtbar.** Am Exemplar steht, was es heute wert ist: Basis (Listenpreis,
   sonst Kaufpreis), Verleihjahr, Staffelsatz, Zustandsabschlag, Ergebnis. Dieselbe Rechnung wie
   der Vorschlag im Melde-Dialog — `pkg/ersatzwert` bekommt den Abschlag als weiteren Faktor,
   eine zweite Rechnung entsteht nicht.
3. **Beschädigung erfassen.** Bei der Rückgabe und in der Exemplar-Akte ein Prozentfeld neben
   der vorhandenen `zustand_notiz`. Der Wert ist ein ZUSTAND des Exemplars, keine Forderung —
   er wirkt auf jeden künftigen Ersatzbetrag, erzeugt aber von sich aus keinen.
4. **Berechnungsgrundlage wählbar.** Eine Einstellung in der Kategorie „Schadensersatz":
   Listenpreis bevorzugen oder immer Einkaufspreis. Die Herleitung nennt weiterhin, welcher
   Preis benutzt wurde.

**Die drei Fragen von heute — entschieden am 17.09.2026:**

1. **Wertverlust an der Theke eintragen? NEIN.** Die Theke ist ein Scanfeld, kein Formular
   (Peter hat zugestimmt). Der Weg bleibt: Buch an der Theke zurücknehmen, danach in der
   Buchakte den Wertverlust eintragen.
2. **Zählt der Zustandsabschlag auch für die Schülerbücherei? JA — gebaut.** Zur Bücherei sagt
   weder der Erlass vom 17.12.2014 noch die Arbeitshilfe ein Wort; beide sprechen ausschließlich
   von Lehrwerken der Lernmittelfreiheit (am 17.09.2026 in der Arbeitshilfe nachgesehen). Was
   dort gilt, ist die Benutzungsordnung. Die Anforderungsliste des Medienzentrums nennt unter
   Nr. 2 „Medien" und nicht „Lernmittel" — also zählt der Abschlag dort auch. Der ALTERSabschlag
   bleibt auf Lernmittel beschränkt.
3. **Neuwert für die Bücherei = Listenpreis. GEBAUT.** Die Benutzungsordnung verlangt „Geld in
   Höhe des Neuwerts", und der Neuwert ist der heutige Preis, nicht der von 2015. Vorher wurde
   mit dem Einkaufspreis gerechnet, weil es keinen anderen gab.

**Dabei aufgefallen und mitbehoben:** Den Betrag rechneten DREI Türen — der Melde-Dialog, der
Bescheid-Vorschlag aus einer Forderung und der aus einer überfälligen Ausleihe. Nur der erste
unterschied Lernmittel von Büchereibuch; die beiden Bescheid-Wege wendeten die Staffel des
Landes auf alles an. Für ein zehn Jahre altes Büchereibuch nannte der Dialog 14,00 € und der
Brief 1,40 € — und der Brief war der falsche. Jetzt rechnen alle drei durch dieselbe Funktion,
und ein Test vergleicht sie mit denselben Zahlen gegeneinander.
