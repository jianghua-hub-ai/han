package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func recoveryConfigFixture(t *testing.T) map[string]string {
	t.Helper()
	root := t.TempDir()
	values := map[string]string{
		"HAN_RECOVERY_MODE":                   "normal",
		"HAN_RECOVERY_DEPLOYMENT_ID":          "11111111-1111-4111-8111-111111111111",
		"HAN_RECOVERY_INSTALLATION_ID":        "22222222-2222-4222-8222-222222222222",
		"HAN_RECOVERY_TARGET_DIGEST":          "sha256:" + strings.Repeat("a", 64),
		"HAN_RECOVERY_INSTALL_COMMITMENT_REF": "approved-installation",
		"HAN_RECOVERY_JOURNAL_ID":             "recovery-journal",
		"HAN_RECOVERY_JOURNAL_EPOCH":          "1", "HAN_RECOVERY_JOURNAL_FILE_MODE": "0600",
		"HAN_RECOVERY_ANCHOR_ADAPTER":     "explicit-test-source",
		"HAN_ARTICLE_DELETION_JOURNAL_ID": "article-journal", "HAN_ARTICLE_DELETION_JOURNAL_EPOCH": "1",
		"HAN_RECOVERY_PROOF_BUDGET": "30s", "HAN_RECOVERY_IO_BUDGET": "30s", "HAN_RECOVERY_DRAIN_BUDGET": "30s",
		"HAN_RECOVERY_READ_MAX_BYTES": "8388608", "HAN_RECOVERY_READ_MAX_RECORDS": "100000",
	}
	for key, name := range map[string]string{
		"HAN_RECOVERY_RUNTIME_LOCK_PATH": "deployment.lock", "HAN_RECOVERY_JOURNAL_PATH": "journal",
		"HAN_RECOVERY_KEYRING_PATH": "keyring", "HAN_RECOVERY_ANCHOR_CONFIG_PATH": "anchor",
	} {
		values[key] = filepath.Join(root, name)
	}
	return values
}

func TestRecoveryConfig(t *testing.T) {
	testRecoveryConfig(t)
}

func testRecoveryConfig(t *testing.T) {
	values := recoveryConfigFixture(t)
	for _, action := range []string{"normal", "verify"} {
		read := make(map[string]int)
		cfg, err := loadRecoveryRootConfig(func(key string) string { read[key]++; return values[key] }, recoveryCommand{action: action})
		if err != nil || cfg.maxRecords != 100000 || cfg.epoch != 1 {
			t.Errorf("valid %s configuration rejected: %v", action, err)
		}
		for _, key := range []string{"HAN_RECOVERY_INITIAL_ADMIN_SECRET", "HAN_RECOVERY_RESTORE_EXECUTABLE", "HAN_RECOVERY_CONTROL_HTTP_ADDR"} {
			if read[key] != 0 {
				t.Errorf("%s read an inapplicable dependency", action)
			}
		}
	}
	testRecoveryConfigInvalid(t, values)
}

func testRecoveryConfigInvalid(t *testing.T, values map[string]string) {
	t.Helper()
	cases := map[string][]string{
		"HAN_RECOVERY_MODE":              {"", "maintenance", "Normal"},
		"HAN_RECOVERY_PROOF_BUDGET":      {"", "0s", "25h", " 1s"},
		"HAN_RECOVERY_READ_MAX_RECORDS":  {"", "01", "0", "9223372036854775808"},
		"HAN_RECOVERY_JOURNAL_FILE_MODE": {"", "0644", "0000"},
		"HAN_RECOVERY_TARGET_DIGEST":     {"", "sha256:" + strings.Repeat("A", 64)},
		"HAN_RECOVERY_JOURNAL_PATH":      {"", "relative"},
	}
	for key, invalid := range cases {
		original := values[key]
		for _, value := range invalid {
			values[key] = value
			if _, err := loadRecoveryRootConfig(func(key string) string { return values[key] }, recoveryCommand{action: "normal"}); err == nil {
				t.Errorf("invalid %s accepted", key)
			}
		}
		values[key] = original
	}
}
