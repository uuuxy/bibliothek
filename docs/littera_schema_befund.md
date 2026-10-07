# Littera-Altbestand: Schema-Befund

> Erhoben am 04.08.2026 direkt aus den vom Sekretariat kopierten Access-Dateien
> (`~/Desktop/Neuer Ordner`) mit `mdbtools`. Alle Zahlen sind gemessen, nicht geschätzt.
> Damit ist der Fahrplanpunkt „Migrations-Tool auf echtes Littera-Schema" nicht mehr blockiert.

## Was tatsächlich vorliegt

Der Fahrplan sprach von einem **MySQL-Dump**. Geliefert wurde **MS Access**:

| Datei | Inhalt |
|---|---|
| `littera_sav.mdb` (139 MB) | **Die einzige Datei mit Daten.** 191 Tabellen |
| `littera.mdb` (6 MB) | Nur Schema, alle Kerntabellen leer |
| `littera_bak.mdb` (4,6 MB) | Leer |
| `ressourcen.mdb` (5 MB) | Ressourcen/Stammwerte |
| `LUSD-XML.xml` | **PGP-verschlüsselt**, kein XML — so nicht importierbar |

**Das Programm der Schule** ist LITTERA Windows (5.4), das Programm des Herstellers für
Bibliotheken; darauf bezieht sich auch das Handbuch. Eigens für Lernmittel gebaut ist es nicht.
Die Schule führt die Lernmittel darin als Behelf mit: über die Signatur „LMF …" (in der
Sicherung von 2010 beginnt `Sig1` bei 24.730 von 61.520 Exemplaren so, gezählt am 07.10.2026)
und über zwei Verrechnungsgruppen, `LMF` und `LMF-Oberstufe`. Das Handbuch nennt dazu einen
Schalter „Lernmittelfreiheit verwenden", der eine Verrechnungsgruppe mit dem Namen LMF anlegt.
Das Lernmittel-Programm des Herstellers (LITTERA LM) ist ein anderes; das Handbuch nennt es nur
dort, wo beide Programme Leserdaten und Ausweise teilen. Was es über Verleih, Gebühren und
Löschen sagt, gilt für die Bibliothek.

### Datenstand — wichtig

| Tabelle | Datensätze |
|---|---|
| `Titel` | 10.732 |
| `Exemplar` | 61.520 |
| `Leser` | 1.991 |
| `Verleih` | 15.615 (davon 15.614 ohne Rückgabe gebucht) |

**Der Bestand endet 2010** — und das ist inzwischen nicht mehr nur wahrscheinlich,
sondern belegt:

* Letzte Zugangsdaten: 2009 (4.029), 2010 (2.996), danach nichts.
* Verleihdaten: 12.329 der 15.615 aus 2010, die Fristen laufen bis 2011 aus.
* 1.723 der 1.991 Leser wurden 2010 zuletzt bewegt.
* **Die Geburtsjahre der Schüler reichen von 1989 bis 2001.** Diese Menschen sind heute
  25 bis 37 Jahre alt.

Ein Import der Personen und Ausleihen aus dieser Datei legt also Schüler an, die vor
fünfzehn Jahren die Schule verlassen haben, und meldet ein Viertel des Bestands als
verliehen. Deshalb sind `-personen` und `-ausleihen` in `cmd/littera-altbestand`
standardmäßig **aus** — der Bestand allein ist unbedenklich, die Bücher stehen im Regal.
Für einen aktuellen Export aus einer noch laufenden Littera-Installation sind beide
Schalter richtig; der Schreibpfad ist dafür gebaut und geprüft.

## Feldzuordnung (Kerntabellen)

### `Titel` → `buecher_titel`
`Buchungsnummer` (Schlüssel) · `Haupttitel` → `titel` · `Untertitel` → `untertitel` ·
`ISBN` → `isbn` · `Erscheinungsjahr` (Text!) · `Auflage` → `auflage` ·
`Medienart` (Long → Nachschlagetabelle) ·
`Verlag` (Long → Nachschlagetabelle, **kein Freitext**) · `Annotation` (nicht übernommen)

