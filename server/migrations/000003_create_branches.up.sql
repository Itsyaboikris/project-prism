CREATE TABLE branches (
    id            UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    experiment_id UUID          NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
    key           TEXT          NOT NULL,
    name          TEXT          NOT NULL,
    weight        NUMERIC(5, 4) NOT NULL CHECK (weight >= 0 AND weight <= 1),
    metadata_json JSONB,
    UNIQUE (experiment_id, key)
);

CREATE INDEX branches_experiment_id_idx ON branches (experiment_id);
