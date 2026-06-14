-- 018_vormerkungen_status.sql

ALTER TABLE vormerkungen 
ADD COLUMN status VARCHAR(50) DEFAULT 'wartend' NOT NULL;

ALTER TABLE vormerkungen 
ADD COLUMN bereitgestellt_exemplar_id UUID REFERENCES buecher_exemplare(id) ON DELETE SET NULL;