**Auflage** (Text 50): Litteras Feld, wie es dasteht. In der Sicherung von 2010 tragen 3.182
von 10.732 Titeln eine Angabe, im Katalog-Export vom Juni 2026 (MAB 403) 6.153 von 14.858;
keine ist länger als 50 Zeichen. Die Bücherei führt darin dreierlei, nach der Form gezählt am
07.10.2026: die Auflagenbezeichnung („2. Aufl.", „Sonderausg."; 2.512 Titel, im Juni 2026
5.033), bei Zeitschriften die Heftnummer („34 / 2010"; 199, im Juni 2026 955) und eine eigene
Nummer der Form „D-Ga-066" (432 Titel mit 11.278 Exemplaren, im Juni 2026 69); dazu 39 und 96
andere Angaben („compact disc", „Windows 95"). Die Übernahme ändert den Wortlaut nicht. Sie
nimmt Leerraum am Rand weg (in der Sicherung von 2010 23 Werte), macht aus Leerraum in Folge
ein Leerzeichen (4 Werte) und kürzt auf die Spaltenbreite von 50 Zeichen, mit Vermerk im
Protokoll.

Titel, Untertitel und Verfasser kommen ohne Litteras Nichtsortierzeichen: Aus „¬Die¬ schwarze
Katze" wird „Die schwarze Katze" (`littera.OhneNichtsortierzeichen`, dieselbe Regel im
Katalog-Import). In der Sicherung von 2010 tragen es 1.403 von 10.732 Titeln und 3 von 7.364
Verfassern, die übrigen Tabellen der Übernahme nicht. Litteras Sortiertitel (`HaupttitelSort`)
kommt nicht mit: Die Titelliste ordnet nach dem Titel, wie er dasteht.

Leerraum in Folge liest die Übernahme mit, wie er in Littera steht; die Datenbank speichert
Titel, Untertitel, Autor und Verlag mit einem Leerzeichen zwischen den Wörtern
(`titeltext_normalform`, Migration 160, an jeder Tür). In der Sicherung von 2010 tragen 103
Titel, 73 Untertitel, 23 Verfasser und 2 Verlage zwei bis vier Leerzeichen in Folge
(„La  Peste"), 11 Titel ein geschütztes Leerzeichen hinter „/" oder „:".

### `Exemplar` → `buecher_exemplare`
`Buchungsnummer` (Schlüssel) · `Titel` (FK) · `Barcode` → `barcode_id` ·
`Zugangsdatum` → `erworben_am` · `Preis` · `Status` (Long) · `Sig1` + `Sig2` (siehe unten) ·
`Eigentumsvermerk` → `eigentum` (siehe unten)

**Eigentumsvermerk** (Text 50, Wertehilfe): Littera führt hier, wem das Exemplar gehört. In der
Medienliste vom 12.06.2026 tragen 16.878 von 67.109 Exemplaren einen von acht Werten, die
übrigen keinen. Übernommen wird nur die feste Liste in `internal/littera/eigentum.go`
(entschieden am 29.09.2026): „Land Hessen" setzt `eigentum = 'land'`, der Schulträger
`'schultraeger'`; fünf weitere Werte (Schule, Bibliothek, Förderverein, Info Schulprojekt,
Dauerleihgabe) kommen nur als Wortlaut mit, bis die Schule sie zuordnet. Der Wortlaut steht in
`erweiterte_eigenschaften` unter `littera_eigentumsvermerk`. Ein Wert außerhalb der Liste kommt
nicht mit und steht im Protokoll mit Exemplarnummer, ohne Wortlaut — in der Sicherung von 2010
ist einer davon ein Personenname. Am meisten bewirkt der erste Wert: 11.160 Exemplare mit
„Land Hessen" haben keine LMF-Signatur, die Faustregel ordnete sie dem Schulträger zu
(`repository.ExemplarTopfSQL`, docs/OFFEN.md 4.24).

### `Leser` → `schueler` — hier ist Vorsicht nötig

`Lesernummer` · `Vorname` · `Nachname` · `Geburtsdatum` · `Anmeldedatum` · Adressfelder
(**DSGVO: nur übernehmen, was gebraucht wird**)

**Die Klasse steht nicht am Leser.** Sie kommt über `Leser.Lesergruppe` →
`Leser_UG.KurzBez` (`10G4`, `06G3`, `05F1` = Jahrgang + Zweig + Zug); `Leser_UG.Obergruppe`
→ `Leser_OG.Obergruppe` liefert den Schulzweig (Förderstufe, Hauptschul-, Realschul-,
Gymnasialzweig). Auflösbar für **1.991 von 1.991** Lesern.

Die naheliegende Quelle `LeserSchueler` (705 Zeilen mit `Jahrgang`, `Abgang`,
`Klassenlehrer`) ist **komplett leer** — alle Werte 0. Nicht darauf bauen.

**Die Weiche steht in den Daten, sie muss nicht geraten werden:**
`Leser_UG.Untergruppe` unterscheidet die Personengruppen. Umgesetzt in `LeserArt`:

| Untergruppe | Art | Ziel |
|---|---|---|
| `Schüler`, **`Sekundarstufe II`**, `Im Ausland` | `ArtSchueler` | `schueler` |
| `Lehrer`, `Lehrerin` | `ArtLehrkraft` | `benutzer` mit Rolle `kollegium` (bis Migration 069: `lehrer`) |
| `Referendar`, `Referendarin` | `ArtLiV` | Kollegium, Personenart `liv` |
| `Abgegangen` | `ArtAbgegangen` | `schueler` mit `ist_abgaenger` |
| `Praktikant`, `Praktikantin` | `ArtPraktikum` | Kollegium, Art `praktikum`, ohne Konto; Ausleihen als Dauerleihe |
| `Sekretärin` | `ArtSekretariat` | Kollegium, Art `sekretariat`, mit Konto; Ausleihen als Dauerleihe |
| `U-plus` | `ArtUPlus` | Kollegium, Art `uplus`, mit Konto; Ausleihen als Dauerleihe |
| `Fachbereich …` | `ArtFachbereich` | Kollegium, Art `fachbereich`, ohne Konto; Ausleihen als Dauerleihe |
| alles andere (`Undefinierte Untergruppe`, `IMPORT`) | `ArtUnbekannt` | **Lauf hält an, bevor er schreibt** |

`Sekundarstufe II` ist der Fall, den man leicht übersieht: eine eigene Untergruppe,
aber die Oberstufenklassen (11T1, 12T3, 13T5) sind selbstverständlich Schüler.
`Im Ausland` kommt mit der Littera-Klasse `AUS`; die richtige Klasse trägt der LUSD-Import
nach. Aus `AUS` lässt sich kein Abgangsjahr ablesen, es gilt das Jahr, das die Anwendung für
jede Klasse ohne Jahrgang einsetzt (`repository.AbgangsjahrOhneKlasse`), mit einer Warnung im
Protokoll. `U-plus` sind Vertretungskräfte. Die Littera-Gruppe steht seit Migration 153 als
Art an der Leserzeile; bis dahin kamen die Sonderkonten als `lehrkraft` an, und ihre Gruppe
stand nur im Protokoll des Laufs.

Gemessen am Altbestand: **1.734 Schüler (davon 14 im Ausland) · 158 Lehrkräfte · 71
abgegangen · 23 Sonderkonten (9 Praktikum, 9 Fachbereich, 3 U-plus, 2 Sekretariat) · 5 ohne
Zuordnung** — und jeder Schüler hat eine Klasse
(`schueler.klasse` ist NOT NULL, ein Test sichert das ab). Eine Gruppe ohne Zuordnung wird
**nicht** geraten und nicht ausgelassen: Ausgelassen fehlten ihre Ausleihen, und die Bücher
stünden als verfügbar im Regal. Der Lauf nennt Gruppe, Personen- und Ausleihzahl und hält
an, im Trockenlauf wie im echten Lauf (`OhneZuordnung`, Entscheidung vom 28.09.2026);
zugeordnet wird in Littera selbst oder in `artAusUntergruppe`.

Feldbelegung, gemessen: Nachname 1.991 · Vorname 1.980 · Geburtsdatum 1.924 ·
Adresse 1.927 · Anmeldedatum 834 · **eMail nur 3** · Abmeldedatum 0.
Die Mahnung per E-Mail hat aus dieser Quelle also praktisch keine Grundlage —
`eltern_email` muss aus LUSD kommen, nicht aus Littera.

**Lesernummern und Exemplarnummern** zählt Littera getrennt, beide ab 1: In der Sicherung von
2010 ist jede der 1.991 Lesernummern zugleich eine Exemplarnummer (gezählt am 07.10.2026).
Littera unterscheidet an der Maske, in die gescannt wird, nicht an der Zahl. Frei gewordene
Nummern kann es wieder vergeben (Einstellung „freie Nummern wieder vergeben"); eine
Lesernummer kann also früher einer anderen Person gehört haben. Hier kommt eine A-Nummer nicht
wieder (Migration 146), und Ausweis und Buch unterscheidet die Form des Scanwerts
([FACHKONZEPT.md](FACHKONZEPT.md) §1).

### `Verleih` → `ausleihen`
`Exemplar` (FK) · `Leser` (FK) · `Verleihdatum` · `Rückgabedatum` (Frist) ·
`IstRückgabedatum` (tatsächlich) · `Zurückgegeben` (Boolean) · `Mahnungen`

`Mahnungen` kommt als Mahnstufe der Ausleihe mit. In der Sicherung von 2010 steht sie bei allen
15.615 Ausleihen auf 0, und `LetzteMahnung` ist nie gesetzt (gezählt am 07.10.2026): Ob die
Bücherei in Littera mahnt, zeigt erst die Sicherung von 2026.

## Die Signatur — betrifft die Umstellung von Migration 060

Littera führt die Signatur **am Exemplar, nicht am Titel**, und **zweiteilig**:

* `Exemplar.Sig1` — die Regaladresse: `LMF Bio 11`, `LMF Deu 6`, `Ea`, `Ga`
* `Exemplar.Sig2` — das Kürzel des Titels darin: `Lin`, `Nat`, `Bos`, `Bär`

Zusammen ergibt das die Aufschrift vom Buchrücken: `LMF Deu 7 / Bie`.

**Belegung: 61.516 von 61.520 Exemplaren haben `Sig1`** — praktisch vollständig.

Zwei Folgerungen:

1. **Die Präfix-Auslegung aus Migration 060 passt zu den echten Daten.** `Sig1` ist
   hierarchisch am Leerzeichen aufgebaut (`LMF` → `LMF Bio` → `LMF Bio 11`), genau die
   Grenze, nach der `repository.SignaturPraefixBedingung` schneidet.

2. **Unser `buecher_titel.signatur` liegt am Titel — Littera am Exemplar.** Gemessen
   über den fertigen Lesepfad: Bei **72 von 10.422** Titeln mit Signatur (0,7 %)
   unterscheiden sich die Exemplar-Signaturen untereinander. (Eine frühere Rohzählung
   ergab 98 — sie zählte den Platzhalter `0` als eigenen Wert mit; `SignaturAus`
   verwirft ihn.) Für 99,3 % ist die Titel-Ebene verlustfrei; für die übrigen nimmt
   `SignaturJeTitel` den häufigsten Wert und meldet den Titel als abweichend, statt
   still den ersten zu nehmen.

**Jedes Exemplar behält seine eigene Signatur:** Sie steht in `erweiterte_eigenschaften` unter
`littera_signatur`, auch wo der Titel die häufigste bekommen hat.

**Was am Titel ankommt, hängt vom Weg ab** (gezählt am 07.10.2026). Die Übernahme aus einer
Sicherung schreibt beide Teile, die Aufschrift des Buchrückens („LMF Deu 7 / Bie"). Der
Katalog-Import ([SCRIPTS.md](SCRIPTS.md), Abschnitt 1a) liest aus dem MAB-Export nur das erste
Teil (Feld 700, Reihung 1), die Regaladresse. In der Sicherung von 2010 fehlt das zweite Teil
nur bei elf Exemplaren mit Signatur, bei den Büchern der Schülerbücherei so wenig wie bei den
Lernmitteln. An den 10.422 Titeln stehen nach der Übernahme 6.251 verschiedene Aufschriften mit
732 verschiedenen Regaladressen; 4.840 Aufschriften gehören zu genau einem Titel.

**Liste und Vorschläge nennen die Regaladresse** (entschieden und gebaut am 07.10.2026). Die
Seite „Signaturen", die Vorschläge im Feld „Signatur" (Buchformular, Bestellkorb) und die
Auswahl der Inventur lesen `GET /api/signaturen`; die Liste fasst nach dem Teil vor dem ersten
„ / " zusammen (`repository.SQLSignaturRegaladresse`, der Trenner ist
`repository.SignaturTrenner`, mit dem `SignaturAus` die Aufschrift zusammensetzt). Am Titel
bleibt die ganze Aufschrift, die Regalansicht zeigt sie. Littera baut die zweite Zeile in der
Vorgabe aus den „3 Anfangsbuchstaben des Verfassers" (Handbuch, „Generierung der Signatur"); in
der Sicherung von 2010 trägt `Sig2` bei 58.653 von 61.520 Exemplaren genau drei Zeichen. `Sig1`
enthält den Trenner nie, `Sig2` bei 235 Exemplaren („PoWi / Ich"): Geteilt wird deshalb am
ersten. Einen Schrägstrich ohne Leerzeichen tragen 1.411 Exemplare in `Sig1` („Pa/KL",
„DvD/Sp"); er trennt nicht (gezählt am 07.10.2026).

**Das Programm erfindet keine Signatur** (entschieden am 22.09.2026): Sie klebt am Buch, und
nach ihr steht es im Regal. Für einen neuen Titel bietet das Feld die Regaladressen an, die an
den Titeln des Bestands stehen (`GET /api/signaturen`, im Buchformular und im Bestellkorb über
`frontend/src/lib/utils/signaturen.js`); aus der Gattung der DNB wird keine gebildet. Ein Fach
liest die Übernahme nur aus einer Lernmittel-Signatur (`lmf.Zerlege`), sonst aus den
Schlagworten.

## Der Standort — am Exemplar und am Titel

Erhoben am 06.10.2026 an der Sicherung von 2010 und am Export der Titelliste vom Juni 2026.

* `Exemplar.Sonderstandort` (Text 255, mit Wertehilfe; `Sonderstandortkurz` trägt denselben
  Wert). Handbuch: Das Exemplar steht „nicht am gewöhnlichen Platz im Regal". 2010 tragen ihn
  1.609 von 61.520 Exemplaren mit 99 Werten: `Videoschrank` (306), `DAZ` (283), `C 123` (198),
  `Lehrerschrank` (147). Die Tabelle `SonderStandorte` ist leer.
* Am Titel hat die Bibliothek den Standort als Person eingetragen, an der Stelle des dritten
  Verfassers: in `Personen_Zuordnung` die vierte Stelle von `Flags`, im Export der Titelliste
  MAB 108a. 2010 stehen dort 8.730 Zuordnungen mit 50 Namen, davon 8.585 mit den fünf Vermerken
  aus `bestandsmarken`; dazu `Sonderstandort D-Bau` (33), `Software/Bibliothek` (25),
  `LMF/Bibliothek` (10). Im Juni 2026 sind es 678 Einträge ohne Komma in 33 Werten: `LMF`
  (156), `U plus` (122), `Klassensatz/Bibliothek` (71), `Bibliothek Klassensatz Regal 10` bis
  `14` (190), `Schulseelsorge` (32), `Sonderstandort D-Bau` (31). `Buchbestand Bibliothek`
  steht dort nur noch in der freien Verfasserangabe (MAB 359).
* Rund 7 der 678 Einträge sind keine Standorte, sondern Verfasser ohne Komma, eine Reihe oder
  ein Herausgeber. An der Stelle stehen außerdem 152 Verfasser in Katalogform; sie bleiben
  Verfasser.
* Der Vermerk `LMF` steht im Juni 2026 an 91 Titeln ohne LMF-Signatur (2010: 210). Die
  Übernahme erkannte Lernmittel bis zum 06.10.2026 nur an der Signatur.
* Beim Rundlauf einer Zeitschrift schreibt Littera Nummer und Namen des letzten Lesers in den
  Sonderstandort (Handbuch, „Exemplar bleibt beim letzten Leser des Rundlaufes"). 2010 trägt
  kein Wert diese Form.

Umsetzung: `internal/littera/standort.go`, Bedienung in [SCRIPTS.md](SCRIPTS.md), Abschnitt 1.

## Werkzeuglage — das alte Werkzeug ist nie gegen echte Daten gelaufen

`cmd/littera_migration` (inzwischen entfernt) fragte ab:

```sql
SELECT TitelID, Titel, Autor, ISBN, Verlag, Jahr, Signatur FROM TITEL
SELECT ExemplarID, TitelID, Barcode, ErworbenAm FROM EXEMPLARE
```

**Von diesen Spalten existierte einzig `ISBN`.** Der Kommentar im Werkzeug sagte es offen:
„Wir nehmen hier Standardnamen an." Es war ein Gerüst, das nie angepasst wurde — gegen
eine echte Littera-Datei brach es bei der ersten Abfrage ab. Dazu kam der ODBC-Zwang
(Treiber nur unter Windows), der es auf dem Arbeitsrechner und im Container ohnehin
unbrauchbar machte. Mit ihm ist die Abhängigkeit `alexbrainman/odbc` und der Build-Tag
`odbc` entfallen.

Ersetzt durch `internal/littera`: liest die `mdb-export`-CSVs (plattformunabhängig),
bildet auf die echten Spaltennamen ab und ist ohne Datenbank testbar. Gegen den
Altbestand belegt: **10.732 Titel, 61.520 Exemplare, 0 verwaiste Exemplare** — jedes
Exemplar findet seinen Titel. Der Lauf gegen die echten Dateien hängt an
`LITTERA_CSV_DIR` (wie die PG-Tests an `TEST_DATABASE_URL`).

### Spalten-Zuordnung für den vollständigen Import — alle geklärt

Diese Liste hieß bis zum 23.08.2026 „Noch offen", obwohl darunter ausschließlich
Erledigtes steht. Offen ist am Littera-Import nur noch EINES: ein **frisches Backup
aus dem laufenden Littera** (Dienstprogramme → Datensicherung, ~100 MB+) — für Leser,
Ausweisnummern und laufende Ausleihen; der MAB-Export (`katalogisat.xml`, Juni 2026,
13.708 Titel) deckt nur den Katalog. Die vorliegende `littera_sav.mdb` ist ein Stand
von **2010** — belegt an Leser 37 (letzte Bewegung 11.11.2010 in der Datei gegen
letzte Ausleihe 17.06.2026 im laufenden LITTERA 5.4) und an einem Buch von 2022 mit
Exemplar-Nr. 105785, während die Datei bei 61.520 endet. Der Export muss
`FremdLeserNummer` und `FremdBarcode` enthalten — dort liegen die Nummern der
Kartenhersteller bzw. Ersatzetiketten, in den Stammdaten stehen sie nicht.

* **Autoren — erledigt.** `Titel.Verfasserangabe` ist nur bei 2.519 von 10.732 Titeln
  brauchbar gefüllt (23 %). Die gepflegte Quelle sind `Personen` + `Personen_Zuordnung`,
  wobei **`Funktion = 0` den Verfasser** bezeichnet (1 = Illustrator, 2 = Herausgeber …
  stehen in `Personen_Funktionen`; die 0 fehlt dort, sie ist die Vorgabe). Damit steigt
  die Abdeckung auf **9.029 von 10.732 (84 %)** — gemessen, nach Abzug der
  Bestandsvermerke (siehe unten; die frühere Angabe 10.002 zählte sie mit). Mehrfachverfasser sind der
  Normalfall (6.178 Titel haben genau zwei) und werden in Erfassungsreihenfolge mit
  `; ` verbunden, nicht alphabetisch: Der Erstgenannte ist der Hauptverfasser.
* **Medienart — erledigt** (`MedienartNamen`, gleiche Bauart wie `Verlag`).
* **Leser — erledigt** (`LeseLeser`, `LeseLesergruppen`, `NurArt`): Einordnung nach
  `LeserArt`, Klasse aus `Leser_UG`.
* **Abgangsjahr — erledigt** (`AbgaengerJahr`, `IstAbschlussklasse`). `schueler.abgaenger_jahr`
  ist NOT NULL, Litteras `Abmeldedatum` aber bei **0 von 1.991** gefüllt — der Wert wird
  aus der Klasse gerechnet. Die Abschlussklassen sind **9H, 10R und 13**; die Regel ist
  nicht neu erfunden, sondern aus `api/student_promotion.go` (`is_graduating`) übernommen,
  damit Import und Schuljahreswechsel dieselbe Aussage treffen. Die Förderstufe rechnet
  bewusst mit dem längsten Weg (13): Solange der Schüler da ist, steht das Jahr nur im
  Profil; als Abgänger markieren ihn Versetzung und LUSD-Import, und beide setzen dann das
  tatsächliche Jahr — ein zu spätes kostet nichts, ein zu frühes stünde falsch in der Akte.
  Eine Klasse ohne Jahrgang („AUS") bekommt das Jahr der Handanlage (Kalenderjahr + 5,
  `repository.AbgangsjahrOhneKlasse`) und eine Warnung im Protokoll.
  Am Altbestand: für 1.720 von 1.734 Schülern ableitbar (die 14 „Im Ausland" nicht), davon
  232 in einer Abschlussklasse.
* **Ausleihen — erledigt** (`LeseAusleihen`, `NurOffene`, `OhneExemplar`, `OhneFrist`).
* **Schreibpfad nach Postgres — erledigt** (`cmd/littera-altbestand`, siehe unten).

## Der Schreibpfad — gemessen an einem echten Lauf

Die Härtung aus `cmd/migrate/pg_writer.go` wurde nicht kopiert, sondern nach
`internal/uebernahme` herausgelöst; beide Werkzeuge benutzen jetzt dieselbe. Bedienung:
`docs/SCRIPTS.md`.

Vollständiger Lauf in der Generalprobe vom 28.09.2026 (`scripts/generalprobe/`, frische
Datenbank aus den Migrationen), 22 Sekunden:

| | Quelle | geschrieben | Abgleich an der DB |
|---|---|---|---|
| Titel | 10.732 | 10.732 | ✓ |
| Exemplare | 61.520 | 61.520 | ✓ |
| Schüler (mit Abgegangenen und „Im Ausland") | 1.810 | 1.810 | ✓ |
| Kollegium (Lehrkräfte und Sonstige) | 181 | 181 | ✓ |
| Ausleihen | 15.615 | 15.612 | ✓ |

„Undefinierte Untergruppe" (5 Personen) hielt den Lauf zuerst an und ist hier als Schüler
zugeordnet, wie es die Bücherei in Littera täte. 1.876 Abwertungen (614 doppelte ISBN, 450
ungültige Prüfziffer, 411 Frist vor Verleihdatum, 180 Platzhalter-Mail, 168 leerer
Haupttitel, 23 Littera-Gruppe eines Kollegium-Kontos, 19 Klasse ohne Jahrgang, 8
Ersatz-Barcode, 2 Etikett ungleich Rechnung, 1 Ersatz-Ausweis) und 3 Ausfälle (2
Doppelbelegungen, 1 Ausleihe auf ein Exemplar, das es nicht gibt). Jede Zeile steht mit ihrem
Littera-Schlüssel im Protokoll. Bis zum 28.09.2026 fehlten 42 Personen und mit ihnen 341
Ausleihen.

### Das Etikett ist entschlüsselt

`Exemplar.Barcode` ist keine Nummer, sondern die Zeichenkette für den Etikettendruck:
`8 *pkpööp#-c.bc-*`. Die Ziffern liegen auf zwei Tastaturreihen (`qwertzuiop` und
`asdfghjklö` stehen beide für 1–9 und 0), dahinter Bibliotheksnummer (0395), Länge und
Prüfzeichen. Aufgelöst in `littera.BarcodeInhalt`; gegen den Altbestand stimmen **61.520
von 61.520** mit der Spalte `Exemplarnummer` überein.

Damit steht fest, dass die vorhandenen Etiketten die Exemplarnummer tragen — der Import
schreibt sie nach `buecher_exemplare.barcode_id`, und der Bestand bleibt ohne
Neubeklebung scannbar. Ob die Lesegeräte den Zifferninhalt liefern, konnte nur ein echtes
Buch beantworten: Zwei Scans in der Bibliothek am 18.08.2026 zeigten, dass der Strichcode
eine EAN-13 liefert (Nummer rechts auf 8 Stellen genullt, Bibliotheksnummer, Stellenzahl,
Prüfziffer). Die Theke rechnet sie auf die Nummer zurück, unter der das System das Exemplar
führt (`84cbe12d`) — `-barcodes neu` ist nicht nötig. Ein neues Etikett brauchen nur Exemplare,
deren Etikett schon ein anderes trägt (13 Exemplare teilen sich 5 Nummern, 8 bekommen eine neue;
SCRIPTS.md, Abschnitt 1).

Nachtrag 28.09.2026: Die Übernahme schreibt die EAN-13 des Etiketts nach `barcode_id`, nicht
die Exemplarnummer. Die Spalte `Barcode` ist diese EAN-13 in der Setzform einer EAN-13-Schrift
(Parität und Prüfziffer stimmen an allen 61.520 Exemplaren) und wird als Etikett übernommen
(`littera.EtikettZiffern`). `littera.EtikettBarcode` rechnete bis dahin für Nummern unter
sechs Stellen links aufgefüllt und mit einer festen 6 an Stelle 12 — anders als die beiden
Scans oben; getroffen hätte es jedes Exemplar dieser Sicherung. Siehe [SCRIPTS.md](SCRIPTS.md).

**Lernmittel mit „LMF" im Strichcode:** Manche Lernmittel tragen im Strichcode „LMF" und eine
lange Nummer; an der Theke bestätigt am 01.10.2026: Sie buchen. In der Sicherung von 2010 steht
„LMF" in keiner Zeichenkette der Spalte `Barcode` und in keiner `Exemplarnummer`, nur in `Sig1`
(gezählt am 07.10.2026); woher diese Etiketten stammen, ist nicht belegt. Die Theke schlägt
eine solche Nummer nach, wie sie gescannt wird, und `LMF-` ist eine Vorsilbe für Bücher wie
`B-`: am Server (`internal/service/omnibox_service.go`) und ohne Netz (`BUCH_VORSILBEN` in
`frontend/src/lib/scanEinordnen.js`); `internal/service/vorsilben_zwilling_test.go` hält beide
gleich. Sie gehört nicht zu den Resten des früheren Titelzusatzes „LMF-" und bleibt.

### Zwei Fehler, die der Schreibpfad aufgedeckt hat

**Das Autorenfeld war verschmutzt.** Die frühere Angabe „10.002 von 10.732 Titeln mit
Autor (93 %)" stimmte der Zahl, nicht der Sache nach: In `Personen` stehen neben Namen
auch Standortvermerke — `Buchbestand Bibliothek` (6.711 Titel), `Bibliothek` (760),
`Klassensatz/Bibliothek` (584), `LMF` (363), `U plus` (202). Littera kann sie nicht
unterscheiden; die Tabelle hat außer dem Namen kein Merkmal. Ohne Filter stünde bei 7.131
Titeln ein Regalvermerk mitten in der Autorenangabe („Shaw, George Bernard; Buchbestand
Bibliothek"). Nach der Bereinigung (`bestandsmarken`, wirkt auch auf den Freitext
`Verfasserangabe`): **9.029 Titel mit einem echten Autor**, 84 %.

**Geburtsdaten landeten in der Zukunft.** Go legt die Jahrhundertgrenze bei zweistelligen
Jahren fest auf 69. Für Ausleihdaten stimmt das, für Geburtsdaten nicht: 69 Personen —
Lehrkräfte der Jahrgänge 1946 bis 1968 — kamen als 2046 bis 2068 an. `GeburtsdatumAus`
holt jedes Datum in der Zukunft um ein Jahrhundert zurück.

### Was NICHT übernommen wird

Anschrift und E-Mail der Schüler. Littera führt sie (Adresse bei 1.927 von 1.991), ihr
Zweck laut `schema.sql` ist aber der Versand von Schadens-Rechnungen und Eltern-Mahnungen
— und die gepflegte Quelle dafür ist die LUSD. Eine Anschrift aus einem Altbestand ist im
Zweifel veraltet, und eine Rechnung an die falsche Adresse ist schlechter als gar keine.
Das Geburtsdatum kommt mit, weil `unique_schueler_name_gebdatum` nur dann greift: Ohne es
legt der spätere LUSD-Import dieselben Schüler ein zweites Mal an.

Sperren und Salden der Leser. Die Übernahme liest aus `Leser` weder die Sperre (`GesperrtAm`,
`SperrenBegründung`) noch den `Saldo`, und die Tabellen der Verrechnung gehören nicht zu den
fünfzehn, die sie liest: Wer in Littera gesperrt ist, kommt ohne Sperre an. In der Sicherung
von 2010 tragen 5 Leser eine Sperre und 4 einen Saldo ungleich 0 (gezählt am 07.10.2026). Was
daraus für die Sicherung von 2026 folgt, steht in [OFFEN.md](OFFEN.md) 7.2.

Lehrkräfte bekommen einen unzustellbaren Platzhalter unter `.invalid` (RFC 2606) statt
einer erfundenen Adresse unter der Schuldomäne — die ginge irgendwann an eine echte,
fremde Person. Und sie werden **inaktiv** angelegt: Die Anmeldung läuft über den Barcode,
158 aktive Konten aus einem Altbestand wären 158 Zugänge für Leute, die vielleicht längst
weg sind. `-lehrer-aktiv` schaltet das um.
