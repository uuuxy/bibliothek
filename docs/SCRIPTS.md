# Kommandozeilen-Skripte und Werkzeuge

---

## 0. Reihenfolge beim Umstieg: Littera, dann LUSD — und nie umgekehrt

Die Importe hängen aneinander; die Reihenfolge steht sonst nur im Code (02.09.2026):

1. **Littera-Bestand** (Katalogisat, Abschnitt 1a, oder Access-Altbestand, Abschnitt 1) —
   Titel und Exemplare mit Barcodes.
2. **Littera-Personen und -Ausleihen im selben Lauf** (`-personen -ausleihen`, Abschnitt 1).
   Die Ausleihen finden ihre Entleiher nur über eine Zuordnung im Speicher des laufenden
   Prozesses; ein späterer Ausleihlauf meldet alle als „ohne Entleiher".
3. **LUSD-Import** (All_Inklusiv-Bericht, der einzige Export mit Schüleradresse). Er erkennt
   Littera-Schüler an Name + Geburtsdatum und übernimmt sie samt Klasse und Anschrift, statt
   sie doppelt anzulegen. Der Littera-Personenlauf kann das NICHT: nach einem LUSD-Import
   legt er jeden Schüler ein zweites Mal an — seine Wiederholungssperre zählt nur
   Littera-Zeilen.

Das Geburtsdatum ist die Brücke: Fehlt es im Littera-Backup, legt die LUSD den Schüler
doppelt an. Vor dem Lauf am frischen Backup prüfen.

## 1. LITTERA-Altbestand (`cmd/littera-altbestand`)

Überträgt den Littera-Altbestand nach PostgreSQL. Quelle sind `mdb-export`-CSVs, nicht
die Access-Datei selbst: Deren ODBC-Treiber gibt es nur unter Windows.

```bash
# 1. Export erzeugen (mdbtools, plattformunabhängig)
for t in Titel Exemplar Verlag Medienart Personen Personen_Zuordnung Leser Leser_UG Verleih \
         Schlagworte Schlag_zuord Verweise_Schlagworte Verweis_Zu_Schlagworte \
         Interessenskreis IntZuMed; do
  mdb-export littera_sav.mdb "$t" > "littera-export/$(echo "$t" | tr 'A-Z' 'a-z').csv"
done

# 2. Trockenlauf – liest und berichtet, schreibt nichts. Mit denselben Schaltern wie der
#    Lauf (etwa -personen -ausleihen): Dann nennt er auch Lesergruppen ohne Zuordnung.
go run ./cmd/littera-altbestand -csv ./littera-export -trocken

# 3. Übernahme
go run ./cmd/littera-altbestand -csv ./littera-export -db "$DATABASE_URL"
```

**Was übernommen wird**, steuern `-bestand` (Vorgabe an), `-personen` und `-ausleihen`
(Vorgabe aus). Die Vorgabe ist Absicht: Der geprüfte Export ist ein Stand von 2010
(Schüler-Geburtsjahre 1989–2001, letzte Ausleihe 2010). Personen und Ausleihen daraus
legen Schüler an, die heute Mitte dreißig sind, und melden ein Viertel des Bestands als
verliehen. Für einen aktuellen Export sind beide Schalter richtig.

**Barcodes:** `-barcodes littera` (Vorgabe) schreibt den Wert, den ein Scanner vom
Buchetikett liest — eine **EAN-13**, nicht die daneben gedruckte Exemplarnummer:

```
5 8 9 6 8 0 0   0 3 9 5   5   6      Exemplar-Nr. 58968 → 5896800039556
└ Exemplarnr., ┘ └ Bibl.-Nr┘ │   └ Prüfziffer
  rechts auf 7    0395       └ Stellenzahl der Exemplarnummer
  Stellen genullt
```

