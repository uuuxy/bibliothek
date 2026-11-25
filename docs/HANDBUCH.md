# Benutzerhandbuch

Stand: 2026-11-23. Für Bibliothekspersonal, Sekretariat und Schulleitung — geschrieben aus
Sicht der Arbeit an der Theke, nicht aus Sicht des Codes. Die fachlichen Regeln dahinter stehen
im [Fachkonzept](FACHKONZEPT.md); dort verweisen die §-Angaben hin.

**Zwei Grundsätze, die überall gelten:**

- Es gibt **keine Passwörter** zu verwalten. Angemeldet wird mit dem Schul-Postfach
  (E-Mail + Mail-Passwort) oder, wo das Konto so eingestellt ist, mit dem Benutzernamen und
  Passwort des Schulnetzes (LDAP). Ist die Domain der Schule eingerichtet, genügt der Name vor
  dem @; den Rest ergänzt das Programm. Wer kein Konto hat, kann sich mit dem Postfach selbst
  anmelden und wird von der Bibliothek freigeschaltet (→ Benutzer & Rechte).
- **Jede Rolle sieht nur ihren Teil.** Lehrkräfte sehen nur _Mein Portal_, Helfer nur die
  Theke ohne Leserakten, Mitarbeitende den Thekenbetrieb, Admins alles.

---

## Öffentliche Seiten — ohne Anmeldung

| Adresse                    | Für wen                                                     | Zeigt                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| -------------------------- | ----------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `https://<schule>/katalog` | Schüler, Eltern, Kollegium — vom Handy, aus dem Klassenraum | Suche nach Titel, Autor, ISBN oder Schlagwort; Cover; „N von M verfügbar". Höchstens 50 Treffer — gibt es mehr, steht die Gesamtzahl darüber. Keine Ausleihdaten, keine Namen.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `https://<schule>/monitor` | der Bildschirm vor der Bibliothek | Endlos-Slideshow: Buch des Monats (die meisten Ausleihen in 30 Tagen, mit Cover), Neu eingetroffen (mit Cover), Beliebt diese Woche. Keine Schulbücher, Klassensätze an Lehrkräfte zählen nicht. An einem Tag der Bücherausgabe oder des Büchertauschs (veröffentlichter LMF-Plan) steht vorn die Folie _Heute_ mit Stunden und Klassen des Tages; der Hinweis aus dem Plan ist dort für alle lesbar. Aktualisiert sich von selbst, überspringt leere Folien (Ferien) und lädt sich um 03:00 Uhr neu, um neue Versionen zu holen; eine Tastatur braucht der Bildschirm nicht. |

Beide Seiten haben keinen Menüpunkt. Die fertigen Adressen stehen unter
_Einstellungen → Erreichbarkeit & Alarme_ zum Kopieren. (§16)

---

## Suchen

Jede Seite hat **eine** Suchleiste oben links, Filter und Knöpfe stehen in der Zeile darunter.
Sie sucht, worum es auf der Seite geht: der Medienkatalog Bücher,
Leserdatei und Mahnwesen Menschen, die Inventur scannt. Eine seitenübergreifende
Suchleiste, die von jeder Verwaltungsseite aus zu Buch- oder Leserakte springt, gibt es
nicht. Gebucht wird nur an der Theke.

Namen findet jede Suche gleich, an der Theke, in der Leserdatei, im Mahnwesen, bei den
Abgängern, in der Buchakte bei den Ausleihern und unter _Benutzer & Rechte_: Ein Teil von Vor-
oder Nachname genügt, die Reihenfolge
zählt nicht, Akzente, Umlaute und ihre Ersatzschreibung („ozturk“ findet Öztürk, „mueller“
Müller) ebenso wenig wie Bindestrich, Apostroph und Leerzeichen („alkahil“ findet Al-Kahil,
„annalena“ Anna-Lena). Eine gescannte Ausweisnummer findet den Leser in Leserdatei, Mahnwesen,
Abgängerliste und unter _Benutzer & Rechte_.

Titel findet jede Suche nach derselben Regel: _Suche & Filter_, die Titel-Verwaltung, der
Klassensatz, die Dialoge _Andere Auflage zuordnen_ und _Titel zusammenführen_, das Feld der Theke,
im Druck-Center _Buch-Etiketten_ und _Fehlende Etiketten_, die Suche unter _Bestellungen_, in
_Mein Portal_ _Schulbücher_ und _Reservieren & Melden_ und der öffentliche Katalog. Jedes Wort muss
Titel, Autor, Fach, Signatur, ISBN (mit oder ohne Bindestriche oder Leerzeichen, zehn- oder
dreizehnstellig) oder ein Schlagwort treffen. Ein Kürzel wie „mathe“ findet auch die Langform
„Mathematik“, eine Zahl gilt als Jahrgang, und Umlaute zählen auch ohne Punkte („rauber“ findet
„Räuber“).

## Ausleihe (Theke)

**Vor dem ersten Scan: Der Scanner muss auf deutsche Tastatur eingestellt sein** (Datalogic:
„Country Mode = Germany“, Schritte unter _Wenn etwas nicht geht_). Sonst tippt er den Bindestrich
als ß: Das Programm liest die Nummer trotzdem, jedes andere Programm am Rechner aber nicht.

Der Startbildschirm nach der Anmeldung. **Ein Feld für alles:**

1. **Schülerausweis scannen** → die Theke öffnet sich: Foto, Klasse, Konto-Status mit offenem
   Betrag, Ausleihen, Vormerkungen; Gebühren stehen im eigenen Reiter.
2. **Buch scannen** → ausgeliehen, die Frist rechnet das Programm (Lernmittel bis zum Donnerstag
   vor den Sommerferien, in der Oberstufe bis zum 31.07.; andere Medien nach Tagen, beim
   Ferien-Leseclub bis zu dessen Datum; fällt eine Frist auf Wochenende, Feiertag oder Ferien,
   gilt der nächste Schultag, §2).
3. **Dasselbe Buch erneut scannen** → zurückgegeben. Ein Buch ohne geöffnete Theke scannen →
   sofort zurückgebucht, auch von jemand anderem („Fremdrückgabe“, mit Hinweis).
4. **Namen tippen** statt scannen: Vorschläge erscheinen beim Tippen, Klick öffnet die Theke.

**An jeder Buchzeile:** Verlängern · Zurückgeben · Verlust oder Schaden melden (Grund,
Ersatzbetrag; die Forderung steht danach unter _Gebühren & Schäden_, der Bescheid ist ein eigener
Schritt).

**Der Scan landet immer im Scanfeld,** auch nach einem Klick in der Akte. Nur ein offenes Fenster
und ein Feld, in dem gerade getippt wird, behalten die Tastatur. **Escape** schließt die Akte und
leert das Scanfeld; in einem Feld, einer Liste oder einem Fenster schließt es nur dieses. Hält die
Akte ungespeicherte Stammdaten, fragt die Theke vorher (_Verwerfen und weiter_).

**Was die Theke nicht bucht** (rot, mit Fehlerton):
- einen Scan, während eine Rückfrage offen ist („Ausleihe blockiert“, „Achtung! Vorgemerkt!“,
  „Zubehör prüfen“): erst das Fenster beantworten, dann erneut scannen;
- einen Scan, während in _Stammdaten bearbeiten_ etwas Ungespeichertes steht („Nicht gebucht: Erst
  die Eingaben in der Maske speichern oder verwerfen, dann erneut scannen.“); ein Scan in ein Feld
  der Maske landet in diesem Feld;
- was nach einem Scan gescannt wurde, der nicht glatt durchging (Fehler, Rückfrage, Hinweis): Die
  Nummern stehen rot unter dem Scanfeld („Gescannt und NICHT gebucht, bitte erneut scannen: …“).
  Sonst bucht die Theke schnelle Scans der Reihe nach, auch bevor der vorige beantwortet ist, und
  dieselbe Nummer zweimal hintereinander zählt einmal.

**Am Ton ist zu hören, was gebucht wurde:** Eine Ausleihe klingt mit zwei Tönen abwärts, eine
Rückgabe und ein Ausweis mit denselben zwei Tönen aufwärts. Klingt es beim Zurücknehmen abwärts,
ist das Buch ausgeliehen worden: erneut scannen nimmt es zurück. Nach einem unbekannten Ausweis
ist das Konto davor noch geöffnet.

