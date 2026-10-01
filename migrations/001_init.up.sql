CREATE TABLE monitors (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    interval_seconds INTEGER NOT NULL CHECK (interval_seconds > 0),
    timeout_seconds INTEGER NOT NULL CHECK (timeout_seconds > 0),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    CHECK (timeout_seconds < interval_seconds)
);

CREATE TABLE checks (
    id UUID PRIMARY KEY,
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    status VARCHAR(10) NOT NULL CHECK (status IN ('up', 'down')),
    status_code INTEGER NULL,
    latency_ms BIGINT NOT NULL,
    error TEXT NULL,
    checked_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_checks_monitor_checked_at
ON checks (monitor_id, checked_at DESC);
