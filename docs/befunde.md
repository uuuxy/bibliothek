# Befund-Register

**Zweck:** Ein Ort für Dinge, die beim Arbeiten auffallen, aber nicht sofort angefasst
werden. Ohne diese Liste gibt es nur zwei Umgangsformen mit einem Fund — sofort
reparieren oder vergessen — und beide sind falsch.

Ergänzt [`invarianten.md`](invarianten.md): Dort steht, was immer wahr sein _muss_.
Hier steht, was aufgefallen ist und noch nicht entschieden oder erledigt wurde.

**Historie:** Die abgearbeiteten Durchgänge (Raster-Durchgänge, Sweeps, Audits
Juli–August 2026) standen bis zum 31.08.2026 vollständig in dieser Datei; sie sind
bewusst entfernt und in der Git-Historie dieser Datei erhalten
(`git log -p docs/befunde.md`, Stand vor der Kürzung: `2e09ec14`).

---

## Die Einordnung

Vor jedem Fund steht dieselbe Frage — **nicht** „ist das hässlich?", sondern
**„kann das still jemandem schaden?"**:

|       | Kategorie                                                                                                                                                                                            | Umgang                                                               |
| ----- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| **A** | Kann **stillschweigend** ein falsches Ergebnis für einen echten Menschen erzeugen: doppelte Mahnung an Eltern, Daten beim falschen Empfänger, verlorene Eingabe, ein Gate, das nicht rot werden kann | **Sofort**, eigener Commit, eigener Test, eigene Deploy-Entscheidung |
| **B** | Fehler, der sich **laut** meldet, oder Unordnung ohne Wirkung nach außen: totes Codestück, wackeliger Test, Doppelung                                                                                | **Hier notieren**, gebündelt abarbeiten                              |
| **C** | „Wenn ich schon mal hier bin" — Umbenennungen, Stilfragen, Refactorings ohne Anlass                                                                                                                  | Nur mit Anlass und Zeit                                              |

Zwei Regeln dazu:

1. **Ein Fund = ein Commit.** Kein Anhängen verwandter Aufräumarbeiten. Was beim
   Reparieren zusätzlich auffällt, kommt in diese Liste, nicht in denselben Commit.
2. **Kategorie A wird belegt, nicht behauptet.** Ein Fund dieser Klasse braucht einen
   Test, der mit dem alten Code rot wird. Ohne diese Gegenprobe ist unklar, ob
   überhaupt etwas repariert wurde.

---

## Offen — abarbeitbar

Leer (Stand 01.09. früh; der Abgänger-in-Aktivliste-Fund von der Duc-Bauer-Spur
ist am Live-Pfad rot nachgestellt und behoben — Gate
`aktivliste-ohne-abgaenger.spec.js` prüft beide Richtungen). Die Abarbeitung vom 31.08.2026 (7 Posten, je ein Commit) steht in der
Git-Historie dieser Datei; dabei zusätzlich gefunden und behoben: der
Etiketten-Druck-E2E war seit Migration 071 still rot (Seed schrieb den alten
Magie-Text statt `bestellstatus`).

## Offen — Entscheidung nötig (Peter)

| Fund | Worum es geht |
| ---- | ------------- |
| Audit-`details` sind über die Anwendung nicht lesbar (bewusste Stufe-0-Minimierung) | Für den einen Fall „Buch liegt auf dem Tresen, Barcode unbekannt: war er mal vergeben?" fehlt ein Weg. Vorschlag: zweckgebundener Endpunkt mit eigener PII-Einstufung und eigenem Recht — Betreiber-Entscheidung, keine Fehlerbehebung. |
| PII-Matrix ist eine Zusage, die nur ein Dokument behauptet | Das PII-Gate prüft Route/Recht/Zeilen-Existenz, nicht den Antwortinhalt. Ein Gate, das Antwortfelder gegen die Matrix hält, wäre der nächste methodische Schritt — eigener Anlauf. |

## Beobachten (nichts zu tun)