**Ein Stapel vom Rückgabetisch** geht über die **Schnellrückgabe**: den Knopf rechts neben dem
Scanfeld anklicken, dann die Bücher scannen. Jedes verliehene Buch wird zurückgebucht, die Meldung
nennt, bei wem es war; ein Konto öffnet sich nicht, ausgeliehen wird nichts. Ohne sie öffnete die
erste Rückgabe das Konto ihres Lesers, und ein Buch aus dem Regal ginge an ihn. Sie endet mit einem
zweiten Klick, mit Escape, mit einem Ausweis und wenn sich die Theke von selbst leert.

**Die Theke warnt von selbst:**

- Beim **Ausgeben von Schulbüchern** blitzt die Theke orange, und über dem Konto steht eine Zeile,
  wenn etwas nicht zur Klasse passt; die Ausleihe ist trotzdem gebucht, erneut scannen nimmt sie
  zurück. Die Zeilen bleiben bis zum nächsten Kind stehen (§18.6):
  - Das Kind hat das Buch schon (_‹Vorname› hat „Mathematik 8“ jetzt zweimal_).
  - Kein anderes Kind der Klasse hat es, obwohl mehr als die Hälfte ihre Bücher schon hat; beim
    Büchertausch zählen die alten Bücher nicht mit, in der Oberstufe kommt der Hinweis nie. Bei
    einem Wahlfach wie Latein kann er beim ersten Kind kommen, obwohl alles stimmt.
  - Eine andere Auflage, als Kinder der Klasse sie haben (nur bei zusammengefassten Auflagen):
    Andere Auflagen haben andere Seitenzahlen.
- Kommt ein **vorgemerktes** Buch zurück, erscheint ein roter Hinweis: nicht ins Regal, es liegt
  drei Tage (bis zum nächsten Schultag) für die nächste Leserin bereit (§4).
- Ein **gesperrter** Ausweis wird mit dem Grund angehalten, dann geht nur die Rückgabe. Wer „Leser
  bearbeiten“ darf, hebt die Sperre im Dialog auf („Sperre aufheben“) (§2.2).
- Eine **offene Forderung** oder zu viele überfällige Bücher halten Schülerbücherei und Geräte an;
  wer „Leser bearbeiten“ darf, übergeht das einmalig („Einmalig ignorieren“).
- Ein **Schulbuch** geht trotzdem raus, ohne Rückfrage: Die Lernmittelfreiheit lässt keine
  automatische Sperre zu. Nur eine Sperre von Hand hält es an, auch beim Verlängern (§2.2).
- **Kollegen** werden nie angehalten. **Überfällige** Bücher sind an der Zeile gekennzeichnet.

**Außerdem:** Geräte laufen über dieselbe Theke, mit Zubehör-Checkliste beim Scan (§5) · Kamera
statt Handscanner (Knopf neben dem Feld) · Passbild per Webcam · Ausweis, Kontoauszug und
DSGVO-Auskunft als PDF (§18).

**Leeren und Sperren:** Nach 5 Minuten ohne Eingabe schließt sich die Akte, nach 15 Minuten kommt
der Sperrbildschirm (einstellbar unter _Datenschutz & Sitzung_); 10 Stunden nach der letzten
Passworteingabe fragt er wieder, auch mitten in der Arbeit („Gesperrt nach 10 Stunden“, nicht
einstellbar). Weiter geht es mit dem eigenen Passwort, auch nach dem Neuladen und wenn der
Mailserver nicht erreichbar ist; Ungespeichertes steht danach noch da. Abmelden fragt vorher, wenn
etwas verloren ginge (_Verwerfen und abmelden_). Ein Scan am Sperrbildschirm zählt nicht als
falsches Passwort („Scan erkannt“).

---

## Medienkatalog

- **Suche & Filter**: ein Feld für Titel, Autor, Fach, Klasse, Signatur und Schlagwort,
  Kartenansicht mit Cover und Signatur. Jedes Wort zählt für sich, ein Kürzel wie „mathe“ steht
  für das Fach und eine ein- oder zweistellige Zahl für den Jahrgang („mathe 7“); Umlaute zählen nicht
  („rauber“ findet „Die Räuber“). Wo sonst Titel gesucht werden, gilt dieselbe Regel (_Suchen_). Ein Barcode, gescannt oder getippt, zeigt den Titel des
  Exemplars, auch über das alte Littera-Etikett und eine lange LMF-Nummer. Ein Verweis findet sein
  Schlagwort („Tierfantasy“ findet „Fantasy“, wenn es unter _Einstellungen → Schlagworte_ so
  eingetragen ist), auch in der Titel-Verwaltung, im öffentlichen Katalog und in _Mein Portal_.
  Sortiert wird nach dem Titel: Umlaute beim Grundbuchstaben, Zahlen nach ihrer Größe („Mathe 5“ vor
  „Mathe 10“), ein Artikel zählt mit („Die Räuber“ unter D).
- **Buchakte** (Klick auf eine Karte): Exemplare mit Status, Ausleiher, Vormerkungen (Warteliste mit
  Schüler-Suche; vormerken geht nur, wenn kein Exemplar frei im Regal steht), Historie; beim Kollegium steht statt der Klasse die Art („Lehrkraft“,
  „Fachbereich“). Das Drucksymbol an einem Exemplar legt sein Etikett ins Druck-Center. Je Exemplar
  steht:
  - **was es heute noch wert ist**, samt Herleitung („3. Verleihjahr → 60 % von 24,90 €“). Einen
    Schaden tragen Sie nach der Rückgabe im Zustands-Dialog als Prozentwert ein (etwa 20 % für einen
    Wasserschaden): Er mindert jeden künftigen Ersatzbetrag, das Buch bleibt im Verkehr (§14).
  - **wem es gehört**, Land oder Schulträger, und woher die Angabe kommt (Littera, von Hand,
    Bestellung oder Vorgabe aus dem Titel). Danach richten sich Etikett, Bestandsbücher und
    Schadensersatz. Ändern: Exemplare ankreuzen, _Eigentum ändern_ (Land, Schulträger oder _Vorgabe_),
    mit Grund; jede Änderung steht im Protokoll.
  - sein **Standort**, wenn es nicht an seinem Platz nach der Signatur steht („Bibliothek, Regal
    3B“). Ändern: ankreuzen, _Standort ändern_ (mit Vorschlägen aus dem Bestand) oder _Standort
    entfernen_. Ein neues Exemplar erbt den Standort, wenn alle Exemplare des Titels denselben
    tragen. An der Ausleihe ändert er nichts: Ein Titel ist Bibliothek oder Lernmittel, mit allen
    Exemplaren.
- **Titel-Verwaltung**: Titel anlegen (die ISBN holt die Angaben, vor dem Speichern prüfen; die
  Signatur trägt man selbst ein), bearbeiten, Cover tauschen, Exemplare aussondern. Pflicht ist der
  Titel, die ISBN darf fehlen (Zeitschrift, Spiel, altes Buch). Heißt beim Anlegen ohne ISBN schon
  ein Titel mit demselben Autor gleich, fragt die Maske _Ist es dasselbe Medium?_: _Titel öffnen_
  führt zum vorhandenen, _Anderes Medium_ legt an. Die Gruppe _An der Schule_ enthält _Bibliothek |
  Lernmittel_, Signatur, Fach, Schulzweig, Jahrgang, Mehrjahresband und Auflagen; Bestand und
  Zähldatum stehen unter _Exemplare_.
  - **Bestand ändern:** Nach oben legt das System die fehlenden Exemplare mit B-Nummer an (danach
    unter _Fehlende Etiketten_ im Druck-Center); nach unten fragt die Maske mit beiden Zahlen und
    sondert zuerst nicht ausgeliehene aus, bestellte nie. Hat inzwischen ein anderer Platz den
    Bestand geändert, lehnt das Speichern ab und zeigt den neuen Stand.
  - **Schlagworte aus der DNB:** Holt die ISBN Titel und Autor, bietet _Schlagworte_ die Wörter der
    DNB zum Anklicken an. Bei einem vorhandenen Titel ohne Schlagworte kommen sie beim Öffnen von
    selbst; trägt er schon welche, holt sie _Vorschläge aus der DNB holen_. Eingetragen wird nur, was jemand anklickt, und erst mit dem Speichern (§13).
  - **Schließen:** Das Kreuz oben links; mit ungespeicherten Eingaben fragt die Maske vorher
    (_Bleiben_ / _Verwerfen_), ebenso beim Wechsel des Menüpunkts (_Verwerfen und weiter_).
