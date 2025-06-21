-- name: GetActiveDomains :many
SELECT domain
FROM domains
WHERE active = 1;

-- name: AddDomain :exec
REPLACE
INTO domains (domain, active)
VALUES (?, ?);

-- name: DeleteDomain :exec
REPLACE
INTO domains(domain, active)
VALUES (?, false);
