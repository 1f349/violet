CREATE TABLE IF NOT EXISTS routes
(
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source      TEXT UNIQUE,
    destination TEXT,
    description TEXT,
    flags       INTEGER DEFAULT 0,
    active      INTEGER DEFAULT 1
);

CREATE TABLE IF NOT EXISTS redirects
(
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source      TEXT UNIQUE,
    destination TEXT,
    description TEXT,
    flags       INTEGER DEFAULT 0,
    code        INTEGER DEFAULT 0,
    active      INTEGER DEFAULT 1
);
