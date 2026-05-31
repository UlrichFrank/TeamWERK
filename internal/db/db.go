package db

import (
	"database/sql"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;`); err != nil {
		return nil, fmt.Errorf("pragma: %w", err)
	}
	return db, nil
}

func MigrateForce(sqlDB *sql.DB, migrationsFS fs.FS, version int) error {
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		return fmt.Errorf("pragma off: %w", err)
	}
	defer func() {
		sqlDB.Exec(`PRAGMA foreign_keys = ON`)
		sqlDB.SetMaxOpenConns(0)
	}()

	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("migration source: %w", err)
	}
	driver, err := sqlite3.WithInstance(sqlDB, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("migration driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("migration init: %w", err)
	}
	return m.Force(version)
}

func Migrate(sqlDB *sql.DB, migrationsFS fs.FS) error {
	// PRAGMA foreign_keys is a no-op inside a transaction (SQLite limitation).
	// golang-migrate wraps migrations in transactions, so we disable FK enforcement
	// at the connection level here and restore it after migrations complete.
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		return fmt.Errorf("pragma off: %w", err)
	}
	defer func() {
		sqlDB.Exec(`PRAGMA foreign_keys = ON`)
		sqlDB.SetMaxOpenConns(0)
	}()

	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("migration source: %w", err)
	}
	driver, err := sqlite3.WithInstance(sqlDB, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("migration driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("migration init: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration up: %w", err)
	}
	return nil
}