| Fund | Warum nur beobachten |
| ---- | -------------------- |
| `golang.org/x/crypto/openpgp` gilt als unwartbar (`GO-2026-5932`) | Kein Fix verfügbar (`Fixed in: N/A`), transitiv, **kein** Aufrufer im eigenen Code (`govulncheck`, zuletzt 31.08. bestätigt). Dependabot meldet sich, falls sich das ändert. |
| Designer-Restlücke: Browser-Schließen binnen 800 ms verliert die letzte Auto-Save-Änderung | `onDestroy` schickt seit `37abcbe0` sofort; `sendBeacon` bewusst nicht gebaut (Randlage). |
| 082-Dedupe der Vormerkungen verlor den neueren `abholbereit`-Eintrag | Auf Prod gelaufen, nicht rückholbar; nur relevant, falls je eine weitere **gewachsene** DB migriert wird. |

## Kategorie C — bewusst nicht ohne Anlass

- Reiterleisten: drei Höhen (30/32/34 px) in den handgebauten Bestands-Leisten —
  vereinheitlichen beim nächsten fachlichen Anfassen (der Umbruch ist seit
  `c35840d4` gelöst, Typografie-Fahrplan Schritt 6 damit halb offen).
- Etikettenraster an zwei Stellen (maßgeblich `api/label_formats.go`,
  Frontend-Kopie `src/lib/etikettformate.js`): Am 31.08.2026 geprüft und
  ENTSCHIEDEN geparkt. Ein Server-Umbau bräuchte einen neuen Endpunkt (existiert
  nicht) plus async-Ladezustand in drei Verbrauchern des täglich benutzten
  Druck-Bildschirms — neuer Ausfallmodus gegen null Laufzeit-Risiko heute. Die
  Drift hält `etikettformate-konsistenz.test.js` mechanisch (Spalten und Zeilen
  einzeln gegen die Go-Datei). Wieder anfassen nur, wenn ein vierter Verbraucher
  oder ein konfigurierbares Format dazukommt.
- 20× `go:S3776` (Cognitive Complexity, alle Backend): Zählweise rechnet den
  Handler-Closure als Ebene; Aufsplitten nur für die Zahl macht nichts lesbarer.
  Lohnend allenfalls `OverrideDueDateHandler` (30), `behandleAbgaenger` (23).
- `javascript:S6551` (`escapeHtml.js`) und `javascript:S8783`
  (`schuelerprofil-sperre.spec.js`): begründete Dauer-Ausnahmen, keine offenen Punkte.
- Tabellen-Inline-Felder (Rückgabedatum, Exemplar-Barcode) sind mit 36 px höher als
  ihre Zeile — `size="sm"`-Variante von `Feld` erst bei Bedienbefund.
- `LabelHeight >= 30`-Schwelle steht 6× in zwei Dateien.
- LUSD-Namensschlüssel nur `lower+trim` (Umlaut-/Bindestrich-Varianten gelten als
  verschiedene Menschen → „mehrdeutig"/neu) — sicher, aber nicht klug.
- Paritätstest vergleicht keine COMMENTs/Seeds (nur Kosmetik; Struktur ist gedeckt).
- Jules-Erbe: sieben Testdateien > 200 Zeilen; Export-CSV-„breaks stream"-Test schwach.
- Go `dupl` 8 Klonpaare (print.go, lookups.go, mahnwesen.go), Frontend `jscpd` 0,41 %.

## Außerhalb dieses Registers (Betrieb, liegt bei Peter)

Die gewichtigen offenen Posten sind keine Code-Befunde und leben in den
Betriebs-Dokumenten bzw. der Selbstprüfung: **S3-Auslagerung der Backups** ·
**frisches Littera-Backup** (littera_sav.mdb ist ein 2010er-Stand) ·
**`SELBSTANMELDUNG_DOMAIN`** auf dem Schulserver setzen · Betriebsbereitschafts-Tab
des Schulservers durchsehen · Datenwert Schulname/„Neuer Text" auf Live korrigieren.

Postgres 18 ist KOMPLETT durch (Repo 31.08., Prod laut Peter längst umgezogen).
Merkposten für lokale PG-Testläufe: `pg_dump` 18.6 liegt in
`/opt/homebrew/opt/libpq/bin` und muss vor den PATH (`brew`-Standard ist 16.15) —
sonst scheitern die zwei Backup-Proben-Tests scheinbar, wie am 31.08. passiert.

Stand: 2026-08-31
