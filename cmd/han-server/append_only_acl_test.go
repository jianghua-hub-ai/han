package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestVerifyAppendOnlyApplicationPrivileges(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mutate  func(*aclFakeDatabase)
		tables  []string
		wantErr error
		wantRaw string
	}{
		{name: "insert_only_passes", mutate: func(*aclFakeDatabase) {}},
		{
			name: "current_user_switched_from_session_user_is_rejected",
			mutate: func(database *aclFakeDatabase) {
				database.currentUser = "han_app"
				database.sessionUser = "postgres"
			},
			wantErr: errAppendOnlyPrivilege,
			wantRaw: "role switching",
		},
		{
			name: "current_superuser_is_rejected",
			mutate: func(database *aclFakeDatabase) {
				database.currentSuper = true
			},
			wantErr: errAppendOnlyPrivilege,
			wantRaw: "superuser",
		},
		{
			name: "session_superuser_is_rejected",
			mutate: func(database *aclFakeDatabase) {
				database.sessionSuper = true
			},
			wantErr: errAppendOnlyPrivilege,
			wantRaw: "superuser",
		},
		{
			name: "owner_member_is_rejected",
			mutate: func(database *aclFakeDatabase) {
				database.setPrivileges("audit_events", func(privileges *appendOnlyPrivileges) {
					privileges.ownerMember = true
				})
			},
			wantErr: errAppendOnlyPrivilege,
			wantRaw: "owner",
		},
		{
			name: "update_granted_is_rejected",
			mutate: func(database *aclFakeDatabase) {
				database.setPrivileges("audit_events", func(privileges *appendOnlyPrivileges) {
					privileges.updateTable = true
				})
			},
			wantErr: errAppendOnlyPrivilege,
			wantRaw: "update=true",
		},
		{
			name: "delete_granted_is_rejected",
			mutate: func(database *aclFakeDatabase) {
				database.setPrivileges("processing_gate_records", func(privileges *appendOnlyPrivileges) {
					privileges.deleteTable = true
				})
			},
			wantErr: errAppendOnlyPrivilege,
			wantRaw: "delete=true",
		},
		{
			name: "truncate_granted_is_rejected",
			mutate: func(database *aclFakeDatabase) {
				database.setPrivileges("audit_events", func(privileges *appendOnlyPrivileges) {
					privileges.truncateTable = true
				})
			},
			wantErr: errAppendOnlyPrivilege,
			wantRaw: "truncate=true",
		},
		{
			name: "insert_missing_is_rejected",
			mutate: func(database *aclFakeDatabase) {
				database.setPrivileges("audit_events", func(privileges *appendOnlyPrivileges) {
					privileges.insertTable = false
				})
			},
			wantErr: errAppendOnlyPrivilege,
			wantRaw: "insert=false",
		},
		{
			name: "column_insert_missing_is_rejected",
			mutate: func(database *aclFakeDatabase) {
				database.setPrivileges("audit_events", func(privileges *appendOnlyPrivileges) {
					privileges.insertColumn = false
				})
			},
			wantErr: errAppendOnlyPrivilege,
			wantRaw: "insert_column=false",
		},
		{
			name: "column_update_granted_is_rejected",
			mutate: func(database *aclFakeDatabase) {
				database.setPrivileges("audit_events", func(privileges *appendOnlyPrivileges) {
					privileges.updateColumn = true
				})
			},
			wantErr: errAppendOnlyPrivilege,
			wantRaw: "update_column=true",
		},
		{
			name: "column_delete_granted_is_rejected",
			mutate: func(database *aclFakeDatabase) {
				database.setPrivileges("audit_events", func(privileges *appendOnlyPrivileges) {
					privileges.deleteColumn = true
				})
			},
			wantErr: errAppendOnlyPrivilege,
			wantRaw: "delete_column=true",
		},
		{
			name: "missing_table_is_rejected",
			mutate: func(database *aclFakeDatabase) {
				database.setPrivileges("audit_events", func(privileges *appendOnlyPrivileges) {
					privileges.visible = 0
				})
			},
			wantErr: errAppendOnlyTableUnresolved,
			wantRaw: "visible relations=0",
		},
		{
			name: "ambiguous_table_is_rejected",
			mutate: func(database *aclFakeDatabase) {
				database.setPrivileges("audit_events", func(privileges *appendOnlyPrivileges) {
					privileges.visible = 2
				})
			},
			wantErr: errAppendOnlyTableUnresolved,
			wantRaw: "visible relations=2",
		},
		{
			name: "ambiguous_same_name_relations_on_search_path_are_rejected",
			mutate: func(database *aclFakeDatabase) {
				database.setPrivileges("audit_events", func(privileges *appendOnlyPrivileges) {
					privileges.onSearchPath = 2
				})
			},
			wantErr: errAppendOnlyTableAmbiguous,
			wantRaw: "same-name relations on effective search_path=2",
		},
		{
			name: "missing_same_name_relation_on_search_path_is_rejected",
			mutate: func(database *aclFakeDatabase) {
				database.setPrivileges("audit_events", func(privileges *appendOnlyPrivileges) {
					privileges.onSearchPath = 0
				})
			},
			wantErr: errAppendOnlyTableAmbiguous,
			wantRaw: "same-name relations on effective search_path=0",
		},
		{
			name:    "query_failure_is_rejected",
			mutate:  func(database *aclFakeDatabase) { database.failure = errors.New("connection lost") },
			wantRaw: "connection lost",
		},
		{
			name:    "invalid_table_name_is_rejected",
			tables:  []string{"audit_events; DROP TABLE users"},
			wantRaw: "invalid append-only table name",
		},
		{name: "empty_table_list_is_rejected", tables: []string{}, wantRaw: "append-only tables are required"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			database := newACLFakeDatabase()
			if test.mutate != nil {
				test.mutate(database)
			}
			tables := test.tables
			if tables == nil {
				tables = appendOnlyTables
			}
			err := verifyAppendOnlyApplicationPrivileges(context.Background(), database, tables)
			if test.wantRaw == "" && test.wantErr == nil {
				if err != nil {
					t.Fatalf("verify privileges: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("verify privileges succeeded; want failure")
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("error = %v; want %v", err, test.wantErr)
			}
			if test.wantRaw != "" && !strings.Contains(err.Error(), test.wantRaw) {
				t.Fatalf("error = %q; want substring %q", err.Error(), test.wantRaw)
			}
		})
	}
}

