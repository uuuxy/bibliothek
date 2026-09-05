# Benutzerhandbuch

Stand: 2026-09-05. Für Bibliothekspersonal, Sekretariat und Schulleitung — geschrieben aus
Sicht der Arbeit am Tresen, nicht aus Sicht des Codes. Die fachlichen Regeln dahinter stehen
im [Fachkonzept](FACHKONZEPT.md); dort verweisen die §-Angaben hin. Ein Produktrundgang als
Video (8 min) zeigt jeden Bereich in Aktion.

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

## Überall suchen (seit 03.09.2026)

Oben auf jeder Verwaltungsseite steht **eine** Suchleiste (nur für Konten mit dem
Theken-Recht; sie bietet außerdem nur an, was das eigene Konto auch öffnen darf — wer den
Medienkatalog sehen darf, aber nicht die Schülerdatei, findet dort keine Ausweise). Sie versteht alles, was die Theke
versteht — Buch-Barcode, Littera-Etikett, Schülerausweis, ISBN, Name, Klasse, Titel — und
**springt nur hin:** ein Buch-Barcode oder eine ISBN öffnet die Buchakte, ein Ausweis die
Schülerakte, ein Name zeigt eine Trefferliste. Gebucht wird nirgends außer an der Theke;
dort und im Portal gibt es diese Leiste deshalb nicht. Ein Scanner funktioniert direkt
hinein (Enter entscheidet), die Taste **/** setzt den Fokus, **Esc** leert das Feld. Die
Filterfelder in den Listen bleiben Filter: Sie sieben nur, was schon auf dem Bildschirm ist.

## Ausleihe (Theke)

Der Startbildschirm nach der Anmeldung. **Ein Feld für alles:**

1. **Schülerausweis scannen** → die Theke öffnet sich: Foto, Klasse, Konto-Status, entliehene
   Bücher, Gebühren, Vormerkungen.
2. **Buch scannen** → ausgeliehen. Frist wird automatisch berechnet (Lernmittel bis zum
   Stichtag 31.07., andere Medien nach Tagen; Ferien werden übersprungen). (§2)
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
- **Überfällige** Bücher und die Mahnstufe stehen direkt an der Ausleihzeile.

**Außerdem:** Geräte (iPads, Taschenrechner, Beamer) laufen über dieselbe Theke, mit
Zubehör-Checkliste beim Scan (§5) · Kamera als Ersatz für den Handscanner (Knopf neben dem
Feld) · Passbild per Webcam · Ausweis drucken, Kontoauszug, DSGVO-Auskunft als PDF (§18).

Nach 5 Minuten ohne Eingabe schließt sich die Akte, nach 15 Minuten der Sperrbildschirm —
beides einstellbar (_Datenschutz & Sitzung_).

---

## Medienkatalog

- **Suche & Filter**: ein Feld für Titel, Autor, Fach, Klasse, Signatur; Kartenansicht mit
  Cover und Signatur.
- **Buchakte** (Klick auf eine Karte): Exemplare mit Status, aktuelle Ausleiher, Vormerkungen
  (Warteliste mit Schüler-Suche), Historie.
- **Titel-Verwaltung**: neuen Titel anlegen (ISBN-Eingabe holt Metadaten und schlägt eine
  Signatur vor — vor dem Speichern prüfen), bearbeiten, Cover tauschen, Meldebestand,
  Exemplare aussondern (Verlust, Schaden, Bestandskorrektur). (§13)
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

## Schuljahreswechsel (LMF-Plan)

Menü _System → Schuljahreswechsel_ (seit 05.09.2026; der Plan wird ein- bis zweimal im
Jahr gebraucht und steht deshalb nicht dauerhaft im Bibliotheks-Menü). Abgänger bleiben
unter _Verwaltung_, der LUSD-Abgleich mit Versetzung unter _Einstellungen →
Schuljahreswechsel_.

Rückgabe- und Ausgabetermine je Klasse — die frühere Excel-Liste der Schule. **Der Plan
ist eine Reihenfolge von Klassen, die das Programm auf Schultage und Stunden verteilt:**
Abschlussklassen zuerst, dann jeder Schultag Stunde 1 bis 6, eine Klasse je Stunde, die
Reihenfolge läuft über die Tage weiter; Wochenenden und die hinterlegten Ferien fallen aus.
Wochentag und Datum tippt niemand mehr — im Excel standen sie zweimal falsch.

- Oben _Bücherrückgabe_ / _Bücherausgabe_ umschalten; je Art gibt es einen Plan pro
  Schuljahr. Rechts _Als PDF_, _Plan verwerfen_, _Plan speichern_.
- **Rahmen**: erster Tag, Beginn am ersten Tag (der Donnerstag im Juni begann in der 3. Stunde), Stunden je Tag (Vorgabe 6).
- **Freie Tage**: Wochenenden und die gesetzlichen Feiertage Hessens (Fronleichnam!)
  überspringt der Plan von selbst; bewegliche Ferientage, pädagogische Tage und
  Brückentage trägt man hier mit Datum und Grund ein. Darunter steht _Übersprungen: …_
  mit jedem Werktag im Plan-Zeitraum, der ausfällt, und seinem Grund — so ist ein
  fehlender Donnerstag in der Tabelle erklärt.
- **Fester Platz** (Stecknadel in der Zeile): Die Klasse mit dem Ausflug bekommt Datum
  und Stunde von Hand — vorbelegt mit ihrem bisherigen Platz —, die übrigen Zeilen
  fließen um sie herum und lassen die belegte Stunde aus. _Lösen_ gibt die Zeile dem
  Fluss zurück. Feste Plätze gelten für diesen Plan; der Vorschlag fürs nächste Jahr
  bringt sie nicht mit.
- **Reihenfolge**: die bekannte Tabelle, nur bearbeitbar. Zeilen ziehen oder mit den
  Pfeilen schieben; _zusammenlegen_ legt zwei Klassen in eine Stunde („10R1/10R2"),
  _trennen_ macht daraus wieder zwei; _davor einfügen_ setzt eine Zeile ohne Klasse
  („Bücher setzen", „Nachzügler", „Aufräumen" — sie braucht einen Vermerk); das × am
  Klassen-Chip nimmt eine Klasse aus dem Plan. Wochentag, Datum und Stunde rechnet der
  Server bei jeder Änderung neu (Vorschau), gespeichert wird erst mit _Plan speichern_.
- **Nicht im Plan**: alle Klassen ohne Zeile. Ein Klick holt eine ans Ende, _Weitere
  Klasse_ nimmt Namen auf, die es noch nicht gibt („07G1" vor dem August-Import). Was
  hier bleibt, gilt als bewusst ausgelassen — die Oberstufe organisiert Rückgabe und
  Ausgabe an dieser Schule selbst.
- **Vorjahr als Vorlage**: Ist der letzte Plan vorbei, beginnt der nächste mit dessen
  Reihenfolge und Auslassungen; neue Klassen hängen hinten an. Ganz ohne Vorjahr gilt die
  Regel: Abschlussklassen zuerst, dann Jahrgang absteigend, Oberstufe unten.

Das Kollegium sieht den gespeicherten Plan in _Mein Portal → LMF-Plan_, für alle gleich
und immer auf dem aktuellen Stand; _Als PDF_ liefert die gewohnte Liste (Wochentag, Datum,
Stunde, Klassen, Besonderheiten), getrennt nach Rückgabe und Ausgabe.

**Der Rückgabe-Termin einer Klasse ist die Frist ihrer Schulbücher.** Beim Speichern
folgen die offenen Schulbuch-Ausleihen der Klassen (die Meldung nennt die Zahl); neue
Ausleihen bekommen ihn gleich. Fällt eine Klasse aus dem Plan oder wird er verworfen, gehen
die Fristen an den allgemeinen Stichtag zurück (_Einstellungen → Ausleihe_, Vorgabe
31.07.). Nicht angefasst: gesperrte Schüler, mehrjährige Ausleihen, von Hand gesetzte
Fristen und Ausgabe-Pläne. (§2.3)

## Schülerdatei

- **Suche** über den ganzen Bestand (Name, Klasse, Barcode). Zeile anklicken → Akte.
- **Akte**: _Ausleihen & Historie_ und _Stammdaten & Adresse_. Dokumente: Ausweis drucken,
  Kontoauszug, Ersatzforderung (nur bei offenem Schaden), DSGVO-Auskunft.
- **Gebühren & Schäden**: offen / bezahlt; _Bezahlt_ bucht aus, _Stornieren_ verlangt einen
  Grund. (§14)
- **Sperren** verlangt eine Begründung — sie steht danach an der Theke.
- **Neuer Schüler** per Formular; klassenweise besser über den LUSD-Import
  (_Einstellungen → Schuljahreswechsel_).
- **Stapelaktionen**: Klasse markieren → Ausweise oder Etiketten für alle drucken.
- Reiter **Ehemalige / Archiv** (wer die Schule verlassen hat) und **Papierkorb**; endgültiges
  Löschen/Anonymisieren nur mit Namensbestätigung (DSGVO-Kette, §8).
- **Doppelter Datensatz?** (Recht „Schüler zusammenführen", ab Werk nur Admin; unten im Reiter _Stammdaten & Adresse_): Steht dieselbe
  Person zweimal in der Kartei — typisch nach einer Namensänderung in der LUSD, die der
  Export ohne Schüler-ID nicht wiedererkannt hat —, beide Datensätze zusammenführen. Es
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

## Abgänger

Abgänger sind die Abschlussklassen des laufenden Schuljahres — 9H (und das freiwillige 10. Hauptschuljahr), 10R und 13 —, also die Kinder, die zum Schuljahresende gehen. Sie
sind noch an der Schule und leihen bis zuletzt aus. Die Regel ist dieselbe wie bei der
Versetzung; eingestellt wird nichts, die Klasse weiß es.

Die Liste zeigt **vom 1. Mai bis 31. Juli** die Abgänger, die **noch Bücher haben** — zum
Einsammeln vor der Entlassung. Kontoauszüge drucken oder an die Klassenleitungen mailen;
wer alles zurückgegeben hat, verschwindet. Außerhalb dieser Zeit steht hier nur der
Hinweis mit den Daten.

Wer die Schule dann **verlassen** hat (Versetzung: Abschlussklasse; LUSD-Import: fehlt im
neuen Export), steht nicht mehr hier, sondern in der Schülerdatei unter _Ehemalige /
Archiv_ — mit offenen Büchern zusätzlich im Mahnwesen. Er bleibt bis zum Ende der
**Karenzzeit** (Vorgabe 90 Tage; _Einstellungen → Datenschutz & Sitzung_) als gesperrter
Datensatz erhalten — Zeit, eine falsche Zuordnung noch zu reparieren — und wird danach
automatisch anonymisiert. Bleibt ein Buch oder eine Forderung dauerhaft offen, meldet
_System → Betriebsbereitschaft_ nach einem Jahr „Ehemalige mit offenen Vorgängen": In der
Akte das Buch als Verlust melden, dann die Forderung bezahlt oder storniert buchen — danach
löscht das System von selbst. Der Buch-Barcode wird dabei nicht neu vergeben, die
Ausweisnummer des Kindes nach der Löschung schon. Die Karenz läuft ab dem **späteren**
Zeitpunkt: dem Abgang oder der letzten Rückgabe beziehungsweise Schadensregulierung. Wer
erst lange nach dem Abgang zurückgibt, hat damit trotzdem die volle Karenz. Die endgültige Löschung ab dem 30. Januar des Folgejahres trifft nur Datensätze, die die Karenz durchlaufen haben, also schon anonymisiert sind (§8).
(§8, [LUSD.md](LUSD.md) §4)

## Bestellwesen

Sechs Reiter: **Bestellbedarf** (Lernmittel unter der Bedarfsschwelle — automatisch; Titel in
den Warenkorb, Lieferant wählen, Bestellung geht als Mail mit Bestätigungs-Link raus) ·
**Wareneingang** (Positionen einbuchen → Etiketten) · **Bestellhistorie** (Detail, Status,
Händlerbestätigung) · **Berichte** (Monat/Jahr/Lieferant als PDF) · **Klassensatz-
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
Menü und Schnittstelle). Rollen: Admin, Mitarbeit, Helfer, Kollegium. (§12)

## Einstellungen

13 Kategorien, jede einzeln speicherbar (§17):

| Kategorie               | Wofür                                                                                                                        |
| ----------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| Schule                  | Name, Anschrift, Eigentumsvermerk auf Etiketten                                                                              |
| Ausleihe & Fristen      | Tage je Buch/Medium, Limit je Schüler, LMF-Stichtag, Ferien-Leseclub (festes Rückgabedatum über die Ferien)                  |
| Mahnwesen               | automatische Sperre: ab wie vielen überfälligen Medien, nach wie vielen Tagen                                                |
| Mahnwesen-Routing       | Klasse → Klassenleitung (Empfänger für Mahnlauf und Abgänger-Kontoauszüge)                                                   |
| Bestellwesen            | Bedarfswarnung, Bedarfsschwelle, Preise erfassen                                                                             |
| Lieferanten             | Händler, Kundennummern, genau ein Hauptlieferant                                                                             |
| Datenschutz & Sitzung   | Löschfristen, Abgänger-Karenzzeit, Theke leeren, Sperrbildschirm                                                             |
| Erreichbarkeit & Alarme | öffentliche Adresse (Basis für Bestätigungs-Link, Katalog, Monitor), Alarm-Empfänger                                         |
| Mail                    | Postausgang mit Verbindungstest, Mail-Vorlagen (Mahnung, Bestellung, Händler)                                                |
| LMF-Aktionen            | alle Lernmittel einer Klasse auf ein neues Datum verlängern                                                                  |
| Datenverwaltung         | Katalog-Import (Littera), Bestands-Import (Kombi-CSV), Cover-Synchronisation, Katalog-Export, Offline-Sicherungen einspielen |
| Betriebsbereitschaft    | Selbstprüfung: eingerichtet, aber nicht in Betrieb? (§15)                                                                    |

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

## Wenn etwas nicht geht

- **Bestellung geht ohne Link raus / Katalog-Adresse fehlt** → _Erreichbarkeit & Alarme_:
  öffentliche Adresse eintragen.
- **Mahnliste kommt bei niemandem an** → _Mahnwesen-Routing_: Klasse hat keine Lehrkraft.
- **Rote Meldung „Kein Backup"** → _Betriebsbereitschaft_ öffnen; dort steht je Punkt, was
  fehlt und wie es zu beheben ist.
- **Scanner tippt ins Leere** → einmal ins Scanfeld klicken; die Theke holt den Fokus nach
  jedem Scan selbst zurück.
- **Netz weg** → weiter scannen; die Theke merkt sich alles lokal und bucht nach, sobald der
  Server wieder da ist.
