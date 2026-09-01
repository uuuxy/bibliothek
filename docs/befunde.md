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

Leer (Stand 01.09. abends): Die drei Posten vom 01.09. sind abgearbeitet —
Eigen-Lock-Harnesse auf `internal/pgtest` (6caa3d4b), Barcode-Snapshot beim
Titel-Löschen an beiden Türen samt Paar-Gates (9889c30c), PII-Antwort-Gate um
die lesenden POSTs erweitert (`TestPIIAntwortenHaltenIhreStufe_LesendePosts`).
Der erste Lauf des POST-Gates fand direkt einen A-Fund: `POST /api/action/batch`
wies durch `Validate.Struct` auf einem Slice JEDEN Offline-Sync als 400 ab —
sofort gefixt, Rot am Live-Pfad gesehen.

Davor: Leer seit 01.09. früh (Abgänger-in-Aktivliste behoben, Gate
`aktivliste-ohne-abgaenger.spec.js`). Die Abarbeitung vom 31.08.2026 (7 Posten,
je ein Commit) steht in der Git-Historie dieser Datei; dabei zusätzlich gefunden
und behoben: der Etiketten-Druck-E2E war seit Migration 071 still rot (Seed
schrieb den alten Magie-Text statt `bestellstatus`).

## Offen — Entscheidung nötig (Peter)

Leer (Stand 01.09. abends). Die tote Vorlage `BESTELLUNG_EINGETROFFEN` ist
ENTSCHIEDEN (Peter, 01.09.): keine Mail — Klassenleitungen wären überlastet,
Eltern-Mail wäre der dünnste §-15-SchDSV-Fall, und Schüler merken ohnehin an
der Theke vor. Stattdessen zeigt das Terminal beim Ausweis-Scan durch die
Mitarbeiterin den **Abholfach-Hinweis** (OmniboxService, PG-Test rot gesehen);
die Vorlage ist mit Migration 092 ausgetragen.

Die drei Posten vom 31.08. sind nach Peters Freigabe alle umgesetzt:
`sonar.projectVersion` (Scan-Skript), die Tresen-Auskunft (zweckgebundener
Leseweg in `audit_log.details`, eigenes Recht `audit_details`) und das
PII-Antwort-Gate (`api/pii_antwort_gate_pg_test.go` — die Stufen der GET-Routen
sind jetzt gemessen, nicht nur behauptet).

## Beobachten (nichts zu tun)

| Fund | Warum nur beobachten |
| ---- | -------------------- |
| `golang.org/x/crypto/openpgp` gilt als unwartbar (`GO-2026-5932`) | Kein Fix verfügbar (`Fixed in: N/A`), transitiv, **kein** Aufrufer im eigenen Code (`govulncheck`, zuletzt 31.08. bestätigt). Dependabot meldet sich, falls sich das ändert. |
| Designer-Restlücke: Browser-Schließen binnen 800 ms verliert die letzte Auto-Save-Änderung | `onDestroy` schickt seit `37abcbe0` sofort; `sendBeacon` bewusst nicht gebaut (Randlage). |
| 082-Dedupe der Vormerkungen verlor den neueren `abholbereit`-Eintrag | Auf Prod gelaufen, nicht rückholbar; nur relevant, falls je eine weitere **gewachsene** DB migriert wird. |

## Kategorie C — bewusst nicht ohne Anlass

- Aus dem Nie-verdrahtet-Sweep (01.09., Agenten-Durchgang, je geprüft und bewusst
  geparkt): `inventur_sessions.gestartet_von` wird geschrieben und 4× gescannt,
  aber nie angezeigt (die Inventur-UI könnte den Starter nennen — Produktfrage);
  `abgaenger_jahr` steht in der Aktiv-Listen-Antwort, obwohl die Liste Abgänger
  ausschließt und kein Listen-Konsument es liest (Profil nutzt es); die
  Geräte-Torso-Reste (`ActionEvent.GeraetID` nie gesetzt, Geräte-Aktionen
  broadcasten gar nicht, Kiosk-Pfad liefert Null-Zeitstempel) warten auf den
  Geräte-Ausbau; der 501-Zweig `/api/admin/books/import` zeigt neben dem
  funktionierenden Import ins Leere.
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

Stand: 2026-09-01
