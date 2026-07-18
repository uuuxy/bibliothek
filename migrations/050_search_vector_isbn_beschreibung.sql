-- =============================================================================
-- Migration 050: ISBN und Beschreibung in den Volltext-Suchvektor aufnehmen
-- =============================================================================
-- Der generierte search_vector deckte bisher nur titel, untertitel, autor und
-- verlag ab. Zentrale Suchbegriffe aus dem Bibliotheksalltag — die ISBN und Wörter
-- aus der beschreibung — fanden dadurch keinen Treffer über die Omnibox.
--
-- GENERATED-Spalten lassen sich vor PostgreSQL 17 nicht im Ausdruck ändern, daher
-- DROP + ADD. Der GIN-Index auf der Spalte wird automatisch mit verworfen und danach
-- neu erstellt. Es findet ein einmaliger Table-Rewrite + Reindex statt — bei großem
-- Katalog entsprechend Wartungsfenster/Laufzeit einplanen.
-- =============================================================================

ALTER TABLE buecher_titel DROP COLUMN search_vector;

ALTER TABLE buecher_titel ADD COLUMN search_vector TSVECTOR GENERATED ALWAYS AS (
    to_tsvector('german',
        coalesce(titel, '') || ' ' ||
        coalesce(untertitel, '') || ' ' ||
        coalesce(autor, '') || ' ' ||
        coalesce(verlag, '') || ' ' ||
        coalesce(isbn, '') || ' ' ||
        coalesce(beschreibung, '')
    )
) STORED;

CREATE INDEX idx_buecher_titel_search ON buecher_titel USING GIN (search_vector);
