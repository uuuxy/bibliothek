# Landesmittel und Kreismittel — Konzept (Entwurf 09.09.2026, Stand 10.09.2026)

**Teil A:** Schadensersatz für verlorene und beschädigte Bücher (Abschnitte 1–6).
**Teil B:** Getrennte Töpfe in der Beschaffung — Bestellung, Rechnung, Berichte (Abschnitt 7).
Beide Teile teilen EIN Vokabular für die Mittelherkunft: `land` (Lernmittelfreiheit) und
`schultraeger` (Schülerbücherei) — `repository/mittel.go`, `bestellungen_verlauf.mittel`.

**Status (10.09.2026):** Teil A ist im ERSTEN SCHNITT GEBAUT — der Bescheid als Brief,
Datenmodell mit Nummernkreis, Staffel, Einstellungen, Erstellen aus dem Mahnwesen,
Nachdruck, Übergabe. Offen bleiben aus Abschnitt 4.7 die Etappen 2 (Massen-Anbindung ans
Mahnwesen, Rückgabe-Hook) und 4 (Altbriefe abräumen, Staffel-Vorschlag im Schaden-Dialog).
Teil B ist ebenfalls im ersten Schnitt gebaut (Abschnitt 7.3).

**Was der Einbau geworden ist (Absprache 10.09.2026, drei Entscheidungen):** Der Bescheid
entsteht in der bestehenden Auswahlleiste des Mahnwesens und nur bei GENAU EINEM
markierten Schüler — jeder Betrag ist Ermessen, jede Referenznummer unwiderruflich. Die
Briefe leben im vierten Reiter derselben Reiterzeile (Alle → Akut fällig → Eskaliert →
Bescheide), dessen Zahl nur die abgelaufenen Fristen nennt. Der Erstell-Schritt ist EIN
Basis-Dialog ohne Schritte (Material 3: Vollbild ist die Form für Telefone, einen Stepper
gibt es dort nicht) und zeigt genau das, was ein Mensch entscheidet: welche Bücher, welcher
Betrag, mit der Herleitung daneben.

**Zwei Fehler, die erst das ANSEHEN des fertigen PDFs zeigte** (die Gates waren grün):
„Ayşe" wurde zu „Ay.e" (der Zeichensatz des PDFs kennt das ş nicht), und der Seitenumbruch
riss die Tabellenkopfzeile entzwei. Beide Klassen haben jetzt ein Gate; für das zweite
liest `internal/pdftest` die Seiten getrennt (`TexteJeSeite`).

**Anlass:** Peter hat am 08./09.09.2026 vier Unterlagen der Schule vorgelegt — die
Verfahrensbeschreibung für den Schadensersatz bei Lernmitteln (Stand 2014), die
Arbeitshilfe dazu, das verbindliche Musteranschreiben an die Erziehungsberechtigten und
eine Anforderungsliste „Mahnverfahren". Auftrag: das Verfahren in die bestehende
Anwendung einbauen, ohne die Struktur zu zerstören, und dabei möglichst viel Bestehendes
weiterverwenden. Zwei Randbedingungen aus dem Gespräch: **Den Brief schickt das
Sekretariat** (im System als Admin angemeldet), nicht die Lernmittel-Stelle. Und es
braucht **zwei Rechnungen**, weil zwei Töpfe: Lernmittel gehören dem Land, die
Schülerbücherei ist aus Mitteln des Schulträgers beschafft.

---

## 1. Was das Verfahren der Schule verlangt (aus den Unterlagen, Stand 09.09.2026)

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
  Datum**. Staffel: 1. Verleihjahr voller Kaufpreis; 2. Jahr 80 %, dann je Jahr −20 % des
  **Neupreises zum Zeitpunkt des Verlusts**; ab dem 5. Jahr 10 %. Zwei Fallgruppen (nicht
  zurückgegeben / so stark beschädigt, dass unbenutzbar), Unterschrift der Schulleitung,
  vorgeschriebener Schlussabsatz mit Einspruchsfrist von einem Monat. Nach Fristablauf
  Original + Buchungsbeleg an die Schulaufsicht, Kopie bleibt; **spätere Rückgabe →
  Schulaufsicht unverzüglich informieren.**
