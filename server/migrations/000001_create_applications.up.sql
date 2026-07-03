CREATE TABLE applications (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL,
    api_key    TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX applications_api_key_active_idx ON applications (api_key) WHERE deleted_at IS NULL;
CREATE INDEX applications_active_idx ON applications (created_at DESC) WHERE deleted_at IS NULL;
