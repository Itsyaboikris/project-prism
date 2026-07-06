CREATE TABLE tracked_events (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    experiment_id  UUID NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
    key            TEXT NOT NULL,
    name           TEXT NOT NULL,
    description    TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ
);

CREATE UNIQUE INDEX tracked_events_experiment_key_active_idx
    ON tracked_events (experiment_id, key)
    WHERE deleted_at IS NULL;

CREATE INDEX tracked_events_experiment_id_idx
    ON tracked_events (experiment_id)
    WHERE deleted_at IS NULL;
