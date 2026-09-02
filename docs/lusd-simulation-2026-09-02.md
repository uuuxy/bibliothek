# LUSD-Import: Simulation Schuljahreswechsel (02.09.2026)

Prüfung des LUSD-Imports mit echten Demo-Exporten der Schule und einem synthetischen
Bestand in Schulgröße, auf dem lokalen Stack (Container 8084, Datenbank vorher geleert).
Alle Zahlen stammen aus Abfragen gegen die Datenbank nach jedem Schritt, Namen wurden
nie ausgegeben. Skripte: Generatoren (`gen_jahr1.py`, `gen_jahr2.py`, `gen_jahr3.py`),
Import über die echte Oberfläche (Playwright, Einstellungen → Schuljahreswechsel),
Prüfskripte (`pruefe_jahr2.py`, `pruefe_jahr3.py`) mit Manifest-Abgleich.

## 0. Die echten Exporte

Zwei LUSD-Berichtsexporte (Blatt `RExcelExport`, 150 Spalten, eine Zeile je
Ansprechpartner, also mehrere Zeilen je Schüler):

| Datei | Zeilen | Schüler | Klassen | Zuordnungsstufe |
|---|---|---|---|---|
| SPH-PaedNet_SuS | 293 | 126 | 61 (05F1 … 13T5, ET, IKL) | Name + Geburtsdatum |
| All_Inklusiv | 167 | 31 | 26 | Name + Geburtsdatum |

- Keine LUSD-ID im Export. `Schueler_Schluessel` ist `Nachname,Vorname,TT.MM.JJJJ` und
  bewusst **kein** ID-Alias (Test `TestLusdHeaderMap_AllInklusivBericht`).
- Klasse kommt aus `Klassen_Klassenbezeichnung`, in der festen Schreibweise (05F1).
- Gefixt: `Schueler_Postleitzahl` hatte keinen Alias, die PLZ fiel still weg (fdfd09d8).
- Offen: `Ansprechpartner_Alle_Email` wird nicht gelesen (siehe Empfehlung unten).

Import SPH (126): 126 neu, alle mit Geburtsdatum und Barcode. Danach Vorschau
All_Inklusiv: 30 neu, 1 zusammengeführt, 125 Abgänger → Massenabgang-Bremse greift.

## 1. Jahr 1: Erstimport in Schulgröße

