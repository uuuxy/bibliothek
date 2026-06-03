-- ============================================================
-- Migration: DSGVO-Härtung LUSD-Import – Datensparsamkeit
-- Rechtsgrundlage: Art. 5 Abs. 1 lit. c DSGVO (Datensparsamkeit)
-- Datum: 2026-05-31
-- ============================================================

-- 1. Geburtsdatum als einziges zusätzlich erlaubtes LUSD-Feld.
--    NULL für Altdatensätze, die vor dieser Migration importiert wurden.
ALTER TABLE schueler
  ADD COLUMN IF NOT EXISTS geburtsdatum DATE DEFAULT NULL;

-- 2. Sicherheitsnachweis: Die Tabelle schueler enthält KEINE
--    adress- oder kontaktbezogenen Spalten (Straße, PLZ, Wohnort,
--    Telefonnummern o. ä.). Das folgende Statement prüft das
--    automatisch und schlägt fehl, falls eine solche Spalte
--    versehentlich existieren sollte.
DO $$
DECLARE
  forbidden_cols TEXT[] := ARRAY[
    'strasse', 'hausnummer', 'plz', 'wohnort', 'ort', 'stadt',
    'telefon', 'telefonnummer', 'handy', 'mobil',
    'email', 'e_mail', 'kontakt',
    'erziehungsberechtigter', 'erziehungsberechtigte'
  ];
  col TEXT;
  found_col TEXT;
BEGIN
  FOREACH col IN ARRAY forbidden_cols LOOP
    SELECT column_name INTO found_col
    FROM information_schema.columns
    WHERE table_name = 'schueler'
      AND table_schema = current_schema()
      AND lower(column_name) = lower(col)
    LIMIT 1;

    IF found_col IS NOT NULL THEN
      RAISE EXCEPTION
        'DSGVO-Verletzung: Spalte "%" in Tabelle schueler enthält '
        'rechtlich unzulässige personenbezogene Daten und muss entfernt '
        'werden: ALTER TABLE schueler DROP COLUMN %;',
        found_col, found_col;
    END IF;
  END LOOP;
END $$;
