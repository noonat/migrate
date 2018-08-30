package migrate

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"reflect"
	"testing"
)

// Validate that TableAdapter satisfies the Adapter interface.
var _ Adapter = &TableAdapter{}

func setupMockDB(t *testing.T) (*sql.DB, *MockData, context.Context) {
	db, err := sql.Open("migrate_test", "")
	if err != nil {
		t.Fatal("error opening mock db")
	}
	md, ctx := WithMockData(context.Background())
	return db, md, ctx
}

func TestExecQueries(t *testing.T) {
	db, md, ctx := setupMockDB(t)
	defer db.Close()

	queries := []string{
		"example query 1",
		"example query 2",
	}
	migrateFunc := ExecQueries(queries)
	err := migrateFunc(ctx, db)
	if err != nil {
		t.Errorf("unexpected err: %v", err)
	}
	md.Check(t, MockData{
		ExecLogs: []MockQueryLog{
			{Query: queries[0]},
			{Query: queries[1]},
		},
	})
}

func TestExecQueriesError(t *testing.T) {
	db, md, ctx := setupMockDB(t)
	defer db.Close()

	md.ExecErr = errors.New("mock error")
	queries := []string{
		"example query 1",
		"example query 2",
	}
	migrateFunc := ExecQueries(queries)
	err := migrateFunc(ctx, db)
	expectedErr := errors.New("error with query 0: mock error")
	if err.Error() != expectedErr.Error() {
		t.Errorf("expected error %q, got %q", expectedErr, err)
	}
	md.Check(t, MockData{})
}

// This is kind of a long test, but mostly because of the expected value
// comparisons. It applies the migrations once, where the second migration
// fails, then runs them again. This is used to validate error handling and
// that it doesn't re-run migrations that have alraedy been run. It then
// downgrades to a specific version to test that code path.
func TestDown(t *testing.T) {
	db, md, ctx := setupMockDB(t)
	defer db.Close()

	adapter := NewPostgreSQLAdapter(t.Logf)
	up := []bool{false, false, false}
	down := []bool{false, false, false}
	migrations := []Migration{
		{
			Comment: "example comment 1",
			Up: func(ctx context.Context, db *sql.DB) error {
				up[0] = true
				return nil
			},
			Down: func(ctx context.Context, db *sql.DB) error {
				down[0] = true
				return nil
			},
		},
		{
			Comment: "example comment 2",
			Up: func(ctx context.Context, db *sql.DB) error {
				if !up[1] {
					up[1] = true
					return errors.New("mock error")
				}
				return nil
			},
			Down: func(ctx context.Context, db *sql.DB) error {
				down[1] = true
				return nil
			},
		},
		{
			Comment: "example comment 3",
			Up: func(ctx context.Context, db *sql.DB) error {
				up[2] = true
				return nil
			},
			Down: func(ctx context.Context, db *sql.DB) error {
				down[2] = true
				return nil
			},
		},
	}

	// The first time this is run, it should fail on the second migration.
	err := Up(ctx, db, adapter, migrations)
	expectedErr := errors.New("error upgrading database to version 2: mock error")
	if err.Error() != expectedErr.Error() {
		t.Errorf("expected err to be %q, got %q", expectedErr, err)
	}
	expectedCreateSQL := `
		CREATE TABLE IF NOT EXISTS schema_versions (
			version INT NOT NULL PRIMARY KEY,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			upgrade TINYINT NOT NULL,
			comment TEXT NOT NULL
		)
	`
	expectedSelectSQL := `SELECT version FROM schema_versions ORDER BY created_at DESC LIMIT 1`
	expectedInsertSQL := `
		INSERT INTO schema_versions (version, upgrade, comment) VALUES ($1, $2, $3)
	`
	md.Check(t, MockData{
		ExecLogs: []MockQueryLog{
			{
				Query: expectedCreateSQL,
			},
			{
				Query: expectedInsertSQL,
				Args: []driver.NamedValue{
					{Name: "", Ordinal: 1, Value: int64(1)},
					{Name: "", Ordinal: 2, Value: true},
					{Name: "", Ordinal: 3, Value: "example comment 1"},
				},
			},
		},
		QueryLogs: []MockQueryLog{
			{Query: expectedSelectSQL},
		},
	})
	expectedUp := []bool{true, true, false}
	expectedDown := []bool{false, false, false}
	if !reflect.DeepEqual(up, expectedUp) {
		t.Errorf("expected up to be %v, got %v", expectedUp, up)
	}
	if !reflect.DeepEqual(down, expectedDown) {
		t.Errorf("expected down to be %v, got %v", expectedDown, down)
	}

	// Running it again should only apply the second and third migrations
	md.Reset()
	md.QueryRows.Version = 1
	err = Up(ctx, db, adapter, migrations)
	if err != nil {
		t.Errorf("unexpected err: %v", err)
	}
	md.Check(t, MockData{
		ExecLogs: []MockQueryLog{
			{
				Query: expectedCreateSQL,
			},
			{
				Query: expectedInsertSQL,
				Args: []driver.NamedValue{
					{Name: "", Ordinal: 1, Value: int64(2)},
					{Name: "", Ordinal: 2, Value: true},
					{Name: "", Ordinal: 3, Value: "example comment 2"},
				},
			},
			{
				Query: expectedInsertSQL,
				Args: []driver.NamedValue{
					{Name: "", Ordinal: 1, Value: int64(3)},
					{Name: "", Ordinal: 2, Value: true},
					{Name: "", Ordinal: 3, Value: "example comment 3"},
				},
			},
		},
		QueryLogs: []MockQueryLog{
			{Query: expectedSelectSQL},
		},
	})
	expectedUp = []bool{true, true, true}
	expectedDown = []bool{false, false, false}
	if !reflect.DeepEqual(up, expectedUp) {
		t.Errorf("expected up to be %v, got %v", expectedUp, up)
	}
	if !reflect.DeepEqual(down, expectedDown) {
		t.Errorf("expected down to be %v, got %v", expectedDown, down)
	}

	// This should call the Down methods for the second and third migrations.
	up = []bool{false, false, false}
	down = []bool{false, false, false}
	md.Reset()
	md.QueryRows.Version = 3
	err = DownToVersion(ctx, db, adapter, 1, migrations)
	if err != nil {
		t.Errorf("unexpected err: %q", err)
	}
	md.Check(t, MockData{
		ExecLogs: []MockQueryLog{
			{
				Query: expectedCreateSQL,
			},
			{
				Query: expectedInsertSQL,
				Args: []driver.NamedValue{
					{Name: "", Ordinal: 1, Value: int64(3)},
					{Name: "", Ordinal: 2, Value: false},
					{Name: "", Ordinal: 3, Value: "example comment 3"},
				},
			},
			{
				Query: expectedInsertSQL,
				Args: []driver.NamedValue{
					{Name: "", Ordinal: 1, Value: int64(2)},
					{Name: "", Ordinal: 2, Value: false},
					{Name: "", Ordinal: 3, Value: "example comment 2"},
				},
			},
		},
		QueryLogs: []MockQueryLog{
			{Query: expectedSelectSQL},
		},
	})
	expectedUp = []bool{false, false, false}
	expectedDown = []bool{false, true, true}
	if !reflect.DeepEqual(up, expectedUp) {
		t.Errorf("expected up to be %v, got %v", expectedUp, up)
	}
	if !reflect.DeepEqual(down, expectedDown) {
		t.Errorf("expected down to be %v, got %v", expectedDown, down)
	}
}
