CREATE TABLE IF NOT EXISTS redirects
(
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source      TEXT,
    pre         INTEGER,
    destination TEXT,
    abs         INTEGER,
    code        INTEGER,
    active      INTEGER DEFAULT 1
);
