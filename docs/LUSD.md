# LUSD-Import — Schülerdaten ohne Schüler-ID

Stand: 2026-09-02. **Das eine Dokument zum LUSD-Import:** Was die Schule exportieren muss,
wie der Import Schüler wiedererkennt, was mit Abgängern passiert, und wie man eine falsche
Zuordnung repariert. Der Code steht in `api/lusd_*.go` und
`repository/lusd_bestand.go`; die fachliche Kurzfassung im [Fachkonzept §8](FACHKONZEPT.md),
die Bedienung im [Handbuch](HANDBUCH.md), die Abnahme in der
[Abnahme-Checkliste §1](abnahme_checkliste.md). Die Messwerte einer Simulation in
Schulgröße (1.890 Schüler, drei Schuljahre) stehen in
[lusd-simulation-2026-09-02.md](lusd-simulation-2026-09-02.md).

**Für wen welcher Abschnitt:**

| Ich bin … | Lies |
|---|---|
| Sekretariat / Schulleitung — „Was muss ich exportieren?" | §1 |
| Admin, der den Import fährt | §2, §3, §5, §6 |
| Bibliothek — „Ein Kind steht mit altem Ausweis an der Theke" | §5 |
| Datenschutz — „Was wird gespeichert, wann gelöscht?" | §4, §7 |
| Entwicklung — „Wo steht was im Code?" | §8 |

---

## 1. Was die LUSD liefern muss

Die LUSD (Landesschuldatenbank Hessen) ist das **führende System** für Schülerdaten
(Entscheidung 02.09.2026): Was sie liefert, überschreibt den Bestand, leere Felder lassen
den Bestand stehen. Handkorrekturen an Adressen überleben den nächsten Import nicht.