Datei `jahr1.xlsx`: 3.809 Zeilen, **1.890 Schüler** (126 echte Demo-Schüler + 1.764
synthetische), 80 Klassen. Namenspools deutsch, türkisch, arabisch (Al-Sayed / Al Sayed /
Alsayed), chinesisch, vietnamesisch (Nguyễn), polnisch (Wiśniewski), mit Umlauten, ß,
Bindestrich, Apostroph (O'Brien), Adelspartikeln, Doppelvornamen. Fallen: 20 Namensdoppel
(gleicher Name, anderes Geburtsdatum), 10 Schreibvarianten (Müller/Mueller), 10 Zeilen
mit Leerzeichen und falscher Groß-/Kleinschreibung.

| Prüfung | Ergebnis |
|---|---|
| Schüler in der DB | 1.890, Manifest-Abgleich: 0 fehlen, 0 zu viel, 0 falsche Klasse |
| Dubletten (Name + Geburtsdatum) | 0 |
| Namensdoppel / Schreibvarianten | je eigener Datensatz, keine Vermischung |
| Leerzeichen an Namen | 0 (Importer trimmt) |
| Groß-/Kleinschreibung | wird **nicht** korrigiert, Namen kommen wie in LUSD geschrieben an |
| Barcode, Geburtsdatum, `lusd_bestaetigt_am` | bei allen 1.890 gesetzt |
| Dauer des Imports | unter 2 Sekunden |
| Audit (`audit_logs`, LUSD_IMPORT) | Eintrag mit Zählern, ohne Namen |

Danach per SQL 6.000 Simulations-Exemplare (`SIM-B-…`) angelegt und **jedem Schüler drei
Bücher** ausgeliehen: 5.670 offene Ausleihen.

## 2. Jahr 2: Schuljahreswechsel

Datei `jahr2.xlsx`: 1.939 Schüler. Alle Klassen eine Stufe höher (Förderstufe endet in 7,
Jahrgang 10 geht in 11T), 30 Zweigwechsel G→R, 202 Abgänge (Jahrgang 13 komplett, 40 %
der 10H/10R, 2 % Wegzug), 245 Neue (230 Fünftklässler, 15 Quereinsteiger), 6 Neue mit
dem Namen eines Bestandsschülers und anderem Geburtsdatum. Bewusste Grenzfälle:
8 Nachnamensänderungen, 5 Geburtsdatum-Korrekturen.

Vorschau: 264 neu, 1.587 Klassenwechsel, 215 Abgänger (11 % → unter der 30-%-Bremse).

| Prüfung | Ergebnis |
|---|---|
| Dubletten | 0 |
| Bleibende (ET/IKL, 88) | alle da, Klasse unverändert, gleiche ID, nicht Abgänger |
| Klassenwechsel (1.587 inkl. Zweigwechsel) | Klasse in allen Fällen wie in der Datei, gleiche ID und Barcode |
| Abgänger (202) | markiert (`ist_abgaenger`, `abgaenger_jahr` 2026), nicht gelöscht, gesperrt mit Grund „Automatisierte Abgänger-Sperre (offene Vorgänge)" — Namen bleiben, weil Bücher offen |
| Neue (251) | alle angelegt |
| Namensgleiche Neue (6) | eigener Datensatz, Bestandsschüler unangetastet |
| Offene Ausleihen | weiterhin 5.670, alle an existierende Schüler; 645 davon bei Abgängern |
| Barcodes der Bestandsschüler | 0 geändert |
| `lusd_bestaetigt_am` | bei allen 1.939 Aktiven auf heute |

**Grenze ohne LUSD-ID (erwartet, bestätigt):** Wer im Export einen anderen Nachnamen oder
ein korrigiertes Geburtsdatum bekommt, ist über Name + Geburtsdatum nicht mehr derselbe:
alter Datensatz wird Abgänger (hier gesperrt, da Bücher offen), neuer Datensatz wird
angelegt. 8 + 5 = 13 solche Doppel entstanden, genau wie vorausgesagt.

## 3. Jahr 3: Anonymisierung, Sperre, Rückkehrer, Mehrdeutigkeit

Datei `jahr3.xlsx`: wie Jahr 2, aber 10 Schüler fehlen, die vorher alles zurückgegeben
haben; 10 fehlen mit offenen Büchern; 5 Abgänger aus Jahr 2 stehen wieder drin; zwei
Zeilen mit identischem Namen **und** Geburtsdatum in zwei Klassen.

| Prüfung | Ergebnis |
|---|---|
| Abgänger ohne offene Bücher (10) | sofort anonymisiert: Vorname „Abgänger", Nachname „Anonymisiert-…", `anonymized_at`, Adresse und Eltern-Mail geleert; Datensatz bleibt |
| Abgänger mit offenen Büchern (10) | Name lesbar, gesperrt mit Grund |
| Rückkehrer (5) | reaktiviert, Klasse aus der Datei, dieselbe ID; **bleiben gesperrt** mit Grund „Sperre wegen offener Vorgänge", weil ihre drei Bücher noch offen sind (so vorgesehen) |
| Zwei Zeilen, gleicher Name + Geburtsdatum | werden als **eine** Person zusammengelegt (Vorschau „1 neu", nicht „mehrdeutig"), Klasse der letzten Zeile |
| Dubletten gesamt | 0; offene Ausleihen 5.640, alle gültig |
| Aktive / Abgänger / gesamt | 1.925 / 230 / 2.155, rechnerisch stimmig |
| Oberfläche: Abgänger-Seite | 220 Abgänger mit offenen Büchern, Sperre sichtbar; Anonymisierte (0 Bücher) erscheinen nicht |
| Oberfläche: Schülerdatei | Jahr-1-Schüler mit Klasse 08G1 und 3 Büchern |

## Befunde und Empfehlungen

1. **LUSD-ID in den Bericht aufnehmen.** Alles, was in dieser Simulation nicht perfekt
   war, hat dieselbe Ursache: der Export trägt keine Schüler-ID. Mit ID gäbe es keine
   Doppel bei Namensänderung oder Datumskorrektur, und zwei Kinder mit gleichem Namen
   und Geburtsdatum würden nicht zusammengelegt. Der Importer kann ID-Modus bereits.
   Bitte im LUSD-Berichtsdesigner die Schüler-ID (Feld z. B. `Schueler_ID`) ergänzen.
2. **Zusammenlegen gleicher Name + Geburtsdatum in einer Datei:** in echten Exporten
   nötig (mehrere Ansprechpartner-Zeilen), für zwei verschiedene Kinder mit identischem
   Namen und Geburtsdatum aber falsch. Realistisch extrem selten; die Vorschau sollte den
   Fall trotzdem als „mehrdeutig" melden, wenn die Klassen der Zeilen verschieden sind.
3. **Abgänger ohne offene Bücher werden beim Import sofort und unumkehrbar anonymisiert.**
   Das ist DSGVO-konform gewollt, aber: Ein Schüler, der wegen Namensänderung als Abgänger
   erkannt wird und keine Bücher hat, verliert sofort seinen Namen. Die Vorschau zeigt die
   Abgänger-Liste vorher, die Bibliothek muss sie bei Namensänderungen ansehen. Mit LUSD-ID
   entfällt das Risiko.
4. **Eltern-E-Mail:** derzeit nur in der DSGVO-Auskunft verwendet, keine Mahnung geht an
   Eltern. Empfehlung: nicht importieren (Datensparsamkeit), solange kein Prozess sie
   braucht. Falls doch: nur Kontakte mit Art Mutter/Vater/Eltern/Pflege/Vormund, erster
   Treffer in Dateireihenfolge, nie Großeltern oder „Sonstige Kontaktperson".
5. **Groß-/Kleinschreibung** wird nicht korrigiert; das ist richtig, Namen gehören LUSD.
