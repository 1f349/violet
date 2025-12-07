CREATE TABLE IF NOT EXISTS routes
(
    id          INTEGER PRIMARY KEY AUTO_INCREMENT,
    source      TEXT UNIQUE NOT NULL,
    destination TEXT        NOT NULL,
    description TEXT        NOT NULL,
    flags       INTEGER     NOT NULL DEFAULT 0,
    active      BOOLEAN     NOT NULL DEFAULT 1
);
