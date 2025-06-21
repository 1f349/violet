CREATE TABLE IF NOT EXISTS domains
(
    id     INTEGER PRIMARY KEY AUTO_INCREMENT,
    domain TEXT UNIQUE NOT NULL,
    active BOOLEAN     NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS favicons
(
    id   INTEGER PRIMARY KEY AUTO_INCREMENT,
    host TEXT NOT NULL,
    svg  TEXT,
    png  TEXT,
    ico  TEXT
);

CREATE TABLE IF NOT EXISTS routes
(
    id          INTEGER PRIMARY KEY AUTO_INCREMENT,
    source      TEXT UNIQUE NOT NULL,
    destination TEXT        NOT NULL,
    description TEXT        NOT NULL,
    flags       INTEGER     NOT NULL DEFAULT 0,
    active      BOOLEAN     NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS redirects
(
    id          INTEGER PRIMARY KEY AUTO_INCREMENT,
    source      TEXT UNIQUE NOT NULL,
    destination TEXT        NOT NULL,
    description TEXT        NOT NULL,
    flags       INTEGER     NOT NULL DEFAULT 0,
    code        INTEGER     NOT NULL DEFAULT 0,
    active      BOOLEAN     NOT NULL DEFAULT 1
);
