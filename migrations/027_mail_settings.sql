CREATE TABLE IF NOT EXISTS mail_settings_config (
    id SERIAL PRIMARY KEY,
    smtp_host VARCHAR(255) NOT NULL DEFAULT 'localhost',
    smtp_port VARCHAR(50) NOT NULL DEFAULT '1025',
    smtp_user VARCHAR(255) NOT NULL DEFAULT '',
    smtp_password_encrypted BYTEA,
    sender_email VARCHAR(255) NOT NULL DEFAULT 'noreply@bibliothek-schule.de'
);

-- Ensure there's only one active configuration by restricting the ID to 1
CREATE UNIQUE INDEX IF NOT EXISTS mail_settings_config_single_row_idx ON mail_settings_config ((1));

-- Insert the default row if it does not exist
INSERT INTO mail_settings_config (id) VALUES (1) ON CONFLICT DO NOTHING;