func TestVerifyAppendOnlyApplicationPrivilegesRequiresQueryer(t *testing.T) {
	t.Parallel()
	if err := verifyAppendOnlyApplicationPrivileges(context.Background(), nil, appendOnlyTables); err == nil {
		t.Fatal("nil queryer accepted")
	}
}

func TestMigrationDatabaseDSN(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "missing", value: "", wantErr: true},
		{name: "blank", value: "   ", wantErr: true},
		{name: "trimmed", value: "  postgres://migrator@127.0.0.1:5433/han  ", want: "postgres://migrator@127.0.0.1:5433/han"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dsn, err := migrationDatabaseDSN(func(string) string { return test.value })
			if test.wantErr {
				if !errors.Is(err, errMigrationDatabaseDSNRequired) {
					t.Fatalf("error = %v; want %v", err, errMigrationDatabaseDSNRequired)
				}
				return
			}
			if err != nil {
				t.Fatalf("migration DSN: %v", err)
			}
			if dsn != test.want {
				t.Fatalf("dsn = %q; want %q", dsn, test.want)
			}
		})
	}
}

func TestMigrationDatabaseDSNRequiresReader(t *testing.T) {
	t.Parallel()
	if _, err := migrationDatabaseDSN(nil); err == nil {
		t.Fatal("nil environment reader accepted")
	}
}

