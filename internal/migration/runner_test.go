package migration

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestLoad_ValidFiles_ReturnsVersionOrder(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"migrations/0002_add_probe.sql": {Data: []byte("SELECT 2;")},
		"migrations/README.md":          {Data: []byte("ignored")},
		"migrations/0001_create.sql":    {Data: []byte("SELECT 1;")},
	}

	got, err := Load(fsys, "migrations")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("Load() count = %d, want 2", len(got))
	}
	if got[0].Version != 1 || got[0].Name != "create" || got[0].SQL != "SELECT 1;" {
		t.Fatalf("Load()[0] = %#v, want version 1 create migration", got[0])
	}
	if got[1].Version != 2 || got[1].Name != "add_probe" || got[1].SQL != "SELECT 2;" {
		t.Fatalf("Load()[1] = %#v, want version 2 add_probe migration", got[1])
	}
}

func TestLoad_InvalidSQLName_ReturnsError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
	}{
		{name: "missing version", path: "migrations/create.sql"},
		{name: "short version", path: "migrations/001_create.sql"},
		{name: "missing name", path: "migrations/0001_.sql"},
		{name: "uppercase name", path: "migrations/0001_Create.sql"},
		{name: "double separator", path: "migrations/0001_create__probe.sql"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fsys := fstest.MapFS{tt.path: {Data: []byte("SELECT 1;")}}
			_, err := Load(fsys, "migrations")
			if err == nil {
				t.Fatal("Load() error = nil, want invalid migration name error")
			}
			if !strings.Contains(err.Error(), tt.path) {
				t.Fatalf("Load() error = %q, want path %q", err, tt.path)
			}
		})
	}
}

func TestLoad_DuplicateVersion_ReturnsError(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"migrations/0001_create.sql": {Data: []byte("SELECT 1;")},
		"migrations/0001_other.sql":  {Data: []byte("SELECT 2;")},
	}

	_, err := Load(fsys, "migrations")
	if err == nil {
		t.Fatal("Load() error = nil, want duplicate version error")
	}
	if !strings.Contains(err.Error(), "0001") {
		t.Fatalf("Load() error = %q, want duplicate version context", err)
	}
}

func TestLoad_TransactionControl_ReturnsError(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"migrations/0001_unsafe.sql": {Data: []byte("SELECT 1; COMMIT;")},
	}

	_, err := Load(fsys, "migrations")
	if err == nil {
		t.Fatal("Load() error = nil, want transaction control error")
	}
	if !strings.Contains(err.Error(), "0001_unsafe.sql") {
		t.Fatalf("Load() error = %q, want migration path", err)
	}
}

func TestNewRunner_InvalidInput_ReturnsError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		pool      *pgxpool.Pool
		fsys      fs.FS
		directory string
	}{
		{
			name:      "nil pool",
			fsys:      fstest.MapFS{},
			directory: "migrations",
		},
		{
			name:      "nil filesystem",
			pool:      new(pgxpool.Pool),
			directory: "migrations",
		},
		{
			name: "empty directory",
			pool: new(pgxpool.Pool),
			fsys: fstest.MapFS{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewRunner(tt.pool, tt.fsys, tt.directory)
			if err == nil {
				t.Fatal("NewRunner() error = nil, want invalid input error")
			}
		})
	}
}
