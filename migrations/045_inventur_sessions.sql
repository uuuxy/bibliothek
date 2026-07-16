-- =============================================================================
-- Migration 045: Inventur als Sessions statt globalem Spaltenzustand
-- =============================================================================
-- Bisher lebte der Inventurfortschritt in EINER globalen Spalte
-- buecher_exemplare.inventur_status ('ausstehend'/'erfasst'/NULL). Das machte
-- Parallelbetrieb unmöglich und riskant:
--
--   * "Inventur starten" setzte den Status GLOBAL zurück (alle 80.000 Bücher auf
--     NULL) und markierte dann den Scope. Startete Kollege B eine zweite Inventur,
--     während Kollege A noch scannte, war A's Fortschritt sofort weg.
--   * "Abschließen" markierte ALLE 'ausstehend'-Exemplare global als Verlust —
--     also auch den Scope eines anderen Kollegen.
--
-- Zwei parallele Inventuren löschten sich damit gegenseitig; im schlimmsten Fall
-- wurden fremde Bücher als "verloren" ausgesondert.
--
-- Neues Modell: jede Inventur ist eine Session mit eigenem Scope. Was in einer
-- Session erfasst wurde, steht session-gebunden in inventur_erfassungen. Der
-- globale Spaltenzustand entfällt — es gibt keine gemeinsame Wahrheit mehr, die
-- zwei Sessions überschreiben könnten.
-- =============================================================================

-- Eine Inventur-Session: Scope (global oder je Signatur-Regalbereich), wer sie
-- gestartet hat, und ob sie noch läuft (abgeschlossen_am IS NULL).
CREATE TABLE IF NOT EXISTS inventur_sessions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope_type        VARCHAR(20) NOT NULL,
    -- ON DELETE CASCADE, nicht SET NULL: Der CHECK unten verlangt für eine
    -- Signatur-Session eine signature_id. SET NULL würde beim Löschen der Signatur
    -- genau diese Bedingung verletzen. Eine Session ist an ihre Signatur gebunden —
    -- verschwindet die Signatur, ist die Session gegenstandslos und wird mitentfernt.
    signature_id      INT REFERENCES signatures(id) ON DELETE CASCADE,
    scope_label       VARCHAR(255) NOT NULL DEFAULT '',
    gestartet_von     UUID REFERENCES benutzer(id) ON DELETE SET NULL,
    gestartet_am      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    abgeschlossen_am  TIMESTAMP WITH TIME ZONE,
    verloren_gemeldet INT,
    CONSTRAINT chk_inv_session_scope
        CHECK (scope_type IN ('global', 'signature')
               AND (scope_type = 'global' OR signature_id IS NOT NULL))
);

-- Ein erfasstes Exemplar je Session. Der Primärschlüssel macht Doppel-Scans
-- desselben Exemplars in derselben Session zu einem No-op (ON CONFLICT).
CREATE TABLE IF NOT EXISTS inventur_erfassungen (
    session_id  UUID NOT NULL REFERENCES inventur_sessions(id) ON DELETE CASCADE,
    exemplar_id UUID NOT NULL REFERENCES buecher_exemplare(id) ON DELETE CASCADE,
    erfasst_am  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    PRIMARY KEY (session_id, exemplar_id)
);

-- Nur EINE offene Session je Scope: verhindert, dass zweimal "global" oder
-- zweimal dieselbe Signatur gleichzeitig läuft (was sich beim Abschluss doch
-- wieder überschnitte). Verschiedene Signaturen parallel bleiben erlaubt.
CREATE UNIQUE INDEX IF NOT EXISTS idx_inv_session_offen_global
    ON inventur_sessions ((true)) WHERE abgeschlossen_am IS NULL AND scope_type = 'global';
CREATE UNIQUE INDEX IF NOT EXISTS idx_inv_session_offen_signature
    ON inventur_sessions (signature_id) WHERE abgeschlossen_am IS NULL AND scope_type = 'signature';

CREATE INDEX IF NOT EXISTS idx_inv_erfassung_exemplar
    ON inventur_erfassungen (exemplar_id);

-- Die globalen Zustandsspalten entfallen samt Constraint — der Fortschritt lebt
-- jetzt in inventur_erfassungen. Eine zweite Wahrheit würde nur auseinanderlaufen.
ALTER TABLE buecher_exemplare DROP CONSTRAINT IF EXISTS chk_inventur_status;
ALTER TABLE buecher_exemplare DROP COLUMN IF EXISTS inventur_status;
ALTER TABLE buecher_exemplare DROP COLUMN IF EXISTS inventur_geprueft_am;
