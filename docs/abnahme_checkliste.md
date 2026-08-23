# Abnahme-Checkliste: Admin-Flows mit echten Daten

> Stand: 2026-08-23. Für die Abnahme mit dem Sekretariat.
> Alle vier Flows sind technisch fertig und durch automatische Tests (Go, Vitest, E2E)
> abgesichert — die Abnahme prüft nur noch, ob die **echten Daten** (Spaltenformat der
> LUSD-Exportdatei, reale Klassenbezeichnungen, gewachsener Buchbestand) so aussehen wie
> erwartet.
>
> **Sicherheitsnetz für alle Abnahmen:** Vorher ein Backup ziehen
> (siehe [resilience_and_recovery.md](resilience_and_recovery.md)). Alle vier Flows haben zusätzlich eine
> unverbindliche Vorschau-Stufe, die **nichts verändert** — erst der jeweils letzte,
> deutlich beschriftete Button schreibt in die Datenbank.
>
> **Ausnahme beachten:** Flow 1–3 lassen sich nachbessern, Flow 4 (Altbestand-Etiketten)
> **nicht**. Dort ist die Vorschau-Zahl die einzige Kontrolle vor einer endgültigen
> Änderung — sie verdient einen zweiten Blick.

---

## 1. LUSD-Import (Schuljahreswechsel-Datenabgleich)

**Vorbereitung:** Klassenliste als **CSV oder XLSX** bereitlegen (Pflichtspalten:
Vorname, Nachname, Klasse). Der Export der Schule ist eine LANIS-Klassenliste und
enthält **weder Schüler-ID noch Geburtsdatum** — das ist bekannt und vorgesehen.

**Zuordnungsstufe zuerst lesen.** Der Import erkennt aus der Datei selbst, worüber er
zuordnen kann, und sagt es im Banner über der Vorschau:

| Stufe | Wann | Was das bedeutet |
|---|---|---|
| LUSD-ID | Datei hat eine Schüler-ID | sicherste Zuordnung |
| Name + Geburtsdatum | keine ID, aber ein Datum | sicher genug |
| **nur Name** | weder ID noch Datum | Namensgleiche werden **nicht** zugeordnet, sondern als „mehrdeutig" gemeldet — Banner erscheint als Warnung |

**Ablauf** (Verwaltung → Datenverwaltung → Schuljahreswechsel):

1. [ ] Datei auswählen → **„Vorschau laden"**. Es wird noch nichts geändert.
2. [ ] **Banner lesen:** Welche Zuordnungsstufe gilt? Bei „nur Name" ist mit
       Mehrdeutigkeiten zu rechnen — das ist kein Fehler, sondern die Schutzmaßnahme.
3. [ ] Vorschau prüfen. Angezeigt werden bis zu **sieben** Gruppen — die letzten vier
       sind die, bei denen das System bewusst NICHTS tut:
   - **Neue Schüler** → Stichprobe: echte Neuzugänge?
   - **Klassenwechsel** → Stichprobe: stimmen alte und neue Klasse?
   - **Zusammengeführt** (Bestandsschüler, den der Export eindeutig trifft) → bekommt
     fehlende LUSD-ID bzw. fehlendes Geburtsdatum nachgetragen, **kein** Duplikat
   - **Abgänger** → Stichprobe: sind die wirklich weg?
   - **Rückkehrer** → früherer Abgänger steht wieder in der Datei; prüfen, ob das
     dieselbe Person ist
   - **Mehrdeutig** → Name kommt mehrfach vor; bleibt unverändert
   - **Nicht abgleichbar** → im Bestand fehlt das Geburtsdatum; bleibt unverändert.
     Abhilfe: Geburtsdatum im Profil nachtragen, dann beim nächsten Import erneut prüfen
4. [ ] **„Import finalisieren"** → Erfolgsmeldung „Import abgeschlossen".
5. [ ] Gegenprobe: 2–3 Schüler aus jeder Gruppe in der Schülerverwaltung suchen und prüfen.
6. [ ] Bei „Mehrdeutig" oder „Nicht abgleichbar": stichprobenartig einen Fall im Profil
       nachpflegen und den Import wiederholen — die Gruppe muss kleiner werden.

**Eingebaute Bremsen:**
- Falsche Datei (fehlende Pflichtspalten, Binärmüll) → verständliche deutsche Fehlermeldung, kein Import.
- Mehr als 30 % der aktiven Schüler würden zu Abgängern → Warnung; der Import verlangt dann
  die zusätzliche rote Bestätigung **„Massenabgang bestätigen & endgültig importieren"**.
  Diese Bremse schützt vor einem versehentlichen Teilexport (z. B. nur eine Jahrgangsstufe exportiert).

**Bestanden, wenn:** Vorschau-Zahlen plausibel, Import läuft durch, Stichproben stimmen.

---

## 2. Schuljahres-Versetzung (Klassen hochzählen)

⏰ **Deadline: vor dem Schuljahreswechsel.** Reihenfolge: erst LUSD-Import abnehmen, dann Versetzung —
oder nur die Versetzung nutzen, wenn kein frischer LUSD-Export vorliegt.

**Ablauf** (Verwaltung → Datenverwaltung → Schuljahreswechsel):

1. [ ] **„Vorschau berechnen"** — der Server rechnet die komplette Versetzung durch und
   verwirft sie wieder (echter Dry-Run). Es wird nichts geändert.
