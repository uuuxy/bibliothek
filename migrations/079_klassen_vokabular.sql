-- =============================================================================
-- Migration 079: Klassen werden kontrolliertes Vokabular (Prüfbericht F3, Klassen-Seite)
-- =============================================================================
-- Die Klasse stand als bloßer Text in vier Tabellen (schueler, klassen_lehrer_mapping,
-- class_books, klassensatz_reservierungen) — nichts verband sie. "7a" beim Schüler und
-- "07A" in der Lehrkraft-Zuordnung ließen die Mahn-Mail still ins Leere laufen; bisher
-- warnte nur der Drift-Detektor der Betriebsbereitschaft.
--
-- Jetzt gibt es EINE klassen-Tabelle als Vokabular. Durchgesetzt wird sie nicht in den
-- (vielen) Go-Schreibpfaden, sondern an der Tabelle selbst — die Lehre aus "Zwei Türen
-- zum selben Zustand": LUSD-Import, Versetzung, Formulare, Littera-Übernahme, Seeds und
-- Hand-SQL schreiben alle durch dieselbe Tür.
--   * BEFORE-Trigger kanonisieren jeden geschriebenen Wert auf die registrierte
--     Schreibweise (Normal-Schlüssel: klein, ohne Leerzeichen, ohne führende Nullen —
--     "05A", "5 A" und "5a" sind dieselbe Klasse) und registrieren Unbekanntes selbst.
--   * FKs mit ON UPDATE CASCADE (Umbenennen zieht alle Tabellen mit) und
--     ON DELETE RESTRICT sichern das Vokabular strukturell ab.
--
-- BEWUSST: lehrer_anliegen.klasse bleibt Freitext (dort stehen auch Oberstufenkurse,
-- Migration 075). Sonderwerte ('ABG', '' nach DSGVO-Anonymisierung, Lesergruppen-Kürzel)
-- werden gewöhnliche Vokabeln — die Versetzung BENENNT NICHT UM, sie hängt Schüler an
-- andere Namen; Bücherlisten kleben damit weiter am Curriculum (5a-Liste bleibt bei
-- der 5), genau wie in der F3-Grundsatzentscheidung festgehalten.
-- =============================================================================

-- 1) Normal-Schlüssel: klein, ohne Leerzeichen, ohne führende Nullen vor Ziffern.
CREATE FUNCTION klassen_normkey(text) RETURNS text
LANGUAGE sql IMMUTABLE AS $$
    SELECT regexp_replace(lower(replace(btrim($1), ' ', '')), '^0+(\d)', '\1')
$$;

CREATE TABLE klassen (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,          -- die angezeigte, kanonische Schreibweise
    erstellt_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    aktualisiert_am TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER trg_klassen_aktualisiert_am
BEFORE UPDATE ON klassen
FOR EACH ROW EXECUTE FUNCTION set_aktualisiert_am();

CREATE UNIQUE INDEX uniq_klassen_normkey ON klassen (klassen_normkey(name));

-- 2) Schreibweisen-Dubletten in den Tabellen mit Unique-/PK-Zwang zusammenlegen,
--    BEVOR kanonisiert wird (sonst kollidierten sie beim Umschreiben). Deterministisch
--    gewinnt die alphabetisch kleinste Schreibweise.
DELETE FROM class_books cb
USING class_books halter
WHERE klassen_normkey(cb.class_name) = klassen_normkey(halter.class_name)
  AND cb.book_id = halter.book_id
  AND cb.class_name > halter.class_name;

DELETE FROM klassen_lehrer_mapping m
USING klassen_lehrer_mapping halter
WHERE klassen_normkey(m.klasse) = klassen_normkey(halter.klasse)
  AND m.klasse > halter.klasse;

-- 3) Rand-Leerzeichen entfernen (Dubletten dazu sind durch Schritt 2 schon weg).
UPDATE schueler SET klasse = btrim(klasse) WHERE klasse <> btrim(klasse);
UPDATE klassen_lehrer_mapping SET klasse = btrim(klasse) WHERE klasse <> btrim(klasse);
UPDATE class_books SET class_name = btrim(class_name) WHERE class_name <> btrim(class_name);
UPDATE klassensatz_reservierungen SET klasse = btrim(klasse) WHERE klasse <> btrim(klasse);

-- 4) Vokabular befüllen. Priorität der Schreibweise: Schüler (LUSD ist die Quelle) vor
--    Lehrkraft-Zuordnung vor Bücherlisten vor Klassensätzen. Auch SOFT-GELÖSCHTE
--    Schüler zählen — der FK gilt für jede Zeile. 'ABG' schreibt der Versetzungslauf.
INSERT INTO klassen (name)
SELECT DISTINCT ON (klassen_normkey(klasse)) klasse
FROM schueler
ORDER BY klassen_normkey(klasse), klasse;

INSERT INTO klassen (name)
SELECT DISTINCT ON (klassen_normkey(klasse)) klasse
FROM klassen_lehrer_mapping m
WHERE NOT EXISTS (SELECT 1 FROM klassen k WHERE klassen_normkey(k.name) = klassen_normkey(m.klasse))
ORDER BY klassen_normkey(klasse), klasse;

INSERT INTO klassen (name)
SELECT DISTINCT ON (klassen_normkey(class_name)) class_name
FROM class_books cb
WHERE NOT EXISTS (SELECT 1 FROM klassen k WHERE klassen_normkey(k.name) = klassen_normkey(cb.class_name))
ORDER BY klassen_normkey(class_name), class_name;

