package testenv

import (
	"strings"
	"testing"
)

func TestLookupRequired_PresentValue_ReturnsValue(t *testing.T) {
	t.Setenv(EnvPostgresDSN, "postgresql://example.invalid/test")

	got, err := LookupRequired(EnvPostgresDSN)
	if err != nil {
		t.Fatalf("LookupRequired() error = %v", err)
	}
	if got != "postgresql://example.invalid/test" {
		t.Fatalf("LookupRequired() = %q, want configured value", got)
	}
}

func TestLookupRequired_MissingOrBlank_ReturnsError(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "missing"},
		{name: "blank", value: "   "},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(EnvBaseURL, tt.value)

			_, err := LookupRequired(EnvBaseURL)
			if err == nil {
				t.Fatal("LookupRequired() error = nil, want missing value error")
			}
			if !strings.Contains(err.Error(), EnvBaseURL) {
				t.Fatalf("LookupRequired() error = %q, want key %q", err, EnvBaseURL)
			}
		})
	}
}
