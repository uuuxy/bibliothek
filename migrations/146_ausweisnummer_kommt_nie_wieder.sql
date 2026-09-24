-- =============================================================================
-- Migration 146: Eine Ausweisnummer kommt nie wieder — auch nicht nach dem Löschen
-- =============================================================================
-- docs/OFFEN.md 5.23, freigegeben am 24.09.2026. ausweis_nummer_start() (Migration 136)
-- rechnete „höchste A-Nummer in leser + 1". Verschwand die höchste Nummer aus der Tabelle,
-- gab der Generator sie an die nächste Person, und eine noch vorhandene Karte buchte an der
-- Theke auf sie. Verschwinden kann eine Nummer auf fünf Wegen, alle am Verhalten
-- nachgestellt (api/ausweisnummer_kommt_nie_wieder_pg_test.go): endgültiges Löschen
-- (PurgeStudent, DSGVO-Löschung), Anonymisieren (ANON-…), Zusammenführen (die Quelle fällt),
-- Leeren an einem Leser ohne aktives Konto und Umschreiben von Hand.
--
-- Littera kennt dasselbe als Einstellung „freie Nummern wieder vergeben" (Handbuch,
-- „Nummernvergabe/Nummernkreis"), und die Schule hat sie genutzt: In littera_sav.mdb bekamen
-- die 349 Anmeldungen von 2010 Nummern zwischen 2 und 1743, während Nummern bis 3531 vergeben
-- waren. Littera verschlüsselt die Lesernummer im Leser-Barcode, bei uns ist die A-Nummer
-- selbst der Scanwert. Die Einstellung, die das Handbuch gegen Überschneidungen empfiehlt
-- („fortlaufend ab der höchsten Nummer"), ist die Regel von 136 — sie versagt genau dann,
-- wenn die höchste Nummer verschwindet.
--
-- Die Antwort: ausweisnummern_ausgeschieden hält jede A-Nummer fest, die aus leser
-- verschwindet, über einen Trigger an der Tabelle — so nimmt ihn jeder Weg, auch einer, den es
-- noch nicht gibt. Der Generator zieht über das Maximum aus beidem. Die Tabelle hält nur die
-- Zahl, keine Person und kein Datum: Sie sagt nicht, wem eine Nummer gehörte.
--
-- Warum keine Sequenz wie barcode_seq (repository/barcode_vergabe.go): Eine Sequenz kennt nur,
-- was sie selbst ausgibt. Der LUSD-Lauf zieht einmal und zählt dann selbst weiter, die
-- Littera-Übernahme erfindet Ersatznummern, und von Hand eingetragene Nummern stehen nur in
-- der Tabelle. Verschwände eine davon vor dem nächsten Ziehen, käme sie trotzdem wieder. Der
-- Trigger sieht jede, gleich woher sie kam.
--
-- Nur Einfügen, keine Zählerzeile: Wer eine Nummer entfernt, hält die Leserzeile und fügt
-- danach eine Zahl ein. Eine gemeinsame Zählerzeile wäre eine zweite Sperre, die jeder
-- Entferner nach seiner Leserzeile nimmt, während der LUSD-Lauf Abgänger mitten im Lauf
-- anonymisiert und danach weitere Leserzeilen braucht. Eine Einfügung wartet nur auf eine
-- gleichzeitige Einfügung derselben Zahl.
--
-- Sichtbarkeit: Solange die entfernende Transaktion offen ist, sehen alle anderen die Zeile
-- noch mit ihrer Nummer; mit dem Commit werden Entfernen und Eintrag zugleich sichtbar. Es gibt
-- keinen Augenblick, in dem der Generator die Nummer nirgends sieht.
--
-- Von Hand bleibt eine frühere Nummer eintragbar, wie heute schon die eines Lesers im
-- Papierkorb (die alte Karte eines zurückgekehrten Schülers). Die Regel gilt dem Generator
-- und der Ersatzvergabe der Littera-Übernahme (internal/littera, ersatzFrei).
--
-- Gemessen am Testserver am 24.09.2026: 8 A-Nummern (höchste A-10008), keine gelöscht,
-- anonymisiert oder doppelt; audit_log, audit_logs und nachbuch_meldungen nennen keine
-- A-Nummer. Aus der Vergangenheit ist nichts nachzutragen, die Tabelle beginnt leer.

CREATE TABLE IF NOT EXISTS ausweisnummern_ausgeschieden (
    nummer bigint PRIMARY KEY
        CONSTRAINT chk_ausweisnummer_ausgeschieden_positiv CHECK (nummer > 0)
);

CREATE OR REPLACE FUNCTION ausweisnummer_ausgeschieden()
RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    n bigint;
BEGIN
    IF TG_OP = 'UPDATE' AND NEW.barcode_id IS NOT DISTINCT FROM OLD.barcode_id THEN
        RETURN NULL;
    END IF;
    IF OLD.barcode_id IS NULL OR OLD.barcode_id !~ '^A-[0-9]{1,15}$' THEN
        RETURN NULL;
    END IF;
    n := substr(OLD.barcode_id, 3)::bigint;
    IF n > 0 THEN
        INSERT INTO ausweisnummern_ausgeschieden (nummer) VALUES (n) ON CONFLICT DO NOTHING;
    END IF;
    RETURN NULL;
END $$;

DROP TRIGGER IF EXISTS trg_leser_ausweisnummer_ausgeschieden ON leser;
CREATE TRIGGER trg_leser_ausweisnummer_ausgeschieden
AFTER DELETE OR UPDATE OF barcode_id ON leser
FOR EACH ROW EXECUTE FUNCTION ausweisnummer_ausgeschieden();

CREATE OR REPLACE FUNCTION ausweis_nummer_start()
RETURNS bigint LANGUAGE plpgsql AS $$
DECLARE
    letzte bigint;
BEGIN
    PERFORM pg_advisory_xact_lock(hashtextextended('leser.barcode_id.A-', 0));
    SELECT GREATEST(
        (SELECT coalesce(max(substr(barcode_id, 3)::bigint), 0)
           FROM leser
          WHERE barcode_id LIKE 'A-%' AND substr(barcode_id, 3) ~ '^[0-9]{1,15}$'),
        (SELECT coalesce(max(nummer), 0) FROM ausweisnummern_ausgeschieden))
    INTO letzte;
    IF letzte > 0 THEN
        RETURN letzte + 1;
    END IF;
    RETURN 10001;
END $$;
