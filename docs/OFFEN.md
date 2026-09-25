# Offene Arbeit

Stand: 25.09.2026

**Die eine Liste.** Hier steht alles, was noch zu tun, zu prüfen oder zu entscheiden ist — Code,
Betrieb und Schule. Einen zweiten Ort gibt es nicht. Erledigtes wird gelöscht, nicht archiviert:
Die Geschichte steht in den Commit-Nachrichten und in `git log -p docs/OFFEN.md`.

Andere Dokumente erklären (Konzept, Anleitung, der Katalog der Bugklassen in
[sweeps.md](sweeps.md)), führen aber keine eigene Offen-Liste.

---

## Was jetzt dran ist

**Die Sichtung vom 16.09.2026** (Abschnitt 9): Die zwei Bedingungen aus 9.9 — DSGVO-Nachweis
sowie Hosting- und Pflegekonzept — sind am 23.09.2026 zurückgestellt. Das Pflegekonzept ist am
24.09.2026 umentschieden und liegt als Entwurf vor; der DSGVO-Nachweis bleibt zurückgestellt.

**Was bei dir liegt — der Reihe nach:**

1. **Nach dem nächsten Update eine Kontrollzählung** (7.8), lesend: Migration 147 räumt die
   drei Protokolleinträge zu gelöschten Lesern; die Zählung muss danach 0 zeigen.
2. **Die Vorschläge vom 25.09.2026 bestätigen oder ändern** — zur Aktualität der Images (7.8)
   und zu den vier offenen Stellen des Pflegekonzepts (9.9, Entwurf in
   [PFLEGEKONZEPT.md](PFLEGEKONZEPT.md)). Danach baue ich in drei Stufen: `update.sh` holt das
   Datenbank-Image und baut ohne Cache; ein Release-Modus für den Schulserver; das Pflegekonzept
   mit den Antworten und eine Vorlage für das Blatt. Dazu nur du: ob die Vertretung schon
   benannt ist.
3. **Die Anfragen an Schule, Schulamt und Schulträger** (Abschnitt 8 und 7.2), soweit noch nicht
   gestellt: Littera-Backup (7.2), B3 und B4 (8.5), E1 und E2 (8.1, 8.2), die Zahlungswege in
   zwei Schritten — erst die Schule, dann der Schulträger (8.3) —, die Sperre der Ehemaligen
   beim Schulbuch (8.7), die Abholfrist bei Vormerkungen (8.8), dazu der Wortlaut des
   Eigentumsvermerks der Schülerbücherei (Einstellungen → Schule; leer heißt, diese Bücher
   tragen keinen Vermerk). Den Echtstart halten diese Antworten auf, nicht der Code.
4. **Der Nachweis von Hand für die Theke ohne Netz** (Abschnitt 2, Stufe 1 und 3 im echten
   Chrome) — zurückgestellt am 24.09.2026. Stufe 2 (die Tür per curl) mache ich am lokalen
   Stack, wenn der Nachweis ansteht.
5. **Der Etiketten-Lauf im Druck-Center** (4.8): ohne Zeitdruck — nötig vor Abnahme-Flow 4, für
   den es noch keinen Termin gibt; mit einem Neuaufbau aus 7.2 entfällt er.

**Im Code, in dieser Reihenfolge** (freigegeben am 23.09.2026; die Stellung von 5.3 ist der
Vorschlag vom 24.09.2026, 4.18 ist am 25.09.2026 nach vorn gezogen):

1. **4.18** Auflagen eines Schulbuchs — nur noch die Messung am Testserver (lesend, Einzeiler
   dort); sie entscheidet, ob die Titelmaske eine Liste mit Vorschlägen zum Zusammenfassen braucht.
2. Die am 24.09.2026 entschiedenen kleinen Punkte — 5.21 (Palettenfarben, Bildschirm für
   Bildschirm), 5.18 (Klassen als Stammdaten, mit Frage-Runde zur Oberfläche).
3. **5.3** — muss stehen, bevor ein echter Bescheid übergeben wird (echte Bescheide gibt es ab
   der Antwort zu E1), und es schließt eine Lücke, die heute schon besteht.
4. Nach der Antwort zu 8.3: **5.4**.
5. **5.10** (Gates und Werkzeuge) und Abschnitt 6 nur mit Anlass.

**In der Doku:** das Pflegekonzept (9.9) — der Entwurf steht seit dem 24.09.2026; es folgen die
Arbeitsnotizen ins Repository und die Probe durch die Vertretung. **Gleich danach im Code:** das Gate gegen Leser-Werte im Protokoll (5.10), entschieden am
24.09.2026.

Mit dem Littera-Backup (7.2) kommen die Littera-Schlagworte aus 4.20. Vor einem zweiten
Personenlauf auf derselben Datenbank muss das Aufräumen stimmen (**5.24**).

**Parallel auf der Schulseite:** Abschnitte 7 und 8 — zuerst S3 (7.3), das Littera-Backup (7.2),
die Anfragen E1, E2 und zu den Zahlungswegen (8.1–8.3), B3 und B4 (8.5) und ein Termin für die
Abnahmen (7.7). Einen echten LUSD-Import erst nach der Littera-Übernahme (7.2).

**Was liegen bleiben darf:** die übrigen B-Punkte in Abschnitt 5, die Beobachtungen in 6 und die
Betriebspunkte in 7. Keiner davon schadet still; sie werden gebündelt erledigt.

---

## So wird die Liste geführt

Vor jedem Fund steht dieselbe Frage — nicht „ist das hässlich?", sondern **„kann das still
jemandem schaden?"**

|       | Kategorie                                                                                                                                                                                            | Umgang                                                               |
| ----- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| **A** | Kann **stillschweigend** ein falsches Ergebnis für einen echten Menschen erzeugen: doppelte Mahnung an Eltern, Daten beim falschen Empfänger, verlorene Eingabe, ein Gate, das nicht rot werden kann | **Sofort**, eigener Commit, eigener Test, eigene Deploy-Entscheidung |
| **B** | Fehler, der sich **laut** meldet, oder Unordnung ohne Wirkung nach außen: totes Codestück, wackeliger Test, Doppelung                                                                                | Hier notieren, gebündelt abarbeiten                                  |
| **C** | „Wenn ich schon mal hier bin" — Umbenennungen, Stilfragen, Refactorings ohne Anlass                                                                                                                  | Nur mit Anlass und Zeit                                              |

1. **Ein Fund = ein Commit.** Was beim Reparieren zusätzlich auffällt, kommt hierher, nicht in
   denselben Commit.
1. **Kategorie A wird belegt, nicht behauptet:** ein Test, der am alten Code rot wird. Bis ein
   Fund nachgestellt ist, heißt er „Verdacht".
2. **Neues kommt nur hierher** — Funde, offene Fragen, Betriebspunkte. Kein Issue, kein anderes
   Dokument. Eine Frage steht hier, bevor die Antwort kommt.
3. **Erledigt heißt:** hier löschen — Datum und Begründung stehen in der Commit-Nachricht, ein
   Archiv gibt es nicht. Eine Antwort bekommt „Entschieden am …" und fällt weg, sobald
   sie umgesetzt ist. Die Nummer eines gelöschten Punkts wird nicht wieder vergeben —
   Kommentare im Code nennen sie als Herkunft.
4. **Die Reihenfolge** wird bei jeder Änderung mitgepflegt.

---

## 2. Offline-Betrieb der Theke — der Nachweis steht aus

Gebaut sind alle drei Stufen (15./16.09.2026): Die Theke hält die Buch-Barcode-Liste im Browser
und nimmt jede Scan-Form offline an, der Sync schickt an `POST /api/action/nachbuchen`, und was
nicht durchging, steht als Meldungsliste am Band. Wie sich das verhält, steht in
[FACHKONZEPT.md](FACHKONZEPT.md) 18.4 und im [Handbuch](HANDBUCH.md).

Offen ist **der Nachweis (2.3):** Stufe 1 und 3 gehören von Hand in den echten Chrome, Stufe 2
über die Tür.

### 2.3 Nachweis am Stack (je Stufe, echter Chrome)

- Stufe 1: DevTools-Drosselung 20 s, Schüler laden, Buch, Escape → IndexedDB trägt den Schüler.
  Offline „zurückgeben" im Profil → Rückgabe. Lehrkraft laden, Buch scannen → Ausleihe auf die
  Lehrkraft. IndexedDB blockiert → „NICHT gespeichert". Zwei Sicherungen einspielen → ein
  Stapel. Zwei parallele Anfragen mit einem Schlüssel gegen `/api/action` → eine Ausleihe.
- Stufe 2: curl mit Cookie gegen `/nachbuchen`: Umbuchung, Doppelscan, gesperrter Ausweis,
  Rückgabe vor älterer Ausleihe (409 `veraltet`), abgebrochener Online-Versand mit gleichem
  Schlüssel (Ausleihe nachgeholt), verloren gemeldetes Buch (Forderung endet, Hinweis in der
  Meldung), zwei parallele Aufrufe auf ein Exemplar, ein Online-Scan parallel zum Nachbuchen
  desselben Kindes (kein Deadlock); `docker stop` der Datenbank → 503 → Eintrag bleibt;
  `pg_stat_activity` beim Nachbuchen von 200 Einträgen (eine Verbindung je Eintrag).
- Stufe 3: Netz aus, Bücher aller Formen und zwei Ausweise scannen, ein Band, kein Vollbild,
  20 Minuten warten ohne Sperre, Netz an → Sperre; anmelden, nachgebucht, Meldungen an einem
  zweiten Browser; Format-1-Sicherung eines anderen Rechners einspielen.

### 2.4 Nicht im Umfang

Anmeldung ohne Server · Schülerdaten auf dem Rechner · Umbuchen im Online-Weg · Geräte offline ·
scannende Person im Protokoll (nachgebucht wird unter dem beim Sync angemeldeten Konto; steht in
der Doku).

## 4. Entscheidungen

Die Nummern bleiben fest; beantwortete Fragen fallen weg, sobald sie umgesetzt sind.

### 4.4 E6: Nach der Übergabe an die Schulaufsicht

**Heute** setzt die Übergabe nur Status und Zeitpunkt (`Uebergebe`, `repository/bescheid.go`).
Die Forderung bleibt offen, und weil die Unterlagen eine Rückmeldung nur für eingegangenes Geld
vorsehen, meist für immer: Der Abgänger wird nie anonymisiert oder gelöscht
(`PredikatAnonymisierung`), Löschen von Hand lehnt der Server ab, die Selbstprüfung meldet ihn
nach einem Jahr. Die Oberfläche sagt dagegen schon: „Der Fall liegt bei der Aufsicht; die Schule
veranlasst nichts mehr." (`bescheidStatus.js`)

