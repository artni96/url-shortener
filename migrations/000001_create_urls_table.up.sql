CREATE TABLE users (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    ip VARCHAR(45) NOT NULL
);

CREATE UNIQUE INDEX ixd_user_ip_unique on users(ip);
CREATE INDEX idx_users_id on users(id);

CREATE TABLE IF NOT EXISTS urls (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    original_url VARCHAR(255) NOT NULL,
    short_url VARCHAR(10) NOT NULL,
    created_by INTEGER,
    is_deleted BOOL
);

CREATE INDEX idx_short_url on urls(short_url);
CREATE UNIQUE INDEX idx_original_url_unique on urls(original_url);
