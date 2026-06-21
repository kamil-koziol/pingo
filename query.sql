-- name: UpsertService :one
INSERT INTO services (name, url, interval_seconds, expected_status)
VALUES (?, ?, ?, ?)
ON CONFLICT(name) DO UPDATE SET 
    url=excluded.url,
    interval_seconds=excluded.interval_seconds,
    expected_status=excluded.expected_status
RETURNING *;

-- name: ListServices :many
SELECT * FROM services;

-- name: ListActiveServices :many
SELECT * FROM services WHERE is_active=true;

-- name: GetServiceByID :one
SELECT * FROM services WHERE id=? LIMIT 1;

-- name: GetServiceByName :one
SELECT * FROM services WHERE name=? LIMIT 1;

-- name: CreatePing :exec
INSERT INTO pings (service_id, status_code, latency_ms, is_up, error_message, timestamp)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetPing :one
SELECT * FROM pings WHERE id=? LIMIT 1;

-- name: ListPings :many
SELECT * FROM pings
WHERE (sqlc.narg(is_up) IS NULL OR is_up = sqlc.narg(is_up)) AND
(sqlc.narg(service_id) IS NULL OR service_id = sqlc.narg(service_id));

