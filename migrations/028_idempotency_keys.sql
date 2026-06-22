-- idempotency_keys
CREATE TABLE IF NOT EXISTS idempotency_keys (
    idempotency_key UUID PRIMARY KEY,
    response_data JSONB NOT NULL,
    status_code INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_idempotency_keys_created_at ON idempotency_keys(created_at);