- **Löschen:** _Titel löschen_ (Buchakte, Maske, in der Titelliste für mehrere) fragt nach dem
  Grund, vorgewählt _Alter Bestand_; die Exemplare im Haus kommen damit ins Abgangsbuch. Verliehene
  nennt der Dialog mit Leser: Sie bleiben beim Leser und der Titel, bis sie zurück sind, außer man
  kreuzt _Auch diese … Exemplare löschen_ an. Ein einzelnes Exemplar löschen Sie in der Buchakte;
  ein verlorenes verliehenes Buch melden Sie in der Leserakte (_Verlust oder Schaden melden_) (§13).
- **Auflagen** (bei Lernmitteln): Eine neue Auflage ist ein eigener Titel mit eigener ISBN, denn die
  Seitenzahlen sind andere. _Andere Auflage zuordnen_ fasst Auflagen zusammen (findet auch Titel
  ohne Exemplar); das Symbol am Ende einer Zeile löst eine wieder. In _Suche & Filter_ ist ein so
  zusammengefasstes Buch eine Karte mit der Summe aller Auflagen („Bestand aus 2 Auflagen: 4. Aufl.
  · 2023 (16), 3. Aufl. · 2019 (42)“); die Titel-Verwaltung listet jede einzeln (§13).
- **Dasselbe Buch zweimal** (etwa aus einer Liste ohne ISBN und aus dem Bestands-Import mit ISBN):
  _Mit anderem Titel zusammenführen_ in der Maske, anderen Titel suchen, antippen, _Zusammenführen_.
  Der Titel mit ISBN bleibt und bekommt alle Exemplare und die fehlenden Angaben; die Zeile vorher
  nennt die Zahl der Exemplare danach, verdoppelt sie sich, standen dieselben Bücher an beiden.
  Tragen beide eine ISBN, sind es zwei Ausgaben: dann _Andere Auflage zuordnen_. Den Knopf hat, wer
  Titel löschen darf (§13).
- **Mehrjahresband**: Bleibt ein Lernmittel über seine Spanne „von … bis“ beim Kind, setzt man den
  Haken bei „Mehrjahresband“; der Hinweis rechnet mit („Bleibt beim Kind bis zum Ende von
  Jahrgang 9.“). Nur mit einer Spanne über mehr als einen Jahrgang (§2).
- **Jahrgang**: zwei Felder, „von“ und „bis“; „bis“ geht mit „von“ mit, solange es leer ist oder
  dieselbe Zahl zeigt (7, Tab, 10 ergibt „7 bis 10“). Die ISBN-Abfrage schlägt ihn nur vor, wenn der
  Titel ihn als Schulstufe nennt („Klasse 7“, „Klassen 7–10“), nicht aus „Band 2“ oder „Green Line
  5“. Bleiben beide Felder leer, ist der Jahrgang unbekannt: Der Titel liegt in keiner Inventur nach
  Klasse und steht im Portal unter keinem Jahrgang. Nur eines von beiden lässt sich nicht speichern.
- **Titel ohne Exemplar** stehen in keinem Katalog und keiner Trefferliste (ein bestelltes Exemplar
  zählt schon). In der Titel-Verwaltung zeigt sie der Umschalter _Mit Exemplaren | Ohne Exemplare_:
  Dort bekommt so ein Titel Exemplare, oder er wird gelöscht.
- **Geräte**: anlegen mit Modell, Seriennummer, Barcode `G-…` und Zubehör-Checkliste. Der Barcode
  bleibt fest, er klebt am Gerät; das Übrige lässt sich ändern.

## Signaturen