**Die Unterlagen der Schule beantworten die Frage** (nachgelesen am 24.09.2026). Arbeitshilfe,
Abschnitt 2: „Das weitere Verfahren wird im Staatlichen Schulamt geführt … Weitere Schritte sind
durch die Schule nicht zu veranlassen." Anforderungsliste (`Ablauf Mahnverfahren.pdf`), Punkt 7,
zum Ausdruck für das Schulamt: „Der Saldo der betroffenen Leser wird entsprechend bereinigt."
Littera kennt keine Übergabe; dort bucht man einen offenen Betrag von Hand aus, um den Leser
löschen zu können. **Entschieden am 24.09.2026: Mit der Übergabe ist der Fall für die Schule
erledigt**; zu melden bleibt eine spätere Rückgabe, das kann die Theke schon. Umbau: 5.3.

### 4.8 Etiketten-Altbestand nachtragen — gemessen, der Lauf steht aus

Der Lauf geht über das Druck-Center („Fehlende Etiketten" → „Altbestand aufräumen", mit
Vorschau und Stichtag).

**Gemessen am Testserver am 21.09.2026** (Exemplare ohne Etikett-Vermerk, nicht ausgesondert,
nach `erworben_am`): 30.654 Exemplare ohne `B-`-Nummer, alle am 15.07.2026 — dem Tag der
Littera-Übernahme. Jede `B-`-Nummer liegt danach: 23.07. (8, davon 1 im Zulauf), 31.07. (7),
02.08. (1), 10.09. (2, beide im Zulauf), 16.09. (9). Der Lauf vergleicht
`erworben_am <= Stichtag`; **der Stichtag 15.07.2026 trifft genau den Altbestand** und lässt
die 27 Neuzugänge offen.

**Der Lauf selbst:** Stichtag 15.07.2026 eintragen; die Vorschau muss 30.654 zeigen. Den ganzen
Lauf nimmt nichts zurück (einzelne Exemplare holt „Etikett zurücksetzen" in der
Nachdruck-Liste zurück). Am Schulserver vorher dieselbe Zählung wiederholen — die Zahlen oben
gelten für den Testserver:

```sql
SELECT (barcode_id LIKE 'B-%') AS b_nummer, (bestellstatus IS NOT NULL) AS im_zulauf,
       erworben_am::date AS tag, count(*)
FROM buecher_exemplare
WHERE etikett_gedruckt = false AND ist_ausgesondert = false
GROUP BY 1, 2, 3 ORDER BY 3, 1;
```

**Wann:** vor Abnahme-Flow 4. Kommt eine neue Littera-Übernahme mit Neuaufbau (7.2), erledigt
sich der Punkt — der Import setzt den Vermerk seit dem 16.08.2026 selbst.

### 4.18 Neue Auflage eines Schulbuchs — ein Werk über den Auflagen

