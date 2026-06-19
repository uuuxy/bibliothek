-- Migration 024: Add inventur_status to buecher_exemplare
-- Adds a dedicated status column for unified inventory (Start/Scan/Finish)

ALTER TABLE buecher_exemplare ADD COLUMN IF NOT EXISTS inventur_status VARCHAR(20) DEFAULT NULL;
