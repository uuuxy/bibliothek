-- =============================================================================
-- Migration 144: Die Regeln der Schlagwort-Verweise halten auch bei gleichzeitigen Schreibern
-- =============================================================================
-- Rasterdurchgang 23.09.2026 über die Schlagworte. Migration 143 sagt zu: kein Titel an einem
-- Verweis, keine Kette von Verweisen — und zwar in der Datenbank, nicht nur in der Tür. Die
-- beiden Trigger prüften dafür je nur ihren eigenen Augenblick. Zwei Schreiber, die sich
-- überschneiden, sahen die offene Änderung des anderen nicht, und beide kamen durch:
--
--   - Das Zusammenführen hat „Detektiv" gesperrt, aber noch nicht zum Verweis gemacht. Das
--     Buchformular hängt einen Titel an „Detektiv" (der Trigger sieht noch kein Verweis), das
--     Zusammenführen macht „Detektiv" zum Verweis (sein Trigger sieht den Titel noch nicht).
--     Danach hängt ein Titel an einem Verweis.
--   - Ein Verweis „Tierfantasy" → „Fantasy" entsteht, während „Fantasy" selbst zum Verweis
--     wird. Danach eine Kette.
--
-- Beides am Verhalten nachgestellt (repository/schlagworte_pflege_pg_test.go, Gleichzeitig…).
--
-- Die Antwort: Jeder Trigger sperrt die Zeile, die er prüft, mit FOR SHARE, bevor er prüft.
-- Hat ein anderer sie zum Ändern gesperrt, wartet er, und seine Prüfung sieht danach den
-- neuen Stand (READ COMMITTED: jede Abfrage einer VOLATILE-Funktion nimmt einen frischen
-- Stand). Umgekehrt wartet ein Ändern der Zeile, bis der Prüfende fertig ist. Wer zu spät
-- kommt, bekommt die Ausnahme statt einer stillen Verletzung — im Fall oben scheitert das
-- Speichern des Buchformulars einmal, ein zweites Speichern löst „Detektiv" zum Ziel auf.
--
-- Für die Schreiber im Code entsteht dadurch keine neue Wartekante: Die Pflege-Türen sperren
-- ihre Wörter vorher FOR UPDATE, und damit kollidierte schon die Fremdschlüssel-Prüfung
-- derselben Anweisung (FOR KEY SHARE, in derselben Zeilenfolge am Ende der Anweisung) — die
-- Sperre kommt jetzt nur früher, vor der Prüfung statt danach. Eine Sperre in Go
-- über alle Schreiber wurde erwogen und verworfen: Das Buchformular ruft das Speichern der
-- Schlagworte erst nach seinem UPDATE auf buecher_titel; ändert es dabei die ISBN (UNIQUE),
-- hält es den Titel FOR UPDATE, und eine vorgeschaltete Sperre hätte mit dem Zusammenführen
-- eines Worts dieses Titels verklemmen können — auch dann, wenn das Formular das Wort gar
-- nicht nennt.
--
-- Idempotent: CREATE OR REPLACE FUNCTION; die Trigger selbst bleiben unverändert. Die
-- Funktionskörper stehen wörtlich so in schema.sql; das Paritäts-Gate vergleicht seit
-- diesem Durchgang auch den Körper (md5 von prosrc), nicht nur den Namen.

CREATE OR REPLACE FUNCTION schlagwort_verweis_pruefen()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.verweis_auf IS NOT NULL THEN
        PERFORM 1 FROM schlagworte WHERE id = NEW.verweis_auf FOR SHARE;
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

CREATE OR REPLACE FUNCTION titel_schlagwort_kein_verweis()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    PERFORM 1 FROM schlagworte WHERE id = NEW.schlagwort_id FOR SHARE;
    IF EXISTS (SELECT 1 FROM schlagworte WHERE id = NEW.schlagwort_id AND verweis_auf IS NOT NULL) THEN
        RAISE EXCEPTION 'schlagwort_verweis_mit_titeln: ein Titel hängt am Ziel eines Verweises, nicht am Verweis';
    END IF;
    RETURN NEW;
END $$;
