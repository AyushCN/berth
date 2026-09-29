package migrations

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"sync/atomic"
	"testing"

	_ "github.com/lib/pq"
)

// The application had no migration runner at all before this, so nothing ever
// executed the SQL in CI or on a fresh deploy. These tests bring a real
// database up and down through the embedded files.
//
// They need a postgres they are allowed to create databases in. Set
// BERTH_MIGRATION_TEST_DSN to a server-level DSN, e.g.
//   postgres://berth:berth@localhost:5432/postgres?sslmode=disable
// Without it they skip, so `go test ./...` stays green offline.

func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("BERTH_MIGRATION_TEST_DSN")
	if dsn == "" {
		t.Skip("set BERTH_MIGRATION_TEST_DSN to run migration tests")
	}
	return dsn
}

// dbCounter keeps each test on its own database; a shared name would have one
// test's still-open connections block the next test's cleanup.
var dbCounter atomic.Int64

func freshDatabase(t *testing.T) string {
	t.Helper()
	base := testDSN(t)

	admin, err := sql.Open("postgres", base)
	if err != nil {
		t.Fatalf("open admin dsn: %v", err)
	}
	defer admin.Close()

	name := fmt.Sprintf("berth_migtest_%d_%d", os.Getpid(), dbCounter.Add(1))
	// Terminate anything still attached from a previous run of this test.
	_, _ = admin.Exec("select pg_terminate_backend(pid) from pg_stat_activity where datname = $1", name)
	if _, err := admin.Exec("DROP DATABASE IF EXISTS " + name); err != nil {
		t.Fatalf("drop stale test db: %v", err)
	}
	if _, err := admin.Exec("CREATE DATABASE " + name); err != nil {
		t.Fatalf("create test db: %v", err)
	}
	t.Cleanup(func() {
		conn, err := sql.Open("postgres", base)
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = conn.Exec("DROP DATABASE IF EXISTS " + name)
	})

	// Point the DSN at the new database, preserving the query string. Slicing
	// on the last '/' silently drops parameters such as sslmode, which makes
	// lib/pq default to requiring SSL.
	u, err := url.Parse(base)
	if err != nil {
		t.Fatalf("parse %s: %v", base, err)
	}
	u.Path = "/" + name
	return u.String()
}

// assertSchema checks the resulting schema through a plain SQL connection, so
// the assertions do not depend on the pgx pool the migrator avoids using.
func assertSchema(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open %s: %v", dsn, err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestMigrateEmptyDatabase is the important one: a fresh deployment used to
// start against an empty schema and fail every query.
func TestMigrateEmptyDatabase(t *testing.T) {
	dsn := freshDatabase(t)

	version, err := Migrate(dsn)
	if err != nil {
		t.Fatalf("Migrate on an empty database: %v", err)
	}
	if version == 0 {
		t.Error("expected a non-zero schema version after migrating")
	}

	db := assertSchema(t, dsn)
	for _, table := range []string{
		"users", "organizations", "projects", "workspaces", "environments",
		"environment_events", "share_links", "workspace_members", "schema_migrations",
	} {
		var exists bool
		if err := db.QueryRow(
			"select exists (select 1 from information_schema.tables where table_name=$1)", table,
		).Scan(&exists); err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("table %q was not created", table)
		}
	}

	// The legacy sandbox model is dropped by migration 000009 and must not come
	// back on a fresh database.
	var sandboxes int
	if err := db.QueryRow(
		"select count(*) from information_schema.tables where table_name='sandboxes'").Scan(&sandboxes); err != nil {
		t.Fatalf("check sandboxes table: %v", err)
	}
	if sandboxes != 0 {
		t.Error("the legacy sandboxes table should not exist after migrating")
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	dsn := freshDatabase(t)

	first, err := Migrate(dsn)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Migrate(dsn)
	if err != nil {
		t.Fatalf("second Migrate should be a no-op, got: %v", err)
	}
	if first != second {
		t.Errorf("version moved from %d to %d on a second run", first, second)
	}
}

func TestCurrentVersion(t *testing.T) {
	dsn := freshDatabase(t)

	if v, dirty, err := CurrentVersion(dsn); err != nil {
		t.Fatalf("CurrentVersion on an empty database: %v", err)
	} else if v != 0 || dirty {
		t.Errorf("expected version 0 and not dirty, got %d dirty=%v", v, dirty)
	}

	if _, err := Migrate(dsn); err != nil {
		t.Fatal(err)
	}

	v, dirty, err := CurrentVersion(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if v == 0 {
		t.Error("expected a version after migrating")
	}
	if dirty {
		t.Error("schema should not be dirty")
	}
}