2. [ ] Vorschau prüfen: Anzahl versetzte Schüler plausibel? Werden Klassen korrekt
   hochgezählt (z. B. `05a` → `06a`)? Höchste Jahrgangsstufe → Abgänger?
3. [ ] Ausführen (rote Bestätigungsstufe) → Erfolgsmeldung.
4. [ ] Gegenprobe: je einen Schüler aus niedrigster und höchster Stufe prüfen.

**Eingebaute Bremsen:** Doppellauf-Schutz (zweiter Lauf innerhalb von 10 Minuten wird abgewiesen),
Vorschau und Ausführung rechnen identisches SQL.

**Bestanden, wenn:** Vorschau-Zahlen stimmen mit der realen Klassenstruktur überein, Gegenproben korrekt.

---

## 3. Klassensatz-Reservierungen „erledigen"

**Vorbereitung:** Eine echte Klassensatz-Anfrage einer Lehrkraft (oder testweise selbst eine anlegen).

**Ablauf** (Bestellungen → Klassensatz-Reservierungen; Warteschlangen-Modell — reservieren
sperrt keinen Bestand):

1. [ ] Offene Reservierung erscheint in der Liste (Titel, Klasse, Anzahl, Anforderer **mit Namen**, „N verfügbar“).
2. [ ] Gegenprobe Warteschlange: eine zweite Reservierung desselben Titels anlegen — sie
       reiht sich **hinter** der ersten ein, und das Portal zeigt der zweiten Lehrkraft
       den Chip „… reserviert für …“ schon am Suchtreffer.
3. [ ] Bücher physisch bereitstellen und übergeben, dann **„Abschließen“**.
4. [ ] Reservierung verschwindet aus der offenen Liste; die zweite rückt nach vorn.
5. [ ] Gegenprobe: die Ausleihe der Exemplare war die ganze Zeit möglich — es gab nie
       einen geblockten Bestand.

**Bestanden, wenn:** Ablauf für das Sekretariat ohne Rückfragen verständlich ist und der
Bestand nach dem Erledigen stimmt.

---

## 4. Altbestand-Etiketten aufräumen (einmalig, **nicht umkehrbar**)

⚠️ **Der einzige Flow ohne Rückweg.** Alle anderen Aktionen dieser Checkliste lassen sich
nachbessern — diese nicht. Es gibt keinen Knopf, der einen zu weit gefassten Stichtag
zurücknimmt. Vorher ein Backup ziehen.

**Warum es das gibt:** `etikett_gedruckt` wurde bis vor Kurzem nirgends gesetzt. Für den
gesamten Altbestand steht deshalb „kein Etikett" — nicht weil keins am Buch klebt, sondern
weil es nie jemand vermerkt hat. Ohne dieses Aufräumen zeigt die Nachdruck-Liste dauerhaft
den ganzen Bestand, und der Hinweis im Bestellwesen nennt eine Zahl ohne Bedeutung.

**Vorbereitung:** Klären, **ab wann das Regal nachweislich beklebt ist**. Das weiß nur der
Betreiber — die Software kann es nicht wissen und rät deshalb bewusst nicht.

**Ablauf** (Druck-Center → **Fehlende Etiketten** → Abschnitt „Altbestand aufräumen" aufklappen):

1. [ ] Stichtag eintragen. Die Zahl der betroffenen Exemplare erscheint sofort — es wird
   noch **nichts** geändert.
2. [ ] Zahl prüfen: Liegt sie in der Größenordnung des Altbestands? Ist sie **auffällig
   nah an der Gesamtzahl der Exemplare**, ist der Stichtag zu spät gewählt.
3. [ ] Bestätigen → Meldung „*n* Exemplare als erledigt vermerkt".
4. [ ] Gegenprobe: Die Nachdruck-Liste enthält jetzt **nur noch** Exemplare aus jüngeren
   Lieferungen. Stichprobe: Ein Buch aus der letzten Lieferung muss noch dastehen.

**Die eine echte Gefahr:** Ein **zu später** Stichtag (z. B. heute). Dann verschwindet die
frische Lieferung, die noch gar kein Etikett hat, still aus der Liste — und niemand
bemerkt es, weil eine leere Liste wie ein erledigter Stapel aussieht. Im Zweifel den
Stichtag **früher** setzen: Zu wenig aufgeräumt ist folgenlos (die Zeilen bleiben sichtbar
und lassen sich jederzeit erneut aufräumen), zu viel aufgeräumt ist es nicht.

**Eingebaute Bremsen:**
- Vorschau-Zahl und Aktion nutzen dieselbe Bedingung — die genannte Zahl ist die, die
  wirklich zuschlägt.
- Ausgesonderte Exemplare bleiben unberührt.
- Bereits vermerkte Exemplare werden nicht erneut angefasst; ein zweiter Lauf mit demselben
  Stichtag meldet folgerichtig `0`.

**Bestanden, wenn:** Nach dem Aufräumen stehen genau die Exemplare auf der Liste, für die
tatsächlich noch ein Etikett gedruckt werden muss.

---

## Nach der Abnahme

- [ ] Ergebnis (bestanden / Auffälligkeiten) im [master_fahrplan.md](master_fahrplan.md) eintragen.
- [ ] Bei Parser-Auffälligkeiten mit der echten LUSD-Datei: die Datei (anonymisiert!)
  als Testfixture sichern, damit die automatischen Tests das echte Format abdecken.
