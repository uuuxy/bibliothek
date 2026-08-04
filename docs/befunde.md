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

*(leer)*

---

## Erledigt (2026-08-04) — Littera-Schreibpfad

Fünf Funde, alle derselben Machart wie am 30.07.: Etwas behauptete etwas, und die
Behauptung war nie gegen die Wirklichkeit gehalten worden. Nur diesmal war die
Wirklichkeit nicht der Code, sondern **das Buch im Regal**.

| Fund | Was es behauptete | Was stimmte |
|---|---|---|
| `barcode_id` aus der Exemplarnummer | „Auf dem Etikett steht die Exemplarnummer" | Der Scanner liest eine EAN-13: aufgedruckt `105785`, gescannt `1057850039567`. Keines der 61.520 Exemplare wäre auffindbar gewesen — gemerkt hätte man es an der Theke. |
| Schülerausweis `[0395] 37` | Der Aufdruck ist der Scanwert | Gescannt kommt `B97601826457`, die Nummer des Kartenherstellers. Sie steht in keinem Stammdatenfeld, nur in Litteras `FremdLeserNummer`. |
| „10.002 Titel mit Autor (93 %)" | Autorenabdeckung | Mitgezählt waren Standortvermerke, die als Personen erfasst sind: `Buchbestand Bibliothek` allein auf 6.711 Titeln. Bei 7.131 Titeln stünde ein Regalvermerk in der Autorenangabe. Echt sind 9.029 (84 %). |
| Geburtsdaten aus Littera | Jahrgänge der Personen | Gos Jahrhundertgrenze liegt bei 69: 69 Lehrkräfte der Jahrgänge 1946–1968 kamen als **2046–2068** an. |
| Präfixlose Omnibox-Auflösung | Buch → Schüler → Suche | Lehrkräfte stehen in `benutzer`, nicht in `schueler`. Ein gescannter Lehrerausweis lief bis in die Volltextsuche. Die passende Abfrage gab es längst — sie hing allein hinter `L-` und ließ sich als rohes SQL am Pool nicht testen. |

**Merksatz dieses Tages:** Aufdruck, Datenbankspalte und Scanwert sind drei verschiedene
Dinge. Für jede Barcode-Quelle einmal real in einen Texteditor scannen, bevor irgendetwas
nach `barcode_id` geschrieben wird.

Dazu, aus demselben Lauf und nicht kleiner: **`npm ci` lief gegen das committete Lockfile
gar nicht durch** (`typescript@^7` gegen `typescript-eslint@8.65`, das `<6.1.0` verlangt).
Lokal war ESLint grün, weil `node_modules` älter war als das Lockfile — auf einem frischen
Klon oder in CI wäre es rot gewesen. Ein grünes Gate beweist nichts über das, was im Repo
steht, wenn die Umgebung abgedriftet ist.

Stand: 2026-08-04

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

### Nachtrag desselben Tages: die Liste selbst abgearbeitet

| Fund | Was es behauptete | Was stimmte |
|---|---|---|
| `schadensfall.spec.js` fiel unter Last um | Test prüft den Elternbrief | `popup.url()` wurde gelesen, bevor die Navigation übernommen hatte — allein grün, im vollen Lauf manchmal rot. Jetzt wird die Adresse abgewartet und die Route zusätzlich am Inhaltstyp geprüft. |
| `SendTemplateMail` | Vorlagen-Versand des Systems | kein einziger Aufrufer; die Vorlagen werden anderswo direkt geladen. Entfernt. |
| `inventur`-Backup-Modul samt Benachrichtigungs-Mail | zweites Backup-System mit Mail bei Änderungen | `NewAPIHandler` hat das Feld nie gesetzt — unerreichbar. Gesichert wird über `jobs.BackupJob`. Die `.env.example` lud dazu ein, `BACKUP_EMAIL_TO` zu setzen und auf Mails zu warten, die nie kommen konnten. Ersatzlos entfernt. |

Der dritte Fund kam aus der Prüfung des zweiten: Der Eintrag stand hier als
„bewusst so entschieden" — und die Begründung hielt der Nachfrage nicht stand. Auch
eine Notiz in dieser Liste ist eine Zusicherung, die jemand prüfen muss.

Stand: 2026-07-30
