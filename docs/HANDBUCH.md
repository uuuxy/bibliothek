# Benutzerhandbuch

Stand: 2026-09-03. Für Bibliothekspersonal, Sekretariat und Schulleitung — geschrieben aus
Sicht der Arbeit am Tresen, nicht aus Sicht des Codes. Die fachlichen Regeln dahinter stehen
im [Fachkonzept](FACHKONZEPT.md); dort verweisen die §-Angaben hin. Ein Produktrundgang als
Video (8 min) zeigt jeden Bereich in Aktion.

**Zwei Grundsätze, die überall gelten:**
- Es gibt **keine Passwörter** zu verwalten. Angemeldet wird mit dem Schul-Postfach
  (E-Mail + Mail-Passwort). Wer kein Konto hat, kann sich damit selbst anmelden und wird von
  der Bibliothek freigeschaltet (→ Benutzer & Rechte).
- **Jede Rolle sieht nur ihren Teil.** Lehrkräfte sehen nur *Mein Portal*, Helfer nur die
  Theke ohne Schülerakten, Mitarbeitende den Tresenbetrieb, Admins alles.

---

## Öffentliche Seiten — ohne Anmeldung

| Adresse | Für wen | Zeigt |
|---|---|---|
| `https://<schule>/katalog` | Schüler, Eltern, Kollegium — vom Handy, aus dem Klassenraum | Suche nach Titel, Autor, ISBN; Cover; „N von M verfügbar". Keine Ausleihdaten, keine Namen. |
| `https://<schule>/monitor` | der Bildschirm vor der Bibliothek | Endlos-Slideshow: Buch des Monats (die meisten Schüler-Leser in 30 Tagen), Neu eingetroffen, Beliebt diese Woche (7 Tage). Gezählt werden Leser, nicht Exemplare — Klassensätze an Lehrkräfte zählen nicht. Keine Schulbücher (Lernmittel), nur Titel mit einem Exemplar im Haus — dieselbe Regel wie im Katalog. Buch des Monats und Neu eingetroffen nur mit Cover. Aktualisiert sich alle 5 Minuten von selbst; ist der Server beim Einschalten noch nicht da, versucht die Seite es alle 30 s erneut. Folien ohne Inhalt (Ferien) werden übersprungen. Um 03:00 Uhr lädt sich die Seite selbst neu und holt so neue Versionen — der Bildschirm braucht keine Tastatur. |

Beide Seiten haben keinen Menüpunkt. Die fertigen Adressen stehen unter
*Einstellungen → Erreichbarkeit & Alarme* zum Kopieren. (§16)

---

## Überall suchen (seit 03.09.2026)

Oben auf jeder Verwaltungsseite steht **eine** Suchleiste. Sie versteht alles, was die Theke
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
beides einstellbar (*Datenschutz & Sitzung*).

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

