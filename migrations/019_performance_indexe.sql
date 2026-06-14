-- Beschleunigt das Filtern von Klassensätzen und Mahnungen
CREATE INDEX IF NOT EXISTS idx_schueler_klasse ON schueler(klasse);

-- Beschleunigt den Vormerkung-Alarm beim Check-in
CREATE INDEX IF NOT EXISTS idx_vormerkungen_status ON vormerkungen(status);
