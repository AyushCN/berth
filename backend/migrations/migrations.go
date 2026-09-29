// Package migrations embeds the SQL schema migrations and applies them on
// startup.
//
// The application previously had no migration runner at all: db.Init only
// opened a connection pool, so the schema had to be applied by hand with psql.
// A fresh deployment started against an empty database and failed every query,
// and the documented workflow (`make migrate-up`, plus a list of migration
// filenames that did not exist) could not be followed.
package migrations

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	// lib/pq registers the "postgres" driver that golang-migrate needs.
	_ "github.com/lib/pq"
)

//go:embed *.sql
var FS embed.FS

// newMigrator opens a dedicated connection for migrations.
//
// It deliberately does not borrow the application's pgxpool. golang-migrate
// holds a connection for the duration of a run and takes an advisory lock,
// and driving that through a pool wrapper deadlocks. Migrations are a one-shot
// startup concern, so a short-lived dedicated connection is simpler and costs
// nothing.
func newMigrator(dsn string) (*migrate.Migrate, func(), error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open migration connection: %w", err)
	}
	cleanup := func() { _ = db.Close() }

	if err := db.Ping(); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to reach the database for migrations: %w", err)
	}

	src, err := iofs.New(FS, ".")
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to open embedded migrations: %w", err)
	}

	driver, err := migratepostgres.WithInstance(db, &migratepostgres.Config{})
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to prepare migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to initialise migrator: %w", err)
	}
	return m, cleanup, nil
}

// Migrate brings the database up to the latest schema version and returns it.
//
// Safe to call on every boot: golang-migrate records the applied version in
// schema_migrations and does nothing when the database is already current.
// migrate.ErrNoChange is swallowed so callers do not have to special-case it.
func Migrate(dsn string) (version uint, err error) {
	m, cleanup, err := newMigrator(dsn)
	if err != nil {
		return 0, err
	}
	defer cleanup()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		// A dirty schema means a migration failed part way through. Report the
		// version it stalled at so it can be repaired deliberately.
		if v, dirty, verr := m.Version(); verr == nil && dirty {
			return v, fmt.Errorf("schema is dirty at version %d; a migration failed part way through and needs manual repair", v)
		}
		return 0, fmt.Errorf("failed to apply migrations: %w", err)
	}

	v, dirty, err := m.Version()
	if err != nil {
		// An empty database reports no version. Up() would have applied
		// everything, so treat this as 0 rather than a failure.
		if errors.Is(err, migrate.ErrNilVersion) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read schema version: %w", err)
	}
	if dirty {
		return v, fmt.Errorf("schema is dirty at version %d", v)
	}
	return v, nil
}

// CurrentVersion reports the applied schema version without changing it.
func CurrentVersion(dsn string) (version uint, dirty bool, err error) {
	m, cleanup, err := newMigrator(dsn)
	if err != nil {
		return 0, false, err
	}
	defer cleanup()

	v, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	return v, dirty, err
}
