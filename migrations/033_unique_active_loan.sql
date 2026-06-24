-- =============================================================================
-- Migration 033: Datenintegrität — höchstens EINE aktive Ausleihe pro Exemplar/Gerät
-- =============================================================================
-- Bisher verhinderte nichts auf DB-Ebene, dass ein Exemplar (oder Gerät) gleichzeitig
-- zwei offene Ausleihen hat. Die Anwendungslogik nutzt zwar FOR UPDATE auf die aktive
-- Ausleihzeile, aber FOR UPDATE kann eine noch nicht existierende Zeile nicht sperren:
-- Zwei zeitgleiche Checkouts desselben Exemplars durch unterschiedliche Personen
-- (unterschiedliche Idempotenz-Keys → der Schüler-Lock greift nicht) sehen beide
-- "keine aktive Ausleihe" und legen je eine an → zwei aktive Ausleihen auf einem Exemplar.
--
-- Diese Migration macht die Invariante "≤ 1 aktive Ausleihe je Exemplar/Gerät" zur
-- harten DB-Garantie. Ein zweiter konkurrierender Checkout scheitert dann sauber an der
-- Unique-Constraint, statt stillschweigend Daten zu korrumpieren.
-- =============================================================================

-- 1. Bestehende Duplikate bereinigen, damit der Unique-Index angelegt werden kann.
--    Pro Exemplar/Gerät bleibt die JÜNGSTE offene Ausleihe aktiv (= aktueller Besitzer),
--    ältere offene Ausleihen werden als zurückgegeben markiert.
WITH ranked_exemplar AS (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY exemplar_id ORDER BY ausgeliehen_am DESC, id DESC) AS rn
    FROM ausleihen
    WHERE rueckgabe_am IS NULL AND exemplar_id IS NOT NULL
)
UPDATE ausleihen a
SET rueckgabe_am = CURRENT_TIMESTAMP
FROM ranked_exemplar r
WHERE a.id = r.id AND r.rn > 1;

WITH ranked_geraet AS (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY geraet_id ORDER BY ausgeliehen_am DESC, id DESC) AS rn
    FROM ausleihen
    WHERE rueckgabe_am IS NULL AND geraet_id IS NOT NULL
)
UPDATE ausleihen a
SET rueckgabe_am = CURRENT_TIMESTAMP
FROM ranked_geraet r
WHERE a.id = r.id AND r.rn > 1;

-- 2. Unique-Indizes für genau eine aktive Ausleihe je Exemplar bzw. Gerät.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_ausleihen_aktiv_exemplar
    ON ausleihen (exemplar_id)
    WHERE rueckgabe_am IS NULL AND exemplar_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_ausleihen_aktiv_geraet
    ON ausleihen (geraet_id)
    WHERE rueckgabe_am IS NULL AND geraet_id IS NOT NULL;
