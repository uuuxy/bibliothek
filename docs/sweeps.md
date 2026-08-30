# Sweeps — der Prüfvorrat über den Bestand

**Zweck:** Daniels Raster (siehe `invarianten.md`, Memory „Raster hat 11 Fragen") prüft
eine **Änderung**: Wenn ein Schreibpfad seine Form wechselt, werden die elf Fragen
gestellt. Es sieht nicht, was schon da ist. Diese Seite ist die zweite Achse — der
**Bestand**: bekannte Bugklassen, je mit einem Suchmuster, das über den ganzen Code
läuft, und einem Gate, das die Klasse danach nicht mehr hereinlässt.

Anlass (29.08.2026): Ein Jules-PR (#529) wollte per Test festschreiben, dass
`GetBookByID` jeden DB-Fehler „buch nicht gefunden" nennt. Der Fund kam nicht vom
Werkzeug, sondern vom Lesen des Codes daneben — Jules' Wert ist der Zeigefinger in
Ecken, die niemand anfasst. Genau das leistet ein Sweep systematisch: eine Klasse,
der ganze Bestand, eine Ratsche.

## Regeln

1. **Ein Sweep je Sitzung.** Zehn halbe Sweeps sind null Gates.
2. **Detektor zuerst rot sehen.** Ein Muster, das nichts findet, meldet ewig „alles gut"
   (Memory „Gate am Rückbau beweisen"). Jede Ratsche trägt eine Gegenprobe am Detektor.
3. **Jeder Treffer wird eingeordnet, nicht vertagt:** behoben, oder mit Begründung im
   Bestand der Ratsche eingefroren. Der Bestand ist eine Liste bewusster Ausnahmen.
4. **Grep ist Hypothese, AST ist Befund.** Der erste Grep zum Fehler-Kollaps fand 11
   Stellen, davon 1 echte; der AST-Detektor fand 26, davon 6 echte. Beide Zahlen ohne
   Lesen des Codes wären falsch gewesen.

## Register

| Bugklasse                     | Form                                                                                               | Gate                                                     | Stand                                                                                              |
| ----------------------------- | -------------------------------------------------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| **Fehler-Kollaps**            | `if err != nil { 404/401/403 }` ohne Einordnung — DB-Fehler wird zum Bedienfehler, Diagnose stirbt | `fehler_kollaps_test.go` (AST)                           | 29.08.2026: 6 behoben, 8 eingefroren (Token/CSRF = korrekt; externe Lookups = Produktfrage, s. u.) |
| NULL-Scan                     | nullbare Spalte in nicht-nullbaren Go-Typ → 500                                                    | nur echter PG-Test (`*_pg_test.go`)                      | Memory „NULL-Scan-Bugklasse"                                                                       |
| Upsert-Blanking               | leerer Eingabewert löscht Bestandswert                                                             | Paarungs-Gates je Import (LUSD, Littera, Schlanke Liste) | Memory „Upsert-Blanking", „LUSD: Klasse ohne COALESCE"                                             |
| Zwei Türen zum selben Zustand | generischer PATCH schreibt Spalten, für die ein Fach-Endpunkt Regeln hat                           | `audit_schreibtueren_test.go`                            | Memory „Zwei Türen"                                                                                |
| Tote Türen                    | Interface-Methode ohne produktiven Aufrufer                                                        | `tote_tueren_test.go`, deadcode-Baseline                 | Memory „Tote-Türen-Detektoren"                                                                     |
| Privilegiertes Request-Feld   | Feld hebt Regel auf, nur UI verbirgt es                                                            | Quelltext-Ratsche (70723d3f)                             | Memory                                                                                             |
| Unbegrenzte Listen            | Endpunkt ohne LIMIT                                                                                | Memory „Unbegrenzte Listen-Endpunkte"                    | **kein Gate** — Prüfvorrat                                                                         |
| Geschwister-Asymmetrie        | zwei gleichartige Pfade, einer kann etwas, der andere nicht (Anliegen-Notiz vs. Klassensatz)       | je Paar ein Paar-Gate (`api/monitor_pg_test.go`)         | 30.08.2026: Katalog↔Monitor (Sichtbarkeit ohne Anmeldung) auf EIN Prädikat gezogen (`repository.OeffentlichSichtbar`); offen: alle Paare „Lehrkraft löst aus → Theke schließt ab → Mail" |
| Verschluckte Ursache          | Fehler wird in eine generische Antwort gebogen, ohne Logzeile (Selbstanmeldung-401, CI 29.08.)     | —                                                        | b8daed0a loggt den einen Fall → 30.08.2026 Ursache gefunden: der Auth-Test-Harness legte `benutzer` mit einer ABGESCHRIEBENEN DDL an (ohne Migration 086) — je nach Paketreihenfolge auf der frischen CI-DB rot; Harness lädt jetzt schema.sql |
| Transaktionsgrenzen           | Mehrschritt-Schreibpfad ohne Tx                                                                    | Sweep 19.08.                                             | Memory „Sweep Transaktionsgrenzen"                                                                 |
| IDOR / Zeit / Vokabular       | 3 Sweeps 19.08., je Gate                                                                           | —                                                        | Memory „Sweep IDOR/Zeit/Vokabular"                                                                 |

## Offene Produktfrage aus dem Kollaps-Sweep

Drei externe Lookups (DNB/OpenLibrary: `api/isbn_handler.go`, `inventur/isbn_suche.go`,
`inventur/cover_aktualisierung.go`) antworten bei Netzausfall „nicht gefunden". Für die
Theke ist das derselbe Bildschirm wie ein echter Nicht-Treffer. Soll der Betreiber den
Unterschied sehen (502 „Katalogdienst nicht erreichbar" statt 404)? Bis zur Entscheidung
im Bestand der Ratsche eingefroren.
