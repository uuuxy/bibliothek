# Landesmittel und Kreismittel — Konzept

Stand: 16.10.2026

**Teil A:** Schadensersatz für verlorene und beschädigte Bücher (Abschnitte 1–6).
**Teil B:** Getrennte Töpfe in der Beschaffung — Bestellung, Rechnung, Berichte (Abschnitt 7).
Beide Teile teilen EIN Vokabular für die Mittelherkunft: `land` (Lernmittelfreiheit) und
`schultraeger` (Schülerbücherei) — `pkg/mitteltopf`, `bestellungen_verlauf.mittel`.

**Was gebaut ist:** Teil A im ersten Schnitt — der Bescheid als Brief, Datenmodell mit
Nummernkreis, Staffel, Einstellungen, Erstellen aus dem Mahnwesen, Nachdruck, Übergabe und
der Rückgabe-Hook; dazu die Bewertung: Listenpreis am Titel (Migration 127, aus der DNB
gefüllt), Beschädigungsgrad am Exemplar, der gerechnete Betragsvorschlag samt Herleitung in
jedem Weg — Melde-Dialog, Bescheid aus einer Forderung, Bescheid aus einer überfälligen
Ausleihe — und die wählbare Berechnungsgrundlage. Offen sind aus Abschnitt 4.7 das
Übergabe-PDF (Etappe 2), die Rechnung für den Schulträger (Etappe 3) und der Rest der
Etappe 4 (Altbriefe abräumen). Die ausgearbeiteten Bauschritte dazu stehen in einer
früheren Fassung von OFFEN.md (4.4, 5.3, 5.4): `git show c8049cf8:docs/OFFEN.md`.
Teil B ist im ersten Schnitt gebaut (Abschnitt 7.3).

**Wie der Bescheid eingebaut ist (drei Entscheidungen, abgesprochen am 10.09.2026):** Der
Bescheid entsteht in der bestehenden Auswahlleiste des Mahnwesens und nur bei GENAU EINEM
markierten Schüler — jeder Betrag ist Ermessen, jede Referenznummer unwiderruflich. Die
Briefe stehen im vierten Reiter derselben Reiterzeile (Alle → Akut fällig → Eskaliert →
Schadensersatz), zusammen mit den Kindern, die einen Verlust oder Schaden gemeldet haben und
noch keinen Brief; die Zahl am Reiter zählt, was bei der Schule liegt. Der Erstell-Schritt
ist EIN Basis-Dialog ohne Schritte (Material 3: Vollbild ist die Form für Telefone, einen
Stepper gibt es dort nicht) und zeigt genau das, was ein Mensch entscheidet: welche Bücher,
welcher Betrag, mit der Herleitung daneben.

**Am fertigen PDF prüfen:** Zwei Fehler zeigen erst die Seiten selbst, auch bei grünen
Gates: ein Zeichen, das der Zeichensatz des PDFs nicht kennt („Ayşe" würde zu „Ay.e"), und
ein Seitenumbruch, der die Kopfzeile der Tabelle trennt. Beide Klassen haben ein Gate; für
das zweite liest `internal/pdftest` die Seiten getrennt (`TexteJeSeite`).

**Grundlage** sind vier Unterlagen der Schule: die Verfahrensbeschreibung für den
Schadensersatz bei Lernmitteln (Stand 2014), die Arbeitshilfe dazu, das verbindliche
Musteranschreiben an die Erziehungsberechtigten und eine Anforderungsliste
„Mahnverfahren". Das Verfahren steht in der bestehenden Anwendung und nimmt so viel
Bestehendes wie möglich. Zwei Randbedingungen der Schule: **Den Brief schickt das
Sekretariat** (im System als Admin angemeldet), nicht die Lernmittel-Stelle. Und es
braucht **zwei Rechnungen**, weil zwei Töpfe: Lernmittel gehören dem Land, die
Schülerbücherei ist aus Mitteln des Schulträgers beschafft.

---

## 1. Was das Verfahren der Schule verlangt (aus den Unterlagen)

### 1.1 Lernmittel (Landesmittel)

- Lernmittel bleiben Eigentum des Landes und sind spätestens beim Verlassen der Schule
  zurückzugeben. Bestehen und Höhe eines Ersatzanspruchs setzt die Schule mit einem
  förmlichen Bescheid fest.
- Bücherei-Bestände sind keine Lernmittel. Zugangsbuch und Bestandskartei sind die
  Bestandsverzeichnisse; die Bestandskartei weist Ausleihe, Rücklauf **und
  Aussonderungen** nach — also nie löschen, immer aussondern. Zustand bei Ausgabe UND
  Rückgabe prüfen.
- Bei Verlust oder Beschädigung: Sacherstattung oder ein Geldbetrag „in angemessener
  Höhe", Maßstab ist der Wiederbeschaffungswert. **Das Geld gehört auf das Konto des
  Landes für Ersatzleistungen**, die Schule bekommt es von der Schulaufsicht erstattet.
  Bei Weigerung geht der Fall an die Schulaufsicht, die den Anspruch durchsetzt.
- Barzahlung darf nur ausnahmsweise vereinnahmt werden (Quittung, binnen 14 Tagen
  weiterleiten). Auf dem Konto des Landes dürfen keine Mittel des Schulträgers geführt
  werden.
- Musteranschreiben und Arbeitshilfe: Referenznummer = Bereichs-Nr. (4) · Kassenjahr (4)
  · Schulnummer (4) · laufende Nr. (4), **je Brief, nie doppelt**. Vierwochenfrist **mit
  Datum**. Staffel über Verleihjahre, bezogen auf den **Neupreis zum Zeitpunkt des
  Verlusts** (nur im ersten Jahr auf den Kaufpreis): 1→100 %, 2→80 %, 3→60 %, 4→40 %,
  5→20 %, 6+→10 %. Die Arbeitshilfe schreibt „Nach 5 Jahren und für jedes weitere Jahr …
  insgesamt 10 %" — also ab dem SECHSTEN Verleihjahr. Zwei Fallgruppen (nicht
  zurückgegeben / so stark beschädigt, dass unbenutzbar), Unterschrift der Schulleitung,
  vorgeschriebener Schlussabsatz mit Einspruchsfrist von einem Monat. Nach Fristablauf
  Original + Buchungsbeleg an die Schulaufsicht, Kopie bleibt; **spätere Rückgabe →
  Schulaufsicht unverzüglich informieren.**
