CREATE TABLE IF NOT EXISTS favicons
(
    id   INTEGER PRIMARY KEY AUTO_INCREMENT,
    host TEXT NOT NULL,
    svg  TEXT,
    png  TEXT,
    ico  TEXT
);
