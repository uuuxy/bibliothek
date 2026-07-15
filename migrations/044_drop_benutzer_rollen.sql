-- =============================================================================
-- Migration 044: Legacy-Tabelle benutzer_rollen entfernen (G5-Abschluss)
-- =============================================================================
-- benutzer_rollen war als Nachfolger des ENUMs benutzer.rolle gedacht, die
-- Umstellung wurde aber nie zu Ende geführt: befüllt wurde sie nur einmalig beim
-- Bootstrap (db/seed.go), gelesen hat sie zuletzt niemand mehr. Genau diese
-- Halbfertigkeit war die Ursache des Handapparat-Bugs — nach dem Bootstrap
-- angelegte Lehrkräfte fehlten in der Tabelle, der INNER JOIN lieferte nichts.
--
-- Seither gilt eindeutig:
--   * WER welche Rolle hat  -> benutzer.rolle (ENUM, kleingeschrieben)
--   * WAS eine Rolle darf   -> role_permissions (GROSS; Middleware mappt per UPPER())
--
-- Kein Datenverlust: die Tabelle enthielt ausschliesslich UPPER(benutzer.rolle),
-- also eine Projektion der bereits vorhandenen Wahrheit.
--
-- WICHTIG: db/seed.go musste im selben Schritt bereinigt werden. Die Migrationen
-- laufen VOR InitPermissions (main.go); ein dort verbliebenes
-- "CREATE TABLE IF NOT EXISTS benutzer_rollen" hätte die Tabelle unmittelbar nach
-- diesem DROP als leere Ruine neu angelegt.
-- =============================================================================

-- Diagnose vor dem irreversiblen DROP: Sollte jemand die Tabelle je von Hand
-- abweichend gepflegt haben, geht diese Information jetzt verloren. Wirkung hatte
-- sie nie (kein Leser), aber sie soll nicht stillschweigend verschwinden.
DO $$
DECLARE
    abweichungen INT;
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'benutzer_rollen') THEN
        SELECT count(*) INTO abweichungen
        FROM benutzer_rollen br
        JOIN benutzer b ON b.id = br.benutzer_id
        WHERE br.rolle IS DISTINCT FROM UPPER(b.rolle::text);

        IF abweichungen > 0 THEN
            RAISE NOTICE 'Migration 044: % Eintrag/Einträge in benutzer_rollen wichen von benutzer.rolle ab und werden verworfen (waren wirkungslos, da kein Leser).', abweichungen;
        END IF;
    END IF;
END $$;

DROP TABLE IF EXISTS benutzer_rollen;
