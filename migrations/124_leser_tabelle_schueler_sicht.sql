-- =============================================================================
-- Migration 124: Die Tabelle heißt leser; schueler wird eine Sicht darauf
-- =============================================================================
-- 51 Abfragen in 30 Dateien lesen `FROM schueler` und meinen damit „Schüler".
-- Sobald Kollegen in derselben Tabelle stehen, müsste jede einzelne davon
-- `art = 'schueler'` ergänzen — und wer eine übersieht, hat einen Kollegen in
-- einer Klassenliste, im Mahnlauf oder in der Abgänger-Versetzung. Ein Fehler,
-- den niemand sieht, weil die Zeile ja „irgendwie dazugehört".
--
-- Deshalb umgekehrt: Die TABELLE heißt `leser` und führt alle Leser. `schueler`
-- ist eine SICHT darauf, die nur Schüler zeigt. Damit behalten alle 51 Abfragen
-- ihre Bedeutung, ohne angefasst zu werden, und können einen Kollegen weder
-- sehen noch anlegen. Neue Abfragen, die ALLE Leser meinen, nennen `leser`.
--
-- Am 16.09.2026 an einer echten Postgres-18-Instanz nachgemessen, statt der
-- Dokumentation zu glauben — eine von selbst schreibbare Sicht kann alles, was
-- die Abfragen brauchen:
--
--   SELECT ... FOR UPDATE      geht  (der Ausleihweg sperrt die Zeile so)
--   INSERT ... RETURNING id    geht  (art fällt auf 'schueler' zurück)
--   UPDATE / DELETE            geht
--   INSERT mit art='lehrkraft' wird abgewiesen (WITH CHECK OPTION)
--   TRUNCATE                   geht NICHT — „is not a table"
--
-- Das letzte trifft drei Test-Helfer, die `TRUNCATE ... schueler ...` fahren;
-- sie nennen jetzt `leser`. Im Betrieb truncatet niemand.
--
-- WITH CHECK OPTION ist der Kern und nicht Zierde: Ohne sie könnte ein
-- Schreibweg durch die Sicht eine Zeile anlegen, die die Sicht danach nicht
-- mehr zeigt — angelegt und sofort unsichtbar.
-- =============================================================================

ALTER TABLE schueler RENAME TO leser;

-- Indexe, Constraints und Trigger wandern mit der Tabelle. Die SELBST BENANNTEN
-- behalten ihren Namen (uniq_schueler_barcode_active, chk_schueler_block_reason,
-- …): Sie stehen in den Schema-Ratschen und in Fehlermeldungen, die der Code
-- auswertet (unique_violation mit CONSTRAINT). Ein Umbenennen wäre ein eigener
-- Schritt ohne Verhaltensänderung.
--
-- Die von Postgres SELBST erzeugten Namen müssen dagegen mit: Sie tragen den
-- Tabellennamen, und ein RENAME zieht sie nicht nach. Eine frisch aus schema.sql
-- gebaute Anlage hieße hier leser_pkey und leser_<spalte>_not_null, eine
-- gewachsene weiter schueler_… — die Paritäts-Ratsche (gewachsen ≡ frisch) fällt
-- darüber, und zwar zu Recht: Zwei Anlagen mit verschiedenen Constraint-Namen
-- sind zwei verschiedene Schemata, auch wenn sie sich gleich verhalten.
--
-- Kein Code hängt an diesen Namen (nachgesehen am 16.09.2026); benannte NOT-NULL-
-- Constraints gibt es überhaupt erst seit Postgres 18.
-- Das Umbenennen des Primärschlüssel-CONSTRAINTS nimmt seinen Index mit; ein
-- zusätzliches ALTER INDEX fände ihn unter dem alten Namen nicht mehr.
ALTER TABLE leser RENAME CONSTRAINT schueler_pkey TO leser_pkey;
ALTER TABLE leser RENAME CONSTRAINT schueler_id_not_null TO leser_id_not_null;
ALTER TABLE leser RENAME CONSTRAINT schueler_vorname_not_null TO leser_vorname_not_null;
ALTER TABLE leser RENAME CONSTRAINT schueler_nachname_not_null TO leser_nachname_not_null;
ALTER TABLE leser RENAME CONSTRAINT schueler_art_not_null TO leser_art_not_null;
ALTER TABLE leser RENAME CONSTRAINT schueler_ist_gesperrt_not_null TO leser_ist_gesperrt_not_null;
ALTER TABLE leser RENAME CONSTRAINT schueler_ist_abgaenger_not_null TO leser_ist_abgaenger_not_null;
ALTER TABLE leser RENAME CONSTRAINT schueler_erstellt_am_not_null TO leser_erstellt_am_not_null;
ALTER TABLE leser RENAME CONSTRAINT schueler_aktualisiert_am_not_null TO leser_aktualisiert_am_not_null;