- Praxis einer Nachbarschule: Rechnung mit Zeitwert, **nur Überweisung, keine
  Barzahlung**; bis zur Zahlung Sperre für weitere Ausleihen.

**Zwei Dinge, die die Schule wissen muss:**

1. Die vorliegenden Unterlagen sind von **2014**. Das Musteranschreiben nennt eine Stelle,
   die es seit 2015 nicht mehr gibt; die aktuell geltende Fassung des Verfahrens ist nicht
   öffentlich. Das Sekretariat sollte sie samt aktuellem Musterschreiben bei der
   Schulaufsicht anfordern. Die Bauweise unten hält den Brieftext an EINER Stelle
   austauschbar; Struktur, Nummer, Konto und Frist dürften unverändert sein.
2. Barzahlung „in der Bibliothek" — was die drei heutigen Briefe verlangen — ist für
   Landeseigentum die **Ausnahme, nicht der Weg**.

### 1.2 Schülerbücherei (Mittel des Schulträgers)

- Errichtung und Ausstattung der Schülerbücherei sind Sache des Schulträgers; ihr Bestand
  gehört ihm.
- Das Benutzungsverhältnis folgt der Benutzungsordnung der Schule: zuerst
  Ersatzbeschaffung, sonst Geld in Höhe des Neuwerts.

Folge: Für den Bestand der Schülerbücherei gibt es **keinen förmlichen Bescheid, keine
Referenznummer im Landesformat** — sondern eine gewöhnliche Rechnung/Zahlungsaufforderung
der Schule, deren Geld an den Schulträger gehört. Wohin genau (Kasse des Trägers mit
Kassenzeichen, Budgetkonto der Schule, bar mit Quittung), ist mit dem Schulträger zu
klären. Sicher ist nur: nicht auf das Konto des Landes.

---

## 2. Was die Anwendung heute kann (am Code nachgesehen)

