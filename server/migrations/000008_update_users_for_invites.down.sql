ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_status_check;

UPDATE users
SET status = 'inactive'
WHERE status = 'invited';

UPDATE users
SET password_hash = ''
WHERE password_hash IS NULL;

ALTER TABLE users
    ALTER COLUMN password_hash SET NOT NULL;

ALTER TABLE users
    ADD CONSTRAINT users_status_check
    CHECK (status IN ('active', 'inactive'));