Sachgruppen (Kürzel + Bezeichnung, z. B. _Jug_ – Jugendliteratur) pflegen und Regale per
Präfixsuche durchsehen („Jug" findet „Jug Her", „Jug Pre" …). Trägt ein Buch eine zweiteilige
Aufschrift wie „LMF Deu 7 / Bie" — so kommen die Signaturen aus Littera —, fasst die Liste
nach dem Teil vor „ / " zusammen; am Titel und im Regal steht die ganze Aufschrift. Dasselbe
gilt für die Vorschläge im Feld _Signatur_ und für die Auswahl der Inventur. Wer dort auf einen
Vorschlag zeigt, sieht daneben bis zu zwei Cover von Titeln unter dieser Signatur. Wer einen
Vorschlag wählt, bekommt die ersten drei Buchstaben des Nachnamens dazu („Sk / Bie“), im
Bestellfenster wie beim Anlegen; das Feld bleibt änderbar. (§13)

## Druck-Center

Vier Reiter (§9):
- **Buch-Etiketten**: Format, Startposition auf einem angebrochenen Bogen, Titel aus dem Katalog oder
  aus einer Klasse; auch das große Lernmittel-Etikett mit Ausleihtabelle (vier je A4-Blatt) und das Signatur-Etikett für den Buchrücken (Regaladresse und Kürzel
  groß, auf dem Bogen der kleinen Barcodes; es nimmt kein Exemplar von „Fehlende Etiketten“). Die
  Vorschau ist das Druck-PDF selbst („Bogen 1 von 3 · 68 Etiketten“).
- **Fehlende Etiketten**: alle Exemplare ohne Etikett; die Zahl steht an der Navigation.
- **Ausweis-Design**: Vorder- und Rückseite. Bis zur gestrichelten Linie druckt der Kartendrucker
  sicher: Text, Bilder und Strichcode gehören innerhalb, Farbflächen dürfen bis an den Rand. Steht
  etwas zu nah am Rand, wird die Linie rot, und der Satz unter der Karte sagt warum.
- **Ausweise je Klasse**: Klasse wählen; die Zeile darunter nennt, wie vielen die Ausweisnummer fehlt
  (sie werden nicht gedruckt). Einzelne Schüler druckt die Leserdatei.

Gedruckt wird über den Druckdialog des Browsers. Der Kartendrucker heißt dort unter Windows „XPS Card
Printer“ (Datacard); „Microsoft XPS Document Writer“ schreibt nur eine Datei.

## Klassensätze

Welche Klasse hat welche Lektüre? Klasse suchen, _Klassensatz anlegen_: Bücher und Zielklasse
wählen, speichern. Zur Wahl stehen die Klassen der Schüler und die mit einer Buchliste; neue Klassen
kommen mit dem LUSD-Abgleich. Der Papierkorb an einer Klasse löscht nur ihre Buchliste.
Reservierungen aus dem Kollegium stehen im Bestellwesen (§4).

Die Liste ergänzt sich von selbst: Hat mehr als die Hälfte einer Klasse (mindestens fünf Kinder)
denselben Titel ausgeliehen, erscheint er mit dem Abzeichen _aus Ausleihen · n Leser_, aus den
Ausleihen gerechnet, nicht gespeichert. Zusammengefasste Auflagen zählen zusammen; haben Kinder eine
andere Auflage als die der Kachel, steht darunter in Orange _Auflagen in der Klasse_.
**Die abgeleiteten Einträge wandern mit den Kindern** (nach der Versetzung steht die Liste der
07R2 ohne Zutun unter _08R2_), **die handgepflegten bleiben am Klassennamen**; zwischen Büchertausch
und Versetzung stehen beide untereinander. Das Kollegium sieht dieselben Einträge in _Mein Portal →
Klassensätze_.

## Schuljahreswechsel (LMF-Plan)

Menü _System → Schuljahreswechsel_. Abgänger stehen unter _Verwaltung_, der LUSD-Abgleich mit
Versetzung unter _Einstellungen → LUSD & Versetzung_. Es gibt zwei Pläne:

- **Büchertausch vor den Sommerferien:** Alle Klassen geben die alten Schulbücher ab und bekommen
  direkt die neuen. Abschlussklassen und Klassen, die neu gebildet werden (die 6er, die auf H, R, G
  verteilt werden), geben **nur zurück**; der Vorschlag trägt bei ihnen „nur Rückgabe“ ein, der Text
  lässt sich ändern.
- **Bücherausgabe nach den Sommerferien:** nur die **Eingangsjahrgänge** (_Einstellungen → Ausleihe &
  Fristen_, Vorgabe „5, 7“).

**Der Plan ist eine Reihenfolge von Klassen, die das Programm auf Schultage und Stunden verteilt:**
Abschlussklassen zuerst, dann jeder Schultag Stunde 1 bis 6, eine Klasse je Stunde; Wochenenden,
Feiertage und Ferien fallen aus. Wochentag und Datum tippt niemand.

- Oben zwischen den beiden Plänen umschalten (je Art einer pro Schuljahr); rechts _Als PDF_, _Plan
  verwerfen_, _Veröffentlichen_ (nur bei einem Entwurf), _Plan speichern_. Ein Satz darunter sagt, ob
  das Kollegium den Plan schon sieht.
- **Entwurf und Veröffentlichen:** _Plan speichern_ legt einen **Entwurf** an, auf jedem PC gleich,
  aber für das Kollegium unsichtbar und ohne Fristen. _Als PDF_ im Planer enthält den Entwurf, das
  Blatt für die Schulleitung. _Veröffentlichen_ gibt ihn frei: Das Kollegium sieht ihn, beim
  Büchertausch werden die Termine die Fristen der Klassen, und jede weitere Speicherung gilt sofort.
  Hat ein anderer Platz inzwischen gespeichert, lehnt _Plan speichern_ ab, die eigenen Änderungen
  bleiben stehen, und der Hinweis bietet _Neu laden und verwerfen_ an.
- **Das PDF ist immer ein Blatt.** Bei vielen Terminen rückt es enger zusammen und stellt Rückgabe und
  Ausgabe in zwei Spalten nebeneinander; weggelassen wird nie etwas.
- **Zeitraum:** Der **Büchertausch endet am Donnerstag vor den Sommerferien in der 4. Stunde**
  (vorbelegt aus der Ferientabelle), die Reihenfolge läuft rückwärts davor; kommen Klassen dazu,
  beginnt er früher. Die **Bücherausgabe beginnt** am ersten Schultag nach den Ferien (vorbelegt
  2. Stunde). Dazu _Stunden je Tag_ (Vorgabe 6). **Die Sommerferien pflegt die Schule selbst** unter
  _Einstellungen → LUSD & Versetzung → Sommerferien_ (Quelle kmk.org/service/ferienregelung); zwei
  Jahre vor dem letzten bekannten Jahr warnt die Betriebsbereitschaft. **Freie Tage** wie bewegliche
  Ferientage oder pädagogische Tage trägt man über den Chip _Tag freihalten_ mit Datum und Grund ein;
  die Zeile _Übersprungen: …_ nennt jeden ausfallenden Werktag mit Grund.
- **Reihenfolge:** Zeilen mit der Maus am Griff ⠿ ziehen. Im Menü ⋮ der Zeile: _Mit der Zeile davor
  zusammenlegen_ (zwei Klassen in einer Stunde, „10R1/10R2“), _In einzelne Stunden trennen_, _Zeile
  davor einfügen_ (eine Zeile ohne Klasse wie „Bücher setzen“ oder „Nachzügler“, sie braucht einen
  Vermerk), _Klasse aus dem Plan nehmen_, _Zeile entfernen_. Die **Besonderheiten** stehen als Text
  in der Zeile: Klick öffnet, Enter übernimmt, Escape stellt den Text von vorher wieder her. Datum und
  Stunde rechnet das Programm bei jeder Änderung neu; gespeichert wird erst mit _Plan speichern_.
- **Fester Platz** (⋮ → _Datum und Stunde festlegen_): Die Klasse mit dem Ausflug bekommt Datum und
  Stunde von Hand, die übrigen fließen um sie herum; _Festen Platz lösen_ gibt sie zurück. Der
  Vorschlag fürs nächste Jahr bringt feste Plätze nicht mit.
- **Noch nicht im Plan** (über der Tabelle): Klassen ohne Zeile, etwa eine neue nach dem LUSD-Import.
  _06F4 einplanen_ setzt sie hinter die letzte Klasse desselben Jahrgangs und Zweigs; wer sie woanders
  will, zieht den Chip auf eine Zeile. _Andere Klasse eintragen_ nimmt einen Namen auf, den es noch
  nicht gibt („07G1“ vor dem August-Import). Was bewusst draußen bleibt (die Oberstufe organisiert
  sich selbst), steht eingeklappt hinter _11 Klassen bleiben draußen_.
- **Klassen ohne Schüler** tragen das Schild „ohne Schüler“: eine Klasse aus dem Vorjahr oder eine vor
  dem Import getippte. Klassen muss niemand anlegen oder löschen; ohne aktive Schüler verschwindet eine
  Klasse von allein aus jeder Liste.
- **Vorjahr als Vorlage:** Ist der letzte Plan vorbei, beginnt der nächste mit dessen Reihenfolge und
  Auslassungen, neue Klassen hängen hinten an. Ohne Vorjahr: Abschlussklassen zuerst, dann Jahrgang
  absteigend.

Das Kollegium sieht den **veröffentlichten** Plan in _Mein Portal → LMF-Plan_, nur den, der als
nächster dran ist; sonst steht dort „Der nächste Plan folgt“. Die Zeilen von heute sind farbig
hinterlegt. **In die Spalte „Besonderheiten“ gehört kein Schülername:** Das ganze Kollegium liest
sie, und das PDF wird weitergegeben; „Restbücher aus 9R2“ ist richtig, ein Name nicht.

**Fristen:** Der Termin einer Klasse beim Büchertausch ist die Frist ihrer Schulbücher; beim
Veröffentlichen und bei jeder weiteren Speicherung folgen die offenen Ausleihen (die Meldung nennt
die Zahl). Ab dem ersten Tag des Büchertauschs gilt jedes neu ausgegebene Schulbuch dem nächsten
Schuljahr, für jedes Kind. Fällt eine Klasse aus dem Plan oder wird er verworfen, gehen die Fristen
an den allgemeinen Stichtag zurück; Schulbücher aus dem Tausch bleiben beim nächsten Schuljahr, eine
von Hand gesetzte Frist bleibt stehen. Nicht angefasst: gesperrte Schüler, mehrjährige Ausleihen und
Ausgabe-Pläne (§2.3).

## Leserdatei

Die Leserdatei führt **alle, die Bücher bekommen können**, Schüler und Kollegium in einer Liste.

- **Wer steht hier drin?** Schüler kommen aus der LUSD. Das Kollegium entsteht von selbst, sobald
  sich jemand über „Mein Portal“ anmeldet, oder Sie legen es mit _Neuer Leser_ an. Die **Art** steht
  in der Akte: Schüler, Lehrkraft, LiV, Praktikum, Sekretariat, U-plus oder Fachbereich. Sie
  entscheidet keine Rechte; für jede Art außer Schüler gilt dasselbe wie fürs Kollegium. Praktikum
  und Fachbereich haben keinen Zugang zu „Mein Portal“; ein Fachbereich ist ein Sammelkonto der
  Kollegen des Fachs. Eine Ausleihe ans Kollegium ist eine Dauerleihe: nie überfällig, nie gemahnt,
  in der Akte „Dauerleihe“, auf Ausdrucken „ohne Frist“.
- **Reiter**: _Aktive Leser_, _Ehemalige_ und (mit dem Recht zum Löschen) _Papierkorb_. **Suche**
  über den ganzen Bestand (Name, Klasse, Ausweisnummer); Zeile anklicken → Akte.
- **Akte**: _Ausleihen & Vormerkungen_, _Gebühren & Schäden_ und _Stammdaten & Adresse_; links
  stehen immer Konto-Status und _Offene Forderungen_. Für jeden dieselben Felder; im Kollegium sind
  Klasse, Abgangsjahr und LUSD-Kennung verschlossen. Die Art lässt sich innerhalb des Kollegiums
  umstellen (LiV → Lehrkraft), nicht über die Schüler-Grenze. Mit ungespeicherten Eingaben fragt die
  Maske vor dem Schließen (_Bleiben_ / _Verwerfen_).
- **Dokumente** in der Akte: Ausweis, Kontoauszug, Ersatzforderung (nur bei einem Schüler mit
  offenem Schaden), DSGVO-Auskunft. Die Auskunft über einen Kollegen nennt auch sein Zugangskonto,
  seine Wünsche und Reservierungen, seine Klassenleitungen und die Vorgänge, die er bearbeitet hat,
  ohne Angaben zu anderen Personen. Ein frei getippter Grund steht darin, wie er gespeichert ist:
  das Blatt vor dem Aushändigen durchlesen. Die Auskunft über jemanden mit Zugang erstellt nur, wer
  auch _Benutzer & Rechte verwalten_ darf. Alle diese Blätter gehen in einem neuen Tab auf;
  gedruckt und gespeichert wird dort, in den Ordner „Downloads“ kommt nichts von selbst.
- **Ausweis drucken**: Auf der Karte steht _Schülerausweis_, _Lehrerausweis_ (Lehrkraft und LiV)
  oder _Leserausweis_ (die übrigen). Eine Gültigkeit trägt nur der Schülerausweis, aus der Klasse
  gerechnet (Hauptschulzweig bis Ende Klasse 9, sonst bis Ende 10, ab Klasse 11 bis Ende 13; für
  einen Druck in der Akte änderbar). Neue Nummern beginnen mit `A-`, ältere Karten mit `S-` oder `L-`
  bleiben gültig. Eine `A-`-Nummer vergibt das Programm nur einmal. Lässt sich das Ausweis-Design
  nicht laden, druckt das Programm nicht (_Nicht gedruckt, das Ausweis-Design ist nicht geladen_);
  der nächste Klick versucht es erneut.
- **Gebühren & Schäden**: offen / bezahlt / storniert; _Bezahlt_ bucht aus, _Stornieren_ verlangt
  einen Grund. Steht eine offene Forderung auf keinem Brief, gibt es _Bescheid erstellen_; erstellte
  Bescheide stehen mit _Nachdruck_ darunter (§14).
- **Verlust oder Schaden melden** an der Ausleihzeile beendet die Ausleihe und legt die Forderung an;
  das Kind steht danach im Mahnwesen unter _Schadensersatz_ statt in der Mahnliste. Der Brief an die
  Eltern ist ein eigener Schritt.
- **Sperren** verlangt eine Begründung, die danach an der Theke steht; **Sperre aufheben** nimmt jede
  Sperre weg. Beides steht im Protokoll. Einen Kollegen sperrt man nicht.
- **Leser anlegen**: Der Dialog fragt zuerst nach der Art. Klassenweise legt man Schüler über den
  LUSD-Import an. Bei Lehrkraft, LiV, Sekretariat und U-plus ist die **Schul-E-Mail Pflicht**: Mit ihr
  entsteht zugleich der Zugang zu „Mein Portal“, und die Person steht später nicht doppelt da, wenn
  sie sich selbst anmeldet. Freigeschaltet wird nur, wenn Sie auch _Benutzer & Rechte_ dürfen, sonst
  entsteht eine Zugangsanfrage. Eine Rolle vergibt dieses Formular nie.
- **Stapelaktionen**: Leser markieren → Ausweise oder Etiketten drucken; eine ganze Klasse unter
  _Ausweise je Klasse_ im Druck-Center.
- **Ehemalige** nennt je Person die offenen und überfälligen Bücher; nach der Karenzzeit ist der
  Eintrag anonymisiert und verschwindet. Im Mahnwesen steht ein Ehemaliger, sobald eine Frist
  abgelaufen ist; läuft sie noch (Mehrjahresband), in der Akte das Rückgabedatum auf heute setzen.
- **Löschen** verlangt den Namen zur Bestätigung und legt die Person in den Papierkorb; mit einem
  Kollegen fällt sein Zugang zu „Mein Portal“ (dann nur mit _Benutzer & Rechte_, ein Admin-Konto nur
  durch einen Admin). Aus dem Papierkorb lässt sie sich bis zu 180 Tage wiederherstellen, ein
  Ehemaliger kürzer, wenn seine Karenzzeit früher endet. _Endgültig löschen_ wirkt sofort und braucht
  ein eigenes Recht (§8).
- **Doppelter Datensatz?** (Recht „Leser zusammenführen“, unten in _Stammdaten & Adresse_): Steht
  dieselbe Person zweimal da, beide zusammenführen. Es bleibt der Datensatz, dessen Ausweis die
  Person in der Hand hat; Ausleihen, Gebühren und Historie des anderen wandern hinüber. Schüler und
  Kollegen lassen sich nicht miteinander verschmelzen. **Wurde für den aufgelösten Datensatz schon
  ein Ausweis gedruckt, diese Karte einziehen:** Ihre Nummer findet danach niemanden mehr
  ([LUSD.md](LUSD.md) §5).

## Mahnwesen

Reiter **Alle · Akut fällig (bis 14 Tage) · Eskaliert · Schadensersatz**, Filter nach Klasse. Die
ersten drei fragen, wer Bücher zu spät hat, der vierte, wer Geld schuldet. Die Liste zeigt zuerst
die 50 am längsten Überfälligen, _Mehr laden_ die nächsten; das Häkchen über der Liste und
_Liste drucken_ umfassen alle Treffer.

**Mahnbriefe:** Kinder anhaken, dann _Mahnbriefe drucken_: je Kind ein Brief an die Eltern für das
Fensterkuvert mit dem Text der Vorlage (_Einstellungen → Mail_); fehlt die Anschrift, steht das im
Adressfeld, und der Brief geht über das Kind. Der Druck zählt die Mahnung, höchstens einmal am Tag
je Buch; ein zweiter Druck am selben Tag (Papierstau) zählt nicht. Die Liste zeigt je Kind, wie oft
und wann zuletzt gemahnt wurde („2× gemahnt, zuletzt 26.09.2026“); an die Zahl knüpft das Programm
nichts. Auf den Brief kommt nur ein Buch, dessen Frist abgelaufen ist. Drucken und den Mahnlauf
bietet die Seite nur Rollen an, die _Bestellungen verwalten_ dürfen; mit _Leserdatei anzeigen_
lässt sich die Liste ansehen und drucken.

**Sammel-Mahnlauf** per Mail an die Klassenleitungen (Klassen wählen, Empfänger prüfen, senden): je
Kind ein Blatt zum Austeilen; ein zweiter Lauf, auch von einem anderen Platz, wartet, bis der erste
fertig ist. **Liste drucken** druckt die Mahnliste so, wie Reiter, Filter und Suche sie zeigen. Mail
und Liste zählen nicht als Mahnung. Lehrkräfte werden nie angemahnt. Ehemalige stehen unter
„Ehemalige“ statt unter ihrer früheren Klasse und bekommen den Mahnbrief über die Auswahl, an eine
Klassenleitung geht nichts. Welche Klasse an welche Lehrkraft geht, wird unter _Einstellungen →
Klassenleitungen_ von Hand gepflegt; die Versetzung rückt die Zuordnungen eine Stufe hoch, außer von
6 nach 7 und von 10 in die Oberstufe, wo die Klassen neu gebildet werden. Die Oberstufe hat keine
Klassenleitung (§3).

**Schadensersatz:** Sobald für ein Kind ein Verlust oder Schaden gemeldet ist, steht es im vierten
Reiter, mit dem nächsten Schritt je Zeile: _Bescheid erstellen_ → _Frist läuft_ (abwarten) → _Frist
abgelaufen_ (_Übergeben_: Original und Buchungsbeleg an die Aufsicht) → _übergeben_; bei _Rückgabe
nach Übergabe_ die Aufsicht informieren. Erledigte Briefe stehen nur noch in der Akte.

**Schadensersatz-Bescheid:** in der Mahnliste genau einen Schüler wählen und _Schadensersatz-
Bescheid_, oder _Bescheid erstellen_ im Reiter _Schadensersatz_ bzw. in der Akte. Der Dialog zeigt
die überfälligen Bücher und offenen Forderungen mit einem Betragsvorschlag samt Herleitung; Beträge
sind änderbar, Zeilen abwählbar. Eine Verlustmeldung vorher ist nicht nötig: _Bescheid erstellen_
bucht die gewählten Bücher als Verlust, vergibt die Referenznummer (nie zweimal), setzt die Frist
und öffnet den Brief, alles oder nichts. _Nachdruck_ ist immer derselbe Brief. Fehlen Angaben unter
_Einstellungen → Schadensersatz_, sagt der Dialog welche.

**Kommt ein abgerechnetes Buch doch zurück**, storniert die Theke beim Scan die Forderung („Buch
reaktiviert. Die Forderung über … € wurde storniert …“), ebenso _Gefunden_ im Fehlbestandsbericht
der Inventur. Liegt der Bescheid schon bei der Schulaufsicht (_Übergeben_), bleibt die Forderung
offen, und der Hinweis sagt, die Aufsicht unverzüglich zu informieren.

## Abgänger

Abgänger sind die Abschlussklassen des laufenden Schuljahres (9H und das freiwillige 10H, 10R, 13),
die zum Schuljahresende gehen; sie leihen bis zuletzt aus. Was sie außer Schulbüchern leihen
oder verlängern, ist spätestens am letzten Schultag vor den Sommerferien fällig; der Ferien-Leseclub
gilt für sie nicht. Die Liste zeigt **vom 1. Mai bis 31. Juli**
die Abgänger, die **noch Bücher haben**, zum Einsammeln vor der Entlassung: Kontoauszüge drucken oder
an die Klassenleitungen mailen (Mailen nur mit _Bestellungen verwalten_, wie beim Mahnwesen); wer alles
zurückgegeben hat, verschwindet. Gedruckt wird, was die
Liste zeigt (gewählte Klasse, Suche); so lässt sich ein einzelner Kontoauszug nachdrucken. Eine
Klasse, die mit keinem Jahrgang von 1 bis 13 beginnt (14A, eine Buchnummer im Feld Klasse), versetzt
der Schuljahreswechsel nicht; seine Vorschau nennt sie.

Wer die Schule **verlassen** hat, steht in der Leserdatei unter _Ehemalige_, mit offenen Büchern
auch im Mahnwesen. Er bleibt bis zum Ende der **Karenzzeit** (Vorgabe 90 Tage, ab Abgang oder letzter
Rückgabe, was später ist) gesperrt erhalten, damit sich eine falsche Zuordnung reparieren lässt, und
wird dann anonymisiert. Bleibt etwas dauerhaft offen, meldet die Betriebsbereitschaft nach einem
Jahr „Ehemalige mit offenen Vorgängen“: Buch als Verlust melden, Forderung bezahlt oder storniert
buchen, danach löscht das System von selbst (§8, [LUSD.md](LUSD.md) §4).

## Bestellwesen

Fünf Reiter (§7):
- **Bestellungen**: links die Bestellung, darunter der Bestellbedarf, rechts die Zusammenfassung mit
  Lieferant, Summen und _Bestellung auslösen_. Der Bestellbedarf nennt die Schulbücher, deren Bestand
  nicht für die Kinder im nächsten Schuljahr reicht; _N hinzufügen_ legt die fehlende Menge in die
  Bestellung, dort trägt der Titel die Marke _Bestellbedarf N_. Der Warenkorb hat bis zu drei Abschnitte,
  _Lernmittelfreiheit · Land_, _Schülerbücherei · Schulträger_ und _Spende · ohne Topf_ (etwa Geld des
  Fördervereins); je Abschnitt geht eine eigene Bestellung an den Händler, und das Feld _Topf_ an jeder
  Position schiebt sie in einen anderen. Eine Spende läuft beim Händler wie die Schülerbücherei, mit deren Kundennummer
  und der Rechnung an die Schule; Anschreiben und Mail sagen „Spende". Den Topf einer Bestellung
  korrigiert man später im Bestelldetail, mit Grund.
- **Wareneingang**: Positionen einbuchen → Etiketten. Der Kopf jeder Bestellung nennt ihren Stand
  (bestätigt, wartet auf den Händler, Mail nicht versendet) und wechselt von selbst.
- **Bestellhistorie** mit Filter _Mittelherkunft_ und Kennzahlen je Topf.
- **Klassensatz-Reservierungen** (Warteschlange aus dem Kollegium; _Abschließen_ schickt die Bereit-Mail)
  und **Meldungen** aus dem Portal (_Abhaken_ mit Notiz an die Lehrkraft).

**Wie der Bestellbedarf rechnet:** _Bestand / Bedarf_ je Zeile. Bestand sind alle nicht
ausgesonderten Exemplare, auch verliehene und bestellte. Bedarf sind die Kinder, die das Buch im
nächsten Schuljahr brauchen, gerechnet aus den Ausleihen dieses Schuljahres und den Kindern, die in
den Jahrgang nachrücken; hat nur ein Zweig oder Wahlfach das Buch, zählt derselbe Anteil. Eine
Reserve gibt es nicht. Über der Liste steht, mit den Schulbüchern von wie vielen Kindern gerechnet
wurde: Sind es deutlich weniger als alle, wurde nicht über das Programm ausgegeben, und der Bedarf
ist zu niedrig. Ein neues Lehrwerk erscheint erst nach seiner ersten Ausgabe. Zusammengefasste
Auflagen stehen einmal da, mit der neuesten Auflage für den Warenkorb und der Aufschlüsselung
darunter; das Filterfeld findet auch die ISBN einer älteren Auflage.

**Neue Auflage bestellen** (Menü ⋮ der Zeile): ISBN der neuen Auflage eingeben oder scannen, _Suchen_;
_Zuordnen und bestellen_ fasst sie mit der Zeile zusammen, führt sie als Lernmittel und legt sie in
den Warenkorb.

**Titel suchen & hinzufügen** fragt den eigenen Bestand und die DNB (antwortet sie nicht, sagt die
Trefferliste das); vor dem Warenkorb fragt ein Fenster nach Signatur, Schlagworten, „Lernmittel?“
und Menge. Bei einem neuen Titel aus der DNB stehen _Vorschläge aus der DNB_ (Wörter, die es in der
Liste schon gibt) und _Neue Schlagworte aus der DNB_ (aus der Normdatei) zum Anklicken. Bei einem
Titel aus dem eigenen Bestand ohne Schlagworte kommen sie von selbst; trägt er schon welche, holt
sie _Vorschläge aus der DNB holen_.

**Kommt die Bestellmail nicht an,** ist die Bestellung trotzdem gespeichert: In der Bestellhistorie
steht „Mail nicht versendet“, in der Bestellung _Erneut senden_. Hat der Händler auf anderem Weg
bestellt, nimmt _Auf anderem Weg bestellt_ den Hinweis nach einer Rückfrage weg, ohne zu senden
(nur ohne Bestätigungs-Link; mit Link wird die Zusage nachgetragen).

**Preis laut Rechnung:** Rechnet der Händler anders ab, steht in der Bestellung unter jedem Preis
_Preis ändern_, jederzeit; der neue Preis gilt für die Exemplare aus der Bestellung und die Summe und
steht im Protokoll.

**Zehn- oder dreizehnstellige ISBN** sind dieselbe Nummer: Das Programm speichert eine zehnstellige
mit richtiger Prüfziffer dreizehnstellig und findet den Titel über beide Längen. Eine mit falscher
Prüfziffer bleibt, wie sie ist, denn umgerechnet führte sie auf ein anderes Buch.

## Inventur

_Inventur starten_ → Umfang wählen (komplett, eine Signatur, Fach/Klasse) → scannen. Gezählt wird, was
der Abschluss erwartet; was während der Inventur an der Theke war oder neu dazukam, erwartet er nicht,
ebenso wenig ein Buch im Abholfach.
Ein verliehenes Buch im Regal bucht der Scan zurück (ist es vorgemerkt: „ins Abholfach legen“).
**Achtung:** _Inventur abschließen_ bucht alles Ungescannte im Umfang als Verlust (die Rückfrage nennt
die Zahl); diese Bücher stehen dann im Fehlbestandsbericht. Ein Buch, das sich findet, holt ein Kreuz
unter _Gefunden_ oder ein Scan an der Theke zurück. _Liste drucken_ im Bericht druckt die fehlenden
Exemplare nach Signatur mit Kästchen zum Abhaken. Verwerfen bucht nichts, der Bereich gilt dann nicht
als geprüft. Eine frühere Inventur lässt sich löschen (⋮ → _Löschen_); die Bücher darin bleiben als
verloren gebucht (§6).

---

## Statistiken

Bestand, aktuell verliehen, Zirkulationsquote, Wiederbeschaffungswert; Ausleihen pro Monat;
überfällige Bücher nach Dauer (Geräte stehen im Mahnwesen); **Renner** (meistausgeliehen) und **Ladenhüter** (seit über zwei Jahren
nicht ausgeliehen, mit Exemplar im Bestand und ohne offene Ausleihe) mit Detailseite und Filter. Ohne Schülernamen — die Statistik zählt
Ausleihen, nicht Personen. Ein bestelltes Buch zählt erst zum Bestand, wenn es eingetroffen
ist. (§11)

## Bestandsbücher

Zwei Reiter, je mit einem Blatt zum Abheften: **Zugangsbuch** (was in einem Zeitraum in den Bestand
kam: Eingangsdatum, Titel, Anzahl, Barcodes, Lieferant) und **Abgangsbuch** (was hinausging: Datum,
Titel, Signatur, Grund, Anzahl, Barcodes). Vorbelegt ist beim Zugangsbuch das vorige Schulhalbjahr,
beim Abgangsbuch das laufende (Stichtage 15.3. und 15.9.); ein Zeitraum umfasst höchstens ein Jahr.
Land, Schulträger und Spende stehen getrennt. Am Bildschirm zeigt jeder Abschnitt zuerst 50 Zeilen, _Mehr
laden_ die nächsten; Stückzahl und Ausdruck umfassen immer alle.

Ein bestelltes Buch erscheint im Zugangsbuch erst nach dem Einbuchen im Wareneingang, mit dem Tag der
Lieferung. Gelöschte Bücher bleiben in beiden Büchern, ein abgehefteter Ausdruck stimmt also auch
danach. Was nicht auf dem Blatt steht, nennt es als Zahl darunter. Bücher ohne Bestellung und ohne
Eigentumsangabe stehen im Zugangsbuch unter „ohne Zuordnung“, weil nicht belegt ist, aus welchem Geld
sie bezahlt wurden (§13).

## Bestellberichte

Monatsbericht, Jahresbericht und Lieferantenabrechnung als PDF; in Blöcken je Topf mit
eigener Summe und der Gesamtsumme darunter, über _Mittelherkunft_ auch für nur einen Topf.
Ohne Preise im Bestellwesen zählen die Blätter Exemplare statt Euro und heißen
entsprechend. (§7)

## System-Logs

_Allgemeines Logbuch_ (jede Buchung) und _Verwaltungsprotokoll_ (wer hat wann was geändert).
Aufbewahrung 24 Monate, einstellbar. Dritter Reiter **Theken-Auskunft** (eigenes Recht
`audit_details`, ab Werk Admin und Leitung): Ein Buch liegt auf der Theke, sein Exemplar ist
längst gelöscht — die Barcode-Suche zeigt, was das Protokoll dazu noch weiß (Titel,
letzte Ausleihen), auch wenn der Titel komplett gelöscht wurde. Jede Abfrage wird
selbst protokolliert; nach DSGVO-Tilgung zeigt auch dieser Weg nichts mehr. (§10)

## Benutzer & Rechte

Reiter **Benutzer** (anlegen, bearbeiten, deaktivieren; **Zugangsanfragen** aus der Selbstanmeldung
freischalten) und **Rollen & Rechte** (Matrix; Änderungen wirken sofort).

**Anmeldung über** legt je Konto fest, wer das Passwort prüft: der _Mailserver der Schule (IMAP)_
oder der _Verzeichnisdienst (LDAP)_ mit der **LDAP-Kennung** (etwa dem Anmeldenamen im Schulnetz).
Die Kennung eines Admin-Kontos ändert nur ein Admin, das eigene Anmeldeverfahren nur eine andere
Person mit Administratorrechten; das eigene Konto lässt sich nicht löschen.

Die **Ausweisnummer** eines freigeschalteten Zugangs vergibt das Programm. Ein geleertes Feld bekommt
die nächste freie Nummer; die alte Karte gilt dann nicht mehr.

**Löschen entfernt den Zugang, nicht die Person:** Wer einen Ausweis trägt, steht weiter in der
Leserdatei, und die Karte gilt weiter. Verlässt jemand die Schule, löschen Sie die Person in der
_Leserdatei_; der Zugang geht dabei mit.

**Vier Rollen: Admin, Leitung, Mitarbeit, Helfer.** Die Leitung darf alles außer Benutzer & Rechte
und den Einstellungen, die „Einstellungen verwalten“ verlangen; unter _Einstellungen_ bedient sie LUSD
& Versetzung, Datenverwaltung, LMF-Aktionen, Lieferanten und Schlagworte. Der Grund: Wer Konten
ändert, ändert E-Mail-Adressen, und die Adresse ist die Anmeldung. **Kollegium steht nicht in der
Matrix:** Es ist der Ausgangszustand jeder Lehrkraft, die sich über „Mein Portal“ selbst anmeldet und
freigeschaltet wird; wer mehr können soll, bekommt eine der vier Rollen (§12).

## Einstellungen

17 Kategorien, jede einzeln speicherbar (§17):

| Kategorie               | Wofür                                                                                                                                                                                                                                                                                                                                                                                         |
| ----------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Schule                  | Name, Anschrift, Eigentumsvermerk auf Etiketten                                                                                                                                                                                                                                                                                                                                               |
| Ausleihe & Fristen      | Tage je Buch/Medium, Limit je Schüler, Eingangsjahrgänge, Ferien-Leseclub (festes Rückgabedatum über die Ferien)                                                                                                                                                                                                                                                                               |
| Mahnwesen               | automatische Sperre: ab wie vielen überfälligen Medien, nach wie vielen Tagen                                                                                                                                                                                                                                                                                                                 |
| Klassenleitungen        | Klasse → Klassenleitung (Empfänger für Mahnlauf und Abgänger-Kontoauszüge)                                                                                                                                                                                                                                                                                                                    |
| Bestellwesen            | Bedarfswarnung, Preise erfassen, Gültigkeit des Bestätigungs-Links                                                                                                                                                                                                                                                                                                                            |
| Lieferanten             | Händler, Kundennummern, genau ein Hauptlieferant                                                                                                                                                                                                                                                                                                                                              |
| Schlagworte             | Wörter umbenennen, zusammenführen und löschen (auch mehrere: vorn auswählen, unten „Löschen“), Verweise für andere Schreibweisen („Tierfantasy“ → „Fantasy“), Filter-Markierung fürs Portal; die Suche zeigt höchstens 200 Treffer und nennt, wie viele es sind |
| Schadensersatz          | Angaben für den Schadensersatz-Bescheid: Schulamtsbereich und Schulnummer (Teil der Referenznummer), Aufsicht, Schulleitung, Geschäftszeichen, Zahlstelle und Bankverbindung, Zahlungsfrist (Vorgabe 28 Tage); **„Immer mit dem Einkaufspreis rechnen"** — aus gilt der heutige Listenpreis (so verlangt es die Arbeitshilfe), an immer der Preis, den die Schule damals bezahlt hat                                                                                                                                                                                 |
| Datenschutz & Sitzung   | Löschfristen, Abgänger-Karenzzeit, Theke leeren, Sperrbildschirm                                                                                                                                                                                                                                                                                                                              |
| Erreichbarkeit & Alarme | öffentliche Adresse (Basis für Bestätigungs-Link, Katalog, Monitor), Alarm-Empfänger, Schalter „Auch Warnungen mailen“                                                                                                                                                                                                                                                                        |
| Mail                    | Postausgang mit Verbindungstest; zweiter Absender mit Häkchen, welche Mails über ihn gehen; Adressen aus dem Webmail (Konten aus Littera bekommen am Ersten jedes Monats ihre Adresse der Schule, „Probelauf“ zeigt es vorher); Vorlagen für den Elternbrief zur Mahnung und die Bestellmail |
| LMF-Aktionen            | alle Lernmittel einer Klasse auf ein neues Datum verlängern                                                                                                                                                                                                                                                                                                                                   |
| Datenverwaltung         | Katalog-Import (Littera), „Bestand mit vorhandenen Nummern“ (CSV, übernimmt die Nummern; ein Buch mit LMF-Nummer bekommt eine kurze, das alte Etikett scannt weiter), Listenimport (ISBN + Stückzahl → Titel samt Exemplaren mit B-Nummern, höchstens 5.000 je Titel und 200.000 je Liste; ein zweiter Klick mit derselben Datei legt nichts doppelt an), „Fehlende Cover laden“, Bestandsliste als CSV (eine Zeile je Exemplar), Offline-Sicherungen einspielen |
| LUSD & Versetzung       | LUSD-Abgleich, Versetzung zum Schuljahresende, Sommerferien für den LMF-Plan                                                                                                                                                                                                                                                                                                                  |
| Sicherung               | Schalter „Jede Nacht selbst sichern“, Vorgabe aus, weil der Server der Schule selbst gesichert wird; an: verschlüsselte Nachtsicherung mit wöchentlicher Probe, aus: ältere Stände werden nach der Aufbewahrung gelöscht                                                                                                                                                                      |
| Betriebsbereitschaft    | Selbstprüfung: eingerichtet, aber nicht in Betrieb? (§15)                                                                                                                                                                                                                                                                                                                                     |
| Fehler im Log           | Fehlermeldungen des Servers aus den letzten 24 Stunden mit Anzahl, letzter Uhrzeit und Ursache, auch ein Absturz (§15) |

Mit ungespeicherten Eingaben fragt die Seite vor dem Wechsel der Kategorie oder des Menüpunkts
(_Bleiben_ / _Verwerfen und weiter_).

## Mein Portal (Kollegium)

Lehrkräfte sehen genau diesen Bereich, mit vier Reitern (§12, Rolle Kollegium):

- **Reservieren & Melden**: Die Suche findet Bücher der Bücherei und Schulbücher, auch bestellte, noch
  nicht eingetroffene („N bestellt“). Am Treffer stehen „Klassensatz reservieren“ (Klasse oder Kurs als
  Freitext, Anzahl, Datum) und „Problem melden“. Findet die Suche nichts, steht „Problem melden“ unter
  „Keine Titel gefunden“, das Gesuchte ist im Formular schon eingetragen. Solange nichts gesucht wird,
  stehen unter dem Suchfeld die eigenen Reservierungen und Meldungen mit Stand und Antwort der
  Bibliothek. Die Schlagworte, die die Bibliothek als Filter markiert hat, stehen als Chips unter dem Suchfeld.
  Höchstens 50 Treffer, die Gesamtzahl steht darüber.
- **Klassensätze** der eigenen Klassen.
- **LMF-Plan**: Rückgabe- und Ausgabetermine je Klasse, für alle gleich, als PDF.
- **Schulbücher** (für die Fachsprecher): je Fach eine Zeile mit Exemplaren, Titeln und Verliehenen,
  aufklappbar zu den Covern; Suche und Filter Jahrgang und Schulzweig. Ein Klick auf ein Cover öffnet
  rechts ein Blatt mit Bestand und ISBN. „Als PDF“ druckt ein Fach mit der aktuellen Filterung; der
  Kopf nennt den Ausschnitt. Gezählt wird nur, was als Lernmittel geführt ist; zusammengefasste
  Auflagen sind eine Zeile. Den **Schulzweig** pflegt die Bibliothek am Buch (leer heißt „gilt für alle
  Zweige“, solche Bücher erscheinen unter jedem Zweig-Filter); Littera hat ihn nicht mitgeliefert.
  Ein Coverbild von außerhalb erscheint im PDF nicht, gedruckt wird nur, was auf dem Server liegt.

---

## Darstellung

Unten in der Seitenleiste schaltet **Dunkles Design** das Programm an diesem Rechner dunkel und
wieder hell. Solange niemand schaltet, folgt es dem Rechner (so, wie Windows oder macOS
eingestellt ist). Die Wahl gilt in diesem Browser an diesem Rechner, nicht an den anderen
Arbeitsplätzen. Ausdrucke, die Etiketten-Vorschau und die Ausweiskarte im Designer bleiben hell,
weil sie Papier zeigen.

## Bedienung ohne Maus

Die Hauptbildschirme lassen sich mit der Tastatur bedienen — geprüft an den Stellen, die
das Browser-Gate misst (§19): Das erste **Tab** auf jeder Seite landet auf „Zum Inhalt
springen" (überspringt die Seitenleiste), **Tab** wandert durch Felder und Knöpfe,
**Enter/Leertaste** löst aus, **Escape** schließt den obersten Dialog und gibt den Fokus an
die Stelle zurück, von der er kam. In Listen öffnet der **Name** (Schüler) beziehungsweise
der **Titel** (Buch, Signaturen-Regal) die Akte. In Listen mit Kästchen wählt **Strg+A**
(Mac: **Cmd+A**) alle Zeilen und **Umschalt+Klick** alle Zeilen bis zum zuletzt geklickten
Kästchen; in einem Textfeld markiert Strg+A weiter den Text.
Wer im Betriebssystem „Bewegung reduzieren" eingestellt hat, sieht keine Ein- und
Ausblendungen. Meldungen am unteren Rand bleiben stehen, solange die Maus oder der Fokus
darauf liegt. An der Theke hält das Scanfeld den Fokus — das ist gewollt, der Scanner tippt
blind. Steht der Sperr-Dialog („Ausleihe blockiert"), liegt der Fokus auf „Abbrechen": Ein
weiterer Scan bricht dort ab, statt eine Sperre aufzuheben oder zu übergehen.

Was nicht ohne Maus geht: die Zeichenfläche des Ausweis-Designers (§19, Bekannte Lücken) und
die Reihenfolge im LMF-Planer, die nur das Ziehen am Griff ändert. Ein
Durchgang mit einem Screenreader hat nicht stattgefunden.

## Wenn etwas nicht geht

- **Bestellung geht ohne Link raus / Katalog-Adresse fehlt** → _Erreichbarkeit & Alarme_:
  öffentliche Adresse eintragen.
- **Mahnliste kommt bei niemandem an** → _Klassenleitungen_: Klasse hat keine Lehrkraft.
- **Rote Meldung „Kein Backup"** → _Betriebsbereitschaft_ öffnen; dort steht je Punkt, was
  fehlt und wie es zu beheben ist.
- **Hinweis „Eine neue Fassung des Programms ist eingespielt"** → Erst abschließen, was offen ist
  (eine Ausleihe, eine Maske mit Eingaben), dann einmal _Neu laden_ klicken. Ein Fenster, an dem
  noch niemand gearbeitet hat, lädt sich selbst neu.
- **„Bitte erlaube Popups, um das PDF zu öffnen."** → Der Browser hat den neuen Tab blockiert;
  gedruckt und gezählt ist nichts. In der Adresszeile für diese Seite Pop-ups erlauben und noch
  einmal klicken.
- **Scanner tippt ins Leere** → einmal ins Scanfeld klicken; die Theke holt den Fokus nach
  jedem Scan selbst zurück.
- **Scanner tippt ß statt Bindestrich** (im Editor erscheint „Sß10001" statt „S-10001") → Der
  Scanner steht auf amerikanischer Tastatur. Das Programm liest die Nummer trotzdem richtig, andere
  Programme am Rechner nicht. Stell den Scanner um: In seiner Kurzanleitung (Datalogic QuickScan QM21X1, „Quick
  Reference Guide", Seiten 17 und 18) nacheinander _ENTER/EXIT PROGRAMMING MODE_, _Country Mode =
  Germany_ und noch einmal _ENTER/EXIT PROGRAMMING MODE_ scannen. Geklappt hat es, wenn er einen
  hohen und vier tiefe Töne gibt und neu startet. Die Einstellung liegt im Scanner, nicht am
  Rechner.
- **Netz weg** → weiterarbeiten. Oben erscheint ein schmales Band, die Theke bleibt bedienbar und
  sperrt sich nicht. Wie sonst scannen, erst den Ausweis, dann die Bücher: Beides wird auf diesem
  Rechner gemerkt und gebucht, sobald die Verbindung zurück ist. Der Ausweis zeigt ohne Netz keinen
  Namen, nur die Nummer.
  - Meldet die Theke **„nicht eindeutig"**, kennt sie die Nummer nicht (ein sehr neues Buch, ein
    fremder Aufkleber). Dieser Scan ist NICHT gebucht — notiere ihn auf Papier.
  - **Namen suchen** geht ohne Netz nicht: Auf dem Theken-Rechner stehen keine Personendaten.
  - **Geräte** lassen sich ohne Netz nicht ausgeben oder zurücknehmen.
  - Muss der Rechner aus, bevor die Verbindung zurück ist: **„Sicherung speichern"** im Band, die
    Datei später unter _Datenverwaltung → Offline-Sicherungen einspielen_ übernehmen. Zweimal
    einspielen schadet nicht.
- **Nach einem Netzausfall: „x Buchungen aus dem Nachbuchen brauchen einen Blick"** → Der Server hat beim Nachbuchen
  etwas anders gebucht, als es gescannt wurde — das Buch lag bei jemand anderem, oder eine
  Rückgabe kam zu spät. Öffne **Meldungen**, sieh die Zeilen durch und setze jede mit „Erledigt"
  ab, um die du dich gekümmert hast. Steht der Knopf nicht da, fehlt das Recht, Schülerdaten zu
  sehen — dann bitte die Bibliothek ansprechen.
