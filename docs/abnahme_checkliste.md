# Abnahme-Checkliste: Admin-Flows mit echten Daten

> Stand: 2026-07-10. Für die Abnahme mit dem Sekretariat, sobald LUSD-Zugriff besteht.
> Alle drei Flows sind technisch fertig und durch automatische Tests (Go, Vitest, 24 E2E-Flows)
> abgesichert — die Abnahme prüft nur noch, ob die **echten Daten** (Spaltenformat der
> LUSD-Exportdatei, reale Klassenbezeichnungen) so aussehen wie erwartet.
>
> **Sicherheitsnetz für alle Abnahmen:** Vorher ein Backup ziehen (`scripts/backup`-Ablauf,
> siehe [backup_cron.md](backup_cron.md)). Alle drei Flows haben zusätzlich eine
> unverbindliche Vorschau-Stufe, die **nichts verändert** — erst der jeweils letzte,
> deutlich beschriftete Button schreibt in die Datenbank.

---

## 1. LUSD-Import (Schuljahreswechsel-Datenabgleich)

**Vorbereitung:** Aktuellen LUSD-Export als CSV-Datei aus LUSD herunterladen
(Pflichtspalten: Vorname, Nachname, Klasse).

**Ablauf** (Verwaltung → Datenverwaltung → Schuljahreswechsel):

1. [ ] CSV-Datei auswählen → **„Vorschau laden"**. Es wird noch nichts geändert.
2. [ ] Vorschau prüfen — drei Gruppen werden angezeigt:
   - **Neue Schüler** (in der Datei, aber nicht im System) → Stichprobe: sind das echte Neuzugänge?
   - **Klassenwechsel** → Stichprobe: stimmen alte und neue Klasse?
   - **Abgänger** (im System, aber nicht mehr in der Datei) → Stichprobe: sind die wirklich weg?
3. [ ] **„Import finalisieren"** → Erfolgsmeldung „Import abgeschlossen".
4. [ ] Gegenprobe: 2–3 Schüler aus jeder Gruppe in der Schülerverwaltung suchen und prüfen.

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

**Ablauf** (Bestellungen → Klassensatz-Reservierungen):

1. [ ] Offene Reservierung erscheint in der Liste (Titel, Klasse, Anzahl, Anforderer).
2. [ ] Bücher physisch bereitstellen, dann **„Erledigen"**.
3. [ ] Reservierung verschwindet aus der offenen Liste; der geblockte Bestand ist wieder frei.
4. [ ] Gegenprobe: der Titel ist im Katalog wieder in voller Stückzahl ausleihbar.

**Bestanden, wenn:** Ablauf für das Sekretariat ohne Rückfragen verständlich ist und der
Bestand nach dem Erledigen stimmt.

---

## Nach der Abnahme

- [ ] Ergebnis (bestanden / Auffälligkeiten) im [master_fahrplan.md](master_fahrplan.md) eintragen.
- [ ] Bei Parser-Auffälligkeiten mit der echten LUSD-Datei: die Datei (anonymisiert!)
  als Testfixture sichern, damit die automatischen Tests das echte Format abdecken.
