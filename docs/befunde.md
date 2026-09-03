# Befund-Register

**Zweck:** Ein Ort für Dinge, die beim Arbeiten auffallen, aber nicht sofort angefasst
werden. Ohne diese Liste gibt es nur zwei Umgangsformen mit einem Fund — sofort
reparieren oder vergessen — und beide sind falsch.

Ergänzt [`invarianten.md`](invarianten.md): Dort steht, was immer wahr sein _muss_.
Hier steht, was aufgefallen ist und noch nicht entschieden oder erledigt wurde.

**Historie:** Die abgearbeiteten Durchgänge (Raster-Durchgänge, Sweeps, Audits
Juli–August 2026) standen bis zum 31.08.2026 vollständig in dieser Datei; sie sind
bewusst entfernt und in der Git-Historie dieser Datei erhalten
(`git log -p docs/befunde.md`; Stände vor den Kürzungen: `2e09ec14` (31.08.),
`47a09f70` (01.09. — Posten-Abarbeitung, v1.8.1, Fahrplan-Auflösung)).

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

Leer (Stand 01.09.2026 abends). Abgearbeitetes steht in der Git-Historie dieser
Datei (`git log -p docs/befunde.md`), nicht hier.

## Offen — Entscheidung nötig (Peter)

Rest aus dem Rasterdurchgang 02.09.2026 (LUSD-Umbenennung, 0aa07f57). Reihenfolge = Abarbeitung;
Peter überlegt beide (03.09.).

1. **Handanlagen als Paar-Kandidaten.** Ein von Hand angelegter, nie LUSD-bestätigter Schüler,
   den die LUSD anders schreibt („Anna Müller" ↔ „Anna Mueller"), wird nicht als Paar
   vorgeschlagen, sondern Neuanlage + „nicht im Export". Rückweg heute: Zusammenführen über
   die Akte. Bauen hieße: `NichtImExport`-Zeilen als Kandidaten (`WarAbgaenger=false`) — und
   die Rubrik „nicht im Export" bedeutet dann etwas anderes.
2. **Karenz ab Rückgabe statt ab Abgang.** `abgaenger_seit` stempelt beim Abgang, auch mit
   offenen Büchern; wer erst nach Ablauf der Karenz zurückgibt, wird in der Folgenacht
   anonymisiert — das Reparaturfenster fehlt dieser Gruppe. Alternative: Uhr bei Rückgabe des
   letzten Vorgangs neu starten (dann PG-Test dieser Konstellation).

## Beobachten (nichts zu tun)

| Fund | Warum nur beobachten |
| ---- | -------------------- |
| `golang.org/x/crypto/openpgp` gilt als unwartbar (`GO-2026-5932`) | Kein Fix verfügbar (`Fixed in: N/A`), transitiv, **kein** Aufrufer im eigenen Code (`govulncheck`, zuletzt 31.08. bestätigt). Dependabot meldet sich, falls sich das ändert. |
| Designer-Restlücke: Browser-Schließen binnen 800 ms verliert die letzte Auto-Save-Änderung | `onDestroy` schickt seit `37abcbe0` sofort; `sendBeacon` bewusst nicht gebaut (Randlage). |
| 082-Dedupe der Vormerkungen verlor den neueren `abholbereit`-Eintrag | Auf Prod gelaufen, nicht rückholbar; nur relevant, falls je eine weitere **gewachsene** DB migriert wird. |
| Barcode des beim Zusammenführen aufgelösten Datensatzes wird wieder vergeben (MAX+1) | Betriebsregel statt Bau (Peter, 03.09.): zweite Karte einziehen — steht im HANDBUCH. Der Ausweis des Kindes selbst ändert sich nie. |
| Abgänger in der Karenz stehen in keiner Liste (Aktivliste blendet Abgänger aus, Abgänger-Reiter nur mit offenen Büchern) | Peter, 03.09.: nicht schlimm — sie gehen ab und haben nichts offen. Erreichbar per Ausweis-Scan und Kandidatensuche. |

## Kategorie C — bewusst nicht ohne Anlass

- Aus dem Nie-verdrahtet-Sweep (01.09., Agenten-Durchgang, je geprüft und bewusst
  geparkt): `inventur_sessions.gestartet_von` wird geschrieben und 4× gescannt,
  aber nie angezeigt (die Inventur-UI könnte den Starter nennen — Produktfrage);
  `abgaenger_jahr` steht in der Aktiv-Listen-Antwort, obwohl die Liste Abgänger
  ausschließt und kein Listen-Konsument es liest (Profil nutzt es); die
  Geräte-Torso-Reste (`ActionEvent.GeraetID` nie gesetzt, Geräte-Aktionen
  broadcasten gar nicht, Kiosk-Pfad liefert Null-Zeitstempel) warten auf den
  Geräte-Ausbau.
- Reiterleisten: drei Höhen (30/32/34 px) in den handgebauten Bestands-Leisten —
  vereinheitlichen beim nächsten fachlichen Anfassen (der Umbruch ist seit
  `c35840d4` gelöst).
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

Seit dem 01.09.2026 ist dieser Abschnitt die EINE Liste der offenen Betriebs-Punkte —
der frühere `master_fahrplan.md` ist aufgelöst (seine Code-Punkte waren erledigt, der
Rest stand doppelt; Littera-Details jetzt in
[littera_schema_befund.md](littera_schema_befund.md), Historie in `git log`):

- **Frisches Littera-Backup** — littera_sav.mdb ist ein 2010er-Stand; Anforderungen
  (FremdLeserNummer/FremdBarcode) in [littera_schema_befund.md](littera_schema_befund.md).
- **Drei Sekretariats-Abnahmen** à ~10 Minuten (LUSD-Import, Versetzung ⏰ vor dem
  Schuljahreswechsel, Klassensatz-Erledigen) — Ablauf in
  [abnahme_checkliste.md](abnahme_checkliste.md).
- **Zielumgebung** (die Seite System → Betriebsbereitschaft zeigt den Ist-Zustand):
  Prod-Secrets mit `ENFORCE_PROD_SECRETS=true` (ohne `BACKUP_ENCRYPTION_KEY` läuft
  KEIN Backup) · Schul-IMAP/SMTP-Zugangsdaten · **`SELBSTANMELDUNG_DOMAIN`** setzen ·
  einmalige manuelle Restore-Probe am echten Ziel ([DEPLOYMENT.md](DEPLOYMENT.md) §6) ·
  `SENTRY_DSN` leer lassen (A6 in
  [datenschutz_offene_punkte.md](datenschutz_offene_punkte.md)) ·
  **S3-Auslagerung der Backups**.
- **GitHub**: PR-Pflicht abschaffen (Solo-Entscheidung 30.07.), „Block force pushes"
  und „Restrict deletions" anlassen.
- Datenwert Schulname/„Neuer Text" auf Live korrigieren.

**Parkdeck** (bewusste Nicht-Entscheidungen, nur mit Anlass wieder anfassen):
Integer-Cent-Refactor (float64/NUMERIC) · Bundle-Splitting (720-kB-Chunk) ·
TypeScript-Migration (null TS-Dateien) · Verschmelzung `inventur/` ins Haupt-API ·
`cmd/migrate` (MySQL) löschen — hat keine Datenquelle mehr, aber seine PG-Tests
sichern mit `internal/uebernahme` geteilten Code · Zukunftsideen API-Versionierung
(`/api/v1`) und Mandantenfähigkeit (RLS).

Stand: 2026-09-01