- **Keine Sperre bei Lernmitteln:** Die Sperre weiterer Ausleihen bis zur Zahlung steht in
  keiner der vorliegenden Unterlagen des Landes. Die Schule hat am 22.09.2026 entschieden:
  Für Lernmittel gibt es keinerlei automatische Sperre, auch keine übergehbare; für die
  Schülerbücherei bleibt sie. Am 28.09.2026 entschieden: auch nicht die der Ehemaligen und
  auch nicht bei der Verlängerung. Gebaut in
  `internal/service/ausleih_sperren.go` (`pruefeAusleihSperren`, `PruefeSperreAmLeser`). „Nur Überweisung, keine Barzahlung" ist
  dagegen belegt (Arbeitshilfe, Abschnitt 1).

**Zwei Dinge, die die Schule wissen muss:**

1. Die vorliegenden Unterlagen sind von **2014**. Das Musteranschreiben nennt eine Stelle,
   die es seit 2015 nicht mehr gibt; die aktuell geltende Fassung des Verfahrens ist nicht
   öffentlich. Das Sekretariat sollte sie samt aktuellem Musterschreiben bei der
   Schulaufsicht anfordern. Die Bauweise unten hält den Brieftext an EINER Stelle
   austauschbar; Struktur, Nummer, Konto und Frist dürften unverändert sein.
2. Barzahlung „in der Bibliothek" ist für Landeseigentum die **Ausnahme, nicht der Weg**.
   Die Rechnung nennt für Lernmittel das Konto des Landes (`pdf/zahlungsweg.go`); der
   Mahnbrief verlangt kein Geld. Bar angenommenes Geld bleibt möglich — als Einzelfall mit Quittung, nicht als
   Weg, den ein Brief vorsieht.

### 1.2 Schülerbücherei (Mittel des Schulträgers)

- Errichtung und Ausstattung der Schülerbücherei sind Sache des Schulträgers; ihr Bestand
  gehört ihm.
- Das Benutzungsverhältnis folgt der Benutzungsordnung der Schule: zuerst
  Ersatzbeschaffung, sonst Geld in Höhe des Neuwerts.
- „Neuwert" ist der LISTENPREIS, nicht der Einkaufspreis — was ein Ersatz heute kostet. Mit
  dem Kaufpreis würde ein 2015 für 8 € beschaffter Roman mit 8 € ersetzt, obwohl er heute
  14 € kostet (`ersatzwert.BerechneNeuwert`). Die Staffel des Landes gilt hier NICHT (kein
  Altersabschlag), ein erfasster Beschädigungsgrad dagegen schon. Zur Schülerbücherei sagt
  weder der Erlass vom 17.12.2014 noch die Arbeitshilfe etwas — beide sprechen
  ausschließlich von Lehrwerken der Lernmittelfreiheit.

Folge: Für den Bestand der Schülerbücherei gibt es **keinen förmlichen Bescheid, keine
Referenznummer im Landesformat** — sondern eine gewöhnliche Rechnung/Zahlungsaufforderung
der Schule, deren Geld an den Schulträger gehört. Wohin genau (Kasse des Trägers mit
Kassenzeichen, Budgetkonto der Schule, bar mit Quittung), ist mit dem Schulträger zu
klären. Sicher ist nur: nicht auf das Konto des Landes.

---

## 2. Was die Anwendung kann (am Code nachgesehen)

| Baustein                             | Fundstelle                                                                                                                                                    | Stand                                                                                                                                                              |
| ------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Forderung anlegen                    | `repository/schaden.go` (`SchadensAbfragen.Melde` beendet Ausleihe + sondert aus `BESCHAEDIGUNG`), `SchadenMeldenDialog.svelte` | ✅ inkl. Idempotenz, Schuldner aus der Ausleihe, 409 bei Neuverleih. Der Betrag ist ein gerechneter VORSCHLAG mit Herleitung — überschreibbar, denn er liegt im Ermessen der Schule.               |
| Forderung erledigen                  | `api/schaden/forderung_erledigen.go`, `repository/forderung_erledigen.go` (Bezahlt / Storno mit Grund, Audit, 409 bei Doppelbuchung)                                              | ✅ „Bezahlt" bedeutet **Barzahlung an der Theke**.                                                                                                              |
| Brief 2: Rechnung je Schüler         | `pdf/rechnung.go`, `api/schaden/rechnung.go` (alle offenen Forderungen)                                                                                                  | DIN 5008, LEFT JOINs (Geräteschaden vorgesehen), Zahlungsweg je Topf (`pdf/zahlungsweg.go`), ohne Nummer und ohne Frist im Landesformat.                  |
| Brief 3: Eltern-Mahnbrief            | `pdf/mahnbrief.go`, Vorlage `MAHNUNG_ELTERN`                                                                                                                  | DIN-5008-Fensterkuvert, Falzmarken, Tabelle der überfälligen Bücher — **die Bauform, die der Bescheid braucht.**                                                   |
| Mahnwesen                            | `api/mahnwesen/`, `repository/mahnbriefe.go`, `Mahnwesen.svelte` + `components/mahnwesen/`                                                                | Überfällige nach Klasse, Auswahl → „Mahnbriefe drucken" (Mahnstufe steigt nur beim Druck), Klassenleitungs-Mail, Sperre ab `max_overdue_days` (nur Schülerbücherei). |
| Schulstammdaten                      | `SchuleKategorie.svelte`, `einstellungen*.go`                                                                                                                 | Name, Anschrift, Eigentumsvermerk; die Angaben des Bescheids (Bereichs-Nr., Schulnummer, Aufsicht, Schulleitung, Geschäftszeichen, Zahlstelle, Bankverbindung) in der Kategorie „Schadensersatz" (`repository/einstellungen.go`). |
| Preise                               | `buecher_exemplare.einkaufspreis` (aus Littera übernommen), `erworben_am` = Littera-Zugangsdatum (echt, nicht Importdatum), `buecher_titel.listenpreis` und `buecher_exemplare.zustand_abwertung_prozent` (Migration 127) | ✅ Der Listenpreis kommt beim Anlegen über die ISBN aus der DNB (MARC21 020 $c) und ist von Hand überschreibbar; leer heißt „nicht erfasst", dann weicht die Staffel auf den Kaufpreis aus und sagt es. |
| Ausleihhistorie je Exemplar          | `ausleihen` (Zeilen bleiben nach der Anonymisierung ohne Person)                                                                                              | Zählbar. Aus Littera kamen nur die **offenen** Ausleihen — für den Altbestand ist die Zahl der Verleihjahre unbekannt.                                             |
| Volljährigkeit                       | `leser.geburtsdatum` (NULL bei Altdaten)                                                                                                                   | Anrede „Erziehungsberechtigte" vs. volljährig ableitbar.                                                                                                           |
| Rollen                               | ADMIN / MITARBEITER / KOLLEGIUM / HELFER; Sekretariat = ADMIN                                                                                                 | Kein eigenes Recht für den Bescheid: Erstellen und Übergeben verlangen `edit_students`, Lesen `view_students` (`api/routen_leser.go`; Grund in 4.7).                 |
| Nummernkreise                        | `barcode_seq`; Regel: EIN Generator, nie recyceln (Register 068)                                                                                              | Die Referenznummer zählt `schadensersatz_nummern` je Topf und Kassenjahr (Migration 110).                                                                                                      |

