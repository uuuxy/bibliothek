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
5. **Zwillings-Pflicht beim Fix.** Jeder Fix-Commit beantwortet die Frage „Wo ist der
   Zwilling?" — der Geschwister-Pfad mit derselben Form (Mahnlauf↔Abgänger-Versand,
   Ausbuchen↔Aussondern, Anliegen↔Klassensatz). Die Antwort steht in der
   Commit-Botschaft: der geprüfte Zwilling und sein Befund, oder „hat keinen".
   Beleg für den Wert: Am 31.08.2026 kamen zwei der 18 Funde genau über diese Frage
   ans Licht, nachträglich gestellt (siehe Geschwister-Asymmetrie im Register).

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
| Geschwister-Asymmetrie        | zwei gleichartige Pfade, einer kann etwas, der andere nicht (Anliegen-Notiz vs. Klassensatz)       | je Paar ein Paar-Gate (`api/monitor_pg_test.go`)         | 30.08.: Katalog↔Monitor auf EIN Prädikat (`repository.OeffentlichSichtbar`). 31.08.: Bereit-Mail-Notiz bekam den Rückweg des Anliegens (089); Abgänger-Versand die Drei-Ausgänge-Semantik des Mahnlaufs; Aussondern die Ausleih-Prüfung des Ausbuchens; Omnibox die Sequenznummer des orderStore |
| Doppelte Wahrheitsquelle      | dieselbe Regel zweimal formuliert (Job↔Wächter, Go↔SQL, Test↔Code)                                 | `jobs/loeschpraedikat_ratsche_test.go`; PG-Paarungstest `pkg/lmf`                                          | 31.08.2026: Anliegen-Löschjob teilt jetzt `PredikatAnliegen`; `lmf.SQLBedingung` trimmt wie `IstSchulbuch` (Lernmittel stand sonst im öffentlichen Katalog); drei Tests leiten das Prädikat ab statt es abzuschreiben |
| Phantom-Erfolg                | Schreibpfad ohne `RowsAffected`-Prüfung: 0 Zeilen = „success" (+ Audit-Eintrag über eine Nicht-Änderung) | `phantom_erfolg_test.go` (AST: verworfene CommandTags, Bestand je Datei:Funktion) + PG-Tests je Pfad                    | 31.08.2026: vormittags DeleteCopy/DecommissionCopy/UpdateCopyStatus; abends Bestands-Sweep über alle 163 Exec-Stellen — 10 weitere Pfade gefixt (Mail-Vorlage, UpdateUser, DeleteUser, DeleteTitle, Vormerkung, Schadensnotiz, Barcode, Inventur-Abort, Omnibox-Reaktivierung, Klassen-Mapping) + 1 Grenzfall (Mail-Bootstrap-Log); 73 Funktionen begründet eingefroren, beide Ratschen-Richtungen rot gesehen |
| Verschluckte Ursache          | Fehler wird in eine generische Antwort gebogen, ohne Logzeile (Selbstanmeldung-401, CI 29.08.)     | —                                                        | b8daed0a loggt den einen Fall → 30.08.2026 Ursache gefunden: der Auth-Test-Harness legte `benutzer` mit einer ABGESCHRIEBENEN DDL an (ohne Migration 086) — je nach Paketreihenfolge auf der frischen CI-DB rot; Harness lädt jetzt schema.sql |
| Transaktionsgrenzen           | Mehrschritt-Schreibpfad ohne Tx                                                                    | Sweep 19.08.                                             | Memory „Sweep Transaktionsgrenzen"                                                                 |
| IDOR / Zeit / Vokabular       | 3 Sweeps 19.08., je Gate                                                                           | —                                                        | Memory „Sweep IDOR/Zeit/Vokabular"                                                                 |

## Produktfrage aus dem Kollaps-Sweep — entschieden 31.08.2026

Drei externe Lookups (DNB/Google/OpenLibrary: `api/isbn_handler.go`, `inventur/isbn_suche.go`,
`inventur/cover_aktualisierung.go`) antworteten bei Netzausfall „nicht gefunden" — derselbe
Bildschirm wie ein echter Nicht-Treffer; bei einer WLAN-Störung katalogisierte die Theke
Bücher von Hand, die längst in der DNB stehen. Entscheidung (Peter): **502 „Katalogdienste
nicht erreichbar"**, 404 nur wenn mindestens eine Quelle geantwortet hat. Umsetzung:
`inventur.ErrKatalogdiensteNichtErreichbar` (Transportfehler je Quelle als Sentinel in
`holeInhalt`), `errors.Is` in allen drei Handlern; die drei Ratschen-Ausnahmen sind
ausgetragen. Gate: `inventur/lookup_ausfall_test.go` (kaputtes Netz → 502, erreichbar ohne
Treffer → 404) — am alten Stand rot gesehen.
