DO $$ 
DECLARE
    title_id UUID;
BEGIN
    -- Delete old load test data if it exists to be idempotent
    DELETE FROM ausleihen WHERE schueler_id IN (SELECT id FROM schueler WHERE barcode_id LIKE 'LTS-%');
    DELETE FROM vormerkungen WHERE schueler_id IN (SELECT id FROM schueler WHERE barcode_id LIKE 'LTS-%');
    DELETE FROM buecher_exemplare WHERE barcode_id LIKE 'LTB-%';
    DELETE FROM buecher_titel WHERE isbn = '1234567890123';
    DELETE FROM schueler WHERE barcode_id LIKE 'LTS-%';

    -- Create one Book Title
    title_id := gen_random_uuid();
    INSERT INTO buecher_titel (id, titel, untertitel, autor, isbn, verlag, erscheinungsjahr, meldebestand)
    VALUES (title_id, 'Loadtest Mathematik', 'Für Test', 'k6', '1234567890123', 'Testverlag', 2024, 0);

    -- Create 5000 copies
    INSERT INTO buecher_exemplare (id, barcode_id, titel_id, erworben_am, ist_ausleihbar)
    SELECT 
        gen_random_uuid(),
        'LTB-' || i,
        title_id,
        CURRENT_DATE,
        true
    FROM generate_series(1, 5000) AS s(i);

    -- Create 500 students
    INSERT INTO schueler (id, barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt)
    SELECT 
        gen_random_uuid(),
        'LTS-' || i,
        'Schüler',
        'Test ' || i,
        'LT-' || ((i % 6) + 1), -- Classes LT-1 to LT-6
        2030,
        false
    FROM generate_series(1, 500) AS s(i);
END $$;
