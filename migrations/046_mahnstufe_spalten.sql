-- =============================================================================
-- Migration 046: Mahnstufen-Spalten auf ausleihen
-- =============================================================================
-- Der Bulk-Mahnlauf (api/mahnwesen_bulk.go) schreibt seit jeher
--   UPDATE ausleihen SET mahnstufe = mahnstufe + 1, letztes_mahndatum = ...
-- die Spalten waren aber weder in schema.sql noch in einer Migration definiert
-- (Schema-Drift). Auf einer frisch aus schema.sql/Migrationen gebauten DB brach
-- der Bulk-Druck deshalb mit „column \"mahnstufe\" does not exist" ab. Diese
-- Migration schließt die Lücke — idempotent (IF NOT EXISTS), damit sie auch auf
-- einer Prod-DB, die die Spalten evtl. schon per Alt-Schema besitzt, ein No-op ist.
--
--   * mahnstufe:         0 = noch nie gemahnt, +1 pro Mahnlauf.
--   * letztes_mahndatum: Zeitpunkt der letzten Mahnung (NULL = noch keine).
-- =============================================================================

ALTER TABLE ausleihen ADD COLUMN IF NOT EXISTS mahnstufe INTEGER NOT NULL DEFAULT 0;
ALTER TABLE ausleihen ADD COLUMN IF NOT EXISTS letztes_mahndatum TIMESTAMP WITH TIME ZONE;