| Baustein                             | Fundstelle                                                                                                                                                    | Stand                                                                                                                                                              |
| ------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Forderung anlegen                    | `repository/damage.go` (`ReportDamage` beendet Ausleihe + sondert aus `BESCHAEDIGUNG`; `MarkCopyDefekt` lässt die Ausleihe offen), `DamageReportModal.svelte` | ✅ inkl. Idempotenz, Schuldner aus der Ausleihe, 409 bei Neuverleih. **Betrag ist Freitext, Vorgabe 15,00 €** — keine Staffel, kein Bezug zum Preis.               |
| Forderung erledigen                  | `api/damage_resolve.go`, `repository/audit_system.go` (Bezahlt / Storno mit Grund, Audit, 409 bei Doppelbuchung)                                              | ✅ „Bezahlt" bedeutet heute **Barzahlung am Tresen**.                                                                                                              |
| Brief 1: Elternbrief je Schadensfall | `pdf/schadensfall.go`, `api/pdf.go` (`elternbrief_generiert`)                                                                                                 | 14-Tage-Frist, „bar in der Bibliothek", „Schulbibliotheksordnung", Unterschrift „Bibliotheksleitung". Für Lernmittel in vier Punkten falsch.                       |
| Brief 2: Rechnung je Schüler         | `pdf/rechnung.go`, `api/print.go` (alle offenen Forderungen)                                                                                                  | DIN 5008, LEFT JOINs (Geräteschaden vorgesehen), aber „bar in der Bibliothek", keine Nummer, kein Topf.                                                            |
| Brief 3: Eltern-Mahnbrief            | `api/reports_pdf.go`, Vorlage `MAHNUNG_ELTERN`                                                                                                                | DIN-5008-Fensterkuvert, Falzmarken, Tabelle der überfälligen Bücher — **die Bauform, die der Bescheid braucht.**                                                   |
| Mahnwesen                            | `api/mahnwesen*.go`, `repository/mahnwesen_*.go`, `Mahnwesen.svelte` + `components/mahnwesen/`                                                                | Überfällige nach Klasse/Jahrgang, Auswahl → „Mahnbriefe drucken" (Mahnstufe steigt nur beim Druck), Klassenleitungs-Mail, Sperre ab `max_overdue_days`.            |
| Schulstammdaten                      | `SchuleKategorie.svelte`, `system_settings*.go`                                                                                                               | Name, Anschrift, Eigentumsvermerk. **Fehlt:** Schulnummer, Schulaufsicht (Nr. + Anschrift), Bankverbindungen, Schulleitung, Geschäftszeichen/Bearbeiter/Durchwahl. |
| Preise                               | `buecher_exemplare.einkaufspreis` (aus Littera übernommen), `erworben_am` = Littera-Zugangsdatum (echt, nicht Importdatum)                                    | Kein Listenpreis/Neupreis.                                                                                                                                         |
| Ausleihhistorie je Exemplar          | `ausleihen` (Zeilen bleiben nach der Anonymisierung ohne Person)                                                                                              | Zählbar. Aus Littera kamen nur die **offenen** Ausleihen — für den Altbestand ist die Zahl der Verleihjahre unbekannt.                                             |
| Volljährigkeit                       | `schueler.geburtsdatum` (NULL bei Altdaten)                                                                                                                   | Anrede „Erziehungsberechtigte" vs. volljährig ableitbar.                                                                                                           |
| Rollen                               | ADMIN / MITARBEITER / KOLLEGIUM / HELFER; Sekretariat = ADMIN                                                                                                 | Neues Recht nach dem Muster `merge_students` (db/seed.go, permissionMetadata.js, schuelerRechte.js, permissions.spec.js, PII-Matrix, FACHKONZEPT).                 |
| Nummernkreise                        | `barcode_seq`; Regel: EIN Generator, nie recyceln (Register 068)                                                                                              | Für die Referenznummer braucht es einen Zähler je Kassenjahr.                                                                                                      |

**Kern-Befund:** Die Forderung (`schadensfaelle`) ist die richtige Wahrheit — Sperre,
Löschblockade, DSGVO-Auskunft, Zusammenführen, Bezahlt/Storno hängen alle daran. Es fehlt
ihr die **Art** (nicht zurückgegeben / beschädigt), der **Topf** (Land / Schulträger) und
der **Brief** als eigener Datensatz mit Nummer, Frist und Status. Die drei Briefe sind drei
Fassungen derselben Sache und werden zu **einem Renderer mit zwei Varianten**.

---

## 3. Anforderungsliste „Mahnverfahren" — Abgleich

