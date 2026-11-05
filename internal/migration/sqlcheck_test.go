package migration

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestValidateSQLTransactionControlConformance(t *testing.T) {
	t.Parallel()

	transactionCommands := []string{"COMMIT", "ROLLBACK"}
	transactionAfterBody := []struct {
		name string
		sql  func(string) string
	}{
		{
			name: "line_comment_lf",
			sql:  func(command string) string { return "SELECT 1; -- ordinary comment\n" + command + ";" },
		},
		{
			name: "line_comment_cr",
			sql:  func(command string) string { return "SELECT 1; -- ordinary comment\r" + command + ";" },
		},
		{
			name: "line_comment_crlf",
			sql:  func(command string) string { return "SELECT 1; -- ordinary comment\r\n" + command + ";" },
		},
		{
			name: "nested_block_comment",
			sql: func(command string) string {
				return "SELECT 1 /* outer /* " + command + " */ still outer */; " + command + ";"
			},
		},
		{
			name: "ordinary_string_doubled_quote",
			sql: func(command string) string {
				return "SELECT 'it''s " + command + "'; " + command + ";"
			},
		},
		{
			name: "ordinary_string_backslash_scs_on",
			sql:  func(command string) string { return "SELECT 'backslash\\'; " + command + ";" },
		},
		{
			name: "escape_string_escaped_quote",
			sql: func(command string) string {
				return "SELECT E'" + command + ` escaped \''; ` + command + ";"
			},
		},
		{
			name: "quoted_identifier",
			sql: func(command string) string {
				return `SELECT 1 AS "identifier "" ` + command + `"; ` + command + ";"
			},
		},
		{
			name: "dollar_quote_empty_tag",
			sql:  func(command string) string { return "SELECT $$ " + command + " $$; " + command + ";" },
		},
		{
			name: "dollar_quote_ascii_tag",
			sql: func(command string) string {
				return "SELECT $body$ " + command + " $body$; " + command + ";"
			},
		},
		{
			name: "dollar_quote_high_bit_tag",
			sql:  func(command string) string { return "SELECT $é$ " + command + " $é$; " + command + ";" },
		},
		{
			name: "dollar_quote_replacement_character_tag",
			sql:  func(command string) string { return "SELECT $�$ " + command + " $�$; " + command + ";" },
		},
		{
			name: "ascii_identifier_adjacent_dollar",
			sql:  func(command string) string { return "SELECT foo$guard$; " + command + ";" },
		},
		{
			name: "unicode_identifier_adjacent_dollar",
			sql:  func(command string) string { return "SELECT é$guard$; " + command + ";" },
		},
		{
			name: "separated_dollar_delimiter",
			sql: func(command string) string {
				return "SELECT 1, $guard$ " + command + " $guard$; " + command + ";"
			},
		},
		{
			name: "invalid_digit_start_dollar_tag",
			sql: func(command string) string {
				return "SELECT $1tag$ " + command + " $1tag$; " + command + ";"
			},
		},
		{
			name: "invalid_embedded_dollar_tag",
			sql: func(command string) string {
				return "SELECT $bad$tag$ " + command + " $bad$tag$; " + command + ";"
			},
		},
		{
			name: "unicode_nonkeyword_begin_end",
			sql: func(command string) string {
				return "BÉGIN ATOMIC SELECT 1; ENDé; " + command + ";"
			},
		},
		{
			name: "ascii_only_case_keyword",
			sql: func(command string) string {
				return `
					CREATE FUNCTION probe() RETURNS integer LANGUAGE SQL
					BEGIN ATOMIC
						SELECT 1 AS caſe;
					END;
				` + command + ";"
			},
		},
		{
			name: "function_begin_atomic",
			sql: func(command string) string {
				return `
					CREATE FUNCTION probe() RETURNS integer LANGUAGE SQL
					BEGIN ATOMIC SELECT 1; END;
				` + command + ";"
			},
		},
		{
			name: "procedure_begin_atomic",
			sql: func(command string) string {
				return `
					CREATE PROCEDURE probe() LANGUAGE SQL
					BEGIN ATOMIC SELECT 1; END;
				` + command + ";"
			},
		},
		{
			name: "begin_atomic_nested_case",
			sql: func(command string) string {
				return `
					CREATE FUNCTION probe(value integer) RETURNS integer LANGUAGE SQL
					BEGIN ATOMIC
						SELECT CASE
							WHEN value > 0 THEN CASE WHEN value > 1 THEN value ELSE 1 END
							ELSE 0
						END;
					END;
				` + command + ";"
			},
		},
		{
			name: "bare_nested_begin_atomic_scanner_robustness",
			sql: func(command string) string {
				return `
					CREATE FUNCTION probe() RETURNS integer LANGUAGE SQL
					BEGIN ATOMIC
						BEGIN ATOMIC SELECT 1; END;
					END;
				` + command + ";"
			},
		},
	}

	for _, tt := range transactionAfterBody {
		tt := tt
		for _, command := range transactionCommands {
			command := command
			t.Run(tt.name+"_then_"+strings.ToLower(command), func(t *testing.T) {
				t.Parallel()

				assertTransactionControlRejected(t, tt.sql(command))
			})
		}
	}

	directTransactionControl := []struct {
		name string
		sql  string
	}{
		{name: "ascii_mixed_case_commit", sql: "cOmMiT;"},
		{name: "ascii_mixed_case_rollback", sql: "rOlLbAcK;"},
		{name: "start_block_comment_transaction", sql: "START/*x*/TRANSACTION;"},
		{name: "set_line_comment_cr_transaction", sql: "SET--x\rTRANSACTION;"},
		{name: "set_transaction", sql: "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;"},
		{
			name: "set_session_characteristics_as_transaction",
			sql:  "SET SESSION CHARACTERISTICS AS TRANSACTION ISOLATION LEVEL SERIALIZABLE;",
		},
		{name: "end_transaction", sql: "END;"},
		{name: "abort_transaction", sql: "ABORT;"},
		{name: "savepoint", sql: "SAVEPOINT s;"},
		{name: "release_savepoint", sql: "RELEASE SAVEPOINT s;"},
		{name: "rollback_to_savepoint", sql: "ROLLBACK TO SAVEPOINT s;"},
		{name: "prepare_transaction", sql: "PREPARE TRANSACTION 'x';"},
		{name: "commit_prepared", sql: "COMMIT PREPARED 'x';"},
		{name: "rollback_prepared", sql: "ROLLBACK PREPARED 'x';"},
	}
	for _, tt := range directTransactionControl {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertTransactionControlRejected(t, tt.sql)
		})
	}

	accepted := []struct {
		name string
		sql  string
	}{
		{name: "line_comment_eof_contains_keyword", sql: "SELECT 1; -- COMMIT at eof"},
		{
			name: "nested_block_comment_contains_keywords",
			sql:  "SELECT 1 /* outer /* COMMIT */ ROLLBACK */;",
		},
		{name: "ordinary_string_contains_keywords", sql: "SELECT 'COMMIT '' ROLLBACK';"},
		{name: "ordinary_string_backslash_scs_on", sql: "SELECT 'backslash\\';"},
		{name: "escape_string_contains_keywords", sql: `SELECT E'COMMIT \' ROLLBACK';`},
		{name: "quoted_identifier_contains_keywords", sql: `SELECT 1 AS "COMMIT "" ROLLBACK";`},
		{name: "dollar_quote_empty_tag", sql: "SELECT $$ COMMIT ROLLBACK $$;"},
		{name: "dollar_quote_ascii_tag", sql: "SELECT $body$ COMMIT ROLLBACK $body$;"},
		{name: "dollar_quote_high_bit_tag", sql: "SELECT $é$ COMMIT ROLLBACK $é$;"},
		{name: "dollar_quote_replacement_character_tag", sql: "SELECT $�$ COMMIT ROLLBACK $�$;"},
		{name: "unicode_lookalike_commit", sql: "COMMIＴ;"},
		{name: "unicode_nonkeywords", sql: "BÉGIN ATOMIC SELECT 1; ENDé;"},
		{
			name: "long_s_identifier",
			sql: `
				CREATE FUNCTION probe() RETURNS integer LANGUAGE SQL
				BEGIN ATOMIC SELECT 1 AS caſe; END;
			`,
		},
		{
			name: "function_begin_atomic_body_only",
			sql: `
				CREATE FUNCTION probe() RETURNS integer LANGUAGE SQL
				BEGIN ATOMIC SELECT 1; END;
			`,
		},
		{
			name: "procedure_begin_atomic_body_only",
			sql: `
				CREATE PROCEDURE probe() LANGUAGE SQL
				BEGIN ATOMIC SELECT 1; END;
			`,
		},
		{
			name: "begin_atomic_nested_case_body_only",
			sql: `
				CREATE FUNCTION probe(value integer) RETURNS integer LANGUAGE SQL
				BEGIN ATOMIC
					SELECT CASE
						WHEN value > 0 THEN CASE WHEN value > 1 THEN value ELSE 1 END
						ELSE 0
					END;
				END;
			`,
		},
		{
			name: "bare_nested_begin_atomic_scanner_robustness_only",
			sql: `
				CREATE FUNCTION probe() RETURNS integer LANGUAGE SQL
				BEGIN ATOMIC
					BEGIN ATOMIC SELECT 1; END;
				END;
			`,
		},
	}
	for _, tt := range accepted {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := rejectTransactionControl(tt.sql); err != nil {
				t.Fatalf("rejectTransactionControl() error = %v, want nil", err)
			}
		})
	}
}

