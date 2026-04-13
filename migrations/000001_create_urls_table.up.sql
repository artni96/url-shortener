CREATE TABLE urls (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    original_url VARCHAR(255) NOT NULL,
    short_url VARCHAR(10) NOT NULL
);

CREATE INDEX idx_short_url on urls(short_url);
CREATE UNIQUE INDEX idx_original_url_unique on urls(original_url);
