-- 022_vormerkungen_ablauf.sql

ALTER TABLE vormerkungen 
ADD COLUMN bereitgestellt_bis TIMESTAMP WITH TIME ZONE;
