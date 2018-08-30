package migrate

import "testing"

func TestTableAdapterFuncs(t *testing.T) {
	tests := []struct {
		Name     string
		Func     func(log LogFunc) *TableAdapter
		Expected TableAdapter
	}{
		{
			Name: "MySQL",
			Func: NewMySQLAdapter,
			Expected: TableAdapter{
				CreateTableOptions: " ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
				PlaceholderVersion: "?",
				PlaceholderUpgrade: "?",
				PlaceholderComment: "?",
			},
		},
		{
			Name: "PostgreSQL",
			Func: NewPostgreSQLAdapter,
			Expected: TableAdapter{
				PlaceholderVersion: "$1",
				PlaceholderUpgrade: "$2",
				PlaceholderComment: "$3",
			},
		},
		{
			Name: "SQLite",
			Func: NewSQLiteAdapter,
			Expected: TableAdapter{
				PlaceholderVersion: "?",
				PlaceholderUpgrade: "?",
				PlaceholderComment: "?",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			a := tt.Func(t.Logf)
			if a == nil {
				t.Error("adapter unexpectedly nil")
				return
			}
			if a.CreateTableOptions != tt.Expected.CreateTableOptions {
				t.Errorf("expected CreateTableOptions to be %q, got %q", tt.Expected.CreateTableOptions, a.CreateTableOptions)
			}
			if a.PlaceholderVersion != tt.Expected.PlaceholderVersion {
				t.Errorf("expected PlaceholderVersion to be %q, got %q", tt.Expected.PlaceholderVersion, a.PlaceholderVersion)
			}
			if a.PlaceholderComment != tt.Expected.PlaceholderComment {
				t.Errorf("expected PlaceholderComment to be %q, got %q", tt.Expected.PlaceholderComment, a.PlaceholderComment)
			}
		})
	}
}
