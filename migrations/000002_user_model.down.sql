ALTER TABLE urls
    DROP CONSTRAINT fk_created_by,
    DROP COLUMN created_by;

DROP INDEX IF EXISTS idx_user_id;
DROP INDEX IF EXISTS idx_username_unique;
DROP TABLE users;
