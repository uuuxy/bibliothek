-- Migration 011_performance_indexes.sql
-- Optimierung für Monitor-Intervallabfragen und Dashboard-Listen

CREATE INDEX IF NOT EXISTS idx_ausleihen_ausgeliehen_am ON ausleihen(ausgeliehen_am);
CREATE INDEX IF NOT EXISTS idx_ausleihen_rueckgabe_am ON ausleihen(rueckgabe_am);
CREATE INDEX IF NOT EXISTS idx_buecher_titel_erstellt_am ON buecher_titel(erstellt_am);

-- Hinweis:
-- idx_ausleihen_exemplar, idx_ausleihen_schueler und idx_ausleihen_rueckgabe_frist existierten bereits.
-- Die GIN trigram Indizes für Fuzzy-Search (titel, vorname, nachname) existierten ebenfalls bereits.
