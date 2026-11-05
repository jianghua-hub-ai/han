package main

import (
	"fmt"
	"io/fs"
	"net"
	"net/netip"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"han/internal/recovery"
)

type recoveryRootConfig struct {
	command                                                          recoveryCommand
	deployment, installation, target, commitment, lockPath           string
	journalPath, journalID, keyringPath, anchorAdapter, anchorConfig string
	articleID                                                        string
	epoch, articleEpoch                                              uint64
	fileMode                                                         fs.FileMode
	proofBudget, ioBudget, drainBudget                               time.Duration
	maxBytes, maxRecords                                             int64
	restoreExecutable, restoreSHA, restoreManifest                   string
	controlAddress, controlOrigin                                    string
}

func loadRecoveryRootConfig(getenv func(string) string, command recoveryCommand) (recoveryRootConfig, error) {
	cfg := recoveryRootConfig{command: command}
	if getenv == nil {
		return cfg, recovery.ErrAdmissionBinding
	}
	mode := getenv(recoveryModeEnvironment)
	if mode != "normal" && mode != "maintenance" || command.action == "normal" && mode != "normal" {
		return cfg, recoveryConfigError(recoveryModeEnvironment)
	}
	if command.action != "normal" && command.action != "verify" && mode != "maintenance" {
		return cfg, recoveryConfigError(recoveryModeEnvironment)
	}
	if err := cfg.readCommon(getenv); err != nil {
		return cfg, err
	}
	if command.action == "enter" || command.action == "resume" {
		if err := cfg.readAction(getenv); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

// recoveryRootConfigurationKeys is the exact data set the root assembly needs.
// HAN_RECOVERY_MODE alone is not configuration: the legacy maintenance mode
// keeps its previous behaviour, and an all-empty data set builds no recovery
// runtime so normal startup is unchanged.
var recoveryRootConfigurationKeys = []string{
	"HAN_RECOVERY_DEPLOYMENT_ID",
	"HAN_RECOVERY_INSTALLATION_ID",
	"HAN_RECOVERY_TARGET_DIGEST",
	"HAN_RECOVERY_INSTALL_COMMITMENT_REF",
	"HAN_RECOVERY_RUNTIME_LOCK_PATH",
	"HAN_RECOVERY_JOURNAL_PATH",
	"HAN_RECOVERY_JOURNAL_ID",
	"HAN_RECOVERY_KEYRING_PATH",
	"HAN_RECOVERY_ANCHOR_ADAPTER",
	"HAN_RECOVERY_ANCHOR_CONFIG_PATH",
	"HAN_ARTICLE_DELETION_JOURNAL_ID",
	"HAN_RECOVERY_PROOF_BUDGET",
	"HAN_RECOVERY_IO_BUDGET",
	"HAN_RECOVERY_DRAIN_BUDGET",
	"HAN_RECOVERY_READ_MAX_BYTES",
	"HAN_RECOVERY_READ_MAX_RECORDS",
	"HAN_RECOVERY_JOURNAL_EPOCH",
	"HAN_ARTICLE_DELETION_JOURNAL_EPOCH",
	"HAN_RECOVERY_JOURNAL_FILE_MODE",
	"HAN_RECOVERY_RESTORE_EXECUTABLE",
	"HAN_RECOVERY_RESTORE_EXECUTABLE_SHA256",
	"HAN_RECOVERY_RESTORE_MANIFEST_PATH",
	"HAN_RECOVERY_CONTROL_HTTP_ADDR",
	"HAN_RECOVERY_CONTROL_ORIGIN",
}

// recoveryRootConfigurationPresent reports whether any recovery data key is
// set. A partially set key set is not silently ignored: it reaches
// loadRecoveryRootConfig and fails closed there.
func recoveryRootConfigurationPresent(getenv func(string) string) bool {
	if getenv == nil {
		return false
	}
	for _, key := range recoveryRootConfigurationKeys {
		if getenv(key) != "" {
			return true
		}
	}
	return false
}

// recoveryStartupPlan is the pure startup decision. It owns no side effect:
// verify keeps maintenance, listeners and workers false by construction.
type recoveryStartupPlan struct {
	present     bool
	verify      bool
	maintenance bool
	action      string
	config      recoveryRootConfig
}

func planRecoveryStartup(getenv func(string) string, command *recoveryCommand) (recoveryStartupPlan, error) {
	if getenv == nil {
		return recoveryStartupPlan{}, recovery.ErrAdmissionBinding
	}
	if command == nil {
		if !recoveryRootConfigurationPresent(getenv) {
			return recoveryStartupPlan{}, nil
		}
		cfg, err := loadRecoveryRootConfig(getenv, recoveryCommand{action: "normal"})
		if err != nil {
			return recoveryStartupPlan{}, err
		}
		return recoveryStartupPlan{present: true, action: "normal", config: cfg}, nil
	}
	cfg, err := loadRecoveryRootConfig(getenv, *command)
	if err != nil {
		return recoveryStartupPlan{}, err
	}
	plan := recoveryStartupPlan{present: true, action: command.action, config: cfg}
	switch command.action {
	case "verify":
		plan.verify = true
	case "normal":
	case "enter", "resume", "install":
		plan.maintenance = true
	default:
		return recoveryStartupPlan{}, recovery.ErrAdmissionBinding
	}
	return plan, nil
}

func recoveryConfigError(key string) error {
	return fmt.Errorf("%w: invalid %s", recovery.ErrAdmissionBinding, key)
}

func (c *recoveryRootConfig) readCommon(getenv func(string) string) error {
	fields := []struct {
		key    string
		target *string
		path   bool
	}{
		{"HAN_RECOVERY_DEPLOYMENT_ID", &c.deployment, false},
		{"HAN_RECOVERY_INSTALLATION_ID", &c.installation, false},
		{"HAN_RECOVERY_TARGET_DIGEST", &c.target, false},
		{"HAN_RECOVERY_INSTALL_COMMITMENT_REF", &c.commitment, false},
		{"HAN_RECOVERY_RUNTIME_LOCK_PATH", &c.lockPath, true},
		{"HAN_RECOVERY_JOURNAL_PATH", &c.journalPath, true},
		{"HAN_RECOVERY_JOURNAL_ID", &c.journalID, false},
		{"HAN_RECOVERY_KEYRING_PATH", &c.keyringPath, true},
		{"HAN_RECOVERY_ANCHOR_ADAPTER", &c.anchorAdapter, false},
		{"HAN_RECOVERY_ANCHOR_CONFIG_PATH", &c.anchorConfig, true},
		{"HAN_ARTICLE_DELETION_JOURNAL_ID", &c.articleID, false},
	}
	for _, field := range fields {
		value := getenv(field.key)
		if !recoveryExactText(value) || field.path && !recoveryAbsolutePath(value) {
			return recoveryConfigError(field.key)
		}
		*field.target = value
	}
	if !recoveryUUIDText(c.deployment) || !recoveryUUIDText(c.installation) || !recoverySHA(c.target) || c.journalPath == c.lockPath {
		return recoveryConfigError("HAN_RECOVERY_INSTALLATION_ID")
	}
	return c.readBudgets(getenv)
}

func (c *recoveryRootConfig) readBudgets(getenv func(string) string) error {
	for _, field := range []struct {
		key    string
		target *time.Duration
	}{
		{"HAN_RECOVERY_PROOF_BUDGET", &c.proofBudget}, {"HAN_RECOVERY_IO_BUDGET", &c.ioBudget}, {"HAN_RECOVERY_DRAIN_BUDGET", &c.drainBudget},
	} {
		value, err := parseRequiredDuration(getenv(field.key), field.key, time.Nanosecond, 24*time.Hour)
		if err != nil {
			return recoveryConfigError(field.key)
		}
		*field.target = value
	}
	for _, field := range []struct {
		key    string
		target *int64
	}{
		{"HAN_RECOVERY_READ_MAX_BYTES", &c.maxBytes}, {"HAN_RECOVERY_READ_MAX_RECORDS", &c.maxRecords},
	} {
		value, err := parsePositiveDecimal(getenv(field.key), field.key, 64)
		if err != nil || field.target == &c.maxBytes && uint64(value) > uint64(^uint(0)>>1) {
			return recoveryConfigError(field.key)
		}
		*field.target = value
	}
	for _, field := range []struct {
		key    string
		target *uint64
	}{
		{"HAN_RECOVERY_JOURNAL_EPOCH", &c.epoch}, {"HAN_ARTICLE_DELETION_JOURNAL_EPOCH", &c.articleEpoch},
	} {
		value, err := parsePositiveDecimal(getenv(field.key), field.key, 64)
		if err != nil {
			return recoveryConfigError(field.key)
		}
		*field.target = uint64(value)
	}
	raw := getenv("HAN_RECOVERY_JOURNAL_FILE_MODE")
	value, err := strconv.ParseUint(raw, 8, 12)
	if err != nil || value == 0 || value & ^uint64(0700) != 0 {
		return recoveryConfigError("HAN_RECOVERY_JOURNAL_FILE_MODE")
	}
	c.fileMode = fs.FileMode(value)
	return nil
}

func (c *recoveryRootConfig) readAction(getenv func(string) string) error {
	for _, field := range []struct {
		key    string
		target *string
	}{
		{"HAN_RECOVERY_RESTORE_EXECUTABLE", &c.restoreExecutable}, {"HAN_RECOVERY_RESTORE_MANIFEST_PATH", &c.restoreManifest},
	} {
		value := getenv(field.key)
		if !recoveryAbsolutePath(value) {
			return recoveryConfigError(field.key)
		}
		*field.target = value
	}
	c.restoreSHA = getenv("HAN_RECOVERY_RESTORE_EXECUTABLE_SHA256")
	if !regexp.MustCompile("^[0-9a-f]{64}$").MatchString(c.restoreSHA) {
		return recoveryConfigError("HAN_RECOVERY_RESTORE_EXECUTABLE_SHA256")
	}
	c.controlAddress, c.controlOrigin = getenv("HAN_RECOVERY_CONTROL_HTTP_ADDR"), getenv("HAN_RECOVERY_CONTROL_ORIGIN")
	return validateRecoveryControl(c.controlAddress, c.controlOrigin, getenv(httpAddressEnvironment), getenv(metricsAddressEnvironment))
}

func recoveryAbsolutePath(value string) bool {
	return recoveryExactText(value) && filepath.IsAbs(value) && filepath.Clean(value) == value
}

func recoveryUUIDText(value string) bool {
	return regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$").MatchString(value)
}

func recoverySHA(value string) bool {
	return regexp.MustCompile("^sha256:[0-9a-f]{64}$").MatchString(value)
}

func validateRecoveryControl(address, origin, business, metrics string) error {
	host, port, err := net.SplitHostPort(address)
	ip, ipErr := netip.ParseAddr(host)
	number, numberErr := strconv.ParseUint(port, 10, 16)
	if err != nil || ipErr != nil || ip.Zone() != "" || !ip.Unmap().IsLoopback() || numberErr != nil || number == 0 {
		return recoveryConfigError("HAN_RECOVERY_CONTROL_HTTP_ADDR")
	}
	for _, other := range []string{business, metrics} {
		_, otherPort, err := net.SplitHostPort(other)
		otherNumber, numberErr := strconv.ParseUint(otherPort, 10, 16)
		if err != nil || numberErr != nil || otherNumber == number {
			return recoveryConfigError("HAN_RECOVERY_CONTROL_HTTP_ADDR")
		}
	}
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || strings.TrimSpace(origin) != origin {
		return recoveryConfigError("HAN_RECOVERY_CONTROL_ORIGIN")
	}
	if p := u.Port(); p != "" {
		n, err := strconv.ParseUint(p, 10, 16)
		if err != nil || n == 0 {
			return recoveryConfigError("HAN_RECOVERY_CONTROL_ORIGIN")
		}
	}
	return nil
}
