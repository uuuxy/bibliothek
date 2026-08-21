-- =============================================================================
-- Migration 083: den toten Append-Only-Anspruch auf audit_log auflösen
-- =============================================================================
-- Migration 003_audit_log_append_only.sql legte einen Trigger an, der JEDEN UPDATE
-- und DELETE auf audit_log mit einer Exception blockiert („Append-Only aus
-- DSGVO/Revisions-Gründen"). Dieser Trigger ist toter, gefährlicher Ballast:
--
--  * Er existiert real NIRGENDS: schema.sql (die Baseline jeder Installation) enthält
--    ihn nicht, und auf der Produktion ist er nicht vorhanden (pg_trigger auf audit_log
--    ist leer) — 003 wurde dort nur als „angewendet" geseedet, nie wirklich ausgeführt.
--  * Er steht im DIREKTEN Widerspruch zu einer gesetzlichen Pflicht: Die DSGVO-
--    Anonymisierung (jobs/cron_dsgvo.go) UND der Hard-Delete-Pfad
--    (repository/audit_users.go) MÜSSEN audit_log.details ändern, um den Personenbezug
--    zu tilgen (Art. 17). Würde der Trigger je aktiv — etwa weil jemand die Migrationen
--    ohne schema.sql-Baseline durchlaufen lässt —, schlüge genau die DSGVO-Löschung
--    still fehl. Der „Schutz" bräche also ausgerechnet die Löschpflicht.
--
-- Append-Only und Löschpflicht sind hier fundamental unvereinbar; für dieses System
-- gewinnt die Löschpflicht. Der reale Nutzen des Triggers wäre ohnehin gering — er
-- hilft nur gegen jemanden mit direktem Datenbankzugriff, der damit ohnehin alles
-- erreicht. Append-Only bleibt eine KONVENTION des Codes (keine UPDATE/DELETE auf
-- audit_log außer der DSGVO-Tilgung), nicht eine DB-erzwungene Regel.
--
-- Idempotent: Wo der Trigger nie existierte (frisch, Prod), ist dies ein No-op.

DROP TRIGGER IF EXISTS audit_log_append_only_trigger ON audit_log;
DROP FUNCTION IF EXISTS prevent_audit_log_modification();
