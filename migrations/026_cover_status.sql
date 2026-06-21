ALTER TABLE buecher_titel ADD COLUMN cover_status VARCHAR(50) DEFAULT 'PENDING';

UPDATE buecher_titel 
SET cover_status = 'FOUND' 
WHERE cover_url IS NOT NULL AND cover_url != '';
