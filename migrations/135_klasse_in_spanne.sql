-- Migration 135: „Klasse" am Titel fällt — die Jahrgangsspanne ist die eine Angabe.
--
-- Ein Titel trug drei Jahrgangsangaben: „Klasse" (grade_level, eine Zahl), „Jahrgang von …
-- bis" (jahrgang_von/bis, die Spanne) und seit dem 22.09.2026 kurz eine dritte, die mit
-- Migration 134 zum Schalter wurde. Klasse und Spanne sagten mit denselben Zahlen fast
-- dasselbe: Der Littera-Import setzte die Klasse aus der Spanne, wenn sie einen Jahrgang
-- umfasste; der Portal-Filter las beide; das Mahnwesen („Jahrgang"), die Klassen-Inventur
-- und die Buchakte lasen nur die Spanne. In der Maske stand die Klasse als Vorgabe 5, die
-- Spanne als Vorgabe 5 bis 10 — wer die Klasse pflegte, ließ die Spanne stehen, und die
-- vier Leser der Spanne sahen davon nichts (docs/OFFEN.md 5.5, 22.09.2026).
--
-- Was die Klasse aussagte, übernimmt die Spanne: Wo eine Klasse 6 bis 13 steht und die
-- Spanne noch die Vorgabe trägt, wird die Spanne dieser eine Jahrgang. Klasse 5 wird nicht
-- übernommen — sie ist die Vorgabe der Maske und keine Aussage. Steht die Spanne schon
-- gepflegt da, gilt sie; sie ist die Angabe mit den vier Lesern.

DO $$
DECLARE
    uebernommen integer;
BEGIN
    UPDATE buecher_titel
       SET jahrgang_von = grade_level, jahrgang_bis = grade_level
     WHERE grade_level BETWEEN 6 AND 13
       AND jahrgang_von = 5 AND jahrgang_bis = 10;
    GET DIAGNOSTICS uebernommen = ROW_COUNT;
    RAISE NOTICE 'Migration 135: Klasse in die Jahrgangsspanne übernommen bei % Titel(n)', uebernommen;
END $$;

ALTER TABLE buecher_titel DROP CONSTRAINT IF EXISTS chk_grade_level_bereich;
ALTER TABLE buecher_titel DROP COLUMN IF EXISTS grade_level;
