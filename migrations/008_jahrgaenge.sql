-- Migration: 008_jahrgaenge.sql
-- Description: Adds fields for automatic, grade-level based overdue calculations

ALTER TABLE buecher_titel
ADD COLUMN IF NOT EXISTS jahrgang_von INTEGER NOT NULL DEFAULT 5,
ADD COLUMN IF NOT EXISTS jahrgang_bis INTEGER NOT NULL DEFAULT 10;
