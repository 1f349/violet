-- name: GetActiveRoutes :many
SELECT source, destination, flags
FROM routes
WHERE active = 1;

-- name: GetActiveRedirects :many
SELECT source, destination, flags, code
FROM redirects
WHERE active = 1;

-- name: GetAllRoutes :many
SELECT source, destination, description, flags, active
FROM routes;

-- name: GetAllRedirects :many
SELECT source, destination, description, flags, code, active
FROM redirects;

-- name: AddRoute :exec
INSERT OR
REPLACE
INTO routes (source, destination, description, flags, active)
VALUES (?, ?, ?, ?, ?);

-- name: AddRedirect :exec
INSERT OR
REPLACE
INTO redirects (source, destination, description, flags, code, active)
VALUES (?, ?, ?, ?, ?, ?);

-- name: RemoveRoute :exec
DELETE
FROM routes
WHERE source = ?;

-- name: RemoveRedirect :exec
DELETE
FROM redirects
WHERE source = ?;