func TestLoad_TransactionControlConformance(t *testing.T) {
	t.Parallel()

	transactionCommands := []string{"COMMIT", "ROLLBACK"}
	malformed := []struct {
		name string
		sql  func(string) string
	}{
		{
			name: "unterminated_block_comment",
			sql:  func(command string) string { return "SELECT 1; /* unterminated " + command + ";" },
		},
		{
			name: "unterminated_ordinary_string",
			sql:  func(command string) string { return "SELECT 'unterminated " + command + ";" },
		},
		{
			name: "unterminated_escape_string",
			sql:  func(command string) string { return "SELECT E'unterminated " + command + ";" },
		},
		{
			name: "unterminated_quoted_identifier",
			sql:  func(command string) string { return `SELECT "unterminated ` + command + ";" },
		},
		{
			name: "unterminated_dollar_quote",
			sql:  func(command string) string { return "SELECT $body$ unterminated " + command + ";" },
		},
		{
			name: "mismatched_dollar_tag",
			sql: func(command string) string {
				return "SELECT $left$ unterminated " + command + "; $right$"
			},
		},
	}

	for _, tt := range malformed {
		tt := tt
		for _, command := range transactionCommands {
			command := command
			t.Run(tt.name+"_"+strings.ToLower(command), func(t *testing.T) {
				t.Parallel()

				const migrationPath = "migrations/0001_malformed.sql"
				fsys := fstest.MapFS{
					migrationPath: {Data: []byte(tt.sql(command))},
				}
				_, err := Load(fsys, "migrations")
				if err == nil {
					t.Fatal("Load() error = nil, want malformed migration rejection")
				}
				if !strings.Contains(err.Error(), migrationPath) {
					t.Fatalf("Load() error = %q, want migration path %q", err, migrationPath)
				}
			})
		}
	}
}

