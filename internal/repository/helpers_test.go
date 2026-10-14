package repository_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"metrics-service/internal/config"
	"metrics-service/internal/repository"
	"metrics-service/internal/storage"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

var testDB *pgxpool.Pool

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	ctx := context.Background()
	dsn := config.Load().TestDatabaseURL

	if err := prepareTestDatabase(ctx, dsn); err != nil {
		fmt.Fprintf(os.Stderr, "prepare test database: %v\n", err)
		return 1
	}

	db, err := storage.NewPool(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect to test database: %v\n", err)
		return 1
	}
	defer db.Close()

	testDB = db
	return m.Run()
}

func prepareTestDatabase(ctx context.Context, dsn string) error {
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

	truncate(t)
	t.Cleanup(func() { truncate(t) })

	return repository.NewMetricRepository(testDB)
}

func truncate(t *testing.T) {
	t.Helper()

	_, err := testDB.Exec(context.Background(), "TRUNCATE metrics RESTART IDENTITY")
	require.NoError(t, err)
}