func TestParseDatabasePoolConfigRejectsRoleSwitchingOptions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		dsn     string
		wantErr bool
	}{
		{
			name:    "url_options_role",
			dsn:     "postgres://app@127.0.0.1:5433/han?options=-c%20role%3Dpostgres",
			wantErr: true,
		},
		{
			name:    "url_options_compact_c_role",
			dsn:     "postgres://app@127.0.0.1:5433/han?options=-crole%3Dpostgres",
			wantErr: true,
		},
		{
			name:    "url_options_long_role",
			dsn:     "postgres://app@127.0.0.1:5433/han?options=--role%3Dpostgres",
			wantErr: true,
		},
		{
			name:    "url_options_session_authorization",
			dsn:     "postgres://app@127.0.0.1:5433/han?options=-c%20session_authorization%3Dpostgres",
			wantErr: true,
		},
		{
			name:    "url_options_compact_c_session_authorization",
			dsn:     "postgres://app@127.0.0.1:5433/han?options=-csession_authorization%3Dpostgres",
			wantErr: true,
		},
		{
			name:    "url_role_parameter",
			dsn:     "postgres://app@127.0.0.1:5433/han?role=postgres",
			wantErr: true,
		},
		{
			name:    "url_role_parameter_uppercase",
			dsn:     "postgres://app@127.0.0.1:5433/han?ROLE=postgres",
			wantErr: true,
		},
		{
			name:    "url_role_parameter_encoded",
			dsn:     "postgres://app@127.0.0.1:5433/han?%72ole=postgres",
			wantErr: true,
		},
		{
			name:    "url_session_authorization_parameter",
			dsn:     "postgres://app@127.0.0.1:5433/han?session_authorization=postgres",
			wantErr: true,
		},
		{
			name:    "url_session_authorization_parameter_mixed_case",
			dsn:     "postgres://app@127.0.0.1:5433/han?Session_Authorization=postgres",
			wantErr: true,
		},
		{
			name: "url_application_name_is_allowed",
			dsn:  "postgres://app@127.0.0.1:5433/han?application_name=han",
		},
		{
			name: "url_application_name_next_to_allowed_options",
			dsn:  "postgres://app@127.0.0.1:5433/han?application_name=han&options=-c%20statement_timeout%3D5000",
		},
		{
			name: "url_options_statement_timeout_is_allowed",
			dsn:  "postgres://app@127.0.0.1:5433/han?options=-c%20statement_timeout%3D5000",
		},
		{
			name:    "keyword_options_role",
			dsn:     "host=127.0.0.1 port=5433 dbname=han user=app options='-c role=postgres'",
			wantErr: true,
		},
		{
			name:    "keyword_options_session_authorization",
			dsn:     "host=127.0.0.1 port=5433 dbname=han user=app options='-c session_authorization=postgres'",
			wantErr: true,
		},
		{
			name: "keyword_options_statement_timeout_is_allowed",
			dsn:  "host=127.0.0.1 port=5433 dbname=han user=app options='-c statement_timeout=5000'",
		},
		{
			name:    "keyword_role_parameter",
			dsn:     "host=127.0.0.1 port=5433 dbname=han user=app role=postgres",
			wantErr: true,
		},
		{
			name:    "keyword_session_authorization_parameter",
			dsn:     "host=127.0.0.1 port=5433 dbname=han user=app session_authorization=postgres",
			wantErr: true,
		},
		{
			name: "keyword_application_name_is_allowed",
			dsn:  "host=127.0.0.1 port=5433 dbname=han user=app application_name=han",
		},
		{
			name: "keyword_options_search_path_value_named_role_is_allowed",
			dsn:  "host=127.0.0.1 port=5433 dbname=han user=app options='-c search_path=role=foo'",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseDatabasePoolConfig(test.dsn, func(string) string { return "" })
			if test.wantErr {
				if !errors.Is(err, errRoleSwitchingDatabaseOption) {
					t.Fatalf("error = %v; want %v", err, errRoleSwitchingDatabaseOption)
				}
				if strings.Contains(err.Error(), "app@") {
					t.Fatalf("error exposed DSN: %q", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse database pool config: %v", err)
			}
		})
	}
}

