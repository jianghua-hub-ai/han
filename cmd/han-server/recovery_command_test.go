package main

import (
	"path/filepath"
	"testing"
)

func TestRecoveryCommand(t *testing.T) {
	testRecoveryCommand(t)
}

func testRecoveryCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "installation.json")
	for _, action := range []string{"enter", "resume"} {
		args := []string{"recovery", action, "--run-id", "run-1", "--backup-ref", "backup-1"}
		got, selected, err := selectRecoveryCommand(args)
		if err != nil || !selected || got.action != action || got.binding.RunID != "run-1" || got.binding.BackupRef != "backup-1" {
			t.Errorf("valid %s rejected: %v", action, err)
		}
	}
	got, selected, err := selectRecoveryCommand([]string{"recovery", "enter", "--installation-manifest", path})
	if err != nil || !selected || got.action != "install" || got.manifest != path {
		t.Error("explicit installation command was not selected")
	}
	got, selected, err = selectRecoveryCommand([]string{"recovery", "verify"})
	if err != nil || !selected || got.action != "verify" {
		t.Error("verify was not selected")
	}
	for _, args := range [][]string{nil, {"maintenance", "password-reset", "retry"}} {
		if _, selected, err := selectRecoveryCommand(args); selected || err != nil {
			t.Error("recovery parser consumed the ordinary or old maintenance entry")
		}
	}
	testRecoveryCommandInvalid(t, path)
}

func testRecoveryCommandInvalid(t *testing.T, manifest string) {
	t.Helper()
	cases := [][]string{
		{"recovery"}, {"recovery", "unknown"}, {"recovery", "verify", "--run-id", "run-1"},
		{"recovery", "enter"}, {"recovery", "resume", "--installation-manifest", manifest},
		{"recovery", "enter", "--installation-manifest", "relative"},
		{"recovery", "enter", "--run-id", "run-1"},
		{"recovery", "enter", "--run-id", "run-1", "--backup-ref", "backup-1", "--run-id", "run-2"},
		{"recovery", "enter", "--run-id", "run-1", "--backup-ref", "backup-1", "--installation-manifest", manifest},
		{"recovery", "enter", "--run-id", " run-1", "--backup-ref", "backup-1"},
		{"recovery", "enter", "--run-id", "run-1", "--backup-ref", "backup-1", "extra"},
		{"recovery", "enter", "--run-id", "run-1", "--backup-ref", "backup-1", "--secret", "forbidden"},
	}
	for i, args := range cases {
		if _, selected, err := selectRecoveryCommand(args); !selected || err == nil {
			t.Errorf("invalid command case %d accepted", i)
		}
	}
}
