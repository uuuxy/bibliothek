-- =============================================================================
-- Migration 043: Strukturierter Aussonderungs-Grund für Exemplare
-- =============================================================================
-- ist_ausgesondert war ein Sammelbecken: laut repository/models.go markiert es
-- "verloren gegangene, beschädigte oder ausgemusterte" Bücher — drei fachlich
-- verschiedene Vorgänge in einem Boolean. Unterschieden wurden sie nur über den
-- Freitext zustand_notiz ('Verlust bei Inventur', 'Automatisch ausgesondert', …).
-- Dadurch liess sich die Verlustquote (zentrale Kennzahl jeder Schulbibliothek)
-- nicht auswerten.
--
-- Diese Migration ergänzt den Grund als strukturiertes Feld:
--   VERLUST           – nicht auffindbar (Inventur, Schülermeldung)
--   BESCHAEDIGUNG     – Schadensfall
--   AUSSORTIERT       – bewusst entfernt (veraltet, verschlissen)
--   BESTANDSKORREKTUR – technische Anpassung (Import/Bestands-Sync), kein echter Abgang
--
-- Bewusst NICHT eingeführt: ein Status "Ausgeliehen". Der Ausleihzustand ergibt
-- sich aus der Tabelle ausleihen (abgesichert durch den partiellen Unique-Index
-- aus Migration 033) — eine zweite Wahrheit würde nur auseinanderlaufen.
-- =============================================================================

ALTER TABLE buecher_exemplare ADD COLUMN IF NOT EXISTS aussonderung_grund VARCHAR(20);

-- Backfill (Best-Effort aus dem bisherigen Freitext, Reihenfolge = Trennschärfe):
--   1./2./3. eindeutige Markertexte der jeweiligen Schreibstellen
--   4. DecommissionCopy setzte keine Notiz -> leer bedeutet bewusste Aussonderung
--   5. alles Übrige stammt aus dem Schadensfall-Pfad (dort landet die freie
--      Schadensbeschreibung in zustand_notiz)
UPDATE buecher_exemplare SET aussonderung_grund =
    CASE
        WHEN zustand_notiz ILIKE '%Verlust%'                 THEN 'VERLUST'
        WHEN zustand_notiz ILIKE '%Automatisch ausgesondert%' THEN 'BESTANDSKORREKTUR'
        WHEN zustand_notiz ILIKE '%Systematisch gelöscht%'    THEN 'AUSSORTIERT'
        WHEN COALESCE(zustand_notiz, '') = ''                 THEN 'AUSSORTIERT'
        ELSE 'BESCHAEDIGUNG'
    END
WHERE ist_ausgesondert = true AND aussonderung_grund IS NULL;

-- Konsistenz erzwingen: im Umlauf = kein Grund, ausgesondert = genau ein gültiger
-- Grund. Damit ist ein ausgesondertes Exemplar ohne Grund unmöglich.
--
-- WICHTIG: das "IS NOT NULL" ist nicht redundant. Ohne es ergäbe der zweite Zweig
-- bei grund = NULL "TRUE AND (NULL IN (...))" = NULL, und ein CHECK schlägt nur bei
-- FALSE an — der Fall, den dieser Constraint gerade verhindern soll, käme durch.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_aussonderung_grund') THEN
        ALTER TABLE buecher_exemplare ADD CONSTRAINT chk_aussonderung_grund
            CHECK (
                (ist_ausgesondert = false AND aussonderung_grund IS NULL)
             OR (ist_ausgesondert = true  AND aussonderung_grund IS NOT NULL
                 AND aussonderung_grund IN
                    ('VERLUST', 'BESCHAEDIGUNG', 'AUSSORTIERT', 'BESTANDSKORREKTUR'))
            );
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_exemplare_aussonderung_grund
    ON buecher_exemplare (aussonderung_grund) WHERE aussonderung_grund IS NOT NULL;
