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

2. **Unser `buecher_titel.signatur` liegt am Titel — Littera am Exemplar.** Gemessen:
   Bei **98 von 10.454** Titeln (0,9 %) unterscheiden sich die Signaturen der Exemplare
   untereinander. Für 99 % ist die Titel-Ebene verlustfrei; für diese 98 muss der Import
   entscheiden (häufigsten Wert nehmen und die Abweichungen protokollieren — nicht still
   den ersten nehmen).

## Werkzeuglage

`cmd/littera_migration` zielt bereits auf Access, nutzt aber ODBC mit dem Treiber
„Microsoft Access Driver (*.mdb, *.accdb)" — **den gibt es nur unter Windows**. Auf dem
Mac und im Container läuft das nicht. `cmd/migrate` liest MySQL und passt gar nicht.

Empfehlung: Die Tabellen mit `mdb-export` nach CSV ziehen (plattformunabhängig, hier
erprobt) und den vorhandenen CSV-/Tabellen-Importpfad nutzen, statt einen ODBC-Zwang
einzubauen, der nur auf einem Windows-Rechner erfüllbar ist.
