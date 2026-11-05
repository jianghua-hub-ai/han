package identity

import (
	"context"
	"errors"
	"strings"
	"testing"

	"han/internal/platform"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestGetAuthorStates_ReturnsReadOnlyStatusMap(t *testing.T) {
	rows := &authorStateRowsStub{records: []AuthorState{
		{ID: "00000000-0000-0000-0000-000000000001", Status: UserStatusActive},
		{ID: "00000000-0000-0000-0000-000000000002", Status: UserStatusSuspended},
		{ID: "00000000-0000-0000-0000-000000000003", Status: UserStatusDeleted},
	}}
	database := &authorStateDatabaseStub{rows: rows}
	reader, err := NewArticleIdentityReader(database)
	if err != nil {
		t.Fatalf("NewArticleIdentityReader() error = %v", err)
	}

	states, err := reader.GetAuthorStates(context.Background(), []string{
		"00000000-0000-0000-0000-000000000001",
		"00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003",
		"00000000-0000-0000-0000-000000000004",
		"00000000-0000-0000-0000-000000000001",
	})
	if err != nil {
		t.Fatalf("GetAuthorStates() error = %v", err)
	}
	if len(states) != 3 ||
		states["00000000-0000-0000-0000-000000000001"] != UserStatusActive ||
		states["00000000-0000-0000-0000-000000000002"] != UserStatusSuspended ||
		states["00000000-0000-0000-0000-000000000003"] != UserStatusDeleted {
		t.Fatalf("GetAuthorStates() states = %#v", states)
	}
	if database.queryCalls != 1 {
		t.Fatalf("Query() calls = %d, want 1", database.queryCalls)
	}
	if len(database.args) != 1 {
		t.Fatalf("Query() args = %#v, want one UUID array", database.args)
	}
	ids, ok := database.args[0].([]pgtype.UUID)
	if !ok || len(ids) != 4 {
		t.Fatalf("Query() UUID arg = %#v, want four deduplicated UUIDs", database.args[0])
	}
	if !rows.closed {
		t.Fatal("GetAuthorStates() did not close rows")
	}
	upperSQL := strings.ToUpper(database.query)
	if !strings.Contains(upperSQL, "SELECT") ||
		strings.Contains(upperSQL, "UPDATE") ||
		strings.Contains(upperSQL, "INSERT") ||
		strings.Contains(upperSQL, "DELETE") {
		t.Fatalf("author state SQL is not read-only: %s", database.query)
	}
}

func TestGetAuthorStates_EmptyInputDoesNotQuery(t *testing.T) {
	database := &authorStateDatabaseStub{}
	reader, err := NewArticleIdentityReader(database)
	if err != nil {
		t.Fatalf("NewArticleIdentityReader() error = %v", err)
	}

	states, err := reader.GetAuthorStates(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetAuthorStates() error = %v", err)
	}
	if len(states) != 0 || database.queryCalls != 0 {
		t.Fatalf("empty input states/calls = %#v/%d", states, database.queryCalls)
	}
}

func TestGetAuthorStates_InvalidInputOrDatabaseFailureFailsClosed(t *testing.T) {
	tests := []struct {
		name      string
		ids       []string
		database  *authorStateDatabaseStub
		wantClose bool
	}{
		{name: "invalid user ID", ids: []string{"not-an-opaque-user-id"}, database: &authorStateDatabaseStub{}},
		{name: "database failure", ids: []string{"00000000-0000-0000-0000-000000000001"}, database: &authorStateDatabaseStub{err: errors.New("synthetic query failure")}},
		{
			name: "unknown status", ids: []string{"00000000-0000-0000-0000-000000000001"},
			database: &authorStateDatabaseStub{rows: &authorStateRowsStub{records: []AuthorState{
				{ID: "00000000-0000-0000-0000-000000000001", Status: "unknown"},
			}}},
			wantClose: true,
		},
		{
			name: "scan failure", ids: []string{"00000000-0000-0000-0000-000000000001"},
			database: &authorStateDatabaseStub{rows: &authorStateRowsStub{
				records:   []AuthorState{{ID: "00000000-0000-0000-0000-000000000001", Status: UserStatusActive}},
				scanErrAt: 1, scanErr: errors.New("synthetic scan failure"),
			}},
			wantClose: true,
		},
		{
			name: "rows failure", ids: []string{"00000000-0000-0000-0000-000000000001"},
			database: &authorStateDatabaseStub{rows: &authorStateRowsStub{
				err: errors.New("synthetic rows failure"),
			}},
			wantClose: true,
		},
		{
			name: "conflicting duplicate row", ids: []string{"00000000-0000-0000-0000-000000000001"},
			database: &authorStateDatabaseStub{rows: &authorStateRowsStub{records: []AuthorState{
				{ID: "00000000-0000-0000-0000-000000000001", Status: UserStatusActive},
				{ID: "00000000-0000-0000-0000-000000000001", Status: UserStatusSuspended},
			}}},
			wantClose: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader, err := NewArticleIdentityReader(test.database)
			if err != nil {
				t.Fatalf("NewArticleIdentityReader() error = %v", err)
			}
			_, err = reader.GetAuthorStates(context.Background(), test.ids)
			if platform.PublicCode(err) != platform.CodePlatformInternalError {
				t.Fatalf("GetAuthorStates() code = %q, want %q (err=%v)", platform.PublicCode(err), platform.CodePlatformInternalError, err)
			}
			if test.wantClose && (test.database.rows == nil || !test.database.rows.closed) {
				t.Fatal("GetAuthorStates() did not close rows after failure")
			}
		})
	}
}

type authorStateDatabaseStub struct {
	err        error
	rows       *authorStateRowsStub
	queryCalls int
	query      string
	args       []any
}

func (s *authorStateDatabaseStub) Query(
	_ context.Context,
	query string,
	args ...any,
) (pgx.Rows, error) {
	s.queryCalls++
	s.query = query
	s.args = append([]any(nil), args...)
	if s.err != nil {
		return nil, s.err
	}
	if s.rows == nil {
		s.rows = &authorStateRowsStub{}
	}
	return s.rows, nil
}

type authorStateRowsStub struct {
	records   []AuthorState
	index     int
	closed    bool
	scanErrAt int
	scanErr   error
	err       error
}

func (s *authorStateRowsStub) Close() { s.closed = true }

func (s *authorStateRowsStub) Next() bool {
	if s.index >= len(s.records) {
		return false
	}
	s.index++
	return true
}

func (s *authorStateRowsStub) Scan(dest ...any) error {
	if s.index == 0 || s.index > len(s.records) || len(dest) != 2 {
		return errors.New("invalid author state scan")
	}
	if s.scanErrAt == s.index {
		return s.scanErr
	}
	record := s.records[s.index-1]
	id, idOK := dest[0].(*string)
	status, statusOK := dest[1].(*UserStatus)
	if !idOK || !statusOK {
		return errors.New("invalid author state destination")
	}
	*id = record.ID
	*status = record.Status
	return nil
}

func (s *authorStateRowsStub) Err() error                                   { return s.err }
func (s *authorStateRowsStub) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (s *authorStateRowsStub) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (s *authorStateRowsStub) Values() ([]any, error)                       { return nil, errors.New("not implemented") }
func (s *authorStateRowsStub) RawValues() [][]byte                          { return nil }
func (s *authorStateRowsStub) Conn() *pgx.Conn                              { return nil }
func (s *authorStateRowsStub) TypeMap() *pgtype.Map                         { return pgtype.NewMap() }
