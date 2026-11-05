package main

import (
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"han/internal/recovery"
)

type recoveryCommand struct {
	action, manifest string
	binding          recovery.RunBinding
}

func selectRecoveryCommand(args []string) (recoveryCommand, bool, error) {
	if len(args) == 0 || args[0] != "recovery" {
		return recoveryCommand{action: "normal"}, false, nil
	}
	command, err := parseRecoveryCommand(args[1:])
	return command, true, err
}

func parseRecoveryCommand(args []string) (recoveryCommand, error) {
	if len(args) == 1 && args[0] == "verify" {
		return recoveryCommand{action: "verify"}, nil
	}
	if len(args) < 1 || (args[0] != "enter" && args[0] != "resume") {
		return recoveryCommand{}, recovery.ErrAdmissionBinding
	}
	values, err := recoveryCommandFlags(args[1:])
	if err != nil {
		return recoveryCommand{}, err
	}
	if len(values) == 1 && args[0] == "enter" && filepath.IsAbs(values["--installation-manifest"]) {
		return recoveryCommand{action: "install", manifest: values["--installation-manifest"]}, nil
	}
	if len(values) != 2 || !recoveryExactText(values["--run-id"]) || !recoveryExactText(values["--backup-ref"]) {
		return recoveryCommand{}, recovery.ErrAdmissionBinding
	}
	return recoveryCommand{action: args[0], binding: recovery.RunBinding{RunID: values["--run-id"], BackupRef: values["--backup-ref"]}}, nil
}

func recoveryCommandFlags(args []string) (map[string]string, error) {
	if len(args)%2 != 0 {
		return nil, recovery.ErrAdmissionBinding
	}
	values := make(map[string]string)
	for i := 0; i < len(args); i += 2 {
		key, value := args[i], args[i+1]
		if key != "--run-id" && key != "--backup-ref" && key != "--installation-manifest" {
			return nil, recovery.ErrAdmissionBinding
		}
		if _, duplicate := values[key]; duplicate || !recoveryExactText(value) {
			return nil, recovery.ErrAdmissionBinding
		}
		values[key] = value
	}
	return values, nil
}

func recoveryExactText(value string) bool {
	return value != "" && len(value) <= 1024 && strings.TrimSpace(value) == value && utf8.ValidString(value) && !strings.ContainsFunc(value, unicode.IsControl)
}