CREATE VIEW schueler AS
    SELECT * FROM leser WHERE art = 'schueler'
    WITH CHECK OPTION;

COMMENT ON VIEW schueler IS
    'Nur die Schüler aus leser. Alle Abfragen, die "Schüler" meinen, lesen diese '
    'Sicht; WITH CHECK OPTION verhindert, dass durch sie ein Nicht-Schüler entsteht.';

-- ── Der Ausweis-Wächter muss die Umbenennung mitbekommen ─────────────────────
--
-- ausweis_eindeutig_ueber_personen() (Migration 118) hält eine Ausweisnummer
-- über schueler UND benutzer eindeutig. Zwei Stellen darin hängen am Namen:
--
--   1. `IF TG_TABLE_NAME = 'schueler'` — nach der Umbenennung heißt die Tabelle
--      'leser', der Vergleich schlägt fehl, und der Trigger liefe für einen
--      Leser in den Benutzer-Zweig. Er würde die falsche Tabelle prüfen.
--   2. `SELECT 1 FROM schueler ...` im Benutzer-Zweig — das ist ab jetzt die
--      SICHT und zeigt nur Schüler. Die Nummer eines Kollegen wäre damit nicht
--      mehr im Blick, und zwei Personen könnten dieselbe tragen. Genau der
--      Fehlgriff an der Theke, gegen den Migration 118 gebaut wurde.
--
-- Beide Stellen lesen jetzt `leser`. Der Rest der Funktion ist unverändert
-- (Fehlertext und CONSTRAINT-Name bleiben, der Code wertet sie aus).
CREATE OR REPLACE FUNCTION ausweis_eindeutig_ueber_personen()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.barcode_id IS NULL THEN
        RETURN NEW;
    END IF;
    IF TG_TABLE_NAME = 'leser' THEN
        IF NEW.deleted_at IS NOT NULL THEN
            RETURN NEW;
        END IF;
        IF TG_OP = 'UPDATE' THEN
            IF NEW.barcode_id IS NOT DISTINCT FROM OLD.barcode_id AND OLD.deleted_at IS NULL THEN
                RETURN NEW;
            END IF;
        END IF;
        PERFORM pg_advisory_xact_lock(hashtext('ausweisnummer'), hashtext(NEW.barcode_id));
        IF EXISTS (SELECT 1 FROM benutzer WHERE barcode_id = NEW.barcode_id) THEN
            RAISE EXCEPTION 'Ausweisnummer % trägt bereits eine Lehrkraft', NEW.barcode_id
                USING ERRCODE = 'unique_violation', CONSTRAINT = 'uniq_ausweis_ueber_personen';
        END IF;
    ELSE
        IF TG_OP = 'UPDATE' THEN
            IF NEW.barcode_id IS NOT DISTINCT FROM OLD.barcode_id THEN
                RETURN NEW;
            END IF;
        END IF;
        PERFORM pg_advisory_xact_lock(hashtext('ausweisnummer'), hashtext(NEW.barcode_id));
        IF EXISTS (SELECT 1 FROM leser WHERE barcode_id = NEW.barcode_id AND deleted_at IS NULL) THEN
            RAISE EXCEPTION 'Ausweisnummer % trägt bereits ein Schüler', NEW.barcode_id
                USING ERRCODE = 'unique_violation', CONSTRAINT = 'uniq_ausweis_ueber_personen';
        END IF;
    END IF;
    RETURN NEW;
END $$;
