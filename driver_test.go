package migrate

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
)

func init() {
	sql.Register("migrate_test", &MockDriver{})
}

type contextKey int

const mockDataKey contextKey = 0

type MockData struct {
	ExecErr   error
	ExecLogs  []MockQueryLog
	QueryErr  error
	QueryLogs []MockQueryLog
	QueryRows MockRows
}

func MockDataFromContext(ctx context.Context) *MockData {
	return ctx.Value(mockDataKey).(*MockData)
}

func WithMockData(ctx context.Context) (*MockData, context.Context) {
	md := &MockData{}
	return md, context.WithValue(ctx, mockDataKey, md)
}

func (md *MockData) Check(t *testing.T, expected MockData) {
	checkLogs(t, "md.ExecLogs", md.ExecLogs, expected.ExecLogs)
	checkLogs(t, "md.QueryLogs", md.QueryLogs, expected.QueryLogs)
}

func (md *MockData) Reset() {
	md.ExecErr = nil
	md.ExecLogs = nil
	md.QueryErr = nil
	md.QueryLogs = nil
	md.QueryRows = MockRows{}
}

func checkLogs(t *testing.T, key string, logs []MockQueryLog, expected []MockQueryLog) {
	if len(logs) != len(expected) {
		t.Errorf("expected %d %s, got %d", len(expected), key, len(logs))
		if len(logs) > len(expected) {
			t.Errorf("%s had %d extra items: %#v", key, len(logs)-len(expected), logs[len(expected):])
		}
	}
	for i, el := range expected {
		if i >= len(logs) {
			break
		}
		l := logs[i]
		if l.Query != el.Query {
			t.Errorf("expected %s[%d].Query to be %q, got %q", key, i, el.Query, l.Query)
		}
		if len(l.Args) != len(el.Args) {
			t.Errorf("expected %d %s[%d].Args, got %d", len(el.Args), key, i, len(l.Args))
			if len(l.Args) > len(el.Args) {
				t.Errorf("%s[%d].Args had %d extra items: %#v", key, i, len(l.Args)-len(el.Args), l.Args[len(el.Args):])
			}
		}
		for j, ela := range el.Args {
			if j >= len(l.Args) {
				break
			}
			la := l.Args[j]
			if la != ela {
				t.Errorf("expected %s[%d].Args[%d] to be %#v, got %#v", key, i, j, ela, la)
			}
		}
	}
}

type MockDriver struct{}

func (md *MockDriver) Open(name string) (driver.Conn, error) {
	return &MockConn{}, nil
}

type MockQueryLog struct {
	Query string
	Args  []driver.NamedValue
}

type MockConn struct{}

func (c *MockConn) Begin() (driver.Tx, error) {
	return nil, errors.New("conn.Begin() not implemented")
}

func (c *MockConn) Close() error {
	return nil
}

func (c *MockConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	md := MockDataFromContext(ctx)
	if md.ExecErr != nil {
		return nil, md.ExecErr
	}
	md.ExecLogs = append(md.ExecLogs, MockQueryLog{Query: query, Args: args})
	return &MockResult{}, nil
}

func (c *MockConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("conn.Prepare() not implemented")
}

func (c *MockConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	md := MockDataFromContext(ctx)
	if md.QueryErr != nil {
		return nil, md.QueryErr
	}
	md.QueryLogs = append(md.QueryLogs, MockQueryLog{Query: query, Args: args})
	return &md.QueryRows, nil
}

type MockResult struct{}

func (r *MockResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (r *MockResult) RowsAffected() (int64, error) {
	return 0, nil
}

// MockRows mocks the Rows object returned by the DB for a Query call. Note
// this implementation assumes that we're only ever going to be called to
// lookup the current schema version. It returns schema version 0 by default,
// but that can be changed by changing the Version field.
type MockRows struct {
	Version int
}

func (r *MockRows) Close() error {
	return nil
}

func (r *MockRows) Columns() []string {
	return []string{"version"}
}

func (r *MockRows) Next(dest []driver.Value) error {
	dest[0] = int64(r.Version)
	return nil
}
