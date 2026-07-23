-- =============================================================================
-- Migration 053: Inventur-Scope um Fach (subject) + Klasse (grade) erweitern
-- =============================================================================
-- Bisher konnte eine Inventur nur 'global' oder je 'signature' laufen. Für gezielte
-- Teil-Inventuren ("nur Mathe, Klasse 5") kommt ein 'filter'-Scope dazu, der optional
-- nach Fach (buecher_titel.subject) und/oder Klasse (jahrgang_von..jahrgang_bis) filtert.
--
-- Rein additiv: zwei nullable Spalten, ein erweiterter CHECK, ein zusätzlicher
-- partieller Unique-Index. Bestehende Sessions/Daten bleiben unberührt.
-- =============================================================================

ALTER TABLE inventur_sessions
    ADD COLUMN IF NOT EXISTS scope_subject TEXT,
    ADD COLUMN IF NOT EXISTS scope_grade   SMALLINT;

-- CHECK erweitern: 'filter' zulassen; jeder Typ verlangt seine Dimension(en).
ALTER TABLE inventur_sessions DROP CONSTRAINT IF EXISTS chk_inv_session_scope;
ALTER TABLE inventur_sessions ADD CONSTRAINT chk_inv_session_scope
    CHECK (
        scope_type IN ('global', 'signature', 'filter')
        AND (scope_type <> 'signature' OR signature_id IS NOT NULL)
        AND (scope_type <> 'filter' OR scope_subject IS NOT NULL OR scope_grade IS NOT NULL)
    );

-- Nur EINE offene Filter-Session je (Fach, Klasse) — verhindert, dass zwei Leute
-- parallel dieselbe Teilmenge inventarisieren und Verluste doppelt buchen.
CREATE UNIQUE INDEX IF NOT EXISTS idx_inv_session_offen_filter
    ON inventur_sessions (coalesce(scope_subject, ''), coalesce(scope_grade, -1))
    WHERE abgeschlossen_am IS NULL AND scope_type = 'filter';
