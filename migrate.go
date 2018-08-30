// Package migrate provides helpers for running SQL database migrations. It's
// designed for migrations that are specified in code and distributed as part
// of the application binary, and applied as part of the application startup
// (rather than via external files and an external tool).
package migrate

import (
	"context"
	"database/sql"
	"fmt"
)

// MigrationFunc is type of function used for the up and down migrations.
type MigrationFunc func(ctx context.Context, db *sql.DB) error

// Migration represents an individual migration step. The Up function is run
// to migrate from the previous version to this version, and the Down function
// can be run to go back the other way. The Comment is inserted into the
// schema_versions table after migrating to this version.
type Migration struct {
	// Comment should be a string describing the migration.
	Comment string

	// Up should be a function to apply the migration.
	Up MigrationFunc

	// Down should be a function to revert the migration.
	Down MigrationFunc
}

// ExecQueries generates a migration function from a list of SQL queries.
// Running the returned function will execute each of the SQL queries as its
// migration step.
func ExecQueries(queries []string) MigrationFunc {
	return func(ctx context.Context, db *sql.DB) error {
		for i, q := range queries {
			_, err := db.ExecContext(ctx, q)
			if err != nil {
				return fmt.Errorf("error with query %d: %s", i, err)
			}
		}
		return nil
	}
}

// Up upgrades the given database to the latest migration in the list
// of passed migrations.
func Up(ctx context.Context, db *sql.DB, adapter Adapter, migrations []Migration) error {
	return UpToVersion(ctx, db, adapter, len(migrations), migrations)
}

// UpToVersion migrates the database to the specified version.
func UpToVersion(ctx context.Context, db *sql.DB, adapter Adapter, targetVersion int, migrations []Migration) error {
	if err := adapter.PrepareSchemaVersions(ctx, db); err != nil {
		return fmt.Errorf("error preparing schema versions: %s", err)
	}
	currentVersion, err := adapter.QuerySchemaVersion(ctx, db)
	if err != nil {
		return fmt.Errorf("error querying current schema version: %s", err)
	}
	adapter.Log("Current database version is %d", currentVersion)
	for i, m := range migrations {
		version := i + 1
		if version <= currentVersion {
			continue
		}
		if version > targetVersion {
			break
		}
		adapter.Log("Upgrading database to version %d", version)
		if err := m.Up(ctx, db); err != nil {
			return fmt.Errorf("error upgrading database to version %d: %s", version, err)
		}
		if err := adapter.InsertSchemaVersion(ctx, db, version, true, m.Comment); err != nil {
			return fmt.Errorf("error inserting schema version for version %d: %s", version, err)
		}
	}
	return nil
}

// DownToVersion migrates the database down to the specified version. This is
// separate from UpToVersion because downgrades can often be destructive, and a
// separate function makes it slightly more difficult to unintentionally
// downgrade (e.g. by passing an incorrect target version).
func DownToVersion(ctx context.Context, db *sql.DB, adapter Adapter, targetVersion int, migrations []Migration) error {
	if err := adapter.PrepareSchemaVersions(ctx, db); err != nil {
		return fmt.Errorf("error preparing schema versions: %s", err)
	}
	currentVersion, err := adapter.QuerySchemaVersion(ctx, db)
	if err != nil {
		return fmt.Errorf("error querying current schema version: %s", err)
	}
	adapter.Log("Current database version is %d", currentVersion)
	for i := len(migrations) - 1; i >= 0; i-- {
		m := migrations[i]
		version := i + 1
		if version > currentVersion {
			continue
		}
		if version <= targetVersion {
			break
		}
		adapter.Log("Downgrading database to version %d", version)
		if err := m.Down(ctx, db); err != nil {
			return fmt.Errorf("error upgrading database to version %d: %s", version, err)
		}
		if err := adapter.InsertSchemaVersion(ctx, db, version, false, m.Comment); err != nil {
			return fmt.Errorf("error inserting schema_versions row for version %d: %s", version, err)
		}
	}
	return nil
}
