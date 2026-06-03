-- Migration 004: Aussonderung – adds decommission status to book copies.
-- Ausgesonderte (decommissioned) copies remain in the database for statistics
-- but are hidden from catalog, kiosk, and inventory lists.

ALTER TABLE buecher_exemplare
    ADD COLUMN IF NOT EXISTS ist_ausgesondert BOOLEAN NOT NULL DEFAULT false;

-- Sparse index: only decommissioned copies, for fast exclusion queries
CREATE INDEX IF NOT EXISTS idx_buecher_exemplare_ausgesondert
    ON buecher_exemplare(ist_ausgesondert)
    WHERE ist_ausgesondert = true;