| Nr. | Wunsch                                                                                                              | Bewertung                                                                                                                                                                                                                                                                                                                    |
| --- | ------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Automatische Abwertung (10 %/Jahr oder je Ausleihe)                                                                 | **Nicht als Dauerzustand bauen.** Die Staffel der Schule (100/80/60/40/20/10 %) wird beim Anlegen einer Forderung als **Vorschlag** berechnet. Ein laufend „abgewerteter Buchwert" hat außerhalb der Forderung keinen Zweck.                                                                                                 |
| 2   | Beschädigung mit Prozentwert bei Katalogisierung/Rückgabe                                                           | Zustandsnotiz gibt es (`zustand_notiz`). Ein Prozentwert ohne Forderung steuert nichts. Bei der Rückgabe „Schaden melden" → Forderung mit Staffel-Vorschlag, Betrag im Ermessen der Schule.                                                                                                                                  |
| 3   | Einkaufspreis UND Listenpreis, wählbar                                                                              | Einkaufspreis ist da. Neupreis zum Zeitpunkt des Verlusts (ab 2. Verleihjahr) fehlt → **Entscheidung E4.**                                                                                                                                                                                                                   |
| 4   | Übersicht Verluste/Beschädigungen                                                                                   | Gibt es (Mahnwesen, Fehlbestand, Gebühren in der Akte). Neu: Liste der Bescheide mit Fristablauf.                                                                                                                                                                                                                            |
| 5   | Versand per Post/E-Mail/App; Referenznummer, jährlich neu                                                           | **Nur Druck/Post.** Das Verfahren verlangt Schriftform; Datenschutz-Entscheidung A3 vom 22.08.2026 (keine Eltern-Mahnmail). Referenznummer ja; „Zurücksetzen zu Jahresbeginn" ergibt sich, weil das Kassenjahr Teil der Nummer ist.                                                                                          |
| 6   | Zahlung bestätigen; Rückgabe → Buch wieder verfügbar, Saldo null; Zahlung ohne Rückgabe → „verloren" + **gelöscht** | Bezahlt gibt es. Rückgabe → Forderung automatisch stornieren (neu). **Gelöscht wird nie**: Die Bestandskartei muss die Aussonderung nachweisen → `ist_ausgesondert`/`aussonderung_grund = 'VERLUST'`, genau wie heute.                                                                                                       |
| 7   | Nach Fristablauf: zweiter Ausdruck für die Schulaufsicht, Rest automatisch „verloren"                               | Liste „Frist abgelaufen" + Aktion **„An die Schulaufsicht übergeben"** (Mensch klickt, kein Cron): Original-Nachdruck + Sammelliste, Bücher werden ausgesondert. Automatik über Nacht wäre eine stille Aussonderung — genau die Bugklasse, die dieses Projekt meidet.                                                        |
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

