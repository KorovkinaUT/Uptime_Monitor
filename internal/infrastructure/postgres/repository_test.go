package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/ulyana/uptime-monitor/internal/domain"
)

func newRepositoryMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("NewPool() error = %v", err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("database expectations error = %v", err)
		}
		mock.Close()
	})
	return mock
}

func TestCheckRepositoryCreate(t *testing.T) {
	//arrange
	mock := newRepositoryMock(t)
	repository := NewCheckRepository(mock)
	check := domain.Check{
		ID:        uuid.New(),
		MonitorID: uuid.New(),
		Status:    domain.CheckStatusUp,
		LatencyMS: 42,
		CheckedAt: time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC),
	}
	mock.ExpectExec(`
		INSERT INTO checks (id, monitor_id, status, status_code, latency_ms, error, checked_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`).
		WithArgs(check.ID, check.MonitorID, check.Status, check.StatusCode, check.LatencyMS, check.Error, check.CheckedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	//act
	err := repository.Create(context.Background(), check)

	//assert
	if err != nil {
		t.Errorf("Create() error = %v, want nil", err)
	}
}

func TestCheckRepositoryListByMonitorID(t *testing.T) {
	//arrange
	mock := newRepositoryMock(t)
	repository := NewCheckRepository(mock)
	monitorID := uuid.New()
	checkID := uuid.New()
	checkedAt := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`
		SELECT id, monitor_id, status, status_code, latency_ms, error, checked_at
		FROM checks
		WHERE monitor_id = $1
		ORDER BY checked_at DESC
		LIMIT $2`).
		WithArgs(monitorID, 100).
		WillReturnRows(mock.NewRows([]string{
			"id", "monitor_id", "status", "status_code", "latency_ms", "error", "checked_at",
		}).AddRow(checkID, monitorID, "down", int64(500), int64(42), "server error", checkedAt))

	//act
	checks, err := repository.ListByMonitorID(context.Background(), monitorID, 100)

	//assert
	if err != nil {
		t.Errorf("ListByMonitorID() error = %v", err)
		return
	}
	if len(checks) != 1 {
		t.Errorf("checks length = %d, want 1", len(checks))
		return
	}
	if checks[0].ID != checkID || checks[0].Status != domain.CheckStatusDown {
		t.Errorf("check identity = %+v, want id %s and down status", checks[0], checkID)
	}
	if checks[0].StatusCode == nil || *checks[0].StatusCode != 500 {
		t.Errorf("status code = %v, want 500", checks[0].StatusCode)
	}
	if checks[0].Error == nil || *checks[0].Error != "server error" {
		t.Errorf("error = %v, want server error", checks[0].Error)
	}
}

func TestMonitorRepositoryCreateReturnsSavedMonitor(t *testing.T) {
	//arrange
	mock := newRepositoryMock(t)
	repository := NewMonitorRepository(mock)
	monitor := domain.Monitor{
		ID:              uuid.New(),
		Name:            "API",
		URL:             "https://example.com",
		IntervalSeconds: 30,
		TimeoutSeconds:  5,
		Enabled:         true,
	}
	mock.ExpectQuery(`
		INSERT INTO monitors (id, name, url, interval_seconds, timeout_seconds, enabled)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, url, interval_seconds, timeout_seconds, enabled`).
		WithArgs(monitor.ID, monitor.Name, monitor.URL, monitor.IntervalSeconds, monitor.TimeoutSeconds, monitor.Enabled).
		WillReturnRows(mock.NewRows([]string{
			"id", "name", "url", "interval_seconds", "timeout_seconds", "enabled",
		}).AddRow(monitor.ID, monitor.Name, monitor.URL, monitor.IntervalSeconds, monitor.TimeoutSeconds, monitor.Enabled))

	//act
	saved, err := repository.Create(context.Background(), monitor)

	//assert
	if err != nil {
		t.Errorf("Create() error = %v", err)
		return
	}
	if saved != monitor {
		t.Errorf("saved monitor = %+v, want %+v", saved, monitor)
	}
}

func TestMonitorRepositoryGetByID(t *testing.T) {
	//arrange
	mock := newRepositoryMock(t)
	repository := NewMonitorRepository(mock)
	monitor := domain.Monitor{
		ID:              uuid.New(),
		Name:            "API",
		URL:             "https://example.com",
		IntervalSeconds: 30,
		TimeoutSeconds:  5,
		Enabled:         true,
	}
	mock.ExpectQuery(`
		SELECT id, name, url, interval_seconds, timeout_seconds, enabled
		FROM monitors
		WHERE id = $1`).
		WithArgs(monitor.ID).
		WillReturnRows(mock.NewRows([]string{
			"id", "name", "url", "interval_seconds", "timeout_seconds", "enabled",
		}).AddRow(monitor.ID, monitor.Name, monitor.URL, monitor.IntervalSeconds, monitor.TimeoutSeconds, monitor.Enabled))

	//act
	got, err := repository.GetByID(context.Background(), monitor.ID)

	//assert
	if err != nil {
		t.Errorf("GetByID() error = %v", err)
		return
	}
	if got != monitor {
		t.Errorf("monitor = %+v, want %+v", got, monitor)
	}
}

