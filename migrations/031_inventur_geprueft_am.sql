-- Migration 031: Add inventur_geprueft_am to buecher_exemplare
-- Diese Spalte wird von MarkExemplarScanned benötigt, um den Zeitpunkt des
-- Inventur-Scans zu protokollieren (inventory_repo.go).

ALTER TABLE buecher_exemplare ADD COLUMN IF NOT EXISTS inventur_geprueft_am TIMESTAMP WITH TIME ZONE;