**Weg in der LUSD:** EXTRAS → BERICHTE → Individueller Bericht (immer XLSX). Der
verschlüsselte Littera-Export (`Littera_Export.txt`) ist für dieses System unbrauchbar.
**Eine Schüler-ID enthält kein LUSD-Bericht** (offizielle Feldliste des HMKB, geprüft
26.08.2026; „Schueler_Schluessel" ist nur `Nachname,Vorname,Geburtsdatum`).

| Spalte im Bericht | Pflicht | Wofür |
|---|---|---|
| `Schueler_Vorname`, `Schueler_Nachname` | ja | Name |
| `Klassen_Klassenbezeichnung` | ja | Klasse (feste Schreibweise „05F1") |
| `Schueler_Geburtsdatum` | **dringend** | der Zuordnungsschlüssel — ohne Datum nur Stufe „nur Name" |
| `Schueler_Eintritt_AktuelleSchule` | empfohlen | **zweiter Schlüssel** gegen Namensänderungen (§3) |
| `Schueler_Straße`, `Schueler_Postleitzahl`, `Schueler_Ort` | optional | Anschrift für Mahnbrief und Ersatzforderung |

Nicht gelesen (bewusst): `Ansprechpartner_Alle_*` — mehrere Kontakte je Schüler, Adresse
und Mail der Eltern wären ein Ratespiel („letzte Zeile gewinnt" = Großeltern). Eltern-Mail
bleibt leer, bis geklärt ist, welcher Kontakt die Mahn-Mail bekommen soll.

Formate: CSV (Komma/Semikolon, BOM) oder XLSX; mehrere Blätter (eines je Klasse) werden als
eine Tabelle gelesen; Kopfzeilen in drei Stilen (`Vorname`, `Schueler_Vorname`,
`SLR_Vorname`). Altes `.xls` wird mit Anleitung abgewiesen.

## 2. Wie der Import Schüler wiedererkennt — drei Stufen

Die Datei entscheidet, die Vorschau nennt die Stufe im Banner:

| Stufe | Wann | Sicherheit |
|---|---|---|
| **LUSD-ID** | Datei hat eine ID-Spalte mit Werten | sicher — kommt in der Praxis nicht vor |
| **Name + Geburtsdatum** | keine ID, Datum vorhanden | der Regelfall; Datum muss in jeder Zeile stehen |
| **nur Name** | weder ID noch Datum (LANIS-Klassenliste) | Notweg; Namensgleiche werden nie zugeordnet, nur gemeldet |

Gedächtnis `lusd_bestaetigt_am`: Jeder Import stempelt jede Zeile, die er im Export
wiedergefunden oder angelegt hat. **Abgänger** ist, wer bestätigt war und jetzt fehlt.
Handanlagen, die nie in einem Export standen, bleiben unangetastet („nicht im Export").

Was ohne Schüler-ID **nicht** von selbst geht: eine **Namensänderung** in der LUSD
(Schreibkorrektur, Umlaut, zweiter Vorname fällt weg, Adoption) oder ein **korrigiertes
Geburtsdatum**. Der Schlüssel findet dann niemanden — bis zum 02.09.2026 hieß das
Abgänger + Neuanlage, und der Abgänger ohne Bücher wurde sofort anonymisiert. Littera
hatte dasselbe Problem (Abgleich über Vorname + Zuname + Geburtsdatum, gedruckte
Abgängerliste zum Abhaken). Die Antwort darauf sind §3 bis §5.

## 3. Umbenennung erkennen: die Vorschau schlägt Paare vor

Aus den Abgängern und den Neuzugängen **einer** Vorschau bildet der Import Paare, die nach
allem, was der Export sonst hergibt, dieselbe Person sind
(`api/lusd_paarung.go`). Signale, absteigend nach Gewicht:

1. **Schuleintritt** (`Schueler_Eintritt_AktuelleSchule` → `schueler.schul_eintritt_am`,
   Migration 094) — übersteht jede Umbenennung. Ein „Neuzugang", dessen Eintritt Jahre
   zurückliegt, ist fast sicher keiner.
2. Geburtsdatum, Name (Vor- oder Nachname, normiert: Umlaut, Bindestrich, zweiter Vorname),
   Klasse (gleich oder Nachbarjahrgang: „05F1" ↔ „06F1"), Anschrift (Straße + PLZ).

| Einstufung | Bedingung | Vorschau |
|---|---|---|
| **sicher** | Schuleintritt + Name, oder Schuleintritt + Geburtsdatum + Vorname | vorangekreuzt |
| **vermutlich** | Geburtsdatum + ein weiteres Signal; oder Name + Klasse/Anschrift bei abweichendem Datum (Datumskorrektur) | angeboten, nicht angekreuzt |

Zwei Schutzregeln (Rasterdurchgang 02.09.2026): Ein **abweichender Vorname bei gleichem
Geburtsdatum** ist ein Gegen-Signal — so sehen Zwillinge aus — und macht ein Paar nie
„sicher" (Grund nennt „Vorname abweichend (Zwilling?)"). Bei **Gleichstand** zweier
Kandidaten mit exakt gleich starken Signalen wird kein Paar gebildet, statt zu raten;
das Kind bleibt Abgänger + Neuzugang, und der Admin führt bei Bedarf über die Akte
zusammen (§5).

Der **Admin** (Recht `import_students`) kreuzt an. Bestätigt heißt: derselbe Datensatz
bekommt den neuen Namen bzw. das korrigierte Geburtsdatum und behält UUID, Ausweis-Barcode,
Ausleihen und Historie. Nicht bestätigt: Abgänger + Neuanlage wie bisher. Jeder Abgänger und
jede Zeile stehen in höchstens einem Paar; bei Konkurrenz gewinnt das stärkere. Auch
Abgänger **früherer** Läufe sind Kandidaten, solange sie nicht anonymisiert sind — dafür
gibt es die Karenzzeit (§4). Paare gibt es nur im Modus Name + Geburtsdatum: Im Nur-Name-Modus
trägt ohne Datum kein Signal, im ID-Modus löst die LUSD-ID jede Umbenennung selbst.

Der Server nimmt nur Paare an, die diese Vorschau selbst vorgeschlagen hat; eine fremde
Kombination wird mit 400 abgewiesen, nicht geraten.

## 4. Abgänger: Karenzzeit statt sofortiger Anonymisierung

Wer bestätigt war und im Export fehlt, wird Abgänger. Seit dem 02.09.2026:

| Zustand | Was passiert |
|---|---|
| offene Ausleihe oder unbezahlter Schaden | gesperrt, Name und Anschrift bleiben (Mahnung/Rechnung), Grund „Automatisierte Abgänger-Sperre (offene Vorgänge)" |
| nichts offen, Karenz > 0 | **nur gesperrt**, Grund „… (Karenzzeit vor Anonymisierung)"; der nächtliche Job anonymisiert nach Ablauf |
| nichts offen, Karenz = 0 | sofort anonymisiert (Verhalten bis 02.09.2026) |

Die **Karenzzeit** steht in *Einstellungen → Datenschutz & Sitzung → Abgänger-Karenzzeit*
(Vorgabe **90 Tage**, Schlüssel `abgaenger_karenz_tage`). Ihre Uhr ist
`schueler.abgaenger_seit` (Migration 094): gesetzt beim ersten Abgang, nicht verschoben
bei weiteren Läufen, geräumt bei Rückkehr. Import, nächtlicher Job
(`jobs/cron_dsgvo.go`) und Selbstprüfung lesen **denselben** Schlüssel und rechnen mit
**demselben** Prädikat (`repository.PredikatAnonymisierung`).

Warum 90 und nicht länger: Die endgültige Löschung (`RunGDPRDeleteAbgaenger`, 30. Januar
des Folgejahres) bleibt unberührt und deckelt die Frist ohnehin. Warum nicht 0: In der
Karenz lässt sich eine falsche Zuordnung noch reparieren (§3 beim nächsten Lauf, §5 von
Hand). Ein anonymisierter Datensatz ist dafür verloren.

## 5. Reparatur von Hand: zwei Datensätze zusammenführen

Fällt eine Dublette erst später auf — das Kind steht mit dem alten Ausweis an der Theke und
gilt als gesperrter Abgänger — führt ein Admin (Recht `manage_students_admin`) beide
Datensätze zusammen: **Schülerakte → Stammdaten & Adresse → „Doppelter Datensatz?"**.

Regeln (`repository/schueler_zusammenfuehren.go`):

- Das **Ziel bleibt** (UUID, Ausweis-Barcode). Faustregel: Es bleibt der Datensatz, dessen
  Ausweis das Kind in der Hand hat.
- Die **Quelle geht auf**: Ausleihen, Gebühren, Vormerkungen (haben beide denselben Titel
  vorgemerkt, bleibt die weiter fortgeschrittene — abholbereit vor wartend), Foto (wenn
  das Ziel keines hat) und Protokollspuren wandern; danach
  wird die Zeile endgültig gelöscht — kein Papierkorb.
- **LUSD führend:** Stammdaten kommen vom Datensatz mit dem jüngeren `lusd_bestaetigt_am`;
  ein **aktiver** Datensatz schlägt dabei einen Abgänger (der Abgänger ist der Stand, den
  die LUSD nicht mehr kennt — etwa wenn das umbenannte Kind von Hand neu angelegt wurde
  und der alte Datensatz wegen des Ausweises bleibt); leere Felder füllt der andere.
- Das Ziel ist danach aktiv; eine automatische Abgänger-Sperre fällt, sofern nichts offen
  ist (sonst „Sperre wegen offener Vorgänge"). Eine **manuelle Sperre bleibt — auf beiden
  Seiten:** War die Quelle manuell gesperrt (Ausweis gestohlen, Hausverbot), trägt das
  Ziel danach diese Sperre mit ihrem Grund.
- Die Kandidatensuche im Dialog sieht **auch Abgänger und Gesperrte** — die Aktivliste der
  Schülerdatei blendet sie aus. Anonymisierte lassen sich nicht zusammenführen.
- **Rückweg:** In derselben Transaktion entsteht in `audit_log` (Tabelle `schueler`,
  Aktion `ZUSAMMENGEFUEHRT`, am Ziel) ein Eintrag mit den vollständigen Stammdaten der
  Quelle, dem Stand des Ziels davor und den Kennungen jeder gewanderten Zeile. Ein falsch
  bestätigtes Paar lässt sich daraus von Hand wieder trennen (Quelle neu anlegen, Zeilen
  zurückschlüsseln — `TestZusammenfuehren_RueckwegAusDemProtokoll` geht den Weg). Wird
  das Ziel später anonymisiert, ersetzt die Tilgung diesen Eintrag als Ganzes.
- Admin-Protokoll: `SCHUELER_ZUSAMMENGEFUEHRT` in `audit_logs` mit beiden Barcodes und den Zählern.

## 6. Der Ablauf beim Schuljahreswechsel

1. Sekretariat: Bericht mit den Spalten aus §1 exportieren (Geburtsdatum und Schuleintritt!).
2. Admin: *Einstellungen → Schuljahreswechsel* → Datei → **Vorschau laden**.
3. Banner lesen (Stufe), dann die Rubriken: **Vermutlich dieselbe Person** zuerst — Paare
   prüfen und ankreuzen; dann Neue, Zusammengeführt (Geburtsdatum/ID nachgetragen),
   Klassenwechsel, Rückkehrer, Abgänger, Nicht im Export, Nicht abgleichbar, Mehrdeutig.
4. **Import finalisieren.** Bei mehr als 30 % Abgängern verlangt der Server die zweite
   Bestätigung (Massenabgang) — beim echten Schuljahreswechsel normal.
5. Gegenprobe: je Rubrik zwei, drei Schüler in der Schülerdatei aufrufen.
6. Danach die Versetzung (Klassen hochzählen), falls der Export noch die alten Klassen trug.

## 7. Datenschutz in einem Absatz

Neu gespeichert wird nur `schul_eintritt_am` (Datum, keine besondere Kategorie; bewusst
**nicht** Geburtsort oder Geschlecht — der Eintritt reicht für die Paarung und ist das am
wenigsten personenbezogene Feld). Es wird mit dem Datensatz anonymisiert und steht in der
Art.-15-Auskunft. Die Karenzzeit **verkürzt** die Aufbewahrung gegenüber vorher (Abgänger
wurden bisher nach 360 Tagen anonymisiert, jetzt nach 90) und verlängert nichts: Wer
offene Vorgänge hat, blieb schon immer stehen. Die Kandidatensuche und die Vorschau liegen
auf PII-Stufe 2 (kein Adressfeld in der Antwort, Gate `pii_antwort_gate_pg_test.go`).

## 8. Für die Entwicklung: wo steht was

| Frage | Datei |
|---|---|
| Kopfzeilen, Aliase, Pflichtspalten | `api/lusd_header.go` |
| Parsen, Modus-Erkennung, Dubletten in der Datei | `api/lusd_parser.go`, `api/lusd_parser_quelle.go` |
| Bestandsindex je Modus | `api/lusd_bestand.go`, `repository/lusd_bestand.go` |
| Klassifizierung (neu / Wechsel / Rückkehrer / Abgänger / mehrdeutig) | `api/lusd_klassifizierung.go` |
| Umbenennungs-Paare | `api/lusd_paarung.go` |
| Anwenden: Batch, Neuanlage, Abgänger, Karenz | `api/lusd_apply.go` |
| Lauf, Bremse, Wahl des Admins, Audit | `api/lusd.go` |
| Zusammenführen | `repository/schueler_zusammenfuehren.go`, `api/student_zusammenfuehren.go` |
| Karenz-Prädikat, Job, Wächter | `repository/loeschfristen.go`, `jobs/cron_dsgvo.go`, `repository/loeschrueckstand.go` |
| Oberfläche | `frontend/src/lib/components/students/LusdImportView.svelte`, `LusdUmbenennungen.svelte`, `lusdVorschauRubriken.js`, `SchuelerZusammenfuehren*.svelte` |
| Tests, die den Jahreszyklus spielen | `api/lusd_namensmodus_pg_test.go`, `api/lusd_umbenennung_pg_test.go`, `api/student_zusammenfuehren_pg_test.go`, `api/lusd_paarung_test.go` |

Grenzen, die bleiben: Ohne Schüler-ID sind zwei Zeilen mit gleichem Namen **und** gleichem
Geburtsdatum in einer Datei eine Person (zusammengelegt, nicht „mehrdeutig"). Eine
Umbenennung **und** ein Umzug **und** ein Klassensprung im selben Jahr ohne Schuleintritt
im Bericht ergibt kein Paar — dann §5.
