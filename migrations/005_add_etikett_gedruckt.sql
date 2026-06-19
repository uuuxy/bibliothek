-- Migration: Add column etikett_gedruckt to buecher_exemplare
ALTER TABLE buecher_exemplare ADD COLUMN IF NOT EXISTS etikett_gedruckt BOOLEAN NOT NULL DEFAULT FALSE;