func TestSQLScannerLongInput_RejectsTransactionControlWithoutPanicking(t *testing.T) {
	t.Parallel()

	padding := strings.Repeat("x", 64*1024)
	tests := []struct {
		name string
		sql  string
	}{
		{name: "line comment", sql: "-- " + padding + "\rCOMMIT;"},
		{name: "ordinary string", sql: "SELECT '" + padding + "'; ROLLBACK;"},
		{name: "dollar quote", sql: "SELECT $body$" + padding + "$body$; COMMIT;"},
		{
			name: "nested block comments",
			sql:  "SELECT 1; /*" + strings.Repeat("/*", 512) + padding + strings.Repeat("*/", 512) + "*/ ROLLBACK;",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertTransactionControlRejected(t, tt.sql)
		})
	}
}

func FuzzSQLTransactionControlScanner(f *testing.F) {
	for _, seed := range []string{
		"-- comment\rCOMMIT;",
		"-- comment\r\nROLLBACK;",
		"SELECT 'COMMIT';",
		`SELECT E'ROLLBACK \'';`,
		`SELECT "COMMIT";`,
		"SELECT $é$ COMMIT $é$;",
		"SELECT $�$ ROLLBACK $�$;",
		"SELECT é$guard$; COMMIT;",
		"/* outer /* COMMIT */ outer */ ROLLBACK;",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, sql string) {
		_ = rejectTransactionControl(sql)
	})
}

func assertTransactionControlRejected(t *testing.T, sql string) {
	t.Helper()

	if err := rejectTransactionControl(sql); err == nil {
		t.Fatalf("rejectTransactionControl() error = nil for %d-byte SQL, want rejection", len(sql))
	}
}

