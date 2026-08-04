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

### `Leser` → `schueler`
`Lesernummer` · `Vorname` · `Nachname` · `eMail` · `Geburtsdatum` · `Anmeldedatum` ·
Adressfelder (**DSGVO: nur übernehmen, was gebraucht wird**)

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

* **Autoren**: Nur 2.877 von 10.732 Titeln haben eine `Verfasserangabe`. Die saubere
  Quelle sind vermutlich die Tabellen `Personen` / `Personen_Zuordnung` — noch nicht
  ausgewertet.
* **Leser und Ausleihen**: Zuordnung steht (siehe oben), Lesepfad noch nicht gebaut.
* **Medienart** (`Titel.Medienart`, Zahl) muss wie `Verlag` über die Nachschlagetabelle
  aufgelöst werden.
