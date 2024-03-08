CREATE TABLE IF NOT EXISTS domains
(
    id     INTEGER PRIMARY KEY AUTOINCREMENT,
    domain TEXT UNIQUE NOT NULL,
    active BOOLEAN     NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS favicons
(
    id   INTEGER PRIMARY KEY AUTOINCREMENT,
    host VARCHAR NOT NULL,
    svg  VARCHAR,
    png  VARCHAR,
    ico  VARCHAR
);

CREATE TABLE IF NOT EXISTS routes
(
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source      TEXT UNIQUE NOT NULL,
    destination TEXT        NOT NULL,
    description TEXT        NOT NULL,
    flags       INTEGER     NOT NULL DEFAULT 0,
    active      BOOLEAN     NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS redirects
(
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source      TEXT UNIQUE NOT NULL,
    destination TEXT        NOT NULL,
    description TEXT        NOT NULL,
    flags       INTEGER     NOT NULL DEFAULT 0,
    code        INTEGER     NOT NULL DEFAULT 0,
    active      BOOLEAN     NOT NULL DEFAULT 1
);
