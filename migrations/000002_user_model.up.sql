CREATE TABLE users (
       id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
       username VARCHAR(128) NOT NULL,
       password VARCHAR(44) NOT NULL
);

CREATE INDEX idx_user_id on users(id);
CREATE UNIQUE INDEX idx_username_unique on users(username);

ALTER TABLE urls
    ADD COLUMN created_by INTEGER,
    ADD CONSTRAINT fk_created_by
        FOREIGN KEY(created_by)
        REFERENCES users(id)
        ON DELETE CASCADE;