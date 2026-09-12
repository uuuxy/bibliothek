-- Migration 115: Die Zusicherungen des LMF-Planers stehen in der Datenbank, nicht nur
-- in Go.
--
-- Drei Sätze galten bisher per Verabredung (Register, Paket 5): Die Art einer Zeile ist
-- die Art ihres Plans; die Position ist je Plan eindeutig; die letzte Stunde des
-- Rückgabe-Plans liegt im Schultag, den derselbe Plan beschreibt. Alle drei hält heute
-- EIN Schreiber ein (repository.SaveLmfPlanIn setzt sie selbst), und genau deshalb ist
-- das hier billig: Die Constraints kosten nichts, solange niemand sie verletzt, und sie
-- stehen, wenn der zweite Schreiber kommt — ein Reparaturskript, eine Hand an psql, ein
-- späterer Einzel-Endpunkt.
--
-- Was sie verhindern, wenn sie greifen: eine Zeile unter der falschen Überschrift im
-- Portal (und ohne Frist-Kopplung), zwei Klassen auf demselben Platz mit einer Anzeige,
-- die von der Speicherreihenfolge abhängt, und ein Plan, der von einer Stunde rückwärts
-- rechnet, die es an diesem Tag nicht gibt.
--
-- Verletzt der Bestand einen der drei Sätze, bricht die Migration LAUT ab: Welche der
-- beiden Zeilen die richtige ist, entscheidet der Mensch, der den Plan gemacht hat.

DO $$
DECLARE
	art_abweichung INT;
	positions_dubletten INT;
	stunde_ausserhalb INT;
BEGIN
	SELECT count(*) INTO art_abweichung
	  FROM lmf_termine t JOIN lmf_plaene p ON p.id = t.plan_id
	 WHERE t.art <> p.art;
	IF art_abweichung > 0 THEN
		RAISE EXCEPTION 'Migration 115: % Zeile(n) tragen eine andere Art als ihr Plan. Anzeigen mit: SELECT t.id, t.art, p.art AS plan_art, p.schuljahr_beginn FROM lmf_termine t JOIN lmf_plaene p ON p.id = t.plan_id WHERE t.art <> p.art;', art_abweichung;
	END IF;

	SELECT count(*) INTO positions_dubletten
	  FROM (SELECT plan_id, position FROM lmf_termine GROUP BY 1, 2 HAVING count(*) > 1) d;
	IF positions_dubletten > 0 THEN
		RAISE EXCEPTION 'Migration 115: % Position(en) kommen in ihrem Plan mehrfach vor. Anzeigen mit: SELECT plan_id, position, count(*) FROM lmf_termine GROUP BY 1, 2 HAVING count(*) > 1;', positions_dubletten;
	END IF;

	SELECT count(*) INTO stunde_ausserhalb
	  FROM lmf_plaene WHERE letzte_stunde IS NOT NULL AND letzte_stunde > stunden_je_tag;
	IF stunde_ausserhalb > 0 THEN
		RAISE EXCEPTION 'Migration 115: % Plan/Pläne haben eine letzte Stunde jenseits ihrer Stunden je Tag. Anzeigen mit: SELECT id, art, schuljahr_beginn, stunden_je_tag, letzte_stunde FROM lmf_plaene WHERE letzte_stunde > stunden_je_tag;', stunde_ausserhalb;
	END IF;
END $$;

-- Bezugspunkt des zusammengesetzten Fremdschlüssels. Die Art eines Plans ändert sich
-- nie (sie ist Teil des Konflikt-Schlüssels beim Speichern), der Schlüssel ist also
-- stabil.
ALTER TABLE lmf_plaene
	ADD CONSTRAINT uniq_lmf_plaene_id_art UNIQUE (id, art);

-- Die Art der Zeile ist die Art ihres Plans — über den Fremdschlüssel geprüft, nicht
-- über einen Trigger: Ein CHECK kann keine zweite Tabelle lesen.
ALTER TABLE lmf_termine
	ADD CONSTRAINT uniq_lmf_termine_plan_position UNIQUE (plan_id, position),
	ADD CONSTRAINT fk_lmf_termine_plan_art FOREIGN KEY (plan_id, art)
		REFERENCES lmf_plaene (id, art) ON DELETE CASCADE;

-- idx_lmf_termine_plan (plan_id, position) fällt: Der Unique-Index oben deckt dieselben
-- Spalten in derselben Reihenfolge, ein zweiter Index darüber kostet nur Schreibzeit.
DROP INDEX IF EXISTS idx_lmf_termine_plan;

ALTER TABLE lmf_plaene
	ADD CONSTRAINT chk_lmf_plaene_letzte_stunde_im_tag
		CHECK (letzte_stunde IS NULL OR letzte_stunde <= stunden_je_tag);
