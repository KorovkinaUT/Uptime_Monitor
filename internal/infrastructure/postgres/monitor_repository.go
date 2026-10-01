package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ulyana/uptime-monitor/internal/application"
	"github.com/ulyana/uptime-monitor/internal/domain"
)

type MonitorRepository struct {
	pool *pgxpool.Pool
}

func NewMonitorRepository(pool *pgxpool.Pool) *MonitorRepository {
	return &MonitorRepository{pool: pool}
}

func (r *MonitorRepository) Create(ctx context.Context, monitor domain.Monitor) error {
	const query = `
INSERT INTO monitors (id, name, url, interval_seconds, timeout_seconds, enabled)
VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.pool.Exec(ctx, query, monitor.ID, monitor.Name, monitor.URL, monitor.IntervalSeconds, monitor.TimeoutSeconds, monitor.Enabled)
	if err != nil {
		return fmt.Errorf("insert monitor: %w", err)
	}
	return nil
}

func (r *MonitorRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Monitor, error) {
	const query = `
SELECT id, name, url, interval_seconds, timeout_seconds, enabled
FROM monitors
WHERE id = $1`

	var monitor domain.Monitor
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&monitor.ID,
		&monitor.Name,
		&monitor.URL,
		&monitor.IntervalSeconds,
		&monitor.TimeoutSeconds,
		&monitor.Enabled,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Monitor{}, domain.ErrMonitorNotFound
	}
	if err != nil {
		return domain.Monitor{}, fmt.Errorf("get monitor: %w", err)
	}
	return monitor, nil
}

func (r *MonitorRepository) ListWithLastCheck(ctx context.Context) ([]application.MonitorWithLastCheck, error) {
	// Fetch the latest check in one query to avoid N+1 reads
	const query = `
SELECT
	m.id,
	m.name,
	m.url,
	m.interval_seconds,
	m.timeout_seconds,
	m.enabled,
	latest.status,
	latest.status_code,
	latest.latency_ms,
	latest.error,
	latest.checked_at
FROM monitors AS m
LEFT JOIN LATERAL (
	SELECT status, status_code, latency_ms, error, checked_at
	FROM checks
	WHERE monitor_id = m.id
	ORDER BY checked_at DESC
	LIMIT 1
) AS latest ON TRUE
ORDER BY m.name, m.id`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list monitors: %w", err)
	}
	defer rows.Close()

	monitors := make([]application.MonitorWithLastCheck, 0)
	for rows.Next() {
		var (
			monitor    domain.Monitor
			status     pgtype.Text
			statusCode pgtype.Int4
			latencyMS  pgtype.Int8
			checkError pgtype.Text
			checkedAt  pgtype.Timestamptz
		)
		if err := rows.Scan(
			&monitor.ID,
			&monitor.Name,
			&monitor.URL,
			&monitor.IntervalSeconds,
			&monitor.TimeoutSeconds,
			&monitor.Enabled,
			&status,
			&statusCode,
			&latencyMS,
			&checkError,
			&checkedAt,
		); err != nil {
			return nil, fmt.Errorf("scan monitor: %w", err)
		}

		result := application.MonitorWithLastCheck{Monitor: monitor}
		if status.Valid {
			check := domain.Check{
				MonitorID: monitor.ID,
				Status:    domain.CheckStatus(status.String),
			}
			if statusCode.Valid {
				value := int(statusCode.Int32)
				check.StatusCode = &value
			}
			if latencyMS.Valid {
				check.LatencyMS = latencyMS.Int64
			}
			if checkError.Valid {
				value := checkError.String
				check.Error = &value
			}
			if checkedAt.Valid {
				check.CheckedAt = checkedAt.Time
			}
			result.LastCheck = &check
		}
		monitors = append(monitors, result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate monitors: %w", err)
	}

	return monitors, nil
}

func (r *MonitorRepository) Update(ctx context.Context, monitor domain.Monitor) error {
	const query = `
UPDATE monitors
SET name = $2, url = $3, interval_seconds = $4, timeout_seconds = $5, enabled = $6
WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query, monitor.ID, monitor.Name, monitor.URL, monitor.IntervalSeconds, monitor.TimeoutSeconds, monitor.Enabled)
	if err != nil {
		return fmt.Errorf("update monitor: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrMonitorNotFound
	}
	return nil
}

func (r *MonitorRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM monitors WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete monitor: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrMonitorNotFound
	}
	return nil
}

func (r *MonitorRepository) ListDue(ctx context.Context, now time.Time) ([]domain.Monitor, error) {
	// Select only enabled monitors whose latest check is overdue
	const query = `
SELECT m.id, m.name, m.url, m.interval_seconds, m.timeout_seconds, m.enabled
FROM monitors AS m
LEFT JOIN LATERAL (
	SELECT checked_at
	FROM checks
	WHERE monitor_id = m.id
	ORDER BY checked_at DESC
	LIMIT 1
) AS latest ON TRUE
WHERE m.enabled = TRUE
  AND (
		latest.checked_at IS NULL
		OR latest.checked_at + (m.interval_seconds * INTERVAL '1 second') <= $1
	)
ORDER BY m.id`

	rows, err := r.pool.Query(ctx, query, now)
	if err != nil {
		return nil, fmt.Errorf("list due monitors: %w", err)
	}
	defer rows.Close()

	monitors := make([]domain.Monitor, 0)
	for rows.Next() {
		var monitor domain.Monitor
		if err := rows.Scan(
			&monitor.ID,
			&monitor.Name,
			&monitor.URL,
			&monitor.IntervalSeconds,
			&monitor.TimeoutSeconds,
			&monitor.Enabled,
		); err != nil {
			return nil, fmt.Errorf("scan due monitor: %w", err)
		}
		monitors = append(monitors, monitor)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate due monitors: %w", err)
	}

	return monitors, nil
}
