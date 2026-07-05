CREATE TABLE invitation_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT        NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX invitation_tokens_active_user_idx
    ON invitation_tokens (user_id)
    WHERE consumed_at IS NULL;

CREATE INDEX invitation_tokens_expires_at_idx
    ON invitation_tokens (expires_at);
