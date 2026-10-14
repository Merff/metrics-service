package repository_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"metrics-service/internal/repository"
	"metrics-service/internal/storage"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

const defaultTestDatabaseURL = "postgres://postgres:postgres@localhost:5433/metrics_test?sslmode=disable"

var testDatabaseURL string

func TestMain(m *testing.M) {
	testDatabaseURL = os.Getenv("TEST_DATABASE_URL")
	if testDatabaseURL == "" {
		testDatabaseURL = defaultTestDatabaseURL
	}

	if err := prepareTestDatabase(testDatabaseURL); err != nil {
		fmt.Fprintf(os.Stderr, "prepare test database: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func prepareTestDatabase(dsn string) error {
	ctx := context.Background()

	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return err
	}

	name := cfg.Database
	if !strings.HasSuffix(name, "_test") {
		return fmt.Errorf("database %q: name must end with _test", name)
	}

	cfg.Database = "postgres"
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connect to server: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	var exists bool
	err = conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", name).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check database: %w", err)
	}

	if !exists {
		if _, err := conn.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
			return fmt.Errorf("create database: %w", err)
		}
	}

	return storage.MigrateUp(dsn)
}

func newTestRepo(t *testing.T) *repository.MetricRepository {
	t.Helper()

	ctx := context.Background()

	db, err := storage.NewPool(ctx, testDatabaseURL)
	require.NoError(t, err)

	var name string
	require.NoError(t, db.QueryRow(ctx, "SELECT current_database()").Scan(&name))
	if !strings.HasSuffix(name, "_test") {
		db.Close()
		t.Fatalf("refusing to truncate database %q: name must end with _test", name)
	}

	truncate(t, db)
	t.Cleanup(func() {
		truncate(t, db)
		db.Close()
	})

	return repository.NewMetricRepository(db)
}

func truncate(t *testing.T, db *pgxpool.Pool) {
	t.Helper()

	_, err := db.Exec(context.Background(), "TRUNCATE metrics RESTART IDENTITY")
	require.NoError(t, err)
}
