-- name: GetFavicons :many
SELECT host, svg, png, ico
FROM favicons;

-- name: UpdateFaviconCache :exec
REPLACE
INTO favicons (host, svg, png, ico)
VALUES (?, ?, ?, ?);
