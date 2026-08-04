-- +goose Up
-- +goose StatementBegin
CREATE TABLE oauth_login_states (
    state         TEXT PRIMARY KEY,
    nonce         TEXT NOT NULL,
    code_verifier TEXT NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX oauth_login_states_expires_at_idx ON oauth_login_states (expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS oauth_login_states_expires_at_idx;
DROP TABLE IF EXISTS oauth_login_states;
-- +goose StatementEnd
