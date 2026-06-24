-- =============================================================================
-- Migration 032: Schema-Abgleich buecher_titel (Omnibox-Reparatur)
-- =============================================================================
-- Hintergrund: Einige Spalten von buecher_titel wurden über die Zeit zu schema.sql
-- hinzugefügt, ohne dass es dafür eine eigene Migration gab. Produktivdatenbanken,
-- die vor dieser Erweiterung initialisiert wurden, erhielten diese Spalten daher nie.
-- Die Omnibox-Abfragen (SearchTitles, SearchTitlesFuzzy, GetCopyByBarcode) lesen u. a.
-- untertitel und ziel_jahrgang — fehlt eine dieser Spalten, schlagen GET /api/search
-- UND POST /api/action mit HTTP 500 fehl ("internal database error").
--
-- Diese Migration ist vollständig idempotent (IF NOT EXISTS) und gleicht eine bestehende
-- Datenbank an den kanonischen Stand von schema.sql an, ohne Daten zu verändern.
-- =============================================================================

-- 0. Voraussetzungen sicherstellen (Trigramm-Indizes benötigen pg_trgm).
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- 1. Fehlende Spalten ergänzen (entspricht den Definitionen in schema.sql)
ALTER TABLE buecher_titel ADD COLUMN IF NOT EXISTS untertitel VARCHAR(255);
ALTER TABLE buecher_titel ADD COLUMN IF NOT EXISTS beschreibung TEXT;
ALTER TABLE buecher_titel ADD COLUMN IF NOT EXISTS cover_url VARCHAR(512);
ALTER TABLE buecher_titel ADD COLUMN IF NOT EXISTS cover_status VARCHAR(50) DEFAULT 'PENDING';
ALTER TABLE buecher_titel ADD COLUMN IF NOT EXISTS signatur VARCHAR(255);
ALTER TABLE buecher_titel ADD COLUMN IF NOT EXISTS medientyp VARCHAR(100) NOT NULL DEFAULT 'Buch';
ALTER TABLE buecher_titel ADD COLUMN IF NOT EXISTS erweiterte_eigenschaften JSONB NOT NULL DEFAULT '{}';
ALTER TABLE buecher_titel ADD COLUMN IF NOT EXISTS ziel_jahrgang INTEGER NOT NULL DEFAULT 0;
ALTER TABLE buecher_titel ADD COLUMN IF NOT EXISTS antolin_stufen VARCHAR(50);
ALTER TABLE buecher_titel ADD COLUMN IF NOT EXISTS antolin_punkte INTEGER;
ALTER TABLE buecher_titel ADD COLUMN IF NOT EXISTS antolin_geprueft_am TIMESTAMP WITH TIME ZONE;

-- 2. Redundante Altspalte aus dem Rename-Paar 029→030 entfernen, falls noch vorhanden.
ALTER TABLE buecher_titel DROP COLUMN IF EXISTS nutzungsdauer_jahre;

-- 3. Volltextsuche-Spalte (generierte Spalte) sicherstellen — wird von SearchTitles gelesen.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'buecher_titel' AND column_name = 'search_vector'
    ) THEN
        ALTER TABLE buecher_titel ADD COLUMN search_vector TSVECTOR GENERATED ALWAYS AS (
            to_tsvector('german',
                coalesce(titel, '') || ' ' ||
                coalesce(untertitel, '') || ' ' ||
                coalesce(autor, '') || ' ' ||
                coalesce(verlag, '')
            )
        ) STORED;
    END IF;
END $$;

-- 4. Indizes für Such-Performance (idempotent).
CREATE INDEX IF NOT EXISTS idx_buecher_titel_search ON buecher_titel USING GIN (search_vector);
CREATE INDEX IF NOT EXISTS idx_buecher_titel_trgm ON buecher_titel USING gin (titel gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_buecher_autor_trgm ON buecher_titel USING gin (autor gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_buecher_isbn_trgm ON buecher_titel USING gin (isbn gin_trgm_ops);
