-- Migration 036: Add configurable school identity settings for PDF letter headers.
-- Values are set by the operator via the settings UI; defaults are empty strings
-- so PDFs remain functional before configuration (header shows "Schulbibliothek").
INSERT INTO system_einstellungen (schluessel, wert) VALUES
    ('schule_name',    ''),
    ('schule_strasse', ''),
    ('schule_plz',     ''),
    ('schule_ort',     '')
ON CONFLICT (schluessel) DO NOTHING;
