CREATE TABLE IF NOT EXISTS routes
(
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    source       TEXT,
    pre          INTEGER,
    destination  TEXT,
    abs          INTEGER,
    cors         INTEGER,
    secure_mode  INTEGER,
    forward_host INTEGER,
    forward_addr INTEGER,
    ignore_cert  INTEGER,
    active       INTEGER DEFAULT 1
);
