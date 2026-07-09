-- ==============================================================================
-- TABULA RASA: Datanbank-Bereinigung für den Echtbetrieb (Produktion)
-- ==============================================================================
-- WARNUNG: Dieses Skript löscht ALLE Bewegungsdaten (Bücher, Ausleihen, Schüler, Logs)
-- unwiderruflich aus der Datenbank! Es darf nur EINMALIG vor dem Live-Gang 
-- ausgeführt werden, um die Testdaten zu entfernen.
--
-- SICHERHEITSHINWEIS: 
-- Die Tabellen 'benutzer', 'benutzer_rollen' (Admin-Accounts) sowie 
-- 'system_einstellungen' und 'subjects' bleiben zwingend erhalten.
-- ==============================================================================

BEGIN;

-- CASCADE löst automatisch alle referenzierten Foreign-Keys auf.
-- RESTART IDENTITY setzt eventuelle SERIAL/Auto-Increment Counter auf 1 zurück.
TRUNCATE TABLE 
    ausleihen,
    schadensfaelle,
    vormerkungen,
    klassensatz_reservierungen,
    buecher_exemplare,
    buecher_titel,
    class_books,
    schueler,
    audit_log,
    geraete,
    lieferanten
RESTART IDENTITY CASCADE;

-- Optional: Falls LUSD-Import-Zwischentabellen existieren, könnten diese hier
-- ebenfalls geleert werden (z.B. 'lusd_schueler_raw'). Aktuell geschieht das im RAM.

COMMIT;

-- Ausgabe zur Bestätigung
DO $$
BEGIN
    RAISE NOTICE 'Tabula Rasa erfolgreich: Alle Bewegungsdaten wurden gelöscht. Admin-Accounts sind erhalten geblieben.';
END $$;
