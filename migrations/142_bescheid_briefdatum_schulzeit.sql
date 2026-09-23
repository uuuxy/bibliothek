-- =============================================================================
-- Migration 142: das Briefdatum eines Bescheids ist der Kalendertag der Schule
-- =============================================================================
-- schadensersatz_bescheide.brief_datum stand seit Migration 110 auf DEFAULT CURRENT_DATE,
-- dem Tag der Datenbank-Sitzung (im Image UTC). Ein Bescheid, der zwischen Mitternacht in
-- Berlin und 2 Uhr entsteht, trug den Vortag als Briefdatum — neben einer Frist, die seit
-- dem 23.09.2026 am Tag der Schule gemessen wird (api/bescheid_handler.go).
--
-- Dieselbe Klasse wie Migration 130 (Wareneingang) und 139 (Zugangsdatum); es war die
-- letzte Spaltenvorgabe dieser Art. Das Gate TestKalendertag_KeineSpaltenvorgabeMitDemTagDerSitzung
-- (repository/kalendertag_vorgaben_pg_test.go) hält es am fertigen Schema fest.
--
-- Nur die Vorgabe; bestehende Briefdaten bleiben. Idempotent: SET DEFAULT.

ALTER TABLE schadensersatz_bescheide
    ALTER COLUMN brief_datum SET DEFAULT ((now() AT TIME ZONE 'Europe/Berlin')::date);
