CREATE TABLE urls (
    id SERIAL PRIMARY KEY,
    originalURL VARCHAR(255) NOT NULL,
    shortURL VARCHAR(10) NOT NULL
);

CREATE INDEX idx_short_url on urls(shortURL);