func TestMonitorRepositoryGetByIDNotFound(t *testing.T) {
	//arrange
	mock := newRepositoryMock(t)
	repository := NewMonitorRepository(mock)
	monitorID := uuid.New()
	mock.ExpectQuery(`
		SELECT id, name, url, interval_seconds, timeout_seconds, enabled
		FROM monitors
		WHERE id = $1`).
		WithArgs(monitorID).
		WillReturnError(pgx.ErrNoRows)

	//act
	_, err := repository.GetByID(context.Background(), monitorID)

	//assert
	if !errors.Is(err, domain.ErrMonitorNotFound) {
		t.Errorf("GetByID() error = %v, want ErrMonitorNotFound", err)
	}
}

func TestMonitorRepositoryListWithLastCheck(t *testing.T) {
	//arrange
	mock := newRepositoryMock(t)
	repository := NewMonitorRepository(mock)
	monitor := domain.Monitor{
		ID:              uuid.New(),
		Name:            "API",
		URL:             "https://example.com",
		IntervalSeconds: 30,
		TimeoutSeconds:  5,
		Enabled:         true,
	}
	checkedAt := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`
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
		ORDER BY m.name, m.id`).
		WillReturnRows(mock.NewRows([]string{
			"id", "name", "url", "interval_seconds", "timeout_seconds", "enabled",
			"status", "status_code", "latency_ms", "error", "checked_at",
		}).AddRow(monitor.ID, monitor.Name, monitor.URL, monitor.IntervalSeconds, monitor.TimeoutSeconds, monitor.Enabled,
			"up", int64(200), int64(18), nil, checkedAt))

	//act
	views, err := repository.ListWithLastCheck(context.Background())

	//assert
	if err != nil {
		t.Errorf("ListWithLastCheck() error = %v", err)
		return
	}
	if len(views) != 1 {
		t.Errorf("views length = %d, want 1", len(views))
		return
	}
	if views[0].Monitor != monitor {
		t.Errorf("monitor = %+v, want %+v", views[0].Monitor, monitor)
	}
	if views[0].LastCheck == nil {
		t.Errorf("LastCheck = nil, want latest check")
		return
	}
	if views[0].LastCheck.Status != domain.CheckStatusUp || views[0].LastCheck.StatusCode == nil {
		t.Errorf("latest check = %+v, want up with status code", views[0].LastCheck)
	}
}

func TestMonitorRepositoryListDue(t *testing.T) {
	//arrange
	mock := newRepositoryMock(t)
	repository := NewMonitorRepository(mock)
	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	monitor := domain.Monitor{ID: uuid.New(), Name: "API", URL: "https://example.com", IntervalSeconds: 30, TimeoutSeconds: 5, Enabled: true}
	mock.ExpectQuery(`
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
		ORDER BY m.id`).
		WithArgs(now).
		WillReturnRows(mock.NewRows([]string{
			"id", "name", "url", "interval_seconds", "timeout_seconds", "enabled",
		}).AddRow(monitor.ID, monitor.Name, monitor.URL, monitor.IntervalSeconds, monitor.TimeoutSeconds, monitor.Enabled))

	//act
	monitors, err := repository.ListDue(context.Background(), now)

	//assert
	if err != nil {
		t.Errorf("ListDue() error = %v", err)
		return
	}
	if len(monitors) != 1 || monitors[0] != monitor {
		t.Errorf("monitors = %+v, want %+v", monitors, []domain.Monitor{monitor})
	}
}

func TestMonitorRepositoryUpdateAndDelete(t *testing.T) {
	//arrange
	mock := newRepositoryMock(t)
	repository := NewMonitorRepository(mock)
	monitor := domain.Monitor{ID: uuid.New(), Name: "API", URL: "https://example.com", IntervalSeconds: 30, TimeoutSeconds: 5, Enabled: true}
	mock.ExpectExec(`
		UPDATE monitors
		SET name = $2, url = $3, interval_seconds = $4, timeout_seconds = $5, enabled = $6
		WHERE id = $1`).
		WithArgs(monitor.ID, monitor.Name, monitor.URL, monitor.IntervalSeconds, monitor.TimeoutSeconds, monitor.Enabled).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`
		DELETE FROM monitors
		WHERE id = $1`).
		WithArgs(monitor.ID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	//act
	updateErr := repository.Update(context.Background(), monitor)
	deleteErr := repository.Delete(context.Background(), monitor.ID)

	//assert
	if updateErr != nil {
		t.Errorf("Update() error = %v, want nil", updateErr)
	}
	if deleteErr != nil {
		t.Errorf("Delete() error = %v, want nil", deleteErr)
	}
}
