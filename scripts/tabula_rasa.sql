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

-- Die Bestätigung MISST, statt zu behaupten: Sie läuft noch in der Transaktion und zählt
-- nach. Bis zum 03.09.2026 stand sie hinter dem COMMIT und außerhalb jeder Transaktion —
-- bricht eine der TRUNCATE-Anweisungen ab, meldet psql für das COMMIT ein ROLLBACK, und
-- der Satz „Tabula Rasa erfolgreich" wurde trotzdem gedruckt. Ein Erfolgssatz, der nichts
-- misst, ist schlimmer als keiner.
DO $$
DECLARE
    rest_buecher  bigint;
    rest_schueler bigint;
BEGIN
    SELECT count(*) INTO rest_buecher FROM buecher_titel;
    SELECT count(*) INTO rest_schueler FROM schueler;
    IF rest_buecher > 0 OR rest_schueler > 0 THEN
        RAISE EXCEPTION 'Tabula Rasa unvollständig: noch % Titel und % Schüler vorhanden — nichts wird geschrieben.',
            rest_buecher, rest_schueler;
    END IF;
    RAISE NOTICE 'Tabula Rasa: Bewegungsdaten geleert, Admin-Accounts erhalten. COMMIT folgt.';
END $$;

COMMIT;
