CREATE TABLE assignments (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID        NOT NULL REFERENCES applications(id),
    experiment_id  UUID        NOT NULL REFERENCES experiments(id),
    branch_id      UUID        NOT NULL REFERENCES branches(id),
    user_id        TEXT        NOT NULL,
    assigned_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    context_json   JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (experiment_id, user_id)
);

CREATE INDEX assignments_experiment_id_user_id_idx ON assignments (experiment_id, user_id);
CREATE INDEX assignments_application_id_idx ON assignments (application_id);
