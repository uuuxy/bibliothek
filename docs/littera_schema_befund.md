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

### Datenstand — wichtig

| Tabelle | Zeilen |
|---|---|
| `Titel` | 11.076 |
| `Exemplar` | 61.580 |
| `Leser` | 2.008 |
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
`ISBN` → `isbn` · `Erscheinungsjahr` (Text!) · `Medienart` (Long → Nachschlagetabelle) ·
`Verlag` (Long → Nachschlagetabelle, **kein Freitext**) · `Annotation` → `beschreibung`

### `Exemplar` → `buecher_exemplare`
`Buchungsnummer` (Schlüssel) · `Titel` (FK) · `Barcode` → `barcode_id` ·
`Zugangsdatum` → `erworben_am` · `Preis` · `Status` (Long) · `Sig1` + `Sig2` (siehe unten)

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
| `Schüler`, **`Sekundarstufe II`** | `ArtSchueler` | `schueler` |
| `Lehrer`, `Lehrerin` | `ArtLehrkraft` | `benutzer` mit Rolle `kollegium` (bis Migration 069: `lehrer`) |
| `Abgegangen` | `ArtAbgegangen` | `schueler` mit `ist_abgaenger` |
| `Praktikant`, `Sekretärin`, `Fachbereich …` | `ArtSonstige` | kein Schüler |
| alles andere (`IMPORT`, `U-plus`, `Im Ausland`, `UNDEF`) | `ArtUnbekannt` | **nicht schreiben** |

`Sekundarstufe II` ist der Fall, den man leicht übersieht: eine eigene Untergruppe,
aber die Oberstufenklassen (11T1, 12T3, 13T5) sind selbstverständlich Schüler.

Gemessen am Altbestand: **1.720 Schüler · 158 Lehrkräfte · 71 abgegangen · 20 sonstige ·
22 unklar** — und jeder der 1.720 Schüler hat eine Klasse (`schueler.klasse` ist NOT NULL,
ein Test sichert das ab). Unklare Zeilen werden bewusst **nicht** stillschweigend zu
Schülern: Bei Personendaten ist eine ausgelassene Zeile das kleinere Übel als eine
falsch einsortierte.

Feldbelegung, gemessen: Nachname 1.991 · Vorname 1.980 · Geburtsdatum 1.924 ·
Adresse 1.927 · Anmeldedatum 834 · **eMail nur 3** · Abmeldedatum 0.
Die Mahnung per E-Mail hat aus dieser Quelle also praktisch keine Grundlage —
`eltern_email` muss aus LUSD kommen, nicht aus Littera.

### `Verleih` → `ausleihen`
`Exemplar` (FK) · `Leser` (FK) · `Verleihdatum` · `Rückgabedatum` (Frist) ·
`IstRückgabedatum` (tatsächlich) · `Zurückgegeben` (Boolean) · `Mahnungen`

## Die Signatur — betrifft die Umstellung von Migration 060

Littera führt die Signatur **am Exemplar, nicht am Titel**, und **zweiteilig**:

* `Exemplar.Sig1` — die Regaladresse: `LMF Bio 11`, `LMF Deu 6`, `Ea`, `Ga`
* `Exemplar.Sig2` — das Kürzel des Titels darin: `Lin`, `Nat`, `Bos`, `Bär`

Zusammen ergibt das die Aufschrift vom Buchrücken: `LMF Deu 7 / Bie`.

**Belegung: 61.546 von 61.580 Exemplaren haben `Sig1`** — praktisch vollständig.

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

### Noch offen für den vollständigen Import

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
  bewusst mit dem längsten Weg (13): Ein zu frühes Abgangsjahr würde einen Schüler
  archivieren, der noch zur Schule geht; ein zu spätes zieht der Versetzungslauf nach.
  Am Altbestand: für **alle 1.720 Schüler ableitbar**, davon 232 in einer Abschlussklasse.
* **Ausleihen — erledigt** (`LeseAusleihen`, `NurOffene`, `OhneExemplar`, `OhneFrist`).
* **Schreibpfad nach Postgres — erledigt** (`cmd/littera-altbestand`, siehe unten).

## Der Schreibpfad — gemessen an einem echten Lauf

Die Härtung aus `cmd/migrate/pg_writer.go` wurde nicht kopiert, sondern nach
`internal/uebernahme` herausgelöst; beide Werkzeuge benutzen jetzt dieselbe. Bedienung:
`docs/SCRIPTS.md`.

Vollständiger Lauf gegen ein frisches PostgreSQL mit dem Produktivschema, 16 Sekunden:

| | Quelle | geschrieben | Abgleich an der DB |
|---|---|---|---|
| Titel | 10.732 | 10.732 | ✓ |
| Exemplare | 61.520 | 61.520 | ✓ |
| Schüler | 1.791 | 1.791 | ✓ |
| Lehrkräfte | 158 | 158 | ✓ |
| Ausleihen | 15.615 | 15.271 | ✓ |

1.804 Abwertungen (614 doppelte ISBN, 450 ungültige Prüfziffer, 406 Frist vor
Verleihdatum, 168 leerer Haupttitel, 157 Platzhalter-Mail, 8 Ersatz-Barcode, 1
Ersatz-Ausweis) und 386 Ausfälle (341 Ausleihen an nicht übernommene Sammelkonten, 42
Personen ohne Zuordnung, 2 Doppelbelegungen, 1 Ausleihe auf ein Exemplar, das es nicht
gibt). Jede Zeile steht mit ihrem Littera-Schlüssel im Protokoll.

### Das Etikett ist entschlüsselt

`Exemplar.Barcode` ist keine Nummer, sondern die Zeichenkette für den Etikettendruck:
`8 *pkpööp#-c.bc-*`. Die Ziffern liegen auf zwei Tastaturreihen (`qwertzuiop` und
`asdfghjklö` stehen beide für 1–9 und 0), dahinter Bibliotheksnummer (0395), Länge und
Prüfzeichen. Aufgelöst in `littera.BarcodeInhalt`; gegen den Altbestand stimmen **61.520
von 61.520** mit der Spalte `Exemplarnummer` überein.

Damit steht fest, dass die vorhandenen Etiketten die Exemplarnummer tragen — der Import
schreibt sie nach `buecher_exemplare.barcode_id`, und der Bestand bleibt ohne
Neubeklebung scannbar. **Offen bleibt eine Frage, die nur ein echtes Buch beantwortet:**
ob die Lesegeräte den Zifferninhalt liefern. Falls nicht, ist `-barcodes neu` der Weg —
dann brauchen alle 61.520 Exemplare ein neues Etikett.

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

Lehrkräfte bekommen einen unzustellbaren Platzhalter unter `.invalid` (RFC 2606) statt
einer erfundenen Adresse unter der Schuldomäne — die ginge irgendwann an eine echte,
fremde Person. Und sie werden **inaktiv** angelegt: Die Anmeldung läuft über den Barcode,
158 aktive Konten aus einem Altbestand wären 158 Zugänge für Leute, die vielleicht längst
weg sind. `-lehrer-aktiv` schaltet das um.
