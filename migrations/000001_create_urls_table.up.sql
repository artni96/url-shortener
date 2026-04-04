CREATE TABLE urls (
    id SERIAL PRIMARY KEY,
    original_url VARCHAR(255) NOT NULL UNIQUE,
    short_url VARCHAR(10) NOT NULL
);

CREATE INDEX idx_short_url on urls(short_url);