Sachgruppen (Kürzel + Bezeichnung, z. B. *Jug* – Jugendliteratur) pflegen und Regale per
Präfixsuche durchsehen („Jug" findet „Jug Her", „Jug Pre" …). (§13)

## Druck-Center

Vier Reiter: **Buch-Etiketten** (Format, Startposition auf dem Bogen, Titel aus dem Katalog
oder aus einer Klasse) · **Fehlende Etiketten** (alle Exemplare, die noch keins haben — die
Zahl steht als Badge in der Navigation) · **Schülerausweise** (Designer für Vorder- und
Rückseite) · **Klassenweise drucken** (Ausweise für eine ganze Klasse). Der Druck läuft
über den Browser-Druckdialog. (§9)

## Klassensätze

Welche Klasse hat welche Lektüre? Klasse suchen, *Klasse hinzufügen* öffnet den Dialog:
Bücher auswählen, Zielklasse eintragen, speichern. Reservierungen aus dem Kollegium stehen
im Bestellwesen (→ *Klassensatz-Reservierungen*). (§4)

---

## Schülerdatei

- **Suche** über den ganzen Bestand (Name, Klasse, Barcode). Zeile anklicken → Akte.
- **Akte**: *Ausleihen & Historie* und *Stammdaten & Adresse*. Dokumente: Ausweis drucken,
  Kontoauszug, Ersatzforderung (nur bei offenem Schaden), DSGVO-Auskunft.
- **Gebühren & Schäden**: offen / bezahlt; *Bezahlt* bucht aus, *Stornieren* verlangt einen
  Grund. (§14)
- **Sperren** verlangt eine Begründung — sie steht danach an der Theke.
- **Neuer Schüler** per Formular; klassenweise besser über den LUSD-Import
  (*Einstellungen → Schuljahreswechsel*).
- **Stapelaktionen**: Klasse markieren → Ausweise oder Etiketten für alle drucken.
- Reiter **Abgänger / Archiv** und **Papierkorb**; endgültiges Löschen/Anonymisieren nur mit
  Namensbestätigung (DSGVO-Kette, §8).
- **Doppelter Datensatz?** (Recht „Schüler zusammenführen", ab Werk nur Admin; unten im Reiter *Stammdaten & Adresse*): Steht dieselbe
  Person zweimal in der Kartei — typisch nach einer Namensänderung in der LUSD, die der
  Export ohne Schüler-ID nicht wiedererkannt hat —, beide Datensätze zusammenführen. Es
  bleibt der Datensatz, dessen Ausweis das Kind in der Hand hat; Ausleihen, Gebühren und
  Historie des anderen wandern hinüber, vom Foto bleibt das jüngere. Die Suche im Dialog
  findet auch Abgänger und Gesperrte. **Wurde für den aufgelösten Datensatz schon ein
  Ausweis gedruckt, diese zweite Karte einziehen und vernichten:** Ihre Nummer ist danach
  frei und kann beim nächsten neu angelegten Schüler wieder vergeben werden.
  ([LUSD.md](LUSD.md) §5)

## Mahnwesen

Register **Alle · Akut fällig (bis 14 Tage) · Eskaliert**, Filter nach Klasse. Mahnbriefe an
Eltern oder für eine ganze Klasse drucken; **Sammel-Mahnlauf** per Mail an die Klassenleitungen
(Klassen wählen, Empfänger prüfen, dann senden). Die Mahnstufe steigt beim **Druck** des
Mahnbriefs, nicht beim Mailversand. Lehrkräfte werden nie angemahnt. Welche Klasse an welche
Lehrkraft geht, steht unter *Einstellungen → Mahnwesen-Routing*. (§3)

## Abgänger

Zeigt nur Abgänger, die **noch Bücher haben**. Kontoauszüge drucken oder an die
Klassenleitungen mailen. Wer alles zurückgegeben hat, verschwindet aus der Liste, bleibt
bis zum Ende der **Karenzzeit** (Vorgabe 90 Tage **ab dem Abgang**, nicht ab der Rückgabe;
*Einstellungen → Datenschutz & Sitzung*) als gesperrter Datensatz erhalten — Zeit, eine
falsche Zuordnung noch zu reparieren — und wird danach automatisch anonymisiert. Wer erst
nach Ablauf der Karenz zurückgibt, wird in der folgenden Nacht anonymisiert.
(§8, [LUSD.md](LUSD.md) §4)

## Bestellwesen

Sechs Reiter: **Bestellbedarf** (Lernmittel unter der Bedarfsschwelle — automatisch; Titel in
den Warenkorb, Lieferant wählen, Bestellung geht als Mail mit Bestätigungs-Link raus) ·
**Wareneingang** (Positionen einbuchen → Etiketten) · **Bestellhistorie** (Detail, Status,
Händlerbestätigung) · **Berichte** (Monat/Jahr/Lieferant als PDF) · **Klassensatz-
Reservierungen** (Warteschlange aus dem Kollegium; *Abschließen* schickt die Bereit-Mail) ·
**Wünsche & Meldungen** (Anliegen aus dem Portal, *Erledigen* mit Notiz an die Lehrkraft).
Lieferanten und Hauptlieferant stehen in den Einstellungen. (§7)

## Inventur

*Neue Bestandsprüfung starten* → Umfang wählen (komplett, eine Signatur, Fach/Klasse) →
scannen; Fortschrittsbalken. **Achtung:** *Inventur abschließen* bucht alles Ungescannte im
Umfang als Verlust — vorher den Fehlbestandsbericht prüfen; dort lassen sich Funde wieder
zurückholen. Laufende Inventuren können fortgesetzt oder verworfen werden. (§6)

---

## Statistiken

Bestand, aktuell verliehen, Zirkulationsquote, Wiederbeschaffungswert; Ausleihen pro Monat;
Überfällige nach Dauer; **Renner** (meistausgeliehen) und **Ladenhüter** (seit über zwei Jahren
nicht ausgeliehen) mit Detailseite und Filter. Ohne Schülernamen — die Statistik zählt
Ausleihen, nicht Personen. (§11)

## System-Logs

*Allgemeines Logbuch* (jede Buchung) und *Admin-Audit-Log* (wer hat wann was geändert).
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

| Kategorie | Wofür |
|---|---|
| Schule | Name, Anschrift, Eigentumsvermerk auf Etiketten |
| Ausleihe & Fristen | Tage je Buch/Medium, Limit je Schüler, LMF-Stichtag, Ferien-Leseclub (festes Rückgabedatum über die Ferien) |
| Mahnwesen | automatische Sperre: ab wie vielen überfälligen Medien, nach wie vielen Tagen |
| Mahnwesen-Routing | Klasse → Klassenleitung (Empfänger für Mahnlauf und Abgänger-Kontoauszüge) |
| Bestellwesen | Bedarfswarnung, Bedarfsschwelle, Preise erfassen |
| Lieferanten | Händler, Kundennummern, genau ein Hauptlieferant |
| Datenschutz & Sitzung | Löschfristen, Abgänger-Karenzzeit, Theke leeren, Sperrbildschirm |
| Erreichbarkeit & Alarme | öffentliche Adresse (Basis für Bestätigungs-Link, Katalog, Monitor), Alarm-Empfänger |
| Mail | Postausgang mit Verbindungstest, Mail-Vorlagen (Mahnung, Bestellung, Händler) |
| LMF-Aktionen | alle Lernmittel einer Klasse auf ein neues Datum verlängern |
| Datenverwaltung | Katalog-Import (Littera), Bestands-Import (Kombi-CSV), Cover-Synchronisation, Katalog-Export, Offline-Sicherungen einspielen |
| Schuljahreswechsel | LUSD-Abgleich (mit Vorschlägen für umbenannte Schüler), Versetzung mit Vorschau — Anleitung: [LUSD.md](LUSD.md) |
| Betriebsbereitschaft | Selbstprüfung: eingerichtet, aber nicht in Betrieb? (§15) |

## Mein Portal (Kollegium)

Lehrkräfte sehen genau diesen Bereich: **Suchen & Reservieren** (Bestand mit Verfügbarkeit
und Warteschlange; Klassensatz reservieren mit Klasse, Anzahl, Datum) · **Klassensätze** der
eigenen Klassen · **Schulbücher** (seit 03.09.2026, für die Fachsprecher: aufgebaut wie die
Klassensätze, nur nach Fach statt Klasse. Oben Suche (Titel, ISBN, Autor, Fach) und die
Filter Jahrgang und Schulzweig; darunter je Fach eine Zeile mit Exemplaren, Titeln und
Verliehenen, die sich zu den Cover-Kacheln der Bücher aufklappt. „Als Excel" sitzt an
jedem Fach und exportiert genau dieses Fach mit der aktuellen Filterung (Fach, Titel,
Autor, ISBN, Jahrgang, Schulzweig, Zahlen, Cover-Link). Gezählt wird nur, was den
Lernmittel-Schalter trägt. Den **Schulzweig** pflegt die Bibliothek am Buch: In der
Buchmaske erscheint das Feld, sobald „Lernmittel" eingeschaltet ist; leer heißt „gilt für
alle Zweige". Littera hat den Zweig nie mitgeliefert, der Altbestand ist also zunächst
ohne. Tauchen dort Standorttexte wie „Buch Deu 6/Cha 126" als Fach auf,
stammt der Bestand aus einem Import vor dem 03.09.2026: `scripts/repair_fach_kategorie.sql`) · **Meine Anliegen** (Buchwunsch oder „Etwas stimmt
nicht" an die Bibliothek). (§12, Rolle Kollegium)

---

## Wenn etwas nicht geht

- **Bestellung geht ohne Link raus / Katalog-Adresse fehlt** → *Erreichbarkeit & Alarme*:
  öffentliche Adresse eintragen.
- **Mahnliste kommt bei niemandem an** → *Mahnwesen-Routing*: Klasse hat keine Lehrkraft.
- **Rote Meldung „Kein Backup"** → *Betriebsbereitschaft* öffnen; dort steht je Punkt, was
  fehlt und wie es zu beheben ist.
- **Scanner tippt ins Leere** → einmal ins Scanfeld klicken; die Theke holt den Fokus nach
  jedem Scan selbst zurück.
- **Netz weg** → weiter scannen; die Theke merkt sich alles lokal und bucht nach, sobald der
  Server wieder da ist.
