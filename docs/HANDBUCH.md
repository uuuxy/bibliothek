# Benutzerhandbuch

Stand: 2026-09-22. Für Bibliothekspersonal, Sekretariat und Schulleitung — geschrieben aus
Sicht der Arbeit am Tresen, nicht aus Sicht des Codes. Die fachlichen Regeln dahinter stehen
im [Fachkonzept](FACHKONZEPT.md); dort verweisen die §-Angaben hin.

**Zwei Grundsätze, die überall gelten:**

- Es gibt **keine Passwörter** zu verwalten. Angemeldet wird mit dem Schul-Postfach
  (E-Mail + Mail-Passwort). Wer kein Konto hat, kann sich damit selbst anmelden und wird von
  der Bibliothek freigeschaltet (→ Benutzer & Rechte).
- **Jede Rolle sieht nur ihren Teil.** Lehrkräfte sehen nur _Mein Portal_, Helfer nur die
  Theke ohne Schülerakten, Mitarbeitende den Tresenbetrieb, Admins alles.

---

## Öffentliche Seiten — ohne Anmeldung

| Adresse                    | Für wen                                                     | Zeigt                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| -------------------------- | ----------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `https://<schule>/katalog` | Schüler, Eltern, Kollegium — vom Handy, aus dem Klassenraum | Suche nach Titel, Autor, ISBN; Cover; „N von M verfügbar". Keine Ausleihdaten, keine Namen.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `https://<schule>/monitor` | der Bildschirm vor der Bibliothek                           | Endlos-Slideshow: Buch des Monats (die meisten Schüler-Leser in 30 Tagen), Neu eingetroffen, Beliebt diese Woche (7 Tage). Gezählt werden Leser, nicht Exemplare — Klassensätze an Lehrkräfte zählen nicht. Keine Schulbücher (Lernmittel), nur Titel mit einem Exemplar im Haus — dieselbe Regel wie im Katalog. Buch des Monats und Neu eingetroffen nur mit Cover. Aktualisiert sich alle 5 Minuten von selbst; ist der Server beim Einschalten noch nicht da, versucht die Seite es alle 30 s erneut. Folien ohne Inhalt (Ferien) werden übersprungen. Um 03:00 Uhr lädt sich die Seite selbst neu und holt so neue Versionen — der Bildschirm braucht keine Tastatur. |

Beide Seiten haben keinen Menüpunkt. Die fertigen Adressen stehen unter
_Einstellungen → Erreichbarkeit & Alarme_ zum Kopieren. (§16)

---

## Suchen

Jede Seite hat **eine** Suchleiste oben über der vollen Breite, Filter und Knöpfe stehen in
der Zeile darunter. Sie sucht, worum es auf der Seite geht: der Medienkatalog Bücher,
Leserdatei und Mahnwesen Menschen, die Inventur scannt. Eine seitenübergreifende
Suchleiste, die von jeder Verwaltungsseite aus zu Buch- oder Schülerakte springt, gibt es
nicht mehr (zurückgebaut am 04.09.2026). Gebucht wird nur an der Theke.

## Ausleihe (Theke)

Der Startbildschirm nach der Anmeldung. **Ein Feld für alles:**

1. **Schülerausweis scannen** → die Theke öffnet sich: Foto, Klasse, Konto-Status, entliehene
   Bücher, Gebühren, Vormerkungen.
2. **Buch scannen** → ausgeliehen. Frist wird automatisch berechnet (Lernmittel bis zum
   Stichtag 31.07., andere Medien nach Tagen oder, bei eingeschaltetem Ferien-Leseclub, bis
   zu dessen festem Rückgabedatum). (§2)
