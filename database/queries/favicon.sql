-- name: GetFavicons :many
SELECT host, svg, png, ico
FROM favicons;

-- name: UpdateFaviconCache :exec
INSERT OR
REPLACE INTO favicons (host, svg, png, ico)
VALUES (?, ?, ?, ?);
