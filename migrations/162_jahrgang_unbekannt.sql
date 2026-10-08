-- =============================================================================
-- Migration 162: Jahrgang am Titel — „unbekannt" ist eine eigene Angabe (docs/OFFEN.md 5.5)
-- =============================================================================
-- jahrgang_von und jahrgang_bis trugen an jedem Titel ohne Angabe die Vorgabe 5 und 10. Ein
-- Titel für die Jahrgänge 5 bis 10 war damit von einem Titel ohne Angabe nicht zu
-- unterscheiden: Die Inventur nach Klasse traf für 5 bis 10 den ganzen Bestand und für 11 bis
-- 13 nichts, der Filter „Jahrgang" der Schulbücher im Portal jedes Lernmittel.
--
-- NULL in beiden Spalten heißt „unbekannt"; die Vorgabe entfällt. Beide Spalten sind gesetzt
-- oder keine, die Werte liegen zwischen 1 und 13, „von" nicht über „bis". Die Regel nennt IS
-- NOT NULL ausdrücklich: Ein Vergleich mit NULL ergibt NULL, und eine CHECK-Bedingung, die NULL
-- ergibt, gilt als erfüllt — eine halbe Spanne ginge sonst durch. Was bisher 5 bis 10 trug,
-- wird „unbekannt". Ein Mehrjahresband behält seine Spanne: Den Schalter hat jemand an der
-- Maske neben der angezeigten Spanne gesetzt, und chk_mehrjahresband_spanne verlangt sie.
--
-- Gemessen am Testserver am 08.10.2026 (Stand 293a9a91, lesend): 13.062 Titel, davon 13.056
-- mit 5 bis 10 (12.481 ohne, 575 mit Lernmittel-Kennung) und sechs mit eigener Spanne (5–5,
-- 5–6, zweimal 6–6, 8–8, 10–10); kein Mehrjahresband; keine Zeile außerhalb von 1 bis 13 oder
-- mit „von" über „bis". Die Änderung lässt sich zurücknehmen: Jede Zeile, die hier NULL wird,
-- trug 5 und 10. Der Trigger auf aktualisiert_am läuft mit, die betroffenen Titel gelten als
-- an diesem Tag geändert.
--
-- Die Migration läuft je Datenbank einmal (schema_migrations). Ein zweiter Lauf von Hand träfe
-- auch eine danach eingetragene Spanne 5 bis 10.
-- =============================================================================

ALTER TABLE buecher_titel
    ALTER COLUMN jahrgang_von DROP DEFAULT,
    ALTER COLUMN jahrgang_von DROP NOT NULL,
    ALTER COLUMN jahrgang_bis DROP DEFAULT,
    ALTER COLUMN jahrgang_bis DROP NOT NULL;

UPDATE buecher_titel
   SET jahrgang_von = NULL, jahrgang_bis = NULL
 WHERE jahrgang_von = 5 AND jahrgang_bis = 10 AND NOT mehrjahresband;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_jahrgang_spanne') THEN
        ALTER TABLE buecher_titel ADD CONSTRAINT chk_jahrgang_spanne
            CHECK ((jahrgang_von IS NULL AND jahrgang_bis IS NULL)
                   OR (jahrgang_von IS NOT NULL AND jahrgang_bis IS NOT NULL
                       AND jahrgang_von BETWEEN 1 AND 13 AND jahrgang_bis BETWEEN 1 AND 13
                       AND jahrgang_von <= jahrgang_bis));
    END IF;
END $$;

COMMENT ON COLUMN buecher_titel.jahrgang_von IS
    'Erster Jahrgang, für den der Titel gilt; NULL zusammen mit jahrgang_bis = unbekannt (Migration 162).';
COMMENT ON COLUMN buecher_titel.jahrgang_bis IS
    'Letzter Jahrgang, für den der Titel gilt; NULL zusammen mit jahrgang_von = unbekannt (Migration 162).';
