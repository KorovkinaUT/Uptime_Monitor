package postgres

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/ulyana/uptime-monitor/internal/domain"
)

// testDatabase holds shared PostgreSQL resources
var testDatabase struct {
	once      sync.Once
	pool      *pgxpool.Pool
	container *postgres.PostgresContainer
	err       error
}

func TestMain(m *testing.M) {
	code := m.Run()

	if testDatabase.pool != nil {
		testDatabase.pool.Close()
	}
	if testDatabase.container != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_ = testDatabase.container.Terminate(ctx)
		cancel()
	}

	os.Exit(code)
}

// testPool starts PostgreSQL and applies the test schema
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	testDatabase.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		testDatabase.container, testDatabase.err = postgres.Run(
			ctx,
			"postgres:16-alpine",
			postgres.WithDatabase("uptime_test"),
			postgres.WithUsername("uptime"),
			postgres.WithPassword("uptime"),
		)
		if testDatabase.err != nil {
			return
		}

		connectionString, err := testDatabase.container.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			testDatabase.err = err
			return
		}

		testDatabase.pool, testDatabase.err = pgxpool.New(ctx, connectionString)
		if testDatabase.err != nil {
			return
		}
		if testDatabase.err = testDatabase.pool.Ping(ctx); testDatabase.err != nil {
			return
		}

		_, filename, _, _ := runtime.Caller(0)
		schemaPath := filepath.Join(filepath.Dir(filename), "../../../migrations/001_init.up.sql")
		schema, err := os.ReadFile(schemaPath)
		if err != nil {
			testDatabase.err = err
			return
		}
		_, testDatabase.err = testDatabase.pool.Exec(ctx, string(schema))
	})

	if testDatabase.err != nil {
		t.Skipf("PostgreSQL container unavailable: %v", testDatabase.err)
		return nil
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := testDatabase.pool.Exec(ctx, "TRUNCATE checks, monitors CASCADE"); err != nil {
			t.Errorf("truncate test database: %v", err)
		}
	})

	return testDatabase.pool
}

// createTestMonitor inserts a monitor for repository tests
func createTestMonitor(t *testing.T, pool *pgxpool.Pool) domain.Monitor {
	t.Helper()

	monitor := domain.Monitor{
		ID:              uuid.New(),
		Name:            "API",
		URL:             "https://example.com",
		IntervalSeconds: 30,
		TimeoutSeconds:  5,
		Enabled:         true,
	}
	saved, err := NewMonitorRepository(pool).Create(context.Background(), monitor)
	if err != nil {
		t.Fatalf("create test monitor: %v", err)
	}
	return saved
}

// insertTestCheck inserts a check for repository tests
func insertTestCheck(t *testing.T, pool *pgxpool.Pool, check domain.Check) {
	t.Helper()

	_, err := pool.Exec(
		context.Background(),
		`INSERT INTO checks (id, monitor_id, status, status_code, latency_ms, error, checked_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		check.ID,
		check.MonitorID,
		check.Status,
		check.StatusCode,
		check.LatencyMS,
		check.Error,
		check.CheckedAt,
	)
	if err != nil {
		t.Fatalf("insert test check: %v", err)
	}
}

// newTestUUID returns a unique identifier for test data
func newTestUUID() uuid.UUID {
	return uuid.New()
}
