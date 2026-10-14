package repository_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"metrics-service/internal/repository"
	"metrics-service/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

const defaultTestDatabaseURL = "postgres://postgres:postgres@localhost:5433/metrics_test?sslmode=disable"

func newTestRepo(t *testing.T) *repository.MetricRepository {
	t.Helper()

	ctx := context.Background()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = defaultTestDatabaseURL
	}

	db, err := storage.NewPool(ctx, dsn)
	require.NoError(t, err, "test database unavailable, run `make test-db`")

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
