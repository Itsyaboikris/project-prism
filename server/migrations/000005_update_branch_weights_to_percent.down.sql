ALTER TABLE branches
    DROP CONSTRAINT IF EXISTS branches_weight_check;

UPDATE branches
SET weight = weight / 100;

ALTER TABLE branches
    ALTER COLUMN weight TYPE NUMERIC(5, 4);

ALTER TABLE branches
    ADD CONSTRAINT branches_weight_check CHECK (weight >= 0 AND weight <= 1);