Belegt an vier gescannten Büchern und an allen 61.520 Exemplaren der Sicherung von 2010.
Litteras Spalte `Barcode` ist dieselbe EAN-13, gesetzt für eine EAN-13-Schrift
(`8 *pkpööp#-c.bc-*` ergibt `8080000039530`); Parität und Prüfziffer stimmen bei allen
61.520. Die Übernahme nimmt deshalb diese Zeichenkette als Etikett (`littera.EtikettZiffern`)
und rechnet aus Exemplar- und Bibliotheksnummer (`littera.EtikettBarcode`) nur, wenn sie
fehlt. Weichen beide ab, gilt das Etikett, mit Vermerk im Protokoll; in der Sicherung von
2010 zweimal (Bibliotheksnummer 0 in der Spalte, 0395 auf dem Etikett). Der Bestand bleibt
damit ohne Neubeklebung scannbar. Bis zum 28.09.2026 rechnete `EtikettBarcode` für Nummern
unter sechs Stellen falsch (links aufgefüllt, an Stelle 12 fest eine 6); getroffen hätte es
jedes Exemplar der Sicherung von 2010.
**Schlagworte (seit dem 30.09.2026, docs/OFFEN.md 4.20):** Mit dem Bestand kommen alle
Schlagworte der Titel mit, aus den Tabellen `Schlagworte` und `Schlag_zuord`, geschrieben über
den Pfad des Buchformulars (`repository.SetzeSchlagworte`): Leerraum wird zusammengezogen,
dieselbe Schreibweise in anderer Groß- und Kleinschreibung ist dasselbe Wort, höchstens 300 je
Titel. Ein leeres oder zu langes Wort fällt weg, eine längere Liste wird gekürzt — beides mit
Vermerk im Protokoll; das Buch selbst kommt in jedem Fall an. Die Verweise
(`Verweise_Schlagworte`, `Verweis_Zu_Schlagworte`) zählt der Lauf nur und nennt sie: In der
Sicherung von 2010 sind beide leer, ihre Form ist an echten Daten noch nicht geprüft.
**Interessenkreise (seit dem 30.09.2026):** Die Tabellen `Interessenskreis` und `IntZuMed` kommen
als Schlagworte mit, hinter den Schlagworten des Titels und über denselben Pfad; ein gleichlautendes
Wort („U plus") zählt einmal. An dieser Schule sind es Zielgruppen — Sekundarstufe 1 und 2,
Lehrer, Referendare, DAZ-Schüler; in der Sicherung von 2010 36 Werte an 8.902 Titeln. Die Stufe
ergibt die Jahrgangsspanne, wo die Signatur keine nennt (wie im Katalogisat-Import). Alle sechs
Dateien sind Pflicht. Der Bericht führt die Schlagworte als eigenen Abschnitt mit Abgleich an der
Datenbank; die Interessenkreise stehen dort als eigene Zeile. **Fach:** Nennt die Signatur keins, kommt es aus den Schlagworten, wenn sie genau ein
Fach nennen — dieselbe Regel wie im Katalogisat-Import (Abschnitt 1a); der Bericht zählt beide
Herkünfte unter „Bestand".
**Eigentum:** Der Bericht nennt unter „Bestand", wie viele Exemplare ihr Eigentum aus dem
Littera-Vermerk bekommen (Land, Schulträger), wie viele einen bekannten Vermerk ohne Zuordnung
tragen und wie viele einen, der nicht in der festen Liste steht. Die letzten stehen einzeln im
Protokoll, mit Exemplarnummer und ohne Wortlaut; nachsehen in Littera
([littera_schema_befund.md](littera_schema_befund.md), Abschnitt `Exemplar`).

`-barcodes neu` vergibt stattdessen frische `B-XXXXX` aus `barcode_seq` — derselben
Sequenz, aus der die Anwendung ihre Barcodes zieht — und setzt voraus, dass jedes Buch
ein neues Etikett bekommt.

**Neue Nummer, neues Etikett:** Eine Nummer aus `barcode_seq` bekommt auch mit
`-barcodes littera` jedes Exemplar, dessen Etikett schon ein anderes trägt („Barcode bereits
vergeben" im Protokoll; in der Sicherung von 2010 teilen sich 13 Exemplare 5 Nummern, 8
bekommen eine neue), aus dem sich kein Etikett bilden lässt oder dessen Nummer zu lang für
die Spalte ist. Diese Exemplare legt der Lauf seit dem 28.09.2026 als ungedruckt an; sie
stehen im Druck-Center unter „Fehlende Etiketten", mit `-barcodes neu` alle. Bis das neue
Etikett klebt, liefert ein Buch mit doppeltem Etikett am Scanner die Nummer des Exemplars,
das die Nummer behalten hat.

Trägt Litteras Tabelle `FremdBarcode` für ein Exemplar ein Ersatzetikett, gewinnt das
gegen die EAN-13 des Etiketts. Dasselbe gilt für Schülerausweise: `FremdLeserNummer` hält
die Nummer des Kartenherstellers (`B97601826457`), und die steht in keinem Stammdatenfeld.
Beide Dateien sind optional; fehlen sie, rechnet der Import mit den Littera-Nummern.

**Härtung:** Ein Savepoint je Datensatz (`internal/uebernahme`), Postgres-Fehler nach
SQLSTATE eingeordnet, Abwertungen (verworfene ISBN, gekürztes Feld) getrennt von
Ausfällen protokolliert nach `littera_import.log`. Nach jedem Abschnitt wird die gemeldete
Zahl gegen den tatsächlichen Zeilenzuwachs gehalten.

**Wiederholung:** Ein zweiter Lauf wird abgelehnt — es gibt keinen natürlichen Schlüssel,
an dem Postgres die Dublette erkennen könnte. Zum Aufräumen:

```sql
DELETE FROM ausleihen a USING buecher_exemplare e
  WHERE a.exemplar_id = e.id AND e.erweiterte_eigenschaften ? 'littera_id';
DELETE FROM buecher_exemplare WHERE erweiterte_eigenschaften ? 'littera_id';
DELETE FROM buecher_titel     WHERE erweiterte_eigenschaften ? 'littera_id';
DELETE FROM ausleihen WHERE schueler_id IN (SELECT id FROM schueler WHERE lusd_id LIKE 'littera:%');
DELETE FROM schueler  WHERE lusd_id LIKE 'littera:%';
DELETE FROM benutzer  WHERE rolle = 'kollegium' AND email LIKE '%@littera.invalid';
```

> Hier stand bis zum 11.08.2026 `rolle = 'lehrer'`. Seit Migration 069 gibt es diesen
> Enum-Wert nicht mehr, und Postgres bricht den **Vergleich** ab, nicht nur den Treffer:
> `ERROR: invalid input value for enum benutzer_rolle: "lehrer"`. Die fünf Anweisungen
> davor laufen durch — wer den Block einfügt, räumt also alles auf **außer** den
> Littera-Lehrkraftkonten und sieht am Ende einen Fehler, der nach „nichts passiert"
> aussieht.

**Lehrkräfte räumt der Block nicht vollständig auf:** Er löscht ihre Konten, ihre Leserzeilen
bleiben mit der Ausweisnummer stehen; eine Lehrkraft, die in Littera eine echte Adresse hat,
trifft er gar nicht. Für den Echtbetrieb ist deshalb am 28.09.2026 entschieden: Die Übernahme
läuft am Schulserver auf einer neu angelegten Datenbank, und muss sie wiederholt werden, wird
die Datenbank neu angelegt statt aufgeräumt. Der Block bleibt für eine Test-Datenbank.

**Ersatznummern nach dem Aufräumen:** Wer keine verwendbare Ausweisnummer mitbringt, bekommt
`A-<Littera-Kennung>`. Das Aufräumen löscht die Zeilen der Schüler, und eine gelöschte A-Nummer
vergibt das Programm nicht noch einmal (Migration 146) — der zweite Lauf gibt diesen Personen
`A-<Littera-Kennung>-2`. Karten, die nach dem ersten Lauf gedruckt wurden, gelten dann nicht
mehr. Nummern aus Littera selbst (Lesernummer, Herstellernummer der Karte) betrifft das nicht:
Sie sind keine A-Nummern.

**Lehrkräfte** werden mit `aktiv = true` angelegt — die Ausweis-Abfrage der Omnibox
filtert darauf, eine inaktive Lehrkraft wäre am Scanner unauffindbar. Den Login sperrt
statt dessen die Adresse: Anmeldung geht ausschließlich über IMAP gegen den Schul-Mail-
server, und `littera-4908@littera.invalid` gibt es dort nicht. `-lehrer-inaktiv` kehrt das
um, macht die Karten aber wertlos.

**Jeder Leser kommt an** (seit 28.09.2026): Praktikanten, Sekretariat, U-plus und die
Sammelkonten der Fachbereiche ins Kollegium (Ausleihen als Dauerleihe, keine Mahnung), seit
dem 30.09.2026 jeweils mit ihrer eigenen Art — Praktikum, Sekretariat, U-plus, Fachbereich;
vorher als Lehrkraft, die Littera-Gruppe nur als Warnung im Protokoll. Praktikum und
Fachbereich bekommen kein Konto, nur die Leserzeile; Sekretariat und U-plus ein Konto mit
Platzhalter-Adresse wie eine Lehrkraft. „Im Ausland" kommt als Schüler mit der
Klasse `AUS`. Ein Schüler, dessen Klasse keinen Jahrgang nennt, bekommt das Abgangsjahr der
Handanlage (Kalenderjahr + 5) und eine Warnung. Eine Lesergruppe, die keiner Art zugeordnet
ist (in der Sicherung von 2010 „Undefinierte Untergruppe", 5 Personen, 21 Ausleihen), **hält
den Lauf mit `-personen` an, bevor er etwas schreibt** — der Trockenlauf mit denselben
Schaltern endet dann ebenfalls mit 1 und nennt Gruppe, Personen- und Ausleihzahl. Zuordnen in
Littera selbst (die Personen in ihre Gruppe setzen, neu sichern und ausgeben) oder, für eine
neue Gruppenbezeichnung, in `artAusUntergruppe` (`internal/littera/leser.go`).

**Rückgabewerte:** 0 vollständig · 1 abgebrochen · 2 unvollständig (Details im Protokoll).

> Das frühere `cmd/littera_migration` ist entfallen. Es fragte `SELECT TitelID, Titel,
Autor, ISBN, Verlag, Jahr, Signatur FROM TITEL` ab — von diesen Spalten existiert in
> einer echten Littera-Datei einzig `ISBN`. Es ist nie gegen echte Daten gelaufen.

### 1a. Katalogisat aus dem MAB2-Export (`cmd/littera-import`)

Der zweite Weg in denselben Bestand — und für den Titelkatalog inzwischen der bessere.
Quelle ist kein Access-Backup, sondern ein **MAB2-Katalogisat (XML)**, das Littera selbst
ausgibt. Der Unterschied ist die Aktualität: Die vorliegende `littera_sav.mdb` ist ein
Stand von 2010, `katalogisat.xml` von Juni 2026 mit 13.708 Titeln.

```bash
# Trockenlauf gibt es hier nicht — der Import läuft in EINER Transaktion,
# ein Fehler rollt alles zurück. Vorher ein Backup ziehen.
go run ./cmd/littera-import -file katalogisat.xml -db "$DATABASE_URL"
```

- `-file` (Pflicht): Pfad zur Katalogisat-XML. `-db` fällt auf `$DATABASE_URL` zurück.
- Läuft über **denselben** Service-Pfad wie `POST /api/import/littera` — es gibt also
  keine zweite Importlogik, die eigene Fehler machen könnte.
- **Keine Dubletten bei Re-Imports:** Titel werden über ISBN oder Titel gegen den Bestand
  gematcht; ein zweiter Lauf legt keinen Titel doppelt an. Einträge mit demselben Titeltext
  oder derselben ISBN legt schon der erste Lauf zu einem Titel zusammen, und beim zweiten
  kommen Titeltext, Autor, Verlag, Jahr und Signatur aus dem Eintrag, der zuletzt passt — am
  Katalogisat vom Juni 2026 änderte der zweite Lauf 789 Titel (docs/OFFEN.md 6.1).
- Signaturen landen in `buecher_titel.signatur` (der echten Spalte) — **unverändert, wie
  Littera sie liefert** („LMF Bio 7"). Bis Migration 093 (02.09.2026) schnitt der Import „LMF"
  aus der Signatur und stellte es dem Titel voran („LMF-Biologie heute 7"); das ist vorbei.
- **Lernmittel ist ein Feld** (`ist_lernmittel`): gesetzt, wenn die Signatur oder das
  Standortfeld MAB 108a die LMF-Kennung trägt. Der Import setzt es nur, löscht es nie.
- **Fach und Jahrgang** kommen aus der Lernmittelsignatur („LMF Bio 7" → Biologie, 7;
  `pkg/lmf.Zerlege`), sonst aus Litteras Schlagwörtern (MAB 710, nur wenn sie genau ein
  Fach nennen) und den Zielgruppen (MAB 070b, Litteras Interessenkreise, alle Werte eines
  Eintrags: Sek I → 5–10, Sek II → 11–13, beide → 5–13). Fach und
  Klassenstufe füllen nur Leerstellen, die Jahrgangsspanne folgt der Quelle.
- **Schlagworte (seit dem 30.09.2026, docs/OFFEN.md 4.20):** Die Wörter aus MAB 710 und dahinter
  die Interessenkreise aus MAB 070b („Lehrer", „Referendare", „Sekundarstufe 2") kommen an
  Titel, die noch keine tragen, über den Pfad des Buchformulars (`repository.SetzeSchlagworte`)
  und aufbereitet wie in der Übernahme aus der Sicherung (`repository.SchlagworteAusFremddaten`):
  leer oder zu lang fällt weg, höchstens 300 je Titel. Wer schon Schlagworte trägt, behält
  seine; ein erneuter Import überschreibt keine Pflege. Steht ein Titel zweimal in der Datei,
  gilt der erste Eintrag mit Schlagworten. Am Katalogisat vom Juni 2026 in eine leere Datenbank
  gemessen, mit den Interessenkreisen: 11.302 Titel, 7.305 davon mit Schlagworten, 38.365
  Zuordnungen, 10.877 Wörter, 18 Sekunden gegen eine lokale Datenbank.
- **Re-Import als Reparatur:** Ein Bestand, der vor Migration 093 importiert wurde, bekommt
  durch einen erneuten Lauf derselben Datei Fach und Jahrgang nachgetragen — die Migration
  hat Titel und Lernmittel-Feld bereits bereinigt, die Titel matchen also.
- Zeitlimit 10 Minuten: ~15.000 Titel brauchen gegen eine entfernte Datenbank Minuten,
  nicht Sekunden.
- **Rückgabewerte:** 0 erfolgreich · 1 abgebrochen (fehlende Datei, DB nicht erreichbar,
  Importfehler). Die Zahl verarbeiteter Titel steht als `verarbeitete_titel` im JSON-Log.

### 1b. Generalprobe mit einer Littera-Sicherung (`scripts/generalprobe/`)

Übernimmt eine Littera-Sicherung in eine frische Datenbank und prüft das Ergebnis durch die
Anwendung — der Probelauf vor dem Umstieg. Läuft auf dem Arbeitsrechner, nicht am Server.

```bash
SICHERUNG="$HOME/littera/littera_sav.mdb"   # oder ein Verzeichnis mit den CSVs aus Abschnitt 1
scripts/generalprobe/generalprobe.sh "$SICHERUNG"
```

Was die Probe tut, in dieser Reihenfolge:

1. **Neuaufbau:** Image aus dem aktuellen Stand, leere Datenbank, das Backend legt sie selbst
   an; alle Migrationen müssen eingetragen sein.
2. **Abschottung, bevor Daten hineinkommen:** Das Backend hängt nur an einem Netz ohne Weg ins
   Internet (erreicht es doch eins, bricht die Probe ab). Anmeldung über die IMAP-Attrappe,
   keine Mail, kein S3, kein Cover-Abgleich.
3. **Export** der fünfzehn Tabellen aus Abschnitt 1 mit `mdb-export`, dazu `FremdLeserNummer` und
   `FremdBarcode`, wenn es sie gibt — oder die CSVs aus dem übergebenen Verzeichnis.
4. **Trockenlauf** mit `-personen -ausleihen`. Nennt er Lesergruppen ohne Zuordnung, prüft die
   Probe, dass der echte Lauf anhält und nichts schreibt. Weiter geht es dann nur mit
   `--gruppe 'Undefinierte Untergruppe=Schüler'`: Das stellt nach, was die Bücherei in Littera
   täte, und zwar nur in der Kopie des Exports.
5. **Übernahme:** alle Abgleiche an der Datenbank, jeder Leser übernommen, keine Ausleihe ohne
   Entleiher; das Protokoll nach Grund gezählt, ohne Werte.
6. **Theke,** über das interne Netz so angesprochen wie von der Oberfläche: Etikettenwerte je
   Stellenzahl der Exemplarnummer (aus Litteras Spalte `Barcode`, unabhängig vom Code der
   Übernahme gelesen), Ausweis, Ausleihe und Rückgabe, Rückgabe eines in Littera verliehenen
   Buchs, Buchliste für die Theke ohne Netz, Etikett-Nachdruck, Mahnwesen mit den Mahnbriefen
   aller Klassen, Leserdatei, Katalog. Ausleihe und Rückgaben werden an der Datenbank belegt.
7. **Nachtsicherung** mit dem Code des Jobs (`nachtsicherung.go`, ohne S3 und Mail) und
   **Wiederherstellung** nach [resilience_and_recovery.md](resilience_and_recovery.md) 2a in
   eine Wegwerf-Datenbank: Einspielen mit `ON_ERROR_STOP`, Zeilen je Tabelle gegen den Dump
   (Schritt 7), Tabellen, Indizes und Trigger gleich, das Programm startet darauf.

**Voraussetzungen:** Docker, Go, Python 3, `psql` und `pg_dump` in Version 18, für eine
`.mdb` die mdbtools; Port 5436 frei. Die Probe braucht einige Minuten.

**Personendaten:** Export, Protokoll und Dumps liegen nur in einem Arbeitsverzeichnis
(`mktemp`, nur für den eigenen Benutzer lesbar) und werden am Ende gelöscht, der Stack mit
`down -v`. Die Ausgabe nennt Zählungen, Nummern und Gruppenbezeichnungen, keinen Namen.
`--behalten` lässt beides zur Durchsicht stehen und nennt die Befehle zum Löschen.

**Rückgabewerte:** 0 jede Prüfung bestanden · 1 eine Prüfung abgewichen oder abgebrochen.

**Ergebnis am 28.09.2026** mit der Sicherung von 2010 (Stand `0437fecc`): jede Prüfung
bestanden. 10.732 Titel, 61.520 Exemplare, 1.991 Leser (1.810 Schüler, 181 im Kollegium),
15.612 von 15.615 Ausleihen; die übrigen drei sind Widersprüche in Littera (ein Exemplar fehlt
im Bestand, zwei sind doppelt verliehen). „Undefinierte Untergruppe" (5 Personen, 21
Ausleihen) hielt den Lauf an und wurde mit `--gruppe` als Schüler nachgestellt.
Am 30.09.2026 mit den Schlagworten wieder bestanden: 24.109 Zuordnungen an 10.364 Titeln,
2.742 Wörter (ein leeres Wort weggelassen, „Brasilien" stand zweimal), und das Fach kommt bei
258 Titeln aus der Signatur und bei 2.945 aus den Schlagworten.
Mit den Interessenkreisen am selben Tag wieder bestanden: 11.344 Interessenkreise an 8.902 Titeln
kommen dazu; geschrieben 35.394 Zuordnungen an 10.499 Titeln (59 fielen mit einem gleichlautenden
Schlagwort zusammen), 2.767 Wörter, Abgleich stimmt. Das Fach bleibt bei 258 und 2.945.

---

## 2. Foto-Migration (`cmd/migrate-fotos`)

Migriert unverschlüsselte Bilddateien vom Dateisystem in die Datenbank.

- **Funktionsweise:** Iteriert über ein Verzeichnis mit Schülerfotos, validiert und verschlüsselt diese (AES-256-GCM), speichert sie als `BYTEA` in `schueler_fotos`, **liest sie zur Gegenprobe zurück** und **löscht die Quelldatei erst danach**.
- **Zweck:** Konsolidierung der Infrastruktur (kein separates Foto-Verzeichnis) + Datensicherheit.

**Warum es selbst aufräumt (seit 23.08.2026).** Bis dahin sagte das Werkzeug nur „Du
kannst das Verzeichnis `uploads/fotos` jetzt sicher löschen" — ob das jemand tat, wusste
niemand. Was liegen blieb, sind unverschlüsselte Schülerfotos auf der Platte, und ihre
Dateinamen sind die Barcode-IDs vom Schülerausweis. Der Server liefert Unterordner von
`uploads/` nicht aus (`inventur/api_routen.go`); ein Hinweis auf der Konsole ist für
lesbare Fotos trotzdem die falsche Sicherung.

Gelöscht wird **erst nach bestandener Gegenprobe**: Das eben geschriebene Foto wird
zurückgelesen, entschlüsselt und mit dem Original verglichen. Bis dahin ist die Datei die
einzige Kopie; ein „INSERT ohne Fehler" heißt noch nicht, dass sich das Bild je wieder
anzeigen lässt. Scheitert die Probe, bleibt die Datei liegen und der Lauf meldet es.

Am Ende sagt das Werkzeug, wie viele Dateien es entfernt hat — und wenn welche übrig
sind, warum das zählt und wie man sie los wird (`shred -u uploads/fotos/*.jpg`).

```bash
# Aufräumen ausdrücklich abschalten (die Quelldateien bleiben dann unverschlüsselt liegen):
FOTOS_BEHALTEN=1 docker compose exec backend ./migrate-fotos
```

> **Im Image seit 12.09.2026.** Bis dahin nannte dieser Abschnitt den Aufruf, ohne dass
> das Werkzeug je im Container lag (das Dockerfile baute es nicht, der Schulserver hat
> kein Go) — dafür lag ein fertiges Binary im Repo. Dass jedes hier genannte Werkzeug
> auch im Image liegt, hält jetzt ein Gate fest (`docs/werkzeuge_im_image_test.go`).

> **Auf dem Schulserver prüfen:** Der Lauf von vor dem 23.08.2026 hat nichts gelöscht.
> `docker compose exec backend ls -la uploads/fotos` sagt, ob dort noch Altbestand liegt.

---

## 3. Datenbank-Backup (`jobs/backup.go` / `scripts/backup.sh`)

Es gibt **drei Wege**, und seit dem 23.08.2026 verschlüsseln alle drei über dieselbe
Ableitung (`internal/backupkrypto`, scrypt + AES-256-GCM). Hier stand bis zum 06.08.2026
eine gemeinsame Zeile für alle — sie beschrieb den automatischen Weg und ließ die
Shell-Wege sicherer aussehen, als sie waren.

|               | Automatisch (`jobs/backup.go`)                            | Manuell (`scripts/backup.sh`)              | Vor jedem Deploy (`./update.sh`)               |
| ------------- | --------------------------------------------------------- | ------------------------------------------ | ---------------------------------------------- |
| Auslöser      | Täglich 02:30 via `jobs/cron.go`                          | `./scripts/backup.sh`                      | Schritt 1 von `./update.sh`                    |
| Pipeline      | `pg_dump → gzip → AES-GCM`                                | `pg_dump → gzip → AES-GCM` (in EINER Pipe) | `pg_dump → gzip`, AES-GCM in Schritt 5         |
| Verschlüsselt | **ja** (`BACKUP_ENCRYPTION_KEY`)                          | **ja** (seit 23.08.2026)                   | **ja, nach gesundem Deploy** (seit 23.08.2026) |
| Dateirechte   | 0600                                                      | 0600 (seit 06.08.2026)                     | 0600 (seit 06.08.2026)                         |
| Rotation      | letzte 14                                                 | 7 Tage (`.enc`) / 2 Tage (Klartext)        | 30 Tage (`.enc`) / 2 Tage (Klartext)           |
| Dateiname     | `backup_<Zeitstempel>.sql.gz.enc`                         | `bibliothek_backup_<Datum>.sql.gz.enc`     | `vordeploy_<Zeitstempel>.sql.gz.enc`           |
| Ablage        | Volume `bibliothek_backups` (`/app/backups` im Container) | `./backups` auf dem Host                   | `./backups` auf dem Host                       |

**Warum `update.sh` erst in Schritt 5 verschlüsselt.** Seine Vorab-Sicherung ist der
Rückweg für genau das Zeitfenster, in dem der neue Container nicht hochkommt — und in dem
damit auch das Verschlüsselungswerkzeug nicht erreichbar wäre. Sie bleibt deshalb im
Klartext, bis Gesundheits- und Commit-Prüfung bestanden sind (Schritt 4/4b); erst dann
verschlüsselt Schritt 5 sie und löscht den Klartext. Der dokumentierte Rollback-Weg
`gunzip … | psql` bleibt für dieses Fenster unangetastet.

**Der Rückweg wird bewiesen, nicht angenommen.** Bevor ein Klartext-Dump gelöscht wird
(und bevor `backup.sh` „erfolgreich" meldet), geht die fertige `.enc`-Datei durch das
`restore-backup` im Container. Die bloße Formprüfung genügt nicht: Läuft die Platte
während des Schreibens voll, trägt die abgeschnittene Datei ihre `BKDF`-Kennung und hat
plausible Größe — auffallen würde der Verlust erst beim Wiederherstellen.

**Klartext als Ausnahme.** Beide Skripte fallen darauf zurück, wenn die Verschlüsselung
nicht möglich ist (Backend-Container aus, `BACKUP_ENCRYPTION_KEY` nicht gesetzt, Image
älter als 23.08.2026) — kein Abbruch, denn ein lesbares Backup ist besser als keines. Sie
sagen es dann laut; die Datei löscht erst ein späterer Lauf eines der beiden Skripte,
frühestens nach 2 Tagen (eine Uhr, die ohne Lauf löscht, gibt es nicht). Wiederherstellung und
Restore-Probe: [resilience_and_recovery.md](resilience_and_recovery.md).

---

## 4. Deployment (`./update.sh`)

**Der Weg, der benutzt wird.** Auf dem Server:

```bash
git pull && ./update.sh
```

`git pull` zuerst und getrennt, weil `update.sh` sich sonst in der gerade laufenden
Fassung selbst aktualisiert.

Führt aus: Backup (siehe oben) → `git pull` → `docker compose up -d --build` →
Gesundheitsprüfung (Docker-Status **und** `/health` im Container) → alte Backups
aufräumen. Bei Fehlschlag: Abbruch mit Rollback-Anleitung, die auf das eben erzeugte
Backup zeigt und auf den Commit des laufenden Images (`GIT_COMMIT`) — nicht auf HEAD, das
nach dem vorgezogenen `git pull` schon der neue Stand ist.

`scripts/deploy.sh` ist der ältere, schlankere Weg (`git pull` →
`docker compose up -d --build` → prüft, ob der Domain-Block im Caddyfile steht, und
hängt ihn ggf. an). Er macht **kein** Backup und **keine** Gesundheitsprüfung.

`./update_caddy.sh` schreibt die Caddy-Konfiguration des Servers neu
(`/root/caddy/Caddyfile`, alle Dienste des Hosts) und lädt sie per `caddy reload`
neu, ohne Caddy neu zu starten. Die Datei
`Caddyfile` im Repo-Root ist nur eine Vorlage zum Nachschlagen — sie wird nirgends
ausgeliefert.

### `scripts/stack-neu.sh` — dasselbe lokal, mit Beweis

```bash
./scripts/stack-neu.sh
```

Baut den Entwicklungs-Stack (`docker-compose.local.yml`) neu und **belegt**, dass der
Container danach den aktuellen Stand ausliefert. `docker compose up -d --build` bricht
bei einem fehlgeschlagenen Build zwar ab, lässt aber den **alten** Container
weiterlaufen — wer nur die letzte Ausgabezeile liest, sieht „Container Started" und
misst anschließend eine Fassung, die es im Repo nicht mehr gibt (am 10.08.2026 war die
e2e-Suite deshalb grün für einen Knopf, den es nicht mehr gab).

Der Beweis läuft über den Bundle-Hash: Vite hängt an jeden Bundle-Namen einen Hash über
den Inhalt. Stimmt der Name des ausgelieferten Bundles mit dem des lokalen Builds
überein, liefert der Container exakt diesen Stand. Zeitstempel und Image-IDs
beantworten die Frage **nicht** — sie ändern sich auch ohne Inhaltsänderung und bleiben
gleich, wenn ein Cache-Layer den alten Stand konserviert.

Seit dem 07.09.2026 bricht das Skript mit Exit 2 ab, solange `frontend/.e2e-hauptlieferant`
liegt — der Merkzettel einer laufenden E2E-Suite (`e2e/global-setup.js`). Ein Neubau
mittendrin zieht dem Lauf den Container unter den Füßen weg; die eine Hälfte der Specs
misst den alten, die andere den neuen Stand. Meist ist es eine **zweite Sitzung**, nicht
die eigene. Ist sicher kein Playwright mehr aktiv: Datei löschen oder `STACK_NEU_TROTZDEM=1`.

Das Skript legt denselben `GIT_COMMIT` ins Image wie `update.sh` auf dem Server, sodass
auch lokal jederzeit gilt:

```bash
docker exec bibliothek-backend-local printenv GIT_COMMIT
```

---

## 5. Concurrency-Lasttest (`cmd/stresstest`)

Simuliert Race Conditions für parallele Barcode-Scans.

```bash
go run cmd/stresstest/main.go -port 8084
```

- Feuert via `sync.Cond` + Goroutinen zeitgleich Dutzende Requests gegen `/api/action`
- Zweck: Verifikation der Transaktionssicherheit (FOR UPDATE + Unique Partial Index)

---

## 6. Paket-Utilities (`pkg/`)

### `pkg/csvutil`

CSV-Formel-Injection-Schutz (OWASP CWE-1236):

```go
import "bibliothek/pkg/csvutil"

safeRow := csvutil.SanitizeRow([]string{titel, autor, ...})
```

Setzt Apostroph-Präfix bei Zellen die mit `= + - @ \t \r \n` beginnen.

### `pkg/imageutil`

Decompression-Bomb-Guard:

```go
import "bibliothek/pkg/imageutil"

if err := imageutil.GuardImageDimensions(r.Body, 50_000_000); err != nil {
    // Bild zu groß oder ungültig
}
```

Liest nur den Bild-Header (`image.DecodeConfig`) — ohne volle RAM-Allokation. Limit: 50 Megapixel.

---

## 7. Übrige Skripte in `scripts/` (Kurzübersicht)

Bis zum 05.08.2026 beschrieb dieses Dokument nur die großen Werkzeuge; die folgenden
Skripte lagen undokumentiert im Verzeichnis. Sie sind bewusst kurz gehalten — der
ausführliche Kommentar steht jeweils im Dateikopf. Die Generalprobe (`scripts/generalprobe/`)
steht in Abschnitt 1b.

### Qualitäts-Gates (lokal, es gibt dafür keinen CI-Job)

| Skript                | Zweck                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| --------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `api_inventar.sh`     | Erzeugt `docs/api_inventar.md`: alle registrierten Go-Routen, alle `/api/`-Aufrufer im Frontend und den Abgleich in beide Richtungen (tote Handler / Geister-Aufrufe).                                                                                                                                                                                                                                                                                                                    |
| `deadcode_gate.sh`    | Gate gegen unerreichbaren Go-Code (`x/tools/cmd/deadcode`), Erreichbarkeit ab allen `main`-Paketen; nur von Tests erreichter Code zählt mit, begründete Ausnahmen stehen in `deadcode_baseline.txt`. Tote **Interface**-Methoden sieht das Werkzeug nicht — dafür läuft `tote_tueren_test.go` in jedem `go test`.                                                                                                                                                                         |
| `govulncheck-gate.sh` | Bekannte Schwachstellen in Go-Abhängigkeiten, aufrufbezogen — mit einer Ausnahmeliste, die sich nicht totstellen kann (`security/vuln-ausnahmen.json`, seit 17.09.2026, läuft im pre-push). Eine Ausnahme braucht Nachweis als Test im Repo und eine Wiedervorlage; das Gate wird von allein rot, wenn eine Schwachstelle NICHT in der Liste steht, wenn eine Wiedervorlage abgelaufen ist oder wenn ein Eintrag gar nicht mehr gemeldet wird (dann gibt es einen Fix und die Ausnahme gehört gelöscht).                                                                              |
| `tag-gate.sh`         | Das Tag-Gate (seit 21.09.2026): prüft für einen Commit, ob ALLE Pflicht-Prüfläufe grün sind — die Jobs aus `ci.yml` und die vier Security-Jobs. Eine Liste für Release (`release.yml`) und versioniertes Image (`docker-publish.yml`); jeder Lauf eines Namens zählt, ein fehlender Name ist ein Fehler. Am echten Commit nachstellbar: `GITHUB_REPOSITORY=uuuxy/bibliothek GITHUB_SHA="$(git rev-parse origin/main)" ./scripts/tag-gate.sh`. Die Liste hält `docs/umgebung_paritaet_test.go` gegen beide Workflows.                                                                  |
| `sonar_scan.sh`       | SonarQube-Analyse **inklusive beider** Coverage-Berichte (Go + Frontend-lcov; seit 23.08.2026 erzeugt Schritt 2 `npm run test:coverage` und bricht bei roten Frontend-Tests ab). Ein bloßer `sonar-scanner`-Aufruf lädt keine Coverage hoch — fehlende Coverage zählt dort als 0 %. Braucht `SONAR_TOKEN` in der Umgebung (nie als `-Dsonar.token=`, das stünde in `ps`). **Vorher `TEST_DATABASE_URL` setzen** — siehe unten, sonst misst der Lauf rund 13 Punkte zu niedrig.            |
| `install-hooks.sh`    | Installiert `scripts/git-hooks/` (pre-commit, pre-push) in `.git/hooks`.                                                                                                                                                                                                                                                                                                                                                                                                                  |
| `backup_krypto.sh`    | Kein eigenständiges Skript, sondern der gemeinsame Verschlüsselungs-Helfer von `backup.sh` und `update.sh` (`source`). Prüft, ob verschlüsselt werden kann, reicht Daten durch `cmd/encrypt-backup` im Backend-Container und beweist am fertigen `.enc` den Rückweg über `restore-backup`, **bevor** ein Klartext-Dump gelöscht wird.                                                                                                                                                     |
| `../security-scan.sh` | Sammel-Scan im **Repo-Root**: `gosec` (SAST), `trivy fs` (Abhängigkeiten/Konfiguration), OWASP-ZAP-API-Scan gegen `/swagger/doc.json` des lokalen Stacks (Port 8084, Swagger gibt es nur dort). Der ZAP-Teil braucht den laufenden Stack und Docker; mit `ADMIN_TOKEN` (Wert des Cookies `session_token` nach der Anmeldung am lokalen Stack) geht das Token als Cookie `session_token` mit, ohne läuft er unangemeldet — er ist kein stiller Durchläufer, sondern eine bewusste Sitzung. |

#### Warum die Coverage niedriger aussieht, als sie ist

Gemessen am 06.08.2026, dreimal dieselbe Codebasis:

| Lauf                                         | Gesamtabdeckung |
| -------------------------------------------- | --------------- |
| `go test ./...` ohne Datenbank               | **32,5 %**      |
| … mit `TEST_DATABASE_URL`                    | **45,2 %**      |
| … und ohne die Fremddatei aus `node_modules` | **45,9 %**      |

Zwei Messfehler, kein Codefehler:

1. **58 Dateien `*_pg_test.go` überspringen sich ohne Datenbank** — still, mit „ok" in
   der Ausgabe. Ihr Produktivcode zählt dann als ungedeckt. Das sind rund 13 Punkte.
2. **`frontend/node_modules/flatted/golang/pkg/flatted/flatted.go`** ist eine fremde
   Go-Datei in einem JS-Paket. `go list ./...` führt sie als Projektpaket — Go kennt
   `node_modules` nicht als Sonderfall. 115 ungedeckte Zeilen im Profil.
   `sonar_scan.sh` filtert sie seit dem 06.08.2026 über `go list | grep -v node_modules`.

**100 % sind kein Ziel und wären kein gutes.** Der Rest verteilt sich so: `cmd/*`
(bei dieser Messung sechs CLI-Werkzeuge, 0 %; am 13.09.2026 sind es neun) und `internal/smtptest` (Testserver, wird von Tests benutzt
statt getestet) sind strukturell ungedeckt und sollen es bleiben. Das Quality Gate misst
deshalb **neuen** Code gegen 80 % — die richtige Frage ist nicht „wie hoch ist die Zahl",
sondern „ist das, was ich gerade geändert habe, abgesichert".

Echte Zahlen erzeugen:

```bash
docker run -d --name biblio-test-pg -e POSTGRES_PASSWORD=test \
  -e POSTGRES_DB=bibliothek_test -p 55432:5432 postgres:18-alpine
export TEST_DATABASE_URL="postgres://postgres:test@localhost:55432/bibliothek_test?sslmode=disable"
SONAR_TOKEN=sqp_… ./scripts/sonar_scan.sh
docker rm -f biblio-test-pg
```

Der DB-Name **muss** „test" enthalten (Sicherheits-Notbremse in `pgtest_support_test.go`
vor dem `DROP SCHEMA`).

### Datenbank-Helfer

| Skript                            | Zweck                                                                                                                                                                                                                                                                                                                                                                 | Vorsicht                                                                                                                                                                                                                                                                                      |
| --------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `seed_demo.sql`                   | Realistischer Demo-Datensatz für Pilot und Schulung.                                                                                                                                                                                                                                                                                                                  | Nur auf Test-/Demo-Datenbanken.                                                                                                                                                                                                                                                               |
| `entferne_demo_daten.sql`         | Entfernt den Datensatz aus `seed_demo.sql` von einem benutzten System: Schüler `DEMO-S-*`, Exemplare `DEMO-B-*` samt Ausleihen, Schäden, Bescheiden, Inventur-Verlusten und Protokollzeilen. Die Vorschau listet jede Verflechtung mit echten Daten einzeln (echtes Exemplar an Demo-Schüler verliehen, echter Schüler mit Demo-Exemplar in der Historie, Abholfach). | **Löscht Daten.** Vorher Backup. Ohne Schalter nur Vorschau (endet mit `ROLLBACK`); zum Löschen `-v ausfuehren=ja` (endet mit `COMMIT`). Den Block am Anfang von `seed_demo.sql` dafür nicht nehmen: Er bricht an Schäden ab und lässt Bescheide und Verluste ohne Bezug stehen.              |
| `seed_loadtest.sql`               | Datenbestand für den k6-Lasttest.                                                                                                                                                                                                                                                                                                                                     | Nur auf Wegwerf-Datenbanken.                                                                                                                                                                                                                                                                  |
| `repair_titel_ortssuffix.sql`     | Entfernt Ortssuffixe aus Titelfeldern (Import-Artefakt).                                                                                                                                                                                                                                                                                                              | Vorher Backup.                                                                                                                                                                                                                                                                                |
| `repair_fach_kategorie.sql`       | Einmalige Reparatur (03.09.2026) für Bestände aus dem CSV-/Excel-Import vor diesem Datum: Litteras Kategorie-Spalte („Buch Pg/Kaf 078829 1. Aufl.“) war ungeprüft als Fach übernommen worden — 1.677 Schein-Fächer in der Systematik. Behält nur kanonische Fächer, zieht Schreibvarianten („Mathe“) zusammen, löscht verwaiste Nicht-Fächer.                         | **Ändert Daten.** Ohne Schalter nur Vorschau (endet mit `ROLLBACK`); zum Schreiben `-v ausfuehren=ja` an den psql-Aufruf hängen (endet mit `COMMIT`). Die letzte Ausgabezeile sagt, was passiert ist. Löscht auch handgepflegte Fächer — die Ausgabe „wird gelöscht" führt die Titelzahl mit. |
| `repair_klasse_nach_135.sql`      | Einmalige Reparatur (22.09.2026) für den Testserver: Migration 135 lief dort vor ihrer Rücknahme (7981e347), löschte `grade_level` und setzte 129 Titeln die Klasse als einjährige Spanne. Holt Klasse und Spanne aus der Vorab-Sicherung `vordeploy_20260922_153818` zurück; setzt nur Spannen zurück, die noch genau der Faltung entsprechen. | **Ändert Daten.** Ohne Schalter nur Vorschau (endet mit `ROLLBACK`), mit `-v ausfuehren=ja` `COMMIT`. Vorbereitung (Sicherung entschlüsseln, Hilfsdatenbank, CSV) und Aufräumen stehen im Kopf der Datei. |
| `signatur_report.sql`             | Report zur Signatur-Harmonisierung nach Littera-Import (Migration 038).                                                                                                                                                                                                                                                                                               | Nur lesend.                                                                                                                                                                                                                                                                                   |
| `e2e_altlasten.sql`               | Entfernt den Bestands-Bodensatz der E2E-Suite (Titel mit Präfix `E2E `, deren Exemplare und Ausleihen). Ohne Schalter nur Vorschau; zum Löschen `-v ausfuehren=ja`.                                                                                                                                                                                                   | **Löscht Daten.** Vorher Backup. Ohne Schalter nur Vorschau; zum Löschen `-v ausfuehren=ja` an den psql-Aufruf hängen. Die letzte Ausgabezeile sagt, was passiert ist — von Hand am COMMIT zu drehen ist nicht mehr nötig.                                                                    |

**Warum der Bestand von Hand aufgeräumt wird, Lieferanten aber automatisch:** Der globale
Teardown der E2E-Suite (`frontend/e2e/global-teardown.js`) räumt nach jedem Lauf
Lieferanten, Testbestellungen und Klassensatz-Reservierungen ab — dort kann ein zu weites
Muster wenig anrichten. Bestand und Ausleihen sind eine andere Klasse:
`ausleihen.exemplar_id` und `schadensfaelle.exemplar_id` stehen auf **RESTRICT**, ein
Löschen bräuchte also eine Kette über vier Tabellen, darunter die Ausleihhistorie. Eine
solche Kette automatisch nach jedem Lauf feuern zu lassen, ist das Aufgeräumtsein nicht
wert — ein Fehler im Zuschnitt löscht dort echte Daten. Gemessen am 12.08.2026: Ein
vollständiger Lauf hinterlässt rund 42 Titel, 75 Exemplare und 22 Ausleihen.

Benutzerkonten der Suite räumt seit dem 12.08.2026 die erzeugende Spec selbst weg
(`admin-mail-config.spec.js`, `afterAll`) — dort ist das Muster eindeutig und die
Fremdschlüssel stehen auf `SET NULL`.

### Einmal-Werkzeuge (`//go:build ignore`, per `go run` gestartet)

| Skript             | Zweck                                                                      |
| ------------------ | -------------------------------------------------------------------------- |
| `import_isbns.go`  | Nachträglicher ISBN-Import in bestehende Titel.                            |
| `monitor_stats.sh` | Protokolliert Systemkennzahlen über ~6 Stunden (Begleitung von Lasttests). |

> `scripts/migrate_photos.go` stand hier bis zum 11.08.2026. Es war ein Doppel von
> `cmd/migrate-fotos` (Abschnitt 2) und wurde entfernt — zwei Wege in dieselbe
> verschlüsselte Ablage, von denen nur einer gepflegt wurde.

### Testdaten-Generator (`cmd/seed`)

Füllt eine Datenbank mit Test-Admin, Schülern, Titeln und Exemplaren — die Vorstufe zum
k6-Lasttest (Abschnitt 5). Liest `DATABASE_URL` und `JWT_SECRET` aus der Umgebung, kennt
keine Flags und fragt **nicht** nach, bevor es schreibt. Ein zweiter Lauf legt nichts
doppelt an und nennt dasselbe Scanner-Konto (`cmd/seed/main_pg_test.go`):

```bash
DATABASE_URL="postgres://…/bibliothek_test" go run ./cmd/seed
```

> **Nur auf Wegwerf-Datenbanken.** Das Werkzeug legt Massendaten an und prüft vorher
> nicht, ob die Zieldatenbank leer ist. Auf einem Echtbestand vermischen sich Testdaten
> unrettbar mit echten Schülern.

## `pruefe_secrets.sh` — Konfigurationsprüfung vor dem Deploy

```bash
./scripts/pruefe_secrets.sh                 # nutzt ./.env
./scripts/pruefe_secrets.sh /root/bibliothek/.env
```

Liest eine `.env` und meldet die Fehlkonfigurationen, die **still** bleiben: bekannte
Default-Secrets aus dem Repository (auch die Beispielwerte aus `.env.example`), fehlender
`BACKUP_ENCRYPTION_KEY` (der nächtliche Job überspringt sich dann kommentarlos),
`IMAP_HOST=mock` (akzeptiert jedes Passwort), `APP_ENV` auf `local`, `development` oder
`test` (lässt Beispielwerte und `mock` gelten), ausdrückliches
`ENFORCE_PROD_SECRETS=false` oder ungesetztes `COOKIE_SECURE`. Was das Skript meldet,
hält `docs/pruefe_secrets_test.go` fest; am Server läuft es nur von Hand.

Das Skript **ändert nichts**. Exit-Code 0 = sauber, 1 = kritischer Befund.

Bei einem Treffer auf `APP_ENCRYPTION_KEY` den Schlüssel **nicht einfach ersetzen** —
Schülerfotos und das SMTP-Passwort wären verloren. Der Weg mit Umschlüsselung steht in
[SECURITY.md](SECURITY.md#app_encryption_key-wechseln).