func TestMigrationPoolConfigRejectsRoleSwitchingOptions(t *testing.T) {
	t.Parallel()
	application := &pgxpool.Config{ConnConfig: &pgx.ConnConfig{
		Host: "127.0.0.1", Port: 5433, Database: "han", User: "han_app",
	}}
	env := func(key string) string {
		switch key {
		case databaseDSNEnvironment:
			return "postgres://han_app@127.0.0.1:5433/han"
		case migrationDatabaseDSNEnvironment:
			return "postgres://han_migrator@127.0.0.1:5433/han?options=-c%20role%3Dpostgres"
		default:
			return ""
		}
	}
	if _, err := migrationPoolConfig(application, env); !errors.Is(err, errRoleSwitchingDatabaseOption) {
		t.Fatalf("migration options error = %v; want %v", err, errRoleSwitchingDatabaseOption)
	}
}

func TestRejectSharedDatabaseIdentity(t *testing.T) {
	t.Parallel()
	application := aclConnConfig("localhost", 5433, "han", "han_app",
		pgconn.FallbackConfig{Host: "127.0.0.2", Port: 5433})
	tests := []struct {
		name       string
		migration  *pgx.ConnConfig
		wantErr    bool
		wantAnyErr bool
	}{
		{
			name:      "different_roles_on_same_endpoint_is_allowed",
			migration: aclConnConfig("127.0.0.1", 5433, "han", "han_migrator"),
		},
		{
			name:      "same_role_name_is_rejected",
			migration: aclConnConfig("127.0.0.3", 5433, "han", "han_app"),
			wantErr:   true,
		},
		{
			name:      "same_role_across_database_is_rejected",
			migration: aclConnConfig("127.0.0.3", 5433, "han_restore", "han_app"),
			wantErr:   true,
		},
		{
			name:      "same_role_across_host_is_rejected",
			migration: aclConnConfig("127.0.0.3", 5433, "han", "han_app"),
			wantErr:   true,
		},
		{
			name: "same_role_on_fallback_is_rejected",
			migration: aclConnConfig("127.0.0.4", 5433, "han", "han_app",
				pgconn.FallbackConfig{Host: "127.0.0.2", Port: 5433}),
			wantErr: true,
		},
		{
			name: "same_host_different_port_is_allowed",
			migration: aclConnConfig("127.0.0.4", 5433, "han", "han_migrator",
				pgconn.FallbackConfig{Host: "127.0.0.2", Port: 5434}),
		},
		{
			name: "nil_fallback_is_rejected",
			migration: &pgx.ConnConfig{
				Host: "127.0.0.4", Port: 5433, Database: "han", User: "han_migrator",
				Fallbacks: []*pgconn.FallbackConfig{nil},
			},
			wantAnyErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := rejectSharedDatabaseIdentity(application, test.migration)
			if test.wantErr || test.wantAnyErr {
				if err == nil {
					t.Fatal("identity accepted; want failure")
				}
				if !test.wantAnyErr && !errors.Is(err, errSharedDatabaseIdentity) {
					t.Fatalf("error = %v; want %v", err, errSharedDatabaseIdentity)
				}
				return
			}
			if err != nil {
				t.Fatalf("identity rejected: %v", err)
			}
		})
	}
	if err := rejectSharedDatabaseIdentity(nil, application); err == nil {
		t.Fatal("missing application configuration accepted")
	}
	if err := rejectSharedDatabaseIdentity(application, nil); err == nil {
		t.Fatal("missing migration configuration accepted")
	}
}

func TestDatabaseEndpointsOfIncludesFallbacks(t *testing.T) {
	t.Parallel()
	config := aclConnConfig("LOCALHOST", 5433, "han", "han_app",
		pgconn.FallbackConfig{Host: "127.0.0.2", Port: 5434})
	endpoints, err := databaseEndpointsOf(config)
	if err != nil {
		t.Fatalf("database endpoints: %v", err)
	}
	want := []databaseIdentity{
		{host: "localhost", port: 5433, database: "han", user: "han_app"},
		{host: "127.0.0.2", port: 5434, database: "han", user: "han_app"},
	}
	if len(endpoints) != len(want) {
		t.Fatalf("endpoint count = %d; want %d", len(endpoints), len(want))
	}
	for index := range want {
		if endpoints[index] != want[index] {
			t.Fatalf("endpoint[%d] = %+v; want %+v", index, endpoints[index], want[index])
		}
	}

	config.Fallbacks = append(config.Fallbacks, nil)
	if _, err := databaseEndpointsOf(config); err == nil {
		t.Fatal("nil fallback accepted")
	}
	if _, err := databaseEndpointsOf(nil); err == nil {
		t.Fatal("nil configuration accepted")
	}
}