3. **Dasselbe Buch erneut scannen** → zurückgegeben. Ein Buch ohne geöffnete Theke scannen →
   wird sofort zurückgebucht, auch wenn es auf jemand anderen verbucht war
   („Fremdrückgabe", mit Hinweis).
4. **Namen tippen** statt scannen: Vorschläge erscheinen beim Tippen, Klick öffnet die Theke.

**An jeder Buchzeile:** Verlängern · Zurückgeben · Verlust/Schaden melden (Grund, Ersatzbetrag —
die Forderung landet in der Akte, der Elternbrief kommt als PDF).

**Die Theke warnt von selbst:**

- Kommt ein **vorgemerktes** Buch zurück, erscheint ein roter Hinweis: nicht ins Regal, die
  nächste Leserin wartet. (§4)
- Ein **gesperrter** Ausweis wird angehalten — mit dem hinterlegten Grund. Ausleihen ist nur
  bewusst per Override möglich; die Sperre aufheben nur mit dem Recht dazu. (§2.2)
- Eine **offene Forderung** oder zu viele überfällige Bücher halten nur die Schülerbücherei an.
  Ein Schulbuch geht immer raus — die Lernmittelfreiheit lässt keine Sperre zu, und darum
  fragt die Theke dabei auch nicht nach einem Override.
- **Überfällige** Bücher und die Mahnstufe stehen direkt an der Ausleihzeile.

**Außerdem:** Geräte (iPads, Taschenrechner, Beamer) laufen über dieselbe Theke, mit
Zubehör-Checkliste beim Scan (§5) · Kamera als Ersatz für den Handscanner (Knopf neben dem
Feld) · Passbild per Webcam · Ausweis drucken, Kontoauszug, DSGVO-Auskunft als PDF (§18).

Nach 5 Minuten ohne Eingabe schließt sich die Akte, nach 15 Minuten der Sperrbildschirm —
beides einstellbar (_Datenschutz & Sitzung_).

---

## Medienkatalog

- **Suche & Filter**: ein Feld für Titel, Autor, Fach, Klasse, Signatur und Schlagwort;
  Kartenansicht mit Cover und Signatur. Ein Verweis findet sein Schlagwort: „Tierfantasy"
  findet die Bücher mit „Fantasy", wenn es unter _Einstellungen → Schlagworte_ so
  eingetragen ist. Dasselbe gilt für die Suche der Titel-Verwaltung, den öffentlichen
  Katalog und _Mein Portal_.
- **Buchakte** (Klick auf eine Karte): Exemplare mit Status, aktuelle Ausleiher, Vormerkungen
  (Warteliste mit Schüler-Suche), Historie. An jedem Exemplar steht, **was es heute noch
  wert ist**, samt Herleitung („3. Verleihjahr → 60 % von 24,90 €"). Ist ein Buch
  beschädigt, tragen Sie im Zustands-Dialog einen Prozentwert ein (z. B. 20 % für einen
  Wasserschaden): Er mindert jeden künftigen Ersatzbetrag, zieht das Buch aber nicht aus
  dem Verkehr. Erfasst wird das nach der Rückgabe hier, nicht an der Theke. (§14)
- **Titel-Verwaltung**: neuen Titel anlegen (ISBN-Eingabe holt Metadaten und schlägt eine
  Signatur vor — vor dem Speichern prüfen), bearbeiten, Cover tauschen, Meldebestand,
  Exemplare aussondern (Verlust, Schaden, Bestandskorrektur). Wird der Bestand nach oben
  korrigiert, legt das System die fehlenden Exemplare mit regulärer B-Nummer an (seit
  07.09.2026 — vorher „SYS-…"); sie stehen danach im Druck-Center unter _Fehlende
  Etiketten_. (§13)
- **Mehrjahresband**: Bei einem Lernmittel heißt die Spanne „Im Unterricht von Jahrgang …
  bis". Bleibt das Buch über diese Spanne beim Kind, statt am Rückgabetermin der Klasse
  zurückzukommen, schaltet man darunter „Mehrjahresband" ein; der Hinweis daneben rechnet
  mit: „Bleibt beim Kind bis zum Ende von Jahrgang 9." Die Frist ist dann der Stichtag des
  Schuljahres, in dem das Kind diesen Jahrgang beendet — ein Kind der 7 gibt ein Buch „7 bis
  9" nach drei Schuljahren zurück. Der Schalter geht nur mit einer Spanne über mehr als
  einen Jahrgang; und Vorsicht mit der Vorgabe 5 bis 10, die jeder neue Titel trägt. (§2)
- **Titel ohne Exemplar** stehen in keinem Katalog und in keiner Trefferliste — weder im
  Portal noch an der Theke (seit 22.09.2026; ein bestelltes Exemplar zählt schon). Sie sind
  aber nicht weg: In der Titel-Verwaltung schaltet der Umschalter _Mit Exemplaren | Ohne
  Exemplare_ auf die Aufräumsicht. Dort bekommt so ein Titel Exemplare, oder er wird
  gelöscht. Auf der Bestellliste steht er ohnehin, sein Bestand liegt unter jeder Schwelle.
- **Geräte**: anlegen mit Modell, Seriennummer, Barcode `G-…` und Zubehör-Checkliste.

## Signaturen

Sachgruppen (Kürzel + Bezeichnung, z. B. _Jug_ – Jugendliteratur) pflegen und Regale per
Präfixsuche durchsehen („Jug" findet „Jug Her", „Jug Pre" …). (§13)

## Druck-Center

Vier Reiter: **Buch-Etiketten** (Format, Startposition auf dem Bogen, Titel aus dem Katalog
oder aus einer Klasse) · **Fehlende Etiketten** (alle Exemplare, die noch keins haben — die
Zahl steht als Badge in der Navigation) · **Schülerausweise** (Designer für Vorder- und
Rückseite) · **Klassenweise drucken** (Ausweise für eine ganze Klasse). Der Druck läuft
über den Browser-Druckdialog. (§9)

## Klassensätze

Welche Klasse hat welche Lektüre? Klasse suchen, _Klasse hinzufügen_ öffnet den Dialog:
Bücher auswählen, Zielklasse eintragen, speichern. Reservierungen aus dem Kollegium stehen
im Bestellwesen (→ _Klassensatz-Reservierungen_). (§4)

Seit 05.09.2026 ergänzt sich die Liste von selbst: Hat mehr als die Hälfte einer Klasse
(mindestens fünf Kinder) denselben Titel ausgeliehen, erscheint er bei der Klasse mit dem
Abzeichen _aus Ausleihen · n Leser_ — Schulbuch wie Lektüre, live aus den Ausleihen
gerechnet, nirgends gespeichert. Die von Hand gepflegte Liste bleibt unverändert; beim
_Bücher verwalten_ sind nur die handgepflegten Titel vorgewählt, ein abgeleiteter lässt sich
per Haken übernehmen. Gibt die Klasse die Bücher zurück, verschwindet das Abzeichen wieder.
Das Kollegium sieht dieselben Einträge in _Mein Portal → Klassensätze_.

**Die Klasse kommt von den Kindern, nicht vom Namen** (07.09.2026 aufgeschrieben, weil die
Frage beim Büchertausch aufkam): Die abgeleiteten Einträge fragen bei jedem Aufruf, welche
Klasse in der Schülerliste steht und was deren Kinder gerade ausgeliehen haben. Beim
Büchertausch im Juni hat die 7R2 also schon die Bücher der 8 — sie stehen bis zu den Ferien
unter _07R2_, weil die LUSD die Kinder erst danach hochschiebt. Ist die Versetzung gelaufen,
steht dieselbe Liste unter _08R2_, ohne dass jemand etwas umträgt; die nachrückende Klasse
erscheint im selben Moment mit ihren eigenen Büchern unter _07R2_. Für den Nachzügler, der
mitten im Jahr dazukommt, zeigt die Übersicht damit immer das, was seine Klasse tatsächlich
hat. **Handgepflegte Einträge wandern nicht mit** — die hängen am Klassennamen („die 7R2
liest Mathe 7" gilt für jede künftige 7R2) und bleiben stehen, bis jemand sie ändert. In den
Wochen zwischen Tausch und Versetzung stehen deshalb beide untereinander: der alte
Handeintrag und der neue mit dem Abzeichen _aus Ausleihen_.

## Schuljahreswechsel (LMF-Plan)

Menü _System → Schuljahreswechsel_ (seit 05.09.2026; der Plan wird ein- bis zweimal im
Jahr gebraucht und steht deshalb nicht dauerhaft im Bibliotheks-Menü). Abgänger bleiben
unter _Verwaltung_, der LUSD-Abgleich mit Versetzung unter _Einstellungen →
LUSD & Versetzung_.

Die frühere Excel-Liste der Schule als zwei Pläne (seit 06.09.2026 so benannt, weil
„Rückgabe" und „Ausgabe" allein unklar waren — es sind zwei verschiedene Dinge zu
verschiedenen Zeiten):

- **Büchertausch vor den Sommerferien:** Alle Klassen geben die alten Schulbücher ab und
  bekommen direkt die neuen. Abschlussklassen und Klassen, die zum neuen Schuljahr neu
  gebildet werden (die 6er, die auf die Zweige H, R, G verteilt werden), geben **nur
  zurück** — der Vorschlag belegt bei ihnen die Besonderheiten mit „nur Rückgabe" vor;
  der Text gehört dann der Bibliothek und lässt sich ändern oder löschen.
- **Bücherausgabe nach den Sommerferien:** Nur die neu gebildeten Klassen bekommen ihre
  Bücher — die **Eingangsjahrgänge** (_Einstellungen → Ausleihe & Fristen_, Vorgabe
  „5, 7"). Der Vorschlag für diesen Plan enthält nur sie; alle anderen Klassen bleiben
  draußen.

**Der Plan ist eine Reihenfolge von Klassen, die das Programm auf Schultage und Stunden verteilt:**
Abschlussklassen zuerst, dann jeder Schultag Stunde 1 bis 6, eine Klasse je Stunde, die
Reihenfolge läuft über die Tage weiter; Wochenenden und die hinterlegten Ferien fallen aus.
Wochentag und Datum tippt niemand mehr — im Excel standen sie zweimal falsch.

- Oben _Büchertausch vor den Sommerferien_ / _Bücherausgabe nach den Sommerferien_
  umschalten; je Art gibt es einen Plan pro Schuljahr. Rechts _Als PDF_, _Plan verwerfen_,
  _Veröffentlichen_ (nur bei einem Entwurf), _Plan speichern_. Die Leiste bleibt beim
  Scrollen oben stehen; der Satz darunter sagt, was man sieht („Vorschlag aus dem
  Vorjahr, noch nicht gespeichert", „Entwurf vom …", „veröffentlicht am …").
- **Entwurf und Veröffentlichen** (seit 06.09.2026): _Plan speichern_ legt einen
  **Entwurf** an — zentral gespeichert, auf jedem PC gleich, aber für das Kollegium
  unsichtbar: nicht im Portal, nicht im PDF des Portals, und es werden keine Fristen
  gesetzt. _Als PDF_ im Planer enthält den Entwurf — das ist das Blatt für die Abnahme
  durch die Schulleitung. _Veröffentlichen_ speichert den aktuellen Stand und gibt ihn
  frei; ab dann sieht das Kollegium den Plan, beim Büchertausch werden die Termine die
  Fristen der Klassen, und jede weitere Speicherung gilt sofort (die Korrektur-Mail von
  früher). Einen zweiten Entwurf neben dem veröffentlichten Plan gibt es nicht — der Plan
  des nächsten Schuljahres ist ein neuer Plan und beginnt wieder als Entwurf.
- **Das PDF ist immer EIN Blatt** (seit 06.09.2026). Ein voller Plan bricht nicht mehr auf
  eine zweite Seite um: Bis rund vierzig Terminen sieht das Blatt aus wie das gewohnte
  Excel; darüber rückt der Satz enger zusammen und stellt ab etwa sechzig Terminen
  Rückgabe und Ausgabe nebeneinander in zwei Spalten (dort steht der Wochentag abgekürzt).
  Reicht eine Spalte für einen Abschnitt nicht, geht er mit „(Fortsetzung)" und eigenem
  Tabellenkopf daneben weiter. Es wird nie etwas weggelassen.
- **Zeitraum** (seit 06.09.2026 je Art verschieden): Der **Büchertausch endet am
  Donnerstag vor den Sommerferien in der 4. Stunde** — _Letzter Tag_ und _Ende am letzten
  Tag_ sind aus der Ferientabelle Hessen (KMK-Beschluss, bis 2030 hinterlegt) vorbelegt,
  die Reihenfolge läuft rückwärts davor, und der Satz darunter nennt Ferien und den
  gerechneten Beginn („Der Plan beginnt Donnerstag, 11.06.26 in der 3. Stunde"). Kommen
  Klassen dazu, beginnt der Plan früher; das Ende bleibt. Die **Bücherausgabe beginnt** —
  _Erster Tag_ (vorbelegt: erster Schultag nach den Ferien), _Beginn am ersten Tag_
  (vorbelegt 2. Stunde, wie im Plan 2026: Montag 10.08., 2. Stunde). Dazu
  _Stunden je Tag_ (Vorgabe 6). Fehlt ein Jahr in der Ferientabelle, sagt der Satz das,
  und der Tag wird von Hand eingetragen. **Die Sommerferien pflegt die Schule selbst**
  (seit 06.09.2026): _Einstellungen → LUSD & Versetzung → Sommerferien_ zeigt die Jahre
  des Programms und nimmt weitere auf (Beginn und Ende, Quelle
  kmk.org/service/ferienregelung); ein eigener Eintrag für ein Programmjahr gilt vor dem
  Programm. Zwei Jahre vor dem letzten bekannten Jahr warnt _Einstellungen →
  Betriebsbereitschaft_ („Ferientabelle") und zeigt auf diese Einstellung. Darunter die
  **freien Tage**: Wochenenden und
  die gesetzlichen Feiertage Hessens (Fronleichnam!) überspringt der Plan von selbst;
  bewegliche Ferientage, pädagogische Tage und Brückentage trägt man mit Datum und Grund
  ein — Chip _Tag freihalten_ in der Spalte _Freie Tage_ neben den drei Feldern, dann
  ein kleines Fenster mit Datum und Grund. Die Zeile _Übersprungen: …_ nennt jeden Werktag im
  Plan-Zeitraum, der ausfällt, mit Grund — so ist ein fehlender Donnerstag in der
  Tabelle erklärt.
- **Reihenfolge**: die bekannte Tabelle, nur bearbeitbar. Zeilen ziehen oder mit den
  Pfeilen schieben; alles Weitere steht im Menü ⋮ der Zeile: _An den Anfang_ und _Ans
  Ende_ (die weiten Wege mit einem Klick), _Mit der Zeile davor
  zusammenlegen_ legt zwei Klassen in eine Stunde („10R1/10R2"; die Klassen stehen dann
  als Chips mit ×), _In einzelne Stunden trennen_ macht daraus wieder zwei, _Zeile davor
  einfügen_ setzt eine Zeile ohne Klasse („Bücher setzen", „Nachzügler", „Aufräumen" — sie
  braucht einen Vermerk), _Klasse aus dem Plan nehmen_, _Zeile entfernen_. Wochentag,
  Datum und Stunde rechnet der Server bei jeder Änderung neu (Vorschau), gespeichert wird
  erst mit _Plan speichern_.
- **Fester Platz** (im Menü ⋮: _Datum und Stunde festlegen_): Die Klasse mit dem
  Ausflug bekommt Datum und Stunde von Hand — vorbelegt mit ihrem bisherigen Platz —, die
  übrigen Zeilen fließen um sie herum und lassen die belegte Stunde aus. _Festen Platz
  lösen_ gibt die Zeile dem Fluss zurück. Feste Plätze gelten für diesen Plan; der
  Vorschlag fürs nächste Jahr bringt sie nicht mit.
- **Noch nicht im Plan** (direkt über der Tabelle, seit 06.09.2026): die Klassen, die
  keine Zeile haben und für die keine Regel gilt — die neue Klasse nach dem LUSD-Import.
  _06F4 einplanen_ setzt sie **an ihren Platz**: hinter die letzte Klasse desselben
  Jahrgangs und Zweigs (06F4 hinter 06F3), sonst hinter den Jahrgang, sonst ans Ende;
  die Tabelle springt dorthin und hebt die Zeile kurz hervor. Wer sie woanders will,
  zieht den Chip auf eine Zeile — die Klasse landet davor. _Andere Klasse eintragen_
  öffnet ein kleines Fenster für einen Namen, den es noch nicht gibt („07G1" vor dem
  August-Import). Was Regel oder gespeicherter Plan bewusst auslassen — die Oberstufe,
  die Rückgabe und Ausgabe an dieser Schule selbst organisiert —, steht eingeklappt
  hinter _11 Klassen bleiben draußen_ und lässt sich von dort genauso einplanen.
- **Klassen ohne Schüler** („ohne Schüler" an der Klasse, oben in der Reihenfolge
  gezählt): eine Klasse aus dem Vorjahr, die es nicht mehr gibt, oder eine vor dem
  LUSD-Import getippte. Klassen muss niemand anlegen oder löschen — ein Name im Plan
  registriert sich selbst, und eine Klasse ohne aktive Schüler verschwindet von allein
  aus jeder Liste. Nach dem LUSD-Import zeigt der Satz, was übrig blieb: raus damit,
  oder es kommt noch. Neue Klassen mit Schülern erscheinen unter _Noch nicht im Plan_.
- **Vorjahr als Vorlage**: Ist der letzte Plan vorbei, beginnt der nächste mit dessen
  Reihenfolge und Auslassungen; neue Klassen hängen hinten an. Ganz ohne Vorjahr gilt die
  Regel: Abschlussklassen zuerst, dann Jahrgang absteigend, Oberstufe unten, am Ende die Zeilen „Nachzügler" und „Aufräumen" wie im Plan der Schule.

Das Kollegium sieht den **veröffentlichten** Plan in _Mein Portal → LMF-Plan_, für alle
gleich und immer auf dem aktuellen Stand; _Als PDF_ liefert die gewohnte Liste (Wochentag,
Datum, Stunde, Klassen, Besonderheiten — so, wie sie im Planer eingetragen sind),
getrennt nach Büchertausch und Bücherausgabe.

**In die Spalte „Besonderheiten" gehört kein Schülername.** Was dort steht, liest das ganze
Kollegium im Portal und steht im PDF, das weitergegeben und ausgedruckt wird — der Plan ordnet
Klassen und Stunden, nicht einzelne Kinder. „Nachzügler" oder „Restbücher aus 9R2" ist richtig,
ein Name daneben nicht.

**Der Termin einer Klasse beim Büchertausch ist die Frist ihrer Schulbücher.** Beim
Veröffentlichen (und bei jeder Speicherung eines veröffentlichten Plans)
folgen die offenen Schulbuch-Ausleihen der Klassen (die Meldung nennt die Zahl); neue
Ausleihen bekommen ihn gleich. Wer am Termintag seiner Klasse oder danach noch ein
Schulbuch bekommt, gibt es erst im nächsten Schuljahr zurück: Die Frist ist dann der
Stichtag des folgenden Schuljahres. Fällt eine Klasse aus dem Plan oder wird er verworfen,
gehen die Fristen an den allgemeinen Stichtag zurück (_Einstellungen → Ausleihe_, Vorgabe
31.07.). Nicht angefasst: gesperrte Schüler, mehrjährige Ausleihen, von Hand gesetzte
Fristen und Ausgabe-Pläne. (§2.3)

## Leserdatei

Bis zum 16.09.2026 hieß dieser Menüpunkt _Schülerdatei_ und führte nur Schüler. Er führt
jetzt **alle, die Bücher bekommen können** — Schüler und Kollegium in einer Liste.

- **Wer steht hier drin?** Schüler kommen aus der LUSD. Lehrkräfte und LiV entstehen
  entweder von selbst, sobald sich jemand über „Mein Portal" anmeldet, oder Sie legen sie
  mit _Neuer Leser_ an. Wer wer ist, steht als **Art** in der Akte — Schüler, Lehrkraft
  oder LiV. Die Art entscheidet keine Rechte; ausleihen darf jeder aktive Leser.
- **Reiter**: _Aktive Leser_, _Ehemalige / Archiv_ und (mit dem Recht zum Löschen)
  _Papierkorb_.
- **Suche** über den ganzen Bestand (Name, Klasse, Barcode). Zeile anklicken → Akte.
- **Akte**: _Ausleihen & Historie_ und _Stammdaten & Adresse_ — für jeden dieselben Felder
  an derselben Stelle. Bei einer Lehrkraft sind drei davon verschlossen, weil sie ihr
  nicht gehören: Klasse, Abgangsjahr und LUSD-Kennung. Geburtsdatum, Ausweisnummer,
  Anschrift und Eltern-E-Mail stehen jedem offen und bleiben beim Kollegen meist leer.
  Die Art lässt sich zwischen Lehrkraft und LiV umstellen; über die Schüler-Grenze geht
  sie nicht — ein Schüler kommt aus der LUSD und bleibt Schüler.
- **Dokumente** in der Akte: Ausweis drucken, Kontoauszug, Ersatzforderung (nur bei offenem
  Schaden), DSGVO-Auskunft. Die drei letzten gibt es nur beim Schüler. Gibt es für das
  Kind einen Schadensersatz-Bescheid, steht unter _Ausleihen & Historie_ die Karte
  **Schadensersatz-Bescheide**: Referenznummer, Positionen, Betrag, Briefdatum, Frist, Zustand
  und _Nachdruck_ (derselbe Brief mit derselben Nummer).
- **Ausweis drucken**: Auf der Karte steht, was sie ist — _Schülerausweis_ oder
  _Lehrerausweis_. Die Gültigkeit („Gültig bis 31.07. …") trägt nur der Schülerausweis;
  der Ausweis einer Lehrkraft läuft mit keinem Schuljahr ab. Neue Ausweisnummern beginnen
  mit `A-`; ältere Karten mit `S-` oder `L-` bleiben gültig und werden weiter gelesen.
- **Gebühren & Schäden**: offen / bezahlt; _Bezahlt_ bucht aus, _Stornieren_ verlangt einen
  Grund. Steht eine offene Forderung noch auf keinem Brief, gibt es hier _Bescheid erstellen_
  (derselbe Dialog wie im Mahnwesen). (§14)
- **Verlust/Schaden melden** an der Ausleihzeile beendet die Ausleihe und legt die Forderung an.
  Das Kind steht danach nicht mehr in der Mahnliste, sondern im Mahnwesen unter _Schadensersatz_.
  Der Brief an die Eltern ist der nächste, eigene Schritt.
- **Sperren** verlangt eine Begründung — sie steht danach an der Theke.
- **Neuer Leser** per Formular. Der Dialog fragt zuerst, wer das ist: Schüler, Lehrkraft
  oder LiV. Klassenweise legt man Schüler besser über den LUSD-Import an
  (_Einstellungen → LUSD & Versetzung_).

  Bei einer Lehrkraft oder LiV ist die **Schul-E-Mail Pflicht**. Sie ist keine
  Kontaktangabe, sondern der Schlüssel: Mit ihr entsteht zugleich der Zugang zu „Mein
  Portal", und die Person steht später nicht doppelt da, wenn sie sich selbst anmeldet.
  Freigeschaltet wird der Zugang nur, wenn Sie auch _Benutzer & Rechte_ dürfen; sonst
  entsteht eine Zugangsanfrage, die ein Admin freischaltet. Eine **Rolle** vergibt dieses
  Formular nie.
- **Stapelaktionen**: Klasse markieren → Ausweise oder Etiketten für alle drucken.
- Reiter **Ehemalige / Archiv** (wer die Schule verlassen hat) und **Papierkorb**; endgültiges
  Löschen/Anonymisieren nur mit Namensbestätigung (DSGVO-Kette, §8).
- **Doppelter Datensatz?** (Recht „Schüler zusammenführen", ab Werk nur Admin; unten im Reiter _Stammdaten & Adresse_): Steht dieselbe
  Person zweimal in der Kartei, beide Datensätze zusammenführen. Der Grund ist je nach Art
  ein anderer, und er steht im Dialog: beim Schüler eine Namensänderung in der LUSD, die
  der Export ohne Schüler-ID nicht wiedererkannt hat; beim Kollegen ein Eintrag von Hand,
  der sich später selbst angemeldet hat. Schüler und Kollegen lassen sich **nicht**
  miteinander verschmelzen — Lehrkraft und LiV schon. Es
  bleibt der Datensatz, dessen Ausweis das Kind in der Hand hat; Ausleihen, Gebühren und
  Historie des anderen wandern hinüber, vom Foto bleibt das jüngere. Die Suche im Dialog
  findet auch Ehemalige und Gesperrte. **Wurde für den aufgelösten Datensatz schon ein
  Ausweis gedruckt, diese zweite Karte einziehen und vernichten:** Ihre Nummer ist danach
  frei und kann beim nächsten neu angelegten Schüler wieder vergeben werden.
  ([LUSD.md](LUSD.md) §5)

## Mahnwesen

Register **Alle · Akut fällig (bis 14 Tage) · Eskaliert**, Filter nach Klasse. Mahnbriefe an
Eltern oder für eine ganze Klasse drucken; **Sammel-Mahnlauf** per Mail an die Klassenleitungen
(Klassen wählen, Empfänger prüfen, dann senden). Die Mahnstufe steigt beim **Druck** des
Mahnbriefs, nicht beim Mailversand. Lehrkräfte werden nie angemahnt. Welche Klasse an welche
Lehrkraft geht, steht unter _Einstellungen → Mahnwesen-Routing_. (§3)

**Zwei Fragen, vier Reiter.** _Alle · Akut fällig · Eskaliert_ fragen, wer Bücher zu spät hat.
Der vierte Reiter **Schadensersatz** fragt, wer Geld schuldet: Sobald für ein Kind ein Verlust
oder Schaden gemeldet ist, steht es dort, mit einem Stand und dem nächsten Schritt je Zeile:
_Bescheid noch nicht erstellt_ → _Bescheid erstellen_; _Frist läuft_ → abwarten; _Frist
abgelaufen_ → _Übergeben_ (Original und Buchungsbeleg an die Aufsicht); _übergeben_ → nichts
mehr zu veranlassen; _Rückgabe nach Übergabe_ → die Aufsicht informieren. Die Zahl am Reiter ist,
was bei der Schule liegt. Erledigte Briefe (bezahlt, storniert) stehen nur noch in der Akte.

**Schadensersatz-Bescheid:** In der Mahnliste genau einen Schüler markieren und _Schadensersatz-
Bescheid_ drücken; außerdem _Bescheid erstellen_ im Reiter _Schadensersatz_ oder in der Akte an
der Gebühren-Karte. Der Dialog zeigt die überfälligen Bücher des Kindes und seine offenen
Forderungen mit einem Betragsvorschlag nach der Staffel der Schule samt Herleitung; Beträge sind
änderbar, Zeilen abwählbar. Eine Verlustmeldung je Buch ist vorher nicht nötig: _Bescheid
erstellen_ bucht die gewählten Bücher als Verlust (die Ausleihe endet, das Kind verschwindet aus
der Mahnliste), vergibt die Referenznummer (nie zweimal), setzt die Frist als Datum und öffnet den
Brief; alles zusammen oder nichts. Ist ein Buch inzwischen zurück oder schon gemeldet, weist der
Dialog ab, ohne eine Nummer zu verbrauchen — neu öffnen genügt. _Nachdruck_ ist immer derselbe
Brief, auch nach einem Umzug. Die Angaben für den Brief stehen in den Einstellungen unter
_Schadensersatz_; fehlen sie, sagt der Dialog welche.

**Kommt ein abgerechnetes Buch doch zurück:** Wird ein Buch mit offener Forderung „nicht
zurückgegeben" an der Theke gescannt, storniert die Theke die Forderung („Buch reaktiviert.
Die Forderung über … € wurde storniert — das Buch ist zurück."). Dasselbe gilt für _Gefunden_ im
Fehlbestandsbericht der Inventur. Liegt der Bescheid dazu schon bei der Schulaufsicht
(_Übergeben_), storniert die Anwendung nichts: Sie zeigt den Hinweis, dass die Aufsicht
unverzüglich zu informieren ist, und die Forderung bleibt offen.

## Abgänger

Abgänger sind die Abschlussklassen des laufenden Schuljahres — 9H (und das freiwillige 10. Hauptschuljahr), 10R und 13 —, also die Kinder, die zum Schuljahresende gehen. Sie
sind noch an der Schule und leihen bis zuletzt aus. Die Regel ist dieselbe wie bei der
Versetzung; eingestellt wird nichts, die Klasse weiß es.

Die Liste zeigt **vom 1. Mai bis 31. Juli** die Abgänger, die **noch Bücher haben** — zum
Einsammeln vor der Entlassung. Kontoauszüge drucken oder an die Klassenleitungen mailen;
wer alles zurückgegeben hat, verschwindet. Außerhalb dieser Zeit steht hier nur der
Hinweis mit den Daten.

**Gedruckt wird, was die Liste zeigt:** mit gewählter Klasse nur diese, mit einer Suche nur
die Gefundenen. So lässt sich auch ein einzelner Kontoauszug mit Freigabezeile nachdrucken —
Namen suchen, drucken. Der Versand per Mail bleibt klassenweise.

Wer die Schule dann **verlassen** hat (Versetzung: Abschlussklasse; LUSD-Import: fehlt im
neuen Export), steht nicht mehr hier, sondern in der Leserdatei unter _Ehemalige /
Archiv_ — mit offenen Büchern zusätzlich im Mahnwesen. Er bleibt bis zum Ende der
**Karenzzeit** (Vorgabe 90 Tage; _Einstellungen → Datenschutz & Sitzung_) als gesperrter
Datensatz erhalten — Zeit, eine falsche Zuordnung noch zu reparieren — und wird danach
automatisch anonymisiert. Bleibt ein Buch oder eine Forderung dauerhaft offen, meldet
_Einstellungen → Betriebsbereitschaft_ nach einem Jahr „Ehemalige mit offenen Vorgängen": In der
Akte das Buch als Verlust melden, dann die Forderung bezahlt oder storniert buchen — danach
löscht das System von selbst. Der Buch-Barcode wird dabei nicht neu vergeben, die
Ausweisnummer des Kindes nach der Löschung schon. Die Karenz läuft ab dem **späteren**
Zeitpunkt: dem Abgang oder der letzten Rückgabe beziehungsweise Schadensregulierung. Wer
erst lange nach dem Abgang zurückgibt, hat damit trotzdem die volle Karenz. Die endgültige Löschung ab dem 30. Januar des Folgejahres trifft nur Datensätze, die die Karenz durchlaufen haben, also schon anonymisiert sind (§8).
(§8, [LUSD.md](LUSD.md) §4)

## Bestellwesen

Fünf Reiter: **Bestellbedarf** (Lernmittel unter der Bedarfsschwelle — automatisch; Titel in
den Warenkorb, Lieferant wählen, Bestellung geht als Mail mit Bestätigungs-Link raus; der
Warenkorb zeigt zwei Abschnitte _Lernmittelfreiheit (Land)_ und _Schülerbücherei
(Schulträger)_ — je Abschnitt geht eine eigene Bestellung an den Händler, eine Position lässt
sich per Knopf in den anderen Abschnitt schieben; im Bestelldetail lässt sich der Topf einer
Bestellung nachträglich mit Grund korrigieren) ·
**Wareneingang** (Positionen einbuchen → Etiketten) · **Bestellhistorie** (Detail, Status,
Händlerbestätigung; Filter _Mittelherkunft_ nach Topf oder „ohne Zuordnung", die Kennzahlen
im Kopf zusätzlich je Topf, sobald mehr als ein Topf Bestellungen hat) · **Klassensatz-
Reservierungen** (Warteschlange aus dem Kollegium; _Abschließen_ schickt die Bereit-Mail) ·
**Wünsche & Meldungen** (Anliegen aus dem Portal, _Erledigen_ mit Notiz an die Lehrkraft).
Lieferanten und Hauptlieferant stehen in den Einstellungen. (§7)

## Inventur

_Neue Bestandsprüfung starten_ → Umfang wählen (komplett, eine Signatur, Fach/Klasse) →
scannen; Fortschrittsbalken. **Achtung:** _Inventur abschließen_ bucht alles Ungescannte im
Umfang als Verlust — vorher den Fehlbestandsbericht prüfen; dort lassen sich Funde wieder
zurückholen. Laufende Inventuren können fortgesetzt oder verworfen werden. (§6)

---

## Statistiken

Bestand, aktuell verliehen, Zirkulationsquote, Wiederbeschaffungswert; Ausleihen pro Monat;
Überfällige nach Dauer; **Renner** (meistausgeliehen) und **Ladenhüter** (seit über zwei Jahren
nicht ausgeliehen) mit Detailseite und Filter. Ohne Schülernamen — die Statistik zählt
Ausleihen, nicht Personen. (§11)

## Bestandsbücher

Zwei Reiter, beide mit einem Blatt zum Ausdrucken und Abheften: **Zugangsbuch** (welche
Exemplare in einem Zeitraum in den Bestand gekommen sind — Eingangsdatum, Nummer, Titel,
Lieferant) und **Abgangsbuch** (welche hinausgegangen sind — Datum, Nummer, Titel,
Signatur, Grund). Vorbelegt ist das laufende Schulhalbjahr mit den Stichtagen 15.3. und
15.9.; beide Datumsfelder lassen sich überschreiben. Land und Schulträger stehen getrennt,
jeweils mit eigener Stückzahl.

Ein bestelltes Buch erscheint im Zugangsbuch erst, wenn Sie es unter _Bestellungen →
Wareneingang_ eingebucht haben — mit dem Tag der Lieferung, nicht dem der Bestellung.
Was noch im Zulauf steht, gehört nicht in den Nachweis: Es liegt noch beim Händler.

**Was nicht auf dem Blatt steht, steht ausdrücklich darunter:** Exemplare, die vor der
Einführung der Bestandsbücher ausgesondert wurden, tragen kein Abgangsdatum — ihre Zahl
nennt der Ausdruck, statt Vollständigkeit zu behaupten. Im Zugangsbuch stehen Bücher ohne hinterlegte
Bestellung unter „ohne Zuordnung", weil nicht belegt ist, aus welchem Geld sie bezahlt
wurden. (§13)

## Bestellberichte

Monatsbericht, Jahresbericht und Lieferantenabrechnung als PDF; in Blöcken je Topf mit
eigener Summe und der Gesamtsumme darunter, über _Mittelherkunft_ auch für nur einen Topf.
Ohne Preise im Bestellwesen zählen die Blätter Exemplare statt Euro und heißen
entsprechend. Bis zum 17.09.2026 ein Reiter im Bestellwesen. (§7)

## System-Logs

_Allgemeines Logbuch_ (jede Buchung) und _Admin-Audit-Log_ (wer hat wann was geändert).
Aufbewahrung 24 Monate, einstellbar. Dritter Reiter **Tresen-Auskunft** (eigenes Recht
`audit_details`, ab Werk nur Admin): Ein Buch liegt auf dem Tresen, sein Exemplar ist
längst gelöscht — die Barcode-Suche zeigt, was das Protokoll dazu noch weiß (Titel,
letzte Ausleihen), auch wenn der Titel komplett gelöscht wurde. Jede Abfrage wird
selbst protokolliert; nach DSGVO-Tilgung zeigt auch dieser Weg nichts mehr. (§10)

## Benutzer & Rechte

Reiter **Benutzer** (anlegen, bearbeiten, deaktivieren; **Zugangsanfragen** aus der
Selbstanmeldung freischalten) und **Rollen & Rechte** (Matrix; Änderungen wirken sofort auf
Menü und Schnittstelle).

**Vier Rollen vergeben Sie hier: Admin, Leitung, Mitarbeit, Helfer.** Die Leitung darf
alles, was der Admin darf, außer zwei Dingen: Benutzer & Rechte und die Einstellungen.
(Der Grund ist nicht Misstrauen: Wer Benutzerkonten ändert, ändert auch E-Mail-Adressen —
und die Adresse ist die Anmeldung.)

**Kollegium steht nicht in der Matrix, und das ist Absicht.** Es ist keine Rolle, sondern
der Ausgangszustand jeder Lehrkraft: Sie meldet sich über „Mein Portal" mit ihrer
Schuladresse selbst an, Sie schalten sie frei, und damit sieht sie den Bestand, merkt vor,
meldet Fehler und leiht an der Theke auf ihren Namen. Das sind erst einmal alle. Wer mehr
können soll, bekommt von Ihnen eine der vier Rollen. Was das Kollegium darf, ist fest
eingebaut und je Schule nicht verstellbar. (§12)

## Einstellungen

15 Kategorien, jede einzeln speicherbar (§17):

| Kategorie               | Wofür                                                                                                                                                                                                                                                                                                                                                                                         |
| ----------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Schule                  | Name, Anschrift, Eigentumsvermerk auf Etiketten                                                                                                                                                                                                                                                                                                                                               |
| Ausleihe & Fristen      | Tage je Buch/Medium, Limit je Schüler, LMF-Stichtag, Ferien-Leseclub (festes Rückgabedatum über die Ferien)                                                                                                                                                                                                                                                                                   |
| Mahnwesen               | automatische Sperre: ab wie vielen überfälligen Medien, nach wie vielen Tagen                                                                                                                                                                                                                                                                                                                 |
| Mahnwesen-Routing       | Klasse → Klassenleitung (Empfänger für Mahnlauf und Abgänger-Kontoauszüge)                                                                                                                                                                                                                                                                                                                    |
| Bestellwesen            | Bedarfswarnung, Bedarfsschwelle, Preise erfassen                                                                                                                                                                                                                                                                                                                                              |
| Lieferanten             | Händler, Kundennummern, genau ein Hauptlieferant                                                                                                                                                                                                                                                                                                                                              |
| Schlagworte             | Wörter am Titel umbenennen und zusammenführen (alle Titel ändern sich mit), löschen — auch mehrere auf einmal: vorn markieren, unten „Löschen“ —, Verweise für andere Schreibweisen („Tierfantasy“ → „Fantasy“), Filter-Markierung fürs Portal                                                                                                                                                |
| Schadensersatz          | Angaben für den Schadensersatz-Bescheid: Schulamtsbereich und Schulnummer (Teil der Referenznummer), Aufsicht, Schulleitung, Geschäftszeichen, Zahlstelle und Bankverbindung, Zahlungsfrist (Vorgabe 28 Tage); **„Immer mit dem Einkaufspreis rechnen"** — aus gilt der heutige Listenpreis (so verlangt es die Arbeitshilfe), an immer der Preis, den die Schule damals bezahlt hat                                                                                                                                                                                 |
| Datenschutz & Sitzung   | Löschfristen, Abgänger-Karenzzeit, Theke leeren, Sperrbildschirm                                                                                                                                                                                                                                                                                                                              |
| Erreichbarkeit & Alarme | öffentliche Adresse (Basis für Bestätigungs-Link, Katalog, Monitor), Alarm-Empfänger                                                                                                                                                                                                                                                                                                          |
| Mail                    | Postausgang mit Verbindungstest, zwei Vorlagen: Elternbrief zur Mahnung (gedruckt) und Bestellmail an den Händler                                                                                                                                                                                                                                                                             |
| LMF-Aktionen            | alle Lernmittel einer Klasse auf ein neues Datum verlängern                                                                                                                                                                                                                                                                                                                                   |
| Datenverwaltung         | Katalog-Import (Littera), Bestands-Import (Kombi-CSV, übernimmt vorhandene Nummern), Listenimport (ISBN + Stückzahl → neue Titel samt Exemplaren mit B-Nummern; der Knopf fehlte vom 21.06. bis 07.09.2026; ein zweiter Klick mit derselben Dateiauswahl legt nichts doppelt an, eine neue Auswahl ist ein neuer Lauf), Cover-Synchronisation, Katalog-Export, Offline-Sicherungen einspielen |
| LUSD & Versetzung       | LUSD-Abgleich, Versetzung zum Schuljahresende, Sommerferien für den LMF-Plan                                                                                                                                                                                                                                                                                                                  |
| Betriebsbereitschaft    | Selbstprüfung: eingerichtet, aber nicht in Betrieb? (§15)                                                                                                                                                                                                                                                                                                                                     |

## Mein Portal (Kollegium)

Lehrkräfte sehen genau diesen Bereich: **Suchen & Reservieren** (Bestand mit Verfügbarkeit
und Warteschlange; Klassensatz reservieren mit Klasse, Anzahl, Datum) · **Klassensätze** der
eigenen Klassen · **LMF-Plan** (seit 05.09.2026: Rückgabe- und Ausgabetermine je Klasse,
für alle gleich, als PDF; → _LMF-Plan_) · **Schulbücher** (seit 03.09.2026, für die Fachsprecher: aufgebaut wie die
Klassensätze, nur nach Fach statt Klasse. Oben Suche (Titel, ISBN, Autor, Fach) und die
Filter Jahrgang und Schulzweig; darunter je Fach eine Zeile mit Exemplaren, Titeln und
Verliehenen, die sich zu den Cover-Kacheln der Bücher aufklappt. „Als PDF" sitzt an
jedem Fach und druckt genau dieses Fach mit der aktuellen Filterung: eine Zeile je Buch
mit Coverbild, Titel, Autor, ISBN, Jahrgang, Schulzweig, Zähldatum und den Zahlen. Die
Spalte **Gezählt** erscheint nur, wenn in der Auswahl überhaupt schon gezählt wurde. Im Kopf steht,
welcher Ausschnitt es ist — ein gefilterter Ausdruck ist sonst nicht von der vollen Liste
zu unterscheiden. Gezählt wird nur, was den
Lernmittel-Schalter trägt. Den **Schulzweig** pflegt die Bibliothek am Buch: In der
Buchmaske erscheint das Feld, sobald „Lernmittel" eingeschaltet ist; leer heißt „gilt für
alle Zweige" — solche Bücher erscheinen deshalb unter **jedem** Zweig-Filter, und die
Auswahl „Ohne Schulzweig" zeigt umgekehrt nur sie. Littera hat den Zweig nie mitgeliefert,
der Altbestand ist also zunächst ohne. Ein Buch, dessen Coverbild von außerhalb kommt
(Deutsche Nationalbibliothek, Google Books), erscheint im PDF ohne Bild: gedruckt wird nur,
was auf dem Server liegt. Tauchen dort Standorttexte wie „Buch Deu 6/Cha 126" als Fach auf,
stammt der Bestand aus einem Import vor dem 03.09.2026: `scripts/repair_fach_kategorie.sql`) · **Meine Anliegen** (Buchwunsch oder „Etwas stimmt
nicht" an die Bibliothek). (§12, Rolle Kollegium)

---

## Bedienung ohne Maus

Die Hauptbildschirme lassen sich mit der Tastatur bedienen — geprüft an den Stellen, die
das Browser-Gate misst (§19): Das erste **Tab** auf jeder Seite landet auf „Zum Inhalt
springen" (überspringt die Seitenleiste), **Tab** wandert durch Felder und Knöpfe,
**Enter/Leertaste** löst aus, **Escape** schließt den obersten Dialog und gibt den Fokus an
die Stelle zurück, von der er kam. In Listen öffnet der **Name** (Schüler) beziehungsweise
der **Titel** (Buch, Signaturen-Regal) die Akte. Im LMF-Planer lassen sich Zeilen außer per
Ziehen auch über ihr Zeilenmenü verschieben (nach oben, nach unten, an den Anfang, ans Ende).
Wer im Betriebssystem „Bewegung reduzieren" eingestellt hat, sieht keine Ein- und
Ausblendungen. Meldungen am oberen Rand bleiben stehen, solange die Maus oder der Fokus
darauf liegt. An der Theke hält das Scanfeld den Fokus — das ist gewollt, der Scanner tippt
blind.

Was nicht ohne Maus geht: die Zeichenfläche des Ausweis-Designers (§19, Bekannte Lücken). Ein
Durchgang mit einem Screenreader hat nicht stattgefunden.

## Wenn etwas nicht geht

- **Bestellung geht ohne Link raus / Katalog-Adresse fehlt** → _Erreichbarkeit & Alarme_:
  öffentliche Adresse eintragen.
- **Mahnliste kommt bei niemandem an** → _Mahnwesen-Routing_: Klasse hat keine Lehrkraft.
- **Rote Meldung „Kein Backup"** → _Betriebsbereitschaft_ öffnen; dort steht je Punkt, was
  fehlt und wie es zu beheben ist.
- **Scanner tippt ins Leere** → einmal ins Scanfeld klicken; die Theke holt den Fokus nach
  jedem Scan selbst zurück.
- **Netz weg** → weiterarbeiten. Oben erscheint ein schmales Band, die Theke bleibt bedienbar,
  und der Bildschirm sperrt sich nicht. Scanne wie sonst: erst den Ausweis, dann die Bücher.
  Beides wird auf diesem Rechner gemerkt und gebucht, sobald die Verbindung zurück ist — du musst
  dafür nichts tun. Der Ausweis zeigt ohne Netz keinen Namen; das Band nennt die Nummer, die es
  sich gemerkt hat.
  - Meldet die Theke **„nicht eindeutig"**, kennt sie die Nummer nicht (ein sehr neues Buch, ein
    fremder Aufkleber). Dieser Scan ist NICHT gebucht — notiere ihn auf Papier.
  - **Namen suchen** geht ohne Netz nicht: Auf dem Theken-Rechner stehen keine Personendaten.
  - **Geräte** lassen sich ohne Netz nicht ausgeben oder zurücknehmen.
  - Muss der Rechner aus, bevor die Verbindung zurück ist: **„Sicherung speichern"** im Band, die
    Datei später unter _Datenverwaltung → Offline-Sicherungen einspielen_ übernehmen. Zweimal
    einspielen schadet nicht.
- **Nach einem Netzausfall: „x Buchungen brauchen einen Blick"** → Der Server hat beim Nachbuchen
  etwas anders gebucht, als es gescannt wurde — das Buch lag bei jemand anderem, oder eine
  Rückgabe kam zu spät. Öffne **Meldungen**, sieh die Zeilen durch und setze jede mit „Erledigt"
  ab, um die du dich gekümmert hast. Steht der Knopf nicht da, fehlt das Recht, Schülerdaten zu
  sehen — dann bitte die Bibliothek ansprechen.