func TestRejectTransactionControl_RejectsTopLevelStatements(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		sql  string
	}{
		{name: "begin", sql: "BEGIN; SELECT 1;"},
		{name: "begin transaction", sql: "BEGIN TRANSACTION;"},
		{name: "start transaction", sql: "START TRANSACTION;"},
		{name: "commit", sql: "COMMIT;"},
		{name: "end", sql: "END;"},
		{name: "rollback", sql: "ROLLBACK;"},
		{name: "abort", sql: "ABORT;"},
		{name: "savepoint", sql: "SAVEPOINT before_change;"},
		{name: "release savepoint", sql: "RELEASE SAVEPOINT before_change;"},
		{name: "prepare transaction", sql: "PREPARE TRANSACTION 'probe';"},
		{name: "commit prepared", sql: "COMMIT PREPARED 'probe';"},
		{name: "rollback prepared", sql: "ROLLBACK PREPARED 'probe';"},
		{name: "later statement", sql: "SELECT 1; /* boundary */ COMMIT;"},
		{name: "ordinary string backslash does not escape quote", sql: `SELECT 'text\'; COMMIT;`},
		{
			name: "commit after SQL standard body",
			sql: `
				CREATE FUNCTION probe() RETURNS integer LANGUAGE SQL
				BEGIN ATOMIC SELECT 1; END;
				COMMIT;
			`,
		},
		{
			name: "ordinary create statement does not open atomic body",
			sql: `
				CREATE TABLE function (
					value integer
				);
				BEGIN ATOMIC;
			`,
		},
		{
			name: "commit after high character dollar tag",
			sql:  "DO $é$ BEGIN; COMMIT; END; $é$; COMMIT;",
		},
		{
			name: "commit after replacement character dollar tag",
			sql:  "DO $�$ BEGIN; COMMIT; END; $�$; COMMIT;",
		},
		{
			name: "non-ASCII identifier cannot open dollar quote",
			sql:  "SELECT 1 AS é$guard$; COMMIT; SELECT 1 AS x$guard$;",
		},
		{
			name: "rollback after long s identifier in atomic body",
			sql: `
				CREATE FUNCTION probe() RETURNS integer LANGUAGE SQL
				BEGIN ATOMIC
					SELECT 1 AS caſe;
				END;
				ROLLBACK;
			`,
		},
		{
			name: "commit after long s identifier in atomic body",
			sql: `
				CREATE FUNCTION probe() RETURNS integer LANGUAGE SQL
				BEGIN ATOMIC
					SELECT 1 AS caſe;
				END;
				COMMIT;
			`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := rejectTransactionControl(tt.sql)
			if err == nil {
				t.Fatal("rejectTransactionControl() error = nil, want transaction control rejection")
			}
			if !strings.Contains(err.Error(), "transaction control") {
				t.Fatalf("rejectTransactionControl() error = %q, want transaction control context", err)
			}
		})
	}
}

func TestRejectTransactionControl_IgnoresNonExecutableText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		sql  string
	}{
		{
			name: "line comment",
			sql:  "-- COMMIT;\nSELECT 1;",
		},
		{
			name: "nested block comments",
			sql:  "/* BEGIN; /* ROLLBACK; */ COMMIT; */ SELECT 1;",
		},
		{
			name: "ordinary string",
			sql:  "SELECT 'BEGIN; COMMIT; ROLLBACK; ''END;''';",
		},
		{
			name: "escape string",
			sql:  `SELECT E'BEGIN; COMMIT; ROLLBACK; \'';`,
		},
		{
			name: "quoted identifier",
			sql:  `SELECT 1 AS "COMMIT; ROLLBACK";`,
		},
		{
			name: "untagged dollar quote",
			sql:  "DO $$ BEGIN RAISE NOTICE 'COMMIT;'; END $$;",
		},
		{
			name: "tagged dollar quote",
			sql:  "CREATE FUNCTION probe() RETURNS void LANGUAGE plpgsql AS $body$ BEGIN COMMIT; END $body$;",
		},
		{
			name: "high character dollar tag",
			sql:  "DO $é$ BEGIN; COMMIT; END; $é$;",
		},
		{
			name: "replacement character dollar tag",
			sql:  "DO $�$ BEGIN; COMMIT; END; $�$;",
		},
		{
			name: "long s identifier in SQL standard body",
			sql: `
				CREATE FUNCTION probe() RETURNS integer LANGUAGE SQL
				BEGIN ATOMIC
					SELECT 1 AS caſe;
				END;
			`,
		},
		{
			name: "transaction words after statement start",
			sql:  `INSERT INTO probe ("commit", value) VALUES ('ROLLBACK', 1);`,
		},
		{
			name: "SQL standard function body",
			sql: `
				CREATE FUNCTION normalize_probe(value integer)
				RETURNS integer
				LANGUAGE SQL
				BEGIN ATOMIC
					SELECT CASE
						WHEN value > 0 THEN value
						ELSE 0
					END;
				END;
			`,
		},
		{
			name: "SQL standard procedure body with nested atomic block",
			sql: `
				CREATE OR REPLACE PROCEDURE update_probe()
				LANGUAGE SQL
				BEGIN ATOMIC
					/* BEGIN; COMMIT; END; */
					INSERT INTO probe(value) VALUES ('END;');
					BEGIN ATOMIC
						UPDATE probe
						SET value = CASE WHEN value = 'x' THEN 'y' ELSE value END;
					END;
				END;
			`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := rejectTransactionControl(tt.sql); err != nil {
				t.Fatalf("rejectTransactionControl() error = %v, want nil", err)
			}
		})
	}
}
