CREATE TABLE users (
   id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
   username VARCHAR(128) NOT NULL,
   password VARCHAR(44) NOT NULL
);

CREATE INDEX idx_user_id on users(id);
CREATE UNIQUE INDEX idx_username_unique on users(username);

CREATE TABLE urls (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    original_url VARCHAR(255) NOT NULL,
    short_url VARCHAR(10) NOT NULL,
    created_by INTEGER,
    CONSTRAINT fk_created_by
        FOREIGN KEY(created_by)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_short_url on urls(short_url);
CREATE UNIQUE INDEX idx_original_url_unique on urls(original_url);
