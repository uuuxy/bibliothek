-- nutzungsdauer_jahre
ALTER TABLE buecher_titel ADD COLUMN IF NOT EXISTS nutzungsdauer_jahre INTEGER NOT NULL DEFAULT 1;