Reine Funktion `Ersatzwert(verleihjahr, kaufpreis, neupreis) (betrag, prozent, basis)`
nach der Staffel der Schule. `verleihjahr = max(Anzahl Schuljahre mit Ausleihe des
Exemplars, Jahre seit erworben_am + 1)` — die zweite Größe fängt den Littera-Altbestand
ohne Historie ab (ein 2019 gekauftes Buch ist nicht im 1. Verleihjahr, nur weil wir seine
Ausleihen nicht kennen). Der Dialog zeigt die Herleitung („3. Verleihjahr → 60 % von
24,90 €") und der Mensch bestätigt oder überschreibt — im Ermessen der Schule.

### 4.3 Schreibpfade

| Weg                                                                                                   | Was passiert                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| ----------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `POST /api/schueler/{id}/bescheide` `{mittel, positionen:[{ausleihe_id \| schadensfall_id, betrag}]}` | Eine Transaktion: Schülerzeile sperren, Positionen prüfen (gehören zum Schüler, Titel passt zum Topf via `ist_lernmittel`), für überfällige Ausleihen je eine Forderung `art = nicht_zurueckgegeben` anlegen (**Ausleihe bleibt offen, Buch wird NICHT ausgesondert** — es kann noch zurückkommen), Nummer ziehen, Frist rechnen, Snapshot schreiben, PDF erzeugen, **erst dann committen** (Muster `erzeugeUndCommitBulkMahnung`: Papier == DB), Audit. |
| `GET /api/bescheide/{id}/pdf`                                                                         | Nachdruck aus dem Snapshot, dieselbe Nummer, `letzter_druck_am`. Nie eine neue Nummer.                                                                                                                                                                                                                                                                                                                                                                   |
| `POST /api/bescheide/{id}/uebergeben`                                                                 | Nach Fristablauf, von Hand: Status `uebergeben`; Forderungen `nicht_zurueckgegeben` → Ausleihe beenden + Exemplar `VERLUST` (Ablauf Nr. 7). Liefert Übergabe-PDF: Original + Sammelliste für die Schulaufsicht.                                                                                                                                                                                                                                          |
| Rückgabe-Hook in `repository/loan.go` (`ReturnBook`)                                                  | Hat die Ausleihe eine offene Forderung `nicht_zurueckgegeben`: Storno mit Grund „Rückgabe am …"; steht der Brief auf `uebergeben` → `schulaufsicht_zu_informieren = true` + Hinweis an der Theke. Das ist der einzige Eingriff in einen Kernpfad — eigener PG-Test, am Rückbau rot gesehen.                                                                                                                                                              |
| Bezahlt / Storno                                                                                      | unverändert (`/api/schadensfaelle/{id}/bezahlt`, `/storno`). Brief gilt als `erledigt`, wenn keine Position mehr offen ist (abgeleitet, nicht doppelt gespeichert).                                                                                                                                                                                                                                                                                      |
| `GET /api/bescheide?status=`                                                                          | Liste für das Sekretariat: offen / **Frist abgelaufen** / übergeben / Schulaufsicht zu informieren.                                                                                                                                                                                                                                                                                                                                                      |

Recht: neu `schadensersatz_bescheide` (ab Werk nur ADMIN — das Sekretariat), Anlegen
einer Forderung bleibt `edit_students`. PII-Stufe 3 → PII-Matrix + Antwort-Gate.

### 4.4 Briefe: ein Renderer, zwei Varianten (`pdf/schadensersatz.go`)

Gebaut auf der DIN-5008-Fensterkuvert-Seite aus `reports_pdf.go` (Falzmarken,
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
- Beide ersetzen `pdf/schadensfall.go` und `pdf/rechnung.go`; `api/pdf.go`, `api/print.go
(Rechnung)` und `elternbrief_generiert*` werden abgeräumt (Deadcode-Gate, Bestandsliste
  der Bauteile). Der Eltern-**Mahnbrief** (überfällig, ohne Geld) bleibt, wie er ist.

### 4.5 Oberfläche

- **Schülerakte → Gebühren & Schäden:** Forderungen tragen Topf und Art; Knopf
  „Bescheid erstellen (Land)" / „Rechnung erstellen (Schulträger)" nur mit Recht; darunter
  die Briefe mit Nummer, Frist, Status, „Nachdruck", „An die Schulaufsicht übergeben".
- **Mahnwesen:** Auswahl-Modus bekommt neben „Mahnbriefe drucken" den Knopf
  **„Schadensersatz-Bescheide (Land)"** — je markiertem Schüler ein Brief über seine
  überfälligen Lernmittel plus offene Lernmittel-Forderungen; die Tabelle des Briefs
  füllt sich aus denselben Daten wie heute der Mahnbrief. Dritter Filter „Bescheide"
  (Frist abgelaufen → Übergabe).
- **Schaden melden** (`DamageReportModal`): Art (verloren / beschädigt), Staffel-Vorschlag
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
Forderung und markiert nach Übergabe (PG) · Topf ≡ `ist_lernmittel` jeder Position
(PG) · Brief-Inhaltsstrom trägt Referenznummer, Frist-Datum, Konto und Schlussabsatz
(Gate am fertigen PDF, wie `etikett-gate`) · Nachdruck ändert weder Nummer noch Betrag ·
Anonymisierung tilgt den Snapshot, lässt die Nummer (DSGVO-Paar-Gate) · Recht am Draht
(`permissions.spec.js`) · Staffel als Tabelle (1→100 % Kaufpreis, 2→80 %, 3→60 %,
4→40 %, 5→20 %, 6+→10 % Neupreis).

### 4.7 Etappen

1. ~~Datenmodell + Nummernkreis + Landes-Bescheid + Einstellungen.~~ **GEBAUT 10.09.2026**
   (Migration 110, `pkg/ersatzwert`, `api/bescheid_pdf.go`, `repository/bescheid.go`,
   `api/bescheid_handler.go`, Kategorie „Schadensersatz", Reiter und Dialog im Mahnwesen).
   Abweichungen vom Plan oben, jeweils mit Grund: Der Bescheid entsteht im **Mahnwesen**
   statt in der Schülerakte (dort steht, wer überfällig ist); `schadensfaelle.art` hat
   **zwei** Werte statt drei (genau die zwei Kästchen des Formulars — „Verlust" IST die
   erste Gruppe); `schueler_id` ist **SET NULL** statt RESTRICT (der Brief überlebt die
   DSGVO-Löschung als Beleg ohne Person, RESTRICT hätte die berechtigte Löschung
   blockiert); das Recht ist `edit_students` statt eines neuen — ein eigenes Recht bliebe
   ab Werk bei niemandem und wäre eine Tür, die keiner öffnen kann.
2. Mahnwesen-Anbindung (Auswahl → Bescheide) + Bescheid-Liste + Übergabe + Rückgabe-Hook.
3. Schulträger-Rechnung + Betriebsbereitschaft-Warnung.
4. Altbriefe abräumen, Staffel-Vorschlag im Schaden-Dialog, Doku (FACHKONZEPT §3/§14,
   HANDBUCH, PII-Matrix, invarianten §4, SECURITY/VVT: neuer Zweck „Bescheid").

---

## 5. Was ich bewusst NICHT baue

- E-Mail-/App-Versand der Bescheide (Schriftform; A3).
- Automatische Aussonderung per Cron nach Fristablauf (stille Bestandsänderung).
- Löschen ausgebuchter Exemplare (Bestandskartei-Nachweis).
- Eltern-Namen / zweite Anschrift (E8).
- Ein frei editierbares Template für den Bescheid.

---

## 6. Entscheidungen (Peter / Sekretariat / Schulträger)

| #      | Frage                                                                                                                                                                                          | Empfehlung                                                                                                                                |
| ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| **E1** | Bereichs-Nr. (4-stellig) und Schulnummer (4-stellig) für die Referenznummer — beides kennt nur das Sekretariat bzw. die Schulaufsicht.                                                         | Sekretariat fragt nach; bis dahin Platzhalter, Betriebsbereitschaft warnt.                                                                |
| **E2** | Aktuelle Fassung des Verfahrens samt Musterschreiben bei der Schulaufsicht anfordern? Ich baue nach dem 2014er Muster; der Text ist an einer Stelle austauschbar.                              | Ja, anfordern — parallel bauen.                                                                                                           |
| **E3** | Frist 28 Tage (Verfahren) oder 6 Wochen (Anforderungsliste)?                                                                                                                                   | 28 Tage, einstellbar.                                                                                                                     |
| **E4** | Preisbasis ab dem 2. Verleihjahr: neues Feld „Listenpreis" am Titel (von Hand gepflegt) oder Einkaufspreis als Näherung?                                                                       | Feld `listenpreis` am Titel, optional; Vorschlag nimmt Listenpreis, sonst Einkaufspreis, und sagt, welchen. Der Mensch bestätigt ohnehin. |
| **E5** | Zahlungsweg für die Schülerbücherei (Kasse des Trägers + Kassenzeichen? Budgetkonto? bar mit Quittung?) — mit dem Schulträger klären.                                                          | Bis zur Antwort: Schulträger-Rechnung druckt sichtbar „(Bankverbindung des Schulträgers nicht hinterlegt)" — nie ein erfundenes Konto.    |
| **E6** | Nach Übergabe an die Schulaufsicht: bleibt der Schüler gesperrt und die Forderung offen, bis das Sekretariat „bezahlt laut Finanzbericht" bucht — oder gilt Übergabe schulseitig als erledigt? | Sperre bleibt, Löschblockade fällt (Übergabe ≙ erledigt für die DSGVO-Kette, sonst hängt der Datensatz ewig an der Schulaufsicht).        |
| **E7** | Kein E-Mail-/App-Versand der Bescheide — bestätigen?                                                                                                                                           | Ja.                                                                                                                                       |
| **E8** | Keine Eltern-Namen / zweite Anschrift im Datenbestand — bestätigen?                                                                                                                            | Ja.                                                                                                                                       |

---

## 7. Teil B — Beschaffung: getrennte Töpfe im Bestellwesen (09.09.2026)

**Anlass:** Peter nach einem Telefonat mit der Schule: Die Bestellungen und Rechnungen
sollen getrennt ausweisen, was aus Landesmitteln (Lernmittelfreiheit) und was aus Mitteln
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

### 7.2 Was die Anwendung vor Teil B tat (am Code nachgesehen, 09.09.2026)

- **Keine Bestellung kannte ihren Topf.** `bestellungen_verlauf` trug Lieferant,
  Kundennummer, Betrag, Exemplare, Bestätigung — sonst nichts. Berichte, Historie,
  Übersicht und Detail konnten deshalb nicht trennen.
- **Das Anschreiben behauptete für JEDE Bestellung die Schülerbücherei:** `order_pdf.go`
  schrieb „hiermit bestellen wir für unsere Schulbibliothek …" und im Betreff
  „Buchbestellung für die Schulbibliothek" — auch wenn ein Lernmittel-Klassensatz bestellt
  wurde. Das war genau der geforderte Vermerk, nur falsch herum: Ein Händler, der den
  Brief ernst nimmt, gewährt auf Lernmittel den falschen Nachlass.
- **Ein Lieferant hatte EINE Kundennummer.** Händler führen für Lernmittel und Bibliothek
  oft getrennte Kundenkonten (anderer Nachlass, andere Rechnungsstelle).
- **Der Bestellbedarf ist Lernmittel** (`reorderFilter`: Vorgabe `lmf`), die Titelsuche
  liefert alles. Ein über die DNB-Suche NEU angelegter Titel (`upsertTitelAusMetadaten`)
  bekam **kein** `ist_lernmittel` — ein neues Schulbuch entstand als Bücherei-Titel und
  blieb es (Frist, Katalog, Löschfrist, Bestellbedarf lesen die Spalte).
- Der Wareneingang hängt an `buecher_exemplare.bestellung_id` (Migration 063) — das
  Zugangsbuch ist damit aus den Daten ableitbar, nur nicht ausgedruckt.

### 7.3 Bauplan (kleiner als Teil A)

**Stand 10.09.2026 — erster Schnitt gebaut:** Schritte 1, 2, 3 und 5 (Migration 109,
Warenkorb in zwei Gruppen mit Verschieben, je Gruppe eine Bestellung, Vermerk auf
Anschreiben und Mail, Lernmittel-Frage im Staging-Fenster, Topf-Chip in Historie und
Detail, Topf nachträglich mit Grund korrigierbar). Offen als zweiter Schnitt: Schritt 4
(Berichte und Historie nach Topf getrennt). Schritt 6 ist nach den Antworten unten auf
die zweite Kundennummer geschrumpft — eine eigene Rechnungsanschrift je Topf gibt es
nicht, beide Rechnungen gehen an die Schule. Der Vermerk nennt weder Träger noch
Behörde, nur den Topf (Peters Vorgabe: kein Rechts- oder Regionalbezug außerhalb der
Formulare an Schüler). Gates: `api/bestellung_mittel_pg_test.go`,
`api/bestellung_mittel_backfill_pg_test.go`, `api/order_pdf_mittel_test.go`,
`api/bestellmail_mittel_test.go`, `api/titel_lernmittel_pg_test.go`,
`api/bestellung_mittel_korrektur_pg_test.go`, `api/lieferant_zweitnummer_pg_test.go`,
`api/mittel_vokabular_paritaet_test.go`, `frontend/src/lib/stores/orderStore.test.js`.

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
   Topfs. EINE Textquelle `api/mittel_vermerk.go`; Platzhalter `{{.Mittel}}` für die
   Vorlage `BESTELLUNG_HAENDLER` (Platzhalter-Paritäts-Gate zieht mit; fehlt der
   Platzhalter, ergänzt der Versand den Vermerk).
4. **Berichte getrennt (offen):** Monats- und Jahresbericht in zwei Blöcken (Land /
   Schulträger) mit je eigener Summe, Gesamtsumme darunter; die Lieferantenabrechnung
   bekommt den Topf-Filter — sie ist das Blatt, gegen das die Händlerrechnung geprüft
   wird. Bestellhistorie: Filter nach Topf; Kennzahlen der Übersicht je Topf.
5. **Neuer Titel aus der DNB-Suche:** Das Staging-Fenster fragt „Lernmittel?" und setzt
   `ist_lernmittel` über `PUT /api/buecher/titel/{id}/lernmittel` — schließt die Lücke
   aus 7.2.
6. **Rückweg:** Der Topf einer bestehenden Bestellung lässt sich im Bestelldetail mit
   Pflicht-Grund korrigieren (`PUT /api/bestellungen/{id}/mittel`, Admin-Audit-Log mit
   von/nach/Grund). Das Anschreiben beim Händler ändert sich nicht; die Korrektur gilt der
   eigenen Zuordnung, aus der die Berichte rechnen — auch für Alt-Bestellungen „ohne
   Zuordnung".
7. **Später, kein Teil dieses Pakets:** Zugangsbuch-Ausdruck je Schulhalbjahr und Topf
   aus dem Wareneingang — die Daten liegen vollständig vor.

### 7.4 Entscheidungen (Teil B)

| #      | Frage                                                                                                                                                                                | Antwort / Empfehlung                                                                                                                                                                                                                             |
| ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **D1** | Eine Bestellung = ein Topf; ein gemischter Warenkorb wird beim Auslösen in zwei Bestellungen geteilt (zwei Mails an denselben Händler).                                              | **Bestätigt (EDV-Servicestelle für Schulbibliotheken, 10.09.2026):** „optimal" — über Littera wird ebenfalls getrennt bestellt; die Trennung trägt den getrennten Finanzen von Schulträger und Land Rechnung. Gebaut.                            |
| **D2** | Hat der Händler getrennte Kundenkonten für Lernmittel und Bibliothek? (Sekretariat fragen.)                                                                                          | Gebaut als optionales Feld „Kundennummer Schülerbücherei" am Lieferanten (Migration 109); leer = dieselbe. Ob der Händler ein zweites Konto führt, trägt das Sekretariat ein, wenn es so ist.                                                    |
| **D3** | Rechnungsanschrift und Wortlaut je Topf: Land = an die Schule. Schulträger = an die Schule oder direkt an den Träger? (dieselbe Frage wie E5)                                        | **Beantwortet (10.09.2026):** Die Rechnung für Mittel des Schulträgers geht direkt an die Schule. Beide Bestellungen tragen dieselbe Anschrift; der Unterschied ist allein der Vermerk (`api/mittel_vermerk.go`). Keine eigene Träger-Anschrift. |
| **D4** | Alt-Bestellungen rückwirkend zuordnen, wo es eindeutig ist; gemischte bleiben „ohne Zuordnung"?                                                                                      | Ja — Backfill in Migration 109, belegt in `api/bestellung_mittel_backfill_pg_test.go`; der Rest über den Rückweg (7.3 Nr. 6) von Hand.                                                                                                           |
| **D5** | Gibt es eine Vereinbarung, nach der ein Anteil der Lernmittel-Mittel für Lehrmittel verwendet werden darf und umgekehrt? Dann müsste das Verschieben einer Position vermerkt werden. | **Beantwortet (10.09.2026):** Nein, im Zuständigkeitsbereich der Schule gibt es keine solche Regelung. Kein Vermerk gebaut; das Verschieben bleibt als Korrektur falsch gekennzeichneter Titel.                                                  |
