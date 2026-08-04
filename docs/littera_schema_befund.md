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

**Der Bestand endet 2010.** Letzte Zugangsdaten: 2009 (4.029), 2010 (2.996); die
Ausleihen laufen bis 2010 aus. Die 15.614 „offenen" Ausleihen sind daher mit hoher
Wahrscheinlichkeit **historischer Altbestand**, keine heute laufenden Ausleihen —
ein Import als offene Ausleihen würde tausende Bücher fälschlich als verliehen führen.
**Vor dem Import zu klären: Ist das die produktive Datenbank oder ein Archiv von 2010?**

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
| `Lehrer`, `Lehrerin` | `ArtLehrkraft` | `benutzer` mit Rolle `lehrer` |
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

## Werkzeuglage — das vorhandene Werkzeug hat nie gegen echte Daten gelaufen

`cmd/littera_migration` fragt ab:

```sql
SELECT TitelID, Titel, Autor, ISBN, Verlag, Jahr, Signatur FROM TITEL
SELECT ExemplarID, TitelID, Barcode, ErworbenAm FROM EXEMPLARE
```

**Von diesen Spalten existiert einzig `ISBN`.** Der Kommentar im Werkzeug sagt es offen:
„Wir nehmen hier Standardnamen an." Es ist ein Gerüst, das nie angepasst wurde — gegen
eine echte Littera-Datei bricht es bei der ersten Abfrage ab. Dazu kommt der ODBC-Zwang
(Treiber nur unter Windows), der es auf dem Arbeitsrechner und im Container ohnehin
unbrauchbar macht.

Ersetzt durch `internal/littera`: liest die `mdb-export`-CSVs (plattformunabhängig),
bildet auf die echten Spaltennamen ab und ist ohne Datenbank testbar. Gegen den
Altbestand belegt: **10.732 Titel, 61.520 Exemplare, 0 verwaiste Exemplare** — jedes
Exemplar findet seinen Titel. Der Lauf gegen die echten Dateien hängt an
`LITTERA_CSV_DIR` (wie die PG-Tests an `TEST_DATABASE_URL`).

### Noch offen für den vollständigen Import

* **Autoren — erledigt.** `Titel.Verfasserangabe` ist nur bei 2.877 von 10.732 Titeln
  gefüllt (27 %). Die gepflegte Quelle sind `Personen` + `Personen_Zuordnung`, wobei
  **`Funktion = 0` den Verfasser** bezeichnet (1 = Illustrator, 2 = Herausgeber … stehen
  in `Personen_Funktionen`; die 0 fehlt dort, sie ist die Vorgabe). Damit steigt die
  Abdeckung auf **10.002 von 10.732 (93 %)** — gemessen. Mehrfachverfasser sind der
  Normalfall (6.178 Titel haben genau zwei) und werden in Erfassungsreihenfolge mit
  `; ` verbunden, nicht alphabetisch: Der Erstgenannte ist der Hauptverfasser.
* **Medienart — erledigt** (`MedienartNamen`, gleiche Bauart wie `Verlag`).
* **Leser — erledigt** (`LeseLeser`, `LeseLesergruppen`, `NurArt`): Einordnung nach
  `LeserArt`, Klasse aus `Leser_UG`. Offen bleibt nur `schueler.abgaenger_jahr` (NOT NULL):
  Littera führt `Abmeldedatum` — im Altbestand bei **0 von 1.991** gefüllt. Der Wert muss
  also aus der Klassenstufe abgeleitet werden (z. B. Jahrgang aus `07H1` + Schuljahr).
* **Ausleihen**: Feldzuordnung steht (siehe oben), Lesepfad noch nicht gebaut.
* **Schreibpfad nach Postgres**: noch offen. Empfehlung, keinen zweiten zu bauen —
  `cmd/migrate/pg_writer.go` ist bereits gehärtet (Savepoints je Titel,
  Barcode-Prüfung, Fehlerklassen) und braucht nur einen anderen Leser davor.
