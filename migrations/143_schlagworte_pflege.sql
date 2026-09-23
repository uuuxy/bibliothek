-- =============================================================================
-- Migration 143: Schlagworte pflegen — Verweise und Filter-Markierung
-- =============================================================================
-- Stufe 1 von docs/OFFEN.md 4.20 (freigegeben am 23.09.2026). Eine freie Liste bleibt
-- brauchbar, wie in Littera, durch Pflege: umbenennen, zusammenführen, löschen, Verweise
-- („Tierfantasy" → „Fantasy") und die Markierung der Wörter, die im Portal als Filter stehen.
--
-- verweis_auf: Ein Verweis ist ein Wort, das auf ein anderes zeigt. Wer ihn tippt, bekommt
-- am Titel das Ziel (repository.SetzeSchlagworte löst auf). Beim Zusammenführen wird das
-- alte Wort zum Verweis auf das neue — wer es gewohnt ist, landet weiter richtig.
-- ON DELETE CASCADE: Fällt ein Wort, fallen die Verweise darauf mit; ein Verweis ohne Ziel
-- hätte keine Bedeutung.
--
-- ist_filter: das Wort steht im Portal als Filter (Stufe 2 liest es). Ein Verweis ist nie
-- Filter — gefiltert wird nach dem Ziel.
--
-- Die Regeln stehen in der Datenbank, nicht nur in der einen Schreib-Tür
-- (repository/schlagworte_pflege.go): kein Verweis auf sich selbst, keine Kette (ein Verweis
-- zeigt auf ein Wort, nicht auf einen Verweis), kein Titel an einem Verweis.
--
-- Idempotent: ADD COLUMN IF NOT EXISTS, Constraints über pg_constraint geprüft,
-- CREATE OR REPLACE / DROP TRIGGER IF EXISTS.

ALTER TABLE schlagworte ADD COLUMN IF NOT EXISTS verweis_auf UUID
    REFERENCES schlagworte(id) ON DELETE CASCADE;
ALTER TABLE schlagworte ADD COLUMN IF NOT EXISTS ist_filter BOOLEAN NOT NULL DEFAULT false;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_schlagwort_verweis_nicht_selbst') THEN
        ALTER TABLE schlagworte ADD CONSTRAINT chk_schlagwort_verweis_nicht_selbst
            CHECK (verweis_auf IS NULL OR verweis_auf <> id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_schlagwort_verweis_kein_filter') THEN
        ALTER TABLE schlagworte ADD CONSTRAINT chk_schlagwort_verweis_kein_filter
            CHECK (verweis_auf IS NULL OR NOT ist_filter);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_schlagworte_verweis_auf ON schlagworte (verweis_auf)
    WHERE verweis_auf IS NOT NULL;

-- Keine Kette, und ein Wort, das Verweise oder Titel trägt, wird nicht selbst zum Verweis.
CREATE OR REPLACE FUNCTION schlagwort_verweis_pruefen()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.verweis_auf IS NOT NULL THEN
        IF EXISTS (SELECT 1 FROM schlagworte WHERE id = NEW.verweis_auf AND verweis_auf IS NOT NULL) THEN
            RAISE EXCEPTION 'schlagwort_verweis_kette: ein Verweis zeigt auf ein Schlagwort, nicht auf einen Verweis';
        END IF;
        IF EXISTS (SELECT 1 FROM schlagworte WHERE verweis_auf = NEW.id) THEN
            RAISE EXCEPTION 'schlagwort_verweis_kette: auf dieses Wort zeigen Verweise';
        END IF;
        IF EXISTS (SELECT 1 FROM titel_schlagworte WHERE schlagwort_id = NEW.id) THEN
            RAISE EXCEPTION 'schlagwort_verweis_mit_titeln: ein Verweis trägt keine Titel';
        END IF;
    END IF;
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS trg_schlagwort_verweis_pruefen ON schlagworte;
CREATE TRIGGER trg_schlagwort_verweis_pruefen
BEFORE INSERT OR UPDATE OF verweis_auf ON schlagworte
FOR EACH ROW EXECUTE FUNCTION schlagwort_verweis_pruefen();

-- Ein Titel hängt nie an einem Verweis, sondern an dessen Ziel.
CREATE OR REPLACE FUNCTION titel_schlagwort_kein_verweis()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM schlagworte WHERE id = NEW.schlagwort_id AND verweis_auf IS NOT NULL) THEN
        RAISE EXCEPTION 'schlagwort_verweis_mit_titeln: ein Titel hängt am Ziel eines Verweises, nicht am Verweis';
    END IF;
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS trg_titel_schlagwort_kein_verweis ON titel_schlagworte;
CREATE TRIGGER trg_titel_schlagwort_kein_verweis
BEFORE INSERT OR UPDATE OF schlagwort_id ON titel_schlagworte
FOR EACH ROW EXECUTE FUNCTION titel_schlagwort_kein_verweis();

COMMENT ON TABLE schlagworte IS
    'Schlagworte des Bestands (Migration 138), frei eintragbar wie in Littera. Ein Wort ist '
    'case-insensitiv eindeutig; die zuerst angelegte Schreibweise gewinnt. Seit Migration 143 '
    'mit Verweisen (verweis_auf) und Filter-Markierung (ist_filter). Geschrieben nur über '
    'repository.SetzeSchlagworte (am Titel) und repository/schlagworte_pflege.go (Pflege).';
