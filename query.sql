-- name: UpsertService :one
INSERT INTO services (name, url, interval_seconds, expected_status)
VALUES (?, ?, ?, ?)
ON CONFLICT(name) DO UPDATE SET 
    url=excluded.url,
    interval_seconds=excluded.interval_seconds,
    expected_status=excluded.expected_status
RETURNING *;

-- name: CreatePing :exec
INSERT INTO pings (service_id, status_code, latency_ms, is_up, error_message, timestamp)
VALUES (?, ?, ?, ?, ?, ?);
