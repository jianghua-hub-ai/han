package main

import (
	"strings"
	"testing"
)

func TestParseDatabasePoolConfig_AmbientPostgresSettingsFailClosed(t *testing.T) {
	t.Parallel()

	ambientKeys := []string{
		"PGHOST",
		"PGPORT",
		"PGDATABASE",
		"PGUSER",
		"PGPASSWORD",
		"PGPASSFILE",
		"PGSERVICE",
		"PGSERVICEFILE",
		"PGSSLMODE",
		"PGSSLCERT",
		"PGSSLKEY",
		"PGSSLROOTCERT",
	}
	const (
		dsn          = "host=dsn.example port=5433 user=han password=dsn-secret dbname=han sslmode=disable"
		ambientValue = "ambient-secret"
	)

	for _, key := range ambientKeys {
		key := key
		t.Run(key, func(t *testing.T) {
			t.Parallel()

			_, err := parseDatabasePoolConfig(dsn, func(name string) string {
				if name == key {
					return ambientValue
				}
				return ""
			})
			if err == nil {
				t.Fatal("parseDatabasePoolConfig() error = nil, want ambient configuration rejection")
			}
			if strings.Contains(err.Error(), dsn) ||
				strings.Contains(err.Error(), "dsn-secret") ||
				strings.Contains(err.Error(), ambientValue) {
				t.Fatalf("parseDatabasePoolConfig() error exposed configuration: %q", err)
			}
		})
	}
}

func TestParseDatabasePoolConfig_UsesOnlyDSNForBothSupportedForms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dsn  string
	}{
		{
			name: "keyword value",
			dsn:  "host=dsn.example port=5433 user=han password=dsn-secret dbname=han sslmode=disable pool_max_conns=7",
		},
		{
			name: "URI",
			dsn:  "postgres://han:dsn-secret@dsn.example:5433/han?sslmode=disable&pool_max_conns=7",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config, err := parseDatabasePoolConfig(tt.dsn, func(string) string { return "" })
			if err != nil {
				t.Fatalf("parseDatabasePoolConfig() error = %v", err)
			}
			if config.ConnConfig.Host != "dsn.example" ||
				config.ConnConfig.Port != 5433 ||
				config.ConnConfig.User != "han" ||
				config.ConnConfig.Database != "han" {
				t.Fatal("parsed connection endpoint or identity did not match DSN values")
			}
			if config.ConnConfig.Password != "dsn-secret" {
				t.Fatal("parsed connection password did not match the DSN value")
			}
			if config.MaxConns != 7 {
				t.Fatalf("MaxConns = %d, want 7", config.MaxConns)
			}
		})
	}
}

func TestParseDatabasePoolConfig_DoesNotUseHomePassfileFallback(t *testing.T) {
	t.Parallel()

	config, err := parseDatabasePoolConfig(
		"host=passfile.invalid user=han dbname=han sslmode=disable",
		func(string) string { return "" },
	)
	if err != nil {
		t.Fatalf("parseDatabasePoolConfig() error = %v", err)
	}
	if config.ConnConfig.Password != "" {
		t.Fatalf("Password = %q, want no ambient passfile password", config.ConnConfig.Password)
	}
}

func TestParseDatabasePoolConfig_RejectsExternalFileAndServiceSources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dsn  string
	}{
		{name: "keyword service", dsn: "host=dsn.example service=external"},
		{name: "keyword service file", dsn: "host=dsn.example servicefile='secret-service-file'"},
		{name: "keyword pass file", dsn: "host=dsn.example passfile='secret-pass-file'"},
		{name: "keyword TLS certificate", dsn: "host=dsn.example sslcert='secret-certificate-file'"},
		{name: "URI service", dsn: "postgres://dsn.example/han?service=external"},
		{name: "URI encoded service file", dsn: "postgres://dsn.example/han?service%66ile=secret-service-file"},
		{name: "URI pass file", dsn: "postgres://dsn.example/han?passfile=secret-pass-file"},
		{name: "URI TLS root certificate", dsn: "postgres://dsn.example/han?sslrootcert=secret-root-file"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseDatabasePoolConfig(tt.dsn, func(string) string { return "" })
			if err == nil {
				t.Fatal("parseDatabasePoolConfig() error = nil, want external source rejection")
			}
			if strings.Contains(err.Error(), tt.dsn) || strings.Contains(err.Error(), "secret-") {
				t.Fatalf("parseDatabasePoolConfig() error exposed configuration: %q", err)
			}
		})
	}
}

func TestParseDatabasePoolConfig_InvalidDSNDoesNotExposeSecret(t *testing.T) {
	t.Parallel()

	const secret = "invalid-dsn-secret"
	dsn := "postgres://han:" + secret + "@dsn.example:not-a-port/han"

	_, err := parseDatabasePoolConfig(dsn, func(string) string { return "" })
	if err == nil {
		t.Fatal("parseDatabasePoolConfig() error = nil, want invalid DSN rejection")
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), dsn) {
		t.Fatalf("parseDatabasePoolConfig() error exposed DSN: %q", err)
	}
}

func TestParseDatabasePoolConfig_RejectsRawQuestionMarkInURLUserinfo(t *testing.T) {
	t.Parallel()

	const secret = "userinfo-secret"
	dsn := "postgres://han?" + secret + "@db.example/han"

	_, err := parseDatabasePoolConfig(dsn, func(string) string { return "" })
	if err == nil {
		t.Fatal("parseDatabasePoolConfig() error = nil, want ambiguous userinfo rejection before pgx parsing")
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), dsn) {
		t.Fatalf("parseDatabasePoolConfig() error exposed DSN: %q", err)
	}
}
