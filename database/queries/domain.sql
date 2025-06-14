-- name: GetActiveDomains :many
SELECT domain
FROM domains
WHERE active = 1;

-- name: AddDomain :exec
INSERT
OR
REPLACE
INTO domains (domain, active)
VALUES (?, ?);

-- name: DeleteDomain :exec
INSERT
OR
REPLACE
INTO domains(domain, active)
VALUES (?, false);

-- name: IsDomainActive :one
SELECT EXISTS(SELECT 1
              FROM domains
              WHERE domain = ? AND active = 1) AS active;
