-- Migration 021: Signatures MDM

-- 1. Create table for signatures
CREATE TABLE IF NOT EXISTS signatures (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT
);

-- 2. Add signature_id column to buecher_titel
ALTER TABLE buecher_titel 
ADD COLUMN IF NOT EXISTS signature_id INT REFERENCES signatures(id) ON DELETE RESTRICT;

-- 3. Extract existing unique subjects and insert into signatures
INSERT INTO signatures (name)
SELECT DISTINCT subject 
FROM buecher_titel 
WHERE subject IS NOT NULL AND TRIM(subject) != ''
ON CONFLICT (name) DO NOTHING;

-- 4. Update the foreign key
UPDATE buecher_titel bt
SET signature_id = s.id
FROM signatures s
WHERE bt.subject = s.name;
