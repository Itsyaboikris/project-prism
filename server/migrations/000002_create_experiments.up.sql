CREATE TABLE experiments (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID        NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    key            TEXT        NOT NULL,
    name           TEXT        NOT NULL,
    description    TEXT,
    status         TEXT        NOT NULL DEFAULT 'draft'
                               CHECK (status IN ('draft', 'active', 'paused', 'completed')),
    start_date     TIMESTAMPTZ,
    end_date       TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ
);

CREATE UNIQUE INDEX experiments_application_key_active_idx ON experiments (application_id, key) WHERE deleted_at IS NULL;
CREATE INDEX experiments_application_id_idx ON experiments (application_id) WHERE deleted_at IS NULL;