Gebaut am 25.09.2026 in sechs Stufen, das Raster über das ganze Vorhaben ist gelaufen (Commits mit
„4.18" in der Nachricht, `git log --grep=Rasterdurchgang`). Wie es arbeitet, steht in
[FACHKONZEPT.md](FACHKONZEPT.md) unter „Auflagen eines Schulbuchs". Offen ist nur die Messung:

**Messung am Testserver, lesend** — wie viele Lernmittel schon in mehreren Auflagen im Katalog
stehen (gleicher Titel, gleicher Verlag); die Zahl entscheidet, ob die Titelmaske zusätzlich
eine Liste mit Vorschlägen zum Zusammenfassen braucht:
`docker exec bibliothek-db psql -U postgres -d bibliothek -c "SELECT count(*) AS gruppen, coalesce(sum(n),0) AS titel, count(*) FILTER (WHERE mit_bestand > 1) AS gruppen_mit_bestand_in_mehreren FROM (SELECT count(*) AS n, count(*) FILTER (WHERE EXISTS (SELECT 1 FROM buecher_exemplare e WHERE e.titel_id = b.id AND NOT e.ist_ausgesondert)) AS mit_bestand FROM buecher_titel b WHERE b.ist_lernmittel GROUP BY lower(trim(b.titel)), lower(trim(coalesce(b.verlag, ''))) HAVING count(*) > 1) x;"`

### 4.20 Littera-Schlagworte übernehmen

**Freigegeben am 23.09.2026:** die **Littera-Schlagworte** (MAB 710) beim nächsten Einspielen
des Backups mitnehmen (7.2). Heute liest der Import sie, leitet das Fach ab und verwirft sie.
Vorher messen, wie viele Titel welche tragen. Die Schlagworte selbst (Migration 138, Pflege mit
Verweisen, Portal-Filter) stehen in [FACHKONZEPT.md](FACHKONZEPT.md).

---

## 5. Abarbeitbar (Kategorie B)

### 5.3 Die Übergabe schließt die Forderung ab (nach 4.4)

E6 ist am 24.09.2026 bejaht (4.4). Fertig sein muss es spätestens mit der Antwort zu E1 (8.1) —
ab dann gibt es echte Bescheide und vier Wochen später die erste Übergabe. Eine Stufe, jeder Punkt mit einem Test, der am Rückbau rot
wird; das Modell der Stufen 1 und 2 beschreibt das [Handbuch](HANDBUCH.md).

- **Abschließen:** Bis dahin zählt eine übergebene Forderung an der Theke weiter als Hinweis:
  Sie hält Bücherei und Gerät an, übergehbar (FACHKONZEPT §2.2). `Uebergebe` setzt in derselben
  Transaktion `ist_bezahlt` an den offenen Forderungen, wie Zahlung und Storno — die 17 Dateien, die nach offenen Forderungen fragen,
  folgen von selbst, die Karenz startet über den Trigger aus Migration 137 (`aktualisiert_am`
  mitsetzen). Dazu ein Kennzeichen je Forderung: Ein Brief kann bezahlte und offene Positionen
  mischen.
- **Rückfrage:** „Übergeben" bucht heute ohne Rückfrage (`BescheideTabelle.svelte`). Übergeben
  wird nur, wenn nicht gezahlt wurde — gezahlt wird aber aufs Konto des Landes, das die Theke
  nicht sieht (wer es einbucht: 8.3). Der Dialog nennt Referenznummer und Betrag und fragt nach
  dem Zahlungseingang.
- **Späte Rückgabe:** `VerbucheRueckkehr` muss übergebene Forderungen ausdrücklich mitnehmen,
  sonst entfällt „Schulaufsicht informieren" (PG-Test). `entferneSchuelerPIIUndLoesche` löscht
  heute jede erledigte Forderung mit dem Leser; eine übergebene bleibt ohne Person stehen wie der
  Bescheid, sonst findet die Theke nach der Löschung nichts.
- **Anzeige:** Akte, Auskunft und Protokoll zeigten sonst „bezahlt" — dritter Zustand „an die
  Schulaufsicht übergeben"; ebenso die Abhilfe in `pruefeEhemaligeOffen` und der Theken-Satz „Die
  Forderung bleibt bis dahin offen".
- **Übergabe-PDF:** Original-Nachdruck und Sammelliste für das Schulamt.
- Vorher am Testserver zählen (lokal 0):
  `SELECT count(*) FROM schadensfaelle f JOIN schadensersatz_bescheide b ON b.id = f.bescheid_id WHERE b.status = 'uebergeben' AND NOT f.ist_bezahlt;`

**Bis dahin (Verdacht, am Code gelesen):** Die Akte bietet „Bezahlt" und „Stornieren" auch auf
einem übergebenen Bescheid, die Selbstprüfung rät nach einem Jahr dazu — und nach einem Storno
löst eine spätere Rückgabe „Schulaufsicht informieren" nicht mehr aus. Wirkt erst mit echten
Bescheiden.

### 5.4 Schadensersatz Teil A, Etappen 3 und 4 (nach 8.3)

Konzept: [mittel_konzept.md](mittel_konzept.md), Abschnitt 4.7.

- **Kreis-Rechnung:** zweite Variante des Renderers (Nummernkreis `SB-Jahr-lfd`, Frist, Tabelle,
  Hinweis auf Ersatzbeschaffung, Zahlungsweg aus den Einstellungen; ohne Referenznummer im
  Landesformat, ohne Rechtsbehelfsbelehrung). Solange die Bankverbindung des Schulträgers fehlt,
  steht sichtbar „(Bankverbindung des Schulträgers nicht hinterlegt)". Warnung in der
  Betriebsbereitschaft. Topf in die Referenznummer, sonst kollidieren Land und Kreis an der
  UNIQUE-Spalte. Heute lehnt der Server jeden Topf außer `land` mit 409 ab.
- **Zahlungen nach 8.3**, wie Littera sie führte (Zahlungsart, Beleg, Auswertung nach Zeitraum
  und Zahlungsart): Zahlungsart beim Knopf „Bezahlt", Quittung als PDF, beide Wege auf der
  Rechnung, Liste der Barzahlungen je Zeitraum und Topf — auch für die Ausnahme beim Land (Bargeld
  binnen 14 Tagen weiterleiten, [mittel_konzept.md](mittel_konzept.md) 1.1). Der Hinweis
  „bereits bezahlt" nennt dann, wann und wie; an ihm zeigt sich eine doppelte Zahlung.
- **Topf einer Forderung, eine Regel:** Der Bescheid nimmt `ist_lernmittel` am Titel, das
  Eigentum den Topf der Bestellung (`ExemplarTopfSQL`). Beim Altbestand ist das dasselbe; bei
  einem aus dem anderen Topf bestellten Exemplar ginge das Geld an den, dem das Buch nicht gehört.
  Vorher am Testserver zählen (lokal 0):
  `SELECT count(*) FROM buecher_exemplare e JOIN buecher_titel t ON t.id = e.titel_id JOIN bestellungen_verlauf bv ON bv.id = e.bestellung_id WHERE bv.mittel <> CASE WHEN t.ist_lernmittel THEN 'land' ELSE 'schultraeger' END;`
- **Altbriefe entfernen** (es geht darum, ob es sie neben dem Bescheid überhaupt weiter geben
  soll): Elternbrief `pdf/schadensfall.go` ← `api/pdf.go` (`GenerateDamagePDFHandler`) ←
  Route in `api/routes_students.go` — die Oberfläche ruft ihn seit dem 15.09.2026 nicht mehr
  auf (`33e92c44`), erreichbar ist er nur noch über die Adresse; Rechnung `pdf/rechnung.go` ←
  `api/print.go` ← `GET /api/print/rechnung/{schueler_id}` in `api/routes_system.go` ← Knopf
  in `students/SchuelerDokumente.svelte`; dazu `api/print_rechnung_pg_test.go` und
  `elternbrief_generiert*`. Der Eltern-Mahnbrief bleibt.
- **Doku:** FACHKONZEPT Abschnitt 3 (Mahnwesen ohne Bescheid) und 14 (PDF-Rechnung, Barzahlung am
  Tresen); SECURITY und VVT-Entwurf mit dem Zweck „Schadensersatz-Bescheid". Den VVT-Satz
  vorziehen, bevor die Schule den Entwurf beschließt (8.5).
- **Release** beim Abschluss.

### 5.5 Bestand, Katalog, Druck

- Listenimport gegen den Nummern-Wächter (Migration 131): Trägt eine Zeile der Datei die
  Ausweisnummer eines Lesers als Buch-Barcode, lehnt der Wächter ab und der ganze Import
  bricht mit der rohen Datenbankmeldung ab (`ON CONFLICT DO NOTHING` fängt nur den Index,
  nicht die Ausnahme). Laut, also richtig — nur die Meldung nennt weder Zeile noch Weg.
  Kategorie C, bis es einmal vorkommt.
- „Klasse" neben der Spanne (22.09.2026): Zwei Jahrgangsangaben am Titel, „Klasse"
  (`grade_level`) und „von … bis" (`jahrgang_von/bis`). Mahnwesen „nach Jahrgang", Inventur
  nach Klasse und die Mehrjahresband-Frist lesen nur die Spanne; Titel-Tabelle,
  Klassenzuweisung und Listenfilter lesen die Klasse, der Portal-Filter liest beide. Die
  Zusammenlegung (Migration 135) ist zurückgenommen: Sie machte aus Klasse N die Spanne N
  bis N, und das trifft die Daten nicht. Lesend gemessen auf dem Testserver: 153 Titel mit
  Klasse, 129 davon Klasse 6–13 bei der Vorgabe 5 bis 10, 89 dieser 129 Lernmittel.
  Mehrjährige Bände tragen ein einziges Jahr („Natur und Technik - Biologie 7 - 10" und
  „Pontes Gesamtband": Klasse 7), Klasse und Signatur widersprechen sich („Forum Geschichte
  4 (Schulbuch Klasse 9)": Signatur Ges9, Klasse 10). Woher die Werte stammen, ist nicht
  belegt. Seit dem 22.09.2026 rät der Listenimport keine Klasse mehr (früher: erste Zahl im
  Titel, sonst 5) und schreibt ohne Spalte „klasse" NULL; eine Klasse 5 aus einem älteren
  Listenimport ist von einer gepflegten nicht zu unterscheiden. Nächster Schritt: je Titel
  entscheiden, welche Spanne gilt (Liste per Einzeiler
  unten), dann die Spalte mit genau diesen Werten ablösen. Dabei mitentscheiden: die Spalte
  „klasse" des Listenimports und der Klassenvorschlag der ISBN-Suche, der auch aus
  „Band 2", „Level 9" und jeder Zahl von 5 bis 13 im Titel eine Klasse macht.
  **Entschieden am 24.09.2026, im selben Umbau:** „Jahrgang unbekannt" wird eine eigene Vorgabe
  (NULL) statt 5 bis 10 — heute ist beides nicht zu unterscheiden, und wer den
  Mehrjahresband-Schalter (Migration 134) auf einem Titel mit der Vorgabe umlegt, bekommt die 10.
  Die drei Leser der Spanne (Mahnwesen „Jahrgang", Inventur, Portal-Filter) lernen „unbekannt"
  mit. Vorher am Testserver messen.

  ```sql
  SELECT grade_level, jahrgang_von, jahrgang_bis, ist_lernmittel, signatur, titel
  FROM buecher_titel WHERE grade_level BETWEEN 1 AND 13
  ORDER BY ist_lernmittel DESC, signatur NULLS LAST, titel;
  ```

- **ISBN-10 und ISBN-13 desselben Buchs:** Die Normalform trennt beide bewusst (Migration 133),
  die Littera-Übernahme behält eine gültige ISBN-10. Seit dem 25.09.2026 rechnet die Bestelltür
  um und schlägt den Titel unter der anderen Länge vor (4.18, Stufe 4; `isbnutil.AndereForm`,
  der Zwilling von `isbnFormen.js`). Nur die Schreibweise vergleichen weiter die Markierung
  „Vorhanden" der Bestellsuche (`sammleExistierendeISBNs`) und die Dublettenkontrolle der
  Maske: Ein DNB-Treffer, dessen ISBN-10 im Katalog steht, heißt in der Trefferliste „Neu",
  erst der Klick führt zur Frage. Gemessen am Testserver am 23.09.2026 (lesend): 100 Titel mit
  ISBN-10, 9.743 mit ISBN-13, 4 Paare mit gleichem Kern — alle aus der Littera-Übernahme vom
  15.07.2026, ohne Exemplare. Unter `3499500252` und `9783499500251` stehen zwei verschiedene
  Bücher („Heinrich Mann" und „Frédéric Chopin", rororo): Die ISBN-10 trägt ein falsches
  Prüfzeichen (richtig wäre `3499500256`; die Prüfung `KlaereISBN` der Übernahme gibt es seit dem
  04.08.2026), und die Rechnung führt von ihr trotzdem auf die ISBN-13. Deshalb wird
  vorgeschlagen, nicht still zusammengeführt.
- **Frage: Etikett-Knopf nur bei `B-`-Nummern?** Der Knopf an der Exemplarkarte übergibt das
  Exemplar ans Druck-Center, steht aber nur bei Nummern mit `B-` — so seit seinem ersten Commit
  (`7daf4ac6`, Juni 2026), ohne Begründung. Littera-Exemplare tragen nackte Mediennummern und
  bekommen keinen; das Druck-Center druckt sie über „Fehlende Etiketten" (4.8 plant genau das).
  Vorschlag: den Knopf bei jeder Nummer zeigen außer den Platzhaltern `AUTO-` und `SYS-`.
- Druck-Center, Buch-Etiketten: Bei 1280 px Fensterbreite schiebt sich die A4-Vorschau (feste
  140 mm, `LabelPreview.svelte`) über den rechten Rand der Layout-Optionen; die Pfeile der
  Auswahlfelder liegen darunter (Sichtprüfung 24.09.2026). Mit 5.21 für diesen Bildschirm.

### 5.10 Gates und Werkzeuge

- **Kein Gate gegen Leser-Werte im Protokoll, die die Tilgung nicht kennt** (entschieden am
  24.09.2026: bauen, nach dem Pflegekonzept). Dreimal derselbe Fehler: LUSD-ID (6d01f27a,
  August), Ausweisnummer (131a534a, 02.09.2026), Sperrgrund (5b50202d, 24.09.2026). Jedes Mal
  kam ein Wert der Leserzeile oder ein Freitext neben `schueler_id` in `audit_logs`, und die
  Tilgung entfernte ihn nicht (im August fehlte der Schritt für `audit_logs` noch ganz). Die
  Rundreise
  (`api/dsgvo_paar_rundreise_pg_test.go`) sieht nur Werte, die sie selbst anlegt. Vorschlag:
  eine Ratsche über jeden Audit-Eintrag mit `schueler_id`, die jeden weiteren Schlüssel als
  „getilgt" oder „bleibt" eingeordnet verlangt; die Liste „getilgt" muss mit der Anweisung
  übereinstimmen. Ausgangspunkt: Die Suche nach `"schueler_id":` findet am 24.09.2026 zehn
  Stellen in `api/` und `internal/`, als Map-Literal und als JSON-Text. Ob das alle Schreiber
  sind, ist nicht geprüft; das klärt der Bau.
- Die Schema-Gegenrichtung ist blind für UNIQUE, Teilindizes und RESTRICT.
- Kein Gate gegen unbegrenzte Listen-Endpunkte.
- Kein Rückweg für ältere Sicherungen beim Wechsel des `BACKUP_ENCRYPTION_KEY`.
- `e2e/icon-trefferflaechen.spec.js` und `e2e/icon-tooltips.spec.js` messen die Bestellhistorie,
  legen aber keine Bestellung an: Allein oder ohne eine `bestell…`-Spec davor laufen sie in die
  Zeitüberschreitung (lokal am 23.09.2026 und am 25.09.2026, Bestellhistorie leer; der globale
  Teardown löscht die E2E-Bestellungen). In der
  vollen Suite legt eine alphabetisch frühere Spec sie an. Nach dem Muster von `seedBenutzer`
  selbst anlegen.
- **`ubuntu-latest` wechselt ab 19. Oktober 2026 auf Ubuntu 26.** Alle zehn Jobs in
  `.github/workflows/` laufen auf `ubuntu-latest`, am 25.09.2026 das Abbild `ubuntu-24.04`
  (Ubuntu 24.04.5, Log des CI-Laufs zu 31de3b62); jede CI-Annotation kündigt den Wechsel an
  (actions/runner-images#14748). Was dabei bricht — etwa Postgres-Client oder die
  Chromium-Abhängigkeiten von Playwright —, meldet sich in der CI. Vorher entscheiden: einen
  Lauf gegen das neue Abbild, oder `ubuntu-24.04` festschreiben und den Wechsel selbst legen.

### 5.18 Klassen als Stammdaten — wie die Lesergruppen in Littera

Am 23.09.2026 entschieden: „Klasse löschen möglich machen". Beim Bauen stellte sich heraus, dass
die Beschreibung, auf der die Entscheidung stand, nicht stimmte:

- Die Auswahllisten lesen **nicht** die Tabelle `klassen`, sondern die Verweise:
  `GET /api/klassen` ist `SELECT DISTINCT klasse FROM schueler`, dazu Klassensätze und
  Zuordnungen. `klassen` ist ein Vokabular, das Trigger selbst füllen; im Go-Code liest es
  niemand.
- Gemessen am Testserver (23.09.2026, lesend): 108 Klassen im Vokabular, 30 ohne jeden Verweis
  (05A–08D aus Migration 079). Die sieht niemand; ein Löschen änderte nichts Sichtbares.
- Sichtbar ist eine vertippte Klasse, solange Schüler, ein Klassensatz, eine Zuordnung, eine
  Reservierung oder ein LMF-Termin sie tragen. Löschen verweigert dann die Datenbank
  (`ON DELETE RESTRICT` an sechs Tabellen).

**Littera** (Handbuch, „Lesergruppen"): Klassen sind Stammdaten unter „Stammdaten →
Lesergruppen", mit Kurzbezeichnung und Bezeichnung unter einer Obergruppe; am Leser ist die
Klasse ein Pflichtfeld und wird nur aus dieser Liste gewählt. Gepflegt wird an der einen Stelle
(für die gleich gebaute Systematik: „einzelne Gruppen löschen, bearbeiten oder ergänzen"). Beim
Import aus der Schulverwaltung entstehen die Gruppen selbst.

**Entschieden am 24.09.2026, nicht gebaut — Stammdaten-Seite wie in Littera:** Die Tabelle
`klassen` wird die eine Liste, und alle Auswahllisten lesen sie statt `SELECT DISTINCT` über die
Verweise. Eine Pflegeseite (Einstellungen → LUSD & Versetzung) zeigt jede Klasse mit der Zahl der
Schüler, Klassensätze und Zuordnungen: „umbenennen in …" zieht über `ON UPDATE CASCADE` alles mit;
gibt es das Ziel schon, werden die Verweise dorthin umgehängt (Zusammenführen), und die alte
Klasse fällt weg; löschen nur ohne Verweis. Der LUSD-Import legt neue Klassen weiter selbst an.
Wer Schüler umhängt, ändert ihre LMF-Termine und Klassensätze mit — die Rückfrage nennt die
Zahlen. In Stufen, vorher eine Frage-Runde zur Oberfläche.

### 5.19 Lesepfade gegen die Sicht `schueler` — was offen bleibt

**Offen aus dem Umbau der Auskunft (gebaut am 24.09.2026):** Die Rohdaten der Protokolleinträge
(`details`) stehen nur in der abgerufenen Auskunft, nicht auf dem Blatt; das Gate
`TestDsgvoPDF_DrucktJedeAngabeDerAuskunft` führt sie als begründete Ausnahme, seit dem
24.09.2026 auch die Details der Kontoereignisse. Offen ist, was davon aufs Blatt gehört.
Nachgesehen am 24.09.2026: Die bearbeitende Person steht in eigenen Spalten (`bearbeiter_id`,
`admin_id`), die die Auskunft nicht ausgibt; Freitexte in den Details — etwa der Grund einer
Sperre (`LESER_GESPERRT`, `LESER_ENTSPERRT`; bis zum 24.09.2026 auch `OVERRIDE_BLOCK`) —
können aber andere Personen nennen.

**Beim Bau der Auskunft für Kollegen gefunden (24.09.2026), jeweils am Code nachgesehen:**

- Ein gelöschter Kollege bleibt ohne Frist im Papierkorb: `PredikatAnonymisierung` nimmt nur
  `art = 'schueler'` (nötig, weil `chk_leser_nur_schueler_werden_abgaenger` die Anonymisierung
  eines Kollegen verbietet), und eine andere Routine gibt es nicht; entfernt wird er nur von
  Hand (Papierkorb → Endgültig löschen, Recht `manage_students_admin`). Die Auskunft sagt das
  so. Offen: eine Frist für gelöschte Kollegen.
- Klassensatz-Reservierungen löscht kein Job; erledigte Wünsche und Meldungen fallen nach der
  eingestellten Frist (Vorgabe 365 Tage).
- Wird nur das Konto gelöscht (Benutzer & Rechte), bleibt die Leserzeile mit Ausweis stehen
  (`loescheUnberuehrteLeserzeile`). Die Einträge über das gelöschte Konto — `USER_CREATE` und
  `USER_UPDATE` mit seiner `ziel_id`, der Löscheintrag in `audit_log` mit Name und E-Mail-Adresse —
  zeigen danach auf kein Konto mehr, und die Auskunft der Person findet sie nicht. Abhilfe:
  beim Löschen die `leser_id` in den Löscheintrag schreiben und die Auskunft darüber suchen
  lassen.
- Ein Kollege mit Zugang, der über die Leserdatei angelegt wird (`POST /api/schueler` mit
  Adresse), hinterlässt keinen Protokolleintrag; über Benutzer & Rechte entsteht `USER_CREATE`.
  Zwei Türen zum selben Zustand, die Rechenschaft hängt an der Tür.

**Auf einer anderen Anlage vorher zählen:** Vormerkungen und Schadensfälle an einem Kollegen
sehen die Lesepfade gegen die Sicht nicht — die Warteschlange geht über eine solche Vormerkung
hinweg. Am Testserver waren am 21.09.2026 beide Zählungen 0; neue Vormerkungen für Kollegen
lehnt die Tür seit dem 21.09.2026 ab.

```
SELECT count(*) FROM vormerkungen v JOIN leser l ON l.id = v.schueler_id WHERE l.art <> 'schueler';
SELECT count(*) FROM schadensfaelle sf JOIN leser l ON l.id = sf.schueler_id WHERE l.art <> 'schueler';
```

### 5.20 Aus der Durchsicht von PR 631 (21.09.2026) — was offen bleibt

- **Anfrage-Log mit Dauer und Anfragekennung:** nicht gebaut, nur die Doku auf den Ist-Stand
  gezogen. Mehr Logzeilen am Schulserver sind eine Betriebsfrage.
- **Totalverlust des Servers ist nicht beschrieben** (gehört zu 7.4). Der Entwurf im PR ist
  nach eigener Angabe unerprobt und scheitert in Schritt 5: Das Backend hat nur benannte
  Volumes, die Sicherung vom zweiten Ort liegt also nicht im Container, und die entschlüsselte
  Datei entsteht in einem Wegwerf-Container und ist danach fort. Schreiben und an einem fremden
  Ziel durchspielen, nicht herleiten.
- **Zu 7.3:** Der zweite Ort muss kein S3 sein — ein zweiter Rechner per Kopierbefehl oder
  eine getauschte Platte tun dasselbe ohne Vertragsfrage. Eine Betriebsentscheidung.

### 5.21 Palettenfarben auf M3-Rollen

Stand 24.09.2026: 1426 Fundstellen mit Tailwind-Palettenfarben (`slate`, `blue`, `emerald` …),
gehalten von der Ratsche `frontend/src/lib/frontend-hygiene-farben.test.js`; Neues entsteht nur
noch auf Rollen. Umstellen ist eine Umgestaltung, keine Umbenennung: Die Palette führt sechs
Textgraustufen, M3 zwei Rollen. Für „in Ordnung" und „Achtung" gibt es seit dem 24.09.2026 die
eigenen Rollen `success` und `warning` in `styles/rollen.css` (M3, „Define custom color
roles"). Vorschlag:
Bildschirm für Bildschirm, die größten zuerst, je Portion ein Commit, am gerenderten Bildschirm
geprüft. Das Muster steht in Buchformular und Bestellfenster: Zustände über ui/StatusChip, Cover
über ui/BuchCover, Rückmeldung beim Zeigen über den State-Layer statt `hover:bg-*`, ein Fehler
über den Fehlerzustand des Feldes statt eines farbigen Kastens. Für die übrige Anwendung
freigegeben am 23.09.2026.

Der Inventur-Bildschirm steht seit dem 24.09.2026 auf Rollen (`UnifiedInventory.svelte`, die
Scan-Rückmeldung in `inventur/ScanRueckmeldung.svelte`). Offen auf demselben Bildschirm: die
beiden Dialoge (`InventoryStartModal` 32, `InventoryFinishModal` 15) und der Fehlbestandsbericht
(`inventur/FehlbestandBericht` 14). `inventur/lib/bookHelpers.js` (48) sind Farbverläufe je Fach
für selbstgebaute Cover-Platzhalter; das gehört zu 6.2 (Cover über `ui/BuchCover`).

Beim Ansehen der Inventur am 24.09.2026 aufgefallen, jeweils am Code nachgesehen:

- Ein unbekannter Barcode zeigt am Scanner den rohen Fehlertext „exemplar für inventur-scan
  nicht ladbar: no rows in result set" (`GetExemplarForInventoryScan` hüllt `pgx.ErrNoRows` ein,
  `ladeExemplarFuerScan` gibt ihn mit 404 unverändert weiter). Der Status stimmt, der Satz nicht.
- Eine verworfene Inventur steht unter „Frühere Inventuren" als „vollständig":
  `AbortInventurSession` setzt `abgeschlossen_am` wie ein Abschluss und `verloren_gemeldet = 0`,
  die Liste fragt nur `abgeschlossen_am IS NOT NULL`, und der Bildschirm schreibt bei 0 Verlusten
  „vollständig". Wer die Liste liest, hält den Bereich für geprüft.

Dazu gehört die Leiste des Ausweisdrucks in der Leserdatei (`students/AuswahlAktionsleiste`,
dunkel in Palettenfarben): Seit dem 23.09.2026 gibt es für markierte Zeilen `ui/AuswahlLeiste`
(Schlagwort-Pflege). Beim Umstellen zu klären: wohin der Hinweis „ohne Ablaufjahr" und das Feld
„Ab Feld" kommen — beides passt nicht in die 64 px hohe Leiste.

### 5.22 Fremdrückgabe über Kreuz verklemmt sich — seit Migration 137

Der Rückgabe-Trigger `trg_leser_stempel_rueckgabe` (Karenz-Uhr) sperrt bei einer Rückgabe die
Leserzeile des Ausleihers — im Normalfall immer, denn die Rückgabe liegt nach dem letzten
Stempel —, und zwar NACH der Ausleihe. Die Fremdrückgabe an der Theke sperrt vorher das Kind der
offenen Sitzung. Geben zwei Kinder an zwei Theken zugleich je das Buch des anderen ab, wartet jede
Transaktion auf die andere, und Postgres bricht nach einer Sekunde eine ab (40P01). Nachgestellt
am 23.09.2026 mit den Schritten des Codes; ohne den Trigger läuft derselbe Ablauf durch
(`TEST_DATABASE_URL=… go test -tags raster -run TestRaster_Fremdrueckgabe ./repository/`). Das
Nachbuchen zweier Theken, die über Kreuz umbuchen, hat dieselbe Folge von Sperren (nicht eigens
nachgestellt).

Wirkung: An der Theke erscheint eine Fehlermeldung, erneutes Scannen bucht. Beim Nachbuchen
wird der Eintrag „wiederholen" und läuft in der nächsten Runde durch. Keine Daten gehen verloren.
Nebenfolge derselben Sperre: Eine Rückgabe wartet, solange ein anderer Vorgang die Leserzeile
hält (etwa ein LUSD-Lauf, der diesen Schüler ändert).

**Entschieden am 24.09.2026: so lassen.** Die Abhilfe ohne Verklemmung wäre der Stempel in einer
eigenen Tabelle statt an der Leserzeile — eine Migration an der Karenz-Uhr (Löschuhr, Wächter,
DSGVO-Auskunft lesen ihn). Sie kommt beim nächsten Umbau der Karenz-Uhr mit; bis dahin steht der
Punkt hier, damit dieser Umbau ihn findet.

### 5.24 Aufräumen vor einem zweiten Littera-Lauf lässt Lehrkräfte stehen

Die Anleitung in `docs/SCRIPTS.md` (Abschnitt 1, „Wiederholung") löscht die Littera-Lehrkräfte
mit `DELETE FROM benutzer … '%@littera.invalid'`. Das hat zwei Lücken:

- Seit Migration 125 zeigt das Konto auf die Leserzeile, nicht umgekehrt: Die Zeile bleibt mit
  ihrer Ausweisnummer stehen, ohne Konto. Nachgestellt am 24.09.2026 an der Test-Datenbank
  (Konto weg, Leserzeile mit Nummer „31" da).
- Eine Lehrkraft, deren Adresse in Littera steht, bekommt diese Adresse statt des Platzhalters
  (`mailadresse` in `internal/littera/schreiber_personen.go`). Das Aufräumen trifft dann weder
  ihr Konto noch ihre Leserzeile. Im Stand von 2010 hat eine von 158 Lehrkräften eine Adresse.

Ein zweiter Lauf legt diese Lehrkräfte ein zweites Mal an und gibt ihnen eine Ersatznummer, weil
ihre Nummer an der alten Zeile hängt; ihre Karte findet an der Theke die alte Zeile. Das
Protokoll meldet jede dieser Ersatznummern. Abhilfe: die Leserzeilen vor den Konten löschen,
über `benutzer.leser_id` (`ON DELETE SET NULL`). Für Lehrkräfte mit echter Adresse fehlt ein
Merkmal, an dem das Aufräumen sie erkennt. Nötig, bevor auf derselben Datenbank ein zweiter
Personenlauf läuft, etwa mit dem frischen Backup aus 7.2.

### 5.25 Eine Forderung für ein Gerät lässt sich nicht anlegen

Die Datenbank sieht sie vor (`check_damage_item`: genau eines von `exemplar_id` und
`geraet_id`), die Rechnung an die Eltern kann sie drucken (`queryRechnungItems`), aber der
einzige Schreiber `meldeSchaden` (`repository/schaden_melden.go`) nimmt nur ein Buch-Exemplar:
Er sondert das Exemplar aus und legt die Forderung mit `exemplar_id` an. Fehlt bei der Rückgabe
Zubehör oder ist ein Gerät kaputt, gibt es keinen Weg zur Forderung; das FACHKONZEPT (Abschnitt
5) behauptete bis zum 24.09.2026 einen. Gesperrt würde nach 4.4 wie heute (Schülerbücherei und
Geräte).

### 5.26 Skripte halten die Regel der Auflagen nicht — `repair_titel_dubletten.sql` legt sie zusammen

Aufgefallen beim Raster zu 4.18 (25.09.2026), am Code gelesen. Die einmalige Reparatur vom
13.07.2026 gruppiert allein über den normalisierten Titel: Zwei Auflagen „Mathe 7" mit
verschiedener ISBN werden ein Titel, die Exemplare wandern an den Keeper, die zweite ISBN
fällt weg — entgegen „Zusammengelegt wird nichts" (4.18). Ebenso zwei gleichnamige Bücher
verschiedener Verlage. Werke räumt das Skript nicht auf; `ON DELETE SET NULL` lässt einen Titel
allein an seinem Werk zurück. Wirkt nur, wenn jemand es von Hand wieder laufen lässt — etwa
nach der Littera-Übernahme (7.2); [SCRIPTS.md](SCRIPTS.md) nennt es ohne „einmalig".
Möglichkeiten: das Skript löschen (der Anlass ist erledigt) oder über Titel, Verlag und
fehlende ISBN gruppieren und Lernmittel ausnehmen.

Dieselbe Lücke ohne Zusammenlegen (Rasterdurchgang 25.09.2026): `e2e_altlasten.sql`,
`entferne_demo_daten.sql` und `seed_demo.sql` löschen Titel per DELETE, `tabula_rasa.sql` leert
`buecher_titel` per `TRUNCATE … CASCADE` — das erreicht `werke` nicht, der Verweis zeigt vom Titel
zum Werk. Zurück bleiben Werke ohne Titel, die keine Ansicht zeigt, oder mit einem einzigen Titel, der
überall wie ein Titel ohne weitere Auflage erscheint (Titelmaske: „Keine andere Auflage
zugeordnet."). Kein Schaden; die Ratsche `auflagen_schreibpfad_ratsche_test.go` liest keine
Skripte.

### 5.27 `tabula_rasa.sql` bricht seit Migration 124 ab

Nachgestellt am 25.09.2026 an der Test-Datenbank: `ERROR: "schueler" is not a table` —
`schueler` ist seit Migration 124 eine Sicht auf `leser`. Das Skript bricht in seiner Transaktion
ab und ändert nichts; laut also. Seine Tabellenliste ist älter als die Leser-Tabelle (124, seit
125 mit dem Kollegium) und als `werke` (148, siehe 5.26). Gedacht ist es für den Schritt vor dem
Echtbetrieb. Frage: Beginnt der Echtbetrieb mit einer leeren Datenbank und der Littera-Übernahme
(7.2)? Dann fällt das Skript weg. Sonst braucht es eine neue Liste aus dem Tabellenbestand und die
Entscheidung, welche Leser bleiben.

---

## 6. Beobachten und Kategorie C (nur mit Anlass)

### 6.1 Beobachtungen

- Der Stand-Merker der Barcode-Liste (Anzahl + `max(aktualisiert_am)`) rechnet mit dem Beginn
  der Transaktion: Ändert eine lange Transaktion ein Etikett und committet nach einem kürzeren
  Schreiber, bleibt es bei 304. Nachgestellt hinter dem Build-Tag `raster`
  (`TEST_DATABASE_URL=… go test -tags raster -run TestRaster_ ./repository/`). Heute ändert nur
  `UpdateCopyBarcode` einen Barcode, als kurzer Einzelbefehl — scharf wird es erst mit einem
  Schreiber, der in einer langen Transaktion umetikettiert (Durchgang 15.09.2026).
- Ein Ausweis aus dem Altbestand ohne Vorsilbe (gemessen `B97601826457`) ist ohne Netz „unklar"
  und wird abgewiesen. Hängt am Ausweis-Neudruck und entscheidet sich mit dem frischen
  Littera-Backup (7.2).
- Band „Keine Verbindung": Der Herzschlag-Wächter (`App.svelte`, 25 s ohne `ping`) unterscheidet
  nicht zwischen „kein Ping gekommen" und „der Tab selbst stand" (Standby, eingefrorener
  Hintergrund-Tab). Beim Aufwachen wäre der Herzschlag alt und das Band stünde bis zum nächsten
  Ping, höchstens 15 s. Nicht nachgestellt.
- Schreibweise der Nummern (entschieden am 23.09.2026: am Server nichts bauen). Der Server
  schlägt Nummern exakt nach (`GetLeserByBarcode`, `GetCopyByBarcode`); die Theke
  vereinheitlicht vorher (`normalisiereScan`, nur mit Ziffer hinter der Vorsilbe) — beim
  Buchen und in der Offline-Warteschlange. Roh fragt nur die Vorschau beim Tippen
  (`/api/search`). Gemessen am Testserver am 23.09.2026: alle gespeicherten Vorsilben groß
  (40 Leser, 4.130 Exemplare, keine klein). **Anlass zum Bauen:** ein zweiter Aufrufer, der
  rohe Nummern schickt — dann die Eingabe am Server vereinheitlichen (nicht `upper(barcode_id)`,
  das nimmt den Index), mit einer gemeinsamen Fall-Tabelle für Go und JS.
- Von Hand lässt sich eine ausgeschiedene `A-`-Nummer wieder eintragen, in der Akte wie in
  „Benutzer & Rechte" — gewollt für die alte Karte eines Schülers, der zurückkommt (Migration
  146). Die Maske sagt dabei nicht, dass die Nummer schon einmal vergeben war; nur der Generator
  und die Littera-Übernahme lesen `ausweisnummern_ausgeschieden`. Anlass zum Bauen: eine alte
  Karte, die auf diesem Weg an eine andere Person gerät.
- Die Sperrprüfung liest aus dem Pool, während die Checkout-Transaktion mit `FOR UPDATE` offen ist
  (drei Abfragen über eine zweite Verbindung). Bei `MaxConns = 50` ohne Wirkung; beim Nachbuchen
  vieler Ausleihen (Abschnitt 2) beobachten.
- `golang.org/x/crypto/openpgp` (`GO-2026-5932`): kein Fix verfügbar, transitiv, kein Aufrufer im
  eigenen Code.
- Designer: Wer den Browser binnen 800 ms schließt, verliert die letzte Auto-Save-Änderung;
  `sendBeacon` bewusst nicht gebaut.
- Migration 082 (Dedupe der Vormerkungen) verlor den neueren `abholbereit`-Eintrag; auf Prod
  gelaufen, nur relevant für eine weitere gewachsene Datenbank.
- Die Rate-Limiter-Maps räumen erst ab 5.000 Einträgen; nur mit vielen frischen Adressen ein
  CPU-Thema.
- `startGDPRWorker` (`main.go`) ruft nur die Leihen-Anonymisierung und die Abgänger-Löschung;
  `RunGDPRAnonymizeOldData` läuft allein im Cron. Keine Wirkung erkennbar, die Doku beschreibt
  beide Wege.
- Portal: Das Menü „Mein Portal" verlangt `create_reservations`, `GET /api/lmf-termine` nur eine
  Sitzung — bewusst, der Inhalt ist PII-Stufe 0.
- Das Druck-Center hängt im Menü an `view_students`, seine Buch-Etiketten brauchen nur
  `view_books` und `edit_books`. Ab Werk hat jede Rolle mit `edit_books` auch `view_students`;
  wer die Rechte anders verteilt, erreicht die Buch-Etiketten nicht (Sammelpunkt wie
  „Einstellungen"). Der Etikett-Knopf der Buchakte fragt deshalb beides ab.
- Ausfallmatrix A3 und B4; A3 erst nach S3 (7.3).

### 6.2 Kategorie C

- Die Akte eines Kollegen ohne Ausweisnummer sagt am gesperrten Ausweisdruck „die Nummer steht
  in „Benutzer & Rechte""; ohne Konto hat er dort keinen Eintrag. Die Nummer kommt mit dem
  freigeschalteten Zugang (`StudentProfileActions.svelte`, `data-tip`).
- Browser-Gates: Die M3- und axe-Gates öffnen die Planer-Dialoge nicht, axe misst nur den
  Anfangszustand; kein Screenreader-Durchgang; der Ausweis-Designer geht nur per Maus.
- 16 Bestandsstellen bauen ihr Cover selbst (Liste in `frontend-hygiene-cover.test.js`, darunter
  `KlassenBuchKachel` im Portal). Umstellen beim fachlichen Anfassen, nicht in einem Rutsch.
- 3.000 Titel ohne ISBN: `inventur.SucheTextDNB` nur mit Bestätigung durch einen Menschen
  verdrahten.
- Die Altersangabe der DNB (653 „(Zielgruppe)ab 10 Jahre", `MetadatenErgebnis.Zielgruppe`) wird
  gelesen und nicht gespeichert: Es gibt keine Spalte und keinen Leser. Anlass zum Bauen: ein
  Leser, etwa ein Filter im Portal.
- Das Nachschlagen (`GET /api/lookup/{isbn}`) liefert einen Untertitel (`subtitle`), den weder
  das ISBN-Feld noch der Scan im Buchformular übernimmt. Beim Anlegen trägt ihn der Server
  nach, beim Ändern nicht (`ergaenzeFehlendeMetadatenFuerAktualisierung`) — der Zwilling zu
  `8f7610aa`, der den Untertitel sonst mitnimmt.
- `TestEtikettenkette_ZaehlerFolgtDenFilternDerListe` schickt `?bis=time.Now()` in der Zone des
  Testprozesses; unter `TZ=Pacific/Midway` ist das der Vortag und der Zähler nennt 0. Kein
  Produktfehler — im Betrieb kommt dieses Datum aus dem Browser in Berlin. Beim nächsten Anfassen
  auf `schulzeit.Zone()` umstellen (gefunden am 18.09.2026, als die volle Suite einmal unter einer
  fremden Prozesszone lief).
- Zwei Regexe für die LMF-Kennung (`pkg/lmf/lmf.go`, `internal/service/import_lmf.go`).
- `migrations/110_schadensersatz_bescheide.sql` nennt drei Barzahlungs-Briefe, es sind zwei;
  eingespielte Migrationen bleiben unverändert.
- Knopfzeile über den Reitern (Mahnwesen): kommt aus dem gemeinsamen Seitengerüst; Anlass wäre ein
  Rundgang über das Gerüst.
- Drei handgebaute Pillen-Gruppen in `StatsDashboard` statt `ui/Segmente.svelte`.
- Schriftstärke der Chips: `ui/ChipFeld`, `ui/FilterChips` und `ui/Segmente` schreiben
  `font-medium`, das im Haus 400 ist (`styles/theme-mass.css`; an `FilterChips` im Browser
  gemessen am 23.09.2026, die beiden anderen tragen dieselbe Klasse). M3 nennt für label-large
  500, die Knöpfe tragen `font-semibold` (500). Alle drei zusammen entscheiden, nicht einzeln.
- `github.com/jung-kurt/gofpdf` ist seit 2021 archiviert und steckt in 16 Dateien; gepflegt wird
  der Ableger `github.com/phpdave11/gofpdf`, den maroto mitbringt. Neue PDFs (5.3, 5.4) nicht
  mehr auf dem archivierten; die 16 beim fachlichen Anfassen umstellen, mit den PDF-Gates.
- Etikettenraster doppelt (`api/label_formats.go` und `etikettformate.js`), gehalten von
  `etikettformate-konsistenz.test.js`; am 31.08.2026 entschieden geparkt.
- Reste des Nie-verdrahtet-Sweeps: `inventur_sessions.gestartet_von` wird nie angezeigt;
  `abgaenger_jahr` in der Aktivlisten-Antwort ohne Konsument; bei den Geräten
  `ActionEvent.GeraetID` ohne Broadcast und mit Null-Zeitstempel.
- Cognitive Complexity: 32 Funktionen über 15 ohne Tests (Messung 05.09.2026); lohnend allenfalls
  `OverrideDueDateHandler` und `behandleAbgaenger`.
- `javascript:S6551` und `javascript:S8783`: begründete Dauer-Ausnahmen.
- `auth.Claims.BarcodeID` liest niemand mehr; die Ausweisnummer kommt seit Migration 125
  als LEFT JOIN aus der Leserzeile in die Sitzung, nur damit das Feld gefüllt bleibt.
- Tabellen-Inline-Felder mit 36 px: eine `size="sm"`-Variante von `Feld` erst bei Bedienbefund.
- `LabelHeight >= 30` steht zweimal (`api/label_pdf.go`, `api/schueler_etikett_pdf.go`).
- Zwei Normalformen für Namen (`repository.Suchnorm`, `normName` in `api/lusd_paarung.go`); beim
  Anfassen der Paarung zusammenführen.
- Der Paritätstest vergleicht keine COMMENTs und Seeds.
- Erbe der PR-Zulieferungen: Go-Testdateien über 200 Zeilen, ein schwacher Export-CSV-Test.
- Klone: Go 9 Gruppen (05.09.2026), Frontend 0,41 %.
- 50 Handler-Dateien in `api/` formulieren rohes SQL neben `repository/` (gezählt am 24.09.2026);
  der Bestand ist seit dem 07.08.2026 eingefroren (`handlerMitSQL` in `api/schichtung_test.go`).
  Umstellen beim fachlichen Anfassen einer Datei, nicht in einem Rutsch.
- Exemplarkarte der Buchakte (`BookExemplarCard.svelte`): Die vier Symbolknöpfe sind 14 px groß
  statt 32 px (`.icon-btn`), drei erklären sich per `title` statt `data-tip`; die Buchakte fehlt
  in `icon-trefferflaechen.spec.js` und `icon-tooltips.spec.js`. Ein 32-px-Knopf bricht die
  Kopfzeile bei 1280 px um (gemessen am 24.09.2026: Karte 70 → 100 px) — die Knöpfe brauchen
  eine eigene Zeile; eine Layoutfrage, nicht einzeln.

### 6.3 Parkdeck (bewusste Nicht-Entscheidungen)

Integer-Cent statt float64 · Bundle-Splitting · TypeScript-Migration · `inventur/` ins Haupt-API
verschmelzen · `cmd/migrate` (MySQL) löschen — seine PG-Tests sichern mit `internal/uebernahme`
geteilten Code · API-Versionierung · Mandantenfähigkeit (RLS) · Trennlinien-Durchgang (25 Dateien
mit `divide-y`, nur als eigener Durchgang mit Messung im Browser) · Zugangsbuch-Ausdruck je
Schulhalbjahr und Topf · Bestandskartei-Ausdruck zum 15.3. und 15.9. (beides nennt
[mittel_konzept.md](mittel_konzept.md), Abschnitt 7.1, als Verfahrensvorgabe; nicht gebaut) ·
ein Schüler wird Lehrkraft (die Datenbank verbietet es, `chk_leser_nur_schueler_werden_abgaenger`;
heute ein zweiter Leser, bei Häufung ein Umzugspfad wie Migration 072) · der Littera-Personenlauf
übergeht Praktikanten, Sekretariat und „Im Ausland" (`internal/littera/leser.go`) ·
Schlagwortliste drucken, Schlagwortkatalog als Datei aus- und einlesen (wie Littera, nur wenn die
Bücherei es braucht; 23.09.2026) · Verweise für Autoren (der Autor ist ein Textfeld, kein
Personensatz).

---

## 7. Betrieb (liegt bei der Schulseite)

Geplante Zielumgebung ist der Schulserver; heute ist der Hetzner-Server die einzige Instanz. Beim
Umzug gilt dieser Abschnitt dort erneut — ebenso das, was am Hetzner-Server schon erfüllt ist
(Commit-Geschichte, 13.09.2026). Fest auf den Testserver eingetragen sind dabei die Adresse in
der Schlussmeldung von `update.sh` und `DOMAIN` in `scripts/deploy.sh`.

### 7.2 Frisches Littera-Backup

`littera_sav.mdb` ist ein Stand von 2010. Ohne die offenen Ausleihen startet das System mit „alles
verfügbar"; Ausweisnummern und Exemplare nach 2010 fehlen, rund 1.350 Titel haben keine Signatur.
Anforderungen in [littera_schema_befund.md](littera_schema_befund.md). Vorher B7 (8.5). Die
teuerste offene Position vor dem Echtstart — früh bei der Schule anfragen.
**Reihenfolge:** Littera-Personen und -Ausleihen im selben Lauf übernehmen, **bevor** ein echter
LUSD-Import läuft. Der Littera-Personenlauf erkennt Schüler aus der LUSD nicht und legt sie ein
zweites Mal an ([SCRIPTS.md](SCRIPTS.md), Abschnitt 0). Das Geburtsdatum im Backup ist die Brücke
für den späteren LUSD-Abgleich — vor dem Lauf prüfen. Hat auf der Datenbank schon ein
Personenlauf stattgefunden, vorher das Aufräumen aus 5.24 richten.
**Vor dem Personenlauf:** Im frischen Backup nachsehen, ob die Tabelle `FremdLeserNummer` gefüllt
ist (im Stand von 2010 ist sie leer). Sie trägt die Nummern, die die Ausweise beim Scannen liefern;
nur mit ihr funktionieren die vorhandenen Ausweise ohne Neudruck. **Ist sie leer, muss jeder Ausweis
neu gedruckt werden:** Kein Ausweis liefert beim Scannen die Lesernummer, und Littera zählt Bücher
und Leser getrennt, beide ab 1 — fast jede Lesernummer ist auf dem Server schon die Nummer eines
Buchs, der Lauf vergibt dann `L-`-Nummern. Ist sie gefüllt, stehen die einzelnen Personen ohne Karte
mit „keine Karte in FremdLeserNummer" im Protokoll des Laufs.
**Am 17.09.2026 aufgefallen:** In `~/Downloads` und auf dem Schreibtisch liegen seit dem
09.09.2026 zwei Littera-Sicherungen vom 01. und 02.09.2026 (856 KB und 602 KB). Ob das die
Datensicherung aus dem laufenden Littera ist, ist UNGEPRÜFT — für eine Vollsicherung wären
100 MB+ zu erwarten, die Größe spricht eher für einen Teilexport. Zum Nachsehen fehlt auf dem
Rechner ein Entpacker für `.7z`.

**Rückweg zu Littera:** Bücher und Schüler behalten ihre Littera-Nummer, Lehrkräfte nicht. Soll der
Rückweg offen bleiben, vor dem Lauf nachtragen und das Littera-Backup vom Umstiegstag aufheben.

### 7.3 S3-Auslagerung der Backups

`S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY` und `S3_BUCKET` sind leer (13.09.2026); alles
liegt auf einer Platte. Nur EU oder Schulträger. Der Code ist fertig.

### 7.4 Manuelle Restore-Probe an einem fremden Ziel

Die automatische Wochenprobe im Container lief am 13.09.2026 erfolgreich; sie ersetzt die
manuelle Probe nicht. Anleitung: [resilience_and_recovery.md](resilience_and_recovery.md),
Abschnitt 2e — dabei die neuen Befehle erproben, die bisher nur am Text geprüft sind. Sinnvoll
nach S3 oder am Schulserver.

### 7.5 Externes Uptime-Signal

Fällt der Server ganz aus, meldet es niemand. Ein externer Monitor ruft alle 5 Minuten `/health`
ab ([DEPLOYMENT.md](DEPLOYMENT.md), Abschnitt 7.0). Etwa fünf Minuten Aufwand; beim Umzug neu
einrichten.

### 7.6 Ruleset `main`

PR-Pflicht entfernen (Solo-Entscheidung 30.07.2026), „Block force pushes" und „Restrict
deletions" anlassen. Am 23.09.2026 trägt das Ruleset noch `pull_request`; Pushes gehen über den
Admin-Bypass.

### 7.7 Abnahmen

Ablauf in [abnahme_checkliste.md](abnahme_checkliste.md), vorher ein Backup.

- Flows 1–3 mit dem Sekretariat: LUSD-Import, Versetzung (vor dem Schuljahreswechsel),
  Klassensatz erledigen. Dabei um Geburtsdatum und Eintrittsdatum im LUSD-Bericht bitten. Ein
  LUSD-Import mit echten Schülern erst nach der Littera-Übernahme (7.2).
- Flow 4 (Altbestand-Etiketten, nicht umkehrbar) erst nach 4.8.
- Flow 5: Selbstanmeldung einer Lehrkraft.
- Danach: Ergebnis hier eintragen; bei Parser-Auffälligkeiten die echte LUSD-Datei anonymisiert als
  Testfixture sichern.

### 7.8 Am Server nachsehen (lesend, Einzeiler)

- „Neuer Text" im Ausweis-Layout: laut Serverlesung vom 13.09.2026 in keinem Wert von
  `system_einstellungen`. Bestätigen, dann erledigt.
- Sind die Admin-Konten deaktiviert? Ist `/app/uploads/fotos` leer? Gibt es Lehrkräfte mit
  Platzhalter-Mail `@lehrer-umzug.invalid`? Braucht `repair_fach_kategorie.sql` einen zweiten
  Lauf?
- **Der Sperrgrund im Protokoll — Kontrolle nach dem Update.** Gemessen am Testserver am
  25.09.2026: 4 Einträge mit `grund` oder `reason`, davon 1 zu einem vorhandenen Leser (bleibt
  bis zu seiner Anonymisierung), 0 anonymisiert, 3 zu gelöschten Lesern. Migration 147 nimmt
  jedem Eintrag zu einer Kennung ohne Leser die fünf Schlüssel der Tilgung. Nach dem Update
  zeigt die Abfrage in der ersten Spalte 1 (die Migration ist gelaufen) und in der zweiten 0;
  dann ist der Punkt erledigt.

  ```
  docker exec bibliothek-db psql -U postgres -d bibliothek -c "SELECT (SELECT count(*) FROM schema_migrations WHERE version = '147_protokoll_verwaiste_leser.sql') AS migration_147, (SELECT count(*) FROM audit_logs a WHERE (a.details ? 'lusd_id' OR a.details ? 'barcode' OR a.details ? 'aufgeloest_barcode' OR a.details ? 'grund' OR a.details ? 'reason') AND a.details ? 'schueler_id' AND NOT EXISTS (SELECT 1 FROM leser l WHERE l.id::text = lower(a.details->>'schueler_id'))) AS verwaist_mit_schluessel;"
  ```

- **Die Postgres-Nebenversion am Server** (gefunden beim Pflegekonzept, 24.09.2026).
  `update.sh` ruft `docker compose up -d --build` auf und holt das Image `postgres:18-alpine`
  nie neu; der Datenbank-Container bleibt auf der Nebenversion des Images, das beim Anlegen
  vorlag (Major-Wechsel 31.08.2026). Nebenversionen mit Sicherheitskorrekturen erscheinen
  vierteljährlich; aktuell ist 18.6 (postgresql.org, abgerufen am 24.09.2026). Der Port ist nur
  an `127.0.0.1` gebunden, das begrenzt das Risiko. Die Lücke besteht unabhängig vom Messwert —
  auch bei 18.6 käme die nächste Nebenversion nicht an; die Zahl zeigt nur, wie weit der Server
  zurückliegt. Gemessen am Testserver am 25.09.2026: 18.6, also aktuell. **Zu entscheiden:** ob
  `update.sh` das Datenbank-Image bei jedem Update holt (dann startet die Datenbank bei einer
  neuen Nebenversion während des Updates neu) oder ob das eine eigene Wartungsaufgabe im
  Pflegekonzept wird. **Vorschlag vom 25.09.2026: bei jedem Update holen.** Postgres rät dazu
  („The community considers performing minor upgrades to be less risky than continuing to run
  an old minor version", postgresql.org/support/versioning); eine Nebenversion braucht weder
  Sicherung noch Neuaufbau. Die Datenbank sortiert unter musl ohne Sprachregeln (lokal
  gemessen: `datlocprovider` = c), ein neues Alpine im Image ändert also die Reihenfolge der
  Indizes nicht. Die CI testet bei jedem Lauf gegen denselben Tag `postgres:18-alpine`. Die
  Hauptversion bleibt im Repo festgeschrieben.

- **Die Alpine-Pakete im Backend-Image** (gefunden am 25.09.2026). Das `Dockerfile` holt
  Sicherheitskorrekturen nur über `apk --no-cache upgrade`. Der Build-Cache hält diese Schicht
  fest, solange die Zeilen davor gleich bleiben, und `update.sh` baut ohne `--pull` und ohne
  `--no-cache`; das Aufräumen in Schritt 7 entfernt nur Schichten, die eine Woche lang niemand
  benutzt hat. Lokal gemessen: Image vom 24.09.2026, die `apk upgrade`-Schicht darin drei Wochen
  alt. Der Trivy-Scan der CI prüft ein frisch gebautes Image, nicht das am Server. Zeigt die
  Zeile unten am Server mehr als eine Woche, besteht die Lücke dort ebenso. Gemessen am
  Testserver am 25.09.2026: 3 Tage — der Server liegt kaum zurück, die Lücke bleibt. **Zu
  entscheiden:** `--pull` beim Bau und eine Zeile im `Dockerfile`, die die Schicht in einem
  festen Takt neu baut, oder `--no-cache` (jedes Update baut dann alles neu). **Vorschlag vom
  25.09.2026: `--pull --no-cache`.** Genau so baut der Sicherheitsscan der CI, und er braucht
  dafür 93 Sekunden (Lauf vom 25.09.2026, Schritt „Build Docker image for scanning"); der
  Takt-Stempel spart ein, zwei Minuten und braucht eigene Mechanik. Dazu eine Regel im
  Pflegekonzept: mindestens einmal im Monat ein Update einspielen, auch ohne neue Funktionen.

## 8. Schule, Schulamt, Schulträger

### 8.1 E1: Schulamts- und Schulnummer

Für die Referenznummer der Bescheide; die Felder stehen in den Einstellungen und sind am
13.09.2026 leer. Mit den Nummern entstehen echte Bescheide; was davor zu richten war (Frist,
Nachdruck, Briefdatum), ist seit dem 23.09.2026 gebaut.

**Beim Schulamt mitfragen — das Kassenjahr:** Das Programm nimmt das Jahr der Frist, ein
Dezember-Brief mit Frist im Januar zählt also ins Folgejahr. Die Arbeitshilfe nennt das
Kassenjahr als Teil der Referenznummer, sagt aber nicht, welches Jahr gemeint ist, und verweist
„im Zweifelsfall" ans Schulamt. Die Nummer lässt sich nach dem Brief nicht mehr ändern.

### 8.2 E2: E-Mail-Erlass vom 11.06.2018 und aktuelles Musterschreiben

Die vorliegenden Unterlagen sind von 2014 und nennen eine inzwischen aufgelöste Stelle. Gebaut
ist nach dem Muster von 2014; der Text ist an einer Stelle austauschbar.

### 8.3 E5: Zahlungswege der Schülerbücherei — und wer eine Zahlung einbucht

Für die Schülerbücherei nennt der Brief keinen Weg, sondern „(Bankverbindung des Schulträgers
nicht hinterlegt)" (`pdf/zahlungsweg.go`). Es fehlen Konto und Verwendungszweck oder
Kassenzeichen, und ob die Schule Geld des Schulträgers bar annehmen darf (in öffentlichen Kassen
meist nur eine eingerichtete Zahlstelle) und wie es zu ihm kommt. **In zwei Schritten fragen**
(Vorschlag vom 24.09.2026):

1. **Die Schule** (Büchereileitung, Sekretariat): Wie wurde Ersatz für Bücher der
   Schülerbücherei bisher bezahlt, wohin ging das Geld? Wer bucht eine Zahlung ein, die auf dem
   Kontoauszug oder im Finanzbericht steht — für beide Töpfe? „Bezahlt" verbucht heute eine
   Barzahlung am Tresen. Keine Bauarbeit vor der Antwort; eine erfundene stünde als Vorgang in
   der Akte.
2. **Der Schulträger** (Fachbereich Schule und Betreuung des Hochtaunuskreises), als Vorschlag
   zum Bestätigen, an die bisherige Praxis angepasst: „Wir bieten beides an: Überweisung auf Ihr
   Konto mit der Rechnungsnummer als Verwendungszweck, und Barzahlung gegen Quittung. Das Bargeld
   geben wir einmal im Monat mit einer Liste an Sie weiter. Ist das zulässig? Welches Konto und
   welches Kassenzeichen gelten?"

**Engpass:** Ohne Antwort kein 5.4.

### 8.4 D2: Getrennte Kundenkonten beim Händler?

Führt der Händler getrennte Kundenkonten für Lernmittel und Schülerbücherei? Das Feld
„Kundennummer Schülerbücherei" am Lieferanten bleibt bis zur Antwort leer.

### 8.5 Datenschutz Teil B

Einzelheiten und Entwürfe in [datenschutz_offene_punkte.md](datenschutz_offene_punkte.md),
Abschnitt B, und in [datenschutz/](datenschutz/).

- **B1/B2** VVT und Datenschutzhinweis beschließen; die Entwürfe haben noch Platzhalter. Vorher den
  Zweck „Schadensersatz-Bescheid" ergänzen (5.4).
- **B3** Foto auf dem Schülerausweis: Erlass oder Einwilligung — vor dem ersten Ausweisdruck mit
  Foto.
- **B4** Schulischen DSB beteiligen, Schwellwertanalyse DSFA schriftlich — vor B1/B2.
- **B5** IT-Sicherheitskonzept mit dem Schulträger, Netzplatzierung.
- **B6** Rolle des Wartenden regeln (AVV oder dienstlich).
- **B7** Löschkonzept gegenüber Littera — vor der Littera-Übernahme (7.2).

Zuerst B3 und B4 anstoßen.

### 8.6 Barrierefreiheit

Gilt für das System die Pflicht zur Barrierefreiheit — mit Erklärung zur Barrierefreiheit und
barrierefreien PDFs (HTML-Druckweg oder begründete Ausnahme)? Bis zur Antwort geparkt; was die
Gates heute prüfen, steht in [FACHKONZEPT.md](FACHKONZEPT.md), Abschnitt 19.

### 8.7 Die Sperre der Ehemaligen beim Schulbuch

Die Schule am 22.09.2026: „Für die Lernmittel darf es keinerlei automatische ‚Sperrung' geben,
auch nicht eine Sperrung, die bestimmte Personen aufheben können." Gebaut ist danach
(24.09.2026, FACHKONZEPT §2.2): Beim Schulbuch hält nur eine Sperre von Hand auf. Die Sperre,
die das Programm den Ehemaligen setzt — Abschlussklasse nach der Versetzung, im LUSD-Export
nicht mehr enthalten —, zählt dort nicht; bei Bücherei und Gerät lässt sie nur die Rückgabe zu.
**Frage an die Schule:** Gilt der Satz auch für diese Kinder? Betroffen sind nur Kinder, die
noch an der Schule sind (E-Phase nach der 10R, Wiederholer); wer gegangen ist, holt keine
Schulbücher ab. Soll die Sperre auch beim Schulbuch gelten, ändert sich eine Stelle
(`pruefeSperreAmLeser`) samt ihrem PG-Test (`TestTheke_EhemaligeSperreNichtAmSchulbuch`).

**Dieselbe Frage gilt der Frist eines Schulbuchs** (am Code nachgesehen am 24.09.2026): Einem
gesperrten Kind — von Hand oder als Ehemaligem — verlängert das Programm kein Buch, auch kein
Schulbuch (`checkAusleiheGesperrt` in `api/ausleihe.go`, für die Einzelverlängerung und die
Frist von Hand), die Klassenverlängerung der Schulbücher (`GlobalExtendLMFHandler`) und der
LMF-Plan (`SetzeLernmittelFristFuerKlassenIn`) lassen es aus. Die Sperre soll zur Rückgabe
zwingen; beim Schulbuch ist das eine Folge der Sperre, die die Schule vielleicht ebenfalls
ausschließen will. Gebaut wird erst mit der Antwort.

### 8.8 Die Abholfrist bei Vormerkungen

Ein vorgemerktes Buch liegt drei Tage bereit, gerechnet ab dem Zeitpunkt, zu dem es zugeteilt
wird: bei der Rückgabe (`INTERVAL '3 days'` in `internal/service/loan_return.go`) oder beim
Nachrücken, wenn der Vorige es nicht abgeholt hat (`repository/vormerkung_nachruecken.go`). Danach verfällt die Vormerkung beim nächsten
stündlichen Lauf, und das Buch geht an den Nächsten in der Warteschlange. Wochenende und Ferien
zählen mit: Ein Buch, das freitags um 10 Uhr zurückkommt, liegt bis Montag 10 Uhr bereit; kommt
es in den letzten drei Tagen vor den Herbstferien zurück, verfällt die Vormerkung in den Ferien.

Die Leihfrist verschiebt seit dem 24.09.2026 ein Ende an einem Wochenende, Feiertag oder in den
Ferien auf den nächsten Schultag (`Tagesfrist` in `internal/service/loan_rules.go`); die
Abholfrist nicht. Littera führt eine „Maximale Reservierungsdauer" in Tagen, die die Schule
einstellt (Stammdaten → Einstellungen → Verleih); Öffnungs- und Schließtage nennt das Handbuch
nur für die Leihfrist.

**Frage an die Schule:** Reichen drei Tage? Soll die Abholfrist wie die Leihfrist auf den
nächsten Schultag fallen, und soll die Zahl einstellbar sein wie in Littera? Bis zur Antwort
bleibt es bei drei Tagen ab der Rückgabe.

---

## 9. Sichtung vom 16.09.2026

Zwölf Punkte, jeder am Code geprüft. Offen ist nur, was hier folgt; das Übrige steht in den
Commits vom 17. und 22.09.2026, die drei begründeten Abweichungen im Mahnwesen (nur Post, nie
löschen, vier statt sechs Wochen) in [mittel_konzept.md](mittel_konzept.md) Abschnitt 3.

Zwei Quellen liegen dem zugrunde: `~/Downloads/Arbeitshilfe_Mahnschreiben.pdf` (Erlass vom
17.12.2014, Az. 674.100.002-00178) und `~/Downloads/Ablauf Mahnverfahren.pdf` (die
Anforderungsliste, abgeglichen in [mittel_konzept.md](mittel_konzept.md) Abschnitt 3).

### 9.9 Zwei Bedingungen neben der Mängelliste

**Entschieden am 23.09.2026: zurückgestellt.** Beides bleibt liegen, bis es ansteht; dann gelten
die Schritte und Fragen unten. **Am 24.09.2026 für das Pflegekonzept umentschieden: jetzt, als
Wartungshandbuch** (Vorschlag am Ende dieses Abschnitts); der DSGVO-Nachweis bleibt
zurückgestellt.

Die Einschätzung am Ende des Protokolls nennt zwei Punkte, die in keinem der zwölf Mängel
stehen:

> „Ein Nachweis der DSVGO-Konformität liegt nicht vor.
> Hosting- und Programmpflegekonzepte sind nicht geplant. Dies könnte ein Ausschlusskriterium
> sein."

- **Nachweis der DSGVO-Konformität.** Das Material liegt vor und ist vollständiger, als der Satz
  vermuten lässt: VVT-Entwurf und Datenschutzhinweis ([datenschutz/](datenschutz/)), die
  PII-Matrix über jede Route ([PII_MATRIX.de.md](PII_MATRIX.de.md)), die Löschfristen samt
  nächtlichem Job. Was fehlt, ist ein Dokument, das man weitergeben kann — und die
  Beschlussfassung der Schule (8.5, B1–B7). **Nächster Schritt, bei mir:** das Dokument aus
  diesem Material zusammenstellen; Arbeit an der Doku, keine Bauarbeit.
- **Hosting- und Programmpflegekonzept.** Der Entwurf steht seit dem 24.09.2026:
  [PFLEGEKONZEPT.md](PFLEGEKONZEPT.md) — mit den drei am 24.09.2026 beantworteten Fragen
  (Betrieb, Pflege mit Vertretung, Ende der Pflege), den wiederkehrenden Aufgaben mit Takt, den
  zwei Handgriffen der Vertretung und der Messung, ob jemand anderes das Programm weiterführen
  kann. Offen:
  1. **Die vier offenen Stellen** in Abschnitt 9 des Entwurfs — bei dir, mit den Vorschlägen vom
     25.09.2026:
     - Einsatz über die eigene Schule hinaus: vorerst nein, nach einem Schuljahr Echtbetrieb neu
       entscheiden. Jede Schule braucht einen eigenen Server und eine eigene Vertretung; wer die
       Pflege für eine andere Schule übernimmt, braucht dort eine Regelung als
       Auftragsverarbeiter (8.5, B6). Die EUPL erlaubt anderen den Betrieb ohne Pflegezusage.
     - Meldeweg und Reaktionszeit: Meldung per E-Mail an die Entwicklung, die Vertretung in
       Kopie, bei Stillstand ein Anruf; nie über GitHub-Issues, weil das Repository öffentlich
       ist. Stillstand: Antwort am selben Schultag, hat ein Update ihn ausgelöst, geht die
       Vertretung auf den Stand davor zurück; Fehler ohne Stillstand: innerhalb einer Woche;
       Wünsche: mit dem nächsten Release. Für Littera bot der Hersteller einen
       „Softwarewartungs- und Pflegevertrag" mit Hotline, Fernwartung und Update-Codes an; ob
       die Schule ihn hatte, ist nicht belegt.
     - Betriebssystem und Docker: die IT des Schulträgers, mit automatischen
       Sicherheitsupdates in der Nacht, dazu das externe Signal (7.5).
     - Release oder `main`: Der Schulserver bekommt nur Releases, der Testserver folgt `main`
       als Vorstufe. Ein Release entsteht nur, wenn alle acht Prüfläufe grün sind.
  2. **Das Blatt bei der Schule** (Abschnitt 7.3 des Entwurfs) — bei dir; eine Vorlage lege ich
     an. Vorschlag: die zwei Schlüssel in einem Passwortmanager und als Papier im verschlossenen
     Umschlag im Tresor der Schule, nie per E-Mail.
  3. **Die Arbeitsnotizen der Entwicklung** (am 24.09.2026 219 Einträge) entlang der Gliederung
     des Entwurfs ins Repository — bei mir.
  4. **Die Probe:** Die Vertretung macht die Wiederherstellung an einem fremden Ziel (7.4) allein
     mit dem Dokument.

**Beides ist Voraussetzung für ein „nutzbar", nicht Beiwerk.**
