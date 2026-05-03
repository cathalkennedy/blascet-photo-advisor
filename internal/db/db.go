package db

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps the sql.DB connection with schema awareness
type DB struct {
	*sql.DB
}

// Open opens a SQLite database at the given path and runs migrations
func Open(path string) (*DB, error) {
	// Ensure parent directory exists
	dbPath := filepath.Clean(path)

	// Open with WAL mode for better concurrency
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)", dbPath)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	db := &DB{DB: sqlDB}

	// Run migrations
	if err := db.migrate(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return db, nil
}

// migrate applies all pending migrations
func (db *DB) migrate() error {
	// Get list of migration files
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("reading migrations directory: %w", err)
	}

	// Parse migration versions and sort
	var migrations []migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		// Extract version from filename (e.g., "001_initial_schema.sql" -> 1)
		parts := strings.SplitN(entry.Name(), "_", 2)
		if len(parts) < 2 {
			continue
		}

		version, err := strconv.Atoi(parts[0])
		if err != nil {
			slog.Warn("skipping invalid migration filename", "file", entry.Name(), "error", err)
			continue
		}

		migrations = append(migrations, migration{
			version:  version,
			filename: entry.Name(),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})

	// Apply migrations
	for _, m := range migrations {
		applied, err := db.isMigrationApplied(m.version)
		if err != nil {
			return fmt.Errorf("checking migration %d: %w", m.version, err)
		}

		if applied {
			continue
		}

		slog.Info("applying migration", "version", m.version, "file", m.filename)

		content, err := migrationsFS.ReadFile("migrations/" + m.filename)
		if err != nil {
			return fmt.Errorf("reading migration %d: %w", m.version, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("starting transaction for migration %d: %w", m.version, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("executing migration %d: %w", m.version, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", m.version); err != nil {
			tx.Rollback()
			return fmt.Errorf("recording migration %d: %w", m.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("committing migration %d: %w", m.version, err)
		}

		slog.Info("migration applied", "version", m.version)
	}

	return nil
}

// isMigrationApplied checks if a migration version has been applied
func (db *DB) isMigrationApplied(version int) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count)
	if err != nil {
		// Table doesn't exist yet, no migrations applied
		if strings.Contains(err.Error(), "no such table") {
			return false, nil
		}
		return false, err
	}
	return count > 0, nil
}

type migration struct {
	version  int
	filename string
}
