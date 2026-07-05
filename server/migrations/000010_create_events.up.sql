CREATE TABLE events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id  UUID NOT NULL REFERENCES applications(id),
    experiment_id   UUID REFERENCES experiments(id),
    branch_id       UUID REFERENCES branches(id),
    user_id         TEXT NOT NULL,
    event_name      TEXT NOT NULL,
    properties_json JSONB,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX events_application_id_occurred_at_idx ON events (application_id, occurred_at DESC);
CREATE INDEX events_experiment_id_occurred_at_idx ON events (experiment_id, occurred_at DESC);
CREATE INDEX events_experiment_id_event_name_branch_id_idx ON events (experiment_id, event_name, branch_id);