INSERT INTO klassen (name)
SELECT DISTINCT ON (klassen_normkey(klasse)) klasse
FROM klassensatz_reservierungen r
WHERE NOT EXISTS (SELECT 1 FROM klassen k WHERE klassen_normkey(k.name) = klassen_normkey(r.klasse))
ORDER BY klassen_normkey(klasse), klasse;

INSERT INTO klassen (name)
SELECT 'ABG'
WHERE NOT EXISTS (SELECT 1 FROM klassen k WHERE klassen_normkey(k.name) = klassen_normkey('ABG'));

-- 5) Alle vier Spalten auf die kanonische Schreibweise ziehen.
UPDATE schueler s SET klasse = k.name
FROM klassen k
WHERE klassen_normkey(s.klasse) = klassen_normkey(k.name) AND s.klasse <> k.name;

UPDATE klassen_lehrer_mapping m SET klasse = k.name
FROM klassen k
WHERE klassen_normkey(m.klasse) = klassen_normkey(k.name) AND m.klasse <> k.name;

UPDATE class_books cb SET class_name = k.name
FROM klassen k
WHERE klassen_normkey(cb.class_name) = klassen_normkey(k.name) AND cb.class_name <> k.name;

UPDATE klassensatz_reservierungen r SET klasse = k.name
FROM klassen k
WHERE klassen_normkey(r.klasse) = klassen_normkey(k.name) AND r.klasse <> k.name;

-- 6) Breite angleichen: Der Rename-CASCADE darf keinen 50-Zeichen-Namen in eine
--    20er-Spalte schieben.
ALTER TABLE schueler ALTER COLUMN klasse TYPE VARCHAR(50);

-- 7) Die EINE Tür: BEFORE-Trigger kanonisieren und registrieren. Verliert die
--    Registrierung ein Rennen (zwei gleichzeitige Erst-Schreiber derselben Klasse),
--    wartet der INSERT am Unique-Index und die zweite Abfrage liest den Gewinner.
CREATE FUNCTION klasse_kanonisieren() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE kanonisch text;
BEGIN
    IF NEW.klasse IS NULL THEN RETURN NEW; END IF;
    NEW.klasse := btrim(NEW.klasse);
    SELECT name INTO kanonisch FROM klassen
    WHERE klassen_normkey(name) = klassen_normkey(NEW.klasse);
    IF kanonisch IS NULL THEN
        INSERT INTO klassen (name) VALUES (NEW.klasse) ON CONFLICT DO NOTHING;
        SELECT name INTO kanonisch FROM klassen
        WHERE klassen_normkey(name) = klassen_normkey(NEW.klasse);
    END IF;
    IF kanonisch IS NOT NULL THEN
        NEW.klasse := kanonisch;
    END IF;
    RETURN NEW;
END $$;

-- Gleiche Logik für class_books, dessen Spalte anders heißt (plpgsql kann
-- Feldnamen nicht parametrisieren).
CREATE FUNCTION class_name_kanonisieren() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE kanonisch text;
BEGIN
    IF NEW.class_name IS NULL THEN RETURN NEW; END IF;
    NEW.class_name := btrim(NEW.class_name);
    SELECT name INTO kanonisch FROM klassen
    WHERE klassen_normkey(name) = klassen_normkey(NEW.class_name);
    IF kanonisch IS NULL THEN
        INSERT INTO klassen (name) VALUES (NEW.class_name) ON CONFLICT DO NOTHING;
        SELECT name INTO kanonisch FROM klassen
        WHERE klassen_normkey(name) = klassen_normkey(NEW.class_name);
    END IF;
    IF kanonisch IS NOT NULL THEN
        NEW.class_name := kanonisch;
    END IF;
    RETURN NEW;
END $$;

CREATE TRIGGER trg_schueler_klasse_vokabular
BEFORE INSERT OR UPDATE OF klasse ON schueler
FOR EACH ROW EXECUTE FUNCTION klasse_kanonisieren();

CREATE TRIGGER trg_klm_klasse_vokabular
BEFORE INSERT OR UPDATE OF klasse ON klassen_lehrer_mapping
FOR EACH ROW EXECUTE FUNCTION klasse_kanonisieren();

CREATE TRIGGER trg_class_books_vokabular
BEFORE INSERT OR UPDATE OF class_name ON class_books
FOR EACH ROW EXECUTE FUNCTION class_name_kanonisieren();

CREATE TRIGGER trg_ksr_klasse_vokabular
BEFORE INSERT OR UPDATE OF klasse ON klassensatz_reservierungen
FOR EACH ROW EXECUTE FUNCTION klasse_kanonisieren();

-- 8) Fremdschlüssel: Umbenennen im Vokabular zieht alle vier Tabellen mit; Löschen
--    ist gesperrt, solange irgendetwas den Namen trägt.
ALTER TABLE schueler
    ADD CONSTRAINT fk_schueler_klasse_vokabular
    FOREIGN KEY (klasse) REFERENCES klassen (name)
    ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE klassen_lehrer_mapping
    ADD CONSTRAINT fk_klm_klasse_vokabular
    FOREIGN KEY (klasse) REFERENCES klassen (name)
    ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE class_books
    ADD CONSTRAINT fk_class_books_klasse_vokabular
    FOREIGN KEY (class_name) REFERENCES klassen (name)
    ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE klassensatz_reservierungen
    ADD CONSTRAINT fk_ksr_klasse_vokabular
    FOREIGN KEY (klasse) REFERENCES klassen (name)
    ON UPDATE CASCADE ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_klassensatz_klasse ON klassensatz_reservierungen (klasse);
