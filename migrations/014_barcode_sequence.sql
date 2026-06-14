CREATE SEQUENCE IF NOT EXISTS barcode_seq START 10000;
SELECT setval('barcode_seq', (
    SELECT COALESCE(MAX(CAST(SUBSTRING(barcode_id FROM '^B-([0-9]+)$') AS INTEGER)), 10000)
    FROM buecher_exemplare
    WHERE barcode_id ~ '^B-[0-9]+$'
));