func aclConnConfig(
	host string,
	port uint16,
	database string,
	user string,
	fallbacks ...pgconn.FallbackConfig,
) *pgx.ConnConfig {
	config := &pgx.ConnConfig{Host: host, Port: port, Database: database, User: user}
	for index := range fallbacks {
		fallback := fallbacks[index]
		config.Fallbacks = append(config.Fallbacks, &fallback)
	}
	return config
}

type aclFakeDatabase struct {
	currentUser  string
	sessionUser  string
	currentSuper bool
	sessionSuper bool
	privileges   map[string]appendOnlyPrivileges
	failure      error
}

func newACLFakeDatabase() *aclFakeDatabase {
	passing := appendOnlyPrivileges{
		visible: 1, onSearchPath: 1, insertTable: true, insertColumn: true,
	}
	return &aclFakeDatabase{
		currentUser: "han_app",
		sessionUser: "han_app",
		privileges: map[string]appendOnlyPrivileges{
			"audit_events":            passing,
			"processing_gate_records": passing,
		},
	}
}

func (database *aclFakeDatabase) setPrivileges(
	table string,
	mutate func(*appendOnlyPrivileges),
) {
	privileges := database.privileges[table]
	mutate(&privileges)
	database.privileges[table] = privileges
}

func (database *aclFakeDatabase) QueryRow(
	_ context.Context,
	sql string,
	args ...any,
) pgx.Row {
	if database.failure != nil {
		return aclFakeRow{failure: database.failure}
	}
	table := ""
	if len(args) > 0 {
		table, _ = args[0].(string)
	}
	switch {
	case strings.Contains(sql, "pg_roles"):
		return aclFakeRow{values: []any{
			database.currentUser, database.sessionUser,
			database.currentSuper, database.sessionSuper,
		}}
	case strings.Contains(sql, "WITH visible AS"):
		privileges := database.privileges[table]
		return aclFakeRow{values: []any{
			privileges.visible,
			privileges.onSearchPath,
			privileges.ownerMember,
			privileges.insertTable,
			privileges.updateTable,
			privileges.deleteTable,
			privileges.truncateTable,
			privileges.insertColumn,
			privileges.updateColumn,
			privileges.deleteColumn,
		}}
	case strings.Contains(sql, "coalesce(min(namespace.nspname)"):
		return aclFakeRow{values: []any{int64(1), int64(1), "han_test"}}
	default:
		return aclFakeRow{failure: errors.New("unexpected query")}
	}
}

type aclFakeRow struct {
	values  []any
	failure error
}

func (row aclFakeRow) Scan(dest ...any) error {
	if row.failure != nil {
		return row.failure
	}
	if len(dest) != len(row.values) {
		return errors.New("fake row destination mismatch")
	}
	for index, target := range dest {
		switch typed := target.(type) {
		case *string:
			value, ok := row.values[index].(string)
			if !ok {
				return errors.New("fake row type mismatch")
			}
			*typed = value
		case *bool:
			value, ok := row.values[index].(bool)
			if !ok {
				return errors.New("fake row type mismatch")
			}
			*typed = value
		case *int64:
			value, ok := row.values[index].(int64)
			if !ok {
				return errors.New("fake row type mismatch")
			}
			*typed = value
		case *int:
			value, ok := row.values[index].(int)
			if !ok {
				return errors.New("fake row type mismatch")
			}
			*typed = value
		default:
			return errors.New("fake row destination mismatch")
		}
	}
	return nil
}