**Kern:** Die Forderung (`schadensfaelle`) ist die Wahrheit — Sperre (nur Schülerbücherei),
Löschblockade, DSGVO-Auskunft, Zusammenführen, Bezahlt/Storno hängen alle daran. Sie trägt
die **Art** (`schadensfaelle.art`: nicht zurückgegeben / beschädigt); den **Topf** liefert
das Eigentum des Exemplars (`repository.ExemplarTopfSQL`); der **Brief** ist ein eigener
Datensatz mit Nummer, Frist und Status (`schadensersatz_bescheide`), seine Positionen sind
die Forderungen (`schadensfaelle.bescheid_id`).

---

## 3. Anforderungsliste „Mahnverfahren" — Abgleich

| Nr. | Wunsch                                                                                                              | Bewertung                                                                                                                                                                                                                                                                                                                    |
| --- | ------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Automatische Abwertung (10 %/Jahr oder je Ausleihe)                                                                 | ✅ Gebaut ist die Staffel des Erlasses selbst, sichtbar an jedem Exemplar („Ersatzwert heute" in der Buchakte): Man will vor dem Schadensfall wissen, was ein Buch noch wert ist. Ein ZWEITER, frei eingestellter Abwertungssatz wäre falsch (zwei Beträge für dasselbe Buch, einer davon im Bescheid). |
| 2   | Beschädigung mit Prozentwert bei Katalogisierung/Rückgabe                                                           | ✅ `zustand_abwertung_prozent` am Exemplar, Feld im Zustands-Dialog der Buchakte, wirkt auf jeden künftigen Ersatzbetrag (auch für den Bücherei-Bestand). Der Wert ist ein ZUSTAND, keine Forderung — er erzeugt von sich aus keinen Betrag. Nicht an der Theke: Die ist ein Scanfeld, kein Formular (entschieden am 17.09.2026); erfasst wird nach der Rückgabe in der Buchakte. |
| 3   | Einkaufspreis UND Listenpreis, wählbar                                                                              | ✅ Migration 127 und die Einstellung „Immer mit dem Einkaufspreis rechnen" in der Kategorie Schadensersatz. Aus = der heutige Listenpreis, wie es die Arbeitshilfe verlangt; an = immer der Einkaufspreis. Die Herleitung nennt, welcher Preis gegolten hat, und unterscheidet „kein Listenpreis hinterlegt" von „so eingestellt". |
| 4   | Übersicht Verluste/Beschädigungen                                                                                   | Gibt es (Mahnwesen, Fehlbestand, Gebühren in der Akte). Neu: Liste der Bescheide mit Fristablauf.                                                                                                                                                                                                                            |
| 5   | Versand per Post/E-Mail/App; Referenznummer, jährlich neu                                                           | **Nur Druck/Post.** Das Verfahren verlangt Schriftform; Datenschutz-Entscheidung A3 vom 22.08.2026 (keine Eltern-Mahnmail). Referenznummer ja; „Zurücksetzen zu Jahresbeginn" ergibt sich, weil das Kassenjahr Teil der Nummer ist.                                                                                          |
| 6   | Zahlung bestätigen; Rückgabe → Buch wieder verfügbar, Saldo null; Zahlung ohne Rückgabe → „verloren" + **gelöscht** | Bezahlt gibt es. Rückgabe → Forderung automatisch stornieren (neu). **Gelöscht wird nie**: Die Bestandskartei muss die Aussonderung nachweisen → `ist_ausgesondert`/`aussonderung_grund = 'VERLUST'`, genau wie heute.                                                                                                       |
| 7   | Nach Fristablauf: zweiter Ausdruck für die Schulaufsicht, Rest automatisch „verloren", Saldo bereinigt              | Liste „Frist abgelaufen" + Aktion **„An die Schulaufsicht übergeben"** (Mensch klickt, kein Cron): Original-Nachdruck + Sammelliste, Bücher werden ausgesondert. Automatik über Nacht wäre eine stille Aussonderung — genau die Bugklasse, die dieses Projekt meidet. Der Satz „Der Saldo der betroffenen Leser wird entsprechend bereinigt" beantwortet E6. |
| —   | Frist **sechs** Wochen                                                                                              | Verfahrensbeschreibung und Arbeitshilfe sagen **vier Wochen mit Datum** → Vorgabe 28 Tage, einstellbar (E3).                                                                                                                                                                                                                 |
| —   | Importfelder Ansprechpartner 2/3 (Eltern-Namen, zweite Anschrift getrennt lebender Eltern)                          | **Nicht bauen (E8).** Das Musteranschreiben adressiert „An die Erziehungsberechtigten des/der Schülers/in" — Elternnamen braucht es nicht; die LUSD liefert eine Anschrift; zusätzliche Daten widersprechen der Datensparsamkeit. Ein zweiter Brief an einen getrennt lebenden Elternteil ist ein Handfall des Sekretariats. |

---

## 4. Bauplan (bestehende Struktur, keine Parallelwelt)

### 4.1 Datenmodell (eine Migration)

```
schadensfaelle
  + art            TEXT NOT NULL DEFAULT 'beschaedigung'
                   CHECK (art IN ('nicht_zurueckgegeben','verlust','beschaedigung'))
  + bescheid_id    UUID NULL REFERENCES schadensersatz_bescheide(id)   -- Position eines Briefs

schadensersatz_bescheide           -- EIN Brief = ein Datensatz, zwei Töpfe
  id, schueler_id (RESTRICT), mittel ('land'|'schultraeger'),
  kassenjahr INT, laufende_nr INT, referenznummer TEXT UNIQUE,     -- Land: Bereich·Jahr·Schulnr·lfd; Schulträger: SB-Jahr-lfd
  brief_datum DATE, frist_bis DATE, gesamtbetrag NUMERIC(10,2),
  empfaenger_snapshot JSONB,       -- Anrede (Erziehungsberechtigte|volljährig), Name, Anschrift zum Briefdatum
  status ('offen'|'uebergeben'|'erledigt'), uebergeben_am, erledigt_am,
  schulaufsicht_zu_informieren BOOL, -- Rückgabe NACH Übergabe
  erstellt_von, erstellt_am, letzter_druck_am
  UNIQUE (mittel, kassenjahr, laufende_nr)

schadensersatz_nummern (mittel, kassenjahr, letzte_nr)   -- der EINE Generator, UPDATE … RETURNING in der Tx
```

Die Positionen des Briefs SIND die Forderungen — keine zweite Positionstabelle, damit
Sperre, Löschblockade, Auskunft und Bezahlt/Storno unverändert weiterlaufen. Der Snapshot
des Empfängers macht den Nachdruck reproduzierbar („Kopie verbleibt in der Schule");
er wird bei der Anonymisierung mit getilgt (DSGVO-Paar-Gate), die Nummer bleibt (die
Schulaufsicht ordnet Zahlungen darüber zu). `mittel` ist dasselbe Vokabular wie in
`bestellungen_verlauf` (Teil B, Migration 109).

### 4.2 Betrag: Vorschlag, kein Automat (`pkg/ersatzwert`)

Reine Funktion `Berechne(verleihjahr, kaufpreis, neupreis, zustandAbschlag, quelle) Vorschlag`
nach der Staffel der Schule (mit dem Zustands-Abschlag des Exemplars und der eingestellten
Preisquelle; `BerechneNeuwert` ist derselbe Weg ohne Staffel). `verleihjahr = max(Anzahl Schuljahre mit Ausleihe des
Exemplars, Schuljahre seit erworben_am + 1)` — die zweite Größe fängt den Littera-Altbestand
ohne Historie ab (ein 2019 gekauftes Buch ist nicht im 1. Verleihjahr, nur weil wir seine
Ausleihen nicht kennen). Der Dialog zeigt die Herleitung („3. Verleihjahr → 60 % von
24,90 €") und der Mensch bestätigt oder überschreibt — im Ermessen der Schule.

### 4.3 Schreibpfade

| Weg                                                                                                   | Was passiert                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| ----------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `POST /api/schueler/{id}/bescheide` `{mittel, positionen:[{ausleihe_id \| schadensfall_id, betrag}]}` | Eine Transaktion: Schülerzeile sperren, Positionen prüfen (gehören zum Schüler, Eigentum passt zum Topf via `repository.ExemplarTopfSQL`), für überfällige Ausleihen den Verlust buchen (`bucheVerluste`: die Ausleihe endet, das Exemplar wird als `VERLUST` ausgesondert, die Forderung `art = nicht_zurueckgegeben` entsteht — so gibt es keinen Brief über ein Buch, dessen Ausleihe weiterläuft; kommt es zurück, storniert der Rückgabe-Hook die Forderung), Nummer ziehen, Frist rechnen, Snapshot schreiben, PDF erzeugen, **erst dann committen** (Muster `erzeugeMahnbriefe`: Papier == DB), Audit. |
| `GET /api/bescheide/{id}/pdf`                                                                         | Nachdruck aus dem Snapshot, dieselbe Nummer, `letzter_druck_am`. Nie eine neue Nummer.                                                                                                                                                                                                                                                                                                                                                                   |
| `POST /api/bescheide/{id}/uebergeben`                                                                 | Nach Fristablauf, von Hand: Status `uebergeben`, nur aus „offen", nur nach Fristablauf und nur, solange etwas zu zahlen ist (`Uebergebe`). Ausleihe und Exemplar sind da schon gebucht (beim Erstellen oder Melden). Offen: das Übergabe-PDF, Original + Sammelliste für die Schulaufsicht (Etappe 2).                                                                                                                                                                                                                                          |
| Rückgabe-Hook in `repository/bescheid_rueckkehr.go` (`verbucheRueckkehr`), gerufen aus der Reaktivierung beim Scan (`versucheReaktivierung`) | Hat das Exemplar eine offene Forderung `nicht_zurueckgegeben`: Storno mit Grund „Rückgabe am …"; steht der Brief auf `uebergeben`, wird NICHTS storniert — der Bescheid bekommt `rueckgabe_nach_uebergabe` und die Theke den Satz, dass die Aufsicht unverzüglich zu informieren ist. **Nicht in `repository/ausleihe.go`:** `SchadensAbfragen.Melde` beendet die Ausleihe schon beim Anlegen der Forderung und sondert das Exemplar aus — wenn das Buch zurückkommt, gibt es keine offene Ausleihe mehr, sondern ein ausgesondertes Exemplar am Scanner. Reaktivierung und Storno liegen in EINER Transaktion. |
| Bezahlt / Storno                                                                                      | unverändert (`/api/schadensfaelle/{id}/bezahlt`, `/storno`). Brief gilt als `erledigt`, wenn keine Position mehr offen ist (abgeleitet, nicht doppelt gespeichert).                                                                                                                                                                                                                                                                                      |
| `GET /api/bescheide?status=`                                                                          | Liste für das Sekretariat. Gebaut ist nur `status=offen` (nicht übergeben), ohne Parameter alle; Frist abgelaufen, übergeben und „Rückgabe nach Übergabe" zeigt die Liste als Zustand je Bescheid, nicht als Filter.                                                                                                                                                                                                                                     |

Recht: `edit_students` zum Erstellen und Übergeben, `view_students` zum Lesen
(`api/routen_leser.go`), kein eigenes Recht (4.7). PII-Stufe 3 → PII-Matrix + Antwort-Gate.

### 4.4 Briefe: ein Renderer, zwei Varianten (`pdf/bescheid.go`)

Gebaut auf der DIN-5008-Fensterkuvert-Seite aus `pdf/mahnbrief.go` (Falzmarken,
Anschriftfeld, „(keine Adresse hinterlegt)"-Regel).

- **Land** (Musteranschreiben der Schule): Kopf mit Geschäftszeichen/Bearbeiter/Durchwahl,
  Betreff wörtlich, der verbindliche Text, ☒/☐ je Fallgruppe mit Tabelle (Name, Titel,
  ISBN, Preis), Frist als Datum, Gesamtbetrag, Konto des Landes, Referenz-Nr., Hinweis auf
  die Übergabe an die Schulaufsicht, Unterschrift Schulleitung, vorgeschriebener
  Schlussabsatz. Der verbindliche Text steht **fest im Code an einer Stelle** (kein frei
  editierbares Template — eine zerschossene Vorlage wäre ein fehlerhafter Bescheid);
  Austausch gegen die aktuelle Fassung ist eine Zeile je Absatz.
- **Schulträger**: Rechnung mit Rechnungsnummer, Frist, Tabelle, Hinweis auf
  Ersatzbeschaffung als Alternative, Zahlungsweg aus den Einstellungen — ohne Nummer im
  Landesformat, ohne den Schlussabsatz des Landesbescheids.
- Beide ersetzen `pdf/rechnung.go` und `api/schaden/rechnung.go`; der Elternbrief je
  Schadensfall ist entfernt, seine Spalten mit Migration 164.
  Der Eltern-**Mahnbrief** (überfällig, ohne Geld) bleibt, wie er ist.

### 4.5 Oberfläche

- **Schülerakte → Gebühren & Schäden:** Forderungen tragen Topf und Art; Knopf
  „Bescheid erstellen (Land)" / „Rechnung erstellen (Schulträger)" nur mit Recht; darunter
  die Briefe mit Nummer, Frist, Status, „Nachdruck", „An die Schulaufsicht übergeben".
- **Mahnwesen:** Auswahl-Modus bekommt neben „Mahnbriefe drucken" den Knopf
  **„Schadensersatz-Bescheide (Land)"** — je markiertem Schüler ein Brief über seine
  überfälligen Lernmittel plus offene Lernmittel-Forderungen; die Tabelle des Briefs
  füllt sich aus denselben Daten wie heute der Mahnbrief. Dritter Filter „Bescheide"
  (Frist abgelaufen → Übergabe).
- **Schaden melden** (`SchadenMeldenDialog`): Art (verloren / beschädigt), Staffel-Vorschlag
  mit Herleitung, Betrag überschreibbar; kein automatisches PDF-Popup mehr — der Brief
  ist ein eigener Schritt des Sekretariats.
- **Einstellungen → neue Kategorie „Schadensersatz"** (ein JSON-Schlüssel nach dem
  Muster `sommerferien`, serverseitig geprüft): Bereichs-Nr. und Schulnummer für die
  Referenznummer, Frist in Tagen (28), Konto Land, Konto Schulträger (leer bis geklärt),
  Geschäftszeichen, Bearbeiter, Durchwahl, Name der Schulleitung, Schulaufsicht (Name +
  Anschrift für den Schlussabsatz).
- **Betriebsbereitschaft:** Warnung, solange Schulnummer/Bereichs-Nr./Konten fehlen —
  ein Bescheid ohne Referenznummer ist keiner.

### 4.6 Gates (jedes am Rückbau rot zu sehen)

Nummernkreis nie doppelt und nie rückwärts (PG, parallel) · Rückgabe storniert die
Forderung und markiert nach Übergabe (PG) · Topf ≡ Eigentum jeder Position nach
`ExemplarTopfSQL` (PG, `api/schaden/eigentum_bescheid_pg_test.go`) · Brief-Inhaltsstrom trägt Referenznummer, Frist-Datum, Konto und Schlussabsatz
(Gate am fertigen PDF, wie `etikett-gate`) · Nachdruck ändert weder Nummer noch Betrag ·
Anonymisierung tilgt den Snapshot, lässt die Nummer (DSGVO-Paar-Gate) · Recht am Draht
(`permissions.spec.js`) · Staffel als Tabelle (1→100 % Kaufpreis, 2→80 %, 3→60 %,
4→40 %, 5→20 %, 6+→10 % Neupreis).

### 4.7 Etappen

1. Datenmodell + Nummernkreis + Landes-Bescheid + Einstellungen — **gebaut**
   (Migration 110, `pkg/ersatzwert`, `pdf/bescheid.go`, `repository/bescheid.go`,
   `api/schaden/bescheid.go`, Kategorie „Schadensersatz", Reiter und Dialog im Mahnwesen).
   Abweichungen vom Plan oben, jeweils mit Grund: Der Bescheid entsteht im **Mahnwesen**
   statt in der Schülerakte (dort steht, wer überfällig ist); `schadensfaelle.art` hat
   **zwei** Werte statt drei (genau die zwei Kästchen des Formulars — „Verlust" IST die
   erste Gruppe); `schueler_id` ist **SET NULL** statt RESTRICT (der Brief überlebt die
   DSGVO-Löschung als Beleg ohne Person, RESTRICT hätte die berechtigte Löschung
   blockiert); das Recht ist `edit_students` statt eines neuen — ein eigenes Recht bliebe
   ab Werk bei niemandem und wäre eine Tür, die keiner öffnen kann.
2. Mahnwesen-Anbindung (Auswahl → Bescheide) + Bescheid-Liste + Übergabe + Rückgabe-Hook.
   Gebaut sind Anbindung, Liste, Rückgabe-Hook und die Übergabe als Status. Das Exemplar
   steht schon beim Erstellen des Briefs oder beim Melden auf `VERLUST` (`bucheVerluste`,
   `SchadensAbfragen.Melde`), und ein Exemplar, das im Fehlbestandsbericht als gefunden
   markiert wird (`MarkiereVerlustAlsGefunden`), storniert die Forderung über denselben Weg wie die Theke
   (`verbucheRueckkehr`). Offen: das Übergabe-PDF.
3. Schulträger-Rechnung + Betriebsbereitschaft-Warnung. **Bis dahin nimmt der Server nur
   `mittel = land` an, und jede Position muss ein Buch des Landes sein** (409 an der Tür,
   `ExemplarTopfSQL`) — sonst bekämen Eltern für ein Bücherei-Buch den Landes-Wortlaut mit
   Landeskonto. Beim Bau
   zwingend: Der Topf muss in die Referenznummer oder in ein eigenes Format — der Zähler
   läuft je Topf, `referenznummer` ist aber global UNIQUE, und die heutige Nummer enthält
   keinen Topf. Der erste Brief des zweiten Topfs kollidiert sonst mit dem ersten des
   anderen (23505, Rollback, der Zähler bleibt stehen — der Topf wäre dauerhaft blockiert).
4. Altbriefe abräumen, Doku (FACHKONZEPT §3/§14, HANDBUCH, PII-Matrix, invarianten §7,
   SECURITY/VVT: neuer Zweck „Bescheid"). Der Elternbrief je Schadensfall ist entfernt
   (Migration 164), der Staffel-Vorschlag im Schaden-Dialog gebaut.

---

## 5. Was bewusst nicht gebaut wird

- E-Mail-/App-Versand der Bescheide (Schriftform; A3).
- Automatische Aussonderung per Cron nach Fristablauf (stille Bestandsänderung).
- Löschen ausgebuchter Exemplare (Bestandskartei-Nachweis).
- Eltern-Namen / zweite Anschrift (E8).
- Ein frei editierbares Template für den Bescheid.

---

## 6. Entscheidungen (Schule / Sekretariat / Schulträger)

| #      | Frage                                                                                                                                                                                          | Empfehlung                                                                                                                                |
| ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| **E1** | Bereichs-Nr. (4-stellig) und Schulnummer (4-stellig) für die Referenznummer — beides kennt nur das Sekretariat bzw. die Schulaufsicht.                                                         | Sekretariat fragt nach; bis dahin Platzhalter, Betriebsbereitschaft warnt.                                                                |
| **E2** | Aktuelle Fassung des Verfahrens samt Musterschreiben bei der Schulaufsicht anfordern? Gebaut ist nach dem Muster von 2014; der Text ist an einer Stelle austauschbar.                              | Ja, anfordern — parallel bauen.                                                                                                           |
| **E3** | Frist 28 Tage (Verfahren) oder 6 Wochen (Anforderungsliste)?                                                                                                                                   | 28 Tage, einstellbar.                                                                                                                     |
| **E4** | Preisbasis ab dem 2. Verleihjahr: neues Feld „Listenpreis" am Titel (von Hand gepflegt) oder Einkaufspreis als Näherung?                                                                       | ✅ **Entschieden am 17.09.2026, gebaut.** Feld `listenpreis` am Titel, optional — aber NICHT von Hand gepflegt: Die DNB liefert den Ladenpreis in jeder ISBN-Abfrage mit, er stand dort ungenutzt. Der Vorschlag nimmt den Listenpreis, sonst den Einkaufspreis, und sagt welchen; welcher gilt, ist zusätzlich einstellbar. |
| **E5** | Zahlungsweg für die Schülerbücherei (Kasse des Trägers + Kassenzeichen? Budgetkonto? bar mit Quittung?) — mit dem Schulträger klären.                                                          | Bis zur Antwort: Schulträger-Rechnung druckt sichtbar „(Bankverbindung des Schulträgers nicht hinterlegt)" — nie ein erfundenes Konto.    |
| **E6** | Nach Übergabe an die Schulaufsicht: bleibt der Schüler gesperrt und die Forderung offen, bis das Sekretariat „bezahlt laut Finanzbericht" bucht — oder gilt Übergabe schulseitig als erledigt? | Übergabe = für die Schule erledigt, auch für die Sperre: Arbeitshilfe Abschnitt 2 („Weitere Schritte sind durch die Schule nicht zu veranlassen") und Anforderungsliste Punkt 7 („Saldo … bereinigt"). |
| **E7** | Kein E-Mail-/App-Versand der Bescheide — bestätigen?                                                                                                                                           | Ja.                                                                                                                                       |
| **E8** | Keine Eltern-Namen / zweite Anschrift im Datenbestand — bestätigen?                                                                                                                            | Ja.                                                                                                                                       |

---

## 7. Teil B — Beschaffung: getrennte Töpfe im Bestellwesen

**Ziel:** Bestellungen und Rechnungen weisen getrennt aus, was aus Landesmitteln (Lernmittelfreiheit) und was aus Mitteln
des Schulträgers (Schülerbücherei) beschafft wird.

### 7.1 Was das Verfahren verlangt

- Die Schule **vermerkt schon auf der Bestellung**, ob die Bücher im Rahmen der
  Lernmittelfreiheit beschafft werden oder für die Schülerbücherei. Der Vermerk gehört auf
  die Bestellung, nicht erst auf die Rechnung: Der Händler gewährt auf
  Lernmittel-Sammelbestellungen einen anderen Nachlass als auf Bibliotheksbestand, und
  eine Bestellung, die beides mischt, kann er nicht richtig rabattieren.
- Der Nachlass auf Lernmittel setzt voraus, dass die Bücher Eigentum des Landes bleiben.
  Bücherei-Bücher als Sammelbestellung zu deklarieren ist nicht zulässig.
- Die Rechnungen der beiden Töpfe werden getrennt geführt: Auf der Lernmittel-Rechnung
  bestätigt die Schule Inventarisierung, sachliche und rechnerische Richtigkeit und
  Verwendungszweck, das Original geht nach Prüfung weiter; die Rechnung des Schulträgers
  bleibt bei der Schule (Auskunft der EDV-Servicestelle für Schulbibliotheken, 10.09.2026).
- Rechnungsunterlagen zehn Jahre, Lieferunterlagen sechs Jahre aufbewahren.
- Zugangsbuch: jede Lieferung mit Eingangsdatum, Titel, Anzahl, Lieferant,
  Inventarnummer; bei EDV-Führung je Schulhalbjahr ein Ausdruck der Neuanschaffungen,
  Bestandskartei zum 15.3. und 15.9.
- Der Topf ist eine **Entscheidung je Bestellung**, die der Titel nur vorschlägt — ein
  falsch gekennzeichneter Titel darf im Warenkorb umgehängt werden.

### 7.2 Warum die Schritte nötig sind

- **Ohne Topf an der Bestellung** können Berichte, Historie, Übersicht und Detail nicht
  trennen.
- **Ein Anschreiben ohne Topf nennt einen Zweck für alle Bestellungen.** Ein Händler, der den
  Brief ernst nimmt, gewährt dann auf Lernmittel oder auf Bücherei-Bestand den falschen
  Nachlass. Das Anschreiben erzeugt `pdf/bestellanschreiben.go`.
- **Händler führen für Lernmittel und Bibliothek oft getrennte Kundenkonten** (anderer
  Nachlass, andere Rechnungsstelle); deshalb kann ein Lieferant zwei Kundennummern tragen.
- **Der Bestellbedarf ist Lernmittel** (`bestellbedarfFilter`: Vorgabe `lmf`), die Titelsuche
  liefert alles. Ein über die DNB-Suche neu angelegter Titel (`schreibeTitelAusMetadaten`)
  bekäme ohne Frage kein `ist_lernmittel` — ein neues Schulbuch entstünde als Bücherei-Titel
  und bliebe es (Frist, Katalog, Löschfrist, Bestellbedarf lesen die Spalte).
- Der Wareneingang hängt an `buecher_exemplare.bestellung_id` (Migration 063); daraus
  entsteht das Zugangsbuch (Berichte → „Bestandsbücher", Reiter Zugangsbuch und
  Abgangsbuch, je Halbjahr und Topf; FACHKONZEPT.md Abschnitt 13).

### 7.3 Bauplan (kleiner als Teil A)

**Gebaut** sind die Schritte 1 bis 7 im ersten Schnitt: Migration 109, Warenkorb in zwei
Gruppen mit Verschieben, je Gruppe eine Bestellung, Vermerk auf Anschreiben und Mail,
Lernmittel-Frage im Staging-Fenster, Topf-Chip in Historie und Detail, Topf nachträglich mit
Grund korrigierbar, Bericht in Blöcken je Topf mit eigener Summe, Topf-Filter für alle drei
Berichtsarten, Filter und Aufteilung der Kennzahlen in der Historie. Eine eigene
Rechnungsanschrift je Topf gibt es nicht, beide Rechnungen gehen an die Schule (D3). Der Vermerk nennt weder Träger noch
Behörde, nur den Topf (Vorgabe: kein Rechts- oder Regionalbezug außerhalb der
Formulare an Schüler). Gates: `api/bestellung/bestellung_mittel_pg_test.go`,
`db/bestellung_mittel_nachtrag_pg_test.go`, `api/bestellung/bestellanschreiben_test.go`,
`internal/service/bestellmail_mittel_test.go`, `api/katalog/titel_lernmittel_pg_test.go`,
`api/bestellung/bestellung_mittel_korrektur_pg_test.go`, `api/bestellung/lieferant_zweitnummer_pg_test.go`,
`pkg/mitteltopf/paritaet_test.go`, `frontend/src/lib/stores/bestellStore.test.js`.

1. **Datenmodell:** `bestellungen_verlauf.mittel TEXT CHECK (mittel IN ('land','schultraeger'))`,
   nullbar für Alt-Bestellungen; eine Backfill-Migration ordnet eindeutige Fälle zu (alle
   Positionen Lernmittel → `land`, keine → `schultraeger`), gemischte bleiben NULL und
   erscheinen als „ohne Zuordnung" — nie geraten. `lieferanten.kundennummer_schultraeger`
   (leer = dieselbe Nummer; fehlt das Feld in einer Anfrage, bleibt die Nummer unangetastet).
   Dasselbe Vokabular wie `schadensersatz_bescheide.mittel` in Teil A.
2. **Eine Bestellung = ein Topf.** Der Warenkorb gruppiert seine Positionen nach dem
   Vorschlag aus `ist_lernmittel` in „Lernmittelfreiheit (Land)" und „Schülerbücherei
   (Schulträger)", jede Gruppe mit eigener Summe; eine Position lässt sich in die andere
   Gruppe schieben (falsch gekennzeichneter Titel). **„Bestellung auslösen" erzeugt je
   Gruppe eine Bestellung** — zwei Mails, zwei Anschreiben, zwei Bestätigungslinks an
   denselben Händler, je mit eigenem Doppelklick-Schutz. Der Server nimmt `mittel` je
   Bestellung an und speichert es; Pflichtfeld für neue Bestellungen (400 an der Tür).
3. **Der Vermerk auf Bestellung und Mail:** Anschreiben-Betreff und -Text nach Topf
   („Bestellung im Rahmen der Lernmittelfreiheit — Sammelbestellung der Schule" /
   „Anschaffung für die Schülerbücherei aus Mitteln des Schulträgers"), Kundennummer des
   Topfs. EINE Textquelle `pkg/mitteltopf`; Platzhalter `{{.Mittel}}` für die
   Vorlage `BESTELLUNG_HAENDLER` (Platzhalter-Paritäts-Gate zieht mit; fehlt der
   Platzhalter, ergänzt der Versand den Vermerk).
4. **Berichte getrennt:** Die Detailliste
   steht in Blöcken (Lernmittelfreiheit, Schülerbücherei, zuletzt die Alt-Bestellungen
   ohne Zuordnung), je mit eigener Summe und der Gesamtsumme darunter; alle drei
   Berichtsarten nehmen den Topf-Filter (`?mittel=land|schultraeger|ohne`), der Titel
   nennt ihn. Die Bestellhistorie filtert serverseitig — sie ist gedeckelt, im Browser
   gefiltert zeigte sie nur den Rest der neuesten 200 —, und die Kennzahlen im Kopf
   stehen zusätzlich je Topf. Der Filterwert `ohne` ist der Weg zu den Alt-Bestellungen,
   die der Rückweg (Nr. 6) noch braucht. Gates: `api/bestellung/bestellbericht_mittel_test.go`
   (PDF-Inhaltsstrom, Summen je Topf = Gesamt), `repository/bestellbericht_mittel_pg_test.go`,
   `api/bestellung/bestellhistorie_mittel_pg_test.go` (Aufteilung = Gesamtzahlen),
   `bestellberichte.test.js`, `BestellHistorieKopf.test.js`,
   `TrefferPruefung.test.js` (der DNB-Titel mit Haken).
5. **Neuer Titel aus der DNB-Suche:** Das Staging-Fenster fragt „Lernmittel?" und setzt
   `ist_lernmittel` über `PUT /api/buecher/titel/{id}/lernmittel` — schließt die Lücke
   aus 7.2.
6. **Rückweg:** Der Topf einer bestehenden Bestellung lässt sich im Bestelldetail mit
   Pflicht-Grund korrigieren (`PUT /api/bestellungen/{id}/mittel`, Admin-Audit-Log mit
   von/nach/Grund). Das Anschreiben beim Händler ändert sich nicht; die Korrektur gilt der
   eigenen Zuordnung, aus der die Berichte rechnen — auch für Alt-Bestellungen „ohne
   Zuordnung".
7. **Zugangsbuch-Ausdruck** je Schulhalbjahr und Topf aus dem Wareneingang: der Bericht
   „Bestandsbücher" (7.2).

### 7.4 Entscheidungen (Teil B)

| #      | Frage                                                                                                                                                                                | Antwort / Empfehlung                                                                                                                                                                                                                             |
| ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **D1** | Eine Bestellung = ein Topf; ein gemischter Warenkorb wird beim Auslösen in zwei Bestellungen geteilt (zwei Mails an denselben Händler).                                              | **Bestätigt (EDV-Servicestelle für Schulbibliotheken, 10.09.2026):** „optimal" — über Littera wird ebenfalls getrennt bestellt; die Trennung trägt den getrennten Finanzen von Schulträger und Land Rechnung. Gebaut.                            |
| **D2** | Hat der Händler getrennte Kundenkonten für Lernmittel und Bibliothek? (Sekretariat fragen.)                                                                                          | Gebaut als optionales Feld „Kundennummer Schülerbücherei" am Lieferanten (Migration 109); leer = dieselbe. Ob der Händler ein zweites Konto führt, trägt das Sekretariat ein, wenn es so ist.                                                    |
| **D3** | Rechnungsanschrift und Wortlaut je Topf: Land = an die Schule. Schulträger = an die Schule oder direkt an den Träger? (dieselbe Frage wie E5)                                        | **Beantwortet (10.09.2026):** Die Rechnung für Mittel des Schulträgers geht direkt an die Schule. Beide Bestellungen tragen dieselbe Anschrift; der Unterschied ist allein der Vermerk (`pkg/mitteltopf/mitteltopf.go`). Keine eigene Träger-Anschrift. |
| **D4** | Alt-Bestellungen rückwirkend zuordnen, wo es eindeutig ist; gemischte bleiben „ohne Zuordnung"?                                                                                      | Ja — Backfill in Migration 109, belegt in `db/bestellung_mittel_nachtrag_pg_test.go`; der Rest über den Rückweg (7.3 Nr. 6) von Hand.                                                                                                           |
| **D5** | Gibt es eine Vereinbarung, nach der ein Anteil der Lernmittel-Mittel für Lehrmittel verwendet werden darf und umgekehrt? Dann müsste das Verschieben einer Position vermerkt werden. | **Beantwortet (10.09.2026):** Nein, im Zuständigkeitsbereich der Schule gibt es keine solche Regelung. Kein Vermerk gebaut; das Verschieben bleibt als Korrektur falsch gekennzeichneter Titel.                                                  |
