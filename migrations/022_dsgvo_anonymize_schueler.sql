-- +goose Up
-- +goose StatementBegin
ALTER TABLE schueler ADD COLUMN IF NOT EXISTS anonymized_at TIMESTAMP;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE schueler DROP COLUMN IF EXISTS anonymized_at;
-- +goose StatementEnd
