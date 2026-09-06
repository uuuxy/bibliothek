-- =============================================================================
-- Migration 101: LMF-Plan — der Büchertausch endet am Donnerstag vor den Sommerferien
-- =============================================================================
-- Peter 06.09.2026, mit seinem Excel „Bücherrückgabe" in der Hand: „Das Programm kann
-- das doch sicherlich automatisch setzen — es endet immer am gleichen Tag: Donnerstags
-- vor den Ferien zur vierten Stunde." Der Rückgabe-Plan hat also einen festen ANKER am
-- ENDE, nicht am Anfang: Die letzte Zeile liegt auf dem letzten Platz, die übrigen
-- fließen rückwärts davor; kommen Zeilen dazu, rückt der Beginn nach vorn, das Ende
-- bleibt. (Sein Plan 2026: 56 Zeilen, Ende Do 25.06. 4. Std. → Beginn Do 11.06. 3. Std.)
--
-- letzter_tag/letzte_stunde sind dieser Anker; erster_tag/startstunde bleiben als
-- GERECHNETER Beginn stehen (wie datum/stunde der Zeilen) — Schuljahr, Sortierung und
-- die Frage „vorbei?" lesen sie wie bisher. Der Ausgabe-Plan nach den Ferien behält den
-- Anker am Anfang (erster Schultag nach den Ferien); dort bleiben die Spalten leer.
-- Der Check bindet den Anker an die Art — zwei Anker an einem Plan gäbe es sonst nur
-- durch einen Fehler. Bestehende Rückgabe-Pläne bekommen als Ende den Platz ihrer
-- letzten fließenden Zeile: Vom Ende her gerechnet ergibt das dieselben Plätze.
ALTER TABLE lmf_plaene
    ADD COLUMN IF NOT EXISTS letzter_tag   DATE,
    ADD COLUMN IF NOT EXISTS letzte_stunde SMALLINT
        CONSTRAINT chk_lmf_plaene_letzte_stunde CHECK (letzte_stunde BETWEEN 1 AND 12);

UPDATE lmf_plaene p
   SET letzter_tag = z.datum, letzte_stunde = z.stunde
  FROM (SELECT DISTINCT ON (plan_id) plan_id, datum, stunde
          FROM lmf_termine WHERE NOT fest
         ORDER BY plan_id, position DESC) z
 WHERE z.plan_id = p.id AND p.art = 'rueckgabe' AND p.letzter_tag IS NULL;

UPDATE lmf_plaene
   SET letzter_tag = erster_tag, letzte_stunde = startstunde
 WHERE art = 'rueckgabe' AND letzter_tag IS NULL;

-- Idempotent wie 094: Backfill nur WHERE … IS NULL, der Check nur, wenn er fehlt.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_lmf_plaene_anker') THEN
        ALTER TABLE lmf_plaene
            ADD CONSTRAINT chk_lmf_plaene_anker
                CHECK ((art = 'rueckgabe') = (letzter_tag IS NOT NULL AND letzte_stunde IS NOT NULL));
    END IF;
END $$;

COMMENT ON COLUMN lmf_plaene.letzter_tag IS
    'Anker des Rückgabe-Plans: letzter Tag (Donnerstag vor den Sommerferien); erster_tag ist dann gerechnet. NULL beim Ausgabe-Plan.';
