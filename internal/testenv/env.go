// Package testenv centralizes environment variables used by tagged tests.
package testenv

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

const (
	// EnvPostgresDSN contains the PostgreSQL connection string for integration tests.
	EnvPostgresDSN = "HAN_TEST_POSTGRES_DSN"
	// EnvBaseURL contains the deployed service URL for browser and end-to-end tests.
	EnvBaseURL = "HAN_TEST_BASE_URL"
	// EnvBrowserBin contains the browser executable path for browser tests.
	EnvBrowserBin = "HAN_TEST_BROWSER_BIN"
)

// LookupRequired returns a required test environment value.
func LookupRequired(key string) (string, error) {
	value, exists := os.LookupEnv(key)
	value = strings.TrimSpace(value)
	if !exists || value == "" {
		return "", fmt.Errorf("required test environment variable %s is not set", key)
	}
	return value, nil
}

// RequirePostgresDSN fails the test when the integration DSN is missing.
func RequirePostgresDSN(t testing.TB) string {
	t.Helper()
	return require(t, EnvPostgresDSN)
}

// RequireBaseURL fails the test when the deployed service URL is missing.
func RequireBaseURL(t testing.TB) string {
	t.Helper()
	return require(t, EnvBaseURL)
}

// RequireBrowserBin fails the test when the browser executable is missing.
func RequireBrowserBin(t testing.TB) string {
	t.Helper()
	return require(t, EnvBrowserBin)
}

func require(t testing.TB, key string) string {
	t.Helper()

	value, err := LookupRequired(key)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
