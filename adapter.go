package migrate

import (
	"context"
	"database/sql"
	"fmt"
)

// Adapter is the interface that wraps the methods required to track information
// about schema versions in the database.
type Adapter interface {
	// Log is used to log information about migrations.
	Log(format string, v ...interface{})

	// PrepareSchemaVersions should ensure that there is a place to store
	// information about migration versions that have been applied to the
	// database schema.
	PrepareSchemaVersions(ctx context.Context, db *sql.DB) error

	// QuerySchemaVersion should return the current migration version applied
	// to the database schema.
	QuerySchemaVersion(ctx context.Context, db *sql.DB) (int, error)

	// InsertSchemaVersion should insert a new schema version in to the
	// database, to reflect that the given migration has been applied.
	InsertSchemaVersion(ctx context.Context, db *sql.DB, version int, upgrade bool, comment string) error
}

// LogFunc is the log function type used by migration logging.
type LogFunc func(format string, v ...interface{})

// TableAdapter implements Adapter by using a table to track migration versions.
// It creates a schema_versions table in your database, and rows are inserted
// into it to track the history of the migrations applied ot the schema. It
// queries the highest version number in the table to determine the current
// migration version.
//
// It provides several fields to customize the behavior for different database
// drivers, and there are constructor functions for common ones.
type TableAdapter struct {
	// Log is the function to use for adapter logging.
	LogFunc LogFunc

	// CreateTableOptions can be used to specify arbitrary SQL to include at
	// the end of the CREATE TABLE statement (to specify a CHARSET for a MySQL
	// table, for instance).
	CreateTableOptions string

	// PlaceholderVersion specifies the placeholder to use in the INSERT query
	// for the version number, the first value in the insert. This would be
	// something like ? for MySQL or $1 for PostgreSQL.
	PlaceholderVersion string

	// PlaceholderUpgrade specifies the placeholder to use in the INSERT query
	// for the upgrade boolean, the second value in the insert. This would be
	// something like ? for MySQL or $2 for PostgreSQL.
	PlaceholderUpgrade string

	// PlaceholderComment specifies the placeholder to use in the INSERT query
	// for the comment, the third value in the insert. This would be something
	// like ? for MySQL or $3 for PostgreSQL.
	PlaceholderComment string
}

// NewMySQLAdapter creates a TableAdapter compatible with
// https://github.com/go-sql-driver/mysql/. This specifies InnoDB for the
// engine and a table charset of utf8mb4. The log parameter can be set to
// log.Printf or a compatible function, or nil if you don't want to log.
func NewMySQLAdapter(log LogFunc) *TableAdapter {
	return &TableAdapter{
		LogFunc:            log,
		CreateTableOptions: " ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		PlaceholderVersion: "?",
		PlaceholderUpgrade: "?",
		PlaceholderComment: "?",
	}
}

// NewPostgreSQLAdapter creates a TableAdapter compatible with
// https://github.com/mattn/go-sqlite3/. The log parameter can be set to
// log.Printf or a compatible function, or nil if you don't want to log.
func NewPostgreSQLAdapter(log LogFunc) *TableAdapter {
	return &TableAdapter{
		LogFunc:            log,
		PlaceholderVersion: "$1",
		PlaceholderUpgrade: "$2",
		PlaceholderComment: "$3",
	}
}

// NewSQLiteAdapter creates a TableAdapter compatible with
// https://github.com/lib/pq/. The log parameter can be set to log.Printf or
// a compatible function, or nil if you don't want to log.
func NewSQLiteAdapter(log LogFunc) *TableAdapter {
	return &TableAdapter{
		LogFunc:            log,
		PlaceholderVersion: "?",
		PlaceholderUpgrade: "?",
		PlaceholderComment: "?",
	}
}

// Log is used to log information about migrations. It calls the underlying
// LogFunc on the TableAdapter, if it is not nil.
func (t *TableAdapter) Log(format string, v ...interface{}) {
	if t.LogFunc != nil {
		t.LogFunc(format, v...)
	}
}

// PrepareSchemaVersions ensures that the schema_versions table exists.
func (t *TableAdapter) PrepareSchemaVersions(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS schema_versions (
			version INT NOT NULL PRIMARY KEY,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			upgrade TINYINT NOT NULL,
			comment TEXT NOT NULL
		)%s
	`, t.CreateTableOptions))
	return err
}

// QuerySchemaVersion returns the current schema version.
func (t *TableAdapter) QuerySchemaVersion(ctx context.Context, db *sql.DB) (int, error) {
	var currentVersion int
	row := db.QueryRowContext(ctx, `SELECT version FROM schema_versions ORDER BY created_at DESC LIMIT 1`)
	if err := row.Scan(&currentVersion); err == sql.ErrNoRows {
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	return currentVersion, nil
}

// InsertSchemaVersion inserts a new version into the schema_versions table.
func (t *TableAdapter) InsertSchemaVersion(ctx context.Context, db *sql.DB, version int, upgrade bool, comment string) error {
	_, err := db.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO schema_versions (version, upgrade, comment) VALUES (%s, %s, %s)
	`, t.PlaceholderVersion, t.PlaceholderUpgrade, t.PlaceholderComment), version, upgrade, comment)
	return err
}
