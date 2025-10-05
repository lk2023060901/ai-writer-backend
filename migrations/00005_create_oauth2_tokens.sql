-- +goose Up
CREATE TABLE IF NOT EXISTS oauth2_tokens (
    id SERIAL PRIMARY KEY,
    provider VARCHAR(50) NOT NULL UNIQUE,
    access_token TEXT NOT NULL,
    token_type VARCHAR(50),
    refresh_token TEXT,
    expiry TIMESTAMP,
    token_json JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_oauth2_tokens_provider ON oauth2_tokens(provider);
CREATE INDEX IF NOT EXISTS idx_oauth2_tokens_expiry ON oauth2_tokens(expiry);

COMMENT ON TABLE oauth2_tokens IS 'OAuth2 tokens storage';
COMMENT ON COLUMN oauth2_tokens.provider IS 'OAuth2 provider identifier (e.g., gmail)';
COMMENT ON COLUMN oauth2_tokens.access_token IS 'Access token';
COMMENT ON COLUMN oauth2_tokens.refresh_token IS 'Refresh token';
COMMENT ON COLUMN oauth2_tokens.token_json IS 'Complete token JSON';

-- +goose Down
DROP TABLE IF EXISTS oauth2_tokens;
