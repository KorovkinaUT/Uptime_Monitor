package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ulyana/uptime-monitor/internal/application"
	"github.com/ulyana/uptime-monitor/internal/domain"
)

// CheckRepository stores check results in PostgreSQL
type CheckRepository struct {
	pool *pgxpool.Pool
}

var _ application.CheckRepository = (*CheckRepository)(nil)

func NewCheckRepository(pool *pgxpool.Pool) *CheckRepository {
	return &CheckRepository{pool: pool}
}

// Create saves one check result
func (r *CheckRepository) Create(ctx context.Context, check domain.Check) error {
	const query = `
		INSERT INTO checks (id, monitor_id, status, status_code, latency_ms, error, checked_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.pool.Exec(ctx, query, check.ID, check.MonitorID, check.Status, check.StatusCode, check.LatencyMS, check.Error, check.CheckedAt)
	if err != nil {
		return fmt.Errorf("insert check: %w", err)
	}
	return nil
}

// ListByMonitorID returns recent checks for a monitor
func (r *CheckRepository) ListByMonitorID(ctx context.Context, monitorID uuid.UUID, limit int) ([]domain.Check, error) {
	const query = `
		SELECT id, monitor_id, status, status_code, latency_ms, error, checked_at
		FROM checks
		WHERE monitor_id = $1
		ORDER BY checked_at DESC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, query, monitorID, limit)
	if err != nil {
		return nil, fmt.Errorf("list checks: %w", err)
	}
	defer rows.Close()

	checks := make([]domain.Check, 0)
	for rows.Next() {
		var (
			check      domain.Check
			statusCode pgtype.Int4
			checkError pgtype.Text
		)
		if err := rows.Scan(
			&check.ID,
			&check.MonitorID,
			&check.Status,
			&statusCode,
			&check.LatencyMS,
			&checkError,
			&check.CheckedAt,
		); err != nil {
			return nil, fmt.Errorf("scan check: %w", err)
		}
		if statusCode.Valid {
			check.StatusCode = new(int(statusCode.Int32))
		}
		if checkError.Valid {
			check.Error = new(checkError.String)
		}
		checks = append(checks, check)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate checks: %w", err)
	}

	return checks, nil
}
