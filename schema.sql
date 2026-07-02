CREATE TABLE IF NOT EXISTS services (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    url TEXT NOT NULL,
    is_active BOOLEAN DEFAULT 1 NOT NULL,
    interval_seconds INTEGER DEFAULT 60 NOT NULL,
    expected_status INTEGER DEFAULT 200 NOT NULL
);

CREATE TABLE IF NOT EXISTS pings (
    id INTEGER PRIMARY KEY,
    service_id INTEGER NOT NULL,
    status_code INTEGER NOT NULL,
    expected_status_code INTEGER NOT NULL,
    latency_ms INTEGER NOT NULL,
    is_up BOOLEAN NOT NULL,
    error_message TEXT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY(service_id) REFERENCES services(id) ON DELETE CASCADE
);
