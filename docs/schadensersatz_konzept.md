# Landesmittel und Kreismittel — Konzept (Entwurf 09.09.2026)

**Teil A:** Schadensersatz für verlorene und beschädigte Bücher (Abschnitte 1–6).
**Teil B:** Getrennte Töpfe in der Beschaffung — Bestellung, Rechnung, Berichte (Abschnitt 7).
Beide Teile teilen EIN Vokabular für die Mittelherkunft: `land` (Lernmittelfreiheit) und
`schultraeger` (Schülerbücherei, Hochtaunuskreis) — und EINE Einstellungs-Kategorie dafür.

**Status:** Entwurf, NICHT gebaut. Wartet auf die Entscheidungen in Abschnitt 6
(Register: `befunde.md` → „Offen — Entscheidung nötig"). Teil B wartet auf D1–D4.

**Anlass:** Peter hat am 08./09.09.2026 vier Dokumente vorgelegt — den Erlass des HKM vom
17.12.2014 (Az. I.4-Gö-674.100.002-00178), die Arbeitshilfe dazu, das verbindliche
Musteranschreiben an die Erziehungsberechtigten und eine Anforderungsliste
„Mahnverfahren der Schulen in Hessen". Auftrag: das Verfahren in die bestehende
Anwendung einbauen, ohne die Struktur zu zerstören, und dabei möglichst viel Bestehendes
weiterverwenden. Zwei Randbedingungen aus dem Gespräch: **Den Brief schickt das
Sekretariat** (im System als Admin angemeldet), nicht die Lernmittel-Stelle. Und es
braucht **zwei Rechnungen**, weil zwei Töpfe: Lernmittel sind Landeseigentum, die
Schülerbücherei ist aus Kreismitteln beschafft.

---

## 1. Rechtslage (gelesen, nicht nacherzählt — Stand 09.09.2026)

### 1.1 Lernmittel (Landesmittel)

| Quelle                                                                                                          | Was daraus folgt                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| --------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **§ 153 HSchG**                                                                                                 | Lernmittel bleiben Eigentum des Landes, sind spätestens beim Verlassen der Schule zurückzugeben. Schadensersatz nach den Grundsätzen der öffentlich-rechtlichen Leihe; **das Land kann Bestehen und Höhe des Anspruchs durch Verwaltungsakt festsetzen.**                                                                                                                                                                                                                                                                                                                                                  |
| **DVO-LMF** (Verordnung über die Durchführung der Lernmittelfreiheit, 21.04.2013, ABl. S. 278)                  | § 2: **Büchereien sind keine Lernmittel.** § 8: Zugangsbuch + Bestandskartei sind die Bestandsverzeichnisse nach § 73 LHO. § 9: Bei Konflikten ist die Schulaufsichtsbehörde zuständig.                                                                                                                                                                                                                                                                                                                                                                                                                    |
| **Leitfaden „Lernmittelfreiheit in Hessen" (HKM, Juni 2021)** Nr. 11.3, 12.3–12.7                               | Bestandskartei weist Ausleihe, Rücklauf **und Aussonderungen** nach (→ nie löschen, immer aussondern). Zustand bei Ausgabe UND Rückgabe prüfen. Bei Verlust/Beschädigung: Sacherstattung oder Geldbetrag „in angemessener Höhe"; **Geld ist dem Buchungskreis Schulen zugunsten des Titels Ersatzleistungen abzuführen, die Schule bekommt es vom Schulamt erstattet.** Weigerung → Bericht ans Schulamt; Durchsetzung per **Leistungsbescheid** der Schulaufsicht; Maßstab **Wiederbeschaffungswert**.                                                                                                    |
| **Erlass vom 05.03.2025** (ABl. 3/25 S. 44 ff., IV.2-674.100.002-00421, in Kraft ab 01.01.2025) Nr. 6.3 und 6.5 | Zahlungseingänge auf dem Konto des Buchungskreises Schulen werden der Schule aus der Verfügungsreserve des Schulamts zurückgegeben. **„Das Verfahren zur Regelung von Schadensersatzleistungen für Schulbücher und digitale Lehrwerke ist im E-Mail-Erlass vom 11. Juni 2018 geregelt."**                                                                                                                                                                                                                                                                                                                  |
| **Anwendungshinweise Schulgirokonten** (HKM, Stand 18.12.2024) Abschnitt I und II Nr. 3                         | Bankverbindung des Buchungskreises Schulen: Hessisches Kultusministerium (HCC-Schulbereich), **IBAN DE86 5005 0000 0001 0024 01** — damit ist das Konto aus dem Musterschreiben von 2014 noch aktuell. Schadenersatz für Lernmittel darf **nur ausnahmsweise bar** vereinnahmt werden (Quittung, binnen 14 Tagen weiterleiten). **Auf Schulgirokonten des Landes dürfen keine Mittel des Schulträgers geführt werden.**                                                                                                                                                                                    |
| Erlass 17.12.2014 + Arbeitshilfe + Musteranschreiben (Peters Dokumente)                                         | Referenznummer = Schulamtsbereich (4) · Kassenjahr (4) · Schulnummer (4) · laufende Nr. (4), **je Brief, nie doppelt**. Vierwochenfrist **mit Datum**. Staffel: 1. Verleihjahr voller Kaufpreis; 2. Jahr 80 %, dann je Jahr −20 % des **Neupreises zum Zeitpunkt des Verlusts**; ab dem 5. Jahr 10 %. Zwei Fallgruppen (nicht zurückgegeben / so stark beschädigt, dass unbenutzbar), Unterschrift Schulleitung, Rechtsbehelfsbelehrung (Widerspruch binnen eines Monats). Nach Fristablauf Original + Buchungsbeleg ans Schulamt, Kopie bleibt; **spätere Rückgabe → Schulamt unverzüglich informieren.** |
| Praxis im selben Schulamtsbezirk: Kaiserin-Friedrich-Gymnasium Bad Homburg                                      | Rechnung mit Zeitwert, **nur Überweisung, keine Barzahlung, Buchung auf dem Konto des Schulamts**; bis zur Zahlung Sperre für weitere Ausleihen.                                                                                                                                                                                                                                                                                                                                                                                                                                                           |

**Zwei Dinge, die die Schule wissen muss:**

1. Die vorliegenden Dokumente sind von **2014**. Das Musteranschreiben nennt das
   „Landesschulamt und Lehrkräfteakademie" — eine Behörde, die 2015 aufgelöst wurde. Der
   Erlass von 2025 verweist auf einen **E-Mail-Erlass vom 11.06.2018** als geltendes
   Verfahren; der ist nicht öffentlich. Das Sekretariat sollte ihn samt aktuellem
   Musterschreiben beim Staatlichen Schulamt (Bad Vilbel) anfordern. Die Bauweise unten
   hält den Brieftext an EINER Stelle austauschbar; Struktur, Nummer, Konto und Frist
   dürften unverändert sein.
2. Barzahlung „in der Bibliothek" — was die drei heutigen Briefe verlangen — ist für
   Landeseigentum die **Ausnahme, nicht der Weg**.

### 1.2 Schülerbücherei (Kreismittel)

| Quelle                                       | Was daraus folgt                                                                                                                                       |
| -------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **§ 155 HSchG**                              | Sachkosten trägt der Schulträger — alles, was nicht das Land nach §§ 151–154 trägt.                                                                    |
| **§ 158 Abs. 1 HSchG**                       | Der Schulträger stattet die Schule „mit den notwendigen Lehrmitteln, **Büchereien**, Einrichtungen …" aus und unterhält sie.                           |
| Leitfaden Schulbibliotheken (HKM, Nov. 2023) | Errichtung und Ausstattung sind Sache des Schulträgers; selbst die Bibliothekssoftware „wird aus dem Budget des Schulträgers gezahlt".                 |
| Rechtsprechung (AG, bibliotheksurteile.de)   | Das Benutzungsverhältnis einer Bibliothek ist **privatrechtlich** (Benutzungsordnung, BGB): zuerst Ersatzbeschaffung, sonst Geld in Höhe des Neuwerts. |

Folge: Der Bestand der Schülerbücherei gehört dem **Hochtaunuskreis**. Für ihn gibt es
**keinen Verwaltungsakt, keine Referenznummer, keine Rechtsbehelfsbelehrung** — sondern
eine gewöhnliche Rechnung/Zahlungsaufforderung der Schule auf Grundlage der
Benutzungsordnung, deren Geld an den Schulträger gehört. Wohin genau (Kreiskasse mit
Kassenzeichen, Budgetkonto der Schule beim Kreis, bar mit Quittung), ist **nicht
öffentlich dokumentiert** und mit dem Fachbereich Schule und Betreuung des Kreises zu
klären. Sicher ist nur: nicht aufs Landes-Schulgirokonto (Abschnitt I der
Anwendungshinweise).

---

## 2. Was die Anwendung heute kann (am Code nachgesehen)

| Baustein                             | Fundstelle                                                                                                                                                    | Stand                                                                                                                                                         |
| ------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Forderung anlegen                    | `repository/damage.go` (`ReportDamage` beendet Ausleihe + sondert aus `BESCHAEDIGUNG`; `MarkCopyDefekt` lässt die Ausleihe offen), `DamageReportModal.svelte` | ✅ inkl. Idempotenz, Schuldner aus der Ausleihe, 409 bei Neuverleih. **Betrag ist Freitext, Vorgabe 15,00 €** — keine Staffel, kein Bezug zum Preis.          |
| Forderung erledigen                  | `api/damage_resolve.go`, `repository/audit_system.go` (Bezahlt / Storno mit Grund, Audit, 409 bei Doppelbuchung)                                              | ✅ „Bezahlt" bedeutet heute **Barzahlung am Tresen**.                                                                                                         |
| Brief 1: Elternbrief je Schadensfall | `pdf/schadensfall.go`, `api/pdf.go` (`elternbrief_generiert`)                                                                                                 | 14-Tage-Frist, „bar in der Bibliothek", „Schulbibliotheksordnung", Unterschrift „Bibliotheksleitung". Für Lernmittel in vier Punkten falsch.                  |
| Brief 2: Rechnung je Schüler         | `pdf/rechnung.go`, `api/print.go` (alle offenen Forderungen)                                                                                                  | DIN 5008, LEFT JOINs (Geräteschaden vorgesehen), aber „bar in der Bibliothek", keine Nummer, kein Topf.                                                       |
| Brief 3: Eltern-Mahnbrief            | `api/reports_pdf.go`, Vorlage `MAHNUNG_ELTERN`                                                                                                                | DIN-5008-Fensterkuvert, Falzmarken, Tabelle der überfälligen Bücher — **die Bauform, die der Bescheid braucht.**                                              |
| Mahnwesen                            | `api/mahnwesen*.go`, `repository/mahnwesen_*.go`, `Mahnwesen.svelte` + `components/mahnwesen/`                                                                | Überfällige nach Klasse/Jahrgang, Auswahl → „Mahnbriefe drucken" (Mahnstufe steigt nur beim Druck), Klassenleitungs-Mail, Sperre ab `max_overdue_days`.       |
| Schulstammdaten                      | `SchuleKategorie.svelte`, `system_settings*.go`                                                                                                               | Name, Anschrift, Eigentumsvermerk. **Fehlt:** Schulnummer, Schulamt (Nr. + Anschrift), Bankverbindungen, Schulleitung, Geschäftszeichen/Bearbeiter/Durchwahl. |
| Preise                               | `buecher_exemplare.einkaufspreis` (aus Littera übernommen), `erworben_am` = Littera-Zugangsdatum (echt, nicht Importdatum)                                    | Kein Listenpreis/Neupreis.                                                                                                                                    |
| Ausleihhistorie je Exemplar          | `ausleihen` (Zeilen bleiben nach der Anonymisierung ohne Person)                                                                                              | Zählbar. Aus Littera kamen nur die **offenen** Ausleihen — für den Altbestand ist die Zahl der Verleihjahre unbekannt.                                        |
| Volljährigkeit                       | `schueler.geburtsdatum` (NULL bei Altdaten)                                                                                                                   | Anrede „Erziehungsberechtigte" vs. volljährig ableitbar.                                                                                                      |
| Rollen                               | ADMIN / MITARBEITER / KOLLEGIUM / HELFER; Sekretariat = ADMIN                                                                                                 | Neues Recht nach dem Muster `merge_students` (db/seed.go, permissionMetadata.js, schuelerRechte.js, permissions.spec.js, PII-Matrix, FACHKONZEPT).            |
| Nummernkreise                        | `barcode_seq`; Regel: EIN Generator, nie recyceln (Register 068)                                                                                              | Für die Referenznummer braucht es einen Zähler je Kassenjahr.                                                                                                 |

**Kern-Befund:** Die Forderung (`schadensfaelle`) ist die richtige Wahrheit — Sperre,
Löschblockade, DSGVO-Auskunft, Zusammenführen, Bezahlt/Storno hängen alle daran. Es fehlt
ihr die **Art** (nicht zurückgegeben / beschädigt), der **Topf** (Land / Kreis) und der
**Brief** als eigener Datensatz mit Nummer, Frist und Status. Die drei Briefe sind drei
Fassungen derselben Sache und werden zu **einem Renderer mit zwei Varianten**.

---

## 3. Anforderungsliste „Mahnverfahren der Schulen in Hessen" — Abgleich

| Nr. | Wunsch                                                                                                              | Bewertung                                                                                                                                                                                                                                                                                                                                  |
| --- | ------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | Automatische Abwertung (10 %/Jahr oder je Ausleihe)                                                                 | **Nicht als Dauerzustand bauen.** Der Erlass gibt die Staffel vor (100/80/60/40/20/10 %); sie wird beim Anlegen einer Forderung als **Vorschlag** berechnet. Ein laufend „abgewerteter Buchwert" hat außerhalb der Forderung keinen Zweck.                                                                                                 |
| 2   | Beschädigung mit Prozentwert bei Katalogisierung/Rückgabe                                                           | Zustandsnotiz gibt es (`zustand_notiz`). Ein Prozentwert ohne Forderung steuert nichts. Bei der Rückgabe „Schaden melden" → Forderung mit Staffel-Vorschlag, Betrag im Ermessen (so will es der Erlass).                                                                                                                                   |
| 3   | Einkaufspreis UND Listenpreis, wählbar                                                                              | Einkaufspreis ist da. Neupreis zum Zeitpunkt des Verlusts (ab 2. Verleihjahr) fehlt → **Entscheidung E4.**                                                                                                                                                                                                                                 |
| 4   | Übersicht Verluste/Beschädigungen                                                                                   | Gibt es (Mahnwesen, Fehlbestand, Gebühren in der Akte). Neu: Liste der Bescheide mit Fristablauf.                                                                                                                                                                                                                                          |
| 5   | Versand per Post/E-Mail/App; Referenznummer, jährlich neu                                                           | **Nur Druck/Post.** Erlass: „schriftlich", mit Rechtsbehelfsbelehrung; Datenschutz-Entscheidung A3 vom 22.08.2026 (keine Eltern-Mahnmail, § 15 SchDSV). Referenznummer ja; „Zurücksetzen zu Jahresbeginn" ergibt sich, weil das Kassenjahr Teil der Nummer ist.                                                                            |
| 6   | Zahlung bestätigen; Rückgabe → Buch wieder verfügbar, Saldo null; Zahlung ohne Rückgabe → „verloren" + **gelöscht** | Bezahlt gibt es. Rückgabe → Forderung automatisch stornieren (neu). **Gelöscht wird nie**: Leitfaden 11.3 verlangt den Nachweis der Aussonderung in der Bestandskartei → `ist_ausgesondert`/`aussonderung_grund = 'VERLUST'`, genau wie heute.                                                                                             |
| 7   | Nach Fristablauf: zweiter Ausdruck fürs Schulamt, Rest automatisch „verloren"                                       | Liste „Frist abgelaufen" + Aktion **„Ans Schulamt übergeben"** (Mensch klickt, kein Cron): Original-Nachdruck + Sammelliste, Bücher werden ausgesondert. Automatik über Nacht wäre eine stille Aussonderung — genau die Bugklasse, die dieses Projekt meidet.                                                                              |
| —   | Frist **sechs** Wochen                                                                                              | Erlass und Arbeitshilfe sagen **vier Wochen mit Datum** → Vorgabe 28 Tage, einstellbar (E3).                                                                                                                                                                                                                                               |
| —   | Importfelder Ansprechpartner 2/3 (Eltern-Namen, zweite Anschrift getrennt lebender Eltern)                          | **Nicht bauen (E8).** Das Musteranschreiben adressiert „An die Erziehungsberechtigten des/der Schülers/in" — Elternnamen braucht es nicht; LUSD liefert eine Anschrift; zusätzliche Daten widersprechen der Datensparsamkeit (SchDSV Anlage 1). Ein zweiter Brief an einen getrennt lebenden Elternteil ist ein Handfall des Sekretariats. |

---

## 4. Bauplan (bestehende Struktur, keine Parallelwelt)

### 4.1 Datenmodell (eine Migration)

```
schadensfaelle
  + art            TEXT NOT NULL DEFAULT 'beschaedigung'
                   CHECK (art IN ('nicht_zurueckgegeben','verlust','beschaedigung'))
  + bescheid_id    UUID NULL REFERENCES schadensersatz_bescheide(id)   -- Position eines Briefs

schadensersatz_bescheide           -- EIN Brief = ein Datensatz, zwei Töpfe
  id, schueler_id (RESTRICT), topf ('land'|'schultraeger'),
  kassenjahr INT, laufende_nr INT, referenznummer TEXT UNIQUE,     -- Land: SSA·Jahr·Schulnr·lfd; Kreis: SB-Jahr-lfd
  brief_datum DATE, frist_bis DATE, gesamtbetrag NUMERIC(10,2),
  empfaenger_snapshot JSONB,       -- Anrede (Erziehungsberechtigte|volljährig), Name, Anschrift zum Briefdatum
  status ('offen'|'uebergeben'|'erledigt'), uebergeben_am, erledigt_am,
  schulamt_zu_informieren BOOL,    -- Rückgabe NACH Übergabe (Erlass Nr. 2 letzter Absatz)
  erstellt_von, erstellt_am, letzter_druck_am
  UNIQUE (topf, kassenjahr, laufende_nr)

schadensersatz_nummern (topf, kassenjahr, letzte_nr)   -- der EINE Generator, UPDATE … RETURNING in der Tx
```

Die Positionen des Briefs SIND die Forderungen — keine zweite Positionstabelle, damit
Sperre, Löschblockade, Auskunft und Bezahlt/Storno unverändert weiterlaufen. Der Snapshot
des Empfängers macht den Nachdruck reproduzierbar („Kopie verbleibt in der Schule");
er wird bei der Anonymisierung mit getilgt (DSGVO-Paar-Gate), die Nummer bleibt (das
Schulamt ordnet Zahlungen darüber zu).

### 4.2 Betrag: Vorschlag, kein Automat (`pkg/ersatzwert`)

Reine Funktion `Ersatzwert(verleihjahr, kaufpreis, neupreis) (betrag, prozent, basis)`
nach der Erlass-Staffel. `verleihjahr = max(Anzahl Schuljahre mit Ausleihe des
Exemplars, Jahre seit erworben_am + 1)` — die zweite Größe fängt den Littera-Altbestand
ohne Historie ab (ein 2019 gekauftes Buch ist nicht im 1. Verleihjahr, nur weil wir seine
Ausleihen nicht kennen). Der Dialog zeigt die Herleitung („3. Verleihjahr → 60 % von
24,90 €") und der Mensch bestätigt oder überschreibt — „im Ermessen der Schule".

### 4.3 Schreibpfade

| Weg                                                                                                 | Was passiert                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| --------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `POST /api/schueler/{id}/bescheide` `{topf, positionen:[{ausleihe_id \| schadensfall_id, betrag}]}` | Eine Transaktion: Schülerzeile sperren, Positionen prüfen (gehören zum Schüler, Titel passt zum Topf via `ist_lernmittel`), für überfällige Ausleihen je eine Forderung `art = nicht_zurueckgegeben` anlegen (**Ausleihe bleibt offen, Buch wird NICHT ausgesondert** — es kann noch zurückkommen), Nummer ziehen, Frist rechnen, Snapshot schreiben, PDF erzeugen, **erst dann committen** (Muster `erzeugeUndCommitBulkMahnung`: Papier == DB), Audit. |
| `GET /api/bescheide/{id}/pdf`                                                                       | Nachdruck aus dem Snapshot, dieselbe Nummer, `letzter_druck_am`. Nie eine neue Nummer.                                                                                                                                                                                                                                                                                                                                                                   |
| `POST /api/bescheide/{id}/uebergeben`                                                               | Nach Fristablauf, von Hand: Status `uebergeben`; Forderungen `nicht_zurueckgegeben` → Ausleihe beenden + Exemplar `VERLUST` (Ablauf Nr. 7, Leitfaden 11.3). Liefert Übergabe-PDF: Original + Sammelliste fürs Schulamt.                                                                                                                                                                                                                                  |
| Rückgabe-Hook in `repository/loan.go` (`ReturnBook`)                                                | Hat die Ausleihe eine offene Forderung `nicht_zurueckgegeben`: Storno mit Grund „Rückgabe am …"; steht der Brief auf `uebergeben` → `schulamt_zu_informieren = true` + Hinweis an der Theke. Das ist der einzige Eingriff in einen Kernpfad — eigener PG-Test, am Rückbau rot gesehen.                                                                                                                                                                   |
| Bezahlt / Storno                                                                                    | unverändert (`/api/schadensfaelle/{id}/bezahlt`, `/storno`). Brief gilt als `erledigt`, wenn keine Position mehr offen ist (abgeleitet, nicht doppelt gespeichert).                                                                                                                                                                                                                                                                                      |
| `GET /api/bescheide?status=`                                                                        | Liste für das Sekretariat: offen / **Frist abgelaufen** / übergeben / Schulamt zu informieren.                                                                                                                                                                                                                                                                                                                                                           |

Recht: neu `schadensersatz_bescheide` (ab Werk nur ADMIN — das Sekretariat), Anlegen
einer Forderung bleibt `edit_students`. PII-Stufe 3 → PII-Matrix + Antwort-Gate.

### 4.4 Briefe: ein Renderer, zwei Varianten (`pdf/schadensersatz.go`)

Gebaut auf der DIN-5008-Fensterkuvert-Seite aus `reports_pdf.go` (Falzmarken,
Anschriftfeld, „(keine Adresse hinterlegt)"-Regel).

- **Land** (Musteranschreiben): Kopf mit Geschäftszeichen/Bearbeiter/Durchwahl, Betreff
  wörtlich, Text mit § 153 HSchG, ☒/☐ je Fallgruppe mit Tabelle (Name, Titel, ISBN,
  Preis), Frist als Datum, Gesamtbetrag, HCC-Konto, Referenz-Nr., Androhung Schulamt,
  Unterschrift Schulleitung, Rechtsbehelfsbelehrung. Der verbindliche Text steht **fest
  im Code an einer Stelle** (kein frei editierbares Template — eine zerschossene Vorlage
  wäre ein fehlerhafter Verwaltungsakt); Austausch gegen die 2018er Fassung ist eine
  Zeile je Absatz.
- **Schulträger**: Rechnung mit Rechnungsnummer, Frist, Tabelle, Hinweis auf
  Ersatzbeschaffung als Alternative, Zahlungsweg aus den Einstellungen — ohne Nummer im
  Landesformat, ohne Rechtsbehelfsbelehrung.
- Beide ersetzen `pdf/schadensfall.go` und `pdf/rechnung.go`; `api/pdf.go`, `api/print.go
(Rechnung)` und `elternbrief_generiert*` werden abgeräumt (Deadcode-Gate, Bestandsliste
  der Bauteile). Der Eltern-**Mahnbrief** (überfällig, ohne Geld) bleibt, wie er ist.

### 4.5 Oberfläche

- **Schülerakte → Gebühren & Schäden:** Forderungen tragen Topf und Art; Knopf
  „Bescheid erstellen (Land)" / „Rechnung erstellen (Kreis)" nur mit Recht; darunter die
  Briefe mit Nummer, Frist, Status, „Nachdruck", „Ans Schulamt übergeben".
- **Mahnwesen:** Auswahl-Modus bekommt neben „Mahnbriefe drucken" den Knopf
  **„Schadensersatz-Bescheide (Land)"** — je markiertem Schüler ein Brief über seine
  überfälligen Lernmittel plus offene Lernmittel-Forderungen; die Tabelle des Briefs
  füllt sich aus denselben Daten wie heute der Mahnbrief. Dritter Filter „Bescheide"
  (Frist abgelaufen → Übergabe).
- **Schaden melden** (`DamageReportModal`): Art (verloren / beschädigt), Staffel-Vorschlag
  mit Herleitung, Betrag überschreibbar; kein automatisches PDF-Popup mehr — der Brief
  ist ein eigener Schritt des Sekretariats.
- **Einstellungen → neue Kategorie „Schadensersatz"** (ein JSON-Schlüssel nach dem
  Muster `sommerferien`, serverseitig geprüft): Schulamts-Nr., Schulnummer, Frist in
  Tagen (28), Konto Land (Vorgabe HCC), Konto Schulträger (leer bis geklärt),
  Geschäftszeichen, Bearbeiter, Durchwahl, Name der Schulleitung, Schulamt
  (Name + Anschrift für die Rechtsbehelfsbelehrung; Vorgabe Bad Vilbel).
- **Betriebsbereitschaft:** Warnung, solange Schulnummer/Schulamts-Nr./Konten fehlen —
  ein Bescheid ohne Referenznummer ist keiner.

### 4.6 Gates (jedes am Rückbau rot zu sehen)

Nummernkreis nie doppelt und nie rückwärts (PG, parallel) · Rückgabe storniert die
Forderung und markiert nach Übergabe (PG) · Topf ≡ `ist_lernmittel` jeder Position
(PG) · Brief-Inhaltsstrom trägt Referenznummer, Frist-Datum, IBAN und
Rechtsbehelfsbelehrung (Gate am fertigen PDF, wie `etikett-gate`) · Nachdruck ändert
weder Nummer noch Betrag · Anonymisierung tilgt den Snapshot, lässt die Nummer
(DSGVO-Paar-Gate) · Recht am Draht (`permissions.spec.js`) · Staffel als Tabelle
(1→100 % Kaufpreis, 2→80 %, 3→60 %, 4→40 %, 5→20 %, 6+→10 % Neupreis).

### 4.7 Etappen

1. Datenmodell + Nummernkreis + Landes-Bescheid aus der Schülerakte + Recht + Einstellungen.
2. Mahnwesen-Anbindung (Auswahl → Bescheide) + Bescheid-Liste + Übergabe + Rückgabe-Hook.
3. Kreis-Rechnung + Betriebsbereitschaft-Warnung.
4. Altbriefe abräumen, Staffel-Vorschlag im Schaden-Dialog, Doku (FACHKONZEPT §3/§14,
   HANDBUCH, PII-Matrix, invarianten §4, SECURITY/VVT: neuer Zweck „Bescheid").

---

## 5. Was ich bewusst NICHT baue

- E-Mail-/App-Versand der Bescheide (Erlass: schriftlich; A3).
- Automatische Aussonderung per Cron nach Fristablauf (stille Bestandsänderung).
- Löschen ausgebuchter Exemplare (Bestandskartei-Nachweis).
- Eltern-Namen / zweite Anschrift (E8).
- Ein frei editierbares Template für den Verwaltungsakt.

---

## 6. Entscheidungen (Peter / Sekretariat / Schulträger)

| #      | Frage                                                                                                                                                                                  | Empfehlung                                                                                                                                |
| ------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| **E1** | Schulamtsbereichs-Nr. (4-stellig) und Schulnummer (4-stellig) für die Referenznummer — beides kennt nur das Sekretariat bzw. das Schulamt.                                             | Sekretariat fragt in Bad Vilbel nach; bis dahin Platzhalter, Betriebsbereitschaft warnt.                                                  |
| **E2** | E-Mail-Erlass vom 11.06.2018 samt aktuellem Musterschreiben beim Schulamt anfordern? Ich baue nach dem 2014er Muster; der Text ist an einer Stelle austauschbar.                       | Ja, anfordern — parallel bauen.                                                                                                           |
| **E3** | Frist 28 Tage (Erlass) oder 6 Wochen (Anforderungsliste)?                                                                                                                              | 28 Tage, einstellbar.                                                                                                                     |
| **E4** | Preisbasis ab dem 2. Verleihjahr: neues Feld „Listenpreis" am Titel (von Hand gepflegt) oder Einkaufspreis als Näherung?                                                               | Feld `listenpreis` am Titel, optional; Vorschlag nimmt Listenpreis, sonst Einkaufspreis, und sagt, welchen. Der Mensch bestätigt ohnehin. |
| **E5** | Zahlungsweg für die Schülerbücherei (Kreiskasse + Kassenzeichen? Budgetkonto? bar mit Quittung?) — mit dem Fachbereich Schule und Betreuung des Hochtaunuskreises klären.              | Bis zur Antwort: Kreis-Rechnung druckt sichtbar „(Bankverbindung des Schulträgers nicht hinterlegt)" — nie ein erfundenes Konto.          |
| **E6** | Nach Übergabe ans Schulamt: bleibt der Schüler gesperrt und die Forderung offen, bis das Sekretariat „bezahlt laut Finanzbericht" bucht — oder gilt Übergabe schulseitig als erledigt? | Sperre bleibt, Löschblockade fällt (Übergabe ≙ erledigt für die DSGVO-Kette, sonst hängt der Datensatz ewig am Schulamt).                 |
| **E7** | Kein E-Mail-/App-Versand der Bescheide — bestätigen?                                                                                                                                   | Ja.                                                                                                                                       |
| **E8** | Keine Eltern-Namen / zweite Anschrift im Datenbestand — bestätigen?                                                                                                                    | Ja.                                                                                                                                       |

---

## 7. Teil B — Beschaffung: getrennte Töpfe im Bestellwesen (09.09.2026)

**Anlass:** Peter nach einem Telefonat mit der Schule: Die Bestellungen und Rechnungen
sollen getrennt ausweisen, was aus Landesmitteln (Lernmittelfreiheit) und was aus
Kreismitteln (Schülerbücherei) beschafft wird.

### 7.1 Was verlangt wird — der Leitfaden sagt es wörtlich

| Quelle (Leitfaden Lernmittelfreiheit, HKM 2021)                                                                                                                                                                                                                      | Was daraus folgt                                                                                                                                                                                                                                                                                       |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **9.4.3** „Zur Vermeidung von Unklarheiten über die Höhe des Nachlasses **vermerkt die Schule auf der Bestellung**, ob die bestellten Bücher im Rahmen der Lernmittelfreiheit beschafft werden oder ob es sich um eine Anschaffung für die Schülerbücherei handelt." | Der Vermerk gehört auf die **Bestellung** — nicht erst auf die Rechnung. Grund ist die Buchpreisbindung: LMF-Sammelbestellungen bekommen 12 % (9.4), die Schülerbücherei den Bibliotheksnachlass (§ 7 Abs. 2 BuchPrG). Eine Bestellung, die beides mischt, kann der Händler nicht richtig rabattieren. |
| **9.4.2**                                                                                                                                                                                                                                                            | Der Nachlass setzt voraus, dass die Bücher „im Eigentum des Landes bleiben" — Bücherei-Bücher (Eigentum des Kreises) fallen nicht darunter; sie als Sammelbestellung zu deklarieren ist unzulässig.                                                                                                    |
| **10.2 / 10.3**                                                                                                                                                                                                                                                      | Auf der LMF-Rechnung bestätigt die Schule Inventarisierung, sachliche und rechnerische Richtigkeit, Verwendungszweck; **die Originalrechnung geht unverzüglich an die Schulaufsichtsbehörde**, Kopie zu den Akten. Die Kreis-Rechnung geht diesen Weg nicht — sie ist Sache des Schulträgers.          |
| **10.4**                                                                                                                                                                                                                                                             | Rechnungsunterlagen 10 Jahre, Lieferunterlagen 6 Jahre aufbewahren.                                                                                                                                                                                                                                    |
| **11.2 / 11.5**                                                                                                                                                                                                                                                      | Zugangsbuch: jede Lieferung mit Eingangsdatum, Titel, Anzahl, Lieferant, Inventarnummer; bei EDV-Führung **je Schulhalbjahr ein Ausdruck der Neuanschaffungen**, Bestandskartei zum 15.3. und 15.9.                                                                                                    |
| **13** (5 %-Vereinbarung)                                                                                                                                                                                                                                            | Wo Land und Schulträger es vereinbart haben, dürfen bis zu 5 % LMF-Mittel für Lehrmittel und umgekehrt verwendet werden — „die Schulen vermerken dies auf den entsprechenden Rechnungen". Der Topf ist also eine **Entscheidung je Bestellung**, die der Titel nur vorschlägt.                         |
| **2.3.1**                                                                                                                                                                                                                                                            | Bibliotheken sind Sache des Schulträgers (§§ 155, 158 HSchG).                                                                                                                                                                                                                                          |

### 7.2 Was die Anwendung heute tut (am Code nachgesehen)

- **Keine Bestellung kennt ihren Topf.** `bestellungen_verlauf` trägt Lieferant, Kundennummer,
  Betrag, Exemplare, Bestätigung — sonst nichts. Berichte, Historie, Übersicht und Detail
  können deshalb nicht trennen.
- **Das Anschreiben behauptet für JEDE Bestellung die Schülerbücherei:** `order_pdf.go`
  schreibt „hiermit bestellen wir für unsere Schulbibliothek …" und im Betreff
  „Buchbestellung für die Schulbibliothek" — auch wenn ein LMF-Klassensatz bestellt wird.
  Das ist genau der Vermerk aus 9.4.3, nur falsch herum: Ein Händler, der den Brief
  ernst nimmt, gewährt auf Lernmittel den falschen Nachlass.
- **Ein Lieferant hat EINE Kundennummer.** Händler führen für Lernmittel und Bibliothek
  oft getrennte Kundenkonten (anderer Rabatt, andere Rechnungsstelle).
- **Der Bestellbedarf ist LMF** (`reorderFilter`: Vorgabe `lmf`), die Titelsuche liefert
  alles. Ein über die DNB-Suche NEU angelegter Titel (`upsertTitelAusMetadaten`) bekommt
  **kein** `ist_lernmittel` — ein neues Schulbuch entsteht heute als Bücherei-Titel und
  bleibt es (Frist, Katalog, Löschfrist, Bestellbedarf lesen die Spalte). Das ist eine
  bestehende Lücke, die Teil B nebenbei schließt.
- Der Wareneingang hängt an `buecher_exemplare.bestellung_id` (Migration 063) — das
  Zugangsbuch (11.2) ist damit heute schon aus den Daten ableitbar, nur nicht ausgedruckt.

### 7.3 Bauplan (kleiner als Teil A)

1. **Datenmodell:** `bestellungen_verlauf.mittel TEXT CHECK (mittel IN ('land','schultraeger'))`,
   nullbar für Alt-Bestellungen; eine Backfill-Migration ordnet eindeutige Fälle zu (alle
   Positionen Lernmittel → `land`, keine → `schultraeger`), gemischte bleiben NULL und
   erscheinen in Berichten als „ohne Zuordnung (vor 09/2026)" — nie geraten.
   `lieferanten.kundennummer_schultraeger` (leer = dieselbe Nummer). Dasselbe Vokabular
   wie `schadensersatz_bescheide.topf` in Teil A → dort in `mittel` umbenennen.
2. **Eine Bestellung = ein Topf.** Der Warenkorb gruppiert seine Positionen nach dem
   Vorschlag aus `ist_lernmittel` in „Lernmittelfreiheit (Land)" und „Schülerbücherei
   (Kreis)", jede Gruppe mit eigener Summe; eine Position lässt sich in die andere Gruppe
   schieben (5 %-Vereinbarung, Fehlkennzeichnung). **„Bestellung auslösen" erzeugt je
   Gruppe eine Bestellung** — zwei Mails, zwei Anschreiben, zwei Bestätigungslinks an
   denselben Händler. Der Server nimmt `mittel` je Bestellung an und speichert es;
   Pflichtfeld für neue Bestellungen.
3. **Der Vermerk auf Bestellung und Mail:** Anschreiben-Betreff und -Text nach Topf
   („Bestellung im Rahmen der Lernmittelfreiheit — Sammelbestellung, Eigentum des Landes
   Hessen" / „Anschaffung für die Schülerbücherei — Schulträger Hochtaunuskreis"),
   Kundennummer des Topfs, Rechnungsanschrift des Topfs aus den Einstellungen.
   Neuer Platzhalter `{{.Mittel}}` für die Vorlage `BESTELLUNG_HAENDLER`
   (Platzhalter-Paritäts-Gate zieht mit).
4. **Berichte getrennt:** Monats- und Jahresbericht in zwei Blöcken (Land / Kreis) mit
   je eigener Summe, Gesamtsumme darunter; die Lieferantenabrechnung bekommt den
   Topf-Filter — sie ist das Blatt, gegen das die Händlerrechnung geprüft wird (10.2).
   Bestellhistorie: Spalte/Chip „Land"/„Kreis" + Filter; Kennzahlen der Übersicht je Topf;
   Bestelldetail zeigt den Topf.
5. **Neuer Titel aus der DNB-Suche:** Das Staging-Fenster fragt „Lernmittel
   (Lernmittelfreiheit)?" und setzt `ist_lernmittel` — schließt die Lücke aus 7.2.
6. **Einstellungen:** EINE Kategorie „Mittel & Konten" für beide Teile: je Topf
   Bezeichnung, Rechnungsanschrift/-vermerk, Bankverbindung (Teil A), Schulamts-/Schulnummer
   (Teil A). Betriebsbereitschaft warnt, solange die Kreis-Angaben fehlen.
7. **Später, kein Teil dieses Pakets:** Zugangsbuch-Ausdruck je Schulhalbjahr und Topf
   (11.5) aus dem Wareneingang — die Daten liegen vollständig vor.

Gates: Bestellung ohne `mittel` → 400 (PG) · gemischter Warenkorb → zwei Bestellungen,
Summen stimmen (PG) · Anschreiben-Inhaltsstrom trägt den Vermerk des Topfs und nie den
anderen (PDF-Gate) · Berichte: Land + Kreis + ohne Zuordnung = Gesamt (PG) · DNB-Titel mit
Haken entsteht als Lernmittel (PG).

### 7.4 Entscheidungen (Teil B)

| #      | Frage                                                                                                                                                                                                                | Empfehlung                                                                                                                                                     |
| ------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **D1** | Eine Bestellung = ein Topf; ein gemischter Warenkorb wird beim Auslösen in zwei Bestellungen geteilt (zwei Mails an denselben Händler).                                                                              | Ja — anders kann der Händler weder Nachlass noch Rechnung sauber trennen (9.4.3).                                                                              |
| **D2** | Hat der Händler getrennte Kundenkonten für Lernmittel und Bibliothek? (Sekretariat fragen.)                                                                                                                          | Zweite Kundennummer je Lieferant, optional; leer = dieselbe.                                                                                                   |
| **D3** | Rechnungsanschrift und Wortlaut je Topf: Land = an die Schule, Original geht ans Schulamt (10.3). Kreis = an die Schule oder direkt an den Kreis? (Fachbereich Schule und Betreuung fragen — dieselbe Frage wie E5.) | Bis zur Antwort steht auf der Kreis-Bestellung nur der Vermerk „Anschaffung für die Schülerbücherei (Schulträger Hochtaunuskreis)"; keine erfundene Anschrift. |
| **D4** | Alt-Bestellungen rückwirkend zuordnen, wo es eindeutig ist; gemischte bleiben „ohne Zuordnung"?                                                                                                                      | Ja.                                                                                                                                                            |
| **D5** | Besteht zwischen Hochtaunuskreis und Land eine 5 %-Vereinbarung (Leitfaden 13)? Dann muss das Verschieben einer Position in den anderen Topf auf dem Beleg vermerkt werden.                                          | Sekretariat/Schulamt fragen; das Verschieben gibt es ohnehin, der Vermerk kommt dazu, falls ja.                                                                |
