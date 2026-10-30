-- =============================================================================
-- Migration 170: Der Generator der Ausweisnummern übergeht die Nummern von Büchern
-- =============================================================================
-- Eine Nummer ist entweder ein Buch oder ein Ausweis (Migration 131). Der Generator kannte nur die
-- Leser und die ausgeschiedenen Nummern. Trug ein Buch die nächste freie Nummer, von Hand im
-- Barcode-Editor oder weil dort ein Ausweis gescannt wurde, scheiterte jedes neue aktive Konto beim
-- Commit an „Nummer … ist schon der Barcode eines Buchs", immer an derselben Nummer, bis jemand das
-- Buch umetikettierte.
--
-- Ohne Messung: Die Änderung liest nur und ändert keine Zeile; sie wirkt auf die nächste Vergabe.
-- Idempotent: CREATE OR REPLACE.
-- =============================================================================

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
        (SELECT coalesce(max(nummer), 0) FROM ausweisnummern_ausgeschieden),
        (SELECT coalesce(max(substr(barcode_id, 3)::bigint), 0)
           FROM buecher_exemplare
          WHERE barcode_id LIKE 'A-%' AND substr(barcode_id, 3) ~ '^[0-9]{1,15}$'))
    INTO letzte;
    IF letzte > 0 THEN
        RETURN letzte + 1;
    END IF;
    RETURN 10001;
END $$;
