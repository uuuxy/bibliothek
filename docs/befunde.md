# Befund-Register

**Zweck:** Ein Ort für Dinge, die beim Arbeiten auffallen, aber nicht sofort angefasst
werden. Ohne diese Liste gibt es nur zwei Umgangsformen mit einem Fund — sofort
reparieren oder vergessen — und beide sind falsch, wenn das System kurz vor dem
Pilotbetrieb steht.

Ergänzt [`invarianten.md`](invarianten.md): Dort steht, was immer wahr sein *muss*.
Hier steht, was aufgefallen ist und noch nicht entschieden oder erledigt wurde.

---

## Die Einordnung

Vor jedem Fund steht dieselbe Frage — **nicht** „ist das hässlich?", sondern
**„kann das still jemandem schaden?"**:

| | Kategorie | Umgang |
|---|---|---|
| **A** | Kann **stillschweigend** ein falsches Ergebnis für einen echten Menschen erzeugen: doppelte Mahnung an Eltern, Daten beim falschen Empfänger, verlorene Eingabe, ein Gate, das nicht rot werden kann | **Sofort**, eigener Commit, eigener Test, eigene Deploy-Entscheidung |
| **B** | Fehler, der sich **laut** meldet, oder Unordnung ohne Wirkung nach außen: totes Codestück, wackeliger Test, Doppelung | **Hier notieren**, gebündelt nach dem Pilotstart |
| **C** | „Wenn ich schon mal hier bin" — Umbenennungen, Stilfragen, Refactorings ohne Anlass | **Nicht** vor dem Pilotstart |

Zwei Regeln dazu:

1. **Ein Fund = ein Commit.** Kein Anhängen verwandter Aufräumarbeiten. Was beim
   Reparieren zusätzlich auffällt, kommt in diese Liste, nicht in denselben Commit.
2. **Kategorie A wird belegt, nicht behauptet.** Ein Fund dieser Klasse braucht einen
   Test, der mit dem alten Code rot wird. Ohne diese Gegenprobe ist unklar, ob
   überhaupt etwas repariert wurde.

---

## Offen

| # | Fund | Kat. | Fundstelle |
|---|---|---|---|
| 1 | E2E-Test fällt gelegentlich um (Popup-Timing), lief in 1 von 3 Läufen rot. Ein Gate, dem man nicht ganz traut, ist ein halbes Gate. | B | `frontend/e2e/schadensfall.spec.js` |
| 2 | `SendTemplateMail` hat keinen einzigen Aufrufer mehr — Vorlagen-Versand über die Datenbank, den niemand benutzt. | B | `mailservice/mailservice.go` |
| 3 | Backup-Mails lesen weiterhin nur die Umgebung, nicht die Oberfläche. **Bewusst so**: Eine Backup-Mail muss auch dann rausgehen, wenn die Datenbank das Problem ist. | C (entschieden) | `inventur/backup_email.go` |

---

## Erledigt (2026-07-30)

Alle fünf Funde eines Tages waren dieselbe Art Fehler — etwas behauptete zu
funktionieren, und nichts prüfte die Behauptung nach:

| Fund | Was es behauptete | Was stimmte |
|---|---|---|
| `expect([200, 400, 500])` im Mail-Test | prüft den Testversand | konnte nicht rot werden; schickte Felder, die die API nie gelesen hat |
| CSRF-Ausnahme für `/api/admin` & Co. | „Inventur-Modul hat eigenes CSRF" | dieses System gab es im Code nicht; sechs Admin-Mutationen ungeschützt |
| SMTP-Einstellungen im Admin-Bereich | steuern den Mailversand | wurden nur vom Test-Knopf gelesen; echte Mails nahmen die Umgebung |
| Diagnose bei SMTP-Fehlern | steht im Formular | jede 500 wurde zu „interner Datenbankfehler" eingedampft |
| Ergebnis des Mailversands | Nachricht zugestellt | war das Ergebnis der Verabschiedung; Abbruch danach galt als Fehlschlag |

**Merksatz:** Der Fundort ist immer die Stelle, an der eine Zusicherung steht, die
niemand prüft — ein Kommentar, ein Testname, eine Eingabemaske. Wer dort sucht,
findet; wer im Code sucht, findet Schönheitsfehler.

Stand: 2026-07-30
